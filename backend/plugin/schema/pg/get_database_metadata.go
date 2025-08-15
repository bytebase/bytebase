package pg

import (
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_POSTGRES, GetDatabaseMetadata)
}

// GetDatabaseMetadata parses the SQL schema text and returns the database metadata.
func GetDatabaseMetadata(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	parseResult, err := pgparser.ParsePostgreSQL(schemaText)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse PostgreSQL schema")
	}

	extractor := &metadataExtractor{
		currentDatabase:      "",
		currentSchema:        "public",
		schemas:              make(map[string]*storepb.SchemaMetadata),
		tables:               make(map[tableKey]*storepb.TableMetadata),
		partitionTables:      make(map[tableKey]bool),
		partitionExpressions: make(map[tableKey]string),
		sequences:            make(map[string]*storepb.SequenceMetadata),
		extensions:           make(map[string]*storepb.ExtensionMetadata),
	}

	// Always ensure public schema exists
	extractor.getOrCreateSchema("public")

	// Only walk the tree if it's not empty
	if parseResult.Tree != nil {
		antlr.ParseTreeWalkerDefault.Walk(extractor, parseResult.Tree)
	}

	if extractor.err != nil {
		return nil, extractor.err
	}

	// Build the final metadata structure
	schemaMetadata := &storepb.DatabaseSchemaMetadata{
		Name: extractor.currentDatabase,
	}

	// Sort schemas for consistent output
	var schemaNames []string
	for name := range extractor.schemas {
		schemaNames = append(schemaNames, name)
	}
	slices.Sort(schemaNames)

	for _, schemaName := range schemaNames {
		schema := extractor.schemas[schemaName]

		// Filter out any tables that are actually materialized views
		// This handles the case where materialized views are incorrectly classified as tables
		filteredTables := []*storepb.TableMetadata{}
		for _, table := range schema.Tables {
			// Check if this table name exists as a materialized view
			isMaterializedView := false
			for _, mv := range schema.MaterializedViews {
				if mv.Name == table.Name {
					isMaterializedView = true
					break
				}
			}

			// Only include in tables if it's not a materialized view
			if !isMaterializedView {
				filteredTables = append(filteredTables, table)
			}
		}
		schema.Tables = filteredTables

		schemaMetadata.Schemas = append(schemaMetadata.Schemas, schema)
	}

	// Add extensions
	var extensionNames []string
	for name := range extractor.extensions {
		extensionNames = append(extensionNames, name)
	}
	slices.Sort(extensionNames)

	for _, name := range extensionNames {
		schemaMetadata.Extensions = append(schemaMetadata.Extensions, extractor.extensions[name])
	}

	return schemaMetadata, nil
}

type tableKey struct {
	schema string
	table  string
}

// metadataExtractor walks the parse tree and extracts metadata
type metadataExtractor struct {
	*parser.BasePostgreSQLParserListener

	currentDatabase      string
	currentSchema        string
	schemas              map[string]*storepb.SchemaMetadata
	tables               map[tableKey]*storepb.TableMetadata
	partitionTables      map[tableKey]bool
	partitionExpressions map[tableKey]string
	sequences            map[string]*storepb.SequenceMetadata
	extensions           map[string]*storepb.ExtensionMetadata
	err                  error
}

// Helper function to get or create schema
func (e *metadataExtractor) getOrCreateSchema(schemaName string) *storepb.SchemaMetadata {
	if schemaName == "" {
		schemaName = "public"
	}

	if schema, exists := e.schemas[schemaName]; exists {
		return schema
	}

	schema := &storepb.SchemaMetadata{
		Name:              schemaName,
		Tables:            []*storepb.TableMetadata{},
		Views:             nil,
		MaterializedViews: nil,
		Procedures:        nil,
		Functions:         nil,
		Sequences:         nil,
		EnumTypes:         nil,
	}
	e.schemas[schemaName] = schema
	return schema
}

// Helper function to get or create table
func (e *metadataExtractor) getOrCreateTable(schemaName, tableName string) *storepb.TableMetadata {
	key := tableKey{
		schema: schemaName,
		table:  tableName,
	}

	if table, exists := e.tables[key]; exists {
		return table
	}

	table := &storepb.TableMetadata{
		Name:             tableName,
		Columns:          []*storepb.ColumnMetadata{},
		Indexes:          []*storepb.IndexMetadata{},
		ForeignKeys:      nil,
		CheckConstraints: nil,
		Triggers:         nil,
		Partitions:       nil,
	}

	// Only add to schema's table list if it's not a partition table
	if !e.partitionTables[key] {
		schema := e.getOrCreateSchema(schemaName)
		schema.Tables = append(schema.Tables, table)
	}
	e.tables[key] = table

	return table
}

// Helper function to find materialized view in a schema
func (e *metadataExtractor) findMaterializedView(schemaName, viewName string) *storepb.MaterializedViewMetadata {
	schema := e.getOrCreateSchema(schemaName)
	if schema.MaterializedViews == nil {
		return nil
	}

	for _, mv := range schema.MaterializedViews {
		if mv.Name == viewName {
			return mv
		}
	}
	return nil
}

// EnterCreateschemastmt is called when entering a create schema statement
func (e *metadataExtractor) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	if e.err != nil {
		return
	}

	// Try to get schema name directly from Colid first
	if ctx.Colid() != nil {
		schemaName := pgparser.NormalizePostgreSQLColid(ctx.Colid())
		e.getOrCreateSchema(schemaName)
	} else if ctx.Optschemaname() != nil && ctx.Optschemaname().Colid() != nil {
		schemaName := pgparser.NormalizePostgreSQLColid(ctx.Optschemaname().Colid())
		e.getOrCreateSchema(schemaName)
	}
}

// EnterCreatestmt is called when entering a create table statement
func (e *metadataExtractor) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if e.err != nil {
		return
	}

	if ctx.Qualified_name(0) == nil {
		return
	}

	qualifiedName := ctx.Qualified_name(0)
	schemaName, tableName := e.extractSchemaAndTableName(qualifiedName)

	// Check if this is a partition table (CREATE TABLE ... PARTITION OF ...)
	// We'll check if there's a second qualified_name which would be the parent table
	if ctx.Qualified_name(1) != nil && ctx.PARTITION() != nil && ctx.OF() != nil {
		// Mark this table as a partition
		key := tableKey{schema: schemaName, table: tableName}
		e.partitionTables[key] = true

		// Get the parent table and add this partition to it
		parentSchema, parentTable := e.extractSchemaAndTableName(ctx.Qualified_name(1))
		parentTableMetadata := e.getOrCreateTable(parentSchema, parentTable)

		// Create partition metadata
		partition := &storepb.TablePartitionMetadata{
			Name: tableName,
		}

		// Extract FOR VALUES clause if present
		if ctx.Partitionboundspec() != nil {
			partition.Value = e.extractPartitionBoundSpec(ctx.Partitionboundspec())
		}

		// Get the parent table's partition expression
		parentKey := tableKey{schema: parentSchema, table: parentTable}
		if expr, ok := e.partitionExpressions[parentKey]; ok {
			partition.Expression = expr
		}

		if parentTableMetadata.Partitions == nil {
			parentTableMetadata.Partitions = []*storepb.TablePartitionMetadata{}
		}
		parentTableMetadata.Partitions = append(parentTableMetadata.Partitions, partition)

		// Don't continue processing this table as a regular table
		return
	}

	// Create the table (it's not a partition table)
	tableMetadata := e.getOrCreateTable(schemaName, tableName)

	// Extract table elements (columns, constraints)
	if tableElementList := ctx.Opttableelementlist(); tableElementList != nil {
		e.extractTableElements(tableElementList, tableMetadata, schemaName)
	}

	// Extract partition info and store the expression
	if partitionSpec := ctx.Optpartitionspec(); partitionSpec != nil {
		// Store the partition expression for this table
		key := tableKey{schema: schemaName, table: tableName}
		e.partitionExpressions[key] = e.extractPartitionExpression(partitionSpec)
	}
}

// extractSchemaAndTableName extracts schema and table name from qualified name
func (e *metadataExtractor) extractSchemaAndTableName(ctx parser.IQualified_nameContext) (string, string) {
	if ctx == nil {
		return e.currentSchema, ""
	}

	parts := pgparser.NormalizePostgreSQLQualifiedName(ctx)
	if len(parts) == 1 {
		return e.currentSchema, parts[0]
	} else if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return e.currentSchema, ""
}

// extractTableElements extracts columns and constraints from table elements
func (e *metadataExtractor) extractTableElements(ctx parser.IOpttableelementlistContext, table *storepb.TableMetadata, schemaName string) {
	if ctx == nil {
		return
	}

	// Get the table element list
	tableElementList := ctx.Tableelementlist()
	if tableElementList == nil {
		return
	}

	// Process all table elements
	for _, tableElement := range tableElementList.AllTableelement() {
		if tableElement == nil {
			continue
		}

		// Handle column definitions
		if columnDef := tableElement.ColumnDef(); columnDef != nil {
			e.extractColumnDef(columnDef, table, schemaName)
		}

		// Handle table constraints
		if tableConstraint := tableElement.Tableconstraint(); tableConstraint != nil {
			e.extractTableConstraint(tableConstraint, table, schemaName)
		}
	}
}

// extractColumnDef extracts column definition
func (e *metadataExtractor) extractColumnDef(ctx parser.IColumnDefContext, table *storepb.TableMetadata, schemaName string) {
	if ctx == nil {
		return
	}

	column := &storepb.ColumnMetadata{
		Nullable: true, // Default to nullable
	}

	// Extract column name
	if ctx.Colid() != nil {
		column.Name = pgparser.NormalizePostgreSQLColid(ctx.Colid())
	}

	// Extract data type
	if ctx.Typename() != nil {
		// Get the raw type name to check for SERIAL
		rawTypeName := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Typename())
		rawTypeName = strings.ToLower(rawTypeName)

		// SERIAL columns are implicitly NOT NULL
		if rawTypeName == "serial" || rawTypeName == "bigserial" || rawTypeName == "smallserial" ||
			rawTypeName == "serial4" || rawTypeName == "serial8" || rawTypeName == "serial2" {
			column.Nullable = false
		}

		column.Type = e.extractTypeNameWithSchema(ctx.Typename(), schemaName)
	}

	// Extract column constraints
	if colquallist := ctx.Colquallist(); colquallist != nil {
		for _, colConstraint := range colquallist.AllColconstraint() {
			if colConstraint != nil && colConstraint.Colconstraintelem() != nil {
				// Extract constraint name if present
				var constraintName string
				if colConstraint.CONSTRAINT() != nil && colConstraint.Name() != nil {
					constraintName = pgparser.NormalizePostgreSQLName(colConstraint.Name())
				}
				e.extractColumnConstraint(colConstraint.Colconstraintelem(), column, table, constraintName, schemaName)
			}
		}
	}

	table.Columns = append(table.Columns, column)
}

// extractTypeName extracts the type name from typename context
func extractTypeName(ctx parser.ITypenameContext) string {
	if ctx == nil {
		return ""
	}
	// Get the full text representation of the type
	typeName := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	// Convert to lowercase to match sync.go output
	typeName = strings.ToLower(typeName)

	return normalizePostgreSQLType(typeName)
}

// normalizePostgreSQLType normalizes PostgreSQL type names to match sync.go output
func normalizePostgreSQLType(typeName string) string {
	// Remove extra whitespace
	typeName = strings.TrimSpace(typeName)

	// Convert common type variations to match sync.go output
	// varchar(n) -> character varying(n)
	if strings.HasPrefix(typeName, "varchar(") {
		return "character varying" + typeName[7:]
	}
	if typeName == "varchar" {
		return "character varying"
	}

	// SERIAL types are stored as integer/bigint in the catalog
	if typeName == "serial" || typeName == "serial4" {
		return "integer"
	}
	if typeName == "bigserial" || typeName == "serial8" {
		return "bigint"
	}
	if typeName == "smallserial" || typeName == "serial2" {
		return "smallint"
	}

	// Handle array types: text[] -> _text
	if strings.HasSuffix(typeName, "[]") {
		// PostgreSQL internal representation uses underscore prefix
		baseType := typeName[:len(typeName)-2]
		// Recursively normalize the base type
		normalizedBase := normalizePostgreSQLType(baseType)
		return "_" + normalizedBase
	}

	// Handle timestamp without explicit timezone
	if typeName == "timestamp" {
		return "timestamp without time zone"
	}

	// Handle decimal -> numeric (sync.go reports decimal as numeric without precision)
	if strings.HasPrefix(typeName, "decimal(") {
		// sync.go returns just "numeric" without precision for decimal types
		return "numeric"
	}
	if typeName == "decimal" {
		return "numeric"
	}

	// Normalize common type aliases to match PostgreSQL catalog output
	switch typeName {
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
	case "char":
		return "character"
	case "timestamptz":
		return "timestamp with time zone"
	case "timetz":
		return "time with time zone"
	default:
		// Return type as-is for unrecognized types
	}

	// Handle specific length specifications for character types
	if strings.HasPrefix(typeName, "char(") {
		return "character" + typeName[4:]
	}

	return typeName
}

// extractTypeNameWithSchema extracts the type name and adds schema prefix for custom types
func (*metadataExtractor) extractTypeNameWithSchema(ctx parser.ITypenameContext, schemaName string) string {
	if ctx == nil {
		return ""
	}

	// Get the raw type name first (without normalization)
	rawTypeName := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	rawTypeName = strings.ToLower(strings.TrimSpace(rawTypeName))

	// Check if this is a built-in PostgreSQL type before normalization
	if isBuiltInType(rawTypeName) || isBuiltInType(normalizePostgreSQLType(rawTypeName)) {
		// For built-in types, return the normalized version
		return normalizePostgreSQLType(rawTypeName)
	}

	// Check if the type already has a schema prefix
	if strings.Contains(rawTypeName, ".") {
		// It's already schema-qualified, just normalize it
		return normalizePostgreSQLType(rawTypeName)
	}

	// For custom types (like enums), add the schema prefix to the raw type name
	return fmt.Sprintf("%s.%s", schemaName, rawTypeName)
}

// isBuiltInType checks if a type is a built-in PostgreSQL type
func isBuiltInType(typeName string) bool {
	// Remove any precision/scale information for checking
	baseType := typeName
	if idx := strings.Index(typeName, "("); idx != -1 {
		baseType = typeName[:idx]
	}

	// Common built-in PostgreSQL types including both aliases and canonical forms
	builtInTypes := map[string]bool{
		// Integer types
		"integer": true, "int": true, "int4": true,
		"bigint": true, "int8": true,
		"smallint": true, "int2": true,
		"serial": true, "serial4": true,
		"bigserial": true, "serial8": true, "smallserial": true, "serial2": true,

		// Numeric types
		"numeric": true, "decimal": true,
		"real": true, "float4": true,
		"double precision": true, "float8": true,
		"money": true,

		// Character types
		"character": true, "char": true,
		"character varying": true, "varchar": true,
		"text": true,
		"name": true,

		// Boolean
		"boolean": true, "bool": true,

		// Date/time types
		"date": true,
		"time": true, "time without time zone": true, "time with time zone": true, "timetz": true,
		"timestamp": true, "timestamp without time zone": true, "timestamp with time zone": true, "timestamptz": true,
		"interval": true,

		// UUID
		"uuid": true,

		// JSON types
		"json": true, "jsonb": true,

		// XML
		"xml": true,

		// Binary data
		"bytea": true,

		// Bit string types
		"bit": true, "bit varying": true,

		// Network address types
		"inet": true, "cidr": true, "macaddr": true, "macaddr8": true,

		// Geometric types
		"point": true, "line": true, "lseg": true, "box": true,
		"path": true, "polygon": true, "circle": true,

		// Full-text search types
		"tsquery": true, "tsvector": true,

		// Range types
		"int4range": true, "int8range": true, "numrange": true,
		"tsrange": true, "tstzrange": true, "daterange": true,

		// Other types
		"pg_lsn": true,
		"oid":    true, "regclass": true, "regproc": true, "regtype": true, "regoper": true,
		"regoperator": true, "regconfig": true, "regdictionary": true,
		"tid": true, "xid": true, "cid": true,
	}

	// Also check for array types (ending with [] or starting with _)
	if strings.HasSuffix(baseType, "[]") {
		arrayBase := baseType[:len(baseType)-2]
		// Recursively check if the base type is built-in
		return isBuiltInType(arrayBase)
	}
	if strings.HasPrefix(baseType, "_") {
		// PostgreSQL internal array notation _typename
		arrayBase := baseType[1:]
		return isBuiltInType(arrayBase)
	}

	return builtInTypes[baseType]
}

// extractColumnConstraint extracts column-level constraints
func (e *metadataExtractor) extractColumnConstraint(ctx parser.IColconstraintelemContext, column *storepb.ColumnMetadata, table *storepb.TableMetadata, constraintName string, schemaName string) {
	if ctx == nil {
		return
	}

	switch {
	case ctx.NOT() != nil && ctx.NULL_P() != nil:
		column.Nullable = false
	case ctx.NULL_P() != nil && ctx.NOT() == nil:
		column.Nullable = true
	case ctx.DEFAULT() != nil:
		if expr := ctx.B_expr(); expr != nil {
			column.Default = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr)
		}
	case ctx.PRIMARY() != nil && ctx.KEY() != nil:
		column.Nullable = false
		// Create primary key index
		index := &storepb.IndexMetadata{
			Primary:      true,
			Unique:       true,
			IsConstraint: true,
			Expressions:  []string{column.Name},
			Descending:   []bool{false},
			Type:         "btree",
			Visible:      false, // Match PostgreSQL database behavior
			// Don't set KeyLength - PostgreSQL database doesn't return this information
		}
		// Use provided constraint name or generate one
		if constraintName != "" {
			index.Name = constraintName
		} else {
			index.Name = fmt.Sprintf("%s_pkey", table.Name)
		}
		table.Indexes = append(table.Indexes, index)
	case ctx.UNIQUE() != nil:
		// Create unique index
		index := &storepb.IndexMetadata{
			Unique:       true,
			IsConstraint: true,
			Expressions:  []string{column.Name},
			Descending:   []bool{false},
			Type:         "btree",
			Visible:      false, // Match PostgreSQL database behavior
			// Don't set KeyLength - PostgreSQL database doesn't return this information
		}
		// Use provided constraint name or generate one
		if constraintName != "" {
			index.Name = constraintName
		} else {
			index.Name = fmt.Sprintf("%s_%s_key", table.Name, column.Name)
		}
		table.Indexes = append(table.Indexes, index)
	case ctx.REFERENCES() != nil:
		// Foreign key constraint
		fk := &storepb.ForeignKeyMetadata{
			Columns: []string{column.Name},
		}
		if constraintName != "" {
			fk.Name = constraintName
		}
		if qualifiedName := ctx.Qualified_name(); qualifiedName != nil {
			refSchema, refTable := e.extractSchemaAndTableName(qualifiedName)
			fk.ReferencedSchema = refSchema
			fk.ReferencedTable = refTable
		}
		if optColumnList := ctx.Opt_column_list(); optColumnList != nil && optColumnList.Columnlist() != nil {
			fk.ReferencedColumns = extractColumnList(optColumnList.Columnlist())
		}
		// Extract ON DELETE/UPDATE actions
		fk.OnDelete = "NO ACTION" // Default
		fk.OnUpdate = "NO ACTION" // Default
		if keyActions := ctx.Key_actions(); keyActions != nil {
			if keyDelete := keyActions.Key_delete(); keyDelete != nil {
				fk.OnDelete = extractKeyAction(keyDelete)
			}
			if keyUpdate := keyActions.Key_update(); keyUpdate != nil {
				fk.OnUpdate = extractKeyActionUpdate(keyUpdate)
			}
		}
		// Extract MATCH type
		fk.MatchType = extractMatchType(ctx)

		if table.ForeignKeys == nil {
			table.ForeignKeys = []*storepb.ForeignKeyMetadata{}
		}
		table.ForeignKeys = append(table.ForeignKeys, fk)
	case ctx.GENERATED() != nil && ctx.IDENTITY_P() != nil:
		// Handle GENERATED ALWAYS AS IDENTITY or GENERATED BY DEFAULT AS IDENTITY
		if generatedWhen := ctx.Generated_when(); generatedWhen != nil {
			if generatedWhen.ALWAYS() != nil {
				column.IdentityGeneration = storepb.ColumnMetadata_ALWAYS
			} else if generatedWhen.BY() != nil && generatedWhen.DEFAULT() != nil {
				column.IdentityGeneration = storepb.ColumnMetadata_BY_DEFAULT
			}
		}

		// Create identity sequence for this column
		e.createIdentitySequence(table, column, schemaName)
	default:
		// Ignore other column constraints
	}
}

// extractTableConstraint extracts table-level constraints
func (e *metadataExtractor) extractTableConstraint(ctx parser.ITableconstraintContext, table *storepb.TableMetadata, _ string) {
	if ctx == nil {
		return
	}

	constraintElem := ctx.Constraintelem()
	if constraintElem == nil {
		return
	}

	// Get constraint name
	var constraintName string
	if ctx.Name() != nil {
		constraintName = pgparser.NormalizePostgreSQLName(ctx.Name())
	}

	switch {
	case constraintElem.PRIMARY() != nil && constraintElem.KEY() != nil:
		// Primary key constraint
		index := &storepb.IndexMetadata{
			Name:         constraintName,
			Primary:      true,
			Unique:       true,
			IsConstraint: true,
			Expressions:  []string{},
			Descending:   []bool{},
			Type:         "btree",
			Visible:      false, // Match PostgreSQL database behavior
			// Don't set KeyLength - PostgreSQL database doesn't return this information
		}
		if index.Name == "" {
			index.Name = fmt.Sprintf("%s_pkey", table.Name)
		}
		// Extract columns
		// For PRIMARY KEY constraints, columns are in direct Columnlist, not Opt_column_list
		if columnList := constraintElem.Columnlist(); columnList != nil {
			columns := extractColumnList(columnList)
			index.Expressions = columns
			for range columns {
				index.Descending = append(index.Descending, false)
			}
		} else if optColumnList := constraintElem.Opt_column_list(); optColumnList != nil && optColumnList.Columnlist() != nil {
			// Fallback to Opt_column_list if direct Columnlist is not available
			columns := extractColumnList(optColumnList.Columnlist())
			index.Expressions = columns
			for range columns {
				index.Descending = append(index.Descending, false)
			}
		}
		table.Indexes = append(table.Indexes, index)

	case constraintElem.UNIQUE() != nil:
		// Unique constraint
		index := &storepb.IndexMetadata{
			Name:         constraintName,
			Unique:       true,
			IsConstraint: true,
			Expressions:  []string{},
			Descending:   []bool{},
			Type:         "btree",
			Visible:      false, // Match PostgreSQL database behavior
			// Don't set KeyLength - PostgreSQL database doesn't return this information
		}
		// Extract columns
		// For UNIQUE constraints, columns are in direct Columnlist, not Opt_column_list
		if columnList := constraintElem.Columnlist(); columnList != nil {
			columns := extractColumnList(columnList)
			index.Expressions = columns
			for range columns {
				index.Descending = append(index.Descending, false)
			}
		} else if optColumnList := constraintElem.Opt_column_list(); optColumnList != nil && optColumnList.Columnlist() != nil {
			// Fallback to Opt_column_list if direct Columnlist is not available
			columns := extractColumnList(optColumnList.Columnlist())
			index.Expressions = columns
			for range columns {
				index.Descending = append(index.Descending, false)
			}
		}
		table.Indexes = append(table.Indexes, index)

	case constraintElem.CHECK() != nil:
		// Check constraint
		check := &storepb.CheckConstraintMetadata{
			Name: constraintName,
		}
		if expr := constraintElem.A_expr(); expr != nil {
			check.Expression = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr)
		}
		if table.CheckConstraints == nil {
			table.CheckConstraints = []*storepb.CheckConstraintMetadata{}
		}
		table.CheckConstraints = append(table.CheckConstraints, check)

	case constraintElem.FOREIGN() != nil && constraintElem.KEY() != nil:
		// Foreign key constraint
		fk := &storepb.ForeignKeyMetadata{
			Name:              constraintName,
			Columns:           []string{},
			ReferencedColumns: []string{},
		}
		// Extract local columns from Columnlist (before REFERENCES)
		if columnList := constraintElem.Columnlist(); columnList != nil {
			fk.Columns = extractColumnList(columnList)
		}
		// Extract referenced table
		if qualifiedName := constraintElem.Qualified_name(); qualifiedName != nil {
			refSchema, refTable := e.extractSchemaAndTableName(qualifiedName)
			fk.ReferencedSchema = refSchema
			fk.ReferencedTable = refTable
		}
		// Extract referenced columns from Opt_column_list (after REFERENCES table_name)
		if optColumnList := constraintElem.Opt_column_list(); optColumnList != nil && optColumnList.Columnlist() != nil {
			fk.ReferencedColumns = extractColumnList(optColumnList.Columnlist())
		} else if len(fk.Columns) > 0 {
			// If referenced columns not specified, assume they match the local columns
			fk.ReferencedColumns = make([]string, len(fk.Columns))
			copy(fk.ReferencedColumns, fk.Columns)
		}
		// Extract ON DELETE/UPDATE actions
		fk.OnDelete = "NO ACTION" // Default
		fk.OnUpdate = "NO ACTION" // Default
		if keyActions := constraintElem.Key_actions(); keyActions != nil {
			if keyDelete := keyActions.Key_delete(); keyDelete != nil {
				fk.OnDelete = extractKeyAction(keyDelete)
			}
			if keyUpdate := keyActions.Key_update(); keyUpdate != nil {
				fk.OnUpdate = extractKeyActionUpdate(keyUpdate)
			}
		}
		// Extract MATCH type
		fk.MatchType = extractMatchTypeFromConstraintElem(constraintElem)
		if table.ForeignKeys == nil {
			table.ForeignKeys = []*storepb.ForeignKeyMetadata{}
		}
		table.ForeignKeys = append(table.ForeignKeys, fk)
	default:
		// Other constraint types not handled
	}
}

// extractKeyAction extracts the foreign key delete action type
func extractKeyAction(ctx parser.IKey_deleteContext) string {
	if ctx == nil {
		return "NO ACTION"
	}
	keyAction := ctx.Key_action()
	if keyAction == nil {
		return "NO ACTION"
	}
	switch {
	case keyAction.CASCADE() != nil:
		return "CASCADE"
	case keyAction.SET() != nil && keyAction.NULL_P() != nil:
		return "SET NULL"
	case keyAction.SET() != nil && keyAction.DEFAULT() != nil:
		return "SET DEFAULT"
	case keyAction.RESTRICT() != nil:
		return "RESTRICT"
	default:
		return "NO ACTION"
	}
}

// extractKeyActionUpdate extracts the foreign key update action type
func extractKeyActionUpdate(ctx parser.IKey_updateContext) string {
	if ctx == nil {
		return "NO ACTION"
	}
	keyAction := ctx.Key_action()
	if keyAction == nil {
		return "NO ACTION"
	}
	switch {
	case keyAction.CASCADE() != nil:
		return "CASCADE"
	case keyAction.SET() != nil && keyAction.NULL_P() != nil:
		return "SET NULL"
	case keyAction.SET() != nil && keyAction.DEFAULT() != nil:
		return "SET DEFAULT"
	case keyAction.RESTRICT() != nil:
		return "RESTRICT"
	default:
		return "NO ACTION"
	}
}

// extractMatchType extracts the match type from column constraint context
func extractMatchType(ctx parser.IColconstraintelemContext) string {
	if ctx == nil || ctx.Key_match() == nil {
		return "" // Default is empty (MATCH SIMPLE)
	}
	keyMatch := ctx.Key_match()
	switch {
	case keyMatch.FULL() != nil:
		return "FULL"
	case keyMatch.PARTIAL() != nil:
		return "PARTIAL"
	case keyMatch.SIMPLE() != nil:
		return "SIMPLE"
	default:
		return ""
	}
}

// extractMatchTypeFromConstraintElem extracts the match type from table constraint context
func extractMatchTypeFromConstraintElem(ctx parser.IConstraintelemContext) string {
	if ctx == nil || ctx.Key_match() == nil {
		return "" // Default is empty (MATCH SIMPLE)
	}
	keyMatch := ctx.Key_match()
	switch {
	case keyMatch.FULL() != nil:
		return "FULL"
	case keyMatch.PARTIAL() != nil:
		return "PARTIAL"
	case keyMatch.SIMPLE() != nil:
		return "SIMPLE"
	default:
		return ""
	}
}

// extractColumnList extracts column names from columnlist
func extractColumnList(ctx parser.IColumnlistContext) []string {
	if ctx == nil {
		return nil
	}

	var columns []string
	for _, colElem := range ctx.AllColumnElem() {
		if colElem.Colid() != nil {
			columns = append(columns, pgparser.NormalizePostgreSQLColid(colElem.Colid()))
		}
	}
	return columns
}

// extractPartitionExpression extracts the partition expression (e.g., "RANGE (sale_date)")
func (*metadataExtractor) extractPartitionExpression(ctx parser.IOptpartitionspecContext) string {
	if ctx == nil {
		return ""
	}

	partitionSpec := ctx.Partitionspec()
	if partitionSpec == nil {
		return ""
	}

	// Extract the full partition expression including the method (e.g., "RANGE (sale_date)")
	expression := ctx.GetParser().GetTokenStream().GetTextFromTokens(partitionSpec.Colid().GetStart(), partitionSpec.CLOSE_PAREN().GetSymbol())
	return expression
}

// EnterAltertablestmt is called when entering an ALTER TABLE statement
func (e *metadataExtractor) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if e.err != nil {
		return
	}

	// Check if this is an ATTACH PARTITION command
	if partitionCmd := ctx.Partition_cmd(); partitionCmd != nil {
		if partitionCmd.ATTACH() != nil && partitionCmd.PARTITION() != nil {
			e.handleAttachPartition(ctx, partitionCmd)
		}
		return
	}

	// Check if this is an ADD CONSTRAINT command
	if alterTableCmdList := ctx.Alter_table_cmds(); alterTableCmdList != nil {
		e.handleAlterTableCommands(ctx, alterTableCmdList)
	}
}

// handleAlterTableCommands processes ALTER TABLE commands like ADD CONSTRAINT
func (e *metadataExtractor) handleAlterTableCommands(ctx *parser.AltertablestmtContext, alterTableCmdList parser.IAlter_table_cmdsContext) {
	// Get the table name from the ALTER TABLE statement
	if relationExpr := ctx.Relation_expr(); relationExpr != nil {
		if qualifiedName := relationExpr.Qualified_name(); qualifiedName != nil {
			schemaName, tableName := e.extractSchemaAndTableName(qualifiedName)

			// Get or create the table
			table := e.getOrCreateTable(schemaName, tableName)

			// Process each alter table command
			for _, alterTableCmd := range alterTableCmdList.AllAlter_table_cmd() {
				if alterTableCmd == nil {
					continue
				}

				// Check if this is an ADD CONSTRAINT command
				if alterTableCmd.ADD_P() != nil && alterTableCmd.Tableconstraint() != nil {
					e.extractTableConstraint(alterTableCmd.Tableconstraint(), table, schemaName)
				}
			}
		}
	}
}

// handleAttachPartition processes ALTER TABLE ATTACH PARTITION statements
func (e *metadataExtractor) handleAttachPartition(ctx *parser.AltertablestmtContext, partitionCmd parser.IPartition_cmdContext) {
	// Get the main table name
	if relationExpr := ctx.Relation_expr(); relationExpr != nil {
		if qualifiedName := relationExpr.Qualified_name(); qualifiedName != nil {
			mainSchema, mainTable := e.extractSchemaAndTableName(qualifiedName)

			// Get the partition name
			if partitionQualifiedName := partitionCmd.Qualified_name(); partitionQualifiedName != nil {
				partitionSchema, partitionName := e.extractSchemaAndTableName(partitionQualifiedName)

				// Get or create the main table
				mainTableMetadata := e.getOrCreateTable(mainSchema, mainTable)

				// Create partition metadata
				partition := &storepb.TablePartitionMetadata{
					Name: partitionName,
				}

				// Extract the FOR VALUES clause if present
				if partitionBound := partitionCmd.Partitionboundspec(); partitionBound != nil {
					partition.Value = e.extractPartitionBoundSpec(partitionBound)
				}

				// Add to main table's partitions
				if mainTableMetadata.Partitions == nil {
					mainTableMetadata.Partitions = []*storepb.TablePartitionMetadata{}
				}
				mainTableMetadata.Partitions = append(mainTableMetadata.Partitions, partition)

				// Ensure the partition table exists in our metadata
				// This handles cases where the partition table was created separately
				_ = e.getOrCreateTable(partitionSchema, partitionName)
			}
		}
	}
}

// EnterCreatematviewstmt is called when entering a CREATE MATERIALIZED VIEW statement
func (e *metadataExtractor) EnterCreatematviewstmt(ctx *parser.CreatematviewstmtContext) {
	if e.err != nil {
		return
	}

	// Get the materialized view target
	mvTarget := ctx.Create_mv_target()
	if mvTarget == nil {
		return
	}

	// Extract schema and view name from the target
	var schemaName, viewName string
	if qualifiedName := mvTarget.Qualified_name(); qualifiedName != nil {
		schemaName, viewName = e.extractSchemaAndTableName(qualifiedName)
	} else {
		return
	}

	schema := e.getOrCreateSchema(schemaName)

	// Create materialized view metadata
	materializedView := &storepb.MaterializedViewMetadata{
		Name: viewName,
	}

	// Extract the view definition
	if ctx.AS() != nil && ctx.Selectstmt() != nil {
		// Get the full SELECT statement text
		selectCtx := ctx.Selectstmt()
		startToken := selectCtx.GetStart()
		stopToken := selectCtx.GetStop()
		if startToken != nil && stopToken != nil {
			materializedView.Definition = ctx.GetParser().GetTokenStream().GetTextFromTokens(startToken, stopToken)
		}
	}

	// Add to schema's materialized views
	if schema.MaterializedViews == nil {
		schema.MaterializedViews = []*storepb.MaterializedViewMetadata{}
	}
	schema.MaterializedViews = append(schema.MaterializedViews, materializedView)
}

// extractPartitionBoundSpec extracts the partition bound specification (FOR VALUES clause)
func (*metadataExtractor) extractPartitionBoundSpec(ctx parser.IPartitionboundspecContext) string {
	if ctx == nil {
		return ""
	}

	// Get the full text of the partition bound specification
	return ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterCreateseqstmt is called when entering a create sequence statement
func (e *metadataExtractor) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if e.err != nil {
		return
	}

	if ctx.Qualified_name() == nil {
		return
	}

	schemaName, sequenceName := e.extractSchemaAndTableName(ctx.Qualified_name())
	schemaMetadata := e.getOrCreateSchema(schemaName)

	sequence := &storepb.SequenceMetadata{
		Name:      sequenceName,
		DataType:  "bigint", // Default for PostgreSQL
		Start:     "1",
		Increment: "1",
		MinValue:  "1",
		MaxValue:  "9223372036854775807",
		Cycle:     false,
		CacheSize: "1",
	}

	// Extract sequence options
	if optSeqList := ctx.Optseqoptlist(); optSeqList != nil && optSeqList.Seqoptlist() != nil {
		for _, seqOptElem := range optSeqList.Seqoptlist().AllSeqoptelem() {
			if seqOptElem == nil {
				continue
			}
			e.extractSequenceOption(seqOptElem, sequence)
		}
	}

	// Store the sequence temporarily for OWNED BY processing
	e.sequences[fmt.Sprintf("%s.%s", schemaName, sequenceName)] = sequence

	if schemaMetadata.Sequences == nil {
		schemaMetadata.Sequences = []*storepb.SequenceMetadata{}
	}
	schemaMetadata.Sequences = append(schemaMetadata.Sequences, sequence)
}

// EnterViewstmt is called when entering a create view statement
func (e *metadataExtractor) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if e.err != nil {
		return
	}

	if ctx.Qualified_name() == nil {
		return
	}

	schemaName, viewName := e.extractSchemaAndTableName(ctx.Qualified_name())
	schemaMetadata := e.getOrCreateSchema(schemaName)

	// Create regular view metadata
	// Note: Materialized views are not currently supported by the parser and are handled by sync
	viewMetadata := &storepb.ViewMetadata{
		Name: viewName,
	}

	// Extract view definition
	if ctx.Selectstmt() != nil {
		viewMetadata.Definition = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Selectstmt())
	}

	if schemaMetadata.Views == nil {
		schemaMetadata.Views = []*storepb.ViewMetadata{}
	}
	schemaMetadata.Views = append(schemaMetadata.Views, viewMetadata)
}

// EnterCreatefunctionstmt is called when entering a create function statement
func (e *metadataExtractor) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if e.err != nil {
		return
	}

	funcNameCtx := ctx.Func_name()
	if funcNameCtx == nil {
		return
	}

	parts := pgparser.NormalizePostgreSQLFuncName(funcNameCtx)
	schemaName := e.currentSchema
	funcName := ""
	if len(parts) == 1 {
		funcName = parts[0]
	} else if len(parts) == 2 {
		schemaName = parts[0]
		funcName = parts[1]
	}

	if funcName == "" {
		return
	}

	schemaMetadata := e.getOrCreateSchema(schemaName)

	functionMetadata := &storepb.FunctionMetadata{
		Name:       funcName,
		Definition: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
		Signature:  e.extractFunctionSignature(ctx, funcName),
	}

	if schemaMetadata.Functions == nil {
		schemaMetadata.Functions = []*storepb.FunctionMetadata{}
	}
	schemaMetadata.Functions = append(schemaMetadata.Functions, functionMetadata)
}

// extractFunctionSignature extracts the function signature with parameter types
func (*metadataExtractor) extractFunctionSignature(ctx *parser.CreatefunctionstmtContext, funcName string) string {
	var signature strings.Builder
	signature.WriteString(`"`)
	signature.WriteString(funcName)
	signature.WriteString(`"(`)

	if funcArgs := ctx.Func_args_with_defaults(); funcArgs != nil {
		if funcArgsList := funcArgs.Func_args_with_defaults_list(); funcArgsList != nil {
			args := funcArgsList.AllFunc_arg_with_default()
			for i, arg := range args {
				if i > 0 {
					signature.WriteString(", ")
				}

				if funcArg := arg.Func_arg(); funcArg != nil {
					// Extract parameter name if present
					if paramName := funcArg.Param_name(); paramName != nil {
						// Param_name returns the parameter name directly as text
						signature.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(paramName))
						signature.WriteString(" ")
					}

					// Extract argument class (IN/OUT/INOUT/VARIADIC)
					if argClass := funcArg.Arg_class(); argClass != nil {
						if argClass.OUT_P() != nil {
							signature.WriteString("OUT ")
						} else if argClass.INOUT() != nil {
							signature.WriteString("INOUT ")
						} else if argClass.VARIADIC() != nil {
							signature.WriteString("VARIADIC ")
						}
						// IN is default and usually omitted
					}

					// Extract parameter type
					if funcType := funcArg.Func_type(); funcType != nil {
						if funcType.Typename() != nil {
							signature.WriteString(extractTypeName(funcType.Typename()))
						}
					}
				}
			}
		}
	}

	signature.WriteString(")")
	return signature.String()
}

// EnterCreateextensionstmt is called when entering a create extension statement
func (e *metadataExtractor) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	if e.err != nil {
		return
	}

	if ctx.Name() == nil {
		return
	}

	extension := &storepb.ExtensionMetadata{
		Name:   pgparser.NormalizePostgreSQLName(ctx.Name()),
		Schema: "public", // Default schema
	}

	// Extract schema from extension options if present
	if optList := ctx.Create_extension_opt_list(); optList != nil {
		for _, optItem := range optList.AllCreate_extension_opt_item() {
			if optItem == nil {
				continue
			}
			// Check if this is a SCHEMA option
			if optItem.SCHEMA() != nil && optItem.Name() != nil {
				schemaName := pgparser.NormalizePostgreSQLName(optItem.Name())
				extension.Schema = schemaName
				break
			}
		}
	}

	e.extensions[extension.Name] = extension
}

// EnterDefinestmt is called when entering a define statement (CREATE TYPE AS ENUM)
func (e *metadataExtractor) EnterDefinestmt(ctx *parser.DefinestmtContext) {
	if e.err != nil {
		return
	}

	// Check if this is CREATE TYPE AS ENUM
	if ctx.CREATE() != nil && ctx.TYPE_P() != nil && ctx.AS() != nil && ctx.ENUM_P() != nil {
		// Extract type name
		typeNames := ctx.AllAny_name()
		if len(typeNames) == 0 {
			return
		}

		// Get the schema and enum name from the first Any_name (which should be the type name)
		typeName := typeNames[0]
		schemaName, enumName := e.extractSchemaAndEnumName(typeName)
		schemaMetadata := e.getOrCreateSchema(schemaName)

		// Create enum metadata
		enumType := &storepb.EnumTypeMetadata{
			Name:   enumName,
			Values: []string{},
		}

		// Extract enum values
		if optEnumValList := ctx.Opt_enum_val_list(); optEnumValList != nil {
			if enumValList := optEnumValList.Enum_val_list(); enumValList != nil {
				for _, sconst := range enumValList.AllSconst() {
					if sconst != nil {
						value := extractStringConstant(sconst)
						if value != "" {
							enumType.Values = append(enumType.Values, value)
						}
					}
				}
			}
		}

		if schemaMetadata.EnumTypes == nil {
			schemaMetadata.EnumTypes = []*storepb.EnumTypeMetadata{}
		}
		schemaMetadata.EnumTypes = append(schemaMetadata.EnumTypes, enumType)
	}
}

// EnterIndexstmt is called when entering a create index statement
func (e *metadataExtractor) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if e.err != nil {
		return
	}

	// Check if this is CREATE INDEX
	if ctx.CREATE() == nil || ctx.INDEX() == nil || ctx.ON() == nil {
		return
	}

	// Extract index name
	var indexName string
	if name := ctx.Name(); name != nil {
		indexName = pgparser.NormalizePostgreSQLName(name)
	}

	// If no explicit name, PostgreSQL will generate one - we can't predict it here
	if indexName == "" {
		return
	}

	// Extract table/view name from relation_expr
	if relationExpr := ctx.Relation_expr(); relationExpr != nil {
		if qualifiedName := relationExpr.Qualified_name(); qualifiedName != nil {
			schemaName, relationName := e.extractSchemaAndTableName(qualifiedName)

			// Try to find materialized view first, then table
			var indexTarget any
			var targetIndexes *[]*storepb.IndexMetadata

			if materializedView := e.findMaterializedView(schemaName, relationName); materializedView != nil {
				// Index is on a materialized view
				indexTarget = materializedView
				if materializedView.Indexes == nil {
					materializedView.Indexes = []*storepb.IndexMetadata{}
				}
				targetIndexes = &materializedView.Indexes
			} else {
				// Index is on a table - get or create the table
				tableMetadata := e.getOrCreateTable(schemaName, relationName)
				indexTarget = tableMetadata
				targetIndexes = &tableMetadata.Indexes
			}

			// If we couldn't find either table or materialized view, skip
			if indexTarget == nil || targetIndexes == nil {
				return
			}

			// Create index metadata
			index := &storepb.IndexMetadata{
				Name:         indexName,
				Expressions:  []string{},
				Descending:   []bool{},
				Unique:       false,
				Primary:      false,
				IsConstraint: false,
				Visible:      false, // Match PostgreSQL database behavior
				// Don't set KeyLength - PostgreSQL database doesn't return this information
			}

			// Check if it's a unique index
			if optUnique := ctx.Opt_unique(); optUnique != nil {
				index.Unique = true
			}

			// Extract index method (BTREE, HASH, GIN, GIST, etc.)
			// Default to btree if not specified
			index.Type = "btree"
			if accessMethod := ctx.Access_method_clause(); accessMethod != nil {
				if accessMethod.USING() != nil && accessMethod.Name() != nil {
					index.Type = strings.ToLower(pgparser.NormalizePostgreSQLName(accessMethod.Name()))
				}
			}

			// Extract index parameters (columns/expressions)
			if indexParams := ctx.Index_params(); indexParams != nil {
				for _, indexElem := range indexParams.AllIndex_elem() {
					if indexElem == nil {
						continue
					}

					// Extract column name or expression
					var expression string
					if colid := indexElem.Colid(); colid != nil {
						// Simple column reference
						expression = pgparser.NormalizePostgreSQLColid(colid)
					} else if funcExpr := indexElem.Func_expr_windowless(); funcExpr != nil {
						// Function expression
						expression = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(funcExpr)
					} else if aExpr := indexElem.A_expr(); aExpr != nil {
						// General expression (in parentheses)
						expression = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(aExpr)
					}

					if expression != "" {
						index.Expressions = append(index.Expressions, expression)

						// Extract sort order (ASC/DESC) from index element options
						isDescending := false
						if options := indexElem.Index_elem_options(); options != nil {
							if ascDesc := options.Opt_asc_desc(); ascDesc != nil {
								if ascDesc.DESC() != nil {
									isDescending = true
								}
								// ASC is default, so we don't need to check for it explicitly
							}
						}
						index.Descending = append(index.Descending, isDescending)
						// Don't set KeyLength - PostgreSQL database doesn't return this information
					}
				}
			}

			// Add the index to the table or materialized view
			*targetIndexes = append(*targetIndexes, index)
		}
	}
}

// extractSequenceOption extracts sequence options
func (*metadataExtractor) extractSequenceOption(ctx parser.ISeqoptelemContext, sequence *storepb.SequenceMetadata) {
	if ctx == nil || sequence == nil {
		return
	}

	switch {
	case ctx.AS() != nil && ctx.Simpletypename() != nil:
		// Data type - preserve the original case
		sequence.DataType = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Simpletypename())
	case ctx.INCREMENT() != nil && ctx.Numericonly() != nil:
		// INCREMENT BY
		sequence.Increment = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Numericonly())
	case ctx.MINVALUE() != nil && ctx.Numericonly() != nil:
		// MINVALUE
		sequence.MinValue = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Numericonly())
	case ctx.MAXVALUE() != nil && ctx.Numericonly() != nil:
		// MAXVALUE
		sequence.MaxValue = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Numericonly())
	case ctx.START() != nil && ctx.Numericonly() != nil:
		// START WITH
		sequence.Start = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Numericonly())
	case ctx.CACHE() != nil && ctx.Numericonly() != nil:
		// CACHE
		sequence.CacheSize = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Numericonly())
	case ctx.CYCLE() != nil:
		// CYCLE
		sequence.Cycle = true
	case ctx.NO() != nil && ctx.CYCLE() != nil:
		// NO CYCLE
		sequence.Cycle = false
	default:
		// Other sequence options
	}
}

// createIdentitySequence creates an identity sequence for a column with GENERATED AS IDENTITY
func (e *metadataExtractor) createIdentitySequence(table *storepb.TableMetadata, column *storepb.ColumnMetadata, schemaName string) {
	// Create identity sequence name following PostgreSQL conventions
	// Format: {table_name}_{column_name}_seq
	sequenceName := fmt.Sprintf("%s_%s_seq", table.Name, column.Name)

	// Determine sequence data type and limits based on column type
	// For identity columns, use positive ranges starting from 1
	var dataType, minValue, maxValue string
	switch strings.ToUpper(column.Type) {
	case "SMALLINT", "INT2":
		dataType = "smallint"
		minValue = "1"
		maxValue = "32767"
	case "INTEGER", "INT", "INT4":
		dataType = "integer"
		minValue = "1"
		maxValue = "2147483647"
	case "BIGINT", "INT8":
		dataType = "bigint"
		minValue = "1"
		maxValue = "9223372036854775807"
	default:
		// Default to bigint for unknown types
		dataType = "bigint"
		minValue = "1"
		maxValue = "9223372036854775807"
	}

	// Create the sequence metadata
	sequence := &storepb.SequenceMetadata{
		Name:        sequenceName,
		DataType:    dataType,
		Start:       "1",
		Increment:   "1",
		MinValue:    minValue,
		MaxValue:    maxValue,
		Cycle:       false,
		CacheSize:   "1",
		OwnerTable:  table.Name,
		OwnerColumn: column.Name,
	}

	// Add the sequence to the schema
	schema := e.getOrCreateSchema(schemaName)
	if schema.Sequences == nil {
		schema.Sequences = []*storepb.SequenceMetadata{}
	}
	schema.Sequences = append(schema.Sequences, sequence)

	// Store in the sequences map for reference
	sequenceKey := fmt.Sprintf("%s.%s", schemaName, sequenceName)
	e.sequences[sequenceKey] = sequence
}

// extractSchemaAndEnumName extracts schema and enum name from Any_name context
func (e *metadataExtractor) extractSchemaAndEnumName(ctx parser.IAny_nameContext) (string, string) {
	if ctx == nil {
		return e.currentSchema, ""
	}

	parts := pgparser.NormalizePostgreSQLAnyName(ctx)
	if len(parts) == 1 {
		return e.currentSchema, parts[0]
	} else if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return e.currentSchema, ""
}

// extractStringConstant extracts string value from Sconst context
func extractStringConstant(ctx parser.ISconstContext) string {
	if ctx == nil {
		return ""
	}
	// Get the text and remove surrounding quotes
	text := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	if len(text) >= 2 && text[0] == '\'' && text[len(text)-1] == '\'' {
		// Remove surrounding single quotes and handle escaped quotes
		result := text[1 : len(text)-1]
		result = strings.ReplaceAll(result, "''", "'")
		return result
	}
	return text
}

// TODO: Add support for more PostgreSQL constructs if needed
// (e.g., triggers, materialized views, custom types, etc.)
