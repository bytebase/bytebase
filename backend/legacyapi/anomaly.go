package api

// AnomalyType is the type of a task.
type AnomalyType string

const (
	// AnomalyInstanceConnection is the anomaly type for instance connections.
	AnomalyInstanceConnection AnomalyType = "bb.anomaly.instance.connection"
	// AnomalyInstanceMigrationSchema is the anomaly type for schema migrations.
	AnomalyInstanceMigrationSchema AnomalyType = "bb.anomaly.instance.migration-schema"
	// AnomalyDatabaseBackupPolicyViolation is the anomaly type for backup policy violations.
	AnomalyDatabaseBackupPolicyViolation AnomalyType = "bb.anomaly.database.backup.policy-violation"
	// AnomalyDatabaseBackupMissing is the anomaly type for missing backups.
	AnomalyDatabaseBackupMissing AnomalyType = "bb.anomaly.database.backup.missing"
	// AnomalyDatabaseConnection is the anomaly type for database connections.
	AnomalyDatabaseConnection AnomalyType = "bb.anomaly.database.connection"
	// AnomalyDatabaseSchemaDrift is the anomaly type for database schema drifts.
	AnomalyDatabaseSchemaDrift AnomalyType = "bb.anomaly.database.schema.drift"
)

// AnomalyInstanceConnectionPayload is the API message for instance connection payloads.
type AnomalyInstanceConnectionPayload struct {
	// Connection failure detail
	Detail string `json:"detail,omitempty"`
}

// AnomalyDatabaseBackupPolicyViolationPayload is the API message for backup policy violation payloads.
type AnomalyDatabaseBackupPolicyViolationPayload struct {
	EnvironmentID          int                      `json:"environmentId,omitempty"`
	ExpectedBackupSchedule BackupPlanPolicySchedule `json:"expectedSchedule,omitempty"`
	ActualBackupSchedule   BackupPlanPolicySchedule `json:"actualSchedule,omitempty"`
}

// AnomalyDatabaseBackupMissingPayload is the API message for missing backup payloads.
type AnomalyDatabaseBackupMissingPayload struct {
	ExpectedBackupSchedule BackupPlanPolicySchedule `json:"expectedSchedule,omitempty"`
	// Time of last successful backup created
	LastBackupTs int64 `json:"lastBackupTs,omitempty"`
}

// AnomalyDatabaseConnectionPayload is the API message for database connection payloads.
type AnomalyDatabaseConnectionPayload struct {
	// Connection failure detail
	Detail string `json:"detail,omitempty"`
}

// AnomalyDatabaseSchemaDriftPayload is the API message for database schema drift payloads.
type AnomalyDatabaseSchemaDriftPayload struct {
	// The schema version corresponds to the expected schema
	Version string `json:"version,omitempty"`
	// The expected latest schema stored in the migration history table
	Expect string `json:"expect,omitempty"`
	// The actual schema dumped from the database
	Actual string `json:"actual,omitempty"`
}
