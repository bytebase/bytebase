package iam

type Permission string

const (
	PermissionInstanceList     Permission = "bb.instance.list"
	PermissionInstanceGet      Permission = "bb.instance.get"
	PermissionInstanceCreate   Permission = "bb.instance.create"
	PermissionInstanceUpdate   Permission = "bb.instance.update"
	PermissionInstanceDelete   Permission = "bb.instance.delete"
	PermissionInstanceUndelete Permission = "bb.instance.undelete"
	PermissionInstanceSync     Permission = "bb.instance.sync"

	PermissionDatabasesList               Permission = "bb.databases.list"
	PermissionDatabasesGet                Permission = "bb.databases.get"
	PermissionDatabasesUpdate             Permission = "bb.databases.update"
	PermissionDatabasesSync               Permission = "bb.databases.sync"
	PermissionDatabasesGetMetadata        Permission = "bb.databases.getMetadata"
	PermissionDatabasesUpdateMetadata     Permission = "bb.databases.updateMetadata"
	PermissionDatabasesGetSchema          Permission = "bb.databases.getSchema"
	PermissionDatabaseGetBackupSetting    Permission = "bb.databases.getBackupSetting"
	PermissionDatabaseUpdateBackupSetting Permission = "bb.databases.updateBackupSetting"
	PermissionBackupsList                 Permission = "bb.backups.list"
	PermissionBackupsCreate               Permission = "bb.backups.create"
	PermissionChangeHistoriesList         Permission = "bb.changeHistories.list"
	PermissionChangeHistoriesGet          Permission = "bb.changeHistories.get"
	PermissionDatabaseSecretsList         Permission = "bb.databaseSecrets.list"
	PermissionDatabaseSecretsUpdate       Permission = "bb.databaseSecrets.update"
	PermissionDatabaseSecretsDelete       Permission = "bb.databaseSecrets.delete"
)
