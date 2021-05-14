package api

import "context"

type StageType string

const (
	StageGeneral              StageType = "bb.stage.general"
	StageDatabaseCreate       StageType = "bb.stage.database.create"
	StageDatabaseGrant        StageType = "bb.stage.database.grant"
	StageDatabaseSchemaUpdate StageType = "bb.stage.database.schema.update"
)

func (e StageType) String() string {
	switch e {
	case StageGeneral:
		return "bb.stage.general"
	case StageDatabaseCreate:
		return "bb.stage.database.create"
	case StageDatabaseGrant:
		return "bb.stage.database.grant"
	case StageDatabaseSchemaUpdate:
		return "bb.stage.database.schema.update"
	}
	return "bb.stage.unknown"
}

type Stage struct {
	ID int `jsonapi:"primary,stage"`

	// Standard fields
	CreatorId   int   `jsonapi:"attr,creatorId"`
	CreatedTs   int64 `jsonapi:"attr,createdTs"`
	UpdaterId   int   `jsonapi:"attr,updaterId"`
	UpdatedTs   int64 `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Related fields
	Environment   *Environment `jsonapi:"relation,environment"`
	EnvironmentId int
	Pipeline      *Pipeline `jsonapi:"relation,pipeline"`
	PipelineId    int
	// TaskList []*Stage `jsonapi:"relation,task"`

	// Domain specific fields
	Name string    `jsonapi:"attr,name"`
	Type StageType `jsonapi:"attr,type"`
}

type StageCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Related fields
	EnvironmentId int `jsonapi:"relation,environmentId"`
	PipelineId    int `jsonapi:"relation,pipelineId"`

	// Domain specific fields
	Name string    `jsonapi:"attr,name"`
	Type StageType `jsonapi:"attr,type"`
}

type StageFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int

	// Related fields
	PipelineId *int
}

type StageService interface {
	CreateStage(ctx context.Context, create *StageCreate) (*Stage, error)
	FindStageList(ctx context.Context, find *StageFind) ([]*Stage, error)
	FindStage(ctx context.Context, find *StageFind) (*Stage, error)
}
