package aireview

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// claudeRequest is the part of the Claude wire format the tests assert on.
type claudeRequest struct {
	System    string `json:"system"`
	MaxTokens int    `json:"max_tokens"`
	Messages  []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"messages"`
	Tools []struct {
		Name string `json:"name"`
	} `json:"tools"`
}

// The scripted Model in reviewer_test.go stands in for ai.Chat, so this test
// holds the loop to the real Claude adapter's wire format.
func TestSettingModelRoundTripsThroughClaude(t *testing.T) {
	t.Parallel()

	var requests []claudeRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request claudeRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		requests = append(requests, request)

		w.Header().Set("Content-Type", "application/json")
		reply := `{"type": "message", "role": "assistant", "content": [
			{"type": "thinking", "thinking": "", "signature": "sig-1"},
			{"type": "tool_use", "id": "toolu_1", "name": "read", "input": {"objects": [{"name": "orders", "schema": "public"}]}, "caller": {"type": "direct"}},
			{"type": "tool_use", "id": "toolu_2", "name": "search", "input": {"text": "orders"}, "caller": {"type": "direct"}}
		], "stop_reason": "tool_use", "usage": {"input_tokens": 2100, "output_tokens": 143, "output_tokens_details": {"thinking_tokens": 32}}}`
		if len(requests) == 2 {
			reply = `{"type": "message", "role": "assistant", "content": [{"type": "text", "text": "{\"findings\": [], \"notes\": []}"}], "stop_reason": "end_turn", "usage": {"input_tokens": 2436, "output_tokens": 21}}`
		}
		_, err := w.Write([]byte(reply))
		require.NoError(t, err)
	}))
	defer server.Close()

	model := NewModel(&storepb.AISetting{Provider: storepb.AISetting_CLAUDE, Endpoint: server.URL, Model: "claude-sonnet-5", ApiKey: "test-key"})
	tools := &catalogTools{}
	result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: "DROP TABLE orders;"}, tools)
	require.NoError(t, err)
	require.Empty(t, result.Findings)
	require.Equal(t, 2, result.Calls)
	require.Equal(t, 2100+143+2436+21, result.TotalTokens)
	require.Len(t, tools.calls, 2)

	require.Len(t, requests, 2)
	first := requests[0]
	require.Equal(t, 16000, first.MaxTokens, "the SQL editor's 4096 is too little for thinking and a long findings list")
	require.Contains(t, first.System, "# Answer format")
	require.Len(t, first.Messages, 1)
	require.Equal(t, "user", first.Messages[0].Role)
	require.Len(t, first.Tools, 2)

	second := requests[1]
	require.Equal(t, 16000, second.MaxTokens)
	require.Len(t, second.Messages, 3)
	require.Equal(t, "assistant", second.Messages[1].Role)
	require.Contains(t, string(second.Messages[1].Content), `"signature":"sig-1"`, "Claude rejects a tool turn that returns without its thinking")
	require.Equal(t, "user", second.Messages[2].Role)
	var results []struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
	}
	require.NoError(t, json.Unmarshal(second.Messages[2].Content, &results))
	require.Len(t, results, 2)
	require.Equal(t, "toolu_1", results[0].ToolUseID)
	require.Equal(t, "toolu_2", results[1].ToolUseID)
}
