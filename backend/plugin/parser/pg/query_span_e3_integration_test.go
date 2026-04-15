package pg

import (
	"context"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ---------------------------------------------------------------------------
// E3 integration tests
// ---------------------------------------------------------------------------
//
// These tests drive the full GetQuerySpan pipeline (parse → initCatalog via
// e3Loader → AnalyzeSelectStmt → extractLineage) against hand-built
// DatabaseSchemaMetadata fixtures that reproduce the bugs E3 is designed to
// prevent, plus the cascade-prevention story.
//
// Each test asserts one of:
//   - A query against an unrelated object succeeds even when a bad object is
//     present in the schema (BYT-9215 / BYT-9261 class, blast-radius).
//   - A query against a broken-but-pseudo'd object still returns lineage.
//   - A cascade chain (bad root → dependent table → dependent view) installs
//     downstream objects as real, with query lineage intact.

func TestE3Integration_BYT9215_BadQuotedIdentifier(t *testing.T) {
	// BYT-9215 class: a table name contains a character sequence the DDL
	// deparse-and-reparse loop chokes on. Under the old path this would kill
	// query span for the entire database. Under E3, the bad table is
	// pseudo-installed and unrelated queries succeed.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{
				// A table with a quoted identifier containing an apostrophe —
				// the kind of input BYT-9215 reported failing under Exec(ddl).
				// Real install should succeed (no DDL roundtrip), but even if
				// it didn't, pseudo would catch it.
				{
					Name: "'weird'table",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "int4"},
					},
				},
				// An unrelated healthy table.
				{
					Name: "accounts",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "int4"},
						{Name: "email", Type: "text"},
					},
				},
			},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, email FROM accounts`)
	if len(span.Results) != 2 {
		t.Fatalf("accounts query: got %d results, want 2", len(span.Results))
	}
	for _, r := range span.Results {
		if len(r.SourceColumns) == 0 {
			t.Errorf("result %q has empty sources (lineage lost)", r.Name)
		}
	}
}

func TestE3Integration_BrokenEnumCascade(t *testing.T) {
	// Real failure chain: table references a user-defined type that does
	// not exist in metadata. Real install of the table fails (omni cannot
	// resolve the type). Pseudo install of the table succeeds with text
	// columns. Query against the table returns its metadata column names.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "tasks",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					// References an enum that is NOT declared in metadata —
					// buildCreateStmt will succeed (typeNameFromString works)
					// but DefineRelation fails (enum not in catalog).
					{Name: "status", Type: "public.nonexistent_enum"},
					{Name: "title", Type: "text"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, status, title FROM tasks`)
	if len(span.Results) != 3 {
		t.Fatalf("tasks query: got %d results, want 3", len(span.Results))
	}
	wantNames := []string{"id", "status", "title"}
	for i, want := range wantNames {
		if i >= len(span.Results) {
			break
		}
		if span.Results[i].Name != want {
			t.Errorf("result[%d]: got %q, want %q", i, span.Results[i].Name, want)
		}
		if len(span.Results[i].SourceColumns) == 0 {
			t.Errorf("result[%d] (%s): empty sources", i, span.Results[i].Name)
		}
	}
}

func TestE3Integration_BrokenRootTableAndHealthyNeighbor(t *testing.T) {
	// A query against a healthy table must succeed even when an unrelated
	// table in the same schema references a broken type. This is the core
	// blast-radius claim: one bad object does not poison all queries.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{
				{
					Name: "broken",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "int4"},
						{Name: "bad", Type: "public.nonexistent_type"},
					},
				},
				{
					Name: "healthy",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "int4"},
						{Name: "label", Type: "text"},
					},
				},
			},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, label FROM healthy`)
	if len(span.Results) != 2 {
		t.Fatalf("healthy query: got %d results, want 2", len(span.Results))
	}
	for _, r := range span.Results {
		if len(r.SourceColumns) == 0 {
			t.Errorf("result %q lost lineage because of unrelated broken table", r.Name)
		}
	}
}

func TestE3Integration_ViewOverBrokenTableStillResolves(t *testing.T) {
	// Chain: enum missing → table T references it (degrades to pseudo) →
	// view V on T installs real (against pseudo T) → query on V resolves.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "records",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "status", Type: "public.nonexistent_enum"},
					{Name: "title", Type: "text"},
				},
			}},
			Views: []*storepb.ViewMetadata{{
				Name:       "records_view",
				Definition: "SELECT id, title, status FROM records",
				DependencyColumns: []*storepb.DependencyColumn{
					{Schema: "public", Table: "records", Column: "id"},
					{Schema: "public", Table: "records", Column: "title"},
					{Schema: "public", Table: "records", Column: "status"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, title, status FROM records_view`)
	if len(span.Results) != 3 {
		t.Fatalf("view query: got %d results, want 3", len(span.Results))
	}
	for i, r := range span.Results {
		if len(r.SourceColumns) == 0 {
			t.Errorf("result[%d] (%s) lost lineage", i, r.Name)
		}
	}
}

func TestE3Integration_SimpleHealthyPath(t *testing.T) {
	// Baseline sanity: a clean schema must produce exact lineage down to
	// (schema, table, column) — no degraded flags.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "users",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "email", Type: "text"},
					{Name: "created_at", Type: "timestamp with time zone"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, email FROM users WHERE id > 0`)
	if len(span.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(span.Results))
	}
	mustHaveExactSource(t, span.Results[0], "users", "id")
	mustHaveExactSource(t, span.Results[1], "users", "email")
}

func TestE3Integration_EnumWorksWhenDeclared(t *testing.T) {
	// When an enum IS declared in metadata, the table installs as real and
	// enum-typed columns resolve through their real type. This is the
	// positive control for TestE3Integration_BrokenEnumCascade.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			EnumTypes: []*storepb.EnumTypeMetadata{{
				Name:   "task_status",
				Values: []string{"pending", "running", "done"},
			}},
			Tables: []*storepb.TableMetadata{{
				Name: "tasks",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "status", Type: "public.task_status"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, status FROM tasks WHERE status = 'pending'`)
	if len(span.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(span.Results))
	}
	mustHaveExactSource(t, span.Results[0], "tasks", "id")
	mustHaveExactSource(t, span.Results[1], "tasks", "status")
}

// ---------- helpers ----------

// e3IntegrationDB / e3IntegrationSchema are the fixed database/schema names
// every E3 integration test uses; keeping them as constants keeps the call
// sites terse and satisfies the unparam linter.
const (
	e3IntegrationDB     = "db"
	e3IntegrationSchema = "public"
)

func mustGetQuerySpan(t *testing.T, meta *storepb.DatabaseSchemaMetadata, sql string) *base.QuerySpan {
	t.Helper()
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{meta})
	span, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	}, base.Statement{Text: sql}, e3IntegrationDB, "", false)
	if err != nil {
		t.Fatalf("GetQuerySpan(%q): %v", sql, err)
	}
	if span == nil {
		t.Fatalf("GetQuerySpan(%q): nil span", sql)
	}
	return span
}

func mustHaveExactSource(t *testing.T, result base.QuerySpanResult, table, column string) {
	t.Helper()
	want := base.ColumnResource{
		Database: e3IntegrationDB,
		Schema:   e3IntegrationSchema,
		Table:    table,
		Column:   column,
	}
	for src := range result.SourceColumns {
		if src == want {
			return
		}
	}
	t.Errorf("result %q missing source %+v; have %+v", result.Name, want, result.SourceColumns)
}
