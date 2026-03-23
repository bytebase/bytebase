package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
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
