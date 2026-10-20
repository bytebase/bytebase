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

func TestSettingModelKeepsGeminiRolesAlternatingAfterAnUnparsedToolCall(t *testing.T) {
	t.Parallel()

	replies := []string{
		`{"candidates": [{"content": {"parts": [{"functionCall": {"name": "search", "args": {"text": "orders"}}}]}, "finishReason": "STOP"}]}`,
		`{"candidates": [{"content": {}, "finishReason": "MALFORMED_FUNCTION_CALL", "finishMessage": "Malformed function call: read(objects="}]}`,
		`{"candidates": [{"content": {"parts": [{"text": "{\"findings\": []}"}]}, "finishReason": "STOP"}]}`,
	}
	var requests []geminiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request geminiRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		requests = append(requests, request)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(replies[min(len(requests), len(replies))-1]))
		require.NoError(t, err)
	}))
	defer server.Close()

	model := NewModel(&storepb.AISetting{Provider: storepb.AISetting_GEMINI, Endpoint: server.URL, Model: "gemini-3.5-flash", ApiKey: "test-key"})
	result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: "DROP TABLE orders;"}, ordersCatalog())
	require.NoError(t, err)
	require.Empty(t, result.Findings)
	require.Len(t, requests, 3)

	contents := requests[2].Contents
	roles := make([]string, 0, len(contents))
	for _, content := range contents {
		roles = append(roles, content.Role)
	}
	require.Equal(t, []string{"user", "model", "user"}, roles)
	last := contents[2].Parts
	require.Len(t, last, 2)
	require.Equal(t, "search", last[0].FunctionResponse.Name)
	require.Equal(t, malformedToolCallMessage, last[1].Text)
}
