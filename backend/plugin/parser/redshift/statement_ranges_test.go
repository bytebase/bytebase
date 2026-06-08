package redshift

import (
	"context"
	"testing"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementRanges(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.Range
	}{
		{
			name:      "multi statement",
			statement: "SELECT 1;\n  SELECT 2;",
			want: []base.Range{
				{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 9},
				},
				{
					Start: lsp.Position{Line: 1, Character: 2},
					End:   lsp.Position{Line: 1, Character: 11},
				},
			},
		},
		{
			name:      "utf16 comment prefix",
			statement: "/*🙂*/ SELECT 1;",
			want: []base.Range{
				{
					Start: lsp.Position{Line: 0, Character: 7},
					End:   lsp.Position{Line: 0, Character: 16},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetStatementRanges(context.Background(), base.StatementRangeContext{}, tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetStatementRangesReturnsParseError(t *testing.T) {
	_, err := GetStatementRanges(context.Background(), base.StatementRangeContext{}, "SELECT * FROM")
	require.Error(t, err)
}
