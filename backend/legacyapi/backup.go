package api

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
