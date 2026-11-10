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

func sentGeminiRequest(t *testing.T, messages []*v1pb.AIChatMessage) chatGeminiRequest {
	t.Helper()

	var payload chatGeminiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"candidates": [{"content": {"parts": [{"text": "done"}]}, "finishReason": "STOP"}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	_, err := Chat(
		context.Background(),
		&storepb.AISetting{Provider: storepb.AISetting_GEMINI, Endpoint: server.URL, Model: "gemini-3.5-flash", ApiKey: "test-key"},
		&v1pb.AIChatRequest{Messages: messages},
	)
	require.NoError(t, err)
	return payload
}

func geminiRoles(contents []chatGeminiContent) []string {
	roles := make([]string, 0, len(contents))
	for _, content := range contents {
		roles = append(roles, content.Role)
	}
	return roles
}

func TestChatGeminiJoinsToolResultsAndTheTextAfterThemInOneUserTurn(t *testing.T) {
	t.Parallel()

	system, prompt, correction := "You review SQL.", "Review this change.", "Call the tool again."
	readID, searchID := "call-read", "call-search"
	readResult, searchResult := `{"rows": 7}`, `{"total": 2}`
	payload := sentGeminiRequest(t, []*v1pb.AIChatMessage{
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_SYSTEM, Content: &system},
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &prompt},
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT, ToolCalls: []*v1pb.AIChatToolCall{
			{Id: readID, Name: "read", Arguments: `{"objects": [{"name": "orders"}]}`},
			{Id: searchID, Name: "search", Arguments: `{"text": "orders"}`},
		}},
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL, Content: &readResult, ToolCallId: &readID},
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL, Content: &searchResult, ToolCallId: &searchID},
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &correction},
	})

	require.NotNil(t, payload.SystemInstruction)
	require.Len(t, payload.SystemInstruction.Parts, 1)
	require.Equal(t, system, payload.SystemInstruction.Parts[0].Text)

	require.Equal(t, []string{"user", "model", "user"}, geminiRoles(payload.Contents))
	require.Len(t, payload.Contents[0].Parts, 1)
	require.Equal(t, prompt, payload.Contents[0].Parts[0].Text)
	calls := payload.Contents[1].Parts
	require.Len(t, calls, 2)
	require.Equal(t, "read", calls[0].FunctionCall.Name)
	require.Equal(t, "search", calls[1].FunctionCall.Name)

	results := payload.Contents[2].Parts
	require.Len(t, results, 3)
	require.Equal(t, "read", results[0].FunctionResponse.Name)
	require.Equal(t, map[string]any{"rows": float64(7)}, results[0].FunctionResponse.Response)
	require.Equal(t, "search", results[1].FunctionResponse.Name)
	require.Equal(t, map[string]any{"total": float64(2)}, results[1].FunctionResponse.Response)
	require.Nil(t, results[2].FunctionResponse)
	require.Equal(t, correction, results[2].Text)
}

func TestChatGeminiJoinsATextAfterThePromptInOneUserTurn(t *testing.T) {
	t.Parallel()

	prompt, correction := "Review this change.", "Call the tool again."
	payload := sentGeminiRequest(t, []*v1pb.AIChatMessage{
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &prompt},
		{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &correction},
	})

	require.Nil(t, payload.SystemInstruction)
	require.Equal(t, []string{"user"}, geminiRoles(payload.Contents))
	require.Len(t, payload.Contents[0].Parts, 2)
	require.Equal(t, prompt, payload.Contents[0].Parts[0].Text)
	require.Equal(t, correction, payload.Contents[0].Parts[1].Text)
}
