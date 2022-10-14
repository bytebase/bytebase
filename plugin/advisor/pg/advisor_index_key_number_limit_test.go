package pg

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestIndexKeyNumberLimitAdvisor(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: `CREATE TABLE t(name char(20));`,
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
			Statement: `CREATE TABLE t(name varchar(225));`,
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
			Statement: `CREATE TABLE t(name char(225), PRIMARY KEY (name));`,
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
			Statement: `CREATE TABLE t(id int, name char(225), CONSTRAINT t_id_name PRIMARY KEY (id, name));`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.IndexKeyNumberExceedsLimit,
					Title:   "index.key-number-limit",
					Content: "The number of index `t_id_name` in table `t` should be not greater than 1",
					Line:    1,
				},
			},
		},
		{
			Statement: `CREATE TABLE t(id int, name char(225), CONSTRAINT t_id_name UNIQUE (id, name));`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.IndexKeyNumberExceedsLimit,
					Title:   "index.key-number-limit",
					Content: "The number of index `t_id_name` in table `t` should be not greater than 1",
					Line:    1,
				},
			},
		},
		{
			Statement: `CREATE INDEX idx_address_phone ON address(id, phone);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.IndexKeyNumberExceedsLimit,
					Title:   "index.key-number-limit",
					Content: "The number of index `idx_address_phone` in table `address` should be not greater than 1",
					Line:    1,
				},
			},
		},
		{
			Statement: `CREATE UNIQUE INDEX idx_address_phone ON address(id, phone);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.IndexKeyNumberExceedsLimit,
					Title:   "index.key-number-limit",
					Content: "The number of index `idx_address_phone` in table `address` should be not greater than 1",
					Line:    1,
				},
			},
		},
		{
			Statement: `ALTER TABLE t ADD CONSTRAINT t_id_name UNIQUE (id, name);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.IndexKeyNumberExceedsLimit,
					Title:   "index.key-number-limit",
					Content: "The number of index `t_id_name` in table `t` should be not greater than 1",
					Line:    1,
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NumberTypeRulePayload{
		Number: 1,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &IndexKeyNumberLimitAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleIndexKeyNumberLimit,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
