package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
)

type mockRoundTripper struct {
	roundTrip func(r *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return m.roundTrip(r)
}

func TestProvider_FetchUserInfo(t *testing.T) {
	p := newProvider(
		vcs.ProviderConfig{
			Client: &http.Client{
				Transport: &mockRoundTripper{
					roundTrip: func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/users/octocat", r.URL.Path)
						return &http.Response{
							StatusCode: http.StatusOK,
							Body: io.NopCloser(strings.NewReader(`
{
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
  "site_admin": false,
  "name": "monalisa octocat",
  "company": "GitHub",
  "blog": "https://github.com/blog",
  "location": "San Francisco",
  "email": "octocat@github.com",
  "hireable": false,
  "bio": "There once was...",
  "twitter_username": "monatheoctocat",
  "public_repos": 2,
  "public_gists": 1,
  "followers": 20,
  "following": 0,
  "created_at": "2008-01-14T04:33:35Z",
  "updated_at": "2008-01-14T04:33:35Z",
  "private_gists": 81,
  "total_private_repos": 100,
  "owned_private_repos": 100,
  "disk_usage": 10000,
  "collaborators": 8,
  "two_factor_authentication": true,
  "plan": {
    "name": "Medium",
    "space": 400,
    "private_repos": 20,
    "collaborators": 0
  }
}
`)),
							Request: r,
						}, nil
					},
				},
			},
		},
	)

	ctx := context.Background()
	got, err := p.FetchUserInfo(ctx, common.OauthContext{}, "", "octocat")
	require.NoError(t, err)

	want := &vcs.UserInfo{
		PublicEmail: "octocat@github.com",
		Name:        "monalisa octocat",
	}
	assert.Equal(t, want, got)
}

func TestOAuth_RefreshToken(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &mockRoundTripper{
			roundTrip: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "/login/oauth/access_token", r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
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
					Request: r,
				}, nil
			},
		},
	}
	token := "old"

	calledRefresher := false
	refresher := func(_, _ string, _ int64) error {
		calledRefresher = true
		return nil
	}

	called := 0
	_, _, err := retry(ctx, client, &token, oauthContext{}, refresher,
		func() (*http.Response, error) {
			called++
			if called == 1 {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(`
{"error":"invalid_token","error_description":"Token is expired. You can either do re-authorization or token refresh."}
`)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}, nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "ghu_16C7e42F292c6912E7710c838347Ae178B4a", token)
	assert.True(t, calledRefresher)
}

func TestRetry_Exceeded(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &mockRoundTripper{
			roundTrip: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "/login/oauth/access_token", r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
					Request:    r,
				}, nil
			},
		},
	}
	_ = client
	token := "old"
	_, _, err := retry(ctx, client, &token, oauthContext{},
		func(_, _ string, _ int64) error { return nil },
		func() (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader(`
{"error":"invalid_token","error_description":"Token is expired. You can either do re-authorization or token refresh."}
`)),
			}, nil
		},
	)
	wantErr := `retries exceeded for oauth refresher with status code 400 and body ""`
	gotErr := fmt.Sprintf("%v", err)
	assert.Equal(t, wantErr, gotErr)
}
