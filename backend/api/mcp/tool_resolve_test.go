package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
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
	// An unserved ceiling refuses every call, so the project-by-project
	// listing meets the same refusal.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": r.URL.Path + " is refused: this workspace's " +
				"stored MCP capability ceiling is not one this build serves",
		})
	})
	s := newTestServerWithMock(t, handler)

	_, err := s.resolveDatabase(testContext(), "employee_db", "", "")
	require.Error(t, err)
	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Contains(t, te.Message, "MCP capability ceiling")
	require.Empty(t, te.Suggestion, "no grant fixes a stored ceiling this build does not serve")
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

// projectScopedMock refuses the workspace-wide listing, as the API does for a
// user whose bb.databases.list comes from project roles, and serves each
// project's own listing.
func projectScopedMock(databases []map[string]any, projects []string) (http.Handler, func() []string) {
	var mu sync.Mutex
	var parents []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "ProjectService/SearchProjects"):
			page := []map[string]any{}
			for _, project := range projects {
				page = append(page, map[string]any{"name": project})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"projects": page})
		case strings.Contains(r.URL.Path, "DatabaseService/ListDatabases"):
			body, _ := io.ReadAll(r.Body)
			var request struct {
				Parent string `json:"parent"`
			}
			_ = json.Unmarshal(body, &request)
			mu.Lock()
			parents = append(parents, request.Parent)
			mu.Unlock()
			if strings.HasPrefix(request.Parent, "workspaces/") {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{"code": "permission_denied", "message": `user does not have permission "bb.databases.list"`})
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			mockListDatabases(databases).ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	return handler, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), parents...)
	}
}

func TestResolve_ListsProjectByProjectWithoutWorkspaceAccess(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("projects/hr/instances/prod-pg/databases/hr_test", "projects/hr/instances/prod-pg", "projects/hr", "POSTGRES", "ds-admin-1"),
		makeDatabase("projects/fin/instances/prod-pg/databases/ledger", "projects/fin/instances/prod-pg", "projects/fin", "POSTGRES", "ds-admin-1"),
	}
	handler, parents := projectScopedMock(databases, []string{"projects/hr", "projects/fin"})
	s := newTestServerWithMock(t, handler)

	resolved, err := s.resolveDatabase(testContext(), "hr_test", "", "")
	require.NoError(t, err)
	require.Equal(t, "projects/hr/instances/prod-pg/databases/hr_test", resolved.resourceName)
	require.Equal(t, []string{"workspaces/wk-test", "projects/hr", "projects/fin"}, parents())
}

func TestResolve_KeepsTheWorkspaceRefusalWhenNoProjectIsVisible(t *testing.T) {
	handler, _ := projectScopedMock(nil, nil)
	s := newTestServerWithMock(t, handler)

	_, err := s.resolveDatabase(testContext(), "hr_test", "", "")
	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Equal(t, toolError{Code: "PERMISSION_DENIED", Message: `user does not have permission "bb.databases.list"`}, *te)
}
