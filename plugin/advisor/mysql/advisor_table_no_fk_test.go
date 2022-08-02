package mysql

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
					Content: "Foreign key is not allowed in the table `tech_book`",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id INT, author_id INT, FOREIGN KEY fk_book_author_id_author_id (author_id) REFERENCES author (id))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableHasFK,
					Title:   "table.no-foreign-key",
					Content: "Foreign key is not allowed in the table `book`",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &TableNoFKAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleTableNoFK,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}
