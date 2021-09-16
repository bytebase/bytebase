package api

import (
	"context"
	"database/sql"
	"encoding/json"
)

type TaskRunStatus string

const (
	TaskRunUnknown  TaskRunStatus = "UNKNOWN"
	TaskRunRunning  TaskRunStatus = "RUNNING"
	TaskRunDone     TaskRunStatus = "DONE"
	TaskRunFailed   TaskRunStatus = "FAILED"
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

type TaskRun struct {
	ID int `jsonapi:"primary,taskRun"`

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
	Name    string        `jsonapi:"attr,name"`
	Status  TaskRunStatus `jsonapi:"attr,status"`
	Type    TaskType      `jsonapi:"attr,type"`
	Comment string        `jsonapi:"attr,comment"`
	Payload string        `jsonapi:"attr,payload"`
}

type TaskRunCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	TaskId int

	// Domain specific fields
	Name    string   `jsonapi:"attr,name"`
	Type    TaskType `jsonapi:"attr,type"`
	Payload string   `jsonapi:"attr,payload"`
}

type TaskRunFind struct {
	ID *int

	// Related fields
	TaskId *int

	// Domain specific fields
	StatusList []TaskRunStatus
}

func (find *TaskRunFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type TaskRunStatusPatch struct {
	ID *int

	// Standard fields
	UpdaterId int

	// Related fields
	TaskId *int

	// Domain specific fields
	Status TaskRunStatus
	// Records the status detail (e.g. error message on failure)
	Comment string
}

type TaskRunService interface {
	CreateTaskRunTx(ctx context.Context, tx *sql.Tx, create *TaskRunCreate) (*TaskRun, error)
	FindTaskRunListTx(ctx context.Context, tx *sql.Tx, find *TaskRunFind) ([]*TaskRun, error)
	FindTaskRunTx(ctx context.Context, tx *sql.Tx, find *TaskRunFind) (*TaskRun, error)
	PatchTaskRunStatusTx(ctx context.Context, tx *sql.Tx, patch *TaskRunStatusPatch) (*TaskRun, error)
}
