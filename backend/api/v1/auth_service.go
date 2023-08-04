package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/nyaruka/phonenumbers"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/idp/oauth2"
	"github.com/bytebase/bytebase/backend/plugin/idp/oidc"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// AuthService implements the auth service.
type AuthService struct {
	v1pb.UnimplementedAuthServiceServer
	store                *store.Store
	secret               string
	refreshTokenDuration time.Duration
	licenseService       enterpriseAPI.LicenseService
	metricReporter       *metricreport.Reporter
	profile              *config.Profile
	postCreateUser       func(ctx context.Context, user *store.UserMessage, firstEndUser bool) error
}

// NewAuthService creates a new AuthService.
func NewAuthService(store *store.Store, secret string, refreshTokenDuration time.Duration, licenseService enterpriseAPI.LicenseService, metricReporter *metricreport.Reporter, profile *config.Profile, postCreateUser func(ctx context.Context, user *store.UserMessage, firstEndUser bool) error) *AuthService {
	return &AuthService{
		store:                store,
		secret:               secret,
		refreshTokenDuration: refreshTokenDuration,
		licenseService:       licenseService,
		metricReporter:       metricReporter,
		profile:              profile,
		postCreateUser:       postCreateUser,
	}
}

// GetUser gets a user.
func (s *AuthService) GetUser(ctx context.Context, request *v1pb.GetUserRequest) (*v1pb.User, error) {
	userID, err := common.GetUserID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
	}
	return convertToUser(user), nil
}

// ListUsers lists all users.
func (s *AuthService) ListUsers(ctx context.Context, request *v1pb.ListUsersRequest) (*v1pb.ListUsersResponse, error) {
	users, err := s.store.ListUsers(ctx, &store.FindUserMessage{ShowDeleted: request.ShowDeleted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user, error: %v", err)
	}
	response := &v1pb.ListUsersResponse{}
	for _, user := range users {
		response.Users = append(response.Users, convertToUser(user))
	}
	return response, nil
}

// CreateUser creates a user.
func (s *AuthService) CreateUser(ctx context.Context, request *v1pb.CreateUserRequest) (*v1pb.User, error) {
	if err := s.userCountGuard(ctx); err != nil {
		return nil, err
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace setting, error: %v", err)
	}

	if setting.DisallowSignup {
		rolePtr := ctx.Value(common.RoleContextKey)
		if rolePtr == nil || rolePtr.(api.Role) != api.Owner {
			return nil, status.Errorf(codes.PermissionDenied, "sign up is disallowed")
		}
	}
	if request.User == nil {
		return nil, status.Errorf(codes.InvalidArgument, "user must be set")
	}
	if request.User.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email must be set")
	}
	if request.User.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user title must be set")
	}

	principalType, err := convertToPrincipalType(request.User.UserType)
	if err != nil {
		return nil, err
	}
	if request.User.UserType != v1pb.UserType_SERVICE_ACCOUNT && request.User.UserType != v1pb.UserType_USER {
		return nil, status.Errorf(codes.InvalidArgument, "support user and service account only")
	}
	if request.User.UserType != v1pb.UserType_SERVICE_ACCOUNT && request.User.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password must be set")
	}

	count, err := s.store.CountUsers(ctx, api.EndUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count users, error: %v", err)
	}
	firstEndUser := count == 0

	if request.User.Phone != "" {
		if err := validatePhone(request.User.Phone); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid phone %q, error: %v", request.User.Phone, err)
		}
	}

	if err := validateEmail(request.User.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email %q, error: %v", request.User.Email, err)
	}
	existingUser, err := s.store.GetUser(ctx, &store.FindUserMessage{
		Email:       &request.User.Email,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user by email, error: %v", err)
	}
	if existingUser != nil {
		return nil, status.Errorf(codes.AlreadyExists, "email %s is already existed", request.User.Email)
	}

	password := request.User.Password
	if request.User.UserType == v1pb.UserType_SERVICE_ACCOUNT {
		pwd, err := common.RandomString(20)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate access key for service account.")
		}
		password = fmt.Sprintf("%s%s", api.ServiceAccountAccessKeyPrefix, pwd)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate password hash, error: %v", err)
	}
	userMessage := &store.UserMessage{
		Email:        request.User.Email,
		Name:         request.User.Title,
		Phone:        request.User.Phone,
		Type:         principalType,
		PasswordHash: string(passwordHash),
	}
	if request.User.UserRole != v1pb.UserRole_USER_ROLE_UNSPECIFIED {
		rolePtr := ctx.Value(common.RoleContextKey)
		// Allow workspace owner to create user with role.
		if rolePtr != nil && rolePtr.(api.Role) == api.Owner {
			userRole := convertUserRole(request.User.UserRole)
			if userRole == api.UnknownRole {
				return nil, status.Errorf(codes.InvalidArgument, "invalid user role %s", request.User.UserRole)
			}
			userMessage.Role = userRole
		} else {
			return nil, status.Errorf(codes.PermissionDenied, "only workspace owner can create user with role")
		}
	}

	user, err := s.store.CreateUser(ctx, userMessage, api.SystemBotID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user, error: %v", err)
	}

	if err := s.postCreateUser(ctx, user, firstEndUser); err != nil {
		return nil, err
	}

	isFirstUser := user.ID == api.PrincipalIDForFirstUser
	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricAPI.PrincipalRegistrationMetricName,
		Value: 1,
		Labels: map[string]any{
			"email": user.Email,
			"name":  user.Name,
			"phone": user.Phone,
			// We only send lark notification for the first principal registration.
			// false means do not notify upfront. Later the notification will be triggered by the scheduler.
			"lark_notified": !isFirstUser,
		},
	})
	bytes, err := json.Marshal(api.ActivityMemberCreatePayload{
		PrincipalID:    user.ID,
		PrincipalName:  user.Name,
		PrincipalEmail: user.Email,
		MemberStatus:   api.Active,
		Role:           user.Role,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to construct activity payload, error: %v", err)
	}
	activityCreate := &store.ActivityMessage{
		CreatorUID:   user.ID,
		ContainerUID: user.ID,
		Type:         api.ActivityMemberCreate,
		Level:        api.ActivityInfo,
		Payload:      string(bytes),
	}
	if _, err := s.store.CreateActivityV2(ctx, activityCreate); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create activity, error: %v", err)
	}
	userResponse := convertToUser(user)
	if request.User.UserType == v1pb.UserType_SERVICE_ACCOUNT {
		userResponse.ServiceKey = password
	}
	return userResponse, nil
}

// UpdateUser updates a user.
func (s *AuthService) UpdateUser(ctx context.Context, request *v1pb.UpdateUserRequest) (*v1pb.User, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.User == nil {
		return nil, status.Errorf(codes.InvalidArgument, "user must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	userID, err := common.GetUserID(request.User.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
	}
	if user.MemberDeleted {
		return nil, status.Errorf(codes.NotFound, "user %q has been deleted", userID)
	}

	role := ctx.Value(common.RoleContextKey).(api.Role)
	if principalID != userID && role != api.Owner {
		return nil, status.Errorf(codes.PermissionDenied, "only workspace owner or user itself can update the user %d", userID)
	}

	var passwordPatch *string
	patch := &store.UpdateUserMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "email":
			if err := validateEmail(request.User.Email); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid email %q format: %v", request.User.Email, err)
			}
			users, err := s.store.ListUsers(ctx, &store.FindUserMessage{
				Email:       &request.User.Email,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find user list, error: %v", err)
			}
			if len(users) != 0 {
				return nil, status.Errorf(codes.AlreadyExists, "email %s is already existed", request.User.Email)
			}
			patch.Email = &request.User.Email
		case "title":
			patch.Name = &request.User.Title
		case "password":
			if user.Type != api.EndUser {
				return nil, status.Errorf(codes.InvalidArgument, "password can be mutated for end users only")
			}
			passwordPatch = &request.User.Password
		case "service_key":
			if user.Type != api.ServiceAccount {
				return nil, status.Errorf(codes.InvalidArgument, "service key can be mutated for service accounts only")
			}
			val, err := common.RandomString(20)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to generate access key for service account.")
			}
			password := fmt.Sprintf("%s%s", api.ServiceAccountAccessKeyPrefix, val)
			passwordPatch = &password
		case "role":
			if role != api.Owner {
				return nil, status.Errorf(codes.PermissionDenied, "only workspace owner can update user role")
			}
			userRole := convertUserRole(request.User.UserRole)
			if userRole == api.UnknownRole {
				return nil, status.Errorf(codes.InvalidArgument, "invalid user role %s", request.User.UserRole)
			}
			patch.Role = &userRole
		case "mfa_enabled":
			if request.User.MfaEnabled {
				if user.MFAConfig.TempOtpSecret == "" || len(user.MFAConfig.TempRecoveryCodes) == 0 {
					return nil, status.Errorf(codes.InvalidArgument, "MFA is not setup yet")
				}
				patch.MFAConfig = &storepb.MFAConfig{
					OtpSecret:     user.MFAConfig.TempOtpSecret,
					RecoveryCodes: user.MFAConfig.TempRecoveryCodes,
				}
			} else {
				setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to find workspace setting, error: %v", err)
				}
				if setting.Require_2Fa {
					return nil, status.Errorf(codes.InvalidArgument, "2FA is required and cannot be disabled")
				}
				patch.MFAConfig = &storepb.MFAConfig{}
			}
		case "phone":
			if request.User.Phone != "" {
				if err := validatePhone(request.User.Phone); err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid phone number %q, error: %v", request.User.Phone, err)
				}
			}
			patch.Phone = &request.User.Phone
		}
	}
	if passwordPatch != nil {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.User.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate password hash, error: %v", err)
		}
		passwordHashStr := string(passwordHash)
		patch.PasswordHash = &passwordHashStr
	}
	// This flag is mainly used for validating OTP code when user setup MFA.
	// We only validate OTP code but not update user.
	if request.OtpCode != nil {
		isValid := validateWithCodeAndSecret(*request.OtpCode, user.MFAConfig.TempOtpSecret)
		if !isValid {
			return nil, status.Errorf(codes.InvalidArgument, "invalid OTP code")
		}
	}
	// This flag will regenerate temp secret and temp recovery codes.
	// It will be used when user setup MFA and regenerating recovery codes.
	if request.RegenerateTempMfaSecret {
		tempSecret, err := generateRandSecret(user.Email)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate MFA secret, error: %v", err)
		}
		tempRecoveryCodes, err := generateRecoveryCodes(10)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate recovery codes, error: %v", err)
		}
		patch.MFAConfig = &storepb.MFAConfig{
			TempOtpSecret:     tempSecret,
			TempRecoveryCodes: tempRecoveryCodes,
		}
		if user.MFAConfig != nil {
			patch.MFAConfig.OtpSecret = user.MFAConfig.OtpSecret
			patch.MFAConfig.RecoveryCodes = user.MFAConfig.RecoveryCodes
		}
	}
	// This flag will update user's recovery codes with temp recovery codes.
	// It will be used when user regenerate recovery codes after two phase commit.
	if request.RegenerateRecoveryCodes {
		if user.MFAConfig.OtpSecret == "" {
			return nil, status.Errorf(codes.InvalidArgument, "MFA is not enabled")
		}
		if len(user.MFAConfig.TempRecoveryCodes) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "No recovery codes to update")
		}
		patch.MFAConfig = &storepb.MFAConfig{
			OtpSecret:     user.MFAConfig.OtpSecret,
			RecoveryCodes: user.MFAConfig.TempRecoveryCodes,
		}
	}

	user, err = s.store.UpdateUser(ctx, userID, patch, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user, error: %v", err)
	}

	userResponse := convertToUser(user)
	if request.User.UserType == v1pb.UserType_SERVICE_ACCOUNT && passwordPatch != nil {
		userResponse.ServiceKey = *passwordPatch
	}
	return userResponse, nil
}

// DeleteUser deletes a user.
func (s *AuthService) DeleteUser(ctx context.Context, request *v1pb.DeleteUserRequest) (*emptypb.Empty, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	userID, err := common.GetUserID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
	}
	if user.MemberDeleted {
		return nil, status.Errorf(codes.NotFound, "user %q has been deleted", userID)
	}

	role := ctx.Value(common.RoleContextKey).(api.Role)
	if role != api.Owner {
		return nil, status.Errorf(codes.PermissionDenied, "only workspace owner can delete the user %d", userID)
	}

	if _, err := s.store.UpdateUser(ctx, userID, &store.UpdateUserMessage{Delete: &deletePatch}, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// UndeleteUser undeletes a user.
func (s *AuthService) UndeleteUser(ctx context.Context, request *v1pb.UndeleteUserRequest) (*v1pb.User, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	userID, err := common.GetUserID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
	}
	if !user.MemberDeleted {
		return nil, status.Errorf(codes.InvalidArgument, "user %q is already active", userID)
	}

	role := ctx.Value(common.RoleContextKey).(api.Role)
	if role != api.Owner {
		return nil, status.Errorf(codes.PermissionDenied, "only workspace owner can undelete the user %d", userID)
	}

	user, err = s.store.UpdateUser(ctx, userID, &store.UpdateUserMessage{Delete: &undeletePatch}, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToUser(user), nil
}

func convertToUser(user *store.UserMessage) *v1pb.User {
	role := v1pb.UserRole_USER_ROLE_UNSPECIFIED
	switch user.Role {
	case api.Owner:
		role = v1pb.UserRole_OWNER
	case api.DBA:
		role = v1pb.UserRole_DBA
	case api.Developer:
		role = v1pb.UserRole_DEVELOPER
	}
	userType := v1pb.UserType_USER_TYPE_UNSPECIFIED
	switch user.Type {
	case api.EndUser:
		userType = v1pb.UserType_USER
	case api.SystemBot:
		userType = v1pb.UserType_SYSTEM_BOT
	case api.ServiceAccount:
		userType = v1pb.UserType_SERVICE_ACCOUNT
	}

	convertedUser := &v1pb.User{
		Name:     fmt.Sprintf("%s%d", common.UserNamePrefix, user.ID),
		State:    convertDeletedToState(user.MemberDeleted),
		Email:    user.Email,
		Phone:    user.Phone,
		Title:    user.Name,
		UserType: userType,
		UserRole: role,
	}
	if user.MFAConfig != nil {
		convertedUser.MfaEnabled = user.MFAConfig.OtpSecret != ""
		convertedUser.MfaSecret = user.MFAConfig.TempOtpSecret
		convertedUser.RecoveryCodes = user.MFAConfig.TempRecoveryCodes
	}
	return convertedUser
}

func convertToPrincipalType(userType v1pb.UserType) (api.PrincipalType, error) {
	var t api.PrincipalType
	switch userType {
	case v1pb.UserType_USER:
		t = api.EndUser
	case v1pb.UserType_SYSTEM_BOT:
		t = api.SystemBot
	case v1pb.UserType_SERVICE_ACCOUNT:
		t = api.ServiceAccount
	default:
		return t, status.Errorf(codes.InvalidArgument, "invalid user type %s", userType)
	}
	return t, nil
}

func convertUserRole(userRole v1pb.UserRole) api.Role {
	switch userRole {
	case v1pb.UserRole_OWNER:
		return api.Owner
	case v1pb.UserRole_DBA:
		return api.DBA
	case v1pb.UserRole_DEVELOPER:
		return api.Developer
	}
	return api.UnknownRole
}

// Login is the auth login method including SSO.
func (s *AuthService) Login(ctx context.Context, request *v1pb.LoginRequest) (*v1pb.LoginResponse, error) {
	var loginUser *store.UserMessage
	mfaSecondLogin := false
	if request.MfaTempToken != nil && *request.MfaTempToken != "" {
		mfaSecondLogin = true
	}

	if !mfaSecondLogin {
		var err error
		if request.IdpName == "" {
			loginUser, err = s.getAndVerifyUser(ctx, request)
		} else {
			loginUser, err = s.getOrCreateUserWithIDP(ctx, request)
		}
		if err != nil {
			return nil, err
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
			return nil, status.Errorf(codes.Unauthenticated, "user not found")
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

	if loginUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "login user not found")
	}
	if loginUser.MemberDeleted {
		return nil, status.Errorf(codes.Unauthenticated, "user has been deactivated by administrators")
	}

	userMFAEnabled := loginUser.MFAConfig != nil && loginUser.MFAConfig.OtpSecret != ""
	// We only allow MFA login (2-step) when the feature is enabled and user has enabled MFA.
	if s.licenseService.IsFeatureEnabled(api.Feature2FA) == nil && !mfaSecondLogin && userMFAEnabled {
		mfaTempToken, err := auth.GenerateMFATempToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate MFA temp token")
		}
		return &v1pb.LoginResponse{
			MfaTempToken: &mfaTempToken,
		}, nil
	}

	var accessToken, refreshToken string
	if loginUser.Type == api.EndUser {
		token, err := auth.GenerateAccessToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate API access token")
		}
		accessToken = token
		if request.Web {
			refreshToken, err = auth.GenerateRefreshToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret, s.refreshTokenDuration)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to generate API access token")
			}
		}
	} else if loginUser.Type == api.ServiceAccount {
		token, err := auth.GenerateAPIToken(loginUser.Name, loginUser.ID, s.profile.Mode, s.secret)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate API access token")
		}
		accessToken = token
	} else {
		return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("user type %s cannot login", loginUser.Type))
	}

	if request.Web {
		if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
			auth.GatewayMetadataAccessTokenKey:  accessToken,
			auth.GatewayMetadataRefreshTokenKey: refreshToken,
			auth.GatewayMetadataUserIDKey:       fmt.Sprintf("%d", loginUser.ID),
		})); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to set grpc header, error: %v", err)
		}
	}

	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricAPI.PrincipalLoginMetricName,
		Value: 1,
		Labels: map[string]any{
			"email": loginUser.Email,
		},
	})
	return &v1pb.LoginResponse{
		Token: accessToken,
	}, nil
}

// Logout is the auth logout method.
func (*AuthService) Logout(ctx context.Context, _ *v1pb.LogoutRequest) (*emptypb.Empty, error) {
	if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
		auth.GatewayMetadataAccessTokenKey:  "",
		auth.GatewayMetadataRefreshTokenKey: "",
		auth.GatewayMetadataUserIDKey:       "",
	})); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set grpc header, error: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *AuthService) getAndVerifyUser(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	user, err := s.store.GetUser(ctx, &store.FindUserMessage{
		Email:       &request.Email,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user by email %q: %v", request.Email, err)
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user %q not found", request.Email)
	}
	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		// If the two passwords don't match, return a 401 status.
		return nil, status.Errorf(codes.Unauthenticated, "incorrect password")
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
	if idp.Type == storepb.IdentityProviderType_OAUTH2 {
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
	} else if idp.Type == storepb.IdentityProviderType_OIDC {
		oauth2Context := request.IdpContext.GetOauth2Context()
		if oauth2Context == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing OAuth2 context")
		}

		oidcIDP, err := oidc.NewIdentityProvider(
			ctx,
			oidc.IdentityProviderConfig{
				Issuer:        idp.Config.GetOidcConfig().Issuer,
				ClientID:      idp.Config.GetOidcConfig().ClientId,
				ClientSecret:  idp.Config.GetOidcConfig().ClientSecret,
				FieldMapping:  idp.Config.GetOidcConfig().FieldMapping,
				SkipTLSVerify: idp.Config.GetOidcConfig().SkipTlsVerify,
				AuthStyle:     idp.Config.GetOidcConfig().GetAuthStyle(),
			},
		)
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
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider type %s not supported", idp.Type.String())
	}
	if userInfo == nil {
		return nil, status.Errorf(codes.NotFound, "identity provider user info not found")
	}

	// The userinfo's email comes from identity provider, it has to be converted to lower-case.
	email := strings.ToLower(userInfo.Identifier)
	if err := validateEmail(email); err != nil {
		// If the email is invalid, we will try to use the domain and identifier to construct the email.
		if idp.Domain != "" {
			domain := extractDomain(idp.Domain)
			email = strings.ToLower(fmt.Sprintf("%s@%s", userInfo.Identifier, domain))
		}
	}
	if email == "" {
		return nil, status.Errorf(codes.NotFound, "unable to identify the user by provider user info")
	}
	users, err := s.store.ListUsers(ctx, &store.FindUserMessage{
		Email:       &email,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users by email %s: %v", email, err)
	}

	var user *store.UserMessage
	if len(users) == 0 {
		if err := s.licenseService.IsFeatureEnabled(api.FeatureSSO); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
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
			Type:         api.EndUser,
			PasswordHash: string(passwordHash),
		}, api.SystemBotID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create user, error: %v", err)
		}
		user = newUser
	} else {
		user = users[0]
	}

	return user, nil
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
			_, err := s.store.UpdateUser(ctx, user.ID, &store.UpdateUserMessage{
				MFAConfig: &storepb.MFAConfig{
					OtpSecret:     user.MFAConfig.OtpSecret,
					RecoveryCodes: user.MFAConfig.RecoveryCodes,
				},
			}, user.ID)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to update user: %v", err)
			}
			return nil
		}
	}
	return status.Errorf(codes.Unauthenticated, "invalid recovery code")
}

func validateEmail(email string) error {
	formattedEmail := strings.ToLower(email)
	if email != formattedEmail {
		return errors.New("email should be lowercase")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return err
	}
	return nil
}

func validatePhone(phone string) error {
	phoneNumber, err := phonenumbers.Parse(phone, "")
	if err != nil {
		return err
	}
	if !phonenumbers.IsValidNumber(phoneNumber) {
		return errors.New("invalid phone number")
	}
	return nil
}

func extractDomain(input string) string {
	pattern := `[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+`
	regExp, err := regexp.Compile(pattern)
	if err != nil {
		// WHen the pattern is invalid, we just return the input.
		return input
	}

	match := regExp.FindString(input)
	domainParts := strings.Split(match, ".")
	// If the domain has at least 3 parts, we will remove the first part.
	if len(domainParts) >= 3 {
		match = strings.Join(domainParts[1:], ".")
	}
	return match
}

const (
	// issuerName is the name of the issuer of the OTP token.
	issuerName = "Bytebase"
)

// generateRandSecret generates a random secret for the given account name.
func generateRandSecret(accountName string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuerName,
		AccountName: accountName,
	})
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}

// validateWithCodeAndSecret validates the given code against the given secret.
func validateWithCodeAndSecret(code, secret string) bool {
	return totp.Validate(code, secret)
}

// generateRecoveryCodes generates n recovery codes.
func generateRecoveryCodes(n int) ([]string, error) {
	recoveryCodes := make([]string, n)
	for i := 0; i < n; i++ {
		code, err := common.RandomString(10)
		if err != nil {
			return nil, err
		}
		recoveryCodes[i] = code
	}
	return recoveryCodes, nil
}

func (s *AuthService) userCountGuard(ctx context.Context) error {
	userLimit := s.licenseService.GetPlanLimitValue(enterpriseAPI.PlanLimitMaximumUser)

	count, err := s.store.CountPrincipal(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if int64(count) >= userLimit {
		return status.Errorf(codes.ResourceExhausted, "reached the maximum user count %d", userLimit)
	}

	return nil
}
