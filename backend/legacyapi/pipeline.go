package api

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

// Pipeline is the API message for pipelines.
type Pipeline struct {
	ID int `jsonapi:"primary,pipeline"`

	// Related fields
	StageList []*Stage `jsonapi:"relation,stage"`

	// Domain specific fields
	Name   string `jsonapi:"attr,name"`
	Status PipelineStatus
}

// PipelineCreate is the API message for creating a pipeline.
type PipelineCreate struct {
	StageList []StageCreate `jsonapi:"attr,stageList"`
	Name      string        `jsonapi:"attr,name"`
}
