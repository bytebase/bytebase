// Package mssql implements Bytebase's schema contracts for SQL Server on top of
// omni's SQL Server catalog. The catalog owns metadata extraction, table and
// database definitions, the diff, and the migration; this package keeps the
// registry wiring and the legacy dump-compatibility shims.
//
// Two shims preserve the pre-cutover dump output at the pinned omni version:
//
//   - the old dump omitted sequences, synonyms, triggers, comments and named
//     default constraints. stripUnrenderedFacts and stripTableUnrenderedFacts
//     keep those omissions so the dump stays byte-identical until each fix can
//     ship with its own release note.
//   - restoreLegacyModuleText and GetViewDefinition/GetFunctionDefinition/
//     GetProcedureDefinition keep the stored module text (including an ALTER
//     header or a trailing semicolon) instead of the catalog's normalized header.
//
// The one text difference the strip does not remove is module ordering: the
// catalog creates a function after any function its body names, and before a
// table default or view body that references it, where the old renderer wrote
// functions and procedures last in metadata order. The text of each module is
// unchanged, only its position.
package mssql

import (
	"context"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mssql/catalog"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.Register(schema.Engine{
		Definitions: schema.Definitions{
			Database:  GetDatabaseDefinition,
			Table:     GetTableDefinition,
			View:      GetViewDefinition,
			Function:  GetFunctionDefinition,
			Procedure: GetProcedureDefinition,
		},
		DatabaseMetadata:      GetDatabaseMetadata,
		DiffMetadataMigration: diffMetadataMigration,
	}, storepb.Engine_MSSQL)
}

// GetDatabaseMetadata parses SQL Server DDL text and returns database metadata
// using omni's SQL Server catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return catalog.GetDatabaseMetadata(schemaText)
}

// GetTableDefinition renders one table as the T-SQL that recreates it.
func GetTableDefinition(schemaName string, table *metadatapb.TableMetadata, sequences []*metadatapb.SequenceMetadata) (string, error) {
	return catalog.GetTableDefinition(schemaName, stripTableUnrenderedFacts(table), sequences)
}

// GetDatabaseDefinition renders a synced SQL Server snapshot as the T-SQL that
// recreates it.
//
// The definition context is ignored, as the old renderer ignored it: SQL Server
// has no SDL form, and its dump carries no session preamble.
func GetDatabaseDefinition(_ schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	stripped := stripUnrenderedFacts(metadata)
	definition, err := catalog.DatabaseDefinition(stripped, catalog.DefinitionOptions{})
	if err != nil {
		return "", err
	}
	return restoreLegacyModuleText(definition, stripped), nil
}

// GetViewDefinition, GetFunctionDefinition and GetProcedureDefinition keep the
// old renderer's exact text. The catalog's per-object renderers trim a stored
// definition's trailing semicolon and rewrite an ALTER header to CREATE; both
// change the string the API returns for one object, so they stay out of the
// dump and the object view until the dump's own fix ships.
func GetViewDefinition(_ string, view *metadatapb.ViewMetadata) (string, error) {
	return legacyViewText(view), nil
}

func GetFunctionDefinition(_ string, function *metadatapb.FunctionMetadata) (string, error) {
	return legacyFunctionText(function), nil
}

func GetProcedureDefinition(_ string, procedure *metadatapb.ProcedureMetadata) (string, error) {
	return legacyProcedureText(procedure), nil
}

func legacyViewText(view *metadatapb.ViewMetadata) string {
	return view.GetDefinition() + ";\n\nGO\n\n"
}

func legacyFunctionText(function *metadatapb.FunctionMetadata) string {
	return function.GetDefinition() + "\n\nGO\n\n"
}

func legacyProcedureText(procedure *metadatapb.ProcedureMetadata) string {
	return procedure.GetDefinition() + "\n\nGO\n\n"
}

// stripUnrenderedFacts preserves the pre-cutover dump. The old renderer never
// wrote sequences, synonyms, triggers, or any extended-property comment, and it
// ignored both the default-constraint name on a column and every SkipDump flag,
// including the schema-level one.
// Each of those is a fix of its own; until one ships with its release note, the
// strip keeps the catalog from adding it to every dump.
func stripUnrenderedFacts(metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	if metadata == nil {
		return nil
	}
	out := proto.CloneOf(metadata)
	for _, schema := range out.GetSchemas() {
		schema.SkipDump = false
		schema.Sequences = nil
		schema.Synonyms = nil
		for _, table := range schema.GetTables() {
			stripTableFacts(table)
		}
		for _, view := range schema.GetViews() {
			view.SkipDump = false
			view.Comment = ""
			view.Triggers = nil
		}
		for _, function := range schema.GetFunctions() {
			function.SkipDump = false
			function.Comment = ""
		}
		for _, procedure := range schema.GetProcedures() {
			procedure.SkipDump = false
			procedure.Comment = ""
		}
	}
	return out
}

func stripTableUnrenderedFacts(table *metadatapb.TableMetadata) *metadatapb.TableMetadata {
	if table == nil {
		return nil
	}
	out := proto.CloneOf(table)
	stripTableFacts(out)
	return out
}

func stripTableFacts(table *metadatapb.TableMetadata) {
	table.SkipDump = false
	table.Comment = ""
	table.Triggers = nil
	for _, column := range table.GetColumns() {
		column.Comment = ""
		column.DefaultConstraintName = ""
	}
}

// restoreLegacyModuleText rewrites the catalog's module sections back to the
// stored text. The old renderer wrote each definition verbatim and appended a
// terminator, so a body ending in ";" gained a second one and an ALTER header
// stayed ALTER; the catalog trims the semicolon and rewrites ALTER to CREATE.
// The catalog's own per-object renderer yields exactly the section it emitted,
// so replacing that section restores the old text without duplicating any of the
// catalog's header logic.
func restoreLegacyModuleText(rendered string, metadata *metadatapb.DatabaseSchemaMetadata) string {
	replacements := moduleReplacements(metadata)
	var out strings.Builder
	out.Grow(len(rendered))
	atBatchStart, atLineStart := true, true
	for i := 0; i < len(rendered); {
		if atBatchStart {
			for i < len(rendered) && (rendered[i] == ' ' || rendered[i] == '\t' || rendered[i] == '\r' || rendered[i] == '\n') {
				out.WriteByte(rendered[i])
				atLineStart = rendered[i] == '\n'
				i++
			}
			if i == len(rendered) {
				break
			}
			if replacement, ok := matchModuleSection(rendered[i:], replacements); ok {
				out.WriteString(replacement.legacy)
				i += len(replacement.catalog)
				atBatchStart, atLineStart = true, true
				continue
			}
		}
		if atLineStart {
			if end, ok := goLineEnd(rendered, i); ok {
				out.WriteString(rendered[i:end])
				i = end
				atBatchStart, atLineStart = true, true
				continue
			}
		}
		var end int
		switch {
		case rendered[i] == '\'':
			end = quotedEnd(rendered, i, '\'')
		case rendered[i] == '"':
			end = quotedEnd(rendered, i, '"')
		case rendered[i] == '[':
			end = quotedEnd(rendered, i, ']')
		case strings.HasPrefix(rendered[i:], "--"):
			end = lineCommentEnd(rendered, i)
		case strings.HasPrefix(rendered[i:], "/*"):
			end = blockCommentEnd(rendered, i)
		default:
			end = i + 1
		}
		out.WriteString(rendered[i:end])
		atLineStart = rendered[end-1] == '\n'
		atBatchStart = false
		i = end
	}
	return out.String()
}

// goLineEnd reports whether a line whose first character is text[i] is a T-SQL
// batch separator, and returns the position after its newline. GO is only a
// separator when it is alone on the line and outside any literal or comment.
func goLineEnd(text string, i int) (int, bool) {
	j := i
	for j < len(text) && (text[j] == ' ' || text[j] == '\t') {
		j++
	}
	if j+2 > len(text) || !strings.EqualFold(text[j:j+2], "GO") {
		return 0, false
	}
	j += 2
	for j < len(text) && (text[j] == ' ' || text[j] == '\t' || text[j] == '\r') {
		j++
	}
	if j == len(text) {
		return j, true
	}
	if text[j] != '\n' {
		return 0, false
	}
	return j + 1, true
}

// quotedEnd returns the position after the closing quote, treating a doubled
// quote as an escaped quote inside the value.
func quotedEnd(text string, i int, closing byte) int {
	j := i + 1
	for j < len(text) {
		if text[j] != closing {
			j++
			continue
		}
		if j+1 < len(text) && text[j+1] == closing {
			j += 2
			continue
		}
		return j + 1
	}
	return len(text)
}

func lineCommentEnd(text string, i int) int {
	if end := strings.IndexByte(text[i:], '\n'); end >= 0 {
		return i + end + 1
	}
	return len(text)
}

// blockCommentEnd handles T-SQL's nested block comments.
func blockCommentEnd(text string, i int) int {
	depth := 1
	for j := i + 2; j < len(text); {
		switch {
		case strings.HasPrefix(text[j:], "/*"):
			depth++
			j += 2
		case strings.HasPrefix(text[j:], "*/"):
			depth--
			j += 2
			if depth == 0 {
				return j
			}
		default:
			j++
		}
	}
	return len(text)
}

type moduleReplacement struct {
	catalog string
	legacy  string
}

func moduleReplacements(metadata *metadatapb.DatabaseSchemaMetadata) []moduleReplacement {
	var replacements []moduleReplacement
	for _, schema := range metadata.GetSchemas() {
		for _, view := range schema.GetViews() {
			if omni, err := catalog.GetViewDefinition(view); err == nil {
				replacements = append(replacements, moduleReplacement{catalog: omni, legacy: legacyViewText(view)})
			}
		}
		for _, function := range schema.GetFunctions() {
			if omni, err := catalog.GetFunctionDefinition(function); err == nil {
				replacements = append(replacements, moduleReplacement{catalog: omni, legacy: legacyFunctionText(function)})
			}
		}
		for _, procedure := range schema.GetProcedures() {
			if omni, err := catalog.GetProcedureDefinition(procedure); err == nil {
				replacements = append(replacements, moduleReplacement{catalog: omni, legacy: legacyProcedureText(procedure)})
			}
		}
	}
	return replacements
}

func matchModuleSection(rest string, replacements []moduleReplacement) (moduleReplacement, bool) {
	for _, replacement := range replacements {
		if strings.HasPrefix(rest, replacement.catalog) {
			return replacement, true
		}
	}
	return moduleReplacement{}, false
}

// reconcileSkipDump reproduces the legacy differ's treatment of SkipDump: an
// object skipped on either side takes no part in the migration. Both snapshots
// carry the source object unchanged in that case, or neither carries it when
// the source lacks it, so the catalog emits neither a drop nor a create/alter.
func reconcileSkipDump(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, caseSensitive bool) (*metadatapb.DatabaseSchemaMetadata, *metadatapb.DatabaseSchemaMetadata) {
	normalize := func(name string) string {
		if caseSensitive {
			return name
		}
		return strings.ToLower(name)
	}
	if oldMeta == nil {
		oldMeta = &metadatapb.DatabaseSchemaMetadata{}
	}
	if newMeta == nil {
		newMeta = &metadatapb.DatabaseSchemaMetadata{}
	}
	oldOut, newOut := proto.CloneOf(oldMeta), proto.CloneOf(newMeta)
	for _, schema := range oldOut.GetSchemas() {
		schema.SkipDump = false
	}
	for _, schema := range newOut.GetSchemas() {
		schema.SkipDump = false
	}
	for _, oldSchema := range oldOut.GetSchemas() {
		if findByName(newOut.GetSchemas(), oldSchema.GetName(), normalize) == nil {
			clearSkipDump(oldSchema)
		}
	}
	for _, oldSchema := range oldOut.GetSchemas() {
		newSchema := findByName(newOut.GetSchemas(), oldSchema.GetName(), normalize)
		if newSchema == nil {
			continue
		}
		newSchema.Tables = reconcileObjects(oldSchema.GetTables(), newSchema.GetTables(),
			func(t *metadatapb.TableMetadata) string { return normalize(t.GetName()) },
			func(t *metadatapb.TableMetadata) bool { return t.GetSkipDump() },
			func(t *metadatapb.TableMetadata) { t.SkipDump = false })
		newSchema.Views = reconcileObjects(oldSchema.GetViews(), newSchema.GetViews(),
			func(v *metadatapb.ViewMetadata) string { return normalize(v.GetName()) },
			func(v *metadatapb.ViewMetadata) bool { return v.GetSkipDump() },
			func(v *metadatapb.ViewMetadata) { v.SkipDump = false })
		newSchema.Functions = reconcileObjects(oldSchema.GetFunctions(), newSchema.GetFunctions(),
			func(f *metadatapb.FunctionMetadata) string { return normalize(f.GetName()) },
			func(f *metadatapb.FunctionMetadata) bool { return f.GetSkipDump() },
			func(f *metadatapb.FunctionMetadata) { f.SkipDump = false })
		newSchema.Procedures = reconcileObjects(oldSchema.GetProcedures(), newSchema.GetProcedures(),
			func(p *metadatapb.ProcedureMetadata) string { return normalize(p.GetName()) },
			func(p *metadatapb.ProcedureMetadata) bool { return p.GetSkipDump() },
			func(p *metadatapb.ProcedureMetadata) { p.SkipDump = false })
		newSchema.Sequences = reconcileObjects(oldSchema.GetSequences(), newSchema.GetSequences(),
			func(s *metadatapb.SequenceMetadata) string { return normalize(s.GetName()) },
			func(s *metadatapb.SequenceMetadata) bool { return s.GetSkipDump() },
			func(s *metadatapb.SequenceMetadata) { s.SkipDump = false })
	}
	for _, newSchema := range newOut.GetSchemas() {
		oldSchema := findByName(oldOut.GetSchemas(), newSchema.GetName(), normalize)
		if oldSchema == nil {
			continue
		}
		for _, table := range newSchema.GetTables() {
			oldTable := findByName(oldSchema.GetTables(), table.GetName(), normalize)
			if oldTable == nil {
				continue
			}
			table.Triggers = reconcileObjects(oldTable.GetTriggers(), table.GetTriggers(),
				func(t *metadatapb.TriggerMetadata) string { return normalize(t.GetName()) },
				func(t *metadatapb.TriggerMetadata) bool { return t.GetSkipDump() },
				func(t *metadatapb.TriggerMetadata) { t.SkipDump = false })
		}
		for _, view := range newSchema.GetViews() {
			oldView := findByName(oldSchema.GetViews(), view.GetName(), normalize)
			if oldView == nil {
				continue
			}
			view.Triggers = reconcileObjects(oldView.GetTriggers(), view.GetTriggers(),
				func(t *metadatapb.TriggerMetadata) string { return normalize(t.GetName()) },
				func(t *metadatapb.TriggerMetadata) bool { return t.GetSkipDump() },
				func(t *metadatapb.TriggerMetadata) { t.SkipDump = false })
		}
	}
	return oldOut, newOut
}

func reconcileObjects[T any](oldObjs, newObjs []T, name func(T) string, skipped func(T) bool, unskip func(T)) []T {
	oldByName := make(map[string]T, len(oldObjs))
	for _, obj := range oldObjs {
		oldByName[name(obj)] = obj
	}
	seen := make(map[string]bool, len(newObjs))
	result := make([]T, 0, len(newObjs)+len(oldObjs))
	for _, newObj := range newObjs {
		objectName := name(newObj)
		oldObj, hasOld := oldByName[objectName]
		switch {
		case hasOld && skipped(oldObj):
			unskip(oldObj)
			result = append(result, oldObj)
		case skipped(newObj):
			if hasOld {
				unskip(oldObj)
				result = append(result, oldObj)
			}
		default:
			result = append(result, newObj)
		}
		seen[objectName] = true
	}
	for _, oldObj := range oldObjs {
		if skipped(oldObj) && !seen[name(oldObj)] {
			unskip(oldObj)
			result = append(result, oldObj)
		}
	}
	return result
}

// clearSkipDump exposes every object of a schema the target removes: the schema
// cannot be dropped while a skipped table or view still occupies it, and the
// legacy differ dropped those objects with the schema.
func clearSkipDump(schema *metadatapb.SchemaMetadata) {
	for _, table := range schema.GetTables() {
		table.SkipDump = false
		for _, trigger := range table.GetTriggers() {
			trigger.SkipDump = false
		}
	}
	for _, view := range schema.GetViews() {
		view.SkipDump = false
		for _, trigger := range view.GetTriggers() {
			trigger.SkipDump = false
		}
	}
	for _, function := range schema.GetFunctions() {
		function.SkipDump = false
	}
	for _, procedure := range schema.GetProcedures() {
		procedure.SkipDump = false
	}
	for _, sequence := range schema.GetSequences() {
		sequence.SkipDump = false
	}
}

func findByName[T interface{ GetName() string }](items []T, name string, normalize func(string) string) T {
	want := normalize(name)
	for _, item := range items {
		if normalize(item.GetName()) == want {
			return item
		}
	}
	var zero T
	return zero
}

// diffMetadataMigration diffs two synced snapshots with omni's catalog. The
// catalog stores each snapshot as the sync reported it, so nothing can fail to
// load; the diff and generator are the decisions.
func diffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	caseSensitive := oldSchema.GetIsObjectCaseSensitive() || newSchema.GetIsObjectCaseSensitive()
	oldMeta, newMeta := reconcileSkipDump(oldSchema.GetProto(), newSchema.GetProto(), caseSensitive)
	from, err := loadCatalogProto(oldMeta)
	if err != nil {
		return "", errors.Wrap(err, "failed to load source schema")
	}
	to, err := loadCatalogProto(newMeta)
	if err != nil {
		return "", errors.Wrap(err, "failed to load target schema")
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}
	return catalog.GenerateMigration(from, to, diff).SQL(), nil
}

func loadCatalogProto(metadata *metadatapb.DatabaseSchemaMetadata) (*catalog.Catalog, error) {
	c := catalog.New()
	if err := c.LoadMetadata(context.Background(), metadata); err != nil {
		return nil, err
	}
	return c, nil
}
