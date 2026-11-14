package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// workflowCasesDir holds the cases the GitOps page's generator is tested on
// (gitopsWorkflowYaml.test.ts writes the expected files). The path is relative
// to this package directory, where `go test` runs.
const workflowCasesDir = "../../../frontend/src/routes/project/gitops-workflow-cases"

func TestGenerateWorkflowFilesMatchesTheGitOpsPage(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(workflowCasesDir, "cases.json"))
	require.NoError(t, err)
	var cases []struct {
		Name   string         `json:"name"`
		Params workflowParams `json:"params"`
	}
	require.NoError(t, json.Unmarshal(data, &cases))
	require.NotEmpty(t, cases)

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join(workflowCasesDir, c.Name+".txt"))
			require.NoError(t, err)
			var rendered strings.Builder
			for _, file := range generateWorkflowFiles(c.Params) {
				rendered.WriteString("=== " + file.Path + " ===\n" + file.Content)
			}
			require.Equal(t, string(expected), rendered.String())
		})
	}
}

func TestBytebaseActionImage(t *testing.T) {
	for version, image := range map[string]string{
		"cloud-20261023": "bytebase/bytebase-action:cloud",
		"3.24.0":         "bytebase/bytebase-action:3.24.0",
		"v3.24.0":        "bytebase/bytebase-action:3.24.0",
		"development":    "bytebase/bytebase-action",
		"":               "bytebase/bytebase-action",
	} {
		require.Equal(t, image, bytebaseActionImage(version), version)
	}
}

func TestIsValidMigrationFolder(t *testing.T) {
	for folder, valid := range map[string]bool{
		"migrations":       true,
		"db/migrations":    true,
		"db/v1.2_changes":  true,
		"":                 false,
		"/migrations":      false,
		"migrations/":      false,
		"db/../migrations": false,
		".":                false,
		"my migrations":    false,
		"migrations/*":     false,
	} {
		require.Equal(t, valid, isValidMigrationFolder(folder), folder)
	}
}
