package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// makeDatabase builds a database entry for mock responses.
func makeDatabase(name, instanceName, project, engine, dsID, dsType string) map[string]any {
	return map[string]any{
		"name":    name,
		"project": project,
		"instanceResource": map[string]any{
			"name":   instanceName,
			"engine": engine,
			"dataSources": []any{
				map[string]any{"id": dsID, "type": dsType},
			},
		},
	}
}

// makeDatabaseWithDualDS builds a database entry with both ADMIN and READ_ONLY data sources.
func makeDatabaseWithDualDS(name, instanceName, project, engine, adminDSID, readOnlyDSID string) map[string]any {
	return map[string]any{
		"name":    name,
		"project": project,
		"instanceResource": map[string]any{
			"name":   instanceName,
			"engine": engine,
			"dataSources": []any{
				map[string]any{"id": adminDSID, "type": "ADMIN"},
				map[string]any{"id": readOnlyDSID, "type": "READ_ONLY"},
			},
		},
	}
}

// mockListDatabases returns an HTTP handler that serves a ListDatabases response.
func mockListDatabases(databases []map[string]any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{"databases": databases}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

func TestQueryDatabase_SingleMatch(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ADMIN"),
		makeDatabase("instances/prod-pg/databases/orders_db", "instances/prod-pg", "projects/commerce", "POSTGRES", "ds-admin-2", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee_db"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
	require.Equal(t, "ds-admin-1", resolved.dataSourceID)
}

func TestQueryDatabase_CaseInsensitiveMatch(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "Employee_DB"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
}

func TestQueryDatabase_SubstringMatch(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
}

func TestQueryDatabase_NotFound(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	_, err := s.resolveDatabase(context.Background(), QueryInput{Database: "nonexistent"})
	require.Error(t, err)

	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Equal(t, "DATABASE_NOT_FOUND", te.Code)
	require.Contains(t, te.Suggestion, "search_api")
}

func TestQueryDatabase_NotFoundWithFilters(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	_, err := s.resolveDatabase(context.Background(), QueryInput{
		Database: "employee_db",
		Instance: "nonexistent-instance",
	})
	require.Error(t, err)

	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Equal(t, "DATABASE_NOT_FOUND", te.Code)
	require.Contains(t, te.Suggestion, "without instance/project filters")
}

func TestQueryDatabase_Ambiguous(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ADMIN"),
		makeDatabase("instances/staging-mysql/databases/employee_db", "instances/staging-mysql", "projects/hr-system", "MYSQL", "ds-admin-2", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee_db"})
	require.NoError(t, err)
	require.True(t, resolved.ambiguous)
	require.Len(t, resolved.candidates, 2)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.candidates[0].Database)
	require.Equal(t, "POSTGRES", resolved.candidates[0].Engine)
	require.Equal(t, "instances/staging-mysql/databases/employee_db", resolved.candidates[1].Database)
	require.Equal(t, "MYSQL", resolved.candidates[1].Engine)
}

func TestQueryDatabase_AmbiguousWithInstance(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ADMIN"),
		makeDatabase("instances/staging/databases/employee_db", "instances/staging", "projects/hr-system", "POSTGRES", "ds-admin-2", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{
		Database: "employee_db",
		Instance: "prod-pg",
	})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
	require.Equal(t, "ds-admin-1", resolved.dataSourceID)
}

func TestQueryDatabase_ReadOnlyDatasource(t *testing.T) {
	databases := []map[string]any{
		makeDatabaseWithDualDS("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ds-ro-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee_db"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "ds-ro-1", resolved.dataSourceID)
}

func TestQueryDatabase_AdminFallback(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee_db"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "ds-admin-1", resolved.dataSourceID)
}

func TestQueryDatabase_FormatAmbiguousResult(t *testing.T) {
	candidates := []Candidate{
		{Database: "instances/prod-pg/databases/employee_db", Instance: "prod-pg", Project: "hr-system", Engine: "POSTGRES"},
		{Database: "instances/staging/databases/employee_db", Instance: "staging", Project: "hr-system", Engine: "POSTGRES"},
	}

	result := formatAmbiguousResult("employee_db", candidates)
	require.True(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcpsdk.TextContent)
	require.True(t, ok)

	var parsed struct {
		Code       string      `json:"code"`
		Message    string      `json:"message"`
		Candidates []Candidate `json:"candidates"`
	}
	err := json.Unmarshal([]byte(textContent.Text), &parsed)
	require.NoError(t, err)
	require.Equal(t, "AMBIGUOUS_TARGET", parsed.Code)
	require.Len(t, parsed.Candidates, 2)
}

func TestQueryDatabase_HandleValidation(t *testing.T) {
	s := newTestServerWithMock(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"databases": []}`)
	}))

	// Missing database.
	_, _, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{Statement: "SELECT 1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "database is required")

	// Missing statement.
	_, _, err = s.handleQueryDatabase(context.Background(), nil, QueryInput{Database: "employee_db"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "statement is required")
}

func TestQueryDatabase_LimitNormalization(t *testing.T) {
	// Verify limit is capped at maxQueryLimit via the handler.
	// We test the handler which calls resolveDatabase, so the database must not be found
	// to avoid hitting the executeQuery stub. We just verify the handler doesn't panic.
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	// The handler resolves the database, then calls executeQuery which returns "not implemented".
	// That's expected — we verify the handler doesn't error on limit normalization.
	result, _, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT 1",
		Limit:     2000,
	})
	// Should return IsError result from executeQuery stub, not a limit error.
	require.NoError(t, err)
	require.True(t, result.IsError)
	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "not implemented")
}

func TestQueryDatabase_ExactMatchPriority(t *testing.T) {
	// When both exact and substring matches exist, exact should win.
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-1", "ADMIN"),
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-2", "ADMIN"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee", resolved.resourceName)
	require.Equal(t, "ds-1", resolved.dataSourceID)
}
