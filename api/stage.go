package api

import (
	"context"
	"encoding/json"
)

// StageRaw is the store model for an Stage.
// Fields have exactly the same meanings as Stage.
type StageRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	PipelineID    int
	EnvironmentID int

	// Domain specific fields
	Name string
}

// ToStage creates an instance of Stage based on the StageRaw.
// This is intended to be called when we need to compose an Stage relationship.
func (raw *StageRaw) ToStage() *Stage {
	return &Stage{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		PipelineID:    raw.PipelineID,
		EnvironmentID: raw.EnvironmentID,

		// Domain specific fields
		Name: raw.Name,
	}
}

// Stage is the API message for a stage.
type Stage struct {
	ID int `jsonapi:"primary,stage"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns PipelineID otherwise would cause circular dependency.
	PipelineID    int `jsonapi:"attr,pipelineId"`
	EnvironmentID int
	Environment   *Environment `jsonapi:"relation,environment"`
	TaskList      []*Task      `jsonapi:"relation,task"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

// StageCreate is the API message for creating a stage.
type StageCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	EnvironmentID int `jsonapi:"attr,environmentId"`
	PipelineID    int
	TaskList      []TaskCreate    `jsonapi:"attr,taskList"`
	TaskDAGList   []TaskDAGCreate `jsonapi:"attr,taskDAGList"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

// StageFind is the API message for finding stages.
type StageFind struct {
	ID *int

	// Related fields
	PipelineID *int
}

func (find *StageFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// StageService is the service for stages.
type StageService interface {
	CreateStage(ctx context.Context, create *StageCreate) (*StageRaw, error)
	FindStageList(ctx context.Context, find *StageFind) ([]*StageRaw, error)
	FindStage(ctx context.Context, find *StageFind) (*StageRaw, error)
}
