# MCP `query_database` Tool Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a composite `query_database` MCP tool that resolves databases and executes SQL queries in a single call, replacing the current 3+ tool call workflow.

**Architecture:** Extract shared HTTP plumbing from `tool_call.go` into `tool_http.go`, then build `tool_query.go` on top. The query tool calls internal Bytebase API endpoints (DatabaseService/ListDatabases, SQLService/Query) via HTTP, keeping MCP loosely coupled from service packages. Tests use `httptest.NewServer` to mock internal APIs.

**Tech Stack:** Go, MCP Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`), `github.com/stretchr/testify`, `net/http/httptest`

**Design doc:** `/Users/vincent/CLAUDE_BB/plans/2026-03-13-mcp-query-database-tool-design.md`

---

### Task 1: Extract shared HTTP helpers into `tool_http.go`

**Files:**
- Create: `backend/api/mcp/tool_http.go`
- Modify: `backend/api/mcp/tool_call.go`

**Step 1: Create `tool_http.go` with shared HTTP plumbing**

Read `backend/api/mcp/tool_call.go` for the existing HTTP logic (lines 70-148). Extract into a new file:

```go
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// apiResponse holds a raw API response.
type apiResponse struct {
	Status int
	Body   json.RawMessage
}

// toolError represents a structured error with actionable suggestion.
type toolError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func (e *toolError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Suggestion)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// apiRequest executes an authenticated POST to an internal Bytebase endpoint.
func (s *Server) apiRequest(ctx context.Context, path string, body any) (*apiResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal request body")
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte("{}"))
	}

	url := fmt.Sprintf("http://localhost:%d%s", s.profile.Port, path)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Connect-Protocol-Version", "1")

	if token := getAccessToken(ctx); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute API request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	return &apiResponse{
		Status: resp.StatusCode,
		Body:   json.RawMessage(respBody),
	}, nil
}

// parseError extracts a structured error from an API error response body.
func parseError(body json.RawMessage) string {
	var errResp struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Message != "" {
			return errResp.Message
		}
		if errResp.Code != "" {
			return errResp.Code
		}
	}
	return ""
}
```

**Step 2: Refactor `tool_call.go` to use `apiRequest`**

Replace the inline HTTP logic in `handleCallAPI` (lines 70-148) with a call to `s.apiRequest()`. Keep `CallInput`, `CallOutput`, `formatCallOutput`, `isBinaryContentType` in `tool_call.go`. Remove `withAccessToken`/`getAccessToken` from `tool_call.go` (they're already there, move the context helpers to `tool_http.go`).

The refactored `handleCallAPI` should look like:

```go
func (s *Server) handleCallAPI(ctx context.Context, _ *mcp.CallToolRequest, input CallInput) (*mcp.CallToolResult, any, error) {
	if input.OperationID == "" {
		return nil, nil, errors.New("operationId is required")
	}

	endpoint, ok := s.openAPIIndex.GetEndpoint(input.OperationID)
	if !ok {
		return nil, nil, errors.Errorf("unknown operation %s, use search_api to find valid operations", input.OperationID)
	}

	body := input.Body
	if body == nil {
		body = make(map[string]any)
	}

	resp, err := s.apiRequest(ctx, endpoint.Path, body)
	if err != nil {
		return nil, nil, err
	}

	// Check for binary response
	contentType := http.DetectContentType(resp.Body)
	if isBinaryContentType(contentType) {
		return nil, nil, errors.Errorf("binary response not supported (content-type: %s)", contentType)
	}

	// Parse JSON response
	var respJSON any
	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &respJSON); err != nil {
			respJSON = string(resp.Body)
		}
	}

	output := CallOutput{
		Status:   resp.Status,
		Response: respJSON,
	}

	if resp.Status >= 400 {
		output.Error = parseError(resp.Body)
		if output.Error == "" {
			output.Error = fmt.Sprintf("HTTP %d", resp.Status)
		}
	}

	text := formatCallOutput(output, endpoint)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
}
```

Note: The binary content type check changes slightly — the original checked the response `Content-Type` header, but `apiRequest` doesn't return headers. Two options:
- (a) Add `Headers http.Header` to `apiResponse` — preferred, minimal change.
- (b) Use `http.DetectContentType` on the body — less accurate.

Go with option (a): add `Headers http.Header` to `apiResponse` and set it in `apiRequest`.

**Step 3: Move context helpers to `tool_http.go`**

Move `accessTokenKey`, `withAccessToken`, `getAccessToken` from `tool_call.go` (lines 190-204) to `tool_http.go`. They're used by both `tool_call.go` and the new `tool_query.go`.

**Step 4: Run existing tests to verify refactor didn't break anything**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/... -run ^TestFormatCallOutput`

Expected: Both `TestFormatCallOutput` and `TestFormatCallOutputError` PASS.

**Step 5: Run full MCP test suite**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/...`

Expected: All existing tests PASS.

**Step 6: Run lint**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && golangci-lint run --allow-parallel-runners ./backend/api/mcp/...`

Expected: No lint errors.

**Step 7: Commit**

```bash
cd /Users/vincent/CLAUDE_BB/bytebase
git add backend/api/mcp/tool_http.go backend/api/mcp/tool_call.go
git commit -m "refactor(mcp): extract shared HTTP helpers into tool_http.go

Extract apiRequest, apiResponse, toolError, parseError, and context
helpers from tool_call.go into tool_http.go for reuse by new tools."
```

---

### Task 2: Write `tool_http_test.go`

**Files:**
- Create: `backend/api/mcp/tool_http_test.go`

**Step 1: Write tests for shared HTTP helpers**

```go
package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
)

// newTestServerWithMock creates an MCP Server pointing at a mock HTTP backend.
// Returns the Server and a cleanup function.
func newTestServerWithMock(t *testing.T, handler http.Handler) *Server {
	t.Helper()
	mockServer := httptest.NewServer(handler)
	t.Cleanup(mockServer.Close)

	// Parse the port from mock server URL
	// httptest.NewServer URL is like "http://127.0.0.1:PORT"
	var port int
	_, err := fmt.Sscanf(mockServer.URL, "http://127.0.0.1:%d", &port)
	require.NoError(t, err)

	profile := &config.Profile{Port: port}
	// We can't use NewServer here because it loads the OpenAPI index.
	// Instead, construct the Server directly with only what we need.
	return &Server{profile: profile}
}

func TestApiRequest_AuthForwarding(t *testing.T) {
	var receivedAuth string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	})

	s := newTestServerWithMock(t, handler)
	ctx := withAccessToken(context.Background(), "test-token-123")

	resp, err := s.apiRequest(ctx, "/test/path", map[string]string{"key": "value"})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status)
	require.Equal(t, "Bearer test-token-123", receivedAuth)
}

func TestApiRequest_ErrorParsing(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "permission denied", "code": "PERMISSION_DENIED"}`))
	})

	s := newTestServerWithMock(t, handler)
	resp, err := s.apiRequest(context.Background(), "/test/path", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.Status)

	msg := parseError(resp.Body)
	require.Equal(t, "permission denied", msg)
}

func TestApiRequest_RawMessage(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Integer value that would become float64 with map[string]any
		w.Write([]byte(`{"id": 12345, "count": 9999999999}`))
	})

	s := newTestServerWithMock(t, handler)
	resp, err := s.apiRequest(context.Background(), "/test/path", nil)
	require.NoError(t, err)

	// Verify raw JSON is preserved (not decoded to map[string]any)
	require.Equal(t, json.RawMessage(`{"id": 12345, "count": 9999999999}`), resp.Body)

	// Decode into typed struct to verify integers are preserved
	var result struct {
		ID    int64 `json:"id"`
		Count int64 `json:"count"`
	}
	err = json.Unmarshal(resp.Body, &result)
	require.NoError(t, err)
	require.Equal(t, int64(12345), result.ID)
	require.Equal(t, int64(9999999999), result.Count)
}
```

Note: `newTestServerWithMock` needs `"fmt"` in the import block for `fmt.Sscanf`.

**Step 2: Run tests**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/... -run ^TestApiRequest`

Expected: All 3 tests PASS.

**Step 3: Commit**

```bash
cd /Users/vincent/CLAUDE_BB/bytebase
git add backend/api/mcp/tool_http_test.go
git commit -m "test(mcp): add tests for shared HTTP helpers"
```

---

### Task 3: Implement `tool_query.go` — types and registration

**Files:**
- Create: `backend/api/mcp/tool_query.go`
- Modify: `backend/api/mcp/server.go` (line 63)

**Step 1: Write the failing test for tool registration**

Create `backend/api/mcp/tool_query_test.go`:

```go
package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func TestQueryDatabaseToolRegistered(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)
	require.NotNil(t, s)
	// The tool should be registered without error.
	// Full handler tests follow in subsequent tasks.
}
```

**Step 2: Run test to verify it passes (baseline)**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/... -run ^TestQueryDatabaseToolRegistered`

Expected: PASS (this is a baseline — NewServer already works).

**Step 3: Create `tool_query.go` with types, description, and registration**

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

const (
	defaultQueryLimit = 100
	maxQueryLimit     = 1000
	queryTimeout      = 30 * time.Second
	resolveTimeout    = 30 * time.Second
)

// QueryInput is the input for the query_database tool.
type QueryInput struct {
	// Database is the database name or substring to match (required).
	Database string `json:"database"`
	// Statement is the SQL query to execute (required).
	Statement string `json:"statement"`
	// Instance narrows database resolution to a specific instance (optional).
	Instance string `json:"instance,omitempty"`
	// Project narrows database resolution to a specific project (optional).
	Project string `json:"project,omitempty"`
	// Limit is the max rows to return (default: 100, max: 1000).
	Limit int `json:"limit,omitempty"`
}

// QueryOutput is the structured output for successful queries.
type QueryOutput struct {
	Columns     []string `json:"columns"`
	ColumnTypes []string `json:"columnTypes"`
	Rows        [][]any  `json:"rows"`
	RowCount    int      `json:"rowCount"`
	Truncated   bool     `json:"truncated"`
	LatencyMs   int64    `json:"latencyMs"`
}

// Candidate represents a database match for ambiguous resolution.
type Candidate struct {
	Database string `json:"database"`
	Instance string `json:"instance"`
	Project  string `json:"project"`
	Engine   string `json:"engine"`
}

const queryDatabaseDescription = `Execute SQL queries against Bytebase databases in a single call. Resolves database by name automatically.

| Parameter | Required | Description |
|-----------|----------|-------------|
| database | Yes | Database name or substring to match |
| statement | Yes | SQL query to execute |
| instance | No | Instance name to narrow search |
| project | No | Project name to narrow search |
| limit | No | Max rows (default: 100, max: 1000) |

**Examples:**
query_database(database="employees", statement="SELECT * FROM users LIMIT 10")
query_database(database="employee_db", instance="prod-pg", statement="SELECT count(*) FROM orders")`

func (s *Server) registerQueryTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "query_database",
		Description: queryDatabaseDescription,
	}, s.handleQueryDatabase)
}

func (s *Server) handleQueryDatabase(ctx context.Context, _ *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
	// Validate input
	if input.Database == "" {
		return nil, nil, errors.New("database is required")
	}
	if input.Statement == "" {
		return nil, nil, errors.New("statement is required")
	}

	// Normalize limit
	limit := input.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	if limit > maxQueryLimit {
		limit = maxQueryLimit
	}

	// Step 1: Resolve database
	resolveCtx, resolveCancel := context.WithTimeout(ctx, resolveTimeout)
	defer resolveCancel()

	resolved, err := s.resolveDatabase(resolveCtx, input)
	if err != nil {
		return nil, nil, err
	}

	// If ambiguous, return candidates
	if resolved.ambiguous {
		return s.formatAmbiguousResult(input.Database, resolved.candidates)
	}

	// Step 2: Execute query
	queryCtx, queryCancel := context.WithTimeout(ctx, queryTimeout)
	defer queryCancel()

	output, err := s.executeQuery(queryCtx, resolved, input.Statement, limit)
	if err != nil {
		text := fmt.Sprintf("Query failed on %s\n\n%s", resolved.resourceName, err.Error())
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
			IsError: true,
		}, nil, nil
	}

	// Step 3: Format output
	text := s.formatQueryOutput(input.Statement, resolved.resourceName, output)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
}
```

**Step 4: Add stub methods for `resolveDatabase`, `executeQuery`, `formatAmbiguousResult`, `formatQueryOutput`**

These go at the bottom of `tool_query.go`. They will be implemented in Tasks 4 and 5.

```go
// resolvedDatabase holds the result of database resolution.
type resolvedDatabase struct {
	resourceName string      // e.g., "instances/prod-pg/databases/employee_db"
	dataSourceID string      // datasource ID to use for queries
	ambiguous    bool        // true if multiple matches found
	candidates   []Candidate // populated when ambiguous
}

func (s *Server) resolveDatabase(_ context.Context, _ QueryInput) (*resolvedDatabase, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) executeQuery(_ context.Context, _ *resolvedDatabase, _ string, _ int) (*QueryOutput, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) formatAmbiguousResult(database string, candidates []Candidate) (*mcp.CallToolResult, any, error) {
	result := struct {
		Code       string      `json:"code"`
		Message    string      `json:"message"`
		Candidates []Candidate `json:"candidates"`
	}{
		Code:       "AMBIGUOUS_TARGET",
		Message:    fmt.Sprintf("Multiple databases match %q. Specify instance or project to narrow.", database),
		Candidates: candidates,
	}
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		IsError: true,
	}, result, nil
}

func (*Server) formatQueryOutput(statement, resourceName string, output *QueryOutput) string {
	var sb strings.Builder

	columnList := strings.Join(output.Columns, ", ")
	fmt.Fprintf(&sb, "Query: %s\n", statement)
	fmt.Fprintf(&sb, "Database: %s\n", resourceName)

	if output.Truncated {
		fmt.Fprintf(&sb, "Result: Showing %d rows, %d columns (%s) | %dms\n",
			output.RowCount, len(output.Columns), columnList, output.LatencyMs)
		sb.WriteString("Truncated: use limit param for more (max 1000).\n")
	} else {
		fmt.Fprintf(&sb, "Result: %d rows, %d columns (%s) | %dms\n",
			output.RowCount, len(output.Columns), columnList, output.LatencyMs)
	}

	sb.WriteString("\n")
	outputJSON, _ := json.Marshal(output)
	sb.Write(outputJSON)

	return sb.String()
}
```

**Step 5: Register the tool in `server.go`**

Modify `backend/api/mcp/server.go` line 63, add `s.registerQueryTool()`:

```go
func (s *Server) registerTools() {
	s.registerSearchTool()
	s.registerCallTool()
	s.registerSkillTool()
	s.registerQueryTool()
}
```

**Step 6: Run tests and lint**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/... -run ^TestQueryDatabaseToolRegistered`

Expected: PASS.

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && golangci-lint run --allow-parallel-runners ./backend/api/mcp/...`

Expected: No errors (stub methods may have unused params — prefix with `_` if lint complains).

**Step 7: Commit**

```bash
cd /Users/vincent/CLAUDE_BB/bytebase
git add backend/api/mcp/tool_query.go backend/api/mcp/tool_query_test.go backend/api/mcp/server.go
git commit -m "feat(mcp): add query_database tool skeleton with types and registration"
```

---

### Task 4: Implement database resolution (`resolveDatabase`)

**Files:**
- Modify: `backend/api/mcp/tool_query.go`
- Modify: `backend/api/mcp/tool_query_test.go`

**Step 1: Write failing tests for database resolution**

Add to `tool_query_test.go`. These tests use `httptest.NewServer` to mock the `DatabaseService/ListDatabases` endpoint.

Key: The internal endpoint path is `/bytebase.v1.DatabaseService/ListDatabases` (Connect RPC POST). The request body has `parent` and optionally `filter`. The response is JSON with a `databases` array.

Reference the proto structure from the exploration: each database has:
- `name`: resource name like `"instances/prod-pg/databases/employee_db"`
- `project`: like `"projects/hr-system"`
- `instanceResource.dataSources`: array with `{id, type}` where type is `"ADMIN"` or `"READ_ONLY"`
- `instanceResource.engine`: like `"POSTGRES"`
- `instanceResource.name`: like `"instances/prod-pg"`

```go
// mockListDatabases returns an http.Handler that responds to ListDatabases requests
// with the given database entries.
func mockListDatabases(databases []map[string]any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bytebase.v1.DatabaseService/ListDatabases" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{"databases": databases}
		json.NewEncoder(w).Encode(resp)
	})
}

func makeDatabase(name, instanceName, project, engine, dsID, dsType string) map[string]any {
	db := map[string]any{
		"name":    name,
		"project": project,
		"instanceResource": map[string]any{
			"name":   instanceName,
			"engine": engine,
			"dataSources": []map[string]any{
				{"id": dsID, "type": dsType},
			},
		},
	}
	return db
}

func makeDatabaseWithDualDS(name, instanceName, project, engine, adminDSID, readOnlyDSID string) map[string]any {
	db := map[string]any{
		"name":    name,
		"project": project,
		"instanceResource": map[string]any{
			"name":   instanceName,
			"engine": engine,
			"dataSources": []map[string]any{
				{"id": adminDSID, "type": "ADMIN"},
				{"id": readOnlyDSID, "type": "READ_ONLY"},
			},
		},
	}
	return db
}

func TestQueryDatabase_SingleMatch(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	s := newTestServerWithMock(t, mockListDatabases([]map[string]any{db}))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee_db"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
	require.Equal(t, "admin-ds", resolved.dataSourceID)
}

func TestQueryDatabase_CaseInsensitiveMatch(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	s := newTestServerWithMock(t, mockListDatabases([]map[string]any{db}))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "Employee_DB"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
}

func TestQueryDatabase_SubstringMatch(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	s := newTestServerWithMock(t, mockListDatabases([]map[string]any{db}))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee"})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
}

func TestQueryDatabase_NotFound(t *testing.T) {
	s := newTestServerWithMock(t, mockListDatabases([]map[string]any{}))

	_, err := s.resolveDatabase(context.Background(), QueryInput{Database: "nonexistent"})
	require.Error(t, err)
	var te *toolError
	require.ErrorAs(t, err, &te)
	require.Equal(t, "DATABASE_NOT_FOUND", te.Code)
}

func TestQueryDatabase_Ambiguous(t *testing.T) {
	db1 := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"ds1", "ADMIN",
	)
	db2 := makeDatabase(
		"instances/staging/databases/employee_db",
		"instances/staging", "projects/hr", "POSTGRES",
		"ds2", "ADMIN",
	)
	s := newTestServerWithMock(t, mockListDatabases([]map[string]any{db1, db2}))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee_db"})
	require.NoError(t, err)
	require.True(t, resolved.ambiguous)
	require.Len(t, resolved.candidates, 2)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.candidates[0].Database)
	require.Equal(t, "instances/staging/databases/employee_db", resolved.candidates[1].Database)
}

func TestQueryDatabase_AmbiguousWithInstance(t *testing.T) {
	db1 := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"ds1", "ADMIN",
	)
	db2 := makeDatabase(
		"instances/staging/databases/employee_db",
		"instances/staging", "projects/hr", "POSTGRES",
		"ds2", "ADMIN",
	)
	s := newTestServerWithMock(t, mockListDatabases([]map[string]any{db1, db2}))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{
		Database: "employee_db",
		Instance: "prod-pg",
	})
	require.NoError(t, err)
	require.False(t, resolved.ambiguous)
	require.Equal(t, "instances/prod-pg/databases/employee_db", resolved.resourceName)
}

func TestQueryDatabase_ReadOnlyDatasource(t *testing.T) {
	db := makeDatabaseWithDualDS(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "readonly-ds",
	)
	s := newTestServerWithMock(t, mockListDatabases([]map[string]any{db}))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee_db"})
	require.NoError(t, err)
	require.Equal(t, "readonly-ds", resolved.dataSourceID)
}

func TestQueryDatabase_AdminFallback(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	s := newTestServerWithMock(t, mockListDatabases([]map[string]any{db}))

	resolved, err := s.resolveDatabase(context.Background(), QueryInput{Database: "employee_db"})
	require.NoError(t, err)
	require.Equal(t, "admin-ds", resolved.dataSourceID)
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/... -run "^TestQueryDatabase_(SingleMatch|CaseInsensitiveMatch|SubstringMatch|NotFound|Ambiguous|ReadOnly|AdminFallback)"`

Expected: FAIL — `resolveDatabase` returns "not implemented".

**Step 3: Implement `resolveDatabase`**

Replace the stub in `tool_query.go`:

```go
func (s *Server) resolveDatabase(ctx context.Context, input QueryInput) (*resolvedDatabase, error) {
	// Build filter for ListDatabases
	body := map[string]any{
		"parent": "workspaces/-",
	}

	// Call DatabaseService/ListDatabases
	resp, err := s.apiRequest(ctx, "/bytebase.v1.DatabaseService/ListDatabases", body)
	if err != nil {
		return nil, err
	}
	if resp.Status >= 400 {
		msg := parseError(resp.Body)
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.Status)
		}
		return nil, &toolError{
			Code:    "API_ERROR",
			Message: msg,
		}
	}

	// Parse response
	var listResp struct {
		Databases []struct {
			Name             string `json:"name"`
			Project          string `json:"project"`
			InstanceResource struct {
				Name        string `json:"name"`
				Engine      string `json:"engine"`
				DataSources []struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"dataSources"`
			} `json:"instanceResource"`
		} `json:"databases"`
	}
	if err := json.Unmarshal(resp.Body, &listResp); err != nil {
		return nil, errors.Wrap(err, "failed to parse database list")
	}

	// Extract the short database name from resource name.
	// "instances/prod-pg/databases/employee_db" -> "employee_db"
	extractDBName := func(resourceName string) string {
		parts := strings.Split(resourceName, "/")
		if len(parts) >= 4 {
			return parts[3]
		}
		return resourceName
	}

	// Extract short instance name from resource name.
	// "instances/prod-pg" -> "prod-pg"
	extractInstanceName := func(resourceName string) string {
		parts := strings.Split(resourceName, "/")
		if len(parts) >= 2 {
			return parts[1]
		}
		return resourceName
	}

	// Extract short project name from resource name.
	// "projects/hr" -> "hr"
	extractProjectName := func(resourceName string) string {
		parts := strings.Split(resourceName, "/")
		if len(parts) >= 2 {
			return parts[1]
		}
		return resourceName
	}

	// Filter by instance and project if provided
	type dbEntry struct {
		resourceName string
		instanceName string
		projectName  string
		engine       string
		dataSources  []struct {
			ID   string
			Type string
		}
	}

	var filtered []dbEntry
	for _, db := range listResp.Databases {
		instName := extractInstanceName(db.InstanceResource.Name)
		projName := extractProjectName(db.Project)

		if input.Instance != "" && !strings.EqualFold(instName, input.Instance) {
			continue
		}
		if input.Project != "" && !strings.EqualFold(projName, input.Project) {
			continue
		}

		entry := dbEntry{
			resourceName: db.Name,
			instanceName: instName,
			projectName:  projName,
			engine:       db.InstanceResource.Engine,
		}
		for _, ds := range db.InstanceResource.DataSources {
			entry.dataSources = append(entry.dataSources, struct {
				ID   string
				Type string
			}{ID: ds.ID, Type: ds.Type})
		}
		filtered = append(filtered, entry)
	}

	// Strict matching: exact -> case-insensitive exact -> substring
	matchTiers := []func(dbName, query string) bool{
		func(dbName, query string) bool { return dbName == query },
		func(dbName, query string) bool { return strings.EqualFold(dbName, query) },
		func(dbName, query string) bool { return strings.Contains(strings.ToLower(dbName), strings.ToLower(query)) },
	}

	var matches []dbEntry
	for _, matchFn := range matchTiers {
		matches = nil
		for _, entry := range filtered {
			dbName := extractDBName(entry.resourceName)
			if matchFn(dbName, input.Database) {
				matches = append(matches, entry)
			}
		}
		if len(matches) > 0 {
			break
		}
	}

	if len(matches) == 0 {
		return nil, &toolError{
			Code:       "DATABASE_NOT_FOUND",
			Message:    fmt.Sprintf("no database matching %q found", input.Database),
			Suggestion: "use call_api(operationId=\"DatabaseService/ListDatabases\", body={\"parent\": \"workspaces/-\"}) to list available databases",
		}
	}

	if len(matches) > 1 {
		candidates := make([]Candidate, 0, len(matches))
		for _, m := range matches {
			candidates = append(candidates, Candidate{
				Database: m.resourceName,
				Instance: m.instanceName,
				Project:  m.projectName,
				Engine:   m.engine,
			})
		}
		return &resolvedDatabase{ambiguous: true, candidates: candidates}, nil
	}

	// Single match — select datasource
	match := matches[0]
	dsID := selectDataSource(match.dataSources)

	return &resolvedDatabase{
		resourceName: match.resourceName,
		dataSourceID: dsID,
	}, nil
}

// selectDataSource picks READ_ONLY if available, otherwise falls back to ADMIN.
func selectDataSource(dataSources []struct{ ID, Type string }) string {
	var adminID string
	for _, ds := range dataSources {
		if ds.Type == "READ_ONLY" {
			return ds.ID
		}
		if ds.Type == "ADMIN" {
			adminID = ds.ID
		}
	}
	return adminID
}
```

**Step 4: Run resolution tests**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/... -run "^TestQueryDatabase_(SingleMatch|CaseInsensitiveMatch|SubstringMatch|NotFound|Ambiguous|ReadOnly|AdminFallback)"`

Expected: All PASS.

**Step 5: Run lint**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && golangci-lint run --allow-parallel-runners ./backend/api/mcp/...`

Expected: No errors.

**Step 6: Commit**

```bash
cd /Users/vincent/CLAUDE_BB/bytebase
git add backend/api/mcp/tool_query.go backend/api/mcp/tool_query_test.go
git commit -m "feat(mcp): implement database resolution for query_database tool

Resolves databases with strict matching (exact > case-insensitive > substring).
Auto-selects READ_ONLY datasource with ADMIN fallback.
Returns AMBIGUOUS_TARGET with candidates when multiple databases match."
```

---

### Task 5: Implement query execution (`executeQuery`)

**Files:**
- Modify: `backend/api/mcp/tool_query.go`
- Modify: `backend/api/mcp/tool_query_test.go`

**Step 1: Write failing tests for query execution**

Add to `tool_query_test.go`. These need a mock that handles both ListDatabases AND SQLService/Query.

```go
// mockQueryServer returns a handler that serves both ListDatabases and Query endpoints.
func mockQueryServer(databases []map[string]any, queryResponse map[string]any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/bytebase.v1.DatabaseService/ListDatabases":
			json.NewEncoder(w).Encode(map[string]any{"databases": databases})
		case "/bytebase.v1.SQLService/Query":
			json.NewEncoder(w).Encode(queryResponse)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})
}

// makeQueryResponse builds a mock SQLService/Query response.
func makeQueryResponse(columns []string, columnTypes []string, rows [][]map[string]any, latency string) map[string]any {
	queryRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		queryRows = append(queryRows, map[string]any{"values": row})
	}
	return map[string]any{
		"results": []map[string]any{
			{
				"columnNames":     columns,
				"columnTypeNames": columnTypes,
				"rows":            queryRows,
				"rowsCount":       fmt.Sprintf("%d", len(rows)),
				"latency":         latency,
				"statement":       "SELECT ...",
			},
		},
	}
}

func TestQueryDatabase_FullFlow(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	rows := [][]map[string]any{
		{{"stringValue": "1"}, {"stringValue": "John"}, {"stringValue": "john@example.com"}},
		{{"stringValue": "2"}, {"stringValue": "Jane"}, {"stringValue": "jane@example.com"}},
	}
	qr := makeQueryResponse(
		[]string{"id", "name", "email"},
		[]string{"int4", "varchar", "varchar"},
		rows, "0.012s",
	)
	s := newTestServerWithMock(t, mockQueryServer([]map[string]any{db}, qr))

	result, output, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT * FROM users",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.IsError)

	qo, ok := output.(*QueryOutput)
	require.True(t, ok)
	require.Equal(t, []string{"id", "name", "email"}, qo.Columns)
	require.Equal(t, []string{"int4", "varchar", "varchar"}, qo.ColumnTypes)
	require.Len(t, qo.Rows, 2)
	require.Equal(t, 2, qo.RowCount)
	require.False(t, qo.Truncated)
}

func TestQueryDatabase_EmptyResult(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	qr := makeQueryResponse(
		[]string{"id", "name"},
		[]string{"int4", "varchar"},
		[][]map[string]any{}, "0.005s",
	)
	s := newTestServerWithMock(t, mockQueryServer([]map[string]any{db}, qr))

	result, output, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT * FROM users WHERE id = -1",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	qo := output.(*QueryOutput)
	require.Len(t, qo.Rows, 0)
	require.Equal(t, 0, qo.RowCount)
	require.False(t, qo.Truncated)
}

func TestQueryDatabase_Truncation(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	// Generate 150 rows
	rows := make([][]map[string]any, 150)
	for i := range rows {
		rows[i] = []map[string]any{{"stringValue": fmt.Sprintf("%d", i+1)}}
	}
	qr := makeQueryResponse([]string{"id"}, []string{"int4"}, rows, "0.230s")
	s := newTestServerWithMock(t, mockQueryServer([]map[string]any{db}, qr))

	result, output, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id FROM users",
		// default limit = 100
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	qo := output.(*QueryOutput)
	require.Equal(t, 100, qo.RowCount)
	require.Len(t, qo.Rows, 100)
	require.True(t, qo.Truncated)

	// Verify text header mentions truncation
	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Truncated")
}

func TestQueryDatabase_CustomLimit(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	rows := make([][]map[string]any, 100)
	for i := range rows {
		rows[i] = []map[string]any{{"stringValue": fmt.Sprintf("%d", i+1)}}
	}
	qr := makeQueryResponse([]string{"id"}, []string{"int4"}, rows, "0.1s")
	s := newTestServerWithMock(t, mockQueryServer([]map[string]any{db}, qr))

	result, output, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id FROM users",
		Limit:     50,
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	qo := output.(*QueryOutput)
	require.Equal(t, 50, qo.RowCount)
	require.Len(t, qo.Rows, 50)
	require.True(t, qo.Truncated)
}

func TestQueryDatabase_LimitCappedAt1000(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	rows := make([][]map[string]any, 5)
	for i := range rows {
		rows[i] = []map[string]any{{"stringValue": fmt.Sprintf("%d", i+1)}}
	}
	qr := makeQueryResponse([]string{"id"}, []string{"int4"}, rows, "0.1s")
	s := newTestServerWithMock(t, mockQueryServer([]map[string]any{db}, qr))

	// limit=2000 should be capped to 1000, but we only have 5 rows so no truncation
	_, output, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id FROM users",
		Limit:     2000,
	})
	require.NoError(t, err)

	qo := output.(*QueryOutput)
	require.Equal(t, 5, qo.RowCount)
	require.False(t, qo.Truncated)
}

func TestQueryDatabase_QueryError(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/bytebase.v1.DatabaseService/ListDatabases":
			json.NewEncoder(w).Encode(map[string]any{
				"databases": []map[string]any{db},
			})
		case "/bytebase.v1.SQLService/Query":
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "syntax error at or near \"SELEC\"",
				"code":    "INVALID_ARGUMENT",
			})
		}
	})
	s := newTestServerWithMock(t, handler)

	result, _, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELEC * FROM users",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Query failed")
	require.Contains(t, text, "syntax error")
}

func TestQueryDatabase_PermissionDenied(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/bytebase.v1.DatabaseService/ListDatabases":
			json.NewEncoder(w).Encode(map[string]any{
				"databases": []map[string]any{db},
			})
		case "/bytebase.v1.SQLService/Query":
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "permission denied",
				"code":    "PERMISSION_DENIED",
			})
		}
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
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/... -run "^TestQueryDatabase_(FullFlow|EmptyResult|Truncation|CustomLimit|LimitCapped|QueryError|PermissionDenied)"`

Expected: FAIL — `executeQuery` returns "not implemented".

**Step 3: Implement `executeQuery`**

Replace the stub in `tool_query.go`:

```go
func (s *Server) executeQuery(ctx context.Context, resolved *resolvedDatabase, statement string, limit int) (*QueryOutput, error) {
	body := map[string]any{
		"name":         resolved.resourceName,
		"dataSourceId": resolved.dataSourceID,
		"statement":    statement,
	}

	resp, err := s.apiRequest(ctx, "/bytebase.v1.SQLService/Query", body)
	if err != nil {
		return nil, err
	}
	if resp.Status >= 400 {
		msg := parseError(resp.Body)
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.Status)
		}
		return nil, &toolError{
			Code:       "QUERY_ERROR",
			Message:    msg,
			Suggestion: "check SQL syntax and permissions",
		}
	}

	// Parse the query response
	var queryResp struct {
		Results []struct {
			ColumnNames     []string          `json:"columnNames"`
			ColumnTypeNames []string          `json:"columnTypeNames"`
			Rows            []json.RawMessage `json:"rows"`
			RowsCount       json.Number       `json:"rowsCount"`
			Latency         string            `json:"latency"`
			Error           string            `json:"error"`
			Statement       string            `json:"statement"`
		} `json:"results"`
	}
	if err := json.Unmarshal(resp.Body, &queryResp); err != nil {
		return nil, errors.Wrap(err, "failed to parse query response")
	}

	if len(queryResp.Results) == 0 {
		return &QueryOutput{}, nil
	}

	result := queryResp.Results[0]

	// Check for query-level error
	if result.Error != "" {
		return nil, &toolError{
			Code:       "QUERY_ERROR",
			Message:    result.Error,
			Suggestion: "check SQL syntax for the database engine",
		}
	}

	// Parse latency
	var latencyMs int64
	if result.Latency != "" {
		latencyMs = parseLatencyMs(result.Latency)
	}

	// Flatten rows: each row is {"values": [{"stringValue": "x"}, {"int32Value": 1}, ...]}
	allRows, err := flattenRows(result.Rows)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse query rows")
	}

	// Apply limit and truncation
	truncated := len(allRows) > limit
	if truncated {
		allRows = allRows[:limit]
	}

	return &QueryOutput{
		Columns:     result.ColumnNames,
		ColumnTypes: result.ColumnTypeNames,
		Rows:        allRows,
		RowCount:    len(allRows),
		Truncated:   truncated,
		LatencyMs:   latencyMs,
	}, nil
}

// flattenRows converts protobuf RowValue oneofs into plain Go values.
func flattenRows(rawRows []json.RawMessage) ([][]any, error) {
	var rows [][]any
	for _, rawRow := range rawRows {
		var row struct {
			Values []json.RawMessage `json:"values"`
		}
		if err := json.Unmarshal(rawRow, &row); err != nil {
			return nil, err
		}

		flatRow := make([]any, 0, len(row.Values))
		for _, rawVal := range row.Values {
			val := flattenRowValue(rawVal)
			flatRow = append(flatRow, val)
		}
		rows = append(rows, flatRow)
	}
	return rows, nil
}

// flattenRowValue extracts the plain value from a protobuf RowValue oneof.
func flattenRowValue(raw json.RawMessage) any {
	var valMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &valMap); err != nil {
		return string(raw)
	}

	// Priority order for oneof fields
	oneofFields := []string{
		"nullValue", "boolValue", "bytesValue", "doubleValue", "floatValue",
		"int32Value", "int64Value", "stringValue", "uint32Value", "uint64Value",
		"valueValue", "timestampValue", "timestampTzValue",
	}

	for _, field := range oneofFields {
		rawVal, ok := valMap[field]
		if !ok {
			continue
		}

		if field == "nullValue" {
			return nil
		}

		// For timestamp types, extract the string representation
		if field == "timestampValue" || field == "timestampTzValue" {
			var ts struct {
				Value string `json:"value"`
			}
			if err := json.Unmarshal(rawVal, &ts); err == nil && ts.Value != "" {
				return ts.Value
			}
			// Fall through to raw string
		}

		// Try to decode as a basic type
		var val any
		if err := json.Unmarshal(rawVal, &val); err == nil {
			return val
		}
		return string(rawVal)
	}

	return nil
}

// parseLatencyMs parses a duration string like "0.012s" into milliseconds.
func parseLatencyMs(latency string) int64 {
	latency = strings.TrimSpace(latency)
	if strings.HasSuffix(latency, "s") {
		latency = strings.TrimSuffix(latency, "s")
		var seconds float64
		if _, err := fmt.Sscanf(latency, "%f", &seconds); err == nil {
			return int64(seconds * 1000)
		}
	}
	return 0
}
```

**Step 4: Run all query tests**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/... -run "^TestQueryDatabase_"`

Expected: All PASS.

**Step 5: Run full test suite and lint**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/...`

Expected: All tests PASS.

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && golangci-lint run --allow-parallel-runners ./backend/api/mcp/...`

Expected: No lint errors.

**Step 6: Commit**

```bash
cd /Users/vincent/CLAUDE_BB/bytebase
git add backend/api/mcp/tool_query.go backend/api/mcp/tool_query_test.go
git commit -m "feat(mcp): implement query execution for query_database tool

Executes SQL via SQLService/Query, flattens protobuf RowValue oneofs
into plain values, enforces row limits with truncation metadata."
```

---

### Task 6: Add masked value and timeout tests

**Files:**
- Modify: `backend/api/mcp/tool_query_test.go`

**Step 1: Write masked value passthrough test**

```go
func TestQueryDatabase_MaskedValues(t *testing.T) {
	db := makeDatabase(
		"instances/prod-pg/databases/employee_db",
		"instances/prod-pg", "projects/hr", "POSTGRES",
		"admin-ds", "ADMIN",
	)
	// Masked values should pass through unchanged
	rows := [][]map[string]any{
		{{"stringValue": "1"}, {"stringValue": "******"}, {"stringValue": "**rn**"}},
	}
	qr := makeQueryResponse(
		[]string{"id", "ssn", "name"},
		[]string{"int4", "varchar", "varchar"},
		rows, "0.005s",
	)
	s := newTestServerWithMock(t, mockQueryServer([]map[string]any{db}, qr))

	_, output, err := s.handleQueryDatabase(context.Background(), nil, QueryInput{
		Database:  "employee_db",
		Statement: "SELECT id, ssn, name FROM users",
	})
	require.NoError(t, err)

	qo := output.(*QueryOutput)
	require.Len(t, qo.Rows, 1)
	require.Equal(t, "******", qo.Rows[0][1])
	require.Equal(t, "**rn**", qo.Rows[0][2])
}

func TestQueryDatabase_Timeout(t *testing.T) {
	// Slow handler that blocks longer than the context timeout
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bytebase.v1.DatabaseService/ListDatabases" {
			// Simulate slow response
			select {
			case <-r.Context().Done():
				return
			case <-time.After(5 * time.Second):
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"databases": []any{}})
			}
		}
	})
	s := newTestServerWithMock(t, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := s.resolveDatabase(ctx, QueryInput{Database: "employee_db"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "context deadline exceeded")
}
```

**Step 2: Run new tests**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/... -run "^TestQueryDatabase_(MaskedValues|Timeout)$"`

Expected: Both PASS.

**Step 3: Run full test suite**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/...`

Expected: All PASS.

**Step 4: Commit**

```bash
cd /Users/vincent/CLAUDE_BB/bytebase
git add backend/api/mcp/tool_query_test.go
git commit -m "test(mcp): add masked values and timeout tests for query_database"
```

---

### Task 7: Update skill description and build verification

**Files:**
- Modify: `backend/api/mcp/tool_skill.go` (line 35)

**Step 1: Update `get_skill` description to mention `query_database`**

In `tool_skill.go`, update `getSkillDescription` to mention the new tool. Users should know they can use `query_database` directly instead of following the `query` skill workflow.

Change line 31-35:
```go
const getSkillDescription = `Get step-by-step guides for Bytebase tasks.

**Tip:** Use query_database tool for SQL queries — it handles database resolution automatically.

**Workflow:** get_skill("task") → search_api(operationId) → call_api(...)

Skills: query, database-change, grant-permission`
```

**Step 2: Run all tests**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go test -v -count=1 ./backend/api/mcp/...`

Expected: All PASS.

**Step 3: Build the project**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`

Expected: Build succeeds.

**Step 4: Run lint**

Run: `cd /Users/vincent/CLAUDE_BB/bytebase && golangci-lint run --allow-parallel-runners ./backend/api/mcp/...`

Expected: No lint errors.

**Step 5: Commit**

```bash
cd /Users/vincent/CLAUDE_BB/bytebase
git add backend/api/mcp/tool_skill.go
git commit -m "docs(mcp): update get_skill description to mention query_database tool"
```

---

## Summary of files

| Action | File |
|--------|------|
| Create | `backend/api/mcp/tool_http.go` |
| Create | `backend/api/mcp/tool_http_test.go` |
| Create | `backend/api/mcp/tool_query.go` |
| Create | `backend/api/mcp/tool_query_test.go` |
| Modify | `backend/api/mcp/tool_call.go` (refactor to use shared helpers) |
| Modify | `backend/api/mcp/server.go` (register new tool) |
| Modify | `backend/api/mcp/tool_skill.go` (update description) |
