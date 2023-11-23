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

	PermissionDatabaseList                Permission = "bb.database.list"
	PermissionDatabaseGet                 Permission = "bb.database.get"
	PermissionDatabaseUpdate              Permission = "bb.database.update"
	PermissionDatabaseSync                Permission = "bb.database.sync"
	PermissionDatabaseGetMetadata         Permission = "bb.database.getMetadata"
	PermissionDatabaseUpdateMetadata      Permission = "bb.database.updateMetadata"
	PermissionDatabaseGetSchema           Permission = "bb.database.getSchema"
	PermissionDatabaseGetBackupSetting    Permission = "bb.database.getBackupSetting"
	PermissionDatabaseUpdateBackupSetting Permission = "bb.database.updateBackupSetting"
	PermissionBackupsList                 Permission = "bb.backups.list"
	PermissionBackupsCreate               Permission = "bb.backups.create"
	PermissionChangeHistoriesList         Permission = "bb.changeHistories.list"
	PermissionChangeHistoriesGet          Permission = "bb.changeHistories.get"
	PermissionDatabaseSecretsList         Permission = "bb.databaseSecrets.list"
	PermissionDatabaseSecretsUpdate       Permission = "bb.databaseSecrets.update"
	PermissionDatabaseSecretsDelete       Permission = "bb.databaseSecrets.delete"
)
