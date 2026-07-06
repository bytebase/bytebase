package mysql

// Unit tests for formatViewBodySDL, the whitespace-only pretty-printer the SDL dump
// applies to MySQL's one-line stored view bodies (writeViewSDL).
//
// Golden corpus: testdata/format-view-body/<case>.stored.sql holds a REAL SHOW CREATE
// VIEW line captured from a live oracle (8.0.32 / 5.7.25 — sakila, the stock sys
// schema, and hand-built edge views), and <case>.golden.sql pins the pretty body the
// dump emits for it (after stripViewBodyDatabaseQualifier, exactly like writeViewSDL).
//
// Properties (each golden case + adversarial inline bodies):
//   - token preservation: the quote/identifier-aware token stream of the output is
//     byte-identical to the input's — the transformation inserts/normalizes whitespace
//     BETWEEN tokens only, never inside a literal/identifier and never changing a token;
//   - idempotence: formatViewBodySDL(formatViewBodySDL(x)) == formatViewBodySDL(x)
//     (the A6/A8 dump-stability guards need same-input → same-bytes);
//   - quoted tokens containing keyword lookalikes (' from ', 'union all', aliases with
//     parens/commas/quotes) survive byte-for-byte.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// fmtViewStoredLineMatcher mirrors the MySQL driver's viewDefMatcher
// (backend/plugin/db/mysql/sync.go): it extracts the database name and the stored body
// from a SHOW CREATE VIEW line, so the golden tests replay the exact production
// pipeline (sync regex → stripViewBodyDatabaseQualifier → formatViewBodySDL).
var fmtViewStoredLineMatcher = regexp.MustCompile("CREATE ALGORITHM=(UNDEFINED|MERGE|TEMPTABLE) DEFINER=`([^`]+)`@`([^`]+)` SQL SECURITY (DEFINER|INVOKER) VIEW `([^`]+)`(?:\\.`([^`]+)`)?( \\(`(?:[^`]|``)+`(?:, ?`(?:[^`]|``)+`)*\\))? AS (?P<def>.+)")

// loadStoredViewCase reads one .stored.sql capture and returns (db, storedBody).
func loadStoredViewCase(t *testing.T, path string) (string, string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	line := strings.TrimRight(string(raw), "\n")
	m := fmtViewStoredLineMatcher.FindStringSubmatch(line)
	require.NotNil(t, m, "capture %s does not match the SHOW CREATE VIEW shape", path)
	return m[5], m[len(m)-1]
}

// viewBodyTokens splits a body into its quote/identifier-aware token stream using the
// same scanner the formatter uses; whitespace between tokens is discarded, whitespace
// INSIDE string literals and backtick identifiers is part of the token.
func viewBodyTokens(body string) []string {
	var toks []string
	for i := 0; i < len(body); {
		switch body[i] {
		case ' ', '\t', '\n', '\r':
			i++
			continue
		default:
		}
		end := scanViewBodyToken(body, i)
		toks = append(toks, body[i:end])
		i = end
	}
	return toks
}

// requireFormatProperties asserts the invariants every formatted body must satisfy.
func requireFormatProperties(t *testing.T, body string) string {
	t.Helper()
	got := formatViewBodySDL(body)

	require.Equal(t, viewBodyTokens(body), viewBodyTokens(got),
		"token stream must be preserved (whitespace-only transformation)")
	require.Equal(t, got, formatViewBodySDL(got), "formatViewBodySDL must be idempotent")
	require.Equal(t, got, formatViewBodySDL(body), "formatViewBodySDL must be deterministic")
	return got
}

func TestFormatViewBodySDLGolden(t *testing.T) {
	stored, err := filepath.Glob(filepath.Join("testdata", "format-view-body", "*.stored.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, stored, "no golden captures found")

	for _, path := range stored {
		name := strings.TrimSuffix(filepath.Base(path), ".stored.sql")
		t.Run(name, func(t *testing.T) {
			db, def := loadStoredViewCase(t, path)
			body := stripViewBodyDatabaseQualifier(def, db)

			golden, err := os.ReadFile(filepath.Join("testdata", "format-view-body", name+".golden.sql"))
			require.NoError(t, err)

			got := requireFormatProperties(t, body)
			require.Equal(t, strings.TrimRight(string(golden), "\n"), got)
		})
	}
}

func TestFormatViewBodySDLAdversarial(t *testing.T) {
	cases := []struct {
		name string
		body string
		// keep are byte sequences (quoted tokens with keyword lookalikes) that must
		// survive verbatim.
		keep []string
	}{
		{
			name: "literal_with_from_and_union",
			body: "select 'a from b' AS `x`,`t`.`b` AS `y` from `t` where (`t`.`b` <> 'union all')",
			keep: []string{"'a from b'", "'union all'"},
		},
		{
			name: "doubled_quote_escape",
			body: "select 'it''s from x' AS `x` from `t`",
			keep: []string{"'it''s from x'"},
		},
		{
			name: "backslash_escape",
			body: `select 'x\' from y' AS ` + "`x` from `t`",
			keep: []string{`'x\' from y'`},
		},
		{
			name: "identifier_named_from",
			body: "select 1 AS ` from ` from `t`",
			keep: []string{"` from `"},
		},
		{
			name: "identifier_with_doubled_backtick",
			body: "select `a``b`.`c` AS `x` from `a``b`",
			keep: []string{"`a``b`"},
		},
		{
			name: "alias_with_parens_quotes_commas",
			body: "select count(0) AS `COUNT(*)`,concat('INDEX (',`t`.`b`,')') AS `CONCAT('INDEX (', INDEX_TYPE, ')')` from `t`",
			keep: []string{"`COUNT(*)`", "`CONCAT('INDEX (', INDEX_TYPE, ')')`"},
		},
		{
			name: "double_quoted_string",
			body: `select "z leFT jOin q" AS ` + "`x` from `t`",
			keep: []string{`"z leFT jOin q"`},
		},
		{
			name: "unterminated_literal_tail",
			body: "select 'abc from `t`",
			keep: []string{"'abc from `t`"},
		},
		{
			name: "unbalanced_parens",
			body: "select concat(`t`.`b` AS `x` from `t`",
			keep: nil,
		},
		{
			name: "already_pretty_input_normalizes",
			body: "select\n  1 AS `x`,\n  2 AS `y`\nfrom `t`",
			keep: nil,
		},
		{
			name: "weird_run_whitespace",
			body: "select \t 1   AS  `x`  ,  2 AS `y`   from   `t`",
			keep: nil,
		},
		{
			name: "empty",
			body: "",
			keep: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := requireFormatProperties(t, tc.body)
			for _, k := range tc.keep {
				require.Contains(t, got, k, "quoted token must survive verbatim")
			}
		})
	}
}

// TestFormatViewBodySDLShapes pins the layout rules on small hand-written bodies where
// the whole output is easy to eyeball (the golden corpus covers the at-scale shapes).
func TestFormatViewBodySDLShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "minimal",
			body: "select 1 AS `one`",
			want: "select\n  1 AS `one`",
		},
		{
			name: "distinct_stays_on_select_line",
			body: "select distinct `t`.`b` AS `b` from `t`",
			want: "select distinct\n  `t`.`b` AS `b`\nfrom `t`",
		},
		{
			name: "join_tree",
			body: "select `a`.`x` AS `x` from ((`a` join `b` on((`a`.`i` = `b`.`i`))) left join `c` on((`b`.`j` = `c`.`j`)))",
			want: "select\n  `a`.`x` AS `x`\nfrom ((`a`\n  join `b` on((`a`.`i` = `b`.`i`)))\n  left join `c` on((`b`.`j` = `c`.`j`)))",
		},
		{
			name: "left_outer_join_stays_glued",
			body: "select `a`.`x` AS `x` from (`a` left outer join `b` on((`a`.`i` = `b`.`i`)))",
			want: "select\n  `a`.`x` AS `x`\nfrom (`a`\n  left outer join `b` on((`a`.`i` = `b`.`i`)))",
		},
		{
			name: "subquery_stays_inline",
			body: "select `a`.`x` AS `x`,(select max(`b`.`y`) from `b` where (`b`.`i` = `a`.`i`)) AS `m` from `a` group by `a`.`x`,`a`.`z` order by `a`.`x` desc limit 5",
			want: "select\n  `a`.`x` AS `x`,\n  (select max(`b`.`y`) from `b` where (`b`.`i` = `a`.`i`)) AS `m`\nfrom `a`\ngroup by `a`.`x`,`a`.`z`\norder by `a`.`x` desc\nlimit 5",
		},
		{
			name: "join_inside_derived_table_stays_inline",
			body: "select `dt`.`x` AS `x` from (select `a`.`x` AS `x` from (`a` join `b` on((`a`.`i` = `b`.`i`)))) `dt`",
			want: "select\n  `dt`.`x` AS `x`\nfrom (select `a`.`x` AS `x` from (`a` join `b` on((`a`.`i` = `b`.`i`)))) `dt`",
		},
		{
			name: "union_members_each_break",
			body: "select `a`.`x` AS `x` from `a` union all select `b`.`x` AS `x` from `b` order by `x`",
			want: "select\n  `a`.`x` AS `x`\nfrom `a`\nunion all\nselect\n  `b`.`x` AS `x`\nfrom `b`\norder by `x`",
		},
		{
			name: "group_concat_order_by_stays_inline",
			body: "select group_concat(`t`.`b` order by `t`.`b` ASC separator ', ') AS `g` from `t`",
			want: "select\n  group_concat(`t`.`b` order by `t`.`b` ASC separator ', ') AS `g`\nfrom `t`",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := requireFormatProperties(t, tc.body)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestWriteViewSDLPretty pins the full writeViewSDL statement framing around the pretty
// body: header, AS on the header line, body from column 0, terminating ";\n\n".
func TestWriteViewSDLPretty(t *testing.T) {
	db, def := loadStoredViewCase(t, filepath.Join("testdata", "format-view-body", "film_list_80.stored.sql"))
	golden, err := os.ReadFile(filepath.Join("testdata", "format-view-body", "film_list_80.golden.sql"))
	require.NoError(t, err)

	var buf strings.Builder
	require.NoError(t, writeViewSDL(&buf, db, &storepb.ViewMetadata{Name: "film_list", Definition: def}))
	want := "CREATE OR REPLACE VIEW `film_list` AS\n" + strings.TrimRight(string(golden), "\n") + ";\n\n"
	require.Equal(t, want, buf.String())
}

// TestMultiFileViewSDLPretty proves the multi-file export emits the identical pretty
// view content as the single-file dump (both share writeViewSDL — the multi-file ≡
// single-file concat invariant the live suite checks at scale).
func TestMultiFileViewSDLPretty(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{{
			Views: []*storepb.ViewMetadata{{
				Name:       "v_pretty",
				Definition: "select `testdb`.`t1`.`a` AS `a` from `testdb`.`t1`",
			}},
		}},
	}
	res, err := GetMultiFileDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	require.NoError(t, err)
	require.Len(t, res.Files, 1)
	require.Equal(t, "views/v_pretty.sql", res.Files[0].Name)
	require.Equal(t, "CREATE OR REPLACE VIEW `v_pretty` AS\nselect\n  `t1`.`a` AS `a`\nfrom `t1`;\n\n", res.Files[0].Content)

	single, err := getSDLFormat(meta)
	require.NoError(t, err)
	require.Contains(t, single, res.Files[0].Content, "single-file dump must contain the identical view slice")
}

// TestGetDatabaseMetadataParsesPrettyView proves the SDL→metadata parsers accept the
// pretty-printed view statement: the omni catalog path (GetDatabaseMetadataOmni, the
// registered production entry) parses a writeViewSDL-produced multi-line
// CREATE OR REPLACE VIEW.
func TestGetDatabaseMetadataParsesPrettyView(t *testing.T) {
	var buf strings.Builder
	buf.WriteString("CREATE TABLE `t1` (\n  `a` int NOT NULL,\n  `b` varchar(10) DEFAULT NULL,\n  PRIMARY KEY (`a`)\n) ENGINE=InnoDB;\n\n")
	require.NoError(t, writeViewSDL(&buf, "testdb", &storepb.ViewMetadata{
		Name:       "v_pretty",
		Definition: "select `testdb`.`t1`.`a` AS `a`,`testdb`.`t1`.`b` AS `b` from `testdb`.`t1` where (`testdb`.`t1`.`a` > 0)",
	}))
	sdl := buf.String()
	require.Contains(t, sdl, "AS\nselect\n  `t1`.`a` AS `a`,", "fixture must actually be pretty-printed")

	t.Run("omni", func(t *testing.T) {
		meta, err := GetDatabaseMetadataOmni(sdl)
		require.NoError(t, err)
		require.Len(t, meta.Schemas, 1)
		require.Len(t, meta.Schemas[0].Views, 1)
		view := meta.Schemas[0].Views[0]
		require.Equal(t, "v_pretty", view.Name)
		require.NotContains(t, view.Definition, "\n",
			"omni deparse must canonicalize the pretty body back to one line (whitespace washes out)")
	})
}

// TestFormatViewBodySDLFailSafe pins the runtime guard (X8). NOTE: live probes against
// 5.7.25 and 8.0.32 show the server ALWAYS re-prints stored view bodies in the
// backslash-escaped canonical form (even for views created under NO_BACKSLASH_ESCAPES),
// so no driver-synced body can trip the scanner today — the guard is defense-in-depth
// for bodies of other origins and for future scanner gaps.
func TestFormatViewBodySDLFailSafe(t *testing.T) {
	t.Run("no_backslash_escapes_style_body_left_unformatted", func(t *testing.T) {
		// Read under NO_BACKSLASH_ESCAPES semantics this is: literal 'a\' (trailing
		// backslash), then literal '  keep  me  '. The backslash-aware scanner instead
		// swallows \' as an escape, mis-tokenizes the second literal's interior as bare
		// words, and the unchecked formatter would collapse its double spaces — data
		// corruption. The trailing mis-scan leaves an unterminated literal, which the
		// guard detects: the body must come back byte-identical.
		body := `select 'a\', '  keep  me  ' AS s from t`
		require.NotEqual(t, body, formatViewBodyUnchecked(body),
			"precondition: the unchecked formatter must actually corrupt this body, else the case is vacuous")
		require.Equal(t, body, formatViewBodySDL(body))
	})

	t.Run("malformed_unterminated_literal_left_unformatted", func(t *testing.T) {
		body := "select 'unterminated from t"
		require.Equal(t, body, formatViewBodySDL(body))
	})

	t.Run("canonical_backslash_escaped_literal_still_formats", func(t *testing.T) {
		// The stored-canonical form ('x\'y', as SHOW CREATE prints) must keep
		// formatting — the guard must not regress legitimate bodies.
		body := `select 'x\'y' AS a,'z"q' AS b from t`
		got := requireFormatProperties(t, body)
		require.Equal(t, "select\n  'x\\'y' AS a,\n  'z\"q' AS b\nfrom t", got)
	})

	t.Run("normal_body_still_formats", func(t *testing.T) {
		got := requireFormatProperties(t, "select `a`.`x` AS `x` from `a`")
		require.Equal(t, "select\n  `a`.`x` AS `x`\nfrom `a`", got)
	})
}
