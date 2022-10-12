package gitlab

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/internal/oauth"
)

func TestProvider_FetchUserInfo(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/api/v4/users/1", r.URL.Path)
						return &http.Response{
							StatusCode: http.StatusOK,
							// Example response taken from https://docs.gitlab.com/ee/api/users.html#single-user
							Body: io.NopCloser(strings.NewReader(`
{
  "id": 1,
  "username": "john_smith",
  "name": "John Smith",
  "state": "active",
  "avatar_url": "http://localhost:3000/uploads/user/avatar/1/cd8.jpeg",
  "web_url": "http://localhost:3000/john_smith",
  "created_at": "2012-05-23T08:00:58Z",
  "bio": "",
  "bot": false,
  "location": null,
  "public_email": "john@example.com",
  "skype": "",
  "linkedin": "",
  "twitter": "",
  "website_url": "",
  "organization": "",
  "job_title": "Operations Specialist",
  "followers": 1,
  "following": 1
}
`)),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.FetchUserInfo(ctx, common.OauthContext{}, "", "1")
	require.NoError(t, err)

	want := &vcs.UserInfo{
		PublicEmail: "john@example.com",
		Name:        "John Smith",
		State:       vcs.StateActive,
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchRepositoryActiveMemberList(t *testing.T) {
	t.Run("missing public email", func(t *testing.T) {
		p := newProvider(
			vcs.ProviderConfig{
				Client: &http.Client{
					Transport: &common.MockRoundTripper{
						MockRoundTrip: func(r *http.Request) (*http.Response, error) {
							switch r.URL.Path {
							case "/api/v4/projects/1/members/all":
								return &http.Response{
									StatusCode: http.StatusOK,
									// Example response derived from https://docs.gitlab.com/ee/api/members.html#list-all-members-of-a-group-or-project-including-inherited-and-invited-members
									Body: io.NopCloser(strings.NewReader(`
[
  {
    "id": 1,
    "username": "raymond_smith",
    "name": "Raymond Smith",
    "state": "active",
    "avatar_url": "https://www.gravatar.com/avatar/c2525a7f58ae3776070e44c106c48e15?s=80&d=identicon",
    "web_url": "http://192.168.1.8:3000/root",
    "created_at": "2012-09-22T14:13:35Z",
    "created_by": {
      "id": 2,
      "username": "john_doe",
      "name": "John Doe",
      "state": "active",
      "avatar_url": "https://www.gravatar.com/avatar/c2525a7f58ae3776070e44c106c48e15?s=80&d=identicon",
      "web_url": "http://192.168.1.8:3000/root"
    },
    "expires_at": "2012-10-22T14:13:35Z",
    "access_level": 30,
    "group_saml_identity": null,
    "membership_state": "active"
  }
]
`)),
								}, nil
							case "/api/v4/users/1":
								return &http.Response{
									StatusCode: http.StatusOK,
									// Example response derived from https://docs.gitlab.com/ee/api/users.html#single-user
									Body: io.NopCloser(strings.NewReader(`
{
  "id": 1,
  "username": "john_smith",
  "name": "John Smith",
  "state": "active",
  "avatar_url": "http://localhost:3000/uploads/user/avatar/1/cd8.jpeg",
  "web_url": "http://localhost:3000/john_smith",
  "created_at": "2012-05-23T08:00:58Z",
  "bio": "",
  "bot": false,
  "location": null,
  "public_email": "",
  "skype": "",
  "linkedin": "",
  "twitter": "",
  "website_url": "",
  "organization": "",
  "job_title": "Operations Specialist",
  "followers": 1,
  "following": 1
}
`)),
								}, nil
							}
							return nil, errors.Errorf("unexpected request path: %s", r.URL.Path)
						},
					},
				},
			},
		)

		ctx := context.Background()
		_, got := p.FetchRepositoryActiveMemberList(ctx, common.OauthContext{}, "", "1")
		want := "[ Raymond Smith ] did not configure their public email in GitLab, please make sure every members' public email is configured before syncing, see https://docs.gitlab.com/ee/user/profile"
		assert.EqualError(t, got, want)
	})

	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						switch r.URL.Path {
						case "/api/v4/projects/1/members/all":
							return &http.Response{
								StatusCode: http.StatusOK,
								// Example response derived from https://docs.gitlab.com/ee/api/members.html#list-all-members-of-a-group-or-project-including-inherited-and-invited-members
								Body: io.NopCloser(strings.NewReader(`
[
  {
    "id": 1,
    "username": "raymond_smith",
    "name": "Raymond Smith",
    "state": "active",
    "avatar_url": "https://www.gravatar.com/avatar/c2525a7f58ae3776070e44c106c48e15?s=80&d=identicon",
    "web_url": "http://192.168.1.8:3000/root",
    "created_at": "2012-09-22T14:13:35Z",
    "created_by": {
      "id": 2,
      "username": "john_doe",
      "name": "John Doe",
      "state": "active",
      "avatar_url": "https://www.gravatar.com/avatar/c2525a7f58ae3776070e44c106c48e15?s=80&d=identicon",
      "web_url": "http://192.168.1.8:3000/root"
    },
    "expires_at": "2012-10-22T14:13:35Z",
    "access_level": 30,
    "group_saml_identity": null,
    "membership_state": "active"
  },
  {
    "id": 2,
    "username": "john_doe",
    "name": "John Doe",
    "state": "archived",
    "avatar_url": "https://www.gravatar.com/avatar/c2525a7f58ae3776070e44c106c48e15?s=80&d=identicon",
    "web_url": "http://192.168.1.8:3000/root",
    "created_at": "2012-09-22T14:13:35Z",
    "created_by": {
      "id": 1,
      "username": "raymond_smith",
      "name": "Raymond Smith",
      "state": "active",
      "avatar_url": "https://www.gravatar.com/avatar/c2525a7f58ae3776070e44c106c48e15?s=80&d=identicon",
      "web_url": "http://192.168.1.8:3000/root"
    },
    "expires_at": "2012-10-22T14:13:35Z",
    "access_level": 30,
    "email": "john@example.com",
    "group_saml_identity": {
      "extern_uid":"ABC-1234567890",
      "provider": "group_saml",
      "saml_provider_id": 10
    },
    "membership_state": "active"
  }
]
`)),
							}, nil
						case "/api/v4/users/1":
							return &http.Response{
								StatusCode: http.StatusOK,
								// Example response taken from https://docs.gitlab.com/ee/api/users.html#single-user
								Body: io.NopCloser(strings.NewReader(`
{
  "id": 1,
  "username": "john_smith",
  "name": "John Smith",
  "state": "active",
  "avatar_url": "http://localhost:3000/uploads/user/avatar/1/cd8.jpeg",
  "web_url": "http://localhost:3000/john_smith",
  "created_at": "2012-05-23T08:00:58Z",
  "bio": "",
  "bot": false,
  "location": null,
  "public_email": "john@example.com",
  "skype": "",
  "linkedin": "",
  "twitter": "",
  "website_url": "",
  "organization": "",
  "job_title": "Operations Specialist",
  "followers": 1,
  "following": 1
}
`)),
							}, nil
						}
						return nil, errors.Errorf("unexpected request path: %s", r.URL.Path)
					},
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.FetchRepositoryActiveMemberList(ctx, common.OauthContext{}, "", "1")
	require.NoError(t, err)

	// Non-active member should be excluded
	want := []*vcs.RepositoryMember{
		{
			Email:        "john@example.com",
			Name:         "Raymond Smith",
			State:        vcs.StateActive,
			Role:         common.ProjectDeveloper,
			VCSRole:      string(ProjectRoleDeveloper),
			RoleProvider: vcs.GitLabSelfHost,
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchCommitByID(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/api/v4/projects/5/repository/commits/6104942438c14ec7bd21c6cd5bd995272b3faff6", r.URL.Path)
						return &http.Response{
							StatusCode: http.StatusOK,
							// Example response taken from https://docs.gitlab.com/ee/api/commits.html#get-a-single-commit
							Body: io.NopCloser(strings.NewReader(`
{
  "id": "6104942438c14ec7bd21c6cd5bd995272b3faff6",
  "short_id": "6104942438c",
  "title": "Sanitize for network graph",
  "author_name": "randx",
  "author_email": "user@example.com",
  "committer_name": "Dmitriy",
  "committer_email": "user@example.com",
  "created_at": "2021-09-20T09:06:12.300+03:00",
  "message": "Sanitize for network graph",
  "committed_date": "2021-09-20T09:06:12.300+03:00",
  "authored_date": "2021-09-20T09:06:12.420+03:00",
  "parent_ids": [
    "ae1d9fb46aa2b07ee9836d49862ec4e2c46fbbba"
  ],
  "last_pipeline" : {
    "id": 8,
    "ref": "master",
    "sha": "2dc6aa325a317eda67812f05600bdf0fcdc70ab0",
    "status": "created"
  },
  "stats": {
    "additions": 15,
    "deletions": 10,
    "total": 25
  },
  "status": "running",
  "web_url": "https://gitlab.example.com/thedude/gitlab-foss/-/commit/6104942438c14ec7bd21c6cd5bd995272b3faff6"
}
`)),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.FetchCommitByID(ctx, common.OauthContext{}, "", "5", "6104942438c14ec7bd21c6cd5bd995272b3faff6")
	require.NoError(t, err)

	want := &vcs.Commit{
		ID:         "6104942438c14ec7bd21c6cd5bd995272b3faff6",
		AuthorName: "randx",
		CreatedTs:  1632117972,
	}
	assert.Equal(t, want, got)
}

func TestProvider_ExchangeOAuthToken(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/oauth/token", r.URL.Path)
						assert.Equal(t, "client_id=test_client_id&client_secret=test_client_secret&code=test_code&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A3000", r.URL.RawQuery)
						return &http.Response{
							StatusCode: http.StatusOK,
							// Example response taken from https://docs.gitlab.com/ee/api/oauth2.html#authorization-code-flow
							Body: io.NopCloser(strings.NewReader(`
{
 "access_token": "de6780bc506a0446309bd9362820ba8aed28aa506c71eedbe1c5c4f9dd350e54",
 "token_type": "bearer",
 "expires_in": 7200,
 "refresh_token": "8257e65c97202ed1726cf9571600918f3bffb2544b26e00a61df9897668c33a1",
 "created_at": 1607635748
}
`)),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.ExchangeOAuthToken(ctx, "",
		&common.OAuthExchange{
			ClientID:     "test_client_id",
			ClientSecret: "test_client_secret",
			Code:         "test_code",
			RedirectURL:  "http://localhost:3000",
		},
	)
	require.NoError(t, err)

	want := &vcs.OAuthToken{
		AccessToken:  "de6780bc506a0446309bd9362820ba8aed28aa506c71eedbe1c5c4f9dd350e54",
		RefreshToken: "8257e65c97202ed1726cf9571600918f3bffb2544b26e00a61df9897668c33a1",
		ExpiresIn:    7200,
		CreatedAt:    1607635748,
		ExpiresTs:    1607642948,
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchAllRepositoryList(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
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
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.FetchAllRepositoryList(ctx, common.OauthContext{}, "")
	require.NoError(t, err)

	want := []*vcs.Repository{
		{
			ID:       4,
			Name:     "Diaspora Client",
			FullPath: "diaspora/diaspora-client",
			WebURL:   "http://example.com/diaspora/diaspora-client",
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchRepositoryFileList(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
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
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.FetchRepositoryFileList(ctx, common.OauthContext{}, "", "1", "main", "")
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

func TestProvider_CreateFile(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/api/v4/projects/1/repository/files/lib%2Fclass.rb", r.URL.RawPath)

						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						wantBody := `{"branch":"master","content":"some content","commit_message":"create a new file"}`
						assert.Equal(t, wantBody, string(body))
						return &http.Response{
							StatusCode: http.StatusOK,
							// Example response taken from https://docs.gitlab.com/ee/api/repository_files.html#create-new-file-in-repository
							Body: io.NopCloser(strings.NewReader(`
{
  "file_path": "app/project.rb",
  "branch": "master"
}
`)),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	err := p.CreateFile(
		ctx,
		common.OauthContext{},
		"",
		"1",
		"lib/class.rb",
		vcs.FileCommitCreate{
			Branch:        "master",
			Content:       "some content",
			CommitMessage: "create a new file",
		},
	)
	require.NoError(t, err)
}

func TestProvider_OverwriteFile(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/api/v4/projects/1/repository/files/lib%2Fclass.rb", r.URL.RawPath)

						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						wantBody := `{"branch":"master","content":"some content","commit_message":"update file","last_commit_id":"7638417db6d59f3c431d3e1f261cc637155684cd"}`
						assert.Equal(t, wantBody, string(body))
						return &http.Response{
							StatusCode: http.StatusOK,
							// Example response taken from https://docs.gitlab.com/ee/api/repository_files.html#update-existing-file-in-repository
							Body: io.NopCloser(strings.NewReader(`
{
  "file_path": "app/project.rb",
  "branch": "master"
}
`)),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	err := p.OverwriteFile(
		ctx,
		common.OauthContext{},
		"",
		"1",
		"lib/class.rb",
		vcs.FileCommitCreate{
			Branch:        "master",
			Content:       "some content",
			CommitMessage: "update file",
			LastCommitID:  "7638417db6d59f3c431d3e1f261cc637155684cd",
		},
	)
	require.NoError(t, err)
}

func TestProvider_ReadFileMeta(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
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
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.ReadFileMeta(ctx, common.OauthContext{}, "", "1", "app/models/key.rb", "master")
	require.NoError(t, err)

	want := &vcs.FileMeta{
		LastCommitID: "27329d3afac51fbf2762428e12f2635d1137c549",
	}
	assert.Equal(t, want, got)
}

func TestProvider_ReadFileContent(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
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
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.ReadFileContent(ctx, common.OauthContext{}, "", "1", "app/models/key.rb", "master")
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
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
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
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.CreateWebhook(ctx, common.OauthContext{}, "", "1", []byte(""))
	require.NoError(t, err)
	assert.Equal(t, "1", got)
}

func TestProvider_PatchWebhook(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/api/v4/projects/1/hooks/1", r.URL.Path)
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("")),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	err := p.PatchWebhook(ctx, common.OauthContext{}, "", "1", "1", []byte(""))
	require.NoError(t, err)
}

func TestProvider_DeleteWebhook(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/api/v4/projects/1/hooks/1", r.URL.Path)
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("")),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	err := p.DeleteWebhook(ctx, common.OauthContext{}, "", "1", "1")
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
					{"error":"invalid_token","error_description":"Token is expired. You can either do re-authorization or token refresh."}
					`)),
					}, nil
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					// Example response taken from https://docs.gitlab.com/ee/api/oauth2.html#authorization-code-with-proof-key-for-code-exchange-pkce
					Body: io.NopCloser(strings.NewReader(`
{
 "access_token": "de6780bc506a0446309bd9362820ba8aed28aa506c71eedbe1c5c4f9dd350e54",
 "token_type": "bearer",
 "expires_in": 7200,
 "refresh_token": "8257e65c97202ed1726cf9571600918f3bffb2544b26e00a61df9897668c33a1",
 "created_at": 1607635748
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
		"https://gitlab.example.com/api/v4/users/octocat",
		&token,
		tokenRefresher(
			"https://gitlab.example.com",
			oauthContext{},
			refresher,
		),
	)
	require.NoError(t, err)
	assert.Equal(t, "de6780bc506a0446309bd9362820ba8aed28aa506c71eedbe1c5c4f9dd350e54", token)
	assert.True(t, calledRefresher)
}

func TestProvider_GetBranch(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
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
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.GetBranch(ctx, common.OauthContext{}, "", "1", "main")
	require.NoError(t, err)

	want := &vcs.BranchInfo{
		Name:         "main",
		LastCommitID: "7b5c3cc8be40ee161ae89a06bba6229da1032a0c",
	}
	assert.Equal(t, want, got)
}

func TestProvider_CreateBranch(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "POST", r.Method)
						assert.Equal(t, "/api/v4/projects/1/repository/branches", r.URL.Path)
						return &http.Response{
							StatusCode: http.StatusOK,
							// Example response taken from https://docs.gitlab.com/ee/api/branches.html#create-repository-branch
							Body: io.NopCloser(strings.NewReader(`
{
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
  },
  "name": "newbranch",
  "merged": false,
  "protected": false,
  "default": false,
  "developers_can_push": false,
  "developers_can_merge": false,
  "can_push": true,
  "web_url": "https://gitlab.example.com/my-group/my-project/-/tree/newbranch"
}
`)),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	err := p.CreateBranch(ctx, common.OauthContext{}, "", "1", &vcs.BranchInfo{
		Name:         "newbranch",
		LastCommitID: "7b5c3cc8be40ee161ae89a06bba6229da1032a0c",
	})
	require.NoError(t, err)
}

func TestProvider_CreatePullRequest(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/api/v4/projects/1/merge_requests", r.URL.Path)
						return &http.Response{
							StatusCode: http.StatusOK,
							// Example response taken from https://docs.gitlab.com/ee/api/merge_requests.html#create-mr
							Body: io.NopCloser(strings.NewReader(`
{
  "id": 1,
  "iid": 1,
  "project_id": 3,
  "title": "test1",
  "description": "fixed login page css paddings",
  "state": "merged",
  "created_at": "2017-04-29T08:46:00Z",
  "updated_at": "2017-04-29T08:46:00Z",
  "target_branch": "master",
  "source_branch": "test1",
  "upvotes": 0,
  "downvotes": 0,
  "author": {
    "id": 1,
    "name": "Administrator",
    "username": "admin",
    "state": "active",
    "avatar_url": null,
    "web_url" : "https://gitlab.example.com/admin"
  },
  "assignee": {
    "id": 1,
    "name": "Administrator",
    "username": "admin",
    "state": "active",
    "avatar_url": null,
    "web_url" : "https://gitlab.example.com/admin"
  },
  "source_project_id": 2,
  "target_project_id": 3,
  "labels": [
    "Community contribution",
    "Manage"
  ],
  "draft": false,
  "work_in_progress": false,
  "merge_when_pipeline_succeeds": true,
  "merge_status": "can_be_merged",
  "merge_error": null,
  "sha": "8888888888888888888888888888888888888888",
  "merge_commit_sha": null,
  "squash_commit_sha": null,
  "user_notes_count": 1,
  "discussion_locked": null,
  "should_remove_source_branch": true,
  "force_remove_source_branch": false,
  "allow_collaboration": false,
  "allow_maintainer_to_push": false,
  "web_url": "http://gitlab.example.com/my-group/my-project/merge_requests/1",
  "references": {
    "short": "!1",
    "relative": "!1",
    "full": "my-group/my-project!1"
  },
  "time_stats": {
    "time_estimate": 0,
    "total_time_spent": 0,
    "human_time_estimate": null,
    "human_total_time_spent": null
  },
  "squash": false,
  "subscribed": false,
  "changes_count": "1",
  "closed_by": null,
  "closed_at": null,
  "latest_build_started_at": "2018-09-07T07:27:38.472Z",
  "latest_build_finished_at": "2018-09-07T08:07:06.012Z",
  "first_deployed_to_production_at": null,
  "pipeline": {
    "id": 29626725,
    "sha": "2be7ddb704c7b6b83732fdd5b9f09d5a397b5f8f",
    "ref": "patch-28",
    "status": "success",
    "web_url": "https://gitlab.example.com/my-group/my-project/pipelines/29626725"
  },
  "diff_refs": {
    "base_sha": "c380d3acebd181f13629a25d2e2acca46ffe1e00",
    "head_sha": "2be7ddb704c7b6b83732fdd5b9f09d5a397b5f8f",
    "start_sha": "c380d3acebd181f13629a25d2e2acca46ffe1e00"
  },
  "diverged_commits_count": 2,
}
`)),
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	res, err := p.CreatePullRequest(ctx, common.OauthContext{}, "", "1", &vcs.PullRequestCreate{
		Title:                 "test1",
		Body:                  "fixed login page css paddings",
		Head:                  "test1",
		Base:                  "master",
		RemoveHeadAfterMerged: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "http://gitlab.example.com/my-group/my-project/merge_requests/1", res.URL)
}

func TestProvider_UpsertEnvironmentVariable(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
						switch r.URL.Path {
						case "/api/v4/projects/1/variables/1":
							if r.Method == "GET" {
								return &http.Response{
									StatusCode: http.StatusOK,
									// Example response taken from https://docs.gitlab.com/ee/api/project_level_variables.html#get-a-single-variable
									Body: io.NopCloser(strings.NewReader(`
{
    "variable_type": "env_var",
    "key": "1",
    "value": "new value",
    "protected": false,
    "masked": false,
    "environment_scope": "*"
}
`)),
								}, nil
							} else if r.Method == "PUT" {
								return &http.Response{
									StatusCode: http.StatusOK,
									// Example response taken from https://docs.gitlab.com/ee/api/project_level_variables.html#update-a-variable
									Body: io.NopCloser(strings.NewReader(`
{
    "variable_type": "env_var",
    "key": "1",
    "value": "new value",
    "protected": false,
    "masked": false,
    "environment_scope": "*"
}
`)),
								}, nil
							}
						case "/api/v4/projects/1/variables":
							assert.Equal(t, "POST", r.Method)
							return &http.Response{
								StatusCode: http.StatusOK,
								// Example response taken from https://docs.gitlab.com/ee/api/project_level_variables.html#create-a-variable
								Body: io.NopCloser(strings.NewReader(`
{
    "variable_type": "env_var",
    "key": "1",
    "value": "new value",
    "protected": false,
    "masked": false,
    "environment_scope": "*"
}
`)),
							}, nil
						}

						return nil, errors.Errorf("Invalid request. %s: %s", r.Method, r.URL.Path)
					},
				},
			},
		},
	)

	ctx := context.Background()
	err := p.UpsertEnvironmentVariable(ctx, common.OauthContext{}, "", "1", "1", "new value")
	require.NoError(t, err)
}

func TestProvider_ListPullRequestFile(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &common.MockRoundTripper{
					MockRoundTrip: func(r *http.Request) (*http.Response, error) {
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
				},
			},
		},
	)
	ctx := context.Background()
	got, err := p.ListPullRequestFile(ctx, common.OauthContext{}, "", "1", "1")
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
