package aireview

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// azureOpenAIEndpoints are the endpoint paths the settings page offers for Azure OpenAI.
var azureOpenAIEndpoints = []struct {
	kind string
	path string
}{
	{kind: "responses", path: "/openai/v1/responses"},
	{kind: "chat completions", path: "/openai/v1/chat/completions"},
}

const (
	azureOpenAIDeployment = "review-deployment"

	azureResponsesToolCall = `{"id": "resp_1", "object": "response", "status": "completed", "error": null, "incomplete_details": null, "output": [{"id": "rs_1", "type": "reasoning", "summary": [], "encrypted_content": "gAAAAB-encrypted-reasoning", "status": null}, {"id": "fc_1", "type": "function_call", "call_id": "call_1", "name": "search", "arguments": "{\"text\":\"orders\"}", "status": "completed"}], "usage": {"total_tokens": 40}, "content_filters": []}`
	azureResponsesPass     = `{"id": "resp_2", "object": "response", "status": "completed", "error": null, "incomplete_details": null, "output": [{"id": "msg_2", "type": "message", "role": "assistant", "status": "completed", "content": [{"type": "output_text", "text": "{\"findings\": []}", "annotations": []}]}], "usage": {"total_tokens": 60}, "content_filters": []}`
	azureChatToolCall      = `{"id": "chatcmpl-1", "object": "chat.completion", "prompt_filter_results": [], "choices": [{"index": 0, "finish_reason": "tool_calls", "message": {"role": "assistant", "content": null, "tool_calls": [{"id": "call_1", "type": "function", "function": {"name": "search", "arguments": "{\"text\":\"orders\"}"}}]}, "content_filter_results": {}}], "usage": {"total_tokens": 40}}`
	azureChatPass          = `{"id": "chatcmpl-2", "object": "chat.completion", "prompt_filter_results": [], "choices": [{"index": 0, "finish_reason": "stop", "message": {"role": "assistant", "content": "{\"findings\": []}"}, "content_filter_results": {}}], "usage": {"total_tokens": 60}}`

	azureRateLimit     = `{"error": {"code": "429", "message": "Requests to the Responses_Create Operation under Azure OpenAI API version v1 have been limited. Please retry after 2 seconds."}}`
	azureContentFilter = `{"error": {"code": "content_filter", "message": "The response was filtered due to the prompt triggering content management policy."}}`
)

func azureOpenAISetting(endpoint string) *storepb.AISetting {
	return &storepb.AISetting{
		Provider: storepb.AISetting_AZURE_OPENAI,
		Endpoint: endpoint,
		Model:    azureOpenAIDeployment,
		ApiKey:   testAPIKey,
	}
}

// azureOpenAIRequest is the part of a request the tests assert on.
type azureOpenAIRequest struct {
	path   string
	apiKey string
	body   map[string]any
}

// The scripted Model in reviewer_test.go stands in for ai.Chat, so this test
// holds the loop to the real adapter's Azure OpenAI wire format.
func TestSettingModelRoundTripsThroughAzureOpenAI(t *testing.T) {
	t.Parallel()

	replies := map[string][]string{
		"responses":        {azureResponsesToolCall, azureResponsesPass},
		"chat completions": {azureChatToolCall, azureChatPass},
	}
	for _, endpoint := range azureOpenAIEndpoints {
		t.Run(endpoint.kind, func(t *testing.T) {
			t.Parallel()

			var requests []azureOpenAIRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				request := azureOpenAIRequest{path: r.URL.Path, apiKey: r.Header.Get("api-key")}
				if err := json.NewDecoder(r.Body).Decode(&request.body); err != nil {
					t.Errorf("failed to decode the request body: %v", err)
				}
				requests = append(requests, request)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(replies[endpoint.kind][min(len(requests), 2)-1]))
			}))
			defer server.Close()

			model := NewModel(azureOpenAISetting(server.URL + endpoint.path))
			tools := &catalogTools{}
			result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: "DROP TABLE orders;"}, tools)
			require.NoError(t, err)
			require.Empty(t, result.Findings)
			require.Equal(t, 2, result.Calls)
			require.Equal(t, 100, result.TotalTokens)
			require.Equal(t, []string{`search({"text":"orders"})`}, tools.calls)

			searchOutput, err := (&catalogTools{}).Call(context.Background(), "search", `{"text":"orders"}`)
			require.NoError(t, err)
			require.Len(t, requests, 2)
			for _, request := range requests {
				require.Equal(t, endpoint.path, request.path)
				require.Equal(t, testAPIKey, request.apiKey)
				require.Equal(t, azureOpenAIDeployment, request.body["model"])
			}
			if endpoint.kind == "responses" {
				input, ok := requests[1].body["input"].([]any)
				require.True(t, ok)
				require.Contains(t, input, map[string]any{"type": "reasoning", "id": "rs_1", "summary": []any{}, "encrypted_content": "gAAAAB-encrypted-reasoning", "status": nil})
				require.Equal(t, map[string]any{"type": "function_call_output", "call_id": "call_1", "output": searchOutput}, input[len(input)-1])
			} else {
				messages, ok := requests[1].body["messages"].([]any)
				require.True(t, ok)
				require.Equal(t, map[string]any{"role": "tool", "tool_call_id": "call_1", "content": searchOutput}, messages[len(messages)-1])
			}
		})
	}
}

func TestSettingModelRetriesAzureOpenAIRateLimitsButNotGuardrailBlocks(t *testing.T) {
	t.Parallel()

	success := map[string]string{"responses": azureResponsesPass, "chat completions": azureChatPass}
	for _, endpoint := range azureOpenAIEndpoints {
		t.Run(endpoint.kind+" rate limit waits retry-after-ms", func(t *testing.T) {
			t.Parallel()

			provider := newScriptedProvider(t,
				providerReply{status: http.StatusTooManyRequests, retryAfterMs: "2000", body: azureRateLimit},
				providerReply{status: http.StatusOK, body: success[endpoint.kind]})
			recorder := &sleepRecorder{}
			model := &settingModel{setting: azureOpenAISetting(provider.url + endpoint.path), sleep: recorder.sleep}
			response, err := model.Chat(context.Background(), chatRequest())
			require.NoError(t, err)
			require.Equal(t, `{"findings": []}`, response.GetContent())
			require.Equal(t, int32(2), provider.requests.Load())
			require.Equal(t, []time.Duration{2 * time.Second}, recorder.waits)
		})

		t.Run(endpoint.kind+" prompt blocked by a guardrail", func(t *testing.T) {
			t.Parallel()

			provider := newScriptedProvider(t, providerReply{status: http.StatusBadRequest, body: azureContentFilter})
			recorder := &sleepRecorder{}
			model := &settingModel{setting: azureOpenAISetting(provider.url + endpoint.path), sleep: recorder.sleep}
			_, err := model.Chat(context.Background(), chatRequest())
			require.EqualError(t, err, "OpenAI API returned status 400: "+azureContentFilter)
			require.Equal(t, int32(1), provider.requests.Load())
			require.Empty(t, recorder.waits)
		})
	}
}

func TestReviewFailsOnAnAzureOpenAIReplyThatWasCutShort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		path  string
		reply string
	}{
		{
			name:  "responses out of output tokens before any text",
			path:  "/openai/v1/responses",
			reply: `{"id": "resp_1", "object": "response", "status": "incomplete", "error": null, "incomplete_details": {"reason": "max_output_tokens"}, "output": [{"id": "rs_1", "type": "reasoning", "summary": [], "encrypted_content": "gAAAAB", "status": null}], "usage": {"total_tokens": 25000}}`,
		},
		{
			name:  "responses output filtered after text that would pass",
			path:  "/openai/v1/responses",
			reply: `{"id": "resp_1", "object": "response", "status": "incomplete", "error": null, "incomplete_details": {"reason": "content_filter"}, "output": [{"id": "msg_1", "type": "message", "role": "assistant", "status": "incomplete", "content": [{"type": "output_text", "text": "{\"findings\": []}", "annotations": []}]}], "usage": {"total_tokens": 60}}`,
		},
		{
			name:  "chat completions output filtered",
			path:  "/openai/v1/chat/completions",
			reply: `{"id": "chatcmpl-1", "object": "chat.completion", "choices": [{"index": 0, "finish_reason": "content_filter", "message": {"role": "assistant", "content": null}, "content_filter_results": {"violence": {"filtered": true, "severity": "high"}}}], "usage": {"total_tokens": 60}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			provider := newScriptedProvider(t, providerReply{status: http.StatusOK, body: test.reply})
			model := NewModel(azureOpenAISetting(provider.url + test.path))
			result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: "DROP TABLE orders;"}, &catalogTools{})
			require.Error(t, err)
			require.Nil(t, result)
		})
	}
}
