package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// SearchInput is the input for the search_api tool.
type SearchInput struct {
	// OperationID gets detailed schema for a specific endpoint.
	// Use this after finding the endpoint you need.
	OperationID string `json:"operationId,omitempty"`
	// Schema gets the definition of a message type.
	// Examples: "bytebase.v1.Instance", "Instance", "Engine"
	Schema string `json:"schema,omitempty"`
	// Service filters results to a specific service.
	// Examples: "SQLService", "DatabaseService", "ProjectService"
	Service string `json:"service,omitempty"`
}

// searchAPIDescription is the description for the search_api tool.
const searchAPIDescription = `Discover Bytebase API endpoints. **Always call before call_api - never guess schemas.**

| Mode | Parameters | Result |
|------|------------|--------|
| List | (none) | All services |
| Browse | service="SQLService" | All endpoints in service |
| Details | operationId="SQLService/Query" | Request/response schema |
| Schema | schema="Instance" | Message type definition |

**Workflow:** search_api() → search_api(service="...") → search_api(operationId="...") → call_api(...)

The results also name what MCP sessions cannot do, and why: those are done in the Bytebase console.`

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

	case input.Service == "":
		// List all services
		text = s.formatServiceList()

	default:
		text = s.formatService(input.Service)
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
		fmt.Fprintf(&sb, "- **%s** (%d endpoints", svc, len(endpoints))
		if unavailable := len(s.openAPIIndex.GetUnavailableEndpoints(svc)); unavailable > 0 {
			fmt.Fprintf(&sb, "; %d not available over MCP", unavailable)
		}
		sb.WriteString(")\n")
	}

	fmt.Fprintf(&sb, "\nTotal: %d services\n", len(services))

	unavailable := s.openAPIIndex.UnavailableServices()
	if len(unavailable) == 0 {
		return sb.String()
	}
	sb.WriteString("\n## Not available over MCP\n\n")
	sb.WriteString("These services exist in Bytebase, but no MCP session can call them, whatever the user's permissions. " +
		mcpUnavailableWayOut + " `search_api(service=\"...\")` lists each operation and why.\n\n")
	for _, svc := range unavailable {
		fmt.Fprintf(&sb, "- **%s**", svc)
		if description := s.describeService(svc); description != "" {
			fmt.Fprintf(&sb, ": %s", description)
		}
		sb.WriteString("\n")
		for _, reason := range refusalReasons(s.openAPIIndex.GetUnavailableEndpoints(svc)) {
			fmt.Fprintf(&sb, "  - Refused because %s.\n", reason)
		}
	}
	return sb.String()
}

// mcpUnavailableWayOut is where the user does what MCP refuses.
const mcpUnavailableWayOut = "Do this work signed in to the Bytebase console."

// formatService lists a service's endpoints and the ones MCP cannot call, and
// tells a service MCP cannot call at all apart from one Bytebase does not have.
func (s *Server) formatService(service string) string {
	endpoints := s.openAPIIndex.GetServiceEndpoints(service)
	unavailable := s.openAPIIndex.GetUnavailableEndpoints(service)
	switch {
	case len(endpoints) > 0 && len(unavailable) > 0:
		return s.formatEndpoints(endpoints, 0) + "## Not available over MCP\n\n" +
			"These exist in Bytebase, but no MCP session can call them, whatever the user's permissions. " +
			mcpUnavailableWayOut + "\n\n" + formatRefusals(unavailable)
	case len(endpoints) > 0:
		return s.formatEndpoints(endpoints, 0)
	case len(unavailable) > 0:
		var sb strings.Builder
		fmt.Fprintf(&sb, "%s is not available over MCP: no MCP session can call its operations, whatever the user's permissions. %s\n\n",
			service, mcpUnavailableWayOut)
		if description := s.describeService(service); description != "" {
			fmt.Fprintf(&sb, "%s: %s\n\n", service, description)
		}
		sb.WriteString(formatRefusals(unavailable))
		return sb.String()
	default:
		return fmt.Sprintf("Unknown service: %s. Bytebase has no service by that name.\n\n%s", service, s.formatServiceList())
	}
}

// describeService is the service's spec description without its leading name:
// "WorkloadIdentityService manages workload identities" becomes "manages
// workload identities".
func (s *Server) describeService(service string) string {
	description := strings.TrimSpace(s.openAPIIndex.ServiceDescription(service))
	return strings.TrimPrefix(description, service+" ")
}

// refusalReasons returns the distinct reasons MCP refuses these endpoints, in
// the order they first appear.
func refusalReasons(endpoints []*EndpointInfo) []string {
	var reasons []string
	for _, ep := range endpoints {
		if !slices.Contains(reasons, ep.Refusal) {
			reasons = append(reasons, ep.Refusal)
		}
	}
	return reasons
}

// formatRefusals lists endpoints MCP cannot call, grouped by the reason.
func formatRefusals(endpoints []*EndpointInfo) string {
	var sb strings.Builder
	for _, reason := range refusalReasons(endpoints) {
		var operations []string
		for _, ep := range endpoints {
			if ep.Refusal == reason {
				operations = append(operations, ep.Service+"/"+ep.Method)
			}
		}
		slices.Sort(operations)
		fmt.Fprintf(&sb, "- %s: refused because %s.\n", strings.Join(operations, ", "), reason)
	}
	return sb.String()
}

// endpointNote is what an agent must know about an endpoint that MCP serves
// only in part.
func endpointNote(ep *EndpointInfo) string {
	if ep.Path != v1connect.SettingServiceGetSettingProcedure {
		return ""
	}
	var readable []string
	for _, setting := range auth.MCPReadableSettings() {
		readable = append(readable, common.SettingNamePrefix+setting.String())
	}
	return fmt.Sprintf("MCP sessions read only these settings: %s. The user's own Bytebase permissions still apply. "+
		"Semantic types come back without the MD5 masking salt.", strings.Join(readable, ", "))
}

func (*Server) formatEndpoints(endpoints []*EndpointInfo, limit int) string {
	var sb strings.Builder

	if limit > 0 && len(endpoints) > limit {
		fmt.Fprintf(&sb, "Showing %d of %d results:\n\n", limit, len(endpoints))
		endpoints = endpoints[:limit]
	} else {
		fmt.Fprintf(&sb, "Found %d endpoints:\n\n", len(endpoints))
	}

	for i, ep := range endpoints {
		fmt.Fprintf(&sb, "### %d. %s/%s\n", i+1, ep.Service, ep.Method)
		fmt.Fprintf(&sb, "%s\n", ep.Summary)

		if len(ep.Permissions) > 0 {
			fmt.Fprintf(&sb, "Permissions: `%s`\n", strings.Join(ep.Permissions, "`, `"))
		}
		if note := endpointNote(ep); note != "" {
			fmt.Fprintf(&sb, "%s\n", note)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (s *Server) formatEndpointDetail(operationID string) string {
	ep, ok := s.openAPIIndex.GetEndpoint(operationID)
	if !ok {
		return fmt.Sprintf("Unknown operationId: %s. Bytebase has no operation by that name.\n\nUse search_api(service=\"...\") to browse endpoints.", operationID)
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "## %s/%s\n\n", ep.Service, ep.Method)
	fmt.Fprintf(&sb, "%s\n\n", ep.Summary)

	if ep.Refusal != "" {
		fmt.Fprintf(&sb, "Not available over MCP: no MCP session can call this operation, whatever the user's permissions, because %s. %s\n",
			ep.Refusal, mcpUnavailableWayOut)
		return sb.String()
	}

	if len(ep.Permissions) > 0 {
		fmt.Fprintf(&sb, "**Permissions:** `%s`\n\n", strings.Join(ep.Permissions, "`, `"))
	}
	if note := endpointNote(ep); note != "" {
		fmt.Fprintf(&sb, "%s\n\n", note)
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

	fmt.Fprintf(&sb, "## %s\n\n", displayName)

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
		if truncated, ok := common.TruncateString(cleanDesc, 97); ok {
			cleanDesc = truncated + "..."
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
