package lsp

import (
	"testing"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"
)

func TestOffsetForPosition(t *testing.T) {
	testCases := []struct {
		content  []byte
		position lsp.Position
		expected int
		valid    bool
	}{
		{
			content:  []byte("Hello, World!"),
			position: lsp.Position{Line: 0, Character: 0},
			expected: 0,
			valid:    true,
		},
		{
			content:  []byte("Hello, World!"),
			position: lsp.Position{Line: 0, Character: 7},
			expected: 7,
			valid:    true,
		},
		{
			content:  []byte("Hello, 世界!"),
			position: lsp.Position{Line: 0, Character: 7}, // Before '世'
			expected: 7,
			valid:    true,
		},
		{
			content:  []byte("Hello, 世界!"),
			position: lsp.Position{Line: 0, Character: 8}, // After '世'
			expected: 10,
			valid:    true,
		},
		{
			content:  []byte("Hello,\nWorld!"),
			position: lsp.Position{Line: 1, Character: 0}, // Start of line 2
			expected: 7,
			valid:    true,
		},
		{
			content:  []byte("Hello,\nWorld!"),
			position: lsp.Position{Line: 1, Character: 5}, // 'World'
			expected: 12,
			valid:    true,
		},
		{
			content:  []byte("Hello,\nWorld!"),
			position: lsp.Position{Line: 1, Character: 10}, // Beyond line boundary
			valid:    false,
		},
		{
			content:  []byte("Hello, 𐍈!"), // '𐍈' is a Unicode character requiring surrogate pairs in UTF-16
			position: lsp.Position{Line: 0, Character: 7},
			expected: 7,
			valid:    true,
		},
		{
			content:  []byte("Hello, 𐍈!"),
			position: lsp.Position{Line: 0, Character: 9}, // After surrogate pairs in UTF-16
			expected: 11,
			valid:    true,
		},
		{
			content:  []byte("Hello,\nWorld!"),
			position: lsp.Position{Line: 2, Character: 0}, // Beyond last line
			valid:    false,
		},
	}

	for idx, tc := range testCases {
		offset, err := offsetForPosition(tc.content, tc.position)
		if !tc.valid {
			require.NotNil(t, err)
			continue
		}

		require.Nil(t, err)
		require.Equal(t, tc.expected, offset, "test cases %d", idx)
	}
}

func TestGetSQLStatementRangesUTF16Position(t *testing.T) {
	testCases := []struct {
		content []byte
		ranges  []lsp.Range
	}{
		{
			content: []byte("SELECT 1;\nSELECT 2"),
			ranges: []lsp.Range{
				{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 1, Character: 0}},
				{Start: lsp.Position{Line: 1, Character: 0}, End: lsp.Position{Line: 2, Character: 0}},
			},
		},
		{
			content: []byte(`SELECT 1;



SELECT * FROM public.workday;`),
			ranges: []lsp.Range{
				{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 1, Character: 0}},
				{Start: lsp.Position{Line: 4, Character: 0}, End: lsp.Position{Line: 4, Character: 30}},
			},
		},
	}

	for idx, tc := range testCases {
		ranges := getSQLStatementRangesUTF16Position(tc.content)
		require.Equal(t, tc.ranges, ranges, "test cases %d", idx)
	}
}
