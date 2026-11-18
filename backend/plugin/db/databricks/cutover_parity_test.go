package databricks

// cutover_parity_test.go is the cutover's differential check. testdata/cutover_parity.json
// was recorded by driving the PRE-CUTOVER driver against the parity workspace in
// parity_workspace_test.go: the requests it sent, the results it returned, the
// metadata it synced, the rows it reported for a script, and the dump it wrote.
// This test drives the adapter over the same workspace with the same corpus and
// compares the two, so every difference it meets is a difference the cutover
// introduced. A case omni deliberately changes is asserted both ways in
// knownDifferences, so the record pins what moved and cannot hide a later
// regression.
//
// Nothing here needs an account, a token, a warehouse or a network: both drivers
// talk HTTP to the server this test starts, and the answers are the ones
// omni/databricks/driver/VERIFY.md recorded from the pinned workspace.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	metadatapb "github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// parityRecording is the shape testdata/cutover_parity.json was written in.
type parityRecording struct {
	PingError        string           `json:"ping_error"`
	PingRequests     []parityRequest  `json:"ping_requests"`
	Instance         json.RawMessage  `json:"instance"`
	InstanceRequests []parityRequest  `json:"instance_requests"`
	Schema           json.RawMessage  `json:"schema"`
	SchemaRequests   []parityRequest  `json:"schema_requests"`
	Queries          []parityQueryLog `json:"queries"`
	Executes         []parityExecLog  `json:"executes"`
	Dump             string           `json:"dump"`
	DumpError        string           `json:"dump_error"`
}

type parityQueryLog struct {
	Name      string            `json:"name"`
	Script    string            `json:"script"`
	Schema    string            `json:"schema"`
	Limit     int               `json:"limit"`
	Explain   bool              `json:"explain"`
	Requests  []parityRequest   `json:"requests"`
	Results   []json.RawMessage `json:"results"`
	CallError string            `json:"call_error"`
}

type parityExecLog struct {
	Name     string            `json:"name"`
	Script   string            `json:"script"`
	Affected int64             `json:"affected"`
	Logs     []json.RawMessage `json:"logs"`
	Error    string            `json:"error"`
	Requests []parityRequest   `json:"requests"`
}

type parityInstanceLog struct {
	Version   string            `json:"version"`
	Databases []json.RawMessage `json:"databases"`
}

// parityCases names the corpus case whose results the new path answers
// differently, and says what it answers instead. Each one is asserted below.
var knownDifferences = map[string]string{
	"no rows":                                 "a statement that returned no rows still reports its columns and does not end the script; the old driver dropped its result",
	"no rows before a statement":              "the statement after a zero-row result runs; the old driver stopped at the zero-row result",
	"a DDL statement":                         "a DDL answers an empty result; the old driver dropped it and stopped the script",
	"a statement the workspace refuses":       "the statement's error is its own result and the earlier results survive; the old driver returned the call's error with no results",
	"a failure between two statements":        "the first statement's result survives the second's failure; the old driver returned no results",
	"a lone terminator between two":           "a line holding only a terminator is sent as the empty statement; the old driver sent the terminator itself",
	"two statements on one line":              "a one-line script is one statement, as the splitter reads it, and its error is a result rather than the call's error",
	"an explain request":                      "the result carries QueryPlan{TEXT}; the API used to fill that in for an explain request",
	"a statement the caller wrote as EXPLAIN": "the result carries QueryPlan{TEXT}, so the viewer draws the plan it used to print as rows",
}

func openParityDriver(t *testing.T, workspace *parityWorkspace) db.Driver {
	t.Helper()
	driver, err := newDriver().Open(context.Background(), storepb.Engine_DATABRICKS, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:                     workspace.host(),
			AuthenticationPrivateKey: parityToken,
			WarehouseId:              parityWarehouseID,
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: parityCatalog},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Close(context.Background()) })
	return driver
}

func readParityRecording(t *testing.T) *parityRecording {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "cutover_parity.json"))
	require.NoError(t, err)
	var recording parityRecording
	require.NoError(t, json.Unmarshal(body, &recording))
	return &recording
}

func TestCutoverParityPing(t *testing.T) {
	workspace := newParityWorkspace(t)
	driver := openParityDriver(t, workspace)
	recording := readParityRecording(t)

	require.Equal(t, recording.PingError, errorText(driver.Ping(context.Background())))
	require.Equal(t, recording.PingRequests, workspace.takeRequests())
}

func TestCutoverParitySyncInstance(t *testing.T) {
	workspace := newParityWorkspace(t)
	driver := openParityDriver(t, workspace)
	recording := readParityRecording(t)

	got, err := driver.SyncInstance(context.Background())
	require.NoError(t, err)
	// Known difference: the old driver read every schema and table of every
	// catalog during an instance sync; the adapter reads the catalogs' names.
	require.Equal(t, []string{
		"GET /api/2.1/unity-catalog/catalogs",
		"POST /api/2.0/sql/statements",
	}, sortedRoutes(workspace.takeRequests()))
	require.Subset(t, sortedRoutes(recording.InstanceRequests), []string{
		"GET /api/2.1/unity-catalog/schemas",
		"GET /api/2.1/unity-catalog/tables",
	})

	var want parityInstanceLog
	require.NoError(t, json.Unmarshal(recording.Instance, &want))
	require.Equal(t, want.Version, got.Version)
	require.Equal(t, databaseNames(t, want.Databases), databaseNamesOf(got.Databases))
	// Known difference: a catalog is listed by name; the old driver read every
	// schema and table of every catalog here and the product reads only the name.
	for _, database := range got.Databases {
		require.Empty(t, database.Schemas, "SyncInstance lists a catalog without its schemas")
	}
	// The old driver's schemas are still in the record, which is what makes the
	// difference visible rather than an omission.
	for _, raw := range want.Databases {
		var database map[string]any
		require.NoError(t, json.Unmarshal(raw, &database))
		require.NotEmpty(t, database["schemas"], "the record holds the old driver's schemas")
	}
}

func TestCutoverParitySyncDBSchema(t *testing.T) {
	workspace := newParityWorkspace(t)
	driver := openParityDriver(t, workspace)
	recording := readParityRecording(t)

	got, err := driver.SyncDBSchema(context.Background())
	require.NoError(t, err)
	require.Equal(t, sortedRoutes(recording.SchemaRequests), sortedRoutes(workspace.takeRequests()))

	want := &metadatapb.DatabaseSchemaMetadata{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(recording.Schema, want))
	// Known difference: the driver sorts the schemas, their tables and the tables'
	// columns, and a view's dependency columns, so a sync does not depend on the
	// service's listing order.
	if diff := cmp.Diff(sortMetadata(want), sortMetadata(got), protocmp.Transform()); diff != "" {
		t.Errorf("synced schema differs from the record (-old +new):\n%s", diff)
	}
	if diff := cmp.Diff(sortMetadata(got), got, protocmp.Transform()); diff != "" {
		t.Errorf("the synced metadata is already in order (-sorted +synced):\n%s", diff)
	}
}

func TestCutoverParityQuery(t *testing.T) {
	workspace := newParityWorkspace(t)
	driver := openParityDriver(t, workspace)
	recording := readParityRecording(t)

	// The corpus is this test's, and the record must hold every case of it.
	require.Len(t, recording.Queries, len(parityQueryCorpus))
	for i, tc := range parityQueryCorpus {
		require.Equal(t, tc.name, recording.Queries[i].Name)
		require.Equal(t, tc.script, recording.Queries[i].Script)
		require.Equal(t, tc.schema, recording.Queries[i].Schema)
		require.Equal(t, tc.limit, recording.Queries[i].Limit)
		require.Equal(t, tc.explain, recording.Queries[i].Explain)
	}

	for _, recorded := range recording.Queries {
		t.Run(recorded.Name, func(t *testing.T) {
			got, err := driver.QueryConn(context.Background(), nil, recorded.Script, db.QueryContext{
				Schema:  recorded.Schema,
				Limit:   recorded.Limit,
				Explain: recorded.Explain,
			})
			requests := workspace.takeRequests()
			if _, known := knownDifferences[recorded.Name]; !known {
				compareRequests(t, recorded.Requests, requests, recorded.Schema)
				require.Empty(t, recorded.CallError, "the record holds a call error for a case with no known difference")
				require.NoError(t, err)
				if diff := cmp.Diff(normalizedResults(t, recorded.Results), normalizeResults(got), protocmp.Transform()); diff != "" {
					t.Errorf("results differ from the record (-old +new):\n%s", diff)
				}
				return
			}
			// A known difference may run fewer or more statements than the old
			// driver; the ones it does run are the old ones.
			require.GreaterOrEqual(t, len(requests), len(recorded.Requests))
			compareRequests(t, recorded.Requests, requests[:len(recorded.Requests)], recorded.Schema)
			compareKnownDifferenceQuery(t, recorded, got, err)
		})
	}
}

// compareKnownDifferenceQuery pins the cases the driver answers differently: the
// old answer from the record, and the new one here.
func compareKnownDifferenceQuery(t *testing.T, recorded parityQueryLog, got []*v1pb.QueryResult, err error) {
	t.Helper()
	switch recorded.Name {
	case "no rows":
		require.NoError(t, err)
		require.Empty(t, recorded.Results, "the old driver dropped the result")
		require.Len(t, got, 1)
		require.Equal(t, []string{"id", "label"}, got[0].ColumnNames)
		require.Empty(t, got[0].Rows)
	case "no rows before a statement":
		require.NoError(t, err)
		require.Empty(t, recorded.Results)
		require.Len(t, recorded.Requests, 1, "the old driver never ran the second statement")
		require.Len(t, got, 2)
		require.Empty(t, got[0].Rows)
		require.Equal(t, []string{"1"}, got[1].ColumnNames)
		require.Len(t, got[1].Rows, 1)
	case "a DDL statement":
		require.NoError(t, err)
		require.Empty(t, recorded.Results, "the old driver dropped the DDL's result")
		require.Len(t, got, 1)
		require.Empty(t, got[0].ColumnNames)
		require.Empty(t, got[0].Rows)
		require.Zero(t, got[0].RowsCount)
	case "a statement the workspace refuses":
		require.NoError(t, err)
		require.Contains(t, recorded.CallError, "FAILED: BAD_REQUEST")
		require.Empty(t, recorded.Results)
		require.Len(t, got, 1)
		require.Contains(t, got[0].Error, "FAILED: BAD_REQUEST")
	case "a failure between two statements":
		require.NoError(t, err)
		require.Contains(t, recorded.CallError, "FAILED: BAD_REQUEST")
		require.Empty(t, recorded.Results, "the old driver returned no results at all")
		require.Len(t, got, 2)
		require.Equal(t, []string{"1"}, got[0].ColumnNames)
		require.Empty(t, got[0].Error)
		require.Contains(t, got[1].Error, "FAILED: BAD_REQUEST")
	case "a lone terminator between two":
		require.NoError(t, err)
		require.Contains(t, recorded.CallError, "FAILED: BAD_REQUEST")
		// The old driver sent the terminator; the contract trims it to nothing and
		// sends the empty statement, which the workspace refuses.
		require.Equal(t, "\n;", recorded.Requests[1].Statement)
		require.Len(t, got, 2)
		require.Empty(t, got[0].Error)
		require.Contains(t, got[1].Error, "FAILED: BAD_REQUEST")
	case "two statements on one line":
		require.NoError(t, err)
		require.Contains(t, recorded.CallError, "FAILED: BAD_REQUEST")
		require.Empty(t, recorded.Results)
		require.Len(t, got, 1)
		require.Contains(t, got[0].Error, "FAILED: BAD_REQUEST")
	case "an explain request", "a statement the caller wrote as EXPLAIN":
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.NotNil(t, got[0].QueryPlan)
		require.Equal(t, v1pb.QueryOption_TEXT, got[0].QueryPlan.Format)
		require.False(t, got[0].QueryPlan.Executed)
		if recorded.Name == "an explain request" {
			// The API filled this in for an explain request, so the final answer is
			// the same; the driver now states it.
			old := &v1pb.QueryResult{}
			require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(recorded.Results[0], old))
			require.Nil(t, old.QueryPlan, "the record holds no plan from the old driver")
		}
	default:
		t.Fatalf("no assertion for known difference %q", recorded.Name)
	}
}

func TestCutoverParityExecute(t *testing.T) {
	workspace := newParityWorkspace(t)
	driver := openParityDriver(t, workspace)
	recording := readParityRecording(t)

	require.Len(t, recording.Executes, len(parityExecuteCorpus))
	for i, tc := range parityExecuteCorpus {
		require.Equal(t, tc.name, recording.Executes[i].Name)
		require.Equal(t, tc.script, recording.Executes[i].Script)
	}

	for _, recorded := range recording.Executes {
		t.Run(recorded.Name, func(t *testing.T) {
			collector := &parityLogCollector{}
			affected, err := driver.Execute(context.Background(), recorded.Script, db.ExecuteOptions{
				CreateTaskRunLog: collector.create,
			})
			compareRequests(t, recorded.Requests, workspace.takeRequests(), "")

			switch recorded.Name {
			case "a statement the workspace refuses", "a failure after a write":
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to execute statement: FAILED: BAD_REQUEST")
				require.Contains(t, recorded.Error, "FAILED: BAD_REQUEST")
				require.Zero(t, affected)
				require.Zero(t, recorded.Affected, "the old driver returned 0")
			default:
				require.NoError(t, err)
				require.Empty(t, recorded.Error)
				require.Zero(t, recorded.Affected, "the old driver returned 0 for every script")
				switch recorded.Name {
				case "an insert", "a write after a read":
					require.EqualValues(t, 2, affected)
				case "an update":
					require.EqualValues(t, 1, affected)
				case "a DDL statement":
					require.Zero(t, affected)
				case "a transaction directive":
					require.EqualValues(t, 2, affected)
				default:
					t.Fatalf("no assertion for %q", recorded.Name)
				}
			}

			// Known difference: the old driver logged nothing, and the adapter
			// reports each command and its response.
			require.Empty(t, recorded.Logs, "the old driver logged nothing")
			require.NotEmpty(t, collector.logs, "the adapter logs the script's commands")
			require.Equal(t, len(recorded.Requests), collector.commands)
		})
	}
}

// A task run log with LogCommandStatement off recovers a command's text from the
// caller's script by range, so the range must name the script the task ran rather
// than the one left after the directive line was cut.
func TestExecuteLogsRangesInTheCallersScript(t *testing.T) {
	workspace := newParityWorkspace(t)
	driver := openParityDriver(t, workspace)

	script := "-- txn-mode=off\nINSERT INTO workspace.omni_record.lookup (id, label) VALUES (1, 'x'), (2, 'y')"
	collector := &parityLogCollector{}
	_, err := driver.Execute(context.Background(), script, db.ExecuteOptions{CreateTaskRunLog: collector.create})
	require.NoError(t, err)
	require.NotEmpty(t, collector.logs)

	var entry storepb.TaskRunLog
	require.NoError(t, protojson.Unmarshal(collector.logs[0], &entry))
	command := entry.GetCommandExecute()
	require.NotNil(t, command)
	got := script[command.GetRange().GetStart():command.GetRange().GetEnd()]
	require.NotContains(t, got, "txn-mode")
	require.Contains(t, got, "INSERT INTO workspace.omni_record.lookup")
}

// A directive can follow another leading comment, so the bytes removed are not
// always a prefix: every range the driver reports has to come back to the
// caller's script, not just the first statement's.
func TestDirectiveShiftMapsEveryRemovedShape(t *testing.T) {
	for _, script := range []string{
		"-- txn-mode=off\nINSERT INTO t VALUES (1)",
		"-- keep\n-- txn-mode=off\nINSERT INTO t VALUES (1)",
		"-- txn-mode=off\n-- keep\nINSERT INTO t VALUES (1)",
		"-- keep\n\n-- txn-isolation=serializable\n\nINSERT INTO t VALUES (1)",
		"INSERT INTO t VALUES (1)",
	} {
		t.Run(script, func(t *testing.T) {
			_, cleaned := base.ParseTransactionConfig(script)
			shift := directiveShift(script, cleaned)
			at := strings.Index(cleaned, "INSERT")
			require.NotEqual(t, -1, at)
			require.Equal(t, strings.Index(script, "INSERT"), shift(at))
			require.Equal(t, len(script), shift(len(cleaned)))
		})
	}
}

func TestCutoverParityDump(t *testing.T) {
	workspace := newParityWorkspace(t)
	driver := openParityDriver(t, workspace)

	var got strings.Builder
	require.NoError(t, driver.Dump(context.Background(), &got, nil))
	workspace.takeRequests()

	// The dump text is the server's own SHOW CREATE, so its fidelity is
	// databricks/driver's to test. What this cutover owns is the difference from
	// the old renderer, whose text is in the record: it wrote the catalog
	// unquoted and rebuilt a materialized view from DESC FORMATTED, which the
	// workspace does not accept as a statement, so the old dump did not restore.
	recording := readParityRecording(t)
	require.Contains(t, recording.Dump, "CREATE CATALOG workspace\n")
	require.Contains(t, recording.Dump, "CREATE MATERIALIZED VIEW `workspace`.`omni_record`.`totals`")
	require.NotContains(t, got.String(), "DESC FORMATTED")
	require.NotContains(t, got.String(), "CREATE CATALOG workspace\n")
	require.Contains(t, got.String(), "CREATE VIEW `workspace`.`omni_record`.`recent` AS SELECT id FROM workspace.omni_record.facts")
}

func TestDatabricksConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		config   db.ConnectionConfig
		want     string
		wantCata string
		wantWrhs string
	}{
		{
			name: "the token is the authentication private key and the catalog is the database",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                     "https://dbc-00000000-0000.cloud.databricks.com",
					AuthenticationPrivateKey: "token",
					WarehouseId:              "warehouse",
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "workspace"},
			},
			want:     "token",
			wantCata: "workspace",
			wantWrhs: "warehouse",
		},
		{
			name: "a resolved token stands in when the data source carries none",
			config: db.ConnectionConfig{
				DataSource:        &storepb.DataSource{Host: "https://dbc.example.com", WarehouseId: "warehouse"},
				ConnectionContext: db.ConnectionContext{DatabaseName: "workspace"},
				Password:          "resolved",
			},
			want:     "resolved",
			wantCata: "workspace",
			wantWrhs: "warehouse",
		},
		{
			// The SDK's own transport verifies the workspace's certificate, so an
			// SSL data source and an SSH one change nothing here, which is what the
			// previous driver did with their settings.
			name: "an SSL or SSH data source adds no TLS and no dialer",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                     "https://dbc.example.com",
					AuthenticationPrivateKey: "token",
					WarehouseId:              "warehouse",
					UseSsl:                   true,
					VerifyTlsCertificate:     true,
					SshHost:                  "bastion.example.com",
					SshPort:                  "22",
					SshUser:                  "user",
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "workspace"},
			},
			want:     "token",
			wantCata: "workspace",
			wantWrhs: "warehouse",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := databricksConfig(tc.config)
			require.Equal(t, tc.want, got.Password)
			require.Equal(t, tc.wantCata, got.Database)
			require.Equal(t, tc.wantWrhs, got.Params["warehouseId"])
			require.Equal(t, tc.config.DataSource.GetHost(), got.Host)
			require.Nil(t, got.TLS)
			require.Nil(t, got.Dial)
		})
	}
}

// TestOpenRefusesAnIncompleteConfig pins the checks that keep Open from building
// a client that cannot name a workspace.
func TestOpenRefusesAnIncompleteConfig(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config db.ConnectionConfig
		want   string
	}{
		{
			name:   "no host",
			config: db.ConnectionConfig{DataSource: &storepb.DataSource{AuthenticationPrivateKey: "token", WarehouseId: "warehouse"}},
			want:   "host must be set",
		},
		{
			name:   "no token",
			config: db.ConnectionConfig{DataSource: &storepb.DataSource{Host: "https://dbc.example.com", WarehouseId: "warehouse"}},
			want:   "token must be set",
		},
		{
			name:   "no warehouse",
			config: db.ConnectionConfig{DataSource: &storepb.DataSource{Host: "https://dbc.example.com", AuthenticationPrivateKey: "token"}},
			want:   "warehouse ID must be set",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newDriver().Open(context.Background(), storepb.Engine_DATABRICKS, tc.config)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

// parityLogCollector records the task run log the adapter writes.
type parityLogCollector struct {
	logs     []json.RawMessage
	commands int
}

func (c *parityLogCollector) create(_ time.Time, entry *storepb.TaskRunLog) error {
	if entry.CommandExecute != nil {
		c.commands++
	}
	body, err := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}.Marshal(entry)
	if err != nil {
		return err
	}
	c.logs = append(c.logs, body)
	return nil
}

// compareRequests requires the same requests in the same order, with the same
// statement text once the splitter's trailing terminator and leading whitespace
// are removed, and the same catalog and row limit.
//
// The schema is the one known difference in the request shape: the old driver
// sent only the catalog, so the record's schema is empty and the adapter sends
// the query's schema.
func compareRequests(t *testing.T, want, got []parityRequest, wantSchema string) {
	t.Helper()
	require.Len(t, got, len(want), "the same requests in the same order")
	for i, expected := range want {
		actual := got[i]
		require.Equal(t, expected.Method, actual.Method, "request %d", i)
		require.Equal(t, expected.Route, actual.Route, "request %d", i)
		if expected.Method != "POST" {
			continue
		}
		require.Equal(t, normalizeStatement(expected.Statement), normalizeStatement(actual.Statement), "request %d", i)
		require.Equal(t, expected.Catalog, actual.Catalog, "request %d", i)
		require.Equal(t, expected.RowLimit, actual.RowLimit, "request %d", i)
		require.Empty(t, expected.Schema, "the record holds the old driver's request, which carried no schema")
		require.Equal(t, wantSchema, actual.Schema, "request %d carries the query's schema", i)
	}
}

// normalizedResults unmarshals the record's results and clears the fields that
// are the run's rather than the statement's: the latency a caller measures, and
// the statement text the splitter sliced.
func normalizedResults(t *testing.T, raw []json.RawMessage) []*v1pb.QueryResult {
	t.Helper()
	results := make([]*v1pb.QueryResult, 0, len(raw))
	for _, body := range raw {
		result := &v1pb.QueryResult{}
		require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(body, result))
		result.Latency = nil
		result.Statement = normalizeStatement(result.Statement)
		results = append(results, result)
	}
	return results
}

func normalizeResults(results []*v1pb.QueryResult) []*v1pb.QueryResult {
	normalized := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		clone := proto.CloneOf(result)
		clone.Latency = nil
		clone.Statement = normalizeStatement(clone.Statement)
		normalized = append(normalized, clone)
	}
	return normalized
}

// sortMetadata orders a synced catalog so two syncs compare regardless of the
// order their listings arrived in.
func sortMetadata(in *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	if in == nil {
		return nil
	}
	out := proto.CloneOf(in)
	slices.SortFunc(out.Schemas, func(a, b *metadatapb.SchemaMetadata) int { return strings.Compare(a.Name, b.Name) })
	for _, schema := range out.Schemas {
		sortNamedTables(schema)
	}
	return out
}

func sortNamedTables(schema *metadatapb.SchemaMetadata) {
	slices.SortFunc(schema.Tables, func(a, b *metadatapb.TableMetadata) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(schema.Views, func(a, b *metadatapb.ViewMetadata) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(schema.MaterializedViews, func(a, b *metadatapb.MaterializedViewMetadata) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(schema.ExternalTables, func(a, b *metadatapb.ExternalTableMetadata) int { return strings.Compare(a.Name, b.Name) })
	for _, table := range schema.Tables {
		slices.SortStableFunc(table.Columns, func(a, b *metadatapb.ColumnMetadata) int {
			if a.Position != b.Position {
				return int(a.Position - b.Position)
			}
			return strings.Compare(a.Name, b.Name)
		})
	}
	for _, view := range schema.Views {
		slices.SortFunc(view.DependencyColumns, func(a, b *metadatapb.DependencyColumn) int {
			return strings.Compare(a.Column, b.Column)
		})
	}
}

func databaseNames(t *testing.T, raw []json.RawMessage) []string {
	t.Helper()
	var names []string
	for _, body := range raw {
		var database metadatapb.DatabaseSchemaMetadata
		require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(body, &database))
		names = append(names, database.Name)
	}
	slices.Sort(names)
	return names
}

func databaseNamesOf(databases []*metadatapb.DatabaseSchemaMetadata) []string {
	names := make([]string, 0, len(databases))
	for _, database := range databases {
		names = append(names, database.Name)
	}
	slices.Sort(names)
	return names
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
