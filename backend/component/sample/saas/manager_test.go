package saas

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sample"
)

func TestCheckAvailableRejectsUnconfiguredManager(t *testing.T) {
	var manager *Manager
	err := manager.CheckAvailable(context.Background())
	require.Error(t, err)
	require.Equal(t, sample.FailureUnavailable, sample.FailureKindOf(err))
}

func TestSampleNamesAreStableAndShort(t *testing.T) {
	database, role := sampleNames("sample-0123456789abcdef")
	require.Equal(t, "bb_sample_d3bc52190e66d2ca", database)
	require.Equal(t, "bb_sample_role_d3bc52190e66d2ca", role)
}
