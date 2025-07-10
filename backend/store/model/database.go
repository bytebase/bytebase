package model

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// DatabaseSchema is the database schema including the metadata and schema (raw dump).
type DatabaseSchema struct {
	metadata              *storepb.DatabaseSchemaMetadata
	schema                []byte
	config                *storepb.DatabaseConfig
	isObjectCaseSensitive bool
	isDetailCaseSensitive bool

	metadataInternal *DatabaseMetadata
	configInternal   *DatabaseConfig
}

func NewDatabaseSchema(
	metadata *storepb.DatabaseSchemaMetadata,
	schema []byte,
	config *storepb.DatabaseConfig,
	engine storepb.Engine,
	isObjectCaseSensitive bool,
) *DatabaseSchema {
	databaseMetadata := NewDatabaseMetadata(metadata, isObjectCaseSensitive, getIsDetailCaseSensitive(engine))
	databaseConfig := NewDatabaseConfig(config)
	return &DatabaseSchema{
		metadata:              metadata,
		schema:                schema,
		config:                config,
		isObjectCaseSensitive: isObjectCaseSensitive,
		isDetailCaseSensitive: getIsDetailCaseSensitive(engine),
		metadataInternal:      databaseMetadata,
		configInternal:        databaseConfig,
	}
}

func (dbs *DatabaseSchema) GetMetadata() *storepb.DatabaseSchemaMetadata {
	return dbs.metadata
}

func (dbs *DatabaseSchema) GetSchema() []byte {
	return dbs.schema
}

func (dbs *DatabaseSchema) GetConfig() *storepb.DatabaseConfig {
	return dbs.config
}

func (dbs *DatabaseSchema) GetDatabaseMetadata() *DatabaseMetadata {
	return dbs.metadataInternal
}

func (dbs *DatabaseSchema) GetInternalConfig() *DatabaseConfig {
	return dbs.configInternal
}

// CompactText returns the compact text representation of the database schema.
func (dbs *DatabaseSchema) CompactText() (string, error) {
	if dbs.metadata == nil {
		return "", nil
	}

	var buf bytes.Buffer
	for _, schema := range dbs.metadata.Schemas {
		schemaName := schema.Name
		// If the schema name is empty, use the database name instead, such as MySQL.
		if schemaName == "" {
			schemaName = dbs.metadata.Name
		}
		for _, table := range schema.Tables {
			// Table with columns.
			if _, err := buf.WriteString(fmt.Sprintf("# Table %s.%s(", schemaName, table.Name)); err != nil {
				return "", err
			}
			for i, column := range table.Columns {
				if i == 0 {
					if _, err := buf.WriteString(column.Name); err != nil {
						return "", err
					}
				} else {
					if _, err := buf.WriteString(fmt.Sprintf(", %s", column.Name)); err != nil {
						return "", err
					}
				}
			}
			if _, err := buf.WriteString(") #\n"); err != nil {
				return "", err
			}

			// Indexes.
			for _, index := range table.Indexes {
				if _, err := buf.WriteString(fmt.Sprintf("# Index %s(%s) ON table %s.%s #\n", index.Name, strings.Join(index.Expressions, ", "), schemaName, table.Name)); err != nil {
					return "", err
				}
			}
		}
	}

	return buf.String(), nil
}

// DatabaseConfig is the config for a database.
type DatabaseConfig struct {
	name     string
	internal map[string]*SchemaConfig
}

// NewDatabaseConfig creates a new database config.
func NewDatabaseConfig(config *storepb.DatabaseConfig) *DatabaseConfig {
	databaseConfig := &DatabaseConfig{
		internal: make(map[string]*SchemaConfig),
	}
	if config == nil {
		return databaseConfig
	}
	databaseConfig.name = config.Name
	for _, schema := range config.Schemas {
		schemaConfig := &SchemaConfig{
			internal: make(map[string]*TableConfig),
		}
		for _, table := range schema.Tables {
			tableConfig := &TableConfig{
				Classification: table.Classification,
				internal:       make(map[string]*storepb.ColumnCatalog),
			}
			for _, column := range table.Columns {
				tableConfig.internal[column.Name] = column
			}
			schemaConfig.internal[table.Name] = tableConfig
		}
		databaseConfig.internal[schema.Name] = schemaConfig
	}
	return databaseConfig
}

// CreateOrGetSchemaConfig creates or gets a new schema config by name.
func (d *DatabaseConfig) CreateOrGetSchemaConfig(name string) *SchemaConfig {
	if config := d.internal[name]; config != nil {
		return config
	}
	d.internal[name] = &SchemaConfig{
		internal: make(map[string]*TableConfig),
	}
	return d.internal[name]
}

// RemoveSchemaConfig delete the schema config by name.
func (d *DatabaseConfig) RemoveSchemaConfig(name string) {
	delete(d.internal, name)
}

func (d *DatabaseConfig) BuildDatabaseConfig() *storepb.DatabaseConfig {
	config := &storepb.DatabaseConfig{Name: d.name}

	for schemaName, sConfig := range d.internal {
		schemaConfig := &storepb.SchemaCatalog{Name: schemaName}

		for tableName, tConfig := range sConfig.internal {
			tableConfig := &storepb.TableCatalog{Name: tableName, Classification: tConfig.Classification}

			for colName, colConfig := range tConfig.internal {
				tableConfig.Columns = append(tableConfig.Columns, &storepb.ColumnCatalog{
					Name:           colName,
					SemanticType:   colConfig.SemanticType,
					Labels:         colConfig.Labels,
					Classification: colConfig.Classification,
				})
			}
			schemaConfig.Tables = append(schemaConfig.Tables, tableConfig)
		}
		config.Schemas = append(config.Schemas, schemaConfig)
	}

	return config
}

// SchemaConfig is the config for a schema.
type SchemaConfig struct {
	internal map[string]*TableConfig
}

// Size returns the table config count for the schema config.
func (s *SchemaConfig) IsEmpty() bool {
	return len(s.internal) == 0
}

// CreateOrGetTableConfig creates or gets the table config by name.
func (s *SchemaConfig) CreateOrGetTableConfig(name string) *TableConfig {
	if config := s.internal[name]; config != nil {
		return config
	}
	s.internal[name] = &TableConfig{
		internal: make(map[string]*storepb.ColumnCatalog),
	}
	return s.internal[name]
}

// RemoveTableConfig delete the table config by name.
func (s *SchemaConfig) RemoveTableConfig(name string) {
	delete(s.internal, name)
}

// TableConfig is the config for a table.
type TableConfig struct {
	Classification string
	internal       map[string]*storepb.ColumnCatalog
}

// CreateOrGetColumnConfig creates or gets the column config by name.
func (t *TableConfig) CreateOrGetColumnConfig(name string) *storepb.ColumnCatalog {
	if config := t.internal[name]; config != nil {
		return config
	}
	t.internal[name] = &storepb.ColumnCatalog{
		Name: name,
	}
	return t.internal[name]
}

// RemoveColumnConfig delete the column config by name.
func (t *TableConfig) RemoveColumnConfig(name string) {
	delete(t.internal, name)
}

// RemoveColumnConfig delete the column config by name.
func (t *TableConfig) IsEmpty() bool {
	return len(t.internal) == 0 && t.Classification == ""
}

// DatabaseMetadata is the metadata for a database.
type DatabaseMetadata struct {
	name                  string
	owner                 string
	searchPath            []string
	isObjectCaseSensitive bool
	isDetailCaseSensitive bool
	internal              map[string]*SchemaMetadata
	linkedDatabase        map[string]*LinkedDatabaseMetadata
}

// NewDatabaseMetadata creates a new database metadata.
func NewDatabaseMetadata(metadata *storepb.DatabaseSchemaMetadata, isObjectCaseSensitive bool, isDetailCaseSensitive bool) *DatabaseMetadata {
	databaseMetadata := &DatabaseMetadata{
		name:                  metadata.Name,
		owner:                 metadata.Owner,
		searchPath:            NormalizeSearchPath(metadata.SearchPath),
		isObjectCaseSensitive: isObjectCaseSensitive,
		isDetailCaseSensitive: isDetailCaseSensitive,
		internal:              make(map[string]*SchemaMetadata),
		linkedDatabase:        make(map[string]*LinkedDatabaseMetadata),
	}
	for _, schema := range metadata.Schemas {
		schemaMetadata := &SchemaMetadata{
			isObjectCaseSensitive:    isObjectCaseSensitive,
			isDetailCaseSensitive:    isDetailCaseSensitive,
			internalTables:           make(map[string]*TableMetadata),
			internalExternalTable:    make(map[string]*ExternalTableMetadata),
			internalViews:            make(map[string]*ViewMetadata),
			internalMaterializedView: make(map[string]*MaterializedViewMetadata),
			internalFunctions:        make([]*FunctionMetadata, 0),
			internalProcedures:       make(map[string]*ProcedureMetadata),
			internalPackages:         make(map[string]*PackageMetadata),
			internalSequences:        make(map[string]*SequenceMetadata),
			proto:                    schema,
		}
		for _, table := range schema.Tables {
			tables, names := buildTablesMetadata(table, isDetailCaseSensitive)
			for i, table := range tables {
				var tableID string
				if isObjectCaseSensitive {
					tableID = names[i]
				} else {
					tableID = strings.ToLower(names[i])
				}
				schemaMetadata.internalTables[tableID] = table
			}
		}
		for _, externalTable := range schema.ExternalTables {
			externalTableMetadata := &ExternalTableMetadata{
				isDetailCaseSensitive: isDetailCaseSensitive,
				internal:              make(map[string]*storepb.ColumnMetadata),
				proto:                 externalTable,
			}
			for _, column := range externalTable.Columns {
				var columnID string
				if isDetailCaseSensitive {
					columnID = column.Name
				} else {
					columnID = strings.ToLower(column.Name)
				}
				externalTableMetadata.internal[columnID] = column
				externalTableMetadata.columns = append(externalTableMetadata.columns, column)
			}
			var tableID string
			if isObjectCaseSensitive {
				tableID = externalTable.Name
			} else {
				tableID = strings.ToLower(externalTable.Name)
			}
			schemaMetadata.internalExternalTable[tableID] = externalTableMetadata
		}
		for _, view := range schema.Views {
			var viewID string
			if isObjectCaseSensitive {
				viewID = view.Name
			} else {
				viewID = strings.ToLower(view.Name)
			}
			schemaMetadata.internalViews[viewID] = &ViewMetadata{
				Definition: view.Definition,
				proto:      view,
			}
		}
		for _, materializedView := range schema.MaterializedViews {
			var viewID string
			if isObjectCaseSensitive {
				viewID = materializedView.Name
			} else {
				viewID = strings.ToLower(materializedView.Name)
			}
			schemaMetadata.internalMaterializedView[viewID] = &MaterializedViewMetadata{
				Definition: materializedView.Definition,
				proto:      materializedView,
			}
		}
		for _, function := range schema.Functions {
			schemaMetadata.internalFunctions = append(schemaMetadata.internalFunctions, &FunctionMetadata{
				Definition: function.Definition,
				proto:      function,
			})
		}
		for _, procedure := range schema.Procedures {
			var procedureID string
			if isDetailCaseSensitive {
				procedureID = procedure.Name
			} else {
				procedureID = strings.ToLower(procedure.Name)
			}
			schemaMetadata.internalProcedures[procedureID] = &ProcedureMetadata{
				Definition: procedure.Definition,
				proto:      procedure,
			}
		}
		for _, p := range schema.Packages {
			var packageID string
			if isDetailCaseSensitive {
				packageID = p.Name
			} else {
				packageID = strings.ToLower(p.Name)
			}
			schemaMetadata.internalPackages[packageID] = &PackageMetadata{
				Definition: p.Definition,
				proto:      p,
			}
		}
		for _, sequence := range schema.Sequences {
			var sequenceID string
			if isDetailCaseSensitive {
				sequenceID = sequence.Name
			} else {
				sequenceID = strings.ToLower(sequence.Name)
			}
			schemaMetadata.internalSequences[sequenceID] = &SequenceMetadata{
				proto: sequence,
			}
		}
		var schemaID string
		if isObjectCaseSensitive {
			schemaID = schema.Name
		} else {
			schemaID = strings.ToLower(schema.Name)
		}
		databaseMetadata.internal[schemaID] = schemaMetadata
	}
	for _, dbLink := range metadata.LinkedDatabases {
		var dbLinkID string
		if isObjectCaseSensitive {
			dbLinkID = dbLink.Name
		} else {
			dbLinkID = strings.ToLower(dbLink.Name)
		}
		databaseMetadata.linkedDatabase[dbLinkID] = &LinkedDatabaseMetadata{
			name:     dbLink.Name,
			username: dbLink.Username,
			host:     dbLink.Host,
		}
	}
	return databaseMetadata
}

func (d *DatabaseMetadata) GetName() string {
	return d.name
}

// GetSchema gets the schema by name.
func (d *DatabaseMetadata) GetSchema(name string) *SchemaMetadata {
	var schemaID string
	if d.isObjectCaseSensitive {
		schemaID = name
	} else {
		schemaID = strings.ToLower(name)
	}
	return d.internal[schemaID]
}

func (d *DatabaseMetadata) SearchTable(searchPath []string, name string) (string, *TableMetadata) {
	// Search in the search path first.
	for _, schemaName := range searchPath {
		schema := d.GetSchema(schemaName)
		if schema == nil {
			continue
		}
		table := schema.GetTable(name)
		if table != nil {
			return schema.proto.Name, table
		}
	}

	return "", nil
}

func (d *DatabaseMetadata) SearchIndex(searchPath []string, name string) (string, *IndexMetadata) {
	// Search in the search path first.
	for _, schemaName := range searchPath {
		schema := d.GetSchema(schemaName)
		if schema == nil {
			continue
		}
		indexes := schema.GetIndexes(name)
		if len(indexes) > 0 {
			return schema.proto.Name, indexes[0] // Return the first index found.
		}
	}
	return "", nil
}

func (d *DatabaseMetadata) SearchView(searchPath []string, name string) (string, *ViewMetadata) {
	// Search in the search path first.
	for _, schemaName := range searchPath {
		schema := d.GetSchema(schemaName)
		if schema == nil {
			continue
		}
		view := schema.GetView(name)
		if view != nil {
			return schema.proto.Name, view
		}
	}
	return "", nil
}

func (d *DatabaseMetadata) SearchExternalTable(searchPath []string, name string) (string, *ExternalTableMetadata) {
	// Search in the search path first.
	for _, schemaName := range searchPath {
		schema := d.GetSchema(schemaName)
		if schema == nil {
			continue
		}
		externalTable := schema.GetExternalTable(name)
		if externalTable != nil {
			return schema.proto.Name, externalTable
		}
	}
	return "", nil
}

func (d *DatabaseMetadata) SearchSequence(searchPath []string, name string) (string, *SequenceMetadata) {
	// Search in the search path first.
	for _, schemaName := range searchPath {
		schema := d.GetSchema(schemaName)
		if schema == nil {
			continue
		}
		sequence := schema.GetSequence(name)
		if sequence != nil {
			return schema.proto.Name, sequence
		}
	}
	return "", nil
}

func (d *DatabaseMetadata) SearchMaterializedView(searchPath []string, name string) (string, *MaterializedViewMetadata) {
	// Search in the search path first.
	for _, schemaName := range searchPath {
		schema := d.GetSchema(schemaName)
		if schema == nil {
			continue
		}
		materializedView := schema.GetMaterializedView(name)
		if materializedView != nil {
			return schema.proto.Name, materializedView
		}
	}
	return "", nil
}

func (d *DatabaseMetadata) SearchFunctions(searchPath []string, name string) ([]string, []*FunctionMetadata) {
	var schemas []string
	var funcs []*FunctionMetadata
	// Search in the search path first.
	for _, schemaName := range searchPath {
		schema := d.GetSchema(schemaName)
		if schema == nil {
			continue
		}
		for _, function := range schema.ListFunctions() {
			if d.isDetailCaseSensitive {
				if function.proto.Name == name {
					schemas = append(schemas, schema.proto.Name)
					funcs = append(funcs, function)
				}
			} else {
				if strings.EqualFold(function.proto.Name, name) {
					schemas = append(schemas, schema.proto.Name)
					funcs = append(funcs, function)
				}
			}
		}
	}
	return schemas, funcs
}

func (d *DatabaseMetadata) SearchObject(searchPath []string, name string) (string, string) {
	// Search in the search path first.
	for _, schemaName := range searchPath {
		schema := d.GetSchema(schemaName)
		if schema == nil {
			continue
		}
		if schema.GetTable(name) != nil || schema.GetView(name) != nil || schema.GetMaterializedView(name) != nil || schema.GetFunction(name) != nil || schema.GetProcedure(name) != nil || schema.GetPackage(name) != nil || schema.GetSequence(name) != nil || schema.GetExternalTable(name) != nil {
			return schema.proto.Name, name
		}
	}
	return "", ""
}

// ListSchemaNames lists the schema names.
func (d *DatabaseMetadata) ListSchemaNames() []string {
	var result []string
	for _, schema := range d.internal {
		result = append(result, schema.GetProto().Name)
	}
	return result
}

func (d *DatabaseMetadata) GetLinkedDatabase(name string) *LinkedDatabaseMetadata {
	var nameID string
	if d.isObjectCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return d.linkedDatabase[nameID]
}

func (d *DatabaseMetadata) GetOwner() string {
	return d.owner
}

func (d *DatabaseMetadata) GetSearchPath() []string {
	return d.searchPath
}

// LinkedDatabaseMetadata is the metadata for a linked database.
type LinkedDatabaseMetadata struct {
	name     string
	username string
	host     string
}

func (l *LinkedDatabaseMetadata) GetName() string {
	return l.name
}

func (l *LinkedDatabaseMetadata) GetUsername() string {
	return l.username
}

func (l *LinkedDatabaseMetadata) GetHost() string {
	return l.host
}

// SchemaMetadata is the metadata for a schema.
type SchemaMetadata struct {
	isObjectCaseSensitive    bool
	isDetailCaseSensitive    bool
	internalTables           map[string]*TableMetadata
	internalExternalTable    map[string]*ExternalTableMetadata
	internalViews            map[string]*ViewMetadata
	internalMaterializedView map[string]*MaterializedViewMetadata
	// Store internal functions by list to take care of the overloadings.
	internalFunctions  []*FunctionMetadata
	internalProcedures map[string]*ProcedureMetadata
	internalSequences  map[string]*SequenceMetadata
	internalPackages   map[string]*PackageMetadata

	proto *storepb.SchemaMetadata
}

func (s *SchemaMetadata) GetOwner() string {
	if s.proto == nil {
		return ""
	}
	return s.proto.Owner
}

// GetTable gets the schema by name.
func (s *SchemaMetadata) GetTable(name string) *TableMetadata {
	var nameID string
	if s.isObjectCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return s.internalTables[nameID]
}

func (s *SchemaMetadata) GetIndexes(name string) []*IndexMetadata {
	var result []*IndexMetadata
	for _, table := range s.internalTables {
		if index := table.GetIndex(name); index != nil {
			result = append(result, index)
		}
	}
	return result
}

// GetView gets the view by name.
func (s *SchemaMetadata) GetView(name string) *ViewMetadata {
	var nameID string
	if s.isObjectCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return s.internalViews[nameID]
}

func (s *SchemaMetadata) GetProcedure(name string) *ProcedureMetadata {
	var nameID string
	if s.isDetailCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return s.internalProcedures[nameID]
}

func (s *SchemaMetadata) GetPackage(name string) *PackageMetadata {
	var nameID string
	if s.isDetailCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return s.internalPackages[nameID]
}

// GetMaterializedView gets the materialized view by name.
func (s *SchemaMetadata) GetMaterializedView(name string) *MaterializedViewMetadata {
	var nameID string
	if s.isObjectCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return s.internalMaterializedView[nameID]
}

// GetExternalTable gets the external table by name.
func (s *SchemaMetadata) GetExternalTable(name string) *ExternalTableMetadata {
	var nameID string
	if s.isObjectCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return s.internalExternalTable[nameID]
}

// ListFunctions lists the functions.
func (s *SchemaMetadata) ListFunctions() []*FunctionMetadata {
	var result []*FunctionMetadata
	result = append(result, s.internalFunctions...)
	return result
}

// GetFunction gets the function by name.
func (s *SchemaMetadata) GetFunction(name string) *FunctionMetadata {
	for _, function := range s.internalFunctions {
		if s.isDetailCaseSensitive {
			if function.proto.Name == name {
				return function
			}
		} else {
			if strings.EqualFold(function.proto.Name, name) {
				return function
			}
		}
	}
	return nil
}

// GetSequence gets the sequence by name.
func (s *SchemaMetadata) GetSequence(name string) *SequenceMetadata {
	var nameID string
	if s.isDetailCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return s.internalSequences[nameID]
}

func (s *SchemaMetadata) GetSequencesByOwnerTable(name string) []*SequenceMetadata {
	var result []*SequenceMetadata
	for _, sequence := range s.internalSequences {
		if s.isObjectCaseSensitive {
			if sequence.GetProto().OwnerTable == name {
				result = append(result, sequence)
			}
		} else {
			if strings.EqualFold(sequence.GetProto().OwnerTable, name) {
				result = append(result, sequence)
			}
		}
	}
	return result
}

// GetProto gets the proto of SchemaMetadata.
func (s *SchemaMetadata) GetProto() *storepb.SchemaMetadata {
	return s.proto
}

// ListTableNames lists the table names.
func (s *SchemaMetadata) ListTableNames() []string {
	var result []string
	for _, table := range s.internalTables {
		result = append(result, table.GetProto().GetName())
	}

	slices.Sort(result)
	return result
}

// ListProcedureNames lists the procedure names.
func (s *SchemaMetadata) ListProcedureNames() []string {
	var result []string
	for _, procedure := range s.internalProcedures {
		result = append(result, procedure.GetProto().GetName())
	}

	slices.Sort(result)
	return result
}

// ListFunctionNames lists the function names.
func (s *SchemaMetadata) ListFunctionNames() []string {
	var result []string
	for _, function := range s.internalFunctions {
		result = append(result, function.GetProto().GetName())
	}

	slices.Sort(result)
	return result
}

// ListViewNames lists the view names.
func (s *SchemaMetadata) ListViewNames() []string {
	var result []string
	for _, view := range s.internalViews {
		result = append(result, view.GetProto().GetName())
	}

	slices.Sort(result)
	return result
}

// ListForeignTableNames lists the foreign table names.
func (s *SchemaMetadata) ListForeignTableNames() []string {
	var result []string
	for _, table := range s.internalExternalTable {
		result = append(result, table.GetProto().GetName())
	}

	slices.Sort(result)
	return result
}

// ListMaterializedViewNames lists the materialized view names.
func (s *SchemaMetadata) ListMaterializedViewNames() []string {
	var result []string
	for _, view := range s.internalMaterializedView {
		result = append(result, view.GetProto().GetName())
	}

	slices.Sort(result)
	return result
}

func buildTablesMetadata(table *storepb.TableMetadata, isDetailCaseSensitive bool) ([]*TableMetadata, []string) {
	if table == nil {
		return nil, nil
	}
	var result []*TableMetadata
	var name []string
	tableMetadata := &TableMetadata{
		isDetailCaseSensitive: isDetailCaseSensitive,
		internalColumn:        make(map[string]*storepb.ColumnMetadata),
		internalIndexes:       make(map[string]*IndexMetadata),
		proto:                 table,
	}
	for _, column := range table.Columns {
		var columnID string
		if isDetailCaseSensitive {
			columnID = column.Name
		} else {
			columnID = strings.ToLower(column.Name)
		}
		tableMetadata.internalColumn[columnID] = column
		tableMetadata.columns = append(tableMetadata.columns, column)
	}
	indexes := buildIndexesMetadata(table)
	for _, index := range indexes {
		var indexID string
		if isDetailCaseSensitive {
			indexID = index.proto.Name
		} else {
			indexID = strings.ToLower(index.proto.Name)
		}
		tableMetadata.internalIndexes[indexID] = index
	}
	tableMetadata.rowCount = table.RowCount
	result = append(result, tableMetadata)
	name = append(name, table.Name)

	if table.Partitions != nil {
		partitionTables, partitionNames := buildTablesMetadataRecursive(table.Columns, table.Partitions, tableMetadata, table, isDetailCaseSensitive)
		result = append(result, partitionTables...)
		name = append(name, partitionNames...)
	}
	return result, name
}

func buildIndexesMetadata(table *storepb.TableMetadata) []*IndexMetadata {
	if table == nil {
		return nil
	}

	var result []*IndexMetadata

	for _, index := range table.Indexes {
		result = append(result, &IndexMetadata{
			tableProto: table,
			proto:      index,
		})
	}

	return result
}

// buildTablesMetadataRecursive builds the partition tables recursively,
// returns the table metadata and the partition names, the length of them must be the same.
func buildTablesMetadataRecursive(originalColumn []*storepb.ColumnMetadata, partitions []*storepb.TablePartitionMetadata, root *TableMetadata, proto *storepb.TableMetadata, isDetailCaseSensitive bool) ([]*TableMetadata, []string) {
	if partitions == nil {
		return nil, nil
	}

	var tables []*TableMetadata
	var names []string

	for _, partition := range partitions {
		partitionMetadata := &TableMetadata{
			partitionOf:    root,
			internalColumn: make(map[string]*storepb.ColumnMetadata),
			proto:          proto,
		}
		for _, column := range originalColumn {
			var columnID string
			if isDetailCaseSensitive {
				columnID = column.Name
			} else {
				columnID = strings.ToLower(column.Name)
			}
			partitionMetadata.internalColumn[columnID] = column
			partitionMetadata.columns = append(partitionMetadata.columns, column)
		}
		tables = append(tables, partitionMetadata)
		names = append(names, partition.Name)
		if partition.Subpartitions != nil {
			subTables, subNames := buildTablesMetadataRecursive(originalColumn, partition.Subpartitions, partitionMetadata, proto, isDetailCaseSensitive)
			tables = append(tables, subTables...)
			names = append(names, subNames...)
		}
	}
	return tables, names
}

// TableMetadata is the metadata for a table.
type TableMetadata struct {
	// If partitionOf is not nil, it means this table is a partition table.
	partitionOf *TableMetadata

	isDetailCaseSensitive bool
	internalColumn        map[string]*storepb.ColumnMetadata
	internalIndexes       map[string]*IndexMetadata
	columns               []*storepb.ColumnMetadata
	rowCount              int64
	proto                 *storepb.TableMetadata
}

func (t *TableMetadata) GetOwner() string {
	return t.proto.Owner
}

func (t *TableMetadata) GetTableComment() string {
	return t.proto.Comment
}

// GetColumn gets the column by name.
func (t *TableMetadata) GetColumn(name string) *storepb.ColumnMetadata {
	var nameID string
	if t.isDetailCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return t.internalColumn[nameID]
}

func (t *TableMetadata) GetIndex(name string) *IndexMetadata {
	var nameID string
	if t.isDetailCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return t.internalIndexes[nameID]
}

func (t *TableMetadata) ListIndexes() []*IndexMetadata {
	var result []*IndexMetadata
	for _, index := range t.internalIndexes {
		result = append(result, index)
	}
	return result
}

func (t *TableMetadata) GetPrimaryKey() *IndexMetadata {
	for _, index := range t.internalIndexes {
		if index.proto.Primary {
			return index
		}
	}
	return nil
}

// GetColumns gets the columns.
func (t *TableMetadata) GetColumns() []*storepb.ColumnMetadata {
	return t.columns
}

func (t *TableMetadata) GetRowCount() int64 {
	return t.rowCount
}

func (t *TableMetadata) GetProto() *storepb.TableMetadata {
	return t.proto
}

// ExternalTableMetadata is the metadata for a external table.
type ExternalTableMetadata struct {
	isDetailCaseSensitive bool
	internal              map[string]*storepb.ColumnMetadata
	columns               []*storepb.ColumnMetadata
	proto                 *storepb.ExternalTableMetadata
}

func (t *ExternalTableMetadata) GetProto() *storepb.ExternalTableMetadata {
	return t.proto
}

// GetColumn gets the column by name.
func (t *ExternalTableMetadata) GetColumn(name string) *storepb.ColumnMetadata {
	var nameID string
	if t.isDetailCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return t.internal[nameID]
}

// GetColumns gets the columns.
func (t *ExternalTableMetadata) GetColumns() []*storepb.ColumnMetadata {
	return t.columns
}

type IndexMetadata struct {
	tableProto *storepb.TableMetadata
	proto      *storepb.IndexMetadata
}

func (i *IndexMetadata) GetProto() *storepb.IndexMetadata {
	return i.proto
}

func (i *IndexMetadata) GetTableProto() *storepb.TableMetadata {
	return i.tableProto
}

// ViewMetadata is the metadata for a view.
type ViewMetadata struct {
	Definition string
	proto      *storepb.ViewMetadata
}

func (v *ViewMetadata) GetProto() *storepb.ViewMetadata {
	return v.proto
}

type MaterializedViewMetadata struct {
	Definition string
	proto      *storepb.MaterializedViewMetadata
}

func (m *MaterializedViewMetadata) GetProto() *storepb.MaterializedViewMetadata {
	return m.proto
}

type FunctionMetadata struct {
	Definition string
	proto      *storepb.FunctionMetadata
}

func (f *FunctionMetadata) GetProto() *storepb.FunctionMetadata {
	return f.proto
}

type ProcedureMetadata struct {
	Definition string
	proto      *storepb.ProcedureMetadata
}

func (p *ProcedureMetadata) GetProto() *storepb.ProcedureMetadata {
	return p.proto
}

type PackageMetadata struct {
	Definition string
	proto      *storepb.PackageMetadata
}

func (p *PackageMetadata) GetProto() *storepb.PackageMetadata {
	return p.proto
}

type SequenceMetadata struct {
	proto *storepb.SequenceMetadata
}

func (p *SequenceMetadata) GetProto() *storepb.SequenceMetadata {
	return p.proto
}

// getIsDetailCaseSensitive is a special case for MySQL, MariaDB, and TiDB.
// From MySQL documentation:
// Partition, subpartition, column, index, stored routine, event, and resource group names are not case-sensitive on any platform, nor are column aliases.
func getIsDetailCaseSensitive(engine storepb.Engine) bool {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_TIDB, storepb.Engine_MSSQL, storepb.Engine_OCEANBASE:
		return false
	default:
		return true
	}
}

func IsSystemPath(path string) bool {
	// PostgreSQL system schemas.
	systemSchemas := []string{"pg_catalog", "information_schema", "pg_toast", "pg_temp_1", "pg_temp_2", "pg_global", "$user"}
	for _, schema := range systemSchemas {
		if strings.EqualFold(path, schema) {
			return true
		}
	}
	return false
}

// NormalizeSearchPath normalizes the search path string into a slice of strings.
func NormalizeSearchPath(searchPath string) []string {
	if searchPath == "" {
		return []string{}
	}

	// Split the search path by comma and trim spaces.
	parts := strings.Split(searchPath, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	// Remove empty parts.
	var result []string
	for _, part := range parts {
		schema := strings.TrimSpace(part)
		if part == "\"$user\"" {
			continue
		}
		if strings.HasPrefix(part, "\"") && strings.HasSuffix(part, "\"") {
			// Remove the quotes from the schema name.
			schema = strings.Trim(schema, "\"")
		} else if strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'") {
			// Remove the single quotes from the schema name.
			schema = strings.Trim(schema, "'")
		} else {
			// For non-quoted schema names, we just return the lower string for PostgreSQL.
			schema = strings.ToLower(schema)
		}
		schema = strings.TrimSpace(schema)
		if IsSystemPath(schema) {
			// Skip system schemas.
			continue
		}
		if schema != "" {
			result = append(result, schema)
		}
	}

	return result
}
