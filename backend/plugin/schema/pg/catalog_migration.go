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
	orderTypeAndTableCreates(plan, newSchema.GetProto())
	restoreIndexDefinitions(plan, newSchema.GetProto())
	releaseCompositeBeforeTableDrop(plan, oldSchema.GetProto(), newSchema.GetProto())
	if err := restoreUnloadableDependentViews(plan, oldSchema.GetProto(), newSchema.GetProto(), fromReport, toReport); err != nil {
		return "", err
	}
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

// orderTypeAndTableCreates puts a created composite or enum type before the
// tables that use it, and a created table before a composite attribute that uses
// its row type.
func orderTypeAndTableCreates(plan *catalog.MigrationPlan, meta *metadatapb.DatabaseSchemaMetadata) {
	type createNode struct {
		position int
		key      string
		relation bool // true for a table, false for a type
	}
	var nodes []createNode
	byKey := make(map[string]int)
	for i, op := range plan.Ops {
		switch op.Type {
		case catalog.OpCreateTable:
			key := op.SchemaName + "." + op.ObjectName
			byKey[key] = len(nodes)
			nodes = append(nodes, createNode{position: i, key: key, relation: true})
		case catalog.OpCreateType:
			key := op.SchemaName + "." + op.ObjectName
			byKey[key] = len(nodes)
			nodes = append(nodes, createNode{position: i, key: key})
		default:
		}
	}
	if len(nodes) < 2 {
		return
	}
	indegree := make([]int, len(nodes))
	dependents := make([][]int, len(nodes))
	addEdge := func(prerequisite, dependent int) {
		if prerequisite == dependent {
			return
		}
		indegree[dependent]++
		dependents[prerequisite] = append(dependents[prerequisite], dependent)
	}
	for _, schema := range meta.GetSchemas() {
		for _, table := range schema.GetTables() {
			tableIndex, ok := byKey[schema.GetName()+"."+table.GetName()]
			if !ok {
				continue
			}
			for _, column := range table.GetColumns() {
				for key, index := range byKey {
					if nodes[index].relation || !referencesType(column.GetType(), key) {
						continue
					}
					addEdge(index, tableIndex)
				}
			}
		}
		for _, composite := range schema.GetCompositeTypes() {
			compositeIndex, ok := byKey[schema.GetName()+"."+composite.GetName()]
			if !ok {
				continue
			}
			for _, attribute := range composite.GetAttributes() {
				for key, index := range byKey {
					if !referencesType(attribute.GetType(), key) {
						continue
					}
					addEdge(index, compositeIndex)
				}
			}
		}
	}
	order := make([]int, 0, len(nodes))
	ready := make([]int, 0, len(nodes))
	for i := range nodes {
		if indegree[i] == 0 {
			ready = append(ready, i)
		}
	}
	for len(ready) > 0 {
		slices.SortStableFunc(ready, func(a, b int) int { return nodes[a].position - nodes[b].position })
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
	if len(order) != len(nodes) {
		return
	}
	positions := make([]int, 0, len(nodes))
	for _, node := range nodes {
		positions = append(positions, node.position)
	}
	slices.Sort(positions)
	ops := make([]catalog.MigrationOp, len(plan.Ops))
	copy(ops, plan.Ops)
	for i, position := range positions {
		ops[position] = plan.Ops[nodes[order[i]].position]
	}
	plan.Ops = ops
}

func referencesType(typeName, key string) bool {
	name := strings.TrimSpace(typeName)
	for strings.HasSuffix(name, "[]") {
		name = strings.TrimSuffix(name, "[]")
	}
	name = strings.ReplaceAll(name, `"`, "")
	if strings.EqualFold(name, key) {
		return true
	}
	return !strings.Contains(name, ".") && strings.EqualFold(name, key[strings.LastIndex(key, ".")+1:])
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
					plan.Ops = restoreAttributeComment(plan.Ops, schema.GetName(), composite.GetName(), attribute)
					continue
				}
				plan.Ops = moveOp(plan.Ops, alter, drop)
			}
		}
	}
}

func restoreAttributeComment(ops []catalog.MigrationOp, schema, composite string, attribute *metadatapb.CompositeTypeAttribute) []catalog.MigrationOp {
	if attribute.GetComment() == "" {
		return ops
	}
	target := quoteQualified(schema, composite) + "." + quoteIdentifier(attribute.GetName())
	for _, op := range ops {
		if op.Type == catalog.OpComment && strings.Contains(op.SQL, target) {
			return ops
		}
	}
	comment := catalog.MigrationOp{
		Type:       catalog.OpComment,
		SchemaName: schema,
		ObjectName: composite,
		SQL:        "COMMENT ON COLUMN " + target + " IS " + quoteLiteral(attribute.GetComment()),
		Phase:      catalog.PhaseMain,
		ObjType:    'r',
	}
	for i, op := range ops {
		if op.Type == catalog.OpAlterType && op.SchemaName == schema && op.ObjectName == composite && strings.Contains(op.SQL, "ADD ATTRIBUTE "+quoteIdentifier(attribute.GetName())) {
			result := make([]catalog.MigrationOp, 0, len(ops)+1)
			result = append(result, ops[:i+1]...)
			result = append(result, comment)
			result = append(result, ops[i+1:]...)
			return result
		}
	}
	return append(ops, comment)
}

func insertAfterObjectCreate(ops []catalog.MigrationOp, key string, op catalog.MigrationOp) []catalog.MigrationOp {
	for i, existing := range ops {
		if (existing.Type == catalog.OpCreateTable || existing.Type == catalog.OpCreateView || existing.Type == catalog.OpCreateType) && existing.SchemaName+"."+existing.ObjectName == key {
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
		ownerSchema, ownerObject := indexOwnerFromSQL(op.SQL)
		index := indexByName(meta, op.SchemaName, ownerSchema, ownerObject, op.ObjectName)
		if index == nil || index.GetDefinition() == "" {
			continue
		}
		plan.Ops[i].SQL = strings.TrimRight(strings.TrimSpace(index.GetDefinition()), ";")
	}
}

func indexByName(meta *metadatapb.DatabaseSchemaMetadata, fallbackSchema, ownerSchema, ownerObject, name string) *metadatapb.IndexMetadata {
	if ownerSchema != "" {
		schema := schemaByName(meta, ownerSchema)
		if schema == nil {
			return nil
		}
		for _, table := range schema.GetTables() {
			if table.GetName() == ownerObject {
				return findIndex(table.GetIndexes(), name)
			}
		}
		for _, view := range schema.GetMaterializedViews() {
			if view.GetName() == ownerObject {
				return findIndex(view.GetIndexes(), name)
			}
		}
		return nil
	}
	schema := schemaByName(meta, fallbackSchema)
	if schema == nil {
		return nil
	}
	for _, table := range schema.GetTables() {
		if index := findIndex(table.GetIndexes(), name); index != nil {
			return index
		}
	}
	for _, view := range schema.GetMaterializedViews() {
		if index := findIndex(view.GetIndexes(), name); index != nil {
			return index
		}
	}
	return nil
}

func findIndex(indexes []*metadatapb.IndexMetadata, name string) *metadatapb.IndexMetadata {
	for _, index := range indexes {
		if index.GetName() == name {
			return index
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
		for _, enum := range schema.GetEnumTypes() {
			if other == nil || enumByName(other, enum.GetName()) == nil {
				result[schema.GetName()+"."+enum.GetName()] = true
			}
		}
		for _, composite := range schema.GetCompositeTypes() {
			if other == nil || compositeByName(other, composite.GetName()) == nil {
				result[schema.GetName()+"."+composite.GetName()] = true
			}
		}
	}
	return result
}

func enumByName(schema *metadatapb.SchemaMetadata, name string) *metadatapb.EnumTypeMetadata {
	if schema == nil {
		return nil
	}
	for _, enum := range schema.GetEnumTypes() {
		if enum.GetName() == name {
			return enum
		}
	}
	return nil
}

func containsObjectReference(sql, schema, object string) bool {
	if strings.Contains(sql, quoteQualified(schema, object)) {
		return true
	}
	needle := schema + "." + object
	for start := 0; start < len(sql); {
		index := strings.Index(sql[start:], needle)
		if index < 0 {
			return false
		}
		index += start
		end := index + len(needle)
		beforeBoundary := index == 0 || !isIdentifierByte(sql[index-1])
		afterBoundary := end == len(sql) || !isIdentifierByte(sql[end])
		if beforeBoundary && afterBoundary {
			return true
		}
		start = end
	}
	return false
}

func isIdentifierByte(b byte) bool {
	return b == '_' || b == '$' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
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
func restoreUnloadableDependentViews(plan *catalog.MigrationPlan, oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, reports ...*catalog.LoadMetadataReport) error {
	affected := affectedTableColumns(oldMeta, newMeta)
	if len(affected) == 0 {
		return nil
	}
	unloadable := make(map[string]bool)
	for _, report := range reports {
		for _, m := range []map[string]error{report.Degraded, report.Missing} {
			for key := range m {
				unloadable[key] = true
			}
		}
	}
	type candidate struct {
		key, dropSQL, createSQL string
		oldDeps, newDeps        []string
		unloadable, direct      bool
	}
	var candidates []candidate
	for _, schema := range newMeta.GetSchemas() {
		oldSchema := schemaByName(oldMeta, schema.GetName())
		if oldSchema == nil {
			continue
		}
		for _, view := range schema.GetViews() {
			key := schema.GetName() + "." + view.GetName()
			oldView := viewByName(oldSchema, view.GetName())
			if oldView == nil {
				continue
			}
			createSQL, err := catalog.ViewDefinition(schema.GetName(), view)
			if err != nil {
				continue
			}
			candidates = append(candidates, candidate{
				key:        key,
				dropSQL:    "DROP VIEW IF EXISTS " + quoteQualified(schema.GetName(), view.GetName()),
				createSQL:  trimDefinition(createSQL),
				oldDeps:    dependencyKeys(oldView.GetDependencyColumns()),
				newDeps:    dependencyKeys(view.GetDependencyColumns()),
				unloadable: unloadable["rel:"+key],
				direct:     dependsOnAffectedColumn(view.GetDependencyColumns(), affected) || dependsOnAffectedColumn(oldView.GetDependencyColumns(), affected),
			})
		}
		for _, view := range schema.GetMaterializedViews() {
			key := schema.GetName() + "." + view.GetName()
			oldView := materializedViewByName(oldSchema, view.GetName())
			if oldView == nil {
				continue
			}
			createSQL, err := catalog.MaterializedViewDefinition(schema.GetName(), view)
			if err != nil {
				continue
			}
			candidates = append(candidates, candidate{
				key:        key,
				dropSQL:    "DROP MATERIALIZED VIEW IF EXISTS " + quoteQualified(schema.GetName(), view.GetName()),
				createSQL:  trimDefinition(createSQL),
				oldDeps:    dependencyKeys(oldView.GetDependencyColumns()),
				newDeps:    dependencyKeys(view.GetDependencyColumns()),
				unloadable: unloadable["rel:"+key],
				direct:     dependsOnAffectedColumn(view.GetDependencyColumns(), affected) || dependsOnAffectedColumn(oldView.GetDependencyColumns(), affected),
			})
		}
	}
	for _, oldSchema := range oldMeta.GetSchemas() {
		newSchema := schemaByName(newMeta, oldSchema.GetName())
		for _, view := range oldSchema.GetViews() {
			if viewByName(newSchema, view.GetName()) != nil {
				continue
			}
			candidates = append(candidates, candidate{
				key:     oldSchema.GetName() + "." + view.GetName(),
				dropSQL: "DROP VIEW IF EXISTS " + quoteQualified(oldSchema.GetName(), view.GetName()),
				oldDeps: dependencyKeys(view.GetDependencyColumns()),
			})
		}
		for _, view := range oldSchema.GetMaterializedViews() {
			if materializedViewByName(newSchema, view.GetName()) != nil {
				continue
			}
			candidates = append(candidates, candidate{
				key:     oldSchema.GetName() + "." + view.GetName(),
				dropSQL: "DROP MATERIALIZED VIEW IF EXISTS " + quoteQualified(oldSchema.GetName(), view.GetName()),
				oldDeps: dependencyKeys(view.GetDependencyColumns()),
			})
		}
	}
	selected := make(map[string]bool)
	for _, candidate := range candidates {
		if candidate.unloadable && candidate.direct {
			selected[candidate.key] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, candidate := range candidates {
			if selected[candidate.key] {
				continue
			}
			for _, dependency := range unionKeys(candidate.oldDeps, candidate.newDeps) {
				if selected[dependency] {
					selected[candidate.key] = true
					changed = true
					break
				}
			}
		}
	}
	for key := range selected {
		if strings.HasPrefix(key, "bbdataarchive.") {
			return errors.Errorf("cannot generate a migration: backup-schema view %s depends on an object that must be recreated", key)
		}
	}
	var items []restoreItem
	for _, candidate := range candidates {
		if selected[candidate.key] {
			items = append(items, restoreItem{key: candidate.key, dropSQL: candidate.dropSQL, createSQL: candidate.createSQL, oldDeps: candidate.oldDeps, newDeps: candidate.newDeps})
		}
	}
	if len(items) == 0 {
		return nil
	}
	dropOrder := restoreOrder(items, func(item restoreItem) []string { return item.oldDeps })
	createOrder := restoreOrder(items, func(item restoreItem) []string { return item.newDeps })
	dropped := make(map[string]bool, len(items))
	for _, item := range items {
		dropped[item.key] = true
	}
	plan.Ops = slices.DeleteFunc(plan.Ops, func(op catalog.MigrationOp) bool {
		return opTargetsSelected(op, dropped, oldMeta)
	})
	ops := make([]catalog.MigrationOp, 0, len(plan.Ops)+len(items)*2)
	for i := len(dropOrder) - 1; i >= 0; i-- {
		item := items[dropOrder[i]]
		schemaName, objectName := splitObjectKey(item.key)
		ops = append(ops, catalog.MigrationOp{
			Type:       catalog.OpDropView,
			SchemaName: schemaName,
			ObjectName: objectName,
			SQL:        item.dropSQL,
			Phase:      catalog.PhasePre,
			ObjType:    'r',
		})
	}
	ops = append(ops, plan.Ops...)
	for _, index := range createOrder {
		item := items[index]
		schemaName, objectName := splitObjectKey(item.key)
		ops = append(ops, catalog.MigrationOp{
			Type:       catalog.OpCreateView,
			SchemaName: schemaName,
			ObjectName: objectName,
			SQL:        item.createSQL,
			Phase:      catalog.PhaseMain,
			ObjType:    'r',
		})
	}
	plan.Ops = ops
	return nil
}

// opTargetsSelected reports whether an operation belongs to a selected restored
// object. Its accessories (comments, indexes, triggers) are omitted because the
// synthesized create from metadata already carries them.
func opTargetsSelected(op catalog.MigrationOp, selected map[string]bool, oldMeta *metadatapb.DatabaseSchemaMetadata) bool {
	if op.Type == catalog.OpComment {
		for key := range selected {
			if op.ObjectName == key || strings.HasPrefix(op.ObjectName, key+".") {
				return true
			}
		}
		return false
	}
	if selected[op.SchemaName+"."+op.ObjectName] {
		return true
	}
	switch op.Type {
	case catalog.OpCreateIndex, catalog.OpDropIndex:
		if ownerSchema, ownerObject := indexOwnerFromSQL(op.SQL); ownerSchema != "" {
			return selected[ownerSchema+"."+ownerObject]
		}
		return indexNameOwner(oldMeta, op.SchemaName, op.ObjectName, selected) != ""
	case catalog.OpCreateTrigger, catalog.OpDropTrigger:
		for key := range selected {
			schemaName, objectName := splitObjectKey(key)
			if containsObjectReference(op.SQL, schemaName, objectName) {
				return true
			}
		}
	default:
	}
	return false
}

func indexOwnerFromSQL(sql string) (string, string) {
	index := strings.Index(sql, " ON ")
	if index < 0 {
		return "", ""
	}
	rest := strings.TrimSpace(sql[index+4:])
	if strings.HasPrefix(rest, "ONLY ") {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "ONLY "))
	}
	schemaName, next, ok := quotedIdentifier(rest, 0)
	if !ok || next >= len(rest) || rest[next] != '.' {
		return "", ""
	}
	objectName, _, ok := quotedIdentifier(rest, next+1)
	if !ok {
		return "", ""
	}
	return schemaName, objectName
}

func quotedIdentifier(text string, start int) (string, int, bool) {
	if start >= len(text) {
		return "", start, false
	}
	if text[start] == '"' {
		var b strings.Builder
		for i := start + 1; i < len(text); i++ {
			if text[i] != '"' {
				b.WriteByte(text[i])
				continue
			}
			if i+1 < len(text) && text[i+1] == '"' {
				b.WriteByte('"')
				i++
				continue
			}
			return b.String(), i + 1, true
		}
		return "", start, false
	}
	end := start
	for end < len(text) && (isIdentifierByte(text[end]) || text[end] == '$') {
		end++
	}
	if end == start {
		return "", start, false
	}
	return text[start:end], end, true
}

func indexNameOwner(meta *metadatapb.DatabaseSchemaMetadata, schemaName, indexName string, selected map[string]bool) string {
	schema := schemaByName(meta, schemaName)
	if schema == nil {
		return ""
	}
	found := ""
	for _, view := range schema.GetMaterializedViews() {
		for _, index := range view.GetIndexes() {
			if index.GetName() != indexName {
				continue
			}
			key := schemaName + "." + view.GetName()
			if !selected[key] {
				continue
			}
			if found != "" {
				return ""
			}
			found = key
		}
	}
	return found
}

func unionKeys(left, right []string) []string {
	seen := make(map[string]bool, len(left)+len(right))
	keys := make([]string, 0, len(left)+len(right))
	for _, key := range append(left, right...) {
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	return keys
}

func dependencyKeys(dependencies []*metadatapb.DependencyColumn) []string {
	keys := make([]string, 0, len(dependencies))
	for _, dependency := range dependencies {
		keys = append(keys, dependency.GetSchema()+"."+dependency.GetTable())
	}
	return keys
}

func splitObjectKey(key string) (string, string) {
	schemaName, objectName, _ := strings.Cut(key, ".")
	return schemaName, objectName
}

// restoreOrder puts each restore item after the selected items it depends on,
// so drops run in reverse and creates in dependency order.
type restoreItem struct {
	key, dropSQL, createSQL string
	oldDeps, newDeps        []string
}

func restoreOrder(items []restoreItem, deps func(restoreItem) []string) []int {
	selected := make(map[string]int, len(items))
	for i, item := range items {
		selected[item.key] = i
	}
	indegree := make([]int, len(items))
	dependents := make([][]int, len(items))
	for i, item := range items {
		seen := make(map[int]bool)
		for _, dep := range deps(item) {
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
