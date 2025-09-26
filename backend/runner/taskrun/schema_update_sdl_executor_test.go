package taskrun

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
)

func TestSheetResourceParsing(t *testing.T) {
	// Test that we can use the common function correctly
	tests := []struct {
		name        string
		resource    string
		expectedID  int
		expectError bool
	}{
		{
			name:        "valid_resource",
			resource:    "projects/123/sheets/456",
			expectedID:  456,
			expectError: false,
		},
		{
			name:        "another_valid_resource",
			resource:    "projects/myproject/sheets/789",
			expectedID:  789,
			expectError: false,
		},
		{
			name:        "invalid_format_missing_sheets",
			resource:    "projects/123/documents/456",
			expectedID:  0,
			expectError: true,
		},
		{
			name:        "invalid_format_wrong_structure",
			resource:    "projects/123",
			expectedID:  0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sheetID, err := common.GetProjectResourceIDSheetUID(tt.resource)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedID, sheetID)
			}
		})
	}
}
