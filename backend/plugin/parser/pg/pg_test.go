package pg

import (
	"context"
	"errors"
	"testing"

	"github.com/bytebase/omni/pg/analysis"
	"github.com/bytebase/omni/pg/ast"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseStatementsSeam(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, "SELECT 1;\nSELECT 2;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	node, ok := GetNode(stmts[0].AST)
	require.True(t, ok)
	require.IsType(t, &ast.SelectStmt{}, node)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].AST.ASTStartPosition())

	_, ok = GetNode(nil)
	require.False(t, ok)

	// Syntax error converts omni byte offset into 1-based line and rune column.
	_, err = base.ParseStatements(storepb.Engine_POSTGRES, "SELECT 1;\nSELECT '你好' FROM ;")
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, int32(2), syntaxErr.Position.Line)
	require.Equal(t, int32(18), syntaxErr.Position.Column)
	require.Contains(t, syntaxErr.Message, "Syntax error at line 2:")
	require.Equal(t, `syntax error at or near ";"`, syntaxErr.RawMessage)
}

func TestDiagnoseSeam(t *testing.T) {
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB} {
		t.Run(engine.String(), func(t *testing.T) {
			diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, engine, "SELECT 1;")
			require.NoError(t, err)
			require.Empty(t, diags)

			diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, engine, "SELECT 1;\nSELECT * FROM")
			require.NoError(t, err)
			require.Len(t, diags, 1)
			require.Equal(t, uint32(1), diags[0].Range.Start.Line)
		})
	}

	// Non-parse errors fall back to line 1, column 1.
	syntaxErr := convertDiagnoseError(&analysis.DiagnoseError{Message: "unexpected", ParseError: false})
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, syntaxErr.Position)
	require.Nil(t, convertDiagnoseError(nil))
	require.False(t, errors.Is(errors.New("other"), base.ErrExplainAnalyzeExecutes))
}

func TestGetStatementRangesSeam(t *testing.T) {
	ranges, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_POSTGRES, "SELECT 1;\nSELECT 2")
	require.NoError(t, err)
	require.Len(t, ranges, 2)
	require.Equal(t, uint32(0), ranges[0].Start.Line)
	require.Equal(t, uint32(1), ranges[1].Start.Line)
}
