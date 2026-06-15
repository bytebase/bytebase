package snowflake

import (
	"testing"

	omniast "github.com/bytebase/omni/snowflake/ast"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestParseSnowflakeStatementsOmniAST locks the omni-only contract: every
// non-empty statement carries an OmniAST wrapping a non-nil omni node
// (reachable via GetOmniNode); the legacy ANTLR tree is gone, so
// base.GetANTLRAST reports false.
func TestParseSnowflakeStatementsOmniAST(t *testing.T) {
	parsed, err := parseSnowflakeStatements("CREATE TABLE t1(id INT);\nSELECT id FROM t1;")
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	for i, ps := range parsed {
		omniAST, ok := ps.AST.(*OmniAST)
		require.True(t, ok, "stmt %d: AST must be *OmniAST", i)
		require.Equal(t, ps.Text, omniAST.Text, "stmt %d", i)

		// The omni node must be reachable through GetOmniNode.
		node, ok := GetOmniNode(ps.AST)
		require.True(t, ok, "stmt %d: GetOmniNode must work", i)
		require.NotNil(t, node, "stmt %d", i)

		// No legacy ANTLR tree rides along anymore.
		_, ok = base.GetANTLRAST(ps.AST)
		require.False(t, ok, "stmt %d: GetANTLRAST must report false", i)

		// ASTStartPosition keeps the legacy shape (line-based, BaseLine()+1),
		// so position reporting is unchanged by the cutover.
		require.EqualValues(t, ps.BaseLine()+1, ps.AST.ASTStartPosition().GetLine(), "stmt %d", i)
	}

	node0, _ := GetOmniNode(parsed[0].AST)
	require.IsType(t, (*omniast.CreateTableStmt)(nil), node0)
	node1, _ := GetOmniNode(parsed[1].AST)
	require.IsType(t, (*omniast.SelectStmt)(nil), node1)
}

// TestParseSnowflakeStatementsEmptyAndError locks the surrounding behavior:
// empty/comment-only inputs produce no statements (omni-based SplitSQL drops
// empty segments), and an omni parse failure fails the whole batch exactly as
// the legacy ANTLR parser did.
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
	}

	// Parse failure keeps the legacy error behavior: the whole batch fails.
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
}
