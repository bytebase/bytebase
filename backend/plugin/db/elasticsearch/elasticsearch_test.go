package elasticsearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestOpenWithAWSAuth(t *testing.T) {
	tests := []struct {
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
			name: "basic auth still works",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:               "localhost",
					Port:               "9200",
					Username:           "elastic",
					AuthenticationType: storepb.DataSource_PASSWORD,
				},
				Password: "password123",
			},
			// This will fail to connect but should not error during Open
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &Driver{}
			_, err := driver.Open(context.Background(), storepb.Engine_ELASTICSEARCH, tt.config)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestURLConstructionWithQueryParams(t *testing.T) {
	testCases := []struct {
		name         string
		baseAddress  string
		route        string
		wantURL      string
		wantRawQuery string
	}{
		{
			name:         "route with single query param",
			baseAddress:  "https://localhost:9200",
			route:        "/_mapping?pretty",
			wantURL:      "https://localhost:9200/_mapping?pretty",
			wantRawQuery: "pretty",
		},
		{
			name:         "route with multiple query params",
			baseAddress:  "https://localhost:9200",
			route:        "/_cat/indices?format=json&v",
			wantURL:      "https://localhost:9200/_cat/indices?format=json&v",
			wantRawQuery: "format=json&v",
		},
		{
			name:         "route without query params",
			baseAddress:  "https://localhost:9200",
			route:        "/_cat/indices",
			wantURL:      "https://localhost:9200/_cat/indices",
			wantRawQuery: "",
		},
		{
			name:         "route with index name containing special chars",
			baseAddress:  "https://localhost:9200",
			route:        "/test-index-2024/_mapping?pretty",
			wantURL:      "https://localhost:9200/test-index-2024/_mapping?pretty",
			wantRawQuery: "pretty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse base URL
			baseURL, err := url.Parse(tc.baseAddress)
			require.NoError(t, err)

			// Apply the fix logic
			routeStr := tc.route
			pathPart, queryPart, hasQuery := strings.Cut(routeStr, "?")

			fullURL := baseURL.JoinPath(pathPart)

			if hasQuery {
				fullURL.RawQuery = queryPart
			}

			// Verify results
			assert.Equal(t, tc.wantURL, fullURL.String(), "Full URL should match")
			assert.Equal(t, tc.wantRawQuery, fullURL.RawQuery, "Query should match")

			// Critical: Verify no URL encoding of the '?' character
			assert.NotContains(t, fullURL.String(), "%3F", "URL should not contain encoded question mark")
		})
	}
}

func TestQueryWithMockServer(t *testing.T) {
	// This test uses a mock HTTP server to test various JSON response types
	// including primitives which real ElasticSearch APIs rarely return
	testCases := []struct {
		name             string
		responseBody     string
		wantColumns      []string
		wantRowCount     int
		wantFirstColName string
	}{
		{
			name:             "object response",
			responseBody:     `{"took":5,"hits":{"total":100}}`,
			wantColumns:      []string{"took", "hits"},
			wantRowCount:     1,
			wantFirstColName: "", // Order not guaranteed for maps
		},
		{
			name:             "array response",
			responseBody:     `[{"index":"test1"},{"index":"test2"}]`,
			wantColumns:      []string{"result"},
			wantRowCount:     1,
			wantFirstColName: "result",
		},
		{
			name:             "primitive string",
			responseBody:     `"hello world"`,
			wantColumns:      []string{"result"},
			wantRowCount:     1,
			wantFirstColName: "result",
		},
		{
			name:             "primitive number",
			responseBody:     `12345`,
			wantColumns:      []string{"result"},
			wantRowCount:     1,
			wantFirstColName: "result",
		},
		{
			name:             "primitive boolean",
			responseBody:     `true`,
			wantColumns:      []string{"result"},
			wantRowCount:     1,
			wantFirstColName: "result",
		},
		{
			name:             "primitive null",
			responseBody:     `null`,
			wantColumns:      []string{"result"},
			wantRowCount:     1,
			wantFirstColName: "result",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			// Parse server URL
			u, err := url.Parse(server.URL)
			require.NoError(t, err)

			// Create driver and connect
			ctx := context.Background()
			driver := &Driver{}
			config := db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:               u.Hostname(),
					Port:               u.Port(),
					Username:           "",
					Password:           "",
					AuthenticationType: storepb.DataSource_PASSWORD,
				},
				Password: "",
			}

			d, err := driver.Open(ctx, storepb.Engine_ELASTICSEARCH, config)
			require.NoError(t, err)
			defer d.Close(ctx)

			// Execute query
			results, err := d.QueryConn(ctx, nil, "GET /test", db.QueryContext{})
			require.NoError(t, err)
			require.NotNil(t, results)
			require.Len(t, results, 1)

			result := results[0]
			assert.Equal(t, len(tc.wantColumns), len(result.ColumnNames), "column count should match")
			assert.Equal(t, tc.wantRowCount, len(result.Rows), "row count should match")

			if tc.wantFirstColName != "" {
				assert.Equal(t, tc.wantFirstColName, result.ColumnNames[0], "first column name should match")
			}

			// Verify we have at least one value
			require.Greater(t, len(result.Rows), 0)
			require.Greater(t, len(result.Rows[0].Values), 0)
		})
	}
}

func TestJSONParsing(t *testing.T) {
	testCases := []struct {
		name        string
		jsonData    string
		wantColumns []string
		wantErr     bool
	}{
		{
			name:        "JSON object response",
			jsonData:    `{"took":5,"hits":{"total":100}}`,
			wantColumns: []string{"took", "hits"},
			wantErr:     false,
		},
		{
			name:        "JSON array response (from _cat API)",
			jsonData:    `[{"health":"yellow","status":"open"},{"health":"green","status":"open"}]`,
			wantColumns: []string{"result"},
			wantErr:     false,
		},
		{
			name:        "JSON primitive string",
			jsonData:    `"hello world"`,
			wantColumns: []string{"result"},
			wantErr:     false,
		},
		{
			name:        "JSON primitive number",
			jsonData:    `42`,
			wantColumns: []string{"result"},
			wantErr:     false,
		},
		{
			name:        "JSON primitive boolean",
			jsonData:    `true`,
			wantColumns: []string{"result"},
			wantErr:     false,
		},
		{
			name:        "JSON primitive null",
			jsonData:    `null`,
			wantColumns: []string{"result"},
			wantErr:     false,
		},
		{
			name:     "invalid JSON",
			jsonData: `{invalid`,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			respBytes := []byte(tc.jsonData)

			// Simulate the parsing logic from QueryConn
			var columnNames []string

			// Unmarshal into any to determine type
			var data any
			err := json.Unmarshal(respBytes, &data)
			if err != nil {
				if !tc.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if tc.wantErr {
				t.Fatal("expected error but got none")
			}

			// Handle based on type
			switch v := data.(type) {
			case map[string]any:
				// Object case
				for key := range v {
					columnNames = append(columnNames, key)
				}
			case []any:
				// Array case
				columnNames = append(columnNames, "result")
			default:
				// Primitive case (string, number, boolean, null)
				columnNames = append(columnNames, "result")
			}

			// Verify column count matches
			assert.Equal(t, len(tc.wantColumns), len(columnNames),
				"number of columns should match")

			// For object responses, verify all expected columns exist
			// (order may vary due to map iteration)
			if len(tc.wantColumns) > 1 {
				for _, expected := range tc.wantColumns {
					assert.Contains(t, columnNames, expected,
						"should contain expected column")
				}
			} else {
				// For single column (array case), verify exact match
				assert.Equal(t, tc.wantColumns, columnNames)
			}
		})
	}
}
