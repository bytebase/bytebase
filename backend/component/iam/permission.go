package iam

type Permission = string

const (
	PermissionAuditLogsExport            Permission = "bb.auditLogs.export"
	PermissionAuditLogsSearch            Permission = "bb.auditLogs.search"
	PermissionBranchesAdmin              Permission = "bb.branches.admin"
	PermissionBranchesCreate             Permission = "bb.branches.create"
	PermissionBranchesDelete             Permission = "bb.branches.delete"
	PermissionBranchesGet                Permission = "bb.branches.get"
	PermissionBranchesList               Permission = "bb.branches.list"
	PermissionBranchesUpdate             Permission = "bb.branches.update"
	PermissionChangeHistoriesGet         Permission = "bb.changeHistories.get"
	PermissionChangeHistoriesList        Permission = "bb.changeHistories.list"
	PermissionChangelistsCreate          Permission = "bb.changelists.create"
	PermissionChangelistsDelete          Permission = "bb.changelists.delete"
	PermissionChangelistsGet             Permission = "bb.changelists.get"
	PermissionChangelistsList            Permission = "bb.changelists.list"
	PermissionChangelistsUpdate          Permission = "bb.changelists.update"
	PermissionDatabaseCatalogsGet        Permission = "bb.databaseCatalogs.get"
	PermissionDatabaseCatalogsUpdate     Permission = "bb.databaseCatalogs.update"
	PermissionDatabaseSecretsDelete      Permission = "bb.databaseSecrets.delete"
	PermissionDatabaseSecretsList        Permission = "bb.databaseSecrets.list"
	PermissionDatabaseSecretsUpdate      Permission = "bb.databaseSecrets.update"
	PermissionDatabasesAdviseIndex       Permission = "bb.databases.adviseIndex"
	PermissionDatabasesCheck             Permission = "bb.databases.check"
	PermissionDatabasesGet               Permission = "bb.databases.get"
	PermissionDatabasesGetSchema         Permission = "bb.databases.getSchema"
	PermissionDatabasesList              Permission = "bb.databases.list"
	PermissionDatabasesSync              Permission = "bb.databases.sync"
	PermissionDatabasesUpdate            Permission = "bb.databases.update"
	PermissionEnvironmentsCreate         Permission = "bb.environments.create"
	PermissionEnvironmentsDelete         Permission = "bb.environments.delete"
	PermissionEnvironmentsGet            Permission = "bb.environments.get"
	PermissionEnvironmentsList           Permission = "bb.environments.list"
	PermissionEnvironmentsUndelete       Permission = "bb.environments.undelete"
	PermissionEnvironmentsUpdate         Permission = "bb.environments.update"
	PermissionIdentityProvidersCreate    Permission = "bb.identityProviders.create"
	PermissionIdentityProvidersDelete    Permission = "bb.identityProviders.delete"
	PermissionIdentityProvidersGet       Permission = "bb.identityProviders.get"
	PermissionIdentityProvidersUndelete  Permission = "bb.identityProviders.undelete"
	PermissionIdentityProvidersUpdate    Permission = "bb.identityProviders.update"
	PermissionInstancesCreate            Permission = "bb.instances.create"
	PermissionInstancesDelete            Permission = "bb.instances.delete"
	PermissionInstancesGet               Permission = "bb.instances.get"
	PermissionInstancesList              Permission = "bb.instances.list"
	PermissionInstancesSync              Permission = "bb.instances.sync"
	PermissionInstancesUndelete          Permission = "bb.instances.undelete"
	PermissionInstancesUpdate            Permission = "bb.instances.update"
	PermissionInstanceRolesGet           Permission = "bb.instanceRoles.get"
	PermissionInstanceRolesList          Permission = "bb.instanceRoles.list"
	PermissionIssueCommentsCreate        Permission = "bb.issueComments.create"
	PermissionIssueCommentsList          Permission = "bb.issueComments.list"
	PermissionIssueCommentsUpdate        Permission = "bb.issueComments.update"
	PermissionIssuesCreate               Permission = "bb.issues.create"
	PermissionIssuesGet                  Permission = "bb.issues.get"
	PermissionIssuesList                 Permission = "bb.issues.list"
	PermissionIssuesUpdate               Permission = "bb.issues.update"
	PermissionPlanCheckRunsList          Permission = "bb.planCheckRuns.list"
	PermissionPlanCheckRunsRun           Permission = "bb.planCheckRuns.run"
	PermissionPlansCreate                Permission = "bb.plans.create"
	PermissionPlansGet                   Permission = "bb.plans.get"
	PermissionPlansList                  Permission = "bb.plans.list"
	PermissionPlansUpdate                Permission = "bb.plans.update"
	PermissionPoliciesCreate             Permission = "bb.policies.create"
	PermissionPoliciesDelete             Permission = "bb.policies.delete"
	PermissionPoliciesGet                Permission = "bb.policies.get"
	PermissionPoliciesList               Permission = "bb.policies.list"
	PermissionPoliciesUpdate             Permission = "bb.policies.update"
	PermissionProjectsCreate             Permission = "bb.projects.create"
	PermissionProjectsDelete             Permission = "bb.projects.delete"
	PermissionProjectsGet                Permission = "bb.projects.get"
	PermissionProjectsGetIAMPolicy       Permission = "bb.projects.getIamPolicy"
	PermissionProjectsList               Permission = "bb.projects.list"
	PermissionProjectsSetIAMPolicy       Permission = "bb.projects.setIamPolicy"
	PermissionProjectsUndelete           Permission = "bb.projects.undelete"
	PermissionProjectsUpdate             Permission = "bb.projects.update"
	PermissionReviewConfigsCreate        Permission = "bb.reviewConfigs.create"
	PermissionReviewConfigsDelete        Permission = "bb.reviewConfigs.delete"
	PermissionReviewConfigsGet           Permission = "bb.reviewConfigs.get"
	PermissionReviewConfigsList          Permission = "bb.reviewConfigs.list"
	PermissionReviewConfigsUpdate        Permission = "bb.reviewConfigs.update"
	PermissionRisksCreate                Permission = "bb.risks.create"
	PermissionRisksDelete                Permission = "bb.risks.delete"
	PermissionRisksList                  Permission = "bb.risks.list"
	PermissionRisksUpdate                Permission = "bb.risks.update"
	PermissionRolesCreate                Permission = "bb.roles.create"
	PermissionRolesDelete                Permission = "bb.roles.delete"
	PermissionRolesList                  Permission = "bb.roles.list"
	PermissionRolesGet                   Permission = "bb.roles.get"
	PermissionRolesUpdate                Permission = "bb.roles.update"
	PermissionRolloutsCreate             Permission = "bb.rollouts.create"
	PermissionRolloutsGet                Permission = "bb.rollouts.get"
	PermissionRolloutsList               Permission = "bb.rollouts.list"
	PermissionRolloutsPreview            Permission = "bb.rollouts.preview"
	PermissionSettingsGet                Permission = "bb.settings.get"
	PermissionSettingsList               Permission = "bb.settings.list"
	PermissionSettingsSet                Permission = "bb.settings.set"
	PermissionSheetsCreate               Permission = "bb.sheets.create"
	PermissionSheetsGet                  Permission = "bb.sheets.get"
	PermissionSheetsUpdate               Permission = "bb.sheets.update"
	PermissionSlowQueriesList            Permission = "bb.slowQueries.list"
	PermissionSQLSelect                  Permission = "bb.sql.select"
	PermissionSQLDdl                     Permission = "bb.sql.ddl"
	PermissionSQLDml                     Permission = "bb.sql.dml"
	PermissionSQLExplain                 Permission = "bb.sql.explain"
	PermissionSQLInfo                    Permission = "bb.sql.info"
	PermissionSQLExport                  Permission = "bb.sql.export"
	PermissionSQLAdmin                   Permission = "bb.sql.admin"
	PermissionTaskRunsCreate             Permission = "bb.taskRuns.create"
	PermissionTaskRunsList               Permission = "bb.taskRuns.list"
	PermissionGroupsCreate               Permission = "bb.groups.create"
	PermissionGroupsDelete               Permission = "bb.groups.delete"
	PermissionGroupsGet                  Permission = "bb.groups.get"
	PermissionGroupsList                 Permission = "bb.groups.list"
	PermissionGroupsUpdate               Permission = "bb.groups.update"
	PermissionUsersCreate                Permission = "bb.users.create"
	PermissionUsersDelete                Permission = "bb.users.delete"
	PermissionUsersUndelete              Permission = "bb.users.undelete"
	PermissionUsersUpdate                Permission = "bb.users.update"
	PermissionVCSConnectorsCreate        Permission = "bb.vcsConnectors.create"
	PermissionVCSConnectorsDelete        Permission = "bb.vcsConnectors.delete"
	PermissionVCSConnectorsGet           Permission = "bb.vcsConnectors.get"
	PermissionVCSConnectorsList          Permission = "bb.vcsConnectors.list"
	PermissionVCSConnectorsUpdate        Permission = "bb.vcsConnectors.update"
	PermissionVCSProvidersCreate         Permission = "bb.vcsProviders.create"
	PermissionVCSProvidersDelete         Permission = "bb.vcsProviders.delete"
	PermissionVCSProvidersGet            Permission = "bb.vcsProviders.get"
	PermissionVCSProvidersList           Permission = "bb.vcsProviders.list"
	PermissionVCSProvidersListProjects   Permission = "bb.vcsProviders.listProjects"
	PermissionVCSProvidersSearchProjects Permission = "bb.vcsProviders.searchProjects"
	PermissionVCSProvidersUpdate         Permission = "bb.vcsProviders.update"
	PermissionWorksheetsGet              Permission = "bb.worksheets.get"
	PermissionWorksheetsManage           Permission = "bb.worksheets.manage"
)

var allPermissions = []Permission{
	PermissionAuditLogsExport,
	PermissionAuditLogsSearch,
	PermissionBranchesAdmin,
	PermissionBranchesCreate,
	PermissionBranchesDelete,
	PermissionBranchesGet,
	PermissionBranchesList,
	PermissionBranchesUpdate,
	PermissionChangeHistoriesGet,
	PermissionChangeHistoriesList,
	PermissionChangelistsCreate,
	PermissionChangelistsDelete,
	PermissionChangelistsGet,
	PermissionChangelistsList,
	PermissionChangelistsUpdate,
	PermissionDatabaseCatalogsGet,
	PermissionDatabaseCatalogsUpdate,
	PermissionDatabaseSecretsDelete,
	PermissionDatabaseSecretsList,
	PermissionDatabaseSecretsUpdate,
	PermissionDatabasesAdviseIndex,
	PermissionDatabasesCheck,
	PermissionDatabasesGet,
	PermissionDatabasesGetSchema,
	PermissionDatabasesList,
	PermissionDatabasesSync,
	PermissionDatabasesUpdate,
	PermissionEnvironmentsCreate,
	PermissionEnvironmentsDelete,
	PermissionEnvironmentsGet,
	PermissionEnvironmentsList,
	PermissionEnvironmentsUndelete,
	PermissionEnvironmentsUpdate,
	PermissionIdentityProvidersCreate,
	PermissionIdentityProvidersDelete,
	PermissionIdentityProvidersGet,
	PermissionIdentityProvidersUndelete,
	PermissionIdentityProvidersUpdate,
	PermissionInstancesCreate,
	PermissionInstancesDelete,
	PermissionInstancesGet,
	PermissionInstancesList,
	PermissionInstancesSync,
	PermissionInstancesUndelete,
	PermissionInstancesUpdate,
	PermissionInstanceRolesGet,
	PermissionInstanceRolesList,
	PermissionIssueCommentsCreate,
	PermissionIssueCommentsList,
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
	PermissionPoliciesCreate,
	PermissionPoliciesDelete,
	PermissionPoliciesGet,
	PermissionPoliciesList,
	PermissionPoliciesUpdate,
	PermissionProjectsCreate,
	PermissionProjectsDelete,
	PermissionProjectsGet,
	PermissionProjectsGetIAMPolicy,
	PermissionProjectsList,
	PermissionProjectsSetIAMPolicy,
	PermissionProjectsUndelete,
	PermissionProjectsUpdate,
	PermissionReviewConfigsCreate,
	PermissionReviewConfigsDelete,
	PermissionReviewConfigsGet,
	PermissionReviewConfigsList,
	PermissionReviewConfigsUpdate,
	PermissionRisksCreate,
	PermissionRisksDelete,
	PermissionRisksList,
	PermissionRisksUpdate,
	PermissionRolesCreate,
	PermissionRolesDelete,
	PermissionRolesList,
	PermissionRolesGet,
	PermissionRolesUpdate,
	PermissionRolloutsCreate,
	PermissionRolloutsGet,
	PermissionRolloutsList,
	PermissionRolloutsPreview,
	PermissionSettingsGet,
	PermissionSettingsList,
	PermissionSettingsSet,
	PermissionSheetsCreate,
	PermissionSheetsGet,
	PermissionSheetsUpdate,
	PermissionSlowQueriesList,
	PermissionSQLSelect,
	PermissionSQLDdl,
	PermissionSQLDml,
	PermissionSQLExplain,
	PermissionSQLInfo,
	PermissionSQLExport,
	PermissionSQLAdmin,
	PermissionTaskRunsCreate,
	PermissionTaskRunsList,
	PermissionGroupsCreate,
	PermissionGroupsDelete,
	PermissionGroupsGet,
	PermissionGroupsList,
	PermissionGroupsUpdate,
	PermissionUsersCreate,
	PermissionUsersDelete,
	PermissionUsersUndelete,
	PermissionUsersUpdate,
	PermissionVCSConnectorsCreate,
	PermissionVCSConnectorsDelete,
	PermissionVCSConnectorsGet,
	PermissionVCSConnectorsList,
	PermissionVCSConnectorsUpdate,
	PermissionVCSProvidersCreate,
	PermissionVCSProvidersDelete,
	PermissionVCSProvidersGet,
	PermissionVCSProvidersList,
	PermissionVCSProvidersListProjects,
	PermissionVCSProvidersSearchProjects,
	PermissionVCSProvidersUpdate,
	PermissionWorksheetsGet,
	PermissionWorksheetsManage,
}

var allPermissionsMap = func() map[Permission]bool {
	m := make(map[Permission]bool)
	for _, p := range allPermissions {
		m[p] = true
	}
	return m
}()

func PermissionsExist(permissions ...string) bool {
	for _, p := range permissions {
		if !allPermissionsMap[p] {
			return false
		}
	}
	return true
}
