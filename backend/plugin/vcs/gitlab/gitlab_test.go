package gitlab

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
)

func TestProvider_FetchAllRepositoryList(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v4/projects", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.gitlab.com/ee/api/projects.html#list-all-projects
			Body: io.NopCloser(strings.NewReader(`
[
  {
    "id": 4,
    "description": null,
    "default_branch": "master",
    "ssh_url_to_repo": "git@example.com:diaspora/diaspora-client.git",
    "http_url_to_repo": "http://example.com/diaspora/diaspora-client.git",
    "web_url": "http://example.com/diaspora/diaspora-client",
    "readme_url": "http://example.com/diaspora/diaspora-client/blob/master/README.md",
    "tag_list": [
      "example",
      "disapora client"
    ],
    "topics": [
      "example",
      "disapora client"
    ],
    "name": "Diaspora Client",
    "name_with_namespace": "Diaspora / Diaspora Client",
    "path": "diaspora-client",
    "path_with_namespace": "diaspora/diaspora-client",
    "created_at": "2013-09-30T13:46:02Z",
    "last_activity_at": "2013-09-30T13:46:02Z",
    "forks_count": 0,
    "avatar_url": "http://example.com/uploads/project/avatar/4/uploads/avatar.png",
    "star_count": 0
  }
]
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.FetchAllRepositoryList(ctx, &common.OauthContext{}, "")
	require.NoError(t, err)

	want := []*vcs.Repository{
		{
			ID:       "4",
			Name:     "Diaspora Client",
			FullPath: "diaspora/diaspora-client",
			WebURL:   "http://example.com/diaspora/diaspora-client",
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchRepositoryFileList(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v4/projects/1/repository/tree", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.gitlab.com/ee/api/repositories.html#list-repository-tree
			Body: io.NopCloser(strings.NewReader(`
[
  {
    "id": "a1e8f8d745cc87e3a9248358d9352bb7f9a0aeba",
    "name": "html",
    "type": "tree",
    "path": "files/html",
    "mode": "040000"
  },
  {
    "id": "7d70e02340bac451f281cecf0a980907974bd8be",
    "name": "whitespace",
    "type": "blob",
    "path": "files/whitespace",
    "mode": "100644"
  }
]
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.FetchRepositoryFileList(ctx, &common.OauthContext{}, "", "1", "main", "")
	require.NoError(t, err)

	// Non-blob type should excluded
	want := []*vcs.RepositoryTreeNode{
		{
			Path: "files/whitespace",
			Type: "blob",
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_ReadFileMeta(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v4/projects/1/repository/files/app%2Fmodels%2Fkey.rb/raw", r.URL.RawPath)
		header := http.Header{}
		header.Set("x-gitlab-file-name", "key.rb")
		header.Set("x-gitlab-file-path", "app/models/key.rb")
		header.Set("x-gitlab-size", "3")
		header.Set("x-gitlab-last-commit-id", "27329d3afac51fbf2762428e12f2635d1137c549")
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response derived from https://docs.gitlab.com/ee/api/repository_files.html#get-file-from-repository
			Body:   io.NopCloser(strings.NewReader(`key`)),
			Header: header,
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.ReadFileMeta(ctx, &common.OauthContext{}, "", "1", "app/models/key.rb", vcs.RefInfo{
		RefType: vcs.RefTypeBranch,
		RefName: "master",
	})
	require.NoError(t, err)

	want := &vcs.FileMeta{
		LastCommitID: "27329d3afac51fbf2762428e12f2635d1137c549",
	}
	assert.Equal(t, want, got)
}

func TestProvider_ReadFileContent(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v4/projects/1/repository/files/app%2Fmodels%2Fkey.rb/raw", r.URL.RawPath)
		header := http.Header{}
		header.Set("x-gitlab-file-name", "key.rb")
		header.Set("x-gitlab-file-path", "app/models/key.rb")
		header.Set("x-gitlab-size", "3")
		header.Set("x-gitlab-last-commit-id", "27329d3afac51fbf2762428e12f2635d1137c549")
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response derived from https://docs.gitlab.com/ee/api/repository_files.html#get-file-from-repository
			Body: io.NopCloser(strings.NewReader(`# Sample GitLab Project

This sample project shows how a project in GitLab looks for demonstration purposes. It contains issues, merge requests and Markdown files in many branches,
named and filled with lorem ipsum.

You can look around to get an idea how to structure your project and, when done, you can safely delete this project.

[Learn more about creating GitLab projects.](https://docs.gitlab.com/ee/gitlab-basics/create-project.html)
`)),
			Header: header,
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.ReadFileContent(ctx, &common.OauthContext{}, "", "1", "app/models/key.rb", vcs.RefInfo{
		RefType: vcs.RefTypeBranch,
		RefName: "master",
	})
	require.NoError(t, err)

	want := `# Sample GitLab Project

This sample project shows how a project in GitLab looks for demonstration purposes. It contains issues, merge requests and Markdown files in many branches,
named and filled with lorem ipsum.

You can look around to get an idea how to structure your project and, when done, you can safely delete this project.

[Learn more about creating GitLab projects.](https://docs.gitlab.com/ee/gitlab-basics/create-project.html)
`
	assert.Equal(t, want, got)
}

func TestProvider_CreateWebhook(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v4/projects/1/hooks", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusCreated,
			// Example response taken from https://docs.gitlab.com/ee/api/projects.html#get-project-hook
			Body: io.NopCloser(strings.NewReader(`
{
  "id": 1,
  "url": "http://example.com/hook",
  "project_id": 3,
  "push_events": true,
  "push_events_branch_filter": "",
  "issues_events": true,
  "confidential_issues_events": true,
  "merge_requests_events": true,
  "tag_push_events": true,
  "note_events": true,
  "confidential_note_events": true,
  "job_events": true,
  "pipeline_events": true,
  "wiki_page_events": true,
  "deployment_events": true,
  "releases_events": true,
  "enable_ssl_verification": true,
  "created_at": "2012-10-12T17:04:47Z"
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.CreateWebhook(ctx, &common.OauthContext{}, "", "1", []byte(""))
	require.NoError(t, err)
	assert.Equal(t, "1", got)
}

func TestProvider_DeleteWebhook(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v4/projects/1/hooks/1", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	},
	)

	ctx := context.Background()
	err := p.DeleteWebhook(ctx, &common.OauthContext{}, "", "1", "1")
	require.NoError(t, err)
}

func TestProvider_GetBranch(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v4/projects/1/repository/branches/main", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.gitlab.com/ee/api/branches.html#get-single-repository-branch
			Body: io.NopCloser(strings.NewReader(`
{
  "name": "main",
  "merged": false,
  "protected": true,
  "default": true,
  "developers_can_push": false,
  "developers_can_merge": false,
  "can_push": true,
  "web_url": "https://gitlab.example.com/my-group/my-project/-/tree/main",
  "commit": {
    "author_email": "john@example.com",
    "author_name": "John Smith",
    "authored_date": "2012-06-27T05:51:39-07:00",
    "committed_date": "2012-06-28T03:44:20-07:00",
    "committer_email": "john@example.com",
    "committer_name": "John Smith",
    "id": "7b5c3cc8be40ee161ae89a06bba6229da1032a0c",
    "short_id": "7b5c3cc",
    "title": "add projects API",
    "message": "add projects API",
    "parent_ids": [
      "4ad91d3c1144c406e50c7b33bae684bd6837faf8"
    ]
  }
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.GetBranch(ctx, &common.OauthContext{}, "", "1", "main")
	require.NoError(t, err)

	want := &vcs.BranchInfo{
		Name:         "main",
		LastCommitID: "7b5c3cc8be40ee161ae89a06bba6229da1032a0c",
	}
	assert.Equal(t, want, got)
}

func TestProvider_ListPullRequestFile(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v4/projects/1/merge_requests/1/changes", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.gitlab.com/ee/api/merge_requests.html#get-single-mr-changes
			Body: io.NopCloser(strings.NewReader(`
{
  "id": 21,
  "iid": 1,
  "project_id": 4,
  "title": "Blanditiis beatae suscipit hic assumenda et molestias nisi asperiores repellat et.",
  "state": "reopened",
  "created_at": "2015-02-02T19:49:39.159Z",
  "updated_at": "2015-02-02T20:08:49.959Z",
  "target_branch": "secret_token",
  "source_branch": "version-1-9",
  "source_project_id": 4,
  "target_project_id": 4,
  "labels": [ ],
  "description": "Qui voluptatibus placeat ipsa alias quasi. Deleniti rem ut sint. Optio velit qui distinctio.",
  "draft": false,
  "work_in_progress": false,
  "merge_when_pipeline_succeeds": true,
  "merge_status": "can_be_merged",
  "subscribed" : true,
  "sha": "8888888888888888888888888888888888888888",
  "merge_commit_sha": null,
  "squash_commit_sha": null,
  "changes": [
    {
    "old_path": "VERSION",
    "new_path": "VERSION",
    "a_mode": "100644",
    "b_mode": "100644",
    "new_file": false,
    "renamed_file": false,
    "deleted_file": false
    }
  ],
  "overflow": false
}
`)),
		}, nil
	},
	)
	ctx := context.Background()
	got, err := p.ListPullRequestFile(ctx, &common.OauthContext{}, "", "1", "1")
	require.NoError(t, err)

	want := []*vcs.PullRequestFile{
		{
			Path:         "VERSION",
			LastCommitID: "8888888888888888888888888888888888888888",
			IsDeleted:    false,
		},
	}
	assert.Equal(t, want, got)
}

func newMockProvider(mockRoundTrip func(r *http.Request) (*http.Response, error)) vcs.Provider {
	return newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: mockRoundTrip,
				},
			},
		},
	)
}
