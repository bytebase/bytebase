package api

import (
	"github.com/bytebase/bytebase/backend/common"
)

// TaskRunStatus is the status of a task run.
type TaskRunStatus string

const (
	// TaskRunUnknown is the task run status for UNKNOWN.
	TaskRunUnknown TaskRunStatus = "UNKNOWN"
	// TaskRunPending is the task run status of PENDING.
	TaskRunPending TaskRunStatus = "PENDING"
	// TaskRunRunning is the task run status for RUNNING.
	TaskRunRunning TaskRunStatus = "RUNNING"
	// TaskRunDone is the task run status for DONE.
	TaskRunDone TaskRunStatus = "DONE"
	// TaskRunFailed is the task run status for FAILED.
	TaskRunFailed TaskRunStatus = "FAILED"
	// TaskRunCanceled is the task run status for CANCELED.
	TaskRunCanceled TaskRunStatus = "CANCELED"
	// TaskRunNotStarted is the task run status for NOT_STARTED.
	TaskRunNotStarted TaskRunStatus = "NOT_STARTED"
)

// TaskRunResultPayload is the result payload for a task run.
type TaskRunResultPayload struct {
	Detail        string `json:"detail,omitempty"`
	MigrationID   string `json:"migrationId,omitempty"`
	ChangeHistory string `json:"changeHistory,omitempty"`
	Version       string `json:"version,omitempty"`
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
