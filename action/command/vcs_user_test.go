package command

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/bitbucket"
	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestGetVCSUserFromGitHubPullRequest(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	require.NoError(t, os.WriteFile(eventPath, []byte(`{"pull_request":{"user":{"id":1001,"login":"alice","name":"Alice","type":"User"}}}`), 0600))
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	user := getVCSUser(world.GitHub)
	require.NotNil(t, user)
	require.Equal(t, v1pb.VCSType_GITHUB, user.VcsType)
	require.Equal(t, "1001", user.UserId)
	require.Equal(t, "alice", user.UserName)
	require.Equal(t, "Alice", user.DisplayName)
}

func TestGetVCSUserSkipsGitHubBot(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	require.NoError(t, os.WriteFile(eventPath, []byte(`{"pull_request":{"user":{"id":41898282,"login":"github-actions[bot]","type":"Bot"}}}`), 0600))
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	require.Nil(t, getVCSUser(world.GitHub))
}

func TestGetVCSUserFromGitHubPullRequestTarget(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	require.NoError(t, os.WriteFile(eventPath, []byte(`{"pull_request":{"user":{"id":1001,"login":"alice","name":"Alice","type":"User"}}}`), 0600))
	t.Setenv("GITHUB_EVENT_NAME", "pull_request_target")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	user := getVCSUser(world.GitHub)
	require.NotNil(t, user)
	require.Equal(t, v1pb.VCSType_GITHUB, user.VcsType)
	require.Equal(t, "1001", user.UserId)
}

func TestGetVCSUserSkipsGitLabWhenAuthorUnavailable(t *testing.T) {
	t.Setenv("CI_PIPELINE_SOURCE", "merge_request_event")
	t.Setenv("GITLAB_USER_ID", "2002")
	t.Setenv("GITLAB_USER_LOGIN", "bob")
	t.Setenv("GITLAB_USER_NAME", "Bob")

	require.Nil(t, getVCSUser(world.GitLab))
}

func TestGetVCSUserFromGitLabMergeRequestAuthor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v4/projects/987/merge_requests/42", r.URL.Path)
		require.Equal(t, "job-token", r.Header.Get("JOB-TOKEN"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"author":{"id":3003,"username":"alice","name":"Alice"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("CI_MERGE_REQUEST_PROJECT_ID", "987")
	t.Setenv("CI_MERGE_REQUEST_IID", "42")
	t.Setenv("CI_API_V4_URL", server.URL+"/api/v4")
	t.Setenv("CI_JOB_TOKEN", "job-token")
	t.Setenv("GITLAB_USER_ID", "2002")
	t.Setenv("GITLAB_USER_LOGIN", "triggerer")
	t.Setenv("GITLAB_USER_NAME", "Pipeline Triggerer")

	user := getVCSUser(world.GitLab)
	require.NotNil(t, user)
	require.Equal(t, v1pb.VCSType_GITLAB, user.VcsType)
	require.Equal(t, "3003", user.UserId)
	require.Equal(t, "alice", user.UserName)
	require.Equal(t, "Alice", user.DisplayName)
}

func TestGetVCSUserSkipsGitLabBotAuthor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"author":{"id":4004,"username":"project_987_bot_a1b2c3","name":"Project bot"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("CI_MERGE_REQUEST_PROJECT_ID", "987")
	t.Setenv("CI_MERGE_REQUEST_IID", "42")
	t.Setenv("CI_API_V4_URL", server.URL+"/api/v4")
	t.Setenv("CI_JOB_TOKEN", "job-token")

	require.Nil(t, getVCSUser(world.GitLab))
}

func TestGetVCSUserSkipsBitbucketWhenAuthorUnavailable(t *testing.T) {
	t.Setenv("BITBUCKET_PR_ID", "10")
	t.Setenv("BITBUCKET_STEP_TRIGGERER_UUID", "{3003}")
	t.Setenv("BITBUCKET_STEP_TRIGGERER_USERNAME", "carol")

	require.Nil(t, getVCSUser(world.Bitbucket))
}

func TestGetVCSUserFromBitbucketPullRequestAuthor(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"author":{"account_id":"557058:author-account","uuid":"{author-uuid}","nickname":"alice","display_name":"Alice","type":"user"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("BITBUCKET_PR_ID", "10")
	t.Setenv("BITBUCKET_REPO_FULL_NAME", "bytebase/example")
	t.Setenv("BITBUCKET_STEP_TRIGGERER_UUID", "{triggerer-uuid}")
	t.Setenv("BITBUCKET_STEP_TRIGGERER_USERNAME", "triggerer")
	t.Setenv("BYTEBASE_BITBUCKET_API_BASE_URL", server.URL)

	user := getVCSUser(world.Bitbucket)
	require.NotNil(t, user)
	require.Equal(t, "/repositories/bytebase/example/pullrequests/10", gotPath)
	require.Equal(t, v1pb.VCSType_BITBUCKET, user.VcsType)
	require.Equal(t, "557058:author-account", user.UserId)
	require.Equal(t, "alice", user.UserName)
	require.Equal(t, "Alice", user.DisplayName)
}

func TestBitbucketDefaultAPIUsesPipelinesProxy(t *testing.T) {
	requestURL, err := buildBitbucketPullRequestURL(getBitbucketAPIBaseURL(), "bytebase", "example", "10")
	require.NoError(t, err)
	require.Equal(t, "http://api.bitbucket.org/2.0/repositories/bytebase/example/pullrequests/10", requestURL)

	client := bitbucket.NewHTTPClient(getBitbucketAPIBaseURL())
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	proxyURL, err := transport.Proxy(&http.Request{URL: mustParseURL(t, requestURL)})
	require.NoError(t, err)
	require.Equal(t, "http://localhost:29418", proxyURL.String())
}

func TestGetVCSUserSkipsBitbucketBotAuthor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"author":{"account_id":"557058:bot-account","nickname":"release[bot]","type":"bot"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("BITBUCKET_PR_ID", "10")
	t.Setenv("BITBUCKET_REPO_FULL_NAME", "bytebase/example")
	t.Setenv("BYTEBASE_BITBUCKET_API_BASE_URL", server.URL)

	require.Nil(t, getVCSUser(world.Bitbucket))
}

func TestGetVCSUserSkipsBitbucketAppUserAuthor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"author":{"account_id":"557058:app-account","nickname":"release-app","type":"app_user"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("BITBUCKET_PR_ID", "10")
	t.Setenv("BITBUCKET_REPO_FULL_NAME", "bytebase/example")
	t.Setenv("BYTEBASE_BITBUCKET_API_BASE_URL", server.URL)

	require.Nil(t, getVCSUser(world.Bitbucket))
}

func TestGetVCSUserSkipsUnsupportedPlatform(t *testing.T) {
	require.Nil(t, getVCSUser(world.LocalPlatform))
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsedURL, err := url.Parse(rawURL)
	require.NoError(t, err)
	return parsedURL
}
