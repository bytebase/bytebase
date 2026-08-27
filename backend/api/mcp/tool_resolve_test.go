package mcp

import (
	"encoding/json"
	"net/http"
	"strings"
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
// 403 only when the stored ceiling is unreadable or unserved, which is exactly
// the case where telling the agent to ask for bb.databases.list would send the
// person it acts for after a grant that cannot fix a broken setting.
func TestResolve_PolicyDenialGetsNoRoleAdvice(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "DatabaseService/ListDatabases") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "/bytebase.v1.DatabaseService/ListDatabases is refused: this workspace's " +
					"stored MCP capability ceiling is not one this build understands",
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
	require.Empty(t, te.Suggestion, "no grant fixes a stored ceiling this build cannot read")
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
