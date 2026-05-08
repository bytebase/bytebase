package mssql

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

const noAction = "NO ACTION"

// GetDatabaseMetadata parses the SQL schema text and returns the database metadata.
func GetDatabaseMetadata(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	return getDatabaseMetadataOmni(schemaText)
}

func getDatabaseMetadataOmni(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	stmts, err := tsql.ParseTSQLOmni(schemaText)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse SQL schema")
	}

	if len(stmts) == 0 {
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

	extractor := &omniMetadataExtractor{
		currentSchema: "dbo",
		schemas:       make(map[string]*storepb.SchemaMetadata),
		tables:        make(map[tableKey]*storepb.TableMetadata),
		schemaText:    schemaText,
	}

	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		extractor.extractStatement(stmt.AST)
	}
	if extractor.err != nil {
		return nil, extractor.err
	}

	schemaMetadata := &storepb.DatabaseSchemaMetadata{
		Name: extractor.currentDatabase,
	}

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

type omniMetadataExtractor struct {
	currentDatabase string
	currentSchema   string
	schemas         map[string]*storepb.SchemaMetadata
	tables          map[tableKey]*storepb.TableMetadata
	indexCounter    int
	schemaText      string
	err             error
}

func (e *omniMetadataExtractor) getOrCreateSchema(schemaName string) *storepb.SchemaMetadata {
	if schemaName == "" {
		schemaName = "dbo"
	}
	if schema, exists := e.schemas[schemaName]; exists {
		return schema
	}
	schema := &storepb.SchemaMetadata{
		Name:       schemaName,
		Tables:     []*storepb.TableMetadata{},
		Views:      nil,
		Procedures: nil,
		Functions:  nil,
		Sequences:  nil,
	}
	e.schemas[schemaName] = schema
	return schema
}

func (e *omniMetadataExtractor) getOrCreateTable(schemaName, tableName string) *storepb.TableMetadata {
	key := tableKey{schema: schemaName, table: tableName}
	if table, exists := e.tables[key]; exists {
		return table
	}
	table := &storepb.TableMetadata{
		Name:             tableName,
		Columns:          []*storepb.ColumnMetadata{},
		Indexes:          []*storepb.IndexMetadata{},
		ForeignKeys:      nil,
		CheckConstraints: nil,
	}
	schema := e.getOrCreateSchema(schemaName)
	schema.Tables = append(schema.Tables, table)
	e.tables[key] = table
	return table
}

func (e *omniMetadataExtractor) extractStatement(node ast.Node) {
	if node == nil || e.err != nil {
		return
	}

	switch n := node.(type) {
	case *ast.CreateSchemaStmt:
		e.getOrCreateSchema(n.Name)
		if n.Elements != nil {
			for _, item := range n.Elements.Items {
				e.extractStatement(item)
			}
		}
	case *ast.CreateTableStmt:
		e.extractCreateTable(n)
	case *ast.CreateIndexStmt:
		e.extractCreateIndex(n)
	case *ast.CreateSpatialIndexStmt:
		e.extractCreateSpatialIndex(n)
	case *ast.CreateViewStmt:
		schemaName, viewName := tableRefSchemaObject(n.Name, e.currentSchema)
		schemaMetadata := e.getOrCreateSchema(schemaName)
		if schemaMetadata.Views == nil {
			schemaMetadata.Views = []*storepb.ViewMetadata{}
		}
		schemaMetadata.Views = append(schemaMetadata.Views, &storepb.ViewMetadata{
			Name:       viewName,
			Definition: e.definitionText(n),
		})
	case *ast.CreateProcedureStmt:
		schemaName, procedureName := tableRefSchemaObject(n.Name, e.currentSchema)
		schemaMetadata := e.getOrCreateSchema(schemaName)
		if schemaMetadata.Procedures == nil {
			schemaMetadata.Procedures = []*storepb.ProcedureMetadata{}
		}
		schemaMetadata.Procedures = append(schemaMetadata.Procedures, &storepb.ProcedureMetadata{
			Name:       procedureName,
			Definition: e.definitionText(n),
		})
	case *ast.CreateFunctionStmt:
		schemaName, functionName := tableRefSchemaObject(n.Name, e.currentSchema)
		schemaMetadata := e.getOrCreateSchema(schemaName)
		if schemaMetadata.Functions == nil {
			schemaMetadata.Functions = []*storepb.FunctionMetadata{}
		}
		schemaMetadata.Functions = append(schemaMetadata.Functions, &storepb.FunctionMetadata{
			Name:       functionName,
			Definition: e.definitionText(n),
		})
	case *ast.CreateSequenceStmt:
		e.extractCreateSequence(n)
	case *ast.ExecStmt:
		e.extractExtendedProperty(n)
	default:
	}
}

func (e *omniMetadataExtractor) extractCreateTable(n *ast.CreateTableStmt) {
	schemaName, tableName := tableRefSchemaObject(n.Name, e.currentSchema)
	table := e.getOrCreateTable(schemaName, tableName)

	if n.Columns != nil {
		for _, item := range n.Columns.Items {
			column, ok := item.(*ast.ColumnDef)
			if !ok {
				continue
			}
			e.extractColumn(column, table, schemaName)
		}
	}
	if n.Constraints != nil {
		for _, item := range n.Constraints.Items {
			constraint, ok := item.(*ast.ConstraintDef)
			if !ok {
				continue
			}
			e.extractTableConstraint(constraint, table, schemaName)
		}
	}
	if n.Indexes != nil {
		for _, item := range n.Indexes.Items {
			index, ok := item.(*ast.InlineIndexDef)
			if !ok {
				continue
			}
			table.Indexes = append(table.Indexes, e.inlineIndexMetadata(index))
		}
	}
}

func (e *omniMetadataExtractor) extractColumn(n *ast.ColumnDef, table *storepb.TableMetadata, schemaName string) {
	column := &storepb.ColumnMetadata{
		Name:     n.Name,
		Position: int32(len(table.Columns) + 1),
		Nullable: true,
	}
	if n.DataType != nil {
		column.Type = e.nodeText(n.DataType)
	}
	if n.Identity != nil {
		column.IsIdentity = true
		column.IdentitySeed = n.Identity.Seed
		column.IdentityIncrement = n.Identity.Increment
	}
	if n.Nullable != nil {
		column.Nullable = !n.Nullable.NotNull
	}
	if n.DefaultExpr != nil {
		column.Default = e.nodeText(n.DefaultExpr)
	}
	if n.Collation != "" {
		column.Collation = n.Collation
	}

	if n.Constraints != nil {
		for _, item := range n.Constraints.Items {
			constraint, ok := item.(*ast.ConstraintDef)
			if !ok {
				continue
			}
			switch constraint.Type {
			case ast.ConstraintPrimaryKey:
				column.Nullable = false
				table.Indexes = append(table.Indexes, e.constraintIndexMetadata(
					constraint,
					fmt.Sprintf("PK_%s", table.Name),
					[]string{column.Name},
					true,
				))
			case ast.ConstraintUnique:
				table.Indexes = append(table.Indexes, e.constraintIndexMetadata(
					constraint,
					fmt.Sprintf("UQ_%s", table.Name),
					[]string{column.Name},
					false,
				))
			case ast.ConstraintDefault:
				if constraint.Expr != nil {
					column.Default = e.nodeText(constraint.Expr)
				}
			case ast.ConstraintCheck:
				e.appendCheckConstraint(table, constraint)
			case ast.ConstraintForeignKey:
				appendForeignKey(table, constraint, []string{column.Name}, schemaName)
			default:
			}
		}
	}

	table.Columns = append(table.Columns, column)
}

func (e *omniMetadataExtractor) extractTableConstraint(n *ast.ConstraintDef, table *storepb.TableMetadata, schemaName string) {
	switch n.Type {
	case ast.ConstraintPrimaryKey:
		table.Indexes = append(table.Indexes, e.constraintIndexMetadata(n, "", indexColumnNames(n.Columns), true))
	case ast.ConstraintUnique:
		table.Indexes = append(table.Indexes, e.constraintIndexMetadata(n, "", indexColumnNames(n.Columns), false))
	case ast.ConstraintCheck:
		e.appendCheckConstraint(table, n)
	case ast.ConstraintForeignKey:
		appendForeignKey(table, n, stringNodeList(n.Columns), schemaName)
	default:
	}
}

func (e *omniMetadataExtractor) extractCreateIndex(n *ast.CreateIndexStmt) {
	schemaName, tableName := tableRefSchemaObject(n.Table, e.currentSchema)
	table := e.getOrCreateTable(schemaName, tableName)

	index := &storepb.IndexMetadata{
		Name:         n.Name,
		Unique:       n.Unique,
		Type:         indexType(n.Columnstore, n.Clustered),
		Expressions:  []string{},
		Descending:   []bool{},
		IsConstraint: false,
	}
	appendIndexColumns(index, n.Columns)
	table.Indexes = append(table.Indexes, index)
}

func (e *omniMetadataExtractor) extractCreateSpatialIndex(n *ast.CreateSpatialIndexStmt) {
	schemaName, tableName := tableRefSchemaObject(n.Table, e.currentSchema)
	table := e.getOrCreateTable(schemaName, tableName)

	index := &storepb.IndexMetadata{
		Name:         n.Name,
		Type:         "SPATIAL",
		Expressions:  []string{},
		Descending:   []bool{},
		IsConstraint: false,
		SpatialConfig: &storepb.SpatialIndexConfig{
			Method: "SPATIAL",
			Tessellation: &storepb.TessellationConfig{
				Scheme: strings.ToUpper(n.Using),
			},
			Dimensional: &storepb.DimensionalConfig{
				Dimensions: 2,
			},
		},
	}
	if n.SpatialColumn != "" {
		index.Expressions = append(index.Expressions, n.SpatialColumn)
		index.Descending = append(index.Descending, false)
	}
	if strings.Contains(index.SpatialConfig.Tessellation.Scheme, "GEOGRAPHY") {
		index.SpatialConfig.Dimensional.DataType = "GEOGRAPHY"
	} else {
		index.SpatialConfig.Dimensional.DataType = "GEOMETRY"
	}
	parseSpatialOptions(n.Options, index)
	table.Indexes = append(table.Indexes, index)
}

func (e *omniMetadataExtractor) extractCreateSequence(n *ast.CreateSequenceStmt) {
	schemaName, sequenceName := tableRefSchemaObject(n.Name, e.currentSchema)
	schemaMetadata := e.getOrCreateSchema(schemaName)
	sequence := &storepb.SequenceMetadata{Name: sequenceName}
	if n.DataType != nil {
		sequence.DataType = e.nodeText(n.DataType)
	}
	if schemaMetadata.Sequences == nil {
		schemaMetadata.Sequences = []*storepb.SequenceMetadata{}
	}
	schemaMetadata.Sequences = append(schemaMetadata.Sequences, sequence)
}

func (e *omniMetadataExtractor) constraintIndexMetadata(n *ast.ConstraintDef, prefix string, columns []string, primary bool) *storepb.IndexMetadata {
	name := n.Name
	if name == "" && prefix != "" {
		e.indexCounter++
		name = fmt.Sprintf("%s_%d", prefix, e.indexCounter)
	}
	index := &storepb.IndexMetadata{
		Name:         name,
		Primary:      primary,
		Unique:       true,
		IsConstraint: true,
		Expressions:  []string{},
		Descending:   []bool{},
		Type:         indexType(false, n.Clustered),
	}
	for _, column := range columns {
		index.Expressions = append(index.Expressions, column)
		index.Descending = append(index.Descending, false)
	}
	if len(columns) == 0 && n.Columns != nil {
		appendIndexColumns(index, n.Columns)
	}
	return index
}

func (*omniMetadataExtractor) inlineIndexMetadata(n *ast.InlineIndexDef) *storepb.IndexMetadata {
	index := &storepb.IndexMetadata{
		Name:         n.Name,
		Unique:       n.Unique,
		Type:         indexType(n.Columnstore, n.Clustered),
		Expressions:  []string{},
		Descending:   []bool{},
		IsConstraint: false,
	}
	appendIndexColumns(index, n.Columns)
	return index
}

func (e *omniMetadataExtractor) appendCheckConstraint(table *storepb.TableMetadata, n *ast.ConstraintDef) {
	check := &storepb.CheckConstraintMetadata{
		Name:       n.Name,
		Expression: e.nodeText(n.Expr),
	}
	if table.CheckConstraints == nil {
		table.CheckConstraints = []*storepb.CheckConstraintMetadata{}
	}
	table.CheckConstraints = append(table.CheckConstraints, check)
}

func appendForeignKey(table *storepb.TableMetadata, n *ast.ConstraintDef, columns []string, fallbackSchema string) {
	fk := &storepb.ForeignKeyMetadata{
		Name:              n.Name,
		Columns:           columns,
		ReferencedColumns: stringNodeList(n.RefColumns),
		OnDelete:          referentialAction(n.OnDelete),
		OnUpdate:          referentialAction(n.OnUpdate),
	}
	if fk.OnDelete == "" {
		fk.OnDelete = noAction
	}
	if fk.OnUpdate == "" {
		fk.OnUpdate = noAction
	}
	if n.RefTable != nil {
		refSchema, refTable := tableRefSchemaObject(n.RefTable, fallbackSchema)
		fk.ReferencedSchema = refSchema
		fk.ReferencedTable = refTable
	}
	if table.ForeignKeys == nil {
		table.ForeignKeys = []*storepb.ForeignKeyMetadata{}
	}
	table.ForeignKeys = append(table.ForeignKeys, fk)
}

func (e *omniMetadataExtractor) extractExtendedProperty(n *ast.ExecStmt) {
	if n.Name == nil || !strings.EqualFold(n.Name.Object, "sp_addextendedproperty") {
		return
	}
	if n.Args == nil || len(n.Args.Items) < 6 {
		return
	}

	var args []string
	for _, item := range n.Args.Items {
		arg, ok := item.(*ast.ExecArg)
		if !ok || arg.Value == nil {
			continue
		}
		switch v := arg.Value.(type) {
		case *ast.Literal:
			args = append(args, v.Str)
		default:
			args = append(args, e.nodeText(v))
		}
	}
	if len(args) < 6 || !strings.EqualFold(args[0], "MS_Description") {
		return
	}

	comment := args[1]
	var schemaName, tableName, columnName string
	for i := 2; i < len(args)-1; i += 2 {
		switch strings.ToUpper(args[i]) {
		case "SCHEMA":
			schemaName = args[i+1]
		case "TABLE":
			tableName = args[i+1]
		case "COLUMN":
			columnName = args[i+1]
		default:
		}
	}
	if schemaName != "" && tableName != "" {
		e.applyComment(schemaName, tableName, columnName, comment)
	}
}

func (e *omniMetadataExtractor) applyComment(schemaName, tableName, columnName, comment string) {
	key := tableKey{schema: schemaName, table: tableName}
	table := e.tables[key]
	if table == nil {
		table = e.getOrCreateTable(schemaName, tableName)
	}
	if columnName != "" {
		for _, column := range table.Columns {
			if column.Name == columnName {
				column.Comment = comment
				return
			}
		}
		return
	}
	table.Comment = comment
}

func parseSpatialOptions(options *ast.List, index *storepb.IndexMetadata) {
	if options == nil {
		return
	}
	index.SpatialConfig.Storage = &storepb.StorageConfig{
		AllowRowLocks:  true,
		AllowPageLocks: true,
	}
	for _, item := range options.Items {
		option, ok := item.(*ast.String)
		if !ok {
			continue
		}
		name, value, ok := strings.Cut(option.Str, "=")
		if !ok {
			continue
		}
		name = strings.ToUpper(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		switch name {
		case "BOUNDING_BOX":
			parseBoundingBox(value, index)
		case "GRIDS":
			parseGridLevels(value, index)
		case "CELLS_PER_OBJECT":
			if cellsPerObject, err := strconv.ParseInt(value, 10, 32); err == nil {
				index.SpatialConfig.Tessellation.CellsPerObject = int32(cellsPerObject)
			}
		case "PAD_INDEX", "FILLFACTOR", "SORT_IN_TEMPDB", "ONLINE", "ALLOW_ROW_LOCKS", "ALLOW_PAGE_LOCKS", "MAXDOP", "DATA_COMPRESSION":
			parseSpatialStorageOption(name, value, index)
		default:
		}
	}
}

func parseBoundingBox(value string, index *storepb.IndexMetadata) {
	value = strings.TrimSpace(strings.Trim(value, "()"))
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return
	}
	var nums [4]float64
	for i, part := range parts {
		num, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return
		}
		nums[i] = num
	}
	index.SpatialConfig.Tessellation.BoundingBox = &storepb.BoundingBox{
		Xmin: nums[0],
		Ymin: nums[1],
		Xmax: nums[2],
		Ymax: nums[3],
	}
}

func parseGridLevels(value string, index *storepb.IndexMetadata) {
	value = strings.TrimSpace(strings.Trim(value, "()"))
	var gridLevels []*storepb.GridLevel
	for _, part := range strings.Split(value, ",") {
		levelPart, densityPart, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		levelPart = strings.TrimSpace(levelPart)
		densityPart = strings.TrimSpace(densityPart)
		var levelNum int32
		if strings.HasPrefix(strings.ToUpper(levelPart), "LEVEL_") {
			if num, err := strconv.ParseInt(levelPart[6:], 10, 32); err == nil {
				levelNum = int32(num)
			}
		}
		gridLevels = append(gridLevels, &storepb.GridLevel{
			Level:   levelNum,
			Density: strings.ToUpper(densityPart),
		})
	}
	index.SpatialConfig.Tessellation.GridLevels = gridLevels
}

func parseSpatialStorageOption(name, value string, index *storepb.IndexMetadata) {
	if index.SpatialConfig.Storage == nil {
		index.SpatialConfig.Storage = &storepb.StorageConfig{}
	}
	storage := index.SpatialConfig.Storage
	switch name {
	case "PAD_INDEX":
		storage.PadIndex = strings.EqualFold(value, "ON")
	case "FILLFACTOR":
		if fillFactor, err := strconv.ParseInt(value, 10, 32); err == nil {
			storage.Fillfactor = int32(fillFactor)
		}
	case "SORT_IN_TEMPDB":
		storage.SortInTempdb = strings.ToUpper(value)
	case "ONLINE":
		storage.Online = strings.EqualFold(value, "ON")
	case "ALLOW_ROW_LOCKS":
		storage.AllowRowLocks = strings.EqualFold(value, "ON")
	case "ALLOW_PAGE_LOCKS":
		storage.AllowPageLocks = strings.EqualFold(value, "ON")
	case "MAXDOP":
		if maxdop, err := strconv.ParseInt(value, 10, 32); err == nil {
			storage.Maxdop = int32(maxdop)
		}
	case "DATA_COMPRESSION":
		storage.DataCompression = strings.ToUpper(value)
	default:
	}
}

func tableRefSchemaObject(ref *ast.TableRef, fallbackSchema string) (string, string) {
	if ref == nil {
		return fallbackSchema, ""
	}
	schemaName := ref.Schema
	if schemaName == "" {
		schemaName = fallbackSchema
	}
	return schemaName, ref.Object
}

func (e *omniMetadataExtractor) nodeText(node ast.Node) string {
	loc := ast.NodeLoc(node)
	if loc.Start < 0 || loc.End < 0 || loc.Start > loc.End || loc.End > len(e.schemaText) {
		return ""
	}
	return e.schemaText[loc.Start:loc.End]
}

func (e *omniMetadataExtractor) definitionText(node ast.Node) string {
	loc := ast.NodeLoc(node)
	if loc.Start < 0 || loc.End < 0 || loc.Start > loc.End || loc.End > len(e.schemaText) {
		return ""
	}
	end := loc.End
	for end < len(e.schemaText) {
		switch e.schemaText[end] {
		case ' ', '\t', '\r', '\n':
			end++
		default:
			if e.schemaText[end] == ';' {
				end++
			}
			return strings.TrimSpace(e.schemaText[loc.Start:end])
		}
	}
	return strings.TrimSpace(e.schemaText[loc.Start:end])
}

func indexType(columnstore bool, clustered *bool) string {
	if columnstore {
		if clustered != nil && *clustered {
			return "CLUSTERED COLUMNSTORE"
		}
		return "NONCLUSTERED COLUMNSTORE"
	}
	if clustered == nil {
		return ""
	}
	if *clustered {
		return "CLUSTERED"
	}
	return "NONCLUSTERED"
}

func appendIndexColumns(index *storepb.IndexMetadata, columns *ast.List) {
	if columns == nil {
		return
	}
	for _, item := range columns.Items {
		column, ok := item.(*ast.IndexColumn)
		if !ok {
			continue
		}
		index.Expressions = append(index.Expressions, column.Name)
		index.Descending = append(index.Descending, column.SortDir == ast.SortDesc)
	}
}

func indexColumnNames(columns *ast.List) []string {
	var names []string
	if columns == nil {
		return names
	}
	for _, item := range columns.Items {
		column, ok := item.(*ast.IndexColumn)
		if !ok {
			continue
		}
		names = append(names, column.Name)
	}
	return names
}

func stringNodeList(list *ast.List) []string {
	var result []string
	if list == nil {
		return result
	}
	for _, item := range list.Items {
		switch n := item.(type) {
		case *ast.String:
			result = append(result, n.Str)
		case *ast.IndexColumn:
			result = append(result, n.Name)
		default:
		}
	}
	return result
}

func referentialAction(action ast.ReferentialAction) string {
	switch action {
	case ast.RefActCascade:
		return "CASCADE"
	case ast.RefActSetNull:
		return "SET NULL"
	case ast.RefActSetDefault:
		return "SET DEFAULT"
	case ast.RefActNoAction:
		return noAction
	default:
		return ""
	}
}
