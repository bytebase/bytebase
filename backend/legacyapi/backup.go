package api

const (
	// BackupRetentionPeriodUnset is the unset value of a backup retention period.
	BackupRetentionPeriodUnset = 0
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

// BackupType is the type of a backup.
type BackupType string

const (
	// BackupTypeAutomatic is the type for automatic backup.
	BackupTypeAutomatic BackupType = "AUTOMATIC"
	// BackupTypePITR is the type of backup taken at PITR cutover stage.
	BackupTypePITR BackupType = "PITR"
	// BackupTypeManual is the type for manual backup.
	BackupTypeManual BackupType = "MANUAL"
)

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

// BinlogInfo is the binlog coordination for MySQL.
type BinlogInfo struct {
	FileName string `json:"fileName"`
	Position int64  `json:"position"`
}

// IsEmpty return true if the BinlogInfo is empty.
func (b BinlogInfo) IsEmpty() bool {
	return b == BinlogInfo{}
}

// BackupPayload contains backup related database specific info, it differs for different database types.
// It is encoded in JSON and stored in the backup table.
type BackupPayload struct {
	// MySQL related fields
	// BinlogInfo is recorded when taking the backup.
	// It is recorded within the same transaction as the dump so that the binlog position is consistent with the dump.
	// Please refer to https://github.com/bytebase/bytebase/blob/main/docs/design/pitr-mysql.md#full-backup for details.
	BinlogInfo BinlogInfo `json:"binlogInfo"`
}

// Backup is the API message for a backup.
type Backup struct {
	ID int `jsonapi:"primary,backup"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatedTs int64     `jsonapi:"attr,createdTs"`
	UpdatedTs int64     `jsonapi:"attr,updatedTs"`

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
	Name                    string     `jsonapi:"attr,name"`
	Type                    BackupType `jsonapi:"attr,type"`
	StorageBackend          BackupStorageBackend
	MigrationHistoryVersion string
	Path                    string
}
