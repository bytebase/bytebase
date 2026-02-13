package milvus

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestCallMilvus_ErrorPaths(t *testing.T) {
	tests := []struct {
		name         string
		handler      func(http.ResponseWriter, *http.Request)
		errorContain string
	}{
		{
			name: "http status failure",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			},
			errorContain: "status 400",
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
		},
		{
			name: "invalid json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("{"))
			},
			errorContain: "failed to decode milvus response",
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
