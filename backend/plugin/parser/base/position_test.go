package base

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestByteOffsetPositionMapper(t *testing.T) {
	sql := "SELECT 1;\nSELECT 日本;\nSELECT 3;"
	mapper := NewByteOffsetPositionMapper(sql)

	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, mapper.Position(0))
	require.Equal(t, &storepb.Position{Line: 2, Column: 1}, mapper.Position(len("SELECT 1;\n")))
	require.Equal(t, &storepb.Position{Line: 2, Column: 9}, mapper.Position(len("SELECT 1;\nSELECT 日")))

	// Out-of-order lookups are supported for callers that occasionally need to
	// revisit an earlier offset.
	require.Equal(t, &storepb.Position{Line: 1, Column: 7}, mapper.Position(len("SELECT")))
}

func TestByteOffsetPositionMapperLineBreaks(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want *storepb.Position
	}{
		{
			name: "lf",
			sql:  "SELECT 1;\nSELECT 日本;",
			want: &storepb.Position{Line: 2, Column: 11},
		},
		{
			name: "crlf",
			sql:  "SELECT 1;\r\nSELECT 日本;",
			want: &storepb.Position{Line: 2, Column: 11},
		},
		{
			name: "cr",
			sql:  "SELECT 1;\rSELECT 日本;",
			want: &storepb.Position{Line: 2, Column: 11},
		},
		{
			name: "mixed",
			sql:  "SELECT 1;\r\nSELECT 日本;\rSELECT 3;\nSELECT 4;",
			want: &storepb.Position{Line: 4, Column: 10},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapper := NewByteOffsetPositionMapper(test.sql)
			for offset := range test.sql {
				require.Equal(t, ByteOffsetToRunePosition(test.sql, offset), mapper.Position(offset), "offset %d", offset)
			}
			require.Equal(t, test.want, mapper.Position(len(test.sql)))
			require.Equal(t, test.want, ByteOffsetToRunePosition(test.sql, len(test.sql)))
		})
	}
}

func TestByteOffsetToRunePositionEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		byteOffset int
		want       *storepb.Position
	}{
		{name: "empty", sql: "", byteOffset: 0, want: &storepb.Position{Line: 1, Column: 1}},
		{name: "ascii start", sql: "SELECT 1", byteOffset: 0, want: &storepb.Position{Line: 1, Column: 1}},
		{name: "ascii mid", sql: "SELECT 1", byteOffset: 7, want: &storepb.Position{Line: 1, Column: 8}},
		{name: "ascii end", sql: "SELECT 1", byteOffset: 8, want: &storepb.Position{Line: 1, Column: 9}},
		{name: "bmp 3-byte rune", sql: "SELECT '中' AS x", byteOffset: 11, want: &storepb.Position{Line: 1, Column: 10}},
		{name: "emoji 4-byte rune", sql: "SELECT '😀' x", byteOffset: 12, want: &storepb.Position{Line: 1, Column: 10}},
		{name: "mid-rune bmp", sql: "SELECT '中'", byteOffset: 9, want: &storepb.Position{Line: 1, Column: 9}},
		{name: "mid-rune emoji", sql: "a😀b", byteOffset: 3, want: &storepb.Position{Line: 1, Column: 2}},
		{name: "at newline", sql: "a\nb", byteOffset: 1, want: &storepb.Position{Line: 1, Column: 2}},
		{name: "multi-byte across newline", sql: "中\n中", byteOffset: 4, want: &storepb.Position{Line: 2, Column: 1}},
		{name: "past end clamped", sql: "abc", byteOffset: 100, want: &storepb.Position{Line: 1, Column: 4}},
		{name: "negative clamped", sql: "abc", byteOffset: -5, want: &storepb.Position{Line: 1, Column: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ByteOffsetToRunePosition(tt.sql, tt.byteOffset))
		})
	}
}

func TestLinePosition(t *testing.T) {
	require.Equal(t, &storepb.Position{Line: 1}, LinePosition(-3))
	require.Equal(t, &storepb.Position{Line: 1}, LinePosition(0))
	require.Equal(t, &storepb.Position{Line: 7}, LinePosition(7))
}
