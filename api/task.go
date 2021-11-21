package api

import (
	"context"
	"encoding/json"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
)

// These are special onboarding tasks for demo purpose when bootstraping the workspace.
const ONBOARDING_TASK_ID1 = 101
const ONBOARDING_TASK_ID2 = 102

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

// TaskType is the type of a task.
type TaskType string

const (
	TaskGeneral              TaskType = "bb.task.general"
	TaskDatabaseCreate       TaskType = "bb.task.database.create"
	TaskDatabaseSchemaUpdate TaskType = "bb.task.database.schema.update"
	TaskDatabaseBackup       TaskType = "bb.task.database.backup"
	TaskDatabaseRestore      TaskType = "bb.task.database.restore"
)

// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention
type TaskDatabaseCreatePayload struct {
	// The project owning the database.
	ProjectID    int    `json:"projectID,omitempty"`
	DatabaseName string `json:"databaseName,omitempty"`
	Statement    string `json:"statement,omitempty"`
	CharacterSet string `json:"character,omitempty"`
	Collation    string `json:"collation,omitempty"`
}

// TaskDatabaseSchemaUpdatePayload is the task payload for database schema update.
type TaskDatabaseSchemaUpdatePayload struct {
	MigrationType     db.MigrationType     `json:"migrationType,omitempty"`
	Statement         string               `json:"statement,omitempty"`
	RollbackStatement string               `json:"rollbackStatement,omitempty"`
	VCSPushEvent      *common.VCSPushEvent `json:"pushEvent,omitempty"`
}

// TaskDatabaseBackupPayload is the task payload for database backup.
type TaskDatabaseBackupPayload struct {
	BackupID int `json:"backupID,omitempty"`
}

// TaskDatabaseRestorePayload is the task payload for database restore.
type TaskDatabaseRestorePayload struct {
	// The database name we restore to. When we restore a backup to a new database, we only have the database name
	// and don't have the database id upon constructing the task yet.
	DatabaseName string `json:"databaseName,omitempty"`
	BackupID     int    `json:"backupID,omitempty"`
}

type Task struct {
	ID int `jsonapi:"primary,task"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns PipelineID and StageID otherwise would cause circular dependency.
	PipelineID int `jsonapi:"attr,pipelineID"`
	StageID    int `jsonapi:"attr,stageID"`
	InstanceID int
	Instance   *Instance `jsonapi:"relation,instance"`
	// Tasks like creating database may not have database.
	DatabaseID       *int
	Database         *Database       `jsonapi:"relation,database"`
	TaskRunList      []*TaskRun      `jsonapi:"relation,taskRun"`
	TaskCheckRunList []*TaskCheckRun `jsonapi:"relation,taskCheckRun"`

	// Domain specific fields
	Name    string     `jsonapi:"attr,name"`
	Status  TaskStatus `jsonapi:"attr,status"`
	Type    TaskType   `jsonapi:"attr,type"`
	Payload string     `jsonapi:"attr,payload"`
}

type TaskCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	PipelineID int
	StageID    int
	InstanceID int `jsonapi:"attr,instanceID"`
	// Tasks like creating database may not have database.
	DatabaseID *int `jsonapi:"attr,databaseID"`

	// Domain specific fields
	Name   string     `jsonapi:"attr,name"`
	Status TaskStatus `jsonapi:"attr,status"`
	Type   TaskType   `jsonapi:"attr,type"`
	// Payload is dirived from fields below it
	Payload           string
	Statement         string `jsonapi:"attr,statement"`
	RollbackStatement string `jsonapi:"attr,rollbackStatement"`
	DatabaseName      string `jsonapi:"attr,databaseName"`
	CharacterSet      string `jsonapi:"attr,characterSet"`
	Collation         string `jsonapi:"attr,collation"`
	BackupID          *int   `jsonapi:"attr,backupID"`
	VCSPushEvent      *common.VCSPushEvent
	MigrationType     db.MigrationType `jsonapi:"attr,migrationType"`
}

type TaskFind struct {
	ID *int

	// Related fields
	PipelineID *int
	StageID    *int

	// Domain specific fields
	StatusList *[]TaskStatus
}

func (find *TaskFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type TaskPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Statement *string `jsonapi:"attr,statement"`
	Payload   *string
}

type TaskStatusPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status  TaskStatus `jsonapi:"attr,status"`
	Code    *common.Code
	Comment *string `jsonapi:"attr,comment"`
	Result  *string
}

type TaskService interface {
	CreateTask(ctx context.Context, create *TaskCreate) (*Task, error)
	FindTaskList(ctx context.Context, find *TaskFind) ([]*Task, error)
	FindTask(ctx context.Context, find *TaskFind) (*Task, error)
	PatchTask(ctx context.Context, patch *TaskPatch) (*Task, error)
	PatchTaskStatus(ctx context.Context, patch *TaskStatusPatch) (*Task, error)
}
