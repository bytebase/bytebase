package command

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func getVCSUser(platform world.JobPlatform) *v1pb.VCSUser {
	switch platform {
	case world.GitHub:
		return getGitHubVCSUser()
	case world.GitLab:
		return getGitLabVCSUser()
	case world.Bitbucket:
		return getBitbucketVCSUser()
	default:
		return nil
	}
}

func getGitHubVCSUser() *v1pb.VCSUser {
	eventName := os.Getenv("GITHUB_EVENT_NAME")
	if eventName != "pull_request" && eventName != "pull_request_target" {
		return nil
	}
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil
	}
	data, err := os.ReadFile(eventPath)
	if err != nil {
		return nil
	}

	var event struct {
		PullRequest struct {
			User struct {
				ID    int64  `json:"id"`
				Login string `json:"login"`
				Name  string `json:"name"`
				Type  string `json:"type"`
			} `json:"user"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return nil
	}

	user := event.PullRequest.User
	if user.ID == 0 || isBotUser(user.Type, user.Login) {
		return nil
	}
	return &v1pb.VCSUser{
		VcsType:     v1pb.VCSType_GITHUB,
		UserId:      strconv.FormatInt(user.ID, 10),
		UserName:    user.Login,
		DisplayName: user.Name,
	}
}

func getGitLabVCSUser() *v1pb.VCSUser {
	return nil
}

func getBitbucketVCSUser() *v1pb.VCSUser {
	return nil
}

func isBotUser(userType, userName string) bool {
	return strings.EqualFold(userType, "bot") || strings.HasSuffix(strings.ToLower(userName), "[bot]")
}
