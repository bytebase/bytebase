package catalog

// This file defines the database state during walk-through. The only two cases will use database state:
//   1. during walk-through
//   2. the underlying implementation of Finder

import (
	"strings"

	"github.com/bytebase/bytebase/plugin/advisor/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func newDatabaseState(d *storepb.DatabaseMetadata, context *FinderContext) *DatabaseState {
	database := &DatabaseState{
		ctx:          context.Copy(),
		name:         d.Name,
		characterSet: d.CharacterSet,
		collation:    d.Collation,
		dbType:       context.EngineType,
		schemaSet:    make(schemaStateMap),
	}

	for _, schema := range d.Schemas {
		database.schemaSet[schema.Name] = newSchemaState(schema, database.ctx)
	}

	return database
}

func newSchemaState(s *storepb.SchemaMetadata, context *FinderContext) *SchemaState {
	schema := &SchemaState{
		ctx:          context.Copy(),
		name:         s.Name,
		tableSet:     make(tableStateMap),
		viewSet:      make(viewStateMap),
		extensionSet: make(extensionStateMap),
	}

	for _, table := range s.Tables {
		schema.tableSet[table.Name] = newTableState(table)
	}

	for _, view := range s.Views {
		schema.viewSet[view.Name] = newViewState(view)
	}

	return schema
}

func newViewState(v *storepb.ViewMetadata) *ViewState {
	return &ViewState{
		name:       v.Name,
		definition: newStringPointer(v.Definition),
		comment:    newStringPointer(v.Comment),
	}
}

func newTableState(t *storepb.TableMetadata) *TableState {
	table := &TableState{
		name:      t.Name,
		engine:    newStringPointer(t.Engine),
		collation: newStringPointer(t.Collation),
		comment:   newStringPointer(t.Comment),
		columnSet: make(columnStateMap),
		indexSet:  make(indexStateMap),
	}

	for i, column := range t.Columns {
		table.columnSet[column.Name] = newColumnState(column, i+1)
	}

	for _, index := range t.Indexes {
		table.indexSet[index.Name] = newIndexState(index)
	}

	return table
}

func newColumnState(c *storepb.ColumnMetadata, position int) *ColumnState {
	defaultValue := (*string)(nil)
	if c.HasDefault {
		defaultValue = copyStringPointer(&c.Default)
	}
	return &ColumnState{
		name:         c.Name,
		position:     newIntPointer(position),
		defaultValue: defaultValue,
		nullable:     newBoolPointer(c.Nullable),
		columnType:   newStringPointer(c.Type),
		characterSet: newStringPointer(c.CharacterSet),
		collation:    newStringPointer(c.Collation),
		comment:      newStringPointer(c.Comment),
	}
}

func newIndexState(i *storepb.IndexMetadata) *IndexState {
	index := &IndexState{
		name:           i.Name,
		indextype:      newStringPointer(i.Type),
		unique:         newBoolPointer(i.Unique),
		primary:        newBoolPointer(i.Primary),
		visible:        newBoolPointer(i.Visible),
		comment:        newStringPointer(i.Comment),
		expressionList: copyStringSlice(i.Expressions),
	}
	return index
}

// DatabaseState is the state for walk-through.
type DatabaseState struct {
	ctx          *FinderContext
	name         string
	characterSet string
	collation    string
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

// FindIndex finds the index.
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
				if index.primary != nil && *index.primary {
					return index
				}
			}
		}
	}
	return nil
}

// ColumnCount is for counting columns.
type ColumnCount struct {
	SchemaName string
	TableName  string
	ColumnType string
}

// CountColumn counts columns.
func (d *DatabaseState) CountColumn(count *ColumnCount) int {
	schema, exists := d.schemaSet[count.SchemaName]
	if !exists {
		return 0
	}
	table, exists := schema.tableSet[count.TableName]
	if !exists {
		return 0
	}
	res := 0
	for _, column := range table.columnSet {
		if column.columnType != nil && strings.EqualFold(*column.columnType, count.ColumnType) {
			res++
		}
	}
	return res
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

// SchemaState is the state for walk-through.
type SchemaState struct {
	ctx          *FinderContext
	name         string
	tableSet     tableStateMap
	viewSet      viewStateMap
	extensionSet extensionStateMap
}
type schemaStateMap map[string]*SchemaState

// TableState is the state for walk-through.
type TableState struct {
	name string
	// engine isn't supported for Postgres, Snowflake, SQLite.
	engine *string
	// collation isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	collation *string
	// comment isn't supported for SQLite.
	comment   *string
	columnSet columnStateMap
	// indexSet isn't supported for ClickHouse, Snowflake.
	indexSet indexStateMap
}

// CountIndex return the index total number.
func (table *TableState) CountIndex() int {
	return len(table.indexSet)
}

func (table *TableState) copy() *TableState {
	return &TableState{
		name:      table.name,
		engine:    copyStringPointer(table.engine),
		collation: copyStringPointer(table.collation),
		comment:   copyStringPointer(table.comment),
		columnSet: table.columnSet.copy(),
		indexSet:  table.indexSet.copy(),
	}
}

type tableStateMap map[string]*TableState

// IndexState is the state for walk-through.
type IndexState struct {
	name string
	// This could refer to a column or an expression.
	expressionList []string
	// Type isn't supported for SQLite.
	indextype *string
	unique    *bool
	primary   *bool
	// Visible isn't supported for Postgres, SQLite.
	visible *bool
	// Comment isn't supported for SQLite.
	comment *string
}

func (idx *IndexState) copy() *IndexState {
	return &IndexState{
		name:           idx.name,
		expressionList: copyStringSlice(idx.expressionList),
		indextype:      copyStringPointer(idx.indextype),
		unique:         copyBoolPointer(idx.unique),
		primary:        copyBoolPointer(idx.primary),
		visible:        copyBoolPointer(idx.visible),
		comment:        copyStringPointer(idx.comment),
	}
}

// Unique returns the unique for the index.
func (idx *IndexState) Unique() bool {
	if idx.unique != nil {
		return *idx.unique
	}
	return false
}

// Primary returns the priamry for the index.
func (idx *IndexState) Primary() bool {
	if idx.primary != nil {
		return *idx.primary
	}
	return false
}

// ExpressionList returns the expression list for the index.
func (idx *IndexState) ExpressionList() []string {
	return idx.expressionList
}

type indexStateMap map[string]*IndexState

func (m indexStateMap) copy() indexStateMap {
	res := make(indexStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

// ColumnState is the state for walk-through.
type ColumnState struct {
	name         string
	position     *int
	defaultValue *string
	// nullable isn't supported for ClickHouse.
	nullable   *bool
	columnType *string
	// characterSet isn't supported for Postgres, ClickHouse, SQLite.
	characterSet *string
	// collation isn't supported for ClickHouse, SQLite.
	collation *string
	// comment isn't supported for SQLite.
	comment *string
}

func (col *ColumnState) copy() *ColumnState {
	return &ColumnState{
		name:         col.name,
		position:     copyIntPointer(col.position),
		defaultValue: copyStringPointer(col.defaultValue),
		nullable:     copyBoolPointer(col.nullable),
		columnType:   copyStringPointer(col.columnType),
		characterSet: copyStringPointer(col.characterSet),
		collation:    copyStringPointer(col.collation),
		comment:      copyStringPointer(col.comment),
	}
}

// Nullable returns nullable for the column.
func (col *ColumnState) Nullable() bool {
	return col.nullable != nil && *col.nullable
}

// Type returns type for the column.
func (col *ColumnState) Type() string {
	if col.columnType != nil {
		return *col.columnType
	}
	return ""
}

type columnStateMap map[string]*ColumnState

func (m columnStateMap) copy() columnStateMap {
	res := make(columnStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

// ViewState is the state for walk-through.
type ViewState struct {
	name       string
	definition *string
	comment    *string
}
type viewStateMap map[string]*ViewState

// ExtensionState is the state for walk-through.
type ExtensionState struct {
	name        string
	version     *string
	description *string
}
type extensionStateMap map[string]*ExtensionState

func copyStringPointer(p *string) *string {
	if p != nil {
		v := *p
		return &v
	}
	return nil
}

func copyBoolPointer(p *bool) *bool {
	if p != nil {
		v := *p
		return &v
	}
	return nil
}

func copyIntPointer(p *int) *int {
	if p != nil {
		v := *p
		return &v
	}
	return nil
}

func copyStringSlice(in []string) []string {
	var res []string
	res = append(res, in...)
	return res
}

func newEmptyStringPointer() *string {
	res := ""
	return &res
}

func newStringPointer(v string) *string {
	return &v
}

func newIntPointer(v int) *int {
	return &v
}

func newTruePointer() *bool {
	v := true
	return &v
}

func newFalsePointer() *bool {
	v := false
	return &v
}

func newBoolPointer(v bool) *bool {
	return &v
}
