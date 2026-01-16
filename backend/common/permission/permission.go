package permission

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed permission.yaml
var permissionYAML []byte

type Permission = string

const (
	AuditLogsExport                      Permission = "bb.auditLogs.export"
	AuditLogsSearch                      Permission = "bb.auditLogs.search"
	ChangelogsGet                        Permission = "bb.changelogs.get"
	ChangelogsList                       Permission = "bb.changelogs.list"
	DatabaseCatalogsGet                  Permission = "bb.databaseCatalogs.get"
	DatabaseCatalogsUpdate               Permission = "bb.databaseCatalogs.update"
	DatabaseGroupsCreate                 Permission = "bb.databaseGroups.create"
	DatabaseGroupsDelete                 Permission = "bb.databaseGroups.delete"
	DatabaseGroupsGet                    Permission = "bb.databaseGroups.get"
	DatabaseGroupsList                   Permission = "bb.databaseGroups.list"
	DatabaseGroupsUpdate                 Permission = "bb.databaseGroups.update"
	DatabasesCheck                       Permission = "bb.databases.check"
	DatabasesGet                         Permission = "bb.databases.get"
	DatabasesGetSchema                   Permission = "bb.databases.getSchema"
	DatabasesList                        Permission = "bb.databases.list"
	DatabasesSync                        Permission = "bb.databases.sync"
	DatabasesUpdate                      Permission = "bb.databases.update"
	IdentityProvidersCreate              Permission = "bb.identityProviders.create"
	IdentityProvidersDelete              Permission = "bb.identityProviders.delete"
	IdentityProvidersGet                 Permission = "bb.identityProviders.get"
	IdentityProvidersUpdate              Permission = "bb.identityProviders.update"
	InstancesCreate                      Permission = "bb.instances.create"
	InstancesDelete                      Permission = "bb.instances.delete"
	InstancesGet                         Permission = "bb.instances.get"
	InstancesList                        Permission = "bb.instances.list"
	InstancesSync                        Permission = "bb.instances.sync"
	InstancesUndelete                    Permission = "bb.instances.undelete"
	InstancesUpdate                      Permission = "bb.instances.update"
	InstanceRolesGet                     Permission = "bb.instanceRoles.get"
	InstanceRolesList                    Permission = "bb.instanceRoles.list"
	IssueCommentsCreate                  Permission = "bb.issueComments.create"
	IssueCommentsList                    Permission = "bb.issueComments.list"
	IssueCommentsUpdate                  Permission = "bb.issueComments.update"
	IssuesCreate                         Permission = "bb.issues.create"
	IssuesGet                            Permission = "bb.issues.get"
	IssuesList                           Permission = "bb.issues.list"
	IssuesUpdate                         Permission = "bb.issues.update"
	PlanCheckRunsGet                     Permission = "bb.planCheckRuns.get"
	PlanCheckRunsRun                     Permission = "bb.planCheckRuns.run"
	PlansCreate                          Permission = "bb.plans.create"
	PlansGet                             Permission = "bb.plans.get"
	PlansList                            Permission = "bb.plans.list"
	PlansUpdate                          Permission = "bb.plans.update"
	PoliciesCreate                       Permission = "bb.policies.create"
	PoliciesDelete                       Permission = "bb.policies.delete"
	PoliciesGet                          Permission = "bb.policies.get"
	PoliciesList                         Permission = "bb.policies.list"
	PoliciesUpdate                       Permission = "bb.policies.update"
	PoliciesGetMaskingRulePolicy         Permission = "bb.policies.getMaskingRulePolicy"
	PoliciesUpdateMaskingRulePolicy      Permission = "bb.policies.updateMaskingRulePolicy"
	PoliciesCreateMaskingRulePolicy      Permission = "bb.policies.createMaskingRulePolicy"
	PoliciesDeleteMaskingRulePolicy      Permission = "bb.policies.deleteMaskingRulePolicy"
	PoliciesGetMaskingExemptionPolicy    Permission = "bb.policies.getMaskingExemptionPolicy"
	PoliciesUpdateMaskingExemptionPolicy Permission = "bb.policies.updateMaskingExemptionPolicy"
	PoliciesCreateMaskingExemptionPolicy Permission = "bb.policies.createMaskingExemptionPolicy"
	PoliciesDeleteMaskingExemptionPolicy Permission = "bb.policies.deleteMaskingExemptionPolicy"
	ProjectsCreate                       Permission = "bb.projects.create"
	ProjectsDelete                       Permission = "bb.projects.delete"
	ProjectsGet                          Permission = "bb.projects.get"
	ProjectsGetIAMPolicy                 Permission = "bb.projects.getIamPolicy"
	ProjectsList                         Permission = "bb.projects.list"
	ProjectsSetIAMPolicy                 Permission = "bb.projects.setIamPolicy"
	ProjectsUndelete                     Permission = "bb.projects.undelete"
	ProjectsUpdate                       Permission = "bb.projects.update"
	ReleasesCheck                        Permission = "bb.releases.check"
	ReleasesCreate                       Permission = "bb.releases.create"
	ReleasesDelete                       Permission = "bb.releases.delete"
	ReleasesGet                          Permission = "bb.releases.get"
	ReleasesList                         Permission = "bb.releases.list"
	ReleasesUndelete                     Permission = "bb.releases.undelete"
	ReleasesUpdate                       Permission = "bb.releases.update"
	ReviewConfigsCreate                  Permission = "bb.reviewConfigs.create"
	ReviewConfigsDelete                  Permission = "bb.reviewConfigs.delete"
	ReviewConfigsGet                     Permission = "bb.reviewConfigs.get"
	ReviewConfigsList                    Permission = "bb.reviewConfigs.list"
	ReviewConfigsUpdate                  Permission = "bb.reviewConfigs.update"
	RevisionsCreate                      Permission = "bb.revisions.create"
	RevisionsDelete                      Permission = "bb.revisions.delete"
	RevisionsGet                         Permission = "bb.revisions.get"
	RevisionsList                        Permission = "bb.revisions.list"
	RolesCreate                          Permission = "bb.roles.create"
	RolesDelete                          Permission = "bb.roles.delete"
	RolesList                            Permission = "bb.roles.list"
	RolesGet                             Permission = "bb.roles.get"
	RolesUpdate                          Permission = "bb.roles.update"
	RolloutsCreate                       Permission = "bb.rollouts.create"
	RolloutsGet                          Permission = "bb.rollouts.get"
	RolloutsList                         Permission = "bb.rollouts.list"
	SettingsGet                          Permission = "bb.settings.get"
	SettingsList                         Permission = "bb.settings.list"
	SettingsSet                          Permission = "bb.settings.set"
	EnvironmentSettingsGet               Permission = "bb.settings.getEnvironment"
	EnvironmentSettingsSet               Permission = "bb.settings.setEnvironment"
	WorkspaceProfileSettingsGet          Permission = "bb.settings.getWorkspaceProfile"
	WorkspaceProfileSettingsSet          Permission = "bb.settings.setWorkspaceProfile"
	SheetsCreate                         Permission = "bb.sheets.create"
	SheetsGet                            Permission = "bb.sheets.get"
	SheetsUpdate                         Permission = "bb.sheets.update"
	SQLSelect                            Permission = "bb.sql.select"
	SQLDdl                               Permission = "bb.sql.ddl"
	SQLDml                               Permission = "bb.sql.dml"
	SQLExplain                           Permission = "bb.sql.explain"
	SQLInfo                              Permission = "bb.sql.info"
	SQLAdmin                             Permission = "bb.sql.admin"
	TaskRunsCreate                       Permission = "bb.taskRuns.create"
	TaskRunsList                         Permission = "bb.taskRuns.list"
	GroupsCreate                         Permission = "bb.groups.create"
	GroupsDelete                         Permission = "bb.groups.delete"
	GroupsGet                            Permission = "bb.groups.get"
	GroupsList                           Permission = "bb.groups.list"
	GroupsUpdate                         Permission = "bb.groups.update"
	UsersCreate                          Permission = "bb.users.create"
	UsersDelete                          Permission = "bb.users.delete"
	UsersGet                             Permission = "bb.users.get"
	UsersList                            Permission = "bb.users.list"
	UsersUndelete                        Permission = "bb.users.undelete"
	UsersUpdate                          Permission = "bb.users.update"
	UsersUpdateEmail                     Permission = "bb.users.updateEmail"
	WorksheetsGet                        Permission = "bb.worksheets.get"
	WorksheetsManage                     Permission = "bb.worksheets.manage"
	WorkspacesGetIamPolicy               Permission = "bb.workspaces.getIamPolicy"
	WorkspacesSetIamPolicy               Permission = "bb.workspaces.setIamPolicy"
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
