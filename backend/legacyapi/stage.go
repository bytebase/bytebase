package api

// StageStatusUpdateType is the type of the stage status update.
// StageStatusUpdate is a computed event of the contained tasks.
type StageStatusUpdateType string

const (
	// StageStatusUpdateTypeBegin means the stage begins. A stage only begins once.
	StageStatusUpdateTypeBegin StageStatusUpdateType = "BEGIN"
	// StageStatusUpdateTypeEnd means the stage ends. A stage can end multiple times.
	// A stage ends if its contained tasks have finished running, i.e. the status of which is one of
	//   - DONE
	//   - FAILED
	//   - CANCELED
	StageStatusUpdateTypeEnd StageStatusUpdateType = "END"
)

// Stage is the API message for a stage.
type Stage struct {
	ID int `jsonapi:"primary,stage"`

	// Related fields
	// Just returns PipelineID otherwise would cause circular dependency.
	PipelineID    int `jsonapi:"attr,pipelineId"`
	EnvironmentID int
	Environment   *Environment `jsonapi:"relation,environment"`
	TaskList      []*Task      `jsonapi:"relation,task"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

// StageAllTaskStatusPatch is the API message for patching task status for all tasks in a stage.
type StageAllTaskStatusPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status  TaskStatus `jsonapi:"attr,status"`
	Comment *string    `jsonapi:"attr,comment"`
}
