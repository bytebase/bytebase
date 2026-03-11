package pg

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "test",
						Columns: []*storepb.ColumnMetadata{
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
				Views: []*storepb.ViewMetadata{
					{
						Name:       "v1",
						Definition: "SELECT id, name FROM test",
						DependencyColumns: []*storepb.DependencyColumn{
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
	filepath := filepath.Join("testdata", "walk_through.yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)
	sm := sheet.NewManager()

	for _, test := range tests {
		// Make a deep copy to avoid mutation across tests
		protoData, ok := proto.Clone(originDatabase).(*storepb.DatabaseSchemaMetadata)
		require.True(t, ok)

		// Create DatabaseMetadata for walk-through
		state := model.NewDatabaseMetadata(protoData, nil, nil, storepb.Engine_POSTGRES, !test.IgnoreCaseSensitive)

		stmts, _ := sm.GetStatementsForChecks(storepb.Engine_POSTGRES, test.Statement)
		asts := base.ExtractASTs(stmts)
		advice := WalkThrough(state, asts)
		if advice != nil {
			// Compare the advice fields
			require.Equal(t, test.Advice.Code, advice.Code)
			require.Equal(t, test.Advice.Content, advice.Content)
			continue
		}

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &storepb.DatabaseSchemaMetadata{}
		err := common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		result := state.GetProto()
		diff := cmp.Diff(want, result, protocmp.Transform(),
			protocmp.SortRepeatedFields(&storepb.DatabaseSchemaMetadata{}, "schemas"),
			protocmp.SortRepeatedFields(&storepb.SchemaMetadata{}, "tables", "views"),
			protocmp.SortRepeatedFields(&storepb.TableMetadata{}, "indexes", "columns"),
		)
		require.Empty(t, diff)
	}
}

func TestWalkThroughANTLR(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "test",
						Columns: []*storepb.ColumnMetadata{
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
				Views: []*storepb.ViewMetadata{
					{
						Name:       "v1",
						Definition: "SELECT id, name FROM test",
						DependencyColumns: []*storepb.DependencyColumn{
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
	filepath := filepath.Join("testdata", "walk_through.yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for _, test := range tests {
		// Make a deep copy to avoid mutation across tests
		protoData, ok := proto.Clone(originDatabase).(*storepb.DatabaseSchemaMetadata)
		require.True(t, ok)

		// Create DatabaseMetadata for walk-through
		state := model.NewDatabaseMetadata(protoData, nil, nil, storepb.Engine_POSTGRES, !test.IgnoreCaseSensitive)

		// Parse using base.ParseStatements to get AST
		stmts, parseErr := base.ParseStatements(storepb.Engine_POSTGRES, test.Statement)
		if parseErr != nil {
			t.Fatalf("Failed to parse SQL: %v\nSQL: %s", parseErr, test.Statement)
		}

		// Call WalkThrough with AST
		advice := WalkThrough(state, base.ExtractASTs(stmts))
		if advice != nil {
			// Compare the advice fields
			require.Equal(t, test.Advice.Code, advice.Code)
			require.Equal(t, test.Advice.Content, advice.Content)
			continue
		}

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &storepb.DatabaseSchemaMetadata{}
		err := common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		result := state.GetProto()
		diff := cmp.Diff(want, result, protocmp.Transform(),
			protocmp.SortRepeatedFields(&storepb.DatabaseSchemaMetadata{}, "schemas"),
			protocmp.SortRepeatedFields(&storepb.SchemaMetadata{}, "tables", "views"),
			protocmp.SortRepeatedFields(&storepb.TableMetadata{}, "indexes", "columns"),
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := newSearchPathTestState(test.searchPath)
			stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, test.sql)
			require.NoError(t, err)
			advice := WalkThroughWithContext(test.session, state, base.ExtractASTs(stmts))
			require.Nil(t, advice)
			test.assert(t, state)
		})
	}
}

func newSearchPathTestState(searchPath string) *model.DatabaseMetadata {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name:       "postgres",
		SearchPath: searchPath,
		Schemas: []*storepb.SchemaMetadata{
			{Name: "alice"},
			{Name: "bob"},
			{
				Name: "app",
				Tables: []*storepb.TableMetadata{
					newSearchPathTestTable("dup"),
				},
			},
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					newSearchPathTestTable("dup"),
				},
			},
		},
	}
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
}

func newSearchPathTestTable(name string) *storepb.TableMetadata {
	return &storepb.TableMetadata{
		Name: name,
		Columns: []*storepb.ColumnMetadata{
			{
				Name:     "id",
				Type:     "int",
				Nullable: false,
				Position: 1,
			},
		},
	}
}
