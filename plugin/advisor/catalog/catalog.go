package catalog

import (
	"context"
	"sort"
)

// Catalog is the service for catalog.
type Catalog interface {
	GetDatabase(ctx context.Context) (*Database, error)
}

// Database is the database.
type Database struct {
	Name         string
	CharacterSet string
	Collation    string
	SchemaList   []*Schema
}

// Empty returns true if the current database has no table.
func (d *Database) Empty() bool {
	for _, schema := range d.SchemaList {
		for _, table := range schema.TableList {
			if table != nil {
				return false
			}
		}
	}

	return true
}

// IndexFind is for find index.
type IndexFind struct {
	SchemaName string
	TableName  string
	IndexName  string
}

// FindIndex finds the index.
func (d *Database) FindIndex(find *IndexFind) []*Index {
	var indexList []*Index
	for _, schema := range d.SchemaList {
		if schema.Name != find.SchemaName {
			continue
		}
		for _, table := range schema.TableList {
			if table.Name != find.TableName {
				continue
			}
			for _, index := range table.IndexList {
				if index.Name == find.IndexName {
					indexList = append(indexList, index)
				}
			}
		}
	}
	sort.Slice(indexList, func(i, j int) bool {
		return indexList[i].Position < indexList[j].Position
	})
	return indexList
}

// Schema is the database schema.
type Schema struct {
	Name          string
	TableList     []*Table
	ViewList      []*View
	ExtensionList []*Extension
}

// Table is the database table.
type Table struct {
	Name string
	// CreatedTs isn't supported for ClickHouse, SQLite.
	CreatedTs int64
	// UpdatedTs isn't supported for SQLite.
	UpdatedTs int64
	Type      string
	// Engine isn't supported for Postgres, Snowflake, SQLite.
	Engine string
	// Collation isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	Collation string
	RowCount  int64
	// DataSize isn't supported for SQLite.
	DataSize int64
	// IndexSize isn't supported for ClickHouse, Snowflake, SQLite.
	IndexSize int64
	// DataFree isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	DataFree int64
	// CreateOptions isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	CreateOptions string
	// Comment isn't supported for SQLite.
	Comment    string
	ColumnList []*Column
	// IndexList isn't supported for ClickHouse, Snowflake.
	IndexList []*Index
}

// Index is the database index.
type Index struct {
	Name string
	// This could refer to a column or an expression.
	Expression string
	Position   int
	// Type isn't supported for SQLite.
	Type    string
	Unique  bool
	Primary bool
	// Visible isn't supported for Postgres, SQLite.
	Visible bool
	// Comment isn't supported for SQLite.
	Comment string
}

// Column the database table column.
type Column struct {
	Name     string
	Position int
	Default  *string
	// Nullable isn't supported for ClickHouse.
	Nullable bool
	Type     string
	// CharacterSet isn't supported for Postgres, ClickHouse, SQLite.
	CharacterSet string
	// Collation isn't supported for ClickHouse, SQLite.
	Collation string
	// Comment isn't supported for SQLite.
	Comment string
}

// View is the database view.
type View struct {
	Name string
	// CreatedTs isn't supported for ClickHouse.
	CreatedTs  int64
	UpdatedTs  int64
	Definition string
	Comment    string
}

// Extension is the database extension.
type Extension struct {
	Name        string
	Version     string
	Description string
}
