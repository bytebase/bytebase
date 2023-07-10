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
	InstanceID int
	// Tasks such as creating database may not have database.
	DatabaseID *int

	// Domain specific fields
	Name   string
	Status api.TaskStatus
	Type   api.TaskType
	// Payload is derived from fields below it
	Payload           string
	EarliestAllowedTs int64
	DatabaseName      string
	// Statement used by grouping batch change, Bytebase use it to render.
	Statement string
}
