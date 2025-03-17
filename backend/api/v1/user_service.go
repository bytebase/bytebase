package v1

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// UserService implements the user service.
type UserService struct {
	v1pb.UnimplementedUserServiceServer
	store          *store.Store
	secret         string
	licenseService enterprise.LicenseService
	metricReporter *metricreport.Reporter
	profile        *config.Profile
	stateCfg       *state.State
	iamManager     *iam.Manager
	postCreateUser func(ctx context.Context, user *store.UserMessage, firstEndUser bool) error
}

// NewUserService creates a new UserService.
func NewUserService(store *store.Store, secret string, licenseService enterprise.LicenseService, metricReporter *metricreport.Reporter, profile *config.Profile, stateCfg *state.State, iamManager *iam.Manager, postCreateUser func(ctx context.Context, user *store.UserMessage, firstEndUser bool) error) (*UserService, error) {
	return &UserService{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		metricReporter: metricReporter,
		profile:        profile,
		stateCfg:       stateCfg,
		iamManager:     iamManager,
		postCreateUser: postCreateUser,
	}, nil
}

// GetUser gets a user.
func (s *UserService) GetUser(ctx context.Context, request *v1pb.GetUserRequest) (*v1pb.User, error) {
	userID, err := common.GetUserID(request.Name)
	var user *store.UserMessage
	if err != nil {
		email, err := common.GetUserEmail(request.Name)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		u, err := s.store.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
		}
		user = u
	} else {
		u, err := s.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
		}
		user = u
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
	}
	return convertToUser(user), nil
}

// StatUsers count users by type and state.
func (s *UserService) StatUsers(ctx context.Context, _ *v1pb.StatUsersRequest) (*v1pb.StatUsersResponse, error) {
	stats, err := s.store.StatUsers(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to stat users, error: %v", err)
	}
	response := &v1pb.StatUsersResponse{}

	for _, stat := range stats {
		response.Stats = append(response.Stats, &v1pb.StatUsersResponse_StatUser{
			State:    convertDeletedToState(stat.Deleted),
			UserType: convertToV1UserType(stat.Type),
			Count:    int32(stat.Count),
		})
	}
	return response, nil
}

// ListUsers lists all users.
func (s *UserService) ListUsers(ctx context.Context, request *v1pb.ListUsersRequest) (*v1pb.ListUsersResponse, error) {
	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindUserMessage{
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		ShowDeleted: request.ShowDeleted,
	}
	if err := parseListUserFilter(find, request.Filter); err != nil {
		return nil, err
	}
	if v := find.ProjectID; v != nil {
		user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
		if !ok {
			return nil, status.Errorf(codes.Internal, "user not found")
		}
		hasPermission, err := s.iamManager.CheckPermission(ctx, iam.PermissionProjectsGet, user, *v)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check user permission")
		}
		if !hasPermission {
			return nil, status.Errorf(codes.PermissionDenied, "user does not have permission %q", iam.PermissionProjectsGet)
		}
	}

	users, err := s.store.ListUsers(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user, error: %v", err)
	}

	nextPageToken := ""
	if len(users) == limitPlusOne {
		users = users[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal next page token, error: %v", err)
		}
	}

	response := &v1pb.ListUsersResponse{
		NextPageToken: nextPageToken,
	}
	for _, user := range users {
		response.Users = append(response.Users, convertToUser(user))
	}
	return response, nil
}

func parseListUserFilter(find *store.FindUserMessage, filter string) error {
	if filter == "" {
		return nil
	}
	e, err := cel.NewEnv()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return status.Errorf(codes.InvalidArgument, "failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	parseToSQL := func(variable, value any) (string, error) {
		switch variable {
		case "email":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("principal.email = $%d", len(positionalArgs)), nil
		case "name":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("principal.name = $%d", len(positionalArgs)), nil
		case "user_type":
			v1UserType, ok := v1pb.UserType_value[value.(string)]
			if !ok {
				return "", status.Errorf(codes.InvalidArgument, "invalid user type filter %q", value)
			}
			principalType, err := convertToPrincipalType(v1pb.UserType(v1UserType))
			if err != nil {
				return "", status.Errorf(codes.InvalidArgument, "failed to parse the user type %q with error: %v", v1UserType, err.Error())
			}
			positionalArgs = append(positionalArgs, principalType)
			return fmt.Sprintf("principal.type = $%d", len(positionalArgs)), nil
		case "state":
			v1State, ok := v1pb.State_value[value.(string)]
			if !ok {
				return "", status.Errorf(codes.InvalidArgument, "invalid state filter %q", value)
			}
			positionalArgs = append(positionalArgs, v1pb.State(v1State) == v1pb.State_DELETED)
			return fmt.Sprintf("principal.deleted = $%d", len(positionalArgs)), nil
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return "", status.Errorf(codes.InvalidArgument, "invalid project filter %q", value)
			}
			find.ProjectID = &projectID
			return "TRUE", nil
		default:
			return "", status.Errorf(codes.InvalidArgument, "unsupport variable %q", variable)
		}
	}

	parseToUserTypeSQL := func(expr celast.Expr, relation string) (string, error) {
		variable, value := getVariableAndValueFromExpr(expr)
		if variable != "user_type" {
			return "", status.Errorf(codes.InvalidArgument, `only "user_type" support "user_type in [xx]"/"!(user_type in [xx])" operator`)
		}

		rawTypeList, ok := value.([]any)
		if !ok {
			return "", status.Errorf(codes.InvalidArgument, "invalid user_type value %q", value)
		}
		if len(rawTypeList) == 0 {
			return "", status.Errorf(codes.InvalidArgument, "empty user_type filter")
		}
		userTypeList := []string{}
		for _, rawType := range rawTypeList {
			v1UserType, ok := v1pb.UserType_value[rawType.(string)]
			if !ok {
				return "", status.Errorf(codes.InvalidArgument, "invalid user type filter %q", rawType)
			}
			principalType, err := convertToPrincipalType(v1pb.UserType(v1UserType))
			if err != nil {
				return "", status.Errorf(codes.InvalidArgument, "failed to parse the user type %q with error: %v", v1UserType, err.Error())
			}
			positionalArgs = append(positionalArgs, principalType)
			userTypeList = append(userTypeList, fmt.Sprintf("$%d", len(positionalArgs)))
		}

		return fmt.Sprintf("principal.type %s (%s)", relation, strings.Join(userTypeList, ",")), nil
	}

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				return getSubConditionFromExpr(expr, getFilter, "OR")
			case celoperators.LogicalAnd:
				return getSubConditionFromExpr(expr, getFilter, "AND")
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", status.Errorf(codes.InvalidArgument, `invalid args for %q`, variable)
				}
				value := args[0].AsLiteral().Value()
				if variable != "name" && variable != "email" {
					return "", status.Errorf(codes.InvalidArgument, `only "name" and "email" support %q operator, but found %q`, celoverloads.Matches, variable)
				}
				strValue, ok := value.(string)
				if !ok {
					return "", status.Errorf(codes.InvalidArgument, "expect string, got %T, hint: filter literals should be string", value)
				}
				return "LOWER(principal." + variable + ") LIKE '%" + strings.ToLower(strValue) + "%'", nil
			case celoperators.In:
				return parseToUserTypeSQL(expr, "IN")
			case celoperators.LogicalNot:
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", status.Errorf(codes.InvalidArgument, `only support !(user_type in ["{type1}", "{type2}"]) format`)
				}
				return parseToUserTypeSQL(args[0], "NOT IN")
			default:
				return "", status.Errorf(codes.InvalidArgument, "unexpected function %v", functionName)
			}
		default:
			return "", status.Errorf(codes.InvalidArgument, "unexpected expr kind %v", expr.Kind())
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return err
	}

	find.Filter = &store.ListResourceFilter{
		Args:  positionalArgs,
		Where: "(" + where + ")",
	}
	return nil
}

// CreateUser creates a user.
func (s *UserService) CreateUser(ctx context.Context, request *v1pb.CreateUserRequest) (*v1pb.User, error) {
	if err := s.userCountGuard(ctx); err != nil {
		return nil, err
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace setting, error: %v", err)
	}

	if setting.DisallowSignup {
		callerUser, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "sign up is disallowed")
		}
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersCreate, callerUser)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "user does not have permission %q", iam.PermissionUsersCreate)
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

	count, err := s.store.CountUsers(ctx, api.EndUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count users, error: %v", err)
	}
	firstEndUser := count == 0

	if request.User.Phone != "" {
		if err := common.ValidatePhone(request.User.Phone); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid phone %q, error: %v", request.User.Phone, err)
		}
	}

	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, request.User.Email, principalType == api.ServiceAccount, false); err != nil {
		return nil, err
	}
	existingUser, err := s.store.GetUserByEmail(ctx, request.User.Email)
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
	} else {
		if password != "" {
			if err := s.validatePassword(ctx, password); err != nil {
				return nil, err
			}
		} else {
			pwd, err := common.RandomString(20)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to generate random password for service account.")
			}
			password = pwd
		}
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

	user, err := s.store.CreateUser(ctx, userMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user, error: %v", err)
	}

	if firstEndUser {
		// The first end user should be workspace admin.
		updateRole := &store.PatchIamPolicyMessage{
			Member: common.FormatUserUID(user.ID),
			Roles:  []string{common.FormatRole(api.WorkspaceAdmin.String())},
		}
		if _, err := s.store.PatchWorkspaceIamPolicy(ctx, updateRole); err != nil {
			return nil, err
		}
	}

	if err := s.postCreateUser(ctx, user, firstEndUser); err != nil {
		return nil, err
	}

	isFirstUser := user.ID == api.PrincipalIDForFirstUser
	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricapi.PrincipalRegistrationMetricName,
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

	userResponse := convertToUser(user)
	if request.User.UserType == v1pb.UserType_SERVICE_ACCOUNT {
		userResponse.ServiceKey = password
	}
	return userResponse, nil
}

func (s *UserService) validatePassword(ctx context.Context, password string) error {
	passwordRestriction, err := s.store.GetPasswordRestrictionSetting(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get password restriction with error: %v", err)
	}
	if len(password) < int(passwordRestriction.MinLength) {
		return status.Errorf(codes.InvalidArgument, "password length should no less than %v characters", passwordRestriction.MinLength)
	}
	if passwordRestriction.RequireNumber && !regexp.MustCompile("[0-9]+").MatchString(password) {
		return status.Errorf(codes.InvalidArgument, "password must contains at least 1 number")
	}
	if passwordRestriction.RequireLetter && !regexp.MustCompile("[a-zA-Z]+").MatchString(password) {
		return status.Errorf(codes.InvalidArgument, "password must contains at least 1 lower case letter")
	}
	if passwordRestriction.RequireUppercaseLetter && !regexp.MustCompile("[A-Z]+").MatchString(password) {
		return status.Errorf(codes.InvalidArgument, "password must contains at least 1 upper case letter")
	}
	if passwordRestriction.RequireSpecialCharacter && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+`).MatchString(password) {
		return status.Errorf(codes.InvalidArgument, "password must contains at least 1 special character")
	}
	return nil
}

// UpdateUser updates a user.
func (s *UserService) UpdateUser(ctx context.Context, request *v1pb.UpdateUserRequest) (*v1pb.User, error) {
	callerUser, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "failed to get caller user")
	}
	if request.User == nil {
		return nil, status.Errorf(codes.InvalidArgument, "user must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	userID, err := common.GetUserID(request.User.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	if callerUser.ID != userID {
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersUpdate, callerUser)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "user does not have permission %q", iam.PermissionUsersUpdate)
		}
	}

	var passwordPatch *string
	patch := &store.UpdateUserMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "email":
			if user.Profile.Source != "" {
				return nil, status.Errorf(codes.InvalidArgument, "cannot change email for external user")
			}
			if err := validateEmailWithDomains(ctx, s.licenseService, s.store, request.User.Email, user.Type == api.ServiceAccount, false); err != nil {
				return nil, err
			}
			existedUser, err := s.store.GetUserByEmail(ctx, request.User.Email)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find user list, error: %v", err)
			}
			if existedUser != nil && existedUser.ID != user.ID {
				return nil, status.Errorf(codes.AlreadyExists, "email %s is already existed", request.User.Email)
			}
			patch.Email = &request.User.Email
		case "title":
			patch.Name = &request.User.Title
		case "password":
			if user.Type != api.EndUser {
				return nil, status.Errorf(codes.InvalidArgument, "password can be mutated for end users only")
			}
			if err := s.validatePassword(ctx, request.User.Password); err != nil {
				return nil, err
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
					isWorkspaceAdmin, err := isUserWorkspaceAdmin(ctx, s.store, callerUser)
					if err != nil {
						return nil, status.Errorf(codes.Internal, "failed to check user roles, error: %v", err)
					}
					// Allow workspace admin to disable 2FA even if it is required.
					if !isWorkspaceAdmin {
						return nil, status.Errorf(codes.InvalidArgument, "2FA is required and cannot be disabled")
					}
				}
				patch.MFAConfig = &storepb.MFAConfig{}
			}
		case "phone":
			if request.User.Phone != "" {
				if err := common.ValidatePhone(request.User.Phone); err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid phone number %q, error: %v", request.User.Phone, err)
				}
			}
			patch.Phone = &request.User.Phone
		}
	}
	if passwordPatch != nil {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(*passwordPatch)); err == nil {
			// return bad request if the passwords match
			return nil, status.Errorf(codes.InvalidArgument, "password cannot be the same")
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte((*passwordPatch)), bcrypt.DefaultCost)
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

	user, err = s.store.UpdateUser(ctx, user, patch)
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
func (s *UserService) DeleteUser(ctx context.Context, request *v1pb.DeleteUserRequest) (*emptypb.Empty, error) {
	callerUser, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "failed to get caller user")
	}
	ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersDelete, callerUser)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "user does not have permission %q", iam.PermissionUsersDelete)
	}

	userID, err := common.GetUserID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	// Check if there is still workspace admin if the current user is deleted.
	policy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, err
	}
	ok = hasExtraWorkspaceAdmin(policy.Policy, user.ID)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "workspace must have at least one admin")
	}

	if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &deletePatch}); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func hasExtraWorkspaceAdmin(policy *storepb.IamPolicy, userID int) bool {
	workspaceAdminRole := common.FormatRole(api.WorkspaceAdmin.String())
	userMember := common.FormatUserUID(userID)
	systemBotMember := common.FormatUserUID(api.SystemBotID)
	for _, binding := range policy.GetBindings() {
		if binding.GetRole() != workspaceAdminRole {
			continue
		}
		for _, member := range binding.GetMembers() {
			if member != userMember && member != systemBotMember {
				return true
			}
		}
	}
	return false
}

// UndeleteUser undeletes a user.
func (s *UserService) UndeleteUser(ctx context.Context, request *v1pb.UndeleteUserRequest) (*v1pb.User, error) {
	if err := s.userCountGuard(ctx); err != nil {
		return nil, err
	}

	callerUser, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "failed to get caller user")
	}
	ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersUndelete, callerUser)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "user does not have permission %q", iam.PermissionUsersUndelete)
	}

	userID, err := common.GetUserID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	user, err = s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &undeletePatch})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return convertToUser(user), nil
}

func convertToV1UserType(userType api.PrincipalType) v1pb.UserType {
	switch userType {
	case api.EndUser:
		return v1pb.UserType_USER
	case api.SystemBot:
		return v1pb.UserType_SYSTEM_BOT
	case api.ServiceAccount:
		return v1pb.UserType_SERVICE_ACCOUNT
	default:
		return v1pb.UserType_USER_TYPE_UNSPECIFIED
	}
}

func convertToUser(user *store.UserMessage) *v1pb.User {
	convertedUser := &v1pb.User{
		Name:     common.FormatUserUID(user.ID),
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

func validateEmailWithDomains(ctx context.Context, licenseService enterprise.LicenseService, stores *store.Store, email string, isServiceAccount bool, checkDomainSetting bool) error {
	if licenseService.IsFeatureEnabled(api.FeatureDomainRestriction) != nil {
		// nolint:nilerr
		// feature not enabled, only validate email and skip domain restriction.
		if err := validateEmail(email); err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid email: %v", err.Error())
		}
		return nil
	}
	setting, err := stores.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to find workspace setting, error: %v", err)
	}

	var allowedDomains []string
	if checkDomainSetting || setting.EnforceIdentityDomain {
		allowedDomains = setting.Domains
	}

	// Check if the email is valid.
	if err := validateEmail(email); err != nil {
		return err
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
			return errors.Errorf("email %q does not belong to domains %v", email, allowedDomains)
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

func (s *UserService) userCountGuard(ctx context.Context) error {
	userLimit := s.licenseService.GetPlanLimitValue(ctx, enterprise.PlanLimitMaximumUser)

	count, err := s.store.CountActiveUsers(ctx)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if count >= userLimit {
		return status.Errorf(codes.ResourceExhausted, "reached the maximum user count %d", userLimit)
	}
	return nil
}

func isUserWorkspaceAdmin(ctx context.Context, stores *store.Store, user *store.UserMessage) (bool, error) {
	workspacePolicy, err := stores.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return false, err
	}
	roles := utils.GetUserFormattedRolesMap(ctx, stores, user, workspacePolicy.Policy)
	return roles[common.FormatRole(api.WorkspaceAdmin.String())], nil
}
