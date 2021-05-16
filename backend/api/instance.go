package api

import "context"

type Instance struct {
	ID int `jsonapi:"primary,instance"`

	// Standard fields
	RowStatus   RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId   int       `jsonapi:"attr,creatorId"`
	CreatedTs   int64     `jsonapi:"attr,createdTs"`
	UpdaterId   int       `jsonapi:"attr,updaterId"`
	UpdatedTs   int64     `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Related fields
	EnvironmentId  int
	Environment    *Environment  `jsonapi:"relation,environment"`
	DataSourceList []*DataSource `jsonapi:"relation,dataSource"`

	// Domain specific fields
	Name         string `jsonapi:"attr,name"`
	ExternalLink string `jsonapi:"attr,externalLink"`
	Host         string `jsonapi:"attr,host"`
	Port         string `jsonapi:"attr,port"`
}

type InstanceCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Related fields
	EnvironmentId string `jsonapi:"attr,environmentId"`

	// Domain specific fields
	Name         string `jsonapi:"attr,name"`
	ExternalLink string `jsonapi:"attr,externalLink"`
	Host         string `jsonapi:"attr,host"`
	Port         string `jsonapi:"attr,port"`
	Username     string `jsonapi:"attr,username"`
	Password     string `jsonapi:"attr,password"`
}

type InstanceFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int
}

type InstancePatch struct {
	ID int `jsonapi:"primary,instancePatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Domain specific fields
	Name         *string `jsonapi:"attr,name"`
	ExternalLink *string `jsonapi:"attr,externalLink"`
	Host         *string `jsonapi:"attr,host"`
	Port         *string `jsonapi:"attr,port"`
	Username     *string `jsonapi:"attr,username"`
	Password     *string `jsonapi:"attr,password"`
}

type InstanceService interface {
	CreateInstance(ctx context.Context, create *InstanceCreate) (*Instance, error)
	FindInstanceList(ctx context.Context, find *InstanceFind) ([]*Instance, error)
	FindInstance(ctx context.Context, find *InstanceFind) (*Instance, error)
	PatchInstance(ctx context.Context, patch *InstancePatch) (*Instance, error)
}
