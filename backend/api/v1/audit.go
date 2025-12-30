package v1

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/pkg/errors"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// used for replacing sensitive fields.
var (
	maskedString string
)

// AuditInterceptor is the v1 audit interceptor for gRPC server.
type AuditInterceptor struct {
	store   *store.Store
	secret  string
	profile *config.Profile
}

// NewAuditInterceptor returns a new v1 API audit interceptor.
func NewAuditInterceptor(store *store.Store, secret string, profile *config.Profile) *AuditInterceptor {
	return &AuditInterceptor{
		store:   store,
		secret:  secret,
		profile: profile,
	}
}

// WrapUnary implements the ConnectRPC interceptor interface for unary RPCs.
func (in *AuditInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		var serviceData *anypb.Any
		ctx = common.WithSetServiceData(ctx, func(a *anypb.Any) {
			serviceData = a
		})

		startTime := time.Now()
		response, rerr := next(ctx, req)
		latency := time.Since(startTime)

		if needAudit(ctx) {
			var respMsg any
			if !common.IsNil(response) {
				respMsg = response.Any()
			}
			if err := createAuditLogConnect(ctx, req.Any(), respMsg, req.Spec().Procedure, in.store, in.secret, in.profile, serviceData, rerr, req.Header(), req.Peer().Addr, latency); err != nil {
				slog.Warn("audit interceptor: failed to create audit log", log.BBError(err), slog.String("method", req.Spec().Procedure))
			}
		}

		return response, rerr
	}
}

// WrapStreamingClient implements the ConnectRPC interceptor interface for streaming clients.
func (*AuditInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

// WrapStreamingHandler implements the ConnectRPC interceptor interface for streaming handlers.
func (in *AuditInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if !needAudit(ctx) {
			return next(ctx, conn)
		}

		wrappedConn := &auditConnectStreamingConn{
			StreamingHandlerConn: conn,
			interceptor:          in,
			ctx:                  ctx,
			method:               conn.Spec().Procedure,
		}
		return next(ctx, wrappedConn)
	}
}

type auditConnectStreamingConn struct {
	connect.StreamingHandlerConn
	interceptor *AuditInterceptor
	ctx         context.Context
	method      string
	curRequest  any
	startTime   time.Time
}

func (c *auditConnectStreamingConn) Receive(msg any) error {
	err := c.StreamingHandlerConn.Receive(msg)
	if err != nil {
		return err
	}
	// Store current request for audit log and start time
	c.curRequest = msg
	c.startTime = time.Now()
	return nil
}

func (c *auditConnectStreamingConn) Send(resp any) error {
	err := c.StreamingHandlerConn.Send(resp)
	if err != nil {
		return err
	}
	// Create audit log for each message pair
	if c.curRequest != nil {
		latency := time.Since(c.startTime)
		if auditErr := createAuditLogConnect(c.ctx, c.curRequest, resp, c.method, c.interceptor.store, c.interceptor.secret, c.interceptor.profile, nil, nil, c.RequestHeader(), c.Peer().Addr, latency); auditErr != nil {
			return auditErr
		}
	}
	return nil
}

func createAuditLogConnect(ctx context.Context, request, response any, method string, storage *store.Store, secret string, profile *config.Profile, serviceData *anypb.Any, rerr error, headers http.Header, peerAddr string, latency time.Duration) error {
	// Skip audit logging for validate-only requests.
	if isValidateOnlyRequest(request) {
		return nil
	}

	requestString, err := getRequestString(request)
	if err != nil {
		return errors.Wrapf(err, "failed to get request string")
	}
	responseString, err := getResponseString(response)
	if err != nil {
		return errors.Wrapf(err, "failed to get response string")
	}

	var user string
	if u, ok := GetUserFromContext(ctx); ok {
		user = common.FormatUserEmail(u.Email)
	} else {
		// Try to get user from successful login response.
		if loginResponse, ok := response.(*v1pb.LoginResponse); ok {
			user = loginResponse.GetUser().GetName()
		}
	}

	authContextAny := ctx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return connect.NewError(connect.CodeInternal, errors.New("auth context not found"))
	}

	requestMetadata := getRequestMetadataFromHeaders(headers, peerAddr)

	var parents []string
	if authContext.HasWorkspaceResource() {
		systemSetting, err := storage.GetSystemSetting(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to get system setting")
		}
		workspaceID := systemSetting.WorkspaceId
		parents = append(parents, common.FormatWorkspace(workspaceID))
	} else {
		for _, projectID := range authContext.GetProjectResources() {
			parents = append(parents, common.FormatProject(projectID))
		}
	}

	createAuditLogCtx := context.WithoutCancel(ctx)
	for _, parent := range parents {
		resource := getRequestResource(request)
		// For login requests, if resource is empty, try to get email from user context or MFA temp token.
		// This handles MFA phase where request doesn't have email field.
		if resource == "" && method == v1connect.AuthServiceLoginProcedure {
			if u, ok := GetUserFromContext(ctx); ok {
				resource = u.Email
			} else if loginRequest, ok := request.(*v1pb.LoginRequest); ok && loginRequest.MfaTempToken != nil {
				// Extract user email from MFA temp token.
				if userEmail, err := auth.GetUserEmailFromMFATempToken(*loginRequest.MfaTempToken, secret); err == nil {
					resource = userEmail
				}
			}
		}

		p := &storepb.AuditLog{
			Parent:          parent,
			Method:          method,
			Resource:        resource,
			Severity:        storepb.AuditLog_INFO,
			User:            user,
			Request:         requestString,
			Response:        responseString,
			Status:          convertErrToStatus(rerr),
			Latency:         durationpb.New(latency),
			ServiceData:     serviceData,
			RequestMetadata: requestMetadata,
		}
		if err := storage.CreateAuditLog(createAuditLogCtx, p); err != nil {
			return err
		}

		// Log audit event to stdout using slog (if enabled)
		if profile.RuntimeEnableAuditLogStdout.Load() {
			logAuditToStdout(ctx, p)
		}
	}

	return nil
}

// logAuditToStdout writes audit log events to stdout using Go's standard slog library.
// Output format is controlled by the global slog handler (JSON in production, text in dev).
// Logs include a "log_type": "audit" field to distinguish from application logs.
// This is a best-effort operation - errors are not returned to avoid failing the audit flow.
func logAuditToStdout(ctx context.Context, p *storepb.AuditLog) {
	attrs := []slog.Attr{
		slog.String("log_type", "audit"),
		slog.String("parent", p.Parent),
		slog.String("method", p.Method),
	}

	if p.Resource != "" {
		attrs = append(attrs, slog.String("resource", p.Resource))
	}
	if p.User != "" {
		attrs = append(attrs, slog.String("user", p.User))
	}

	if p.Status != nil {
		attrs = append(attrs, slog.Int("status_code", int(p.Status.Code)))
		if p.Status.Message != "" {
			attrs = append(attrs, slog.String("status_message", p.Status.Message))
		}
	}

	if p.Latency != nil {
		attrs = append(attrs,
			slog.Int64("latency_ms", p.Latency.AsDuration().Milliseconds()),
		)
	}

	if p.RequestMetadata != nil {
		if p.RequestMetadata.CallerIp != "" {
			attrs = append(attrs, slog.String("client_ip", p.RequestMetadata.CallerIp))
		}
		if p.RequestMetadata.CallerSuppliedUserAgent != "" {
			attrs = append(attrs, slog.String("user_agent", p.RequestMetadata.CallerSuppliedUserAgent))
		}
	}

	// Include audit severity as an attribute (not as slog level)
	// Audit logs are always logged at INFO level - they represent business events, not system health
	// The severity field helps categorize the audit event itself
	if p.Severity != storepb.AuditLog_SEVERITY_UNSPECIFIED {
		attrs = append(attrs, slog.String("severity", p.Severity.String()))
	}

	slog.LogAttrs(ctx, slog.LevelInfo, p.Method, attrs...)
}

func getRequestResource(request any) string {
	if request == nil || reflect.ValueOf(request).IsNil() {
		return ""
	}
	switch r := request.(type) {
	case *v1pb.QueryRequest:
		return r.Name
	case *v1pb.AdminExecuteRequest:
		return r.Name
	case *v1pb.ExportRequest:
		return r.Name
	case *v1pb.UpdateDatabaseRequest:
		return r.Database.Name
	case *v1pb.BatchUpdateDatabasesRequest:
		return r.Parent
	case *v1pb.UpdateDatabaseCatalogRequest:
		return r.GetCatalog().GetName()
	case *v1pb.SetIamPolicyRequest:
		return r.Resource
	case *v1pb.CreateUserRequest:
		return r.GetUser().GetName()
	case *v1pb.UpdateUserRequest:
		return r.GetUser().GetName()
	case *v1pb.LoginRequest:
		return r.GetEmail()
	case *v1pb.CreateInstanceRequest:
		return r.GetInstance().GetName()
	case *v1pb.UpdateInstanceRequest:
		return r.GetInstance().GetName()
	case *v1pb.DeleteInstanceRequest:
		return r.GetName()
	case *v1pb.UndeleteInstanceRequest:
		return r.GetName()
	case *v1pb.AddDataSourceRequest:
		return r.GetName()
	case *v1pb.RemoveDataSourceRequest:
		return r.GetName()
	case *v1pb.UpdateDataSourceRequest:
		return r.GetName()
	case *v1pb.UpdateSettingRequest:
		return r.GetSetting().GetName()
	default:
	}
	return ""
}

func getRequestString(request any) (string, error) {
	m := func() protoreflect.ProtoMessage {
		if request == nil || reflect.ValueOf(request).IsNil() {
			return nil
		}
		switch r := request.(type) {
		case *v1pb.ExportRequest:
			return redactExportRequest(r)
		case *v1pb.CreateUserRequest:
			return redactCreateUserRequest(r)
		case *v1pb.UpdateUserRequest:
			return redactUpdateUserRequest(r)
		case *v1pb.LoginRequest:
			return redactLoginRequest(r)
		case *v1pb.CreateInstanceRequest:
			r.Instance = redactInstance(r.Instance)
			return r
		case *v1pb.UpdateInstanceRequest:
			r.Instance = redactInstance(r.Instance)
			return r
		case *v1pb.AddDataSourceRequest:
			r.DataSource = redactDataSource(r.DataSource)
			return r
		case *v1pb.UpdateDataSourceRequest:
			r.DataSource = redactDataSource(r.DataSource)
			return r
		case *v1pb.RemoveDataSourceRequest:
			r.DataSource = redactDataSource(r.DataSource)
			return r
		default:
			if p, ok := r.(protoreflect.ProtoMessage); ok {
				return p
			}
			return nil
		}
	}()
	if m == nil {
		return "", nil
	}

	b, err := protojson.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getResponseString(response any) (string, error) {
	m := func() protoreflect.ProtoMessage {
		if response == nil || reflect.ValueOf(response).IsNil() {
			return nil
		}
		switch r := response.(type) {
		case *v1pb.QueryResponse:
			return redactQueryResponse(r)
		case *v1pb.AdminExecuteResponse:
			return redactAdminExecuteResponse(r)
		case *v1pb.ExportResponse:
			return nil
		case *v1pb.LoginResponse:
			return redactLoginResponse(r)
		case *v1pb.User:
			return redactUser(r)
		case *v1pb.Instance:
			return redactInstance(r)
		default:
			if p, ok := r.(protoreflect.ProtoMessage); ok {
				return p
			}
			return nil
		}
	}()
	if m == nil {
		return "", nil
	}
	b, err := protojson.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func redactExportRequest(r *v1pb.ExportRequest) *v1pb.ExportRequest {
	if r == nil {
		return nil
	}
	r = proto.CloneOf(r)
	if r.Password != "" {
		r.Password = maskedString
	}
	return r
}

func redactLoginRequest(r *v1pb.LoginRequest) *v1pb.LoginRequest {
	if r == nil {
		return nil
	}

	// Clone to avoid mutating original
	r = proto.CloneOf(r)

	// Mask sensitive fields.
	if r.Password != "" {
		r.Password = maskedString
	}
	if r.OtpCode != nil {
		r.OtpCode = &maskedString
	}
	if r.RecoveryCode != nil {
		r.RecoveryCode = &maskedString
	}
	if r.MfaTempToken != nil {
		r.MfaTempToken = &maskedString
	}
	if r.IdpContext != nil {
		r.IdpContext = nil
	}
	return r
}

func redactCreateUserRequest(r *v1pb.CreateUserRequest) *v1pb.CreateUserRequest {
	if r == nil {
		return nil
	}
	return &v1pb.CreateUserRequest{
		User: redactUser(r.User),
	}
}

func redactUpdateUserRequest(r *v1pb.UpdateUserRequest) *v1pb.UpdateUserRequest {
	if r == nil {
		return nil
	}
	return &v1pb.UpdateUserRequest{
		User:                    redactUser(r.User),
		UpdateMask:              r.UpdateMask,
		OtpCode:                 r.OtpCode,
		RegenerateTempMfaSecret: r.RegenerateTempMfaSecret,
		RegenerateRecoveryCodes: r.RegenerateRecoveryCodes,
	}
}

func redactUser(r *v1pb.User) *v1pb.User {
	if r == nil {
		return nil
	}
	return &v1pb.User{
		Name:     r.Name,
		Email:    r.Email,
		Title:    r.Title,
		UserType: r.UserType,
	}
}

func redactInstance(i *v1pb.Instance) *v1pb.Instance {
	if i == nil {
		return nil
	}
	// Clone the instance to avoid modifying the original response
	cloned := proto.CloneOf(i)
	var dataSources []*v1pb.DataSource
	for _, d := range cloned.DataSources {
		dataSources = append(dataSources, redactDataSource(d))
	}
	cloned.DataSources = dataSources
	return cloned
}

func redactDataSource(d *v1pb.DataSource) *v1pb.DataSource {
	// Clone the datasource to avoid modifying the original
	cloned, ok := proto.Clone(d).(*v1pb.DataSource)
	if !ok {
		return d
	}
	if cloned.Password != "" {
		cloned.Password = maskedString
	}
	if cloned.SslCa != "" {
		cloned.SslCa = maskedString
	}
	if cloned.SslCert != "" {
		cloned.SslCert = maskedString
	}
	if cloned.SslKey != "" {
		cloned.SslKey = maskedString
	}
	if cloned.SshPassword != "" {
		cloned.SshPassword = maskedString
	}
	if cloned.SshPrivateKey != "" {
		cloned.SshPrivateKey = maskedString
	}
	if cloned.AuthenticationPrivateKey != "" {
		cloned.AuthenticationPrivateKey = maskedString
	}
	if cloned.ExternalSecret != nil {
		cloned.ExternalSecret = new(v1pb.DataSourceExternalSecret)
	}
	if cloned.SaslConfig != nil {
		if krbConf := cloned.SaslConfig.GetKrbConfig(); krbConf != nil {
			krbConf.Keytab = []byte(maskedString)
			cloned.SaslConfig.Mechanism = &v1pb.SASLConfig_KrbConfig{KrbConfig: krbConf}
		}
	}
	if cloned.MasterPassword != "" {
		cloned.MasterPassword = maskedString
	}
	return cloned
}

func redactAdminExecuteResponse(r *v1pb.AdminExecuteResponse) *v1pb.AdminExecuteResponse {
	if r == nil {
		return nil
	}
	n := &v1pb.AdminExecuteResponse{
		Results: nil,
	}
	for _, result := range r.Results {
		if result == nil {
			n.Results = append(n.Results, &v1pb.QueryResult{})
			continue
		}
		n.Results = append(n.Results, &v1pb.QueryResult{
			ColumnNames:     result.ColumnNames,
			ColumnTypeNames: result.ColumnTypeNames,
			Rows:            nil, // Redacted
			Error:           result.Error,
			Latency:         result.Latency,
			Statement:       result.Statement,
			DetailedError:   result.DetailedError,
			Masked:          redactMaskingReasons(result.Masked), // Redact icon data
		})
	}

	return n
}

func redactQueryResponse(r *v1pb.QueryResponse) *v1pb.QueryResponse {
	if r == nil {
		return nil
	}
	n := &v1pb.QueryResponse{
		Results: nil,
	}
	for _, result := range r.Results {
		n.Results = append(n.Results, &v1pb.QueryResult{
			ColumnNames:     result.ColumnNames,
			ColumnTypeNames: result.ColumnTypeNames,
			Rows:            nil, // Redacted
			RowsCount:       result.RowsCount,
			Error:           result.Error,
			Latency:         result.Latency,
			Statement:       result.Statement,
			DetailedError:   result.DetailedError,
			Masked:          redactMaskingReasons(result.Masked), // Redact icon data
		})
	}
	return n
}

func redactMaskingReasons(reasons []*v1pb.MaskingReason) []*v1pb.MaskingReason {
	if reasons == nil {
		return nil
	}
	var redacted []*v1pb.MaskingReason
	for _, reason := range reasons {
		if reason == nil {
			redacted = append(redacted, nil)
			continue
		}
		redacted = append(redacted, &v1pb.MaskingReason{
			SemanticTypeId:      reason.SemanticTypeId,
			SemanticTypeTitle:   reason.SemanticTypeTitle,
			MaskingRuleId:       reason.MaskingRuleId,
			Algorithm:           reason.Algorithm,
			Context:             reason.Context,
			ClassificationLevel: reason.ClassificationLevel,
			// Omit SemanticTypeIcon to avoid polluting audit logs with base64 data
		})
	}
	return redacted
}

func redactLoginResponse(r *v1pb.LoginResponse) *v1pb.LoginResponse {
	if r == nil {
		return nil
	}

	n := &v1pb.LoginResponse{
		RequireResetPassword: r.RequireResetPassword,
	}
	if r.User != nil {
		n.User = redactUser(r.User)
	}
	return n
}

func needAudit(ctx context.Context) bool {
	authCtx, ok := common.GetAuthContextFromContext(ctx)
	if !ok {
		slog.Warn("audit interceptor: failed to get auth context")
		return false
	}
	return authCtx.Audit
}

// getRequestMetadataFromHeaders extracts request metadata from HTTP headers for ConnectRPC.
func getRequestMetadataFromHeaders(headers http.Header, peerAddr string) *storepb.RequestMetadata {
	userAgent := headers.Get("User-Agent")
	// Extract caller IP with fallback chain:
	// 1. X-Real-IP (set by reverse proxy, most trustworthy single IP)
	// 2. X-Forwarded-For (standard but can contain client-spoofed data)
	// 3. Peer address from ConnectRPC (direct connection fallback)
	callerIP := headers.Get("X-Real-IP")
	if callerIP == "" {
		callerIP = headers.Get("X-Forwarded-For")
	}
	if callerIP == "" {
		callerIP = peerAddr
	}

	return &storepb.RequestMetadata{
		CallerIp:                callerIP,
		CallerSuppliedUserAgent: userAgent,
	}
}

// isValidateOnlyRequest checks if a request has validate_only field set to true
// using protoreflect to generically detect the field.
func isValidateOnlyRequest(request any) bool {
	if request == nil {
		return false
	}

	// Check if the value is nil (for pointer types).
	val := reflect.ValueOf(request)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return false
	}

	protoMsg, ok := request.(proto.Message)
	if !ok {
		return false
	}

	// Use protoreflect to check for validate_only field.
	msg := protoMsg.ProtoReflect()
	fields := msg.Descriptor().Fields()
	validateOnlyField := fields.ByName("validate_only")
	if validateOnlyField == nil {
		return false
	}

	// Check if the field is set and is true.
	return msg.Get(validateOnlyField).Bool()
}

// expect
// 1. connect.Error
// 2. other unknown errors
func convertErrToStatus(err error) *spb.Status {
	if err == nil {
		return nil
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return &spb.Status{
			Code:    int32(codes.Unknown),
			Message: err.Error(),
		}
	}

	st := &spb.Status{
		Code:    int32(connectErr.Code()),
		Message: connectErr.Message(),
	}
	for _, detail := range connectErr.Details() {
		st.Details = append(st.Details, &anypb.Any{
			TypeUrl: detail.Type(),
			Value:   detail.Bytes(),
		})
	}
	return st
}
