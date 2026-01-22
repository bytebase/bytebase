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
				Path:      m,
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
			Path:      m,
			Version:   version,
			Statement: content,
		})
	}

	return files, nil
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
