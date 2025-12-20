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
	proto   *storepb.DatabaseSchemaMetadata
	config  *storepb.DatabaseConfig
	rawDump []byte

	// Case sensitivity flags
	isObjectCaseSensitive bool
	isDetailCaseSensitive bool

	// Metadata fields (formerly in DatabaseMetadata)
	searchPath     []string
	internal       map[string]*SchemaMetadata
	linkedDatabase map[string]*storepb.LinkedDatabaseMetadata
}

// SchemaMetadata is the unified metadata for a schema, combining proto metadata and catalog config.
type SchemaMetadata struct {
	isObjectCaseSensitive    bool
	isDetailCaseSensitive    bool
	internalTables           map[string]*TableMetadata
	internalExternalTable    map[string]*ExternalTableMetadata
	internalViews            map[string]*storepb.ViewMetadata
	internalMaterializedView map[string]*storepb.MaterializedViewMetadata
	internalProcedures       map[string]*storepb.ProcedureMetadata
	internalSequences        map[string]*storepb.SequenceMetadata
	internalPackages         map[string]*storepb.PackageMetadata

	proto  *storepb.SchemaMetadata
	config *storepb.SchemaCatalog
}

// TableMetadata is the unified metadata for a table, combining proto metadata and catalog config.
type TableMetadata struct {
	// If partitionOf is not nil, it means this table is a partition table.
	partitionOf *TableMetadata

	isDetailCaseSensitive bool
	internalColumn        map[string]*ColumnMetadata
	internalIndexes       map[string]*IndexMetadata

	proto  *storepb.TableMetadata
	config *storepb.TableCatalog
}

// ExternalTableMetadata is the metadata for a external table.
type ExternalTableMetadata struct {
	isDetailCaseSensitive bool
	internal              map[string]*storepb.ColumnMetadata
	proto                 *storepb.ExternalTableMetadata
}

type IndexMetadata struct {
	tableProto *storepb.TableMetadata
	proto      *storepb.IndexMetadata
}

// ColumnMetadata is the unified metadata for a column, combining proto metadata and catalog config.
type ColumnMetadata struct {
	proto  *storepb.ColumnMetadata
	config *storepb.ColumnCatalog
}

// normalizeNameByCaseSensitivity normalizes a name based on case sensitivity.
// If caseSensitive is true, returns the name as-is; otherwise returns lowercase.
func normalizeNameByCaseSensitivity(name string, caseSensitive bool) string {
	if caseSensitive {
		return name
	}
	return strings.ToLower(name)
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
		proto:                 metadata,
		rawDump:               schema,
		config:                config,
		isObjectCaseSensitive: isObjectCaseSensitive,
		isDetailCaseSensitive: isDetailCaseSensitive,
		searchPath:            normalizeSearchPath(metadata.SearchPath),
		internal:              make(map[string]*SchemaMetadata),
		linkedDatabase:        make(map[string]*storepb.LinkedDatabaseMetadata),
	}

	// Build a map of schema catalogs for quick lookup
	schemaCatalogMap := make(map[string]*storepb.SchemaCatalog)
	if config != nil {
		for _, schemaCatalog := range config.Schemas {
			schemaCatalogMap[schemaCatalog.Name] = schemaCatalog
		}
	}

	// Build schema metadata maps
	for _, s := range metadata.Schemas {
		// Get matching schema catalog if it exists
		schemaCatalog := schemaCatalogMap[s.Name]

		// Build a map of table catalogs for this schema
		tableCatalogMap := make(map[string]*storepb.TableCatalog)
		if schemaCatalog != nil {
			for _, tableCatalog := range schemaCatalog.Tables {
				tableCatalogMap[tableCatalog.Name] = tableCatalog
			}
		}

		schemaMetadata := &SchemaMetadata{
			isObjectCaseSensitive:    isObjectCaseSensitive,
			isDetailCaseSensitive:    isDetailCaseSensitive,
			internalTables:           make(map[string]*TableMetadata),
			internalExternalTable:    make(map[string]*ExternalTableMetadata),
			internalViews:            make(map[string]*storepb.ViewMetadata),
			internalMaterializedView: make(map[string]*storepb.MaterializedViewMetadata),
			internalProcedures:       make(map[string]*storepb.ProcedureMetadata),
			internalPackages:         make(map[string]*storepb.PackageMetadata),
			internalSequences:        make(map[string]*storepb.SequenceMetadata),
			proto:                    s,
			config:                   schemaCatalog,
		}
		for _, table := range s.Tables {
			tableCatalog := tableCatalogMap[table.Name]
			tables, names := buildTablesMetadata(table, tableCatalog, isDetailCaseSensitive)
			for i, table := range tables {
				tableID := normalizeNameByCaseSensitivity(names[i], isObjectCaseSensitive)
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
				columnID := normalizeNameByCaseSensitivity(column.Name, isDetailCaseSensitive)
				externalTableMetadata.internal[columnID] = column
			}
			tableID := normalizeNameByCaseSensitivity(externalTable.Name, isObjectCaseSensitive)
			schemaMetadata.internalExternalTable[tableID] = externalTableMetadata
		}
		for _, view := range s.Views {
			viewID := normalizeNameByCaseSensitivity(view.Name, isObjectCaseSensitive)
			schemaMetadata.internalViews[viewID] = view
		}
		for _, materializedView := range s.MaterializedViews {
			viewID := normalizeNameByCaseSensitivity(materializedView.Name, isObjectCaseSensitive)
			schemaMetadata.internalMaterializedView[viewID] = materializedView
		}
		for _, procedure := range s.Procedures {
			procedureID := normalizeNameByCaseSensitivity(procedure.Name, isDetailCaseSensitive)
			schemaMetadata.internalProcedures[procedureID] = procedure
		}
		for _, p := range s.Packages {
			packageID := normalizeNameByCaseSensitivity(p.Name, isDetailCaseSensitive)
			schemaMetadata.internalPackages[packageID] = p
		}
		for _, sequence := range s.Sequences {
			sequenceID := normalizeNameByCaseSensitivity(sequence.Name, isDetailCaseSensitive)
			schemaMetadata.internalSequences[sequenceID] = sequence
		}
		schemaID := normalizeNameByCaseSensitivity(s.Name, isObjectCaseSensitive)
		dbMetadata.internal[schemaID] = schemaMetadata
	}

	for _, dbLink := range metadata.LinkedDatabases {
		dbLinkID := normalizeNameByCaseSensitivity(dbLink.Name, isObjectCaseSensitive)
		dbMetadata.linkedDatabase[dbLinkID] = dbLink
	}

	return dbMetadata
}

func (d *DatabaseMetadata) GetProto() *storepb.DatabaseSchemaMetadata {
	return d.proto
}

func (d *DatabaseMetadata) GetRawDump() []byte {
	return d.rawDump
}

func (d *DatabaseMetadata) GetConfig() *storepb.DatabaseConfig {
	return d.config
}

func (d *DatabaseMetadata) GetSearchPath() []string {
	return d.searchPath
}

func (d *DatabaseMetadata) GetSchemaMetadata(name string) *SchemaMetadata {
	schemaID := normalizeNameByCaseSensitivity(name, d.isObjectCaseSensitive)
	return d.internal[schemaID]
}

func (d *DatabaseMetadata) DatabaseName() string {
	if d.proto == nil {
		return ""
	}
	return d.proto.Name
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

func (d *DatabaseMetadata) GetLinkedDatabase(name string) *storepb.LinkedDatabaseMetadata {
	nameID := normalizeNameByCaseSensitivity(name, d.isObjectCaseSensitive)
	return d.linkedDatabase[nameID]
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
	d.proto.Schemas = append(d.proto.Schemas, newSchemaProto)

	// Create SchemaMetadata wrapper
	schemaMeta := &SchemaMetadata{
		isObjectCaseSensitive:    d.isObjectCaseSensitive,
		isDetailCaseSensitive:    d.isDetailCaseSensitive,
		internalTables:           make(map[string]*TableMetadata),
		internalExternalTable:    make(map[string]*ExternalTableMetadata),
		internalViews:            make(map[string]*storepb.ViewMetadata),
		internalMaterializedView: make(map[string]*storepb.MaterializedViewMetadata),
		internalProcedures:       make(map[string]*storepb.ProcedureMetadata),
		internalPackages:         make(map[string]*storepb.PackageMetadata),
		internalSequences:        make(map[string]*storepb.SequenceMetadata),
		proto:                    newSchemaProto,
	}

	// Add to internal map
	schemaID := normalizeNameByCaseSensitivity(schemaName, d.isObjectCaseSensitive)
	d.internal[schemaID] = schemaMeta

	return schemaMeta
}

func (d *DatabaseMetadata) DropSchema(schemaName string) error {
	// Check if schema exists
	if d.GetSchemaMetadata(schemaName) == nil {
		return errors.Errorf("schema %q does not exist in database %q", schemaName, d.proto.GetName())
	}

	// Remove from internal map
	schemaID := normalizeNameByCaseSensitivity(schemaName, d.isObjectCaseSensitive)
	delete(d.internal, schemaID)

	// Remove from proto's schema list
	newSchemas := make([]*storepb.SchemaMetadata, 0, len(d.proto.Schemas)-1)
	for _, schema := range d.proto.Schemas {
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
	d.proto.Schemas = newSchemas

	return nil
}

// GetTable gets the schema by name.
func (s *SchemaMetadata) GetTable(name string) *TableMetadata {
	if s == nil {
		return nil
	}
	nameID := normalizeNameByCaseSensitivity(name, s.isObjectCaseSensitive)
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
func (s *SchemaMetadata) GetView(name string) *storepb.ViewMetadata {
	nameID := normalizeNameByCaseSensitivity(name, s.isObjectCaseSensitive)
	return s.internalViews[nameID]
}

func (s *SchemaMetadata) GetProcedure(name string) *storepb.ProcedureMetadata {
	nameID := normalizeNameByCaseSensitivity(name, s.isDetailCaseSensitive)
	return s.internalProcedures[nameID]
}

func (s *SchemaMetadata) GetPackage(name string) *storepb.PackageMetadata {
	nameID := normalizeNameByCaseSensitivity(name, s.isDetailCaseSensitive)
	return s.internalPackages[nameID]
}

// GetMaterializedView gets the materialized view by name.
func (s *SchemaMetadata) GetMaterializedView(name string) *storepb.MaterializedViewMetadata {
	nameID := normalizeNameByCaseSensitivity(name, s.isObjectCaseSensitive)
	return s.internalMaterializedView[nameID]
}

// GetExternalTable gets the external table by name.
func (s *SchemaMetadata) GetExternalTable(name string) *ExternalTableMetadata {
	nameID := normalizeNameByCaseSensitivity(name, s.isObjectCaseSensitive)
	return s.internalExternalTable[nameID]
}

// GetFunction gets the function by name.
// Note: For overloaded functions, this returns the first match by name only.
// Use signature-based lookup for precise matching.
func (s *SchemaMetadata) GetFunction(name string) *storepb.FunctionMetadata {
	for _, function := range s.proto.GetFunctions() {
		if s.isDetailCaseSensitive {
			if function.Name == name {
				return function
			}
		} else {
			if strings.EqualFold(function.Name, name) {
				return function
			}
		}
	}
	return nil
}

// GetSequence gets the sequence by name.
func (s *SchemaMetadata) GetSequence(name string) *storepb.SequenceMetadata {
	nameID := normalizeNameByCaseSensitivity(name, s.isDetailCaseSensitive)
	return s.internalSequences[nameID]
}

func (s *SchemaMetadata) GetSequencesByOwnerTable(name string) []*storepb.SequenceMetadata {
	var result []*storepb.SequenceMetadata
	for _, sequence := range s.internalSequences {
		if s.isObjectCaseSensitive {
			if sequence.OwnerTable == name {
				result = append(result, sequence)
			}
		} else {
			if strings.EqualFold(sequence.OwnerTable, name) {
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

// GetCatalog gets the catalog of SchemaMetadata.
func (s *SchemaMetadata) GetCatalog() *storepb.SchemaCatalog {
	return s.config
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
		result = append(result, procedure.GetName())
	}

	slices.Sort(result)
	return result
}

// ListViewNames lists the view names.
func (s *SchemaMetadata) ListViewNames() []string {
	var result []string
	for _, view := range s.internalViews {
		result = append(result, view.GetName())
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
		result = append(result, view.GetName())
	}

	slices.Sort(result)
	return result
}

// ListSequenceNames lists the sequence names.
func (s *SchemaMetadata) ListSequenceNames() []string {
	var result []string
	for _, sequence := range s.internalSequences {
		result = append(result, sequence.GetName())
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
		internalColumn:        make(map[string]*ColumnMetadata),
		internalIndexes:       make(map[string]*IndexMetadata),
		proto:                 newTableProto,
	}

	// Add to internal map
	tableID := normalizeNameByCaseSensitivity(tableName, s.isObjectCaseSensitive)
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
	tableID := normalizeNameByCaseSensitivity(tableName, s.isObjectCaseSensitive)
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
	oldTableID := normalizeNameByCaseSensitivity(oldName, s.isObjectCaseSensitive)
	delete(s.internalTables, oldTableID)

	// Update the table name in the proto
	oldTable.proto.Name = newName

	// Add back to internal map using new name
	newTableID := normalizeNameByCaseSensitivity(newName, s.isObjectCaseSensitive)
	s.internalTables[newTableID] = oldTable

	return nil
}

// CreateView creates a new view in the schema.
// Returns an error if the view already exists.
func (s *SchemaMetadata) CreateView(viewName string, definition string, dependencyColumns []*storepb.DependencyColumn) (*storepb.ViewMetadata, error) {
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

	// Add to internal map
	viewID := normalizeNameByCaseSensitivity(viewName, s.isObjectCaseSensitive)
	s.internalViews[viewID] = newViewProto

	return newViewProto, nil
}

// DropView drops a view from the schema.
// Returns an error if the view does not exist.
func (s *SchemaMetadata) DropView(viewName string) error {
	// Check if view exists
	if s.GetView(viewName) == nil {
		return errors.Errorf("view %q does not exist in schema %q", viewName, s.proto.Name)
	}

	// Remove from internal map
	viewID := normalizeNameByCaseSensitivity(viewName, s.isObjectCaseSensitive)
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
	oldViewID := normalizeNameByCaseSensitivity(oldName, s.isObjectCaseSensitive)
	delete(s.internalViews, oldViewID)

	// Update the view name in the proto
	oldView.Name = newName

	// Add back to internal map using new name
	newViewID := normalizeNameByCaseSensitivity(newName, s.isObjectCaseSensitive)
	s.internalViews[newViewID] = oldView

	return nil
}

// GetDependentViews returns all views that depend on the given table and column.
// This is used to check if a column can be dropped or if a table can be dropped.
func (s *SchemaMetadata) GetDependentViews(tableName string, columnName string) []string {
	var dependentViews []string

	for _, view := range s.internalViews {
		viewProto := view
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

func buildTablesMetadata(table *storepb.TableMetadata, tableCatalog *storepb.TableCatalog, isDetailCaseSensitive bool) ([]*TableMetadata, []string) {
	if table == nil {
		return nil, nil
	}

	// Build a map of column catalogs
	columnCatalogMap := make(map[string]*storepb.ColumnCatalog)
	if tableCatalog != nil {
		for _, columnCatalog := range tableCatalog.Columns {
			columnCatalogMap[columnCatalog.Name] = columnCatalog
		}
	}

	var result []*TableMetadata
	var name []string
	tableMetadata := &TableMetadata{
		isDetailCaseSensitive: isDetailCaseSensitive,
		internalColumn:        make(map[string]*ColumnMetadata),
		internalIndexes:       make(map[string]*IndexMetadata),
		proto:                 table,
		config:                tableCatalog,
	}
	for _, column := range table.Columns {
		columnCatalog := columnCatalogMap[column.Name]
		columnID := normalizeNameByCaseSensitivity(column.Name, isDetailCaseSensitive)
		tableMetadata.internalColumn[columnID] = &ColumnMetadata{
			proto:  column,
			config: columnCatalog,
		}
	}
	indexes := buildIndexesMetadata(table)
	for _, index := range indexes {
		indexID := normalizeNameByCaseSensitivity(index.proto.Name, isDetailCaseSensitive)
		tableMetadata.internalIndexes[indexID] = index
	}
	result = append(result, tableMetadata)
	name = append(name, table.Name)

	if table.Partitions != nil {
		partitionTables, partitionNames := buildTablesMetadataRecursive(table.Columns, columnCatalogMap, table.Partitions, tableMetadata, table, isDetailCaseSensitive)
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
func buildTablesMetadataRecursive(originalColumn []*storepb.ColumnMetadata, columnCatalogMap map[string]*storepb.ColumnCatalog, partitions []*storepb.TablePartitionMetadata, root *TableMetadata, proto *storepb.TableMetadata, isDetailCaseSensitive bool) ([]*TableMetadata, []string) {
	if partitions == nil {
		return nil, nil
	}

	var tables []*TableMetadata
	var names []string

	for _, partition := range partitions {
		partitionMetadata := &TableMetadata{
			partitionOf:    root,
			internalColumn: make(map[string]*ColumnMetadata),
			proto:          proto,
		}
		for _, column := range originalColumn {
			columnCatalog := columnCatalogMap[column.Name]
			columnID := normalizeNameByCaseSensitivity(column.Name, isDetailCaseSensitive)
			partitionMetadata.internalColumn[columnID] = &ColumnMetadata{
				proto:  column,
				config: columnCatalog,
			}
		}
		tables = append(tables, partitionMetadata)
		names = append(names, partition.Name)
		if partition.Subpartitions != nil {
			subTables, subNames := buildTablesMetadataRecursive(originalColumn, columnCatalogMap, partition.Subpartitions, partitionMetadata, proto, isDetailCaseSensitive)
			tables = append(tables, subTables...)
			names = append(names, subNames...)
		}
	}
	return tables, names
}

func (t *TableMetadata) GetOwner() string {
	return t.proto.Owner
}

func (t *TableMetadata) GetTableComment() string {
	return t.proto.Comment
}

// GetColumn gets the column by name.
func (t *TableMetadata) GetColumn(name string) *ColumnMetadata {
	if t == nil {
		return nil
	}
	nameID := normalizeNameByCaseSensitivity(name, t.isDetailCaseSensitive)
	return t.internalColumn[nameID]
}

func (t *TableMetadata) GetIndex(name string) *IndexMetadata {
	if t == nil {
		return nil
	}
	nameID := normalizeNameByCaseSensitivity(name, t.isDetailCaseSensitive)
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

func (t *TableMetadata) GetProto() *storepb.TableMetadata {
	return t.proto
}

func (t *TableMetadata) GetCatalog() *storepb.TableCatalog {
	return t.config
}

// CreateColumn creates a new column in the table.
// Returns an error if the column already exists.
func (t *TableMetadata) CreateColumn(columnProto *storepb.ColumnMetadata, columnCatalog *storepb.ColumnCatalog) error {
	// Check if column already exists
	if t.GetColumn(columnProto.Name) != nil {
		return errors.Errorf("column %q already exists in table %q", columnProto.Name, t.proto.Name)
	}

	// Add to proto's column list
	t.proto.Columns = append(t.proto.Columns, columnProto)

	// Create ColumnMetadata wrapper and add to internal map
	columnID := normalizeNameByCaseSensitivity(columnProto.Name, t.isDetailCaseSensitive)
	t.internalColumn[columnID] = &ColumnMetadata{
		proto:  columnProto,
		config: columnCatalog,
	}

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
	columnID := normalizeNameByCaseSensitivity(columnName, t.isDetailCaseSensitive)
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
	columnID := normalizeNameByCaseSensitivity(columnName, t.isDetailCaseSensitive)
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
	oldColumnID := normalizeNameByCaseSensitivity(oldName, t.isDetailCaseSensitive)
	delete(t.internalColumn, oldColumnID)

	// Update the column name in the proto
	oldColumn.proto.Name = newName

	// Add back to internal map using new name
	newColumnID := normalizeNameByCaseSensitivity(newName, t.isDetailCaseSensitive)
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

func (t *ExternalTableMetadata) GetProto() *storepb.ExternalTableMetadata {
	return t.proto
}

// GetColumn gets the column by name.
func (t *ExternalTableMetadata) GetColumn(name string) *storepb.ColumnMetadata {
	nameID := normalizeNameByCaseSensitivity(name, t.isDetailCaseSensitive)
	return t.internal[nameID]
}

func (i *IndexMetadata) GetProto() *storepb.IndexMetadata {
	return i.proto
}

func (i *IndexMetadata) GetTableProto() *storepb.TableMetadata {
	return i.tableProto
}

func (c *ColumnMetadata) GetProto() *storepb.ColumnMetadata {
	return c.proto
}

func (c *ColumnMetadata) GetCatalog() *storepb.ColumnCatalog {
	return c.config
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
	indexID := normalizeNameByCaseSensitivity(indexProto.Name, t.isDetailCaseSensitive)
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
	indexID := normalizeNameByCaseSensitivity(indexName, t.isDetailCaseSensitive)
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
	oldIndexID := normalizeNameByCaseSensitivity(oldName, t.isDetailCaseSensitive)
	delete(t.internalIndexes, oldIndexID)

	// Update the index name in the proto
	oldIndex.proto.Name = newName

	// Add back to internal map using new name
	newIndexID := normalizeNameByCaseSensitivity(newName, t.isDetailCaseSensitive)
	t.internalIndexes[newIndexID] = oldIndex

	return nil
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
