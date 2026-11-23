package snowflake

import (
	"context"
	"testing"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementRangesSeam(t *testing.T) {
	got, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_SNOWFLAKE, "SELECT 1; SELECT 2;")
	require.NoError(t, err)
	require.Equal(t, []base.Range{
		{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 0, Character: 9}},
		{Start: lsp.Position{Line: 0, Character: 10}, End: lsp.Position{Line: 0, Character: 19}},
	}, got)

	// UTF-16 surrogate pair column conversion in utf16Mapper.
	got, err = GetStatementRanges(context.Background(), base.StatementRangeContext{}, "/*🙂*/ SELECT 1;")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, lsp.Position{Line: 0, Character: 0}, got[0].Start)
	require.Equal(t, lsp.Position{Line: 0, Character: 16}, got[0].End)

	got, err = GetStatementRanges(context.Background(), base.StatementRangeContext{}, "")
	require.NoError(t, err)
	require.Nil(t, got)
}
