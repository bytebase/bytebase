package api

import (
	"context"
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

// AnomalyRaw is the store model for an Anomaly.
// Fields have exactly the same meanings as Anomaly.
type AnomalyRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	InstanceID int
	DatabaseID *int

	// Domain specific fields
	Type AnomalyType
	// Calculated field derived from type
	Severity AnomalySeverity
	Payload  string
}

// ToAnomaly creates an instance of Anomaly based on the AnomalyRaw.
// This is intended to be called when we need to compose an Anomaly relationship.
func (raw *AnomalyRaw) ToAnomaly() *Anomaly {
	return &Anomaly{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		InstanceID: raw.InstanceID,
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Type: raw.Type,
		// Calculated field derived from type
		Severity: raw.Severity,
		Payload:  raw.Payload,
	}
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

// AnomalyService is the service for anomaly.
type AnomalyService interface {
	// UpsertActiveAnomaly would update the existing active anomaly if both database id and type match, otherwise create a new one.
	UpsertActiveAnomaly(ctx context.Context, upsert *AnomalyUpsert) (*AnomalyRaw, error)
	FindAnomalyList(ctx context.Context, find *AnomalyFind) ([]*AnomalyRaw, error)
	ArchiveAnomaly(ctx context.Context, archive *AnomalyArchive) error
}
