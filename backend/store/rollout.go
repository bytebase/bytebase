package store

import api "github.com/bytebase/bytebase/backend/legacyapi"

// PipelineCreate is the API message for creating a pipeline.
type PipelineCreate struct {
	StageList []StageCreate
	Name      string
}

// StageCreate is the API message for creating a stage.
type StageCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	EnvironmentID    int
	PipelineID       int
	TaskList         []TaskCreate
	TaskIndexDAGList []api.TaskIndexDAG

	// Domain specific fields
	Name string
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
	Name   string         `jsonapi:"attr,name"`
	Status api.TaskStatus `jsonapi:"attr,status"`
	Type   api.TaskType   `jsonapi:"attr,type"`
	// Payload is derived from fields below it
	Payload           string
	EarliestAllowedTs int64  `jsonapi:"attr,earliestAllowedTs"`
	DatabaseName      string `jsonapi:"attr,databaseName"`
	CharacterSet      string `jsonapi:"attr,characterSet"`
	Collation         string `jsonapi:"attr,collation"`
	Labels            string `jsonapi:"attr,labels"`
	BackupID          *int   `jsonapi:"attr,backupId"`
	// Statement used by grouping batch change, Bytebase use it to render.
	Statement string `jsonapi:"attr,statement"`
}
