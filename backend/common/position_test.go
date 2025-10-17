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

func TestConvertPGParserErrorCursorPosToPosition(t *testing.T) {
	testCases := []struct {
		name           string
		cursorPos      int
		text           string
		expectedLine   int32
		expectedColumn int32
	}{
		{
			name:           "ASCII single line",
			cursorPos:      8,
			text:           "SELECT * FROM t",
			expectedLine:   1,
			expectedColumn: 8,
		},
		{
			name:           "multi-byte characters",
			cursorPos:      8,
			text:           "SELECT 你好 FROM t",
			expectedLine:   1,
			expectedColumn: 8, // Character-based (8th character is '你')
		},
		{
			name:           "multi-line",
			cursorPos:      20,
			text:           "SELECT * FROM t\nWHERE x = 1",
			expectedLine:   2,
			expectedColumn: 4, // 4th character on line 2
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pos := ConvertPGParserErrorCursorPosToPosition(tc.cursorPos, tc.text)
			require.Equal(t, tc.expectedLine, pos.Line, "line mismatch")
			require.Equal(t, tc.expectedColumn, pos.Column, "column mismatch")
		})
	}
}

func TestConvertTiDBParserErrorPositionToPosition(t *testing.T) {
	testCases := []struct {
		name         string
		line         int
		column       int
		expectedLine int32
		expectedCol  int32
	}{
		{
			name:         "normal position",
			line:         2,
			column:       10,
			expectedLine: 2,
			expectedCol:  10,
		},
		{
			name:         "line less than 1",
			line:         0,
			column:       5,
			expectedLine: 1,
			expectedCol:  5,
		},
		{
			name:         "column less than 1",
			line:         3,
			column:       0,
			expectedLine: 3,
			expectedCol:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pos := ConvertTiDBParserErrorPositionToPosition(tc.line, tc.column)
			require.Equal(t, tc.expectedLine, pos.Line, "line mismatch")
			require.Equal(t, tc.expectedCol, pos.Column, "column mismatch")
		})
	}
}
