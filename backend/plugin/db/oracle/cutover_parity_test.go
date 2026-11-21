package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"

	// The recording run dumps through the schema registry, which the pre-cutover
	// driver used; the replay reads the recorded text. Keeping the import means
	// one file both records and replays.
	_ "github.com/bytebase/bytebase/backend/plugin/schema/oracle"
)

// testdata/cutover_parity.json holds what the pre-cutover Oracle driver
// returned for the corpus below. It was recorded through this file with
// CUTOVER_PARITY_RECORD=1 against the driver this package held before the
// cutover, and this test is the cutover's differential check even though only
// one driver is left to run: the corpus, the two paths' shared db.Driver seam
// and the recorded expectations are what compare.
//
// A case whose statement text or result omni deliberately changes is named in
// knownDifferences and asserted on its own, so the allowlist cannot hide a
// later regression.

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
	Limit     int                 `json:"limit"`
	Explain   bool                `json:"explain"`
	Format    string              `json:"format"`
	MaxBytes  int64               `json:"max_bytes"`
	Results   []parityQueryResult `json:"results"`
	CallError string              `json:"call_error"`
}

type parityExecute struct {
	Name           string   `json:"name"`
	Script         string   `json:"script"`
	CreateDatabase bool     `json:"create_database"`
	Affected       int64    `json:"affected"`
	Error          string   `json:"error"`
	Logs           []string `json:"logs"`
}

type parityRecording struct {
	Queries []parityQuery   `json:"queries"`
	Execute []parityExecute `json:"execute"`
}

// trimSQLPlusStatements removes the blanked SQL*Plus command from a result's
// statement text.
func trimSQLPlusStatements(results []parityQueryResult) {
	for i := range results {
		results[i].Statement = strings.TrimLeft(results[i].Statement, " \t\r\n")
	}
}

// trimLoggedStatementSpace removes the single leading space the adapter's
// SQL*Plus blanking leaves on the statement a command precedes.
func trimLoggedStatementSpace(logs []string) []string {
	out := make([]string, len(logs))
	for i, log := range logs {
		out[i] = strings.Replace(log, `statement:" `, `statement:"`, 1)
	}
	return out
}

// parityDump is the dump the pre-cutover driver wrote for one synced schema.
type parityDump struct {
	Text string `json:"text"`
}

// parityIndices rewrites the EXPLAIN PLAN statement ID, which the old driver
// picked at random and the omni driver numbers per driver. It is the only part
// of a statement or a plan query that is not determined by the corpus.
var parityIndices = regexp.MustCompile(`(STATEMENT_ID = |statement_id = )'[^']*'`)

func normalizeParityStatement(statement string) string {
	return parityIndices.ReplaceAllString(statement, "$1'<id>'")
}

// normalizeProtoText collapses the extra spaces protobuf's text format inserts
// at random between fields, which makes a message's String() unsuitable for an
// exact comparison. A value's own text keeps its single spaces.
func normalizeProtoText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// normalizeLogs collapses the same spaces in a task run's log entries.
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
		Statement:   normalizeParityStatement(result.Statement),
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
	format   v1pb.QueryOption_ExplainFormat
	maxBytes int64
}

func (o parityQueryOptions) queryContext() db.QueryContext {
	queryContext := db.QueryContext{
		Schema:               o.schema,
		Limit:                o.limit,
		Explain:              o.explain,
		MaximumSQLResultSize: o.maxBytes,
	}
	if o.format != v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED {
		queryContext.Option = &v1pb.QueryOption{ExplainFormat: o.format}
	}
	return queryContext
}

// parityQueries is the corpus. Every case runs on its own driver, which is how
// the API opens one: an editor request gets one driver and one QueryConn call.
var parityQueries = []struct {
	name string
	opts parityQueryOptions
	// calls holds the scripts run in order on that one driver. The session case
	// runs two, in the way the admin execute stream reuses a driver.
	calls []string
}{
	{name: "select", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"SELECT ID, NAME, AMOUNT, CREATED FROM PARITY_T ORDER BY ID"}},
	{name: "types", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{`SELECT 42 AS C_INT, 'hi' AS C_VARCHAR, 1.25 AS C_NUMBER, TO_DATE('2026-01-02 03:04:05', 'YYYY-MM-DD HH24:MI:SS') AS C_DATE, TIMESTAMP '2026-01-02 03:04:05.123' AS C_TS, TIMESTAMP '2026-01-02 03:04:05.123 +02:00' AS C_TSTZ, CAST(TIMESTAMP '2026-01-02 03:04:05.123 +02:00' AS TIMESTAMP WITH LOCAL TIME ZONE) AS C_TSLTZ, CAST(NULL AS VARCHAR2(10)) AS C_NULL FROM DUAL`}},
	{name: "multi statement", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"SELECT 1 AS A FROM DUAL;\nSELECT 2 AS B FROM DUAL"}},
	// A SQL*Plus command is not SQL: Bytebase's splitter skipped it, and the
	// adapter has to, because the driver sends every segment it splits.
	{name: "sqlplus command", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"SET DEFINE OFF\nSELECT 1 AS A FROM DUAL"}},
	{name: "ddl through the query path", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"CREATE TABLE PARITY_DDL (ID NUMBER(10))"}},
	{name: "dml through the query path", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"INSERT INTO PARITY_T (ID, NAME) VALUES (4, 'delta')"}},
	{name: "error stops the script", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"SELECT 1 AS A FROM DUAL;\nSELECT * FROM NOPE;\nSELECT 2 AS B FROM DUAL"}},
	{name: "limit", opts: parityQueryOptions{limit: 2, maxBytes: 1 << 20}, calls: []string{"SELECT ID FROM PARITY_T ORDER BY ID"}},
	{name: "limit kept", opts: parityQueryOptions{limit: 2, maxBytes: 1 << 20}, calls: []string{"SELECT ID FROM PARITY_T ORDER BY ID FETCH FIRST 1 ROWS ONLY"}},
	{name: "explain text", opts: parityQueryOptions{explain: true, format: v1pb.QueryOption_TEXT, maxBytes: 1 << 20}, calls: []string{"SELECT 1 AS N FROM DUAL"}},
	{name: "explain json", opts: parityQueryOptions{explain: true, format: v1pb.QueryOption_JSON, maxBytes: 1 << 20}, calls: []string{"SELECT 1 AS N FROM DUAL"}},
	{name: "truncation", opts: parityQueryOptions{maxBytes: 120}, calls: []string{"SELECT LEVEL AS N, 'abcdefghij' AS PAD FROM DUAL CONNECT BY LEVEL <= 10"}},
	{name: "session setting persists", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"ALTER SESSION SET NLS_DATE_FORMAT = 'YYYY/MM/DD'", "SELECT TO_CHAR(DATE '2026-01-02') AS D FROM DUAL"}},
	{name: "schema", opts: parityQueryOptions{schema: paritySchemaB, maxBytes: 1 << 20}, calls: []string{"SELECT NAME FROM PARITY_SCHEMA_T"}},
	{name: "select declaring plsql", opts: parityQueryOptions{maxBytes: 1 << 20}, calls: []string{"WITH FUNCTION f RETURN NUMBER IS BEGIN RETURN 1; END; SELECT f() AS V FROM DUAL"}},
}

// parityExecutes is the Execute corpus. Each case runs on its own driver, so a
// case that writes cannot be undone by a later case's transaction.
var parityExecutes = []struct {
	name           string
	createDatabase bool
	script         string
}{
	{name: "ddl", script: "CREATE TABLE PARITY_EXEC_DDL (ID NUMBER(10))"},
	{name: "dml", script: "INSERT INTO PARITY_T (ID, NAME) VALUES (5, 'epsilon');\nINSERT INTO PARITY_T (ID, NAME) VALUES (6, 'zeta')"},
	{name: "transaction", script: "INSERT INTO PARITY_T (ID, NAME) VALUES (7, 'eta')"},
	{name: "transaction off", script: "-- txn-mode = off\nINSERT INTO PARITY_T (ID, NAME) VALUES (8, 'theta');\nINSERT INTO PARITY_T (ID, NAME) VALUES (9, 'iota')"},
	{name: "error rolls back", script: "INSERT INTO PARITY_T (ID, NAME) VALUES (10, 'kappa');\nINSERT INTO NOPE (ID) VALUES (1)"},
	{name: "empty script", script: "-- only a comment\n"},
	// The old driver split with parser/plsql, whose SQL*Plus handling and
	// trailing-fragment rule have their own tests there; these two cases pin what
	// Execute sends for them.
	{name: "sqlplus command", script: "SET DEFINE OFF\nCREATE TABLE PARITY_EXEC_SPLUS (ID NUMBER(10))"},
	{name: "trailing fragment", script: "SELECT 1 FROM DUAL;\nCASCADE"},
	{name: "create database refused", createDatabase: true, script: "CREATE TABLE PARITY_NEVER (ID NUMBER(10))"},
}

const (
	// paritySchemaA is the schema the driver reads. The corpus uses fixed names,
	// not the container's per-test schema, because a statement the old driver
	// rejected names the schema it looked in and the recording has to hold across
	// runs.
	paritySchemaA = "PARITY_DB"
	paritySchemaB = "PARITY_OTHER"
)

// parityEnv is the container the corpus runs on: one schema the driver reads,
// one the schema case switches to, and a table in each.
type parityEnv struct {
	container *testcontainer.Container
	schemaA   string
	// version is the server's release, which the driver is opened with: the
	// product passes the instance's stored version, and the limit rewrite and the
	// pre-12.2 sync path read it.
	version string
}

func newParityEnv(t *testing.T) *parityEnv {
	t.Helper()
	ctx := t.Context()
	container := testcontainer.SharedOracleContainer(t)
	env := &parityEnv{container: container, schemaA: paritySchemaA}
	for _, statement := range []string{
		fmt.Sprintf("DROP USER %s CASCADE", paritySchemaB),
		fmt.Sprintf("DROP USER %s CASCADE", paritySchemaA),
		fmt.Sprintf("CREATE USER %s NO AUTHENTICATION QUOTA UNLIMITED ON USERS", paritySchemaA),
		fmt.Sprintf("CREATE USER %s NO AUTHENTICATION QUOTA UNLIMITED ON USERS", paritySchemaB),
		fmt.Sprintf("GRANT CREATE TABLE TO %s", paritySchemaA),
		fmt.Sprintf("GRANT CREATE TABLE TO %s", paritySchemaB),
		fmt.Sprintf("CREATE TABLE %s.PARITY_SCHEMA_T (NAME VARCHAR2(50))", paritySchemaB),
		fmt.Sprintf("INSERT INTO %s.PARITY_SCHEMA_T (NAME) VALUES ('other-schema')", paritySchemaB),
		fmt.Sprintf("CREATE TABLE %s.PARITY_T (ID NUMBER(10), NAME VARCHAR2(50), AMOUNT NUMBER(10,2), CREATED DATE)", paritySchemaA),
		fmt.Sprintf("INSERT INTO %s.PARITY_T (ID, NAME, AMOUNT, CREATED) VALUES (1, 'alpha', 12.34, DATE '2026-01-02')", paritySchemaA),
		fmt.Sprintf("INSERT INTO %s.PARITY_T (ID, NAME, AMOUNT, CREATED) VALUES (2, 'beta', NULL, NULL)", paritySchemaA),
		fmt.Sprintf("INSERT INTO %s.PARITY_T (ID, NAME, AMOUNT, CREATED) VALUES (3, 'gamma', 0.5, DATE '2026-01-03')", paritySchemaA),
	} {
		_, err := container.GetDB().ExecContext(ctx, statement)
		// The drops before the creates fail on the first run; everything else must
		// succeed.
		if strings.HasPrefix(statement, "DROP USER") {
			continue
		}
		require.NoError(t, err, statement)
	}
	opened := openParityOracle(ctx, t, container, paritySchemaA, "")
	instance, err := opened.SyncInstance(ctx)
	require.NoError(t, err)
	require.NoError(t, opened.Close(ctx))
	env.version = instance.Version
	return env
}

// openParityOracle opens a driver on schemaName the way the API does: as the
// admin account, with CURRENT_SCHEMA switched to the database and the instance's
// stored version.
func openParityOracle(ctx context.Context, t *testing.T, container *testcontainer.Container, schemaName, engineVersion string) db.Driver {
	t.Helper()
	opened, err := newDriver().Open(ctx, storepb.Engine_ORACLE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:        container.GetHost(),
			Port:        container.GetPort(),
			Username:    container.GetUsername(),
			ServiceName: testcontainer.OracleServiceName,
		},
		Password:          container.GetPassword(),
		ConnectionContext: db.ConnectionContext{DatabaseName: schemaName, EngineVersion: engineVersion},
	})
	require.NoError(t, err)
	return opened
}

// openParityDriver opens one driver on schemaA the way the API does: as the
// admin account, with CURRENT_SCHEMA switched to the database.
func (e *parityEnv) openParityDriver(ctx context.Context, t *testing.T) *parityDriver {
	t.Helper()
	opened := openParityOracle(ctx, t, e.container, e.schemaA, e.version)
	driver := &parityDriver{driver: opened}
	t.Cleanup(func() { _ = opened.Close(context.Background()) })
	return driver
}

type parityDriver struct {
	driver db.Driver
}

func (p *parityDriver) queryConn(ctx context.Context, t *testing.T, script string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	t.Helper()
	results, err := p.driver.QueryConn(ctx, nil, script, queryContext)
	if err != nil || !queryContext.Explain {
		return results, err
	}
	// Fill the plan format the way the API does, so this compares what a user
	// reads rather than which side named the format: the old driver left it to
	// backend/api/v1, and the adapter states it.
	for _, result := range results {
		if result.Error == "" && result.QueryPlan == nil {
			if format, ok := db.ExplainResultFormat(storepb.Engine_ORACLE, queryContext.Option.GetExplainFormat()); ok {
				result.QueryPlan = &v1pb.QueryResult_QueryPlan{Format: format}
			}
		}
	}
	return results, nil
}

func (p *parityDriver) execute(ctx context.Context, script string, opts db.ExecuteOptions) (int64, error) {
	return p.driver.Execute(ctx, script, opts)
}

// assertKnownQueryDifference pins what the new path does for a case whose result
// moves with omni, so the allowlist cannot hide a later regression.
func assertKnownQueryDifference(t *testing.T, got parityQuery) {
	t.Helper()
	switch got.Name {
	case "select declaring plsql":
		// The old driver ran a SELECT that declares PL/SQL through Exec, so its
		// rows were dropped; the contract decides by the statement's shape and
		// returns them.
		require.Empty(t, got.CallError)
		require.Len(t, got.Results, 1)
		require.Empty(t, got.Results[0].Error)
		require.Len(t, got.Results[0].Rows, 1)
	case "schema":
		// The old driver ignored QueryContext.Schema; the contract switches the
		// session, so the named schema's table is the one read.
		require.Empty(t, got.CallError)
		require.Len(t, got.Results, 1)
		require.Empty(t, got.Results[0].Error)
		require.Len(t, got.Results[0].Rows, 1)
		require.Contains(t, got.Results[0].Rows[0][0], "other-schema")
	case "truncation":
		// The contract counts bytes, so the cut point differs from proto.Size at
		// power-of-two row counts; the message is the same.
		require.Len(t, got.Results, 1)
		require.NotEmpty(t, got.Results[0].Error)
	default:
		t.Fatalf("no assertion for known difference %q", got.Name)
	}
}

// knownDifferences names the corpus cases whose statement or result moves with
// the omni driver, and says what the new path does instead. Each entry is
// asserted above, so a change to one of these expectations still fails.
var knownDifferences = map[string]string{
	"select declaring plsql": "the contract runs a statement by its shape, so a SELECT that declares PL/SQL returns its rows; the old driver ran it through Exec",
	"schema":                 "the contract applies QueryContext.Schema to the session; the old driver ignored it",
	"truncation":             "the contract counts bytes per row, so the cut point differs from proto.Size at power-of-two row counts",
}

func TestCutoverParityAgainstRecordedOldPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	t.Cleanup(cancel)
	env := newParityEnv(t)

	record := os.Getenv("CUTOVER_PARITY_RECORD") == "1"
	goldenPath := filepath.Join("testdata", "cutover_parity.json")
	var golden parityRecording
	if record {
		golden = parityRecording{Queries: []parityQuery{}, Execute: []parityExecute{}}
	} else {
		goldenBytes, err := os.ReadFile(goldenPath)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(goldenBytes, &golden))
		normalizeRecording(&golden)
		require.Len(t, golden.Queries, len(parityQueries), "the corpus and its recording must have the same cases")
		require.Len(t, golden.Execute, len(parityExecutes), "the corpus and its recording must have the same cases")
	}

	for _, entry := range parityQueries {
		t.Run("query/"+entry.name, func(t *testing.T) {
			client := env.openParityDriver(ctx, t)
			got := parityQuery{Name: entry.name, Script: strings.Join(entry.calls, "\n-- next --\n"), Schema: entry.opts.schema, Limit: entry.opts.limit, Explain: entry.opts.explain, Format: entry.opts.format.String(), MaxBytes: entry.opts.maxBytes}
			for _, script := range entry.calls {
				results, err := client.queryConn(ctx, t, script, entry.opts.queryContext())
				if err != nil {
					got.CallError = err.Error()
					break
				}
				for _, result := range results {
					got.Results = append(got.Results, summarizeParityResult(result))
				}
			}

			if record {
				got.CallError = normalizeParityStatement(got.CallError)
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
				assertKnownQueryDifference(t, got)
				return
			}
			require.Equal(t, want.CallError, got.CallError)
			if got.Name == "sqlplus command" {
				// See the Execute case: the blanked command is leading whitespace on
				// the statement that follows it, and nothing else about the result
				// moves.
				t.Log("known difference: a skipped SQL*Plus command leaves leading whitespace")
				trimSQLPlusStatements(want.Results)
				trimSQLPlusStatements(got.Results)
			}
			if diff := cmp.Diff(want.Results, got.Results); diff != "" {
				t.Errorf("results (-want +got):\n%s", diff)
			}
		})
	}

	for _, entry := range parityExecutes {
		t.Run("execute/"+entry.name, func(t *testing.T) {
			client := env.openParityDriver(ctx, t)
			var logs []string
			affected, err := client.execute(ctx, entry.script, db.ExecuteOptions{
				CreateDatabase:      entry.createDatabase,
				LogCommandStatement: true,
				CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
					logs = append(logs, log.String())
					return nil
				},
			})
			got := parityExecute{
				Name:           entry.name,
				Script:         entry.script,
				CreateDatabase: entry.createDatabase,
				Affected:       affected,
				Logs:           normalizeLogs(logs),
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
			require.Equal(t, want.Affected, got.Affected, "affected rows")
			require.Equal(t, want.Error, got.Error, "error")
			if got.Name == "sqlplus command" {
				// The adapter blanks a SQL*Plus command instead of removing it, so
				// every other statement keeps its byte offsets and the statement
				// after it starts at the command's line terminator, now a space.
				// Oracle runs the same statement either way.
				t.Log("known difference: a skipped SQL*Plus command leaves one leading space")
				require.Equal(t, trimLoggedStatementSpace(want.Logs), trimLoggedStatementSpace(got.Logs), "task-run log")
				return
			}
			require.Equal(t, want.Logs, got.Logs, "task-run log")
		})
	}

	if record {
		payload, err := json.MarshalIndent(golden, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(goldenPath, append(payload, '\n'), 0o644))
	}
}

// parityDumpStatements is the schema the dump corpus syncs and renders.
var parityDumpStatements = []string{
	`CREATE TABLE %[1]s.ORDERS (ID NUMBER(10) NOT NULL, ORDER_NO VARCHAR2(64) NOT NULL, TOTAL NUMBER(10,2) DEFAULT 0, CREATED DATE DEFAULT SYSDATE, CONSTRAINT ORDERS_PK PRIMARY KEY (ID), CONSTRAINT ORDERS_NO_UQ UNIQUE (ORDER_NO), CONSTRAINT ORDERS_TOTAL_CK CHECK (TOTAL >= 0))`,
	`CREATE INDEX %[1]s.ORDERS_CREATED_IDX ON %[1]s.ORDERS (CREATED)`,
	`CREATE SEQUENCE %[1]s.ORDERS_SEQ START WITH 5 INCREMENT BY 2 MAXVALUE 9999 CACHE 20 CYCLE`,
	`CREATE VIEW %[1]s.ORDERS_VIEW AS SELECT ID, ORDER_NO FROM %[1]s.ORDERS`,
	`COMMENT ON TABLE %[1]s.ORDERS IS 'orders'`,
	`COMMENT ON COLUMN %[1]s.ORDERS.TOTAL IS 'total'`,
}

// TestDumpAgainstRecordedOldPath compares the dump the adapter writes through
// omni's catalog against the text the pre-cutover driver wrote through
// backend/plugin/schema/oracle over the same schema. The old text was recorded
// through this file with CUTOVER_PARITY_RECORD=1 against the driver this package
// held before the cutover, so it is a differential check on the user-visible
// dump even though only one driver is left to run.
//
// The differences are the ones the cutover notes name: omni's catalog reads the
// whole sequence definition and renders comments, where the old renderer wrote a
// CREATE SEQUENCE without MINVALUE/CACHE and no COMMENT statement at all.
// Quoted identifiers, and a synced key column in particular, render the same in
// both.
func TestDumpAgainstRecordedOldPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	t.Cleanup(cancel)
	container, schemaName := testcontainer.NewOracleSchema(t)
	for _, statement := range parityDumpStatements {
		_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf(statement, schemaName))
		require.NoError(t, err, statement)
	}

	probe := openParityOracle(ctx, t, container, schemaName, "")
	instance, err := probe.SyncInstance(ctx)
	require.NoError(t, err)
	require.NoError(t, probe.Close(ctx))

	opened := openParityOracle(ctx, t, container, schemaName, instance.Version)
	defer opened.Close(ctx)
	metadata, err := opened.SyncDBSchema(ctx)
	require.NoError(t, err)
	var text strings.Builder
	require.NoError(t, opened.Dump(ctx, &text, metadata))

	goldenPath := filepath.Join("testdata", "cutover_dump.json")
	if os.Getenv("CUTOVER_PARITY_RECORD") == "1" {
		payload, err := json.MarshalIndent(parityDump{Text: text.String()}, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(goldenPath, append(payload, '\n'), 0o644))
		return
	}

	goldenBytes, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	var golden parityDump
	require.NoError(t, json.Unmarshal(goldenBytes, &golden))

	// A known difference is removed from both sides, so the rest of the text has
	// to match byte for byte.
	for _, fragment := range []string{"INCREMENT BY 2", "MAXVALUE 9999", "MINVALUE 1", "CACHE 20", "CYCLE"} {
		require.Contains(t, text.String(), fragment, "omni renders the sequence's full definition")
	}
	require.Contains(t, text.String(), `COMMENT ON TABLE "ORDERS" IS 'orders';`, "omni renders object comments")
	require.Contains(t, text.String(), `PRIMARY KEY ("ID")`, "a synced key column renders quoted once")
	if diff := cmp.Diff(normalizeDump(golden.Text), normalizeDump(text.String())); diff != "" {
		t.Errorf("dump (-old +new, known differences removed):\n%s", diff)
	}
}

// normalizeDump removes what the cutover changes in a dump, so the rest of the
// text compares byte for byte: the sequence's definition, which the old sync did
// not read, and the COMMENT statements the old renderer did not write.
func normalizeDump(text string) []string {
	var kept []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "COMMENT ON ") {
			continue
		}
		line = dumpSequenceName.ReplaceAllString(line, `CREATE SEQUENCE $1;`)
		kept = append(kept, line)
	}
	return kept
}

// dumpSequenceName matches a CREATE SEQUENCE statement with its definition.
var dumpSequenceName = regexp.MustCompile(`^CREATE SEQUENCE ("(?:[^"]|"")*").*$`)
