package mysql

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingIndexConvention(t *testing.T) {
	invalidIndexName := advisor.RandomString(65)

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
					Content: "Index in table `tech_book` mismatches the naming convention, expect \"^idx_tech_book_id_name$\" but found `tech_book_id_name`",
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
					Content: fmt.Sprintf("Index in table `tech_book` mismatches the naming convention, expect \"^idx_tech_book_id_name$\" but found `%s`", invalidIndexName),
				},
				{
					Status:  advisor.Error,
					Code:    advisor.NamingIndexConventionMismatch,
					Title:   "naming.index.idx",
					Content: fmt.Sprintf("Index `%s` in table `tech_book` mismatches the naming convention, its length should be within 64 characters", invalidIndexName),
				},
			},
		},
		{
			Statement: fmt.Sprintf(
				"ALTER TABLE tech_book RENAME INDEX %s TO idx_tech_book_%s",
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
				"ALTER TABLE tech_book RENAME INDEX %s TO idx_tech_book",
				advisor.MockOldIndexName,
			),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingIndexConventionMismatch,
					Title:   "naming.index.idx",
					Content: "Index in table `tech_book` mismatches the naming convention, expect \"^idx_tech_book_id_name$\" but found `idx_tech_book`",
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD INDEX idx_tech_book_id_name (id, name)",
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
			Statement: "ALTER TABLE tech_book ADD INDEX tech_book_id_name (id, name)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingIndexConventionMismatch,
					Title:   "naming.index.idx",
					Content: "Index in table `tech_book` mismatches the naming convention, expect \"^idx_tech_book_id_name$\" but found `tech_book_id_name`",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), INDEX idx_tech_book_name (name))",
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
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), INDEX (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingIndexConventionMismatch,
					Title:   "naming.index.idx",
					Content: "Index in table `tech_book` mismatches the naming convention, expect \"^idx_tech_book_name$\" but found ``",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^idx_{{table}}_{{column_list}}$",
		MaxLength: 64,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingIndexConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleIDXNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, advisor.MockMySQLDatabase)
}
