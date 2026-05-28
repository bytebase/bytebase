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
	if os.Getenv("GITHUB_EVENT_NAME") != "pull_request" {
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
	if os.Getenv("CI_PIPELINE_SOURCE") != "merge_request_event" {
		return nil
	}
	userID := os.Getenv("GITLAB_USER_ID")
	if userID == "" {
		return nil
	}
	userName := os.Getenv("GITLAB_USER_LOGIN")
	if userName == "" {
		userName = os.Getenv("GITLAB_USER_NAME")
	}
	if isBotUser("", userName) {
		return nil
	}
	return &v1pb.VCSUser{
		VcsType:     v1pb.VCSType_GITLAB,
		UserId:      userID,
		UserName:    userName,
		DisplayName: os.Getenv("GITLAB_USER_NAME"),
	}
}

func getBitbucketVCSUser() *v1pb.VCSUser {
	if os.Getenv("BITBUCKET_PR_ID") == "" {
		return nil
	}
	userID := strings.Trim(os.Getenv("BITBUCKET_STEP_TRIGGERER_UUID"), "{}")
	if userID == "" {
		return nil
	}
	userName := os.Getenv("BITBUCKET_STEP_TRIGGERER_USERNAME")
	if isBotUser("", userName) {
		return nil
	}
	return &v1pb.VCSUser{
		VcsType:  v1pb.VCSType_BITBUCKET,
		UserId:   userID,
		UserName: userName,
	}
}

func isBotUser(userType, userName string) bool {
	return strings.EqualFold(userType, "bot") || strings.HasSuffix(strings.ToLower(userName), "[bot]")
}
