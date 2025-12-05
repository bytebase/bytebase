package mssql

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/tsql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_MSSQL, GetDatabaseMetadata)
}

// GetDatabaseMetadata parses the SQL schema text and returns the database metadata.
func GetDatabaseMetadata(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	parseResults, err := tsql.ParseTSQL(schemaText)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse SQL schema")
	}

	// Handle empty schema (no statements) by returning empty metadata with default dbo schema
	if len(parseResults) == 0 {
		return &storepb.DatabaseSchemaMetadata{
			Name: "",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name:   "dbo",
					Tables: []*storepb.TableMetadata{},
				},
			},
		}, nil
	}

	extractor := &metadataExtractor{
		currentDatabase: "",
		currentSchema:   "dbo", // Default schema for MSSQL
		schemas:         make(map[string]*storepb.SchemaMetadata),
		tables:          make(map[tableKey]*storepb.TableMetadata),
	}

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
		schemaName = "dbo" // Default schema for MSSQL
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

// EnterCreate_spatial_index is called when entering a create spatial index parse tree node
func (e *metadataExtractor) EnterCreate_spatial_index(ctx *parser.Create_spatial_indexContext) {
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
		Type:         "SPATIAL",
		Expressions:  []string{},
		Descending:   []bool{},
		IsConstraint: false,
		SpatialConfig: &storepb.SpatialIndexConfig{
			Method: "SPATIAL",
		},
	}

	// Index name
	idList := ctx.AllId_()
	if len(idList) > 0 {
		index.Name, _ = tsql.NormalizeTSQLIdentifier(idList[0])
	}

	// Extract column - spatial indexes have a single column
	if len(idList) > 1 {
		colName, _ := tsql.NormalizeTSQLIdentifier(idList[1])
		index.Expressions = append(index.Expressions, colName)
		index.Descending = append(index.Descending, false)
	}

	// Parse spatial index configuration
	e.parseSpatialIndexConfig(ctx, index)

	tableMetadata.Indexes = append(tableMetadata.Indexes, index)
}

func (e *metadataExtractor) parseSpatialIndexConfig(ctx *parser.Create_spatial_indexContext, index *storepb.IndexMetadata) {
	// Initialize tessellation config
	index.SpatialConfig.Tessellation = &storepb.TessellationConfig{}

	// Parse tessellation scheme using ANTLR context
	if tessellationCtx := ctx.Spatial_tessellation_scheme(); tessellationCtx != nil {
		scheme := tessellationCtx.GetText()
		// Remove "USING" prefix if present
		scheme = strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(scheme), "USING"))
		index.SpatialConfig.Tessellation.Scheme = scheme
	}

	// Parse WITH clause parameters using ANTLR context
	if optionsCtx := ctx.Spatial_index_options(); optionsCtx != nil {
		e.parseSpatialIndexWithClause(optionsCtx, index)
	}

	// Set dimensional configuration
	if index.SpatialConfig.Dimensional == nil {
		index.SpatialConfig.Dimensional = &storepb.DimensionalConfig{
			Dimensions: 2, // SQL Server spatial indexes are always 2D
		}
	}

	// Determine data type based on tessellation scheme
	if index.SpatialConfig.Tessellation != nil {
		if strings.Contains(index.SpatialConfig.Tessellation.Scheme, "GEOGRAPHY") {
			index.SpatialConfig.Dimensional.DataType = "GEOGRAPHY"
		} else {
			index.SpatialConfig.Dimensional.DataType = "GEOMETRY"
		}
	}
}

func (*metadataExtractor) parseSpatialIndexWithClause(optionsCtx parser.ISpatial_index_optionsContext, index *storepb.IndexMetadata) {
	if index.SpatialConfig.Tessellation == nil {
		index.SpatialConfig.Tessellation = &storepb.TessellationConfig{}
	}

	// Initialize storage config for spatial index options
	if index.SpatialConfig.Storage == nil {
		index.SpatialConfig.Storage = &storepb.StorageConfig{}
	}

	// ALLOW_ROW_LOCKS and ALLOW_PAGE_LOCKS default to ON for spatial indexes
	// Set defaults first
	index.SpatialConfig.Storage.AllowRowLocks = true
	index.SpatialConfig.Storage.AllowPageLocks = true

	// Parse each spatial index option
	optionsList := optionsCtx.AllSpatial_index_option()
	for _, optionCtx := range optionsList {
		parseSpatialIndexOption(optionCtx, index)
	}
}

func parseSpatialIndexOption(optionCtx parser.ISpatial_index_optionContext, index *storepb.IndexMetadata) {
	// Handle BOUNDING_BOX
	if optionCtx.BOUNDING_BOX() != nil {
		coords := optionCtx.AllSigned_decimal()
		if len(coords) == 4 {
			xmin, _ := strconv.ParseFloat(coords[0].GetText(), 64)
			ymin, _ := strconv.ParseFloat(coords[1].GetText(), 64)
			xmax, _ := strconv.ParseFloat(coords[2].GetText(), 64)
			ymax, _ := strconv.ParseFloat(coords[3].GetText(), 64)

			index.SpatialConfig.Tessellation.BoundingBox = &storepb.BoundingBox{
				Xmin: xmin,
				Ymin: ymin,
				Xmax: xmax,
				Ymax: ymax,
			}
		}
	}

	// Handle GRIDS
	if optionCtx.GRIDS() != nil {
		var gridLevels []*storepb.GridLevel

		gridLevelsList := optionCtx.AllSpatial_grid_level()
		for _, gridLevelCtx := range gridLevelsList {
			levelText := gridLevelCtx.GetText()
			// Parse "LEVEL_1=LOW" format
			parts := strings.Split(levelText, "=")
			if len(parts) == 2 {
				levelPart := strings.TrimSpace(parts[0])
				densityPart := strings.TrimSpace(parts[1])

				// Extract level number from "LEVEL_1", "LEVEL_2", etc.
				levelNum := 0
				if strings.HasPrefix(strings.ToUpper(levelPart), "LEVEL_") {
					if num, err := strconv.Atoi(levelPart[6:]); err == nil {
						levelNum = num
					}
				}

				gridLevels = append(gridLevels, &storepb.GridLevel{
					Level:   int32(levelNum),
					Density: strings.ToUpper(densityPart),
				})
			}
		}

		index.SpatialConfig.Tessellation.GridLevels = gridLevels
	}

	// Handle CELLS_PER_OBJECT
	if optionCtx.CELLS_PER_OBJECT() != nil {
		if decimalCtx := optionCtx.DECIMAL(); decimalCtx != nil {
			if cellsPerObject, err := strconv.Atoi(decimalCtx.GetText()); err == nil {
				index.SpatialConfig.Tessellation.CellsPerObject = int32(cellsPerObject)
			}
		}
	}

	// Handle rebuild_index_option (includes standard index options)
	if rebuildCtx := optionCtx.Rebuild_index_option(); rebuildCtx != nil {
		parseRebuildIndexOption(rebuildCtx, index)
	}
}

func parseRebuildIndexOption(rebuildCtx parser.IRebuild_index_optionContext, index *storepb.IndexMetadata) {
	// Handle PAD_INDEX
	if rebuildCtx.PAD_INDEX() != nil {
		if onOffCtx := rebuildCtx.On_off(); onOffCtx != nil {
			index.SpatialConfig.Storage.PadIndex = strings.EqualFold(onOffCtx.GetText(), "ON")
		}
	}

	// Handle FILLFACTOR
	if rebuildCtx.FILLFACTOR() != nil {
		if decimalCtx := rebuildCtx.DECIMAL(); decimalCtx != nil {
			if fillFactor, err := strconv.Atoi(decimalCtx.GetText()); err == nil {
				index.SpatialConfig.Storage.Fillfactor = int32(fillFactor)
			}
		}
	}

	// Handle SORT_IN_TEMPDB
	if rebuildCtx.SORT_IN_TEMPDB() != nil {
		if onOffCtx := rebuildCtx.On_off(); onOffCtx != nil {
			index.SpatialConfig.Storage.SortInTempdb = strings.ToUpper(onOffCtx.GetText())
		}
	}

	// Handle ONLINE
	if rebuildCtx.ONLINE() != nil {
		if onOffCtx := rebuildCtx.On_off(); onOffCtx != nil {
			index.SpatialConfig.Storage.Online = strings.EqualFold(onOffCtx.GetText(), "ON")
		}
	}

	// Handle ALLOW_ROW_LOCKS
	if rebuildCtx.ALLOW_ROW_LOCKS() != nil {
		if onOffCtx := rebuildCtx.On_off(); onOffCtx != nil {
			index.SpatialConfig.Storage.AllowRowLocks = strings.EqualFold(onOffCtx.GetText(), "ON")
		}
	}

	// Handle ALLOW_PAGE_LOCKS
	if rebuildCtx.ALLOW_PAGE_LOCKS() != nil {
		if onOffCtx := rebuildCtx.On_off(); onOffCtx != nil {
			index.SpatialConfig.Storage.AllowPageLocks = strings.EqualFold(onOffCtx.GetText(), "ON")
		}
	}

	// Handle MAXDOP
	if rebuildCtx.MAXDOP() != nil {
		if decimalCtx := rebuildCtx.DECIMAL(); decimalCtx != nil {
			if maxdop, err := strconv.Atoi(decimalCtx.GetText()); err == nil {
				index.SpatialConfig.Storage.Maxdop = int32(maxdop)
			}
		}
	}

	// Handle DATA_COMPRESSION
	if rebuildCtx.DATA_COMPRESSION() != nil {
		if rebuildCtx.NONE() != nil {
			index.SpatialConfig.Storage.DataCompression = "NONE"
		} else if rebuildCtx.ROW() != nil {
			index.SpatialConfig.Storage.DataCompression = "ROW"
		} else if rebuildCtx.PAGE() != nil {
			index.SpatialConfig.Storage.DataCompression = "PAGE"
		} else if rebuildCtx.COLUMNSTORE() != nil {
			index.SpatialConfig.Storage.DataCompression = "COLUMNSTORE"
		} else if rebuildCtx.COLUMNSTORE_ARCHIVE() != nil {
			index.SpatialConfig.Storage.DataCompression = "COLUMNSTORE_ARCHIVE"
		}
	}
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

	// Set position based on current column count (1-based)
	column.Position = int32(len(table.Columns) + 1)

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
					column.Nullable = nullNotNull.NOT() == nil
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
					column.Default = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr)
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
				default:
					// Keep the default NO ACTION
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
				default:
					// Keep the default NO ACTION
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

	// Get the parser and token stream
	parser := ctx.GetParser()
	if parser == nil {
		return ctx.GetText()
	}

	tokens := parser.GetTokenStream()
	if tokens == nil {
		return ctx.GetText()
	}

	// Get the full text from the context
	fullText := tokens.GetTextFromRuleContext(ctx)

	// Remove IDENTITY specification if present
	// IDENTITY columns should have their type without the IDENTITY part
	// The IDENTITY info is stored in separate fields
	if idx := strings.Index(strings.ToUpper(fullText), "IDENTITY"); idx != -1 {
		// Extract just the data type part before IDENTITY
		dataType := strings.TrimSpace(fullText[:idx])
		return dataType
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

// EnterExecute_statement is called when entering an execute statement (like EXEC sp_addextendedproperty)
func (e *metadataExtractor) EnterExecute_statement(ctx *parser.Execute_statementContext) {
	if e.err != nil {
		return
	}

	// Check if this is an extended property statement
	e.handleExtendedProperty(ctx)
}

// handleExtendedProperty parses extended property statements and applies comments to tables/columns
func (e *metadataExtractor) handleExtendedProperty(ctx *parser.Execute_statementContext) {
	// Use parser AST instead of string manipulation
	e.parseExtendedPropertyFromAST(ctx)
}

// parseExtendedPropertyFromAST parses the extended property statement using the AST
func (e *metadataExtractor) parseExtendedPropertyFromAST(ctx *parser.Execute_statementContext) {
	// Check if this is sp_addextendedproperty by examining the procedure name
	if ctx.Execute_body() == nil {
		return
	}

	executeBody := ctx.Execute_body()
	if executeBody.Func_proc_name_server_database_schema() == nil {
		return
	}

	// Get procedure name
	procName := executeBody.Func_proc_name_server_database_schema()
	if procName.GetProcedure() == nil {
		return
	}

	procedureName, _ := tsql.NormalizeTSQLIdentifier(procName.GetProcedure())
	if !strings.EqualFold(procedureName, "sp_addextendedproperty") {
		return
	}

	// Parse the arguments using the AST
	e.parseExtendedPropertyFromExecuteBody(executeBody)
}

// EnterExecute_body is called when entering an execute_body parse tree node
func (e *metadataExtractor) EnterExecute_body(ctx *parser.Execute_bodyContext) {
	if e.err != nil {
		return
	}

	// Parse extended property statements
	e.parseExtendedPropertyFromExecuteBody(ctx)
}

// parseExtendedPropertyFromExecuteBody extracts extended property information from the execute body AST
func (e *metadataExtractor) parseExtendedPropertyFromExecuteBody(executeBody parser.IExecute_bodyContext) {
	// Use parser-based approach to extract procedure name and parameters
	if executeBody == nil {
		return
	}

	// Get the procedure name
	procNameCtx := executeBody.Func_proc_name_server_database_schema()
	if procNameCtx == nil {
		return
	}

	// Extract the procedure name and check if it's sp_addextendedproperty
	procName := e.extractProcedureName(procNameCtx)
	if !strings.EqualFold(procName, "sp_addextendedproperty") {
		return
	}

	// Extract parameters using the existing flatten function
	var args []string
	if argCtx := executeBody.Execute_statement_arg(); argCtx != nil {
		unnamedArgs := tsql.FlattenExecuteStatementArgExecuteStatementArgUnnamed(argCtx)
		for _, unnamed := range unnamedArgs {
			if value := e.extractUnnamedArgumentValue(unnamed); value != "" {
				args = append(args, value)
			}
		}
	}

	if len(args) < 6 {
		return
	}

	// Parse extended property parameters
	e.parseExtendedPropertyFromStringArgs(args)
}

// parseExtendedPropertyFromStringArgs parses extended property information from string arguments
func (e *metadataExtractor) parseExtendedPropertyFromStringArgs(args []string) {
	if len(args) < 6 {
		return
	}

	// Validate first argument is MS_Description
	if !strings.EqualFold(args[0], "MS_Description") {
		return
	}

	comment := args[1]

	// Parse level information
	var schemaName, tableName, columnName string
	for i := 2; i < len(args)-1; i += 2 {
		if i+1 >= len(args) {
			break
		}

		levelType := strings.ToUpper(args[i])
		levelName := args[i+1]

		switch levelType {
		case "SCHEMA":
			schemaName = levelName
		case "TABLE":
			tableName = levelName
		case "COLUMN":
			columnName = levelName
		default:
			// Ignore other level types
		}
	}

	if schemaName != "" && tableName != "" {
		e.applyComment(schemaName, tableName, columnName, comment)
	}
}

// extractProcedureName extracts the procedure name from the parser context
func (*metadataExtractor) extractProcedureName(ctx parser.IFunc_proc_name_server_database_schemaContext) string {
	if ctx == nil {
		return ""
	}

	// Try to get the procedure name at different levels
	if dbSchemaCtx := ctx.Func_proc_name_database_schema(); dbSchemaCtx != nil {
		if schemaCtx := dbSchemaCtx.Func_proc_name_schema(); schemaCtx != nil {
			if procName := schemaCtx.GetProcedure(); procName != nil {
				return procName.GetText()
			}
		}
	}

	// Fallback to getting the full text if structure is not as expected
	return ctx.GetText()
}

// extractUnnamedArgumentValue extracts the value from an unnamed execute statement argument
func (e *metadataExtractor) extractUnnamedArgumentValue(unnamed parser.IExecute_statement_arg_unnamedContext) string {
	if unnamed == nil {
		return ""
	}

	// Get the text directly from the context
	text := unnamed.GetText()
	return e.cleanArgumentValue(text)
}

// cleanArgumentValue cleans up an argument value by removing quotes and handling escapes
func (*metadataExtractor) cleanArgumentValue(value string) string {
	value = strings.TrimSpace(value)

	// Remove surrounding quotes if present
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		// Remove quotes and handle escaped quotes
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}

	// Handle N'string' format
	if len(value) >= 3 && strings.HasPrefix(strings.ToUpper(value), "N'") && value[len(value)-1] == '\'' {
		return strings.ReplaceAll(value[2:len(value)-1], "''", "'")
	}

	return value
}

// applyComment applies a comment to the appropriate table or column metadata
func (e *metadataExtractor) applyComment(schemaName, tableName, columnName, comment string) {
	tableKey := tableKey{schema: schemaName, table: tableName}
	tableMetadata := e.tables[tableKey]
	if tableMetadata == nil {
		// Table doesn't exist yet, create it
		tableMetadata = e.getOrCreateTable(schemaName, tableName)
	}

	if columnName != "" {
		// Column comment
		for _, col := range tableMetadata.Columns {
			if col.Name == columnName {
				col.Comment = comment
				break
			}
		}
	} else {
		// Table comment
		tableMetadata.Comment = comment
	}
}
