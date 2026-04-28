//go:build !minidemo

package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	dbRegisterPattern     = regexp.MustCompile(`db\.Register\(\s*storepb\.Engine_[A-Z0-9_]+`)
	parserRegisterPattern = regexp.MustCompile(`base\.Register[A-Za-z0-9_]*\(\s*storepb\.Engine_[A-Z0-9_]+`)
)

func TestUltimateImportsAllDBRegistrationPackages(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	expected := mustCollectRegistrationPackages(t, filepath.Join(repoRoot, "backend/plugin/db"), dbRegisterPattern, "github.com/bytebase/bytebase/backend/plugin/db")
	imported := mustCollectServerDeps(t, repoRoot)

	missing := setDifference(expected, imported)
	require.Emptyf(t, missing, "missing db plugin dependencies in backend/server build graph: %v", sortedKeys(missing))
}

func TestUltimateImportsAllParserRegistrationPackages(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	expected := mustCollectRegistrationPackages(t, filepath.Join(repoRoot, "backend/plugin/parser"), parserRegisterPattern, "github.com/bytebase/bytebase/backend/plugin/parser")
	imported := mustCollectServerDeps(t, repoRoot)

	missing := setDifference(expected, imported)
	require.Emptyf(t, missing, "missing parser plugin dependencies in backend/server build graph: %v", sortedKeys(missing))
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	_, filePath, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filePath), "..", ".."))
}

func mustCollectRegistrationPackages(t *testing.T, rootDir string, registerPattern *regexp.Regexp, importPathPrefix string) map[string]struct{} {
	t.Helper()
	pkgs := make(map[string]struct{})

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !registerPattern.Match(content) {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		segments := strings.Split(relPath, string(filepath.Separator))
		if len(segments) > 0 {
			pkgs[filepath.ToSlash(filepath.Join(importPathPrefix, segments[0]))] = struct{}{}
		}
		return nil
	})
	require.NoError(t, err)
	return pkgs
}

func mustCollectServerDeps(t *testing.T, repoRoot string) map[string]struct{} {
	t.Helper()
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", "github.com/bytebase/bytebase/backend/server")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	require.NoError(t, err)

	deps := make(map[string]struct{})
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		deps[line] = struct{}{}
	}
	return deps
}

func setDifference(left, right map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	for key := range left {
		if _, ok := right[key]; !ok {
			result[key] = struct{}{}
		}
	}
	return result
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
