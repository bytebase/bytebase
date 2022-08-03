package pg

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingFKConvention(t *testing.T) {
	invalidFKName := advisor.RandomString(42)
	maxLength := 32

	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_author_id_author_id FOREIGN KEY (author_id) REFERENCES author (id)",
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
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT fk_author_id FOREIGN KEY (author_id) REFERENCES author (id)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingFKConventionMismatch,
					Title:   "naming.index.fk",
					Content: "Foreign key in table \"tech_book\" mismatches the naming convention, expect \"^fk_tech_book_author_id_author_id$\" but found \"fk_author_id\"",
				},
			},
		},
		{
			Statement: fmt.Sprintf("ALTER TABLE tech_book ADD CONSTRAINT %s FOREIGN KEY (author_id) REFERENCES author (id)", invalidFKName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingFKConventionMismatch,
					Title:   "naming.index.fk",
					Content: fmt.Sprintf("Foreign key in table \"tech_book\" mismatches the naming convention, expect \"^fk_tech_book_author_id_author_id$\" but found \"%s\"", invalidFKName),
				},
				{
					Status:  advisor.Error,
					Code:    advisor.NamingFKConventionMismatch,
					Title:   "naming.index.fk",
					Content: fmt.Sprintf("Foreign key \"%s\" in table \"tech_book\" mismatches the naming convention, its length should be within %d characters", invalidFKName, maxLength),
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD COLUMN author_id INT CONSTRAINT fk_tech_book_author_id_author_id REFERENCES author (id)",
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
			Statement: "ALTER TABLE tech_book ADD COLUMN author_id INT CONSTRAINT fk_author_id REFERENCES author (id)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingFKConventionMismatch,
					Title:   "naming.index.fk",
					Content: "Foreign key in table \"tech_book\" mismatches the naming convention, expect \"^fk_tech_book_author_id_author_id$\" but found \"fk_author_id\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id INT, author_id INT, CONSTRAINT fk_book_author_id_author_id FOREIGN KEY (author_id) REFERENCES author (id))",
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
			Statement: "CREATE TABLE book(id INT, author_id INT, CONSTRAINT fk_book_author_id FOREIGN KEY (author_id) REFERENCES author (id))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingFKConventionMismatch,
					Title:   "naming.index.fk",
					Content: "Foreign key in table \"book\" mismatches the naming convention, expect \"^fk_book_author_id_author_id$\" but found \"fk_book_author_id\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id INT, author_id INT CONSTRAINT fk_book_author_id_author_id REFERENCES author (id))",
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
			Statement: "CREATE TABLE book(id INT, author_id INT CONSTRAINT fk_book_author_id REFERENCES author (id))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingFKConventionMismatch,
					Title:   "naming.index.fk",
					Content: "Foreign key in table \"book\" mismatches the naming convention, expect \"^fk_book_author_id_author_id$\" but found \"fk_book_author_id\"",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
		MaxLength: maxLength,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingFKConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleFKNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
