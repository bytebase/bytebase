package catalog

// This file defines the database state during walk-through. The only two cases will use database state:
//   1. during walk-through
//   2. the underlying implementation of Finder

import (
	"fmt"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

// NewDatabaseState creates a new database state from schema metadata and context.
func NewDatabaseState(d *storepb.DatabaseSchemaMetadata, ignoreCaseSensitive bool, engineType storepb.Engine) *DatabaseState {
	database := &DatabaseState{
		ignoreCaseSensitive: ignoreCaseSensitive,
		name:                d.Name,
		characterSet:        d.CharacterSet,
		collation:           d.Collation,
		dbType:              engineType,
		schemaSet:           make(schemaStateMap),
	}

	for _, schema := range d.Schemas {
		database.schemaSet[schema.Name] = newSchemaState(schema, database.ignoreCaseSensitive, database.dbType)
	}

	for _, schema := range d.Schemas {
		for _, view := range schema.Views {
			for _, dependencyColumn := range view.DependencyColumns {
				if schemaState, exist := database.schemaSet[dependencyColumn.Schema]; exist {
					if tableState, exist := schemaState.getTable(dependencyColumn.Table); exist {
						tableState.dependencyView[fmt.Sprintf("%q.%q", schema.Name, view.Name)] = true
						if columnState, exist := tableState.columnSet[dependencyColumn.Column]; exist {
							columnState.dependencyView[fmt.Sprintf("%q.%q", schema.Name, view.Name)] = true
						}
					}
				}
			}
		}
	}

	return database
}

func newSchemaState(s *storepb.SchemaMetadata, ignoreCaseSensitive bool, engineType storepb.Engine) *SchemaState {
	schema := &SchemaState{
		ignoreCaseSensitive: ignoreCaseSensitive,
		name:                s.Name,
		tableSet:            make(tableStateMap),
		viewSet:             make(viewStateMap),
		identifierMap:       make(identifierMap),
	}

	for _, table := range s.Tables {
		tableState := newTableState(table, engineType)
		schema.tableSet[table.Name] = tableState

		schema.identifierMap[table.Name] = true
		for indexName := range tableState.indexSet {
			schema.identifierMap[indexName] = true
		}
	}

	for _, view := range s.Views {
		schema.viewSet[view.Name] = newViewState(view)

		schema.identifierMap[view.Name] = true
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

func newTableState(t *storepb.TableMetadata, engineType storepb.Engine) *TableState {
	table := &TableState{
		name:           t.Name,
		engine:         newStringPointer(t.Engine),
		collation:      newStringPointer(t.Collation),
		comment:        newStringPointer(t.Comment),
		columnSet:      make(columnStateMap),
		indexSet:       make(IndexStateMap),
		dependencyView: make(map[string]bool),
	}

	for i, column := range t.Columns {
		columnName := column.Name
		switch engineType {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			columnName = strings.ToLower(columnName)
		default:
			// For other engine types, keep the original column name without normalization
		}
		table.columnSet[columnName] = newColumnState(column, i+1)
	}

	for _, index := range t.Indexes {
		indexName := index.Name
		switch engineType {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			indexName = strings.ToLower(indexName)
		default:
			// For other engine types, keep the original index name without normalization
		}
		table.indexSet[indexName] = newIndexState(index)
	}

	return table
}

func newColumnState(c *storepb.ColumnMetadata, position int) *ColumnState {
	defaultValue := (*string)(nil)
	if c.Default != "" {
		defaultValue = copyStringPointer(&c.Default)
	}
	return &ColumnState{
		name:           c.Name,
		position:       newIntPointer(position),
		defaultValue:   defaultValue,
		nullable:       newBoolPointer(c.Nullable),
		columnType:     newStringPointer(c.Type),
		characterSet:   newStringPointer(c.CharacterSet),
		collation:      newStringPointer(c.Collation),
		comment:        newStringPointer(c.Comment),
		dependencyView: make(map[string]bool),
	}
}

func newIndexState(i *storepb.IndexMetadata) *IndexState {
	index := &IndexState{
		name:           i.Name,
		indexType:      newStringPointer(i.Type),
		unique:         newBoolPointer(i.Unique),
		primary:        newBoolPointer(i.Primary),
		visible:        newBoolPointer(i.Visible),
		comment:        newStringPointer(i.Comment),
		expressionList: copyStringSlice(i.Expressions),
		// We rudely think that pk and uk are constraints here.
		// But in fact, we can create uk by CREATE UNIQUE INDEX statements.
		isConstraint: *newBoolPointer(i.Primary || i.Unique),
	}
	return index
}

// DatabaseState is the state for walk-through.
type DatabaseState struct {
	ignoreCaseSensitive bool
	name                string
	characterSet        string
	collation           string
	dbType              storepb.Engine
	schemaSet           schemaStateMap
	deleted             bool
}

// Index returns the index map of the table.
func (d *DatabaseState) Index(schemaName string, tableName string) *IndexStateMap {
	schema, exists := d.schemaSet[schemaName]
	if !exists {
		return nil
	}
	return schema.Index(tableName)
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

func (d *DatabaseState) HasSchema(name string) bool {
	_, exists := d.schemaSet[name]
	return exists
}

// DatabaseName returns the database name.
func (d *DatabaseState) DatabaseName() string {
	return d.name
}

// GetIndex gets the index.
func (d *DatabaseState) GetIndex(schemaName string, tableName string, indexName string) (string, *IndexState) {
	switch d.dbType {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		indexName = strings.ToLower(indexName)
	default:
		// For other engine types, keep the original index name without normalization
	}
	// There are two cases to find a index:
	// 1. find an index in specific table. e.g. MySQL and TiDB.
	// 2. find an index in the schema. e.g. PostgreSQL.
	// In PostgreSQL, the index name is unique in a schema, not a table.
	// In MySQL and TiDB, the index name is unique in a table.
	// So for case one, we need match table name, but for case two, we don't need.
	needMatchTable := (d.dbType != storepb.Engine_POSTGRES || schemaName == "" || tableName != "")
	if needMatchTable {
		schema, exists := d.schemaSet[schemaName]
		if !exists {
			return "", nil
		}
		table, exists := schema.getTable(tableName)
		if !exists {
			return "", nil
		}
		index, exists := table.indexSet[indexName]
		if !exists {
			return "", nil
		}
		return table.name, index
	}
	for _, schema := range d.schemaSet {
		if schema.name != schemaName {
			continue
		}
		for _, table := range schema.tableSet {
			// no need to further match table name because index is already unique in the schema
			index, exists := table.indexSet[indexName]
			if !exists {
				return "", nil
			}
			return table.name, index
		}
	}
	return "", nil
}

// GetPrimaryKey gets the primary key.
func (d *DatabaseState) GetPrimaryKey(schemaName string, tableName string) *IndexState {
	for _, schema := range d.schemaSet {
		if schema.name != schemaName {
			continue
		}
		for _, table := range schema.tableSet {
			if !compareIdentifier(table.name, tableName, schema.ignoreCaseSensitive) {
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

// HasPrimaryKey checks if a table has a primary key.
func (d *DatabaseState) HasPrimaryKey(schemaName string, tableName string) bool {
	return d.GetPrimaryKey(schemaName, tableName) != nil
}

// CountColumn counts columns.
func (d *DatabaseState) CountColumn(schemaName string, tableName string, columnType string) int {
	schema, exists := d.schemaSet[schemaName]
	if !exists {
		return 0
	}
	table, exists := schema.getTable(tableName)
	if !exists {
		return 0
	}
	res := 0
	for _, column := range table.columnSet {
		if column.columnType != nil && strings.EqualFold(*column.columnType, columnType) {
			res++
		}
	}
	return res
}

// GetColumn gets the column.
func (d *DatabaseState) GetColumn(schemaName string, tableName string, columnName string) *ColumnState {
	switch d.dbType {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		columnName = strings.ToLower(columnName)
	default:
		// For other engine types, keep the original column name without normalization
	}
	schema, exists := d.schemaSet[schemaName]
	if !exists {
		return nil
	}
	table, exists := schema.getTable(tableName)
	if !exists {
		return nil
	}
	column, exists := table.columnSet[columnName]
	if !exists {
		return nil
	}
	return column
}

// GetTable gets the table.
func (d *DatabaseState) GetTable(schemaName string, tableName string) *TableState {
	schema, exists := d.schemaSet[schemaName]
	if !exists {
		return nil
	}
	table, exists := schema.getTable(tableName)
	if !exists {
		return nil
	}
	return table
}

type identifierMap map[string]bool

// SchemaState is the state for walk-through.
type SchemaState struct {
	ignoreCaseSensitive bool
	name                string
	tableSet            tableStateMap
	viewSet             viewStateMap

	// PostgreSQL specific fields
	// All relation names in PostgreSQL must be distinct in schema level.
	identifierMap identifierMap
}

func (s *SchemaState) Index(tableName string) *IndexStateMap {
	table, exists := s.getTable(tableName)
	if !exists {
		return nil
	}
	return table.Index()
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
	indexSet IndexStateMap

	// dependencyView is used to record the dependency view for the table.
	// Used to check if the table is used by any view.
	dependencyView map[string]bool
}

// ListColumns return the list of columns.
func (t *TableState) ListColumns() []*ColumnState {
	var columns []*ColumnState
	for _, v := range t.columnSet {
		columns = append(columns, v)
	}
	return columns
}

// CountIndex return the index total number.
func (t *TableState) CountIndex() int {
	return len(t.indexSet)
}

// Index return the index map of table.
func (t *TableState) Index() *IndexStateMap {
	return &t.indexSet
}

func (t *TableState) copy() *TableState {
	return &TableState{
		name:      t.name,
		engine:    copyStringPointer(t.engine),
		collation: copyStringPointer(t.collation),
		comment:   copyStringPointer(t.comment),
		columnSet: t.columnSet.copy(),
		indexSet:  t.indexSet.copy(),
	}
}

type tableStateMap map[string]*TableState

// IndexState is the state for walk-through.
type IndexState struct {
	name string
	// This could refer to a column or an expression.
	expressionList []string
	// Type isn't supported for SQLite.
	indexType *string
	unique    *bool
	primary   *bool
	// Visible isn't supported for Postgres, SQLite.
	visible *bool
	// Comment isn't supported for SQLite.
	comment *string

	// PostgreSQL specific fields.

	// PostgreSQL treats INDEX and CONSTRAINT differently.
	isConstraint bool
}

func (idx *IndexState) copy() *IndexState {
	return &IndexState{
		name:           idx.name,
		expressionList: copyStringSlice(idx.expressionList),
		indexType:      copyStringPointer(idx.indexType),
		unique:         copyBoolPointer(idx.unique),
		primary:        copyBoolPointer(idx.primary),
		visible:        copyBoolPointer(idx.visible),
		comment:        copyStringPointer(idx.comment),
		isConstraint:   idx.isConstraint,
	}
}

// Unique returns the unique for the index.
func (idx *IndexState) Unique() bool {
	if idx.unique != nil {
		return *idx.unique
	}
	return false
}

// Primary returns the primary for the index.
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

type IndexStateMap map[string]*IndexState

func (m IndexStateMap) copy() IndexStateMap {
	res := make(IndexStateMap)
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

	// dependencyView is used to record the dependency view for the column.
	// Used to check if the column is used by any view.
	dependencyView map[string]bool
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

// HasDefault returns if column has default value.
func (col *ColumnState) HasDefault() bool {
	switch strings.ToLower(col.Type()) {
	case "serial", "smallserial", "bigserial":
		return true
	default:
		// For other column types, check if defaultValue is not nil
	}
	return col.defaultValue != nil
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

func copyStringPointer(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func copyBoolPointer(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func copyIntPointer(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
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

// Schema-level operations.

// createSchemaState creates a new schema state with the given name.
func (d *DatabaseState) createSchemaState(schemaName string) *SchemaState {
	schema := &SchemaState{
		ignoreCaseSensitive: d.ignoreCaseSensitive,
		name:                schemaName,
		tableSet:            make(tableStateMap),
		viewSet:             make(viewStateMap),
		identifierMap:       make(identifierMap),
	}
	d.schemaSet[schemaName] = schema
	return schema
}

// GetOrCreateSchema gets an existing schema or creates a new one if it doesn't exist.
func (d *DatabaseState) GetOrCreateSchema(schemaName string) *SchemaState {
	if schema, exists := d.schemaSet[schemaName]; exists {
		return schema
	}
	return d.createSchemaState(schemaName)
}

// GetSchema gets an existing schema.
//
//nolint:revive
func (d *DatabaseState) GetSchema(schemaName string) (*SchemaState, *WalkThroughError) {
	if schemaName == "" {
		schemaName = publicSchemaName
	}
	if schema, exists := d.schemaSet[schemaName]; exists {
		return schema, nil
	}
	if schemaName != publicSchemaName {
		return nil, &WalkThroughError{
			Code:    code.SchemaNotExists,
			Content: fmt.Sprintf("The schema %q doesn't exist", schemaName),
		}
	}
	return d.createSchemaState(publicSchemaName), nil
}

// DropSchema drops a schema.
func (d *DatabaseState) DropSchema(schemaName string) *WalkThroughError {
	schema, exists := d.schemaSet[schemaName]
	if !exists {
		return &WalkThroughError{
			Code:    code.SchemaNotExists,
			Content: fmt.Sprintf("Schema %q does not exist", schemaName),
		}
	}

	// Delete all identifiers in this schema
	for tableName := range schema.tableSet {
		delete(schema.identifierMap, tableName)
	}
	for viewName := range schema.viewSet {
		delete(schema.identifierMap, viewName)
	}

	// Delete the schema
	delete(d.schemaSet, schemaName)
	return nil
}

// Table-level operations.

// CreateTable creates a new table in the schema.
func (s *SchemaState) CreateTable(tableName string) (*TableState, *WalkThroughError) {
	if _, exists := s.getTable(tableName); exists {
		return nil, NewTableExistsError(tableName)
	}

	table := &TableState{
		name:           tableName,
		engine:         newEmptyStringPointer(),
		collation:      newEmptyStringPointer(),
		comment:        newEmptyStringPointer(),
		columnSet:      make(columnStateMap),
		indexSet:       make(IndexStateMap),
		dependencyView: make(map[string]bool),
	}
	s.tableSet[tableName] = table

	// For PostgreSQL, track in identifier map
	if s.identifierMap != nil {
		s.identifierMap[tableName] = true
	}

	return table, nil
}

// GetTable gets a table from the schema.
//
//nolint:revive
func (s *SchemaState) GetTable(tableName string) (*TableState, *WalkThroughError) {
	table, exists := s.tableSet[tableName]
	if !exists {
		return nil, &WalkThroughError{
			Code:    code.TableNotExists,
			Content: fmt.Sprintf("The table %q does not exist in schema %q", tableName, s.name),
		}
	}
	return table, nil
}

// DropTable drops a table from the schema.
func (s *SchemaState) DropTable(tableName string, checkViewDependency func(*TableState) ([]string, *WalkThroughError)) *WalkThroughError {
	table, err := s.GetTable(tableName)
	if err != nil {
		return err
	}

	// Check for view dependencies if checker is provided
	if checkViewDependency != nil {
		viewList, err := checkViewDependency(table)
		if err != nil {
			return err
		}
		if len(viewList) > 0 {
			return &WalkThroughError{
				Code:    code.TableIsReferencedByView,
				Content: fmt.Sprintf("Cannot drop table %q.%q, it's referenced by view: %s", s.name, table.name, strings.Join(viewList, ", ")),
			}
		}
	}

	// Delete all indexes associated with the table (for PostgreSQL identifier map)
	if s.identifierMap != nil {
		for indexName := range table.indexSet {
			delete(s.identifierMap, indexName)
		}
		delete(s.identifierMap, table.name)
	}

	delete(s.tableSet, tableName)
	return nil
}

// RenameTable renames a table in the schema.
//
//nolint:revive
func (s *SchemaState) RenameTable(oldName string, newName string) *WalkThroughError {
	if oldName == newName {
		return nil
	}

	table, exists := s.getTable(oldName)
	if !exists {
		return NewTableNotExistsError(oldName)
	}

	if _, exists := s.getTable(newName); exists {
		return NewTableExistsError(newName)
	}

	// For PostgreSQL, update identifier map
	if s.identifierMap != nil {
		if _, exists := s.identifierMap[newName]; exists {
			return NewRelationExistsError(newName, s.name)
		}
		delete(s.identifierMap, table.name)
		s.identifierMap[newName] = true
	}

	table.name = newName
	delete(s.tableSet, oldName)
	s.tableSet[newName] = table
	return nil
}

// Column-level operations.

// CreateColumn creates a new column in the table.
//
//nolint:revive
func (t *TableState) CreateColumn(columnName string, columnType *string, nullable *bool, defaultValue *string, position *int, characterSet *string, collation *string, comment *string) *WalkThroughError {
	if _, exists := t.columnSet[strings.ToLower(columnName)]; exists {
		return &WalkThroughError{
			Code:    code.ColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", columnName, t.name),
		}
	}

	pos := len(t.columnSet) + 1
	if position != nil {
		pos = *position
	}

	col := &ColumnState{
		name:           columnName,
		position:       &pos,
		defaultValue:   defaultValue,
		nullable:       nullable,
		columnType:     columnType,
		characterSet:   characterSet,
		collation:      collation,
		comment:        comment,
		dependencyView: make(map[string]bool),
	}
	t.columnSet[strings.ToLower(columnName)] = col
	return nil
}

// GetColumn gets a column from the table.
//
//nolint:revive
func (t *TableState) GetColumn(columnName string) (*ColumnState, *WalkThroughError) {
	column, exists := t.columnSet[strings.ToLower(columnName)]
	if !exists {
		return nil, NewColumnNotExistsError(t.name, columnName)
	}
	return column, nil
}

// DropColumn drops a column from the table.
//
//nolint:revive
func (t *TableState) DropColumn(columnName string, checkViewDependency func(*ColumnState) ([]string, *WalkThroughError)) *WalkThroughError {
	column, exists := t.columnSet[strings.ToLower(columnName)]
	if !exists {
		return NewColumnNotExistsError(t.name, columnName)
	}

	// Cannot drop all columns in a table using ALTER TABLE DROP COLUMN.
	if len(t.columnSet) == 1 {
		return &WalkThroughError{
			Code:    code.DropAllColumns,
			Content: fmt.Sprintf("Can't delete all columns with ALTER TABLE; use DROP TABLE %s instead", t.name),
		}
	}

	// Check for view dependencies if checker is provided
	if checkViewDependency != nil {
		viewList, err := checkViewDependency(column)
		if err != nil {
			return err
		}
		if len(viewList) > 0 {
			return &WalkThroughError{
				Code:    code.ColumnIsReferencedByView,
				Content: fmt.Sprintf("Cannot drop column %q in table %q, it's referenced by view: %s", column.name, t.name, strings.Join(viewList, ", ")),
			}
		}
	}

	// Modify the column position
	for _, col := range t.columnSet {
		if *col.position > *column.position {
			*col.position--
		}
	}

	// Remove column from indexes, and drop indexes that become empty
	for _, index := range t.indexSet {
		// Remove the column from the index key list
		var newKeyList []string
		for _, key := range index.expressionList {
			if !strings.EqualFold(key, columnName) {
				newKeyList = append(newKeyList, key)
			}
		}
		index.expressionList = newKeyList

		// If all columns that make up an index are dropped, the index is dropped as well
		if len(index.expressionList) == 0 {
			delete(t.indexSet, strings.ToLower(index.name))
		}
	}

	delete(t.columnSet, strings.ToLower(columnName))
	return nil
}

// RenameColumn renames a column in the table.
//
//nolint:revive
func (t *TableState) RenameColumn(oldName string, newName string) *WalkThroughError {
	if strings.EqualFold(oldName, newName) {
		return nil
	}

	column, exists := t.columnSet[strings.ToLower(oldName)]
	if !exists {
		return NewColumnNotExistsError(t.name, oldName)
	}

	if _, exists := t.columnSet[strings.ToLower(newName)]; exists {
		return &WalkThroughError{
			Code:    code.ColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", newName, t.name),
		}
	}

	// Rename column in all indexes that reference it
	for _, index := range t.indexSet {
		for i, key := range index.expressionList {
			if strings.EqualFold(key, oldName) {
				index.expressionList[i] = newName
			}
		}
	}

	column.name = newName
	delete(t.columnSet, strings.ToLower(oldName))
	t.columnSet[strings.ToLower(newName)] = column
	return nil
}

// Index-level operations.

// CreateIndex creates a new index in the table.
//
//nolint:revive
func (t *TableState) CreateIndex(indexName string, expressionList []string, unique bool, indexType string, primary bool, visible *bool, comment *string, isConstraint bool, identifierMap identifierMap) *WalkThroughError {
	if len(expressionList) == 0 {
		return &WalkThroughError{
			Code:    code.IndexEmptyKeys,
			Content: fmt.Sprintf("Index `%s` in table `%s` has empty key", indexName, t.name),
		}
	}

	// Auto-generate name if empty
	if indexName == "" {
		suffix := 1
		baseName := expressionList[0]
		for {
			indexName = baseName
			if suffix > 1 {
				indexName = fmt.Sprintf("%s_%d", baseName, suffix)
			}
			if _, exists := t.indexSet[strings.ToLower(indexName)]; !exists {
				break
			}
			suffix++
		}
	} else {
		// Check if index already exists
		if _, exists := t.indexSet[strings.ToLower(indexName)]; exists {
			return NewIndexExistsError(t.name, indexName)
		}
	}

	// For PostgreSQL, check identifier map
	if identifierMap != nil {
		if _, exists := identifierMap[indexName]; exists {
			return NewRelationExistsError(indexName, "")
		}
	}

	if visible == nil {
		visible = newTruePointer()
	}
	if comment == nil {
		comment = newEmptyStringPointer()
	}

	index := &IndexState{
		name:           indexName,
		expressionList: expressionList,
		indexType:      &indexType,
		unique:         &unique,
		primary:        &primary,
		visible:        visible,
		comment:        comment,
		isConstraint:   isConstraint,
	}
	t.indexSet[strings.ToLower(indexName)] = index

	// For PostgreSQL, track in identifier map
	if identifierMap != nil {
		identifierMap[indexName] = true
	}

	return nil
}

// CreatePrimaryKey creates a primary key index in the table.
//
//nolint:revive
func (t *TableState) CreatePrimaryKey(keyList []string, indexType string, identifierMap identifierMap) *WalkThroughError {
	pkName := PrimaryKeyName

	if _, exists := t.indexSet[strings.ToLower(pkName)]; exists {
		return &WalkThroughError{
			Code:    code.PrimaryKeyExists,
			Content: fmt.Sprintf("Primary key exists in table `%s`", t.name),
		}
	}

	// For PostgreSQL, check identifier map
	if identifierMap != nil {
		if _, exists := identifierMap[pkName]; exists {
			return NewRelationExistsError(pkName, "")
		}
	}

	pk := &IndexState{
		name:           pkName,
		expressionList: keyList,
		indexType:      &indexType,
		unique:         newTruePointer(),
		primary:        newTruePointer(),
		visible:        newTruePointer(),
		comment:        newEmptyStringPointer(),
		isConstraint:   true,
	}
	t.indexSet[strings.ToLower(pkName)] = pk

	// For PostgreSQL, track in identifier map
	if identifierMap != nil {
		identifierMap[pkName] = true
	}

	return nil
}

// GetIndex gets an index from the table.
func (t *TableState) GetIndex(indexName string) (*IndexState, *WalkThroughError) {
	index, exists := t.indexSet[strings.ToLower(indexName)]
	if !exists {
		return nil, NewIndexNotExistsError(t.name, indexName)
	}
	return index, nil
}

// DropIndex drops an index from the table.
//
//nolint:revive
func (t *TableState) DropIndex(indexName string, identifierMap identifierMap) *WalkThroughError {
	if _, exists := t.indexSet[strings.ToLower(indexName)]; !exists {
		if strings.EqualFold(indexName, PrimaryKeyName) {
			return &WalkThroughError{
				Code:    code.PrimaryKeyNotExists,
				Content: fmt.Sprintf("Primary key does not exist in table `%s`", t.name),
			}
		}
		return NewIndexNotExistsError(t.name, indexName)
	}

	// For PostgreSQL, remove from identifier map
	if identifierMap != nil {
		delete(identifierMap, indexName)
	}

	delete(t.indexSet, strings.ToLower(indexName))
	return nil
}

// RenameIndex renames an index in the table.
//
//nolint:revive
func (t *TableState) RenameIndex(oldName string, newName string, identifierMap identifierMap) *WalkThroughError {
	// For MySQL, the primary key has a special name 'PRIMARY'.
	if strings.ToUpper(oldName) == PrimaryKeyName || strings.ToUpper(newName) == PrimaryKeyName {
		incorrectName := oldName
		if strings.ToUpper(oldName) != PrimaryKeyName {
			incorrectName = newName
		}
		return &WalkThroughError{
			Code:    code.IncorrectIndexName,
			Content: fmt.Sprintf("Incorrect index name `%s`", incorrectName),
		}
	}

	index, exists := t.indexSet[strings.ToLower(oldName)]
	if !exists {
		return NewIndexNotExistsError(t.name, oldName)
	}

	if _, exists := t.indexSet[strings.ToLower(newName)]; exists {
		return NewIndexExistsError(t.name, newName)
	}

	// For PostgreSQL, update identifier map
	if identifierMap != nil {
		if _, exists := identifierMap[newName]; exists {
			return NewRelationExistsError(newName, "")
		}
		delete(identifierMap, oldName)
		identifierMap[newName] = true
	}

	index.name = newName
	delete(t.indexSet, strings.ToLower(oldName))
	t.indexSet[strings.ToLower(newName)] = index
	return nil
}

// View-level operations.

// CreateView creates a new view in the schema.
func (s *SchemaState) CreateView(viewName string, definition *string, comment *string) (*ViewState, *WalkThroughError) {
	view := &ViewState{
		name:       viewName,
		definition: definition,
		comment:    comment,
	}
	s.viewSet[viewName] = view

	// For PostgreSQL, track in identifier map
	if s.identifierMap != nil {
		s.identifierMap[viewName] = true
	}

	return view, nil
}

// DropView drops a view from the schema.
func (s *SchemaState) DropView(viewName string, identifierMap identifierMap) *WalkThroughError {
	// For PostgreSQL, remove from identifier map
	if identifierMap != nil {
		delete(identifierMap, viewName)
	}

	delete(s.viewSet, viewName)
	return nil
}

// Database-level operations.

// MarkDeleted marks the database as deleted.
func (d *DatabaseState) MarkDeleted() {
	d.deleted = true
}

// IsDeleted returns true if the database is marked as deleted.
func (d *DatabaseState) IsDeleted() bool {
	return d.deleted
}

// SetCharacterSet sets the character set of the database.
func (d *DatabaseState) SetCharacterSet(characterSet string) {
	d.characterSet = characterSet
}

// SetCollation sets the collation of the database.
func (d *DatabaseState) SetCollation(collation string) {
	d.collation = collation
}

// SetTableEngine sets the engine of the table.
func (t *TableState) SetEngine(engine string) {
	t.engine = newStringPointer(engine)
}

// SetTableCollation sets the collation of the table.
func (t *TableState) SetCollation(collation string) {
	t.collation = newStringPointer(collation)
}

// SetTableComment sets the comment of the table.
func (t *TableState) SetComment(comment string) {
	t.comment = newStringPointer(comment)
}

// SetColumnNullable sets the nullable property of the column.
func (col *ColumnState) SetNullable(nullable bool) {
	col.nullable = newBoolPointer(nullable)
}

// SetColumnDefault sets the default value of the column.
func (col *ColumnState) SetDefault(defaultValue *string) {
	col.defaultValue = defaultValue
}

// SetColumnType sets the type of the column.
func (col *ColumnState) SetType(columnType string) {
	col.columnType = newStringPointer(columnType)
}

// SetIndexVisibility sets the visibility of the index.
func (idx *IndexState) SetVisibility(visible bool) {
	idx.visible = newBoolPointer(visible)
}
