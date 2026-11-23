package pg

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

	t.Run("valid DDL updates metadata through registry and preserves existing tables", func(t *testing.T) {
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
		require.NotNil(t, state.GetSchemaMetadata("public").GetTable("test"))
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

func TestWalkThroughSessionUserAndConfiguredSearchPath(t *testing.T) {
	tests := []struct {
		name       string
		searchPath string
		session    schema.WalkThroughContext
		sql        string
		assert     func(*testing.T, *model.DatabaseMetadata)
	}{
		{
			name:       "session user expands $user from configured search_path",
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
			name:       "configured multi-schema search_path is forwarded as default",
			searchPath: `public, app`,
			sql:        `SET search_path TO app; SET search_path TO DEFAULT; CREATE TABLE default_target (id int);`,
			assert: func(t *testing.T, state *model.DatabaseMetadata) {
				t.Helper()
				require.NotNil(t, state.GetSchemaMetadata("public").GetTable("default_target"))
				require.Nil(t, state.GetSchemaMetadata("app").GetTable("default_target"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
				Name:       "postgres",
				SearchPath: tc.searchPath,
				Schemas:    []*metadatapb.SchemaMetadata{{Name: "alice"}, {Name: "app"}, {Name: "public"}},
			}, nil, nil, storepb.Engine_POSTGRES, true)
			wtCtx := tc.session
			wtCtx.RawSQL = tc.sql
			advice := WalkThroughWithContext(context.Background(), wtCtx, state, nil)
			require.Nil(t, advice)
			tc.assert(t, state)
		})
	}
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
