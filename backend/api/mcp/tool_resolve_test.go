package mcp

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

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
		buildDatabaseFilter("app", projectInstance, ""),
	)
	require.Equal(t,
		`name.contains("app") && instance == "instances/instance-a" && project == "projects/project-a"`,
		buildDatabaseFilter("app", "instance-a", "project-a"),
	)
}

// TestResolve_PolicyDenialPassesThrough: an unserved ceiling refuses the lookup,
// and the refusal names its own way out.
func TestResolve_PolicyDenialPassesThrough(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "DatabaseService/ListDatabases") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "/bytebase.v1.DatabaseService/ListDatabases is refused: this workspace's " +
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

// listingAPI answers ListDatabases the way the server answers a caller who
// can read the readable projects. It fails any workspace-parent listing, which
// project roles cannot make.
type listingAPI struct {
	databases []map[string]any
	readable  []string

	mu      sync.Mutex
	parents []string
}

func (a *listingAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Parent string `json:"parent"`
		Filter string `json:"filter"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	a.mu.Lock()
	a.parents = append(a.parents, body.Parent)
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
	case body.Parent == "projects/-":
		reply(http.StatusOK, map[string]any{"databases": applyMockFilter(inProjects(a.readable...), body.Filter)})
	case slices.Contains(a.readable, body.Parent):
		reply(http.StatusOK, map[string]any{"databases": applyMockFilter(inProjects(body.Parent), body.Filter)})
	case len(inProjects(body.Parent)) > 0:
		reply(http.StatusForbidden, map[string]any{"message": `user does not have permission "bb.projects.get" in "` + body.Parent + `"`})
	case strings.HasPrefix(body.Parent, "projects/"):
		reply(http.StatusNotFound, map[string]any{"message": `project "` + strings.TrimPrefix(body.Parent, "projects/") + `" not found`})
	default:
		reply(http.StatusInternalServerError, map[string]any{"message": "unexpected parent " + body.Parent})
	}
}

func (a *listingAPI) requested() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.parents)
}

func hrAndFinanceDatabases() []map[string]any {
	return []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr", "POSTGRES", "ds-1"),
		makeDatabase("instances/prod-pg/databases/payroll_db", "instances/prod-pg", "projects/finance", "POSTGRES", "ds-2"),
	}
}

func TestResolve_ListsThroughTheWildcard(t *testing.T) {
	for _, tc := range []struct {
		name     string
		readable []string
	}{
		{name: "project role", readable: []string{"projects/hr"}},
		{name: "workspace admin", readable: []string{"projects/hr", "projects/finance"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &listingAPI{databases: hrAndFinanceDatabases(), readable: tc.readable}
			s := newTestServerWithMock(t, api)

			resolved, err := s.resolveDatabase(testContext(), "employee_db", "", "")
			require.NoError(t, err)
			require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
			require.Equal(t, "projects/hr", resolved.project)
			require.Equal(t, []string{"projects/-"}, api.requested())
		})
	}
}

func TestResolve_UnreadableProjectIsNotFound(t *testing.T) {
	for _, tc := range []struct {
		name     string
		readable []string
	}{
		{name: "project role", readable: []string{"projects/hr"}},
		{name: "no project role"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestServerWithMock(t, &listingAPI{databases: hrAndFinanceDatabases(), readable: tc.readable})

			_, err := s.resolveDatabase(testContext(), "payroll_db", "", "")
			var te *toolError
			require.ErrorAs(t, err, &te)
			require.Equal(t, "DATABASE_NOT_FOUND", te.Code)
			require.Contains(t, te.Suggestion, "projects you cannot read")
		})
	}
}

func TestResolve_WildcardKeepsTheInstanceFilter(t *testing.T) {
	api := &listingAPI{
		databases: []map[string]any{
			makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr", "POSTGRES", "ds-1"),
			makeDatabase("instances/staging-pg/databases/employee_db", "instances/staging-pg", "projects/ops", "POSTGRES", "ds-2"),
			makeDatabase("projects/ops/instances/ops-pg/databases/employee_db", "projects/ops/instances/ops-pg", "projects/ops", "POSTGRES", "ds-3"),
			makeDatabase("instances/prod-pg/databases/payroll_db", "instances/prod-pg", "projects/finance", "POSTGRES", "ds-4"),
		},
		readable: []string{"projects/hr", "projects/ops"},
	}
	s := newTestServerWithMock(t, api)

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "", "")
	require.NoError(t, err)
	require.True(t, resolved.ambiguous)
	require.ElementsMatch(t, []string{"hr", "ops", "ops"}, []string{
		resolved.candidates[0].Project, resolved.candidates[1].Project, resolved.candidates[2].Project,
	})

	for instance, want := range map[string]string{
		"staging-pg":                    "instances/staging-pg/databases/employee_db",
		"projects/ops/instances/ops-pg": "projects/ops/instances/ops-pg/databases/employee_db",
	} {
		resolved, err := s.resolveDatabase(testContext(), "employee_db", instance, "")
		require.NoError(t, err)
		require.False(t, resolved.ambiguous, instance)
		require.Equal(t, want, resolved.resourceName)
	}
}

func TestResolve_NamedProjectIsListedDirectly(t *testing.T) {
	for _, tc := range []struct {
		name     string
		project  string
		database string
		want     string
	}{
		{name: "readable", project: "hr", database: "employee_db"},
		{name: "refused", project: "finance", database: "payroll_db", want: `user does not have permission "bb.projects.get" in "projects/finance"`},
		{name: "missing", project: "gone", database: "employee_db", want: `project "gone" not found`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &listingAPI{databases: hrAndFinanceDatabases(), readable: []string{"projects/hr"}}
			s := newTestServerWithMock(t, api)

			resolved, err := s.resolveDatabase(testContext(), tc.database, "", tc.project)
			require.Equal(t, []string{"projects/" + tc.project}, api.requested())
			if tc.want == "" {
				require.NoError(t, err)
				require.Equal(t, "projects/hr", resolved.project)
				return
			}
			require.ErrorContains(t, err, tc.want, "the server's answer for the named project comes through")
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
