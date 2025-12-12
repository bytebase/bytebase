package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchInput is the input for the search_api tool.
type SearchInput struct {
	// Query is a free-text search query to find relevant API endpoints.
	// Examples: "create database", "execute sql", "list projects", "user management"
	Query string `json:"query,omitempty"`
	// Service filters results to a specific service.
	// Use list_services mode first to discover available services.
	// Examples: "SQLService", "DatabaseService", "ProjectService"
	Service string `json:"service,omitempty"`
	// Limit is the maximum number of results to return (default: 5, max: 50).
	Limit int `json:"limit,omitempty"`
	// IncludeSchema includes request schema details in the results.
	IncludeSchema bool `json:"includeSchema,omitempty"`
}

// searchAPIDescription is the description for the search_api tool.
const searchAPIDescription = `Search and discover Bytebase API endpoints.

**IMPORTANT: You MUST call search_api before call_api. Never guess operationIds or request schemas - always discover them first.**

## Usage

| Mode | Parameters | Use Case |
|------|------------|----------|
| List services | (none) | Discover available API categories |
| Search | query="execute sql" | Find endpoints by functionality |
| Browse | service="SQLService" | List all endpoints in a service |
| Details | service="...", includeSchema=true | Get request schema for call_api |

## Required Workflow

1. search_api(query="your task") - find relevant endpoints
2. search_api(service="...", includeSchema=true) - get request schema
3. call_api(operationId="...", body={...}) - execute with correct schema

## Response Fields

- operationId: Required for call_api
- summary: What the endpoint does
- permissions: Required permissions
- requestSchema: Input parameters (with includeSchema=true)`

func (s *Server) registerSearchTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_api",
		Description: searchAPIDescription,
	}, s.handleSearchAPI)
}

func (s *Server) handleSearchAPI(_ context.Context, _ *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, any, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	var text string

	switch {
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
			text = s.formatEndpoints(endpoints, limit, input.IncludeSchema)
		}

	default:
		// Search by query
		endpoints := s.openAPIIndex.Search(input.Query)
		if len(endpoints) == 0 {
			text = fmt.Sprintf("No endpoints found for query: %q\n\nTry:\n- Different keywords\n- Listing services with search_api() (no parameters)\n- Browsing a service with search_api(service=\"ServiceName\")", input.Query)
		} else {
			text = s.formatEndpoints(endpoints, limit, input.IncludeSchema)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
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

func (s *Server) formatEndpoints(endpoints []*EndpointInfo, limit int, includeSchema bool) string {
	var sb strings.Builder

	if len(endpoints) > limit {
		sb.WriteString(fmt.Sprintf("Showing %d of %d results:\n\n", limit, len(endpoints)))
		endpoints = endpoints[:limit]
	} else {
		sb.WriteString(fmt.Sprintf("Found %d endpoints:\n\n", len(endpoints)))
	}

	for i, ep := range endpoints {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, ep.OperationID))
		sb.WriteString(fmt.Sprintf("**%s**\n", ep.Summary))

		if len(ep.Permissions) > 0 {
			sb.WriteString(fmt.Sprintf("Permissions: `%s`\n", strings.Join(ep.Permissions, "`, `")))
		}

		if includeSchema && ep.RequestSchemaRef != "" {
			props := s.openAPIIndex.GetRequestSchema(ep.RequestSchemaRef)
			if len(props) > 0 {
				sb.WriteString("\nRequest body:\n```json\n{\n")
				for _, prop := range props {
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
				sb.WriteString("}\n```\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
