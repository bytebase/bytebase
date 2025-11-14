package oracle

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	oracleparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_ORACLE, GetDatabaseMetadata)
}

// GetDatabaseMetadata parses the Oracle schema text and returns the database metadata.
func GetDatabaseMetadata(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	results, err := oracleparser.ParsePLSQL(schemaText)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse Oracle schema")
	}
	if len(results) == 0 {
		return nil, errors.New("no parse results")
	}

	extractor := &metadataExtractor{
		currentSchema:     "",
		currentTable:      "",
		tables:            make(map[string]*storepb.TableMetadata),
		views:             make(map[string]*storepb.ViewMetadata),
		materializedViews: make(map[string]*storepb.MaterializedViewMetadata),
		functions:         make(map[string]*storepb.FunctionMetadata),
		procedures:        make(map[string]*storepb.ProcedureMetadata),
		triggers:          make(map[string]*storepb.TriggerMetadata),
		sequences:         make(map[string]*storepb.SequenceMetadata),
		packages:          make(map[string]*storepb.PackageMetadata),
		inlinePrimaryKeys: make(map[string][]string),
		inlineUniqueKeys:  make(map[string][]string),
	}

	// Walk all parse result trees to extract metadata from all statements
	for _, result := range results {
		if result.Tree != nil {
			antlr.ParseTreeWalkerDefault.Walk(extractor, result.Tree)
		}
	}

	if extractor.err != nil {
		return nil, extractor.err
	}

	// Build the final metadata structure
	schemaMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    extractor.currentDatabase,
		Schemas: []*storepb.SchemaMetadata{},
	}

	// Create single schema for Oracle (Oracle uses schemas, not separate databases)
	schemaName := extractor.currentSchema
	if schemaName == "" {
		schemaName = "PUBLIC"
	}

	schema := &storepb.SchemaMetadata{
		Name:              schemaName,
		Tables:            []*storepb.TableMetadata{},
		Views:             []*storepb.ViewMetadata{},
		MaterializedViews: []*storepb.MaterializedViewMetadata{},
		Procedures:        []*storepb.ProcedureMetadata{},
		Functions:         []*storepb.FunctionMetadata{},
		Sequences:         []*storepb.SequenceMetadata{},
	}

	// Sort and add tables (exclude tables that are actually materialized views)
	var tableNames []string
	for name := range extractor.tables {
		// Skip tables that are actually materialized views
		if _, isMaterializedView := extractor.materializedViews[name]; !isMaterializedView {
			tableNames = append(tableNames, name)
		}
	}
	slices.Sort(tableNames)
	for _, name := range tableNames {
		schema.Tables = append(schema.Tables, extractor.tables[name])
	}

	// Sort and add views
	var viewNames []string
	for name := range extractor.views {
		viewNames = append(viewNames, name)
	}
	slices.Sort(viewNames)
	for _, name := range viewNames {
		schema.Views = append(schema.Views, extractor.views[name])
	}

	// Sort and add materialized views
	var materializedViewNames []string
	for name := range extractor.materializedViews {
		materializedViewNames = append(materializedViewNames, name)
	}
	slices.Sort(materializedViewNames)
	for _, name := range materializedViewNames {
		schema.MaterializedViews = append(schema.MaterializedViews, extractor.materializedViews[name])
	}

	// Sort and add functions
	var functionNames []string
	for name := range extractor.functions {
		functionNames = append(functionNames, name)
	}
	slices.Sort(functionNames)
	for _, name := range functionNames {
		schema.Functions = append(schema.Functions, extractor.functions[name])
	}

	// Sort and add procedures
	var procedureNames []string
	for name := range extractor.procedures {
		procedureNames = append(procedureNames, name)
	}
	slices.Sort(procedureNames)
	for _, name := range procedureNames {
		schema.Procedures = append(schema.Procedures, extractor.procedures[name])
	}

	// Sort and add sequences
	var sequenceNames []string
	for name := range extractor.sequences {
		sequenceNames = append(sequenceNames, name)
	}
	slices.Sort(sequenceNames)
	for _, name := range sequenceNames {
		schema.Sequences = append(schema.Sequences, extractor.sequences[name])
	}

	schemaMetadata.Schemas = append(schemaMetadata.Schemas, schema)
	return schemaMetadata, nil
}

// metadataExtractor walks the parse tree and extracts Oracle metadata
type metadataExtractor struct {
	*parser.BasePlSqlParserListener

	currentDatabase   string
	currentSchema     string
	currentTable      string // Track current table being processed
	tables            map[string]*storepb.TableMetadata
	views             map[string]*storepb.ViewMetadata
	materializedViews map[string]*storepb.MaterializedViewMetadata
	functions         map[string]*storepb.FunctionMetadata
	procedures        map[string]*storepb.ProcedureMetadata
	triggers          map[string]*storepb.TriggerMetadata
	sequences         map[string]*storepb.SequenceMetadata
	packages          map[string]*storepb.PackageMetadata
	err               error
	inlinePrimaryKeys map[string][]string // Track inline primary key columns by table
	inlineUniqueKeys  map[string][]string // Track inline unique key columns by table
}

// Helper function to get or create table
func (e *metadataExtractor) getOrCreateTable(tableName string) *storepb.TableMetadata {
	if table, exists := e.tables[tableName]; exists {
		return table
	}

	table := &storepb.TableMetadata{
		Name:             tableName,
		Columns:          []*storepb.ColumnMetadata{},
		Indexes:          []*storepb.IndexMetadata{},
		ForeignKeys:      []*storepb.ForeignKeyMetadata{},
		CheckConstraints: []*storepb.CheckConstraintMetadata{},
		Triggers:         []*storepb.TriggerMetadata{},
		Partitions:       []*storepb.TablePartitionMetadata{},
	}
	e.tables[tableName] = table
	return table
}

// EnterCreate_table is called when entering a create table statement
func (e *metadataExtractor) EnterCreate_table(ctx *parser.Create_tableContext) {
	if e.err != nil {
		return
	}

	if ctx.Table_name() == nil {
		return
	}

	// Extract schema and table name
	if ctx.Schema_name() != nil {
		e.currentSchema = oracleparser.NormalizeSchemaName(ctx.Schema_name())
	}

	tableName := oracleparser.NormalizeTableName(ctx.Table_name())
	if tableName == "" {
		return
	}

	// Set current table for inline constraint tracking
	e.currentTable = tableName
	table := e.getOrCreateTable(tableName)

	// Extract table elements (columns, constraints)
	if ctx.Relational_table() != nil {
		e.extractRelationalTable(ctx.Relational_table(), table)

		// Process inline constraints after all columns are parsed
		e.processInlineConstraints(tableName, table)
	}

	// Clear current table
	e.currentTable = ""
}

// EnterCreate_index is called when entering a create index statement
func (e *metadataExtractor) EnterCreate_index(ctx *parser.Create_indexContext) {
	if e.err != nil {
		return
	}

	if ctx.Index_name() == nil {
		return
	}

	// Extract schema and index name
	schemaName, indexName := oracleparser.NormalizeIndexName(ctx.Index_name())
	if indexName == "" {
		return
	}
	if schemaName != "" {
		e.currentSchema = schemaName
	}

	// Extract table name from the index definition
	tableName := ""
	if ctx.Table_index_clause() != nil && ctx.Table_index_clause().Tableview_name() != nil {
		tableName = normalizeTableViewName(ctx.Table_index_clause().Tableview_name())
	} else if ctx.Cluster_index_clause() != nil {
		// Handle cluster indexes if needed
		return
	}

	if tableName == "" {
		return
	}

	// Get or create the table
	table := e.getOrCreateTable(tableName)

	// Create index metadata
	index := &storepb.IndexMetadata{
		Name:        indexName,
		Primary:     false,
		Unique:      ctx.UNIQUE() != nil,
		Type:        "NORMAL", // Oracle default
		Expressions: []string{},
		Visible:     true, // Oracle indexes are visible by default
	}

	// Determine index type
	if ctx.BITMAP() != nil {
		index.Type = "BITMAP"
	}

	// Extract index expressions
	if ctx.Table_index_clause() != nil {
		e.extractIndexExpressions(ctx.Table_index_clause(), index)
	}

	table.Indexes = append(table.Indexes, index)
}

// extractIndexExpressions extracts column expressions from table index clause using ANTLR parser
func (*metadataExtractor) extractIndexExpressions(ctx parser.ITable_index_clauseContext, index *storepb.IndexMetadata) {
	if ctx == nil {
		return
	}

	isFunctionBased := false
	var descendingFlags []bool

	// Extract index expressions using ANTLR parser
	indexExprOptions := ctx.AllIndex_expr_option()
	for _, exprOption := range indexExprOptions {
		if exprOption == nil {
			continue
		}

		var exprText string
		isDescending := false

		// Get the index expression
		if indexExpr := exprOption.Index_expr(); indexExpr != nil {
			if columnName := indexExpr.Column_name(); columnName != nil {
				// Simple column reference - extract without quotes
				_, _, exprText = oracleparser.NormalizeColumnName(columnName)
				// Remove quotes if present to match database metadata format
				exprText = strings.Trim(exprText, "\"")
			} else if expression := indexExpr.Expression(); expression != nil {
				// Function-based expression
				exprText = getTextFromContext(expression)
				isFunctionBased = true
			}
		}

		// Check for ASC/DESC modifiers
		isDescending = exprOption.DESC() != nil
		// Note: Don't mark simple column DESC as function-based in the parser
		// Oracle may internally treat it as function-based, but for DDL generation purposes,
		// we should generate it as a normal index with explicit ASC/DESC modifiers

		// Store the expression (without ASC/DESC modifiers)
		if exprText != "" {
			index.Expressions = append(index.Expressions, exprText)
			descendingFlags = append(descendingFlags, isDescending)
		}
	}

	// Always set descending flags to match the number of expressions
	if len(descendingFlags) > 0 {
		index.Descending = descendingFlags
	}

	// Check if we have any descending columns to determine if this is function-based
	hasDescending := false
	for _, desc := range descendingFlags {
		if desc {
			hasDescending = true
			break
		}
	}

	// Update index type for function-based indexes
	// Oracle treats indexes with DESC columns or functions as "FUNCTION-BASED NORMAL"
	if (isFunctionBased || hasDescending) && index.Type == "NORMAL" {
		index.Type = "FUNCTION-BASED NORMAL"
	}
}

// EnterCreate_view is called when entering a create view statement
func (e *metadataExtractor) EnterCreate_view(ctx *parser.Create_viewContext) {
	if e.err != nil {
		return
	}

	// Extract schema and view name
	schemaName := ""
	viewName := ""

	if ctx.Schema_name() != nil {
		schemaName = oracleparser.NormalizeSchemaName(ctx.Schema_name())
		e.currentSchema = schemaName
	}

	// Extract view name from GetV() which returns id_expression
	if ctx.GetV() != nil {
		viewName = oracleparser.NormalizeIDExpression(ctx.GetV())
	}

	if viewName == "" {
		return
	}

	// Extract the select statement as the view definition
	definition := ""
	if ctx.Select_only_statement() != nil {
		definition = getTextFromContext(ctx.Select_only_statement())
	}

	view := &storepb.ViewMetadata{
		Name:       viewName,
		Definition: definition,
	}

	e.views[viewName] = view
}

// EnterCreate_materialized_view is called when entering a create materialized view statement
func (e *metadataExtractor) EnterCreate_materialized_view(ctx *parser.Create_materialized_viewContext) {
	if e.err != nil {
		return
	}

	// Extract materialized view name from tableview_name
	viewName := ""

	if ctx.Tableview_name() != nil {
		viewName = normalizeTableViewName(ctx.Tableview_name())
	}

	if viewName == "" {
		return
	}

	// Extract the select statement as the definition
	definition := ""
	if ctx.Select_only_statement() != nil {
		definition = getTextFromContext(ctx.Select_only_statement())
		// Ensure the definition ends with a newline to match database format
		if definition != "" && !strings.HasSuffix(definition, "\n") {
			definition += "\n"
		}
	}

	materializedView := &storepb.MaterializedViewMetadata{
		Name:       viewName,
		Definition: definition,
	}

	e.materializedViews[viewName] = materializedView

	// Ensure this materialized view is not also treated as a table
	// Oracle parsing might have triggered table creation first
	delete(e.tables, viewName)
}

// EnterCreate_sequence is called when entering a create sequence statement
func (e *metadataExtractor) EnterCreate_sequence(ctx *parser.Create_sequenceContext) {
	if e.err != nil {
		return
	}

	if ctx.Sequence_name() == nil {
		return
	}

	// Extract sequence name
	sequenceName := normalizeSequenceName(ctx.Sequence_name())
	if sequenceName == "" {
		return
	}

	sequence := &storepb.SequenceMetadata{
		Name: sequenceName,
	}

	// Extract sequence options
	e.extractSequenceOptions(ctx, sequence)

	e.sequences[sequenceName] = sequence
}

// extractSequenceOptions extracts sequence specification options using ANTLR parser
func (*metadataExtractor) extractSequenceOptions(ctx *parser.Create_sequenceContext, sequence *storepb.SequenceMetadata) {
	// Extract START WITH clause using ANTLR parser
	startClauses := ctx.AllSequence_start_clause()
	for _, startClause := range startClauses {
		if startClause.START() != nil && startClause.WITH() != nil && startClause.UNSIGNED_INTEGER() != nil {
			startValue := startClause.UNSIGNED_INTEGER().GetText()
			if startValue != "" {
				// Store all start values - Oracle metadata may not match DDL
				sequence.Start = startValue
				break
			}
		}
	}

	// Extract sequence specifications using ANTLR parser
	sequenceSpecs := ctx.AllSequence_spec()
	for _, spec := range sequenceSpecs {
		// Extract INCREMENT BY value
		if spec.INCREMENT() != nil && spec.BY() != nil && spec.UNSIGNED_INTEGER() != nil {
			incrementValue := spec.UNSIGNED_INTEGER().GetText()
			if incrementValue != "" && incrementValue != "1" {
				// Only store non-default increment values (Oracle defaults to 1)
				sequence.Increment = incrementValue
			}
		}

		// We could extract other sequence properties here if needed:
		// - MAXVALUE/NOMAXVALUE
		// - MINVALUE/NOMINVALUE
		// - CYCLE/NOCYCLE
		// - CACHE/NOCACHE
		// - ORDER/NOORDER
	}
}

// EnterCreate_function_body is called when entering a create function statement
func (e *metadataExtractor) EnterCreate_function_body(ctx *parser.Create_function_bodyContext) {
	if e.err != nil {
		return
	}

	if ctx.Function_name() == nil {
		return
	}

	// Extract function name
	functionName := normalizeFunctionName(ctx.Function_name())
	if functionName == "" {
		return
	}

	// Extract the function body and ensure it has proper formatting
	definition := getTextFromContext(ctx)

	// Clean up the definition to match expected format
	// Remove CREATE OR REPLACE prefix if present
	if strings.HasPrefix(strings.ToUpper(definition), "CREATE OR REPLACE ") {
		definition = definition[18:] // Remove "CREATE OR REPLACE "
	} else if strings.HasPrefix(strings.ToUpper(definition), "CREATE ") {
		definition = definition[7:] // Remove "CREATE "
	}

	// Ensure definition starts with FUNCTION keyword
	if !strings.HasPrefix(strings.ToUpper(definition), "FUNCTION") {
		definition = "FUNCTION " + definition
	}

	function := &storepb.FunctionMetadata{
		Name:       functionName,
		Definition: definition,
	}

	e.functions[functionName] = function
}

// EnterCreate_procedure_body is called when entering a create procedure statement
func (e *metadataExtractor) EnterCreate_procedure_body(ctx *parser.Create_procedure_bodyContext) {
	if e.err != nil {
		return
	}

	if ctx.Procedure_name() == nil {
		return
	}

	// Extract procedure name
	procedureName := normalizeProcedureName(ctx.Procedure_name())
	if procedureName == "" {
		return
	}

	// Extract the procedure body
	definition := getTextFromContext(ctx)

	procedure := &storepb.ProcedureMetadata{
		Name:       procedureName,
		Definition: definition,
	}

	e.procedures[procedureName] = procedure
}

// extractRelationalTable extracts columns and constraints from relational table definition
func (e *metadataExtractor) extractRelationalTable(ctx parser.IRelational_tableContext, table *storepb.TableMetadata) {
	if ctx == nil {
		return
	}

	// Extract all relational properties
	for _, prop := range ctx.AllRelational_property() {
		if prop == nil {
			continue
		}

		switch {
		case prop.Column_definition() != nil:
			column := e.extractColumnDefinition(prop.Column_definition())
			if column != nil {
				// Set position based on current column count (1-indexed)
				column.Position = int32(len(table.Columns) + 1)
				table.Columns = append(table.Columns, column)
			}
		case prop.Virtual_column_definition() != nil:
			column := e.extractVirtualColumnDefinition(prop.Virtual_column_definition())
			if column != nil {
				// Set position based on current column count (1-indexed)
				column.Position = int32(len(table.Columns) + 1)
				table.Columns = append(table.Columns, column)
			}
		case prop.Out_of_line_constraint() != nil:
			e.extractOutOfLineConstraint(prop.Out_of_line_constraint(), table)
		case prop.Out_of_line_ref_constraint() != nil:
			e.extractOutOfLineRefConstraint(prop.Out_of_line_ref_constraint(), table)
		default:
			// Other relational properties
		}
	}
}

// extractColumnDefinition extracts column metadata from column definition
func (e *metadataExtractor) extractColumnDefinition(ctx parser.IColumn_definitionContext) *storepb.ColumnMetadata {
	if ctx == nil || ctx.Column_name() == nil {
		return nil
	}

	// Extract column name
	columnName := normalizeColumnName(ctx.Column_name())
	if columnName == "" {
		return nil
	}

	column := &storepb.ColumnMetadata{
		Name:     columnName,
		Type:     "VARCHAR2(100)", // Default type
		Nullable: true,            // Default to nullable
	}

	// Extract data type
	if ctx.Datatype() != nil {
		column.Type = e.extractDataType(ctx.Datatype())
	} else if ctx.Regular_id() != nil {
		// Handle user-defined types
		column.Type = strings.ToUpper(ctx.Regular_id().GetText())
	}

	// Extract default value using ANTLR parser
	if ctx.DEFAULT() != nil {
		if expr := ctx.Expression(); expr != nil {
			column.Default = getTextFromContext(expr)
		}
	}

	// Extract collation if specified
	if ctx.COLLATE() != nil && ctx.Column_collation_name() != nil {
		if collationName := ctx.Column_collation_name().Id_expression(); collationName != nil {
			column.Collation = oracleparser.NormalizeIDExpression(collationName)
		}
	}

	// Extract identity clause
	if ctx.Identity_clause() != nil {
		e.extractIdentityClause(ctx.Identity_clause(), column)
	}

	// Extract inline constraints
	for _, constraint := range ctx.AllInline_constraint() {
		e.extractInlineConstraint(constraint, column)
	}

	// Check visibility - don't add to comment as this is structural info
	if ctx.INVISIBLE() != nil {
		// TODO: Set invisible column flag when available in proto
		_ = ctx.INVISIBLE() // Acknowledge the context for now
	}

	return column
}

// extractVirtualColumnDefinition extracts virtual column metadata
func (e *metadataExtractor) extractVirtualColumnDefinition(ctx parser.IVirtual_column_definitionContext) *storepb.ColumnMetadata {
	if ctx == nil || ctx.Column_name() == nil {
		return nil
	}

	columnName := normalizeColumnName(ctx.Column_name())
	if columnName == "" {
		return nil
	}

	column := &storepb.ColumnMetadata{
		Name:     columnName,
		Type:     "NUMBER", // Default type for virtual columns
		Nullable: true,
	}

	// Extract data type if specified
	if ctx.Datatype() != nil {
		column.Type = e.extractDataType(ctx.Datatype())
	}

	// Mark as virtual and extract expression from context
	// Virtual columns have expressions like: col_name AS (expression)
	text := getTextFromContext(ctx)
	if asIdx := strings.Index(strings.ToUpper(text), "AS"); asIdx != -1 {
		// Find the expression in parentheses
		remaining := text[asIdx+2:]
		if openIdx := strings.Index(remaining, "("); openIdx != -1 {
			// Find matching closing parenthesis
			parenCount := 1
			i := openIdx + 1
			for i < len(remaining) && parenCount > 0 {
				switch remaining[i] {
				case '(':
					parenCount++
				case ')':
					parenCount--
				default:
					// Other characters
				}
				i++
			}
			if parenCount == 0 {
				column.Default = remaining[openIdx+1 : i-1]
			}
		}
	}

	// Mark as virtual column - don't add to comment as this is structural info
	// TODO: Set virtual column flag when available in proto

	// Extract inline constraints
	for _, constraint := range ctx.AllInline_constraint() {
		e.extractInlineConstraint(constraint, column)
	}

	return column
}

// extractDataType extracts Oracle data type using ANTLR parser with fallback to text
func (*metadataExtractor) extractDataType(ctx parser.IDatatypeContext) string {
	if ctx == nil {
		return ""
	}

	// For now, use the original approach to get full text and normalize it
	// The ANTLR parsing of complex types like INTERVAL and TIMESTAMP WITH TIME ZONE
	// requires more detailed grammar understanding than currently implemented
	return normalizeDataTypeText(getTextFromContext(ctx))
}

// extractInlineConstraint extracts inline column constraints
func (e *metadataExtractor) extractInlineConstraint(ctx parser.IInline_constraintContext, column *storepb.ColumnMetadata) {
	if ctx == nil {
		return
	}

	// Handle NOT NULL constraint
	if ctx.NOT() != nil && ctx.NULL_() != nil {
		column.Nullable = false
	}

	// Handle NULL constraint (explicitly nullable)
	if ctx.NULL_() != nil && ctx.NOT() == nil {
		column.Nullable = true
	}

	// Handle PRIMARY KEY constraint
	if ctx.PRIMARY() != nil && ctx.KEY() != nil {
		column.Nullable = false
		// Track this column for primary key index creation
		if e.currentTable != "" {
			if e.inlinePrimaryKeys[e.currentTable] == nil {
				e.inlinePrimaryKeys[e.currentTable] = []string{}
			}
			e.inlinePrimaryKeys[e.currentTable] = append(e.inlinePrimaryKeys[e.currentTable], column.Name)
		}
	}

	// Handle UNIQUE constraint
	if ctx.UNIQUE() != nil {
		// Note: UNIQUE columns can be nullable in Oracle
		// Track this column for unique index creation
		if e.currentTable != "" {
			if e.inlineUniqueKeys[e.currentTable] == nil {
				e.inlineUniqueKeys[e.currentTable] = []string{}
			}
			e.inlineUniqueKeys[e.currentTable] = append(e.inlineUniqueKeys[e.currentTable], column.Name)
		}
	}

	// Handle CHECK constraint - check if the text contains CHECK keyword
	text := strings.ToUpper(getTextFromContext(ctx))
	if strings.Contains(text, "CHECK") {
		// Check constraints are parsed but not added to column comments
		// This is structural information handled elsewhere
		_ = text // Acknowledge we checked the text
	}

	// Handle REFERENCES (inline foreign key)
	if ctx.References_clause() != nil {
		// Don't add foreign key info to comment - this is structural information
		_ = ctx.References_clause() // Acknowledge the context
	}
}

// extractIdentityClause extracts identity column information
func (*metadataExtractor) extractIdentityClause(ctx parser.IIdentity_clauseContext, column *storepb.ColumnMetadata) {
	if ctx == nil {
		return
	}

	// Mark as identity column
	if ctx.ALWAYS() != nil {
		column.IsIdentity = true
		// TODO: Set identity generation type when proto is updated
	} else if ctx.BY() != nil && ctx.DEFAULT() != nil {
		column.IsIdentity = true
		// TODO: Set identity generation type when proto is updated
	}
	// Don't add identity info to comment - this is structural information

	// Identity columns are NOT NULL by default
	column.Nullable = false
}

// extractOutOfLineConstraint extracts out-of-line table constraints
func (e *metadataExtractor) extractOutOfLineConstraint(ctx parser.IOut_of_line_constraintContext, table *storepb.TableMetadata) {
	if ctx == nil {
		return
	}

	// Extract constraint name
	constraintName := ""
	if ctx.Constraint_name() != nil {
		constraintName = normalizeConstraintName(ctx.Constraint_name())
	}

	// Handle different constraint types
	switch {
	case ctx.PRIMARY() != nil && ctx.KEY() != nil:
		e.extractPrimaryKeyConstraint(ctx, table, constraintName)
	case ctx.UNIQUE() != nil:
		e.extractUniqueConstraint(ctx, table, constraintName)
	case ctx.CHECK() != nil:
		e.extractCheckConstraint(ctx, table, constraintName)
	default:
		// Other constraint types
	}
}

// extractOutOfLineRefConstraint extracts out-of-line foreign key constraints
func (e *metadataExtractor) extractOutOfLineRefConstraint(ctx parser.IOut_of_line_ref_constraintContext, table *storepb.TableMetadata) {
	if ctx == nil {
		return
	}

	// Extract constraint name
	constraintName := ""
	if ctx.Constraint_name() != nil {
		constraintName = normalizeConstraintName(ctx.Constraint_name())
	}

	if ctx.FOREIGN() != nil && ctx.KEY() != nil {
		e.extractForeignKeyConstraint(ctx, table, constraintName)
	}
}

// extractPrimaryKeyConstraint extracts primary key constraint
func (*metadataExtractor) extractPrimaryKeyConstraint(ctx parser.IOut_of_line_constraintContext, table *storepb.TableMetadata, constraintName string) {
	if constraintName == "" {
		constraintName = fmt.Sprintf("PK_%s", table.Name)
	}

	// Extract column names
	var columns []string
	for _, colName := range ctx.AllColumn_name() {
		if normalized := normalizeColumnName(colName); normalized != "" {
			columns = append(columns, normalized)
		}
	}

	if len(columns) == 0 {
		return
	}

	// Create primary key index
	index := &storepb.IndexMetadata{
		Name:         constraintName,
		Primary:      true,
		Unique:       true,
		Type:         "NORMAL",
		Expressions:  columns,
		Visible:      true, // Oracle indexes are visible by default
		IsConstraint: true, // This represents a primary key constraint
	}

	table.Indexes = append(table.Indexes, index)

	// Mark columns as NOT NULL
	for _, col := range table.Columns {
		for _, pkCol := range columns {
			if col.Name == pkCol {
				col.Nullable = false
				break
			}
		}
	}
}

// extractUniqueConstraint extracts unique constraint
func (*metadataExtractor) extractUniqueConstraint(ctx parser.IOut_of_line_constraintContext, table *storepb.TableMetadata, constraintName string) {
	if constraintName == "" {
		constraintName = fmt.Sprintf("UK_%s_%d", table.Name, len(table.Indexes)+1)
	}

	// Extract column names
	var columns []string
	for _, colName := range ctx.AllColumn_name() {
		if normalized := normalizeColumnName(colName); normalized != "" {
			columns = append(columns, normalized)
		}
	}

	if len(columns) == 0 {
		return
	}

	// Create unique index
	index := &storepb.IndexMetadata{
		Name:         constraintName,
		Primary:      false,
		Unique:       true,
		Type:         "NORMAL",
		Expressions:  columns,
		Visible:      false, // Oracle constraint-based unique indexes are not visible
		IsConstraint: true,  // This represents a unique constraint
	}

	table.Indexes = append(table.Indexes, index)
}

// extractCheckConstraint extracts check constraint using ANTLR parser
func (*metadataExtractor) extractCheckConstraint(ctx parser.IOut_of_line_constraintContext, table *storepb.TableMetadata, constraintName string) {
	if constraintName == "" {
		constraintName = fmt.Sprintf("CHK_%s_%d", table.Name, len(table.CheckConstraints)+1)
	}

	// Extract check condition using ANTLR parser
	if ctx.CHECK() != nil {
		if condition := ctx.Condition(); condition != nil {
			check := &storepb.CheckConstraintMetadata{
				Name:       constraintName,
				Expression: getTextFromContext(condition),
			}
			table.CheckConstraints = append(table.CheckConstraints, check)
		}
	}
}

// extractForeignKeyConstraint extracts foreign key constraint
func (e *metadataExtractor) extractForeignKeyConstraint(ctx parser.IOut_of_line_ref_constraintContext, table *storepb.TableMetadata, constraintName string) {
	if constraintName == "" {
		constraintName = fmt.Sprintf("FK_%s_%d", table.Name, len(table.ForeignKeys)+1)
	}

	// Extract local columns from the context
	var columns []string
	// The foreign key columns are usually specified after FOREIGN KEY
	text := getTextFromContext(ctx)
	if fkIdx := strings.Index(strings.ToUpper(text), "FOREIGNKEY"); fkIdx != -1 {
		// Find the column list in parentheses
		if openIdx := strings.Index(text[fkIdx:], "("); openIdx != -1 {
			if closeIdx := strings.Index(text[fkIdx+openIdx:], ")"); closeIdx != -1 {
				colList := text[fkIdx+openIdx+1 : fkIdx+openIdx+closeIdx]
				for _, col := range strings.Split(colList, ",") {
					col = strings.TrimSpace(col)
					col = strings.Trim(col, "\"")
					if col != "" {
						columns = append(columns, strings.ToUpper(col))
					}
				}
			}
		}
	}

	if len(columns) == 0 || ctx.References_clause() == nil {
		return
	}

	// Extract referenced table and columns
	referencedTable, referencedColumns := e.extractReferencesClause(ctx.References_clause())
	if referencedTable == "" || len(referencedColumns) == 0 {
		return
	}

	foreignKey := &storepb.ForeignKeyMetadata{
		Name:              constraintName,
		Columns:           columns,
		ReferencedTable:   referencedTable,
		ReferencedColumns: referencedColumns,
	}

	// Extract ON DELETE action from the text
	text = strings.ToUpper(getTextFromContext(ctx))
	if strings.Contains(text, "ONDELETECASCADE") {
		foreignKey.OnDelete = "CASCADE"
	} else if strings.Contains(text, "ONDELETESETNULL") {
		foreignKey.OnDelete = "SET NULL"
	}

	table.ForeignKeys = append(table.ForeignKeys, foreignKey)
}

// extractReferencesClause extracts referenced table and columns from references clause
func (*metadataExtractor) extractReferencesClause(ctx parser.IReferences_clauseContext) (string, []string) {
	if ctx == nil || ctx.Tableview_name() == nil {
		return "", nil
	}

	// Extract referenced table name
	referencedTable := normalizeTableViewName(ctx.Tableview_name())

	// Extract referenced columns from parentheses after table name
	var referencedColumns []string
	text := getTextFromContext(ctx)
	tableEnd := strings.Index(text, referencedTable) + len(referencedTable)
	if tableEnd < len(text) {
		remaining := text[tableEnd:]
		if openIdx := strings.Index(remaining, "("); openIdx != -1 {
			if closeIdx := strings.Index(remaining[openIdx:], ")"); closeIdx != -1 {
				colList := remaining[openIdx+1 : openIdx+closeIdx]
				for _, col := range strings.Split(colList, ",") {
					col = strings.TrimSpace(col)
					col = strings.Trim(col, "\"")
					if col != "" {
						referencedColumns = append(referencedColumns, strings.ToUpper(col))
					}
				}
			}
		}
	}

	// If no columns specified, default to primary key columns (Oracle behavior)
	if len(referencedColumns) == 0 {
		referencedColumns = []string{"ID"} // Default assumption
	}

	return referencedTable, referencedColumns
}

// normalizeColumnName extracts and normalizes column name
func normalizeColumnName(ctx parser.IColumn_nameContext) string {
	if ctx == nil {
		return ""
	}

	// Extract all parts of the column name
	var parts []string
	if ctx.Identifier() != nil {
		parts = append(parts, oracleparser.NormalizeIdentifierContext(ctx.Identifier()))
	}
	for _, idExpr := range ctx.AllId_expression() {
		parts = append(parts, oracleparser.NormalizeIDExpression(idExpr))
	}

	// Return the last part as the column name
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// normalizeConstraintName extracts and normalizes constraint name
func normalizeConstraintName(ctx parser.IConstraint_nameContext) string {
	if ctx == nil {
		return ""
	}

	// Extract all parts of the constraint name
	var parts []string
	if ctx.Identifier() != nil {
		parts = append(parts, oracleparser.NormalizeIdentifierContext(ctx.Identifier()))
	}
	for _, idExpr := range ctx.AllId_expression() {
		parts = append(parts, oracleparser.NormalizeIDExpression(idExpr))
	}

	// Return the full constraint name
	return strings.Join(parts, ".")
}

// normalizeTableViewName extracts and normalizes table/view name
func normalizeTableViewName(ctx parser.ITableview_nameContext) string {
	if ctx == nil {
		return ""
	}

	// Extract the table/view name parts
	var parts []string
	if ctx.Identifier() != nil {
		parts = append(parts, oracleparser.NormalizeIdentifierContext(ctx.Identifier()))
	}
	if ctx.Id_expression() != nil {
		parts = append(parts, oracleparser.NormalizeIDExpression(ctx.Id_expression()))
	}

	// Return the last part as the table/view name
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// normalizeSequenceName extracts and normalizes sequence name
func normalizeSequenceName(ctx parser.ISequence_nameContext) string {
	if ctx == nil {
		return ""
	}

	// Extract all parts of the sequence name
	var parts []string
	for _, idExpr := range ctx.AllId_expression() {
		parts = append(parts, oracleparser.NormalizeIDExpression(idExpr))
	}

	// Return the last part as the sequence name
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// normalizeFunctionName extracts and normalizes function name
func normalizeFunctionName(ctx parser.IFunction_nameContext) string {
	if ctx == nil {
		return ""
	}

	// Function name is a simple identifier or schema.function format
	// Get the text and parse it
	text := getTextFromContext(ctx)
	parts := strings.Split(text, ".")
	for i, part := range parts {
		parts[i] = strings.ToUpper(strings.Trim(part, "\""))
	}

	// Return the last part as the function name
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// normalizeProcedureName extracts and normalizes procedure name
func normalizeProcedureName(ctx parser.IProcedure_nameContext) string {
	if ctx == nil {
		return ""
	}

	// Procedure name is a simple identifier or schema.procedure format
	// Get the text and parse it
	text := getTextFromContext(ctx)
	parts := strings.Split(text, ".")
	for i, part := range parts {
		parts[i] = strings.ToUpper(strings.Trim(part, "\""))
	}

	// Return the last part as the procedure name
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// EnterComment_on_table is called when entering a comment on table statement
func (e *metadataExtractor) EnterComment_on_table(ctx *parser.Comment_on_tableContext) {
	if e.err != nil {
		return
	}

	if ctx.Tableview_name() == nil {
		return
	}

	// Extract table name
	tableName := normalizeTableViewName(ctx.Tableview_name())
	if tableName == "" {
		return
	}

	// Extract comment text
	comment := ""
	if ctx.Quoted_string() != nil {
		comment = oracleparser.NormalizeQuotedString(ctx.Quoted_string())
	}

	if comment != "" {
		if table, exists := e.tables[tableName]; exists {
			table.Comment = comment
		}
	}
}

// EnterComment_on_column is called when entering a comment on column statement
func (e *metadataExtractor) EnterComment_on_column(ctx *parser.Comment_on_columnContext) {
	if e.err != nil {
		return
	}

	if ctx.Column_name() == nil {
		return
	}

	// Extract table and column names from the column_name context
	// Column name can be in format: table.column or just column
	parts := []string{}
	if ctx.Column_name().Identifier() != nil {
		parts = append(parts, oracleparser.NormalizeIdentifierContext(ctx.Column_name().Identifier()))
	}
	for _, idExpr := range ctx.Column_name().AllId_expression() {
		parts = append(parts, oracleparser.NormalizeIDExpression(idExpr))
	}

	var tableName, columnName string
	switch len(parts) {
	case 1:
		// Just column name, need to find the table
		// In this case, we might need to use the most recently created table
		// or handle it differently based on context
		return
	case 2:
		// table.column format
		tableName = parts[0]
		columnName = parts[1]
	case 3:
		// schema.table.column format
		tableName = parts[1]
		columnName = parts[2]
	default:
		return
	}

	// Extract comment text
	comment := ""
	if ctx.Quoted_string() != nil {
		comment = oracleparser.NormalizeQuotedString(ctx.Quoted_string())
	}

	if tableName != "" && columnName != "" && comment != "" {
		if table, exists := e.tables[tableName]; exists {
			for _, col := range table.Columns {
				if col.Name == columnName {
					// Set the actual user comment
					col.Comment = comment
					break
				}
			}
		}
	}
}

// normalizeDataTypeText normalizes Oracle data type text to match database output format
func normalizeDataTypeText(text string) string {
	if text == "" {
		return ""
	}

	// Convert to uppercase but preserve the precision/scale information
	normalized := strings.ToUpper(text)

	// Handle special cases for Oracle data types
	switch {
	case strings.HasPrefix(normalized, "VARCHAR2"):
		// Handle VARCHAR2 with BYTE/CHAR specifiers
		if strings.HasPrefix(normalized, "VARCHAR2(") && strings.HasSuffix(normalized, ")") {
			start := strings.Index(normalized, "(") + 1
			end := strings.LastIndex(normalized, ")")
			if start < end {
				sizeStr := normalized[start:end]
				if size := parseInt(sizeStr); size > 0 {
					// Oracle defaults to BYTE semantics, add BYTE specifier
					return fmt.Sprintf("VARCHAR2(%d BYTE)", size)
				}
			}
		}
		return normalized
	case strings.HasPrefix(normalized, "CHAR("):
		// Handle CHAR with BYTE/CHAR specifiers
		if strings.HasSuffix(normalized, ")") {
			start := strings.Index(normalized, "(") + 1
			end := strings.LastIndex(normalized, ")")
			if start < end {
				sizeStr := normalized[start:end]
				if size := parseInt(sizeStr); size > 0 {
					// Oracle defaults to BYTE semantics, add BYTE specifier
					return fmt.Sprintf("CHAR(%d BYTE)", size)
				}
			}
		}
		return normalized
	case strings.HasPrefix(normalized, "TIMESTAMP"):
		// Handle TIMESTAMP variations and apply Oracle defaults
		// Check for both spaced and non-spaced formats
		if strings.Contains(normalized, "WITH LOCAL TIME ZONE") || strings.Contains(normalized, "WITHLOCALTIMEZONE") {
			// Handle TIMESTAMP WITH LOCAL TIME ZONE
			if normalized == "TIMESTAMP WITH LOCAL TIME ZONE" {
				return "TIMESTAMP(6) WITH LOCAL TIME ZONE"
			}
			// If it already has precision, keep proper spacing
			if strings.Contains(normalized, "(") && strings.Contains(normalized, ")") {
				// Ensure proper spacing
				result := strings.ReplaceAll(normalized, "WITHLOCALTIMEZONE", " WITH LOCAL TIME ZONE")
				return result
			}
			return "TIMESTAMP(6) WITH LOCAL TIME ZONE"
		} else if strings.Contains(normalized, "WITH TIME ZONE") || strings.Contains(normalized, "WITHTIMEZONE") {
			// Handle TIMESTAMP WITH TIME ZONE
			if normalized == "TIMESTAMP WITH TIME ZONE" {
				return "TIMESTAMP(6) WITH TIME ZONE"
			}
			// If it already has precision, keep proper spacing
			if strings.Contains(normalized, "(") && strings.Contains(normalized, ")") {
				// Ensure proper spacing
				result := strings.ReplaceAll(normalized, "WITHTIMEZONE", " WITH TIME ZONE")
				return result
			}
			return "TIMESTAMP(6) WITH TIME ZONE"
		} else if normalized == "TIMESTAMP" {
			// Plain TIMESTAMP gets default precision
			return "TIMESTAMP(6)"
		}
		return normalized
	case strings.HasPrefix(normalized, "INTERVAL"):
		// Handle INTERVAL types with proper spacing and precision
		// Check for both spaced and non-spaced formats
		if strings.Contains(normalized, "YEAR TO MONTH") || strings.Contains(normalized, "YEARTOMONTH") {
			// Handle INTERVAL YEAR TO MONTH
			if normalized == "INTERVAL YEAR TO MONTH" {
				return "INTERVAL YEAR(2) TO MONTH"
			}
			// If it already has precision, keep proper spacing
			if strings.Contains(normalized, "(") && strings.Contains(normalized, ")") {
				// Ensure proper spacing
				result := strings.ReplaceAll(normalized, "YEARTOMONTH", " YEAR TO MONTH")
				result = strings.ReplaceAll(result, "INTERVALYEAR", "INTERVAL YEAR")
				return result
			}
			return "INTERVAL YEAR(2) TO MONTH"
		} else if strings.Contains(normalized, "DAY TO SECOND") || strings.Contains(normalized, "DAYTOSECOND") {
			// Handle INTERVAL DAY TO SECOND
			if normalized == "INTERVAL DAY TO SECOND" {
				return "INTERVAL DAY(2) TO SECOND(6)"
			}
			// If it already has precision, keep proper spacing
			if strings.Contains(normalized, "(") && strings.Contains(normalized, ")") {
				// Ensure proper spacing
				result := strings.ReplaceAll(normalized, "DAYTOSECOND", " DAY TO SECOND")
				result = strings.ReplaceAll(result, "INTERVALDAY", "INTERVAL DAY")
				return result
			}
			return "INTERVAL DAY(2) TO SECOND(6)"
		}
		return normalized
	case strings.HasPrefix(normalized, "LONGRAW"):
		return "LONG RAW"
	case strings.HasPrefix(normalized, "DOUBLE"):
		return "DOUBLE PRECISION"
	case normalized == "UROWID":
		// Oracle's default UROWID size is 4000
		return "UROWID(4000)"
	case normalized == "FLOAT":
		// Oracle's default FLOAT precision is 126
		return "FLOAT(126)"
	case strings.HasPrefix(normalized, "NVARCHAR2"):
		// Handle Oracle NVARCHAR2 type expansion - Oracle expands sizes based on character set
		// For AL16UTF16: NVARCHAR2(2000) becomes NVARCHAR2(4000) in the database
		// We need to expand parsed DDL sizes to match what Oracle stores
		if strings.HasPrefix(normalized, "NVARCHAR2(") && strings.HasSuffix(normalized, ")") {
			start := strings.Index(normalized, "(") + 1
			end := strings.LastIndex(normalized, ")")
			if start < end {
				sizeStr := normalized[start:end]
				if size := parseInt(sizeStr); size > 0 {
					// Expand to match Oracle's internal storage
					// Oracle doubles the size for AL16UTF16 character set
					if size == 2000 {
						return "NVARCHAR2(4000)"
					} else if size <= 2000 {
						return fmt.Sprintf("NVARCHAR2(%d)", size*2)
					}
				}
			}
		}
		return normalized
	case strings.HasPrefix(normalized, "NCHAR"):
		// Handle Oracle NCHAR type expansion - Oracle expands NCHAR sizes based on character set
		// AL16UTF16 uses 2 bytes per character, so NCHAR(100) becomes NCHAR(200) in the database
		// We need to expand parsed DDL sizes to match what Oracle stores
		if strings.HasPrefix(normalized, "NCHAR(") && strings.HasSuffix(normalized, ")") {
			start := strings.Index(normalized, "(") + 1
			end := strings.LastIndex(normalized, ")")
			if start < end {
				sizeStr := normalized[start:end]
				if size := parseInt(sizeStr); size > 0 {
					// Expand to match Oracle's internal storage (double for AL16UTF16)
					return fmt.Sprintf("NCHAR(%d)", size*2)
				}
			}
		}
		return normalized
	default:
		return normalized
	}
}

// parseInt helper function to parse integer from string
func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// getTextFromContext gets text from ANTLR context preserving original spacing
func getTextFromContext(ctx any) string {
	// Try to get parser-aware context first
	if parserCtx, ok := ctx.(interface {
		GetParser() antlr.Parser
	}); ok {
		parser := parserCtx.GetParser()
		if parser != nil {
			if ruleCtx, ok := ctx.(antlr.RuleContext); ok {
				return parser.GetTokenStream().GetTextFromRuleContext(ruleCtx)
			}
		}
	}

	// Fallback to GetText() for other contexts
	if textCtx, ok := ctx.(interface{ GetText() string }); ok {
		return textCtx.GetText()
	}

	return ""
}

// processInlineConstraints creates indexes for inline PRIMARY KEY and UNIQUE constraints
// This simulates Oracle's automatic index creation behavior
func (e *metadataExtractor) processInlineConstraints(tableName string, table *storepb.TableMetadata) {
	// Process inline primary key constraints
	if primaryKeyColumns := e.inlinePrimaryKeys[tableName]; len(primaryKeyColumns) > 0 {
		constraintName := fmt.Sprintf("PK_%s", tableName)

		// Check if a primary key index already exists (from out-of-line constraints)
		hasExistingPK := false
		for _, idx := range table.Indexes {
			if idx.Primary {
				hasExistingPK = true
				break
			}
		}

		// Oracle automatically creates a unique index for PRIMARY KEY constraints
		if !hasExistingPK {
			index := &storepb.IndexMetadata{
				Name:         constraintName,
				Primary:      true,
				Unique:       true,
				Type:         "NORMAL",
				Expressions:  primaryKeyColumns,
				Visible:      true, // Oracle indexes are visible by default
				IsConstraint: true, // This represents a primary key constraint
			}
			table.Indexes = append(table.Indexes, index)
		}

		// Clean up the tracking
		delete(e.inlinePrimaryKeys, tableName)
	}

	// Process inline unique constraints
	if uniqueKeyColumns := e.inlineUniqueKeys[tableName]; len(uniqueKeyColumns) > 0 {
		for _, columnName := range uniqueKeyColumns {
			constraintName := fmt.Sprintf("UK_%s_%s", tableName, columnName)

			// Check if a unique index already exists for this column
			hasExistingUnique := false
			for _, idx := range table.Indexes {
				if idx.Unique && !idx.Primary && len(idx.Expressions) == 1 && idx.Expressions[0] == columnName {
					hasExistingUnique = true
					break
				}
			}

			// Oracle automatically creates a unique index for UNIQUE constraints
			if !hasExistingUnique {
				index := &storepb.IndexMetadata{
					Name:         constraintName,
					Primary:      false,
					Unique:       true,
					Type:         "NORMAL",
					Expressions:  []string{columnName},
					Visible:      true, // Oracle indexes are visible by default
					IsConstraint: true, // This represents a unique constraint
				}
				table.Indexes = append(table.Indexes, index)
			}
		}

		// Clean up the tracking
		delete(e.inlineUniqueKeys, tableName)
	}
}
