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

	PermissionIssuesList           Permission = "bb.issues.list"
	PermissionIssuesGet            Permission = "bb.issues.get"
	PermissionIssuesCreate         Permission = "bb.issues.create"
	PermissionIssuesUpdate         Permission = "bb.issues.update"
	PermissionIssuesApprove        Permission = "bb.issues.approve"
	PermissionIssuesReject         Permission = "bb.issues.reject"
	PermissionIssueRerequestReview Permission = "bb.issues.rerequestReview"
	PermissionIssueCommentsCreate  Permission = "bb.issueComments.create"
	PermissionIssueCommentsUpdate  Permission = "bb.issueComments.update"
)
