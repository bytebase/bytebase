// Package auth handles the auth of gRPC server.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/golang-jwt/jwt/v5"
	errs "github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	issuer = "bytebase"
	// Signing key section. For now, this is only used for signing, not for verifying since we only
	// have 1 version. But it will be used to maintain backward compatibility if we change the signing mechanism.
	keyID = "v1"
	// AccessTokenAudience is the audience for user access tokens.
	AccessTokenAudience = "bb.user.access"
	// MFATempTokenAudience is the audience for MFA temporary tokens.
	MFATempTokenAudience = "bb.user.mfa-temp"
	// OAuth2AccessTokenAudience is the audience OAuth2 access tokens carried
	// before they were bound to the MCP resource URI (P1a PR 3). No longer
	// minted; still recognized at /mcp so tokens issued by a pre-upgrade
	// release keep working there until they expire. The general API refuses
	// it like any other MCP-minted token.
	OAuth2AccessTokenAudience = "bb.oauth2.access"
	// TokenUseMCP is the token_use claim value marking a token as an MCP OAuth2
	// credential. The audience of such a token is a per-deployment resource URI,
	// so consumers that need to recognize "an MCP token, whatever the deployment"
	// (the general API interceptor, the SwitchWorkspace guard) key on this claim
	// instead of the audience.
	TokenUseMCP      = "mcp"
	apiTokenDuration = 1 * time.Hour
	// DefaultAccessTokenDuration is the default access token expiration duration.
	DefaultAccessTokenDuration = 1 * time.Hour
	// OAuth2AccessTokenDuration is the lifetime of OAuth2 (MCP) access tokens,
	// minted by the oauth2 token endpoint. It also bounds how long legacy
	// bb.oauth2.access acceptance at /mcp outlives an upgrade: nothing mints
	// that audience anymore, so acceptance drains no later than one of these
	// lifetimes after the last legacy-capable replica leaves service.
	OAuth2AccessTokenDuration = 1 * time.Hour
	// DefaultRefreshTokenDuration is the default refresh token expiration duration.
	DefaultRefreshTokenDuration = 7 * 24 * time.Hour

	// AccessTokenCookieName is the cookie name of access token.
	AccessTokenCookieName = "access-token"
	// RefreshTokenCookieName is the cookie name of refresh token.
	RefreshTokenCookieName = "refresh-token"
)

// APIAuthInterceptor is the auth interceptor for gRPC server.
type APIAuthInterceptor struct {
	store          *store.Store
	secret         string
	licenseService *enterprise.LicenseService
	bus            *bus.Bus
	profile        *config.Profile
}

// New returns a new API auth interceptor.
func New(
	store *store.Store,
	secret string,
	licenseService *enterprise.LicenseService,
	bus *bus.Bus,
	profile *config.Profile,
) *APIAuthInterceptor {
	return &APIAuthInterceptor{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		bus:            bus,
		profile:        profile,
	}
}

// WrapUnary implements the ConnectRPC interceptor interface for unary RPCs.
func (in *APIAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		accessTokenStr, err := GetTokenFromHeaders(req.Header())
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		authContext, err := getAuthContext(req.Spec().Procedure)
		if err != nil {
			return nil, err
		}
		ctx = context.WithValue(ctx, common.AuthContextKey, authContext)

		user, workspaceID, err := in.getUserConnect(ctx, accessTokenStr)
		if err != nil {
			if IsAuthenticationSkipped(req.Spec().Procedure, authContext) {
				return next(ctx, req)
			}
			return nil, err
		}

		ctx = context.WithValue(ctx, common.UserContextKey, user)
		ctx = context.WithValue(ctx, common.WorkspaceIDContextKey, workspaceID)
		return next(ctx, req)
	}
}

// WrapStreamingClient implements the ConnectRPC interceptor interface for streaming clients.
func (*APIAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

// WrapStreamingHandler implements the ConnectRPC interceptor interface for streaming handlers.
func (in *APIAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		accessTokenStr, err := GetTokenFromHeaders(conn.RequestHeader())
		if err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}

		authContext, err := getAuthContext(conn.Spec().Procedure)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, common.AuthContextKey, authContext)

		user, claims, err := in.authenticate(ctx, accessTokenStr)
		if err != nil {
			if IsAuthenticationSkipped(conn.Spec().Procedure, authContext) {
				return next(ctx, conn)
			}
			return connect.NewError(connect.CodeUnauthenticated, err)
		}

		in.profile.LastActiveTS.Store(time.Now().Unix())
		ctx = context.WithValue(ctx, common.UserContextKey, user)
		ctx = context.WithValue(ctx, common.WorkspaceIDContextKey, claims.WorkspaceID)

		var tokenExpiry time.Time
		if claims.ExpiresAt != nil {
			tokenExpiry = claims.ExpiresAt.Time
		}

		return next(ctx, &authStreamingConn{
			StreamingHandlerConn: conn,
			tokenExpiry:          tokenExpiry,
		})
	}
}

// authStreamingConn wraps a streaming connection to check token expiry on every received message.
type authStreamingConn struct {
	connect.StreamingHandlerConn
	tokenExpiry time.Time
}

func (c *authStreamingConn) Receive(msg any) error {
	if !c.tokenExpiry.IsZero() && time.Now().After(c.tokenExpiry) {
		return connect.NewError(connect.CodeUnauthenticated, errs.New("access token expired"))
	}
	return c.StreamingHandlerConn.Receive(msg)
}

// isMCPProvenance reports whether parsed claims identify a token minted by
// the MCP authorization server: the modern shape carries token_use=mcp
// (whatever audience it also carries), and the legacy pre-PR-3 shape is
// recognized by its bb.oauth2.access audience, which nothing else ever
// minted. The general-API rejection and IsMCPOriginatedToken (behind
// SwitchWorkspace's guard) key on this one definition — keep them keying on
// it: two security decision points drifting apart is how one surface ends up
// admitting what the other refuses.
func isMCPProvenance(tokenUse string, audience jwt.ClaimStrings) bool {
	return tokenUse == TokenUseMCP || audienceContains(audience, OAuth2AccessTokenAudience)
}

// checkTokenAudience decides whether a token's audience admits it to the
// general (non-/mcp) API. An MCP token (isMCPProvenance) is refused outright:
// since PR 4's private in-memory transport, /mcp tool traffic authenticates
// with the internal delegated credential, so nothing legitimate presents one
// here anymore and admitting it would keep it a universal API bearer (P1a PR
// 5, retiring PR 3's audit-only admission; the legacy audience keeps draining
// at /mcp only, where old-replica tool traffic genuinely needs it during a
// rolling upgrade). Web session tokens pass; anything else is refused.
func checkTokenAudience(claims *claimsMessage) error {
	if isMCPProvenance(claims.TokenUse, claims.Audience) {
		// Warn rather than refuse silently: the audit-only observation window
		// the spec planned never shipped in a release, so this is the only
		// server-side signal that an integration was relying on the old
		// admission. Denial auditing proper is PR 5b.
		slog.Warn("refused an MCP token on the general API",
			slog.String("principal", claims.Subject),
			slog.String("workspace", claims.WorkspaceID),
			slog.String("audience", strings.Join(claims.Audience, ",")),
			slog.String("token_use", claims.TokenUse))
		return errs.New("MCP tokens are only accepted at /mcp; use a service account for the API")
	}
	if audienceContains(claims.Audience, AccessTokenAudience) {
		return nil
	}
	return errs.Errorf(
		"invalid access token, audience mismatch, got %q, expected %q",
		claims.Audience,
		AccessTokenAudience,
	)
}

// authenticate is the shared authentication logic that validates JWT tokens.
// Returns the user and claims, or an error. This is the single source of truth
// for token validation.
func (in *APIAuthInterceptor) authenticate(ctx context.Context, accessTokenStr string) (*store.UserMessage, *claimsMessage, error) {
	if accessTokenStr == "" {
		return nil, nil, errs.New("access token not found")
	}

	claims := &claimsMessage{}
	if _, err := jwt.ParseWithClaims(accessTokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errs.Errorf("unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(in.secret), nil
			}
		}
		return nil, errs.Errorf("unexpected access token kid=%v", t.Header["kid"])
	}); err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, nil, errs.New("access token expired")
		}
		return nil, nil, errs.New("failed to parse claim")
	}

	if err := checkTokenAudience(claims); err != nil {
		return nil, nil, err
	}

	user, err := resolvePrincipal(ctx, in.store, in.profile, claims.Subject, claims.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}
	return user, claims, nil
}

// resolvePrincipal turns a verified token's identity claims into a live user:
// account lookup, deactivation check, workspace-membership verification, and
// principal resolution. Shared by the public interceptor and the internal MCP
// interceptor so both surfaces re-resolve identity state identically on every
// request — a credential is never trusted for anything beyond who and where.
func resolvePrincipal(ctx context.Context, stores *store.Store, profile *config.Profile, subject, workspaceID string) (*store.UserMessage, error) {
	account, err := stores.GetAccountByEmail(ctx, subject)
	if err != nil {
		return nil, errs.Errorf("failed to find principal %q in the access token", subject)
	}
	if account == nil {
		return nil, errs.Errorf("principal %q not exists in the access token", subject)
	}
	if account.MemberDeleted {
		return nil, errs.Errorf("principal %q has been deactivated by administrators", account.Email)
	}

	// Verify workspace membership.
	// We always require workspace_id in the claims even for non-SaaS (single workspace) mode
	if workspaceID == "" {
		return nil, errs.New("empty workspace in the token")
	}
	if err := verifyWorkspaceMembership(ctx, stores, profile, workspaceID, account); err != nil {
		return nil, err
	}

	// Convert to UserMessage for context storage.
	user, err := stores.ResolvePrincipalAsUser(ctx, account)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errs.Errorf("user %q not found", account.Email)
	}
	return user, nil
}

// verifyWorkspaceMembership checks that the account is a member of the workspace.
func verifyWorkspaceMembership(ctx context.Context, stores *store.Store, profile *config.Profile, workspaceID string, account *store.AccountMessage) error {
	switch account.Type {
	case storepb.PrincipalType_SERVICE_ACCOUNT, storepb.PrincipalType_WORKLOAD_IDENTITY:
		// Service accounts and workload identities have workspace on their record.
		if account.Workspace != workspaceID {
			return errs.Errorf("principal %q does not belong to workspace %q", account.Email, workspaceID)
		}
		return nil

	case storepb.PrincipalType_END_USER:
		// END_USER membership is verified via workspace IAM policy.
		// Check direct membership, group membership, and allUsers.
		iamPolicy, err := stores.GetWorkspaceIamPolicy(ctx, workspaceID)
		if err != nil {
			return errs.Wrap(err, "failed to get workspace IAM policy")
		}
		userMember := common.FormatUserEmail(account.Email)
		for _, binding := range iamPolicy.Policy.Bindings {
			for _, member := range binding.Members {
				if member == userMember {
					return nil
				}
				if member == common.AllUsers && !profile.SaaS {
					return nil
				}
				// Check group membership.
				if strings.HasPrefix(member, common.GroupPrefix) {
					groupMembers, _ := stores.GetGroupMembersSnapshot(ctx, workspaceID, member)
					if groupMembers != nil && groupMembers[userMember] {
						return nil
					}
				}
			}
		}
		return errs.Errorf("user %q is not a member of workspace %q", account.Email, workspaceID)

	default:
		return errs.Errorf("unknown principal type %v", account.Type)
	}
}

// authenticateConnect is a ConnectRPC-specific version that returns ConnectRPC errors.
func (in *APIAuthInterceptor) authenticateConnect(ctx context.Context, accessTokenStr string) (*store.UserMessage, *claimsMessage, error) {
	user, claims, err := in.authenticate(ctx, accessTokenStr)
	if err != nil {
		return nil, nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return user, claims, nil
}

// getUserConnect is a ConnectRPC-specific version that returns ConnectRPC errors.
// Returns the user and workspace ID from the token claims.
func (in *APIAuthInterceptor) getUserConnect(ctx context.Context, accessTokenStr string) (*store.UserMessage, string, error) {
	user, claims, err := in.authenticateConnect(ctx, accessTokenStr)
	if err != nil {
		return nil, "", err
	}

	// Only update for authorized request.
	in.profile.LastActiveTS.Store(time.Now().Unix())
	return user, claims.WorkspaceID, nil
}

// GetUserEmailFromMFATempToken returns the user email from the MFA temp token.
func GetUserEmailFromMFATempToken(token string, secret string) (string, error) {
	userEmail, _, err := GetUserEmailAndLoginMethodFromMFATempToken(token, secret)
	return userEmail, err
}

// GetUserEmailAndLoginMethodFromMFATempToken returns the user email and original login method from the MFA temp token.
func GetUserEmailAndLoginMethodFromMFATempToken(token, secret string) (string, string, error) {
	claims := &claimsMessage{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, connect.NewError(connect.CodeUnauthenticated, errs.Errorf("unexpected MFA temp token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256))
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(secret), nil
			}
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, errs.Errorf("unexpected MFA temp token kid=%v", t.Header["kid"]))
	})
	if err != nil {
		return "", "", connect.NewError(connect.CodeUnauthenticated, errs.New("failed to parse claim"))
	}
	if !audienceContains(claims.Audience, MFATempTokenAudience) {
		return "", "", connect.NewError(connect.CodeUnauthenticated, errs.New("invalid MFA temp token, audience mismatch"))
	}
	return claims.Subject, claims.LoginMethod, nil
}

// AuthenticateToken validates a JWT access token and returns the user and token expiry.
// This is a non-ConnectRPC version that returns regular errors instead of ConnectRPC errors.
// Its only caller is the LSP websocket handshake.
func (in *APIAuthInterceptor) AuthenticateToken(ctx context.Context, accessTokenStr string) (*store.UserMessage, string, time.Time, error) {
	user, claims, err := in.authenticate(ctx, accessTokenStr)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	var tokenExpiry time.Time
	if claims.ExpiresAt != nil {
		tokenExpiry = claims.ExpiresAt.Time
	}

	return user, claims.WorkspaceID, tokenExpiry, nil
}

// GetTokenFromHeaders extracts the access token from HTTP headers for ConnectRPC.
func GetTokenFromHeaders(headers http.Header) (string, error) {
	// Check Authorization header first
	authHeader := headers.Get("Authorization")
	if authHeader != "" {
		authHeaderParts := strings.Fields(authHeader)
		if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
			return "", errs.Errorf("authorization header format must be Bearer {token}")
		}
		return authHeaderParts[1], nil
	}

	// Check HTTP cookies
	var accessToken string
	for _, cookieHeader := range headers.Values("cookie") {
		header := http.Header{}
		header.Add("Cookie", cookieHeader)
		request := http.Request{Header: header}
		if cookie, _ := request.Cookie(AccessTokenCookieName); cookie != nil {
			accessToken = cookie.Value
			break
		}
	}
	return accessToken, nil
}

func audienceContains(audience jwt.ClaimStrings, token string) bool {
	return slices.Contains(audience, token)
}

func getAuthContext(fullMethod string) (*common.AuthContext, error) {
	methodTokens := strings.Split(fullMethod, "/")
	if len(methodTokens) != 3 {
		return nil, errs.Errorf("invalid full method name %q", fullMethod)
	}
	rd, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(methodTokens[1]))
	if err != nil {
		return nil, errs.Wrapf(err, "invalid registry service descriptor, full method name %q", fullMethod)
	}
	sd, ok := rd.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, errs.Errorf("invalid service descriptor, full method name %q", fullMethod)
	}
	md, ok := sd.Methods().ByName(protoreflect.Name(methodTokens[2])).Options().(*descriptorpb.MethodOptions)
	if !ok {
		return nil, errs.Errorf("invalid method options, full method name %q", fullMethod)
	}
	allowWithoutCredentialAny := proto.GetExtension(md, v1pb.E_AllowWithoutCredential)
	allowWithoutCredential, ok := allowWithoutCredentialAny.(bool)
	if !ok {
		return nil, errs.Errorf("invalid allow without credential extension, full method name %q", fullMethod)
	}
	permissionAny := proto.GetExtension(md, v1pb.E_Permission)
	permission, ok := permissionAny.(string)
	if !ok {
		return nil, errs.Errorf("invalid permission extension, full method name %q", fullMethod)
	}
	authMethodAny := proto.GetExtension(md, v1pb.E_AuthMethod)
	am, ok := authMethodAny.(v1pb.AuthMethod)
	if !ok {
		return nil, errs.Errorf("invalid auth method extension, full method name %q", fullMethod)
	}
	var authMethod common.AuthMethod
	switch am {
	case v1pb.AuthMethod_AUTH_METHOD_UNSPECIFIED:
		authMethod = common.AuthMethodUnspecified
	case v1pb.AuthMethod_IAM:
		authMethod = common.AuthMethodIAM
	case v1pb.AuthMethod_CUSTOM:
		authMethod = common.AuthMethodCustom
	default:
		return nil, errs.Errorf("unknown auth method %v for full method name %q", am, fullMethod)
	}
	auditAny := proto.GetExtension(md, v1pb.E_Audit)
	audit, ok := auditAny.(bool)
	if !ok {
		return nil, errs.Errorf("invalid audit extension, full method name %q", fullMethod)
	}
	mcpMethodClassAny := proto.GetExtension(md, v1pb.E_McpMethodClass)
	mcpMethodClass, ok := mcpMethodClassAny.(v1pb.MCPMethodClass)
	if !ok {
		return nil, errs.Errorf("invalid MCP method class extension, full method name %q", fullMethod)
	}
	mcpDenialReasonAny := proto.GetExtension(md, v1pb.E_McpDenialReason)
	mcpDenialReason, ok := mcpDenialReasonAny.(v1pb.MCPDenialReason)
	if !ok {
		return nil, errs.Errorf("invalid MCP denial reason extension, full method name %q", fullMethod)
	}

	return &common.AuthContext{
		AllowWithoutCredential: allowWithoutCredential,
		Permission:             permission,
		AuthMethod:             authMethod,
		Audit:                  audit,
		MCPMethodClass:         mcpMethodClass,
		MCPDenialReason:        mcpDenialReason,
	}, nil
}

// MCPClassIsRefused reports whether NO MCP capability ceiling serves methods of
// this class, so a session can never call one whatever the workspace is
// configured for. It exists here rather than beside the gate because two
// packages need the same answer for different reasons — the gate refuses on it,
// and the MCP OpenAPI index declines to advertise on it — and they must not
// disagree about which methods an agent is offered.
//
// It is a predicate rather than a copy of the serving table: the table lives in
// backend/api/v1 with the code that evaluates it, and
// TestLintRefusedClassesMatchTheServingTable holds this function against it, so
// adding a ceiling mode that serves a class cannot leave this stale.
//
// UNSPECIFIED counts as refused. The gate refuses an unclassified method, and
// CI rejects one, so advertising it would offer work that cannot be done.
func MCPClassIsRefused(class v1pb.MCPMethodClass) bool {
	switch class {
	case v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE:
		return false
	default:
		return true
	}
}

// MCPCeilingServesAnything reports whether a workspace capability ceiling
// serves any method class at all, which is what the /mcp connection gate needs
// to decide whether a session may open.
//
// It lives here for the same reason MCPClassIsRefused does: the serving table
// is in backend/api/v1 with the code that evaluates it, the connection gate is
// in backend/api/mcp, and the two must not disagree about which ceilings this
// build serves. TestLintCeilingAdmissionMatchesTheServingTable holds this
// function against that table over the whole enum, so a mode that starts or
// stops serving a class cannot leave the connection gate stale.
//
// Everything this build cannot interpret is refused: UNSPECIFIED is a zero
// value nobody resolved, and a stored number no release ever wrote — the
// reserved 2, or anything from a newer build — is a ceiling nobody decided
// about. Both fall to the default.
func MCPCeilingServesAnything(capability storepb.WorkspaceProfileSetting_MCPCapability) bool {
	switch capability {
	case storepb.WorkspaceProfileSetting_READ_WRITE, storepb.WorkspaceProfileSetting_READ_ONLY:
		return true
	default:
		return false
	}
}

// MCPMethodClassOfProcedure resolves the MCP classification for a connect
// procedure path such as "/bytebase.v1.AuthService/Login". Callers that only
// decide what to ADVERTISE may treat an error as "not classified"; the
// enforcement path must not — it reads the class off the AuthContext, which
// fails the request outright when the annotation cannot be resolved.
func MCPMethodClassOfProcedure(procedure string) (v1pb.MCPMethodClass, error) {
	tokens := strings.Split(procedure, "/")
	if len(tokens) != 3 {
		return v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED, errs.Errorf("invalid procedure name %q", procedure)
	}
	rd, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(tokens[1]))
	if err != nil {
		return v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED, errs.Wrapf(err, "invalid service descriptor for procedure %q", procedure)
	}
	sd, ok := rd.(protoreflect.ServiceDescriptor)
	if !ok {
		return v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED, errs.Errorf("invalid service descriptor for procedure %q", procedure)
	}
	method := sd.Methods().ByName(protoreflect.Name(tokens[2]))
	if method == nil {
		return v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED, errs.Errorf("unknown method for procedure %q", procedure)
	}
	md, ok := method.Options().(*descriptorpb.MethodOptions)
	if !ok {
		return v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED, errs.Errorf("invalid method options for procedure %q", procedure)
	}
	class, ok := proto.GetExtension(md, v1pb.E_McpMethodClass).(v1pb.MCPMethodClass)
	if !ok {
		return v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED, errs.Errorf("invalid MCP method class extension for procedure %q", procedure)
	}
	return class, nil
}
