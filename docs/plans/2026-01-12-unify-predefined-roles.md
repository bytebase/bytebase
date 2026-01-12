# Unify Predefined Roles Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move predefined role definitions from IAM manager to store package, with permissions in a shared package.

**Architecture:** Create `backend/common/permission/` for shared permission types. Define predefined roles as Go code in `backend/store/predefined_roles.go`. Store's `ListRoles()` returns both custom and predefined roles.

**Tech Stack:** Go, no external dependencies

---

## Task 1: Create shared permission package

**Files:**
- Create: `backend/common/permission/permission.go`
- Create: `backend/common/permission/permission.yaml` (move from iam)

**Step 1: Create permission package**

```go
// backend/common/permission/permission.go
package permission

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed permission.yaml
var permissionYAML []byte

type Permission = string

const (
	PermissionAuditLogsExport             Permission = "bb.auditLogs.export"
	PermissionAuditLogsSearch             Permission = "bb.auditLogs.search"
	PermissionChangelogsGet               Permission = "bb.changelogs.get"
	PermissionChangelogsList              Permission = "bb.changelogs.list"
	PermissionDatabaseCatalogsGet         Permission = "bb.databaseCatalogs.get"
	PermissionDatabaseCatalogsUpdate      Permission = "bb.databaseCatalogs.update"
	PermissionDatabaseGroupsCreate        Permission = "bb.databaseGroups.create"
	PermissionDatabaseGroupsDelete        Permission = "bb.databaseGroups.delete"
	PermissionDatabaseGroupsGet           Permission = "bb.databaseGroups.get"
	PermissionDatabaseGroupsList          Permission = "bb.databaseGroups.list"
	PermissionDatabaseGroupsUpdate        Permission = "bb.databaseGroups.update"
	PermissionDatabasesCheck              Permission = "bb.databases.check"
	PermissionDatabasesGet                Permission = "bb.databases.get"
	PermissionDatabasesGetSchema          Permission = "bb.databases.getSchema"
	PermissionDatabasesList               Permission = "bb.databases.list"
	PermissionDatabasesSync               Permission = "bb.databases.sync"
	PermissionDatabasesUpdate             Permission = "bb.databases.update"
	PermissionIdentityProvidersCreate     Permission = "bb.identityProviders.create"
	PermissionIdentityProvidersDelete     Permission = "bb.identityProviders.delete"
	PermissionIdentityProvidersGet        Permission = "bb.identityProviders.get"
	PermissionIdentityProvidersUpdate     Permission = "bb.identityProviders.update"
	PermissionInstancesCreate             Permission = "bb.instances.create"
	PermissionInstancesDelete             Permission = "bb.instances.delete"
	PermissionInstancesGet                Permission = "bb.instances.get"
	PermissionInstancesList               Permission = "bb.instances.list"
	PermissionInstancesSync               Permission = "bb.instances.sync"
	PermissionInstancesUndelete           Permission = "bb.instances.undelete"
	PermissionInstancesUpdate             Permission = "bb.instances.update"
	PermissionInstanceRolesGet            Permission = "bb.instanceRoles.get"
	PermissionInstanceRolesList           Permission = "bb.instanceRoles.list"
	PermissionIssueCommentsCreate         Permission = "bb.issueComments.create"
	PermissionIssueCommentsList           Permission = "bb.issueComments.list"
	PermissionIssueCommentsUpdate         Permission = "bb.issueComments.update"
	PermissionIssuesCreate                Permission = "bb.issues.create"
	PermissionIssuesGet                   Permission = "bb.issues.get"
	PermissionIssuesList                  Permission = "bb.issues.list"
	PermissionIssuesUpdate                Permission = "bb.issues.update"
	PermissionPlanCheckRunsGet            Permission = "bb.planCheckRuns.get"
	PermissionPlanCheckRunsRun            Permission = "bb.planCheckRuns.run"
	PermissionPlansCreate                 Permission = "bb.plans.create"
	PermissionPlansGet                    Permission = "bb.plans.get"
	PermissionPlansList                   Permission = "bb.plans.list"
	PermissionPlansUpdate                 Permission = "bb.plans.update"
	PermissionPoliciesCreate              Permission = "bb.policies.create"
	PermissionPoliciesDelete              Permission = "bb.policies.delete"
	PermissionPoliciesGet                 Permission = "bb.policies.get"
	PermissionPoliciesList                Permission = "bb.policies.list"
	PermissionPoliciesUpdate              Permission = "bb.policies.update"
	PermissionProjectsCreate              Permission = "bb.projects.create"
	PermissionProjectsDelete              Permission = "bb.projects.delete"
	PermissionProjectsGet                 Permission = "bb.projects.get"
	PermissionProjectsGetIAMPolicy        Permission = "bb.projects.getIamPolicy"
	PermissionProjectsList                Permission = "bb.projects.list"
	PermissionProjectsSetIAMPolicy        Permission = "bb.projects.setIamPolicy"
	PermissionProjectsUndelete            Permission = "bb.projects.undelete"
	PermissionProjectsUpdate              Permission = "bb.projects.update"
	PermissionReleasesCheck               Permission = "bb.releases.check"
	PermissionReleasesCreate              Permission = "bb.releases.create"
	PermissionReleasesDelete              Permission = "bb.releases.delete"
	PermissionReleasesGet                 Permission = "bb.releases.get"
	PermissionReleasesList                Permission = "bb.releases.list"
	PermissionReleasesUndelete            Permission = "bb.releases.undelete"
	PermissionReleasesUpdate              Permission = "bb.releases.update"
	PermissionReviewConfigsCreate         Permission = "bb.reviewConfigs.create"
	PermissionReviewConfigsDelete         Permission = "bb.reviewConfigs.delete"
	PermissionReviewConfigsGet            Permission = "bb.reviewConfigs.get"
	PermissionReviewConfigsList           Permission = "bb.reviewConfigs.list"
	PermissionReviewConfigsUpdate         Permission = "bb.reviewConfigs.update"
	PermissionRevisionsCreate             Permission = "bb.revisions.create"
	PermissionRevisionsDelete             Permission = "bb.revisions.delete"
	PermissionRevisionsGet                Permission = "bb.revisions.get"
	PermissionRevisionsList               Permission = "bb.revisions.list"
	PermissionRolesCreate                 Permission = "bb.roles.create"
	PermissionRolesDelete                 Permission = "bb.roles.delete"
	PermissionRolesList                   Permission = "bb.roles.list"
	PermissionRolesGet                    Permission = "bb.roles.get"
	PermissionRolesUpdate                 Permission = "bb.roles.update"
	PermissionRolloutsCreate              Permission = "bb.rollouts.create"
	PermissionRolloutsGet                 Permission = "bb.rollouts.get"
	PermissionRolloutsList                Permission = "bb.rollouts.list"
	PermissionSettingsGet                 Permission = "bb.settings.get"
	PermissionSettingsList                Permission = "bb.settings.list"
	PermissionSettingsSet                 Permission = "bb.settings.set"
	PermissionEnvironmentSettingsGet      Permission = "bb.settings.getEnvironment"
	PermissionEnvironmentSettingsSet      Permission = "bb.settings.setEnvironment"
	PermissionWorkspaceProfileSettingsGet Permission = "bb.settings.getWorkspaceProfile"
	PermissionWorkspaceProfileSettingsSet Permission = "bb.settings.setWorkspaceProfile"
	PermissionSheetsCreate                Permission = "bb.sheets.create"
	PermissionSheetsGet                   Permission = "bb.sheets.get"
	PermissionSheetsUpdate                Permission = "bb.sheets.update"
	PermissionSQLSelect                   Permission = "bb.sql.select"
	PermissionSQLDdl                      Permission = "bb.sql.ddl"
	PermissionSQLDml                      Permission = "bb.sql.dml"
	PermissionSQLExplain                  Permission = "bb.sql.explain"
	PermissionSQLInfo                     Permission = "bb.sql.info"
	PermissionSQLAdmin                    Permission = "bb.sql.admin"
	PermissionTaskRunsCreate              Permission = "bb.taskRuns.create"
	PermissionTaskRunsList                Permission = "bb.taskRuns.list"
	PermissionGroupsCreate                Permission = "bb.groups.create"
	PermissionGroupsDelete                Permission = "bb.groups.delete"
	PermissionGroupsGet                   Permission = "bb.groups.get"
	PermissionGroupsList                  Permission = "bb.groups.list"
	PermissionGroupsUpdate                Permission = "bb.groups.update"
	PermissionUsersCreate                 Permission = "bb.users.create"
	PermissionUsersDelete                 Permission = "bb.users.delete"
	PermissionUsersGet                    Permission = "bb.users.get"
	PermissionUsersList                   Permission = "bb.users.list"
	PermissionUsersUndelete               Permission = "bb.users.undelete"
	PermissionUsersUpdate                 Permission = "bb.users.update"
	PermissionUsersUpdateEmail            Permission = "bb.users.updateEmail"
	PermissionWorksheetsGet               Permission = "bb.worksheets.get"
	PermissionWorksheetsManage            Permission = "bb.worksheets.manage"
	PermissionWorkspacesGetIamPolicy      Permission = "bb.workspaces.getIamPolicy"
	PermissionWorkspacesSetIamPolicy      Permission = "bb.workspaces.setIamPolicy"
)

var allPermissions = func() []Permission {
	var data struct {
		Permissions []Permission `yaml:"permissions"`
	}
	if err := yaml.Unmarshal(permissionYAML, &data); err != nil {
		panic("failed to load permissions from YAML: " + err.Error())
	}
	return data.Permissions
}()

var allPermissionsMap = func() map[Permission]bool {
	m := make(map[Permission]bool)
	for _, p := range allPermissions {
		m[p] = true
	}
	return m
}()

func PermissionsExist(permissions ...string) bool {
	for _, p := range permissions {
		if !PermissionExist(p) {
			return false
		}
	}
	return true
}

func PermissionExist(permission string) bool {
	return allPermissionsMap[permission]
}
```

**Step 2: Move permission.yaml**

Run: `mv backend/component/iam/permission.yaml backend/common/permission/permission.yaml`

**Step 3: Verify package compiles**

Run: `go build ./backend/common/permission/...`
Expected: Success

---

## Task 2: Create predefined roles in store

**Files:**
- Create: `backend/store/predefined_roles.go`

**Step 1: Create predefined roles file**

```go
// backend/store/predefined_roles.go
package store

import "github.com/bytebase/bytebase/backend/common/permission"

// Predefined role IDs.
const (
	WorkspaceAdminRole       = "workspaceAdmin"
	WorkspaceDBARole         = "workspaceDBA"
	WorkspaceMemberRole      = "workspaceMember"
	ProjectOwnerRole         = "projectOwner"
	ProjectDeveloperRole     = "projectDeveloper"
	SQLEditorUserRole        = "sqlEditorUser"
	ProjectReleaserRole      = "projectReleaser"
	GitOpsServiceAgentRole   = "gitopsServiceAgent"
	ProjectViewerRole        = "projectViewer"
)

func permissionSet(perms ...permission.Permission) map[string]bool {
	m := make(map[string]bool, len(perms))
	for _, p := range perms {
		m[p] = true
	}
	return m
}

var PredefinedRoles = []*RoleMessage{
	{
		ResourceID:  WorkspaceAdminRole,
		Name:        "Workspace admin",
		Permissions: permissionSet(
			permission.PermissionAuditLogsExport,
			permission.PermissionAuditLogsSearch,
			permission.PermissionChangelogsGet,
			permission.PermissionChangelogsList,
			permission.PermissionDatabaseCatalogsGet,
			permission.PermissionDatabaseCatalogsUpdate,
			permission.PermissionDatabaseGroupsCreate,
			permission.PermissionDatabaseGroupsDelete,
			permission.PermissionDatabaseGroupsGet,
			permission.PermissionDatabaseGroupsList,
			permission.PermissionDatabaseGroupsUpdate,
			permission.PermissionDatabasesCheck,
			permission.PermissionDatabasesGet,
			permission.PermissionDatabasesGetSchema,
			permission.PermissionDatabasesList,
			permission.PermissionDatabasesSync,
			permission.PermissionDatabasesUpdate,
			permission.PermissionIdentityProvidersCreate,
			permission.PermissionIdentityProvidersDelete,
			permission.PermissionIdentityProvidersGet,
			permission.PermissionIdentityProvidersUpdate,
			permission.PermissionInstancesCreate,
			permission.PermissionInstancesDelete,
			permission.PermissionInstancesGet,
			permission.PermissionInstancesList,
			permission.PermissionInstancesSync,
			permission.PermissionInstancesUndelete,
			permission.PermissionInstancesUpdate,
			permission.PermissionInstanceRolesGet,
			permission.PermissionInstanceRolesList,
			permission.PermissionIssueCommentsCreate,
			permission.PermissionIssueCommentsList,
			permission.PermissionIssueCommentsUpdate,
			permission.PermissionIssuesCreate,
			permission.PermissionIssuesGet,
			permission.PermissionIssuesList,
			permission.PermissionIssuesUpdate,
			permission.PermissionPlanCheckRunsGet,
			permission.PermissionPlanCheckRunsRun,
			permission.PermissionPlansCreate,
			permission.PermissionPlansGet,
			permission.PermissionPlansList,
			permission.PermissionPlansUpdate,
			permission.PermissionPoliciesCreate,
			permission.PermissionPoliciesDelete,
			permission.PermissionPoliciesGet,
			permission.PermissionPoliciesList,
			permission.PermissionPoliciesUpdate,
			permission.PermissionProjectsCreate,
			permission.PermissionProjectsDelete,
			permission.PermissionProjectsGet,
			permission.PermissionProjectsGetIAMPolicy,
			permission.PermissionProjectsList,
			permission.PermissionProjectsSetIAMPolicy,
			permission.PermissionProjectsUndelete,
			permission.PermissionProjectsUpdate,
			permission.PermissionReleasesCheck,
			permission.PermissionReleasesCreate,
			permission.PermissionReleasesDelete,
			permission.PermissionReleasesGet,
			permission.PermissionReleasesList,
			permission.PermissionReleasesUndelete,
			permission.PermissionReleasesUpdate,
			permission.PermissionReviewConfigsCreate,
			permission.PermissionReviewConfigsDelete,
			permission.PermissionReviewConfigsGet,
			permission.PermissionReviewConfigsList,
			permission.PermissionReviewConfigsUpdate,
			permission.PermissionRevisionsCreate,
			permission.PermissionRevisionsDelete,
			permission.PermissionRevisionsGet,
			permission.PermissionRevisionsList,
			permission.PermissionRolesCreate,
			permission.PermissionRolesDelete,
			permission.PermissionRolesList,
			permission.PermissionRolesGet,
			permission.PermissionRolesUpdate,
			permission.PermissionRolloutsCreate,
			permission.PermissionRolloutsGet,
			permission.PermissionRolloutsList,
			permission.PermissionSettingsGet,
			permission.PermissionEnvironmentSettingsGet,
			permission.PermissionWorkspaceProfileSettingsGet,
			permission.PermissionSettingsList,
			permission.PermissionSettingsSet,
			permission.PermissionEnvironmentSettingsSet,
			permission.PermissionWorkspaceProfileSettingsSet,
			permission.PermissionSheetsCreate,
			permission.PermissionSheetsGet,
			permission.PermissionSheetsUpdate,
			permission.PermissionTaskRunsCreate,
			permission.PermissionTaskRunsList,
			permission.PermissionGroupsCreate,
			permission.PermissionGroupsDelete,
			permission.PermissionGroupsGet,
			permission.PermissionGroupsList,
			permission.PermissionGroupsUpdate,
			permission.PermissionSQLAdmin,
			permission.PermissionSQLDdl,
			permission.PermissionSQLDml,
			permission.PermissionSQLExplain,
			permission.PermissionSQLInfo,
			permission.PermissionSQLSelect,
			permission.PermissionUsersCreate,
			permission.PermissionUsersDelete,
			permission.PermissionUsersGet,
			permission.PermissionUsersList,
			permission.PermissionUsersUndelete,
			permission.PermissionUsersUpdate,
			permission.PermissionUsersUpdateEmail,
			permission.PermissionWorksheetsGet,
			permission.PermissionWorksheetsManage,
			permission.PermissionWorkspacesGetIamPolicy,
			permission.PermissionWorkspacesSetIamPolicy,
		),
	},
	{
		ResourceID:  WorkspaceDBARole,
		Name:        "Workspace DBA",
		Permissions: permissionSet(
			permission.PermissionAuditLogsExport,
			permission.PermissionAuditLogsSearch,
			permission.PermissionChangelogsGet,
			permission.PermissionChangelogsList,
			permission.PermissionDatabaseCatalogsGet,
			permission.PermissionDatabaseCatalogsUpdate,
			permission.PermissionDatabaseGroupsCreate,
			permission.PermissionDatabaseGroupsDelete,
			permission.PermissionDatabaseGroupsGet,
			permission.PermissionDatabaseGroupsList,
			permission.PermissionDatabaseGroupsUpdate,
			permission.PermissionDatabasesCheck,
			permission.PermissionDatabasesGet,
			permission.PermissionDatabasesGetSchema,
			permission.PermissionDatabasesList,
			permission.PermissionDatabasesSync,
			permission.PermissionDatabasesUpdate,
			permission.PermissionInstancesCreate,
			permission.PermissionInstancesDelete,
			permission.PermissionInstancesGet,
			permission.PermissionInstancesList,
			permission.PermissionInstancesSync,
			permission.PermissionInstancesUndelete,
			permission.PermissionInstancesUpdate,
			permission.PermissionInstanceRolesGet,
			permission.PermissionInstanceRolesList,
			permission.PermissionIssueCommentsCreate,
			permission.PermissionIssueCommentsList,
			permission.PermissionIssueCommentsUpdate,
			permission.PermissionIssuesCreate,
			permission.PermissionIssuesGet,
			permission.PermissionIssuesList,
			permission.PermissionIssuesUpdate,
			permission.PermissionPlanCheckRunsGet,
			permission.PermissionPlanCheckRunsRun,
			permission.PermissionPlansCreate,
			permission.PermissionPlansGet,
			permission.PermissionPlansList,
			permission.PermissionPlansUpdate,
			permission.PermissionPoliciesCreate,
			permission.PermissionPoliciesDelete,
			permission.PermissionPoliciesGet,
			permission.PermissionPoliciesList,
			permission.PermissionPoliciesUpdate,
			permission.PermissionProjectsCreate,
			permission.PermissionProjectsDelete,
			permission.PermissionProjectsGet,
			permission.PermissionProjectsGetIAMPolicy,
			permission.PermissionProjectsList,
			permission.PermissionProjectsSetIAMPolicy,
			permission.PermissionProjectsUndelete,
			permission.PermissionProjectsUpdate,
			permission.PermissionReleasesCheck,
			permission.PermissionReleasesCreate,
			permission.PermissionReleasesDelete,
			permission.PermissionReleasesGet,
			permission.PermissionReleasesList,
			permission.PermissionReleasesUndelete,
			permission.PermissionReleasesUpdate,
			permission.PermissionReviewConfigsCreate,
			permission.PermissionReviewConfigsDelete,
			permission.PermissionReviewConfigsGet,
			permission.PermissionReviewConfigsList,
			permission.PermissionReviewConfigsUpdate,
			permission.PermissionRevisionsCreate,
			permission.PermissionRevisionsDelete,
			permission.PermissionRevisionsGet,
			permission.PermissionRevisionsList,
			permission.PermissionRolesCreate,
			permission.PermissionRolesDelete,
			permission.PermissionRolesList,
			permission.PermissionRolesGet,
			permission.PermissionRolesUpdate,
			permission.PermissionRolloutsCreate,
			permission.PermissionRolloutsGet,
			permission.PermissionRolloutsList,
			permission.PermissionSettingsGet,
			permission.PermissionEnvironmentSettingsGet,
			permission.PermissionWorkspaceProfileSettingsGet,
			permission.PermissionSettingsList,
			permission.PermissionSettingsSet,
			permission.PermissionEnvironmentSettingsSet,
			permission.PermissionWorkspaceProfileSettingsSet,
			permission.PermissionSheetsCreate,
			permission.PermissionSheetsGet,
			permission.PermissionSheetsUpdate,
			permission.PermissionSQLAdmin,
			permission.PermissionSQLDdl,
			permission.PermissionSQLDml,
			permission.PermissionSQLExplain,
			permission.PermissionSQLInfo,
			permission.PermissionSQLSelect,
			permission.PermissionUsersGet,
			permission.PermissionUsersList,
			permission.PermissionTaskRunsCreate,
			permission.PermissionTaskRunsList,
			permission.PermissionGroupsGet,
			permission.PermissionGroupsList,
			permission.PermissionWorksheetsGet,
			permission.PermissionWorksheetsManage,
			permission.PermissionWorkspacesGetIamPolicy,
		),
	},
	{
		ResourceID:  WorkspaceMemberRole,
		Name:        "Workspace member",
		Permissions: permissionSet(
			permission.PermissionInstanceRolesGet,
			permission.PermissionInstanceRolesList,
			permission.PermissionPoliciesGet,
			permission.PermissionPoliciesList,
			permission.PermissionReviewConfigsGet,
			permission.PermissionReviewConfigsList,
			permission.PermissionRolesList,
			permission.PermissionRolesGet,
			permission.PermissionEnvironmentSettingsGet,
			permission.PermissionWorkspaceProfileSettingsGet,
			permission.PermissionGroupsCreate,
			permission.PermissionGroupsGet,
			permission.PermissionGroupsList,
			permission.PermissionUsersGet,
			permission.PermissionUsersList,
			permission.PermissionWorkspacesGetIamPolicy,
		),
	},
	{
		ResourceID:  ProjectOwnerRole,
		Name:        "Project owner",
		Permissions: permissionSet(
			permission.PermissionAuditLogsExport,
			permission.PermissionAuditLogsSearch,
			permission.PermissionChangelogsGet,
			permission.PermissionChangelogsList,
			permission.PermissionDatabaseCatalogsGet,
			permission.PermissionDatabaseCatalogsUpdate,
			permission.PermissionDatabaseGroupsCreate,
			permission.PermissionDatabaseGroupsDelete,
			permission.PermissionDatabaseGroupsGet,
			permission.PermissionDatabaseGroupsList,
			permission.PermissionDatabaseGroupsUpdate,
			permission.PermissionDatabasesCheck,
			permission.PermissionDatabasesGet,
			permission.PermissionDatabasesGetSchema,
			permission.PermissionDatabasesList,
			permission.PermissionDatabasesSync,
			permission.PermissionDatabasesUpdate,
			permission.PermissionIssueCommentsCreate,
			permission.PermissionIssueCommentsList,
			permission.PermissionIssueCommentsUpdate,
			permission.PermissionIssuesCreate,
			permission.PermissionIssuesGet,
			permission.PermissionIssuesList,
			permission.PermissionIssuesUpdate,
			permission.PermissionPlanCheckRunsGet,
			permission.PermissionPlanCheckRunsRun,
			permission.PermissionPlansCreate,
			permission.PermissionPlansGet,
			permission.PermissionPlansList,
			permission.PermissionPlansUpdate,
			permission.PermissionPoliciesCreate,
			permission.PermissionPoliciesDelete,
			permission.PermissionPoliciesGet,
			permission.PermissionPoliciesList,
			permission.PermissionPoliciesUpdate,
			permission.PermissionProjectsGet,
			permission.PermissionProjectsGetIAMPolicy,
			permission.PermissionProjectsSetIAMPolicy,
			permission.PermissionProjectsUpdate,
			permission.PermissionReleasesCheck,
			permission.PermissionReleasesCreate,
			permission.PermissionReleasesDelete,
			permission.PermissionReleasesGet,
			permission.PermissionReleasesList,
			permission.PermissionReleasesUndelete,
			permission.PermissionReleasesUpdate,
			permission.PermissionRevisionsCreate,
			permission.PermissionRevisionsDelete,
			permission.PermissionRevisionsGet,
			permission.PermissionRevisionsList,
			permission.PermissionRolloutsCreate,
			permission.PermissionRolloutsGet,
			permission.PermissionRolloutsList,
			permission.PermissionSheetsCreate,
			permission.PermissionSheetsGet,
			permission.PermissionSheetsUpdate,
			permission.PermissionSQLDdl,
			permission.PermissionSQLDml,
			permission.PermissionSQLExplain,
			permission.PermissionSQLInfo,
			permission.PermissionSQLSelect,
			permission.PermissionUsersGet,
			permission.PermissionUsersList,
			permission.PermissionTaskRunsList,
			permission.PermissionWorksheetsGet,
		),
	},
	{
		ResourceID:  ProjectDeveloperRole,
		Name:        "Project developer",
		Permissions: permissionSet(
			permission.PermissionAuditLogsSearch,
			permission.PermissionChangelogsGet,
			permission.PermissionChangelogsList,
			permission.PermissionDatabaseCatalogsGet,
			permission.PermissionDatabaseGroupsGet,
			permission.PermissionDatabaseGroupsList,
			permission.PermissionDatabasesCheck,
			permission.PermissionDatabasesGet,
			permission.PermissionDatabasesGetSchema,
			permission.PermissionDatabasesList,
			permission.PermissionDatabasesSync,
			permission.PermissionIssueCommentsCreate,
			permission.PermissionIssueCommentsList,
			permission.PermissionIssueCommentsUpdate,
			permission.PermissionIssuesCreate,
			permission.PermissionIssuesGet,
			permission.PermissionIssuesList,
			permission.PermissionIssuesUpdate,
			permission.PermissionPlanCheckRunsGet,
			permission.PermissionPlanCheckRunsRun,
			permission.PermissionPlansCreate,
			permission.PermissionPlansGet,
			permission.PermissionPlansList,
			permission.PermissionProjectsGet,
			permission.PermissionProjectsGetIAMPolicy,
			permission.PermissionReleasesCheck,
			permission.PermissionReleasesGet,
			permission.PermissionReleasesList,
			permission.PermissionRevisionsGet,
			permission.PermissionRevisionsList,
			permission.PermissionRolloutsCreate,
			permission.PermissionRolloutsGet,
			permission.PermissionRolloutsList,
			permission.PermissionSheetsCreate,
			permission.PermissionSheetsGet,
			permission.PermissionSheetsUpdate,
			permission.PermissionTaskRunsList,
		),
	},
	{
		ResourceID:  SQLEditorUserRole,
		Name:        "SQL Editor User",
		Permissions: permissionSet(
			permission.PermissionDatabaseCatalogsGet,
			permission.PermissionDatabaseGroupsGet,
			permission.PermissionDatabaseGroupsList,
			permission.PermissionDatabasesCheck,
			permission.PermissionDatabasesGet,
			permission.PermissionDatabasesGetSchema,
			permission.PermissionDatabasesList,
			permission.PermissionDatabasesSync,
			permission.PermissionProjectsGet,
			permission.PermissionProjectsGetIAMPolicy,
			permission.PermissionSQLDdl,
			permission.PermissionSQLDml,
			permission.PermissionSQLExplain,
			permission.PermissionSQLInfo,
			permission.PermissionSQLSelect,
			permission.PermissionWorksheetsGet,
		),
	},
	{
		ResourceID:  ProjectReleaserRole,
		Name:        "Project releaser",
		Permissions: permissionSet(
			permission.PermissionDatabaseCatalogsGet,
			permission.PermissionChangelogsGet,
			permission.PermissionDatabaseGroupsGet,
			permission.PermissionDatabaseGroupsList,
			permission.PermissionDatabasesCheck,
			permission.PermissionDatabasesGet,
			permission.PermissionDatabasesGetSchema,
			permission.PermissionDatabasesList,
			permission.PermissionIssueCommentsCreate,
			permission.PermissionIssueCommentsList,
			permission.PermissionIssueCommentsUpdate,
			permission.PermissionIssuesGet,
			permission.PermissionIssuesList,
			permission.PermissionPlanCheckRunsGet,
			permission.PermissionPlanCheckRunsRun,
			permission.PermissionPlansGet,
			permission.PermissionPlansList,
			permission.PermissionProjectsGet,
			permission.PermissionProjectsGetIAMPolicy,
			permission.PermissionReleasesGet,
			permission.PermissionReleasesList,
			permission.PermissionRevisionsGet,
			permission.PermissionRevisionsList,
			permission.PermissionRolloutsCreate,
			permission.PermissionRolloutsGet,
			permission.PermissionRolloutsList,
			permission.PermissionSheetsGet,
			permission.PermissionTaskRunsCreate,
			permission.PermissionTaskRunsList,
		),
	},
	{
		ResourceID:  GitOpsServiceAgentRole,
		Name:        "GitOps Service Agent",
		Permissions: permissionSet(
			permission.PermissionPlansCreate,
			permission.PermissionPlansGet,
			permission.PermissionReleasesCheck,
			permission.PermissionReleasesCreate,
			permission.PermissionReleasesGet,
			permission.PermissionRolloutsCreate,
			permission.PermissionRolloutsGet,
			permission.PermissionRolloutsList,
			permission.PermissionTaskRunsCreate,
			permission.PermissionTaskRunsList,
		),
	},
	{
		ResourceID:  ProjectViewerRole,
		Name:        "Project viewer",
		Permissions: permissionSet(
			permission.PermissionDatabaseCatalogsGet,
			permission.PermissionDatabaseGroupsGet,
			permission.PermissionDatabaseGroupsList,
			permission.PermissionDatabasesGet,
			permission.PermissionDatabasesGetSchema,
			permission.PermissionDatabasesList,
			permission.PermissionIssueCommentsCreate,
			permission.PermissionIssueCommentsList,
			permission.PermissionIssueCommentsUpdate,
			permission.PermissionIssuesCreate,
			permission.PermissionProjectsGet,
			permission.PermissionProjectsGetIAMPolicy,
			permission.PermissionIssuesGet,
		),
	},
}

// predefinedRolesMap is a map from role ID to RoleMessage for fast lookup.
var predefinedRolesMap = func() map[string]*RoleMessage {
	m := make(map[string]*RoleMessage, len(PredefinedRoles))
	for _, role := range PredefinedRoles {
		m[role.ResourceID] = role
	}
	return m
}()

// GetPredefinedRole returns a predefined role by ID.
func GetPredefinedRole(roleID string) *RoleMessage {
	return predefinedRolesMap[roleID]
}

// IsPredefinedRole checks if a role ID is a predefined role.
func IsPredefinedRole(roleID string) bool {
	_, ok := predefinedRolesMap[roleID]
	return ok
}
```

**Step 2: Verify compilation**

Run: `go build ./backend/store/...`
Expected: Success

---

## Task 3: Update store ListRoles to include predefined roles

**Files:**
- Modify: `backend/store/role.go:134-182`

**Step 1: Update ListRoles function**

In `backend/store/role.go`, change the `ListRoles` function to append predefined roles:

```go
// ListRoles returns a list of roles.
func (s *Store) ListRoles(ctx context.Context, find *FindRoleMessage) ([]*RoleMessage, error) {
	// If looking for a specific role, check predefined first
	if v := find.ResourceID; v != nil {
		if role := GetPredefinedRole(*v); role != nil {
			return []*RoleMessage{role}, nil
		}
	}

	q := qb.Q().Space(`
		SELECT
			resource_id, name, description, permissions
		FROM role
		WHERE TRUE
	`)

	if v := find.ResourceID; v != nil {
		q.And("resource_id = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*RoleMessage
	for rows.Next() {
		role := &RoleMessage{
			Permissions: map[string]bool{},
		}
		var permissionBytes []byte
		if err := rows.Scan(&role.ResourceID, &role.Name, &role.Description, &permissionBytes); err != nil {
			return nil, err
		}
		var rolePermissions storepb.RolePermissions
		if err := common.ProtojsonUnmarshaler.Unmarshal(permissionBytes, &rolePermissions); err != nil {
			return nil, err
		}
		for _, v := range rolePermissions.Permissions {
			role.Permissions[v] = true
		}
		s.rolesCache.Add(role.ResourceID, role)
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Append predefined roles when listing all roles
	if find.ResourceID == nil {
		roles = append(roles, PredefinedRoles...)
	}

	return roles, nil
}
```

**Step 2: Verify compilation**

Run: `go build ./backend/store/...`
Expected: Success

---

## Task 4: Update IAM manager to use shared permission package

**Files:**
- Modify: `backend/component/iam/permission.go`
- Modify: `backend/component/iam/manager.go`

**Step 1: Update permission.go to re-export from shared package**

Replace `backend/component/iam/permission.go` with:

```go
package iam

import "github.com/bytebase/bytebase/backend/common/permission"

// Re-export Permission type and functions from shared package.
type Permission = permission.Permission

// Re-export all permission constants.
const (
	PermissionAuditLogsExport             = permission.PermissionAuditLogsExport
	PermissionAuditLogsSearch             = permission.PermissionAuditLogsSearch
	PermissionChangelogsGet               = permission.PermissionChangelogsGet
	PermissionChangelogsList              = permission.PermissionChangelogsList
	PermissionDatabaseCatalogsGet         = permission.PermissionDatabaseCatalogsGet
	PermissionDatabaseCatalogsUpdate      = permission.PermissionDatabaseCatalogsUpdate
	PermissionDatabaseGroupsCreate        = permission.PermissionDatabaseGroupsCreate
	PermissionDatabaseGroupsDelete        = permission.PermissionDatabaseGroupsDelete
	PermissionDatabaseGroupsGet           = permission.PermissionDatabaseGroupsGet
	PermissionDatabaseGroupsList          = permission.PermissionDatabaseGroupsList
	PermissionDatabaseGroupsUpdate        = permission.PermissionDatabaseGroupsUpdate
	PermissionDatabasesCheck              = permission.PermissionDatabasesCheck
	PermissionDatabasesGet                = permission.PermissionDatabasesGet
	PermissionDatabasesGetSchema          = permission.PermissionDatabasesGetSchema
	PermissionDatabasesList               = permission.PermissionDatabasesList
	PermissionDatabasesSync               = permission.PermissionDatabasesSync
	PermissionDatabasesUpdate             = permission.PermissionDatabasesUpdate
	PermissionIdentityProvidersCreate     = permission.PermissionIdentityProvidersCreate
	PermissionIdentityProvidersDelete     = permission.PermissionIdentityProvidersDelete
	PermissionIdentityProvidersGet        = permission.PermissionIdentityProvidersGet
	PermissionIdentityProvidersUpdate     = permission.PermissionIdentityProvidersUpdate
	PermissionInstancesCreate             = permission.PermissionInstancesCreate
	PermissionInstancesDelete             = permission.PermissionInstancesDelete
	PermissionInstancesGet                = permission.PermissionInstancesGet
	PermissionInstancesList               = permission.PermissionInstancesList
	PermissionInstancesSync               = permission.PermissionInstancesSync
	PermissionInstancesUndelete           = permission.PermissionInstancesUndelete
	PermissionInstancesUpdate             = permission.PermissionInstancesUpdate
	PermissionInstanceRolesGet            = permission.PermissionInstanceRolesGet
	PermissionInstanceRolesList           = permission.PermissionInstanceRolesList
	PermissionIssueCommentsCreate         = permission.PermissionIssueCommentsCreate
	PermissionIssueCommentsList           = permission.PermissionIssueCommentsList
	PermissionIssueCommentsUpdate         = permission.PermissionIssueCommentsUpdate
	PermissionIssuesCreate                = permission.PermissionIssuesCreate
	PermissionIssuesGet                   = permission.PermissionIssuesGet
	PermissionIssuesList                  = permission.PermissionIssuesList
	PermissionIssuesUpdate                = permission.PermissionIssuesUpdate
	PermissionPlanCheckRunsGet            = permission.PermissionPlanCheckRunsGet
	PermissionPlanCheckRunsRun            = permission.PermissionPlanCheckRunsRun
	PermissionPlansCreate                 = permission.PermissionPlansCreate
	PermissionPlansGet                    = permission.PermissionPlansGet
	PermissionPlansList                   = permission.PermissionPlansList
	PermissionPlansUpdate                 = permission.PermissionPlansUpdate
	PermissionPoliciesCreate              = permission.PermissionPoliciesCreate
	PermissionPoliciesDelete              = permission.PermissionPoliciesDelete
	PermissionPoliciesGet                 = permission.PermissionPoliciesGet
	PermissionPoliciesList                = permission.PermissionPoliciesList
	PermissionPoliciesUpdate              = permission.PermissionPoliciesUpdate
	PermissionProjectsCreate              = permission.PermissionProjectsCreate
	PermissionProjectsDelete              = permission.PermissionProjectsDelete
	PermissionProjectsGet                 = permission.PermissionProjectsGet
	PermissionProjectsGetIAMPolicy        = permission.PermissionProjectsGetIAMPolicy
	PermissionProjectsList                = permission.PermissionProjectsList
	PermissionProjectsSetIAMPolicy        = permission.PermissionProjectsSetIAMPolicy
	PermissionProjectsUndelete            = permission.PermissionProjectsUndelete
	PermissionProjectsUpdate              = permission.PermissionProjectsUpdate
	PermissionReleasesCheck               = permission.PermissionReleasesCheck
	PermissionReleasesCreate              = permission.PermissionReleasesCreate
	PermissionReleasesDelete              = permission.PermissionReleasesDelete
	PermissionReleasesGet                 = permission.PermissionReleasesGet
	PermissionReleasesList                = permission.PermissionReleasesList
	PermissionReleasesUndelete            = permission.PermissionReleasesUndelete
	PermissionReleasesUpdate              = permission.PermissionReleasesUpdate
	PermissionReviewConfigsCreate         = permission.PermissionReviewConfigsCreate
	PermissionReviewConfigsDelete         = permission.PermissionReviewConfigsDelete
	PermissionReviewConfigsGet            = permission.PermissionReviewConfigsGet
	PermissionReviewConfigsList           = permission.PermissionReviewConfigsList
	PermissionReviewConfigsUpdate         = permission.PermissionReviewConfigsUpdate
	PermissionRevisionsCreate             = permission.PermissionRevisionsCreate
	PermissionRevisionsDelete             = permission.PermissionRevisionsDelete
	PermissionRevisionsGet                = permission.PermissionRevisionsGet
	PermissionRevisionsList               = permission.PermissionRevisionsList
	PermissionRolesCreate                 = permission.PermissionRolesCreate
	PermissionRolesDelete                 = permission.PermissionRolesDelete
	PermissionRolesList                   = permission.PermissionRolesList
	PermissionRolesGet                    = permission.PermissionRolesGet
	PermissionRolesUpdate                 = permission.PermissionRolesUpdate
	PermissionRolloutsCreate              = permission.PermissionRolloutsCreate
	PermissionRolloutsGet                 = permission.PermissionRolloutsGet
	PermissionRolloutsList                = permission.PermissionRolloutsList
	PermissionSettingsGet                 = permission.PermissionSettingsGet
	PermissionSettingsList                = permission.PermissionSettingsList
	PermissionSettingsSet                 = permission.PermissionSettingsSet
	PermissionEnvironmentSettingsGet      = permission.PermissionEnvironmentSettingsGet
	PermissionEnvironmentSettingsSet      = permission.PermissionEnvironmentSettingsSet
	PermissionWorkspaceProfileSettingsGet = permission.PermissionWorkspaceProfileSettingsGet
	PermissionWorkspaceProfileSettingsSet = permission.PermissionWorkspaceProfileSettingsSet
	PermissionSheetsCreate                = permission.PermissionSheetsCreate
	PermissionSheetsGet                   = permission.PermissionSheetsGet
	PermissionSheetsUpdate                = permission.PermissionSheetsUpdate
	PermissionSQLSelect                   = permission.PermissionSQLSelect
	PermissionSQLDdl                      = permission.PermissionSQLDdl
	PermissionSQLDml                      = permission.PermissionSQLDml
	PermissionSQLExplain                  = permission.PermissionSQLExplain
	PermissionSQLInfo                     = permission.PermissionSQLInfo
	PermissionSQLAdmin                    = permission.PermissionSQLAdmin
	PermissionTaskRunsCreate              = permission.PermissionTaskRunsCreate
	PermissionTaskRunsList                = permission.PermissionTaskRunsList
	PermissionGroupsCreate                = permission.PermissionGroupsCreate
	PermissionGroupsDelete                = permission.PermissionGroupsDelete
	PermissionGroupsGet                   = permission.PermissionGroupsGet
	PermissionGroupsList                  = permission.PermissionGroupsList
	PermissionGroupsUpdate                = permission.PermissionGroupsUpdate
	PermissionUsersCreate                 = permission.PermissionUsersCreate
	PermissionUsersDelete                 = permission.PermissionUsersDelete
	PermissionUsersGet                    = permission.PermissionUsersGet
	PermissionUsersList                   = permission.PermissionUsersList
	PermissionUsersUndelete               = permission.PermissionUsersUndelete
	PermissionUsersUpdate                 = permission.PermissionUsersUpdate
	PermissionUsersUpdateEmail            = permission.PermissionUsersUpdateEmail
	PermissionWorksheetsGet               = permission.PermissionWorksheetsGet
	PermissionWorksheetsManage            = permission.PermissionWorksheetsManage
	PermissionWorkspacesGetIamPolicy      = permission.PermissionWorkspacesGetIamPolicy
	PermissionWorkspacesSetIamPolicy      = permission.PermissionWorkspacesSetIamPolicy
)

// Re-export functions.
var (
	PermissionsExist = permission.PermissionsExist
	PermissionExist  = permission.PermissionExist
)
```

**Step 2: Update manager.go**

Remove from `backend/component/iam/manager.go`:
- Remove `//go:embed acl.yaml` and `aclYaml` variable
- Remove `acl` struct type
- Remove `PredefinedRoles` field from `Manager` struct
- Remove `loadPredefinedRoles()` function
- Update `NewManager()` to not load predefined roles
- Update `ReloadCache()` to not append predefined roles (store already includes them)

New `manager.go`:

```go
package iam

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

type Manager struct {
	// rolePermissions is a map from role to permissions. Key is "roles/{role}".
	rolePermissions map[string]map[Permission]bool
	groupMembers    map[string]map[string]bool
	// member - groups mapping
	memberGroups   map[string][]string
	store          *store.Store
	licenseService *enterprise.LicenseService
}

func NewManager(store *store.Store, licenseService *enterprise.LicenseService) (*Manager, error) {
	m := &Manager{
		store:          store,
		licenseService: licenseService,
	}
	return m, nil
}

// Check if the user has permission on the resource hierarchy.
// CEL on the binding is not considered.
// When multiple projects are specified, the user should have permission on every projects.
func (m *Manager) CheckPermission(ctx context.Context, p Permission, user *store.UserMessage, projectIDs ...string) (bool, error) {
	policyMessage, err := m.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return false, err
	}
	if ok := check(user, p, policyMessage.Policy, m.rolePermissions, m.groupMembers); ok {
		return true, nil
	}

	if len(projectIDs) > 0 {
		allOK := true
		for _, projectID := range projectIDs {
			project, err := m.store.GetProject(ctx, &store.FindProjectMessage{
				ResourceID:  &projectID,
				ShowDeleted: true,
			})
			if err != nil {
				return false, err
			}
			if project == nil {
				return false, errors.Errorf("project %q not found", projectID)
			}
			policyMessage, err := m.store.GetProjectIamPolicy(ctx, project.ResourceID)
			if err != nil {
				return false, err
			}
			if ok := check(user, p, policyMessage.Policy, m.rolePermissions, m.groupMembers); !ok {
				allOK = false
				break
			}
		}
		return allOK, nil
	}
	return false, nil
}

func (m *Manager) ReloadCache(ctx context.Context) error {
	// ListRoles now returns custom + predefined roles
	roles, err := m.store.ListRoles(ctx, &store.FindRoleMessage{})
	if err != nil {
		return err
	}

	rolePermissions := make(map[string]map[Permission]bool)
	for _, role := range roles {
		rolePermissions[common.FormatRole(role.ResourceID)] = role.Permissions
	}
	m.rolePermissions = rolePermissions

	groups, err := m.store.ListGroups(ctx, &store.FindGroupMessage{})
	if err != nil {
		return err
	}
	groupMembers := make(map[string]map[string]bool)
	memberGroups := make(map[string][]string)
	for _, group := range groups {
		usersSet := make(map[string]bool)
		groupName := utils.FormatGroupName(group)
		for _, m := range group.Payload.GetMembers() {
			usersSet[m.Member] = true
			if _, ok := memberGroups[m.Member]; !ok {
				memberGroups[m.Member] = []string{}
			}
			memberGroups[m.Member] = append(memberGroups[m.Member], groupName)
		}
		groupMembers[groupName] = usersSet
	}
	m.groupMembers = groupMembers
	m.memberGroups = memberGroups
	return nil
}

// GetPermissions returns all permissions for the given role.
// Role format is roles/{role}.
func (m *Manager) GetPermissions(role string) (map[Permission]bool, error) {
	permissions, ok := m.rolePermissions[role]
	if !ok {
		return nil, nil
	}
	return permissions, nil
}

func (m *Manager) GetUserGroups(email string) []string {
	return m.memberGroups[common.FormatUserEmail(email)]
}

func check(user *store.UserMessage, p Permission, policy *storepb.IamPolicy, rolePermissions map[string]map[Permission]bool, groupMembers map[string]map[string]bool) bool {
	userName := common.FormatUserEmail(user.Email)

	for _, binding := range policy.GetBindings() {
		permissions, ok := rolePermissions[binding.GetRole()]
		if !ok {
			continue
		}
		if !permissions[p] {
			continue
		}
		for _, member := range binding.GetMembers() {
			if member == common.AllUsers {
				return true
			}
			if member == userName {
				return true
			}
			if strings.HasPrefix(member, common.GroupPrefix) {
				if groupMembers, ok := groupMembers[member]; ok {
					if groupMembers[userName] {
						return true
					}
				}
			}
		}
	}
	return false
}
```

**Step 3: Verify compilation**

Run: `go build ./backend/component/iam/...`
Expected: Success

---

## Task 5: Update role_service.go

**Files:**
- Modify: `backend/api/v1/role_service.go`

**Step 1: Update ListRoles**

Change `ListRoles` to not manually append predefined roles:

```go
// ListRoles lists roles.
func (s *RoleService) ListRoles(ctx context.Context, _ *connect.Request[v1pb.ListRolesRequest]) (*connect.Response[v1pb.ListRolesResponse], error) {
	roleMessages, err := s.store.ListRoles(ctx, &store.FindRoleMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to list roles"))
	}

	var roles []*v1pb.Role
	for _, role := range roleMessages {
		roleType := v1pb.Role_CUSTOM
		if store.IsPredefinedRole(role.ResourceID) {
			roleType = v1pb.Role_BUILT_IN
		}
		roles = append(roles, convertToRole(role, roleType))
	}

	return connect.NewResponse(&v1pb.ListRolesResponse{
		Roles: roles,
	}), nil
}
```

**Step 2: Update GetRole**

Change `GetRole` to use store lookup:

```go
// GetRole gets a role.
func (s *RoleService) GetRole(ctx context.Context, req *connect.Request[v1pb.GetRoleRequest]) (*connect.Response[v1pb.Role], error) {
	roleName := req.Msg.Name
	roleID, err := common.GetRoleID(roleName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	roles, err := s.store.ListRoles(ctx, &store.FindRoleMessage{ResourceID: &roleID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get role"))
	}
	if len(roles) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("role not found: %s", roleID))
	}

	role := roles[0]
	roleType := v1pb.Role_CUSTOM
	if store.IsPredefinedRole(role.ResourceID) {
		roleType = v1pb.Role_BUILT_IN
	}
	return connect.NewResponse(convertToRole(role, roleType)), nil
}
```

**Step 3: Update CreateRole, UpdateRole, DeleteRole**

Replace `s.getBuildinRole()` calls with `store.IsPredefinedRole()`:

In `CreateRole`:
```go
if store.IsPredefinedRole(req.Msg.RoleId) {
	return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("role %s is a built-in role", req.Msg.RoleId))
}
```

In `UpdateRole`:
```go
if store.IsPredefinedRole(roleID) {
	return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot change the build-in role %s", req.Msg.Role.Name))
}
```

In `DeleteRole`:
```go
if store.IsPredefinedRole(roleID) {
	return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot delete the build-in role %s", req.Msg.Name))
}
```

**Step 4: Remove getBuildinRole function**

Delete the `getBuildinRole` method entirely.

**Step 5: Verify compilation**

Run: `go build ./backend/api/v1/...`
Expected: Success

---

## Task 6: Update project_service.go

**Files:**
- Modify: `backend/api/v1/project_service.go:1090`

**Step 1: Remove PredefinedRoles append**

Find line ~1090 and remove the append:

Before:
```go
roleMessages, err := stores.ListRoles(ctx, &store.FindRoleMessage{})
if err != nil {
	return false, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list roles"))
}
roleMessages = append(roleMessages, iamManager.PredefinedRoles...)
```

After:
```go
roleMessages, err := stores.ListRoles(ctx, &store.FindRoleMessage{})
if err != nil {
	return false, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list roles"))
}
// ListRoles already includes predefined roles
```

**Step 2: Verify compilation**

Run: `go build ./backend/api/v1/...`
Expected: Success

---

## Task 7: Update tests

**Files:**
- Modify: `backend/component/iam/permission_test.go`

**Step 1: Update permission_test.go**

Update test to use store.PredefinedRoles:

```go
package iam

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/store"
)

// Test that every permission in predefined roles is also defined as a constant.
func TestPermissionExists(t *testing.T) {
	a := require.New(t)

	for _, role := range store.PredefinedRoles {
		for p := range role.Permissions {
			exist := permission.PermissionsExist(p)
			a.True(exist, "permission %s is not defined as a constant", p)
		}
	}
}

// Test that permissions are equal in permission.yaml and allPermissions.
func TestPermissionEquals(t *testing.T) {
	a := require.New(t)

	// This test is now in the permission package
	// Just verify the re-exports work
	a.True(PermissionExist(string(PermissionInstancesCreate)))
}
```

**Step 2: Verify tests pass**

Run: `go test -v ./backend/component/iam/...`
Expected: PASS

---

## Task 8: Update frontend copy script

**Files:**
- Modify: `frontend/scripts/copy_config_files.sh`

**Step 1: Update path**

```bash
#!/bin/sh
cp ../backend/enterprise/plan.yaml ./src/types/
cp ../backend/common/permission/permission.yaml ./src/types/iam/
node ./scripts/generate_permissions.js
```

**Step 2: Test script**

Run: `cd frontend && ./scripts/copy_config_files.sh`
Expected: Success, permission.yaml copied

---

## Task 9: Delete acl.yaml

**Files:**
- Delete: `backend/component/iam/acl.yaml`

**Step 1: Delete file**

Run: `rm backend/component/iam/acl.yaml`

**Step 2: Verify full build**

Run: `go build ./backend/...`
Expected: Success

---

## Task 10: Run linters and tests

**Step 1: Run gofmt**

Run: `gofmt -w backend/common/permission/ backend/store/predefined_roles.go backend/component/iam/ backend/api/v1/role_service.go backend/api/v1/project_service.go`

**Step 2: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No errors

**Step 3: Run tests**

Run: `go test ./backend/component/iam/... ./backend/store/... ./backend/api/v1/...`
Expected: All pass

---

## Task 11: Commit

Run:
```bash
but status
but commit <branch> -m "refactor(iam): move predefined roles to store package

- Create backend/common/permission/ package for shared permission types
- Move permission.yaml to backend/common/permission/
- Define predefined roles as Go code in backend/store/predefined_roles.go
- Store.ListRoles() now returns both custom and predefined roles
- IAM manager focuses on permission checking only
- Update frontend copy script path"
```
