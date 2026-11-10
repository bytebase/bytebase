package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// The requests and replies in this file follow the Azure OpenAI v1 API pages on
// learn.microsoft.com: api-version-lifecycle, how-to/responses,
// how-to/function-calling, how-to/reasoning, and how-to/quota.

const (
	azureDeployment = "review-deployment"
	azureAPIKey     = "azure-test-key"
)

// azureRequest is one request the fake Azure resource received.
type azureRequest struct {
	method      string
	path        string
	query       string
	apiKey      string
	auth        string
	contentType string
	body        map[string]any
}

// newAzureServer records each request and answers it with the next reply.
func newAzureServer(t *testing.T, replies ...string) (*httptest.Server, *[]azureRequest) {
	t.Helper()
	var requests []azureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := azureRequest{
			method:      r.Method,
			path:        r.URL.Path,
			query:       r.URL.RawQuery,
			apiKey:      r.Header.Get("api-key"),
			auth:        r.Header.Get("Authorization"),
			contentType: r.Header.Get("Content-Type"),
		}
		if err := json.NewDecoder(r.Body).Decode(&request.body); err != nil {
			t.Errorf("failed to decode the request body: %v", err)
		}
		requests = append(requests, request)
		if len(requests) > len(replies) {
			t.Errorf("unexpected request %d", len(requests))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(replies[len(requests)-1]))
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func azureSetting(endpoint string) *storepb.AISetting {
	return &storepb.AISetting{
		Provider: storepb.AISetting_AZURE_OPENAI,
		Endpoint: endpoint,
		Model:    azureDeployment,
		ApiKey:   azureAPIKey,
	}
}

func azureSearchTool() *v1pb.AIChatToolDefinition {
	return &v1pb.AIChatToolDefinition{
		Name:             "search",
		Description:      "Find objects by name.",
		ParametersSchema: `{"type": "object", "properties": {"text": {"type": "string"}}, "required": ["text"]}`,
	}
}

var azureSearchParameters = map[string]any{
	"type":       "object",
	"properties": map[string]any{"text": map[string]any{"type": "string"}},
	"required":   []any{"text"},
}

func azureReviewMessages() []*v1pb.AIChatMessage {
	system := "You review SQL."
	user := "DROP TABLE orders;"
	return []*v1pb.AIChatMessage{
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM, Content: &system},
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &user},
	}
}

// requireAzureHeaders asserts what Azure documents for every call: a POST with
// the key in the api-key header, a JSON body, and the deployment name as the model.
func requireAzureHeaders(t *testing.T, request azureRequest) {
	t.Helper()
	require.Equal(t, http.MethodPost, request.method)
	require.Equal(t, azureAPIKey, request.apiKey)
	require.Empty(t, request.auth)
	require.Equal(t, "application/json", request.contentType)
	require.Equal(t, azureDeployment, request.body["model"])
}

const (
	azureResponsesText = `{
		"id": "resp_2",
		"object": "response",
		"created_at": 1741369940.0,
		"status": "completed",
		"error": null,
		"incomplete_details": null,
		"model": "gpt-5.6-terra",
		"output": [{
			"id": "msg_2",
			"type": "message",
			"role": "assistant",
			"status": "completed",
			"content": [{"type": "output_text", "text": "{\"findings\": []}", "annotations": [], "logprobs": null}]
		}],
		"parallel_tool_calls": true,
		"store": false,
		"usage": {"input_tokens": 140, "input_tokens_details": {"cached_tokens": 0}, "output_tokens": 10, "output_tokens_details": {"reasoning_tokens": 0}, "total_tokens": 150},
		"content_filters": [{"source_type": "prompt", "blocked": false, "content_filter_results": {"hate": {"filtered": false, "severity": "safe"}}}]
	}`
	azureChatCompletionsText = `{
		"id": "chatcmpl-2",
		"object": "chat.completion",
		"created": 1741369940,
		"model": "gpt-4o-2024-11-20",
		"prompt_filter_results": [{"prompt_index": 0, "content_filter_results": {"hate": {"filtered": false, "severity": "safe"}}}],
		"choices": [{
			"index": 0,
			"finish_reason": "stop",
			"logprobs": null,
			"message": {"role": "assistant", "content": "{\"findings\": []}", "refusal": null},
			"content_filter_results": {"hate": {"filtered": false, "severity": "safe"}}
		}],
		"usage": {"completion_tokens": 10, "prompt_tokens": 140, "total_tokens": 150}
	}`
)

func TestChatAzureOpenAISendsTheDocumentedRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		endpoint      string
		wantPath      string
		wantQuery     string
		wantResponses bool
	}{
		{
			name:          "v1 responses, the default the settings page offers",
			endpoint:      "/openai/v1/responses",
			wantPath:      "/openai/v1/responses",
			wantResponses: true,
		},
		{
			name:     "v1 chat completions",
			endpoint: "/openai/v1/chat/completions",
			wantPath: "/openai/v1/chat/completions",
		},
		{
			name:          "v1 responses with the optional api-version",
			endpoint:      "/openai/v1/responses?api-version=preview",
			wantPath:      "/openai/v1/responses",
			wantQuery:     "api-version=preview",
			wantResponses: true,
		},
		{
			name:      "dated chat completions with the deployment in the path",
			endpoint:  "/openai/deployments/review-deployment/chat/completions?api-version=2024-10-21",
			wantPath:  "/openai/deployments/review-deployment/chat/completions",
			wantQuery: "api-version=2024-10-21",
		},
		{
			name:          "dated responses",
			endpoint:      "/openai/responses?api-version=2025-04-01-preview",
			wantPath:      "/openai/responses",
			wantQuery:     "api-version=2025-04-01-preview",
			wantResponses: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			reply := azureChatCompletionsText
			if test.wantResponses {
				reply = azureResponsesText
			}
			server, requests := newAzureServer(t, reply)
			response, err := Chat(context.Background(), azureSetting(server.URL+test.endpoint), &v1pb.AIChatRequest{
				Messages:        azureReviewMessages(),
				ToolDefinitions: []*v1pb.AIChatToolDefinition{azureSearchTool()},
			})
			require.NoError(t, err)
			require.Equal(t, `{"findings": []}`, response.GetContent())
			require.EqualValues(t, 150, response.GetUsage().GetTotalTokens())

			require.Len(t, *requests, 1)
			request := (*requests)[0]
			requireAzureHeaders(t, request)
			require.Equal(t, test.wantPath, request.path)
			require.Equal(t, test.wantQuery, request.query)
			if test.wantResponses {
				require.Contains(t, request.body, "input")
				require.NotContains(t, request.body, "messages")
				require.Equal(t, false, request.body["store"])
				require.Equal(t, []any{"reasoning.encrypted_content"}, request.body["include"])
			} else {
				require.Contains(t, request.body, "messages")
				require.NotContains(t, request.body, "input")
			}
		})
	}
}

func TestChatAzureOpenAIResponsesRoundTripsAToolCall(t *testing.T) {
	t.Parallel()

	reasoning := map[string]any{
		"id":                "rs_1",
		"type":              "reasoning",
		"summary":           []any{},
		"encrypted_content": "gAAAAB-encrypted-reasoning",
		"status":            nil,
	}
	functionCall := map[string]any{
		"id":        "fc_1",
		"type":      "function_call",
		"call_id":   "call_1",
		"name":      "search",
		"arguments": `{"text":"orders"}`,
		"status":    "completed",
	}
	toolCallReply, err := json.Marshal(map[string]any{
		"id":                  "resp_1",
		"object":              "response",
		"created_at":          1741369938.0,
		"status":              "completed",
		"error":               nil,
		"incomplete_details":  nil,
		"model":               "gpt-5.6-terra",
		"output":              []any{reasoning, functionCall},
		"parallel_tool_calls": true,
		"store":               false,
		"usage": map[string]any{
			"input_tokens":          90,
			"input_tokens_details":  map[string]any{"cached_tokens": 0},
			"output_tokens":         30,
			"output_tokens_details": map[string]any{"reasoning_tokens": 18},
			"total_tokens":          120,
		},
		"content_filters": []any{map[string]any{"source_type": "prompt", "blocked": false}},
	})
	require.NoError(t, err)
	server, requests := newAzureServer(t, string(toolCallReply), azureResponsesText)
	setting := azureSetting(server.URL + "/openai/v1/responses")
	messages := azureReviewMessages()
	tools := []*v1pb.AIChatToolDefinition{azureSearchTool()}

	first, err := Chat(context.Background(), setting, &v1pb.AIChatRequest{Messages: messages, ToolDefinitions: tools})
	require.NoError(t, err)
	require.Nil(t, first.Content)
	require.EqualValues(t, 120, first.GetUsage().GetTotalTokens())
	require.Len(t, first.ToolCalls, 1)
	require.Equal(t, "call_1", first.ToolCalls[0].GetId())
	require.Equal(t, "search", first.ToolCalls[0].GetName())
	require.Equal(t, `{"text":"orders"}`, first.ToolCalls[0].GetArguments())

	toolResult := `[{"kind": "table", "schema": "public", "name": "orders"}]`
	messages = append(messages,
		&v1pb.AIChatMessage{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT, ToolCalls: first.ToolCalls},
		&v1pb.AIChatMessage{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL, Content: &toolResult, ToolCallId: &first.ToolCalls[0].Id},
	)
	second, err := Chat(context.Background(), setting, &v1pb.AIChatRequest{Messages: messages, ToolDefinitions: tools})
	require.NoError(t, err)
	require.Equal(t, `{"findings": []}`, second.GetContent())
	require.Empty(t, second.ToolCalls)

	require.Len(t, *requests, 2)
	for _, request := range *requests {
		requireAzureHeaders(t, request)
		require.Equal(t, "/openai/v1/responses", request.path)
		require.Empty(t, request.query)
		require.Equal(t, false, request.body["store"])
		require.Equal(t, []any{"reasoning.encrypted_content"}, request.body["include"])
		require.Equal(t, []any{map[string]any{
			"type":        "function",
			"name":        "search",
			"description": "Find objects by name.",
			"parameters":  azureSearchParameters,
		}}, request.body["tools"])
	}
	conversation := []any{
		map[string]any{"role": "developer", "content": "You review SQL."},
		map[string]any{"role": "user", "content": "DROP TABLE orders;"},
	}
	require.Equal(t, conversation, (*requests)[0].body["input"])
	// With store false Azure keeps nothing, so every output item goes back as
	// received, the encrypted reasoning included.
	require.Equal(t, append(conversation,
		reasoning,
		functionCall,
		map[string]any{"type": "function_call_output", "call_id": "call_1", "output": toolResult},
	), (*requests)[1].body["input"])
}

func TestChatAzureOpenAIChatCompletionsRoundTripsAToolCall(t *testing.T) {
	t.Parallel()

	toolCall := map[string]any{
		"id":       "call_1",
		"type":     "function",
		"function": map[string]any{"name": "search", "arguments": `{"text":"orders"}`},
	}
	toolCallReply, err := json.Marshal(map[string]any{
		"id":      "chatcmpl-1",
		"object":  "chat.completion",
		"created": 1741369938,
		"model":   "gpt-4o-2024-11-20",
		"prompt_filter_results": []any{map[string]any{
			"prompt_index":           0,
			"content_filter_results": map[string]any{"hate": map[string]any{"filtered": false, "severity": "safe"}},
		}},
		"choices": []any{map[string]any{
			"index":         0,
			"finish_reason": "tool_calls",
			"logprobs":      nil,
			"message": map[string]any{
				"role":       "assistant",
				"content":    nil,
				"refusal":    nil,
				"tool_calls": []any{toolCall},
			},
			"content_filter_results": map[string]any{},
		}},
		"usage": map[string]any{"completion_tokens": 20, "prompt_tokens": 100, "total_tokens": 120},
	})
	require.NoError(t, err)
	server, requests := newAzureServer(t, string(toolCallReply), azureChatCompletionsText)
	setting := azureSetting(server.URL + "/openai/v1/chat/completions")
	messages := azureReviewMessages()
	tools := []*v1pb.AIChatToolDefinition{azureSearchTool()}

	first, err := Chat(context.Background(), setting, &v1pb.AIChatRequest{Messages: messages, ToolDefinitions: tools})
	require.NoError(t, err)
	require.Nil(t, first.Content)
	require.EqualValues(t, 120, first.GetUsage().GetTotalTokens())
	require.Len(t, first.ToolCalls, 1)
	require.Equal(t, "call_1", first.ToolCalls[0].GetId())
	require.Equal(t, "search", first.ToolCalls[0].GetName())
	require.Equal(t, `{"text":"orders"}`, first.ToolCalls[0].GetArguments())

	toolResult := `[{"kind": "table", "schema": "public", "name": "orders"}]`
	messages = append(messages,
		&v1pb.AIChatMessage{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT, ToolCalls: first.ToolCalls},
		&v1pb.AIChatMessage{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL, Content: &toolResult, ToolCallId: &first.ToolCalls[0].Id},
	)
	second, err := Chat(context.Background(), setting, &v1pb.AIChatRequest{Messages: messages, ToolDefinitions: tools})
	require.NoError(t, err)
	require.Equal(t, `{"findings": []}`, second.GetContent())
	require.Empty(t, second.ToolCalls)

	require.Len(t, *requests, 2)
	for _, request := range *requests {
		requireAzureHeaders(t, request)
		require.Equal(t, "/openai/v1/chat/completions", request.path)
		require.Empty(t, request.query)
		require.Equal(t, []any{map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        "search",
				"description": "Find objects by name.",
				"parameters":  azureSearchParameters,
			},
		}}, request.body["tools"])
	}
	conversation := []any{
		map[string]any{"role": "system", "content": "You review SQL."},
		map[string]any{"role": "user", "content": "DROP TABLE orders;"},
	}
	require.Equal(t, conversation, (*requests)[0].body["messages"])
	require.Equal(t, append(conversation,
		map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{toolCall}},
		map[string]any{"role": "tool", "content": toolResult, "tool_call_id": "call_1"},
	), (*requests)[1].body["messages"])
}

func TestChatAzureOpenAIReadsTheDocumentedFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		header map[string]string
		body   string
		want   time.Duration
	}{
		{
			name:   "rate limit with retry-after-ms",
			status: http.StatusTooManyRequests,
			header: map[string]string{"retry-after-ms": "2000", "x-ratelimit-remaining-requests": "0"},
			body:   `{"error": {"code": "429", "message": "Requests to the Responses_Create Operation under Azure OpenAI API version v1 have been limited. Please retry after 2 seconds."}}`,
			want:   2 * time.Second,
		},
		{
			name:   "capacity throttling with retry-after-ms",
			status: http.StatusTooManyRequests,
			header: map[string]string{"retry-after-ms": "500"},
			body:   `{"error": {"code": "429", "message": "The service is temporarily unable to process your request. Please try again later."}}`,
			want:   500 * time.Millisecond,
		},
		{
			name:   "prompt blocked by a guardrail",
			status: http.StatusBadRequest,
			body:   `{"error": {"code": "content_filter", "message": "The response was filtered due to the prompt triggering content management policy."}}`,
		},
	}
	for _, endpoint := range []struct {
		kind string
		path string
	}{
		{kind: "responses", path: "/openai/v1/responses"},
		{kind: "chat completions", path: "/openai/v1/chat/completions"},
	} {
		for _, test := range tests {
			t.Run(fmt.Sprintf("%s %s", endpoint.kind, test.name), func(t *testing.T) {
				t.Parallel()

				server := failingProvider(t, test.status, test.header, test.body)
				_, err := Chat(context.Background(), azureSetting(server.URL+endpoint.path), &v1pb.AIChatRequest{Messages: azureReviewMessages()})

				var httpErr *HTTPError
				require.ErrorAs(t, err, &httpErr)
				require.Equal(t, HTTPError{
					Provider:   "OpenAI",
					StatusCode: test.status,
					RetryAfter: test.want,
					Body:       test.body,
				}, *httpErr, "every 429 Azure documents is cleared by a wait, so none is a used-up quota")
			})
		}
	}
}
