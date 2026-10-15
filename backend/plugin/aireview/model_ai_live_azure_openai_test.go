package aireview

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestReviewLiveAzureOpenAI reviews a risky change with a real Azure OpenAI
// deployment, once through each endpoint the settings page offers. It needs a
// resource that CI does not have. The endpoint is the resource's, without a path:
//
//	AIREVIEW_LIVE_AZURE_OPENAI_ENDPOINT=https://YOUR-RESOURCE-NAME.openai.azure.com \
//	AIREVIEW_LIVE_AZURE_OPENAI_API_KEY=... \
//	AIREVIEW_LIVE_AZURE_OPENAI_DEPLOYMENT=... \
//	go test ./backend/plugin/aireview/ -run '^TestReviewLiveAzureOpenAI$' -v -count=1
//
// Azure rejects tools on chat completions for a gpt-5.6 deployment, so run that
// case with a deployment such as gpt-4o and -run '^TestReviewLiveAzureOpenAI$/^chat_completions$'.
func TestReviewLiveAzureOpenAI(t *testing.T) {
	resource := os.Getenv("AIREVIEW_LIVE_AZURE_OPENAI_ENDPOINT")
	apiKey := os.Getenv("AIREVIEW_LIVE_AZURE_OPENAI_API_KEY")
	deployment := os.Getenv("AIREVIEW_LIVE_AZURE_OPENAI_DEPLOYMENT")
	if resource == "" || apiKey == "" || deployment == "" {
		t.Skip("AIREVIEW_LIVE_AZURE_OPENAI_ENDPOINT, AIREVIEW_LIVE_AZURE_OPENAI_API_KEY, and AIREVIEW_LIVE_AZURE_OPENAI_DEPLOYMENT are not all set")
	}

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
	for _, endpoint := range azureOpenAIEndpoints {
		t.Run(endpoint.kind, func(t *testing.T) {
			model := NewModel(&storepb.AISetting{
				Provider: storepb.AISetting_AZURE_OPENAI,
				Endpoint: strings.TrimRight(resource, "/") + endpoint.path,
				Model:    deployment,
				ApiKey:   apiKey,
			})
			tools := &catalogTools{}

			result, err := NewReviewer(model).Review(context.Background(), request, tools)
			require.NoError(t, err)
			t.Logf("model calls: %d, tokens: %d, tool calls: %v", result.Calls, result.TotalTokens, tools.calls)
			for _, finding := range result.Findings {
				t.Logf("%s line %d: %s\n  rule: %s\n  evidence: %s\n  fix: %s", finding.Severity, finding.Line, finding.Title, finding.Rule, finding.Evidence, finding.Fix)
			}
			for _, note := range result.Notes {
				t.Logf("note: %s", note)
			}
			require.NotEmpty(t, tools.calls, "the model should look up the objects before it judges")
			require.NotEmpty(t, result.Findings)
		})
	}
}
