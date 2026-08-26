package v1

import (
	"context"
	"fmt"
	"log/slog"
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
	"github.com/bytebase/bytebase/backend/common/permission"
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

// GetUser gets a user. Only returns the user if they are a member of the caller's workspace.
func (s *UserService) GetUser(ctx context.Context, request *connect.Request[v1pb.GetUserRequest]) (*connect.Response[v1pb.User], error) {
	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}

	// Only scope to workspace IAM in SaaS mode to prevent cross-workspace access.
	// In non-SaaS mode, users may exist in the principal table before being added to IAM.
	var workspace string
	if s.profile.SaaS {
		workspace = common.GetWorkspaceIDFromContext(ctx)
	}
	users, err := s.store.BatchGetUsersByEmails(ctx, workspace, []string{email})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user"))
	}
	if len(users) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}
	v1User, err := convertToUser(ctx, s.iamManager, users[0])
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	return connect.NewResponse(v1User), nil
}

// BatchGetUsers get users in batch.
func (s *UserService) BatchGetUsers(ctx context.Context, request *connect.Request[v1pb.BatchGetUsersRequest]) (*connect.Response[v1pb.BatchGetUsersResponse], error) {
	// Extract and validate emails from names
	emails := make([]string, 0, len(request.Msg.Names))
	for _, name := range request.Msg.Names {
		email, err := common.GetUserEmail(name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if err := validateEndUserEmail(email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	// Only scope to workspace IAM in SaaS mode.
	var workspace string
	if s.profile.SaaS {
		workspace = common.GetWorkspaceIDFromContext(ctx)
	}
	users, err := s.store.BatchGetUsersByEmails(ctx, workspace, emails)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to batch get users"))
	}

	// Build a map for quick lookup
	response := &v1pb.BatchGetUsersResponse{}
	for _, user := range users {
		v1User, err := convertToUser(ctx, s.iamManager, user)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
		}
		response.Users = append(response.Users, v1User)
	}

	return connect.NewResponse(response), nil
}

// GetCurrentUser gets the current authenticated user.
func (s *UserService) GetCurrentUser(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[v1pb.User], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("authenticated user not found"))
	}
	v1User, err := convertToUser(ctx, s.iamManager, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	return connect.NewResponse(v1User), nil
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
	filterResult, err := store.GetAccountListFilter(request.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.FilterQ = filterResult.Query
	find.ProjectID = filterResult.ProjectID

	// Set workspace scope:
	// - When ProjectID is set, workspace is required for the project member CTE query.
	// - In SaaS mode (no ProjectID), scope to workspace IAM members to prevent cross-workspace listing.
	if find.ProjectID != nil {
		find.Workspace = common.GetWorkspaceIDFromContext(ctx)

		user, ok := GetUserFromContext(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
		}
		hasPermission, err := s.iamManager.CheckPermission(ctx, permission.ProjectsGet, user, common.GetWorkspaceIDFromContext(ctx), *find.ProjectID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check user permission"))
		}
		if !hasPermission {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", permission.ProjectsGet))
		}
	} else if s.profile.SaaS {
		find.Workspace = common.GetWorkspaceIDFromContext(ctx)
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
		v1User, err := convertToUser(ctx, s.iamManager, user)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
		}
		response.Users = append(response.Users, v1User)
	}
	return connect.NewResponse(response), nil
}

// CreateUser creates a user in the caller's workspace (admin action, self-hosted only).
// In SaaS mode, admins should add users via workspace IAM policy instead.
func (s *UserService) CreateUser(ctx context.Context, request *connect.Request[v1pb.CreateUserRequest]) (*connect.Response[v1pb.User], error) {
	if s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.Errorf("CreateUser is not available in SaaS mode, add users via workspace IAM policy instead"))
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
	if request.Msg.User.Phone != "" {
		if err := common.ValidatePhone(request.Msg.User.Phone); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid phone %q", request.Msg.User.Phone))
		}
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	email := request.Msg.User.Email

	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, workspaceID, email, false); err != nil {
		return nil, err
	}

	existingUser, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user by email"))
	}
	if existingUser != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("email %s exists", email))
	}

	// Validate password.
	password := request.Msg.User.Password
	if password != "" {
		if err := s.validatePassword(ctx, workspaceID, password); err != nil {
			return nil, err
		}
	} else {
		pwd, err := common.RandomString(20)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate random password for user"))
		}
		password = pwd
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate password hash"))
	}

	if err := s.preAddUserGuard(ctx, workspaceID); err != nil {
		return nil, err
	}

	user, err := s.store.CreateUser(ctx, &store.UserMessage{
		Email:        email,
		Name:         request.Msg.User.Title,
		Phone:        request.Msg.User.Phone,
		PasswordHash: string(passwordHash),
		Profile:      &storepb.UserProfile{},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create user"))
	}

	userResponse, err := convertToUser(ctx, s.iamManager, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	userResponse.Workspace = common.FormatWorkspace(workspaceID)
	return connect.NewResponse(userResponse), nil
}

func (s *UserService) validatePassword(ctx context.Context, workspaceID, password string) error {
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
	passwordRestriction := convertToStorePasswordRestriction(restriction.PasswordRestriction)
	return validatePasswordWithRestriction(password, passwordRestriction)
}

func validatePasswordWithRestriction(password string, passwordRestriction *storepb.WorkspaceProfileSetting_PasswordRestriction) error {
	if err := common.ValidatePassword(password, passwordRestriction); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
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
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user"))
	}
	if user == nil {
		if request.Msg.AllowMissing {
			ok, err := s.iamManager.CheckPermission(ctx, permission.UsersCreate, callerUser, common.GetWorkspaceIDFromContext(ctx))
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", permission.UsersCreate))
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
		// In SaaS mode, only self-updates are allowed. A workspace admin should not
		// edit another user's profile since the principal is global.
		if s.profile.SaaS {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("updating other users is not allowed in SaaS mode"))
		}
		ok, err := s.iamManager.CheckPermission(ctx, permission.UsersUpdate, callerUser, common.GetWorkspaceIDFromContext(ctx))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
		}
		if !ok {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", permission.UsersUpdate))
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
			if callerUser.ID == user.ID {
				// Changing your own password is ChangePassword's job: it is the
				// method that knows the old one mattered.
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("use ChangePassword to change your own password"))
			}
			if err := s.validatePassword(ctx, common.GetWorkspaceIDFromContext(ctx), request.Msg.User.Password); err != nil {
				return nil, err
			}
			passwordPatch = &request.Msg.User.Password
		case "mfa_enabled":
			// MFA is enrolled and disabled through its own methods, which can
			// state what they write; a boolean on an update mask cannot.
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("use EnableMFA, ConfirmRecoveryCodes or DisableMFA to change MFA"))
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
		patch.PasswordHash = new(string(passwordHash))

		// Revoke all refresh tokens for this user (including current session)
		// User must re-login after password change for security
		if err := s.store.DeleteWebRefreshTokensByUser(ctx, user.Email); err != nil {
			slog.Error("failed to revoke refresh tokens on password change", log.BBError(err), slog.String("user", user.Email))
		}
	}
	user, err = s.store.UpdateUser(ctx, user, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}

	userResponse, err := convertToUser(ctx, s.iamManager, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	return connect.NewResponse(userResponse), nil
}

// DeleteUser deletes a user.
func (s *UserService) DeleteUser(ctx context.Context, request *connect.Request[v1pb.DeleteUserRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.Errorf("DeleteUser is not available in SaaS mode, remove users from the workspace IAM policy instead"))
	}

	callerUser, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	ok, err := s.iamManager.CheckPermission(ctx, permission.UsersDelete, callerUser, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", permission.UsersDelete))
	}

	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
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
	policy, err := s.store.GetWorkspaceIamPolicy(ctx, common.GetWorkspaceIDFromContext(ctx))
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

func (s *UserService) hasExtraWorkspaceAdmin(ctx context.Context, policy *storepb.IamPolicy, user *store.UserMessage) (bool, error) {
	workspaceAdminRole := common.FormatRole(store.WorkspaceAdminRole)
	userMember := common.FormatUserEmail(user.Email)

	for _, binding := range policy.GetBindings() {
		if binding.GetRole() != workspaceAdminRole {
			continue
		}
		for _, member := range binding.GetMembers() {
			if member == userMember {
				continue
			}
			if member == common.AllUsers && !s.profile.SaaS {
				// allUsers means every user is an admin. Count all active end users
				// (not just workspace members) since allUsers includes everyone.
				count, err := s.store.CountActivePrincipals(ctx)
				if err != nil {
					return false, err
				}
				return count > 1, nil
			}
			users := utils.GetUsersByMember(ctx, s.store, common.GetWorkspaceIDFromContext(ctx), member)
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
	if s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.Errorf("UndeleteUser is not available in SaaS mode, add users back to the workspace IAM policy instead"))
	}

	callerUser, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	ok, err := s.iamManager.CheckPermission(ctx, permission.UsersUndelete, callerUser, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", permission.UsersUndelete))
	}

	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
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

	if err := s.preUndeleteUserGuard(ctx, workspaceID, user); err != nil {
		return nil, err
	}

	user, err = s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &undeletePatch})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	v1User, err := convertToUser(ctx, s.iamManager, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	return connect.NewResponse(v1User), nil
}

// UpdateEmail updates a user's email address.
func (s *UserService) UpdateEmail(ctx context.Context, request *connect.Request[v1pb.UpdateEmailRequest]) (*connect.Response[v1pb.User], error) {
	if s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.Errorf("CreateUser is not available in SaaS mode, add users via workspace IAM policy instead"))
	}

	callerUser, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	ok, err := s.iamManager.CheckPermission(ctx, permission.UsersUpdateEmail, callerUser, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", permission.UsersUpdateEmail))
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
	if err := validateEmailWithDomains(ctx, s.licenseService, s.store, common.GetWorkspaceIDFromContext(ctx), request.Msg.Email, false); err != nil {
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

	v1User, err := convertToUser(ctx, s.iamManager, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	return connect.NewResponse(v1User), nil
}

// convertToUser converts a stored user for a read. The MFA enrollment secrets
// stay out: temp_otp_secret is the TOTP seed and temp_recovery_codes are the
// codes that get past the second factor, and a response that carries either
// hands a reader the account's second factor. Every caller but one is a read,
// so this is the default and the exception has to be asked for by name.
var errSupersededPendingMFAState = connect.NewError(connect.CodeFailedPrecondition,
	errors.New("the pending MFA state has been superseded; restart from the mint step"))

// Self-service credential methods. These replace the UpdateUser field masks
// and flags that used to drive password changes and MFA enrollment — same
// operations, addressed as methods rather than as a mask plus three booleans,
// so each one states who may call it and what it writes.
//
// They are auth_method=CUSTOM: the ACL interceptor enforces nothing for them,
// so every handler checks the caller itself. DisableMFA is the one method with
// an administrator path.

// ChangePassword changes the caller's own password.
func (s *UserService) ChangePassword(ctx context.Context, request *connect.Request[v1pb.ChangePasswordRequest]) (*connect.Response[v1pb.User], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	if caller.Type != storepb.PrincipalType_END_USER {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password can be mutated for end users only"))
	}
	if err := s.validatePassword(ctx, common.GetWorkspaceIDFromContext(ctx), request.Msg.NewPassword); err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(caller.PasswordHash), []byte(request.Msg.NewPassword)) == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password cannot be the same"))
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Msg.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate password hash"))
	}

	hash := string(passwordHash)
	user, err := s.store.UpdateUser(ctx, caller, &store.UpdateUserMessage{PasswordHash: &hash})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}
	// Every session of this account, the caller's own included, has to be
	// re-established with the new password.
	if err := s.store.DeleteWebRefreshTokensByUser(ctx, user.Email); err != nil {
		slog.Error("failed to revoke refresh tokens on password change", log.BBError(err), slog.String("user", user.Email))
	}
	return s.convertUserResponse(ctx, user)
}

// StartMFAEnrollment mints a pending TOTP secret and recovery codes and hands
// them back. Any live factor is preserved: an enrollment that is abandoned, or
// superseded by a later one, changes nothing about how the account signs in.
func (s *UserService) StartMFAEnrollment(ctx context.Context, request *connect.Request[v1pb.StartMFAEnrollmentRequest]) (*connect.Response[v1pb.StartMFAEnrollmentResponse], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	tempSecret, err := generateRandSecret(caller.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate MFA secret"))
	}
	tempRecoveryCodes, err := generateRecoveryCodes(10)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate recovery codes"))
	}
	createdTime := timestamppb.Now()

	if _, err := s.store.UpdateUser(ctx, caller, &store.UpdateUserMessage{
		MFAConfig: &storepb.MFAConfig{
			OtpSecret:                caller.MFAConfig.GetOtpSecret(),
			RecoveryCodes:            caller.MFAConfig.GetRecoveryCodes(),
			TempOtpSecret:            tempSecret,
			TempRecoveryCodes:        tempRecoveryCodes,
			TempOtpSecretCreatedTime: createdTime,
		},
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}

	return connect.NewResponse(&v1pb.StartMFAEnrollmentResponse{
		OtpSecret:      tempSecret,
		RecoveryCodes:  tempRecoveryCodes,
		ExpireTime:     timestamppb.New(createdTime.AsTime().Add(mfaTempSecretExpiration)),
		PendingVersion: createdTime,
	}), nil
}

// EnableMFA verifies an otp_code against the pending enrollment and writes
// nothing. It exists to catch a mistyped authenticator before the caller is
// handed recovery codes; ConfirmRecoveryCodes is what makes the factor live.
func (s *UserService) EnableMFA(ctx context.Context, request *connect.Request[v1pb.EnableMFARequest]) (*connect.Response[v1pb.User], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	// Verification only, so a read is enough: this writes nothing, and the
	// promotion re-checks the version against the row it updates.
	if err := checkPendingMFAState(caller, request.Msg.PendingVersion); err != nil {
		return nil, err
	}
	if isMFATempSecretExpired(caller.MFAConfig) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA setup has expired, please restart the enrollment"))
	}
	if !totp.Validate(request.Msg.OtpCode, caller.MFAConfig.GetTempOtpSecret()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid OTP code"))
	}
	return s.convertUserResponse(ctx, caller)
}

// DisableMFA clears the account's entire MFA config, live and pending alike, so
// a confirmation left open cannot silently re-enable it.
func (s *UserService) DisableMFA(ctx context.Context, request *connect.Request[v1pb.DisableMFARequest]) (*connect.Response[v1pb.User], error) {
	callerUser, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user"))
	}
	if user == nil || user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}

	if callerUser.ID != user.ID {
		// Same cross-user gate as UpdateUser: the principal is global, so a
		// SaaS workspace admin gets no cross-user reach here either.
		if s.profile.SaaS {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("updating other users is not allowed in SaaS mode"))
		}
		hasPermission, err := s.iamManager.CheckPermission(ctx, permission.UsersUpdate, callerUser, common.GetWorkspaceIDFromContext(ctx))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
		}
		if !hasPermission {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", permission.UsersUpdate))
		}
	}

	// A non-admin caller cannot turn MFA off while the workspace requires it,
	// self-service or admin-assisted alike — unchanged from UpdateUser.
	setting, err := s.store.GetWorkspaceProfileSetting(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting"))
	}
	if setting.Require_2Fa {
		isWorkspaceAdmin, err := isUserWorkspaceAdmin(ctx, s.store, callerUser, common.GetWorkspaceIDFromContext(ctx))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check user roles"))
		}
		if !isWorkspaceAdmin {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("2FA is required and cannot be disabled"))
		}
	}

	updatedUser, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{MFAConfig: &storepb.MFAConfig{}})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}
	return s.convertUserResponse(ctx, updatedUser)
}

// RegenerateRecoveryCodes mints a pending recovery-code set beside the live
// one. The old codes keep working until ConfirmRecoveryCodes promotes these.
func (s *UserService) RegenerateRecoveryCodes(ctx context.Context, request *connect.Request[v1pb.RegenerateRecoveryCodesRequest]) (*connect.Response[v1pb.RegenerateRecoveryCodesResponse], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	if caller.MFAConfig.GetOtpSecret() == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("MFA is not enabled"))
	}
	tempRecoveryCodes, err := generateRecoveryCodes(10)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate recovery codes"))
	}
	version := timestamppb.Now()

	if _, err := s.store.UpdateUser(ctx, caller, &store.UpdateUserMessage{
		MFAConfig: &storepb.MFAConfig{
			OtpSecret:                caller.MFAConfig.GetOtpSecret(),
			RecoveryCodes:            caller.MFAConfig.GetRecoveryCodes(),
			TempRecoveryCodes:        tempRecoveryCodes,
			TempOtpSecretCreatedTime: version,
		},
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}

	return connect.NewResponse(&v1pb.RegenerateRecoveryCodesResponse{
		RecoveryCodes:  tempRecoveryCodes,
		PendingVersion: version,
	}), nil
}

// ConfirmRecoveryCodes promotes the pending recovery codes. On a first-time
// enrollment the pending TOTP secret goes live in the same write, so the factor
// and the codes that recover it start existing together.
func (s *UserService) ConfirmRecoveryCodes(ctx context.Context, request *connect.Request[v1pb.ConfirmRecoveryCodesRequest]) (*connect.Response[v1pb.User], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	if len(caller.MFAConfig.GetTempRecoveryCodes()) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("no pending recovery codes to confirm"))
	}

	mfaConfig := &storepb.MFAConfig{
		OtpSecret:     caller.MFAConfig.GetOtpSecret(),
		RecoveryCodes: caller.MFAConfig.GetTempRecoveryCodes(),
	}
	// A pending secret means an enrollment — the account's first factor, or a
	// replacement for the one it already has — and either way that secret is
	// the one the caller just proved they can generate codes from, so it is
	// the one that goes live. Only RegenerateRecoveryCodes leaves no pending
	// secret, and only there does the live factor survive untouched.
	if pendingSecret := caller.MFAConfig.GetTempOtpSecret(); pendingSecret != "" {
		if isMFATempSecretExpired(caller.MFAConfig) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA setup has expired, please restart the enrollment"))
		}
		mfaConfig.OtpSecret = pendingSecret
	} else if mfaConfig.OtpSecret == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("no MFA enrollment in progress, call StartMFAEnrollment first"))
	}

	// The pending state is one shared slot, so two flows can race for it, and
	// the version predicate rides on the write rather than sitting in front of
	// it: a mint from another tab, or an administrator's DisableMFA, can land
	// between a check and an unconditional write, and this request would then
	// promote a secret the caller never saw or revive a factor that
	// administrator just cleared.
	user, err := s.store.UpdateUserMFAConfigIfPending(ctx, caller.ID, request.Msg.PendingVersion.AsTime(), mfaConfig)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}
	if user == nil {
		return nil, errSupersededPendingMFAState
	}
	return s.convertUserResponse(ctx, user)
}

// checkPendingMFAState rejects a confirmation whose pending_version is no
// longer the account's pending state — a later mint superseded the enrollment
// or code set this request thinks it is confirming, or a DisableMFA cleared it.
func checkPendingMFAState(user *store.UserMessage, pendingVersion *timestamppb.Timestamp) error {
	stored := user.MFAConfig.GetTempOtpSecretCreatedTime()
	if stored == nil || !stored.AsTime().Equal(pendingVersion.AsTime()) {
		return errSupersededPendingMFAState
	}
	return nil
}

// resolveSelfUser parses name and requires it to be the caller's own account.
// Nothing upstream enforces this for CUSTOM methods, so every self-service
// method goes through here.
func (*UserService) resolveSelfUser(ctx context.Context, name string) (*store.UserMessage, error) {
	caller, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	email, err := common.GetUserEmail(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}
	if caller.Email != email {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("name must be the caller's own account"))
	}
	return caller, nil
}

func (s *UserService) convertUserResponse(ctx context.Context, user *store.UserMessage) (*connect.Response[v1pb.User], error) {
	v1User, err := convertToUser(ctx, s.iamManager, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	return connect.NewResponse(v1User), nil
}

func convertToUser(ctx context.Context, iamManager *iam.Manager, user *store.UserMessage) (*v1pb.User, error) {
	groups, err := iamManager.GetUserGroups(ctx, common.GetWorkspaceIDFromContext(ctx), user.Email)
	if err != nil {
		return nil, err
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	convertedUser := &v1pb.User{
		Name:  common.FormatUserEmail(user.Email),
		State: convertDeletedToState(user.MemberDeleted),
		Email: user.Email,
		Phone: user.Phone,
		Title: user.Name,
		Profile: &v1pb.User_Profile{
			LastLoginTime:          user.Profile.LastLoginTime,
			LastChangePasswordTime: user.Profile.LastChangePasswordTime,
			Source:                 user.Profile.Source,
		},
		Groups:    groups,
		Workspace: common.FormatWorkspace(workspaceID),
	}

	if user.MFAConfig != nil {
		convertedUser.MfaEnabled = user.MFAConfig.OtpSecret != ""
	}
	return convertedUser, nil
}

func validateEndUserEmail(email string) error {
	if common.IsServiceAccountEmail(email) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email for end users cannot end with %v", common.ServiceAccountSuffix))
	}
	if common.IsWorkloadIdentityEmail(email) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email for end users cannot end with %v", common.WorkloadIdentitySuffix))
	}
	// Check if the email is valid.
	if err := common.ValidateEmail(email); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid email: %v", err.Error()))
	}
	return nil
}

func validateEmailWithDomains(ctx context.Context, licenseService *enterprise.LicenseService, stores *store.Store, workspaceID, email string, checkDomainSetting bool) error {
	if err := validateEndUserEmail(email); err != nil {
		return err
	}
	if licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_USER_EMAIL_DOMAIN_RESTRICTION) != nil {
		// nolint:nilerr
		return nil
	}
	setting, err := stores.GetWorkspaceProfileSetting(ctx, workspaceID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting"))
	}

	var allowedDomains []string
	if checkDomainSetting || setting.EnforceIdentityDomain {
		allowedDomains = setting.Domains
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
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email %q does not belong to allowed domains", email))
		}
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

// countUsersInIamPolicy counts distinct user members in an IAM policy,
// expanding group memberships. When allUsers is present and not in SaaS mode,
// returns the total active principal count instead. Soft-deleted principals do
// not occupy a seat, so emails whose principal is deleted are excluded; emails
// without a principal yet (e.g. pending SaaS invites) are still counted.
func countUsersInIamPolicy(ctx context.Context, s *store.Store, workspaceID string, policy *storepb.IamPolicy, saas bool) (int, error) {
	emails := make(map[string]struct{})
	var groupRefs []string
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member == common.AllUsers {
				if !saas {
					return s.CountActivePrincipals(ctx)
				}
				continue
			}
			if strings.HasPrefix(member, "users/") {
				// Principal emails are stored lower-cased, but IAM members may keep
				// mixed casing; normalize so dedup and the deleted lookup below match.
				emails[strings.ToLower(strings.TrimPrefix(member, "users/"))] = struct{}{}
			} else if strings.HasPrefix(member, "groups/") {
				groupRefs = append(groupRefs, strings.TrimPrefix(member, "groups/"))
			}
		}
	}
	for _, ref := range groupRefs {
		members, _ := s.GetGroupMembersSnapshot(ctx, workspaceID, "groups/"+ref)
		for member := range members {
			if strings.HasPrefix(member, "users/") {
				emails[strings.ToLower(strings.TrimPrefix(member, "users/"))] = struct{}{}
			}
		}
	}
	if len(emails) == 0 {
		return 0, nil
	}
	emailList := make([]string, 0, len(emails))
	for email := range emails {
		emailList = append(emailList, email)
	}
	// Pass an empty workspace to look up principals by email only: the seat set is
	// already derived from policy, we just need each principal's deleted state.
	users, err := s.BatchGetUsersByEmails(ctx, "", emailList)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to batch get users by emails")
	}
	deleted := 0
	for _, user := range users {
		if user.MemberDeleted {
			if _, ok := emails[user.Email]; ok {
				deleted++
			}
		}
	}
	return len(emails) - deleted, nil
}

// userCountGuard checks seat limits before adding a new IAM member (e.g. SSO login).
// Uses >= because the new user has not been counted yet.
// If policy is nil, reads the current workspace IAM policy.
func userCountGuard(ctx context.Context, s *store.Store, licenseService *enterprise.LicenseService, workspaceID string, policy *storepb.IamPolicy, saas bool) error {
	if policy == nil {
		p, err := s.GetWorkspaceIamPolicy(ctx, workspaceID)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get workspace IAM policy"))
		}
		policy = p.Policy
	}
	userLimit := licenseService.GetUserLimit(ctx, workspaceID)
	count, err := countUsersInIamPolicy(ctx, s, workspaceID, policy, saas)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to count users in IAM policy"))
	}
	if count >= userLimit {
		return connect.NewError(connect.CodeResourceExhausted, errors.Errorf("workspace has %d users, reaching the limit of %d", count, userLimit))
	}
	return nil
}

// preAddUserGuard checks seat limits before creating a principal.
// Only enforces when the IAM policy contains allUsers, because without allUsers
// a new principal does not occupy a seat until explicitly added to IAM.
func (s *UserService) preAddUserGuard(ctx context.Context, workspaceID string) error {
	p, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get workspace IAM policy"))
	}
	if !policyContainsAllUsers(p.Policy) {
		return nil
	}
	return userCountGuard(ctx, s.store, s.licenseService, workspaceID, p.Policy, s.profile.SaaS)
}

// preUndeleteUserGuard checks seat limits before reactivating a principal.
// A soft-deleted principal does not occupy a seat, so undeleting one that is
// already referenced by workspace IAM (via allUsers, a direct binding, or a
// group) re-occupies a seat and must respect the limit. When the principal is
// not referenced, undeleting adds no seat and is always allowed.
func (s *UserService) preUndeleteUserGuard(ctx context.Context, workspaceID string, user *store.UserMessage) error {
	p, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get workspace IAM policy"))
	}
	if len(utils.GetUserIAMPolicyBindings(ctx, s.store, workspaceID, user, p.Policy)) == 0 {
		return nil
	}
	return userCountGuard(ctx, s.store, s.licenseService, workspaceID, p.Policy, s.profile.SaaS)
}

func policyContainsAllUsers(policy *storepb.IamPolicy) bool {
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member == common.AllUsers {
				return true
			}
		}
	}
	return false
}

func isUserWorkspaceAdmin(ctx context.Context, stores *store.Store, user *store.UserMessage, workspaceID string) (bool, error) {
	workspacePolicy, err := stores.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	roles := utils.GetUserFormattedRolesMap(ctx, stores, workspaceID, user, workspacePolicy.Policy)
	return roles[common.FormatRole(store.WorkspaceAdminRole)], nil
}
