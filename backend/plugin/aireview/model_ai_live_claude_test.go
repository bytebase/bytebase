package aireview

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestReviewLiveClaude reviews a risky change with a real Claude model. It
// needs a key and network access that CI does not have:
//
//	AIREVIEW_LIVE_CLAUDE_API_KEY=... go test ./backend/plugin/aireview/ -run '^TestReviewLiveClaude$' -v -count=1
func TestReviewLiveClaude(t *testing.T) {
	apiKey := os.Getenv("AIREVIEW_LIVE_CLAUDE_API_KEY")
	if apiKey == "" {
		t.Skip("AIREVIEW_LIVE_CLAUDE_API_KEY is not set")
	}
	modelName := os.Getenv("AIREVIEW_LIVE_CLAUDE_MODEL")
	if modelName == "" {
		modelName = "claude-sonnet-5"
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
	model := NewModel(&storepb.AISetting{
		Provider: storepb.AISetting_CLAUDE,
		Endpoint: "https://api.anthropic.com/v1/messages",
		Model:    modelName,
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
}
