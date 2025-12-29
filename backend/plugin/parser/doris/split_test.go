package doris

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitDorisSQLStatements(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.Statement
	}{
		{
			name:      "simple single statement",
			statement: "SELECT 1;",
			want: []base.Statement{
				{
					Text:     "SELECT 1;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 1, Column: 10},
					Range:    &storepb.Range{Start: 0, End: 9},
				},
			},
		},
		{
			name:      "multiple statements on one line",
			statement: "SELECT 1; SELECT 2;",
			want: []base.Statement{
				{
					Text:     "SELECT 1;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 1, Column: 10},
					Range:    &storepb.Range{Start: 0, End: 9},
				},
				{
					Text:     " SELECT 2;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 11},
					End:      &storepb.Position{Line: 1, Column: 20},
					Range:    &storepb.Range{Start: 9, End: 19},
				},
			},
		},
		{
			name:      "multi-line statement",
			statement: "SELECT\n  1;",
			want: []base.Statement{
				{
					Text:     "SELECT\n  1;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 2, Column: 5},
					Range:    &storepb.Range{Start: 0, End: 11},
				},
			},
		},
		{
			name:      "multiple statements on multiple lines",
			statement: "SELECT 1;\nSELECT 2;",
			want: []base.Statement{
				{
					Text:     "SELECT 1;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 1, Column: 10},
					Range:    &storepb.Range{Start: 0, End: 9},
				},
				{
					Text:     "\nSELECT 2;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 2, Column: 1},
					End:      &storepb.Position{Line: 2, Column: 10},
					Range:    &storepb.Range{Start: 9, End: 19},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			list, err := SplitSQL(tc.statement)
			require.NoError(t, err)
			require.Equal(t, len(tc.want), len(list), "statement count mismatch")

			for i, want := range tc.want {
				got := list[i]
				require.Equal(t, want.Text, got.Text, "Text mismatch at index %d", i)
				require.Equal(t, want.BaseLine, got.BaseLine, "BaseLine mismatch at index %d", i)
				require.NotNil(t, got.Start, "Start should not be nil at index %d", i)
				require.Equal(t, want.Start.Line, got.Start.Line, "Start.Line mismatch at index %d", i)
				require.Equal(t, want.Start.Column, got.Start.Column, "Start.Column mismatch at index %d", i)
				require.NotNil(t, got.End, "End should not be nil at index %d", i)
				require.Equal(t, want.End.Line, got.End.Line, "End.Line mismatch at index %d", i)
				require.Equal(t, want.End.Column, got.End.Column, "End.Column mismatch at index %d", i)
				if want.Range != nil {
					require.Equal(t, want.Range.Start, got.Range.Start, "Range.Start mismatch at index %d", i)
					require.Equal(t, want.Range.End, got.Range.End, "Range.End mismatch at index %d", i)
				}
			}
		})
	}
}
