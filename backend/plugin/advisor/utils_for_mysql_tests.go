package advisor

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// RunSQLReviewRuleTest helps to test the SQL review rule.
func RunSQLReviewRuleDebugTest(t *testing.T, rule SQLReviewRuleType, dbType storepb.Engine, record bool) {
	var tests []TestCase

	fileName := "test_advisor.yaml"
	filepath := filepath.Join("test", fileName)
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err, rule)

	for i, tc := range tests {
		database := MockMySQLDatabase
		if dbType == storepb.Engine_POSTGRES {
			database = MockPostgreSQLDatabase
		}
		finder := catalog.NewFinder(database, &catalog.FinderContext{CheckIntegrity: true, EngineType: dbType})

		payload, err := SetDefaultSQLReviewRulePayload(rule, dbType)
		require.NoError(t, err)

		ruleList := []*storepb.SQLReviewRule{
			{
				Type:    string(rule),
				Level:   storepb.SQLReviewRuleLevel_WARNING,
				Payload: string(payload),
			},
		}

		ctx := SQLReviewCheckContext{
			Charset:         "",
			Collation:       "",
			DbType:          dbType,
			Catalog:         &testCatalog{finder: finder},
			Driver:          nil,
			Context:         context.Background(),
			CurrentSchema:   "SYS",
			CurrentDatabase: "TEST_DB",
		}

		adviceList, err := SQLReviewCheck(tc.Statement, ruleList, ctx)
		// Sort adviceList by (line, content)
		slices.SortFunc[Advice](adviceList, func(i, j Advice) bool {
			if i.Line != j.Line {
				return i.Line < j.Line
			}
			return i.Content < j.Content
		})
		require.NoError(t, err)
		if record {
			tests[i].Want = adviceList
		} else {
			require.Equalf(t, tc.Want, adviceList, "rule: %s, statements: %s", rule, tc.Statement)
		}
	}

	if record {
		err := yamlFile.Close()
		require.NoError(t, err)
		byteValue, err := yaml.Marshal(tests)
		require.NoError(t, err)
		err = os.WriteFile(filepath, byteValue, 0644)
		require.NoError(t, err)
	}
}
