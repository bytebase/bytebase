package mysql

// Schema Integrity Advisor - validates MySQL DDL statements

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysqlantlr "github.com/bytebase/parser/mysql"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	tidbmysql "github.com/pingcap/tidb/pkg/parser/mysql"
	tidbtypes "github.com/pingcap/tidb/pkg/types"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

const (
	siPrimaryKeyName = "PRIMARY"
	fullTextName     = "FULLTEXT"
	spatialName      = "SPATIAL"
)

var (
	_ advisor.Advisor = (*SchemaIntegrityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleSchemaIntegrity, &SchemaIntegrityAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleSchemaIntegrity, &SchemaIntegrityAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleSchemaIntegrity, &SchemaIntegrityAdvisor{})
}

type SchemaIntegrityAdvisor struct {
}

func (*SchemaIntegrityAdvisor) Check(_ context.Context, ctx advisor.Context) ([]*storepb.Advice, error) {
	nodeList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", ctx.AST)
	}

	dbState := newDatabaseStateFromCatalog(ctx.DBSchema)

	for _, node := range nodeList {
		if err := dbState.mysqlChangeState(node); err != nil {
			if sve, ok := err.(*schemaViolationError); ok {
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

	// No violations found
	return make([]*storepb.Advice, 0), nil
}

type schemaViolationError struct {
	Code    int32
	Message string
}

func (e *schemaViolationError) Error() string {
	return e.Message
}

func newSchemaViolationError(code int32, message string) *schemaViolationError {
	return &schemaViolationError{
		Code:    code,
		Message: message,
	}
}

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

func newDatabaseStateFromCatalog(dbSchema *storepb.DatabaseSchemaMetadata) *siDatabaseState {
	db := &siDatabaseState{
		ctx: &siFinderContext{
			CheckIntegrity:      true,
			EngineType:          storepb.Engine_MYSQL,
			IgnoreCaseSensitive: true,
		},
		name:         "",
		characterSet: "",
		collation:    "",
		dbType:       storepb.Engine_MYSQL,
		schemaSet:    make(siSchemaStateMap),
		deleted:      false,
		usable:       true,
	}

	if dbSchema == nil {
		return db
	}

	db.name = dbSchema.Name
	db.characterSet = dbSchema.CharacterSet
	db.collation = dbSchema.Collation

	if len(dbSchema.Schemas) > 0 {
		schema := dbSchema.Schemas[0]
		schemaState := &siSchemaState{
			ctx:           db.ctx.Copy(),
			name:          "",
			tableSet:      make(siTableStateMap),
			viewSet:       make(siViewStateMap),
			identifierMap: make(siIdentifierMap),
		}

		for _, table := range schema.Tables {
			tableState := &siTableState{
				name:      table.Name,
				engine:    newStringPointer(table.Engine),
				collation: newStringPointer(table.Collation),
				comment:   newStringPointer(table.Comment),
				columnSet: make(siColumnStateMap),
				indexSet:  make(siIndexStateMap),
			}

			for i, column := range table.Columns {
				columnName := strings.ToLower(column.Name)
				colState := &siColumnState{
					name:         column.Name,
					position:     newIntPointer(i + 1),
					nullable:     newBoolPointer(column.Nullable),
					columnType:   newStringPointer(column.Type),
					characterSet: newStringPointer(column.CharacterSet),
					collation:    newStringPointer(column.Collation),
					comment:      newStringPointer(column.Comment),
				}
				if column.Default != "" {
					colState.defaultValue = copyStringPointer(&column.Default)
				}
				tableState.columnSet[columnName] = colState
			}

			for _, index := range table.Indexes {
				indexName := strings.ToLower(index.Name)
				indexState := &siIndexState{
					name:           index.Name,
					expressionList: copyStringSlice(index.Expressions),
					indexType:      newStringPointer(index.Type),
					unique:         newBoolPointer(index.Unique),
					primary:        newBoolPointer(index.Primary),
					visible:        newBoolPointer(index.Visible),
					comment:        newStringPointer(index.Comment),
					isConstraint:   index.Primary || index.Unique,
				}
				tableState.indexSet[indexName] = indexState
			}

			schemaState.tableSet[table.Name] = tableState
		}

		db.schemaSet[""] = schemaState
	}

	return db
}

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

type siSchemaState struct {
	ctx           *siFinderContext
	name          string
	tableSet      siTableStateMap
	viewSet       siViewStateMap
	identifierMap siIdentifierMap
}

type siTableState struct {
	name string
	// engine isn't supported for Postgres, Snowflake, SQLite.
	engine *string
	// collation isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	collation *string
	// comment isn't supported for SQLite.
	comment   *string
	columnSet siColumnStateMap
	// indexSet isn't supported for ClickHouse, Snowflake.
	indexSet siIndexStateMap

	// dependencyView is used to record the dependency view for the table.
	// Used to check if the table is used by any view.
	dependencyView map[string]bool // nolint:unused
}

type siColumnState struct {
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
	dependencyView map[string]bool // nolint:unused
}

type siIndexState struct {
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

type siViewState struct {
	name       string  // nolint:unused
	definition *string // nolint:unused
	comment    *string // nolint:unused
}

type siSchemaStateMap map[string]*siSchemaState

type siTableStateMap map[string]*siTableState

type siColumnStateMap map[string]*siColumnState

type siIndexStateMap map[string]*siIndexState

type siViewStateMap map[string]*siViewState

type siIdentifierMap map[string]bool

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

func newEmptyStringPointer() *string {
	res := ""
	return &res
}

func (t *siTableState) copy() *siTableState {
	return &siTableState{
		name:      t.name,
		engine:    copyStringPointer(t.engine),
		collation: copyStringPointer(t.collation),
		comment:   copyStringPointer(t.comment),
		columnSet: t.columnSet.copy(),
		indexSet:  t.indexSet.copy(),
	}
}

func (col *siColumnState) copy() *siColumnState {
	return &siColumnState{
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

func (idx *siIndexState) copy() *siIndexState {
	return &siIndexState{
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

func (m siColumnStateMap) copy() siColumnStateMap {
	res := make(siColumnStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

func (m siIndexStateMap) copy() siIndexStateMap {
	res := make(siIndexStateMap)
	for k, v := range m {
		res[k] = v.copy()
	}
	return res
}

func compareIdentifier(a, b string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func (d *siDatabaseState) isCurrentDatabase(database string) bool {
	return compareIdentifier(d.name, database, d.ctx.IgnoreCaseSensitive)
}

func (s *siSchemaState) getTable(table string) (*siTableState, bool) {
	for k, v := range s.tableSet {
		if compareIdentifier(k, table, s.ctx.IgnoreCaseSensitive) {
			return v, true
		}
	}

	return nil, false
}

func (d *siDatabaseState) createSchema() *siSchemaState {
	schema := &siSchemaState{
		ctx:      d.ctx.Copy(),
		name:     "",
		tableSet: make(siTableStateMap),
		viewSet:  make(siViewStateMap),
	}

	d.schemaSet[""] = schema
	return schema
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

func (s *siSchemaState) renameTable(ctx *siFinderContext, oldName string, newName string) error {
	if oldName == newName {
		return nil
	}

	table, exists := s.getTable(oldName)
	if !exists {
		if ctx.CheckIntegrity {
			return newSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", oldName))
		}
		table = s.createIncompleteTable(oldName)
	}

	if _, exists := s.getTable(newName); exists {
		return newSchemaViolationError(607, fmt.Sprintf("Table `%s` already exists", newName))
	}

	table.name = newName
	delete(s.tableSet, oldName)
	s.tableSet[newName] = table
	return nil
}

func (t *siTableState) renameIndex(ctx *siFinderContext, oldName string, newName string) error {
	// For MySQL, the primary key has a special name 'PRIMARY'.
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
			return newSchemaViolationError(809, fmt.Sprintf("Index `%s` does not exist in table `%s`", oldName, t.name))
		}
		index = t.createIncompleteIndex(oldName)
	}

	if _, exists := t.indexSet[strings.ToLower(newName)]; exists {
		return newSchemaViolationError(805, fmt.Sprintf("Index `%s` already exists in table `%s`", newName, t.name))
	}

	index.name = newName
	delete(t.indexSet, strings.ToLower(oldName))
	t.indexSet[strings.ToLower(newName)] = index
	return nil
}

func (t *siTableState) renameColumn(ctx *siFinderContext, oldName string, newName string) error {
	if strings.EqualFold(oldName, newName) {
		return nil
	}

	column, exists := t.columnSet[strings.ToLower(oldName)]
	if !exists {
		if ctx.CheckIntegrity {
			return newSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", oldName, t.name))
		}
		column = t.createIncompleteColumn(oldName)
	}

	if _, exists := t.columnSet[strings.ToLower(newName)]; exists {
		return newSchemaViolationError(412, fmt.Sprintf("Column `%s` already exists in table `%s", newName, t.name))
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

func (t *siTableState) dropColumn(ctx *siFinderContext, columnName string) error {
	if ctx.CheckIntegrity {
		return t.completeTableDropColumn(columnName)
	}
	return t.incompleteTableDropColumn(columnName)
}

func (t *siTableState) dropIndex(ctx *siFinderContext, indexName string) error {
	if ctx.CheckIntegrity {
		if _, exists := t.indexSet[strings.ToLower(indexName)]; !exists {
			if strings.EqualFold(indexName, siPrimaryKeyName) {
				return newSchemaViolationError(808, fmt.Sprintf("Primary key does not exist in table `%s`", t.name))
			}
			return newSchemaViolationError(809, fmt.Sprintf("Index `%s` does not exist in table `%s`", indexName, t.name))
		}
	}

	delete(t.indexSet, strings.ToLower(indexName))
	return nil
}

func (t *siTableState) completeTableDropColumn(columnName string) error {
	column, exists := t.columnSet[strings.ToLower(columnName)]
	if !exists {
		return newSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.name))
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

type siMysqlListener struct {
	*mysqlantlr.BaseMySQLParserListener

	baseLine      int
	lineNumber    int
	text          string
	databaseState *siDatabaseState
	err           error
}

func (l *siMysqlListener) EnterQuery(ctx *mysqlantlr.QueryContext) {
	l.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	l.lineNumber = l.baseLine + ctx.GetStart().GetLine()
}

func (d *siDatabaseState) mysqlChangeState(in *mysqlparser.ParseResult) error {
	if d.deleted {
		return newSchemaViolationError(703, fmt.Sprintf("Database `%s` is deleted", d.name))
	}

	listener := &siMysqlListener{
		baseLine:      in.BaseLine,
		databaseState: d,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, in.Tree)
	return listener.err
}

// EnterCreateTable is called when production createTable is entered.
func (l *siMysqlListener) EnterCreateTable(ctx *mysqlantlr.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if databaseName != "" && !l.databaseState.isCurrentDatabase(databaseName) {
		l.err = newSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, l.databaseState.name))
		return
	}

	schema, exists := l.databaseState.schemaSet[""]
	if !exists {
		schema = l.databaseState.createSchema()
	}
	if _, exists = schema.getTable(tableName); exists {
		if ctx.IfNotExists() != nil {
			return
		}
		l.err = newSchemaViolationError(607, fmt.Sprintf("Table `%s` already exists", tableName))
		return
	}

	if ctx.DuplicateAsQueryExpression() != nil {
		l.err = newSchemaViolationError(205, fmt.Sprintf("Disallow the CREATE TABLE AS statement but \"%s\" uses", l.text))
		return
	}

	if ctx.LIKE_SYMBOL() != nil {
		_, referTable := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
		l.err = l.databaseState.mysqlCopyTable(databaseName, tableName, referTable)
		return
	}

	table := &siTableState{
		name:      tableName,
		engine:    newEmptyStringPointer(),
		collation: newEmptyStringPointer(),
		comment:   newEmptyStringPointer(),
		columnSet: make(siColumnStateMap),
		indexSet:  make(siIndexStateMap),
	}
	schema.tableSet[table.name] = table

	if ctx.TableElementList() == nil {
		return
	}

	hasAutoIncrement := false
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		switch {
		// handle column
		case tableElement.ColumnDefinition() != nil:
			if tableElement.ColumnDefinition().FieldDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil {
				continue
			}
			if mysqlparser.IsAutoIncrement(tableElement.ColumnDefinition().FieldDefinition()) {
				if hasAutoIncrement {
					l.err = newSchemaViolationError(1, fmt.Sprintf("There can be only one auto column for table `%s`", table.name))
				}
				hasAutoIncrement = true
			}
			_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
			if err := table.mysqlCreateColumn(l.databaseState.ctx, columnName, tableElement.ColumnDefinition().FieldDefinition(), nil /* position */); err != nil {
				l.err = err
				return
			}
		case tableElement.TableConstraintDef() != nil:
			if err := table.mysqlCreateConstraint(l.databaseState.ctx, tableElement.TableConstraintDef()); err != nil {
				l.err = err
				return
			}
		default:
			// Ignore other table element types
		}
	}
}

// EnterDropTable is called when production dropTable is entered.
func (l *siMysqlListener) EnterDropTable(ctx *mysqlantlr.DropTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRefList() == nil {
		return
	}

	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		if databaseName != "" && !l.databaseState.isCurrentDatabase(databaseName) {
			l.err = newSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, tableName))
		}

		schema, exists := l.databaseState.schemaSet[""]
		if !exists {
			schema = l.databaseState.createSchema()
		}

		table, exists := schema.getTable(tableName)
		if !exists {
			if ctx.IfExists() != nil || !l.databaseState.ctx.CheckIntegrity {
				return
			}
			l.err = newSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", tableName))
			return
		}

		delete(schema.tableSet, table.name)
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (l *siMysqlListener) EnterAlterTable(ctx *mysqlantlr.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRef() == nil {
		// todo: maybe need to do error handle.
		return
	}

	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	table, err := l.databaseState.mysqlFindTableState(databaseName, tableName)
	if err != nil {
		l.err = err
		return
	}

	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllCreateTableOptionsSpaceSeparated() {
		for _, op := range option.AllCreateTableOption() {
			switch {
			// engine.
			case op.ENGINE_SYMBOL() != nil:
				if op.EngineRef() == nil {
					continue
				}
				engine := op.EngineRef().GetText()
				table.engine = newStringPointer(engine)
			// table comment.
			case op.COMMENT_SYMBOL() != nil && op.TextStringLiteral() != nil:
				comment := mysqlparser.NormalizeMySQLTextStringLiteral(op.TextStringLiteral())
				table.comment = newStringPointer(comment)
			// table collation.
			case op.DefaultCollation() != nil && op.DefaultCollation().CollationName() != nil:
				collation := mysqlparser.NormalizeMySQLCollationName(op.DefaultCollation().CollationName())
				table.collation = newStringPointer(collation)
			default:
			}
		}
	}

	// alter table add column, change column, modify column.
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		switch {
		case item.ADD_SYMBOL() != nil:
			switch {
			// add single column.
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				if err := table.mysqlCreateColumn(l.databaseState.ctx, columnName, item.FieldDefinition(), positionFromPlaceContext(item.Place())); err != nil {
					l.err = err
					return
				}
			// add multi columns.
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					if err := table.mysqlCreateColumn(l.databaseState.ctx, columnName, tableElement.ColumnDefinition().FieldDefinition(), nil); err != nil {
						l.err = err
						return
					}
				}
			// add constraint.
			case item.TableConstraintDef() != nil:
				if err := table.mysqlCreateConstraint(l.databaseState.ctx, item.TableConstraintDef()); err != nil {
					l.err = err
					return
				}
			default:
				// Ignore other ADD variations
			}
		// drop column or key.
		case item.DROP_SYMBOL() != nil && item.ALTER_SYMBOL() == nil:
			switch {
			// drop foreign key.
			// we do not deal with DROP FOREIGN KEY statements.
			case item.FOREIGN_SYMBOL() != nil && item.KEY_SYMBOL() != nil:
			// drop column.
			case item.ColumnInternalRef() != nil:
				columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
				if err := table.dropColumn(l.databaseState.ctx, columnName); err != nil {
					l.err = err
					return
				}
				// drop primary key.
			case item.PRIMARY_SYMBOL() != nil && item.KEY_SYMBOL() != nil:
				if err := table.dropIndex(l.databaseState.ctx, siPrimaryKeyName); err != nil {
					l.err = err
					return
				}
				// drop key/index.
			case item.KeyOrIndex() != nil && item.IndexRef() != nil:
				_, _, indexName := mysqlparser.NormalizeIndexRef(item.IndexRef())
				if err := table.dropIndex(l.databaseState.ctx, indexName); err != nil {
					l.err = err
					return
				}
			default:
				// Ignore other DROP variations
			}
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			if err := table.mysqlChangeColumn(l.databaseState.ctx, columnName, columnName, item.FieldDefinition(), positionFromPlaceContext(item.Place())); err != nil {
				l.err = err
				return
			}
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			if err := table.mysqlChangeColumn(l.databaseState.ctx, oldColumnName, newColumnName, item.FieldDefinition(), positionFromPlaceContext(item.Place())); err != nil {
				l.err = err
				return
			}
		// rename column
		case item.RENAME_SYMBOL() != nil && item.COLUMN_SYMBOL() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			if err := table.renameColumn(l.databaseState.ctx, oldColumnName, newColumnName); err != nil {
				l.err = err
				return
			}
		case item.ALTER_SYMBOL() != nil:
			switch {
			// alter column.
			case item.ColumnInternalRef() != nil:
				if err := table.mysqlAlterColumn(l.databaseState.ctx, item); err != nil {
					l.err = err
					return
				}
			// alter index visibility.
			case item.INDEX_SYMBOL() != nil && item.IndexRef() != nil && item.Visibility() != nil:
				_, _, indexName := mysqlparser.NormalizeIndexRef(item.IndexRef())
				if err := table.mysqlChangeIndexVisibility(l.databaseState.ctx, indexName, item.Visibility()); err != nil {
					l.err = err
					return
				}
			default:
			}
		// rename table.
		case item.RENAME_SYMBOL() != nil && item.TableName() != nil:
			_, newTableName := mysqlparser.NormalizeMySQLTableName(item.TableName())
			schema := l.databaseState.schemaSet[""]
			if err := schema.renameTable(l.databaseState.ctx, table.name, newTableName); err != nil {
				l.err = err
				return
			}
		// rename index.
		case item.RENAME_SYMBOL() != nil && item.KeyOrIndex() != nil && item.IndexRef() != nil && item.IndexName() != nil:
			_, _, oldIndexName := mysqlparser.NormalizeIndexRef(item.IndexRef())
			newIndexName := mysqlparser.NormalizeIndexName(item.IndexName())
			if err := table.renameIndex(l.databaseState.ctx, oldIndexName, newIndexName); err != nil {
				l.err = err
				return
			}
		default:
			// Ignore other alter table actions
		}
	}
}

// EnterDropIndex is called when production dropIndex is entered.
func (l *siMysqlListener) EnterDropIndex(ctx *mysqlantlr.DropIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRef() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	table, err := l.databaseState.mysqlFindTableState(databaseName, tableName)
	if err != nil {
		l.err = err
		return
	}

	if ctx.IndexRef() == nil {
		return
	}

	_, _, indexName := mysqlparser.NormalizeIndexRef(ctx.IndexRef())
	if err := table.dropIndex(l.databaseState.ctx, indexName); err != nil {
		l.err = err
	}
}

func (l *siMysqlListener) EnterCreateIndex(ctx *mysqlantlr.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	table, err := l.databaseState.mysqlFindTableState(databaseName, tableName)
	if err != nil {
		l.err = err
		return
	}

	unique := false
	isSpatial := false
	tp := "BTREE"

	if ctx.GetType_() == nil {
		return
	}
	switch ctx.GetType_().GetTokenType() {
	case mysqlantlr.MySQLParserFULLTEXT_SYMBOL:
		tp = fullTextName
	case mysqlantlr.MySQLParserSPATIAL_SYMBOL:
		isSpatial = true
		tp = spatialName
	case mysqlantlr.MySQLParserINDEX_SYMBOL:
	default:
		// Other index types
	}
	if ctx.UNIQUE_SYMBOL() != nil {
		unique = true
	}

	indexName := ""
	if ctx.IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexName())
	}
	if ctx.IndexNameAndType() != nil && ctx.IndexNameAndType().IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexNameAndType().IndexName())
	}

	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().KeyListVariants() == nil {
		return
	}
	if err := table.mysqlValidateKeyListVariants(l.databaseState.ctx, ctx.CreateIndexTarget().KeyListVariants(), false /* primary */, isSpatial); err != nil {
		l.err = err
		return
	}

	columnList := mysqlparser.NormalizeKeyListVariants(ctx.CreateIndexTarget().KeyListVariants())
	if err := table.mysqlCreateIndex(indexName, columnList, unique, tp, mysqlantlr.NewEmptyTableConstraintDefContext(), ctx); err != nil {
		l.err = err
		return
	}
}

// EnterAlterDatabase is called when production alterDatabase is entered.
func (l *siMysqlListener) EnterAlterDatabase(ctx *mysqlantlr.AlterDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.SchemaRef() != nil {
		databaseName := mysqlparser.NormalizeMySQLSchemaRef(ctx.SchemaRef())
		if !l.databaseState.isCurrentDatabase(databaseName) {
			l.err = newSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, l.databaseState.name))
			return
		}
	}

	for _, option := range ctx.AllAlterDatabaseOption() {
		if option.CreateDatabaseOption() == nil {
			continue
		}

		switch {
		case option.CreateDatabaseOption().DefaultCharset() != nil && option.CreateDatabaseOption().DefaultCharset().CharsetName() != nil:
			charset := mysqlparser.NormalizeMySQLCharsetName(option.CreateDatabaseOption().DefaultCharset().CharsetName())
			l.databaseState.characterSet = charset
		case option.CreateDatabaseOption().DefaultCollation() != nil && option.CreateDatabaseOption().DefaultCollation().CollationName() != nil:
			collation := mysqlparser.NormalizeMySQLCollationName(option.CreateDatabaseOption().DefaultCollation().CollationName())
			l.databaseState.collation = collation
		default:
			// Other options
		}
	}
}

// EnterDropDatabase is called when production dropDatabase is entered.
func (l *siMysqlListener) EnterDropDatabase(ctx *mysqlantlr.DropDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.SchemaRef() == nil {
		return
	}

	databaseName := mysqlparser.NormalizeMySQLSchemaRef(ctx.SchemaRef())
	if !l.databaseState.isCurrentDatabase(databaseName) {
		l.err = newSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, l.databaseState.name))
		return
	}

	l.databaseState.deleted = true
}

// EnterCreateDatabase is called when production createDatabase is entered.
func (l *siMysqlListener) EnterCreateDatabase(ctx *mysqlantlr.CreateDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.SchemaName() == nil {
		return
	}
	databaseName := mysqlparser.NormalizeMySQLSchemaName(ctx.SchemaName())
	l.err = newSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, l.databaseState.name))
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (l *siMysqlListener) EnterRenameTableStatement(ctx *mysqlantlr.RenameTableStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	for _, pair := range ctx.AllRenamePair() {
		schema, exists := l.databaseState.schemaSet[""]
		if !exists {
			schema = l.databaseState.createSchema()
		}

		_, oldTableName := mysqlparser.NormalizeMySQLTableRef(pair.TableRef())
		_, newTableName := mysqlparser.NormalizeMySQLTableName(pair.TableName())

		if l.databaseState.mysqlTheCurrentDatabase(pair) {
			if compareIdentifier(oldTableName, newTableName, l.databaseState.ctx.IgnoreCaseSensitive) {
				return
			}
			table, exists := schema.getTable(oldTableName)
			if !exists {
				if schema.ctx.CheckIntegrity {
					l.err = newSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", oldTableName))
					return
				}
				table = schema.createIncompleteTable(oldTableName)
			}
			if _, exists := schema.getTable(newTableName); exists {
				l.err = newSchemaViolationError(607, fmt.Sprintf("Table `%s` already exists", newTableName))
				return
			}
			delete(schema.tableSet, table.name)
			table.name = newTableName
			schema.tableSet[table.name] = table
		} else if l.databaseState.mysqlMoveToOtherDatabase(pair) {
			_, exists := schema.getTable(oldTableName)
			if !exists && schema.ctx.CheckIntegrity {
				l.err = newSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", oldTableName))
				return
			}
			delete(schema.tableSet, oldTableName)
		} else {
			l.err = newSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", l.databaseState.mysqlTargetDatabase(pair), l.databaseState.name))
			return
		}
	}
}

func (l *siMysqlListener) EnterCreateTrigger(ctx *mysqlantlr.CreateTriggerContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TriggerName() == nil {
		return
	}

	// Check if related table exists.
	if ctx.TableRef() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	_, err := l.databaseState.mysqlFindTableState(databaseName, tableName)
	if err != nil {
		l.err = err
		return
	}
}

func (*siMysqlListener) EnterCreateProcedure(ctx *mysqlantlr.CreateProcedureContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.ProcedureName() == nil {
		return
	}
	// Skip other checks for now.
}

func (*siMysqlListener) EnterCreateEvent(ctx *mysqlantlr.CreateEventContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.EventName() == nil {
		return
	}
	// Skip other checks for now.
}

func (d *siDatabaseState) mysqlTargetDatabase(renamePair mysqlantlr.IRenamePairContext) string {
	oldDatabaseName, _ := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
	if oldDatabaseName != "" && !d.isCurrentDatabase(oldDatabaseName) {
		return oldDatabaseName
	}
	newDatabaseName, _ := mysqlparser.NormalizeMySQLTableName(renamePair.TableName())
	return newDatabaseName
}

func (d *siDatabaseState) mysqlMoveToOtherDatabase(renamePair mysqlantlr.IRenamePairContext) bool {
	oldDatabaseName, _ := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
	if oldDatabaseName != "" && !d.isCurrentDatabase(oldDatabaseName) {
		return false
	}
	newDatabaseName, _ := mysqlparser.NormalizeMySQLTableName(renamePair.TableName())
	return oldDatabaseName != newDatabaseName
}

func (d *siDatabaseState) mysqlTheCurrentDatabase(renamePair mysqlantlr.IRenamePairContext) bool {
	newDatabaseName, _ := mysqlparser.NormalizeMySQLTableName(renamePair.TableName())
	if newDatabaseName != "" && !d.isCurrentDatabase(newDatabaseName) {
		return false
	}
	oldDatabaseName, _ := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
	if oldDatabaseName != "" && !d.isCurrentDatabase(oldDatabaseName) {
		return false
	}
	return true
}

func (t *siTableState) mysqlChangeIndexVisibility(ctx *siFinderContext, indexName string, visibility mysqlantlr.IVisibilityContext) error {
	index, exists := t.indexSet[strings.ToLower(indexName)]
	if !exists {
		if ctx.CheckIntegrity {
			return newSchemaViolationError(809, fmt.Sprintf("Index `%s` does not exist in table `%s`", indexName, t.name))
		}
		index = t.createIncompleteIndex(indexName)
	}
	switch {
	case visibility.VISIBLE_SYMBOL() != nil:
		index.visible = newTruePointer()
	case visibility.INVISIBLE_SYMBOL() != nil:
		index.visible = newFalsePointer()
	default:
		// No visibility specified
	}
	return nil
}

func (t *siTableState) mysqlAlterColumn(ctx *siFinderContext, itemDef mysqlantlr.IAlterListItemContext) error {
	if itemDef.ColumnInternalRef() == nil {
		// should not reach here.
		return nil
	}
	columnName := mysqlparser.NormalizeMySQLColumnInternalRef(itemDef.ColumnInternalRef())
	colState, exists := t.columnSet[strings.ToLower(columnName)]
	if !exists {
		if ctx.CheckIntegrity {
			return newSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.name))
		}
		colState = t.createIncompleteColumn(columnName)
	}

	switch {
	case itemDef.SET_SYMBOL() != nil:
		switch {
		// SET DEFAULT.
		case itemDef.DEFAULT_SYMBOL() != nil:
			if itemDef.SignedLiteral() != nil && itemDef.SignedLiteral().Literal() != nil && itemDef.SignedLiteral().Literal().NullLiteral() == nil {
				if colState.columnType != nil {
					switch strings.ToLower(*colState.columnType) {
					case "blob", "tinyblob", "mediumblob", "longblob",
						"text", "tinytext", "mediumtext", "longtext",
						"json",
						"geometry":
						return newSchemaViolationError(423, fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName))
					default:
						// Other column types allow default values
					}
				}

				var defaultValue string
				switch {
				case itemDef.ExprWithParentheses() != nil:
					defaultValue = itemDef.ExprWithParentheses().GetText()
				case itemDef.SignedLiteral() != nil:
					defaultValue = itemDef.SignedLiteral().GetText()
				default:
					// No default value expression
				}

				colState.defaultValue = &defaultValue
			} else {
				if colState.nullable != nil && !*colState.nullable {
					return errors.Errorf("Invalid default value for column `%s`", columnName)
				}

				colState.defaultValue = nil
			}
		// SET VISIBLE/INVISIBLE.
		default:
		}
	case itemDef.DROP_SYMBOL() != nil && itemDef.DEFAULT_SYMBOL() != nil:
		// DROP DEFAULT.
		colState.defaultValue = nil
	default:
		// Other ALTER operations
	}
	return nil
}

func (t *siTableState) mysqlChangeColumn(ctx *siFinderContext, oldColumnName string, newColumnName string, fieldDef mysqlantlr.IFieldDefinitionContext, position *siMysqlColumnPosition) error {
	if ctx.CheckIntegrity {
		return t.mysqlCompleteTableChangeColumn(ctx, oldColumnName, newColumnName, fieldDef, position)
	}
	return t.mysqlIncompleteTableChangeColumn(ctx, oldColumnName, newColumnName, fieldDef, position)
}

// mysqlIncompleteTableChangeColumn changes column definition.
// It does not maintain the position of the column.
func (t *siTableState) mysqlIncompleteTableChangeColumn(ctx *siFinderContext, oldColumnName string, newColumnName string, fieldDef mysqlantlr.IFieldDefinitionContext, position *siMysqlColumnPosition) error {
	delete(t.columnSet, strings.ToLower(oldColumnName))

	// rename column from indexSet
	t.renameColumnInIndexKey(oldColumnName, newColumnName)

	// create a new column in columnSet
	return t.mysqlCreateColumn(ctx, newColumnName, fieldDef, position)
}

// mysqlCompleteTableChangeColumn changes column definition.
// It works as:
// 1. drop column from tableState.columnSet, but do not drop column from indexSet.
// 2. rename column from indexSet.
// 3. create a new column in columnSet.
func (t *siTableState) mysqlCompleteTableChangeColumn(ctx *siFinderContext, oldColumnName string, newColumnName string, fieldDef mysqlantlr.IFieldDefinitionContext, position *siMysqlColumnPosition) error {
	column, exists := t.columnSet[strings.ToLower(oldColumnName)]
	if !exists {
		return newSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", oldColumnName, t.name))
	}

	pos := *column.position

	if position == nil {
		position = &siMysqlColumnPosition{
			tp: siColumnPositionNone,
		}
	}
	if position.tp == siColumnPositionNone {
		if pos == 1 {
			position.tp = siColumnPositionFirst
		} else {
			for _, col := range t.columnSet {
				if *col.position == pos-1 {
					position.tp = siColumnPositionAfter
					position.relativeColumn = col.name
					break
				}
			}
		}
	}

	// drop column from columnSet.
	for _, col := range t.columnSet {
		if *col.position > pos {
			*col.position--
		}
	}
	delete(t.columnSet, strings.ToLower(column.name))

	// rename column from indexSet
	t.renameColumnInIndexKey(oldColumnName, newColumnName)

	// create a new column in columnSet
	return t.mysqlCreateColumn(ctx, newColumnName, fieldDef, position)
}

type siColumnPositionType int

const (
	siColumnPositionNone siColumnPositionType = iota
	siColumnPositionFirst
	siColumnPositionAfter
)

type siMysqlColumnPosition struct {
	tp             siColumnPositionType
	relativeColumn string
}

func positionFromPlaceContext(place mysqlantlr.IPlaceContext) *siMysqlColumnPosition {
	columnPosition := &siMysqlColumnPosition{
		tp: siColumnPositionNone,
	}
	if place, ok := place.(*mysqlantlr.PlaceContext); ok {
		if place != nil {
			switch {
			case place.FIRST_SYMBOL() != nil:
				columnPosition.tp = siColumnPositionFirst
			case place.AFTER_SYMBOL() != nil:
				columnPosition.tp = siColumnPositionAfter
				columnName := mysqlparser.NormalizeMySQLIdentifier(place.Identifier())
				columnPosition.relativeColumn = columnName
			default:
				// No position specified
			}
		}
	}
	return columnPosition
}

func (d *siDatabaseState) mysqlCopyTable(databaseName, tableName, referTable string) error {
	targetTable, err := d.mysqlFindTableState(databaseName, referTable)
	if err != nil {
		return err
	}

	schema := d.schemaSet[""]
	table := targetTable.copy()
	table.name = tableName
	schema.tableSet[table.name] = table
	return nil
}

func (d *siDatabaseState) mysqlFindTableState(databaseName, tableName string) (*siTableState, error) {
	if databaseName != "" && !d.isCurrentDatabase(databaseName) {
		return nil, newSchemaViolationError(702, fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, d.name))
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema()
	}

	table, exists := schema.getTable(tableName)
	if !exists {
		if schema.ctx.CheckIntegrity {
			return nil, newSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", tableName))
		}
		table = schema.createIncompleteTable(tableName)
	}

	return table, nil
}

func (t *siTableState) mysqlCreateConstraint(ctx *siFinderContext, constraintDef mysqlantlr.ITableConstraintDefContext) error {
	if constraintDef.GetType_() != nil {
		switch constraintDef.GetType_().GetTokenType() {
		// PRIMARY KEY.
		case mysqlantlr.MySQLParserPRIMARY_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := t.mysqlValidateKeyListVariants(ctx, constraintDef.KeyListVariants(), true /* primary */, false /* isSpatial*/); err != nil {
				return err
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := t.mysqlCreatePrimaryKey(keyList, mysqlGetIndexType(constraintDef)); err != nil {
				return err
			}
		// normal KEY/INDEX.
		case mysqlantlr.MySQLParserKEY_SYMBOL, mysqlantlr.MySQLParserINDEX_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := t.mysqlValidateKeyListVariants(ctx, constraintDef.KeyListVariants(), false /* primary */, false /* isSpatial */); err != nil {
				return err
			}

			indexName := ""
			if constraintDef.IndexNameAndType() != nil && constraintDef.IndexNameAndType().IndexName() != nil {
				indexName = mysqlparser.NormalizeIndexName(constraintDef.IndexNameAndType().IndexName())
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := t.mysqlCreateIndex(indexName, keyList, false /* unique */, mysqlGetIndexType(constraintDef), constraintDef, mysqlantlr.NewEmptyCreateIndexContext()); err != nil {
				return err
			}
		// UNIQUE KEY.
		case mysqlantlr.MySQLParserUNIQUE_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := t.mysqlValidateKeyListVariants(ctx, constraintDef.KeyListVariants(), false /* primary */, false /* isSpatial*/); err != nil {
				return err
			}

			indexName := ""
			if constraintDef.ConstraintName() != nil {
				indexName = mysqlparser.NormalizeConstraintName(constraintDef.ConstraintName())
			}
			if constraintDef.IndexNameAndType() != nil && constraintDef.IndexNameAndType().IndexName() != nil {
				indexName = mysqlparser.NormalizeIndexName(constraintDef.IndexNameAndType().IndexName())
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := t.mysqlCreateIndex(indexName, keyList, true /* unique */, mysqlGetIndexType(constraintDef), constraintDef, mysqlantlr.NewEmptyCreateIndexContext()); err != nil {
				return err
			}
		// FULLTEXT KEY.
		case mysqlantlr.MySQLParserFULLTEXT_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := t.mysqlValidateKeyListVariants(ctx, constraintDef.KeyListVariants(), false /* primary */, false /* isSpatial*/); err != nil {
				return err
			}
			indexName := ""
			if constraintDef.IndexName() != nil {
				indexName = mysqlparser.NormalizeIndexName(constraintDef.IndexName())
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := t.mysqlCreateIndex(indexName, keyList, false /* unique */, mysqlGetIndexType(constraintDef), constraintDef, mysqlantlr.NewEmptyCreateIndexContext()); err != nil {
				return err
			}
		case mysqlantlr.MySQLParserFOREIGN_SYMBOL:
			// we do not deal with FOREIGN KEY constraints.
		default:
			// Other constraint types
		}
	}

	// we do not deal with check constraints.
	// if constraintDef.CheckConstraint() != nil {}
	return nil
}

// mysqlValidateKeyListVariants validates the key list variants.
func (t *siTableState) mysqlValidateKeyListVariants(ctx *siFinderContext, keyList mysqlantlr.IKeyListVariantsContext, primary bool, isSpatial bool) error {
	if keyList.KeyList() != nil {
		columns := mysqlparser.NormalizeKeyList(keyList.KeyList())
		if err := t.mysqlValidateColumnList(ctx, columns, primary, isSpatial); err != nil {
			return err
		}
	}
	if keyList.KeyListWithExpression() != nil {
		expressions := mysqlparser.NormalizeKeyListWithExpression(keyList.KeyListWithExpression())
		if err := t.mysqlValidateExpressionList(ctx, expressions, primary, isSpatial); err != nil {
			return err
		}
	}
	return nil
}

func (t *siTableState) mysqlValidateColumnList(ctx *siFinderContext, columnList []string, primary bool, isSpatial bool) error {
	for _, columnName := range columnList {
		column, exists := t.columnSet[strings.ToLower(columnName)]
		if !exists {
			if ctx.CheckIntegrity {
				return newSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.name))
			}
		} else {
			if primary {
				column.nullable = newFalsePointer()
			}
			if isSpatial && column.nullable != nil && *column.nullable {
				return errors.Errorf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.name)
			}
		}
	}
	return nil
}

// mysqlValidateExpressionList validates the expression list.
// TODO: update expression validation.
func (t *siTableState) mysqlValidateExpressionList(_ *siFinderContext, expressionList []string, primary bool, isSpatial bool) error {
	for _, expression := range expressionList {
		column, exists := t.columnSet[strings.ToLower(expression)]
		// If expression is not a column, we do not need to validate it.
		if !exists {
			continue
		}

		if primary {
			column.nullable = newFalsePointer()
		}
		if isSpatial && column.nullable != nil && *column.nullable {
			return errors.Errorf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.name)
		}
	}
	return nil
}

func mysqlGetIndexType(tableConstraint mysqlantlr.ITableConstraintDefContext) string {
	if tableConstraint.GetType_() == nil {
		return "BTREE"
	}

	// I still need to handle IndexNameAndType to get index type(algorithm).
	switch tableConstraint.GetType_().GetTokenType() {
	case mysqlantlr.MySQLParserPRIMARY_SYMBOL,
		mysqlantlr.MySQLParserKEY_SYMBOL,
		mysqlantlr.MySQLParserINDEX_SYMBOL,
		mysqlantlr.MySQLParserUNIQUE_SYMBOL:

		if tableConstraint.IndexNameAndType() != nil {
			if tableConstraint.IndexNameAndType().IndexType() != nil {
				indexType := tableConstraint.IndexNameAndType().IndexType().GetText()
				return strings.ToUpper(indexType)
			}
		}

		for _, option := range tableConstraint.AllIndexOption() {
			if option == nil || option.IndexTypeClause() == nil {
				continue
			}

			indexType := option.IndexTypeClause().IndexType().GetText()
			return strings.ToUpper(indexType)
		}
	case mysqlantlr.MySQLParserFULLTEXT_SYMBOL:
		return "FULLTEXT"
	case mysqlantlr.MySQLParserFOREIGN_SYMBOL:
		// Foreign key - no specific index type
	default:
		// Other constraint types
	}
	// for mysql, we use BTREE as default index type.
	return "BTREE"
}

func (t *siTableState) mysqlCreateColumn(ctx *siFinderContext, columnName string, fieldDef mysqlantlr.IFieldDefinitionContext, position *siMysqlColumnPosition) error {
	if _, exists := t.columnSet[strings.ToLower(columnName)]; exists {
		return newSchemaViolationError(412, fmt.Sprintf("Column `%s` already exists in table `%s`", columnName, t.name))
	}

	// todo: handle position.
	pos := len(t.columnSet) + 1
	if position != nil && ctx.CheckIntegrity {
		var err error
		pos, err = t.mysqlReorderColumn(position)
		if err != nil {
			return err
		}
	}
	columnType := ""
	characterSet := ""
	collation := ""
	if fieldDef.DataType() == nil {
		// todo: add more error detail.
		return nil
	}
	columnType = mysqlparser.NormalizeMySQLDataType(fieldDef.DataType(), true /* compact */)
	characterSet = mysqlparser.GetCharSetName(fieldDef.DataType())
	collation = mysqlparser.GetCollationName(fieldDef)

	col := &siColumnState{
		name:         columnName,
		position:     &pos,
		defaultValue: nil,
		nullable:     newTruePointer(),
		columnType:   newStringPointer(columnType),
		characterSet: newStringPointer(characterSet),
		collation:    newStringPointer(collation),
		comment:      newEmptyStringPointer(),
	}
	setNullDefault := false

	for _, attribute := range fieldDef.AllColumnAttribute() {
		if attribute == nil {
			continue
		}
		if attribute.CheckConstraint() != nil {
			// we do not deal with CHECK constraint.
			continue
		}
		// not null.
		if attribute.NullLiteral() != nil && attribute.NOT_SYMBOL() != nil {
			col.nullable = newFalsePointer()
		}
		if attribute.GetValue() != nil {
			switch attribute.GetValue().GetTokenType() {
			// default value.
			case mysqlantlr.MySQLParserDEFAULT_SYMBOL:
				if err := mysqlCheckDefault(columnName, fieldDef); err != nil {
					return err
				}
				if attribute.SignedLiteral() == nil {
					continue
				}
				// handle default null.
				if attribute.SignedLiteral().Literal() != nil && attribute.SignedLiteral().Literal().NullLiteral() != nil {
					setNullDefault = true
					continue
				}
				// handle default 'null' etc.
				defaultValue := mysqlparser.NormalizeMySQLSignedLiteral(attribute.SignedLiteral())
				col.defaultValue = &defaultValue
			// comment.
			case mysqlantlr.MySQLParserCOMMENT_SYMBOL:
				if attribute.TextLiteral() == nil {
					continue
				}
				comment := mysqlparser.NormalizeMySQLTextLiteral(attribute.TextLiteral())
				col.comment = &comment
			// on update now().
			case mysqlantlr.MySQLParserON_SYMBOL:
				if attribute.UPDATE_SYMBOL() == nil || attribute.NOW_SYMBOL() == nil {
					continue
				}
				if !mysqlparser.IsTimeType(fieldDef.DataType()) {
					return errors.Errorf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", col.name)
				}
			// primary key.
			case mysqlantlr.MySQLParserKEY_SYMBOL:
				// the key attribute for in a column meaning primary key.
				col.nullable = newFalsePointer()
				// we need to check the key type which generated by tidb parser.
				if err := t.mysqlCreatePrimaryKey([]string{strings.ToLower(col.name)}, "BTREE"); err != nil {
					return err
				}
			// unique key.
			case mysqlantlr.MySQLParserUNIQUE_SYMBOL:
				// unique index.
				if err := t.mysqlCreateIndex("", []string{strings.ToLower(col.name)}, true /* unique */, "BTREE", mysqlantlr.NewEmptyTableConstraintDefContext(), mysqlantlr.NewEmptyCreateIndexContext()); err != nil {
					return err
				}
			// auto_increment.
			case mysqlantlr.MySQLParserAUTO_INCREMENT_SYMBOL:
				// we do not deal with AUTO_INCREMENT.
			// column_format.
			case mysqlantlr.MySQLParserCOLUMN_FORMAT_SYMBOL:
				// we do not deal with COLUMN_FORMAT.
			// storage.
			case mysqlantlr.MySQLParserSTORAGE_SYMBOL:
				// we do not deal with STORAGE.
			default:
				// Other column attributes
			}
		}
	}

	if col.nullable != nil && !*col.nullable && setNullDefault {
		return errors.Errorf("Invalid default value for column `%s`", col.name)
	}

	t.columnSet[strings.ToLower(col.name)] = col
	return nil
}

// reorderColumn reorders the columns for new column and returns the new column position.
func (t *siTableState) mysqlReorderColumn(position *siMysqlColumnPosition) (int, error) {
	switch position.tp {
	case siColumnPositionNone:
		return len(t.columnSet) + 1, nil
	case siColumnPositionFirst:
		for _, column := range t.columnSet {
			*column.position++
		}
		return 1, nil
	case siColumnPositionAfter:
		columnName := strings.ToLower(position.relativeColumn)
		column, exist := t.columnSet[columnName]
		if !exist {
			return 0, newSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, t.name))
		}
		for _, col := range t.columnSet {
			if *col.position > *column.position {
				*col.position++
			}
		}
		return *column.position + 1, nil
	default:
		return 0, errors.Errorf("Unsupported column position type: %d", position.tp)
	}
}

func (t *siTableState) mysqlCreateIndex(name string, keyList []string, unique bool, tp string, tableConstraint mysqlantlr.ITableConstraintDefContext, createIndexDef mysqlantlr.ICreateIndexContext) error {
	if len(keyList) == 0 {
		return errors.Errorf("Index `%s` in table `%s` has empty key", name, t.name)
	}
	// construct a index name if name is empty.
	if name != "" {
		if _, exists := t.indexSet[strings.ToLower(name)]; exists {
			return newSchemaViolationError(805, fmt.Sprintf("Index `%s` already exists in table `%s`", name, t.name))
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
		primary:        newFalsePointer(),
		visible:        newTruePointer(),
		comment:        newEmptyStringPointer(),
	}

	// need to check the visibility of index.
	// we need a for-loop to determined the visibility of index.

	// NORMAL KEY/INDEX.
	// PRIMARY KEY.
	// UNIQUE KEY.

	// for create table statement.
	for _, attribute := range tableConstraint.AllIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// for create index statement.
	for _, attribute := range createIndexDef.AllIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// FULLTEXT INDEX.
	// for create table statement.
	for _, attribute := range tableConstraint.AllFulltextIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// for create index statement.
	for _, attribute := range createIndexDef.AllFulltextIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// SPATIAL INDEX.
	// for create table statement.
	for _, attribute := range tableConstraint.AllSpatialIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// for create index statement.
	for _, attribute := range createIndexDef.AllSpatialIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	t.indexSet[strings.ToLower(name)] = index
	return nil
}

func (t *siTableState) mysqlCreatePrimaryKey(keys []string, tp string) error {
	if _, exists := t.indexSet[strings.ToLower(siPrimaryKeyName)]; exists {
		return errors.Errorf("Primary key exists in table `%s`", t.name)
	}

	pk := &siIndexState{
		name:           siPrimaryKeyName,
		expressionList: keys,
		indexType:      &tp,
		unique:         newTruePointer(),
		primary:        newTruePointer(),
		visible:        newTruePointer(),
		comment:        newEmptyStringPointer(),
	}
	t.indexSet[strings.ToLower(pk.name)] = pk
	return nil
}

func mysqlCheckDefault(columnName string, fieldDefinition mysqlantlr.IFieldDefinitionContext) error {
	if fieldDefinition.DataType() == nil || fieldDefinition.DataType().GetType_() == nil {
		return nil
	}

	switch fieldDefinition.DataType().GetType_().GetTokenType() {
	case mysqlantlr.MySQLParserTEXT_SYMBOL,
		mysqlantlr.MySQLParserTINYTEXT_SYMBOL,
		mysqlantlr.MySQLParserMEDIUMTEXT_SYMBOL,
		mysqlantlr.MySQLParserLONGTEXT_SYMBOL,
		mysqlantlr.MySQLParserBLOB_SYMBOL,
		mysqlantlr.MySQLParserTINYBLOB_SYMBOL,
		mysqlantlr.MySQLParserMEDIUMBLOB_SYMBOL,
		mysqlantlr.MySQLParserLONGBLOB_SYMBOL,
		mysqlantlr.MySQLParserLONG_SYMBOL,
		mysqlantlr.MySQLParserSERIAL_SYMBOL,
		mysqlantlr.MySQLParserJSON_SYMBOL,
		mysqlantlr.MySQLParserGEOMETRY_SYMBOL,
		mysqlantlr.MySQLParserGEOMETRYCOLLECTION_SYMBOL,
		mysqlantlr.MySQLParserPOINT_SYMBOL,
		mysqlantlr.MySQLParserMULTIPOINT_SYMBOL,
		mysqlantlr.MySQLParserLINESTRING_SYMBOL,
		mysqlantlr.MySQLParserMULTILINESTRING_SYMBOL,
		mysqlantlr.MySQLParserPOLYGON_SYMBOL,
		mysqlantlr.MySQLParserMULTIPOLYGON_SYMBOL:
		return newSchemaViolationError(423, fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName))
	default:
		// Other data types are allowed to have default values
	}

	return checkDefaultConvert(columnName, fieldDefinition)
}

func checkDefaultConvert(columnName string, fieldDefinition mysqlantlr.IFieldDefinitionContext) error {
	if fieldDefinition == nil {
		return nil
	}
	list, err := tidb.ParseTiDB(fmt.Sprintf("CREATE TABLE t(%s %s)", columnName, fieldDefinition.GetParser().GetTokenStream().GetTextFromRuleContext(fieldDefinition)), "", "")
	if err != nil {
		// For now, we do not handle this case.
		// nolint:nilerr
		return nil
	}
	if len(list) != 1 {
		return nil
	}
	createTable, ok := list[0].(*tidbast.CreateTableStmt)
	if !ok {
		return nil
	}
	if len(createTable.Cols) != 1 {
		return nil
	}
	col := createTable.Cols[0]
	for _, option := range col.Options {
		if option.Tp == tidbast.ColumnOptionDefaultValue {
			return checkDefault(columnName, col.Tp, option.Expr)
		}
	}

	return nil
}

func checkDefault(columnName string, columnType *tidbtypes.FieldType, value tidbast.ExprNode) error {
	if value.GetType().GetType() != tidbmysql.TypeNull {
		switch columnType.GetType() {
		case tidbmysql.TypeBlob, tidbmysql.TypeTinyBlob, tidbmysql.TypeMediumBlob, tidbmysql.TypeLongBlob, tidbmysql.TypeJSON, tidbmysql.TypeGeometry:
			return newSchemaViolationError(423, fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName))
		default:
			// Other column types allow default values
		}
	}

	if valueExpr, yes := value.(tidbast.ValueExpr); yes {
		datum := tidbtypes.NewDatum(valueExpr.GetValue())
		if _, err := datum.ConvertTo(tidbtypes.Context{}, columnType); err != nil {
			return errors.Errorf("%s", err.Error())
		}
	}
	return nil
}
