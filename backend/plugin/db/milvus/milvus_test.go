package milvus

import (
	"context"
	"encoding/json"
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
