# Google Chat Webhook Integration Design

**Date**: 2026-04-23
**Status**: Approved
**Linear**: BYT-9249
**Scope**: Project webhook notifications to Google Chat spaces. Channel notifications only; direct messages are out of scope for this pass.

---

## Goal

Add Google Chat as a first-class project webhook destination for the same approval and rollout notification events already supported by Slack and Microsoft Teams:

- Issue created
- Approval required
- Issue sent back
- Issue approved
- Rollout failed
- Rollout completed

The first release uses Google Chat incoming webhooks, which post asynchronous one-way messages into a Chat space. This matches the Linear direction to start with channel notifications and handle direct messages later.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Integration model | Project webhook provider | Bytebase already converts events into `webhook.Context`; each provider owns formatting and HTTP delivery. |
| Day-1 target | Google Chat spaces via incoming webhook URL | Fits the customer request without adding OAuth, service accounts, or user identity mapping. |
| Direct messages | Not supported initially | Google Chat DM support needs Chat API auth and user resolution; it does not fit the simple incoming webhook path. |
| Workspace IM setting | No `AppIMSetting` entry for Google Chat | Existing IM settings are used for provider credentials that enable DM delivery. Channel webhooks store the destination URL on each project webhook. |
| URL validation | Allow `chat.googleapis.com` and validate the webhook URL shape | Existing providers use domain allowlists. Google Chat webhook URLs have a stable credential-bearing `/v1/spaces/.../messages?key=...&token=...` shape, so validating path and required query params catches configuration mistakes early. |
| Message format | Google Chat `cardsV2` plus a plain `text` summary | Cards give parity with Slack/Teams polish. The `text` field gives Chat a compact notification summary. If manual pre-merge testing shows incoming webhooks reject the selected card shape, simplify the builder to text-only before merging. |
| Schema migration | None | Project webhooks are stored in JSONB payloads; adding a new enum value does not require a table change. |

## Non-goals

- Google Chat direct messages.
- Google Chat OAuth, service account credentials, or Chat API space/user management.
- A workspace-level Google Chat IM integration page.
- Notification event redesign.
- Retry or delivery status persistence beyond the existing webhook send behavior.

---

## Architecture

Bytebase already has the right provider boundary:

1. Application code creates a `backend/component/webhook.Event`.
2. `backend/component/webhook.Manager` looks up project webhooks subscribed to the event.
3. The manager converts the event to `backend/plugin/webhook.Context`.
4. `backend/plugin/webhook.Post(type, context)` dispatches to the registered provider receiver.

Google Chat should be added as another provider receiver, not as a parallel notification pipeline.

### Proto and Generated Types

Add `GOOGLE_CHAT = 8` to both webhook type enums:

- `proto/store/store/common.proto`
- `proto/v1/v1/common.proto`

Then regenerate backend and frontend generated code with the repo's proto workflow:

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
```

The project webhook proto does not need a new field. Existing fields are enough:

- `type = GOOGLE_CHAT`
- `title`
- `url`
- `notification_types`
- `direct_message = false`

### Backend Provider

Create a new provider package:

- `backend/plugin/webhook/googlechat/googlechat.go`
- `backend/plugin/webhook/googlechat/googlechat_test.go`

The receiver should:

- register itself with `webhook.Register(storepb.WebhookType_GOOGLE_CHAT, &Receiver{})`
- build a Google Chat message from `webhook.Context`
- post JSON to `context.URL`
- use `webhook.Timeout`
- set `Content-Type: application/json; charset=UTF-8`
- return a detailed error when the HTTP status is outside `2xx`

Register the package with blank imports in:

- `backend/server/minimal.go`
- `backend/server/ultimate.go`

`backend/api/v1/setting_service.go` does not need a Google Chat import because Google Chat has no workspace IM credentials in this scope.

### Message Shape

The message builder should produce a small, durable structure from existing `webhook.Context` fields:

- summary text: event title
- title: event title, linked or paired with a "View in Bytebase" button when `Link` exists
- description when present
- issue or rollout title
- compact metadata:
  - project title
  - issue creator for issue events
  - environment for rollout events
  - actor when present

The builder should escape user-controlled text for the selected Google Chat format. Use the current Google Chat `cardsV2` message structure and keep a plain `text` field as a compact notification summary. Do not implement a second runtime POST fallback path; if the card payload is not accepted during manual validation, simplify the message builder before merging.

### URL Validation

Extend `backend/plugin/webhook/validator.go` for Google Chat:

- allowed host: `chat.googleapis.com`
- required scheme: `https`
- required path shape: `/v1/spaces/{space}/messages`
- required query params: non-empty `key` and `token`

This is stricter than the current domain-only validation for most providers, but it is appropriate for Google Chat because valid incoming webhook URLs have a well-known shape and the query parameters are required credentials.

### Frontend

Add Google Chat to the project webhook UI only:

- `frontend/src/types/v1/projectWebhook.ts`
  - name: `Google Chat`
  - placeholder: `https://chat.googleapis.com/v1/spaces/.../messages?key=...&token=...`
  - doc URL: `https://developers.google.com/workspace/chat/quickstart/webhooks`
  - `supportDirectMessage: false`
- `frontend/src/react/components/WebhookTypeIcon.tsx`
  - add a Google Chat icon asset
- locale files
  - add `common.google-chat` or equivalent existing naming pattern

Do not add Google Chat to `frontend/src/react/pages/settings/IMPage.tsx`, because there is no workspace-level IM credential for the channel-only integration.

---

## Data Flow

1. A user creates a project webhook with type `GOOGLE_CHAT`, a Google Chat incoming webhook URL, and selected notification events.
2. `ProjectService.AddWebhook` converts the v1 webhook to the store payload and validates the URL.
3. When a subscribed activity occurs, `Manager.CreateEvent` loads matching project webhooks and builds one `webhook.Context`.
4. `webhook.Post(GOOGLE_CHAT, ctx)` dispatches to the Google Chat receiver.
5. The receiver posts the formatted JSON payload to the Google Chat webhook URL.
6. Google Chat renders the message in the configured Chat space.

## Error Handling

- Invalid Google Chat URL shape returns `InvalidArgument` during create, update, and test webhook flows.
- Unknown webhook type remains rejected by the shared validator.
- HTTP non-2xx responses from Google Chat return an error containing status code and response body.
- Runtime send failures continue to follow existing webhook manager behavior: retry through `common.Retry`, then log a warning without blocking the originating user flow.
- Direct message attempts should not be possible from the UI because Google Chat reports `supportDirectMessage: false`. If old or hand-written API input sets `direct_message = true`, the Google Chat receiver should ignore it and post to the configured webhook URL.

## Testing

Backend:

- `backend/plugin/webhook/validator_test.go`
  - accepts a valid Google Chat incoming webhook URL
  - rejects wrong host
  - rejects missing `key`
  - rejects missing `token`
  - rejects non-`/v1/spaces/.../messages` path
  - rejects non-HTTPS Google Chat URL
- `backend/plugin/webhook/googlechat/googlechat_test.go`
  - builds approval-required message with issue metadata and Bytebase link
  - builds sent-back and approved issue messages
  - builds rollout failed and completed messages with environment metadata
  - escapes user-controlled text
  - posts successfully to an `httptest.Server`
  - reports status and body on non-2xx response

Frontend:

- Existing project webhook form behavior should continue to show all providers.
- Google Chat should appear as a project webhook destination.
- Google Chat should not show direct message controls.

Manual:

- Create a Google Chat incoming webhook in a test Chat space.
- Create a Bytebase project webhook using that URL.
- Run the test webhook action.
- Trigger at least one approval event and one rollout event.
- Confirm the message renders cleanly and the Bytebase link opens the expected project page.

## References

- Google Chat incoming webhooks: https://developers.google.com/workspace/chat/quickstart/webhooks
- Google Chat message creation and cards: https://developers.google.com/workspace/chat/create-messages
