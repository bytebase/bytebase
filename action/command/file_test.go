package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/world"
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

func TestGetReleaseFilesRepositoryPath(t *testing.T) {
	tests := []struct {
		name        string
		outerGit    string            // .git in the temp dir: "dir", a gitdir for a .git file, or "" for none
		repoGit     string            // .git in repo/, as outerGit
		submodules  map[string]string // directory under repo/ to its .git file's gitdir
		env         map[string]string
		cwd         string // relative to the temp dir
		pattern     string // {tmp} is replaced with the temp dir
		declarative bool
		rollout     bool
		want        []string // {tmp} is replaced with the temp dir
	}{
		{
			name:    "subdirectory run",
			repoGit: "dir",
			cwd:     "repo/db",
			pattern: "migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "subdirectory run with a parent pattern",
			repoGit: "dir",
			cwd:     "repo/db",
			pattern: "../shared/*.sql",
			want:    []string{"shared/003_shared.sql"},
		},
		{
			name:    "repository root run",
			repoGit: "dir",
			cwd:     "repo",
			pattern: "db/migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "absolute pattern",
			repoGit: "dir",
			cwd:     "repo/db",
			pattern: "{tmp}/repo/db/migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "git file from a worktree checkout",
			repoGit: "/elsewhere/.git/worktrees/repo",
			cwd:     "repo/db",
			pattern: "migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:       "submodule run is named from the superproject",
			repoGit:    "dir",
			submodules: map[string]string{"db": "../.git/modules/db"},
			cwd:        "repo/db",
			pattern:    "migrations/*.sql",
			want:       []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:       "nested submodule run is named from the outermost superproject",
			repoGit:    "dir",
			submodules: map[string]string{"db": "../.git/modules/db", "db/migrations": "../../.git/modules/db/modules/migrations"},
			cwd:        "repo/db/migrations",
			pattern:    "*.sql",
			want:       []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "GitHub workspace without git",
			env:     map[string]string{"GITHUB_WORKSPACE": "{tmp}/repo"},
			cwd:     "repo/db",
			pattern: "migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "GitLab project directory without git",
			env:     map[string]string{"CI_PROJECT_DIR": "{tmp}/repo"},
			cwd:     "repo/db",
			pattern: "migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "Bitbucket clone directory without git",
			env:     map[string]string{"BITBUCKET_CLONE_DIR": "{tmp}/repo"},
			cwd:     "repo/db",
			pattern: "migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "Azure sources directory without git",
			env:     map[string]string{"BUILD_SOURCESDIRECTORY": "{tmp}/repo"},
			cwd:     "repo/db",
			pattern: "migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:       "submodule in a linked worktree is named from the superproject",
			repoGit:    "/main/.git/worktrees/repo",
			submodules: map[string]string{"db": "/main/.git/worktrees/repo/modules/db"},
			cwd:        "repo/db",
			pattern:    "migrations/*.sql",
			want:       []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:       "submodule named worktrees",
			repoGit:    "dir",
			submodules: map[string]string{"db": "../.git/modules/worktrees"},
			cwd:        "repo/db",
			pattern:    "migrations/*.sql",
			want:       []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "linked worktree named modules",
			repoGit: "/main/.git/worktrees/modules",
			cwd:     "repo/db",
			pattern: "migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:     "linked worktree inside its main checkout",
			outerGit: "dir",
			repoGit:  "{tmp}/.git/worktrees/repo",
			cwd:      "repo/db",
			pattern:  "migrations/*.sql",
			want:     []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:     "independent clone inside another repository",
			outerGit: "dir",
			repoGit:  "dir",
			cwd:      "repo/db",
			pattern:  "migrations/*.sql",
			want:     []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "git root wins over the checkout variable",
			repoGit: "dir",
			env:     map[string]string{"GITHUB_WORKSPACE": "{tmp}"},
			cwd:     "repo/db",
			pattern: "migrations/*.sql",
			want:    []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:    "no git and no checkout variable",
			cwd:     "repo/db",
			pattern: "migrations/*.sql",
			want:    []string{"migrations/001_init.sql", "migrations/002_add.sql"},
		},
		{
			name:    "file outside the repository",
			repoGit: "dir",
			cwd:     "repo",
			pattern: "{tmp}/outside/*.sql",
			want:    []string{"{tmp}/outside/004_outside.sql"},
		},
		{
			name:        "declarative check",
			repoGit:     "dir",
			cwd:         "repo/db",
			pattern:     "migrations/*.sql",
			declarative: true,
			want:        []string{"db/migrations/001_init.sql", "db/migrations/002_add.sql"},
		},
		{
			name:        "declarative rollout keeps the pattern",
			repoGit:     "dir",
			cwd:         "repo/db",
			pattern:     "migrations/*.sql",
			declarative: true,
			rollout:     true,
			want:        []string{"migrations/*.sql"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			for _, f := range []string{
				"repo/db/migrations/001_init.sql",
				"repo/db/migrations/002_add.sql",
				"repo/shared/003_shared.sql",
				"outside/004_outside.sql",
			} {
				path := filepath.Join(tmp, filepath.FromSlash(f))
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte("SELECT 1;"), 0o600))
			}
			expand := func(s string) string { return strings.ReplaceAll(s, "{tmp}", filepath.ToSlash(tmp)) }
			writeGit(t, tmp, expand(tt.outerGit))
			writeGit(t, filepath.Join(tmp, "repo"), expand(tt.repoGit))
			for dir, gitdir := range tt.submodules {
				writeGit(t, filepath.Join(tmp, "repo", filepath.FromSlash(dir)), gitdir)
			}
			for _, key := range repositoryRootEnvs {
				t.Setenv(key, expand(tt.env[key]))
			}
			t.Chdir(filepath.Join(tmp, filepath.FromSlash(tt.cwd)))

			w := world.NewWorld()
			w.CurrentTime = time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
			w.FilePattern = filepath.FromSlash(expand(tt.pattern))
			w.Declarative = tt.declarative
			w.IsRollout = tt.rollout

			files, err := getReleaseFiles(w)
			require.NoError(t, err)
			var got []string
			for _, f := range files {
				got = append(got, filepath.ToSlash(f.Path))
			}
			var want []string
			for _, p := range tt.want {
				want = append(want, expand(p))
			}
			require.Equal(t, want, got)
		})
	}
}

// writeGit gives dir a .git directory ("dir") or a .git file pointing at
// gitdir, creating gitdir when it is relative, as Git does for submodules.
func writeGit(t *testing.T, dir, git string) {
	t.Helper()
	switch git {
	case "":
	case "dir":
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	default:
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: "+git+"\n"), 0o600))
		if !filepath.IsAbs(git) {
			require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.FromSlash(git)), 0o755))
		}
	}
}
