// Package recovery implements self-hosted workspace recovery operations.
package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

const (
	enablePasswordSigninAuditMethod = "/bytebase.cli.Recovery/EnablePasswordSignin"
	resetUserPasswordAuditMethod    = "/bytebase.cli.Recovery/ResetUserPassword"
	addUserToWorkspaceAuditMethod   = "/bytebase.cli.Recovery/AddUserToWorkspace"
)

// Service performs self-hosted recovery operations against the metadata store.
type Service struct {
	store *store.Store
}

// EnablePasswordSigninResult describes an existing-administrator recovery.
type EnablePasswordSigninResult struct {
	WorkspaceID      string
	UsableAdminCount int
}

// ResetUserPasswordRequest describes a user password reset.
type ResetUserPasswordRequest struct {
	WorkspaceID string
	Email       string
	Password    []byte
}

// ResetUserPasswordResult describes a user password reset.
type ResetUserPasswordResult struct {
	WorkspaceID string
	Email       string
	Changed     bool
}

// UserNotInWorkspaceError indicates that an active end user has no effective
// membership in the selected workspace.
type UserNotInWorkspaceError struct {
	WorkspaceID string
	Email       string
}

func (e *UserNotInWorkspaceError) Error() string {
	return fmt.Sprintf("user %q does not belong to workspace %q", e.Email, e.WorkspaceID)
}

// Role describes a role available for workspace recovery.
type Role struct {
	Name  string
	Title string
}

// AddUserToWorkspaceRequest describes an IAM membership recovery.
type AddUserToWorkspaceRequest struct {
	WorkspaceID string
	Email       string
	Role        string
}

// AddUserToWorkspaceResult describes an IAM membership recovery.
type AddUserToWorkspaceResult struct {
	WorkspaceID string
	Email       string
	Role        string
	Changed     bool
}

// NewService creates a recovery service.
func NewService(stores *store.Store) *Service {
	return &Service{store: stores}
}

// GetWorkspaceID returns the only active workspace resource ID.
func (s *Service) GetWorkspaceID(ctx context.Context) (string, error) {
	count, err := s.store.CountWorkspaces(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to count workspaces")
	}
	if count != 1 {
		return "", errors.Errorf("found %d workspaces; recovery requires exactly one workspace", count)
	}
	return s.store.GetWorkspaceID(ctx)
}

// EnablePasswordSignin enables password sign-in when the workspace already has
// an effective, active end-user administrator with a password credential.
func (s *Service) EnablePasswordSignin(ctx context.Context, workspaceID string) (*EnablePasswordSigninResult, error) {
	if err := s.requireWorkspace(ctx, workspaceID); err != nil {
		return nil, err
	}

	policy, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace IAM policy")
	}
	effectiveAdmins := utils.GetUsersByRoleInIAMPolicy(
		ctx,
		s.store,
		workspaceID,
		store.WorkspaceAdminRole,
		true,
		policy.Policy,
	)
	usableAdminCount := 0
	for _, effectiveAdmin := range effectiveAdmins {
		if effectiveAdmin.Type != storepb.PrincipalType_END_USER {
			continue
		}
		admin, err := s.store.GetUserByEmail(ctx, effectiveAdmin.Email)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load effective administrator %q", effectiveAdmin.Email)
		}
		if admin != nil && !admin.MemberDeleted && admin.PasswordHash != "" {
			usableAdminCount++
		}
	}
	if usableAdminCount == 0 {
		return nil, errors.New("no usable workspace administrator has an active end-user identity with a password credential")
	}
	auditRequest, err := json.Marshal(struct {
		Workspace string `json:"workspace"`
	}{
		Workspace: common.FormatWorkspace(workspaceID),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal recovery audit request")
	}

	if _, err := s.store.UpdateSettingAtomic(
		ctx,
		workspaceID,
		storepb.SettingName_WORKSPACE_PROFILE,
		func(current proto.Message) (proto.Message, error) {
			profile, ok := current.(*storepb.WorkspaceProfileSetting)
			if !ok {
				return nil, errors.New("workspace profile setting has an unexpected type")
			}
			profile.DisallowPasswordSignin = false
			return profile, nil
		},
		nil,
	); err != nil {
		return nil, errors.Wrap(err, "failed to enable password sign-in")
	}

	result := &EnablePasswordSigninResult{
		WorkspaceID:      workspaceID,
		UsableAdminCount: usableAdminCount,
	}
	if err := s.createAuditLog(ctx, workspaceID, enablePasswordSigninAuditMethod, string(auditRequest)); err != nil {
		return result, errors.Wrap(err, "password sign-in was enabled, but failed to create the recovery audit log")
	}
	return result, nil
}

// ListRoles lists roles available for workspace membership recovery.
func (s *Service) ListRoles(ctx context.Context, workspaceID string) ([]*Role, error) {
	if err := s.requireWorkspace(ctx, workspaceID); err != nil {
		return nil, err
	}

	roleMessages, err := s.store.ListRoles(ctx, &store.FindRoleMessage{Workspace: workspaceID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list workspace roles")
	}
	roles := make([]*Role, 0, len(roleMessages))
	for _, role := range roleMessages {
		roles = append(roles, &Role{Name: common.FormatRole(role.ResourceID), Title: role.Name})
	}
	priority := map[string]int{
		common.FormatRole(store.WorkspaceAdminRole):  0,
		common.FormatRole(store.WorkspaceDBARole):    1,
		common.FormatRole(store.WorkspaceMemberRole): 2,
	}
	slices.SortFunc(roles, func(left, right *Role) int {
		leftPriority, leftPreferred := priority[left.Name]
		rightPriority, rightPreferred := priority[right.Name]
		if leftPreferred != rightPreferred {
			if leftPreferred {
				return -1
			}
			return 1
		}
		if leftPreferred && leftPriority != rightPriority {
			return leftPriority - rightPriority
		}
		leftTitle := strings.ToLower(left.Title)
		rightTitle := strings.ToLower(right.Title)
		if leftTitle != rightTitle {
			return strings.Compare(leftTitle, rightTitle)
		}
		return strings.Compare(left.Name, right.Name)
	})
	return roles, nil
}

// AddUserToWorkspace assigns one direct role to an existing, active end user.
func (s *Service) AddUserToWorkspace(ctx context.Context, request AddUserToWorkspaceRequest) (*AddUserToWorkspaceResult, error) {
	if err := s.requireWorkspace(ctx, request.WorkspaceID); err != nil {
		return nil, err
	}

	email := strings.ToLower(request.Email)
	if err := common.ValidateEmail(email); err != nil {
		return nil, errors.Wrap(err, "invalid user email")
	}
	profile, err := s.store.GetWorkspaceProfileSetting(ctx, request.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace profile setting")
	}
	if profile.EnforceIdentityDomain && !emailBelongsToDomains(email, profile.Domains) {
		return nil, errors.Errorf("email %q does not belong to the workspace's allowed domains", email)
	}
	if _, err := s.getActiveEndUser(ctx, email); err != nil {
		return nil, err
	}

	roleID, err := common.GetRoleID(request.Role)
	if err != nil {
		return nil, errors.Wrap(err, "invalid role")
	}
	role, err := s.store.GetRole(ctx, &store.FindRoleMessage{Workspace: request.WorkspaceID, ResourceID: &roleID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to find role")
	}
	if role == nil {
		return nil, errors.Errorf("role %q does not exist", request.Role)
	}

	policy, err := s.store.GetWorkspaceIamPolicy(ctx, request.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace IAM policy")
	}
	member := common.FormatUserEmail(email)
	directRoles := make([]string, 0)
	selected := false
	for _, binding := range policy.Policy.Bindings {
		if slices.Contains(binding.Members, member) {
			directRoles = append(directRoles, binding.Role)
			selected = selected || binding.Role == request.Role
		}
	}

	result := &AddUserToWorkspaceResult{
		WorkspaceID: request.WorkspaceID,
		Email:       email,
		Role:        request.Role,
	}
	if !selected {
		directRoles = append(directRoles, request.Role)
		if _, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
			Workspace: request.WorkspaceID,
			Member:    member,
			Roles:     directRoles,
		}); err != nil {
			return nil, errors.Wrap(err, "failed to add user to workspace IAM policy")
		}
		result.Changed = true
	}

	auditRequest, err := json.Marshal(struct {
		Workspace string `json:"workspace"`
		Email     string `json:"email"`
		Role      string `json:"role"`
	}{
		Workspace: common.FormatWorkspace(request.WorkspaceID),
		Email:     email,
		Role:      request.Role,
	})
	if err != nil {
		return result, errors.Wrap(err, "failed to marshal recovery audit request")
	}
	if err := s.createAuditLog(ctx, request.WorkspaceID, addUserToWorkspaceAuditMethod, string(auditRequest)); err != nil {
		if result.Changed {
			return result, errors.Wrap(err, "user was added to the workspace, but failed to create the recovery audit log")
		}
		return result, errors.Wrap(err, "failed to create the recovery audit log")
	}
	return result, nil
}

// ResetUserPassword replaces the password of an existing, active end user in
// the workspace without changing any other access state.
func (s *Service) ResetUserPassword(ctx context.Context, request ResetUserPasswordRequest) (*ResetUserPasswordResult, error) {
	if err := s.requireWorkspace(ctx, request.WorkspaceID); err != nil {
		return nil, err
	}

	email := strings.ToLower(request.Email)
	if err := common.ValidateEmail(email); err != nil {
		return nil, errors.Wrap(err, "invalid user email")
	}

	profile, err := s.store.GetWorkspaceProfileSetting(ctx, request.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace profile setting")
	}
	if profile.EnforceIdentityDomain && !emailBelongsToDomains(email, profile.Domains) {
		return nil, errors.Errorf("email %q does not belong to the workspace's allowed domains", email)
	}
	if err := common.ValidatePassword(string(request.Password), profile.PasswordRestriction); err != nil {
		return nil, errors.Wrap(err, "password does not satisfy the workspace restriction")
	}

	user, err := s.getActiveEndUser(ctx, email)
	if err != nil {
		return nil, err
	}

	workspace, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		WorkspaceID:    &request.WorkspaceID,
		Email:          email,
		IncludeAllUser: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify workspace membership")
	}
	if workspace == nil {
		return nil, &UserNotInWorkspaceError{WorkspaceID: request.WorkspaceID, Email: email}
	}
	auditRequest, err := json.Marshal(struct {
		Workspace string `json:"workspace"`
		Email     string `json:"email"`
	}{
		Workspace: common.FormatWorkspace(request.WorkspaceID),
		Email:     email,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal recovery audit request")
	}

	result := &ResetUserPasswordResult{WorkspaceID: request.WorkspaceID, Email: email}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), request.Password) != nil {
		passwordHash, err := bcrypt.GenerateFromPassword(request.Password, bcrypt.DefaultCost)
		if err != nil {
			return nil, errors.Wrap(err, "failed to hash user password")
		}
		if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{PasswordHash: new(string(passwordHash))}); err != nil {
			return nil, errors.Wrap(err, "failed to reset user password")
		}
		result.Changed = true
	}

	if err := s.createAuditLog(ctx, request.WorkspaceID, resetUserPasswordAuditMethod, string(auditRequest)); err != nil {
		return result, errors.Wrap(err, "user password reset completed, but failed to create the recovery audit log")
	}
	return result, nil
}

func (s *Service) getActiveEndUser(ctx context.Context, email string) (*store.UserMessage, error) {
	account, err := s.store.GetAccountByEmail(ctx, email)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find user identity")
	}
	if account == nil {
		return nil, errors.Errorf("user %q does not exist", email)
	}
	if account.Type != storepb.PrincipalType_END_USER || account.MemberDeleted {
		return nil, errors.Errorf("user %q is not an active end user", email)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load user identity")
	}
	if user == nil || user.MemberDeleted {
		return nil, errors.Errorf("user %q is not an active end user", email)
	}
	return user, nil
}

func (s *Service) requireWorkspace(ctx context.Context, workspaceID string) error {
	workspace, err := s.store.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return errors.Wrap(err, "failed to find workspace")
	}
	if workspace == nil {
		return errors.Errorf("workspace %q was not found", workspaceID)
	}
	return nil
}

func emailBelongsToDomains(email string, domains []string) bool {
	if len(domains) == 0 {
		return true
	}
	for _, domain := range domains {
		if strings.HasSuffix(email, "@"+strings.ToLower(domain)) {
			return true
		}
	}
	return false
}

func (s *Service) createAuditLog(ctx context.Context, workspaceID, method, request string) error {
	workspace := common.FormatWorkspace(workspaceID)
	return s.store.CreateAuditLog(context.WithoutCancel(ctx), workspaceID, &storepb.AuditLog{
		Parent:   workspace,
		Method:   method,
		Resource: workspace,
		Severity: storepb.AuditLog_WARNING,
		Request:  request,
		Status:   &statuspb.Status{},
	})
}
