//nolint:revive
package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestConvertANTLRPositionToPosition(t *testing.T) {
	testCases := []struct {
		description   string
		text          string
		antlrPosition *ANTLRPosition
		want          *storepb.Position
	}{
		{
			description:   "first line first column",
			text:          "hello, world",
			antlrPosition: &ANTLRPosition{Line: 1, Column: 0},
			want:          &storepb.Position{Line: 1, Column: 1},
		},
		{
			description:   "ASCII text",
			text:          "hello, world",
			antlrPosition: &ANTLRPosition{Line: 1, Column: 6},
			want:          &storepb.Position{Line: 1, Column: 7}, // ANTLR column 6 (0-based) -> column 7 (1-based)
		},
		{
			description:   "multi-byte characters",
			text:          "你好\n世界",
			antlrPosition: &ANTLRPosition{Line: 2, Column: 1},
			want:          &storepb.Position{Line: 2, Column: 2}, // Character-based, not byte-based
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			got := ConvertANTLRPositionToPosition(tc.antlrPosition, tc.text)
			require.Equal(t, tc.want, got)
		})
	}
}
