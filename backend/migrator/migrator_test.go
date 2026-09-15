package migrator

import (
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/require"
)

func TestLatestVersion(t *testing.T) {
	files, err := getSortedVersionedFiles()
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("3.23.4"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.23/0004##workload_identity_audiences.sql", files[len(files)-1].path)
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
	tests := []struct {
		name    string
		path    string
		version string
		wantErr bool
	}{
		{
			name:    "main migration",
			path:    "migration/3.23/0004##workload_identity_audiences.sql",
			version: "3.23.4",
		},
		{
			name:    "release patch backport",
			path:    "migration/3.23.1-backport/0000##workload_identity_audiences.sql",
			version: "3.23.1-backport.0",
		},
		{
			name:    "release patch backport ordinal",
			path:    "migration/3.23.1-backport/0010##later_backport.sql",
			version: "3.23.1-backport.10",
		},
		{
			name:    "invalid filename ordinal",
			path:    "migration/3.23.1-backport/000a##later_backport.sql",
			wantErr: true,
		},
		{
			name:    "invalid nested path",
			path:    "migration/3.23.1-backport/nested/0000##later_backport.sql",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version, err := getVersionFromPath(test.path)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, semver.MustParse(test.version), *version)
		})
	}
}

func TestReleasePatchMigrationVersionOrder(t *testing.T) {
	paths := []string{
		"migration/3.23/0000##review_run.sql",
		"migration/3.23.1-backport/0000##workload_identity_audiences.sql",
		"migration/3.23.1-backport/0009##later_backport.sql",
		"migration/3.23.1-backport/0010##later_backport.sql",
		"migration/3.23/0001##issue_comment_thread.sql",
	}

	versions := make([]semver.Version, 0, len(paths))
	for _, path := range paths {
		version, err := getVersionFromPath(path)
		require.NoError(t, err)
		versions = append(versions, *version)
	}

	for i := 1; i < len(versions); i++ {
		require.Truef(t, versions[i-1].LT(versions[i]), "%s must sort before %s", versions[i-1], versions[i])
	}
}
