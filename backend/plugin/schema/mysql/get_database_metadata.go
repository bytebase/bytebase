package mysql

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_MYSQL, GetDatabaseMetadata)
}

// GetDatabaseMetadata parses the MySQL schema text and returns the database metadata.
func GetDatabaseMetadata(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	parseResult, err := mysqlparser.ParseMySQL(schemaText)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse MySQL schema")
	}

	extractor := &metadataExtractor{
		currentDatabase: "",
		currentSchema:   "",
		tables:          make(map[string]*storepb.TableMetadata),
		views:           make(map[string]*storepb.ViewMetadata),
		functions:       make(map[string]*storepb.FunctionMetadata),
		procedures:      make(map[string]*storepb.ProcedureMetadata),
		triggers:        make(map[string]*storepb.TriggerMetadata),
	}

	// Walk each parse tree
	for _, result := range parseResult {
		if result.Tree != nil {
			antlr.ParseTreeWalkerDefault.Walk(extractor, result.Tree)
		}
	}

	if extractor.err != nil {
		return nil, extractor.err
	}

	// Build the final metadata structure
	schemaMetadata := &storepb.DatabaseSchemaMetadata{
		Name: extractor.currentDatabase,
	}

	// MySQL doesn't have schemas in the same way as PostgreSQL
	// All objects are in a single schema
	schema := &storepb.SchemaMetadata{
		Name: "",
	}

	// Sort and add tables
	var tableNames []string
	for name := range extractor.tables {
		tableNames = append(tableNames, name)
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

	schemaMetadata.Schemas = []*storepb.SchemaMetadata{schema}

	return schemaMetadata, nil
}

// metadataExtractor walks the parse tree and extracts metadata
type metadataExtractor struct {
	*mysql.BaseMySQLParserListener

	currentDatabase string
	currentSchema   string
	tables          map[string]*storepb.TableMetadata
	views           map[string]*storepb.ViewMetadata
	functions       map[string]*storepb.FunctionMetadata
	procedures      map[string]*storepb.ProcedureMetadata
	triggers        map[string]*storepb.TriggerMetadata
	err             error
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

// EnterCreateDatabase is called when entering a CREATE DATABASE statement
func (e *metadataExtractor) EnterCreateDatabase(ctx *mysql.CreateDatabaseContext) {
	if e.err != nil {
		return
	}

	if ctx.SchemaName() != nil {
		e.currentDatabase = mysqlparser.NormalizeMySQLSchemaName(ctx.SchemaName())
	}
}

// EnterCreateTable is called when entering a CREATE TABLE statement
func (e *metadataExtractor) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if e.err != nil {
		return
	}

	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	table := e.getOrCreateTable(tableName)

	// Extract table elements
	if ctx.TableElementList() != nil {
		e.extractTableElements(ctx.TableElementList(), table)
	}

	// Extract partitioning
	if ctx.PartitionClause() != nil {
		e.extractPartitions(ctx.PartitionClause(), table)
	}

	// Extract table comment
	e.extractTableComment(ctx, table)
}

// extractTableElements extracts columns and constraints from table elements
func (e *metadataExtractor) extractTableElements(ctx mysql.ITableElementListContext, table *storepb.TableMetadata) {
	if ctx == nil {
		return
	}

	for _, element := range ctx.AllTableElement() {
		if element == nil {
			continue
		}

		// Handle column definitions
		if columnDef := element.ColumnDefinition(); columnDef != nil {
			e.extractColumnDefinition(columnDef, table)
		}

		// Handle table constraints
		if constraint := element.TableConstraintDef(); constraint != nil {
			e.extractTableConstraint(constraint, table)
		}
	}
}

// extractColumnDefinition extracts column metadata
func (e *metadataExtractor) extractColumnDefinition(ctx mysql.IColumnDefinitionContext, table *storepb.TableMetadata) {
	if ctx == nil || ctx.FieldDefinition() == nil {
		return
	}

	if ctx.ColumnName() == nil {
		return
	}

	_, _, columnName := mysqlparser.NormalizeMySQLColumnName(ctx.ColumnName())
	column := &storepb.ColumnMetadata{
		Name:     columnName,
		Position: int32(len(table.Columns) + 1),
	}

	// Extract data type
	if ctx.FieldDefinition() != nil && ctx.FieldDefinition().DataType() != nil {
		column.Type = e.extractDataType(ctx.FieldDefinition().DataType())
	}

	// Extract column attributes (NULL/NOT NULL, DEFAULT, etc.)
	if ctx.FieldDefinition() != nil {
		e.extractFieldAttributes(ctx.FieldDefinition(), column)
	}

	// Extract comment
	if ctx.FieldDefinition() != nil {
		if comment := e.extractColumnComment(ctx.FieldDefinition()); comment != "" {
			column.Comment = comment
		}
	}

	table.Columns = append(table.Columns, column)
}

// extractDataType extracts the data type as a string
func (*metadataExtractor) extractDataType(ctx mysql.IDataTypeContext) string {
	if ctx == nil {
		return ""
	}

	// Get the text representation of the data type and normalize to lowercase
	dataType := strings.ToLower(ctx.GetText())

	// MySQL normalizations: BOOLEAN is stored as TINYINT(1)
	if dataType == "boolean" || dataType == "bool" {
		dataType = "tinyint(1)"
	}

	// MySQL spatial type normalizations
	if dataType == "geometrycollection" {
		dataType = "geomcollection"
	}

	// Handle UNSIGNED and ZEROFILL attributes that are part of the data type
	// MySQL parser concatenates these without spaces, so we need to add them back
	dataType = normalizeDataTypeAttributes(dataType)

	return dataType
}

// normalizeDataTypeAttributes adds proper spacing for MySQL data type attributes
func normalizeDataTypeAttributes(dataType string) string {
	// Handle combined UNSIGNED ZEROFILL first
	if strings.Contains(dataType, "unsignedzerofill") {
		dataType = strings.ReplaceAll(dataType, "unsignedzerofill", "unsigned zerofill")
	}

	// Handle UNSIGNED attribute
	if strings.Contains(dataType, "unsigned") {
		// Replace patterns like "intunsigned" with "int unsigned"
		// Find where "unsigned" starts
		if idx := strings.Index(dataType, "unsigned"); idx > 0 {
			// Check if there's a character before "unsigned" that's not a space
			if dataType[idx-1] != ' ' {
				// Insert space before "unsigned"
				dataType = dataType[:idx] + " " + dataType[idx:]
			}
		}
	}

	// Handle BINARY attribute
	if strings.Contains(dataType, "binary") && !strings.Contains(dataType, "varbinary") {
		// Replace patterns like "char(10)binary" with "char(10)"
		// MySQL stores BINARY as part of the collation, not the data type
		if idx := strings.Index(dataType, "binary"); idx > 0 {
			if dataType[idx-1] != ' ' && dataType[idx-1] != ')' {
				// Insert space before "binary"
				dataType = dataType[:idx] + " " + dataType[idx:]
			} else if dataType[idx-1] == ')' {
				// For patterns like "char(10)binary", MySQL normalizes to just "char(10)"
				dataType = dataType[:idx]
			}
		}
	}

	// Handle ZEROFILL attribute
	if strings.Contains(dataType, "zerofill") {
		// Replace patterns like "intzerofill" with "int zerofill"
		if idx := strings.Index(dataType, "zerofill"); idx > 0 {
			if dataType[idx-1] != ' ' {
				dataType = dataType[:idx] + " " + dataType[idx:]
			}
		}

		// MySQL implicitly adds UNSIGNED when ZEROFILL is specified
		// If we have zerofill but not unsigned, add unsigned
		if strings.Contains(dataType, "zerofill") && !strings.Contains(dataType, "unsigned") {
			// Insert "unsigned " before "zerofill"
			dataType = strings.ReplaceAll(dataType, "zerofill", "unsigned zerofill")
		}

		// MySQL adds default display width for INT UNSIGNED ZEROFILL when none specified
		if strings.HasPrefix(dataType, "int ") && strings.Contains(dataType, "unsigned zerofill") && !strings.Contains(dataType, "(") {
			dataType = strings.ReplaceAll(dataType, "int ", "int(10) ")
		}
	}

	return dataType
}

// extractFieldAttributes extracts field attributes like NULL/NOT NULL, DEFAULT, AUTO_INCREMENT, PRIMARY KEY
func (*metadataExtractor) extractFieldAttributes(ctx mysql.IFieldDefinitionContext, column *storepb.ColumnMetadata) {
	if ctx == nil {
		return
	}

	// Default to nullable
	column.Nullable = true
	hasExplicitDefault := false

	// Check for GENERATED columns
	// For generated columns, MySQL stores them as having NULL default, not the generation expression
	if ctx.GENERATED_SYMBOL() != nil && ctx.ALWAYS_SYMBOL() != nil && ctx.AS_SYMBOL() != nil && ctx.ExprWithParentheses() != nil {
		// Generated columns have no default value in the traditional sense
		column.DefaultNull = true
		hasExplicitDefault = true
	}

	for _, attr := range ctx.AllColumnAttribute() {
		if attr == nil {
			continue
		}

		// Check for PRIMARY KEY - inline definition makes column NOT NULL
		if attr.PRIMARY_SYMBOL() != nil && attr.KEY_SYMBOL() != nil {
			column.Nullable = false
		}

		// Check for NULL/NOT NULL
		if attr.NullLiteral() != nil {
			if attr.NOT_SYMBOL() != nil {
				column.Nullable = false
			} else {
				column.Nullable = true
			}
		}

		// Check for DEFAULT value
		if attr.DEFAULT_SYMBOL() != nil {
			hasExplicitDefault = true
			if attr.SignedLiteral() != nil {
				defaultValue := mysqlparser.NormalizeMySQLSignedLiteral(attr.SignedLiteral())
				column.DefaultExpression = normalizeDefaultValue(defaultValue)
			} else if attr.ExprWithParentheses() != nil {
				column.DefaultExpression = attr.ExprWithParentheses().GetText()
			} else {
				// Check for special keywords like CURRENT_TIMESTAMP
				// Parse the entire attribute text to find DEFAULT keyword and what follows
				attrText := attr.GetText()
				attrTextUpper := strings.ToUpper(attrText)

				defaultIdx := strings.Index(attrTextUpper, "DEFAULT")
				if defaultIdx >= 0 && len(attrTextUpper) > defaultIdx+7 {
					remaining := attrTextUpper[defaultIdx+7:]
					if strings.HasPrefix(remaining, "CURRENT_TIMESTAMP") || strings.HasPrefix(remaining, "NOW()") {
						column.DefaultExpression = "CURRENT_TIMESTAMP"
					}
				}
			}
		}

		// Check for AUTO_INCREMENT
		if attr.AUTO_INCREMENT_SYMBOL() != nil {
			hasExplicitDefault = true
			if column.DefaultExpression == "" {
				column.DefaultExpression = "AUTO_INCREMENT"
			}
		}
	}

	// If column is nullable and has no explicit default, set DefaultNull to true
	if column.Nullable && !hasExplicitDefault && column.DefaultExpression == "" {
		column.DefaultNull = true
	}
}

// extractColumnComment extracts column comment
func (*metadataExtractor) extractColumnComment(ctx mysql.IFieldDefinitionContext) string {
	if ctx == nil {
		return ""
	}

	for _, attr := range ctx.AllColumnAttribute() {
		if attr == nil || attr.COMMENT_SYMBOL() == nil {
			continue
		}

		if attr.TextLiteral() != nil {
			return mysqlparser.NormalizeMySQLTextLiteral(attr.TextLiteral())
		}
	}

	return ""
}

// extractTableComment extracts table comment
func (*metadataExtractor) extractTableComment(ctx *mysql.CreateTableContext, table *storepb.TableMetadata) {
	if ctx == nil || ctx.CreateTableOptions() == nil {
		return
	}

	// Look for COMMENT option in the CREATE TABLE statement
	for _, option := range ctx.CreateTableOptions().AllCreateTableOption() {
		if option == nil {
			continue
		}

		if option.COMMENT_SYMBOL() != nil && option.TextStringLiteral() != nil {
			comment := mysqlparser.NormalizeMySQLTextStringLiteral(option.TextStringLiteral())
			table.Comment = comment
			break
		}
	}
}

// extractTableConstraint extracts table-level constraints
func (e *metadataExtractor) extractTableConstraint(ctx mysql.ITableConstraintDefContext, table *storepb.TableMetadata) {
	if ctx == nil {
		return
	}

	// Extract PRIMARY KEY
	if ctx.GetType_() != nil && ctx.GetType_().GetTokenType() == mysql.MySQLParserPRIMARY_SYMBOL {
		e.extractPrimaryKey(ctx, table)
	}

	// Extract FOREIGN KEY
	if ctx.FOREIGN_SYMBOL() != nil && ctx.KEY_SYMBOL() != nil {
		e.extractForeignKey(ctx, table)
	}

	// Extract INDEX/KEY
	if (ctx.INDEX_SYMBOL() != nil || ctx.KEY_SYMBOL() != nil) && ctx.FOREIGN_SYMBOL() == nil && ctx.PRIMARY_SYMBOL() == nil {
		e.extractIndex(ctx, table)
	}

	// Extract UNIQUE constraint
	if ctx.UNIQUE_SYMBOL() != nil {
		e.extractUniqueIndex(ctx, table)
	}

	// Extract CHECK constraint
	if ctx.CheckConstraint() != nil {
		// Extract constraint name if present
		constraintName := ""
		if ctx.ConstraintName() != nil {
			constraintName = mysqlparser.NormalizeConstraintName(ctx.ConstraintName())
		}
		e.extractCheckConstraint(ctx.CheckConstraint(), table, constraintName)
	}
}

// extractPrimaryKey extracts primary key constraint
func (*metadataExtractor) extractPrimaryKey(ctx mysql.ITableConstraintDefContext, table *storepb.TableMetadata) {
	if ctx.KeyListVariants() == nil {
		return
	}

	keyColumns := mysqlparser.NormalizeKeyListVariants(ctx.KeyListVariants())

	if len(keyColumns) > 0 {
		// Mark primary key columns as NOT NULL
		for _, column := range table.Columns {
			for _, pkCol := range keyColumns {
				if column.Name == pkCol {
					column.Nullable = false
					break
				}
			}
		}

		index := &storepb.IndexMetadata{
			Name:        "PRIMARY",
			Type:        "PRIMARY",
			Expressions: keyColumns,
			Primary:     true,
			Unique:      true,
		}
		table.Indexes = append(table.Indexes, index)
	}
}

// extractForeignKey extracts foreign key constraint
func (*metadataExtractor) extractForeignKey(ctx mysql.ITableConstraintDefContext, table *storepb.TableMetadata) {
	if ctx.KeyList() == nil || ctx.References() == nil {
		return
	}

	// Extract constraint name
	constraintName := ""
	if ctx.ConstraintName() != nil {
		constraintName = mysqlparser.NormalizeConstraintName(ctx.ConstraintName())
	}

	// Extract columns
	var columns []string
	for _, keyPart := range ctx.KeyList().AllKeyPart() {
		if keyPart.Identifier() != nil {
			columns = append(columns, mysqlparser.NormalizeMySQLIdentifier(keyPart.Identifier()))
		}
	}

	// Extract referenced table and columns
	references := ctx.References()
	if references.TableRef() == nil {
		return
	}

	_, referencedTable := mysqlparser.NormalizeMySQLTableRef(references.TableRef())

	var referencedColumns []string
	if references.IdentifierListWithParentheses() != nil {
		referencedColumns = mysqlparser.NormalizeIdentifierListWithParentheses(references.IdentifierListWithParentheses())
	}

	// Extract ON DELETE and ON UPDATE actions
	// MySQL defaults to "NO ACTION" if not specified
	onDelete := "NO ACTION"
	onUpdate := "NO ACTION"

	// Parse the full text of the references clause to find ON DELETE/UPDATE actions
	if references != nil {
		refText := references.GetText()
		refTextUpper := strings.ToUpper(refText)

		// Extract ON DELETE action
		onDeleteIdx := strings.Index(refTextUpper, "ONDELETE")
		if onDeleteIdx >= 0 {
			remaining := refTextUpper[onDeleteIdx+8:] // Skip "ONDELETE"
			if strings.HasPrefix(remaining, "CASCADE") {
				onDelete = "CASCADE"
			} else if strings.HasPrefix(remaining, "SETNULL") {
				onDelete = "SET NULL"
			} else if strings.HasPrefix(remaining, "RESTRICT") {
				onDelete = "RESTRICT"
			} else if strings.HasPrefix(remaining, "NOACTION") {
				onDelete = "NO ACTION"
			}
		}

		// Extract ON UPDATE action
		onUpdateIdx := strings.Index(refTextUpper, "ONUPDATE")
		if onUpdateIdx >= 0 {
			remaining := refTextUpper[onUpdateIdx+8:] // Skip "ONUPDATE"
			if strings.HasPrefix(remaining, "CASCADE") {
				onUpdate = "CASCADE"
			} else if strings.HasPrefix(remaining, "SETNULL") {
				onUpdate = "SET NULL"
			} else if strings.HasPrefix(remaining, "RESTRICT") {
				onUpdate = "RESTRICT"
			} else if strings.HasPrefix(remaining, "NOACTION") {
				onUpdate = "NO ACTION"
			}
		}
	}

	fk := &storepb.ForeignKeyMetadata{
		Name:              constraintName,
		Columns:           columns,
		ReferencedTable:   referencedTable,
		ReferencedColumns: referencedColumns,
		OnDelete:          onDelete,
		OnUpdate:          onUpdate,
	}

	table.ForeignKeys = append(table.ForeignKeys, fk)
}

// extractIndex extracts regular index
func (*metadataExtractor) extractIndex(ctx mysql.ITableConstraintDefContext, table *storepb.TableMetadata) {
	if ctx.KeyList() == nil {
		return
	}

	// Extract index name
	indexName := ""
	if ctx.IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexName())
	}

	// Extract columns/expressions
	var expressions []string
	for _, keyPart := range ctx.KeyList().AllKeyPart() {
		if keyPart.Identifier() != nil {
			columnName := mysqlparser.NormalizeMySQLIdentifier(keyPart.Identifier())
			// Check for order (ASC/DESC)
			if keyPart.Direction() != nil {
				if keyPart.Direction().DESC_SYMBOL() != nil {
					columnName += " DESC"
				}
			}
			expressions = append(expressions, columnName)
		}
	}

	if len(expressions) > 0 {
		indexType := "BTREE"

		index := &storepb.IndexMetadata{
			Name:        indexName,
			Type:        indexType,
			Expressions: expressions,
			Primary:     false,
			Unique:      false,
		}
		table.Indexes = append(table.Indexes, index)
	}
}

// extractUniqueIndex extracts unique index
func (*metadataExtractor) extractUniqueIndex(ctx mysql.ITableConstraintDefContext, table *storepb.TableMetadata) {
	if ctx.KeyList() == nil {
		return
	}

	// Extract index name
	indexName := ""
	if ctx.IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexName())
	}

	// Extract columns
	var expressions []string
	for _, keyPart := range ctx.KeyList().AllKeyPart() {
		if keyPart.Identifier() != nil {
			columnName := mysqlparser.NormalizeMySQLIdentifier(keyPart.Identifier())
			// Check for order (ASC/DESC)
			if keyPart.Direction() != nil {
				if keyPart.Direction().DESC_SYMBOL() != nil {
					columnName += " DESC"
				}
			}
			expressions = append(expressions, columnName)
		}
	}

	if len(expressions) > 0 {
		index := &storepb.IndexMetadata{
			Name:        indexName,
			Type:        "BTREE",
			Expressions: expressions,
			Primary:     false,
			Unique:      true,
		}
		table.Indexes = append(table.Indexes, index)
	}
}

// extractCheckConstraint extracts check constraint
func (*metadataExtractor) extractCheckConstraint(ctx mysql.ICheckConstraintContext, table *storepb.TableMetadata, constraintName string) {
	if ctx == nil || ctx.ExprWithParentheses() == nil {
		return
	}

	// Extract expression
	expression := ctx.ExprWithParentheses().GetText()

	// If no constraint name provided, generate one like MySQL does
	if constraintName == "" {
		constraintName = fmt.Sprintf("%s_chk_%d", table.Name, len(table.CheckConstraints)+1)
	}

	check := &storepb.CheckConstraintMetadata{
		Name:       constraintName,
		Expression: expression,
	}

	table.CheckConstraints = append(table.CheckConstraints, check)
}

// extractPartitions extracts partition information
func (*metadataExtractor) extractPartitions(ctx mysql.IPartitionClauseContext, table *storepb.TableMetadata) {
	if ctx == nil || ctx.PartitionTypeDef() == nil {
		return
	}

	// Extract partition expression based on partition type
	partitionExpr := ""
	partitionType := storepb.TablePartitionMetadata_TYPE_UNSPECIFIED

	// Determine partition type and extract expression
	typeDef := ctx.PartitionTypeDef()
	switch def := typeDef.(type) {
	case *mysql.PartitionDefKeyContext:
		// PARTITION BY KEY or LINEAR KEY
		if def.LINEAR_SYMBOL() != nil {
			partitionType = storepb.TablePartitionMetadata_LINEAR_KEY
		} else {
			partitionType = storepb.TablePartitionMetadata_KEY
		}
		if def.IdentifierList() != nil {
			columns := mysqlparser.NormalizeMySQLIdentifierList(def.IdentifierList())
			partitionExpr = strings.Join(columns, ",")
		}
	case *mysql.PartitionDefHashContext:
		// PARTITION BY HASH or LINEAR HASH
		if def.LINEAR_SYMBOL() != nil {
			partitionType = storepb.TablePartitionMetadata_LINEAR_HASH
		} else {
			partitionType = storepb.TablePartitionMetadata_HASH
		}
		if def.BitExpr() != nil {
			partitionExpr = def.BitExpr().GetText()
		}
	case *mysql.PartitionDefRangeListContext:
		// PARTITION BY RANGE/LIST [COLUMNS]
		if def.RANGE_SYMBOL() != nil {
			if def.COLUMNS_SYMBOL() != nil {
				partitionType = storepb.TablePartitionMetadata_RANGE_COLUMNS
			} else {
				partitionType = storepb.TablePartitionMetadata_RANGE
			}
		} else if def.LIST_SYMBOL() != nil {
			if def.COLUMNS_SYMBOL() != nil {
				partitionType = storepb.TablePartitionMetadata_LIST_COLUMNS
			} else {
				partitionType = storepb.TablePartitionMetadata_LIST
			}
		}

		// Extract expression
		if def.COLUMNS_SYMBOL() != nil && def.IdentifierList() != nil {
			columns := mysqlparser.NormalizeMySQLIdentifierList(def.IdentifierList())
			partitionExpr = strings.Join(columns, ",")
		} else if def.BitExpr() != nil {
			partitionExpr = def.BitExpr().GetText()
		}
	}

	// For KEY partitions without explicit columns, MySQL uses all primary key columns
	// We'll set the expression to empty string in this case

	// Extract default partition count for KEY/HASH partitions
	useDefault := ""
	if ctx.Real_ulong_number() != nil {
		// This indicates PARTITIONS n syntax
		useDefault = ctx.Real_ulong_number().GetText()
	}

	// Extract partition definitions
	if ctx.PartitionDefinitions() != nil {
		for _, partDef := range ctx.PartitionDefinitions().AllPartitionDefinition() {
			if partDef == nil || partDef.Identifier() == nil {
				continue
			}

			partition := &storepb.TablePartitionMetadata{
				Name:       mysqlparser.NormalizeMySQLIdentifier(partDef.Identifier()),
				Type:       partitionType,
				Expression: partitionExpr,
				UseDefault: useDefault,
			}

			// Extract partition value (for RANGE/LIST partitions)
			if partDef.PartitionValueItemListParen() != nil {
				// This contains VALUES LESS THAN or VALUES IN
				partition.Value = partDef.PartitionValueItemListParen().GetText()
			}

			table.Partitions = append(table.Partitions, partition)
		}
	} else if useDefault != "" {
		// For KEY/HASH partitions with only PARTITIONS n syntax
		// MySQL creates default partitions named p0, p1, p2, etc.
		// Parse the partition count and create the appropriate number of partitions
		if partitionCount, err := strconv.Atoi(useDefault); err == nil {
			for i := 0; i < partitionCount; i++ {
				partition := &storepb.TablePartitionMetadata{
					Name:       fmt.Sprintf("p%d", i),
					Type:       partitionType,
					Expression: partitionExpr,
					UseDefault: useDefault,
				}
				table.Partitions = append(table.Partitions, partition)
			}
		}
	}
}

// EnterCreateView is called when entering a CREATE VIEW statement
func (e *metadataExtractor) EnterCreateView(ctx *mysql.CreateViewContext) {
	if e.err != nil {
		return
	}

	if ctx.ViewName() == nil {
		return
	}

	_, viewName := mysqlparser.NormalizeMySQLViewName(ctx.ViewName())

	view := &storepb.ViewMetadata{
		Name: viewName,
	}

	// Extract view definition
	// For MySQL, the view definition is the entire CREATE VIEW statement
	if ctx.GetStart() != nil && ctx.GetStop() != nil {
		startIndex := ctx.GetStart().GetTokenIndex()
		stopIndex := ctx.GetStop().GetTokenIndex()
		tokens := ctx.GetParser().GetTokenStream()

		var definitionParts []string
		for i := startIndex; i <= stopIndex; i++ {
			token := tokens.Get(i)
			if token != nil {
				definitionParts = append(definitionParts, token.GetText())
			}
		}
		view.Definition = strings.Join(definitionParts, " ")
	}

	e.views[viewName] = view
}

// EnterCreateFunction is called when entering a CREATE FUNCTION statement
func (e *metadataExtractor) EnterCreateFunction(ctx *mysql.CreateFunctionContext) {
	if e.err != nil {
		return
	}

	if ctx.FunctionName() == nil {
		return
	}

	_, functionName := mysqlparser.NormalizeMySQLFunctionName(ctx.FunctionName())

	function := &storepb.FunctionMetadata{
		Name: functionName,
	}

	// Extract function definition
	// Get the full text from the start of CREATE to the end of the statement
	if ctx.GetStart() != nil && ctx.GetStop() != nil {
		startIndex := ctx.GetStart().GetTokenIndex()
		stopIndex := ctx.GetStop().GetTokenIndex()
		tokens := ctx.GetParser().GetTokenStream()

		var definitionParts []string
		for i := startIndex; i <= stopIndex; i++ {
			token := tokens.Get(i)
			if token != nil {
				definitionParts = append(definitionParts, token.GetText())
			}
		}
		function.Definition = strings.Join(definitionParts, " ")
	}

	e.functions[functionName] = function
}

// EnterCreateProcedure is called when entering a CREATE PROCEDURE statement
func (e *metadataExtractor) EnterCreateProcedure(ctx *mysql.CreateProcedureContext) {
	if e.err != nil {
		return
	}

	if ctx.ProcedureName() == nil {
		return
	}

	_, procedureName := mysqlparser.NormalizeMySQLProcedureName(ctx.ProcedureName())

	procedure := &storepb.ProcedureMetadata{
		Name: procedureName,
	}

	// Extract procedure definition
	// Get the full text from the start of CREATE to the end of the statement
	if ctx.GetStart() != nil && ctx.GetStop() != nil {
		startIndex := ctx.GetStart().GetTokenIndex()
		stopIndex := ctx.GetStop().GetTokenIndex()
		tokens := ctx.GetParser().GetTokenStream()

		var definitionParts []string
		for i := startIndex; i <= stopIndex; i++ {
			token := tokens.Get(i)
			if token != nil {
				definitionParts = append(definitionParts, token.GetText())
			}
		}
		procedure.Definition = strings.Join(definitionParts, " ")
	}

	e.procedures[procedureName] = procedure
}

// EnterCreateTrigger is called when entering a CREATE TRIGGER statement
func (e *metadataExtractor) EnterCreateTrigger(ctx *mysql.CreateTriggerContext) {
	if e.err != nil {
		return
	}

	if ctx.TriggerName() == nil || ctx.TableRef() == nil {
		return
	}

	_, triggerName := mysqlparser.NormalizeMySQLTriggerName(ctx.TriggerName())
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())

	trigger := &storepb.TriggerMetadata{
		Name: triggerName,
	}

	// Add trigger to the appropriate table
	table := e.getOrCreateTable(tableName)
	table.Triggers = append(table.Triggers, trigger)
}

// EnterAlterTable is called when entering an ALTER TABLE statement
func (e *metadataExtractor) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if e.err != nil {
		return
	}

	if ctx.TableRef() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	table := e.getOrCreateTable(tableName)

	// Extract alter specifications
	if ctx.AlterTableActions() != nil && ctx.AlterTableActions().AlterCommandList() != nil {
		if ctx.AlterTableActions().AlterCommandList().AlterList() != nil {
			for _, alterCmd := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
				if alterCmd == nil {
					continue
				}

				// Handle ADD COLUMN
				if alterCmd.ADD_SYMBOL() != nil && alterCmd.Identifier() != nil && alterCmd.FieldDefinition() != nil {
					columnName := mysqlparser.NormalizeMySQLIdentifier(alterCmd.Identifier())
					e.extractFieldDefinitionForAlter(columnName, alterCmd.FieldDefinition(), table)
				}

				// Handle ADD INDEX
				if alterCmd.ADD_SYMBOL() != nil && alterCmd.TableConstraintDef() != nil {
					e.extractTableConstraint(alterCmd.TableConstraintDef(), table)
				}
			}
		}
	}
}

// extractFieldDefinitionForAlter extracts field definition from ALTER TABLE ADD COLUMN
func (e *metadataExtractor) extractFieldDefinitionForAlter(columnName string, ctx mysql.IFieldDefinitionContext, table *storepb.TableMetadata) {
	if ctx == nil {
		return
	}

	// Check if column already exists
	for _, col := range table.Columns {
		if col.Name == columnName {
			return
		}
	}

	column := &storepb.ColumnMetadata{
		Name:     columnName,
		Position: int32(len(table.Columns) + 1),
	}

	// Extract data type
	if ctx.DataType() != nil {
		column.Type = e.extractDataType(ctx.DataType())
	}

	// Extract column attributes
	e.extractFieldAttributes(ctx, column)

	// Extract comment
	if comment := e.extractColumnComment(ctx); comment != "" {
		column.Comment = comment
	}

	table.Columns = append(table.Columns, column)
}

// EnterCreateIndex is called when entering a CREATE INDEX statement
func (e *metadataExtractor) EnterCreateIndex(ctx *mysql.CreateIndexContext) {
	if e.err != nil {
		return
	}

	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	table := e.getOrCreateTable(tableName)

	// Extract index name
	indexName := ""
	if ctx.IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexName())
	}

	// Extract columns/expressions
	var expressions []string
	if ctx.CreateIndexTarget().KeyListVariants() != nil {
		expressions = mysqlparser.NormalizeKeyListVariants(ctx.CreateIndexTarget().KeyListVariants())
	}

	if len(expressions) > 0 {
		indexType := "BTREE"
		// Check for index type
		if ctx.FULLTEXT_SYMBOL() != nil {
			indexType = "FULLTEXT"
		} else if ctx.SPATIAL_SYMBOL() != nil {
			indexType = "SPATIAL"
		}

		// Check if it's a unique index
		unique := ctx.UNIQUE_SYMBOL() != nil

		index := &storepb.IndexMetadata{
			Name:        indexName,
			Type:        indexType,
			Expressions: expressions,
			Primary:     false,
			Unique:      unique,
		}
		table.Indexes = append(table.Indexes, index)
	}
}

// normalizeDefaultValue normalizes default values to match MySQL's internal representation
func normalizeDefaultValue(defaultValue string) string {
	if defaultValue == "" {
		return ""
	}

	// Handle boolean values - MySQL stores them as 0/1
	switch strings.ToLower(defaultValue) {
	case "true":
		return "1"
	case "false":
		return "0"
	}

	return defaultValue
}
