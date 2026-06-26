package store

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataDBConnectionCloseRunsCleanup(t *testing.T) {
	cleanupCalls := 0
	conn := &metadataDBConnection{
		cleanup: func() error {
			cleanupCalls++
			return nil
		},
	}

	err := conn.close()

	require.NoError(t, err)
	require.Equal(t, 1, cleanupCalls)
}

func TestMetadataDBConnectionCloseReturnsCleanupError(t *testing.T) {
	conn := &metadataDBConnection{
		cleanup: func() error {
			return errors.New("cleanup failed")
		},
	}

	err := conn.close()

	require.ErrorContains(t, err, "cleanup failed")
}
