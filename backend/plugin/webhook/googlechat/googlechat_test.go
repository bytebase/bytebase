package googlechat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

func TestBuildMessageIssueApproved(t *testing.T) {
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
			ID:          42,
			Name:        "Grant read access to prod",
			Description: "Need access for release verification",
			Creator:     webhook.Creator{Name: "Alice", Email: "alice@example.com"},
		},
	}

	msg := BuildMessage(ctx)

	a.Empty(msg.Text)
	a.Len(msg.CardsV2, 1)
	card := msg.CardsV2[0].Card
	a.NotNil(card.Header)
	a.Equal("✅ Issue approved", card.Header.Title)
	a.Contains(card.Header.Subtitle, "My Project")
	a.NotEmpty(card.Sections)

	body, err := json.Marshal(msg)
	a.NoError(err)
	bodyText := string(body)
	var payload map[string]any
	a.NoError(json.Unmarshal(body, &payload))
	a.NotContains(payload, "text")
	a.Contains(bodyText, "Grant read access to prod")
	a.Contains(bodyText, "Need access for release verification")
	a.Contains(bodyText, "Alice")
	a.Contains(bodyText, "Bob")
	a.Contains(bodyText, "View in Bytebase")
	a.Contains(bodyText, "https://bb.example.com/projects/proj-1/issues/42")

	body, err = marshal(msg)
	a.NoError(err)
	a.Contains(string(body), "<b>Need access for release verification</b>")
}

func TestBuildMessageRolloutFailed(t *testing.T) {
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

	card := msg.CardsV2[0].Card
	a.NotNil(card.Header)
	a.Equal("❗ Rollout failed", card.Header.Title)

	body, err := json.Marshal(msg)
	a.NoError(err)
	bodyText := string(body)
	a.Contains(bodyText, "Rollout failed")
	a.Contains(bodyText, "Deploy v2")
	a.Contains(bodyText, "environments/prod")
	a.Contains(bodyText, "My Project")
}

func TestBuildMessageTitleOnlyOmitsEmptySection(t *testing.T) {
	a := require.New(t)
	msg := BuildMessage(webhook.Context{
		Title: "Issue created",
	})

	card := msg.CardsV2[0].Card
	a.NotNil(card.Header)
	a.Equal("Issue created", card.Header.Title)
	a.Empty(card.Sections)

	body, err := marshal(msg)
	a.NoError(err)
	a.NotContains(string(body), `"sections"`)
}

func TestEscapeText(t *testing.T) {
	a := require.New(t)
	a.Equal("hello &amp; world", escapeText("hello & world"))
	a.Equal("a &lt;b&gt; c", escapeText("a <b> c"))
	a.Equal("&#39;quote&#39; &quot;double&quot;", escapeText("'quote' \"double\""))
}

func TestBuildMessageEscapesUserContent(t *testing.T) {
	a := require.New(t)
	ctx := webhook.Context{
		Level:       webhook.WebhookWarn,
		Title:       "Issue <sent> back",
		Description: "Reason: <script>alert('xss')</script>",
		Link:        "https://bb.example.com/issues/1",
		Project:     &webhook.Project{Name: "projects/p", Title: "Project <A>"},
		Issue: &webhook.Issue{
			Name:    "Issue with <angle>",
			Creator: webhook.Creator{Name: "O'Brien & Sons"},
		},
		ActorName: "Admin <root>",
	}

	msg := BuildMessage(ctx)
	card := msg.CardsV2[0].Card
	a.NotNil(card.Header)
	a.Equal("⚠️ Issue &lt;sent&gt; back", card.Header.Title)

	body, err := marshal(msg)
	a.NoError(err)
	bodyText := string(body)
	a.Contains(bodyText, "Issue &lt;sent&gt; back")
	a.Contains(bodyText, "&lt;script&gt;")
	a.Contains(bodyText, "Project &lt;A&gt;")
	a.Contains(bodyText, "O&#39;Brien &amp; Sons")
	a.Contains(bodyText, "Admin &lt;root&gt;")
}

func TestLevelEmoji(t *testing.T) {
	a := require.New(t)
	a.Equal("", levelEmoji(webhook.WebhookInfo))
	a.Equal("✅ ", levelEmoji(webhook.WebhookSuccess))
	a.Equal("⚠️ ", levelEmoji(webhook.WebhookWarn))
	a.Equal("❗ ", levelEmoji(webhook.WebhookError))
}

func TestPostMessage(t *testing.T) {
	a := require.New(t)
	var received MessagePayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.Equal(http.MethodPost, r.Method)
		a.Equal("application/json; charset=UTF-8", r.Header.Get("Content-Type"))
		a.NoError(json.NewDecoder(r.Body).Decode(&received))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"spaces/AAAA/messages/1"}`))
	}))
	defer server.Close()

	err := postMessage(webhook.Context{
		URL:   server.URL,
		Title: "Issue created",
	})

	a.NoError(err)
	a.Empty(received.Text)
}

func TestPostMessageNon2xx(t *testing.T) {
	a := require.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid payload", http.StatusBadRequest)
	}))
	defer server.Close()

	err := postMessage(webhook.Context{
		URL:   server.URL,
		Title: "Issue created",
	})

	a.Error(err)
	a.Contains(err.Error(), "status code: 400")
	a.Contains(err.Error(), "invalid payload")
}

func TestPostMessageNon2xxRedactsGoogleChatCredentials(t *testing.T) {
	a := require.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid payload", http.StatusBadRequest)
	}))
	defer server.Close()

	err := postMessage(webhook.Context{
		URL:   server.URL + "/v1/spaces/AAAA/messages?key=chat-key&token=chat-token&threadKey=thread-key",
		Title: "Issue created",
	})

	a.Error(err)
	a.NotContains(err.Error(), "chat-key")
	a.NotContains(err.Error(), "chat-token")
	a.NotContains(err.Error(), "key=")
	a.NotContains(err.Error(), "token=")
	a.NotContains(err.Error(), "threadKey")
	a.NotContains(err.Error(), "spaces/AAAA")
	a.Contains(err.Error(), "status code: 400")
	a.Contains(err.Error(), "invalid payload")
}

func TestPostMessageNetworkErrorRedactsGoogleChatCredentials(t *testing.T) {
	a := require.New(t)
	err := postMessage(webhook.Context{
		URL:   "http://127.0.0.1:1/v1/spaces/AAAA/messages?key=chat-key&token=chat-token",
		Title: "Issue created",
	})

	a.Error(err)
	a.NotContains(err.Error(), "chat-key")
	a.NotContains(err.Error(), "chat-token")
	a.NotContains(err.Error(), "key=")
	a.NotContains(err.Error(), "token=")
	a.NotContains(err.Error(), "spaces/AAAA")
	a.Contains(err.Error(), "failed to POST Google Chat webhook")
}
