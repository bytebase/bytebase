package pg

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestColumnTypeDisallowList(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: `CREATE TABLE t(a char(5));`,
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
			Statement: `CREATE TABLE t(a int, b bigint, c real);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.DisabledColumnType,
					Title:   "column.type-disallow-list",
					Content: "Disallow column type INT but column \"t\".\"a\" is",
					Line:    1,
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.DisabledColumnType,
					Title:   "column.type-disallow-list",
					Content: "Disallow column type BIGINT but column \"t\".\"b\" is",
					Line:    1,
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.DisabledColumnType,
					Title:   "column.type-disallow-list",
					Content: "Disallow column type REAL but column \"t\".\"c\" is",
					Line:    1,
				},
			},
		},
		{
			Statement: `
				CREATE TABLE t(d char(5));
				ALTER TABLE t add COLUMN a int;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.DisabledColumnType,
					Title:   "column.type-disallow-list",
					Content: "Disallow column type INT but column \"t\".\"a\" is",
					Line:    3,
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.StringArrayTypeRulePayload{
		List: []string{"int", "float", "bigint", "real"},
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &ColumnTypeDisallowListAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleColumnTypeDisallowList,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
