package base

import (
	"testing"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestConvertPositionToUTF16Position(t *testing.T) {
	testCases := []struct {
		description string
		text        string
		position    *storepb.Position
		want        *lsp.Position
	}{
		{
			description: "empty text",
			text:        "",
			position:    &storepb.Position{Line: 0, Column: 0},
			want:        &lsp.Position{Line: 0, Character: 0},
		},
		{
			description: "ascii",
			text:        "hello, world",
			position:    &storepb.Position{Line: 0, Column: 6},
			want:        &lsp.Position{Line: 0, Character: 6},
		},
		{
			description: "surrogate pairs",
			// 𝄞 encoded in utf16 with 2 code units, and 4 code units in utf8.
			text:     "abc𝄞def",
			position: &storepb.Position{Line: 0, Column: 7},
			want: &lsp.Position{
				Line:      0,
				Character: 5,
			},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := convertPositionToUTF16Position(tc.position, tc.text)
		a.Equalf(tc.want, got, "Test case: %s", tc.description)
	}
}
