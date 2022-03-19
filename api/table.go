package api

import (
	"context"
	"encoding/json"
)

// TableRaw is the store model for an Table.
// Fields have exactly the same meanings as Table.
type TableRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Name          string
	Type          string
	Engine        string
	Collation     string
	RowCount      int64
	DataSize      int64
	IndexSize     int64
	DataFree      int64
	CreateOptions string
	Comment       string
}

// ToTable creates an instance of Table based on the TableRaw.
// This is intended to be called when we need to compose an Table relationship.
func (raw *TableRaw) ToTable() *Table {
	return &Table{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:          raw.Name,
		Type:          raw.Type,
		Engine:        raw.Engine,
		Collation:     raw.Collation,
		RowCount:      raw.RowCount,
		DataSize:      raw.DataSize,
		IndexSize:     raw.IndexSize,
		DataFree:      raw.DataFree,
		CreateOptions: raw.CreateOptions,
		Comment:       raw.Comment,
	}
}

// Table is the API message for a table.
type Table struct {
	ID int `jsonapi:"primary,table"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// TODO(dragonly): seems like not using this field?
	DatabaseID int
	Database   *Database `jsonapi:"relation,database"`

	// Domain specific fields
	Name          string    `jsonapi:"attr,name"`
	Type          string    `jsonapi:"attr,type"`
	Engine        string    `jsonapi:"attr,engine"`
	Collation     string    `jsonapi:"attr,collation"`
	RowCount      int64     `jsonapi:"attr,rowCount"`
	DataSize      int64     `jsonapi:"attr,dataSize"`
	IndexSize     int64     `jsonapi:"attr,indexSize"`
	DataFree      int64     `jsonapi:"attr,dataFree"`
	CreateOptions string    `jsonapi:"attr,createOptions"`
	Comment       string    `jsonapi:"attr,comment"`
	ColumnList    []*Column `jsonapi:"attr,columnList"`
	IndexList     []*Index  `jsonapi:"attr,indexList"`
}

// TableCreate is the API message for creating a table.
type TableCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int
	CreatedTs int64
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Name          string
	Type          string
	Engine        string
	Collation     string
	RowCount      int64
	DataSize      int64
	IndexSize     int64
	DataFree      int64
	CreateOptions string
	Comment       string
}

// TableFind is the API message for finding tables.
type TableFind struct {
	ID *int

	// Related fields
	DatabaseID *int

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

// TableDelete is the API message for deleting a table.
type TableDelete struct {
	// Related fields
	DatabaseID int
}

// TableService is the service for tables.
type TableService interface {
	CreateTable(ctx context.Context, create *TableCreate) (*TableRaw, error)
	FindTableList(ctx context.Context, find *TableFind) ([]*TableRaw, error)
	FindTable(ctx context.Context, find *TableFind) (*TableRaw, error)
	DeleteTable(ctx context.Context, delete *TableDelete) error
}
