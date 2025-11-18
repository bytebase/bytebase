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
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema/catalogutil"
	"github.com/bytebase/bytebase/backend/store/model"
)

type testData struct {
	Statement string
	// Use custom yaml tag to avoid generate field name `ignorecasesensitive`.
	IgnoreCaseSensitive bool `yaml:"ignore_case_sensitive"`
	Want                string
	Err                 *catalogutil.WalkThroughError
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
	sm := sheet.NewManager(nil)

	for _, test := range tests {
		var protoData *storepb.DatabaseSchemaMetadata
		if originDatabase != nil {
			// Make a deep copy to avoid mutation across tests
			protoData = proto.Clone(originDatabase).(*storepb.DatabaseSchemaMetadata)
		} else {
			protoData = &storepb.DatabaseSchemaMetadata{}
		}

		// Create DatabaseMetadata for walk-through
		state := model.NewDatabaseMetadata(protoData, !test.IgnoreCaseSensitive, !test.IgnoreCaseSensitive)

		asts, _ := sm.GetASTsForChecks(storepb.Engine_POSTGRES, test.Statement)
		err := WalkThrough(state, asts)
		if err != nil {
			walkErr, yes := err.(*catalogutil.WalkThroughError)
			require.True(t, yes)
			require.Equal(t, test.Err, walkErr)
			continue
		}
		require.NoError(t, err, test.Statement)

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &storepb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		// Sort proto for deterministic comparison
		state.SortProto()
		result := state.GetProto()
		diff := cmp.Diff(want, result, protocmp.Transform())
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
		var protoData *storepb.DatabaseSchemaMetadata
		if originDatabase != nil {
			// Make a deep copy to avoid mutation across tests
			protoData = proto.Clone(originDatabase).(*storepb.DatabaseSchemaMetadata)
		} else {
			protoData = &storepb.DatabaseSchemaMetadata{}
		}

		// Create DatabaseMetadata for walk-through
		state := model.NewDatabaseMetadata(protoData, !test.IgnoreCaseSensitive, !test.IgnoreCaseSensitive)

		// Parse using ANTLR parser instead of legacy parser
		parseResult, parseErr := pgparser.ParsePostgreSQL(test.Statement)
		if parseErr != nil {
			t.Fatalf("Failed to parse SQL with ANTLR: %v\nSQL: %s", parseErr, test.Statement)
		}

		// Call WalkThrough with ANTLR tree
		err := WalkThrough(state, parseResult)
		if err != nil {
			walkErr, yes := err.(*catalogutil.WalkThroughError)
			require.True(t, yes)
			require.Equal(t, test.Err, walkErr)
			continue
		}
		require.NoError(t, err, test.Statement)

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &storepb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		// Sort proto for deterministic comparison
		state.SortProto()
		result := state.GetProto()
		diff := cmp.Diff(want, result, protocmp.Transform())
		require.Empty(t, diff)
	}
}
