package mysql

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingUKConvention(t *testing.T) {
	invalidUKName := advisor.RandomString(65)

	tests := []advisor.TestCase{
		{
			Statement: "CREATE UNIQUE INDEX uk_tech_book_id_name ON tech_book(id, name)",
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
			Statement: "CREATE UNIQUE INDEX tech_book_id_name ON tech_book(id, name)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table `tech_book` mismatches the naming convention, expect \"^uk_tech_book_id_name$\" but found `tech_book_id_name`",
				},
			},
		},
		{
			Statement: fmt.Sprintf("CREATE UNIQUE INDEX %s ON tech_book(id, name)", invalidUKName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: fmt.Sprintf("Unique key in table `tech_book` mismatches the naming convention, expect \"^uk_tech_book_id_name$\" but found `%s`", invalidUKName),
				},
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: fmt.Sprintf("Unique key `%s` in table `tech_book` mismatches the naming convention, its length should be within 64 characters", invalidUKName),
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD UNIQUE uk_tech_book_id_name (id, name)",
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
			Statement: "ALTER TABLE tech_book ADD UNIQUE tech_book_id_name (id, name)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table `tech_book` mismatches the naming convention, expect \"^uk_tech_book_id_name$\" but found `tech_book_id_name`",
				},
			},
		},
		{
			Statement: fmt.Sprintf(
				"ALTER TABLE tech_book RENAME INDEX %s TO uk_tech_book_%s",
				advisor.MockOldUKName,
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
				"ALTER TABLE tech_book RENAME INDEX %s TO uk_tech_book",
				advisor.MockOldUKName,
			),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table `tech_book` mismatches the naming convention, expect \"^uk_tech_book_id_name$\" but found `uk_tech_book`",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE INDEX uk_tech_book_name (name))",
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
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE KEY (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table `tech_book` mismatches the naming convention, expect \"^uk_tech_book_name$\" but found ``",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE INDEX (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table `tech_book` mismatches the naming convention, expect \"^uk_tech_book_name$\" but found ``",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE KEY (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table `tech_book` mismatches the naming convention, expect \"^uk_tech_book_name$\" but found ``",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^uk_{{table}}_{{column_list}}$",
		MaxLength: 64,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingUKConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleUKNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, advisor.MockMySQLDatabase)
}
