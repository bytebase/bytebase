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
	TaskSkipped         TaskStatus = "SKIPPED"
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
	case TaskSkipped:
		return "SKIPPED"
	}
	return "UNKNOWN"
}

func (e TaskStatus) IsEndStatus() bool {
	return e == TaskDone || e == TaskSkipped
}

type TaskType string

const (
	TaskGeneral              TaskType = "bb.task.general"
	TaskDatabaseSchemaUpdate TaskType = "bb.task.database.schema.update"
)

type TaskDatabaseSchemaUpdatePayload struct {
	Sql         string
	RollbackSql string
}

type Task struct {
	ID int `jsonapi:"primary,task"`

	// Standard fields
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

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
	Payload []byte     `jsonapi:"attr,payload"`
}

type TaskCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Related fields
	PipelineId int
	StageId    int
	DatabaseId int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name    string     `jsonapi:"attr,name"`
	Status  TaskStatus `jsonapi:"attr,status"`
	Type    TaskType   `jsonapi:"attr,type"`
	Payload []byte     `jsonapi:"attr,payload"`
}

type TaskFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int

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

type TaskApprove struct {
	// Domain specific fields
	Comment string `jsonapi:"attr,comment"`
}

type TaskStatusPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

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
