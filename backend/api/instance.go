package api

import "context"

type Instance struct {
	ID int `jsonapi:"primary,instance"`

	// Related fields
	EnvironmentId  int
	Environment    *Environment  `jsonapi:"relation,environment"`
	DataSourceList []*DataSource `jsonapi:"relation,dataSource"`

	// Standard fields
	RowStatus   RowStatus `jsonapi:"attr,rowStatus"`
	WorkspaceId int
	CreatorId   int   `jsonapi:"attr,creatorId"`
	CreatedTs   int64 `jsonapi:"attr,createdTs"`
	UpdaterId   int   `jsonapi:"attr,updaterId"`
	UpdatedTs   int64 `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name         string `jsonapi:"attr,name"`
	ExternalLink string `jsonapi:"attr,externalLink"`
	Host         string `jsonapi:"attr,host"`
	Port         string `jsonapi:"attr,port"`
	// Username     string `jsonapi:"attr,username"`
	// Password     string `jsonapi:"attr,password"`
}

type InstanceCreate struct {
	// Related fields
	EnvironmentId string `jsonapi:"attr,environmentId"`

	// Standard fields
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Name         string `jsonapi:"attr,name"`
	ExternalLink string `jsonapi:"attr,externalLink"`
	Host         string `jsonapi:"attr,host"`
	Port         string `jsonapi:"attr,port"`
	Username     string `jsonapi:"attr,username"`
	Password     string `jsonapi:"attr,password"`
}

type InstanceFind struct {
	// Standard fields
	ID          *int
	WorkspaceId *int
}

type InstancePatch struct {
	// Standard fields
	ID          int     `jsonapi:"primary,instancePatch"`
	RowStatus   *string `jsonapi:"attr,rowStatus"`
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

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
	PatchInstanceByID(ctx context.Context, patch *InstancePatch) (*Instance, error)
}
