package gitops

import (
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
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

func getChangesByFileList(files []*vcs.PullRequestFile, branch string) []*fileChange {
	changes := []*fileChange{}
	for _, v := range files {
		if v.IsDeleted {
			continue
		}
		if filepath.Dir(v.Path) != branch {
			continue
		}
		change, err := getFileChange(v.Path)
		if err != nil {
			slog.Error("failed to get file change info", slog.String("path", v.Path), log.BBError(err))
		}
		if change != nil {
			change.path = v.Path
			changes = append(changes, change)
		}
	}
	return changes
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
	description = strings.TrimLeft(description, "_")
	changeType := v1pb.Plan_ChangeDatabaseConfig_MIGRATE
	switch {
	case strings.HasPrefix(description, "ddl"):
		description = strings.TrimPrefix(description, "ddl")
		changeType = v1pb.Plan_ChangeDatabaseConfig_MIGRATE
	case strings.HasPrefix(description, "dml"):
		description = strings.TrimPrefix(description, "dml")
		changeType = v1pb.Plan_ChangeDatabaseConfig_DATA
	case strings.HasPrefix(description, "ghost"):
		description = strings.TrimPrefix(description, "ghost")
		changeType = v1pb.Plan_ChangeDatabaseConfig_MIGRATE_GHOST
	}
	description = strings.TrimLeft(description, "_")
	return &fileChange{
		path:        path,
		version:     version,
		changeType:  changeType,
		description: description,
	}, nil
}
