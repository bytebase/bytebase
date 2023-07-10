package store

import api "github.com/bytebase/bytebase/backend/legacyapi"

// Rollout is the API message for creating a pipeline.
type Rollout struct {
	Name      string
	StageList []RolloutStage
}

// RolloutStage is the API message for a rollout stage.
type RolloutStage struct {
	Name             string
	EnvironmentID    int
	PipelineID       int
	TaskList         []RolloutTask
	TaskIndexDAGList []api.TaskIndexDAG
}

// RolloutTask is the API message for a rollout task.
type RolloutTask struct {
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
