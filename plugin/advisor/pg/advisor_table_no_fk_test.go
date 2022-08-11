package pg

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestTableNoFK(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_author_id_author_id FOREIGN KEY (author_id) REFERENCES author (id)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableHasFK,
					Title:   "table.no-foreign-key",
					Content: "Foreign key is not allowed in the table \"public\".\"tech_book\", related statement: \"ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_author_id_author_id FOREIGN KEY (author_id) REFERENCES author (id)\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id INT, author_id INT, CONSTRAINT fk_book_author_id_author_id FOREIGN KEY (author_id) REFERENCES author (id))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableHasFK,
					Title:   "table.no-foreign-key",
					Content: "Foreign key is not allowed in the table \"public\".\"book\", related statement: \"CREATE TABLE book(id INT, author_id INT, CONSTRAINT fk_book_author_id_author_id FOREIGN KEY (author_id) REFERENCES author (id))\"",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &TableNoFKAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleTableNoFK,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, advisor.MockPostgreSQLDatabase)
}
