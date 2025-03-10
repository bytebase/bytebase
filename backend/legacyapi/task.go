package api

// TaskStatus is the status of a task.
type TaskStatus string

const (
	// TaskPending is the task status for PENDING.
	TaskPending TaskStatus = "PENDING"
	// TaskPendingApproval is the task status for PENDING_APPROVAL.
	TaskPendingApproval TaskStatus = "PENDING_APPROVAL"
	// TaskRunning is the task status for RUNNING.
	TaskRunning TaskStatus = "RUNNING"
	// TaskDone is the task status for DONE.
	TaskDone TaskStatus = "DONE"
	// TaskFailed is the task status for FAILED.
	TaskFailed TaskStatus = "FAILED"
	// TaskCanceled is the task status for CANCELED.
	TaskCanceled TaskStatus = "CANCELED"
	// TaskSkipped is the task status for SKIPPED.
	TaskSkipped TaskStatus = "SKIPPED"
)

// TaskType is the type of a task.
type TaskType string

const (
	// TaskDatabaseCreate is the task type for creating databases.
	TaskDatabaseCreate TaskType = "bb.task.database.create"
	// TaskDatabaseSchemaBaseline is the task type for database schema baseline.
	TaskDatabaseSchemaBaseline TaskType = "bb.task.database.schema.baseline"
	// TaskDatabaseSchemaUpdate is the task type for updating database schemas.
	TaskDatabaseSchemaUpdate TaskType = "bb.task.database.schema.update"
	// TaskDatabaseSchemaUpdateGhost is the task type for updating database schemas using gh-ost.
	TaskDatabaseSchemaUpdateGhost TaskType = "bb.task.database.schema.update-ghost"
	// TaskDatabaseDataUpdate is the task type for updating database data.
	TaskDatabaseDataUpdate TaskType = "bb.task.database.data.update"
	// TaskDatabaseDataExport is the task type for exporting database data.
	TaskDatabaseDataExport TaskType = "bb.task.database.data.export"
)

func (t TaskType) ChangeDatabasePayload() bool {
	switch t {
	case
		TaskDatabaseDataUpdate,
		TaskDatabaseSchemaUpdate,
		TaskDatabaseSchemaUpdateGhost,
		TaskDatabaseSchemaBaseline:
		return true
	default:
		return false
	}
}

// Sequential returns whether the task should be executed sequentially.
func (t TaskType) Sequential() bool {
	switch t {
	case
		TaskDatabaseSchemaUpdate,
		TaskDatabaseSchemaUpdateGhost:
		return true
	default:
		return false
	}
}
