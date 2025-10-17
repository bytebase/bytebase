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
			description:   "empty text",
			text:          "",
			antlrPosition: &ANTLRPosition{Line: 1, Column: 0},
			want:          &storepb.Position{Line: 0, Column: 0},
		},
		{
			description:   "ascii",
			text:          "hello, world",
			antlrPosition: &ANTLRPosition{Line: 1, Column: 6},
			want:          &storepb.Position{Line: 0, Column: 6},
		},
		{
			description:   "multi-bytes characters",
			text:          "你好\n世界",
			antlrPosition: &ANTLRPosition{Line: 2, Column: 1},
			want:          &storepb.Position{Line: 1, Column: 3},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := ConvertANTLRPositionToPosition(tc.antlrPosition, tc.text)
		a.Equalf(tc.want, got, "Test case: %s", tc.description)
	}
}
