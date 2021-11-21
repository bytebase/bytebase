package api

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
)

type TaskCheckRunStatus string

const (
	TaskCheckRunUnknown  TaskCheckRunStatus = "UNKNOWN"
	TaskCheckRunRunning  TaskCheckRunStatus = "RUNNING"
	TaskCheckRunDone     TaskCheckRunStatus = "DONE"
	TaskCheckRunFailed   TaskCheckRunStatus = "FAILED"
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

type TaskCheckStatus string

const (
	TaskCheckStatusSuccess TaskCheckStatus = "SUCCESS"
	TaskCheckStatusWarn    TaskCheckStatus = "WARN"
	TaskCheckStatusError   TaskCheckStatus = "ERROR"
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
	TaskCheckDatabaseStatementFakeAdvise    TaskCheckType = "bb.task-check.database.statement.fake-advise"
	TaskCheckDatabaseStatementSyntax        TaskCheckType = "bb.task-check.database.statement.syntax"
	TaskCheckDatabaseStatementCompatibility TaskCheckType = "bb.task-check.database.statement.compatibility"
	TaskCheckDatabaseConnect                TaskCheckType = "bb.task-check.database.connect"
	TaskCheckInstanceMigrationSchema        TaskCheckType = "bb.task-check.instance.migration-schema"
)

type TaskCheckDatabaseStatementAdvisePayload struct {
	Statement string  `json:"statement,omitempty"`
	DbType    db.Type `json:"dbType,omitempty"`
	Charset   string  `json:"charset,omitempty"`
	Collation string  `json:"collation,omitempty"`
}

type TaskCheckResult struct {
	Status  TaskCheckStatus `json:"status,omitempty"`
	Code    common.Code     `json:"code,omitempty"`
	Title   string          `json:"title,omitempty"`
	Content string          `json:"content,omitempty"`
}

type TaskCheckRunResultPayload struct {
	Detail     string            `json:"detail,omitempty"`
	ResultList []TaskCheckResult `json:"resultList,omitempty"`
}

type TaskCheckRun struct {
	ID int `jsonapi:"primary,taskCheckRun"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	TaskID int `jsonapi:"attr,taskID"`

	// Domain specific fields
	Status  TaskCheckRunStatus `jsonapi:"attr,status"`
	Type    TaskCheckType      `jsonapi:"attr,type"`
	Code    common.Code        `jsonapi:"attr,code"`
	Comment string             `jsonapi:"attr,comment"`
	Result  string             `jsonapi:"attr,result"`
	Payload string             `jsonapi:"attr,payload"`
}

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

type TaskCheckRunStatusPatch struct {
	ID *int

	// Standard fields
	UpdaterID int

	// Domain specific fields
	Status TaskCheckRunStatus
	Code   common.Code
	Result string
}

type TaskCheckRunService interface {
	// For a particular task and a particular check type, we only create a new TaskCheckRun if matches all conditions below:
	// 1. There is no existing RUNNING check run. If this is the case, then returns that RUNNING check run.
	// 2. If SkipIfAlreadyTerminated is false, or if SkipIfAlreadyTerminated is true and there is no DONE/FAILED/CANCELED check run. If this is the case,
	//    then returns that terminated check run.
	CreateTaskCheckRunIfNeeded(ctx context.Context, create *TaskCheckRunCreate) (*TaskCheckRun, error)
	CreateTaskCheckRunTx(ctx context.Context, tx *sql.Tx, create *TaskCheckRunCreate) (*TaskCheckRun, error)
	FindTaskCheckRunList(ctx context.Context, find *TaskCheckRunFind) ([]*TaskCheckRun, error)
	FindTaskCheckRunListTx(ctx context.Context, tx *sql.Tx, find *TaskCheckRunFind) ([]*TaskCheckRun, error)
	FindTaskCheckRunTx(ctx context.Context, tx *sql.Tx, find *TaskCheckRunFind) (*TaskCheckRun, error)
	PatchTaskCheckRunStatus(ctx context.Context, patch *TaskCheckRunStatusPatch) (*TaskCheckRun, error)
	PatchTaskCheckRunStatusTx(ctx context.Context, tx *sql.Tx, patch *TaskCheckRunStatusPatch) (*TaskCheckRun, error)
}
