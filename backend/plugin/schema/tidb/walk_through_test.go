package tidb

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
		state := model.NewDatabaseMetadata(protoData, nil, nil, storepb.Engine_TIDB, !test.IgnoreCaseSensitive)

		stmts, _ := sm.GetStatementsForChecks(storepb.Engine_TIDB, test.Statement)
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
