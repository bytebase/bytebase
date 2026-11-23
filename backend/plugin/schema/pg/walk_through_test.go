package pg

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Registers the PostgreSQL ParseStatements func these tests reach through
	// base. Nothing in this package's own imports pulls it in.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestWalkThroughSeam(t *testing.T) {
	newState := func() *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: "postgres",
			Schemas: []*metadatapb.SchemaMetadata{{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{{
					Name: "test",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "int", Nullable: false, Position: 1},
					},
				}},
			}},
		}, nil, nil, storepb.Engine_POSTGRES, true)
	}

	t.Run("empty RawSQL is a no-op", func(t *testing.T) {
		state := newState()
		advice := schema.WalkThroughWithContext(context.Background(), storepb.Engine_POSTGRES, schema.WalkThroughContext{}, state, nil)
		require.Nil(t, advice)
		require.NotNil(t, state.GetSchemaMetadata("public").GetTable("test"))
	})

	t.Run("valid DDL updates metadata through registry", func(t *testing.T) {
		state := newState()
		advice := schema.WalkThroughWithContext(
			context.Background(),
			storepb.Engine_POSTGRES,
			schema.WalkThroughContext{RawSQL: "CREATE TABLE t(a int, b int);"},
			state,
			nil,
		)
		require.Nil(t, advice)
		tbl := state.GetSchemaMetadata("public").GetTable("t")
		require.NotNil(t, tbl)
		require.Len(t, tbl.GetProto().GetColumns(), 2)
	})

	t.Run("maps catalog SQLSTATE errors to Bytebase advice codes", func(t *testing.T) {
		cases := []struct {
			name     string
			sql      string
			wantCode code.Code
		}{
			{
				name:     "duplicate table",
				sql:      "CREATE TABLE test(a int);",
				wantCode: code.TableExists,
			},
			{
				name:     "undefined table",
				sql:      "ALTER TABLE missing ADD COLUMN a int;",
				wantCode: code.TableNotExists,
			},
			{
				name:     "duplicate column",
				sql:      "ALTER TABLE test ADD COLUMN id int;",
				wantCode: code.ColumnExists,
			},
			{
				name:     "column reference in DEFAULT",
				sql:      `CREATE TABLE t_default_column(a int, b int DEFAULT a);`,
				wantCode: code.DDLSimulationFailed,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				state := newState()
				advice := schema.WalkThroughWithContext(
					context.Background(),
					storepb.Engine_POSTGRES,
					schema.WalkThroughContext{RawSQL: tc.sql},
					state,
					nil,
				)
				require.NotNil(t, advice)
				require.Equal(t, storepb.Advice_ERROR, advice.Status)
				require.Equal(t, tc.wantCode.Int32(), advice.Code)
			})
		}
	})
}

func TestWalkThroughSearchPathState(t *testing.T) {
	tests := []struct {
		name       string
		searchPath string
		session    schema.WalkThroughContext
		sql        string
		assert     func(*testing.T, *model.DatabaseMetadata)
	}{
		{
			name:       "session user expands $user on initial resolution",
			searchPath: `"$user", public`,
			session:    schema.WalkThroughContext{SessionUser: "alice"},
			sql:        `CREATE TABLE session_target (id int);`,
			assert: func(t *testing.T, state *model.DatabaseMetadata) {
				t.Helper()
				require.NotNil(t, state.GetSchemaMetadata("alice").GetTable("session_target"))
				require.Nil(t, state.GetSchemaMetadata("public").GetTable("session_target"))
			},
		},
		{
			name:       "set role recomputes $user target schema",
			searchPath: `"$user", public`,
			session:    schema.WalkThroughContext{SessionUser: "alice"},
			sql:        `SET ROLE bob; CREATE TABLE role_target (id int);`,
			assert: func(t *testing.T, state *model.DatabaseMetadata) {
				t.Helper()
				require.NotNil(t, state.GetSchemaMetadata("bob").GetTable("role_target"))
				require.Nil(t, state.GetSchemaMetadata("alice").GetTable("role_target"))
			},
		},
		{
			name:       "set search_path changes unqualified create target",
			searchPath: `public`,
			sql:        `SET search_path TO app, public; CREATE TABLE path_target (id int);`,
			assert: func(t *testing.T, state *model.DatabaseMetadata) {
				t.Helper()
				require.NotNil(t, state.GetSchemaMetadata("app").GetTable("path_target"))
				require.Nil(t, state.GetSchemaMetadata("public").GetTable("path_target"))
			},
		},
		{
			name:       "ordered lookup picks first matching schema",
			searchPath: `public`,
			sql:        `SET search_path TO app, public; CREATE INDEX idx_dup_id ON dup(id);`,
			assert: func(t *testing.T, state *model.DatabaseMetadata) {
				t.Helper()
				require.NotNil(t, state.GetSchemaMetadata("app").GetTable("dup").GetIndex("idx_dup_id"))
				require.Nil(t, state.GetSchemaMetadata("public").GetTable("dup").GetIndex("idx_dup_id"))
			},
		},
		{
			name:       "set search_path to default restores configured path",
			searchPath: `public, app`,
			sql:        `SET search_path TO app; SET search_path TO DEFAULT; CREATE TABLE default_target (id int);`,
			assert: func(t *testing.T, state *model.DatabaseMetadata) {
				t.Helper()
				require.NotNil(t, state.GetSchemaMetadata("public").GetTable("default_target"))
				require.Nil(t, state.GetSchemaMetadata("app").GetTable("default_target"))
			},
		},
		{
			name:       "explicit schema still wins over current path",
			searchPath: `public`,
			sql:        `SET search_path TO app; CREATE TABLE public.explicit_target (id int);`,
			assert: func(t *testing.T, state *model.DatabaseMetadata) {
				t.Helper()
				require.NotNil(t, state.GetSchemaMetadata("public").GetTable("explicit_target"))
				require.Nil(t, state.GetSchemaMetadata("app").GetTable("explicit_target"))
			},
		},
		{
			name:       "missing first path entry is skipped rather than auto-created",
			searchPath: `missing_schema, public`,
			sql:        `CREATE TABLE skipped_missing (id int);`,
			assert: func(t *testing.T, state *model.DatabaseMetadata) {
				t.Helper()
				require.Nil(t, state.GetSchemaMetadata("missing_schema"))
				require.NotNil(t, state.GetSchemaMetadata("public").GetTable("skipped_missing"))
			},
		},
	}

	for i := range tests {
		test := &tests[i]
		t.Run(test.name, func(t *testing.T) {
			state := newSearchPathTestState(test.searchPath)
			stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, test.sql)
			require.NoError(t, err)
			wtCtx := test.session
			wtCtx.RawSQL = test.sql
			advice := WalkThroughWithContext(context.Background(), wtCtx, state, base.ExtractASTs(stmts))
			require.Nil(t, advice)
			test.assert(t, state)
		})
	}
}

func newSearchPathTestState(searchPath string) *model.DatabaseMetadata {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name:       "postgres",
		SearchPath: searchPath,
		Schemas: []*metadatapb.SchemaMetadata{
			{Name: "alice"},
			{Name: "bob"},
			{
				Name: "app",
				Tables: []*metadatapb.TableMetadata{
					newSearchPathTestTable("dup"),
				},
			},
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					newSearchPathTestTable("dup"),
				},
			},
		},
	}
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
}

func newSearchPathTestTable(name string) *metadatapb.TableMetadata {
	return &metadatapb.TableMetadata{
		Name: name,
		Columns: []*metadatapb.ColumnMetadata{
			{
				Name:     "id",
				Type:     "int",
				Nullable: false,
				Position: 1,
			},
		},
	}
}

// TestClone_SearchPath tests Clone preserves search path and session user.
func TestClone_SearchPath(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*metadatapb.SchemaMetadata{
			{Name: "public"},
			{Name: "alice"},
		},
	}

	catBefore := catalog.New()
	catBefore.SetSessionUser("alice")
	catBefore.SetSearchPath([]string{"$user", "public"})
	_, err := catBefore.LoadMetadata(context.Background(), meta, catalog.LoadMetadataOptions{Full: true})
	require.NoError(t, err)

	catAfter := catBefore.Clone()

	// CREATE TABLE without schema should use search path ($user → alice)
	_, err = catAfter.Exec(`CREATE TABLE my_table (id int);`, nil)
	require.NoError(t, err)

	// Table should be in alice schema on clone
	require.NotNil(t, catAfter.GetRelation("alice", "my_table"))

	// Original should NOT have it
	require.Nil(t, catBefore.GetRelation("alice", "my_table"))

	// Diff should show the new table
	diff := catalog.Diff(catBefore, catAfter)
	require.False(t, diff.IsEmpty())

	foundAliceTable := false
	for _, rel := range diff.Relations {
		if rel.SchemaName == "alice" && rel.Name == "my_table" && rel.Action == catalog.DiffAdd {
			foundAliceTable = true
		}
	}
	require.True(t, foundAliceTable, "diff should show alice.my_table as added")
}

// TestClone_WalkThroughFunction tests the actual WalkThroughWithContext function
// using Clone produces correct FinalMetadata.
func TestClone_WalkThroughFunction(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "test",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "name", Type: "text", Position: 2, Nullable: true},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "test_pkey", Expressions: []string{"id"}, Unique: true, Primary: true},
						},
					},
				},
			},
		},
	}

	state := model.NewDatabaseMetadata(meta, nil, nil, storepb.Engine_POSTGRES, true)
	wtCtx := schema.WalkThroughContext{
		RawSQL: `CREATE TABLE public.new_table (id int PRIMARY KEY, val text);`,
	}

	advice := WalkThroughWithContext(context.Background(), wtCtx, state, nil)
	require.Nil(t, advice, "walk-through should succeed")

	// Check FinalMetadata has the new table
	newTbl := state.GetSchemaMetadata("public").GetTable("new_table")
	require.NotNil(t, newTbl, "new_table should exist in FinalMetadata")

	// Check original table is preserved
	origTbl := state.GetSchemaMetadata("public").GetTable("test")
	require.NotNil(t, origTbl, "test table should still exist")
	require.Equal(t, 2, len(origTbl.GetProto().Columns))
}

func TestWalkThroughOwnerChangeAndWrapperPreservation(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:  "test",
						Owner: "alice",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
						},
					},
				},
			},
		},
	}
	cfg := &storepb.DatabaseConfig{Name: "postgres"}
	state := model.NewDatabaseMetadata(meta, nil, cfg, storepb.Engine_POSTGRES, true)

	advice := schema.WalkThroughWithContext(
		context.Background(),
		storepb.Engine_POSTGRES,
		schema.WalkThroughContext{RawSQL: `ALTER TABLE public.test OWNER TO bob;`},
		state,
		nil,
	)
	require.Nil(t, advice)
	require.Equal(t, "bob", state.GetSchemaMetadata("public").GetTable("test").GetProto().GetOwner())
	require.Equal(t, cfg, state.GetConfig())
	require.True(t, state.GetIsObjectCaseSensitive())
}
