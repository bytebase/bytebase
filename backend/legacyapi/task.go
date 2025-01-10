package api

import (
	"encoding/json"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TaskStatus is the status of a task.
type TaskStatus string

const (
	// TaskPending is the task status for PENDING.
	TaskPending TaskStatus = "PENDING"
	// TaskPendingApproval is the task status for PENDING_APPROVAL.
	TaskPendingApproval TaskStatus = "PENDING_APPROVAL"
	// TaskRunning is the task status for RUNNING.
	TaskRunning TaskStatus = "RUNNING"
	// TaskDone is the task status for DONE.
	TaskDone TaskStatus = "DONE"
	// TaskFailed is the task status for FAILED.
	TaskFailed TaskStatus = "FAILED"
	// TaskCanceled is the task status for CANCELED.
	TaskCanceled TaskStatus = "CANCELED"
	// TaskSkipped is the task status for SKIPPED.
	TaskSkipped TaskStatus = "SKIPPED"
)

// TaskType is the type of a task.
type TaskType string

const (
	// TaskGeneral is the task type for general tasks.
	TaskGeneral TaskType = "bb.task.general"
	// TaskDatabaseCreate is the task type for creating databases.
	TaskDatabaseCreate TaskType = "bb.task.database.create"
	// TaskDatabaseSchemaBaseline is the task type for database schema baseline.
	TaskDatabaseSchemaBaseline TaskType = "bb.task.database.schema.baseline"
	// TaskDatabaseSchemaUpdate is the task type for updating database schemas.
	TaskDatabaseSchemaUpdate TaskType = "bb.task.database.schema.update"
	// TaskDatabaseSchemaUpdateSDL is the task type for updating database schemas via state-based migration.
	TaskDatabaseSchemaUpdateSDL TaskType = "bb.task.database.schema.update-sdl"
	// TaskDatabaseSchemaUpdateGhostSync is the task type for gh-ost syncing ghost table.
	TaskDatabaseSchemaUpdateGhostSync TaskType = "bb.task.database.schema.update.ghost.sync"
	// TaskDatabaseSchemaUpdateGhostCutover is the task type for gh-ost switching the original table and the ghost table.
	TaskDatabaseSchemaUpdateGhostCutover TaskType = "bb.task.database.schema.update.ghost.cutover"
	// TaskDatabaseDataUpdate is the task type for updating database data.
	TaskDatabaseDataUpdate TaskType = "bb.task.database.data.update"
	// TaskDatabaseDataExport is the task type for exporting database data.
	TaskDatabaseDataExport TaskType = "bb.task.database.data.export"
)

func (t TaskType) ChangeDatabasePayload() bool {
	switch t {
	case
		TaskDatabaseDataUpdate,
		TaskDatabaseSchemaUpdate,
		TaskDatabaseSchemaUpdateSDL,
		TaskDatabaseSchemaBaseline,
		TaskDatabaseSchemaUpdateGhostSync:
		return true
	default:
		return false
	}
}

// Sequential returns whether the task should be executed sequentially.
func (t TaskType) Sequential() bool {
	switch t {
	case
		TaskDatabaseSchemaUpdate,
		TaskDatabaseSchemaUpdateSDL,
		TaskDatabaseSchemaUpdateGhostSync,
		TaskDatabaseSchemaUpdateGhostCutover:
		return true
	default:
		return false
	}
}

// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention

// Progress is a generalized struct which can track the progress of a task.
type Progress struct {
	// TotalUnit is the total unit count of the task
	TotalUnit int64 `json:"totalUnit"`
	// CompletedUnit is the finished task units
	CompletedUnit int64 `json:"completedUnit"`
	// CreatedTs is when the task starts
	CreatedTs int64 `json:"createdTs"`
	// UpdatedTs is when the progress gets updated most recently
	UpdatedTs int64 `json:"updatedTs"`
	// Payload is reserved for the future
	// Might be something like {comment:"postponing due to network lag"}
	Payload string `json:"payload"`
}

// TaskFind is the API message for finding tasks.
type TaskFind struct {
	ID  *int
	IDs *[]int

	// Related fields
	PipelineID *int
	StageID    *int
	DatabaseID *int

	// Domain specific fields
	TypeList *[]TaskType
	// Payload contains JSONB expressions
	// Ref: https://www.postgresql.org/docs/current/functions-json.html
	Payload         string
	NoBlockingStage bool
	NonRollbackTask bool

	LatestTaskRunStatusList *[]TaskRunStatus
}

func (find *TaskFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// TaskPatch is the API message for patching a task.
type TaskPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	DatabaseID        *int
	EarliestAllowedTs *int64 `jsonapi:"attr,earliestAllowedTs"`

	// Payload and others cannot be set at the same time.
	Payload *string

	SheetID               *int `jsonapi:"attr,sheetId"`
	SchemaVersion         *string
	ExportFormat          *storepb.ExportFormat
	ExportPassword        *string
	PreUpdateBackupDetail *storepb.PreUpdateBackupDetail

	// Flags for gh-ost.
	Flags *map[string]string
}

func GetSheetUIDFromTaskPayload(payload string) (*int, error) {
	var taskPayload struct {
		SheetID int `json:"sheetId"`
	}
	if err := json.Unmarshal([]byte(payload), &taskPayload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task payload")
	}
	if taskPayload.SheetID == 0 {
		return nil, nil
	}
	return &taskPayload.SheetID, nil
}
