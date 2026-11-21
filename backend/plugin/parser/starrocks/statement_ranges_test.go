package starrocks

import (
	"context"
	"testing"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementRangesSeam(t *testing.T) {
	got, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_STARROCKS, "SELECT 1;\n  SELECT 2;")
	require.NoError(t, err)
	require.Equal(t, []base.Range{
		{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 0, Character: 9}},
		{Start: lsp.Position{Line: 1, Character: 2}, End: lsp.Position{Line: 1, Character: 11}},
	}, got)
}
