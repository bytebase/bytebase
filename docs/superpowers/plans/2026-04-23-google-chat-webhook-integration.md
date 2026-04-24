# Google Chat Webhook Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Google Chat as a channel-only project webhook destination for Bytebase approval and rollout notifications.

**Architecture:** Extend the existing project webhook provider model. Add a new `GOOGLE_CHAT` webhook enum value, convert it through the v1/store boundary, validate Google Chat incoming webhook URLs, register a Google Chat receiver, and expose it in the project webhook UI with direct messages disabled.

**Tech Stack:** Go, protobuf/buf, ConnectRPC service converters, Bytebase webhook plugin registry, React/TypeScript, Vite/Vitest, Tailwind UI conventions.

---

## Scope Check

The approved design covers one integration path: Google Chat incoming webhooks to Chat spaces. It does not include workspace IM settings, direct messages, OAuth, service accounts, or Google Chat API user resolution. This is one implementation plan because the backend provider and frontend option are coupled by a single new `WebhookType`.

## File Map

- Modify `proto/store/store/common.proto`: add store enum value `GOOGLE_CHAT = 8`.
- Modify `proto/v1/v1/common.proto`: add v1 enum value `GOOGLE_CHAT = 8`.
- Regenerate protobuf output under `backend/generated-go/` and `frontend/src/types/proto-es/`.
- Modify `backend/api/v1/project_service_converter.go`: map Google Chat between v1 and store webhook types.
- Modify `backend/plugin/webhook/validator.go`: allow `chat.googleapis.com` and enforce Google Chat URL shape.
- Modify `backend/plugin/webhook/validator_test.go`: cover valid and invalid Google Chat URL cases.
- Create `backend/plugin/webhook/googlechat/googlechat.go`: provider payload builder and HTTP sender.
- Create `backend/plugin/webhook/googlechat/googlechat_test.go`: payload, escaping, and HTTP behavior tests.
- Modify `backend/server/minimal.go`: blank-import Google Chat provider.
- Modify `backend/server/ultimate.go`: blank-import Google Chat provider.
- Create `frontend/src/assets/im/google-chat.svg`: local SVG asset for the destination selector.
- Modify `frontend/src/react/components/WebhookTypeIcon.tsx`: map Google Chat to the new asset.
- Modify `frontend/src/types/v1/projectWebhook.ts`: expose Google Chat in the project webhook destination list with `supportDirectMessage: false`.
- Create `frontend/src/types/v1/projectWebhook.test.ts`: assert Google Chat appears and direct messages are disabled.
- Modify `frontend/src/locales/{en-US,es-ES,ja-JP,vi-VN,zh-CN}.json`: add `common.google-chat`.

---

### Task 1: Proto Enum and Converter Wiring

**Files:**
- Modify: `proto/store/store/common.proto`
- Modify: `proto/v1/v1/common.proto`
- Modify: `backend/api/v1/project_service_converter.go:101-143`
- Generated: `backend/generated-go/**`
- Generated: `frontend/src/types/proto-es/**`

- [ ] **Step 1: Add failing converter test by extending the existing package tests**

Create `backend/api/v1/project_service_converter_webhook_test.go`:

```go
package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestConvertWebhookTypeGoogleChat(t *testing.T) {
	a := require.New(t)

	storeType, err := convertToStoreWebhookType(v1pb.WebhookType_GOOGLE_CHAT)
	a.NoError(err)
	a.Equal(storepb.WebhookType_GOOGLE_CHAT, storeType)

	v1Type := convertToV1WebhookType(storepb.WebhookType_GOOGLE_CHAT)
	a.Equal(v1pb.WebhookType_GOOGLE_CHAT, v1Type)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test -v -count=1 ./backend/api/v1 -run ^TestConvertWebhookTypeGoogleChat$
```

Expected: FAIL because `WebhookType_GOOGLE_CHAT` is undefined.

- [ ] **Step 3: Add the proto enum values**

In `proto/store/store/common.proto`, add after `LARK = 7;`:

```proto
  // Google Chat integration.
  GOOGLE_CHAT = 8;
```

In `proto/v1/v1/common.proto`, add after `LARK = 7;`:

```proto
  // Google Chat integration.
  GOOGLE_CHAT = 8;
```

- [ ] **Step 4: Format, lint, and regenerate protobuf outputs**

Run:

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
```

Expected: all commands exit 0 and generated Go/TypeScript enum definitions include `GOOGLE_CHAT`.

- [ ] **Step 5: Add converter cases**

In `backend/api/v1/project_service_converter.go`, update `convertToStoreWebhookType`:

```go
	case v1pb.WebhookType_GOOGLE_CHAT:
		return storepb.WebhookType_GOOGLE_CHAT, nil
```

Update `convertToV1WebhookType`:

```go
	case storepb.WebhookType_GOOGLE_CHAT:
		return v1pb.WebhookType_GOOGLE_CHAT
```

- [ ] **Step 6: Run the converter test to verify it passes**

Run:

```bash
go test -v -count=1 ./backend/api/v1 -run ^TestConvertWebhookTypeGoogleChat$
```

Expected: PASS.

- [ ] **Step 7: Commit the proto and converter slice**

Run:

```bash
git add proto/store/store/common.proto proto/v1/v1/common.proto backend/generated-go frontend/src/types/proto-es backend/api/v1/project_service_converter.go backend/api/v1/project_service_converter_webhook_test.go
git commit -m "feat: add google chat webhook type"
```

---

### Task 2: Google Chat URL Validation

**Files:**
- Modify: `backend/plugin/webhook/validator.go`
- Modify: `backend/plugin/webhook/validator_test.go`

- [ ] **Step 1: Add failing Google Chat validator test cases**

In `backend/plugin/webhook/validator_test.go`, add these cases inside `TestValidateWebhookURL` before the URL format tests:

```go
		// Google Chat tests
		{
			name:        "valid google chat URL",
			webhookType: storepb.WebhookType_GOOGLE_CHAT,
			webhookURL:  "https://chat.googleapis.com/v1/spaces/AAAA123/messages?key=chat-key&token=chat-token",
			wantErr:     false,
		},
		{
			name:        "invalid google chat domain",
			webhookType: storepb.WebhookType_GOOGLE_CHAT,
			webhookURL:  "https://evil.googleapis.com/v1/spaces/AAAA123/messages?key=chat-key&token=chat-token",
			wantErr:     true,
		},
		{
			name:        "invalid google chat scheme",
			webhookType: storepb.WebhookType_GOOGLE_CHAT,
			webhookURL:  "http://chat.googleapis.com/v1/spaces/AAAA123/messages?key=chat-key&token=chat-token",
			wantErr:     true,
		},
		{
			name:        "invalid google chat path",
			webhookType: storepb.WebhookType_GOOGLE_CHAT,
			webhookURL:  "https://chat.googleapis.com/v1/spaces/AAAA123?key=chat-key&token=chat-token",
			wantErr:     true,
		},
		{
			name:        "google chat missing key",
			webhookType: storepb.WebhookType_GOOGLE_CHAT,
			webhookURL:  "https://chat.googleapis.com/v1/spaces/AAAA123/messages?token=chat-token",
			wantErr:     true,
		},
		{
			name:        "google chat missing token",
			webhookType: storepb.WebhookType_GOOGLE_CHAT,
			webhookURL:  "https://chat.googleapis.com/v1/spaces/AAAA123/messages?key=chat-key",
			wantErr:     true,
		},
```

- [ ] **Step 2: Run the validator tests to verify they fail**

Run:

```bash
go test -v -count=1 ./backend/plugin/webhook -run ^TestValidateWebhookURL$
```

Expected: FAIL because Google Chat is not in the allowlist.

- [ ] **Step 3: Add Google Chat allowlist and shape validation**

In `backend/plugin/webhook/validator.go`, add Google Chat to `allowedDomains`:

```go
		storepb.WebhookType_GOOGLE_CHAT: {
			"chat.googleapis.com",
		},
```

Add this helper below `ValidateWebhookURL`:

```go
func validateGoogleChatURL(u *url.URL) error {
	if u.Scheme != "https" {
		return errors.Errorf("invalid Google Chat URL scheme: %s (only https is allowed)", u.Scheme)
	}

	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(parts) != 4 || parts[0] != "v1" || parts[1] != "spaces" || parts[2] == "" || parts[3] != "messages" {
		return errors.Errorf("invalid Google Chat webhook path: %s", u.EscapedPath())
	}

	query := u.Query()
	if query.Get("key") == "" {
		return errors.Errorf("missing Google Chat webhook key")
	}
	if query.Get("token") == "" {
		return errors.Errorf("missing Google Chat webhook token")
	}

	return nil
}
```

Inside `ValidateWebhookURL`, after the hostname/domain match succeeds, call the helper for Google Chat:

```go
			if hostname == domain {
				if webhookType == storepb.WebhookType_GOOGLE_CHAT {
					return validateGoogleChatURL(u)
				}
				return nil
			}
```

Keep wildcard-domain behavior unchanged for the existing providers.

- [ ] **Step 4: Run the validator tests to verify they pass**

Run:

```bash
go test -v -count=1 ./backend/plugin/webhook -run ^TestValidateWebhookURL$
```

Expected: PASS.

- [ ] **Step 5: Commit the validator slice**

Run:

```bash
git add backend/plugin/webhook/validator.go backend/plugin/webhook/validator_test.go
git commit -m "feat: validate google chat webhook urls"
```

---

### Task 3: Google Chat Provider Payload and HTTP Sender

**Files:**
- Create: `backend/plugin/webhook/googlechat/googlechat.go`
- Create: `backend/plugin/webhook/googlechat/googlechat_test.go`
- Modify: `backend/server/minimal.go`
- Modify: `backend/server/ultimate.go`

- [ ] **Step 1: Write failing provider tests**

Create `backend/plugin/webhook/googlechat/googlechat_test.go`:

```go
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

	a.Equal("Issue approved", msg.Text)
	a.Len(msg.CardsV2, 1)
	card := msg.CardsV2[0].Card
	a.NotNil(card.Header)
	a.Contains(card.Header.Title, "Issue approved")
	a.Contains(card.Header.Subtitle, "My Project")
	a.NotEmpty(card.Sections)

	body, err := json.Marshal(msg)
	a.NoError(err)
	bodyText := string(body)
	a.Contains(bodyText, "Grant read access to prod")
	a.Contains(bodyText, "Need access for release verification")
	a.Contains(bodyText, "Alice")
	a.Contains(bodyText, "Bob")
	a.Contains(bodyText, "View in Bytebase")
	a.Contains(bodyText, "https://bb.example.com/projects/proj-1/issues/42")
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

	body, err := json.Marshal(msg)
	a.NoError(err)
	bodyText := string(body)
	a.Contains(bodyText, "Rollout failed")
	a.Contains(bodyText, "Deploy v2")
	a.Contains(bodyText, "environments/prod")
	a.Contains(bodyText, "My Project")
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
	body, err := json.Marshal(msg)
	a.NoError(err)
	bodyText := string(body)
	a.Contains(bodyText, "Issue &lt;sent&gt; back")
	a.Contains(bodyText, "&lt;script&gt;")
	a.Contains(bodyText, "Project &lt;A&gt;")
	a.Contains(bodyText, "O&#39;Brien &amp; Sons")
	a.Contains(bodyText, "Admin &lt;root&gt;")
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
	a.Equal("Issue created", received.Text)
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
```

- [ ] **Step 2: Run provider tests to verify they fail**

Run:

```bash
go test -v -count=1 ./backend/plugin/webhook/googlechat
```

Expected: FAIL because the package implementation does not exist.

- [ ] **Step 3: Implement the Google Chat receiver**

Create `backend/plugin/webhook/googlechat/googlechat.go`:

```go
// Package googlechat implements Google Chat incoming webhook integration.
package googlechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

type MessagePayload struct {
	Text    string       `json:"text"`
	CardsV2 []CardWithID `json:"cardsV2,omitempty"`
}

type CardWithID struct {
	CardID string `json:"cardId,omitempty"`
	Card   Card   `json:"card"`
}

type Card struct {
	Header   *CardHeader `json:"header,omitempty"`
	Sections []Section   `json:"sections,omitempty"`
}

type CardHeader struct {
	Title    string `json:"title,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
}

type Section struct {
	Widgets []Widget `json:"widgets,omitempty"`
}

type Widget struct {
	TextParagraph *TextParagraph `json:"textParagraph,omitempty"`
	ButtonList    *ButtonList    `json:"buttonList,omitempty"`
}

type TextParagraph struct {
	Text string `json:"text"`
}

type ButtonList struct {
	Buttons []Button `json:"buttons"`
}

type Button struct {
	Text    string   `json:"text"`
	OnClick OnClick `json:"onClick"`
}

type OnClick struct {
	OpenLink OpenLink `json:"openLink"`
}

type OpenLink struct {
	URL string `json:"url"`
}

func init() {
	webhook.Register(storepb.WebhookType_GOOGLE_CHAT, &Receiver{})
}

type Receiver struct{}

func (*Receiver) Post(context webhook.Context) error {
	return postMessage(context)
}

func BuildMessage(ctx webhook.Context) MessagePayload {
	sections := []Section{}
	widgets := []Widget{}

	if ctx.Description != "" {
		widgets = append(widgets, textWidget(ctx.Description))
	}
	if ctx.Issue != nil {
		widgets = append(widgets, textWidgetHTML(fmt.Sprintf("<b>%s</b>", escapeText(ctx.Issue.Name))))
		if ctx.Issue.Description != "" {
			widgets = append(widgets, textWidget(ctx.Issue.Description))
		}
	} else if ctx.Rollout != nil && ctx.Rollout.Title != "" {
		widgets = append(widgets, textWidgetHTML(fmt.Sprintf("<b>%s</b>", escapeText(ctx.Rollout.Title))))
	}

	metadata := metadataLines(ctx)
	if len(metadata) > 0 {
		widgets = append(widgets, textWidgetHTML(strings.Join(metadata, "<br>")))
	}

	if len(widgets) > 0 {
		sections = append(sections, Section{Widgets: widgets})
	}
	if ctx.Link != "" {
		sections = append(sections, Section{
			Widgets: []Widget{{
				ButtonList: &ButtonList{
					Buttons: []Button{{
						Text: "View in Bytebase",
						OnClick: OnClick{OpenLink: OpenLink{
							URL: ctx.Link,
						}},
					}},
				},
			}},
		})
	}

	subtitle := ""
	if ctx.Project != nil {
		subtitle = escapeText(ctx.Project.Title)
	}

	return MessagePayload{
		Text: ctx.Title,
		CardsV2: []CardWithID{{
			CardID: "bytebase-notification",
			Card: Card{
				Header: &CardHeader{
					Title:    levelPrefix(ctx.Level) + escapeText(ctx.Title),
					Subtitle: subtitle,
				},
				Sections: sections,
			},
		}},
	}
}

func metadataLines(ctx webhook.Context) []string {
	var lines []string
	if ctx.Project != nil {
		lines = append(lines, fmt.Sprintf("<b>Project:</b> %s", escapeText(ctx.Project.Title)))
	}
	if ctx.Issue != nil && ctx.Issue.Creator.Name != "" {
		lines = append(lines, fmt.Sprintf("<b>Creator:</b> %s", escapeText(ctx.Issue.Creator.Name)))
	}
	if ctx.Rollout != nil && ctx.Environment != "" {
		lines = append(lines, fmt.Sprintf("<b>Environment:</b> %s", escapeText(ctx.Environment)))
	}
	if ctx.ActorName != "" {
		lines = append(lines, fmt.Sprintf("<b>By:</b> %s", escapeText(ctx.ActorName)))
	}
	return lines
}

func textWidget(text string) Widget {
	return textWidgetHTML(escapeText(text))
}

func textWidgetHTML(text string) Widget {
	return Widget{
		TextParagraph: &TextParagraph{
			Text: text,
		},
	}
}

func levelPrefix(level webhook.Level) string {
	switch level {
	case webhook.WebhookSuccess:
		return "SUCCESS: "
	case webhook.WebhookWarn:
		return "ACTION REQUIRED: "
	case webhook.WebhookError:
		return "ERROR: "
	default:
		return ""
	}
}

func escapeText(s string) string {
	return html.EscapeString(s)
}

func postMessage(context webhook.Context) error {
	body, err := json.Marshal(BuildMessage(context))
	if err != nil {
		return errors.Wrapf(err, "failed to marshal webhook POST request to %s", context.URL)
	}
	req, err := http.NewRequest(http.MethodPost, context.URL, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrapf(err, "failed to construct webhook POST request to %s", context.URL)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{Timeout: webhook.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to POST webhook to %s", context.URL)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read POST webhook response from %s", context.URL)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.Errorf("failed to POST webhook %s, status code: %d, response body: %s", context.URL, resp.StatusCode, b)
	}
	return nil
}
```

- [ ] **Step 4: Run provider tests to verify they pass**

Run:

```bash
go test -v -count=1 ./backend/plugin/webhook/googlechat
```

Expected: PASS.

- [ ] **Step 5: Register the provider**

In `backend/server/minimal.go`, add:

```go
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/googlechat"
```

In `backend/server/ultimate.go`, add:

```go
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/googlechat"
```

- [ ] **Step 6: Run focused provider tests**

Run:

```bash
go test -v -count=1 ./backend/plugin/webhook/googlechat ./backend/plugin/webhook -run 'Test(BuildMessage|PostMessage|EscapeText|ValidateWebhookURL)'
```

Expected: PASS.

- [ ] **Step 7: Commit the provider slice**

Run:

```bash
git add backend/plugin/webhook/googlechat backend/server/minimal.go backend/server/ultimate.go
git commit -m "feat: add google chat webhook receiver"
```

---

### Task 4: Frontend Webhook Destination

**Files:**
- Create: `frontend/src/assets/im/google-chat.svg`
- Modify: `frontend/src/react/components/WebhookTypeIcon.tsx`
- Modify: `frontend/src/types/v1/projectWebhook.ts`
- Create: `frontend/src/types/v1/projectWebhook.test.ts`
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/es-ES.json`
- Modify: `frontend/src/locales/ja-JP.json`
- Modify: `frontend/src/locales/vi-VN.json`
- Modify: `frontend/src/locales/zh-CN.json`

- [ ] **Step 1: Write failing frontend test**

Create `frontend/src/types/v1/projectWebhook.test.ts`:

```ts
import { describe, expect, test } from "vitest";
import { WebhookType } from "../proto-es/v1/common_pb";
import { projectWebhookV1TypeItemList } from "./projectWebhook";

describe("projectWebhookV1TypeItemList", () => {
  test("includes Google Chat as a channel-only webhook destination", () => {
    const item = projectWebhookV1TypeItemList().find(
      (item) => item.type === WebhookType.GOOGLE_CHAT
    );

    expect(item).toMatchObject({
      name: "Google Chat",
      urlPrefix: "https://chat.googleapis.com",
      urlPlaceholder:
        "https://chat.googleapis.com/v1/spaces/.../messages?key=...&token=...",
      docUrl: "https://developers.google.com/workspace/chat/quickstart/webhooks",
      supportDirectMessage: false,
    });
  });
});
```

- [ ] **Step 2: Run the frontend test to verify it fails**

Run:

```bash
pnpm --dir frontend test -- projectWebhook.test.ts
```

Expected: FAIL because the project webhook list does not contain Google Chat.

- [ ] **Step 3: Add the Google Chat SVG asset**

Create `frontend/src/assets/im/google-chat.svg`:

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48" role="img" aria-label="Google Chat">
  <path fill="#34A853" d="M8 10a6 6 0 0 1 6-6h20a6 6 0 0 1 6 6v15a6 6 0 0 1-6 6H21L11 41v-10a6 6 0 0 1-3-5V10Z"/>
  <path fill="#FBBC04" d="M14 13h20v5H14z"/>
  <path fill="#4285F4" d="M14 22h15v5H14z"/>
  <path fill="#EA4335" d="M34 4a6 6 0 0 1 6 6v15a6 6 0 0 1-6 6h-3V4h3Z" opacity=".9"/>
</svg>
```

- [ ] **Step 4: Add icon mapping**

In `frontend/src/react/components/WebhookTypeIcon.tsx`, add the import:

```ts
import googleChatIcon from "@/assets/im/google-chat.svg";
```

Add the map entry:

```ts
  [WebhookType.GOOGLE_CHAT]: googleChatIcon,
```

- [ ] **Step 5: Add locale keys**

Add `"google-chat": "Google Chat"` to the nested `common` object in these files. Keep each `common` object sorted alphabetically so `frontend/src/plugins/i18n.test.ts` continues to pass.

```text
frontend/src/locales/en-US.json
frontend/src/locales/es-ES.json
frontend/src/locales/ja-JP.json
frontend/src/locales/vi-VN.json
frontend/src/locales/zh-CN.json
```

Use the same value `Google Chat` in each locale file because this is the product name.

- [ ] **Step 6: Add Google Chat to project webhook type list**

In `frontend/src/types/v1/projectWebhook.ts`, add this item after Teams:

```ts
    {
      type: WebhookType.GOOGLE_CHAT,
      name: t("common.google-chat"),
      urlPrefix: "https://chat.googleapis.com",
      urlPlaceholder:
        "https://chat.googleapis.com/v1/spaces/.../messages?key=...&token=...",
      docUrl: "https://developers.google.com/workspace/chat/quickstart/webhooks",
      supportDirectMessage: false,
    },
```

- [ ] **Step 7: Run focused frontend test**

Run:

```bash
pnpm --dir frontend test -- projectWebhook.test.ts
```

Expected: PASS.

- [ ] **Step 8: Run frontend fix and checks**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
```

Expected: all commands exit 0.

- [ ] **Step 9: Commit the frontend slice**

Run:

```bash
git add frontend/src/assets/im/google-chat.svg frontend/src/react/components/WebhookTypeIcon.tsx frontend/src/types/v1/projectWebhook.ts frontend/src/types/v1/projectWebhook.test.ts frontend/src/locales/en-US.json frontend/src/locales/es-ES.json frontend/src/locales/ja-JP.json frontend/src/locales/vi-VN.json frontend/src/locales/zh-CN.json
git commit -m "feat(frontend): expose google chat webhook destination"
```

---

### Task 5: End-to-End Verification

**Files:**
- No new source files.
- Uses all files changed by Tasks 1-4.

- [ ] **Step 1: Run Go formatting**

Run:

```bash
gofmt -w backend/api/v1/project_service_converter.go backend/api/v1/project_service_converter_webhook_test.go backend/plugin/webhook/validator.go backend/plugin/webhook/validator_test.go backend/plugin/webhook/googlechat/googlechat.go backend/plugin/webhook/googlechat/googlechat_test.go backend/server/minimal.go backend/server/ultimate.go
```

Expected: command exits 0.

- [ ] **Step 2: Run focused backend tests**

Run:

```bash
go test -v -count=1 ./backend/api/v1 ./backend/plugin/webhook ./backend/plugin/webhook/googlechat -run '^(TestConvertWebhookTypeGoogleChat|TestValidateWebhookURL|TestBuildMessage|TestEscapeText|TestPostMessage)'
```

Expected: PASS.

- [ ] **Step 3: Run full backend lint until clean**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: PASS. If lint reports issues, run:

```bash
golangci-lint run --fix --allow-parallel-runners
golangci-lint run --allow-parallel-runners
```

Repeat the second command until it exits 0.

- [ ] **Step 4: Run relevant frontend tests**

Run:

```bash
pnpm --dir frontend test -- projectWebhook.test.ts
pnpm --dir frontend check
pnpm --dir frontend type-check
```

Expected: all commands exit 0.

- [ ] **Step 5: Build backend**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: command exits 0 and writes `./bytebase-build/bytebase`.

- [ ] **Step 6: Run manual Google Chat webhook test**

Use a Google Chat incoming webhook URL from a test Chat space.

1. Start Bytebase with a configured external URL.
2. In a project, create a Google Chat webhook with that URL.
3. Click Test Webhook.
4. Trigger one approval notification event.
5. Trigger one rollout completion or failure event.

Expected:

- Google Chat receives the test message.
- Google Chat receives the approval event.
- Google Chat receives the rollout event.
- Each message includes the event title, relevant issue or rollout title, project metadata, and a working Bytebase link.
- The project webhook form does not show direct message controls for Google Chat.

- [ ] **Step 7: Final status check**

Run:

```bash
git status --short
git log --oneline -5
```

Expected: only intentional changes are present, and the last commits correspond to the task slices above.
