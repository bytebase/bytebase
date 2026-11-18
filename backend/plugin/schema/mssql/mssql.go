// Package mssql implements Bytebase's schema contracts for SQL Server on top of
// omni's SQL Server catalog. The catalog owns the database definition, the diff
// and the migration; this package keeps the registry wiring, the metadata
// extractor in get_database_metadata.go, and the single-table renderer in
// table_definition.go.
//
// Two facts keep the cutover partial at the pinned omni version:
//
//   - catalog exports no single-table definition. GetTableDefinition therefore
//     still runs Bytebase's renderer; only the database and per-module
//     definition entry points moved.
//   - the old dump omitted sequences, synonyms, triggers, comments and named
//     default constraints. stripUnrenderedFacts keeps those omissions so the
//     dump stays byte-identical until each fix can ship with its own release
//     note.
//
// The one text difference the strip does not remove is module ordering: the
// catalog creates a function after any function its body names, and before a
// table default or view body that references it, where the old renderer wrote
// functions and procedures last in metadata order. The text of each module is
// unchanged (restoreLegacyModuleText rewrites the catalog's normalized module
// headers back to the stored text), only its position.
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
	schema.RegisterGetDatabaseDefinition(storepb.Engine_MSSQL, GetDatabaseDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_MSSQL, GetTableDefinition)
	schema.RegisterGetViewDefinition(storepb.Engine_MSSQL, GetViewDefinition)
	schema.RegisterGetFunctionDefinition(storepb.Engine_MSSQL, GetFunctionDefinition)
	schema.RegisterGetProcedureDefinition(storepb.Engine_MSSQL, GetProcedureDefinition)
	schema.RegisterDiffMetadataMigration(storepb.Engine_MSSQL, diffMetadataMigration)
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
			table.SkipDump = false
			table.Comment = ""
			table.Triggers = nil
			for _, column := range table.GetColumns() {
				column.Comment = ""
				column.DefaultConstraintName = ""
			}
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

// stripSkipDump ignores SkipDump transitions in a metadata migration, as the
// legacy differ did: the flag says whether a dump includes an object, not
// whether a migration should drop or create it.
func stripSkipDump(metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	if metadata == nil {
		return nil
	}
	out := proto.CloneOf(metadata)
	for _, schema := range out.GetSchemas() {
		schema.SkipDump = false
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
	return out
}

// diffMetadataMigration diffs two synced snapshots with omni's catalog. The
// catalog stores each snapshot as the sync reported it, so nothing can fail to
// load; the diff and generator are the decisions.
func diffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	from, err := loadCatalog(oldSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load source schema")
	}
	to, err := loadCatalog(newSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load target schema")
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}
	return catalog.GenerateMigration(from, to, diff).SQL(), nil
}

func loadCatalog(d *model.DatabaseMetadata) (*catalog.Catalog, error) {
	c := catalog.New()
	var metadata *metadatapb.DatabaseSchemaMetadata
	if d != nil {
		metadata = stripSkipDump(d.GetProto())
	}
	if err := c.LoadMetadata(context.Background(), metadata); err != nil {
		return nil, err
	}
	return c, nil
}
