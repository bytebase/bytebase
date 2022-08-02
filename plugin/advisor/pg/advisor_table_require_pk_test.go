package pg

import (
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestRequirePK(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE t(id INT PRIMARY KEY)",
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
			Statement: "CREATE TABLE t(id INT, PRIMARY KEY (id))",
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
			Statement: "CREATE TABLE t(id INT)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table \"public\".\"t\" requires PRIMARY KEY, related statement: \"CREATE TABLE t(id INT)\"",
				},
			},
		},
		{
			Statement: fmt.Sprintf("ALTER TABLE %q DROP CONSTRAINT %q", advisor.MockTableName, advisor.MockOldPostgreSQLPKName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table \"public\".\"tech_book\" requires PRIMARY KEY, related statement: \"ALTER TABLE \\\"tech_book\\\" DROP CONSTRAINT \\\"old_pk\\\"\"",
				},
			},
		},
		{
			Statement: fmt.Sprintf("ALTER TABLE %q DROP CONSTRAINT %q", advisor.MockTableName, advisor.MockOldIndexName),
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
			Statement: fmt.Sprintf("ALTER TABLE %q DROP CONSTRAINT constraint_not_in_catalog", advisor.MockTableName),
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
			Statement: fmt.Sprintf("ALTER TABLE %q DROP COLUMN id", advisor.MockTableName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table \"public\".\"tech_book\" requires PRIMARY KEY, related statement: \"ALTER TABLE \\\"tech_book\\\" DROP COLUMN id\"",
				},
			},
		},
		{
			Statement: fmt.Sprintf("ALTER TABLE %q DROP COLUMN column_not_in_pk", advisor.MockTableName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &TableRequirePKAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleTableRequirePK,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, advisor.MockPostgreSQLDatabase)
}
