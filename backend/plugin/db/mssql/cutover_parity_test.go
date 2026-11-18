package mssql

import (
	"bytes"
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

// The golden file holds what the driver this replaced ran and returned for a
// corpus of Execute, QueryConn and Dump calls. It was recorded on the pinned
// server before that driver was deleted, so this test is the cutover's
// differential check even though only one driver is left to run it. A case whose
// behaviour omni deliberately changes is named in knownDifferences below and
// pinned by its own assertion instead.

type parityExecute struct {
	Name     string   `json:"name"`
	Script   string   `json:"script"`
	Affected int64    `json:"affected"`
	Error    string   `json:"error"`
	Logs     []string `json:"logs"`
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
	Name      string         `json:"name"`
	Script    string         `json:"script"`
	Limit     int            `json:"limit"`
	Explain   bool           `json:"explain"`
	Format    string         `json:"format"`
	Masking   bool           `json:"masking"`
	MaxBytes  int64          `json:"max_bytes"`
	CallError string         `json:"call_error"`
	Results   []parityResult `json:"results"`
}

type parityRecording struct {
	Execute []parityExecute `json:"execute"`
	Queries []parityQuery   `json:"queries"`
	Dump    string          `json:"dump"`
}

// parityExecuteCorpus is the change-script corpus. The setup statements are its
// first entries, so the recording proves they ran as before too.
func parityExecuteCorpus() []struct{ name, script string } {
	return []struct{ name, script string }{
		{name: "setup table", script: `CREATE TABLE dbo.cutover_t (id INT IDENTITY(1,1) PRIMARY KEY, code VARCHAR(20) NOT NULL, amount DECIMAL(18,2) NULL, at DATETIMEOFFSET NULL, flag BIT NOT NULL DEFAULT 0, bin VARBINARY(10) NULL, note NVARCHAR(50) NOT NULL DEFAULT 'it''s')`},
		{name: "setup referring table", script: `CREATE TABLE dbo.cutover_u (id INT PRIMARY KEY, s_id INT NOT NULL CONSTRAINT cutover_fk REFERENCES dbo.cutover_t(id), v INT NULL CONSTRAINT cutover_v_check CHECK (v >= 0))`},
		{name: "setup filtered index", script: `CREATE INDEX cutover_filtered ON dbo.cutover_u (v) INCLUDE (s_id) WHERE v IS NOT NULL`},
		{name: "setup view", script: `CREATE VIEW dbo.cutover_v AS SELECT id, code FROM dbo.cutover_t WHERE code <> ''`},
		{name: "setup view with a semicolon", script: "CREATE VIEW dbo.cutover_v2 AS SELECT id FROM dbo.cutover_t;"},
		{name: "setup table with a trigger", script: "CREATE TABLE dbo.cutover_w (id INT PRIMARY KEY, v INT NULL)"},
		{name: "setup instead-of trigger", script: "CREATE TRIGGER cutover_w_trg ON dbo.cutover_w INSTEAD OF INSERT AS BEGIN INSERT INTO dbo.cutover_w (id, v) SELECT id, v FROM inserted; END"},
		{name: "setup procedure", script: `CREATE PROCEDURE dbo.cutover_two_sets AS BEGIN UPDATE dbo.cutover_t SET flag = flag; SELECT id FROM dbo.cutover_t ORDER BY id; SELECT 'second' AS s; END`},
		{name: "setup sequence", script: `CREATE SEQUENCE dbo.cutover_seq AS DECIMAL(18,0) START WITH 10 INCREMENT BY 5 MINVALUE 1 MAXVALUE 1000 CACHE 20`},
		{name: "setup rows", script: `INSERT INTO dbo.cutover_t (code, amount, at, bin) VALUES ('a', 1.50, '2024-01-02T03:04:05.1234567+02:00', 0x0102), ('b', NULL, NULL, 0x), ('c', 2.25, NULL, NULL)`},
		{name: "multi statement batch", script: "INSERT INTO dbo.cutover_t (code) VALUES ('d');\nUPDATE dbo.cutover_t SET code = 'e' WHERE code = 'd';"},
		{name: "auto-commit directive", script: "-- txn-mode = off\nINSERT INTO dbo.cutover_t (code) VALUES ('f');\nINSERT INTO dbo.cutover_t (code) VALUES ('g');"},
		{name: "go batches", script: "-- txn-mode = off\nINSERT INTO dbo.cutover_t (code) VALUES ('h');\nGO\nINSERT INTO dbo.cutover_t (code) VALUES ('i');\nGO 2"},
		{name: "failing statement", script: "INSERT INTO dbo.cutover_t (code) VALUES ('j');\nINSERT INTO dbo.cutover_t (id) VALUES (1);"},
	}
}

// parityQueryCorpus is the editor corpus, with the options each case ran with.
func parityQueryCorpus() []struct {
	name    string
	script  string
	context db.QueryContext
} {
	qc := func() db.QueryContext { return db.QueryContext{MaximumSQLResultSize: 1 << 30} }
	return []struct {
		name    string
		script  string
		context db.QueryContext
	}{
		{name: "set and a limited select", script: "SET NOCOUNT OFF;\nSELECT TOP 3 id FROM dbo.cutover_t ORDER BY id;", context: db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 30}},
		{name: "declare and use a variable", script: "DECLARE @x INT;\nSELECT @x = MIN(id) FROM dbo.cutover_t;\nSELECT @x AS x;", context: qc()},
		{name: "procedure extras", script: "EXEC dbo.cutover_two_sets;", context: db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 30}},
		{name: "procedure extras across go", script: "EXEC dbo.cutover_two_sets;\nGO\nDECLARE @y INT;\nSELECT @y = MIN(id) FROM dbo.cutover_t;\nSELECT id FROM dbo.cutover_t WHERE id = 1;", context: qc()},
		{name: "masking withholds", script: "EXEC dbo.cutover_two_sets;\nGO\nUPDATE dbo.cutover_t SET code = code OUTPUT inserted.code;\nSELECT id FROM dbo.cutover_t WHERE id = 1;", context: db.QueryContext{MaskingEnabled: true, MaximumSQLResultSize: 1 << 30}},
		{name: "explain xml", script: "SELECT * FROM dbo.cutover_t;", context: db.QueryContext{Explain: true, Option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_XML}, MaximumSQLResultSize: 1 << 30}},
		{name: "explain text", script: "SELECT * FROM dbo.cutover_t;", context: db.QueryContext{Explain: true, MaximumSQLResultSize: 1 << 30}},
		{name: "size cut", script: "SELECT TOP 100 name FROM sys.all_objects;\nSELECT 1 AS one;", context: db.QueryContext{MaximumSQLResultSize: 1}},
		{name: "types", script: "SELECT CAST(0x AS VARBINARY(10)) AS empty_bin, CAST(0x01 AS VARBINARY(10)) AS bin, CAST(1.25 AS DECIMAL(18,2)) AS dec, CAST(1 AS BIT) AS flag, CAST('03:04:05' AS TIME) AS t, CAST('2024-01-02' AS DATE) AS d, CAST('2024-01-02T03:04:05.123+02:00' AS DATETIMEOFFSET) AS dto;", context: qc()},
		{name: "divide by zero alone", script: "SELECT 1/0 AS x;", context: qc()},
		{name: "divide by zero in a batch", script: "SELECT 1 AS one;\nSELECT 1/0 AS x;", context: qc()},
		{name: "unmatchable procedure", script: "EXEC dbo.cutover_two_sets;\nSELECT 1 AS one;", context: qc()},
		{name: "procedure extras after a failing batch", script: "EXEC dbo.cutover_two_sets;\nGO\nINSERT INTO dbo.cutover_t (id) VALUES (1);", context: qc()},
		{name: "noexec", script: "SET NOEXEC ON;\nSELECT 1 AS one;", context: qc()},
	}
}

// knownExecuteDifferences names the change-script cases whose result moves with
// the omni driver, and says what the new path does instead.
var knownExecuteDifferences = map[string]string{
	"failing statement": "the omni driver names the statement the server reported the error at",
}

// knownDifferences names the query cases whose result moves with the omni
// driver, and says what the new path does instead. Each entry is asserted
// below, so a change to one of these expectations still fails.
var knownDifferences = map[string]string{
	"failing statement":                      "the omni driver names the statement the server reported the error at",
	"types":                                  "an empty binary cell is an empty byte value; the old driver reported NULL for it",
	"divide by zero alone":                   "a lone failing statement's error rides on its result, not on the call",
	"divide by zero in a batch":              "the error that arrives while reading a batch's rows is the client's own text, not the server messages the old driver unpacked",
	"procedure extras after a failing batch": "the omni driver returns the procedure's later result sets after every statement's result even when a later batch fails; the old driver dropped them",
	"explain xml":                            "the omni driver states the plan's format in QueryPlan as well as returning its rows",
	"explain text":                           "the omni driver states the plan's format in QueryPlan as well as returning its rows",
}

// assertKnownExecuteDifference pins what the new path does for an Execute case
// the recording cannot match.
func assertKnownExecuteDifference(t *testing.T, got parityExecute) {
	t.Helper()
	switch got.Name {
	case "failing statement":
		require.Contains(t, got.Error, "INSERT INTO dbo.cutover_t (id) VALUES (1);")
	default:
		t.Fatalf("no assertion for known difference %q", got.Name)
	}
}

// assertKnownDifference pins what the new path does for a query case the
// recording cannot match, so the allowlist cannot hide a later regression.
func assertKnownDifference(t *testing.T, got parityQuery, gotErr error) {
	t.Helper()
	switch got.Name {
	case "types":
		require.Len(t, got.Results, 1)
		require.Equal(t, `bytes_value:""`, got.Results[0].Rows[0][0])
	case "divide by zero alone":
		require.NoError(t, gotErr)
		require.Len(t, got.Results, 1)
		require.Contains(t, got.Results[0].Error, "Divide by zero error encountered.")
	case "divide by zero in a batch":
		require.ErrorContains(t, gotErr, "mssql: Divide by zero error encountered. (8134)")
	case "procedure extras after a failing batch":
		// The procedure's first result and the failing statement's own result
		// come first, then the procedure's second result set.
		require.NoError(t, gotErr)
		require.Len(t, got.Results, 3)
		require.Contains(t, got.Results[1].Error, "IDENTITY_INSERT")
		require.Equal(t, "EXEC dbo.cutover_two_sets;", got.Results[2].Statement)
		require.Equal(t, []string{"s"}, got.Results[2].ColumnNames)
	case "explain xml":
		require.NoError(t, gotErr)
		require.Len(t, got.Results, 1)
		require.Equal(t, "format:XML", got.Results[0].QueryPlan)
		require.NotEmpty(t, got.Results[0].Rows)
	case "explain text":
		require.NoError(t, gotErr)
		require.Len(t, got.Results, 1)
		require.Equal(t, "format:TEXT", got.Results[0].QueryPlan)
		require.NotEmpty(t, got.Results[0].Rows)
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

func TestCutoverParityAgainstRecordedOldPath(t *testing.T) {
	ctx := context.Background()
	goldenBytes, err := os.ReadFile(filepath.Join("testdata", "cutover_parity.json"))
	require.NoError(t, err)
	var golden parityRecording
	require.NoError(t, json.Unmarshal(goldenBytes, &golden))

	opened, _ := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))

	execute := func(name, script string) parityExecute {
		var logs []string
		affected, err := opened.Execute(ctx, script, db.ExecuteOptions{
			LogCommandStatement: true,
			CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
				logs = append(logs, log.String())
				return nil
			},
		})
		entry := parityExecute{Name: name, Script: script, Affected: affected, Logs: logs}
		if err != nil {
			entry.Error = err.Error()
		}
		return entry
	}

	corpus := parityExecuteCorpus()
	require.Len(t, golden.Execute, len(corpus), "the corpus and its recording must have the same cases")
	for i, want := range golden.Execute {
		got := execute(want.Name, want.Script)
		require.Equal(t, want.Name, corpus[i].name)
		require.Equal(t, want.Script, corpus[i].script)
		if reason, known := knownExecuteDifferences[got.Name]; known {
			t.Logf("%s: known difference: %s", got.Name, reason)
			assertKnownExecuteDifference(t, got)
			continue
		}
		require.Equal(t, want.Affected, got.Affected, got.Name)
		require.Equal(t, want.Error, got.Error, got.Name)
		require.Equal(t, normalizeLogs(want.Logs), normalizeLogs(got.Logs), got.Name)
	}

	queries := parityQueryCorpus()
	require.Len(t, golden.Queries, len(queries), "the corpus and its recording must have the same cases")
	for i, query := range queries {
		t.Run(query.name, func(t *testing.T) {
			want := golden.Queries[i]
			require.Equal(t, want.Name, query.name)
			require.Equal(t, want.Script, query.script)

			results, callErr := opened.QueryConn(ctx, nil, query.script, query.context)
			got := parityQuery{Name: query.name, Script: query.script}
			if callErr != nil {
				got.CallError = callErr.Error()
			}
			for _, result := range results {
				got.Results = append(got.Results, summarizeParityResult(result))
			}

			if reason, known := knownDifferences[got.Name]; known {
				t.Logf("known difference: %s", reason)
				assertKnownDifference(t, got, callErr)
				return
			}
			if diff := cmp.Diff(want.Results, got.Results); diff != "" {
				t.Errorf("results (-want +got):\n%s", diff)
			}
			require.Equal(t, want.CallError, got.CallError)
		})
	}

	// The dump is where the cutover most visibly moves: the recording is what
	// the old renderer wrote for the same synced schema, and each assertion
	// below states one difference the new renderer makes. omni's driver suite
	// owns the rendering's own fidelity; this is the cutover's evidence.
	require.NotContains(t, golden.Dump, "CREATE SEQUENCE", "the old renderer wrote no sequence")
	require.NotContains(t, golden.Dump, "CREATE TRIGGER", "the old renderer wrote no trigger")
	require.NotContains(t, golden.Dump, "datetimeoffset(7)", "the old renderer dropped the precision")
	require.NotContains(t, golden.Dump, "CONSTRAINT [DF__", "the old renderer dropped a default constraint's name")
	require.Contains(t, golden.Dump, ";;", "the old renderer doubled a view's semicolon")
	require.Contains(t, golden.Dump, "[s_id] ASC", "the old renderer made an INCLUDE column a key column")
	require.NotContains(t, golden.Dump, "WHERE ([v] IS NOT NULL)", "the old renderer dropped the filtered index's predicate")

	var newDump bytes.Buffer
	metadata, err := opened.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.NoError(t, opened.Dump(ctx, &newDump, metadata))
	dump := newDump.String()
	require.Contains(t, dump, "CREATE SEQUENCE [dbo].[cutover_seq]")
	require.Contains(t, dump, "CREATE TRIGGER cutover_w_trg")
	require.Contains(t, dump, "datetimeoffset(7)")
	require.Contains(t, dump, "CONSTRAINT [DF__")
	require.NotContains(t, dump, ";;")
	require.NotContains(t, dump, "[s_id] ASC")
	require.Contains(t, dump, ") INCLUDE ([s_id])\nWHERE ([v] IS NOT NULL);")
}
