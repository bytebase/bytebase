package milvus

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/milvus"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestOpenAndPing(t *testing.T) {
	opened, err := newDriver().Open(context.Background(), storepb.Engine_MILVUS, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host: "127.0.0.1",
			Port: "65535",
		},
	})
	require.NoError(t, err)

	err = opened.Ping(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to ping milvus")
}

func TestQueryConnValidation(t *testing.T) {
	d := &Driver{}
	_, err := d.QueryConn(context.Background(), nil, "insert into c1 values (1)", db.QueryContext{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "readonly query statements only")

	_, err = d.QueryConn(context.Background(), nil, "select * from c1", db.QueryContext{Explain: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support EXPLAIN")
}

func TestQueryConn_RoutesAndConverts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/vectordb/collections/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": []string{"c1", "c2"},
			})
		case "/v2/vectordb/entities/query":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": []map[string]any{
					{"id": 1, "name": "a"},
					{"id": 2, "name": "b"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	d := &Driver{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	results, err := d.QueryConn(context.Background(), nil, "show collections; select id,name from c1 limit 2;", db.QueryContext{})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, []string{"collection"}, results[0].ColumnNames)
	require.Equal(t, int64(2), results[0].RowsCount)
	require.Equal(t, []string{"id", "name"}, results[1].ColumnNames)
	require.Equal(t, int64(2), results[1].RowsCount)
}

func TestQueryConn_SearchAndHybrid(t *testing.T) {
	var received []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		received = append(received, r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "c1", payload["collectionName"])

		switch r.URL.Path {
		case "/v2/vectordb/entities/search":
			require.Equal(t, float64(2), payload["limit"])
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": []map[string]any{
					{"id": 1, "distance": 0.91},
					{"id": 2, "distance": 0.78},
				},
			})
		case "/v2/vectordb/entities/hybrid_search":
			require.Equal(t, float64(2), payload["limit"])
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": []map[string]any{
					{"id": 1, "score": 0.99},
					{"id": 2, "score": 0.89},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	d := &Driver{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	stmt := `
search c1 with {"data":[[0.1,0.2,0.3,0.4]],"annsField":"vector"};
hybrid search c1 with {"search":[{"data":[[0.1,0.2,0.3,0.4]],"annsField":"vector","limit":2}],"rerank":{"strategy":"rrf","params":{"k":60}}};
`
	results, err := d.QueryConn(context.Background(), nil, stmt, db.QueryContext{Limit: 2})
	require.NoError(t, err)
	require.Equal(t, []string{"/v2/vectordb/entities/search", "/v2/vectordb/entities/hybrid_search"}, received)
	require.Len(t, results, 2)
	require.Contains(t, results[0].ColumnNames, "distance")
	require.Contains(t, results[0].ColumnNames, "score")
	require.Contains(t, results[1].ColumnNames, "distance")
	require.Contains(t, results[1].ColumnNames, "score")
	require.Len(t, results[0].Rows, 2)
	require.Len(t, results[1].Rows, 2)
}

func TestSyncInstanceAndSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/version":
			require.Equal(t, http.MethodGet, r.Method)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"version": "2.5.5",
			})
		case "/v2/vectordb/collections/list":
			require.Equal(t, http.MethodPost, r.Method)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": []string{"c2", "c1"},
			})
		case "/v2/vectordb/collections/describe":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(body, &payload))
			collectionName, _ := payload["collectionName"].(string)
			require.NotEmpty(t, collectionName)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"fields": []map[string]any{
						{
							"name":     "id",
							"dataType": "Int64",
							"nullable": false,
						},
						{
							"name":     "vector",
							"dataType": "FloatVector",
							"nullable": false,
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	d := &Driver{
		config: db.ConnectionConfig{
			ConnectionContext: db.ConnectionContext{
				DatabaseName: "milvus",
			},
		},
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	instance, err := d.SyncInstance(context.Background())
	require.NoError(t, err)
	require.Equal(t, "2.5.5", instance.Version)
	require.Len(t, instance.Databases, 1)
	require.Equal(t, "milvus", instance.Databases[0].Name)
	require.Len(t, instance.Databases[0].Schemas, 1)
	require.Len(t, instance.Databases[0].Schemas[0].Tables, 2)
	require.Equal(t, "c1", instance.Databases[0].Schemas[0].Tables[0].Name)
	require.Equal(t, "c2", instance.Databases[0].Schemas[0].Tables[1].Name)

	schema, err := d.SyncDBSchema(context.Background())
	require.NoError(t, err)
	require.Equal(t, "milvus", schema.Name)
	require.Len(t, schema.Schemas, 1)
	require.Len(t, schema.Schemas[0].Tables, 2)
	require.Equal(t, "c1", schema.Schemas[0].Tables[0].Name)
	require.Len(t, schema.Schemas[0].Tables[0].Columns, 2)
	require.Equal(t, "id", schema.Schemas[0].Tables[0].Columns[0].Name)
	require.Equal(t, "Int64", schema.Schemas[0].Tables[0].Columns[0].Type)
	require.False(t, schema.Schemas[0].Tables[0].Columns[0].Nullable)
}

func TestExecute_RoutesAndPayloads(t *testing.T) {
	var (
		calls       []string
		payloadByOp = make(map[string]map[string]any)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		calls = append(calls, r.URL.Path)
		payloadByOp[r.URL.Path] = payload
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0})
	}))
	defer server.Close()

	d := &Driver{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	statement := `
	create collection c1 with {"dimension":4};
	create partition p1 in c1;
	create index on c1 field vector with {"metricType":"L2"};
	create alias a1 for c1;
	create user u1 with password 'pass123';
	create role r1;
	grant role r1 to user u1;
	insert into c1 values [{"id":1,"vector":[0.1,0.2,0.3,0.4]}];
	delete from c1 where id > 0;
	release collection c1;
	`

	_, err := d.Execute(context.Background(), statement, db.ExecuteOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{
		"/v2/vectordb/collections/create",
		"/v2/vectordb/partitions/create",
		"/v2/vectordb/indexes/create",
		"/v2/vectordb/aliases/create",
		"/v2/vectordb/users/create",
		"/v2/vectordb/roles/create",
		"/v2/vectordb/users/grant_role",
		"/v2/vectordb/entities/insert",
		"/v2/vectordb/entities/delete",
		"/v2/vectordb/collections/release",
	}, calls)
	require.Equal(t, "c1", payloadByOp["/v2/vectordb/collections/release"]["collectionName"])
	require.Equal(t, "id > 0", payloadByOp["/v2/vectordb/entities/delete"]["filter"])
	indexParams, ok := payloadByOp["/v2/vectordb/indexes/create"]["indexParams"].([]any)
	require.True(t, ok)
	require.Len(t, indexParams, 1)
	indexParam, ok := indexParams[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "vector", indexParam["fieldName"])
	require.Equal(t, "L2", indexParam["metricType"])
}

func TestExecute_RetryOnUnavailable(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := attempts.Add(1)
		if current <= 2 {
			http.Error(w, "try again", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0})
	}))
	defer server.Close()

	d := &Driver{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := d.Execute(context.Background(), "drop collection c1;", db.ExecuteOptions{MaximumRetries: 3})
	require.NoError(t, err)
	require.Equal(t, int32(3), attempts.Load())
}

func TestExecute_Errors(t *testing.T) {
	d := &Driver{}

	_, err := d.Execute(context.Background(), "truncate collection c1;", db.ExecuteOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported milvus execute statement")

	_, err = d.Execute(context.Background(), `create collection c1 with {"dimension": 4,};`, db.ExecuteOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid JSON payload")
}

func TestExecute_StopsOnFailure(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		if strings.Contains(r.URL.Path, "/collections/drop") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0})
	}))
	defer server.Close()

	d := &Driver{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := d.Execute(context.Background(), "create role r1; drop collection c1; create user u1 with password 'p';", db.ExecuteOptions{})
	require.Error(t, err)
	require.Equal(t, []string{
		"/v2/vectordb/roles/create",
		"/v2/vectordb/collections/drop",
	}, calls)
}

func TestExecute_CancelDuringRetryBackoff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "busy", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	d := &Driver{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := d.Execute(ctx, "drop collection c1;", db.ExecuteOptions{MaximumRetries: 3})
	require.Error(t, err)
	require.Contains(t, err.Error(), "retry canceled")
}

func TestNormalizeCreateIndexParams(t *testing.T) {
	params, err := normalizeCreateIndexParams("vector", map[string]any{
		"indexType":  "AUTOINDEX",
		"metricType": "L2",
	})
	require.NoError(t, err)
	require.Len(t, params, 1)
	require.Equal(t, "vector", params[0]["fieldName"])
	require.Equal(t, "AUTOINDEX", params[0]["indexType"])

	params, err = normalizeCreateIndexParams("vector", map[string]any{
		"indexParams": map[string]any{
			"metricType": "IP",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "vector", params[0]["fieldName"])
	require.Equal(t, "IP", params[0]["metricType"])

	_, err = normalizeCreateIndexParams("vector", map[string]any{
		"indexParams": "bad",
	})
	require.Error(t, err)
}

func TestCallMilvus_ErrorPaths(t *testing.T) {
	tests := []struct {
		name         string
		handler      func(http.ResponseWriter, *http.Request)
		errorContain string
		errorCode    connect.Code
	}{
		{
			name: "http status failure",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			},
			errorContain: "status 400",
			errorCode:    connect.CodeInvalidArgument,
		},
		{
			name: "milvus non zero code",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    123,
					"message": "boom",
				})
			},
			errorContain: "boom",
			errorCode:    connect.CodeFailedPrecondition,
		},
		{
			name: "invalid json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("{"))
			},
			errorContain: "failed to decode milvus response",
			errorCode:    connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.handler))
			defer server.Close()

			d := &Driver{
				httpClient: server.Client(),
				baseURL:    server.URL,
			}
			_, err := d.callMilvus(context.Background(), "/v2/vectordb/collections/list", map[string]any{})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContain)
			require.Equal(t, tt.errorCode, connect.CodeOf(err))
		})
	}
}

func TestSyncDBSchema_DescribeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/vectordb/collections/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": []string{"c1"},
			})
		case "/v2/vectordb/collections/describe":
			http.Error(w, "failed", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	d := &Driver{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := d.SyncDBSchema(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), `failed to describe collection "c1"`)
}

func TestFetchVersion_FallbackPlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/version", r.URL.Path)
		_, _ = io.Copy(w, bytes.NewBufferString("2.6.0"))
	}))
	defer server.Close()

	d := &Driver{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}
	version, err := d.fetchVersion(context.Background())
	require.NoError(t, err)
	require.Equal(t, "2.6.0", version)
}
