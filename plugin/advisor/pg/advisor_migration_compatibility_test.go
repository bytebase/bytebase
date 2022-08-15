package pg

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestBasic(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "DROP DATABASE d1",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityDropDatabase,
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
					Code:    advisor.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP TABLE t1\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 RENAME to t2",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityRenameTable,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 RENAME to t2\" may cause incompatibility with the existing data and code",
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
				},
			},
		},
		{
			Statement: "CREATE UNIQUE INDEX idx1 ON t1 (f1)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddUniqueKey,
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
					Code:    advisor.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP TABLE t1;\" may cause incompatibility with the existing data and code",
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityDropTable,
					Title:   "schema.backward-compatibility",
					Content: "\"DROP TABLE t2;\" may cause incompatibility with the existing data and code",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &CompatibilityAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleSchemaBackwardCompatibility,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockPostgreSQLDatabase)
}

func TestAlterTable(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE t1 ADD f1 TEXT",
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
			Statement: "ALTER TABLE t1 RENAME COLUMN f1 to f2",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityRenameColumn,
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
					Code:    advisor.CompatibilityDropColumn,
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
					Code:    advisor.CompatibilityAddPrimaryKey,
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
					Code:    advisor.CompatibilityAddUniqueKey,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD UNIQUE (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD FOREIGN KEY (f1) REFERENCES t2(f2)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddForeignKey,
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
					Code:    advisor.CompatibilityAddCheck,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD CHECK (f1 > 0)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD CHECK (f1 > 0) NOT VALID",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD CONSTRAINT check_f1_bigger_0 CHECK (f1 > 0)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAddCheck,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ADD CONSTRAINT check_f1_bigger_0 CHECK (f1 > 0)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			Statement: "ALTER TABLE t1 ADD CONSTRAINT check_f1_bigger_0 CHECK (f1 > 0) NOT VALID",
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
			Statement: "ALTER TABLE t1 RENAME TO t2",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityRenameTable,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 RENAME TO t2\" may cause incompatibility with the existing data and code",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &CompatibilityAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleSchemaBackwardCompatibility,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockPostgreSQLDatabase)
}

func TestAlterTableChangeColumnType(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE t1 ALTER COLUMN f1 TYPE TEXT",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.CompatibilityAlterColumn,
					Title:   "schema.backward-compatibility",
					Content: "\"ALTER TABLE t1 ALTER COLUMN f1 TYPE TEXT\" may cause incompatibility with the existing data and code",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &CompatibilityAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleSchemaBackwardCompatibility,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockPostgreSQLDatabase)
}
