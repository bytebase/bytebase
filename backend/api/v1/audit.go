package v1

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
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

const (
	// maxAuditPayloadChars is the maximum characters for request/response payloads in stdout logs.
	// Set to 100KB (102400 chars) to match AWS CloudTrail industry standard for audit logs.
	maxAuditPayloadChars = 102400
)

// AuditInterceptor is the v1 audit interceptor for gRPC server.
type AuditInterceptor struct {
	store   *store.Store
	secret  string
	profile *config.Profile

	// createAuditLogFunc, when set, replaces createAuditLog for streaming sends.
	// Test-only seam that lets unit tests observe audit persistence ordering
	// without a database.
	createAuditLogFunc func(context.Context, *auditEntry) error
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

		// Allow handlers to announce the workspace a request should be audited
		// against. Needed for allow_without_credential methods (Login/Signup/
		// ExchangeToken) where the workspace is resolved inside the handler.
		var handlerAuditWorkspaceID string
		ctx = common.WithSetAuditWorkspaceID(ctx, func(workspaceID string) {
			handlerAuditWorkspaceID = workspaceID
		})

		startTime := time.Now()
		response, rerr := next(ctx, req)
		latency := time.Since(startTime)

		if needAudit(ctx) {
			var respMsg any
			if !common.IsNil(response) {
				respMsg = response.Any()
			}
			entry := &auditEntry{
				request:                 req.Any(),
				response:                respMsg,
				method:                  req.Spec().Procedure,
				serviceData:             serviceData,
				handlerAuditWorkspaceID: handlerAuditWorkspaceID,
				rerr:                    rerr,
				headers:                 req.Header(),
				peerAddr:                req.Peer().Addr,
				latency:                 latency,
			}
			if err := in.createAuditLog(ctx, entry); err != nil {
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
	// Create the audit log for each message pair before delivering the
	// response, so a client that observes a successful response can rely on
	// the audit entry already being durably persisted.
	if c.curRequest != nil {
		entry := &auditEntry{
			request:  c.curRequest,
			response: resp,
			method:   c.method,
			headers:  c.RequestHeader(),
			peerAddr: c.Peer().Addr,
			latency:  time.Since(c.startTime),
		}
		writeAuditLog := c.interceptor.createAuditLog
		if c.interceptor.createAuditLogFunc != nil {
			writeAuditLog = c.interceptor.createAuditLogFunc
		}
		if auditErr := writeAuditLog(c.ctx, entry); auditErr != nil {
			return auditErr
		}
	}
	return c.StreamingHandlerConn.Send(resp)
}

// auditEntry bundles the per-request data needed to write an audit log.
// Bundling here keeps AuditInterceptor.createAuditLog to a short signature
// (previously 13 positional args — easy to misorder).
type auditEntry struct {
	request  any
	response any
	method   string
	// serviceData is populated by handlers via common.WithSetServiceData.
	serviceData *anypb.Any
	// handlerAuditWorkspaceID is populated by handlers via
	// common.SetAuditWorkspaceID. Used as the audit-parent fallback for
	// allow_without_credential methods (Login/Signup/ExchangeToken) where
	// authContext.Resources is empty because no workspace is in the context.
	handlerAuditWorkspaceID string
	rerr                    error
	headers                 http.Header
	peerAddr                string
	latency                 time.Duration
}

func (in *AuditInterceptor) createAuditLog(ctx context.Context, e *auditEntry) error {
	// Skip audit logging for validate-only requests.
	if isValidateOnlyRequest(e.request) {
		return nil
	}

	requestString, err := getRequestString(e.request)
	if err != nil {
		return errors.Wrapf(err, "failed to get request string")
	}
	responseString, err := getResponseString(e.response)
	if err != nil {
		return errors.Wrapf(err, "failed to get response string")
	}

	var user string
	if u, ok := GetUserFromContext(ctx); ok {
		user = common.FormatUserEmail(u.Email)
	} else {
		// Try to get user from successful login response.
		if loginResponse, ok := e.response.(*v1pb.LoginResponse); ok {
			user = loginResponse.GetUser().GetName()
		}
	}

	authContextAny := ctx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return connect.NewError(connect.CodeInternal, errors.New("auth context not found"))
	}

	requestMetadata := getRequestMetadataFromHeaders(e.headers, e.peerAddr)

	// Build the list of parents to audit under. Normally these come from the
	// ACL interceptor via authContext.Resources. For audited methods that run
	// without a workspace-bound caller (Login/Signup/ExchangeToken), Resources
	// is empty; fall back to the workspace the handler announced, or any
	// workspace embedded in the response, so the audit entry is still written.
	type auditParent struct {
		parent           string
		auditWorkspaceID string
	}
	// One row per DISTINCT parent: batch requests repeat the same resource
	// once per item, and since ACL-denied internal-chain calls are audited
	// too, an unprivileged caller reaches this fan-out — duplicates would let
	// one denied batch call naming N items write N identical rows.
	var parents []auditParent
	seenParent := make(map[string]bool)
	appendParent := func(ap auditParent) {
		if seenParent[ap.parent] {
			return
		}
		seenParent[ap.parent] = true
		parents = append(parents, ap)
	}
	for _, authResource := range authContext.Resources {
		switch authResource.Type {
		case common.ResourceTypeProject:
			appendParent(auditParent{
				parent: common.FormatProject(authResource.ID),
			})
		case common.ResourceTypeWorkspace:
			appendParent(auditParent{
				parent:           common.FormatWorkspace(authResource.ID),
				auditWorkspaceID: authResource.ID,
			})
		default:
		}
	}
	if len(parents) == 0 {
		fallbackWS := e.handlerAuditWorkspaceID
		if fallbackWS == "" {
			if loginResp, ok := e.response.(*v1pb.LoginResponse); ok {
				if wsID, err := common.GetWorkspaceID(loginResp.GetUser().GetWorkspace()); err == nil && wsID != "" {
					fallbackWS = wsID
				}
			}
		}
		// Defensive: covers edge cases where populateRawResources produced no
		// resource despite the caller being authenticated (e.g. malformed
		// inputs like "instances/" that match the prefix but fail the regex).
		if fallbackWS == "" {
			fallbackWS = common.GetWorkspaceIDFromContext(ctx)
		}
		if fallbackWS != "" {
			parents = append(parents, auditParent{
				parent:           common.FormatWorkspace(fallbackWS),
				auditWorkspaceID: fallbackWS,
			})
		}
	}

	createAuditLogCtx := context.WithoutCancel(ctx)
	for _, ap := range parents {
		resource := getRequestResource(e.request, e.method)
		// For login requests, if resource is empty, try to get email from user context or MFA temp token.
		// This handles MFA phase where request doesn't have email field.
		if resource == "" && e.method == v1connect.AuthServiceLoginProcedure {
			if u, ok := GetUserFromContext(ctx); ok {
				resource = u.Email
			} else if loginRequest, ok := e.request.(*v1pb.LoginRequest); ok && loginRequest.MfaTempToken != nil {
				// Extract user email from MFA temp token.
				if userEmail, err := auth.GetUserEmailFromMFATempToken(*loginRequest.MfaTempToken, in.secret); err == nil {
					resource = userEmail
				}
			}
		}

		p := &storepb.AuditLog{
			Parent:          ap.parent,
			Method:          e.method,
			Resource:        resource,
			Severity:        storepb.AuditLog_INFO,
			User:            user,
			Request:         requestString,
			Response:        responseString,
			Status:          convertErrToStatus(e.rerr),
			Latency:         durationpb.New(e.latency),
			ServiceData:     e.serviceData,
			RequestMetadata: requestMetadata,
			McpDelegation:   mcpDelegationFromAuthContext(authContext),
		}
		// Resolve workspace for audit log.
		workspaceIDForAudit := ap.auditWorkspaceID
		if workspaceIDForAudit == "" {
			workspaceIDForAudit = common.GetWorkspaceIDFromContext(createAuditLogCtx)
		}
		if workspaceIDForAudit == "" {
			// Skip audit log if no workspace can be determined (e.g., unauthenticated request).
			continue
		}
		if err := in.store.CreateAuditLog(createAuditLogCtx, workspaceIDForAudit, p); err != nil {
			return err
		}

		// Log audit event to stdout using slog (if enabled)
		if in.profile.RuntimeEnableAuditLogStdout.Load() {
			logAuditToStdout(ctx, p)
		}
	}

	return nil
}

// mcpDelegationFromAuthContext copies the delegated MCP grant state onto the
// audit row, verbatim: empty grant values (legacy sessions) are recorded
// empty, never resolved to a synthetic label. A nil return — every
// public-chain request — leaves the row without MCP fields; presence of the
// message is the MCP-origin marker.
func mcpDelegationFromAuthContext(authContext *common.AuthContext) *storepb.MCPDelegation {
	g := authContext.DelegatedGrant
	if g == nil {
		return nil
	}
	return &storepb.MCPDelegation{
		Scope:         g.Scope,
		Resource:      g.Resource,
		ClientId:      g.ClientID,
		CorrelationId: g.CorrelationID,
	}
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

	attrs = append(attrs, mcpDelegationAttrs(p.McpDelegation)...)

	// Include request payload (truncated to 100KB for log manageability)
	// Request is already redacted for sensitive data by getRequestString()
	if p.Request != "" {
		request := p.Request
		if truncated, wasTruncated := common.TruncateString(p.Request, maxAuditPayloadChars); wasTruncated {
			request = truncated + "...[truncated]"
		}
		attrs = append(attrs, slog.String("request", request))
	}

	// Include response payload (truncated to 100KB for log manageability)
	// Response is already redacted for sensitive data by getResponseString()
	if p.Response != "" {
		response := p.Response
		if truncated, wasTruncated := common.TruncateString(p.Response, maxAuditPayloadChars); wasTruncated {
			response = truncated + "...[truncated]"
		}
		attrs = append(attrs, slog.String("response", response))
	}

	slog.LogAttrs(ctx, slog.LevelInfo, p.Method, attrs...)
}

// mcpDelegationAttrs renders the MCP provenance for stdout audit lines:
// "mcp": true marks the row as MCP-originated even when the grant fields are
// empty (legacy sessions); the correlation ID is what operators pivot on to
// reassemble an agent session.
func mcpDelegationAttrs(d *storepb.MCPDelegation) []slog.Attr {
	if d == nil {
		return nil
	}
	attrs := []slog.Attr{
		slog.Bool("mcp", true),
		// Minted for every session, legacy included — never empty.
		slog.String("mcp_correlation_id", d.CorrelationId),
	}
	if d.Scope != "" {
		attrs = append(attrs, slog.String("mcp_scope", d.Scope))
	}
	if d.Resource != "" {
		attrs = append(attrs, slog.String("mcp_resource", d.Resource))
	}
	if d.ClientId != "" {
		attrs = append(attrs, slog.String("mcp_client_id", d.ClientId))
	}
	return attrs
}

func getRequestResource(request any, method string) string {
	if request == nil || reflect.ValueOf(request).IsNil() {
		return ""
	}
	switch r := request.(type) {
	case *v1pb.BatchUpdateDatabasesRequest:
		return r.GetParent()
	case *v1pb.UpdateDatabaseCatalogRequest:
		return r.GetCatalog().GetName()
	case *v1pb.CreateUserRequest:
		return r.GetUser().GetName()
	case *v1pb.LoginRequest:
		return r.GetEmail()
	case *v1pb.SignupRequest:
		return r.GetEmail()
	case *v1pb.ExchangeTokenRequest:
		return r.GetEmail()
	case *v1pb.CreateInstanceRequest:
		if r.GetParent() == "" {
			return common.FormatInstance(r.GetInstanceId())
		}
		if projectID, err := common.GetProjectID(r.GetParent()); err == nil {
			return common.FormatProjectInstance(projectID, r.GetInstanceId())
		}
		return ""
	case *v1pb.CreateProjectRequest:
		return common.FormatProject(r.GetProjectId())
	case *v1pb.CreateReviewConfigRequest:
		return r.GetReviewConfig().GetName()
	}

	message, ok := request.(proto.Message)
	if !ok {
		return ""
	}
	shortMethod := method[strings.LastIndex(method, "/")+1:]
	return getResourceFromSingleRequest(message.ProtoReflect(), shortMethod)
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
		case *v1pb.SignupRequest:
			return redactSignupRequest(r)
		case *v1pb.ExchangeTokenRequest:
			return redactExchangeTokenRequest(r)
		case *v1pb.CreateProjectRequest:
			r = proto.CloneOf(r)
			r.Project = redactProject(r.Project)
			return r
		case *v1pb.UpdateProjectRequest:
			r = proto.CloneOf(r)
			r.Project = redactProject(r.Project)
			return r
		case *v1pb.CreateInstanceRequest:
			r = proto.CloneOf(r)
			r.Instance = redactInstance(r.Instance)
			return r
		case *v1pb.UpdateInstanceRequest:
			r = proto.CloneOf(r)
			r.Instance = redactInstance(r.Instance)
			return r
		case *v1pb.AddDataSourceRequest:
			r = proto.CloneOf(r)
			r.DataSource = redactDataSource(r.DataSource)
			return r
		case *v1pb.UpdateDataSourceRequest:
			r = proto.CloneOf(r)
			r.DataSource = redactDataSource(r.DataSource)
			return r
		case *v1pb.RemoveDataSourceRequest:
			r = proto.CloneOf(r)
			r.DataSource = redactDataSource(r.DataSource)
			return r
		case *v1pb.UpdateSettingRequest:
			r = proto.CloneOf(r)
			r.Setting = redactSetting(r.Setting)
			return r
		case *v1pb.CreateIdentityProviderRequest:
			r = proto.CloneOf(r)
			r.IdentityProvider = redactIdentityProvider(r.IdentityProvider)
			return r
		case *v1pb.CreateSheetRequest:
			// The clone is already private; drop the content in place instead
			// of cloning the potentially large sheet a second time.
			r = proto.CloneOf(r)
			if r.Sheet != nil {
				r.Sheet.Content = nil
			}
			return r
		case *v1pb.BatchCreateSheetsRequest:
			r = proto.CloneOf(r)
			for _, cr := range r.Requests {
				if cr.Sheet != nil {
					cr.Sheet.Content = nil
				}
			}
			return r
		case *v1pb.UpdateIdentityProviderRequest:
			r = proto.CloneOf(r)
			r.IdentityProvider = redactIdentityProvider(r.IdentityProvider)
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
			return redactExportResponse(r)
		case *v1pb.LoginResponse:
			return redactLoginResponse(r)
		case *v1pb.ExchangeTokenResponse:
			return redactExchangeTokenResponse(r)
		case *v1pb.RotateDirectorySyncTokenResponse:
			return redactRotateDirectorySyncTokenResponse(r)
		case *v1pb.ServiceAccount:
			return redactServiceAccount(r)
		case *v1pb.Setting:
			return redactSetting(r)
		case *v1pb.IdentityProvider:
			return redactIdentityProvider(r)
		case *v1pb.User:
			return redactUser(r)
		case *v1pb.Instance:
			return redactInstance(r)
		case *v1pb.Project:
			return redactProject(r)
		case *v1pb.Sheet:
			return redactSheet(r)
		case *v1pb.BatchCreateSheetsResponse:
			n := &v1pb.BatchCreateSheetsResponse{}
			for _, sheet := range r.Sheets {
				n.Sheets = append(n.Sheets, redactSheet(sheet))
			}
			return n
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

// redactExportResponse drops the exported file content but keeps the applied
// access grant so audit logs record which grant authorized the export.
func redactExportResponse(r *v1pb.ExportResponse) *v1pb.ExportResponse {
	if r == nil || r.AppliedAccessGrant == "" {
		return nil
	}
	return &v1pb.ExportResponse{
		AppliedAccessGrant: r.AppliedAccessGrant,
	}
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
	if r.EmailCode != nil {
		r.EmailCode = &maskedString
	}
	if r.IdpContext != nil {
		r.IdpContext = nil
	}
	return r
}

func redactSignupRequest(r *v1pb.SignupRequest) *v1pb.SignupRequest {
	if r == nil {
		return nil
	}
	r = proto.CloneOf(r)
	if r.Password != "" {
		r.Password = maskedString
	}
	return r
}

// redactExchangeTokenRequest masks the external OIDC JWT. The token is a
// credential — it could be replayed against the original IdP or, if logged,
// reveal workload identity claims. The caller's email is kept for audit
// correlation.
func redactExchangeTokenRequest(r *v1pb.ExchangeTokenRequest) *v1pb.ExchangeTokenRequest {
	if r == nil {
		return nil
	}
	r = proto.CloneOf(r)
	if r.Token != "" {
		r.Token = maskedString
	}
	return r
}

// redactExchangeTokenResponse drops the issued Bytebase access token. Logging
// it would give anyone with audit-log read access a valid API token for the
// workload identity. Returns an empty response so the audit entry still
// records that the call happened.
func redactExchangeTokenResponse(r *v1pb.ExchangeTokenResponse) *v1pb.ExchangeTokenResponse {
	if r == nil {
		return nil
	}
	return &v1pb.ExchangeTokenResponse{}
}

// redactRotateDirectorySyncTokenResponse drops the newly minted SCIM token. The
// whole point of hashing it at rest is that the plaintext exists only in the
// single response to the admin who rotated it; writing it to the audit log would
// hand a working credential to anyone who can read that log. Returns an empty
// response so the audit entry still records that the rotation happened.
func redactRotateDirectorySyncTokenResponse(r *v1pb.RotateDirectorySyncTokenResponse) *v1pb.RotateDirectorySyncTokenResponse {
	if r == nil {
		return nil
	}
	return &v1pb.RotateDirectorySyncTokenResponse{}
}

// redactServiceAccount drops the API key. Create and key rotation are the only
// responses that carry it — the read path never populates it — and it is a live
// credential, so logging it would hand a working key to anyone who can read the
// audit log or its stdout stream.
func redactServiceAccount(r *v1pb.ServiceAccount) *v1pb.ServiceAccount {
	if r == nil {
		return nil
	}
	r = proto.CloneOf(r)
	if r.ServiceKey != "" {
		r.ServiceKey = maskedString
	}
	return r
}

// redactIdentityProvider masks the IdP credentials. The read path already blanks
// these (convertToIdentityProvider), so they only reach the audit log through
// the create/update request payload.
func redactIdentityProvider(r *v1pb.IdentityProvider) *v1pb.IdentityProvider {
	if r == nil {
		return nil
	}
	r = proto.CloneOf(r)
	switch config := r.GetConfig().GetConfig().(type) {
	case *v1pb.IdentityProviderConfig_Oauth2Config:
		if config.Oauth2Config.GetClientSecret() != "" {
			config.Oauth2Config.ClientSecret = maskedString
		}
	case *v1pb.IdentityProviderConfig_OidcConfig:
		if config.OidcConfig.GetClientSecret() != "" {
			config.OidcConfig.ClientSecret = maskedString
		}
	case *v1pb.IdentityProviderConfig_LdapConfig:
		if config.LdapConfig.GetBindPassword() != "" {
			config.LdapConfig.BindPassword = maskedString
		}
	default:
	}
	return r
}

func redactWebhook(w *v1pb.Webhook) *v1pb.Webhook {
	if w == nil {
		return nil
	}
	cloned := proto.CloneOf(w)
	if cloned.Url != "" {
		cloned.Url = maskedString
	}
	return cloned
}

func redactProject(p *v1pb.Project) *v1pb.Project {
	if p == nil {
		return nil
	}
	cloned := proto.CloneOf(p)
	for i, webhook := range cloned.Webhooks {
		cloned.Webhooks[i] = redactWebhook(webhook)
	}
	return cloned
}

// redactSetting masks every credential a settings payload can carry. The read
// path blanks these, so they reach the audit log only through UpdateSetting's
// request. Each secret is masked rather than dropped so the log still records
// that the field was being written.
func redactSetting(r *v1pb.Setting) *v1pb.Setting {
	if r == nil {
		return nil
	}
	r = proto.CloneOf(r)
	switch value := r.GetValue().GetValue().(type) {
	case *v1pb.SettingValue_Email:
		if smtp := value.Email.GetSmtp(); smtp.GetPassword() != "" {
			smtp.Password = maskedString
		}
	case *v1pb.SettingValue_Ai:
		if value.Ai.GetApiKey() != "" {
			value.Ai.ApiKey = maskedString
		}
	case *v1pb.SettingValue_AppIm:
		maskAppIMSecrets(value.AppIm)
	default:
	}
	return r
}

func maskAppIMSecrets(s *v1pb.AppIMSetting) {
	for _, setting := range s.GetSettings() {
		if v := setting.GetSlack(); v.GetToken() != "" {
			v.Token = maskedString
		}
		if v := setting.GetFeishu(); v.GetAppSecret() != "" {
			v.AppSecret = maskedString
		}
		if v := setting.GetWecom(); v.GetSecret() != "" {
			v.Secret = maskedString
		}
		if v := setting.GetLark(); v.GetAppSecret() != "" {
			v.AppSecret = maskedString
		}
		if v := setting.GetDingtalk(); v.GetClientSecret() != "" {
			v.ClientSecret = maskedString
		}
		if v := setting.GetTeams(); v.GetClientSecret() != "" {
			v.ClientSecret = maskedString
		}
	}
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
		Name:  r.Name,
		Email: r.Email,
		Title: r.Title,
	}
}

// redactSheet strips the sheet content from audit payloads. Sheets carry full
// SQL statements (potentially megabytes); the audit record keeps the resource
// name and content size.
func redactSheet(s *v1pb.Sheet) *v1pb.Sheet {
	if s == nil {
		return nil
	}
	cloned := proto.CloneOf(s)
	cloned.Content = nil
	return cloned
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
	if cloned.SslCaPath != "" {
		cloned.SslCaPath = maskedString
	}
	if cloned.SslCert != "" {
		cloned.SslCert = maskedString
	}
	if cloned.SslCertPath != "" {
		cloned.SslCertPath = maskedString
	}
	if cloned.SslKey != "" {
		cloned.SslKey = maskedString
	}
	if cloned.SslKeyPath != "" {
		cloned.SslKeyPath = maskedString
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
		Results:            nil,
		AppliedAccessGrant: r.AppliedAccessGrant,
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
	if val.Kind() == reflect.Pointer && val.IsNil() {
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
