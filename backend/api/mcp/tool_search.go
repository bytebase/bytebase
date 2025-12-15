package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchInput is the input for the search_api tool.
type SearchInput struct {
	// OperationID gets detailed schema for a specific endpoint.
	// Use this after finding the endpoint you need.
	OperationID string `json:"operationId,omitempty"`
	// Schema gets the definition of a message type.
	// Examples: "bytebase.v1.Instance", "Instance", "Engine"
	Schema string `json:"schema,omitempty"`
	// Query is a free-text search query to find relevant API endpoints.
	// Examples: "create database", "execute sql", "list projects"
	Query string `json:"query,omitempty"`
	// Service filters results to a specific service.
	// Examples: "SQLService", "DatabaseService", "ProjectService"
	Service string `json:"service,omitempty"`
	// Limit is the maximum number of results to return (default: 5, max: 50).
	Limit int `json:"limit,omitempty"`
}

// searchAPIDescription is the description for the search_api tool.
const searchAPIDescription = `Discover Bytebase API endpoints. **Always call before call_api - never guess schemas.**

| Mode | Parameters | Result |
|------|------------|--------|
| List | (none) | All services |
| Browse | service="SQLService" | All endpoints in service |
| Search | query="database" | Matching endpoints |
| Filter | service+query | Search within service |
| Details | operationId="SQLService/Query" | Request/response schema |
| Schema | schema="Instance" | Message type definition |

**Workflow:** search_api() → search_api(operationId="...") → call_api(...)`

func (s *Server) registerSearchTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_api",
		Description: searchAPIDescription,
	}, s.handleSearchAPI)
}

func (s *Server) handleSearchAPI(_ context.Context, _ *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, any, error) {
	var text string

	switch {
	case input.OperationID != "":
		// Detail mode: get full schema for a specific endpoint
		text = s.formatEndpointDetail(input.OperationID)

	case input.Schema != "":
		// Schema lookup mode: get properties of a message type
		text = s.formatSchemaDetail(input.Schema)

	case input.Query == "" && input.Service == "":
		// List all services
		text = s.formatServiceList()

	case input.Service != "" && input.Query != "":
		// Search within a service
		endpoints := s.openAPIIndex.SearchInService(input.Query, input.Service)
		if len(endpoints) == 0 {
			text = fmt.Sprintf("No endpoints found for query %q in service %s\n\nTry:\n- Different keywords\n- Browsing the service with search_api(service=\"%s\")",
				input.Query, input.Service, input.Service)
		} else {
			text = s.formatEndpoints(endpoints, s.getLimit(input.Limit))
		}

	case input.Service != "":
		// List all endpoints in a service (no limit)
		endpoints := s.openAPIIndex.GetServiceEndpoints(input.Service)
		if len(endpoints) == 0 {
			text = fmt.Sprintf("No endpoints found for service: %s\n\nAvailable services:\n%s",
				input.Service, s.formatServiceList())
		} else {
			text = s.formatEndpoints(endpoints, 0)
		}

	default:
		// Search by query
		endpoints := s.openAPIIndex.Search(input.Query)
		if len(endpoints) == 0 {
			text = fmt.Sprintf("No endpoints found for query: %q\n\nTry:\n- Different keywords\n- Listing services with search_api() (no parameters)\n- Browsing a service with search_api(service=\"ServiceName\")", input.Query)
		} else {
			text = s.formatEndpoints(endpoints, s.getLimit(input.Limit))
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func (*Server) getLimit(limit int) int {
	if limit <= 0 {
		return 5
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func (s *Server) formatServiceList() string {
	services := s.openAPIIndex.Services()
	var sb strings.Builder

	sb.WriteString("## Available Services\n\n")
	sb.WriteString("Use `search_api(service=\"ServiceName\")` to list endpoints in a service.\n\n")

	for _, svc := range services {
		endpoints := s.openAPIIndex.GetServiceEndpoints(svc)
		sb.WriteString(fmt.Sprintf("- **%s** (%d endpoints)\n", svc, len(endpoints)))
	}

	sb.WriteString(fmt.Sprintf("\nTotal: %d services\n", len(services)))
	return sb.String()
}

func (*Server) formatEndpoints(endpoints []*EndpointInfo, limit int) string {
	var sb strings.Builder

	if limit > 0 && len(endpoints) > limit {
		sb.WriteString(fmt.Sprintf("Showing %d of %d results:\n\n", limit, len(endpoints)))
		endpoints = endpoints[:limit]
	} else {
		sb.WriteString(fmt.Sprintf("Found %d endpoints:\n\n", len(endpoints)))
	}

	for i, ep := range endpoints {
		sb.WriteString(fmt.Sprintf("### %d. %s/%s\n", i+1, ep.Service, ep.Method))
		sb.WriteString(fmt.Sprintf("%s\n", ep.Summary))

		if len(ep.Permissions) > 0 {
			sb.WriteString(fmt.Sprintf("Permissions: `%s`\n", strings.Join(ep.Permissions, "`, `")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (s *Server) formatEndpointDetail(operationID string) string {
	ep, ok := s.openAPIIndex.GetEndpoint(operationID)
	if !ok {
		return fmt.Sprintf("Unknown operationId: %s\n\nUse search_api(query=\"...\") to find valid operations.", operationID)
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s/%s\n\n", ep.Service, ep.Method))
	sb.WriteString(fmt.Sprintf("%s\n\n", ep.Summary))

	if len(ep.Permissions) > 0 {
		sb.WriteString(fmt.Sprintf("**Permissions:** `%s`\n\n", strings.Join(ep.Permissions, "`, `")))
	}

	// Request schema
	if ep.RequestSchemaRef != "" {
		props := s.openAPIIndex.GetRequestSchema(ep.RequestSchemaRef)
		if len(props) > 0 {
			sb.WriteString("### Request Body\n```json\n{\n")
			for _, prop := range props {
				s.formatProperty(&sb, prop)
			}
			sb.WriteString("}\n```\n\n")
		}
	}

	// Response schema
	if ep.ResponseSchemaRef != "" {
		props := s.openAPIIndex.GetRequestSchema(ep.ResponseSchemaRef) // same method works for response
		if len(props) > 0 {
			sb.WriteString("### Response Body\n```json\n{\n")
			for _, prop := range props {
				s.formatProperty(&sb, prop)
			}
			sb.WriteString("}\n```\n")
		}
	}

	return sb.String()
}

func (s *Server) formatSchemaDetail(schemaName string) string {
	props, ok := s.openAPIIndex.GetSchema(schemaName)
	if !ok {
		return fmt.Sprintf("Unknown schema: %s\n\nUse search_api(operationId=\"...\") to see schemas in request/response bodies.", schemaName)
	}

	var sb strings.Builder

	// Normalize name for display
	displayName := schemaName
	if !strings.HasPrefix(schemaName, "bytebase.v1.") {
		displayName = "bytebase.v1." + schemaName
	}

	sb.WriteString(fmt.Sprintf("## %s\n\n", displayName))

	// Check if it's an enum
	if len(props) == 1 && props[0].Name == "enum" {
		sb.WriteString("**Enum values:** ")
		sb.WriteString(props[0].Description)
		sb.WriteString("\n")
		return sb.String()
	}

	for _, prop := range props {
		s.formatProperty(&sb, prop)
	}

	return sb.String()
}

func (*Server) formatProperty(sb *strings.Builder, prop PropertyInfo) {
	required := ""
	if prop.Required {
		required = " (required)"
	}

	desc := ""
	// Check if type has a known short description
	if shortDesc, ok := GetTypeDescription(prop.Type); ok {
		desc = fmt.Sprintf(" // %s", shortDesc)
	} else if prop.Description != "" {
		// Remove newlines and truncate long descriptions
		cleanDesc := strings.ReplaceAll(prop.Description, "\n", " ")
		cleanDesc = strings.ReplaceAll(cleanDesc, "\r", "")
		// Truncate at 100 chars
		if len(cleanDesc) > 100 {
			cleanDesc = cleanDesc[:97] + "..."
		}
		desc = fmt.Sprintf(" // %s", cleanDesc)
	}

	sb.WriteString("  \"")
	sb.WriteString(prop.Name)
	sb.WriteString("\": ")
	sb.WriteString(prop.Type)
	sb.WriteString(required)
	sb.WriteString(desc)
	sb.WriteString("\n")
}
