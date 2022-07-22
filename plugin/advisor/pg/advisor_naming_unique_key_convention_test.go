package pg

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingUKConvention(t *testing.T) {
	invalidUKName := advisor.RandomString(42)
	maxLength := 32

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
					Content: "Unique key in table \"tech_book\" mismatches the naming convention, expect \"^uk_tech_book_id_name$\" but found \"tech_book_id_name\"",
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
					Content: fmt.Sprintf("Unique key in table \"tech_book\" mismatches the naming convention, expect \"^uk_tech_book_id_name$\" but found \"%s\"", invalidUKName),
				},
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: fmt.Sprintf("Unique key \"%s\" in table \"tech_book\" mismatches the naming convention, its length should be within %d characters", invalidUKName, maxLength),
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT uk_tech_book_id_name UNIQUE (id, name)",
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
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT tech_book_id_name UNIQUE (id, name)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table \"tech_book\" mismatches the naming convention, expect \"^uk_tech_book_id_name$\" but found \"tech_book_id_name\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), CONSTRAINT uk_tech_book_name UNIQUE (name))",
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
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), CONSTRAINT tech_book_name UNIQUE (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table \"tech_book\" mismatches the naming convention, expect \"^uk_tech_book_name$\" but found \"tech_book_name\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table \"tech_book\" mismatches the naming convention, expect \"^uk_tech_book_name$\" but found \"\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20) UNIQUE)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table \"tech_book\" mismatches the naming convention, expect \"^uk_tech_book_name$\" but found \"\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingUKConventionMismatch,
					Title:   "naming.index.uk",
					Content: "Unique key in table \"tech_book\" mismatches the naming convention, expect \"^uk_tech_book_name$\" but found \"\"",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^uk_{{table}}_{{column_list}}$",
		MaxLength: maxLength,
	})
	require.NoError(t, err)
	advisor.RunSchemaReviewRuleTests(t, tests, &NamingUKConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleUKNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, &advisor.MockCatalogService{})
}
