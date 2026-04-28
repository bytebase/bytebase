package v1

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMilvusWiringContract(t *testing.T) {
	repoRoot := mustFindRepoRoot(t)

	requiredDirs := []string{
		"backend/plugin/db/milvus",
		"backend/plugin/parser/milvus",
	}

	for _, relPath := range requiredDirs {
		t.Run("milvus_scaffold_exists_"+strings.ReplaceAll(relPath, "/", "_"), func(t *testing.T) {
			require.DirExists(t, filepath.Join(repoRoot, relPath))
		})
	}

	require.FileExists(t, filepath.Join(repoRoot, "backend/plugin/db/milvus/milvus_test.go"))

	tests := []struct {
		name     string
		filePath string
		patterns []string
	}{
		{
			name:     "store proto declares MILVUS engine",
			filePath: "proto/store/store/common.proto",
			patterns: []string{"MILVUS"},
		},
		{
			name:     "v1 proto declares MILVUS engine",
			filePath: "proto/v1/v1/common.proto",
			patterns: []string{"MILVUS"},
		},
		{
			name:     "backend api engine mappings include MILVUS both directions",
			filePath: "backend/api/v1/common.go",
			patterns: []string{"storepb.Engine_MILVUS", "v1pb.Engine_MILVUS"},
		},
		{
			name:     "ultimate server registers milvus db and parser plugins",
			filePath: "backend/server/ultimate.go",
			patterns: []string{"backend/plugin/db/milvus", "backend/plugin/parser/milvus"},
		},
		{
			name:     "frontend supported engine list and name include MILVUS",
			filePath: "frontend/src/utils/v1/instance.ts",
			patterns: []string{"Engine.MILVUS"},
		},
		{
			name:     "frontend instance form has milvus port and icon wiring",
			filePath: "frontend/src/components/InstanceForm/constants.ts",
			patterns: []string{"Engine.MILVUS", "@/assets/db/milvus"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := mustReadRepoFile(t, repoRoot, tc.filePath)
			for _, pattern := range tc.patterns {
				require.Containsf(t, content, pattern, "file %q must contain %q", tc.filePath, pattern)
			}
		})
	}
}

func mustFindRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func mustReadRepoFile(t *testing.T, repoRoot, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, relPath))
	require.NoError(t, err)
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}
