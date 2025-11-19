package model

import (
	"slices"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// DatabaseMetadata is the unified database schema including metadata, config, and raw dump.
// This struct combines what were previously separate types: DatabaseMetadata, and DatabaseConfig.
type DatabaseMetadata struct {
	// Proto representations for serialization
	metadata *storepb.DatabaseSchemaMetadata
	config   *storepb.DatabaseConfig
	schema   []byte

	// Case sensitivity flags
	isObjectCaseSensitive bool
	isDetailCaseSensitive bool

	// Metadata fields (formerly in DatabaseMetadata)
	name           string
	owner          string
	searchPath     []string
	internal       map[string]*SchemaMetadata
	linkedDatabase map[string]*LinkedDatabaseMetadata

	// Config fields (formerly in DatabaseConfig)
	configName     string
	configInternal map[string]*SchemaConfig
}

func NewDatabaseMetadata(
	metadata *storepb.DatabaseSchemaMetadata,
	schema []byte,
	config *storepb.DatabaseConfig,
	engine storepb.Engine,
	isObjectCaseSensitive bool,
) *DatabaseMetadata {
	isDetailCaseSensitive := getIsDetailCaseSensitive(engine)
	dbMetadata := &DatabaseMetadata{
		metadata:              metadata,
		schema:                schema,
		config:                config,
		isObjectCaseSensitive: isObjectCaseSensitive,
		isDetailCaseSensitive: isDetailCaseSensitive,
		name:                  metadata.Name,
		owner:                 metadata.Owner,
		searchPath:            normalizeSearchPathSlice(metadata.SearchPath),
		internal:              make(map[string]*SchemaMetadata),
		linkedDatabase:        make(map[string]*LinkedDatabaseMetadata),
		configInternal:        make(map[string]*SchemaConfig),
	}

	if config != nil {
		dbMetadata.configName = config.Name
	}

	// Build schema metadata maps
	for _, s := range metadata.Schemas {
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
			proto:                    s,
		}
		for _, table := range s.Tables {
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
		for _, externalTable := range s.ExternalTables {
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
		for _, view := range s.Views {
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
		for _, materializedView := range s.MaterializedViews {
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
		for _, function := range s.Functions {
			schemaMetadata.internalFunctions = append(schemaMetadata.internalFunctions, &FunctionMetadata{
				Definition: function.Definition,
				proto:      function,
			})
		}
		for _, procedure := range s.Procedures {
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
		for _, p := range s.Packages {
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
		for _, sequence := range s.Sequences {
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
			schemaID = s.Name
		} else {
			schemaID = strings.ToLower(s.Name)
		}
		dbMetadata.internal[schemaID] = schemaMetadata
	}

	for _, dbLink := range metadata.LinkedDatabases {
		var dbLinkID string
		if isObjectCaseSensitive {
			dbLinkID = dbLink.Name
		} else {
			dbLinkID = strings.ToLower(dbLink.Name)
		}
		dbMetadata.linkedDatabase[dbLinkID] = &LinkedDatabaseMetadata{
			name:     dbLink.Name,
			username: dbLink.Username,
			host:     dbLink.Host,
		}
	}

	// Build config maps
	if config != nil {
		for _, schemaProto := range config.Schemas {
			schemaConfig := &SchemaConfig{
				internal: make(map[string]*TableConfig),
			}
			for _, table := range schemaProto.Tables {
				tableConfig := &TableConfig{
					Classification: table.Classification,
					internal:       make(map[string]*storepb.ColumnCatalog),
				}
				for _, column := range table.Columns {
					tableConfig.internal[column.Name] = column
				}
				schemaConfig.internal[table.Name] = tableConfig
			}
			dbMetadata.configInternal[schemaProto.Name] = schemaConfig
		}
	}

	return dbMetadata
}

// Proto getters for serialization
func (d *DatabaseMetadata) GetMetadata() *storepb.DatabaseSchemaMetadata {
	return d.metadata
}

func (d *DatabaseMetadata) GetSchema() []byte {
	return d.schema
}

func (d *DatabaseMetadata) GetConfig() *storepb.DatabaseConfig {
	return d.config
}

// Metadata methods (formerly from DatabaseMetadata)
func (d *DatabaseMetadata) GetName() string {
	return d.name
}

func (d *DatabaseMetadata) GetOwner() string {
	return d.owner
}

func (d *DatabaseMetadata) GetSearchPath() []string {
	return d.searchPath
}

func (d *DatabaseMetadata) GetSchemaMetadata(name string) *SchemaMetadata {
	var schemaID string
	if d.isObjectCaseSensitive {
		schemaID = name
	} else {
		schemaID = strings.ToLower(name)
	}
	return d.internal[schemaID]
}

func (d *DatabaseMetadata) DatabaseName() string {
	if d.metadata == nil {
		return ""
	}
	return d.metadata.Name
}

func (d *DatabaseMetadata) HasNoTable() bool {
	for _, schema := range d.internal {
		if schema != nil && schema.proto != nil && len(schema.proto.Tables) > 0 {
			return false
		}
	}
	return true
}

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

func (d *DatabaseMetadata) GetProto() *storepb.DatabaseSchemaMetadata {
	return d.metadata
}

func (d *DatabaseMetadata) SortProto() {
	// Sort schemas by name
	slices.SortFunc(d.metadata.Schemas, func(x, y *storepb.SchemaMetadata) int {
		return strings.Compare(x.Name, y.Name)
	})

	// Sort tables and indexes within each schema
	for _, schema := range d.metadata.Schemas {
		// Sort tables by name
		slices.SortFunc(schema.Tables, func(x, y *storepb.TableMetadata) int {
			return strings.Compare(x.Name, y.Name)
		})

		// Sort indexes within each table by name
		for _, table := range schema.Tables {
			slices.SortFunc(table.Indexes, func(x, y *storepb.IndexMetadata) int {
				return strings.Compare(x.Name, y.Name)
			})

			// Sort columns by position
			slices.SortFunc(table.Columns, func(x, y *storepb.ColumnMetadata) int {
				if x.Position < y.Position {
					return -1
				} else if x.Position > y.Position {
					return 1
				}
				return 0
			})
		}

		// Sort views by name
		slices.SortFunc(schema.Views, func(x, y *storepb.ViewMetadata) int {
			return strings.Compare(x.Name, y.Name)
		})
	}
}

func (d *DatabaseMetadata) GetIsObjectCaseSensitive() bool {
	return d.isObjectCaseSensitive
}

func (d *DatabaseMetadata) CreateSchema(schemaName string) *SchemaMetadata {
	// Create new schema proto
	newSchemaProto := &storepb.SchemaMetadata{
		Name:   schemaName,
		Tables: []*storepb.TableMetadata{},
		Views:  []*storepb.ViewMetadata{},
	}

	// Add to proto's schema list
	d.metadata.Schemas = append(d.metadata.Schemas, newSchemaProto)

	// Create SchemaMetadata wrapper
	schemaMeta := &SchemaMetadata{
		isObjectCaseSensitive:    d.isObjectCaseSensitive,
		isDetailCaseSensitive:    d.isDetailCaseSensitive,
		internalTables:           make(map[string]*TableMetadata),
		internalExternalTable:    make(map[string]*ExternalTableMetadata),
		internalViews:            make(map[string]*ViewMetadata),
		internalMaterializedView: make(map[string]*MaterializedViewMetadata),
		internalFunctions:        []*FunctionMetadata{},
		internalProcedures:       make(map[string]*ProcedureMetadata),
		internalPackages:         make(map[string]*PackageMetadata),
		internalSequences:        make(map[string]*SequenceMetadata),
		proto:                    newSchemaProto,
	}

	// Add to internal map
	var schemaID string
	if d.isObjectCaseSensitive {
		schemaID = schemaName
	} else {
		schemaID = strings.ToLower(schemaName)
	}
	d.internal[schemaID] = schemaMeta

	return schemaMeta
}

func (d *DatabaseMetadata) DropSchema(schemaName string) error {
	// Check if schema exists
	if d.GetSchemaMetadata(schemaName) == nil {
		return errors.Errorf("schema %q does not exist in database %q", schemaName, d.name)
	}

	// Remove from internal map
	var schemaID string
	if d.isObjectCaseSensitive {
		schemaID = schemaName
	} else {
		schemaID = strings.ToLower(schemaName)
	}
	delete(d.internal, schemaID)

	// Remove from proto's schema list
	newSchemas := make([]*storepb.SchemaMetadata, 0, len(d.metadata.Schemas)-1)
	for _, schema := range d.metadata.Schemas {
		if d.isObjectCaseSensitive {
			if schema.Name != schemaName {
				newSchemas = append(newSchemas, schema)
			}
		} else {
			if !strings.EqualFold(schema.Name, schemaName) {
				newSchemas = append(newSchemas, schema)
			}
		}
	}
	d.metadata.Schemas = newSchemas

	return nil
}

// Config methods (formerly from DatabaseConfig)
func (d *DatabaseMetadata) GetSchemaConfig(name string) *SchemaConfig {
	if d == nil {
		return nil
	}
	if config := d.configInternal[name]; config != nil {
		return config
	}
	return &SchemaConfig{
		internal: make(map[string]*TableConfig),
	}
}

func (d *DatabaseMetadata) BuildDatabaseConfig() *storepb.DatabaseConfig {
	if d == nil {
		return nil
	}
	config := &storepb.DatabaseConfig{Name: d.configName}

	for schemaName, sConfig := range d.configInternal {
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

// GetTableConfig gets the table config by name.
// If not found, returns a new empty table config.
func (s *SchemaConfig) GetTableConfig(name string) *TableConfig {
	if s == nil {
		return nil
	}
	if config := s.internal[name]; config != nil {
		return config
	}
	return &TableConfig{
		internal: make(map[string]*storepb.ColumnCatalog),
	}
}

// TableConfig is the config for a table.
type TableConfig struct {
	Classification string
	internal       map[string]*storepb.ColumnCatalog
}

// GetColumnConfig gets the column config by name.
// If not found, returns a new empty column config.
func (t *TableConfig) GetColumnConfig(name string) *storepb.ColumnCatalog {
	if t == nil {
		return nil
	}
	if config := t.internal[name]; config != nil {
		return config
	}
	return &storepb.ColumnCatalog{
		Name: name,
	}
}

// normalizeSearchPathSlice normalizes the search path from a string to a slice.
func normalizeSearchPathSlice(searchPath string) []string {
	if searchPath == "" {
		return []string{}
	}
	// Split by comma and remove empty strings and "$user"
	parts := strings.Split(searchPath, ",")
	var result []string
	for _, path := range parts {
		trimmed := strings.TrimSpace(path)
		if trimmed != "" && trimmed != "$user" {
			result = append(result, trimmed)
		}
	}
	return result
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
	if s == nil {
		return nil
	}
	var nameID string
	if s.isObjectCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return s.internalTables[nameID]
}

// GetIndex gets the index by name.
// Index names are unique within a schema in most databases.
func (s *SchemaMetadata) GetIndex(name string) *IndexMetadata {
	if s == nil {
		return nil
	}
	for _, table := range s.internalTables {
		if index := table.GetIndex(name); index != nil {
			return index
		}
	}
	return nil
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

// CreateTable creates a new table in the schema.
// Returns the created TableMetadata or an error if the table already exists.
func (s *SchemaMetadata) CreateTable(tableName string) (*TableMetadata, error) {
	// Check if table already exists
	if s.GetTable(tableName) != nil {
		return nil, errors.Errorf("table %q already exists in schema %q", tableName, s.proto.Name)
	}

	// Create new table proto
	newTableProto := &storepb.TableMetadata{
		Name:    tableName,
		Columns: []*storepb.ColumnMetadata{},
		Indexes: []*storepb.IndexMetadata{},
	}

	// Add to proto's table list
	s.proto.Tables = append(s.proto.Tables, newTableProto)

	// Create TableMetadata wrapper
	tableMeta := &TableMetadata{
		isDetailCaseSensitive: s.isDetailCaseSensitive,
		internalColumn:        make(map[string]*storepb.ColumnMetadata),
		internalIndexes:       make(map[string]*IndexMetadata),
		columns:               []*storepb.ColumnMetadata{},
		proto:                 newTableProto,
	}

	// Add to internal map
	var tableID string
	if s.isObjectCaseSensitive {
		tableID = tableName
	} else {
		tableID = strings.ToLower(tableName)
	}
	s.internalTables[tableID] = tableMeta

	return tableMeta, nil
}

// DropTable drops a table from the schema.
// Returns an error if the table does not exist.
func (s *SchemaMetadata) DropTable(tableName string) error {
	// Check if table exists
	if s.GetTable(tableName) == nil {
		return errors.Errorf("table %q does not exist in schema %q", tableName, s.proto.Name)
	}

	// Remove from internal map
	var tableID string
	if s.isObjectCaseSensitive {
		tableID = tableName
	} else {
		tableID = strings.ToLower(tableName)
	}
	delete(s.internalTables, tableID)

	// Remove from proto's table list
	newTables := make([]*storepb.TableMetadata, 0, len(s.proto.Tables)-1)
	for _, table := range s.proto.Tables {
		if s.isObjectCaseSensitive {
			if table.Name != tableName {
				newTables = append(newTables, table)
			}
		} else {
			if !strings.EqualFold(table.Name, tableName) {
				newTables = append(newTables, table)
			}
		}
	}
	s.proto.Tables = newTables

	return nil
}

// RenameTable renames a table in the schema.
// Returns an error if the old table doesn't exist or new table already exists.
func (s *SchemaMetadata) RenameTable(oldName string, newName string) error {
	if oldName == newName {
		return nil
	}

	// Check if old table exists
	oldTable := s.GetTable(oldName)
	if oldTable == nil {
		return errors.Errorf("table %q does not exist in schema %q", oldName, s.proto.Name)
	}

	// Check if new table already exists
	if s.GetTable(newName) != nil {
		return errors.Errorf("table %q already exists in schema %q", newName, s.proto.Name)
	}

	// Remove from internal map using old name
	var oldTableID string
	if s.isObjectCaseSensitive {
		oldTableID = oldName
	} else {
		oldTableID = strings.ToLower(oldName)
	}
	delete(s.internalTables, oldTableID)

	// Update the table name in the proto
	oldTable.proto.Name = newName

	// Add back to internal map using new name
	var newTableID string
	if s.isObjectCaseSensitive {
		newTableID = newName
	} else {
		newTableID = strings.ToLower(newName)
	}
	s.internalTables[newTableID] = oldTable

	return nil
}

// CreateView creates a new view in the schema.
// Returns an error if the view already exists.
func (s *SchemaMetadata) CreateView(viewName string, definition string, dependencyColumns []*storepb.DependencyColumn) (*ViewMetadata, error) {
	// Check if view already exists
	if s.GetView(viewName) != nil {
		return nil, errors.Errorf("view %q already exists in schema %q", viewName, s.proto.Name)
	}

	// Create new view proto
	newViewProto := &storepb.ViewMetadata{
		Name:              viewName,
		Definition:        definition,
		DependencyColumns: dependencyColumns,
	}

	// Add to proto's view list
	s.proto.Views = append(s.proto.Views, newViewProto)

	// Create ViewMetadata wrapper
	viewMeta := &ViewMetadata{
		Definition: definition,
		proto:      newViewProto,
	}

	// Add to internal map
	var viewID string
	if s.isObjectCaseSensitive {
		viewID = viewName
	} else {
		viewID = strings.ToLower(viewName)
	}
	s.internalViews[viewID] = viewMeta

	return viewMeta, nil
}

// DropView drops a view from the schema.
// Returns an error if the view does not exist.
func (s *SchemaMetadata) DropView(viewName string) error {
	// Check if view exists
	if s.GetView(viewName) == nil {
		return errors.Errorf("view %q does not exist in schema %q", viewName, s.proto.Name)
	}

	// Remove from internal map
	var viewID string
	if s.isObjectCaseSensitive {
		viewID = viewName
	} else {
		viewID = strings.ToLower(viewName)
	}
	delete(s.internalViews, viewID)

	// Remove from proto's view list
	newViews := make([]*storepb.ViewMetadata, 0, len(s.proto.Views)-1)
	for _, view := range s.proto.Views {
		if s.isObjectCaseSensitive {
			if view.Name != viewName {
				newViews = append(newViews, view)
			}
		} else {
			if !strings.EqualFold(view.Name, viewName) {
				newViews = append(newViews, view)
			}
		}
	}
	s.proto.Views = newViews

	return nil
}

// RenameView renames a view in the schema.
// Returns an error if the old view doesn't exist or new view already exists.
func (s *SchemaMetadata) RenameView(oldName string, newName string) error {
	if oldName == newName {
		return nil
	}

	// Check if old view exists
	oldView := s.GetView(oldName)
	if oldView == nil {
		return errors.Errorf("view %q does not exist in schema %q", oldName, s.proto.Name)
	}

	// Check if new view already exists
	if s.GetView(newName) != nil {
		return errors.Errorf("view %q already exists in schema %q", newName, s.proto.Name)
	}

	// Remove from internal map using old name
	var oldViewID string
	if s.isObjectCaseSensitive {
		oldViewID = oldName
	} else {
		oldViewID = strings.ToLower(oldName)
	}
	delete(s.internalViews, oldViewID)

	// Update the view name in the proto
	oldView.proto.Name = newName

	// Add back to internal map using new name
	var newViewID string
	if s.isObjectCaseSensitive {
		newViewID = newName
	} else {
		newViewID = strings.ToLower(newName)
	}
	s.internalViews[newViewID] = oldView

	return nil
}

// GetDependentViews returns all views that depend on the given table and column.
// This is used to check if a column can be dropped or if a table can be dropped.
func (s *SchemaMetadata) GetDependentViews(tableName string, columnName string) []string {
	var dependentViews []string

	for _, view := range s.internalViews {
		viewProto := view.GetProto()
		for _, dep := range viewProto.DependencyColumns {
			// Schema is implicitly the same schema, or explicitly matches
			tableMatches := false
			if s.isObjectCaseSensitive {
				tableMatches = dep.Table == tableName
			} else {
				tableMatches = strings.EqualFold(dep.Table, tableName)
			}

			if tableMatches {
				// If columnName is empty, we're checking for table dependency
				if columnName == "" {
					dependentViews = append(dependentViews, viewProto.Name)
					break
				}

				// Check column dependency
				if s.isDetailCaseSensitive {
					if dep.Column == columnName {
						dependentViews = append(dependentViews, viewProto.Name)
						break
					}
				} else {
					if strings.EqualFold(dep.Column, columnName) {
						dependentViews = append(dependentViews, viewProto.Name)
						break
					}
				}
			}
		}
	}

	return dependentViews
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
	if t == nil {
		return nil
	}
	var nameID string
	if t.isDetailCaseSensitive {
		nameID = name
	} else {
		nameID = strings.ToLower(name)
	}
	return t.internalColumn[nameID]
}

func (t *TableMetadata) GetIndex(name string) *IndexMetadata {
	if t == nil {
		return nil
	}
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

// CreateColumn creates a new column in the table.
// Returns an error if the column already exists.
func (t *TableMetadata) CreateColumn(columnProto *storepb.ColumnMetadata) error {
	// Check if column already exists
	if t.GetColumn(columnProto.Name) != nil {
		return errors.Errorf("column %q already exists in table %q", columnProto.Name, t.proto.Name)
	}

	// Add to proto's column list
	t.proto.Columns = append(t.proto.Columns, columnProto)

	// Add to internal map
	var columnID string
	if t.isDetailCaseSensitive {
		columnID = columnProto.Name
	} else {
		columnID = strings.ToLower(columnProto.Name)
	}
	t.internalColumn[columnID] = columnProto

	// Add to columns slice
	t.columns = append(t.columns, columnProto)

	return nil
}

// DropColumn drops a column from the table.
// Returns an error if the column does not exist.
func (t *TableMetadata) DropColumn(columnName string) error {
	return t.dropColumnInternal(columnName, true)
}

// dropColumnInternal is the internal implementation that allows controlling position renumbering.
func (t *TableMetadata) dropColumnInternal(columnName string, renumberPositions bool) error {
	// Check if column exists
	if t.GetColumn(columnName) == nil {
		return errors.Errorf("column %q does not exist in table %q", columnName, t.proto.Name)
	}

	// Remove from internal map
	var columnID string
	if t.isDetailCaseSensitive {
		columnID = columnName
	} else {
		columnID = strings.ToLower(columnName)
	}
	delete(t.internalColumn, columnID)

	// Remove from proto's column list
	newColumns := make([]*storepb.ColumnMetadata, 0, len(t.proto.Columns)-1)
	for _, column := range t.proto.Columns {
		if t.isDetailCaseSensitive {
			if column.Name != columnName {
				newColumns = append(newColumns, column)
			}
		} else {
			if !strings.EqualFold(column.Name, columnName) {
				newColumns = append(newColumns, column)
			}
		}
	}
	t.proto.Columns = newColumns

	// Rebuild columns slice
	t.columns = newColumns

	// Renumber positions to be sequential (1-indexed) if requested
	// MySQL/TiDB: renumber positions (1, 2, 3, ...)
	// PostgreSQL: keep original positions (gaps are allowed)
	if renumberPositions {
		for i, col := range newColumns {
			col.Position = int32(i + 1)
		}
	}

	// Remove column from indexes that reference it
	for _, index := range t.internalIndexes {
		var newExpressions []string
		for _, expr := range index.proto.Expressions {
			if t.isDetailCaseSensitive {
				if expr != columnName {
					newExpressions = append(newExpressions, expr)
				}
			} else {
				if !strings.EqualFold(expr, columnName) {
					newExpressions = append(newExpressions, expr)
				}
			}
		}
		index.proto.Expressions = newExpressions
	}

	// Remove empty indexes (indexes that had all columns dropped)
	var indexesToRemove []string
	for indexName, index := range t.internalIndexes {
		if len(index.proto.Expressions) == 0 {
			indexesToRemove = append(indexesToRemove, indexName)
		}
	}
	for _, indexName := range indexesToRemove {
		delete(t.internalIndexes, indexName)
	}

	// Remove empty indexes from proto
	newIndexes := make([]*storepb.IndexMetadata, 0)
	for _, index := range t.proto.Indexes {
		if len(index.Expressions) > 0 {
			newIndexes = append(newIndexes, index)
		}
	}
	t.proto.Indexes = newIndexes

	return nil
}

// DropColumnWithoutRenumbering drops a column from the table without renumbering positions.
// This is used for PostgreSQL where column positions are stable (attnum) and shouldn't be renumbered.
// Returns an error if the column doesn't exist.
func (t *TableMetadata) DropColumnWithoutRenumbering(columnName string) error {
	return t.dropColumnInternal(columnName, false)
}

// DropColumnWithoutUpdatingIndexes drops a column from the table without updating index expressions.
// This is used when changing a column definition (MODIFY/CHANGE COLUMN) where we want to:
// 1. Drop the old column from the column list
// 2. Manually rename it in index expressions
// 3. Create a new column with the new definition
// Returns an error if the column doesn't exist.
func (t *TableMetadata) DropColumnWithoutUpdatingIndexes(columnName string) error {
	// Check if column exists
	if t.GetColumn(columnName) == nil {
		return errors.Errorf("column %q does not exist in table %q", columnName, t.proto.Name)
	}

	// Remove from internal map
	var columnID string
	if t.isDetailCaseSensitive {
		columnID = columnName
	} else {
		columnID = strings.ToLower(columnName)
	}
	delete(t.internalColumn, columnID)

	// Remove from proto's column list
	newColumns := make([]*storepb.ColumnMetadata, 0, len(t.proto.Columns)-1)
	for _, column := range t.proto.Columns {
		if t.isDetailCaseSensitive {
			if column.Name != columnName {
				newColumns = append(newColumns, column)
			}
		} else {
			if !strings.EqualFold(column.Name, columnName) {
				newColumns = append(newColumns, column)
			}
		}
	}
	t.proto.Columns = newColumns

	// Rebuild columns slice
	t.columns = newColumns

	// NOTE: We intentionally do NOT renumber positions here
	// The caller (tidbCompleteTableChangeColumn) will handle position adjustments
	// as part of the column reordering logic.

	// NOTE: We intentionally do NOT update index expressions here
	// The caller is responsible for updating index expressions as needed

	return nil
}

// RenameColumn renames a column in the table.
// Returns an error if the old column doesn't exist or new column already exists.
func (t *TableMetadata) RenameColumn(oldName string, newName string) error {
	if oldName == newName {
		return nil
	}

	// Check if old column exists
	oldColumn := t.GetColumn(oldName)
	if oldColumn == nil {
		return errors.Errorf("column %q does not exist in table %q", oldName, t.proto.Name)
	}

	// Check if new column already exists
	if t.GetColumn(newName) != nil {
		return errors.Errorf("column %q already exists in table %q", newName, t.proto.Name)
	}

	// Remove from internal map using old name
	var oldColumnID string
	if t.isDetailCaseSensitive {
		oldColumnID = oldName
	} else {
		oldColumnID = strings.ToLower(oldName)
	}
	delete(t.internalColumn, oldColumnID)

	// Update the column name in the proto
	oldColumn.Name = newName

	// Add back to internal map using new name
	var newColumnID string
	if t.isDetailCaseSensitive {
		newColumnID = newName
	} else {
		newColumnID = strings.ToLower(newName)
	}
	t.internalColumn[newColumnID] = oldColumn

	// Update column references in indexes
	for _, index := range t.internalIndexes {
		for i, expr := range index.proto.Expressions {
			if t.isDetailCaseSensitive {
				if expr == oldName {
					index.proto.Expressions[i] = newName
				}
			} else {
				if strings.EqualFold(expr, oldName) {
					index.proto.Expressions[i] = newName
				}
			}
		}
	}

	return nil
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

// Primary returns true if the index is a primary key.
func (i *IndexMetadata) Primary() bool {
	if i.proto == nil {
		return false
	}
	return i.proto.Primary
}

// Unique returns true if the index is unique.
func (i *IndexMetadata) Unique() bool {
	if i.proto == nil {
		return false
	}
	return i.proto.Unique
}

// ExpressionList returns the list of expressions/columns in the index.
func (i *IndexMetadata) ExpressionList() []string {
	if i.proto == nil {
		return nil
	}
	return i.proto.Expressions
}

// CreateIndex creates a new index in the table.
// Returns an error if the index already exists.
func (t *TableMetadata) CreateIndex(indexProto *storepb.IndexMetadata) error {
	// Check if index already exists
	if t.GetIndex(indexProto.Name) != nil {
		return errors.Errorf("index %q already exists in table %q", indexProto.Name, t.proto.Name)
	}

	// Add to proto's index list
	t.proto.Indexes = append(t.proto.Indexes, indexProto)

	// Add to internal map
	var indexID string
	if t.isDetailCaseSensitive {
		indexID = indexProto.Name
	} else {
		indexID = strings.ToLower(indexProto.Name)
	}
	t.internalIndexes[indexID] = &IndexMetadata{
		tableProto: t.proto,
		proto:      indexProto,
	}

	return nil
}

// DropIndex drops an index from the table.
// Returns an error if the index does not exist.
func (t *TableMetadata) DropIndex(indexName string) error {
	// Check if index exists
	if t.GetIndex(indexName) == nil {
		return errors.Errorf("index %q does not exist in table %q", indexName, t.proto.Name)
	}

	// Remove from internal map
	var indexID string
	if t.isDetailCaseSensitive {
		indexID = indexName
	} else {
		indexID = strings.ToLower(indexName)
	}
	delete(t.internalIndexes, indexID)

	// Remove from proto's index list
	newIndexes := make([]*storepb.IndexMetadata, 0, len(t.proto.Indexes)-1)
	for _, index := range t.proto.Indexes {
		if t.isDetailCaseSensitive {
			if index.Name != indexName {
				newIndexes = append(newIndexes, index)
			}
		} else {
			if !strings.EqualFold(index.Name, indexName) {
				newIndexes = append(newIndexes, index)
			}
		}
	}
	t.proto.Indexes = newIndexes

	return nil
}

// RenameIndex renames an index in the table.
// Returns an error if the old index doesn't exist or new index already exists.
func (t *TableMetadata) RenameIndex(oldName string, newName string) error {
	if oldName == newName {
		return nil
	}

	// Check if old index exists
	oldIndex := t.GetIndex(oldName)
	if oldIndex == nil {
		return errors.Errorf("index %q does not exist in table %q", oldName, t.proto.Name)
	}

	// Check if new index already exists
	if t.GetIndex(newName) != nil {
		return errors.Errorf("index %q already exists in table %q", newName, t.proto.Name)
	}

	// Remove from internal map using old name
	var oldIndexID string
	if t.isDetailCaseSensitive {
		oldIndexID = oldName
	} else {
		oldIndexID = strings.ToLower(oldName)
	}
	delete(t.internalIndexes, oldIndexID)

	// Update the index name in the proto
	oldIndex.proto.Name = newName

	// Add back to internal map using new name
	var newIndexID string
	if t.isDetailCaseSensitive {
		newIndexID = newName
	} else {
		newIndexID = strings.ToLower(newName)
	}
	t.internalIndexes[newIndexID] = oldIndex

	return nil
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
