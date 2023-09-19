package api

// Pipeline is the API message for pipelines.
type Pipeline struct {
	ID int `jsonapi:"primary,pipeline"`

	// Related fields
	StageList []*Stage `jsonapi:"relation,stage"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}
