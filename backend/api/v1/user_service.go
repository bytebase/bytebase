package v1

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"

	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// UserService implements the user service.
type UserService struct {
	v1connect.UnimplementedUserServiceHandler
	store          *store.Store
	licenseService *enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewUserService creates a new UserService.
func NewUserService(store *store.Store, licenseService *enterprise.LicenseService, profile *config.Profile, iamManager *iam.Manager) *UserService {
	return &UserService{
		store:          store,
		licenseService: licenseService,
		profile:        profile,
		iamManager:     iamManager,
	}
}

// GetUser gets a user.
func (s *UserService) GetUser(ctx context.Context, request *connect.Request[v1pb.GetUserRequest]) (*connect.Response[v1pb.User], error) {
	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user"))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}
	return connect.NewResponse(convertToUser(ctx, s.iamManager, user)), nil
}

// BatchGetUsers get users in batch.
func (s *UserService) BatchGetUsers(ctx context.Context, request *connect.Request[v1pb.BatchGetUsersRequest]) (*connect.Response[v1pb.BatchGetUsersResponse], error) {
	response := &v1pb.BatchGetUsersResponse{}
	for _, name := range request.Msg.Names {
		user, err := s.GetUser(ctx, connect.NewRequest(&v1pb.GetUserRequest{Name: name}))
		if err != nil {
			return nil, err
		}
		response.Users = append(response.Users, user.Msg)
	}
	return connect.NewResponse(response), nil
}

// GetCurrentUser gets the current authenticated user.
func (s *UserService) GetCurrentUser(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[v1pb.User], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("authenticated user not found"))
	}
	return connect.NewResponse(convertToUser(ctx, s.iamManager, user)), nil
}

// ListUsers lists all users.
func (s *UserService) ListUsers(ctx context.Context, request *connect.Request[v1pb.ListUsersRequest]) (*connect.Response[v1pb.ListUsersResponse], error) {
	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.Msg.PageToken,
		limit:   int(request.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindUserMessage{
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		ShowDeleted: request.Msg.ShowDeleted,
	}
	filterResult, err := store.GetListUserFilter(request.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if filterResult != nil {
		find.FilterQ = filterResult.Query
		find.ProjectID = filterResult.ProjectID
	}
	if v := find.ProjectID; v != nil {
		user, ok := GetUserFromContext(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
		}
		hasPermission, err := s.iamManager.CheckPermission(ctx, iam.PermissionProjectsGet, user, *v)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check user permission"))
		}
		if !hasPermission {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionProjectsGet))
		}
	}

	users, err := s.store.ListUsers(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list user"))
	}

	nextPageToken := ""
	if len(users) == limitPlusOne {
		users = users[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	response := &v1pb.ListUsersResponse{
		NextPageToken: nextPageToken,
	}
	for _, user := range users {
		response.Users = append(response.Users, convertToUser(ctx, s.iamManager, user))
	}
	return connect.NewResponse(response), nil
}

// CreateUser creates a user.
func (s *UserService) CreateUser(ctx context.Context, request *connect.Request[v1pb.CreateUserRequest]) (*connect.Response[v1pb.User], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DISALLOW_SELF_SERVICE_SIGNUP); err == nil {
		setting, err := s.store.GetWorkspaceProfileSetting(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting"))
		}
		if setting.DisallowSignup || s.profile.SaaS {
			callerUser, ok := GetUserFromContext(ctx)
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("sign up is disallowed"))
			}
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersCreate, callerUser)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionUsersCreate))
			}
		}
	}
	if request.Msg.User == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("user must be set"))
	}
	if request.Msg.User.Email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email must be set"))
	}
	if request.Msg.User.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("user title must be set"))
	}

	principalType, err := convertToPrincipalType(request.Msg.User.UserType)
	if err != nil {
		return nil, err
	}
	if principalType != storepb.PrincipalType_SERVICE_ACCOUNT && principalType != storepb.PrincipalType_END_USER && principalType != storepb.PrincipalType_WORKLOAD_IDENTITY {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("support user, service account, and workload identity only"))
	}

	// Validate workload identity specific requirements
	if principalType == storepb.PrincipalType_WORKLOAD_IDENTITY {
		if !common.IsWorkloadIdentityEmail(request.Msg.User.Email) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("workload identity email must end with %s", common.WorkloadIdentityEmailSuffix))
		}
		if request.Msg.User.WorkloadIdentityConfig == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("workload_identity_config is required for workload identity"))
		}
	}

	if principalType == storepb.PrincipalType_END_USER {
		if err := s.userCountGuard(ctx); err != nil {
			return nil, err
		}
	}

	count, err := s.store.CountActiveEndUsers(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to count users"))
	}
	firstEndUser := count == 0

	if request.Msg.User.Phone != "" {
		if err := common.ValidatePhone(request.Msg.User.Phone); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid phone %q", request.Msg.User.Phone))
		}
	}

	// Skip domain restrictions for service accounts and workload identities
	skipDomainRestriction := principalType == storepb.PrincipalType_SERVICE_ACCOUNT || principalType == storepb.PrincipalType_WORKLOAD_IDENTITY
	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, request.Msg.User.Email, skipDomainRestriction, false); err != nil {
		return nil, err
	}
	existingUser, err := s.store.GetUserByEmail(ctx, request.Msg.User.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user by email"))
	}
	if existingUser != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("email %s exists", request.Msg.User.Email))
	}

	password := request.Msg.User.Password
	switch principalType {
	case storepb.PrincipalType_SERVICE_ACCOUNT:
		pwd, err := common.RandomString(20)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate access key for service account"))
		}
		password = fmt.Sprintf("%s%s", common.ServiceAccountAccessKeyPrefix, pwd)
	case storepb.PrincipalType_WORKLOAD_IDENTITY:
		// Workload identity uses OIDC tokens, not passwords
		// Generate a random unusable password for security
		pwd, err := common.RandomString(64)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate secure password for workload identity"))
		}
		password = pwd
	default:
		if password != "" {
			if err := s.validatePassword(ctx, password); err != nil {
				return nil, err
			}
		} else {
			pwd, err := common.RandomString(20)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate random password for user"))
			}
			password = pwd
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate password hash"))
	}

	// Build profile with workload identity config if applicable
	var profile *storepb.UserProfile
	if principalType == storepb.PrincipalType_WORKLOAD_IDENTITY {
		wic := request.Msg.User.WorkloadIdentityConfig
		profile = &storepb.UserProfile{
			WorkloadIdentityConfig: &storepb.WorkloadIdentityConfig{
				ProviderType:     storepb.WorkloadIdentityConfig_ProviderType(wic.ProviderType),
				IssuerUrl:        wic.IssuerUrl,
				AllowedAudiences: wic.AllowedAudiences,
				SubjectPattern:   wic.SubjectPattern,
			},
		}
	}

	userMessage := &store.UserMessage{
		Email:        request.Msg.User.Email,
		Name:         request.Msg.User.Title,
		Phone:        request.Msg.User.Phone,
		Type:         principalType,
		PasswordHash: string(passwordHash),
		Profile:      profile,
	}

	user, err := s.store.CreateUser(ctx, userMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create user"))
	}

	if firstEndUser {
		// The first end user should be workspace admin.
		updateRole := &store.PatchIamPolicyMessage{
			Member: common.FormatUserEmail(user.Email),
			Roles:  []string{common.FormatRole(common.WorkspaceAdmin)},
		}
		if _, err := s.store.PatchWorkspaceIamPolicy(ctx, updateRole); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	userResponse := convertToUser(ctx, s.iamManager, user)
	if request.Msg.User.UserType == v1pb.UserType_SERVICE_ACCOUNT {
		userResponse.ServiceKey = password
	}
	return connect.NewResponse(userResponse), nil
}

func (s *UserService) validatePassword(ctx context.Context, password string) error {
	setting, err := s.store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to get password restriction with error: %v", err))
	}
	passwordRestriction := setting.GetPasswordRestriction()

	if len(password) < int(passwordRestriction.GetMinLength()) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password length should no less than %v characters", passwordRestriction.GetMinLength()))
	}
	if passwordRestriction.GetRequireNumber() && !regexp.MustCompile("[0-9]+").MatchString(password) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must contains at least 1 number"))
	}
	if passwordRestriction.GetRequireLetter() && !regexp.MustCompile("[a-zA-Z]+").MatchString(password) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must contains at least 1 lower case letter"))
	}
	if passwordRestriction.GetRequireUppercaseLetter() && !regexp.MustCompile("[A-Z]+").MatchString(password) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must contains at least 1 upper case letter"))
	}
	if passwordRestriction.GetRequireSpecialCharacter() && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+`).MatchString(password) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must contains at least 1 special character"))
	}
	return nil
}

// UpdateUser updates a user.
func (s *UserService) UpdateUser(ctx context.Context, request *connect.Request[v1pb.UpdateUserRequest]) (*connect.Response[v1pb.User], error) {
	callerUser, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	if request.Msg.User == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("user must be set"))
	}
	if request.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
	}

	email, err := common.GetUserEmail(request.Msg.User.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user"))
	}
	if user == nil {
		if request.Msg.AllowMissing {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersCreate, callerUser)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionUsersCreate))
			}
			return s.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
				User: request.Msg.User,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q has been deleted", email))
	}

	if callerUser.ID != user.ID {
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersUpdate, callerUser)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
		}
		if !ok {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionUsersUpdate))
		}
	}

	var passwordPatch *string
	patch := &store.UpdateUserMessage{}
	for _, path := range request.Msg.UpdateMask.Paths {
		switch path {
		case "email":
			// Email updates are not supported through UpdateUser. Use UpdateEmail API instead.
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email updates are not supported through UpdateUser, use UpdateEmail API instead"))
		case "title":
			patch.Name = &request.Msg.User.Title
		case "password":
			if user.Type != storepb.PrincipalType_END_USER {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password can be mutated for end users only"))
			}
			if err := s.validatePassword(ctx, request.Msg.User.Password); err != nil {
				return nil, err
			}
			passwordPatch = &request.Msg.User.Password
		case "service_key":
			if user.Type != storepb.PrincipalType_SERVICE_ACCOUNT {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("service key can be mutated for service accounts only"))
			}
			val, err := common.RandomString(20)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate access key for service account"))
			}
			password := fmt.Sprintf("%s%s", common.ServiceAccountAccessKeyPrefix, val)
			passwordPatch = &password
		case "mfa_enabled":
			if request.Msg.User.MfaEnabled {
				if user.MFAConfig.TempOtpSecret == "" || len(user.MFAConfig.TempRecoveryCodes) == 0 {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA is not setup yet"))
				}
				if isMFATempSecretExpired(user.MFAConfig) {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA setup has expired, please regenerate the temporary secret"))
				}
				// Promote temp secrets to permanent and clear temp fields to prevent reuse
				patch.MFAConfig = &storepb.MFAConfig{
					OtpSecret:                user.MFAConfig.TempOtpSecret,
					RecoveryCodes:            user.MFAConfig.TempRecoveryCodes,
					TempOtpSecret:            "",
					TempRecoveryCodes:        nil,
					TempOtpSecretCreatedTime: nil,
				}
			} else {
				setting, err := s.store.GetWorkspaceProfileSetting(ctx)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting"))
				}
				if setting.Require_2Fa {
					isWorkspaceAdmin, err := isUserWorkspaceAdmin(ctx, s.store, callerUser)
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check user roles"))
					}
					// Allow workspace admin to disable 2FA even if it is required.
					if !isWorkspaceAdmin {
						return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("2FA is required and cannot be disabled"))
					}
				}
				patch.MFAConfig = &storepb.MFAConfig{}
			}
		case "phone":
			if request.Msg.User.Phone != "" {
				if err := common.ValidatePhone(request.Msg.User.Phone); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid phone number %q", request.Msg.User.Phone))
				}
			}
			patch.Phone = &request.Msg.User.Phone
		default:
		}
	}
	if passwordPatch != nil {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(*passwordPatch)); err == nil {
			// return bad request if the passwords match
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password cannot be the same"))
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte((*passwordPatch)), bcrypt.DefaultCost)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate password hash"))
		}
		passwordHashStr := string(passwordHash)
		patch.PasswordHash = &passwordHashStr

		// Revoke all refresh tokens for this user (including current session)
		// User must re-login after password change for security
		if err := s.store.DeleteWebRefreshTokensByUser(ctx, user.Email); err != nil {
			slog.Error("failed to revoke refresh tokens on password change", log.BBError(err), slog.String("user", user.Email))
		}
	}
	// This flag is mainly used for validating OTP code when user setup MFA.
	// We only validate OTP code but not update user.
	if request.Msg.OtpCode != nil {
		if isMFATempSecretExpired(user.MFAConfig) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA setup has expired, please regenerate the temporary secret"))
		}
		isValid := validateWithCodeAndSecret(*request.Msg.OtpCode, user.MFAConfig.TempOtpSecret)
		if !isValid {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid OTP code"))
		}
	}
	// This flag will regenerate temp secret and temp recovery codes.
	// It will be used when user setup MFA and regenerating recovery codes.
	if request.Msg.RegenerateTempMfaSecret {
		tempSecret, err := generateRandSecret(user.Email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate MFA secret"))
		}
		tempRecoveryCodes, err := generateRecoveryCodes(10)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate recovery codes"))
		}
		patch.MFAConfig = &storepb.MFAConfig{
			TempOtpSecret:            tempSecret,
			TempRecoveryCodes:        tempRecoveryCodes,
			TempOtpSecretCreatedTime: timestamppb.Now(),
		}
		if user.MFAConfig != nil {
			patch.MFAConfig.OtpSecret = user.MFAConfig.OtpSecret
			patch.MFAConfig.RecoveryCodes = user.MFAConfig.RecoveryCodes
		}
	}
	// This flag will update user's recovery codes with temp recovery codes.
	// It will be used when user regenerate recovery codes after two phase commit.
	if request.Msg.RegenerateRecoveryCodes {
		if user.MFAConfig.OtpSecret == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA is not enabled"))
		}
		if len(user.MFAConfig.TempRecoveryCodes) == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("No recovery codes to update"))
		}
		patch.MFAConfig = &storepb.MFAConfig{
			OtpSecret:     user.MFAConfig.OtpSecret,
			RecoveryCodes: user.MFAConfig.TempRecoveryCodes,
		}
	}

	user, err = s.store.UpdateUser(ctx, user, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}

	userResponse := convertToUser(ctx, s.iamManager, user)
	if request.Msg.User.UserType == v1pb.UserType_SERVICE_ACCOUNT && passwordPatch != nil {
		userResponse.ServiceKey = *passwordPatch
	}
	return connect.NewResponse(userResponse), nil
}

// DeleteUser deletes a user.
func (s *UserService) DeleteUser(ctx context.Context, request *connect.Request[v1pb.DeleteUserRequest]) (*connect.Response[emptypb.Empty], error) {
	callerUser, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersDelete, callerUser)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionUsersDelete))
	}

	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user"))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q has been deleted", email))
	}

	// Check if there is still workspace admin if the current user is deleted.
	policy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, err
	}
	hasExtraWorkspaceAdmin, err := s.hasExtraWorkspaceAdmin(ctx, policy.Policy, user)
	if err != nil {
		return nil, err
	}
	if !hasExtraWorkspaceAdmin {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("workspace must have at least one admin"))
	}

	if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &deletePatch}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *UserService) getActiveUserCount(ctx context.Context) (int, error) {
	userStat, err := s.store.StatUsers(ctx)
	if err != nil {
		return 0, connect.NewError(connect.CodeInternal, errors.Errorf("failed to stat users with error: %v", err.Error()))
	}
	activeEndUserCount := 0
	for _, stat := range userStat {
		if !stat.Deleted && stat.Type == storepb.PrincipalType_END_USER {
			activeEndUserCount = stat.Count
			break
		}
	}
	return activeEndUserCount, nil
}

func (s *UserService) hasExtraWorkspaceAdmin(ctx context.Context, policy *storepb.IamPolicy, user *store.UserMessage) (bool, error) {
	workspaceAdminRole := common.FormatRole(common.WorkspaceAdmin)
	userMember := common.FormatUserEmail(user.Email)

	for _, binding := range policy.GetBindings() {
		if binding.GetRole() != workspaceAdminRole {
			continue
		}
		for _, member := range binding.GetMembers() {
			if member == userMember {
				continue
			}
			if member == common.AllUsers {
				activeEndUserCount, err := s.getActiveUserCount(ctx)
				if err != nil {
					return false, err
				}
				return activeEndUserCount > 1, nil
			}
			users := utils.GetUsersByMember(ctx, s.store, member)
			for _, user := range users {
				if !user.MemberDeleted && user.Type == storepb.PrincipalType_END_USER {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// UndeleteUser undeletes a user.
func (s *UserService) UndeleteUser(ctx context.Context, request *connect.Request[v1pb.UndeleteUserRequest]) (*connect.Response[v1pb.User], error) {
	callerUser, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersUndelete, callerUser)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionUsersUndelete))
	}

	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user"))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}
	if !user.MemberDeleted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("user %q is already active", email))
	}
	if user.Type == storepb.PrincipalType_END_USER {
		if err := s.userCountGuard(ctx); err != nil {
			return nil, err
		}
	}

	user, err = s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &undeletePatch})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertToUser(ctx, s.iamManager, user)), nil
}

// UpdateEmail updates a user's email address.
func (s *UserService) UpdateEmail(ctx context.Context, request *connect.Request[v1pb.UpdateEmailRequest]) (*connect.Response[v1pb.User], error) {
	callerUser, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersUpdateEmail, callerUser)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionUsersUpdateEmail))
	}

	// Get user by email from the name field
	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user"))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user not found"))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user has been deleted"))
	}

	// Check if new email is the same as current email (no-op)
	if user.Email == request.Msg.Email {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("new email is the same as current email"))
	}

	// Validate email format and domain restrictions
	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, request.Msg.Email, user.Type == storepb.PrincipalType_SERVICE_ACCOUNT, false); err != nil {
		return nil, err
	}

	// Check if email already exists
	existedUser, err := s.store.GetUserByEmail(ctx, request.Msg.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user by email"))
	}
	if existedUser != nil && existedUser.ID != user.ID {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("email %s already exists", request.Msg.Email))
	}

	// Update the email
	user, err = s.store.UpdateUserEmail(ctx, user, request.Msg.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user email"))
	}

	return connect.NewResponse(convertToUser(ctx, s.iamManager, user)), nil
}

func convertToV1UserType(userType storepb.PrincipalType) v1pb.UserType {
	switch userType {
	case storepb.PrincipalType_END_USER:
		return v1pb.UserType_USER
	case storepb.PrincipalType_SYSTEM_BOT:
		return v1pb.UserType_SYSTEM_BOT
	case storepb.PrincipalType_SERVICE_ACCOUNT:
		return v1pb.UserType_SERVICE_ACCOUNT
	case storepb.PrincipalType_WORKLOAD_IDENTITY:
		return v1pb.UserType_WORKLOAD_IDENTITY
	default:
		return v1pb.UserType_USER_TYPE_UNSPECIFIED
	}
}

func convertToUser(ctx context.Context, iamManager *iam.Manager, user *store.UserMessage) *v1pb.User {
	convertedUser := &v1pb.User{
		Name:     common.FormatUserEmail(user.Email),
		State:    convertDeletedToState(user.MemberDeleted),
		Email:    user.Email,
		Phone:    user.Phone,
		Title:    user.Name,
		UserType: convertToV1UserType(user.Type),
		Profile: &v1pb.User_Profile{
			LastLoginTime:          user.Profile.LastLoginTime,
			LastChangePasswordTime: user.Profile.LastChangePasswordTime,
			Source:                 user.Profile.Source,
		},
		Groups: iamManager.GetUserGroups(user.Email),
	}

	// Add workload identity config if present
	if user.Profile != nil && user.Profile.WorkloadIdentityConfig != nil {
		wic := user.Profile.WorkloadIdentityConfig
		convertedUser.WorkloadIdentityConfig = &v1pb.WorkloadIdentityConfig{
			ProviderType:     v1pb.WorkloadIdentityConfig_ProviderType(wic.ProviderType),
			IssuerUrl:        wic.IssuerUrl,
			AllowedAudiences: wic.AllowedAudiences,
			SubjectPattern:   wic.SubjectPattern,
		}
	}

	if user.MFAConfig != nil {
		convertedUser.MfaEnabled = user.MFAConfig.OtpSecret != ""
		// Only expose temporary MFA secrets and recovery codes to the user themselves
		if currentUser, ok := GetUserFromContext(ctx); ok && currentUser.ID == user.ID {
			convertedUser.TempOtpSecret = user.MFAConfig.TempOtpSecret
			convertedUser.TempRecoveryCodes = user.MFAConfig.TempRecoveryCodes
			convertedUser.TempOtpSecretCreatedTime = user.MFAConfig.TempOtpSecretCreatedTime
		}
	}
	return convertedUser
}

func convertToPrincipalType(userType v1pb.UserType) (storepb.PrincipalType, error) {
	var t storepb.PrincipalType
	switch userType {
	case v1pb.UserType_USER:
		t = storepb.PrincipalType_END_USER
	case v1pb.UserType_SYSTEM_BOT:
		t = storepb.PrincipalType_SYSTEM_BOT
	case v1pb.UserType_SERVICE_ACCOUNT:
		t = storepb.PrincipalType_SERVICE_ACCOUNT
	case v1pb.UserType_WORKLOAD_IDENTITY:
		t = storepb.PrincipalType_WORKLOAD_IDENTITY
	default:
		return t, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid user type %s", userType))
	}
	return t, nil
}

func validateEmailWithDomains(ctx context.Context, licenseService *enterprise.LicenseService, stores *store.Store, email string, isServiceAccount bool, checkDomainSetting bool) error {
	if licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_USER_EMAIL_DOMAIN_RESTRICTION) != nil {
		// nolint:nilerr
		// feature not enabled, only validate email and skip domain restriction.
		if err := validateEmail(email); err != nil {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid email: %v", err.Error()))
		}
		return nil
	}
	setting, err := stores.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting"))
	}

	var allowedDomains []string
	if checkDomainSetting || setting.EnforceIdentityDomain {
		allowedDomains = setting.Domains
	}

	// Check if the email is valid.
	if err := validateEmail(email); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid email: %v", err.Error()))
	}
	// Domain restrictions are not applied to service account.
	if isServiceAccount {
		return nil
	}
	// Enforce domain restrictions.
	if len(allowedDomains) > 0 {
		ok := false
		for _, v := range allowedDomains {
			if strings.HasSuffix(email, fmt.Sprintf("@%s", v)) {
				ok = true
				break
			}
		}
		if !ok {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email %q does not belong to domains %v", email, allowedDomains))
		}
	}
	return nil
}

func validateEmail(email string) error {
	if email != strings.ToLower(email) {
		return errors.New("email should be lowercase")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return err
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
	// mfaTempSecretExpiration is the duration after which temporary MFA secrets expire.
	// Industry standard is 2-5 minutes for temporary MFA verification tokens.
	mfaTempSecretExpiration = 5 * time.Minute
)

// isMFATempSecretExpired checks if the temporary MFA secret has expired.
func isMFATempSecretExpired(mfaConfig *storepb.MFAConfig) bool {
	if mfaConfig == nil || mfaConfig.TempOtpSecretCreatedTime == nil {
		return true
	}
	createdAt := mfaConfig.TempOtpSecretCreatedTime.AsTime()
	return time.Since(createdAt) > mfaTempSecretExpiration
}

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

// generateRecoveryCodes generates n recovery codes.
func generateRecoveryCodes(n int) ([]string, error) {
	recoveryCodes := make([]string, n)
	for i := range n {
		code, err := common.RandomString(10)
		if err != nil {
			return nil, err
		}
		recoveryCodes[i] = code
	}
	return recoveryCodes, nil
}

func (s *UserService) userCountGuard(ctx context.Context) error {
	userLimit := s.licenseService.GetUserLimit(ctx)

	count, err := s.store.CountActiveEndUsers(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if count >= userLimit {
		return connect.NewError(connect.CodeResourceExhausted, errors.Errorf("reached the maximum user count %d", userLimit))
	}
	return nil
}

func isUserWorkspaceAdmin(ctx context.Context, stores *store.Store, user *store.UserMessage) (bool, error) {
	workspacePolicy, err := stores.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return false, err
	}
	roles := utils.GetUserFormattedRolesMap(ctx, stores, user, workspacePolicy.Policy)
	return roles[common.FormatRole(common.WorkspaceAdmin)], nil
}
