package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/advisor"

	_ "github.com/bytebase/bytebase/backend/plugin/parser/engine/pg"
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
					Line:    1,
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
