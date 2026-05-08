package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestBYT9367_ParseStatementsClassification verifies that for inputs where a
// trailing ';' is separated from the statement's parse-tree stop by hidden
// tokens (the BYT-9367 shape), each per-AST classification is correct.
//
// Asserts per-AST (not the deduplicated GetStatementTypes set) so a single
// misclassified AST is detected even when other ASTs share its type.
func TestBYT9367_ParseStatementsClassification(t *testing.T) {
	cases := []struct {
		name      string
		statement string
		want      []storepb.StatementType
	}{
		{
			name:      "cell_b_space_before_semi",
			statement: "insert into t values('a',1) ;\n\ninsert into t values('b',2) ;",
			want:      []storepb.StatementType{storepb.StatementType_INSERT, storepb.StatementType_INSERT},
		},
		{
			name:      "cell_c_comment_before_semi",
			statement: "insert into t values('a',1) /* note */ ;\ninsert into t values('b',2) ;",
			want:      []storepb.StatementType{storepb.StatementType_INSERT, storepb.StatementType_INSERT},
		},
		{
			name:      "cell_d_multinewline_before_semi",
			statement: "insert into t values('a',1)\n\n;\n\ninsert into t values('b',2);",
			want:      []storepb.StatementType{storepb.StatementType_INSERT, storepb.StatementType_INSERT},
		},
		{
			// Cell f: bail-on-default-channel branch. The loop walks past '\n'
			// and bails on SELECT, leaving prevStopTokenIndex unchanged. Locks
			// AST count = 2 so the bail branch can't accidentally consume the
			// next statement's first token.
			name:      "cell_f_no_separator",
			statement: "BEGIN NULL; END;\nSELECT 1 FROM dual",
			want:      []storepb.StatementType{storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		},
		{
			// Cell h: trailing ';' with hidden tokens before it at EOF. Loop
			// fires correctly at the end of the token stream (no out-of-bounds).
			name:      "cell_h_eof_after_semi",
			statement: "insert into t values('a',1) ;",
			want:      []storepb.StatementType{storepb.StatementType_INSERT},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmts, err := base.ParseStatements(storepb.Engine_ORACLE, tc.statement)
			require.NoError(t, err)

			asts := base.ExtractASTs(stmts)
			require.Len(t, asts, len(tc.want))

			got := make([]storepb.StatementType, len(asts))
			for i, ast := range asts {
				antlrAST, ok := base.GetANTLRAST(ast)
				require.True(t, ok, "expected ANTLR AST")
				got[i] = getStatementType(antlrAST.Tree)
			}
			require.Equal(t, tc.want, got)
		})
	}
}
