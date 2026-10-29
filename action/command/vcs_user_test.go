package command

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

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

func TestGetVCSUserFromAzureDevOpsPullRequestCreator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/collection/_apis/git/pullrequests/42", r.URL.Path)
		require.Equal(t, "7.1", r.URL.Query().Get("api-version"))
		require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"createdBy":{"id":"author-id","uniqueName":"alice@example.com","displayName":"Alice"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "42")
	t.Setenv("SYSTEM_COLLECTIONURI", server.URL+"/collection/")
	t.Setenv("SYSTEM_ACCESSTOKEN", "access-token")

	user := getVCSUser(world.AzureDevOps)
	require.NotNil(t, user)
	require.Equal(t, v1pb.VCSType_AZURE_DEVOPS, user.VcsType)
	require.Equal(t, "author-id", user.UserId)
	require.Equal(t, "alice@example.com", user.UserName)
	require.Equal(t, "Alice", user.DisplayName)
}

func TestGetVCSUserFromAzureDevOpsPullRequestCreatorWithTeamFoundationCollectionURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collection/_apis/git/pullrequests/42", r.URL.Path)
		_, err := w.Write([]byte(`{"createdBy":{"id":"author-id","uniqueName":"alice@example.com","displayName":"Alice"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "42")
	t.Setenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI", server.URL+"/collection/")
	t.Setenv("SYSTEM_ACCESSTOKEN", "access-token")

	require.NotNil(t, getVCSUser(world.AzureDevOps))
}

func TestGetVCSUserSkipsAzureDevOpsWhenPullRequestMetadataUnavailable(t *testing.T) {
	t.Setenv("SYSTEM_COLLECTIONURI", "https://dev.azure.com/bytebase/")
	t.Setenv("SYSTEM_ACCESSTOKEN", "access-token")
	t.Setenv("BUILD_REQUESTEDFORID", "requester-id")
	t.Setenv("BUILD_REQUESTEDFOR", "Requester")

	require.Nil(t, getVCSUser(world.AzureDevOps))
}

func TestGetVCSUserSkipsAzureDevOpsWhenAccessTokenUnavailable(t *testing.T) {
	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "42")
	t.Setenv("SYSTEM_COLLECTIONURI", "https://dev.azure.com/bytebase/")

	require.Nil(t, getVCSUser(world.AzureDevOps))
}

func TestGetVCSUserSkipsAzureDevOpsServicePrincipalCreator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"createdBy":{"id":"bot-id","uniqueName":"build-service","displayName":"Build Service","descriptor":"aadsp.service-principal"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "42")
	t.Setenv("SYSTEM_COLLECTIONURI", server.URL)
	t.Setenv("SYSTEM_ACCESSTOKEN", "access-token")

	require.Nil(t, getVCSUser(world.AzureDevOps))
}

func TestGetVCSUserSkipsAzureDevOpsBuildServiceCreator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"createdBy":{"id":"bot-id","uniqueName":"build-service","displayName":"Build Service","descriptor":"svc.build-service"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "42")
	t.Setenv("SYSTEM_COLLECTIONURI", server.URL)
	t.Setenv("SYSTEM_ACCESSTOKEN", "access-token")

	require.Nil(t, getVCSUser(world.AzureDevOps))
}

func TestGetVCSUserSkipsAzureDevOpsBotCreator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"createdBy":{"id":"bot-id","uniqueName":"release[bot]","displayName":"Release Bot"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "42")
	t.Setenv("SYSTEM_COLLECTIONURI", server.URL)
	t.Setenv("SYSTEM_ACCESSTOKEN", "access-token")

	require.Nil(t, getVCSUser(world.AzureDevOps))
}

func TestGetVCSUserSkipsAzureDevOpsCreatorWithoutID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"createdBy":{"uniqueName":"alice@example.com","displayName":"Alice"}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "42")
	t.Setenv("SYSTEM_COLLECTIONURI", server.URL)
	t.Setenv("SYSTEM_ACCESSTOKEN", "access-token")

	require.Nil(t, getVCSUser(world.AzureDevOps))
}

func TestGetVCSUserSkipsAzureDevOpsInvalidPullRequestResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`invalid`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "42")
	t.Setenv("SYSTEM_COLLECTIONURI", server.URL)
	t.Setenv("SYSTEM_ACCESSTOKEN", "access-token")

	require.Nil(t, getVCSUser(world.AzureDevOps))
}

func TestGetVCSUserSkipsAzureDevOpsNonOKPullRequestResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "42")
	t.Setenv("SYSTEM_COLLECTIONURI", server.URL)
	t.Setenv("SYSTEM_ACCESSTOKEN", "access-token")

	require.Nil(t, getVCSUser(world.AzureDevOps))
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

func TestGetVCSSource(t *testing.T) {
	githubPush := map[string]string{
		"GITHUB_SERVER_URL": "https://github.com",
		"GITHUB_REPOSITORY": "example/app",
		"GITHUB_SHA":        "0a1b2c3d",
		"GITHUB_EVENT_NAME": "push",
		"GITHUB_REF_NAME":   "main",
		githubEventPayload:  "",
	}
	with := func(base map[string]string, extra map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range base {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	tests := []struct {
		name     string
		platform world.JobPlatform
		env      map[string]string
		flags    func(w *world.World)
		want     *v1pb.Release_VCSSource
	}{
		{
			name:     "GitHub pull request run uses the source branch and the PR",
			platform: world.GitHub,
			env: with(githubPush, map[string]string{
				"GITHUB_EVENT_NAME": "pull_request",
				"GITHUB_REF_NAME":   "42/merge",
				"GITHUB_HEAD_REF":   "feature/orders",
				githubEventPayload:  `{"pull_request":{"number":42,"title":"Add the orders table","html_url":"https://github.com/example/app/pull/42","user":{"login":"alexdoe"}}}`,
			}),
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_GITHUB, Url: "https://github.com/example/app/commit/0a1b2c3d",
				CommitSha: "0a1b2c3d", Branch: "feature/orders", AuthorLogin: "alexdoe",
				PullRequest: &v1pb.Release_VCSSource_PullRequest{Url: "https://github.com/example/app/pull/42", Number: 42, Title: "Add the orders table"},
			},
		},
		{
			name:     "GitHub squash-merge push takes the PR number from the title once",
			platform: world.GitHub,
			env: with(githubPush, map[string]string{
				githubEventPayload: `{"head_commit":{"message":"Add the orders table (#42)\n\n* first commit","author":{"name":"Alex Doe","username":"alexdoe"}}}`,
			}),
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_GITHUB, Url: "https://github.com/example/app/commit/0a1b2c3d",
				CommitSha: "0a1b2c3d", Branch: "main", CommitTitle: "Add the orders table",
				AuthorName: "Alex Doe", AuthorLogin: "alexdoe",
				PullRequest: &v1pb.Release_VCSSource_PullRequest{Url: "https://github.com/example/app/pull/42", Number: 42},
			},
		},
		{
			name:     "GitHub merge-commit push takes the PR title from the message body",
			platform: world.GitHub,
			env: with(githubPush, map[string]string{
				githubEventPayload: `{"head_commit":{"message":"Merge pull request #42 from example/feature/orders\n\nAdd the orders table","author":{"name":"Alex Doe","username":"alexdoe"}}}`,
			}),
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_GITHUB, Url: "https://github.com/example/app/commit/0a1b2c3d",
				CommitSha: "0a1b2c3d", Branch: "main", CommitTitle: "Merge pull request #42 from example/feature/orders",
				AuthorName: "Alex Doe", AuthorLogin: "alexdoe",
				PullRequest: &v1pb.Release_VCSSource_PullRequest{Url: "https://github.com/example/app/pull/42", Number: 42, Title: "Add the orders table"},
			},
		},
		{
			name:     "GitHub rebase push has a commit title and no PR",
			platform: world.GitHub,
			env: with(githubPush, map[string]string{
				githubEventPayload: `{"head_commit":{"message":"Add the orders index\n\nbody","author":{"name":"Alex Doe","username":"alexdoe"}}}`,
			}),
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_GITHUB, Url: "https://github.com/example/app/commit/0a1b2c3d",
				CommitSha: "0a1b2c3d", Branch: "main", CommitTitle: "Add the orders index",
				AuthorName: "Alex Doe", AuthorLogin: "alexdoe",
			},
		},
		{
			name:     "GitLab merge request pipeline uses the source branch and the MR",
			platform: world.GitLab,
			env: map[string]string{
				"CI_PROJECT_URL":                      "https://gitlab.example.com/group/app",
				"CI_COMMIT_SHA":                       "9f8e7d6c",
				"CI_COMMIT_REF_NAME":                  "refs/merge-requests/7/head",
				"CI_MERGE_REQUEST_SOURCE_BRANCH_NAME": "feature/orders",
				"CI_COMMIT_TITLE":                     "Add the orders table",
				"CI_COMMIT_MESSAGE":                   "Add the orders table",
				"CI_COMMIT_AUTHOR":                    "Alex Doe <alex@example.com>",
				"CI_MERGE_REQUEST_IID":                "7",
				"CI_MERGE_REQUEST_TITLE":              "Orders table",
			},
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_GITLAB, Url: "https://gitlab.example.com/group/app/-/commit/9f8e7d6c",
				CommitSha: "9f8e7d6c", Branch: "feature/orders", CommitTitle: "Add the orders table", AuthorName: "Alex Doe",
				PullRequest: &v1pb.Release_VCSSource_PullRequest{Url: "https://gitlab.example.com/group/app/-/merge_requests/7", Number: 7, Title: "Orders table"},
			},
		},
		{
			name:     "GitLab push takes the MR number from the merge commit message",
			platform: world.GitLab,
			env: map[string]string{
				"CI_PROJECT_URL":     "https://gitlab.example.com/group/app",
				"CI_COMMIT_SHA":      "9f8e7d6c",
				"CI_COMMIT_REF_NAME": "main",
				"CI_COMMIT_TITLE":    "Merge branch 'feature/orders' into 'main'",
				"CI_COMMIT_MESSAGE":  "Merge branch 'feature/orders' into 'main'\n\nOrders table\n\nSee merge request group/app!7",
				"CI_COMMIT_AUTHOR":   "Alex Doe <alex@example.com>",
			},
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_GITLAB, Url: "https://gitlab.example.com/group/app/-/commit/9f8e7d6c",
				CommitSha: "9f8e7d6c", Branch: "main", CommitTitle: "Merge branch 'feature/orders' into 'main'", AuthorName: "Alex Doe",
				PullRequest: &v1pb.Release_VCSSource_PullRequest{Url: "https://gitlab.example.com/group/app/-/merge_requests/7", Number: 7},
			},
		},
		{
			name:     "Bitbucket push has no title or author",
			platform: world.Bitbucket,
			env: map[string]string{
				"BITBUCKET_GIT_HTTP_ORIGIN": "https://bitbucket.org/example/app",
				"BITBUCKET_COMMIT":          "5e6f7a8b",
				"BITBUCKET_BRANCH":          "main",
			},
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_BITBUCKET, Url: "https://bitbucket.org/example/app/commits/5e6f7a8b",
				CommitSha: "5e6f7a8b", Branch: "main",
			},
		},
		{
			name:     "Bitbucket pull request pipeline has the PR number",
			platform: world.Bitbucket,
			env: map[string]string{
				"BITBUCKET_GIT_HTTP_ORIGIN": "https://bitbucket.org/example/app",
				"BITBUCKET_COMMIT":          "5e6f7a8b",
				"BITBUCKET_BRANCH":          "feature/orders",
				"BITBUCKET_PR_ID":           "12",
			},
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_BITBUCKET, Url: "https://bitbucket.org/example/app/commits/5e6f7a8b",
				CommitSha: "5e6f7a8b", Branch: "feature/orders",
				PullRequest: &v1pb.Release_VCSSource_PullRequest{Url: "https://bitbucket.org/example/app/pull-requests/12", Number: 12},
			},
		},
		{
			name:     "Azure PR build keeps a slashed source branch whole",
			platform: world.AzureDevOps,
			env: map[string]string{
				"SYSTEM_COLLECTIONURI":             "https://dev.azure.com/example/",
				"SYSTEM_TEAMPROJECT":               "shop",
				"BUILD_REPOSITORY_NAME":            "app",
				"BUILD_SOURCEVERSION":              "1c2d3e4f",
				"BUILD_SOURCEBRANCH":               "refs/pull/31/merge",
				"BUILD_SOURCEBRANCHNAME":           "merge",
				"SYSTEM_PULLREQUEST_SOURCEBRANCH":  "refs/heads/feature/orders",
				"SYSTEM_PULLREQUEST_PULLREQUESTID": "31",
				"BUILD_SOURCEVERSIONMESSAGE":       "Merge pull request 31 from feature/orders into main",
				"BUILD_REQUESTEDFOR":               "Alex Doe",
			},
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_AZURE_DEVOPS, Url: "https://dev.azure.com/example/shop/_git/app/commit/1c2d3e4f",
				CommitSha: "1c2d3e4f", Branch: "feature/orders", CommitTitle: "Merge pull request 31 from feature/orders into main", AuthorName: "Alex Doe",
				PullRequest: &v1pb.Release_VCSSource_PullRequest{Url: "https://dev.azure.com/example/shop/_git/app/pullrequest/31", Number: 31},
			},
		},
		{
			name:     "Azure push takes the PR number from the completion message",
			platform: world.AzureDevOps,
			env: map[string]string{
				"SYSTEM_COLLECTIONURI":       "https://dev.azure.com/example/",
				"SYSTEM_TEAMPROJECT":         "shop",
				"BUILD_REPOSITORY_NAME":      "app",
				"BUILD_SOURCEVERSION":        "1c2d3e4f",
				"BUILD_SOURCEBRANCH":         "refs/heads/main",
				"BUILD_SOURCEVERSIONMESSAGE": "Merged PR 31: Add the orders table\n\nbody",
				"BUILD_REQUESTEDFOR":         "Alex Doe",
			},
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_AZURE_DEVOPS, Url: "https://dev.azure.com/example/shop/_git/app/commit/1c2d3e4f",
				CommitSha: "1c2d3e4f", Branch: "main", CommitTitle: "Add the orders table", AuthorName: "Alex Doe",
				PullRequest: &v1pb.Release_VCSSource_PullRequest{Url: "https://dev.azure.com/example/shop/_git/app/pullrequest/31", Number: 31},
			},
		},
		{
			name:     "undetected platform without flags has no source",
			platform: world.LocalPlatform,
			want:     nil,
		},
		{
			name:     "undetected platform takes what the flags pass",
			platform: world.LocalPlatform,
			flags: func(w *world.World) {
				w.SourceTitle = "Add the orders table"
				w.SourceURL = "https://git.example.com/app/commit/77aa"
				w.SourceAuthor = "Alex Doe"
				w.SourceRef = "main"
			},
			want: &v1pb.Release_VCSSource{
				Url: "https://git.example.com/app/commit/77aa", CommitTitle: "Add the orders table", AuthorName: "Alex Doe", Branch: "main",
			},
		},
		{
			name:     "flags win over detection",
			platform: world.GitHub,
			env: with(githubPush, map[string]string{
				githubEventPayload: `{"head_commit":{"message":"Add the orders index","author":{"name":"Alex Doe","username":"alexdoe"}}}`,
			}),
			flags: func(w *world.World) {
				w.SourceTitle = "Release train 12"
				w.SourceRef = "release/12"
				w.SourceAuthor = "Release Bot"
				w.SourceURL = "https://github.com/example/app/commit/ffff"
			},
			// The flag's commit URL replaces the detected commit, SHA included.
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_GITHUB, Url: "https://github.com/example/app/commit/ffff",
				Branch: "release/12", CommitTitle: "Release train 12",
				AuthorName: "Release Bot", AuthorLogin: "alexdoe",
			},
		},
		{
			name:     "flags win over a detected pull request",
			platform: world.GitHub,
			env: with(githubPush, map[string]string{
				"GITHUB_EVENT_NAME": "pull_request",
				githubEventPayload:  `{"pull_request":{"number":42,"title":"Add the orders table","html_url":"https://github.com/example/app/pull/42","user":{"login":"alexdoe"}}}`,
			}),
			flags: func(w *world.World) {
				w.SourceTitle = "Release train 12"
				w.SourceURL = "https://github.com/example/app/pull/99"
			},
			want: &v1pb.Release_VCSSource{
				VcsType: v1pb.VCSType_GITHUB, Url: "https://github.com/example/app/pull/99",
				Branch: "main", CommitTitle: "Release train 12", AuthorLogin: "alexdoe",
				PullRequest: &v1pb.Release_VCSSource_PullRequest{Url: "https://github.com/example/app/pull/99", Number: 42, Title: "Release train 12"},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range vcsSourceEnvKeys {
				t.Setenv(key, "")
			}
			for key, value := range tc.env {
				if key == githubEventPayload {
					if value != "" {
						eventPath := filepath.Join(t.TempDir(), "event.json")
						require.NoError(t, os.WriteFile(eventPath, []byte(value), 0600))
						t.Setenv("GITHUB_EVENT_PATH", eventPath)
					}
					continue
				}
				t.Setenv(key, value)
			}
			w := world.NewWorld()
			w.Platform = tc.platform
			if tc.flags != nil {
				tc.flags(w)
			}
			got := getVCSSource(w)
			require.True(t, proto.Equal(tc.want, got), "want %v\ngot  %v", tc.want, got)
		})
	}
}

// A case's GitHub event payload, written to the file GITHUB_EVENT_PATH names.
const githubEventPayload = "github event payload"

// Every variable getVCSSource reads, cleared per case so the runner's own CI
// environment cannot leak in.
var vcsSourceEnvKeys = []string{
	"GITHUB_SERVER_URL", "GITHUB_REPOSITORY", "GITHUB_SHA", "GITHUB_EVENT_NAME", "GITHUB_EVENT_PATH", "GITHUB_REF_NAME", "GITHUB_HEAD_REF",
	"CI_PROJECT_URL", "CI_COMMIT_SHA", "CI_COMMIT_REF_NAME", "CI_MERGE_REQUEST_SOURCE_BRANCH_NAME", "CI_COMMIT_TITLE", "CI_COMMIT_MESSAGE", "CI_COMMIT_AUTHOR", "CI_MERGE_REQUEST_IID", "CI_MERGE_REQUEST_TITLE",
	"BITBUCKET_GIT_HTTP_ORIGIN", "BITBUCKET_COMMIT", "BITBUCKET_BRANCH", "BITBUCKET_PR_ID",
	"SYSTEM_COLLECTIONURI", "SYSTEM_TEAMPROJECT", "BUILD_REPOSITORY_NAME", "BUILD_SOURCEVERSION", "BUILD_SOURCEBRANCH", "BUILD_SOURCEBRANCHNAME", "SYSTEM_PULLREQUEST_SOURCEBRANCH", "SYSTEM_PULLREQUEST_PULLREQUESTID", "BUILD_SOURCEVERSIONMESSAGE", "BUILD_REQUESTEDFOR",
}
