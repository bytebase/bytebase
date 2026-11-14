package command

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const versionFormat = "20060102.150405"

// getReleaseFiles returns the release files.
func getReleaseFiles(w *world.World) ([]*v1pb.Release_File, error) {
	// Use doublestar for recursive glob support (**/*.sql)
	matches, err := doublestar.FilepathGlob(w.FilePattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, errors.Errorf("no files found for pattern: %s", w.FilePattern)
	}

	slices.Sort(matches)

	root := repositoryRoot()

	if w.Declarative {
		// For declarative files, we need to concat all the file contents if it is rollout.
		if w.IsRollout {
			var contents []byte
			for _, m := range matches {
				content, err := os.ReadFile(m)
				if err != nil {
					return nil, err
				}
				// Add newline separator between files to prevent SQL statements from being concatenated
				if len(contents) > 0 {
					contents = append(contents, '\n')
				}
				contents = append(contents, content...)
			}
			return []*v1pb.Release_File{
				{
					// use file pattern as the path
					Path:      w.FilePattern,
					Version:   w.CurrentTime.Format(versionFormat),
					Statement: contents,
				},
			}, nil
		}

		var files []*v1pb.Release_File
		for _, m := range matches {
			content, err := os.ReadFile(m)
			if err != nil {
				return nil, err
			}
			files = append(files, &v1pb.Release_File{
				Path:      repositoryPath(root, m),
				Version:   w.CurrentTime.Format(versionFormat),
				Statement: content,
			})
		}
		return files, nil
	}

	var files []*v1pb.Release_File
	for _, m := range matches {
		content, err := os.ReadFile(m)
		if err != nil {
			return nil, err
		}

		base := strings.TrimSuffix(filepath.Base(m), filepath.Ext(m))

		version := extractVersion(base)
		if version == "" {
			w.Logger.Warn("version not found. ignore the file", "file", m)
			continue
		}

		files = append(files, &v1pb.Release_File{
			Path:      repositoryPath(root, m),
			Version:   version,
			Statement: content,
		})
	}

	return files, nil
}

// repositoryRootEnvs are the checkout directories of GitHub Actions, GitLab CI,
// Bitbucket Pipelines and Azure Pipelines.
var repositoryRootEnvs = []string{"GITHUB_WORKSPACE", "CI_PROJECT_DIR", "BITBUCKET_CLONE_DIR", "BUILD_SOURCESDIRECTORY"}

// repositoryRoot returns the root of the checked-out repository, or "" if
// unknown. It looks for .git itself because the action images have no git binary.
func repositoryRoot() string {
	if dir, err := os.Getwd(); err == nil {
		if root := gitRepositoryRoot(dir); root != "" {
			return root
		}
	}
	for _, key := range repositoryRootEnvs {
		if dir := os.Getenv(key); dir != "" {
			return dir
		}
	}
	return ""
}

// gitRepositoryRoot returns the working tree at or above dir that is not a
// submodule, or "" if there is none. A submodule's git directory lies under
// modules/ of the git directory of the repository above it, and its files are
// named from that superproject, whose URL and SHA the release records.
func gitRepositoryRoot(dir string) string {
	root, rootGitDir := "", ""
	for {
		if gitDir := gitDirectory(dir); gitDir != "" {
			if root != "" {
				if _, ok := pathWithin(filepath.Join(gitDir, "modules"), rootGitDir); !ok {
					return root
				}
			}
			root, rootGitDir = dir, gitDir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return root
		}
		dir = parent
	}
}

// gitDirectory returns the git directory of the working tree at dir: dir/.git,
// or the gitdir a .git file points at. It returns "" if dir has no .git.
func gitDirectory(dir string) string {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return ""
	}
	gitDir := gitPath
	if !info.IsDir() {
		if content, err := os.ReadFile(gitPath); err == nil {
			gitDir = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(content)), "gitdir:"))
			if !filepath.IsAbs(gitDir) {
				gitDir = filepath.Join(dir, gitDir)
			}
		}
	}
	// Git writes a linked worktree's gitdir as a resolved path, so resolve
	// symlinks before comparing it with a path built from the working directory.
	if resolved, err := filepath.EvalSymlinks(gitDir); err == nil {
		return resolved
	}
	return filepath.Clean(gitDir)
}

// repositoryPath returns match relative to root with forward slashes, which is
// how the VCS file links and the SQL review annotations address a file. A match
// that cannot be placed under root is returned unchanged.
func repositoryPath(root, match string) string {
	if root == "" {
		return match
	}
	abs, err := filepath.Abs(match)
	if err != nil {
		return match
	}
	rel, ok := pathWithin(root, abs)
	if !ok {
		return match
	}
	return filepath.ToSlash(rel)
}

// pathWithin returns path relative to base when path lies under base.
func pathWithin(base, path string) (string, bool) {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

var versionReg = regexp.MustCompile(`^[vV]?(\d+(\.\d+)*)`)

// extractVersion extracts version from a string and removes the optional "v" or "V" prefix
func extractVersion(s string) string {
	matches := versionReg.FindStringSubmatch(s)
	if len(matches) < 2 {
		return ""
	}

	// Return the first capture group which contains just the version numbers
	return matches[1]
}
