package neon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewClient("test-secret-key")
	client.baseURL = server.URL
	return client
}

func TestResolve(t *testing.T) {
	for _, manual := range []bool{false, true} {
		t.Run(fmt.Sprint(manual), func(t *testing.T) {
			reveals := 0
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "Bearer test-secret-key", r.Header.Get("Authorization"))
				switch r.URL.Path {
				case "/projects/project/branches/br-test/databases":
					_, _ = fmt.Fprint(w, `{"databases":[{"name":"app","owner_name":"owner"}]}`)
				case "/projects/project/branches/br-test/roles":
					_, _ = fmt.Fprint(w, `{"roles":[{"name":"owner"}]}`)
				case "/projects/project/branches/br-test/endpoints":
					_, _ = fmt.Fprint(w, `{"endpoints":[{"id":"ep-test","branch_id":"br-test","host":"ep-test.us-east-2.aws.neon.tech","type":"read_write"}]}`)
				case "/projects/project/branches/br-test/roles/owner/reveal_password":
					reveals++
					_, _ = fmt.Fprint(w, `{"password":"existing-secret"}`)
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					w.WriteHeader(500)
				}
			})
			var password *string
			if manual {
				p := "manual-secret"
				password = &p
			}
			connection, err := client.Resolve(context.Background(), "project", "br-test", "owner", "app", password)
			require.NoError(t, err)
			require.Equal(t, "ep-test.us-east-2.aws.neon.tech", connection.Host)
			require.Equal(t, "5432", connection.Port)
			require.Equal(t, "owner", connection.Username)
			require.Equal(t, "app", connection.Database)
			if manual {
				require.Equal(t, 0, reveals)
				require.Equal(t, "manual-secret", connection.Password)
			} else {
				require.Equal(t, 1, reveals)
				require.Equal(t, "existing-secret", connection.Password)
			}
			_, err = client.Resolve(context.Background(), "project", "br-test", "owner", "missing", password)
			require.ErrorIs(t, err, ErrDatabaseNotFound)
			_, err = client.Resolve(context.Background(), "project", "br-test", "missing", "app", password)
			require.ErrorIs(t, err, ErrRoleNotFound)
			emptyPassword := ""
			_, err = client.Resolve(context.Background(), "project", "br-test", "owner", "app", &emptyPassword)
			require.ErrorIs(t, err, ErrPasswordRequired)
		})
	}
}

func TestAPIErrorsDoNotExposeSecrets(t *testing.T) {
	for _, status := range []int{401, 403, 404, 412, 429, 500, 302} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(status)
				_, _ = fmt.Fprint(w, `{"message":"test-secret-key existing-secret"}`)
			})
			_, err := client.GetRolePassword(context.Background(), "project", "branch", "owner")
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, status, apiErr.StatusCode)
			require.NotContains(t, err.Error(), "secret")
			require.Equal(t, 1, calls)
		})
	}
}

func TestRedirectDoesNotSendKey(t *testing.T) {
	received := false
	destination := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { received = true }))
	defer destination.Close()
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, http.StatusFound) })
	_, err := client.ListOrganizations(context.Background())
	require.Error(t, err)
	require.False(t, received)
}

func TestPagination(t *testing.T) {
	calls := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "org-test", r.URL.Query().Get("org_id"))
		switch r.URL.Query().Get("cursor") {
		case "":
			_, _ = fmt.Fprint(w, `{"projects":[{"id":"p1","name":"one"}],"pagination":{"cursor":"opaque + /"}}`)
		case "opaque + /":
			_, _ = fmt.Fprint(w, `{"projects":[{"id":"p2","name":"two"}],"pagination":{"cursor":"last"}}`)
		case "last":
			_, _ = fmt.Fprint(w, `{"projects":[]}`)
		default:
			t.Error("invalid cursor")
		}
	})
	projects, err := client.ListProjects(context.Background(), "org-test")
	require.NoError(t, err)
	require.Len(t, projects, 2)
	require.Equal(t, 3, calls)
}

func TestBranchesNextPagination(t *testing.T) {
	calls := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "created_at", r.URL.Query().Get("sort_by"))
		if r.URL.Query().Get("cursor") == "" {
			_, _ = fmt.Fprint(w, `{"branches":[{"id":"br-first","name":"first"}],"pagination":{"next":"second"}}`)
		} else {
			_, _ = fmt.Fprint(w, `{"branches":[{"id":"br-second","name":"second"}]}`)
		}
	})
	branches, err := client.ListBranches(context.Background(), "project")
	require.NoError(t, err)
	require.Len(t, branches, 2)
	require.Equal(t, 2, calls)
}

func TestInvalidResponses(t *testing.T) {
	for _, body := range []string{`<html>secret</html>`, `{}`, `null`, `{"projects":null}`, `{"projects":[],"unavailable_project_ids":["p1"]}`} {
		t.Run(body, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, body) })
			_, err := client.ListProjects(context.Background(), "")
			require.Error(t, err)
			require.NotContains(t, err.Error(), "secret")
		})
	}
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"projects":[{"id":"p1","name":"first"}],"pagination":{"cursor":"same"}}`)
	})
	_, err := client.ListProjects(context.Background(), "")
	require.ErrorContains(t, err, "pagination")
}

func TestDiscoveryPartialBranchFailure(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me/organizations":
			_, _ = fmt.Fprint(w, `{"organizations":[{"id":"org-test"}]}`)
		case "/projects":
			require.Equal(t, "org-test", r.URL.Query().Get("org_id"))
			_, _ = fmt.Fprint(w, `{"projects":[{"id":"project","name":"My project"}]}`)
		case "/projects/project/branches":
			_, _ = fmt.Fprint(w, `{"branches":[{"id":"br-first","name":"main"},{"id":"br-second","name":"dev"}]}`)
		case "/projects/project/branches/br-first/databases":
			w.WriteHeader(403)
		case "/projects/project/branches/br-second/databases":
			_, _ = fmt.Fprint(w, `{"databases":[{"name":"app","owner_name":"owner"}]}`)
		case "/projects/project/branches/br-second/roles":
			_, _ = fmt.Fprint(w, `{"roles":[{"name":"owner","password":"should-not-be-retained"}]}`)
		case "/projects/project/branches/br-second/endpoints":
			_, _ = fmt.Fprint(w, `{"endpoints":[{"branch_id":"br-second","host":"ep-test.aws.neon.tech","type":"read_write"}]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(500)
		}
	})
	page, err := client.DiscoverPage(context.Background(), "", 50, "")
	require.NoError(t, err)
	branches := page.Branches
	require.Len(t, branches, 2)
	require.NotEmpty(t, branches[0].Error)
	require.Empty(t, branches[1].Error)
	require.True(t, branches[1].EndpointAvailable)
	require.Equal(t, "My project", branches[1].ProjectName)
	require.NotContains(t, fmt.Sprint(branches), "should-not-be-retained")
}

func TestDirectEndpoint(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "neon.tech.evil.com", "ep-test-pooler.aws.neon.tech", "evil@ep.aws.neon.tech", "ep.aws.neon.tech/path"} {
		_, err := directEndpoint([]Endpoint{{Host: host, Type: "read_write", BranchID: "br-test"}}, "br-test")
		require.Error(t, err)
	}
	for _, endpoint := range []Endpoint{{Host: "ep.aws.neon.tech", Type: "read_only", BranchID: "br-test"}, {Host: "ep.aws.neon.tech", Type: "read_write", BranchID: "br-other"}, {Host: "ep.aws.neon.tech", Type: "read_write", BranchID: "br-test", Disabled: true}} {
		_, err := directEndpoint([]Endpoint{endpoint}, "br-test")
		require.Error(t, err)
	}
}

func TestCancellationAndInput(t *testing.T) {
	client := testClient(t, func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := client.ListOrganizations(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	_, err = client.ListBranches(context.Background(), "../secrets")
	require.ErrorIs(t, err, ErrInvalidArgument)
	client.apiKey = "bad\r\nkey"
	_, err = client.ListOrganizations(context.Background())
	require.Error(t, err)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestTransportErrorIsRedacted(t *testing.T) {
	client := NewClient("test-secret-key")
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, baseURL+"/users/me/organizations", r.URL.String())
		return nil, errors.New("test-secret-key")
	})
	_, err := client.ListOrganizations(context.Background())
	require.Error(t, err)
	require.False(t, strings.Contains(err.Error(), "secret"))
}

func TestPaginationLimit(t *testing.T) {
	calls := 0
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = fmt.Fprintf(w, `{"branches":[{"id":"br-test","name":"main"}],"pagination":{"next":"page-%d"}}`, calls)
	})
	_, err := client.ListBranches(context.Background(), "project")
	require.ErrorContains(t, err, "page limit")
	require.Equal(t, maxPages, calls)
}

func TestResponseSizeLimit(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, strings.Repeat(" ", 4*1024*1024+1)) })
	_, err := client.ListOrganizations(context.Background())
	require.ErrorContains(t, err, "read Neon API response")
}

func TestRevealPasswordEscapesRole(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/projects/project/branches/branch/roles/a%2Fb%3F%23/reveal_password", r.URL.EscapedPath())
		require.Empty(t, r.URL.RawQuery)
		_, _ = fmt.Fprint(w, `{"password":"unchanged"}`)
	})
	password, err := client.GetRolePassword(context.Background(), "project", "branch", "a/b?#")
	require.NoError(t, err)
	require.Equal(t, "unchanged", password)
}

func TestDiscoverProjectWithScopedKey(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/project":
			_, _ = fmt.Fprint(w, `{"project":{"id":"project","name":"Scoped project"}}`)
		case "/projects/project/branches":
			_, _ = fmt.Fprint(w, `{"branches":[{"id":"br-main","name":"main"}]}`)
		case "/projects/project/branches/br-main/databases":
			_, _ = fmt.Fprint(w, `{"databases":[{"name":"app","owner_name":"owner"}]}`)
		case "/projects/project/branches/br-main/roles":
			_, _ = fmt.Fprint(w, `{"roles":[{"name":"owner"}]}`)
		case "/projects/project/branches/br-main/endpoints":
			_, _ = fmt.Fprint(w, `{"endpoints":[{"branch_id":"br-main","host":"ep-test.aws.neon.tech","type":"read_write"}]}`)
		default:
			t.Errorf("project-scoped discovery must not enumerate organizations/projects: %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	page, err := client.DiscoverPage(context.Background(), "project", 50, "")
	require.NoError(t, err)
	branches := page.Branches
	require.Len(t, branches, 1)
	require.Equal(t, "Scoped project", branches[0].ProjectName)
	require.Equal(t, "project", branches[0].ProjectID)
	require.True(t, branches[0].EndpointAvailable)
	require.Empty(t, branches[0].Error)
}

func TestDiscoverProjectRejectsInvalidResponse(t *testing.T) {
	for _, body := range []string{`{}`, `{"project":{"id":"other","name":"Other"}}`, `{"project":{"id":"project"}}`} {
		t.Run(body, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, body) })
			_, err := client.DiscoverPage(context.Background(), "project", 50, "")
			require.ErrorContains(t, err, "invalid Neon project response")
		})
	}
}
