package snowflake

import (
	"testing"

	omniast "github.com/bytebase/omni/snowflake/ast"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type testAST struct{}

func (testAST) ASTStartPosition() *storepb.Position {
	return nil
}

// TestParseSnowflakeStatementsAST locks the omni-only contract: every
// non-empty statement carries an AST wrapping a non-nil omni node
// (reachable via GetNode); the legacy ANTLR tree is gone.
func TestParseSnowflakeStatementsAST(t *testing.T) {
	parsed, err := parseSnowflakeStatements("CREATE TABLE t1(id INT);\nSELECT id FROM t1;")
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	for i, ps := range parsed {
		wrapper, ok := ps.AST.(*AST)
		require.True(t, ok, "stmt %d: AST must be *AST", i)
		require.Equal(t, ps.Text, wrapper.Text, "stmt %d", i)

		// The omni node must be reachable through GetNode.
		node, ok := GetNode(ps.AST)
		require.True(t, ok, "stmt %d: GetNode must work", i)
		require.NotNil(t, node, "stmt %d", i)

		// ASTStartPosition keeps the legacy shape (line-based, BaseLine()+1),
		// so position reporting is unchanged by the cutover.
		require.EqualValues(t, ps.BaseLine()+1, ps.AST.ASTStartPosition().GetLine(), "stmt %d", i)
	}

	node0, _ := GetNode(parsed[0].AST)
	require.IsType(t, (*omniast.CreateTableStmt)(nil), node0)
	node1, _ := GetNode(parsed[1].AST)
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
		_, ok := GetNode(ps.AST)
		require.True(t, ok, "stmt %d", i)
	}

	// Parse failure keeps the legacy error behavior: the whole batch fails.
	_, err = parseSnowflakeStatements("SELECT 1;\nSELECT * FROM;")
	require.Error(t, err)
}

// TestGetNodeEdgeCases covers the accessor's non-AST inputs.
func TestGetNodeEdgeCases(t *testing.T) {
	node, ok := GetNode(nil)
	require.False(t, ok)
	require.Nil(t, node)

	node, ok = GetNode(testAST{})
	require.False(t, ok)
	require.Nil(t, node)
}
