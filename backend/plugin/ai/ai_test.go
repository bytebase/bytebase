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

func TestChatGeminiPartMarshalsThoughtSignatureOnPart(t *testing.T) {
	t.Parallel()

	payload := chatGeminiRequest{
		Contents: []chatGeminiContent{{
			Role: "model",
			Parts: []chatGeminiPart{{
				FunctionCall: &chatGeminiFunctionCall{
					Name: "list_tables",
					Args: map[string]any{"database": "db"},
				},
				ThoughtSignature: "sig-123",
			}},
		}},
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(payloadBytes, &raw))

	contents, ok := raw["contents"].([]any)
	require.True(t, ok)
	require.Len(t, contents, 1)

	content, ok := contents[0].(map[string]any)
	require.True(t, ok)

	parts, ok := content["parts"].([]any)
	require.True(t, ok)
	require.Len(t, parts, 1)

	part, ok := parts[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "sig-123", part["thoughtSignature"])

	functionCall, ok := part["functionCall"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "list_tables", functionCall["name"])
	require.NotContains(t, functionCall, "thoughtSignature")
}

func TestChatGeminiResponsePartUnmarshalsThoughtSignatureFromPart(t *testing.T) {
	t.Parallel()

	var part chatGeminiResponsePart
	err := json.Unmarshal([]byte(`{
		"functionCall": {
			"name": "list_tables",
			"args": {"database": "db"}
		},
		"thoughtSignature": "sig-123"
	}`), &part)
	require.NoError(t, err)
	require.NotNil(t, part.FunctionCall)
	require.Equal(t, "list_tables", part.FunctionCall.Name)
	require.Equal(t, "sig-123", part.ThoughtSignature)
}

func TestBuildChatToolCallMetadataExtractsGeminiThoughtSignature(t *testing.T) {
	t.Parallel()

	rawToolCall := json.RawMessage(`{
		"id": "call-1",
		"type": "function",
		"function": {
			"name": "get_page_state",
			"arguments": "{}"
		},
		"extra_content": {
			"google": {
				"thought_signature": "sig-123"
			}
		}
	}`)

	metadata, err := buildChatToolCallMetadata(rawToolCall, "")
	require.NoError(t, err)
	require.NotNil(t, metadata)
	require.Equal(t, "sig-123", geminiThoughtSignatureFromMetadata(*metadata))
	require.JSONEq(t, string(rawToolCall), string(openAIToolCallFromMetadata(*metadata)))
}

func TestGeminiThoughtSignatureFromMetadataSupportsLegacyFormats(t *testing.T) {
	t.Parallel()

	t.Run("legacy raw tool call with thought signature", func(t *testing.T) {
		t.Parallel()

		metadata := `{
			"id": "call-1",
			"type": "function",
			"function": {
				"name": "get_page_state",
				"arguments": "{}"
			},
			"extra_content": {
				"google": {
					"thought_signature": "sig-legacy"
				}
			}
		}`
		require.Equal(t, "sig-legacy", geminiThoughtSignatureFromMetadata(metadata))
	})

	t.Run("legacy raw tool call without thought signature", func(t *testing.T) {
		t.Parallel()

		metadata := `{
			"id": "call-2",
			"type": "function",
			"function": {
				"name": "search_api",
				"arguments": "{}"
			}
		}`
		require.Empty(t, geminiThoughtSignatureFromMetadata(metadata))
	})

	t.Run("native Gemini thought signature string", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "sig-native", geminiThoughtSignatureFromMetadata("sig-native"))
	})
}

func TestChatGeminiGeneratesUniqueToolCallIDs(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [
						{"functionCall": {"name": "search_api", "args": {"service": "SQLService"}}},
						{"functionCall": {"name": "search_api", "args": {"operationId": "SQLService/Query"}}}
					]
				}
			}]
		}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	resp, err := chatGemini(
		context.Background(),
		&storepb.AISetting{
			Endpoint: server.URL,
			Model:    "gemini-2.5-pro",
			ApiKey:   "test-key",
		},
		&v1pb.AIChatRequest{},
	)
	require.NoError(t, err)
	require.Len(t, resp.ToolCalls, 2)
	require.NotEqual(t, resp.ToolCalls[0].Id, resp.ToolCalls[1].Id)
}

func TestChatOpenAIResponsesEndpointUsesResponsesWireFormat(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "test-key", r.Header.Get("api-key"))
		require.Empty(t, r.Header.Get("Authorization"))

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "deployment", payload["model"])
		require.Equal(t, false, payload["store"])
		require.Equal(t, []any{"reasoning.encrypted_content"}, payload["include"])
		require.NotContains(t, payload, "messages")
		require.Contains(t, payload, "input")
		tools, ok := payload["tools"].([]any)
		require.True(t, ok)
		require.Equal(t, map[string]any{
			"type":        "function",
			"name":        "list_tables",
			"description": "Lists tables",
			"parameters":  map[string]any{"type": "object"},
		}, tools[0])

		_, err := w.Write([]byte(`{
			"output": [{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "hello"}]}],
			"usage": {"total_tokens": 3}
		}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	content := "Find tables"
	resp, err := chatOpenAI(
		context.Background(),
		&storepb.AISetting{
			Provider: storepb.AISetting_AZURE_OPENAI,
			Endpoint: server.URL + "/openai/v1/responses",
			Model:    "deployment",
			ApiKey:   "test-key",
		},
		&v1pb.AIChatRequest{
			Messages: []*v1pb.AIChatMessage{{
				Role:    v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER,
				Content: &content,
			}},
			ToolDefinitions: []*v1pb.AIChatToolDefinition{{
				Name:             "list_tables",
				Description:      "Lists tables",
				ParametersSchema: `{"type":"object"}`,
			}},
		},
	)
	require.NoError(t, err)
	require.Equal(t, "hello", resp.GetContent())
	require.EqualValues(t, 3, resp.GetUsage().GetTotalTokens())
}

func TestChatOpenAIResponsesReplaysOutputForToolResult(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		input, ok := payload["input"].([]any)
		require.True(t, ok)

		switch requestCount {
		case 1:
			_, err := w.Write([]byte(`{
				"output": [
					{"type": "reasoning", "id": "rs_1", "encrypted_content": "ciphertext"},
					{"type": "function_call", "call_id": "call_1", "name": "list_tables", "arguments": "{}"}
				]
			}`))
			require.NoError(t, err)
		case 2:
			require.Contains(t, input, map[string]any{
				"type":              "reasoning",
				"id":                "rs_1",
				"encrypted_content": "ciphertext",
			})
			require.Contains(t, input, map[string]any{
				"type":      "function_call",
				"call_id":   "call_1",
				"name":      "list_tables",
				"arguments": "{}",
			})
			require.Contains(t, input, map[string]any{
				"type":    "function_call_output",
				"call_id": "call_1",
				"output":  "users, projects",
			})
			_, err := w.Write([]byte(`{"output": []}`))
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	}))
	defer server.Close()

	userContent := "Find tables"
	setting := &storepb.AISetting{
		Endpoint: server.URL + "/responses",
		Model:    "deployment",
		ApiKey:   "test-key",
	}
	first, err := chatOpenAI(context.Background(), setting, &v1pb.AIChatRequest{
		Messages: []*v1pb.AIChatMessage{{
			Role:    v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER,
			Content: &userContent,
		}},
	})
	require.NoError(t, err)
	require.Len(t, first.ToolCalls, 1)

	toolResult := "users, projects"
	_, err = chatOpenAI(context.Background(), setting, &v1pb.AIChatRequest{
		Messages: []*v1pb.AIChatMessage{
			{
				Role:    v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER,
				Content: &userContent,
			},
			{
				Role:      v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT,
				ToolCalls: first.ToolCalls,
			},
			{
				Role:       v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL,
				Content:    &toolResult,
				ToolCallId: &first.ToolCalls[0].Id,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, requestCount)
}

func TestChatClaudeReplaysThinkingBlocksForToolResult(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))

		switch requestCount {
		case 1:
			_, err := w.Write([]byte(`{
				"content": [
					{"type": "thinking", "thinking": "", "signature": "sig_1"},
					{"type": "text", "text": "I will check the tables."},
					{"type": "tool_use", "id": "toolu_1", "name": "list_tables", "input": {}}
				]
			}`))
			require.NoError(t, err)
		case 2:
			messages, ok := payload["messages"].([]any)
			require.True(t, ok)
			require.Contains(t, messages, map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "thinking", "thinking": "", "signature": "sig_1"},
					map[string]any{"type": "text", "text": "I will check the tables."},
					map[string]any{"type": "tool_use", "id": "toolu_1", "name": "list_tables", "input": map[string]any{}},
				},
			})
			_, err := w.Write([]byte(`{"content": []}`))
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	}))
	defer server.Close()

	userContent := "Find tables"
	setting := &storepb.AISetting{
		Endpoint: server.URL,
		Model:    "claude-sonnet-5",
		ApiKey:   "test-key",
	}
	first, err := chatClaude(context.Background(), setting, &v1pb.AIChatRequest{
		Messages: []*v1pb.AIChatMessage{{
			Role:    v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER,
			Content: &userContent,
		}},
	})
	require.NoError(t, err)
	require.Len(t, first.ToolCalls, 1)
	require.NotNil(t, first.ToolCalls[0].Metadata)
	require.Equal(t, "I will check the tables.", first.GetContent())

	toolResult := "users, projects"
	_, err = chatClaude(context.Background(), setting, &v1pb.AIChatRequest{
		Messages: []*v1pb.AIChatMessage{
			{
				Role:    v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER,
				Content: &userContent,
			},
			{
				Role:      v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT,
				ToolCalls: first.ToolCalls,
			},
			{
				Role:       v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_TOOL,
				Content:    &toolResult,
				ToolCallId: &first.ToolCalls[0].Id,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, requestCount)
}

func TestChatOpenAIChatCompletionsEndpointUsesChatCompletions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Contains(t, payload, "messages")
		require.NotContains(t, payload, "input")
		_, err := w.Write([]byte(`{
			"choices": [{"message": {"role": "assistant", "content": "hello"}}]
		}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	content := "Hello"
	resp, err := chatOpenAI(context.Background(), &storepb.AISetting{
		Endpoint: server.URL + "/chat/completions",
		Model:    "gpt-5.5",
		ApiKey:   "test-key",
	}, &v1pb.AIChatRequest{
		Messages: []*v1pb.AIChatMessage{{
			Role:    v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER,
			Content: &content,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, "hello", resp.GetContent())
}

func TestChatOpenAIResponsesFormatsAssistantHistoryAsOutputMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		input, ok := payload["input"].([]any)
		require.True(t, ok)
		require.Contains(t, input, map[string]any{
			"type":   "message",
			"role":   "assistant",
			"status": "completed",
			"content": []any{map[string]any{
				"type": "output_text",
				"text": "Previous answer",
			}},
		})
		_, err := w.Write([]byte(`{"output": []}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	userContent := "Follow up"
	assistantContent := "Previous answer"
	_, err := chatOpenAI(context.Background(), &storepb.AISetting{
		Endpoint: server.URL + "/responses",
		Model:    "deployment",
	}, &v1pb.AIChatRequest{
		Messages: []*v1pb.AIChatMessage{
			{
				Role:    v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER,
				Content: &userContent,
			},
			{
				Role:    v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_ASSISTANT,
				Content: &assistantContent,
			},
		},
	})
	require.NoError(t, err)
}
