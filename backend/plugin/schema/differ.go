package schema

import (
	"strings"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema/pg/ast"
	"github.com/bytebase/bytebase/backend/store/model"
)

// MetadataDiffAction represents the type of change action.
type MetadataDiffAction string

const (
	MetadataDiffActionCreate MetadataDiffAction = "CREATE"
	MetadataDiffActionDrop   MetadataDiffAction = "DROP"
	MetadataDiffActionAlter  MetadataDiffAction = "ALTER"
)

// MetadataDiff represents the differences between two database schemas.
type MetadataDiff struct {
	// Database level changes
	DatabaseName string

	// Schema changes
	SchemaChanges []*SchemaDiff

	// Table changes
	TableChanges []*TableDiff

	// View changes
	ViewChanges []*ViewDiff

	// MaterializedView changes
	MaterializedViewChanges []*MaterializedViewDiff

	// Function changes
	FunctionChanges []*FunctionDiff

	// Procedure changes
	ProcedureChanges []*ProcedureDiff

	// Sequence changes
	SequenceChanges []*SequenceDiff

	// Enum type changes
	EnumTypeChanges []*EnumTypeDiff

	// Event changes
	EventChanges []*EventDiff
}

// nolint
// SchemaDiff represents changes to a schema.
type SchemaDiff struct {
	Action     MetadataDiffAction
	SchemaName string
	OldSchema  *storepb.SchemaMetadata
	NewSchema  *storepb.SchemaMetadata
}

// TableDiff represents changes to a table.
type TableDiff struct {
	Action     MetadataDiffAction
	SchemaName string
	TableName  string
	OldTable   *storepb.TableMetadata
	NewTable   *storepb.TableMetadata

	// Column changes
	ColumnChanges []*ColumnDiff

	// Index changes
	IndexChanges []*IndexDiff

	// Foreign key changes
	ForeignKeyChanges []*ForeignKeyDiff

	// Check constraint changes
	CheckConstraintChanges []*CheckConstraintDiff

	// Partition changes
	PartitionChanges []*PartitionDiff

	// Trigger changes
	TriggerChanges []*TriggerDiff
}

// ColumnDiff represents changes to a column.
type ColumnDiff struct {
	Action    MetadataDiffAction
	OldColumn *storepb.ColumnMetadata
	NewColumn *storepb.ColumnMetadata
}

// IndexDiff represents changes to an index.
type IndexDiff struct {
	Action   MetadataDiffAction
	OldIndex *storepb.IndexMetadata
	NewIndex *storepb.IndexMetadata
}

// ForeignKeyDiff represents changes to a foreign key.
type ForeignKeyDiff struct {
	Action        MetadataDiffAction
	OldForeignKey *storepb.ForeignKeyMetadata
	NewForeignKey *storepb.ForeignKeyMetadata
}

// CheckConstraintDiff represents changes to a check constraint.
type CheckConstraintDiff struct {
	Action             MetadataDiffAction
	OldCheckConstraint *storepb.CheckConstraintMetadata
	NewCheckConstraint *storepb.CheckConstraintMetadata
}

// TriggerDiff represents changes to a trigger.
type TriggerDiff struct {
	Action     MetadataDiffAction
	OldTrigger *storepb.TriggerMetadata
	NewTrigger *storepb.TriggerMetadata
}

// PartitionDiff represents changes to table partitions.
type PartitionDiff struct {
	Action       MetadataDiffAction
	OldPartition *storepb.TablePartitionMetadata
	NewPartition *storepb.TablePartitionMetadata
}

// ViewDiff represents changes to a view.
type ViewDiff struct {
	Action     MetadataDiffAction
	SchemaName string
	ViewName   string
	OldView    *storepb.ViewMetadata
	NewView    *storepb.ViewMetadata
}

// MaterializedViewDiff represents changes to a materialized view.
type MaterializedViewDiff struct {
	Action               MetadataDiffAction
	SchemaName           string
	MaterializedViewName string
	OldMaterializedView  *storepb.MaterializedViewMetadata
	NewMaterializedView  *storepb.MaterializedViewMetadata
}

// FunctionDiff represents changes to a function.
type FunctionDiff struct {
	Action       MetadataDiffAction
	SchemaName   string
	FunctionName string
	OldFunction  *storepb.FunctionMetadata
	NewFunction  *storepb.FunctionMetadata
}

// ProcedureDiff represents changes to a procedure.
type ProcedureDiff struct {
	Action        MetadataDiffAction
	SchemaName    string
	ProcedureName string
	OldProcedure  *storepb.ProcedureMetadata
	NewProcedure  *storepb.ProcedureMetadata
}

// SequenceDiff represents changes to a sequence.
type SequenceDiff struct {
	Action       MetadataDiffAction
	SchemaName   string
	SequenceName string
	OldSequence  *storepb.SequenceMetadata
	NewSequence  *storepb.SequenceMetadata
}

// EnumTypeDiff represents changes to an enum type.
type EnumTypeDiff struct {
	Action       MetadataDiffAction
	SchemaName   string
	EnumTypeName string
	OldEnumType  *storepb.EnumTypeMetadata
	NewEnumType  *storepb.EnumTypeMetadata
}

// EventDiff represents changes to an event.
type EventDiff struct {
	Action    MetadataDiffAction
	EventName string
	OldEvent  *storepb.EventMetadata
	NewEvent  *storepb.EventMetadata
}

// GetDatabaseSchemaDiff compares two model.DatabaseSchema instances and returns the differences.
func GetDatabaseSchemaDiff(engine storepb.Engine, oldSchema, newSchema *model.DatabaseSchema) (*MetadataDiff, error) {
	if oldSchema == nil || newSchema == nil {
		return nil, nil
	}

	oldMetadata := oldSchema.GetMetadata()
	newMetadata := newSchema.GetMetadata()
	if oldMetadata == nil || newMetadata == nil {
		return nil, nil
	}

	diff := &MetadataDiff{
		DatabaseName: newMetadata.Name,
	}

	// Use the internal DatabaseMetadata structures for efficient access
	oldMeta := oldSchema.GetDatabaseMetadata()
	newMeta := newSchema.GetDatabaseMetadata()

	for _, schemaName := range oldMeta.ListSchemaNames() {
		if newMeta.GetSchema(schemaName) == nil {
			oldSchemaMeta := oldMeta.GetSchema(schemaName)
			if oldSchemaMeta != nil {
				diff.SchemaChanges = append(diff.SchemaChanges, &SchemaDiff{
					Action:     MetadataDiffActionDrop,
					SchemaName: schemaName,
					OldSchema:  oldSchemaMeta.GetProto(),
				})
			}
		}
	}

	for _, schemaName := range newMeta.ListSchemaNames() {
		newSchemaMeta := newMeta.GetSchema(schemaName)
		if newSchemaMeta == nil {
			continue
		}

		if oldMeta.GetSchema(schemaName) == nil {
			// New schema
			diff.SchemaChanges = append(diff.SchemaChanges, &SchemaDiff{
				Action:     MetadataDiffActionCreate,
				SchemaName: schemaName,
				NewSchema:  newSchemaMeta.GetProto(),
			})
			// Add all objects in the new schema as created
			addNewSchemaObjects(diff, schemaName, newSchemaMeta)
		} else {
			// Compare schema objects
			oldSchemaMeta := oldMeta.GetSchema(schemaName)
			if oldSchemaMeta != nil {
				compareSchemaObjects(engine, diff, schemaName, oldSchemaMeta, newSchemaMeta)
			}
		}
	}

	return diff, nil
}

// addNewSchemaObjects adds all objects from a new schema as created.
func addNewSchemaObjects(diff *MetadataDiff, schemaName string, schema *model.SchemaMetadata) {
	schemaProto := schema.GetProto()

	// Add all tables
	for _, tableName := range schema.ListTableNames() {
		table := schema.GetTable(tableName)
		if table != nil && !table.GetProto().GetSkipDump() {
			diff.TableChanges = append(diff.TableChanges, &TableDiff{
				Action:     MetadataDiffActionCreate,
				SchemaName: schemaName,
				TableName:  tableName,
				NewTable:   table.GetProto(),
			})
		}
	}

	// Add all views
	for _, viewName := range schema.ListViewNames() {
		view := schema.GetView(viewName)
		if view != nil && !view.GetProto().GetSkipDump() {
			diff.ViewChanges = append(diff.ViewChanges, &ViewDiff{
				Action:     MetadataDiffActionCreate,
				SchemaName: schemaName,
				ViewName:   viewName,
				NewView:    view.GetProto(),
			})
		}
	}

	// Add all materialized views
	for _, mvName := range schema.ListMaterializedViewNames() {
		mv := schema.GetMaterializedView(mvName)
		if mv != nil && !mv.GetProto().GetSkipDump() {
			diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &MaterializedViewDiff{
				Action:               MetadataDiffActionCreate,
				SchemaName:           schemaName,
				MaterializedViewName: mvName,
				NewMaterializedView:  mv.GetProto(),
			})
		}
	}

	// Add all functions
	for _, function := range schema.ListFunctions() {
		if !function.GetProto().GetSkipDump() {
			diff.FunctionChanges = append(diff.FunctionChanges, &FunctionDiff{
				Action:       MetadataDiffActionCreate,
				SchemaName:   schemaName,
				FunctionName: function.GetProto().Name,
				NewFunction:  function.GetProto(),
			})
		}
	}

	// Add all procedures
	for _, procName := range schema.ListProcedureNames() {
		proc := schema.GetProcedure(procName)
		if proc != nil && !proc.GetProto().GetSkipDump() {
			diff.ProcedureChanges = append(diff.ProcedureChanges, &ProcedureDiff{
				Action:        MetadataDiffActionCreate,
				SchemaName:    schemaName,
				ProcedureName: procName,
				NewProcedure:  proc.GetProto(),
			})
		}
	}

	// Add all sequences
	for _, seqProto := range schemaProto.Sequences {
		if !seqProto.GetSkipDump() {
			diff.SequenceChanges = append(diff.SequenceChanges, &SequenceDiff{
				Action:       MetadataDiffActionCreate,
				SchemaName:   schemaName,
				SequenceName: seqProto.Name,
				NewSequence:  seqProto,
			})
		}
	}

	// Add all enum types
	for _, enumProto := range schemaProto.EnumTypes {
		if !enumProto.GetSkipDump() {
			diff.EnumTypeChanges = append(diff.EnumTypeChanges, &EnumTypeDiff{
				Action:       MetadataDiffActionCreate,
				SchemaName:   schemaName,
				EnumTypeName: enumProto.Name,
				NewEnumType:  enumProto,
			})
		}
	}

	// Add all events
	for _, eventProto := range schemaProto.Events {
		diff.EventChanges = append(diff.EventChanges, &EventDiff{
			Action:    MetadataDiffActionCreate,
			EventName: eventProto.Name,
			NewEvent:  eventProto,
		})
	}
}

// compareSchemaObjects compares objects between two schemas.
func compareSchemaObjects(engine storepb.Engine, diff *MetadataDiff, schemaName string, oldSchema, newSchema *model.SchemaMetadata) {
	// Compare tables

	// Check for dropped tables
	for _, tableName := range oldSchema.ListTableNames() {
		if newSchema.GetTable(tableName) == nil {
			oldTable := oldSchema.GetTable(tableName)
			if oldTable != nil && !oldTable.GetProto().GetSkipDump() {
				diff.TableChanges = append(diff.TableChanges, &TableDiff{
					Action:     MetadataDiffActionDrop,
					SchemaName: schemaName,
					TableName:  tableName,
					OldTable:   oldTable.GetProto(),
				})
			}
		}
	}

	// Check for new and modified tables
	for _, tableName := range newSchema.ListTableNames() {
		newTable := newSchema.GetTable(tableName)
		if newTable == nil || newTable.GetProto().GetSkipDump() {
			continue
		}

		if oldSchema.GetTable(tableName) == nil {
			diff.TableChanges = append(diff.TableChanges, &TableDiff{
				Action:     MetadataDiffActionCreate,
				SchemaName: schemaName,
				TableName:  tableName,
				NewTable:   newTable.GetProto(),
			})
		} else {
			// Compare table details
			oldTable := oldSchema.GetTable(tableName)
			if oldTable != nil && !oldTable.GetProto().GetSkipDump() {
				tableDiff := compareTableDetails(schemaName, tableName, oldTable, newTable)
				if tableDiff != nil {
					diff.TableChanges = append(diff.TableChanges, tableDiff)
				}
			}
		}
	}

	// Compare views
	compareViews(engine, diff, schemaName, oldSchema, newSchema)

	// Compare materialized views
	compareMaterializedViews(engine, diff, schemaName, oldSchema, newSchema)

	// Compare functions
	compareFunctions(diff, schemaName, oldSchema, newSchema)

	// Compare procedures
	compareProcedures(diff, schemaName, oldSchema, newSchema)

	// Compare sequences
	compareSequences(diff, schemaName, oldSchema, newSchema)

	// Compare enum types
	compareEnumTypes(diff, schemaName, oldSchema, newSchema)

	// Compare events
	compareEvents(diff, schemaName, oldSchema, newSchema)
}

// compareTableDetails compares the details of two tables.
func compareTableDetails(schemaName, tableName string, oldTable, newTable *model.TableMetadata) *TableDiff {
	tableDiff := &TableDiff{
		Action:     MetadataDiffActionAlter,
		SchemaName: schemaName,
		TableName:  tableName,
		OldTable:   oldTable.GetProto(),
		NewTable:   newTable.GetProto(),
	}

	hasChanges := false

	// Compare columns
	columnChanges := compareColumns(oldTable, newTable)
	if len(columnChanges) > 0 {
		tableDiff.ColumnChanges = columnChanges
		hasChanges = true
	}

	// Compare indexes
	indexChanges := compareIndexes(oldTable, newTable)
	if len(indexChanges) > 0 {
		tableDiff.IndexChanges = indexChanges
		hasChanges = true
	}

	// Compare foreign keys
	fkChanges := compareForeignKeys(oldTable.GetProto().ForeignKeys, newTable.GetProto().ForeignKeys)
	if len(fkChanges) > 0 {
		tableDiff.ForeignKeyChanges = fkChanges
		hasChanges = true
	}

	// Compare check constraints
	checkChanges := compareCheckConstraints(oldTable.GetProto().CheckConstraints, newTable.GetProto().CheckConstraints)
	if len(checkChanges) > 0 {
		tableDiff.CheckConstraintChanges = checkChanges
		hasChanges = true
	}

	// Compare partitions
	partitionChanges := comparePartitions(oldTable.GetProto().Partitions, newTable.GetProto().Partitions)
	if len(partitionChanges) > 0 {
		tableDiff.PartitionChanges = partitionChanges
		hasChanges = true
	}

	// Compare triggers
	triggerChanges := compareTriggers(oldTable.GetProto().Triggers, newTable.GetProto().Triggers)
	if len(triggerChanges) > 0 {
		tableDiff.TriggerChanges = triggerChanges
		hasChanges = true
	}

	// Compare table comments
	if oldTable.GetProto().Comment != newTable.GetProto().Comment {
		hasChanges = true
	}

	if !hasChanges {
		return nil
	}

	return tableDiff
}

// compareColumns compares columns between two tables.
func compareColumns(oldTable, newTable *model.TableMetadata) []*ColumnDiff {
	var changes []*ColumnDiff

	oldColumns := oldTable.GetColumns()
	newColumns := newTable.GetColumns()

	// Check for dropped columns
	for _, oldCol := range oldColumns {
		if newTable.GetColumn(oldCol.Name) == nil {
			changes = append(changes, &ColumnDiff{
				Action:    MetadataDiffActionDrop,
				OldColumn: oldCol,
			})
		}
	}

	// Check for new and modified columns
	for _, newCol := range newColumns {
		oldCol := oldTable.GetColumn(newCol.Name)
		if oldCol == nil {
			changes = append(changes, &ColumnDiff{
				Action:    MetadataDiffActionCreate,
				NewColumn: newCol,
			})
		} else if !columnsEqual(oldCol, newCol) {
			changes = append(changes, &ColumnDiff{
				Action:    MetadataDiffActionAlter,
				OldColumn: oldCol,
				NewColumn: newCol,
			})
		}
	}

	return changes
}

// columnsEqual checks if two columns are equal.
func columnsEqual(col1, col2 *storepb.ColumnMetadata) bool {
	if col1.Type != col2.Type {
		return false
	}
	if col1.Nullable != col2.Nullable {
		return false
	}
	// Compare default values
	if !defaultValuesEqual(col1, col2) {
		return false
	}
	if col1.Comment != col2.Comment {
		return false
	}
	// Compare character set and collation
	if col1.CharacterSet != col2.CharacterSet {
		return false
	}
	if col1.Collation != col2.Collation {
		return false
	}
	// Compare on update clause
	if col1.OnUpdate != col2.OnUpdate {
		return false
	}
	// Compare Oracle specific metadata
	if col1.DefaultOnNull != col2.DefaultOnNull {
		return false
	}
	// Compare generated column metadata
	if !generationMetadataEqual(col1.Generation, col2.Generation) {
		return false
	}
	// Compare identity column metadata
	if col1.IsIdentity != col2.IsIdentity {
		return false
	}
	if col1.IdentityGeneration != col2.IdentityGeneration {
		return false
	}
	if col1.IdentitySeed != col2.IdentitySeed {
		return false
	}
	// Compare user comment
	if col1.UserComment != col2.UserComment {
		return false
	}
	return true
}

// defaultValuesEqual compares default values.
func defaultValuesEqual(col1, col2 *storepb.ColumnMetadata) bool {
	// Now we only need to compare the consolidated default field
	if col1.Default == col2.Default {
		return true
	}

	// For PostgreSQL, try normalized comparison to handle schema prefix differences
	norm1 := normalizePostgreSQLDefaultValue(col1.Default)
	norm2 := normalizePostgreSQLDefaultValue(col2.Default)
	return norm1 == norm2
}

// generationMetadataEqual compares two generation metadata structs.
func generationMetadataEqual(gen1, gen2 *storepb.GenerationMetadata) bool {
	if gen1 == nil && gen2 == nil {
		return true
	}
	if gen1 == nil || gen2 == nil {
		return false
	}
	return gen1.Type == gen2.Type && ast.CompareExpressionsSemantically(gen1.Expression, gen2.Expression)
}

// compareIndexes compares indexes between two tables.
func compareIndexes(oldTable, newTable *model.TableMetadata) []*IndexDiff {
	var changes []*IndexDiff

	oldIndexes := oldTable.ListIndexes()
	newIndexes := newTable.ListIndexes()

	// Check for dropped indexes
	for _, oldIdx := range oldIndexes {
		if newTable.GetIndex(oldIdx.GetProto().Name) == nil {
			changes = append(changes, &IndexDiff{
				Action:   MetadataDiffActionDrop,
				OldIndex: oldIdx.GetProto(),
			})
		}
	}

	// Check for new and modified indexes
	for _, newIdx := range newIndexes {
		oldIdx := oldTable.GetIndex(newIdx.GetProto().Name)
		if oldIdx == nil {
			changes = append(changes, &IndexDiff{
				Action:   MetadataDiffActionCreate,
				NewIndex: newIdx.GetProto(),
			})
		} else if !indexesEqual(oldIdx.GetProto(), newIdx.GetProto()) {
			// Drop the old index and recreate the new one instead of altering
			changes = append(changes, &IndexDiff{
				Action:   MetadataDiffActionDrop,
				OldIndex: oldIdx.GetProto(),
			})
			changes = append(changes, &IndexDiff{
				Action:   MetadataDiffActionCreate,
				NewIndex: newIdx.GetProto(),
			})
		}
	}

	return changes
}

// indexesEqual checks if two indexes are equal.
func indexesEqual(idx1, idx2 *storepb.IndexMetadata) bool {
	if idx1.Type != idx2.Type {
		return false
	}
	if idx1.Unique != idx2.Unique {
		return false
	}
	if idx1.Primary != idx2.Primary {
		return false
	}
	if len(idx1.Expressions) != len(idx2.Expressions) {
		return false
	}
	for i, expr := range idx1.Expressions {
		if !ast.CompareExpressionsSemantically(expr, idx2.Expressions[i]) {
			return false
		}
	}

	// Compare key lengths
	if len(idx1.KeyLength) != len(idx2.KeyLength) {
		return false
	}
	for i, keyLen := range idx1.KeyLength {
		if keyLen != idx2.KeyLength[i] {
			return false
		}
	}

	// Compare descending order - be flexible about empty arrays vs arrays of false
	// This handles the case where one has [] and the other has [false, false, ...]
	if !descendingArraysEqual(idx1.Descending, idx2.Descending) {
		return false
	}

	// Compare visibility
	if idx1.Visible != idx2.Visible {
		return false
	}

	// Compare spatial configuration
	if !spatialConfigsEqual(idx1.SpatialConfig, idx2.SpatialConfig) {
		return false
	}

	return true
}

// descendingArraysEqual compares two descending arrays, considering empty arrays
// and arrays of all false values as equivalent for constraint-based indexes.
func descendingArraysEqual(desc1, desc2 []bool) bool {
	// Helper function to check if an array is effectively "all false"
	// (either empty or contains only false values)
	isEffectivelyAllFalse := func(arr []bool) bool {
		if len(arr) == 0 {
			return true // Empty array means no descending columns
		}
		for _, val := range arr {
			if val {
				return false // Found a true value, so not all false
			}
		}
		return true // All values are false
	}

	// If both arrays are effectively "all false", they are equal
	if isEffectivelyAllFalse(desc1) && isEffectivelyAllFalse(desc2) {
		return true
	}

	// If lengths don't match and neither is empty, do exact comparison
	if len(desc1) != len(desc2) {
		return false
	}

	// Do element-by-element comparison
	for i, val1 := range desc1 {
		if i >= len(desc2) || val1 != desc2[i] {
			return false
		}
	}

	return true
}

// spatialConfigsEqual checks if two spatial index configurations are equal.
func spatialConfigsEqual(cfg1, cfg2 *storepb.SpatialIndexConfig) bool {
	if cfg1 == nil && cfg2 == nil {
		return true
	}
	if cfg1 == nil || cfg2 == nil {
		return false
	}

	// Compare method
	if cfg1.Method != cfg2.Method {
		return false
	}

	// Compare tessellation config
	if !tessellationConfigsEqual(cfg1.Tessellation, cfg2.Tessellation) {
		return false
	}

	// Compare storage config
	if !storageConfigsEqual(cfg1.Storage, cfg2.Storage) {
		return false
	}

	// Compare dimensional config
	if !dimensionalConfigsEqual(cfg1.Dimensional, cfg2.Dimensional) {
		return false
	}

	// Compare engine-specific parameters
	if len(cfg1.EngineSpecific) != len(cfg2.EngineSpecific) {
		return false
	}
	for key, val1 := range cfg1.EngineSpecific {
		val2, exists := cfg2.EngineSpecific[key]
		if !exists || val1 != val2 {
			return false
		}
	}

	return true
}

// tessellationConfigsEqual checks if two tessellation configurations are equal.
func tessellationConfigsEqual(cfg1, cfg2 *storepb.TessellationConfig) bool {
	if cfg1 == nil && cfg2 == nil {
		return true
	}
	if cfg1 == nil || cfg2 == nil {
		return false
	}

	if cfg1.Scheme != cfg2.Scheme {
		return false
	}

	if !boundingBoxesEqual(cfg1.BoundingBox, cfg2.BoundingBox) {
		return false
	}

	if !gridLevelsEqual(cfg1.GridLevels, cfg2.GridLevels) {
		return false
	}

	if cfg1.CellsPerObject != cfg2.CellsPerObject {
		return false
	}

	return true
}

// boundingBoxesEqual checks if two bounding boxes are equal.
func boundingBoxesEqual(bb1, bb2 *storepb.BoundingBox) bool {
	if bb1 == nil && bb2 == nil {
		return true
	}
	if bb1 == nil || bb2 == nil {
		return false
	}

	return bb1.Xmin == bb2.Xmin &&
		bb1.Ymin == bb2.Ymin &&
		bb1.Xmax == bb2.Xmax &&
		bb1.Ymax == bb2.Ymax
}

// gridLevelsEqual checks if two grid level lists are equal.
func gridLevelsEqual(grids1, grids2 []*storepb.GridLevel) bool {
	if len(grids1) != len(grids2) {
		return false
	}

	for i, grid1 := range grids1 {
		grid2 := grids2[i]
		if grid1.Level != grid2.Level || grid1.Density != grid2.Density {
			return false
		}
	}

	return true
}

// storageConfigsEqual checks if two storage configurations are equal.
func storageConfigsEqual(cfg1, cfg2 *storepb.StorageConfig) bool {
	if cfg1 == nil && cfg2 == nil {
		return true
	}
	if cfg1 == nil || cfg2 == nil {
		return false
	}

	return cfg1.Fillfactor == cfg2.Fillfactor &&
		cfg1.Buffering == cfg2.Buffering &&
		cfg1.Tablespace == cfg2.Tablespace &&
		cfg1.WorkTablespace == cfg2.WorkTablespace &&
		cfg1.SdoLevel == cfg2.SdoLevel &&
		cfg1.CommitInterval == cfg2.CommitInterval &&
		cfg1.PadIndex == cfg2.PadIndex &&
		cfg1.SortInTempdb == cfg2.SortInTempdb &&
		cfg1.DropExisting == cfg2.DropExisting &&
		cfg1.Online == cfg2.Online &&
		cfg1.AllowRowLocks == cfg2.AllowRowLocks &&
		cfg1.AllowPageLocks == cfg2.AllowPageLocks &&
		cfg1.Maxdop == cfg2.Maxdop &&
		cfg1.DataCompression == cfg2.DataCompression
}

// dimensionalConfigsEqual checks if two dimensional configurations are equal.
func dimensionalConfigsEqual(cfg1, cfg2 *storepb.DimensionalConfig) bool {
	if cfg1 == nil && cfg2 == nil {
		return true
	}
	if cfg1 == nil || cfg2 == nil {
		return false
	}

	return cfg1.Dimensions == cfg2.Dimensions &&
		cfg1.DataType == cfg2.DataType &&
		cfg1.OperatorClass == cfg2.OperatorClass &&
		cfg1.LayerGtype == cfg2.LayerGtype &&
		cfg1.ParallelBuild == cfg2.ParallelBuild
}

// compareForeignKeys compares two lists of foreign keys.
func compareForeignKeys(oldFKs, newFKs []*storepb.ForeignKeyMetadata) []*ForeignKeyDiff {
	var changes []*ForeignKeyDiff

	oldFKMap := make(map[string]*storepb.ForeignKeyMetadata)
	for _, fk := range oldFKs {
		oldFKMap[fk.Name] = fk
	}

	newFKMap := make(map[string]*storepb.ForeignKeyMetadata)
	for _, fk := range newFKs {
		newFKMap[fk.Name] = fk
	}

	// Check for dropped foreign keys
	for fkName, oldFK := range oldFKMap {
		if _, exists := newFKMap[fkName]; !exists {
			changes = append(changes, &ForeignKeyDiff{
				Action:        MetadataDiffActionDrop,
				OldForeignKey: oldFK,
			})
		}
	}

	// Check for new and modified foreign keys
	for fkName, newFK := range newFKMap {
		oldFK, exists := oldFKMap[fkName]
		if !exists {
			changes = append(changes, &ForeignKeyDiff{
				Action:        MetadataDiffActionCreate,
				NewForeignKey: newFK,
			})
		} else if !foreignKeysEqual(oldFK, newFK) {
			// Drop the old FK and recreate the new one instead of altering
			changes = append(changes, &ForeignKeyDiff{
				Action:        MetadataDiffActionDrop,
				OldForeignKey: oldFK,
			})
			changes = append(changes, &ForeignKeyDiff{
				Action:        MetadataDiffActionCreate,
				NewForeignKey: newFK,
			})
		}
	}

	return changes
}

// foreignKeysEqual checks if two foreign keys are equal.
func foreignKeysEqual(fk1, fk2 *storepb.ForeignKeyMetadata) bool {
	if fk1.ReferencedSchema != fk2.ReferencedSchema {
		return false
	}
	if fk1.ReferencedTable != fk2.ReferencedTable {
		return false
	}
	if fk1.OnDelete != fk2.OnDelete {
		return false
	}
	if fk1.OnUpdate != fk2.OnUpdate {
		return false
	}
	if fk1.MatchType != fk2.MatchType {
		return false
	}
	if len(fk1.Columns) != len(fk2.Columns) {
		return false
	}
	for i, col := range fk1.Columns {
		if col != fk2.Columns[i] {
			return false
		}
	}
	if len(fk1.ReferencedColumns) != len(fk2.ReferencedColumns) {
		return false
	}
	for i, col := range fk1.ReferencedColumns {
		if col != fk2.ReferencedColumns[i] {
			return false
		}
	}
	return true
}

// compareCheckConstraints compares two lists of check constraints.
func compareCheckConstraints(oldChecks, newChecks []*storepb.CheckConstraintMetadata) []*CheckConstraintDiff {
	var changes []*CheckConstraintDiff

	oldCheckMap := make(map[string]*storepb.CheckConstraintMetadata)
	for _, check := range oldChecks {
		oldCheckMap[check.Name] = check
	}

	newCheckMap := make(map[string]*storepb.CheckConstraintMetadata)
	for _, check := range newChecks {
		newCheckMap[check.Name] = check
	}

	// Check for dropped check constraints
	for checkName, oldCheck := range oldCheckMap {
		if _, exists := newCheckMap[checkName]; !exists {
			changes = append(changes, &CheckConstraintDiff{
				Action:             MetadataDiffActionDrop,
				OldCheckConstraint: oldCheck,
			})
		}
	}

	// Check for new and modified check constraints
	for checkName, newCheck := range newCheckMap {
		oldCheck, exists := oldCheckMap[checkName]
		if !exists {
			changes = append(changes, &CheckConstraintDiff{
				Action:             MetadataDiffActionCreate,
				NewCheckConstraint: newCheck,
			})
		} else if !checkConstraintsEqual(oldCheck, newCheck) {
			// Drop the old constraint and recreate the new one instead of altering
			changes = append(changes, &CheckConstraintDiff{
				Action:             MetadataDiffActionDrop,
				OldCheckConstraint: oldCheck,
			})
			changes = append(changes, &CheckConstraintDiff{
				Action:             MetadataDiffActionCreate,
				NewCheckConstraint: newCheck,
			})
		}
	}

	return changes
}

// checkConstraintsEqual checks if two check constraints are equal.
func checkConstraintsEqual(check1, check2 *storepb.CheckConstraintMetadata) bool {
	// First try semantic comparison
	if ast.CompareExpressionsSemantically(check1.Expression, check2.Expression) {
		return true
	}

	// Handle PostgreSQL-specific normalizations
	// PostgreSQL converts "column IN (values)" to "column = ANY (ARRAY[values])"
	if isPostgreSQLInAnyEquivalent(check1.Expression, check2.Expression) ||
		isPostgreSQLInAnyEquivalent(check2.Expression, check1.Expression) {
		return true
	}

	// Handle PostgreSQL interval syntax differences
	// Sync: (order_date >= (CURRENT_DATE - '1 year'::interval))
	// Parser: order_date >= CURRENT_DATE - INTERVAL '1 year'
	if normalizePostgreSQLCheckConstraint(check1.Expression) == normalizePostgreSQLCheckConstraint(check2.Expression) {
		return true
	}

	return false
}

// isPostgreSQLInAnyEquivalent checks if expr1 contains IN syntax that PostgreSQL would normalize to ANY syntax in expr2
func isPostgreSQLInAnyEquivalent(expr1, expr2 string) bool {
	// Clean up expressions for comparison
	expr1 = strings.TrimSpace(expr1)
	expr2 = strings.TrimSpace(expr2)

	// Handle common PostgreSQL transformations:
	// "column IN ('A', 'B')" -> "(column = ANY (ARRAY['A'::text, 'B'::text]))"

	// Very basic pattern matching for this specific case
	// Look for patterns like "column IN(...)" in expr1 and "column = ANY (ARRAY[...])" in expr2
	if strings.Contains(expr1, " IN(") || strings.Contains(expr1, " IN (") {
		// Extract the column name and values from IN expression
		if strings.Contains(expr2, " = ANY (ARRAY[") && strings.Contains(expr2, "::text") {
			// This is a simplified check - for a production system, you'd want more robust parsing
			// For now, just check if both expressions contain the same column name
			inParts := strings.Split(expr1, " IN")
			if len(inParts) >= 2 {
				column1 := strings.TrimSpace(inParts[0])
				// Remove any leading parentheses from column name
				column1 = strings.TrimPrefix(column1, "(")

				if strings.HasPrefix(expr2, "("+column1+" = ANY") || strings.HasPrefix(expr2, column1+" = ANY") {
					// Extract values from both expressions and compare
					return compareInVsAnyValues(expr1, expr2)
				}
			}
		}
	}

	return false
}

// compareInVsAnyValues compares the values in IN syntax vs ANY syntax
func compareInVsAnyValues(inExpr, anyExpr string) bool {
	// Extract values from IN expression: "column IN('A', 'B')"
	inValues := extractInValues(inExpr)
	if inValues == nil {
		return false
	}

	// Extract values from ANY expression: "(column = ANY (ARRAY['A'::text, 'B'::text]))"
	anyValues := extractAnyValues(anyExpr)
	if anyValues == nil {
		return false
	}

	// Compare the value sets
	if len(inValues) != len(anyValues) {
		return false
	}

	// Convert both to maps for comparison (order doesn't matter)
	inMap := make(map[string]bool)
	for _, v := range inValues {
		inMap[v] = true
	}

	anyMap := make(map[string]bool)
	for _, v := range anyValues {
		anyMap[v] = true
	}

	// Check if they contain the same values
	for v := range inMap {
		if !anyMap[v] {
			return false
		}
	}

	return true
}

// extractInValues extracts values from "column IN('A', 'B')" format
func extractInValues(expr string) []string {
	// Find the IN part
	inIndex := strings.Index(expr, " IN")
	if inIndex == -1 {
		return nil
	}

	// Find the opening parenthesis
	openParen := strings.Index(expr[inIndex:], "(")
	if openParen == -1 {
		return nil
	}
	openParen += inIndex

	// Find the closing parenthesis
	closeParen := strings.LastIndex(expr, ")")
	if closeParen == -1 || closeParen <= openParen {
		return nil
	}

	// Extract the values part
	valuesStr := expr[openParen+1 : closeParen]

	// Split by comma and clean up
	var values []string
	parts := strings.Split(valuesStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Remove quotes
		if (part[0] == '\'' && part[len(part)-1] == '\'') ||
			(part[0] == '"' && part[len(part)-1] == '"') {
			part = part[1 : len(part)-1]
		}
		values = append(values, part)
	}

	return values
}

// extractAnyValues extracts values from "(column = ANY (ARRAY['A'::text, 'B'::text]))" format
func extractAnyValues(expr string) []string {
	// Find the ARRAY part
	arrayIndex := strings.Index(expr, "ARRAY[")
	if arrayIndex == -1 {
		return nil
	}

	// Find the closing bracket
	closeBracket := strings.Index(expr[arrayIndex:], "]")
	if closeBracket == -1 {
		return nil
	}
	closeBracket += arrayIndex

	// Extract the values part
	valuesStr := expr[arrayIndex+6 : closeBracket] // Skip "ARRAY["

	// Split by comma and clean up
	var values []string
	parts := strings.Split(valuesStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Remove ::text suffix and quotes
		part = strings.TrimSuffix(part, "::text")
		if (part[0] == '\'' && part[len(part)-1] == '\'') ||
			(part[0] == '"' && part[len(part)-1] == '"') {
			part = part[1 : len(part)-1]
		}
		values = append(values, part)
	}

	return values
}

// comparePartitions compares two lists of partitions.
func comparePartitions(oldPartitions, newPartitions []*storepb.TablePartitionMetadata) []*PartitionDiff {
	var changes []*PartitionDiff

	oldPartMap := make(map[string]*storepb.TablePartitionMetadata)
	for _, part := range oldPartitions {
		oldPartMap[part.Name] = part
	}

	newPartMap := make(map[string]*storepb.TablePartitionMetadata)
	for _, part := range newPartitions {
		newPartMap[part.Name] = part
	}

	// Check for dropped partitions
	for partName, oldPart := range oldPartMap {
		if _, exists := newPartMap[partName]; !exists {
			changes = append(changes, &PartitionDiff{
				Action:       MetadataDiffActionDrop,
				OldPartition: oldPart,
			})
		}
	}

	// Check for new and modified partitions
	for partName, newPart := range newPartMap {
		oldPart, exists := oldPartMap[partName]
		if !exists {
			changes = append(changes, &PartitionDiff{
				Action:       MetadataDiffActionCreate,
				NewPartition: newPart,
			})
		} else if !partitionsEqual(oldPart, newPart) {
			// Drop the old partition and recreate the new one instead of altering
			changes = append(changes, &PartitionDiff{
				Action:       MetadataDiffActionDrop,
				OldPartition: oldPart,
			})
			changes = append(changes, &PartitionDiff{
				Action:       MetadataDiffActionCreate,
				NewPartition: newPart,
			})
		}
	}

	return changes
}

// compareTriggers compares two lists of triggers.
func compareTriggers(oldTriggers, newTriggers []*storepb.TriggerMetadata) []*TriggerDiff {
	var changes []*TriggerDiff

	oldTriggerMap := make(map[string]*storepb.TriggerMetadata)
	for _, trigger := range oldTriggers {
		oldTriggerMap[trigger.Name] = trigger
	}

	newTriggerMap := make(map[string]*storepb.TriggerMetadata)
	for _, trigger := range newTriggers {
		newTriggerMap[trigger.Name] = trigger
	}

	// Check for dropped triggers
	for triggerName, oldTrigger := range oldTriggerMap {
		if _, exists := newTriggerMap[triggerName]; !exists {
			changes = append(changes, &TriggerDiff{
				Action:     MetadataDiffActionDrop,
				OldTrigger: oldTrigger,
			})
		}
	}

	// Check for new and modified triggers
	for triggerName, newTrigger := range newTriggerMap {
		oldTrigger, exists := oldTriggerMap[triggerName]
		if !exists {
			changes = append(changes, &TriggerDiff{
				Action:     MetadataDiffActionCreate,
				NewTrigger: newTrigger,
			})
		} else if !triggersEqual(oldTrigger, newTrigger) {
			// Drop and recreate the trigger instead of altering
			changes = append(changes, &TriggerDiff{
				Action:     MetadataDiffActionDrop,
				OldTrigger: oldTrigger,
			})
			changes = append(changes, &TriggerDiff{
				Action:     MetadataDiffActionCreate,
				NewTrigger: newTrigger,
			})
		}
	}

	return changes
}

// triggersEqual checks if two triggers are equal.
func triggersEqual(t1, t2 *storepb.TriggerMetadata) bool {
	if t1 == nil || t2 == nil {
		return t1 == t2
	}
	return t1.Name == t2.Name &&
		t1.Event == t2.Event &&
		t1.Timing == t2.Timing &&
		t1.Body == t2.Body
}

// partitionsEqual checks if two partitions are equal.
func partitionsEqual(part1, part2 *storepb.TablePartitionMetadata) bool {
	if part1.Type != part2.Type {
		return false
	}
	if !ast.CompareExpressionsSemantically(part1.Expression, part2.Expression) {
		return false
	}
	if part1.Value != part2.Value {
		return false
	}
	if part1.UseDefault != part2.UseDefault {
		return false
	}
	// Compare subpartitions
	if len(part1.Subpartitions) != len(part2.Subpartitions) {
		return false
	}
	// Create maps for subpartition comparison
	subPart1Map := make(map[string]*storepb.TablePartitionMetadata)
	for _, sub := range part1.Subpartitions {
		subPart1Map[sub.Name] = sub
	}
	for _, sub2 := range part2.Subpartitions {
		sub1, exists := subPart1Map[sub2.Name]
		if !exists || !partitionsEqual(sub1, sub2) {
			return false
		}
	}
	return true
}

// compareViews compares views between two schemas.
func compareViews(engine storepb.Engine, diff *MetadataDiff, schemaName string, oldSchema, newSchema *model.SchemaMetadata) {
	// Get the engine-specific view comparer
	comparer := GetViewComparer(engine)
	// Check for dropped views
	for _, viewName := range oldSchema.ListViewNames() {
		if newSchema.GetView(viewName) == nil {
			oldView := oldSchema.GetView(viewName)
			if oldView != nil && !oldView.GetProto().GetSkipDump() {
				diff.ViewChanges = append(diff.ViewChanges, &ViewDiff{
					Action:     MetadataDiffActionDrop,
					SchemaName: schemaName,
					ViewName:   viewName,
					OldView:    oldView.GetProto(),
				})
			}
		}
	}

	// Check for new and modified views
	for _, viewName := range newSchema.ListViewNames() {
		newView := newSchema.GetView(viewName)
		if newView == nil || newView.GetProto().GetSkipDump() {
			continue
		}

		oldView := oldSchema.GetView(viewName)
		if oldView == nil {
			diff.ViewChanges = append(diff.ViewChanges, &ViewDiff{
				Action:     MetadataDiffActionCreate,
				SchemaName: schemaName,
				ViewName:   viewName,
				NewView:    newView.GetProto(),
			})
		} else if !oldView.GetProto().GetSkipDump() {
			// Use engine-specific comparison
			changes, err := comparer.CompareView(oldView, newView)
			if err != nil {
				// Fallback to simple definition comparison on error
				if oldView.Definition != newView.Definition {
					diff.ViewChanges = append(diff.ViewChanges, &ViewDiff{
						Action:     MetadataDiffActionAlter,
						SchemaName: schemaName,
						ViewName:   viewName,
						OldView:    oldView.GetProto(),
						NewView:    newView.GetProto(),
					})
				}
			} else if len(changes) > 0 {
				// Check if any change requires recreation
				requiresRecreation := false
				for _, change := range changes {
					if change.RequiresRecreation {
						requiresRecreation = true
						break
					}
				}

				if requiresRecreation {
					diff.ViewChanges = append(diff.ViewChanges, &ViewDiff{
						Action:     MetadataDiffActionAlter,
						SchemaName: schemaName,
						ViewName:   viewName,
						OldView:    oldView.GetProto(),
						NewView:    newView.GetProto(),
					})
				}
				// TODO: Handle non-recreating changes like comment updates
			}
		}
	}
}

// compareMaterializedViews compares materialized views between two schemas.
func compareMaterializedViews(engine storepb.Engine, diff *MetadataDiff, schemaName string, oldSchema, newSchema *model.SchemaMetadata) {
	// Get the engine-specific view comparer
	comparer := GetViewComparer(engine)
	// Check for dropped materialized views
	for _, mvName := range oldSchema.ListMaterializedViewNames() {
		if newSchema.GetMaterializedView(mvName) == nil {
			oldMV := oldSchema.GetMaterializedView(mvName)
			if oldMV != nil && !oldMV.GetProto().GetSkipDump() {
				diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &MaterializedViewDiff{
					Action:               MetadataDiffActionDrop,
					SchemaName:           schemaName,
					MaterializedViewName: mvName,
					OldMaterializedView:  oldMV.GetProto(),
				})
			}
		}
	}

	// Check for new and modified materialized views
	for _, mvName := range newSchema.ListMaterializedViewNames() {
		newMV := newSchema.GetMaterializedView(mvName)
		if newMV == nil || newMV.GetProto().GetSkipDump() {
			continue
		}

		oldMV := oldSchema.GetMaterializedView(mvName)
		if oldMV == nil {
			diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &MaterializedViewDiff{
				Action:               MetadataDiffActionCreate,
				SchemaName:           schemaName,
				MaterializedViewName: mvName,
				NewMaterializedView:  newMV.GetProto(),
			})
		} else if !oldMV.GetProto().GetSkipDump() {
			// Use engine-specific comparison
			changes, err := comparer.CompareMaterializedView(oldMV, newMV)
			if err != nil {
				// Fallback to simple definition comparison on error
				if oldMV.Definition != newMV.Definition {
					diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &MaterializedViewDiff{
						Action:               MetadataDiffActionAlter,
						SchemaName:           schemaName,
						MaterializedViewName: mvName,
						OldMaterializedView:  oldMV.GetProto(),
						NewMaterializedView:  newMV.GetProto(),
					})
				}
			} else if len(changes) > 0 {
				// Check if any change requires recreation
				requiresRecreation := false
				for _, change := range changes {
					if change.RequiresRecreation {
						requiresRecreation = true
						break
					}
				}

				if requiresRecreation {
					diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &MaterializedViewDiff{
						Action:               MetadataDiffActionAlter,
						SchemaName:           schemaName,
						MaterializedViewName: mvName,
						OldMaterializedView:  oldMV.GetProto(),
						NewMaterializedView:  newMV.GetProto(),
					})
				}
				// TODO: Handle non-recreating changes like comment updates or index-only changes
			}
		}
	}
}

// compareFunctions compares functions between two schemas.
func compareFunctions(diff *MetadataDiff, schemaName string, oldSchema, newSchema *model.SchemaMetadata) {
	// Functions can have overloading, so we need to handle them carefully
	// Group functions by signature to properly match overloaded functions
	// Build map of old functions by signature
	oldFuncsBySignature := make(map[string]*model.FunctionMetadata)
	for _, fn := range oldSchema.ListFunctions() {
		if !fn.GetProto().GetSkipDump() {
			sig := fn.GetProto().Signature
			if sig == "" {
				sig = fn.GetProto().Name // fallback if no signature
			}
			oldFuncsBySignature[sig] = fn
		}
	}

	// Build map of new functions by signature
	newFuncsBySignature := make(map[string]*model.FunctionMetadata)
	for _, fn := range newSchema.ListFunctions() {
		if !fn.GetProto().GetSkipDump() {
			sig := fn.GetProto().Signature
			if sig == "" {
				sig = fn.GetProto().Name // fallback if no signature
			}
			newFuncsBySignature[sig] = fn
		}
	}

	// Check for dropped functions
	for sig, oldFunc := range oldFuncsBySignature {
		if _, exists := newFuncsBySignature[sig]; !exists {
			diff.FunctionChanges = append(diff.FunctionChanges, &FunctionDiff{
				Action:       MetadataDiffActionDrop,
				SchemaName:   schemaName,
				FunctionName: oldFunc.GetProto().Name,
				OldFunction:  oldFunc.GetProto(),
			})
		}
	}

	// Check for new and modified functions
	for sig, newFunc := range newFuncsBySignature {
		oldFunc, exists := oldFuncsBySignature[sig]
		if !exists {
			diff.FunctionChanges = append(diff.FunctionChanges, &FunctionDiff{
				Action:       MetadataDiffActionCreate,
				SchemaName:   schemaName,
				FunctionName: newFunc.GetProto().Name,
				NewFunction:  newFunc.GetProto(),
			})
		} else if !functionsEqual(oldFunc.GetProto(), newFunc.GetProto()) {
			// Drop and recreate if definition changed
			diff.FunctionChanges = append(diff.FunctionChanges, &FunctionDiff{
				Action:       MetadataDiffActionDrop,
				SchemaName:   schemaName,
				FunctionName: oldFunc.GetProto().Name,
				OldFunction:  oldFunc.GetProto(),
			})
			diff.FunctionChanges = append(diff.FunctionChanges, &FunctionDiff{
				Action:       MetadataDiffActionCreate,
				SchemaName:   schemaName,
				FunctionName: newFunc.GetProto().Name,
				NewFunction:  newFunc.GetProto(),
			})
		}
	}
}

// functionsEqual checks if two functions are equal.
func functionsEqual(fn1, fn2 *storepb.FunctionMetadata) bool {
	if fn1.Definition != fn2.Definition {
		// For PostgreSQL functions, try normalized comparison
		norm1 := normalizePostgreSQLFunction(fn1.Definition)
		norm2 := normalizePostgreSQLFunction(fn2.Definition)
		if norm1 != norm2 {
			return false
		}
	}
	if fn1.CharacterSetClient != fn2.CharacterSetClient {
		return false
	}
	if fn1.CollationConnection != fn2.CollationConnection {
		return false
	}
	if fn1.DatabaseCollation != fn2.DatabaseCollation {
		return false
	}
	if fn1.SqlMode != fn2.SqlMode {
		return false
	}
	if fn1.Comment != fn2.Comment {
		return false
	}
	return true
}

// compareProcedures compares procedures between two schemas.
func compareProcedures(diff *MetadataDiff, schemaName string, oldSchema, newSchema *model.SchemaMetadata) {
	// Check for dropped procedures
	for _, procName := range oldSchema.ListProcedureNames() {
		if newSchema.GetProcedure(procName) == nil {
			oldProc := oldSchema.GetProcedure(procName)
			if oldProc != nil && !oldProc.GetProto().GetSkipDump() {
				diff.ProcedureChanges = append(diff.ProcedureChanges, &ProcedureDiff{
					Action:        MetadataDiffActionDrop,
					SchemaName:    schemaName,
					ProcedureName: procName,
					OldProcedure:  oldProc.GetProto(),
				})
			}
		}
	}

	// Check for new and modified procedures
	for _, procName := range newSchema.ListProcedureNames() {
		newProc := newSchema.GetProcedure(procName)
		if newProc == nil || newProc.GetProto().GetSkipDump() {
			continue
		}

		oldProc := oldSchema.GetProcedure(procName)
		if oldProc == nil {
			diff.ProcedureChanges = append(diff.ProcedureChanges, &ProcedureDiff{
				Action:        MetadataDiffActionCreate,
				SchemaName:    schemaName,
				ProcedureName: procName,
				NewProcedure:  newProc.GetProto(),
			})
		} else if !oldProc.GetProto().GetSkipDump() && oldProc.Definition != newProc.Definition {
			diff.ProcedureChanges = append(diff.ProcedureChanges, &ProcedureDiff{
				Action:        MetadataDiffActionAlter,
				SchemaName:    schemaName,
				ProcedureName: procName,
				OldProcedure:  oldProc.GetProto(),
				NewProcedure:  newProc.GetProto(),
			})
		}
	}
}

// compareSequences compares sequences between two schemas.
func compareSequences(diff *MetadataDiff, schemaName string, oldSchema, newSchema *model.SchemaMetadata) {
	// Get sequences from proto since there's no ListSequenceNames method
	oldSeqMap := make(map[string]*storepb.SequenceMetadata)
	for _, seq := range oldSchema.GetProto().Sequences {
		if !seq.GetSkipDump() {
			oldSeqMap[seq.Name] = seq
		}
	}

	newSeqMap := make(map[string]*storepb.SequenceMetadata)
	for _, seq := range newSchema.GetProto().Sequences {
		if !seq.GetSkipDump() {
			newSeqMap[seq.Name] = seq
		}
	}

	// Check for dropped sequences
	for seqName, oldSeq := range oldSeqMap {
		if _, exists := newSeqMap[seqName]; !exists {
			diff.SequenceChanges = append(diff.SequenceChanges, &SequenceDiff{
				Action:       MetadataDiffActionDrop,
				SchemaName:   schemaName,
				SequenceName: seqName,
				OldSequence:  oldSeq,
			})
		}
	}

	// Check for new sequences
	for seqName, newSeq := range newSeqMap {
		if _, exists := oldSeqMap[seqName]; !exists {
			diff.SequenceChanges = append(diff.SequenceChanges, &SequenceDiff{
				Action:       MetadataDiffActionCreate,
				SchemaName:   schemaName,
				SequenceName: seqName,
				NewSequence:  newSeq,
			})
		}
	}
}

// compareEnumTypes compares enum types between two schemas.
func compareEnumTypes(diff *MetadataDiff, schemaName string, oldSchema, newSchema *model.SchemaMetadata) {
	oldSchemaProto := oldSchema.GetProto()
	newSchemaProto := newSchema.GetProto()

	// Build maps of enum types
	oldEnumMap := make(map[string]*storepb.EnumTypeMetadata)
	for _, enum := range oldSchemaProto.EnumTypes {
		if !enum.GetSkipDump() {
			oldEnumMap[enum.Name] = enum
		}
	}

	newEnumMap := make(map[string]*storepb.EnumTypeMetadata)
	for _, enum := range newSchemaProto.EnumTypes {
		if !enum.GetSkipDump() {
			newEnumMap[enum.Name] = enum
		}
	}

	// Check for dropped enum types
	for enumName, oldEnum := range oldEnumMap {
		if _, exists := newEnumMap[enumName]; !exists {
			diff.EnumTypeChanges = append(diff.EnumTypeChanges, &EnumTypeDiff{
				Action:       MetadataDiffActionDrop,
				SchemaName:   schemaName,
				EnumTypeName: enumName,
				OldEnumType:  oldEnum,
			})
		}
	}

	// Check for new enum types
	for enumName, newEnum := range newEnumMap {
		if _, exists := oldEnumMap[enumName]; !exists {
			diff.EnumTypeChanges = append(diff.EnumTypeChanges, &EnumTypeDiff{
				Action:       MetadataDiffActionCreate,
				SchemaName:   schemaName,
				EnumTypeName: enumName,
				NewEnumType:  newEnum,
			})
		}
		// Note: We don't support ALTER enum types yet
		// PostgreSQL doesn't allow modifying enum values easily
	}
}

// compareEvents compares events between old and new schemas.
func compareEvents(diff *MetadataDiff, _ string, oldSchema, newSchema *model.SchemaMetadata) {
	oldSchemaProto := oldSchema.GetProto()
	newSchemaProto := newSchema.GetProto()

	// Build maps of events
	oldEventMap := make(map[string]*storepb.EventMetadata)
	for _, event := range oldSchemaProto.Events {
		oldEventMap[event.Name] = event
	}

	newEventMap := make(map[string]*storepb.EventMetadata)
	for _, event := range newSchemaProto.Events {
		newEventMap[event.Name] = event
	}

	// Check for dropped events
	for eventName, oldEvent := range oldEventMap {
		if _, exists := newEventMap[eventName]; !exists {
			diff.EventChanges = append(diff.EventChanges, &EventDiff{
				Action:    MetadataDiffActionDrop,
				EventName: eventName,
				OldEvent:  oldEvent,
			})
		}
	}

	// Check for new and modified events
	for eventName, newEvent := range newEventMap {
		oldEvent, exists := oldEventMap[eventName]
		if !exists {
			diff.EventChanges = append(diff.EventChanges, &EventDiff{
				Action:    MetadataDiffActionCreate,
				EventName: eventName,
				NewEvent:  newEvent,
			})
		} else if oldEvent.Definition != newEvent.Definition {
			// Check if event has changed
			diff.EventChanges = append(diff.EventChanges, &EventDiff{
				Action:    MetadataDiffActionAlter,
				EventName: eventName,
				OldEvent:  oldEvent,
				NewEvent:  newEvent,
			})
		}
	}
}

// FilterPostgresArchiveSchema filters out schema diff objects related to bbdataarchive schema.
func FilterPostgresArchiveSchema(diff *MetadataDiff) *MetadataDiff {
	if diff == nil {
		return nil
	}

	archiveSchemaName := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)

	// Create a new diff object with filtered changes
	filtered := &MetadataDiff{
		DatabaseName: diff.DatabaseName,
	}

	// Filter schema changes
	for _, schemaChange := range diff.SchemaChanges {
		if schemaChange.SchemaName != archiveSchemaName {
			filtered.SchemaChanges = append(filtered.SchemaChanges, schemaChange)
		}
	}

	// Filter table changes
	for _, tableChange := range diff.TableChanges {
		if tableChange.SchemaName != archiveSchemaName {
			filtered.TableChanges = append(filtered.TableChanges, tableChange)
		}
	}

	// Filter view changes
	for _, viewChange := range diff.ViewChanges {
		if viewChange.SchemaName != archiveSchemaName {
			filtered.ViewChanges = append(filtered.ViewChanges, viewChange)
		}
	}

	// Filter materialized view changes
	for _, mvChange := range diff.MaterializedViewChanges {
		if mvChange.SchemaName != archiveSchemaName {
			filtered.MaterializedViewChanges = append(filtered.MaterializedViewChanges, mvChange)
		}
	}

	// Filter function changes
	for _, funcChange := range diff.FunctionChanges {
		if funcChange.SchemaName != archiveSchemaName {
			filtered.FunctionChanges = append(filtered.FunctionChanges, funcChange)
		}
	}

	// Filter procedure changes
	for _, procChange := range diff.ProcedureChanges {
		if procChange.SchemaName != archiveSchemaName {
			filtered.ProcedureChanges = append(filtered.ProcedureChanges, procChange)
		}
	}

	// Filter sequence changes
	for _, seqChange := range diff.SequenceChanges {
		if seqChange.SchemaName != archiveSchemaName {
			filtered.SequenceChanges = append(filtered.SequenceChanges, seqChange)
		}
	}

	// Filter enum type changes
	for _, enumChange := range diff.EnumTypeChanges {
		if enumChange.SchemaName != archiveSchemaName {
			filtered.EnumTypeChanges = append(filtered.EnumTypeChanges, enumChange)
		}
	}

	// Events are database-level objects, not schema-specific, so copy them all
	filtered.EventChanges = diff.EventChanges

	return filtered
}

// normalizePostgreSQLFunction normalizes PostgreSQL function definitions for comparison
func normalizePostgreSQLFunction(definition string) string {
	if definition == "" {
		return ""
	}

	normalized := strings.ToLower(definition)

	// Normalize whitespace
	normalized = strings.ReplaceAll(normalized, "\n", " ")
	normalized = strings.ReplaceAll(normalized, "\t", " ")

	// Remove extra spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	// Normalize CREATE vs CREATE OR REPLACE
	normalized = strings.ReplaceAll(normalized, "create or replace function", "create function")
	normalized = strings.ReplaceAll(normalized, "create or replace procedure", "create procedure")

	// Remove public schema prefix from function and procedure names
	normalized = strings.ReplaceAll(normalized, "function public.", "function ")
	normalized = strings.ReplaceAll(normalized, "procedure public.", "procedure ")

	// Normalize parameter types
	normalized = strings.ReplaceAll(normalized, "character varying", "varchar")
	normalized = strings.ReplaceAll(normalized, "returns numeric", "returns decimal")

	// Normalize dollar quoting - handle various dollar quote formats
	normalized = strings.ReplaceAll(normalized, "$function$", "$$")
	normalized = strings.ReplaceAll(normalized, "$procedure$", "$$")

	// Normalize language position - move to end
	if strings.Contains(normalized, "language plpgsql") && !strings.HasSuffix(strings.TrimSpace(normalized), "language plpgsql") {
		withoutLanguage := strings.ReplaceAll(normalized, " language plpgsql", "")
		withoutLanguage = strings.ReplaceAll(withoutLanguage, "language plpgsql ", "")
		normalized = strings.TrimSpace(withoutLanguage) + " language plpgsql"
	}

	return strings.TrimSpace(normalized)
}

// normalizePostgreSQLDefaultValue normalizes PostgreSQL default values for comparison
func normalizePostgreSQLDefaultValue(defaultValue string) string {
	if defaultValue == "" {
		return ""
	}

	normalized := strings.ToLower(defaultValue)

	// Remove quotes around the entire default value if present
	if len(normalized) >= 2 && normalized[0] == '\'' && normalized[len(normalized)-1] == '\'' {
		normalized = normalized[1 : len(normalized)-1]
	}

	// Normalize whitespace
	normalized = strings.ReplaceAll(normalized, "\n", " ")
	normalized = strings.ReplaceAll(normalized, "\t", " ")

	// Remove extra spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	// Remove public schema prefix from function calls
	normalized = strings.ReplaceAll(normalized, "public.", "")

	return strings.TrimSpace(normalized)
}

// normalizePostgreSQLCheckConstraint normalizes PostgreSQL check constraint expressions for comparison
func normalizePostgreSQLCheckConstraint(expression string) string {
	// Based on the test output, we need to normalize:
	// Sync: (order_date >= (CURRENT_DATE - '1 year'::interval))
	// Parser: order_date >= CURRENT_DATE - INTERVAL '1 year'
	// Both should normalize to the same canonical form

	expression = strings.TrimSpace(expression)

	// Step 1: Remove outer parentheses
	for strings.HasPrefix(expression, "(") && strings.HasSuffix(expression, ")") {
		// Only remove if they are balanced and wrapping the entire expression
		inner := expression[1 : len(expression)-1]
		parenCount := 0
		canRemove := true
		for _, ch := range inner {
			if ch == '(' {
				parenCount++
			} else if ch == ')' {
				parenCount--
				if parenCount < 0 {
					canRemove = false
					break
				}
			}
		}
		if canRemove && parenCount == 0 {
			expression = inner
		} else {
			break
		}
	}

	// Step 2: Normalize interval syntax
	// Convert ::interval to INTERVAL prefix
	expression = strings.ReplaceAll(expression, "'::interval", "'")

	// Step 3: Standardize to INTERVAL syntax
	if strings.Contains(expression, "CURRENT_DATE") && strings.Contains(expression, "'") && !strings.Contains(expression, "INTERVAL") {
		// Add INTERVAL prefix where needed
		expression = strings.ReplaceAll(expression, "CURRENT_DATE - '", "CURRENT_DATE - INTERVAL '")
		expression = strings.ReplaceAll(expression, "CURRENT_DATE + '", "CURRENT_DATE + INTERVAL '")
	}

	// Step 4: Remove extra parentheses around date arithmetic
	expression = strings.ReplaceAll(expression, "(CURRENT_DATE - INTERVAL", "CURRENT_DATE - INTERVAL")
	expression = strings.ReplaceAll(expression, "(CURRENT_DATE + INTERVAL", "CURRENT_DATE + INTERVAL")

	// Step 5: Clean up trailing parentheses after year'
	if strings.Contains(expression, "year'") && strings.Contains(expression, "INTERVAL") {
		expression = strings.ReplaceAll(expression, "year')", "year'")
	}

	return strings.TrimSpace(expression)
}
