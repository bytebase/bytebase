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
	require.Equal(t, semver.MustParse("3.15.1"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.15/0001##add_database_release_field.sql", files[len(files)-1].path)
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
