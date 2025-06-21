package mssql

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_MSSQL, GetDatabaseMetadata)
}

// GetDatabaseMetadata parses the SQL schema text and returns the database metadata.
func GetDatabaseMetadata(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	parseResult, err := tsql.ParseTSQL(schemaText)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse SQL schema")
	}

	if parseResult.Tree == nil {
		return nil, errors.Errorf("empty parse tree")
	}

	extractor := &metadataExtractor{
		currentDatabase: "",
		currentSchema:   defaultSchema,
		schemas:         make(map[string]*storepb.SchemaMetadata),
		tables:          make(map[tableKey]*storepb.TableMetadata),
	}

	antlr.ParseTreeWalkerDefault.Walk(extractor, parseResult.Tree)

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
		schemaMetadata.Schemas = append(schemaMetadata.Schemas, extractor.schemas[schemaName])
	}

	return schemaMetadata, nil
}

type tableKey struct {
	schema string
	table  string
}

// metadataExtractor walks the parse tree and extracts metadata
type metadataExtractor struct {
	*parser.BaseTSqlParserListener

	currentDatabase string
	currentSchema   string
	schemas         map[string]*storepb.SchemaMetadata
	tables          map[tableKey]*storepb.TableMetadata
	err             error
	indexCounter    int // Counter for generating unique index names
}

// Helper function to get or create schema
func (e *metadataExtractor) getOrCreateSchema(schemaName string) *storepb.SchemaMetadata {
	if schemaName == "" {
		schemaName = defaultSchema
	}

	if schema, exists := e.schemas[schemaName]; exists {
		return schema
	}

	schema := &storepb.SchemaMetadata{
		Name:   schemaName,
		Tables: []*storepb.TableMetadata{},
		// Initialize as nil for consistency with expected test results
		Views:      nil,
		Procedures: nil,
		Functions:  nil,
		Sequences:  nil,
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
		Name:    tableName,
		Columns: []*storepb.ColumnMetadata{},
		Indexes: []*storepb.IndexMetadata{},
		// Initialize as nil for consistency with expected test results
		ForeignKeys:      nil,
		CheckConstraints: nil,
	}

	schema := e.getOrCreateSchema(schemaName)
	schema.Tables = append(schema.Tables, table)
	e.tables[key] = table

	return table
}

// EnterCreate_schema is called when entering a create_schema parse tree node
func (e *metadataExtractor) EnterCreate_schema(ctx *parser.Create_schemaContext) {
	if e.err != nil {
		return
	}

	if ctx.GetSchema_name() != nil {
		schemaName, _ := tsql.NormalizeTSQLIdentifier(ctx.GetSchema_name())
		e.getOrCreateSchema(schemaName)
	}
}

// EnterCreate_table is called when entering a create_table parse tree node
func (e *metadataExtractor) EnterCreate_table(ctx *parser.Create_tableContext) {
	if e.err != nil {
		return
	}

	tableNameCtx := ctx.Table_name()
	if tableNameCtx == nil {
		return
	}

	schema, table := e.normalizeTableNameSeparated(tableNameCtx, e.currentDatabase, e.currentSchema)

	tableMetadata := e.getOrCreateTable(schema, table)

	// Extract columns
	if columnDefList := ctx.Column_def_table_constraints(); columnDefList != nil {
		e.extractTableElements(columnDefList, tableMetadata, schema)
	}

	// Extract table indices (including columnstore indexes)
	if tableIndices := ctx.AllTable_indices(); tableIndices != nil {
		for _, tableIndex := range tableIndices {
			e.extractTableIndex(tableIndex, tableMetadata)
		}
	}
}

// EnterCreate_index is called when entering a create index parse tree node
func (e *metadataExtractor) EnterCreate_index(ctx *parser.Create_indexContext) {
	if e.err != nil {
		return
	}

	// Extract table reference
	if ctx.Table_name() == nil {
		return
	}

	schema, table := e.normalizeTableNameSeparated(ctx.Table_name(), e.currentDatabase, e.currentSchema)

	tableMetadata := e.getOrCreateTable(schema, table)

	// Extract index metadata
	index := &storepb.IndexMetadata{
		Expressions:  []string{},
		Descending:   []bool{},
		IsConstraint: false,
	}

	// Index name
	idList := ctx.AllId_()
	if len(idList) > 0 {
		index.Name, _ = tsql.NormalizeTSQLIdentifier(idList[0])
	}

	// Index type
	if ctx.UNIQUE() != nil {
		index.Unique = true
	}
	if clustered := ctx.Clustered(); clustered != nil {
		if clustered.CLUSTERED() != nil {
			index.Type = "CLUSTERED"
		} else if clustered.NONCLUSTERED() != nil {
			index.Type = "NONCLUSTERED"
		}
	}

	// Extract columns
	if columnList := ctx.Column_name_list_with_order(); columnList != nil {
		e.extractIndexColumns(columnList, index)
	}

	tableMetadata.Indexes = append(tableMetadata.Indexes, index)
}

// EnterCreate_view is called when entering a create view parse tree node
func (e *metadataExtractor) EnterCreate_view(ctx *parser.Create_viewContext) {
	if e.err != nil {
		return
	}

	simpleNameCtx := ctx.Simple_name()
	if simpleNameCtx == nil {
		return
	}

	schema, view := e.normalizeSimpleNameSeparated(simpleNameCtx, e.currentSchema)

	schemaMetadata := e.getOrCreateSchema(schema)

	viewMetadata := &storepb.ViewMetadata{
		Name:       view,
		Definition: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
	}

	// Initialize Views slice if nil
	if schemaMetadata.Views == nil {
		schemaMetadata.Views = []*storepb.ViewMetadata{}
	}
	schemaMetadata.Views = append(schemaMetadata.Views, viewMetadata)
}

// EnterCreate_or_alter_procedure is called when entering a create procedure parse tree node
func (e *metadataExtractor) EnterCreate_or_alter_procedure(ctx *parser.Create_or_alter_procedureContext) {
	if e.err != nil {
		return
	}

	funcProcNameCtx := ctx.Func_proc_name_schema()
	if funcProcNameCtx == nil {
		return
	}

	schema, procedure := e.normalizeFuncProcNameSeparated(funcProcNameCtx, e.currentSchema)

	schemaMetadata := e.getOrCreateSchema(schema)

	procedureMetadata := &storepb.ProcedureMetadata{
		Name:       procedure,
		Definition: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
	}

	// Initialize Procedures slice if nil
	if schemaMetadata.Procedures == nil {
		schemaMetadata.Procedures = []*storepb.ProcedureMetadata{}
	}
	schemaMetadata.Procedures = append(schemaMetadata.Procedures, procedureMetadata)
}

// EnterCreate_or_alter_function is called when entering a create function parse tree node
func (e *metadataExtractor) EnterCreate_or_alter_function(ctx *parser.Create_or_alter_functionContext) {
	if e.err != nil {
		return
	}

	funcNameCtx := ctx.Func_proc_name_schema()
	if funcNameCtx == nil {
		return
	}

	schema, function := e.normalizeFuncProcNameSeparated(funcNameCtx, e.currentSchema)

	schemaMetadata := e.getOrCreateSchema(schema)

	functionMetadata := &storepb.FunctionMetadata{
		Name:       function,
		Definition: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
	}

	// Initialize Functions slice if nil
	if schemaMetadata.Functions == nil {
		schemaMetadata.Functions = []*storepb.FunctionMetadata{}
	}
	schemaMetadata.Functions = append(schemaMetadata.Functions, functionMetadata)
}

// EnterCreate_columnstore_index is called when entering a create clustered columnstore index parse tree node
func (e *metadataExtractor) EnterCreate_columnstore_index(ctx *parser.Create_columnstore_indexContext) {
	if e.err != nil {
		return
	}

	// Extract table reference
	if ctx.Table_name() == nil {
		return
	}

	schema, table := e.normalizeTableNameSeparated(ctx.Table_name(), e.currentDatabase, e.currentSchema)

	tableMetadata := e.getOrCreateTable(schema, table)

	// Extract index metadata
	index := &storepb.IndexMetadata{
		Type:         "CLUSTERED COLUMNSTORE",
		Expressions:  []string{}, // Clustered columnstore indexes don't have specific columns
		Descending:   []bool{},
		IsConstraint: false,
	}

	// Index name
	if ctx.Id_(0) != nil {
		index.Name, _ = tsql.NormalizeTSQLIdentifier(ctx.Id_(0))
	}

	tableMetadata.Indexes = append(tableMetadata.Indexes, index)
}

// EnterCreate_nonclustered_columnstore_index is called when entering a create nonclustered columnstore index parse tree node
func (e *metadataExtractor) EnterCreate_nonclustered_columnstore_index(ctx *parser.Create_nonclustered_columnstore_indexContext) {
	if e.err != nil {
		return
	}

	// Extract table reference
	if ctx.Table_name() == nil {
		return
	}

	schema, table := e.normalizeTableNameSeparated(ctx.Table_name(), e.currentDatabase, e.currentSchema)

	tableMetadata := e.getOrCreateTable(schema, table)

	// Extract index metadata
	index := &storepb.IndexMetadata{
		Type:         "NONCLUSTERED COLUMNSTORE",
		Expressions:  []string{},
		Descending:   []bool{},
		IsConstraint: false,
	}

	// Index name
	if ctx.Id_(0) != nil {
		index.Name, _ = tsql.NormalizeTSQLIdentifier(ctx.Id_(0))
	}

	// Extract columns
	if columnList := ctx.Column_name_list_with_order(); columnList != nil {
		e.extractIndexColumns(columnList, index)
	}

	tableMetadata.Indexes = append(tableMetadata.Indexes, index)
}

// EnterCreate_sequence is called when entering a create sequence parse tree node
func (e *metadataExtractor) EnterCreate_sequence(ctx *parser.Create_sequenceContext) {
	if e.err != nil {
		return
	}

	if ctx.GetSequence_name() == nil {
		return
	}

	sequenceName, _ := tsql.NormalizeTSQLIdentifier(ctx.GetSequence_name())

	// Sequences in MSSQL are schema-scoped, but the parser doesn't provide schema info
	// so we'll use the default schema
	schemaMetadata := e.getOrCreateSchema(e.currentSchema)

	sequenceMetadata := &storepb.SequenceMetadata{
		Name: sequenceName,
	}

	if dataType := ctx.Data_type(); dataType != nil {
		sequenceMetadata.DataType = extractDataType(dataType)
	}

	// Initialize Sequences slice if nil
	if schemaMetadata.Sequences == nil {
		schemaMetadata.Sequences = []*storepb.SequenceMetadata{}
	}
	schemaMetadata.Sequences = append(schemaMetadata.Sequences, sequenceMetadata)
}

// extractTableElements extracts columns and constraints from table definition
func (e *metadataExtractor) extractTableElements(ctx parser.IColumn_def_table_constraintsContext, table *storepb.TableMetadata, schemaName string) {
	if columnDefCtxList := ctx.AllColumn_def_table_constraint(); columnDefCtxList != nil {
		for _, columnDefCtx := range columnDefCtxList {
			if columnDef := columnDefCtx.Column_definition(); columnDef != nil {
				e.extractColumn(columnDef, table)
			} else if constraint := columnDefCtx.Table_constraint(); constraint != nil {
				e.extractTableConstraint(constraint, table, schemaName)
			}
		}
	}
}

// extractColumn extracts column metadata from column definition
func (e *metadataExtractor) extractColumn(ctx parser.IColumn_definitionContext, table *storepb.TableMetadata) {
	if ctx == nil {
		return
	}

	column := &storepb.ColumnMetadata{}

	// Column name
	if ctx.Id_() != nil {
		column.Name, _ = tsql.NormalizeTSQLIdentifier(ctx.Id_())
	}

	// Data type and IDENTITY handling
	if dataTypeCtx := ctx.Data_type(); dataTypeCtx != nil {
		column.Type = extractDataType(dataTypeCtx)

		// Check if IDENTITY is part of the data type context
		// The parser sometimes includes IDENTITY in the data type context
		if dataTypeCtx.IDENTITY() != nil {
			column.IsIdentity = true
			column.IdentitySeed = 1
			column.IdentityIncrement = 1

			if seed := dataTypeCtx.GetSeed(); seed != nil {
				if val, err := strconv.ParseInt(seed.GetText(), 10, 64); err == nil {
					column.IdentitySeed = val
				} else {
					e.err = errors.Wrapf(err, "failed to parse identity seed for column %s", column.Name)
				}
			}

			if increment := dataTypeCtx.GetInc(); increment != nil {
				if val, err := strconv.ParseInt(increment.GetText(), 10, 64); err == nil {
					column.IdentityIncrement = val
				} else {
					e.err = errors.Wrapf(err, "failed to parse identity increment for column %s", column.Name)
				}
			}
		}
	}

	// Nullability and other properties
	column.Nullable = true // Default to nullable

	if columnDefElems := ctx.AllColumn_definition_element(); columnDefElems != nil {
		for _, elem := range columnDefElems {
			// Handle column constraints
			if elem.Column_constraint() != nil {
				constraint := elem.Column_constraint()

				// Handle nullability
				if nullNotNull := constraint.Null_notnull(); nullNotNull != nil {
					if nullNotNull.NOT() != nil {
						column.Nullable = false
					} else {
						column.Nullable = true
					}
				}

				// Handle PRIMARY KEY
				if constraint.PRIMARY() != nil && constraint.KEY() != nil {
					column.Nullable = false // Primary keys are not nullable
					// Create a primary key index
					index := &storepb.IndexMetadata{
						Primary:      true,
						Unique:       true,
						IsConstraint: true,
						Expressions:  []string{column.Name},
						Descending:   []bool{false},
					}

					// Get constraint name if specified
					if constraint.GetConstraint() != nil {
						index.Name, _ = tsql.NormalizeTSQLIdentifier(constraint.GetConstraint())
					} else {
						// Generate a unique name for unnamed constraints
						e.indexCounter++
						index.Name = fmt.Sprintf("PK_%s_%d", table.Name, e.indexCounter)
					}

					// Check for CLUSTERED/NONCLUSTERED
					if clustered := constraint.Clustered(); clustered != nil {
						if clustered.CLUSTERED() != nil {
							index.Type = "CLUSTERED"
						} else if clustered.NONCLUSTERED() != nil {
							index.Type = "NONCLUSTERED"
						}
					}

					table.Indexes = append(table.Indexes, index)
				}

				// Handle UNIQUE
				if constraint.UNIQUE() != nil {
					// Create a unique index
					index := &storepb.IndexMetadata{
						Unique:       true,
						IsConstraint: true,
						Expressions:  []string{column.Name},
						Descending:   []bool{false},
					}

					// Get constraint name if specified
					if constraint.GetConstraint() != nil {
						index.Name, _ = tsql.NormalizeTSQLIdentifier(constraint.GetConstraint())
					} else {
						// Generate a unique name for unnamed constraints
						e.indexCounter++
						index.Name = fmt.Sprintf("UQ_%s_%d", table.Name, e.indexCounter)
					}

					// Check for CLUSTERED/NONCLUSTERED
					if clustered := constraint.Clustered(); clustered != nil {
						if clustered.CLUSTERED() != nil {
							index.Type = "CLUSTERED"
						} else if clustered.NONCLUSTERED() != nil {
							index.Type = "NONCLUSTERED"
						}
					}

					table.Indexes = append(table.Indexes, index)
				}
			}

			// Note: IDENTITY is handled in the data type context above since the parser
			// includes it as part of the data type

			// Handle DEFAULT
			if elem.DEFAULT() != nil {
				// Get the default expression (everything after DEFAULT keyword)
				// This is a simplified implementation
				if expr := elem.Expression(); expr != nil {
					column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{
						DefaultExpression: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr),
					}
				}
			}

			// Handle collation
			if elem.COLLATE() != nil && elem.Id_() != nil {
				column.Collation, _ = tsql.NormalizeTSQLIdentifier(elem.Id_())
			}
		}
	}

	table.Columns = append(table.Columns, column)
}

// extractTableConstraint extracts table-level constraints
func (e *metadataExtractor) extractTableConstraint(ctx parser.ITable_constraintContext, table *storepb.TableMetadata, schemaName string) {
	if ctx == nil {
		return
	}

	// Get constraint name
	var constraintName string
	if constraintCtx := ctx.GetConstraint(); constraintCtx != nil {
		constraintName, _ = tsql.NormalizeTSQLIdentifier(constraintCtx)
	}

	// Handle different constraint types
	if ctx.PRIMARY() != nil && ctx.KEY() != nil {
		// Primary key constraint
		index := &storepb.IndexMetadata{
			Name:         constraintName,
			Primary:      true,
			Unique:       true,
			IsConstraint: true,
			Expressions:  []string{},
			Descending:   []bool{},
		}

		if clustered := ctx.Clustered(); clustered != nil {
			if clustered.CLUSTERED() != nil {
				index.Type = "CLUSTERED"
			} else if clustered.NONCLUSTERED() != nil {
				index.Type = "NONCLUSTERED"
			}
		}

		// Extract columns
		if columnList := ctx.Column_name_list_with_order(); columnList != nil {
			e.extractIndexColumns(columnList, index)
		}

		table.Indexes = append(table.Indexes, index)
	} else if ctx.UNIQUE() != nil {
		// Unique constraint
		index := &storepb.IndexMetadata{
			Name:         constraintName,
			Unique:       true,
			IsConstraint: true,
			Expressions:  []string{},
			Descending:   []bool{},
		}

		if clustered := ctx.Clustered(); clustered != nil {
			if clustered.CLUSTERED() != nil {
				index.Type = "CLUSTERED"
			} else if clustered.NONCLUSTERED() != nil {
				index.Type = "NONCLUSTERED"
			}
		}

		// Extract columns
		if columnList := ctx.Column_name_list_with_order(); columnList != nil {
			e.extractIndexColumns(columnList, index)
		}

		table.Indexes = append(table.Indexes, index)
	} else if ctx.Check_constraint() != nil {
		// Check constraint
		check := &storepb.CheckConstraintMetadata{
			Name: constraintName,
		}

		if checkConstraint := ctx.Check_constraint(); checkConstraint != nil && checkConstraint.Search_condition() != nil {
			expr := checkConstraint.Search_condition()
			check.Expression = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr)
		}

		// Initialize CheckConstraints slice if nil
		if table.CheckConstraints == nil {
			table.CheckConstraints = []*storepb.CheckConstraintMetadata{}
		}
		table.CheckConstraints = append(table.CheckConstraints, check)
	} else if ctx.FOREIGN() != nil && ctx.KEY() != nil {
		// Foreign key constraint
		fk := &storepb.ForeignKeyMetadata{
			Name:              constraintName,
			Columns:           []string{},
			ReferencedColumns: []string{},
		}

		// Extract local columns
		if columnList := ctx.GetFk(); columnList != nil {
			if idList := columnList.AllId_(); idList != nil {
				for _, id := range idList {
					colName, _ := tsql.NormalizeTSQLIdentifier(id)
					fk.Columns = append(fk.Columns, colName)
				}
			}
		}

		// Extract referenced table and columns
		if fkOptions := ctx.Foreign_key_options(); fkOptions != nil {
			if fkOptions.Table_name() != nil {
				refSchema, refTable := e.normalizeTableNameSeparated(fkOptions.Table_name(), e.currentDatabase, schemaName)
				fk.ReferencedSchema = refSchema
				fk.ReferencedTable = refTable
			}

			if pkList := fkOptions.GetPk(); pkList != nil {
				if idList := pkList.AllId_(); idList != nil {
					for _, id := range idList {
						colName, _ := tsql.NormalizeTSQLIdentifier(id)
						fk.ReferencedColumns = append(fk.ReferencedColumns, colName)
					}
				}
			}

			// Extract ON DELETE/UPDATE actions
			fk.OnDelete = "NO ACTION" // Default
			fk.OnUpdate = "NO ACTION" // Default
			if onDelete := fkOptions.On_delete(0); onDelete != nil {
				switch {
				case onDelete.CASCADE() != nil:
					fk.OnDelete = "CASCADE"
				case onDelete.SET() != nil && onDelete.NULL_() != nil:
					fk.OnDelete = "SET NULL"
				case onDelete.SET() != nil && onDelete.DEFAULT() != nil:
					fk.OnDelete = "SET DEFAULT"
				}
			}
			if onUpdate := fkOptions.On_update(0); onUpdate != nil {
				switch {
				case onUpdate.CASCADE() != nil:
					fk.OnUpdate = "CASCADE"
				case onUpdate.SET() != nil && onUpdate.NULL_() != nil:
					fk.OnDelete = "SET NULL"
				case onUpdate.SET() != nil && onUpdate.DEFAULT() != nil:
					fk.OnUpdate = "SET DEFAULT"
				}
			}
		}

		// Initialize ForeignKeys slice if nil
		if table.ForeignKeys == nil {
			table.ForeignKeys = []*storepb.ForeignKeyMetadata{}
		}
		table.ForeignKeys = append(table.ForeignKeys, fk)
	}
}

// extractIndexColumns extracts column information for indexes
func (*metadataExtractor) extractIndexColumns(ctx parser.IColumn_name_list_with_orderContext, index *storepb.IndexMetadata) {
	if ctx == nil {
		return
	}

	if idList := ctx.AllColumn_name_with_order(); idList != nil {
		for _, id := range idList {
			colName, _ := tsql.NormalizeTSQLIdentifier(id.Id_())
			index.Expressions = append(index.Expressions, colName)

			// Check if DESC is specified - simplified, assuming ASC by default
			// The actual parser might have different structure for ORDER
			if id.DESC() != nil {
				index.Descending = append(index.Descending, true)
			} else {
				index.Descending = append(index.Descending, false)
			}
		}
	}
}

// extractTableIndex extracts index information from table_indices context
func (e *metadataExtractor) extractTableIndex(ctx parser.ITable_indicesContext, table *storepb.TableMetadata) {
	if ctx == nil {
		return
	}

	index := &storepb.IndexMetadata{
		Expressions:  []string{},
		Descending:   []bool{},
		IsConstraint: false,
	}

	// Index name
	idList := ctx.AllId_()
	if len(idList) > 0 {
		index.Name, _ = tsql.NormalizeTSQLIdentifier(idList[0])
	}

	// Check for UNIQUE
	if ctx.UNIQUE() != nil {
		index.Unique = true
	}

	// Check for index type
	if ctx.CLUSTERED() != nil && ctx.COLUMNSTORE() != nil {
		index.Type = "CLUSTERED COLUMNSTORE"
		// Clustered columnstore indexes don't have specific columns
	} else if ctx.NONCLUSTERED() != nil && ctx.COLUMNSTORE() != nil {
		index.Type = "NONCLUSTERED COLUMNSTORE"
		// Extract columns for nonclustered columnstore
		if columnList := ctx.Column_name_list(); columnList != nil {
			e.extractColumnNameList(columnList, index)
		}
	} else if ctx.COLUMNSTORE() != nil {
		// COLUMNSTORE without NONCLUSTERED means nonclustered by default
		index.Type = "NONCLUSTERED COLUMNSTORE"
		// Extract columns
		if columnList := ctx.Column_name_list(); columnList != nil {
			e.extractColumnNameList(columnList, index)
		}
	} else {
		// Regular index
		if clustered := ctx.Clustered(); clustered != nil {
			if clustered.CLUSTERED() != nil {
				index.Type = "CLUSTERED"
			} else if clustered.NONCLUSTERED() != nil {
				index.Type = "NONCLUSTERED"
			}
		}
		// Extract columns
		if columnList := ctx.Column_name_list_with_order(); columnList != nil {
			e.extractIndexColumns(columnList, index)
		}
	}

	table.Indexes = append(table.Indexes, index)
}

// extractColumnNameList extracts column names without order for columnstore indexes
func (*metadataExtractor) extractColumnNameList(ctx parser.IColumn_name_listContext, index *storepb.IndexMetadata) {
	if ctx == nil {
		return
	}

	if idList := ctx.AllId_(); idList != nil {
		for _, id := range idList {
			colName, _ := tsql.NormalizeTSQLIdentifier(id)
			index.Expressions = append(index.Expressions, colName)
			// Columnstore indexes don't have DESC/ASC specification
			index.Descending = append(index.Descending, false)
		}
	}
}

// Helper functions

func extractDataType(ctx parser.IData_typeContext) string {
	if ctx == nil {
		return ""
	}

	// The parser sometimes includes IDENTITY in the data type context
	// We need to extract just the actual data type
	fullText := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)

	// Check if IDENTITY is included
	if identityIdx := strings.Index(strings.ToUpper(fullText), "IDENTITY"); identityIdx > 0 {
		// Return just the data type part before IDENTITY
		return strings.TrimSpace(fullText[:identityIdx])
	}

	return fullText
}

// normalizeFuncProcNameSeparated extracts schema and name from func_proc_name_schema context
func (*metadataExtractor) normalizeFuncProcNameSeparated(ctx parser.IFunc_proc_name_schemaContext, defaultSchema string) (string, string) {
	schema := defaultSchema
	name := ""

	if s := ctx.GetSchema(); s != nil {
		if id, _ := tsql.NormalizeTSQLIdentifier(s); id != "" {
			schema = id
		}
	}
	if p := ctx.GetProcedure(); p != nil {
		if id, _ := tsql.NormalizeTSQLIdentifier(p); id != "" {
			name = id
		}
	}

	return schema, name
}

// normalizeTableNameSeparated extracts schema and table from table_name context
func (*metadataExtractor) normalizeTableNameSeparated(ctx parser.ITable_nameContext, _, fallbackSchemaName string) (string, string) {
	schema := fallbackSchemaName
	table := ""
	if s := ctx.GetSchema(); s != nil {
		if id, _ := tsql.NormalizeTSQLIdentifier(s); id != "" {
			schema = id
		}
	}
	if t := ctx.GetTable(); t != nil {
		if id, _ := tsql.NormalizeTSQLIdentifier(t); id != "" {
			table = id
		}
	}
	return schema, table
}

// normalizeSimpleNameSeparated extracts schema and name from simple_name context
func (*metadataExtractor) normalizeSimpleNameSeparated(ctx parser.ISimple_nameContext, fallbackSchemaName string) (string, string) {
	schema := fallbackSchemaName
	name := ""
	if s := ctx.GetSchema(); s != nil {
		if id, _ := tsql.NormalizeTSQLIdentifier(s); id != "" {
			schema = id
		}
	}
	if n := ctx.GetName(); n != nil {
		if id, _ := tsql.NormalizeTSQLIdentifier(n); id != "" {
			name = id
		}
	}
	return schema, name
}
