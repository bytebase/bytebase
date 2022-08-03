package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestMySQLTableDropNamingConvention(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "DROP TABLE IF EXISTS foo_delete",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "DROP TABLE IF EXISTS foo",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableDropNamingConventionMismatch,
					Title:   "table.drop-naming-convention",
					Content: "`foo` mismatches drop table naming convention, naming format should be \"_delete$\"",
				},
			},
		},
		{
			Statement: "DROP TABLE IF EXISTS foo_delete, bar",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableDropNamingConventionMismatch,
					Title:   "table.drop-naming-convention",
					Content: "`bar` mismatches drop table naming convention, naming format should be \"_delete$\"",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format: "_delete$",
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &TableDropNamingConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleTableDropNamingConvention,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, advisor.MockMySQLDatabase)
}
