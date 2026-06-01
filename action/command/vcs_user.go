package command

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/action/bitbucket"
	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

var gitLabBotUserNameRE = regexp.MustCompile(`^(project|group)_\d+_bot_[a-z0-9]+$`)

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
	projectID := os.Getenv("CI_MERGE_REQUEST_PROJECT_ID")
	mergeRequestIID := os.Getenv("CI_MERGE_REQUEST_IID")
	apiURL := os.Getenv("CI_API_V4_URL")
	jobToken := os.Getenv("CI_JOB_TOKEN")
	if projectID == "" || mergeRequestIID == "" || apiURL == "" || jobToken == "" {
		return nil
	}

	requestURL, err := buildGitLabMergeRequestURL(apiURL, projectID, mergeRequestIID)
	if err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("JOB-TOKEN", jobToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		slog.Warn("failed to read GitLab merge request for VCS user attribution", "error", err)
		return nil
	}
	var mergeRequest struct {
		Author struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"author"`
	}
	if err := json.Unmarshal(data, &mergeRequest); err != nil {
		slog.Warn("failed to parse GitLab merge request for VCS user attribution", "error", err)
		return nil
	}

	author := mergeRequest.Author
	if author.ID == 0 || isGitLabBotUser(author.Username) {
		return nil
	}
	return &v1pb.VCSUser{
		VcsType:     v1pb.VCSType_GITLAB,
		UserId:      strconv.FormatInt(author.ID, 10),
		UserName:    author.Username,
		DisplayName: author.Name,
	}
}

func getBitbucketVCSUser() *v1pb.VCSUser {
	pullRequestID := os.Getenv("BITBUCKET_PR_ID")
	if pullRequestID == "" {
		return nil
	}
	workspace, repoSlug := getBitbucketRepository()
	if workspace == "" || repoSlug == "" {
		return nil
	}

	apiURL := getBitbucketAPIBaseURL()
	requestURL, err := buildBitbucketPullRequestURL(apiURL, workspace, repoSlug, pullRequestID)
	if err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")

	resp, err := bitbucket.NewHTTPClient(apiURL).Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		slog.Warn("failed to read Bitbucket pull request for VCS user attribution", "error", err)
		return nil
	}
	var pullRequest struct {
		Author struct {
			AccountID   string `json:"account_id"`
			UUID        string `json:"uuid"`
			Nickname    string `json:"nickname"`
			DisplayName string `json:"display_name"`
			Type        string `json:"type"`
		} `json:"author"`
	}
	if err := json.Unmarshal(data, &pullRequest); err != nil {
		slog.Warn("failed to parse Bitbucket pull request for VCS user attribution", "error", err)
		return nil
	}

	author := pullRequest.Author
	userID := author.AccountID
	if userID == "" {
		userID = strings.Trim(author.UUID, "{}")
	}
	if userID == "" || isBitbucketBotUser(author.Type, author.Nickname) {
		return nil
	}
	return &v1pb.VCSUser{
		VcsType:     v1pb.VCSType_BITBUCKET,
		UserId:      userID,
		UserName:    author.Nickname,
		DisplayName: author.DisplayName,
	}
}

func buildGitLabMergeRequestURL(apiURL, projectID, mergeRequestIID string) (string, error) {
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/projects/" + url.PathEscape(projectID) + "/merge_requests/" + url.PathEscape(mergeRequestIID)
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""
	return parsedURL.String(), nil
}

func getBitbucketRepository() (string, string) {
	if fullName := os.Getenv("BITBUCKET_REPO_FULL_NAME"); fullName != "" {
		parts := strings.SplitN(fullName, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return firstNonEmpty(os.Getenv("BITBUCKET_WORKSPACE"), os.Getenv("BITBUCKET_REPO_OWNER")), os.Getenv("BITBUCKET_REPO_SLUG")
}

func getBitbucketAPIBaseURL() string {
	if apiURL := os.Getenv("BYTEBASE_BITBUCKET_API_BASE_URL"); apiURL != "" {
		return apiURL
	}
	return bitbucket.APIBaseURL
}

func buildBitbucketPullRequestURL(apiURL, workspace, repoSlug, pullRequestID string) (string, error) {
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/repositories/" + url.PathEscape(workspace) + "/" + url.PathEscape(repoSlug) + "/pullrequests/" + url.PathEscape(pullRequestID)
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""
	return parsedURL.String(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func isGitLabBotUser(userName string) bool {
	lowerUserName := strings.ToLower(userName)
	return isBotUser("", lowerUserName) || gitLabBotUserNameRE.MatchString(lowerUserName)
}

func isBitbucketBotUser(userType, userName string) bool {
	return isBotUser(userType, userName) || strings.EqualFold(userType, "app") || strings.EqualFold(userType, "app_user")
}

func isBotUser(userType, userName string) bool {
	return strings.EqualFold(userType, "bot") || strings.HasSuffix(strings.ToLower(userName), "[bot]")
}
