package api

// StageStatusUpdateType is the type of the stage status update.
// StageStatusUpdate is a computed event of the contained tasks.
type StageStatusUpdateType string

const (
	// StageStatusUpdateTypeEnd means the stage ends. A stage can end multiple times.
	// A stage ends if its contained tasks have finished running, i.e. the status of which is one of
	//   - DONE
	//   - FAILED
	//   - CANCELED
	StageStatusUpdateTypeEnd StageStatusUpdateType = "END"
)
