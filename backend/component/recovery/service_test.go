package recovery_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/recovery"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestGetWorkspaceID(t *testing.T) {
	ctx := context.Background()
	stores := setupRecoveryStore(ctx, t)
	service := recovery.NewService(stores)

	workspaceID, err := service.GetWorkspaceID(ctx)
	require.ErrorContains(t, err, "recovery requires exactly one workspace")
	require.Empty(t, workspaceID)

	createRecoveryWorkspace(ctx, t, stores, "recovery-single-workspace", "admin-single@example.com", recoveryProfile())
	workspaceID, err = service.GetWorkspaceID(ctx)
	require.NoError(t, err)
	require.Equal(t, "recovery-single-workspace", workspaceID)

	require.NoError(t, stores.DeleteWorkspace(ctx, "recovery-single-workspace"))
	workspaceID, err = service.GetWorkspaceID(ctx)
	require.NoError(t, err)
	require.Empty(t, workspaceID)

	createRecoveryWorkspace(ctx, t, stores, "recovery-second-workspace", "admin-second@example.com", recoveryProfile())
	workspaceID, err = service.GetWorkspaceID(ctx)
	require.ErrorContains(t, err, "found 2 workspaces")
	require.Empty(t, workspaceID)
}

func TestEnablePasswordSignin(t *testing.T) {
	ctx := context.Background()
	stores := setupRecoveryStore(ctx, t)
	service := recovery.NewService(stores)

	t.Run("effective admin enables password signin and records audit", func(t *testing.T) {
		const (
			workspaceID = "recovery-enable-success"
			adminEmail  = "admin-enable@example.com"
		)
		createRecoveryWorkspace(ctx, t, stores, workspaceID, adminEmail, recoveryProfile())
		createRecoveryUser(ctx, t, stores, adminEmail, "KnownPassword1!", false, nil, nil)

		result, err := service.EnablePasswordSignin(ctx, workspaceID)
		require.NoError(t, err)
		require.Equal(t, &recovery.EnablePasswordSigninResult{
			WorkspaceID:      workspaceID,
			UsableAdminCount: 1,
		}, result)

		profile, err := stores.GetWorkspaceProfileSetting(ctx, workspaceID)
		require.NoError(t, err)
		require.False(t, profile.DisallowPasswordSignin)
		require.True(t, profile.DisallowSignup)
		require.Equal(t, "https://recovery.example.com", profile.ExternalUrl)
		require.EqualValues(t, 12, profile.PasswordRestriction.MinLength)

		result, err = service.EnablePasswordSignin(ctx, workspaceID)
		require.NoError(t, err)
		require.Equal(t, 1, result.UsableAdminCount)

		logs, err := stores.SearchAuditLogs(ctx, &store.AuditLogFind{Workspace: workspaceID})
		require.NoError(t, err)
		require.Len(t, logs, 2)
		log := logs[0].Payload
		require.Equal(t, "/bytebase.cli.Recovery/EnablePasswordSignin", log.Method)
		require.Equal(t, common.FormatWorkspace(workspaceID), log.Parent)
		require.Equal(t, common.FormatWorkspace(workspaceID), log.Resource)
		require.Empty(t, log.User)
		require.Equal(t, storepb.AuditLog_WARNING, log.Severity)
		require.Contains(t, log.Request, workspaceID)
		require.NotContains(t, strings.ToLower(log.Request), "password")
	})

	t.Run("group admin is effective", func(t *testing.T) {
		const (
			workspaceID = "recovery-enable-group"
			adminEmail  = "group-admin@example.com"
		)
		createRecoveryWorkspace(ctx, t, stores, workspaceID, "missing@example.com", recoveryProfile())
		createRecoveryUser(ctx, t, stores, adminEmail, "KnownPassword1!", false, nil, nil)
		group, err := stores.CreateGroup(ctx, &store.GroupMessage{
			Workspace: workspaceID,
			Email:     "recovery-admins@example.com",
			Title:     "Recovery admins",
			Payload: &storepb.GroupPayload{Members: []*storepb.GroupMember{{
				Member: common.FormatUserEmail(adminEmail),
				Role:   storepb.GroupMember_MEMBER,
			}}},
		})
		require.NoError(t, err)
		setWorkspaceIAMPolicy(ctx, t, stores, workspaceID, &storepb.IamPolicy{Bindings: []*storepb.Binding{{
			Role:    common.FormatRole(store.WorkspaceAdminRole),
			Members: []string{common.FormatGroupEmail(group.Email)},
		}}})

		result, err := service.EnablePasswordSignin(ctx, workspaceID)
		require.NoError(t, err)
		require.Equal(t, 1, result.UsableAdminCount)
	})

	for _, tc := range []struct {
		name          string
		workspaceID   string
		password      string
		deleted       bool
		wantErrorText string
	}{
		{name: "no effective admin", workspaceID: "recovery-enable-no-admin", wantErrorText: "no usable workspace administrator"},
		{name: "deleted admin", workspaceID: "recovery-enable-deleted", password: "KnownPassword1!", deleted: true, wantErrorText: "no usable workspace administrator"},
		{name: "admin without password", workspaceID: "recovery-enable-no-password", wantErrorText: "no usable workspace administrator"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adminEmail := tc.workspaceID + "@example.com"
			createRecoveryWorkspace(ctx, t, stores, tc.workspaceID, adminEmail, recoveryProfile())
			if tc.name != "no effective admin" {
				createRecoveryUser(ctx, t, stores, adminEmail, tc.password, tc.deleted, nil, nil)
			}

			result, err := service.EnablePasswordSignin(ctx, tc.workspaceID)
			require.ErrorContains(t, err, tc.wantErrorText)
			require.Nil(t, result)
			profile, getErr := stores.GetWorkspaceProfileSetting(ctx, tc.workspaceID)
			require.NoError(t, getErr)
			require.True(t, profile.DisallowPasswordSignin)
		})
	}
}

func TestResetUserPassword(t *testing.T) {
	ctx := context.Background()
	stores := setupRecoveryStore(ctx, t)
	service := recovery.NewService(stores)

	t.Run("returns the workspace password restriction", func(t *testing.T) {
		const workspaceID = "recovery-password-restriction"
		profile := recoveryProfile()
		createRecoveryWorkspace(ctx, t, stores, workspaceID, "admin@example.com", profile)

		restriction, err := service.GetPasswordRestriction(ctx, workspaceID)

		require.NoError(t, err)
		require.True(t, proto.Equal(profile.PasswordRestriction, restriction))
	})

	t.Run("resets an existing user password and only password metadata", func(t *testing.T) {
		const (
			workspaceID = "recovery-reset-success"
			email       = "existing-admin@example.com"
			oldPassword = "OriginalPassword1!"
			newPassword = "ReplacementPassword1!"
		)
		workspaceProfile := recoveryProfile()
		createRecoveryWorkspace(ctx, t, stores, workspaceID, email, workspaceProfile)
		userProfile := &storepb.UserProfile{Source: "entra-id", LastLoginWorkspace: "another-workspace"}
		mfa := &storepb.MFAConfig{OtpSecret: "otp-secret", RecoveryCodes: []string{"recovery-code"}}
		createRecoveryUser(ctx, t, stores, email, oldPassword, false, userProfile, mfa)
		require.NoError(t, stores.CreateWebRefreshToken(ctx, &store.WebRefreshTokenMessage{
			TokenHash: "existing-admin-token",
			UserEmail: email,
			ExpiresAt: time.Now().Add(time.Hour),
		}))
		policyBefore, err := stores.GetWorkspaceIamPolicy(ctx, workspaceID)
		require.NoError(t, err)
		policySnapshot := proto.CloneOf(policyBefore.Policy)
		settingSnapshot := proto.CloneOf(workspaceProfile)

		result, err := service.ResetUserPassword(ctx, recovery.ResetUserPasswordRequest{
			WorkspaceID: workspaceID,
			Email:       "EXISTING-ADMIN@EXAMPLE.COM",
			Password:    []byte(newPassword),
		})
		require.NoError(t, err)
		require.Equal(t, &recovery.ResetUserPasswordResult{WorkspaceID: workspaceID, Email: email, Changed: true}, result)

		user, err := stores.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		require.False(t, user.MemberDeleted)
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)))
		require.True(t, proto.Equal(mfa, user.MFAConfig))
		require.Equal(t, "entra-id", user.Profile.Source)
		require.Equal(t, "another-workspace", user.Profile.LastLoginWorkspace)
		require.NotNil(t, user.Profile.LastChangePasswordTime)

		policyAfter, err := stores.GetWorkspaceIamPolicy(ctx, workspaceID)
		require.NoError(t, err)
		require.True(t, proto.Equal(policySnapshot, policyAfter.Policy))
		profileAfter, err := stores.GetWorkspaceProfileSetting(ctx, workspaceID)
		require.NoError(t, err)
		require.True(t, proto.Equal(settingSnapshot, profileAfter))
		token, err := stores.GetWebRefreshToken(ctx, "existing-admin-token")
		require.NoError(t, err)
		require.NotNil(t, token)

		result, err = service.ResetUserPassword(ctx, recovery.ResetUserPasswordRequest{
			WorkspaceID: workspaceID,
			Email:       email,
			Password:    []byte(newPassword),
		})
		require.NoError(t, err)
		require.False(t, result.Changed)

		logs, err := stores.SearchAuditLogs(ctx, &store.AuditLogFind{Workspace: workspaceID})
		require.NoError(t, err)
		require.Len(t, logs, 2)
		for _, auditLog := range logs {
			payload := auditLog.Payload
			require.Equal(t, "/bytebase.cli.Recovery/ResetUserPassword", payload.Method)
			require.Equal(t, storepb.AuditLog_WARNING, payload.Severity)
			require.Empty(t, payload.User)
			require.Contains(t, payload.Request, email)
			require.NotContains(t, payload.Request, newPassword)
			require.NotContains(t, strings.ToLower(payload.Request), "password")
			require.NotContains(t, payload.Request, "$2")
		}
	})

	t.Run("accepts a user who belongs through a group", func(t *testing.T) {
		const (
			workspaceID = "recovery-reset-group"
			email       = "group-reset-admin@example.com"
		)
		createRecoveryWorkspace(ctx, t, stores, workspaceID, "missing@example.com", recoveryProfile())
		createRecoveryUser(ctx, t, stores, email, "OriginalPassword1!", false, nil, nil)
		group, err := stores.CreateGroup(ctx, &store.GroupMessage{
			Workspace: workspaceID,
			Email:     "reset-admins@example.com",
			Title:     "Reset admins",
			Payload: &storepb.GroupPayload{Members: []*storepb.GroupMember{{
				Member: common.FormatUserEmail(email),
				Role:   storepb.GroupMember_MEMBER,
			}}},
		})
		require.NoError(t, err)
		setWorkspaceIAMPolicy(ctx, t, stores, workspaceID, &storepb.IamPolicy{Bindings: []*storepb.Binding{{
			Role:    "roles/workspaceMember",
			Members: []string{common.FormatGroupEmail(group.Email)},
		}}})

		result, err := service.ResetUserPassword(ctx, recovery.ResetUserPasswordRequest{
			WorkspaceID: workspaceID,
			Email:       email,
			Password:    []byte("ReplacementPassword1!"),
		})
		require.NoError(t, err)
		require.True(t, result.Changed)
		member, err := service.IsUserInWorkspace(ctx, workspaceID, email)
		require.NoError(t, err)
		require.True(t, member)
	})

	t.Run("accepts a direct non-admin workspace member", func(t *testing.T) {
		const (
			workspaceID = "recovery-reset-member"
			email       = "workspace-member@example.com"
		)
		createRecoveryWorkspace(ctx, t, stores, workspaceID, "missing@example.com", recoveryProfile())
		createRecoveryUser(ctx, t, stores, email, "OriginalPassword1!", false, nil, nil)
		setWorkspaceIAMPolicy(ctx, t, stores, workspaceID, &storepb.IamPolicy{Bindings: []*storepb.Binding{{
			Role:    "roles/workspaceMember",
			Members: []string{common.FormatUserEmail(email)},
		}}})

		result, err := service.ResetUserPassword(ctx, recovery.ResetUserPasswordRequest{
			WorkspaceID: workspaceID,
			Email:       email,
			Password:    []byte("ReplacementPassword1!"),
		})
		require.NoError(t, err)
		require.True(t, result.Changed)
		member, err := service.IsUserInWorkspace(ctx, workspaceID, email)
		require.NoError(t, err)
		require.True(t, member)
	})

	t.Run("reports a user covered by allUsers as a workspace member", func(t *testing.T) {
		const (
			workspaceID = "recovery-reset-all-users"
			email       = "all-users-member@example.com"
		)
		createRecoveryWorkspace(ctx, t, stores, workspaceID, "missing@example.com", recoveryProfile())
		createRecoveryUser(ctx, t, stores, email, "OriginalPassword1!", false, nil, nil)
		setWorkspaceIAMPolicy(ctx, t, stores, workspaceID, &storepb.IamPolicy{Bindings: []*storepb.Binding{{
			Role:    "roles/workspaceMember",
			Members: []string{common.AllUsers},
		}}})

		member, err := service.IsUserInWorkspace(ctx, workspaceID, email)
		require.NoError(t, err)
		require.True(t, member)
	})

	t.Run("resets a user outside the workspace before reporting membership", func(t *testing.T) {
		const (
			workspaceID = "recovery-reset-unrelated"
			email       = "unrelated-user@example.com"
			newPassword = "ReplacementPassword1!"
		)
		createRecoveryWorkspace(ctx, t, stores, workspaceID, "missing@example.com", recoveryProfile())
		createRecoveryUser(ctx, t, stores, email, "OriginalPassword1!", false, nil, nil)

		result, err := service.ResetUserPassword(ctx, recovery.ResetUserPasswordRequest{
			WorkspaceID: workspaceID,
			Email:       email,
			Password:    []byte(newPassword),
		})
		require.NoError(t, err)
		require.True(t, result.Changed)
		member, err := service.IsUserInWorkspace(ctx, workspaceID, email)
		require.NoError(t, err)
		require.False(t, member)
		user, err := stores.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)))
	})

	for _, tc := range []struct {
		name        string
		workspaceID string
		email       string
		create      func(context.Context, *testing.T, *store.Store, string, string)
	}{
		{
			name:        "service account",
			workspaceID: "recovery-reset-service-account",
			email:       "automation@service.bytebase.com",
			create: func(ctx context.Context, t *testing.T, stores *store.Store, workspaceID, email string) {
				_, err := stores.CreateServiceAccount(ctx, &store.CreateServiceAccountMessage{
					Email: email, Name: "Automation", Workspace: workspaceID, ServiceKeyHash: "unused",
				})
				require.NoError(t, err)
			},
		},
		{
			name:        "workload identity",
			workspaceID: "recovery-reset-workload-identity",
			email:       "automation@workload.bytebase.com",
			create: func(ctx context.Context, t *testing.T, stores *store.Store, workspaceID, email string) {
				_, err := stores.CreateWorkloadIdentity(ctx, &store.CreateWorkloadIdentityMessage{
					Email: email, Name: "Automation", Workspace: workspaceID, Config: &storepb.WorkloadIdentityConfig{},
				})
				require.NoError(t, err)
			},
		},
	} {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			createRecoveryWorkspace(ctx, t, stores, tc.workspaceID, "missing@example.com", recoveryProfile())
			tc.create(ctx, t, stores, tc.workspaceID, tc.email)

			result, err := service.ResetUserPassword(ctx, recovery.ResetUserPasswordRequest{
				WorkspaceID: tc.workspaceID,
				Email:       tc.email,
				Password:    []byte("ReplacementPassword1!"),
			})
			require.ErrorContains(t, err, "not an active end user")
			require.Nil(t, result)
		})
	}

	for _, tc := range []struct {
		name        string
		workspaceID string
		email       string
		profile     *storepb.WorkspaceProfileSetting
		password    string
		createUser  bool
		deleted     bool
		wantError   string
	}{
		{
			name:        "missing user",
			workspaceID: "recovery-reset-missing",
			email:       "missing-admin@example.com",
			profile:     recoveryProfile(),
			password:    "ReplacementPassword1!",
			wantError:   "does not exist",
		},
		{
			name:        "deleted user",
			workspaceID: "recovery-reset-deleted",
			email:       "deleted-admin@example.com",
			profile:     recoveryProfile(),
			password:    "ReplacementPassword1!",
			createUser:  true,
			deleted:     true,
			wantError:   "not an active end user",
		},
		{
			name:        "outside enforced domain",
			workspaceID: "recovery-reset-domain",
			email:       "admin@blocked.example",
			profile: func() *storepb.WorkspaceProfileSetting {
				profile := recoveryProfile()
				profile.EnforceIdentityDomain = true
				profile.Domains = []string{"allowed.example"}
				return profile
			}(),
			password:   "ReplacementPassword1!",
			createUser: true,
			wantError:  "allowed domains",
		},
		{
			name:        "password restriction",
			workspaceID: "recovery-reset-password",
			email:       "admin-password@example.com",
			profile:     recoveryProfile(),
			password:    "too-short",
			createUser:  true,
			wantError:   "password length",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			createRecoveryWorkspace(ctx, t, stores, tc.workspaceID, tc.email, tc.profile)
			if tc.createUser {
				createRecoveryUser(ctx, t, stores, tc.email, "OriginalPassword1!", tc.deleted, nil, nil)
			}
			result, err := service.ResetUserPassword(ctx, recovery.ResetUserPasswordRequest{
				WorkspaceID: tc.workspaceID,
				Email:       tc.email,
				Password:    []byte(tc.password),
			})
			require.ErrorContains(t, err, tc.wantError)
			require.Nil(t, result)
			user, getErr := stores.GetUserByEmail(ctx, tc.email)
			require.NoError(t, getErr)
			if tc.createUser {
				require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("OriginalPassword1!")))
			} else {
				require.Nil(t, user)
			}
			profile, getErr := stores.GetWorkspaceProfileSetting(ctx, tc.workspaceID)
			require.NoError(t, getErr)
			require.True(t, profile.DisallowPasswordSignin)
		})
	}
}

func TestListRoles(t *testing.T) {
	ctx := context.Background()
	stores := setupRecoveryStore(ctx, t)
	service := recovery.NewService(stores)
	const workspaceID = "recovery-list-roles"
	createRecoveryWorkspace(ctx, t, stores, workspaceID, "admin-list-roles@example.com", recoveryProfile())
	_, err := stores.CreateRole(ctx, &store.RoleMessage{
		ResourceID: "recoveryOperator",
		Workspace:  workspaceID,
		Name:       "Recovery operator",
	})
	require.NoError(t, err)

	roles, err := service.ListRoles(ctx, workspaceID)
	require.NoError(t, err)
	require.Greater(t, len(roles), 3)
	require.Equal(t, []*recovery.Role{
		{Name: "roles/workspaceAdmin", Title: "Workspace admin"},
		{Name: "roles/workspaceDBA", Title: "Workspace DBA"},
		{Name: "roles/workspaceMember", Title: "Workspace member"},
	}, roles[:3])
	require.Contains(t, roles, &recovery.Role{Name: "roles/recoveryOperator", Title: "Recovery operator"})

	var remaining []string
	for _, role := range roles[3:] {
		remaining = append(remaining, strings.ToLower(role.Title)+"\x00"+role.Name)
	}
	sorted := append([]string(nil), remaining...)
	slices.Sort(sorted)
	require.Equal(t, sorted, remaining)
}

func TestAddUserToWorkspace(t *testing.T) {
	ctx := context.Background()
	stores := setupRecoveryStore(ctx, t)
	service := recovery.NewService(stores)

	t.Run("adds one direct role and records an audit", func(t *testing.T) {
		const (
			workspaceID = "recovery-add-user"
			adminEmail  = "existing-recovery-admin@example.com"
			email       = "restored-user@example.com"
			role        = "roles/workspaceDBA"
		)
		createRecoveryWorkspace(ctx, t, stores, workspaceID, adminEmail, recoveryProfile())
		createRecoveryUser(ctx, t, stores, adminEmail, "OriginalPassword1!", false, nil, nil)
		createRecoveryUser(ctx, t, stores, email, "OriginalPassword1!", false, nil, nil)

		result, err := service.AddUserToWorkspace(ctx, recovery.AddUserToWorkspaceRequest{
			WorkspaceID: workspaceID,
			Email:       strings.ToUpper(email),
			Role:        role,
		})
		require.NoError(t, err)
		require.Equal(t, &recovery.AddUserToWorkspaceResult{
			WorkspaceID: workspaceID,
			Email:       email,
			Role:        role,
			Changed:     true,
		}, result)

		policy, err := stores.GetWorkspaceIamPolicy(ctx, workspaceID)
		require.NoError(t, err)
		require.True(t, bindingContains(policy.Policy, "roles/workspaceAdmin", common.FormatUserEmail(adminEmail)))
		require.True(t, bindingContains(policy.Policy, role, common.FormatUserEmail(email)))

		result, err = service.AddUserToWorkspace(ctx, recovery.AddUserToWorkspaceRequest{
			WorkspaceID: workspaceID,
			Email:       email,
			Role:        role,
		})
		require.NoError(t, err)
		require.False(t, result.Changed)

		logs, err := stores.SearchAuditLogs(ctx, &store.AuditLogFind{Workspace: workspaceID})
		require.NoError(t, err)
		require.Len(t, logs, 2)
		for _, auditLog := range logs {
			payload := auditLog.Payload
			require.Equal(t, "/bytebase.cli.Recovery/AddUserToWorkspace", payload.Method)
			require.Contains(t, payload.Request, email)
			require.Contains(t, payload.Request, role)
			require.NotContains(t, strings.ToLower(payload.Request), "password")
			require.NotContains(t, payload.Request, "$2")
		}
	})

	t.Run("rejects an unknown role", func(t *testing.T) {
		const (
			workspaceID = "recovery-add-unknown-role"
			email       = "unknown-role-user@example.com"
		)
		createRecoveryWorkspace(ctx, t, stores, workspaceID, "admin-unknown-role@example.com", recoveryProfile())
		createRecoveryUser(ctx, t, stores, email, "OriginalPassword1!", false, nil, nil)

		result, err := service.AddUserToWorkspace(ctx, recovery.AddUserToWorkspaceRequest{
			WorkspaceID: workspaceID,
			Email:       email,
			Role:        "roles/doesNotExist",
		})
		require.ErrorContains(t, err, "role")
		require.Nil(t, result)
	})

	t.Run("rejects an email outside the enforced domain", func(t *testing.T) {
		const (
			workspaceID = "recovery-add-domain"
			email       = "user@blocked.example"
		)
		profile := recoveryProfile()
		profile.EnforceIdentityDomain = true
		profile.Domains = []string{"allowed.example"}
		createRecoveryWorkspace(ctx, t, stores, workspaceID, "admin@allowed.example", profile)
		createRecoveryUser(ctx, t, stores, email, "OriginalPassword1!", false, nil, nil)

		result, err := service.AddUserToWorkspace(ctx, recovery.AddUserToWorkspaceRequest{
			WorkspaceID: workspaceID,
			Email:       email,
			Role:        "roles/workspaceMember",
		})
		require.ErrorContains(t, err, "allowed domains")
		require.Nil(t, result)
	})

	t.Run("rejects a non-end-user identity", func(t *testing.T) {
		const (
			workspaceID = "recovery-add-service-account"
			email       = "recovery-add@service.bytebase.com"
		)
		createRecoveryWorkspace(ctx, t, stores, workspaceID, "admin-add-service-account@example.com", recoveryProfile())
		_, err := stores.CreateServiceAccount(ctx, &store.CreateServiceAccountMessage{
			Email: email, Name: "Recovery automation", Workspace: workspaceID, ServiceKeyHash: "unused",
		})
		require.NoError(t, err)

		result, err := service.AddUserToWorkspace(ctx, recovery.AddUserToWorkspaceRequest{
			WorkspaceID: workspaceID,
			Email:       email,
			Role:        "roles/workspaceMember",
		})
		require.ErrorContains(t, err, "not an active end user")
		require.Nil(t, result)
	})

	for _, tc := range []struct {
		name       string
		email      string
		createUser bool
		deleted    bool
	}{
		{name: "missing user", email: "missing-add-user@example.com"},
		{name: "deleted user", email: "deleted-add-user@example.com", createUser: true, deleted: true},
	} {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			workspaceID := "recovery-add-" + strings.ReplaceAll(tc.name, " ", "-")
			createRecoveryWorkspace(ctx, t, stores, workspaceID, "admin-"+tc.email, recoveryProfile())
			if tc.createUser {
				createRecoveryUser(ctx, t, stores, tc.email, "OriginalPassword1!", tc.deleted, nil, nil)
			}

			result, err := service.AddUserToWorkspace(ctx, recovery.AddUserToWorkspaceRequest{
				WorkspaceID: workspaceID,
				Email:       tc.email,
				Role:        "roles/workspaceMember",
			})
			require.Error(t, err)
			require.Nil(t, result)
		})
	}
}

func setupRecoveryStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	require.NoError(t, migrator.MigrateSchema(ctx, container.GetDB()))

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })
	return stores
}

func createRecoveryWorkspace(ctx context.Context, t *testing.T, stores *store.Store, workspaceID, adminEmail string, profile *storepb.WorkspaceProfileSetting) {
	t.Helper()
	_, err := stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: workspaceID,
		Payload:    &storepb.WorkspacePayload{Title: workspaceID},
	}, adminEmail)
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_WORKSPACE_PROFILE,
		Workspace: workspaceID,
		Value:     profile,
	})
	require.NoError(t, err)
}

func recoveryProfile() *storepb.WorkspaceProfileSetting {
	return &storepb.WorkspaceProfileSetting{
		DisallowPasswordSignin: true,
		DisallowSignup:         true,
		ExternalUrl:            "https://recovery.example.com",
		PasswordRestriction: &storepb.WorkspaceProfileSetting_PasswordRestriction{
			MinLength:               12,
			RequireNumber:           true,
			RequireLetter:           true,
			RequireUppercaseLetter:  true,
			RequireSpecialCharacter: true,
		},
	}
}

func createRecoveryUser(ctx context.Context, t *testing.T, stores *store.Store, email, password string, deleted bool, profile *storepb.UserProfile, mfa *storepb.MFAConfig) {
	t.Helper()
	var passwordHash string
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)
		passwordHash = string(hash)
	}
	user, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        email,
		Name:         email,
		PasswordHash: passwordHash,
		Profile:      profile,
	})
	require.NoError(t, err)
	if deleted || mfa != nil {
		_, err = stores.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &deleted, MFAConfig: mfa})
		require.NoError(t, err)
	}
}

func setWorkspaceIAMPolicy(ctx context.Context, t *testing.T, stores *store.Store, workspaceID string, policy *storepb.IamPolicy) {
	t.Helper()
	payload, err := protojson.Marshal(policy)
	require.NoError(t, err)
	_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace:         workspaceID,
		Resource:          common.FormatWorkspace(workspaceID),
		ResourceType:      storepb.Policy_WORKSPACE,
		Payload:           string(payload),
		Type:              storepb.Policy_IAM,
		InheritFromParent: false,
		Enforce:           true,
	})
	require.NoError(t, err)
}

func bindingContains(policy *storepb.IamPolicy, role, member string) bool {
	for _, binding := range policy.Bindings {
		if binding.Role == role && slices.Contains(binding.Members, member) {
			return true
		}
	}
	return false
}
