package api

import (
	"context"
	"encoding/json"
)

// AnomalyType is the type of a task.
type AnomalyType string

const (
	AnomalyBackupPolicyViolation AnomalyType = "bb.anomaly.backup.policy-violation"
	AnomalyBackupMissing         AnomalyType = "bb.anomaly.backup.missing"
)

type AnomalyBackupPolicyViolationPayload struct {
	EnvironmentId          int                      `json:"environmentId,omitempty"`
	ExpectedBackupSchedule BackupPlanPolicySchedule `json:"expectedSchedule,omitempty"`
	ActualBackupSchedule   BackupPlanPolicySchedule `json:"actualSchedule,omitempty"`
}

type AnomalyBackupMissingPayload struct {
	ExpectedBackupSchedule BackupPlanPolicySchedule `json:"expectedSchedule,omitempty"`
	// Time of last successful backup created
	LastBackupTs int64 `json:"lastBackupTs,omitempty"`
}

type Anomaly struct {
	ID int `jsonapi:"primary,anomaly"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	InstanceId int `jsonapi:"attr,instanceId"`
	DatabaseId int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Type    AnomalyType `jsonapi:"attr,type"`
	Payload string      `jsonapi:"attr,payload"`
}

type AnomalyUpsert struct {
	// Standard fields
	CreatorId int

	// Related fields
	InstanceId int
	DatabaseId int

	// Domain specific fields
	Type    AnomalyType `jsonapi:"attr,type"`
	Payload string      `jsonapi:"attr,payload"`
}

type AnomalyFind struct {
	// Standard fields
	RowStatus *RowStatus

	// Related fields
	DatabaseId int
}

func (find *AnomalyFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type AnomalyArchive struct {
	DatabaseId int
	Type       AnomalyType
}

type AnomalyService interface {
	// UpsertAnomaly would update the existing user if name matches.
	UpsertAnomaly(ctx context.Context, upsert *AnomalyUpsert) (*Anomaly, error)
	FindAnomalyList(ctx context.Context, find *AnomalyFind) ([]*Anomaly, error)
	ArchiveAnomaly(ctx context.Context, archive *AnomalyArchive) error
}
