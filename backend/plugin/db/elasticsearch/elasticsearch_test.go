package elasticsearch

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	dbutil "github.com/bytebase/bytebase/backend/plugin/db/util"
)

// testServer is one Elasticsearch endpoint answering from a table of routes. An
// answer starting with "!" is answered with a 500, one starting with "text:" as
// text/plain, and one starting with "png:" as an image, so a test can pin how
// each is read.
type testServer struct {
	*httptest.Server
	requests []string
	routes   map[string]string
}

func newTestServer(t *testing.T, routes map[string]string) *testServer {
	t.Helper()
	s := &testServer{routes: routes}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := r.URL.Path
		if r.URL.RawQuery != "" {
			route += "?" + r.URL.RawQuery
		}
		request := r.Method + " " + route
		s.requests = append(s.requests, request)
		answer, ok := s.routes[request]
		if !ok {
			answer = s.routes["*"]
		}
		status := http.StatusOK
		if strings.HasPrefix(answer, "!") {
			status = http.StatusInternalServerError
			answer = strings.TrimPrefix(answer, "!")
		}
		contentType := "application/json"
		if strings.HasPrefix(answer, "text:") {
			contentType = "text/plain"
			answer = strings.TrimPrefix(answer, "text:")
		}
		if strings.HasPrefix(answer, "png:") {
			contentType = "image/png"
			answer = strings.TrimPrefix(answer, "png:")
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(s.Close)
	return s
}

// configFor returns the connection config that reaches s the way a data source
// is reached: host and port, so the driver builds the endpoint itself.
func (s *testServer) configFor(t *testing.T) db.ConnectionConfig {
	t.Helper()
	parsed, err := url.Parse(s.URL)
	require.NoError(t, err)
	return db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     parsed.Hostname(),
			Port:     parsed.Port(),
			Username: "elastic",
		},
		Password: "password123",
	}
}

func (s *testServer) open(t *testing.T) db.Driver {
	t.Helper()
	opened, err := (&Driver{}).Open(context.Background(), storepb.Engine_ELASTICSEARCH, s.configFor(t))
	require.NoError(t, err)
	return opened
}

func TestOpenWithAWSAuth(t *testing.T) {
	for _, tc := range []struct {
		name    string
		config  db.ConnectionConfig
		wantErr string
	}{
		{
			name: "missing region",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:               "search-test.us-east-1.es.amazonaws.com",
					Port:               "443",
					AuthenticationType: storepb.DataSource_AWS_RDS_IAM,
				},
			},
			wantErr: "region is required for AWS IAM authentication",
		},
		{
			// Open reaches nothing, so a basic-auth login opens even though no
			// endpoint answers it.
			name: "basic auth",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:               "localhost",
					Port:               "9200",
					Username:           "elastic",
					AuthenticationType: storepb.DataSource_PASSWORD,
				},
				Password: "password123",
			},
		},
		{
			// https:// plus use_ssl=false names an https endpoint, so TLS is
			// what the driver negotiates.
			name: "a host that names https",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "https://localhost",
					Port:     "9200",
					Username: "elastic",
				},
				Password: "password123",
			},
		},
		{
			// A host that names http:// contradicts use_ssl, and the driver
			// refuses rather than quietly sending credentials in the clear.
			name: "a host that contradicts use_ssl",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "http://localhost",
					Port:     "9200",
					Username: "elastic",
					UseSsl:   true,
				},
				Password: "password123",
			},
			wantErr: "but TLS is required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := (&Driver{}).Open(context.Background(), storepb.Engine_ELASTICSEARCH, tc.config)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestOpenWithAnSSHHost pins that the tunnel is the path: Open dials the jump
// host, where the driver this replaced ignored it and reached the endpoint
// directly.
func TestOpenWithAnSSHHost(t *testing.T) {
	_, err := (&Driver{}).Open(context.Background(), storepb.Engine_ELASTICSEARCH, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "9200",
			Username: "elastic",
			SshHost:  "127.0.0.1",
			SshPort:  "1",
		},
		Password: "password123",
	})
	require.ErrorContains(t, err, "127.0.0.1:1")
}

func TestTLSSettings(t *testing.T) {
	t.Run("use_ssl with no material keeps the old default", func(t *testing.T) {
		// The old driver encrypted without verifying when it was asked for
		// https and given nothing to verify with.
		tlsConfig, err := tlsSettings(&storepb.DataSource{UseSsl: true})
		require.NoError(t, err)
		require.True(t, tlsConfig.InsecureSkipVerify)
		require.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
	})

	t.Run("a CA certificate becomes the endpoint's roots", func(t *testing.T) {
		certificate := serverCertificatePEM(t)
		// The old client added the CA whether or not verification was on,
		// which is where the roots come from here too.
		tlsConfig, err := tlsSettings(&storepb.DataSource{UseSsl: true, SslCa: certificate})
		require.NoError(t, err)
		require.NotNil(t, tlsConfig.RootCAs)
		require.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
	})

	t.Run("a CA that is not a certificate is an error", func(t *testing.T) {
		_, err := tlsSettings(&storepb.DataSource{UseSsl: true, SslCa: "not a certificate"})
		require.ErrorContains(t, err, "failed to add CA certificate")
	})
}

// serverCertificatePEM returns a certificate the root pool accepts.
func serverCertificatePEM(t *testing.T) string {
	t.Helper()
	server := httptest.NewTLSServer(http.NotFoundHandler())
	defer server.Close()
	certificate := server.Certificate()
	require.NotNil(t, certificate)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw}))
}

// TestSyncInstanceMatchesRecordedAnswer replays the answer the pinned
// Elasticsearch 8.19.25 server gave the driver this replaced, and requires the
// adapter to read it the same way: the same version, the same one database, the
// same indices with their document counts and sizes, and the same users.
func TestSyncInstanceMatchesRecordedAnswer(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"GET /": `{"version":{"number":"8.19.25"}}`,
		// hidden-one is hidden and stays out of the metadata; closed-index
		// answers no count and no size, which is what a closed index reports.
		"GET /*/_settings?filter_path=**.hidden&expand_wildcards=open,hidden": `{
			"alpha":{"settings":{"index":{"hidden":"false"}}},
			"beta":{"settings":{"index":{"hidden":"false"}}},
			"hidden-one":{"settings":{"index":{"hidden":"true"}}},
			"closed-index":{"settings":{"index":{}}}
		}`,
		// The cat API's order is the server's, not the metadata's.
		"GET /_cat/indices?format=json&pretty": `[
			{"index":"beta","docs.count":"2","store.size":"227b"},
			{"index":"alpha","docs.count":"4","store.size":"1.5kb"},
			{"index":"hidden-one","docs.count":"1","store.size":"1kb"},
			{"index":"closed-index","docs.count":null,"store.size":null}
		]`,
		// The users arrive as an object keyed by name.
		"GET /_security/user": `{
			"elastic":{"roles":["superuser"]},
			"u1":{"roles":["r1","r2"]}
		}`,
	})

	opened := server.open(t)
	instance, err := opened.SyncInstance(context.Background())
	require.NoError(t, err)

	require.Equal(t, "8.19.25", instance.Version)
	require.Equal(t, []*metadatapb.DatabaseSchemaMetadata{{
		Name: "node",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{
				{Name: "alpha", RowCount: 4, DataSize: 1536},
				{Name: "beta", RowCount: 2, DataSize: 227},
				{Name: "closed-index"},
			},
		}},
	}}, instance.Databases)

	u1Role := "[r1, r2]"
	elasticRole := "[superuser]"
	require.Equal(t, []*storepb.InstanceRole{
		{Name: "elastic", Attribute: &elasticRole},
		{Name: "u1", Attribute: &u1Role},
	}, instance.Metadata.GetRoles())

	// The statements are the ones this driver issues, in this order.
	require.Equal(t, []string{
		"GET /",
		"GET /*/_settings?filter_path=**.hidden&expand_wildcards=open,hidden",
		"GET /_cat/indices?format=json&pretty",
		"GET /_security/user",
	}, server.requests)

	schema, err := opened.SyncDBSchema(context.Background())
	require.NoError(t, err)
	require.True(t, proto.Equal(instance.Databases[0], schema), "SyncDBSchema and SyncInstance read the same database")
}

// TestSyncInstanceFailsOpenOnUsers pins that a user list the endpoint refuses is
// not a sync failure: most deployments have no internal users.
func TestSyncInstanceFailsOpenOnUsers(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"GET /": `{"version":{"number":"8.19.25"}}`,
		"GET /*/_settings?filter_path=**.hidden&expand_wildcards=open,hidden": `{}`,
		"GET /_cat/indices?format=json&pretty":                                `[]`,
		"GET /_security/user":                                                 "!{}",
	})
	instance, err := server.open(t).SyncInstance(context.Background())
	require.NoError(t, err)
	require.Empty(t, instance.Metadata.GetRoles())
	require.Equal(t, "8.19.25", instance.Version)
}

func TestQueryConnReadsBothAnswerShapes(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"POST /alpha/_search": `{
			"took": 5,
			"timed_out": false,
			"hits": {"total": {"value": 2, "relation": "eq"}, "max_score": 1.0},
			"_shards": {"total": 1, "successful": 1, "failed": 0}
		}`,
		"GET /_cat/indices?format=json": `[{"index":"alpha","docs.count":"2"}]`,
		"GET /_cat/indices?h=index":     "text:alpha",
	})

	results, err := server.open(t).QueryConn(context.Background(), nil, `
POST /alpha/_search
{"query": {"match_all": {}}}

GET /_cat/indices?format=json

GET /_cat/indices?h=index`, db.QueryContext{})
	require.NoError(t, err)
	require.Len(t, results, 3)

	// An object answer is one column per key, in name order, one row, and the
	// numbers keep the digits the endpoint wrote: max_score stays 1.0, where
	// the driver this replaced decoded through float64 and printed 1.
	search := results[0]
	require.Equal(t, []string{"_shards", "hits", "timed_out", "took"}, search.ColumnNames)
	require.Empty(t, search.ColumnTypeNames, "Elasticsearch states no column type")
	require.Len(t, search.Rows, 1)
	require.Equal(t, int64(1), search.RowsCount)
	require.Equal(t, `{"failed":0,"successful":1,"total":1}`, search.Rows[0].Values[0].GetStringValue())
	require.Contains(t, search.Rows[0].Values[1].GetStringValue(), `"max_score":1.0`)
	require.Equal(t, "false", search.Rows[0].Values[2].GetStringValue())
	require.Equal(t, "5", search.Rows[0].Values[3].GetStringValue())
	require.Nil(t, search.GetQueryPlan())
	require.Equal(t, "POST /alpha/_search\n{\"query\": {\"match_all\": {}}}\n", search.Statement)

	// An array answer is one column holding it.
	array := results[1]
	require.Equal(t, []string{"result"}, array.ColumnNames)
	require.Equal(t, `[{"docs.count":"2","index":"alpha"}]`, array.Rows[0].Values[0].GetStringValue())

	// A text answer is the body as written.
	text := results[2]
	require.Equal(t, []string{"text/plain"}, text.ColumnNames)
	require.Equal(t, "alpha", text.Rows[0].Values[0].GetStringValue())

	// The requests are the caller's, query strings and all.
	require.Equal(t, []string{
		"POST /alpha/_search",
		"GET /_cat/indices?format=json",
		"GET /_cat/indices?h=index",
	}, server.requests)
}

// TestQueryConnFailsTheCall pins how a request that fails answers: the call
// fails with the endpoint's own status and body, as it did before. Earlier
// results of the same script do not come back.
func TestQueryConnFailsTheCall(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"GET /alpha/_doc/1": `{}`,
		"*":                 `!{"error":{"type":"index_not_found_exception"}}`,
	})
	results, err := server.open(t).QueryConn(context.Background(), nil, `
GET /alpha/_doc/1
GET /nope/_search`, db.QueryContext{})
	require.Nil(t, results)
	require.ErrorContains(t, err, "request failed with status code 500")
	require.ErrorContains(t, err, "index_not_found_exception")
}

// TestQueryConnEmptyAnswer pins an answer with no body: a HEAD, or an empty
// 200. The status is the whole verdict, so it is a result with no columns and no
// rows; the driver this replaced failed the call trying to parse it.
func TestQueryConnEmptyAnswer(t *testing.T) {
	server := newTestServer(t, map[string]string{"HEAD /alpha": ""})
	results, err := server.open(t).QueryConn(context.Background(), nil, "HEAD /alpha", db.QueryContext{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].ColumnNames)
	require.Empty(t, results[0].Rows)
	require.Equal(t, "HEAD /alpha\n", results[0].Statement)
}

// TestQueryConnUnsupportedContentType pins that an answer whose type the driver
// cannot read is an error naming the type.
func TestQueryConnUnsupportedContentType(t *testing.T) {
	server := newTestServer(t, map[string]string{"GET /": "png:not json"})
	results, err := server.open(t).QueryConn(context.Background(), nil, "GET /", db.QueryContext{})
	require.Nil(t, results)
	require.ErrorContains(t, err, "Content-Type not supported: image/png")
}

// TestQueryConnTruncatesAtTheCap pins the result-size cap: a REST answer is one
// row, so the row that crosses the cap is kept and the result says it is
// truncated, with the message every driver uses.
func TestQueryConnTruncatesAtTheCap(t *testing.T) {
	server := newTestServer(t, map[string]string{"GET /alpha/_search": `{"took": 1}`})
	// One byte is smaller than any answer's columns, so the row crosses the cap.
	const limit = int64(1)
	results, err := server.open(t).QueryConn(context.Background(), nil, "GET /alpha/_search", db.QueryContext{MaximumSQLResultSize: limit})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1, "the row that crossed the cap is kept")
	require.Equal(t, dbutil.FormatMaximumSQLResultSizeMessage(limit), results[0].Error)
}

// TestQueryConnBulkFailure pins that a bulk request the endpoint accepted but
// did not fully apply fails: the verdict is in the body, which the driver this
// replaced read past.
func TestQueryConnBulkFailure(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"POST /alpha/_bulk": `{
			"took": 1,
			"errors": true,
			"items": [
				{"index": {"_index": "alpha", "_id": "10", "status": 201}},
				{"update": {"_index": "alpha", "_id": "missing", "status": 404, "error": {"type": "document_missing_exception", "reason": "[missing]: document missing"}}}
			]
		}`,
	})
	results, err := server.open(t).QueryConn(context.Background(), nil, `
POST /alpha/_bulk
{"index":{"_id":"10"}}
{"name":"row 10"}
{"update":{"_id":"missing"}}
{"doc":{"name":"x"}}`, db.QueryContext{})
	require.Nil(t, results)
	require.ErrorContains(t, err, "1 of 2 bulk items failed: [missing]: document missing")
}

// TestExplainProfilesTheSearch holds an explain to what it is: the search runs
// with profiling on, the profile is the only column, the hits do not come back,
// and the result says it is a JSON plan that ran.
func TestExplainProfilesTheSearch(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"POST /alpha/_search": `{
			"took": 3,
			"hits": {"total": {"value": 1}, "hits": [{"_source": {"ssn": "111-22-3333"}}]},
			"profile": {"shards": [{"id": "[node][alpha][0]"}]}
		}`,
	})
	results, err := server.open(t).QueryConn(context.Background(), nil, `
POST /alpha/_search
{"query": {"match_all": {}}}`, db.QueryContext{Explain: true})
	require.NoError(t, err)
	require.Len(t, results, 1)

	result := results[0]
	require.Equal(t, []string{"profile"}, result.ColumnNames)
	require.Len(t, result.Rows, 1)
	profile := result.Rows[0].Values[0].GetStringValue()
	require.Contains(t, profile, `"profile"`)
	require.Contains(t, profile, `"took":3`)
	require.NotContains(t, profile, "ssn", "the hits are not part of a plan")
	require.Equal(t, &v1pb.QueryResult_QueryPlan{Format: v1pb.QueryOption_JSON, Executed: true}, result.GetQueryPlan())
	require.Contains(t, result.Statement, `"profile":true`)

	// The search that ran is the caller's, with profiling on and nothing else
	// changed.
	require.Equal(t, []string{"POST /alpha/_search"}, server.requests)
}

// TestExplainRefusesWhatIsNotASearch pins the line between explaining and doing:
// profiling runs the request, so anything that is not a search is refused and
// never sent.
func TestExplainRefusesWhatIsNotASearch(t *testing.T) {
	server := newTestServer(t, map[string]string{"*": `{}`})
	for _, script := range []string{
		"PUT /alpha/_doc/1\n{\"name\":\"row 1\"}",
		"DELETE /alpha",
		"POST /alpha/_delete_by_query\n{\"query\":{\"match_all\":{}}}",
		"POST /_bulk\n{\"index\":{}}\n{\"name\":\"row 1\"}",
		"POST /alpha/_msearch\n{}\n{\"query\":{\"match_all\":{}}}",
		"POST /alpha/_doc/_search\n{\"id\":1}",
	} {
		results, err := server.open(t).QueryConn(context.Background(), nil, script, db.QueryContext{Explain: true})
		require.Nil(t, results, script)
		require.ErrorContains(t, err, "is not a search to explain", script)
	}
	require.Empty(t, server.requests, "an explain of a write sent nothing")
}

// TestExplainWithoutAProfile pins that a search the endpoint did not profile is
// an error, not a plan made of the search's own hits.
func TestExplainWithoutAProfile(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"POST /alpha/_search": `{"took": 1, "hits": {"total": {"value": 0}}}`,
	})
	results, err := server.open(t).QueryConn(context.Background(), nil, "POST /alpha/_search\n{}", db.QueryContext{Explain: true})
	require.Nil(t, results)
	require.ErrorContains(t, err, "the search answered with no profile to explain")
}

// TestExecuteReportsTheScriptSteps pins the task run's log: one command per
// request with its text, and one response, and 0 affected rows because an
// Elasticsearch answer states its counts in its own columns.
func TestExecuteReportsTheScriptSteps(t *testing.T) {
	const script = "\nPOST /alpha/_bulk\n{\"index\":{}}\n{\"name\":\"row 1\"}"
	server := newTestServer(t, map[string]string{
		"POST /alpha/_bulk": `{"took": 1, "errors": false, "items": [{"index": {"status": 201}}]}`,
	})
	var logs []*storepb.TaskRunLog
	affected, err := server.open(t).Execute(context.Background(), script, db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), affected)

	var types []storepb.TaskRunLog_Type
	for _, entry := range logs {
		types = append(types, entry.Type)
	}
	require.Equal(t, []storepb.TaskRunLog_Type{
		storepb.TaskRunLog_COMMAND_EXECUTE,
		storepb.TaskRunLog_COMMAND_RESPONSE,
	}, types)
	command := logs[0].GetCommandExecute()
	require.Equal(t, "POST /alpha/_bulk\n{\"index\":{}}\n{\"name\":\"row 1\"}", command.Statement)
	require.Empty(t, logs[1].GetCommandResponse().GetError())
}

// TestExecuteFailsOnARefusedRequest pins that a request the endpoint refuses
// ends the script with its error and reports nothing as affected.
func TestExecuteFailsOnARefusedRequest(t *testing.T) {
	server := newTestServer(t, map[string]string{"*": `!{"error":"refused"}`})
	affected, err := server.open(t).Execute(context.Background(), "POST /alpha/_bulk\n{\"index\":{}}", db.ExecuteOptions{})
	require.Equal(t, int64(0), affected)
	require.ErrorContains(t, err, "request failed with status code 500")
}

// TestPingKeepsTheAWSHints pins the connection test's error text on a managed
// OpenSearch domain: a support case reads the hint that names the role which
// could not be assumed.
func TestPingKeepsTheAWSHints(t *testing.T) {
	for _, tc := range []struct {
		name   string
		driver *Driver
		inner  string
		want   string
	}{
		{
			name:   "a role that could not be assumed",
			driver: &Driver{aws: true, awsRoleARN: "arn:aws:iam::123456789012:role/search"},
			inner:  "ping failed: [403 Forbidden] {\"message\":\"no permissions\"}",
			want:   "authentication failed: unable to assume role arn:aws:iam::123456789012:role/search: ping failed: [403 Forbidden] {\"message\":\"no permissions\"}",
		},
		{
			name:   "no role configured",
			driver: &Driver{aws: true},
			inner:  "ping failed: [401 Unauthorized] {}",
			want:   "authentication failed (consider using cross-account role if accessing different AWS account): ping failed: [401 Unauthorized] {}",
		},
		{
			name:   "an endpoint that is not there",
			driver: &Driver{aws: true},
			inner:  "ping failed: [404 Not Found] {}",
			want:   "endpoint not found (check if path is correct): ping failed: [404 Not Found] {}",
		},
		{
			name:   "anything else",
			driver: &Driver{aws: true},
			inner:  "ping failed: [500 Internal Server Error] {}",
			want:   "failed to connect to OpenSearch: ping failed: [500 Internal Server Error] {}",
		},
		{
			// An unmanaged endpoint keeps the driver's own text.
			name:   "not a managed domain",
			driver: &Driver{},
			inner:  "ping failed: [403 Forbidden] {}",
			want:   "ping failed: [403 Forbidden] {}",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.driver.connectionError(errors.New(tc.inner)).Error())
		})
	}
}

// TestDumpWritesNothing pins the no-op: a synced index does not carry what
// recreates it, so there is nothing to write and nothing to refuse.
func TestDumpWritesNothing(t *testing.T) {
	server := newTestServer(t, map[string]string{})
	var out strings.Builder
	require.NoError(t, server.open(t).Dump(context.Background(), &out, &metadatapb.DatabaseSchemaMetadata{Name: "node"}))
	require.Empty(t, out.String())
	require.Empty(t, server.requests)
}

// TestGetDB pins that there is no database/sql handle to hand out.
func TestGetDB(t *testing.T) {
	require.Nil(t, (&Driver{}).GetDB())
}

// TestToRowValue pins the mapping of a contract value: every answer this driver
// reads is text.
func TestToRowValue(t *testing.T) {
	require.Equal(t, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "alpha"}},
		toRowValue(omnidriver.Value{Kind: omnidriver.String, String: "alpha"}))
	require.Equal(t, dbutil.NullRowValue, toRowValue(omnidriver.Value{Kind: omnidriver.Null}))
}

// TestToQueryPlan pins the plan mapping: the format crosses as its enum name.
func TestToQueryPlan(t *testing.T) {
	require.Nil(t, toQueryPlan(nil))
	require.Equal(t, &v1pb.QueryResult_QueryPlan{Format: v1pb.QueryOption_JSON, Executed: true},
		toQueryPlan(&omnidriver.Plan{Format: "JSON", Executed: true}))
	require.Equal(t, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED,
		toQueryPlan(&omnidriver.Plan{Format: "not a format"}).GetFormat())
}
