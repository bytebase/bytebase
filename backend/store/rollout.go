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
	TaskList         []TaskMessage
	TaskIndexDAGList []api.TaskIndexDAG
}
