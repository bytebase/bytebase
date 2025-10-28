package catalog

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	parser "github.com/bytebase/parser/postgresql"
)

// pgAntlrWalkThrough walks through the ANTLR parse tree and builds catalog state.
func (d *DatabaseState) pgAntlrWalkThrough(tree any) error {
	root, ok := tree.(parser.IRootContext)
	if !ok {
		return errors.Errorf("invalid ANTLR tree type %T", tree)
	}

	// Build listener with database state
	listener := &pgAntlrCatalogListener{
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

// pgAntlrCatalogListener builds catalog state by listening to ANTLR parse tree events.
type pgAntlrCatalogListener struct {
	*parser.BasePostgreSQLParserListener

	databaseState *DatabaseState
	err           *WalkThroughError
	currentLine   int
}

// Helper method to set error with line number
func (l *pgAntlrCatalogListener) setError(err *WalkThroughError) {
	if l.err != nil {
		return // Keep first error
	}
	if err != nil && err.Line == 0 {
		err.Line = l.currentLine
	}
	l.err = err
}

// Helper method to check if database is deleted
func (l *pgAntlrCatalogListener) checkDatabaseNotDeleted() bool {
	if l.databaseState.deleted {
		l.setError(&WalkThroughError{
			Type:    ErrorTypeDatabaseIsDeleted,
			Content: fmt.Sprintf(`Database %q is deleted`, l.databaseState.name),
		})
		return false
	}
	return true
}

// ========================================
// CREATE TABLE handling
// ========================================

// EnterCreatestmt handles CREATE TABLE statements.
func (l *pgAntlrCatalogListener) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
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

	tableName := extractTableName(qualifiedNames[0])
	schemaName := extractSchemaName(qualifiedNames[0])

	if tableName == "" {
		return
	}

	// Check database name if specified
	databaseName := extractDatabaseName(qualifiedNames[0])
	if databaseName != "" && l.databaseState.name != databaseName {
		l.setError(&WalkThroughError{
			Type:    ErrorTypeAccessOtherDatabase,
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
			Type:    ErrorTypeTableExists,
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
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("The column %q already exists in table %q", columnName, table.name),
		}
	}

	// Get column type
	var columnType string
	if columnDef.Typename() != nil {
		// TODO: We need to deparse the type, for now just use a placeholder
		columnType = "text" // This should be extracted from Typename() context
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
				// TODO: Extract default expression
				defaultValue := "DEFAULT" // Placeholder
				columnState.defaultValue = &defaultValue
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

				// Only create index if constraint has a name (unnamed constraints auto-generated)
				if constraintName != "" {
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

		// Only create index if constraint has a name
		if constraintName != "" {
			// Check if identifier already exists
			if _, exists := schema.identifierMap[constraintName]; exists {
				return NewRelationExistsError(constraintName, schema.name)
			}

			// Validate columns exist
			for _, colName := range columnList {
				if _, exists := table.columnSet[colName]; !exists {
					return NewColumnNotExistsError(table.name, colName)
				}
			}

			// Create unique index
			index := &IndexState{
				name:           constraintName,
				expressionList: columnList,
				indexType:      newStringPointer("btree"),
				unique:         newTruePointer(),
				primary:        newFalsePointer(),
				isConstraint:   true,
			}
			table.indexSet[index.name] = index
			schema.identifierMap[index.name] = true
		}
	}

	// Note: We skip CHECK, FOREIGN KEY, EXCLUSION constraints for now
	// as the legacy implementation also skips them

	return nil
}

// ========================================
// CREATE INDEX handling
// ========================================

// EnterIndexstmt handles CREATE INDEX statements.
func (l *pgAntlrCatalogListener) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
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
			Type:    ErrorTypeIndexEmptyKeys,
			Content: fmt.Sprintf("Index %q in table %q has empty key", indexName, tableName),
		})
		return
	}

	// Generate index name if not provided
	isUnique := ctx.Opt_unique() != nil && ctx.Opt_unique().UNIQUE() != nil
	if indexName == "" {
		indexName = generateIndexName(tableName, columnList, isUnique)
	}

	// Check if index name already exists
	if _, exists := schema.identifierMap[indexName]; exists {
		if ifNotExists {
			return
		}
		l.setError(NewRelationExistsError(indexName, schema.name))
		return
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
// func (l *pgAntlrCatalogListener) EnterCreateschemastatement(ctx *parser.CreateschemaContext) {
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
func (l *pgAntlrCatalogListener) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	if !l.checkDatabaseNotDeleted() {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// TODO: Implement ALTER TABLE logic
	// Similar to pgAlterTable() in walk_through_for_pg.go
	// This is complex - handles:
	// - RENAME COLUMN
	// - RENAME CONSTRAINT
	// - RENAME TABLE
	// - SET SCHEMA
	// - ADD COLUMN
	// - DROP COLUMN
	// - ALTER COLUMN TYPE
	// - SET DEFAULT
	// - DROP DEFAULT
	// - SET NOT NULL
	// - DROP NOT NULL
	// - ADD CONSTRAINT
	// - DROP CONSTRAINT
}

// ========================================
// DROP statements handling
// ========================================

// EnterDropstmt handles DROP TABLE/VIEW/INDEX statements.
func (l *pgAntlrCatalogListener) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
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
		}
	} else if ctx.INDEX() != nil {
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
}

func (l *pgAntlrCatalogListener) dropTable(anyName parser.IAny_nameContext, ifExists bool) *WalkThroughError {
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
		return &WalkThroughError{
			Type:    ErrorTypeTableIsReferencedByView,
			Content: fmt.Sprintf("Cannot drop table %q.%q, it's referenced by view: %s", schema.name, table.name, strings.Join(viewList, ", ")),
			Payload: viewList,
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

func (l *pgAntlrCatalogListener) dropView(anyName parser.IAny_nameContext, ifExists bool) *WalkThroughError {
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

func (l *pgAntlrCatalogListener) dropIndex(anyName parser.IAny_nameContext, ifExists bool) *WalkThroughError {
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

// TODO: EnterDropindexstmt - Need to find correct ANTLR context name
// func (l *pgAntlrCatalogListener) EnterDropindexstmt(ctx *parser.DropIndexContext) {
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
// func (l *pgAntlrCatalogListener) EnterDropschemastatement(ctx *parser.DropschemaContext) {
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
// TODO: Implement full RENAME support - currently stubbed to avoid compilation errors
func (l *pgAntlrCatalogListener) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
		return
	}

	if !l.checkDatabaseNotDeleted() {
		return
	}

	l.currentLine = ctx.GetStart().GetLine()

	// TODO: Need to determine correct ANTLR context structure for extracting old/new names
	// For now, just return to make it compile - we'll implement properly after running tests
	// to see the actual structure
	return
}

// ========================================
// CREATE VIEW handling
// ========================================

// EnterViewstmt handles CREATE VIEW statements.
func (l *pgAntlrCatalogListener) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.err != nil {
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

	viewName := extractTableName(ctx.Qualified_name())
	schemaName := extractSchemaName(ctx.Qualified_name())
	databaseName := extractDatabaseName(ctx.Qualified_name())

	// Check if accessing other database
	if databaseName != "" && l.databaseState.name != databaseName {
		l.setError(&WalkThroughError{
			Type:    ErrorTypeAccessOtherDatabase,
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
func generateIndexName(tableName string, columnList []string, isUnique bool) string {
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
	if isUnique {
		builder.WriteString("1") // Unique indexes get a "1" suffix initially
	}

	return builder.String()
}
