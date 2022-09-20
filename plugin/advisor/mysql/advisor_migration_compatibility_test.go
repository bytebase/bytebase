package mysql

import (
	"testing"

	_ "github.com/pingcap/tidb/types/parser_driver"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestBasic(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "DROP DATABASE test",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityDropDatabase,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP DATABASE test\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "DROP TABLE tech_book",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP TABLE tech_book\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "RENAME TABLE tech_book to t2",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityRenameTable,
					Title:   "schema.backward-compatibility",
					Content: "\"RENAME TABLE tech_book to t2\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "DROP VIEW v1",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP VIEW v1\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "CREATE UNIQUE INDEX idx1 ON tech_book (id)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddUniqueKey,
					Title:   "schema.backward-compatibility",
					Content: "\"CREATE UNIQUE INDEX idx1 ON tech_book (id)\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "CREATE TABLE t1(a int);DROP TABLE t1;DROP TABLE tech_book;",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP TABLE t1;\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP TABLE tech_book;\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: `
				CREATE TABLE t(a int);
				CREATE UNIQUE INDEX uk_t_a on t(a);
			`,
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
			Statement: `
				CREATE TABLE t(a int);
				ALTER TABLE t ADD PRIMARY KEY (a);
			`,
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

	advisor.RunSQLReviewRuleTests(t, tests, &CompatibilityAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleSchemaBackwardCompatibility,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}

func TestAlterTable(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE tech_book ADD f1 TEXT",
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
			Statement: "ALTER TABLE tech_book RENAME COLUMN id to f2",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityRenameColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book RENAME COLUMN id to f2\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book DROP COLUMN id",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityDropColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book DROP COLUMN id\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book DROP PRIMARY KEY;ALTER TABLE tech_book ADD PRIMARY KEY (id)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddPrimaryKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book ADD PRIMARY KEY (id)\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD UNIQUE (id)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddUniqueKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book ADD UNIQUE (id)\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD UNIQUE KEY (id)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddUniqueKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book ADD UNIQUE KEY (id)\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD UNIQUE INDEX (id)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddUniqueKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book ADD UNIQUE INDEX (id)\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD FOREIGN KEY (id) REFERENCES t2(f2)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddForeignKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book ADD FOREIGN KEY (id) REFERENCES t2(f2)\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD CHECK (id > 0)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddCheck,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book ADD CHECK (id > 0)\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD CHECK (id > 0) NOT ENFORCED",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ALTER CHECK chk1 ENFORCED",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAlterCheck,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book ALTER CHECK chk1 ENFORCED\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ALTER CHECK chk1 NOT ENFORCED",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT CHECK (id > 0)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddCheck,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book ADD CONSTRAINT CHECK (id > 0)\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT CHECK (id > 0) NOT ENFORCED",
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
			Statement: "ALTER TABLE tech_book RENAME TO t2",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityRenameTable,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book RENAME TO t2\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &CompatibilityAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleSchemaBackwardCompatibility,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}

func TestAlterTableChangeColumnType(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE tech_book CHANGE name f2 TEXT",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book CHANGE name f2 TEXT\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book MODIFY name TEXT",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book MODIFY name TEXT\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book MODIFY name TEXT NULL",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book MODIFY name TEXT NULL\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book MODIFY name TEXT NOT NULL",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book MODIFY name TEXT NOT NULL\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book MODIFY name TEXT COMMENT 'bla'",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE tech_book MODIFY name TEXT COMMENT 'bla'\" may cause incompatibility with the existing data and code",
					Line:    1,
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &CompatibilityAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleSchemaBackwardCompatibility,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}
