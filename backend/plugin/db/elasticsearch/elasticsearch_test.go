package elasticsearch

import (
	"context"
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
