package catalog

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// PgWalkThrough walks through the PostgreSQL ANTLR parse tree and builds catalog state.
func PgWalkThrough(d *DatabaseState, ast any) error {
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
	listener := &pgCatalogListener{
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

// pgCatalogListener builds catalog state by listening to ANTLR parse tree events.
type pgCatalogListener struct {
	*parser.BasePostgreSQLParserListener

	databaseState *DatabaseState
	err           *WalkThroughError
	currentLine   int
}

// Helper method to set error with line number
func (l *pgCatalogListener) setError(err *WalkThroughError) {
	if l.err != nil {
		return // Keep first error
	}
	if err != nil && err.Line == 0 {
		err.Line = l.currentLine
	}
	l.err = err
}

// ========================================
// CREATE TABLE handling
// ========================================

// EnterCreatestmt handles CREATE TABLE statements.
func (l *pgCatalogListener) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Extract table name and schema
	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := extractTableName(qualifiedNames[0])
	schemaName := extractSchemaName(qualifiedNames[0])

	if tableName == "" {
		return
	}

	// Check database name if specified
	databaseName := extractDatabaseName(qualifiedNames[0])
	if databaseName != "" && l.databaseState.name != databaseName {
		l.setError(&WalkThroughError{
			Code:    code.NotCurrentDatabase,
			Content: fmt.Sprintf("Database %q is not the current database %q", databaseName, l.databaseState.name),
		})
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
		l.setError(&WalkThroughError{
			Code:    code.TableExists,
			Content: fmt.Sprintf(`The table %q already exists in the schema %q`, tableName, schema.name),
		})
		return
	}

	// Create table state
	table := &TableState{
		name:      tableName,
		columnSet: make(columnStateMap),
		indexSet:  make(IndexStateMap),
	}
	schema.tableSet[table.name] = table

	// Process column definitions
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Handle column definitions
			if elem.ColumnDef() != nil {
				if err := createColumn(schema, table, elem.ColumnDef()); err != nil {
					l.setError(err)
					return
				}
			}
			// Handle table-level constraints
			if elem.Tableconstraint() != nil {
				if err := createTableConstraint(schema, table, elem.Tableconstraint()); err != nil {
					l.setError(err)
					return
				}
			}
		}
	}
}

// createColumn creates a column in the table.
func createColumn(schema *SchemaState, table *TableState, columnDef parser.IColumnDefContext) *WalkThroughError {
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
		return &WalkThroughError{
			Code:    code.ColumnExists,
			Content: fmt.Sprintf("The column %q already exists in table %q", columnName, table.name),
		}
	}

	// Get column type
	var columnType string
	if columnDef.Typename() != nil {
		columnType = extractTypeName(columnDef.Typename())
	}

	// Create column state
	pos := len(table.columnSet) + 1
	columnState := &ColumnState{
		name:         columnName,
		position:     &pos,
		defaultValue: nil,
		nullable:     newTruePointer(),
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
				columnState.nullable = newFalsePointer()
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
				columnState.nullable = newFalsePointer()

				// Create primary key index
				index := &IndexState{
					name:           constraintName,
					expressionList: []string{columnName},
					indexType:      newStringPointer("btree"),
					unique:         newTruePointer(),
					primary:        newTruePointer(),
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
					constraintName = generateIndexName(table.name, []string{columnName}, true)
				}

				index := &IndexState{
					name:           constraintName,
					expressionList: []string{columnName},
					indexType:      newStringPointer("btree"),
					unique:         newTruePointer(),
					primary:        newFalsePointer(),
					isConstraint:   true,
				}
				table.indexSet[index.name] = index
				schema.identifierMap[index.name] = true
			}
		}
	}

	return nil
}

// createTableConstraint creates a table-level constraint.
func createTableConstraint(schema *SchemaState, table *TableState, constraint parser.ITableconstraintContext) *WalkThroughError {
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
				column.nullable = newFalsePointer()
			} else {
				return NewColumnNotExistsError(table.name, colName)
			}
		}

		// Generate PK name if not provided
		pkName := constraintName
		if pkName == "" {
			pkName = schema.pgGeneratePrimaryKeyName(table.name)
		}

		// Check if identifier already exists
		if _, exists := schema.identifierMap[pkName]; exists {
			return NewRelationExistsError(pkName, schema.name)
		}

		// Create primary key index
		index := &IndexState{
			name:           pkName,
			expressionList: columnList,
			indexType:      newStringPointer("btree"),
			unique:         newTruePointer(),
			primary:        newTruePointer(),
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
			indexName = generateIndexName(table.name, columnList, true)
		}

		// Check if identifier already exists
		if _, exists := schema.identifierMap[indexName]; exists {
			return NewRelationExistsError(indexName, schema.name)
		}

		// Validate columns exist
		for _, colName := range columnList {
			if _, exists := table.columnSet[colName]; !exists {
				return NewColumnNotExistsError(table.name, colName)
			}
		}

		// Create unique index
		index := &IndexState{
			name:           indexName,
			expressionList: columnList,
			indexType:      newStringPointer("btree"),
			unique:         newTruePointer(),
			primary:        newFalsePointer(),
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
func (l *pgCatalogListener) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Extract relation (table) name
	relationExpr := ctx.Relation_expr()
	if relationExpr == nil || relationExpr.Qualified_name() == nil {
		return
	}

	tableName := extractTableName(relationExpr.Qualified_name())
	schemaName := extractSchemaName(relationExpr.Qualified_name())
	schema, err := l.databaseState.getSchema(schemaName)
	if err != nil {
		l.setError(err)
		return
	}

	table, exists := schema.tableSet[tableName]
	if !exists {
		l.setError(NewTableNotExistsError(tableName))
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
		l.setError(&WalkThroughError{
			Code:    code.IndexEmptyKeys,
			Content: fmt.Sprintf("Index %q in table %q has empty key", indexName, tableName),
		})
		return
	}

	// Generate index name if not provided
	isUnique := ctx.Opt_unique() != nil && ctx.Opt_unique().UNIQUE() != nil
	wasAutoGenerated := indexName == ""
	if indexName == "" {
		indexName = generateIndexName(tableName, columnList, isUnique)
	}

	// Check if index name already exists
	if _, exists := schema.identifierMap[indexName]; exists {
		if ifNotExists {
			return
		}
		// If name was auto-generated, try with numeric suffix
		if wasAutoGenerated {
			indexName = generateUniqueIndexName(schema, tableName, columnList, isUnique)
		} else {
			l.setError(NewRelationExistsError(indexName, schema.name))
			return
		}
	}

	// Check that all columns exist (skip expressions)
	for _, colName := range columnList {
		if colName != "expr" {
			if _, exists := table.columnSet[colName]; !exists {
				l.setError(NewColumnNotExistsError(tableName, colName))
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
	index := &IndexState{
		name:           indexName,
		expressionList: columnList,
		indexType:      newStringPointer(indexType),
		unique:         newBoolPointer(isUnique),
		primary:        newFalsePointer(),
		isConstraint:   false,
	}

	table.indexSet[index.name] = index
	schema.identifierMap[index.name] = true
}

// ========================================
// CREATE SCHEMA handling
// ========================================

// TODO: EnterCreateschemastatement - Need to find correct ANTLR context name
// func (l *pgCatalogListener) EnterCreateschemastatement(ctx *parser.CreateschemaContext) {
// 	if !isTopLevel(ctx.GetParent()) || l.err != nil {
// 		return
// 	}
//
// 	if !l.checkDatabaseNotDeleted() {
// 		return
// 	}
//
// 	l.currentLine = ctx.GetStart().GetLine()
//
// 	// TODO: Implement CREATE SCHEMA logic
// 	// Similar to pgCreateSchema() in walk_through_for_pg.go
// }

// ========================================
// ALTER TABLE handling
// ========================================

// EnterAltertablestmt handles ALTER TABLE statements.
func (l *pgCatalogListener) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Extract table name
	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := extractTableName(ctx.Relation_expr().Qualified_name())
	schemaName := extractSchemaName(ctx.Relation_expr().Qualified_name())
	databaseName := extractDatabaseName(ctx.Relation_expr().Qualified_name())

	// Check database access
	if databaseName != "" && l.databaseState.name != databaseName {
		l.setError(&WalkThroughError{
			Code:    code.NotCurrentDatabase,
			Content: fmt.Sprintf("Database %q is not the current database %q", databaseName, l.databaseState.name),
		})
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
func (l *pgCatalogListener) processAlterTableCmd(schema *SchemaState, table *TableState, cmd parser.IAlter_table_cmdContext) {
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
				typeString := extractTypeName(cmd.Typename())
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
func (l *pgCatalogListener) alterTableDropColumn(schema *SchemaState, table *TableState, columnName string, ifExists bool) {
	column, exists := table.columnSet[columnName]
	if !exists {
		if ifExists {
			return
		}
		l.setError(NewColumnNotExistsError(table.name, columnName))
		return
	}

	// Check if column is referenced by any views
	viewList, err := l.databaseState.existedViewList(column.dependencyView)
	if err != nil {
		l.setError(&WalkThroughError{
			Code:    code.Internal,
			Content: fmt.Sprintf("Failed to check view dependency: %v", err),
		})
		return
	}
	if len(viewList) > 0 {
		l.setError(&WalkThroughError{
			Code:    code.ColumnIsReferencedByView,
			Content: fmt.Sprintf("Cannot drop column %q in table %q.%q, it's referenced by view: %s", column.name, schema.name, table.name, strings.Join(viewList, ", ")),
		})
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
func (l *pgCatalogListener) alterTableAlterColumnType(schema *SchemaState, table *TableState, columnName string, typeString string) {
	column, err := table.getColumn(columnName)
	if err != nil {
		l.setError(err)
		return
	}

	// Check if column is referenced by any views
	viewList, viewErr := l.databaseState.existedViewList(column.dependencyView)
	if viewErr != nil {
		l.setError(&WalkThroughError{
			Code:    code.Internal,
			Content: fmt.Sprintf("Failed to check view dependency: %v", viewErr),
		})
		return
	}
	if len(viewList) > 0 {
		l.setError(&WalkThroughError{
			Code:    code.ColumnIsReferencedByView,
			Content: fmt.Sprintf("Cannot alter type of column %q in table %q.%q, it's referenced by view: %s", column.name, schema.name, table.name, strings.Join(viewList, ", ")),
		})
		return
	}

	// Update column type
	column.columnType = &typeString
}

// alterTableAddColumn handles ADD COLUMN command.
func (l *pgCatalogListener) alterTableAddColumn(schema *SchemaState, table *TableState, columndef parser.IColumnDefContext, ifNotExists bool) {
	if columndef == nil {
		return
	}

	columnName := pgparser.NormalizePostgreSQLColid(columndef.Colid())

	// Check if column already exists
	if _, exists := table.columnSet[columnName]; exists {
		if ifNotExists {
			return
		}
		l.setError(&WalkThroughError{
			Code:    code.ColumnExists,
			Content: fmt.Sprintf("The column %q already exists in table %q", columnName, table.name),
		})
		return
	}

	// Get position for new column
	pos := len(table.columnSet) + 1

	// Extract column type
	var typeString string
	if columndef.Typename() != nil {
		typeString = extractTypeName(columndef.Typename())
	}

	// Create column state
	columnState := &ColumnState{
		name:         columnName,
		position:     &pos,
		nullable:     newTruePointer(),
		columnType:   &typeString,
		defaultValue: nil,
	}
	table.columnSet[columnName] = columnState

	// Process column constraints if any (inline processing like in createColumn)
	if columndef.Colquallist() != nil {
		allQuals := columndef.Colquallist().AllColconstraint()
		for _, qual := range allQuals {
			if qual.Colconstraintelem() == nil {
				continue
			}
			elem := qual.Colconstraintelem()

			// Handle NOT NULL
			if elem.NOT() != nil && elem.NULL_P() != nil {
				columnState.nullable = newFalsePointer()
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
					constraintName = generateIndexName(table.name, []string{columnName}, true)
				}
				// Check for collision
				if _, exists := schema.identifierMap[constraintName]; exists {
					constraintName = generateUniqueIndexName(schema, table.name, []string{columnName}, true)
				}
				// Create index
				index := &IndexState{
					name:           constraintName,
					expressionList: []string{columnName},
					indexType:      newStringPointer("btree"),
					unique:         newTruePointer(),
					primary:        newFalsePointer(),
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
					l.setError(NewRelationExistsError(constraintName, schema.name))
					return
				}
				columnState.nullable = newFalsePointer()
				// Create primary key index
				index := &IndexState{
					name:           constraintName,
					expressionList: []string{columnName},
					indexType:      newStringPointer("btree"),
					unique:         newTruePointer(),
					primary:        newTruePointer(),
					isConstraint:   true,
				}
				table.indexSet[index.name] = index
				schema.identifierMap[index.name] = true
			}
		}
	}
}

// alterTableAddConstraint handles ADD CONSTRAINT command.
func (l *pgCatalogListener) alterTableAddConstraint(schema *SchemaState, table *TableState, constraint parser.ITableconstraintContext) {
	if constraint == nil {
		return
	}

	// Reuse the constraint creation logic from CREATE TABLE
	err := createTableConstraint(schema, table, constraint)
	if err != nil {
		l.setError(err)
	}
}

// alterTableDropConstraint handles DROP CONSTRAINT command.
func (l *pgCatalogListener) alterTableDropConstraint(schema *SchemaState, table *TableState, constraintName string, ifExists bool) {
	// Check if constraint exists as an index
	if index, exists := table.indexSet[constraintName]; exists {
		delete(schema.identifierMap, index.name)
		delete(table.indexSet, index.name)
		return
	}

	if !ifExists {
		l.setError(&WalkThroughError{
			Code:    code.ConstraintNotExists,
			Content: fmt.Sprintf("Constraint %q for table %q does not exist", constraintName, table.name),
		})
	}
}

// alterTableSetDefault handles ALTER COLUMN SET DEFAULT command.
func (l *pgCatalogListener) alterTableSetDefault(table *TableState, columnName string, defaultValue string) {
	column, err := table.getColumn(columnName)
	if err != nil {
		l.setError(err)
		return
	}

	column.defaultValue = &defaultValue
}

// alterTableDropDefault handles ALTER COLUMN DROP DEFAULT command.
func (l *pgCatalogListener) alterTableDropDefault(table *TableState, columnName string) {
	column, err := table.getColumn(columnName)
	if err != nil {
		l.setError(err)
		return
	}

	column.defaultValue = nil
}

// alterTableSetNotNull handles ALTER COLUMN SET NOT NULL command.
func (l *pgCatalogListener) alterTableSetNotNull(table *TableState, columnName string) {
	column, err := table.getColumn(columnName)
	if err != nil {
		l.setError(err)
		return
	}

	column.nullable = newFalsePointer()
}

// renameTable handles RENAME TO for tables.
func (*pgCatalogListener) renameTable(schema *SchemaState, table *TableState, newName string) *WalkThroughError {
	// Check if new name already exists
	if _, exists := schema.identifierMap[newName]; exists {
		return NewRelationExistsError(newName, schema.name)
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
func (*pgCatalogListener) renameColumn(table *TableState, oldName string, newName string) *WalkThroughError {
	column, err := table.getColumn(oldName)
	if err != nil {
		return err
	}

	if oldName == newName {
		return nil
	}

	// Check if new name already exists
	if _, exists := table.columnSet[newName]; exists {
		return &WalkThroughError{
			Code:    code.ColumnExists,
			Content: fmt.Sprintf("The column %q already exists in table %q", newName, table.name),
		}
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
func (*pgCatalogListener) renameConstraint(schema *SchemaState, table *TableState, oldName string, newName string) *WalkThroughError {
	index, exists := table.indexSet[oldName]
	if !exists {
		// We haven't dealt with foreign and check constraints, so skip if not exists
		return nil
	}

	// Check if new name already exists
	if _, exists := schema.identifierMap[newName]; exists {
		return NewRelationExistsError(newName, schema.name)
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
func (l *pgCatalogListener) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
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

// ========================================
// DROP DATABASE handling
// ========================================

// EnterDropdbstmt handles DROP DATABASE statements.
// PostgreSQL does not allow dropping the currently connected database.
// Walk-through also disallows DROP DATABASE for other databases as it's out of scope.
func (l *pgCatalogListener) EnterDropdbstmt(ctx *parser.DropdbstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Extract database name
	if ctx.Name() == nil {
		return
	}

	databaseName := pgparser.NormalizePostgreSQLName(ctx.Name())

	// PostgreSQL does not allow dropping the currently open database.
	// This matches the real PostgreSQL behavior: "ERROR: cannot drop the currently open database"
	if l.databaseState.isCurrentDatabase(databaseName) {
		l.setError(&WalkThroughError{
			Code:    code.NotCurrentDatabase,
			Content: fmt.Sprintf("Cannot drop the currently open database %q", databaseName),
		})
		return
	}

	// DROP DATABASE for other databases is out of scope for single-database walk-through
	l.setError(NewAccessOtherDatabaseError(l.databaseState.name, databaseName))
}

func (l *pgCatalogListener) dropTable(anyName parser.IAny_nameContext, ifExists bool) *WalkThroughError {
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
	viewList, viewErr := l.databaseState.existedViewList(table.dependencyView)
	if viewErr != nil {
		return &WalkThroughError{
			Code:    code.Internal,
			Content: fmt.Sprintf("Failed to check view dependency: %v", viewErr),
		}
	}
	if len(viewList) > 0 {
		return &WalkThroughError{
			Code:    code.TableIsReferencedByView,
			Content: fmt.Sprintf("Cannot drop table %q.%q, it's referenced by view: %s", schema.name, table.name, strings.Join(viewList, ", ")),
		}
	}

	// Delete all indexes associated with the table
	for indexName := range table.indexSet {
		delete(schema.identifierMap, indexName)
	}

	delete(schema.identifierMap, table.name)
	delete(schema.tableSet, table.name)
	return nil
}

func (l *pgCatalogListener) dropView(anyName parser.IAny_nameContext, ifExists bool) *WalkThroughError {
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

func (l *pgCatalogListener) dropIndex(anyName parser.IAny_nameContext, ifExists bool) *WalkThroughError {
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

func (l *pgCatalogListener) dropSchema(schemaNameCtx parser.INameContext, ifExists bool) *WalkThroughError {
	schemaName := pgparser.NormalizePostgreSQLName(schemaNameCtx)

	schema, exists := l.databaseState.schemaSet[schemaName]
	if !exists {
		if ifExists {
			return nil
		}
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
	delete(l.databaseState.schemaSet, schemaName)
	return nil
}

// TODO: EnterDropindexstmt - Need to find correct ANTLR context name
// func (l *pgCatalogListener) EnterDropindexstmt(ctx *parser.DropIndexContext) {
// 	if !isTopLevel(ctx.GetParent()) || l.err != nil {
// 		return
// 	}
//
// 	if !l.checkDatabaseNotDeleted() {
// 		return
// 	}
//
// 	l.currentLine = ctx.GetStart().GetLine()
//
// 	// TODO: Implement DROP INDEX logic
// 	// Similar to pgDropIndexList() in walk_through_for_pg.go
// }

// TODO: EnterDropschemastatement - Need to find correct ANTLR context name
// func (l *pgCatalogListener) EnterDropschemastatement(ctx *parser.DropschemaContext) {
// 	if !isTopLevel(ctx.GetParent()) || l.err != nil {
// 		return
// 	}
//
// 	if !l.checkDatabaseNotDeleted() {
// 		return
// 	}
//
// 	l.currentLine = ctx.GetStart().GetLine()
//
// 	// TODO: Implement DROP SCHEMA logic
// 	// Similar to pgDropSchema() in walk_through_for_pg.go
// }

// ========================================
// RENAME statements handling
// ========================================

// EnterRenamestmt handles RENAME INDEX/CONSTRAINT/TABLE/COLUMN statements.
func (l *pgCatalogListener) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Check if this is INDEX rename (ALTER INDEX ... RENAME TO ...)
	if ctx.INDEX() != nil {
		// ALTER INDEX index_name RENAME TO new_name
		if ctx.Qualified_name() != nil && ctx.AllName() != nil && len(ctx.AllName()) > 0 {
			indexName := extractTableName(ctx.Qualified_name())
			schemaName := extractSchemaName(ctx.Qualified_name())
			newName := pgparser.NormalizePostgreSQLName(ctx.AllName()[0])

			schema, err := l.databaseState.getSchema(schemaName)
			if err != nil {
				l.setError(err)
				return
			}

			// Find the index across all tables
			var foundIndex *IndexState
			var foundTable *TableState
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
		tableName = extractTableName(ctx.Relation_expr().Qualified_name())
		schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
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
func (l *pgCatalogListener) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// Extract view name
	if ctx.Qualified_name() == nil {
		return
	}

	viewName := extractTableName(ctx.Qualified_name())
	schemaName := extractSchemaName(ctx.Qualified_name())
	databaseName := extractDatabaseName(ctx.Qualified_name())

	// Check if accessing other database
	if databaseName != "" && l.databaseState.name != databaseName {
		l.setError(&WalkThroughError{
			Code:    code.NotCurrentDatabase,
			Content: fmt.Sprintf("Database %q is not the current database %q", databaseName, l.databaseState.name),
		})
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
	view := &ViewState{
		name: viewName,
	}
	schema.viewSet[view.name] = view
}

// ========================================
// Helper functions
// ========================================

// isTopLevel checks if the context is at the top level (not nested).
func isTopLevel(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}

	switch ctx := ctx.(type) {
	case *parser.RootContext, *parser.StmtblockContext:
		return true
	case *parser.StmtmultiContext, *parser.StmtContext:
		return isTopLevel(ctx.GetParent())
	default:
		return false
	}
}

// extractTypeName extracts the type name from a Typename context.
// Simply uses GetText() to get the full type representation.
// PostgreSQL normalizes some type names (e.g., int -> integer),
// which will be handled by the parser's type normalization.
func extractTypeName(typename parser.ITypenameContext) string {
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

// extractTableName extracts the table name from a qualified_name context.
// For "schema.table" or "db.schema.table", returns "table"
func extractTableName(qualifiedName parser.IQualified_nameContext) string {
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

// extractSchemaName extracts the schema name from a qualified_name context.
// For "schema.table", returns "schema"
// For "db.schema.table", returns "schema"
// For "table", returns ""
func extractSchemaName(qualifiedName parser.IQualified_nameContext) string {
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

// extractDatabaseName extracts the database name from a qualified_name context.
// For "db.schema.table", returns "db"
// For "schema.table" or "table", returns ""
func extractDatabaseName(qualifiedName parser.IQualified_nameContext) string {
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

// generateIndexName generates an index name based on table name and columns.
// Format: tablename_col1_col2_idx (with suffix for uniqueness if needed)
func generateIndexName(tableName string, columnList []string, _ bool) string {
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

// generateUniqueIndexName generates a unique index name by adding numeric suffixes.
// Tries name1, name2, name3, etc. until finding an available name.
func generateUniqueIndexName(schema *SchemaState, tableName string, columnList []string, isUnique bool) string {
	baseName := generateIndexName(tableName, columnList, isUnique)

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
