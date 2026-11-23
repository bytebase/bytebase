package elasticsearch

import (
	"context"
	"testing"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementRangesSeam(t *testing.T) {
	ranges, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_ELASTICSEARCH, "GET /users/_search\n{}\n\nGET /orders/_doc/1")
	require.NoError(t, err)
	require.Equal(t, []base.Range{
		{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 1, Character: 2},
		},
		{
			Start: lsp.Position{Line: 3, Character: 0},
			End:   lsp.Position{Line: 3, Character: 18},
		},
	}, ranges)

	empty, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_ELASTICSEARCH, "")
	require.NoError(t, err)
	require.Empty(t, empty)
}
