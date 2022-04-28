package api

import (
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
