package iam

type Permission string

const (
	// Workspace-level permissions.

	// Instance service.
	PermissionInstancesList     Permission = "bb.instances.list"
	PermissionInstancesGet      Permission = "bb.instances.get"
	PermissionInstancesCreate   Permission = "bb.instances.create"
	PermissionInstancesUpdate   Permission = "bb.instances.update"
	PermissionInstancesDelete   Permission = "bb.instances.delete"
	PermissionInstancesUndelete Permission = "bb.instances.undelete"
	PermissionInstancesSync     Permission = "bb.instances.sync"

	// Instance role service.
	PermissionInstanceRolesList     Permission = "bb.instanceRoles.list"
	PermissionInstanceRolesGet      Permission = "bb.instanceRoles.get"
	PermissionInstanceRolesCreate   Permission = "bb.instanceRoles.create"
	PermissionInstanceRolesUpdate   Permission = "bb.instanceRoles.update"
	PermissionInstanceRolesDelete   Permission = "bb.instanceRoles.delete"
	PermissionInstanceRolesUndelete Permission = "bb.instanceRoles.undelete"

	// Environment service.
	PermissionEnvironmentsList     Permission = "bb.environments.list"
	PermissionEnvironmentsGet      Permission = "bb.environments.get"
	PermissionEnvironmentsCreate   Permission = "bb.environments.create"
	PermissionEnvironmentsUpdate   Permission = "bb.environments.update"
	PermissionEnvironmentsDelete   Permission = "bb.environments.delete"
	PermissionEnvironmentsUndelete Permission = "bb.environments.undelete"

	// Risk service.
	PermissionRisksList   Permission = "bb.risks.list"
	PermissionRisksCreate Permission = "bb.risks.create"
	PermissionRisksUpdate Permission = "bb.risks.update"
	PermissionRisksDelete Permission = "bb.risks.delete"

	// Role service.
	PermissionRolesList   Permission = "bb.roles.list"
	PermissionRolesCreate Permission = "bb.roles.create"
	PermissionRolesUpdate Permission = "bb.roles.update"
	PermissionRolesDelete Permission = "bb.roles.delete"

	// Database service.
	PermissionDatabasesList                Permission = "bb.databases.list"
	PermissionDatabasesGet                 Permission = "bb.databases.get"
	PermissionDatabasesUpdate              Permission = "bb.databases.update"
	PermissionDatabasesSync                Permission = "bb.databases.sync"
	PermissionDatabasesAdviseIndex         Permission = "bb.databases.adviseIndex"
	PermissionDatabasesGetMetadata         Permission = "bb.databases.getMetadata"
	PermissionDatabasesUpdateMetadata      Permission = "bb.databases.updateMetadata"
	PermissionDatabasesGetSchema           Permission = "bb.databases.getSchema"
	PermissionDatabasesGetBackupSetting    Permission = "bb.databases.getBackupSetting"
	PermissionDatabasesUpdateBackupSetting Permission = "bb.databases.updateBackupSetting"
	PermissionBackupsList                  Permission = "bb.backups.list"
	PermissionBackupsCreate                Permission = "bb.backups.create"
	PermissionChangeHistoriesList          Permission = "bb.changeHistories.list"
	PermissionChangeHistoriesGet           Permission = "bb.changeHistories.get"
	PermissionDatabaseSecretsList          Permission = "bb.databaseSecrets.list"
	PermissionDatabaseSecretsUpdate        Permission = "bb.databaseSecrets.update"
	PermissionDatabaseSecretsDelete        Permission = "bb.databaseSecrets.delete"
	PermissionSlowQueriesList              Permission = "bb.slowQueries.list"

	// External version control service.
	PermissionExternalVersionControlsGet            Permission = "bb.externalVersionControls.get"
	PermissionExternalVersionControlsList           Permission = "bb.externalVersionControls.list"
	PermissionExternalVersionControlsCreate         Permission = "bb.externalVersionControls.create"
	PermissionExternalVersionControlsUpdate         Permission = "bb.externalVersionControls.update"
	PermissionExternalVersionControlsDelete         Permission = "bb.externalVersionControls.delete"
	PermissionExternalVersionControlsSearchProjects Permission = "bb.externalVersionControls.searchProjects"
	PermissionExternalVersionControlsListProjects   Permission = "bb.externalVersionControls.listProjects"

	// Setting service.
	PermissionSettingsList Permission = "bb.settings.list"
	PermissionSettingsGet  Permission = "bb.settings.get"
	PermissionSettingsSet  Permission = "bb.settings.set"

	// Org policy service.
	PermissionPoliciesList   Permission = "bb.policies.list"
	PermissionPoliciesGet    Permission = "bb.policies.get"
	PermissionPoliciesCreate Permission = "bb.policies.create"
	PermissionPoliciesUpdate Permission = "bb.policies.update"
	PermissionPoliciesDelete Permission = "bb.policies.delete"

	// Identity provider service.
	PermissionIdentityProvidersGet      Permission = "bb.identityProviders.get"
	PermissionIdentityProvidersCreate   Permission = "bb.identityProviders.create"
	PermissionIdentityProvidersUpdate   Permission = "bb.identityProviders.update"
	PermissionIdentityProvidersDelete   Permission = "bb.identityProviders.delete"
	PermissionIdentityProvidersUndelete Permission = "bb.identityProviders.undelete"

	// Project-level permissions.

	// Project Service.
	PermissionProjectsList         Permission = "bb.projects.list"
	PermissionProjectsGet          Permission = "bb.projects.get"
	PermissionProjectsCreate       Permission = "bb.projects.create"
	PermissionProjectsUpdate       Permission = "bb.projects.update"
	PermissionProjectsDelete       Permission = "bb.projects.delete"
	PermissionProjectsUndelete     Permission = "bb.projects.undelete"
	PermissionProjectsGetIAMPolicy Permission = "bb.projects.getIamPolicy"
	PermissionProjectsSetIAMPolicy Permission = "bb.projects.setIamPolicy"

	// Issue service.
	PermissionIssuesList          Permission = "bb.issues.list"
	PermissionIssuesGet           Permission = "bb.issues.get"
	PermissionIssuesCreate        Permission = "bb.issues.create"
	PermissionIssuesUpdate        Permission = "bb.issues.update"
	PermissionIssueCommentsCreate Permission = "bb.issueComments.create"
	PermissionIssueCommentsUpdate Permission = "bb.issueComments.update"

	// Changelist service.
	PermissionChangelistsList   Permission = "bb.changelists.list"
	PermissionChangelistsGet    Permission = "bb.changelists.get"
	PermissionChangelistsUpdate Permission = "bb.changelists.update"
	PermissionChangelistsCreate Permission = "bb.changelists.create"
	PermissionChangelistsDelete Permission = "bb.changelists.delete"

	// Rollout service.
	PermissionPlansList         Permission = "bb.plans.list"
	PermissionPlansGet          Permission = "bb.plans.get"
	PermissionPlansCreate       Permission = "bb.plans.create"
	PermissionPlansUpdate       Permission = "bb.plans.update"
	PermissionRolloutsGet       Permission = "bb.rollouts.get"
	PermissionRolloutsCreate    Permission = "bb.rollouts.create"
	PermissionRolloutsPreview   Permission = "bb.rollouts.preview"
	PermissionTaskRunsList      Permission = "bb.taskRuns.list"
	PermissionPlanCheckRunsList Permission = "bb.planCheckRuns.list"
	PermissionPlanCheckRunsRun  Permission = "bb.planCheckRuns.run"
	PermissionTasksRun          Permission = "bb.tasks.run"
	PermissionTasksSkip         Permission = "bb.tasks.skip"
	PermissionTaskRunsCancel    Permission = "bb.taskRuns.cancel"

	// Branch service.
	PermissionBranchesList   Permission = "bb.branches.list"
	PermissionBranchesGet    Permission = "bb.branches.get"
	PermissionBranchesCreate Permission = "bb.branches.create"
	PermissionBranchesUpdate Permission = "bb.branches.update"
	PermissionBranchesDelete Permission = "bb.branches.delete"
)
