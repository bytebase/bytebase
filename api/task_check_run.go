package api

import (
	"context"
	"database/sql"
	"encoding/json"

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
	TaskCheckDatabaseStatementFakeAdvise TaskCheckType = "bb.task-check.database.statement.fake-advise"
	TaskCheckDatabaseStatementSyntax     TaskCheckType = "bb.task-check.database.statement.syntax"
	TaskCheckDatabaseConnect             TaskCheckType = "bb.task-check.database.connect"
)

type TaskCheckCode int

const (
	// 0 ~ 99 Common error code
	TaskCheckOk           TaskCheckCode = 0
	TaskCheckDbConnection TaskCheckCode = 1

	// 101 ~ 199 statement advisor error code
	TaskCheckStatementSyntax TaskCheckCode = 101
)

type TaskCheckDatabaseStatementAdvisePayload struct {
	Statement string  `json:"statement,omitempty"`
	DbType    db.Type `json:"dbType,omitempty"`
	Charset   string  `json:"charset,omitempty"`
	Collation string  `json:"collation,omitempty"`
}

type TaskCheckResult struct {
	Status  TaskCheckStatus `json:"status"`
	Code    TaskCheckCode   `json:"code"`
	Title   string          `json:"title"`
	Content string          `json:"content"`
}

type TaskCheckRunResultPayload struct {
	ResultList []TaskCheckResult `json:"resultList"`
}

type TaskCheckRun struct {
	ID int `jsonapi:"primary,taskCheckRun"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	TaskId int `jsonapi:"attr,taskId"`

	// Domain specific fields
	Status  TaskCheckRunStatus `jsonapi:"attr,status"`
	Type    TaskCheckType      `jsonapi:"attr,type"`
	Comment string             `jsonapi:"attr,comment"`
	Result  string             `jsonapi:"attr,result"`
	Payload string             `jsonapi:"attr,payload"`
}

type TaskCheckRunCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	TaskId int

	// Domain specific fields
	Type    TaskCheckType `jsonapi:"attr,type"`
	Comment string        `jsonapi:"attr,comment"`
	Payload string        `jsonapi:"attr,payload"`

	// If true, then we will skip creating the task check run if there has already been a DONE check run
	// for this (TaskId, Type) pair. The check is done at the store layer so that we can wrap it in the
	// same transaction.
	// This is NOT persisted into the db
	SkipIfAlreadyTerminated bool
}

type TaskCheckRunFind struct {
	ID *int

	// Related fields
	TaskId *int
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
	UpdaterId int

	// Domain specific fields
	Status TaskCheckRunStatus
	// Records the status detail (e.g. error message on failure)
	Comment string
	Result  string
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
