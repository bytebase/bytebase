# Email Service Integration Design

**Date**: 2026-04-12
**Status**: Draft
**Scope**: Backend infrastructure for email delivery configuration. No frontend UI, no invite flow (Spec B).

---

## Goal

Allow Bytebase to send transactional emails (workspace invites, notifications). This spec covers the **email service configuration** layer: storing SMTP credentials at the workspace level via the existing `setting` table, injecting global SMTP config from profile env vars on workspace creation, exposing config via existing `GetSetting`/`UpdateSetting` APIs, and providing a mail sender plugin.

## Decisions (from brainstorming)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Provider scope | SMTP only (day 1) | YAGNI; SMTP covers ~95% of providers. Proto is extensible via `type` enum + `oneof`. |
| Storage | Existing `setting` table with `EMAIL` setting name | No new table/migration. Reuses `UpsertSetting`, caching, `PK(workspace, name)` singleton enforcement. |
| Global config source | Profile env vars, injected on workspace creation | Follows the Gemini/AI pattern (`getAdditionalWorkspaceSettings`). No platform-admin API needed. |
| Resource shape | Singleton via setting table | One EMAIL setting per workspace. Get/Update via existing `SettingService` endpoints. |
| Credentials at rest | Plain JSONB | Consistent with IDP `client_secret` and AI `api_key`. Revisit in a security-hardening pass. |
| Test endpoint | Custom method on `SettingService` | `POST /v1/{parent=workspaces/*}/settings/EMAIL:test`. Keeps everything in one service. |

## Non-goals

- Invite member flow (Spec B).
- Frontend UI for email config.
- Password encryption at rest.
- Email templates / i18n of email bodies.
- Retry / dead-letter queue for failed sends.
- Rate limiting on test endpoint.

---

## 1. Data Model

### Setting table (existing, no changes)

```sql
CREATE TABLE setting (
    name text NOT NULL,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    value jsonb NOT NULL,
    PRIMARY KEY (workspace, name)
);
```

Singleton per workspace is enforced by the composite PK. The new `EMAIL` entry is stored as a `protojson.Marshal`'d `EmailSetting` message in the `value` column.

### SettingName enum addition

In `proto/store/store/setting.proto`:

```proto
enum SettingName {
  // ... existing values 0-8 ...
  EMAIL = 9;
}
```

### Store proto: `EmailSetting` message

Add to `proto/store/store/setting.proto` (alongside `AppIMSetting`, `AISetting`, etc.):

```proto
message EmailSetting {
  string from = 1;
  string from_name = 2;

  Type type = 3;

  enum Type {
    TYPE_UNSPECIFIED = 0;
    SMTP = 1;
    // Future: SENDGRID = 2; MAILGUN = 3; SES = 4;
  }

  oneof config {
    SMTPConfig smtp = 4;
    // Future: SendGridConfig sendgrid = 5; etc.
  }

  message SMTPConfig {
    string host = 1;
    int32 port = 2;
    string username = 3;
    string password = 4;
    Encryption encryption = 5;
    Authentication authentication = 6;

    enum Encryption {
      ENCRYPTION_UNSPECIFIED = 0;
      NONE = 1;       // plain SMTP, port 25
      STARTTLS = 2;   // port 587
      SSL_TLS = 3;    // port 465
    }

    enum Authentication {
      AUTHENTICATION_UNSPECIFIED = 0;
      NONE = 1;
      PLAIN = 2;
      LOGIN = 3;
      CRAM_MD5 = 4;
    }
  }
}
```

The `type` enum + `oneof config` pattern allows adding SendGrid/Mailgun/SES later without breaking stored data.

---

## 2. API Surface

### Existing endpoints (no new proto service)

Email config is read and written through the existing `SettingService`:

- **Get**: `GET /v1/{name=settings/EMAIL}` — existing `GetSetting` RPC. Permission: `bb.settings.get`.
- **Update**: `PATCH /v1/{setting.name=settings/EMAIL}` — existing `UpdateSetting` RPC. Permission: `bb.settings.set`. Audit: true.

Password stripping (`INPUT_ONLY` semantic) is handled in the store→v1 converter for the `EMAIL` case.

### New RPC: TestEmailSetting

Added to `SettingService` in `proto/v1/v1/setting_service.proto`:

```proto
rpc TestEmailSetting(TestEmailSettingRequest) returns (TestEmailSettingResponse) {
  option (google.api.http) = {
    post: "/v1/{parent=workspaces/*}/settings/EMAIL:test"
    body: "*"
  };
  option (bytebase.v1.permission) = "bb.settings.set";
  option (bytebase.v1.auth_method) = IAM;
}

message TestEmailSettingRequest {
  // Parent workspace. Format: workspaces/{workspace}
  string parent = 1 [(google.api.field_behavior) = REQUIRED];
  // The email config to test. Not persisted.
  EmailSetting email_setting = 2 [(google.api.field_behavior) = REQUIRED];
  // The recipient to send the test email to.
  string to = 3 [(google.api.field_behavior) = REQUIRED];
}

message TestEmailSettingResponse {
  bool success = 1;
  // Human-readable error (SMTP error string) if success=false.
  string error = 2;
}
```

### v1 proto additions

In `proto/v1/v1/setting_service.proto`, add `EmailSetting` to the v1 `SettingValue` oneof:

```proto
message SettingValue {
  oneof value {
    // ... existing cases ...
    EmailSetting email = N;  // next available field number
  }
}
```

The v1 `EmailSetting` message mirrors the store proto but with `INPUT_ONLY` on the password field:

```proto
message EmailSetting {
  string from = 1;
  string from_name = 2;
  Type type = 3;

  enum Type {
    TYPE_UNSPECIFIED = 0;
    SMTP = 1;
  }

  oneof config {
    SMTPConfig smtp = 4;
  }

  message SMTPConfig {
    string host = 1;
    int32 port = 2;
    string username = 3;
    string password = 4 [(google.api.field_behavior) = INPUT_ONLY];
    Encryption encryption = 5;
    Authentication authentication = 6;

    enum Encryption { ENCRYPTION_UNSPECIFIED = 0; NONE = 1; STARTTLS = 2; SSL_TLS = 3; }
    enum Authentication { AUTHENTICATION_UNSPECIFIED = 0; NONE = 1; PLAIN = 2; LOGIN = 3; CRAM_MD5 = 4; }
  }
}
```

---

## 3. Store Layer

No new store file. Changes to existing files:

### `backend/store/setting.go`

- `getSettingMessage()` — add case for `storepb.SettingName_EMAIL` returning `&storepb.EmailSetting{}`.
- Cache works automatically (keyed as `"workspaces/{ws}/settings/EMAIL"`).
- `GetSetting` / `UpsertSetting` / `ListSettings` — no changes needed (generic over all setting names).

---

## 4. Global Config Injection

### No profile field

The `EMAIL_CONFIG` env var is **not** stored in `Profile`. Instead, `getAdditionalWorkspaceSettings()` reads it directly from the environment at call time via `os.Getenv("EMAIL_CONFIG")`.

### Workspace creation injection

In `backend/api/v1/auth_service.go`, extend `getAdditionalWorkspaceSettings()`:

```go
func (s *AuthService) getAdditionalWorkspaceSettings() []store.AdditionalSetting {
    var settings []store.AdditionalSetting
    // Existing: Gemini AI injection
    if s.profile.GeminiAPIKey != "" {
        settings = append(settings, store.AdditionalSetting{ /* ... */ })
    }
    // New: email config injection
    if raw := os.Getenv("EMAIL_CONFIG"); raw != "" {
        emailSetting := &storepb.EmailSetting{}
        if err := protojson.Unmarshal([]byte(raw), emailSetting); err != nil {
            slog.Error("failed to parse EMAIL_CONFIG env var", log.BBError(err))
        } else {
            settings = append(settings, store.AdditionalSetting{
                Name:    storepb.SettingName_EMAIL,
                Payload: emailSetting,
            })
        }
    }
    return settings
}
```

Parse errors are logged but don't block workspace creation — a broken env var shouldn't prevent signup.

This means:
- SaaS deployment sets `EMAIL_CONFIG` env var once (JSON string).
- Every new workspace gets the email config pre-configured.
- Workspace admin can later override via `UpdateSetting`.
- Pre-login invite sender (Spec B) reads the workspace's EMAIL setting from the store — no special "global" path needed, since the config was already injected at creation time.

---

## 5. Service Layer

### Changes to `backend/api/v1/setting_service.go`

**`UpdateSetting`** — add `case storepb.SettingName_EMAIL:` in the validation switch:

1. Validate `type` is set and matches the `oneof` variant.
2. For SMTP: validate `host` non-empty, `port` > 0, `from` is a valid email, `encryption` set.
3. **Password handling**: if the incoming `password` is empty, load the existing EMAIL setting and carry over the stored password. This is the standard edit-password semantic (same approach as IDP `client_secret` updates).
4. Call `store.UpsertSetting`.

**`GetSetting`** — no case needed (generic path works). Password stripping happens in the converter.

### Changes to `backend/api/v1/setting_service_converter.go`

**`convertToSettingMessage`** — add `case storepb.SettingName_EMAIL:`:

- Convert `storepb.EmailSetting` → `v1pb.EmailSetting`.
- Strip `SMTPConfig.Password` (set to empty string) before returning.

### New method: `TestEmailSetting`

In `backend/api/v1/setting_service.go`:

1. Parse workspace from `req.Msg.Parent`, verify matches context.
2. Convert v1 `EmailSetting` → store `EmailSetting`.
3. Validate same as UpdateSetting.
4. If empty password and stored EMAIL setting exists → substitute stored password.
5. Build `Sender` via `mail.NewSender(storeSetting)`.
6. Send test email (`Subject: "Bytebase email config test"`).
7. Return `TestEmailSettingResponse{Success: err == nil, Error: errString}`. SMTP failures are returned in the response body, not as gRPC errors.

---

## 6. Mail Sender Plugin

File: `backend/plugin/mail/mail.go`

```go
type Sender interface {
    Send(ctx context.Context, req *SendRequest) error
}

type SendRequest struct {
    To       []string
    Subject  string
    TextBody string
    HTMLBody string
}

func NewSender(cfg *storepb.EmailSetting) (Sender, error)
```

`NewSender` dispatches on `cfg.Type`:
- `SMTP` → returns `smtp.NewSender(cfg.GetSmtp())`.
- Unknown → returns an error.

No registry pattern for v1 (only one provider). Refactor to registry when adding a second provider.

### SMTP implementation

File: `backend/plugin/mail/smtp/smtp.go`

- Uses Go stdlib `net/smtp` + `crypto/tls`. No third-party dependency.
- Handles three encryption modes:
  - `NONE` → `smtp.Dial`
  - `STARTTLS` → `smtp.Dial` + `c.StartTLS()`
  - `SSL_TLS` → `tls.Dial` wrapped in `smtp.NewClient`
- Auth mechanism from `Authentication` enum; `UNSPECIFIED` defaults to `PLAIN`.
- Connection timeout: 10s. Send timeout: 30s.
- Builds RFC-5322 message. `multipart/alternative` if HTML body present, `text/plain` otherwise.

---

## 7. Error Handling

| Scenario | Response |
|----------|----------|
| Validation error (bad type, missing fields, invalid email) | `CodeInvalidArgument`, field-specific message |
| No EMAIL setting exists for workspace | `GetSetting` returns nil → converter returns empty/default |
| SMTP failure in `TestEmailSetting` | `TestEmailSettingResponse{success: false, error: "..."}` (NOT a gRPC error) |
| SMTP failure in real send (future invite) | Returned as Go error to caller |
| Unknown `Type` in store → v1 conversion | Log warning, return generic fields only |

---

## 8. Testing

### Unit tests

- `backend/plugin/mail/smtp/smtp_test.go`: in-process SMTP listener, verify three encryption modes, four auth mechanisms, message formatting.

### Integration tests

- `backend/tests/mail_config_test.go`:
  - `GetSetting(EMAIL)` → nil/empty (not configured).
  - `UpdateSetting(EMAIL)` → success.
  - `GetSetting(EMAIL)` → password stripped.
  - Update with empty password → preserved.
  - `TestEmailSetting` with local SMTP fake → `success=true`.
  - `TestEmailSetting` against unreachable host → `success=false`, error populated.

---

## 9. Rollout

- **No migration**: `setting` table already exists. `EMAIL` rows created via `UpsertSetting` on first use or workspace creation.
- **Proto generation**: `cd proto && buf format -w . && buf lint && buf generate`.
- **No new permissions**: reuses existing `bb.settings.get` / `bb.settings.set`.
- **No feature flag**: the feature is dormant until Spec B adds the invite flow.
- **Not a breaking change**: nothing removed or renamed.
- **Profile env var**: single `EMAIL_CONFIG` (JSON-stringified `EmailSetting` proto).
