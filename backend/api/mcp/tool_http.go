package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// apiResponse holds the HTTP response from an internal API call.
type apiResponse struct {
	Status  int
	Body    json.RawMessage
	Headers http.Header
}

// apiRequest executes an HTTP POST to an internal Bytebase API endpoint.
// It handles URL building, auth forwarding, Connect-RPC headers, and JSON marshaling.
func (s *Server) apiRequest(ctx context.Context, path string, body any) (*apiResponse, error) {
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal request body")
		}
	} else {
		bodyBytes = []byte("{}")
	}

	url := fmt.Sprintf("http://localhost:%d%s", s.profile.Port, path)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Connect-Protocol-Version", "1")

	if token := getAccessToken(ctx); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute API request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	return &apiResponse{
		Status:  resp.StatusCode,
		Body:    json.RawMessage(respBody),
		Headers: resp.Header,
	}, nil
}

// parseError extracts the error message from an API error response body.
func parseError(body json.RawMessage) string {
	var errMap map[string]any
	if json.Unmarshal(body, &errMap) != nil {
		return ""
	}
	if msg, ok := errMap["message"].(string); ok {
		return msg
	}
	if code, ok := errMap["code"].(string); ok {
		return code
	}
	return ""
}

// toolError is a structured error returned by MCP tools.
type toolError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func (e *toolError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Suggestion)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
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
