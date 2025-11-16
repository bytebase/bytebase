package pg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// CheckSDLIntegrity performs comprehensive integrity checks across SDL files.
// This function always handles multiple files to properly validate cross-file references
// (foreign keys, views) and detect duplicate definitions across files.
//
// Parameters:
//   - files: map[filePath]sqlContent - All SDL files to check together.
//     For single-file checking, use a map with one entry.
//
// Returns:
//   - map[filePath][]*storepb.Advice - Per-file advice list
//   - error - Parse or system errors
func CheckSDLIntegrity(files map[string]string) (map[string][]*storepb.Advice, error) {
	if len(files) == 0 {
		return make(map[string][]*storepb.Advice), nil
	}

	// Parse each file and build individual symbol tables
	fileCheckers := make([]*fileSymbolTable, 0, len(files))
	for filePath, statement := range files {
		tree, err := pgparser.ParsePostgreSQL(statement)
		if err != nil {
			// Return parse error for this file
			return map[string][]*storepb.Advice{
				filePath: {{
					Status:  storepb.Advice_ERROR,
					Code:    code.StatementSyntaxError.Int32(),
					Title:   "SQL syntax error",
					Content: fmt.Sprintf("Failed to parse SQL in file '%s': %v", filePath, err),
				}},
			}, nil
		}

		checker := &sdlIntegrityChecker{
			BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
			symbolTable:                  newSymbolTable(),
		}

		// Build symbol table for this file
		antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

		fileCheckers = append(fileCheckers, &fileSymbolTable{
			filePath:   filePath,
			checker:    checker,
			adviceList: make([]*storepb.Advice, 0),
		})
	}

	// Merge all symbol tables
	merged := mergeSymbolTables(fileCheckers)

	// Detect cross-file duplicates
	crossFileDuplicates := detectCrossFileDuplicates(fileCheckers)
	for filePath, advices := range crossFileDuplicates {
		for _, fc := range fileCheckers {
			if fc.filePath == filePath {
				fc.adviceList = append(fc.adviceList, advices...)
				break
			}
		}
	}

	// Validate references with merged symbol table
	for _, fc := range fileCheckers {
		fc.checker.validateReferencesWithMergedTable(merged)
		fc.adviceList = append(fc.adviceList, fc.checker.adviceList...)
	}

	// Aggregate results per file
	results := make(map[string][]*storepb.Advice)
	for _, fc := range fileCheckers {
		results[fc.filePath] = fc.adviceList
	}

	return results, nil
}

// checkSingleStatement is a helper for backward compatibility with single-statement checks.
// It wraps the statement in a single-file map and extracts the results.
func checkSingleStatement(statement string) ([]*storepb.Advice, error) {
	const defaultFileName = "statement.sql"
	results, err := CheckSDLIntegrity(map[string]string{
		defaultFileName: statement,
	})
	if err != nil {
		return nil, err
	}
	return results[defaultFileName], nil
}

// fileSymbolTable tracks the symbol table and advices for a single file
type fileSymbolTable struct {
	filePath   string
	checker    *sdlIntegrityChecker
	adviceList []*storepb.Advice
}

// mergedSymbolTable provides a global view of all objects across all files
type mergedSymbolTable struct {
	allTables                map[string]*tableDef
	allIndexes               map[string]*indexDef
	allViews                 map[string]*viewDef
	allSchemaConstraintNames map[string]map[string]*constraintDef // schema -> constraint_name (for PK/UK)
	allTableConstraintNames  map[string]map[string]*constraintDef // schema.table -> constraint_name (for CHECK/FK)
	objectSources            map[string]string                    // qualifiedName -> filePath
}

// mergeSymbolTables merges symbol tables from all files into a unified view
func mergeSymbolTables(fileCheckers []*fileSymbolTable) *mergedSymbolTable {
	merged := &mergedSymbolTable{
		allTables:                make(map[string]*tableDef),
		allIndexes:               make(map[string]*indexDef),
		allViews:                 make(map[string]*viewDef),
		allSchemaConstraintNames: make(map[string]map[string]*constraintDef),
		allTableConstraintNames:  make(map[string]map[string]*constraintDef),
		objectSources:            make(map[string]string),
	}

	for _, fc := range fileCheckers {
		// Merge tables
		for key, table := range fc.checker.symbolTable.tables {
			if _, exists := merged.allTables[key]; !exists {
				merged.allTables[key] = table
				merged.objectSources[key] = fc.filePath
			}
		}

		// Merge indexes
		for key, index := range fc.checker.symbolTable.indexes {
			if _, exists := merged.allIndexes[key]; !exists {
				merged.allIndexes[key] = index
				merged.objectSources["index:"+key] = fc.filePath
			}
		}

		// Merge views
		for key, view := range fc.checker.symbolTable.views {
			if _, exists := merged.allViews[key]; !exists {
				merged.allViews[key] = view
				merged.objectSources["view:"+key] = fc.filePath
			}
		}

		// Merge schema-level constraint names (PRIMARY KEY and UNIQUE)
		for schema, constraints := range fc.checker.symbolTable.schemaConstraintNames {
			if merged.allSchemaConstraintNames[schema] == nil {
				merged.allSchemaConstraintNames[schema] = make(map[string]*constraintDef)
			}
			for name, constraint := range constraints {
				if _, exists := merged.allSchemaConstraintNames[schema][name]; !exists {
					merged.allSchemaConstraintNames[schema][name] = constraint
					merged.objectSources["constraint:"+schema+"."+name] = fc.filePath
				}
			}
		}

		// Merge table-level constraint names (CHECK and FOREIGN KEY)
		for tableKey, constraints := range fc.checker.symbolTable.tableConstraintNames {
			if merged.allTableConstraintNames[tableKey] == nil {
				merged.allTableConstraintNames[tableKey] = make(map[string]*constraintDef)
			}
			for name, constraint := range constraints {
				if _, exists := merged.allTableConstraintNames[tableKey][name]; !exists {
					merged.allTableConstraintNames[tableKey][name] = constraint
					merged.objectSources["constraint:"+tableKey+"."+name] = fc.filePath
				}
			}
		}
	}

	return merged
}

// objectFirstSeen tracks where an object was first defined
type objectFirstSeen struct {
	filePath string
	line     int
	obj      any // *tableDef, *indexDef, *viewDef, or *constraintDef
}

// detectCrossFileDuplicates finds objects defined in multiple files
func detectCrossFileDuplicates(fileCheckers []*fileSymbolTable) map[string][]*storepb.Advice {
	advices := make(map[string][]*storepb.Advice)

	// Track first occurrence of each object
	seenTables := make(map[string]*objectFirstSeen)
	seenIndexes := make(map[string]*objectFirstSeen)
	seenViews := make(map[string]*objectFirstSeen)
	seenConstraints := make(map[string]map[string]*objectFirstSeen)

	for _, fc := range fileCheckers {
		// Check tables
		for key, table := range fc.checker.symbolTable.tables {
			if first, exists := seenTables[key]; exists {
				advice := &storepb.Advice{
					Status: storepb.Advice_ERROR,
					Code:   code.SDLDuplicateTableName.Int32(),
					Title:  "Duplicate table name across files",
					Content: fmt.Sprintf(
						"Table '%s.%s' is defined in multiple SDL files.\n\n"+
							"First definition: %s (line %d)\n"+
							"Duplicate definition: %s (line %d)\n\n"+
							"Each table must be defined in exactly one file in the SDL project.",
						table.schemaName, table.tableName,
						first.filePath, first.line,
						fc.filePath, table.line,
					),
					StartPosition: &storepb.Position{Line: int32(table.line)},
				}
				advices[fc.filePath] = append(advices[fc.filePath], advice)
			} else {
				seenTables[key] = &objectFirstSeen{
					filePath: fc.filePath,
					line:     table.line,
					obj:      table,
				}
			}
		}

		// Check indexes
		for key, index := range fc.checker.symbolTable.indexes {
			if first, exists := seenIndexes[key]; exists {
				advice := &storepb.Advice{
					Status: storepb.Advice_ERROR,
					Code:   code.SDLDuplicateIndexName.Int32(),
					Title:  "Duplicate index name across files",
					Content: fmt.Sprintf(
						"Index '%s.%s' is defined in multiple SDL files.\n\n"+
							"First definition: %s (line %d)\n"+
							"Duplicate definition: %s (line %d)\n\n"+
							"Each index must be defined in exactly one file in the SDL project.",
						index.schemaName, index.indexName,
						first.filePath, first.line,
						fc.filePath, index.line,
					),
					StartPosition: &storepb.Position{Line: int32(index.line)},
				}
				advices[fc.filePath] = append(advices[fc.filePath], advice)
			} else {
				seenIndexes[key] = &objectFirstSeen{
					filePath: fc.filePath,
					line:     index.line,
					obj:      index,
				}
			}
		}

		// Check views
		for key, view := range fc.checker.symbolTable.views {
			if first, exists := seenViews[key]; exists {
				advice := &storepb.Advice{
					Status: storepb.Advice_ERROR,
					Code:   code.SDLDuplicateTableName.Int32(),
					Title:  "Duplicate view name across files",
					Content: fmt.Sprintf(
						"View '%s.%s' is defined in multiple SDL files.\n\n"+
							"First definition: %s (line %d)\n"+
							"Duplicate definition: %s (line %d)\n\n"+
							"Each view must be defined in exactly one file in the SDL project.",
						view.schemaName, view.viewName,
						first.filePath, first.line,
						fc.filePath, view.line,
					),
					StartPosition: &storepb.Position{Line: int32(view.line)},
				}
				advices[fc.filePath] = append(advices[fc.filePath], advice)
			} else {
				seenViews[key] = &objectFirstSeen{
					filePath: fc.filePath,
					line:     view.line,
					obj:      view,
				}
			}
		}

		// Check schema-level constraint names (PRIMARY KEY and UNIQUE)
		for schema, constraints := range fc.checker.symbolTable.schemaConstraintNames {
			if seenConstraints[schema] == nil {
				seenConstraints[schema] = make(map[string]*objectFirstSeen)
			}

			for name, constraint := range constraints {
				if first, exists := seenConstraints[schema][name]; exists {
					advice := &storepb.Advice{
						Status: storepb.Advice_ERROR,
						Code:   code.SDLDuplicateConstraintName.Int32(),
						Title:  "Duplicate constraint name across files",
						Content: fmt.Sprintf(
							"Constraint '%s' in schema '%s' is defined in multiple SDL files.\n\n"+
								"First definition: %s on table '%s' in %s (line %d)\n"+
								"Duplicate definition: %s on table '%s' in %s (line %d)\n\n"+
								"PostgreSQL requires PRIMARY KEY and UNIQUE constraint names to be unique within a schema.\n"+
								"Each constraint must be defined in exactly one file in the SDL project.",
							name, schema,
							first.obj.(*constraintDef).constraintType, first.obj.(*constraintDef).tableName, first.filePath, first.line,
							constraint.constraintType, constraint.tableName, fc.filePath, constraint.line,
						),
						StartPosition: &storepb.Position{Line: int32(constraint.line)},
					}
					advices[fc.filePath] = append(advices[fc.filePath], advice)
				} else {
					seenConstraints[schema][name] = &objectFirstSeen{
						filePath: fc.filePath,
						line:     constraint.line,
						obj:      constraint,
					}
				}
			}
		}

		// Check table-level constraint names (CHECK and FOREIGN KEY)
		// These are already scoped to table, so duplicates across files are allowed if on different tables
		// No cross-file check needed for table-level constraints
	}

	return advices
}

// validateReferencesWithMergedTable validates references using the merged symbol table
func (c *sdlIntegrityChecker) validateReferencesWithMergedTable(merged *mergedSymbolTable) {
	// Check for multiple primary keys
	c.checkMultiplePrimaryKeys()

	// Validate foreign key references with merged table
	c.validateForeignKeysWithMergedTable(merged)

	// Validate view dependencies with merged table
	c.validateViewDependenciesWithMergedTable(merged)
}

// validateForeignKeysWithMergedTable validates FK references using merged symbol table
func (c *sdlIntegrityChecker) validateForeignKeysWithMergedTable(merged *mergedSymbolTable) {
	for _, table := range c.symbolTable.tables {
		for _, constraint := range table.constraints {
			if constraint.constraintType != "FOREIGN KEY" {
				continue
			}

			// Look up referenced table in merged symbol table
			refKey := qualifiedName(constraint.fkReferencedSchema, constraint.fkReferencedTable)
			refTable := merged.allTables[refKey]

			if refTable == nil {
				c.adviceList = append(c.adviceList, &storepb.Advice{
					Status: storepb.Advice_ERROR,
					Code:   code.SDLForeignKeyTableNotFound.Int32(),
					Title:  "Foreign key references non-existent table",
					Content: fmt.Sprintf(
						"Foreign key constraint '%s' on table '%s.%s' references table '%s.%s' which does not exist in any SDL file.\n\n"+
							"Foreign key columns: %s\n\n"+
							"Make sure the referenced table is defined in one of the SDL files.",
						c.getConstraintName(constraint),
						table.schemaName, table.tableName,
						constraint.fkReferencedSchema, constraint.fkReferencedTable,
						formatColumnList(constraint.fkColumns),
					),
					StartPosition: &storepb.Position{Line: int32(constraint.line)},
				})
				continue
			}

			// Validate columns and types
			if len(constraint.fkColumns) != len(constraint.fkReferencedColumns) {
				continue
			}

			for i, fkCol := range constraint.fkColumns {
				refCol := constraint.fkReferencedColumns[i]

				sourceCol := table.columns[fkCol]
				if sourceCol == nil {
					continue
				}

				targetCol := refTable.columns[refCol]
				if targetCol == nil {
					c.adviceList = append(c.adviceList, &storepb.Advice{
						Status: storepb.Advice_ERROR,
						Code:   code.SDLForeignKeyColumnNotFound.Int32(),
						Title:  "Foreign key references non-existent column",
						Content: fmt.Sprintf(
							"Foreign key constraint '%s' on table '%s.%s' references column '%s' in table '%s.%s', but this column does not exist.\n\n"+
								"Foreign key definition:\n"+
								"  Source column: %s.%s.%s\n"+
								"  Referenced column: %s.%s.%s (DOES NOT EXIST)\n\n"+
								"Available columns in '%s.%s': %s",
							c.getConstraintName(constraint),
							table.schemaName, table.tableName,
							refCol,
							constraint.fkReferencedSchema, constraint.fkReferencedTable,
							table.schemaName, table.tableName, fkCol,
							constraint.fkReferencedSchema, constraint.fkReferencedTable, refCol,
							constraint.fkReferencedSchema, constraint.fkReferencedTable,
							c.getColumnList(refTable),
						),
						StartPosition: &storepb.Position{Line: int32(constraint.line)},
					})
					continue
				}
			}
		}
	}
}

// validateViewDependenciesWithMergedTable validates view dependencies using merged symbol table
func (c *sdlIntegrityChecker) validateViewDependenciesWithMergedTable(merged *mergedSymbolTable) {
	for _, view := range c.symbolTable.views {
		if view.selectStmt == nil {
			continue
		}

		// Extract dependencies using the same approach as generate_migration
		dependencies := c.getViewDependenciesFromAST(view.selectStmt, view.schemaName)

		for _, dep := range dependencies {
			// Skip system schemas
			if pgparser.IsSystemSchema(dep.schemaName) {
				continue
			}
			// Skip system tables/views by name
			if pgparser.IsSystemTable(dep.tableName) || pgparser.IsSystemView(dep.tableName) {
				continue
			}

			refKey := qualifiedName(dep.schemaName, dep.tableName)
			// Check if the reference exists in either tables or views
			if merged.allTables[refKey] == nil && merged.allViews[refKey] == nil {
				objectType := "table or view"
				c.adviceList = append(c.adviceList, &storepb.Advice{
					Status: storepb.Advice_ERROR,
					Code:   code.SDLViewDependencyNotFound.Int32(),
					Title:  "View references non-existent table or view",
					Content: fmt.Sprintf(
						"View '%s.%s' (line %d) references %s '%s.%s' which does not exist in any SDL file.\n\n"+
							"Views must reference tables or views that are defined in the SDL project.\n\n"+
							"Fix: Define %s '%s.%s' in one of the SDL files, or remove the view if the object is external.",
						view.schemaName, view.viewName, view.line,
						objectType, dep.schemaName, dep.tableName,
						objectType, dep.schemaName, dep.tableName,
					),
					StartPosition: &storepb.Position{Line: int32(view.line)},
				})
			}
		}
	}
}

// formatColumnList formats a list of columns for display
func formatColumnList(columns []string) string {
	if len(columns) == 0 {
		return "<none>"
	}
	result := ""
	for i, col := range columns {
		if i > 0 {
			result += ", "
		}
		result += col
	}
	return result
}

// sdlIntegrityChecker performs multi-pass integrity checking
type sdlIntegrityChecker struct {
	*parser.BasePostgreSQLParserListener
	symbolTable *symbolTable
	adviceList  []*storepb.Advice
}

// symbolTable stores all defined objects for cross-referencing
type symbolTable struct {
	// schema.table -> TableDef
	tables map[string]*tableDef
	// schema.index -> IndexDef
	indexes map[string]*indexDef
	// schema.view -> ViewDef
	views map[string]*viewDef
	// Schema-level constraint names (for PRIMARY KEY and UNIQUE constraints)
	// schema -> constraint_name -> constraintDef
	schemaConstraintNames map[string]map[string]*constraintDef
	// Table-level constraint names (for CHECK and FOREIGN KEY constraints)
	// schema.table -> constraint_name -> constraintDef
	tableConstraintNames map[string]map[string]*constraintDef
}

type tableDef struct {
	schemaName string
	tableName  string
	columns    map[string]*columnDef // column_name -> ColumnDef
	// Track primary keys to detect duplicates
	primaryKeys []*constraintDef
	// Track all constraints for validation
	constraints []*constraintDef
	line        int
}

type columnDef struct {
	name     string
	dataType *dataType
	notNull  bool
	position int
	line     int
}

type dataType struct {
	// Base type name: INTEGER, TEXT, VARCHAR, NUMERIC, TIMESTAMP, etc.
	baseType string
	// For VARCHAR(n), CHAR(n)
	length int
	// For NUMERIC(p,s)
	precision int
	scale     int
}

type indexDef struct {
	schemaName string
	indexName  string
	tableName  string
	line       int
}

type viewDef struct {
	schemaName string
	viewName   string
	// Store the SELECT statement context for dependency analysis
	selectStmt parser.ISelectstmtContext
	line       int
}

type constraintDef struct {
	name           string
	constraintType string // "PRIMARY KEY", "UNIQUE", "CHECK", "FOREIGN KEY"
	schemaName     string
	tableName      string
	line           int
	// For FK constraints
	fkReferencedSchema  string
	fkReferencedTable   string
	fkColumns           []string
	fkReferencedColumns []string
	// For CHECK constraints
	checkExpr parser.IA_exprContext
}

func newSymbolTable() *symbolTable {
	return &symbolTable{
		tables:                make(map[string]*tableDef),
		indexes:               make(map[string]*indexDef),
		views:                 make(map[string]*viewDef),
		schemaConstraintNames: make(map[string]map[string]*constraintDef),
		tableConstraintNames:  make(map[string]map[string]*constraintDef),
	}
}

func (st *symbolTable) addTable(schema, table string, line int) *tableDef {
	key := qualifiedName(schema, table)
	if _, exists := st.tables[key]; !exists {
		st.tables[key] = &tableDef{
			schemaName:  schema,
			tableName:   table,
			columns:     make(map[string]*columnDef),
			primaryKeys: make([]*constraintDef, 0),
			constraints: make([]*constraintDef, 0),
			line:        line,
		}
	}
	return st.tables[key]
}

func (st *symbolTable) getTable(schema, table string) *tableDef {
	return st.tables[qualifiedName(schema, table)]
}

func (st *symbolTable) addIndex(schema, index, table string, line int) {
	key := qualifiedName(schema, index)
	st.indexes[key] = &indexDef{
		schemaName: schema,
		indexName:  index,
		tableName:  table,
		line:       line,
	}
}

func (st *symbolTable) addView(schema, view string, selectStmt parser.ISelectstmtContext, line int) {
	key := qualifiedName(schema, view)
	st.views[key] = &viewDef{
		schemaName: schema,
		viewName:   view,
		selectStmt: selectStmt,
		line:       line,
	}
}

func (st *symbolTable) addConstraintName(schema, table, constraintName string, constraint *constraintDef) bool {
	// PRIMARY KEY and UNIQUE constraints must be unique within schema
	// CHECK and FOREIGN KEY constraints must be unique within table
	isPKOrUK := constraint.constraintType == "PRIMARY KEY" || constraint.constraintType == "UNIQUE"

	if isPKOrUK {
		// Schema-level uniqueness for PK/UK
		if st.schemaConstraintNames[schema] == nil {
			st.schemaConstraintNames[schema] = make(map[string]*constraintDef)
		}
		if _, exists := st.schemaConstraintNames[schema][constraintName]; exists {
			return false // Duplicate
		}
		st.schemaConstraintNames[schema][constraintName] = constraint
	} else {
		// Table-level uniqueness for CHECK/FK
		tableKey := qualifiedName(schema, table)
		if st.tableConstraintNames[tableKey] == nil {
			st.tableConstraintNames[tableKey] = make(map[string]*constraintDef)
		}
		if _, exists := st.tableConstraintNames[tableKey][constraintName]; exists {
			return false // Duplicate
		}
		st.tableConstraintNames[tableKey][constraintName] = constraint
	}
	return true
}

func qualifiedName(schema, name string) string {
	if schema == "" {
		schema = "public"
	}
	return schema + "." + name
}

// Pass 1: Collect definitions

func (c *sdlIntegrityChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		schemaName = extractSchemaName(allQualifiedNames[0])
		tableName = extractTableName(allQualifiedNames[0])
	}

	if schemaName == "" {
		schemaName = "public"
	}

	line := ctx.GetStart().GetLine()

	// Check for duplicate table name
	if existing := c.symbolTable.getTable(schemaName, tableName); existing != nil {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status: storepb.Advice_ERROR,
			Code:   code.SDLDuplicateTableName.Int32(),
			Title:  "Duplicate table name",
			Content: fmt.Sprintf(
				"Table '%s.%s' is defined multiple times in the SDL.\n\n"+
					"First definition at line %d\n"+
					"Duplicate definition at line %d\n\n"+
					"Each table can only be defined once per schema.",
				schemaName, tableName, existing.line, line,
			),
			StartPosition: &storepb.Position{
				Line: int32(line),
			},
		})
		return
	}

	// Add table to symbol table
	table := c.symbolTable.addTable(schemaName, tableName, line)

	// Collect column definitions and constraints
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()

		position := 0
		for _, elem := range allElements {
			// Collect column definitions
			if elem.ColumnDef() != nil {
				colDef := elem.ColumnDef()
				colName := colDef.Colid().GetText()

				// Check for duplicate column name
				if _, exists := table.columns[colName]; exists {
					c.adviceList = append(c.adviceList, &storepb.Advice{
						Status: storepb.Advice_ERROR,
						Code:   code.SDLDuplicateColumnName.Int32(),
						Title:  "Duplicate column name",
						Content: fmt.Sprintf(
							"Column '%s' is defined multiple times in table '%s.%s'.\n\n"+
								"Each column can only be defined once per table.",
							colName, schemaName, tableName,
						),
						StartPosition: &storepb.Position{
							Line: int32(colDef.GetStart().GetLine()),
						},
					})
					continue
				}

				// Extract column data type
				dataType := c.extractDataType(colDef.Typename())

				// Check for NOT NULL constraint
				notNull := false
				if colDef.Colquallist() != nil {
					for _, qual := range colDef.Colquallist().AllColconstraint() {
						if qual.Colconstraintelem() != nil {
							elem := qual.Colconstraintelem()
							if elem.NOT() != nil && elem.NULL_P() != nil {
								notNull = true
							}
						}
					}
				}

				table.columns[colName] = &columnDef{
					name:     colName,
					dataType: dataType,
					notNull:  notNull,
					position: position,
					line:     colDef.GetStart().GetLine(),
				}
				position++
			}

			// Collect table constraints
			if elem.Tableconstraint() != nil {
				c.collectTableConstraint(elem.Tableconstraint(), table)
			}
		}
	}
}

func (c *sdlIntegrityChecker) collectTableConstraint(ctx parser.ITableconstraintContext, table *tableDef) {
	constraint := &constraintDef{
		schemaName: table.schemaName,
		tableName:  table.tableName,
		line:       ctx.GetStart().GetLine(),
	}

	// Get constraint name if present
	if ctx.CONSTRAINT() != nil {
		if ctx.Name() != nil && ctx.Name().Colid() != nil {
			constraint.name = ctx.Name().Colid().GetText()
		}
	}

	if ctx.Constraintelem() == nil {
		return
	}

	elem := ctx.Constraintelem()

	// Determine constraint type and extract details
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		constraint.constraintType = "PRIMARY KEY"
		table.primaryKeys = append(table.primaryKeys, constraint)
	} else if elem.UNIQUE() != nil {
		constraint.constraintType = "UNIQUE"
	} else if elem.CHECK() != nil {
		constraint.constraintType = "CHECK"
		// Extract CHECK expression
		if elem.A_expr() != nil {
			constraint.checkExpr = elem.A_expr()
		}
	} else if elem.FOREIGN() != nil && elem.KEY() != nil {
		constraint.constraintType = "FOREIGN KEY"
		// Extract FK columns
		if elem.Columnlist() != nil {
			for _, col := range elem.Columnlist().AllColumnElem() {
				if col.Colid() != nil {
					constraint.fkColumns = append(constraint.fkColumns, col.Colid().GetText())
				}
			}
		}
		// Extract referenced table and columns
		if elem.Qualified_name() != nil {
			constraint.fkReferencedSchema = extractSchemaName(elem.Qualified_name())
			constraint.fkReferencedTable = extractTableName(elem.Qualified_name())
			if constraint.fkReferencedSchema == "" {
				constraint.fkReferencedSchema = "public"
			}
		}
		if elem.Opt_column_list() != nil && elem.Opt_column_list().Columnlist() != nil {
			for _, col := range elem.Opt_column_list().Columnlist().AllColumnElem() {
				if col.Colid() != nil {
					constraint.fkReferencedColumns = append(constraint.fkReferencedColumns, col.Colid().GetText())
				}
			}
		}
	}

	// Check for duplicate constraint name
	// PRIMARY KEY and UNIQUE: must be unique within schema
	// CHECK and FOREIGN KEY: must be unique within table
	if constraint.name != "" {
		if !c.symbolTable.addConstraintName(table.schemaName, table.tableName, constraint.name, constraint) {
			isPKOrUK := constraint.constraintType == "PRIMARY KEY" || constraint.constraintType == "UNIQUE"
			var existing *constraintDef
			var scope string
			var scopeDetail string

			if isPKOrUK {
				existing = c.symbolTable.schemaConstraintNames[table.schemaName][constraint.name]
				scope = "schema"
				scopeDetail = "PostgreSQL requires PRIMARY KEY and UNIQUE constraint names to be unique within a schema."
			} else {
				tableKey := qualifiedName(table.schemaName, table.tableName)
				existing = c.symbolTable.tableConstraintNames[tableKey][constraint.name]
				scope = "table"
				scopeDetail = "PostgreSQL requires CHECK and FOREIGN KEY constraint names to be unique within a table."
			}

			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status: storepb.Advice_ERROR,
				Code:   code.SDLDuplicateConstraintName.Int32(),
				Title:  "Duplicate constraint name",
				Content: fmt.Sprintf(
					"Constraint '%s' is defined multiple times in %s '%s'.\n\n"+
						"First definition: %s on table '%s' (line %d)\n"+
						"Duplicate definition: %s on table '%s' (line %d)\n\n"+
						"%s\n"+
						"Use different names or let PostgreSQL generate unique names automatically.",
					constraint.name, scope, table.schemaName,
					existing.constraintType, existing.tableName, existing.line,
					constraint.constraintType, table.tableName, constraint.line,
					scopeDetail,
				),
				StartPosition: &storepb.Position{
					Line: int32(constraint.line),
				},
			})
		}
	}

	table.constraints = append(table.constraints, constraint)
}

func (*sdlIntegrityChecker) extractDataType(typeCtx parser.ITypenameContext) *dataType {
	if typeCtx == nil {
		return &dataType{baseType: "UNKNOWN"}
	}

	dt := &dataType{}

	// Use GetText() to extract the full type string
	typeText := strings.ToUpper(typeCtx.GetText())

	// Parse the type to separate base type from modifiers
	// e.g., "VARCHAR(100)" -> baseType="VARCHAR", length=100
	// e.g., "NUMERIC(10,2)" -> baseType="NUMERIC", precision=10, scale=2

	// Find opening parenthesis
	parenIdx := strings.Index(typeText, "(")
	if parenIdx == -1 {
		// No modifiers, just the base type
		dt.baseType = typeText
	} else {
		// Extract base type
		dt.baseType = typeText[:parenIdx]

		// Extract modifiers (numbers inside parentheses)
		modifiers := typeText[parenIdx+1:]
		if closeIdx := strings.Index(modifiers, ")"); closeIdx != -1 {
			modifiers = modifiers[:closeIdx]
		}

		// Split by comma for precision/scale
		parts := strings.Split(modifiers, ",")
		if len(parts) >= 1 {
			if length, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
				dt.length = length
				dt.precision = length // For NUMERIC types, this is precision
			}
		}
		if len(parts) >= 2 {
			if scale, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				dt.scale = scale
			}
		}
	}

	// Check for array notation [] and strip it from base type
	if strings.HasSuffix(typeText, "[]") {
		dt.baseType = strings.TrimSuffix(dt.baseType, "[]")
	}

	return dt
}

func (c *sdlIntegrityChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get index name
	var indexName string
	if ctx.Name() != nil && ctx.Name().Colid() != nil {
		indexName = ctx.Name().Colid().GetText()
	}

	// Get table name and schema
	var schemaName, tableName string
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
		tableName = extractTableName(ctx.Relation_expr().Qualified_name())
	}

	if schemaName == "" {
		schemaName = "public"
	}

	if indexName == "" {
		return // Unnamed index check is handled by sdl_check.go
	}

	line := ctx.GetStart().GetLine()

	// Check for duplicate index name
	key := qualifiedName(schemaName, indexName)
	if existing, exists := c.symbolTable.indexes[key]; exists {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status: storepb.Advice_ERROR,
			Code:   code.SDLDuplicateIndexName.Int32(),
			Title:  "Duplicate index name",
			Content: fmt.Sprintf(
				"Index '%s.%s' is defined multiple times in the SDL.\n\n"+
					"First definition at line %d\n"+
					"Duplicate definition at line %d\n\n"+
					"Each index name must be unique within a schema.",
				schemaName, indexName, existing.line, line,
			),
			StartPosition: &storepb.Position{
				Line: int32(line),
			},
		})
		return
	}

	c.symbolTable.addIndex(schemaName, indexName, tableName, line)
}

func (c *sdlIntegrityChecker) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var viewName, schemaName string
	if ctx.Qualified_name() != nil {
		schemaName = extractSchemaName(ctx.Qualified_name())
		viewName = extractTableName(ctx.Qualified_name())
	}

	if schemaName == "" {
		schemaName = "public"
	}

	line := ctx.GetStart().GetLine()

	// Get the SELECT statement
	var selectStmt parser.ISelectstmtContext
	if ctx.Selectstmt() != nil {
		selectStmt = ctx.Selectstmt()
	}

	c.symbolTable.addView(schemaName, viewName, selectStmt, line)
}

// Pass 2: Validate references

func (c *sdlIntegrityChecker) checkMultiplePrimaryKeys() {
	for _, table := range c.symbolTable.tables {
		if len(table.primaryKeys) > 1 {
			var pkDescriptions []string
			for i, pk := range table.primaryKeys {
				name := pk.name
				if name == "" {
					name = "<unnamed>"
				}
				pkDescriptions = append(pkDescriptions, fmt.Sprintf("  %d. %s at line %d", i+1, name, pk.line))
			}

			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status: storepb.Advice_ERROR,
				Code:   code.SDLMultiplePrimaryKey.Int32(),
				Title:  "Multiple primary keys defined",
				Content: fmt.Sprintf(
					"Table '%s.%s' has multiple PRIMARY KEY constraints defined.\n\n"+
						"Found %d primary key constraints:\n%s\n\n"+
						"A table can only have one PRIMARY KEY constraint.\n"+
						"If you need to enforce uniqueness on multiple column combinations, use UNIQUE constraints instead.",
					table.schemaName, table.tableName,
					len(table.primaryKeys),
					strings.Join(pkDescriptions, "\n"),
				),
				StartPosition: &storepb.Position{
					Line: int32(table.primaryKeys[1].line), // Point to second PK
				},
			})
		}
	}
}

type tableReference struct {
	schemaName string
	tableName  string
}

// getViewDependenciesFromAST extracts view dependencies using ExtractAccessTables
// This is the same approach used in generate_migration.go for consistent behavior
func (c *sdlIntegrityChecker) getViewDependenciesFromAST(selectStmt parser.ISelectstmtContext, schemaName string) []tableReference {
	if selectStmt == nil {
		return []tableReference{}
	}

	// Extract CTE names from WITH clause first
	cteNames := c.extractCTENames(selectStmt)

	// Get the SELECT statement text from the AST
	var selectStatement string
	if tokenStream := selectStmt.GetParser().GetTokenStream(); tokenStream != nil {
		start := selectStmt.GetStart()
		stop := selectStmt.GetStop()
		if start != nil && stop != nil {
			selectStatement = tokenStream.GetTextFromTokens(start, stop)
		}
	}

	if selectStatement == "" {
		return []tableReference{}
	}

	// Use ExtractAccessTables to get all table/view references
	accessTables, err := pgparser.ExtractAccessTables(selectStatement, pgparser.ExtractAccessTablesOption{
		DefaultDatabase:        "",
		DefaultSchema:          schemaName,
		SkipMetadataValidation: true,
	})
	if err != nil {
		return []tableReference{}
	}

	// Convert to tableReference and deduplicate, filtering out CTEs
	refMap := make(map[string]tableReference)
	for _, resource := range accessTables {
		// Skip CTE references
		if cteNames[resource.Table] {
			continue
		}

		resourceSchema := resource.Schema
		if resourceSchema == "" {
			resourceSchema = schemaName
		}

		key := qualifiedName(resourceSchema, resource.Table)
		if _, exists := refMap[key]; !exists {
			refMap[key] = tableReference{
				schemaName: resourceSchema,
				tableName:  resource.Table,
			}
		}
	}

	refs := make([]tableReference, 0, len(refMap))
	for _, ref := range refMap {
		refs = append(refs, ref)
	}

	return refs
}

// extractCTENames extracts all CTE names from a SELECT statement's WITH clause
func (*sdlIntegrityChecker) extractCTENames(selectStmt parser.ISelectstmtContext) map[string]bool {
	cteNames := make(map[string]bool)

	if selectStmt == nil {
		return cteNames
	}

	// Get the select_no_parens from selectstmt
	selectNoParens := selectStmt.Select_no_parens()
	if selectNoParens == nil {
		return cteNames
	}

	// Get the WITH clause
	withClause := selectNoParens.With_clause()
	if withClause == nil {
		return cteNames
	}

	// Extract CTE names from cte_list
	cteList := withClause.Cte_list()
	if cteList == nil {
		return cteNames
	}

	for _, cte := range cteList.AllCommon_table_expr() {
		if cte.Name() != nil {
			cteName := pgparser.NormalizePostgreSQLName(cte.Name())
			cteNames[cteName] = true
		}
	}

	return cteNames
}

// Helper functions

func (*sdlIntegrityChecker) getConstraintName(constraint *constraintDef) string {
	if constraint.name != "" {
		return constraint.name
	}
	return "<unnamed>"
}

func (*sdlIntegrityChecker) getColumnList(table *tableDef) string {
	cols := make([]string, 0, len(table.columns))
	for colName := range table.columns {
		cols = append(cols, colName)
	}
	if len(cols) == 0 {
		return "<no columns>"
	}
	return strings.Join(cols, ", ")
}
