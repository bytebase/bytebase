package cassandra

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseCassandraStatementsSeam(t *testing.T) {
	// Registry dispatch, AST wrapping, GetNode, and multi-line start positions.
	stmts, err := base.ParseStatements(storepb.Engine_CASSANDRA, "SELECT * FROM users;\nSELECT * FROM orders;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, int32(1), stmts[0].Start.Line)
	require.Equal(t, int32(2), stmts[1].Start.Line)
	node, ok := GetNode(stmts[0].AST)
	require.True(t, ok)
	require.NotNil(t, node)

	// Empty input produces no statements.
	stmts, err = parseCassandraStatements("")
	require.NoError(t, err)
	require.Empty(t, stmts)

	// Syntax error position adjustment across lines and leading indentation.
	_, err = parseCassandraStatements("SELECT * FROM t;\n  SELCT * FROM users")
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, int32(2), syntaxErr.Position.Line)
	require.Equal(t, int32(3), syntaxErr.Position.Column)
}

func TestCassandraValidateQuerySeam(t *testing.T) {
	require.True(t, base.HasQueryValidator(storepb.Engine_CASSANDRA))

	readOnly, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_CASSANDRA, "SELECT * FROM users")
	require.NoError(t, err)
	require.True(t, readOnly)
	require.True(t, allQuery)

	readOnly, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_CASSANDRA, "INSERT INTO users (id) VALUES (1)")
	require.NoError(t, err)
	require.False(t, readOnly)
	require.False(t, allQuery)

	readOnly, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_CASSANDRA, "SELECT FROM")
	require.Error(t, err)
	require.False(t, readOnly)
	require.False(t, allQuery)
}
