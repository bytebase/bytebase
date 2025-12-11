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
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/idp/ldap"
	"github.com/bytebase/bytebase/backend/plugin/idp/oauth2"
	"github.com/bytebase/bytebase/backend/plugin/idp/oidc"
	"github.com/bytebase/bytebase/backend/plugin/idp/wif"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
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
	metricReporter *metricreport.Reporter
	profile        *config.Profile
	stateCfg       *state.State
	iamManager     *iam.Manager
}

// NewAuthService creates a new AuthService.
func NewAuthService(store *store.Store, secret string, licenseService *enterprise.LicenseService, metricReporter *metricreport.Reporter, profile *config.Profile, stateCfg *state.State, iamManager *iam.Manager) *AuthService {
	return &AuthService{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		metricReporter: metricReporter,
		profile:        profile,
		stateCfg:       stateCfg,
		iamManager:     iamManager,
	}
}

// Login is the auth login method including SSO.
func (s *AuthService) Login(ctx context.Context, req *connect.Request[v1pb.LoginRequest]) (*connect.Response[v1pb.LoginResponse], error) {
	request := req.Msg
	var loginUser *store.UserMessage
	mfaSecondLogin := request.MfaTempToken != nil && *request.MfaTempToken != ""
	loginViaIDP := request.GetIdpName() != ""

	response := &v1pb.LoginResponse{}
	resp := connect.NewResponse(response)
	if !mfaSecondLogin {
		var err error
		if loginViaIDP {
			loginUser, err = s.getOrCreateUserWithIDP(ctx, request)
			if err != nil {
				return nil, err
			}
		} else {
			loginUser, err = s.getAndVerifyUser(ctx, request)
			if err != nil {
				return nil, err
			}
			// Reset password restriction only works for end user with email & password login.
			response.RequireResetPassword = s.needResetPassword(ctx, loginUser)
		}
	} else {
		userID, err := auth.GetUserIDFromMFATempToken(*request.MfaTempToken, s.profile.Mode, s.secret)
		if err != nil {
			return nil, err
		}
		user, err := s.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user, error"))
		}
		if user == nil {
			return nil, invalidCredentialsError
		}

		// Check if user is locked out due to too many failed MFA attempts.
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
		loginUser = user
	}

	if loginUser.MemberDeleted {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("user has been deactivated by administrators"))
	}

	setting, err := s.store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting, error"))
	}
	isWorkspaceAdmin, err := isUserWorkspaceAdmin(ctx, s.store, loginUser)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check user roles, error"))
	}
	if !isWorkspaceAdmin && loginUser.Type == storepb.PrincipalType_END_USER && !mfaSecondLogin {
		// Disallow password signin for end users.
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DISALLOW_PASSWORD_SIGNIN); err == nil {
			if setting.DisallowPasswordSignin && !loginViaIDP {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("password signin is disallowed"))
			}
		}

		// Check domain restriction for end users.
		if err := validateEmailWithDomains(ctx, s.licenseService, s.store, loginUser.Email, false, false); err != nil {
			return nil, err
		}
	}

	tokenDuration := auth.GetTokenDuration(ctx, s.store, s.licenseService)
	userMFAEnabled := loginUser.MFAConfig != nil && loginUser.MFAConfig.OtpSecret != ""
	// We only allow MFA login (2-step) when the feature is enabled and user has enabled MFA.
	if s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_TWO_FA) == nil && !mfaSecondLogin && userMFAEnabled {
		mfaTempToken, err := auth.GenerateMFATempToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret, mfaTempTokenDuration)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate MFA temp token"))
		}
		return connect.NewResponse(&v1pb.LoginResponse{
			MfaTempToken: &mfaTempToken,
		}), nil
	}

	switch loginUser.Type {
	case storepb.PrincipalType_END_USER:
		token, err := auth.GenerateAccessToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret, tokenDuration)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate API access token"))
		}
		response.Token = token
	case storepb.PrincipalType_SERVICE_ACCOUNT:
		token, err := auth.GenerateAPIToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate API access token"))
		}
		response.Token = token
	default:
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("user type %s cannot login", loginUser.Type))
	}

	if request.Web {
		// Only allow end users to use web login, not service accounts.
		if loginUser.Type != storepb.PrincipalType_END_USER {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("only users can use web login"))
		}

		origin := req.Header().Get("Origin")
		if origin == "" {
			origin = req.Header().Get("grpcgateway-origin")
		}

		cookie := auth.GetTokenCookie(ctx, s.store, s.licenseService, origin, response.Token)
		resp.Header().Add("Set-Cookie", cookie.String())
	}

	if _, err := s.store.UpdateUser(ctx, loginUser, &store.UpdateUserMessage{
		Profile: &storepb.UserProfile{
			LastLoginTime:          timestamppb.Now(),
			LastChangePasswordTime: loginUser.Profile.GetLastChangePasswordTime(),
		},
	}); err != nil {
		slog.Error("failed to update user profile", log.BBError(err), slog.String("user", loginUser.Email))
	}

	response.User = convertToUser(ctx, loginUser)

	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricapi.PrincipalLoginMetricName,
		Value: 1,
		Labels: map[string]any{
			"email": loginUser.Email,
		},
	})

	return resp, nil
}

func (s *AuthService) needResetPassword(ctx context.Context, user *store.UserMessage) bool {
	// Reset password restriction only works for end user with email & password login.
	if user.Type != storepb.PrincipalType_END_USER {
		return false
	}
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_PASSWORD_RESTRICTIONS); err != nil {
		return false
	}

	setting, err := s.store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		slog.Error("failed to get workspace setting", log.BBError(err))
		return false
	}
	passwordRestriction := setting.GetPasswordRestriction()

	if user.Profile.LastLoginTime == nil {
		if !passwordRestriction.GetRequireResetPasswordForFirstLogin() {
			return false
		}
		count, err := s.store.CountUsers(ctx, storepb.PrincipalType_END_USER)
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

// Logout is the auth logout method.
func (s *AuthService) Logout(ctx context.Context, req *connect.Request[v1pb.LogoutRequest]) (*connect.Response[emptypb.Empty], error) {
	accessTokenStr, err := auth.GetTokenFromHeaders(req.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	s.stateCfg.ExpireCache.Add(accessTokenStr, true)

	resp := connect.NewResponse(&emptypb.Empty{})

	origin := req.Header().Get("Origin")
	if origin == "" {
		origin = req.Header().Get("grpcgateway-origin")
	}
	cookie := auth.GetTokenCookie(ctx, s.store, s.licenseService, origin, "")
	resp.Header().Add("Set-Cookie", cookie.String())
	return resp, nil
}

func (s *AuthService) getAndVerifyUser(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	// Check if user is locked out due to too many failed password attempts.
	if err := s.checkPasswordLockout(ctx, request.Email); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user by email %q", request.Email))
	}
	if user == nil {
		return nil, invalidCredentialsError
	}
	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		// If the two passwords don't match, return a 401 status.
		return nil, invalidCredentialsError
	}
	return user, nil
}

func (s *AuthService) getOrCreateUserWithIDP(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	idpID, err := common.GetIdentityProviderID(request.IdpName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get identity provider ID"))
	}
	idp, err := s.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &idpID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get identity provider"))
	}
	if idp == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("identity provider not found"))
	}

	setting, err := s.store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get workspace setting"))
	}

	// Use command-line flag value if set, otherwise use database value
	externalURL := common.GetEffectiveExternalURL(s.profile.ExternalURL, setting.ExternalUrl)

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
	if err := validateEmail(email); err != nil {
		// If the email is invalid, we will try to use the domain and identifier to construct the email.
		domain := extractDomain(idp.Domain)
		if domain != "" {
			email = strings.ToLower(fmt.Sprintf("%s@%s", email, domain))
		} else {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid email %q", userInfo.Identifier))
		}
	}
	// If the email is still invalid, we will return an error.
	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, email, false /* isServiceAccount */, false); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list users by email %s", email))
	}
	if user != nil {
		if user.MemberDeleted {
			if err := s.userCountGuard(ctx); err != nil {
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
			if err := s.syncUserGroups(ctx, user, userInfo.Groups); err != nil {
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
	if err := s.licenseService.IsFeatureEnabled(featurePlan); err != nil {
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
	if err := s.userCountGuard(ctx); err != nil {
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
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create user, error"))
	}
	if userInfo.HasGroups {
		// Sync user groups with the identity provider.
		// The userInfo.Groups is the groups that the user belongs to in the identity provider.
		if err := s.syncUserGroups(ctx, newUser, userInfo.Groups); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to sync user groups"))
		}
	}
	return newUser, nil
}

func (s *AuthService) userCountGuard(ctx context.Context) error {
	userLimit := s.licenseService.GetUserLimit(ctx)

	count, err := s.store.CountActiveUsers(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if count >= userLimit {
		return connect.NewError(connect.CodeResourceExhausted, errors.Errorf("reached the maximum user count %d", userLimit))
	}
	return nil
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
func (s *AuthService) syncUserGroups(ctx context.Context, user *store.UserMessage, groups []string) error {
	bbGroups, err := s.store.ListGroups(ctx, &store.FindGroupMessage{})
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
		var isBBGroupMember bool
		for _, member := range bbGroup.Payload.Members {
			if member.Member == common.FormatUserUID(user.ID) {
				isBBGroupMember = true
				break
			}
		}
		if isMember != isBBGroupMember {
			if isMember {
				// Add the user to the group.
				bbGroup.Payload.Members = append(bbGroup.Payload.Members, &storepb.GroupMember{
					Role:   storepb.GroupMember_MEMBER,
					Member: common.FormatUserUID(user.ID),
				})
			} else {
				// Remove the user from the group.
				bbGroup.Payload.Members = slices.DeleteFunc(bbGroup.Payload.Members, func(member *storepb.GroupMember) bool {
					return member.Member == common.FormatUserUID(user.ID)
				})
			}
			if _, err := s.store.UpdateGroup(ctx, &store.UpdateGroupMessage{
				ID:      bbGroup.ID,
				Payload: bbGroup.Payload,
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
			errors.Errorf("email must end with %s", common.WorkloadIdentityEmailSuffix))
	}

	// Find workload identity by email
	user, err := s.store.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to find workload identity"))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q not found", request.Email))
	}
	if user.Type != storepb.PrincipalType_WORKLOAD_IDENTITY {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.Errorf("email %q is not a workload identity", request.Email))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodePermissionDenied,
			errors.New("workload identity has been deactivated"))
	}

	// Get workload identity config
	wicConfig := user.Profile.GetWorkloadIdentityConfig()
	if wicConfig == nil {
		return nil, connect.NewError(connect.CodeInternal,
			errors.New("workload identity config not found"))
	}

	// Validate OIDC token
	if _, err = wif.ValidateToken(ctx, request.Token, wicConfig); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			errors.Wrap(err, "token validation failed"))
	}

	// Generate Bytebase API token (1 hour duration, same as service account)
	token, err := auth.GenerateAPIToken(user.Name, user.ID, s.profile.Mode, s.secret)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			errors.Wrap(err, "failed to generate access token"))
	}

	// Update last login time
	if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
		Profile: &storepb.UserProfile{
			LastLoginTime:          timestamppb.Now(),
			WorkloadIdentityConfig: wicConfig,
		},
	}); err != nil {
		slog.Error("failed to update workload identity profile", log.BBError(err), slog.String("email", user.Email))
	}

	return connect.NewResponse(&v1pb.ExchangeTokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
	}), nil
}
