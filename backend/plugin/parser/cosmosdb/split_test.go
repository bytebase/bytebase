package cosmosdb

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
		want      *base.Statement // nil means no statements expected
	}{
		{
			name:      "Single SELECT statement",
			statement: "SELECT * FROM c",
			want: &base.Statement{
				Text:     "SELECT * FROM c",
				BaseLine: 0,
				Start:    &storepb.Position{Line: 1, Column: 1},
				End:      &storepb.Position{Line: 1, Column: 16}, // 1-based exclusive (after 'c')
				Range:    &storepb.Range{Start: 0, End: 15},
				Empty:    false,
			},
		},
		{
			name:      "SELECT with WHERE",
			statement: "SELECT c.id FROM users c WHERE c.active = true",
			want: &base.Statement{
				Text:     "SELECT c.id FROM users c WHERE c.active = true",
				BaseLine: 0,
				Start:    &storepb.Position{Line: 1, Column: 1},
				End:      &storepb.Position{Line: 1, Column: 47}, // 1-based exclusive (after 'e')
				Range:    &storepb.Range{Start: 0, End: 46},
				Empty:    false,
			},
		},
		{
			name:      "Multi-line SELECT",
			statement: "SELECT\n  * FROM c",
			want: &base.Statement{
				Text:     "SELECT\n  * FROM c",
				BaseLine: 0,
				Start:    &storepb.Position{Line: 1, Column: 1},
				End:      &storepb.Position{Line: 2, Column: 11}, // 1-based exclusive (after 'c')
				Range:    &storepb.Range{Start: 0, End: 17},
				Empty:    false,
			},
		},
		{
			name:      "Empty string",
			statement: "",
			want:      nil,
		},
		{
			name:      "Only whitespace",
			statement: "   \n  \t  ",
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := SplitSQL(tt.statement)
			require.NoError(t, err)

			if tt.want == nil {
				require.Empty(t, list)
				return
			}

			require.Len(t, list, 1)
			got := list[0]

			require.Equal(t, tt.want.Text, got.Text, "Text mismatch")
			require.Equal(t, tt.want.BaseLine, got.BaseLine, "BaseLine mismatch")
			require.Equal(t, tt.want.Empty, got.Empty, "Empty mismatch")

			require.NotNil(t, got.Start, "Start should not be nil")
			require.Equal(t, tt.want.Start.Line, got.Start.Line, "Start.Line mismatch")
			require.Equal(t, tt.want.Start.Column, got.Start.Column, "Start.Column mismatch")

			require.NotNil(t, got.End, "End should not be nil")
			require.Equal(t, tt.want.End.Line, got.End.Line, "End.Line mismatch")
			require.Equal(t, tt.want.End.Column, got.End.Column, "End.Column mismatch")

			require.NotNil(t, got.Range, "Range should not be nil")
			require.Equal(t, tt.want.Range.Start, got.Range.Start, "Range.Start mismatch")
			require.Equal(t, tt.want.Range.End, got.Range.End, "Range.End mismatch")
		})
	}
}
