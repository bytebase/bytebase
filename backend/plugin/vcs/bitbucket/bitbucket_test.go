package bitbucket

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
		assert.Equal(t, "/2.0/user/permissions/repositories", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-user-permissions-repositories-get
			Body: io.NopCloser(strings.NewReader(`
{
  "pagelen": 10,
  "values": [
    {
      "type": "repository_permission",
      "user": {
        "type": "user",
        "nickname": "evzijst",
        "display_name": "Erik van Zijst",
        "uuid": "{d301aafa-d676-4ee0-88be-962be7417567}"
      },
      "repository": {
        "type": "repository",
        "name": "geordi",
        "full_name": "bitbucket/geordi",
        "uuid": "{85d08b4e-571d-44e9-a507-fa476535aa98}"
      },
      "permission": "admin"
    }
  ],
  "page": 1,
  "size": 1
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.FetchAllRepositoryList(ctx, &common.OauthContext{}, bitbucketCloudURL)
	require.NoError(t, err)

	want := []*vcs.Repository{
		{
			ID:       "{85d08b4e-571d-44e9-a507-fa476535aa98}",
			Name:     "geordi",
			FullPath: "bitbucket/geordi",
			WebURL:   "https://bitbucket.org/bitbucket/geordi",
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_FetchRepositoryFileList(t *testing.T) {
	// Example response derived from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#directory-listings
	const response = `
{
  "pagelen": 10,
  "values": [
    {
      "links": {
        "self": {
          "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/src/eefd5ef5d3df01aed629f650959d6706d54cd335/tests/__init__.py"
        },
        "meta": {
          "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/src/eefd5ef5d3df01aed629f650959d6706d54cd335/tests/__init__.py?format=meta"
        }
      },
      "path": "tests/__init__.py",
      "commit": {
        "type": "commit",
        "hash": "eefd5ef5d3df01aed629f650959d6706d54cd335",
        "links": {
          "self": {
            "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/commit/eefd5ef5d3df01aed629f650959d6706d54cd335"
          },
          "html": {
            "href": "https://bitbucket.org/atlassian/bbql/commits/eefd5ef5d3df01aed629f650959d6706d54cd335"
          }
        }
      },
      "attributes": [],
      "type": "commit_file",
      "size": 0
    }
  ],
  "page": 1,
  "size": 1
}
`
	t.Run("no path prefix", func(t *testing.T) {
		ctx := context.Background()
		p := newMockProvider(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "/2.0/repositories/atlassian/bbql/src/eefd5ef/", r.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
			}, nil
		})

		got, err := p.FetchRepositoryFileList(ctx, &common.OauthContext{}, bitbucketCloudURL, "atlassian/bbql", "eefd5ef", "")
		require.NoError(t, err)

		want := []*vcs.RepositoryTreeNode{
			{
				Path: "tests/__init__.py",
				Type: "commit_file",
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("has path prefix", func(t *testing.T) {
		ctx := context.Background()
		p := newMockProvider(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "/2.0/repositories/atlassian/bbql/src/eefd5ef/tests", r.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
			}, nil
		})

		got, err := p.FetchRepositoryFileList(ctx, &common.OauthContext{}, bitbucketCloudURL, "atlassian/bbql", "eefd5ef", "tests")
		require.NoError(t, err)

		// Non-blob type should be excluded
		want := []*vcs.RepositoryTreeNode{
			{
				Path: "tests/__init__.py",
				Type: "commit_file",
			},
		}
		assert.Equal(t, want, got)
	})
}

func TestProvider_ReadFileMeta(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/atlassian/bbql/src/eefd5ef/tests/__init__.py", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response derived from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#file-meta-data
			Body: io.NopCloser(strings.NewReader(`
{
  "links": {
    "self": {
      "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/src/eefd5ef5d3df01aed629f650959d6706d54cd335/tests/__init__.py"
    },
    "meta": {
      "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/src/eefd5ef5d3df01aed629f650959d6706d54cd335/tests/__init__.py?format=meta"
    }
  },
  "path": "tests/__init__.py",
  "commit": {
    "type": "commit",
    "hash": "eefd5ef5d3df01aed629f650959d6706d54cd335",
    "links": {
      "self": {
        "href": "https://api.bitbucket.org/2.0/repositories/atlassian/bbql/commit/eefd5ef5d3df01aed629f650959d6706d54cd335"
      },
      "html": {
        "href": "https://bitbucket.org/atlassian/bbql/commits/eefd5ef5d3df01aed629f650959d6706d54cd335"
      }
    }
  },
  "attributes": [],
  "type": "commit_file",
  "size": 100
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.ReadFileMeta(ctx, &common.OauthContext{}, bitbucketCloudURL, "atlassian/bbql", "tests/__init__.py", vcs.RefInfo{
		RefType: vcs.RefTypeCommit,
		RefName: "eefd5ef",
	})
	require.NoError(t, err)

	want := &vcs.FileMeta{
		Name:         "__init__.py",
		Path:         "tests/__init__.py",
		Size:         100,
		LastCommitID: "eefd5ef5d3df01aed629f650959d6706d54cd335",
	}
	assert.Equal(t, want, got)
}

func TestProvider_ReadFileContent(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/atlassian/bbql/src/eefd5ef/tests/__init__.py", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("#!/bin/sh\nhalt")),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.ReadFileContent(ctx, &common.OauthContext{}, bitbucketCloudURL, "atlassian/bbql", "tests/__init__.py", vcs.RefInfo{
		RefType: vcs.RefTypeCommit,
		RefName: "eefd5ef",
	})
	require.NoError(t, err)

	want := "#!/bin/sh\nhalt"
	assert.Equal(t, want, got)
}

func TestProvider_GetBranch(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/atlassian/aui/refs/branches/master", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-refs/#api-repositories-workspace-repo-slug-refs-branches-name-get
			Body: io.NopCloser(strings.NewReader(`
{
      "name": "master",
      "links": {
        "commits": {
          "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui/commits/master"
        },
        "self": {
          "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui/refs/branches/master"
        },
        "html": {
          "href": "https://bitbucket.org/atlassian/aui/branch/master"
        }
      },
      "default_merge_strategy": "squash",
      "merge_strategies": [
        "merge_commit",
        "squash",
        "fast_forward"
      ],
      "type": "branch",
      "target": {
        "hash": "e7d158ff7ed5538c28f94cd97a9ad569680fc94e",
        "repository": {
          "links": {
            "self": {
              "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui"
            },
            "html": {
              "href": "https://bitbucket.org/atlassian/aui"
            },
            "avatar": {
              "href": "https://bytebucket.org/ravatar/%7B585074de-7b60-4fd1-81ed-e0bc7fafbda5%7D?ts=86317"
            }
          },
          "type": "repository",
          "name": "aui",
          "full_name": "atlassian/aui",
          "uuid": "{585074de-7b60-4fd1-81ed-e0bc7fafbda5}"
        },
        "links": {
          "self": {
            "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui/commit/e7d158ff7ed5538c28f94cd97a9ad569680fc94e"
          },
          "comments": {
            "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui/commit/e7d158ff7ed5538c28f94cd97a9ad569680fc94e/comments"
          },
          "patch": {
            "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui/patch/e7d158ff7ed5538c28f94cd97a9ad569680fc94e"
          },
          "html": {
            "href": "https://bitbucket.org/atlassian/aui/commits/e7d158ff7ed5538c28f94cd97a9ad569680fc94e"
          },
          "diff": {
            "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui/diff/e7d158ff7ed5538c28f94cd97a9ad569680fc94e"
          },
          "approve": {
            "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui/commit/e7d158ff7ed5538c28f94cd97a9ad569680fc94e/approve"
          },
          "statuses": {
            "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui/commit/e7d158ff7ed5538c28f94cd97a9ad569680fc94e/statuses"
          }
        },
        "author": {
          "raw": "psre-renovate-bot <psre-renovate-bot@atlassian.com>",
          "type": "author",
          "user": {
            "display_name": "psre-renovate-bot",
            "uuid": "{250a442a-3ab3-4fcb-87c3-3c8f3df65ec7}",
            "links": {
              "self": {
                "href": "https://api.bitbucket.org/2.0/users/%7B250a442a-3ab3-4fcb-87c3-3c8f3df65ec7%7D"
              },
              "html": {
                "href": "https://bitbucket.org/%7B250a442a-3ab3-4fcb-87c3-3c8f3df65ec7%7D/"
              },
              "avatar": {
                "href": "https://secure.gravatar.com/avatar/6972ee037c9f36360170a86f544071a2?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FP-3.png"
              }
            },
            "nickname": "Renovate Bot",
            "type": "user",
            "account_id": "5d5355e8c6b9320d9ea5b28d"
          }
        },
        "parents": [
          {
            "hash": "eab868a309e75733de80969a7bed1ec6d4651e06",
            "type": "commit",
            "links": {
              "self": {
                "href": "https://api.bitbucket.org/2.0/repositories/atlassian/aui/commit/eab868a309e75733de80969a7bed1ec6d4651e06"
              },
              "html": {
                "href": "https://bitbucket.org/atlassian/aui/commits/eab868a309e75733de80969a7bed1ec6d4651e06"
              }
            }
          }
        ],
        "date": "2021-04-12T06:44:38+00:00",
        "message": "Merged in issue/NONE-renovate-master-babel-monorepo (pull request #2883)", 
        "type": "commit"
      }
} 
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.GetBranch(ctx, &common.OauthContext{}, bitbucketCloudURL, "atlassian/aui", "master")
	require.NoError(t, err)

	want := &vcs.BranchInfo{
		Name:         "master",
		LastCommitID: "e7d158ff7ed5538c28f94cd97a9ad569680fc94e",
	}
	assert.Equal(t, want, got)
}

func TestProvider_ListPullRequestFile(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/1/pullrequests/10086/diffstat", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			// Example response taken from https://developer.atlassian.com/cloud/bitbucket/rest/api-group-commits/#sample-output
			Body: io.NopCloser(strings.NewReader(`
{
    "pagelen": 500,
    "values": [
        {
            "type": "diffstat",
            "status": "modified",
            "lines_removed": 1,
            "lines_added": 2,
            "old": {
                "path": "setup.py",
                "escaped_path": "setup.py",
                "type": "commit_file",
                "links": {
                    "self": {
                        "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/src/e1749643d655d7c7014001a6c0f58abaf42ad850/setup.py"
                    }
                }
            },
            "new": {
                "path": "setup.py",
                "escaped_path": "setup.py",
                "type": "commit_file",
                "links": {
                    "self": {
                        "href": "https://api.bitbucket.org/2.0/repositories/bitbucket/geordi/src/d222fa235229c55dad20b190b0b571adf737d5a6/setup.py"
                    }
                }
            }
        }
    ],
    "page": 1,
    "size": 1
}
`)),
		}, nil
	},
	)
	ctx := context.Background()
	got, err := p.ListPullRequestFile(ctx, &common.OauthContext{}, bitbucketCloudURL, "1", "10086")
	require.NoError(t, err)

	want := []*vcs.PullRequestFile{
		{
			Path:         "setup.py",
			LastCommitID: "d222fa235229c55dad20b190b0b571adf737d5a6",
			IsDeleted:    false,
		},
	}
	assert.Equal(t, want, got)
}

func TestProvider_CreateWebhook(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/2.0/repositories/1/hooks", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`
{
  "uuid": "6611a7e9-6890-4e8e-84c5-b9707397da8b",
  "description": "Webhook Description",
  "url": "https://example.com/",
  "subject_type": "repository",
  "active": true,
  "events": [
	"repo:push",
	"issue:created",
	"issue:updated"
  ]
}
`)),
		}, nil
	},
	)

	ctx := context.Background()
	got, err := p.CreateWebhook(
		ctx,
		&common.OauthContext{},
		bitbucketCloudURL,
		"1",
		[]byte(`
{
  "description": "Webhook Description",
  "url": "https://example.com/",
  "active": true,
  "events": [
	"repo:push",
	"issue:created",
	"issue:updated"
  ]
}`),
	)
	require.NoError(t, err)
	assert.Equal(t, "6611a7e9-6890-4e8e-84c5-b9707397da8b", got)
}

func TestProvider_DeleteWebhook(t *testing.T) {
	p := newMockProvider(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/2.0/repositories/1/hooks/1", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	},
	)

	ctx := context.Background()
	err := p.DeleteWebhook(ctx, &common.OauthContext{}, bitbucketCloudURL, "1", "1")
	require.NoError(t, err)
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
