package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestRequirePK(t *testing.T) {
	tests := []test{
		{
			statement: "CREATE TABLE t(id INT PRIMARY KEY)",
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
			statement: "CREATE TABLE t(id INT, PRIMARY KEY (id))",
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
			statement: "CREATE TABLE t(id INT)",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			statement: `CREATE TABLE t(id INT);
						DROP TABLE t`,
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
			statement: `CREATE TABLE t(id INT);
						ALTER TABLE t ADD CONSTRAINT PRIMARY KEY (id)`,
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
			statement: `CREATE TABLE t(id INT PRIMARY KEY);
						ALTER TABLE t DROP PRIMARY KEY`,
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			statement: "CREATE TABLE t(id INT PRIMARY KEY);" +
				"ALTER TABLE t DROP INDEX `PRIMARY`",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			statement: `CREATE TABLE t(id INT);
						ALTER TABLE t ADD COLUMN name varchar(30) PRIMARY KEY`,
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
			statement: `CREATE TABLE t(id INT);
						ALTER TABLE t CHANGE COLUMN id id INT PRIMARY KEY`,
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
			// Use MockCatalogService
			statement: `ALTER TABLE t CHANGE COLUMN id uid INT`,
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
			statement: `CREATE TABLE t(id INT);
						ALTER TABLE t MODIFY COLUMN id INT PRIMARY KEY`,
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
			// Use MockCatalogService
			statement: `ALTER TABLE t MODIFY COLUMN id INT PRIMARY KEY`,
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
			statement: `CREATE TABLE t(id INT, name varchar(30), PRIMARY KEY(id, name));
						ALTER TABLE t DROP COLUMN id`,
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
			statement: `CREATE TABLE t(id INT, name varchar(30), comment varchar(255), PRIMARY KEY(id, name));
						ALTER TABLE t DROP COLUMN id, DROP COLUMN name`,
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			// Use MockCatalogService
			statement: `ALTER TABLE t DROP COLUMN id, DROP COLUMN name`,
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
		{
			// Use MockCatalogService
			statement: `ALTER TABLE t DROP COLUMN uid, DROP COLUMN name`,
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
			// Use MockCatalogService
			statement: `ALTER TABLE t CHANGE COLUMN id uid int;
						ALTER TABLE t DROP COLUMN uid, DROP COLUMN name`,
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.TableNoPK,
					Title:   "table.require-pk",
					Content: "Table `t` requires PRIMARY KEY",
				},
			},
		},
	}

	runSchemaReviewRuleTests(t, tests, &TableRequirePKAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleTableRequirePK,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, &MockCatalogService{})
}
