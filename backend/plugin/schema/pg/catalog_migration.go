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
	// $user expands to the session login, which this layer does not have, so it stays
	// unresolved and rejectAmbiguousUserTypes refuses the migrations it would decide.
	sourceSearchPath := getConfiguredSearchPath(oldSchema)
	targetSearchPath := getConfiguredSearchPath(newSchema)
	sourceDrops := sourceOnlyMissingDrops(oldSchema.GetProto(), newSchema.GetProto(), fromReport, toReport)
	if err := unloadableChanges(oldSchema.GetProto(), newSchema.GetProto(), sourceDrops, fromReport, toReport); err != nil {
		return "", err
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() && len(sourceDrops) == 0 {
		return "", nil
	}
	if err := rejectAmbiguousUserTypes(oldSchema.GetProto(), newSchema.GetProto(), sourceSearchPath, targetSearchPath); err != nil {
		return "", err
	}
	plan := catalog.GenerateMigration(from, to, diff)
	orderTypeAndTableCreates(plan, newSchema.GetProto(), targetSearchPath)
	restoreIndexDefinitions(plan, newSchema.GetProto())
	restored, err := restoreUnloadableDependentViews(plan, from, oldSchema.GetProto(), newSchema.GetProto(), fromReport, toReport)
	if err != nil {
		return "", err
	}
	if err := releaseCompositeBeforeTableDrop(plan, oldSchema.GetProto(), newSchema.GetProto(), restored, sourceSearchPath, targetSearchPath); err != nil {
		return "", err
	}
	restoreRecreatedViewComments(plan, newSchema.GetProto())
	plan.Ops = insertSourceOnlyDrops(plan.Ops, sourceDrops)
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

// orderTypeAndTableCreates puts a created type, or a created table whose row type
// a column uses, before the objects that use it, and a created table before a
// composite attribute that uses its row type.
func orderTypeAndTableCreates(plan *catalog.MigrationPlan, meta *metadatapb.DatabaseSchemaMetadata, searchPath []string) {
	type createNode struct {
		position int
		key      string
		relation bool // true for a table, false for a type
	}
	var nodes []createNode
	byKey := make(map[string]int)
	created := make(map[string]bool)
	for i, op := range plan.Ops {
		switch op.Type {
		case catalog.OpCreateTable:
			key := objectKey(op.SchemaName, op.ObjectName)
			byKey[key] = len(nodes)
			created[key] = true
			nodes = append(nodes, createNode{position: i, key: key, relation: true})
		case catalog.OpCreateType:
			key := objectKey(op.SchemaName, op.ObjectName)
			byKey[key] = len(nodes)
			created[key] = true
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
			tableIndex, ok := byKey[objectKey(schema.GetName(), table.GetName())]
			if !ok {
				continue
			}
			for _, column := range table.GetColumns() {
				key, ok := referencedRelation(column.GetType(), created, searchPath)
				if !ok {
					continue
				}
				addEdge(byKey[key], tableIndex)
			}
		}
		for _, composite := range schema.GetCompositeTypes() {
			compositeIndex, ok := byKey[objectKey(schema.GetName(), composite.GetName())]
			if !ok {
				continue
			}
			for _, attribute := range composite.GetAttributes() {
				key, ok := referencedRelation(attribute.GetType(), created, searchPath)
				if !ok {
					continue
				}
				addEdge(byKey[key], compositeIndex)
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
	reference, end, ok := parseTypeReferenceAt(strings.TrimSpace(typeName), 0)
	if !ok || strings.TrimSpace(trailingModifier(typeName[end:])) != "" {
		return false
	}
	schemaName, objectName := splitObjectKey(key)
	if reference.qualified && !reference.schema.matches(schemaName) {
		return false
	}
	return reference.object.matches(objectName)
}

// identifier is one component of a type name. A quoted component keeps its dots
// (`"a.b"."c"` is not `"a"."b.c"`) and PostgreSQL matches its case exactly; an
// unquoted one folds case.
type identifier struct {
	name   string
	quoted bool
}

func (i identifier) matches(stored string) bool {
	if i.quoted {
		return i.name == stored
	}
	// PostgreSQL folds an unquoted identifier to lower case before resolving it.
	return strings.ToLower(i.name) == stored
}

// typeReference is a parsed type name: object is the whole name unqualified.
type typeReference struct {
	schema, object identifier
	qualified      bool
}

// releaseCompositeBeforeTableDrop releases the composite attributes that hold the
// row type of a relation this migration drops, ahead of that relation's drop. A
// CASCADE drop would otherwise remove the attribute first.
//
// ADD ATTRIBUTE appends, so re-adding a released attribute after an attribute that
// stays in place would reorder the composite. Every target attribute from the first
// released position onward is therefore released and re-added in target order,
// after the creates their types need.
//
// restored holds every selected view whose create the plan defers, whether it is
// recreated or added by the target. A table column or a composite type that holds
// one of their row types cannot be ordered around that, so the migration is
// refused instead.
func releaseCompositeBeforeTableDrop(plan *catalog.MigrationPlan, oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, restored map[string]bool, sourceSearchPath, targetSearchPath []string) error {
	dropped := relationSet(oldMeta, newMeta)
	if len(dropped) == 0 && len(restored) == 0 {
		return nil
	}
	for key := range restored {
		dropped[key] = true
	}
	if err := refuseRestoredViewRowTypeDependents(oldMeta, newMeta, restored, sourceSearchPath, targetSearchPath); err != nil {
		return err
	}
	if err := refuseRestoredViewRoutines(oldMeta, newMeta, restored, sourceSearchPath, targetSearchPath); err != nil {
		return err
	}
	created := relationSet(newMeta, oldMeta)
	for _, schema := range oldMeta.GetSchemas() {
		newSchema := schemaByName(newMeta, schema.GetName())
		for _, composite := range schema.GetCompositeTypes() {
			newComposite := compositeByName(newSchema, composite.GetName())
			if newComposite == nil {
				continue
			}
			releaseCompositeAttributes(plan, schema.GetName(), composite, newComposite, dropped, created, restored, sourceSearchPath, targetSearchPath)
		}
	}
	return nil
}

// releaseCompositeAttributes handles one composite type. An attribute is
// re-added when its released type needs a create, or when its target type is a
// restored view's row type, which the catalog cannot order against the drop. The
// first such target position fixes where the re-added suffix starts.
func releaseCompositeAttributes(plan *catalog.MigrationPlan, schemaName string, oldComposite, newComposite *metadatapb.CompositeTypeMetadata, dropped, created, restored map[string]bool, sourceSearchPath, targetSearchPath []string) {
	compositeName := oldComposite.GetName()
	firstTarget := -1
	dropKeys := make(map[string]bool)
	for i, target := range newComposite.GetAttributes() {
		old := attributeByName(oldComposite, target.GetName())
		dropKey := ""
		if old != nil {
			dropKey, _ = referencedRelation(old.GetType(), dropped, sourceSearchPath)
		}
		restoredKey, restoredType := referencedRelation(target.GetType(), restored, targetSearchPath)
		if !restoredType && (dropKey == "" || !typeNeedsCreate(target.GetType(), created, restored, targetSearchPath)) {
			continue
		}
		if firstTarget < 0 {
			firstTarget = i
		}
		if dropKey != "" {
			dropKeys[dropKey] = true
		}
		if restoredType {
			dropKeys[restoredKey] = true
		}
	}
	// An attribute outside the re-added suffix keeps its position, so the
	// catalog's own release only has to move ahead of the relation drop. One the
	// target removes needs no re-add at all.
	for _, attribute := range oldComposite.GetAttributes() {
		dropKey, ok := referencedRelation(attribute.GetType(), dropped, sourceSearchPath)
		if !ok {
			continue
		}
		if position, _, kept := targetAttribute(newComposite, attribute.GetName()); kept && firstTarget >= 0 && position >= firstTarget {
			continue
		}
		releaseCatalogAttribute(plan, schemaName, compositeName, attribute.GetName(), dropKey)
	}
	if firstTarget < 0 {
		return
	}
	suffix := newComposite.GetAttributes()[firstTarget:]
	// A later suffix attribute can be released from a different relation than the
	// one that triggered the re-add, so all their drops bound the releases.
	for _, target := range suffix {
		old := attributeByName(oldComposite, target.GetName())
		if old == nil {
			continue
		}
		if dropKey, ok := referencedRelation(old.GetType(), dropped, sourceSearchPath); ok {
			dropKeys[dropKey] = true
		}
	}
	for _, target := range suffix {
		plan.Ops = removeCompositeAttributeOps(plan.Ops, schemaName, compositeName, target.GetName())
	}
	var releases []catalog.MigrationOp
	for _, target := range suffix {
		if attributeByName(oldComposite, target.GetName()) == nil {
			continue
		}
		releases = append(releases, catalog.MigrationOp{
			Type:       catalog.OpAlterType,
			SchemaName: schemaName,
			ObjectName: compositeName,
			SQL:        "ALTER TYPE " + quoteQualified(schemaName, compositeName) + " DROP ATTRIBUTE " + quoteIdentifier(target.GetName()),
			Phase:      catalog.PhasePre,
			ObjType:    'r',
		})
	}
	if len(releases) > 0 {
		drop := -1
		for key := range dropKeys {
			if index := findDropObject(plan, key); index >= 0 && (drop < 0 || index < drop) {
				drop = index
			}
		}
		// A view the target adds has no drop to precede; the release only has to
		// come before its deferred create.
		if drop < 0 {
			drop = 0
		}
		plan.Ops = slices.Insert(plan.Ops, drop, releases...)
	}
	anchor := -1
	for _, target := range suffix {
		for key := range createKeysFor(target.GetType(), created, restored, targetSearchPath) {
			if index := findCreateOp(plan.Ops, key); index > anchor {
				anchor = index
			}
		}
	}
	if anchor < 0 {
		return
	}
	adds := make([]catalog.MigrationOp, 0, len(suffix))
	for _, target := range suffix {
		adds = append(adds, catalog.MigrationOp{
			Type:       catalog.OpAlterType,
			SchemaName: schemaName,
			ObjectName: compositeName,
			SQL:        addAttributeSQL(schemaName, compositeName, target, target.GetType()),
			Phase:      catalog.PhaseMain,
			ObjType:    'r',
		})
	}
	plan.Ops = slices.Insert(plan.Ops, anchor+1, adds...)
	for _, target := range suffix {
		plan.Ops = restoreAttributeComment(plan.Ops, schemaName, compositeName, target)
	}
}

// releaseCatalogAttribute moves the catalog's own release of an attribute ahead of
// the drop of the relation whose row type it holds.
func releaseCatalogAttribute(plan *catalog.MigrationPlan, schemaName, compositeName, attributeName, dropKey string) {
	alter := findAlterAttribute(plan, schemaName, compositeName, attributeName)
	drop := findDropObject(plan, dropKey)
	if alter >= 0 && drop >= 0 && alter > drop {
		plan.Ops = moveOp(plan.Ops, alter, drop)
	}
}

// typeNeedsCreate reports whether a type name denotes an object this migration
// creates or recreates, so an attribute of it must be added after that create.
func typeNeedsCreate(typeName string, created, restored map[string]bool, searchPath []string) bool {
	if _, ok := referencedRelation(typeName, created, searchPath); ok {
		return true
	}
	_, ok := referencedRelation(typeName, restored, searchPath)
	return ok
}

// createKeysFor returns the objects whose create an attribute of the type needs to
// follow: a relation created in this migration, including a restored view.
func createKeysFor(typeName string, created, restored map[string]bool, searchPath []string) map[string]bool {
	keys := make(map[string]bool)
	if key, ok := referencedRelation(typeName, created, searchPath); ok {
		keys[key] = true
	}
	if key, ok := referencedRelation(typeName, restored, searchPath); ok {
		keys[key] = true
	}
	return keys
}

// refuseRestoredViewRowTypeDependents rejects an object that holds a restored
// view's row type and cannot be ordered around the view's drop. Releasing a table
// column would discard its data, and a created composite type would need the view
// before its CREATE TYPE. An existing composite's attributes are re-added after
// the recreation instead.
func refuseRestoredViewRowTypeDependents(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, restored map[string]bool, sourceSearchPath, targetSearchPath []string) error {
	if len(restored) == 0 {
		return nil
	}
	for _, meta := range []*metadatapb.DatabaseSchemaMetadata{oldMeta, newMeta} {
		searchPath := sourceSearchPath
		if meta == newMeta {
			searchPath = targetSearchPath
		}
		for _, schema := range meta.GetSchemas() {
			for _, table := range schema.GetTables() {
				for _, column := range table.GetColumns() {
					key, ok := referencedRelation(column.GetType(), restored, searchPath)
					if !ok {
						continue
					}
					return errors.Errorf("cannot generate a migration: column %s of table %s uses the row type of view %s, which this migration must create", quoteIdentifier(column.GetName()), quoteQualified(schema.GetName(), table.GetName()), quoteRestoredView(key))
				}
			}
		}
	}
	for _, schema := range newMeta.GetSchemas() {
		oldSchema := schemaByName(oldMeta, schema.GetName())
		for _, composite := range schema.GetCompositeTypes() {
			if compositeByName(oldSchema, composite.GetName()) != nil {
				continue
			}
			for _, attribute := range composite.GetAttributes() {
				key, ok := referencedRelation(attribute.GetType(), restored, targetSearchPath)
				if !ok {
					continue
				}
				return errors.Errorf("cannot generate a migration: attribute %s of type %s uses the row type of view %s, which this migration must create", quoteIdentifier(attribute.GetName()), quoteQualified(schema.GetName(), composite.GetName()), quoteRestoredView(key))
			}
		}
	}
	return nil
}

// refuseRestoredViewRoutines rejects a routine whose parameter or return type is a
// restored view's row type: the view cannot be dropped while the routine holds it,
// and the routine's body mentioning the view is not such a dependency.
func refuseRestoredViewRoutines(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, restored map[string]bool, sourceSearchPath, targetSearchPath []string) error {
	for _, meta := range []*metadatapb.DatabaseSchemaMetadata{oldMeta, newMeta} {
		searchPath := sourceSearchPath
		if meta == newMeta {
			searchPath = targetSearchPath
		}
		objects := typeObjects(meta)
		for _, schema := range meta.GetSchemas() {
			for _, function := range schema.GetFunctions() {
				if err := refuseRoutine(schema.GetName(), function.GetName(), function.GetSignature(), function.GetDefinition(), restored, objects, searchPath); err != nil {
					return err
				}
			}
			for _, procedure := range schema.GetProcedures() {
				if err := refuseRoutine(schema.GetName(), procedure.GetName(), procedure.GetSignature(), procedure.GetDefinition(), restored, objects, searchPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// typeObjects is every object a type name can resolve to, including a sequence's
// row type. An unqualified routine type follows search_path, so a same-named
// earlier object shadows a restored view instead of standing in for it.
func typeObjects(meta *metadatapb.DatabaseSchemaMetadata) map[string]bool {
	objects := make(map[string]bool)
	for _, schema := range meta.GetSchemas() {
		for _, table := range schema.GetTables() {
			objects[objectKey(schema.GetName(), table.GetName())] = true
		}
		for _, view := range schema.GetViews() {
			objects[objectKey(schema.GetName(), view.GetName())] = true
		}
		for _, view := range schema.GetMaterializedViews() {
			objects[objectKey(schema.GetName(), view.GetName())] = true
		}
		for _, enum := range schema.GetEnumTypes() {
			objects[objectKey(schema.GetName(), enum.GetName())] = true
		}
		for _, composite := range schema.GetCompositeTypes() {
			objects[objectKey(schema.GetName(), composite.GetName())] = true
		}
		for _, sequence := range schema.GetSequences() {
			objects[objectKey(schema.GetName(), sequence.GetName())] = true
		}
	}
	return objects
}

func refuseRoutine(schemaName, name, signature, definition string, restored, objects map[string]bool, searchPath []string) error {
	for key := range restored {
		if !parametersNameType(routineArgumentList(signature), key, objects, searchPath) &&
			!parametersNameType(routineArgumentList(definition), key, objects, searchPath) &&
			!routineResultNamesType(definition, key, objects, searchPath) {
			continue
		}
		return errors.Errorf("cannot generate a migration: routine %s uses the row type of view %s, which this migration must create", quoteQualified(schemaName, name), quoteRestoredView(key))
	}
	return nil
}

// parametersNameType reports whether an argument list declares the type as a
// parameter's type. A default expression is not a declaration, so it is cut.
func parametersNameType(parameters, key string, objects map[string]bool, searchPath []string) bool {
	depth, start := 0, 0
	for i := 0; i <= len(parameters); i++ {
		atEnd := i == len(parameters)
		if !atEnd {
			switch parameters[i] {
			case '"', '\'':
				i = closingQuote(parameters, i, parameters[i])
				continue
			case '$':
				if end, ok := dollarQuoteEnd(parameters, i); ok {
					i = end
				}
				continue
			case '(':
				depth++
				continue
			case ')':
				depth--
				continue
			case ',':
				if depth != 0 {
					continue
				}
			default:
				continue
			}
		}
		declaration := parameters[start:i]
		if index := indexOutsideQuotes(declaration, "DEFAULT"); index >= 0 {
			declaration = declaration[:index]
		}
		if index := indexOutsideQuotes(declaration, "="); index >= 0 {
			declaration = declaration[:index]
		}
		if namesType(declaration, key, objects, searchPath) {
			return true
		}
		start = i + 1
	}
	return false
}

// routineArgumentList is the parenthesized parameter list of a routine's
// definition, which also names an OUT parameter's type; the signature carries
// input arguments only.
func routineArgumentList(definition string) string {
	open := indexOutsideQuotes(definition, "(")
	if open < 0 {
		return ""
	}
	depth := 0
	for i := open; i < len(definition); i++ {
		switch definition[i] {
		case '"', '\'':
			i = closingQuote(definition, i, definition[i])
		case '$':
			if end, ok := dollarQuoteEnd(definition, i); ok {
				i = end
			}
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return definition[open+1 : i]
			}
		default:
		}
	}
	return ""
}

// closingQuote returns the index of the quote closing the identifier or literal
// that starts at start, where a doubled quote is an escape.
func closingQuote(text string, start int, quote byte) int {
	escaped := quote == '\'' && start > 0 && (text[start-1] == 'E' || text[start-1] == 'e') &&
		(start-1 == 0 || !isIdentifierByte(text[start-2]))
	for i := start + 1; i < len(text); i++ {
		if escaped && text[i] == '\\' {
			i++
			continue
		}
		if text[i] != quote {
			continue
		}
		if i+1 < len(text) && text[i+1] == quote {
			i++
			continue
		}
		return i
	}
	return len(text) - 1
}

// dollarQuoteEnd returns the index of the closing tag of the dollar-quoted string
// starting at start. It reports false when start does not begin one, as with a
// $1 parameter placeholder.
func dollarQuoteEnd(text string, start int) (int, bool) {
	if start >= len(text) || text[start] != '$' {
		return 0, false
	}
	i := start + 1
	if i < len(text) && text[i] != '$' {
		if text[i] != '_' && (text[i] < 'a' || text[i] > 'z') && (text[i] < 'A' || text[i] > 'Z') {
			return 0, false
		}
		i++
		for i < len(text) && text[i] != '$' && isIdentifierByte(text[i]) {
			i++
		}
	}
	if i >= len(text) || text[i] != '$' {
		return 0, false
	}
	tag := text[start : i+1]
	closing := strings.Index(text[i+1:], tag)
	if closing < 0 {
		return len(text) - 1, true
	}
	return i + closing + len(tag), true
}

// routineResultNamesType reports whether a routine's result declaration names the
// type: its RETURNS clause, or the columns of RETURNS TABLE.
func routineResultNamesType(definition, key string, objects map[string]bool, searchPath []string) bool {
	clause := routineReturnClause(definition)
	if strings.HasPrefix(clause, "TABLE") {
		return parametersNameType(routineArgumentList(clause), key, objects, searchPath)
	}
	return namesType(clause, key, objects, searchPath)
}

// routineReturnClause is the text between RETURNS and the first clause after it,
// which is where a routine's result type is declared.
func routineReturnClause(definition string) string {
	index := indexOutsideQuotes(definition, "RETURNS ")
	if index < 0 {
		return ""
	}
	rest := definition[index+len("RETURNS "):]
	end := len(rest)
	for _, marker := range []string{"\nAS ", "\n AS ", " LANGUAGE ", "\nLANGUAGE ", " IMMUTABLE", " STABLE", " VOLATILE", " STRICT", " SECURITY ", " COST ", " ROWS ", " WINDOW ", "\n"} {
		if i := indexOutsideQuotes(rest, marker); i >= 0 && i < end {
			end = i
		}
	}
	return rest[:end]
}

// namesType reports whether text ends in a declaration of the relation's type.
// The trailing type is parsed, so a partly quoted reference like `public."V"`
// matches as well as a fully quoted, unqualified, or unquoted one. An unqualified
// name is resolved first, so it names only the relation the search path makes
// visible, not every relation that shares its name.
func namesType(text, key string, objects map[string]bool, searchPath []string) bool {
	text = strings.TrimSpace(text)
	if builtinMultiwordType(text) {
		return false
	}
	for start := 0; start < len(text); start++ {
		if start > 0 && !startsDeclaration(text[start-1]) {
			continue
		}
		reference, end, ok := parseTypeReferenceAt(text, start)
		if !ok || strings.TrimSpace(trailingModifier(text[end:])) != "" {
			continue
		}
		if resolved, ok := resolveTypeReference(reference, objects, searchPath); ok && resolved == key {
			return true
		}
	}
	return false
}

// builtinMultiwordType reports whether text declares a built-in type spelled with
// several words. Its last word is part of that type, not a reference of its own.
func builtinMultiwordType(text string) bool {
	for _, prefix := range []string{"SETOF ", "VARIADIC "} {
		text = strings.TrimSpace(strings.TrimPrefix(text, prefix))
	}
	if index := strings.LastIndex(text, "("); index >= 0 && strings.HasSuffix(text, ")") {
		text = strings.TrimSpace(text[:index])
	}
	for _, multiword := range []string{
		"double precision",
		"character varying",
		"bit varying",
		"time with time zone",
		"time without time zone",
		"timestamp with time zone",
		"timestamp without time zone",
	} {
		if text == multiword || strings.HasSuffix(text, " "+multiword) {
			return true
		}
	}
	return false
}

// trailingModifier drops a type modifier such as `(10,2)` that follows the type.
func trailingModifier(rest string) string {
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "(") && strings.HasSuffix(rest, ")") {
		return ""
	}
	return rest
}

// parseTypeReferenceAt parses a type name starting at start and reports where it
// ends, including any array marker.
func parseTypeReferenceAt(text string, start int) (typeReference, int, bool) {
	first, next, firstQuoted, ok := scanIdentifier(text, start)
	if !ok {
		return typeReference{}, start, false
	}
	if next >= len(text) || text[next] != '.' {
		end := next
		for end+1 < len(text) && text[end] == '[' && text[end+1] == ']' {
			end += 2
		}
		return typeReference{object: identifier{name: first, quoted: firstQuoted}}, end, true
	}
	second, next, secondQuoted, ok := scanIdentifier(text, next+1)
	if !ok {
		return typeReference{}, start, false
	}
	end := next
	for end+1 < len(text) && text[end] == '[' && text[end+1] == ']' {
		end += 2
	}
	return typeReference{schema: identifier{name: first, quoted: firstQuoted}, object: identifier{name: second, quoted: secondQuoted}, qualified: true}, end, true
}

// startsDeclaration reports whether b can precede a type name in a declaration.
func startsDeclaration(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '(', ',', ';', '[':
		return true
	default:
		return false
	}
}

func quoteRestoredView(key string) string {
	schemaName, objectName := splitObjectKey(key)
	return quoteQualified(schemaName, objectName)
}

// removeCompositeAttributeOps drops the catalog's operations on one composite
// attribute, which the release pass rewrites itself.
func removeCompositeAttributeOps(ops []catalog.MigrationOp, schemaName, compositeName, attributeName string) []catalog.MigrationOp {
	reference := quoteIdentifier(attributeName)
	return slices.DeleteFunc(ops, func(op catalog.MigrationOp) bool {
		switch op.Type {
		case catalog.OpAlterType:
			return op.SchemaName == schemaName && op.ObjectName == compositeName && strings.Contains(op.SQL, "ATTRIBUTE "+reference)
		case catalog.OpComment:
			target, ok := parseCommentTarget(op.SQL)
			return ok && target.kind == "COLUMN" && len(target.parts) == 3 &&
				target.parts[0] == schemaName && target.parts[1] == compositeName && target.parts[2] == attributeName
		default:
			return false
		}
	})
}

func findCreateOp(ops []catalog.MigrationOp, key string) int {
	for i, op := range ops {
		switch op.Type {
		case catalog.OpCreateTable, catalog.OpCreateView, catalog.OpCreateType:
			if objectKey(op.SchemaName, op.ObjectName) == key {
				return i
			}
		default:
		}
	}
	return -1
}

func targetAttribute(composite *metadatapb.CompositeTypeMetadata, name string) (int, *metadatapb.CompositeTypeAttribute, bool) {
	for i, attribute := range composite.GetAttributes() {
		if attribute.GetName() == name {
			return i, attribute, true
		}
	}
	return 0, nil, false
}

func attributeByName(composite *metadatapb.CompositeTypeMetadata, name string) *metadatapb.CompositeTypeAttribute {
	_, attribute, _ := targetAttribute(composite, name)
	return attribute
}

func restoreAttributeComment(ops []catalog.MigrationOp, schema, composite string, attribute *metadatapb.CompositeTypeAttribute) []catalog.MigrationOp {
	if attribute.GetComment() == "" {
		return ops
	}
	target := quoteQualified(schema, composite) + "." + quoteIdentifier(attribute.GetName())
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
			return slices.Insert(ops, i+1, comment)
		}
	}
	return append(ops, comment)
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
				return indexInTable(table, name)
			}
			if index := partitionIndexByName(table.GetPartitions(), ownerSchema, ownerObject, ownerSchema, name); index != nil {
				return index
			}
		}
		for _, view := range schema.GetMaterializedViews() {
			if view.GetName() == ownerObject {
				return findIndex(view.GetIndexes(), name)
			}
		}
		// A partition keeps its metadata under a table that can live in another
		// schema, so the owner's schema alone does not find it.
		for _, parent := range meta.GetSchemas() {
			for _, table := range parent.GetTables() {
				if index := partitionIndexByName(table.GetPartitions(), parent.GetName(), ownerObject, ownerSchema, name); index != nil {
					return index
				}
			}
		}
		return nil
	}
	schema := schemaByName(meta, fallbackSchema)
	if schema == nil {
		return nil
	}
	for _, table := range schema.GetTables() {
		if index := indexInTable(table, name); index != nil {
			return index
		}
	}
	for _, view := range schema.GetMaterializedViews() {
		if index := findIndex(view.GetIndexes(), name); index != nil {
			return index
		}
	}
	// A partition lives under its parent table, which can be in any schema.
	for _, parent := range meta.GetSchemas() {
		for _, table := range parent.GetTables() {
			if index := indexInTable(table, name); index != nil {
				return index
			}
		}
	}
	return nil
}

// indexInTable returns the index of that name on a table or one of its partitions.
func indexInTable(table *metadatapb.TableMetadata, name string) *metadatapb.IndexMetadata {
	if index := findIndex(table.GetIndexes(), name); index != nil {
		return index
	}
	var found *metadatapb.IndexMetadata
	forEachPartition(table.GetPartitions(), func(partition *metadatapb.TablePartitionMetadata) {
		if found == nil {
			found = findIndex(partition.GetIndexes(), name)
		}
	})
	return found
}

// partitionExists reports whether a snapshot holds a partition of that name in
// that schema, wherever its parent table lives.
func partitionExists(meta *metadatapb.DatabaseSchemaMetadata, schemaName, partitionName string) bool {
	found := false
	for _, schema := range meta.GetSchemas() {
		for _, table := range schema.GetTables() {
			forEachPartition(table.GetPartitions(), func(partition *metadatapb.TablePartitionMetadata) {
				partitionSchema := partition.GetSchema()
				if partitionSchema == "" {
					partitionSchema = schema.GetName()
				}
				if partitionSchema == schemaName && partition.GetName() == partitionName {
					found = true
				}
			})
		}
	}
	return found
}

// partitionIndexByName returns the index of that name on the partition named
// partitionName or one of its subpartitions. A partition without its own schema
// keeps its parent's. An empty requiredSchema accepts any schema.
func partitionIndexByName(partitions []*metadatapb.TablePartitionMetadata, parentSchema, partitionName, requiredSchema, name string) *metadatapb.IndexMetadata {
	var found *metadatapb.IndexMetadata
	forEachPartition(partitions, func(partition *metadatapb.TablePartitionMetadata) {
		if found != nil || partition.GetName() != partitionName {
			return
		}
		partitionSchema := partition.GetSchema()
		if partitionSchema == "" {
			partitionSchema = parentSchema
		}
		if requiredSchema != "" && partitionSchema != requiredSchema {
			return
		}
		found = findIndex(partition.GetIndexes(), name)
	})
	return found
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
	operand := "ATTRIBUTE " + quoteIdentifier(attribute)
	for i, op := range plan.Ops {
		if op.Type == catalog.OpAlterType && op.SchemaName == schema && op.ObjectName == composite && strings.Contains(op.SQL, operand) {
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
		if objectKey(op.SchemaName, op.ObjectName) == key {
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
				result[objectKey(schema.GetName(), table.GetName())] = true
			}
			forEachPartition(table.GetPartitions(), func(partition *metadatapb.TablePartitionMetadata) {
				partitionSchema := partition.GetSchema()
				if partitionSchema == "" {
					partitionSchema = schema.GetName()
				}
				if !partitionExists(right, partitionSchema, partition.GetName()) {
					result[objectKey(partitionSchema, partition.GetName())] = true
				}
			})
		}
		for _, view := range schema.GetViews() {
			if other == nil || viewByName(other, view.GetName()) == nil {
				result[objectKey(schema.GetName(), view.GetName())] = true
			}
		}
		for _, view := range schema.GetMaterializedViews() {
			if other == nil || materializedViewByName(other, view.GetName()) == nil {
				result[objectKey(schema.GetName(), view.GetName())] = true
			}
		}
		for _, enum := range schema.GetEnumTypes() {
			if other == nil || enumByName(other, enum.GetName()) == nil {
				result[objectKey(schema.GetName(), enum.GetName())] = true
			}
		}
		for _, composite := range schema.GetCompositeTypes() {
			if other == nil || compositeByName(other, composite.GetName()) == nil {
				result[objectKey(schema.GetName(), composite.GetName())] = true
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

func isIdentifierByte(b byte) bool {
	return b == '_' || b == '$' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

func referencedRelation(typeName string, relations map[string]bool, searchPath []string) (string, bool) {
	name := strings.TrimSpace(typeName)
	reference, end, ok := parseTypeReferenceAt(name, 0)
	if !ok || strings.TrimSpace(trailingModifier(name[end:])) != "" {
		return "", false
	}
	return resolveTypeReference(reference, relations, searchPath)
}

// rejectAmbiguousUserTypes refuses a migration whose unqualified type depends on
// an unresolved $user entry: no known schema before it names the object, and some
// relation does, so whether the user's schema shadows that relation is unknown.
func rejectAmbiguousUserTypes(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, sourceSearchPath, targetSearchPath []string) error {
	if !slices.Contains(sourceSearchPath, "$user") && !slices.Contains(targetSearchPath, "$user") {
		return nil
	}
	for _, meta := range []*metadatapb.DatabaseSchemaMetadata{oldMeta, newMeta} {
		searchPath := sourceSearchPath
		if meta == newMeta {
			searchPath = targetSearchPath
		}
		objects := typeObjects(meta)
		for _, schema := range meta.GetSchemas() {
			for _, table := range schema.GetTables() {
				for _, column := range table.GetColumns() {
					if ambiguousTypeText(column.GetType(), objects, searchPath) {
						return errors.Errorf("cannot generate a migration: column %s of table %s has type %s, whose $user search-path resolution is ambiguous", quoteIdentifier(column.GetName()), quoteQualified(schema.GetName(), table.GetName()), column.GetType())
					}
				}
			}
			for _, composite := range schema.GetCompositeTypes() {
				for _, attribute := range composite.GetAttributes() {
					if ambiguousTypeText(attribute.GetType(), objects, searchPath) {
						return errors.Errorf("cannot generate a migration: attribute %s of type %s has type %s, whose $user search-path resolution is ambiguous", quoteIdentifier(attribute.GetName()), quoteQualified(schema.GetName(), composite.GetName()), attribute.GetType())
					}
				}
			}
			for _, function := range schema.GetFunctions() {
				if ambiguousRoutineTypes(function.GetSignature(), function.GetDefinition(), objects, searchPath) {
					return errors.Errorf("cannot generate a migration: routine %s has a type whose $user search-path resolution is ambiguous", quoteQualified(schema.GetName(), function.GetName()))
				}
			}
			for _, procedure := range schema.GetProcedures() {
				if ambiguousRoutineTypes(procedure.GetSignature(), procedure.GetDefinition(), objects, searchPath) {
					return errors.Errorf("cannot generate a migration: routine %s has a type whose $user search-path resolution is ambiguous", quoteQualified(schema.GetName(), procedure.GetName()))
				}
			}
		}
	}
	return nil
}

func ambiguousRoutineTypes(signature, definition string, objects map[string]bool, searchPath []string) bool {
	if ambiguousParameters(routineArgumentList(signature), objects, searchPath) || ambiguousParameters(routineArgumentList(definition), objects, searchPath) {
		return true
	}
	clause := routineReturnClause(definition)
	if strings.HasPrefix(clause, "TABLE") {
		return ambiguousParameters(routineArgumentList(clause), objects, searchPath)
	}
	return ambiguousTypeText(clause, objects, searchPath)
}

func ambiguousParameters(parameters string, objects map[string]bool, searchPath []string) bool {
	depth, start := 0, 0
	for i := 0; i <= len(parameters); i++ {
		atEnd := i == len(parameters)
		if !atEnd {
			switch parameters[i] {
			case '"', '\'':
				i = closingQuote(parameters, i, parameters[i])
				continue
			case '$':
				if end, ok := dollarQuoteEnd(parameters, i); ok {
					i = end
				}
				continue
			case '(':
				depth++
				continue
			case ')':
				depth--
				continue
			case ',':
				if depth != 0 {
					continue
				}
			default:
				continue
			}
		}
		declaration := parameters[start:i]
		if index := indexOutsideQuotes(declaration, "DEFAULT"); index >= 0 {
			declaration = declaration[:index]
		}
		if index := indexOutsideQuotes(declaration, "="); index >= 0 {
			declaration = declaration[:index]
		}
		if ambiguousTypeText(declaration, objects, searchPath) {
			return true
		}
		start = i + 1
	}
	return false
}

func ambiguousTypeText(text string, objects map[string]bool, searchPath []string) bool {
	text = strings.TrimSpace(text)
	for start := 0; start < len(text); start++ {
		if start > 0 && !startsDeclaration(text[start-1]) {
			continue
		}
		reference, end, ok := parseTypeReferenceAt(text, start)
		if !ok || strings.TrimSpace(trailingModifier(text[end:])) != "" {
			continue
		}
		if ambiguousSearchPathType(reference, objects, searchPath) {
			return true
		}
	}
	return false
}

func ambiguousSearchPathType(reference typeReference, relations map[string]bool, searchPath []string) bool {
	if reference.qualified {
		return false
	}
	userIndex := -1
	for i, schema := range searchPath {
		if schema == "$user" {
			userIndex = i
			break
		}
	}
	if userIndex < 0 {
		return false
	}
	for _, searchSchema := range searchPath[:userIndex] {
		for key := range relations {
			schemaName, objectName := splitObjectKey(key)
			if schemaName == searchSchema && reference.object.matches(objectName) {
				return false
			}
		}
	}
	for key := range relations {
		if _, objectName := splitObjectKey(key); reference.object.matches(objectName) {
			return true
		}
	}
	return false
}

// resolveTypeReference resolves a parsed type name to the object it names. An
// unqualified name resolves to the first relation in search-path order; an empty
// path is PostgreSQL's default, which reaches public after any current-user schema.
func resolveTypeReference(reference typeReference, relations map[string]bool, searchPath []string) (string, bool) {
	if reference.qualified {
		for key := range relations {
			schemaName, objectName := splitObjectKey(key)
			if reference.schema.matches(schemaName) && reference.object.matches(objectName) {
				return key, true
			}
		}
		return "", false
	}
	if len(searchPath) == 0 {
		searchPath = []string{"public"}
	}
	for _, searchSchema := range searchPath {
		if searchSchema == "$user" {
			// The caller resolves $user with the database owner. An unresolved
			// placeholder cannot be guessed from schema names, and public is not a
			// safe answer because the user's schema may shadow it.
			return "", false
		}
		for key := range relations {
			schemaName, objectName := splitObjectKey(key)
			if schemaName == searchSchema && reference.object.matches(objectName) {
				return key, true
			}
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

// restoreUnloadableDependentViews recreates a view the catalog could not parse
// when a column it depends on is dropped or retyped. The unparsed view is a
// stand-in on both sides, so the catalog has no dependency edge and would alter
// the column under a view that must be dropped first. The snapshot's
// DependencyColumns is the only record of that edge. It returns the views whose
// creation the plan defers, which releaseCompositeBeforeTableDrop treats as
// recreated relations; a view the target adds is one of them.
func restoreUnloadableDependentViews(plan *catalog.MigrationPlan, from *catalog.Catalog, oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, reports ...*catalog.LoadMetadataReport) (map[string]bool, error) {
	affected := affectedTableColumns(oldMeta, newMeta)
	if len(affected) == 0 {
		return nil, nil
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
		drop, create            bool
	}
	var candidates []candidate
	for _, schema := range newMeta.GetSchemas() {
		oldSchema := schemaByName(oldMeta, schema.GetName())
		for _, view := range schema.GetViews() {
			key := objectKey(schema.GetName(), view.GetName())
			oldView := viewByName(oldSchema, view.GetName())
			createSQL, err := catalog.ViewDefinition(schema.GetName(), view)
			if err != nil {
				continue
			}
			entry := candidate{
				key:        key,
				dropSQL:    "DROP VIEW IF EXISTS " + quoteQualified(schema.GetName(), view.GetName()),
				createSQL:  trimDefinition(createSQL),
				newDeps:    dependencyKeys(view.GetDependencyColumns()),
				unloadable: unloadable[reportedRelKey(key)],
				create:     true,
			}
			if oldView != nil {
				entry.oldDeps = dependencyKeys(oldView.GetDependencyColumns())
				entry.drop = true
				entry.direct = dependsOnAffectedColumn(view.GetDependencyColumns(), affected) || dependsOnAffectedColumn(oldView.GetDependencyColumns(), affected)
			}
			candidates = append(candidates, entry)
		}
		for _, view := range schema.GetMaterializedViews() {
			key := objectKey(schema.GetName(), view.GetName())
			oldView := materializedViewByName(oldSchema, view.GetName())
			createSQL, err := catalog.MaterializedViewDefinition(schema.GetName(), view)
			if err != nil {
				continue
			}
			entry := candidate{
				key:        key,
				dropSQL:    "DROP MATERIALIZED VIEW IF EXISTS " + quoteQualified(schema.GetName(), view.GetName()),
				createSQL:  trimDefinition(createSQL),
				newDeps:    dependencyKeys(view.GetDependencyColumns()),
				unloadable: unloadable[reportedRelKey(key)],
				create:     true,
			}
			if oldView != nil {
				entry.oldDeps = dependencyKeys(oldView.GetDependencyColumns())
				entry.drop = true
				entry.direct = dependsOnAffectedColumn(view.GetDependencyColumns(), affected) || dependsOnAffectedColumn(oldView.GetDependencyColumns(), affected)
			}
			candidates = append(candidates, entry)
		}
	}
	for _, oldSchema := range oldMeta.GetSchemas() {
		newSchema := schemaByName(newMeta, oldSchema.GetName())
		for _, view := range oldSchema.GetViews() {
			if viewByName(newSchema, view.GetName()) != nil {
				continue
			}
			candidates = append(candidates, candidate{
				key:     objectKey(oldSchema.GetName(), view.GetName()),
				dropSQL: "DROP VIEW IF EXISTS " + quoteQualified(oldSchema.GetName(), view.GetName()),
				oldDeps: dependencyKeys(view.GetDependencyColumns()),
				drop:    true,
			})
		}
		for _, view := range oldSchema.GetMaterializedViews() {
			if materializedViewByName(newSchema, view.GetName()) != nil {
				continue
			}
			candidates = append(candidates, candidate{
				key:     objectKey(oldSchema.GetName(), view.GetName()),
				dropSQL: "DROP MATERIALIZED VIEW IF EXISTS " + quoteQualified(oldSchema.GetName(), view.GetName()),
				oldDeps: dependencyKeys(view.GetDependencyColumns()),
				drop:    true,
			})
		}
	}
	byKey := make(map[string]candidate, len(candidates))
	for _, candidate := range candidates {
		byKey[candidate.key] = candidate
	}
	reachesAffected := func(candidate candidate) bool {
		seen := make(map[string]bool)
		var visit func(key string) bool
		visit = func(key string) bool {
			if seen[key] {
				return false
			}
			seen[key] = true
			if _, ok := affected[key]; ok {
				return true
			}
			dep, ok := byKey[key]
			if !ok {
				return false
			}
			for _, dependency := range unionKeys(dep.oldDeps, dep.newDeps) {
				if visit(dependency) {
					return true
				}
			}
			return false
		}
		for _, dependency := range unionKeys(candidate.oldDeps, candidate.newDeps) {
			if visit(dependency) {
				return true
			}
		}
		return false
	}
	selected := make(map[string]bool)
	for _, candidate := range candidates {
		if candidate.unloadable && (candidate.direct || reachesAffected(candidate)) {
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
		if schemaName, _ := splitObjectKey(key); schemaName == "bbdataarchive" {
			return nil, errors.Errorf("cannot generate a migration: backup-schema view %s depends on an object that must be recreated", key)
		}
	}
	if err := refuseAmbiguousIndexComments(plan.Ops, selected, oldMeta, newMeta); err != nil {
		return nil, err
	}
	var items []restoreItem
	for _, candidate := range candidates {
		if selected[candidate.key] {
			items = append(items, restoreItem{key: candidate.key, dropSQL: candidate.dropSQL, createSQL: candidate.createSQL, oldDeps: candidate.oldDeps, newDeps: candidate.newDeps, drop: candidate.drop, create: candidate.create})
		}
	}
	if len(items) == 0 {
		return nil, nil
	}
	dropOrder := restoreOrder(items, func(item restoreItem) []string { return item.oldDeps })
	createOrder := restoreOrder(items, func(item restoreItem) []string { return item.newDeps })
	dropped := make(map[string]bool, len(items))
	restored := make(map[string]bool, len(items))
	for _, item := range items {
		dropped[item.key] = true
		if item.create {
			restored[item.key] = true
		}
	}
	plan.Ops = slices.DeleteFunc(plan.Ops, func(op catalog.MigrationOp) bool {
		return opTargetsSelected(op, dropped, oldMeta, newMeta, from)
	})
	ops := make([]catalog.MigrationOp, 0, len(plan.Ops)+len(items)*2)
	for i := len(dropOrder) - 1; i >= 0; i-- {
		item := items[dropOrder[i]]
		if !item.drop {
			continue
		}
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
		if !item.create {
			continue
		}
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
	return restored, nil
}

// refuseAmbiguousIndexComments rejects a comment on an index name that several of
// a schema's relations own while one of them is selected: the comment cannot be
// attributed, and leaving it in the plan runs it against the wrong index.
func refuseAmbiguousIndexComments(ops []catalog.MigrationOp, selected map[string]bool, oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata) error {
	for _, op := range ops {
		if op.Type != catalog.OpComment {
			continue
		}
		target, ok := parseCommentTarget(op.SQL)
		if !ok || target.kind != "INDEX" || len(target.parts) != 2 {
			continue
		}
		owners := make(map[string]bool)
		for _, owner := range append(indexNameOwners(oldMeta, target.parts[0], target.parts[1]), indexNameOwners(newMeta, target.parts[0], target.parts[1])...) {
			owners[owner] = true
		}
		if len(owners) < 2 {
			continue
		}
		for owner := range owners {
			if !selected[owner] {
				continue
			}
			schemaName, objectName := splitObjectKey(owner)
			return errors.Errorf("cannot generate a migration: index %s has a comment but its name is used by more than one relation, and %s must be recreated", quoteQualified(target.parts[0], target.parts[1]), quoteQualified(schemaName, objectName))
		}
	}
	return nil
}

// opTargetsSelected reports whether an operation belongs to a selected restored
// object. Its accessories (comments, indexes, triggers) are omitted because the
// synthesized create from metadata already carries them. An accessory is matched
// through its owner, since a trigger or index name alone can equal a view's.
func opTargetsSelected(op catalog.MigrationOp, selected map[string]bool, oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, from *catalog.Catalog) bool {
	switch op.Type {
	case catalog.OpComment:
		return commentTargetsSelected(op.SQL, selected, oldMeta, newMeta)
	case catalog.OpCreateView, catalog.OpAlterView, catalog.OpDropView:
		return selected[objectKey(op.SchemaName, op.ObjectName)]
	case catalog.OpCreateIndex, catalog.OpDropIndex:
		if ownerSchema, ownerObject := indexOwnerFromSQL(op.SQL); ownerSchema != "" {
			return selected[objectKey(ownerSchema, ownerObject)]
		}
		// A DROP INDEX statement names only the schema and index, and CockroachDB
		// allows one index name on several of a schema's relations.
		if op.Type == catalog.OpDropIndex && from != nil {
			if index := from.GetIndexByOID(op.ObjOID); index != nil {
				if relation := from.GetRelationByOID(index.RelOID); relation != nil && relation.Schema != nil {
					return selected[objectKey(relation.Schema.Name, relation.Name)]
				}
			}
		}
		return indexNameOwner(oldMeta, op.SchemaName, op.ObjectName, selected) != ""
	case catalog.OpCreateTrigger, catalog.OpDropTrigger:
		// A trigger's owner is its table, which the operation carries; its
		// statement also names the function it calls.
		return selected[objectKey(op.SchemaName, op.ParentObject)]
	default:
	}
	return false
}

// commentTarget is a parsed COMMENT ON target: the object kind, its identifiers,
// and for a trigger or policy the relation after ON.
type commentTarget struct {
	kind  string
	parts []string
	on    []string
}

// parseCommentTarget reads a COMMENT ON statement's target instead of searching
// its text, so a comment's value cannot pass for another object's name.
func parseCommentTarget(sql string) (commentTarget, bool) {
	rest, ok := strings.CutPrefix(sql, "COMMENT ON ")
	if !ok {
		return commentTarget{}, false
	}
	var target commentTarget
	for _, kind := range []string{"MATERIALIZED VIEW", "FOREIGN TABLE", "TABLE", "VIEW", "COLUMN", "INDEX", "TRIGGER"} {
		if after, found := strings.CutPrefix(rest, kind+" "); found {
			target.kind, rest = kind, after
			break
		}
	}
	if target.kind == "" {
		return commentTarget{}, false
	}
	parts, rest, ok := scanIdentifiers(rest)
	if !ok {
		return commentTarget{}, false
	}
	target.parts = parts
	rest, found := strings.CutPrefix(rest, "ON ")
	if !found {
		return target, true
	}
	on, _, ok := scanIdentifiers(rest)
	if !ok {
		return commentTarget{}, false
	}
	target.on = on
	return target, true
}

// scanIdentifiers reads the dotted identifiers at the start of text and returns
// the remainder, which begins at the first space.
func scanIdentifiers(text string) ([]string, string, bool) {
	var parts []string
	for {
		name, next, _, ok := scanIdentifier(text, 0)
		if !ok {
			return nil, "", false
		}
		parts = append(parts, name)
		if next < len(text) && text[next] == '.' {
			text = text[next+1:]
			continue
		}
		return parts, strings.TrimSpace(text[next:]), true
	}
}

// commentTargetsSelected reports whether a COMMENT ON statement belongs to a
// selected relation. Its index and trigger comments do too: the synthesized create
// from metadata carries them.
func commentTargetsSelected(sql string, selected map[string]bool, oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata) bool {
	target, ok := parseCommentTarget(sql)
	if !ok {
		return false
	}
	switch target.kind {
	case "INDEX":
		if len(target.parts) != 2 {
			return false
		}
		// A create-only selected relation has its index only in the target.
		return indexNameOwner(oldMeta, target.parts[0], target.parts[1], selected) != "" ||
			indexNameOwner(newMeta, target.parts[0], target.parts[1], selected) != ""
	case "TRIGGER":
		if len(target.on) != 2 {
			return false
		}
		return selected[objectKey(target.on[0], target.on[1])]
	default:
		if len(target.parts) < 2 {
			return false
		}
		return selected[objectKey(target.parts[0], target.parts[1])]
	}
}

func indexOwnerFromSQL(sql string) (string, string) {
	index := indexOutsideQuotes(sql, " ON ")
	if index < 0 {
		return "", ""
	}
	rest := strings.TrimSpace(sql[index+len(" ON "):])
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

// scanIdentifier is quotedIdentifier plus whether the identifier was quoted.
func scanIdentifier(text string, start int) (string, int, bool, bool) {
	name, next, ok := quotedIdentifier(text, start)
	if !ok {
		return "", start, false, false
	}
	return name, next, start < len(text) && text[start] == '"', true
}

// indexOutsideQuotes is strings.Index for needle, skipping quoted identifiers, so
// an index named `"x ON y"` does not look like the owner clause.
func indexOutsideQuotes(text, needle string) int {
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '"', '\'':
			i = closingQuote(text, i, text[i])
			continue
		case '$':
			if end, ok := dollarQuoteEnd(text, i); ok {
				i = end
				continue
			}
		default:
		}
		if strings.HasPrefix(text[i:], needle) {
			return i
		}
	}
	return -1
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
	owners := indexNameOwners(meta, schemaName, indexName)
	if len(owners) != 1 || !selected[owners[0]] {
		return ""
	}
	return owners[0]
}

// indexNameOwners returns the relations of a schema that own an index name.
// CockroachDB allows the same name on several, unlike PostgreSQL.
func indexNameOwners(meta *metadatapb.DatabaseSchemaMetadata, schemaName, indexName string) []string {
	schema := schemaByName(meta, schemaName)
	if schema == nil {
		return nil
	}
	var owners []string
	add := func(name string, indexes []*metadatapb.IndexMetadata) {
		for _, index := range indexes {
			if index.GetName() == indexName {
				owners = append(owners, objectKey(schemaName, name))
				return
			}
		}
	}
	for _, table := range schema.GetTables() {
		add(table.GetName(), table.GetIndexes())
	}
	for _, view := range schema.GetMaterializedViews() {
		add(view.GetName(), view.GetIndexes())
	}
	return owners
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
		keys = append(keys, objectKey(dependency.GetSchema(), dependency.GetTable()))
	}
	return keys
}

func reportedRelKey(key string) string {
	schemaName, objectName := splitObjectKey(key)
	return "rel:" + schemaName + "." + objectName
}

func objectKey(schemaName, objectName string) string {
	return schemaName + "\x00" + objectName
}

func splitObjectKey(key string) (string, string) {
	schemaName, objectName, _ := strings.Cut(key, "\x00")
	return schemaName, objectName
}

// restoreOrder puts each restore item after the selected items it depends on,
// so drops run in reverse and creates in dependency order.
type restoreItem struct {
	key, dropSQL, createSQL string
	oldDeps, newDeps        []string
	drop, create            bool
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
				// A dependent view sees a column's type and collation. Its other
				// properties (default, nullability, comment, position, identity and
				// generation) are altered in place without changing the view's
				// definition, so they are deliberately not treated as affecting it.
				if newColumn != nil && newColumn.GetType() == column.GetType() &&
					newColumn.GetCollation() == column.GetCollation() &&
					newColumn.GetCollationSchema() == column.GetCollationSchema() {
					continue
				}
				tableKey := objectKey(schema.GetName(), table.GetName())
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
		columns, ok := affected[objectKey(dependency.GetSchema(), dependency.GetTable())]
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
			commented[objectKey(op.SchemaName, op.ObjectName)] = true
		default:
		}
	}
	for _, schema := range meta.GetSchemas() {
		for _, view := range schema.GetViews() {
			key := objectKey(schema.GetName(), view.GetName())
			if view.GetComment() != "" && !commented[key] && planCreatesObject(plan, key) {
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
			key := objectKey(schema.GetName(), view.GetName())
			if view.GetComment() != "" && !commented[key] && planCreatesObject(plan, key) {
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

func planCreatesObject(plan *catalog.MigrationPlan, key string) bool {
	for _, op := range plan.Ops {
		switch op.Type {
		case catalog.OpCreateTable, catalog.OpCreateView, catalog.OpCreateType:
			if objectKey(op.SchemaName, op.ObjectName) == key {
				return true
			}
		default:
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
// not install as defined and that the migration would have to write. An object
// only the source has is dropped, and a drop needs no parsed definition; that
// covers a stand-in and an object the catalog left out but can still drop from
// metadata (sourceDrops).
func unloadableChanges(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, sourceDrops map[string]catalog.MigrationOp, reports ...*catalog.LoadMetadataReport) error {
	reasons := make(map[string]error)
	standIns := make(map[string]bool)
	for _, report := range reports {
		for key, err := range report.Degraded {
			if _, seen := reasons[key]; !seen {
				reasons[key] = err
			}
			standIns[key] = true
		}
		for key, err := range report.Missing {
			if _, seen := reasons[key]; !seen {
				reasons[key] = err
			}
		}
	}
	if len(reasons) == 0 {
		return nil
	}
	oldObjects, newObjects := snapshotObjects(oldMeta), snapshotObjects(newMeta)
	var changed []string
	for key, err := range reasons {
		if newObjects[key] == nil && (standIns[key] || sourceDrops[key].Type != "") {
			continue
		}
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
			objects["func:"+qualified(fn.GetName())+"|"+routineIdentity(fn.GetSignature(), fn.GetDefinition())] = fn
		}
		for _, p := range s.GetProcedures() {
			objects["func:"+qualified(p.GetName())+"|"+routineIdentity(p.GetSignature(), p.GetDefinition())] = p
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

// routineIdentity distinguishes routine overloads, the way the load report does.
func routineIdentity(signature, definition string) string {
	if signature != "" {
		return signature
	}
	return definition
}

// sourceOnlyMissingDrops builds the drops for objects only the source snapshot has
// and the load left out entirely. The catalog never installed them, so it cannot
// diff their removal; their metadata still names them, and a drop needs no parsed
// definition.
func sourceOnlyMissingDrops(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, reports ...*catalog.LoadMetadataReport) map[string]catalog.MigrationOp {
	missing := make(map[string]bool)
	for _, report := range reports {
		for key := range report.Missing {
			missing[key] = true
		}
	}
	drops := make(map[string]catalog.MigrationOp)
	if len(missing) == 0 {
		return drops
	}
	newObjects := snapshotObjects(newMeta)
	add := func(key string, op catalog.MigrationOp) {
		if !missing[key] || newObjects[key] != nil {
			return
		}
		op.Phase = catalog.PhasePre
		drops[key] = op
	}
	addMissingIndexDrops := func(schemaName, owner string, indexes []*metadatapb.IndexMetadata) {
		if newObjects["rel:"+schemaName+"."+owner] == nil {
			return
		}
		for _, index := range indexes {
			key := "idx:" + schemaName + "." + owner + "." + index.GetName()
			if index.GetIsConstraint() {
				constraintName := index.GetName()
				if constraints := index.GetConstraints(); len(constraints) > 0 {
					constraintName = constraints[0].GetName()
				}
				add(key, catalog.MigrationOp{
					Type:         catalog.OpDropConstraint,
					SchemaName:   schemaName,
					ObjectName:   constraintName,
					ParentObject: owner,
					SQL:          "ALTER TABLE " + quoteQualified(schemaName, owner) + " DROP CONSTRAINT " + quoteIdentifier(constraintName),
					ObjType:      'c',
				})
				continue
			}
			add(key, catalog.MigrationOp{
				Type:       catalog.OpDropIndex,
				SchemaName: schemaName,
				ObjectName: index.GetName(),
				SQL:        "DROP INDEX " + quoteQualified(schemaName, index.GetName()),
				ObjType:    'i',
			})
		}
	}
	addMissingConstraintDrops := func(schemaName, owner string, checks []*metadatapb.CheckConstraintMetadata, excludes []*metadatapb.ExcludeConstraintMetadata, foreignKeys []*metadatapb.ForeignKeyMetadata) {
		if newObjects["rel:"+schemaName+"."+owner] == nil {
			return
		}
		addConstraint := func(name string) {
			add("con:"+schemaName+"."+owner+"."+name, catalog.MigrationOp{
				Type:         catalog.OpDropConstraint,
				SchemaName:   schemaName,
				ObjectName:   name,
				ParentObject: owner,
				SQL:          "ALTER TABLE " + quoteQualified(schemaName, owner) + " DROP CONSTRAINT " + quoteIdentifier(name),
				ObjType:      'c',
			})
		}
		for _, check := range checks {
			addConstraint(check.GetName())
		}
		for _, exclude := range excludes {
			addConstraint(exclude.GetName())
		}
		for _, foreignKey := range foreignKeys {
			addConstraint(foreignKey.GetName())
		}
	}

	for _, extension := range oldMeta.GetExtensions() {
		add("ext:"+extension.GetName(), catalog.MigrationOp{
			Type:       catalog.OpDropExtension,
			ObjectName: extension.GetName(),
			SQL:        "DROP EXTENSION " + quoteIdentifier(extension.GetName()),
		})
	}
	for _, schema := range oldMeta.GetSchemas() {
		schemaName := schema.GetName()
		qualified := func(name string) string { return schemaName + "." + name }
		for _, enum := range schema.GetEnumTypes() {
			add("type:"+qualified(enum.GetName()), catalog.MigrationOp{
				Type:       catalog.OpDropType,
				SchemaName: schemaName,
				ObjectName: enum.GetName(),
				SQL:        "DROP TYPE " + quoteQualified(schemaName, enum.GetName()),
			})
		}
		for _, composite := range schema.GetCompositeTypes() {
			add("type:"+qualified(composite.GetName()), catalog.MigrationOp{
				Type:       catalog.OpDropType,
				SchemaName: schemaName,
				ObjectName: composite.GetName(),
				SQL:        "DROP TYPE " + quoteQualified(schemaName, composite.GetName()),
			})
		}
		for _, sequence := range schema.GetSequences() {
			add("seq:"+qualified(sequence.GetName()), catalog.MigrationOp{
				Type:       catalog.OpDropSequence,
				SchemaName: schemaName,
				ObjectName: sequence.GetName(),
				SQL:        "DROP SEQUENCE " + quoteQualified(schemaName, sequence.GetName()),
			})
		}
		for _, function := range schema.GetFunctions() {
			key := "func:" + qualified(function.GetName()) + "|" + routineIdentity(function.GetSignature(), function.GetDefinition())
			add(key, catalog.MigrationOp{
				Type:       catalog.OpDropFunction,
				SchemaName: schemaName,
				ObjectName: function.GetName(),
				SQL:        "DROP FUNCTION " + quoteQualified(schemaName, function.GetName()) + routineDropArguments(function.GetSignature(), function.GetDefinition()),
				ObjType:    'f',
			})
		}
		for _, procedure := range schema.GetProcedures() {
			key := "func:" + qualified(procedure.GetName()) + "|" + routineIdentity(procedure.GetSignature(), procedure.GetDefinition())
			add(key, catalog.MigrationOp{
				Type:       catalog.OpDropFunction,
				SchemaName: schemaName,
				ObjectName: procedure.GetName(),
				SQL:        "DROP PROCEDURE " + quoteQualified(schemaName, procedure.GetName()) + routineDropArguments(procedure.GetSignature(), procedure.GetDefinition()),
				ObjType:    'f',
			})
		}
		for _, table := range schema.GetTables() {
			addMissingIndexDrops(schemaName, table.GetName(), table.GetIndexes())
			addMissingConstraintDrops(schemaName, table.GetName(), table.GetCheckConstraints(), table.GetExcludeConstraints(), table.GetForeignKeys())
			forEachPartition(table.GetPartitions(), func(partition *metadatapb.TablePartitionMetadata) {
				partitionSchema := partition.GetSchema()
				if partitionSchema == "" {
					partitionSchema = schemaName
				}
				addMissingIndexDrops(partitionSchema, partition.GetName(), partition.GetIndexes())
				addMissingConstraintDrops(partitionSchema, partition.GetName(), partition.GetCheckConstraints(), partition.GetExcludeConstraints(), nil)
			})
		}
		for _, view := range schema.GetMaterializedViews() {
			addMissingIndexDrops(schemaName, view.GetName(), view.GetIndexes())
		}
	}
	return drops
}

// routineDropArguments is the parenthesized argument list a DROP accepts, which
// names the overload without repeating its name. A routine without one drops by
// name when its name is unique, as PostgreSQL allows.
func routineDropArguments(signature, definition string) string {
	for _, text := range []string{signature, definition} {
		if indexOutsideQuotes(text, "(") >= 0 {
			return "(" + routineArgumentList(text) + ")"
		}
	}
	return ""
}

// insertSourceOnlyDrops puts a routine or constraint drop before the catalog's
// own PhasePre drops, which may drop the type, table or referenced table it
// depends on, and other metadata-derived drops after them, so an index is gone
// before the column it names changes.
func insertSourceOnlyDrops(ops []catalog.MigrationOp, drops map[string]catalog.MigrationOp) []catalog.MigrationOp {
	if len(drops) == 0 {
		return ops
	}
	keys := make([]string, 0, len(drops))
	for key := range drops {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	early := make([]catalog.MigrationOp, 0, len(drops))
	late := make([]catalog.MigrationOp, 0, len(drops))
	for _, key := range keys {
		switch drops[key].Type {
		case catalog.OpDropFunction, catalog.OpDropConstraint:
			early = append(early, drops[key])
		default:
			late = append(late, drops[key])
		}
	}
	at := 0
	for at < len(ops) && ops[at].Phase == catalog.PhasePre {
		at++
	}
	result := make([]catalog.MigrationOp, 0, len(ops)+len(early)+len(late))
	result = append(result, early...)
	result = append(result, ops[:at]...)
	result = append(result, late...)
	result = append(result, ops[at:]...)
	return result
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
