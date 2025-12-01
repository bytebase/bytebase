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

func TestExtractMigrationTypeFromContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		enableGhost bool
	}{
		{
			name: "ghost type",
			content: `-- migration-type: ghost
ALTER TABLE large_table ADD COLUMN new_col VARCHAR(255);`,
			enableGhost: true,
		},
		{
			name: "case insensitive Ghost",
			content: `-- migration-type: Ghost
ALTER TABLE large_table ADD COLUMN new_col VARCHAR(255);`,
			enableGhost: true,
		},
		{
			name: "case insensitive GHOST",
			content: `-- migration-type: GHOST
ALTER TABLE large_table ADD COLUMN new_col VARCHAR(255);`,
			enableGhost: true,
		},
		{
			name: "with extra spaces",
			content: `--   migration-type:   ghost
ALTER TABLE large_table ADD COLUMN new_col VARCHAR(255);`,
			enableGhost: true,
		},
		{
			name: "with multiple comment lines",
			content: `-- This is a migration file
-- Author: John Doe
-- migration-type: ghost
-- Date: 2024-01-01

ALTER TABLE large_table ADD COLUMN new_col VARCHAR(255);`,
			enableGhost: true,
		},
		{
			name: "no migration type specified",
			content: `-- This is a migration file
ALTER TABLE users ADD COLUMN age INT;`,
			enableGhost: false,
		},
		{
			name: "ddl type - should not enable ghost",
			content: `-- migration-type: ddl
ALTER TABLE users ADD COLUMN age INT;`,
			enableGhost: false,
		},
		{
			name: "dml type - should not enable ghost",
			content: `-- migration-type: dml
UPDATE users SET active = true;`,
			enableGhost: false,
		},
		{
			name: "migration type after SQL statement",
			content: `ALTER TABLE users ADD COLUMN age INT;
-- migration-type: ghost`,
			enableGhost: false,
		},
		{
			name: "invalid migration type",
			content: `-- migration-type: invalid
ALTER TABLE users ADD COLUMN age INT;`,
			enableGhost: false,
		},
		{
			name:        "empty content",
			content:     ``,
			enableGhost: false,
		},
		{
			name: "only comments with ghost",
			content: `-- migration-type: ghost
-- More comments`,
			enableGhost: true,
		},
		{
			name: "with blank lines before statement",
			content: `-- migration-type: ghost

ALTER TABLE users ADD COLUMN age INT;`,
			enableGhost: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMigrationTypeFromContent(tt.content)
			require.Equal(t, tt.enableGhost, result)
		})
	}
}
