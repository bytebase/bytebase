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
			name:          "search for sql",
			query:         "sql",
			expectContain: []string{"SQL"},
		},
		{
			name:          "search for database",
			query:         "database",
			expectContain: []string{"Database"},
		},
		{
			name:          "search for project",
			query:         "project",
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
	require.Contains(t, text, "SQLService/")
	require.Contains(t, text, "endpoints")
	// Browse mode should not include schema
	require.NotContains(t, text, "Request Body")
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

func TestSearchAPIDetailMode(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test detail mode with full operationId
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "bytebase.v1.SQLService.Query",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "SQLService/Query")
	require.Contains(t, text, "Request Body")
	require.Contains(t, text, "```json")
}

func TestSearchAPIDetailModeShortFormat(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test detail mode with short operationId format (Service/Method)
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "SQLService/Query",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "SQLService/Query")
	require.Contains(t, text, "Request Body")
	require.Contains(t, text, "```json")
}

func TestSearchAPIDetailModeWithResponse(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test detail mode shows response schema
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "bytebase.v1.DatabaseService.ListDatabases",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Response Body")
}

func TestSearchAPIDetailModeNotFound(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test detail mode with unknown operationId
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "unknown.operation.id",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Unknown operationId")
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

	// Test query with no results - use a query that truly won't match anything
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Query: "zzzzqqqq",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "No endpoints found")
	require.Contains(t, text, "Try:")
}

func TestSearchAPIServiceShowsAll(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test browsing a service shows all endpoints (no limit)
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Service: "DatabaseService",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	// Should show "Found X endpoints" not "Showing X of Y"
	require.Contains(t, text, "Found")
	require.NotContains(t, text, "Showing")
}

func TestSearchAPIQueryWithService(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test query within a specific service
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Service: "SQLService",
		Query:   "query",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "SQLService/")
	// Should only return SQLService endpoints
	require.NotContains(t, text, "DatabaseService/")
}

func TestSearchAPISchemaLookup(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test schema lookup with full name
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "bytebase.v1.Instance",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "bytebase.v1.Instance")
	require.Contains(t, text, "\"name\":")
	require.Contains(t, text, "\"engine\":")
}

func TestSearchAPISchemaLookupShortName(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test schema lookup with short name
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "Instance",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "bytebase.v1.Instance")
	require.Contains(t, text, "\"name\":")
}

func TestSearchAPISchemaLookupNotFound(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test schema lookup with unknown name
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "NonExistentSchema",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Unknown schema")
}

func TestSearchAPISchemaLookupEnum(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test enum schema lookup
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "Engine",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Enum values:")
}

func TestSearchAPIProtobufDescriptionTruncation(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test that protobuf types have short descriptions
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "InstanceService/CreateInstance",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	text := result.Content[0].(*mcpsdk.TextContent).Text

	// Should NOT contain verbose protobuf documentation
	require.NotContains(t, text, "A Timestamp represents a point in time")
	require.NotContains(t, text, "A Duration represents a signed")

	// Should contain short description if there's a timestamp field
	if strings.Contains(text, "google.protobuf.Timestamp") {
		require.Contains(t, text, "ISO 8601")
	}
}
