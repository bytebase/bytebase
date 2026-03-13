package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

// CallInput is the input for the call_api tool.
type CallInput struct {
	// OperationID is the unique identifier for the API endpoint to call.
	// Use search_api to find the correct operationId for your task.
	// Example: "bytebase.v1.SQLService.Query"
	OperationID string `json:"operationId"`
	// Body is the JSON request body to send to the endpoint.
	// The structure depends on the endpoint - use search_api with includeSchema=true to see the expected format.
	Body map[string]any `json:"body,omitempty"`
}

// CallOutput is the output for the call_api tool.
type CallOutput struct {
	// Status is the HTTP status code of the response.
	Status int `json:"status"`
	// Response is the JSON response body from the API.
	Response any `json:"response,omitempty"`
	// Error is the error message if the request failed.
	Error string `json:"error,omitempty"`
}

// callAPIDescription is the description for the call_api tool.
const callAPIDescription = `Execute a Bytebase API endpoint. **Use search_api first to get operationId and schema.**

| Parameter | Required | Description |
|-----------|----------|-------------|
| operationId | Yes | e.g., "SQLService/Query" |
| body | No | JSON request body |

**Resource names:** projects/my-project, instances/prod/databases/main

**Example:**
call_api(operationId="SQLService/Query", body={"name": "instances/i/databases/db", "statement": "SELECT 1"})`

func (s *Server) registerCallTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "call_api",
		Description: callAPIDescription,
	}, s.handleCallAPI)
}

func (s *Server) handleCallAPI(ctx context.Context, _ *mcp.CallToolRequest, input CallInput) (*mcp.CallToolResult, any, error) {
	// Validate input
	if input.OperationID == "" {
		return nil, nil, errors.New("operationId is required")
	}

	// Look up endpoint
	endpoint, ok := s.openAPIIndex.GetEndpoint(input.OperationID)
	if !ok {
		return nil, nil, errors.Errorf("unknown operation %s, use search_api to find valid operations", input.OperationID)
	}

	// Build request body
	body := input.Body
	if body == nil {
		body = make(map[string]any)
	}

	// Execute API request
	resp, err := s.apiRequest(ctx, endpoint.Path, body)
	if err != nil {
		return nil, nil, err
	}

	// Check for binary response - not supported
	contentType := resp.Headers.Get("Content-Type")
	if isBinaryContentType(contentType) {
		return nil, nil, errors.Errorf("binary response not supported (content-type: %s)", contentType)
	}

	// Parse JSON response
	output := CallOutput{
		Status: resp.Status,
	}
	if len(resp.Body) > 0 {
		var respJSON any
		if json.Unmarshal(resp.Body, &respJSON) != nil {
			respJSON = string(resp.Body)
		}
		output.Response = respJSON
	}

	// Check for error response
	if resp.Status >= 400 {
		output.Error = parseError(resp.Body)
		if output.Error == "" {
			output.Error = fmt.Sprintf("HTTP %d", resp.Status)
		}
	}

	text := formatCallOutput(output, endpoint)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
}

func formatCallOutput(output CallOutput, endpoint *EndpointInfo) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "## %s\n\n", endpoint.OperationID)

	if output.Error != "" {
		fmt.Fprintf(&sb, "**Error** (HTTP %d): %s\n\n", output.Status, output.Error)
	} else {
		fmt.Fprintf(&sb, "**Status:** %d OK\n\n", output.Status)
	}

	if output.Response != nil {
		respBytes, _ := json.MarshalIndent(output.Response, "", "  ")
		sb.WriteString("**Response:**\n```json\n")
		sb.Write(respBytes)
		sb.WriteString("\n```\n")
	}

	return sb.String()
}

func isBinaryContentType(ct string) bool {
	ct = strings.ToLower(ct)
	binaryPrefixes := []string{
		"application/octet-stream",
		"application/zip",
		"application/pdf",
		"application/gzip",
		"image/",
		"audio/",
		"video/",
	}
	for _, prefix := range binaryPrefixes {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	return false
}
