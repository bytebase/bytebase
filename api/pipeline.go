package api

import (
	"context"
	"encoding/json"
)

// OnboardingPipelineID is the ID for onboarding pipelines.
const OnboardingPipelineID = 101

// PipelineStatus is the status for pipelines.
type PipelineStatus string

const (
	// PipelineOpen is the pipeline status for OPEN.
	PipelineOpen PipelineStatus = "OPEN"
	// PipelineDone is the pipeline status for DONE.
	PipelineDone PipelineStatus = "DONE"
	// PipelineCanceled is the pipeline status for CANCELED.
	PipelineCanceled PipelineStatus = "CANCELED"
)

func (e PipelineStatus) String() string {
	switch e {
	case PipelineOpen:
		return "OPEN"
	case PipelineDone:
		return "DONE"
	case PipelineCanceled:
		return "CANCELED"
	}
	return ""
}

// PipelineRaw is the store model for an Pipeline.
// Fields have exactly the same meanings as Pipeline.
type PipelineRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Name   string
	Status PipelineStatus
}

// ToPipeline creates an instance of Pipeline based on the PipelineRaw.
// This is intended to be called when we need to compose an Pipeline relationship.
func (raw *PipelineRaw) ToPipeline() *Pipeline {
	return &Pipeline{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Domain specific fields
		Name:   raw.Name,
		Status: raw.Status,
	}
}

// Pipeline is the API message for pipelines.
type Pipeline struct {
	ID int `jsonapi:"primary,pipeline"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	StageList []*Stage `jsonapi:"relation,stage"`

	// Domain specific fields
	Name   string         `jsonapi:"attr,name"`
	Status PipelineStatus `jsonapi:"attr,status"`
}

// PipelineCreate is the API message for creating a pipeline.
type PipelineCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	StageList []StageCreate `jsonapi:"attr,stageList"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

// PipelineFind is the API message for finding pipelines.
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

// PipelinePatch is the API message for patching a pipeline.
type PipelinePatch struct {
	ID int `jsonapi:"primary,pipelinePatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status *PipelineStatus `jsonapi:"attr,status"`
}

// PipelineService is the service for pipelines.
type PipelineService interface {
	CreatePipeline(ctx context.Context, create *PipelineCreate) (*PipelineRaw, error)
	FindPipelineList(ctx context.Context, find *PipelineFind) ([]*PipelineRaw, error)
	FindPipeline(ctx context.Context, find *PipelineFind) (*PipelineRaw, error)
	PatchPipeline(ctx context.Context, patch *PipelinePatch) (*PipelineRaw, error)
}
