package mcp

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestResolve_ProjectPopulated(t *testing.T) {
	databases := []databaseEntry{
		{
			Name:    "instances/prod-pg/databases/employee_db",
			Project: "projects/hr-system",
			InstanceResource: instanceResource{
				Name:        "instances/prod-pg",
				Engine:      "POSTGRES",
				DataSources: []dataSource{{ID: "ds-admin-1", Type: "ADMIN"}},
			},
		},
	}

	resolved, err := matchDatabases(databases, "employee_db", "", "")
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "projects/hr-system", resolved.project)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
	require.Equal(t, "POSTGRES", resolved.engine)
}

func TestResolve_ProjectInAmbiguous(t *testing.T) {
	databases := []databaseEntry{
		{
			Name:    "instances/prod-pg/databases/app",
			Project: "projects/payments",
			InstanceResource: instanceResource{
				Name:        "instances/prod-pg",
				Engine:      "POSTGRES",
				DataSources: []dataSource{{ID: "ds-1", Type: "ADMIN"}},
			},
		},
		{
			Name:    "instances/staging-pg/databases/app",
			Project: "projects/staging",
			InstanceResource: instanceResource{
				Name:        "instances/staging-pg",
				Engine:      "POSTGRES",
				DataSources: []dataSource{{ID: "ds-2", Type: "ADMIN"}},
			},
		},
	}

	resolved, err := matchDatabases(databases, "app", "", "")
	require.NoError(t, err)
	require.True(t, resolved.ambiguous)
	require.Len(t, resolved.candidates, 2)
	require.Equal(t, "projects/payments", resolved.projects["instances/prod-pg/databases/app"])
	require.Equal(t, "projects/staging", resolved.projects["instances/staging-pg/databases/app"])
}

func TestResolve_ProjectInstanceDatabaseAndFilter(t *testing.T) {
	const (
		projectInstance = "projects/project-a/instances/instance-a"
		databaseName    = "projects/project-a/instances/instance-a/databases/app"
	)

	databases := []databaseEntry{{
		Name:    databaseName,
		Project: "projects/project-a",
		InstanceResource: instanceResource{
			Name:        projectInstance,
			Engine:      "POSTGRES",
			DataSources: []dataSource{{ID: "ds-admin-1", Type: "ADMIN"}},
		},
	}}

	resolved, err := matchDatabases(databases, "app", "", "")
	require.NoError(t, err)
	require.Equal(t, databaseName, resolved.resourceName)

	require.Equal(t,
		`name.contains("app") && instance == "projects/project-a/instances/instance-a"`,
		buildDatabaseFilter("app", projectInstance),
	)
}

// TestResolve_PolicyDenialPassesThrough: an unserved ceiling refuses the lookup,
// and the refusal names its own way out.
func TestResolve_PolicyDenialPassesThrough(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "DatabaseService/SearchDatabases") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "/bytebase.v1.DatabaseService/SearchDatabases is refused: this workspace's " +
					"stored MCP capability ceiling is not one this build serves",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	s := newTestServerWithMock(t, handler)

	_, err := s.resolveDatabase(testContext(), "employee_db", "", "")
	require.Error(t, err)
	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Contains(t, te.Message, "MCP capability ceiling")
	require.Empty(t, te.Suggestion, "no grant fixes a stored ceiling this build does not serve")
}

// listingAPI answers SearchDatabases and ListDatabases the way the server
// answers a caller who holds bb.databases.get and bb.databases.list in the
// readable projects, as every built-in project role does. It fails any
// workspace-parent listing, which project roles cannot make.
type listingAPI struct {
	databases []map[string]any
	readable  []string

	mu       sync.Mutex
	requests []string
}

func (a *listingAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Parent string `json:"parent"`
		Filter string `json:"filter"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	search := strings.HasSuffix(r.URL.Path, "DatabaseService/SearchDatabases")
	a.mu.Lock()
	if search {
		a.requests = append(a.requests, "SearchDatabases")
	} else {
		a.requests = append(a.requests, body.Parent)
	}
	a.mu.Unlock()

	reply := func(status int, payload map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}
	inProjects := func(projects ...string) []map[string]any {
		var scoped []map[string]any
		for _, db := range a.databases {
			if slices.Contains(projects, db["project"].(string)) {
				scoped = append(scoped, db)
			}
		}
		return scoped
	}
	switch {
	case search:
		reply(http.StatusOK, map[string]any{"databases": applyMockFilter(inProjects(a.readable...), body.Filter)})
	case slices.Contains(a.readable, body.Parent):
		reply(http.StatusOK, map[string]any{"databases": applyMockFilter(inProjects(body.Parent), body.Filter)})
	case len(inProjects(body.Parent)) > 0:
		reply(http.StatusForbidden, map[string]any{"message": `user does not have permission "bb.databases.list" in "` + body.Parent + `"`})
	case strings.HasPrefix(body.Parent, "projects/"):
		reply(http.StatusNotFound, map[string]any{"message": `project "` + strings.TrimPrefix(body.Parent, "projects/") + `" not found`})
	default:
		reply(http.StatusInternalServerError, map[string]any{"message": "unexpected parent " + body.Parent})
	}
}

func (a *listingAPI) requested() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.requests)
}

func hrAndFinanceDatabases() []map[string]any {
	return []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr", "POSTGRES", "ds-1"),
		makeDatabase("instances/prod-pg/databases/payroll_db", "instances/prod-pg", "projects/finance", "POSTGRES", "ds-2"),
	}
}

func TestResolve_SearchesWhenNoProjectIsNamed(t *testing.T) {
	api := &listingAPI{databases: hrAndFinanceDatabases(), readable: []string{"projects/hr"}}
	s := newTestServerWithMock(t, api)

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "", "")
	require.NoError(t, err)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
	require.Equal(t, "projects/hr", resolved.project)
	require.Equal(t, []string{"SearchDatabases"}, api.requested())
}

func TestResolve_ReadsEveryPage(t *testing.T) {
	var (
		mu     sync.Mutex
		tokens []string
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PageToken string `json:"pageToken"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		tokens = append(tokens, body.PageToken)
		requests := len(tokens)
		mu.Unlock()
		// A loop that resends the first page fails here instead of hanging.
		if requests > 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		databases := hrAndFinanceDatabases()
		page := map[string]any{"databases": databases[:1]}
		if body.PageToken == "" {
			page = map[string]any{"databases": databases[1:], "nextPageToken": "page-2"}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	})
	s := newTestServerWithMock(t, handler)

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "", "")
	require.NoError(t, err)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"", "page-2"}, tokens)
}

func TestResolve_UnreadableProjectIsNotFound(t *testing.T) {
	for _, tc := range []struct {
		name     string
		readable []string
		instance string
	}{
		{name: "project role", readable: []string{"projects/hr"}},
		{name: "no project role"},
		{name: "instance filter", readable: []string{"projects/hr"}, instance: "prod-pg"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestServerWithMock(t, &listingAPI{databases: hrAndFinanceDatabases(), readable: tc.readable})

			_, err := s.resolveDatabase(testContext(), "payroll_db", tc.instance, "")
			var te *toolError
			require.ErrorAs(t, err, &te)
			require.Equal(t, "DATABASE_NOT_FOUND", te.Code)
			require.Contains(t, te.Suggestion, "projects where you lack bb.databases.get")
		})
	}
}

func TestResolve_NamedProjectIsListedDirectly(t *testing.T) {
	for _, tc := range []struct {
		name     string
		project  string
		database string
		// wantDenied is the whole tool error for a refusal; wantText is the
		// plain-text error for any other failure.
		wantDenied *toolError
		wantText   string
	}{
		{name: "readable", project: "hr", database: "employee_db"},
		{name: "refused", project: "finance", database: "payroll_db", wantDenied: &toolError{
			Code:    "PERMISSION_DENIED",
			Message: `user does not have permission "bb.databases.list" in "projects/finance"`,
		}},
		{name: "missing", project: "gone", database: "employee_db", wantText: `list databases failed: HTTP 404: project "gone" not found`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &listingAPI{databases: hrAndFinanceDatabases(), readable: []string{"projects/hr"}}
			s := newTestServerWithMock(t, api)

			resolved, err := s.resolveDatabase(testContext(), tc.database, "", tc.project)
			require.Equal(t, []string{"projects/" + tc.project}, api.requested())
			var te *toolError
			switch {
			case tc.wantDenied != nil:
				require.ErrorAs(t, err, &te)
				require.Equal(t, *tc.wantDenied, *te, "the server's refusal names the project, and nothing is added")
			case tc.wantText != "":
				require.False(t, errors.As(err, &te), "a missing project is not reported as a tool error code")
				require.EqualError(t, err, tc.wantText)
			default:
				require.NoError(t, err)
				require.Equal(t, "projects/hr", resolved.project)
			}
		})
	}
}

// TestTranslateMetadataError_PassesTheRefusalThrough is the same contract on
// the get_schema door.
func TestTranslateMetadataError_PassesTheRefusalThrough(t *testing.T) {
	body, err := json.Marshal(map[string]any{
		"message": "/bytebase.v1.DatabaseService/GetDatabaseMetadata is not available to MCP sessions",
	})
	require.NoError(t, err)

	got := translateMetadataError(&apiResponse{Status: http.StatusForbidden, Body: body})
	var te *toolError
	require.ErrorAs(t, got, &te)
	require.Contains(t, te.Message, "MCP sessions")
	require.Empty(t, te.Suggestion)

	plain, err := json.Marshal(map[string]any{"message": `user does not have permission "bb.databases.getSchema"`})
	require.NoError(t, err)
	got = translateMetadataError(&apiResponse{Status: http.StatusForbidden, Body: plain})
	require.ErrorAs(t, got, &te)
	require.Equal(t, toolError{Code: "PERMISSION_DENIED", Message: `user does not have permission "bb.databases.getSchema"`}, *te,
		"an ACL denial names its permission, and nothing is added")
}
