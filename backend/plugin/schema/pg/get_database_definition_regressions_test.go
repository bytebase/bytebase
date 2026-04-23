package pg

import (
	"strings"
	"testing"

	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestGetDatabaseDefinition_IndexCommentUsesRealNewline reproduces the A1
// bug where writeIndexComment used a Go raw string literal with `\n\n`,
// producing four literal characters `\`, `n`, `\`, `n` instead of two line
// feeds. The regression surfaced as omni pgparser choking on the backslash
// right after `COMMENT ON INDEX ... IS '...';`.
func TestGetDatabaseDefinition_IndexCommentUsesRealNewline(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "t",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "integer", Nullable: false},
				},
				Indexes: []*storepb.IndexMetadata{{
					Name:        "idx_t_id",
					Type:        "btree",
					Expressions: []string{"id"},
					Descending:  []bool{false},
					Definition:  "CREATE INDEX idx_t_id ON public.t USING btree (id);",
					Comment:     "sample comment",
				}},
			}},
		}},
	}

	ddl, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	if err != nil {
		t.Fatalf("GetDatabaseDefinition: %v", err)
	}

	if strings.Contains(ddl, `\n`) {
		t.Errorf("DDL contains literal backslash-n escape; got:\n%s", ddl)
	}
	if !strings.Contains(ddl, `COMMENT ON INDEX "public"."idx_t_id" IS 'sample comment';`) {
		t.Errorf("expected COMMENT ON INDEX statement, got:\n%s", ddl)
	}

	cat := catalog.New()
	if _, execErr := cat.Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); execErr != nil {
		t.Errorf("omni catalog parse error: %v\nDDL:\n%s", execErr, ddl)
	}
}

// TestGetDatabaseDefinition_TableWithOnlyCheckConstraints reproduces the A2
// bug where a table with zero columns but with CHECK constraints produced
// `CREATE TABLE "..."."..." (,` — an invalid leading comma. We force the
// case by declaring a CHECK constraint on a table with no regular columns.
func TestGetDatabaseDefinition_TableWithOnlyCheckConstraints(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "cons_only",
				CheckConstraints: []*storepb.CheckConstraintMetadata{{
					Name:       "cons_only_dummy",
					Expression: "(1 = 1)",
				}},
			}},
		}},
	}

	ddl, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	if err != nil {
		t.Fatalf("GetDatabaseDefinition: %v", err)
	}
	if strings.Contains(ddl, "(,") {
		t.Errorf("DDL still has `(,` leading comma; got:\n%s", ddl)
	}
	if !strings.Contains(ddl, `CREATE TABLE "public"."cons_only" (`) {
		t.Errorf("expected CREATE TABLE for cons_only, got:\n%s", ddl)
	}
	// Omni will reject a table with zero columns semantically, but it must
	// at least parse cleanly; the test asserts the parse-level contract.
	if _, parseErr := catalog.New().Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); parseErr != nil {
		t.Errorf("omni catalog parse error: %v\nDDL:\n%s", parseErr, ddl)
	}
}

// TestGetDatabaseDefinition_TableNameContainingDot reproduces A3: a table
// whose name literally contains a period was writing out as `"".".."` —
// because the schema + "." + object object-ID was split back on the first
// dot and produced an empty schema. The fix uses a NUL separator.
func TestGetDatabaseDefinition_TableNameContainingDot(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "weird.name",
				Columns: []*storepb.ColumnMetadata{
					{Name: "c", Type: "text", Nullable: true},
				},
			}},
		}},
	}

	ddl, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	if err != nil {
		t.Fatalf("GetDatabaseDefinition: %v", err)
	}
	if strings.Contains(ddl, `""."`) {
		t.Errorf("DDL contains empty schema `\"\".\"` prefix; got:\n%s", ddl)
	}
	if !strings.Contains(ddl, `CREATE TABLE "public"."weird.name"`) {
		t.Errorf("expected schema to be public, got:\n%s", ddl)
	}
	if _, parseErr := catalog.New().Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); parseErr != nil {
		t.Errorf("omni catalog parse error: %v\nDDL:\n%s", parseErr, ddl)
	}
}
