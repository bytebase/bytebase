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
	augmentMetadataDiffForMigration(diff, oldSchema, newSchema)
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
			len(diff.CompositeTypeChanges) == 0 &&
			len(diff.ExtensionChanges) == 0 &&
			len(diff.EventTriggerChanges) == 0 &&
			len(diff.EventChanges) == 0 &&
			len(diff.CommentChanges) == 0
}

type affectedTableColumns struct {
	columns map[string]bool
}

func augmentMetadataDiffForMigration(diff *schema.MetadataDiff, oldSchema, newSchema *model.DatabaseMetadata) {
	if diff == nil || oldSchema == nil || newSchema == nil {
		return
	}
	affectedTables := collectDependencySensitiveTableColumns(diff)
	if len(affectedTables) == 0 {
		return
	}
	addUnchangedDependentViews(diff, oldSchema, newSchema, affectedTables)
	addUnchangedDependentMaterializedViews(diff, oldSchema, newSchema, affectedTables)
}

func collectDependencySensitiveTableColumns(diff *schema.MetadataDiff) map[string]affectedTableColumns {
	affectedTables := make(map[string]affectedTableColumns)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action != schema.MetadataDiffActionAlter {
			continue
		}
		for _, columnDiff := range tableDiff.ColumnChanges {
			columnName := dependentObjectBlockingColumnName(columnDiff)
			if columnName == "" {
				continue
			}
			tableID := getMigrationObjectID(tableDiff.SchemaName, tableDiff.TableName)
			affected := affectedTables[tableID]
			if affected.columns == nil {
				affected.columns = make(map[string]bool)
			}
			affected.columns[columnName] = true
			affectedTables[tableID] = affected
		}
	}
	return affectedTables
}

func dependentObjectBlockingColumnName(columnDiff *schema.ColumnDiff) string {
	switch columnDiff.Action {
	case schema.MetadataDiffActionDrop:
		return columnDiff.OldColumn.GetName()
	case schema.MetadataDiffActionAlter:
		if columnDiff.OldColumn.GetType() != columnDiff.NewColumn.GetType() {
			return columnDiff.OldColumn.GetName()
		}
		return ""
	default:
		return ""
	}
}

func addUnchangedDependentViews(diff *schema.MetadataDiff, oldSchema, newSchema *model.DatabaseMetadata, affectedTables map[string]affectedTableColumns) {
	changedViews := make(map[string]bool)
	for _, viewDiff := range diff.ViewChanges {
		changedViews[getMigrationObjectID(viewDiff.SchemaName, viewDiff.ViewName)] = true
	}
	for _, schemaMeta := range newSchema.GetProto().GetSchemas() {
		oldSchemaMeta := oldSchema.GetSchemaMetadata(schemaMeta.GetName())
		if oldSchemaMeta == nil {
			continue
		}
		for _, newView := range schemaMeta.GetViews() {
			viewID := getMigrationObjectID(schemaMeta.GetName(), newView.GetName())
			if changedViews[viewID] || newView.GetSkipDump() {
				continue
			}
			oldView := oldSchemaMeta.GetView(newView.GetName())
			if oldView == nil || oldView.GetSkipDump() {
				continue
			}
			if !dependencyColumnsReferenceAffectedTable(oldView.GetDependencyColumns(), affectedTables) && !dependencyColumnsReferenceAffectedTable(newView.GetDependencyColumns(), affectedTables) {
				continue
			}
			diff.ViewChanges = append(diff.ViewChanges, &schema.ViewDiff{
				Action:     schema.MetadataDiffActionAlter,
				SchemaName: schemaMeta.GetName(),
				ViewName:   newView.GetName(),
				OldView:    oldView,
				NewView:    newView,
			})
			changedViews[viewID] = true
		}
	}
}

func addUnchangedDependentMaterializedViews(diff *schema.MetadataDiff, oldSchema, newSchema *model.DatabaseMetadata, affectedTables map[string]affectedTableColumns) {
	changedViews := make(map[string]bool)
	for _, viewDiff := range diff.MaterializedViewChanges {
		changedViews[getMigrationObjectID(viewDiff.SchemaName, viewDiff.MaterializedViewName)] = true
	}
	for _, schemaMeta := range newSchema.GetProto().GetSchemas() {
		oldSchemaMeta := oldSchema.GetSchemaMetadata(schemaMeta.GetName())
		if oldSchemaMeta == nil {
			continue
		}
		for _, newView := range schemaMeta.GetMaterializedViews() {
			viewID := getMigrationObjectID(schemaMeta.GetName(), newView.GetName())
			if changedViews[viewID] || newView.GetSkipDump() {
				continue
			}
			oldView := oldSchemaMeta.GetMaterializedView(newView.GetName())
			if oldView == nil || oldView.GetSkipDump() {
				continue
			}
			if !dependencyColumnsReferenceAffectedTable(oldView.GetDependencyColumns(), affectedTables) && !dependencyColumnsReferenceAffectedTable(newView.GetDependencyColumns(), affectedTables) {
				continue
			}
			diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
				Action:               schema.MetadataDiffActionAlter,
				SchemaName:           schemaMeta.GetName(),
				MaterializedViewName: newView.GetName(),
				OldMaterializedView:  oldView,
				NewMaterializedView:  newView,
			})
			changedViews[viewID] = true
		}
	}
}

func dependencyColumnsReferenceAffectedTable(dependencies []*storepb.DependencyColumn, affectedTables map[string]affectedTableColumns) bool {
	for _, dep := range dependencies {
		affected, ok := affectedTables[getMigrationObjectID(dep.GetSchema(), dep.GetTable())]
		if !ok {
			continue
		}
		if dep.GetColumn() == "" || affected.columns[dep.GetColumn()] {
			return true
		}
	}
	return false
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
	if err := writeDeferredDropPhase(&buf, diff); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// postViewCompositeCreates selects created composites whose attributes
// reference a created view or materialized view row type, expanded over
// composite-to-composite references, for emission after the views phase.
func postViewCompositeCreates(diff *schema.MetadataDiff) map[*schema.CompositeTypeDiff]bool {
	createdViews := make(map[string]bool)
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate {
			createdViews[viewDiff.SchemaName+"\x00"+viewDiff.ViewName] = true
		}
	}
	for _, viewDiff := range diff.MaterializedViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate {
			createdViews[viewDiff.SchemaName+"\x00"+viewDiff.MaterializedViewName] = true
		}
	}
	post := make(map[*schema.CompositeTypeDiff]bool)
	if len(createdViews) == 0 {
		return post
	}
	postKeys := make(map[string]bool)
	for changed := true; changed; {
		changed = false
		for _, compositeDiff := range diff.CompositeTypeChanges {
			if compositeDiff.Action != schema.MetadataDiffActionCreate || post[compositeDiff] {
				continue
			}
			for _, attribute := range compositeDiff.NewCompositeType.GetAttributes() {
				depSchema, depName, ok := parseQualifiedTypeIdent(attribute.Type)
				if !ok {
					continue
				}
				depKey := depSchema + "\x00" + depName
				if createdViews[depKey] || postKeys[depKey] {
					post[compositeDiff] = true
					postKeys[compositeDiff.SchemaName+"\x00"+compositeDiff.CompositeTypeName] = true
					changed = true
					break
				}
			}
		}
	}
	return post
}

// postViewCompositeAlters selects altered composites whose added or retyped
// attributes reference a created view or materialized view row type; their
// alter statements run after the views phase.
func postViewCompositeAlters(diff *schema.MetadataDiff) map[*schema.CompositeTypeDiff]bool {
	createdViews := make(map[string]bool)
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate {
			createdViews[viewDiff.SchemaName+"\x00"+viewDiff.ViewName] = true
		}
	}
	for _, viewDiff := range diff.MaterializedViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate {
			createdViews[viewDiff.SchemaName+"\x00"+viewDiff.MaterializedViewName] = true
		}
	}
	post := make(map[*schema.CompositeTypeDiff]bool)
	if len(createdViews) == 0 {
		return post
	}
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action != schema.MetadataDiffActionAlter {
			continue
		}
		oldAttributes := make(map[string]*storepb.CompositeTypeAttribute)
		for _, attribute := range compositeDiff.OldCompositeType.GetAttributes() {
			oldAttributes[attribute.Name] = attribute
		}
		for _, newAttribute := range compositeDiff.NewCompositeType.GetAttributes() {
			oldAttribute := oldAttributes[newAttribute.Name]
			if oldAttribute != nil && oldAttribute.Type == newAttribute.Type {
				continue
			}
			if depSchema, depName, ok := parseQualifiedTypeIdent(newAttribute.Type); ok && createdViews[depSchema+"\x00"+depName] {
				post[compositeDiff] = true
				break
			}
		}
	}
	return post
}

// deferredDropSet resolves which drops must wait until the alter phase has
// released the last reference.
type deferredDropSet struct {
	composites []*schema.CompositeTypeDiff
	enums      map[*schema.EnumTypeDiff]bool
	schemas    map[*schema.SchemaDiff]bool
}

// computeDeferredDrops seeds the set with composites released by a column
// retype and schemas owning such composites, then expands to a fixpoint: a
// dropped schema is also deferred when any pooled deferred composite
// references a type inside it, and every deferred schema contributes its own
// composites to the pool. Finally, dropped enums referenced by any pooled
// composite are deferred.
func computeDeferredDrops(diff *schema.MetadataDiff) *deferredDropSet {
	deferred := &deferredDropSet{
		enums:   make(map[*schema.EnumTypeDiff]bool),
		schemas: make(map[*schema.SchemaDiff]bool),
	}
	var topLevelDrops []*schema.CompositeTypeDiff
	deferredComposites := make(map[*schema.CompositeTypeDiff]bool)
	var pool []*storepb.CompositeTypeMetadata
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action != schema.MetadataDiffActionDrop {
			continue
		}
		topLevelDrops = append(topLevelDrops, compositeDiff)
		if compositeDropReleasedByRetype(diff, compositeDiff) {
			deferredComposites[compositeDiff] = true
			pool = append(pool, compositeDiff.OldCompositeType)
		}
	}

	// Combined fixpoint: pooled composites pull in (a) top-level dropped
	// composites they reference and (b) dropped schemas containing types
	// they reference; every newly deferred schema contributes its own
	// composites to the pool.
	for changed := true; changed; {
		changed = false
		for _, candidate := range topLevelDrops {
			if deferredComposites[candidate] {
				continue
			}
			for _, member := range pool {
				if compositeReferencesType(member, candidate.SchemaName, candidate.CompositeTypeName) {
					deferredComposites[candidate] = true
					pool = append(pool, candidate.OldCompositeType)
					changed = true
					break
				}
			}
		}
		for _, schemaDiff := range diff.SchemaChanges {
			if schemaDiff.Action != schema.MetadataDiffActionDrop || deferred.schemas[schemaDiff] {
				continue
			}
			if !schemaOwnsRetypeReleasedComposite(diff, schemaDiff) && !compositePoolReferencesSchema(pool, schemaDiff.SchemaName) {
				continue
			}
			deferred.schemas[schemaDiff] = true
			for _, compositeType := range schemaDiff.OldSchema.GetCompositeTypes() {
				if !compositeType.GetSkipDump() {
					pool = append(pool, compositeType)
				}
			}
			changed = true
		}
	}
	for _, compositeDiff := range topLevelDrops {
		if deferredComposites[compositeDiff] {
			deferred.composites = append(deferred.composites, compositeDiff)
		}
	}

	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action != schema.MetadataDiffActionDrop {
			continue
		}
		if typeReleasedByCompositeAlter(diff, enumDiff.SchemaName, enumDiff.EnumTypeName) {
			deferred.enums[enumDiff] = true
			continue
		}
		for _, compositeType := range pool {
			if compositeReferencesType(compositeType, enumDiff.SchemaName, enumDiff.EnumTypeName) {
				deferred.enums[enumDiff] = true
				break
			}
		}
	}
	return deferred
}

// writeDeferredDropPhase emits the deferred drops after the alter phase.
// Type drops are globalized across schema boundaries so cross-schema (even
// cyclic) references between deferred schemas cannot produce an invalid
// order: first the deferred schemas' non-type objects, then every deferred
// composite (top-level and schema-owned) in one reverse-dependency pass,
// then every deferred enum — nothing references an enum by that point —
// and finally the bare DROP SCHEMA statements.
func writeDeferredDropPhase(out *strings.Builder, diff *schema.MetadataDiff) error {
	deferred := computeDeferredDrops(diff)

	deferredSchemas := make([]*schema.SchemaDiff, 0, len(deferred.schemas))
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionDrop && deferred.schemas[schemaDiff] {
			deferredSchemas = append(deferredSchemas, schemaDiff)
		}
	}
	slices.SortStableFunc(deferredSchemas, func(a, b *schema.SchemaDiff) int {
		return strings.Compare(a.SchemaName, b.SchemaName)
	})

	for _, schemaDiff := range deferredSchemas {
		if skipPostgresSchemaDDL(schemaDiff.SchemaName) {
			continue
		}
		if err := writeDropSchemaNonTypeObjects(out, schemaDiff.SchemaName, schemaDiff.OldSchema); err != nil {
			return err
		}
	}

	compositeDrops := slices.Clone(deferred.composites)
	for _, schemaDiff := range deferredSchemas {
		if skipPostgresSchemaDDL(schemaDiff.SchemaName) {
			continue
		}
		schemaObjects := buildDropSchemaObjectsDiff(schemaDiff.SchemaName, schemaDiff.OldSchema)
		compositeDrops = append(compositeDrops, schemaObjects.CompositeTypeChanges...)
	}
	if err := writeCompositeTypeDropsInReverseDependencyOrder(out, compositeDrops); err != nil {
		return err
	}

	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionDrop && deferred.enums[enumDiff] {
			if err := writeEnumTypeDiff(out, enumDiff); err != nil {
				return err
			}
		}
	}
	for _, schemaDiff := range deferredSchemas {
		if skipPostgresSchemaDDL(schemaDiff.SchemaName) {
			continue
		}
		schemaObjects := buildDropSchemaObjectsDiff(schemaDiff.SchemaName, schemaDiff.OldSchema)
		for _, enumDiff := range schemaObjects.EnumTypeChanges {
			if err := writeEnumTypeDiff(out, enumDiff); err != nil {
				return err
			}
		}
	}

	for _, schemaDiff := range deferredSchemas {
		if skipPostgresSchemaDDL(schemaDiff.SchemaName) {
			continue
		}
		if _, err := fmt.Fprintf(out, "DROP SCHEMA IF EXISTS \"%s\";\n\n", schemaDiff.SchemaName); err != nil {
			return err
		}
	}
	return nil
}

// compositePoolReferencesSchema reports whether any composite in the pool has
// an attribute whose (always schema-qualified) type lives in the schema.
func compositePoolReferencesSchema(pool []*storepb.CompositeTypeMetadata, schemaName string) bool {
	for _, compositeType := range pool {
		for _, attribute := range compositeType.GetAttributes() {
			if depSchema, _, ok := parseQualifiedTypeIdent(attribute.Type); ok && depSchema == schemaName {
				return true
			}
		}
	}
	return false
}

// schemaOwnsRetypeReleasedComposite reports whether the dropped schema owns a
// composite that a column retype in the alter phase releases.
func schemaOwnsRetypeReleasedComposite(diff *schema.MetadataDiff, schemaDiff *schema.SchemaDiff) bool {
	for _, compositeType := range schemaDiff.OldSchema.GetCompositeTypes() {
		if compositeType.GetSkipDump() {
			continue
		}
		probe := &schema.CompositeTypeDiff{
			SchemaName:        schemaDiff.SchemaName,
			CompositeTypeName: compositeType.GetName(),
		}
		if compositeDropReleasedByRetype(diff, probe) {
			return true
		}
	}
	return false
}

// compositeReferencesType reports whether any attribute of the composite
// references the given type. Attribute types are stored schema-qualified for
// user-defined types; the bare fallback matches by name alone, which may
// defer a drop unnecessarily — a safe direction.
func compositeReferencesType(composite *storepb.CompositeTypeMetadata, typeSchema, typeName string) bool {
	for _, attribute := range composite.GetAttributes() {
		if typeStringReferencesComposite(attribute.Type, typeSchema, typeName) {
			return true
		}
	}
	return false
}

// partitionDroppedCompositeTypes splits composite type drops into those safe
// to run in the drop phase and those that must wait until after the alter
// phase because a column retype releases the last reference to the type.
// immediateCompositeDrops returns the top-level composite drops that run in
// the drop phase — everything not claimed by the deferred set.
func immediateCompositeDrops(diff *schema.MetadataDiff, deferred *deferredDropSet) []*schema.CompositeTypeDiff {
	deferredSet := make(map[*schema.CompositeTypeDiff]bool, len(deferred.composites))
	for _, compositeDiff := range deferred.composites {
		deferredSet[compositeDiff] = true
	}
	var immediate []*schema.CompositeTypeDiff
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action == schema.MetadataDiffActionDrop && !deferredSet[compositeDiff] {
			immediate = append(immediate, compositeDiff)
		}
	}
	return immediate
}

// dropGraphComposites splits the immediate top-level composite drops into
// those ordered inside the dependent-object graph and those held back to the
// post-graph batch because a schema-owned composite (which only drops after
// the graph) references them, directly or transitively. A composite that is
// both held back and a row-type referencer of a dropped table has genuinely
// conflicting constraints; resolving that requires schema-owned objects in
// the same graph.
func dropGraphComposites(diff *schema.MetadataDiff) (map[string]*schema.CompositeTypeDiff, []*schema.CompositeTypeDiff) {
	immediate := immediateCompositeDrops(diff, computeDeferredDrops(diff))
	var pool []*storepb.CompositeTypeMetadata
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action != schema.MetadataDiffActionDrop || skipPostgresSchemaDDL(schemaDiff.SchemaName) {
			continue
		}
		for _, compositeType := range schemaDiff.OldSchema.GetCompositeTypes() {
			if !compositeType.GetSkipDump() {
				pool = append(pool, compositeType)
			}
		}
	}
	held := make(map[*schema.CompositeTypeDiff]bool)
	for changed := true; changed; {
		changed = false
		for _, candidate := range immediate {
			if held[candidate] {
				continue
			}
			for _, member := range pool {
				if compositeReferencesType(member, candidate.SchemaName, candidate.CompositeTypeName) {
					held[candidate] = true
					pool = append(pool, candidate.OldCompositeType)
					changed = true
					break
				}
			}
		}
	}
	inGraph := make(map[string]*schema.CompositeTypeDiff)
	var heldBack []*schema.CompositeTypeDiff
	for _, compositeDiff := range immediate {
		if held[compositeDiff] {
			heldBack = append(heldBack, compositeDiff)
		} else {
			inGraph[getMigrationObjectID(compositeDiff.SchemaName, compositeDiff.CompositeTypeName)] = compositeDiff
		}
	}
	return inGraph, heldBack
}

// typeReleasedByCompositeAlter reports whether an ALTERED composite's old
// attributes reference the given type while its new version does not — the
// release happens only when the create-phase alter runs.
func typeReleasedByCompositeAlter(diff *schema.MetadataDiff, typeSchema, typeName string) bool {
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action != schema.MetadataDiffActionAlter {
			continue
		}
		for _, oldAttribute := range compositeDiff.OldCompositeType.GetAttributes() {
			if typeStringReferencesComposite(oldAttribute.Type, typeSchema, typeName) {
				return true
			}
		}
	}
	return false
}

func compositeDropReleasedByRetype(diff *schema.MetadataDiff, compositeDiff *schema.CompositeTypeDiff) bool {
	if typeReleasedByCompositeAlter(diff, compositeDiff.SchemaName, compositeDiff.CompositeTypeName) {
		return true
	}
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action != schema.MetadataDiffActionAlter {
			continue
		}
		for _, columnDiff := range tableDiff.ColumnChanges {
			var oldType string
			switch columnDiff.Action {
			case schema.MetadataDiffActionAlter:
				// A retype away from the composite releases it in the alter
				// phase.
				oldType = columnDiff.OldColumn.GetType()
				if oldType == columnDiff.NewColumn.GetType() {
					continue
				}
			case schema.MetadataDiffActionDrop:
				// A column drop on a surviving table runs in
				// writeDropAlterTableObjects, after the dependent-object
				// graph — too late for a graph-ordered type drop.
				oldType = columnDiff.OldColumn.GetType()
			default:
				continue
			}
			if typeStringReferencesComposite(oldType, compositeDiff.SchemaName, compositeDiff.CompositeTypeName) {
				return true
			}
		}
	}
	return false
}

// columnTypeCompositeIDs resolves a column type string to candidate
// composite IDs in the given map. Qualified references resolve exactly;
// bare references (including the "_name" array form column sync produces)
// match every same-named composite — over-edging only tightens ordering.
func columnTypeCompositeIDs(columnType string, composites map[string]*schema.CompositeTypeDiff) []string {
	if depSchema, depName, ok := parseQualifiedTypeIdent(columnType); ok {
		id := getMigrationObjectID(depSchema, depName)
		if composites[id] != nil {
			return []string{id}
		}
		return nil
	}
	base := strings.TrimSuffix(strings.TrimSpace(columnType), "[]")
	base = strings.TrimPrefix(base, "_")
	var ids []string
	for id, compositeDiff := range composites {
		if compositeDiff.CompositeTypeName == base {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

// typeStringReferencesComposite matches a column type string against a
// composite type. Column sync renders user-defined column types as
// "schema.name" and array columns as the bare "_name" element type, so both
// forms plus explicit array suffixes are recognized. The bare form matches
// by name alone, which may defer a drop unnecessarily — a safe direction.
func typeStringReferencesComposite(typeStr, schemaName, compositeName string) bool {
	if depSchema, depName, ok := parseQualifiedTypeIdent(typeStr); ok {
		return depSchema == schemaName && depName == compositeName
	}
	base := strings.TrimSuffix(strings.TrimSpace(typeStr), "[]")
	base = strings.TrimPrefix(base, "_")
	return base == compositeName
}

func writeCompositeTypeDropsInReverseDependencyOrder(out *strings.Builder, drops []*schema.CompositeTypeDiff) error {
	droppedCompositeDiffs := make(map[string]*schema.CompositeTypeDiff, len(drops))
	droppedComposites := make([]qualifiedCompositeType, 0, len(drops))
	for _, compositeDiff := range drops {
		droppedCompositeDiffs[compositeDiff.SchemaName+"\x00"+compositeDiff.CompositeTypeName] = compositeDiff
		droppedComposites = append(droppedComposites, qualifiedCompositeType{Schema: compositeDiff.SchemaName, Composite: compositeDiff.OldCompositeType})
	}
	orderedDrops := sortCompositeTypesTopologically(droppedComposites)
	for i := len(orderedDrops) - 1; i >= 0; i-- {
		t := orderedDrops[i]
		if err := writeCompositeTypeDiff(out, droppedCompositeDiffs[t.Schema+"\x00"+t.Composite.Name]); err != nil {
			return err
		}
	}
	return nil
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
	if err := writeDropSequenceOwnershipBeforeTableDrops(out, diff); err != nil {
		return err
	}
	if err := writeReleasingCompositeAlters(out, diff); err != nil {
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
	// Non-deferred dropped schemas participate in the same global type-drop
	// ordering as top-level drops: a schema-owned composite may reference a
	// top-level type (or vice versa), so their non-type objects drop first,
	// then every composite in one reverse-dependency pass, then enums, and
	// the bare DROP SCHEMA statements last.
	deferred := computeDeferredDrops(diff)
	var immediateSchemaDrops []*schema.SchemaDiff
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionDrop && !deferred.schemas[schemaDiff] && !skipPostgresSchemaDDL(schemaDiff.SchemaName) {
			immediateSchemaDrops = append(immediateSchemaDrops, schemaDiff)
		}
	}
	slices.SortStableFunc(immediateSchemaDrops, func(a, b *schema.SchemaDiff) int {
		return strings.Compare(a.SchemaName, b.SchemaName)
	})
	for _, schemaDiff := range immediateSchemaDrops {
		if err := writeDropSchemaNonTypeObjects(out, schemaDiff.SchemaName, schemaDiff.OldSchema); err != nil {
			return err
		}
	}

	// Drop composite types before enum types: composite attributes may
	// reference enums. Dependents drop before the composites they reference.
	// Composites released only by a column retype cannot be dropped yet —
	// the retype runs in the later alter phase — so those drops are deferred
	// to the end of the migration.
	// Top-level immediate composite drops are ordered inside the dependent-
	// object graph (writeDropDependentObjects), except those a schema-owned
	// composite references — they drop here, merged with the schema-owned
	// composites in one reverse-dependency pass. Residual corner: a
	// schema-owned composite referencing a top-level dropped table drops
	// after it — full generality needs the schema-owned objects in the
	// same graph.
	_, remainingDrops := dropGraphComposites(diff)
	for _, schemaDiff := range immediateSchemaDrops {
		schemaObjects := buildDropSchemaObjectsDiff(schemaDiff.SchemaName, schemaDiff.OldSchema)
		remainingDrops = append(remainingDrops, schemaObjects.CompositeTypeChanges...)
	}
	if err := writeCompositeTypeDropsInReverseDependencyOrder(out, remainingDrops); err != nil {
		return err
	}
	// Enums referenced by a deferred composite cannot be dropped until that
	// composite is gone; they are deferred alongside it.
	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionDrop && !deferred.enums[enumDiff] {
			if err := writeEnumTypeDiff(out, enumDiff); err != nil {
				return err
			}
		}
	}
	for _, schemaDiff := range immediateSchemaDrops {
		schemaObjects := buildDropSchemaObjectsDiff(schemaDiff.SchemaName, schemaDiff.OldSchema)
		for _, enumDiff := range schemaObjects.EnumTypeChanges {
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
	for _, schemaDiff := range immediateSchemaDrops {
		if _, err := fmt.Fprintf(out, "DROP SCHEMA IF EXISTS \"%s\";\n\n", schemaDiff.SchemaName); err != nil {
			return err
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
	// Composite alters run after table creation: an added or retyped
	// attribute may reference a row type created in this migration, while
	// table creation itself only references the composite by name and never
	// depends on its attribute set. Statements that release a dropped
	// table's row type were already emitted in the drop phase. Alters that
	// gain a created view's row type wait until after the views phase.
	released := releasingCompositeAttributes(diff)
	postViewAlters := postViewCompositeAlters(diff)
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action == schema.MetadataDiffActionAlter && !postViewAlters[compositeDiff] {
			if err := writeCompositeTypeDiffWithReleased(out, compositeDiff, released); err != nil {
				return err
			}
		}
	}
	if err := writeAlterTables(out, diff); err != nil {
		return err
	}
	if err := writeSequenceOwnershipAfterTables(out, diff); err != nil {
		return err
	}
	if err := writeCreateRoutinesViewsAndMaterializedViews(out, diff); err != nil {
		return err
	}
	// Composites whose attributes use a created view or materialized view
	// row type (directly or through another such composite) are created only
	// now. Residual: a created table consuming such a composite still emits
	// earlier — resolving that needs views in the create graph.
	postView := postViewCompositeCreates(diff)
	var postViewCreates []qualifiedCompositeType
	postViewDiffs := make(map[string]*schema.CompositeTypeDiff)
	for compositeDiff := range postView {
		postViewDiffs[compositeDiff.SchemaName+"\x00"+compositeDiff.CompositeTypeName] = compositeDiff
		postViewCreates = append(postViewCreates, qualifiedCompositeType{Schema: compositeDiff.SchemaName, Composite: compositeDiff.NewCompositeType})
	}
	for _, t := range sortCompositeTypesTopologically(postViewCreates) {
		if err := writeCompositeTypeDiff(out, postViewDiffs[t.Schema+"\x00"+t.Composite.Name]); err != nil {
			return err
		}
	}
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action == schema.MetadataDiffActionAlter && postViewAlters[compositeDiff] {
			if err := writeCompositeTypeDiffWithReleased(out, compositeDiff, released); err != nil {
				return err
			}
		}
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
		if viewDiff.Action == schema.MetadataDiffActionDrop || alteredViewDependsOnTableDDL(viewDiff, diff) {
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

func alteredViewDependsOnTableDDL(viewDiff *schema.ViewDiff, diff *schema.MetadataDiff) bool {
	if viewDiff.Action != schema.MetadataDiffActionAlter {
		return false
	}
	tableDDLTargets := make(map[string]bool)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop || tableDiff.Action == schema.MetadataDiffActionAlter {
			tableDDLTargets[getMigrationObjectID(tableDiff.SchemaName, tableDiff.TableName)] = true
		}
	}
	for _, dep := range viewDiff.OldView.GetDependencyColumns() {
		if tableDDLTargets[getMigrationObjectID(dep.GetSchema(), dep.GetTable())] {
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
	// Top-level immediate composite drops participate in the same graph:
	// tables/views drop before composites their columns use, and composites
	// drop before tables whose row types their attributes reference.
	composites, _ := dropGraphComposites(diff)
	graph := base.NewGraph()
	for _, id := range sortedMapKeys(composites) {
		graph.AddNode(id)
	}
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
	// Views, matviews, and routines drop before every composite type, like
	// they do before every table — except when the composite's attributes
	// reference that view's row type, where only the composite -> view edge
	// applies (the blanket edge would create a cycle and break the order).
	compositeRowTypeRefs := make(map[string]map[string]bool)
	for compositeID, compositeDiff := range composites {
		refs := make(map[string]bool)
		for _, attribute := range compositeDiff.OldCompositeType.GetAttributes() {
			if depSchema, depName, ok := parseQualifiedTypeIdent(attribute.Type); ok {
				refs[getMigrationObjectID(depSchema, depName)] = true
			}
		}
		compositeRowTypeRefs[compositeID] = refs
	}
	for _, compositeID := range sortedMapKeys(composites) {
		refs := compositeRowTypeRefs[compositeID]
		for _, id := range sortedMapKeys(maps.views) {
			if !refs[id] {
				graph.AddEdge(id, compositeID)
			}
		}
		for _, id := range sortedMapKeys(maps.materializedViews) {
			if !refs[id] {
				graph.AddEdge(id, compositeID)
			}
		}
		for _, id := range sortedMapKeys(maps.procedures) {
			graph.AddEdge(id, compositeID)
		}
		for _, id := range sortedMapKeys(maps.functions) {
			graph.AddEdge(id, compositeID)
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
		// A table whose column uses a dropped composite drops before it.
		for _, column := range tableDiff.OldTable.GetColumns() {
			for _, depID := range columnTypeCompositeIDs(column.GetType(), composites) {
				graph.AddEdge(tableID, depID)
			}
		}
	}
	for compositeID, compositeDiff := range composites {
		for _, attribute := range compositeDiff.OldCompositeType.GetAttributes() {
			depSchema, depName, ok := parseQualifiedTypeIdent(attribute.Type)
			if !ok {
				continue
			}
			depID := getMigrationObjectID(depSchema, depName)
			// A composite drops before the composite or table row type it
			// references.
			if depID != compositeID && hasNode(depID) {
				graph.AddEdge(compositeID, depID)
			}
		}
	}

	orderedIDs, err := graph.TopologicalSort()
	if err != nil {
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.views)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.materializedViews)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.procedures)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.functions)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(composites)...)
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
		case composites[id] != nil:
			if err := writeCompositeTypeDiff(out, composites[id]); err != nil {
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
	// Created composite types share the graph: a composite referencing a
	// created table's row type comes after that table, and a table whose
	// column uses a created composite comes after the composite. Composites
	// referencing created view row types wait for the views phase instead.
	postView := postViewCompositeCreates(diff)
	composites := make(map[string]*schema.CompositeTypeDiff)
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action == schema.MetadataDiffActionCreate && !postView[compositeDiff] {
			composites[getMigrationObjectID(compositeDiff.SchemaName, compositeDiff.CompositeTypeName)] = compositeDiff
		}
	}
	graph := base.NewGraph()
	for _, id := range sortedMapKeys(composites) {
		graph.AddNode(id)
	}
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
		for _, column := range tableDiff.NewTable.GetColumns() {
			for _, depID := range columnTypeCompositeIDs(column.GetType(), composites) {
				graph.AddEdge(depID, tableID)
			}
		}
	}
	for compositeID, compositeDiff := range composites {
		for _, attribute := range compositeDiff.NewCompositeType.GetAttributes() {
			depSchema, depName, ok := parseQualifiedTypeIdent(attribute.Type)
			if !ok {
				continue
			}
			depID := getMigrationObjectID(depSchema, depName)
			if depID == compositeID {
				continue
			}
			if composites[depID] != nil || maps.tables[depID] != nil {
				graph.AddEdge(depID, compositeID)
			}
		}
	}

	orderedIDs, err := graph.TopologicalSort()
	if err != nil {
		orderedIDs = append(sortedMapKeys(composites), sortedMapKeys(maps.tables)...)
	}
	for _, id := range orderedIDs {
		if compositeDiff := composites[id]; compositeDiff != nil {
			if err := writeCompositeTypeDiff(out, compositeDiff); err != nil {
				return err
			}
			continue
		}
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

func writeDropSequenceOwnershipBeforeTableDrops(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, sequenceDiff := range diff.SequenceChanges {
		if sequenceDiff.Action != schema.MetadataDiffActionAlter || !sequenceOwnershipChanged(sequenceDiff.OldSequence, sequenceDiff.NewSequence) {
			continue
		}
		if sequenceDiff.OldSequence.GetOwnerTable() == "" || sequenceDiff.OldSequence.GetOwnerColumn() == "" {
			continue
		}
		if !tableIsDropped(diff, sequenceDiff.SchemaName, sequenceDiff.OldSequence.GetOwnerTable()) {
			continue
		}
		if err := writeAlterSequenceOwnedByNone(out, sequenceDiff.SchemaName, sequenceDiff.SequenceName); err != nil {
			return err
		}
	}
	return nil
}

func writeSequenceOwnershipAfterTables(out *strings.Builder, diff *schema.MetadataDiff) error {
	for _, sequenceDiff := range diff.SequenceChanges {
		switch sequenceDiff.Action {
		case schema.MetadataDiffActionCreate:
			if sequenceDiff.NewSequence.GetOwnerTable() != "" && sequenceDiff.NewSequence.GetOwnerColumn() != "" {
				if err := writeAlterSequenceOwnedBy(out, sequenceDiff.SchemaName, sequenceDiff.NewSequence); err != nil {
					return err
				}
			}
		case schema.MetadataDiffActionAlter:
			if !sequenceOwnershipChanged(sequenceDiff.OldSequence, sequenceDiff.NewSequence) {
				continue
			}
			if sequenceDiff.NewSequence.GetOwnerTable() != "" && sequenceDiff.NewSequence.GetOwnerColumn() != "" {
				if err := writeAlterSequenceOwnedBy(out, sequenceDiff.SchemaName, sequenceDiff.NewSequence); err != nil {
					return err
				}
			} else if !tableIsDropped(diff, sequenceDiff.SchemaName, sequenceDiff.OldSequence.GetOwnerTable()) {
				if err := writeAlterSequenceOwnedByNone(out, sequenceDiff.SchemaName, sequenceDiff.SequenceName); err != nil {
					return err
				}
			}
		default:
		}
	}
	return nil
}

func sequenceOwnershipChanged(oldSequence, newSequence *storepb.SequenceMetadata) bool {
	return oldSequence.GetOwnerTable() != newSequence.GetOwnerTable() || oldSequence.GetOwnerColumn() != newSequence.GetOwnerColumn()
}

func tableIsDropped(diff *schema.MetadataDiff, schemaName, tableName string) bool {
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop && tableDiff.SchemaName == schemaName && tableDiff.TableName == tableName {
			return true
		}
	}
	return false
}

func writeCreateRoutinesViewsAndMaterializedViews(out *strings.Builder, diff *schema.MetadataDiff) error {
	maps := buildCreateObjectMaps(diff)
	dependentFunctions := functionsDependingOnCreatedViewsOrMaterializedViews(maps)
	for _, id := range sortedMapKeys(maps.functions) {
		if dependentFunctions[id] {
			continue
		}
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
	for _, id := range sortedMapKeys(dependentFunctions) {
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
	for functionID := range dependentFunctions {
		for _, dep := range maps.functions[functionID].NewFunction.GetDependencyTables() {
			depID := getMigrationObjectID(dep.GetSchema(), dep.GetTable())
			if hasNode(depID) {
				graph.AddEdge(depID, functionID)
			}
		}
	}
	orderedIDs, err := graph.TopologicalSort()
	if err != nil {
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.views)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(maps.materializedViews)...)
		orderedIDs = append(orderedIDs, sortedMapKeys(dependentFunctions)...)
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
		case maps.functions[id] != nil:
			if err := writeFunctionDiff(out, maps.functions[id]); err != nil {
				return err
			}
		default:
		}
	}
	return nil
}

func functionsDependingOnCreatedViewsOrMaterializedViews(maps *metadataObjectMaps) map[string]bool {
	viewIDs := make(map[string]bool)
	for id := range maps.views {
		viewIDs[id] = true
	}
	for id := range maps.materializedViews {
		viewIDs[id] = true
	}
	dependentFunctions := make(map[string]bool)
	for functionID, functionDiff := range maps.functions {
		for _, dep := range functionDiff.NewFunction.GetDependencyTables() {
			if viewIDs[getMigrationObjectID(dep.GetSchema(), dep.GetTable())] {
				dependentFunctions[functionID] = true
				break
			}
		}
	}
	return dependentFunctions
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
	if err := writeDropSchemaNonTypeObjects(out, schemaName, schemaMeta); err != nil {
		return err
	}
	return writeDropSchemaTypes(out, schemaName, schemaMeta)
}

func writeDropSchemaNonTypeObjects(out *strings.Builder, schemaName string, schemaMeta *storepb.SchemaMetadata) error {
	if schemaMeta == nil {
		return nil
	}
	diff := buildDropSchemaObjectsDiff(schemaName, schemaMeta)
	// This pass emits only non-type objects; the caller drops the schema's
	// types afterwards. Clearing the type changes keeps the shared graph in
	// writeDropDependentObjects from emitting them here too.
	diff.CompositeTypeChanges = nil
	diff.EnumTypeChanges = nil
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
	return nil
}

func writeDropSchemaTypes(out *strings.Builder, schemaName string, schemaMeta *storepb.SchemaMetadata) error {
	if schemaMeta == nil {
		return nil
	}
	diff := buildDropSchemaObjectsDiff(schemaName, schemaMeta)
	// Composite types drop before enum types they may reference.
	if err := writeCompositeTypeDropsInReverseDependencyOrder(out, diff.CompositeTypeChanges); err != nil {
		return err
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
	for _, compositeType := range schemaMeta.GetCompositeTypes() {
		if compositeType.GetSkipDump() {
			continue
		}
		diff.CompositeTypeChanges = append(diff.CompositeTypeChanges, &schema.CompositeTypeDiff{
			Action:            schema.MetadataDiffActionDrop,
			SchemaName:        schemaName,
			CompositeTypeName: compositeType.GetName(),
			OldCompositeType:  compositeType,
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
	if oldSequence.GetComment() != sequence.GetComment() {
		return writeComment(out, fmt.Sprintf("SEQUENCE \"%s\".\"%s\"", schemaName, sequence.Name), sequence.GetComment())
	}
	return nil
}

func writeAlterSequenceOwnedByNone(out *strings.Builder, schemaName, sequenceName string) error {
	_, err := fmt.Fprintf(out, "ALTER SEQUENCE \"%s\".\"%s\" OWNED BY NONE;\n\n", schemaName, sequenceName)
	return err
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

// releasedCompositeAttributeKey identifies an old composite attribute whose
// releasing statement (DROP ATTRIBUTE or ALTER ATTRIBUTE ... TYPE) is emitted
// early in the drop phase because its old type references a dropped table's
// row type.
func releasedCompositeAttributeKey(schemaName, typeName, attributeName string) string {
	return schemaName + "\x00" + typeName + "\x00" + attributeName
}

// Modes for a released composite attribute.
const (
	releasedModeRetypeEarly = iota + 1
	releasedModeDropReadd
)

// releasingCompositeAttributes finds old attributes of ALTERED composites
// whose type references a dropped table and which the alter removes or
// retypes — those statements must run before the table drop. When the
// retype's target type is itself created in this migration, the early
// statement is a DROP ATTRIBUTE and the create phase re-adds it.
func releasingCompositeAttributes(diff *schema.MetadataDiff) map[string]int {
	droppedTables := make(map[string]bool)
	createdTypes := make(map[string]bool)
	for _, tableDiff := range diff.TableChanges {
		switch tableDiff.Action {
		case schema.MetadataDiffActionDrop:
			droppedTables[tableDiff.SchemaName+"\x00"+tableDiff.TableName] = true
		case schema.MetadataDiffActionCreate:
			createdTypes[tableDiff.SchemaName+"\x00"+tableDiff.TableName] = true
		default:
		}
	}
	// Dropped views and materialized views provide row types too.
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop {
			droppedTables[viewDiff.SchemaName+"\x00"+viewDiff.ViewName] = true
		}
	}
	for _, viewDiff := range diff.MaterializedViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop {
			droppedTables[viewDiff.SchemaName+"\x00"+viewDiff.MaterializedViewName] = true
		}
	}
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action == schema.MetadataDiffActionCreate {
			createdTypes[compositeDiff.SchemaName+"\x00"+compositeDiff.CompositeTypeName] = true
		}
	}
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate {
			createdTypes[viewDiff.SchemaName+"\x00"+viewDiff.ViewName] = true
		}
	}
	for _, viewDiff := range diff.MaterializedViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate {
			createdTypes[viewDiff.SchemaName+"\x00"+viewDiff.MaterializedViewName] = true
		}
	}
	released := make(map[string]int)
	if len(droppedTables) == 0 {
		return released
	}
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action != schema.MetadataDiffActionAlter {
			continue
		}
		newAttributes := make(map[string]*storepb.CompositeTypeAttribute)
		for _, attribute := range compositeDiff.NewCompositeType.GetAttributes() {
			newAttributes[attribute.Name] = attribute
		}
		for _, oldAttribute := range compositeDiff.OldCompositeType.GetAttributes() {
			depSchema, depName, ok := parseQualifiedTypeIdent(oldAttribute.Type)
			if !ok || !droppedTables[depSchema+"\x00"+depName] {
				continue
			}
			newAttribute := newAttributes[oldAttribute.Name]
			if newAttribute == nil || newAttribute.Type == oldAttribute.Type {
				if newAttribute == nil {
					released[releasedCompositeAttributeKey(compositeDiff.SchemaName, compositeDiff.CompositeTypeName, oldAttribute.Name)] = releasedModeRetypeEarly
				}
				continue
			}
			mode := releasedModeRetypeEarly
			if newSchema, newName, ok := parseQualifiedTypeIdent(newAttribute.Type); ok && createdTypes[newSchema+"\x00"+newName] {
				// The target type does not exist during the drop phase.
				mode = releasedModeDropReadd
			}
			released[releasedCompositeAttributeKey(compositeDiff.SchemaName, compositeDiff.CompositeTypeName, oldAttribute.Name)] = mode
		}
	}
	return released
}

// writeReleasingCompositeAlters emits, before the dependent-object graph,
// the alter statements that release dropped tables' row types. Residual
// corner: a dropped view over such a composite drops later inside the graph.
func writeReleasingCompositeAlters(out *strings.Builder, diff *schema.MetadataDiff) error {
	released := releasingCompositeAttributes(diff)
	if len(released) == 0 {
		return nil
	}
	for _, compositeDiff := range diff.CompositeTypeChanges {
		if compositeDiff.Action != schema.MetadataDiffActionAlter {
			continue
		}
		newAttributes := make(map[string]*storepb.CompositeTypeAttribute)
		for _, attribute := range compositeDiff.NewCompositeType.GetAttributes() {
			newAttributes[attribute.Name] = attribute
		}
		for _, oldAttribute := range compositeDiff.OldCompositeType.GetAttributes() {
			mode := released[releasedCompositeAttributeKey(compositeDiff.SchemaName, compositeDiff.CompositeTypeName, oldAttribute.Name)]
			if mode == 0 {
				continue
			}
			newAttribute := newAttributes[oldAttribute.Name]
			if mode == releasedModeRetypeEarly && newAttribute != nil {
				if err := writeAlterCompositeAttribute(out, compositeDiff.SchemaName, compositeDiff.CompositeTypeName, alterAttributeAction, newAttribute); err != nil {
					return err
				}
				continue
			}
			// Attribute dropped outright, or its retype target is created
			// only later in this migration: drop now, re-add in the create
			// phase.
			if _, err := fmt.Fprintf(out, "ALTER TYPE \"%s\".\"%s\" DROP ATTRIBUTE \"%s\";\n\n", compositeDiff.SchemaName, compositeDiff.CompositeTypeName, oldAttribute.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeCompositeTypeDiff(out *strings.Builder, compositeDiff *schema.CompositeTypeDiff) error {
	return writeCompositeTypeDiffWithReleased(out, compositeDiff, nil)
}

// writeCompositeTypeDiffWithReleased is writeCompositeTypeDiff for the alter
// path, skipping releasing statements already emitted in the drop phase.
func writeCompositeTypeDiffWithReleased(out *strings.Builder, compositeDiff *schema.CompositeTypeDiff, released map[string]int) error {
	switch compositeDiff.Action {
	case schema.MetadataDiffActionCreate:
		if err := writeCompositeType(out, compositeDiff.SchemaName, compositeDiff.NewCompositeType); err != nil {
			return err
		}
		if _, err := out.WriteString(";\n\n"); err != nil {
			return err
		}
		if compositeTypeHasComments(compositeDiff.NewCompositeType) {
			return writeCompositeTypeComments(out, compositeDiff.SchemaName, compositeDiff.NewCompositeType)
		}
		return nil
	case schema.MetadataDiffActionDrop:
		_, err := fmt.Fprintf(out, "DROP TYPE \"%s\".\"%s\";\n\n", compositeDiff.SchemaName, compositeDiff.CompositeTypeName)
		return err
	case schema.MetadataDiffActionAlter:
		return writeAlterCompositeType(out, compositeDiff.SchemaName, compositeDiff.OldCompositeType, compositeDiff.NewCompositeType, released)
	default:
		return nil
	}
}

// alterAttributeAction is the writeAlterCompositeAttribute action that emits
// ALTER ATTRIBUTE ... TYPE rather than ADD ATTRIBUTE.
const alterAttributeAction = "ALTER ATTRIBUTE"

// writeAlterCompositeType diffs attributes by name and emits real DDL.
// ALTER ATTRIBUTE ... TYPE fails at execution when any table column uses the
// type — PostgreSQL has no online path for that change; the reviewer sees the
// statement and the executor surfaces the error. Attribute reordering has no
// DDL at all and only produces a warning comment.
func writeAlterCompositeType(out *strings.Builder, schemaName string, oldComposite, newComposite *storepb.CompositeTypeMetadata, released map[string]int) error {
	oldAttributeMap := make(map[string]*storepb.CompositeTypeAttribute)
	for _, attribute := range oldComposite.GetAttributes() {
		oldAttributeMap[attribute.Name] = attribute
	}
	newAttributeMap := make(map[string]*storepb.CompositeTypeAttribute)
	for _, attribute := range newComposite.GetAttributes() {
		newAttributeMap[attribute.Name] = attribute
	}

	// Dropped attributes, in old declaration order. Dropping an attribute
	// discards its data in every value of the type.
	for _, oldAttribute := range oldComposite.GetAttributes() {
		if released[releasedCompositeAttributeKey(schemaName, newComposite.GetName(), oldAttribute.Name)] != 0 {
			continue
		}
		if _, exists := newAttributeMap[oldAttribute.Name]; !exists {
			if _, err := fmt.Fprintf(out, "ALTER TYPE \"%s\".\"%s\" DROP ATTRIBUTE \"%s\";\n\n", schemaName, newComposite.GetName(), oldAttribute.Name); err != nil {
				return err
			}
		}
	}

	// Added attributes, in new declaration order.
	for _, newAttribute := range newComposite.GetAttributes() {
		if _, exists := oldAttributeMap[newAttribute.Name]; !exists {
			if err := writeAlterCompositeAttribute(out, schemaName, newComposite.GetName(), "ADD ATTRIBUTE", newAttribute); err != nil {
				return err
			}
		}
	}

	// Attributes whose type or collation changed.
	for _, newAttribute := range newComposite.GetAttributes() {
		oldAttribute, exists := oldAttributeMap[newAttribute.Name]
		if !exists {
			continue
		}
		mode := released[releasedCompositeAttributeKey(schemaName, newComposite.GetName(), newAttribute.Name)]
		if mode == releasedModeDropReadd {
			// The early drop-phase statement removed the attribute; re-add
			// it now that its target type exists.
			if err := writeAlterCompositeAttribute(out, schemaName, newComposite.GetName(), "ADD ATTRIBUTE", newAttribute); err != nil {
				return err
			}
			continue
		}
		if mode != 0 {
			continue
		}
		if oldAttribute.Type != newAttribute.Type || oldAttribute.Collation != newAttribute.Collation {
			if err := writeAlterCompositeAttribute(out, schemaName, newComposite.GetName(), alterAttributeAction, newAttribute); err != nil {
				return err
			}
		}
	}

	if compositeAttributesReordered(oldComposite, newComposite) {
		if _, err := fmt.Fprintf(out, "-- WARNING: PostgreSQL does not support reordering the attributes of \"%s\".\"%s\"\n\n", schemaName, newComposite.GetName()); err != nil {
			return err
		}
	}

	if oldComposite.GetComment() != newComposite.GetComment() {
		if err := writeComment(out, fmt.Sprintf("TYPE \"%s\".\"%s\"", schemaName, newComposite.GetName()), newComposite.GetComment()); err != nil {
			return err
		}
	}
	for _, newAttribute := range newComposite.GetAttributes() {
		oldAttribute, exists := oldAttributeMap[newAttribute.Name]
		oldComment := ""
		if exists {
			oldComment = oldAttribute.Comment
		}
		// Comments of added attributes are covered here as well: ADD ATTRIBUTE
		// carries no comment syntax.
		if oldComment != newAttribute.Comment {
			if err := writeComment(out, fmt.Sprintf("COLUMN \"%s\".\"%s\".\"%s\"", schemaName, newComposite.GetName(), newAttribute.Name), newAttribute.Comment); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeAlterCompositeAttribute(out *strings.Builder, schemaName, typeName, action string, attribute *storepb.CompositeTypeAttribute) error {
	typeClause := attribute.Type
	if action == alterAttributeAction {
		typeClause = "TYPE " + attribute.Type
	}
	if _, err := fmt.Fprintf(out, "ALTER TYPE \"%s\".\"%s\" %s \"%s\" %s", schemaName, typeName, action, attribute.Name, typeClause); err != nil {
		return err
	}
	// The collation is stored as an emit-ready identifier reference
	// (quoted as needed, schema-qualified outside pg_catalog).
	if attribute.Collation != "" {
		if _, err := fmt.Fprintf(out, " COLLATE %s", attribute.Collation); err != nil {
			return err
		}
	}
	_, err := out.WriteString(";\n\n")
	return err
}

// compositeAttributesReordered reports whether the attribute order ALTER TYPE
// can achieve differs from the target order. Surviving attributes keep their
// old relative positions and ADD ATTRIBUTE always appends at the end, so any
// target that deviates from that achievable order — including an attribute
// inserted in the middle — is unreachable and warrants the warning.
func compositeAttributesReordered(oldComposite, newComposite *storepb.CompositeTypeMetadata) bool {
	oldNames := make(map[string]bool)
	for _, attribute := range oldComposite.GetAttributes() {
		oldNames[attribute.Name] = true
	}
	newNames := make(map[string]bool)
	for _, attribute := range newComposite.GetAttributes() {
		newNames[attribute.Name] = true
	}
	var achieved []string
	for _, attribute := range oldComposite.GetAttributes() {
		if newNames[attribute.Name] {
			achieved = append(achieved, attribute.Name)
		}
	}
	for _, attribute := range newComposite.GetAttributes() {
		if !oldNames[attribute.Name] {
			achieved = append(achieved, attribute.Name)
		}
	}
	var target []string
	for _, attribute := range newComposite.GetAttributes() {
		target = append(target, attribute.Name)
	}
	return !slices.Equal(achieved, target)
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
