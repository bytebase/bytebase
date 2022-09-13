package catalog

// This file defines the database state during walk-through. The only two cases will use database state:
//   1. during walk-through
//   2. the underlying implementation of Finder

import (
	"github.com/bytebase/bytebase/plugin/advisor/db"
)

func newDatabaseState(d *Database, context *FinderContext) *databaseState {
	database := &databaseState{
		name:         d.Name,
		characterSet: d.CharacterSet,
		collation:    d.Collation,
		dbType:       d.DbType,
		schemaSet:    make(schemaStateMap),
		context:      context.Copy(),
	}

	for _, schema := range d.SchemaList {
		database.schemaSet[schema.Name] = newSchemaState(schema, context)
	}

	return database
}

func newSchemaState(s *Schema, context *FinderContext) *schemaState {
	schema := &schemaState{
		name:         s.Name,
		tableSet:     make(tableStateMap),
		viewSet:      make(viewStateMap),
		extensionSet: make(extensionStateMap),
		context:      context.Copy(),
	}

	for _, table := range s.TableList {
		schema.tableSet[table.Name] = newTableState(table, context)
	}

	for _, view := range s.ViewList {
		schema.viewSet[view.Name] = newViewState(view)
	}

	for _, extension := range s.ExtensionList {
		schema.extensionSet[extension.Name] = newExtensionState(extension)
	}

	return schema
}

func newViewState(v *View) *viewState {
	return &viewState{
		name:       v.Name,
		definition: v.Definition,
		comment:    v.Comment,
	}
}

func newExtensionState(e *Extension) *extensionState {
	return &extensionState{
		name:        e.Name,
		version:     e.Version,
		description: e.Description,
	}
}

func newTableState(t *Table, context *FinderContext) *tableState {
	table := &tableState{
		name:          t.Name,
		tableType:     t.Type,
		engine:        t.Engine,
		collation:     t.Collation,
		rowCount:      t.RowCount,
		dataSize:      t.DataSize,
		indexSize:     t.IndexSize,
		dataFree:      t.DataFree,
		createOptions: t.CreateOptions,
		comment:       t.Comment,
		columnSet:     make(columnStateMap),
		indexSet:      make(indexStateMap),
		context:       context.Copy(),
	}

	for _, column := range t.ColumnList {
		table.columnSet[column.Name] = newColumnState(column)
	}

	for _, index := range t.IndexList {
		table.indexSet[index.Name] = newIndexState(index)
	}

	return table
}

func newColumnState(c *Column) *columnState {
	return &columnState{
		name:         c.Name,
		position:     c.Position,
		defaultValue: c.Default,
		nullable:     c.Nullable,
		columnType:   c.Type,
		characterSet: c.CharacterSet,
		collation:    c.Collation,
		comment:      c.Comment,
	}
}

func newIndexState(i *Index) *indexState {
	index := &indexState{
		name:      i.Name,
		indextype: i.Type,
		unique:    i.Unique,
		primary:   i.Primary,
		visible:   i.Visible,
		comment:   i.Comment,
	}
	index.expressionList = append(index.expressionList, i.ExpressionList...)
	return index
}

type databaseState struct {
	name         string
	characterSet string
	collation    string
	dbType       db.Type
	schemaSet    schemaStateMap

	context *FinderContext
}
type schemaState struct {
	name         string
	tableSet     tableStateMap
	viewSet      viewStateMap
	extensionSet extensionStateMap

	context *FinderContext
}
type schemaStateMap map[string]*schemaState

type tableState struct {
	name      string
	tableType string
	// engine isn't supported for Postgres, Snowflake, SQLite.
	engine string
	// collation isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	collation string
	rowCount  int64
	// dataSize isn't supported for SQLite.
	dataSize int64
	// indexSize isn't supported for ClickHouse, Snowflake, SQLite.
	indexSize int64
	// dataFree isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	dataFree int64
	// createOptions isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	createOptions string
	// comment isn't supported for SQLite.
	comment   string
	columnSet columnStateMap
	// indexSet isn't supported for ClickHouse, Snowflake.
	indexSet indexStateMap

	context *FinderContext
}
type tableStateMap map[string]*tableState

func (table *tableState) convertToCatalog() *Table {
	return &Table{
		Name:          table.name,
		Type:          table.tableType,
		Engine:        table.engine,
		Collation:     table.collation,
		RowCount:      table.rowCount,
		DataSize:      table.dataSize,
		IndexSize:     table.indexSize,
		DataFree:      table.dataFree,
		CreateOptions: table.createOptions,
		Comment:       table.comment,
		ColumnList:    table.columnSet.convertToCatalog(),
		IndexList:     table.indexSet.convertToCatalog(),
	}
}

type indexState struct {
	name string
	// This could refer to a column or an expression.
	expressionList []string
	// Type isn't supported for SQLite.
	indextype string
	unique    bool
	primary   bool
	// Visible isn't supported for Postgres, SQLite.
	visible bool
	// Comment isn't supported for SQLite.
	comment string
}
type indexStateMap map[string]*indexState

func (m indexStateMap) convertToCatalog() []*Index {
	var res []*Index
	for _, index := range m {
		res = append(res, index.convertToCatalog())
	}
	return res
}

func (index *indexState) convertToCatalog() *Index {
	return &Index{
		Name:           index.name,
		ExpressionList: index.expressionList,
		Type:           index.indextype,
		Unique:         index.unique,
		Primary:        index.primary,
		Visible:        index.visible,
		Comment:        index.comment,
	}
}

type columnState struct {
	name         string
	position     int
	defaultValue *string
	// nullable isn't supported for ClickHouse.
	nullable   bool
	columnType string
	// characterSet isn't supported for Postgres, ClickHouse, SQLite.
	characterSet string
	// collation isn't supported for ClickHouse, SQLite.
	collation string
	// comment isn't supported for SQLite.
	comment string
}
type columnStateMap map[string]*columnState

func (m columnStateMap) convertToCatalog() []*Column {
	var res []*Column
	for _, column := range m {
		res = append(res, column.convertToCatalog())
	}
	return res
}

func (column *columnState) convertToCatalog() *Column {
	return &Column{
		Name:         column.name,
		Position:     column.position,
		Default:      column.defaultValue,
		Nullable:     column.nullable,
		Type:         column.columnType,
		CharacterSet: column.characterSet,
		Collation:    column.collation,
		Comment:      column.comment,
	}
}

type viewState struct {
	name       string
	definition string
	comment    string
}
type viewStateMap map[string]*viewState

type extensionState struct {
	name        string
	version     string
	description string
}
type extensionStateMap map[string]*extensionState
