package command

import (
	"context"
	"encoding/json"
	"fmt"
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

// readGitHubEvent decodes the payload of the event that triggered the
// workflow, which GitHub writes to the file GITHUB_EVENT_PATH names.
func readGitHubEvent(event any) bool {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return false
	}
	data, err := os.ReadFile(eventPath)
	if err != nil {
		return false
	}
	return json.Unmarshal(data, event) == nil
}

var gitLabBotUserNameRE = regexp.MustCompile(`^(project|group)_\d+_bot_[a-z0-9]+$`)

func getVCSUser(platform world.JobPlatform) *v1pb.VCSUser {
	switch platform {
	case world.GitHub:
		return getGitHubVCSUser()
	case world.GitLab:
		return getGitLabVCSUser()
	case world.Bitbucket:
		return getBitbucketVCSUser()
	case world.AzureDevOps:
		return getAzureDevOpsVCSUser()
	default:
		return nil
	}
}

func getGitHubVCSUser() *v1pb.VCSUser {
	eventName := os.Getenv("GITHUB_EVENT_NAME")
	if eventName != "pull_request" && eventName != "pull_request_target" {
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
	if !readGitHubEvent(&event) {
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

func getAzureDevOpsVCSUser() *v1pb.VCSUser {
	pullRequestID := os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTID")
	collectionURI := firstNonEmpty(os.Getenv("SYSTEM_COLLECTIONURI"), os.Getenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI"))
	accessToken := os.Getenv("SYSTEM_ACCESSTOKEN")
	if pullRequestID == "" || collectionURI == "" || accessToken == "" {
		return nil
	}

	requestURL, err := buildAzureDevOpsPullRequestURL(collectionURI, pullRequestID)
	if err != nil {
		slog.Warn("failed to build Azure DevOps pull request URL for VCS user attribution", "error", err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		slog.Warn("failed to create Azure DevOps pull request request for VCS user attribution", "error", err)
		return nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("failed to get Azure DevOps pull request for VCS user attribution", "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("failed to get Azure DevOps pull request for VCS user attribution", "status", resp.Status)
		return nil
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		slog.Warn("failed to read Azure DevOps pull request for VCS user attribution", "error", err)
		return nil
	}
	var pullRequest struct {
		CreatedBy struct {
			ID          string `json:"id"`
			UniqueName  string `json:"uniqueName"`
			DisplayName string `json:"displayName"`
			Descriptor  string `json:"descriptor"`
		} `json:"createdBy"`
	}
	if err := json.Unmarshal(data, &pullRequest); err != nil {
		slog.Warn("failed to parse Azure DevOps pull request for VCS user attribution", "error", err)
		return nil
	}

	creator := pullRequest.CreatedBy
	if creator.ID == "" || isAzureDevOpsBotUser(creator.Descriptor, creator.UniqueName) {
		return nil
	}
	return &v1pb.VCSUser{
		VcsType:     v1pb.VCSType_AZURE_DEVOPS,
		UserId:      creator.ID,
		UserName:    creator.UniqueName,
		DisplayName: creator.DisplayName,
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

func buildAzureDevOpsPullRequestURL(collectionURI, pullRequestID string) (string, error) {
	parsedURL, err := url.Parse(collectionURI)
	if err != nil {
		return "", err
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/_apis/git/pullrequests/" + url.PathEscape(pullRequestID)
	parsedURL.RawQuery = url.Values{"api-version": {"7.1"}}.Encode()
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

func isAzureDevOpsBotUser(descriptor, userName string) bool {
	lowerDescriptor := strings.ToLower(descriptor)
	return strings.HasPrefix(lowerDescriptor, "svc.") || strings.HasPrefix(lowerDescriptor, "aadsp.") || isBotUser("", userName)
}

func isBotUser(userType, userName string) bool {
	return strings.EqualFold(userType, "bot") || strings.HasSuffix(strings.ToLower(userName), "[bot]")
}

// The markers a platform writes into the commit that merged a pull request.
// A rebase merge writes none, so its source has no pull request.
var (
	gitHubSquashMergeRE  = regexp.MustCompile(`^(.*?)\s*\(#(\d+)\)$`)
	gitHubMergeCommitRE  = regexp.MustCompile(`^Merge pull request #(\d+) from `)
	gitLabMergeRequestRE = regexp.MustCompile(`(?m)^See merge request \S+!(\d+)\s*$`)
	azureMergedPRRE      = regexp.MustCompile(`^Merged PR (\d+): (.*)$`)
)

// getVCSSource describes the commit the release was read from, from the CI
// environment and the --source-* flags. A flag wins over the detected value.
// It reads only environment variables and, on GitHub, the event payload: the
// action image has no git binary.
func getVCSSource(w *world.World) *v1pb.Release_VCSSource {
	var source *v1pb.Release_VCSSource
	switch w.Platform {
	case world.GitHub:
		source = getGitHubVCSSource()
	case world.GitLab:
		source = getGitLabVCSSource()
	case world.Bitbucket:
		source = getBitbucketVCSSource()
	case world.AzureDevOps:
		source = getAzureDevOpsVCSSource()
	default:
	}
	if w.SourceTitle == "" && w.SourceURL == "" && w.SourceAuthor == "" && w.SourceRef == "" {
		return source
	}
	if source == nil {
		source = &v1pb.Release_VCSSource{}
	}
	// The Source row shows a pull request's title and link over the commit's,
	// so a flag replaces both. A flag URL is another commit: the detected SHA
	// no longer matches it.
	if w.SourceTitle != "" {
		source.CommitTitle = w.SourceTitle
		if source.PullRequest != nil {
			source.PullRequest.Title = w.SourceTitle
		}
	}
	if w.SourceURL != "" {
		source.Url = w.SourceURL
		source.CommitSha = ""
		if source.PullRequest != nil {
			source.PullRequest.Url = w.SourceURL
		}
	}
	if w.SourceAuthor != "" {
		source.AuthorName = w.SourceAuthor
	}
	if w.SourceRef != "" {
		source.Branch = w.SourceRef
	}
	return source
}

func getGitHubVCSSource() *v1pb.Release_VCSSource {
	repoURL := os.Getenv("GITHUB_SERVER_URL") + "/" + os.Getenv("GITHUB_REPOSITORY")
	sha := os.Getenv("GITHUB_SHA")
	source := &v1pb.Release_VCSSource{
		VcsType:   v1pb.VCSType_GITHUB,
		Url:       repoURL + "/commit/" + sha,
		CommitSha: sha,
		// GITHUB_REF_NAME is the synthetic "<N>/merge" in a pull request run.
		Branch: firstNonEmpty(os.Getenv("GITHUB_HEAD_REF"), os.Getenv("GITHUB_REF_NAME")),
	}

	var event struct {
		PullRequest *struct {
			Number  int64  `json:"number"`
			Title   string `json:"title"`
			HTMLURL string `json:"html_url"`
			User    struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"pull_request"`
		HeadCommit *struct {
			Message string `json:"message"`
			Author  struct {
				Name     string `json:"name"`
				Username string `json:"username"`
			} `json:"author"`
		} `json:"head_commit"`
	}
	if !readGitHubEvent(&event) {
		return source
	}
	if pr := event.PullRequest; pr != nil && pr.Number > 0 {
		source.AuthorLogin = pr.User.Login
		source.PullRequest = &v1pb.Release_VCSSource_PullRequest{
			Url:    firstNonEmpty(pr.HTMLURL, fmt.Sprintf("%s/pull/%d", repoURL, pr.Number)),
			Number: pr.Number,
			Title:  pr.Title,
		}
		return source
	}
	if commit := event.HeadCommit; commit != nil {
		source.AuthorName = commit.Author.Name
		source.AuthorLogin = commit.Author.Username
		title, body := splitCommitMessage(commit.Message)
		source.CommitTitle = title
		if m := gitHubSquashMergeRE.FindStringSubmatch(title); m != nil {
			source.CommitTitle = m[1]
			source.PullRequest = gitHubPullRequest(repoURL, m[2], "")
		} else if m := gitHubMergeCommitRE.FindStringSubmatch(title); m != nil {
			// GitHub puts the pull request's title on the merge commit's body.
			source.PullRequest = gitHubPullRequest(repoURL, m[1], firstLine(body))
		}
	}
	return source
}

func gitHubPullRequest(repoURL, number, title string) *v1pb.Release_VCSSource_PullRequest {
	n, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return nil
	}
	return &v1pb.Release_VCSSource_PullRequest{Url: fmt.Sprintf("%s/pull/%d", repoURL, n), Number: n, Title: title}
}

func getGitLabVCSSource() *v1pb.Release_VCSSource {
	repoURL := os.Getenv("CI_PROJECT_URL")
	sha := os.Getenv("CI_COMMIT_SHA")
	source := &v1pb.Release_VCSSource{
		VcsType:     v1pb.VCSType_GITLAB,
		Url:         repoURL + "/-/commit/" + sha,
		CommitSha:   sha,
		Branch:      firstNonEmpty(os.Getenv("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME"), os.Getenv("CI_COMMIT_REF_NAME")),
		CommitTitle: os.Getenv("CI_COMMIT_TITLE"),
		AuthorName:  authorNameOf(os.Getenv("CI_COMMIT_AUTHOR")),
	}
	number, title := os.Getenv("CI_MERGE_REQUEST_IID"), os.Getenv("CI_MERGE_REQUEST_TITLE")
	if number == "" {
		if m := gitLabMergeRequestRE.FindStringSubmatch(os.Getenv("CI_COMMIT_MESSAGE")); m != nil {
			number = m[1]
		}
	}
	if n, err := strconv.ParseInt(number, 10, 64); err == nil {
		source.PullRequest = &v1pb.Release_VCSSource_PullRequest{
			Url:    fmt.Sprintf("%s/-/merge_requests/%d", repoURL, n),
			Number: n,
			Title:  title,
		}
	}
	return source
}

// Bitbucket Pipelines exposes neither the commit message nor its author.
func getBitbucketVCSSource() *v1pb.Release_VCSSource {
	repoURL := os.Getenv("BITBUCKET_GIT_HTTP_ORIGIN")
	sha := os.Getenv("BITBUCKET_COMMIT")
	source := &v1pb.Release_VCSSource{
		VcsType:   v1pb.VCSType_BITBUCKET,
		Url:       repoURL + "/commits/" + sha,
		CommitSha: sha,
		Branch:    os.Getenv("BITBUCKET_BRANCH"),
	}
	if n, err := strconv.ParseInt(os.Getenv("BITBUCKET_PR_ID"), 10, 64); err == nil {
		source.PullRequest = &v1pb.Release_VCSSource_PullRequest{
			Url:    fmt.Sprintf("%s/pull-requests/%d", repoURL, n),
			Number: n,
		}
	}
	return source
}

func getAzureDevOpsVCSSource() *v1pb.Release_VCSSource {
	repoURL := os.Getenv("SYSTEM_COLLECTIONURI") + os.Getenv("SYSTEM_TEAMPROJECT") + "/_git/" + os.Getenv("BUILD_REPOSITORY_NAME")
	sha := os.Getenv("BUILD_SOURCEVERSION")
	// BUILD_SOURCEBRANCH is "refs/pull/<N>/merge" in a PR build, and
	// BUILD_SOURCEBRANCHNAME keeps only the last segment of "feature/x".
	branch := firstNonEmpty(os.Getenv("SYSTEM_PULLREQUEST_SOURCEBRANCH"), os.Getenv("BUILD_SOURCEBRANCH"))
	title, _ := splitCommitMessage(os.Getenv("BUILD_SOURCEVERSIONMESSAGE"))
	source := &v1pb.Release_VCSSource{
		VcsType:     v1pb.VCSType_AZURE_DEVOPS,
		Url:         repoURL + "/commit/" + sha,
		CommitSha:   sha,
		Branch:      strings.TrimPrefix(branch, "refs/heads/"),
		CommitTitle: title,
		AuthorName:  os.Getenv("BUILD_REQUESTEDFOR"),
	}
	number := os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTID")
	if number == "" {
		if m := azureMergedPRRE.FindStringSubmatch(title); m != nil {
			number = m[1]
			source.CommitTitle = m[2]
		}
	}
	if n, err := strconv.ParseInt(number, 10, 64); err == nil {
		source.PullRequest = &v1pb.Release_VCSSource_PullRequest{
			Url:    fmt.Sprintf("%s/pullrequest/%d", repoURL, n),
			Number: n,
		}
	}
	return source
}

// splitCommitMessage returns the message's first line and the rest.
func splitCommitMessage(message string) (string, string) {
	title, body, _ := strings.Cut(strings.TrimSpace(message), "\n")
	return strings.TrimSpace(title), strings.TrimSpace(body)
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(line)
}

// authorNameOf drops the email from git's "Name <email>" form.
func authorNameOf(author string) string {
	name, _, _ := strings.Cut(author, " <")
	return strings.TrimSpace(name)
}
