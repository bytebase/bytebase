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

const (
	// unencodablePayload stands in for a payload protojson refused, so the row
	// records that something was there and could not be written down.
	unencodablePayload = `{"_bytebase":"payload could not be encoded"}`
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

		// The method's audit annotation decides whether there is a row; this
		// mark decides which calls to a DENIALS method count.
		//
		// Three refusals mark: the ACL interceptor on both chains, and on the
		// internal chain the MCP ceiling gate and the read-only clamp inside
		// SQLService.Query. Marking is what qualifies, not where the code sits
		// — the clamp is handler code. A method's own permission check inside
		// its handler marks nothing, so its refusals reach the log only under
		// ALL.
		//
		// Recording a request that was never recorded is why redaction has to
		// cover more than the audited RPCs: a denial must not transcribe the
		// secret it refused. Since redaction is driven by the field annotation
		// rather than by a per-RPC redactor, a refused method is covered the
		// moment its fields are annotated. The population is every v1 method, on
		// either chain.
		policyDenied := false
		ctx = common.WithSetPolicyDenied(ctx, func() { policyDenied = true })

		startTime := time.Now()
		response, rerr := next(ctx, req)
		latency := time.Since(startTime)

		if needAudit(ctx, policyDenied) {
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
				policyDenied:            policyDenied,
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
		// A stream has no denial to pass: aclStreamingConn refuses in Receive,
		// which never reaches the Send that writes rows. See the DEFER in
		// acl.go.
		if !needAudit(ctx, false) {
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
	// common.SetAuditWorkspaceID. Used as the validated audit parent for
	// allow_without_credential methods where authContext.Resources is empty
	// because no workspace is in the context.
	handlerAuditWorkspaceID string
	rerr                    error
	headers                 http.Header
	peerAddr                string
	latency                 time.Duration
	// policyDenied is set via common.SetPolicyDenied. With the method's audit
	// annotation it decides whether there is a row, and it decides the severity.
	policyDenied bool
}

func (in *AuditInterceptor) createAuditLog(ctx context.Context, e *auditEntry) error {
	// Skip audit logging for validate-only requests that SUCCEEDED. A dry run
	// that Bytebase accepted changed nothing, so recording it is noise — that
	// is the whole reason for the skip.
	//
	// The outcome is what decides, not the flag. A refused attempt is worth
	// exactly as much as any other refused attempt, and six request messages
	// carry validate_only — UpdateDataSource, AddDataSource, CreateInstance,
	// CreateIdentityProvider, UpdateSetting and CreateDatabaseGroup, every one
	// of them audited, three of them on methods MCP forbids. Skipping on the
	// flag alone made setting it a switch that turns off the record of being
	// caught.
	//
	// EVERY failure records, not only a policy denial, and that is deliberate
	// rather than incidental. The instance form runs a validate-only
	// connection test before each save and on each Test Connection click, so
	// the rows this adds are mostly failed connection tests, not refusals —
	// a real volume change on a common flow. Keying on a denial code instead
	// would be the same shape of hole this closes: it would record the
	// refusals that happen to carry that code and silently drop every other
	// rejected attempt.
	if e.rerr == nil && isValidateOnlyRequest(e.request) {
		return nil
	}

	requestString := marshalAuditPayload(e.request)
	responseString := marshalAuditPayload(e.response)

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
	// without a workspace-bound caller, Resources is empty; fall back to the
	// workspace the handler announced, or any workspace embedded in the
	// response, so the audit entry is still written.
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
	// The reset flow is allow_without_credential, so an unauthenticated caller
	// could name any workspace and write rows into it. Its audit parent
	// therefore comes only from the workspace the HANDLER validated, and never
	// from the request or the fallback below.
	//
	// A request carrying a delegated MCP grant is the one exception, because
	// its workspace was not named by the caller: the internal MCP interceptor
	// verified the credential and bound the workspace before the request
	// reached this chain. Without the exception these three methods are the
	// last silent denials in the system — the ceiling gate refuses them before
	// dispatch, so the handler never runs and never announces a workspace —
	// and they are the flow that mails or consumes the secret a login accepts.
	// Presence of the grant is the marker, never a field value.
	handlerValidatedWorkspaceMethod := (e.method == v1connect.AuthServiceRequestPasswordResetProcedure ||
		e.method == v1connect.AuthServiceResetPasswordProcedure ||
		e.method == v1connect.AuthServiceSendEmailLoginCodeProcedure) &&
		authContext.DelegatedGrant == nil
	if handlerValidatedWorkspaceMethod {
		if e.handlerAuditWorkspaceID != "" {
			appendParent(auditParent{
				parent:           common.FormatWorkspace(e.handlerAuditWorkspaceID),
				auditWorkspaceID: e.handlerAuditWorkspaceID,
			})
		}
	} else {
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
	}
	if len(parents) == 0 && !handlerValidatedWorkspaceMethod {
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

	// service_data and status.details are the only fields on the row that need
	// redacting, and neither passes marshalAuditPayload. Both carry
	// google.protobuf.Any values whose packed type the descriptor cannot see:
	// service_data holds a before-image Setting or an IAM policy delta, and
	// status.details holds whatever a failing handler attached.
	//
	// The rest of the row is left alone deliberately. request_metadata and
	// mcp_delegation are ordinary store messages with nothing annotated under
	// them, and request and response are already redacted strings.
	// TestAuditRowNeedsNoRedactionBeyondTheAnyPayloads fails if that stops
	// being true.
	//
	// Computed once: none of the three varies by parent.
	serviceData := redactAuditServiceData(e.serviceData)
	auditStatus := redactAuditStatus(convertErrToStatus(e.rerr))
	mcpDelegation := mcpDelegationFromAuthContext(authContext)

	// status.code carries every failure, not only the refusals, so severity is
	// what separates a refused caller from routine traffic. The recovery
	// service already stamps WARNING for this class.
	severity := storepb.AuditLog_INFO
	if e.policyDenied {
		severity = storepb.AuditLog_WARNING
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
			Severity:        severity,
			User:            user,
			Request:         requestString,
			Response:        responseString,
			Status:          auditStatus,
			Latency:         durationpb.New(e.latency),
			ServiceData:     serviceData,
			RequestMetadata: requestMetadata,
			McpDelegation:   mcpDelegation,
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
			common.LogAuditToStdout(ctx, p)
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
		return normalizeEmail(r.GetEmail())
	case *v1pb.SignupRequest:
		return r.GetEmail()
	case *v1pb.ExchangeTokenRequest:
		return r.GetEmail()
	case *v1pb.RequestPasswordResetRequest:
		return normalizeEmail(r.GetEmail())
	case *v1pb.ResetPasswordRequest:
		return normalizeEmail(r.GetEmail())
	case *v1pb.SendEmailLoginCodeRequest:
		return normalizeEmail(r.GetEmail())
	case *v1pb.CreateInstanceRequest:
		if r.GetParent() == "" {
			return common.FormatInstance(r.GetInstanceId())
		}
		if projectID, err := common.GetProjectID(r.GetParent()); err == nil {
			return common.FormatProjectInstance(projectID, r.GetInstanceId())
		}
		return ""
	case *v1pb.PrepareSampleProjectInstanceRequest:
		return r.GetParent()
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

// marshalAuditPayload renders one side of an audited call. Redaction is driven
// by the (bytebase.v1.audit_behavior) annotation on the message's own
// descriptor — see audit_redact.go — so nothing here switches on a type, and a
// field is protected on every RPC that carries it rather than on the ones
// someone wrote a redactor for.
//
// One function for both directions: the annotation says what to do with a
// field, and which side of the call it arrived on does not change that. It is
// called from createAuditLog rather than from WrapUnary because the streaming
// path builds its own auditEntry and calls createAuditLog directly, so a walk
// that lived in the unary interceptor would skip AdminExecute entirely.
func marshalAuditPayload(payload any) string {
	if payload == nil {
		return ""
	}
	// IsNil panics on a kind that cannot be nil, and Receive/Send stash
	// whatever the stream hands them, so the kind is checked first.
	if value := reflect.ValueOf(payload); value.Kind() == reflect.Pointer && value.IsNil() {
		return ""
	}
	message, ok := payload.(proto.Message)
	if !ok {
		return ""
	}
	b, err := protojson.Marshal(redactForAudit(message))
	if err != nil {
		// A payload that will not encode must not cost the record. protojson
		// rejects invalid UTF-8, and QueryResult.error and .statement are
		// filled straight from the driver — so one bad byte from an Oracle or
		// MySQL connection in a non-UTF-8 charset used to mean either no audit
		// row at all (unary, where WrapUnary swallows the error) or a killed
		// AdminExecute stream. Degrade the payload instead; everything else on
		// the row, including the method and the outcome, still gets written.
		slog.Warn("audit: payload could not be encoded, recording a placeholder",
			log.BBError(err), slog.String("type", string(message.ProtoReflect().Descriptor().FullName())))
		return unencodablePayload
	}
	return string(b)
}

// markPolicyDenied records that a policy refused this request and returns the
// error unchanged, so a DENIALS method's refusal reaches the audit log.
//
// Two kinds of caller. The ACL interceptor calls it at each of its verdicts,
// not once on every error it returns: doACLCheck also returns store failures, a
// missing auth context and unresolvable resource names, and the mark is for
// decisions about the caller. A CUSTOM-auth handler calls it wherever it makes
// the same IAM verdict ACL would have made — CheckPermission says no — because
// for those methods the handler IS the access-control layer, and without the
// mark their audit = DENIALS annotation would be false.
//
// TestLintHandlerIAMVerdictsAreMarked holds that second population.
//
// DEFER: a streaming denial reaches this too, but nothing records it —
// AuditInterceptor.WrapStreamingHandler returns early on !needAudit and never
// registers a setter, and it writes its rows on Send, which a denial in
// Receive never reaches. AdminExecute is the only streaming RPC; upgrade when
// a second one lands or AdminExecute's denials are wanted.
func markPolicyDenied(ctx context.Context, err error) error {
	common.SetPolicyDenied(ctx)
	return err
}

// needAudit reports whether this call reaches the audit log.
//
// The annotation declares the method's eligibility; the mark says what
// happened to this call. The mark selects within the declared mode and never
// widens it.
//
// A lint requires every v1 method to declare a mode, so UNSPECIFIED here means
// a call arrived with no auth context, or a build that was never linted.
func needAudit(ctx context.Context, denied bool) bool {
	authCtx, ok := common.GetAuthContextFromContext(ctx)
	if !ok {
		slog.Warn("audit interceptor: failed to get auth context")
		return false
	}
	switch authCtx.AuditMode {
	case v1pb.AuditMode_ALL:
		return true
	case v1pb.AuditMode_DENIALS:
		return denied
	default:
		return false
	}
}

// getRequestMetadataFromHeaders extracts request metadata from HTTP headers for ConnectRPC.
func getRequestMetadataFromHeaders(headers http.Header, peerAddr string) *storepb.RequestMetadata {
	// The forwarding headers first, then the peer address ConnectRPC reports
	// for a direct connection.
	callerIP := common.CallerIPFromHeaders(headers)
	if callerIP == "" {
		callerIP = common.StripPort(peerAddr)
	}
	return &storepb.RequestMetadata{
		CallerIp:                callerIP,
		CallerSuppliedUserAgent: headers.Get("User-Agent"),
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
