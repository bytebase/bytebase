package api

import "context"

type TaskRunStatus string

const (
	TaskRunUnknown  TaskRunStatus = "UNKNOWN"
	TaskRunPending  TaskRunStatus = "PENDING"
	TaskRunRunning  TaskRunStatus = "RUNNING"
	TaskRunDone     TaskRunStatus = "DONE"
	TaskRunFailed   TaskRunStatus = "FAILED"
	TaskRunCanceled TaskRunStatus = "CANCELED"
)

func (e TaskRunStatus) String() string {
	switch e {
	case TaskRunPending:
		return "PENDING"
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
	ID int

	// Standard fields
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Related fields
	TaskId int `jsonapi:"attr,taskId"`

	// Domain specific fields
	Name    string        `jsonapi:"attr,name"`
	Status  TaskRunStatus `jsonapi:"attr,status"`
	Type    TaskType      `jsonapi:"attr,type"`
	Payload []byte        `jsonapi:"attr,payload"`
}

type TaskRunCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Related fields
	TaskId int

	// Domain specific fields
	Name    string   `jsonapi:"attr,name"`
	Type    TaskType `jsonapi:"attr,type"`
	Payload []byte   `jsonapi:"attr,payload"`
}

type TaskRunFind struct {
	ID *int

	// Related fields
	TaskId *int

	// Domain specific fields
	Status *TaskRunStatus
}

type TaskRunStatusChange struct {
	ID int

	// Domain specific fields
	Status TaskRunStatus
}

type TaskRunService interface {
	CreateTaskRun(ctx context.Context, create *TaskRunCreate) (*TaskRun, error)
	FindTaskRunList(ctx context.Context, find *TaskRunFind) ([]*TaskRun, error)
	FindTaskRun(ctx context.Context, find *TaskRunFind) (*TaskRun, error)
}
