package dynamodb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/partiql"
)

// The differential test runs one corpus through db.Driver and records what the
// endpoint was asked and what the product received. The golden file is the
// contract this adapter keeps. Re-record with:
//
//	RECORD_DYNAMODB_GOLDEN=1 go test -run TestDriverDifferential ./backend/plugin/db/dynamodb/
//
// The golden excludes cases whose contract is asserted directly in
// TestContractBehavior: Limit, MaximumSQLResultSize truncation, binary values, a
// missing attribute, and secondary-index sync.
func TestDriverDifferential(t *testing.T) {
	golden := differentialGolden{
		Success: runDifferentialCorpus(t, stsAnswers),
		Refused: runIdentityOnly(t, stsRefuses),
		EmptyID: runIdentityOnly(t, stsEmpty),
	}

	if os.Getenv("RECORD_DYNAMODB_GOLDEN") == "1" {
		encoded, err := json.MarshalIndent(golden, "", "  ")
		if err != nil {
			t.Fatalf("marshal golden: %v", err)
		}
		if err := os.WriteFile(filepath.Join("testdata", "driver_golden.json"), append(encoded, '\n'), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	recorded, err := os.ReadFile(filepath.Join("testdata", "driver_golden.json"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var want differentialGolden
	if err := json.Unmarshal(recorded, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}
	if diff := compareGolden(golden, want); diff != "" {
		t.Errorf("the adapter changed the driver's statements or results:\n%s", diff)
	}
}

type differentialGolden struct {
	Success observation `json:"success"`
	Refused observation `json:"refused"`
	EmptyID observation `json:"emptyIdentity"`
}

type observation struct {
	Requests     []string          `json:"requests"`
	STSRequests  []string          `json:"stsRequests,omitempty"`
	Queries      []json.RawMessage `json:"queries,omitempty"`
	QueryErrors  []string          `json:"queryErrors,omitempty"`
	ExecuteRows  int64             `json:"executeRows,omitempty"`
	ExecuteLog   []json.RawMessage `json:"executeLog,omitempty"`
	ExecuteError string            `json:"executeError,omitempty"`
	Instance     json.RawMessage   `json:"instance,omitempty"`
	InstanceErr  string            `json:"instanceError,omitempty"`
	Schema       json.RawMessage   `json:"schema,omitempty"`
	SchemaErr    string            `json:"schemaError,omitempty"`
}

func compareGolden(got, want differentialGolden) string {
	gotJSON, _ := json.MarshalIndent(got, "", "  ")
	wantJSON, _ := json.MarshalIndent(want, "", "  ")
	if bytes.Equal(gotJSON, wantJSON) {
		return ""
	}
	return fmt.Sprintf("got:\n%s\nwant:\n%s", gotJSON, wantJSON)
}

func runDifferentialCorpus(t *testing.T, sts stsMode) observation {
	t.Helper()
	driver, fake, stsLog := openDifferentialDriver(t, sts)
	ctx := context.Background()
	var obs observation

	if err := driver.Ping(ctx); err != nil {
		t.Fatalf("Ping() = %v", err)
	}

	results, err := driver.QueryConn(ctx, nil, `SELECT * FROM "users"; SELECT * FROM "empty";`, db.QueryContext{})
	if err != nil {
		obs.QueryErrors = append(obs.QueryErrors, err.Error())
	}
	obs.Queries = append(obs.Queries, marshalResults(t, results)...)

	if _, err := driver.QueryConn(ctx, nil, `SELECT * FROM "users"`, db.QueryContext{Explain: true}); err != nil {
		obs.QueryErrors = append(obs.QueryErrors, err.Error())
	}

	results, err = driver.QueryConn(ctx, nil, `SELECT * FROM "fail"`, db.QueryContext{})
	if err != nil {
		obs.QueryErrors = append(obs.QueryErrors, err.Error())
	}
	obs.Queries = append(obs.Queries, marshalResults(t, results)...)

	var logs []*storepb.TaskRunLog
	count, err := driver.Execute(ctx, `INSERT INTO "users" VALUE {'pk': 'a'}; UPDATE "users" SET name='b';`, db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		},
	})
	obs.ExecuteRows = count
	obs.ExecuteError = testErrorString(err)
	obs.ExecuteLog = marshalMessages(t, logs)

	logs = nil
	_, err = driver.Execute(ctx, `INSERT INTO "users" VALUE {'pk': 'a'}; INSERT INTO "fail" VALUE {'pk': 'b'}; INSERT INTO "users" VALUE {'pk': 'c'};`, db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		},
	})
	obs.ExecuteError = testErrorString(err)
	obs.ExecuteLog = append(obs.ExecuteLog, marshalMessages(t, logs)...)

	instance, err := driver.SyncInstance(ctx)
	if err != nil {
		obs.InstanceErr = err.Error()
	} else {
		obs.Instance = marshalInstance(t, instance)
	}

	schema, err := driver.SyncDBSchema(ctx)
	if err != nil {
		obs.SchemaErr = err.Error()
	} else {
		obs.Schema = marshalMessage(t, schema)
	}

	obs.Requests = fake.requests
	obs.STSRequests = stsLog.requests
	return obs
}

func runIdentityOnly(t *testing.T, sts stsMode) observation {
	t.Helper()
	driver, fake, stsLog := openDifferentialDriver(t, sts)
	var obs observation
	instance, err := driver.SyncInstance(context.Background())
	if err != nil {
		obs.InstanceErr = err.Error()
	} else {
		obs.Instance = marshalInstance(t, instance)
	}
	obs.Requests = fake.requests
	obs.STSRequests = stsLog.requests
	return obs
}

type stsMode int

const (
	stsAnswers stsMode = iota
	stsRefuses
	stsEmpty
)

func openDifferentialDriver(t *testing.T, sts stsMode) (*Driver, *fakeDynamoDB, *fakeSTS) {
	t.Helper()
	fake := newFakeDynamoDB(t)
	stsServer, stsLog := newFakeSTS(t, sts)
	setHermeticAWSEnv(t)
	t.Setenv("AWS_ENDPOINT_URL_STS", stsServer.URL)

	ds := &storepb.DataSource{
		Host:   fake.host,
		Port:   fake.port,
		Region: "us-east-1",
		IamExtension: &storepb.DataSource_AwsCredential{
			AwsCredential: &storepb.DataSource_AWSCredential{
				AccessKeyId:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
		},
	}
	inner := newDriver()
	driver, err := inner.Open(context.Background(), storepb.Engine_DYNAMODB, db.ConnectionConfig{
		ConnectionContext: db.ConnectionContext{DatabaseName: "123456789012-us-east-1"},
		DataSource:        ds,
	})
	if err != nil {
		t.Fatalf("Open() = %v", err)
	}
	dynamo, ok := driver.(*Driver)
	if !ok {
		t.Fatalf("Open() returned %T, want *Driver", driver)
	}
	t.Cleanup(func() {
		_ = driver.Close(context.Background())
		fake.server.Close()
		stsServer.Close()
	})
	return dynamo, fake, stsLog
}

func marshalResults(t *testing.T, results []*v1pb.QueryResult) []json.RawMessage {
	t.Helper()
	out := make([]json.RawMessage, 0, len(results))
	for _, result := range results {
		cloned, ok := proto.Clone(result).(*v1pb.QueryResult)
		if !ok {
			t.Fatalf("clone %T", result)
		}
		// Latency is the endpoint's answer time, not the driver's behavior.
		cloned.Latency = nil
		out = append(out, marshalMessage(t, cloned))
	}
	return out
}

// normalizeRequest removes the SDK's random idempotency token, which differs
// per call and is not part of what the driver chose to send.
func normalizeRequest(body []byte) string {
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return string(body)
	}
	delete(object, "ClientRequestToken")
	normalized, err := json.Marshal(object)
	if err != nil {
		return string(body)
	}
	return string(normalized)
}

func marshalMessages(t *testing.T, messages []*storepb.TaskRunLog) []json.RawMessage {
	t.Helper()
	out := make([]json.RawMessage, 0, len(messages))
	for _, message := range messages {
		out = append(out, marshalMessage(t, message))
	}
	return out
}

func marshalInstance(t *testing.T, instance *db.InstanceMetadata) json.RawMessage {
	t.Helper()
	databases := make([]json.RawMessage, 0, len(instance.Databases))
	for _, database := range instance.Databases {
		databases = append(databases, marshalMessage(t, database))
	}
	encoded, err := json.Marshal(struct {
		Version   string            `json:"version"`
		Databases []json.RawMessage `json:"databases"`
	}{Version: instance.Version, Databases: databases})
	if err != nil {
		t.Fatalf("marshal instance: %v", err)
	}
	return encoded
}

func marshalMessage(t *testing.T, message proto.Message) json.RawMessage {
	t.Helper()
	encoded, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(message)
	if err != nil {
		t.Fatalf("marshal %T: %v", message, err)
	}
	return encoded
}

type fakeDynamoDB struct {
	t        *testing.T
	server   *httptest.Server
	host     string
	port     string
	requests []string
	tables   []string
	paged    bool
}

func newFakeDynamoDB(t *testing.T) *fakeDynamoDB {
	t.Helper()
	fake := &fakeDynamoDB{t: t, tables: []string{"users"}, paged: true}
	fake.server = httptest.NewServer(http.HandlerFunc(fake.serve))
	host, port := splitHostPort(t, fake.server.URL)
	fake.host, fake.port = host, port
	return fake
}

func (f *fakeDynamoDB) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.requests = append(f.requests, r.Header.Get("X-Amz-Target")+" "+normalizeRequest(body))
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.Header().Set("x-amzn-requestid", "test-request-id")
	target := strings.TrimPrefix(r.Header.Get("X-Amz-Target"), "DynamoDB_20120810.")
	switch target {
	case "ListTables":
		var request struct {
			Limit                   int32  `json:"Limit"`
			ExclusiveStartTableName string `json:"ExclusiveStartTableName"`
		}
		_ = json.Unmarshal(body, &request)
		if request.Limit <= 1 {
			writeJSON(w, map[string]any{"TableNames": f.tables[:1]})
			return
		}
		if !f.paged {
			writeJSON(w, map[string]any{"TableNames": f.tables})
			return
		}
		if request.ExclusiveStartTableName == "" {
			writeJSON(w, map[string]any{"TableNames": f.tables[:1], "LastEvaluatedTableName": f.tables[0]})
			return
		}
		writeJSON(w, map[string]any{"TableNames": []string{}})
	case "DescribeTable":
		var request struct {
			TableName string `json:"TableName"`
		}
		_ = json.Unmarshal(body, &request)
		switch request.TableName {
		case "users":
			writeJSON(w, map[string]any{"Table": map[string]any{
				"TableName":   "users",
				"TableStatus": "ACTIVE",
				"ItemCount":   3,
				"KeySchema": []map[string]any{
					{"AttributeName": "pk", "KeyType": "HASH"},
					{"AttributeName": "sk", "KeyType": "RANGE"},
				},
			}})
		case "orders":
			writeJSON(w, map[string]any{"Table": map[string]any{
				"TableName":   "orders",
				"TableStatus": "ACTIVE",
				"ItemCount":   1,
				"KeySchema": []map[string]any{
					{"AttributeName": "pk", "KeyType": "HASH"},
				},
				"GlobalSecondaryIndexes": []map[string]any{{
					"IndexName": "orders_by_sku",
					"KeySchema": []map[string]any{{"AttributeName": "sku", "KeyType": "HASH"}},
				}},
				"LocalSecondaryIndexes": []map[string]any{{
					"IndexName": "orders_by_created",
					"KeySchema": []map[string]any{
						{"AttributeName": "pk", "KeyType": "HASH"},
						{"AttributeName": "created", "KeyType": "RANGE"},
					},
				}},
			}})
		default:
			writeDynamoError(w, "ResourceNotFoundException", "table not found: "+request.TableName)
		}
	case "ExecuteStatement":
		var request struct {
			Statement string `json:"Statement"`
			NextToken string `json:"NextToken"`
		}
		_ = json.Unmarshal(body, &request)
		if strings.Contains(request.Statement, `"fail"`) {
			writeDynamoError(w, "ValidationException", "statement failed")
			return
		}
		if strings.Contains(request.Statement, `"empty"`) {
			writeJSON(w, map[string]any{"Items": []any{}})
			return
		}
		if strings.Contains(request.Statement, `"limit"`) {
			first := map[string]any{"pk": map[string]any{"S": "a"}}
			second := map[string]any{"pk": map[string]any{"S": "b"}}
			third := map[string]any{"pk": map[string]any{"S": "c"}}
			switch request.NextToken {
			case "":
				writeJSON(w, map[string]any{"Items": []any{first}, "NextToken": "limit2"})
			case "limit2":
				writeJSON(w, map[string]any{"Items": []any{second}, "NextToken": "limit3"})
			default:
				writeJSON(w, map[string]any{"Items": []any{third}})
			}
			return
		}
		if strings.Contains(request.Statement, `"big"`) {
			writeJSON(w, map[string]any{"Items": []any{map[string]any{"pk": map[string]any{"S": strings.Repeat("x", 100)}}}})
			return
		}
		if strings.Contains(request.Statement, `"binary"`) {
			writeJSON(w, map[string]any{"Items": []any{map[string]any{"blob": map[string]any{"B": "AAEC"}}}})
			return
		}
		if strings.Contains(request.Statement, `"sparse"`) {
			writeJSON(w, map[string]any{"Items": []any{
				map[string]any{"pk": map[string]any{"S": "a"}, "name": map[string]any{"S": "alice"}},
				map[string]any{"pk": map[string]any{"S": "b"}},
			}})
			return
		}
		if request.NextToken == "" {
			writeJSON(w, map[string]any{
				"Items": []any{
					map[string]any{"pk": map[string]any{"S": "a"}, "sk": map[string]any{"N": "1"}, "name": map[string]any{"S": "alice"}},
					map[string]any{"pk": map[string]any{"S": "b"}, "sk": map[string]any{"N": "2"}, "name": map[string]any{"S": "bob"}},
				},
				"NextToken": "page2",
			})
			return
		}
		writeJSON(w, map[string]any{
			"Items": []any{
				map[string]any{"pk": map[string]any{"S": "c"}, "sk": map[string]any{"N": "3"}, "name": map[string]any{"S": "carol"}},
			},
		})
	case "ExecuteTransaction":
		if strings.Contains(string(body), "fail") {
			writeDynamoError(w, "ValidationException", "transaction failed")
			return
		}
		writeJSON(w, map[string]any{})
	default:
		writeDynamoError(w, "UnknownOperationException", "unknown operation "+target)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	encoded, _ := json.Marshal(value)
	_, _ = w.Write(encoded)
}

func writeDynamoError(w http.ResponseWriter, kind, message string) {
	w.WriteHeader(http.StatusBadRequest)
	writeJSON(w, map[string]any{
		"__type":  "com.amazonaws.dynamodb.v20120810#" + kind,
		"message": message,
	})
}

type fakeSTS struct {
	server   *httptest.Server
	requests []string
}

func newFakeSTS(t *testing.T, mode stsMode) (*httptest.Server, *fakeSTS) {
	t.Helper()
	fake := &fakeSTS{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fake.requests = append(fake.requests, string(body))
		w.Header().Set("Content-Type", "text/xml")
		if mode == stsRefuses {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<ErrorResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><Error><Type>Sender</Type><Code>AccessDenied</Code><Message>denied</Message></Error><RequestId>test-sts-request</RequestId></ErrorResponse>`))
			return
		}
		account := "<Account>123456789012</Account>"
		if mode == stsEmpty {
			account = ""
		}
		_, _ = fmt.Fprintf(w, `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><GetCallerIdentityResult><Arn>arn:aws:iam::123456789012:user/test</Arn><UserId>AIDATEST</UserId>%s</GetCallerIdentityResult><ResponseMetadata><RequestId>test-sts-request</RequestId></ResponseMetadata></GetCallerIdentityResponse>`, account)
	}))
	return fake.server, fake
}

func testErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func splitHostPort(t *testing.T, rawURL string) (string, string) {
	t.Helper()
	withoutScheme := strings.TrimPrefix(rawURL, "http://")
	host, port, ok := strings.Cut(withoutScheme, ":")
	if !ok {
		t.Fatalf("server URL %q has no port", rawURL)
	}
	return host, port
}
