package pg

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	pgTypeCharacterVarying  = "character varying"
	pgTypeDoublePrecision   = "double precision"
	viewCommentObjectFormat = "VIEW \"%s\".\"%s\""
)

func pgDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return pgDiffMetadataMigrationForEngine(storepb.Engine_POSTGRES, oldSchema, newSchema)
}

func pgDiffCockroachMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return pgDiffMetadataMigrationForEngine(storepb.Engine_COCKROACHDB, oldSchema, newSchema)
}

func pgDiffMetadataMigrationForEngine(engine storepb.Engine, oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	diff, err := schema.GetDatabaseSchemaDiff(engine, oldSchema, newSchema)
	if err != nil {
		return "", err
	}
	if metadataDiffEmpty(diff) {
		return "", nil
	}
	return pgGenerateMetadataMigration(diff)
}

func metadataDiffEmpty(diff *schema.MetadataDiff) bool {
	return diff == nil ||
		len(diff.SchemaChanges) == 0 &&
			len(diff.TableChanges) == 0 &&
			len(diff.ViewChanges) == 0 &&
			len(diff.MaterializedViewChanges) == 0 &&
			len(diff.FunctionChanges) == 0 &&
			len(diff.ProcedureChanges) == 0 &&
			len(diff.SequenceChanges) == 0 &&
			len(diff.EnumTypeChanges) == 0 &&
			len(diff.ExtensionChanges) == 0 &&
			len(diff.EventTriggerChanges) == 0 &&
			len(diff.EventChanges) == 0 &&
			len(diff.CommentChanges) == 0
}

func pgGenerateMetadataMigration(diff *schema.MetadataDiff) (string, error) {
	if err := validateSupportedMetadataDiff(diff); err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := writeDropPhase(&buf, diff); err != nil {
		return "", err
	}
	if err := writeCreateOrAlterPhase(&buf, diff); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writeDropPhase(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, eventTriggerDiff := range diff.EventTriggerChanges {
		if eventTriggerDiff.Action == schema.MetadataDiffActionDrop || eventTriggerDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeEventTriggerDiff(out, eventTriggerDiff); err != nil {
				return err
			}
		}
	}

	if err := writeDropTableTriggers(out, diff); err != nil {
		return err
	}
	if err := writeDropAlterTableForeignKeys(out, diff); err != nil {
		return err
	}
	if err := writeDropTableForeignKeysForTableDrops(out, diff); err != nil {
		return err
	}
	if err := writeDropDependentObjects(out, diff); err != nil {
		return err
	}
	if err := writeDropAlterTableObjects(out, diff); err != nil {
		return err
	}

	for _, sequenceDiff := range diff.SequenceChanges {
		if sequenceDiff.Action == schema.MetadataDiffActionDrop {
			if isSequenceOwnedByDroppedTable(diff, sequenceDiff.SchemaName, sequenceDiff.OldSequence) {
				continue
			}
			if err := writeSequenceDiff(out, sequenceDiff); err != nil {
				return err
			}
		}
	}
	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeEnumTypeDiff(out, enumDiff); err != nil {
				return err
			}
		}
	}
	for _, extensionDiff := range diff.ExtensionChanges {
		if extensionDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeExtensionDiff(out, extensionDiff); err != nil {
				return err
			}
		}
	}
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeSchemaDiff(out, schemaDiff); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeCreateOrAlterPhase(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeSchemaDiff(out, schemaDiff); err != nil {
				return err
			}
		}
	}
	for _, extensionDiff := range diff.ExtensionChanges {
		if extensionDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeExtensionDiff(out, extensionDiff); err != nil {
				return err
			}
		}
	}
	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionCreate || enumDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeEnumTypeDiff(out, enumDiff); err != nil {
				return err
			}
		}
	}
	for _, sequenceDiff := range diff.SequenceChanges {
		if sequenceDiff.Action == schema.MetadataDiffActionCreate || sequenceDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeSequenceDiff(out, sequenceDiff); err != nil {
				return err
			}
		}
	}

	if err := writeCreateTables(out, diff); err != nil {
		return err
	}
	if err := writeAlterTables(out, diff); err != nil {
		return err
	}
	if err := writeCreateSequenceOwnership(out, diff); err != nil {
		return err
	}
	if err := writeCreateRoutinesViewsAndMaterializedViews(out, diff); err != nil {
		return err
	}
	if err := writeCreateTableForeignKeys(out, diff); err != nil {
		return err
	}
	if err := writeCreateTableTriggers(out, diff); err != nil {
		return err
	}
	if err := writeCommentChanges(out, diff); err != nil {
		return err
	}
	for _, eventTriggerDiff := range diff.EventTriggerChanges {
		if eventTriggerDiff.Action == schema.MetadataDiffActionCreate || eventTriggerDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeEventTriggerDiff(out, eventTriggerDiff); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSupportedMetadataDiff(diff *schema.MetadataDiff) error {
	if len(diff.EventChanges) > 0 {
		return errors.New("unsupported PostgreSQL metadata diff: event")
	}
	return nil
}

type metadataObjectMaps struct {
	tables            map[string]*schema.TableDiff
	views             map[string]*schema.ViewDiff
	materializedViews map[string]*schema.MaterializedViewDiff
	functions         map[string]*schema.FunctionDiff
	procedures        map[string]*schema.ProcedureDiff
}

func newMetadataObjectMaps() *metadataObjectMaps {
	return &metadataObjectMaps{
		tables:            make(map[string]*schema.TableDiff),
		views:             make(map[string]*schema.ViewDiff),
		materializedViews: make(map[string]*schema.MaterializedViewDiff),
		functions:         make(map[string]*schema.FunctionDiff),
		procedures:        make(map[string]*schema.ProcedureDiff),
	}
}

func getMigrationObjectID(schemaName, objectName string) string {
	return fmt.Sprintf("%s.%s", schemaName, objectName)
}

func buildDropObjectMaps(diff *schema.MetadataDiff) *metadataObjectMaps {
	maps := newMetadataObjectMaps()
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop {
			maps.tables[getMigrationObjectID(tableDiff.SchemaName, tableDiff.TableName)] = tableDiff
		}
	}
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop || alteredViewDependsOnAlteredTable(viewDiff, diff) {
			maps.views[getMigrationObjectID(viewDiff.SchemaName, viewDiff.ViewName)] = viewDiff
		}
	}
	for _, materializedViewDiff := range diff.MaterializedViewChanges {
		if materializedViewDiff.Action == schema.MetadataDiffActionDrop || materializedViewDiff.Action == schema.MetadataDiffActionAlter {
			maps.materializedViews[getMigrationObjectID(materializedViewDiff.SchemaName, materializedViewDiff.MaterializedViewName)] = materializedViewDiff
		}
	}
	for _, functionDiff := range diff.FunctionChanges {
		if functionDiff.Action == schema.MetadataDiffActionDrop {
			maps.functions[getMigrationObjectID(functionDiff.SchemaName, functionDiff.FunctionName)] = functionDiff
		}
	}
	for _, procedureDiff := range diff.ProcedureChanges {
		if procedureDiff.Action == schema.MetadataDiffActionDrop {
			maps.procedures[getMigrationObjectID(procedureDiff.SchemaName, procedureDiff.ProcedureName)] = procedureDiff
		}
	}
	return maps
}

func alteredViewDependsOnAlteredTable(viewDiff *schema.ViewDiff, diff *schema.MetadataDiff) bool {
	if viewDiff.Action != schema.MetadataDiffActionAlter {
		return false
	}
	alteredTables := make(map[string]bool)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			alteredTables[getMigrationObjectID(tableDiff.SchemaName, tableDiff.TableName)] = true
		}
	}
	for _, dep := range viewDiff.OldView.GetDependencyColumns() {
		if alteredTables[getMigrationObjectID(dep.GetSchema(), dep.GetTable())] {
			return true
		}
	}
	return false
}

func buildCreateObjectMaps(diff *schema.MetadataDiff) *metadataObjectMaps {
	maps := newMetadataObjectMaps()
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			maps.tables[getMigrationObjectID(tableDiff.SchemaName, tableDiff.TableName)] = tableDiff
		}
	}
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate || viewDiff.Action == schema.MetadataDiffActionAlter {
			maps.views[getMigrationObjectID(viewDiff.SchemaName, viewDiff.ViewName)] = viewDiff
		}
	}
	for _, materializedViewDiff := range diff.MaterializedViewChanges {
		if materializedViewDiff.Action == schema.MetadataDiffActionCreate || materializedViewDiff.Action == schema.MetadataDiffActionAlter {
			maps.materializedViews[getMigrationObjectID(materializedViewDiff.SchemaName, materializedViewDiff.MaterializedViewName)] = materializedViewDiff
		}
	}
	for _, functionDiff := range diff.FunctionChanges {
		if functionDiff.Action == schema.MetadataDiffActionCreate || functionDiff.Action == schema.MetadataDiffActionAlter {
			maps.functions[getMigrationObjectID(functionDiff.SchemaName, functionDiff.FunctionName)] = functionDiff
		}
	}
	for _, procedureDiff := range diff.ProcedureChanges {
		if procedureDiff.Action == schema.MetadataDiffActionCreate || procedureDiff.Action == schema.MetadataDiffActionAlter {
			maps.procedures[getMigrationObjectID(procedureDiff.SchemaName, procedureDiff.ProcedureName)] = procedureDiff
		}
	}
	return maps
}

func sortedMapKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func writeDropTableTriggers(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, tableDiff := range diff.TableChanges {
		switch tableDiff.Action {
		case schema.MetadataDiffActionDrop:
			for _, trigger := range tableDiff.OldTable.GetTriggers() {
				if err := writeDropTrigger(out, tableDiff.SchemaName, tableDiff.TableName, trigger.Name); err != nil {
					return err
				}
			}
		case schema.MetadataDiffActionAlter:
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionDrop || triggerDiff.Action == schema.MetadataDiffActionAlter {
					if err := writeDropTrigger(out, tableDiff.SchemaName, tableDiff.TableName, triggerDiff.TriggerName); err != nil {
						return err
					}
				}
			}
		default:
		}
	}
	return nil
}

func writeDropAlterTableForeignKeys(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action != schema.MetadataDiffActionAlter {
			continue
		}
		for _, fkDiff := range tableDiff.ForeignKeyChanges {
			if fkDiff.Action == schema.MetadataDiffActionDrop || fkDiff.Action == schema.MetadataDiffActionAlter {
				if err := writeDropForeignKey(out, tableDiff.SchemaName, tableDiff.TableName, fkDiff.OldForeignKey.GetName()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeDropTableForeignKeysForTableDrops(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action != schema.MetadataDiffActionDrop {
			continue
		}
		for _, fk := range tableDiff.OldTable.GetForeignKeys() {
			if err := writeDropForeignKey(out, tableDiff.SchemaName, tableDiff.TableName, fk.GetName()); err != nil {
				return err
			}
		}
	}
	return nil
}

func isSequenceOwnedByDroppedTable(diff *schema.MetadataDiff, schemaName string, sequence *storepb.SequenceMetadata) bool {
	if sequence.GetOwnerTable() == "" || sequence.GetOwnerColumn() == "" {
		return false
	}
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop &&
			tableDiff.SchemaName == schemaName &&
			tableDiff.TableName == sequence.GetOwnerTable() {
			return true
		}
	}
	return false
}

func writeDropDependentObjects(out *strings.Builder, diff *schema.MetadataDiff) error {
	maps := buildDropObjectMaps(diff)
	graph := base.NewGraph()
	for _, id := range sortedMapKeys(maps.tables) {
		graph.AddNode(id)
	}
	for _, id := range sortedMapKeys(maps.views) {
		graph.AddNode(id)
	}
	for _, id := range sortedMapKeys(maps.materializedViews) {
		graph.AddNode(id)
	}
	for _, id := range sortedMapKeys(maps.functions) {
		graph.AddNode(id)
	}
	for _, id := range sortedMapKeys(maps.procedures) {
		graph.AddNode(id)
	}
	for _, tableID := range sortedMapKeys(maps.tables) {
		for _, id := range sortedMapKeys(maps.views) {
			graph.AddEdge(id, tableID)
		}
		for _, id := range sortedMapKeys(maps.materializedViews) {
			graph.AddEdge(id, tableID)
		}
		for _, id := range sortedMapKeys(maps.procedures) {
			graph.AddEdge(id, tableID)
		}
		for _, id := range sortedMapKeys(maps.functions) {
			graph.AddEdge(id, tableID)
		}
	}

	hasNode := func(id string) bool {
		_, ok := graph.NodeMap[id]
		return ok
	}
	for viewID, viewDiff := range maps.views {
		for _, dep := range viewDiff.OldView.GetDependencyColumns() {
			depID := getMigrationObjectID(dep.GetSchema(), dep.GetTable())
			if hasNode(depID) {
				graph.AddEdge(viewID, depID)
			}
		}
	}
	for viewID, viewDiff := range maps.materializedViews {
		for _, dep := range viewDiff.OldMaterializedView.GetDependencyColumns() {
			depID := getMigrationObjectID(dep.GetSchema(), dep.GetTable())
			if hasNode(depID) {
				graph.AddEdge(viewID, depID)
			}
		}
	}
	for functionID, functionDiff := range maps.functions {
		for _, dep := range functionDiff.OldFunction.GetDependencyTables() {
			depID := getMigrationObjectID(dep.GetSchema(), dep.GetTable())
			if hasNode(depID) {
				graph.AddEdge(functionID, depID)
			}
		}
	}
	for tableID, tableDiff := range maps.tables {
		for _, fk := range tableDiff.OldTable.GetForeignKeys() {
			depID := getMigrationObjectID(fk.GetReferencedSchema(), fk.GetReferencedTable())
			if hasNode(depID) {
				graph.AddEdge(tableID, depID)
			}
		}
	}

	orderedIDs, err := graph.TopologicalSort()
	if err != nil {
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.views)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.materializedViews)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.procedures)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.functions)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.tables)...)
	}
	for _, id := range orderedIDs {
		switch {
		case maps.views[id] != nil:
			if err := writeDropView(out, maps.views[id].SchemaName, maps.views[id].ViewName); err != nil {
				return err
			}
		case maps.materializedViews[id] != nil:
			if err := writeDropMaterializedView(out, maps.materializedViews[id].SchemaName, maps.materializedViews[id].MaterializedViewName); err != nil {
				return err
			}
		case maps.procedures[id] != nil:
			if err := writeProcedureDiff(out, maps.procedures[id]); err != nil {
				return err
			}
		case maps.functions[id] != nil:
			if err := writeFunctionDiff(out, maps.functions[id]); err != nil {
				return err
			}
		case maps.tables[id] != nil:
			if err := writeTableDiff(out, maps.tables[id]); err != nil {
				return err
			}
		default:
		}
	}
	return nil
}

func writeDropAlterTableObjects(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action != schema.MetadataDiffActionAlter {
			continue
		}
		for _, partitionDiff := range tableDiff.PartitionChanges {
			if partitionDiff.Action == schema.MetadataDiffActionDrop {
				if err := writeDropPartition(out, tableDiff.SchemaName, partitionDiff.OldPartition); err != nil {
					return err
				}
			}
		}
		for _, primaryKeyDiff := range tableDiff.PrimaryKeyChanges {
			if primaryKeyDiff.Action == schema.MetadataDiffActionDrop || primaryKeyDiff.Action == schema.MetadataDiffActionAlter {
				if err := writeDropTableConstraint(out, tableDiff.SchemaName, tableDiff.TableName, primaryKeyDiff.OldPrimaryKey.GetName()); err != nil {
					return err
				}
			}
		}
		for _, uniqueConstraintDiff := range tableDiff.UniqueConstraintChanges {
			if uniqueConstraintDiff.Action == schema.MetadataDiffActionDrop || uniqueConstraintDiff.Action == schema.MetadataDiffActionAlter {
				if err := writeDropTableConstraint(out, tableDiff.SchemaName, tableDiff.TableName, uniqueConstraintDiff.OldUniqueConstraint.GetName()); err != nil {
					return err
				}
			}
		}
		for _, checkDiff := range tableDiff.CheckConstraintChanges {
			if checkDiff.Action == schema.MetadataDiffActionDrop || checkDiff.Action == schema.MetadataDiffActionAlter {
				if err := writeDropTableConstraint(out, tableDiff.SchemaName, tableDiff.TableName, checkDiff.OldCheckConstraint.GetName()); err != nil {
					return err
				}
			}
		}
		for _, excludeDiff := range tableDiff.ExcludeConstraintChanges {
			if excludeDiff.Action == schema.MetadataDiffActionDrop || excludeDiff.Action == schema.MetadataDiffActionAlter {
				if err := writeDropTableConstraint(out, tableDiff.SchemaName, tableDiff.TableName, excludeDiff.OldExcludeConstraint.GetName()); err != nil {
					return err
				}
			}
		}
		for _, indexDiff := range tableDiff.IndexChanges {
			if indexDiff.Action == schema.MetadataDiffActionDrop || indexDiff.Action == schema.MetadataDiffActionAlter {
				if err := writeDropIndexDiff(out, tableDiff.SchemaName, tableDiff.TableName, indexDiff.OldIndex); err != nil {
					return err
				}
			}
		}
		for _, columnDiff := range tableDiff.ColumnChanges {
			if columnDiff.Action == schema.MetadataDiffActionDrop {
				if err := writeColumnDiff(out, tableDiff.SchemaName, tableDiff.TableName, columnDiff); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeCreateTables(out *strings.Builder, diff *schema.MetadataDiff) error {
	maps := buildCreateObjectMaps(diff)
	graph := base.NewGraph()
	for _, id := range sortedMapKeys(maps.tables) {
		graph.AddNode(id)
	}
	for tableID, tableDiff := range maps.tables {
		for _, fk := range tableDiff.NewTable.GetForeignKeys() {
			depID := getMigrationObjectID(fk.GetReferencedSchema(), fk.GetReferencedTable())
			if _, ok := maps.tables[depID]; ok {
				graph.AddEdge(depID, tableID)
			}
		}
	}

	orderedIDs, err := graph.TopologicalSort()
	if err != nil {
		orderedIDs = sortedMapKeys(maps.tables)
	}
	for _, id := range orderedIDs {
		if err := writeTableDiff(out, maps.tables[id]); err != nil {
			return err
		}
	}
	return nil
}

func writeAlterTables(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeAlterTableDiff(out, tableDiff); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeCreateSequenceOwnership(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, sequenceDiff := range diff.SequenceChanges {
		if sequenceDiff.Action == schema.MetadataDiffActionCreate && sequenceDiff.NewSequence.GetOwnerTable() != "" && sequenceDiff.NewSequence.GetOwnerColumn() != "" {
			if err := writeAlterSequenceOwnedBy(out, sequenceDiff.SchemaName, sequenceDiff.NewSequence); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeCreateRoutinesViewsAndMaterializedViews(out *strings.Builder, diff *schema.MetadataDiff) error {
	maps := buildCreateObjectMaps(diff)
	for _, id := range sortedMapKeys(maps.functions) {
		if err := writeFunctionDiff(out, maps.functions[id]); err != nil {
			return err
		}
	}
	for _, id := range sortedMapKeys(maps.procedures) {
		if err := writeProcedureDiff(out, maps.procedures[id]); err != nil {
			return err
		}
	}

	graph := base.NewGraph()
	for _, id := range sortedMapKeys(maps.views) {
		graph.AddNode(id)
	}
	for _, id := range sortedMapKeys(maps.materializedViews) {
		graph.AddNode(id)
	}
	hasNode := func(id string) bool {
		_, ok := graph.NodeMap[id]
		return ok
	}
	for viewID, viewDiff := range maps.views {
		for _, dep := range viewDiff.NewView.GetDependencyColumns() {
			depID := getMigrationObjectID(dep.GetSchema(), dep.GetTable())
			if hasNode(depID) {
				graph.AddEdge(depID, viewID)
			}
		}
	}
	for viewID, viewDiff := range maps.materializedViews {
		for _, dep := range viewDiff.NewMaterializedView.GetDependencyColumns() {
			depID := getMigrationObjectID(dep.GetSchema(), dep.GetTable())
			if hasNode(depID) {
				graph.AddEdge(depID, viewID)
			}
		}
	}
	orderedIDs, err := graph.TopologicalSort()
	if err != nil {
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.views)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.materializedViews)...)
	}
	for _, id := range orderedIDs {
		switch {
		case maps.views[id] != nil:
			if err := writeViewDiff(out, maps.views[id]); err != nil {
				return err
			}
		case maps.materializedViews[id] != nil:
			if err := writeCreateMaterializedViewDiff(out, maps.materializedViews[id]); err != nil {
				return err
			}
		default:
		}
	}
	return nil
}

func writeCreateTableForeignKeys(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, tableDiff := range diff.TableChanges {
		switch tableDiff.Action {
		case schema.MetadataDiffActionCreate:
			for _, fk := range tableDiff.NewTable.GetForeignKeys() {
				if err := writeForeignKey(out, tableDiff.SchemaName, tableDiff.TableName, fk); err != nil {
					return err
				}
			}
		case schema.MetadataDiffActionAlter:
			for _, fkDiff := range tableDiff.ForeignKeyChanges {
				if fkDiff.Action == schema.MetadataDiffActionCreate || fkDiff.Action == schema.MetadataDiffActionAlter {
					if err := writeForeignKey(out, tableDiff.SchemaName, tableDiff.TableName, fkDiff.NewForeignKey); err != nil {
						return err
					}
				}
			}
		default:
		}
	}
	return nil
}

func writeCreateTableTriggers(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, tableDiff := range diff.TableChanges {
		switch tableDiff.Action {
		case schema.MetadataDiffActionCreate:
			for _, trigger := range tableDiff.NewTable.GetTriggers() {
				if err := writeTrigger(out, tableDiff.SchemaName, tableDiff.TableName, trigger); err != nil {
					return err
				}
			}
		case schema.MetadataDiffActionAlter:
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionCreate || triggerDiff.Action == schema.MetadataDiffActionAlter {
					if err := writeTrigger(out, tableDiff.SchemaName, tableDiff.TableName, triggerDiff.NewTrigger); err != nil {
						return err
					}
				}
			}
		default:
		}
	}
	return nil
}

func writeCommentChanges(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, commentDiff := range diff.CommentChanges {
		if err := writeCommentDiff(out, commentDiff); err != nil {
			return err
		}
	}
	return nil
}

func writeTableDiff(out *strings.Builder, tableDiff *schema.TableDiff) error {
	switch tableDiff.Action {
	case schema.MetadataDiffActionCreate:
		return writeTableForMetadataMigration(out, tableDiff.SchemaName, tableDiff.NewTable)
	case schema.MetadataDiffActionDrop:
		_, err := fmt.Fprintf(out, "DROP TABLE \"%s\".\"%s\";\n\n", tableDiff.SchemaName, tableDiff.TableName)
		return err
	case schema.MetadataDiffActionAlter:
		return writeAlterTableDiff(out, tableDiff)
	default:
		return nil
	}
}

func writeTableForMetadataMigration(out *strings.Builder, schemaName string, table *storepb.TableMetadata) error {
	if err := writeCreateTableForMetadataMigration(out, schemaName, table); err != nil {
		return err
	}
	if len(table.Partitions) > 0 {
		if err := writePartitionClause(out, table.Partitions[0]); err != nil {
			return err
		}
	}
	if _, err := out.WriteString(";\n\n"); err != nil {
		return err
	}
	if table.GetComment() != "" {
		if err := writeTableComment(out, schemaName, table); err != nil {
			return err
		}
	}
	for _, column := range table.GetColumns() {
		if column.GetComment() != "" {
			if err := writeColumnComment(out, schemaName, table.GetName(), column); err != nil {
				return err
			}
		}
	}
	for _, partition := range table.GetPartitions() {
		if err := writePartitionTable(out, schemaName, table.GetColumns(), partition); err != nil {
			return err
		}
	}
	for _, partition := range table.GetPartitions() {
		if err := writeAttachPartition(out, schemaName, table.GetName(), partition); err != nil {
			return err
		}
	}
	for _, index := range table.GetIndexes() {
		if index.GetPrimary() {
			if err := writePrimaryKey(out, schemaName, table.GetName(), index); err != nil {
				return err
			}
		}
	}
	for _, partition := range table.GetPartitions() {
		if err := writePartitionPrimaryKey(out, schemaName, partition); err != nil {
			return err
		}
	}
	for _, index := range table.GetIndexes() {
		if index.GetUnique() && !index.GetPrimary() && index.GetIsConstraint() {
			if err := writeUniqueKey(out, schemaName, table.GetName(), index); err != nil {
				return err
			}
		}
	}
	for _, partition := range table.GetPartitions() {
		if err := writePartitionUniqueKey(out, schemaName, partition); err != nil {
			return err
		}
	}
	for _, index := range table.GetIndexes() {
		if !index.GetPrimary() && !index.GetIsConstraint() {
			if err := writeCreateRegularIndex(out, schemaName, table.GetName(), index, len(table.GetPartitions()) > 0); err != nil {
				return err
			}
		}
	}
	for _, partition := range table.GetPartitions() {
		if err := writePartitionIndexForMetadataMigration(out, schemaName, partition); err != nil {
			return err
		}
	}
	for _, partition := range table.GetPartitions() {
		if err := writeAttachPartitionIndex(out, schemaName, partition); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateTableForMetadataMigration(out *strings.Builder, schemaName string, table *storepb.TableMetadata) error {
	if _, err := fmt.Fprintf(out, "CREATE TABLE \"%s\".\"%s\" (", schemaName, table.GetName()); err != nil {
		return err
	}
	for i, column := range table.GetColumns() {
		if i > 0 {
			if _, err := out.WriteString(","); err != nil {
				return err
			}
		}
		if _, err := out.WriteString("\n    "); err != nil {
			return err
		}
		if err := writeColumnDefinition(out, column); err != nil {
			return err
		}
	}
	for _, check := range table.GetCheckConstraints() {
		if _, err := out.WriteString(",\n    "); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "CONSTRAINT \"%s\" CHECK %s", check.GetName(), check.GetExpression()); err != nil {
			return err
		}
	}
	for _, exclude := range table.GetExcludeConstraints() {
		if _, err := out.WriteString(",\n    "); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "CONSTRAINT \"%s\" %s", exclude.GetName(), exclude.GetExpression()); err != nil {
			return err
		}
	}
	_, err := out.WriteString("\n)")
	return err
}

func writeDropPartition(out *strings.Builder, schemaName string, partition *storepb.TablePartitionMetadata) error {
	if partition == nil {
		return nil
	}
	for _, subpartition := range partition.GetSubpartitions() {
		if err := writeDropPartition(out, schemaName, subpartition); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(out, "DROP TABLE \"%s\".\"%s\";\n\n", schemaName, partition.GetName())
	return err
}

func writeCreatePartitionDiff(out *strings.Builder, schemaName, tableName string, columns []*storepb.ColumnMetadata, partition *storepb.TablePartitionMetadata) error {
	if partition == nil {
		return nil
	}
	if err := writePartitionTable(out, schemaName, columns, partition); err != nil {
		return err
	}
	if err := writeAttachPartition(out, schemaName, tableName, partition); err != nil {
		return err
	}
	if err := writePartitionPrimaryKey(out, schemaName, partition); err != nil {
		return err
	}
	if err := writePartitionUniqueKey(out, schemaName, partition); err != nil {
		return err
	}
	if err := writePartitionIndexForMetadataMigration(out, schemaName, partition); err != nil {
		return err
	}
	return writeAttachPartitionIndex(out, schemaName, partition)
}

func writePartitionIndexForMetadataMigration(out *strings.Builder, schemaName string, partition *storepb.TablePartitionMetadata) error {
	for _, index := range partition.GetIndexes() {
		if !index.GetIsConstraint() && !index.GetPrimary() {
			if err := writeCreateRegularIndex(out, schemaName, partition.GetName(), index, len(partition.GetSubpartitions()) > 0); err != nil {
				return err
			}
		}
	}
	for _, subpartition := range partition.GetSubpartitions() {
		if err := writePartitionIndexForMetadataMigration(out, schemaName, subpartition); err != nil {
			return err
		}
	}
	return nil
}

func writeAlterTableDiff(out *strings.Builder, tableDiff *schema.TableDiff) error {
	for _, columnDiff := range tableDiff.ColumnChanges {
		if columnDiff.Action == schema.MetadataDiffActionDrop {
			continue
		}
		if err := writeColumnDiff(out, tableDiff.SchemaName, tableDiff.TableName, columnDiff); err != nil {
			return err
		}
	}
	for _, partitionDiff := range tableDiff.PartitionChanges {
		if partitionDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeCreatePartitionDiff(out, tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable.GetColumns(), partitionDiff.NewPartition); err != nil {
				return err
			}
		}
	}
	for _, primaryKeyDiff := range tableDiff.PrimaryKeyChanges {
		if primaryKeyDiff.Action == schema.MetadataDiffActionCreate || primaryKeyDiff.Action == schema.MetadataDiffActionAlter {
			if err := writePrimaryKey(out, tableDiff.SchemaName, tableDiff.TableName, primaryKeyDiff.NewPrimaryKey); err != nil {
				return err
			}
		}
	}
	for _, uniqueConstraintDiff := range tableDiff.UniqueConstraintChanges {
		if uniqueConstraintDiff.Action == schema.MetadataDiffActionCreate || uniqueConstraintDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeUniqueKey(out, tableDiff.SchemaName, tableDiff.TableName, uniqueConstraintDiff.NewUniqueConstraint); err != nil {
				return err
			}
		}
	}
	for _, checkDiff := range tableDiff.CheckConstraintChanges {
		if checkDiff.Action == schema.MetadataDiffActionCreate || checkDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeAddCheckConstraint(out, tableDiff.SchemaName, tableDiff.TableName, checkDiff.NewCheckConstraint); err != nil {
				return err
			}
		}
	}
	for _, excludeDiff := range tableDiff.ExcludeConstraintChanges {
		if excludeDiff.Action == schema.MetadataDiffActionCreate || excludeDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeAddExcludeConstraint(out, tableDiff.SchemaName, tableDiff.TableName, excludeDiff.NewExcludeConstraint); err != nil {
				return err
			}
		}
	}
	for _, indexDiff := range tableDiff.IndexChanges {
		if indexDiff.Action == schema.MetadataDiffActionCreate || indexDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeCreateIndexDiff(out, tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex); err != nil {
				return err
			}
		}
	}
	return writeTableCommentDiff(out, tableDiff.SchemaName, tableDiff.TableName, tableDiff.OldTable.GetComment(), tableDiff.NewTable.GetComment())
}

func writeColumnDiff(out *strings.Builder, schemaName, tableName string, columnDiff *schema.ColumnDiff) error {
	switch columnDiff.Action {
	case schema.MetadataDiffActionCreate:
		if _, err := fmt.Fprintf(out, "ALTER TABLE \"%s\".\"%s\" ADD COLUMN ", schemaName, tableName); err != nil {
			return err
		}
		if err := writeColumnDefinition(out, columnDiff.NewColumn); err != nil {
			return err
		}
		if _, err := out.WriteString(";\n\n"); err != nil {
			return err
		}
		if columnDiff.NewColumn.GetComment() != "" {
			return writeColumnComment(out, schemaName, tableName, columnDiff.NewColumn)
		}
		return nil
	case schema.MetadataDiffActionDrop:
		_, err := fmt.Fprintf(out, "ALTER TABLE \"%s\".\"%s\" DROP COLUMN \"%s\";\n\n", schemaName, tableName, columnDiff.OldColumn.Name)
		return err
	case schema.MetadataDiffActionAlter:
		return writeAlterColumnDiff(out, schemaName, tableName, columnDiff)
	default:
		return nil
	}
}

func writeAlterColumnDiff(out *strings.Builder, schemaName, tableName string, columnDiff *schema.ColumnDiff) error {
	oldColumn := columnDiff.OldColumn
	newColumn := columnDiff.NewColumn
	if oldColumn.Type != newColumn.Type {
		if _, err := fmt.Fprintf(out, "ALTER TABLE \"%s\".\"%s\" ALTER COLUMN \"%s\" TYPE %s", schemaName, tableName, newColumn.Name, newColumn.Type); err != nil {
			return err
		}
		if requiresExplicitCasting(oldColumn.Type, newColumn.Type) {
			if _, err := fmt.Fprintf(out, " USING \"%s\"::%s", newColumn.Name, newColumn.Type); err != nil {
				return err
			}
		}
		if _, err := out.WriteString(";\n\n"); err != nil {
			return err
		}
	}
	if oldColumn.Default != newColumn.Default {
		if newColumn.Default == "" {
			if _, err := fmt.Fprintf(out, "ALTER TABLE \"%s\".\"%s\" ALTER COLUMN \"%s\" DROP DEFAULT;\n\n", schemaName, tableName, newColumn.Name); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintf(out, "ALTER TABLE \"%s\".\"%s\" ALTER COLUMN \"%s\" SET DEFAULT %s;\n\n", schemaName, tableName, newColumn.Name, newColumn.Default); err != nil {
			return err
		}
	}
	if oldColumn.Nullable != newColumn.Nullable {
		nullability := "DROP NOT NULL"
		if !newColumn.Nullable {
			nullability = "SET NOT NULL"
		}
		if _, err := fmt.Fprintf(out, "ALTER TABLE \"%s\".\"%s\" ALTER COLUMN \"%s\" %s;\n\n", schemaName, tableName, newColumn.Name, nullability); err != nil {
			return err
		}
	}
	if oldColumn.GetComment() != newColumn.GetComment() {
		if err := writeColumnCommentDiff(out, schemaName, tableName, newColumn.Name, newColumn.GetComment()); err != nil {
			return err
		}
	}
	return nil
}

func requiresExplicitCasting(oldType, newType string) bool {
	oldBaseType := extractBaseType(oldType)
	newBaseType := extractBaseType(newType)
	if oldBaseType == newBaseType {
		return requiresUsingForSameType(oldType, newType)
	}
	incompatibleConversions := map[string][]string{
		"text":                        {"integer", "numeric", "real", pgTypeDoublePrecision, "smallint", "bigint", "boolean"},
		pgTypeCharacterVarying:        {"integer", "numeric", "real", pgTypeDoublePrecision, "smallint", "bigint", "boolean"},
		"character":                   {"integer", "numeric", "real", pgTypeDoublePrecision, "smallint", "bigint", "boolean"},
		"jsonb":                       {"text", pgTypeCharacterVarying, "character"},
		"json":                        {"text", pgTypeCharacterVarying, "character"},
		"integer[]":                   {"text", pgTypeCharacterVarying, "character"},
		"text[]":                      {"text", pgTypeCharacterVarying, "character"},
		"bigint":                      {"smallint", "integer"},
		pgTypeDoublePrecision:         {"real"},
		"timestamp with time zone":    {"date"},
		"timestamp without time zone": {"date"},
		"numeric":                     {"integer", "smallint", "bigint", "real", pgTypeDoublePrecision},
	}
	for _, target := range incompatibleConversions[oldBaseType] {
		if target == newBaseType {
			return true
		}
	}
	return false
}

func extractBaseType(typeName string) string {
	if idx := strings.Index(typeName, "("); idx >= 0 {
		return strings.TrimSpace(typeName[:idx])
	}
	return typeName
}

func requiresUsingForSameType(oldType, newType string) bool {
	oldBaseType := extractBaseType(oldType)
	newBaseType := extractBaseType(newType)
	if oldBaseType != newBaseType {
		return false
	}
	if strings.Contains(oldType, pgTypeCharacterVarying) {
		oldLen := extractLength(oldType)
		newLen := extractLength(newType)
		return oldLen > 0 && newLen > 0 && oldLen > newLen*2
	}
	return false
}

func extractLength(typeName string) int {
	start := strings.Index(typeName, "(")
	end := strings.Index(typeName, ")")
	if start < 0 || end < 0 || end <= start {
		return 0
	}
	length, err := strconv.Atoi(strings.TrimSpace(typeName[start+1 : end]))
	if err != nil {
		return 0
	}
	return length
}

func writeColumnDefinition(out *strings.Builder, column *storepb.ColumnMetadata) error {
	if _, err := fmt.Fprintf(out, "\"%s\" %s", column.Name, column.Type); err != nil {
		return err
	}
	if column.Default != "" {
		if _, err := fmt.Fprintf(out, " DEFAULT %s", column.Default); err != nil {
			return err
		}
	}
	switch column.GetIdentityGeneration() {
	case storepb.ColumnMetadata_ALWAYS:
		if _, err := out.WriteString(" GENERATED ALWAYS AS IDENTITY"); err != nil {
			return err
		}
	case storepb.ColumnMetadata_BY_DEFAULT:
		if _, err := out.WriteString(" GENERATED BY DEFAULT AS IDENTITY"); err != nil {
			return err
		}
	default:
	}
	if generation := column.GetGeneration(); generation != nil && generation.GetExpression() != "" {
		generationType := "STORED"
		if generation.GetType() == storepb.GenerationMetadata_TYPE_VIRTUAL {
			generationType = "VIRTUAL"
		}
		if _, err := fmt.Fprintf(out, " GENERATED ALWAYS AS (%s) %s", generation.GetExpression(), generationType); err != nil {
			return err
		}
	}
	if !column.Nullable {
		if _, err := out.WriteString(" NOT NULL"); err != nil {
			return err
		}
	}
	return nil
}

func writeDropTrigger(out *strings.Builder, schemaName, tableName, triggerName string) error {
	_, err := fmt.Fprintf(out, "DROP TRIGGER \"%s\" ON \"%s\".\"%s\";\n\n", triggerName, schemaName, tableName)
	return err
}

func writeDropForeignKey(out *strings.Builder, schemaName, tableName, fkName string) error {
	return writeDropTableConstraint(out, schemaName, tableName, fkName)
}

func writeDropTableConstraint(out *strings.Builder, schemaName, tableName, constraintName string) error {
	if constraintName == "" {
		return nil
	}
	_, err := fmt.Fprintf(out, "ALTER TABLE \"%s\".\"%s\" DROP CONSTRAINT \"%s\";\n\n", schemaName, tableName, constraintName)
	return err
}

func writeDropIndexDiff(out *strings.Builder, schemaName, tableName string, index *storepb.IndexMetadata) error {
	if index.GetIsConstraint() {
		return writeDropTableConstraint(out, schemaName, tableName, index.GetName())
	}
	if index.GetName() == "" {
		return nil
	}
	_, err := fmt.Fprintf(out, "DROP INDEX \"%s\".\"%s\";\n\n", schemaName, index.GetName())
	return err
}

func writeAddCheckConstraint(out *strings.Builder, schemaName, tableName string, check *storepb.CheckConstraintMetadata) error {
	if check == nil {
		return nil
	}
	if _, err := fmt.Fprintf(out, "ALTER TABLE \"%s\".\"%s\" ADD CONSTRAINT \"%s\" CHECK %s;\n\n", schemaName, tableName, check.GetName(), check.GetExpression()); err != nil {
		return err
	}
	return nil
}

func writeAddExcludeConstraint(out *strings.Builder, schemaName, tableName string, exclude *storepb.ExcludeConstraintMetadata) error {
	if exclude == nil {
		return nil
	}
	if _, err := fmt.Fprintf(out, "ALTER TABLE \"%s\".\"%s\" ADD CONSTRAINT \"%s\" %s;\n\n", schemaName, tableName, exclude.GetName(), exclude.GetExpression()); err != nil {
		return err
	}
	return nil
}

func writeCreateIndexDiff(out *strings.Builder, schemaName, tableName string, index *storepb.IndexMetadata) error {
	if index == nil {
		return nil
	}
	switch {
	case index.GetPrimary():
		return writePrimaryKey(out, schemaName, tableName, index)
	case index.GetUnique() && index.GetIsConstraint():
		return writeUniqueKey(out, schemaName, tableName, index)
	default:
		return writeCreateRegularIndex(out, schemaName, tableName, index, false)
	}
}

func writeCreateRegularIndex(out *strings.Builder, schemaName, tableName string, index *storepb.IndexMetadata, useOnlyClause bool) error {
	if index.GetDefinition() != "" {
		if err := writeDefinitionStatement(out, index.GetDefinition()); err != nil {
			return err
		}
		if index.GetComment() != "" {
			return writeIndexComment(out, schemaName, index)
		}
		return nil
	}
	return writeIndex(out, schemaName, tableName, index, useOnlyClause)
}

func writeTableCommentDiff(out *strings.Builder, schemaName, tableName, oldComment, newComment string) error {
	if oldComment == newComment {
		return nil
	}
	return writeComment(out, fmt.Sprintf("TABLE \"%s\".\"%s\"", schemaName, tableName), newComment)
}

func writeColumnCommentDiff(out *strings.Builder, schemaName, tableName, columnName, comment string) error {
	return writeComment(out, fmt.Sprintf("COLUMN \"%s\".\"%s\".\"%s\"", schemaName, tableName, columnName), comment)
}

func writeCommentDiff(out *strings.Builder, commentDiff *schema.CommentDiff) error {
	switch commentDiff.ObjectType {
	case schema.CommentObjectTypeSchema:
		return writeComment(out, fmt.Sprintf("SCHEMA \"%s\"", commentDiff.ObjectName), commentDiff.NewComment)
	case schema.CommentObjectTypeTable:
		return writeComment(out, fmt.Sprintf("TABLE \"%s\".\"%s\"", commentDiff.SchemaName, commentDiff.ObjectName), commentDiff.NewComment)
	case schema.CommentObjectTypeColumn:
		return writeComment(out, fmt.Sprintf("COLUMN \"%s\".\"%s\".\"%s\"", commentDiff.SchemaName, commentDiff.ObjectName, commentDiff.ColumnName), commentDiff.NewComment)
	case schema.CommentObjectTypeView:
		return writeComment(out, fmt.Sprintf(viewCommentObjectFormat, commentDiff.SchemaName, commentDiff.ObjectName), commentDiff.NewComment)
	case schema.CommentObjectTypeMaterializedView:
		return writeComment(out, fmt.Sprintf("MATERIALIZED VIEW \"%s\".\"%s\"", commentDiff.SchemaName, commentDiff.ObjectName), commentDiff.NewComment)
	case schema.CommentObjectTypeFunction:
		return writeComment(out, fmt.Sprintf("FUNCTION %s", formatQualifiedRoutineSignature(commentDiff.SchemaName, commentDiff.ObjectName)), commentDiff.NewComment)
	case schema.CommentObjectTypeSequence:
		return writeComment(out, fmt.Sprintf("SEQUENCE \"%s\".\"%s\"", commentDiff.SchemaName, commentDiff.ObjectName), commentDiff.NewComment)
	case schema.CommentObjectTypeIndex:
		indexName := commentDiff.IndexName
		if indexName == "" {
			indexName = commentDiff.ObjectName
		}
		return writeComment(out, fmt.Sprintf("INDEX \"%s\".\"%s\"", commentDiff.SchemaName, indexName), commentDiff.NewComment)
	case schema.CommentObjectTypeTrigger:
		return writeComment(out, fmt.Sprintf("TRIGGER \"%s\" ON \"%s\".\"%s\"", commentDiff.ObjectName, commentDiff.SchemaName, commentDiff.TableName), commentDiff.NewComment)
	case schema.CommentObjectTypeType:
		return writeComment(out, fmt.Sprintf("TYPE \"%s\".\"%s\"", commentDiff.SchemaName, commentDiff.ObjectName), commentDiff.NewComment)
	case schema.CommentObjectTypeExtension:
		return writeComment(out, fmt.Sprintf("EXTENSION \"%s\"", commentDiff.ObjectName), commentDiff.NewComment)
	case schema.CommentObjectTypeEventTrigger:
		return writeComment(out, fmt.Sprintf("EVENT TRIGGER \"%s\"", commentDiff.ObjectName), commentDiff.NewComment)
	default:
		return nil
	}
}

func writeComment(out io.Writer, object, comment string) error {
	if comment == "" {
		_, err := fmt.Fprintf(out, "COMMENT ON %s IS NULL;\n\n", object)
		return err
	}
	_, err := fmt.Fprintf(out, "COMMENT ON %s IS '%s';\n\n", object, escapeSingleQuote(comment))
	return err
}

func writeSchemaDiff(out *strings.Builder, schemaDiff *schema.SchemaDiff) error {
	switch schemaDiff.Action {
	case schema.MetadataDiffActionCreate:
		if skipPostgresSchemaDDL(schemaDiff.SchemaName) {
			return nil
		}
		return writeSchema(out, schemaDiff.NewSchema)
	case schema.MetadataDiffActionDrop:
		if skipPostgresSchemaDDL(schemaDiff.SchemaName) {
			return nil
		}
		if err := writeDropSchemaObjects(out, schemaDiff.SchemaName, schemaDiff.OldSchema); err != nil {
			return err
		}
		_, err := fmt.Fprintf(out, "DROP SCHEMA IF EXISTS \"%s\";\n\n", schemaDiff.SchemaName)
		return err
	default:
		return nil
	}
}

func skipPostgresSchemaDDL(schemaName string) bool {
	return schemaName == "pg_catalog" || schemaName == "public"
}

func writeDropSchemaObjects(out *strings.Builder, schemaName string, schemaMeta *storepb.SchemaMetadata) error {
	if schemaMeta == nil {
		return nil
	}
	diff := buildDropSchemaObjectsDiff(schemaName, schemaMeta)
	if err := writeDropTableTriggers(out, diff); err != nil {
		return err
	}
	if err := writeDropTableForeignKeysForTableDrops(out, diff); err != nil {
		return err
	}
	if err := writeDropDependentObjects(out, diff); err != nil {
		return err
	}
	for _, sequenceDiff := range diff.SequenceChanges {
		if isSequenceOwnedByDroppedTable(diff, sequenceDiff.SchemaName, sequenceDiff.OldSequence) {
			continue
		}
		if err := writeSequenceDiff(out, sequenceDiff); err != nil {
			return err
		}
	}
	for _, enumDiff := range diff.EnumTypeChanges {
		if err := writeEnumTypeDiff(out, enumDiff); err != nil {
			return err
		}
	}
	return nil
}

func buildDropSchemaObjectsDiff(schemaName string, schemaMeta *storepb.SchemaMetadata) *schema.MetadataDiff {
	diff := &schema.MetadataDiff{}
	for _, table := range schemaMeta.GetTables() {
		if table.GetSkipDump() {
			continue
		}
		diff.TableChanges = append(diff.TableChanges, &schema.TableDiff{
			Action:     schema.MetadataDiffActionDrop,
			SchemaName: schemaName,
			TableName:  table.GetName(),
			OldTable:   table,
		})
	}
	for _, view := range schemaMeta.GetViews() {
		if view.GetSkipDump() {
			continue
		}
		diff.ViewChanges = append(diff.ViewChanges, &schema.ViewDiff{
			Action:     schema.MetadataDiffActionDrop,
			SchemaName: schemaName,
			ViewName:   view.GetName(),
			OldView:    view,
		})
	}
	for _, materializedView := range schemaMeta.GetMaterializedViews() {
		if materializedView.GetSkipDump() {
			continue
		}
		diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
			Action:               schema.MetadataDiffActionDrop,
			SchemaName:           schemaName,
			MaterializedViewName: materializedView.GetName(),
			OldMaterializedView:  materializedView,
		})
	}
	for _, function := range schemaMeta.GetFunctions() {
		if function.GetSkipDump() {
			continue
		}
		functionName := function.GetSignature()
		if functionName == "" {
			functionName = function.GetName()
		}
		diff.FunctionChanges = append(diff.FunctionChanges, &schema.FunctionDiff{
			Action:       schema.MetadataDiffActionDrop,
			SchemaName:   schemaName,
			FunctionName: functionName,
			OldFunction:  function,
		})
	}
	for _, procedure := range schemaMeta.GetProcedures() {
		if procedure.GetSkipDump() {
			continue
		}
		procedureName := procedure.GetSignature()
		if procedureName == "" {
			procedureName = procedure.GetName()
		}
		diff.ProcedureChanges = append(diff.ProcedureChanges, &schema.ProcedureDiff{
			Action:        schema.MetadataDiffActionDrop,
			SchemaName:    schemaName,
			ProcedureName: procedureName,
			OldProcedure:  procedure,
		})
	}
	for _, sequence := range schemaMeta.GetSequences() {
		if sequence.GetSkipDump() {
			continue
		}
		diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
			Action:       schema.MetadataDiffActionDrop,
			SchemaName:   schemaName,
			SequenceName: sequence.GetName(),
			OldSequence:  sequence,
		})
	}
	for _, enumType := range schemaMeta.GetEnumTypes() {
		if enumType.GetSkipDump() {
			continue
		}
		diff.EnumTypeChanges = append(diff.EnumTypeChanges, &schema.EnumTypeDiff{
			Action:       schema.MetadataDiffActionDrop,
			SchemaName:   schemaName,
			EnumTypeName: enumType.GetName(),
			OldEnumType:  enumType,
		})
	}
	return diff
}

func writeExtensionDiff(out *strings.Builder, extensionDiff *schema.ExtensionDiff) error {
	switch extensionDiff.Action {
	case schema.MetadataDiffActionCreate:
		return writeExtension(out, extensionDiff.NewExtension)
	case schema.MetadataDiffActionDrop:
		_, err := fmt.Fprintf(out, "DROP EXTENSION \"%s\";\n\n", extensionDiff.ExtensionName)
		return err
	default:
		return nil
	}
}

func writeEventTriggerDiff(out *strings.Builder, eventTriggerDiff *schema.EventTriggerDiff) error {
	switch eventTriggerDiff.Action {
	case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
		return writeEventTrigger(out, eventTriggerDiff.NewEventTrigger)
	case schema.MetadataDiffActionDrop:
		_, err := fmt.Fprintf(out, "DROP EVENT TRIGGER \"%s\";\n\n", eventTriggerDiff.EventTriggerName)
		return err
	default:
		return nil
	}
}

func writeSequenceDiff(out *strings.Builder, sequenceDiff *schema.SequenceDiff) error {
	switch sequenceDiff.Action {
	case schema.MetadataDiffActionCreate:
		return writeCreateSequence(out, sequenceDiff.SchemaName, sequenceDiff.NewSequence)
	case schema.MetadataDiffActionDrop:
		_, err := fmt.Fprintf(out, "DROP SEQUENCE \"%s\".\"%s\";\n\n", sequenceDiff.SchemaName, sequenceDiff.SequenceName)
		return err
	case schema.MetadataDiffActionAlter:
		return writeAlterSequence(out, sequenceDiff.SchemaName, sequenceDiff.OldSequence, sequenceDiff.NewSequence)
	default:
		return nil
	}
}

func writeAlterSequence(out *strings.Builder, schemaName string, oldSequence *storepb.SequenceMetadata, sequence *storepb.SequenceMetadata) error {
	if _, err := fmt.Fprintf(out, "ALTER SEQUENCE \"%s\".\"%s\"", schemaName, sequence.Name); err != nil {
		return err
	}
	if sequence.DataType != "" {
		if _, err := fmt.Fprintf(out, "\n    AS %s", sequence.DataType); err != nil {
			return err
		}
	}
	if sequence.Start != "" {
		if _, err := fmt.Fprintf(out, "\n    START WITH %s", sequence.Start); err != nil {
			return err
		}
	}
	if sequence.Increment != "" {
		if _, err := fmt.Fprintf(out, "\n    INCREMENT BY %s", sequence.Increment); err != nil {
			return err
		}
	}
	if sequence.MinValue != "" {
		if _, err := fmt.Fprintf(out, "\n    MINVALUE %s", sequence.MinValue); err != nil {
			return err
		}
	}
	if sequence.MaxValue != "" {
		if _, err := fmt.Fprintf(out, "\n    MAXVALUE %s", sequence.MaxValue); err != nil {
			return err
		}
	}
	if sequence.CacheSize != "" {
		if _, err := fmt.Fprintf(out, "\n    CACHE %s", sequence.CacheSize); err != nil {
			return err
		}
	}
	if sequence.Cycle {
		if _, err := out.WriteString("\n    CYCLE"); err != nil {
			return err
		}
	} else if _, err := out.WriteString("\n    NO CYCLE"); err != nil {
		return err
	}
	if _, err := out.WriteString(";\n\n"); err != nil {
		return err
	}
	if oldSequence.GetOwnerTable() != sequence.GetOwnerTable() || oldSequence.GetOwnerColumn() != sequence.GetOwnerColumn() {
		if sequence.GetOwnerColumn() != "" && sequence.GetOwnerTable() != "" {
			if err := writeAlterSequenceOwnedBy(out, schemaName, sequence); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintf(out, "ALTER SEQUENCE \"%s\".\"%s\" OWNED BY NONE;\n\n", schemaName, sequence.Name); err != nil {
			return err
		}
	}
	if oldSequence.GetComment() != sequence.GetComment() {
		return writeComment(out, fmt.Sprintf("SEQUENCE \"%s\".\"%s\"", schemaName, sequence.Name), sequence.GetComment())
	}
	return nil
}

func writeFunctionDiff(out *strings.Builder, functionDiff *schema.FunctionDiff) error {
	switch functionDiff.Action {
	case schema.MetadataDiffActionCreate:
		return writeFunctionStatement(out, functionDiff.SchemaName, functionDiff.NewFunction, false)
	case schema.MetadataDiffActionAlter:
		return writeFunctionStatement(out, functionDiff.SchemaName, functionDiff.NewFunction, true)
	case schema.MetadataDiffActionDrop:
		signature := functionDiff.FunctionName
		if functionDiff.OldFunction.GetSignature() != "" {
			signature = functionDiff.OldFunction.GetSignature()
		}
		return writeDropRoutine(out, "FUNCTION", functionDiff.SchemaName, signature)
	default:
		return nil
	}
}

func writeFunctionStatement(out *strings.Builder, schemaName string, function *storepb.FunctionMetadata, replace bool) error {
	definition := function.GetDefinition()
	if replace {
		definition = convertCreateFunctionToCreateOrReplace(definition)
	}
	if err := writeDefinitionStatement(out, definition); err != nil {
		return err
	}
	if function.GetComment() != "" {
		return writeFunctionComment(out, schemaName, function)
	}
	return nil
}

func convertCreateFunctionToCreateOrReplace(definition string) string {
	trimmedLeft := strings.TrimLeft(definition, " \t\r\n")
	prefixLen := len(definition) - len(trimmedLeft)
	upper := strings.ToUpper(trimmedLeft)
	if strings.HasPrefix(upper, "CREATE OR REPLACE FUNCTION") {
		return definition
	}
	if strings.HasPrefix(upper, "CREATE FUNCTION") {
		return definition[:prefixLen] + "CREATE OR REPLACE FUNCTION" + trimmedLeft[len("CREATE FUNCTION"):]
	}
	return definition
}

func writeProcedureDiff(out *strings.Builder, procedureDiff *schema.ProcedureDiff) error {
	switch procedureDiff.Action {
	case schema.MetadataDiffActionCreate:
		return writeProcedure(out, procedureDiff.SchemaName, procedureDiff.NewProcedure)
	case schema.MetadataDiffActionDrop:
		signature := procedureDiff.ProcedureName
		if procedureDiff.OldProcedure.GetSignature() != "" {
			signature = procedureDiff.OldProcedure.GetSignature()
		}
		return writeDropRoutine(out, "PROCEDURE", procedureDiff.SchemaName, signature)
	case schema.MetadataDiffActionAlter:
		signature := procedureDiff.ProcedureName
		if procedureDiff.OldProcedure.GetSignature() != "" {
			signature = procedureDiff.OldProcedure.GetSignature()
		}
		if err := writeDropRoutine(out, "PROCEDURE", procedureDiff.SchemaName, signature); err != nil {
			return err
		}
		return writeProcedure(out, procedureDiff.SchemaName, procedureDiff.NewProcedure)
	default:
		return nil
	}
}

func writeProcedure(out *strings.Builder, schemaName string, procedure *storepb.ProcedureMetadata) error {
	if err := writeDefinitionStatement(out, procedure.Definition); err != nil {
		return err
	}
	if procedure.Comment != "" {
		return writeRoutineComment(out, "PROCEDURE", schemaName, procedure.Signature, procedure.Name, procedure.Comment)
	}
	return nil
}

func writeDefinitionStatement(out *strings.Builder, definition string) error {
	definition = strings.TrimSpace(definition)
	if _, err := out.WriteString(definition); err != nil {
		return err
	}
	if !strings.HasSuffix(definition, ";") {
		if _, err := out.WriteString(";"); err != nil {
			return err
		}
	}
	_, err := out.WriteString("\n\n")
	return err
}

func writeDropRoutine(out *strings.Builder, routineType, schemaName, signature string) error {
	_, err := fmt.Fprintf(out, "DROP %s %s;\n\n", routineType, formatQualifiedRoutineSignature(schemaName, signature))
	return err
}

func formatQualifiedRoutineSignature(schemaName, signature string) string {
	signature = strings.TrimSpace(signature)
	openParen := strings.Index(signature, "(")
	if openParen < 0 {
		return fmt.Sprintf("\"%s\".\"%s\"()", schemaName, strings.Trim(signature, `"`))
	}
	name := strings.TrimSpace(signature[:openParen])
	if lastDot := strings.LastIndex(name, "."); lastDot >= 0 {
		name = name[lastDot+1:]
	}
	name = strings.Trim(name, `"`)
	return fmt.Sprintf("\"%s\".\"%s\"%s", schemaName, name, signature[openParen:])
}

func writeRoutineComment(out *strings.Builder, routineType, schemaName, signature, name, comment string) error {
	if signature == "" {
		signature = name + "()"
	}
	if _, err := fmt.Fprintf(out, "COMMENT ON %s %s IS '%s';\n\n", routineType, formatQualifiedRoutineSignature(schemaName, signature), escapeSingleQuote(comment)); err != nil {
		return err
	}
	return nil
}

func writeEnumTypeDiff(out *strings.Builder, enumDiff *schema.EnumTypeDiff) error {
	switch enumDiff.Action {
	case schema.MetadataDiffActionCreate:
		if err := writeEnum(out, enumDiff.SchemaName, enumDiff.NewEnumType); err != nil {
			return err
		}
		if _, err := out.WriteString(";\n\n"); err != nil {
			return err
		}
		if enumDiff.NewEnumType.Comment != "" {
			return writeEnumComment(out, enumDiff.SchemaName, enumDiff.NewEnumType)
		}
		return nil
	case schema.MetadataDiffActionDrop:
		_, err := fmt.Fprintf(out, "DROP TYPE \"%s\".\"%s\";\n\n", enumDiff.SchemaName, enumDiff.EnumTypeName)
		return err
	case schema.MetadataDiffActionAlter:
		return writeAlterEnumTypeAddValues(out, enumDiff.SchemaName, enumDiff.OldEnumType, enumDiff.NewEnumType)
	default:
		return nil
	}
}

func writeAlterEnumTypeAddValues(out *strings.Builder, schemaName string, oldEnum *storepb.EnumTypeMetadata, newEnum *storepb.EnumTypeMetadata) error {
	oldValueSet := make(map[string]bool)
	for _, value := range oldEnum.GetValues() {
		oldValueSet[value] = true
	}
	newValueSet := make(map[string]bool)
	for _, value := range newEnum.GetValues() {
		newValueSet[value] = true
	}

	var removedValues []string
	for _, value := range oldEnum.GetValues() {
		if !newValueSet[value] {
			removedValues = append(removedValues, value)
		}
	}
	if len(removedValues) > 0 {
		if _, err := fmt.Fprintf(out, "-- WARNING: PostgreSQL does not support removing enum values from \"%s\".\"%s\": ", schemaName, newEnum.GetName()); err != nil {
			return err
		}
		for i, value := range removedValues {
			if i > 0 {
				if _, err := out.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(out, "'%s'", escapeSingleQuote(value)); err != nil {
				return err
			}
		}
		if _, err := out.WriteString("\n\n"); err != nil {
			return err
		}
	}

	for i, newValue := range newEnum.GetValues() {
		if oldValueSet[newValue] {
			continue
		}
		if _, err := fmt.Fprintf(out, "ALTER TYPE \"%s\".\"%s\" ADD VALUE '%s'", schemaName, newEnum.GetName(), escapeSingleQuote(newValue)); err != nil {
			return err
		}
		for j := i + 1; j < len(newEnum.GetValues()); j++ {
			nextValue := newEnum.GetValues()[j]
			if oldValueSet[nextValue] {
				if _, err := fmt.Fprintf(out, " BEFORE '%s'", escapeSingleQuote(nextValue)); err != nil {
					return err
				}
				break
			}
		}
		if _, err := out.WriteString(";\n\n"); err != nil {
			return err
		}
	}
	if oldEnum.GetComment() != newEnum.GetComment() {
		return writeComment(out, fmt.Sprintf("TYPE \"%s\".\"%s\"", schemaName, newEnum.GetName()), newEnum.GetComment())
	}
	return nil
}

func writeViewDiff(out *strings.Builder, viewDiff *schema.ViewDiff) error {
	switch viewDiff.Action {
	case schema.MetadataDiffActionCreate:
		if err := writeViewSDL(out, viewDiff.SchemaName, viewDiff.NewView); err != nil {
			return err
		}
		if _, err := out.WriteString(";\n\n"); err != nil {
			return err
		}
		if viewDiff.NewView.GetComment() != "" {
			return writeComment(out, fmt.Sprintf(viewCommentObjectFormat, viewDiff.SchemaName, viewDiff.ViewName), viewDiff.NewView.GetComment())
		}
		return nil
	case schema.MetadataDiffActionAlter:
		definition := strings.TrimSuffix(strings.TrimSpace(viewDiff.NewView.Definition), ";")
		if _, err := fmt.Fprintf(out, "CREATE OR REPLACE VIEW \"%s\".\"%s\" AS %s", viewDiff.SchemaName, viewDiff.ViewName, definition); err != nil {
			return err
		}
		if _, err := out.WriteString(";\n\n"); err != nil {
			return err
		}
		if viewDiff.NewView.GetComment() != "" {
			return writeComment(out, fmt.Sprintf(viewCommentObjectFormat, viewDiff.SchemaName, viewDiff.ViewName), viewDiff.NewView.GetComment())
		}
		return nil
	case schema.MetadataDiffActionDrop:
		return writeDropView(out, viewDiff.SchemaName, viewDiff.ViewName)
	default:
		return nil
	}
}

func writeDropView(out *strings.Builder, schemaName, viewName string) error {
	_, err := fmt.Fprintf(out, "DROP VIEW \"%s\".\"%s\";\n\n", schemaName, viewName)
	return err
}

func writeCreateMaterializedViewDiff(out *strings.Builder, viewDiff *schema.MaterializedViewDiff) error {
	return writeMaterializedView(out, viewDiff.SchemaName, viewDiff.NewMaterializedView)
}

func writeDropMaterializedView(out *strings.Builder, schemaName, viewName string) error {
	_, err := fmt.Fprintf(out, "DROP MATERIALIZED VIEW \"%s\".\"%s\";\n\n", schemaName, viewName)
	return err
}
