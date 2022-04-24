package api

// TaskDAG describes task dependency relationship.
// FromTaskID blocks ToTaskID
// It's rather DAGEdge than DAG
type TaskDAG struct {
	ID int

	// Standard fields
	CreatedTs int64
	UpdatedTs int64

	// Domain Specific fields
	FromTaskID int
	ToTaskID   int
	Payload    string
}

// TaskDAGCreate is the API message to create TaskDAG.
type TaskDAGCreate struct {
	// Domain Specific fields
	FromTaskID int
	ToTaskID   int
	Payload    string
}

// TaskDAGFind is the API message to find TaskDAG.
type TaskDAGFind struct {
	ToTaskID int
}

// TaskIndexDAG describes task dependency relationship using array index to represent task.
// It is needed because we don't know task id before insertion.
type TaskIndexDAG struct {
	FromIndex int
	ToIndex   int
}
