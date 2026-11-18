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
// ignored both the default-constraint name on a column and every SkipDump flag.
// Each of those is a fix of its own; until one ships with its release note, the
// strip keeps the catalog from adding it to every dump.
func stripUnrenderedFacts(metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	if metadata == nil {
		return nil
	}
	out := proto.CloneOf(metadata)
	for _, schema := range out.GetSchemas() {
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
	for i := 0; i < len(rendered); {
		// A module section starts the dump or follows a batch separator, and a
		// match consumes the whole section. Text inside an earlier module's
		// body therefore cannot be mistaken for a later module's section.
		if i == 0 || (i >= len("GO\n\n") && rendered[i-len("GO\n\n"):i] == "GO\n\n") {
			if replacement, ok := matchModuleSection(rendered[i:], replacements); ok {
				out.WriteString(replacement.legacy)
				i += len(replacement.catalog)
				continue
			}
		}
		out.WriteByte(rendered[i])
		i++
	}
	return out.String()
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
		metadata = d.GetProto()
	}
	if err := c.LoadMetadata(context.Background(), metadata); err != nil {
		return nil, err
	}
	return c, nil
}
