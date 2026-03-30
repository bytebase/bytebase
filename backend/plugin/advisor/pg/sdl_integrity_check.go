package pg

import (
	"fmt"
	"strings"

	omnipg "github.com/bytebase/omni/pg"
	"github.com/bytebase/omni/pg/ast"

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
		// Validate syntax using ANTLR parser first (stricter than omni for some edge cases)
		if _, err := pgparser.ParsePostgreSQL(statement); err != nil {
			return map[string][]*storepb.Advice{
				filePath: {{
					Status:  storepb.Advice_ERROR,
					Code:    code.StatementSyntaxError.Int32(),
					Title:   "SQL syntax error",
					Content: fmt.Sprintf("Failed to parse SQL in file '%s': %v", filePath, err),
				}},
			}, nil
		}

		stmts, err := pgparser.ParsePg(statement)
		if err != nil {
			return map[string][]*storepb.Advice{
				filePath: {{
					Status:  storepb.Advice_ERROR,
					Code:    code.StatementSyntaxError.Int32(),
					Title:   "SQL syntax error",
					Content: fmt.Sprintf("Failed to parse SQL in file '%s': %v", filePath, err),
				}},
			}, nil
		}

		st, advices := buildFileSymbolTable(stmts, statement)

		checker := &sdlIntegrityChecker{
			symbolTable: st,
			adviceList:  advices,
		}

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
						getConstraintName(constraint),
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
							getConstraintName(constraint),
							table.schemaName, table.tableName,
							refCol,
							constraint.fkReferencedSchema, constraint.fkReferencedTable,
							table.schemaName, table.tableName, fkCol,
							constraint.fkReferencedSchema, constraint.fkReferencedTable, refCol,
							constraint.fkReferencedSchema, constraint.fkReferencedTable,
							getColumnList(refTable),
						),
						StartPosition: &storepb.Position{Line: int32(constraint.line)},
					})
				}
			}
		}
	}
}

// validateViewDependenciesWithMergedTable validates view dependencies using merged symbol table
func (c *sdlIntegrityChecker) validateViewDependenciesWithMergedTable(merged *mergedSymbolTable) {
	for _, view := range c.symbolTable.views {
		if view.selectQuery == nil {
			continue
		}

		dependencies := getViewDependencies(view.selectQuery, view.schemaName)

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

// sdlIntegrityChecker holds the symbol table and advices for integrity checking
type sdlIntegrityChecker struct {
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
	schemaName  string
	viewName    string
	selectQuery ast.Node // the SELECT query AST node
	line        int
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

func (st *symbolTable) addView(schema, view string, selectQuery ast.Node, line int) {
	key := qualifiedName(schema, view)
	st.views[key] = &viewDef{
		schemaName:  schema,
		viewName:    view,
		selectQuery: selectQuery,
		line:        line,
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

// buildFileSymbolTable iterates omni statements and builds the symbol table.
func buildFileSymbolTable(stmts []omnipg.Statement, fileText string) (*symbolTable, []*storepb.Advice) {
	st := newSymbolTable()
	var advices []*storepb.Advice
	for _, stmt := range stmts {
		switch n := stmt.AST.(type) {
		case *ast.CreateStmt:
			processCreateStmt(n, fileText, st, &advices)
		case *ast.IndexStmt:
			processIndexStmt(n, fileText, st, &advices)
		case *ast.ViewStmt:
			processViewStmt(n, fileText, st)
		default:
		}
	}
	return st, advices
}

// locToLine converts a byte offset to a 1-based line number.
func locToLine(fileText string, loc ast.Loc) int {
	if loc.Start < 0 || fileText == "" {
		return 1
	}
	pos := pgparser.ByteOffsetToRunePosition(fileText, loc.Start)
	return int(pos.Line)
}

func processCreateStmt(n *ast.CreateStmt, fileText string, st *symbolTable, advices *[]*storepb.Advice) {
	if n.Relation == nil {
		return
	}

	tableName := n.Relation.Relname
	schemaName := n.Relation.Schemaname
	if schemaName == "" {
		schemaName = "public"
	}

	line := locToLine(fileText, n.Loc)

	// Check for duplicate table name
	if existing := st.getTable(schemaName, tableName); existing != nil {
		*advices = append(*advices, &storepb.Advice{
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

	table := st.addTable(schemaName, tableName, line)

	// Process columns from TableElts
	position := 0
	if n.TableElts != nil {
		for _, item := range n.TableElts.Items {
			switch elem := item.(type) {
			case *ast.ColumnDef:
				processColumnDef(elem, fileText, schemaName, tableName, table, &position, advices)
			case *ast.Constraint:
				processTableConstraint(elem, fileText, table, st, advices)
			default:
			}
		}
	}

	// Process Constraints list
	if n.Constraints != nil {
		for _, item := range n.Constraints.Items {
			if c, ok := item.(*ast.Constraint); ok {
				processTableConstraint(c, fileText, table, st, advices)
			}
		}
	}
}

func processColumnDef(col *ast.ColumnDef, fileText, schemaName, tableName string, table *tableDef, position *int, advices *[]*storepb.Advice) {
	colName := col.Colname

	colLine := locToLine(fileText, col.Loc)

	// Check for duplicate column name
	if _, exists := table.columns[colName]; exists {
		*advices = append(*advices, &storepb.Advice{
			Status: storepb.Advice_ERROR,
			Code:   code.SDLDuplicateColumnName.Int32(),
			Title:  "Duplicate column name",
			Content: fmt.Sprintf(
				"Column '%s' is defined multiple times in table '%s.%s'.\n\n"+
					"Each column can only be defined once per table.",
				colName, schemaName, tableName,
			),
			StartPosition: &storepb.Position{
				Line: int32(colLine),
			},
		})
		return
	}

	dt := extractOmniDataType(col.TypeName)

	// Check for NOT NULL: either via IsNotNull flag or via column constraints
	notNull := col.IsNotNull
	if !notNull && col.Constraints != nil {
		for _, cItem := range col.Constraints.Items {
			if c, ok := cItem.(*ast.Constraint); ok && c.Contype == ast.CONSTR_NOTNULL {
				notNull = true
				break
			}
		}
	}

	table.columns[colName] = &columnDef{
		name:     colName,
		dataType: dt,
		notNull:  notNull,
		position: *position,
		line:     colLine,
	}
	*position++
}

func processTableConstraint(c *ast.Constraint, fileText string, table *tableDef, st *symbolTable, advices *[]*storepb.Advice) {
	constraint := &constraintDef{
		schemaName: table.schemaName,
		tableName:  table.tableName,
		line:       locToLine(fileText, c.Loc),
		name:       c.Conname,
	}

	switch c.Contype {
	case ast.CONSTR_PRIMARY:
		constraint.constraintType = "PRIMARY KEY"
		table.primaryKeys = append(table.primaryKeys, constraint)
	case ast.CONSTR_UNIQUE:
		constraint.constraintType = "UNIQUE"
	case ast.CONSTR_CHECK:
		constraint.constraintType = "CHECK"
	case ast.CONSTR_FOREIGN:
		constraint.constraintType = "FOREIGN KEY"
		// Extract FK columns
		if c.FkAttrs != nil {
			for _, item := range c.FkAttrs.Items {
				if s, ok := item.(*ast.String); ok {
					constraint.fkColumns = append(constraint.fkColumns, s.Str)
				}
			}
		}
		// Extract referenced table
		if c.Pktable != nil {
			constraint.fkReferencedTable = c.Pktable.Relname
			constraint.fkReferencedSchema = c.Pktable.Schemaname
			if constraint.fkReferencedSchema == "" {
				constraint.fkReferencedSchema = "public"
			}
		}
		// Extract referenced columns
		if c.PkAttrs != nil {
			for _, item := range c.PkAttrs.Items {
				if s, ok := item.(*ast.String); ok {
					constraint.fkReferencedColumns = append(constraint.fkReferencedColumns, s.Str)
				}
			}
		}
	default:
		return
	}

	// Check for duplicate constraint name
	if constraint.name != "" {
		if !st.addConstraintName(table.schemaName, table.tableName, constraint.name, constraint) {
			isPKOrUK := constraint.constraintType == "PRIMARY KEY" || constraint.constraintType == "UNIQUE"
			var existing *constraintDef
			var scope string
			var scopeDetail string

			if isPKOrUK {
				existing = st.schemaConstraintNames[table.schemaName][constraint.name]
				scope = "schema"
				scopeDetail = "PostgreSQL requires PRIMARY KEY and UNIQUE constraint names to be unique within a schema."
			} else {
				tableKey := qualifiedName(table.schemaName, table.tableName)
				existing = st.tableConstraintNames[tableKey][constraint.name]
				scope = "table"
				scopeDetail = "PostgreSQL requires CHECK and FOREIGN KEY constraint names to be unique within a table."
			}

			*advices = append(*advices, &storepb.Advice{
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

func processIndexStmt(n *ast.IndexStmt, fileText string, st *symbolTable, advices *[]*storepb.Advice) {
	indexName := n.Idxname

	var schemaName, tableName string
	if n.Relation != nil {
		schemaName = n.Relation.Schemaname
		tableName = n.Relation.Relname
	}
	if schemaName == "" {
		schemaName = "public"
	}

	if indexName == "" {
		return // Unnamed index check is handled by sdl_check.go
	}

	line := locToLine(fileText, n.Loc)

	// Check for duplicate index name
	key := qualifiedName(schemaName, indexName)
	if existing, exists := st.indexes[key]; exists {
		*advices = append(*advices, &storepb.Advice{
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

	st.addIndex(schemaName, indexName, tableName, line)
}

func processViewStmt(n *ast.ViewStmt, fileText string, st *symbolTable) {
	if n.View == nil {
		return
	}

	viewName := n.View.Relname
	schemaName := n.View.Schemaname
	if schemaName == "" {
		schemaName = "public"
	}

	line := locToLine(fileText, n.Loc)

	st.addView(schemaName, viewName, n.Query, line)
}

// extractOmniDataType extracts data type information from an omni TypeName.
func extractOmniDataType(tn *ast.TypeName) *dataType {
	if tn == nil {
		return &dataType{baseType: "UNKNOWN"}
	}

	dt := &dataType{}

	// Build the type name from Names list
	typeName := omniTypeNameStr(tn)
	dt.baseType = strings.ToUpper(typeName)

	// Extract length/precision from Typmods
	if tn.Typmods != nil {
		for i, item := range tn.Typmods.Items {
			val := extractIntValue(item)
			if val < 0 {
				continue
			}
			if i == 0 {
				dt.length = val
				dt.precision = val
			}
			if i == 1 {
				dt.scale = val
			}
		}
	}

	// Check for array type
	if tn.ArrayBounds != nil && len(tn.ArrayBounds.Items) > 0 {
		dt.baseType = strings.TrimSuffix(dt.baseType, "[]")
	}

	return dt
}

// extractIntValue extracts an integer value from an AST node.
func extractIntValue(item ast.Node) int {
	switch v := item.(type) {
	case *ast.Integer:
		return int(v.Ival)
	case *ast.A_Const:
		if iv, ok := v.Val.(*ast.Integer); ok {
			return int(iv.Ival)
		}
	default:
	}
	return -1
}

// omniTypeNameStr builds a type name string from a TypeName node.
func omniTypeNameStr(tn *ast.TypeName) string {
	if tn == nil || tn.Names == nil {
		return "UNKNOWN"
	}

	var parts []string
	for _, item := range tn.Names.Items {
		if s, ok := item.(*ast.String); ok {
			parts = append(parts, s.Str)
		}
	}

	if len(parts) == 0 {
		return "UNKNOWN"
	}

	// The Names list typically contains [catalog, typename] e.g. ["pg_catalog", "int4"]
	// We want just the last part (the actual type name).
	typeName := parts[len(parts)-1]

	// Map common internal type names to SQL type names
	typeMap := map[string]string{
		"int2":    "SMALLINT",
		"int4":    "INTEGER",
		"int8":    "BIGINT",
		"float4":  "REAL",
		"float8":  "DOUBLE PRECISION",
		"bool":    "BOOLEAN",
		"varchar": "VARCHAR",
		"bpchar":  "CHARACTER",
		"numeric": "NUMERIC",
	}

	if mapped, ok := typeMap[typeName]; ok {
		return mapped
	}

	return typeName
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

// getViewDependencies extracts view dependencies by walking the AST for RangeVar references.
func getViewDependencies(query ast.Node, defaultSchema string) []tableReference {
	if query == nil {
		return nil
	}

	cteNames := extractCTENames(query)
	var refs []tableReference
	seen := make(map[string]bool)

	ast.Inspect(query, func(n ast.Node) bool {
		rv, ok := n.(*ast.RangeVar)
		if !ok {
			return true
		}
		if rv.Relname == "" {
			return true
		}
		// Skip CTE references
		if cteNames[rv.Relname] {
			return true
		}

		schema := rv.Schemaname
		if schema == "" {
			schema = defaultSchema
		}
		key := schema + "." + rv.Relname
		if !seen[key] {
			seen[key] = true
			refs = append(refs, tableReference{schemaName: schema, tableName: rv.Relname})
		}
		return true
	})

	return refs
}

// extractCTENames extracts all CTE names from a query's WITH clause.
func extractCTENames(query ast.Node) map[string]bool {
	cteNames := make(map[string]bool)
	sel, ok := query.(*ast.SelectStmt)
	if !ok || sel == nil || sel.WithClause == nil || sel.WithClause.Ctes == nil {
		return cteNames
	}
	for _, item := range sel.WithClause.Ctes.Items {
		if cte, ok := item.(*ast.CommonTableExpr); ok {
			cteNames[cte.Ctename] = true
		}
	}
	return cteNames
}

// Helper functions

func getConstraintName(constraint *constraintDef) string {
	if constraint.name != "" {
		return constraint.name
	}
	return "<unnamed>"
}

func getColumnList(table *tableDef) string {
	cols := make([]string, 0, len(table.columns))
	for colName := range table.columns {
		cols = append(cols, colName)
	}
	if len(cols) == 0 {
		return "<no columns>"
	}
	return strings.Join(cols, ", ")
}
