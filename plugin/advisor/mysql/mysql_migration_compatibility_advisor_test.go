package mysql

import (
	"reflect"
	"testing"

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
					Title:   "Incompatible migration",
					Content: "DROP DATABASE d1 is backward incompatible",
				},
			},
		},
		{
			statement: "DROP TABLE t1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "DROP TABLE t1 is backward incompatible",
				},
			},
		},
		{
			statement: "RENAME TABLE t1 to t2",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "RENAME TABLE t1 to t2 is backward incompatible",
				},
			},
		},
		{
			statement: "DROP VIEW v1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "DROP VIEW v1 is backward incompatible",
				},
			},
		},
		{
			statement: "CREATE UNIQUE INDEX idx1 ON t1 (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "CREATE UNIQUE INDEX idx1 ON t1 (f1) is backward incompatible",
				},
			},
		},
		{
			statement: "DROP TABLE t1;DROP TABLE t2;",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "DROP TABLE t1; is backward incompatible",
				},
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "DROP TABLE t2; is backward incompatible",
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
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 RENAME COLUMN f1 to f2 is backward incompatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 DROP COLUMN f1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 DROP COLUMN f1 is backward incompatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD PRIMARY KEY (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 ADD PRIMARY KEY (f1) is backward incompatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD UNIQUE (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 ADD UNIQUE (f1) is backward incompatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD UNIQUE KEY (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 ADD UNIQUE KEY (f1) is backward incompatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD UNIQUE INDEX (f1)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 ADD UNIQUE INDEX (f1) is backward incompatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD FOREIGN KEY (f1) REFERENCES t2(f2)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 ADD FOREIGN KEY (f1) REFERENCES t2(f2) is backward incompatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD CHECK (f1 > 0)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 ADD CHECK (f1 > 0) is backward incompatible",
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
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 ALTER CHECK chk1 ENFORCED is backward incompatible",
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
					Title:   "Incompatible migration",
					Content: "ALTER TABLE t1 ADD CONSTRAINT CHECK (f1 > 0) is backward incompatible",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD CONSTRAINT CHECK (f1 > 0) NOT ENFORCED",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Title:   "OK",
					Content: "Migration is backward compatible",
				},
			},
		},
	}

	runTests(t, tests)
}
