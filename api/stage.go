package api

import (
	"context"
	"encoding/json"
)

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
	TaskList      []TaskCreate `jsonapi:"attr,taskList"`

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
	CreateStage(ctx context.Context, create *StageCreate) (*Stage, error)
	FindStageList(ctx context.Context, find *StageFind) ([]*Stage, error)
	FindStage(ctx context.Context, find *StageFind) (*Stage, error)
}
