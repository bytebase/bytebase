package pg

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"

	metadatapb "github.com/bytebase/omni/metadata"
	omnipg "github.com/bytebase/omni/pg"
	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB} {
		schema.RegisterGetDatabaseDefinition(engine, GetDatabaseDefinition)
		schema.RegisterGetSchemaDefinition(engine, catalog.SchemaDefinition)
		schema.RegisterGetTableDefinition(engine, catalog.TableDefinition)
		schema.RegisterGetViewDefinition(engine, catalog.ViewDefinition)
		schema.RegisterGetMaterializedViewDefinition(engine, catalog.MaterializedViewDefinition)
		schema.RegisterGetFunctionDefinition(engine, catalog.FunctionDefinition)
		schema.RegisterGetSequenceDefinition(engine, catalog.SequenceDefinition)
		schema.RegisterGetMultiFileDatabaseDefinition(engine, GetMultiFileDatabaseDefinition)
	}
}

// GetDatabaseDefinition renders metadata through omni's catalog.DatabaseDefinition
// after dropping the backup schema when asked and repairing legacy metadata.
func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(prepareDefinitionMetadata(ctx, metadata), catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}

// GetMultiFileDatabaseDefinition renders metadata through omni's
// catalog.MultiFileDefinition, prepared as GetDatabaseDefinition prepares it.
func GetMultiFileDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (*schema.MultiFileSchemaResult, error) {
	files, err := catalog.MultiFileDefinition(prepareDefinitionMetadata(ctx, metadata))
	if err != nil {
		return nil, err
	}
	result := &schema.MultiFileSchemaResult{}
	if files != nil {
		result.Files = make([]schema.File, 0, len(files))
		for _, file := range files {
			result.Files = append(result.Files, schema.File{Name: file.Name, Content: file.Content})
		}
	}
	return result, nil
}

func prepareDefinitionMetadata(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	metadata = filterBackupSchemaIfNecessary(ctx, metadata)

	// Clone before mutating: the caller's *DatabaseSchemaMetadata is often a
	// shared pointer returned from store.dbSchemaCache (see
	// backend/store/model/database.go:GetProto), so concurrent callers could
	// race on in-place normalization writes below.
	metadata = proto.CloneOf(metadata)

	// Repair historical non-canonical metadata shapes (e.g. index key
	// expressions stored without outer parens) so emission produces valid SQL.
	// See legacy_normalize.go for removal criteria.
	normalizeLegacyMetadata(metadata)
	return metadata
}

func filterBackupSchemaIfNecessary(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	if !ctx.SkipBackupSchema {
		return metadata
	}

	filtered := &metadatapb.DatabaseSchemaMetadata{
		Extensions:    metadata.Extensions,
		EventTriggers: metadata.EventTriggers,
	}
	for _, schema := range metadata.Schemas {
		if schema.Name == common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES) {
			continue
		}
		filtered.Schemas = append(filtered.Schemas, schema)
	}
	return filtered
}

// The writers below render statement fragments for metadata_migration.go. They
// are copies of omni pg/catalog's definition writers, kept until the metadata
// migration moves to omni's catalog.Diff and catalog.GenerateMigration.

func writeTrigger(out io.Writer, schema string, table string, trigger *metadatapb.TriggerMetadata) error {
	if _, err := io.WriteString(out, trigger.Body); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	if len(trigger.Comment) > 0 {
		if err := writeTriggerComment(out, schema, table, trigger); err != nil {
			return err
		}
	}

	return nil
}

func writeTriggerComment(out io.Writer, schema string, table string, trigger *metadatapb.TriggerMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON TRIGGER "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, trigger.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" ON "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, table); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(trigger.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeEventTrigger(out io.Writer, eventTrigger *metadatapb.EventTriggerMetadata) error {
	// Use the stored definition if available
	if eventTrigger.Definition != "" {
		if _, err := io.WriteString(out, eventTrigger.Definition); err != nil {
			return err
		}
		if _, err := io.WriteString(out, ";\n\n"); err != nil {
			return err
		}
	} else {
		// Build the CREATE EVENT TRIGGER statement
		if _, err := io.WriteString(out, `CREATE EVENT TRIGGER "`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, eventTrigger.Name); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `" ON `); err != nil {
			return err
		}
		if _, err := io.WriteString(out, eventTrigger.Event); err != nil {
			return err
		}

		// Add WHEN TAG IN clause if tags are specified
		if len(eventTrigger.Tags) > 0 {
			if _, err := io.WriteString(out, "\n  WHEN TAG IN ("); err != nil {
				return err
			}
			for i, tag := range eventTrigger.Tags {
				if i > 0 {
					if _, err := io.WriteString(out, ", "); err != nil {
						return err
					}
				}
				if _, err := io.WriteString(out, "'"); err != nil {
					return err
				}
				if _, err := io.WriteString(out, tag); err != nil {
					return err
				}
				if _, err := io.WriteString(out, "'"); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(out, ")"); err != nil {
				return err
			}
		}

		// Add EXECUTE FUNCTION clause
		if _, err := io.WriteString(out, "\n  EXECUTE FUNCTION "); err != nil {
			return err
		}
		if eventTrigger.FunctionSchema != "" {
			if _, err := io.WriteString(out, `"`); err != nil {
				return err
			}
			if _, err := io.WriteString(out, eventTrigger.FunctionSchema); err != nil {
				return err
			}
			if _, err := io.WriteString(out, `".`); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, eventTrigger.FunctionName); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"();\n\n`); err != nil {
			return err
		}
	}

	// Write event trigger comment if present
	if eventTrigger.Comment != "" {
		if err := writeEventTriggerComment(out, eventTrigger); err != nil {
			return err
		}
	}

	return nil
}

func writeEventTriggerComment(out io.Writer, eventTrigger *metadatapb.EventTriggerMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON EVENT TRIGGER "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, eventTrigger.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(eventTrigger.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeEnum(out io.Writer, schema string, enum *metadatapb.EnumTypeMetadata) error {
	if _, err := io.WriteString(out, `CREATE TYPE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, enum.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" AS ENUM (\n"); err != nil {
		return err
	}
	for i, value := range enum.Values {
		if i > 0 {
			if _, err := io.WriteString(out, ",\n"); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `    '`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, escapeSingleQuote(value)); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `'`); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, "\n)"); err != nil {
		return err
	}

	return nil
}

func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func writeCompositeType(out io.Writer, schema string, composite *metadatapb.CompositeTypeMetadata) error {
	if _, err := io.WriteString(out, `CREATE TYPE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, composite.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" AS (\n"); err != nil {
		return err
	}
	for i, attribute := range composite.Attributes {
		if i > 0 {
			if _, err := io.WriteString(out, ",\n"); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `    "`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, attribute.Name); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `" `); err != nil {
			return err
		}
		if _, err := io.WriteString(out, attribute.Type); err != nil {
			return err
		}
		// The collation is stored as an emit-ready identifier reference
		// (quoted as needed, schema-qualified outside pg_catalog).
		if attribute.Collation != "" {
			if _, err := io.WriteString(out, ` COLLATE `); err != nil {
				return err
			}
			if _, err := io.WriteString(out, attribute.Collation); err != nil {
				return err
			}
		}
	}

	if _, err := io.WriteString(out, "\n)"); err != nil {
		return err
	}

	return nil
}

func compositeTypeHasComments(composite *metadatapb.CompositeTypeMetadata) bool {
	if composite.Comment != "" {
		return true
	}
	for _, attribute := range composite.Attributes {
		if attribute.Comment != "" {
			return true
		}
	}
	return false
}

func writeCompositeTypeComments(out io.Writer, schema string, composite *metadatapb.CompositeTypeMetadata) error {
	if composite.Comment != "" {
		if _, err := fmt.Fprintf(out, "COMMENT ON TYPE \"%s\".\"%s\" IS '%s';\n\n", schema, composite.Name, escapeSingleQuote(composite.Comment)); err != nil {
			return err
		}
	}
	for _, attribute := range composite.Attributes {
		if attribute.Comment == "" {
			continue
		}
		if _, err := fmt.Fprintf(out, "COMMENT ON COLUMN \"%s\".\"%s\".\"%s\" IS '%s';\n\n", schema, composite.Name, attribute.Name, escapeSingleQuote(attribute.Comment)); err != nil {
			return err
		}
	}
	return nil
}

// qualifiedCompositeType pairs a composite type with its schema for
// cross-schema ordering.
type qualifiedCompositeType struct {
	Schema    string
	Composite *metadatapb.CompositeTypeMetadata
}

// parseQualifiedTypeIdent splits a type reference produced by the sync layer
// (format_type output, schema-qualified for user-defined types) into
// (schema, name), ignoring any typmod or array suffix. It returns ok=false
// for unqualified references — built-in types are never schema-qualified.
func parseQualifiedTypeIdent(s string) (string, string, bool) {
	var parts []string
	for i := 0; i < len(s); {
		if s[i] == '"' {
			var b strings.Builder
			i++
			for i < len(s) {
				if s[i] == '"' {
					if i+1 < len(s) && s[i+1] == '"' {
						b.WriteByte('"')
						i += 2
						continue
					}
					i++
					break
				}
				b.WriteByte(s[i])
				i++
			}
			parts = append(parts, b.String())
		} else {
			start := i
			for i < len(s) && s[i] != '.' && s[i] != '(' && s[i] != '[' {
				i++
			}
			parts = append(parts, strings.TrimSpace(s[start:i]))
		}
		if i < len(s) && s[i] == '.' {
			i++
			continue
		}
		break
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], true
	}
	return "", "", false
}

// sortCompositeTypesTopologically orders composite types so that a composite
// referenced by another composite's attribute is emitted first. PostgreSQL
// rejects self-containment, so the graph is acyclic; any leftover on an
// unexpected cycle falls back to (schema, name) order. A composite attribute
// typed with a table or view row type is not reordered here — that requires
// catalog-grade dependency data (see pg_dump's pg_depend-driven sort).
func sortCompositeTypesTopologically(types []qualifiedCompositeType) []qualifiedCompositeType {
	key := func(schema, name string) string { return schema + "\x00" + name }
	nodes := make(map[string]qualifiedCompositeType, len(types))
	for _, t := range types {
		nodes[key(t.Schema, t.Composite.Name)] = t
	}
	inDegree := make(map[string]int, len(nodes))
	dependents := make(map[string][]string, len(nodes))
	for k := range nodes {
		inDegree[k] = 0
	}
	for k, t := range nodes {
		for _, attribute := range t.Composite.Attributes {
			depSchema, depName, ok := parseQualifiedTypeIdent(attribute.Type)
			if !ok {
				continue
			}
			depKey := key(depSchema, depName)
			if depKey == k {
				continue
			}
			if _, exists := nodes[depKey]; !exists {
				continue
			}
			dependents[depKey] = append(dependents[depKey], k)
			inDegree[k]++
		}
	}

	var ready []string
	for k, d := range inDegree {
		if d == 0 {
			ready = append(ready, k)
		}
	}
	slices.Sort(ready)

	ordered := make([]qualifiedCompositeType, 0, len(nodes))
	for len(ready) > 0 {
		k := ready[0]
		ready = ready[1:]
		ordered = append(ordered, nodes[k])
		delete(inDegree, k)
		for _, dependent := range dependents[k] {
			if _, remaining := inDegree[dependent]; !remaining {
				continue
			}
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
		slices.Sort(ready)
	}

	if len(ordered) < len(nodes) {
		var rest []string
		for k := range inDegree {
			rest = append(rest, k)
		}
		slices.Sort(rest)
		for _, k := range rest {
			ordered = append(ordered, nodes[k])
		}
	}

	return ordered
}

// signatureTypesOnly strips parameter names from a function/procedure signature,
// keeping only types. PostgreSQL's COMMENT ON FUNCTION/PROCEDURE requires
// type-only signatures: "func(integer, text)" not "func(id integer, name text)".
//
// It works by wrapping the signature in a CREATE FUNCTION statement, parsing it
// with omni to discover parameter names, then removing those names from the
// original signature text to preserve the original type spelling.
func signatureTypesOnly(signature string) string {
	openParen := strings.Index(signature, "(")
	if openParen < 0 {
		return signature
	}
	closeParen := strings.LastIndex(signature, ")")
	if closeParen <= openParen {
		return signature
	}
	funcName := signature[:openParen]
	paramStr := signature[openParen+1 : closeParen]
	if strings.TrimSpace(paramStr) == "" {
		return signature
	}

	// Parse the signature as a CREATE FUNCTION to extract param names from AST.
	sql := fmt.Sprintf("CREATE FUNCTION %s RETURNS void LANGUAGE sql AS 'SELECT 1'", signature)
	stmts, err := omnipg.Parse(sql)
	if err != nil {
		return signature
	}
	fn, ok := stmts[0].AST.(*ast.CreateFunctionStmt)
	if !ok || fn.Parameters == nil {
		return signature
	}

	// Collect parameter names from AST.
	var paramNames []string
	for _, item := range fn.Parameters.Items {
		if fp, ok := item.(*ast.FunctionParameter); ok && fp.Name != "" {
			paramNames = append(paramNames, fp.Name)
		}
	}

	// Strip parameter names from the original text, preserving original type spelling.
	rawParams := strings.Split(paramStr, ",")
	var typesOnly []string
	nameIdx := 0
	for _, raw := range rawParams {
		param := strings.TrimSpace(raw)
		words := strings.Fields(param)
		if len(words) == 0 {
			continue
		}
		// Skip mode keywords (IN, OUT, INOUT, VARIADIC).
		startIdx := 0
		for startIdx < len(words) {
			w := strings.ToLower(words[startIdx])
			if w == "in" || w == "out" || w == "inout" || w == "variadic" {
				startIdx++
			} else {
				break
			}
		}
		// Check if first non-mode word matches a known param name from AST.
		if startIdx < len(words) && nameIdx < len(paramNames) && words[startIdx] == paramNames[nameIdx] {
			typesOnly = append(typesOnly, strings.Join(words[startIdx+1:], " "))
			nameIdx++
		} else {
			typesOnly = append(typesOnly, strings.Join(words[startIdx:], " "))
		}
	}

	return funcName + "(" + strings.Join(typesOnly, ", ") + ")"
}

func writeEnumComment(out io.Writer, schema string, enum *metadatapb.EnumTypeMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON TYPE "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, enum.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(enum.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeMaterializedView(out io.Writer, schema string, view *metadatapb.MaterializedViewMetadata) error {
	if _, err := io.WriteString(out, `CREATE MATERIALIZED VIEW "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" AS \n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, strings.TrimRight(view.Definition, ";")); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n  WITH NO DATA;\n\n"); err != nil {
		return err
	}

	for _, index := range view.Indexes {
		if err := writeIndex(out, schema, view.Name, index, false); err != nil {
			return err
		}
	}

	if len(view.Comment) > 0 {
		if err := writeMaterializedViewComment(out, schema, view); err != nil {
			return err
		}
	}

	return nil
}

func writeMaterializedViewComment(out io.Writer, schema string, view *metadatapb.MaterializedViewMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON MATERIALIZED VIEW "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(view.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeCreateSequence(out io.Writer, schema string, sequence *metadatapb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `CREATE SEQUENCE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\"\n    "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "AS "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.DataType); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	START WITH "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Start); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	INCREMENT BY "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Increment); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	MINVALUE "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.MinValue); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	MAXVALUE "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.MaxValue); err != nil {
		return err
	}
	if sequence.Cycle {
		if _, err := io.WriteString(out, "\n	CYCLE"); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, "\n	NO CYCLE"); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	if len(sequence.Comment) > 0 {
		if err := writeSequenceComment(out, schema, sequence); err != nil {
			return err
		}
	}

	return nil
}

func writeSequenceComment(out io.Writer, schema string, sequence *metadatapb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON SEQUENCE "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(sequence.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeAlterSequenceOwnedBy(out io.Writer, schema string, sequence *metadatapb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `ALTER SEQUENCE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" OWNED BY \""); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.OwnerTable); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.OwnerColumn); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\";\n\n")
	return err
}

func writeCreateTable(out io.Writer, schema string, tableName string, columns []*metadatapb.ColumnMetadata, checks []*metadatapb.CheckConstraintMetadata, excludes []*metadatapb.ExcludeConstraintMetadata) error {
	if _, err := io.WriteString(out, `CREATE TABLE "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, tableName); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" (`); err != nil {
		return err
	}

	// first tracks whether we've written any element inside the parens yet so
	// the leading comma on constraints is suppressed when there are no columns
	// (e.g. tables that only carry CHECK constraints).
	first := true
	writeItemSep := func() error {
		if first {
			first = false
			_, err := io.WriteString(out, "\n    ")
			return err
		}
		_, err := io.WriteString(out, ",\n    ")
		return err
	}

	for _, column := range columns {
		if err := writeItemSep(); err != nil {
			return err
		}

		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}

		if _, err := io.WriteString(out, column.Name); err != nil {
			return err
		}

		if _, err := io.WriteString(out, `" `); err != nil {
			return err
		}

		if _, err := io.WriteString(out, column.Type); err != nil {
			return err
		}

		// Handle default values
		if column.Default != "" {
			if _, err := io.WriteString(out, ` DEFAULT `); err != nil {
				return err
			}
			if _, err := io.WriteString(out, column.Default); err != nil {
				return err
			}
		}

		if !column.Nullable {
			if _, err := io.WriteString(out, ` NOT NULL`); err != nil {
				return err
			}
		}
	}

	for _, check := range checks {
		if err := writeItemSep(); err != nil {
			return err
		}
		_, _ = io.WriteString(out, `CONSTRAINT "`)
		_, _ = io.WriteString(out, check.Name)
		_, _ = io.WriteString(out, `" CHECK `)
		_, _ = io.WriteString(out, check.Expression)
	}

	for _, exclude := range excludes {
		if err := writeItemSep(); err != nil {
			return err
		}
		_, _ = io.WriteString(out, `CONSTRAINT "`)
		_, _ = io.WriteString(out, exclude.Name)
		_, _ = io.WriteString(out, `" `)
		_, _ = io.WriteString(out, exclude.Expression) // Already includes "EXCLUDE USING ..."
	}

	_, err := io.WriteString(out, "\n)")
	return err
}

func writeForeignKey(out io.Writer, schema string, table string, fk *metadatapb.ForeignKeyMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\"\n    ADD CONSTRAINT \""); err != nil {
		return err
	}
	if _, err := io.WriteString(out, fk.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" FOREIGN KEY ("); err != nil {
		return err
	}
	for i, column := range fk.Columns {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, column); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ")\n    REFERENCES \""); err != nil {
		return err
	}
	if _, err := io.WriteString(out, fk.ReferencedSchema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, fk.ReferencedTable); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" ("); err != nil {
		return err
	}
	for i, column := range fk.ReferencedColumns {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, column); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ")"); err != nil {
		return err
	}

	// Add ON DELETE clause if specified
	if fk.OnDelete != "" && fk.OnDelete != "NO ACTION" {
		if _, err := io.WriteString(out, "\n    ON DELETE "); err != nil {
			return err
		}
		if _, err := io.WriteString(out, fk.OnDelete); err != nil {
			return err
		}
	}

	// Add ON UPDATE clause if specified
	if fk.OnUpdate != "" && fk.OnUpdate != "NO ACTION" {
		if _, err := io.WriteString(out, "\n    ON UPDATE "); err != nil {
			return err
		}
		if _, err := io.WriteString(out, fk.OnUpdate); err != nil {
			return err
		}
	}

	_, err := io.WriteString(out, ";\n\n")
	return err
}

func writeAttachPartitionIndex(out io.Writer, schema string, partition *metadatapb.TablePartitionMetadata) error {
	for _, index := range partition.Indexes {
		if err := writeAttachIndex(out, schema, index); err != nil {
			return err
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writeAttachPartitionIndex(out, schema, subpartition); err != nil {
			return err
		}
	}
	return nil
}

func writeAttachIndex(out io.Writer, schema string, index *metadatapb.IndexMetadata) error {
	if len(index.ParentIndexName) == 0 || len(index.ParentIndexSchema) == 0 {
		return nil
	}

	if _, err := io.WriteString(out, `ALTER INDEX "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.ParentIndexSchema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.ParentIndexName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ATTACH PARTITION "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\";\n\n")
	return err
}

func writeIndex(out io.Writer, schema string, table string, index *metadatapb.IndexMetadata, useOnlyClause bool) error {
	return writeIndexInternal(out, schema, table, index, useOnlyClause, true)
}

func writeIndexKeyList(out io.Writer, index *metadatapb.IndexMetadata) error {
	if _, err := io.WriteString(out, `(`); err != nil {
		return err
	}

	for i, expression := range index.Expressions {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}

		// expression is expected to be in canonical pg_get_indexdef form —
		// see IndexMetadata.expressions in proto/store/store/database.proto.
		// normalizeLegacyMetadata (called at the top of GetDatabaseDefinition)
		// repairs historical rows that pre-date the contract.
		if _, err := io.WriteString(out, expression); err != nil {
			return err
		}

		// Add opclass if it's not the default
		if i < len(index.OpclassNames) && i < len(index.OpclassDefaults) {
			if !index.OpclassDefaults[i] && index.OpclassNames[i] != "" {
				if _, err := io.WriteString(out, " "); err != nil {
					return err
				}
				if _, err := io.WriteString(out, index.OpclassNames[i]); err != nil {
					return err
				}
			}
		}

		// Add DESC if this column is marked as descending
		if i < len(index.Descending) && index.Descending[i] {
			if _, err := io.WriteString(out, " DESC"); err != nil {
				return err
			}
		}
		// Note: NULLS ordering information is not available in IndexMetadata
		// so we omit it in the generated DDL
	}

	_, err := io.WriteString(out, ")")
	return err
}

func writeIndexComment(out io.Writer, schema string, index *metadatapb.IndexMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON INDEX "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(index.Comment)); err != nil {
		return err
	}

	_, err := io.WriteString(out, "';\n\n")
	return err
}

func writePartitionUniqueKey(out io.Writer, schema string, partition *metadatapb.TablePartitionMetadata) error {
	for _, index := range partition.Indexes {
		if index.Unique && !index.Primary && index.IsConstraint {
			if err := writeUniqueKey(out, schema, partition.Name, index); err != nil {
				return err
			}
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writePartitionUniqueKey(out, schema, subpartition); err != nil {
			return err
		}
	}
	return nil
}

func writeUniqueKey(out io.Writer, schema string, table string, index *metadatapb.IndexMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE ONLY "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ADD CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" UNIQUE (`); err != nil {
		return err
	}
	for i, expression := range index.Expressions {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, expression); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ");\n\n"); err != nil {
		return err
	}

	if len(index.Comment) > 0 {
		if err := writeConstraintComment(out, schema, table, index); err != nil {
			return err
		}
	}

	return nil
}

func writePartitionPrimaryKey(out io.Writer, schema string, partition *metadatapb.TablePartitionMetadata) error {
	for _, index := range partition.Indexes {
		if index.Primary {
			if err := writePrimaryKey(out, schema, partition.Name, index); err != nil {
				return err
			}
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writePartitionPrimaryKey(out, schema, subpartition); err != nil {
			return err
		}
	}
	return nil
}

func writePrimaryKey(out io.Writer, schema string, table string, index *metadatapb.IndexMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE ONLY "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ADD CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" PRIMARY KEY (`); err != nil {
		return err
	}
	for i, expression := range index.Expressions {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, expression); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ");\n\n"); err != nil {
		return err
	}

	if len(index.Comment) > 0 {
		if err := writeConstraintComment(out, schema, table, index); err != nil {
			return err
		}
	}

	return nil
}

func writeConstraintComment(out io.Writer, schema string, table string, index *metadatapb.IndexMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON CONSTRAINT "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" ON "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, table); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(index.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeColumnComment(out io.Writer, schema string, table string, column *metadatapb.ColumnMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON COLUMN "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, column.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(column.Comment)); err != nil {
		return err
	}
	_, err := io.WriteString(out, "';\n\n")
	return err
}

func writeTableComment(out io.Writer, schema string, table *metadatapb.TableMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON TABLE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(table.Comment)); err != nil {
		return err
	}
	_, err := io.WriteString(out, "';\n\n")
	return err
}

func writePartitionClause(out io.Writer, partition *metadatapb.TablePartitionMetadata) error {
	if _, err := io.WriteString(out, " PARTITION BY "); err != nil {
		return err
	}
	_, err := io.WriteString(out, partition.Expression)
	return err
}

func writeAttachPartition(out io.Writer, schema string, tableName string, partition *metadatapb.TablePartitionMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE ONLY "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, tableName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ATTACH PARTITION "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, partition.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, partition.Value); err != nil {
		return err
	}
	_, err := io.WriteString(out, ";\n\n")
	return err
}

func writePartitionTable(out io.Writer, schema string, columns []*metadatapb.ColumnMetadata, partition *metadatapb.TablePartitionMetadata) error {
	if err := writeCreateTable(out, schema, partition.Name, columns, partition.CheckConstraints, partition.ExcludeConstraints); err != nil {
		return err
	}

	if len(partition.Subpartitions) > 0 {
		if err := writePartitionClause(out, partition.Subpartitions[0]); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	// Construct subpartition tables.
	for _, subpartition := range partition.Subpartitions {
		if err := writePartitionTable(out, schema, columns, subpartition); err != nil {
			return err
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writeAttachPartition(out, schema, partition.Name, subpartition); err != nil {
			return err
		}
	}

	return nil
}

// isDefinitionProcedure checks if the definition string represents a PROCEDURE (not a FUNCTION).
// Uses omni parser: procedures have a DefElem{Defname: "isProcedure"} in Options.
func isDefinitionProcedure(definition string) bool {
	if definition == "" {
		return false
	}
	stmts, err := omnipg.Parse(definition)
	if err != nil || len(stmts) == 0 {
		return false
	}
	fn, ok := stmts[0].AST.(*ast.CreateFunctionStmt)
	if !ok || fn == nil || fn.Options == nil {
		return false
	}
	for _, item := range fn.Options.Items {
		if def, ok := item.(*ast.DefElem); ok && def.Defname == "isProcedure" {
			return true
		}
	}
	return false
}

func writeFunctionComment(out io.Writer, schema string, function *metadatapb.FunctionMetadata) error {
	// Determine if this is a PROCEDURE or FUNCTION by checking the definition
	objectType := "FUNCTION"
	if isDefinitionProcedure(function.Definition) {
		objectType = "PROCEDURE"
	}

	if _, err := io.WriteString(out, "COMMENT ON "+objectType+" \""); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `".`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, signatureTypesOnly(function.Signature)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, ` IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(function.Comment)); err != nil {
		return err
	}

	_, err := io.WriteString(out, "';\n\n")
	return err
}

func writeExtension(out io.Writer, extension *metadatapb.ExtensionMetadata) error {
	if _, err := io.WriteString(out, `CREATE EXTENSION IF NOT EXISTS "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, extension.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"`); err != nil {
		return err
	}

	// Add WITH SCHEMA clause if schema is specified and not empty
	if extension.Schema != "" {
		if _, err := io.WriteString(out, ` WITH SCHEMA "`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, extension.Schema); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
	}

	// Add VERSION clause if version is specified
	if extension.Version != "" {
		if _, err := io.WriteString(out, ` VERSION '`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, extension.Version); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `'`); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, `;`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, "\n"); err != nil {
		return err
	}

	// Write extension description (comment) if present
	if extension.Description != "" {
		if _, err := io.WriteString(out, "\n"); err != nil {
			return err
		}
		if err := writeExtensionComment(out, extension); err != nil {
			return err
		}
	} else {
		// Add blank line even if no description
		if _, err := io.WriteString(out, "\n"); err != nil {
			return err
		}
	}

	return nil
}

func writeExtensionComment(out io.Writer, extension *metadatapb.ExtensionMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON EXTENSION "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, extension.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(extension.Description)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

// writeIndexInternal is the core index writing function with options for different modes
func writeIndexInternal(out io.Writer, schema string, table string, index *metadatapb.IndexMetadata, useOnlyClause bool, includeTerminatorAndComment bool) error {
	if index.Unique {
		if _, err := io.WriteString(out, `CREATE UNIQUE INDEX "`); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, `CREATE INDEX "`); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}

	// TODO: useOnlyClause assumes partition-structured metadata. If legacy
	// PostgreSQL table inheritance support is ever needed, revisit with
	// definition-driven ONLY extraction.
	if useOnlyClause {
		if _, err := io.WriteString(out, `" ON ONLY "`); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, `" ON "`); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"`); err != nil {
		return err
	}

	// Add USING clause for non-btree indexes
	if index.Type != "" && index.Type != "btree" {
		if _, err := fmt.Fprintf(out, " USING %s", strings.ToUpper(index.Type)); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, ` `); err != nil {
		return err
	}
	if err := writeIndexKeyList(out, index); err != nil {
		return err
	}

	// Write INCLUDE and WHERE clauses from the full index definition (from pg_get_indexdef).
	if index.Definition != "" {
		includeClause, whereClause := extractIndexClauses(index.Definition)
		if includeClause != "" {
			if _, err := fmt.Fprintf(out, " %s", includeClause); err != nil {
				return err
			}
		}
		if whereClause != "" {
			if _, err := fmt.Fprintf(out, " WHERE %s", whereClause); err != nil {
				return err
			}
		}
	}

	if includeTerminatorAndComment {
		if _, err := io.WriteString(out, ";\n\n"); err != nil {
			return err
		}

		if len(index.Comment) > 0 {
			if err := writeIndexComment(out, schema, index); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeViewSDL(out io.Writer, schemaName string, view *metadatapb.ViewMetadata) error {
	if _, err := io.WriteString(out, `CREATE VIEW "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" AS `); err != nil {
		return err
	}

	// The view definition should already include the SELECT statement
	definition := strings.TrimSpace(view.Definition)
	// Remove trailing semicolon if present
	definition = strings.TrimSuffix(definition, ";")

	_, err := io.WriteString(out, definition)
	return err
}

func writeSchema(out io.Writer, schema *metadatapb.SchemaMetadata) error {
	if schema.Name == "public" {
		return nil
	}

	if _, err := io.WriteString(out, `CREATE SCHEMA IF NOT EXISTS "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `";`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

// extractIndexClauses parses a CREATE INDEX statement using the omni parser and
// extracts the INCLUDE clause and WHERE clause as strings.
func extractIndexClauses(definition string) (includeClause, whereClause string) {
	stmts, err := omnipg.Parse(definition)
	if err != nil || len(stmts) == 0 {
		return "", ""
	}
	idx, ok := stmts[0].AST.(*ast.IndexStmt)
	if !ok || idx == nil {
		return "", ""
	}

	// Extract INCLUDE columns using Loc to preserve original quoting.
	if idx.IndexIncludingParams != nil {
		var cols []string
		for _, item := range idx.IndexIncludingParams.Items {
			if elem, ok := item.(*ast.IndexElem); ok && elem.Loc.Start >= 0 && elem.Loc.End > elem.Loc.Start {
				cols = append(cols, definition[elem.Loc.Start:elem.Loc.End])
			}
		}
		if len(cols) > 0 {
			includeClause = "INCLUDE (" + strings.Join(cols, ", ") + ")"
		}
	}

	// Extract WHERE clause using AST node Loc for precise extraction.
	if idx.WhereClause != nil {
		loc := ast.NodeLoc(idx.WhereClause)
		if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(definition) {
			raw := definition[loc.Start:loc.End]
			// Strip outer parentheses if the entire predicate is wrapped.
			if strings.HasPrefix(raw, "(") && strings.HasSuffix(raw, ")") {
				inner := raw[1 : len(raw)-1]
				// Only strip if parentheses are balanced.
				if isBalancedParens(inner) {
					raw = inner
				}
			}
			whereClause = strings.TrimSpace(raw)
		}
	}

	return includeClause, whereClause
}

// isBalancedParens checks if a string has balanced parentheses.
func isBalancedParens(s string) bool {
	depth := 0
	for _, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return false
			}
		default:
		}
	}
	return depth == 0
}
