package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingFKConvention(t *testing.T) {
	tests := []test{
		{
			statement: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_author_id_author_id FOREIGN KEY (author_id) REFERENCES author (id)",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			statement: "ALTER TABLE tech_book ADD CONSTRAINT fk_author_id FOREIGN KEY (author_id) REFERENCES author (id)",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingFKConventionMismatch,
					Title:   "Mismatch foreign key naming convention",
					Content: "\"ALTER TABLE tech_book ADD CONSTRAINT fk_author_id FOREIGN KEY (author_id) REFERENCES author (id)\" mismatches foreign key naming convention, expect \"^fk_tech_book_author_id_author_id$\" but found \"fk_author_id\"",
				},
			},
		},
		{
			statement: "CREATE TABLE book(id INT, author_id INT, FOREIGN KEY fk_book_author_id_author_id (author_id) REFERENCES author (id))",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			statement: "CREATE TABLE book(id INT, author_id INT, FOREIGN KEY fk_book_author_id (author_id) REFERENCES author (id))",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingFKConventionMismatch,
					Title:   "Mismatch foreign key naming convention",
					Content: "\"CREATE TABLE book(id INT, author_id INT, FOREIGN KEY fk_book_author_id (author_id) REFERENCES author (id))\" mismatches foreign key naming convention, expect \"^fk_book_author_id_author_id$\" but found \"fk_book_author_id\"",
				},
			},
		},
	}

	payload, err := json.Marshal(api.NamingRulePayload{
		Format: "^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &NamingFKConventionAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleFKNaming,
		Level:   api.SchemaRuleLevelError,
		Payload: string(payload),
	}, &MockCatalogService{})
}
