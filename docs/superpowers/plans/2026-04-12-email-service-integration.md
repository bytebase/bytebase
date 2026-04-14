# Email Service Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add email (SMTP) configuration as a workspace setting, with a mail sender plugin and test-email endpoint.

**Architecture:** Email config is stored as a new `EMAIL` setting in the existing `setting` table (no new DB table). Global SMTP config is injected from `EMAIL_CONFIG` env var during workspace creation via `getAdditionalWorkspaceSettings()`. A `backend/plugin/mail/` plugin provides the `Sender` interface for sending emails.

**Tech Stack:** Go stdlib `net/smtp` + `crypto/tls`, protobuf, Connect-RPC.

**Spec:** `docs/superpowers/specs/2026-04-12-email-service-integration-design.md`

---

### Task 1: Store proto — add EMAIL setting name and EmailSetting message

**Files:**
- Modify: `proto/store/store/setting.proto:13-23` (SettingName enum) and append after line 342

- [ ] **Step 1: Add EMAIL to SettingName enum**

In `proto/store/store/setting.proto`, add `EMAIL = 9` after `ENVIRONMENT = 8`:

```proto
enum SettingName {
  SETTING_NAME_UNSPECIFIED = 0;
  SYSTEM = 1;
  WORKSPACE_PROFILE = 2;
  WORKSPACE_APPROVAL = 3;
  APP_IM = 4;
  AI = 5;
  DATA_CLASSIFICATION = 6;
  SEMANTIC_TYPES = 7;
  ENVIRONMENT = 8;
  EMAIL = 9;
}
```

- [ ] **Step 2: Add EmailSetting message**

Append after the `EnvironmentSetting` message (after line 342):

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
    string password = 4;
    Encryption encryption = 5;
    Authentication authentication = 6;

    enum Encryption {
      ENCRYPTION_UNSPECIFIED = 0;
      NONE = 1;
      STARTTLS = 2;
      SSL_TLS = 3;
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

- [ ] **Step 3: Lint proto**

Run: `cd proto && buf format -w . && buf lint`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add proto/store/store/setting.proto
git commit -m "proto(store): add EMAIL setting name and EmailSetting message"
```

---

### Task 2: V1 proto — add EMAIL to SettingValue, EmailSetting message, and TestEmailSetting RPC

**Files:**
- Modify: `proto/v1/v1/setting_service.proto:93-102` (SettingName enum), `:114-124` (SettingValue oneof), `:19-49` (service RPCs), and append messages at EOF

- [ ] **Step 1: Add EMAIL to v1 SettingName enum**

In `proto/v1/v1/setting_service.proto`, add `EMAIL = 8` after `ENVIRONMENT = 7` (line 101):

```proto
  enum SettingName {
    SETTING_NAME_UNSPECIFIED = 0;
    WORKSPACE_PROFILE = 1;
    WORKSPACE_APPROVAL = 2;
    APP_IM = 3;
    AI = 4;
    DATA_CLASSIFICATION = 5;
    SEMANTIC_TYPES = 6;
    ENVIRONMENT = 7;
    EMAIL = 8;
  }
```

- [ ] **Step 2: Add email to SettingValue oneof**

Add `EmailSetting email = 8;` to the `SettingValue` message (after line 122):

```proto
message SettingValue {
  oneof value {
    AppIMSetting app_im = 1;
    WorkspaceProfileSetting workspace_profile = 2;
    WorkspaceApprovalSetting workspace_approval = 3;
    DataClassificationSetting data_classification = 4;
    SemanticTypeSetting semantic_type = 5;
    AISetting ai = 6;
    EnvironmentSetting environment = 7;
    EmailSetting email = 8;
  }
}
```

- [ ] **Step 3: Add TestEmailSetting RPC to SettingService**

Add after the `UpdateSetting` RPC (line 48), before the closing brace:

```proto
  // Sends a test email using the provided config (without persisting).
  // Permissions required: bb.settings.set
  rpc TestEmailSetting(TestEmailSettingRequest) returns (TestEmailSettingResponse) {
    option (google.api.http) = {
      post: "/v1/{parent=workspaces/*}/settings/EMAIL:test"
      body: "*"
    };
    option (bytebase.v1.permission) = "bb.settings.set";
    option (bytebase.v1.auth_method) = IAM;
  }
```

- [ ] **Step 4: Add v1 EmailSetting, TestEmailSettingRequest, TestEmailSettingResponse messages**

Append at the end of `setting_service.proto` (after line 463):

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
    // INPUT_ONLY — never returned in GET responses.
    string password = 4 [(google.api.field_behavior) = INPUT_ONLY];
    Encryption encryption = 5;
    Authentication authentication = 6;

    enum Encryption {
      ENCRYPTION_UNSPECIFIED = 0;
      NONE = 1;
      STARTTLS = 2;
      SSL_TLS = 3;
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
  // Human-readable error if success=false.
  string error = 2;
}
```

- [ ] **Step 5: Format, lint, and generate**

Run:
```bash
cd proto && buf format -w . && buf lint && buf generate
```
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add proto/ backend/generated-go/ frontend/src/types/proto-es/
git commit -m "proto(v1): add EMAIL setting, EmailSetting message, and TestEmailSetting RPC"
```

---

### Task 3: Store layer — register EMAIL in getSettingMessage

**Files:**
- Modify: `backend/store/setting.go:24-43` (getSettingMessage switch)

- [ ] **Step 1: Add EMAIL case**

In `backend/store/setting.go`, add a case for `EMAIL` in the `getSettingMessage` function (after the `SettingName_ENVIRONMENT` case, around line 41):

```go
	case storepb.SettingName_ENVIRONMENT:
		return &storepb.EnvironmentSetting{}, nil
	case storepb.SettingName_EMAIL:
		return &storepb.EmailSetting{}, nil
```

- [ ] **Step 2: Build**

Run: `go build ./backend/store/...`
Expected: Success.

- [ ] **Step 3: Commit**

```bash
git add backend/store/setting.go
git commit -m "feat(store): register EMAIL setting name in getSettingMessage"
```

---

### Task 4: Converter — add EMAIL case to setting name converters and value converter

**Files:**
- Modify: `backend/api/v1/setting_service_converter.go:15-124` (convertToSettingMessage), `:140-165` (convertStoreSettingNameToV1), `:167-190` (convertV1SettingNameToStore), and append converter functions at EOF

- [ ] **Step 1: Add EMAIL case to convertToSettingMessage**

In `setting_service_converter.go`, add the EMAIL case before the `default` case (before line 121):

```go
	case storepb.SettingName_EMAIL:
		storeValue, ok := setting.Value.(*storepb.EmailSetting)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("invalid setting value type for %s", setting.Name))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_Email{
					Email: convertToEmailSetting(storeValue),
				},
			},
		}, nil
```

- [ ] **Step 2: Add EMAIL case to convertStoreSettingNameToV1**

In the `convertStoreSettingNameToV1` function (around line 158), add before `case storepb.SettingName_SYSTEM`:

```go
	case storepb.SettingName_EMAIL:
		return v1pb.Setting_EMAIL
```

- [ ] **Step 3: Add EMAIL case to convertV1SettingNameToStore**

In the `convertV1SettingNameToStore` function (around line 186), add before `default`:

```go
	case v1pb.Setting_EMAIL:
		return storepb.SettingName_EMAIL
```

- [ ] **Step 4: Add converter helper functions**

Append at the end of `setting_service_converter.go`:

```go
func convertEmailSetting(v1Setting *v1pb.EmailSetting) *storepb.EmailSetting {
	if v1Setting == nil {
		return nil
	}
	storeSetting := &storepb.EmailSetting{
		From:     v1Setting.From,
		FromName: v1Setting.FromName,
		Type:     storepb.EmailSetting_Type(v1Setting.Type),
	}
	if v1Smtp := v1Setting.GetSmtp(); v1Smtp != nil {
		storeSetting.Config = &storepb.EmailSetting_Smtp{
			Smtp: &storepb.EmailSetting_SMTPConfig{
				Host:           v1Smtp.Host,
				Port:           v1Smtp.Port,
				Username:       v1Smtp.Username,
				Password:       v1Smtp.Password,
				Encryption:     storepb.EmailSetting_SMTPConfig_Encryption(v1Smtp.Encryption),
				Authentication: storepb.EmailSetting_SMTPConfig_Authentication(v1Smtp.Authentication),
			},
		}
	}
	return storeSetting
}

func convertToEmailSetting(storeSetting *storepb.EmailSetting) *v1pb.EmailSetting {
	if storeSetting == nil {
		return nil
	}
	v1Setting := &v1pb.EmailSetting{
		From:     storeSetting.From,
		FromName: storeSetting.FromName,
		Type:     v1pb.EmailSetting_Type(storeSetting.Type),
	}
	if storeSmtp := storeSetting.GetSmtp(); storeSmtp != nil {
		v1Setting.Config = &v1pb.EmailSetting_Smtp{
			Smtp: &v1pb.EmailSetting_SMTPConfig{
				Host:       storeSmtp.Host,
				Port:       storeSmtp.Port,
				Username:   storeSmtp.Username,
				Password:   "", // INPUT_ONLY: never return password
				Encryption: v1pb.EmailSetting_SMTPConfig_Encryption(storeSmtp.Encryption),
				Authentication: v1pb.EmailSetting_SMTPConfig_Authentication(storeSmtp.Authentication),
			},
		}
	}
	return v1Setting
}
```

- [ ] **Step 5: Build**

Run: `go build ./backend/api/v1/...`
Expected: Success.

- [ ] **Step 6: Commit**

```bash
git add backend/api/v1/setting_service_converter.go
git commit -m "feat(api): add EMAIL setting converter functions"
```

---

### Task 5: Service layer — add EMAIL validation in UpdateSetting

**Files:**
- Modify: `backend/api/v1/setting_service.go:533-612` (UpdateSetting switch, add case before `default`)

- [ ] **Step 1: Add EMAIL case to UpdateSetting validation**

In `setting_service.go`, add a `case storepb.SettingName_EMAIL:` before the `default:` case (before line 611):

```go
	case storepb.SettingName_EMAIL:
		if request.Msg.UpdateMask == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask is required"))
		}
		payload := convertEmailSetting(request.Msg.Setting.Value.GetEmail())
		if payload == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email setting is required"))
		}

		oldEmailSetting := &storepb.EmailSetting{}
		if existing, err := s.store.GetSetting(ctx, workspaceID, storepb.SettingName_EMAIL); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get email setting: %v", err))
		} else if existing != nil {
			oldEmailSetting = proto.CloneOf(existing.Value.(*storepb.EmailSetting))
		}

		for _, path := range request.Msg.UpdateMask.Paths {
			switch path {
			case "value.email.from":
				oldEmailSetting.From = payload.From
			case "value.email.from_name":
				oldEmailSetting.FromName = payload.FromName
			case "value.email.type":
				oldEmailSetting.Type = payload.Type
			case "value.email.smtp":
				newSmtp := payload.GetSmtp()
				if newSmtp == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("smtp config is required when type is SMTP"))
				}
				// Preserve existing password if new password is empty.
				if newSmtp.Password == "" {
					if oldSmtp := oldEmailSetting.GetSmtp(); oldSmtp != nil {
						newSmtp.Password = oldSmtp.Password
					}
				}
				oldEmailSetting.Config = &storepb.EmailSetting_Smtp{Smtp: newSmtp}
			default:
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %v", path))
			}
		}

		// Validate the final state.
		if oldEmailSetting.Type == storepb.EmailSetting_SMTP {
			smtp := oldEmailSetting.GetSmtp()
			if smtp == nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("smtp config is required when type is SMTP"))
			}
			if smtp.Host == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("smtp host is required"))
			}
			if smtp.Port <= 0 {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("smtp port must be positive"))
			}
		}
		if oldEmailSetting.From == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("from address is required"))
		}

		storeSettingValue = oldEmailSetting
```

- [ ] **Step 2: Build and lint**

Run:
```bash
go build ./backend/api/v1/...
golangci-lint run --allow-parallel-runners ./backend/api/v1/...
```
Expected: Success, 0 lint issues.

- [ ] **Step 3: Commit**

```bash
git add backend/api/v1/setting_service.go
git commit -m "feat(api): add EMAIL validation in UpdateSetting"
```

---

### Task 6: Mail sender plugin — SMTP implementation

**Files:**
- Create: `backend/plugin/mail/mail.go`
- Create: `backend/plugin/mail/smtp.go`
- Create: `backend/plugin/mail/smtp_test.go`

- [ ] **Step 1: Create mail plugin interface**

Create `backend/plugin/mail/mail.go`:

```go
package mail

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Sender sends emails.
type Sender interface {
	Send(ctx context.Context, req *SendRequest) error
}

// SendRequest is the request to send an email.
type SendRequest struct {
	To       []string
	Subject  string
	TextBody string
	HTMLBody string
}

// NewSender creates a Sender from the stored email configuration.
func NewSender(cfg *storepb.EmailSetting) (Sender, error) {
	if cfg == nil {
		return nil, errors.Errorf("email setting is nil")
	}
	switch cfg.Type {
	case storepb.EmailSetting_SMTP:
		smtp := cfg.GetSmtp()
		if smtp == nil {
			return nil, errors.Errorf("smtp config is nil")
		}
		return newSMTPSender(cfg.From, cfg.FromName, smtp), nil
	default:
		return nil, errors.Errorf("unsupported email type: %v", cfg.Type)
	}
}
```

- [ ] **Step 2: Create SMTP sender**

Create `backend/plugin/mail/smtp.go`:

```go
package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

const (
	connectTimeout = 10 * time.Second
	sendTimeout    = 30 * time.Second
)

type smtpSender struct {
	from     string
	fromName string
	config   *storepb.EmailSetting_SMTPConfig
}

func newSMTPSender(from, fromName string, config *storepb.EmailSetting_SMTPConfig) *smtpSender {
	return &smtpSender{from: from, fromName: fromName, config: config}
}

func (s *smtpSender) Send(ctx context.Context, req *SendRequest) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	client, err := s.dial(addr)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to SMTP server %s", addr)
	}
	defer client.Close()

	if s.config.Encryption == storepb.EmailSetting_SMTPConfig_STARTTLS {
		tlsConfig := &tls.Config{ServerName: s.config.Host}
		if err := client.StartTLS(tlsConfig); err != nil {
			return errors.Wrap(err, "STARTTLS failed")
		}
	}

	if auth := s.auth(); auth != nil {
		if err := client.Auth(auth); err != nil {
			return errors.Wrap(err, "SMTP authentication failed")
		}
	}

	if err := client.Mail(s.from); err != nil {
		return errors.Wrap(err, "MAIL FROM failed")
	}
	for _, to := range req.To {
		if err := client.Rcpt(to); err != nil {
			return errors.Wrapf(err, "RCPT TO %s failed", to)
		}
	}

	w, err := client.Data()
	if err != nil {
		return errors.Wrap(err, "DATA command failed")
	}
	msg := s.buildMessage(req)
	if _, err := w.Write([]byte(msg)); err != nil {
		return errors.Wrap(err, "failed to write message")
	}
	if err := w.Close(); err != nil {
		return errors.Wrap(err, "failed to close data writer")
	}

	return client.Quit()
}

func (s *smtpSender) dial(addr string) (*smtp.Client, error) {
	switch s.config.Encryption {
	case storepb.EmailSetting_SMTPConfig_SSL_TLS:
		tlsConfig := &tls.Config{ServerName: s.config.Host}
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: connectTimeout}, "tcp", addr, tlsConfig)
		if err != nil {
			return nil, err
		}
		return smtp.NewClient(conn, s.config.Host)
	default:
		conn, err := net.DialTimeout("tcp", addr, connectTimeout)
		if err != nil {
			return nil, err
		}
		return smtp.NewClient(conn, s.config.Host)
	}
}

func (s *smtpSender) auth() smtp.Auth {
	if s.config.Authentication == storepb.EmailSetting_SMTPConfig_NONE {
		return nil
	}
	if s.config.Username == "" && s.config.Password == "" {
		return nil
	}
	switch s.config.Authentication {
	case storepb.EmailSetting_SMTPConfig_CRAM_MD5:
		return smtp.CRAMMD5Auth(s.config.Username, s.config.Password)
	default:
		// PLAIN, LOGIN, and UNSPECIFIED all use PlainAuth.
		return smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}
}

func (s *smtpSender) buildMessage(req *SendRequest) string {
	var b strings.Builder

	fromHeader := s.from
	if s.fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("utf-8", s.fromName), s.from)
	}
	fmt.Fprintf(&b, "From: %s\r\n", fromHeader)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(req.To, ", "))
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", req.Subject))
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")

	if req.HTMLBody != "" {
		boundary := "bytebase-email-boundary"
		fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary)
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		fmt.Fprintf(&b, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
		fmt.Fprintf(&b, "%s\r\n", req.TextBody)
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		fmt.Fprintf(&b, "Content-Type: text/html; charset=utf-8\r\n\r\n")
		fmt.Fprintf(&b, "%s\r\n", req.HTMLBody)
		fmt.Fprintf(&b, "--%s--\r\n", boundary)
	} else {
		fmt.Fprintf(&b, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
		fmt.Fprintf(&b, "%s\r\n", req.TextBody)
	}

	return b.String()
}
```

- [ ] **Step 3: Create SMTP unit test**

Create `backend/plugin/mail/smtp_test.go`:

```go
package mail

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestBuildMessage_PlainText(t *testing.T) {
	s := &smtpSender{from: "noreply@example.com", fromName: "Bytebase"}
	msg := s.buildMessage(&SendRequest{
		To:       []string{"user@example.com"},
		Subject:  "Test Subject",
		TextBody: "Hello, World!",
	})
	assert.Contains(t, msg, "From: =?utf-8?q?Bytebase?= <noreply@example.com>")
	assert.Contains(t, msg, "To: user@example.com")
	assert.Contains(t, msg, "Subject: =?utf-8?q?Test_Subject?=")
	assert.Contains(t, msg, "Content-Type: text/plain; charset=utf-8")
	assert.Contains(t, msg, "Hello, World!")
	assert.NotContains(t, msg, "multipart")
}

func TestBuildMessage_HTML(t *testing.T) {
	s := &smtpSender{from: "noreply@example.com", fromName: ""}
	msg := s.buildMessage(&SendRequest{
		To:       []string{"a@b.com", "c@d.com"},
		Subject:  "HTML Test",
		TextBody: "plain text",
		HTMLBody: "<p>html body</p>",
	})
	assert.Contains(t, msg, "From: noreply@example.com")
	assert.Contains(t, msg, "To: a@b.com, c@d.com")
	assert.Contains(t, msg, "multipart/alternative")
	assert.Contains(t, msg, "plain text")
	assert.Contains(t, msg, "<p>html body</p>")
}

func TestNewSender_SMTP(t *testing.T) {
	sender, err := NewSender(&storepb.EmailSetting{
		From: "test@example.com",
		Type: storepb.EmailSetting_SMTP,
		Config: &storepb.EmailSetting_Smtp{
			Smtp: &storepb.EmailSetting_SMTPConfig{
				Host:       "localhost",
				Port:       25,
				Encryption: storepb.EmailSetting_SMTPConfig_NONE,
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, sender)
}

func TestNewSender_NilConfig(t *testing.T) {
	_, err := NewSender(nil)
	assert.Error(t, err)
}

func TestNewSender_UnsupportedType(t *testing.T) {
	_, err := NewSender(&storepb.EmailSetting{
		From: "test@example.com",
		Type: storepb.EmailSetting_TYPE_UNSPECIFIED,
	})
	assert.Error(t, err)
}

func TestSMTPSend_ConnectionRefused(t *testing.T) {
	// Find a port that's guaranteed not listening.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	sender, err := NewSender(&storepb.EmailSetting{
		From: "test@example.com",
		Type: storepb.EmailSetting_SMTP,
		Config: &storepb.EmailSetting_Smtp{
			Smtp: &storepb.EmailSetting_SMTPConfig{
				Host:       "127.0.0.1",
				Port:       int32(port),
				Encryption: storepb.EmailSetting_SMTPConfig_NONE,
			},
		},
	})
	require.NoError(t, err)

	err = sender.Send(context.Background(), &SendRequest{
		To:       []string{"user@example.com"},
		Subject:  "Test",
		TextBody: "body",
	})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "connect") || strings.Contains(err.Error(), "refused"))
}

func TestAuth_None(t *testing.T) {
	s := &smtpSender{config: &storepb.EmailSetting_SMTPConfig{
		Authentication: storepb.EmailSetting_SMTPConfig_NONE,
	}}
	assert.Nil(t, s.auth())
}

func TestAuth_Plain(t *testing.T) {
	s := &smtpSender{config: &storepb.EmailSetting_SMTPConfig{
		Authentication: storepb.EmailSetting_SMTPConfig_PLAIN,
		Username:       "user",
		Password:       "pass",
		Host:           "smtp.example.com",
	}}
	assert.NotNil(t, s.auth())
}
```

- [ ] **Step 4: Run tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/mail`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add backend/plugin/mail/
git commit -m "feat(plugin): add mail sender plugin with SMTP implementation"
```

---

### Task 7: TestEmailSetting RPC implementation

**Files:**
- Modify: `backend/api/v1/setting_service.go` (add TestEmailSetting method + import `mail` plugin)

- [ ] **Step 1: Add import for mail plugin**

In `setting_service.go`, add to the imports (around line 26):

```go
	"github.com/bytebase/bytebase/backend/plugin/mail"
```

- [ ] **Step 2: Add TestEmailSetting method**

Add after the `UpdateSetting` method (around line 680):

```go
// TestEmailSetting sends a test email using the provided config.
func (s *SettingService) TestEmailSetting(ctx context.Context, req *connect.Request[v1pb.TestEmailSettingRequest]) (*connect.Response[v1pb.TestEmailSettingResponse], error) {
	emailSetting := convertEmailSetting(req.Msg.EmailSetting)
	if emailSetting == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email_setting is required"))
	}

	// Substitute stored password if not provided.
	if smtp := emailSetting.GetSmtp(); smtp != nil && smtp.Password == "" {
		workspaceID := common.GetWorkspaceIDFromContext(ctx)
		if existing, err := s.store.GetSetting(ctx, workspaceID, storepb.SettingName_EMAIL); err == nil && existing != nil {
			if oldEmail, ok := existing.Value.(*storepb.EmailSetting); ok {
				if oldSmtp := oldEmail.GetSmtp(); oldSmtp != nil {
					smtp.Password = oldSmtp.Password
				}
			}
		}
	}

	sender, err := mail.NewSender(emailSetting)
	if err != nil {
		return connect.NewResponse(&v1pb.TestEmailSettingResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}

	err = sender.Send(ctx, &mail.SendRequest{
		To:       []string{req.Msg.To},
		Subject:  "Bytebase email config test",
		TextBody: "This is a test email from Bytebase to verify your email configuration.",
	})
	if err != nil {
		return connect.NewResponse(&v1pb.TestEmailSettingResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}

	return connect.NewResponse(&v1pb.TestEmailSettingResponse{
		Success: true,
	}), nil
}
```

- [ ] **Step 3: Build and lint**

Run:
```bash
go build ./backend/api/v1/...
golangci-lint run --allow-parallel-runners ./backend/api/v1/...
```
Expected: Success, 0 lint issues.

- [ ] **Step 4: Commit**

```bash
git add backend/api/v1/setting_service.go
git commit -m "feat(api): implement TestEmailSetting RPC"
```

---

### Task 8: Workspace creation — inject EMAIL_CONFIG env var

**Files:**
- Modify: `backend/api/v1/auth_service.go:1393-1410` (getAdditionalWorkspaceSettings)

- [ ] **Step 1: Add os and protojson imports**

Ensure these imports are present in `auth_service.go` (they may already be):

```go
	"os"
```

and

```go
	"google.golang.org/protobuf/encoding/protojson"
```

- [ ] **Step 2: Add EMAIL_CONFIG injection**

In `getAdditionalWorkspaceSettings()`, add after the Gemini block (after line 1408):

```go
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
```

- [ ] **Step 3: Build and lint**

Run:
```bash
go build ./backend/api/v1/...
golangci-lint run --allow-parallel-runners ./backend/api/v1/...
```
Expected: Success, 0 lint issues.

- [ ] **Step 4: Commit**

```bash
git add backend/api/v1/auth_service.go
git commit -m "feat(auth): inject EMAIL_CONFIG env var into workspace settings on creation"
```

---

### Task 9: Frontend fix + type-check + full backend build

**Files:** None new — validation pass.

- [ ] **Step 1: Full backend build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Success.

- [ ] **Step 2: Backend lint (full)**

Run: `golangci-lint run --allow-parallel-runners`
Expected: 0 issues (run repeatedly until clean).

- [ ] **Step 3: Frontend fix**

Run: `pnpm --dir frontend fix`
Expected: No errors.

- [ ] **Step 4: Frontend type-check**

Run: `pnpm --dir frontend type-check`
Expected: Success.

- [ ] **Step 5: Backend tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/mail`
Expected: All tests pass.

- [ ] **Step 6: Commit any auto-fixed files**

If `pnpm fix` or `golangci-lint --fix` modified files:
```bash
git add -A
git commit -m "chore: fix lint and formatting"
```
