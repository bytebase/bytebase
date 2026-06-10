package snowflake

import (
	"testing"

	omniast "github.com/bytebase/omni/snowflake/ast"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestParseSnowflakeStatementsDualAST locks the dual-AST transition contract:
// every non-empty statement carries ONE OmniAST wrapping BOTH the omni node
// (for migrated advisors, via GetOmniNode) and the legacy ANTLR tree (for the
// un-migrated ANTLR advisors, via base.GetANTLRAST).
func TestParseSnowflakeStatementsDualAST(t *testing.T) {
	parsed, err := parseSnowflakeStatements("CREATE TABLE t1(id INT);\nSELECT id FROM t1;")
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	for i, ps := range parsed {
		omniAST, ok := ps.AST.(*OmniAST)
		require.True(t, ok, "stmt %d: AST must be *OmniAST", i)
		require.Equal(t, ps.Text, omniAST.Text, "stmt %d", i)

		// The legacy ANTLR tree must remain reachable through base.GetANTLRAST.
		antlrAST, ok := base.GetANTLRAST(ps.AST)
		require.True(t, ok, "stmt %d: GetANTLRAST must work", i)
		require.NotNil(t, antlrAST.Tree, "stmt %d", i)
		require.NotNil(t, antlrAST.Tokens, "stmt %d", i)

		// The omni node must be reachable through GetOmniNode.
		node, ok := GetOmniNode(ps.AST)
		require.True(t, ok, "stmt %d: GetOmniNode must work", i)
		require.NotNil(t, node, "stmt %d", i)

		// ASTStartPosition must stay identical to the legacy ANTLRAST position
		// (line-based, BaseLine()+1), so position reporting is unchanged by
		// the wrapper.
		require.Equal(t, antlrAST.StartPosition, ps.AST.ASTStartPosition(), "stmt %d", i)
		require.EqualValues(t, ps.BaseLine()+1, ps.AST.ASTStartPosition().GetLine(), "stmt %d", i)
	}

	node0, _ := GetOmniNode(parsed[0].AST)
	require.IsType(t, (*omniast.CreateTableStmt)(nil), node0)
	node1, _ := GetOmniNode(parsed[1].AST)
	require.IsType(t, (*omniast.SelectStmt)(nil), node1)
}

// TestParseSnowflakeStatementsOmniLeniency locks the transition-leniency
// contract: a statement the legacy ANTLR parser accepts but omni rejects must
// NOT fail the batch — it gets an OmniAST with a nil omni node (GetOmniNode
// false) while base.GetANTLRAST keeps working for the ANTLR advisors.
//
// "SELECT LEFT('COL1', 1)" is such a statement today (from the legacy
// grammar's own examples corpus): the legacy grammar parses LEFT(...) as a
// function call while omni rejects it ("syntax error at or near LEFT"). If a
// future omni version learns to parse it, swap in another legacy-only
// statement or drop down to constructing the OmniAST directly.
func TestParseSnowflakeStatementsOmniLeniency(t *testing.T) {
	parsed, err := parseSnowflakeStatements("SELECT 1;\nSELECT LEFT('COL1', 1);")
	require.NoError(t, err, "an omni-only parse failure must not fail the batch")
	require.Len(t, parsed, 2)

	// Statement 1: both trees present.
	node, ok := GetOmniNode(parsed[0].AST)
	require.True(t, ok)
	require.NotNil(t, node)
	_, ok = base.GetANTLRAST(parsed[0].AST)
	require.True(t, ok)

	// Statement 2: omni cannot parse it -> nil omni node, but the wrapper and
	// the legacy ANTLR tree are still attached.
	omniAST, ok := parsed[1].AST.(*OmniAST)
	require.True(t, ok, "AST must still be *OmniAST")
	require.Nil(t, omniAST.Node)
	_, ok = GetOmniNode(parsed[1].AST)
	require.False(t, ok, "GetOmniNode must report false for a nil omni node")
	antlrAST, ok := base.GetANTLRAST(parsed[1].AST)
	require.True(t, ok, "GetANTLRAST must still work")
	require.NotNil(t, antlrAST.Tree)
	require.Equal(t, antlrAST.StartPosition, parsed[1].AST.ASTStartPosition())
}

// TestParseSnowflakeStatementsEmptyAndError locks the surrounding behavior the
// dual-AST change must not disturb: empty/comment-only inputs produce no
// statements (omni-based SplitSQL drops empty segments), and a legacy ANTLR
// parse failure still fails the whole batch exactly as before.
func TestParseSnowflakeStatementsEmptyAndError(t *testing.T) {
	// Comments and bare semicolons yield no statements and no error.
	parsed, err := parseSnowflakeStatements(";; -- only a comment\n;")
	require.NoError(t, err)
	require.Empty(t, parsed)

	// Comment-only prefix does not shift the statement pairing.
	parsed, err = parseSnowflakeStatements("-- header\nSELECT 1;;SELECT 2;")
	require.NoError(t, err)
	require.Len(t, parsed, 2)
	for i, ps := range parsed {
		_, ok := GetOmniNode(ps.AST)
		require.True(t, ok, "stmt %d", i)
		_, ok = base.GetANTLRAST(ps.AST)
		require.True(t, ok, "stmt %d", i)
	}

	// Legacy parse failure keeps today's error behavior: the whole batch fails.
	_, err = parseSnowflakeStatements("SELECT 1;\nSELECT * FROM;")
	require.Error(t, err)
}

// TestGetOmniNodeEdgeCases covers the accessor's non-OmniAST inputs.
func TestGetOmniNodeEdgeCases(t *testing.T) {
	node, ok := GetOmniNode(nil)
	require.False(t, ok)
	require.Nil(t, node)

	// A plain legacy ANTLRAST is not an OmniAST.
	node, ok = GetOmniNode(&base.ANTLRAST{})
	require.False(t, ok)
	require.Nil(t, node)

	// And the reverse direction: AsANTLRAST on a wrapper without a legacy tree.
	antlrAST, ok := base.GetANTLRAST(&OmniAST{})
	require.False(t, ok)
	require.Nil(t, antlrAST)
}
