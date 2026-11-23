package tidb

import (
	"testing"

	omniparser "github.com/bytebase/omni/tidb/parser"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// stubAST is a base.AST that is not an *AST, for the GetNode negative case.
type stubAST struct{}

func (stubAST) ASTStartPosition() *storepb.Position { return nil }

func TestGetNode(t *testing.T) {
	a := require.New(t)

	// nil base.AST returns (nil, false).
	node, ok := GetNode(nil)
	a.Nil(node)
	a.False(ok)

	// A base.AST that is not an *AST returns (nil, false).
	node, ok = GetNode(stubAST{})
	a.Nil(node)
	a.False(ok)

	// AST with non-nil node round-trips.
	list, err := omniparser.Parse("SELECT 1")
	a.NoError(err)
	a.NotNil(list)
	a.NotEmpty(list.Items)
	wrapped := &AST{
		Node:          list.Items[0],
		Text:          "SELECT 1",
		StartPosition: &storepb.Position{Line: 1, Column: 1},
	}
	node, ok = GetNode(wrapped)
	a.True(ok)
	a.Equal(list.Items[0], node)
}

func TestASTStartPosition(t *testing.T) {
	a := require.New(t)
	pos := &storepb.Position{Line: 5, Column: 7}
	o := &AST{StartPosition: pos}
	a.Equal(pos, o.ASTStartPosition())
}

// TestParseTiDBStatements_AllAccepted pins the happy path for the registered
// TiDB ParseStatementsFunc: omni accepts every statement, so every result
// entry is an *AST and no error is returned.
func TestParseTiDBStatements_AllAccepted(t *testing.T) {
	const input = "CREATE TABLE t (id INT); INSERT INTO t VALUES (1);"

	result, err := parseTiDBStatements(input)
	require.NoError(t, err)
	require.Len(t, result, 2)
	for _, ps := range result {
		require.IsType(t, &AST{}, ps.AST,
			"happy path: every statement must be *AST")
	}
}

// TestParseTiDBStatements_RejectSurfacesError pins that omni is the sole
// TiDB parser: a statement omni rejects surfaces as a *base.SyntaxError with
// no pingcap fallback. The COLUMNAR INDEX and TRAFFIC CAPTURE cases are TiDB
// grammar newer than the pinned v8.5.5 that pingcap-master still accepts —
// the accepted regression from retiring the omni/pingcap dispatcher. The last
// case is a genuine syntax error. Error strings are verified against the live
// omni pin (they differ per case, so the assertion checks the converted type
// and a populated position, not a shared substring).
func TestParseTiDBStatements_RejectSurfacesError(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"traffic capture, newer than v8.5.5", "TRAFFIC CAPTURE TO 's3://x' DURATION = '1h';"},
		{"create columnar index, newer than v8.5.5", "CREATE COLUMNAR INDEX idx ON t (id);"},
		{"alter add columnar index, newer than v8.5.5", "ALTER TABLE t ADD COLUMNAR INDEX idx (id);"},
		{"genuine syntax error", "SELECT FROM WHERE;"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseTiDBStatements(tc.input)
			require.Error(t, err,
				"omni rejection must surface as an error; there is no pingcap fallback")
			syntaxErr, ok := err.(*base.SyntaxError)
			require.True(t, ok,
				"omni parse error must convert to *base.SyntaxError; got %T", err)
			require.NotEmpty(t, syntaxErr.RawMessage,
				"converted error must carry omni's parser complaint")
			require.NotNil(t, syntaxErr.Position,
				"converted error must carry a position in the original input")
		})
	}
}
