package tidb

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Register the mysql completer (Engine_MYSQL and, pre-swap, Engine_TIDB) so
	// the no-regression test can compare the old ANTLR path against the new shim.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

func hasCandidate(cands []base.Candidate, typ base.CandidateType, text string) bool {
	for _, c := range cands {
		if c.Type == typ && c.Text == text {
			return true
		}
	}
	return false
}

func hasKeyword(cands []base.Candidate, text string) bool {
	for _, c := range cands {
		if c.Type == base.CandidateTypeKeyword && strings.EqualFold(c.Text, text) {
			return true
		}
	}
	return false
}

func metadataFunc(meta *storepb.DatabaseSchemaMetadata) base.GetDatabaseMetadataFunc {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
		return databaseName, model.NewDatabaseMetadata(meta, nil, nil, storepb.Engine_TIDB, true /* isObjectCaseSensitive */), nil
	}
}

func listDBNamesFunc(_ context.Context, _ string) ([]string, error) {
	return []string{"db"}, nil
}

// catchCaretLineColumn returns the SQL without the "|" caret marker and the
// 1-based line + 0-based column of the caret position.
func catchCaretLineColumn(s string) (string, int, int) {
	for i, c := range s {
		if c == '|' {
			text := s[:i] + s[i+1:]
			line := 1
			col := 0
			for _, ch := range s[:i] {
				if ch == '\n' {
					line++
					col = 0
				} else {
					col++
				}
			}
			return text, line, col
		}
	}
	return s, 1, -1
}

// candidateSet returns the non-empty candidates of the given types, keyed by
// type+text.
func candidateSet(cands []base.Candidate, types ...base.CandidateType) map[string]base.Candidate {
	want := make(map[base.CandidateType]bool, len(types))
	for _, t := range types {
		want[t] = true
	}
	m := make(map[string]base.Candidate)
	for _, c := range cands {
		if want[c.Type] && c.Text != "" {
			m[string(c.Type)+":"+c.Text] = c
		}
	}
	return m
}

// Reserved-word identifiers (table "order", columns "select"/"key") must surface
// as completion candidates AND their completion text must be backtick-quoted so
// accepting a suggestion inserts valid SQL. Surfacing proves the synthesized
// catalog DDL backticks identifiers; the quoted candidate text proves the shim
// quotes reserved object names on the way out.
func TestCompletion_ReservedWordIdentifiersSurface(t *testing.T) {
	meta := metadataFunc(&storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "order",
						Columns: []*storepb.ColumnMetadata{
							{Name: "select"},
							{Name: "key"},
						},
					},
				},
			},
		},
	})
	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "testdb",
		Metadata:        meta,
	}

	stmt := "SELECT * FROM "
	got, err := Completion(context.Background(), cCtx, stmt, 1, len(stmt))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeTable, "`order`"),
		"reserved-word table 'order' should surface quoted as a TABLE candidate; got %v", got)

	stmt2 := "SELECT  FROM `order`"
	got2, err := Completion(context.Background(), cCtx, stmt2, 1, len("SELECT "))
	require.NoError(t, err)
	require.True(t, hasCandidate(got2, base.CandidateTypeColumn, "`select`"),
		"reserved-word column 'select' should surface quoted as a COLUMN candidate; got %v", got2)
	require.True(t, hasCandidate(got2, base.CandidateTypeColumn, "`key`"),
		"reserved-word column 'key' should surface quoted as a COLUMN candidate; got %v", got2)
}

// A column whose stored type cannot be parsed must not silently vanish:
// buildCatalog retries the whole table with a generic column type, so every
// column name still surfaces. Without that retry the failing CREATE TABLE would
// drop the entire table from the catalog.
func TestCompletion_UnparseableColumnTypeFallsBackToGeneric(t *testing.T) {
	meta := metadataFunc(&storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{Name: "t", Columns: []*storepb.ColumnMetadata{
						{Name: "good", Type: "int"},
						{Name: "weird", Type: ")"}, // unparseable type — forces the generic retry
					}},
				},
			},
		},
	})
	cCtx := base.CompletionContext{Scene: base.SceneTypeAll, DefaultDatabase: "db", Metadata: meta}

	stmt := "SELECT  FROM t"
	got, err := Completion(context.Background(), cCtx, stmt, 1, len("SELECT "))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "good"),
		"column 'good' should surface; got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "weird"),
		"column 'weird' with an unparseable type should still surface via the generic-type retry; got %v", got)
}

// In the read-only query scene, write statements (DML/DDL incl. TiDB BATCH)
// must be filtered out, while read keywords (SELECT) stay.
func TestCompletion_QuerySceneFiltersWriteKeywords(t *testing.T) {
	meta := metadataFunc(&storepb.DatabaseSchemaMetadata{
		Name:    "testdb",
		Schemas: []*storepb.SchemaMetadata{{Name: ""}},
	})
	ccx := func(scene base.SceneType) base.CompletionContext {
		return base.CompletionContext{Scene: scene, DefaultDatabase: "testdb", Metadata: meta}
	}

	const stmt = "SELECT 1; "
	all, err := Completion(context.Background(), ccx(base.SceneTypeAll), stmt, 1, len(stmt))
	require.NoError(t, err)
	query, err := Completion(context.Background(), ccx(base.SceneTypeQuery), stmt, 1, len(stmt))
	require.NoError(t, err)

	// Read keywords are kept in both scenes.
	for _, kw := range []string{"SELECT", "WITH", "SHOW"} {
		require.True(t, hasKeyword(all, kw), "%s in ALL; got %v", kw, all)
		require.True(t, hasKeyword(query, kw), "%s must be kept in QUERY; got %v", kw, query)
	}

	// INSERT is a write keyword — present in ALL, dropped in QUERY.
	require.True(t, hasKeyword(all, "INSERT"), "INSERT in ALL; got %v", all)
	require.False(t, hasKeyword(query, "INSERT"), "INSERT must be dropped in QUERY; got %v", query)

	// BATCH (TiDB non-transactional DML) — present in ALL, dropped in QUERY.
	require.True(t, hasKeyword(all, "BATCH"), "BATCH in ALL; got %v", all)
	require.False(t, hasKeyword(query, "BATCH"), "BATCH must be dropped in QUERY; got %v", query)
}

// BATCH is TiDB-only grammar (added to omni in #157); the mysql ANTLR grammar
// cannot produce it. Its presence for Engine_TIDB and absence for Engine_MYSQL
// proves Engine_TIDB now routes through the omni shim, not the mysql completer.
func TestCompletion_TiDBRoutesThroughOmni(t *testing.T) {
	meta := metadataFunc(&storepb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*storepb.SchemaMetadata{{Name: ""}},
	})
	cCtx := base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "db",
		Metadata:          meta,
		ListDatabaseNames: listDBNamesFunc,
	}
	const stmt = "SELECT 1; "

	mysqlRes, err := base.Completion(context.Background(), storepb.Engine_MYSQL, cCtx, stmt, 1, len(stmt))
	require.NoError(t, err)
	tidbRes, err := base.Completion(context.Background(), storepb.Engine_TIDB, cCtx, stmt, 1, len(stmt))
	require.NoError(t, err)

	require.False(t, hasKeyword(mysqlRes, "BATCH"), "mysql ANTLR must not know BATCH")
	require.True(t, hasKeyword(tidbRes, "BATCH"), "Engine_TIDB must route through omni (BATCH present); got %v", tidbRes)
}

// The TiDB shim must not lose schema-object candidates relative to the prior
// mysql-ANTLR path, compared in SceneTypeAll so engine candidate deltas are
// isolated from scene filtering. The guarantee is context-appropriate:
//   - table/view contexts (after FROM/JOIN): no TABLE or VIEW lost.
//   - column contexts (projection, WHERE, after "alias."): no COLUMN lost.
//
// Classified, accepted strategy difference (NOT asserted): in column contexts
// the mysql ANTLR completer also offers DATABASE/TABLE *qualifier* names (for
// fully-qualified db.table.col refs) that omni omits, because omni resolves the
// in-scope columns directly instead. Tracked as a follow-up candidate gap.
func TestCompletion_NoCoreCandidateLossVsMySQL(t *testing.T) {
	meta := metadataFunc(&storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{Name: "t1", Columns: []*storepb.ColumnMetadata{{Name: "c1"}}},
					{Name: "t2", Columns: []*storepb.ColumnMetadata{{Name: "c1"}, {Name: "c2"}}},
				},
				Views: []*storepb.ViewMetadata{
					{Name: "v1", Definition: "CREATE VIEW v1 AS SELECT c1 FROM t1"},
				},
			},
		},
	})
	cCtx := base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "db",
		Metadata:          meta,
		ListDatabaseNames: listDBNamesFunc,
	}

	assertNoLoss := func(in string, types ...base.CandidateType) {
		text, line, col := catchCaretLineColumn(in)
		mysqlRes, err := base.Completion(context.Background(), storepb.Engine_MYSQL, cCtx, text, line, col)
		require.NoError(t, err)
		tidbRes, err := base.Completion(context.Background(), storepb.Engine_TIDB, cCtx, text, line, col)
		require.NoError(t, err)

		tidbSet := candidateSet(tidbRes, types...)
		for key, c := range candidateSet(mysqlRes, types...) {
			require.Contains(t, tidbSet, key,
				"input %q: TiDB dropped %v that MySQL produced; tidb=%v", in, c, tidbRes)
		}
	}

	// Table/view contexts: no TABLE or VIEW candidate may be lost.
	for _, in := range []string{
		"SELECT * FROM |",
		"SELECT * FROM t1 JOIN |",
		"SELECT * FROM t1, |",
	} {
		assertNoLoss(in, base.CandidateTypeTable, base.CandidateTypeView)
	}

	// Column contexts: no COLUMN candidate may be lost.
	for _, in := range []string{
		"SELECT | FROM t1",
		"SELECT t1.| FROM t1",
		"SELECT * FROM t1 WHERE |",
		"SELECT | FROM t1 JOIN t2 ON t1.c1 = t2.c1",
	} {
		assertNoLoss(in, base.CandidateTypeColumn)
	}
}

// Completion must be limited to the statement containing the caret: table refs
// from earlier statements in the buffer must not leak into the candidate set.
func TestCompletion_LimitsToStatementAtCaret(t *testing.T) {
	meta := metadataFunc(&storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{Name: "", Tables: []*storepb.TableMetadata{
			{Name: "t1", Columns: []*storepb.ColumnMetadata{{Name: "a1"}, {Name: "a2"}}},
			{Name: "t2", Columns: []*storepb.ColumnMetadata{{Name: "b1"}, {Name: "b2"}}},
		}}},
	})
	cCtx := base.CompletionContext{Scene: base.SceneTypeAll, DefaultDatabase: "db", Metadata: meta}

	stmt := "SELECT * FROM t1; SELECT  FROM t2"
	caret := len("SELECT * FROM t1; SELECT ") // projection of the 2nd statement
	got, err := Completion(context.Background(), cCtx, stmt, 1, caret)
	require.NoError(t, err)

	// t2's columns are in scope for the 2nd statement.
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "b1"), "t2.b1 should be in scope; got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "b2"), "t2.b2 should be in scope; got %v", got)
	// t1's columns must NOT leak from the earlier statement.
	require.False(t, hasCandidate(got, base.CandidateTypeColumn, "a1"), "t1.a1 must not leak into the 2nd statement; got %v", got)
	require.False(t, hasCandidate(got, base.CandidateTypeColumn, "a2"), "t1.a2 must not leak into the 2nd statement; got %v", got)
}

func TestQuoteIdentifierIfNeeded(t *testing.T) {
	cases := []struct {
		name            string
		caretInBacktick bool
		want            string
	}{
		{"users", false, "users"},       // bare identifier — unchanged
		{"t1", false, "t1"},             // bare with digit suffix — unchanged
		{"comment", false, "comment"},   // non-reserved keyword — stays bare
		{"order", false, "`order`"},     // reserved keyword
		{"select", false, "`select`"},   // reserved keyword
		{"Order", false, "`Order`"},     // reserved (case-insensitive), original case preserved
		{"foo-bar", false, "`foo-bar`"}, // non-bare character
		{"1col", false, "`1col`"},       // leading digit
		{"a`b", false, "`a``b`"},        // embedded backtick is doubled
		{"order", true, "order"},        // caret already inside a backtick — no extra quotes
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, quoteIdentifierIfNeeded(tc.name, tc.caretInBacktick),
			"quoteIdentifierIfNeeded(%q, %v)", tc.name, tc.caretInBacktick)
	}
}

func TestCaretInsideBacktickIdentifier(t *testing.T) {
	cases := []struct {
		statement string
		pos       int
		want      bool
	}{
		{"SELECT * FROM ", 14, false},  // no backticks
		{"SELECT * FROM `o", 16, true}, // one open backtick before caret
		{"SELECT `a`, ", 12, false},    // closed pair before caret
		{"`", 1, true},                 // single backtick
		{"ab", 99, false},              // pos clamped past end, no backticks
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, caretInsideBacktickIdentifier(tc.statement, tc.pos),
			"caretInsideBacktickIdentifier(%q, %d)", tc.statement, tc.pos)
	}
}

// Cross-database qualified completion: when a statement references a non-default
// database as a qualifier, buildCatalog must load that database so
// `other_db.tbl.col` resolves, and every known database name must surface as a
// DATABASE candidate. (The `FROM other_db.|` table-list case is a separate
// omni-side limitation — omni ignores the table qualifier — tracked as a
// follow-up, not asserted here.)
func TestCompletion_QualifiedColumnAcrossDatabases(t *testing.T) {
	appMeta := &storepb.DatabaseSchemaMetadata{
		Name: "appdb",
		Schemas: []*storepb.SchemaMetadata{{Name: "", Tables: []*storepb.TableMetadata{
			{Name: "t1", Columns: []*storepb.ColumnMetadata{{Name: "c1"}}},
		}}},
	}
	otherMeta := &storepb.DatabaseSchemaMetadata{
		Name: "otherdb",
		Schemas: []*storepb.SchemaMetadata{{Name: "", Tables: []*storepb.TableMetadata{
			{Name: "t2", Columns: []*storepb.ColumnMetadata{{Name: "c2"}, {Name: "c3"}}},
		}}},
	}
	metaFn := func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
		m := appMeta
		if databaseName == "otherdb" {
			m = otherMeta
		}
		return databaseName, model.NewDatabaseMetadata(m, nil, nil, storepb.Engine_TIDB, true), nil
	}
	cCtx := base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "appdb",
		Metadata:          metaFn,
		ListDatabaseNames: func(_ context.Context, _ string) ([]string, error) { return []string{"appdb", "otherdb"}, nil },
	}

	// Qualified column from a non-default database resolves.
	stmt := "SELECT otherdb.t2. FROM otherdb.t2"
	got, err := Completion(context.Background(), cCtx, stmt, 1, len("SELECT otherdb.t2."))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "c2"),
		"cross-db column 'c2' should surface; got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "c3"),
		"cross-db column 'c3' should surface; got %v", got)

	// Every known database name surfaces as a DATABASE candidate.
	stmt2 := "SELECT * FROM "
	got2, err := Completion(context.Background(), cCtx, stmt2, 1, len(stmt2))
	require.NoError(t, err)
	require.True(t, hasCandidate(got2, base.CandidateTypeDatabase, "otherdb"),
		"non-default database 'otherdb' should surface as a DATABASE candidate; got %v", got2)
}
