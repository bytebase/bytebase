package command

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const versionFormat = "20060102.150405"

// getReleaseFiles returns the release files and the digest of the release.
func getReleaseFiles(w *world.World) ([]*v1pb.Release_File, string, error) {
	matches, err := filepath.Glob(w.FilePattern)
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
				contents = append(contents, content...)
			}
			return []*v1pb.Release_File{
				{
					// use file pattern as the path
					Path:       w.FilePattern,
					Type:       v1pb.Release_File_DECLARATIVE,
					Version:    w.CurrentTime.Format(versionFormat),
					ChangeType: v1pb.Release_File_DDL,
					Statement:  contents,
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
				Path:       m,
				Type:       v1pb.Release_File_DECLARATIVE,
				Version:    w.CurrentTime.Format(versionFormat),
				ChangeType: v1pb.Release_File_DDL,
				Statement:  content,
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
		var t v1pb.Release_File_ChangeType
		switch {
		case strings.HasSuffix(base, "dml"):
			t = v1pb.Release_File_DML
		case strings.HasSuffix(base, "ghost"):
			t = v1pb.Release_File_DDL_GHOST
		default:
			// Default to DDL for files without recognized suffixes
			t = v1pb.Release_File_DDL
		}

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
			Path:       m,
			Type:       v1pb.Release_File_VERSIONED,
			Version:    version,
			ChangeType: t,
			Statement:  content,
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
