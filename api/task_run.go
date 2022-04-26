package api

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/bytebase/bytebase/common"
)

// TaskRunStatus is the status of a task run.
type TaskRunStatus string

const (
	// TaskRunUnknown is the task run status for UNKNOWN.
	TaskRunUnknown TaskRunStatus = "UNKNOWN"
	// TaskRunRunning is the task run status for RUNNING.
	TaskRunRunning TaskRunStatus = "RUNNING"
	// TaskRunDone is the task run status for DONE.
	TaskRunDone TaskRunStatus = "DONE"
	// TaskRunFailed is the task run status for FAILED.
	TaskRunFailed TaskRunStatus = "FAILED"
	// TaskRunCanceled is the task run status for CANCELED.
	TaskRunCanceled TaskRunStatus = "CANCELED"
)

func (e TaskRunStatus) String() string {
	switch e {
	case TaskRunRunning:
		return "RUNNING"
	case TaskRunDone:
		return "DONE"
	case TaskRunFailed:
		return "FAILED"
	case TaskRunCanceled:
		return "CANCELED"
	}
	return "UNKNOWN"
}

// TaskRunResultPayload is the result payload for a task run.
type TaskRunResultPayload struct {
	Detail      string `json:"detail,omitempty"`
	MigrationID int64  `json:"migrationId,omitempty"`
	Version     string `json:"version,omitempty"`
}

// TaskSchemaUpdateGhostSyncRunResultDetail is the detail for a gh-ost sync task.
type TaskSchemaUpdateGhostSyncRunResultDetail struct {
	Progress TaskSchemaUpdateGhostSyncRunProgress `json:"progress,omitempty"`
}

// TaskSchemaUpdateGhostSyncRunProgress describes gh-ost sync task progress.
type TaskSchemaUpdateGhostSyncRunProgress struct {
	RowsCopied   int64  `json:"rowsCopied"`
	RowsEstimate int64  `json:"rowsEstimate"`
	ETA          string `json:"eta"`
}

// TaskRunRaw is the store model for an TaskRun.
// Fields have exactly the same meanings as TaskRun.
type TaskRunRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	TaskID int

	// Domain specific fields
	Name    string
	Status  TaskRunStatus
	Type    TaskType
	Code    common.Code
	Comment string
	Result  string
	Payload string
}

// ToTaskRun creates an instance of TaskRun based on the TaskRunRaw.
// This is intended to be called when we need to compose an TaskRun relationship.
func (raw *TaskRunRaw) ToTaskRun() *TaskRun {
	return &TaskRun{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		TaskID: raw.TaskID,

		// Domain specific fields
		Name:    raw.Name,
		Status:  raw.Status,
		Type:    raw.Type,
		Code:    raw.Code,
		Comment: raw.Comment,
		Result:  raw.Result,
		Payload: raw.Payload,
	}
}

// TaskRun is the API message for a task run.
type TaskRun struct {
	ID int `jsonapi:"primary,taskRun"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	TaskID int `jsonapi:"attr,taskId"`

	// Domain specific fields
	Name    string        `jsonapi:"attr,name"`
	Status  TaskRunStatus `jsonapi:"attr,status"`
	Type    TaskType      `jsonapi:"attr,type"`
	Code    common.Code   `jsonapi:"attr,code"`
	Comment string        `jsonapi:"attr,comment"`
	Result  string        `jsonapi:"attr,result"`
	Payload string        `jsonapi:"attr,payload"`
}

// TaskRunCreate is the API message for creating a task run.
type TaskRunCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	TaskID int

	// Domain specific fields
	Name    string   `jsonapi:"attr,name"`
	Type    TaskType `jsonapi:"attr,type"`
	Payload string   `jsonapi:"attr,payload"`
}

// TaskRunFind is the API message for finding task runs.
type TaskRunFind struct {
	ID *int

	// Related fields
	TaskID *int

	// Domain specific fields
	StatusList *[]TaskRunStatus
}

func (find *TaskRunFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// TaskRunStatusPatch is the API message for patching a task run.
type TaskRunStatusPatch struct {
	ID *int

	// Standard fields
	UpdaterID int

	// Related fields
	TaskID *int

	// Domain specific fields
	Status TaskRunStatus
	Code   *common.Code
	// Records the status detail (e.g. error message on failure)
	Comment *string
	Result  *string
}

// TaskRunService is the service for task runs.
type TaskRunService interface {
	CreateTaskRunTx(ctx context.Context, tx *sql.Tx, create *TaskRunCreate) (*TaskRunRaw, error)
	FindTaskRunListTx(ctx context.Context, tx *sql.Tx, find *TaskRunFind) ([]*TaskRunRaw, error)
	FindTaskRunTx(ctx context.Context, tx *sql.Tx, find *TaskRunFind) (*TaskRunRaw, error)
	PatchTaskRunStatusTx(ctx context.Context, tx *sql.Tx, patch *TaskRunStatusPatch) (*TaskRunRaw, error)
}
