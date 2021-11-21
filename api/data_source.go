package api

import (
	"context"
	"database/sql"
	"encoding/json"
)

const (
	ADMIN_DATA_SOURCE_NAME = "Admin data source"
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
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns InstanceID and DatabaseID otherwise would cause circular dependency.
	InstanceID int `jsonapi:"attr,instanceID"`
	DatabaseID int `jsonapi:"attr,databaseID"`

	// Domain specific fields
	Name     string         `jsonapi:"attr,name"`
	Type     DataSourceType `jsonapi:"attr,type"`
	Username string         `jsonapi:"attr,username"`
	Password string         `jsonapi:"attr,password"`
}

type DataSourceCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	InstanceID int
	DatabaseID int

	// Domain specific fields
	Name     string         `jsonapi:"attr,name"`
	Type     DataSourceType `jsonapi:"attr,type"`
	Username string         `jsonapi:"attr,username"`
	Password string         `jsonapi:"attr,password"`
}

type DataSourceFind struct {
	// Related fields
	InstanceID *int
	DatabaseID *int

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
	UpdaterID int

	// Domain specific fields
	Username *string `jsonapi:"attr,username"`
	Password *string `jsonapi:"attr,password"`
}

type DataSourceService interface {
	CreateDataSource(ctx context.Context, create *DataSourceCreate) (*DataSource, error)
	// This is specifically used to create the admin data source when creating the instance.
	CreateDataSourceTx(ctx context.Context, tx *sql.Tx, create *DataSourceCreate) (*DataSource, error)
	FindDataSourceList(ctx context.Context, find *DataSourceFind) ([]*DataSource, error)
	FindDataSource(ctx context.Context, find *DataSourceFind) (*DataSource, error)
	PatchDataSource(ctx context.Context, patch *DataSourcePatch) (*DataSource, error)
}
