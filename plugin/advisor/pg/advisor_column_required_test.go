package pg

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestColumnRequirement(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table \"book\" requires columns: created_ts, creator_id, updated_ts, updater_id",
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp)`,
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
			Statement: `ALTER TABLE book RENAME COLUMN creator_id TO creator;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table \"book\" requires columns: creator_id",
				},
			},
		},
		{
			Statement: `ALTER TABLE book DROP COLUMN creator_id;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table \"book\" requires columns: creator_id",
				},
			},
		},
	}
	payload, err := json.Marshal(advisor.RequiredColumnRulePayload{
		ColumnList: []string{
			"id",
			"created_ts",
			"updated_ts",
			"creator_id",
			"updater_id",
		},
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &ColumnRequirementAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleRequiredColumn,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
