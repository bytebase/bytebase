package saas

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSampleNamesAreStableAndShort(t *testing.T) {
	database, role := sampleNames("sample-0123456789abcdef")
	require.Equal(t, "bb_sample_d3bc52190e66d2ca", database)
	require.Equal(t, "bb_sample_role_d3bc52190e66d2ca", role)
}
