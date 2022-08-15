package pg

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingIndexConvention(t *testing.T) {
	invalidIndexName := advisor.RandomString(33)

	tests := []advisor.TestCase{
		{
			Statement: "CREATE INDEX idx_tech_book_id_name ON tech_book(id, name)",
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
			Statement: "CREATE INDEX tech_book_id_name ON tech_book(id, name)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingIndexConventionMismatch,
					Title:   "naming.index.idx",
					Content: "Index in table \"tech_book\" mismatches the naming convention, expect \"^idx_tech_book_id_name$\" but found \"tech_book_id_name\"",
				},
			},
		},
		{
			Statement: fmt.Sprintf("CREATE INDEX %s ON tech_book(id, name)", invalidIndexName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingIndexConventionMismatch,
					Title:   "naming.index.idx",
					Content: fmt.Sprintf("Index in table \"tech_book\" mismatches the naming convention, expect \"^idx_tech_book_id_name$\" but found \"%s\"", invalidIndexName),
				},
				{
					Status:  advisor.Error,
					Code:    advisor.NamingIndexConventionMismatch,
					Title:   "naming.index.idx",
					Content: fmt.Sprintf("Index \"%s\" in table \"tech_book\" mismatches the naming convention, its length should be within 32 characters", invalidIndexName),
				},
			},
		},
		{
			Statement: fmt.Sprintf(
				"ALTER INDEX %s RENAME TO idx_tech_book_%s",
				advisor.MockOldIndexName,
				strings.Join(advisor.MockIndexColumnList, "_"),
			),
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
			Statement: fmt.Sprintf(
				"ALTER INDEX %s RENAME TO idx_tech_book",
				advisor.MockOldIndexName,
			),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingIndexConventionMismatch,
					Title:   "naming.index.idx",
					Content: "Index in table \"tech_book\" mismatches the naming convention, expect \"^idx_tech_book_id_name$\" but found \"idx_tech_book\"",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^idx_{{table}}_{{column_list}}$",
		MaxLength: 32,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingIndexConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleIDXNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
