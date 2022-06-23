package mysql

import (
	"testing"

	_ "github.com/pingcap/tidb/types/parser_driver"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestBasic(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "DROP DATABASE d1",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropDatabase,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP DATABASE d1\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "DROP TABLE t1",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP TABLE t1\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "RENAME TABLE t1 to t2",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityRenameTable,
					Title:   "schema.backward-compatibility",
					Content: "\"RENAME TABLE t1 to t2\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "DROP VIEW v1",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP VIEW v1\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "CREATE UNIQUE INDEX idx1 ON t1 (f1)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddUniqueKey,
					Title:   "schema.backward-compatibility",
					Content: "\"CREATE UNIQUE INDEX idx1 ON t1 (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "DROP TABLE t1;DROP TABLE t2;",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP TABLE t1;\" may cause incompatibility with the existing data and code",
				},
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP TABLE t2;\" may cause incompatibility with the existing data and code",
				},
			},
		},
	}

	advisor.RunSchemaReviewRuleTests(t, tests, &CompatibilityAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleSchemaBackwardCompatibility,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, &advisor.MockCatalogService{})
}

func TestAlterTable(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE t1 ADD f1 TEXT",
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
			Statement: "ALTER TABLE t1 RENAME COLUMN f1 to f2",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityRenameColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 RENAME COLUMN f1 to f2\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 DROP COLUMN f1",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 DROP COLUMN f1\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD PRIMARY KEY (f1)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddPrimaryKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD PRIMARY KEY (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD UNIQUE (f1)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddUniqueKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD UNIQUE (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD UNIQUE KEY (f1)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddUniqueKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD UNIQUE KEY (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD UNIQUE INDEX (f1)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddUniqueKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD UNIQUE INDEX (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD FOREIGN KEY (f1) REFERENCES t2(f2)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddForeignKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD FOREIGN KEY (f1) REFERENCES t2(f2)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD CHECK (f1 > 0)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddCheck,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD CHECK (f1 > 0)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD CHECK (f1 > 0) NOT ENFORCED",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ALTER CHECK chk1 ENFORCED",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterCheck,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ALTER CHECK chk1 ENFORCED\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ALTER CHECK chk1 NOT ENFORCED",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD CONSTRAINT CHECK (f1 > 0)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddCheck,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD CONSTRAINT CHECK (f1 > 0)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD CONSTRAINT CHECK (f1 > 0) NOT ENFORCED",
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
			Statement: "ALTER TABLE t1 RENAME TO t2",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityRenameTable,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 RENAME TO t2\" may cause incompatibility with the existing data and code",
				},
			},
		},
	}

	advisor.RunSchemaReviewRuleTests(t, tests, &CompatibilityAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleSchemaBackwardCompatibility,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, &advisor.MockCatalogService{})
}

func TestAlterTableChangeColumnType(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE t1 CHANGE f1 f2 TEXT",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 CHANGE f1 f2 TEXT\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 MODIFY f1 TEXT",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 MODIFY f1 TEXT\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 MODIFY f1 TEXT NULL",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 MODIFY f1 TEXT NULL\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 MODIFY f1 TEXT NOT NULL",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 MODIFY f1 TEXT NOT NULL\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 MODIFY f1 TEXT COMMENT 'bla'",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 MODIFY f1 TEXT COMMENT 'bla'\" may cause incompatibility with the existing data and code",
				},
			},
		},
	}

	advisor.RunSchemaReviewRuleTests(t, tests, &CompatibilityAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleSchemaBackwardCompatibility,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, &advisor.MockCatalogService{})
}
