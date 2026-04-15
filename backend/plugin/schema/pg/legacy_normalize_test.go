package pg

import (
	"strings"
	"testing"

	omnipg "github.com/bytebase/omni/pg"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestStripMatchedOuterParens(t *testing.T) {
	cases := []struct {
		in       string
		want     string
		stripped bool
	}{
		{"(a)", "a", true},
		{"(payload ->> 'k'::text)", "payload ->> 'k'::text", true},
		{"((a))", "(a)", true},
		{"(a + b)", "a + b", true},
		{"a + b", "a + b", false},
		{"(a)+(b)", "(a)+(b)", false},
		{"('(' || name || ')')", "'(' || name || ')'", true},
		{`("name" || ' x')`, `"name" || ' x'`, true},
		{"(", "(", false},
		{")", ")", false},
		{"", "", false},
		{"(a", "(a", false},
		{"a)", "a)", false},
	}
	for _, c := range cases {
		got, ok := stripMatchedOuterParens(c.in)
		if got != c.want || ok != c.stripped {
			t.Errorf("stripMatchedOuterParens(%q) = (%q, %v); want (%q, %v)",
				c.in, got, ok, c.want, c.stripped)
		}
	}
}

func TestIsBareColumnIdent(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"name", true},
		{"  name  ", true},
		{"_col", true},
		{"col1", true},
		// PostgreSQL allows `$` in unquoted identifiers after the first
		// character. Columns like `col$1` must be classified as bare
		// identifiers — otherwise PK/UNIQUE constraint emission on such
		// columns would wrap them as expression keys, which PG rejects.
		// Codex review (PR #20009).
		{"col$1", true},
		{"a$b$c", true},
		{"$col", false}, // leading '$' is not allowed
		// PostgreSQL also allows Unicode letters in unquoted identifiers.
		// Misclassifying these as expressions would break PK/UNIQUE emission
		// on columns like `naïve`. Codex review (PR #20009).
		{"naïve", true},
		{"café", true},
		{"名前", true},
		{"1col", false}, // digit start still rejected
		{`"Name"`, true},
		{`"has ""quote"" inside"`, true},
		{"name + 1", false},
		{"payload ->> 'k'", false},
		{"(name)", false},
		{"lower(name)", false},
		{"name.other", false},
		{"1name", false},
		{"", false},
		{`""`, false},
		{`"a`, false},
		{`a"`, false},
	}
	for _, c := range cases {
		if got := isBareColumnIdent(c.s); got != c.want {
			t.Errorf("isBareColumnIdent(%q) = %v; want %v", c.s, got, c.want)
		}
	}
}

func TestIsBareFunctionCall(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"lower(name)", true},
		{"coalesce(a, b)", true},
		{"substring(name, 1, 3)", true},
		{"public.foo(a)", true},
		{"lower (name)", true},     // whitespace before '(' is allowed
		{"  lower(name)  ", true},  // surrounding whitespace trimmed
		{"name", false},            // bare ident, not a call
		{"name + 1", false},        // operator expression
		{"(lower(name))", false},   // outer parens — not bare
		{"lower(name) + 1", false}, // not terminated with ')'
		{"(name)", false},          // starts with '('
		{"", false},
		{"1lower(name)", false}, // digit start
		// Compound expressions that happen to end with ')' — the '(' after
		// the leading ident does NOT match the trailing ')'. Misclassifying
		// these as bare calls would leave legacy stripped-parens entries
		// unwrapped, producing invalid CREATE INDEX SQL. Codex review
		// (PR #20009 r3082599144).
		{"lower(name) + abs(score)", false},
		{"abs(x) - abs(y)", false},
		{"f() OR g()", false},
		{"coalesce(a, b) || coalesce(c, d)", false},
		// True nested-call cases stay correctly classified.
		{"coalesce(lower(name), upper(name))", true},
		{"f(g(h(x)))", true},
	}
	for _, c := range cases {
		if got := isBareFunctionCall(c.s); got != c.want {
			t.Errorf("isBareFunctionCall(%q) = %v; want %v", c.s, got, c.want)
		}
	}
}

func TestCanonicalizeIndexKeyExpression(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Canonical forms — round-trip unchanged.
		{"id", "id"},
		{`"Name"`, `"Name"`},
		{"lower(name)", "lower(name)"},
		{"coalesce(a, b)", "coalesce(a, b)"},
		{"(payload ->> 'k'::text)", "(payload ->> 'k'::text)"},
		{"(id + 1)", "(id + 1)"},
		{"(name::text)", "(name::text)"},

		// Legacy stripped-parens expressions — wrap.
		{"payload ->> 'k'::text", "(payload ->> 'k'::text)"},
		{"id + 1", "(id + 1)"},
		{"name::text", "(name::text)"},
		{"a || b || c", "(a || b || c)"},
		{"a @> b", "(a @> b)"},

		// Over-wrapped — collapse to single pair.
		{"((payload ->> 'k'))", "(payload ->> 'k')"},
		{"((id + 1))", "(id + 1)"},
		{"(lower(name))", "lower(name)"}, // stripped → bare func call

		// Whitespace tolerance.
		{"  id  ", "id"},
		{"  (a + b)  ", "(a + b)"},

		// Compound expressions involving function calls — must wrap, even
		// though the input starts with `ident(` and ends with `)`.
		// PR #20009 r3082599144.
		{"lower(name) + abs(score)", "(lower(name) + abs(score))"},
		{"abs(x) - abs(y)", "(abs(x) - abs(y))"},
		{"f() OR g()", "(f() OR g())"},

		// Empty / edge.
		{"", ""},
	}
	for _, c := range cases {
		if got := canonicalizeIndexKeyExpression(c.in); got != c.want {
			t.Errorf("canonicalizeIndexKeyExpression(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

// TestIndexEmissionParses is the regression lock for BYT-9261: for every
// expression family that can appear in a functional index, the DDL produced by
// GetDatabaseDefinition must parse cleanly through omni — whether the stored
// form is canonical (pg_get_indexdef-native) or legacy (stripped parens).
func TestIndexEmissionParses(t *testing.T) {
	exprs := []string{
		// Identifier keys — stored bare, emitted bare.
		"name",
		`"Name"`,
		// Function-call keys — stored bare, emitted bare.
		"lower(name)",
		"coalesce(a, b)",
		// Operator/expression keys — canonical form is parenthesized. Legacy
		// data strips the parens.
		"payload ->> 'k'::text",
		"payload -> 'k'",
		"payload #> '{a,b}'",
		"payload #>> '{a,b}'",
		"name::text",
		"first_name || ' ' || last_name",
		"a + 1",
		"a - b",
		"a = 1",
		"name ~ '^foo'",
		"name ~* 'foo'",
		`payload @> '{"k":1}'::jsonb`,
		"tags <@ ARRAY['a','b']",
	}

	for _, expr := range exprs {
		shapes := []struct {
			name, stored string
		}{
			{"bare (legacy or identifier/func)", expr},
			{"parenthesized (canonical for expressions)", "(" + expr + ")"},
		}
		for _, shape := range shapes {
			meta := &storepb.DatabaseSchemaMetadata{
				Name: "db",
				Schemas: []*storepb.SchemaMetadata{{
					Name: "public",
					Tables: []*storepb.TableMetadata{{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "a", Position: 1, Type: "integer"},
							{Name: "b", Position: 2, Type: "integer"},
							{Name: "name", Position: 3, Type: "text"},
							{Name: "first_name", Position: 4, Type: "text"},
							{Name: "last_name", Position: 5, Type: "text"},
							{Name: "tags", Position: 6, Type: "text[]"},
							{Name: "payload", Position: 7, Type: "jsonb"},
						},
						Indexes: []*storepb.IndexMetadata{{
							Name:        "idx",
							Type:        "btree",
							Expressions: []string{shape.stored},
						}},
					}},
				}},
			}
			ddl, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
			if err != nil {
				t.Fatalf("GetDatabaseDefinition(expr=%q, shape=%s): %v", expr, shape.name, err)
			}
			if _, err := omnipg.Parse(ddl); err != nil {
				t.Errorf("omni parse failed for expr=%q shape=%s:\n  DDL: %s\n  err: %v",
					expr, shape.name, indexLine(ddl), err)
			}
		}
	}
}

// TestNormalizeLegacyMetadata_PartitionAndMV locks the traversal of
// TablePartitionMetadata.Indexes (including nested Subpartitions) and
// MaterializedViewMetadata.Indexes — which are structurally easy to miss.
func TestNormalizeLegacyMetadata_PartitionAndMV(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "t",
				Partitions: []*storepb.TablePartitionMetadata{{
					Name: "t_p1",
					Indexes: []*storepb.IndexMetadata{{
						Name:        "partition_idx",
						Expressions: []string{"payload ->> 'k'::text"}, // legacy: no parens
					}},
					Subpartitions: []*storepb.TablePartitionMetadata{{
						Name: "t_p1_sub",
						Indexes: []*storepb.IndexMetadata{{
							Name:        "subpartition_idx",
							Expressions: []string{"a + b"}, // legacy: no parens
						}},
					}},
				}},
			}},
			MaterializedViews: []*storepb.MaterializedViewMetadata{{
				Name: "mv",
				Indexes: []*storepb.IndexMetadata{{
					Name:        "mv_idx",
					Expressions: []string{"lower(name)"}, // canonical: bare func call
				}},
			}},
		}},
	}

	normalizeLegacyMetadata(meta)

	got := []string{
		meta.Schemas[0].Tables[0].Partitions[0].Indexes[0].Expressions[0],
		meta.Schemas[0].Tables[0].Partitions[0].Subpartitions[0].Indexes[0].Expressions[0],
		meta.Schemas[0].MaterializedViews[0].Indexes[0].Expressions[0],
	}
	want := []string{
		"(payload ->> 'k'::text)", // wrapped
		"(a + b)",                 // wrapped
		"lower(name)",             // unchanged
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("slot %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestGetMultiFileDatabaseDefinition_LegacyIndexExpressionsParse locks the
// multi-file / SDL export path: legacy stripped-parens expressions stored on
// tables, partitions, and materialized views must all emit as parseable SQL
// through GetMultiFileDatabaseDefinition, which has its own entry point and
// was previously missing the normalization pass (Codex review, PR #20009).
func TestGetMultiFileDatabaseDefinition_LegacyIndexExpressionsParse(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "t",
				Columns: []*storepb.ColumnMetadata{
					{Name: "a", Position: 1, Type: "integer"},
					{Name: "b", Position: 2, Type: "integer"},
					{Name: "payload", Position: 3, Type: "jsonb"},
				},
				Indexes: []*storepb.IndexMetadata{{
					Name:        "t_legacy_idx",
					Type:        "btree",
					Expressions: []string{"payload ->> 'k'::text"}, // legacy
				}},
				Partitions: []*storepb.TablePartitionMetadata{{
					Name: "t_p1",
					Indexes: []*storepb.IndexMetadata{{
						Name:        "t_p1_legacy_idx",
						Type:        "btree",
						Expressions: []string{"a + b"}, // legacy
					}},
				}},
			}},
			MaterializedViews: []*storepb.MaterializedViewMetadata{{
				Name:       "mv",
				Definition: "SELECT 1 AS x",
				Indexes: []*storepb.IndexMetadata{{
					Name:        "mv_legacy_idx",
					Type:        "btree",
					Expressions: []string{"payload ->> 'k'::text"}, // legacy
				}},
			}},
		}},
	}

	result, err := GetMultiFileDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	if err != nil {
		t.Fatalf("GetMultiFileDatabaseDefinition: %v", err)
	}

	var sawIndex bool
	for _, f := range result.Files {
		if !strings.Contains(f.Content, "CREATE INDEX") && !strings.Contains(f.Content, "CREATE UNIQUE INDEX") {
			continue
		}
		sawIndex = true
		if _, err := omnipg.Parse(f.Content); err != nil {
			t.Errorf("omni parse failed for %s:\n%s\nerr: %v", f.Name, f.Content, err)
		}
	}
	if !sawIndex {
		t.Fatal("no CREATE INDEX emitted in multi-file output; test setup is wrong")
	}
}

// TestGetDatabaseDefinition_DoesNotMutateInput guards the proto.Clone at the
// top of GetDatabaseDefinition. The caller's metadata (often a shared pointer
// from store.dbSchemaCache) must not be altered by the normalization pass.
func TestGetDatabaseDefinition_DoesNotMutateInput(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "t",
				Columns: []*storepb.ColumnMetadata{
					{Name: "payload", Position: 1, Type: "jsonb"},
				},
				Indexes: []*storepb.IndexMetadata{{
					Name: "idx",
					Type: "btree",
					// Deliberately non-canonical: the normalizer would rewrite this.
					Expressions: []string{"payload ->> 'k'"},
				}},
			}},
		}},
	}
	before := meta.Schemas[0].Tables[0].Indexes[0].Expressions[0]

	if _, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta); err != nil {
		t.Fatal(err)
	}

	after := meta.Schemas[0].Tables[0].Indexes[0].Expressions[0]
	if before != after {
		t.Errorf("caller's metadata was mutated: before=%q, after=%q", before, after)
	}
}

func indexLine(ddl string) string {
	for ln := range strings.SplitSeq(ddl, "\n") {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "CREATE INDEX") || strings.HasPrefix(trimmed, "CREATE UNIQUE INDEX") {
			return trimmed
		}
	}
	return ""
}
