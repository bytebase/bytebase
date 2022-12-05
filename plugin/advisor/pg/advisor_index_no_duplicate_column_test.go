package pg

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestIndexNoDuplicateColumn(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: `CREATE TABLE t (a INT, PRIMARY KEY (a));`,
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
			Statement: `CREATE TABLE t (
				a int,
				PRIMARY KEY (a, a));`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.DuplicateColumnInIndex,
					Title:   "index.no-duplicate-column",
					Content: "PRIMARY KEY \"\" has duplicate column \"t\".\"a\"",
					Line:    3,
				},
			},
		},
		{
			Statement: `
				CREATE TABLE t(a int);
				CREATE INDEX idx_a on t(a, a);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.DuplicateColumnInIndex,
					Title:   "index.no-duplicate-column",
					Content: "INDEX \"idx_a\" has duplicate column \"t\".\"a\"",
					Line:    3,
				},
			},
		},
		{
			Statement: `
				CREATE TABLE t(a int);
				ALTER TABLE t ADD CONSTRAINT uk_a UNIQUE (a, a);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.DuplicateColumnInIndex,
					Title:   "index.no-duplicate-column",
					Content: "UNIQUE KEY \"uk_a\" has duplicate column \"t\".\"a\"",
					Line:    3,
				},
			},
		},
		{
			Statement: `
				CREATE TABLE t(a int);
				ALTER TABLE t ADD CONSTRAINT pk_a PRIMARY KEY (a, a);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.DuplicateColumnInIndex,
					Title:   "index.no-duplicate-column",
					Content: "PRIMARY KEY \"pk_a\" has duplicate column \"t\".\"a\"",
					Line:    3,
				},
			},
		},
		{
			Statement: `
				CREATE TABLE t(a int);
				ALTER TABLE t ADD CONSTRAINT fk_a FOREIGN KEY (a, a) REFERENCES t1(a, b);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.DuplicateColumnInIndex,
					Title:   "index.no-duplicate-column",
					Content: "FOREIGN KEY \"fk_a\" has duplicate column \"t\".\"a\"",
					Line:    3,
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &IndexNoDuplicateColumnAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleIndexNoDuplicateColumn,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockPostgreSQLDatabase)
}
