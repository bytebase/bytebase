package pg

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestCharsetAllowlist(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: `CREATE DATABASE test WITH ENCODING 'UTF8'`,
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
			Statement: `CREATE DATABASE test`,
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
			Statement: `CREATE DATABASE test WITH ENCODING 'LATIN1'`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.DisabledCharset,
					Title:   "system.charset.allowlist",
					Content: "\"\" used disabled encoding 'latin1'",
				},
			},
		},
		{
			Statement: "/* this is a comment */",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.StringArrayTypeRulePayload{
		List: []string{"utf8"},
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &EncodingAllowlistAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleCharsetAllowlist,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
