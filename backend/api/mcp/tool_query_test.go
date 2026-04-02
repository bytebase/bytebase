package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
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

// mockListDatabases returns an HTTP handler that routes by URL path:
// - ListProjects: returns distinct projects extracted from the databases
// - ListDatabases: filters databases by the parent project and CEL filter
func mockListDatabases(databases []map[string]any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "ProjectService/ListProjects") {
			// Extract distinct projects from databases.
			seen := map[string]bool{}
			var projects []map[string]any
			for _, db := range databases {
				proj, ok := db["project"].(string)
				if ok && proj != "" && !seen[proj] {
					seen[proj] = true
					projects = append(projects, map[string]any{"name": proj})
				}
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"projects": projects})
			return
		}

		var reqBody struct {
			Parent string `json:"parent"`
			Filter string `json:"filter"`
		}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		// Filter by parent project.
		var scoped []map[string]any
		for _, db := range databases {
			proj, ok := db["project"].(string)
			if !ok || (reqBody.Parent != "" && proj != reqBody.Parent) {
				continue
			}
			scoped = append(scoped, db)
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
// Supports: name.matches("x"), instance == "instances/x"
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
		// Check name.matches("x") — substring match
		if idx := strings.Index(filter, `name.matches(`); idx >= 0 {
			start := idx + len(`name.matches("`)
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

func TestQueryDatabase_SingleMatch(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
		makeDatabase("instances/prod-pg/databases/orders_db", "instances/prod-pg", "projects/commerce", "POSTGRES", "ds-admin-2"),
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
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "Employee_DB"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
}

func TestQueryDatabase_SubstringMatch(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
}

func TestQueryDatabase_NotFound(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
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
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
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
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
		makeDatabase("instances/staging-mysql/databases/employee_db", "instances/staging-mysql", "projects/hr-system", "MYSQL", "ds-admin-2"),
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
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
		makeDatabase("instances/staging/databases/employee_db", "instances/staging", "projects/hr-system", "POSTGRES", "ds-admin-2"),
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
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
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
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
	qr := makeQueryResponse([]string{"id"}, []string{"int4"}, [][]any{{"1"}}, "0.001s")
	s := newTestServerWithMock(t, mockQueryServer(databases, qr))

	result, structured, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
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

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee"})
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

	result, structured, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
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

	result, structured, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
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

	result, structured, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
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

	result, structured, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
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

	result, structured, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
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

	result, _, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELEC * FROM users",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "syntax error")
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

	result, _, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT * FROM users",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "permission denied")
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

	_, structured, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
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
	_, err := s.executeQuery(ctx, resolved, "SELECT 1", 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "QUERY_ERROR")
}
