package slack

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

func TestBuildMessage_IssueApproved(t *testing.T) {
	a := require.New(t)

	ctx := webhook.Context{
		Level:       webhook.WebhookSuccess,
		Title:       "Issue approved",
		Description: "Bob approved the issue",
		Link:        "https://bb.example.com/projects/proj-1/issues/42",
		ActorName:   "Bob",
		ActorEmail:  "bob@example.com",
		Project:     &webhook.Project{Name: "projects/proj-1", Title: "My Project"},
		Issue: &webhook.Issue{
			ID:      42,
			Name:    "Grant read access to prod",
			Creator: webhook.Creator{Name: "Alice", Email: "alice@example.com"},
		},
	}

	msg := BuildMessage(ctx)

	a.Equal("Issue approved", msg.Text)
	a.Len(msg.Attachments, 1)
	a.Equal("#2EB67D", msg.Attachments[0].Color)

	blocks := msg.Attachments[0].BlockList
	// Expected: title, description, issue tile, context, actions = 5 blocks
	a.Len(blocks, 5)

	// Title block — linked
	a.Equal("section", blocks[0].Type)
	a.Contains(blocks[0].Text.Text, "✅")
	a.Contains(blocks[0].Text.Text, "<https://bb.example.com/projects/proj-1/issues/42|Issue approved>")

	// Description block
	a.Equal("section", blocks[1].Type)
	a.Equal("Bob approved the issue", blocks[1].Text.Text)

	// Issue tile — prominent, bold
	a.Equal("section", blocks[2].Type)
	a.Equal("*Grant read access to prod*", blocks[2].Text.Text)

	// Context block — project, creator, actor (no issue name)
	a.Equal("context", blocks[3].Type)
	a.Len(blocks[3].ElementList, 1)
	contextJSON, _ := json.Marshal(blocks[3].ElementList[0])
	a.Contains(string(contextJSON), "My Project")
	a.Contains(string(contextJSON), "Alice")
	a.Contains(string(contextJSON), "Bob")
	a.NotContains(string(contextJSON), "Grant read access") // issue name is in tile, not context

	// Actions block
	a.Equal("actions", blocks[4].Type)
}

func TestBuildMessage_RolloutFailed(t *testing.T) {
	a := require.New(t)

	ctx := webhook.Context{
		Level:       webhook.WebhookError,
		Title:       "Rollout failed",
		Description: "Rollout failed",
		Link:        "https://bb.example.com/projects/proj-1/plans/10/rollout",
		Project:     &webhook.Project{Name: "projects/proj-1", Title: "My Project"},
		Rollout:     &webhook.Rollout{UID: 10, Title: "Deploy v2"},
		Environment: "environments/prod",
	}

	msg := BuildMessage(ctx)

	a.Equal("#E01E5A", msg.Attachments[0].Color)
	blocks := msg.Attachments[0].BlockList

	// Title should have error emoji
	a.Contains(blocks[0].Text.Text, "❗")

	// Rollout tile — prominent
	a.Equal("section", blocks[2].Type)
	a.Equal("*Deploy v2*", blocks[2].Text.Text)

	// Context should have project and env (rollout name is in tile)
	a.Equal("context", blocks[3].Type)
	contextJSON, _ := json.Marshal(blocks[3].ElementList[0])
	a.Contains(string(contextJSON), "My Project")
	a.Contains(string(contextJSON), "environments/prod")
	a.NotContains(string(contextJSON), "Deploy v2")
}

func TestBuildMessage_NoLink(t *testing.T) {
	a := require.New(t)

	ctx := webhook.Context{
		Level: webhook.WebhookInfo,
		Title: "Issue created",
	}

	msg := BuildMessage(ctx)

	a.Equal("#1264A3", msg.Attachments[0].Color)
	blocks := msg.Attachments[0].BlockList

	// Title should NOT be linked
	a.NotContains(blocks[0].Text.Text, "<")

	// No actions block when no link
	for _, b := range blocks {
		a.NotEqual("actions", b.Type)
	}
}

func TestBuildMessage_Warn(t *testing.T) {
	a := require.New(t)

	ctx := webhook.Context{
		Level: webhook.WebhookWarn,
		Title: "Issue sent back",
		Link:  "https://bb.example.com/projects/proj-1/issues/1",
	}

	msg := BuildMessage(ctx)

	a.Equal("#ECB22E", msg.Attachments[0].Color)
	a.Contains(msg.Attachments[0].BlockList[0].Text.Text, "⚠️")
}

func TestBuildMessage_JSONStructure(t *testing.T) {
	a := require.New(t)

	ctx := webhook.Context{
		Level:       webhook.WebhookSuccess,
		Title:       "Issue approved",
		Description: "Alice approved the issue",
		Link:        "https://bb.example.com/projects/p/issues/1",
		Project:     &webhook.Project{Name: "projects/p", Title: "P"},
		Issue: &webhook.Issue{
			Name:    "My Issue",
			Creator: webhook.Creator{Name: "Bob"},
		},
		ActorName: "Alice",
	}

	msg := BuildMessage(ctx)
	b, err := json.Marshal(msg)
	a.NoError(err)

	// Verify the JSON is valid and has expected top-level structure
	var parsed map[string]any
	a.NoError(json.Unmarshal(b, &parsed))
	a.Equal("Issue approved", parsed["text"])
	a.NotNil(parsed["attachments"])

	attachments, ok := parsed["attachments"].([]any)
	a.True(ok)
	a.Len(attachments, 1)

	att, ok := attachments[0].(map[string]any)
	a.True(ok)
	a.Equal("#2EB67D", att["color"])
	a.NotNil(att["blocks"])
}

func TestEscapeMrkdwn(t *testing.T) {
	a := require.New(t)
	a.Equal("hello &amp; world", escapeMrkdwn("hello & world"))
	a.Equal("a &lt;b&gt; c", escapeMrkdwn("a <b> c"))
	a.Equal("no special chars", escapeMrkdwn("no special chars"))
	a.Equal("&amp;&lt;&gt;", escapeMrkdwn("&<>"))
}

func TestBuildMessage_EscapesUserContent(t *testing.T) {
	a := require.New(t)

	ctx := webhook.Context{
		Level:       webhook.WebhookWarn,
		Title:       "Issue <sent> back",
		Description: "Bob & Alice sent back: <script>alert('xss')</script>",
		Link:        "https://bb.example.com/issues/1",
		Project:     &webhook.Project{Name: "p", Title: "Project <A>"},
		Issue: &webhook.Issue{
			Name:    "Issue with <angle> brackets",
			Creator: webhook.Creator{Name: "O'Brien & Sons"},
		},
		ActorName: "Admin <root>",
	}

	msg := BuildMessage(ctx)
	blocks := msg.Attachments[0].BlockList

	// Title should have escaped angle brackets
	a.Contains(blocks[0].Text.Text, "Issue &lt;sent&gt; back")

	// Description should escape HTML
	a.Contains(blocks[1].Text.Text, "&amp;")
	a.Contains(blocks[1].Text.Text, "&lt;script&gt;")

	// Issue tile should escape
	a.Contains(blocks[2].Text.Text, "&lt;angle&gt;")

	// Context should escape — check via the BlockMarkdown Text field directly.
	contextElem, ok := blocks[3].ElementList[0].(BlockMarkdown)
	a.True(ok)
	a.Contains(contextElem.Text, "Project &lt;A&gt;")
	a.Contains(contextElem.Text, "O'Brien &amp; Sons")
	a.Contains(contextElem.Text, "Admin &lt;root&gt;")
}

func TestLevelColor(t *testing.T) {
	a := require.New(t)
	a.Equal("#1264A3", levelColor(webhook.WebhookInfo))
	a.Equal("#2EB67D", levelColor(webhook.WebhookSuccess))
	a.Equal("#ECB22E", levelColor(webhook.WebhookWarn))
	a.Equal("#E01E5A", levelColor(webhook.WebhookError))
}

func TestLevelEmoji(t *testing.T) {
	a := require.New(t)
	a.Equal("", levelEmoji(webhook.WebhookInfo))
	a.Equal("✅ ", levelEmoji(webhook.WebhookSuccess))
	a.Equal("⚠️ ", levelEmoji(webhook.WebhookWarn))
	a.Equal("❗ ", levelEmoji(webhook.WebhookError))
}
