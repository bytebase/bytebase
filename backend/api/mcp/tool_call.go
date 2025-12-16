package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal request body")
	}

	// Build HTTP request
	url := fmt.Sprintf("http://localhost:%d%s", s.profile.Port, endpoint.Path)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create HTTP request")
	}

	// Set headers for Connect RPC
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Connect-Protocol-Version", "1")

	// Forward the auth token from the MCP context
	if token := getAccessToken(ctx); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	// Execute request with timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to execute API request")
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read response body")
	}

	// Check for binary response - not supported
	contentType := resp.Header.Get("Content-Type")
	if isBinaryContentType(contentType) {
		return nil, nil, errors.Errorf("binary response not supported (content-type: %s)", contentType)
	}

	// Parse JSON response
	var respJSON any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &respJSON); err != nil {
			// If not valid JSON, return as string
			respJSON = string(respBody)
		}
	}

	output := CallOutput{
		Status:   resp.StatusCode,
		Response: respJSON,
	}

	// Check for error response
	if resp.StatusCode >= 400 {
		if errMap, ok := respJSON.(map[string]any); ok {
			if msg, ok := errMap["message"].(string); ok {
				output.Error = msg
			} else if code, ok := errMap["code"].(string); ok {
				output.Error = code
			}
		}
		if output.Error == "" {
			output.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	text := formatCallOutput(output, endpoint)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
}

func formatCallOutput(output CallOutput, endpoint *EndpointInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s\n\n", endpoint.OperationID))

	if output.Error != "" {
		sb.WriteString(fmt.Sprintf("**Error** (HTTP %d): %s\n\n", output.Status, output.Error))
	} else {
		sb.WriteString(fmt.Sprintf("**Status:** %d OK\n\n", output.Status))
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

// Context key for storing the access token.
type accessTokenKey struct{}

// withAccessToken adds the access token to the context.
func withAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenKey{}, token)
}

// getAccessToken retrieves the access token from the context.
func getAccessToken(ctx context.Context) string {
	if token, ok := ctx.Value(accessTokenKey{}).(string); ok {
		return token
	}
	return ""
}
