package mcp

import (
	"context"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func TestSearchAPIListServices(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test listing all services (no parameters)
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Available Services")
	require.Contains(t, text, "SQLService")
	require.Contains(t, text, "DatabaseService")
	require.Contains(t, text, "ProjectService")
}

func TestSearchAPIByQuery(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	tests := []struct {
		name          string
		query         string
		expectContain []string
	}{
		{
			name:          "search for sql query",
			query:         "execute sql",
			expectContain: []string{"SQLService", "Query"},
		},
		{
			name:          "search for database",
			query:         "list databases",
			expectContain: []string{"Database"},
		},
		{
			name:          "search for project",
			query:         "create project",
			expectContain: []string{"Project"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
				Query: tc.query,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Content, 1)

			text := result.Content[0].(*mcpsdk.TextContent).Text
			for _, expected := range tc.expectContain {
				require.Contains(t, text, expected, "expected %q in result for query %q", expected, tc.query)
			}
		})
	}
}

func TestSearchAPIByService(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test browsing a specific service
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Service: "SQLService",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "bytebase.v1.SQLService")
	require.Contains(t, text, "endpoints")
}

func TestSearchAPIByServiceNotFound(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test browsing a non-existent service
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Service: "NonExistentService",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "No endpoints found")
	require.Contains(t, text, "Available services")
}

func TestSearchAPIWithSchema(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test with includeSchema=true
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Service:       "SQLService",
		IncludeSchema: true,
		Limit:         1,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Request body")
	require.Contains(t, text, "```json")
}

func TestSearchAPILimit(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test with custom limit
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Service: "DatabaseService",
		Limit:   2,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	// Should show "Showing X of Y results" if limited
	if strings.Contains(text, "Showing") {
		require.Contains(t, text, "Showing 2 of")
	}
}

func TestSearchAPIDefaultLimit(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test default limit (should be 5)
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Query: "list",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	// Count how many "###" headers (each endpoint starts with ###)
	count := strings.Count(text, "### ")
	require.LessOrEqual(t, count, 5, "default limit should be 5")
}

func TestSearchAPIMaxLimit(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test max limit enforcement (should cap at 50)
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Query: "list",
		Limit: 100, // Should be capped to 50
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	count := strings.Count(text, "### ")
	require.LessOrEqual(t, count, 50, "max limit should be 50")
}

func TestSearchAPINoResults(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test query with no results
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Query: "xyznonexistentquery123",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "No endpoints found")
	require.Contains(t, text, "Try:")
}
