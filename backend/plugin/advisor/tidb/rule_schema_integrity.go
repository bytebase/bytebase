package tidb

// Schema Integrity Advisor - validates TiDB DDL statements against the existing schema

import (
	"context"
	"fmt"
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/types"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

const (
	siPrimaryKeyName = "PRIMARY"
	siFullTextName   = "FULLTEXT"
	siSpatialName    = "SPATIAL"
)

var (
	_ advisor.Advisor = (*SchemaIntegrityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.SchemaRuleSchemaIntegrity, &SchemaIntegrityAdvisor{})
}

type SchemaIntegrityAdvisor struct {
}

func (*SchemaIntegrityAdvisor) Check(_ context.Context, ctx advisor.Context) ([]*storepb.Advice, error) {
	nodeList, ok := ctx.AST.([]tidbast.StmtNode)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", ctx.AST)
	}

	dbState := siNewDatabaseStateFromCatalog(ctx.DBSchema)

	for _, node := range nodeList {
		if err := dbState.changeState(node); err != nil {
			if sve, ok := err.(*siSchemaViolationError); ok {
				return []*storepb.Advice{{
					Status:  storepb.Advice_ERROR,
					Code:    sve.Code,
					Title:   string(ctx.Rule.Type),
					Content: sve.Message,
				}}, nil
			}
			return nil, err
		}
	}

	// No violations found, return empty advice list
	return make([]*storepb.Advice, 0), nil
}

// siSchemaViolationError represents a SQL schema validation error
type siSchemaViolationError struct {
	Code    int32
	Message string
}

func (e *siSchemaViolationError) Error() string {
	return e.Message
}

func siNewSchemaViolationError(code int32, message string) *siSchemaViolationError {
	return &siSchemaViolationError{
		Code:    code,
		Message: message,
	}
}

// siFinderContext holds configuration for schema validation
type siFinderContext struct {
	CheckIntegrity      bool
	EngineType          storepb.Engine
	IgnoreCaseSensitive bool
}

func (c *siFinderContext) Copy() *siFinderContext {
	return &siFinderContext{
		CheckIntegrity:      c.CheckIntegrity,
		EngineType:          c.EngineType,
		IgnoreCaseSensitive: c.IgnoreCaseSensitive,
	}
}

// siNewDatabaseStateFromCatalog creates a database state from the catalog
func siNewDatabaseStateFromCatalog(dbSchema *storepb.DatabaseSchemaMetadata) *siDatabaseState {
	db := &siDatabaseState{
		ctx: &siFinderContext{
			CheckIntegrity:      true,
			EngineType:          storepb.Engine_TIDB,
			IgnoreCaseSensitive: true,
		},
		name:         dbSchema.Name,
		characterSet: dbSchema.CharacterSet,
		collation:    dbSchema.Collation,
		dbType:       storepb.Engine_TIDB,
		schemaSet:    make(siSchemaStateMap),
		usable:       true,
	}

	for _, schema := range dbSchema.Schemas {
		db.schemaSet[schema.Name] = siNewSchemaState(schema, db.ctx)
	}

	return db
}

func siNewSchemaState(s *storepb.SchemaMetadata, context *siFinderContext) *siSchemaState {
	schema := &siSchemaState{
		ctx:      context.Copy(),
		name:     s.Name,
		tableSet: make(siTableStateMap),
	}

	for _, table := range s.Tables {
		tableState := siNewTableState(table, context)
		schema.tableSet[table.Name] = tableState
	}

	return schema
}

func siNewTableState(t *storepb.TableMetadata, _ *siFinderContext) *siTableState {
	table := &siTableState{
		name:      t.Name,
		engine:    siNewStringPointer(t.Engine),
		collation: siNewStringPointer(t.Collation),
		comment:   siNewStringPointer(t.Comment),
		columnSet: make(siColumnStateMap),
		indexSet:  make(siIndexStateMap),
	}

	for i, column := range t.Columns {
		columnName := strings.ToLower(column.Name)
		table.columnSet[columnName] = siNewColumnState(column, i+1)
	}

	for _, index := range t.Indexes {
		indexName := strings.ToLower(index.Name)
		table.indexSet[indexName] = siNewIndexState(index)
	}

	return table
}

func siNewColumnState(c *storepb.ColumnMetadata, position int) *siColumnState {
	defaultValue := (*string)(nil)
	if c.Default != "" {
		defaultValue = siCopyStringPointer(&c.Default)
	}
	return &siColumnState{
		name:         c.Name,
		position:     siNewIntPointer(position),
		defaultValue: defaultValue,
		nullable:     siNewBoolPointer(c.Nullable),
		columnType:   siNewStringPointer(c.Type),
		characterSet: siNewStringPointer(c.CharacterSet),
		collation:    siNewStringPointer(c.Collation),
		comment:      siNewStringPointer(c.Comment),
	}
}

func siNewIndexState(i *storepb.IndexMetadata) *siIndexState {
	return &siIndexState{
		name:           i.Name,
		indexType:      siNewStringPointer(i.Type),
		unique:         siNewBoolPointer(i.Unique),
		primary:        siNewBoolPointer(i.Primary),
		visible:        siNewBoolPointer(i.Visible),
		comment:        siNewStringPointer(i.Comment),
		expressionList: siCopyStringSlice(i.Expressions),
	}
}

// State types

type siDatabaseState struct {
	ctx          *siFinderContext
	name         string
	characterSet string
	collation    string
	dbType       storepb.Engine
	schemaSet    siSchemaStateMap
	deleted      bool
	usable       bool
}

type siSchemaStateMap map[string]*siSchemaState

type siSchemaState struct {
	ctx      *siFinderContext
	name     string
	tableSet siTableStateMap
}

type siTableStateMap map[string]*siTableState

type siTableState struct {
	name      string
	engine    *string
	collation *string
	comment   *string
	columnSet siColumnStateMap
	indexSet  siIndexStateMap
}

type siColumnStateMap map[string]*siColumnState

type siColumnState struct {
	name         string
	position     *int
	defaultValue *string
	nullable     *bool
	columnType   *string
	characterSet *string
	collation    *string
	comment      *string
}

type siIndexStateMap map[string]*siIndexState

type siIndexState struct {
	name           string
	expressionList []string
	indexType      *string
	unique         *bool
	primary        *bool
	visible        *bool
	comment        *string
}

// Helper functions

func siCopyStringPointer(p *string) *string {
	if p != nil {
		v := *p
		return &v
	}
	return nil
}

func siCopyBoolPointer(p *bool) *bool {
	if p != nil {
		v := *p
		return &v
	}
	return nil
}

func siCopyIntPointer(p *int) *int {
	if p != nil {
		v := *p
		return &v
	}
	return nil
}

func siCopyStringSlice(in []string) []string {
	var res []string
	res = append(res, in...)
	return res
}

func siNewEmptyStringPointer() *string {
	res := ""
	return &res
}

func siNewStringPointer(v string) *string {
	return &v
}

func siNewIntPointer(v int) *int {
	return &v
}

func siNewTruePointer() *bool {
	v := true
	return &v
}

func siNewFalsePointer() *bool {
	v := false
	return &v
}

func siNewBoolPointer(v bool) *bool {
	return &v
}

func siCompareIdentifier(a, b string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// State modification methods

func (d *siDatabaseState) changeState(in tidbast.StmtNode) error {
	if d.deleted {
		return siNewSchemaViolationError(703, fmt.Sprintf("Database `%s` is deleted", d.name))
	}

	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL and TiDB.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	if _, exists := d.schemaSet[""]; !exists {
		d.createSchema()
	}

	switch node := in.(type) {
	case *tidbast.CreateTableStmt:
		return d.createTable(node)
	case *tidbast.DropTableStmt:
		return d.dropTable(node)
	case *tidbast.AlterTableStmt:
		return d.alterTable(node)
	case *tidbast.CreateIndexStmt:
		return d.createIndex(node)
	case *tidbast.DropIndexStmt:
		return d.dropIndex(node)
	case *tidbast.AlterDatabaseStmt:
		return d.alterDatabase(node)
	case *tidbast.DropDatabaseStmt:
		return d.dropDatabase(node)
	case *tidbast.CreateDatabaseStmt:
		return siNewSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", node.Name.O, d.name))
	case *tidbast.RenameTableStmt:
		return d.renameTable(node)
	default:
		return nil
	}
}

func (d *siDatabaseState) isCurrentDatabase(database string) bool {
	return siCompareIdentifier(d.name, database, d.ctx.IgnoreCaseSensitive)
}

func (d *siDatabaseState) createSchema() *siSchemaState {
	schema := &siSchemaState{
		ctx:      d.ctx.Copy(),
		name:     "",
		tableSet: make(siTableStateMap),
	}
	d.schemaSet[""] = schema
	return schema
}

func (s *siSchemaState) getTable(table string) (*siTableState, bool) {
	for k, v := range s.tableSet {
		if siCompareIdentifier(k, table, s.ctx.IgnoreCaseSensitive) {
			return v, true
		}
	}
	return nil, false
}

func (s *siSchemaState) createIncompleteTable(name string) *siTableState {
	table := &siTableState{
		name:      name,
		columnSet: make(siColumnStateMap),
		indexSet:  make(siIndexStateMap),
	}
	s.tableSet[name] = table
	return table
}

func (s *siSchemaState) renameTable(ctx *siFinderContext, oldName string, newName string) error {
	if oldName == newName {
		return nil
	}

	table, exists := s.getTable(oldName)
	if !exists {
		if ctx.CheckIntegrity {
			return siNewSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", oldName))
		}
		table = s.createIncompleteTable(oldName)
	}

	if _, exists := s.getTable(newName); exists {
		return siNewSchemaViolationError(607, fmt.Sprintf("Table `%s` already exists", newName))
	}

	table.name = newName
	delete(s.tableSet, oldName)
	s.tableSet[newName] = table
	return nil
}

func (t *siTableState) createIncompleteColumn(name string) *siColumnState {
	column := &siColumnState{
		name: name,
	}
	t.columnSet[strings.ToLower(name)] = column
	return column
}

func (t *siTableState) createIncompleteIndex(name string) *siIndexState {
	index := &siIndexState{
		name: name,
	}
	t.indexSet[strings.ToLower(name)] = index
	return index
}

func (t *siTableState) copy() *siTableState {
	return &siTableState{
		name:      t.name,
		engine:    siCopyStringPointer(t.engine),
		collation: siCopyStringPointer(t.collation),
		comment:   siCopyStringPointer(t.comment),
		columnSet: t.columnSet.copy(),
		indexSet:  t.indexSet.copy(),
	}
}

func (m siColumnStateMap) copy() siColumnStateMap {
	res := make(siColumnStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

func (col *siColumnState) copy() *siColumnState {
	return &siColumnState{
		name:         col.name,
		position:     siCopyIntPointer(col.position),
		defaultValue: siCopyStringPointer(col.defaultValue),
		nullable:     siCopyBoolPointer(col.nullable),
		columnType:   siCopyStringPointer(col.columnType),
		characterSet: siCopyStringPointer(col.characterSet),
		collation:    siCopyStringPointer(col.collation),
		comment:      siCopyStringPointer(col.comment),
	}
}

func (m siIndexStateMap) copy() siIndexStateMap {
	res := make(siIndexStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

func (idx *siIndexState) copy() *siIndexState {
	return &siIndexState{
		name:           idx.name,
		expressionList: siCopyStringSlice(idx.expressionList),
		indexType:      siCopyStringPointer(idx.indexType),
		unique:         siCopyBoolPointer(idx.unique),
		primary:        siCopyBoolPointer(idx.primary),
		visible:        siCopyBoolPointer(idx.visible),
		comment:        siCopyStringPointer(idx.comment),
	}
}

// Statement handlers

func (d *siDatabaseState) renameTable(node *tidbast.RenameTableStmt) error {
	for _, tableToTable := range node.TableToTables {
		schema, exists := d.schemaSet[""]
		if !exists {
			schema = d.createSchema()
		}
		oldTableName := tableToTable.OldTable.Name.O
		newTableName := tableToTable.NewTable.Name.O
		if d.theCurrentDatabase(tableToTable) {
			if siCompareIdentifier(oldTableName, newTableName, d.ctx.IgnoreCaseSensitive) {
				return nil
			}
			table, exists := schema.getTable(oldTableName)
			if !exists {
				if schema.ctx.CheckIntegrity {
					return siNewSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", oldTableName))
				}
				table = schema.createIncompleteTable(oldTableName)
			}
			if _, exists := schema.getTable(newTableName); exists {
				return siNewSchemaViolationError(607, fmt.Sprintf("Table `%s` already exists", newTableName))
			}
			delete(schema.tableSet, table.name)
			table.name = newTableName
			schema.tableSet[table.name] = table
		} else if d.moveToOtherDatabase(tableToTable) {
			_, exists := schema.getTable(tableToTable.OldTable.Name.O)
			if !exists && schema.ctx.CheckIntegrity {
				return siNewSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", tableToTable.OldTable.Name.O))
			}
			delete(schema.tableSet, tableToTable.OldTable.Name.O)
		} else {
			return siNewSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", d.targetDatabase(tableToTable), d.name))
		}
	}
	return nil
}

func (d *siDatabaseState) targetDatabase(node *tidbast.TableToTable) string {
	if node.OldTable.Schema.O != "" && !d.isCurrentDatabase(node.OldTable.Schema.O) {
		return node.OldTable.Schema.O
	}
	return node.NewTable.Schema.O
}

func (d *siDatabaseState) moveToOtherDatabase(node *tidbast.TableToTable) bool {
	if node.OldTable.Schema.O != "" && !d.isCurrentDatabase(node.OldTable.Schema.O) {
		return false
	}
	return node.OldTable.Schema.O != node.NewTable.Schema.O
}

func (d *siDatabaseState) theCurrentDatabase(node *tidbast.TableToTable) bool {
	if node.NewTable.Schema.O != "" && !d.isCurrentDatabase(node.NewTable.Schema.O) {
		return false
	}
	if node.OldTable.Schema.O != "" && !d.isCurrentDatabase(node.OldTable.Schema.O) {
		return false
	}
	return true
}

func (d *siDatabaseState) dropDatabase(node *tidbast.DropDatabaseStmt) error {
	if !d.isCurrentDatabase(node.Name.O) {
		return siNewSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", node.Name.O, d.name))
	}

	d.deleted = true
	return nil
}

func (d *siDatabaseState) alterDatabase(node *tidbast.AlterDatabaseStmt) error {
	if !node.AlterDefaultDatabase && !d.isCurrentDatabase(node.Name.O) {
		return siNewSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", node.Name.O, d.name))
	}

	for _, option := range node.Options {
		switch option.Tp {
		case tidbast.DatabaseOptionCharset:
			d.characterSet = option.Value
		case tidbast.DatabaseOptionCollate:
			d.collation = option.Value
		default:
			// Other database options
		}
	}
	return nil
}

func (d *siDatabaseState) findTableState(tableName *tidbast.TableName) (*siTableState, error) {
	if tableName.Schema.O != "" && !d.isCurrentDatabase(tableName.Schema.O) {
		return nil, siNewSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", tableName.Schema.O, d.name))
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema()
	}

	table, exists := schema.getTable(tableName.Name.O)
	if !exists {
		if schema.ctx.CheckIntegrity {
			return nil, siNewSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", tableName.Name.O))
		}
		table = schema.createIncompleteTable(tableName.Name.O)
	}

	return table, nil
}

func (d *siDatabaseState) dropIndex(node *tidbast.DropIndexStmt) error {
	table, err := d.findTableState(node.Table)
	if err != nil {
		return err
	}

	return table.dropIndex(d.ctx, node.IndexName)
}

func (d *siDatabaseState) createIndex(node *tidbast.CreateIndexStmt) error {
	table, err := d.findTableState(node.Table)
	if err != nil {
		return err
	}

	unique := false
	tp := tidbast.IndexTypeBtree.String()
	isSpatial := false

	switch node.KeyType {
	case tidbast.IndexKeyTypeNone:
	case tidbast.IndexKeyTypeUnique:
		unique = true
	case tidbast.IndexKeyTypeFulltext:
		tp = siFullTextName
	case tidbast.IndexKeyTypeSpatial:
		isSpatial = true
		tp = siSpatialName
	default:
		// Other index key types
	}

	keyList, err := table.validateAndGetKeyStringList(d.ctx, node.IndexPartSpecifications, false /* primary */, isSpatial)
	if err != nil {
		return err
	}

	return table.createIndex(node.IndexName, keyList, unique, tp, node.IndexOption)
}

func (d *siDatabaseState) alterTable(node *tidbast.AlterTableStmt) error {
	table, err := d.findTableState(node.Table)
	if err != nil {
		return err
	}

	for _, spec := range node.Specs {
		switch spec.Tp {
		case tidbast.AlterTableOption:
			for _, option := range spec.Options {
				switch option.Tp {
				case tidbast.TableOptionCollate:
					table.collation = siNewStringPointer(option.StrValue)
				case tidbast.TableOptionComment:
					table.comment = siNewStringPointer(option.StrValue)
				case tidbast.TableOptionEngine:
					table.engine = siNewStringPointer(option.StrValue)
				default:
					// Other table options
				}
			}
		case tidbast.AlterTableAddColumns:
			for _, column := range spec.NewColumns {
				var pos *tidbast.ColumnPosition
				if len(spec.NewColumns) == 1 {
					pos = spec.Position
				}
				if err := table.createColumn(d.ctx, column, pos); err != nil {
					return err
				}
			}
			// MySQL/TiDB can add table constraints in ALTER TABLE ADD COLUMN statements.
			for _, constraint := range spec.NewConstraints {
				if err := table.createConstraint(d.ctx, constraint); err != nil {
					return err
				}
			}
		case tidbast.AlterTableAddConstraint:
			if err := table.createConstraint(d.ctx, spec.Constraint); err != nil {
				return err
			}
		case tidbast.AlterTableDropColumn:
			if err := table.dropColumn(d.ctx, spec.OldColumnName.Name.O); err != nil {
				return err
			}
		case tidbast.AlterTableDropPrimaryKey:
			if err := table.dropIndex(d.ctx, siPrimaryKeyName); err != nil {
				return err
			}
		case tidbast.AlterTableDropIndex:
			if err := table.dropIndex(d.ctx, spec.Name); err != nil {
				return err
			}
		case tidbast.AlterTableDropForeignKey:
			// we do not deal with DROP FOREIGN KEY statements.
		case tidbast.AlterTableModifyColumn:
			if err := table.changeColumn(d.ctx, spec.NewColumns[0].Name.Name.O, spec.NewColumns[0], spec.Position); err != nil {
				return err
			}
		case tidbast.AlterTableChangeColumn:
			if err := table.changeColumn(d.ctx, spec.OldColumnName.Name.O, spec.NewColumns[0], spec.Position); err != nil {
				return err
			}
		case tidbast.AlterTableRenameColumn:
			if err := table.renameColumn(d.ctx, spec.OldColumnName.Name.O, spec.NewColumnName.Name.O); err != nil {
				return err
			}
		case tidbast.AlterTableRenameTable:
			schema := d.schemaSet[""]
			if err := schema.renameTable(d.ctx, table.name, spec.NewTable.Name.O); err != nil {
				return err
			}
		case tidbast.AlterTableAlterColumn:
			if err := table.changeColumnDefault(d.ctx, spec.NewColumns[0]); err != nil {
				return err
			}
		case tidbast.AlterTableRenameIndex:
			if err := table.renameIndex(d.ctx, spec.FromKey.O, spec.ToKey.O); err != nil {
				return err
			}
		case tidbast.AlterTableIndexInvisible:
			if err := table.changeIndexVisibility(d.ctx, spec.IndexName.O, spec.Visibility); err != nil {
				return err
			}
		default:
			// Other ALTER TABLE types
		}
	}

	return nil
}

func (t *siTableState) changeIndexVisibility(ctx *siFinderContext, indexName string, visibility tidbast.IndexVisibility) error {
	index, exists := t.indexSet[strings.ToLower(indexName)]
	if !exists {
		if ctx.CheckIntegrity {
			return siNewSchemaViolationError(809, fmt.Sprintf("Index `%s` does not exist in table `%s`", indexName, t.name))
		}
		index = t.createIncompleteIndex(indexName)
	}
	switch visibility {
	case tidbast.IndexVisibilityVisible:
		index.visible = siNewTruePointer()
	case tidbast.IndexVisibilityInvisible:
		index.visible = siNewFalsePointer()
	default:
		// Keep current visibility
	}
	return nil
}

func (t *siTableState) renameIndex(ctx *siFinderContext, oldName string, newName string) error {
	// For MySQL/TiDB, the primary key has a special name 'PRIMARY'.
	// And the other indexes cannot use the name which case-insensitive equals 'PRIMARY'.
	if strings.ToUpper(oldName) == siPrimaryKeyName || strings.ToUpper(newName) == siPrimaryKeyName {
		incorrectName := oldName
		if strings.ToUpper(oldName) != siPrimaryKeyName {
			incorrectName = newName
		}
		return errors.Errorf("Incorrect index name `%s`", incorrectName)
	}

	index, exists := t.indexSet[strings.ToLower(oldName)]
	if !exists {
		if ctx.CheckIntegrity {
			return siNewSchemaViolationError(809, fmt.Sprintf("Index `%s` does not exist in table `%s`", oldName, t.name))
		}
		index = t.createIncompleteIndex(oldName)
	}

	if _, exists := t.indexSet[strings.ToLower(newName)]; exists {
		return siNewSchemaViolationError(805, fmt.Sprintf("Index `%s` already exists in table `%s`", newName, t.name))
	}

	index.name = newName
	delete(t.indexSet, strings.ToLower(oldName))
	t.indexSet[strings.ToLower(newName)] = index
	return nil
}

func (t *siTableState) changeColumnDefault(ctx *siFinderContext, column *tidbast.ColumnDef) error {
	columnName := column.Name.Name.L
	colState, exists := t.columnSet[columnName]
	if !exists {
		if ctx.CheckIntegrity {
			return siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.name))
		}
		colState = t.createIncompleteColumn(columnName)
	}

	if len(column.Options) == 1 {
		// SET DEFAULT
		if column.Options[0].Expr.GetType().GetType() != mysql.TypeNull {
			if colState.columnType != nil {
				switch strings.ToLower(*colState.columnType) {
				case "blob", "tinyblob", "mediumblob", "longblob",
					"text", "tinytext", "mediumtext", "longtext",
					"json",
					"geometry":
					return siNewSchemaViolationError(423, fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName))
				default:
					// Other column types allow default values
				}
			}

			defaultValue, err := siRestoreNode(column.Options[0].Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			colState.defaultValue = &defaultValue
		} else {
			if colState.nullable != nil && !*colState.nullable {
				return errors.Errorf("Invalid default value for column `%s`", columnName)
			}
			colState.defaultValue = nil
		}
	} else {
		// DROP DEFAULT
		colState.defaultValue = nil
	}
	return nil
}

func (t *siTableState) renameColumn(ctx *siFinderContext, oldName string, newName string) error {
	if strings.EqualFold(oldName, newName) {
		return nil
	}

	column, exists := t.columnSet[strings.ToLower(oldName)]
	if !exists {
		if ctx.CheckIntegrity {
			return siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", oldName, t.name))
		}
		column = t.createIncompleteColumn(oldName)
	}

	if _, exists := t.columnSet[strings.ToLower(newName)]; exists {
		return siNewSchemaViolationError(412, fmt.Sprintf("Column `%s` already exists in table `%s", newName, t.name))
	}

	column.name = newName
	delete(t.columnSet, strings.ToLower(oldName))
	t.columnSet[strings.ToLower(newName)] = column

	t.renameColumnInIndexKey(oldName, newName)
	return nil
}

func (t *siTableState) renameColumnInIndexKey(oldName string, newName string) {
	if strings.EqualFold(oldName, newName) {
		return
	}
	for _, index := range t.indexSet {
		for i, key := range index.expressionList {
			if strings.EqualFold(key, oldName) {
				index.expressionList[i] = newName
			}
		}
	}
}

func (t *siTableState) completeTableChangeColumn(ctx *siFinderContext, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) error {
	column, exists := t.columnSet[strings.ToLower(oldName)]
	if !exists {
		return siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", oldName, t.name))
	}

	pos := *column.position

	// generate Position struct for creating new column
	// Create a local copy to avoid modifying the input parameter
	var localPosition *tidbast.ColumnPosition
	if position == nil {
		localPosition = &tidbast.ColumnPosition{Tp: tidbast.ColumnPositionNone}
	} else {
		// Create a copy of the position to avoid modifying the original
		localPosition = &tidbast.ColumnPosition{
			Tp:             position.Tp,
			RelativeColumn: position.RelativeColumn,
		}
	}

	if localPosition.Tp == tidbast.ColumnPositionNone {
		if pos == 1 {
			localPosition.Tp = tidbast.ColumnPositionFirst
		} else {
			for _, col := range t.columnSet {
				if *col.position == pos-1 {
					localPosition.Tp = tidbast.ColumnPositionAfter
					localPosition.RelativeColumn = &tidbast.ColumnName{Name: tidbast.NewCIStr(col.name)}
					break
				}
			}
		}
	}
	position = localPosition

	// drop column from columnSet
	for _, col := range t.columnSet {
		if *col.position > pos {
			*col.position--
		}
	}
	delete(t.columnSet, strings.ToLower(column.name))

	// rename column from indexSet
	t.renameColumnInIndexKey(oldName, newColumn.Name.Name.O)

	// create a new column in columnSet
	return t.createColumn(ctx, newColumn, position)
}

func (t *siTableState) incompleteTableChangeColumn(ctx *siFinderContext, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) error {
	delete(t.columnSet, strings.ToLower(oldName))

	// rename column from indexSet
	t.renameColumnInIndexKey(oldName, newColumn.Name.Name.O)

	// create a new column in columnSet
	return t.createColumn(ctx, newColumn, position)
}

func (t *siTableState) changeColumn(ctx *siFinderContext, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) error {
	if ctx.CheckIntegrity {
		return t.completeTableChangeColumn(ctx, oldName, newColumn, position)
	}
	return t.incompleteTableChangeColumn(ctx, oldName, newColumn, position)
}

func (t *siTableState) dropIndex(ctx *siFinderContext, indexName string) error {
	if ctx.CheckIntegrity {
		if _, exists := t.indexSet[strings.ToLower(indexName)]; !exists {
			if strings.EqualFold(indexName, siPrimaryKeyName) {
				return siNewSchemaViolationError(808, fmt.Sprintf("Primary key does not exist in table `%s`", t.name))
			}
			return siNewSchemaViolationError(809, fmt.Sprintf("Index `%s` does not exist in table `%s`", indexName, t.name))
		}
	}

	delete(t.indexSet, strings.ToLower(indexName))
	return nil
}

func (t *siTableState) completeTableDropColumn(columnName string) error {
	column, exists := t.columnSet[strings.ToLower(columnName)]
	if !exists {
		return siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.name))
	}

	// Cannot drop all columns in a table using ALTER TABLE DROP COLUMN.
	if len(t.columnSet) == 1 {
		return errors.Errorf("Can't delete all columns with ALTER TABLE; use DROP TABLE %s instead", t.name)
	}

	// If columns are dropped from a table, the columns are also removed from any index of which they are a part.
	for _, index := range t.indexSet {
		index.dropColumn(columnName)
		// If all columns that make up an index are dropped, the index is dropped as well.
		if len(index.expressionList) == 0 {
			delete(t.indexSet, strings.ToLower(index.name))
		}
	}

	// modify the column position
	for _, col := range t.columnSet {
		if *col.position > *column.position {
			*col.position--
		}
	}

	delete(t.columnSet, strings.ToLower(columnName))
	return nil
}

func (t *siTableState) incompleteTableDropColumn(columnName string) error {
	// If columns are dropped from a table, the columns are also removed from any index of which they are a part.
	for _, index := range t.indexSet {
		if len(index.expressionList) == 0 {
			continue
		}
		index.dropColumn(columnName)
		// If all columns that make up an index are dropped, the index is dropped as well.
		if len(index.expressionList) == 0 {
			delete(t.indexSet, strings.ToLower(index.name))
		}
	}

	delete(t.columnSet, strings.ToLower(columnName))
	return nil
}

func (t *siTableState) dropColumn(ctx *siFinderContext, columnName string) error {
	if ctx.CheckIntegrity {
		return t.completeTableDropColumn(columnName)
	}
	return t.incompleteTableDropColumn(columnName)
}

func (idx *siIndexState) dropColumn(columnName string) {
	if len(idx.expressionList) == 0 {
		return
	}
	var newKeyList []string
	for _, key := range idx.expressionList {
		if !strings.EqualFold(key, columnName) {
			newKeyList = append(newKeyList, key)
		}
	}

	idx.expressionList = newKeyList
}

func (t *siTableState) reorderColumn(position *tidbast.ColumnPosition) (int, error) {
	switch position.Tp {
	case tidbast.ColumnPositionNone:
		return len(t.columnSet) + 1, nil
	case tidbast.ColumnPositionFirst:
		for _, column := range t.columnSet {
			*column.position++
		}
		return 1, nil
	case tidbast.ColumnPositionAfter:
		columnName := position.RelativeColumn.Name.L
		column, exist := t.columnSet[columnName]
		if !exist {
			return 0, siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.name))
		}
		for _, col := range t.columnSet {
			if *col.position > *column.position {
				*col.position++
			}
		}
		return *column.position + 1, nil
	default:
		return 0, errors.Errorf("Unsupported column position type: %d", position.Tp)
	}
}

func (d *siDatabaseState) dropTable(node *tidbast.DropTableStmt) error {
	if !node.IsView {
		for _, name := range node.Tables {
			if name.Schema.O != "" && !d.isCurrentDatabase(name.Schema.O) {
				return siNewSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", name.Schema.O, d.name))
			}

			schema, exists := d.schemaSet[""]
			if !exists {
				schema = d.createSchema()
			}

			table, exists := schema.getTable(name.Name.O)
			if !exists {
				if node.IfExists || !d.ctx.CheckIntegrity {
					return nil
				}
				return siNewSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", name.Name.O))
			}

			delete(schema.tableSet, table.name)
		}
	}
	return nil
}

func (d *siDatabaseState) copyTable(node *tidbast.CreateTableStmt) error {
	targetTable, err := d.findTableState(node.ReferTable)
	if err != nil {
		if strings.Contains(err.Error(), "is not the current database") {
			return errors.Errorf("Reference table `%s` in other database `%s`, skip walkthrough", node.ReferTable.Name.O, node.ReferTable.Schema.O)
		}
	}

	schema := d.schemaSet[""]
	table := targetTable.copy()
	table.name = node.Table.Name.O
	schema.tableSet[table.name] = table
	return nil
}

func (d *siDatabaseState) createTable(node *tidbast.CreateTableStmt) error {
	if node.Table.Schema.O != "" && !d.isCurrentDatabase(node.Table.Schema.O) {
		return siNewSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", node.Table.Schema.O, d.name))
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema()
	}

	if _, exists = schema.getTable(node.Table.Name.O); exists {
		if node.IfNotExists {
			return nil
		}
		return siNewSchemaViolationError(607, fmt.Sprintf("Table `%s` already exists", node.Table.Name.O))
	}

	if node.Select != nil {
		return siNewSchemaViolationError(205, fmt.Sprintf("Disallow the CREATE TABLE AS statement but \"%s\" uses", node.Text()))
	}

	if node.ReferTable != nil {
		return d.copyTable(node)
	}

	table := &siTableState{
		name:      node.Table.Name.O,
		engine:    siNewEmptyStringPointer(),
		collation: siNewEmptyStringPointer(),
		comment:   siNewEmptyStringPointer(),
		columnSet: make(siColumnStateMap),
		indexSet:  make(siIndexStateMap),
	}
	schema.tableSet[table.name] = table
	hasAutoIncrement := false

	for _, column := range node.Cols {
		if siIsAutoIncrement(column) {
			if hasAutoIncrement {
				return siNewSchemaViolationError(1, fmt.Sprintf("There can be only one auto column for table `%s`", table.name))
			}
			hasAutoIncrement = true
		}
		if err := table.createColumn(d.ctx, column, nil /* position */); err != nil {
			return err
		}
	}

	for _, constraint := range node.Constraints {
		if err := table.createConstraint(d.ctx, constraint); err != nil {
			return err
		}
	}

	return nil
}

func (t *siTableState) createConstraint(ctx *siFinderContext, constraint *tidbast.Constraint) error {
	switch constraint.Tp {
	case tidbast.ConstraintPrimaryKey:
		keyList, err := t.validateAndGetKeyStringList(ctx, constraint.Keys, true /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createPrimaryKey(keyList, siGetIndexType(constraint.Option)); err != nil {
			return err
		}
	case tidbast.ConstraintKey, tidbast.ConstraintIndex:
		keyList, err := t.validateAndGetKeyStringList(ctx, constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, false /* unique */, siGetIndexType(constraint.Option), constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex:
		keyList, err := t.validateAndGetKeyStringList(ctx, constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, true /* unique */, siGetIndexType(constraint.Option), constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintForeignKey:
		// we do not deal with FOREIGN KEY constraints
	case tidbast.ConstraintFulltext:
		keyList, err := t.validateAndGetKeyStringList(ctx, constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, false /* unique */, siFullTextName, constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintCheck:
		// we do not deal with CHECK constraints
	default:
		// Other constraint types
	}

	return nil
}

func (t *siTableState) validateAndGetKeyStringList(ctx *siFinderContext, keyList []*tidbast.IndexPartSpecification, primary bool, isSpatial bool) ([]string, error) {
	var res []string
	for _, key := range keyList {
		if key.Expr != nil {
			str, err := siRestoreNode(key, format.DefaultRestoreFlags)
			if err != nil {
				return nil, err
			}
			res = append(res, str)
		} else {
			columnName := key.Column.Name.L
			column, exists := t.columnSet[columnName]
			if !exists {
				if ctx.CheckIntegrity {
					return nil, siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.name))
				}
			} else {
				if primary {
					column.nullable = siNewFalsePointer()
				}
				if isSpatial && column.nullable != nil && *column.nullable {
					return nil, errors.Errorf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.name)
				}
			}

			res = append(res, columnName)
		}
	}
	return res, nil
}

func siIsAutoIncrement(column *tidbast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionAutoIncrement {
			return true
		}
	}
	return false
}

func siCheckDefault(columnName string, columnType *types.FieldType, value tidbast.ExprNode) error {
	if value.GetType().GetType() != mysql.TypeNull {
		switch columnType.GetType() {
		case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob, mysql.TypeJSON, mysql.TypeGeometry:
			return siNewSchemaViolationError(423, fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName))
		default:
			// Other column types allow default values
		}
	}

	if valueExpr, yes := value.(tidbast.ValueExpr); yes {
		datum := types.NewDatum(valueExpr.GetValue())
		if _, err := datum.ConvertTo(types.Context{}, columnType); err != nil {
			return errors.Errorf("%s", err.Error())
		}
	}
	return nil
}

func (t *siTableState) createColumn(ctx *siFinderContext, column *tidbast.ColumnDef, position *tidbast.ColumnPosition) error {
	if _, exists := t.columnSet[column.Name.Name.L]; exists {
		return siNewSchemaViolationError(412, fmt.Sprintf("Column `%s` already exists in table `%s`", column.Name.Name.O, t.name))
	}

	pos := len(t.columnSet) + 1
	if position != nil && ctx.CheckIntegrity {
		var err error
		pos, err = t.reorderColumn(position)
		if err != nil {
			return err
		}
	}

	vTrue := true
	col := &siColumnState{
		name:         column.Name.Name.L,
		position:     &pos,
		defaultValue: nil,
		nullable:     &vTrue,
		columnType:   siNewStringPointer(column.Tp.CompactStr()),
		characterSet: siNewStringPointer(column.Tp.GetCharset()),
		collation:    siNewStringPointer(column.Tp.GetCollate()),
		comment:      siNewEmptyStringPointer(),
	}
	setNullDefault := false

	for _, option := range column.Options {
		switch option.Tp {
		case tidbast.ColumnOptionPrimaryKey:
			col.nullable = siNewFalsePointer()
			if err := t.createPrimaryKey([]string{col.name}, tidbast.IndexTypeBtree.String()); err != nil {
				return err
			}
		case tidbast.ColumnOptionNotNull:
			col.nullable = siNewFalsePointer()
		case tidbast.ColumnOptionAutoIncrement:
			// we do not deal with AUTO-INCREMENT
		case tidbast.ColumnOptionDefaultValue:
			if err := siCheckDefault(col.name, column.Tp, option.Expr); err != nil {
				return err
			}
			if option.Expr.GetType().GetType() != mysql.TypeNull {
				defaultValue, err := siRestoreNode(option.Expr, format.RestoreStringWithoutCharset)
				if err != nil {
					return err
				}
				col.defaultValue = &defaultValue
			} else {
				setNullDefault = true
			}
		case tidbast.ColumnOptionUniqKey:
			if err := t.createIndex("", []string{col.name}, true /* unique */, tidbast.IndexTypeBtree.String(), nil); err != nil {
				return err
			}
		case tidbast.ColumnOptionNull:
			col.nullable = siNewTruePointer()
		case tidbast.ColumnOptionOnUpdate:
			// we do not deal with ON UPDATE
			if column.Tp.GetType() != mysql.TypeDatetime && column.Tp.GetType() != mysql.TypeTimestamp {
				return errors.Errorf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", col.name)
			}
		case tidbast.ColumnOptionComment:
			comment, err := siRestoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			col.comment = &comment
		case tidbast.ColumnOptionGenerated:
			// we do not deal with GENERATED ALWAYS AS
		case tidbast.ColumnOptionReference:
			// MySQL/TiDB will ignore the inline REFERENCE
			// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
		case tidbast.ColumnOptionCollate:
			col.collation = siNewStringPointer(option.StrValue)
		case tidbast.ColumnOptionCheck:
			// we do not deal with CHECK constraint
		case tidbast.ColumnOptionColumnFormat:
			// we do not deal with COLUMN_FORMAT
		case tidbast.ColumnOptionStorage:
			// we do not deal with STORAGE
		case tidbast.ColumnOptionAutoRandom:
			// we do not deal with AUTO_RANDOM
		default:
			// Other column options
		}
	}

	if col.nullable != nil && !*col.nullable && setNullDefault {
		return errors.Errorf("Invalid default value for column `%s`", col.name)
	}

	t.columnSet[strings.ToLower(col.name)] = col
	return nil
}

func (t *siTableState) createIndex(name string, keyList []string, unique bool, tp string, option *tidbast.IndexOption) error {
	if len(keyList) == 0 {
		return errors.Errorf("Index `%s` in table `%s` has empty key", name, t.name)
	}
	if name != "" {
		if _, exists := t.indexSet[strings.ToLower(name)]; exists {
			return siNewSchemaViolationError(805, fmt.Sprintf("Index `%s` already exists in table `%s`", name, t.name))
		}
	} else {
		suffix := 1
		for {
			name = keyList[0]
			if suffix > 1 {
				name = fmt.Sprintf("%s_%d", keyList[0], suffix)
			}
			if _, exists := t.indexSet[strings.ToLower(name)]; !exists {
				break
			}
			suffix++
		}
	}

	index := &siIndexState{
		name:           name,
		expressionList: keyList,
		indexType:      &tp,
		unique:         &unique,
		primary:        siNewFalsePointer(),
		visible:        siNewTruePointer(),
		comment:        siNewEmptyStringPointer(),
	}

	if option != nil && option.Visibility == tidbast.IndexVisibilityInvisible {
		index.visible = siNewFalsePointer()
	}

	t.indexSet[strings.ToLower(name)] = index
	return nil
}

func (t *siTableState) createPrimaryKey(keys []string, tp string) error {
	if _, exists := t.indexSet[strings.ToLower(siPrimaryKeyName)]; exists {
		return errors.Errorf("Primary key exists in table `%s`", t.name)
	}

	pk := &siIndexState{
		name:           siPrimaryKeyName,
		expressionList: keys,
		indexType:      &tp,
		unique:         siNewTruePointer(),
		primary:        siNewTruePointer(),
		visible:        siNewTruePointer(),
		comment:        siNewEmptyStringPointer(),
	}
	t.indexSet[strings.ToLower(pk.name)] = pk
	return nil
}

func siRestoreNode(node tidbast.Node, flag format.RestoreFlags) (string, error) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", errors.Errorf("%s", err.Error())
	}
	return buffer.String(), nil
}

func siGetIndexType(option *tidbast.IndexOption) string {
	if option != nil {
		switch option.Tp {
		case tidbast.IndexTypeBtree,
			tidbast.IndexTypeHash,
			tidbast.IndexTypeRtree:
			return option.Tp.String()
		default:
			// Other index types
		}
	}
	return tidbast.IndexTypeBtree.String()
}
