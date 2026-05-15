package v1

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
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
	"github.com/bytebase/bytebase/backend/common/qb"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/idp/ldap"
	"github.com/bytebase/bytebase/backend/plugin/idp/oauth2"
	"github.com/bytebase/bytebase/backend/plugin/idp/oidc"
	"github.com/bytebase/bytebase/backend/plugin/idp/wif"
	"github.com/bytebase/bytebase/backend/plugin/mailer"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

const (
	emailCodeLength         = 6
	emailCodeExpiry         = 10 * time.Minute
	emailCodeMaxAttempts    = 5
	emailCodeResendCooldown = 60 * time.Second

	// mfaTempTokenDuration is the duration for MFA temporary tokens.
	// Following industry standards (Okta: 5 minutes, Auth0: 10 minutes, AWS Cognito: 3 minutes).
	// A short duration reduces the attack window for TOTP brute-force attempts.
	mfaTempTokenDuration = 5 * time.Minute

	// Login rate limiting configuration.
	// Password phase: 10 failed attempts within 10 minutes.
	passwordMaxFailedAttempts = 10               // Will be used for password rate limiting
	passwordLockoutWindow     = 10 * time.Minute // Will be used for password rate limiting

	// MFA phase: 5 failed attempts within 5 minutes.
	mfaMaxFailedAttempts = 5
	mfaLockoutWindow     = 5 * time.Minute

	// Error messages for authentication failures.
	// These constants are used both for error responses and for querying audit logs during rate limiting.
	errMsgInvalidCredentials  = "invalid email or password"
	errMsgInvalidMFACode      = "invalid MFA code"
	errMsgInvalidRecoveryCode = "invalid recovery code"
	errMsgTooManyPassword     = "too many failed login attempts, please try again later" // Will be used for password rate limiting
	errMsgTooManyMFA          = "too many failed MFA attempts, please try again later"
)

var (
	invalidCredentialsError = connect.NewError(connect.CodeUnauthenticated, errors.Errorf(errMsgInvalidCredentials))
)

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

// Login is the auth login method including SSO.
func (s *AuthService) Login(ctx context.Context, req *connect.Request[v1pb.LoginRequest]) (*connect.Response[v1pb.LoginResponse], error) {
	request := req.Msg
	mfaSecondLogin := request.GetMfaTempToken() != ""

	// 1. Authenticate user (password, IDP, or MFA completion)
	loginUser, err := s.authenticateLogin(ctx, request)
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
	// If the user has no workspace (e.g. left all workspaces), provision a new one.
	if workspaceID == "" {
		workspaceID, err = s.provisionWorkspaceForNewUser(ctx, loginUser.Email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to provision workspace"))
		}
	}
	common.SetAuditWorkspaceID(ctx, workspaceID)

	// 3. Post-auth checks (deleted, domain, license)
	if err := s.validateLoginPermissions(ctx, loginUser, workspaceID, request); err != nil {
		return nil, err
	}

	// 4. Check if MFA challenge needed (returns early with temp token)
	if resp, err := s.checkMFARequired(ctx, loginUser, workspaceID, mfaSecondLogin); err != nil {
		return nil, err
	} else if resp != nil {
		return resp, nil
	}

	// 5. Generate token (workspace already resolved)
	token, err := s.generateLoginToken(ctx, loginUser, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate access token"))
	}

	// 6. Build response and finalize
	requireResetPassword := s.needResetPassword(ctx, loginUser, workspaceID)
	return s.finalizeLogin(ctx, req, loginUser, token, workspaceID, requireResetPassword)
}

func (s *AuthService) needResetPassword(ctx context.Context, user *store.UserMessage, workspaceID string) bool {
	// Reset password restriction only works for end user with email & password login.
	if user.Type != storepb.PrincipalType_END_USER {
		return false
	}

	restriction, err := getAccountRestriction(
		ctx,
		s.store,
		s.licenseService,
		s.profile.SaaS,
		workspaceID,
	)
	if err != nil {
		slog.Error("failed to get workspace restriction", log.BBError(err), slog.String("workspace", workspaceID))
		return false
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
	if request.Email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email must be set"))
	}
	if request.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("title must be set"))
	}
	if request.Password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must be set"))
	}
	if err := validateEndUserEmail(request.Email); err != nil {
		return nil, err
	}

	// Check if principal already exists.
	existingUser, err := s.store.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user by email"))
	}
	if existingUser != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("email %s is already registered", request.Email))
	}

	// Resolve the target workspace (read-only) so we can check restrictions BEFORE
	// any write — otherwise a rejected signup would leave an orphan user/workspace behind.
	targetWorkspaceID, _, err := s.resolveWorkspaceIDByEmail(ctx, request.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to resolve target workspace"))
	}
	// Announce the workspace on every exit path so denied signups (DisallowSignup,
	// password restriction) still produce audit entries. Uses targetWorkspaceID (resolved
	// before any writes) rather than the provisioned workspaceID.
	defer func() { common.SetAuditWorkspaceID(ctx, targetWorkspaceID) }()

	restriction, err := getAccountRestriction(
		ctx,
		s.store,
		s.licenseService,
		s.profile.SaaS,
		targetWorkspaceID,
	)
	if err != nil {
		return nil, err
	}
	if restriction.DisallowSignup {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("sign up is disallowed for this workspace %v", targetWorkspaceID))
	}
	if err := validatePasswordWithRestriction(request.Password, convertToStorePasswordRestriction(restriction.PasswordRestriction)); err != nil {
		return nil, err
	}

	workspaceID, err := s.provisionWorkspaceForNewUser(ctx, request.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to provision workspace"))
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate password hash"))
	}

	// Step 2: Create the principal (global identity).
	user, err := s.store.CreateUser(ctx, &store.UserMessage{
		Email:        request.Email,
		Name:         request.Title,
		PasswordHash: string(passwordHash),
		Profile:      &storepb.UserProfile{},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create user"))
	}

	// Step 3: Generate token and finalize login.
	tokenDuration := auth.GetAccessTokenDuration(ctx, s.store, s.licenseService, workspaceID)
	token, err := auth.GenerateAccessToken(user.Email, workspaceID, s.secret, tokenDuration)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate access token"))
	}

	response := &v1pb.LoginResponse{}
	resp := connect.NewResponse(response)

	// Signup is always web-based — set tokens as HTTP-only cookies.
	origin := req.Header().Get("Origin")
	cookie := auth.GetTokenCookie(ctx, s.store, s.licenseService, workspaceID, origin, token)
	resp.Header().Add("Set-Cookie", cookie.String())

	refreshToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to generate refresh token"))
	}
	refreshTokenDuration := auth.GetRefreshTokenDuration(ctx, s.store, s.licenseService, workspaceID)
	if err := s.store.CreateWebRefreshToken(ctx, &store.WebRefreshTokenMessage{
		TokenHash: auth.HashToken(refreshToken),
		UserEmail: user.Email,
		ExpiresAt: time.Now().Add(refreshTokenDuration),
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create refresh token"))
	}
	refreshCookie := auth.GetRefreshTokenCookie(origin, refreshToken, refreshTokenDuration)
	resp.Header().Add("Set-Cookie", refreshCookie.String())

	// Update last login time and workspace.
	if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
		Profile: &storepb.UserProfile{
			LastLoginTime:      timestamppb.Now(),
			Source:             user.Profile.GetSource(),
			LastLoginWorkspace: workspaceID,
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

	return resp, nil
}

// resolveWorkspaceIDByEmail determines which workspace a signing-up email would
// land in WITHOUT mutating anything. Used by signup/signup-via-code to look up the
// applicable workspace restriction before creating a user or provisioning workspaces, so
// a rejected signup doesn't leave orphan state behind. Returns empty for SaaS brand-new
// signup (no pre-invite, no workspace) — the caller should apply default restriction.
// resolveWorkspaceIDByEmail returns (workspaceID, isMember).
// isMember is true when the user already has an IAM binding in the returned workspace.
// When false, the returned workspaceID is the self-host singleton (user needs to be added).
func (s *AuthService) resolveWorkspaceIDByEmail(ctx context.Context, email string) (string, bool, error) {
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

// provisionWorkspaceForNewUser returns a workspace ID for a freshly-created user.
// If the email was pre-invited to existing workspaces (via IAM), returns the first one.
// Otherwise creates a new workspace (SaaS: per-user; self-hosted: joins the singleton).
// Called by both the Signup RPC and the email-code signup branch of Login.
func (s *AuthService) provisionWorkspaceForNewUser(ctx context.Context, email string) (string, error) {
	// Step 1: Resolve the target workspace. isMember indicates whether the user already has
	// an IAM binding. For pre-invited users we must NOT patch IAM — PatchWorkspaceIamPolicy
	// is a set-replacement that would downgrade an admin to member.
	workspaceID, isMember, err := s.resolveWorkspaceIDByEmail(ctx, email)
	if err != nil {
		return "", err
	}

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

	// Step 2: No existing workspace — create a new one with the user as admin.
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
	respHeaders.Add("Set-Cookie", auth.GetTokenCookie(ctx, s.store, s.licenseService, workspaceID, origin, "").String())
	respHeaders.Add("Set-Cookie", auth.GetRefreshTokenCookie(origin, "", 0).String())
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

	newRefreshToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to generate refresh token"))
	}

	// Inherit expiration from the original token (absolute session lifetime)
	if err := s.store.CreateWebRefreshToken(ctx, &store.WebRefreshTokenMessage{
		TokenHash: auth.HashToken(newRefreshToken),
		UserEmail: user.Email,
		ExpiresAt: stored.ExpiresAt,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create refresh token"))
	}

	// 6. Set cookies and return
	resp := connect.NewResponse(&v1pb.RefreshResponse{})
	origin := req.Header().Get("Origin")
	resp.Header().Add("Set-Cookie", auth.GetTokenCookie(ctx, s.store, s.licenseService, workspaceID, origin, accessToken).String())
	resp.Header().Add("Set-Cookie", auth.GetRefreshTokenCookie(origin, newRefreshToken, time.Until(stored.ExpiresAt)).String())

	return resp, nil
}

func (s *AuthService) getAndVerifyUser(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	// Check if user is locked out due to too many failed password attempts.
	if err := s.checkPasswordLockout(ctx, request.Email); err != nil {
		return nil, err
	}

	// GetAccountByEmail is cross-workspace, which is correct for login.
	// Email is globally unique (PK). The token gets workspace from account.Workspace (SA/WI)
	// or from the default workspace (END_USER).
	account, err := s.store.GetAccountByEmail(ctx, request.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user by email %q", request.Email))
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
	return user, nil
}

// getOrCreateUserWithIDP authenticates a user via an identity provider (SSO).
// Login API has allow_without_credential, so there's no workspace in the token context.
// We resolve workspace from the IDP entity (IDP resource_id is globally unique).
func (s *AuthService) getOrCreateUserWithIDP(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	idpID, err := common.GetIdentityProviderID(request.IdpName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get identity provider ID"))
	}
	// Look up IDP without workspace filter — IDP resource_id is globally unique.
	// The workspace is resolved from the IDP entity.
	idp, err := s.store.GetIdentityProviderByID(ctx, idpID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get identity provider"))
	}
	if idp == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("identity provider not found"))
	}

	// For workspace-scoped IDPs, use the IDP's workspace.
	// For global IDPs (SaaS), workspace is resolved after authentication from user membership.
	workspaceID := idp.Workspace
	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID)
	if err != nil {
		return nil, err
	}

	var userInfo *storepb.IdentityProviderUserInfo
	switch idp.Type {
	case storepb.IdentityProviderType_OAUTH2:
		oauth2Context := request.IdpContext.GetOauth2Context()
		if oauth2Context == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("missing OAuth2 context"))
		}
		oauth2IdentityProvider, err := oauth2.NewIdentityProvider(idp.Config.GetOauth2Config())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create new OAuth2 identity provider"))
		}
		redirectURL := fmt.Sprintf("%s/oauth/callback", externalURL)
		token, err := oauth2IdentityProvider.ExchangeToken(ctx, redirectURL, oauth2Context.Code)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to exchange token"))
		}
		userInfo, _, err = oauth2IdentityProvider.UserInfo(token)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user info"))
		}
	case storepb.IdentityProviderType_OIDC:
		oidcContext := request.IdpContext.GetOidcContext()
		if oidcContext == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("missing OIDC context"))
		}

		oidcIDP, err := oidc.NewIdentityProvider(ctx, idp.Config.GetOidcConfig())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create new OIDC identity provider"))
		}

		redirectURL := fmt.Sprintf("%s/oidc/callback", externalURL)
		token, err := oidcIDP.ExchangeToken(ctx, redirectURL, oidcContext.Code)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to exchange token"))
		}

		userInfo, _, err = oidcIDP.UserInfo(ctx, token, "")
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user info"))
		}
	case storepb.IdentityProviderType_LDAP:
		idpConfig := idp.Config.GetLdapConfig()
		ldapIDP, err := ldap.NewIdentityProvider(
			ldap.IdentityProviderConfig{
				Host:             idpConfig.Host,
				Port:             int(idpConfig.Port),
				SkipTLSVerify:    idpConfig.SkipTlsVerify,
				BindDN:           idpConfig.BindDn,
				BindPassword:     idpConfig.BindPassword,
				BaseDN:           idpConfig.BaseDn,
				UserFilter:       idpConfig.UserFilter,
				SecurityProtocol: idpConfig.SecurityProtocol,
				FieldMapping:     idpConfig.FieldMapping,
			},
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create new LDAP identity provider"))
		}

		userInfo, err = ldapIDP.Authenticate(request.Email, request.Password)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user info"))
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("identity provider type %s not supported", idp.Type.String()))
	}
	if userInfo == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("failed to get user info from identity provider %q", idp.Title))
	}
	if userInfo.Identifier == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("missing identifier in user info from identity provider %q", idp.Title))
	}
	// The userinfo's email comes from identity provider, it has to be converted to lower-case.
	email := strings.ToLower(userInfo.Identifier)
	if err := common.ValidateEmail(email); err != nil {
		// If the email is invalid, we will try to use the domain and identifier to construct the email.
		domain := extractDomain(idp.Domain)
		if domain != "" {
			email = strings.ToLower(fmt.Sprintf("%s@%s", email, domain))
		} else {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid email %q", userInfo.Identifier))
		}
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list users by email %s", email))
	}
	// User login through global SSO, then we should auth-resolve the workspace id.
	if user != nil && workspaceID == "" {
		wsID, err := s.resolveWorkspaceForLogin(ctx, user, "")
		if err != nil {
			slog.Warn("failed to resolve workspace", slog.String("user", user.Email), log.BBError(err))
		}
		workspaceID = wsID
	}
	// First time login through SSO
	if user == nil {
		if workspaceID != "" {
			if err := validateEmailWithDomains(ctx, s.licenseService, s.store, workspaceID, email, false); err != nil {
				return nil, err
			}

			// We will only block new create creation and still allow SSO login from existing users.
			featurePlan := v1pb.PlanFeature_FEATURE_ENTERPRISE_SSO
			if idp.Type == storepb.IdentityProviderType_OAUTH2 && googleGitHubDomains[idp.Domain] {
				featurePlan = v1pb.PlanFeature_FEATURE_GOOGLE_AND_GITHUB_SSO
			}
			if err := s.licenseService.IsFeatureEnabled(ctx, workspaceID, featurePlan); err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}

			if err := userCountGuard(ctx, s.store, s.licenseService, workspaceID, nil, s.profile.SaaS); err != nil {
				return nil, err
			}
		}

		// Create new user from identity provider.
		password, err := common.RandomString(20)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate random password"))
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate password hash"))
		}

		newUser, err := s.store.CreateUser(ctx, &store.UserMessage{
			Name:         userInfo.DisplayName,
			Email:        email,
			Phone:        userInfo.Phone,
			Type:         storepb.PrincipalType_END_USER,
			PasswordHash: string(passwordHash),
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create user"))
		}

		user = newUser
	}

	if workspaceID == "" {
		// Global IDP: create a new workspace for the user (same as Signup flow).
		wsID, err := common.RandomString(16)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to generate workspace ID"))
		}
		ws, err := s.store.CreateWorkspace(ctx, &store.WorkspaceMessage{
			ResourceID:         wsID,
			Payload:            &storepb.WorkspacePayload{Title: "Default Workspace"},
			AdditionalSettings: s.getAdditionalWorkspaceSettings(),
		}, email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create workspace"))
		}
		workspaceID = ws.ResourceID
	} else {
		// Workspace-scoped IDP: add user as member only if not already in the workspace.
		ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
			WorkspaceID:    &workspaceID,
			Email:          email,
			IncludeAllUser: !s.profile.SaaS,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check workspace membership"))
		}
		if ws == nil {
			if _, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
				Workspace: workspaceID,
				Member:    common.FormatUserEmail(email),
				Roles:     []string{common.FormatRole(store.WorkspaceMemberRole)},
			}); err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to add user to workspace"))
			}
		}
	}

	if user.MemberDeleted {
		if err := userCountGuard(ctx, s.store, s.licenseService, workspaceID, nil, s.profile.SaaS); err != nil {
			return nil, err
		}
		// Undelete the user when login via SSO.
		user, err = s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &undeletePatch})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to undelete user"))
		}
	}

	if userInfo.HasGroups {
		if err := s.syncUserGroups(ctx, user, workspaceID, userInfo.Groups); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to sync user groups"))
		}
	}
	return user, nil
}

// countRecentLoginFailures counts the number of failed login attempts for a given email
// within the specified time window, matching any of the provided error messages.
func (s *AuthService) countRecentLoginFailures(ctx context.Context, email string, window time.Duration, errMessages ...string) (int, error) {
	if len(errMessages) == 0 {
		return 0, errors.New("at least one error message is required")
	}

	windowStart := time.Now().Add(-window)

	// Build filter query for login failures.
	filterQ := qb.Q().Space("TRUE")
	filterQ.And("payload->>'method' = ?", "/bytebase.v1.AuthService/Login")
	filterQ.And("payload->>'resource' = ?", email)
	filterQ.And("(payload->'status'->>'code')::int != 0")

	// Build OR condition for error messages.
	if len(errMessages) == 1 {
		filterQ.And("payload->'status'->>'message' = ?", errMessages[0])
	} else {
		// For multiple messages, build: (msg = ? OR msg = ? OR ...)
		orConditions := qb.Q()
		for i, msg := range errMessages {
			if i == 0 {
				orConditions.Space("payload->'status'->>'message' = ?", msg)
			} else {
				orConditions.Or("payload->'status'->>'message' = ?", msg)
			}
		}
		filterQ.And("(?)", orConditions)
	}

	filterQ.And("created_at >= ?", windowStart)

	// Search across all workspaces — lockout is per-email, not per-workspace.
	logs, err := s.store.SearchAuditLogs(ctx, &store.AuditLogFind{
		FilterQ: filterQ,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to search audit logs for login failures")
	}

	return len(logs), nil
}

// checkPasswordLockout checks if the user has exceeded the password failure rate limit.
// Returns an error if the user is locked out due to too many failed password attempts.
func (s *AuthService) checkPasswordLockout(ctx context.Context, email string) error {
	count, err := s.countRecentLoginFailures(ctx, email, passwordLockoutWindow, errMsgInvalidCredentials)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to count recent password failures"))
	}

	if count >= passwordMaxFailedAttempts {
		return connect.NewError(connect.CodeResourceExhausted, errors.Errorf(errMsgTooManyPassword))
	}

	return nil
}

// checkMFALockout checks if the user has exceeded the MFA failure rate limit.
// Returns an error if the user is locked out due to too many failed MFA attempts.
func (s *AuthService) checkMFALockout(ctx context.Context, email string) error {
	count, err := s.countRecentLoginFailures(ctx, email, mfaLockoutWindow, errMsgInvalidMFACode, errMsgInvalidRecoveryCode)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to count recent MFA failures"))
	}

	if count >= mfaMaxFailedAttempts {
		return connect.NewError(connect.CodeResourceExhausted, errors.Errorf(errMsgTooManyMFA))
	}

	return nil
}

func challengeMFACode(user *store.UserMessage, mfaCode string) error {
	if !validateWithCodeAndSecret(mfaCode, user.MFAConfig.OtpSecret) {
		return connect.NewError(connect.CodeUnauthenticated, errors.Errorf(errMsgInvalidMFACode))
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
	return connect.NewError(connect.CodeUnauthenticated, errors.Errorf(errMsgInvalidRecoveryCode))
}

// validateWithCodeAndSecret validates the given code against the given secret.
func validateWithCodeAndSecret(code, secret string) bool {
	return totp.Validate(code, secret)
}

// syncUserGroups syncs the user groups with the given groups.
// The given groups are the groups that the user belongs to in the identity provider.
// Supported groups format: ["group1", "group2", ...], ["dev@bb.com", ...]
func (s *AuthService) syncUserGroups(ctx context.Context, user *store.UserMessage, workspaceID string, groups []string) error {
	bbGroups, err := s.store.ListGroups(ctx, &store.FindGroupMessage{Workspace: workspaceID})
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list groups"))
	}

	groupChanged := false
	for _, bbGroup := range bbGroups {
		var isMember bool
		for _, group := range groups {
			if bbGroup.Email == group || bbGroup.Title == group {
				isMember = true
				break
			}
		}
		isBBGroupMember := getMemberInGroup(user, bbGroup) != nil
		if isMember != isBBGroupMember {
			if isMember {
				// Add the user to the group.
				bbGroup.Payload.Members = append(bbGroup.Payload.Members, &storepb.GroupMember{
					Role:   storepb.GroupMember_MEMBER,
					Member: common.FormatUserEmail(user.Email),
				})
			} else {
				// Remove the user from the group.
				bbGroup.Payload.Members = slices.DeleteFunc(bbGroup.Payload.Members, func(member *storepb.GroupMember) bool {
					return member.Member == common.FormatUserEmail(user.Email)
				})
			}
			if _, err := s.store.UpdateGroup(ctx, &store.UpdateGroupMessage{
				ID:        bbGroup.ID,
				Workspace: bbGroup.Workspace,
				Payload:   bbGroup.Payload,
			}); err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update group %q", bbGroup.Email))
			}
			groupChanged = true
		}
	}

	// Reload IAM cache if group membership changed.
	if groupChanged {
		if err := s.iamManager.ReloadCache(ctx); err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to reload IAM cache"))
		}
	}

	return nil
}

// authenticateLogin handles all authentication paths: password, IDP, or MFA completion.
// Returns the authenticated user and whether password reset is required.
func (s *AuthService) authenticateLogin(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	mfaSecondLogin := request.GetMfaTempToken() != ""

	if mfaSecondLogin {
		return s.completeMFALogin(ctx, request)
	}

	if request.GetIdpName() != "" {
		return s.getOrCreateUserWithIDP(ctx, request)
	}

	if request.EmailCode != nil && *request.EmailCode != "" {
		return s.authenticateEmailCodeLogin(ctx, request)
	}

	return s.getAndVerifyUser(ctx, request)
}

// completeMFALogin validates MFA temp token and verifies OTP or recovery code.
func (s *AuthService) completeMFALogin(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	userEmail, err := auth.GetUserEmailFromMFATempToken(*request.MfaTempToken, s.secret)
	if err != nil {
		return nil, err
	}
	user, err := s.store.GetUserByEmail(ctx, userEmail)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user"))
	}
	if user == nil {
		return nil, invalidCredentialsError
	}

	if err := s.checkMFALockout(ctx, user.Email); err != nil {
		return nil, err
	}

	if request.OtpCode != nil {
		if err := challengeMFACode(user, *request.OtpCode); err != nil {
			return nil, err
		}
	} else if request.RecoveryCode != nil {
		if err := s.challengeRecoveryCode(ctx, user, *request.RecoveryCode); err != nil {
			return nil, err
		}
	} else {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("OTP or recovery code is required for MFA"))
	}
	return user, nil
}

// validateLoginPermissions checks if the user is allowed to login.
func (s *AuthService) validateLoginPermissions(ctx context.Context, user *store.UserMessage, workspaceID string, request *v1pb.LoginRequest) error {
	if user.MemberDeleted {
		return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("user has been deactivated by administrators"))
	}

	isAdmin, err := isUserWorkspaceAdmin(ctx, s.store, user, workspaceID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check user roles"))
	}

	// Skip restrictions for workspace admins and service accounts
	if isAdmin || user.Type != storepb.PrincipalType_END_USER {
		return nil
	}

	// Skip restrictions for MFA second login (already validated in first step)
	mfaSecondLogin := request.GetMfaTempToken() != ""
	if mfaSecondLogin {
		return nil
	}

	restriction, err := getAccountRestriction(
		ctx,
		s.store,
		s.licenseService,
		s.profile.SaaS,
		workspaceID,
	)
	if err != nil {
		return err
	}
	if request.GetIdpName() == "" {
		if request.Password != "" {
			if restriction.DisallowPasswordSignin {
				return connect.NewError(connect.CodePermissionDenied, errors.Errorf("password signin is disallowed"))
			}
		}
		if request.EmailCode != nil && *request.EmailCode != "" {
			if !restriction.AllowEmailCodeSignin {
				return connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("email code login is not enabled for this workspace"))
			}
		}
	}

	// Check domain restriction
	return validateEmailWithDomains(ctx, s.licenseService, s.store, workspaceID, user.Email, false)
}

// checkMFARequired checks if MFA is required and returns a response with temp token if so.
// Returns (nil, nil) if MFA is not required or already completed.
func (s *AuthService) checkMFARequired(ctx context.Context, user *store.UserMessage, workspaceID string, mfaSecondLogin bool) (*connect.Response[v1pb.LoginResponse], error) {
	if mfaSecondLogin {
		return nil, nil
	}

	userMFAEnabled := user.MFAConfig != nil && user.MFAConfig.OtpSecret != ""
	mfaFeatureEnabled := s.licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_TWO_FA) == nil
	if !mfaFeatureEnabled || !userMFAEnabled {
		return nil, nil
	}

	mfaTempToken, err := auth.GenerateMFATempToken(user.Email, s.secret, mfaTempTokenDuration)
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

	// Reject OAuth2 tokens — they are bound to a specific workspace via the OAuth client
	// and must not be used to mint plain user tokens for other workspaces.
	accessTokenStr, _ := auth.GetTokenFromHeaders(req.Header())
	if accessTokenStr != "" {
		tokenClaims, err := auth.ExtractClaimsFromExpiredToken(accessTokenStr, s.secret)
		if err == nil && slices.Contains(tokenClaims.Audience, auth.OAuth2AccessTokenAudience) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("OAuth2 tokens cannot be used to switch workspaces"))
		}
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
		// Check MFA lockout before verifying.
		if err := s.checkMFALockout(ctx, user.Email); err != nil {
			return nil, err
		}
		// Verify the MFA temp token and OTP/recovery code.
		mfaEmail, err := auth.GetUserEmailFromMFATempToken(*request.MfaTempToken, s.secret)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid MFA temp token"))
		}
		if mfaEmail != user.Email {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("MFA token does not match user"))
		}
		if request.OtpCode != nil {
			if err := challengeMFACode(user, *request.OtpCode); err != nil {
				return nil, err
			}
		} else if request.RecoveryCode != nil {
			if err := s.challengeRecoveryCode(ctx, user, *request.RecoveryCode); err != nil {
				return nil, err
			}
		} else {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("OTP or recovery code required"))
		}
	} else {
		// First step: check if MFA is required for the target workspace.
		if resp, err := s.checkMFARequired(ctx, user, workspaceID, false); err != nil {
			return nil, err
		} else if resp != nil {
			return resp, nil
		}
	}

	return s.switchWorkspaceInternal(ctx, user, workspaceID, request.Web, req.Header())
}

// switchWorkspaceInternal generates new tokens for the target workspace and
// returns a LoginResponse with cookies set. Used by SwitchWorkspace,
// LeaveWorkspace, and DeleteWorkspace.
func (s *AuthService) switchWorkspaceInternal(ctx context.Context, user *store.UserMessage, workspaceID string, web bool, reqHeaders http.Header) (*connect.Response[v1pb.LoginResponse], error) {
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

		origin := reqHeaders.Get("Origin")
		cookie := auth.GetTokenCookie(ctx, s.store, s.licenseService, workspaceID, origin, token)
		resp.Header().Add("Set-Cookie", cookie.String())

		newRefreshToken, err := auth.GenerateOpaqueToken()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to generate refresh token"))
		}
		if err := s.store.CreateWebRefreshToken(ctx, &store.WebRefreshTokenMessage{
			TokenHash: auth.HashToken(newRefreshToken),
			UserEmail: user.Email,
			ExpiresAt: sessionExpiresAt,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create refresh token"))
		}
		refreshCookie := auth.GetRefreshTokenCookie(origin, newRefreshToken, time.Until(sessionExpiresAt))
		resp.Header().Add("Set-Cookie", refreshCookie.String())
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

// finalizeLogin builds the response, sets cookies if needed, and updates the user profile.
func (s *AuthService) finalizeLogin(ctx context.Context, req *connect.Request[v1pb.LoginRequest], user *store.UserMessage, token string, workspaceID string, requireResetPassword bool) (*connect.Response[v1pb.LoginResponse], error) {
	response := &v1pb.LoginResponse{
		RequireResetPassword: requireResetPassword,
	}
	resp := connect.NewResponse(response)

	if req.Msg.Web {
		if user.Type != storepb.PrincipalType_END_USER {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("only users can use web login"))
		}
		origin := req.Header().Get("Origin")
		cookie := auth.GetTokenCookie(ctx, s.store, s.licenseService, workspaceID, origin, token)
		resp.Header().Add("Set-Cookie", cookie.String())

		// Issue refresh token for web login
		refreshToken, err := auth.GenerateOpaqueToken()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to generate refresh token"))
		}
		refreshTokenDuration := auth.GetRefreshTokenDuration(ctx, s.store, s.licenseService, workspaceID)
		if err := s.store.CreateWebRefreshToken(ctx, &store.WebRefreshTokenMessage{
			TokenHash: auth.HashToken(refreshToken),
			UserEmail: user.Email,
			ExpiresAt: time.Now().Add(refreshTokenDuration),
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create refresh token"))
		}
		refreshCookie := auth.GetRefreshTokenCookie(origin, refreshToken, refreshTokenDuration)
		resp.Header().Add("Set-Cookie", refreshCookie.String())
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

// RequestPasswordReset sends a password reset email. Always returns success to avoid leaking email existence.
func (s *AuthService) RequestPasswordReset(ctx context.Context, req *connect.Request[v1pb.RequestPasswordResetRequest]) (*connect.Response[emptypb.Empty], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}

	// Send synchronously, but swallow errors to avoid email enumeration — a fast silent
	// "success" for unknown emails must be indistinguishable from an SMTP failure for
	// known ones. Errors are logged server-side for operator visibility.
	if err := s.sendEmailVerificationCode(
		ctx,
		req.Msg.Workspace,
		email,
		storepb.EmailVerificationCodePurpose_PASSWORD_RESET,
		"[Bytebase] Reset your password",
		"Hi,\n\nYour password reset code is: %s\n\nThis code expires in %d minutes. If you didn't request this, you can safely ignore this email.\n\n— Bytebase",
	); err != nil {
		slog.Warn("failed to send password reset email", slog.String("to", email), log.BBError(err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ResetPassword verifies the 6-digit code and updates the user's password.
// Also revokes all refresh tokens to force re-login.
func (s *AuthService) ResetPassword(ctx context.Context, req *connect.Request[v1pb.ResetPasswordRequest]) (*connect.Response[emptypb.Empty], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}
	if req.Msg.Code == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("code is required"))
	}
	if req.Msg.NewPassword == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("new_password is required"))
	}

	codeRow, err := s.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_PASSWORD_RESET, req.Msg.Code)
	if err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user"))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user not found"))
	}

	// Validate the user is a member of the workspace captured at send time.
	// Reject if forged — prevents bypassing password policy via a weaker workspace.
	if codeRow.Workspace != "" {
		ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
			WorkspaceID:    &codeRow.Workspace,
			Email:          email,
			IncludeAllUser: !s.profile.SaaS,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to verify workspace membership"))
		}
		if ws == nil {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user is not a member of the workspace"))
		}
	}
	restriction, err := getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, codeRow.Workspace)
	if err != nil {
		return nil, err
	}
	if err := validatePasswordWithRestriction(req.Msg.NewPassword, convertToStorePasswordRestriction(restriction.PasswordRestriction)); err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Msg.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to hash password"))
	}
	passwordHashStr := string(passwordHash)
	if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
		Email:        &user.Email,
		PasswordHash: &passwordHashStr,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update password"))
	}

	if err := s.store.DeleteWebRefreshTokensByUser(ctx, user.Email); err != nil {
		slog.Warn("failed to revoke refresh tokens after password reset", log.BBError(err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
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

// generateEmailCode returns a cryptographically-random 6-digit numeric code.
func generateEmailCode() (string, error) {
	const digits = "0123456789"
	b := make([]byte, emailCodeLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b), nil
}

// hashEmailCode returns HMAC-SHA256(code) hex-encoded, keyed with the server's auth secret.
// HMAC with a server-side secret (vs. bare SHA-256) prevents offline brute force of the
// 10^6-size code space if the DB is ever compromised — the attacker would also need the
// auth secret to verify candidate codes.
func (s *AuthService) hashEmailCode(code string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

// SendEmailLoginCode sends a 6-digit verification code. Always returns success
// (no email enumeration). Rate limit: 60-sec resend cooldown enforced atomically
// via the store — effective cap ≈ 60 sends/hour/email.
func (s *AuthService) SendEmailLoginCode(ctx context.Context, req *connect.Request[v1pb.SendEmailLoginCodeRequest]) (*connect.Response[emptypb.Empty], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}
	workspaceID, err := parseOptionalWorkspace(req.Msg.Workspace)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Gate on AllowEmailCodeSignin — no point emailing a code the workspace won't accept.
	// getAccountRestriction handles all cases (including empty workspace for brand-new SaaS
	// signup, where it resolves via EMAIL_CONFIG + SaaS override).
	restriction, err := getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, workspaceID)
	if err != nil {
		return nil, err
	}
	if !restriction.AllowEmailCodeSignin {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("email code login is not enabled for this workspace"))
	}

	// Send synchronously so the caller learns about actionable failures (missing EMAIL
	// setting, SMTP unreachable, etc.). No enumeration risk here: LOGIN always attempts to
	// send regardless of whether the email exists (sign-up is handled on verify).
	if err := s.sendEmailVerificationCode(
		ctx,
		req.Msg.Workspace,
		email,
		storepb.EmailVerificationCodePurpose_LOGIN,
		"[Bytebase] Your sign-in code",
		"Hi,\n\nYour sign-in code is: %s\n\nThis code expires in %d minutes. If you didn't request this, you can safely ignore this email.\n\n— Bytebase",
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// resolvePreLoginEmailSetting returns the EMAIL setting to use for unauthenticated flows
// (email-code sign-in, password reset). Resolution order:
//  1. If `workspaceID` is provided, use that workspace's EMAIL setting. The frontend
//     resolves the workspace (from the route query or actuator context) and always passes
//     it when one exists — self-host, multi-workspace SaaS, and pre-invited emails all
//     flow through this path.
//  2. EMAIL_CONFIG env var — deployment-wide fallback for SaaS brand-new signup, where
//     the caller has no workspace context yet (no pre-invite, no workspace in the URL).
func resolvePreLoginEmailSetting(
	ctx context.Context,
	stores *store.Store,
	workspaceID string,
) (*storepb.EmailSetting, error) {
	if workspaceID != "" {
		emailSettingMsg, err := stores.GetSetting(ctx, workspaceID, storepb.SettingName_EMAIL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load email setting")
		}
		if emailSettingMsg == nil {
			return nil, nil
		}
		es, ok := emailSettingMsg.Value.(*storepb.EmailSetting)
		if !ok {
			return nil, nil
		}
		return es, nil
	}

	if raw := os.Getenv("EMAIL_CONFIG"); raw != "" {
		emailSetting := &storepb.EmailSetting{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(raw), emailSetting); err != nil {
			return nil, errors.Wrap(err, "failed to parse EMAIL_CONFIG")
		}
		return emailSetting, nil
	}

	return nil, nil
}

// sendEmailVerificationCode generates a code, atomically stores its hash (subject to cooldown),
// and emails the plain code. Returns nil on a successful send as well as on silent-skip paths
// (cooldown active, or PASSWORD_RESET for an unknown email) — both are intentionally
// indistinguishable to the caller, since both correspond to "no new email was delivered".
// Returns an error only on actionable failures (missing EMAIL setting, SMTP failure, DB error).
// Callers decide whether to propagate the error: `SendEmailLoginCode` surfaces it (users need
// to know email delivery failed), `RequestPasswordReset` swallows it to avoid revealing that
// the account exists. `bodyFmt` must contain one %s for the 6-digit code.
func (s *AuthService) sendEmailVerificationCode(ctx context.Context, workspaceName *string, email string, purpose storepb.EmailVerificationCodePurpose, subject, bodyFmt string) error {
	// For password reset, only send to existing principals — no upsert, no email for unknown addresses.
	// Prevents arbitrary email spam. LOGIN intentionally skips this check (also covers signup).
	if purpose == storepb.EmailVerificationCodePurpose_PASSWORD_RESET {
		account, err := s.store.GetAccountByEmail(ctx, email)
		if err != nil {
			return errors.Wrap(err, "failed to look up account for password reset")
		}
		if account == nil || account.Type != storepb.PrincipalType_END_USER {
			return nil // silent: account doesn't exist
		}
	}

	workspaceID, err := parseOptionalWorkspace(workspaceName)
	if err != nil {
		return errors.Wrap(err, "failed to parse workspace id")
	}

	// Resolve the EMAIL setting FIRST — fail fast if misconfigured so we don't write a
	// verification row we can't actually deliver.
	emailSetting, err := resolvePreLoginEmailSetting(ctx, s.store, workspaceID)
	if err != nil {
		return err
	}
	if emailSetting == nil {
		return errors.Errorf("cannot found email config for workspace %v", workspaceID)
	}

	code, err := generateEmailCode()
	if err != nil {
		return errors.Wrap(err, "failed to generate code")
	}

	now := time.Now()
	sent, err := s.store.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
		Email:      email,
		Purpose:    purpose,
		CodeHash:   s.hashEmailCode(code),
		ExpiresAt:  now.Add(emailCodeExpiry),
		LastSentAt: now,
		Workspace:  workspaceID,
	}, emailCodeResendCooldown)
	if err != nil {
		return errors.Wrap(err, "failed to upsert verification code")
	}
	if !sent {
		return nil // cooldown active — silent skip
	}

	sender, err := mailer.NewSender(emailSetting)
	if err != nil {
		return errors.Wrap(err, "failed to create mail sender")
	}

	body := fmt.Sprintf(bodyFmt, code, int(emailCodeExpiry.Minutes()))
	if err := sender.Send(ctx, &mailer.SendRequest{
		To:       []string{email},
		Subject:  subject,
		TextBody: body,
	}); err != nil {
		// Delete the row so the cooldown doesn't block an immediate retry.
		// Match on code_hash to avoid wiping a newer code from a concurrent request.
		_ = s.store.DeleteEmailVerificationCodeIfMatch(ctx, email, purpose, s.hashEmailCode(code))
		return errors.Wrap(err, "failed to send email")
	}
	return nil
}

// verifyEmailCode checks a submitted code against the stored row.
// Enforces expiry, attempt limit (5), and constant-time hash compare.
// On successful match, deletes the row (one-time use) and returns it so the
// caller can use its captured workspace context (e.g. for gate checks and
// workspace assignment on the email-code signup path).
func (s *AuthService) verifyEmailCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose, submittedCode string) (*store.EmailVerificationCodeMessage, error) {
	row, err := s.store.GetEmailVerificationCode(ctx, email, purpose)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get email verification code"))
	}
	if row == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("invalid or expired code"))
	}
	if time.Now().After(row.ExpiresAt) {
		_ = s.store.DeleteEmailVerificationCodeIfMatch(ctx, email, purpose, row.CodeHash)
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("invalid or expired code"))
	}
	if row.Attempts >= emailCodeMaxAttempts {
		_ = s.store.DeleteEmailVerificationCodeIfMatch(ctx, email, purpose, row.CodeHash)
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("too many attempts"))
	}
	if subtle.ConstantTimeCompare([]byte(s.hashEmailCode(submittedCode)), []byte(row.CodeHash)) != 1 {
		_ = s.store.IncrementEmailVerificationCodeAttempts(ctx, email, purpose)
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("invalid or expired code"))
	}
	_ = s.store.DeleteEmailVerificationCodeIfMatch(ctx, email, purpose, row.CodeHash)
	return row, nil
}

// authenticateEmailCodeLogin handles the email + 6-digit code flow.
// Existing users: verify code → return user (downstream pipeline handles workspace-level gates).
// Unknown emails: verify code → gate checks on pre-invited workspace → create user + provision workspace.
func (s *AuthService) authenticateEmailCodeLogin(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	if request.Password != "" || request.GetIdpName() != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email_code is mutually exclusive with password and idp_name"))
	}
	email := strings.ToLower(strings.TrimSpace(request.Email))
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}

	codeRow, err := s.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, *request.EmailCode)
	if err != nil {
		return nil, err
	}

	// Existing user → return. allow_email_code_signin is checked later in validateLoginPermissions
	// against the actually-resolved login workspace (which may not match the send-time workspace
	// for multi-workspace users — resolveWorkspaceForLogin prefers LastLoginWorkspace).
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user"))
	}
	if user != nil {
		return user, nil
	}

	// Unknown email → signup path. Validate email format to prevent reserved-namespace collisions.
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}

	// Gate checks run BEFORE user creation to prevent orphan accounts.
	// We only consult AllowEmailCodeSignin here: DisallowSignup governs password self-service
	// signup (the Signup RPC), not email-code onboarding — the two paths are independent.
	// Admins who want to block new users via email-code set AllowEmailCodeSignin=false.
	restriction, err := getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, codeRow.Workspace)
	if err != nil {
		return nil, err
	}
	if !restriction.AllowEmailCodeSignin {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("email code login is not enabled for this workspace"))
	}
	// Signup is always allowed for SaaS
	if !s.profile.SaaS {
		if restriction.DisallowSignup {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("sign up is disallowed for this workspace"))
		}
	}

	// Provision workspace BEFORE creating the user so retries are self-healing: if user
	// creation fails, the next attempt's FindWorkspace(email) finds the already-provisioned
	// workspace via its IAM binding and returns it. The reverse order would leave a user
	// without a workspace, and subsequent retries would early-return via GetUserByEmail and
	// never re-run provisioning — permanently stuck. Matches the Signup RPC's ordering.
	if _, err := s.provisionWorkspaceForNewUser(ctx, email); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to provision workspace"))
	}

	// Create principal with random bcrypt password.
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate random password"))
	}
	passwordHash, err := bcrypt.GenerateFromPassword(randomBytes, bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to hash password"))
	}

	// Derive display name from email local-part.
	name := email
	if i := strings.Index(email, "@"); i > 0 {
		name = email[:i]
	}

	newUser, err := s.store.CreateUser(ctx, &store.UserMessage{
		Email:        email,
		Name:         name,
		Type:         storepb.PrincipalType_END_USER,
		PasswordHash: string(passwordHash),
		Profile:      &storepb.UserProfile{},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create user"))
	}

	return newUser, nil
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
				Model:    "gemini-2.5-pro",
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
