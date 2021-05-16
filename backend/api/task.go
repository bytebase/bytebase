package api

import "context"

type TaskStatus string

const (
	TaskPending TaskStatus = "PENDING"
	TaskRunning TaskStatus = "RUNNING"
	TaskDone    TaskStatus = "DONE"
	TaskFailed  TaskStatus = "FAILED"
	TaskSkipped TaskStatus = "SKIPPED"
)

func (e TaskStatus) String() string {
	switch e {
	case TaskPending:
		return "PENDING"
	case TaskRunning:
		return "RUNNING"
	case TaskDone:
		return "DONE"
	case TaskFailed:
		return "FAILED"
	case TaskSkipped:
		return "SKIPPED"
	}
	return "UNKNOWN"
}

type TaskType string

const (
	TaskGeneral              TaskType = "bb.task.general"
	TaskApprove              TaskType = "bb.task.approve"
	TaskDatabaseSchemaUpdate TaskType = "bb.task.database.schema.update"
)

type TaskWhen string

const (
	TaskOnSuccess TaskWhen = "ON_SUCCESS"
	TaskManual    TaskWhen = "MANUAL"
)

type Task struct {
	ID int `jsonapi:"primary,task"`

	// Standard fields
	CreatorId   int
	Creator     *Principal `jsonapi:"relation,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"relation,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Related fields
	// Just returns PipelineId and StageId otherwise would cause circular dependency.
	PipelineId int `jsonapi:"attr,pipelineId"`
	StageId    int `jsonapi:"attr,stageId"`
	DatabaseId int
	Database   *Database `jsonapi:"relation,database"`

	// Domain specific fields
	Name    string     `jsonapi:"attr,name"`
	Status  TaskStatus `jsonapi:"attr,status"`
	Type    TaskType   `jsonapi:"attr,type"`
	When    TaskWhen   `jsonapi:"attr,when"`
	Payload string     `jsonapi:"attr,payload"`
}

type TaskCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Related fields
	PipelineId int `jsonapi:"relation,pipelineId"`
	StageId    int `jsonapi:"relation,stageId"`
	DatabaseId int `jsonapi:"relation,databaseId"`

	// Domain specific fields
	Name    string   `jsonapi:"attr,name"`
	Type    TaskType `jsonapi:"attr,type"`
	When    TaskWhen `jsonapi:"attr,when"`
	Payload string   `jsonapi:"attr,payload"`
}

type TaskFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int

	// Related fields
	PipelineId *int
	StageId    *int
}

type TaskPatch struct {
	ID int `jsonapi:"primary,environmentPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Domain specific fields
	// This is the container containing the pipeline this task belongs.
	ContainerId *int
	Status      *TaskStatus `jsonapi:"attr,status"`
	Comment     *string     `jsonapi:"attr,comment"`
}

type TaskService interface {
	CreateTask(ctx context.Context, create *TaskCreate) (*Task, error)
	FindTaskList(ctx context.Context, find *TaskFind) ([]*Task, error)
	FindTask(ctx context.Context, find *TaskFind) (*Task, error)
	PatchTask(ctx context.Context, patch *TaskPatch) (*Task, error)
}
