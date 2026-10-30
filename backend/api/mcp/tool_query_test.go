package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// makeDatabase builds a database entry with an ADMIN datasource for mock responses.
func makeDatabase(name, instanceName, project, engine, dsID string) map[string]any {
	return map[string]any{
		"name":    name,
		"project": project,
		"instanceResource": map[string]any{
			"name":   instanceName,
			"engine": engine,
			"dataSources": []any{
				map[string]any{"id": dsID, "type": "ADMIN"},
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

// mockListDatabases returns an HTTP handler that serves database listings for
// a caller who holds bb.databases.get and bb.databases.list in every project:
// SearchDatabases lists all databases, and ListDatabases with a project parent
// lists that project's.
func mockListDatabases(databases []map[string]any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var reqBody struct {
			Parent string `json:"parent"`
			Filter string `json:"filter"`
		}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		var scoped []map[string]any
		if strings.HasSuffix(r.URL.Path, "DatabaseService/SearchDatabases") {
			scoped = databases
		} else {
			for _, db := range databases {
				proj, ok := db["project"].(string)
				if ok && proj == reqBody.Parent {
					scoped = append(scoped, db)
				}
			}
		}

		result := scoped
		if reqBody.Filter != "" {
			result = applyMockFilter(scoped, reqBody.Filter)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"databases": result})
	})
}

// applyMockFilter applies a simplified filter to mock databases.
// Supports: name.contains("x"), instance == "instances/x"
func applyMockFilter(databases []map[string]any, filter string) []map[string]any {
	var result []map[string]any
	for _, db := range databases {
		name, ok := db["name"].(string)
		if !ok {
			continue
		}
		ir, ok := db["instanceResource"].(map[string]any)
		if !ok {
			continue
		}
		instanceName, ok := ir["name"].(string)
		if !ok {
			continue
		}

		match := true
		// Check name.contains("x") — substring match
		if idx := strings.Index(filter, `name.contains(`); idx >= 0 {
			start := idx + len(`name.contains("`)
			end := strings.Index(filter[start:], `"`)
			if end > 0 {
				substr := filter[start : start+end]
				if !strings.Contains(strings.ToLower(name), strings.ToLower(substr)) {
					match = false
				}
			}
		}
		// Check instance == "instances/x"
		if idx := strings.Index(filter, `instance == "`); idx >= 0 {
			start := idx + len(`instance == "`)
			end := strings.Index(filter[start:], `"`)
			if end > 0 {
				expected := filter[start : start+end]
				if instanceName != expected {
					match = false
				}
			}
		}
		if match {
			result = append(result, db)
		}
	}
	return result
}

// makeQueryResponse builds a mock SQL Query API response.
func makeQueryResponse(columns, columnTypes []string, rows [][]any, latency string) map[string]any {
	apiRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		values := make([]map[string]any, 0, len(row))
		for _, v := range row {
			values = append(values, map[string]any{"stringValue": v})
		}
		apiRows = append(apiRows, map[string]any{"values": values})
	}
	return map[string]any{
		"results": []map[string]any{
			{
				"columnNames":     columns,
				"columnTypeNames": columnTypes,
				"rows":            apiRows,
				"rowsCount":       fmt.Sprintf("%d", len(rows)),
				"latency":         latency,
				"error":           "",
				"statement":       "SELECT ...",
			},
		},
	}
}

// mockQueryServer returns an HTTP handler that routes to ListDatabases or Query
// based on the request URL path.
func mockQueryServer(databases []map[string]any, queryResp map[string]any) http.Handler {
	listHandler := mockListDatabases(databases)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "SQLService/Query") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(queryResp)
			return
		}
		listHandler.ServeHTTP(w, r)
	})
}

func TestBuildDatabaseFilter(t *testing.T) {
	tests := []struct {
		name     string
		database string
		instance string
		expect   string
	}{
		{
			name:     "database only",
			database: "employee_db",
			expect:   `name.contains("employee_db")`,
		},
		{
			name:     "with instance",
			database: "employee_db",
			instance: "prod-pg",
			expect:   `name.contains("employee_db") && instance == "instances/prod-pg"`,
		},
		{
			name:     "with canonical project instance",
			database: "employee_db",
			instance: "projects/hr-system/instances/prod-pg",
			expect:   `name.contains("employee_db") && instance == "projects/hr-system/instances/prod-pg"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildDatabaseFilter(tc.database, tc.instance)
			require.Equal(t, tc.expect, got)
		})
	}
}

func TestQueryDatabase_SingleMatch(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
		makeDatabase("instances/prod-pg/databases/orders_db", "instances/prod-pg", "projects/commerce", "POSTGRES", "ds-admin-2"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "", "")
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
	require.Equal(t, "ds-admin-1", resolved.dataSourceID)
}

func TestQueryDatabase_CaseInsensitiveMatch(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "Employee_DB", "", "")
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
}

func TestQueryDatabase_SubstringMatch(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "employee", "", "")
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
}

func TestQueryDatabase_NotFound(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	_, err := s.resolveDatabase(testContext(), "nonexistent", "", "")
	require.Error(t, err)

	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Equal(t, "DATABASE_NOT_FOUND", te.Code)
	require.Contains(t, te.Suggestion, "search_api")
}

func TestQueryDatabase_NotFoundWithFilters(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	_, err := s.resolveDatabase(testContext(), "employee_db", "nonexistent-instance", "")
	require.Error(t, err)

	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Equal(t, "DATABASE_NOT_FOUND", te.Code)
	require.Contains(t, te.Suggestion, "without instance/project filters")
}

func TestQueryDatabase_Ambiguous(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
		makeDatabase("instances/staging-mysql/databases/employee_db", "instances/staging-mysql", "projects/hr-system", "MYSQL", "ds-admin-2"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "", "")
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
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
		makeDatabase("instances/staging/databases/employee_db", "instances/staging", "projects/hr-system", "POSTGRES", "ds-admin-2"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "prod-pg", "")
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
	require.Equal(t, "ds-admin-1", resolved.dataSourceID)
}

func TestQueryDatabase_ProjectInstanceRequiresCanonicalInstanceFilter(t *testing.T) {
	const (
		instanceName = "projects/hr-system/instances/prod-pg"
		databaseName = "projects/hr-system/instances/prod-pg/databases/employee_db"
	)
	databases := []map[string]any{
		makeDatabase(databaseName, instanceName, "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "employee_db", instanceName, "")
	require.NoError(t, err)
	require.Equal(t, databaseName, resolved.resourceName)

	_, err = s.resolveDatabase(testContext(), "employee_db", "prod-pg", "")
	require.Error(t, err)
	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Equal(t, "DATABASE_NOT_FOUND", te.Code)
}

func TestQueryDatabase_WorkspaceInstanceWithProjectFilter(t *testing.T) {
	databaseName := "instances/prod-pg/databases/employee_db"
	databases := []map[string]any{
		makeDatabase(databaseName, "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "prod-pg", "hr-system")
	require.NoError(t, err)
	require.Equal(t, databaseName, resolved.resourceName)
}

func TestQueryDatabase_ReadOnlyDatasource(t *testing.T) {
	databases := []map[string]any{
		makeDatabaseWithDualDS("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1", "ds-ro-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "", "")
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "ds-ro-1", resolved.dataSourceID)
}

func TestQueryDatabase_AdminFallback(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "employee_db", "", "")
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
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	qr := makeQueryResponse([]string{"id"}, []string{"int4"}, [][]any{{"1"}}, "0.001s")
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	result, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT 1",
		Limit:     2000,
	})
	// Should succeed — limit capped to 1000, but only 1 row returned.
	require.NoError(t, err)
	require.False(t, result.IsError)
	output, ok := structured.(*QueryOutput)
	require.True(t, ok)
	require.Equal(t, 1, output.RowCount)
	require.False(t, output.Truncated)
}

func TestQueryDatabase_ExactMatchPriority(t *testing.T) {
	// When both exact and substring matches exist, exact should win.
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-1"),
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-2"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(testContext(), "employee", "", "")
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee", resolved.resourceName)
	require.Equal(t, "ds-1", resolved.dataSourceID)
}

// --- Task 5 tests: query execution ---

func TestQueryDatabase_FullFlow(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	qr := makeQueryResponse(
		[]string{"id", "name"},
		[]string{"int4", "varchar"},
		[][]any{{"1", "John"}, {"2", "Jane"}},
		"0.012s",
	)
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	result, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id, name FROM users",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*QueryOutput)
	require.True(t, ok)
	require.Equal(t, []string{"id", "name"}, output.Columns)
	require.Equal(t, []string{"int4", "varchar"}, output.ColumnTypes)
	require.Equal(t, 2, output.RowCount)
	require.False(t, output.Truncated)
	require.Equal(t, int64(12), output.LatencyMs)
	require.Len(t, output.Rows, 2)
	require.Equal(t, "1", output.Rows[0][0])
	require.Equal(t, "John", output.Rows[0][1])

	// Verify text header.
	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Result: 2 rows")
	require.NotContains(t, text, "Truncated")
}

func TestQueryDatabase_EmptyResult(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	qr := makeQueryResponse(
		[]string{"id", "name"},
		[]string{"int4", "varchar"},
		[][]any{},
		"0.005s",
	)
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	result, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT * FROM users WHERE id = -1",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*QueryOutput)
	require.True(t, ok)
	require.Empty(t, output.Rows)
	require.Equal(t, 0, output.RowCount)
	require.False(t, output.Truncated)
}

func TestQueryDatabase_Truncation(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	// Generate 150 rows.
	rows := make([][]any, 150)
	for i := range rows {
		rows[i] = []any{fmt.Sprintf("%d", i+1)}
	}
	qr := makeQueryResponse([]string{"id"}, []string{"int4"}, rows, "0.050s")
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	result, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id FROM users",
		// Default limit is 100.
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*QueryOutput)
	require.True(t, ok)
	require.Equal(t, 100, output.RowCount)
	require.True(t, output.Truncated)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Truncated")
}

func TestQueryDatabase_CustomLimit(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	rows := make([][]any, 100)
	for i := range rows {
		rows[i] = []any{fmt.Sprintf("%d", i+1)}
	}
	qr := makeQueryResponse([]string{"id"}, []string{"int4"}, rows, "0.020s")
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	result, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id FROM users",
		Limit:     50,
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*QueryOutput)
	require.True(t, ok)
	require.Equal(t, 50, output.RowCount)
	require.True(t, output.Truncated)
}

func TestQueryDatabase_LimitCappedAt1000(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	rows := make([][]any, 5)
	for i := range rows {
		rows[i] = []any{fmt.Sprintf("%d", i+1)}
	}
	qr := makeQueryResponse([]string{"id"}, []string{"int4"}, rows, "0.003s")
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	result, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id FROM users",
		Limit:     2000,
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*QueryOutput)
	require.True(t, ok)
	require.Equal(t, 5, output.RowCount)
	require.False(t, output.Truncated)
}

func TestQueryDatabase_QueryError(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "SQLService/Query") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "syntax error at or near \"SELEC\"",
				"code":    "INVALID_ARGUMENT",
			})
			return
		}
		mockListDatabases(databases).ServeHTTP(w, r)
	})
	s := newTestServerWithMock(t, handler)

	result, _, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELEC * FROM users",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "syntax error")
}

// TestQueryDatabase_PolicyDenialGetsNoAccessAdvice: the masked-write guard
// refuses through SQLService/Query, and no access grant lifts a workspace
// setting.
func TestQueryDatabase_PolicyDenialGetsNoAccessAdvice(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "SQLService/Query") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "/bytebase.v1.SQLService/Query is not available to MCP sessions " +
					"for this request because its SQL contains \"******\"",
				"code": "PERMISSION_DENIED",
			})
			return
		}
		mockListDatabases(databases).ServeHTTP(w, r)
	})
	s := newTestServerWithMock(t, handler)

	result, _, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "UPDATE users SET name = '******'",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "******")
	require.NotContains(t, text, "suggestion", "the refusal names its own way out")
}

func TestQueryDatabase_PermissionDenied(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "SQLService/Query") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "permission denied",
				"code":    "PERMISSION_DENIED",
			})
			return
		}
		mockListDatabases(databases).ServeHTTP(w, r)
	})
	s := newTestServerWithMock(t, handler)

	result, _, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT * FROM users",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "permission denied")
	require.NotContains(t, text, "suggestion", "the ACL refuses a missing bb.databases.get before any access grant applies")
}

// The SQL access check answers inside a 200 result, the shape the server
// returns when the caller can see the database but not read its tables.
func TestQueryDatabase_TableAccessDenialAdvisesAnAccessGrant(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("projects/hr/instances/prod-pg/databases/employee_db", "projects/hr/instances/prod-pg", "projects/hr", "POSTGRES", "ds-admin-1"),
	}
	const wantGrant = "in a Read-write session, request an access grant for failedStatement with request_access_grant and, " +
		"once approved, run it by itself (the project must allow just-in-time access); otherwise the user requests access in the SQL Editor"
	for name, tt := range map[string]struct {
		statement string
		result    map[string]any
		want      string
	}{
		"the access check refused the tables": {
			result: map[string]any{
				"error":     "permission_denied: permission denied to access resources: [projects/hr/instances/prod-pg/databases/employee_db/schemas/public/tables/employee]",
				"statement": "SELECT first_name FROM employee",
				"permissionDenied": map[string]any{
					"requiredPermissions": []string{"bb.sql.select"},
					"resources":           []string{"projects/hr/instances/prod-pg/databases/employee_db/schemas/public/tables/employee"},
				},
			},
			want: wantGrant,
		},
		// A grant must name a read-only statement and applies only to that exact
		// statement, so the batch as a whole can neither be granted nor use the grant.
		"the access check refused a read in a batch that also writes": {
			statement: "SELECT first_name FROM employee; UPDATE employee SET first_name = 'x'",
			result: map[string]any{
				"error":     "permission_denied: permission denied to access resources: [projects/hr/instances/prod-pg/databases/employee_db/schemas/public/tables/employee]",
				"statement": "SELECT first_name FROM employee;",
				"permissionDenied": map[string]any{
					"requiredPermissions": []string{"bb.sql.select"},
					"resources":           []string{"projects/hr/instances/prod-pg/databases/employee_db/schemas/public/tables/employee"},
				},
			},
			want: wantGrant,
		},
		"the access check refused a metadata read": {
			result: map[string]any{
				"error":     "permission_denied: permission denied to access resources: projects/hr/instances/prod-pg/databases/employee_db",
				"statement": "SELECT first_name FROM employee",
				"permissionDenied": map[string]any{
					"requiredPermissions": []string{"bb.sql.info"},
					"resources":           []string{"projects/hr/instances/prod-pg/databases/employee_db"},
				},
			},
			want: wantGrant,
		},
		// A grant from request_access_grant targets the queried database only.
		"the access check refused a table in another database": {
			result: map[string]any{
				"error": "permission_denied: permission denied to access resources: [projects/hr/instances/prod-pg/databases/payroll/schemas/public/tables/salary]",
				"permissionDenied": map[string]any{
					"requiredPermissions": []string{"bb.sql.select"},
					"resources":           []string{"projects/hr/instances/prod-pg/databases/payroll/schemas/public/tables/salary"},
				},
			},
		},
		"the access check refused tables in this and another database": {
			result: map[string]any{
				"error": "permission_denied: permission denied to access resources: [two tables]",
				"permissionDenied": map[string]any{
					"requiredPermissions": []string{"bb.sql.select"},
					"resources": []string{
						"projects/hr/instances/prod-pg/databases/employee_db/schemas/public/tables/employee",
						"projects/hr/instances/prod-pg/databases/employee_db2/schemas/public/tables/salary",
					},
				},
			},
		},
		// No grant lifts these: a grant's statement must be read-only, and a
		// granted target still needs bb.sql.explain from IAM.
		"the access check refused a write": {
			result: map[string]any{
				"error":            "permission_denied: permission denied to access resources: [projects/hr/instances/prod-pg/databases/employee_db/schemas/public/tables/employee]",
				"permissionDenied": map[string]any{"requiredPermissions": []string{"bb.sql.dml"}},
			},
		},
		"the access check refused an EXPLAIN": {
			result: map[string]any{
				"error":            "permission_denied: permission denied to access resources: projects/hr/instances/prod-pg/databases/employee_db",
				"permissionDenied": map[string]any{"requiredPermissions": []string{"bb.sql.explain"}},
			},
		},
		"a statement the engine rejected": {
			result: map[string]any{"error": `syntax error at or near "SELEC"`},
			want:   "Bytebase cannot always tell which statements of a failed request ran; if it held a write, check the data before you retry",
		},
	} {
		t.Run(name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "SQLService/Query") {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{tt.result}})
					return
				}
				mockListDatabases(databases).ServeHTTP(w, r)
			})
			s := newTestServerWithMock(t, handler)
			statement := tt.statement
			if statement == "" {
				statement = "SELECT first_name FROM employee"
			}

			result, _, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
				Database:  "employee_db",
				Statement: statement,
			})
			require.NoError(t, err)
			require.True(t, result.IsError)
			var denial queryError
			require.NoError(t, json.Unmarshal([]byte(result.Content[0].(*mcpsdk.TextContent).Text), &denial))
			require.Equal(t, tt.result["error"], denial.Message)
			require.Equal(t, tt.want, denial.Suggestion)
		})
	}
}

// --- Task 6 tests: masked values and timeout ---

func TestQueryDatabase_MaskedValues(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	qr := makeQueryResponse(
		[]string{"id", "ssn", "name"},
		[]string{"int4", "varchar", "varchar"},
		[][]any{{"1", "******", "**rn**"}},
		"0.010s",
	)
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	_, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id, ssn, name FROM users",
	})
	require.NoError(t, err)

	output, ok := structured.(*QueryOutput)
	require.True(t, ok)
	require.Equal(t, "******", output.Rows[0][1])
	require.Equal(t, "**rn**", output.Rows[0][2])
}

func TestQueryDatabase_Timeout(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "SQLService/Query") {
			// Sleep longer than the context timeout.
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			return
		}
		mockListDatabases(databases).ServeHTTP(w, r)
	})
	s := newTestServerWithMock(t, handler)

	// Use a very short timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resolved := &resolvedDatabase{
		resourceName: "instances/prod-pg/databases/employee_db",
		dataSourceID: "ds-admin-1",
	}
	_, err := s.executeQuery(ctx, resolved, QueryInput{Statement: "SELECT 1"}, 100)
	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Equal(t, "QUERY_ERROR", te.Code)
	require.Equal(t, "Bytebase cannot always tell which statements of a failed request ran; if it held a write, check the data before you retry", te.Suggestion)
}

// Query returns a result per statement and stops at the first failure, so the
// failure can follow results for statements that ran. The splitter keeps the
// whitespace before each statement after the first.
func TestQueryDatabase_FailureReportsWhatRan(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	const (
		ranAdvice     = "Bytebase ran the statements in ran, in a session that has since closed; check the data before you retry"
		unknownAdvice = "Bytebase cannot always tell which statements of a failed request ran; if it held a write, check the data before you retry"
		withheld      = "Rows returned by a procedure call or an OUTPUT clause are hidden because data masking can't be applied to them."
		wantGrant     = "in a Read-write session, request an access grant for failedStatement with request_access_grant and, " +
			"once approved, run it by itself (the project must allow just-in-time access); otherwise the user requests access in the SQL Editor"
	)
	for name, tt := range map[string]struct {
		statement      string
		results        []map[string]any
		wantMessage    string
		wantFailed     string
		wantRan        []string
		wantSuggestion string
	}{
		"a later statement failed": {
			statement: "INSERT INTO t VALUES (1); INSERT INTO t VALUES (2); INSERT INTO t VALUES (1), (3);",
			results: []map[string]any{
				{"columnNames": []string{"Affected Rows"}, "statement": "INSERT INTO t VALUES (1);"},
				{"columnNames": []string{"Affected Rows"}, "statement": " INSERT INTO t VALUES (2);"},
				{"error": `duplicate key value violates unique constraint "t_pkey"`, "statement": " INSERT INTO t VALUES (1), (3);"},
			},
			wantFailed:     "INSERT INTO t VALUES (1), (3);",
			wantRan:        []string{"INSERT INTO t VALUES (1);", "INSERT INTO t VALUES (2);"},
			wantSuggestion: ranAdvice,
		},
		"the first statement of a batch failed": {
			statement: "UPDATE t SET a = 1; UPDATE t SET b = 2;",
			results: []map[string]any{
				{"error": `column "a" does not exist`, "statement": "UPDATE t SET a = 1;"},
			},
			wantFailed:     "UPDATE t SET a = 1;",
			wantSuggestion: unknownAdvice,
		},
		"a single statement failed": {
			statement: "UPDATE t SET a = 1",
			results: []map[string]any{
				{"error": `column "a" does not exist`, "statement": "UPDATE t SET a = 1"},
			},
			wantFailed:     "UPDATE t SET a = 1",
			wantSuggestion: unknownAdvice,
		},
		"the access check refused a read after a statement ran": {
			statement: "INSERT INTO t VALUES (1); SELECT first_name FROM employee;",
			results: []map[string]any{
				{"columnNames": []string{"Affected Rows"}, "statement": "INSERT INTO t VALUES (1);"},
				{
					"error":     "permission_denied: permission denied to access resources: [instances/prod-pg/databases/employee_db/schemas/public/tables/employee]",
					"statement": " SELECT first_name FROM employee;",
					"permissionDenied": map[string]any{
						"requiredPermissions": []string{"bb.sql.select"},
						"resources":           []string{"instances/prod-pg/databases/employee_db/schemas/public/tables/employee"},
					},
				},
			},
			wantFailed:     "SELECT first_name FROM employee;",
			wantRan:        []string{"INSERT INTO t VALUES (1);"},
			wantSuggestion: ranAdvice + "; " + wantGrant,
		},
		"the access check refused a write after a statement ran": {
			statement: "INSERT INTO t VALUES (1); DROP TABLE t;",
			results: []map[string]any{
				{"columnNames": []string{"Affected Rows"}, "statement": "INSERT INTO t VALUES (1);"},
				{
					"error":            "permission_denied: permission denied to access resources: [instances/prod-pg/databases/employee_db/schemas/public/tables/t]",
					"statement":        " DROP TABLE t;",
					"permissionDenied": map[string]any{"requiredPermissions": []string{"bb.sql.ddl"}},
				},
			},
			wantFailed:     "DROP TABLE t;",
			wantRan:        []string{"INSERT INTO t VALUES (1);"},
			wantSuggestion: ranAdvice,
		},
		"a read over the size limit before a failed statement": {
			statement: "SELECT * FROM big; INSERT INTO t VALUES (1), (3);",
			results: []map[string]any{
				{"columnNames": []string{"id"}, "error": "Output of query exceeds max allowed output size of 10MB", "statement": "SELECT * FROM big;"},
				{"error": `duplicate key value violates unique constraint "t_pkey"`, "statement": " INSERT INTO t VALUES (1), (3);"},
			},
			wantFailed:     "INSERT INTO t VALUES (1), (3);",
			wantRan:        []string{"SELECT * FROM big;"},
			wantSuggestion: ranAdvice,
		},
		"rows withheld from a write that ran": {
			statement: "INSERT INTO t OUTPUT inserted.id VALUES (1); UPDATE t SET a = 1;",
			results: []map[string]any{
				{
					"columnNames": []string{"id"},
					"error":       "Rows returned by a procedure call or an OUTPUT clause are hidden because data masking can't be applied to them.",
					"statement":   "INSERT INTO t OUTPUT inserted.id VALUES (1);",
				},
				{"columnNames": []string{"Affected Rows"}, "statement": " UPDATE t SET a = 1;"},
			},
			wantMessage:    "Rows returned by a procedure call or an OUTPUT clause are hidden because data masking can't be applied to them.",
			wantFailed:     "INSERT INTO t OUTPUT inserted.id VALUES (1);",
			wantRan:        []string{"INSERT INTO t OUTPUT inserted.id VALUES (1);", "UPDATE t SET a = 1;"},
			wantSuggestion: ranAdvice,
		},
		"a procedure call that returned two result sets is named once per set": {
			statement: "EXEC dbo.two_sets;\nGO\nUPDATE dbo.t SET id = id OUTPUT inserted.id;\nSELECT id FROM dbo.t WHERE id = 1;",
			results: []map[string]any{
				{"columnNames": []string{"id"}, "error": withheld, "statement": "EXEC dbo.two_sets;"},
				{"columnNames": []string{"id"}, "error": withheld, "statement": "UPDATE dbo.t SET id = id OUTPUT inserted.id;"},
				{"columnNames": []string{"id"}, "statement": "SELECT id FROM dbo.t WHERE id = 1;"},
				{"columnNames": []string{"s"}, "error": withheld, "statement": "EXEC dbo.two_sets;"},
			},
			wantMessage: withheld,
			wantFailed:  "EXEC dbo.two_sets;",
			wantRan: []string{
				"EXEC dbo.two_sets;", "UPDATE dbo.t SET id = id OUTPUT inserted.id;", "SELECT id FROM dbo.t WHERE id = 1;", "EXEC dbo.two_sets;",
			},
			wantSuggestion: ranAdvice,
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := newTestServerWithMock(t, mockQueryServer(databases, map[string]any{"results": tt.results}))

			result, _, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
				Database:  "employee_db",
				Statement: tt.statement,
			})
			require.NoError(t, err)
			require.True(t, result.IsError)
			var failure queryError
			require.NoError(t, json.Unmarshal([]byte(result.Content[0].(*mcpsdk.TextContent).Text), &failure))
			wantMessage := tt.wantMessage
			if wantMessage == "" {
				wantMessage = fmt.Sprint(tt.results[len(tt.results)-1]["error"])
			}
			require.Equal(t, wantMessage, failure.Message)
			require.Equal(t, tt.wantFailed, failure.FailedStatement)
			require.Equal(t, tt.wantRan, failure.Ran)
			require.Equal(t, tt.wantSuggestion, failure.Suggestion)
		})
	}
}

// Query answers some failures with an HTTP error and no result. A 5xx can
// follow statements that ran; a 4xx refuses the request before it runs.
func TestQueryDatabase_HTTPErrorAdvice(t *testing.T) {
	for name, tt := range map[string]struct {
		engine         string
		status         int
		body           map[string]any
		wantSuggestion string
	}{
		"a SQL Server batch failed partway": {
			engine:         "MSSQL",
			status:         http.StatusInternalServerError,
			body:           map[string]any{"code": "internal", "message": "Violation of PRIMARY KEY constraint 'PK_t'."},
			wantSuggestion: "Bytebase cannot always tell which statements of a failed request ran; if it held a write, check the data before you retry",
		},
		"a PostgreSQL request failed with no result": {
			engine:         "POSTGRES",
			status:         http.StatusInternalServerError,
			body:           map[string]any{"code": "internal", "message": "failed to get database connection: connection refused"},
			wantSuggestion: "Bytebase cannot always tell which statements of a failed request ran; if it held a write, check the data before you retry",
		},
		"Query refused a statement it could not parse": {
			engine:         "POSTGRES",
			status:         http.StatusBadRequest,
			body:           map[string]any{"code": "invalid_argument", "message": `syntax error at or near "SELEC"`},
			wantSuggestion: "check your SQL syntax and try again",
		},
	} {
		t.Run(name, func(t *testing.T) {
			databases := []map[string]any{
				makeDatabase("instances/prod/databases/employee_db", "instances/prod", "projects/hr-system", tt.engine, "ds-admin-1"),
			}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "SQLService/Query") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.status)
					_ = json.NewEncoder(w).Encode(tt.body)
					return
				}
				mockListDatabases(databases).ServeHTTP(w, r)
			})
			s := newTestServerWithMock(t, handler)

			result, _, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
				Database:  "employee_db",
				Statement: "INSERT INTO t VALUES (1); INSERT INTO t VALUES (2);",
			})
			require.NoError(t, err)
			require.True(t, result.IsError)
			var failure toolError
			require.NoError(t, json.Unmarshal([]byte(result.Content[0].(*mcpsdk.TextContent).Text), &failure))
			require.Equal(t, tt.body["message"], failure.Message)
			require.Equal(t, tt.wantSuggestion, failure.Suggestion)
		})
	}
}

// protoQueryResponse encodes resp with protojson.MarshalOptions{}, the options
// connect-go's JSON codec uses for the Query response.
func protoQueryResponse(t *testing.T, resp *v1pb.QueryResponse) map[string]any {
	t.Helper()
	raw, err := protojson.MarshalOptions{}.Marshal(resp)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}

func TestMaskedColumns(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   []MaskedColumn
	}{
		{
			name:   "unmasked column entry is empty",
			result: `{"columnNames":["id","ssn"],"masked":[{},{"semanticTypeId":"ssn","semanticTypeTitle":"SSN","algorithm":"Full mask","context":"Global masking rule: PII"}]}`,
			want:   []MaskedColumn{{Column: "ssn", Index: 1, SemanticType: "SSN", Algorithm: "Full mask", Context: "Global masking rule: PII"}},
		},
		{
			name:   "column from several source columns has no semantic type",
			result: `{"columnNames":["contact"],"masked":[{"algorithm":"Full mask","context":"Multiple source columns"}]}`,
			want:   []MaskedColumn{{Column: "contact", Index: 0, Algorithm: "Full mask", Context: "Multiple source columns"}},
		},
		{
			name:   "list shorter than the columns",
			result: `{"columnNames":["ssn","id"],"masked":[{"semanticTypeId":"ssn","semanticTypeTitle":"SSN","algorithm":"Full mask"}]}`,
			want:   []MaskedColumn{{Column: "ssn", Index: 0, SemanticType: "SSN", Algorithm: "Full mask"}},
		},
		{
			name:   "list longer than the columns",
			result: `{"columnNames":["id"],"masked":[{},{"semanticTypeId":"ssn","algorithm":"Full mask"}]}`,
		},
		{
			name:   "columns sharing a name",
			result: `{"columnNames":["email","email"],"masked":[{},{"semanticTypeId":"email","semanticTypeTitle":"Email","algorithm":"Partial mask"}]}`,
			want:   []MaskedColumn{{Column: "email", Index: 1, SemanticType: "Email", Algorithm: "Partial mask"}},
		},
		{
			name:   "no masked field",
			result: `{"columnNames":["id"]}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var result queryResult
			require.NoError(t, json.Unmarshal([]byte(tc.result), &result))
			require.Equal(t, tc.want, maskedColumns(result.ColumnNames, result.Masked))
		})
	}
}

func TestQueryDatabase_MaskedColumnsNameTheirReasons(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	qr := protoQueryResponse(t, &v1pb.QueryResponse{
		Results: []*v1pb.QueryResult{{
			ColumnNames:     []string{"id", "ssn", "email"},
			ColumnTypeNames: []string{"int4", "varchar", "varchar"},
			Rows: []*v1pb.QueryRow{{Values: []*v1pb.RowValue{
				{Kind: &v1pb.RowValue_Int32Value{Int32Value: 1}},
				{Kind: &v1pb.RowValue_StringValue{StringValue: "******"}},
				{Kind: &v1pb.RowValue_StringValue{StringValue: "j***@example.com"}},
			}}},
			Masked: []*v1pb.MaskingReason{
				nil,
				{SemanticTypeId: "ssn", SemanticTypeTitle: "SSN", Algorithm: "Full mask", Context: "Global masking rule: PII"},
				{SemanticTypeId: "email", SemanticTypeTitle: "Email", Algorithm: "Partial mask", Context: "Column-level semantic type: Email"},
			},
		}},
	})
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	result, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id, ssn, email FROM users",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*QueryOutput)
	require.True(t, ok)
	require.Equal(t, []MaskedColumn{
		{Column: "ssn", Index: 1, SemanticType: "SSN", Algorithm: "Full mask", Context: "Global masking rule: PII"},
		{Column: "email", Index: 2, SemanticType: "Email", Algorithm: "Partial mask", Context: "Column-level semantic type: Email"},
	}, output.MaskedColumns)
	require.Empty(t, output.AppliedAccessGrant)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Masked columns: ssn, email.\n")
	require.Contains(t, text, `"maskedColumns":[{"column":"ssn","index":1,`)
}

func TestQueryDatabase_UnmaskedResultNamesNoMaskedColumns(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	tests := []struct {
		name       string
		masked     []*v1pb.MaskingReason
		wantOnWire any
	}{
		{
			name:       "masking ran and masked nothing",
			masked:     []*v1pb.MaskingReason{nil, nil},
			wantOnWire: []any{map[string]any{}, map[string]any{}},
		},
		{
			name:       "masking did not run",
			wantOnWire: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			qr := protoQueryResponse(t, &v1pb.QueryResponse{
				Results: []*v1pb.QueryResult{{
					ColumnNames:     []string{"id", "name"},
					ColumnTypeNames: []string{"int4", "varchar"},
					Rows: []*v1pb.QueryRow{{Values: []*v1pb.RowValue{
						{Kind: &v1pb.RowValue_Int32Value{Int32Value: 1}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: "John"}},
					}}},
					Masked: tc.masked,
				}},
			})
			results, ok := qr["results"].([]any)
			require.True(t, ok)
			wire, ok := results[0].(map[string]any)
			require.True(t, ok)
			require.Equal(t, tc.wantOnWire, wire["masked"])
			s := newTestServerWithMock(t, mockQueryServer(databases, qr))

			result, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
				Database:  "employee_db",
				Statement: "SELECT id, name FROM users",
			})
			require.NoError(t, err)
			require.False(t, result.IsError)

			output, ok := structured.(*QueryOutput)
			require.True(t, ok)
			require.Empty(t, output.MaskedColumns)

			text := result.Content[0].(*mcpsdk.TextContent).Text
			require.NotContains(t, text, "Masked columns")
			require.NotContains(t, text, "maskedColumns")
			require.NotContains(t, text, "appliedAccessGrant")
		})
	}
}

func TestQueryDatabase_AppliedAccessGrantPassesThrough(t *testing.T) {
	const grant = "projects/hr-system/accessGrants/g1"
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	qr := protoQueryResponse(t, &v1pb.QueryResponse{
		Results: []*v1pb.QueryResult{{
			ColumnNames:     []string{"ssn"},
			ColumnTypeNames: []string{"varchar"},
			Rows: []*v1pb.QueryRow{{Values: []*v1pb.RowValue{
				{Kind: &v1pb.RowValue_StringValue{StringValue: "123-45-6789"}},
			}}},
		}},
		AppliedAccessGrant: grant,
	})
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	result, structured, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT ssn FROM users",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*QueryOutput)
	require.True(t, ok)
	require.Equal(t, grant, output.AppliedAccessGrant)
	require.Empty(t, output.MaskedColumns)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, `"appliedAccessGrant":"`+grant+`"`)
	require.NotContains(t, text, "Masked columns")
}

func TestQueryDatabase_ForwardsSchemaAndContainerOnlyWhenSet(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	tests := []struct {
		name      string
		schema    string
		container string
	}{
		{name: "neither"},
		{name: "schema", schema: "public"},
		{name: "container", container: "employee"},
		{name: "both", schema: "public", container: "employee"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// require must not run on the transport's goroutine, so the handler
			// only records the body.
			var queryBody []byte
			listHandler := mockListDatabases(databases)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "SQLService/Query") {
					queryBody, _ = io.ReadAll(r.Body)
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(makeQueryResponse([]string{"id"}, []string{"int4"}, [][]any{{"1"}}, "0.001s"))
					return
				}
				listHandler.ServeHTTP(w, r)
			})
			s := newTestServerWithMock(t, handler)

			result, _, err := s.handleQueryDatabase(testContext(), nil, QueryInput{
				Database:  "employee_db",
				Statement: "SELECT id FROM users",
				Schema:    tc.schema,
				Container: tc.container,
			})
			require.NoError(t, err)
			require.False(t, result.IsError)

			// The server decodes the body with DiscardUnknown too, so a misnamed
			// key would lose the value without an error.
			var request v1pb.QueryRequest
			require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(queryBody, &request))
			for _, field := range []struct {
				name string
				want string
				got  *string
			}{
				{name: "schema", want: tc.schema, got: request.Schema},
				{name: "container", want: tc.container, got: request.Container},
			} {
				if field.want == "" {
					require.Nil(t, field.got, "%s must be left out when unset, not sent empty", field.name)
					continue
				}
				require.NotNil(t, field.got, "%s must reach QueryRequest.%s", field.name, field.name)
				require.Equal(t, field.want, *field.got)
			}
		})
	}
}
