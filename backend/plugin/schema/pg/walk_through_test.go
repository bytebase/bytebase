package pg

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Registers the PostgreSQL ParseStatements func these tests reach through
	// base. Nothing in this package's own imports pulls it in.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

type testData struct {
	Statement string
	// Use custom yaml tag to avoid generate field name `ignorecasesensitive`.
	IgnoreCaseSensitive bool `yaml:"ignore_case_sensitive"`
	Want                string
	Advice              *storepb.Advice
}

func TestWalkThrough(t *testing.T) {
	originDatabase := &metadatapb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "test",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
								Position: 1,
							},
							{
								Name:     "name",
								Type:     "varchar(20)",
								Nullable: true,
								Position: 2,
							},
						},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{
						Name:       "v1",
						Definition: "SELECT id, name FROM test",
						DependencyColumns: []*metadatapb.DependencyColumn{
							{
								Schema: "public",
								Table:  "test",
								Column: "id",
							},
							{
								Schema: "public",
								Table:  "test",
								Column: "name",
							},
						},
					},
				},
			},
		},
	}

	tests := []testData{}
	targetPath := filepath.Join("testdata", "walk_through.yaml")
	yamlFile, err := os.Open(targetPath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)
	sm := sheet.NewManager()

	for i := range tests {
		test := &tests[i]
		// Make a deep copy to avoid mutation across tests
		protoData, ok := proto.Clone(originDatabase).(*metadatapb.DatabaseSchemaMetadata)
		require.True(t, ok)

		// Create DatabaseMetadata for walk-through
		state := model.NewDatabaseMetadata(protoData, nil, nil, storepb.Engine_POSTGRES, !test.IgnoreCaseSensitive)

		stmts, _ := sm.GetStatementsForChecks(storepb.Engine_POSTGRES, test.Statement)
		asts := base.ExtractASTs(stmts)
		advice := WalkThroughWithContext(context.Background(), schema.WalkThroughContext{RawSQL: test.Statement}, state, asts)
		if test.Advice != nil {
			require.NotNil(t, advice)
			require.Equal(t, test.Advice.Code, advice.Code)
			require.Equal(t, test.Advice.Content, advice.Content)
			continue
		}
		if advice != nil {
			continue
		}

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &metadatapb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		result := state.GetProto()
		diff := cmp.Diff(want, result, protocmp.Transform(),
			protocmp.SortRepeatedFields(&metadatapb.DatabaseSchemaMetadata{}, "schemas"),
			protocmp.SortRepeatedFields(&metadatapb.SchemaMetadata{}, "tables", "views"),
			protocmp.SortRepeatedFields(&metadatapb.TableMetadata{}, "indexes", "columns"),
		)
		require.Empty(t, diff)
	}
}

func TestWalkThroughANTLR(t *testing.T) {
	originDatabase := &metadatapb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "test",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
								Position: 1,
							},
							{
								Name:     "name",
								Type:     "varchar(20)",
								Nullable: true,
								Position: 2,
							},
						},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{
						Name:       "v1",
						Definition: "SELECT id, name FROM test",
						DependencyColumns: []*metadatapb.DependencyColumn{
							{
								Schema: "public",
								Table:  "test",
								Column: "id",
							},
							{
								Schema: "public",
								Table:  "test",
								Column: "name",
							},
						},
					},
				},
			},
		},
	}

	tests := []testData{}
	targetPath := filepath.Join("testdata", "walk_through.yaml")
	yamlFile, err := os.Open(targetPath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for i := range tests {
		test := &tests[i]
		// Make a deep copy to avoid mutation across tests
		protoData, ok := proto.Clone(originDatabase).(*metadatapb.DatabaseSchemaMetadata)
		require.True(t, ok)

		// Create DatabaseMetadata for walk-through
		state := model.NewDatabaseMetadata(protoData, nil, nil, storepb.Engine_POSTGRES, !test.IgnoreCaseSensitive)

		// Parse using base.ParseStatements to get AST
		stmts, parseErr := base.ParseStatements(storepb.Engine_POSTGRES, test.Statement)
		if parseErr != nil {
			t.Fatalf("Failed to parse SQL: %v\nSQL: %s", parseErr, test.Statement)
		}
		asts := base.ExtractASTs(stmts)

		// Call WalkThrough with AST
		advice := WalkThroughWithContext(context.Background(), schema.WalkThroughContext{RawSQL: test.Statement}, state, asts)
		if advice != nil {
			// Compare the advice fields
			if test.Advice != nil {
				require.Equal(t, test.Advice.Code, advice.Code)
				require.Equal(t, test.Advice.Content, advice.Content)
			}
			continue
		}

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &metadatapb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		result := state.GetProto()
		diff := cmp.Diff(want, result, protocmp.Transform(),
			protocmp.SortRepeatedFields(&metadatapb.DatabaseSchemaMetadata{}, "schemas"),
			protocmp.SortRepeatedFields(&metadatapb.SchemaMetadata{}, "tables", "views"),
			protocmp.SortRepeatedFields(&metadatapb.TableMetadata{}, "indexes", "columns"),
		)
		require.Empty(t, diff)
	}
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
