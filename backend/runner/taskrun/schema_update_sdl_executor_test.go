package taskrun

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
)

func TestSheetResourceParsing(t *testing.T) {
	// Test that we can use the common function correctly
	tests := []struct {
		name           string
		resource       string
		expectedSha256 string
		expectError    bool
	}{
		{
			name:           "valid_resource",
			resource:       "projects/123/sheets/abc123",
			expectedSha256: "abc123",
			expectError:    false,
		},
		{
			name:           "another_valid_resource",
			resource:       "projects/myproject/sheets/def456",
			expectedSha256: "def456",
			expectError:    false,
		},
		{
			name:           "invalid_format_missing_sheets",
			resource:       "projects/123/documents/456",
			expectedSha256: "",
			expectError:    true,
		},
		{
			name:           "invalid_format_wrong_structure",
			resource:       "projects/123",
			expectedSha256: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sheetSha256, err := common.GetProjectResourceIDSheetSha256(tt.resource)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedSha256, sheetSha256)
			}
		})
	}
}
