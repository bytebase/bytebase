package v1

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/idp/oidc"
	"github.com/bytebase/bytebase/backend/plugin/idp/wif"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	// mfaTempTokenDuration is the duration for MFA temporary tokens.
	// Following industry standards (Okta: 5 minutes, Auth0: 10 minutes, AWS Cognito: 3 minutes).
	// A short duration reduces the attack window for TOTP brute-force attempts.
	mfaTempTokenDuration = 5 * time.Minute

	// Error messages for authentication failures.
	errMsgInvalidCredentials  = "invalid email or password"
	errMsgInvalidMFACode      = "invalid MFA code"
	errMsgInvalidRecoveryCode = "invalid recovery code"
)

const setCookieHeader = "Set-Cookie"

var (
	invalidCredentialsError = connect.NewError(connect.CodeUnauthenticated, errors.New(errMsgInvalidCredentials))
)

type loginAuthMethod string

const (
	loginAuthMethodPassword  loginAuthMethod = "password"
	loginAuthMethodIDP       loginAuthMethod = "idp"
	loginAuthMethodEmailCode loginAuthMethod = "email_code"
)

func loginAuthMethodFromRequest(request *v1pb.LoginRequest) loginAuthMethod {
	if request.GetIdpName() != "" {
		return loginAuthMethodIDP
	}
	if request.EmailCode != nil && *request.EmailCode != "" {
		return loginAuthMethodEmailCode
	}
	return loginAuthMethodPassword
}

func loginAuthMethodFromMFATempToken(token, secret string) (string, loginAuthMethod, error) {
	email, method, err := auth.GetUserEmailAndLoginMethodFromMFATempToken(token, secret)
	if err != nil {
		return "", loginAuthMethodPassword, err
	}
	switch m := loginAuthMethod(method); m {
	case loginAuthMethodIDP, loginAuthMethodEmailCode:
		return email, m, nil
	default:
		return email, loginAuthMethodPassword, nil
	}
}

func (m loginAuthMethod) requiresPasswordReset() bool {
	return m == loginAuthMethodPassword
}

// AuthService implements the auth service.
type AuthService struct {
	v1connect.UnimplementedAuthServiceHandler
	store          *store.Store
	secret         string
	licenseService *enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewAuthService creates a new AuthService.
func NewAuthService(store *store.Store, secret string, licenseService *enterprise.LicenseService, profile *config.Profile, iamManager *iam.Manager) *AuthService {
	return &AuthService{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		profile:        profile,
		iamManager:     iamManager,
	}
}

// GetAuthenticationInfo returns everything the login page renders: the sign-in
// restrictions and the identity providers it offers.
func (s *AuthService) GetAuthenticationInfo(
	ctx context.Context,
	req *connect.Request[v1pb.GetAuthenticationInfoRequest],
) (*connect.Response[v1pb.AuthenticationInfo], error) {
	workspaceID, err := s.resolveAuthenticationWorkspaceID(ctx, &req.Msg.Workspace)
	if err != nil {
		return nil, err
	}

	restriction, err := getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, workspaceID)
	if err != nil {
		return nil, err
	}

	identityProviders, err := s.listLoginIdentityProviders(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	info := &v1pb.AuthenticationInfo{
		Restriction:       restriction,
		IdentityProviders: identityProviders,
	}
	if workspaceID != "" {
		info.Workspace = common.FormatWorkspace(workspaceID)
	}
	return connect.NewResponse(info), nil
}

// resolveAuthenticationWorkspaceID resolves the workspace targeted by an
// authentication request and announces it to the audit interceptor. An explicit
// existing workspace is used regardless of account membership; self-hosted falls
// back to the singleton workspace when the request omits it.
func (s *AuthService) resolveAuthenticationWorkspaceID(ctx context.Context, workspaceName *string) (string, error) {
	workspaceID, err := parseOptionalWorkspace(workspaceName)
	if err != nil {
		return "", connect.NewError(connect.CodeInvalidArgument, err)
	}
	if workspaceID != "" {
		workspace, err := s.store.GetWorkspaceByID(ctx, workspaceID)
		if err != nil {
			return "", connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get workspace"))
		}
		if workspace == nil {
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("workspace %q not found", common.FormatWorkspace(workspaceID)))
		}
	} else if !s.profile.SaaS {
		workspaceID, err = s.store.GetWorkspaceID(ctx)
		if err != nil {
			return "", connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get workspace"))
		}
	}
	if workspaceID != "" {
		common.SetAuditWorkspaceID(ctx, workspaceID)
	}
	return workspaceID, nil
}

// listLoginIdentityProviders returns the providers the login page renders for a
// workspace, falling back to the global providers when the workspace has none.
func (s *AuthService) listLoginIdentityProviders(ctx context.Context, workspaceID string) ([]*v1pb.LoginIdentityProvider, error) {
	find := &store.FindIdentityProviderMessage{}
	if workspaceID != "" {
		find.Workspace = &workspaceID
	}
	identityProviders, err := s.store.ListIdentityProviders(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to list identity providers"))
	}
	// Global providers are the SaaS shared login path. Self-hosted has none to
	// fall back to, and counting them there would disagree with the
	// workspace-scoped check UpdateSetting makes for disallow_password_signin.
	if len(identityProviders) == 0 && workspaceID != "" && s.profile.SaaS {
		identityProviders, err = s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to list global identity providers"))
		}
	}

	var loginIdentityProviders []*v1pb.LoginIdentityProvider
	for _, identityProvider := range identityProviders {
		loginIdentityProviders = append(loginIdentityProviders, convertToLoginIdentityProvider(identityProvider))
	}
	return loginIdentityProviders, nil
}

// convertToLoginIdentityProvider publishes the fields a browser needs to start
// an SSO redirect and nothing else. This response is served without a
// credential, so fields are added here by hand, never copied from the config.
func convertToLoginIdentityProvider(identityProvider *store.IdentityProviderMessage) *v1pb.LoginIdentityProvider {
	provider := &v1pb.LoginIdentityProvider{
		Name:  common.IdentityProviderNamePrefix + identityProvider.ResourceID,
		Type:  v1pb.IdentityProviderType(identityProvider.Type),
		Title: identityProvider.Title,
	}

	if v := identityProvider.Config.GetOauth2Config(); v != nil {
		provider.AuthorizationRequest = &v1pb.AuthorizationRequest{
			Endpoint: v.AuthUrl,
			ClientId: v.ClientId,
			Scopes:   v.Scopes,
		}
	} else if v := identityProvider.Config.GetOidcConfig(); v != nil {
		provider.AuthorizationRequest = &v1pb.AuthorizationRequest{
			ClientId: v.ClientId,
			Scopes:   v.Scopes,
		}
		// The authorization endpoint is not stored; it comes from the issuer's
		// discovery document, which the plugin caches.
		openidConfiguration, err := oidc.GetOpenIDConfiguration(v.Issuer, v.SkipTlsVerify)
		if err != nil {
			slog.Warn("failed to fetch openid configuration", slog.String("issuer", v.Issuer), log.BBError(err))
		} else {
			provider.AuthorizationRequest.Endpoint = openidConfiguration.AuthorizationEndpoint
		}
	}
	return provider
}

// Login is the auth login method including SSO.
func (s *AuthService) Login(ctx context.Context, req *connect.Request[v1pb.LoginRequest]) (*connect.Response[v1pb.LoginResponse], error) {
	request := req.Msg
	mfaSecondLogin := request.GetMfaTempToken() != ""
	// Resolve the audit workspace before authentication so failures for unknown
	// accounts can be recorded. SaaS requires an explicit workspace; self-hosted
	// falls back to the singleton workspace.
	if _, err := s.resolveAuthenticationWorkspaceID(ctx, request.Workspace); err != nil {
		return nil, err
	}

	// 1. Authenticate user (password, IDP, or MFA completion)
	loginUser, loginMethod, err := s.authenticateLogin(ctx, request)
	if err != nil {
		return nil, err
	}

	// 2. Reject deactivated users before any workspace provisioning.
	if loginUser.MemberDeleted {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user has been deactivated"))
	}

	// 3. Resolve workspace early so all subsequent checks can use it.
	// Login is allow_without_credential, so workspace is NOT in the context from auth middleware.
	preferredWS, _ := parseOptionalWorkspace(request.Workspace)
	workspaceID, err := s.resolveWorkspaceForLogin(ctx, loginUser, preferredWS)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to resolve workspace"))
	}
	// If the user has no workspace (e.g. left all workspaces), provision one.
	if workspaceID == "" {
		targetWorkspaceID, isMember, err := s.resolveWorkspaceIDByEmail(ctx, loginUser.Email, preferredWS)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to resolve target workspace"))
		}
		workspaceID, err = s.provisionResolvedWorkspace(ctx, loginUser.Email, targetWorkspaceID, isMember)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to provision workspace"))
		}
	}
	common.SetAuditWorkspaceID(ctx, workspaceID)

	// 4. Post-auth checks (deleted, domain, license). The fetched restriction
	// is reused by needResetPassword below to spare a duplicate settings read.
	restriction, err := s.validateLoginPermissions(ctx, loginUser, workspaceID, request)
	if err != nil {
		return nil, err
	}

	// 5. Check if MFA challenge needed (returns early with temp token)
	if resp, err := s.checkMFARequired(ctx, loginUser, workspaceID, mfaSecondLogin, loginMethod); err != nil {
		return nil, err
	} else if resp != nil {
		return resp, nil
	}

	// 6. Generate token (workspace already resolved)
	token, err := s.generateLoginToken(ctx, loginUser, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate access token"))
	}

	// 7. Build response and finalize
	requireResetPassword := loginMethod.requiresPasswordReset() && s.needResetPassword(ctx, loginUser, workspaceID, restriction)
	return s.finalizeLogin(ctx, req.Header(), request.Web, loginUser, token, workspaceID, requireResetPassword)
}

func (s *AuthService) needResetPassword(ctx context.Context, user *store.UserMessage, workspaceID string, restriction *v1pb.Restriction) bool {
	// Reset password restriction only works for end user with email & password login.
	if user.Type != storepb.PrincipalType_END_USER {
		return false
	}

	if restriction == nil {
		var err error
		restriction, err = getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, workspaceID)
		if err != nil {
			slog.Error("failed to get workspace restriction", log.BBError(err), slog.String("workspace", workspaceID))
			return false
		}
	}

	// Don't need to reset password if password signin is not allowed.
	if restriction.DisallowPasswordSignin {
		return false
	}

	if user.Profile.LastLoginTime == nil {
		if !restriction.PasswordRestriction.GetRequireResetPasswordForFirstLogin() {
			return false
		}
		iamPolicy, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
		if err != nil {
			slog.Error("failed to get workspace IAM policy", log.BBError(err), slog.String("workspace", workspaceID))
			return false
		}
		count, err := countUsersInIamPolicy(ctx, s.store, workspaceID, iamPolicy.Policy, s.profile.SaaS)
		if err != nil {
			slog.Error("failed to count users in workspace IAM policy", log.BBError(err), slog.String("workspace", workspaceID))
			return false
		}
		// The 1st workspace admin login don't need to reset the password
		return count > 1
	}

	if restriction.PasswordRestriction.GetPasswordRotation() != nil {
		lastChangePasswordTime := user.CreatedAt
		if user.Profile.LastChangePasswordTime != nil {
			lastChangePasswordTime = user.Profile.LastChangePasswordTime.AsTime()
		}
		if lastChangePasswordTime.Add(restriction.PasswordRestriction.GetPasswordRotation().AsDuration()).Before(time.Now()) {
			return true
		}
	}

	return false
}

// Signup registers a new user account (self-service).
// Creates a principal and assigns a workspace:
// - If the user's email was pre-invited to a workspace, joins that workspace.
// - Otherwise, creates a new workspace with the user as admin.
func (s *AuthService) Signup(ctx context.Context, req *connect.Request[v1pb.SignupRequest]) (*connect.Response[v1pb.LoginResponse], error) {
	request := req.Msg
	email := normalizeEmail(request.Email)
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email must be set"))
	}
	if request.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("title must be set"))
	}
	if request.Password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must be set"))
	}
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}

	// Whether signup is allowed at all never depends on the address, so decide it
	// without reading anything the address selects. Otherwise a registered
	// account and a stranger reach the same denial by different amounts of work
	// and are told apart by latency: SaaS refuses every signup outright, and
	// self-hosted takes its restriction from the singleton workspace, while
	// resolving by email costs one query for a member and two for a stranger.
	if s.profile.SaaS {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("sign up is disallowed for this workspace"))
	}
	workspaceID, err := s.store.GetWorkspaceID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to resolve workspace"))
	}
	// Announce it on every exit path so denied signups still produce audit entries.
	common.SetAuditWorkspaceID(ctx, workspaceID)

	restriction, err := getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, workspaceID)
	if err != nil {
		return nil, err
	}
	if restriction.DisallowSignup {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("sign up is disallowed for this workspace"))
	}

	// Past the gates the address matters: resolve where it would land, read-only,
	// so a rejected signup leaves no orphan user or workspace behind.
	targetWorkspaceID, targetIsMember, err := s.resolveWorkspaceIDByEmail(ctx, email, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to resolve target workspace"))
	}
	// Existence is checked only once the workspace would accept a signup at all.
	// Answering AlreadyExists ahead of that gate makes a workspace that refuses
	// every signup — every SaaS workspace, since the override above always sets
	// DisallowSignup — an account-existence oracle for anonymous callers.
	//
	// It stays ahead of the password policy, though. Only DisallowSignup decides
	// whether existence may be disclosed here; past it, a workspace that accepts
	// signups answers AlreadyExists to any caller who submits a valid password
	// anyway, so ordering the policy first would hide nothing and would tell a
	// registered user to pick a better password when what they need is to log in.
	existingUser, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user by email"))
	}
	if existingUser != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("email %s is already registered", request.Email))
	}

	if err := validatePasswordWithRestriction(request.Password, convertToStorePasswordRestriction(restriction.PasswordRestriction)); err != nil {
		return nil, err
	}

	workspaceID, err = s.provisionResolvedWorkspace(ctx, email, targetWorkspaceID, targetIsMember)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to provision workspace"))
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate password hash"))
	}

	// Step 2: Create the principal (global identity).
	user, err := s.store.CreateUser(ctx, &store.UserMessage{
		Email:        email,
		Name:         request.Title,
		PasswordHash: string(passwordHash),
		Profile:      &storepb.UserProfile{},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create user"))
	}

	// Step 3: Generate token and finalize login. Signup is always web-based —
	// finalizeLogin sets the tokens as HTTP-only cookies.
	token, err := s.generateLoginToken(ctx, user, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate access token"))
	}
	return s.finalizeLogin(ctx, req.Header(), true, user, token, workspaceID, false)
}

// resolveWorkspaceIDByEmail determines which workspace a signing-up email would
// land in WITHOUT mutating anything. Used by signup/signup-via-code to look up the
// applicable workspace restriction before creating a user or provisioning workspaces, so
// a rejected signup doesn't leave orphan state behind. Returns empty for SaaS brand-new
// signup (no pre-invite, no workspace) — the caller should apply default restriction.
// resolveWorkspaceIDByEmail returns (workspaceID, isMember).
// isMember is true when the user already has an IAM binding in the returned workspace.
// When false, the returned workspaceID is the self-host singleton (user needs to be added).
// A non-empty preferredWorkspaceID wins when it is one of the email's own
// memberships — mirroring resolveWorkspaceForLogin for existing users — so a
// multi-invited email lands in the workspace the login flow named rather than
// the oldest invitation.
func (s *AuthService) resolveWorkspaceIDByEmail(ctx context.Context, email, preferredWorkspaceID string) (string, bool, error) {
	if preferredWorkspaceID != "" {
		preferredWS, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
			WorkspaceID:    &preferredWorkspaceID,
			Email:          email,
			IncludeAllUser: !s.profile.SaaS,
		})
		if err != nil {
			return "", false, errors.Wrapf(err, "failed to find workspace")
		}
		if preferredWS != nil {
			return preferredWS.ResourceID, true, nil
		}
	}
	existingWS, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		Email:          email,
		IncludeAllUser: !s.profile.SaaS,
	})
	if err != nil {
		return "", false, errors.Wrapf(err, "failed to find workspaces")
	}
	if existingWS != nil {
		return existingWS.ResourceID, true, nil
	}
	if !s.profile.SaaS {
		singletonID, err := s.store.GetWorkspaceID(ctx)
		if err != nil {
			return "", false, errors.Wrapf(err, "failed to resolve singleton workspace")
		}
		return singletonID, false, nil
	}
	return "", false, nil
}

// provisionResolvedWorkspace assigns a workspace for a user who has none: the
// pre-invited workspace (or self-hosted singleton) resolved by the caller via
// resolveWorkspaceIDByEmail — callers that gate-check first reuse that same
// snapshot here — or a freshly created workspace with the user as admin.
// isMember indicates whether the user already has an IAM binding; for
// pre-invited users we must NOT patch IAM — PatchWorkspaceIamPolicy is a
// set-replacement that would downgrade an admin to member.

func (s *AuthService) provisionResolvedWorkspace(ctx context.Context, email, workspaceID string, isMember bool) (string, error) {
	if workspaceID != "" {
		if !s.profile.SaaS && !isMember {
			// Self-hosted, new user joining the singleton workspace — add as member.
			if _, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
				Workspace: workspaceID,
				Member:    common.FormatUserEmail(email),
				Roles:     []string{common.FormatRole(store.WorkspaceMemberRole)},
			}); err != nil {
				return "", errors.Wrapf(err, "failed to add user to workspace")
			}
		}
		return workspaceID, nil
	}

	// No existing workspace — create a new one with the user as admin.
	wsID, err := common.RandomString(16)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate workspace ID")
	}
	ws, err := s.store.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID:         wsID,
		Payload:            &storepb.WorkspacePayload{Title: "Default workspace"},
		AdditionalSettings: s.getAdditionalWorkspaceSettings(),
	}, email)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create workspace")
	}

	return ws.ResourceID, nil
}

// Logout is the auth logout method.
func (s *AuthService) Logout(ctx context.Context, req *connect.Request[v1pb.LogoutRequest]) (*connect.Response[emptypb.Empty], error) {
	resp := connect.NewResponse(&emptypb.Empty{})
	s.clearSessionAndSetCookies(ctx, req.Header(), resp.Header(), common.GetWorkspaceIDFromContext(ctx))
	return resp, nil
}

// clearSessionAndSetCookies deletes the refresh token and sets expired cookies on the response headers.
func (s *AuthService) clearSessionAndSetCookies(ctx context.Context, reqHeaders http.Header, respHeaders http.Header, workspaceID string) {
	if refreshToken := auth.GetRefreshTokenFromCookie(reqHeaders); refreshToken != "" {
		if err := s.store.DeleteWebRefreshToken(ctx, auth.HashToken(refreshToken)); err != nil {
			slog.Error("failed to delete refresh token", log.BBError(err))
		}
	}
	origin := reqHeaders.Get("Origin")
	respHeaders.Add(setCookieHeader, auth.GetTokenCookie(ctx, s.store, s.licenseService, workspaceID, origin, "").String())
	respHeaders.Add(setCookieHeader, auth.GetRefreshTokenCookie(origin, "", 0).String())
}

// Refresh exchanges a refresh token for new access and refresh tokens.
func (s *AuthService) Refresh(ctx context.Context, req *connect.Request[v1pb.RefreshRequest]) (*connect.Response[v1pb.RefreshResponse], error) {
	// 1. Extract refresh token from cookie
	refreshToken := auth.GetRefreshTokenFromCookie(req.Header())
	if refreshToken == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("refresh token not found"))
	}

	// 2. Look up and delete atomically
	tokenHash := auth.HashToken(refreshToken)
	stored, err := s.store.GetAndDeleteWebRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get refresh token"))
	}
	if stored == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid refresh token"))
	}

	// 3. Check expiration
	if time.Now().After(stored.ExpiresAt) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("refresh token expired"))
	}

	// 4. Get user
	user, err := s.store.GetUserByEmail(ctx, stored.UserEmail)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get user"))
	}
	if user == nil || user.MemberDeleted {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found"))
	}

	// 5. Extract workspace from the access token cookie (still present because cookie
	// outlives the JWT by 30 seconds). This ensures per-session workspace isolation.
	// Also verify the token's subject matches the refresh token's user to prevent
	// pairing a refresh token with an access token from a different session.
	accessTokenStr, err := auth.GetTokenFromHeaders(req.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Wrap(err, "invalid access token header"))
	}
	if accessTokenStr == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("access token cookie required for refresh"))
	}
	tokenClaims, err := auth.ExtractClaimsFromExpiredToken(accessTokenStr, s.secret)
	if err != nil || tokenClaims.WorkspaceID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("failed to extract workspace from access token"))
	}
	if tokenClaims.Subject != stored.UserEmail {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("access token does not match refresh token"))
	}
	workspaceID := tokenClaims.WorkspaceID

	// Verify the user is still a member of the workspace.
	ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		WorkspaceID:    &workspaceID,
		Email:          user.Email,
		IncludeAllUser: !s.profile.SaaS,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to verify workspace membership"))
	}
	if ws == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("user %q is no longer a member of workspace %q", user.Email, workspaceID))
	}

	accessTokenDuration := auth.GetAccessTokenDuration(ctx, s.store, s.licenseService, workspaceID)
	accessToken, err := auth.GenerateAccessToken(user.Email, workspaceID, s.secret, accessTokenDuration)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to generate access token"))
	}

	// 6. Rotate the session: the new refresh token inherits the original
	// expiration (absolute session lifetime).
	resp := connect.NewResponse(&v1pb.RefreshResponse{})
	if err := s.issueSessionCookies(ctx, resp.Header(), req.Header().Get("Origin"), user.Email, workspaceID, accessToken, stored.ExpiresAt); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *AuthService) getAndVerifyUser(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	email := normalizeEmail(request.Email)
	// An invalid-syntax email can never be an account; reject it before the
	// claim with the same error a wrong password gets.
	if !common.IsValidEmail(email) {
		return nil, invalidCredentialsError
	}
	// Claim an attempt slot before the credential is checked; unknown emails
	// lock at the same attempt with the same error (no existence oracle).
	// A successful login holds its slot only until the clear below, so more
	// than loginAttemptMax concurrent correct logins for one identity can see
	// transient refusals — accepted; they end when the first clear lands.
	if err := s.claimLoginAttempt(ctx, email, storepb.LoginAttemptKind_PASSWORD); err != nil {
		return nil, err
	}

	// GetAccountByEmail is cross-workspace, which is correct for login.
	// Email is globally unique (PK). The token gets workspace from account.Workspace (SA/WI)
	// or from the default workspace (END_USER).
	account, err := s.store.GetAccountByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user by email %q", email))
	}
	if account == nil {
		return nil, invalidCredentialsError
	}

	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(request.Password)); err != nil {
		// If the two passwords don't match, return a 401 status.
		return nil, invalidCredentialsError
	}

	// Convert AccountMessage to UserMessage for downstream use.
	user, err := s.store.ResolvePrincipalAsUser(ctx, account)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to resolve principal %q", account.Email))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("user %q not found", account.Email))
	}
	s.clearLoginAttempt(ctx, email, storepb.LoginAttemptKind_PASSWORD)
	return user, nil
}

func challengeMFACode(user *store.UserMessage, mfaCode string) error {
	if !totp.Validate(mfaCode, user.MFAConfig.OtpSecret) {
		return connect.NewError(connect.CodeUnauthenticated, errors.New(errMsgInvalidMFACode))
	}
	return nil
}

func (s *AuthService) challengeRecoveryCode(ctx context.Context, user *store.UserMessage, recoveryCode string) error {
	for i, code := range user.MFAConfig.RecoveryCodes {
		if subtle.ConstantTimeCompare([]byte(code), []byte(recoveryCode)) == 1 {
			// If the recovery code is valid, delete it from the user's recovery code list.
			user.MFAConfig.RecoveryCodes = slices.Delete(user.MFAConfig.RecoveryCodes, i, i+1)
			_, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
				MFAConfig: &storepb.MFAConfig{
					OtpSecret:     user.MFAConfig.OtpSecret,
					RecoveryCodes: user.MFAConfig.RecoveryCodes,
				},
			})
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
			}
			return nil
		}
	}
	return connect.NewError(connect.CodeUnauthenticated, errors.New(errMsgInvalidRecoveryCode))
}

// authenticateLogin handles all authentication paths: password, IDP, or MFA completion.
func (s *AuthService) authenticateLogin(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, loginAuthMethod, error) {
	mfaSecondLogin := request.GetMfaTempToken() != ""

	if mfaSecondLogin {
		return s.completeMFALogin(ctx, request)
	}

	loginMethod := loginAuthMethodFromRequest(request)
	if loginMethod == loginAuthMethodIDP {
		user, err := s.getOrCreateUserWithIDP(ctx, request)
		return user, loginAuthMethodIDP, err
	}

	if loginMethod == loginAuthMethodEmailCode {
		user, err := s.authenticateEmailCodeLogin(ctx, request)
		return user, loginAuthMethodEmailCode, err
	}

	user, err := s.getAndVerifyUser(ctx, request)
	return user, loginAuthMethodPassword, err
}

// completeMFALogin validates MFA temp token and verifies OTP or recovery code.
func (s *AuthService) completeMFALogin(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, loginAuthMethod, error) {
	userEmail, loginMethod, err := loginAuthMethodFromMFATempToken(*request.MfaTempToken, s.secret)
	if err != nil {
		return nil, loginAuthMethodPassword, err
	}
	// A request carrying no code at all is not a guess: nothing is compared,
	// so nothing may consume an attempt slot.
	if request.OtpCode == nil && request.RecoveryCode == nil {
		return nil, loginAuthMethodPassword, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("OTP or recovery code is required for MFA"))
	}
	// Claim an MFA slot for the email inside the signed temp token: slots are
	// per identity, so a fresh temp token does not buy fresh guesses.
	if err := s.claimLoginAttempt(ctx, userEmail, storepb.LoginAttemptKind_MFA); err != nil {
		return nil, loginAuthMethodPassword, err
	}
	user, err := s.store.GetUserByEmail(ctx, userEmail)
	if err != nil {
		return nil, loginAuthMethodPassword, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user"))
	}
	if user == nil {
		return nil, loginAuthMethodPassword, invalidCredentialsError
	}

	if err := s.challengeMFAAndClear(ctx, user, userEmail, request.OtpCode, request.RecoveryCode); err != nil {
		return nil, loginAuthMethodPassword, err
	}
	return user, loginMethod, nil
}

// validateLoginPermissions checks if the user is allowed to login. It returns
// the account restriction it fetched (nil when the checks were skipped) so the
// caller can reuse it.
func (s *AuthService) validateLoginPermissions(ctx context.Context, user *store.UserMessage, workspaceID string, request *v1pb.LoginRequest) (*v1pb.Restriction, error) {
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("user has been deactivated by administrators"))
	}

	// Login restrictions only apply to end users.
	if user.Type != storepb.PrincipalType_END_USER {
		return nil, nil
	}

	// Skip restrictions for MFA second login (already validated in first step)
	if request.GetMfaTempToken() != "" {
		return nil, nil
	}

	restriction, err := getAccountRestriction(
		ctx,
		s.store,
		s.licenseService,
		s.profile.SaaS,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	if request.GetIdpName() == "" {
		if request.Password != "" && restriction.DisallowPasswordSignin {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("password signin is disallowed"))
		}
		if request.EmailCode != nil && *request.EmailCode != "" && !restriction.AllowEmailCodeSignin {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("email code login is not enabled for this workspace"))
		}
	}

	// Check domain restriction
	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, workspaceID, user.Email, false); err != nil {
		return nil, err
	}
	return restriction, nil
}

// checkMFARequired checks if MFA is required and returns a response with temp token if so.
// Returns (nil, nil) if MFA is not required or already completed.
func (s *AuthService) checkMFARequired(ctx context.Context, user *store.UserMessage, workspaceID string, mfaSecondLogin bool, loginMethod loginAuthMethod) (*connect.Response[v1pb.LoginResponse], error) {
	if mfaSecondLogin {
		return nil, nil
	}

	userMFAEnabled := user.MFAConfig != nil && user.MFAConfig.OtpSecret != ""
	mfaFeatureEnabled := s.licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_TWO_FA) == nil
	if !mfaFeatureEnabled || !userMFAEnabled {
		return nil, nil
	}

	mfaTempToken, err := auth.GenerateMFATempTokenWithLoginMethod(user.Email, string(loginMethod), s.secret, mfaTempTokenDuration)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate MFA temp token"))
	}

	return connect.NewResponse(&v1pb.LoginResponse{
		MfaTempToken: &mfaTempToken,
	}), nil
}

// generateLoginToken generates the appropriate token based on user type.
func (s *AuthService) generateLoginToken(ctx context.Context, user *store.UserMessage, workspaceID string) (string, error) {
	tokenDuration := auth.GetAccessTokenDuration(ctx, s.store, s.licenseService, workspaceID)

	var token string
	var err error
	switch user.Type {
	case storepb.PrincipalType_END_USER:
		token, err = auth.GenerateAccessToken(user.Email, workspaceID, s.secret, tokenDuration)
	case storepb.PrincipalType_SERVICE_ACCOUNT:
		token, err = auth.GenerateAPIToken(user.Email, workspaceID, s.secret)
	default:
		return "", connect.NewError(connect.CodeUnauthenticated, errors.Errorf("user type %s cannot login", user.Type))
	}
	if err != nil {
		return "", err
	}
	return token, nil
}

// resolveWorkspaceForLogin determines the workspace for a login token.
// For SA/WI: looks up the account record to get workspace.
// For END_USER: resolution order:
//  1. preferredWorkspaceID (from the login request's ?workspace= hint, e.g. invite links)
//  2. Last login workspace (from user profile)
//  3. First workspace from IAM membership
//
// Each candidate is validated for membership before use.
func (s *AuthService) resolveWorkspaceForLogin(ctx context.Context, user *store.UserMessage, preferredWorkspaceID string) (string, error) {
	// Determine member name format based on user type.
	switch user.Type {
	case storepb.PrincipalType_SERVICE_ACCOUNT:
		// SA has workspace on its record — look it up directly.
		sa, err := s.store.GetServiceAccountByEmail(ctx, user.Email)
		if err != nil {
			return "", errors.Wrap(err, "failed to get service account")
		}
		if sa != nil {
			return sa.Workspace, nil
		}
		return "", errors.Errorf("service account %q not found", user.Email)
	case storepb.PrincipalType_END_USER:
		includeAllUser := !s.profile.SaaS

		// Prefer the workspace from the login request hint (e.g. invite link).
		if preferredWorkspaceID != "" {
			ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
				WorkspaceID:    &preferredWorkspaceID,
				Email:          user.Email,
				IncludeAllUser: includeAllUser,
			})
			if err != nil {
				return "", errors.Wrap(err, "failed to find workspace")
			}
			if ws != nil {
				return ws.ResourceID, nil
			}
			// Not a member of preferred workspace — fall through.
		}

		// Prefer the last login workspace if it's still valid.
		if lastWS := user.Profile.GetLastLoginWorkspace(); lastWS != "" {
			ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
				WorkspaceID:    &lastWS,
				Email:          user.Email,
				IncludeAllUser: includeAllUser,
			})
			if err != nil {
				return "", errors.Wrap(err, "failed to find workspace")
			}
			if ws != nil {
				return ws.ResourceID, nil
			}
			// Last login workspace no longer valid — fall through to default.
		}

		// Use the first workspace the user is a member of.
		ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
			Email:          user.Email,
			IncludeAllUser: includeAllUser,
		})
		if err != nil {
			return "", errors.Wrap(err, "failed to find workspace")
		}
		if ws == nil {
			return "", nil
		}
		return ws.ResourceID, nil
	default:
		return "", errors.Errorf("unsupported user type %s for login", user.Type)
	}
}

// SwitchWorkspace switches the current user's active workspace and issues new tokens.
func (s *AuthService) SwitchWorkspace(ctx context.Context, req *connect.Request[v1pb.SwitchWorkspaceRequest]) (*connect.Response[v1pb.LoginResponse], error) {
	request := req.Msg
	if request.Workspace == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace is required"))
	}

	workspaceID, err := common.GetWorkspaceID(request.Workspace)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid workspace name"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found"))
	}
	if user.Type != storepb.PrincipalType_END_USER {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only end users can switch workspaces"))
	}

	if err := s.rejectMCPOriginatedTokenMint(ctx, req.Header(), "switch workspaces"); err != nil {
		return nil, err
	}

	// Verify the user is a member of the target workspace.
	ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		WorkspaceID:    &workspaceID,
		Email:          user.Email,
		IncludeAllUser: !s.profile.SaaS,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to find workspace"))
	}
	if ws == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("not a member of workspace %q", workspaceID))
	}

	// Validate the target workspace's sign-in policies.
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user has been deactivated"))
	}
	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, workspaceID, user.Email, false); err != nil {
		return nil, err
	}

	// Check MFA requirement for the target workspace.
	mfaSecondStep := request.GetMfaTempToken() != ""
	if mfaSecondStep {
		// Verify the MFA temp token, then claim from the same per-identity
		// bucket the login MFA step draws from.
		mfaEmail, err := auth.GetUserEmailFromMFATempToken(*request.MfaTempToken, s.secret)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid MFA temp token"))
		}
		if mfaEmail != user.Email {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("MFA token does not match user"))
		}
		// An argument error is not a guess — refuse it before the claim.
		if request.OtpCode == nil && request.RecoveryCode == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("OTP or recovery code required"))
		}
		if err := s.claimLoginAttempt(ctx, mfaEmail, storepb.LoginAttemptKind_MFA); err != nil {
			return nil, err
		}
		if err := s.challengeMFAAndClear(ctx, user, mfaEmail, request.OtpCode, request.RecoveryCode); err != nil {
			return nil, err
		}
	} else {
		// First step: check if MFA is required for the target workspace.
		if resp, err := s.checkMFARequired(ctx, user, workspaceID, false, loginAuthMethodPassword); err != nil {
			return nil, err
		} else if resp != nil {
			return resp, nil
		}
	}

	return s.switchWorkspaceInternal(ctx, user, workspaceID, request.Web, req.Header())
}

// rejectMCPOriginatedTokenMint refuses a request that originated from an MCP
// session. Every path into switchWorkspaceInternal hands the
// caller a plain bb.user.access token in the response body when there is no
// refresh cookie — and an MCP session never has one. That token is not
// audience-bound to the MCP resource, survives revocation of the OAuth grant,
// and ignores the workspace MCP kill switch, so an MCP session must not be
// able to obtain one, for its own workspace or any other. An MCP session is
// also bound to a specific workspace via the grant.
//
// It recognizes MCP origin two independent ways. The AuthContext is the
// structural one: the internal interceptor stamps the delegated grant on every
// request arriving on the internal MCP transport, and presence alone marks the
// request (see common.DelegatedGrant), so this holds even for a caller that
// forwards no headers. The bearer check then covers the public chain, where
// the AuthContext carries no grant.
//
// The bearer half asks auth rather than reading an audience itself. Audience
// stopped being the recognizer: the delegated credential is signed with a
// derived key, so its claims do not verify under the raw secret at all
// (TestSwitchWorkspaceMCPRecognition pins that), and since P1a PR 3 an
// external MCP token's audience is a per-deployment resource URI that cannot
// be matched by value, leaving token_use to identify it. auth.IsMCPOriginatedToken
// holds both recognizers.
//
// Callers must run this BEFORE any mutation — leaving or deleting a workspace
// and only then refusing the token would strand the caller.
func (s *AuthService) rejectMCPOriginatedTokenMint(ctx context.Context, reqHeaders http.Header, action string) error {
	authCtx, ok := common.GetAuthContextFromContext(ctx)
	mcpOriginated := ok && authCtx.DelegatedGrant != nil
	if !mcpOriginated {
		accessTokenStr, _ := auth.GetTokenFromHeaders(reqHeaders)
		mcpOriginated = auth.IsMCPOriginatedToken(accessTokenStr, s.secret)
	}
	if mcpOriginated {
		return connect.NewError(connect.CodePermissionDenied, errors.Errorf("MCP sessions cannot %s", action))
	}
	return nil
}

// switchWorkspaceInternal generates new tokens for the target workspace and
// returns a LoginResponse with cookies set. Used by SwitchWorkspace,
// LeaveWorkspace, and DeleteWorkspace.
func (s *AuthService) switchWorkspaceInternal(ctx context.Context, user *store.UserMessage, workspaceID string, web bool, reqHeaders http.Header) (*connect.Response[v1pb.LoginResponse], error) {
	// Each caller already refuses MCP-originated requests up front, where the
	// refusal still precedes that caller's mutations. Repeating it at the mint
	// costs nothing and makes the boundary structural: a future caller that
	// forgets cannot reopen the escape. This function acquired two unguarded
	// callers once already.
	if err := s.rejectMCPOriginatedTokenMint(ctx, reqHeaders, "obtain a workspace token"); err != nil {
		return nil, err
	}

	token, err := s.generateLoginToken(ctx, user, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to generate token"))
	}

	// Update last login workspace.
	if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
		Profile: &storepb.UserProfile{
			LastLoginTime:          user.Profile.GetLastLoginTime(),
			LastChangePasswordTime: user.Profile.GetLastChangePasswordTime(),
			Source:                 user.Profile.GetSource(),
			LastLoginWorkspace:     workspaceID,
		},
	}); err != nil {
		slog.Error("failed to update user profile", log.BBError(err))
	}

	response := &v1pb.LoginResponse{}
	resp := connect.NewResponse(response)

	if web {
		oldRefreshToken := auth.GetRefreshTokenFromCookie(reqHeaders)
		if oldRefreshToken == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("refresh token cookie required for web workspace switch"))
		}
		oldStored, err := s.store.GetAndDeleteWebRefreshToken(ctx, auth.HashToken(oldRefreshToken))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to consume refresh token"))
		}
		if oldStored == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired refresh token"))
		}
		if oldStored.UserEmail != user.Email {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("refresh token does not belong to current user"))
		}
		sessionExpiresAt := oldStored.ExpiresAt
		if sessionExpiresAt.IsZero() {
			sessionExpiresAt = time.Now().Add(auth.GetRefreshTokenDuration(ctx, s.store, s.licenseService, workspaceID))
		}

		if err := s.issueSessionCookies(ctx, resp.Header(), reqHeaders.Get("Origin"), user.Email, workspaceID, token, sessionExpiresAt); err != nil {
			return nil, err
		}
	} else {
		response.Token = token
	}

	v1User, err := convertToUser(ctx, s.iamManager, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to convert user"))
	}
	v1User.Workspace = common.FormatWorkspace(workspaceID)
	response.User = v1User

	return resp, nil
}

// finalizeLogin builds the response, sets cookies if needed, and updates the
// user profile. Shared by Login and Signup.
func (s *AuthService) finalizeLogin(ctx context.Context, header http.Header, web bool, user *store.UserMessage, token string, workspaceID string, requireResetPassword bool) (*connect.Response[v1pb.LoginResponse], error) {
	response := &v1pb.LoginResponse{
		RequireResetPassword: requireResetPassword,
	}
	resp := connect.NewResponse(response)

	if web {
		if user.Type != storepb.PrincipalType_END_USER {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("only users can use web login"))
		}
		// A fresh session: the refresh token gets the full refresh duration.
		d := auth.GetRefreshTokenDuration(ctx, s.store, s.licenseService, workspaceID)
		if err := s.issueSessionCookies(ctx, resp.Header(), header.Get("Origin"), user.Email, workspaceID, token, time.Now().Add(d)); err != nil {
			return nil, err
		}
	} else {
		// For non-web clients (CLI, API), return the token in the response body.
		response.Token = token
	}

	if user.Type == storepb.PrincipalType_END_USER {
		if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
			Profile: &storepb.UserProfile{
				LastLoginTime:          timestamppb.Now(),
				LastChangePasswordTime: user.Profile.GetLastChangePasswordTime(),
				Source:                 user.Profile.GetSource(),
				LastLoginWorkspace:     workspaceID,
			},
		}); err != nil {
			slog.Error("failed to update user profile", log.BBError(err), slog.String("user", user.Email))
		}

		v1User, err := convertToUser(ctx, s.iamManager, user)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
		}
		v1User.Workspace = common.FormatWorkspace(workspaceID)
		response.User = v1User
	}

	return resp, nil
}

// issueSessionCookies mints one web session: it stores a refresh token
// expiring at refreshExpiresAt and sets both HTTP-only cookies on respHeader.
// The refresh cookie lives until that same expiry (its max age truncates to
// whole seconds).
func (s *AuthService) issueSessionCookies(ctx context.Context, respHeader http.Header, origin, userEmail, workspaceID, accessToken string, refreshExpiresAt time.Time) error {
	respHeader.Add(setCookieHeader, auth.GetTokenCookie(ctx, s.store, s.licenseService, workspaceID, origin, accessToken).String())
	refreshToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to generate refresh token"))
	}
	if err := s.store.CreateWebRefreshToken(ctx, &store.WebRefreshTokenMessage{
		TokenHash: auth.HashToken(refreshToken),
		UserEmail: userEmail,
		ExpiresAt: refreshExpiresAt,
	}); err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create refresh token"))
	}
	respHeader.Add(setCookieHeader, auth.GetRefreshTokenCookie(origin, refreshToken, time.Until(refreshExpiresAt)).String())
	return nil
}

// ExchangeToken exchanges an external OIDC token for a Bytebase access token.
// Used by CI/CD pipelines with Workload Identity Federation.
func (s *AuthService) ExchangeToken(ctx context.Context, req *connect.Request[v1pb.ExchangeTokenRequest]) (*connect.Response[v1pb.ExchangeTokenResponse], error) {
	request := req.Msg

	if request.Token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("token is required"))
	}
	if request.Email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email is required"))
	}

	if err := validateWorkloadIdentityEmail(request.Email); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			invalidAccountEmailError("workload identity", request.Email, err))
	}

	// Find workload identity by email (cross-workspace lookup since this is unauthenticated).
	wi, err := s.store.GetWorkloadIdentityByEmail(ctx, request.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to find workload identity"))
	}
	if wi == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q not found", request.Email))
	}
	// Announce the workspace as soon as we know it (from the WI record) so
	// that a deactivated-WI attempt — which compliance wants to see — still
	// lands in the audit log.
	common.SetAuditWorkspaceID(ctx, wi.Workspace)
	if wi.MemberDeleted {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			errors.New("workload identity has been deactivated"))
	}

	// Get workload identity config
	wicConfig := wi.Config
	if wicConfig == nil {
		return nil, connect.NewError(connect.CodeInternal,
			errors.New("workload identity config not found"))
	}

	// Validate OIDC token
	if _, err = wif.ValidateToken(ctx, request.Token, wicConfig); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			errors.Wrap(err, "token validation failed"))
	}

	// Generate Bytebase API token using workspace from the WI record.
	token, err := auth.GenerateAPIToken(wi.Email, wi.Workspace, s.secret)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			errors.Wrap(err, "failed to generate access token"))
	}

	return connect.NewResponse(&v1pb.ExchangeTokenResponse{
		AccessToken: token,
	}), nil
}

func getAccountRestriction(
	ctx context.Context,
	stores *store.Store,
	licenseService *enterprise.LicenseService,
	saas bool,
	workspaceID string,
) (*v1pb.Restriction, error) {
	defaultPasswordRestriction := &v1pb.WorkspaceProfileSetting_PasswordRestriction{
		MinLength: 8,
	}
	restriction := &v1pb.Restriction{
		DisallowSignup:         false,
		DisallowPasswordSignin: false,
		AllowEmailCodeSignin:   false,
		PasswordResetEnabled:   false,
		PasswordRestriction:    defaultPasswordRestriction,
	}

	emailSetting, err := resolvePreLoginEmailSetting(ctx, stores, workspaceID)
	if err != nil {
		return nil, err
	}

	if workspaceID != "" {
		setting, err := stores.GetWorkspaceProfileSetting(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find profile setting for workspace %v", workspaceID))
		}

		restriction = &v1pb.Restriction{
			PasswordRestriction:    convertToV1PasswordRestriction(setting.GetPasswordRestriction()),
			DisallowSignup:         setting.DisallowSignup,
			DisallowPasswordSignin: setting.DisallowPasswordSignin,
			AllowEmailCodeSignin:   setting.AllowEmailCodeSignin,
		}

		// Override if features are not enabled
		if licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_DISALLOW_SELF_SERVICE_SIGNUP) != nil {
			restriction.DisallowSignup = false
		}
		if licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_DISALLOW_PASSWORD_SIGNIN) != nil {
			restriction.DisallowPasswordSignin = false
		}
		if licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_PASSWORD_RESTRICTIONS) != nil {
			restriction.PasswordRestriction = defaultPasswordRestriction
		}
	}

	// Override for SaaS
	if saas {
		restriction.DisallowSignup = true
		restriction.DisallowPasswordSignin = true
		restriction.AllowEmailCodeSignin = true
	}

	if !restriction.DisallowPasswordSignin {
		restriction.PasswordResetEnabled = emailSetting != nil
	}
	if emailSetting == nil {
		restriction.AllowEmailCodeSignin = false
	}

	return restriction, nil
}

// parseOptionalWorkspace extracts the workspace ID from an optional "workspaces/{id}"
// resource name. Returns empty when the caller has no workspace context yet
// (SaaS brand-new signup flow).
func parseOptionalWorkspace(name *string) (string, error) {
	if name == nil || *name == "" {
		return "", nil
	}
	return common.GetWorkspaceID(*name)
}

// getAdditionalWorkspaceSettings returns extra settings to inject during workspace creation.
// In SaaS mode with Gemini API key configured, injects AI settings.
func (*AuthService) getAdditionalWorkspaceSettings() []store.AdditionalSetting {
	var settings []store.AdditionalSetting
	if geminiAPIKey := os.Getenv("GEMINI_API_KEY"); geminiAPIKey != "" {
		settings = append(settings, store.AdditionalSetting{
			Name: storepb.SettingName_AI,
			Payload: &storepb.AISetting{
				Enabled:  true,
				Provider: storepb.AISetting_GEMINI,
				ApiKey:   geminiAPIKey,
				Endpoint: "https://generativelanguage.googleapis.com/v1beta",
				Model:    "gemini-3.7-flash",
			},
		})
	}
	if raw := os.Getenv("EMAIL_CONFIG"); raw != "" { //nolint:nestif
		emailSetting := &storepb.EmailSetting{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(raw), emailSetting); err != nil {
			slog.Error("failed to parse EMAIL_CONFIG env var", log.BBError(err))
		} else if err := validateEmailSetting(emailSetting); err != nil {
			slog.Error("invalid EMAIL_CONFIG env var", log.BBError(err))
		} else {
			settings = append(settings, store.AdditionalSetting{
				Name:    storepb.SettingName_EMAIL,
				Payload: emailSetting,
			})
		}
	}
	return settings
}
