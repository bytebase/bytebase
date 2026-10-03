package mcp

import (
	"encoding/json"
	"net/http"
	"path"
	"slices"
	"strconv"
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

// TestResolve_PolicyDenialGetsNoRoleAdvice covers the resolve door. It answers
// 403 when the stored ceiling is unserved, which is exactly the case where
// telling the agent to ask for bb.databases.list would send the person it acts
// for after a grant that cannot fix a broken setting.
func TestResolve_PolicyDenialGetsNoRoleAdvice(t *testing.T) {
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

// projectRoleAPI answers ListDatabases and SearchProjects the way the server
// answers a caller whose roles are on the readable projects only. Unless
// workspaceWide is set, the workspace parent is refused, as it is for a
// caller without workspace-wide bb.databases.list.
type projectRoleAPI struct {
	databases     []map[string]any
	readable      []string
	searchPages   [][]string
	workspaceWide bool

	mu       sync.Mutex
	requests []string
}

func (a *projectRoleAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Parent    string `json:"parent"`
		Filter    string `json:"filter"`
		PageToken string `json:"pageToken"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	method := path.Base(r.URL.Path)
	a.mu.Lock()
	a.requests = append(a.requests, strings.TrimSpace(method+" "+body.Parent))
	a.mu.Unlock()

	reply := func(status int, payload map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}
	if method == "SearchProjects" {
		page, _ := strconv.Atoi(body.PageToken)
		var projects []map[string]any
		for _, name := range a.searchPages[page] {
			projects = append(projects, map[string]any{"name": name})
		}
		next := ""
		if page+1 < len(a.searchPages) {
			next = strconv.Itoa(page + 1)
		}
		reply(http.StatusOK, map[string]any{"projects": projects, "nextPageToken": next})
		return
	}

	var scoped []map[string]any
	switch {
	case strings.HasPrefix(body.Parent, "workspaces/") && a.workspaceWide:
		scoped = a.databases
	case strings.HasPrefix(body.Parent, "workspaces/"):
		reply(http.StatusForbidden, map[string]any{"message": `user does not have permission "bb.databases.list"`})
		return
	case slices.Contains(a.readable, body.Parent):
		for _, db := range a.databases {
			if db["project"] == body.Parent {
				scoped = append(scoped, db)
			}
		}
	case slices.ContainsFunc(a.databases, func(db map[string]any) bool { return db["project"] == body.Parent }):
		reply(http.StatusForbidden, map[string]any{"message": `user does not have permission "bb.projects.get" in "` + body.Parent + `"`})
		return
	default:
		reply(http.StatusNotFound, map[string]any{"message": `project "` + body.Parent + `" not found`})
		return
	}
	reply(http.StatusOK, map[string]any{"databases": applyMockFilter(scoped, body.Filter)})
}

func (a *projectRoleAPI) requested() []string {
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

func TestResolve_ProjectRoleListsThroughReadableProjects(t *testing.T) {
	api := &projectRoleAPI{
		databases: hrAndFinanceDatabases(),
		readable:  []string{"projects/hr"},
		// SearchProjects filters a page after reading it, so a page can be
		// empty and still have a next one.
		searchPages: [][]string{{}, {"projects/hr"}},
	}
	s := newTestServerWithMock(t, api)

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "", "")
	require.NoError(t, err)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
	require.Equal(t, "projects/hr", resolved.project)
	require.Equal(t, []string{
		"ListDatabases workspaces/wk-test",
		"SearchProjects",
		"SearchProjects",
		"ListDatabases projects/hr",
	}, api.requested())

	_, err = s.resolveDatabase(testContext(), "payroll_db", "", "")
	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Equal(t, "DATABASE_NOT_FOUND", te.Code)
	require.NotContains(t, api.requested(), "ListDatabases projects/finance",
		"only the projects SearchProjects returned may be listed")
}

func TestResolve_ProjectRoleWithNamedProject(t *testing.T) {
	for _, tc := range []struct {
		name     string
		project  string
		database string
		wantCode string
		wantText string
	}{
		{name: "readable", project: "hr", database: "employee_db"},
		{name: "refused", project: "finance", database: "payroll_db", wantCode: "PERMISSION_DENIED", wantText: "projects/finance"},
		{name: "missing", project: "gone", database: "employee_db", wantCode: "DATABASE_NOT_FOUND"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &projectRoleAPI{databases: hrAndFinanceDatabases(), readable: []string{"projects/hr"}}
			s := newTestServerWithMock(t, api)

			resolved, err := s.resolveDatabase(testContext(), tc.database, "", tc.project)
			require.Equal(t, []string{
				"ListDatabases workspaces/wk-test",
				"ListDatabases projects/" + tc.project,
			}, api.requested(), "a named project is listed alone, without a project search")
			if tc.wantCode == "" {
				require.NoError(t, err)
				require.Equal(t, "projects/hr", resolved.project)
				return
			}
			var te *toolError
			require.ErrorAs(t, err, &te)
			require.Equal(t, tc.wantCode, te.Code)
			require.Contains(t, te.Message, tc.wantText)
		})
	}
}

func TestResolve_WorkspaceCallerListsOnce(t *testing.T) {
	api := &projectRoleAPI{databases: hrAndFinanceDatabases(), workspaceWide: true}
	s := newTestServerWithMock(t, api)

	resolved, err := s.resolveDatabase(testContext(), "payroll_db", "", "")
	require.NoError(t, err)
	require.Equal(t, "projects/finance", resolved.project)
	require.Equal(t, []string{"ListDatabases workspaces/wk-test"}, api.requested())
}

// TestTranslateMetadataError_PolicyDenialGetsNoRoleAdvice is the same contract
// on the get_schema door.
func TestTranslateMetadataError_PolicyDenialGetsNoRoleAdvice(t *testing.T) {
	body, err := json.Marshal(map[string]any{
		"message": "/bytebase.v1.DatabaseService/GetDatabaseMetadata is not available to MCP sessions",
	})
	require.NoError(t, err)

	got := translateMetadataError(&apiResponse{Status: http.StatusForbidden, Body: body})
	var te *toolError
	require.ErrorAs(t, got, &te)
	require.Contains(t, te.Message, "MCP sessions")
	require.Empty(t, te.Suggestion)

	plain, err := json.Marshal(map[string]any{"message": "permission denied"})
	require.NoError(t, err)
	got = translateMetadataError(&apiResponse{Status: http.StatusForbidden, Body: plain})
	require.ErrorAs(t, got, &te)
	require.Contains(t, te.Suggestion, "bb.databases.getSchema",
		"a plain ACL denial keeps the permission advice")
}
