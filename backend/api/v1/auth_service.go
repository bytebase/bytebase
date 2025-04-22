package v1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/idp/ldap"
	"github.com/bytebase/bytebase/backend/plugin/idp/oauth2"
	"github.com/bytebase/bytebase/backend/plugin/idp/oidc"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type CreateUserFunc func(ctx context.Context, user *store.UserMessage, firstEndUser bool) error

var (
	invalidUserOrPasswordError = status.Errorf(codes.Unauthenticated, "The email or password is not valid.")
)

// AuthService implements the auth service.
type AuthService struct {
	v1pb.UnimplementedAuthServiceServer
	store          *store.Store
	secret         string
	licenseService enterprise.LicenseService
	metricReporter *metricreport.Reporter
	profile        *config.Profile
	stateCfg       *state.State
	iamManager     *iam.Manager
	postCreateUser func(ctx context.Context, user *store.UserMessage, firstEndUser bool) error
}

// NewAuthService creates a new AuthService.
func NewAuthService(store *store.Store, secret string, licenseService enterprise.LicenseService, metricReporter *metricreport.Reporter, profile *config.Profile, stateCfg *state.State, iamManager *iam.Manager, postCreateUser func(ctx context.Context, user *store.UserMessage, firstEndUser bool) error) *AuthService {
	return &AuthService{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		metricReporter: metricReporter,
		profile:        profile,
		stateCfg:       stateCfg,
		iamManager:     iamManager,
		postCreateUser: postCreateUser,
	}
}

// Login is the auth login method including SSO.
func (s *AuthService) Login(ctx context.Context, request *v1pb.LoginRequest) (*v1pb.LoginResponse, error) {
	var loginUser *store.UserMessage
	mfaSecondLogin := request.MfaTempToken != nil && *request.MfaTempToken != ""
	loginViaIDP := request.GetIdpName() != ""

	response := &v1pb.LoginResponse{}
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
			return nil, err
		}
		if user == nil {
			return nil, invalidUserOrPasswordError
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
			return nil, status.Errorf(codes.Unauthenticated, "OTP or recovery code is required for MFA")
		}
		loginUser = user
	}

	if loginUser.MemberDeleted {
		return nil, status.Errorf(codes.Unauthenticated, "user has been deactivated by administrators")
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace setting, error: %v", err)
	}
	isWorkspaceAdmin, err := isUserWorkspaceAdmin(ctx, s.store, loginUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check user roles, error: %v", err)
	}
	if !isWorkspaceAdmin && loginUser.Type == base.EndUser && !mfaSecondLogin {
		// Disallow password signin for end users.
		if setting.DisallowPasswordSignin && !loginViaIDP {
			return nil, status.Errorf(codes.PermissionDenied, "password signin is disallowed")
		}
		// Check domain restriction for end users.
		if err := validateEmailWithDomains(ctx, s.licenseService, s.store, loginUser.Email, false, false); err != nil {
			return nil, err
		}
	}

	tokenDuration := auth.GetTokenDuration(ctx, s.store)
	userMFAEnabled := loginUser.MFAConfig != nil && loginUser.MFAConfig.OtpSecret != ""
	// We only allow MFA login (2-step) when the feature is enabled and user has enabled MFA.
	if s.licenseService.IsFeatureEnabled(base.Feature2FA) == nil && !mfaSecondLogin && userMFAEnabled {
		mfaTempToken, err := auth.GenerateMFATempToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret, tokenDuration)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate MFA temp token")
		}
		return &v1pb.LoginResponse{
			MfaTempToken: &mfaTempToken,
		}, nil
	}

	switch loginUser.Type {
	case base.EndUser:
		token, err := auth.GenerateAccessToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret, tokenDuration)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate API access token")
		}
		response.Token = token
	case base.ServiceAccount:
		token, err := auth.GenerateAPIToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate API access token")
		}
		response.Token = token
	default:
		return nil, status.Errorf(codes.Unauthenticated, "user type %s cannot login", loginUser.Type)
	}

	if request.Web {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "failed to parse metadata from incoming context")
		}
		// Pass the request origin header to response.
		var origin string
		for _, v := range md.Get("grpcgateway-origin") {
			origin = v
		}
		if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
			auth.GatewayMetadataAccessTokenKey:   response.Token,
			auth.GatewayMetadataUserIDKey:        fmt.Sprintf("%d", loginUser.ID),
			auth.GatewayMetadataRequestOriginKey: origin,
		})); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to set grpc header, error: %v", err)
		}
	}

	if _, err := s.store.UpdateUser(ctx, loginUser, &store.UpdateUserMessage{
		Profile: &storepb.UserProfile{
			LastLoginTime:          timestamppb.Now(),
			LastChangePasswordTime: loginUser.Profile.GetLastChangePasswordTime(),
		},
	}); err != nil {
		slog.Error("failed to update user profile", log.BBError(err), slog.String("user", loginUser.Email))
	}

	response.User = convertToUser(loginUser)

	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricapi.PrincipalLoginMetricName,
		Value: 1,
		Labels: map[string]any{
			"email": loginUser.Email,
		},
	})
	return response, nil
}

func (s *AuthService) needResetPassword(ctx context.Context, user *store.UserMessage) bool {
	// Reset password restriction only works for end user with email & password login.
	if user.Type != base.EndUser {
		return false
	}
	if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
		return false
	}

	passwordRestriction, err := s.store.GetPasswordRestrictionSetting(ctx)
	if err != nil {
		slog.Error("failed to get password restriction", log.BBError(err))
		return false
	}

	if user.Profile.LastLoginTime == nil {
		if !passwordRestriction.RequireResetPasswordForFirstLogin {
			return false
		}
		count, err := s.store.CountUsers(ctx, base.EndUser)
		if err != nil {
			slog.Error("failed to count end users", log.BBError(err))
			return false
		}
		// The 1st workspace admin login don't need to reset the password
		return count > 1
	}

	if passwordRestriction.PasswordRotation != nil && passwordRestriction.PasswordRotation.GetNanos() > 0 {
		lastChangePasswordTime := user.CreatedAt
		if user.Profile.LastChangePasswordTime != nil {
			lastChangePasswordTime = user.Profile.LastChangePasswordTime.AsTime()
		}
		if lastChangePasswordTime.Add(time.Duration(passwordRestriction.PasswordRotation.GetNanos())).Before(time.Now()) {
			return true
		}
	}

	return false
}

// Logout is the auth logout method.
func (s *AuthService) Logout(ctx context.Context, _ *v1pb.LogoutRequest) (*emptypb.Empty, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to parse metadata from incoming context")
	}
	accessTokenStr, err := auth.GetTokenFromMetadata(md)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	s.stateCfg.ExpireCache.Add(accessTokenStr, true)

	if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
		auth.GatewayMetadataAccessTokenKey: "",
		auth.GatewayMetadataUserIDKey:      "",
	})); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set grpc header, error: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *AuthService) getAndVerifyUser(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	user, err := s.store.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user by email %q: %v", request.Email, err)
	}
	if user == nil {
		return nil, invalidUserOrPasswordError
	}
	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		// If the two passwords don't match, return a 401 status.
		return nil, invalidUserOrPasswordError
	}
	return user, nil
}

func (s *AuthService) getOrCreateUserWithIDP(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	idpID, err := common.GetIdentityProviderID(request.IdpName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get identity provider ID: %v", err)
	}
	idp, err := s.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &idpID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get identity provider: %v", err)
	}
	if idp == nil {
		return nil, status.Errorf(codes.NotFound, "identity provider not found")
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace setting: %v", err)
	}

	var userInfo *storepb.IdentityProviderUserInfo
	switch idp.Type {
	case storepb.IdentityProviderType_OAUTH2:
		oauth2Context := request.IdpContext.GetOauth2Context()
		if oauth2Context == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing OAuth2 context")
		}
		oauth2IdentityProvider, err := oauth2.NewIdentityProvider(idp.Config.GetOauth2Config())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create new OAuth2 identity provider: %v", err)
		}
		redirectURL := fmt.Sprintf("%s/oauth/callback", setting.ExternalUrl)
		token, err := oauth2IdentityProvider.ExchangeToken(ctx, redirectURL, oauth2Context.Code)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to exchange token: %v", err)
		}
		userInfo, err = oauth2IdentityProvider.UserInfo(token)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get user info: %v", err)
		}
	case storepb.IdentityProviderType_OIDC:
		oauth2Context := request.IdpContext.GetOauth2Context()
		if oauth2Context == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing OAuth2 context")
		}

		oidcIDP, err := oidc.NewIdentityProvider(ctx, idp.Config.GetOidcConfig())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create new OIDC identity provider: %v", err)
		}

		redirectURL := fmt.Sprintf("%s/oidc/callback", setting.ExternalUrl)
		token, err := oidcIDP.ExchangeToken(ctx, redirectURL, oauth2Context.Code)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to exchange token: %v", err)
		}

		userInfo, err = oidcIDP.UserInfo(ctx, token, "")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get user info: %v", err)
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
				SecurityProtocol: ldap.SecurityProtocol(idpConfig.SecurityProtocol),
				FieldMapping:     idpConfig.FieldMapping,
			},
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create new LDAP identity provider: %v", err)
		}

		userInfo, err = ldapIDP.Authenticate(request.Email, request.Password)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get user info: %v", err)
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "identity provider type %s not supported", idp.Type.String())
	}
	if userInfo == nil {
		return nil, status.Errorf(codes.NotFound, "identity provider user info not found")
	}

	// The userinfo's email comes from identity provider, it has to be converted to lower-case.
	email := strings.ToLower(userInfo.Identifier)
	if err := validateEmail(email); err != nil {
		// If the email is invalid, we will try to use the domain and identifier to construct the email.
		domain := extractDomain(idp.Domain)
		if domain != "" {
			email = strings.ToLower(fmt.Sprintf("%s@%s", email, domain))
		} else {
			return nil, status.Errorf(codes.InvalidArgument, "invalid email %q, error: %v", userInfo.Identifier, err)
		}
	}
	// If the email is still invalid, we will return an error.
	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, email, false /* isServiceAccount */, false); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users by email %s: %v", email, err)
	}
	if user != nil {
		if user.MemberDeleted {
			// Undelete the user when login via SSO.
			user, err = s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &undeletePatch})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to undelete user: %v", err)
			}
		}
		if userInfo.HasGroups {
			// Sync user groups with the identity provider.
			// The userInfo.Groups is the groups that the user belongs to in the identity provider.
			if err := s.syncUserGroups(ctx, user, userInfo.Groups); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to sync user groups: %v", err)
			}
		}
		return user, nil
	}

	if err := s.licenseService.IsFeatureEnabled(base.FeatureSSO); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	// Create new user from identity provider.
	password, err := common.RandomString(20)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate random password")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate password hash")
	}
	newUser, err := s.store.CreateUser(ctx, &store.UserMessage{
		Name:         userInfo.DisplayName,
		Email:        email,
		Phone:        userInfo.Phone,
		Type:         base.EndUser,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user, error: %v", err)
	}
	if userInfo.HasGroups {
		// Sync user groups with the identity provider.
		// The userInfo.Groups is the groups that the user belongs to in the identity provider.
		if err := s.syncUserGroups(ctx, newUser, userInfo.Groups); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to sync user groups: %v", err)
		}
	}
	return newUser, nil
}

func challengeMFACode(user *store.UserMessage, mfaCode string) error {
	if !validateWithCodeAndSecret(mfaCode, user.MFAConfig.OtpSecret) {
		return status.Errorf(codes.Unauthenticated, "invalid MFA code")
	}
	return nil
}

func (s *AuthService) challengeRecoveryCode(ctx context.Context, user *store.UserMessage, recoveryCode string) error {
	for i, code := range user.MFAConfig.RecoveryCodes {
		if code == recoveryCode {
			// If the recovery code is valid, delete it from the user's recovery code list.
			user.MFAConfig.RecoveryCodes = slices.Delete(user.MFAConfig.RecoveryCodes, i, i+1)
			_, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
				MFAConfig: &storepb.MFAConfig{
					OtpSecret:     user.MFAConfig.OtpSecret,
					RecoveryCodes: user.MFAConfig.RecoveryCodes,
				},
			})
			if err != nil {
				return status.Errorf(codes.Internal, "failed to update user: %v", err)
			}
			return nil
		}
	}
	return status.Errorf(codes.Unauthenticated, "invalid recovery code")
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
		return status.Errorf(codes.Internal, "failed to list groups: %v", err)
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
			if _, err := s.store.UpdateGroup(ctx, bbGroup.Email, &store.UpdateGroupMessage{
				Payload: bbGroup.Payload,
			}); err != nil {
				return status.Errorf(codes.Internal, "failed to update group %q: %v", bbGroup.Email, err)
			}
			groupChanged = true
		}
	}

	// Reload IAM cache if group membership changed.
	if groupChanged {
		if err := s.iamManager.ReloadCache(ctx); err != nil {
			return status.Errorf(codes.Internal, "failed to reload IAM cache: %v", err)
		}
	}

	return nil
}
