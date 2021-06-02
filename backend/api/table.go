package api

import (
	"context"
	"encoding/json"
)

type Table struct {
	ID int `jsonapi:"primary,table"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	DatabaseId int
	Database   *Database `jsonapi:"relation,database"`

	// Domain specific fields
	Name                 string     `jsonapi:"attr,name"`
	Engine               string     `jsonapi:"attr,engine"`
	Collation            string     `jsonapi:"attr,collation"`
	RowCount             int64      `jsonapi:"attr,rowCount"`
	DataSize             int64      `jsonapi:"attr,dataSize"`
	IndexSize            int64      `jsonapi:"attr,indexSize"`
	SyncStatus           SyncStatus `jsonapi:"attr,syncStatus"`
	LastSuccessfulSyncTs int64      `jsonapi:"attr,lastSuccessfulSyncTs"`
}

type TableCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	DatabaseId int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name      string `jsonapi:"attr,name"`
	Engine    string `jsonapi:"attr,engine"`
	Collation string `jsonapi:"attr,collation"`
	RowCount  int64
	DataSize  int64
	IndexSize int64
}

type TableFind struct {
	ID *int

	// Related fields
	DatabaseId *int

	// Domain specific fields
	Name *string
}

func (find *TableFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type TablePatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	SyncStatus           *SyncStatus
	LastSuccessfulSyncTs *int64
}

type TableService interface {
	CreateTable(ctx context.Context, create *TableCreate) (*Table, error)
	FindTableList(ctx context.Context, find *TableFind) ([]*Table, error)
	FindTable(ctx context.Context, find *TableFind) (*Table, error)
	PatchTable(ctx context.Context, patch *TablePatch) (*Table, error)
}
