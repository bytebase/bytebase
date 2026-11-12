package neon

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDiscoveryPageEnrichesOnlyReturnedBranches(t *testing.T) {
	var inspected []string
	var mu sync.Mutex
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/project":
			_, _ = fmt.Fprint(w, `{"project":{"id":"project","name":"Project"}}`)
		case "/projects/project/branches":
			if r.URL.Query().Get("cursor") == "" {
				_, _ = fmt.Fprint(w, `{"branches":[{"id":"b1","name":"One"},{"id":"b2","name":"Two"}],"pagination":{"next":"next"}}`)
			} else {
				require.Equal(t, "next", r.URL.Query().Get("cursor"))
				_, _ = fmt.Fprint(w, `{"branches":[{"id":"b3","name":"Three"}]}`)
			}
		default:
			parts := strings.Split(r.URL.Path, "/")
			id := parts[4]
			switch parts[5] {
			case "databases":
				mu.Lock()
				inspected = append(inspected, id)
				mu.Unlock()
				_, _ = fmt.Fprint(w, `{"databases":[]}`)
			case "roles":
				_, _ = fmt.Fprint(w, `{"roles":[]}`)
			case "endpoints":
				_, _ = fmt.Fprintf(w, `{"endpoints":[{"branch_id":%q,"host":"ep.aws.neon.tech","type":"read_write"}]}`, id)
			default:
				t.Errorf("unexpected path %s", r.URL.Path)
			}
		}
	})
	page, err := client.DiscoverPage(context.Background(), "project", 1, "")
	require.NoError(t, err)
	require.Equal(t, []string{"b1"}, inspected)
	require.NotEmpty(t, page.NextPageToken)
	raw, err := base64.RawURLEncoding.DecodeString(page.NextPageToken)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "test-secret-key")
	page, err = client.DiscoverPage(context.Background(), "project", 2, page.NextPageToken)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"b1", "b2", "b3"}, inspected)
	require.Len(t, page.Branches, 2)
	require.Empty(t, page.NextPageToken)
}

func TestDiscoveryContinuesAcrossProjectsAndOrganizations(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me/organizations":
			_, _ = fmt.Fprint(w, `{"organizations":[{"id":"org-a"},{"id":"org-b"}]}`)
		case "/projects":
			if r.URL.Query().Get("org_id") == "org-a" {
				if r.URL.Query().Get("cursor") == "" {
					_, _ = fmt.Fprint(w, `{"projects":[{"id":"denied","name":"Denied"}],"pagination":{"cursor":"more"}}`)
				} else {
					require.Equal(t, "more", r.URL.Query().Get("cursor"))
					_, _ = fmt.Fprint(w, `{"projects":[{"id":"empty","name":"Empty"}]}`)
				}
			} else {
				require.Equal(t, "org-b", r.URL.Query().Get("org_id"))
				_, _ = fmt.Fprint(w, `{"projects":[{"id":"good","name":"Good"}]}`)
			}
		case "/projects/denied/branches":
			w.WriteHeader(http.StatusForbidden)
		case "/projects/empty/branches":
			_, _ = fmt.Fprint(w, `{"branches":[]}`)
		case "/projects/good/branches":
			_, _ = fmt.Fprint(w, `{"branches":[{"id":"b1","name":"One"},{"id":"b2","name":"Two"}]}`)
		default:
			if strings.HasSuffix(r.URL.Path, "/databases") {
				w.WriteHeader(http.StatusForbidden)
			} else {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
		}
	})
	page, err := client.DiscoverPage(context.Background(), "", 1, "")
	require.NoError(t, err)
	require.Len(t, page.Failures, 1)
	require.Equal(t, "denied", page.Failures[0].ProjectID)
	require.Len(t, page.Branches, 1)
	require.Error(t, page.Branches[0].Error)
	page, err = client.DiscoverPage(context.Background(), "", 2, page.NextPageToken)
	require.NoError(t, err)
	require.Len(t, page.Branches, 1)
	require.Equal(t, "b2", page.Branches[0].ID)
	require.Empty(t, page.NextPageToken)
}

func TestDiscoverySharedErrorsDiscardPartialPage(t *testing.T) {
	for _, status := range []int{401, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/projects/project":
					_, _ = fmt.Fprint(w, `{"project":{"id":"project","name":"Project"}}`)
				case "/projects/project/branches":
					_, _ = fmt.Fprint(w, `{"branches":[{"id":"b1","name":"One"},{"id":"b2","name":"Two"}]}`)
				case "/projects/project/branches/b1/databases":
					w.WriteHeader(http.StatusForbidden)
				case "/projects/project/branches/b2/databases":
					w.Header().Set("Retry-After", "12")
					w.WriteHeader(status)
					_, _ = fmt.Fprint(w, "private-key private-password")
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
				}
			})
			page, err := client.DiscoverPage(context.Background(), "project", 50, "")
			require.Nil(t, page)
			var api *APIError
			require.ErrorAs(t, err, &api)
			require.Equal(t, status, api.StatusCode)
			require.Equal(t, 12*time.Second, api.RetryAfter)
			require.NotContains(t, err.Error(), "private")
		})
	}
}

func TestDiscoveryBoundsEmptyTraversal(t *testing.T) {
	calls := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/projects/project" {
			_, _ = fmt.Fprint(w, `{"project":{"id":"project","name":"Project"}}`)
			return
		}
		calls++
		_, _ = fmt.Fprintf(w, `{"branches":[],"pagination":{"next":"cursor-%d"}}`, calls)
	})
	page, err := client.DiscoverPage(context.Background(), "project", 50, "")
	require.NoError(t, err)
	require.Empty(t, page.Branches)
	require.Equal(t, 10, calls)
	require.NotEmpty(t, page.NextPageToken)
	_, err = client.DiscoverPage(context.Background(), "project", 50, page.NextPageToken)
	require.NoError(t, err)
	require.Equal(t, 20, calls)
}

func TestDiscoveryInvalidCursor(t *testing.T) {
	client := testClient(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("invalid input must not contact Neon") })
	for _, token := range []string{"invalid", base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"f":"other"}`)), base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"f":"project","j":-1}`)), strings.Repeat("a", 16385)} {
		_, err := client.DiscoverPage(context.Background(), "project", 50, token)
		require.ErrorIs(t, err, ErrInvalidPageToken)
	}
}

func TestDiscoveryCancellation(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/project":
			_, _ = fmt.Fprint(w, `{"project":{"id":"project","name":"Project"}}`)
		case "/projects/project/branches":
			_, _ = fmt.Fprint(w, `{"branches":[{"id":"b1","name":"One"}]}`)
		default:
			<-r.Context().Done()
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	page, err := client.DiscoverPage(ctx, "project", 50, "")
	require.Nil(t, page)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestDiscoveryEnrichmentConcurrency(t *testing.T) {
	var mu sync.Mutex
	active, peak := 0, 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/project":
			_, _ = fmt.Fprint(w, `{"project":{"id":"project","name":"Project"}}`)
		case "/projects/project/branches":
			_, _ = fmt.Fprint(w, `{"branches":[{"id":"b1","name":"One"},{"id":"b2","name":"Two"},{"id":"b3","name":"Three"},{"id":"b4","name":"Four"},{"id":"b5","name":"Five"}]}`)
		default:
			mu.Lock()
			active++
			peak = max(peak, active)
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			active--
			mu.Unlock()
			w.WriteHeader(http.StatusForbidden)
		}
	})
	page, err := client.DiscoverPage(context.Background(), "project", 50, "")
	require.NoError(t, err)
	require.Len(t, page.Branches, 5)
	require.LessOrEqual(t, peak, 4)
	require.Greater(t, peak, 1)
	for i, branch := range page.Branches {
		require.Equal(t, fmt.Sprintf("b%d", i+1), branch.ID)
	}
}

func TestDiscoveryEmptyProjectPageEndsOrganization(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me/organizations":
			_, _ = fmt.Fprint(w, `{"organizations":[{"id":"org-a"},{"id":"org-b"}]}`)
		case "/projects":
			if r.URL.Query().Get("org_id") == "org-b" {
				_, _ = fmt.Fprint(w, `{"projects":[{"id":"second","name":"Second"}]}`)
			} else if r.URL.Query().Get("cursor") == "" {
				_, _ = fmt.Fprint(w, `{"projects":[{"id":"first","name":"First"}],"pagination":{"cursor":"last"}}`)
			} else {
				require.Equal(t, "last", r.URL.Query().Get("cursor"))
				_, _ = fmt.Fprint(w, `{"projects":[],"pagination":{"cursor":"last"}}`)
			}
		case "/projects/first/branches", "/projects/second/branches":
			_, _ = fmt.Fprint(w, `{"branches":[{"id":"branch","name":"Branch"}]}`)
		default:
			if strings.HasSuffix(r.URL.Path, "/databases") {
				w.WriteHeader(http.StatusForbidden)
			} else {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
		}
	})
	page, err := client.DiscoverPage(context.Background(), "", 50, "")
	require.NoError(t, err)
	require.Len(t, page.Branches, 2)
	require.Equal(t, "first", page.Branches[0].ProjectID)
	require.Equal(t, "second", page.Branches[1].ProjectID)
	require.Empty(t, page.NextPageToken)
}

func TestDiscoveryReportsUnavailableProjectsAndContinues(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me/organizations":
			_, _ = fmt.Fprint(w, `{"organizations":[{"id":"org-a"}]}`)
		case "/projects":
			if r.URL.Query().Get("cursor") == "" {
				_, _ = fmt.Fprint(w, `{"projects":[{"id":"first","name":"First"},{"id":"second","name":"Second"}],"unavailable_project_ids":["down-a","down-b"],"pagination":{"cursor":"more"}}`)
			} else {
				_, _ = fmt.Fprint(w, `{"projects":[],"unavailable_project_ids":["down-c"]}`)
			}
		case "/projects/first/branches", "/projects/second/branches":
			_, _ = fmt.Fprint(w, `{"branches":[{"id":"b1","name":"One"},{"id":"b2","name":"Two"}]}`)
		default:
			if strings.HasSuffix(r.URL.Path, "/databases") {
				w.WriteHeader(http.StatusForbidden)
			} else {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
		}
	})
	var branches, failures []string
	token := ""
	for {
		page, err := client.DiscoverPage(context.Background(), "", 1, token)
		require.NoError(t, err)
		for _, branch := range page.Branches {
			branches = append(branches, branch.ProjectID+"/"+branch.ID)
		}
		for _, failure := range page.Failures {
			require.ErrorIs(t, failure.Error, ErrUnavailable)
			failures = append(failures, failure.ProjectID)
		}
		if token = page.NextPageToken; token == "" {
			break
		}
	}
	require.Equal(t, []string{"first/b1", "first/b2", "second/b1", "second/b2"}, branches)
	require.Equal(t, []string{"down-a", "down-b", "down-c"}, failures)
}
