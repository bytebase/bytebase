package cosmosdb

// This file is the cutover's differential check: it replays the corpus in
// testdata/cutover_parity.json against the adapter and compares what the
// adapter sends the emulator and returns to its caller with what the
// pre-cutover driver recorded. The golden was recorded from the old driver
// against the same emulator, through the same recording proxy, before that
// driver was deleted, so a difference here is a difference the cutover
// introduced. A case omni deliberately changes is named in knownDifferences and
// pinned by an assertion of its own, so the allowlist cannot hide a later
// regression.
//
// The emulator advertises the fixed endpoint 127.0.0.1:8081 (its address
// override), so the recorder must own 8081 and the emulator must publish
// another host port. That is why this test starts its own emulator instead of
// testcontainer.SharedCosmosContainer: the shared fixture publishes 8081, which
// the recorder needs. Run it alone, with no other Cosmos container on the
// machine:
//
//	AZURE_COSMOS_CUTOVER=1 go test -count=1 -run '^TestCutoverParity$' \
//	    -timeout 900s ./backend/plugin/db/cosmosdb/

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	omnidriver "github.com/bytebase/omni/driver"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"

	// Registers the CosmosDB query validator that QueryConn calls.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/cosmosdb"
)

const (
	parityDatabase  = "cutover_parity_db"
	parityContainer = "cutover_parity_c"
	// parityEmulatorAddr is where the emulator publishes its gateway, away from
	// the 8081 the recorder owns.
	parityEmulatorAddr = "https://127.0.0.1:18081"
	// parityRecorderAddr is the endpoint the emulator advertises, so both the
	// SDK and the emulator's REST listings reach the recorder.
	parityRecorderAddr = "https://127.0.0.1:8081"
)

type parityRequest struct {
	Method  string            `json:"method"`
	URI     string            `json:"uri"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

type parityResult struct {
	Statement       string   `json:"statement"`
	ColumnNames     []string `json:"column_names"`
	ColumnTypeNames []string `json:"column_type_names"`
	Rows            []string `json:"rows"`
	Error           string   `json:"error"`
	RowsCount       int64    `json:"rows_count"`
}

type parityQuery struct {
	Name      string          `json:"name"`
	Script    string          `json:"script"`
	Container string          `json:"container"`
	Limit     int             `json:"limit"`
	Requests  []parityRequest `json:"requests"`
	Results   []parityResult  `json:"results"`
	CallError string          `json:"call_error"`
}

type paritySync struct {
	Name     string          `json:"name"`
	Requests []parityRequest `json:"requests"`
	Instance string          `json:"instance"`
	Schema   string          `json:"schema"`
	Error    string          `json:"error"`
}

type parityRecording struct {
	Syncs   []paritySync  `json:"syncs"`
	Queries []parityQuery `json:"queries"`
}

// knownDifferences names the corpus cases whose requests, results or error text
// move with the omni driver, and says what the new path does instead. Each one
// is asserted below.
var knownDifferences = map[string]string{
	"sync instance":         "the omni driver sorts the account's database names; the old driver kept the account's own order",
	"sync missing database": "the omni driver reports driver.ErrNotFound and names the missing database before the listing's error",
	"limit 1500":            "a limit at or above the gateway's 1000-item page is asked for as that page; the old driver named no page size and the gateway chose the same page",
	"empty":                 "the omni driver splits an empty script into no statements; the old driver sent the empty text and reported the gateway's syntax error",
}

// parityRecorder records what a driver sends the emulator, minus headers that
// differ per request (the authorization date and signature, the activity id,
// the continuation token) or per build (the SDK's user agent).
type parityRecorder struct {
	mu       sync.Mutex
	requests []parityRequest
}

func (r *parityRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = nil
}

func (r *parityRecorder) take() []parityRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.requests
	r.requests = nil
	return out
}

var skippedHeaders = map[string]bool{
	"Authorization":                     true,
	"X-Ms-Date":                         true,
	"X-Ms-Activity-Id":                  true,
	"X-Ms-Session-Token":                true,
	"X-Ms-Continuation":                 true,
	"X-Ms-Cosmos-Correlated-Activityid": true,
	"User-Agent":                        true,
	"Content-Length":                    true,
	"Accept-Encoding":                   true,
}

func (r *parityRecorder) record(req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(strings.NewReader(string(body)))
	headers := map[string]string{}
	for name, values := range req.Header {
		name = http.CanonicalHeaderKey(name)
		if skippedHeaders[name] {
			continue
		}
		headers[name] = strings.Join(values, ",")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, parityRequest{
		Method:  req.Method,
		URI:     req.URL.RequestURI(),
		Body:    string(body),
		Headers: headers,
	})
}

// startParityRecorder serves TLS on 127.0.0.1:8081, records each request and
// forwards it to the emulator.
func startParityRecorder(t *testing.T) *parityRecorder {
	t.Helper()
	recorder := &parityRecorder{}
	target, err := url.Parse(parityEmulatorAddr)
	require.NoError(t, err)
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.Out.Host = target.Host
		},
		// The emulator signs its own certificate.
		Transport: &http.Transport{TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, //nolint:gosec
		}},
	}
	listener, err := net.Listen("tcp", "127.0.0.1:8081")
	require.NoError(t, err, "bind the recorder; another Cosmos emulator is probably running")
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		recorder.record(req)
		proxy.ServeHTTP(w, req)
	}))
	_ = server.Listener.Close()
	server.Listener = listener
	server.EnableHTTP2 = false
	server.StartTLS()
	t.Cleanup(server.Close)
	return recorder
}

// startParityEmulator starts the emulator whose gateway the recorder forwards
// to. It publishes 18081 and advertises 127.0.0.1 (the recorder's 8081).
func startParityEmulator(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	port := "8081/tcp"
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:2.14.28",
			ExposedPorts: []string{port},
			Env: map[string]string{
				"AZURE_COSMOS_EMULATOR_PARTITION_COUNT":         "8",
				"AZURE_COSMOS_EMULATOR_ENABLE_DATA_PERSISTENCE": "false",
				"AZURE_COSMOS_EMULATOR_IP_ADDRESS_OVERRIDE":     "127.0.0.1",
			},
			HostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.PortBindings = network.PortMap{network.MustParsePort(port): {{HostPort: "18081"}}}
			},
			WaitingFor: wait.ForHTTP("/_explorer/emulator.pem").
				WithPort(port).
				WithTLS(true, &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}).
				WithStartupTimeout(10 * time.Minute),
		},
		Started: true,
	})
	require.NoError(t, err, "start the Cosmos emulator")
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
}

// seedParityFixture writes the corpus's database, containers and items. The
// corpus queries read two containers: cutover_parity_c holds the ordered rows
// and a second account database proves SyncInstance's ordering.
func seedParityFixture(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	credential, err := azcosmos.NewKeyCredential(cosmosDBEmulatorKey)
	require.NoError(t, err)
	client, err := azcosmos.NewClientWithKey(parityEmulatorAddr, credential, &azcosmos.ClientOptions{
		ClientOptions: azcore.ClientOptions{Transport: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}, //nolint:gosec
		}}},
	})
	require.NoError(t, err)
	// The emulator is started fresh for this test, so only a re-run against a
	// kept container finds the database there already.
	if _, err := client.CreateDatabase(ctx, azcosmos.DatabaseProperties{ID: parityDatabase}, nil); err != nil {
		t.Logf("create the corpus database: %v", err)
	}
	if _, err := client.CreateDatabase(ctx, azcosmos.DatabaseProperties{ID: "aaa-earlier"}, nil); err != nil {
		t.Logf("create the ordering database: %v", err)
	}
	database, err := client.NewDatabase(parityDatabase)
	require.NoError(t, err)
	if _, err := database.CreateContainer(ctx, azcosmos.ContainerProperties{
		ID:                     parityContainer,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{Paths: []string{"/country"}},
	}, nil); err != nil {
		t.Logf("create the corpus container: %v", err)
	}
	container, err := database.NewContainer(parityContainer)
	require.NoError(t, err)
	for i, country := range []string{"AE", "US", "GB"} {
		for j := range 4 {
			item, err := json.Marshal(map[string]any{
				"id":         fmt.Sprintf("%s-%d", country, j),
				"country":    country,
				"name":       fmt.Sprintf("city-%s-%d", country, j),
				"population": (i + 1) * 100 * (j + 1),
			})
			require.NoError(t, err)
			_, err = container.CreateItem(ctx, azcosmos.NewPartitionKeyString(country), item, nil)
			if err != nil {
				t.Logf("seed %s-%d: %v", country, j, err)
			}
		}
	}
}

func openParityDriver(t *testing.T, database string) db.Driver {
	t.Helper()
	driver, err := (&Driver{}).Open(context.Background(), storepb.Engine_COSMOSDB, db.ConnectionConfig{
		DataSource:        &storepb.DataSource{Host: parityRecorderAddr},
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Close(context.Background()) })
	return driver
}

func summarizeParityResults(results []*v1pb.QueryResult) []parityResult {
	out := make([]parityResult, 0, len(results))
	for _, result := range results {
		rows := make([]string, 0, len(result.Rows))
		for _, row := range result.Rows {
			parts := make([]string, 0, len(row.Values))
			for _, value := range row.Values {
				parts = append(parts, normalizeParityRow(value.String()))
			}
			rows = append(rows, strings.Join(parts, "|"))
		}
		out = append(out, parityResult{
			Statement:       result.Statement,
			ColumnNames:     result.ColumnNames,
			ColumnTypeNames: result.ColumnTypeNames,
			Rows:            rows,
			Error:           result.Error,
			RowsCount:       result.RowsCount,
		})
	}
	return out
}

// parityRequests keeps the calls the corpus makes; the recorder also sees the
// emulator's explorer endpoints and the TLS handshake requests none of these
// cases needs.
func parityRequests(requests []parityRequest) []parityRequest {
	out := make([]parityRequest, 0, len(requests))
	for _, request := range requests {
		if strings.HasPrefix(request.URI, "/dbs") {
			out = append(out, request)
		}
	}
	return out
}

// normalizeParityRow drops the document metadata the emulator adds to every
// item it stores (_rid, _self, _etag, _ts), which changes with each emulator
// instance, and renders the JSON object canonically so the recording compares
// what the driver returned rather than the key order the store wrote.
func normalizeParityRow(row string) string {
	const prefix = `string_value:"`
	if !strings.HasPrefix(row, prefix) || !strings.HasSuffix(row, `"`) {
		return row
	}
	text, err := strconv.Unquote(row[len("string_value:"):])
	if err != nil {
		return row
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(text), &object); err != nil {
		return row
	}
	for key := range object {
		if strings.HasPrefix(key, "_") {
			delete(object, key)
		}
	}
	encoded, err := json.Marshal(object)
	if err != nil {
		return row
	}
	quoted := strconv.Quote(string(encoded))
	return prefix + quoted[1:len(quoted)-1] + `"`
}

// normalizeActivityID drops the emulator's per-request activity id, which a
// 404 body carries in the error text the caller sees.
var activityIDPattern = regexp.MustCompile(`ActivityId: [0-9a-fA-F-]+`)

func normalizeActivityID(text string) string {
	return activityIDPattern.ReplaceAllString(text, "ActivityId: <id>")
}

// runParityCorpus runs the corpus against driver, keeping the requests it makes
// and the results it returns for each call.
func runParityCorpus(t *testing.T, driver, missingDriver db.Driver, recorder *parityRecorder) parityRecording {
	t.Helper()
	ctx := context.Background()
	var recording parityRecording

	for _, name := range []string{"sync instance", "sync schema", "ping"} {
		recorder.reset()
		entry := paritySync{Name: name}
		switch name {
		case "sync instance":
			instance, err := driver.SyncInstance(ctx)
			if err != nil {
				entry.Error = err.Error()
			} else {
				encoded, err := json.Marshal(instance)
				require.NoError(t, err)
				entry.Instance = string(encoded)
			}
		case "sync schema":
			schema, err := driver.SyncDBSchema(ctx)
			if err != nil {
				entry.Error = err.Error()
			} else {
				encoded, err := json.Marshal(schema)
				require.NoError(t, err)
				entry.Schema = string(encoded)
			}
		case "ping":
			if err := driver.Ping(ctx); err != nil {
				entry.Error = err.Error()
			}
		default:
			t.Fatalf("unknown sync case %q", name)
		}
		entry.Requests = parityRequests(recorder.take())
		recording.Syncs = append(recording.Syncs, entry)
	}

	// A database the account does not have: the old driver reported the
	// listing's error, and the omni driver reports driver.ErrNotFound.
	recorder.reset()
	_, err := missingDriver.SyncDBSchema(ctx)
	require.Error(t, err, "a missing database must fail the sync")
	missingEntry := paritySync{Name: "sync missing database", Requests: parityRequests(recorder.take()), Error: err.Error()}
	recording.Syncs = append(recording.Syncs, missingEntry)

	for _, query := range parityCorpus {
		recorder.reset()
		results, err := driver.QueryConn(ctx, nil, query.script, db.QueryContext{Container: query.container, Limit: query.limit})
		entry := parityQuery{
			Name:      query.name,
			Script:    query.script,
			Container: query.container,
			Limit:     query.limit,
			Requests:  parityRequests(recorder.take()),
		}
		if err != nil {
			entry.CallError = err.Error()
		}
		entry.Results = summarizeParityResults(results)
		recording.Queries = append(recording.Queries, entry)
	}
	return recording
}

type parityCorpusCase struct {
	name      string
	script    string
	container string
	limit     int
}

// parityCorpus is the cutover's corpus. The golden names the same cases in the
// same order.
var parityCorpus = []parityCorpusCase{
	{name: "select star", script: `SELECT * FROM c`, container: parityContainer},
	{name: "projection", script: `SELECT c.name, c.population FROM c ORDER BY c.name`, container: parityContainer},
	{name: "aggregate", script: `SELECT COUNT(1) AS n, SUM(c.population) AS total FROM c`, container: parityContainer},
	{name: "value aggregate", script: `SELECT VALUE COUNT(1) FROM c`, container: parityContainer},
	{name: "order by", script: `SELECT c.name FROM c ORDER BY c.population DESC`, container: parityContainer},
	{name: "distinct", script: `SELECT DISTINCT c.country FROM c ORDER BY c.country`, container: parityContainer},
	{name: "limit 7", script: `SELECT c.name FROM c ORDER BY c.name`, container: parityContainer, limit: 7},
	{name: "limit 1500", script: `SELECT c.name FROM c ORDER BY c.name`, container: parityContainer, limit: 1500},
	{name: "parse error", script: `SELECT FROM`, container: parityContainer},
	{name: "missing container", script: `SELECT * FROM c`},
	{name: "empty", script: ``, container: parityContainer},
	{name: "trailing semicolon", script: `SELECT c.name FROM c ORDER BY c.name;`, container: parityContainer},
}

func TestCutoverParity(t *testing.T) {
	if os.Getenv("AZURE_COSMOS_CUTOVER") != "1" {
		t.Skip("set AZURE_COSMOS_CUTOVER=1 to run the cutover parity check; it starts an emulator and binds 127.0.0.1:8081")
	}
	startParityEmulator(t)
	recorder := startParityRecorder(t)
	seedParityFixture(t)
	driver := openParityDriver(t, parityDatabase)
	missingDriver := openParityDriver(t, parityDatabase+"_missing")

	goldenBytes, err := os.ReadFile("testdata/cutover_parity.json")
	require.NoError(t, err)
	var golden parityRecording
	require.NoError(t, json.Unmarshal(goldenBytes, &golden))

	got := runParityCorpus(t, driver, missingDriver, recorder)
	require.Len(t, got.Syncs, len(golden.Syncs), "the corpus and its recording must have the same sync cases")
	require.Len(t, got.Queries, len(golden.Queries), "the corpus and its recording must have the same query cases")

	for i, want := range golden.Syncs {
		actual := got.Syncs[i]
		require.Equal(t, want.Name, actual.Name)
		if reason, known := knownDifferences[actual.Name]; known {
			t.Logf("%s: known difference: %s", actual.Name, reason)
			require.Equal(t, want.Requests, actual.Requests, actual.Name)
			switch actual.Name {
			case "sync instance":
				var wantInstance, gotInstance struct {
					Version   string
					Databases []struct{ Name string }
				}
				require.NoError(t, json.Unmarshal([]byte(want.Instance), &wantInstance))
				require.NoError(t, json.Unmarshal([]byte(actual.Instance), &gotInstance))
				wantNames := make([]string, 0, len(wantInstance.Databases))
				for _, database := range wantInstance.Databases {
					wantNames = append(wantNames, database.Name)
				}
				gotNames := make([]string, 0, len(gotInstance.Databases))
				for _, database := range gotInstance.Databases {
					gotNames = append(gotNames, database.Name)
				}
				// The new path names the same databases in sorted order.
				require.Equal(t, sortedCopy(wantNames), gotNames)
				require.Equal(t, wantInstance.Version, gotInstance.Version)
			case "sync missing database":
				prefix := fmt.Sprintf("%v: database %s does not exist: ", omnidriver.ErrNotFound, parityDatabase+"_missing")
				require.True(t, strings.HasPrefix(actual.Error, prefix), "got error %q", actual.Error)
				require.Equal(t, normalizeActivityID(want.Error), normalizeActivityID(strings.TrimPrefix(actual.Error, prefix)))
			default:
				t.Fatalf("no assertion for known difference %q", actual.Name)
			}
			continue
		}
		require.Equal(t, want.Requests, actual.Requests, actual.Name)
		require.Equal(t, want.Instance, actual.Instance, actual.Name)
		require.Equal(t, want.Schema, actual.Schema, actual.Name)
		require.Equal(t, want.Error, actual.Error, actual.Name)
	}

	for i, want := range golden.Queries {
		actual := got.Queries[i]
		require.Equal(t, want.Name, actual.Name)
		require.Equal(t, want.Script, actual.Script)
		if reason, known := knownDifferences[actual.Name]; known {
			t.Logf("%s: known difference: %s", actual.Name, reason)
			switch actual.Name {
			case "limit 1500":
				require.Len(t, actual.Requests, len(want.Requests))
				askedForThePage := false
				for i := range actual.Requests {
					actualRequest := actual.Requests[i]
					wantRequest := want.Requests[i]
					if size := actualRequest.Headers["X-Ms-Max-Item-Count"]; size != "" {
						require.Equal(t, "1000", size)
						askedForThePage = true
						delete(actualRequest.Headers, "X-Ms-Max-Item-Count")
					}
					require.Equal(t, wantRequest.Headers, actualRequest.Headers, "headers of request %d", i)
				}
				require.True(t, askedForThePage, "the page the limit asks for is named on at least one request")
			case "empty":
				require.Empty(t, actual.Requests)
				require.Empty(t, actual.Results)
				require.Empty(t, actual.CallError)
				continue
			default:
				t.Fatalf("no assertion for known difference %q", actual.Name)
			}
			// The rows are unchanged by the page size: the gateway serves that
			// page without being asked for it.
			require.Equal(t, want.Results, actual.Results)
			require.Equal(t, want.CallError, actual.CallError)
			continue
		}
		require.Equal(t, want.Requests, actual.Requests, actual.Name)
		require.Equal(t, want.Results, actual.Results, actual.Name)
		require.Equal(t, want.CallError, actual.CallError, actual.Name)
	}
}

func sortedCopy(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)
	return out
}
