package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// claudeParallelToolUseReply is a claude-sonnet-5 reply that calls two tools at once.
const claudeParallelToolUseReply = `{"model":"claude-sonnet-5","id":"msg_011Cg1p54tbDVDRbm4fQJkuW","type":"message","role":"assistant","content":[{"type":"thinking","thinking":"","signature":"EpUECscCCBIYAipAigTKC74ZSFufS5ml4CkcNdWZ4uxv2BwwxbQA8KazXjRTqt55Jjsq6SA0BdiW/FgAY4ofAsZ+pf32L5SLABPh1zIPY2xhdWRlLXNvbm5ldC01OABCCHRoaW5raW5nWiQxYTU5NTdlZS0wNzJhLTRhZjItYjdhYS0wYmU3MDNjNWMxMzCaAREKD2NsYXVkZS1zb25uZXQtNagB3fm81gbCAT0IBRAAGg8BD7kCU+gDTzcFsw4EJq0gAjIEEnHPJToEjA2WMkIE1N6MUEoEETUNSVACWgTZBeFWYAJoAXAB2gFgChZwcmVmaXhfYmVmb3JlX2Jsb2NrX3YxEiByuG/a9x+QeTzXTjzga6+KFIGRoQYicqTphtfqRKND5CIAMiDCwBPIapyS4i+Wbz2GF3aPi3pEyQKtzAl7+h4JivtOH0gDEgzR8qcO/wzHH99ilVYaDMo7edEn68Q/I0lw7CIwxKWtUMYG+mzApWDsNah7EW7wi+CkJnubi2/HCyMD/GzlUQKayLF+MZZrkujuZYjdKntfH/3EMtbCq0e6OLRKaqoDfb0m862T5/1CryN2gfrmrOxZuVI6TdQQzppDuRasveEs0RQNtGePxPCVflP9Yx2ktJLio2asyO6nDnnfMe33c7TsDhy6wDq2plkjvn2aSaDBDQfbf3YFYfkXuscjD8Mh7KSql3CqJI9qOggYAQ=="},{"type":"tool_use","id":"toolu_013EgXcQDNpKPQoYB3SsWFxY","name":"read","input":{"objects":[{"name":"orders","schema":"public"}]},"caller":{"type":"direct"}},{"type":"tool_use","id":"toolu_01Jkq5nRsJX4dh5kHHcBhYXg","name":"search","input":{"text":"legacy_status"},"caller":{"type":"direct"}}],"container":null,"stop_reason":"tool_use","stop_sequence":null,"stop_details":null,"usage":{"input_tokens":2100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":0},"output_tokens":143,"output_tokens_details":{"thinking_tokens":32},"service_tier":"standard","inference_geo":"global"},"diagnostics":null}`

// claudeRequest is the part of the Claude wire format the tests assert on.
type claudeRequest struct {
	MaxTokens int `json:"max_tokens"`
	Messages  []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"messages"`
}

func newClaudeServer(t *testing.T, reply string, requests *[]claudeRequest) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request claudeRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		*requests = append(*requests, request)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(reply))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestChatClaudeOutputLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options []ChatOption
		want    int
	}{
		{name: "the SQL editor asks for no limit", want: 4096},
		{name: "a caller asks for a larger limit", options: []ChatOption{WithMaxOutputTokens(16000)}, want: 16000},
		{name: "a zero limit keeps the default", options: []ChatOption{WithMaxOutputTokens(0)}, want: 4096},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests []claudeRequest
			server := newClaudeServer(t, `{"content": [{"type": "text", "text": "ok"}]}`, &requests)
			setting := &storepb.AISetting{Provider: storepb.AISetting_CLAUDE, Endpoint: server.URL, Model: "claude-sonnet-5", ApiKey: "test-key"}
			userContent := "Explain the query."
			_, err := Chat(context.Background(), setting, &v1pb.AIChatRequest{
				Messages: []*v1pb.AIChatMessage{{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &userContent}},
			}, tc.options...)
			require.NoError(t, err)
			require.Len(t, requests, 1)
			require.Equal(t, tc.want, requests[0].MaxTokens)
		})
	}
}

func TestChatClaudeSendsParallelToolResultsInOneUserTurn(t *testing.T) {
	t.Parallel()

	var requests []claudeRequest
	server := newClaudeServer(t, claudeParallelToolUseReply, &requests)
	setting := &storepb.AISetting{Provider: storepb.AISetting_CLAUDE, Endpoint: server.URL, Model: "claude-sonnet-5", ApiKey: "test-key"}
	userContent := "Review the change."
	messages := []*v1pb.AIChatMessage{{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &userContent}}

	first, err := chatClaude(context.Background(), setting, &v1pb.AIChatRequest{Messages: messages})
	require.NoError(t, err)
	require.Len(t, first.ToolCalls, 2)

	messages = append(messages, &v1pb.AIChatMessage{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT, ToolCalls: first.ToolCalls})
	for _, call := range first.ToolCalls {
		output := "result of " + call.Name
		messages = append(messages, &v1pb.AIChatMessage{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL, Content: &output, ToolCallId: &call.Id})
	}
	_, err = chatClaude(context.Background(), setting, &v1pb.AIChatRequest{Messages: messages})
	require.NoError(t, err)

	require.Len(t, requests, 2)
	second := requests[1]
	require.Len(t, second.Messages, 3, "both results go in the one user turn after the tool calls")
	require.Equal(t, "assistant", second.Messages[1].Role)
	require.Equal(t, "user", second.Messages[2].Role)
	require.JSONEq(t, `[
		{"type": "tool_result", "tool_use_id": "toolu_013EgXcQDNpKPQoYB3SsWFxY", "content": "result of read"},
		{"type": "tool_result", "tool_use_id": "toolu_01Jkq5nRsJX4dh5kHHcBhYXg", "content": "result of search"}
	]`, string(second.Messages[2].Content))
}
