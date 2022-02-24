package cmd

import (
	"testing"
)

func TestVersion(t *testing.T) {
	tt := []testTable{
		{
			args: []string{"version"},
			expected: `bb version: development
Golang version: unknown
Git commit hash: unknown
Built on: unknown
Built by: unknown
`,
		},
		{
			args: []string{"version", "--help"},
			expected: `Print the version of bb

Usage:
  bb version [flags]

Flags:
  -h, --help   help for version
`,
		},
	}

	tableTest(t, tt)
}
