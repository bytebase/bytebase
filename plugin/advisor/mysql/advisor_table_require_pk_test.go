package mysql

import (
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
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			Statement: `CREATE TABLE t(id INT);
						DROP TABLE t`,
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
			Statement: `CREATE TABLE t(id INT);
						ALTER TABLE t ADD CONSTRAINT PRIMARY KEY (id)`,
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
			Statement: `CREATE TABLE t(id INT PRIMARY KEY);
						ALTER TABLE t DROP PRIMARY KEY`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			Statement: "CREATE TABLE t(id INT PRIMARY KEY);" +
				"ALTER TABLE t DROP INDEX `PRIMARY`",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			Statement: `CREATE TABLE t(id INT);
						ALTER TABLE t ADD COLUMN name varchar(30) PRIMARY KEY`,
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
			Statement: `CREATE TABLE t(id INT);
						ALTER TABLE t CHANGE COLUMN id id INT PRIMARY KEY`,
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
			// Use MockCatalogService
			Statement: `ALTER TABLE t CHANGE COLUMN id uid INT`,
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
			Statement: `CREATE TABLE t(id INT);
						ALTER TABLE t MODIFY COLUMN id INT PRIMARY KEY`,
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
			// Use MockCatalogService
			Statement: `ALTER TABLE t MODIFY COLUMN id INT PRIMARY KEY`,
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
			Statement: `CREATE TABLE t(id INT, name varchar(30), PRIMARY KEY(id, name));
						ALTER TABLE t DROP COLUMN id`,
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
			Statement: `CREATE TABLE t(id INT, name varchar(30), comment varchar(255), PRIMARY KEY(id, name));
						ALTER TABLE t DROP COLUMN id, DROP COLUMN name`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			// Use MockCatalogService
			Statement: `ALTER TABLE t DROP COLUMN id, DROP COLUMN name`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			// Use MockCatalogService
			Statement: `ALTER TABLE tech_book DROP COLUMN uid, DROP COLUMN name`,
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
			// Use MockCatalogService
			Statement: `ALTER TABLE tech_book CHANGE COLUMN id uid int;
						ALTER TABLE tech_book DROP COLUMN uid, DROP COLUMN name`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `tech_book` requires PRIMARY KEY",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &TableRequirePKAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleTableRequirePK,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}
