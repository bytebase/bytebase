package v1

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
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
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

const (
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

	// 2. Resolve workspace early so all subsequent checks can use it.
	// Login is allow_without_credential, so workspace is NOT in the context from auth middleware.
	workspaceID, err := s.resolveWorkspaceForLogin(ctx, loginUser)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to resolve workspace"))
	}

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
	if err := s.licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_PASSWORD_RESTRICTIONS); err != nil {
		return false
	}

	setting, err := s.store.GetWorkspaceProfileSetting(ctx, workspaceID)
	if err != nil {
		slog.Error("failed to get workspace setting", log.BBError(err))
		return false
	}
	passwordRestriction := setting.GetPasswordRestriction()

	if user.Profile.LastLoginTime == nil {
		if !passwordRestriction.GetRequireResetPasswordForFirstLogin() {
			return false
		}
		count, err := s.store.CountActiveEndUsersPerWorkspace(ctx, workspaceID)
		if err != nil {
			slog.Error("failed to count end users", log.BBError(err))
			return false
		}
		// The 1st workspace admin login don't need to reset the password
		return count > 1
	}

	if passwordRestriction.GetPasswordRotation() != nil {
		lastChangePasswordTime := user.CreatedAt
		if user.Profile.LastChangePasswordTime != nil {
			lastChangePasswordTime = user.Profile.LastChangePasswordTime.AsTime()
		}
		if lastChangePasswordTime.Add(passwordRestriction.GetPasswordRotation().AsDuration()).Before(time.Now()) {
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

	// Step 1: Resolve workspace — check if user belongs to an existing workspace.
	existingWS, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		Email:          request.Email,
		IncludeAllUser: !s.profile.SaaS,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspaces"))
	}

	var workspaceID string
	isMember := existingWS != nil
	if isMember {
		workspaceID = existingWS.ResourceID
	} else if !s.profile.SaaS {
		// Self-hosted: join the existing workspace (will be added as member in step 3).
		// No workspace membership found. Check if any workspace exists.
		existingWsID, err := s.store.GetWorkspaceID(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check workspace"))
		}

		workspaceID = existingWsID
	}

	if workspaceID != "" {
		if !s.profile.SaaS {
			if err := s.licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_DISALLOW_SELF_SERVICE_SIGNUP); err == nil {
				setting, err := s.store.GetWorkspaceProfileSetting(ctx, workspaceID)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get workspace setting"))
				}
				if setting.DisallowSignup {
					return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("sign up is disallowed for this workspace"))
				}
			}
		}

		if err := validatePassword(ctx, s.store, workspaceID, request.Password); err != nil {
			return nil, err
		}
	} else if err := validatePasswordWithRestriction(request.Password, convertToStorePasswordRestriction(defaultAccountRestriction.PasswordRestriction)); err != nil {
		return nil, err
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

	// Step 3: Resolve workspace if not yet assigned.
	if workspaceID == "" {
		// Create a new workspace with the user as admin.
		wsID, err := common.RandomString(16)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to generate workspace ID"))
		}
		ws, err := s.store.CreateWorkspace(ctx, &store.WorkspaceMessage{
			ResourceID: wsID,
			Name:       "Default workspace",
		}, request.Email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create workspace"))
		}
		workspaceID = ws.ResourceID
	} else if !isMember {
		// Self-hosted: add user as workspace member to the existing workspace.
		if _, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
			Workspace: workspaceID,
			Member:    common.FormatUserEmail(request.Email),
			Roles:     []string{common.FormatRole(store.WorkspaceMemberRole)},
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to add user to workspace"))
		}
	}

	// Step 4: Generate token and finalize login.
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

// Logout is the auth logout method.
func (s *AuthService) Logout(ctx context.Context, req *connect.Request[v1pb.LogoutRequest]) (*connect.Response[emptypb.Empty], error) {
	// Delete refresh token from database if present
	if refreshToken := auth.GetRefreshTokenFromCookie(req.Header()); refreshToken != "" {
		if err := s.store.DeleteWebRefreshToken(ctx, auth.HashToken(refreshToken)); err != nil {
			slog.Error("failed to delete refresh token on logout", log.BBError(err))
		}
	}

	resp := connect.NewResponse(&emptypb.Empty{})

	origin := req.Header().Get("Origin")
	// Clear access token cookie
	resp.Header().Add("Set-Cookie", auth.GetTokenCookie(ctx, s.store, s.licenseService, common.GetWorkspaceIDFromContext(ctx), origin, "").String())
	// Clear refresh token cookie
	resp.Header().Add("Set-Cookie", auth.GetRefreshTokenCookie(origin, "", 0).String())
	return resp, nil
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
	return s.accountToUser(ctx, account)
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
	// For global IDPs (SaaS), resolve workspace from user's IAM membership.
	if workspaceID == "" {
		ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
			Email:          email,
			IncludeAllUser: !s.profile.SaaS,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to find workspace for SSO user"))
		}
		if ws != nil {
			workspaceID = ws.ResourceID
		}
		// If no workspace found, workspaceID stays empty — the user will need to be invited first.
	}

	// If the email is still invalid, we will return an error.
	if workspaceID != "" {
		if err := validateEmailWithDomains(ctx, s.licenseService, s.store, workspaceID, email, false); err != nil {
			return nil, err
		}
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list users by email %s", email))
	}

	if user != nil {
		if user.MemberDeleted {
			if err := userCountGuard(ctx, s.store, s.licenseService, workspaceID); err != nil {
				return nil, err
			}
			// Undelete the user when login via SSO.
			user, err = s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &undeletePatch})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to undelete user"))
			}
		}
		if userInfo.HasGroups {
			// Sync user groups with the identity provider.
			// The userInfo.Groups is the groups that the user belongs to in the identity provider.
			if err := s.syncUserGroups(ctx, user, workspaceID, userInfo.Groups); err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to sync user groups"))
			}
		}
		return user, nil
	}

	// For expired license, we will only block new create creation and still allow SSO login from existing users.
	featurePlan := v1pb.PlanFeature_FEATURE_ENTERPRISE_SSO
	if idp.Type == storepb.IdentityProviderType_OAUTH2 && googleGitHubDomains[idp.Domain] {
		featurePlan = v1pb.PlanFeature_FEATURE_GOOGLE_AND_GITHUB_SSO
	}
	if err := s.licenseService.IsFeatureEnabled(ctx, workspaceID, featurePlan); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
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
	if err := userCountGuard(ctx, s.store, s.licenseService, workspaceID); err != nil {
		return nil, err
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

	// Add the new user to the workspace IAM policy as a member.
	// The IDP is workspace-scoped, so authenticating through it grants workspace access.
	if _, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: workspaceID,
		Member:    common.FormatUserEmail(email),
		Roles:     []string{common.FormatRole(store.WorkspaceMemberRole)},
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to add user to workspace"))
	}

	if userInfo.HasGroups {
		// Sync user groups with the identity provider.
		// The userInfo.Groups is the groups that the user belongs to in the identity provider.
		if err := s.syncUserGroups(ctx, newUser, workspaceID, userInfo.Groups); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to sync user groups"))
		}
	}
	return newUser, nil
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

	loginViaIDP := request.GetIdpName() != ""

	// Check disallow password signin
	if err := s.licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_DISALLOW_PASSWORD_SIGNIN); err == nil {
		setting, err := s.store.GetWorkspaceProfileSetting(ctx, workspaceID)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting"))
		}
		if setting.DisallowPasswordSignin && !loginViaIDP {
			return connect.NewError(connect.CodePermissionDenied, errors.Errorf("password signin is disallowed"))
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
// For END_USER: prefers the last login workspace (from user profile) if still valid,
// otherwise falls back to the first workspace from IAM membership.
func (s *AuthService) resolveWorkspaceForLogin(ctx context.Context, user *store.UserMessage) (string, error) {
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
			return "", errors.Errorf("%q is not a member of any workspace", user.Email)
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

	// Generate new token with target workspace.
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

	// Build response.
	response := &v1pb.LoginResponse{}
	resp := connect.NewResponse(response)

	if request.Web {
		// Require a valid refresh token cookie — prevents non-web clients from
		// upgrading a short-lived bearer token into a long-lived web session.
		oldRefreshToken := auth.GetRefreshTokenFromCookie(req.Header())
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

		origin := req.Header().Get("Origin")
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

	// Validate email format
	if !common.IsWorkloadIdentityEmail(request.Email) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.Errorf("email must end with %s", common.WorkloadIdentitySuffix))
	}

	// Find workload identity by email (cross-workspace lookup since this is unauthenticated).
	wi, err := s.store.GetWorkloadIdentityByEmail(ctx, request.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to find workload identity"))
	}
	if wi == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q not found", request.Email))
	}
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

// accountToUser converts an AccountMessage to a UserMessage.
// For END_USER, loads the full user record. For SA/WI, constructs a minimal UserMessage.
func (s *AuthService) accountToUser(ctx context.Context, account *store.AccountMessage) (*store.UserMessage, error) {
	if account.Type == storepb.PrincipalType_END_USER {
		user, err := s.store.GetUserByEmail(ctx, account.Email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user %q", account.Email))
		}
		if user == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("user %q not found", account.Email))
		}
		return user, nil
	}

	// SA/WI: construct a minimal UserMessage with the fields available from AccountMessage.
	return &store.UserMessage{
		Email:         account.Email,
		Name:          account.Name,
		Type:          account.Type,
		MemberDeleted: account.MemberDeleted,
	}, nil
}
