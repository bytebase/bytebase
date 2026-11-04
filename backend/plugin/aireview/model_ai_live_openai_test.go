package aireview

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestReviewLiveOpenAI reviews a risky change with real OpenAI models, once on
// each endpoint. It needs a key and network access that CI does not have:
//
//	AIREVIEW_LIVE_OPENAI_API_KEY=... go test ./backend/plugin/aireview/ -run '^TestReviewLiveOpenAI$' -v -count=1
//
// AIREVIEW_LIVE_OPENAI_MODEL replaces the model of the Responses case.
func TestReviewLiveOpenAI(t *testing.T) {
	apiKey := os.Getenv("AIREVIEW_LIVE_OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("AIREVIEW_LIVE_OPENAI_API_KEY is not set")
	}
	responsesModel := os.Getenv("AIREVIEW_LIVE_OPENAI_MODEL")
	if responsesModel == "" {
		responsesModel = "gpt-5.6-terra"
	}

	tests := []struct {
		name     string
		endpoint string
		model    string
	}{
		{name: "responses", endpoint: "https://api.openai.com/v1/responses", model: responsesModel},
		{name: "chat completions", endpoint: "https://api.openai.com/v1/chat/completions", model: "gpt-4o"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statement := strings.Join([]string{
				"-- Speed up the order history page.",
				"CREATE INDEX idx_orders_created_at ON orders (created_at);",
				"",
				"ALTER TABLE orders DROP COLUMN legacy_status;",
			}, "\n")
			request := &Request{
				WorkspacePolicy: "An index on a table with more than 1 million rows must be created with CONCURRENTLY.",
				Target: Target{
					Engine:      "POSTGRES",
					Version:     "16.2",
					Environment: "prod",
					Schemas:     []SchemaSummary{{Name: "public", ObjectCount: 2}},
				},
				Statement: statement,
			}
			model := NewModel(&storepb.AISetting{
				Provider: storepb.AISetting_OPEN_AI,
				Endpoint: tc.endpoint,
				Model:    tc.model,
				ApiKey:   apiKey,
			})
			tools := &recordingTools{Tools: ordersCatalog()}

			result, err := NewReviewer(model).Review(context.Background(), request, tools)
			require.NoError(t, err)
			t.Logf("model: %s, model calls: %d, tokens: %d, tool calls: %v", tc.model, result.Calls, result.TotalTokens, tools.calls)
			for _, finding := range result.Findings {
				t.Logf("%s line %d: %s\n  rule: %s\n  reason: %s\n  evidence: %q\n  fix: %s", finding.Severity, finding.Line, finding.Title, finding.Rule, finding.Reason, finding.Evidence, finding.Fix)
			}
			for _, note := range result.Notes {
				t.Logf("note: %s", note)
			}
			require.NotEmpty(t, tools.calls, "the model should look up the objects before it judges")
			require.NotEmpty(t, result.Findings)
		})
	}
}
