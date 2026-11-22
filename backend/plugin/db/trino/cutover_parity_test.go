package trino

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// The golden file holds what the pre-cutover driver sent over the wire and what
// it returned for a corpus of Execute and QueryConn calls. It was recorded from
// the old path against the same pinned Trino before that path was deleted, so
// this test is the cutover's differential check even though only one driver is
// left to run. A case whose behaviour omni deliberately changes is named in
// knownDifferences below and pinned by its own expectation instead.

type parityRequest struct {
	Method  string            `json:"method"`
	URI     string            `json:"uri"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

type parityResult struct {
	Statement       string     `json:"statement"`
	ColumnNames     []string   `json:"column_names"`
	ColumnTypeNames []string   `json:"column_type_names"`
	Rows            [][]string `json:"rows"`
	Error           string     `json:"error"`
	RowsCount       int64      `json:"rows_count"`
	QueryPlan       string     `json:"query_plan"`
}

type parityQuery struct {
	Name      string          `json:"name"`
	Script    string          `json:"script"`
	Requests  []parityRequest `json:"requests"`
	Results   []parityResult  `json:"results"`
	CallError string          `json:"call_error"`
}

type parityExecute struct {
	Name     string          `json:"name"`
	Script   string          `json:"script"`
	Requests []parityRequest `json:"requests"`
	Affected int64           `json:"affected"`
	Error    string          `json:"error"`
	Logs     []string        `json:"logs"`
}

type parityRecording struct {
	InstanceRequests []parityRequest `json:"instance_requests"`
	Instance         json.RawMessage `json:"instance"`
	Execute          []parityExecute `json:"execute"`
	Queries          []parityQuery   `json:"queries"`
}

type parityRecorder struct {
	mu       sync.Mutex
	requests []parityRequest
}

func (w *parityRecorder) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.requests = nil
}

func (w *parityRecorder) take() []parityRequest {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := w.requests
	w.requests = nil
	return out
}

func recordParityRequest(w *parityRecorder, r *http.Request, body []byte) {
	if r.Method != http.MethodPost {
		return
	}
	headers := map[string]string{}
	for _, name := range []string{"X-Trino-Catalog", "X-Trino-Schema", "X-Trino-User", "X-Trino-Source", "X-Trino-Language"} {
		if value := r.Header.Get(name); value != "" {
			headers[name] = value
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.requests = append(w.requests, parityRequest{
		Method:  r.Method,
		URI:     r.URL.RequestURI(),
		Body:    string(body),
		Headers: headers,
	})
}

func startParityProxy(t *testing.T, target *url.URL) (*httptest.Server, *parityRecorder) {
	t.Helper()
	recorder := &parityRecorder{}
	proxy := &httputil.ReverseProxy{Rewrite: func(request *httputil.ProxyRequest) {
		request.SetURL(target)
		// Trino refuses a request that carries X-Forwarded-For unless the
		// server has been configured to accept it.
		request.Out.Header.Del("X-Forwarded-For")
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		recordParityRequest(recorder, r, body)
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server, recorder
}

// assertKnownQueryDifference pins what the new path does for a case whose result
// moves with omni, so the allowlist cannot hide a later regression.
func assertKnownQueryDifference(t *testing.T, got parityQuery) {
	t.Helper()
	switch got.Name {
	case "exact numeric and timestamps":
		require.Len(t, got.Results, 1)
		require.Equal(t, "string_value:\"1.25\"", got.Results[0].Rows[0][0])
	case "classification values", "classification with", "classification table":
		require.Len(t, got.Results, 1)
		require.NotEmpty(t, got.Results[0].Rows)
	case "limit before a trailing comment":
		require.Len(t, got.Requests, 1)
		require.Contains(t, got.Requests[0].Body, " LIMIT 2 -- note")
	case "explain text":
		require.Len(t, got.Results, 1)
		require.Equal(t, "format:TEXT", got.Results[0].QueryPlan)
	case "explain json":
		require.Len(t, got.Results, 1)
		require.Equal(t, "format:JSON", got.Results[0].QueryPlan)
	case "explain of an explain analyze":
		require.Len(t, got.Results, 1)
		require.Contains(t, got.Results[0].Error, "EXPLAIN ANALYZE executes")
	case "truncation":
		require.Len(t, got.Results, 1)
		require.NotEmpty(t, got.Results[0].Error)
	default:
		t.Fatalf("no assertion for known difference %q", got.Name)
	}
}

// normalizeLogs collapses the extra spaces protobuf's text format inserts at
// random between fields, which makes TaskRunLog.String() unsuitable for an
// exact comparison.
func normalizeLogs(logs []string) []string {
	out := make([]string, len(logs))
	for i, log := range logs {
		out[i] = strings.Join(strings.Fields(log), " ")
	}
	return out
}

func summarizeParityResult(result *v1pb.QueryResult) parityResult {
	rows := make([][]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		values := make([]string, 0, len(row.Values))
		for _, value := range row.Values {
			values = append(values, value.String())
		}
		rows = append(rows, values)
	}
	plan := ""
	if result.QueryPlan != nil {
		plan = result.QueryPlan.String()
	}
	return parityResult{
		Statement:       result.Statement,
		ColumnNames:     result.ColumnNames,
		ColumnTypeNames: result.ColumnTypeNames,
		Rows:            rows,
		Error:           result.Error,
		RowsCount:       result.RowsCount,
		QueryPlan:       plan,
	}
}

// knownDifferences names the corpus cases whose result or request text moves
// with the omni driver, and says what the new path does instead. Each entry is
// asserted below, so a change to one of these expectations still fails.
var knownDifferences = map[string]string{
	"multi statement with a comment":  "the omni splitter keeps the comment in the command text; Bytebase's old splitter stripped it",
	"exact numeric and timestamps":    "DECIMAL crosses as text and timestamps keep the value's own precision and zone",
	"classification values":           "omni reads VALUES as a query and returns its rows; Bytebase ran it as a write",
	"classification with":             "omni reads WITH as a query and returns its rows; Bytebase ran it as a write",
	"classification table":            "omni reads TABLE as a query and returns its rows; Bytebase ran it as a write",
	"limit before a trailing comment": "the omni splitter keeps the trailing comment and places LIMIT before it",
	"explain text":                    "the omni driver reports the plan in QueryPlan as well as in the rows; Bytebase only returned the rows",
	"explain json":                    "the omni driver reports the plan in QueryPlan as well as in the rows; Bytebase only returned the rows",
	"explain of an explain analyze":   "both refuse to plan an EXPLAIN ANALYZE; the omni driver's refusal names the reason",
	"truncation":                      "the contract counts bytes, so the cut point can differ from proto.Size at power-of-two rows",
}

func TestCutoverParityAgainstRecordedOldPath(t *testing.T) {
	ctx := context.Background()
	goldenBytes, err := os.ReadFile(filepath.Join("testdata", "cutover_parity.json"))
	require.NoError(t, err)
	var golden parityRecording
	require.NoError(t, json.Unmarshal(goldenBytes, &golden))

	c := testcontainer.SharedTrinoContainer(t)
	// Make the corpus re-runnable: the recording started from a fresh catalog.
	direct, err := (&Driver{}).Open(ctx, storepb.Engine_TRINO, db.ConnectionConfig{
		DataSource: &storepb.DataSource{Host: c.GetHost(), Port: c.GetPort(), Username: c.GetUsername(), Database: "memory"},
	})
	require.NoError(t, err)
	for _, table := range []string{"cutover_parity_t", "cutover_query_ddl"} {
		_, err = direct.Execute(ctx, fmt.Sprintf("DROP TABLE IF EXISTS memory.default.%s", table), db.ExecuteOptions{})
		require.NoError(t, err)
	}
	require.NoError(t, direct.Close(ctx))

	target, err := url.Parse(fmt.Sprintf("http://%s:%s", c.GetHost(), c.GetPort()))
	require.NoError(t, err)
	proxy, recorder := startParityProxy(t, target)
	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)

	opened, err := (&Driver{}).Open(ctx, storepb.Engine_TRINO, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     proxyURL.Hostname(),
			Port:     proxyURL.Port(),
			Username: c.GetUsername(),
			Database: "memory",
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "memory"},
	})
	require.NoError(t, err)
	defer opened.Close(ctx)

	recorder.reset()
	instance, err := opened.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, golden.InstanceRequests, recorder.take())
	instanceJSON, err := json.Marshal(instance)
	require.NoError(t, err)
	require.JSONEq(t, string(golden.Instance), string(instanceJSON))

	execute := func(name, script string) parityExecute {
		recorder.reset()
		var logs []string
		affected, err := opened.Execute(ctx, script, db.ExecuteOptions{
			LogCommandStatement: true,
			CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
				logs = append(logs, log.String())
				return nil
			},
		})
		entry := parityExecute{
			Name:     name,
			Script:   script,
			Requests: recorder.take(),
			Affected: affected,
			Logs:     logs,
		}
		if err != nil {
			entry.Error = err.Error()
		}
		return entry
	}

	if len(golden.Execute) != 3 {
		t.Fatalf("golden Execute entries = %d, want 3", len(golden.Execute))
	}
	for _, want := range golden.Execute {
		got := execute(want.Name, want.Script)
		if reason, known := knownDifferences[got.Name]; known {
			t.Logf("%s: known difference: %s", got.Name, reason)
			if got.Name == "multi statement with a comment" {
				require.Contains(t, got.Requests[0].Body, "-- a note")
			}
			continue
		}
		require.Equal(t, want.Requests, got.Requests, got.Name)
		require.Equal(t, want.Affected, got.Affected, got.Name)
		require.Equal(t, want.Error, got.Error, got.Name)
		require.Equal(t, normalizeLogs(want.Logs), normalizeLogs(got.Logs), got.Name)
	}

	qc := func() db.QueryContext { return db.QueryContext{MaximumSQLResultSize: 1 << 20} }
	queries := []struct {
		name    string
		script  string
		context db.QueryContext
	}{
		{name: "select", script: "SELECT id, name FROM memory.default.cutover_parity_t ORDER BY id", context: qc()},
		{name: "types", script: "SELECT CAST(42 AS INTEGER) AS c_integer, CAST(1.25 AS DOUBLE) AS c_double, CAST('hi' AS VARCHAR) AS c_varchar, DATE '2026-01-02' AS c_date, TIME '12:34:56.789' AS c_time, X'0102' AS c_varbinary", context: qc()},
		{name: "exact numeric and timestamps", script: "SELECT CAST(1.25 AS DECIMAL(5,2)) AS c_decimal, TIMESTAMP '2026-01-02 12:34:56' AS c_timestamp, TIMESTAMP '2026-01-02 12:34:56.789' AS c_timestamp_fraction, TIMESTAMP '2026-01-02 12:34:56 +02:00' AS c_timestamp_tz", context: qc()},
		{name: "ddl through the query path", script: "CREATE TABLE memory.default.cutover_query_ddl (id BIGINT)", context: qc()},
		{name: "classification show", script: "SHOW CATALOGS", context: qc()},
		{name: "classification describe", script: "DESCRIBE memory.default.cutover_parity_t", context: qc()},
		{name: "classification values", script: "VALUES 1", context: qc()},
		{name: "classification with", script: "WITH t AS (SELECT 1 AS x) SELECT * FROM t", context: qc()},
		{name: "classification table", script: "TABLE memory.default.cutover_parity_t", context: qc()},
		{name: "classification set", script: "SET SESSION query_max_run_time = '1m'", context: qc()},
		{name: "limit", script: "SELECT id FROM memory.default.cutover_parity_t ORDER BY id", context: db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 20}},
		{name: "limit kept", script: "SELECT id FROM memory.default.cutover_parity_t ORDER BY id LIMIT 1", context: db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 20}},
		{name: "limit before a trailing comment", script: "SELECT id FROM memory.default.cutover_parity_t ORDER BY id -- note", context: db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 20}},
		{name: "explain text", script: "SELECT id FROM memory.default.cutover_parity_t", context: db.QueryContext{Explain: true, Option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_TEXT}, MaximumSQLResultSize: 1 << 20}},
		{name: "explain json", script: "SELECT id FROM memory.default.cutover_parity_t", context: db.QueryContext{Explain: true, Option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON}, MaximumSQLResultSize: 1 << 20}},
		{name: "multi statement", script: "SELECT 1 AS a;\nSELECT 2 AS b", context: qc()},
		{name: "error stops the script", script: "SELECT 1 AS a;\nSELECT * FROM nope.nope;\nSELECT 2 AS b", context: qc()},
		{name: "schema", script: "SELECT id FROM cutover_parity_t ORDER BY id", context: db.QueryContext{Schema: "default", MaximumSQLResultSize: 1 << 20}},
		{name: "explain of an explain analyze", script: "EXPLAIN ANALYZE SELECT 1", context: db.QueryContext{Explain: true, Option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON}, MaximumSQLResultSize: 1 << 20}},
		{name: "truncation", script: "SELECT * FROM (VALUES 'aaaaaaaaaa', 'bbbbbbbbbb', 'cccccccccc') AS t(x)", context: db.QueryContext{MaximumSQLResultSize: 40}},
	}
	require.Len(t, queries, len(golden.Queries), "the corpus and its recording must have the same cases")

	for i, query := range queries {
		t.Run(query.name, func(t *testing.T) {
			want := golden.Queries[i]
			require.Equal(t, want.Name, query.name)
			require.Equal(t, want.Script, query.script)

			recorder.reset()
			results, callErr := opened.QueryConn(ctx, query.script, query.context)
			requests := recorder.take()
			got := parityQuery{Name: query.name, Script: query.script, Requests: requests}
			if callErr != nil {
				got.CallError = callErr.Error()
			}
			for _, result := range results {
				got.Results = append(got.Results, summarizeParityResult(result))
			}

			if reason, known := knownDifferences[got.Name]; known {
				t.Logf("known difference: %s", reason)
				assertKnownQueryDifference(t, got)
				return
			}
			if diff := cmp.Diff(want.Requests, got.Requests); diff != "" {
				t.Errorf("requests (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(want.Results, got.Results); diff != "" {
				t.Errorf("results (-want +got):\n%s", diff)
			}
			require.Equal(t, want.CallError, got.CallError)
		})
	}
}
