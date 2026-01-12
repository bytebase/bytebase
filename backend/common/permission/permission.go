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

func Exists(permissions ...string) bool {
	for _, p := range permissions {
		if !Exist(p) {
			return false
		}
	}
	return true
}

func Exist(permission string) bool {
	return allPermissionsMap[permission]
}
