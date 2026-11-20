package mysql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

func metadataFunc(meta *metadatapb.DatabaseSchemaMetadata) base.GetDatabaseMetadataFunc {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
		return databaseName, model.NewDatabaseMetadata(meta, nil, nil, storepb.Engine_MYSQL, true /* isObjectCaseSensitive */), nil
	}
}

func TestCompletion_RegisteredEngines(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "",
			Tables: []*metadatapb.TableMetadata{{Name: "users", Columns: []*metadatapb.ColumnMetadata{{Name: "id"}}}},
		}},
	})
	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "db",
		Metadata:        meta,
	}

	for _, engine := range []storepb.Engine{
		storepb.Engine_MYSQL,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_STARROCKS,
		storepb.Engine_DORIS,
	} {
		stmt := "SELECT * FROM "
		got, err := base.Completion(context.Background(), engine, cCtx, stmt, 1, len(stmt))
		require.NoError(t, err, "engine %v", engine)
		require.True(t, hasCandidate(got, base.CandidateTypeTable, "users"),
			"engine %v should offer table 'users'; got %v", engine, got)
	}
}

func TestCompletion_ReservedWordIdentifiersSurface(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "order",
						Columns: []*metadatapb.ColumnMetadata{
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

func TestCompletion_UnparseableColumnTypeFallsBackToStandIn(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{Name: "t", Columns: []*metadatapb.ColumnMetadata{
						{Name: "good", Type: "int"},
						{Name: "weird", Type: ")"},
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
		"column 'weird' with an unparseable type should still surface via the stand-in; got %v", got)
}

func TestCompletion_QuerySceneFiltersWriteKeywords(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name:    "testdb",
		Schemas: []*metadatapb.SchemaMetadata{{Name: ""}},
	})
	ccx := func(scene base.SceneType) base.CompletionContext {
		return base.CompletionContext{Scene: scene, DefaultDatabase: "testdb", Metadata: meta}
	}

	const stmt = "SELECT 1; "
	all, err := Completion(context.Background(), ccx(base.SceneTypeAll), stmt, 1, len(stmt))
	require.NoError(t, err)
	query, err := Completion(context.Background(), ccx(base.SceneTypeQuery), stmt, 1, len(stmt))
	require.NoError(t, err)

	for _, kw := range []string{"SELECT", "WITH", "SHOW"} {
		require.True(t, hasKeyword(all, kw), "%s in ALL; got %v", kw, all)
		require.True(t, hasKeyword(query, kw), "%s must be kept in QUERY; got %v", kw, query)
	}

	for _, kw := range []string{"INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP"} {
		require.True(t, hasKeyword(all, kw), "%s in ALL; got %v", kw, all)
		require.False(t, hasKeyword(query, kw), "%s must be dropped in QUERY; got %v", kw, query)
	}
}

func TestCompletion_QuerySceneKeepsReadSubkeywords(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: ""}},
	})
	q := base.CompletionContext{Scene: base.SceneTypeQuery, DefaultDatabase: "db", Metadata: meta}

	for _, tc := range []struct{ stmt, keep string }{
		{"SHOW ", "CREATE"},
		{"SELECT 1 FOR ", "UPDATE"},
	} {
		got, err := Completion(context.Background(), q, tc.stmt, 1, len(tc.stmt))
		require.NoError(t, err)
		require.True(t, hasKeyword(got, tc.keep),
			"%q must be kept after %q in the query scene; got %v", tc.keep, tc.stmt, got)
	}
}

func TestCompletion_QuerySceneFiltersWriteKeywordPrefix(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: ""}},
	})
	q := base.CompletionContext{Scene: base.SceneTypeQuery, DefaultDatabase: "db", Metadata: meta}

	got, err := Completion(context.Background(), q, "CRE", 1, len("CRE"))
	require.NoError(t, err)
	require.False(t, hasKeyword(got, "CREATE"),
		"CREATE must be filtered for a prefix at statement start; got %v", got)

	got, err = Completion(context.Background(), q, "SELECT 1; INS", 1, len("SELECT 1; INS"))
	require.NoError(t, err)
	require.False(t, hasKeyword(got, "INSERT"),
		"INSERT must be filtered for a prefix after ';'; got %v", got)

	got, err = Completion(context.Background(), q, "SHOW CRE", 1, len("SHOW CRE"))
	require.NoError(t, err)
	require.True(t, hasKeyword(got, "CREATE"),
		"CREATE must be kept after SHOW even with a prefix; got %v", got)
}

func TestCompletion_InstanceLevelDatabaseCandidates(t *testing.T) {
	meta := func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
		return databaseName, model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name:    databaseName,
			Schemas: []*metadatapb.SchemaMetadata{{Name: ""}},
		}, nil, nil, storepb.Engine_MYSQL, true), nil
	}
	cCtx := base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "",
		Metadata:          meta,
		ListDatabaseNames: func(_ context.Context, _ string) ([]string, error) { return []string{"appdb", "otherdb"}, nil },
	}

	stmt := "SELECT * FROM "
	got, err := Completion(context.Background(), cCtx, stmt, 1, len(stmt))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeDatabase, "appdb"),
		"instance-level db candidate 'appdb' should surface without a default database; got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeDatabase, "otherdb"),
		"instance-level db candidate 'otherdb' should surface without a default database; got %v", got)
}

func TestCompletion_LimitsToStatementAtCaret(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{
			{Name: "t1", Columns: []*metadatapb.ColumnMetadata{{Name: "a1"}, {Name: "a2"}}},
			{Name: "t2", Columns: []*metadatapb.ColumnMetadata{{Name: "b1"}, {Name: "b2"}}},
		}}},
	})
	cCtx := base.CompletionContext{Scene: base.SceneTypeAll, DefaultDatabase: "db", Metadata: meta}

	stmt := "SELECT * FROM t1; SELECT  FROM t2"
	caret := len("SELECT * FROM t1; SELECT ")
	got, err := Completion(context.Background(), cCtx, stmt, 1, caret)
	require.NoError(t, err)

	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "b1"), "t2.b1 should be in scope; got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "b2"), "t2.b2 should be in scope; got %v", got)
	require.False(t, hasCandidate(got, base.CandidateTypeColumn, "a1"), "t1.a1 must not leak into the 2nd statement; got %v", got)
	require.False(t, hasCandidate(got, base.CandidateTypeColumn, "a2"), "t1.a2 must not leak into the 2nd statement; got %v", got)
}

func TestCompletion_ViewWithBareSelectDefinitionSurfaces(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "",
			Tables: []*metadatapb.TableMetadata{{Name: "t1", Columns: []*metadatapb.ColumnMetadata{{Name: "id"}}}},
			Views:  []*metadatapb.ViewMetadata{{Name: "v1", Definition: "SELECT id FROM t1"}},
		}},
	})
	cCtx := base.CompletionContext{Scene: base.SceneTypeAll, DefaultDatabase: "db", Metadata: meta}

	stmt := "SELECT * FROM "
	got, err := Completion(context.Background(), cCtx, stmt, 1, len(stmt))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeView, "v1"),
		"view with a bare SELECT definition should surface as a VIEW candidate; got %v", got)

	stmt2 := "SELECT  FROM v1"
	got2, err := Completion(context.Background(), cCtx, stmt2, 1, len("SELECT "))
	require.NoError(t, err)
	require.True(t, hasCandidate(got2, base.CandidateTypeColumn, "id"),
		"view column 'id' should surface when selecting from v1; got %v", got2)
}

func TestCompletion_FunctionCandidatesGetParens(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{
			{Name: "t1", Columns: []*metadatapb.ColumnMetadata{{Name: "id"}}},
		}}},
	})
	cCtx := base.CompletionContext{Scene: base.SceneTypeAll, DefaultDatabase: "db", Metadata: meta}

	stmt := "SELECT  FROM t1"
	got, err := Completion(context.Background(), cCtx, stmt, 1, len("SELECT "))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeFunction, "COUNT()"),
		"function candidates should carry (); got %v", got)
	require.False(t, hasCandidate(got, base.CandidateTypeFunction, "COUNT"),
		"bare function name without () should not appear; got %v", got)
}

func TestCompletion_QuotesReservedNameAfterStringLiteralWithBacktick(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{
			{Name: "order", Columns: []*metadatapb.ColumnMetadata{{Name: "id"}}},
		}}},
	})
	cCtx := base.CompletionContext{Scene: base.SceneTypeAll, DefaultDatabase: "db", Metadata: meta}

	stmt := "SELECT '`' FROM "
	got, err := Completion(context.Background(), cCtx, stmt, 1, len(stmt))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeTable, "`order`"),
		"reserved table must stay quoted despite a backtick in a string literal; got %v", got)
}

func TestCompletion_QualifiedColumnAcrossDatabases(t *testing.T) {
	appMeta := &metadatapb.DatabaseSchemaMetadata{
		Name: "appdb",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{
			{Name: "t1", Columns: []*metadatapb.ColumnMetadata{{Name: "c1"}}},
		}}},
	}
	otherMeta := &metadatapb.DatabaseSchemaMetadata{
		Name: "otherdb",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{
			{Name: "t2", Columns: []*metadatapb.ColumnMetadata{{Name: "c2"}, {Name: "c3"}}},
		}}},
	}
	loads := map[string]int{}
	metaFn := func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
		loads[databaseName]++
		switch databaseName {
		case "appdb":
			return databaseName, model.NewDatabaseMetadata(appMeta, nil, nil, storepb.Engine_MYSQL, true), nil
		case "otherdb":
			return databaseName, model.NewDatabaseMetadata(otherMeta, nil, nil, storepb.Engine_MYSQL, true), nil
		default:
			return "", nil, nil
		}
	}
	listCalls := 0
	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "appdb",
		Metadata:        metaFn,
		ListDatabaseNames: func(_ context.Context, _ string) ([]string, error) {
			listCalls++
			return []string{"appdb", "otherdb", "unuseddb"}, nil
		},
	}

	stmt := "SELECT otherdb.t2. FROM otherdb.t2"
	got, err := Completion(context.Background(), cCtx, stmt, 1, len("SELECT otherdb.t2."))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "c2"),
		"cross-db column 'c2' should surface; got %v", got)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "c3"),
		"cross-db column 'c3' should surface; got %v", got)
	require.Zero(t, loads["unuseddb"], "unreferenced database must not be loaded")
	require.Zero(t, listCalls, "qualified cross-db completion must resolve the qualifier directly without ListDatabaseNames")

	stmt2 := "SELECT * FROM "
	got2, err := Completion(context.Background(), cCtx, stmt2, 1, len(stmt2))
	require.NoError(t, err)
	require.True(t, hasCandidate(got2, base.CandidateTypeDatabase, "otherdb"),
		"non-default database 'otherdb' should surface as a DATABASE candidate; got %v", got2)
	require.True(t, hasCandidate(got2, base.CandidateTypeDatabase, "unuseddb"),
		"non-default database 'unuseddb' should surface as a DATABASE candidate; got %v", got2)
	require.Equal(t, 1, listCalls, "database_ref position should enumerate databases once")
}

func TestCompletion_SkipsDatabaseEnumerationWhenNotNeeded(t *testing.T) {
	meta := metadataFunc(&metadatapb.DatabaseSchemaMetadata{
		Name: "appdb",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{
			{Name: "t1", Columns: []*metadatapb.ColumnMetadata{{Name: "c1"}}},
		}}},
	})
	var metaCalls []string
	trackedMeta := func(ctx context.Context, inst, db string) (string, *model.DatabaseMetadata, error) {
		metaCalls = append(metaCalls, db)
		return meta(ctx, inst, db)
	}
	listCalls := 0
	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "appdb",
		Metadata:        trackedMeta,
		ListDatabaseNames: func(_ context.Context, _ string) ([]string, error) {
			listCalls++
			return []string{"appdb", "otherdb"}, nil
		},
	}

	stmt := "SELECT  FROM t1"
	got, err := Completion(context.Background(), cCtx, stmt, 1, len("SELECT "))
	require.NoError(t, err)
	require.True(t, hasCandidate(got, base.CandidateTypeColumn, "c1"))
	require.Zero(t, listCalls, "ListDatabaseNames must not be called when neither database_ref nor cross-db qualifiers apply")
	require.Equal(t, []string{"appdb"}, metaCalls)

	metaCalls = nil
	stmtAlias := "SELECT a. FROM t1 a WHERE a.c1 = 1"
	gotAlias, err := Completion(context.Background(), cCtx, stmtAlias, 1, len("SELECT a."))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotAlias, base.CandidateTypeColumn, "c1"))
	require.Zero(t, listCalls, "ListDatabaseNames must not be called for table/alias column qualifiers")
	require.Equal(t, []string{"appdb"}, metaCalls, "table/alias qualifiers must not trigger Metadata lookups for non-db names")
}

// Completion cost must be bounded by the caret's statement, not the whole
// sheet: a broken statement anywhere after the caret must not make completion
// scale with the entire trailing document (BYT-9886).
func TestCompletionWithBrokenTrailingStatementScalesLinearly(t *testing.T) {
	a := require.New(t)

	var sheet strings.Builder
	sheet.WriteString("SELECT a2. FROM t1 JOIN t2 a2 ON t1.c1 = a2.c1;\n")
	sheet.WriteString("SELEC broken FROM oops;\n")
	for i := range 800 {
		fmt.Fprintf(&sheet, "SELECT col_a, col_b, col_c FROM table_%04d WHERE col_a = %d AND col_b LIKE 'pattern%%' ORDER BY col_c LIMIT 100;\n", i, i)
	}

	started := time.Now()
	result, err := base.Completion(context.Background(), storepb.Engine_MYSQL, base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "db",
		Metadata:          getMetadataForTest,
		ListDatabaseNames: listDatabaseNamesForTest,
	}, sheet.String(), 1, 10 /* caret right after "SELECT a2." */)
	elapsed := time.Since(started)

	a.NoError(err)
	a.True(hasCandidate(result, base.CandidateTypeColumn, "c1"))
	a.True(hasCandidate(result, base.CandidateTypeColumn, "c2"))
	a.Less(elapsed, 2*time.Second)
}

func listDatabaseNamesForTest(_ context.Context, _ string) ([]string, error) {
	return []string{"db"}, nil
}

func getMetadataForTest(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
	if databaseName != "db" {
		return "", nil, nil
	}

	return "db", model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: databaseName,
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:    "t1",
						Columns: []*metadatapb.ColumnMetadata{{Name: "c1"}},
					},
					{
						Name:    "t2",
						Columns: []*metadatapb.ColumnMetadata{{Name: "c1"}, {Name: "c2"}},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{
						Name:       "v1",
						Definition: "SELECT * FROM t1",
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_MYSQL, true /* isObjectCaseSensitive */), nil
}
