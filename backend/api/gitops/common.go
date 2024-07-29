package gitops

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/utils"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	versionRE = regexp.MustCompile(`^[0-9]+`)
)

type webhookAction int

const (
	webhookActionCreateIssue webhookAction = iota
	webhookActionSQLReview
)

type pullRequestInfo struct {
	action      webhookAction
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

func getChangesByFileList(files []*vcs.PullRequestFile, rootDir string) []*fileChange {
	changes := []*fileChange{}
	for _, v := range files {
		if v.IsDeleted {
			continue
		}
		prFilePath := v.Path
		if !strings.HasPrefix(prFilePath, "/") {
			prFilePath = fmt.Sprintf("/%s", prFilePath)
		}
		if filepath.Dir(prFilePath) != rootDir {
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

func getPullRequestID(url string) string {
	fields := strings.Split(url, "/")
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

const (
	commentPrefixBytebaseBot = "[Bytebase Bot]"
	commentPrefixSQLReview   = "[Bytebase SQL Review]"
)

func getPullRequestComment(externalURL, issue string) string {
	return fmt.Sprintf("This pull request has triggered a Bytebase rollout 🚀. Check out the status at %s/%s.", externalURL, issue)
}

func convertFileContentToUTF8String(content string) string {
	convertedContent, err := utils.ConvertBytesToUTF8String([]byte(content))
	if err != nil {
		// After failed to convert to UTF-8, we will try to convert to valid UTF-8.
		convertedContent = strings.ToValidUTF8(content, "")
	}
	return convertedContent
}
