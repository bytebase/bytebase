package iam

type Permission string

const (
	PermissionInstancesList     Permission = "bb.instances.list"
	PermissionInstancesGet      Permission = "bb.instances.get"
	PermissionInstancesCreate   Permission = "bb.instances.create"
	PermissionInstancesUpdate   Permission = "bb.instances.update"
	PermissionInstancesDelete   Permission = "bb.instances.delete"
	PermissionInstancesUndelete Permission = "bb.instances.undelete"
	PermissionInstancesSync     Permission = "bb.instances.sync"

	PermissionDatabasesList                Permission = "bb.databases.list"
	PermissionDatabasesGet                 Permission = "bb.databases.get"
	PermissionDatabasesUpdate              Permission = "bb.databases.update"
	PermissionDatabasesSync                Permission = "bb.databases.sync"
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

	PermissionEnvironmentsList     Permission = "bb.environments.list"
	PermissionEnvironmentsGet      Permission = "bb.environments.get"
	PermissionEnvironmentsCreate   Permission = "bb.environments.create"
	PermissionEnvironmentsUpdate   Permission = "bb.environments.update"
	PermissionEnvironmentsDelete   Permission = "bb.environments.delete"
	PermissionEnvironmentsUndelete Permission = "bb.environments.undelete"

	PermissionIssuesList          Permission = "bb.issues.list"
	PermissionIssuesGet           Permission = "bb.issues.get"
	PermissionIssuesCreate        Permission = "bb.issues.create"
	PermissionIssuesUpdate        Permission = "bb.issues.update"
	PermissionIssueCommentsCreate Permission = "bb.issueComments.create"
	PermissionIssueCommentsUpdate Permission = "bb.issueComments.update"

	// Project Service.
	PermissionProjectsList         Permission = "bb.projects.list"
	PermissionProjectsGet          Permission = "bb.projects.get"
	PermissionProjectsCreate       Permission = "bb.projects.create"
	PermissionProjectsUpdate       Permission = "bb.projects.update"
	PermissionProjectsDelete       Permission = "bb.projects.delete"
	PermissionProjectsUndelete     Permission = "bb.projects.undelete"
	PermissionProjectsGetIAMPolicy Permission = "bb.projects.getIamPolicy"
	PermissionProjectsSetIAMPolicy Permission = "bb.projects.setIamPolicy"

	PermissionRisksList   Permission = "bb.risks.list"
	PermissionRisksCreate Permission = "bb.risks.create"
	PermissionRisksUpdate Permission = "bb.risks.update"
	PermissionRisksDelete Permission = "bb.risks.delete"

	PermissionRolesList   Permission = "bb.roles.list"
	PermissionRolesCreate Permission = "bb.roles.create"
	PermissionRolesUpdate Permission = "bb.roles.update"
	PermissionRolesDelete Permission = "bb.roles.delete"

	PermissionChangelistsList   Permission = "bb.changelists.list"
	PermissionChangelistsGet    Permission = "bb.changelists.get"
	PermissionChangelistsUpdate Permission = "bb.changelists.update"
	PermissionChangelistsCreate Permission = "bb.changelists.create"
	PermissionChangelistsDelete Permission = "bb.changelists.delete"
)
