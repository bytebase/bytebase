package redshift

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// testdata/cutover_parity.json holds what the pre-cutover Redshift driver
// returned for the corpus below. It was recorded through this file with
// CUTOVER_PARITY_RECORD=1 against the driver this package held before the
// cutover, so this test is the cutover's differential check even though only one
// driver is left to run: the corpus, the two paths' shared db.Driver seam and the
// recorded expectations are what compare.
//
// The recorder is a PostgreSQL server, because a Redshift cluster is not
// available to this repository's tests and none of its catalog exists on
// PostgreSQL. That is enough for the three paths recorded here — a statement's
// text after the limit, explain and search-path rewrites, its decoded rows, its
// error and its task-run log — because both drivers send those statements to
// whatever server they are opened on, and the rewrites come from the same
// redshift analysis on both sides. It proves nothing about the sync or the dump:
// their queries read Redshift's own catalog (SVV_ALL_SCHEMAS, SVV_ALL_TABLES,
// SHOW VIEW) and there is no stand-in for it. Those paths' recorded expectations
// live in omni instead, taped from a real cluster: redshift/driver/testdata/tapes
// holds the six recorded conversations and testdata/sync/dev.json the metadata
// the sync built, pinned by redshift/testdata/pins.tsv at Redshift 1.0.539507.
//
// A case whose statement text or result omni deliberately changes is named in
// knownDifferences and asserted on its own, so the allowlist cannot hide a later
// regression.

type parityQueryResult struct {
	Statement   string     `json:"statement"`
	Columns     []string   `json:"columns"`
	ColumnTypes []string   `json:"column_types"`
	Rows        [][]string `json:"rows"`
	Error       string     `json:"error"`
	RowsCount   int64      `json:"rows_count"`
	Plan        string     `json:"plan"`
}

type parityQuery struct {
	Name      string              `json:"name"`
	Script    string              `json:"script"`
	Schema    string              `json:"schema"`
	DataShare bool                `json:"datashare"`
	Limit     int                 `json:"limit"`
	Explain   bool                `json:"explain"`
	MaxBytes  int64               `json:"max_bytes"`
	Masking   bool                `json:"masking"`
	Results   []parityQueryResult `json:"results"`
	CallError string              `json:"call_error"`
}

type parityExecute struct {
	Name      string   `json:"name"`
	Script    string   `json:"script"`
	DataShare bool     `json:"datashare"`
	Affected  int64    `json:"affected"`
	Error     string   `json:"error"`
	Logs      []string `json:"logs"`
}

type parityCount struct {
	Name      string `json:"name"`
	Statement string `json:"statement"`
	Rows      int64  `json:"rows"`
	Error     string `json:"error"`
}

type parityRecording struct {
	Queries []parityQuery   `json:"queries"`
	Execute []parityExecute `json:"execute"`
	Count   []parityCount   `json:"count"`
}

// normalizeProtoText collapses the extra spaces protobuf's text format inserts
// at random between fields, which makes a message's String() unsuitable for an
// exact comparison. A value's own text keeps its single spaces.
func normalizeProtoText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func normalizeLogs(logs []string) []string {
	out := make([]string, len(logs))
	for i, log := range logs {
		out[i] = normalizeProtoText(log)
	}
	return out
}

// normalizeRecording collapses protobuf's spacing in a recording, so a golden
// recorded before this normalization compares like the run that reads it.
func normalizeRecording(recording *parityRecording) {
	for i := range recording.Queries {
		for j := range recording.Queries[i].Results {
			result := &recording.Queries[i].Results[j]
			for k := range result.Rows {
				for l := range result.Rows[k] {
					result.Rows[k][l] = normalizeProtoText(result.Rows[k][l])
				}
			}
			result.Plan = normalizeProtoText(result.Plan)
		}
	}
	for i := range recording.Execute {
		recording.Execute[i].Logs = normalizeLogs(recording.Execute[i].Logs)
	}
}

func summarizeParityResult(result *v1pb.QueryResult) parityQueryResult {
	rows := make([][]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		values := make([]string, 0, len(row.Values))
		for _, value := range row.Values {
			values = append(values, normalizeProtoText(value.String()))
		}
		rows = append(rows, values)
	}
	plan := ""
	if result.QueryPlan != nil {
		plan = normalizeProtoText(result.QueryPlan.String())
	}
	return parityQueryResult{
		Statement:   result.Statement,
		Columns:     result.ColumnNames,
		ColumnTypes: result.ColumnTypeNames,
		Rows:        rows,
		Error:       result.Error,
		RowsCount:   result.RowsCount,
		Plan:        plan,
	}
}

// parityQueryOptions describes one corpus case in the product's QueryContext
// terms, so the same table drives the recording and the replay.
type parityQueryOptions struct {
	schema   string
	limit    int
	explain  bool
	maxBytes int64
	masking  bool
}

func (o parityQueryOptions) queryContext() db.QueryContext {
	return db.QueryContext{
		Schema:               o.schema,
		Limit:                o.limit,
		Explain:              o.explain,
		MaximumSQLResultSize: o.maxBytes,
		MaskingEnabled:       o.masking,
	}
}

// parityQueries is the corpus. Every case runs on its own driver, which is how
// the API opens one: an editor request gets one driver and one QueryConn call.
var parityQueries = []struct {
	name      string
	datashare bool
	opts      parityQueryOptions
	// calls holds the scripts run in order on that one driver. The session case
	// runs two, in the way the admin execute stream reuses a driver.
	calls []string
}{
	{name: "select", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"SELECT id, name FROM parity.parity_t ORDER BY id"}},
	{name: "types", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{`SELECT 42 AS c_int, 'hi'::text AS c_text, true AS c_bool, 1.25::float8 AS c_float, DATE '2026-01-02' AS c_date, TIMESTAMP '2026-01-02 03:04:05.123' AS c_ts, TIMESTAMPTZ '2026-01-02 03:04:05.123+02' AS c_tstz, NULL::varchar AS c_null, B'1010' AS c_bit, TIME '01:02:03.5' AS c_time`}},
	{name: "multi statement", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"SELECT 1 AS a;\nSELECT 2 AS b"}},
	{name: "schema", opts: parityQueryOptions{schema: "parity", maxBytes: 1 << 20}, calls: []string{"SELECT name FROM parity_t ORDER BY id"}},
	{name: "limit", opts: parityQueryOptions{limit: 2, maxBytes: 1 << 20}, calls: []string{"SELECT id FROM parity.parity_t ORDER BY id"}},
	{name: "limit kept", opts: parityQueryOptions{limit: 2, maxBytes: 1 << 20}, calls: []string{"SELECT id FROM parity.parity_t ORDER BY id LIMIT 1"}},
	{name: "explain", opts: parityQueryOptions{explain: true, maxBytes: 1 << 20}, calls: []string{"SELECT id FROM parity.parity_t WHERE id > 1"}},
	{name: "ddl through the query path", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"CREATE TABLE parity.parity_ddl (id int)"}},
	{name: "dml through the query path", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"INSERT INTO parity.parity_t (id, name) VALUES (4, 'delta')"}},
	{name: "error stops the script", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"SELECT 1 AS a;\nSELECT * FROM parity.parity_missing;\nSELECT 2 AS b"}},
	{name: "truncation", opts: parityQueryOptions{maxBytes: 120}, calls: []string{"SELECT g AS n, 'abcdefghij' AS pad FROM generate_series(1, 10) AS g ORDER BY g"}},
	{name: "session setting persists", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"SET search_path TO parity", "SELECT name FROM parity_t ORDER BY id"}},
	{name: "masking leaves a traced select alone", opts: parityQueryOptions{maxBytes: 1 << 20, masking: true}, calls: []string{"SELECT id FROM parity.parity_t ORDER BY id"}},
	{name: "empty script", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"-- only a comment\n"}},
	// A datashare database is addressed without its own name: the driver strips
	// it from the relations a statement names. The literal is there to say which
	// occurrences the old driver stripped.
	{name: "datashare qualifier", datashare: true, opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{
		"SELECT 'keep {database}.parity' AS label",
		"SELECT id FROM {database}.parity.parity_t ORDER BY id",
	}},
}

// databasePlaceholder is the corpus database's name in a case that must address
// it, such as the datashare qualifier. The container names each test's database
// differently, so a recording and a replay agree only on the stand-in.
const databasePlaceholder = "{database}"

// normalizeQueries replaces the corpus database's name with the placeholder in
// what a case sent and returned, so the recording reads the same on another run.
func (e *parityEnv) normalizeQuery(query *parityQuery) {
	replace := func(text string) string {
		return strings.ReplaceAll(text, e.database, databasePlaceholder)
	}
	query.Script = replace(query.Script)
	query.CallError = replace(query.CallError)
	for i := range query.Results {
		result := &query.Results[i]
		result.Statement = replace(result.Statement)
		result.Error = replace(result.Error)
		for j := range result.Rows {
			for k := range result.Rows[j] {
				result.Rows[j][k] = replace(result.Rows[j][k])
			}
		}
	}
}

// parityExecutes is the Execute corpus. Each case runs on its own driver, so a
// case that writes cannot be undone by a later case's transaction.
var parityExecutes = []struct {
	name      string
	datashare bool
	script    string
}{
	{name: "ddl", script: "CREATE TABLE parity.parity_exec_ddl (id int)"},
	{name: "dml", script: "INSERT INTO parity.parity_t (id, name) VALUES (5, 'epsilon');\nINSERT INTO parity.parity_t (id, name) VALUES (6, 'zeta')"},
	{name: "transaction", script: "INSERT INTO parity.parity_t (id, name) VALUES (7, 'eta')"},
	{name: "transaction off", script: "-- txn-mode = off\nINSERT INTO parity.parity_t (id, name) VALUES (8, 'theta');\nINSERT INTO parity.parity_t (id, name) VALUES (9, 'iota')"},
	{name: "error rolls back", script: "INSERT INTO parity.parity_t (id, name) VALUES (10, 'kappa');\nINSERT INTO parity.parity_missing (id) VALUES (1)"},
	{name: "empty script", script: "-- only a comment\n"},
	{name: "truncate in a transactional script", script: "TRUNCATE parity.parity_t"},
	{name: "transaction control in a script", script: "BEGIN;\nINSERT INTO parity.parity_t (id, name) VALUES (11, 'lambda');\nCOMMIT"},
	{name: "datashare refused", datashare: true, script: "INSERT INTO parity.parity_t (id, name) VALUES (12, 'mu')"},
}

// parityCounts is the CountAffectedRows corpus, which plancheck's affected-row
// limit reads.
var parityCounts = []struct {
	name      string
	statement string
}{
	{name: "select estimate", statement: "SELECT * FROM parity.parity_t WHERE id > 1"},
	{name: "search path prefix", statement: "SET LOCAL search_path TO \"parity\";\nSELECT * FROM parity_t WHERE id > 1"},
}

// parityEnv is the server the corpus runs on: a schema the driver reads, one
// the schema case switches to, and a table in each.
type parityEnv struct {
	container *testcontainer.Container
	database  string
}

func newParityEnv(t *testing.T) *parityEnv {
	t.Helper()
	ctx := t.Context()
	container := testcontainer.SharedPgContainer(t)
	database, db := testcontainer.NewPgDatabase(t)
	for _, statement := range []string{
		"CREATE SCHEMA parity",
		"CREATE TABLE parity.parity_t (id int, name text)",
		"INSERT INTO parity.parity_t (id, name) VALUES (1, 'alpha'), (2, 'beta'), (3, 'gamma')",
		"CREATE TABLE public.parity_schema_t (label text)",
		"INSERT INTO public.parity_schema_t (label) VALUES ('other-schema')",
	} {
		_, err := db.ExecContext(ctx, statement)
		require.NoError(t, err, statement)
	}
	return &parityEnv{container: container, database: database}
}

// openParityDriver opens one driver on the corpus database the way the API does:
// as the superuser, on the database the caller named.
func (e *parityEnv) openParityDriver(ctx context.Context, t *testing.T, datashare bool) db.Driver {
	t.Helper()
	opened, err := newDriver().Open(ctx, storepb.Engine_REDSHIFT, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     e.container.GetHost(),
			Port:     e.container.GetPort(),
			Username: e.container.GetUsername(),
		},
		Password: e.container.GetPassword(),
		ConnectionContext: db.ConnectionContext{
			DatabaseName: e.database,
			DataShare:    datashare,
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = opened.Close(context.Background()) })
	return opened
}

type parityDriver struct {
	driver db.Driver
}

func (p *parityDriver) queryConn(ctx context.Context, t *testing.T, script string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	t.Helper()
	return p.driver.QueryConn(ctx, script, queryContext)
}

// assertKnownQueryDifference pins what the new path does for a case whose result
// moves with omni, so the allowlist cannot hide a later regression.
func assertKnownQueryDifference(t *testing.T, want, got parityQuery) {
	t.Helper()
	switch got.Name {
	case "explain":
		// The plan is the same text and the same format; the contract carries it as
		// a plan, and backend/api/v1 fills the same format for a driver that leaves
		// it nil.
		require.Len(t, got.Results, len(want.Results))
		for i := range got.Results {
			require.Equal(t, "format:TEXT", got.Results[i].Plan)
			got.Results[i].Plan = ""
		}
		req := cmp.Diff(want.Results, got.Results)
		require.Empty(t, req, "results without the plan field")
	case "datashare qualifier":
		// The contract strips the database's name from the relations a statement
		// names, not from every occurrence in its text: the literal keeps it, and
		// the relation's qualifier is stripped as before.
		require.Len(t, got.Results, 2)
		require.Equal(t, "SELECT 'keep {database}.parity' AS label", got.Results[0].Statement)
		require.Equal(t, [][]string{{`string_value:"keep {database}.parity"`}}, got.Results[0].Rows)
		require.Equal(t, want.Results[1], got.Results[1])
	case "truncation":
		// The message is the same; the cut row is measured differently, so the
		// rows that fit before it differ. The contract sums a row's bytes as it
		// reads them, where the old conversion measured the built QueryRow proto.
		require.Len(t, got.Results, 1)
		require.Equal(t, want.Results[0].Statement, got.Results[0].Statement)
		require.Equal(t, want.Results[0].Error, got.Results[0].Error)
		require.LessOrEqual(t, len(got.Results[0].Rows), len(want.Results[0].Rows))
		require.Equal(t, int64(len(got.Results[0].Rows)), got.Results[0].RowsCount)
	default:
		t.Fatalf("no assertion for known difference %q", got.Name)
	}
}

// knownDifferences names the corpus cases whose statement or result moves with
// the omni driver, and says what the new path does instead. Each entry is
// asserted in assertKnownQueryDifference, so a change to one of these
// expectations still fails.
var knownDifferences = map[string]string{
	"datashare qualifier": "the contract strips the database's name from the relations a datashare statement names; the old driver replaced every occurrence, including one in a literal",
	"explain":             "the contract marks a rewritten EXPLAIN's rows as a TEXT plan; backend/api/v1 fills the same format when a driver leaves it nil",
	"truncation":          "the contract sums a row's bytes as it reads them, so it cuts a different row than the built QueryRow proto's size did; the message is the same",
}

func TestCutoverParityAgainstRecordedOldPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	t.Cleanup(cancel)
	env := newParityEnv(t)

	record := os.Getenv("CUTOVER_PARITY_RECORD") == "1"
	goldenPath := filepath.Join("testdata", "cutover_parity.json")
	var golden parityRecording
	if record {
		golden = parityRecording{Queries: []parityQuery{}, Execute: []parityExecute{}, Count: []parityCount{}}
	} else {
		goldenBytes, err := os.ReadFile(goldenPath)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(goldenBytes, &golden))
		normalizeRecording(&golden)
		require.Len(t, golden.Queries, len(parityQueries), "the corpus and its recording must have the same cases")
		require.Len(t, golden.Execute, len(parityExecutes), "the corpus and its recording must have the same cases")
		require.Len(t, golden.Count, len(parityCounts), "the corpus and its recording must have the same cases")
	}

	for _, entry := range parityQueries {
		entry := entry
		t.Run("query/"+entry.name, func(t *testing.T) {
			client := &parityDriver{driver: env.openParityDriver(ctx, t, entry.datashare)}
			calls := make([]string, len(entry.calls))
			for i, call := range entry.calls {
				calls[i] = strings.ReplaceAll(call, databasePlaceholder, env.database)
			}
			got := parityQuery{Name: entry.name, Script: strings.Join(calls, "\n-- next --\n"), Schema: entry.opts.schema, DataShare: entry.datashare, Limit: entry.opts.limit, Explain: entry.opts.explain, MaxBytes: entry.opts.maxBytes, Masking: entry.opts.masking}
			for _, script := range calls {
				results, err := client.queryConn(ctx, t, script, entry.opts.queryContext())
				if err != nil {
					got.CallError = err.Error()
					break
				}
				for _, result := range results {
					got.Results = append(got.Results, summarizeParityResult(result))
				}
			}

			env.normalizeQuery(&got)
			if record {
				golden.Queries = append(golden.Queries, got)
				return
			}

			var want parityQuery
			for i := range golden.Queries {
				if golden.Queries[i].Name == entry.name {
					want = golden.Queries[i]
					break
				}
			}
			if reason, known := knownDifferences[got.Name]; known {
				t.Logf("known difference: %s", reason)
				assertKnownQueryDifference(t, want, got)
				return
			}
			require.Equal(t, want.CallError, got.CallError)
			if diff := cmp.Diff(want.Results, got.Results); diff != "" {
				t.Errorf("results (-want +got):\n%s", diff)
			}
		})
	}

	for _, entry := range parityExecutes {
		t.Run("execute/"+entry.name, func(t *testing.T) {
			client := env.openParityDriver(ctx, t, entry.datashare)
			var logs []string
			affected, err := client.Execute(ctx, entry.script, db.ExecuteOptions{
				LogCommandStatement: true,
				CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
					logs = append(logs, log.String())
					return nil
				},
			})
			got := parityExecute{
				Name:      entry.name,
				Script:    entry.script,
				DataShare: entry.datashare,
				Affected:  affected,
				Logs:      normalizeLogs(logs),
			}
			if err != nil {
				got.Error = err.Error()
			}

			if record {
				golden.Execute = append(golden.Execute, got)
				return
			}

			var want parityExecute
			for i := range golden.Execute {
				if golden.Execute[i].Name == entry.name {
					want = golden.Execute[i]
					break
				}
			}
			if reason, known := knownExecuteDifferences[got.Name]; known {
				t.Logf("known difference: %s", reason)
				assertKnownExecuteDifference(t, got)
				return
			}
			require.Equal(t, want.Affected, got.Affected, "affected rows")
			require.Equal(t, want.Error, got.Error, "error")
			require.Equal(t, want.Logs, got.Logs, "task-run log")
		})
	}

	for _, entry := range parityCounts {
		t.Run("count/"+entry.name, func(t *testing.T) {
			client := env.openParityDriver(ctx, t, false)
			estimator, ok := client.(db.RowEstimator)
			require.True(t, ok, "the driver must estimate affected rows for plancheck")
			rows, err := estimator.CountAffectedRows(ctx, entry.statement)
			got := parityCount{Name: entry.name, Statement: entry.statement, Rows: rows}
			if err != nil {
				got.Error = err.Error()
			}

			if record {
				golden.Count = append(golden.Count, got)
				return
			}

			var want parityCount
			for i := range golden.Count {
				if golden.Count[i].Name == entry.name {
					want = golden.Count[i]
					break
				}
			}
			require.Equal(t, want.Rows, got.Rows, "rows estimate")
			require.Equal(t, want.Error, got.Error, "error")
		})
	}

	if record {
		payload, err := json.MarshalIndent(golden, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(goldenPath, append(payload, '\n'), 0o644))
	}
}

// knownExecuteDifferences names the Execute cases whose behaviour moves with
// omni's driver, and says what the new path does instead. Each entry is asserted
// in assertKnownExecuteDifference, so a change to one of these expectations
// still fails.
var knownExecuteDifferences = map[string]string{
	"truncate in a transactional script": "the contract refuses a statement Redshift commits on its own in a transactional script, and names autocommit mode",
	"transaction control in a script":    "the contract refuses a script that decides its own transaction, and names autocommit mode",
}

func assertKnownExecuteDifference(t *testing.T, got parityExecute) {
	t.Helper()
	switch got.Name {
	case "truncate in a transactional script":
		require.Zero(t, got.Affected)
		require.Contains(t, got.Error, "commits on its own")
		require.Contains(t, got.Error, "autocommit mode")
		require.Empty(t, got.Logs)
	case "transaction control in a script":
		require.Zero(t, got.Affected)
		require.Contains(t, got.Error, "transaction control")
		require.Contains(t, got.Error, "autocommit mode")
		require.Empty(t, got.Logs)
	default:
		t.Fatalf("no assertion for known difference %q", got.Name)
	}
}

// TestDumpWritesNothing pins the no-op dump: Redshift's omni driver has no
// Dumper, and Bytebase's Dump wrote nothing either, so the adapter keeps the
// no-op rather than inventing a renderer. VERIFY.md in omni's redshift/driver
// says what a dump would need.
func TestDumpWritesNothing(t *testing.T) {
	var out strings.Builder
	require.NoError(t, (&Driver{}).Dump(t.Context(), &out, nil))
	require.Empty(t, out.String())
}
