package iam

type Permission = string

const (
	// Workspace-level permissions.
	PermissionEnvironmentsCreate         Permission = "bb.environments.create"
	PermissionEnvironmentsDelete         Permission = "bb.environments.delete"
	PermissionEnvironmentsGet            Permission = "bb.environments.get"
	PermissionEnvironmentsList           Permission = "bb.environments.list"
	PermissionEnvironmentsUndelete       Permission = "bb.environments.undelete"
	PermissionEnvironmentsUpdate         Permission = "bb.environments.update"
	PermissionVCSProvidersCreate         Permission = "bb.vcsProviders.create"
	PermissionVCSProvidersDelete         Permission = "bb.vcsProviders.delete"
	PermissionVCSProvidersGet            Permission = "bb.vcsProviders.get"
	PermissionVCSProvidersList           Permission = "bb.vcsProviders.list"
	PermissionVCSProvidersListProjects   Permission = "bb.vcsProviders.listProjects"
	PermissionVCSProvidersSearchProjects Permission = "bb.vcsProviders.searchProjects"
	PermissionVCSProvidersUpdate         Permission = "bb.vcsProviders.update"
	PermissionReviewConfigsCreate        Permission = "bb.reviewConfigs.create"
	PermissionReviewConfigsDelete        Permission = "bb.reviewConfigs.delete"
	PermissionReviewConfigsGet           Permission = "bb.reviewConfigs.get"
	PermissionReviewConfigsList          Permission = "bb.reviewConfigs.list"
	PermissionReviewConfigsUpdate        Permission = "bb.reviewConfigs.update"
	PermissionIdentityProvidersCreate    Permission = "bb.identityProviders.create"
	PermissionIdentityProvidersDelete    Permission = "bb.identityProviders.delete"
	PermissionIdentityProvidersGet       Permission = "bb.identityProviders.get"
	PermissionIdentityProvidersUndelete  Permission = "bb.identityProviders.undelete"
	PermissionIdentityProvidersUpdate    Permission = "bb.identityProviders.update"
	PermissionInstancesAdminExecute      Permission = "bb.instances.adminExecute"
	PermissionInstancesCreate            Permission = "bb.instances.create"
	PermissionInstancesDelete            Permission = "bb.instances.delete"
	PermissionInstancesGet               Permission = "bb.instances.get"
	PermissionInstancesList              Permission = "bb.instances.list"
	PermissionInstancesSync              Permission = "bb.instances.sync"
	PermissionInstancesUndelete          Permission = "bb.instances.undelete"
	PermissionInstancesUpdate            Permission = "bb.instances.update"
	PermissionPoliciesCreate             Permission = "bb.policies.create"
	PermissionPoliciesDelete             Permission = "bb.policies.delete"
	PermissionPoliciesGet                Permission = "bb.policies.get"
	PermissionPoliciesList               Permission = "bb.policies.list"
	PermissionPoliciesUpdate             Permission = "bb.policies.update"
	PermissionProjectsCreate             Permission = "bb.projects.create"
	PermissionProjectsDelete             Permission = "bb.projects.delete"
	PermissionProjectsList               Permission = "bb.projects.list"
	PermissionProjectsUndelete           Permission = "bb.projects.undelete"
	PermissionRisksCreate                Permission = "bb.risks.create"
	PermissionRisksDelete                Permission = "bb.risks.delete"
	PermissionRisksList                  Permission = "bb.risks.list"
	PermissionRisksUpdate                Permission = "bb.risks.update"
	PermissionRolesCreate                Permission = "bb.roles.create"
	PermissionRolesDelete                Permission = "bb.roles.delete"
	PermissionRolesList                  Permission = "bb.roles.list"
	PermissionRolesUpdate                Permission = "bb.roles.update"
	PermissionUserGroupsCreate           Permission = "bb.userGroups.create"
	PermissionUserGroupsDelete           Permission = "bb.userGroups.delete"
	PermissionUserGroupsGet              Permission = "bb.userGroups.get"
	PermissionUserGroupsList             Permission = "bb.userGroups.list"
	PermissionUserGroupsUpdate           Permission = "bb.userGroups.update"
	PermissionSettingsGet                Permission = "bb.settings.get"
	PermissionSettingsList               Permission = "bb.settings.list"
	PermissionSettingsSet                Permission = "bb.settings.set"

	// Project-level permissions.
	PermissionAuditLogsGet          Permission = "bb.auditLogs.get"
	PermissionBranchesCreate        Permission = "bb.branches.create"
	PermissionBranchesDelete        Permission = "bb.branches.delete"
	PermissionBranchesGet           Permission = "bb.branches.get"
	PermissionBranchesList          Permission = "bb.branches.list"
	PermissionBranchesUpdate        Permission = "bb.branches.update"
	PermissionChangeHistoriesGet    Permission = "bb.changeHistories.get"
	PermissionChangeHistoriesList   Permission = "bb.changeHistories.list"
	PermissionVCSConnectorsCreate   Permission = "bb.vcsConnectors.create"
	PermissionVCSConnectorsDelete   Permission = "bb.vcsConnectors.delete"
	PermissionVCSConnectorsGet      Permission = "bb.vcsConnectors.get"
	PermissionVCSConnectorsList     Permission = "bb.vcsConnectors.list"
	PermissionVCSConnectorsUpdate   Permission = "bb.vcsConnectors.update"
	PermissionChangelistsCreate     Permission = "bb.changelists.create"
	PermissionChangelistsDelete     Permission = "bb.changelists.delete"
	PermissionChangelistsGet        Permission = "bb.changelists.get"
	PermissionChangelistsList       Permission = "bb.changelists.list"
	PermissionChangelistsUpdate     Permission = "bb.changelists.update"
	PermissionDatabaseSecretsDelete Permission = "bb.databaseSecrets.delete"
	PermissionDatabaseSecretsList   Permission = "bb.databaseSecrets.list"
	PermissionDatabaseSecretsUpdate Permission = "bb.databaseSecrets.update"
	PermissionDatabasesAdviseIndex  Permission = "bb.databases.adviseIndex"
	PermissionDatabasesExport       Permission = "bb.databases.export"
	PermissionDatabasesGet          Permission = "bb.databases.get"
	PermissionDatabasesGetSchema    Permission = "bb.databases.getSchema"
	PermissionDatabasesList         Permission = "bb.databases.list"
	PermissionDatabasesQuery        Permission = "bb.databases.query"
	PermissionDatabasesExecute      Permission = "bb.databases.execute"
	PermissionDatabasesSync         Permission = "bb.databases.sync"
	PermissionDatabasesUpdate       Permission = "bb.databases.update"
	PermissionIssueCommentsCreate   Permission = "bb.issueComments.create"
	PermissionIssueCommentsUpdate   Permission = "bb.issueComments.update"
	PermissionIssuesCreate          Permission = "bb.issues.create"
	PermissionIssuesGet             Permission = "bb.issues.get"
	PermissionIssuesList            Permission = "bb.issues.list"
	PermissionIssuesUpdate          Permission = "bb.issues.update"
	PermissionPlanCheckRunsList     Permission = "bb.planCheckRuns.list"
	PermissionPlanCheckRunsRun      Permission = "bb.planCheckRuns.run"
	PermissionPlansCreate           Permission = "bb.plans.create"
	PermissionPlansGet              Permission = "bb.plans.get"
	PermissionPlansList             Permission = "bb.plans.list"
	PermissionPlansUpdate           Permission = "bb.plans.update"
	PermissionSheetsCreate          Permission = "bb.sheets.create"
	PermissionSheetsUpdate          Permission = "bb.sheets.update"
	PermissionSheetsGet             Permission = "bb.sheets.get"
	PermissionProjectsGet           Permission = "bb.projects.get"
	PermissionProjectsGetIAMPolicy  Permission = "bb.projects.getIamPolicy"
	PermissionProjectsSetIAMPolicy  Permission = "bb.projects.setIamPolicy"
	PermissionProjectsUpdate        Permission = "bb.projects.update"
	PermissionRolloutsCreate        Permission = "bb.rollouts.create"
	PermissionRolloutsGet           Permission = "bb.rollouts.get"
	PermissionRolloutsPreview       Permission = "bb.rollouts.preview"
	PermissionSlowQueriesList       Permission = "bb.slowQueries.list"
	PermissionTaskRunsList          Permission = "bb.taskRuns.list"
)

var allWorkspacePermissions = []Permission{
	PermissionEnvironmentsCreate,
	PermissionEnvironmentsDelete,
	PermissionEnvironmentsGet,
	PermissionEnvironmentsList,
	PermissionEnvironmentsUndelete,
	PermissionEnvironmentsUpdate,
	PermissionVCSProvidersCreate,
	PermissionVCSProvidersDelete,
	PermissionVCSProvidersGet,
	PermissionVCSProvidersList,
	PermissionVCSProvidersListProjects,
	PermissionVCSProvidersSearchProjects,
	PermissionVCSProvidersUpdate,
	PermissionReviewConfigsCreate,
	PermissionReviewConfigsDelete,
	PermissionReviewConfigsGet,
	PermissionReviewConfigsList,
	PermissionReviewConfigsUpdate,
	PermissionIdentityProvidersCreate,
	PermissionIdentityProvidersDelete,
	PermissionIdentityProvidersGet,
	PermissionIdentityProvidersUndelete,
	PermissionIdentityProvidersUpdate,
	PermissionInstancesAdminExecute,
	PermissionInstancesCreate,
	PermissionInstancesDelete,
	PermissionInstancesGet,
	PermissionInstancesList,
	PermissionInstancesSync,
	PermissionInstancesUndelete,
	PermissionInstancesUpdate,
	PermissionPoliciesCreate,
	PermissionPoliciesDelete,
	PermissionPoliciesGet,
	PermissionPoliciesList,
	PermissionPoliciesUpdate,
	PermissionProjectsCreate,
	PermissionProjectsDelete,
	PermissionProjectsList,
	PermissionProjectsUndelete,
	PermissionRisksCreate,
	PermissionRisksDelete,
	PermissionRisksList,
	PermissionRisksUpdate,
	PermissionRolesCreate,
	PermissionRolesDelete,
	PermissionRolesList,
	PermissionRolesUpdate,
	PermissionUserGroupsCreate,
	PermissionUserGroupsDelete,
	PermissionUserGroupsGet,
	PermissionUserGroupsList,
	PermissionUserGroupsUpdate,
	PermissionSettingsGet,
	PermissionSettingsList,
	PermissionSettingsSet,
}
var allProjectPermissions = []Permission{
	PermissionAuditLogsGet,
	PermissionBranchesCreate,
	PermissionBranchesDelete,
	PermissionBranchesGet,
	PermissionBranchesList,
	PermissionBranchesUpdate,
	PermissionChangeHistoriesGet,
	PermissionChangeHistoriesList,
	PermissionVCSConnectorsCreate,
	PermissionVCSConnectorsDelete,
	PermissionVCSConnectorsGet,
	PermissionVCSConnectorsList,
	PermissionVCSConnectorsUpdate,
	PermissionChangelistsCreate,
	PermissionChangelistsDelete,
	PermissionChangelistsGet,
	PermissionChangelistsList,
	PermissionChangelistsUpdate,
	PermissionDatabaseSecretsDelete,
	PermissionDatabaseSecretsList,
	PermissionDatabaseSecretsUpdate,
	PermissionDatabasesAdviseIndex,
	PermissionDatabasesExport,
	PermissionDatabasesGet,
	PermissionDatabasesGetSchema,
	PermissionDatabasesList,
	PermissionDatabasesQuery,
	PermissionDatabasesExecute,
	PermissionDatabasesSync,
	PermissionDatabasesUpdate,
	PermissionIssueCommentsCreate,
	PermissionIssueCommentsUpdate,
	PermissionIssuesCreate,
	PermissionIssuesGet,
	PermissionIssuesList,
	PermissionIssuesUpdate,
	PermissionPlanCheckRunsList,
	PermissionPlanCheckRunsRun,
	PermissionPlansCreate,
	PermissionPlansGet,
	PermissionPlansList,
	PermissionPlansUpdate,
	PermissionSheetsCreate,
	PermissionSheetsUpdate,
	PermissionSheetsGet,
	PermissionProjectsGet,
	PermissionProjectsGetIAMPolicy,
	PermissionProjectsSetIAMPolicy,
	PermissionProjectsUpdate,
	PermissionRolloutsCreate,
	PermissionRolloutsGet,
	PermissionRolloutsPreview,
	PermissionSlowQueriesList,
	PermissionTaskRunsList,
}

var allWorkspacePermissionsMap = func() map[Permission]bool {
	m := make(map[Permission]bool)
	for _, p := range allWorkspacePermissions {
		m[p] = true
	}
	return m
}()

var allProjectPermissionsMap = func() map[Permission]bool {
	m := make(map[Permission]bool)
	for _, p := range allProjectPermissions {
		m[p] = true
	}
	return m
}()

func PermissionExist(p Permission) bool {
	if allWorkspacePermissionsMap[p] {
		return true
	}
	if allProjectPermissionsMap[p] {
		return true
	}
	return false
}

type PermissionLevel string

const (
	PermissionLevelWorkspace PermissionLevel = "WORKSPACE"
	PermissionLevelProject   PermissionLevel = "PROJECT"
)

func (p PermissionLevel) String() string {
	return string(p)
}

func GetPermissionLevel(permission Permission) PermissionLevel {
	if allWorkspacePermissionsMap[permission] {
		return PermissionLevelWorkspace
	}
	if allProjectPermissionsMap[permission] {
		return PermissionLevelProject
	}
	return ""
}
