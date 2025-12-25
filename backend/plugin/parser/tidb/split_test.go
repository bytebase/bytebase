package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQL(t *testing.T) {
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
					End:      &storepb.Position{Line: 1},
					Range:    &storepb.Range{Start: 0, End: 9},
					Empty:    false,
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
					End:      &storepb.Position{Line: 2},
					Range:    &storepb.Range{Start: 0, End: 11},
					Empty:    false,
				},
			},
		},
		{
			name:      "multiple statements",
			statement: "SELECT 1;\nSELECT 2;",
			want: []base.Statement{
				{
					Text:     "SELECT 1;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 1},
					Range:    &storepb.Range{Start: 0, End: 9},
					Empty:    false,
				},
				{
					Text:     "SELECT 2;",
					BaseLine: 1,
					Start:    &storepb.Position{Line: 2, Column: 1},
					End:      &storepb.Position{Line: 2},
					Range:    &storepb.Range{Start: 10, End: 19},
					Empty:    false,
				},
			},
		},
		{
			name:      "multi-byte characters - Chinese",
			statement: "SELECT '中文';",
			want: []base.Statement{
				{
					Text:     "SELECT '中文';",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					// Column is 1-based character offset: S(1) E(2) L(3) E(4) C(5) T(6) ' '(7) '(8) 中(9) 文(10) '(11) ;(12) = 12 chars
					End:   &storepb.Position{Line: 1},
					Range: &storepb.Range{Start: 0, End: 16}, // byte length: 8 + 3 + 3 + 2 = 16
					Empty: false,
				},
			},
		},
		{
			name:      "multi-byte characters - emoji",
			statement: "SELECT '🎉';",
			want: []base.Statement{
				{
					Text:     "SELECT '🎉';",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					// Column is 1-based character offset: S(1) E(2) L(3) E(4) C(5) T(6) ' '(7) '(8) 🎉(9) '(10) ;(11) = 11 chars
					End:   &storepb.Position{Line: 1},
					Range: &storepb.Range{Start: 0, End: 14}, // byte length: 8 + 4 + 2 = 14
					Empty: false,
				},
			},
		},
		{
			name:      "multi-byte on second line",
			statement: "SELECT\n  '中文';",
			want: []base.Statement{
				{
					Text:     "SELECT\n  '中文';",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 2},
					Range:    &storepb.Range{Start: 0, End: 18}, // 7 + 3 + 3 + 3 + 2 = 18
					Empty:    false,
				},
			},
		},
		{
			name:      "multiple statements with multi-byte",
			statement: "SELECT '中';\nSELECT '文';",
			want: []base.Statement{
				{
					Text:     "SELECT '中';",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 1},
					Range:    &storepb.Range{Start: 0, End: 13}, // 8 + 3 + 2 = 13
					Empty:    false,
				},
				{
					Text:     "SELECT '文';",
					BaseLine: 1,
					Start:    &storepb.Position{Line: 2, Column: 1},
					End:      &storepb.Position{Line: 2},
					Range:    &storepb.Range{Start: 14, End: 27},
					Empty:    false,
				},
			},
		},
		{
			name:      "statement with leading spaces and multi-byte",
			statement: "  SELECT '中';",
			want: []base.Statement{
				{
					Text:     "SELECT '中';",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 3}, // starts after 2 spaces (1-based: column 3)
					End:      &storepb.Position{Line: 1},
					Range:    &storepb.Range{Start: 2, End: 15}, // 2 + 8 + 3 + 2 = 15
					Empty:    false,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SplitSQL(tc.statement)
			require.NoError(t, err)
			require.Equal(t, len(tc.want), len(got), "number of statements mismatch")
			for i, want := range tc.want {
				require.Equal(t, want.Text, got[i].Text, "Text mismatch at index %d", i)
				require.Equal(t, want.BaseLine, got[i].BaseLine, "BaseLine mismatch at index %d", i)
				require.Equal(t, want.Start, got[i].Start, "Start mismatch at index %d", i)
				require.Equal(t, want.End, got[i].End, "End mismatch at index %d", i)
				require.Equal(t, want.Range, got[i].Range, "Range mismatch at index %d", i)
				require.Equal(t, want.Empty, got[i].Empty, "Empty mismatch at index %d", i)
			}
		})
	}
}
