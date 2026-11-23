package tsql

import (
	"context"
	"testing"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestDiagnoseSeam(t *testing.T) {
	diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_MSSQL, "SELECT 1;")
	require.NoError(t, err)
	require.Empty(t, diags)

	// Multi-line + non-ASCII rune offset maps to 0-based LSP line/character.
	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_MSSQL, "SELECT 1;\nSELECT '你好' FROM ;")
	require.NoError(t, err)
	require.Len(t, diags, 1)
	require.Equal(t, lsp.SeverityError, diags[0].Severity)
	require.Equal(t, uint32(1), diags[0].Range.Start.Line)
	require.Equal(t, uint32(17), diags[0].Range.Start.Character)

	fallback := base.SyntaxErrorAtOffset("", -1, "unexpected")
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, fallback.Position)
	require.Equal(t, "unexpected", fallback.Message)
}
