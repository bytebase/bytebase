package mongodb

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"

	metadatapb "github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// The golden under testdata holds what the pre-cutover driver did for this
// corpus, recorded with -record-cutover-parity while that driver was still in
// the tree. The test now runs the same corpus through the adapter over omni's
// mongo/driver and compares against the recording, so a case whose behaviour
// moves is one named in the knownDifferences tables below and asserted by hand
// instead. A compile proves no caller broke; this is what proves the
// statements, the ordering, the rows, the column names and the errors did not.

var recordParityGolden = flag.Bool("record-cutover-parity", false, "rewrite testdata/cutover_parity.json from the driver in the tree")

const parityGoldenPath = "testdata/cutover_parity.json"

// parityCorpus is the database every case reads. Every _id is written out, so
// two runs of the corpus hold the same documents, and a statement's output is
// comparable byte for byte.
const parityCorpus = `
db.createCollection("users");
db.users.insertMany([
  {_id: 1, name: "alice", age: 30, tags: ["admin"], profile: {city: "lisbon"}},
  {_id: 2, name: "bob", age: 17, tags: []},
  {_id: 3, name: "carol", age: 42}
]);
db.users.createIndex({name: 1}, {unique: true});
db.users.createIndex({age: -1, name: 1});
db.createCollection("orders");
db.orders.insertMany([
  {_id: 1, customer_id: 1, total: 100.5},
  {_id: 2, customer_id: 2, total: 900}
]);
db.orders.createIndex({customer_id: 1, created_at: -1});
db.createCollection("events");
db.events.createIndex({kind: 1, _id: -1});
db.sessions.createIndex({created_at: 1}, {expireAfterSeconds: 3600});
db.createCollection("capped", {capped: true, size: 4096});
db.runCommand({create: "v_adults", viewOn: "users", pipeline: [{$match: {age: {$gte: 18}}}]});
db['test.collection'].insertMany([{_id: 1, name: "Alice"}, {_id: 2, name: "Bob"}]);
db.createCollection("readings");
db.readings.insertMany([
  {_id: 1, value: 30}, {_id: 2, value: 30}, {_id: 3, value: 1.5}, {_id: 4, value: null}
]);
`

type parityRecording struct {
	Instance parityInstance  `json:"instance"`
	Schema   json.RawMessage `json:"schema"`
	Execute  []parityExecute `json:"execute"`
	Queries  []parityQuery   `json:"queries"`
	Dump     parityDump      `json:"dump"`
}

type parityInstance struct {
	// Database is the corpus's database, which the run allocates, so the two
	// runs' names differ and are not compared.
	Database string `json:"database"`
	Version  string `json:"version"`
	// Databases holds every database the shared container listed; other tests
	// in this package create databases of their own, so only the corpus's
	// presence is compared.
	Databases []string `json:"databases"`
	Roles     []string `json:"roles"`
}

type parityLog struct {
	Type string `json:"type"`
	// Range is the command's byte range as "start-end" when the run logged a
	// range rather than a statement.
	Range       string  `json:"range,omitempty"`
	Statement   string  `json:"statement,omitempty"`
	Affected    int64   `json:"affected,omitempty"`
	AllAffected []int64 `json:"all_affected,omitempty"`
	Error       string  `json:"error,omitempty"`
}

type parityExecute struct {
	Name     string      `json:"name"`
	Script   string      `json:"script"`
	Affected int64       `json:"affected"`
	Error    string      `json:"error"`
	Logs     []parityLog `json:"logs"`
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
	Results   []parityResult `json:"results"`
	CallError string         `json:"call_error"`
}

type parityDump struct {
	Bytes int    `json:"bytes"`
	Head  string `json:"head"`
	Error string `json:"error"`
}

// knownExecuteDifferences names the Execute cases whose result moves with the
// driver, with the reason. Each one is asserted by hand below, so the
// allowlist cannot hide a later change to it.
var knownExecuteDifferences = map[string]string{
	"insert one":                        "the driver reports the documents a write changed; the old driver reported none",
	"insert many":                       "the driver reports the documents a write changed; the old driver reported none",
	"statements separated by a newline": "the driver reports the documents a write changed; the old driver reported none",
	"statement failure mid script":      "the driver reports what the statements before the failure changed; the old driver reported none",
}

// knownQueryDifferences names the QueryConn cases whose result moves with the
// driver, with the reason.
var knownQueryDifferences = map[string]string{
	"count with limit":           "a row limit no longer reaches a count, which changed what the count counted",
	"result size cap":            "the driver stops at MaximumSQLResultSize, which the old driver read as no cap",
	"distinct scalar values":     "every value that is not a document is wrapped as {\"value\": ...}; the old driver rendered int32 as a number and null as <nil>",
	"explain request":            "the driver reports the plan it produced on the result, which the old driver left unset",
	"explain verbosity replaced": "the driver reports the plan it produced on the result, which the old driver left unset",
	"statement error":            "a statement's failure is its own result's, with the call returning no error",
	"parse error":                "text the mongo parser cannot split is one result carrying the parser's error, not a failed call",
	"multi statement":            "the driver answers one result per statement; the old driver sent the script to a client that refused more than one",
	"empty script":               "the driver runs nothing and answers no results; the old driver failed the call",
	"write with no plan":         "the refusal to plan a write is its result's error, not a failed call",
}

func TestCutoverParity(t *testing.T) {
	ctx := context.Background()
	c, database := testcontainer.NewMongoDatabase(t)
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     c.GetHost(),
			Port:     c.GetPort(),
			Username: c.GetUsername(),
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
		Password:          c.GetPassword(),
	}
	opened, err := (&Driver{}).Open(ctx, storepb.Engine_MONGODB, config)
	require.NoError(t, err)
	t.Cleanup(func() { _ = opened.Close(ctx) })
	require.NoError(t, opened.Ping(ctx))

	_, err = opened.Execute(ctx, parityCorpus, db.ExecuteOptions{})
	require.NoError(t, err)

	got := parityRecording{Instance: recordParityInstance(ctx, t, opened, database)}
	got.Schema = recordParitySchema(ctx, t, opened)
	got.Execute = recordParityExecute(ctx, t, opened)
	got.Queries = recordParityQueries(ctx, t, opened)
	got.Dump = recordParityDump(ctx, t, opened, got.Schema)

	if *recordParityGolden {
		encoded, err := json.MarshalIndent(got, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(filepath.Dir(parityGoldenPath), 0o755))
		require.NoError(t, os.WriteFile(parityGoldenPath, append(encoded, '\n'), 0o644))
		return
	}

	encoded, err := os.ReadFile(parityGoldenPath)
	require.NoError(t, err)
	var want parityRecording
	require.NoError(t, json.Unmarshal(encoded, &want))

	compareParityInstance(t, want.Instance, got.Instance)
	compareParitySchema(t, want.Schema, got.Schema)
	compareParityExecute(t, want.Execute, got.Execute)
	compareParityQueries(t, want.Queries, got.Queries)
	compareParityDump(t, want.Dump, got.Dump)
}

func recordParityInstance(ctx context.Context, t *testing.T, opened db.Driver, database string) parityInstance {
	t.Helper()
	instance, err := opened.SyncInstance(ctx)
	require.NoError(t, err)
	out := parityInstance{Database: database, Version: instance.Version}
	for _, metadata := range instance.Databases {
		out.Databases = append(out.Databases, metadata.Name)
	}
	for _, role := range instance.Metadata.GetRoles() {
		out.Roles = append(out.Roles, role.Name)
	}
	return out
}

func compareParityInstance(t *testing.T, want, got parityInstance) {
	t.Helper()
	require.Equal(t, want.Version, got.Version)
	require.Contains(t, got.Databases, got.Database)
	// Known difference: the driver names a user by its name where the old
	// driver used the users collection's _id, which carries the database the
	// user is defined in.
	require.Len(t, got.Roles, len(want.Roles))
	for i, name := range got.Roles {
		require.Equal(t, "admin."+name, want.Roles[i])
	}
}

func recordParitySchema(ctx context.Context, t *testing.T, opened db.Driver) json.RawMessage {
	t.Helper()
	schema, err := opened.SyncDBSchema(ctx)
	require.NoError(t, err)
	encoded, err := protojson.Marshal(schema)
	require.NoError(t, err)
	return encoded
}

// compareParitySchema compares the synced metadata after removing what the old
// driver did not record and the driver now does: a collection's create options,
// an index's options beyond its key and uniqueness, and a view's definition.
// Those are asserted to be present below, since they are what a dump reads.
func compareParitySchema(t *testing.T, wantRaw, gotRaw json.RawMessage) {
	t.Helper()
	got := &metadatapb.DatabaseSchemaMetadata{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(gotRaw, got))
	assertSchemaAdditions(t, got)
	want := normalizedParitySchema(t, wantRaw)
	if diff := cmp.Diff(want, normalizedParitySchema(t, gotRaw), protocmp.Transform()); diff != "" {
		t.Errorf("synced schema (-recorded +got):\n%s", diff)
	}
}

func normalizedParitySchema(t *testing.T, raw json.RawMessage) *metadatapb.DatabaseSchemaMetadata {
	t.Helper()
	schema := &metadatapb.DatabaseSchemaMetadata{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(raw, schema))
	for _, collection := range schema.GetSchemas() {
		for _, table := range collection.GetTables() {
			// Storage sizes move between containers; they are the server's
			// allocation, not the schema.
			table.DataSize = 0
			table.IndexSize = 0
			table.CreateOptions = ""
			for _, index := range table.GetIndexes() {
				index.Definition = ""
			}
		}
		for _, view := range collection.GetViews() {
			view.Definition = ""
		}
	}
	return schema
}

// assertSchemaAdditions pins what the driver records that the old driver did
// not, which is what its dump reads.
func assertSchemaAdditions(t *testing.T, schema *metadatapb.DatabaseSchemaMetadata) {
	t.Helper()
	require.Len(t, schema.GetSchemas(), 1)
	tables := map[string]*metadatapb.TableMetadata{}
	for _, table := range schema.GetSchemas()[0].GetTables() {
		tables[table.GetName()] = table
		for _, index := range table.GetIndexes() {
			require.NotEmpty(t, index.GetDefinition(), "index %s on %s", index.GetName(), table.GetName())
		}
	}
	require.Contains(t, tables["capped"].GetCreateOptions(), "capped")
	expiring := map[string]string{}
	for _, index := range tables["sessions"].GetIndexes() {
		expiring[index.GetName()] = index.GetDefinition()
	}
	require.Contains(t, expiring["created_at_1"], "expireAfterSeconds")
	for _, view := range schema.GetSchemas()[0].GetViews() {
		require.Contains(t, view.GetDefinition(), "viewOn")
	}
}

func recordParityExecute(ctx context.Context, t *testing.T, opened db.Driver) []parityExecute {
	t.Helper()
	cases := []struct{ name, script string }{
		{name: "insert one", script: `db.exec_t.insertOne({_id: 1});`},
		{name: "insert many", script: `db.exec_t.insertMany([{_id: 2}, {_id: 3}]);`},
		{name: "statements separated by a newline", script: "db.exec_t.insertOne({_id: 4})\ndb.exec_t.insertOne({_id: 5})"},
		{name: "statement failure mid script", script: "db.exec_t.insertOne({_id: 6});\ndb.runCommand({nosuchcommand: 1});\ndb.exec_t.insertOne({_id: 7})"},
		{name: "parse failure", script: "db.exec_t.find();\nthis is not mongosh"},
		{name: "empty script", script: " \n "},
	}
	var out []parityExecute
	for _, each := range cases {
		var logs []*storepb.TaskRunLog
		entry := parityExecute{Name: each.name, Script: each.script}
		affected, err := opened.Execute(ctx, each.script, db.ExecuteOptions{
			CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
				logs = append(logs, log)
				return nil
			},
		})
		entry.Affected = affected
		if err != nil {
			entry.Error = err.Error()
		}
		for _, log := range logs {
			entry.Logs = append(entry.Logs, summarizeParityLog(log))
		}
		out = append(out, entry)
	}
	return out
}

func summarizeParityLog(log *storepb.TaskRunLog) parityLog {
	out := parityLog{Type: log.GetType().String()}
	if command := log.GetCommandExecute(); command != nil {
		out.Statement = command.GetStatement()
		if command.GetRange() != nil {
			out.Range = fmt.Sprintf("%d-%d", command.GetRange().GetStart(), command.GetRange().GetEnd())
		}
	}
	if response := log.GetCommandResponse(); response != nil {
		out.Affected = response.GetAffectedRows()
		out.AllAffected = response.GetAllAffectedRows()
		out.Error = response.GetError()
	}
	return out
}

// normalizeParityLogs drops the affected-row counts the old driver never
// reported; the counts the driver reports now are pinned per case instead.
func normalizeParityLogs(logs []parityLog) []parityLog {
	out := make([]parityLog, len(logs))
	for i, log := range logs {
		log.Affected = 0
		log.AllAffected = nil
		out[i] = log
	}
	return out
}

func compareParityExecute(t *testing.T, want, got []parityExecute) {
	t.Helper()
	require.Len(t, got, len(want), "the corpus and its recording must have the same cases")
	for i, recorded := range want {
		each := got[i]
		require.Equal(t, recorded.Name, each.Name)
		require.Equal(t, recorded.Script, each.Script)
		// Statements issued, their ranges, their order and their errors are
		// what a cutover must not move, whatever the counts do.
		require.Equal(t, normalizeParityLogs(recorded.Logs), normalizeParityLogs(each.Logs), each.Name)
		require.Equal(t, recorded.Error, each.Error, each.Name)
		if reason, known := knownExecuteDifferences[each.Name]; known {
			t.Logf("%s: known difference: %s", each.Name, reason)
			assertKnownExecuteDifference(t, recorded, each)
			continue
		}
		require.Equal(t, recorded.Affected, each.Affected, each.Name)
	}
}

// assertKnownExecuteDifference pins the count the driver reports where the old
// driver reported none, so a change to it fails here.
func assertKnownExecuteDifference(t *testing.T, recorded, got parityExecute) {
	t.Helper()
	require.Zero(t, recorded.Affected)
	switch got.Name {
	case "insert one":
		require.Equal(t, int64(1), got.Affected)
		require.Equal(t, []int64{1}, got.Logs[1].AllAffected)
	case "insert many":
		require.Equal(t, int64(2), got.Affected)
		require.Equal(t, []int64{2}, got.Logs[1].AllAffected)
	case "statements separated by a newline":
		require.Equal(t, int64(2), got.Affected)
		require.Equal(t, []int64{1}, got.Logs[1].AllAffected)
		require.Equal(t, []int64{1}, got.Logs[3].AllAffected)
	case "statement failure mid script":
		require.Equal(t, int64(1), got.Affected)
		require.Equal(t, []int64{1}, got.Logs[1].AllAffected)
		require.NotEmpty(t, got.Logs[3].Error)
	default:
		t.Fatalf("no assertion for known difference %q", got.Name)
	}
}

func recordParityQueries(ctx context.Context, t *testing.T, opened db.Driver) []parityQuery {
	t.Helper()
	read := func() db.QueryContext {
		return db.QueryContext{MaximumSQLResultSize: 1 << 20}
	}
	cases := []struct {
		name    string
		script  string
		context db.QueryContext
	}{
		{name: "find", script: `db.users.find({_id: 1})`, context: read()},
		{name: "bracket notation", script: `db['test.collection'].find()`, context: read()},
		{name: "limit rows", script: `db.users.find()`, context: db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 20}},
		{name: "limit kept", script: `db.users.find().limit(1)`, context: db.QueryContext{Limit: 5, MaximumSQLResultSize: 1 << 20}},
		{name: "count with limit", script: `db.users.countDocuments({})`, context: db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 20}},
		{name: "distinct scalar values", script: `db.readings.distinct("value")`, context: read()},
		{name: "aggregate", script: `db.users.aggregate([{$group: {_id: null, n: {$sum: 1}}}])`, context: read()},
		{name: "no documents", script: `db.events.find()`, context: read()},
		{name: "write through the query path", script: `db.readings.insertOne({_id: 9, value: 2.5})`, context: read()},
		{name: "result size cap", script: `db.users.find()`, context: db.QueryContext{MaximumSQLResultSize: 60}},
		{name: "explain request", script: `db.users.find({age: {$gt: 18}})`, context: db.QueryContext{Explain: true, Option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON}, MaximumSQLResultSize: 1 << 20}},
		{name: "explain verbosity replaced", script: `db.users.find({age: {$gt: 18}}).explain("executionStats")`, context: db.QueryContext{Explain: true, Option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON}, MaximumSQLResultSize: 1 << 20}},
		{name: "explain ran as a query", script: `db.users.find().explain()`, context: read()},
		{name: "explain measured as a query", script: `db.users.find().explain("executionStats")`, context: read()},
		{name: "write with no plan", script: `db.readings.insertOne({_id: 10})`, context: db.QueryContext{Explain: true, MaximumSQLResultSize: 1 << 20}},
		{name: "statement error", script: `db.runCommand({nosuchcommand: 1})`, context: read()},
		{name: "parse error", script: `this is not mongosh`, context: read()},
		{name: "multi statement", script: "db.users.find();\ndb.orders.find()", context: read()},
		{name: "empty script", script: " \n ", context: read()},
	}
	var out []parityQuery
	for _, each := range cases {
		entry := parityQuery{Name: each.name, Script: each.script}
		results, err := opened.QueryConn(ctx, each.script, each.context)
		if err != nil {
			entry.CallError = err.Error()
		}
		for _, result := range results {
			entry.Results = append(entry.Results, summarizeParityResult(result))
		}
		out = append(out, entry)
	}
	return out
}

func summarizeParityResult(result *v1pb.QueryResult) parityResult {
	out := parityResult{
		Statement:       result.GetStatement(),
		ColumnNames:     result.GetColumnNames(),
		ColumnTypeNames: result.GetColumnTypeNames(),
		Error:           result.GetError(),
		RowsCount:       result.GetRowsCount(),
	}
	if plan := result.GetQueryPlan(); plan != nil {
		// Protobuf's text format puts an extra space between fields at random,
		// so the plan is compared with its spacing normalized.
		out.QueryPlan = strings.Join(strings.Fields(plan.String()), " ")
	}
	for _, row := range result.GetRows() {
		values := make([]string, 0, len(row.GetValues()))
		for _, value := range row.GetValues() {
			if text, ok := value.GetKind().(*v1pb.RowValue_StringValue); ok {
				values = append(values, parityHostPattern.ReplaceAllString(text.StringValue, `"host":"<container>"`))
				continue
			}
			values = append(values, value.String())
		}
		out.Rows = append(out.Rows, values)
	}
	return out
}

// parityHostPattern matches the server host a plan document names. The pinned
// container's hostname is allocated per run, so two runs' plans differ there
// and nowhere else.
var parityHostPattern = regexp.MustCompile(`"host":"[^"]*"`)

func compareParityQueries(t *testing.T, want, got []parityQuery) {
	t.Helper()
	require.Len(t, got, len(want), "the corpus and its recording must have the same cases")
	for i, recorded := range want {
		each := got[i]
		require.Equal(t, recorded.Name, each.Name)
		require.Equal(t, recorded.Script, each.Script)
		if reason, known := knownQueryDifferences[each.Name]; known {
			t.Logf("%s: known difference: %s", each.Name, reason)
			assertKnownQueryDifference(t, recorded, each)
			continue
		}
		if diff := cmp.Diff(recorded.Results, each.Results); diff != "" {
			t.Errorf("%s: results (-recorded +got):\n%s", each.Name, diff)
		}
		require.Equal(t, recorded.CallError, each.CallError, each.Name)
	}
}

// assertKnownQueryDifference pins what the driver answers where the old driver
// did not, so a change to one of these expectations still fails.
func assertKnownQueryDifference(t *testing.T, recorded, got parityQuery) {
	t.Helper()
	switch got.Name {
	case "count with limit":
		require.Equal(t, [][]string{{`{"value":2}`}}, recorded.Results[0].Rows)
		require.Equal(t, [][]string{{`{"value":3}`}}, got.Results[0].Rows)
		require.Equal(t, recorded.Results[0].Statement, got.Results[0].Statement)
	case "result size cap":
		require.Empty(t, recorded.Results[0].Error)
		require.Len(t, recorded.Results[0].Rows, 3)
		require.Equal(t, recorded.Results[0].Statement, got.Results[0].Statement)
		require.Len(t, got.Results[0].Rows, 1)
		require.NotEmpty(t, got.Results[0].Error)
	case "distinct scalar values":
		require.ElementsMatch(t, []string{"30", "1.5", "<nil>"}, parityRowValues(recorded.Results[0]))
		require.ElementsMatch(t, []string{`{"value":30}`, `{"value":1.5}`, `{"value":null}`}, parityRowValues(got.Results[0]))
	case "explain request", "explain verbosity replaced":
		// The rows are the same plan the old driver returned; only the result
		// now says it is one.
		require.Empty(t, recorded.Results[0].QueryPlan)
		require.Equal(t, recorded.Results[0].Rows, got.Results[0].Rows)
		require.Equal(t, recorded.Results[0].Statement, got.Results[0].Statement)
		require.Contains(t, got.Results[0].QueryPlan, "format:JSON")
	case "statement error":
		require.NotEmpty(t, recorded.CallError)
		require.Empty(t, got.CallError)
		require.Len(t, got.Results, 1)
		require.Contains(t, got.Results[0].Error, "nosuchcommand")
	case "write with no plan":
		require.NotEmpty(t, recorded.CallError)
		require.Empty(t, got.CallError)
		require.Len(t, got.Results, 1)
		require.Contains(t, got.Results[0].Error, "insertOne")
	case "parse error":
		require.NotEmpty(t, recorded.CallError)
		require.Empty(t, got.CallError)
		require.Len(t, got.Results, 1)
		require.Contains(t, got.Results[0].Error, "syntax error")
		require.Equal(t, "this is not mongosh", got.Results[0].Statement)
	case "multi statement":
		require.NotEmpty(t, recorded.CallError)
		require.Empty(t, got.CallError)
		require.Len(t, got.Results, 2)
		require.Equal(t, []string{"db.users.find()", "db.orders.find()"}, []string{got.Results[0].Statement, got.Results[1].Statement})
		require.Len(t, got.Results[0].Rows, 3)
		require.Len(t, got.Results[1].Rows, 2)
	case "empty script":
		require.Len(t, recorded.Results, 1)
		require.Empty(t, recorded.Results[0].Rows)
		require.Empty(t, got.Results)
	default:
		t.Fatalf("no assertion for known difference %q", got.Name)
	}
}

// parityRowValues flattens a result's rows into one list of cell text, for a
// case whose row order the server decides.
func parityRowValues(result parityResult) []string {
	var values []string
	for _, row := range result.Rows {
		values = append(values, row...)
	}
	return values
}

func recordParityDump(ctx context.Context, t *testing.T, opened db.Driver, schemaRaw json.RawMessage) parityDump {
	t.Helper()
	schema := &metadatapb.DatabaseSchemaMetadata{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(schemaRaw, schema))
	var out strings.Builder
	entry := parityDump{}
	if err := opened.Dump(ctx, &out, schema); err != nil {
		entry.Error = err.Error()
	}
	entry.Bytes = out.Len()
	if head, _, found := strings.Cut(out.String(), "\n"); found {
		entry.Head = head
	}
	return entry
}

// compareParityDump holds the dump change: the old driver wrote nothing, and
// the driver renders the synced database as a mongosh script.
func compareParityDump(t *testing.T, want, got parityDump) {
	t.Helper()
	require.Zero(t, want.Bytes, "the recorded tree wrote no dump")
	require.Empty(t, want.Error)
	require.Empty(t, got.Error)
	require.Positive(t, got.Bytes)
	require.Contains(t, got.Head, "MongoDB schema dump of database")
}
