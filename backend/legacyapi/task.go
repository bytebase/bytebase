package api

import (
	"encoding/json"

	"github.com/pkg/errors"
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
	// TaskDatabaseCreate is the task type for creating databases.
	TaskDatabaseCreate TaskType = "bb.task.database.create"
	// TaskDatabaseSchemaBaseline is the task type for database schema baseline.
	TaskDatabaseSchemaBaseline TaskType = "bb.task.database.schema.baseline"
	// TaskDatabaseSchemaUpdate is the task type for updating database schemas.
	TaskDatabaseSchemaUpdate TaskType = "bb.task.database.schema.update"
	// TaskDatabaseSchemaUpdateSDL is the task type for updating database schemas via state-based migration.
	TaskDatabaseSchemaUpdateSDL TaskType = "bb.task.database.schema.update-sdl"
	// TaskDatabaseSchemaUpdateGhost is the task type for updating database schemas using gh-ost.
	TaskDatabaseSchemaUpdateGhost TaskType = "bb.task.database.schema.update-ghost"
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
		TaskDatabaseSchemaUpdateGhost,
		TaskDatabaseSchemaBaseline:
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
		TaskDatabaseSchemaUpdateGhost:
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
