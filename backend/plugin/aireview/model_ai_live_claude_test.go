package aireview

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
	tools := &recordingTools{Tools: ordersCatalog()}

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

// TestReviewLiveClaudeReadsPromptCache sends the first request of a review
// twice and requires the second to read the first from Claude's prompt cache.
// A missed cache raises no error, only the cost, so nothing else would show
// that request-dependent text had entered the start of the prompt.
//
//	AIREVIEW_LIVE_CLAUDE_API_KEY=... go test ./backend/plugin/aireview/ -run '^TestReviewLiveClaudeReadsPromptCache$' -v -count=1
func TestReviewLiveClaudeReadsPromptCache(t *testing.T) {
	apiKey := os.Getenv("AIREVIEW_LIVE_CLAUDE_API_KEY")
	if apiKey == "" {
		t.Skip("AIREVIEW_LIVE_CLAUDE_API_KEY is not set")
	}
	modelName := os.Getenv("AIREVIEW_LIVE_CLAUDE_MODEL")
	if modelName == "" {
		modelName = "claude-sonnet-5-5"
	}

	request := &Request{
		WorkspacePolicy: "An index on a table with more than 1 million rows must be created with CONCURRENTLY.",
		Target:          Target{Engine: "POSTGRES", Version: "16.2", Environment: "prod"},
		Statement:       "CREATE INDEX idx_orders_created_at ON orders (created_at);",
	}
	numbered, _ := numberLines(request.Statement)
	chat := &v1pb.AIChatRequest{
		Messages:        buildMessages(request, numbered, "nonce", true),
		ToolDefinitions: ordersCatalog().Definitions(),
	}
	model := NewModel(&storepb.AISetting{
		Provider: storepb.AISetting_CLAUDE,
		Endpoint: "https://api.anthropic.com/v1/messages",
		Model:    modelName,
		ApiKey:   apiKey,
	})

	first, err := model.Chat(context.Background(), chat)
	require.NoError(t, err)
	second, err := model.Chat(context.Background(), chat)
	require.NoError(t, err)
	t.Logf("input tokens: %d, read from cache: first call %d, second call %d", second.GetUsage().GetInputTokens(), first.GetUsage().GetCachedInputTokens(), second.GetUsage().GetCachedInputTokens())
	// The whole request is stored, and Claude counts a few tokens of framing outside it.
	require.Greater(t, second.GetUsage().GetCachedInputTokens(), second.GetUsage().GetInputTokens()*9/10)
}
