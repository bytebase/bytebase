package scheduler

import "context"

type TaskId = int

type TaskRunId = string

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

type Task struct {
	ID      TaskId
	Name    string
	Type    string
	Payload []byte
}

type TaskRun struct {
	ID        TaskRunId
	CreatedTs int64
	UpdatedTs int64
	TaskId    TaskId
	Name      string
	Type      string
	Status    TaskRunStatus
	Payload   []byte
}

type TaskRunCreate struct {
	ID      TaskRunId
	TaskId  TaskId
	Name    string
	Type    string
	Payload []byte
}

type TaskRunFind struct {
	ID     *TaskRunId
	Status *TaskRunStatus
}

type TaskRunStatusPatch struct {
	ID     TaskRunId
	Status TaskRunStatus
}

type TaskRunService interface {
	CreateTaskRun(ctx context.Context, create *TaskRunCreate) (*TaskRun, error)
	FindTaskRunList(ctx context.Context, find *TaskRunFind) ([]*TaskRun, error)
	FindTaskRun(ctx context.Context, find *TaskRunFind) (*TaskRun, error)
	PatchTaskRunStatus(ctx context.Context, patch *TaskRunStatusPatch) (*TaskRun, error)
}
