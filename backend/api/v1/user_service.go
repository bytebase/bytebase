package v1

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// UserService implements the user service.
type UserService struct {
	v1connect.UnimplementedUserServiceHandler
	store          *store.Store
	secret         string
	licenseService *enterprise.LicenseService
	metricReporter *metricreport.Reporter
	profile        *config.Profile
	stateCfg       *state.State
	iamManager     *iam.Manager
}

// NewUserService creates a new UserService.
func NewUserService(store *store.Store, secret string, licenseService *enterprise.LicenseService, metricReporter *metricreport.Reporter, profile *config.Profile, stateCfg *state.State, iamManager *iam.Manager) *UserService {
	return &UserService{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		metricReporter: metricReporter,
		profile:        profile,
		stateCfg:       stateCfg,
		iamManager:     iamManager,
	}
}

// GetUser gets a user.
func (s *UserService) GetUser(ctx context.Context, request *connect.Request[v1pb.GetUserRequest]) (*connect.Response[v1pb.User], error) {
	userID, err := common.GetUserID(request.Msg.Name)
	var user *store.UserMessage
	if err != nil {
		email, err := common.GetUserEmail(request.Msg.Name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		u, err := s.store.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
		}
		user = u
	} else {
		u, err := s.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
		}
		user = u
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %d not found", userID))
	}
	return connect.NewResponse(convertToUser(user)), nil
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
func (*UserService) GetCurrentUser(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[v1pb.User], error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("authenticated user not found"))
	}
	return connect.NewResponse(convertToUser(user)), nil
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
	if err := parseListUserFilter(find, request.Msg.Filter); err != nil {
		return nil, err
	}
	if v := find.ProjectID; v != nil {
		user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
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
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list user, error: %v", err))
	}

	nextPageToken := ""
	if len(users) == limitPlusOne {
		users = users[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal next page token, error: %v", err))
		}
	}

	response := &v1pb.ListUsersResponse{
		NextPageToken: nextPageToken,
	}
	for _, user := range users {
		response.Users = append(response.Users, convertToUser(user))
	}
	return connect.NewResponse(response), nil
}

func parseListUserFilter(find *store.FindUserMessage, filter string) error {
	if filter == "" {
		return nil
	}
	e, err := cel.NewEnv()
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
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
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid user type filter %q", value))
			}
			principalType, err := convertToPrincipalType(v1pb.UserType(v1UserType))
			if err != nil {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse the user type %q with error: %v", v1UserType, err))
			}
			positionalArgs = append(positionalArgs, principalType)
			return fmt.Sprintf("principal.type = $%d", len(positionalArgs)), nil
		case "state":
			v1State, ok := v1pb.State_value[value.(string)]
			if !ok {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid state filter %q", value))
			}
			positionalArgs = append(positionalArgs, v1pb.State(v1State) == v1pb.State_DELETED)
			return fmt.Sprintf("principal.deleted = $%d", len(positionalArgs)), nil
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid project filter %q", value))
			}
			find.ProjectID = &projectID
			return "TRUE", nil
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
		}
	}

	parseToUserTypeSQL := func(expr celast.Expr, relation string) (string, error) {
		variable, value := getVariableAndValueFromExpr(expr)
		if variable != "user_type" {
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only "user_type" support "user_type in [xx]"/"!(user_type in [xx])" operator`))
		}

		rawTypeList, ok := value.([]any)
		if !ok {
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid user_type value %q", value))
		}
		if len(rawTypeList) == 0 {
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("empty user_type filter"))
		}
		userTypeList := []string{}
		for _, rawType := range rawTypeList {
			v1UserType, ok := v1pb.UserType_value[rawType.(string)]
			if !ok {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid user type filter %q", rawType))
			}
			principalType, err := convertToPrincipalType(v1pb.UserType(v1UserType))
			if err != nil {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse the user type %q with error: %v", v1UserType, err))
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
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`invalid args for %q`, variable))
				}
				value := args[0].AsLiteral().Value()
				if variable != "name" && variable != "email" {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only "name" and "email" support %q operator, but found %q`, celoverloads.Matches, variable))
				}
				strValue, ok := value.(string)
				if !ok {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect string, got %T, hint: filter literals should be string", value))
				}
				return "LOWER(principal." + variable + ") LIKE '%" + strings.ToLower(strValue) + "%'", nil
			case celoperators.In:
				return parseToUserTypeSQL(expr, "IN")
			case celoperators.LogicalNot:
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only support !(user_type in ["{type1}", "{type2}"]) format`))
				}
				return parseToUserTypeSQL(args[0], "NOT IN")
			default:
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function %v", functionName))
			}
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected expr kind %v", expr.Kind()))
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
func (s *UserService) CreateUser(ctx context.Context, request *connect.Request[v1pb.CreateUserRequest]) (*connect.Response[v1pb.User], error) {
	if err := s.userCountGuard(ctx); err != nil {
		return nil, err
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find workspace setting, error: %v", err))
	}

	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DISALLOW_SELF_SERVICE_SIGNUP); err == nil {
		if setting.DisallowSignup || s.profile.SaaS {
			callerUser, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
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
	if request.Msg.User.UserType != v1pb.UserType_SERVICE_ACCOUNT && request.Msg.User.UserType != v1pb.UserType_USER {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("support user and service account only"))
	}

	count, err := s.store.CountUsers(ctx, storepb.PrincipalType_END_USER)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to count users, error: %v", err))
	}
	firstEndUser := count == 0

	if request.Msg.User.Phone != "" {
		if err := common.ValidatePhone(request.Msg.User.Phone); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid phone %q, error: %v", request.Msg.User.Phone, err))
		}
	}

	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, request.Msg.User.Email, principalType == storepb.PrincipalType_SERVICE_ACCOUNT, false); err != nil {
		return nil, err
	}
	existingUser, err := s.store.GetUserByEmail(ctx, request.Msg.User.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find user by email, error: %v", err))
	}
	if existingUser != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("email %s is already existed", request.Msg.User.Email))
	}

	password := request.Msg.User.Password
	if request.Msg.User.UserType == v1pb.UserType_SERVICE_ACCOUNT {
		pwd, err := common.RandomString(20)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate access key for service account"))
		}
		password = fmt.Sprintf("%s%s", common.ServiceAccountAccessKeyPrefix, pwd)
	} else {
		if password != "" {
			if err := s.validatePassword(ctx, password); err != nil {
				return nil, err
			}
		} else {
			pwd, err := common.RandomString(20)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate random password for service account"))
			}
			password = pwd
		}
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate password hash, error: %v", err))
	}
	userMessage := &store.UserMessage{
		Email:        request.Msg.User.Email,
		Name:         request.Msg.User.Title,
		Phone:        request.Msg.User.Phone,
		Type:         principalType,
		PasswordHash: string(passwordHash),
	}

	user, err := s.store.CreateUser(ctx, userMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create user, error: %v", err))
	}

	if firstEndUser {
		// The first end user should be workspace admin.
		updateRole := &store.PatchIamPolicyMessage{
			Member: common.FormatUserUID(user.ID),
			Roles:  []string{common.FormatRole(common.WorkspaceAdmin)},
		}
		if _, err := s.store.PatchWorkspaceIamPolicy(ctx, updateRole); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	isFirstUser := user.ID == common.PrincipalIDForFirstUser
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
	if request.Msg.User.UserType == v1pb.UserType_SERVICE_ACCOUNT {
		userResponse.ServiceKey = password
	}
	return connect.NewResponse(userResponse), nil
}

func (s *UserService) validatePassword(ctx context.Context, password string) error {
	passwordRestriction, err := s.store.GetPasswordRestrictionSetting(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to get password restriction with error: %v", err))
	}
	if len(password) < int(passwordRestriction.MinLength) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password length should no less than %v characters", passwordRestriction.MinLength))
	}
	if passwordRestriction.RequireNumber && !regexp.MustCompile("[0-9]+").MatchString(password) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must contains at least 1 number"))
	}
	if passwordRestriction.RequireLetter && !regexp.MustCompile("[a-zA-Z]+").MatchString(password) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must contains at least 1 lower case letter"))
	}
	if passwordRestriction.RequireUppercaseLetter && !regexp.MustCompile("[A-Z]+").MatchString(password) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must contains at least 1 upper case letter"))
	}
	if passwordRestriction.RequireSpecialCharacter && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+`).MatchString(password) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password must contains at least 1 special character"))
	}
	return nil
}

// UpdateUser updates a user.
func (s *UserService) UpdateUser(ctx context.Context, request *connect.Request[v1pb.UpdateUserRequest]) (*connect.Response[v1pb.User], error) {
	callerUser, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	if request.Msg.User == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("user must be set"))
	}
	if request.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
	}

	userID, err := common.GetUserID(request.Msg.User.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
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
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %d not found", userID))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q has been deleted", userID))
	}

	if callerUser.ID != userID {
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
			if err := validateEmailWithDomains(ctx, s.licenseService, s.store, request.Msg.User.Email, user.Type == storepb.PrincipalType_SERVICE_ACCOUNT, false); err != nil {
				return nil, err
			}
			existedUser, err := s.store.GetUserByEmail(ctx, request.Msg.User.Email)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find user list, error: %v", err))
			}
			if existedUser != nil && existedUser.ID != user.ID {
				return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("email %s is already existed", request.Msg.User.Email))
			}
			patch.Email = &request.Msg.User.Email
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
				patch.MFAConfig = &storepb.MFAConfig{
					OtpSecret:     user.MFAConfig.TempOtpSecret,
					RecoveryCodes: user.MFAConfig.TempRecoveryCodes,
				}
			} else {
				setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find workspace setting, error: %v", err))
				}
				if setting.Require_2Fa {
					isWorkspaceAdmin, err := isUserWorkspaceAdmin(ctx, s.store, callerUser)
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check user roles, error: %v", err))
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
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid phone number %q, error: %v", request.Msg.User.Phone, err))
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
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate password hash, error: %v", err))
		}
		passwordHashStr := string(passwordHash)
		patch.PasswordHash = &passwordHashStr
	}
	// This flag is mainly used for validating OTP code when user setup MFA.
	// We only validate OTP code but not update user.
	if request.Msg.OtpCode != nil {
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
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate MFA secret, error: %v", err))
		}
		tempRecoveryCodes, err := generateRecoveryCodes(10)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate recovery codes, error: %v", err))
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
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update user, error: %v", err))
	}

	userResponse := convertToUser(user)
	if request.Msg.User.UserType == v1pb.UserType_SERVICE_ACCOUNT && passwordPatch != nil {
		userResponse.ServiceKey = *passwordPatch
	}
	return connect.NewResponse(userResponse), nil
}

// DeleteUser deletes a user.
func (s *UserService) DeleteUser(ctx context.Context, request *connect.Request[v1pb.DeleteUserRequest]) (*connect.Response[emptypb.Empty], error) {
	callerUser, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
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

	userID, err := common.GetUserID(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %d not found", userID))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q has been deleted", userID))
	}

	// Check if there is still workspace admin if the current user is deleted.
	policy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, err
	}
	hasExtraWorkspaceAdmin, err := s.hasExtraWorkspaceAdmin(ctx, policy.Policy, user.ID)
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

func (s *UserService) hasExtraWorkspaceAdmin(ctx context.Context, policy *storepb.IamPolicy, userID int) (bool, error) {
	workspaceAdminRole := common.FormatRole(common.WorkspaceAdmin)
	userMember := common.FormatUserUID(userID)

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
	if err := s.userCountGuard(ctx); err != nil {
		return nil, err
	}

	callerUser, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
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

	userID, err := common.GetUserID(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %d not found", userID))
	}
	if !user.MemberDeleted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("user %q is already active", userID))
	}

	user, err = s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &undeletePatch})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertToUser(user)), nil
}

func convertToV1UserType(userType storepb.PrincipalType) v1pb.UserType {
	switch userType {
	case storepb.PrincipalType_END_USER:
		return v1pb.UserType_USER
	case storepb.PrincipalType_SYSTEM_BOT:
		return v1pb.UserType_SYSTEM_BOT
	case storepb.PrincipalType_SERVICE_ACCOUNT:
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

	for _, group := range user.Groups {
		convertedUser.Groups = append(convertedUser.Groups, common.FormatGroupEmail(group))
	}

	if user.MFAConfig != nil {
		convertedUser.MfaEnabled = user.MFAConfig.OtpSecret != ""
		convertedUser.MfaSecret = user.MFAConfig.TempOtpSecret
		convertedUser.RecoveryCodes = user.MFAConfig.TempRecoveryCodes
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
	setting, err := stores.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to find workspace setting, error: %v", err))
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

func isUserWorkspaceAdmin(ctx context.Context, stores *store.Store, user *store.UserMessage) (bool, error) {
	workspacePolicy, err := stores.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return false, err
	}
	roles := utils.GetUserFormattedRolesMap(ctx, stores, user, workspacePolicy.Policy)
	return roles[common.FormatRole(common.WorkspaceAdmin)], nil
}
