package mysql

import (
	"reflect"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"go.uber.org/zap"
)

func TestMigraionCompatibilityAdvisor(t *testing.T) {
	type test struct {
		statement string
		want      []advisor.Advice
	}

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
