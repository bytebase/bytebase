package model

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DBSchema is the database schema including the metadata and schema (raw dump).
type DBSchema struct {
	metadata *storepb.DatabaseSchemaMetadata
	schema   []byte
	config   *storepb.DatabaseConfig

	metadataInternal *DatabaseMetadata
	configInternal   *DatabaseConfig
}

func NewDBSchema(metadata *storepb.DatabaseSchemaMetadata, schema []byte, config *storepb.DatabaseConfig) *DBSchema {
	databaseMetadata := NewDatabaseMetadata(metadata)
	databaseConfig := NewDatabaseConfig(config)
	return &DBSchema{
		metadata:         metadata,
		schema:           schema,
		config:           config,
		metadataInternal: databaseMetadata,
		configInternal:   databaseConfig,
	}
}

func (dbs *DBSchema) GetMetadata() *storepb.DatabaseSchemaMetadata {
	return dbs.metadata
}

func (dbs *DBSchema) GetSchema() []byte {
	return dbs.schema
}

func (dbs *DBSchema) GetConfig() *storepb.DatabaseConfig {
	return dbs.config
}

func (dbs *DBSchema) GetDatabaseMetadata() *DatabaseMetadata {
	return dbs.metadataInternal
}

func (dbs *DBSchema) GetInternalConfig() *DatabaseConfig {
	return dbs.configInternal
}

// TableExists checks if the table exists.
func (dbs *DBSchema) TableExists(schemaName string, tableName string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		schemaName = strings.ToLower(schemaName)
		tableName = strings.ToLower(tableName)
	}
	for _, schema := range dbs.metadata.Schemas {
		currentSchemaName := schema.Name
		if ignoreCaseSensitive {
			currentSchemaName = strings.ToLower(currentSchemaName)
		}
		if currentSchemaName != schemaName {
			continue
		}
		for _, table := range schema.Tables {
			currentTableName := table.Name
			if ignoreCaseSensitive {
				currentTableName = strings.ToLower(currentTableName)
			}
			if currentTableName == tableName {
				return true
			}
		}
	}
	return false
}

// ViewExists checks if the view exists.
func (dbs *DBSchema) ViewExists(schemaName string, name string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		schemaName = strings.ToLower(schemaName)
		name = strings.ToLower(name)
	}
	for _, schema := range dbs.metadata.Schemas {
		currentSchemaName := schema.Name
		if ignoreCaseSensitive {
			currentSchemaName = strings.ToLower(currentSchemaName)
		}
		if currentSchemaName != schemaName {
			continue
		}
		for _, view := range schema.Views {
			currentViewName := view.Name
			if ignoreCaseSensitive {
				currentViewName = strings.ToLower(currentViewName)
			}
			if currentViewName == name {
				return true
			}
		}
	}
	return false
}

// CompactText returns the compact text representation of the database schema.
func (dbs *DBSchema) CompactText() (string, error) {
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

// FindIndex finds the index by name.
func (dbs *DBSchema) FindIndex(schemaName string, tableName string, indexName string) *storepb.IndexMetadata {
	for _, schema := range dbs.metadata.Schemas {
		if schema.Name != schemaName {
			continue
		}
		for _, table := range schema.Tables {
			if table.Name != tableName {
				continue
			}
			for _, index := range table.Indexes {
				if index.Name == indexName {
					return index
				}
			}
		}
	}
	return nil
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
	name           string
	owner          string
	internal       map[string]*SchemaMetadata
	linkedDatabase map[string]*LinkedDatabaseMetadata
}

// NewDatabaseMetadata creates a new database metadata.
func NewDatabaseMetadata(metadata *storepb.DatabaseSchemaMetadata) *DatabaseMetadata {
	databaseMetadata := &DatabaseMetadata{
		name:           metadata.Name,
		owner:          metadata.Owner,
		internal:       make(map[string]*SchemaMetadata),
		linkedDatabase: make(map[string]*LinkedDatabaseMetadata),
	}
	for _, schema := range metadata.Schemas {
		schemaMetadata := &SchemaMetadata{
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
			tables, names := buildTablesMetadata(table)
			for i, table := range tables {
				schemaMetadata.internalTables[names[i]] = table
			}
		}
		for _, externalTable := range schema.ExternalTables {
			externalTableMetadata := &ExternalTableMetadata{
				internal: make(map[string]*storepb.ColumnMetadata),
			}
			for _, column := range externalTable.Columns {
				externalTableMetadata.internal[column.Name] = column
				externalTableMetadata.columns = append(externalTableMetadata.columns, column)
			}
			schemaMetadata.internalExternalTable[externalTable.Name] = externalTableMetadata
		}
		for _, view := range schema.Views {
			schemaMetadata.internalViews[view.Name] = &ViewMetadata{
				Definition: view.Definition,
				proto:      view,
			}
		}
		for _, materializedView := range schema.MaterializedViews {
			schemaMetadata.internalMaterializedView[materializedView.Name] = &MaterializedViewMetadata{
				Definition: materializedView.Definition,
			}
		}
		for _, function := range schema.Functions {
			schemaMetadata.internalFunctions = append(schemaMetadata.internalFunctions, &FunctionMetadata{
				Definition: function.Definition,
				proto:      function,
			})
		}
		for _, procedure := range schema.Procedures {
			schemaMetadata.internalProcedures[procedure.Name] = &ProcedureMetadata{
				Definition: procedure.Definition,
				proto:      procedure,
			}
		}
		for _, p := range schema.Packages {
			schemaMetadata.internalPackages[p.Name] = &PackageMetadata{
				Definition: p.Definition,
				proto:      p,
			}
		}
		for _, sequence := range schema.Sequences {
			schemaMetadata.internalSequences[sequence.Name] = &SequenceMetadata{
				proto: sequence,
			}
		}
		databaseMetadata.internal[schema.Name] = schemaMetadata
	}
	for _, dbLink := range metadata.LinkedDatabases {
		databaseMetadata.linkedDatabase[dbLink.Name] = &LinkedDatabaseMetadata{
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
	return d.internal[name]
}

// ListSchemaNames lists the schema names.
func (d *DatabaseMetadata) ListSchemaNames() []string {
	var result []string
	for schemaName := range d.internal {
		result = append(result, schemaName)
	}
	return result
}

func (d *DatabaseMetadata) GetLinkedDatabase(name string) *LinkedDatabaseMetadata {
	return d.linkedDatabase[name]
}

func (d *DatabaseMetadata) GetOwner() string {
	return d.owner
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
	return s.internalTables[name]
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
	return s.internalViews[name]
}

func (s *SchemaMetadata) GetProcedure(name string) *ProcedureMetadata {
	return s.internalProcedures[name]
}

func (s *SchemaMetadata) GetPackage(name string) *PackageMetadata {
	return s.internalPackages[name]
}

// GetMaterializedView gets the materialized view by name.
func (s *SchemaMetadata) GetMaterializedView(name string) *MaterializedViewMetadata {
	return s.internalMaterializedView[name]
}

// GetExternalTable gets the external table by name.
func (s *SchemaMetadata) GetExternalTable(name string) *ExternalTableMetadata {
	return s.internalExternalTable[name]
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
		if function.proto.Name == name {
			return function
		}
	}
	return nil
}

// GetSequence gets the sequence by name.
func (s *SchemaMetadata) GetSequence(name string) *SequenceMetadata {
	return s.internalSequences[name]
}

// GetProto gets the proto of SchemaMetadata.
func (s *SchemaMetadata) GetProto() *storepb.SchemaMetadata {
	return s.proto
}

// ListTableNames lists the table names.
func (s *SchemaMetadata) ListTableNames() []string {
	var result []string
	for tableName := range s.internalTables {
		result = append(result, tableName)
	}

	sort.Strings(result)
	return result
}

// ListProcedureNames lists the procedure names.
func (s *SchemaMetadata) ListProcedureNames() []string {
	var result []string
	for procedureName := range s.internalProcedures {
		result = append(result, procedureName)
	}

	sort.Strings(result)
	return result
}

// ListFunctionNames lists the function names.
func (s *SchemaMetadata) ListFunctionNames() []string {
	var result []string
	for _, function := range s.internalFunctions {
		result = append(result, function.GetProto().GetName())
	}

	sort.Strings(result)
	return result
}

// ListViewNames lists the view names.
func (s *SchemaMetadata) ListViewNames() []string {
	var result []string
	for viewName := range s.internalViews {
		result = append(result, viewName)
	}

	sort.Strings(result)
	return result
}

// ListForeignTableNames lists the foreign table names.
func (s *SchemaMetadata) ListForeignTableNames() []string {
	var result []string
	for tableName := range s.internalExternalTable {
		result = append(result, tableName)
	}

	sort.Strings(result)
	return result
}

// ListMaterializedViewNames lists the materialized view names.
func (s *SchemaMetadata) ListMaterializedViewNames() []string {
	var result []string
	for viewName := range s.internalMaterializedView {
		result = append(result, viewName)
	}

	sort.Strings(result)
	return result
}

func buildTablesMetadata(table *storepb.TableMetadata) ([]*TableMetadata, []string) {
	if table == nil {
		return nil, nil
	}
	var result []*TableMetadata
	var name []string
	tableMetadata := &TableMetadata{
		internalColumn:  make(map[string]*storepb.ColumnMetadata),
		internalIndexes: make(map[string]*IndexMetadata),
		proto:           table,
	}
	for _, column := range table.Columns {
		tableMetadata.internalColumn[column.Name] = column
		tableMetadata.columns = append(tableMetadata.columns, column)
	}
	indexes := buildIndexesMetadata(table)
	for _, index := range indexes {
		tableMetadata.internalIndexes[index.proto.Name] = index
	}
	tableMetadata.rowCount = table.RowCount
	result = append(result, tableMetadata)
	name = append(name, table.Name)

	if table.Partitions != nil {
		partitionTables, partitionNames := buildTablesMetadataRecursive(table.Columns, table.Partitions, tableMetadata, table)
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
func buildTablesMetadataRecursive(originalColumn []*storepb.ColumnMetadata, partitions []*storepb.TablePartitionMetadata, root *TableMetadata, proto *storepb.TableMetadata) ([]*TableMetadata, []string) {
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
			partitionMetadata.internalColumn[column.Name] = column
			partitionMetadata.columns = append(partitionMetadata.columns, column)
		}
		tables = append(tables, partitionMetadata)
		names = append(names, partition.Name)
		if partition.Subpartitions != nil {
			subTables, subNames := buildTablesMetadataRecursive(originalColumn, partition.Subpartitions, partitionMetadata, proto)
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

	internalColumn  map[string]*storepb.ColumnMetadata
	internalIndexes map[string]*IndexMetadata
	columns         []*storepb.ColumnMetadata
	rowCount        int64
	proto           *storepb.TableMetadata
}

func (t *TableMetadata) GetOwner() string {
	return t.proto.Owner
}

func (t *TableMetadata) GetTableComment() string {
	return t.proto.Comment
}

// GetColumn gets the column by name.
func (t *TableMetadata) GetColumn(name string) *storepb.ColumnMetadata {
	return t.internalColumn[name]
}

func (t *TableMetadata) GetIndex(name string) *IndexMetadata {
	return t.internalIndexes[name]
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
	internal map[string]*storepb.ColumnMetadata
	columns  []*storepb.ColumnMetadata
}

// GetColumn gets the column by name.
func (t *ExternalTableMetadata) GetColumn(name string) *storepb.ColumnMetadata {
	return t.internal[name]
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
