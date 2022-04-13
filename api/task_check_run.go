package api

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
)

// TaskCheckRunStatus is the status of a task check run.
type TaskCheckRunStatus string

const (
	// TaskCheckRunUnknown is the task check run status for UNKNOWN.
	TaskCheckRunUnknown TaskCheckRunStatus = "UNKNOWN"
	// TaskCheckRunRunning is the task check run status for RUNNING.
	TaskCheckRunRunning TaskCheckRunStatus = "RUNNING"
	// TaskCheckRunDone is the task check run status for DONE.
	TaskCheckRunDone TaskCheckRunStatus = "DONE"
	// TaskCheckRunFailed is the task check run status for FAILED.
	TaskCheckRunFailed TaskCheckRunStatus = "FAILED"
	// TaskCheckRunCanceled is the task check run status for CANCELED.
	TaskCheckRunCanceled TaskCheckRunStatus = "CANCELED"
)

func (e TaskCheckRunStatus) String() string {
	switch e {
	case TaskCheckRunRunning:
		return "RUNNING"
	case TaskCheckRunDone:
		return "DONE"
	case TaskCheckRunFailed:
		return "FAILED"
	case TaskCheckRunCanceled:
		return "CANCELED"
	}
	return "UNKNOWN"
}

// TaskCheckStatus is the status of a task check.
type TaskCheckStatus string

const (
	// TaskCheckStatusSuccess is the task check status for SUCCESS.
	TaskCheckStatusSuccess TaskCheckStatus = "SUCCESS"
	// TaskCheckStatusWarn is the task check status for WARN.
	TaskCheckStatusWarn TaskCheckStatus = "WARN"
	// TaskCheckStatusError is the task check status for ERROR.
	TaskCheckStatusError TaskCheckStatus = "ERROR"
)

func (e TaskCheckStatus) String() string {
	switch e {
	case TaskCheckStatusSuccess:
		return "SUCCESS"
	case TaskCheckStatusWarn:
		return "WARN"
	case TaskCheckStatusError:
		return "ERROR"
	}
	return "UNKNOWN"
}

// TaskCheckType is the type of a taskCheck.
type TaskCheckType string

const (
	// TaskCheckDatabaseStatementFakeAdvise is the task check type for fake advise.
	TaskCheckDatabaseStatementFakeAdvise TaskCheckType = "bb.task-check.database.statement.fake-advise"
	// TaskCheckDatabaseStatementSyntax is the task check type for statement syntax.
	TaskCheckDatabaseStatementSyntax TaskCheckType = "bb.task-check.database.statement.syntax"
	// TaskCheckDatabaseStatementCompatibility is the task check type for statement compatibility.
	TaskCheckDatabaseStatementCompatibility TaskCheckType = "bb.task-check.database.statement.compatibility"
	// TaskCheckDatabaseStatementRequireWhere is the task check type for the WHRER clause requirement for UPDATE/DELETE.
	TaskCheckDatabaseStatementRequireWhere TaskCheckType = "bb.task-check.database.statement.where.require"
	// TaskCheckDatabaseConnect is the task check type for database connection.
	TaskCheckDatabaseConnect TaskCheckType = "bb.task-check.database.connect"
	// TaskCheckInstanceMigrationSchema is the task check type for migrating schemas.
	TaskCheckInstanceMigrationSchema TaskCheckType = "bb.task-check.instance.migration-schema"
	// TaskCheckGeneralEarliestAllowedTime is the task check type for earliest allowed time.
	TaskCheckGeneralEarliestAllowedTime TaskCheckType = "bb.task-check.general.earliest-allowed-time"
)

// TaskCheckEarliestAllowedTimePayload is the task check payload for earliest allowed time.
type TaskCheckEarliestAllowedTimePayload struct {
	EarliestAllowedTs int64 `json:"earliestAllowedTs,omitempty"`
}

// TaskCheckDatabaseStatementAdvisePayload is the task check payload for database statement advise.
type TaskCheckDatabaseStatementAdvisePayload struct {
	Statement string  `json:"statement,omitempty"`
	DbType    db.Type `json:"dbType,omitempty"`
	Charset   string  `json:"charset,omitempty"`
	Collation string  `json:"collation,omitempty"`
}

// TaskCheckResult is the result of task checks.
type TaskCheckResult struct {
	Status  TaskCheckStatus `json:"status,omitempty"`
	Code    common.Code     `json:"code,omitempty"`
	Title   string          `json:"title,omitempty"`
	Content string          `json:"content,omitempty"`
}

// TaskCheckRunResultPayload is the result payload of a task check run.
type TaskCheckRunResultPayload struct {
	Detail     string            `json:"detail,omitempty"`
	ResultList []TaskCheckResult `json:"resultList,omitempty"`
}

// TaskCheckRunRaw is the store model for a TaskCheckRun.
// Fields have exactly the same meanings as TaskCheckRun.
type TaskCheckRunRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	TaskID int

	// Domain specific fields
	Status  TaskCheckRunStatus
	Type    TaskCheckType
	Code    common.Code
	Comment string
	Result  string
	Payload string
}

// ToTaskCheckRun creates an instance of TaskCheckRun based on the TaskCheckRunRaw.
// This is intended to be called when we need to compose a TaskCheckRun relationship.
func (raw *TaskCheckRunRaw) ToTaskCheckRun() *TaskCheckRun {
	return &TaskCheckRun{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		TaskID: raw.TaskID,

		// Domain specific fields
		Status:  raw.Status,
		Type:    raw.Type,
		Code:    raw.Code,
		Comment: raw.Comment,
		Result:  raw.Result,
		Payload: raw.Payload,
	}
}

// TaskCheckRun is the API message for task check run.
type TaskCheckRun struct {
	ID int `jsonapi:"primary,taskCheckRun"`

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
	Status  TaskCheckRunStatus `jsonapi:"attr,status"`
	Type    TaskCheckType      `jsonapi:"attr,type"`
	Code    common.Code        `jsonapi:"attr,code"`
	Comment string             `jsonapi:"attr,comment"`
	Result  string             `jsonapi:"attr,result"`
	Payload string             `jsonapi:"attr,payload"`
}

// TaskCheckRunCreate is the API message for creating a task check run.
type TaskCheckRunCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	TaskID int

	// Domain specific fields
	Type    TaskCheckType `jsonapi:"attr,type"`
	Comment string        `jsonapi:"attr,comment"`
	Payload string        `jsonapi:"attr,payload"`

	// If true, then we will skip creating the task check run if there has already been a DONE check run
	// for this (TaskID, Type) pair. The check is done at the store layer so that we can wrap it in the
	// same transaction.
	// This is NOT persisted into the db
	SkipIfAlreadyTerminated bool
}

// TaskCheckRunFind is the API message for finding task check runs.
type TaskCheckRunFind struct {
	ID *int

	// Related fields
	TaskID *int
	Type   *TaskCheckType

	// Domain specific fields
	StatusList *[]TaskCheckRunStatus
	// If true, only returns the latest
	Latest bool
}

func (find *TaskCheckRunFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// TaskCheckRunStatusPatch is the API message for patching a task check run.
type TaskCheckRunStatusPatch struct {
	ID *int

	// Standard fields
	UpdaterID int

	// Domain specific fields
	Status TaskCheckRunStatus
	Code   common.Code
	Result string
}

// TaskCheckRunService is the service for task check runs.
type TaskCheckRunService interface {
	// For a particular task and a particular check type, we only create a new TaskCheckRun if matches all conditions below:
	// 1. There is no existing RUNNING check run. If this is the case, then returns that RUNNING check run.
	// 2. If SkipIfAlreadyTerminated is false, or if SkipIfAlreadyTerminated is true and there is no DONE/FAILED/CANCELED check run. If this is the case,
	//    then returns that terminated check run.
	CreateTaskCheckRunIfNeeded(ctx context.Context, create *TaskCheckRunCreate) (*TaskCheckRunRaw, error)
	FindTaskCheckRunList(ctx context.Context, find *TaskCheckRunFind) ([]*TaskCheckRunRaw, error)
	FindTaskCheckRunListTx(ctx context.Context, tx *sql.Tx, find *TaskCheckRunFind) ([]*TaskCheckRunRaw, error)
	PatchTaskCheckRunStatus(ctx context.Context, patch *TaskCheckRunStatusPatch) (*TaskCheckRunRaw, error)
}
