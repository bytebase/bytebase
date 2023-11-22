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
	PermissionDatabaseListBackups         Permission = "bb.database.listBackups"
	PermissionDatabaseCreateBackup        Permission = "bb.database.createBackup"
	PermissionDatabaseListChangeHistories Permission = "bb.database.listChangeHistories"
	PermissionDatabaseGetChangeHistory    Permission = "bb.database.getChangeHistory"
	PermissionDatabaseListSecrets         Permission = "bb.database.listSecrets"
	PermissionDatabaseUpdateSecret        Permission = "bb.database.updateSecret"
	PermissionDatabaseDeleteSecret        Permission = "bb.database.deleteSecret"
)
