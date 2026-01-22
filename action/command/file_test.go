package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple version",
			input:    "3_desc.sql",
			expected: "3",
		},
		{
			name:     "version with minor",
			input:    "3.1_desc.sql",
			expected: "3.1",
		},
		{
			name:     "semantic version",
			input:    "3.1.1_desc.sql",
			expected: "3.1.1",
		},
		{
			name:     "with v prefix",
			input:    "v3.1.1_desc.sql",
			expected: "3.1.1",
		},
		{
			name:     "with V prefix",
			input:    "V3.1.1_desc.sql",
			expected: "3.1.1",
		},
		{
			name:     "with additional text",
			input:    "3.1.1_description_with_more_text.sql",
			expected: "3.1.1",
		},
		{
			name:     "with pre-release version",
			input:    "3.1.1-beta_desc.sql",
			expected: "3.1.1",
		},
		{
			name:     "with v prefix and pre-release version",
			input:    "v3.1.1-alpha_desc.sql",
			expected: "3.1.1",
		},
		{
			name:     "timestamp version",
			input:    "202101130001_desc.sql",
			expected: "202101130001",
		},
		{
			name:     "timestamp version with v prefix",
			input:    "v202101130001_desc.sql",
			expected: "202101130001",
		},
		{
			name:     "timestamp version with V prefix",
			input:    "V202101130001_desc.sql",
			expected: "202101130001",
		},
		{
			name:     "timestamp version with additional text",
			input:    "202101130001_migration_description.sql",
			expected: "202101130001",
		},
		{
			name:     "no version",
			input:    "no_version_here.sql",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just v",
			input:    "v_desc.sql",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersion(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
