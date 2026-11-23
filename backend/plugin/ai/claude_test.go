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
	System    json.RawMessage `json:"system"`
	MaxTokens int             `json:"max_tokens"`
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
		model   string
		options []ChatOption
		want    int
	}{
		{name: "the SQL editor asks for no limit", model: "claude-sonnet-5", want: 4096},
		{name: "a caller asks for a larger limit", model: "claude-sonnet-5", options: []ChatOption{WithMaxOutputTokens(16000)}, want: 16000},
		{name: "an older model caps the larger limit", model: "claude-3-5-haiku-20241022", options: []ChatOption{WithMaxOutputTokens(16000)}, want: 8192},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests []claudeRequest
			server := newClaudeServer(t, `{"content": [{"type": "text", "text": "ok"}]}`, &requests)
			setting := &storepb.AISetting{Provider: storepb.AISetting_CLAUDE, Endpoint: server.URL, Model: tc.model, ApiKey: "test-key"}
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

func TestClaudeMaxTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		model     string
		requested int
		want      int
	}{
		// Callers that ask for no limit get the default whatever the model.
		{model: "claude-sonnet-5", requested: 0, want: 4096},
		{model: "claude-3-haiku-20240307", requested: 0, want: 4096},

		// Models at or above the request keep it.
		{model: "claude-sonnet-5", requested: 16000, want: 16000},
		{model: "claude-haiku-4-5", requested: 16000, want: 16000},
		{model: "claude-3-7-sonnet-20250219", requested: 16000, want: 16000},
		{model: "claude-3-7-sonnet-latest", requested: 16000, want: 16000},
		{model: "anthropic.claude-sonnet-4-20250514-v1:0", requested: 16000, want: 16000},
		{model: "a-custom-model", requested: 16000, want: 16000},
		{model: "claude-20-sonnet", requested: 16000, want: 16000},
		{model: "anthropic.claude-v20-test", requested: 16000, want: 16000},

		{model: "claude-3-5-sonnet-20241022", requested: 16000, want: 8192},
		{model: "claude-3-5-sonnet-20240620", requested: 16000, want: 8192},
		{model: "claude-3-5-sonnet-latest", requested: 16000, want: 8192},
		{model: "anthropic.claude-3-5-sonnet-20241022-v2:0", requested: 16000, want: 8192},
		{model: "claude-3-5-sonnet-v2@20241022", requested: 16000, want: 8192},
		{model: "claude-3-5-haiku-20241022", requested: 16000, want: 8192},
		{model: "anthropic.claude-3-5-haiku-20241022-v1:0", requested: 16000, want: 8192},
		{model: "us.anthropic.claude-3-5-haiku-20241022-v1:0", requested: 16000, want: 8192},
		{model: "myclaude-3-5-sonnet-20240620", requested: 16000, want: 8192},
		{model: "proxy/claude-3-5-sonnet20240620", requested: 16000, want: 8192},
		{model: "claude-3-opus-20240229", requested: 16000, want: 4096},
		{model: "CLAUDE-3-OPUS-LATEST", requested: 16000, want: 4096},
		{model: "claude-3-sonnet-20240229", requested: 16000, want: 4096},
		{model: "claude-3-haiku-20240307", requested: 16000, want: 4096},
		{model: "claude-3-haiku@20240307", requested: 16000, want: 4096},
		{model: "claude-2", requested: 16000, want: 4096},
		{model: "claude-2.1", requested: 16000, want: 4096},
		{model: "claude-2.0", requested: 16000, want: 4096},
		{model: "anthropic.claude-v2:1", requested: 16000, want: 4096},
		{model: "claude-instant-1.2", requested: 16000, want: 4096},
		{model: "anthropic.claude-instant-v1", requested: 16000, want: 4096},

		// A request below the ceiling is sent as asked.
		{model: "claude-3-5-haiku-20241022", requested: 2000, want: 2000},
		{model: "claude-3-opus-20240229", requested: 1024, want: 1024},
	}
	for _, tc := range tests {
		require.Equal(t, tc.want, claudeMaxTokens(tc.model, tc.requested), "%s asking for %d", tc.model, tc.requested)
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
		{"type": "tool_result", "tool_use_id": "toolu_01Jkq5nRsJX4dh5kHHcBhYXg", "content": "result of search", "cache_control": {"type": "ephemeral"}}
	]`, string(second.Messages[2].Content))
}

func TestChatClaudeCacheBreakpoints(t *testing.T) {
	t.Parallel()

	const (
		system = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM
		user   = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER
		model  = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT
		tool   = v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL
	)
	message := func(role v1pb.AIChatMessageRole, content string) *v1pb.AIChatMessage {
		return &v1pb.AIChatMessage{Role: role, Content: &content}
	}
	toolID := "toolu_1"
	toolCall := &v1pb.AIChatMessage{Role: model, ToolCalls: []*v1pb.AIChatToolCall{{Id: toolID, Name: "read", Arguments: `{"name": "orders"}`}}}
	toolResult := func(content string) *v1pb.AIChatMessage {
		return &v1pb.AIChatMessage{Role: tool, Content: &content, ToolCallId: &toolID}
	}
	tools := []*v1pb.AIChatToolDefinition{{Name: "read", Description: "Read a table.", ParametersSchema: `{"type": "object"}`}}
	const systemBreakpoint = `[{"type": "text", "text": "Review the change.", "cache_control": {"type": "ephemeral"}}]`

	tests := []struct {
		name     string
		messages []*v1pb.AIChatMessage
		tools    []*v1pb.AIChatToolDefinition
		// wantSystem is empty when the request must carry no system prompt.
		wantSystem string
		// wantMessages holds the content of every message, in order.
		wantMessages []string
	}{
		{
			name:         "a lone message is sent once, so nothing is stored",
			messages:     []*v1pb.AIChatMessage{message(user, "Lint the files.")},
			wantMessages: []string{`"Lint the files."`},
		},
		{
			name:         "a system prompt is stored for the requests that share it",
			messages:     []*v1pb.AIChatMessage{message(system, "Review the change."), message(user, "DROP TABLE orders;")},
			wantSystem:   systemBreakpoint,
			wantMessages: []string{`"DROP TABLE orders;"`},
		},
		{
			name:         "tools mean a next call, which reads this one",
			messages:     []*v1pb.AIChatMessage{message(system, "Review the change."), message(user, "DROP TABLE orders;")},
			tools:        tools,
			wantSystem:   systemBreakpoint,
			wantMessages: []string{`[{"type": "text", "text": "DROP TABLE orders;", "cache_control": {"type": "ephemeral"}}]`},
		},
		{
			name:       "the breakpoint moves to the end of a running conversation",
			messages:   []*v1pb.AIChatMessage{message(system, "Review the change."), message(user, "DROP TABLE orders;"), toolCall, toolResult("12 rows")},
			tools:      tools,
			wantSystem: systemBreakpoint,
			wantMessages: []string{
				`"DROP TABLE orders;"`,
				`[{"type": "tool_use", "id": "toolu_1", "name": "read", "input": {"name": "orders"}}]`,
				`[{"type": "tool_result", "tool_use_id": "toolu_1", "content": "12 rows", "cache_control": {"type": "ephemeral"}}]`,
			},
		},
		{
			name:     "earlier turns mean a running conversation without tools too",
			messages: []*v1pb.AIChatMessage{message(user, "Summarize."), message(model, "Not JSON."), message(user, "Reply with JSON.")},
			wantMessages: []string{
				`"Summarize."`,
				`[{"type": "text", "text": "Not JSON."}]`,
				`[{"type": "text", "text": "Reply with JSON.", "cache_control": {"type": "ephemeral"}}]`,
			},
		},
		{
			name:         "blank text takes no breakpoint, which Claude would reject",
			messages:     []*v1pb.AIChatMessage{message(system, " "), message(user, "")},
			tools:        tools,
			wantMessages: []string{`""`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests []claudeRequest
			server := newClaudeServer(t, `{"content": [{"type": "text", "text": "ok"}]}`, &requests)
			setting := &storepb.AISetting{Provider: storepb.AISetting_CLAUDE, Endpoint: server.URL, Model: "claude-sonnet-5-5", ApiKey: "test-key"}
			_, err := Chat(context.Background(), setting, &v1pb.AIChatRequest{Messages: tc.messages, ToolDefinitions: tc.tools})
			require.NoError(t, err)
			require.Len(t, requests, 1)
			if tc.wantSystem == "" {
				require.Empty(t, requests[0].System)
			} else {
				require.JSONEq(t, tc.wantSystem, string(requests[0].System))
			}
			require.Len(t, requests[0].Messages, len(tc.wantMessages))
			for i, want := range tc.wantMessages {
				require.JSONEq(t, want, string(requests[0].Messages[i].Content), "message %d", i)
			}
		})
	}
}
