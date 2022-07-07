//go:build !release
// +build !release

package pg

import (
	"testing"

	"github.com/bmizerany/assert"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"

	_ "github.com/bytebase/bytebase/plugin/parser/engine/pg"
)

func TestPostgreSQLSyntax(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int);",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "Syntax OK",
					Content: "OK",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int) ENGINE=INNODB;",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.StatementSyntaxError,
					Title:   "Syntax error",
					Content: "syntax error at or near \"ENGINE\"",
				},
			},
		},
	}

	adv := &SyntaxAdvisor{}

	for _, tc := range tests {
		adviceList, err := adv.Check(advisor.Context{}, tc.Statement)
		require.NoError(t, err)
		assert.Equal(t, tc.Want, adviceList)
	}
}
