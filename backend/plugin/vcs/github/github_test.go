package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal/oauth"
)

func TestProvider_FetchCommitByID(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/repos/octocat/Hello-World/git/commits/7638417db6d59f3c431d3e1f261cc637155684cd", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/rest/git/commits#get-a-commit
			Body: io.NopCloser(strings.NewReader(`
{
  "sha": "7638417db6d59f3c431d3e1f261cc637155684cd",
  "node_id": "MDY6Q29tbWl0NmRjYjA5YjViNTc4NzVmMzM0ZjYxYWViZWQ2OTVlMmU0MTkzZGI1ZQ==",
  "url": "https://api.github.com/repos/octocat/Hello-World/git/commits/7638417db6d59f3c431d3e1f261cc637155684cd",
  "html_url": "https://github.com/octocat/Hello-World/commit/7638417db6d59f3c431d3e1f261cc637155684cd",
  "author": {
    "date": "2014-11-07T22:01:45Z",
    "name": "Monalisa Octocat",
    "email": "octocat@github.com"
  },
  "committer": {
    "date": "2014-11-07T22:01:45Z",
    "name": "Monalisa Octocat",
    "email": "octocat@github.com"
  },
  "message": "added readme, because im a good github citizen",
  "tree": {
    "url": "https://api.github.com/repos/octocat/Hello-World/git/trees/691272480426f78a0138979dd3ce63b77f706feb",
    "sha": "691272480426f78a0138979dd3ce63b77f706feb"
  },
  "parents": [
    {
      "url": "https://api.github.com/repos/octocat/Hello-World/git/commits/1acc419d4d6a9ce985db7be48c6349a0475975b5",
      "sha": "1acc419d4d6a9ce985db7be48c6349a0475975b5",
      "html_url": "https://github.com/octocat/Hello-World/commit/7638417db6d59f3c431d3e1f261cc637155684cd"
    }
  ],
  "verification": {
    "verified": false,
    "reason": "unsigned",
    "signature": null,
    "payload": null
  }
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.FetchCommitByID(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", "7638417db6d59f3c431d3e1f261cc637155684cd")
	require.NoError(t, err)

	want := &vcs.Commit{
		ID:         "7638417db6d59f3c431d3e1f261cc637155684cd",
		AuthorName: "Monalisa Octocat",
		CreatedTs:  1415397705,
	}
	assert.Equal(t, want, got)
}

func TestProvider_ExchangeOAuthToken(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/login/oauth/access_token", r.URL.Path)
		assert.Equal(t, "client_id=test_client_id&client_secret=test_client_secret&code=test_code&redirect_uri=http%3A%2F%2Flocalhost%3A3000", r.URL.RawQuery)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps#response
			Body: io.NopCloser(strings.NewReader(`
{
  "access_token":"gho_16C7e42F292c6912E7710c838347Ae178B4a",
  "scope":"repo,gist",
  "token_type":"bearer"
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.ExchangeOAuthToken(
		ctx,
		githubComURL,
		&common.OAuthExchange{
			ClientID:     "test_client_id",
			ClientSecret: "test_client_secret",
			Code:         "test_code",
			RedirectURL:  "http://localhost:3000",
		},
	)
	require.NoError(t, err)

	want := &vcs.OAuthToken{
		AccessToken: "gho_16C7e42F292c6912E7710c838347Ae178B4a",
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchAllRepositoryList(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/user/repos", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response derived from https://docs.github.com/en/rest/repos/repos#list-repositories-for-the-authenticated-user
			Body: io.NopCloser(strings.NewReader(`
[
  {
    "id": 1296269,
    "node_id": "MDEwOlJlcG9zaXRvcnkxMjk2MjY5",
    "name": "Hello-World",
    "full_name": "octocat/Hello-World",
    "owner": {
      "login": "octocat",
      "id": 1,
      "node_id": "MDQ6VXNlcjE=",
      "avatar_url": "https://github.com/images/error/octocat_happy.gif",
      "gravatar_id": "",
      "url": "https://api.github.com/users/octocat",
      "html_url": "https://github.com/octocat",
      "followers_url": "https://api.github.com/users/octocat/followers",
      "following_url": "https://api.github.com/users/octocat/following{/other_user}",
      "gists_url": "https://api.github.com/users/octocat/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/octocat/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/octocat/subscriptions",
      "organizations_url": "https://api.github.com/users/octocat/orgs",
      "repos_url": "https://api.github.com/users/octocat/repos",
      "events_url": "https://api.github.com/users/octocat/events{/privacy}",
      "received_events_url": "https://api.github.com/users/octocat/received_events",
      "type": "User",
      "site_admin": false
    },
    "private": false,
    "html_url": "https://github.com/octocat/Hello-World",
    "description": "This your first repo!",
    "fork": false,
    "url": "https://api.github.com/repos/octocat/Hello-World",
    "archive_url": "https://api.github.com/repos/octocat/Hello-World/{archive_format}{/ref}",
    "assignees_url": "https://api.github.com/repos/octocat/Hello-World/assignees{/user}",
    "blobs_url": "https://api.github.com/repos/octocat/Hello-World/git/blobs{/sha}",
    "branches_url": "https://api.github.com/repos/octocat/Hello-World/branches{/branch}",
    "collaborators_url": "https://api.github.com/repos/octocat/Hello-World/collaborators{/collaborator}",
    "comments_url": "https://api.github.com/repos/octocat/Hello-World/comments{/number}",
    "commits_url": "https://api.github.com/repos/octocat/Hello-World/commits{/sha}",
    "compare_url": "https://api.github.com/repos/octocat/Hello-World/compare/{base}...{head}",
    "contents_url": "https://api.github.com/repos/octocat/Hello-World/contents/{+path}",
    "contributors_url": "https://api.github.com/repos/octocat/Hello-World/contributors",
    "deployments_url": "https://api.github.com/repos/octocat/Hello-World/deployments",
    "downloads_url": "https://api.github.com/repos/octocat/Hello-World/downloads",
    "events_url": "https://api.github.com/repos/octocat/Hello-World/events",
    "forks_url": "https://api.github.com/repos/octocat/Hello-World/forks",
    "git_commits_url": "https://api.github.com/repos/octocat/Hello-World/git/commits{/sha}",
    "git_refs_url": "https://api.github.com/repos/octocat/Hello-World/git/refs{/sha}",
    "git_tags_url": "https://api.github.com/repos/octocat/Hello-World/git/tags{/sha}",
    "git_url": "git:github.com/octocat/Hello-World.git",
    "issue_comment_url": "https://api.github.com/repos/octocat/Hello-World/issues/comments{/number}",
    "issue_events_url": "https://api.github.com/repos/octocat/Hello-World/issues/events{/number}",
    "issues_url": "https://api.github.com/repos/octocat/Hello-World/issues{/number}",
    "keys_url": "https://api.github.com/repos/octocat/Hello-World/keys{/key_id}",
    "labels_url": "https://api.github.com/repos/octocat/Hello-World/labels{/name}",
    "languages_url": "https://api.github.com/repos/octocat/Hello-World/languages",
    "merges_url": "https://api.github.com/repos/octocat/Hello-World/merges",
    "milestones_url": "https://api.github.com/repos/octocat/Hello-World/milestones{/number}",
    "notifications_url": "https://api.github.com/repos/octocat/Hello-World/notifications{?since,all,participating}",
    "pulls_url": "https://api.github.com/repos/octocat/Hello-World/pulls{/number}",
    "releases_url": "https://api.github.com/repos/octocat/Hello-World/releases{/id}",
    "ssh_url": "git@github.com:octocat/Hello-World.git",
    "stargazers_url": "https://api.github.com/repos/octocat/Hello-World/stargazers",
    "statuses_url": "https://api.github.com/repos/octocat/Hello-World/statuses/{sha}",
    "subscribers_url": "https://api.github.com/repos/octocat/Hello-World/subscribers",
    "subscription_url": "https://api.github.com/repos/octocat/Hello-World/subscription",
    "tags_url": "https://api.github.com/repos/octocat/Hello-World/tags",
    "teams_url": "https://api.github.com/repos/octocat/Hello-World/teams",
    "trees_url": "https://api.github.com/repos/octocat/Hello-World/git/trees{/sha}",
    "clone_url": "https://github.com/octocat/Hello-World.git",
    "mirror_url": "git:git.example.com/octocat/Hello-World",
    "hooks_url": "https://api.github.com/repos/octocat/Hello-World/hooks",
    "svn_url": "https://svn.github.com/octocat/Hello-World",
    "homepage": "https://github.com",
    "language": null,
    "forks_count": 9,
    "stargazers_count": 80,
    "watchers_count": 80,
    "size": 108,
    "default_branch": "master",
    "open_issues_count": 0,
    "is_template": true,
    "topics": [
      "octocat",
      "atom",
      "electron",
      "api"
    ],
    "has_issues": true,
    "has_projects": true,
    "has_wiki": true,
    "has_pages": false,
    "has_downloads": true,
    "archived": false,
    "disabled": false,
    "visibility": "public",
    "pushed_at": "2011-01-26T19:06:43Z",
    "created_at": "2011-01-26T19:01:12Z",
    "updated_at": "2011-01-26T19:14:43Z",
    "permissions": {
      "admin": true,
      "push": true,
      "pull": true
    },
    "allow_rebase_merge": true,
    "template_repository": null,
    "temp_clone_token": "ABTLWHOULUVAXGTRYU7OC2876QJ2O",
    "allow_squash_merge": true,
    "allow_auto_merge": false,
    "delete_branch_on_merge": true,
    "allow_merge_commit": true,
    "subscribers_count": 42,
    "network_count": 0,
    "license": {
      "key": "mit",
      "name": "MIT License",
      "url": "https://api.github.com/licenses/mit",
      "spdx_id": "MIT",
      "node_id": "MDc6TGljZW5zZW1pdA==",
      "html_url": "https://github.com/licenses/mit"
    },
    "forks": 1,
    "open_issues": 1,
    "watchers": 1
  },
  {
    "id": 1296270,
    "name": "Hello-World2",
    "full_name": "octocat/Hello-World2",
    "html_url": "https://github.com/octocat/Hello-World2",
    "permissions": {
      "admin": false,
      "push": false,
      "pull": true
    }
  }
]
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.FetchAllRepositoryList(ctx, common.OauthContext{}, githubComURL)
	require.NoError(t, err)

	// Repositories without admin permissions should be excluded
	want := []*vcs.Repository{
		{
			ID:       "1296269",
			Name:     "Hello-World",
			FullPath: "octocat/Hello-World",
			WebURL:   "https://github.com/octocat/Hello-World",
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchRepositoryFileList(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/repos/octocat/Hello-World/git/trees/main", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/rest/git/trees#get-a-tree
			Body: io.NopCloser(strings.NewReader(`
{
  "sha": "9fb037999f264ba9a7fc6274d15fa3ae2ab98312",
  "url": "https://api.github.com/repos/octocat/Hello-World/trees/9fb037999f264ba9a7fc6274d15fa3ae2ab98312",
  "tree": [
    {
      "path": "file.rb",
      "mode": "100644",
      "type": "blob",
      "size": 30,
      "sha": "44b4fc6d56897b048c772eb4087f854f46256132",
      "url": "https://api.github.com/repos/octocat/Hello-World/git/blobs/44b4fc6d56897b048c772eb4087f854f46256132"
    },
    {
      "path": "subdir",
      "mode": "040000",
      "type": "tree",
      "sha": "f484d249c660418515fb01c2b9662073663c242e",
      "url": "https://api.github.com/repos/octocat/Hello-World/git/blobs/f484d249c660418515fb01c2b9662073663c242e"
    },
    {
      "path": "subdir/exec_file",
      "mode": "100755",
      "type": "blob",
      "size": 75,
      "sha": "45b983be36b73c0788dc9cbcb76cbb80fc7bb057",
      "url": "https://api.github.com/repos/octocat/Hello-World/git/blobs/45b983be36b73c0788dc9cbcb76cbb80fc7bb057"
    },
    {
      "path": "anotherdir/.gitignore",
      "mode": "100755",
      "type": "blob",
      "size": 75,
      "sha": "5ff01e0bbbd12a36679ddf2ddd186bac8ad5c6b4",
      "url": "https://api.github.com/repos/octocat/Hello-World/git/blobs/5ff01e0bbbd12a36679ddf2ddd186bac8ad5c6b4"
    }
  ],
  "truncated": false
}
`)),
		}, nil
	},
	)

	t.Run("no path prefix", func(t *testing.T) {
		ctx := context.Background()
		got, err := p.FetchRepositoryFileList(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", "main", "")
		require.NoError(t, err)

		// Non-blob type should excluded
		want := []*vcs.RepositoryTreeNode{
			{
				Path: "file.rb",
				Type: "blob",
			},
			{
				Path: "subdir/exec_file",
				Type: "blob",
			},
			{
				Path: "anotherdir/.gitignore",
				Type: "blob",
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("has path prefix", func(t *testing.T) {
		ctx := context.Background()
		got, err := p.FetchRepositoryFileList(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", "main", "subdir")
		require.NoError(t, err)

		// Non-blob type should be excluded
		want := []*vcs.RepositoryTreeNode{
			{
				Path: "subdir/exec_file",
				Type: "blob",
			},
		}
		assert.Equal(t, want, got)
	})
}

func TestProvider_CreateFile(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/repos/octocat/Hello-World/contents/notes%2Fhello.txt", r.URL.RawPath)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		wantBody := `{"message":"my commit message","content":"bXkgbmV3IGZpbGUgY29udGVudHM=","branch":"master"}`
		assert.Equal(t, wantBody, string(body))
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/rest/repos/contents#create-or-update-file-contents
			Body: io.NopCloser(strings.NewReader(`
{
  "content": {
    "name": "hello.txt",
    "path": "notes/hello.txt",
    "sha": "95b966ae1c166bd92f8ae7d1c313e738c731dfc3",
    "size": 9,
    "url": "https://api.github.com/repos/octocat/Hello-World/contents/notes/hello.txt",
    "html_url": "https://github.com/octocat/Hello-World/blob/master/notes/hello.txt",
    "git_url": "https://api.github.com/repos/octocat/Hello-World/git/blobs/95b966ae1c166bd92f8ae7d1c313e738c731dfc3",
    "download_url": "https://raw.githubusercontent.com/octocat/HelloWorld/master/notes/hello.txt",
    "type": "file",
    "_links": {
      "self": "https://api.github.com/repos/octocat/Hello-World/contents/notes/hello.txt",
      "git": "https://api.github.com/repos/octocat/Hello-World/git/blobs/95b966ae1c166bd92f8ae7d1c313e738c731dfc3",
      "html": "https://github.com/octocat/Hello-World/blob/master/notes/hello.txt"
    }
  },
  "commit": {
    "sha": "7638417db6d59f3c431d3e1f261cc637155684cd",
    "node_id": "MDY6Q29tbWl0NzYzODQxN2RiNmQ1OWYzYzQzMWQzZTFmMjYxY2M2MzcxNTU2ODRjZA==",
    "url": "https://api.github.com/repos/octocat/Hello-World/git/commits/7638417db6d59f3c431d3e1f261cc637155684cd",
    "html_url": "https://github.com/octocat/Hello-World/git/commit/7638417db6d59f3c431d3e1f261cc637155684cd",
    "author": {
      "date": "2014-11-07T22:01:45Z",
      "name": "Monalisa Octocat",
      "email": "octocat@github.com"
    },
    "committer": {
      "date": "2014-11-07T22:01:45Z",
      "name": "Monalisa Octocat",
      "email": "octocat@github.com"
    },
    "message": "my commit message",
    "tree": {
      "url": "https://api.github.com/repos/octocat/Hello-World/git/trees/691272480426f78a0138979dd3ce63b77f706feb",
      "sha": "691272480426f78a0138979dd3ce63b77f706feb"
    },
    "parents": [
      {
        "url": "https://api.github.com/repos/octocat/Hello-World/git/commits/1acc419d4d6a9ce985db7be48c6349a0475975b5",
        "html_url": "https://github.com/octocat/Hello-World/git/commit/1acc419d4d6a9ce985db7be48c6349a0475975b5",
        "sha": "1acc419d4d6a9ce985db7be48c6349a0475975b5"
      }
    ],
    "verification": {
      "verified": false,
      "reason": "unsigned",
      "signature": null,
      "payload": null
    }
  }
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	err := p.CreateFile(
		ctx,
		common.OauthContext{},
		githubComURL,
		"octocat/Hello-World",
		"notes/hello.txt",
		vcs.FileCommitCreate{
			Branch:        "master",
			Content:       "my new file contents",
			CommitMessage: "my commit message",
		},
	)
	require.NoError(t, err)
}

func TestProvider_OverwriteFile(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/repos/octocat/Hello-World/contents/notes%2Fhello.txt", r.URL.RawPath)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		wantBody := `{"message":"update file","content":"bXkgbmV3IGZpbGUgY29udGVudHM=","sha":"7638417db6d59f3c431d3e1f261cc637155684cd","branch":"master"}`
		assert.Equal(t, wantBody, string(body))
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/rest/repos/contents#create-or-update-file-contents
			Body: io.NopCloser(strings.NewReader(`
{
  "content": {
    "name": "hello.txt",
    "path": "notes/hello.txt",
    "sha": "95b966ae1c166bd92f8ae7d1c313e738c731dfc3",
    "size": 9,
    "url": "https://api.github.com/repos/octocat/Hello-World/contents/notes/hello.txt",
    "html_url": "https://github.com/octocat/Hello-World/blob/master/notes/hello.txt",
    "git_url": "https://api.github.com/repos/octocat/Hello-World/git/blobs/95b966ae1c166bd92f8ae7d1c313e738c731dfc3",
    "download_url": "https://raw.githubusercontent.com/octocat/HelloWorld/master/notes/hello.txt",
    "type": "file",
    "_links": {
      "self": "https://api.github.com/repos/octocat/Hello-World/contents/notes/hello.txt",
      "git": "https://api.github.com/repos/octocat/Hello-World/git/blobs/95b966ae1c166bd92f8ae7d1c313e738c731dfc3",
      "html": "https://github.com/octocat/Hello-World/blob/master/notes/hello.txt"
    }
  },
  "commit": {
    "sha": "7638417db6d59f3c431d3e1f261cc637155684cd",
    "node_id": "MDY6Q29tbWl0NzYzODQxN2RiNmQ1OWYzYzQzMWQzZTFmMjYxY2M2MzcxNTU2ODRjZA==",
    "url": "https://api.github.com/repos/octocat/Hello-World/git/commits/7638417db6d59f3c431d3e1f261cc637155684cd",
    "html_url": "https://github.com/octocat/Hello-World/git/commit/7638417db6d59f3c431d3e1f261cc637155684cd",
    "author": {
      "date": "2014-11-07T22:01:45Z",
      "name": "Monalisa Octocat",
      "email": "octocat@github.com"
    },
    "committer": {
      "date": "2014-11-07T22:01:45Z",
      "name": "Monalisa Octocat",
      "email": "octocat@github.com"
    },
    "message": "my commit message",
    "tree": {
      "url": "https://api.github.com/repos/octocat/Hello-World/git/trees/691272480426f78a0138979dd3ce63b77f706feb",
      "sha": "691272480426f78a0138979dd3ce63b77f706feb"
    },
    "parents": [
      {
        "url": "https://api.github.com/repos/octocat/Hello-World/git/commits/1acc419d4d6a9ce985db7be48c6349a0475975b5",
        "html_url": "https://github.com/octocat/Hello-World/git/commit/1acc419d4d6a9ce985db7be48c6349a0475975b5",
        "sha": "1acc419d4d6a9ce985db7be48c6349a0475975b5"
      }
    ],
    "verification": {
      "verified": false,
      "reason": "unsigned",
      "signature": null,
      "payload": null
    }
  }
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	err := p.OverwriteFile(
		ctx,
		common.OauthContext{},
		githubComURL,
		"octocat/Hello-World",
		"notes/hello.txt",
		vcs.FileCommitCreate{
			Branch:        "master",
			Content:       "my new file contents",
			CommitMessage: "update file",
			SHA:           "7638417db6d59f3c431d3e1f261cc637155684cd",
		},
	)
	require.NoError(t, err)
}

func TestProvider_ReadFileMeta(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/repos/octocat/Hello-World/contents/README.md":
			return &http.Response{
				StatusCode: http.StatusOK,
				// Example response derived from https://docs.github.com/en/rest/repos/contents#get-repository-content
				Body: io.NopCloser(strings.NewReader(`
{
  "type": "file",
  "encoding": "base64",
  "size": 442,
  "name": "README.md",
  "path": "README.md",
  "content": "IyBTYW1wbGUgR2l0TGFiIFByb2plY3QKClRoaXMgc2FtcGxlIHByb2plY3Qgc2hvd3MgaG93IGEgcHJvamVjdCBpbiBHaXRMYWIgbG9va3MgZm9yIGRlbW9uc3RyYXRpb24gcHVycG9zZXMuIEl0IGNvbnRhaW5zIGlzc3VlcywgbWVyZ2UgcmVxdWVzdHMgYW5kIE1hcmtkb3duIGZpbGVzIGluIG1hbnkgYnJhbmNoZXMsCm5hbWVkIGFuZCBmaWxsZWQgd2l0aCBsb3JlbSBpcHN1bS4KCllvdSBjYW4gbG9vayBhcm91bmQgdG8gZ2V0IGFuIGlkZWEgaG93IHRvIHN0cnVjdHVyZSB5b3VyIHByb2plY3QgYW5kLCB3aGVuIGRvbmUsIHlvdSBjYW4gc2FmZWx5IGRlbGV0ZSB0aGlzIHByb2plY3QuCgpbTGVhcm4gbW9yZSBhYm91dCBjcmVhdGluZyBHaXRMYWIgcHJvamVjdHMuXShodHRwczovL2RvY3MuZ2l0bGFiLmNvbS9lZS9naXRsYWItYmFzaWNzL2NyZWF0ZS1wcm9qZWN0Lmh0bWwpCg==",
  "sha": "3d21ec53a331a6f037a91c368710b99387d012c1",
  "url": "https://api.github.com/repos/octocat/Hello-World/contents/README.md",
  "git_url": "https://api.github.com/repos/octocat/Hello-World/git/blobs/3d21ec53a331a6f037a91c368710b99387d012c1",
  "html_url": "https://github.com/octocat/Hello-World/blob/master/README.md",
  "download_url": "https://raw.githubusercontent.com/octocat/Hello-World/master/README.md",
  "_links": {
    "git": "https://api.github.com/repos/octocat/Hello-World/git/blobs/3d21ec53a331a6f037a91c368710b99387d012c1",
    "self": "https://api.github.com/repos/octocat/Hello-World/contents/README.md",
    "html": "https://github.com/octocat/Hello-World/blob/master/README.md"
  }
}
`)),
			}, nil
		case "/repos/octocat/Hello-World/git/ref/heads/master":
			return &http.Response{
				StatusCode: http.StatusOK,
				// Example response derived from https://docs.github.com/en/rest/git/refs?apiVersion=2022-11-28#get-a-reference
				Body: io.NopCloser(strings.NewReader(`
{
  "ref": "refs/heads/master",
  "node_id": "MDM6UmVmcmVmcy9oZWFkcy9mZWF0dXJlQQ==",
  "url": "https://api.github.com/repos/octocat/Hello-World/git/refs/heads/master",
  "object": {
    "type": "commit",
    "sha": "aa218f56b14c9653891f9e74264a383fa43fefbd",
    "url": "https://api.github.com/repos/octocat/Hello-World/git/commits/aa218f56b14c9653891f9e74264a383fa43fefbd"
  }
}
        `)),
			}, nil
		default:
			assert.Truef(t, false, "Unsupported API: %s", r.URL.Path)
			return nil, nil
		}
	},
	)

	ctx := context.Background()
	got, err := p.ReadFileMeta(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", "README.md", vcs.RefInfo{
		RefType: vcs.RefTypeBranch,
		RefName: "master",
	})
	require.NoError(t, err)

	want := &vcs.FileMeta{
		Name:         "README.md",
		Path:         "README.md",
		Size:         442,
		SHA:          "3d21ec53a331a6f037a91c368710b99387d012c1",
		LastCommitID: "aa218f56b14c9653891f9e74264a383fa43fefbd",
	}
	assert.Equal(t, want, got)
}

func TestProvider_ReadFileContent(t *testing.T) {
	const want = `# Sample GitLab Project

This sample project shows how a project in GitLab looks for demonstration purposes. It contains issues, merge requests and Markdown files in many branches,
named and filled with lorem ipsum.

You can look around to get an idea how to structure your project and, when done, you can safely delete this project.

[Learn more about creating GitLab projects.](https://docs.gitlab.com/ee/gitlab-basics/create-project.html)
`

	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/repos/octocat/Hello-World/contents/README.md", r.URL.Path)
		assert.Equal(t, "application/vnd.github.raw", r.Header.Get("Accept"))
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response derived from https://docs.github.com/en/rest/repos/contents#get-repository-content
			Body: io.NopCloser(strings.NewReader(want)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.ReadFileContent(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", "README.md", vcs.RefInfo{
		RefType: vcs.RefTypeBranch,
		RefName: "master",
	})
	require.NoError(t, err)

	assert.Equal(t, want, got)
}

func TestProvider_CreateWebhook(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/repos/1/hooks", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusCreated,
			// Example response taken from https://docs.github.com/en/rest/webhooks/repos#create-a-repository-webhook
			Body: io.NopCloser(strings.NewReader(`
{
  "type": "Repository",
  "id": 12345678,
  "name": "web",
  "active": true,
  "events": [
    "push",
    "pull_request"
  ],
  "config": {
    "content_type": "json",
    "insecure_ssl": "0",
    "url": "https://example.com/webhook"
  },
  "updated_at": "2019-06-03T00:57:16Z",
  "created_at": "2019-06-03T00:57:16Z",
  "url": "https://api.github.com/repos/octocat/Hello-World/hooks/12345678",
  "test_url": "https://api.github.com/repos/octocat/Hello-World/hooks/12345678/test",
  "ping_url": "https://api.github.com/repos/octocat/Hello-World/hooks/12345678/pings",
  "deliveries_url": "https://api.github.com/repos/octocat/Hello-World/hooks/12345678/deliveries",
  "last_response": {
    "code": null,
    "status": "unused",
    "message": null
  }
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.CreateWebhook(ctx, common.OauthContext{}, githubComURL, "1", []byte(""))
	require.NoError(t, err)
	assert.Equal(t, "12345678", got)
}

func TestProvider_PatchWebhook(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/repos/1/hooks/1", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	},
	)

	ctx := context.Background()
	err := p.PatchWebhook(ctx, common.OauthContext{}, githubComURL, "1", "1", []byte(""))
	require.NoError(t, err)
}

func TestProvider_DeleteWebhook(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/repos/1/hooks/1", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	},
	)

	ctx := context.Background()
	err := p.DeleteWebhook(ctx, common.OauthContext{}, githubComURL, "1", "1")
	require.NoError(t, err)
}

func TestOAuth_RefreshToken(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				if token == "expired" {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body: io.NopCloser(strings.NewReader(`
					{"error":"invalid_grant","error_description":"The provided authorization grant is invalid, expired, revoked, does not match the redirection URI used in the authorization request, or was issued to another client."}
					`)),
					}, nil
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					// Example response taken from https://docs.github.com/en/developers/apps/building-github-apps/refreshing-user-to-server-access-tokens#renewing-a-user-token-with-a-refresh-token
					Body: io.NopCloser(strings.NewReader(`
{
  "access_token": "ghu_16C7e42F292c6912E7710c838347Ae178B4a",
  "expires_in": "28800",
  "refresh_token": "ghr_1B4a2e77838347a7E420ce178F2E7c6912E169246c34E1ccbF66C46812d16D5B1A9Dc86A1498",
  "refresh_token_expires_in": "15811200",
  "scope": "",
  "token_type": "bearer"
}
`)),
				}, nil
			},
		},
	}
	token := "expired"

	calledRefresher := false
	refresher := func(_, _ string, _ int64) error {
		calledRefresher = true
		return nil
	}

	_, _, _, err := oauth.Get(
		ctx,
		client,
		"https://api.github.com/users/octocat",
		&token,
		tokenRefresher(
			githubComURL,
			oauthContext{},
			refresher,
		),
	)
	require.NoError(t, err)
	assert.Equal(t, "ghu_16C7e42F292c6912E7710c838347Ae178B4a", token)
	assert.True(t, calledRefresher)
}

func TestProvider_GetBranch(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repos/octocat/Hello-World/git/ref/heads/featureA", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/rest/git/refs#get-a-reference
			Body: io.NopCloser(strings.NewReader(`
{
  "ref": "refs/heads/featureA",
  "node_id": "MDM6UmVmcmVmcy9oZWFkcy9mZWF0dXJlQQ==",
  "url": "https://api.github.com/repos/octocat/Hello-World/git/refs/heads/featureA",
  "object": {
    "type": "commit",
    "sha": "aa218f56b14c9653891f9e74264a383fa43fefbd",
    "url": "https://api.github.com/repos/octocat/Hello-World/git/commits/aa218f56b14c9653891f9e74264a383fa43fefbd"
  }
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.GetBranch(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", "featureA")
	require.NoError(t, err)

	want := &vcs.BranchInfo{
		Name:         "featureA",
		LastCommitID: "aa218f56b14c9653891f9e74264a383fa43fefbd",
	}
	assert.Equal(t, want, got)
}

func TestProvider_CreateBranch(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repos/octocat/Hello-World/git/refs", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/rest/git/refs#create-a-reference
			Body: io.NopCloser(strings.NewReader(`
{
  "ref": "refs/heads/featureA",
  "node_id": "MDM6UmVmcmVmcy9oZWFkcy9mZWF0dXJlQQ==",
  "url": "https://api.github.com/repos/octocat/Hello-World/git/refs/heads/featureA",
  "object": {
    "type": "commit",
    "sha": "aa218f56b14c9653891f9e74264a383fa43fefbd",
    "url": "https://api.github.com/repos/octocat/Hello-World/git/commits/aa218f56b14c9653891f9e74264a383fa43fefbd"
  }
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	err := p.CreateBranch(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", &vcs.BranchInfo{
		Name:         "featureA",
		LastCommitID: "aa218f56b14c9653891f9e74264a383fa43fefbd",
	})
	require.NoError(t, err)
}

func TestProvider_CreatePullRequest(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repos/octocat/Hello-World/pulls", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/rest/pulls/pulls#create-a-pull-request
			Body: io.NopCloser(strings.NewReader(`
{
  "url": "https://api.github.com/repos/octocat/Hello-World/pulls/1347",
  "id": 1,
  "node_id": "MDExOlB1bGxSZXF1ZXN0MQ==",
  "html_url": "https://github.com/octocat/Hello-World/pull/1347",
  "diff_url": "https://github.com/octocat/Hello-World/pull/1347.diff",
  "patch_url": "https://github.com/octocat/Hello-World/pull/1347.patch",
  "issue_url": "https://api.github.com/repos/octocat/Hello-World/issues/1347",
  "commits_url": "https://api.github.com/repos/octocat/Hello-World/pulls/1347/commits",
  "review_comments_url": "https://api.github.com/repos/octocat/Hello-World/pulls/1347/comments",
  "review_comment_url": "https://api.github.com/repos/octocat/Hello-World/pulls/comments{/number}",
  "comments_url": "https://api.github.com/repos/octocat/Hello-World/issues/1347/comments",
  "statuses_url": "https://api.github.com/repos/octocat/Hello-World/statuses/6dcb09b5b57875f334f61aebed695e2e4193db5e",
  "number": 1347,
  "state": "open",
  "locked": true,
  "title": "Amazing new feature",
  "body": "Please pull these awesome changes in!",
  "active_lock_reason": "too heated",
  "created_at": "2011-01-26T19:01:12Z",
  "updated_at": "2011-01-26T19:01:12Z",
  "closed_at": "2011-01-26T19:01:12Z",
  "merged_at": "2011-01-26T19:01:12Z",
  "merge_commit_sha": "e5bd3914e2e596debea16f433f57875b5b90bcd6",
  "head": {
    "label": "octocat:new-topic",
    "ref": "new-topic",
    "sha": "6dcb09b5b57875f334f61aebed695e2e4193db5e",
    "user": {
      "login": "octocat",
      "id": 1,
      "node_id": "MDQ6VXNlcjE=",
      "avatar_url": "https://github.com/images/error/octocat_happy.gif",
      "gravatar_id": "",
      "url": "https://api.github.com/users/octocat",
      "html_url": "https://github.com/octocat",
      "followers_url": "https://api.github.com/users/octocat/followers",
      "following_url": "https://api.github.com/users/octocat/following{/other_user}",
      "gists_url": "https://api.github.com/users/octocat/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/octocat/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/octocat/subscriptions",
      "organizations_url": "https://api.github.com/users/octocat/orgs",
      "repos_url": "https://api.github.com/users/octocat/repos",
      "events_url": "https://api.github.com/users/octocat/events{/privacy}",
      "received_events_url": "https://api.github.com/users/octocat/received_events",
      "type": "User",
      "site_admin": false
    }
  },
  "base": {
    "label": "octocat:master",
    "ref": "master",
    "sha": "6dcb09b5b57875f334f61aebed695e2e4193db5e",
    "user": {
      "login": "octocat",
      "id": 1,
      "node_id": "MDQ6VXNlcjE=",
      "avatar_url": "https://github.com/images/error/octocat_happy.gif",
      "gravatar_id": "",
      "url": "https://api.github.com/users/octocat",
      "html_url": "https://github.com/octocat",
      "followers_url": "https://api.github.com/users/octocat/followers",
      "following_url": "https://api.github.com/users/octocat/following{/other_user}",
      "gists_url": "https://api.github.com/users/octocat/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/octocat/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/octocat/subscriptions",
      "organizations_url": "https://api.github.com/users/octocat/orgs",
      "repos_url": "https://api.github.com/users/octocat/repos",
      "events_url": "https://api.github.com/users/octocat/events{/privacy}",
      "received_events_url": "https://api.github.com/users/octocat/received_events",
      "type": "User",
      "site_admin": false
    }
  },
  "author_association": "OWNER",
  "auto_merge": null,
  "draft": false,
  "merged": false,
  "mergeable": true,
  "rebaseable": true,
  "mergeable_state": "clean",
  "comments": 10,
  "review_comments": 0,
  "maintainer_can_modify": true,
  "commits": 3,
  "additions": 100,
  "deletions": 3,
  "changed_files": 5
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	res, err := p.CreatePullRequest(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", &vcs.PullRequestCreate{
		Title:                 "Amazing new feature",
		Body:                  "Please pull these awesome changes in!",
		Head:                  "new-topic",
		Base:                  "master",
		RemoveHeadAfterMerged: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/octocat/Hello-World/pull/1347", res.URL)
}

func TestProvider_UpsertEnvironmentVariable(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/repos/octocat/Hello-World/actions/secrets/public-key":
			assert.Equal(t, "GET", r.Method)
			return &http.Response{
				StatusCode: http.StatusOK,
				// Example response taken from https://docs.github.com/en/rest/actions/secrets#get-a-repository-public-key
				Body: io.NopCloser(strings.NewReader(`
{
  "key_id": "568250167242549743",
  "key": "YJf3Ojcv8TSEBCtR0wtTR/F2bD3nBl1lxiwkfV/TYQk="
}
`)),
			}, nil
		case "/repos/octocat/Hello-World/actions/secrets/1":
			assert.Equal(t, "PUT", r.Method)
			return &http.Response{
				StatusCode: http.StatusOK,
				// Example response taken from https://docs.github.com/en/rest/actions/secrets#create-or-update-a-repository-secret
				Body: io.NopCloser(strings.NewReader("")),
			}, nil
		}

		return nil, errors.Errorf("Invalid request. %s: %s", r.Method, r.URL.Path)
	},
	)

	ctx := context.Background()
	err := p.UpsertEnvironmentVariable(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", "1", "new value")
	require.NoError(t, err)
}

func TestProvider_ListPullRequestFile(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/repos/octocat/Hello-World/pulls/1/files", r.URL.Path)
		if r.URL.Query().Get("page") == "2" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/rest/pulls/pulls#list-pull-requests-files
			Body: io.NopCloser(strings.NewReader(`
[
  {
    "sha": "bbcd538c8e72b8c175046e27cc8f907076331401",
    "filename": "file1.txt",
    "status": "added",
    "additions": 103,
    "deletions": 21,
    "changes": 124,
    "blob_url": "https://github.com/octocat/Hello-World/blob/6dcb09b5b57875f334f61aebed695e2e4193db5e/file1.txt",
    "raw_url": "https://github.com/octocat/Hello-World/raw/6dcb09b5b57875f334f61aebed695e2e4193db5e/file1.txt",
    "contents_url": "https://api.github.com/repos/octocat/Hello-World/contents/file1.txt?ref=6dcb09b5b57875f334f61aebed695e2e4193db5e",
    "patch": "@@ -132,7 +132,7 @@ module Test @@ -1000,7 +1000,7 @@ module Test"
  }
]
`)),
		}, nil
	},
	)
	ctx := context.Background()
	got, err := p.ListPullRequestFile(ctx, common.OauthContext{}, githubComURL, "octocat/Hello-World", "1")
	require.NoError(t, err)

	want := []*vcs.PullRequestFile{
		{
			Path:         "file1.txt",
			LastCommitID: "6dcb09b5b57875f334f61aebed695e2e4193db5e",
			IsDeleted:    false,
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_GetDiffFileList(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v3/repos/1/compare/before_sha...after_sha", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://docs.github.com/en/rest/commits/commits#compare-two-commits
			Body: io.NopCloser(strings.NewReader(`
{
	"files": [
    {
      "sha": "bbcd538c8e72b8c175046e27cc8f907076331401",
      "filename": "file1.txt",
      "status": "added",
      "additions": 103,
      "deletions": 21,
      "changes": 124,
      "blob_url": "https://github.com/octocat/Hello-World/blob/6dcb09b5b57875f334f61aebed695e2e4193db5e/file1.txt",
      "raw_url": "https://github.com/octocat/Hello-World/raw/6dcb09b5b57875f334f61aebed695e2e4193db5e/file1.txt",
      "contents_url": "https://api.github.com/repos/octocat/Hello-World/contents/file1.txt?ref=6dcb09b5b57875f334f61aebed695e2e4193db5e",
      "patch": "@@ -132,7 +132,7 @@ module Test @@ -1000,7 +1000,7 @@ module Test"
    }
  ]
}
`)),
		}, nil
	},
	)
	ctx := context.Background()
	got, err := p.GetDiffFileList(ctx, common.OauthContext{}, "", "1", "before_sha", "after_sha")
	require.NoError(t, err)

	want := []vcs.FileDiff{
		{
			Path: "file1.txt",
			Type: vcs.FileDiffTypeAdded,
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
