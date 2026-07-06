package mysql

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	// MySQL-only for now: OceanBase is intentionally NOT registered pending validation
	// against a live OceanBase oracle.
	schema.RegisterGetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, GetMultiFileDatabaseDefinition)
}

// GetMultiFileDatabaseDefinition emits the database schema as one SDL file per object,
// using a FLAT MySQL layout (MySQL has no schema namespace — schema == database):
//
//	tables/{table}.sql        CREATE TABLE + inline indexes/FKs/checks/generated cols/partitions/options
//	views/{view}.sql          CREATE OR REPLACE VIEW (own-database qualifier stripped)
//	functions/{fn}.sql        CREATE FUNCTION
//	procedures/{proc}.sql     CREATE PROCEDURE
//	triggers/{trigger}.sql    CREATE TRIGGER
//	events/{event}.sql        CREATE EVENT
//
// Each file's content is byte-identical to that object's slice of the single-file dump
// (getSDLFormat): the SAME per-object writers (writeTableSDL, writeRoutineSDL,
// writeViewSDL, writeTriggerSDL, writeEventSDL) produce the content, so single-file and
// multi-file always agree. Objects are emitted sorted by name for deterministic output,
// and each declarative rollout re-concatenates the files (action/command/file.go) into
// the identical statement set the single-file dump carries.
func GetMultiFileDatabaseDefinition(_ schema.GetDefinitionContext, metadata *storepb.DatabaseSchemaMetadata) (*schema.MultiFileSchemaResult, error) {
	if len(metadata.Schemas) == 0 {
		return &schema.MultiFileSchemaResult{Files: []schema.File{}}, nil
	}

	// The caller's *DatabaseSchemaMetadata may be a shared pointer (store.dbSchemaCache),
	// but every writer below only reads it — the single-file dump path reads the same
	// shared pointer without cloning — so no defensive deep-clone is taken.
	sm := metadata.Schemas[0]

	var files []schema.File
	names := newFileNameAllocator()

	// Tables (sorted by name). Each file is the writeTableSDL output — CREATE TABLE with
	// inline indexes / foreign keys / checks / generated columns / partitions / options.
	tables := make([]*storepb.TableMetadata, 0, len(sm.Tables))
	for _, table := range sm.Tables {
		if table.SkipDump {
			continue
		}
		tables = append(tables, table)
	}
	slices.SortFunc(tables, func(a, b *storepb.TableMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, table := range tables {
		var buf strings.Builder
		if err := writeTableSDL(&buf, table); err != nil {
			return nil, err
		}
		files = append(files, schema.File{
			Name:    names.alloc("tables", table.Name),
			Content: buf.String(),
		})
	}

	// Functions (sorted by name). writeRoutineSDL emits the canonical CREATE FUNCTION.
	functions := make([]*storepb.FunctionMetadata, 0, len(sm.Functions))
	for _, function := range sm.Functions {
		if function.SkipDump {
			continue
		}
		functions = append(functions, function)
	}
	slices.SortFunc(functions, func(a, b *storepb.FunctionMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, function := range functions {
		var buf strings.Builder
		if err := writeRoutineSDL(&buf, function.Definition); err != nil {
			return nil, err
		}
		// A routine whose definition is empty produces no content; skip its file so the
		// concatenation stays byte-identical to the single-file dump.
		if buf.Len() == 0 {
			continue
		}
		files = append(files, schema.File{
			Name:    names.alloc("functions", function.Name),
			Content: buf.String(),
		})
	}

	// Procedures (sorted by name). writeRoutineSDL emits the canonical CREATE PROCEDURE.
	procedures := make([]*storepb.ProcedureMetadata, 0, len(sm.Procedures))
	for _, procedure := range sm.Procedures {
		if procedure.SkipDump {
			continue
		}
		procedures = append(procedures, procedure)
	}
	slices.SortFunc(procedures, func(a, b *storepb.ProcedureMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, procedure := range procedures {
		var buf strings.Builder
		if err := writeRoutineSDL(&buf, procedure.Definition); err != nil {
			return nil, err
		}
		if buf.Len() == 0 {
			continue
		}
		files = append(files, schema.File{
			Name:    names.alloc("procedures", procedure.Name),
			Content: buf.String(),
		})
	}

	// Views (sorted by name). writeViewSDL strips the dumped database's own qualifier from
	// the body, matching the single-file form.
	views := make([]*storepb.ViewMetadata, 0, len(sm.Views))
	for _, view := range sm.Views {
		if view.SkipDump {
			continue
		}
		views = append(views, view)
	}
	slices.SortFunc(views, func(a, b *storepb.ViewMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, view := range views {
		var buf strings.Builder
		if err := writeViewSDL(&buf, metadata.Name, view); err != nil {
			return nil, err
		}
		files = append(files, schema.File{
			Name:    names.alloc("views", view.Name),
			Content: buf.String(),
		})
	}

	// Triggers hang off tables (TableMetadata.Triggers); emit them in deterministic
	// (table, trigger) order — the same traversal getSDLFormat uses. Each file carries a
	// single CREATE TRIGGER.
	for _, table := range tables {
		triggers := make([]*storepb.TriggerMetadata, 0, len(table.Triggers))
		for _, trigger := range table.Triggers {
			if trigger.SkipDump {
				continue
			}
			triggers = append(triggers, trigger)
		}
		slices.SortFunc(triggers, func(a, b *storepb.TriggerMetadata) int { return cmp.Compare(a.Name, b.Name) })
		for _, trigger := range triggers {
			var buf strings.Builder
			if err := writeTriggerSDL(&buf, table.Name, trigger); err != nil {
				return nil, err
			}
			files = append(files, schema.File{
				Name:    names.alloc("triggers", trigger.Name),
				Content: buf.String(),
			})
		}
	}

	// Events (sorted by name). writeEventSDL strips the DEFINER and emits the canonical
	// CREATE EVENT.
	events := make([]*storepb.EventMetadata, 0, len(sm.Events))
	events = append(events, sm.Events...)
	slices.SortFunc(events, func(a, b *storepb.EventMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, event := range events {
		var buf strings.Builder
		if err := writeEventSDL(&buf, event); err != nil {
			return nil, err
		}
		if buf.Len() == 0 {
			continue
		}
		files = append(files, schema.File{
			Name:    names.alloc("events", event.Name),
			Content: buf.String(),
		})
	}

	return &schema.MultiFileSchemaResult{Files: files}, nil
}

// fileNameAllocator maps object names to safe, unique "<dir>/<name>.sql" file paths.
// MySQL identifiers permit characters that are unsafe or ambiguous in a file path — '/'
// (path separator), backticks, spaces, dots, control chars — and are case sensitive on
// some platforms while file systems may be case insensitive. The allocator sanitizes the
// name into a safe stem and appends a numeric suffix on any collision (within a
// directory, case-insensitively) so distinct objects never share a path.
type fileNameAllocator struct {
	// taken holds every lower-cased "<dir>/<stem>" already handed out — both base stems
	// and generated "<stem>_N" suffixes — so a real object whose sanitized name equals a
	// previously generated suffix (e.g. tables `Foo`, `foo`, `foo_1`) cannot be assigned a
	// colliding path.
	taken map[string]bool
}

func newFileNameAllocator() *fileNameAllocator {
	return &fileNameAllocator{taken: make(map[string]bool)}
}

// alloc returns a unique "<dir>/<safe-name>.sql" path for objectName within dir. Every
// returned stem — the base stem AND any generated "<stem>_N" — is reserved, and generation
// retries until a truly-unused stem is found, so two objects never share a path even when a
// real name sanitizes to a value equal to another object's generated suffix.
func (a *fileNameAllocator) alloc(dir, objectName string) string {
	base := sanitizeFileStem(objectName)
	stem := base
	for n := 1; a.taken[strings.ToLower(dir+"/"+stem)]; n++ {
		stem = fmt.Sprintf("%s_%d", base, n)
	}
	a.taken[strings.ToLower(dir+"/"+stem)] = true
	return fmt.Sprintf("%s/%s.sql", dir, stem)
}

// sanitizeFileStem turns an object name into a file-name-safe stem. It replaces every
// character that is not a portable file-name character (ASCII letter, digit, '-', '_') —
// including '/', '\\', backticks, spaces, dots, and control bytes — with '_', collapses
// runs of '_' introduced by the replacement, and falls back to "_" for a name that
// sanitizes to empty. De-duplication of the resulting stems is handled by the caller.
func sanitizeFileStem(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	prevUnderscore := false
	for _, r := range name {
		safe := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if safe {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		// Collapse consecutive replacements so "a b" and "a  b" do not both become "a_b"
		// while "a__b" (a real name) is preserved as-is above.
		if !prevUnderscore {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	stem := b.String()
	// Trim replacement underscores that landed at the very edges (e.g. a leading '.' or a
	// trailing space) so the stem is tidy; interior and originally-present underscores stay.
	stem = strings.Trim(stem, "_")
	if stem == "" {
		return "_"
	}
	return stem
}
