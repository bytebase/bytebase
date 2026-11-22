package ai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	stopUnspecified       = v1pb.AIChatResponse_STOP_REASON_UNSPECIFIED
	stopTurnCompleted     = v1pb.AIChatResponse_TURN_COMPLETED
	stopTokenLimit        = v1pb.AIChatResponse_TOKEN_LIMIT
	stopBlocked           = v1pb.AIChatResponse_BLOCKED
	stopMalformedToolCall = v1pb.AIChatResponse_MALFORMED_TOOL_CALL
	stopOther             = v1pb.AIChatResponse_OTHER
)

// chatWithScriptedReply sends one user message through Chat to a server that
// answers every request with reply. endpointPath selects the OpenAI wire format.
func chatWithScriptedReply(t *testing.T, provider storepb.AISetting_Provider, endpointPath string, reply string) (*v1pb.AIChatResponse, error) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(reply))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	content := "Review this change."
	return Chat(
		context.Background(),
		&storepb.AISetting{Provider: provider, Endpoint: server.URL + endpointPath, Model: "test-model", ApiKey: "test-key"},
		&v1pb.AIChatRequest{Messages: []*v1pb.AIChatMessage{{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &content}}},
	)
}

func TestOpenAIChatCompletionsStopReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		provider    storepb.AISetting_Provider
		reply       string
		want        v1pb.AIChatResponse_StopReason
		wantContent string
	}{
		{
			name:  "stop",
			reply: `{"choices": [{"message": {"role": "assistant", "content": "done"}, "finish_reason": "stop"}]}`,
			want:  stopTurnCompleted,
		},
		{
			name:  "tool_calls",
			reply: `{"choices": [{"message": {"role": "assistant", "content": null, "tool_calls": [{"id": "call_1", "type": "function", "function": {"name": "read", "arguments": "{}"}}]}, "finish_reason": "tool_calls"}]}`,
			want:  stopTurnCompleted,
		},
		{
			name:  "function_call",
			reply: `{"choices": [{"message": {"role": "assistant", "content": null}, "finish_reason": "function_call"}]}`,
			want:  stopTurnCompleted,
		},
		{
			name:        "length keeps the truncated text",
			reply:       `{"choices": [{"message": {"role": "assistant", "content": "{\"findings\": ["}, "finish_reason": "length"}]}`,
			want:        stopTokenLimit,
			wantContent: `{"findings": [`,
		},
		{
			name:  "length with tool calls",
			reply: `{"choices": [{"message": {"role": "assistant", "content": null, "tool_calls": [{"id": "call_1", "type": "function", "function": {"name": "read", "arguments": "{\"objects\": ["}}]}, "finish_reason": "length"}]}`,
			want:  stopTokenLimit,
		},
		{
			name:  "content_filter",
			reply: `{"choices": [{"message": {"role": "assistant", "content": null}, "finish_reason": "content_filter"}]}`,
			want:  stopBlocked,
		},
		{
			name:     "content_filter from Azure OpenAI",
			provider: storepb.AISetting_AZURE_OPENAI,
			reply:    `{"choices": [{"message": {"role": "assistant", "content": ""}, "finish_reason": "content_filter"}]}`,
			want:     stopBlocked,
		},
		{
			name:  "refusal beside a stop",
			reply: `{"choices": [{"message": {"role": "assistant", "content": null, "refusal": "I can't help with that request."}, "finish_reason": "stop"}]}`,
			want:  stopBlocked,
		},
		{
			name:  "null refusal is not a refusal",
			reply: `{"choices": [{"message": {"role": "assistant", "content": "done", "refusal": null}, "finish_reason": "stop"}]}`,
			want:  stopTurnCompleted,
		},
		{
			name:  "empty refusal is not a refusal",
			reply: `{"choices": [{"message": {"role": "assistant", "content": "done", "refusal": ""}, "finish_reason": "stop"}]}`,
			want:  stopTurnCompleted,
		},
		{
			name:  "null finish_reason",
			reply: `{"choices": [{"message": {"role": "assistant", "content": "done"}, "finish_reason": null}]}`,
			want:  stopUnspecified,
		},
		{
			name:  "no finish_reason",
			reply: `{"choices": [{"message": {"role": "assistant", "content": "done"}}]}`,
			want:  stopUnspecified,
		},
		{
			name:  "no choices",
			reply: `{"choices": []}`,
			want:  stopUnspecified,
		},
		{
			name:  "unknown finish_reason",
			reply: `{"choices": [{"message": {"role": "assistant", "content": "done"}, "finish_reason": "insufficient_system_resource"}]}`,
			want:  stopOther,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			provider := tc.provider
			if provider == storepb.AISetting_PROVIDER_UNSPECIFIED {
				provider = storepb.AISetting_OPEN_AI
			}
			response, err := chatWithScriptedReply(t, provider, "/chat/completions", tc.reply)
			require.NoError(t, err)
			require.Equal(t, tc.want, response.GetStopReason())
			if tc.wantContent != "" {
				require.Equal(t, tc.wantContent, response.GetContent())
			}
		})
	}
}

func TestOpenAIResponsesStopReason(t *testing.T) {
	t.Parallel()

	const textOutput = `"output": [{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "{\"findings\": ["}]}]`
	const refusalOutput = `"output": [{"type": "message", "role": "assistant", "content": [{"type": "refusal", "refusal": "I can't help with that request."}]}]`
	tests := []struct {
		name        string
		reply       string
		want        v1pb.AIChatResponse_StopReason
		wantContent string
		wantErrText string
	}{
		{
			name:  "completed",
			reply: `{"status": "completed", ` + textOutput + `}`,
			want:  stopTurnCompleted,
		},
		{
			name:  "completed with a function call",
			reply: `{"status": "completed", "output": [{"type": "function_call", "call_id": "call_1", "name": "read", "arguments": "{}"}]}`,
			want:  stopTurnCompleted,
		},
		{
			name:  "no status",
			reply: `{` + textOutput + `}`,
			want:  stopUnspecified,
		},
		{
			name:        "refusal part of a completed response keeps the refusal text",
			reply:       `{"status": "completed", ` + refusalOutput + `}`,
			want:        stopBlocked,
			wantContent: "I can't help with that request.",
		},
		{
			name:  "refusal part without a status",
			reply: `{` + refusalOutput + `}`,
			want:  stopBlocked,
		},
		{
			name:  "failed without an error object",
			reply: `{"status": "failed", ` + textOutput + `}`,
			want:  stopOther,
		},
		{
			name:  "cancelled",
			reply: `{"status": "cancelled", ` + textOutput + `}`,
			want:  stopOther,
		},
		{
			name:  "queued",
			reply: `{"status": "queued", "output": []}`,
			want:  stopOther,
		},
		{
			name:  "in_progress",
			reply: `{"status": "in_progress", "output": []}`,
			want:  stopOther,
		},
		{
			name:  "unknown status",
			reply: `{"status": "expired", ` + textOutput + `}`,
			want:  stopOther,
		},
		{
			name:        "incomplete at max_output_tokens is an error",
			reply:       `{"status": "incomplete", "incomplete_details": {"reason": "max_output_tokens"}, ` + textOutput + `}`,
			wantErrText: "OpenAI Responses API returned an incomplete response: max_output_tokens",
		},
		{
			name:        "incomplete for content_filter is an error",
			reply:       `{"status": "incomplete", "incomplete_details": {"reason": "content_filter"}, "output": []}`,
			wantErrText: "OpenAI Responses API returned an incomplete response: content_filter",
		},
		{
			name:        "incomplete with a refusal part is an error",
			reply:       `{"status": "incomplete", "incomplete_details": {"reason": "content_filter"}, ` + refusalOutput + `}`,
			wantErrText: "OpenAI Responses API returned an incomplete response: content_filter",
		},
		{
			name:        "incomplete at max_messages is an error",
			reply:       `{"status": "incomplete", "incomplete_details": {"reason": "max_messages"}, ` + textOutput + `}`,
			wantErrText: "OpenAI Responses API returned an incomplete response: max_messages",
		},
		{
			name:        "incomplete when steered is an error",
			reply:       `{"status": "incomplete", "incomplete_details": {"reason": "steered"}, ` + textOutput + `}`,
			wantErrText: "OpenAI Responses API returned an incomplete response: steered",
		},
		{
			name:        "incomplete without details is an error",
			reply:       `{"status": "incomplete", ` + textOutput + `}`,
			wantErrText: "OpenAI Responses API returned an incomplete response: ",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, err := chatWithScriptedReply(t, storepb.AISetting_OPEN_AI, "/responses", tc.reply)
			if tc.wantErrText != "" {
				require.ErrorContains(t, err, tc.wantErrText)
				require.Nil(t, response)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, response.GetStopReason())
			if tc.wantContent != "" {
				require.Equal(t, tc.wantContent, response.GetContent())
			}
		})
	}
}

func TestClaudeStopReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		reply       string
		want        v1pb.AIChatResponse_StopReason
		wantContent string
	}{
		{
			name:  "end_turn",
			reply: `{"content": [{"type": "text", "text": "done"}], "stop_reason": "end_turn"}`,
			want:  stopTurnCompleted,
		},
		{
			name:  "stop_sequence",
			reply: `{"content": [{"type": "text", "text": "done"}], "stop_reason": "stop_sequence"}`,
			want:  stopTurnCompleted,
		},
		{
			name:  "tool_use",
			reply: `{"content": [{"type": "tool_use", "id": "toolu_1", "name": "read", "input": {}}], "stop_reason": "tool_use"}`,
			want:  stopTurnCompleted,
		},
		{
			name:        "max_tokens keeps the truncated text",
			reply:       `{"content": [{"type": "text", "text": "{\"findings\": ["}], "stop_reason": "max_tokens"}`,
			want:        stopTokenLimit,
			wantContent: `{"findings": [`,
		},
		{
			name:  "max_tokens with a tool call",
			reply: `{"content": [{"type": "tool_use", "id": "toolu_1", "name": "read", "input": {}}], "stop_reason": "max_tokens"}`,
			want:  stopTokenLimit,
		},
		{
			name:  "model_context_window_exceeded",
			reply: `{"content": [{"type": "text", "text": "{\"findings\": ["}], "stop_reason": "model_context_window_exceeded"}`,
			want:  stopTokenLimit,
		},
		{
			name:        "refusal keeps the text before it",
			reply:       `{"content": [{"type": "text", "text": "I can't help with that request."}], "stop_reason": "refusal", "stop_details": {"type": "refusal", "category": "cyber", "explanation": null}}`,
			want:        stopBlocked,
			wantContent: "I can't help with that request.",
		},
		{
			name:  "refusal without content",
			reply: `{"content": [], "stop_reason": "refusal"}`,
			want:  stopBlocked,
		},
		{
			name:  "pause_turn",
			reply: `{"content": [{"type": "text", "text": "done"}], "stop_reason": "pause_turn"}`,
			want:  stopOther,
		},
		{
			name:  "null stop_reason",
			reply: `{"content": [{"type": "text", "text": "done"}], "stop_reason": null}`,
			want:  stopUnspecified,
		},
		{
			name:  "no stop_reason",
			reply: `{"content": [{"type": "text", "text": "done"}]}`,
			want:  stopUnspecified,
		},
		{
			name:  "unknown stop_reason",
			reply: `{"content": [{"type": "text", "text": "done"}], "stop_reason": "compaction"}`,
			want:  stopOther,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, err := chatWithScriptedReply(t, storepb.AISetting_CLAUDE, "", tc.reply)
			require.NoError(t, err)
			require.Equal(t, tc.want, response.GetStopReason())
			if tc.wantContent != "" {
				require.Equal(t, tc.wantContent, response.GetContent())
			}
		})
	}
}

func TestGeminiStopReasonForEveryFinishReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		finishReason string
		want         v1pb.AIChatResponse_StopReason
	}{
		{finishReason: "FINISH_REASON_UNSPECIFIED", want: stopUnspecified},
		{finishReason: "STOP", want: stopTurnCompleted},
		{finishReason: "MAX_TOKENS", want: stopTokenLimit},
		{finishReason: "CONTINUATION", want: stopTokenLimit},
		{finishReason: "SAFETY", want: stopBlocked},
		{finishReason: "RECITATION", want: stopBlocked},
		{finishReason: "LANGUAGE", want: stopBlocked},
		{finishReason: "BLOCKLIST", want: stopBlocked},
		{finishReason: "PROHIBITED_CONTENT", want: stopBlocked},
		{finishReason: "SPII", want: stopBlocked},
		{finishReason: "IMAGE_SAFETY", want: stopBlocked},
		{finishReason: "IMAGE_PROHIBITED_CONTENT", want: stopBlocked},
		{finishReason: "IMAGE_RECITATION", want: stopBlocked},
		{finishReason: "ESCALATION", want: stopBlocked},
		{finishReason: "PUP_LIMITED_DISABLED", want: stopBlocked},
		{finishReason: "MALFORMED_FUNCTION_CALL", want: stopMalformedToolCall},
		{finishReason: "OTHER", want: stopOther},
		{finishReason: "IMAGE_OTHER", want: stopOther},
		{finishReason: "NO_IMAGE", want: stopOther},
		{finishReason: "UNEXPECTED_TOOL_CALL", want: stopOther},
		{finishReason: "TOO_MANY_TOOL_CALLS", want: stopOther},
		{finishReason: "MISSING_THOUGHT_SIGNATURE", want: stopOther},
		{finishReason: "MALFORMED_RESPONSE", want: stopOther},
		{finishReason: "A_REASON_ADDED_LATER", want: stopOther},
	}
	for _, tc := range tests {
		t.Run(tc.finishReason, func(t *testing.T) {
			t.Parallel()

			reply := fmt.Sprintf(`{"candidates": [{"content": {"parts": [{"text": "{\"findings\": ["}]}, "finishReason": %q}]}`, tc.finishReason)
			response, err := chatWithScriptedReply(t, storepb.AISetting_GEMINI, "", reply)
			require.NoError(t, err)
			require.Equal(t, tc.want, response.GetStopReason())
			require.Equal(t, `{"findings": [`, response.GetContent())
		})
	}
}

func TestGeminiStopReasonForCandidateShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		reply string
		want  v1pb.AIChatResponse_StopReason
	}{
		{
			name:  "no finishReason",
			reply: `{"candidates": [{"content": {"parts": [{"text": "done"}]}}]}`,
			want:  stopUnspecified,
		},
		{
			name:  "function call that stops with STOP",
			reply: `{"candidates": [{"content": {"parts": [{"functionCall": {"name": "read", "args": {}}}]}, "finishReason": "STOP"}]}`,
			want:  stopTurnCompleted,
		},
		{
			name:  "safety block without content",
			reply: `{"candidates": [{"finishReason": "SAFETY", "safetyRatings": [{"category": "HARM_CATEGORY_HATE_SPEECH", "probability": "HIGH"}]}]}`,
			want:  stopBlocked,
		},
		{
			name:  "malformed function call without content",
			reply: `{"candidates": [{"content": {}, "finishReason": "MALFORMED_FUNCTION_CALL", "finishMessage": "Malformed function call: read(objects="}]}`,
			want:  stopMalformedToolCall,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, err := chatWithScriptedReply(t, storepb.AISetting_GEMINI, "", tc.reply)
			require.NoError(t, err)
			require.Equal(t, tc.want, response.GetStopReason())
		})
	}
}

func TestGeminiStopReasonWithoutACandidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		reply string
		want  v1pb.AIChatResponse_StopReason
	}{
		{
			name:  "no prompt feedback",
			reply: `{"usageMetadata": {"totalTokenCount": 12}}`,
			want:  stopUnspecified,
		},
		{
			name:  "prompt feedback without a block reason",
			reply: `{"promptFeedback": {"safetyRatings": []}}`,
			want:  stopUnspecified,
		},
		{
			name:  "prompt blocked with BLOCK_REASON_UNSPECIFIED",
			reply: `{"promptFeedback": {"blockReason": "BLOCK_REASON_UNSPECIFIED"}}`,
			want:  stopUnspecified,
		},
		{
			name:  "prompt blocked for SAFETY",
			reply: `{"promptFeedback": {"blockReason": "SAFETY"}}`,
			want:  stopBlocked,
		},
		{
			name:  "prompt blocked for OTHER",
			reply: `{"promptFeedback": {"blockReason": "OTHER"}}`,
			want:  stopBlocked,
		},
		{
			name:  "prompt blocked for BLOCKLIST",
			reply: `{"promptFeedback": {"blockReason": "BLOCKLIST"}}`,
			want:  stopBlocked,
		},
		{
			name:  "prompt blocked for PROHIBITED_CONTENT",
			reply: `{"promptFeedback": {"blockReason": "PROHIBITED_CONTENT"}}`,
			want:  stopBlocked,
		},
		{
			name:  "prompt blocked for IMAGE_SAFETY",
			reply: `{"promptFeedback": {"blockReason": "IMAGE_SAFETY"}}`,
			want:  stopBlocked,
		},
		{
			name:  "unknown block reason",
			reply: `{"promptFeedback": {"blockReason": "A_REASON_ADDED_LATER"}}`,
			want:  stopOther,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, err := chatWithScriptedReply(t, storepb.AISetting_GEMINI, "", tc.reply)
			require.NoError(t, err)
			require.Equal(t, tc.want, response.GetStopReason())
		})
	}
}
