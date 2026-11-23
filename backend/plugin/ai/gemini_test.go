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

func TestChatGeminiSendsToolParametersAsJSONSchema(t *testing.T) {
	t.Parallel()

	tools := []*v1pb.AIChatToolDefinition{
		{
			Name:        "get_overview",
			Description: "Read the current overview.",
			ParametersSchema: `{
				"type": "object",
				"properties": {"project": {"type": "string"}},
				"additionalProperties": false
			}`,
		},
		{
			Name:        "prepare_draft",
			Description: "Prepare an unsaved draft.",
			ParametersSchema: `{
				"type": "object",
				"properties": {
					"scope": {
						"type": "object",
						"properties": {"project": {"type": "string"}},
						"required": ["project"],
						"additionalProperties": false
					},
					"targets": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {"database": {"type": "string"}},
							"required": ["database"],
							"additionalProperties": false
						}
					}
				},
				"required": ["scope", "targets"],
				"additionalProperties": false
			}`,
		},
	}
	var payload struct {
		Tools []struct {
			FunctionDeclarations []map[string]json.RawMessage `json:"functionDeclarations"`
		} `json:"tools"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"Hi"}]},"finishReason":"STOP"}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	prompt := "say hi"
	response, err := Chat(context.Background(),
		&storepb.AISetting{Provider: storepb.AISetting_GEMINI, Endpoint: server.URL, Model: "gemini-3.5-flash", ApiKey: "test-key"},
		&v1pb.AIChatRequest{
			Messages:        []*v1pb.AIChatMessage{{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &prompt}},
			ToolDefinitions: tools,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "Hi", response.GetContent())
	require.Len(t, payload.Tools, 1)
	declarations := payload.Tools[0].FunctionDeclarations
	require.Len(t, declarations, len(tools))
	for i, tool := range tools {
		declaration := declarations[i]
		require.NotContains(t, declaration, "parameters")
		require.Contains(t, declaration, "parametersJsonSchema")
		require.JSONEq(t, tool.ParametersSchema, string(declaration["parametersJsonSchema"]))
		var name, description string
		require.NoError(t, json.Unmarshal(declaration["name"], &name))
		require.NoError(t, json.Unmarshal(declaration["description"], &description))
		require.Equal(t, tool.Name, name)
		require.Equal(t, tool.Description, description)
	}
}
