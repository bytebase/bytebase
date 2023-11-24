package plancheck

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type statementTypeCheckData struct {
	Statement  string
	ChangeType storepb.PlanCheckRunConfig_ChangeDatabaseType
	Want       []string
}

func TestMySQLStatementTypeCheck(t *testing.T) {
	tests := []string{
		"mysql_statement_type_check",
	}

	for _, test := range tests {
		runStatementTypeCheck(t, test, storepb.Engine_MYSQL, false /* record */)
	}
}

func runStatementTypeCheck(t *testing.T, file string, engineType storepb.Engine, record bool) {
	filepath := filepath.Join("test", file+".yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)

	tests := []statementTypeCheckData{}
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for i, test := range tests {
		var checkResults []*storepb.PlanCheckRunResult_Result
		var err error
		switch engineType {
		case storepb.Engine_MYSQL:
			checkResults, err = mysqlStatementTypeCheck(test.Statement, test.ChangeType)
			require.NoError(t, err)
		default:
		}

		if record {
			resultSet := []string{}
			for _, result := range checkResults {
				resultSet = append(resultSet, protojson.Format(result))
			}
			tests[i].Want = resultSet
		} else {
			want := []*storepb.PlanCheckRunResult_Result{}
			for _, result := range tests[i].Want {
				res := &storepb.PlanCheckRunResult_Result{}
				err := protojson.Unmarshal([]byte(result), res)
				require.NoError(t, err)
				want = append(want, res)
			}
			equalCheckRunResultProtos(t, want, checkResults, tests[i].Statement)
		}
	}

	if record {
		byteValue, err = yaml.Marshal(tests)
		require.NoError(t, err)
		err = os.WriteFile(filepath, byteValue, 0644)
		require.NoError(t, err)
	}
}

func equalCheckRunResultProtos(t *testing.T, want, got []*storepb.PlanCheckRunResult_Result, message string) {
	require.Equal(t, len(want), len(got))
	for i := 0; i < len(want); i++ {
		diff := cmp.Diff(want[i], got[i], protocmp.Transform())
		require.Equal(t, "", diff, message)
	}
}
