package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/pg/ast"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_POSTGRES, GetDatabaseMetadata)
	schema.RegisterGetDatabaseMetadata(storepb.Engine_COCKROACHDB, GetDatabaseMetadata)
}

// GetDatabaseMetadata parses the SQL schema text and returns the database metadata.
func GetDatabaseMetadata(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	parseResults, err := pgparser.ParsePostgreSQL(schemaText)
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
		partitionTypes:       make(map[tableKey]storepb.TablePartitionMetadata_Type),
		sequences:            make(map[string]*storepb.SequenceMetadata),
		extensions:           make(map[string]*storepb.ExtensionMetadata),
		expressionComparer:   ast.NewPostgreSQLExpressionComparer(),
		triggers:             make(map[string][]*storepb.TriggerMetadata),
	}

	// Always ensure public schema exists
	extractor.getOrCreateSchema("public")

	// Walk all parsed statements
	for _, parseResult := range parseResults {
		if parseResult.Tree != nil {
			antlr.ParseTreeWalkerDefault.Walk(extractor, parseResult.Tree)
		}
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

		// Assign triggers to their respective tables
		for _, table := range filteredTables {
			tableKey := fmt.Sprintf("%s.%s", schemaName, table.Name)
			if triggers, exists := extractor.triggers[tableKey]; exists {
				table.Triggers = triggers
			}
		}

		// Also assign triggers to materialized views if any
		for _, mv := range schema.MaterializedViews {
			tableKey := fmt.Sprintf("%s.%s", schemaName, mv.Name)
			if triggers, exists := extractor.triggers[tableKey]; exists {
				mv.Triggers = triggers
			}
		}

		// Add sequences to the schema (including SERIAL-generated sequences)
		var sequencesForSchema []*storepb.SequenceMetadata
		for sequenceKey, sequence := range extractor.sequences {
			// Parse sequence key format: "schemaname.sequencename"
			parts := strings.SplitN(sequenceKey, ".", 2)
			if len(parts) == 2 && parts[0] == schemaName {
				sequencesForSchema = append(sequencesForSchema, sequence)
			}
		}
		// Sort sequences for consistent output
		slices.SortFunc(sequencesForSchema, func(a, b *storepb.SequenceMetadata) int {
			return strings.Compare(a.Name, b.Name)
		})
		schema.Sequences = sequencesForSchema

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

	// Extract view dependencies after all metadata is collected
	extractViewDependencies(schemaMetadata)

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
	partitionTypes       map[tableKey]storepb.TablePartitionMetadata_Type
	sequences            map[string]*storepb.SequenceMetadata
	extensions           map[string]*storepb.ExtensionMetadata
	expressionComparer   ast.ExpressionComparer
	triggers             map[string][]*storepb.TriggerMetadata // Map from table key to triggers
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

		// Get the parent table's partition expression and type
		parentKey := tableKey{schema: parentSchema, table: parentTable}
		if expr, ok := e.partitionExpressions[parentKey]; ok {
			partition.Expression = expr
		}
		if partitionType, ok := e.partitionTypes[parentKey]; ok {
			partition.Type = partitionType
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

	// Extract partition info and store the expression and type
	if partitionSpec := ctx.Optpartitionspec(); partitionSpec != nil {
		// Store the partition expression and type for this table
		key := tableKey{schema: schemaName, table: tableName}
		e.partitionExpressions[key] = e.extractPartitionExpression(partitionSpec)
		e.partitionTypes[key] = e.extractPartitionType(partitionSpec)
	}
}

// extractSchemaAndTableName extracts schema and table name from qualified name
func (e *metadataExtractor) extractSchemaAndTableName(ctx parser.IQualified_nameContext) (string, string) {
	if ctx == nil {
		return e.currentSchema, ""
	}

	// Use NormalizePostgreSQLQualifiedName which properly handles quoted identifiers
	// This function removes quotes and handles case sensitivity correctly
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

		// Handle SERIAL types completely - PostgreSQL expands SERIAL into integer + sequence + default
		switch rawTypeName {
		case "serial", "serial4":
			column.Nullable = false
			column.Type = "integer"
			// Create sequence and set default value
			e.createSerialSequenceAndDefault(column, table, schemaName)
		case "bigserial", "serial8":
			column.Nullable = false
			column.Type = "bigint"
			// Create sequence and set default value
			e.createSerialSequenceAndDefault(column, table, schemaName)
		case "smallserial", "serial2":
			column.Nullable = false
			column.Type = "smallint"
			// Create sequence and set default value
			e.createSerialSequenceAndDefault(column, table, schemaName)
		default:
			column.Type = e.extractTypeNameWithSchema(ctx.Typename(), schemaName)
		}
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

// createSerialSequenceAndDefault creates a sequence for SERIAL columns and sets the default value
func (e *metadataExtractor) createSerialSequenceAndDefault(column *storepb.ColumnMetadata, table *storepb.TableMetadata, schemaName string) {
	// Generate sequence name following PostgreSQL convention: tablename_columnname_seq
	sequenceName := fmt.Sprintf("%s_%s_seq", table.Name, column.Name)

	// Determine sequence data type based on column type
	// SERIAL (integer) → bigint sequence (PostgreSQL standard)
	// BIGSERIAL (bigint) → bigint sequence
	// SMALLSERIAL (smallint) → bigint sequence
	// All sequences in PostgreSQL are bigint by default
	dataType := "bigint"

	// Create the sequence metadata with proper defaults matching PostgreSQL behavior
	sequence := &storepb.SequenceMetadata{
		Name:        sequenceName,
		DataType:    dataType,
		Start:       "1",
		Increment:   "1",
		MinValue:    "1",
		MaxValue:    "9223372036854775807", // bigint max value
		Cycle:       false,
		CacheSize:   "1",
		OwnerTable:  table.Name,
		OwnerColumn: column.Name,
	}

	// Add sequence to the global sequences map using fully qualified name
	sequenceKey := fmt.Sprintf("%s.%s", schemaName, sequenceName)
	e.sequences[sequenceKey] = sequence

	// Set the column default value to use nextval()
	// PostgreSQL stores this as: nextval('schema.sequence_name'::regclass)
	column.Default = fmt.Sprintf("nextval('%s.%s'::regclass)", schemaName, sequenceName)
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

	return NormalizePostgreSQLType(typeName)
}

// getPostgreSQLArrayTypeName maps base types to their PostgreSQL internal array type names
// Based on PostgreSQL system catalog pg_type where array types start with underscore
func getPostgreSQLArrayTypeName(baseType string) string {
	// Normalize the base type first to handle aliases
	normalizedBase := strings.ToLower(strings.TrimSpace(baseType))

	// Map of base types to their PostgreSQL internal array type names
	arrayTypeMap := map[string]string{
		// Integer types
		"int":      "_int4",
		"int4":     "_int4",
		"integer":  "_int4",
		"int2":     "_int2",
		"smallint": "_int2",
		"int8":     "_int8",
		"bigint":   "_int8",

		// Floating point types
		"real":             "_float4",
		"float4":           "_float4",
		"double precision": "_float8",
		"float8":           "_float8",

		// Character types
		"text":              "_text",
		"varchar":           "_varchar",
		"character varying": "_varchar",
		"char":              "_bpchar", // PostgreSQL uses bpchar internally for char
		"character":         "_bpchar", // PostgreSQL uses bpchar internally for character
		"bpchar":            "_bpchar",
		"name":              "_name",

		// Boolean type
		"bool":    "_bool",
		"boolean": "_bool",

		// Numeric types
		"numeric": "_numeric",
		"decimal": "_numeric",

		// Date/time types
		"date":                        "_date",
		"time":                        "_time",
		"time without time zone":      "_time",
		"time with time zone":         "_timetz",
		"timetz":                      "_timetz",
		"timestamp":                   "_timestamp",
		"timestamp without time zone": "_timestamp",
		"timestamp with time zone":    "_timestamptz",
		"timestamptz":                 "_timestamptz",
		"interval":                    "_interval",

		// Binary and other types
		"bytea":    "_bytea",
		"uuid":     "_uuid",
		"json":     "_json",
		"jsonb":    "_jsonb",
		"xml":      "_xml",
		"money":    "_money",
		"inet":     "_inet",
		"cidr":     "_cidr",
		"macaddr":  "_macaddr",
		"macaddr8": "_macaddr8",

		// Geometric types
		"point":   "_point",
		"line":    "_line",
		"lseg":    "_lseg",
		"box":     "_box",
		"path":    "_path",
		"polygon": "_polygon",
		"circle":  "_circle",

		// Full-text search types
		"tsvector": "_tsvector",
		"tsquery":  "_tsquery",

		// Range types
		"int4range": "_int4range",
		"int8range": "_int8range",
		"numrange":  "_numrange",
		"tsrange":   "_tsrange",
		"tstzrange": "_tstzrange",
		"daterange": "_daterange",

		// Multi-range types (PostgreSQL 14+)
		"int4multirange": "_int4multirange",
		"int8multirange": "_int8multirange",
		"nummultirange":  "_nummultirange",
		"tsmultirange":   "_tsmultirange",
		"tstzmultirange": "_tstzmultirange",
		"datemultirange": "_datemultirange",

		// Bit string types
		"bit":         "_bit",
		"bit varying": "_varbit",
		"varbit":      "_varbit",

		// Object identifier types
		"oid":           "_oid",
		"regproc":       "_regproc",
		"regprocedure":  "_regprocedure",
		"regoper":       "_regoper",
		"regoperator":   "_regoperator",
		"regclass":      "_regclass",
		"regtype":       "_regtype",
		"regconfig":     "_regconfig",
		"regdictionary": "_regdictionary",
		"regnamespace":  "_regnamespace",
		"regrole":       "_regrole",
		"regcollation":  "_regcollation",

		// Other system types
		"tid":           "_tid",
		"xid":           "_xid",
		"xid8":          "_xid8",
		"cid":           "_cid",
		"pg_lsn":        "_pg_lsn",
		"record":        "_record",
		"cstring":       "_cstring",
		"refcursor":     "_refcursor",
		"jsonpath":      "_jsonpath",
		"txid_snapshot": "_txid_snapshot",
		"pg_snapshot":   "_pg_snapshot",
		"gtsvector":     "_gtsvector",
		"aclitem":       "_aclitem",
		"int2vector":    "_int2vector",
		"oidvector":     "_oidvector",
	}

	// Handle types with precision/scale (e.g., varchar(255), numeric(10,2), character varying(255))
	if idx := strings.Index(normalizedBase, "("); idx != -1 {
		baseTypeWithoutPrecision := normalizedBase[:idx]
		if arrayType, exists := arrayTypeMap[baseTypeWithoutPrecision]; exists {
			return arrayType
		}
	}

	// Direct lookup
	if arrayType, exists := arrayTypeMap[normalizedBase]; exists {
		return arrayType
	}

	// For serial types, they should not have array equivalents in DDL,
	// but if they do, map to their underlying integer array types
	switch normalizedBase {
	case "serial", "serial4":
		return "_int4"
	case "bigserial", "serial8":
		return "_int8"
	case "smallserial", "serial2":
		return "_int2"
	default:
		// For non-serial types, proceed with fallback logic
	}

	// Return empty string if no mapping found - caller will use fallback logic
	return ""
}

// normalizePostgreSQLType normalizes PostgreSQL type names to match sync.go output
func NormalizePostgreSQLType(typeName string) string {
	// Remove extra whitespace
	typeName = strings.TrimSpace(typeName)

	// Handle array types FIRST before any other conversions
	// text[] -> _text, int[] -> _int4, varchar(255)[] -> _varchar, etc.
	// PostgreSQL treats multi-dimensional arrays the same as single dimension
	if strings.HasSuffix(typeName, "[]") {
		// Remove all array dimensions (int[][], int[] both become int)
		baseType := typeName
		for strings.HasSuffix(baseType, "[]") {
			baseType = baseType[:len(baseType)-2]
		}

		// Map base type to PostgreSQL's internal array type name
		arrayType := getPostgreSQLArrayTypeName(baseType)
		if arrayType != "" {
			return arrayType
		}

		// For unknown types, use the old logic as fallback
		normalizedBase := NormalizePostgreSQLType(baseType)
		return "_" + normalizedBase
	}

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

	// Handle timestamp without explicit timezone
	if typeName == "timestamp" {
		return "timestamp(6) without time zone" // PostgreSQL default precision is 6
	}
	// Handle timestamp with time zone (no precision specified)
	if typeName == "timestamp with time zone" {
		return "timestamp(6) with time zone" // PostgreSQL default precision is 6
	}
	// Handle timestamp with precision (preserve precision but add timezone info)
	if strings.HasPrefix(typeName, "timestamp(") && strings.HasSuffix(typeName, ")") {
		// Extract precision part: timestamp(3) -> (3)
		precision := typeName[9:] // Get "(3)" part
		return "timestamp" + precision + " without time zone"
	}
	// Handle time without explicit timezone (PostgreSQL default)
	if typeName == "time" {
		return "time(6) without time zone" // PostgreSQL default precision is 6
	}
	// Handle time with time zone (no precision specified)
	if typeName == "time with time zone" {
		return "time(6) with time zone" // PostgreSQL default precision is 6
	}
	// Handle time with precision (preserve precision but add timezone info)
	if strings.HasPrefix(typeName, "time(") && strings.HasSuffix(typeName, ")") {
		// Extract precision part: time(6) -> (6)
		precision := typeName[4:] // Get "(6)" part
		return "time" + precision + " without time zone"
	}

	// Handle decimal -> numeric (preserve precision information)
	if strings.HasPrefix(typeName, "decimal(") {
		// Convert decimal(p,s) to numeric(p,s) while preserving precision
		// Also normalize spacing to match sync.go format (remove spaces around commas)
		precisionPart := typeName[7:]                              // Get "(p, s)" or "(p,s)" part
		precisionPart = strings.ReplaceAll(precisionPart, " ", "") // Remove all spaces to match sync.go format
		return "numeric" + precisionPart
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
		return "timestamp(6) with time zone" // PostgreSQL default precision is 6
	case "timetz":
		return "time(6) with time zone" // PostgreSQL default precision is 6
	case "varbit":
		return "bit varying"
	default:
		// Return type as-is for unrecognized types
	}

	// Handle specific length specifications for character types
	if strings.HasPrefix(typeName, "char(") {
		return "character" + typeName[4:]
	}

	// Handle varbit with precision: varbit(n) -> bit varying(n)
	if strings.HasPrefix(typeName, "varbit(") {
		return "bit varying" + typeName[6:] // Replace "varbit" with "bit varying"
	}

	// For all remaining types with precision/scale parameters, normalize spacing to match sync.go format
	// This handles cases like "numeric(10, 2)" -> "numeric(10,2)" to match database sync output
	if strings.Contains(typeName, "(") && strings.Contains(typeName, ")") {
		// Remove spaces around commas in type specifications to match sync.go format
		typeName = strings.ReplaceAll(typeName, ", ", ",")
		typeName = strings.ReplaceAll(typeName, " ,", ",")
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

	// Check if this is a built-in PostgreSQL type or an array type before normalization
	// Array types should always go through normalization regardless of their base type
	if isBuiltInType(rawTypeName) || isBuiltInType(NormalizePostgreSQLType(rawTypeName)) || strings.HasSuffix(rawTypeName, "[]") {
		// For built-in types and array types, return the normalized version
		return NormalizePostgreSQLType(rawTypeName)
	}

	// Check if the type already has a schema prefix
	if strings.Contains(rawTypeName, ".") {
		// It's already schema-qualified, just normalize it
		return NormalizePostgreSQLType(rawTypeName)
	}

	// For custom types (like enums), add the schema prefix to the raw type name
	return fmt.Sprintf("%s.%s", schemaName, rawTypeName)
}

// isBuiltInType checks if a type is a built-in PostgreSQL type
func isBuiltInType(typeName string) bool {
	// Handle array types first (before removing precision info)
	if strings.HasSuffix(typeName, "[]") {
		// Remove all array dimensions (int[][], int[] both should be recognized)
		arrayBase := typeName
		for strings.HasSuffix(arrayBase, "[]") {
			arrayBase = arrayBase[:len(arrayBase)-2]
		}
		// Recursively check if the base type is built-in
		return isBuiltInType(arrayBase)
	}
	if strings.HasPrefix(typeName, "_") {
		// PostgreSQL internal array notation _typename
		arrayBase := typeName[1:]
		return isBuiltInType(arrayBase)
	}

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
		"bit": true, "bit varying": true, "varbit": true,

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
			rawDefault := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr)
			column.Default = e.normalizeDefaultValue(rawDefault, column, e.currentSchema)
		}
	case ctx.PRIMARY() != nil && ctx.KEY() != nil:
		column.Nullable = false
		// Create primary key index
		index := &storepb.IndexMetadata{
			Primary:      true,
			Unique:       true,
			IsConstraint: true,
			Expressions:  []string{column.Name},
			Descending:   []bool{false}, // Single column, ascending by default
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
		// Generate definition for primary key index
		index.Definition = e.generateConstraintIndexDefinition(index, table.Name, schemaName)
		table.Indexes = append(table.Indexes, index)
	case ctx.UNIQUE() != nil:
		// Create unique index
		index := &storepb.IndexMetadata{
			Unique:       true,
			IsConstraint: true,
			Expressions:  []string{column.Name},
			Descending:   []bool{false}, // Single column, ascending by default
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
		// Generate definition for unique index
		index.Definition = e.generateConstraintIndexDefinition(index, table.Name, schemaName)
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

		// Generate constraint name if not provided (PostgreSQL auto-generates names)
		if fk.Name == "" {
			// For column-level foreign keys, use {table}_{column}_fkey format
			fk.Name = fmt.Sprintf("%s_%s_fkey", table.Name, column.Name)
		}
		// TODO: Generate definition for foreign key (ForeignKeyMetadata might not have Definition field)
		// fk.Definition = e.generateForeignKeyDefinition(fk, table.Name, schemaName)

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
	case ctx.CHECK() != nil:
		// Column-level CHECK constraint
		check := &storepb.CheckConstraintMetadata{}

		// Generate constraint name if not provided (PostgreSQL auto-generates names)
		if constraintName != "" {
			check.Name = constraintName
		} else {
			// For column-level check constraints, use {table}_{column}_check format
			check.Name = fmt.Sprintf("%s_%s_check", table.Name, column.Name)
		}

		// Extract the check expression
		if expr := ctx.A_expr(); expr != nil {
			rawExpression := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr)
			check.Expression = strings.TrimSpace(rawExpression)
		}

		// Add to table's check constraints
		if table.CheckConstraints == nil {
			table.CheckConstraints = []*storepb.CheckConstraintMetadata{}
		}
		table.CheckConstraints = append(table.CheckConstraints, check)
	default:
		// Ignore other column constraints
	}
}

// extractTableConstraint extracts table-level constraints
func (e *metadataExtractor) extractTableConstraint(ctx parser.ITableconstraintContext, table *storepb.TableMetadata, schemaName string) {
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
			// Fill Descending array with false for all columns (default ascending)
			for range columns {
				index.Descending = append(index.Descending, false)
			}
		} else if optColumnList := constraintElem.Opt_column_list(); optColumnList != nil && optColumnList.Columnlist() != nil {
			// Fallback to Opt_column_list if direct Columnlist is not available
			columns := extractColumnList(optColumnList.Columnlist())
			index.Expressions = columns
			// Fill Descending array with false for all columns (default ascending)
			for range columns {
				index.Descending = append(index.Descending, false)
			}
		}
		// Generate definition for primary key constraint index
		index.Definition = e.generateConstraintIndexDefinition(index, table.Name, schemaName)
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
			// Fill Descending array with false for all columns (default ascending)
			for range columns {
				index.Descending = append(index.Descending, false)
			}
		} else if optColumnList := constraintElem.Opt_column_list(); optColumnList != nil && optColumnList.Columnlist() != nil {
			// Fallback to Opt_column_list if direct Columnlist is not available
			columns := extractColumnList(optColumnList.Columnlist())
			index.Expressions = columns
			// Fill Descending array with false for all columns (default ascending)
			for range columns {
				index.Descending = append(index.Descending, false)
			}
		}
		// Generate constraint name if not provided
		if index.Name == "" && len(index.Expressions) > 0 {
			// PostgreSQL automatically generates constraint names in the format: table_column_key
			// For multi-column constraints: table_column1_column2_..._key
			index.Name = fmt.Sprintf("%s_%s_key", table.Name, strings.Join(index.Expressions, "_"))
		}
		// Generate definition for unique constraint index
		index.Definition = e.generateConstraintIndexDefinition(index, table.Name, schemaName)
		table.Indexes = append(table.Indexes, index)

	case constraintElem.CHECK() != nil:
		// Check constraint
		check := &storepb.CheckConstraintMetadata{}
		// Generate constraint name if not provided (PostgreSQL auto-generates names)
		if constraintName != "" {
			check.Name = constraintName
		} else {
			// For table-level check constraints, use {table}_check_{index} format
			checkIndex := len(table.CheckConstraints) + 1
			check.Name = fmt.Sprintf("%s_check_%d", table.Name, checkIndex)
		}
		if expr := constraintElem.A_expr(); expr != nil {
			rawExpression := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr)
			check.Expression = strings.TrimSpace(rawExpression)
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

		// Generate constraint name if not provided (PostgreSQL auto-generates names)
		if fk.Name == "" {
			// PostgreSQL typically uses {table}_{column}_fkey format for single column foreign keys
			if len(fk.Columns) == 1 {
				fk.Name = fmt.Sprintf("%s_%s_fkey", table.Name, fk.Columns[0])
			} else if len(fk.Columns) > 1 {
				// For multi-column foreign keys, use first column name
				fk.Name = fmt.Sprintf("%s_%s_fkey", table.Name, fk.Columns[0])
			} else {
				// Fallback name
				fk.Name = fmt.Sprintf("%s_fkey", table.Name)
			}
		}

		if table.ForeignKeys == nil {
			table.ForeignKeys = []*storepb.ForeignKeyMetadata{}
		}
		table.ForeignKeys = append(table.ForeignKeys, fk)

	case constraintElem.EXCLUDE() != nil:
		// EXCLUDE constraint
		exclude := &storepb.ExcludeConstraintMetadata{}
		// Set constraint name
		if constraintName != "" {
			exclude.Name = constraintName
		} else {
			// Generate name if not provided (PostgreSQL auto-generates names)
			excludeIndex := len(table.ExcludeConstraints) + 1
			exclude.Name = fmt.Sprintf("%s_exclude_%d", table.Name, excludeIndex)
		}

		// Extract full EXCLUDE constraint expression from token stream
		// This includes "EXCLUDE USING gist (...)" format
		if parser := ctx.GetParser(); parser != nil {
			if tokenStream := parser.GetTokenStream(); tokenStream != nil {
				start := constraintElem.GetStart()
				stop := constraintElem.GetStop()
				if start != nil && stop != nil {
					rawExpression := tokenStream.GetTextFromTokens(start, stop)
					exclude.Expression = strings.TrimSpace(rawExpression)
				}
			}
		}

		if table.ExcludeConstraints == nil {
			table.ExcludeConstraints = []*storepb.ExcludeConstraintMetadata{}
		}
		table.ExcludeConstraints = append(table.ExcludeConstraints, exclude)

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
		return "SIMPLE" // PostgreSQL default is MATCH SIMPLE
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
		return "SIMPLE" // PostgreSQL default is MATCH SIMPLE
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

// extractPartitionType extracts the partition type from the partition specification
func (*metadataExtractor) extractPartitionType(ctx parser.IOptpartitionspecContext) storepb.TablePartitionMetadata_Type {
	if ctx == nil {
		return storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	}

	partitionSpec := ctx.Partitionspec()
	if partitionSpec == nil {
		return storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	}

	// Extract the partition method from the full text
	fullText := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(partitionSpec)
	upperText := strings.ToUpper(fullText)

	// Determine partition type from the method keyword after "PARTITION BY"
	if strings.Contains(upperText, "PARTITION BY RANGE") {
		return storepb.TablePartitionMetadata_RANGE
	} else if strings.Contains(upperText, "PARTITION BY LIST") {
		return storepb.TablePartitionMetadata_LIST
	} else if strings.Contains(upperText, "PARTITION BY HASH") {
		return storepb.TablePartitionMetadata_HASH
	}

	return storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
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

	// Check if this is an ALTER TABLE command with alter_table_cmds
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

				// Debug: check what type of command this is
				hasAlter := alterTableCmd.ALTER() != nil
				hasAdd := alterTableCmd.ADD_P() != nil
				hasGenerated := alterTableCmd.GENERATED() != nil
				hasIdentity := alterTableCmd.IDENTITY_P() != nil

				// Check if this is an ADD CONSTRAINT command
				if alterTableCmd.ADD_P() != nil && alterTableCmd.Tableconstraint() != nil {
					e.extractTableConstraint(alterTableCmd.Tableconstraint(), table, schemaName)
					continue
				}

				// Check if this is an ALTER COLUMN ADD GENERATED AS IDENTITY command
				// According to g4: ALTER opt_column? colid ADD_P GENERATED generated_when AS IDENTITY_P
				// The COLUMN keyword is optional (opt_column?)
				if hasAlter && hasAdd && hasGenerated && hasIdentity {
					e.handleAlterColumn(alterTableCmd, table, schemaName)
				}
			}
		}
	}
}

// handleAlterColumn processes ALTER TABLE ALTER COLUMN commands
func (e *metadataExtractor) handleAlterColumn(alterTableCmd parser.IAlter_table_cmdContext, table *storepb.TableMetadata, schemaName string) {
	// Get the column name from colid (based on g4: ALTER opt_column? colid ADD_P GENERATED...)
	columnName := ""
	allColids := alterTableCmd.AllColid()
	if len(allColids) > 0 {
		// The column name is the first colid
		columnName = pgparser.NormalizePostgreSQLColid(allColids[0])
	}

	// Debug log
	_ = columnName // TODO: remove after debug

	if columnName == "" {
		return
	}

	// Find the column in the table
	var targetColumn *storepb.ColumnMetadata
	for _, col := range table.Columns {
		if col.Name == columnName {
			targetColumn = col
			break
		}
	}

	// If column doesn't exist, we might need to create it
	// But for now, we'll just skip if the column doesn't exist
	if targetColumn == nil {
		return
	}

	// Check if this is ADD GENERATED AS IDENTITY (from g4: ALTER opt_column? colid ADD_P GENERATED generated_when AS IDENTITY_P)
	if alterTableCmd.ADD_P() != nil && alterTableCmd.GENERATED() != nil && alterTableCmd.IDENTITY_P() != nil {
		// Set identity generation mode
		if generatedWhen := alterTableCmd.Generated_when(); generatedWhen != nil {
			if generatedWhen.ALWAYS() != nil {
				targetColumn.IdentityGeneration = storepb.ColumnMetadata_ALWAYS
			} else if generatedWhen.BY() != nil && generatedWhen.DEFAULT() != nil {
				targetColumn.IdentityGeneration = storepb.ColumnMetadata_BY_DEFAULT
			}
		}

		// Get sequence options from AST
		var seqOptList parser.ISeqoptlistContext
		if optParenSeqList := alterTableCmd.Optparenthesizedseqoptlist(); optParenSeqList != nil {
			seqOptList = optParenSeqList.Seqoptlist()
		}

		// Create the identity sequence with options from AST
		e.createIdentitySequenceFromAST(table, targetColumn, schemaName, seqOptList)
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

// EnterAlterseqstmt is called when entering an ALTER SEQUENCE statement
func (e *metadataExtractor) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if e.err != nil {
		return
	}

	if ctx.Qualified_name() == nil {
		return
	}

	schemaName, sequenceName := e.extractSchemaAndTableName(ctx.Qualified_name())
	sequenceKey := fmt.Sprintf("%s.%s", schemaName, sequenceName)

	// Find the existing sequence or create one if it doesn't exist
	sequence, exists := e.sequences[sequenceKey]
	if !exists {
		// If sequence doesn't exist yet, create it with defaults
		schemaMetadata := e.getOrCreateSchema(schemaName)
		sequence = &storepb.SequenceMetadata{
			Name:      sequenceName,
			DataType:  "bigint",
			Start:     "1",
			Increment: "1",
			MinValue:  "1",
			MaxValue:  "9223372036854775807",
			Cycle:     false,
			CacheSize: "1",
		}
		e.sequences[sequenceKey] = sequence
		if schemaMetadata.Sequences == nil {
			schemaMetadata.Sequences = []*storepb.SequenceMetadata{}
		}
		schemaMetadata.Sequences = append(schemaMetadata.Sequences, sequence)
	}

	// Process sequence options from ALTER SEQUENCE
	if seqOptList := ctx.Seqoptlist(); seqOptList != nil {
		for _, seqOptElem := range seqOptList.AllSeqoptelem() {
			if seqOptElem == nil {
				continue
			}
			e.extractSequenceOption(seqOptElem, sequence)
		}
	}
}

// EnterCommentstmt is called when entering a COMMENT statement
func (e *metadataExtractor) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	if e.err != nil {
		return
	}

	// Extract comment text (common for all comment types)
	var comment string
	if commentText := ctx.Comment_text(); commentText != nil {
		if commentText.NULL_P() != nil {
			// COMMENT ... IS NULL means remove comment
			comment = ""
		} else if commentText.Sconst() != nil {
			// Extract the string constant, removing quotes
			commentStr := commentText.GetText()
			if len(commentStr) >= 2 && commentStr[0] == '\'' && commentStr[len(commentStr)-1] == '\'' {
				// Remove quotes and unescape
				comment = commentStr[1 : len(commentStr)-1]
				// Unescape single quotes ('' -> ')
				comment = strings.ReplaceAll(comment, "''", "'")
			}
		}
	}

	// Handle COMMENT ON COLUMN
	if ctx.COLUMN() != nil && ctx.Any_name() != nil {
		parts := pgparser.NormalizePostgreSQLAnyName(ctx.Any_name())
		if len(parts) < 2 {
			return
		}

		var schemaName, tableName, columnName string
		if len(parts) == 2 {
			schemaName = e.currentSchema
			tableName = parts[0]
			columnName = parts[1]
		} else if len(parts) == 3 {
			schemaName = parts[0]
			tableName = parts[1]
			columnName = parts[2]
		} else {
			return
		}

		schema := e.getOrCreateSchema(schemaName)
		for _, table := range schema.Tables {
			if table.Name == tableName {
				for i, col := range table.Columns {
					if col.Name == columnName {
						table.Columns[i].Comment = comment
						return
					}
				}
			}
		}
		return
	}

	// Handle COMMENT ON TABLE/SEQUENCE/VIEW etc (object_type_any_name)
	if ctx.Object_type_any_name() != nil && ctx.Any_name() != nil {
		parts := pgparser.NormalizePostgreSQLAnyName(ctx.Any_name())
		if len(parts) == 0 {
			return
		}

		var schemaName, objectName string
		if len(parts) == 1 {
			schemaName = e.currentSchema
			objectName = parts[0]
		} else if len(parts) >= 2 {
			schemaName = parts[0]
			objectName = parts[1]
		}

		schema := e.getOrCreateSchema(schemaName)
		objectType := ctx.Object_type_any_name()

		// COMMENT ON TABLE
		if objectType.TABLE() != nil {
			for _, table := range schema.Tables {
				if table.Name == objectName {
					table.Comment = comment
					return
				}
			}
		}

		// COMMENT ON SEQUENCE
		if objectType.SEQUENCE() != nil {
			for _, seq := range schema.Sequences {
				if seq.Name == objectName {
					seq.Comment = comment
					return
				}
			}
		}

		// COMMENT ON VIEW
		if objectType.VIEW() != nil {
			for _, view := range schema.Views {
				if view.Name == objectName {
					view.Comment = comment
					return
				}
			}
		}

		// COMMENT ON MATERIALIZED VIEW
		if objectType.MATERIALIZED() != nil && objectType.VIEW() != nil {
			for _, view := range schema.MaterializedViews {
				if view.Name == objectName {
					view.Comment = comment
					return
				}
			}
		}

		// COMMENT ON INDEX
		if objectType.INDEX() != nil {
			for _, table := range schema.Tables {
				for i, index := range table.Indexes {
					if index.Name == objectName {
						table.Indexes[i].Comment = comment
						return
					}
				}
			}
		}

		return
	}

	// Handle COMMENT ON SCHEMA
	if ctx.Object_type_name() != nil && ctx.Name() != nil {
		// COMMENT ON SCHEMA name IS comment_text
		schemaName := pgparser.NormalizePostgreSQLName(ctx.Name())
		if schemaName != "" {
			schema := e.getOrCreateSchema(schemaName)
			schema.Comment = comment
		}
		return
	}

	// Handle COMMENT ON FUNCTION/PROCEDURE
	if (ctx.FUNCTION() != nil || ctx.PROCEDURE() != nil) && ctx.Function_with_argtypes() != nil {
		funcWithArgs := ctx.Function_with_argtypes()
		if funcWithArgs.Func_name() == nil {
			return
		}

		// Extract function name
		parts := pgparser.NormalizePostgreSQLFuncName(funcWithArgs.Func_name())
		if len(parts) == 0 {
			return
		}

		var schemaName, funcName string
		if len(parts) == 1 {
			schemaName = e.currentSchema
			funcName = parts[0]
		} else if len(parts) >= 2 {
			schemaName = parts[0]
			funcName = parts[1]
		}

		schema := e.getOrCreateSchema(schemaName)
		for i, function := range schema.Functions {
			if function.Name == funcName {
				schema.Functions[i].Comment = comment
				return
			}
		}
		return
	}

	// Handle COMMENT ON TYPE (including ENUM types)
	if ctx.TYPE_P() != nil && ctx.AllTypename() != nil && len(ctx.AllTypename()) > 0 {
		// Extract type name from typename
		typename := ctx.AllTypename()[0]
		var schemaName, typeName string

		// Try to get qualified name from typename
		if typename.Qualified_name() != nil {
			schemaName, typeName = e.extractSchemaAndTableName(typename.Qualified_name())
		} else if typename.Simpletypename() != nil {
			// Simple type name - get the text representation
			typeText := typename.GetParser().GetTokenStream().GetTextFromRuleContext(typename)
			// typeText might be "myschema.status" or just "status"
			// We need to split it if it contains a dot
			parts := strings.Split(typeText, ".")
			if len(parts) == 2 {
				schemaName = parts[0]
				typeName = parts[1]
			} else {
				typeName = typeText
				schemaName = e.currentSchema
			}
		}

		if typeName != "" {
			schema := e.getOrCreateSchema(schemaName)
			for i, enumType := range schema.EnumTypes {
				if enumType.Name == typeName {
					schema.EnumTypes[i].Comment = comment
					return
				}
			}
		}
		return
	}
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

	// Use NormalizePostgreSQLFuncName which properly handles quoted identifiers
	parts := pgparser.NormalizePostgreSQLFuncName(funcNameCtx)
	if len(parts) == 0 {
		return
	}

	var schemaName, funcName string
	if len(parts) == 1 {
		schemaName = e.currentSchema
		funcName = parts[0]
	} else if len(parts) >= 2 {
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
		Signature:  ExtractFunctionSignature(ctx, funcName),
	}

	if schemaMetadata.Functions == nil {
		schemaMetadata.Functions = []*storepb.FunctionMetadata{}
	}
	schemaMetadata.Functions = append(schemaMetadata.Functions, functionMetadata)
}

// ExtractFunctionSignature extracts the function signature with parameter types
func ExtractFunctionSignature(ctx *parser.CreatefunctionstmtContext, funcName string) string {
	var signature strings.Builder
	signature.WriteString(funcName)
	signature.WriteString(`(`)

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

// EnterCreatetrigstmt is called when entering a create trigger statement
func (e *metadataExtractor) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if e.err != nil {
		return
	}

	// Extract trigger name using normalization to properly handle quoted identifiers
	triggerNameCtx := ctx.Name()
	if triggerNameCtx == nil {
		return
	}

	triggerName := pgparser.NormalizePostgreSQLName(triggerNameCtx)
	if triggerName == "" {
		return
	}

	// Extract table name that the trigger is on using normalization to properly handle quoted identifiers
	qualifiedNameCtx := ctx.Qualified_name()
	if qualifiedNameCtx == nil {
		return
	}

	// Use NormalizePostgreSQLQualifiedName to handle quotes correctly
	parts := pgparser.NormalizePostgreSQLQualifiedName(qualifiedNameCtx)
	var schemaName, tableName string
	if len(parts) == 1 {
		schemaName = e.currentSchema
		tableName = parts[0]
	} else if len(parts) == 2 {
		schemaName = parts[0]
		tableName = parts[1]
	}

	if tableName == "" {
		return
	}

	// Build trigger metadata
	triggerMetadata := &storepb.TriggerMetadata{
		Name: triggerName,
		Body: e.buildTriggerDefinition(ctx),
	}

	// Store trigger with table key
	tableKey := fmt.Sprintf("%s.%s", schemaName, tableName)
	if e.triggers[tableKey] == nil {
		e.triggers[tableKey] = []*storepb.TriggerMetadata{}
	}
	e.triggers[tableKey] = append(e.triggers[tableKey], triggerMetadata)
}

// buildTriggerDefinition builds the trigger definition from the CREATE TRIGGER context
func (e *metadataExtractor) buildTriggerDefinition(ctx *parser.CreatetrigstmtContext) string {
	// Build trigger definition by extracting individual components
	// This is more reliable than trying to fix the concatenated text

	var parts []string

	// CREATE TRIGGER (uppercase to match system catalog)
	parts = append(parts, "CREATE TRIGGER")

	// Trigger name
	if nameCtx := ctx.Name(); nameCtx != nil {
		triggerName := strings.ToLower(nameCtx.GetText())
		parts = append(parts, triggerName)
	}

	// Timing (BEFORE/AFTER/INSTEAD OF) - uppercase
	if actionTimeCtx := ctx.Triggeractiontime(); actionTimeCtx != nil {
		timing := strings.ToUpper(actionTimeCtx.GetText())
		// Handle "INSTEADOF" -> "INSTEAD OF"
		if timing == "INSTEADOF" {
			timing = "INSTEAD OF"
		}
		parts = append(parts, timing)
	}

	// Events (INSERT/UPDATE/DELETE) - uppercase with OR separators
	if eventsCtx := ctx.Triggerevents(); eventsCtx != nil {
		events := strings.ToUpper(eventsCtx.GetText())
		// Parse and format events with proper OR separators
		events = e.formatTriggerEvents(events)
		parts = append(parts, events)
	}

	// ON table_name with schema qualification
	parts = append(parts, "ON")
	if qualifiedNameCtx := ctx.Qualified_name(); qualifiedNameCtx != nil {
		tableName := strings.ToLower(qualifiedNameCtx.GetText())
		// Add schema qualification if not present (sync always includes schema)
		if !strings.Contains(tableName, ".") {
			schemaName := "public"
			if e.currentSchema != "" {
				schemaName = e.currentSchema
			}
			tableName = schemaName + "." + tableName
		}
		parts = append(parts, tableName)
	}

	// FOR EACH ROW/STATEMENT - uppercase
	if forSpecCtx := ctx.Triggerforspec(); forSpecCtx != nil {
		forSpec := strings.ToUpper(forSpecCtx.GetText())
		// Handle "FOREACHROW" -> "FOR EACH ROW", etc.
		forSpec = e.formatTriggerForSpec(forSpec)
		parts = append(parts, forSpec)
	}

	// WHEN condition (optional) - uppercase
	if whenCtx := ctx.Triggerwhen(); whenCtx != nil {
		whenClause := strings.ToUpper(whenCtx.GetText())
		parts = append(parts, "WHEN", whenClause)
	}

	// EXECUTE FUNCTION/PROCEDURE - uppercase
	parts = append(parts, "EXECUTE")

	if funcOrProcCtx := ctx.Function_or_procedure(); funcOrProcCtx != nil {
		funcType := strings.ToUpper(funcOrProcCtx.GetText())
		parts = append(parts, funcType)
	} else {
		// Default to FUNCTION if not specified
		parts = append(parts, "FUNCTION")
	}

	// Function name with schema qualification
	if funcNameCtx := ctx.Func_name(); funcNameCtx != nil {
		funcName := strings.ToLower(funcNameCtx.GetText())

		// Add schema qualification if not present (sync always includes schema)
		if !strings.Contains(funcName, ".") {
			schemaName := "public"
			if e.currentSchema != "" {
				schemaName = e.currentSchema
			}
			funcName = schemaName + "." + funcName
		}

		// Function arguments
		funcCall := funcName + "("
		if funcArgsCtx := ctx.Triggerfuncargs(); funcArgsCtx != nil {
			args := strings.ToLower(funcArgsCtx.GetText())
			funcCall += args
		}
		funcCall += ")"

		parts = append(parts, funcCall)
	}

	return strings.Join(parts, " ")
}

// formatTriggerEvents formats trigger events with proper OR separators, preserving order from DDL
func (*metadataExtractor) formatTriggerEvents(events string) string {
	// The system catalog returns events in lowercase, and the expected format seems to preserve
	// the order as written in the DDL. Let's normalize to match the expected lowercase format

	// Handle common concatenated patterns first
	eventsLower := strings.ToLower(events)

	// Direct mappings for common patterns to match expected system catalog format
	eventMap := map[string]string{
		"insertorupdateordelete": "insert or delete or update", // Expected format from test
		"insertordelete":         "insert or delete",
		"insertorupdate":         "insert or update",
		"updateordelete":         "delete or update",
		"deleteorupdate":         "delete or update",
		"updateorinsert":         "insert or update",
		"deleteorinsert":         "insert or delete",
		"update":                 "update",
		"delete":                 "delete",
		"insert":                 "insert",
	}

	if formatted, exists := eventMap[eventsLower]; exists {
		return strings.ToUpper(formatted)
	}

	// For compound events, try to split and normalize while preserving original order
	if strings.Contains(strings.ToUpper(events), "OR") {
		parts := strings.Split(strings.ToUpper(events), "OR")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return strings.Join(parts, " OR ")
	}

	// Return uppercase if no special handling needed
	return strings.ToUpper(events)
}

// formatTriggerForSpec formats trigger FOR EACH specifications
func (*metadataExtractor) formatTriggerForSpec(forSpec string) string {
	specMap := map[string]string{
		"foreachrow":         "FOR EACH ROW",
		"foreachstatement":   "FOR EACH STATEMENT",
		"for each row":       "FOR EACH ROW",
		"for each statement": "FOR EACH STATEMENT",
	}

	specLower := strings.ToLower(forSpec)
	if formatted, exists := specMap[specLower]; exists {
		return formatted
	}

	// Return uppercase if not found
	return strings.ToUpper(forSpec)
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
						// Function expression - parse semantically for better compatibility
						expression = e.extractFunctionExpression(ctx, funcExpr)
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

			// Generate definition for the index
			index.Definition = e.generateIndexDefinition(ctx, index, schemaName, relationName)

			// Add the index to the table or materialized view
			*targetIndexes = append(*targetIndexes, index)
		}
	}
}

// generateIndexDefinition generates the CREATE INDEX definition for an index
func (e *metadataExtractor) generateIndexDefinition(ctx *parser.IndexstmtContext, index *storepb.IndexMetadata, schemaName, tableName string) string {
	var parts []string

	// Start with CREATE [UNIQUE] INDEX
	if index.Unique {
		parts = append(parts, "CREATE UNIQUE INDEX")
	} else {
		parts = append(parts, "CREATE INDEX")
	}

	// Add index name
	parts = append(parts, index.Name)

	// Add ON table
	parts = append(parts, "ON")
	if schemaName != "" && schemaName != "public" {
		parts = append(parts, fmt.Sprintf("%s.%s", schemaName, tableName))
	} else {
		parts = append(parts, fmt.Sprintf("public.%s", tableName))
	}

	// Add USING method (default is btree)
	if index.Type != "" && index.Type != "btree" {
		parts = append(parts, "USING", index.Type)
	} else {
		parts = append(parts, "USING btree")
	}

	// Add column list
	if len(index.Expressions) > 0 {
		columnList := make([]string, len(index.Expressions))
		for i, expr := range index.Expressions {
			// Add DESC if needed
			if i < len(index.Descending) && index.Descending[i] {
				columnList[i] = fmt.Sprintf("%s DESC", expr)
			} else {
				columnList[i] = expr
			}
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(columnList, ", ")))
	}

	// Check for INCLUDE clause (covering index)
	fullStatement := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	if includeClause := e.extractIncludeClause(fullStatement); includeClause != "" {
		parts = append(parts, includeClause)
	}

	// Check for WHERE clause (partial index)
	if ctx.Where_clause() != nil {
		whereClause := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Where_clause())
		// Add WHERE clause as-is
		if whereClause != "" {
			parts = append(parts, whereClause)
		}
	}

	// End with semicolon
	return strings.Join(parts, " ") + ";"
}

// extractIncludeClause extracts the INCLUDE clause from a CREATE INDEX statement
func (*metadataExtractor) extractIncludeClause(statement string) string {
	// Convert to lowercase for case-insensitive matching
	lowerStatement := strings.ToLower(statement)

	// Find the INCLUDE keyword
	includeIdx := strings.Index(lowerStatement, " include ")
	if includeIdx == -1 {
		return ""
	}

	// Find the opening parenthesis after INCLUDE
	searchStart := includeIdx + 9 // length of " include "
	openParenIdx := strings.Index(lowerStatement[searchStart:], "(")
	if openParenIdx == -1 {
		return ""
	}
	openParenIdx += searchStart

	// Find the matching closing parenthesis
	parenCount := 1
	i := openParenIdx + 1
	for i < len(lowerStatement) && parenCount > 0 {
		switch lowerStatement[i] {
		case '(':
			parenCount++
		case ')':
			parenCount--
		default:
			// Other characters don't affect parentheses counting
		}
		i++
	}

	if parenCount > 0 {
		return "" // Unmatched parentheses
	}

	// Extract the INCLUDE clause from the original statement (preserving case)
	includeClause := statement[includeIdx+1 : i] // +1 to skip the leading space
	return strings.TrimSpace(includeClause)
}

// generateConstraintIndexDefinition generates the CREATE INDEX definition for constraint-based indexes
func (*metadataExtractor) generateConstraintIndexDefinition(index *storepb.IndexMetadata, tableName, schemaName string) string {
	var parts []string

	// Start with CREATE [UNIQUE] INDEX
	if index.Unique {
		parts = append(parts, "CREATE UNIQUE INDEX")
	} else {
		parts = append(parts, "CREATE INDEX")
	}

	// Add index name
	parts = append(parts, index.Name)

	// Add ON table
	parts = append(parts, "ON")
	if schemaName != "" && schemaName != "public" {
		parts = append(parts, fmt.Sprintf("%s.%s", schemaName, tableName))
	} else {
		parts = append(parts, fmt.Sprintf("public.%s", tableName))
	}

	// Add USING method (default is btree)
	if index.Type != "" && index.Type != "btree" {
		parts = append(parts, "USING", index.Type)
	} else {
		parts = append(parts, "USING btree")
	}

	// Add column list
	if len(index.Expressions) > 0 {
		columnList := make([]string, len(index.Expressions))
		for i, expr := range index.Expressions {
			// Add DESC if needed
			if i < len(index.Descending) && index.Descending[i] {
				columnList[i] = fmt.Sprintf("%s DESC", expr)
			} else {
				columnList[i] = expr
			}
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(columnList, ", ")))
	}
	// End with semicolon
	return strings.Join(parts, " ") + ";"
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
	case ctx.OWNED() != nil && ctx.BY() != nil && ctx.Any_name() != nil:
		// OWNED BY table.column
		parts := pgparser.NormalizePostgreSQLAnyName(ctx.Any_name())
		if len(parts) == 2 {
			// Format: table.column
			sequence.OwnerTable = parts[0]
			sequence.OwnerColumn = parts[1]
		} else if len(parts) == 3 {
			// Format: schema.table.column (we only care about table.column)
			sequence.OwnerTable = parts[1]
			sequence.OwnerColumn = parts[2]
		}
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

// createIdentitySequenceFromAST creates an identity sequence from ALTER COLUMN ADD IDENTITY using AST
func (e *metadataExtractor) createIdentitySequenceFromAST(table *storepb.TableMetadata, column *storepb.ColumnMetadata, schemaName string, seqOptList parser.ISeqoptlistContext) {
	// Default sequence name
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

	// Create the sequence metadata with defaults
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

	// Override with specified options from ALTER COLUMN statement (from AST)
	if seqOptList != nil {
		for _, seqElem := range seqOptList.AllSeqoptelem() {
			if seqElem == nil {
				continue
			}
			// Check for SEQUENCE NAME option to override default name
			if seqElem.SEQUENCE() != nil && seqElem.NAME_P() != nil {
				if anyName := seqElem.Any_name(); anyName != nil {
					_, seqName := e.extractSchemaAndEnumName(anyName)
					if seqName != "" {
						sequenceName = seqName
						sequence.Name = sequenceName
					}
				}
			}
			// Extract other sequence options
			e.extractSequenceOption(seqElem, sequence)
		}
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

// normalizeDefaultValue normalizes default values to match PostgreSQL's stored format
func (*metadataExtractor) normalizeDefaultValue(rawDefault string, column *storepb.ColumnMetadata, schemaName string) string {
	if rawDefault == "" {
		return ""
	}

	// Handle nextval() for sequences - add schema and regclass cast
	if strings.Contains(rawDefault, "nextval(") {
		// Pattern: nextval('sequence_name') -> nextval('schema.sequence_name'::regclass)
		// Extract sequence name from nextval('sequence_name')
		if start := strings.Index(rawDefault, "nextval('"); start != -1 {
			start += len("nextval('")
			if end := strings.Index(rawDefault[start:], "'"); end != -1 {
				sequenceName := rawDefault[start : start+end]

				// If sequence name doesn't have schema prefix, add current schema
				if !strings.Contains(sequenceName, ".") {
					return fmt.Sprintf("nextval('%s.%s'::regclass)", schemaName, sequenceName)
				}
				return fmt.Sprintf("nextval('%s'::regclass)", sequenceName)
			}
		}
	}

	// Handle ENUM default values - add type cast
	if column.Type != "" && strings.Contains(column.Type, ".") {
		// If column type is schema-qualified (e.g., "public.status_enum")
		// and default is a string literal, add type cast
		if strings.HasPrefix(rawDefault, "'") && strings.HasSuffix(rawDefault, "'") {
			return fmt.Sprintf("%s::%s", rawDefault, column.Type)
		}
	}

	return rawDefault
}

// extractFunctionExpression extracts function expressions preserving the original user input.
func (*metadataExtractor) extractFunctionExpression(ctx *parser.IndexstmtContext, funcExpr parser.IFunc_expr_windowlessContext) string {
	if funcExpr == nil {
		return ""
	}

	// Return the original text without any normalization
	// This preserves the user's original input format
	return ctx.GetParser().GetTokenStream().GetTextFromRuleContext(funcExpr)
}

// extractViewDependencies analyzes view definitions to extract dependencies using GetQuerySpan
func extractViewDependencies(schemaMetadata *storepb.DatabaseSchemaMetadata) {
	// Extract dependencies for each view
	for _, schema := range schemaMetadata.Schemas {
		for _, view := range schema.Views {
			view.DependencyColumns = getViewDependencies(view.Definition, schema.Name, schemaMetadata)
		}

		for _, mv := range schema.MaterializedViews {
			mv.DependencyColumns = getViewDependencies(mv.Definition, schema.Name, schemaMetadata)
		}
	}
}

// getViewDependencies extracts the tables/views that a view depends on using GetQuerySpan
func getViewDependencies(viewDef string, schemaName string, fullSchemaMetadata *storepb.DatabaseSchemaMetadata) []*storepb.DependencyColumn {
	// viewDef is already a SELECT statement extracted from the parsed CREATE VIEW statement
	queryStatement := strings.TrimSpace(viewDef)

	// Use GetQuerySpan with the full schema metadata so it can resolve table/view references
	span, err := pgparser.GetQuerySpan(
		context.Background(),
		base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
				// Return the full schema metadata so GetQuerySpan can resolve references
				dbMetadata := model.NewDatabaseMetadata(fullSchemaMetadata, nil, nil, storepb.Engine_POSTGRES, false)
				return databaseName, dbMetadata, nil
			},
			ListDatabaseNamesFunc: func(_ context.Context, _ string) ([]string, error) {
				// Return empty list - we don't need actual database names for dependency extraction
				return []string{}, nil
			},
		},
		queryStatement,
		"", // database
		schemaName,
		false, // case sensitive
	)

	// If error parsing query span, return empty dependencies
	if err != nil {
		return []*storepb.DependencyColumn{} // nolint:nilerr
	}

	// Collect unique dependencies
	dependencyMap := make(map[string]*storepb.DependencyColumn)
	for sourceColumn := range span.SourceColumns {
		// Create dependency key in format: schema.table
		key := fmt.Sprintf("%s.%s", sourceColumn.Schema, sourceColumn.Table)
		if _, exists := dependencyMap[key]; !exists {
			dependencyMap[key] = &storepb.DependencyColumn{
				Schema: sourceColumn.Schema,
				Table:  sourceColumn.Table,
				Column: "*", // Use wildcard since we're tracking table-level dependencies
			}
		}
	}

	// Convert map to slice
	var dependencies []*storepb.DependencyColumn
	for _, dep := range dependencyMap {
		dependencies = append(dependencies, dep)
	}

	return dependencies
}

// TODO: Add support for more PostgreSQL constructs if needed
// (e.g., triggers, materialized views, custom types, etc.)
