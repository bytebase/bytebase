package pg

// Schema Integrity Advisor - validates PostgreSQL DDL statements

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

const (
	siPublicSchemaName = "public"
)

var (
	_ advisor.Advisor = (*SchemaIntegrityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleSchemaIntegrity, &SchemaIntegrityAdvisor{})
}

type SchemaIntegrityAdvisor struct {
}

func (*SchemaIntegrityAdvisor) Check(_ context.Context, ctx advisor.Context) ([]*storepb.Advice, error) {
	parseResult, ok := ctx.AST.(*pgparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", ctx.AST)
	}

	dbState := siNewDatabaseStateFromCatalog(ctx.DBSchema)

	if err := dbState.pgWalkThrough(parseResult); err != nil {
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

	// No violations found, return empty advice list
	return make([]*storepb.Advice, 0), nil
}

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

func siNewDatabaseStateFromCatalog(dbSchema *storepb.DatabaseSchemaMetadata) *siDatabaseState {
	db := &siDatabaseState{
		ctx: &siFinderContext{
			CheckIntegrity:      true,
			EngineType:          storepb.Engine_POSTGRES,
			IgnoreCaseSensitive: false,
		},
		name:         "",
		characterSet: "",
		collation:    "",
		dbType:       storepb.Engine_POSTGRES,
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

	for _, schema := range dbSchema.Schemas {
		schemaState := &siSchemaState{
			ctx:           db.ctx.Copy(),
			name:          schema.Name,
			tableSet:      make(siTableStateMap),
			viewSet:       make(siViewStateMap),
			identifierMap: make(siIdentifierMap),
		}

		for _, table := range schema.Tables {
			tableState := &siTableState{
				name:      table.Name,
				engine:    siNewStringPointer(table.Engine),
				collation: siNewStringPointer(table.Collation),
				comment:   siNewStringPointer(table.Comment),
				columnSet: make(siColumnStateMap),
				indexSet:  make(siIndexStateMap),
			}

			for i, column := range table.Columns {
				colState := &siColumnState{
					name:         column.Name,
					position:     siNewIntPointer(i + 1),
					nullable:     siNewBoolPointer(column.Nullable),
					columnType:   siNewStringPointer(column.Type),
					characterSet: siNewStringPointer(column.CharacterSet),
					collation:    siNewStringPointer(column.Collation),
					comment:      siNewStringPointer(column.Comment),
				}
				if column.Default != "" {
					colState.defaultValue = siCopyStringPointer(&column.Default)
				}
				tableState.columnSet[column.Name] = colState
			}

			for _, index := range table.Indexes {
				indexState := &siIndexState{
					name:           index.Name,
					expressionList: siCopyStringSlice(index.Expressions),
					indexType:      siNewStringPointer(index.Type),
					unique:         siNewBoolPointer(index.Unique),
					primary:        siNewBoolPointer(index.Primary),
					visible:        siNewBoolPointer(index.Visible),
					comment:        siNewStringPointer(index.Comment),
					isConstraint:   index.Primary || index.Unique,
				}
				tableState.indexSet[index.Name] = indexState
			}

			schemaState.tableSet[table.Name] = tableState
			schemaState.identifierMap[table.Name] = true
			for indexName := range tableState.indexSet {
				schemaState.identifierMap[indexName] = true
			}
		}

		for _, view := range schema.Views {
			schemaState.viewSet[view.Name] = &siViewState{
				name:       view.Name,
				definition: siNewStringPointer(view.Definition),
				comment:    siNewStringPointer(view.Comment),
			}
			schemaState.identifierMap[view.Name] = true
		}

		db.schemaSet[schema.Name] = schemaState
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
	name      string
	engine    *string
	collation *string
	comment   *string
	columnSet siColumnStateMap
	indexSet  siIndexStateMap

	// dependencyView is used to record the dependency view for the table.
	// Used to check if the table is used by any view.
	dependencyView map[string]bool // nolint:unused
}

type siColumnState struct {
	name         string
	position     *int
	defaultValue *string
	nullable     *bool
	columnType   *string
	characterSet *string
	collation    *string
	comment      *string

	// dependencyView is used to record the dependency view for the column.
	// Used to check if the column is used by any view.
	dependencyView map[string]bool // nolint:unused
}

type siIndexState struct {
	name string
	// This could refer to a column or an expression.
	expressionList []string
	indexType      *string
	unique         *bool
	primary        *bool
	visible        *bool
	comment        *string

	// PostgreSQL specific fields.
	// PostgreSQL treats INDEX and CONSTRAINT differently.
	isConstraint bool
}

type siViewState struct {
	name       string
	definition *string
	comment    *string
}

type siSchemaStateMap map[string]*siSchemaState

type siTableStateMap map[string]*siTableState

type siColumnStateMap map[string]*siColumnState

type siIndexStateMap map[string]*siIndexState

type siViewStateMap map[string]*siViewState

type siIdentifierMap map[string]bool

func siCopyStringPointer(p *string) *string {
	if p != nil {
		v := *p
		return &v
	}
	return nil
}

// nolint:unused
func siCopyBoolPointer(p *bool) *bool {
	if p != nil {
		v := *p
		return &v
	}
	return nil
}

// nolint:unused
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

// pgWalkThrough walks through the ANTLR parse tree and builds catalog state.
func (d *siDatabaseState) pgWalkThrough(ast any) error {
	// ANTLR-based walkthrough
	parseResult, ok := ast.(*pgparser.ParseResult)
	if !ok {
		return errors.Errorf("PostgreSQL walk-through expects *pgparser.ParseResult, got %T", ast)
	}

	root, ok := parseResult.Tree.(parser.IRootContext)
	if !ok {
		return errors.Errorf("invalid ANTLR tree type %T", parseResult.Tree)
	}

	// Build listener with database state
	listener := &siPgCatalogListener{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		databaseState:                d,
	}

	// Walk through the parse tree
	antlr.ParseTreeWalkerDefault.Walk(listener, root)

	// Return any error encountered during walk
	if listener.err != nil {
		return listener.err
	}

	return nil
}

// siPgCatalogListener builds catalog state by listening to ANTLR parse tree events.
type siPgCatalogListener struct {
	*parser.BasePostgreSQLParserListener

	databaseState *siDatabaseState
	err           error
	currentLine   int
}

// Helper method to set error with line number
func (l *siPgCatalogListener) setError(err error) {
	if l.err != nil {
		return // Keep first error
	}
	l.err = err
}

// Helper method to check if database is deleted
func (l *siPgCatalogListener) checkDatabaseNotDeleted() bool {
	if l.databaseState.deleted {
		l.setError(errors.Errorf(`Database %q is deleted`, l.databaseState.name))
		return false
	}
	return true
}

// ========================================
// CREATE TABLE handling
// ========================================

// EnterCreatestmt handles CREATE TABLE statements.
func (l *siPgCatalogListener) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !siIsTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	if !l.checkDatabaseNotDeleted() {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Extract table name and schema
	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := siExtractTableName(qualifiedNames[0])
	schemaName := siExtractSchemaName(qualifiedNames[0])

	if tableName == "" {
		return
	}

	// Check database name if specified
	databaseName := siExtractDatabaseName(qualifiedNames[0])
	if databaseName != "" && l.databaseState.name != databaseName {
		l.setError(errors.Errorf("Database %q is not the current database %q", databaseName, l.databaseState.name))
		return
	}

	// Get or create schema
	schema, err := l.databaseState.getSchema(schemaName)
	if err != nil {
		l.setError(err)
		return
	}

	// Check if table already exists
	if _, exists := schema.tableSet[tableName]; exists {
		// Check IF NOT EXISTS clause
		ifNotExists := ctx.IF_P() != nil && ctx.NOT() != nil && ctx.EXISTS() != nil
		if ifNotExists {
			return
		}
		l.setError(errors.Errorf(`The table %q already exists in the schema %q`, tableName, schema.name))
		return
	}

	// Create table state
	table := &siTableState{
		name:      tableName,
		columnSet: make(siColumnStateMap),
		indexSet:  make(siIndexStateMap),
	}
	schema.tableSet[table.name] = table

	// Process column definitions
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Handle column definitions
			if elem.ColumnDef() != nil {
				if err := siCreateColumn(schema, table, elem.ColumnDef()); err != nil {
					l.setError(err)
					return
				}
			}
			// Handle table-level constraints
			if elem.Tableconstraint() != nil {
				if err := siCreateTableConstraint(schema, table, elem.Tableconstraint()); err != nil {
					l.setError(err)
					return
				}
			}
		}
	}
}

// siCreateColumn creates a column in the table.
func siCreateColumn(schema *siSchemaState, table *siTableState, columnDef parser.IColumnDefContext) error {
	if columnDef == nil {
		return nil
	}

	// Extract column name
	var columnName string
	if columnDef.Colid() != nil {
		columnName = pgparser.NormalizePostgreSQLColid(columnDef.Colid())
	}
	if columnName == "" {
		return nil
	}

	// Check if column already exists
	if _, exists := table.columnSet[columnName]; exists {
		return errors.Errorf("The column %q already exists in table %q", columnName, table.name)
	}

	// Get column type
	var columnType string
	if columnDef.Typename() != nil {
		columnType = siExtractTypeName(columnDef.Typename())
	}

	// Create column state
	pos := len(table.columnSet) + 1
	columnState := &siColumnState{
		name:         columnName,
		position:     &pos,
		defaultValue: nil,
		nullable:     siNewTruePointer(),
		columnType:   &columnType,
		collation:    nil,
	}
	table.columnSet[columnState.name] = columnState

	// Process column constraints
	if columnDef.Colquallist() != nil {
		allQuals := columnDef.Colquallist().AllColconstraint()
		for _, qual := range allQuals {
			if qual.Colconstraintelem() == nil {
				continue
			}
			elem := qual.Colconstraintelem()

			// Handle NOT NULL
			if elem.NOT() != nil && elem.NULL_P() != nil {
				columnState.nullable = siNewFalsePointer()
			}

			// Handle DEFAULT
			if elem.DEFAULT() != nil {
				// Extract default expression from B_expr
				if elem.B_expr() != nil {
					defaultValue := elem.B_expr().GetText()
					columnState.defaultValue = &defaultValue
				}
			}

			// Handle PRIMARY KEY
			if elem.PRIMARY() != nil && elem.KEY() != nil {
				constraintName := ""
				if qual.Name() != nil {
					constraintName = pgparser.NormalizePostgreSQLName(qual.Name())
				}
				if constraintName == "" {
					constraintName = schema.pgGeneratePrimaryKeyName(table.name)
				}

				// Set column as NOT NULL
				columnState.nullable = siNewFalsePointer()

				// Create primary key index
				index := &siIndexState{
					name:           constraintName,
					expressionList: []string{columnName},
					indexType:      siNewStringPointer("btree"),
					unique:         siNewTruePointer(),
					primary:        siNewTruePointer(),
					isConstraint:   true,
				}
				table.indexSet[index.name] = index
				schema.identifierMap[index.name] = true
			}

			// Handle UNIQUE
			if elem.UNIQUE() != nil && (elem.PRIMARY() == nil || elem.KEY() == nil) {
				constraintName := ""
				if qual.Name() != nil {
					constraintName = pgparser.NormalizePostgreSQLName(qual.Name())
				}

				// Generate index name if not specified
				if constraintName == "" {
					constraintName = siGenerateIndexName(table.name, []string{columnName}, true)
				}

				index := &siIndexState{
					name:           constraintName,
					expressionList: []string{columnName},
					indexType:      siNewStringPointer("btree"),
					unique:         siNewTruePointer(),
					primary:        siNewFalsePointer(),
					isConstraint:   true,
				}
				table.indexSet[index.name] = index
				schema.identifierMap[index.name] = true
			}
		}
	}

	return nil
}

// siCreateTableConstraint creates a table-level constraint.
func siCreateTableConstraint(schema *siSchemaState, table *siTableState, constraint parser.ITableconstraintContext) error {
	if constraint == nil || constraint.Constraintelem() == nil {
		return nil
	}

	elem := constraint.Constraintelem()

	// Extract constraint name
	constraintName := ""
	if constraint.Name() != nil {
		constraintName = pgparser.NormalizePostgreSQLName(constraint.Name())
	}

	// Handle PRIMARY KEY constraint
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		var columnList []string
		if elem.Columnlist() != nil {
			allColumns := elem.Columnlist().AllColumnElem()
			for _, col := range allColumns {
				if col.Colid() != nil {
					colName := pgparser.NormalizePostgreSQLColid(col.Colid())
					columnList = append(columnList, colName)
				}
			}
		}

		// Set all PK columns as NOT NULL
		for _, colName := range columnList {
			if column, exists := table.columnSet[colName]; exists {
				column.nullable = siNewFalsePointer()
			} else {
				return siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", colName, table.name))
			}
		}

		// Generate PK name if not provided
		pkName := constraintName
		if pkName == "" {
			pkName = schema.pgGeneratePrimaryKeyName(table.name)
		}

		// Check if identifier already exists
		if _, exists := schema.identifierMap[pkName]; exists {
			return siNewSchemaViolationError(1, fmt.Sprintf("Relation %q already exists in schema %q", pkName, schema.name))
		}

		// Create primary key index
		index := &siIndexState{
			name:           pkName,
			expressionList: columnList,
			indexType:      siNewStringPointer("btree"),
			unique:         siNewTruePointer(),
			primary:        siNewTruePointer(),
			isConstraint:   true,
		}
		table.indexSet[index.name] = index
		schema.identifierMap[index.name] = true
	}

	// Handle UNIQUE constraint
	if elem.UNIQUE() != nil && (elem.PRIMARY() == nil || elem.KEY() == nil) {
		var columnList []string
		if elem.Columnlist() != nil {
			allColumns := elem.Columnlist().AllColumnElem()
			for _, col := range allColumns {
				if col.Colid() != nil {
					colName := pgparser.NormalizePostgreSQLColid(col.Colid())
					columnList = append(columnList, colName)
				}
			}
		}

		// Generate index name if not specified
		indexName := constraintName
		if indexName == "" {
			indexName = siGenerateIndexName(table.name, columnList, true)
		}

		// Check if identifier already exists
		if _, exists := schema.identifierMap[indexName]; exists {
			return siNewSchemaViolationError(1, fmt.Sprintf("Relation %q already exists in schema %q", indexName, schema.name))
		}

		// Validate columns exist
		for _, colName := range columnList {
			if _, exists := table.columnSet[colName]; !exists {
				return siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", colName, table.name))
			}
		}

		// Create unique index
		index := &siIndexState{
			name:           indexName,
			expressionList: columnList,
			indexType:      siNewStringPointer("btree"),
			unique:         siNewTruePointer(),
			primary:        siNewFalsePointer(),
			isConstraint:   true,
		}
		table.indexSet[index.name] = index
		schema.identifierMap[index.name] = true
	}

	// Note: We skip CHECK, FOREIGN KEY, EXCLUSION constraints for now
	// as the legacy implementation also skips them

	return nil
}

// ========================================
// CREATE INDEX handling
// ========================================

// EnterIndexstmt handles CREATE INDEX statements.
func (l *siPgCatalogListener) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !siIsTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	if !l.checkDatabaseNotDeleted() {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Extract relation (table) name
	relationExpr := ctx.Relation_expr()
	if relationExpr == nil || relationExpr.Qualified_name() == nil {
		return
	}

	tableName := siExtractTableName(relationExpr.Qualified_name())
	schemaName := siExtractSchemaName(relationExpr.Qualified_name())
	schema, err := l.databaseState.getSchema(schemaName)
	if err != nil {
		l.setError(err)
		return
	}

	table, exists := schema.tableSet[tableName]
	if !exists {
		l.setError(siNewSchemaViolationError(604, fmt.Sprintf("Table `%s` does not exist", tableName)))
		return
	}

	// Extract index name (can be empty for auto-generated names)
	indexName := ""
	if ctx.Name() != nil {
		indexName = pgparser.NormalizePostgreSQLName(ctx.Name())
	}

	// Check IF NOT EXISTS
	ifNotExists := ctx.IF_P() != nil && ctx.NOT() != nil && ctx.EXISTS() != nil

	// Extract column list
	var columnList []string
	if ctx.Index_params() != nil {
		allParams := ctx.Index_params().AllIndex_elem()
		for _, param := range allParams {
			if param.Colid() != nil {
				colName := pgparser.NormalizePostgreSQLColid(param.Colid())
				columnList = append(columnList, colName)
			} else if param.Func_expr_windowless() != nil {
				// Expression index - use placeholder
				columnList = append(columnList, "expr")
			}
		}
	}

	if len(columnList) == 0 {
		l.setError(errors.Errorf("Index %q in table %q has empty key", indexName, tableName))
		return
	}

	// Generate index name if not provided
	isUnique := ctx.Opt_unique() != nil && ctx.Opt_unique().UNIQUE() != nil
	wasAutoGenerated := indexName == ""
	if indexName == "" {
		indexName = siGenerateIndexName(tableName, columnList, isUnique)
	}

	// Check if index name already exists
	if _, exists := schema.identifierMap[indexName]; exists {
		if ifNotExists {
			return
		}
		// If name was auto-generated, try with numeric suffix
		if wasAutoGenerated {
			indexName = siGenerateUniqueIndexName(schema, tableName, columnList, isUnique)
		} else {
			l.setError(siNewSchemaViolationError(1, fmt.Sprintf("Relation %q already exists in schema %q", indexName, schema.name)))
			return
		}
	}

	// Check that all columns exist (skip expressions)
	for _, colName := range columnList {
		if colName != "expr" {
			if _, exists := table.columnSet[colName]; !exists {
				l.setError(siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", colName, tableName)))
				return
			}
		}
	}

	// Determine index type
	indexType := "btree" // default
	if ctx.Access_method_clause() != nil && ctx.Access_method_clause().Name() != nil {
		method := pgparser.NormalizePostgreSQLName(ctx.Access_method_clause().Name())
		indexType = method
	}

	// Create index state
	index := &siIndexState{
		name:           indexName,
		expressionList: columnList,
		indexType:      siNewStringPointer(indexType),
		unique:         siNewBoolPointer(isUnique),
		primary:        siNewFalsePointer(),
		isConstraint:   false,
	}

	table.indexSet[index.name] = index
	schema.identifierMap[index.name] = true
}

// ========================================
// ALTER TABLE handling
// ========================================

// EnterAltertablestmt handles ALTER TABLE statements.
func (l *siPgCatalogListener) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !siIsTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	if !l.checkDatabaseNotDeleted() {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Extract table name
	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := siExtractTableName(ctx.Relation_expr().Qualified_name())
	schemaName := siExtractSchemaName(ctx.Relation_expr().Qualified_name())
	databaseName := siExtractDatabaseName(ctx.Relation_expr().Qualified_name())

	// Check database access
	if databaseName != "" && l.databaseState.name != databaseName {
		l.setError(errors.Errorf("Database %q is not the current database %q", databaseName, l.databaseState.name))
		return
	}

	// Get schema and table
	schema, err := l.databaseState.getSchema(schemaName)
	if err != nil {
		l.setError(err)
		return
	}

	table, err := schema.pgGetTable(tableName)
	if err != nil {
		l.setError(err)
		return
	}

	// Process alter table commands
	if ctx.Alter_table_cmds() == nil {
		return
	}

	allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
	for _, cmd := range allCmds {
		l.processAlterTableCmd(schema, table, cmd)
		if l.err != nil {
			return
		}
	}
}

// processAlterTableCmd handles individual ALTER TABLE commands.
func (l *siPgCatalogListener) processAlterTableCmd(schema *siSchemaState, table *siTableState, cmd parser.IAlter_table_cmdContext) {
	// RENAME operations are handled by EnterRenamestmt, not here

	// Handle ADD COLUMN
	if cmd.ADD_P() != nil && cmd.COLUMN() != nil {
		if cmd.ColumnDef() != nil {
			ifNotExists := cmd.IF_P() != nil && cmd.NOT() != nil && cmd.EXISTS() != nil
			l.alterTableAddColumn(schema, table, cmd.ColumnDef(), ifNotExists)
		}
		return
	}

	// Handle ADD CONSTRAINT
	if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
		l.alterTableAddConstraint(schema, table, cmd.Tableconstraint())
		return
	}

	// Handle DROP CONSTRAINT
	if cmd.DROP() != nil && cmd.CONSTRAINT() != nil {
		ifExists := cmd.IF_P() != nil && cmd.EXISTS() != nil
		if cmd.Name() != nil {
			constraintName := pgparser.NormalizePostgreSQLName(cmd.Name())
			l.alterTableDropConstraint(schema, table, constraintName, ifExists)
		}
		return
	}

	// Handle DROP COLUMN
	if cmd.DROP() != nil && cmd.Opt_column() != nil {
		ifExists := cmd.IF_P() != nil && cmd.EXISTS() != nil
		// AllColid() returns a list - get the first column name
		allColids := cmd.AllColid()
		if len(allColids) > 0 {
			columnName := pgparser.NormalizePostgreSQLColid(allColids[0])
			l.alterTableDropColumn(schema, table, columnName, ifExists)
		}
		return
	}

	// Handle ALTER COLUMN commands
	if cmd.ALTER() != nil && cmd.Opt_column() != nil {
		allColids := cmd.AllColid()
		if len(allColids) > 0 {
			columnName := pgparser.NormalizePostgreSQLColid(allColids[0])

			// Check for SET DATA TYPE
			if cmd.TYPE_P() != nil && cmd.Typename() != nil {
				typeString := siExtractTypeName(cmd.Typename())
				l.alterTableAlterColumnType(schema, table, columnName, typeString)
				return
			}

			// Check for SET/DROP DEFAULT
			if cmd.Alter_column_default() != nil {
				altDefault := cmd.Alter_column_default()
				if altDefault.SET() != nil && altDefault.A_expr() != nil {
					// SET DEFAULT
					defaultValue := altDefault.A_expr().GetText()
					l.alterTableSetDefault(table, columnName, defaultValue)
				} else if altDefault.DROP() != nil {
					// DROP DEFAULT
					l.alterTableDropDefault(table, columnName)
				}
				return
			}

			// Check for SET NOT NULL
			if cmd.SET() != nil && cmd.NOT() != nil && cmd.NULL_P() != nil {
				l.alterTableSetNotNull(table, columnName)
				return
			}
		}
		return
	}
}

// alterTableDropColumn handles DROP COLUMN command.
func (l *siPgCatalogListener) alterTableDropColumn(schema *siSchemaState, table *siTableState, columnName string, ifExists bool) {
	column, exists := table.columnSet[columnName]
	if !exists {
		if ifExists {
			return
		}
		l.setError(siNewSchemaViolationError(405, fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.name)))
		return
	}

	// Check if column is referenced by any views
	viewList, err := l.databaseState.existedViewList(column.dependencyView)
	if err != nil {
		l.setError(err)
		return
	}
	if len(viewList) > 0 {
		l.setError(errors.Errorf("Cannot drop column %q in table %q.%q, it's referenced by view: %s", column.name, schema.name, table.name, strings.Join(viewList, ", ")))
		return
	}

	// Drop the constraints and indexes involving the column.
	var dropIndexList []string

	for _, index := range table.indexSet {
		for _, key := range index.expressionList {
			// TODO(zp): deal with expression key.
			if key == columnName {
				dropIndexList = append(dropIndexList, index.name)
				break // Once we find the column in this index, mark for deletion and move to next index
			}
		}
	}
	for _, indexName := range dropIndexList {
		delete(table.indexSet, indexName)
	}

	// TODO(zp): deal with other constraints.

	// TODO(zp): deal with CASCADE.

	// Delete the column
	delete(table.columnSet, columnName)
}

// alterTableAlterColumnType handles ALTER COLUMN TYPE command.
func (l *siPgCatalogListener) alterTableAlterColumnType(schema *siSchemaState, table *siTableState, columnName string, typeString string) {
	column, err := table.getColumn(columnName)
	if err != nil {
		l.setError(err)
		return
	}

	// Check if column is referenced by any views
	viewList, viewErr := l.databaseState.existedViewList(column.dependencyView)
	if viewErr != nil {
		l.setError(viewErr)
		return
	}
	if len(viewList) > 0 {
		l.setError(errors.Errorf("Cannot alter type of column %q in table %q.%q, it's referenced by view: %s", column.name, schema.name, table.name, strings.Join(viewList, ", ")))
		return
	}

	// Update column type
	column.columnType = &typeString
}

// alterTableAddColumn handles ADD COLUMN command.
func (l *siPgCatalogListener) alterTableAddColumn(schema *siSchemaState, table *siTableState, columndef parser.IColumnDefContext, ifNotExists bool) {
	if columndef == nil {
		return
	}

	columnName := pgparser.NormalizePostgreSQLColid(columndef.Colid())

	// Check if column already exists
	if _, exists := table.columnSet[columnName]; exists {
		if ifNotExists {
			return
		}
		l.setError(errors.Errorf("The column %q already exists in table %q", columnName, table.name))
		return
	}

	// Get position for new column
	pos := len(table.columnSet) + 1

	// Extract column type
	var typeString string
	if columndef.Typename() != nil {
		typeString = siExtractTypeName(columndef.Typename())
	}

	// Create column state
	columnState := &siColumnState{
		name:         columnName,
		position:     &pos,
		nullable:     siNewTruePointer(),
		columnType:   &typeString,
		defaultValue: nil,
	}
	table.columnSet[columnName] = columnState

	// Process column constraints if any (inline processing like in siCreateColumn)
	if columndef.Colquallist() != nil {
		allQuals := columndef.Colquallist().AllColconstraint()
		for _, qual := range allQuals {
			if qual.Colconstraintelem() == nil {
				continue
			}
			elem := qual.Colconstraintelem()

			// Handle NOT NULL
			if elem.NOT() != nil && elem.NULL_P() != nil {
				columnState.nullable = siNewFalsePointer()
			}

			// Handle DEFAULT
			if elem.DEFAULT() != nil {
				if elem.B_expr() != nil {
					defaultValue := elem.B_expr().GetText()
					columnState.defaultValue = &defaultValue
				}
			}

			// Handle UNIQUE - creates an index
			if elem.UNIQUE() != nil && (elem.PRIMARY() == nil || elem.KEY() == nil) {
				var constraintName string
				if qual.Name() != nil {
					constraintName = pgparser.NormalizePostgreSQLName(qual.Name())
				}
				if constraintName == "" {
					constraintName = siGenerateIndexName(table.name, []string{columnName}, true)
				}
				// Check for collision
				if _, exists := schema.identifierMap[constraintName]; exists {
					constraintName = siGenerateUniqueIndexName(schema, table.name, []string{columnName}, true)
				}
				// Create index
				index := &siIndexState{
					name:           constraintName,
					expressionList: []string{columnName},
					indexType:      siNewStringPointer("btree"),
					unique:         siNewTruePointer(),
					primary:        siNewFalsePointer(),
					isConstraint:   true,
				}
				table.indexSet[index.name] = index
				schema.identifierMap[index.name] = true
			}

			// Handle PRIMARY KEY - creates an index
			if (elem.PRIMARY() != nil && elem.KEY() != nil) || (elem.UNIQUE() != nil && elem.PRIMARY() != nil) {
				var constraintName string
				if qual.Name() != nil {
					constraintName = pgparser.NormalizePostgreSQLName(qual.Name())
				}
				if constraintName == "" {
					constraintName = schema.pgGeneratePrimaryKeyName(table.name)
				}
				// Check for collision
				if _, exists := schema.identifierMap[constraintName]; exists {
					l.setError(siNewSchemaViolationError(1, fmt.Sprintf("Relation %q already exists in schema %q", constraintName, schema.name)))
					return
				}
				columnState.nullable = siNewFalsePointer()
				// Create primary key index
				index := &siIndexState{
					name:           constraintName,
					expressionList: []string{columnName},
					indexType:      siNewStringPointer("btree"),
					unique:         siNewTruePointer(),
					primary:        siNewTruePointer(),
					isConstraint:   true,
				}
				table.indexSet[index.name] = index
				schema.identifierMap[index.name] = true
			}
		}
	}
}

// alterTableAddConstraint handles ADD CONSTRAINT command.
func (l *siPgCatalogListener) alterTableAddConstraint(schema *siSchemaState, table *siTableState, constraint parser.ITableconstraintContext) {
	if constraint == nil {
		return
	}

	// Reuse the constraint creation logic from CREATE TABLE
	err := siCreateTableConstraint(schema, table, constraint)
	if err != nil {
		l.setError(err)
	}
}

// alterTableDropConstraint handles DROP CONSTRAINT command.
func (l *siPgCatalogListener) alterTableDropConstraint(schema *siSchemaState, table *siTableState, constraintName string, ifExists bool) {
	// Check if constraint exists as an index
	if index, exists := table.indexSet[constraintName]; exists {
		delete(schema.identifierMap, index.name)
		delete(table.indexSet, index.name)
		return
	}

	if !ifExists {
		l.setError(errors.Errorf("Constraint %q for table %q does not exist", constraintName, table.name))
	}
}

// alterTableSetDefault handles ALTER COLUMN SET DEFAULT command.
func (l *siPgCatalogListener) alterTableSetDefault(table *siTableState, columnName string, defaultValue string) {
	column, err := table.getColumn(columnName)
	if err != nil {
		l.setError(err)
		return
	}

	column.defaultValue = &defaultValue
}

// alterTableDropDefault handles ALTER COLUMN DROP DEFAULT command.
func (l *siPgCatalogListener) alterTableDropDefault(table *siTableState, columnName string) {
	column, err := table.getColumn(columnName)
	if err != nil {
		l.setError(err)
		return
	}

	column.defaultValue = nil
}

// alterTableSetNotNull handles ALTER COLUMN SET NOT NULL command.
func (l *siPgCatalogListener) alterTableSetNotNull(table *siTableState, columnName string) {
	column, err := table.getColumn(columnName)
	if err != nil {
		l.setError(err)
		return
	}

	column.nullable = siNewFalsePointer()
}

// renameTable handles RENAME TO for tables.
func (*siPgCatalogListener) renameTable(schema *siSchemaState, table *siTableState, newName string) error {
	// Check if new name already exists
	if _, exists := schema.identifierMap[newName]; exists {
		return siNewSchemaViolationError(1, fmt.Sprintf("Relation %q already exists in schema %q", newName, schema.name))
	}

	// Remove old name from maps
	delete(schema.identifierMap, table.name)
	delete(schema.tableSet, table.name)

	// Update table name
	table.name = newName

	// Add new name to maps
	schema.identifierMap[table.name] = true
	schema.tableSet[table.name] = table

	return nil
}

// renameColumn handles RENAME COLUMN.
func (*siPgCatalogListener) renameColumn(table *siTableState, oldName string, newName string) error {
	column, err := table.getColumn(oldName)
	if err != nil {
		return err
	}

	if oldName == newName {
		return nil
	}

	// Check if new name already exists
	if _, exists := table.columnSet[newName]; exists {
		return errors.Errorf("The column %q already exists in table %q", newName, table.name)
	}

	// Rename column in all indexes that reference it
	for _, index := range table.indexSet {
		for i, key := range index.expressionList {
			if key == oldName {
				index.expressionList[i] = newName
			}
		}
	}

	// Update column name
	delete(table.columnSet, column.name)
	column.name = newName
	table.columnSet[column.name] = column

	return nil
}

// renameConstraint handles RENAME CONSTRAINT.
func (*siPgCatalogListener) renameConstraint(schema *siSchemaState, table *siTableState, oldName string, newName string) error {
	index, exists := table.indexSet[oldName]
	if !exists {
		// We haven't dealt with foreign and check constraints, so skip if not exists
		return nil
	}

	// Check if new name already exists
	if _, exists := schema.identifierMap[newName]; exists {
		return siNewSchemaViolationError(1, fmt.Sprintf("Relation %q already exists in schema %q", newName, schema.name))
	}

	// Remove old name from maps
	delete(schema.identifierMap, index.name)
	delete(table.indexSet, index.name)

	// Update index name
	index.name = newName

	// Add new name to maps
	schema.identifierMap[index.name] = true
	table.indexSet[index.name] = index

	return nil
}

// ========================================
// DROP statements handling
// ========================================

// EnterDropstmt handles DROP TABLE/VIEW/INDEX statements.
func (l *siPgCatalogListener) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !siIsTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	if !l.checkDatabaseNotDeleted() {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Check IF EXISTS
	ifExists := ctx.IF_P() != nil && ctx.EXISTS() != nil

	// Check object type and get list of names
	if ctx.Object_type_any_name() != nil {
		objType := ctx.Object_type_any_name()

		if objType.TABLE() != nil {
			// DROP TABLE
			if ctx.Any_name_list() != nil {
				for _, anyName := range ctx.Any_name_list().AllAny_name() {
					if err := l.dropTable(anyName, ifExists); err != nil {
						l.setError(err)
						return
					}
				}
			}
		} else if objType.VIEW() != nil {
			// DROP VIEW
			if ctx.Any_name_list() != nil {
				for _, anyName := range ctx.Any_name_list().AllAny_name() {
					if err := l.dropView(anyName, ifExists); err != nil {
						l.setError(err)
						return
					}
				}
			}
		} else if objType.INDEX() != nil {
			// DROP INDEX
			if ctx.Any_name_list() != nil {
				for _, anyName := range ctx.Any_name_list().AllAny_name() {
					if err := l.dropIndex(anyName, ifExists); err != nil {
						l.setError(err)
						return
					}
				}
			}
		}
	} else if ctx.Drop_type_name() != nil && ctx.Drop_type_name().SCHEMA() != nil {
		// DROP SCHEMA
		if ctx.Name_list() != nil {
			for _, schemaName := range ctx.Name_list().AllName() {
				if err := l.dropSchema(schemaName, ifExists); err != nil {
					l.setError(err)
					return
				}
			}
		}
	}
}

func (l *siPgCatalogListener) dropTable(anyName parser.IAny_nameContext, ifExists bool) error {
	parts := pgparser.NormalizePostgreSQLAnyName(anyName)
	if len(parts) == 0 {
		return nil
	}

	var schemaName, tableName string
	if len(parts) == 1 {
		schemaName = ""
		tableName = parts[0]
	} else {
		schemaName = parts[0]
		tableName = parts[1]
	}

	schema, err := l.databaseState.getSchema(schemaName)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}

	table, err := schema.pgGetTable(tableName)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}

	// Check for view dependencies
	viewList, err := l.databaseState.existedViewList(table.dependencyView)
	if err != nil {
		return err
	}
	if len(viewList) > 0 {
		return errors.Errorf("Cannot drop table %q.%q, it's referenced by view: %s", schema.name, table.name, strings.Join(viewList, ", "))
	}

	// Delete all indexes associated with the table
	for indexName := range table.indexSet {
		delete(schema.identifierMap, indexName)
	}

	delete(schema.identifierMap, table.name)
	delete(schema.tableSet, table.name)
	return nil
}

func (l *siPgCatalogListener) dropView(anyName parser.IAny_nameContext, ifExists bool) error {
	parts := pgparser.NormalizePostgreSQLAnyName(anyName)
	if len(parts) == 0 {
		return nil
	}

	var schemaName, viewName string
	if len(parts) == 1 {
		schemaName = ""
		viewName = parts[0]
	} else {
		schemaName = parts[0]
		viewName = parts[1]
	}

	schema, err := l.databaseState.getSchema(schemaName)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}

	delete(schema.identifierMap, viewName)
	delete(schema.viewSet, viewName)
	return nil
}

func (l *siPgCatalogListener) dropIndex(anyName parser.IAny_nameContext, ifExists bool) error {
	parts := pgparser.NormalizePostgreSQLAnyName(anyName)
	if len(parts) == 0 {
		return nil
	}

	var schemaName, indexName string
	if len(parts) == 1 {
		schemaName = ""
		indexName = parts[0]
	} else {
		schemaName = parts[0]
		indexName = parts[1]
	}

	schema, err := l.databaseState.getSchema(schemaName)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}

	table, index, err := schema.getIndex(indexName)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}

	delete(schema.identifierMap, index.name)
	delete(table.indexSet, index.name)
	return nil
}

func (l *siPgCatalogListener) dropSchema(schemaNameCtx parser.INameContext, ifExists bool) error {
	schemaName := pgparser.NormalizePostgreSQLName(schemaNameCtx)

	schema, exists := l.databaseState.schemaSet[schemaName]
	if !exists {
		if ifExists {
			return nil
		}
		return errors.Errorf("Schema %q does not exist", schemaName)
	}

	// Delete all identifiers in this schema
	for tableName := range schema.tableSet {
		delete(schema.identifierMap, tableName)
	}
	for viewName := range schema.viewSet {
		delete(schema.identifierMap, viewName)
	}

	// Delete the schema
	delete(l.databaseState.schemaSet, schemaName)
	return nil
}

// ========================================
// RENAME statements handling
// ========================================

// EnterRenamestmt handles RENAME INDEX/CONSTRAINT/TABLE/COLUMN statements.
func (l *siPgCatalogListener) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !siIsTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	if !l.checkDatabaseNotDeleted() {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Check if this is INDEX rename (ALTER INDEX ... RENAME TO ...)
	if ctx.INDEX() != nil {
		// ALTER INDEX index_name RENAME TO new_name
		if ctx.Qualified_name() != nil && ctx.AllName() != nil && len(ctx.AllName()) > 0 {
			indexName := siExtractTableName(ctx.Qualified_name())
			schemaName := siExtractSchemaName(ctx.Qualified_name())
			newName := pgparser.NormalizePostgreSQLName(ctx.AllName()[0])

			schema, err := l.databaseState.getSchema(schemaName)
			if err != nil {
				l.setError(err)
				return
			}

			// Find the index across all tables
			var foundIndex *siIndexState
			var foundTable *siTableState
			for _, table := range schema.tableSet {
				if index, exists := table.indexSet[indexName]; exists {
					foundIndex = index
					foundTable = table
					break
				}
			}

			if foundIndex == nil {
				// Index not found, silently ignore (PostgreSQL behavior)
				return
			}

			if err := l.renameConstraint(schema, foundTable, indexName, newName); err != nil {
				l.setError(err)
			}
		}
		return
	}

	// Extract relation (table) if present
	var tableName, schemaName string
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		tableName = siExtractTableName(ctx.Relation_expr().Qualified_name())
		schemaName = siExtractSchemaName(ctx.Relation_expr().Qualified_name())
	}

	schema, err := l.databaseState.getSchema(schemaName)
	if err != nil {
		l.setError(err)
		return
	}

	// Check if this is column rename
	if ctx.Opt_column() != nil {
		// RENAME COLUMN: ALTER TABLE table RENAME COLUMN oldname TO newname
		// Column names use Name(), not Colid() in RENAME statements
		allNames := ctx.AllName()
		if len(allNames) >= 2 && tableName != "" {
			oldName := pgparser.NormalizePostgreSQLName(allNames[0])
			newName := pgparser.NormalizePostgreSQLName(allNames[1])

			table, err := schema.pgGetTable(tableName)
			if err != nil {
				l.setError(err)
				return
			}

			if err := l.renameColumn(table, oldName, newName); err != nil {
				l.setError(err)
			}
		}
		return
	}

	// Check if this is constraint rename
	if ctx.CONSTRAINT() != nil && tableName != "" {
		// RENAME CONSTRAINT: ALTER TABLE table RENAME CONSTRAINT oldname TO newname
		allNames := ctx.AllName()
		if len(allNames) >= 2 {
			oldName := pgparser.NormalizePostgreSQLName(allNames[0])
			newName := pgparser.NormalizePostgreSQLName(allNames[1])

			table, err := schema.pgGetTable(tableName)
			if err != nil {
				l.setError(err)
				return
			}

			if err := l.renameConstraint(schema, table, oldName, newName); err != nil {
				l.setError(err)
			}
		}
		return
	}

	// Otherwise it's table rename: ALTER TABLE oldname RENAME TO newname
	if tableName != "" && ctx.AllName() != nil && len(ctx.AllName()) > 0 {
		newName := pgparser.NormalizePostgreSQLName(ctx.AllName()[0])

		table, err := schema.pgGetTable(tableName)
		if err != nil {
			l.setError(err)
			return
		}

		if err := l.renameTable(schema, table, newName); err != nil {
			l.setError(err)
		}
		return
	}
}

// ========================================
// CREATE VIEW handling
// ========================================

// EnterViewstmt handles CREATE VIEW statements.
func (l *siPgCatalogListener) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !siIsTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	if !l.checkDatabaseNotDeleted() {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Extract view name
	if ctx.Qualified_name() == nil {
		return
	}

	viewName := siExtractTableName(ctx.Qualified_name())
	schemaName := siExtractSchemaName(ctx.Qualified_name())
	databaseName := siExtractDatabaseName(ctx.Qualified_name())

	// Check if accessing other database
	if databaseName != "" && l.databaseState.name != databaseName {
		l.setError(errors.Errorf("Database %q is not the current database %q", databaseName, l.databaseState.name))
		return
	}

	schema, err := l.databaseState.getSchema(schemaName)
	if err != nil {
		l.setError(err)
		return
	}

	// Check if view already exists - currently we don't check views
	// This matches the legacy behavior in walk_through_for_pg.go:619-622
	if _, exists := schema.viewSet[viewName]; exists {
		return
	}

	// Create view state
	view := &siViewState{
		name: viewName,
	}
	schema.viewSet[view.name] = view
}

// ========================================
// Helper functions
// ========================================

// siIsTopLevel checks if the context is at the top level (not nested).
func siIsTopLevel(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}

	switch ctx := ctx.(type) {
	case *parser.RootContext, *parser.StmtblockContext:
		return true
	case *parser.StmtmultiContext, *parser.StmtContext:
		return siIsTopLevel(ctx.GetParent())
	default:
		return false
	}
}

// siExtractTypeName extracts the type name from a Typename context.
// Simply uses GetText() to get the full type representation.
// PostgreSQL normalizes some type names (e.g., int -> integer),
// which will be handled by the parser's type normalization.
func siExtractTypeName(typename parser.ITypenameContext) string {
	if typename == nil {
		return ""
	}

	// Use GetText() to get the full type string
	// The parser should have already normalized type names
	typeText := typename.GetText()

	// Normalize common PostgreSQL type aliases
	switch strings.ToLower(typeText) {
	case "int", "int4":
		return "integer"
	case "int2":
		return "smallint"
	case "int8":
		return "bigint"
	case "float4":
		return "real"
	case "float8":
		return "double precision"
	case "bool":
		return "boolean"
	default:
		return typeText
	}
}

// siExtractTableName extracts the table name from a qualified_name context.
// For "schema.table" or "db.schema.table", returns "table"
func siExtractTableName(qualifiedName parser.IQualified_nameContext) string {
	if qualifiedName == nil {
		return ""
	}

	parts := pgparser.NormalizePostgreSQLQualifiedName(qualifiedName)
	if len(parts) == 0 {
		return ""
	}
	// Last part is always the table/object name
	return parts[len(parts)-1]
}

// siExtractSchemaName extracts the schema name from a qualified_name context.
// For "schema.table", returns "schema"
// For "db.schema.table", returns "schema"
// For "table", returns ""
func siExtractSchemaName(qualifiedName parser.IQualified_nameContext) string {
	if qualifiedName == nil {
		return ""
	}

	parts := pgparser.NormalizePostgreSQLQualifiedName(qualifiedName)
	switch len(parts) {
	case 1:
		// Just table name, no schema
		return ""
	case 2:
		// schema.table
		return parts[0]
	case 3:
		// db.schema.table
		return parts[1]
	default:
		return ""
	}
}

// siExtractDatabaseName extracts the database name from a qualified_name context.
// For "db.schema.table", returns "db"
// For "schema.table" or "table", returns ""
func siExtractDatabaseName(qualifiedName parser.IQualified_nameContext) string {
	if qualifiedName == nil {
		return ""
	}

	parts := pgparser.NormalizePostgreSQLQualifiedName(qualifiedName)
	if len(parts) == 3 {
		// db.schema.table
		return parts[0]
	}
	return ""
}

// siGenerateIndexName generates an index name based on table name and columns.
// Format: tablename_col1_col2_idx (with suffix for uniqueness if needed)
func siGenerateIndexName(tableName string, columnList []string, _ bool) string {
	var builder strings.Builder
	builder.WriteString(tableName)

	expressionID := 0
	for _, col := range columnList {
		builder.WriteByte('_')
		if col == "expr" {
			builder.WriteString("expr")
			if expressionID > 0 {
				builder.WriteString(fmt.Sprintf("%d", expressionID))
			}
			expressionID++
		} else {
			builder.WriteString(col)
		}
	}

	builder.WriteString("_idx")

	return builder.String()
}

// siGenerateUniqueIndexName generates a unique index name by adding numeric suffixes.
// Tries name1, name2, name3, etc. until finding an available name.
func siGenerateUniqueIndexName(schema *siSchemaState, tableName string, columnList []string, isUnique bool) string {
	baseName := siGenerateIndexName(tableName, columnList, isUnique)

	// Try with numeric suffixes starting from 1
	for i := 1; i < 1000; i++ {
		candidateName := fmt.Sprintf("%s%d", baseName, i)
		if _, exists := schema.identifierMap[candidateName]; !exists {
			return candidateName
		}
	}

	// Fallback (should never reach here)
	return fmt.Sprintf("%s_collision", baseName)
}

// ========================================
// Database state helper methods
// ========================================

func (d *siDatabaseState) getSchema(schemaName string) (*siSchemaState, error) {
	if schemaName == "" {
		schemaName = siPublicSchemaName
	}
	schema, exists := d.schemaSet[schemaName]
	if !exists {
		if schemaName != siPublicSchemaName {
			return nil, siNewSchemaViolationError(1901, fmt.Sprintf("The schema %q doesn't exist", schemaName))
		}
		schema = &siSchemaState{
			ctx:           d.ctx.Copy(),
			name:          siPublicSchemaName,
			tableSet:      make(siTableStateMap),
			viewSet:       make(siViewStateMap),
			identifierMap: make(siIdentifierMap),
		}
		d.schemaSet[siPublicSchemaName] = schema
	}
	return schema, nil
}

func (*siDatabaseState) existedViewList(_ map[string]bool) ([]string, error) {
	// For now, return empty list - view dependencies not fully implemented
	return []string{}, nil
}

// ========================================
// Schema state helper methods
// ========================================

func (s *siSchemaState) pgGetTable(tableName string) (*siTableState, error) {
	table, exists := s.tableSet[tableName]
	if !exists {
		return nil, siNewSchemaViolationError(604, fmt.Sprintf("The table %q does not exist in schema %q", tableName, s.name))
	}
	return table, nil
}

func (s *siSchemaState) getIndex(indexName string) (*siTableState, *siIndexState, error) {
	for _, table := range s.tableSet {
		if index, exists := table.indexSet[indexName]; exists {
			return table, index, nil
		}
	}

	return nil, nil, siNewSchemaViolationError(809, fmt.Sprintf("Index %q does not exists in schema %q", indexName, s.name))
}

func (s *siSchemaState) pgGeneratePrimaryKeyName(tableName string) string {
	pkName := fmt.Sprintf("%s_pkey", tableName)
	if _, exists := s.identifierMap[pkName]; !exists {
		return pkName
	}
	suffix := 1
	for {
		if _, exists := s.identifierMap[fmt.Sprintf("%s%d", pkName, suffix)]; !exists {
			return fmt.Sprintf("%s%d", pkName, suffix)
		}
		suffix++
	}
}

// ========================================
// Table state helper methods
// ========================================

func (t *siTableState) getColumn(columnName string) (*siColumnState, error) {
	column, exists := t.columnSet[columnName]
	if !exists {
		return nil, siNewSchemaViolationError(405, fmt.Sprintf("The column %q does not exist in the table %q", columnName, t.name))
	}
	return column, nil
}
