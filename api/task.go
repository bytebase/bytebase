package api

import (
	"encoding/json"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
)

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
)

// TaskType is the type of a task.
type TaskType string

const (
	// TaskGeneral is the task type for general tasks.
	TaskGeneral TaskType = "bb.task.general"
	// TaskDatabaseCreate is the task type for creating databases.
	TaskDatabaseCreate TaskType = "bb.task.database.create"
	// TaskDatabaseSchemaBaseline is the task type for database schema baseline.
	TaskDatabaseSchemaBaseline TaskType = "bb.task.database.schema.baseline"
	// TaskDatabaseSchemaUpdate is the task type for updating database schemas.
	TaskDatabaseSchemaUpdate TaskType = "bb.task.database.schema.update"
	// TaskDatabaseSchemaUpdateSDL is the task type for updating database schemas via state-based migration.
	TaskDatabaseSchemaUpdateSDL TaskType = "bb.task.database.schema.update-sdl"
	// TaskDatabaseSchemaUpdateGhostSync is the task type for gh-ost syncing ghost table.
	TaskDatabaseSchemaUpdateGhostSync TaskType = "bb.task.database.schema.update.ghost.sync"
	// TaskDatabaseSchemaUpdateGhostCutover is the task type for gh-ost switching the original table and the ghost table.
	TaskDatabaseSchemaUpdateGhostCutover TaskType = "bb.task.database.schema.update.ghost.cutover"
	// TaskDatabaseDataUpdate is the task type for updating database data.
	TaskDatabaseDataUpdate TaskType = "bb.task.database.data.update"
	// TaskDatabaseBackup is the task type for creating database backups.
	TaskDatabaseBackup TaskType = "bb.task.database.backup"
	// TaskDatabaseRestorePITRRestore is the task type for restoring databases using PITR.
	TaskDatabaseRestorePITRRestore TaskType = "bb.task.database.restore.pitr.restore"
	// TaskDatabaseRestorePITRCutover is the task type for swapping the pitr and original database.
	TaskDatabaseRestorePITRCutover TaskType = "bb.task.database.restore.pitr.cutover"
)

// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention

// TaskDatabasePITRRestorePayload is the task payload for database PITR restore.
type TaskDatabasePITRRestorePayload struct {
	// The project owning the database.
	ProjectID int `json:"projectId,omitempty"`

	// DatabaseName is the target database name.
	// It is nil for the case of in-place PITR.
	DatabaseName *string `json:"databaseName,omitempty"`

	// TargetInstanceId must be within the same environment as the instance of the original database.
	// Only used when doing PITR to a new database now.
	TargetInstanceID *int `json:"targetInstanceId,omitempty"`

	// BackupID and PointInTimeTs only allow one non-nil.

	// Only used when doing restore full backup only.
	BackupID *int `json:"backupId,omitempty"`

	// After the PITR operations, the database will be recovered to the state at this time.
	// Represented in UNIX timestamp in seconds.
	PointInTimeTs *int64 `json:"pointInTimeTs,omitempty"`
}

// TaskDatabasePITRCutoverPayload is the task payload for PITR cutover.
// It is currently only a placeholder.
type TaskDatabasePITRCutoverPayload struct{}

// TaskDatabaseCreatePayload is the task payload for creating databases.
type TaskDatabaseCreatePayload struct {
	// The project owning the database.
	ProjectID     int    `json:"projectId,omitempty"`
	DatabaseName  string `json:"databaseName,omitempty"`
	Statement     string `json:"statement,omitempty"`
	CharacterSet  string `json:"character,omitempty"`
	Collation     string `json:"collation,omitempty"`
	Labels        string `json:"labels,omitempty"`
	SchemaVersion string `json:"schemaVersion,omitempty"`
}

// TaskDatabaseSchemaBaselinePayload is the task payload for database schema baseline.
type TaskDatabaseSchemaBaselinePayload struct {
	Statement     string `json:"statement,omitempty"`
	SchemaVersion string `json:"schemaVersion,omitempty"`
	// TODO(d): remove this vcs pushevent since it should not be passed in from frontend.
	VCSPushEvent *vcs.PushEvent `json:"pushEvent,omitempty"`
}

// TaskDatabaseSchemaUpdatePayload is the task payload for database schema update (DDL).
type TaskDatabaseSchemaUpdatePayload struct {
	Statement     string         `json:"statement,omitempty"`
	SchemaVersion string         `json:"schemaVersion,omitempty"`
	VCSPushEvent  *vcs.PushEvent `json:"pushEvent,omitempty"`
}

// TaskDatabaseSchemaUpdateSDLPayload is the task payload for database schema update (SDL).
type TaskDatabaseSchemaUpdateSDLPayload struct {
	Statement     string         `json:"statement,omitempty"`
	SchemaVersion string         `json:"schemaVersion,omitempty"`
	VCSPushEvent  *vcs.PushEvent `json:"pushEvent,omitempty"`
}

// TaskDatabaseSchemaUpdateGhostSyncPayload is the task payload for gh-ost syncing ghost table.
type TaskDatabaseSchemaUpdateGhostSyncPayload struct {
	Statement     string         `json:"statement,omitempty"`
	SchemaVersion string         `json:"schemaVersion,omitempty"`
	VCSPushEvent  *vcs.PushEvent `json:"pushEvent,omitempty"`
	// SocketFileName is the socket file that gh-ost listens on.
	// The name follows this template,
	// `./tmp/gh-ost.{{ISSUE_ID}}.{{TASK_ID}}.{{DATABASE_ID}}.{{DATABASE_NAME}}.{{TABLE_NAME}}.sock`
	// SocketFileName will be composed when needed. We don't store it explicitly.
}

// TaskDatabaseSchemaUpdateGhostCutoverPayload is the task payload for gh-ost switching the original table and the ghost table.
type TaskDatabaseSchemaUpdateGhostCutoverPayload struct{}

// TaskDatabaseDataUpdatePayload is the task payload for database data update (DML).
type TaskDatabaseDataUpdatePayload struct {
	Statement     string         `json:"statement,omitempty"`
	SchemaVersion string         `json:"schemaVersion,omitempty"`
	VCSPushEvent  *vcs.PushEvent `json:"pushEvent,omitempty"`

	// MySQL rollback SQL related.

	// ThreadID is the ID of the connection executing the migration.
	// We use it to filter the binlog events of the migration transaction.
	ThreadID string `json:"threadID,omitempty"`
	// MigrationID is the ID of the migration history record.
	// We use it to get the schema when the transaction ran.
	MigrationID int `json:"migrationID,omitempty"`
	// BinlogXxx are obtained before and after executing the migration.
	// We use them to locate the range of binlog for the migration transaction.
	BinlogFileStart string `json:"binlogFileStart,omitempty"`
	BinlogFileEnd   string `json:"binlogFileEnd,omitempty"`
	BinlogPosStart  int64  `json:"binlogPosStart,omitempty"`
	BinlogPosEnd    int64  `json:"binlogPosEnd,omitempty"`
	RollbackError   string `json:"rollbackError,omitempty"`
	// RollbackStatement is the generated rollback SQL statement for the DML task.
	RollbackStatement string `json:"rollbackStatement,omitempty"`
}

// TaskDatabaseBackupPayload is the task payload for database backup.
type TaskDatabaseBackupPayload struct {
	BackupID int `json:"backupId,omitempty"`
}

// TaskRollbackRequestPayload is the payload of the DML task rollback API.
// We use []string instead of []int because the latter is not supported by the golang jsonapi library.
type TaskRollbackRequestPayload struct {
	TaskIDList []string `jsonapi:"attr,taskIdList"`
}

// Task is the API message for a task.
type Task struct {
	ID int `jsonapi:"primary,task"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns PipelineID and StageID otherwise would cause circular dependency.
	PipelineID int `jsonapi:"attr,pipelineId"`
	StageID    int `jsonapi:"attr,stageId"`
	InstanceID int
	Instance   *Instance `jsonapi:"relation,instance"`
	// Could be empty for creating database task when the task isn't yet completed successfully.
	DatabaseID       *int
	Database         *Database       `jsonapi:"relation,database"`
	TaskRunList      []*TaskRun      `jsonapi:"relation,taskRun"`
	TaskCheckRunList []*TaskCheckRun `jsonapi:"relation,taskCheckRun"`

	// Domain specific fields
	Name              string     `jsonapi:"attr,name"`
	Status            TaskStatus `jsonapi:"attr,status"`
	Type              TaskType   `jsonapi:"attr,type"`
	Payload           string     `jsonapi:"attr,payload"`
	EarliestAllowedTs int64      `jsonapi:"attr,earliestAllowedTs"`
	// BlockedBy is an array of Task ID.
	// We use string here to workaround jsonapi limitations. https://github.com/google/jsonapi/issues/209
	BlockedBy []string `jsonapi:"attr,blockedBy"`
	// RollbackFrom is the task ID from which the rollback SQL statement is generated for this task.
	RollbackFrom int `jsonapi:"attr,rollbackFrom"`
	// Progress is loaded from the task scheduler in memory, NOT from the database
	Progress Progress `jsonapi:"attr,progress"`
}

// Progress is a generalized struct which can track the progress of a task.
type Progress struct {
	// TotalUnit is the total unit count of the task
	TotalUnit int64 `json:"totalUnit"`
	// CompletedUnit is the finished task units
	CompletedUnit int64 `json:"completedUnit"`
	// CreatedTs is when the task starts
	CreatedTs int64 `json:"createdTs"`
	// UpdatedTs is when the progress gets updated most recently
	UpdatedTs int64 `json:"updatedTs"`
	// Payload is reserved for the future
	// Might be something like {comment:"postponing due to network lag"}
	Payload string `json:"payload"`
}

// TaskCreate is the API message for creating a task.
type TaskCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	PipelineID int
	StageID    int
	InstanceID int `jsonapi:"attr,instanceId"`
	// Tasks such as creating database may not have database.
	DatabaseID *int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name   string     `jsonapi:"attr,name"`
	Status TaskStatus `jsonapi:"attr,status"`
	Type   TaskType   `jsonapi:"attr,type"`
	// Payload is derived from fields below it
	Payload           string
	EarliestAllowedTs int64  `jsonapi:"attr,earliestAllowedTs"`
	Statement         string `jsonapi:"attr,statement"`
	DatabaseName      string `jsonapi:"attr,databaseName"`
	CharacterSet      string `jsonapi:"attr,characterSet"`
	Collation         string `jsonapi:"attr,collation"`
	Labels            string `jsonapi:"attr,labels"`
	BackupID          *int   `jsonapi:"attr,backupId"`
	VCSPushEvent      *vcs.PushEvent
}

// TaskFind is the API message for finding tasks.
type TaskFind struct {
	ID *int

	// Related fields
	PipelineID *int
	StageID    *int
	DatabaseID *int

	// Domain specific fields
	StatusList *[]TaskStatus
	TypeList   *[]TaskType
	// Payload contains JSONB expressions
	// Ref: https://www.postgresql.org/docs/current/functions-json.html
	Payload string
}

func (find *TaskFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// TaskPatch is the API message for patching a task.
type TaskPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	DatabaseID        *int
	Statement         *string `jsonapi:"attr,statement"`
	Payload           *string
	EarliestAllowedTs *int64 `jsonapi:"attr,earliestAllowedTs"`
	RollbackFrom      *int
}

// TaskStatusPatch is the API message for patching a task status.
type TaskStatusPatch struct {
	IDList []int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status  TaskStatus `jsonapi:"attr,status"`
	Code    *common.Code
	Comment *string `jsonapi:"attr,comment"`
	Result  *string
}
