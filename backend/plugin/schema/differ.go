package schema

import (
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	// Check if both have no default
	hasDefault1 := col1.DefaultNull || col1.Default != "" || col1.DefaultExpression != ""
	hasDefault2 := col2.DefaultNull || col2.Default != "" || col2.DefaultExpression != ""

	if !hasDefault1 && !hasDefault2 {
		return true
	}
	if hasDefault1 != hasDefault2 {
		return false
	}

	// Check default null
	if col1.DefaultNull != col2.DefaultNull {
		return false
	}

	// Check default string value
	if col1.Default != col2.Default {
		return false
	}

	// Check default expression
	if col1.DefaultExpression != col2.DefaultExpression {
		return false
	}

	return true
}

// generationMetadataEqual compares two generation metadata structs.
func generationMetadataEqual(gen1, gen2 *storepb.GenerationMetadata) bool {
	if gen1 == nil && gen2 == nil {
		return true
	}
	if gen1 == nil || gen2 == nil {
		return false
	}
	return gen1.Type == gen2.Type && gen1.Expression == gen2.Expression
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
		if expr != idx2.Expressions[i] {
			return false
		}
	}
	return true
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
	return check1.Expression == check2.Expression
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
	if part1.Expression != part2.Expression {
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
		return false
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
