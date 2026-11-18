package databricks

// parity_workspace_test.go is the workspace the cutover's differential test runs
// against: an HTTP server that answers the Databricks REST API the driver uses —
// the SQL Statement Execution API, the Unity Catalog listings, and the workspace
// listing Ping reads. No account, token, warehouse or network is needed, and the
// answers are the ones recorded from the pinned workspace in
// omni/databricks/driver/VERIFY.md, narrowed to what the cutover compares.
//
// It lives here rather than beside the golden so that the recording (made from
// the pre-cutover driver) and the replay (made from the adapter) ask the same
// server the same questions; a request either driver sends that this server does
// not answer fails the recording, not just the comparison.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
)

const (
	parityCatalog      = "workspace"
	parityWarehouseID  = "45847ad2fbd0b316"
	parityToken        = "parity-token"
	paritySQLVersion   = "4.2.0 0000000000000000000000000000000000000000"
	parityQueryProbe   = "SELECT 1"
	parityFailingQuery = "SELECT broken"
)

// parityStatementRequest is one statement a driver sent, as the SQL Statement
// Execution API states it. It is what the differential compares first: the same
// statements, in the same order, with the same catalog, schema and row limit.
type parityStatementRequest struct {
	Statement   string `json:"statement"`
	WarehouseID string `json:"warehouse_id"`
	Catalog     string `json:"catalog"`
	Schema      string `json:"schema"`
	RowLimit    int64  `json:"row_limit"`
}

// parityRequest is one HTTP request the server answered, reduced to what a
// driver controls: the method and the route, and for a statement the catalog,
// schema, row limit and text the driver asked the warehouse to run.
type parityRequest struct {
	Method    string `json:"method"`
	Route     string `json:"route"`
	Statement string `json:"statement,omitempty"`
	Catalog   string `json:"catalog,omitempty"`
	Schema    string `json:"schema,omitempty"`
	RowLimit  int64  `json:"row_limit,omitempty"`
}

// parityWorkspace is the fake workspace. Every request it answers is recorded,
// in order.
type parityWorkspace struct {
	server *httptest.Server

	mu       sync.Mutex
	requests []parityRequest
}

func newParityWorkspace(t *testing.T) *parityWorkspace {
	t.Helper()
	workspace := &parityWorkspace{}
	workspace.server = httptest.NewServer(http.HandlerFunc(workspace.serve))
	t.Cleanup(workspace.server.Close)
	return workspace
}

// host is the workspace URL a driver opens.
func (w *parityWorkspace) host() string {
	return w.server.URL
}

// takeRequests returns the requests recorded since the last call.
func (w *parityWorkspace) takeRequests() []parityRequest {
	w.mu.Lock()
	defer w.mu.Unlock()
	requests := w.requests
	w.requests = nil
	return requests
}

func (w *parityWorkspace) record(request parityRequest) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.requests = append(w.requests, request)
}

func (w *parityWorkspace) serve(rw http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	route := r.URL.Path

	// The SDK asks for the workspace's host metadata while it resolves its
	// configuration. The previous driver's client did so on a best-effort basis
	// and the omni driver's resolves nothing, so the answer is "no metadata" and
	// the request is not part of the comparison.
	if strings.HasSuffix(route, "/.well-known/databricks-config") {
		writeJSON(rw, http.StatusNotFound, map[string]any{"error_code": "NOT_FOUND", "message": "no host metadata"})
		return
	}

	switch {
	case r.Method == http.MethodPost && route == "/api/2.0/sql/statements":
		statement, ok := executeParityStatement(rw, body)
		if !ok {
			return
		}
		w.record(parityRequest{
			Method:    r.Method,
			Route:     route,
			Statement: statement.Statement,
			Catalog:   statement.Catalog,
			Schema:    statement.Schema,
			RowLimit:  statement.RowLimit,
		})
	case r.Method == http.MethodGet && route == "/api/2.0/workspace/list":
		w.record(parityRequest{Method: r.Method, Route: route})
		writeJSON(rw, http.StatusOK, map[string]any{"objects": []any{}})
	case r.Method == http.MethodGet && route == "/api/2.1/unity-catalog/catalogs":
		w.record(parityRequest{Method: r.Method, Route: route})
		writeJSON(rw, http.StatusOK, map[string]any{"catalogs": parityCatalogs()})
	case r.Method == http.MethodGet && route == "/api/2.1/unity-catalog/schemas":
		w.record(parityRequest{Method: r.Method, Route: route})
		writeJSON(rw, http.StatusOK, map[string]any{"schemas": paritySchemas(r.URL.Query().Get("catalog_name"))})
	case r.Method == http.MethodGet && route == "/api/2.1/unity-catalog/tables":
		w.record(parityRequest{Method: r.Method, Route: route})
		writeJSON(rw, http.StatusOK, map[string]any{
			"tables": parityTables(r.URL.Query().Get("catalog_name"), r.URL.Query().Get("schema_name")),
		})
	case r.Method == http.MethodGet && strings.HasPrefix(route, "/api/2.1/unity-catalog/tables/"):
		w.record(parityRequest{Method: r.Method, Route: route})
		fullName := strings.TrimPrefix(route, "/api/2.1/unity-catalog/tables/")
		table, ok := parityTable(fullName)
		if !ok {
			writeJSON(rw, http.StatusNotFound, map[string]any{"error_code": "NOT_FOUND", "message": "table " + fullName})
			return
		}
		writeJSON(rw, http.StatusOK, table)
	default:
		w.record(parityRequest{Method: r.Method, Route: route})
		writeJSON(rw, http.StatusNotFound, map[string]any{"error_code": "NOT_FOUND", "message": "no route " + route})
	}
}

// executeParityStatement answers one statement's execution request and returns
// the request the driver sent.
func executeParityStatement(rw http.ResponseWriter, body []byte) (parityStatementRequest, bool) {
	var request parityStatementRequest
	if err := json.Unmarshal(body, &request); err != nil {
		writeJSON(rw, http.StatusBadRequest, map[string]any{"error_code": "BAD_REQUEST", "message": err.Error()})
		return request, false
	}
	answer, ok := parityAnswer(request.Statement)
	if !ok {
		writeJSON(rw, http.StatusOK, serviceFailure(request.Statement))
		return request, true
	}
	writeJSON(rw, http.StatusOK, answer)
	return request, true
}

// serviceFailure is the API's answer to a statement the engine refuses: a
// failed status whose ServiceError the SDK turns into "FAILED: <code> <message>".
func serviceFailure(statement string) map[string]any {
	return map[string]any{
		"statement_id": "01f0bad",
		"status": map[string]any{
			"state": "FAILED",
			"error": map[string]any{
				"error_code": "BAD_REQUEST",
				"message":    "no canned answer for statement " + statement,
			},
		},
	}
}

// parityAnswer returns the recorded answer to a statement, or reports that the
// corpus does not hold one.
func parityAnswer(statement string) (map[string]any, bool) {
	switch normalizeStatement(statement) {
	case parityQueryProbe, "SELECT 2":
		return succeeded(
			[]map[string]any{column("1", "INT", "INT")},
			[][]any{{"1"}},
		), true
	case "SELECT CAST(NULL AS INT) AS n":
		return succeeded(
			[]map[string]any{column("n", "INT", "INT")},
			[][]any{{nil}},
		), true
	case "SELECT 'x' AS s":
		return succeeded(
			[]map[string]any{column("s", "STRING", "STRING")},
			[][]any{{"x"}},
		), true
	case "SELECT 321 AS a, 'b' AS s, CAST(NULL AS STRING) AS z":
		return succeeded(
			[]map[string]any{
				column("a", "INT", "INT"),
				column("s", "STRING", "STRING"),
				column("z", "STRING", "STRING"),
			},
			[][]any{{"321", "b", nil}},
		), true
	case "SELECT id, label, amount FROM workspace.omni_record.lookup ORDER BY id":
		return succeeded(
			[]map[string]any{
				column("id", "INT", "INT"),
				column("label", "STRING", "STRING"),
				column("amount", "DECIMAL", "DECIMAL(12,2)"),
			},
			[][]any{{"1", "first", "1.50"}, {"2", "second", "2.25"}},
		), true
	case "SELECT * FROM workspace.omni_record.lookup WHERE 1 = 0":
		return succeeded(
			[]map[string]any{
				column("id", "INT", "INT"),
				column("label", "STRING", "STRING"),
			},
			nil,
		), true
	case "INSERT INTO workspace.omni_record.lookup (id, label) VALUES (1, 'x'), (2, 'y')":
		return succeeded(
			[]map[string]any{column("num_affected_rows", "LONG", "LONG")},
			[][]any{{"2"}},
		), true
	case "UPDATE workspace.omni_record.lookup SET label = 'z' WHERE id = 1":
		return succeeded(
			[]map[string]any{column("num_affected_rows", "LONG", "LONG")},
			[][]any{{"1"}},
		), true
	case "CREATE TABLE workspace.omni_record.created (id INT)":
		return emptyResult(), true
	case "CREATE TABLE workspace.omni_record.t AS SELECT 1":
		return emptyResult(), true
	case "EXPLAIN FORMATTED " + parityQueryProbe:
		return succeeded(
			[]map[string]any{column("plan", "STRING", "STRING")},
			[][]any{{"== Physical Plan ==\nProject [1]\n"}},
		), true
	case parityFailingQuery:
		return serviceFailure(parityFailingQuery), true
	case "-- lead\nSELECT 1", "SELECT 1 -- tail", "DELIMITER ;;\nSELECT 1":
		// A comment line, a trailing comment and a DELIMITER block are carried
		// into the statement the driver sends; the answer is the probe's.
		return succeeded(
			[]map[string]any{column("1", "INT", "INT")},
			[][]any{{"1"}},
		), true
	case "SHOW CREATE TABLE `workspace`.`omni_record`.`lookup`":
		return succeeded(
			[]map[string]any{column("createtab_stmt", "STRING", "STRING")},
			[][]any{{"CREATE TABLE workspace.omni_record.lookup (\n  `id` INT,\n  `label` STRING NOT NULL\n)\nUSING delta\nCOMMENT 'a synced comment'"}},
		), true
	case "SHOW CREATE TABLE `workspace`.`omni_record`.`facts`":
		return succeeded(
			[]map[string]any{column("createtab_stmt", "STRING", "STRING")},
			[][]any{{"CREATE TABLE workspace.omni_record.facts (\n  `id` BIGINT\n)\nUSING DELTA"}},
		), true
	case "SHOW CREATE TABLE `workspace`.`omni_record`.`recent`":
		return succeeded(
			[]map[string]any{column("createtab_stmt", "STRING", "STRING")},
			[][]any{{"CREATE VIEW omni_record.recent AS SELECT id FROM workspace.omni_record.facts"}},
		), true
	case "SHOW CREATE TABLE `workspace`.`omni_record`.`totals`":
		return succeeded(
			[]map[string]any{column("createtab_stmt", "STRING", "STRING")},
			[][]any{{"CREATE MATERIALIZED VIEW workspace.omni_record.totals AS SELECT id FROM workspace.omni_record.facts"}},
		), true
	case "SHOW CREATE TABLE `workspace`.`omni_record`.`ext`":
		return succeeded(
			[]map[string]any{column("createtab_stmt", "STRING", "STRING")},
			[][]any{{"CREATE EXTERNAL TABLE workspace.omni_record.ext (\n  `id` INT\n)\nLOCATION 's3://example-bucket/ext'"}},
		), true
	case "DESC FORMATTED `workspace`.`omni_record`.`totals`":
		return succeeded(
			[]map[string]any{
				column("col_name", "STRING", "STRING"),
				column("data_type", "STRING", "STRING"),
			},
			[][]any{
				{"id", "int"},
				{"Table Properties", "[serialization.format=1,delta.minReaderVersion=1]"},
			},
		), true
	case "SELECT VERSION()":
		return succeeded(
			[]map[string]any{column("version()", "STRING", "STRING")},
			[][]any{{paritySQLVersion}},
		), true
	}
	return nil, false
}

func succeeded(columns []map[string]any, rows [][]any) map[string]any {
	manifest := map[string]any{
		"format": "JSON_ARRAY",
		"schema": map[string]any{
			"column_count": len(columns),
			"columns":      columns,
		},
		"truncated": false,
	}
	response := map[string]any{
		"statement_id": "01f0abc",
		"status":       map[string]any{"state": "SUCCEEDED"},
		"manifest":     manifest,
	}
	if rows == nil {
		response["result"] = map[string]any{"data_array": []any{}}
		return response
	}
	response["result"] = map[string]any{"data_array": rows}
	return response
}

// emptyResult is the API's answer to a statement that answered no result set:
// no manifest, and an empty inline result.
func emptyResult() map[string]any {
	return map[string]any{
		"statement_id": "01f0ddl",
		"status":       map[string]any{"state": "SUCCEEDED"},
		"result":       map[string]any{"data_array": []any{}},
	}
}

func column(name, typeName, typeText string) map[string]any {
	return map[string]any{
		"name":      name,
		"type_name": typeName,
		"type_text": typeText,
	}
}

// normalizeStatement reduces a statement to the canned answer's key: the old
// splitter left a statement's trailing semicolon and leading whitespace, the
// contract's trimmer does not.
func normalizeStatement(statement string) string {
	return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(statement), ";"))
}

func parityCatalogs() []map[string]any {
	return []map[string]any{
		{"name": "system", "full_name": "system", "catalog_type": "SYSTEM_CATALOG"},
		{"name": parityCatalog, "full_name": parityCatalog, "catalog_type": "MANAGED_CATALOG"},
	}
}

func paritySchemas(catalog string) []map[string]any {
	if catalog != parityCatalog {
		return []map[string]any{{"name": "information_schema", "catalog_name": catalog}}
	}
	return []map[string]any{
		{"name": "default", "catalog_name": catalog},
		{"name": "information_schema", "catalog_name": catalog},
		{"name": "omni_record", "catalog_name": catalog, "comment": "the cutover's corpus"},
	}
}

func parityTables(catalog, schema string) []map[string]any {
	if catalog != parityCatalog || schema != "omni_record" {
		return []map[string]any{}
	}
	var tables []map[string]any
	for _, name := range []string{"ext", "facts", "lookup", "recent", "stream", "totals"} {
		table, _ := parityTable(catalog + ".omni_record." + name)
		tables = append(tables, table)
	}
	return tables
}

// parityTable returns one table's Unity Catalog answer. A streaming table is
// listed and skipped by both drivers.
func parityTable(fullName string) (map[string]any, bool) {
	if !strings.HasPrefix(fullName, parityCatalog+".omni_record.") {
		return nil, false
	}
	name := strings.TrimPrefix(fullName, parityCatalog+".omni_record.")
	table := map[string]any{
		"catalog_name":       parityCatalog,
		"schema_name":        "omni_record",
		"name":               name,
		"full_name":          fullName,
		"table_id":           "table-" + name,
		"table_type":         "MANAGED",
		"data_source_format": "DELTA",
	}
	switch name {
	case "lookup":
		table["comment"] = "a synced comment"
		table["columns"] = []map[string]any{
			{"name": "id", "position": 0, "type_name": "INT", "type_text": "INT", "nullable": true},
			{"name": "label", "position": 1, "type_name": "STRING", "type_text": "STRING", "nullable": false, "comment": "a column comment"},
			{"name": "amount", "position": 2, "type_name": "DECIMAL", "type_text": "DECIMAL(12,2)", "nullable": true},
		}
	case "facts":
		table["columns"] = []map[string]any{
			{"name": "id", "position": 0, "type_name": "BIGINT", "type_text": "BIGINT", "nullable": true},
			{"name": "lookup_id", "position": 1, "type_name": "INT", "type_text": "INT", "nullable": true},
		}
	case "recent":
		table["table_type"] = "VIEW"
		table["view_definition"] = "SELECT id, lookup_id FROM workspace.omni_record.facts WHERE id > 0"
		table["columns"] = []map[string]any{
			{"name": "id", "position": 0, "type_name": "BIGINT", "type_text": "BIGINT", "nullable": true},
			{"name": "lookup_id", "position": 1, "type_name": "INT", "type_text": "INT", "nullable": true},
		}
	case "totals":
		table["table_type"] = "MATERIALIZED_VIEW"
		table["view_definition"] = "SELECT lookup_id, SUM(id) AS total FROM workspace.omni_record.facts GROUP BY lookup_id"
		table["columns"] = []map[string]any{
			{"name": "lookup_id", "position": 0, "type_name": "INT", "type_text": "INT", "nullable": true},
			{"name": "total", "position": 1, "type_name": "LONG", "type_text": "LONG", "nullable": true},
		}
	case "ext":
		table["table_type"] = "EXTERNAL"
		table["columns"] = []map[string]any{
			{"name": "id", "position": 0, "type_name": "INT", "type_text": "INT", "nullable": true},
		}
	case "stream":
		table["table_type"] = "STREAMING_TABLE"
		table["columns"] = []map[string]any{
			{"name": "id", "position": 0, "type_name": "INT", "type_text": "INT", "nullable": true},
		}
	default:
		return nil, false
	}
	return table, true
}

func writeJSON(rw http.ResponseWriter, status int, body any) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	_ = json.NewEncoder(rw).Encode(body)
}

// sortedRoutes returns the routes in the request log, sorted, for an assertion
// that does not depend on listing order.
func sortedRoutes(requests []parityRequest) []string {
	seen := map[string]bool{}
	var routes []string
	for _, request := range requests {
		route := request.Method + " " + request.Route
		if !seen[route] {
			seen[route] = true
			routes = append(routes, route)
		}
	}
	slices.Sort(routes)
	return routes
}

// parityQueryCase is one QueryConn call in the corpus: the script, and the query
// context the editor would send with it.
type parityQueryCase struct {
	name    string
	script  string
	schema  string
	limit   int
	explain bool
}

// parityQueryCorpus is the query path's differential corpus. It covers the
// splitter's boundaries, the two statement shapes the reader distinguishes, the
// cell kinds the conversion maps, an empty result, a write that answers a
// `num_affected_rows` column, a statement the workspace refuses, and the explain
// rewrite.
var parityQueryCorpus = []parityQueryCase{
	{name: "probe", script: parityQueryProbe},
	{name: "probe with a row limit", script: parityQueryProbe, limit: 100},
	{name: "probe in a schema", script: parityQueryProbe, schema: "omni_record"},
	{name: "a null cell", script: "SELECT CAST(NULL AS INT) AS n"},
	{name: "a string cell", script: "SELECT 'x' AS s"},
	{name: "typed cells", script: "SELECT 321 AS a, 'b' AS s, CAST(NULL AS STRING) AS z"},
	{name: "rows", script: "SELECT id, label, amount FROM workspace.omni_record.lookup ORDER BY id"},
	{name: "no rows", script: "SELECT * FROM workspace.omni_record.lookup WHERE 1 = 0"},
	{name: "no rows before a statement", script: "SELECT * FROM workspace.omni_record.lookup WHERE 1 = 0;\nSELECT 1"},
	{name: "two statements", script: "SELECT 1;\nSELECT 2"},
	{name: "two statements on one line", script: "SELECT 1; SELECT 2"},
	{name: "a trailing semicolon", script: "SELECT 1;"},
	{name: "a leading comment", script: "-- lead\nSELECT 1"},
	{name: "a trailing comment", script: "SELECT 1 -- tail"},
	{name: "a lone terminator between two", script: "SELECT 1;\n;\nSELECT 2"},
	{name: "a delimiter block", script: "DELIMITER ;;\nSELECT 1;;\nDELIMITER ;"},
	{name: "an insert", script: "INSERT INTO workspace.omni_record.lookup (id, label) VALUES (1, 'x'), (2, 'y')"},
	{name: "an update", script: "UPDATE workspace.omni_record.lookup SET label = 'z' WHERE id = 1"},
	{name: "a DDL statement", script: "CREATE TABLE workspace.omni_record.created (id INT)"},
	{name: "an explain request", script: parityQueryProbe, explain: true},
	{name: "a statement the caller wrote as EXPLAIN", script: "EXPLAIN FORMATTED SELECT 1"},
	{name: "a statement the workspace refuses", script: parityFailingQuery},
	{name: "a failure between two statements", script: "SELECT 1;\nSELECT broken;\nSELECT 2"},
}

// parityExecuteCase is one Execute call in the corpus.
type parityExecuteCase struct {
	name   string
	script string
}

// parityExecuteCorpus is the change-script corpus: a write, a write with a DDL
// statement, a script the workspace refuses, and a script carrying the
// transaction directive the product still sends.
var parityExecuteCorpus = []parityExecuteCase{
	{name: "an insert", script: "INSERT INTO workspace.omni_record.lookup (id, label) VALUES (1, 'x'), (2, 'y')"},
	{name: "an update", script: "UPDATE workspace.omni_record.lookup SET label = 'z' WHERE id = 1"},
	{name: "a DDL statement", script: "CREATE TABLE workspace.omni_record.created (id INT)"},
	{name: "a write after a read", script: "SELECT 1;\nINSERT INTO workspace.omni_record.lookup (id, label) VALUES (1, 'x'), (2, 'y')"},
	{name: "a transaction directive", script: "-- txn-mode=off\nINSERT INTO workspace.omni_record.lookup (id, label) VALUES (1, 'x'), (2, 'y')"},
	{name: "a statement the workspace refuses", script: parityFailingQuery},
	{name: "a failure after a write", script: "INSERT INTO workspace.omni_record.lookup (id, label) VALUES (1, 'x'), (2, 'y');\nSELECT broken"},
}
