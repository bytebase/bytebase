package base

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestByteOffsetPositionMapper(t *testing.T) {
	sql := "SELECT 1;\nSELECT 日本;\nSELECT 3;"
	mapper := NewByteOffsetPositionMapper(sql)

	require.Equal(t, int32(1), mapper.Position(0).Line)
	require.Equal(t, int32(1), mapper.Position(0).Column)
	require.Equal(t, int32(2), mapper.Position(len("SELECT 1;\n")).Line)
	require.Equal(t, int32(1), mapper.Position(len("SELECT 1;\n")).Column)
	require.Equal(t, int32(2), mapper.Position(len("SELECT 1;\nSELECT 日")).Line)
	require.Equal(t, int32(9), mapper.Position(len("SELECT 1;\nSELECT 日")).Column)

	// Out-of-order lookups are supported for callers that occasionally need to
	// revisit an earlier offset.
	require.Equal(t, int32(1), mapper.Position(len("SELECT")).Line)
	require.Equal(t, int32(7), mapper.Position(len("SELECT")).Column)
}

func TestByteOffsetPositionMapperLineBreaksMatchCalculateLineAndColumn(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "lf",
			sql:  "SELECT 1;\nSELECT 日本;",
		},
		{
			name: "crlf",
			sql:  "SELECT 1;\r\nSELECT 日本;",
		},
		{
			name: "cr",
			sql:  "SELECT 1;\rSELECT 日本;",
		},
		{
			name: "mixed",
			sql:  "SELECT 1;\r\nSELECT 日本;\rSELECT 3;\nSELECT 4;",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapper := NewByteOffsetPositionMapper(test.sql)
			offsets := []int{len(test.sql)}
			for offset := range test.sql {
				offsets = append(offsets, offset)
			}
			for _, offset := range offsets {
				line, column := CalculateLineAndColumn(test.sql, offset)
				got := mapper.Position(offset)
				require.Equal(t, int32(line+1), got.Line, "offset %d", offset)
				require.Equal(t, int32(column+1), got.Column, "offset %d", offset)
			}
		})
	}
}
