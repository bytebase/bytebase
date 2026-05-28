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
