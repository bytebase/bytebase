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
	secret         string
	licenseService *enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewUserService creates a new UserService.
func NewUserService(store *store.Store, secret string, licenseService *enterprise.LicenseService, profile *config.Profile, iamManager *iam.Manager) *UserService {
	return &UserService{
		store:          store,
		secret:         secret,
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
	// Extract and validate emails from names.
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
	// The store returns its own order; answer in request order.
	userByEmail := make(map[string]*store.UserMessage, len(users))
	for _, user := range users {
		userByEmail[strings.ToLower(user.Email)] = user
	}

	response := &v1pb.BatchGetUsersResponse{}
	for i, name := range request.Msg.Names {
		user, ok := userByEmail[strings.ToLower(emails[i])]
		if !ok {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", name))
		}
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

	var newPassword *string
	patch := &store.UpdateUserMessage{}
	for _, path := range request.Msg.UpdateMask.Paths {
		switch path {
		case "email":
			// Email updates are not supported through UpdateUser. Use UpdateEmail API instead.
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email updates are not supported through UpdateUser, use UpdateEmail API instead"))
		case "title":
			patch.Name = &request.Msg.User.Title
		case "password":
			// Admin-assisted reset only: an admin recovering a locked-out user
			// cannot know the credential being replaced, and bb.users.update
			// plus the audit log are the control there. A caller changing
			// their own password must prove the current one via ChangePassword.
			if callerUser.ID == user.ID {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("use ChangePassword to change your own password"))
			}
			if user.Type != storepb.PrincipalType_END_USER {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password can be mutated for end users only"))
			}
			if err := s.validatePassword(ctx, common.GetWorkspaceIDFromContext(ctx), request.Msg.User.Password); err != nil {
				return nil, err
			}
			newPassword = &request.Msg.User.Password
		case "mfa_enabled":
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("mfa_enabled is output only, use EnableMFA or DisableMFA instead"))
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
	if newPassword != nil {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(*newPassword)); err == nil {
			// return bad request if the passwords match
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password cannot be the same"))
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(*newPassword), bcrypt.DefaultCost)
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

// Self-service credential changes (docs/design/reauthenticate-credential-changes.md).
//
// Every mutation that rewrites live authentication material — password hash,
// permanent TOTP secret, permanent recovery codes — requires a CredentialProof
// that the caller currently holds a credential on the account. A password
// change additionally ends the account's other web sessions.
//
// These methods are auth_method=CUSTOM: the ACL interceptor enforces nothing
// for them, so each handler independently rejects any name other than the
// caller's own. DisableMFA is the one method with a real admin path.

// errFirstTimeEnrollmentNeedsPassword is the refusal a self-hosted first-time
// enrollment gets when it carries no proof. It names both ways out, because
// the server cannot tell which applies: an account whose owner chose a
// password proves it directly, while an SSO-provisioned one holds a random
// password nobody was ever told and needs an administrator to reset it first.
var errFirstTimeEnrollmentNeedsPassword = connect.NewError(connect.CodeFailedPrecondition,
	errors.New("enrolling MFA requires your current password; if you sign in through SSO and never chose one, ask a workspace admin to reset your password, then enroll with it"))

// RequestReauthCode sends a one-time REAUTH code to the caller's own email,
// usable as CredentialProof.email_code. Bytebase Cloud only: self-hosted
// deployments cannot send email, so the eligibility check below refuses them
// before any mail machinery runs.
//
// The signed-in half of the pair AuthService.SendEmailLoginCode completes:
// that one gets a caller into an account and is unauthenticated, this one
// proves the caller already holds the account. Same mail machinery, different
// purpose row — a LOGIN code is not spendable here, and vice versa.
func (s *UserService) RequestReauthCode(ctx context.Context, request *connect.Request[v1pb.RequestReauthCodeRequest]) (*connect.Response[emptypb.Empty], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	// Eligibility is enforced again at redemption — verifyCredentialProof's
	// email_code branch calls checkEmailCodeEligible — so this check is not
	// the gate. It is here to refuse ineligible callers outright rather than
	// mailing them a code they could never spend.
	if err := s.checkEmailCodeEligible(caller); err != nil {
		return nil, err
	}

	// Unlike RequestPasswordReset, a send failure is surfaced: the caller is
	// already authenticated as the only account this could be about, so there
	// is no existence oracle to protect — only a dead end to diagnose.
	if err := sendEmailVerificationCode(
		ctx,
		s.store,
		s.secret,
		common.GetWorkspaceIDFromContext(ctx),
		caller.Email,
		storepb.EmailVerificationCodePurpose_REAUTH,
		emailCodeTemplate{
			Subject: "[Bytebase] Confirm your identity",
			BodyFmt: "Hi,\n\nYour verification code is: %s\n\nThis code expires in %d minutes. If you didn't request this, you can safely ignore this email.\n\n— Bytebase",
		},
		// Unbudgeted: the caller is already authenticated and the recipient is
		// their own address, so this is not a path an anonymous sender can point
		// at a recipient list.
		"", 0,
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ChangePassword changes the caller's own password after verifying a
// CredentialProof, and ends the account's web sessions — the caller's own
// included. OAuth grants deliberately survive: widening revocation to MCP
// grants is a separate change with its own blast radius.
func (s *UserService) ChangePassword(ctx context.Context, request *connect.Request[v1pb.ChangePasswordRequest]) (*connect.Response[v1pb.User], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	restriction, err := getAccountRestriction(ctx, s.store, s.licenseService, s.profile.SaaS, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, err
	}
	if restriction.DisallowPasswordSignin {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("password sign-in is disabled for this workspace"))
	}
	if err := s.validatePassword(ctx, common.GetWorkspaceIDFromContext(ctx), request.Msg.NewPassword); err != nil {
		return nil, err
	}
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(request.Msg.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate password hash"))
	}

	claimed, err := s.claimCredentialProof(ctx, caller.Email, request.Msg.Credential, false)
	if err != nil {
		return nil, err
	}
	// Changing the password is not touching the MFA factor, so
	// current_password is acceptable with or without live MFA. A recovery-code
	// proof is spent by the verify itself, so nothing about mfa_config is
	// rewritten here from a read taken before it.
	if err := s.verifyCredentialProof(ctx, caller, request.Msg.Credential, false); err != nil {
		return nil, err
	}
	// Proven, so its bucket has done its job. Every refusal below this line is
	// about the request, not the credential, and must not count against the
	// bucket an ordinary login draws from.
	s.clearCredentialProofClaims(ctx, caller.Email, claimed)
	claimed = nil

	if bcrypt.CompareHashAndPassword([]byte(caller.PasswordHash), []byte(request.Msg.NewPassword)) == nil {
		// Not just UX: any PasswordHash write moves the password-rotation
		// deadline, which a same-password "change" must not reset.
		//
		// This comparison answers "is this the current password?", so it stays
		// behind the proof. Reached without one it is an unbounded password
		// oracle: a stolen session exhausts the lockout bucket, then reads the
		// answer off which error comes back for each candidate new_password.
		//
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password cannot be the same"))
	}

	profile := caller.Profile
	profile.LastChangePasswordTime = timestamppb.Now()
	user, err := s.store.UpdateUser(ctx, caller, &store.UpdateUserMessage{
		PasswordHash: new(string(newPasswordHash)),
		Profile:      profile,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}
	// Every session of this account, the caller's own included, has to be
	// re-established with the new password.
	if err := s.store.DeleteWebRefreshTokensByUser(ctx, user.Email); err != nil {
		slog.Error("failed to revoke refresh tokens on password change", log.BBError(err), slog.String("user", user.Email))
	}
	s.clearCredentialProofClaims(ctx, caller.Email, claimed)
	return s.convertUserResponse(ctx, user)
}

// StartMFAEnrollment mints a pending TOTP secret and recovery codes. Nothing
// goes live, so no proof is required. The write goes through
// SetPendingMFAState, which sets the three pending fields on the row itself
// rather than writing back a whole config assembled from this read — a
// read-modify-write here would resurrect codes a concurrent confirmation
// rotated away.
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
	// Microseconds, not nanoseconds: this value is handed to the caller,
	// stored, and later matched as a Postgres timestamptz, which cannot hold
	// a finer instant. Minting one the store cannot represent would make the
	// confirmation reject the enrollment it just created.
	version := timestamppb.New(time.Now().Truncate(time.Microsecond))

	// The live factor is carried forward untouched: minting an enrollment
	// changes nothing about how the account signs in today.
	if err := s.store.SetPendingMFAState(ctx, caller.ID, tempSecret, tempRecoveryCodes, version.AsTime()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}

	return connect.NewResponse(&v1pb.StartMFAEnrollmentResponse{
		OtpSecret:      tempSecret,
		RecoveryCodes:  tempRecoveryCodes,
		ExpireTime:     timestamppb.New(version.AsTime().Add(mfaTempSecretExpiration)),
		PendingVersion: version,
	}), nil
}

// EnableMFA verifies the pending enrollment's otp_code and writes nothing —
// for a rotation as much as for a first enrollment. ConfirmRecoveryCodes owns
// promotion in both flows, where the secret goes live in the same write as the
// codes that recover it, so an account is never MFA-required with none saved
// and a rotation is not half-applied if the user abandons the download step.
// Nothing is revoked either: MFA gates minting a session, not using one.
func (s *UserService) EnableMFA(ctx context.Context, request *connect.Request[v1pb.EnableMFARequest]) (*connect.Response[v1pb.User], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	if request.Msg.PendingVersion == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("pending_version is required"))
	}
	// Non-guess refusals run before the lockout claim: an argument or state
	// error is not a guess (the SwitchWorkspace MFA step states the same
	// rule), and a stale-tab retry against an expired enrollment must not
	// consume slots from the bucket real MFA logins draw from.
	if err := checkEnableMFAConfirmable(caller, request.Msg.PendingVersion); err != nil {
		return nil, err
	}
	if err := s.checkEnableMFACredentialShape(caller, request.Msg.Credential); err != nil {
		return nil, err
	}

	// The enrollment otp_code claims the MFA bucket even when no
	// CredentialProof is present at all, so verifying against the pending
	// TOTP seed is never an unbounded guessing oracle.
	claimed, err := s.claimCredentialProof(ctx, caller.Email, request.Msg.Credential, true)
	if err != nil {
		return nil, err
	}

	// The shape check above admits a credential exactly when one is required:
	// replacing a live factor is proven with the factor, and a first-time
	// enrollment on self-hosted with the password.
	if request.Msg.Credential != nil {
		rotation := caller.MFAConfig.GetOtpSecret() != ""
		if err := s.verifyCredentialProof(ctx, caller, request.Msg.Credential, true); err != nil {
			return nil, firstTimeEnrollmentProofError(err, rotation, request.Msg.Credential)
		}
		// Proven, so its bucket has done its job. Releasing it here rather
		// than at the end keeps a mistyped new-device code below from
		// draining the bucket an ordinary login draws from.
		s.clearCredentialProofClaims(ctx, caller.Email, claimed)
		claimed = nil
	}

	// The enrollment code proves the caller configured the new device
	// correctly; skipping it would leave a typo'd authenticator undetected
	// until the user is locked out at their next login.
	if !totp.Validate(request.Msg.OtpCode, caller.MFAConfig.GetTempOtpSecret()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid OTP code"))
	}

	// Verification only, so this writes nothing: promotion belongs to
	// ConfirmRecoveryCodes, atomically with the codes and under the version
	// predicate, so the account is never MFA-required with none saved.
	s.clearCredentialProofClaims(ctx, caller.Email, claimed)
	return s.convertUserResponse(ctx, caller)
}

// DisableMFA turns MFA off by clearing the entire MFAConfig — live and
// pending state alike, so a stale confirmation cannot silently re-enable it.
// Self-service requires proof by the factor itself; the admin path
// (self-hosted only) requires bb.users.update and no proof, since an admin
// recovering a locked-out user cannot know the factor being removed.
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

	selfService := callerUser.ID == user.ID
	if !selfService {
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

	// Only a live factor has to be proven. With none, this call is cancelling
	// a pending enrollment or is a no-op, and there is nothing to prove with:
	// requiring a factor here would strand a user who abandoned an enrollment.
	var claimed []storepb.LoginAttemptKind
	if selfService && user.MFAConfig.GetOtpSecret() != "" {
		if request.Msg.Credential == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("credential is required"))
		}
		if err := s.checkFactorProofShape(user, request.Msg.Credential); err != nil {
			return nil, err
		}
		claimed, err = s.claimCredentialProof(ctx, user.Email, request.Msg.Credential, false)
		if err != nil {
			return nil, err
		}
		// Turning the factor off must be proven with the factor: a password
		// can be minted from mailbox possession alone (ResetPassword), so
		// accepting one here would let a stolen session plus a mailbox strip
		// the second factor.
		if err := s.verifyCredentialProof(ctx, user, request.Msg.Credential, true); err != nil {
			return nil, err
		}
		// See ChangePassword: released the moment it is proven, so a failed
		// write below cannot charge the caller for a correct proof.
		s.clearCredentialProofClaims(ctx, user.Email, claimed)
		claimed = nil
	}
	// The admin path stays idempotent: a retry, or a second admin racing the
	// first, finds MFA already off and still reaches the desired state —
	// erroring would report failure for a satisfied intent. The wipe clears
	// any pending enrollment either way.
	updatedUser, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
		MFAConfig: &storepb.MFAConfig{},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}
	s.clearCredentialProofClaims(ctx, user.Email, claimed)
	return s.convertUserResponse(ctx, updatedUser)
}

// RegenerateRecoveryCodes mints a pending recovery-code set alongside the
// still-live old one. Nothing goes live, so no proof — same as
// StartMFAEnrollment, and locked for the same reason.
func (s *UserService) RegenerateRecoveryCodes(ctx context.Context, request *connect.Request[v1pb.RegenerateRecoveryCodesRequest]) (*connect.Response[v1pb.RegenerateRecoveryCodesResponse], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	tempRecoveryCodes, err := generateRecoveryCodes(10)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate recovery codes"))
	}
	version := timestamppb.New(time.Now().Truncate(time.Microsecond))

	if caller.MFAConfig.GetOtpSecret() == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("MFA is not enabled"))
	}
	// Minting into the shared temp state supersedes any in-flight TOTP
	// rotation; its EnableMFA fails the pending_version check rather than
	// promoting against a half-overwritten set. The empty pending secret is
	// deliberate: these codes belong to the live factor, so confirming them
	// must not also promote a secret an abandoned enrollment left behind.
	if err := s.store.SetPendingMFAState(ctx, caller.ID, "", tempRecoveryCodes, version.AsTime()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}

	return connect.NewResponse(&v1pb.RegenerateRecoveryCodesResponse{
		RecoveryCodes:  tempRecoveryCodes,
		PendingVersion: version,
	}), nil
}

// ConfirmRecoveryCodes promotes the exact pending set named by
// pending_version. For a rotation it replaces the live recovery codes; for
// first-time enrollment it promotes the TOTP secret and recovery codes
// together, gated on a fresh otp_code. Both branches change live credential
// material, so both revoke sessions.
func (s *UserService) ConfirmRecoveryCodes(ctx context.Context, request *connect.Request[v1pb.ConfirmRecoveryCodesRequest]) (*connect.Response[v1pb.User], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	if request.Msg.Credential == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("credential is required"))
	}
	if request.Msg.PendingVersion == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("pending_version is required"))
	}
	// Non-guess refusals before the claim, same rule as EnableMFA.
	firstTime := caller.MFAConfig.GetOtpSecret() == ""
	if err := checkRecoveryCodesConfirmable(caller, request.Msg.PendingVersion, request.Msg.OtpCode, firstTime); err != nil {
		return nil, err
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

	if err := s.checkFactorProofShape(caller, request.Msg.Credential); err != nil {
		return nil, err
	}
	claimed, err := s.claimCredentialProof(ctx, caller.Email, request.Msg.Credential, request.Msg.OtpCode != "")
	if err != nil {
		return nil, err
	}

	if firstTime {
		// First-time enrollment: this is where the secret and the codes go
		// live, so this is where the proof binds. A fresh otp_code, not a
		// replay of EnableMFA's — promotion is the moment MFA starts gating
		// logins.
		if err := s.verifyCredentialProof(ctx, caller, request.Msg.Credential, true); err != nil {
			return nil, firstTimeEnrollmentProofError(err, false, request.Msg.Credential)
		}
		// Released once proven, so a mistyped new-device code costs the user
		// nothing but a retry. See EnableMFA.
		s.clearCredentialProofClaims(ctx, caller.Email, claimed)
		claimed = nil
		if !totp.Validate(request.Msg.OtpCode, caller.MFAConfig.GetTempOtpSecret()) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid OTP code"))
		}
	} else {
		if err := s.verifyCredentialProof(ctx, caller, request.Msg.Credential, true); err != nil {
			return nil, err
		}
		// Same release as the first-time branch above: the compare-and-set
		// below can lose to another tab's mint or an administrator's
		// DisableMFA, and losing that race is not a failed proof.
		s.clearCredentialProofClaims(ctx, caller.Email, claimed)
		claimed = nil
	}

	// The pending state is one shared slot, so two flows can race for it, and
	// the version predicate rides on the write rather than sitting in front of
	// it: a mint from another tab, or an administrator's DisableMFA, can land
	// between the check above and an unconditional write, and this request
	// would then promote a secret the caller never saw or revive a factor that
	// administrator just cleared.
	//
	// Turning MFA on, or rotating the codes, is not a reason to sign the
	// account out of its other devices.
	user, err := s.store.UpdateUserMFAConfigIfPending(ctx, caller.ID, request.Msg.PendingVersion.AsTime(), mfaConfig)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}
	if user == nil {
		return nil, errSupersededPendingMFAState
	}
	s.clearCredentialProofClaims(ctx, caller.Email, claimed)
	return s.convertUserResponse(ctx, user)
}

// resolveSelfUser parses name and requires it to be the caller's own account.
// Nothing upstream enforces this for CUSTOM methods, so every self-service
// credential method goes through here.
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

// claimCredentialProof claims the T9 lockout slot for the credential this
// request will verify, before any verification runs — a proof channel that
// verified first and counted afterwards would let a caller abandon the request
// on failure and guess without bound.
//
// Exactly one slot, never two. An enrollment otp_code is checked against a
// secret StartMFAEnrollment just handed this same caller, so it is not a
// secret to guess; it needs the bound only when it is the one thing being
// verified, which is Cloud first-time enrollment. Claiming a second bucket
// alongside a real proof would also mean a refusal on that second claim leaves
// the first one counted against a request that verified nothing, draining an
// unrelated bucket on every retry.
func (s *UserService) claimCredentialProof(ctx context.Context, email string, credential *v1pb.CredentialProof, verifiesEnrollmentOtp bool) ([]storepb.LoginAttemptKind, error) {
	var kinds []storepb.LoginAttemptKind
	if credential != nil {
		kind, err := credentialProofKind(credential)
		if err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	} else if verifiesEnrollmentOtp {
		kinds = append(kinds, storepb.LoginAttemptKind_MFA)
	}
	for _, kind := range kinds {
		if err := claimLoginAttempt(ctx, s.store, email, kind); err != nil {
			return nil, err
		}
	}
	return kinds, nil
}

// clearCredentialProofClaims forgets the request's claims after every
// verification in it succeeded.
func (s *UserService) clearCredentialProofClaims(ctx context.Context, email string, kinds []storepb.LoginAttemptKind) {
	for _, kind := range kinds {
		clearLoginAttempt(ctx, s.store, email, kind)
	}
}

func credentialProofKind(credential *v1pb.CredentialProof) (storepb.LoginAttemptKind, error) {
	switch credential.GetProof().(type) {
	case *v1pb.CredentialProof_CurrentPassword:
		return storepb.LoginAttemptKind_PASSWORD, nil
	case *v1pb.CredentialProof_OtpCode, *v1pb.CredentialProof_RecoveryCode:
		return storepb.LoginAttemptKind_MFA, nil
	case *v1pb.CredentialProof_EmailCode:
		return storepb.LoginAttemptKind_EMAIL_CODE, nil
	default:
		return storepb.LoginAttemptKind_LOGIN_ATTEMPT_KIND_UNSPECIFIED, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("credential must set exactly one proof"))
	}
}

// verifyCredentialProof verifies one CredentialProof against the account. touchesFactor marks methods that
// rewrite the MFA factor itself: while a live factor exists they accept only
// otp_code or recovery_code — current_password is not independent (a
// ResetPassword mints one from mailbox possession alone) and email_code is
// strictly weaker than the factor.
//
// Single-use proofs are spent here, in the same step that accepts them, the
// way the login path spends a recovery code: `user` is the pointer the store's
// user cache handed out, so consuming one by editing it in place and leaving
// the write to a caller would desynchronize that cache from the row on every
// path that then writes nothing, or whose write fails.
func (s *UserService) verifyCredentialProof(ctx context.Context, user *store.UserMessage, credential *v1pb.CredentialProof, touchesFactor bool) error {
	liveMFA := user.MFAConfig.GetOtpSecret() != ""
	// GetProof, not the field: a nil credential must land in the default
	// case, never dereference.
	switch proof := credential.GetProof().(type) {
	case *v1pb.CredentialProof_CurrentPassword:
		if touchesFactor && liveMFA {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("changing the MFA factor requires an OTP or recovery code, not the password"))
		}
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(proof.CurrentPassword)) != nil {
			return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("invalid password"))
		}
		return nil
	case *v1pb.CredentialProof_OtpCode:
		if !liveMFA {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA is not enabled"))
		}
		if !totp.Validate(proof.OtpCode, user.MFAConfig.GetOtpSecret()) {
			return connect.NewError(connect.CodeUnauthenticated, errors.Errorf(errMsgInvalidMFACode))
		}
		return nil
	case *v1pb.CredentialProof_RecoveryCode:
		if !liveMFA {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA is not enabled"))
		}
		// The row decides, not the list read into memory: the spend is the
		// predicate on the UPDATE, so a code cannot be accepted twice.
		consumed, err := s.store.ConsumeRecoveryCode(ctx, user.ID, proof.RecoveryCode)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to verify recovery code"))
		}
		if !consumed {
			return connect.NewError(connect.CodeUnauthenticated, errors.Errorf(errMsgInvalidRecoveryCode))
		}
		return nil
	case *v1pb.CredentialProof_EmailCode:
		if err := s.checkEmailCodeEligible(user); err != nil {
			return err
		}
		// A REAUTH code is single-use, and the spend is the same statement
		// that matches it: validating a row that was read first would let two
		// concurrent requests both accept one code, each deleting a row the
		// other had already used to authorize a credential change.
		consumed, err := s.store.ConsumeEmailVerificationCode(ctx, user.Email, storepb.EmailVerificationCodePurpose_REAUTH, hashEmailCode(s.secret, proof.EmailCode))
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to verify code"))
		}
		if !consumed {
			// Wrong, already spent, or expired — indistinguishable on purpose.
			return connect.NewError(connect.CodeUnauthenticated, errors.Errorf(errMsgInvalidEmailCode))
		}
		return nil
	default:
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("credential must set exactly one proof"))
	}
}

// checkEmailCodeEligible enforces email_code's server-side eligibility rule:
// Bytebase Cloud only, and never against a live MFA factor. A mailbox code is
// bootstrap proof for an account that has nothing else to prove — on Cloud,
// where no caller-chosen password can exist, that is every account without
// MFA — and never a substitute for a credential that does exist.
//
// The Cloud restriction is scope, not capability: a self-hosted workspace with
// its mail delivery setting configured sends password-reset codes today and
// could send these. It is withheld there because self-hosted keeps an
// administrator who can reset a password, which is the recovery route
// errFirstTimeEnrollmentNeedsPassword names.
func (s *UserService) checkEmailCodeEligible(user *store.UserMessage) error {
	if !s.profile.SaaS {
		return connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("email verification codes are only available on Bytebase Cloud"))
	}
	if user.MFAConfig.GetOtpSecret() != "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("this account has MFA enabled; prove it with an OTP or recovery code"))
	}
	return nil
}

// checkPendingMFAState rejects a confirmation whose pending_version no longer
// matches the account's pending MFA state — a later mint superseded the set
// this request thinks it is confirming, or a DisableMFA wiped it.
var errSupersededPendingMFAState = connect.NewError(connect.CodeFailedPrecondition,
	errors.New("the pending MFA state has been superseded; restart from the mint step"))

func checkPendingMFAState(user *store.UserMessage, pendingVersion *timestamppb.Timestamp) error {
	stored := user.MFAConfig.GetTempOtpSecretCreatedTime()
	if stored == nil || !stored.AsTime().Equal(pendingVersion.AsTime()) {
		return errSupersededPendingMFAState
	}
	return nil
}

// checkEnableMFAConfirmable refuses the stale-enrollment states EnableMFA
// cannot act on: a superseded pending version, no enrollment in progress, or
// an expired one.
func checkEnableMFAConfirmable(user *store.UserMessage, pendingVersion *timestamppb.Timestamp) error {
	if err := checkPendingMFAState(user, pendingVersion); err != nil {
		return err
	}
	if user.MFAConfig.GetTempOtpSecret() == "" {
		return connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("no MFA enrollment in progress, call StartMFAEnrollment first"))
	}
	if isMFATempSecretExpired(user.MFAConfig) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA setup has expired, please restart the enrollment"))
	}
	return nil
}

// checkEnableMFACredentialShape refuses an EnableMFA whose credential does not
// fit the account's shape. Replacing a live factor must be proven with the
// factor; a first-time enrollment on self-hosted must be proven with the
// password, which is also the only proof channel there.
//
// Whether the caller has a password is answered by whether they can supply
// one, not by a stored timestamp. Inferring it from lastChangePasswordTime
// meant an SSO account — whose password is a random string nobody was ever
// told — read as "has a password" and was asked to prove it, which is a dead
// end rather than an answer. The refusal below names the route out instead.
func (s *UserService) checkEnableMFACredentialShape(user *store.UserMessage, credential *v1pb.CredentialProof) error {
	if user.MFAConfig.GetOtpSecret() != "" || !s.profile.SaaS {
		if credential == nil {
			return errFirstTimeEnrollmentNeedsPassword
		}
		return s.checkFactorProofShape(user, credential)
	}
	if credential != nil {
		// Cloud passwordless enrollment: email_code would be the only
		// possible proof, and it is single-use — consumed once, at
		// ConfirmRecoveryCodes, where the mutation actually happens. This
		// call writes nothing for this account shape, so it needs no proof
		// at all.
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("credential must be omitted for first-time enrollment on an account without a password"))
	}
	return nil
}

// checkFactorProofShape refuses, before any slot is claimed, a proof channel
// that a live factor could never accept. verifyCredentialProof rejects these
// too, but it does so after the claim — and a request naming a channel this
// account cannot use has guessed at nothing, so five of them must not lock the
// bucket ordinary password or email-code logins draw from.
func (s *UserService) checkFactorProofShape(user *store.UserMessage, credential *v1pb.CredentialProof) error {
	if credential == nil {
		return nil
	}
	if user.MFAConfig.GetOtpSecret() != "" {
		switch credential.GetProof().(type) {
		case *v1pb.CredentialProof_OtpCode, *v1pb.CredentialProof_RecoveryCode:
			return nil
		default:
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("changing the MFA factor requires an OTP or recovery code, not the password"))
		}
	}
	// No live factor, so exactly one channel can succeed and it is decided by
	// the deployment: self-hosted accounts prove the password they have, Cloud
	// accounts an emailed code because they never have a caller-chosen one.
	if !s.profile.SaaS {
		if _, ok := credential.GetProof().(*v1pb.CredentialProof_CurrentPassword); !ok {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("this account has no MFA factor; prove it with your current password"))
		}
		return nil
	}
	if _, ok := credential.GetProof().(*v1pb.CredentialProof_EmailCode); !ok {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("this account has no MFA factor or password; prove it with an emailed code"))
	}
	return nil
}

// checkRecoveryCodesConfirmable refuses the state and argument shapes
// ConfirmRecoveryCodes cannot act on, for both branches.
func checkRecoveryCodesConfirmable(user *store.UserMessage, pendingVersion *timestamppb.Timestamp, otpCode string, firstTime bool) error {
	if err := checkPendingMFAState(user, pendingVersion); err != nil {
		return err
	}
	if len(user.MFAConfig.GetTempRecoveryCodes()) == 0 {
		return connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("no pending recovery codes to confirm"))
	}
	if !firstTime {
		// The live factor is the proof for a rotation or a replacement; the
		// new device was already validated by EnableMFA.
		if otpCode != "" {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("otp_code is only for first-time enrollment; use credential to prove the existing factor"))
		}
		return nil
	}
	// A fresh code, not a replay of EnableMFA's: TOTP codes outlive one ~30s
	// window and the download screen in between routinely takes longer — and
	// promotion is exactly the moment MFA goes live.
	if otpCode == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("otp_code is required to complete first-time enrollment"))
	}
	return nil
}

// firstTimeEnrollmentProofError makes a failed first-time-enrollment password
// proof actionable. The 3.22 backfill stamps every pre-existing account as
// having a chosen password — including SSO-provisioned ones whose password is
// a random hash nobody knows — so a failed compare here may mean "no password
// exists to know", and the recovery route is the same admin reset an
// unstamped account is pointed at. Rotation keeps the bare refusal: a live
// factor means the password was never the required proof.
func firstTimeEnrollmentProofError(err error, rotation bool, credential *v1pb.CredentialProof) error {
	if rotation || connect.CodeOf(err) != connect.CodeUnauthenticated {
		return err
	}
	if _, ok := credential.GetProof().(*v1pb.CredentialProof_CurrentPassword); !ok {
		return err
	}
	return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("invalid password; if you sign in through SSO and never chose a password, ask a workspace admin to reset your password, then enroll with it"))
}

func (s *UserService) convertUserResponse(ctx context.Context, user *store.UserMessage) (*connect.Response[v1pb.User], error) {
	v1User, err := convertToUser(ctx, s.iamManager, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	return connect.NewResponse(v1User), nil
}

// convertToUser converts a stored user for a read. MFA enrollment state never
// appears here: the pending TOTP seed and recovery codes are returned exactly
// once, by StartMFAEnrollment, and a read that carried either would hand a
// reader the account's second factor.
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

// generateRandSecret generates a random TOTP secret for the given account
// name, returning the secret and its otpauth:// provisioning URI.
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
