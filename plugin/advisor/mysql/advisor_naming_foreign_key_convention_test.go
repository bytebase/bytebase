package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingFKConvention(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_author_id_author_id FOREIGN KEY (author_id) REFERENCES author (id)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
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
					Code:    common.NamingFKConventionMismatch,
					Title:   "naming.index.fk",
					Content: "Foreign key in table `tech_book` mismatches the naming convention, expect \"^fk_tech_book_author_id_author_id$\" but found `fk_author_id`",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id INT, author_id INT, FOREIGN KEY fk_book_author_id_author_id (author_id) REFERENCES author (id))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id INT, author_id INT, FOREIGN KEY fk_book_author_id (author_id) REFERENCES author (id))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingFKConventionMismatch,
					Title:   "naming.index.fk",
					Content: "Foreign key in table `book` mismatches the naming convention, expect \"^fk_book_author_id_author_id$\" but found `fk_book_author_id`",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format: "^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
	})
	require.NoError(t, err)
	advisor.RunSchemaReviewRuleTests(t, tests, &NamingFKConventionAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleFKNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, &advisor.MockCatalogService{})
}
