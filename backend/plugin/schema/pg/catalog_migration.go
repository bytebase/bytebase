package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/store/model"
)

// diffMetadataMigration generates the migration between two synced snapshots
// with omni's catalog. PostgreSQL and CockroachDB share it: their metadata is
// PostgreSQL-shaped and omni has no CockroachDB catalog.
//
// An object omni cannot install as the snapshot defines it (a stand-in, or one
// left out) is safe to leave in place only when it is the same on both sides: the
// two loads then agree on it and the diff skips it. Where it differs, the diff would
// compare stand-ins, so the migration is refused rather than guessed.
func diffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	from, fromReport, err := loadMetadataCatalog(oldSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load source schema")
	}
	to, toReport, err := loadMetadataCatalog(newSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load target schema")
	}
	if err := unloadableChanges(oldSchema.GetProto(), newSchema.GetProto(), fromReport, toReport); err != nil {
		return "", err
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}
	plan := catalog.GenerateMigration(from, to, diff)
	restoreIndexDefinitions(plan, newSchema.GetProto())
	releaseCompositeBeforeTableDrop(plan, oldSchema.GetProto(), newSchema.GetProto())
	restoreUnloadableDependentViews(plan, oldSchema.GetProto(), newSchema.GetProto(), fromReport, toReport)
	restoreRecreatedViewComments(plan, newSchema.GetProto())
	plan = plan.Filter(func(op catalog.MigrationOp) bool {
		return op.SchemaName != "bbdataarchive"
	})
	return plan.SQL(), nil
}

// loadMetadataCatalog installs a snapshot under the search path its database is
// configured with, which its view bodies resolve against.
func loadMetadataCatalog(d *model.DatabaseMetadata) (*catalog.Catalog, *catalog.LoadMetadataReport, error) {
	c := catalog.New()
	meta := &metadatapb.DatabaseSchemaMetadata{}
	if d != nil {
		if searchPath := getConfiguredSearchPath(d); len(searchPath) > 0 {
			c.SetSearchPath(searchPath)
		}
		if p := d.GetProto(); p != nil {
			meta = withIdentityColumns(p)
		}
	}
	report, err := c.LoadMetadata(context.Background(), meta, catalog.LoadMetadataOptions{Full: true})
	if err != nil {
		return nil, nil, err
	}
	return c, report, nil
}

// releaseCompositeBeforeTableDrop moves or splits an ALTER TYPE that releases a
// composite attribute from a table's or view's row type ahead of that object's
// drop. A CASCADE drop would otherwise remove the attribute first. When the new
// attribute type needs an object created in this migration, the release is
// DROP ATTRIBUTE before the drop and ADD ATTRIBUTE after the plan.
func releaseCompositeBeforeTableDrop(plan *catalog.MigrationPlan, oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata) {
	dropped := relationSet(oldMeta, newMeta)
	created := relationSet(newMeta, oldMeta)
	if len(dropped) == 0 {
		return
	}
	for _, schema := range oldMeta.GetSchemas() {
		newSchema := schemaByName(newMeta, schema.GetName())
		for _, composite := range schema.GetCompositeTypes() {
			newComposite := compositeByName(newSchema, composite.GetName())
			for _, attribute := range composite.GetAttributes() {
				dropKey, ok := referencedRelation(attribute.GetType(), dropped)
				if !ok {
					continue
				}
				newType := attributeType(newComposite, attribute.GetName())
				if newType == attribute.GetType() {
					continue
				}
				alter := findAlterAttribute(plan, schema.GetName(), composite.GetName(), attribute.GetName())
				drop := findDropObject(plan, dropKey)
				if alter < 0 || drop < 0 || alter < drop {
					continue
				}
				if createdKey, dependsOnCreated := referencedRelation(newType, created); dependsOnCreated {
					plan.Ops[alter].SQL = "ALTER TYPE " + quoteQualified(schema.GetName(), composite.GetName()) + " DROP ATTRIBUTE " + quoteIdentifier(attribute.GetName())
					plan.Ops[alter].Phase = catalog.PhasePre
					plan.Ops = moveOp(plan.Ops, alter, drop)
					add := catalog.MigrationOp{
						Type:       catalog.OpAlterType,
						SchemaName: schema.GetName(),
						ObjectName: composite.GetName(),
						SQL:        addAttributeSQL(schema.GetName(), composite.GetName(), attribute, newType),
						Phase:      catalog.PhaseMain,
						ObjType:    'r',
					}
					plan.Ops = insertAfterObjectCreate(plan.Ops, createdKey, add)
					continue
				}
				plan.Ops = moveOp(plan.Ops, alter, drop)
			}
		}
	}
}

func insertAfterObjectCreate(ops []catalog.MigrationOp, key string, op catalog.MigrationOp) []catalog.MigrationOp {
	for i, existing := range ops {
		if (existing.Type == catalog.OpCreateTable || existing.Type == catalog.OpCreateView) && existing.SchemaName+"."+existing.ObjectName == key {
			result := make([]catalog.MigrationOp, 0, len(ops)+1)
			result = append(result, ops[:i+1]...)
			result = append(result, op)
			result = append(result, ops[i+1:]...)
			return result
		}
	}
	return append(ops, op)
}

func restoreIndexDefinitions(plan *catalog.MigrationPlan, meta *metadatapb.DatabaseSchemaMetadata) {
	for i, op := range plan.Ops {
		if op.Type != catalog.OpCreateIndex {
			continue
		}
		index := indexByName(meta, op.SchemaName, op.ObjectName)
		if index == nil || index.GetDefinition() == "" {
			continue
		}
		plan.Ops[i].SQL = strings.TrimRight(strings.TrimSpace(index.GetDefinition()), ";")
	}
}

func indexByName(meta *metadatapb.DatabaseSchemaMetadata, schemaName, name string) *metadatapb.IndexMetadata {
	schema := schemaByName(meta, schemaName)
	if schema == nil {
		return nil
	}
	for _, table := range schema.GetTables() {
		for _, index := range table.GetIndexes() {
			if index.GetName() == name {
				return index
			}
		}
	}
	for _, view := range schema.GetMaterializedViews() {
		for _, index := range view.GetIndexes() {
			if index.GetName() == name {
				return index
			}
		}
	}
	return nil
}

func addAttributeSQL(schema, composite string, attribute *metadatapb.CompositeTypeAttribute, newType string) string {
	sql := "ALTER TYPE " + quoteQualified(schema, composite) + " ADD ATTRIBUTE " + quoteIdentifier(attribute.GetName()) + " " + newType
	if attribute.GetCollation() != "" {
		sql += " COLLATE " + attribute.GetCollation()
	}
	return sql
}

func findAlterAttribute(plan *catalog.MigrationPlan, schema, composite, attribute string) int {
	for i, op := range plan.Ops {
		if op.Type == catalog.OpAlterType && op.SchemaName == schema && op.ObjectName == composite && strings.Contains(op.SQL, quoteIdentifier(attribute)) {
			return i
		}
	}
	return -1
}

func findDropObject(plan *catalog.MigrationPlan, key string) int {
	for i, op := range plan.Ops {
		if op.Type != catalog.OpDropTable && op.Type != catalog.OpDropView {
			continue
		}
		if op.SchemaName+"."+op.ObjectName == key {
			return i
		}
	}
	return -1
}

func moveOp(ops []catalog.MigrationOp, from, to int) []catalog.MigrationOp {
	op := ops[from]
	rest := make([]catalog.MigrationOp, 0, len(ops)-1)
	rest = append(rest, ops[:from]...)
	rest = append(rest, ops[from+1:]...)
	result := make([]catalog.MigrationOp, 0, len(ops))
	result = append(result, rest[:to]...)
	result = append(result, op)
	result = append(result, rest[to:]...)
	return result
}

func relationSet(left, right *metadatapb.DatabaseSchemaMetadata) map[string]bool {
	result := make(map[string]bool)
	for _, schema := range left.GetSchemas() {
		other := schemaByName(right, schema.GetName())
		for _, table := range schema.GetTables() {
			if other == nil || tableByName(other, table.GetName()) == nil {
				result[schema.GetName()+"."+table.GetName()] = true
			}
		}
		for _, view := range schema.GetViews() {
			if other == nil || viewByName(other, view.GetName()) == nil {
				result[schema.GetName()+"."+view.GetName()] = true
			}
		}
		for _, view := range schema.GetMaterializedViews() {
			if other == nil || materializedViewByName(other, view.GetName()) == nil {
				result[schema.GetName()+"."+view.GetName()] = true
			}
		}
	}
	return result
}

func referencedRelation(typeName string, relations map[string]bool) (string, bool) {
	name := strings.TrimSpace(typeName)
	for strings.HasSuffix(name, "[]") {
		name = strings.TrimSuffix(name, "[]")
	}
	name = strings.ReplaceAll(name, `"`, "")
	for key := range relations {
		if strings.EqualFold(name, key) {
			return key, true
		}
	}
	return "", false
}

func compositeByName(schema *metadatapb.SchemaMetadata, name string) *metadatapb.CompositeTypeMetadata {
	if schema == nil {
		return nil
	}
	for _, composite := range schema.GetCompositeTypes() {
		if composite.GetName() == name {
			return composite
		}
	}
	return nil
}

func attributeType(composite *metadatapb.CompositeTypeMetadata, name string) string {
	if composite == nil {
		return ""
	}
	for _, attribute := range composite.GetAttributes() {
		if attribute.GetName() == name {
			return attribute.GetType()
		}
	}
	return ""
}

// restoreUnloadableDependentViews recreates a view the catalog could not parse
// when a column it depends on is dropped or retyped. The unparsed view is a
// stand-in on both sides, so the catalog has no dependency edge and would alter
// the column under a view that must be dropped first. The snapshot's
// DependencyColumns is the only record of that edge.
func restoreUnloadableDependentViews(plan *catalog.MigrationPlan, oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, reports ...*catalog.LoadMetadataReport) {
	affected := affectedTableColumns(oldMeta, newMeta)
	if len(affected) == 0 {
		return
	}
	unloadable := make(map[string]bool)
	for _, report := range reports {
		for _, m := range []map[string]error{report.Degraded, report.Missing} {
			for key := range m {
				unloadable[key] = true
			}
		}
	}
	var items []restoreItem
	for _, schema := range newMeta.GetSchemas() {
		oldSchema := schemaByName(oldMeta, schema.GetName())
		if oldSchema == nil {
			continue
		}
		add := func(key string, dependencies []*metadatapb.DependencyColumn, dropSQL, createSQL string) {
			var deps []string
			for _, dependency := range dependencies {
				deps = append(deps, dependency.GetSchema()+"."+dependency.GetTable())
			}
			items = append(items, restoreItem{key: key, dropSQL: dropSQL, createSQL: createSQL, deps: deps})
		}
		for _, view := range schema.GetViews() {
			key := schema.GetName() + "." + view.GetName()
			oldView := viewByName(oldSchema, view.GetName())
			if oldView == nil || !proto.Equal(oldView, view) || !unloadable["rel:"+key] {
				continue
			}
			if !dependsOnAffectedColumn(view.GetDependencyColumns(), affected) && !dependsOnAffectedColumn(oldView.GetDependencyColumns(), affected) {
				continue
			}
			createSQL, err := catalog.ViewDefinition(schema.GetName(), view)
			if err != nil {
				continue
			}
			add(key, view.GetDependencyColumns(),
				"DROP VIEW IF EXISTS "+quoteQualified(schema.GetName(), view.GetName()),
				trimDefinition(createSQL))
		}
		for _, view := range schema.GetMaterializedViews() {
			key := schema.GetName() + "." + view.GetName()
			oldView := materializedViewByName(oldSchema, view.GetName())
			if oldView == nil || !proto.Equal(oldView, view) || !unloadable["rel:"+key] {
				continue
			}
			if !dependsOnAffectedColumn(view.GetDependencyColumns(), affected) && !dependsOnAffectedColumn(oldView.GetDependencyColumns(), affected) {
				continue
			}
			createSQL, err := catalog.MaterializedViewDefinition(schema.GetName(), view)
			if err != nil {
				continue
			}
			add(key, view.GetDependencyColumns(),
				"DROP MATERIALIZED VIEW IF EXISTS "+quoteQualified(schema.GetName(), view.GetName()),
				trimDefinition(createSQL))
		}
	}
	if len(items) == 0 {
		return
	}
	order := restoreOrder(items)
	ops := make([]catalog.MigrationOp, 0, len(plan.Ops)+len(items)*2)
	for i := len(order) - 1; i >= 0; i-- {
		item := items[order[i]]
		schemaName, _, _ := strings.Cut(item.key, ".")
		ops = append(ops, catalog.MigrationOp{
			Type:       catalog.OpDropView,
			SchemaName: schemaName,
			ObjectName: strings.TrimPrefix(item.key, schemaName+"."),
			SQL:        item.dropSQL,
			Phase:      catalog.PhasePre,
			ObjType:    'r',
		})
	}
	ops = append(ops, plan.Ops...)
	for _, index := range order {
		item := items[index]
		schemaName, _, _ := strings.Cut(item.key, ".")
		ops = append(ops, catalog.MigrationOp{
			Type:       catalog.OpCreateView,
			SchemaName: schemaName,
			ObjectName: strings.TrimPrefix(item.key, schemaName+"."),
			SQL:        item.createSQL,
			Phase:      catalog.PhaseMain,
			ObjType:    'r',
		})
	}
	plan.Ops = ops
}

// restoreOrder puts each restore item after the selected items it depends on,
// so drops run in reverse and creates in dependency order.
type restoreItem struct {
	key, dropSQL, createSQL string
	deps                    []string
}

func restoreOrder(items []restoreItem) []int {
	selected := make(map[string]int, len(items))
	for i, item := range items {
		selected[item.key] = i
	}
	indegree := make([]int, len(items))
	dependents := make([][]int, len(items))
	for i, item := range items {
		seen := make(map[int]bool)
		for _, dep := range item.deps {
			j, ok := selected[dep]
			if !ok || j == i || seen[j] {
				continue
			}
			seen[j] = true
			indegree[i]++
			dependents[j] = append(dependents[j], i)
		}
	}
	var ready []int
	for i := range items {
		if indegree[i] == 0 {
			ready = append(ready, i)
		}
	}
	order := make([]int, 0, len(items))
	for len(ready) > 0 {
		slices.Sort(ready)
		index := ready[0]
		ready = ready[1:]
		order = append(order, index)
		for _, dependent := range dependents[index] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
	}
	for i := range items {
		if len(order) == len(items) {
			break
		}
		found := false
		for _, placed := range order {
			if placed == i {
				found = true
				break
			}
		}
		if !found {
			order = append(order, i)
		}
	}
	return order
}

func trimDefinition(sql string) string {
	return strings.TrimRight(strings.TrimSpace(sql), "; \t\n\r")
}

type affectedColumnSet map[string]map[string]bool

func affectedTableColumns(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata) affectedColumnSet {
	affected := make(affectedColumnSet)
	for _, schema := range oldMeta.GetSchemas() {
		newSchema := schemaByName(newMeta, schema.GetName())
		if newSchema == nil {
			continue
		}
		for _, table := range schema.GetTables() {
			newTable := tableByName(newSchema, table.GetName())
			if newTable == nil {
				continue
			}
			for _, column := range table.GetColumns() {
				newColumn := columnByName(newTable, column.GetName())
				if newColumn != nil && newColumn.GetType() == column.GetType() {
					continue
				}
				tableKey := schema.GetName() + "." + table.GetName()
				if affected[tableKey] == nil {
					affected[tableKey] = make(map[string]bool)
				}
				affected[tableKey][column.GetName()] = true
			}
		}
	}
	return affected
}

func dependsOnAffectedColumn(dependencies []*metadatapb.DependencyColumn, affected affectedColumnSet) bool {
	for _, dependency := range dependencies {
		columns, ok := affected[dependency.GetSchema()+"."+dependency.GetTable()]
		if !ok {
			continue
		}
		if dependency.GetColumn() == "" || columns[dependency.GetColumn()] {
			return true
		}
	}
	return false
}

func schemaByName(meta *metadatapb.DatabaseSchemaMetadata, name string) *metadatapb.SchemaMetadata {
	for _, schema := range meta.GetSchemas() {
		if schema.GetName() == name {
			return schema
		}
	}
	return nil
}

func tableByName(schema *metadatapb.SchemaMetadata, name string) *metadatapb.TableMetadata {
	for _, table := range schema.GetTables() {
		if table.GetName() == name {
			return table
		}
	}
	return nil
}

func columnByName(table *metadatapb.TableMetadata, name string) *metadatapb.ColumnMetadata {
	for _, column := range table.GetColumns() {
		if column.GetName() == name {
			return column
		}
	}
	return nil
}

func viewByName(schema *metadatapb.SchemaMetadata, name string) *metadatapb.ViewMetadata {
	for _, view := range schema.GetViews() {
		if view.GetName() == name {
			return view
		}
	}
	return nil
}

func materializedViewByName(schema *metadatapb.SchemaMetadata, name string) *metadatapb.MaterializedViewMetadata {
	for _, view := range schema.GetMaterializedViews() {
		if view.GetName() == name {
			return view
		}
	}
	return nil
}

// withIdentityColumns sets the identity flag a CockroachDB snapshot leaves off.
// Its sync records identityGeneration but not isIdentity, while the catalog only
// installs an identity column when isIdentity is set; without this, a migration
// creates an ordinary NOT NULL column and loses generation.
func withIdentityColumns(meta *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	out := proto.CloneOf(meta)
	for _, schema := range out.GetSchemas() {
		for _, table := range schema.GetTables() {
			markIdentityColumns(table.GetColumns())
		}
	}
	return out
}

func markIdentityColumns(columns []*metadatapb.ColumnMetadata) {
	for _, column := range columns {
		if column.GetIdentityGeneration() != metadatapb.ColumnMetadata_IDENTITY_GENERATION_UNSPECIFIED {
			column.IsIdentity = true
		}
	}
}

// restoreRecreatedViewComments reapplies a view's stored comment after the plan
// recreates it because a dependency changed. The catalog only diffs comments
// that changed, so the recreation silently dropped an unchanged one.
func restoreRecreatedViewComments(plan *catalog.MigrationPlan, meta *metadatapb.DatabaseSchemaMetadata) {
	commented := make(map[string]bool)
	for _, op := range plan.Ops {
		switch {
		case op.Type == catalog.OpComment:
			commented[op.ObjectName] = true
		case op.Type == catalog.OpCreateView && strings.Contains(op.SQL, "COMMENT ON"):
			commented[op.SchemaName+"."+op.ObjectName] = true
		default:
		}
	}
	for _, schema := range meta.GetSchemas() {
		for _, view := range schema.GetViews() {
			key := schema.GetName() + "." + view.GetName()
			if view.GetComment() != "" && !commented[key] && planCreatesView(plan, key) {
				plan.Ops = append(plan.Ops, catalog.MigrationOp{
					Type:          catalog.OpComment,
					SchemaName:    schema.GetName(),
					ObjectName:    key,
					SQL:           fmt.Sprintf("COMMENT ON VIEW %s IS %s", quoteQualified(schema.GetName(), view.GetName()), quoteLiteral(view.GetComment())),
					Transactional: true,
					Phase:         catalog.PhaseMain,
					ObjType:       'r',
				})
			}
		}
		for _, view := range schema.GetMaterializedViews() {
			key := schema.GetName() + "." + view.GetName()
			if view.GetComment() != "" && !commented[key] && planCreatesView(plan, key) {
				plan.Ops = append(plan.Ops, catalog.MigrationOp{
					Type:          catalog.OpComment,
					SchemaName:    schema.GetName(),
					ObjectName:    key,
					SQL:           fmt.Sprintf("COMMENT ON MATERIALIZED VIEW %s IS %s", quoteQualified(schema.GetName(), view.GetName()), quoteLiteral(view.GetComment())),
					Transactional: true,
					Phase:         catalog.PhaseMain,
					ObjType:       'r',
				})
			}
		}
	}
}

func planCreatesView(plan *catalog.MigrationPlan, key string) bool {
	for _, op := range plan.Ops {
		if op.Type == catalog.OpCreateView && op.SchemaName+"."+op.ObjectName == key {
			return true
		}
	}
	return false
}

func quoteQualified(schema, name string) string {
	return quoteIdentifier(schema) + "." + quoteIdentifier(name)
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quoteLiteral(text string) string {
	return "'" + strings.ReplaceAll(text, "'", "''") + "'"
}

// unloadableChanges returns an error naming each object that either load could
// not install as defined and that differs between the two snapshots.
func unloadableChanges(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, reports ...*catalog.LoadMetadataReport) error {
	reasons := make(map[string]error)
	for _, report := range reports {
		for _, m := range []map[string]error{report.Degraded, report.Missing} {
			for key, err := range m {
				if _, seen := reasons[key]; !seen {
					reasons[key] = err
				}
			}
		}
	}
	if len(reasons) == 0 {
		return nil
	}
	oldObjects, newObjects := snapshotObjects(oldMeta), snapshotObjects(newMeta)
	var changed []string
	for key, err := range reasons {
		if !proto.Equal(oldObjects[key], newObjects[key]) {
			changed = append(changed, key+": "+err.Error())
		}
	}
	if len(changed) == 0 {
		return nil
	}
	slices.Sort(changed)
	return errors.Errorf("cannot generate a migration for objects whose definitions could not be loaded: %s", strings.Join(changed, "; "))
}

// snapshotObjects indexes a snapshot's objects by the keys LoadMetadataReport
// uses. A constraint key maps to its table, which holds the constraint.
func snapshotObjects(meta *metadatapb.DatabaseSchemaMetadata) map[string]proto.Message {
	objects := make(map[string]proto.Message)
	for _, ext := range meta.GetExtensions() {
		objects["ext:"+ext.GetName()] = ext
	}
	for _, s := range meta.GetSchemas() {
		schema := s.GetName()
		objects["schema:"+schema] = &metadatapb.SchemaMetadata{Name: schema, Owner: s.GetOwner(), Comment: s.GetComment()}
		qualified := func(name string) string { return schema + "." + name }
		for _, e := range s.GetEnumTypes() {
			objects["type:"+qualified(e.GetName())] = e
		}
		for _, ct := range s.GetCompositeTypes() {
			objects["type:"+qualified(ct.GetName())] = ct
		}
		for _, seq := range s.GetSequences() {
			objects["seq:"+qualified(seq.GetName())] = seq
		}
		for _, fn := range s.GetFunctions() {
			identity := fn.GetSignature()
			if identity == "" {
				identity = fn.GetDefinition()
			}
			objects["func:"+qualified(fn.GetName())+"|"+identity] = fn
		}
		for _, v := range s.GetViews() {
			objects["rel:"+qualified(v.GetName())] = v
		}
		for _, mv := range s.GetMaterializedViews() {
			objects["rel:"+qualified(mv.GetName())] = mv
			for _, idx := range mv.GetIndexes() {
				objects["idx:"+qualified(mv.GetName())+"."+idx.GetName()] = idx
			}
		}
		for _, t := range s.GetTables() {
			def := tableDefinition(t)
			objects["rel:"+qualified(t.GetName())] = def
			addTableParts(objects, qualified(t.GetName()), def)
			forEachPartition(t.GetPartitions(), func(p *metadatapb.TablePartitionMetadata) {
				objects["rel:"+qualified(p.GetName())] = p
				for _, idx := range p.GetIndexes() {
					objects["idx:"+qualified(p.GetName())+"."+idx.GetName()] = idx
				}
			})
		}
	}
	return objects
}

// addTableParts indexes a table's indexes, and maps each of its constraints to the
// table itself.
func addTableParts(objects map[string]proto.Message, table string, t *metadatapb.TableMetadata) {
	for _, idx := range t.GetIndexes() {
		objects["idx:"+table+"."+idx.GetName()] = idx
		objects["con:"+table+"."+idx.GetName()] = t
	}
	for _, c := range t.GetCheckConstraints() {
		objects["con:"+table+"."+c.GetName()] = t
	}
	for _, c := range t.GetExcludeConstraints() {
		objects["con:"+table+"."+c.GetName()] = t
	}
	for _, fk := range t.GetForeignKeys() {
		objects["con:"+table+"."+fk.GetName()] = t
	}
}

func forEachPartition(partitions []*metadatapb.TablePartitionMetadata, f func(*metadatapb.TablePartitionMetadata)) {
	for _, p := range partitions {
		f(p)
		forEachPartition(p.GetSubpartitions(), f)
	}
}

// tableDefinition is t without the statistics sync records, which change with
// the data rather than the definition.
func tableDefinition(t *metadatapb.TableMetadata) *metadatapb.TableMetadata {
	t = proto.CloneOf(t)
	t.RowCount, t.DataSize, t.IndexSize, t.DataFree = 0, 0, 0, 0
	return t
}
