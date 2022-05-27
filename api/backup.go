package api

import (
	"encoding/json"
)

// BackupStatus is the status of a backup.
type BackupStatus string

const (
	// BackupStatusPendingCreate is the status for PENDING_CREATE.
	BackupStatusPendingCreate BackupStatus = "PENDING_CREATE"
	// BackupStatusDone is the status for DONE.
	BackupStatusDone BackupStatus = "DONE"
	// BackupStatusFailed is the status for FAILED.
	BackupStatusFailed BackupStatus = "FAILED"
)

func (e BackupStatus) String() string {
	switch e {
	case BackupStatusPendingCreate:
		return "PENDING_CREATE"
	case BackupStatusDone:
		return "DONE"
	case BackupStatusFailed:
		return "FAILED"
	}
	return "UNKNOWN"
}

// BackupType is the type of a backup.
type BackupType string

const (
	// BackupTypeAutomatic is the type for automatic backup.
	BackupTypeAutomatic BackupType = "AUTOMATIC"
	// BackupTypeManual is the type for manual backup.
	BackupTypeManual BackupType = "MANUAL"
)

func (e BackupType) String() string {
	switch e {
	case BackupTypeAutomatic:
		return "AUTOMATIC"
	case BackupTypeManual:
		return "MANUAL"
	}
	return "UNKNOWN"
}

// BackupStorageBackend is the storage backend of a backup.
type BackupStorageBackend string

const (
	// BackupStorageBackendLocal is the local storage backend for a backup.
	BackupStorageBackendLocal BackupStorageBackend = "LOCAL"
	// BackupStorageBackendS3 is the AWS S3 storage backend for a backup. Not used yet.
	BackupStorageBackendS3 BackupStorageBackend = "S3"
	// BackupStorageBackendGCS is the Google Cloud Storage (GCS) storage backend for a backup. Not used yet.
	BackupStorageBackendGCS BackupStorageBackend = "GCS"
	// BackupStorageBackendOSS is the AliCloud Object Storage Service (OSS) storage backend for a backup. Not used yet.
	BackupStorageBackendOSS BackupStorageBackend = "OSS"
)

func (e BackupStorageBackend) String() string {
	switch e {
	case BackupStorageBackendLocal:
		return "LOCAL"
	case BackupStorageBackendS3:
		return "S3"
	case BackupStorageBackendGCS:
		return "GCS"
	case BackupStorageBackendOSS:
		return "OSS"
	}
	return "UNKNOWN"
}

// BinlogInfo is the binlog coordination for MySQL.
type BinlogInfo struct {
	FileName string `json:"fileName"`
	Position int64  `json:"position"`
}

// BackupPayload contains backup related database specific info, it differs for different database types.
// It is encoded in JSON and stored in the backup table.
type BackupPayload struct {
	// MySQL related fields
	// BinlogInfo is recorded when taking the backup.
	// It is recorded when we dump the MySQL database with sophisticated timing.
	// Please refer to https://github.com/bytebase/bytebase/blob/main/docs/design/pitr-mysql.md#full-backup for details.
	BinlogInfo BinlogInfo `json:"binlogInfo"`
}

// Backup is the API message for a backup.
type Backup struct {
	ID int `jsonapi:"primary,backup"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	DatabaseID int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name           string               `jsonapi:"attr,name"`
	Status         BackupStatus         `jsonapi:"attr,status"`
	Type           BackupType           `jsonapi:"attr,type"`
	StorageBackend BackupStorageBackend `jsonapi:"attr,storageBackend"`
	// Upon taking the database backup, we will also record the current migration history version if exists.
	// And when restoring the backup, we will record this in the migration history.
	MigrationHistoryVersion string `jsonapi:"attr,migrationHistoryVersion"`
	Path                    string `jsonapi:"attr,path"`
	Comment                 string `jsonapi:"attr,comment"`
	// Payload contains data such as binlog position info which will not be created at first.
	// It is filled when the backup task executor takes database backups.
	Payload BackupPayload `jsonapi:"attr,payload"`
}

// BackupCreate is the API message for creating a backup.
type BackupCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	DatabaseID int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name                    string               `jsonapi:"attr,name"`
	Type                    BackupType           `jsonapi:"attr,type"`
	StorageBackend          BackupStorageBackend `jsonapi:"attr,storageBackend"`
	MigrationHistoryVersion string
	Path                    string
}

// BackupFind is the API message for finding backups.
type BackupFind struct {
	ID *int

	// Related fields
	DatabaseID *int

	// Domain specific fields
	Name   *string
	Status *BackupStatus
}

func (find *BackupFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// BackupPatch is the API message for patching a backup.
type BackupPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status  string
	Comment string
	Payload string
}

// BackupSetting is the backup setting for a database.
type BackupSetting struct {
	ID int `jsonapi:"primary,backupSetting"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	DatabaseID int `jsonapi:"attr,databaseId"`
	// Do not return this to the client since the client always has the database context and fetching the
	// database object and all its own related objects is a bit expensive.
	Database *Database

	// Domain specific fields
	Enabled   bool `jsonapi:"attr,enabled"`
	Hour      int  `jsonapi:"attr,hour"`
	DayOfWeek int  `jsonapi:"attr,dayOfWeek"`
	// HookURL is the callback url to be requested (using HTTP GET) after a successful backup.
	HookURL string `jsonapi:"attr,hookUrl"`
}

// BackupSettingFind is the message to get a backup settings.
type BackupSettingFind struct {
	ID *int

	// Related fields
	DatabaseID *int

	// Domain specific fields
}

// BackupSettingUpsert is the message to upsert a backup settings.
// NOTE: We use PATCH for Upsert, this is inspired by https://google.aip.dev/134#patch-and-put
type BackupSettingUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	// CreatorID is the ID of the creator.
	UpdaterID int

	// Related fields
	DatabaseID    int `jsonapi:"attr,databaseId"`
	EnvironmentID int

	// Domain specific fields
	Enabled   bool   `jsonapi:"attr,enabled"`
	Hour      int    `jsonapi:"attr,hour"`
	DayOfWeek int    `jsonapi:"attr,dayOfWeek"`
	HookURL   string `jsonapi:"attr,hookUrl"`
}

// BackupSettingsMatch is the message to find backup settings matching the conditions.
type BackupSettingsMatch struct {
	Hour      int
	DayOfWeek int
}
