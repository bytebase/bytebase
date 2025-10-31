package v1

import (
	"strings"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestDatabaseGroupTitleValidation(t *testing.T) {
	validator, err := protovalidate.New()
	require.NoError(t, err)

	tests := []struct {
		name      string
		title     string
		wantError bool
		errMsg    string
	}{
		{
			name:      "valid title",
			title:     "Valid Group Title",
			wantError: false,
		},
		{
			name:      "title at max length (200 chars)",
			title:     strings.Repeat("a", 200),
			wantError: false,
		},
		{
			name:      "title exceeding max length (201 chars)",
			title:     strings.Repeat("a", 201),
			wantError: true,
			errMsg:    "title",
		},
		{
			name:      "single character title (valid)",
			title:     "A",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			databaseGroup := &v1pb.DatabaseGroup{
				Title: tt.title,
			}

			err := validator.Validate(databaseGroup)
			if tt.wantError {
				require.Error(t, err, "Expected validation error for title: %q", tt.title)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err, "Expected no validation error for title: %q", tt.title)
			}
		})
	}
}
