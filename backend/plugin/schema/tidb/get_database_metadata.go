package tidb

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/types"
	"github.com/pkg/errors"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/pkg/types/parser_driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_TIDB, GetDatabaseMetadata)
}

// GetDatabaseMetadata parses the database schema text and returns the metadata using TiDB's parser.
func GetDatabaseMetadata(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	// Use TiDB's parser to parse the SQL
	p := parser.New()
	p.EnableWindowFunc(true)
	mode, err := mysql.GetSQLMode(mysql.DefaultSQLMode)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get SQL mode")
	}
	mode = mysql.DelSQLMode(mode, mysql.ModeNoZeroDate)
	mode = mysql.DelSQLMode(mode, mysql.ModeNoZeroInDate)
	p.SetSQLMode(mode)

	stmts, _, err := p.Parse(schemaText, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse schema")
	}

	extractor := &metadataExtractor{
		schemas: make(map[string]*storepb.SchemaMetadata),
		tables:  make(map[tableKey]*storepb.TableMetadata),
		views:   make(map[viewKey]*storepb.ViewMetadata),
		result: &storepb.DatabaseSchemaMetadata{
			Name: "",
		},
	}

	// Process each statement
	for _, stmt := range stmts {
		err := extractor.processStatement(stmt)
		if err != nil {
			return nil, err
		}
	}

	// Build the final metadata structure
	defaultSchema := &storepb.SchemaMetadata{
		Name: "",
	}

	// Add tables to schema
	var tables []*storepb.TableMetadata
	for _, table := range extractor.tables {
		tables = append(tables, table)
	}
	slices.SortFunc(tables, func(a, b *storepb.TableMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
	defaultSchema.Tables = tables

	// Add views to schema
	var views []*storepb.ViewMetadata
	for _, view := range extractor.views {
		views = append(views, view)
	}
	slices.SortFunc(views, func(a, b *storepb.ViewMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
	defaultSchema.Views = views

	extractor.result.Schemas = []*storepb.SchemaMetadata{defaultSchema}

	return extractor.result, nil
}

type tableKey struct {
	schema string
	name   string
}

type viewKey struct {
	schema string
	name   string
}

type metadataExtractor struct {
	schemas map[string]*storepb.SchemaMetadata
	tables  map[tableKey]*storepb.TableMetadata
	views   map[viewKey]*storepb.ViewMetadata

	result *storepb.DatabaseSchemaMetadata
}

func (m *metadataExtractor) processStatement(stmt ast.StmtNode) error {
	switch node := stmt.(type) {
	case *ast.CreateTableStmt:
		return m.processCreateTable(node)
	case *ast.CreateViewStmt:
		return m.processCreateView(node)
	case *ast.CreateDatabaseStmt:
		return m.processCreateDatabase(node)
	// Add more statement types as needed
	default:
		// Ignore unsupported statement types
		return nil
	}
}

func (m *metadataExtractor) processCreateDatabase(stmt *ast.CreateDatabaseStmt) error {
	dbName := stmt.Name.O
	m.result.Name = dbName

	// Extract character set and collation from options
	for _, option := range stmt.Options {
		switch option.Tp {
		case ast.DatabaseOptionCharset:
			m.result.CharacterSet = option.Value
		case ast.DatabaseOptionCollate:
			m.result.Collation = option.Value
		default:
			// Ignore other database options
		}
	}

	return nil
}

func (m *metadataExtractor) processCreateTable(stmt *ast.CreateTableStmt) error {
	tableName := stmt.Table.Name.O
	schemaName := ""
	if stmt.Table.Schema.O != "" {
		schemaName = stmt.Table.Schema.O
	}

	table := &storepb.TableMetadata{
		Name:    tableName,
		Engine:  "InnoDB", // Default for TiDB
		Comment: "",
	}

	// Process columns
	for i, col := range stmt.Cols {
		column := &storepb.ColumnMetadata{
			Name:     col.Name.Name.O,
			Position: int32(i + 1),
			Nullable: true, // Default, will be overridden by constraints
		}

		// Process column type
		column.Type = m.getColumnType(col.Tp)

		// Process column options
		for _, option := range col.Options {
			switch option.Tp {
			case ast.ColumnOptionNotNull:
				column.Nullable = false
			case ast.ColumnOptionNull:
				column.Nullable = true
			case ast.ColumnOptionAutoIncrement:
				column.Default = "AUTO_INCREMENT"
			case ast.ColumnOptionDefaultValue:
				if defaultValue := m.getDefaultValue(option.Expr); defaultValue != "" {
					column.Default = defaultValue
				}
			case ast.ColumnOptionComment:
				// Handle comment - extract from various expression types
				if comment := m.extractCommentFromExpr(option.Expr); comment != "" {
					column.Comment = comment
				}
			case ast.ColumnOptionOnUpdate:
				if onUpdate := m.getExpressionValue(option.Expr); onUpdate != "" {
					column.OnUpdate = onUpdate
				}
			case ast.ColumnOptionGenerated:
				// Handle generated columns
				m.processGeneratedColumn(option, column)
			case ast.ColumnOptionPrimaryKey:
				// Mark column as primary key
				column.Nullable = false
				// We'll handle creating the PRIMARY index after processing all columns
			case ast.ColumnOptionUniqKey:
				// Handle unique constraint at column level
				// We'll handle creating unique indexes after processing all columns
			default:
				// Ignore other column options
			}
		}

		table.Columns = append(table.Columns, column)
	}

	// Create indexes for column-level constraints
	m.processColumnLevelConstraints(stmt, table)

	// Process table-level constraints
	for _, constraint := range stmt.Constraints {
		switch constraint.Tp {
		case ast.ConstraintPrimaryKey:
			expressions, keyLengths, descending := m.getIndexColumnsInfo(constraint.Keys)
			index := &storepb.IndexMetadata{
				Name:        "PRIMARY",
				Primary:     true,
				Unique:      true,
				Visible:     true,
				Type:        "BTREE",
				Expressions: expressions,
				KeyLength:   keyLengths,
				Descending:  descending,
			}
			table.Indexes = append(table.Indexes, index)

		case ast.ConstraintKey, ast.ConstraintIndex:
			expressions, keyLengths, descending := m.getIndexColumnsInfo(constraint.Keys)
			index := &storepb.IndexMetadata{
				Name:        constraint.Name,
				Primary:     false,
				Unique:      false,
				Visible:     true,
				Type:        m.getIndexType(constraint),
				Expressions: expressions,
				KeyLength:   keyLengths,
				Descending:  descending,
			}
			table.Indexes = append(table.Indexes, index)

		case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
			expressions, keyLengths, descending := m.getIndexColumnsInfo(constraint.Keys)
			index := &storepb.IndexMetadata{
				Name:        constraint.Name,
				Primary:     false,
				Unique:      true,
				Visible:     true,
				Type:        m.getIndexType(constraint),
				Expressions: expressions,
				KeyLength:   keyLengths,
				Descending:  descending,
			}
			table.Indexes = append(table.Indexes, index)

		case ast.ConstraintForeignKey:
			fk := &storepb.ForeignKeyMetadata{
				Name:              constraint.Name,
				Columns:           m.getColumnNames(constraint.Keys),
				ReferencedTable:   constraint.Refer.Table.Name.O,
				ReferencedColumns: m.getColumnNames(constraint.Refer.IndexPartSpecifications),
			}

			// Handle schema-qualified table references
			if constraint.Refer.Table.Schema.O != "" {
				fk.ReferencedSchema = constraint.Refer.Table.Schema.O
			}

			// Handle ON DELETE and ON UPDATE actions
			if constraint.Refer.OnDelete != nil {
				fk.OnDelete = m.getReferenceAction(constraint.Refer.OnDelete.ReferOpt)
			} else {
				fk.OnDelete = "NO ACTION" // Default for TiDB/MySQL
			}
			if constraint.Refer.OnUpdate != nil {
				fk.OnUpdate = m.getReferenceAction(constraint.Refer.OnUpdate.ReferOpt)
			} else {
				fk.OnUpdate = "NO ACTION" // Default for TiDB/MySQL
			}

			table.ForeignKeys = append(table.ForeignKeys, fk)

		case ast.ConstraintCheck:
			// Handle check constraints
			m.processCheckConstraint(constraint, table)
		default:
			// Ignore other constraint types
		}
	}

	// Process table options
	for _, option := range stmt.Options {
		switch option.Tp {
		case ast.TableOptionEngine:
			table.Engine = option.StrValue
		case ast.TableOptionCharset:
			table.Charset = option.StrValue
		case ast.TableOptionCollate:
			table.Collation = option.StrValue
		case ast.TableOptionComment:
			table.Comment = option.StrValue
			// Check for TiDB-specific features in the comment
			m.processTiDBTableComment(option.StrValue, table)
		case ast.TableOptionAutoIncrement:
			// Handle AUTO_INCREMENT starting value
			// Store auto increment value in comment for now
			// as TableMetadata doesn't have AutoIncrementValue field
			_ = option.UintValue // Acknowledge the value is available
		case ast.TableOptionRowFormat:
			// Handle row format (DYNAMIC, COMPACT, etc.)
			// Store in comment for now as TableMetadata doesn't have RowFormat field
		default:
			// Ignore other table options
		}
	}

	// Handle TiDB-specific AUTO_RANDOM
	m.processAutoRandom(stmt, table)

	// Ensure foreign key indexes exist (after all other indexes have been processed)
	m.ensureForeignKeyIndexes(table)

	key := tableKey{schema: schemaName, name: tableName}
	m.tables[key] = table

	return nil
}

func (m *metadataExtractor) processCreateView(stmt *ast.CreateViewStmt) error {
	viewName := stmt.ViewName.Name.O
	schemaName := ""
	if stmt.ViewName.Schema.O != "" {
		schemaName = stmt.ViewName.Schema.O
	}

	view := &storepb.ViewMetadata{
		Name:       viewName,
		Definition: m.extractViewDefinition(stmt),
		Comment:    "",
	}

	key := viewKey{schema: schemaName, name: viewName}
	m.views[key] = view

	return nil
}

// Helper methods

func (*metadataExtractor) getColumnType(tp *types.FieldType) string {
	if tp == nil {
		return ""
	}

	// Get base type name first
	var baseType string
	switch tp.GetType() {
	case mysql.TypeTiny:
		baseType = "tinyint"
	case mysql.TypeShort:
		baseType = "smallint"
	case mysql.TypeInt24:
		baseType = "mediumint"
	case mysql.TypeLong:
		baseType = "int"
	case mysql.TypeLonglong:
		baseType = "bigint"
	case mysql.TypeFloat:
		baseType = "float"
	case mysql.TypeDouble:
		baseType = "double"
	case mysql.TypeNewDecimal:
		baseType = "decimal"
	case mysql.TypeDate:
		baseType = "date"
	case mysql.TypeTimestamp:
		baseType = "timestamp"
	case mysql.TypeDatetime:
		baseType = "datetime"
	case mysql.TypeYear:
		baseType = "year"
	case mysql.TypeVarchar, mysql.TypeVarString:
		if tp.GetCharset() == "binary" {
			baseType = "varbinary"
		} else {
			baseType = "varchar"
		}
	case mysql.TypeString:
		if tp.GetCharset() == "binary" {
			baseType = "binary"
		} else {
			baseType = "char"
		}
	case mysql.TypeTinyBlob:
		if tp.GetCharset() == "binary" {
			baseType = "tinyblob"
		} else {
			baseType = "tinytext"
		}
	case mysql.TypeBlob:
		if tp.GetCharset() == "binary" {
			baseType = "blob"
		} else {
			baseType = "text"
		}
	case mysql.TypeMediumBlob:
		if tp.GetCharset() == "binary" {
			baseType = "mediumblob"
		} else {
			baseType = "mediumtext"
		}
	case mysql.TypeLongBlob:
		if tp.GetCharset() == "binary" {
			baseType = "longblob"
		} else {
			baseType = "longtext"
		}
	case mysql.TypeJSON:
		baseType = "json"
	case mysql.TypeEnum, mysql.TypeSet:
		// For ENUM and SET types, use the full string representation
		// as it includes the value list: enum('a','b','c') or set('x','y','z')
		fullType := strings.ToLower(tp.String())
		return fullType
	case mysql.TypeBit:
		baseType = "bit"
	default:
		// Fall back to string representation - preserve full type including precision/length
		fullType := strings.ToLower(tp.String())

		// Normalize common type variations for TiDB compatibility (preserve precision/length)
		if strings.HasPrefix(fullType, "integer") {
			return strings.Replace(fullType, "integer", "int", 1)
		}
		if strings.HasPrefix(fullType, "numeric") {
			return strings.Replace(fullType, "numeric", "decimal", 1)
		}
		if fullType == "boolean" || fullType == "bool" {
			return "tinyint(1)"
		}

		return fullType
	}

	// Add length/precision information for applicable types
	if tp.GetFlen() > 0 {
		switch tp.GetType() {
		case mysql.TypeVarchar, mysql.TypeVarString, mysql.TypeString:
			// VARCHAR(n), CHAR(n)
			return fmt.Sprintf("%s(%d)", baseType, tp.GetFlen())
		case mysql.TypeNewDecimal:
			// DECIMAL(precision, scale)
			if tp.GetDecimal() > 0 {
				return fmt.Sprintf("%s(%d,%d)", baseType, tp.GetFlen(), tp.GetDecimal())
			}
			return fmt.Sprintf("%s(%d)", baseType, tp.GetFlen())
		case mysql.TypeFloat, mysql.TypeDouble:
			// FLOAT(precision), DOUBLE(precision)
			if tp.GetDecimal() > 0 {
				return fmt.Sprintf("%s(%d,%d)", baseType, tp.GetFlen(), tp.GetDecimal())
			}
		case mysql.TypeBit:
			// BIT(n)
			return fmt.Sprintf("%s(%d)", baseType, tp.GetFlen())
		case mysql.TypeTiny:
			// TINYINT(n) - special case for BOOLEAN which becomes TINYINT(1)
			return fmt.Sprintf("%s(%d)", baseType, tp.GetFlen())
		case mysql.TypeYear:
			// YEAR(4) - MySQL/TiDB YEAR type
			return fmt.Sprintf("%s(%d)", baseType, tp.GetFlen())
		default:
			// Return baseType for other types that don't need length specification
		}
	}

	// Special cases for types that should always include length even if GetFlen() returns 0
	if tp.GetType() == mysql.TypeYear {
		return "year(4)" // YEAR is always YEAR(4) in MySQL/TiDB
	}

	return baseType
}

func (*metadataExtractor) getDefaultValue(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}

	switch node := expr.(type) {
	case *ast.FuncCallExpr:
		return node.FnName.O
	default:
		// For other expression types, return the text representation
		if textNode, ok := expr.(interface{ Text() string }); ok {
			text := textNode.Text()
			// Clean up common default value formats
			text = strings.Trim(text, "'\"")
			if strings.HasPrefix(text, "_utf8mb4") {
				// Remove charset prefix
				if idx := strings.Index(text, "'"); idx != -1 {
					text = text[idx+1:]
					text = strings.TrimSuffix(text, "'")
				}
			}
			return text
		}
	}
	return ""
}

func (*metadataExtractor) getExpressionValue(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}

	switch node := expr.(type) {
	case *ast.FuncCallExpr:
		return node.FnName.O
	default:
		// For other expression types, return the text representation
		if textNode, ok := expr.(interface{ Text() string }); ok {
			return textNode.Text()
		}
	}
	return ""
}

func (*metadataExtractor) getColumnNames(keys []*ast.IndexPartSpecification) []string {
	if keys == nil {
		return nil
	}

	var names []string
	for _, key := range keys {
		if key.Column != nil {
			names = append(names, key.Column.Name.O)
		}
	}
	return names
}

// getIndexColumnsInfo extracts column names and key lengths from index key specifications
func (*metadataExtractor) getIndexColumnsInfo(keys []*ast.IndexPartSpecification) (expressions []string, keyLengths []int64, descending []bool) {
	if keys == nil {
		return nil, nil, nil
	}

	var hasDescending bool

	for _, key := range keys {
		if key.Column != nil {
			expressions = append(expressions, key.Column.Name.O)

			// Extract key length if specified (for prefix indexes like MySQL)
			if key.Length > 0 {
				keyLengths = append(keyLengths, int64(key.Length))
			} else {
				keyLengths = append(keyLengths, -1) // -1 indicates no prefix length specified
			}

			// Extract descending flag
			if key.Desc {
				hasDescending = true
			}
			descending = append(descending, key.Desc)
		}
	}

	// TiDB always returns KeyLength arrays, but not Descending arrays
	// Match TiDB database behavior
	if !hasDescending {
		descending = nil
	}

	return expressions, keyLengths, descending
}

func (*metadataExtractor) getReferenceAction(action ast.ReferOptionType) string {
	switch action {
	case ast.ReferOptionCascade:
		return "CASCADE"
	case ast.ReferOptionSetNull:
		return "SET NULL"
	case ast.ReferOptionRestrict:
		return "RESTRICT"
	case ast.ReferOptionNoAction:
		return "NO ACTION"
	case ast.ReferOptionSetDefault:
		return "SET DEFAULT"
	default:
		return ""
	}
}

func (m *metadataExtractor) processTiDBTableComment(comment string, table *storepb.TableMetadata) {
	// Process TiDB-specific table comment features like PK_AUTO_RANDOM_BITS
	pkAutoRandomBitsRegex := regexp.MustCompile(`PK_AUTO_RANDOM_BITS=(\d+)`)
	if matches := pkAutoRandomBitsRegex.FindStringSubmatch(comment); len(matches) > 1 {
		// Find the primary key column and set AUTO_RANDOM
		for _, col := range table.Columns {
			if m.isPrimaryKeyColumn(col, table) {
				bits := matches[1]
				col.Default = fmt.Sprintf("AUTO_RANDOM(%s)", bits)
				break
			}
		}
	}
}

func (*metadataExtractor) processAutoRandom(_ *ast.CreateTableStmt, _ *storepb.TableMetadata) {
	// TiDB's AUTO_RANDOM is typically stored in table comments or special constraints
	// This is a simplified implementation - full AUTO_RANDOM support would require
	// parsing TiDB-specific syntax extensions
}

func (*metadataExtractor) isPrimaryKeyColumn(column *storepb.ColumnMetadata, table *storepb.TableMetadata) bool {
	for _, index := range table.Indexes {
		if index.Primary {
			for _, expr := range index.Expressions {
				if expr == column.Name {
					return true
				}
			}
		}
	}
	return false
}

// Helper methods for enhanced functionality

func (*metadataExtractor) extractCommentFromExpr(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}

	// For all expression types, try to get text representation
	if textNode, ok := expr.(interface{ Text() string }); ok {
		text := textNode.Text()
		// Remove quotes from string literals
		if (strings.HasPrefix(text, "'") && strings.HasSuffix(text, "'")) ||
			(strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"")) {
			return text[1 : len(text)-1]
		}
		return text
	}
	return ""
}

func (*metadataExtractor) processGeneratedColumn(option *ast.ColumnOption, column *storepb.ColumnMetadata) {
	if option.Expr != nil {
		// Extract the generation expression
		var generationExpr string
		if textNode, ok := option.Expr.(interface{ Text() string }); ok {
			generationExpr = textNode.Text()
		}

		// Store generation information in the type or comment
		if option.Stored {
			column.Type = column.Type + " GENERATED ALWAYS AS (" + generationExpr + ") STORED"
		} else {
			column.Type = column.Type + " GENERATED ALWAYS AS (" + generationExpr + ") VIRTUAL"
		}
	}
}

func (*metadataExtractor) getIndexType(constraint *ast.Constraint) string {
	// Default to BTREE for most indexes
	indexType := "BTREE"

	// Check for specific index types in TiDB
	if constraint.Option != nil {
		if constraint.Option.Tp == ast.IndexTypeHash {
			indexType = "HASH"
		} else if constraint.Option.Comment != "" {
			// Check for FULLTEXT or SPATIAL in comments
			comment := strings.ToUpper(constraint.Option.Comment)
			if strings.Contains(comment, "FULLTEXT") {
				indexType = "FULLTEXT"
			} else if strings.Contains(comment, "SPATIAL") {
				indexType = "SPATIAL"
			}
		}
	}

	return indexType
}

func (*metadataExtractor) processCheckConstraint(constraint *ast.Constraint, table *storepb.TableMetadata) {
	// TiDB check constraints are handled differently
	// For now, skip processing as the AST structure may not have ExprInCheck
	// In a full implementation, we would need to check the constraint structure
	_ = constraint
	_ = table
}

func (*metadataExtractor) extractViewDefinition(stmt *ast.CreateViewStmt) string {
	if stmt.Select == nil {
		return "SELECT ..."
	}

	// For a complete implementation, we would need to reconstruct the SELECT statement
	// This is a simplified version that returns a placeholder
	// In a production implementation, you would use the AST to reconstruct the full SELECT
	if textNode, ok := stmt.Select.(interface{ Text() string }); ok {
		return textNode.Text()
	}

	return "SELECT ..."
}

// processColumnLevelConstraints creates indexes for column-level PRIMARY KEY and UNIQUE constraints
func (*metadataExtractor) processColumnLevelConstraints(stmt *ast.CreateTableStmt, table *storepb.TableMetadata) {
	var primaryKeyColumns []string
	var uniqueConstraints []struct {
		columnName string
		indexName  string
	}

	// Find columns with PRIMARY KEY or UNIQUE constraints
	for i, col := range stmt.Cols {
		columnName := col.Name.Name.O

		for _, option := range col.Options {
			switch option.Tp {
			case ast.ColumnOptionPrimaryKey:
				primaryKeyColumns = append(primaryKeyColumns, columnName)
				// Mark the column as not nullable
				table.Columns[i].Nullable = false

			case ast.ColumnOptionUniqKey:
				uniqueConstraints = append(uniqueConstraints, struct {
					columnName string
					indexName  string
				}{
					columnName: columnName,
					indexName:  columnName, // Use column name as index name for single-column unique
				})
			default:
				// Ignore other column options
			}
		}
	}

	// Create PRIMARY index if we have primary key columns
	if len(primaryKeyColumns) > 0 {
		// TiDB always provides KeyLength arrays
		keyLengths := make([]int64, len(primaryKeyColumns))
		for i := range keyLengths {
			keyLengths[i] = -1 // No prefix length for column-level constraints
		}

		index := &storepb.IndexMetadata{
			Name:        "PRIMARY",
			Primary:     true,
			Unique:      true,
			Visible:     true,
			Type:        "BTREE",
			Expressions: primaryKeyColumns,
			KeyLength:   keyLengths,
			// Don't set Descending for column-level constraints to match TiDB
		}
		table.Indexes = append(table.Indexes, index)
	}

	// Create UNIQUE indexes for column-level unique constraints
	for _, unique := range uniqueConstraints {
		index := &storepb.IndexMetadata{
			Name:        unique.indexName,
			Primary:     false,
			Unique:      true,
			Visible:     true,
			Type:        "BTREE",
			Expressions: []string{unique.columnName},
			KeyLength:   []int64{-1}, // TiDB provides KeyLength arrays
			// Don't set Descending to match TiDB database behavior
		}
		table.Indexes = append(table.Indexes, index)
	}
}

// ensureForeignKeyIndexes creates indexes for all foreign key columns if they don't already exist
func (m *metadataExtractor) ensureForeignKeyIndexes(table *storepb.TableMetadata) {
	for _, fk := range table.ForeignKeys {
		m.ensureForeignKeyIndex(fk, table)
	}
}

// ensureForeignKeyIndex creates an index for foreign key columns if one doesn't already exist
func (*metadataExtractor) ensureForeignKeyIndex(fk *storepb.ForeignKeyMetadata, table *storepb.TableMetadata) {
	// Check if an index already exists that covers the foreign key columns
	fkColumns := fk.Columns

	for _, existingIndex := range table.Indexes {
		// Check if this index starts with the foreign key columns
		if len(existingIndex.Expressions) >= len(fkColumns) {
			match := true
			for i, fkCol := range fkColumns {
				if i >= len(existingIndex.Expressions) || existingIndex.Expressions[i] != fkCol {
					match = false
					break
				}
			}
			if match {
				// An index already exists that covers these columns
				return
			}
		}
	}

	// No suitable index exists, create one
	indexName := fk.Name
	if indexName == "" {
		// Generate a name based on the columns
		indexName = strings.Join(fkColumns, "_")
	}

	// TiDB provides KeyLength arrays for all indexes
	keyLengths := make([]int64, len(fkColumns))
	for i := range keyLengths {
		keyLengths[i] = -1 // No prefix length for FK indexes
	}

	index := &storepb.IndexMetadata{
		Name:        indexName,
		Primary:     false,
		Unique:      false,
		Visible:     true,
		Type:        "BTREE",
		Expressions: fkColumns,
		KeyLength:   keyLengths,
		// Don't set Descending to match TiDB database behavior
	}

	table.Indexes = append(table.Indexes, index)
}
