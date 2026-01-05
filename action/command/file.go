package command

import (
	"crypto/sha256"
	"encoding/hex"
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

// getReleaseFiles returns the release files and the digest of the release.
func getReleaseFiles(w *world.World) ([]*v1pb.Release_File, string, error) {
	// Use doublestar for recursive glob support (**/*.sql)
	matches, err := doublestar.FilepathGlob(w.FilePattern)
	if err != nil {
		return nil, "", err
	}
	if len(matches) == 0 {
		return nil, "", errors.Errorf("no files found for pattern: %s", w.FilePattern)
	}

	slices.Sort(matches)

	h := sha256.New()

	if w.Declarative {
		// For declarative files, we need to concat all the file contents if it is rollout.
		if w.IsRollout {
			var contents []byte
			for _, m := range matches {
				content, err := os.ReadFile(m)
				if err != nil {
					return nil, "", err
				}
				if _, err := h.Write([]byte(m)); err != nil {
					return nil, "", errors.Wrapf(err, "failed to write file path")
				}
				if _, err := h.Write(content); err != nil {
					return nil, "", errors.Wrapf(err, "failed to write file content")
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
					Path:        w.FilePattern,
					Version:     w.CurrentTime.Format(versionFormat),
					EnableGhost: false,
					Statement:   contents,
				},
			}, hex.EncodeToString(h.Sum(nil)), nil
		}

		var files []*v1pb.Release_File
		for _, m := range matches {
			content, err := os.ReadFile(m)
			if err != nil {
				return nil, "", err
			}
			if _, err := h.Write([]byte(m)); err != nil {
				return nil, "", errors.Wrapf(err, "failed to write file path")
			}
			if _, err := h.Write(content); err != nil {
				return nil, "", errors.Wrapf(err, "failed to write file content")
			}
			files = append(files, &v1pb.Release_File{
				Path:        m,
				Version:     w.CurrentTime.Format(versionFormat),
				EnableGhost: false,
				Statement:   content,
			})
		}
		return files, hex.EncodeToString(h.Sum(nil)), nil
	}

	var files []*v1pb.Release_File
	for _, m := range matches {
		content, err := os.ReadFile(m)
		if err != nil {
			return nil, "", err
		}

		base := strings.TrimSuffix(filepath.Base(m), filepath.Ext(m))

		// Extract migration type from SQL front matter comments
		t := extractMigrationTypeFromContent(string(content))

		version := extractVersion(base)
		if version == "" {
			w.Logger.Warn("version not found. ignore the file", "file", m)
			continue
		}

		if _, err := h.Write([]byte(m)); err != nil {
			return nil, "", errors.Wrapf(err, "failed to write file path")
		}
		if _, err := h.Write(content); err != nil {
			return nil, "", errors.Wrapf(err, "failed to write file content")
		}
		files = append(files, &v1pb.Release_File{
			Path:        m,
			Version:     version,
			EnableGhost: t,
			Statement:   content,
		})
	}

	return files, hex.EncodeToString(h.Sum(nil)), nil
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

var migrationTypeReg = regexp.MustCompile(`(?i)^--\s*migration-type:\s*(\w+)`)

// extractMigrationTypeFromContent extracts enable_ghost setting from SQL front matter comments.
// Returns true if "-- migration-type: ghost" is found.
// Example:
//
//	-- migration-type: ghost
//	ALTER TABLE ...
func extractMigrationTypeFromContent(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Stop at first non-comment line
		if line != "" && !strings.HasPrefix(line, "--") {
			break
		}
		matches := migrationTypeReg.FindStringSubmatch(line)
		if len(matches) >= 2 {
			if strings.ToLower(matches[1]) == "ghost" {
				return true
			}
		}
	}
	return false
}
