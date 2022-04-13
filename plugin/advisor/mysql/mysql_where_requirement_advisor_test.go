package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestWhereRequirement(t *testing.T) {
	tests := []test{
		{
			statement: "DELETE FROM t1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.StatementNoWhere,
					Title:   "Potential modification of unexpected data",
					Content: "\"DELETE FROM t1\" may modify unexpected data",
				},
			},
		},
		{
			statement: "UPDATE t1 SET a = 1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.StatementNoWhere,
					Title:   "Potential modification of unexpected data",
					Content: "\"UPDATE t1 SET a = 1\" may modify unexpected data",
				},
			},
		},
		{
			statement: "DELETE FROM t1 WHERE a > 0",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "DELETE/UPDATE statements contain WHERE clause",
				},
			},
		},
		{
			statement: "UPDATE t1 SET a = 1 WHERE a > 10",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "DELETE/UPDATE statements contain WHERE clause",
				},
			},
		},
	}

	runTests(t, tests, &WhereRequirementAdvisor{})
}
