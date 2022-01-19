package mysql

import (
	"reflect"
	"testing"

	_ "github.com/pingcap/tidb/types/parser_driver"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"go.uber.org/zap"
)

type test struct {
	statement string
	want      []advisor.Advice
}

func runTests(t *testing.T, tests []test) {
	adv := CompatibilityAdvisor{}
	logger, _ := zap.NewDevelopmentConfig().Build()
	ctx := advisor.AdvisorContext{
		Logger:    logger,
		Charset:   "",
		Collation: "",
	}
	for _, tc := range tests {
		adviceList, err := adv.Check(ctx, tc.statement)
		if err != nil {
			t.Errorf("statement=%s: expected no error, got %v", tc.statement, err)
		} else {
			if !reflect.DeepEqual(tc.want, adviceList) {
				t.Errorf("statement=%s: expected %+v, got %+v", tc.statement, tc.want, adviceList)
			}
		}
	}
}

func TestBasic(t *testing.T) {
	tests := []test{
		{
			statement: "DROP DATABASE d1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropDatabase,
					Title:   "Potential incompatible migration",
					Content: "\"DROP DATABASE d1\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "DROP TABLE t1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropTable,
					Title:   "Potential incompatible migration",
					Content: "\"DROP TABLE t1\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "RENAME TABLE t1 to t2",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityRenameTable,
					Title:   "Potential incompatible migration",
					Content: "\"RENAME TABLE t1 to t2\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "DROP VIEW v1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropTable,
					Title:   "Potential incompatible migration",
					Content: "\"DROP VIEW v1\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "CREATE UNIQUE INDEX idx1 ON t1 (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddUniqueKey,
					Title:   "Potential incompatible migration",
					Content: "\"CREATE UNIQUE INDEX idx1 ON t1 (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "DROP TABLE t1;DROP TABLE t2;",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropTable,
					Title:   "Potential incompatible migration",
					Content: "\"DROP TABLE t1;\" may cause incompatibility with the existing data and code",
				},
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropTable,
					Title:   "Potential incompatible migration",
					Content: "\"DROP TABLE t2;\" may cause incompatibility with the existing data and code",
				},
			},
		},
	}

	runTests(t, tests)
}

func TestAlterTable(t *testing.T) {
	tests := []test{
		{
			statement: "ALTER TABLE t1 ADD f1 TEXT",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "Migration is backward compatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 RENAME COLUMN f1 to f2",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityRenameColumn,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 RENAME COLUMN f1 to f2\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 DROP COLUMN f1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityDropColumn,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 DROP COLUMN f1\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD PRIMARY KEY (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddPrimaryKey,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 ADD PRIMARY KEY (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD UNIQUE (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddUniqueKey,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 ADD UNIQUE (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD UNIQUE KEY (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddUniqueKey,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 ADD UNIQUE KEY (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD UNIQUE INDEX (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddUniqueKey,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 ADD UNIQUE INDEX (f1)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD FOREIGN KEY (f1) REFERENCES t2(f2)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddForeignKey,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 ADD FOREIGN KEY (f1) REFERENCES t2(f2)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD CHECK (f1 > 0)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddCheck,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 ADD CHECK (f1 > 0)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD CHECK (f1 > 0) NOT ENFORCED",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Title:   "OK",
					Content: "Migration is backward compatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ALTER CHECK chk1 ENFORCED",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterCheck,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 ALTER CHECK chk1 ENFORCED\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ALTER CHECK chk1 NOT ENFORCED",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Title:   "OK",
					Content: "Migration is backward compatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD CONSTRAINT CHECK (f1 > 0)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAddCheck,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 ADD CONSTRAINT CHECK (f1 > 0)\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD CONSTRAINT CHECK (f1 > 0) NOT ENFORCED",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "Migration is backward compatible",
				},
			},
		},
	}

	runTests(t, tests)
}

func TestAlterTableChangeColumnType(t *testing.T) {
	tests := []test{
		{
			statement: "ALTER TABLE t1 CHANGE f1 f2 TEXT",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 CHANGE f1 f2 TEXT\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 MODIFY f1 TEXT",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 MODIFY f1 TEXT\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 MODIFY f1 TEXT NULL",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 MODIFY f1 TEXT NULL\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 MODIFY f1 TEXT NOT NULL",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 MODIFY f1 TEXT NOT NULL\" may cause incompatibility with the existing data and code",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 MODIFY f1 TEXT COMMENT 'bla'",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.CompatibilityAlterColumn,
					Title:   "Potential incompatible migration",
					Content: "\"ALTER TABLE t1 MODIFY f1 TEXT COMMENT 'bla'\" may cause incompatibility with the existing data and code",
				},
			},
		},
	}

	runTests(t, tests)
}

func TestMysql8WindowFunction(t *testing.T) {
	tests := []test{
		{
			statement: "SELECT row_number() OVER ( ORDER BY id ), id FROM xxx;",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "Migration is backward compatible",
				},
			},
		},
	}

	runTests(t, tests)
}
