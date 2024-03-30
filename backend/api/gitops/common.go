package gitops

import (
	"path/filepath"
	"regexp"
	"strings"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	versionRE = regexp.MustCompile(`^[0-9]+`)
)

type pullRequestInfo struct {
	email       string
	title       string
	description string
	url         string
	changes     []*fileChange
}

type fileChange struct {
	path        string
	version     string
	changeType  v1pb.Plan_ChangeDatabaseConfig_Type
	description string
	content     string
}

func getFileChange(path string) (*fileChange, error) {
	filename := filepath.Base(path)
	if filepath.Ext(filename) != ".sql" {
		return nil, nil
	}

	matches := versionRE.FindAllString(filename, -1)
	if len(matches) == 0 {
		return nil, nil
	}
	version := matches[0]
	description := strings.TrimPrefix(filename, version)
	description = strings.TrimLeft(description, "_-#")
	changeType := v1pb.Plan_ChangeDatabaseConfig_MIGRATE
	switch {
	case strings.HasPrefix(description, "ddl"):
		description = strings.TrimPrefix(description, "ddl")
	case strings.HasPrefix(description, "migrate"):
		description = strings.TrimPrefix(description, "migrate")
	case strings.HasPrefix(description, "dml"):
		description = strings.TrimPrefix(description, "dml")
		changeType = v1pb.Plan_ChangeDatabaseConfig_DATA
	case strings.HasPrefix(description, "data"):
		description = strings.TrimPrefix(description, "data")
		changeType = v1pb.Plan_ChangeDatabaseConfig_DATA
	}
	description = strings.TrimLeft(description, "_-#")
	return &fileChange{
		path:        path,
		version:     version,
		changeType:  changeType,
		description: description,
	}, nil
}
