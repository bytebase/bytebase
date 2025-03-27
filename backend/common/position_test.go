package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPositionToANTLRPosition(t *testing.T) {
	testCases := []struct {
		description string
		text        string
		position    Position
		want        ANTLRPosition
	}{
		{
			description: "empty text",
			text:        "",
			position:    Position{Line: 0, Column: 0},
			want:        ANTLRPosition{Line: 1, Column: 0},
		},
		{
			description: "ascii",
			text:        "hello, world",
			position:    Position{Line: 0, Column: 6},
			want:        ANTLRPosition{Line: 1, Column: 6},
		},
		{
			description: "multi-bytes characters",
			text:        "你好\n世界",
			position:    Position{Line: 1, Column: 3},
			want:        ANTLRPosition{Line: 2, Column: 1},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := tc.position.ToANTLRPosition(tc.text)
		a.Equalf(tc.want, got, "Test case: %s", tc.description)
	}
}

func TestANTLRPositionToPosition(t *testing.T) {
	testCases := []struct {
		description   string
		text          string
		antlrPosition ANTLRPosition
		want          Position
	}{
		{
			description:   "empty text",
			text:          "",
			antlrPosition: ANTLRPosition{Line: 1, Column: 0},
			want:          Position{Line: 0, Column: 0},
		},
		{
			description:   "ascii",
			text:          "hello, world",
			antlrPosition: ANTLRPosition{Line: 1, Column: 6},
			want:          Position{Line: 0, Column: 6},
		},
		{
			description:   "multi-bytes characters",
			text:          "你好\n世界",
			antlrPosition: ANTLRPosition{Line: 2, Column: 1},
			want:          Position{Line: 1, Column: 3},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := tc.antlrPosition.ToPosition(tc.text)
		a.Equalf(tc.want, got, "Test case: %s", tc.description)
	}
}
