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
		schema.tableSet[table.Name] = newTableState(table)
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

func newTableState(t *Table) *tableState {
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
	deleted      bool

	context *FinderContext
}

// HasNoTable returns true if the current database has no table.
func (d *databaseState) HasNoTable() bool {
	for _, schema := range d.schemaSet {
		if len(schema.tableSet) != 0 {
			return false
		}
	}

	return true
}

// DatabaseName returns the database name.
func (d *databaseState) DatabaseName() string {
	return d.name
}

// IndexFind is for find index.
type IndexFind struct {
	SchemaName string
	TableName  string
	IndexName  string
}

func (d *databaseState) FindIndex(find *IndexFind) (string, *Index) {
	// There are two cases to find a index:
	// 1. find an index in specific table. e.g. MySQL and TiDB.
	// 2. find an index in the schema. e.g. PostgreSQL.
	// In PostgreSQL, the index name is unique in a schema, not a table.
	// In MySQL and TiDB, the index name is unique in a table.
	// So for case one, we need match table name, but for case two, we don't need.
	needMatchTable := (d.dbType != db.Postgres || find.SchemaName == "" || find.TableName != "")
	if needMatchTable {
		schema, exists := d.schemaSet[find.SchemaName]
		if !exists {
			return "", nil
		}
		table, exists := schema.tableSet[find.TableName]
		if !exists {
			return "", nil
		}
		index, exists := table.indexSet[find.IndexName]
		if !exists {
			return "", nil
		}
		return table.name, index.convertToCatalog()
	}
	for _, schema := range d.schemaSet {
		if schema.name != find.SchemaName {
			continue
		}
		for _, table := range schema.tableSet {
			// no need to further match table name because index is already unique in the schema
			index, exists := table.indexSet[find.IndexName]
			if !exists {
				return "", nil
			}
			return table.name, index.convertToCatalog()
		}
	}
	return "", nil
}

// PrimaryKeyFind is for finding primary key.
type PrimaryKeyFind struct {
	SchemaName string
	TableName  string
}

// FindPrimaryKey finds the primary key.
func (d *databaseState) FindPrimaryKey(find *PrimaryKeyFind) *Index {
	for _, schema := range d.schemaSet {
		if schema.name != find.SchemaName {
			continue
		}
		for _, table := range schema.tableSet {
			if table.name != find.TableName {
				continue
			}
			for _, index := range table.indexSet {
				if index.primary {
					return index.convertToCatalog()
				}
			}
		}
	}
	return nil
}

// ColumnFind is for finding column.
type ColumnFind struct {
	SchemaName string
	TableName  string
	ColumnName string
}

// FindColumn finds the column.
func (d *databaseState) FindColumn(find *ColumnFind) *Column {
	schema, exists := d.schemaSet[find.SchemaName]
	if !exists {
		return nil
	}
	table, exists := schema.tableSet[find.TableName]
	if !exists {
		return nil
	}
	column, exists := table.columnSet[find.ColumnName]
	if !exists {
		return nil
	}
	return column.convertToCatalog()
}

// TableFind is for find table.
type TableFind struct {
	SchemaName string
	TableName  string
}

// FindTable finds the table.
func (d *databaseState) FindTable(find *TableFind) *Table {
	schema, exists := d.schemaSet[find.SchemaName]
	if !exists {
		return nil
	}
	table, exists := schema.tableSet[find.TableName]
	if !exists {
		return nil
	}
	return table.convertToCatalog()
}

func (d *databaseState) copy() *databaseState {
	newDb := newDatabaseState(d.convertToCatalog(), &FinderContext{CheckIntegrity: d.context.CheckIntegrity})
	newDb.deleted = d.deleted
	return newDb
}

func (d *databaseState) convertToCatalog() *Database {
	return &Database{
		Name:         d.name,
		CharacterSet: d.characterSet,
		Collation:    d.collation,
		DbType:       d.dbType,
		SchemaList:   d.schemaSet.convertToCatalog(),
	}
}

type schemaState struct {
	name         string
	tableSet     tableStateMap
	viewSet      viewStateMap
	extensionSet extensionStateMap

	context *FinderContext
}
type schemaStateMap map[string]*schemaState

func (m schemaStateMap) convertToCatalog() []*Schema {
	var res []*Schema
	for _, schema := range m {
		res = append(res, schema.convertToCatalog())
	}
	return res
}

func (schema *schemaState) convertToCatalog() *Schema {
	return &Schema{
		Name:          schema.name,
		TableList:     schema.tableSet.convertToCatalog(),
		ViewList:      schema.viewSet.convertToCatalog(),
		ExtensionList: schema.extensionSet.convertToCatalog(),
	}
}

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
}
type tableStateMap map[string]*tableState

func (m tableStateMap) convertToCatalog() []*Table {
	var res []*Table
	for _, table := range m {
		res = append(res, table.convertToCatalog())
	}
	return res
}

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

func (table *tableState) copy() *tableState {
	return newTableState(table.convertToCatalog())
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

func (m viewStateMap) convertToCatalog() []*View {
	var res []*View
	for _, view := range m {
		res = append(res, view.convertToCatalog())
	}
	return res
}

func (view *viewState) convertToCatalog() *View {
	return &View{
		Name:       view.name,
		Definition: view.definition,
		Comment:    view.comment,
	}
}

type extensionState struct {
	name        string
	version     string
	description string
}
type extensionStateMap map[string]*extensionState

func (m extensionStateMap) convertToCatalog() []*Extension {
	var res []*Extension
	for _, extension := range m {
		res = append(res, extension.convertToCatalog())
	}
	return res
}

func (extension *extensionState) convertToCatalog() *Extension {
	return &Extension{
		Name:        extension.name,
		Version:     extension.version,
		Description: extension.description,
	}
}
