package audit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateServerID(t *testing.T) {
	t.Run("generates valid server ID", func(t *testing.T) {
		id, err := GenerateServerID()
		require.NoError(t, err)
		require.NotEmpty(t, id)

		// Verify format: source-timestamp-random
		parts := strings.Split(id, "-")
		assert.GreaterOrEqual(t, len(parts), 3)

		// Verify length constraint
		assert.LessOrEqual(t, len(id), 255)
	})

	t.Run("respects BYTEBASE_INSTANCE_ID env var", func(t *testing.T) {
		t.Setenv("BYTEBASE_INSTANCE_ID", "test-server")

		id, err := GenerateServerID()
		require.NoError(t, err)
		assert.Contains(t, id, "test-server")
	})

	t.Run("truncates very long source", func(t *testing.T) {
		longSource := strings.Repeat("x", 300)
		t.Setenv("BYTEBASE_INSTANCE_ID", longSource)

		id, err := GenerateServerID()
		require.NoError(t, err)
		assert.LessOrEqual(t, len(id), 255)
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id, err := GenerateServerID()
			require.NoError(t, err)
			assert.False(t, ids[id], "duplicate ID: %s", id)
			ids[id] = true
		}
	})
}

func TestValidateServerID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wantError bool
	}{
		{"valid", "bytebase-20251029-abc123", false},
		{"empty", "", true},
		{"too long", strings.Repeat("x", 256), true},
		{"max length", strings.Repeat("x", 255), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerID(tt.id)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
