package api

import (
	"context"
	"encoding/json"
)

type TaskStatus string

const (
	TaskPending         TaskStatus = "PENDING"
	TaskPendingApproval TaskStatus = "PENDING_APPROVAL"
	TaskRunning         TaskStatus = "RUNNING"
	TaskDone            TaskStatus = "DONE"
	TaskFailed          TaskStatus = "FAILED"
	TaskCanceled        TaskStatus = "CANCELED"
)

func (e TaskStatus) String() string {
	switch e {
	case TaskPending:
		return "PENDING"
	case TaskPendingApproval:
		return "PENDING_APPROVAL"
	case TaskRunning:
		return "RUNNING"
	case TaskDone:
		return "DONE"
	case TaskFailed:
		return "FAILED"
	case TaskCanceled:
		return "CANCELED"
	}
	return "UNKNOWN"
}

type TaskType string

const (
	TaskGeneral              TaskType = "bb.task.general"
	TaskDatabaseSchemaUpdate TaskType = "bb.task.database.schema.update"
)

// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention
type TaskDatabaseSchemaUpdatePayload struct {
	Statement         string `json:"statement,omitempty"`
	RollbackStatement string `json:"rollbackStatement,omitempty"`
}

type Task struct {
	ID int `jsonapi:"primary,task"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns PipelineId and StageId otherwise would cause circular dependency.
	PipelineId  int `jsonapi:"attr,pipelineId"`
	StageId     int `jsonapi:"attr,stageId"`
	DatabaseId  int
	Database    *Database  `jsonapi:"relation,database"`
	TaskRunList []*TaskRun `jsonapi:"relation,taskRun"`

	// Domain specific fields
	Name    string     `jsonapi:"attr,name"`
	Status  TaskStatus `jsonapi:"attr,status"`
	Type    TaskType   `jsonapi:"attr,type"`
	Payload string     `jsonapi:"attr,payload"`
}

type TaskCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	PipelineId int
	StageId    int
	DatabaseId int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name    string     `jsonapi:"attr,name"`
	Status  TaskStatus `jsonapi:"attr,status"`
	Type    TaskType   `jsonapi:"attr,type"`
	Payload string     `jsonapi:"attr,payload"`
}

type TaskFind struct {
	ID *int

	// Related fields
	PipelineId *int
	StageId    *int

	// Domain specific fields
	Status *TaskStatus `jsonapi:"attr,status"`
}

func (find *TaskFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type TaskStatusPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Status  TaskStatus `jsonapi:"attr,status"`
	Comment string     `jsonapi:"attr,comment"`
}

type TaskService interface {
	CreateTask(ctx context.Context, create *TaskCreate) (*Task, error)
	FindTaskList(ctx context.Context, find *TaskFind) ([]*Task, error)
	FindTask(ctx context.Context, find *TaskFind) (*Task, error)
	PatchTaskStatus(ctx context.Context, patch *TaskStatusPatch) (*Task, error)
}
