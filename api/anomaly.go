package api

import (
	"encoding/json"
)

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

// AnomalySeverity is the severity of anomaly.
type AnomalySeverity string

const (
	// AnomalySeverityMedium is the medium severity.
	AnomalySeverityMedium AnomalySeverity = "MEDIUM"
	// AnomalySeverityHigh is the high severity.
	AnomalySeverityHigh AnomalySeverity = "HIGH"
	// AnomalySeverityCritical is the critical severity.
	AnomalySeverityCritical AnomalySeverity = "CRITICAL"
)

// AnomalySeverityFromType maps the severity from a anomaly type.
func AnomalySeverityFromType(anomalyType AnomalyType) AnomalySeverity {
	switch anomalyType {
	case AnomalyDatabaseBackupPolicyViolation:
		return AnomalySeverityMedium
	case AnomalyDatabaseBackupMissing:
		return AnomalySeverityHigh
	case AnomalyInstanceConnection:
	case AnomalyInstanceMigrationSchema:
	case AnomalyDatabaseConnection:
	case AnomalyDatabaseSchemaDrift:
		return AnomalySeverityCritical
	}
	return AnomalySeverityCritical
}

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

// Anomaly is the API message for an anomaly.
type Anomaly struct {
	ID int `jsonapi:"primary,anomaly"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	InstanceID int `jsonapi:"attr,instanceId"`
	// Instance anomaly doesn't have databaseID
	DatabaseID *int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Type AnomalyType `jsonapi:"attr,type"`
	// Calculated field derived from type
	Severity AnomalySeverity `jsonapi:"attr,severity"`
	Payload  string          `jsonapi:"attr,payload"`
}

// AnomalyUpsert is the API message for creating an anomaly.
type AnomalyUpsert struct {
	// Standard fields
	CreatorID int

	// Related fields
	InstanceID int
	DatabaseID *int

	// Domain specific fields
	Type    AnomalyType `jsonapi:"attr,type"`
	Payload string      `jsonapi:"attr,payload"`
}

// AnomalyFind is the API message for finding anomalies.
type AnomalyFind struct {
	// Standard fields
	RowStatus *RowStatus

	// Related fields
	InstanceID *int
	DatabaseID *int
	Type       *AnomalyType
	// Only applicable if InstanceID is specified, if true, then we only return instance anomaly (database_id is NULL)
	InstanceOnly bool
}

func (find *AnomalyFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// AnomalyArchive is the API message for archiving an anomaly.
type AnomalyArchive struct {
	InstanceID *int
	DatabaseID *int
	Type       AnomalyType
}
