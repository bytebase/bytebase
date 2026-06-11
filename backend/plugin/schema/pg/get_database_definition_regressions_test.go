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
	// A genuine zero-column table (no indexes or foreign keys) must keep its
	// column-independent CHECK constraint — only privilege-filtered broken
	// metadata goes down the bare-table path.
	if !strings.Contains(ddl, "cons_only_dummy") {
		t.Errorf("expected CHECK constraint cons_only_dummy to be preserved, got:\n%s", ddl)
	}
	// Omni will reject a table with zero columns semantically, but it must
	// at least parse cleanly; the test asserts the parse-level contract.
	if _, parseErr := catalog.New().Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); parseErr != nil {
		t.Errorf("omni catalog parse error: %v\nDDL:\n%s", parseErr, ddl)
	}
}

// TestGetDatabaseDefinition_TableWithConstraintsButNoColumns reproduces a
// dump replay failure observed on a real workspace export: the sync user
// lacked column privileges on some tables, so information_schema.columns
// returned nothing while pg_catalog still exposed the tables' constraints.
// The dump then emitted `CREATE TABLE "s1"."t1" ();` followed by
// `ALTER TABLE ... ADD CONSTRAINT ... PRIMARY KEY (c1, c2);`, which
// PostgreSQL rejects because the columns do not exist. The dump must skip
// column-dependent DDL for such tables instead of producing non-replayable
// statements.
func TestGetDatabaseDefinition_TableWithConstraintsButNoColumns(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "s1",
			Sequences: []*storepb.SequenceMetadata{
				{
					// Owned by a column missing from t1's metadata: the sequence
					// itself must dump, its OWNED BY clause must not.
					Name:        "t1_c1_seq",
					DataType:    "bigint",
					Start:       "1",
					Increment:   "1",
					MinValue:    "1",
					MaxValue:    "9223372036854775807",
					OwnerTable:  "t1",
					OwnerColumn: "c1",
				},
				{
					// Owned by a column of t4, a zero-column table with no
					// broken-metadata signal (no indexes/FKs/partitions):
					// OWNED BY must still be skipped — a sequence cannot be
					// owned by a column of a zero-column table.
					Name:        "t4_id_seq",
					DataType:    "bigint",
					Start:       "1",
					Increment:   "1",
					MinValue:    "1",
					MaxValue:    "9223372036854775807",
					OwnerTable:  "t4",
					OwnerColumn: "id",
				},
			},
			Tables: []*storepb.TableMetadata{
				{
					// Broken sync: constraints and indexes, but no columns.
					Name: "t1",
					Indexes: []*storepb.IndexMetadata{
						{
							Name:         "pk_t1",
							Primary:      true,
							Unique:       true,
							IsConstraint: true,
							Expressions:  []string{"c1", "c2"},
						},
						{
							Name:         "uk_t1_c1",
							Unique:       true,
							IsConstraint: true,
							Expressions:  []string{"c1"},
						},
						{
							Name:        "idx_t1_c2",
							Type:        "btree",
							Expressions: []string{"c2"},
							Definition:  `CREATE INDEX idx_t1_c2 ON s1.t1 USING btree (c2);`,
							Comment:     "comment on a never-created index",
						},
					},
					CheckConstraints: []*storepb.CheckConstraintMetadata{{
						Name:       "chk_t1_c1",
						Expression: "(c1 > 0)",
					}},
					ForeignKeys: []*storepb.ForeignKeyMetadata{{
						Name:              "fk_t1_t2",
						Columns:           []string{"c1"},
						ReferencedSchema:  "s1",
						ReferencedTable:   "t2",
						ReferencedColumns: []string{"id"},
					}},
					Triggers: []*storepb.TriggerMetadata{{
						Name:    "trg_t1_c1",
						Body:    `CREATE TRIGGER trg_t1_c1 BEFORE UPDATE OF c1 ON s1.t1 FOR EACH ROW EXECUTE FUNCTION s1.f()`,
						Comment: "comment on a never-created trigger",
					}},
					Rules: []*storepb.RuleMetadata{{
						Name:       "rule_t1",
						Event:      "INSERT",
						Definition: `CREATE RULE rule_t1 AS ON INSERT TO s1.t1 WHERE new.c1 > 0 DO INSTEAD NOTHING;`,
					}},
				},
				{
					// Healthy table; its own constraints must still dump, but
					// its foreign key into the broken table must not.
					Name: "t2",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "integer", Nullable: false},
						{Name: "t1_c1", Type: "integer", Nullable: true},
					},
					Indexes: []*storepb.IndexMetadata{{
						Name:         "pk_t2",
						Primary:      true,
						Unique:       true,
						IsConstraint: true,
						Expressions:  []string{"id"},
					}},
					ForeignKeys: []*storepb.ForeignKeyMetadata{{
						Name:              "fk_t2_t1",
						Columns:           []string{"t1_c1"},
						ReferencedSchema:  "s1",
						ReferencedTable:   "t1",
						ReferencedColumns: []string{"c1"},
					}},
				},
				{
					// Broken sync on a partitioned table with no indexes: the
					// partition key references missing columns, so the table
					// must dump bare, without the PARTITION BY clause.
					Name: "t3",
					Partitions: []*storepb.TablePartitionMetadata{{
						Name:       "t3_p1",
						Expression: "RANGE (created_at)",
					}},
				},
				{
					// Genuine zero-column table: no indexes, foreign keys, or
					// partitions. Its column-independent CHECK must be kept.
					Name: "t4",
					CheckConstraints: []*storepb.CheckConstraintMetadata{{
						Name:       "chk_t4_ok",
						Expression: "(1 = 1)",
					}},
				},
			},
		}},
	}

	banned := []string{"pk_t1", "uk_t1_c1", "idx_t1_c2", "chk_t1_c1", "fk_t1_t2", "fk_t2_t1", "trg_t1_c1", "rule_t1", "t3_p1", "PARTITION BY", "OWNED BY", "COMMENT ON INDEX"}

	for name, ctx := range map[string]schema.GetDefinitionContext{
		"dump": {},
		"sdl":  {SDLFormat: true},
	} {
		ddl, err := GetDatabaseDefinition(ctx, meta)
		if err != nil {
			t.Fatalf("[%s] GetDatabaseDefinition: %v", name, err)
		}

		if !strings.Contains(ddl, `CREATE TABLE "s1"."t1" (`) {
			t.Errorf("[%s] expected bare CREATE TABLE for t1, got:\n%s", name, ddl)
		}
		for _, b := range banned {
			if strings.Contains(ddl, b) {
				t.Errorf("[%s] DDL references %q, which depends on columns missing from t1's metadata; got:\n%s", name, b, ddl)
			}
		}
		if !strings.Contains(ddl, "pk_t2") {
			t.Errorf("[%s] expected pk_t2 on the healthy table, got:\n%s", name, ddl)
		}
		if !strings.Contains(ddl, `CREATE SEQUENCE "s1"."t1_c1_seq"`) {
			t.Errorf("[%s] expected CREATE SEQUENCE for t1_c1_seq, got:\n%s", name, ddl)
		}
		if !strings.Contains(ddl, `CREATE TABLE "s1"."t3" (`) {
			t.Errorf("[%s] expected bare CREATE TABLE for t3, got:\n%s", name, ddl)
		}
		if !strings.Contains(ddl, "chk_t4_ok") {
			t.Errorf("[%s] expected genuine zero-column table t4 to keep its CHECK constraint, got:\n%s", name, ddl)
		}

		if _, parseErr := catalog.New().Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); parseErr != nil {
			t.Errorf("[%s] omni catalog parse error: %v\nDDL:\n%s", name, parseErr, ddl)
		}
	}

	multiFile, err := GetMultiFileDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	if err != nil {
		t.Fatalf("GetMultiFileDatabaseDefinition: %v", err)
	}
	var combined strings.Builder
	for _, file := range multiFile.Files {
		combined.WriteString(file.Content)
	}
	for _, b := range banned {
		if strings.Contains(combined.String(), b) {
			t.Errorf("[multi-file] DDL references %q, which depends on columns missing from t1's metadata; got:\n%s", b, combined.String())
		}
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
