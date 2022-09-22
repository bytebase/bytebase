package catalog

// This file defines the database state during walk-through. The only two cases will use database state:
//   1. during walk-through
//   2. the underlying implementation of Finder

import (
	"github.com/bytebase/bytebase/plugin/advisor/db"
)

func newDatabaseState(d *Database, context *FinderContext) *DatabaseState {
	database := &DatabaseState{
		complete:     context.CheckIntegrity,
		name:         d.Name,
		characterSet: newStateString(d.CharacterSet),
		collation:    newStateString(d.Collation),
		dbType:       d.DbType,
		schemaSet:    make(schemaStateMap),
	}

	for _, schema := range d.SchemaList {
		database.schemaSet[schema.Name] = newSchemaState(schema, database.complete)
	}

	return database
}

func newSchemaState(s *Schema, complete bool) *SchemaState {
	schema := &SchemaState{
		complete:     complete,
		name:         s.Name,
		tableSet:     make(tableStateMap),
		viewSet:      make(viewStateMap),
		extensionSet: make(extensionStateMap),
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

func newViewState(v *View) *ViewState {
	return &ViewState{
		name:       v.Name,
		definition: newStateString(v.Definition),
		comment:    newStateString(v.Comment),
	}
}

func newExtensionState(e *Extension) *ExtensionState {
	return &ExtensionState{
		name:        e.Name,
		version:     newStateString(e.Version),
		description: newStateString(e.Description),
	}
}

func newTableState(t *Table) *TableState {
	table := &TableState{
		complete:  true,
		name:      t.Name,
		tableType: newStateString(t.Type),
		engine:    newStateString(t.Engine),
		collation: newStateString(t.Collation),
		comment:   newStateString(t.Comment),
		columnSet: make(columnStateMap),
		indexSet:  make(indexStateMap),
	}

	for _, column := range t.ColumnList {
		table.columnSet[column.Name] = newColumnState(column)
	}

	for _, index := range t.IndexList {
		table.indexSet[index.Name] = newIndexState(index)
	}

	return table
}

func newColumnState(c *Column) *ColumnState {
	return &ColumnState{
		complete:     true,
		name:         c.Name,
		position:     newStateInt(int64(c.Position)),
		defaultValue: newStateStringPointer(c.Default),
		nullable:     newStateBool(c.Nullable),
		columnType:   newStateString(c.Type),
		characterSet: newStateString(c.CharacterSet),
		collation:    newStateString(c.Collation),
		comment:      newStateString(c.Comment),
	}
}

func newIndexState(i *Index) *IndexState {
	index := &IndexState{
		complete:       true,
		name:           i.Name,
		indextype:      newStateString(i.Type),
		unique:         newStateBool(i.Unique),
		primary:        newStateBool(i.Primary),
		visible:        newStateBool(i.Visible),
		comment:        newStateString(i.Comment),
		expressionList: newStateStringSlice(i.ExpressionList),
	}
	return index
}

type DatabaseState struct {
	complete     bool
	name         string
	characterSet stateString
	collation    stateString
	dbType       db.Type
	schemaSet    schemaStateMap
	deleted      bool
}

// HasNoTable returns true if the current database has no table.
func (d *DatabaseState) HasNoTable() bool {
	for _, schema := range d.schemaSet {
		if len(schema.tableSet) != 0 {
			return false
		}
	}

	return true
}

// DatabaseName returns the database name.
func (d *DatabaseState) DatabaseName() string {
	return d.name
}

// IndexFind is for find index.
type IndexFind struct {
	SchemaName string
	TableName  string
	IndexName  string
}

func (d *DatabaseState) FindIndex(find *IndexFind) (string, *IndexState) {
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
		return table.name, index
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
			return table.name, index
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
func (d *DatabaseState) FindPrimaryKey(find *PrimaryKeyFind) *IndexState {
	for _, schema := range d.schemaSet {
		if schema.name != find.SchemaName {
			continue
		}
		for _, table := range schema.tableSet {
			if table.name != find.TableName {
				continue
			}
			for _, index := range table.indexSet {
				if index.primary.isTrue() {
					return index
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
func (d *DatabaseState) FindColumn(find *ColumnFind) *ColumnState {
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
	return column
}

// TableFind is for find table.
type TableFind struct {
	SchemaName string
	TableName  string
}

// FindTable finds the table.
func (d *DatabaseState) FindTable(find *TableFind) *TableState {
	schema, exists := d.schemaSet[find.SchemaName]
	if !exists {
		return nil
	}
	table, exists := schema.tableSet[find.TableName]
	if !exists {
		return nil
	}
	return table
}

func (d *DatabaseState) copy() *DatabaseState {
	return &DatabaseState{
		complete:     d.complete,
		name:         d.name,
		characterSet: d.characterSet,
		collation:    d.collation,
		dbType:       d.dbType,
		schemaSet:    d.schemaSet.copy(),
		deleted:      d.deleted,
	}
}

type SchemaState struct {
	complete     bool
	name         string
	tableSet     tableStateMap
	viewSet      viewStateMap
	extensionSet extensionStateMap
}

func (schema *SchemaState) copy() *SchemaState {
	return &SchemaState{
		complete:     schema.complete,
		name:         schema.name,
		tableSet:     schema.tableSet.copy(),
		viewSet:      schema.viewSet.copy(),
		extensionSet: schema.extensionSet.copy(),
	}
}

type schemaStateMap map[string]*SchemaState

func (m schemaStateMap) copy() schemaStateMap {
	res := make(schemaStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

type TableState struct {
	complete bool

	name      string
	tableType stateString
	// engine isn't supported for Postgres, Snowflake, SQLite.
	engine stateString
	// collation isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	collation stateString
	// comment isn't supported for SQLite.
	comment   stateString
	columnSet columnStateMap
	// indexSet isn't supported for ClickHouse, Snowflake.
	indexSet indexStateMap
}

func (table *TableState) copy() *TableState {
	return &TableState{
		complete:  table.complete,
		name:      table.name,
		tableType: table.tableType,
		engine:    table.engine,
		collation: table.collation,
		comment:   table.comment,
		columnSet: table.columnSet.copy(),
		indexSet:  table.indexSet.copy(),
	}
}

type tableStateMap map[string]*TableState

func (m tableStateMap) copy() tableStateMap {
	res := make(tableStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

type IndexState struct {
	complete bool

	name string
	// This could refer to a column or an expression.
	expressionList stateStringSlice
	// Type isn't supported for SQLite.
	indextype stateString
	unique    stateBool
	primary   stateBool
	// Visible isn't supported for Postgres, SQLite.
	visible stateBool
	// Comment isn't supported for SQLite.
	comment stateString
}

func (idx *IndexState) copy() *IndexState {
	return &IndexState{
		complete:       idx.complete,
		name:           idx.name,
		expressionList: idx.expressionList.copy(),
		indextype:      idx.indextype,
		unique:         idx.unique,
		primary:        idx.primary,
		visible:        idx.visible,
		comment:        idx.comment,
	}
}

func (idx *IndexState) Unique() bool {
	return idx.unique.isTrue()
}

func (idx *IndexState) Primary() bool {
	return idx.primary.isTrue()
}

func (idx *IndexState) ExpressionList() []string {
	if idx.expressionList.defined {
		return idx.expressionList.value
	}
	return nil
}

type indexStateMap map[string]*IndexState

func (m indexStateMap) copy() indexStateMap {
	res := make(indexStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

type ColumnState struct {
	complete bool

	name         string
	position     stateInt
	defaultValue stateStringPointer
	// nullable isn't supported for ClickHouse.
	nullable   stateBool
	columnType stateString
	// characterSet isn't supported for Postgres, ClickHouse, SQLite.
	characterSet stateString
	// collation isn't supported for ClickHouse, SQLite.
	collation stateString
	// comment isn't supported for SQLite.
	comment stateString
}

func (col *ColumnState) copy() *ColumnState {
	return &ColumnState{
		complete:     col.complete,
		name:         col.name,
		position:     col.position,
		defaultValue: col.defaultValue.copy(),
		nullable:     col.nullable,
		columnType:   col.columnType,
		characterSet: col.characterSet,
		collation:    col.collation,
		comment:      col.comment,
	}
}

func (col *ColumnState) Nullable() bool {
	return col.nullable.defined && col.nullable.value
}

func (col *ColumnState) Type() string {
	return col.columnType.String()
}

type columnStateMap map[string]*ColumnState

func (m columnStateMap) copy() columnStateMap {
	res := make(columnStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

type ViewState struct {
	name       string
	definition stateString
	comment    stateString
}

func (view *ViewState) copy() *ViewState {
	return &ViewState{
		name:       view.name,
		definition: view.definition,
		comment:    view.comment,
	}
}

type viewStateMap map[string]*ViewState

func (m viewStateMap) copy() viewStateMap {
	res := make(viewStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

type ExtensionState struct {
	name        string
	version     stateString
	description stateString
}

func (extension *ExtensionState) copy() *ExtensionState {
	return &ExtensionState{
		name:        extension.name,
		version:     extension.version,
		description: extension.description,
	}
}

type extensionStateMap map[string]*ExtensionState

func (m extensionStateMap) copy() extensionStateMap {
	res := make(extensionStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}
