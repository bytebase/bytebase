package main

import (
	"os"
	"path/filepath"
	"strings"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func getReleaseFiles(pattern string) ([]*v1pb.Release_File, error) {
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
		case strings.Contains(base, "dml"):
			t = v1pb.Release_File_DML
		case strings.Contains(base, "ghost"):
			t = v1pb.Release_File_DDL_GHOST
		}
		version := ""
		fields := strings.Fields(base)
		if len(fields) > 0 {
			version = fields[0]
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
