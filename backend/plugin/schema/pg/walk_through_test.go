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

		// Parse using base.Parse to get AST
		unifiedASTs, parseErr := base.Parse(storepb.Engine_POSTGRES, test.Statement)
		if parseErr != nil {
			t.Fatalf("Failed to parse SQL: %v\nSQL: %s", parseErr, test.Statement)
		}

		// Call WalkThrough with AST
		advice := WalkThrough(state, unifiedASTs)
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
