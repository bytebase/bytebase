package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestBYT9367_ParseStatementsClassification verifies that for inputs where a
// trailing ';' is separated from the statement's parse-tree stop by hidden tokens
// (the BYT-9367 shape), GetStatementTypes correctly classifies BOTH statements,
// not just the first.
func TestBYT9367_ParseStatementsClassification(t *testing.T) {
	cases := []struct {
		name      string
		statement string
	}{
		{"cell_b_space_before_semi", "insert into t values('a',1) ;\n\ninsert into t values('b',2) ;"},
		{"cell_c_comment_before_semi", "insert into t values('a',1) /* note */ ;\ninsert into t values('b',2) ;"},
		{"cell_d_multinewline_before_semi", "insert into t values('a',1)\n\n;\n\ninsert into t values('b',2);"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmts, err := base.ParseStatements(storepb.Engine_ORACLE, tc.statement)
			require.NoError(t, err)
			require.Len(t, stmts, 2, "expected 2 ParsedStatement entries")

			asts := base.ExtractASTs(stmts)
			require.Len(t, asts, 2, "expected 2 ASTs")

			types, err := GetStatementTypes(asts)
			require.NoError(t, err)

			// Both statements are INSERT — set should contain only INSERT.
			require.Equal(t, []storepb.StatementType{storepb.StatementType_INSERT}, types,
				"expected both statements classified as INSERT, got %v", types)
		})
	}
}
