package tidb

import (
	"context"
	"testing"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementRangesSeam(t *testing.T) {
	// Registry dispatch and 0-based line / UTF-16 character conversion (emoji counts as 2 UTF-16 code units).
	ranges, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_TIDB, "SELECT 1;\n  SELECT '🎉';")
	require.NoError(t, err)
	require.Equal(t, []base.Range{
		{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 0, Character: 9}},
		{Start: lsp.Position{Line: 1, Character: 2}, End: lsp.Position{Line: 1, Character: 14}},
	}, ranges)

	ranges, err = base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_TIDB, "   -- comment\n")
	require.NoError(t, err)
	require.Empty(t, ranges)
}
