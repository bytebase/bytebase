package api

import (
	"context"
	"encoding/json"
)

type Stage struct {
	ID int `jsonapi:"primary,stage"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns PipelineId otherwise would cause circular dependency.
	PipelineId    int `jsonapi:"attr,pipelineId"`
	EnvironmentId int
	Environment   *Environment `jsonapi:"relation,environment"`
	TaskList      []*Task      `jsonapi:"relation,task"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

type StageCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	EnvironmentId int `jsonapi:"attr,environmentId"`
	PipelineId    int
	TaskList      []TaskCreate `jsonapi:"attr,taskList"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

type StageFind struct {
	ID *int

	// Related fields
	PipelineId *int
}

func (find *StageFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type StageService interface {
	CreateStage(ctx context.Context, create *StageCreate) (*Stage, error)
	FindStageList(ctx context.Context, find *StageFind) ([]*Stage, error)
	FindStage(ctx context.Context, find *StageFind) (*Stage, error)
}
