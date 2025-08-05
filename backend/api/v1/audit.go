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

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// used for replacing sensitive fields.
var (
	maskedString string
)

// ACLInterceptor is the v1 ACL interceptor for gRPC server.
type AuditInterceptor struct {
	store *store.Store
}

// NewAuditInterceptor returns a new v1 API ACL interceptor.
func NewAuditInterceptor(store *store.Store) *AuditInterceptor {
	return &AuditInterceptor{
		store: store,
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
			if err := createAuditLogConnect(ctx, req.Any(), respMsg, req.Spec().Procedure, in.store, serviceData, rerr, req.Header(), latency); err != nil {
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
		if auditErr := createAuditLogConnect(c.ctx, c.curRequest, resp, c.method, c.interceptor.store, nil, nil, c.RequestHeader(), latency); auditErr != nil {
			return auditErr
		}
	}
	return nil
}

func createAuditLogConnect(ctx context.Context, request, response any, method string, storage *store.Store, serviceData *anypb.Any, rerr error, headers http.Header, latency time.Duration) error {
	requestString, err := getRequestString(request)
	if err != nil {
		return errors.Wrapf(err, "failed to get request string")
	}
	responseString, err := getResponseString(response)
	if err != nil {
		return errors.Wrapf(err, "failed to get response string")
	}

	var user string
	if u, ok := ctx.Value(common.UserContextKey).(*store.UserMessage); ok {
		user = common.FormatUserUID(u.ID)
	} else {
		if loginResponse, ok := response.(*v1pb.LoginResponse); ok {
			user = loginResponse.GetUser().GetName()
		}
	}

	authContextAny := ctx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return connect.NewError(connect.CodeInternal, errors.New("auth context not found"))
	}

	requestMetadata := getRequestMetadataFromHeaders(headers)

	var parents []string
	if authContext.HasWorkspaceResource() {
		workspaceID, err := storage.GetWorkspaceID(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to get workspace id")
		}
		parents = append(parents, common.FormatWorkspace(workspaceID))
	} else {
		for _, projectID := range authContext.GetProjectResources() {
			parents = append(parents, common.FormatProject(projectID))
		}
	}

	createAuditLogCtx := context.WithoutCancel(ctx)
	for _, parent := range parents {
		p := &storepb.AuditLog{
			Parent:          parent,
			Method:          method,
			Resource:        getRequestResource(request),
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
	}

	return nil
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
	case *v1pb.CreateRiskRequest:
		return r.GetRisk().GetName()
	case *v1pb.DeleteRiskRequest:
		return r.Name
	case *v1pb.UpdateRiskRequest:
		return r.GetRisk().GetName()
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
			r = proto.CloneOf(r)
			r.Password = maskedString
			return r
		case *v1pb.CreateUserRequest:
			return redactCreateUserRequest(r)
		case *v1pb.UpdateUserRequest:
			return redactUpdateUserRequest(r)
		case *v1pb.LoginRequest:
			return redactLoginRequest(proto.CloneOf(r))
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

func redactLoginRequest(r *v1pb.LoginRequest) *v1pb.LoginRequest {
	if r == nil {
		return nil
	}

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
	var dataSources []*v1pb.DataSource
	for _, d := range i.DataSources {
		dataSources = append(dataSources, redactDataSource(d))
	}
	i.DataSources = dataSources
	return i
}

func redactDataSource(d *v1pb.DataSource) *v1pb.DataSource {
	if d.Password != "" {
		d.Password = maskedString
	}
	if d.SslCa != "" {
		d.SslCa = maskedString
	}
	if d.SslCert != "" {
		d.SslCert = maskedString
	}
	if d.SslKey != "" {
		d.SslKey = maskedString
	}
	if d.SshPassword != "" {
		d.SshPassword = maskedString
	}
	if d.SshPrivateKey != "" {
		d.SshPrivateKey = maskedString
	}
	if d.AuthenticationPrivateKey != "" {
		d.AuthenticationPrivateKey = maskedString
	}
	if d.ExternalSecret != nil {
		d.ExternalSecret = new(v1pb.DataSourceExternalSecret)
	}
	if d.SaslConfig != nil {
		if krbConf := d.SaslConfig.GetKrbConfig(); krbConf != nil {
			krbConf.Keytab = []byte(maskedString)
			d.SaslConfig.Mechanism = &v1pb.SASLConfig_KrbConfig{KrbConfig: krbConf}
		}
	}
	if d.MasterPassword != "" {
		d.MasterPassword = maskedString
	}
	return d
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
			AllowExport:     result.AllowExport,
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
func getRequestMetadataFromHeaders(headers http.Header) *storepb.RequestMetadata {
	userAgent := headers.Get("User-Agent")
	// For ConnectRPC, we don't have direct access to peer info like gRPC
	// The caller IP will need to be extracted from X-Forwarded-For or similar headers
	callerIP := headers.Get("X-Forwarded-For")
	if callerIP == "" {
		callerIP = headers.Get("X-Real-IP")
	}

	return &storepb.RequestMetadata{
		CallerIp:                callerIP,
		CallerSuppliedUserAgent: userAgent,
	}
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
