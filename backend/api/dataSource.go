package api

import (
	"context"
	"encoding/json"
)

type DataSourceType string

const (
	Admin DataSourceType = "ADMIN"
	RW    DataSourceType = "RW"
	RO    DataSourceType = "RO"
)

func (e DataSourceType) String() string {
	switch e {
	case Admin:
		return "ADMIN"
	case RW:
		return "RW"
	case RO:
		return "RO"
	}
	return ""
}

type DataSource struct {
	ID int `jsonapi:"primary,dataSource"`

	// Standard fields
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Related fields
	// Just returns InstanceId and DatabaseId otherwise would cause circular dependency.
	InstanceId int `jsonapi:"attr,instanceId"`
	DatabaseId int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name     string         `jsonapi:"attr,name"`
	Type     DataSourceType `jsonapi:"attr,type"`
	Username string         `jsonapi:"attr,username"`
	Password string         `jsonapi:"attr,password"`
}

type DataSourceCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Related fields
	InstanceId int
	DatabaseId int

	// Domain specific fields
	Name     string         `jsonapi:"attr,name"`
	Type     DataSourceType `jsonapi:"attr,type"`
	Username string         `jsonapi:"attr,username"`
	Password string         `jsonapi:"attr,password"`
}

type DataSourceFind struct {
	// Standard fields
	WorkspaceId *int

	// Related fields
	InstanceId *int
	DatabaseId *int

	// Domain specific fields
	Type *DataSourceType
}

func (find *DataSourceFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type DataSourcePatch struct {
	ID int `jsonapi:"primary,dataSourcePatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Domain specific fields
	Username *string `jsonapi:"attr,username"`
	Password *string `jsonapi:"attr,password"`
}

type DataSourceService interface {
	CreateDataSource(ctx context.Context, create *DataSourceCreate) (*DataSource, error)
	FindDataSourceList(ctx context.Context, find *DataSourceFind) ([]*DataSource, error)
	FindDataSource(ctx context.Context, find *DataSourceFind) (*DataSource, error)
	PatchDataSource(ctx context.Context, patch *DataSourcePatch) (*DataSource, error)
}
