package api

import (
	"context"
	"encoding/json"
)

const ONBOARDING_PIPELINE_ID = 101

// Pipeline status
type PipelineStatus string

const (
	Pipeline_Open     PipelineStatus = "OPEN"
	Pipeline_Done     PipelineStatus = "DONE"
	Pipeline_Canceled PipelineStatus = "CANCELED"
)

func (e PipelineStatus) String() string {
	switch e {
	case Pipeline_Open:
		return "OPEN"
	case Pipeline_Done:
		return "DONE"
	case Pipeline_Canceled:
		return "CANCELED"
	}
	return ""
}

type Pipeline struct {
	ID int `jsonapi:"primary,pipeline"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	StageList []*Stage `jsonapi:"relation,stage"`

	// Domain specific fields
	Name   string         `jsonapi:"attr,name"`
	Status PipelineStatus `jsonapi:"attr,status"`
}

type PipelineCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	StageList []StageCreate `jsonapi:"attr,stageList"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

type PipelineFind struct {
	ID *int

	// Domain specific fields
	Status *PipelineStatus
}

func (find *PipelineFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type PipelinePatch struct {
	ID int `jsonapi:"primary,pipelinePatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status *PipelineStatus `jsonapi:"attr,status"`
}

type PipelineService interface {
	CreatePipeline(ctx context.Context, create *PipelineCreate) (*Pipeline, error)
	FindPipelineList(ctx context.Context, find *PipelineFind) ([]*Pipeline, error)
	FindPipeline(ctx context.Context, find *PipelineFind) (*Pipeline, error)
	PatchPipeline(ctx context.Context, patch *PipelinePatch) (*Pipeline, error)
}
