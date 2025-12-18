package pg

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	// PublicSchemaName is the default schema name for PostgreSQL.
	PublicSchemaName = "public"
)

func init() {
	schema.RegisterWalkThrough(storepb.Engine_POSTGRES, WalkThrough)
}

// WalkThrough walks through the PostgreSQL ANTLR parse tree and builds catalog metadata.
func WalkThrough(d *model.DatabaseMetadata, ast []base.AST) *storepb.Advice {
	// Extract ANTLRAST from AST
	var antlrASTList []*base.ANTLRAST
	for _, unifiedAST := range ast {
		antlrAST, ok := base.GetANTLRAST(unifiedAST)
		if !ok {
			return &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.Internal.Int32(),
				Title:   "PostgreSQL walk-through expects ANTLR-based parser result",
				Content: "PostgreSQL walk-through expects ANTLR-based parser result",
				StartPosition: &storepb.Position{
					Line: 0,
				},
			}
		}
		antlrASTList = append(antlrASTList, antlrAST)
	}

	// Build listener with database state
	listener := &pgCatalogListener{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		databaseState:                d,
	}

	// Walk through all parse results
	for _, antlrAST := range antlrASTList {
		root, ok := antlrAST.Tree.(parser.IRootContext)
		if !ok {
			return &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.Internal.Int32(),
				Title:   fmt.Sprintf("invalid ANTLR tree type %T", antlrAST.Tree),
				Content: fmt.Sprintf("invalid ANTLR tree type %T", antlrAST.Tree),
				StartPosition: &storepb.Position{
					Line: 0,
				},
			}
		}
		// Walk through the parse tree
		antlr.ParseTreeWalkerDefault.Walk(listener, root)
		// Return immediately if error encountered
		if listener.advice != nil {
			return listener.advice
		}
	}

	// Return any error encountered during walk
	return listener.advice
}

// pgCatalogListener builds catalog state by listening to ANTLR parse tree events.
type pgCatalogListener struct {
	*parser.BasePostgreSQLParserListener
	databaseState *model.DatabaseMetadata
	advice        *storepb.Advice
	currentLine   int
}

// ========================================
// CREATE TABLE handling
// ========================================
// EnterCreatestmt handles CREATE TABLE statements.
func (l *pgCatalogListener) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.advice != nil {
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
	if databaseName != "" && l.databaseState.DatabaseName() != databaseName {
		l.advice = &storepb.Advice{
			Status:  storepb.Advice_WARNING,
			Code:    code.NotCurrentDatabase.Int32(),
			Title:   fmt.Sprintf("Database %q is not the current database %q", databaseName, l.databaseState.DatabaseName()),
			Content: fmt.Sprintf("Database %q is not the current database %q", databaseName, l.databaseState.DatabaseName()),
			StartPosition: &storepb.Position{
				Line: int32(l.currentLine),
			},
		}
		return
	}
	// Get or create schema
	schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
	if err != nil {
		l.advice = err
		return
	}
	// Check if table already exists
	if schema.GetTable(tableName) != nil {
		// Check IF NOT EXISTS clause
		ifNotExists := ctx.IF_P() != nil && ctx.NOT() != nil && ctx.EXISTS() != nil
		if ifNotExists {
			return
		}
		l.advice = &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    code.TableExists.Int32(),
			Title:   fmt.Sprintf(`The table %q already exists in the schema %q`, tableName, schema.GetProto().Name),
			Content: fmt.Sprintf(`The table %q already exists in the schema %q`, tableName, schema.GetProto().Name),
			StartPosition: &storepb.Position{
				Line: int32(l.currentLine),
			},
		}
		return
	}
	// Create table
	table, createErr := schema.CreateTable(tableName)
	if createErr != nil {
		l.advice = &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    code.TableExists.Int32(),
			Title:   createErr.Error(),
			Content: createErr.Error(),
			StartPosition: &storepb.Position{
				Line: int32(l.currentLine),
			},
		}
		return
	}
	// Process column definitions
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Handle column definitions
			if elem.ColumnDef() != nil {
				if err := pgCreateColumn(schema, table, elem.ColumnDef()); err != nil {
					l.advice = err
					return
				}
			}
			// Handle table-level constraints
			if elem.Tableconstraint() != nil {
				if err := createTableConstraint(schema, table, elem.Tableconstraint(), l.currentLine); err != nil {
					l.advice = err
					return
				}
			}
		}
	}
}

// pgCreateColumn creates a column in the table.
func pgCreateColumn(schema *model.SchemaMetadata, table *model.TableMetadata, columnDef parser.IColumnDefContext) *storepb.Advice {
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
	if table.GetColumn(columnName) != nil {
		return &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    code.ColumnExists.Int32(),
			Title:   fmt.Sprintf("The column %q already exists in table %q", columnName, table.GetProto().Name),
			Content: fmt.Sprintf("The column %q already exists in table %q", columnName, table.GetProto().Name),
			StartPosition: &storepb.Position{
				Line: 0,
			},
		}
	}
	// Get column type
	var columnType string
	if columnDef.Typename() != nil {
		columnType = extractTypeNameFromContext(columnDef.Typename())
	}
	// Create column metadata
	col := &storepb.ColumnMetadata{
		Name:     columnName,
		Position: int32(len(table.GetProto().GetColumns()) + 1),
		Default:  "",
		Nullable: true,
		Type:     columnType,
	}
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
				col.Nullable = false
			}
			// Handle DEFAULT
			if elem.DEFAULT() != nil {
				// Extract default expression from B_expr
				if elem.B_expr() != nil {
					defaultValue := elem.B_expr().GetText()
					col.Default = defaultValue
				}
			}
			// Handle PRIMARY KEY
			if elem.PRIMARY() != nil && elem.KEY() != nil {
				constraintName := ""
				if qual.Name() != nil {
					constraintName = pgparser.NormalizePostgreSQLName(qual.Name())
				}
				if constraintName == "" {
					constraintName = pgGeneratePrimaryKeyName(schema, table.GetProto().Name)
				}
				// Set column as NOT NULL
				col.Nullable = false
				// Create primary key index
				index := &storepb.IndexMetadata{
					Name:         constraintName,
					Expressions:  []string{columnName},
					Type:         "btree",
					Unique:       true,
					Primary:      true,
					IsConstraint: true,
				}
				if err := table.CreateIndex(index); err != nil {
					return &storepb.Advice{
						Status:        storepb.Advice_ERROR,
						Code:          code.PrimaryKeyExists.Int32(),
						Title:         err.Error(),
						Content:       err.Error(),
						StartPosition: &storepb.Position{Line: 0},
					}
				}
			}
			// Handle UNIQUE
			if elem.UNIQUE() != nil && (elem.PRIMARY() == nil || elem.KEY() == nil) {
				constraintName := ""
				if qual.Name() != nil {
					constraintName = pgparser.NormalizePostgreSQLName(qual.Name())
				}
				// Generate index name if not specified
				if constraintName == "" {
					constraintName = generateIndexName(table.GetProto().Name, []string{columnName}, true)
				}
				index := &storepb.IndexMetadata{
					Name:         constraintName,
					Expressions:  []string{columnName},
					Type:         "btree",
					Unique:       true,
					Primary:      false,
					IsConstraint: true,
				}
				if err := table.CreateIndex(index); err != nil {
					return &storepb.Advice{
						Status:        storepb.Advice_ERROR,
						Code:          code.IndexExists.Int32(),
						Title:         err.Error(),
						Content:       err.Error(),
						StartPosition: &storepb.Position{Line: 0},
					}
				}
			}
		}
	}
	// Create the column
	if err := table.CreateColumn(col, nil /* columnCatalog */); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}
	return nil
}

// createTableConstraint creates a table-level constraint.
func createTableConstraint(schema *model.SchemaMetadata, table *model.TableMetadata, constraint parser.ITableconstraintContext, line int) *storepb.Advice {
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
			column := table.GetColumn(colName)
			if column != nil {
				column.GetProto().Nullable = false
			} else {
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.ColumnNotExists.Int32(),
					Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", colName, table.GetProto().Name),
					Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", colName, table.GetProto().Name),
					StartPosition: &storepb.Position{Line: int32(line)},
				}
			}
		}
		// Generate PK name if not provided
		pkName := constraintName
		if pkName == "" {
			pkName = pgGeneratePrimaryKeyName(schema, table.GetProto().Name)
		}
		// Check if primary key already exists
		if table.GetPrimaryKey() != nil {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.PrimaryKeyExists.Int32(),
				Title:         fmt.Sprintf("Primary key already exists in table %q", table.GetProto().Name),
				Content:       fmt.Sprintf("Primary key already exists in table %q", table.GetProto().Name),
				StartPosition: &storepb.Position{Line: int32(line)},
			}
		}
		// Create primary key index
		index := &storepb.IndexMetadata{
			Name:         pkName,
			Expressions:  columnList,
			Type:         "btree",
			Unique:       true,
			Primary:      true,
			IsConstraint: true,
		}
		if err := table.CreateIndex(index); err != nil {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.PrimaryKeyExists.Int32(),
				Title:         err.Error(),
				Content:       err.Error(),
				StartPosition: &storepb.Position{Line: int32(line)},
			}
		}
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
			indexName = generateIndexName(table.GetProto().Name, columnList, true)
		}
		// Check if index already exists
		if table.GetIndex(indexName) != nil {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.IndexExists.Int32(),
				Title:         fmt.Sprintf("Index %q already exists in table %q", indexName, table.GetProto().Name),
				Content:       fmt.Sprintf("Index %q already exists in table %q", indexName, table.GetProto().Name),
				StartPosition: &storepb.Position{Line: int32(line)},
			}
		}
		// Validate columns exist
		for _, colName := range columnList {
			if table.GetColumn(colName) == nil {
				return &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.ColumnNotExists.Int32(),
					Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", colName, table.GetProto().Name),
					Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", colName, table.GetProto().Name),
					StartPosition: &storepb.Position{Line: int32(line)},
				}
			}
		}
		// Create unique index (from UNIQUE constraint)
		index := &storepb.IndexMetadata{
			Name:         indexName,
			Expressions:  columnList,
			Type:         "btree",
			Unique:       true,
			Primary:      false,
			IsConstraint: true,
		}
		if err := table.CreateIndex(index); err != nil {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.IndexExists.Int32(),
				Title:         err.Error(),
				Content:       err.Error(),
				StartPosition: &storepb.Position{Line: int32(line)},
			}
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
func (l *pgCatalogListener) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.advice != nil {
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
	schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
	if err != nil {
		l.advice = err
		return
	}
	table := schema.GetTable(tableName)
	if table == nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableNotExists.Int32(),
			Title:         fmt.Sprintf("Table `%s` does not exist", tableName),
			Content:       fmt.Sprintf("Table `%s` does not exist", tableName),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
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
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexEmptyKeys.Int32(),
			Title:         fmt.Sprintf("Index %q in table %q has empty key", indexName, tableName),
			Content:       fmt.Sprintf("Index %q in table %q has empty key", indexName, tableName),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// Generate index name if not provided
	isUnique := ctx.Opt_unique() != nil && ctx.Opt_unique().UNIQUE() != nil
	wasAutoGenerated := indexName == ""
	if indexName == "" {
		indexName = generateIndexName(tableName, columnList, isUnique)
	}
	// Check if index name already exists
	if table.GetIndex(indexName) != nil {
		if ifNotExists {
			return
		}
		// If name was auto-generated, try with numeric suffix
		if wasAutoGenerated {
			indexName = generateUniqueIndexName(schema, tableName, columnList, isUnique)
		} else {
			l.advice = &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.RelationExists.Int32(),
				Title:         fmt.Sprintf("Relation %q already exists in schema %q", indexName, schema.GetProto().Name),
				Content:       fmt.Sprintf("Relation %q already exists in schema %q", indexName, schema.GetProto().Name),
				StartPosition: &storepb.Position{Line: int32(l.currentLine)},
			}
			return
		}
	}
	// Check that all columns exist (skip expressions)
	for _, colName := range columnList {
		if colName != "expr" {
			if table.GetColumn(colName) == nil {
				l.advice = &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.ColumnNotExists.Int32(),
					Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", colName, tableName),
					Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", colName, tableName),
					StartPosition: &storepb.Position{Line: int32(l.currentLine)},
				}
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
	// Create index
	index := &storepb.IndexMetadata{
		Name:        indexName,
		Expressions: columnList,
		Type:        indexType,
		Unique:      isUnique,
		Primary:     false,
	}
	if err := table.CreateIndex(index); err != nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
}

// ========================================
// CREATE SCHEMA handling
// ========================================
// TODO: EnterCreateschemastatement - Need to find correct ANTLR context name
//
//	func (l *pgCatalogListener) EnterCreateschemastatement(ctx *parser.CreateschemaContext) {
//		if !isTopLevel(ctx.GetParent()) || l.advice != nil {
//			return
//		}
//
//		if !l.checkDatabaseNotDeleted() {
//			return
//		}
//
//		l.currentLine = ctx.GetStart().GetLine()
//
//		// TODO: Implement CREATE SCHEMA logic
//		// Similar to pgCreateSchema() in walk_through_for_pg.go
//	}
//
// ========================================
// ALTER TABLE handling
// ========================================
// EnterAltertablestmt handles ALTER TABLE statements.
func (l *pgCatalogListener) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.advice != nil {
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
	if databaseName != "" && l.databaseState.DatabaseName() != databaseName {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_WARNING,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         fmt.Sprintf("Database %q is not the current database %q", databaseName, l.databaseState.DatabaseName()),
			Content:       fmt.Sprintf("Database %q is not the current database %q", databaseName, l.databaseState.DatabaseName()),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// Get schema and table
	schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
	if err != nil {
		l.advice = err
		return
	}
	table := schema.GetTable(tableName)
	if table == nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableNotExists.Int32(),
			Title:         fmt.Sprintf("Table `%s` does not exist", tableName),
			Content:       fmt.Sprintf("Table `%s` does not exist", tableName),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// Process alter table commands
	if ctx.Alter_table_cmds() == nil {
		return
	}
	allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
	for _, cmd := range allCmds {
		l.processAlterTableCmd(schema, table, cmd)
		if l.advice != nil {
			return
		}
	}
}

// processAlterTableCmd handles individual ALTER TABLE commands.
func (l *pgCatalogListener) processAlterTableCmd(schema *model.SchemaMetadata, table *model.TableMetadata, cmd parser.IAlter_table_cmdContext) {
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
				typeString := extractTypeNameFromContext(cmd.Typename())
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
func (l *pgCatalogListener) alterTableDropColumn(schema *model.SchemaMetadata, table *model.TableMetadata, columnName string, ifExists bool) {
	column := table.GetColumn(columnName)
	if column == nil {
		if ifExists {
			return
		}
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnNotExists.Int32(),
			Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// Check if any views depend on this column
	dependentViews := schema.GetDependentViews(table.GetProto().Name, columnName)
	if len(dependentViews) > 0 {
		// Format view names with schema prefix to match old error format
		var viewNames []string
		for _, v := range dependentViews {
			viewNames = append(viewNames, fmt.Sprintf("%q.%q", schema.GetProto().Name, v))
		}
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnIsReferencedByView.Int32(),
			Title:         fmt.Sprintf("Cannot drop column %q in table %q.%q, it's referenced by view: %s", columnName, schema.GetProto().Name, table.GetProto().Name, strings.Join(viewNames, ", ")),
			Content:       fmt.Sprintf("Cannot drop column %q in table %q.%q, it's referenced by view: %s", columnName, schema.GetProto().Name, table.GetProto().Name, strings.Join(viewNames, ", ")),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// Drop the constraints and indexes involving the column
	var dropIndexList []string
	for _, index := range table.GetProto().Indexes {
		for _, key := range index.Expressions {
			// TODO(zp): deal with expression key.
			if key == columnName {
				dropIndexList = append(dropIndexList, index.Name)
				break // Once we find the column in this index, mark for deletion and move to next index
			}
		}
	}
	for _, indexName := range dropIndexList {
		if err := table.DropIndex(indexName); err != nil {
			l.advice = &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.IndexNotExists.Int32(),
				Title:         err.Error(),
				Content:       err.Error(),
				StartPosition: &storepb.Position{Line: int32(l.currentLine)},
			}
			return
		}
	}
	// TODO(zp): deal with other constraints.
	// TODO(zp): deal with CASCADE.
	// Validate column exists before dropping
	if table.GetColumn(columnName) == nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnNotExists.Int32(),
			Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// Delete the column without renumbering positions
	// PostgreSQL maintains stable column positions (attnum) even after dropping columns
	if err := table.DropColumnWithoutRenumbering(columnName); err != nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.Internal.Int32(),
			Title:         fmt.Sprintf("failed to drop column: %v", err),
			Content:       fmt.Sprintf("failed to drop column: %v", err),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
}

// alterTableAlterColumnType handles ALTER COLUMN TYPE command.
func (l *pgCatalogListener) alterTableAlterColumnType(schema *model.SchemaMetadata, table *model.TableMetadata, columnName string, typeString string) {
	column := table.GetColumn(columnName)
	if column == nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnNotExists.Int32(),
			Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// Check if any views depend on this column
	dependentViews := schema.GetDependentViews(table.GetProto().Name, columnName)
	if len(dependentViews) > 0 {
		// Format view names with schema prefix to match old error format
		var viewNames []string
		for _, v := range dependentViews {
			viewNames = append(viewNames, fmt.Sprintf("%q.%q", schema.GetProto().Name, v))
		}
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnIsReferencedByView.Int32(),
			Title:         fmt.Sprintf("Cannot alter type of column %q in table %q.%q, it's referenced by view: %s", columnName, schema.GetProto().Name, table.GetProto().Name, strings.Join(viewNames, ", ")),
			Content:       fmt.Sprintf("Cannot alter type of column %q in table %q.%q, it's referenced by view: %s", columnName, schema.GetProto().Name, table.GetProto().Name, strings.Join(viewNames, ", ")),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// Update column type
	column.GetProto().Type = typeString
}

// alterTableAddColumn handles ADD COLUMN command.
func (l *pgCatalogListener) alterTableAddColumn(schema *model.SchemaMetadata, table *model.TableMetadata, columndef parser.IColumnDefContext, ifNotExists bool) {
	if columndef == nil {
		return
	}
	columnName := pgparser.NormalizePostgreSQLColid(columndef.Colid())
	// Check if column already exists
	if table.GetColumn(columnName) != nil {
		if ifNotExists {
			return
		}
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnExists.Int32(),
			Title:         fmt.Sprintf("The column %q already exists in table %q", columnName, table.GetProto().Name),
			Content:       fmt.Sprintf("The column %q already exists in table %q", columnName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// Extract column type
	var typeString string
	if columndef.Typename() != nil {
		typeString = extractTypeNameFromContext(columndef.Typename())
	}
	// Create column metadata
	col := &storepb.ColumnMetadata{
		Name:     columnName,
		Position: int32(len(table.GetProto().GetColumns()) + 1),
		Nullable: true,
		Type:     typeString,
		Default:  "",
	}
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
				col.Nullable = false
			}
			// Handle DEFAULT
			if elem.DEFAULT() != nil {
				if elem.B_expr() != nil {
					defaultValue := elem.B_expr().GetText()
					col.Default = defaultValue
				}
			}
			// Handle UNIQUE - creates an index
			if elem.UNIQUE() != nil && (elem.PRIMARY() == nil || elem.KEY() == nil) {
				var constraintName string
				if qual.Name() != nil {
					constraintName = pgparser.NormalizePostgreSQLName(qual.Name())
				}
				if constraintName == "" {
					constraintName = generateIndexName(table.GetProto().Name, []string{columnName}, true)
				}
				// Check for collision
				if table.GetIndex(constraintName) != nil {
					constraintName = generateUniqueIndexName(schema, table.GetProto().Name, []string{columnName}, true)
				}
				// Create index
				index := &storepb.IndexMetadata{
					Name:        constraintName,
					Expressions: []string{columnName},
					Type:        "btree",
					Unique:      true,
					Primary:     false,
				}
				if err := table.CreateIndex(index); err != nil {
					l.advice = &storepb.Advice{
						Status:        storepb.Advice_ERROR,
						Code:          code.IndexExists.Int32(),
						Title:         err.Error(),
						Content:       err.Error(),
						StartPosition: &storepb.Position{Line: int32(l.currentLine)},
					}
					return
				}
			}
			// Handle PRIMARY KEY - creates an index
			if (elem.PRIMARY() != nil && elem.KEY() != nil) || (elem.UNIQUE() != nil && elem.PRIMARY() != nil) {
				var constraintName string
				if qual.Name() != nil {
					constraintName = pgparser.NormalizePostgreSQLName(qual.Name())
				}
				if constraintName == "" {
					constraintName = pgGeneratePrimaryKeyName(schema, table.GetProto().Name)
				}
				// Check for collision
				if table.GetIndex(constraintName) != nil {
					l.advice = &storepb.Advice{
						Status:        storepb.Advice_ERROR,
						Code:          code.RelationExists.Int32(),
						Title:         fmt.Sprintf("Relation %q already exists in schema %q", constraintName, schema.GetProto().Name),
						Content:       fmt.Sprintf("Relation %q already exists in schema %q", constraintName, schema.GetProto().Name),
						StartPosition: &storepb.Position{Line: int32(l.currentLine)},
					}
					return
				}
				col.Nullable = false
				// Create primary key index
				index := &storepb.IndexMetadata{
					Name:         constraintName,
					Expressions:  []string{columnName},
					Type:         "btree",
					Unique:       true,
					Primary:      true,
					IsConstraint: true,
				}
				if err := table.CreateIndex(index); err != nil {
					l.advice = &storepb.Advice{
						Status:        storepb.Advice_ERROR,
						Code:          code.PrimaryKeyExists.Int32(),
						Title:         err.Error(),
						Content:       err.Error(),
						StartPosition: &storepb.Position{Line: int32(l.currentLine)},
					}
					return
				}
			}
		}
	}
	// Create the column
	if err := table.CreateColumn(col, nil /* columnCatalog */); err != nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
}

// alterTableAddConstraint handles ADD CONSTRAINT command.
func (l *pgCatalogListener) alterTableAddConstraint(schema *model.SchemaMetadata, table *model.TableMetadata, constraint parser.ITableconstraintContext) {
	if constraint == nil {
		return
	}
	// Reuse the constraint creation logic from CREATE TABLE
	err := createTableConstraint(schema, table, constraint, l.currentLine)
	if err != nil {
		l.advice = err
	}
}

// alterTableDropConstraint handles DROP CONSTRAINT command.
func (l *pgCatalogListener) alterTableDropConstraint(_ *model.SchemaMetadata, table *model.TableMetadata, constraintName string, ifExists bool) {
	// Check if constraint exists as an index
	if table.GetIndex(constraintName) != nil {
		if err := table.DropIndex(constraintName); err != nil {
			l.advice = &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.IndexNotExists.Int32(),
				Title:         err.Error(),
				Content:       err.Error(),
				StartPosition: &storepb.Position{Line: int32(l.currentLine)},
			}
		}
		return
	}
	if !ifExists {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ConstraintNotExists.Int32(),
			Title:         fmt.Sprintf("Constraint %q for table %q does not exist", constraintName, table.GetProto().Name),
			Content:       fmt.Sprintf("Constraint %q for table %q does not exist", constraintName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
}

// alterTableSetDefault handles ALTER COLUMN SET DEFAULT command.
func (l *pgCatalogListener) alterTableSetDefault(table *model.TableMetadata, columnName string, defaultValue string) {
	column := table.GetColumn(columnName)
	if column == nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnNotExists.Int32(),
			Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	column.GetProto().Default = defaultValue
}

// alterTableDropDefault handles ALTER COLUMN DROP DEFAULT command.
func (l *pgCatalogListener) alterTableDropDefault(table *model.TableMetadata, columnName string) {
	column := table.GetColumn(columnName)
	if column == nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnNotExists.Int32(),
			Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	column.GetProto().Default = ""
}

// alterTableSetNotNull handles ALTER COLUMN SET NOT NULL command.
func (l *pgCatalogListener) alterTableSetNotNull(table *model.TableMetadata, columnName string) {
	column := table.GetColumn(columnName)
	if column == nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnNotExists.Int32(),
			Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	column.GetProto().Nullable = false
}

// renameTable handles RENAME TO for tables.
func (*pgCatalogListener) pgRenameTable(schema *model.SchemaMetadata, table *model.TableMetadata, newName string, line int) *storepb.Advice {
	// Check if new name already exists
	if schema.GetTable(newName) != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.RelationExists.Int32(),
			Title:         fmt.Sprintf("Relation %q already exists in schema %q", newName, schema.GetProto().Name),
			Content:       fmt.Sprintf("Relation %q already exists in schema %q", newName, schema.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(line)},
		}
	}
	// Use the RenameTable method from schema
	oldName := table.GetProto().Name
	if err := schema.RenameTable(oldName, newName); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableNotExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: int32(line)},
		}
	}
	return nil
}

// renameColumn handles RENAME COLUMN.
func (*pgCatalogListener) renameColumn(table *model.TableMetadata, oldName string, newName string, line int) *storepb.Advice {
	if oldName == newName {
		return nil
	}
	// Validate old column exists
	if table.GetColumn(oldName) == nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnNotExists.Int32(),
			Title:         fmt.Sprintf("Column `%s` does not exist in table `%s`", oldName, table.GetProto().Name),
			Content:       fmt.Sprintf("Column `%s` does not exist in table `%s`", oldName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(line)},
		}
	}
	// Validate new column doesn't already exist
	if table.GetColumn(newName) != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ColumnExists.Int32(),
			Title:         fmt.Sprintf("Column `%s` already exists in table `%s`", newName, table.GetProto().Name),
			Content:       fmt.Sprintf("Column `%s` already exists in table `%s`", newName, table.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(line)},
		}
	}
	// Use the RenameColumn method
	if err := table.RenameColumn(oldName, newName); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.Internal.Int32(),
			Title:         fmt.Sprintf("failed to rename column: %v", err),
			Content:       fmt.Sprintf("failed to rename column: %v", err),
			StartPosition: &storepb.Position{Line: int32(line)},
		}
	}
	// Rename column in all indexes that reference it
	for _, index := range table.GetProto().Indexes {
		for i, key := range index.Expressions {
			if key == oldName {
				index.Expressions[i] = newName
			}
		}
	}
	return nil
}

// renameConstraint handles RENAME CONSTRAINT.
func (*pgCatalogListener) renameConstraint(schema *model.SchemaMetadata, table *model.TableMetadata, oldName string, newName string, line int) *storepb.Advice {
	index := table.GetIndex(oldName)
	if index == nil {
		// We haven't dealt with foreign and check constraints, so skip if not exists
		return nil
	}
	// Check if new name already exists
	if table.GetIndex(newName) != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.RelationExists.Int32(),
			Title:         fmt.Sprintf("Relation %q already exists in schema %q", newName, schema.GetProto().Name),
			Content:       fmt.Sprintf("Relation %q already exists in schema %q", newName, schema.GetProto().Name),
			StartPosition: &storepb.Position{Line: int32(line)},
		}
	}
	// Use the RenameIndex method
	if err := table.RenameIndex(oldName, newName); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexNotExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: int32(line)},
		}
	}
	return nil
}

// ========================================
// DROP statements handling
// ========================================
// EnterDropstmt handles DROP TABLE/VIEW/INDEX statements.
func (l *pgCatalogListener) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.advice != nil {
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
						l.advice = err
						return
					}
				}
			}
		} else if objType.VIEW() != nil {
			// DROP VIEW
			if ctx.Any_name_list() != nil {
				for _, anyName := range ctx.Any_name_list().AllAny_name() {
					if err := l.dropView(anyName, ifExists); err != nil {
						l.advice = err
						return
					}
				}
			}
		} else if objType.INDEX() != nil {
			// DROP INDEX
			if ctx.Any_name_list() != nil {
				for _, anyName := range ctx.Any_name_list().AllAny_name() {
					if err := l.dropIndex(anyName, ifExists); err != nil {
						l.advice = err
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
					l.advice = err
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
	if !isTopLevel(ctx.GetParent()) || l.advice != nil {
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
	if isCurrentDatabase(l.databaseState, databaseName) {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_WARNING,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         fmt.Sprintf("Cannot drop the currently open database %q", databaseName),
			Content:       fmt.Sprintf("Cannot drop the currently open database %q", databaseName),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	// DROP DATABASE for other databases is out of scope for single-database walk-through
	l.advice = &storepb.Advice{
		Status:        storepb.Advice_WARNING,
		Code:          code.NotCurrentDatabase.Int32(),
		Title:         fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, l.databaseState.DatabaseName()),
		Content:       fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, l.databaseState.DatabaseName()),
		StartPosition: &storepb.Position{Line: int32(l.currentLine)},
	}
}
func (l *pgCatalogListener) dropTable(anyName parser.IAny_nameContext, ifExists bool) *storepb.Advice {
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
	schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}
	table := schema.GetTable(tableName)
	if table == nil {
		if ifExists {
			return nil
		}
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableNotExists.Int32(),
			Title:         fmt.Sprintf("Table `%s` does not exist", tableName),
			Content:       fmt.Sprintf("Table `%s` does not exist", tableName),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
	// Check if any views depend on this table
	dependentViews := schema.GetDependentViews(tableName, "")
	if len(dependentViews) > 0 {
		// Format view names with schema prefix to match old error format
		var viewNames []string
		for _, v := range dependentViews {
			viewNames = append(viewNames, fmt.Sprintf("%q.%q", schema.GetProto().Name, v))
		}
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableIsReferencedByView.Int32(),
			Title:         fmt.Sprintf("Cannot drop table %q.%q, it's referenced by view: %s", schema.GetProto().Name, tableName, strings.Join(viewNames, ", ")),
			Content:       fmt.Sprintf("Cannot drop table %q.%q, it's referenced by view: %s", schema.GetProto().Name, tableName, strings.Join(viewNames, ", ")),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
	// Drop the table using the schema method
	if err := schema.DropTable(tableName); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableNotExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
	return nil
}
func (l *pgCatalogListener) dropView(anyName parser.IAny_nameContext, ifExists bool) *storepb.Advice {
	parts := pgparser.NormalizePostgreSQLAnyName(anyName)
	if len(parts) == 0 {
		return nil
	}
	var schemaName, viewName string
	switch len(parts) {
	case 1:
		viewName = parts[0]
	case 2:
		schemaName = parts[0]
		viewName = parts[1]
	default:
		return nil
	}
	schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
	if err != nil {
		return err
	}
	// Try to drop the view
	if dropErr := schema.DropView(viewName); dropErr != nil {
		if !ifExists {
			return &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.ViewNotExists.Int32(),
				Title:         dropErr.Error(),
				Content:       dropErr.Error(),
				StartPosition: &storepb.Position{Line: int32(l.currentLine)},
			}
		}
	}
	return nil
}
func (l *pgCatalogListener) dropIndex(anyName parser.IAny_nameContext, ifExists bool) *storepb.Advice {
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
	schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}
	table, index, err := getIndexFromSchema(schema, indexName)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}
	// Drop the index from the table
	if err := table.DropIndex(index.GetProto().Name); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexNotExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
	return nil
}
func (l *pgCatalogListener) dropSchema(schemaNameCtx parser.INameContext, ifExists bool) *storepb.Advice {
	schemaName := pgparser.NormalizePostgreSQLName(schemaNameCtx)
	schema := l.databaseState.GetSchemaMetadata(schemaName)
	if schema == nil {
		if ifExists {
			return nil
		}
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.SchemaNotExists.Int32(),
			Title:         fmt.Sprintf("Schema %q does not exist", schemaName),
			Content:       fmt.Sprintf("Schema %q does not exist", schemaName),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
	// Drop the schema (this will also drop all objects in the schema)
	// Note: We don't check for objects in the schema to match the old behavior
	// In real PostgreSQL, this would require CASCADE if the schema is not empty
	if err := l.databaseState.DropSchema(schemaName); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.SchemaNotExists.Int32(),
			Title:         err.Error(),
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
	return nil
}

// TODO: EnterDropindexstmt - Need to find correct ANTLR context name
//
//	func (l *pgCatalogListener) EnterDropindexstmt(ctx *parser.DropIndexContext) {
//		if !isTopLevel(ctx.GetParent()) || l.advice != nil {
//			return
//		}
//
//		if !l.checkDatabaseNotDeleted() {
//			return
//		}
//
//		l.currentLine = ctx.GetStart().GetLine()
//
//		// TODO: Implement DROP INDEX logic
//		// Similar to pgDropIndexList() in walk_through_for_pg.go
//	}
//
// TODO: EnterDropschemastatement - Need to find correct ANTLR context name
//
//	func (l *pgCatalogListener) EnterDropschemastatement(ctx *parser.DropschemaContext) {
//		if !isTopLevel(ctx.GetParent()) || l.advice != nil {
//			return
//		}
//
//		if !l.checkDatabaseNotDeleted() {
//			return
//		}
//
//		l.currentLine = ctx.GetStart().GetLine()
//
//		// TODO: Implement DROP SCHEMA logic
//		// Similar to pgDropSchema() in walk_through_for_pg.go
//	}
//
// ========================================
// RENAME statements handling
// ========================================
// EnterRenamestmt handles RENAME INDEX/CONSTRAINT/TABLE/COLUMN statements.
func (l *pgCatalogListener) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.advice != nil {
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
			schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
			if err != nil {
				l.advice = err
				return
			}
			// Find the index across all tables
			foundTable, _, err := getIndexFromSchema(schema, indexName)
			if err != nil {
				// Index not found, silently ignore (PostgreSQL behavior)
				return
			}
			if err := l.renameConstraint(schema, foundTable, indexName, newName, l.currentLine); err != nil {
				l.advice = err
			}
		}
		return
	}
	// Check if this is VIEW rename (ALTER VIEW ... RENAME TO ...)
	if ctx.Qualified_name() != nil && ctx.AllName() != nil && len(ctx.AllName()) > 0 {
		// Could be ALTER VIEW view_name RENAME TO new_name
		// Try to see if the qualified name refers to a view
		viewName := extractTableName(ctx.Qualified_name())
		schemaName := extractSchemaName(ctx.Qualified_name())
		newName := pgparser.NormalizePostgreSQLName(ctx.AllName()[0])
		schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
		if err != nil {
			l.advice = err
			return
		}
		// Check if it's a view
		if schema.GetView(viewName) != nil {
			if err := schema.RenameView(viewName, newName); err != nil {
				l.advice = &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.ViewNotExists.Int32(),
					Title:         err.Error(),
					Content:       err.Error(),
					StartPosition: &storepb.Position{Line: int32(l.currentLine)},
				}
			}
			return
		}
	}
	// Extract relation (table) if present
	var tableName, schemaName string
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(ctx.Relation_expr().Qualified_name())
		schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
	}
	schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
	if err != nil {
		l.advice = err
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
			table := schema.GetTable(tableName)
			if table == nil {
				l.advice = &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableNotExists.Int32(),
					Title:         fmt.Sprintf("Table `%s` does not exist", tableName),
					Content:       fmt.Sprintf("Table `%s` does not exist", tableName),
					StartPosition: &storepb.Position{Line: int32(l.currentLine)},
				}
				return
			}
			if err := l.renameColumn(table, oldName, newName, l.currentLine); err != nil {
				l.advice = err
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
			table := schema.GetTable(tableName)
			if table == nil {
				l.advice = &storepb.Advice{
					Status:        storepb.Advice_ERROR,
					Code:          code.TableNotExists.Int32(),
					Title:         fmt.Sprintf("Table `%s` does not exist", tableName),
					Content:       fmt.Sprintf("Table `%s` does not exist", tableName),
					StartPosition: &storepb.Position{Line: int32(l.currentLine)},
				}
				return
			}
			if err := l.renameConstraint(schema, table, oldName, newName, l.currentLine); err != nil {
				l.advice = err
			}
		}
		return
	}
	// Otherwise it's table rename: ALTER TABLE oldname RENAME TO newname
	if tableName != "" && ctx.AllName() != nil && len(ctx.AllName()) > 0 {
		newName := pgparser.NormalizePostgreSQLName(ctx.AllName()[0])
		table := schema.GetTable(tableName)
		if table == nil {
			l.advice = &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          code.TableNotExists.Int32(),
				Title:         fmt.Sprintf("Table `%s` does not exist", tableName),
				Content:       fmt.Sprintf("Table `%s` does not exist", tableName),
				StartPosition: &storepb.Position{Line: int32(l.currentLine)},
			}
			return
		}
		if err := l.pgRenameTable(schema, table, newName, l.currentLine); err != nil {
			l.advice = err
		}
		return
	}
}

// ========================================
// CREATE VIEW handling
// ========================================
// EnterViewstmt handles CREATE VIEW statements.
func (l *pgCatalogListener) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) || l.advice != nil {
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
	if databaseName != "" && l.databaseState.DatabaseName() != databaseName {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_WARNING,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         fmt.Sprintf("Database %q is not the current database %q", databaseName, l.databaseState.DatabaseName()),
			Content:       fmt.Sprintf("Database %q is not the current database %q", databaseName, l.databaseState.DatabaseName()),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
		return
	}
	schema, err := getOrCreatePublicSchema(l.databaseState, schemaName, l.currentLine)
	if err != nil {
		l.advice = err
		return
	}
	// Check if view already exists - silently ignore duplicates
	// This matches the legacy behavior in the old DatabaseState implementation
	if schema.GetView(viewName) != nil {
		return
	}
	// Get the view definition (the SELECT statement)
	// We don't parse the dependencies yet - that would require full SQL analysis
	// For now, create the view with empty dependency list
	definition := ""
	if ctx.Selectstmt() != nil {
		definition = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Selectstmt())
	}
	// Create the view
	_, createErr := schema.CreateView(viewName, definition, nil)
	if createErr != nil {
		l.advice = &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.ViewExists.Int32(),
			Title:         createErr.Error(),
			Content:       createErr.Error(),
			StartPosition: &storepb.Position{Line: int32(l.currentLine)},
		}
	}
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

// extractTypeNameFromContext extracts the type name from a Typename context.
// Simply uses GetText() to get the full type representation.
// PostgreSQL normalizes some type names (e.g., int -> integer),
// which will be handled by the parser's type normalization.
func extractTypeNameFromContext(typename parser.ITypenameContext) string {
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
func generateUniqueIndexName(schema *model.SchemaMetadata, tableName string, columnList []string, isUnique bool) string {
	baseName := generateIndexName(tableName, columnList, isUnique)
	// Try with numeric suffixes starting from 1
	for i := 1; i < 1000; i++ {
		candidateName := fmt.Sprintf("%s%d", baseName, i)
		// Check if index exists in the schema
		if schema.GetIndex(candidateName) == nil {
			return candidateName
		}
	}
	// Fallback (should never reach here)
	return fmt.Sprintf("%s_collision", baseName)
}

// pgGeneratePrimaryKeyName generates a primary key name following PostgreSQL conventions.
// Uses the pattern "{tableName}_pkey" and adds numeric suffixes if there are collisions.
func pgGeneratePrimaryKeyName(schema *model.SchemaMetadata, tableName string) string {
	pkName := fmt.Sprintf("%s_pkey", tableName)
	// For PostgreSQL, check if the index name already exists in the schema
	if schema.GetIndex(pkName) == nil {
		return pkName
	}
	suffix := 1
	for {
		candidateName := fmt.Sprintf("%s%d", pkName, suffix)
		if schema.GetIndex(candidateName) == nil {
			return candidateName
		}
		suffix++
	}
}

// getOrCreatePublicSchema gets a schema by name, or creates the "public" schema if it doesn't exist.
// For PostgreSQL, the "public" schema is auto-created, but other schemas must exist.
func getOrCreatePublicSchema(d *model.DatabaseMetadata, schemaName string, line int) (*model.SchemaMetadata, *storepb.Advice) {
	if schemaName == "" {
		schemaName = PublicSchemaName
	}
	schema := d.GetSchemaMetadata(schemaName)
	if schema != nil {
		return schema, nil
	}
	// Only auto-create the "public" schema
	if schemaName != PublicSchemaName {
		return nil, &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.SchemaNotExists.Int32(),
			Title:         fmt.Sprintf("The schema %q doesn't exist", schemaName),
			Content:       fmt.Sprintf("The schema %q doesn't exist", schemaName),
			StartPosition: &storepb.Position{Line: int32(line)},
		}
	}
	return d.CreateSchema(PublicSchemaName), nil
}

// getIndexFromSchema finds an index by name within a schema and returns the table and index.
// For PostgreSQL, index names are unique within a schema (not just within a table).
func getIndexFromSchema(s *model.SchemaMetadata, indexName string) (*model.TableMetadata, *model.IndexMetadata, *storepb.Advice) {
	// For PostgreSQL, index names are unique within a schema
	index := s.GetIndex(indexName)
	if index == nil {
		return nil, nil, &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.IndexNotExists.Int32(),
			Title:         fmt.Sprintf("Index %q does not exist in schema %q", indexName, s.GetProto().Name),
			Content:       fmt.Sprintf("Index %q does not exist in schema %q", indexName, s.GetProto().Name),
			StartPosition: &storepb.Position{Line: 0},
		}
	}
	// Find the table that owns this index
	table := s.GetTable(index.GetTableProto().Name)
	if table == nil {
		return nil, nil, &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.TableNotExists.Int32(),
			Title:         fmt.Sprintf("Table %q does not exist in schema %q", index.GetTableProto().Name, s.GetProto().Name),
			Content:       fmt.Sprintf("Table %q does not exist in schema %q", index.GetTableProto().Name, s.GetProto().Name),
			StartPosition: &storepb.Position{Line: 0},
		}
	}
	return table, index, nil
}

// isCurrentDatabase returns true if the given database is the current database.
func isCurrentDatabase(d *model.DatabaseMetadata, database string) bool {
	return d.DatabaseName() == database
}
