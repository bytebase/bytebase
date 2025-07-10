package command

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func getReleaseFiles(w *world.World, pattern string) ([]*v1pb.Release_File, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var files []*v1pb.Release_File
	for _, m := range matches {
		content, err := os.ReadFile(m)
		if err != nil {
			return nil, err
		}
		base := filepath.Base(m)
		t := v1pb.Release_File_DDL
		switch {
		case strings.HasSuffix(base, "dml"):
			t = v1pb.Release_File_DML
		case strings.HasSuffix(base, "ghost"):
			t = v1pb.Release_File_DDL_GHOST
		}

		version := extractVersion(base)
		if version == "" {
			w.Logger.Warn("version not found. ignore the file", "file", m)
			continue
		}

		files = append(files, &v1pb.Release_File{
			Path:       m,
			Type:       v1pb.ReleaseFileType_VERSIONED,
			Version:    version,
			ChangeType: t,
			Statement:  content,
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
