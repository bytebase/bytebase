package migrator

import (
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/require"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestLatestVersion(t *testing.T) {
	files, err := getSortedVersionedFiles()
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("3.11.20"), *files[len(files)-1].version)
}

func TestVersionUnique(t *testing.T) {
	files, err := getSortedVersionedFiles()
	require.NoError(t, err)
	versions := make(map[string]struct{})
	for _, file := range files {
		if file.version == nil {
			continue
		}
		if _, ok := versions[file.version.String()]; ok {
			require.Fail(t, "duplicate version %s", file.version.String())
		}
		versions[file.version.String()] = struct{}{}
	}
}

func TestGetVersionFromPath(t *testing.T) {
	v, err := getVersionFromPath("migration/3.5/0000##vcs.sql")
	require.NoError(t, err)
	require.Equal(t, *v, semver.MustParse("3.5.0"))
}
