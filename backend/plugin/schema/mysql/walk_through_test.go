package mysql

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	omniast "github.com/bytebase/omni/mysql/ast"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
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
		Name: "test",
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
		state := model.NewDatabaseMetadata(protoData, nil, nil, storepb.Engine_MYSQL, !test.IgnoreCaseSensitive)

		stmts, _ := sm.GetStatementsForChecks(storepb.Engine_MYSQL, test.Statement)
		asts := base.ExtractASTs(stmts)
		advice := WalkThroughOmni(schema.WalkThroughContext{RawSQL: test.Statement}, state, asts)
		if advice != nil {
			// Compare the advice fields
			require.NotNil(t, test.Advice, "unexpected advice for statement %q: %+v", test.Statement, advice)
			require.Equal(t, test.Advice.Code, advice.Code, "statement %q advice %+v", test.Statement, advice)
			require.Equal(t, test.Advice.Content, advice.Content, "statement %q advice %+v", test.Statement, advice)
			continue
		}

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &storepb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		result := state.GetProto()
		diff := cmp.Diff(want, result, protocmp.Transform(),
			protocmp.SortRepeatedFields(&storepb.DatabaseSchemaMetadata{}, "schemas"),
			protocmp.SortRepeatedFields(&storepb.SchemaMetadata{}, "tables", "views"),
			protocmp.SortRepeatedFields(&storepb.TableMetadata{}, "indexes", "columns"),
		)
		require.Empty(t, diff, "statement %q", test.Statement)
	}
}

func TestWalkThroughOmniCreateTableIfNotExistsCTASExistingTable(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "test",
		Schemas: []*storepb.SchemaMetadata{
			{
				Tables: []*storepb.TableMetadata{
					{
						Name: "t1",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:     "a",
								Position: 1,
								Nullable: true,
								Type:     "int",
							},
						},
					},
				},
			},
		},
	}

	state := model.NewDatabaseMetadata(originDatabase, nil, nil, storepb.Engine_MYSQL, true)
	require.NotNil(t, state.GetSchemaMetadata("").GetTable("t1"))
	statement := "CREATE TABLE IF NOT EXISTS t1 AS SELECT 1;"
	sm := sheet.NewManager()
	stmts, _ := sm.GetStatementsForChecks(storepb.Engine_MYSQL, statement)
	asts := base.ExtractASTs(stmts)
	omniAST, ok := asts[0].(*mysqlparser.OmniAST)
	require.True(t, ok)
	createTable, ok := omniAST.Node.(*omniast.CreateTableStmt)
	require.True(t, ok)
	require.True(t, createTable.IfNotExists)

	advice := WalkThroughOmni(schema.WalkThroughContext{RawSQL: statement}, state, asts)
	require.Nil(t, advice)
	require.NotNil(t, state.GetSchemaMetadata("").GetTable("t1"))
}
