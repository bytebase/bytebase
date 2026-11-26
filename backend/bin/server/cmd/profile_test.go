package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBaseProfileTurnsOnSQLReviewV2OnlyForCloud(t *testing.T) {
	originalFlags := flags
	t.Cleanup(func() { flags = originalFlags })

	flags.saas = false
	require.False(t, getBaseProfile(t.TempDir()).SQLReviewV2, "a self-hosted release build keeps V1")

	flags.saas = true
	require.True(t, getBaseProfile(t.TempDir()).SQLReviewV2)
}
