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
const searchAPIDescription = `Search and discover Bytebase API endpoints.

**IMPORTANT: You MUST call search_api before call_api. Never guess operationIds or request schemas.**

## Usage

| Mode | Parameters | Result |
|------|------------|--------|
| List services | (none) | All available API categories |
| Search | query="execute sql" | Matching endpoints with descriptions |
| Browse | service="SQLService" | All endpoints in a service |
| Details | operationId="..." | Full request/response schema |

## Workflow

1. search_api(query="your task") - find relevant endpoints
2. search_api(operationId="...") - get full schema
3. call_api(operationId="...", body={...}) - execute`

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

	case input.Query == "" && input.Service == "":
		// List all services
		text = s.formatServiceList()

	case input.Service != "":
		// List endpoints in a service
		endpoints := s.openAPIIndex.GetServiceEndpoints(input.Service)
		if len(endpoints) == 0 {
			text = fmt.Sprintf("No endpoints found for service: %s\n\nAvailable services:\n%s",
				input.Service, s.formatServiceList())
		} else {
			text = s.formatEndpoints(endpoints, s.getLimit(input.Limit))
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

func (s *Server) formatEndpoints(endpoints []*EndpointInfo, limit int) string {
	var sb strings.Builder

	if len(endpoints) > limit {
		sb.WriteString(fmt.Sprintf("Showing %d of %d results:\n\n", limit, len(endpoints)))
		endpoints = endpoints[:limit]
	} else {
		sb.WriteString(fmt.Sprintf("Found %d endpoints:\n\n", len(endpoints)))
	}

	for i, ep := range endpoints {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, ep.OperationID))
		sb.WriteString(fmt.Sprintf("%s\n", ep.Summary))

		if len(ep.Permissions) > 0 {
			sb.WriteString(fmt.Sprintf("Permissions: `%s`\n", strings.Join(ep.Permissions, "`, `")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Use `search_api(operationId=\"...\")` to get full request/response schema.\n")
	return sb.String()
}

func (s *Server) formatEndpointDetail(operationID string) string {
	ep, ok := s.openAPIIndex.GetEndpoint(operationID)
	if !ok {
		return fmt.Sprintf("Unknown operationId: %s\n\nUse search_api(query=\"...\") to find valid operations.", operationID)
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s\n\n", ep.OperationID))
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

func (*Server) formatProperty(sb *strings.Builder, prop PropertyInfo) {
	required := ""
	if prop.Required {
		required = " (required)"
	}
	desc := ""
	if prop.Description != "" {
		// Remove newlines for inline comment
		cleanDesc := strings.ReplaceAll(prop.Description, "\n", " ")
		cleanDesc = strings.ReplaceAll(cleanDesc, "\r", "")
		desc = fmt.Sprintf(" // %s", cleanDesc)
	}
	sb.WriteString(fmt.Sprintf("  %q: %s%s%s\n", prop.Name, prop.Type, required, desc))
}
