# Email + 6-Digit Code Signin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add passwordless signin/signup via email + 6-digit code, and refactor password reset onto the same mechanism.

**Architecture:** A new `email_verification_code` table (PK = `email`, `purpose`) backs both flows. Login gains an `email_code` field in `LoginRequest`, threading through the existing workspace/MFA/token pipeline. Password reset switches from JWT to DB-backed code with cooldown + attempt limits, making it HA-safe.

**Tech Stack:** Go stdlib (`crypto/rand`, `crypto/sha256`, `crypto/subtle`), Connect-RPC, protobuf, PostgreSQL, Vue 3.

**Spec:** `docs/superpowers/specs/2026-04-14-email-code-signin-design.md`

---

### Task 1: Store proto — EmailVerificationCodePurpose enum

**Files:**
- Create: `proto/store/store/email_verification_code.proto`

- [ ] **Step 1: Create the proto file**

```proto
syntax = "proto3";

package bytebase.store;

option go_package = "generated-go/store";

// EmailVerificationCodePurpose distinguishes login codes from password reset codes.
// Stored as the enum name string in email_verification_code.purpose column.
enum EmailVerificationCodePurpose {
  EMAIL_VERIFICATION_CODE_PURPOSE_UNSPECIFIED = 0;
  LOGIN = 1;
  PASSWORD_RESET = 2;
}
```

- [ ] **Step 2: Format and lint**

```bash
cd proto && buf format -w . && buf lint
```

Expected: no errors.

---

### Task 2: Store proto — add allow_email_code_signin to WorkspaceProfileSetting

**Files:**
- Modify: `proto/store/store/setting.proto:37-147` (WorkspaceProfileSetting message)

- [ ] **Step 1: Add field before closing `}` at line 147**

Find the last field (around `query_timeout` at line 146) and the closing brace at line 147. Insert a new field:

```proto
  // Allow signin/signup using email + a 6-digit one-time verification code.
  // Requires the EMAIL setting to be configured on the workspace.
  bool allow_email_code_signin = 22;
```

(Use the next available field number; check the highest existing number in the message and use next.)

- [ ] **Step 2: Format and lint**

```bash
cd proto && buf format -w . && buf lint
```

Expected: no errors.

---

### Task 3: V1 proto — mirror allow_email_code_signin to setting_service + Restriction

**Files:**
- Modify: `proto/v1/v1/setting_service.proto:185-267` (WorkspaceProfileSetting)
- Modify: `proto/v1/v1/actuator_service.proto:65-74` (Restriction)

- [ ] **Step 1: Add field to v1 WorkspaceProfileSetting (before closing `}` at line 267)**

```proto
  // Allow signin/signup using email + a 6-digit one-time verification code.
  // Requires the EMAIL setting to be configured on the workspace.
  bool allow_email_code_signin = 22;
```

(Use the next available field number matching the store proto.)

- [ ] **Step 2: Add field to Restriction message**

Replace the Restriction message (lines 65-74) with:

```proto
message Restriction {
  // Whether self-service user signup is disabled.
  bool disallow_signup = 1 [(google.api.field_behavior) = OUTPUT_ONLY];

  // Whether password-based signin is disabled (except for workspace admins).
  bool disallow_password_signin = 2 [(google.api.field_behavior) = OUTPUT_ONLY];

  // Password complexity and restriction requirements.
  WorkspaceProfileSetting.PasswordRestriction password_restriction = 3 [(google.api.field_behavior) = OUTPUT_ONLY];

  // Whether email + 6-digit code signin is enabled for this workspace.
  bool allow_email_code_signin = 4 [(google.api.field_behavior) = OUTPUT_ONLY];
}
```

- [ ] **Step 3: Format and lint**

```bash
cd proto && buf format -w . && buf lint
```

Expected: no errors.

---

### Task 4: V1 proto — add email_code to LoginRequest and SendEmailLoginCode RPC; switch ResetPassword to code

**Files:**
- Modify: `proto/v1/v1/auth_service.proto`

- [ ] **Step 1: Add `email_code` field to LoginRequest**

Find `LoginRequest` (lines 103-129). Add a new field after `mfa_temp_token` (field 8):

```proto
  // 6-digit code from email for passwordless login/signup.
  // Pairs with `email`. Mutually exclusive with `password` and `idp_name`.
  optional string email_code = 9;
```

- [ ] **Step 2: Add SendEmailLoginCode RPC to AuthService**

Inside the `service AuthService { ... }` block, add before the closing `}`:

```proto
  // Sends a 6-digit verification code to the email for login/signup.
  // Always returns success (no email enumeration). Enforces 60-sec resend cooldown.
  // Permissions required: None
  rpc SendEmailLoginCode(SendEmailLoginCodeRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      post: "/v1/auth:sendEmailLoginCode"
      body: "*"
    };
    option (bytebase.v1.allow_without_credential) = true;
    option (bytebase.v1.audit) = true;
  }
```

- [ ] **Step 3: Add SendEmailLoginCodeRequest message**

Append at the end of the file (before the final `}` if any, or at bottom):

```proto
message SendEmailLoginCodeRequest {
  // The email address to send the code to.
  string email = 1;
}
```

- [ ] **Step 4: Replace ResetPasswordRequest message**

Find the existing `ResetPasswordRequest` message (added in earlier work). Replace:

```proto
message ResetPasswordRequest {
  // The email address of the account.
  string email = 1;
  // The 6-digit code from the reset email.
  string code = 2;
  // The new password to set.
  string new_password = 3;
}
```

- [ ] **Step 5: Format, lint, generate**

```bash
cd proto && buf format -w . && buf lint && buf generate
```

Expected: no errors. All generated files (backend/generated-go/, frontend/src/types/proto-es/) are updated.

---

### Task 5: Database migration — email_verification_code table

**Files:**
- Create: `backend/migrator/migration/3.18/0000_add_email_verification_code.sql`
- Modify: `backend/migrator/migration/LATEST.sql` (insert new CREATE TABLE)
- Modify: `backend/migrator/migrator_test.go` (`TestLatestVersion`)

- [ ] **Step 1: Create the migration file**

File: `backend/migrator/migration/3.18/0000_add_email_verification_code.sql`

```sql
CREATE TABLE email_verification_code (
    email         text NOT NULL,
    -- Stored as EmailVerificationCodePurpose enum name (proto/store/store/email_verification_code.proto)
    purpose       text NOT NULL,
    code_hash     text NOT NULL,
    attempts      int  NOT NULL DEFAULT 0,
    expires_at    timestamptz NOT NULL,
    last_sent_at  timestamptz NOT NULL,
    PRIMARY KEY (email, purpose)
);

CREATE INDEX idx_email_verification_code_expires_at ON email_verification_code (expires_at);
```

- [ ] **Step 2: Mirror in LATEST.sql**

Open `backend/migrator/migration/LATEST.sql`. Find the `web_refresh_token` CREATE TABLE (around line 631-638). Immediately after its trailing `CREATE INDEX` statements, append:

```sql

CREATE TABLE email_verification_code (
    email         text NOT NULL,
    -- Stored as EmailVerificationCodePurpose enum name (proto/store/store/email_verification_code.proto)
    purpose       text NOT NULL,
    code_hash     text NOT NULL,
    attempts      int  NOT NULL DEFAULT 0,
    expires_at    timestamptz NOT NULL,
    last_sent_at  timestamptz NOT NULL,
    PRIMARY KEY (email, purpose)
);

CREATE INDEX idx_email_verification_code_expires_at ON email_verification_code (expires_at);
```

- [ ] **Step 3: Bump TestLatestVersion**

Open `backend/migrator/migrator_test.go`. Find `TestLatestVersion`. Update the expected version to `3.18` (or match the new directory you created).

- [ ] **Step 4: Run migrator test**

```bash
go test -v -count=1 -run TestLatestVersion github.com/bytebase/bytebase/backend/migrator
```

Expected: PASS.

---

### Task 6: Store layer — email_verification_code.go

**Files:**
- Create: `backend/store/email_verification_code.go`
- Create: `backend/store/email_verification_code_test.go`

- [ ] **Step 1: Write the failing test**

File: `backend/store/email_verification_code_test.go`

```go
package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestEmailVerificationCode_UpsertGet(t *testing.T) {
	ctx := t.Context()
	s := setupStoreForTest(t)

	code := &EmailVerificationCodeMessage{
		Email:      "test@example.com",
		Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
		CodeHash:   "hash1",
		ExpiresAt:  time.Now().Add(10 * time.Minute),
		LastSentAt: time.Now(),
	}
	require.NoError(t, s.UpsertEmailVerificationCode(ctx, code))

	got, err := s.GetEmailVerificationCode(ctx, code.Email, code.Purpose)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, code.Email, got.Email)
	assert.Equal(t, code.Purpose, got.Purpose)
	assert.Equal(t, code.CodeHash, got.CodeHash)
	assert.Equal(t, 0, got.Attempts)
}

func TestEmailVerificationCode_UpsertReplacesAttempts(t *testing.T) {
	ctx := t.Context()
	s := setupStoreForTest(t)

	first := &EmailVerificationCodeMessage{
		Email:      "test2@example.com",
		Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
		CodeHash:   "hash1",
		ExpiresAt:  time.Now().Add(10 * time.Minute),
		LastSentAt: time.Now(),
	}
	require.NoError(t, s.UpsertEmailVerificationCode(ctx, first))
	require.NoError(t, s.IncrementEmailVerificationCodeAttempts(ctx, first.Email, first.Purpose))
	require.NoError(t, s.IncrementEmailVerificationCodeAttempts(ctx, first.Email, first.Purpose))

	second := &EmailVerificationCodeMessage{
		Email:      first.Email,
		Purpose:    first.Purpose,
		CodeHash:   "hash2",
		ExpiresAt:  time.Now().Add(10 * time.Minute),
		LastSentAt: time.Now(),
	}
	require.NoError(t, s.UpsertEmailVerificationCode(ctx, second))

	got, err := s.GetEmailVerificationCode(ctx, second.Email, second.Purpose)
	require.NoError(t, err)
	assert.Equal(t, "hash2", got.CodeHash)
	assert.Equal(t, 0, got.Attempts, "upsert should reset attempts")
}

func TestEmailVerificationCode_PurposesCoexist(t *testing.T) {
	ctx := t.Context()
	s := setupStoreForTest(t)

	require.NoError(t, s.UpsertEmailVerificationCode(ctx, &EmailVerificationCodeMessage{
		Email: "test3@example.com", Purpose: storepb.EmailVerificationCodePurpose_LOGIN,
		CodeHash: "login-hash", ExpiresAt: time.Now().Add(time.Minute), LastSentAt: time.Now(),
	}))
	require.NoError(t, s.UpsertEmailVerificationCode(ctx, &EmailVerificationCodeMessage{
		Email: "test3@example.com", Purpose: storepb.EmailVerificationCodePurpose_PASSWORD_RESET,
		CodeHash: "reset-hash", ExpiresAt: time.Now().Add(time.Minute), LastSentAt: time.Now(),
	}))

	loginRow, _ := s.GetEmailVerificationCode(ctx, "test3@example.com", storepb.EmailVerificationCodePurpose_LOGIN)
	resetRow, _ := s.GetEmailVerificationCode(ctx, "test3@example.com", storepb.EmailVerificationCodePurpose_PASSWORD_RESET)
	require.NotNil(t, loginRow)
	require.NotNil(t, resetRow)
	assert.Equal(t, "login-hash", loginRow.CodeHash)
	assert.Equal(t, "reset-hash", resetRow.CodeHash)
}

func TestEmailVerificationCode_Delete(t *testing.T) {
	ctx := t.Context()
	s := setupStoreForTest(t)

	require.NoError(t, s.UpsertEmailVerificationCode(ctx, &EmailVerificationCodeMessage{
		Email: "test4@example.com", Purpose: storepb.EmailVerificationCodePurpose_LOGIN,
		CodeHash: "h", ExpiresAt: time.Now().Add(time.Minute), LastSentAt: time.Now(),
	}))
	require.NoError(t, s.DeleteEmailVerificationCode(ctx, "test4@example.com", storepb.EmailVerificationCodePurpose_LOGIN))

	got, err := s.GetEmailVerificationCode(ctx, "test4@example.com", storepb.EmailVerificationCodePurpose_LOGIN)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestEmailVerificationCode_DeleteExpired(t *testing.T) {
	ctx := t.Context()
	s := setupStoreForTest(t)

	// Expired row.
	require.NoError(t, s.UpsertEmailVerificationCode(ctx, &EmailVerificationCodeMessage{
		Email: "expired@example.com", Purpose: storepb.EmailVerificationCodePurpose_LOGIN,
		CodeHash: "h", ExpiresAt: time.Now().Add(-time.Hour), LastSentAt: time.Now(),
	}))
	// Fresh row.
	require.NoError(t, s.UpsertEmailVerificationCode(ctx, &EmailVerificationCodeMessage{
		Email: "fresh@example.com", Purpose: storepb.EmailVerificationCodePurpose_LOGIN,
		CodeHash: "h", ExpiresAt: time.Now().Add(time.Hour), LastSentAt: time.Now(),
	}))

	n, err := s.DeleteExpiredEmailVerificationCodes(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	expired, _ := s.GetEmailVerificationCode(ctx, "expired@example.com", storepb.EmailVerificationCodePurpose_LOGIN)
	fresh, _ := s.GetEmailVerificationCode(ctx, "fresh@example.com", storepb.EmailVerificationCodePurpose_LOGIN)
	assert.Nil(t, expired)
	assert.NotNil(t, fresh)
}
```

Note: `setupStoreForTest(t)` — check existing store tests (e.g. `backend/store/web_refresh_token_test.go` if it exists, or `backend/store/setting_test.go`) for the project-standard test harness. Use the same helper. If no harness exists, skip the DB-backed tests and rely on integration tests.

- [ ] **Step 2: Run test to verify failure (undefined symbols)**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/store -run TestEmailVerificationCode
```

Expected: build error / undefined.

- [ ] **Step 3: Create the store file**

File: `backend/store/email_verification_code.go`

```go
package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type EmailVerificationCodeMessage struct {
	Email      string
	Purpose    storepb.EmailVerificationCodePurpose
	CodeHash   string
	Attempts   int
	ExpiresAt  time.Time
	LastSentAt time.Time
}

// UpsertEmailVerificationCode inserts or replaces the row for (email, purpose).
// Upserting always resets attempts to 0, because the row represents a single code send.
func (s *Store) UpsertEmailVerificationCode(ctx context.Context, msg *EmailVerificationCodeMessage) error {
	q := qb.Q().Space(`
		INSERT INTO email_verification_code (email, purpose, code_hash, attempts, expires_at, last_sent_at)
		VALUES (?, ?, ?, 0, ?, ?)
		ON CONFLICT (email, purpose) DO UPDATE SET
			code_hash = EXCLUDED.code_hash,
			attempts = 0,
			expires_at = EXCLUDED.expires_at,
			last_sent_at = EXCLUDED.last_sent_at
	`, msg.Email, msg.Purpose.String(), msg.CodeHash, msg.ExpiresAt, msg.LastSentAt)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to upsert email verification code")
	}
	return nil
}

// GetEmailVerificationCode returns (nil, nil) if no row exists.
func (s *Store) GetEmailVerificationCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose) (*EmailVerificationCodeMessage, error) {
	q := qb.Q().Space(`
		SELECT email, purpose, code_hash, attempts, expires_at, last_sent_at
		FROM email_verification_code
		WHERE email = ? AND purpose = ?
	`, email, purpose.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	msg := &EmailVerificationCodeMessage{}
	var purposeStr string
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&msg.Email, &purposeStr, &msg.CodeHash, &msg.Attempts, &msg.ExpiresAt, &msg.LastSentAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get email verification code")
	}
	msg.Purpose = storepb.EmailVerificationCodePurpose(storepb.EmailVerificationCodePurpose_value[purposeStr])
	return msg, nil
}

// IncrementEmailVerificationCodeAttempts atomically +1 to the attempts column.
func (s *Store) IncrementEmailVerificationCodeAttempts(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose) error {
	q := qb.Q().Space(`
		UPDATE email_verification_code
		SET attempts = attempts + 1
		WHERE email = ? AND purpose = ?
	`, email, purpose.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to increment email verification code attempts")
	}
	return nil
}

// DeleteEmailVerificationCode removes the row for (email, purpose).
func (s *Store) DeleteEmailVerificationCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose) error {
	q := qb.Q().Space(`
		DELETE FROM email_verification_code
		WHERE email = ? AND purpose = ?
	`, email, purpose.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to delete email verification code")
	}
	return nil
}

// DeleteExpiredEmailVerificationCodes deletes all rows where expires_at < NOW().
// Returns the number of rows deleted.
func (s *Store) DeleteExpiredEmailVerificationCodes(ctx context.Context) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM email_verification_code
		WHERE expires_at < NOW()
	`)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, err
	}
	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete expired email verification codes")
	}
	return result.RowsAffected()
}
```

- [ ] **Step 4: Build**

```bash
go build ./backend/store/...
```

Expected: clean build.

- [ ] **Step 5: Run store tests**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/store -run TestEmailVerificationCode
```

Expected: all tests PASS. If the test harness `setupStoreForTest` doesn't exist, skip the test file and let integration tests (Task 14) cover this.

---

### Task 7: Remove legacy password-reset JWT code

**Files:**
- Modify: `backend/api/auth/tokens.go`
- Modify: `backend/api/auth/auth.go` (if it references the removed functions — unlikely, but check)

- [ ] **Step 1: Delete functions from `tokens.go`**

Remove these items:

1. The constant `PasswordResetTokenAudience` (around line 42).
2. The constant `passwordResetTokenDuration` (15-min duration).
3. Function `GeneratePasswordResetToken(userEmail, secret string) (string, error)` (lines ~59-63).
4. Function `GetEmailFromPasswordResetToken(token, secret string) (string, error)` (lines ~65-84).

- [ ] **Step 2: Build to surface any remaining references**

```bash
go build ./backend/... 2>&1 | head
```

Expected: errors pointing to `auth_service.go` (we'll fix those in Task 9/10). Other errors mean something else uses these — update accordingly.

---

### Task 8: Extract workspace-provisioning helper from Signup

**Files:**
- Modify: `backend/api/v1/auth_service.go`

- [ ] **Step 1: Locate the Signup workspace-provisioning block**

Open `auth_service.go`, find the `Signup` method (starts around line 182). Inside it, locate the block that:
1. Looks up existing workspaces for the email (via `ListWorkspacesByEmail`).
2. If none: creates a new workspace with `getAdditionalWorkspaceSettings()`.
3. If some: joins the first one.
4. Ends with the new user's workspace ID known.

- [ ] **Step 2: Extract to a helper**

Define a new method on `*AuthService`:

```go
// provisionWorkspaceForNewUser returns a workspace ID for the freshly-created user.
// If the email was pre-invited to existing workspaces (via IAM), returns the first one.
// Otherwise creates a new workspace (SaaS: per-user; self-hosted: joins the singleton).
// Called by both the Signup RPC and the email-code signup branch of Login.
func (s *AuthService) provisionWorkspaceForNewUser(ctx context.Context, email string) (string, error) {
	// MOVE the existing Signup block's body here.
	// Return the resolved workspace ID.
}
```

Replace the original code inside `Signup` with a call to this helper. Preserve exact semantics — this is a refactor, no behavior change.

- [ ] **Step 3: Build + ensure no regressions**

```bash
go build ./backend/... && go test -count=1 github.com/bytebase/bytebase/backend/api/v1
```

Expected: clean build; existing auth tests pass.

---

### Task 9: SendEmailLoginCode RPC + shared email-sending helper

**Files:**
- Modify: `backend/api/v1/auth_service.go`

- [ ] **Step 1: Add constants**

Near the top of `auth_service.go` (where other auth constants live):

```go
const (
	emailCodeLength         = 6
	emailCodeExpiry         = 10 * time.Minute
	emailCodeMaxAttempts    = 5
	emailCodeResendCooldown = 60 * time.Second
)
```

The 60-sec cooldown caps sends at ~60/hour/email, which is the only rate limit. See the "Rate-limiting design note" in Step 3.

- [ ] **Step 2: Add the code-generation helper**

Inside `auth_service.go`:

```go
// generateEmailCode returns a cryptographically-random 6-digit numeric code as a string.
func generateEmailCode() (string, error) {
	const digits = "0123456789"
	bytes := make([]byte, emailCodeLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i := range bytes {
		bytes[i] = digits[int(bytes[i])%len(digits)]
	}
	return string(bytes), nil
}

// hashEmailCode returns HMAC-SHA256(code) hex-encoded, keyed with the server's auth secret.
// HMAC with a server-side secret (vs. bare SHA-256) prevents offline brute force of the
// 10^6-size code space if the DB is ever compromised — the attacker would also need the
// auth secret to verify candidate codes.
func (s *AuthService) hashEmailCode(code string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}
```

Ensure imports include `crypto/hmac`, `crypto/rand`, `crypto/sha256`, `encoding/hex`.

**Note:** `hashEmailCode` is a method on `*AuthService` (needs `s.secret`). All call sites in this plan use `s.hashEmailCode(code)` — update references below if you see a package-level call.

- [ ] **Step 3: Add SendEmailLoginCode RPC**

```go
// SendEmailLoginCode sends a 6-digit verification code. Always returns success
// (no email enumeration). Rate limit: 60-sec resend cooldown enforced via DB
// last_sent_at inside sendEmailVerificationCode → effective cap ≈ 60 sends/hour/email.
func (s *AuthService) SendEmailLoginCode(ctx context.Context, req *connect.Request[v1pb.SendEmailLoginCodeRequest]) (*connect.Response[emptypb.Empty], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}

	// Fire-and-forget: cooldown check + send. Always return success to caller.
	go s.sendEmailVerificationCode(
		context.WithoutCancel(ctx),
		email,
		storepb.EmailVerificationCodePurpose_LOGIN,
		"[Bytebase] Your sign-in code",
		"Hi,\n\nYour sign-in code is: %s\n\nThis code expires in 10 minutes. If you didn't request this, you can safely ignore this email.\n\n— Bytebase",
	)

	return connect.NewResponse(&emptypb.Empty{}), nil
}
```

**Rate-limiting design note:** The spec initially proposed a separate 10/hour cap via audit-log counting, but `SendEmailLoginCode` is unauthenticated and has no workspace context — the audit middleware skips writing logs in that case (`backend/api/v1/audit.go:217-224`), so the count query would always return 0 and the cap would be ineffective. The 60-second `last_sent_at` cooldown inside `sendEmailVerificationCode` (enforced on the `(email, purpose)` row) already provides rate limiting: maximum ~60 sends per hour per email. This is strict enough without adding complexity. Remove `emailCodeSendMaxPerHour` from the constants block in Step 1.

- [ ] **Step 4: Add the shared `sendEmailVerificationCode` helper**

```go
// sendEmailVerificationCode generates a code, stores its hash, and emails the plain code.
// Errors are logged, never returned (avoids email enumeration).
// `bodyFmt` must contain one %s for the 6-digit code.
func (s *AuthService) sendEmailVerificationCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose, subject, bodyFmt string) {
	// For password reset, only send to existing principals — no upsert, no email for unknown addresses.
	// Prevents arbitrary email spam and matches the legacy sendPasswordResetEmail behavior.
	// LOGIN purpose intentionally skips this check because it also covers signup.
	if purpose == storepb.EmailVerificationCodePurpose_PASSWORD_RESET {
		account, err := s.store.GetAccountByEmail(ctx, email)
		if err != nil {
			slog.Warn("failed to look up account for password reset", log.BBError(err))
			return
		}
		if account == nil || account.Type != storepb.PrincipalType_END_USER {
			return // silent: account doesn't exist
		}
	}

	// Cooldown check.
	existing, err := s.store.GetEmailVerificationCode(ctx, email, purpose)
	if err != nil {
		slog.Warn("failed to check existing email verification code", log.BBError(err))
		return
	}
	now := time.Now()
	if existing != nil && now.Sub(existing.LastSentAt) < emailCodeResendCooldown {
		return // silent: cooldown not expired
	}

	// Generate code + hash.
	code, err := generateEmailCode()
	if err != nil {
		slog.Warn("failed to generate email code", log.BBError(err))
		return
	}

	if err := s.store.UpsertEmailVerificationCode(ctx, &store.EmailVerificationCodeMessage{
		Email:      email,
		Purpose:    purpose,
		CodeHash:   s.hashEmailCode(code),
		ExpiresAt:  now.Add(emailCodeExpiry),
		LastSentAt: now,
	}); err != nil {
		slog.Warn("failed to upsert email verification code", log.BBError(err))
		return
	}

	// Resolve workspace for EMAIL setting. Fall back to env-var EMAIL_CONFIG if no workspace yet.
	var emailSetting *storepb.EmailSetting
	workspaces, _ := s.store.ListWorkspacesByEmail(ctx, &store.FindWorkspaceMessage{Email: email, IncludeAllUser: true})
	if len(workspaces) > 0 {
		emailSettingMsg, err := s.store.GetSetting(ctx, workspaces[0].ResourceID, storepb.SettingName_EMAIL)
		if err == nil && emailSettingMsg != nil {
			if es, ok := emailSettingMsg.Value.(*storepb.EmailSetting); ok {
				emailSetting = es
			}
		}
	}
	if emailSetting == nil {
		// Fallback to EMAIL_CONFIG env var for brand-new signups (workspace doesn't exist yet).
		if raw := os.Getenv("EMAIL_CONFIG"); raw != "" {
			emailSetting = &storepb.EmailSetting{}
			if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(raw), emailSetting); err != nil {
				slog.Warn("failed to parse EMAIL_CONFIG", log.BBError(err))
				return
			}
		}
	}
	if emailSetting == nil {
		slog.Warn("no email setting configured; cannot send verification code", slog.String("email", email))
		return
	}

	sender, err := mailer.NewSender(emailSetting)
	if err != nil {
		slog.Warn("failed to create mail sender", log.BBError(err))
		return
	}

	body := fmt.Sprintf(bodyFmt, code)
	if err := sender.Send(ctx, &mailer.SendRequest{
		To:       []string{email},
		Subject:  subject,
		TextBody: body,
	}); err != nil {
		slog.Warn("failed to send email verification code", slog.String("to", email), log.BBError(err))
	}
}
```

- [ ] **Step 5: Build**

```bash
go build ./backend/... 2>&1 | tail
```

Expected: clean (existing password-reset code may still reference things — we fix that in Task 11).

---

### Task 10: Login email_code branch + verifyEmailCode helper

**Files:**
- Modify: `backend/api/v1/auth_service.go`

- [ ] **Step 1: Add the verifyEmailCode helper**

```go
// verifyEmailCode checks a submitted code against the stored row.
// Enforces expiry, attempt limit, and constant-time hash compare.
// On successful match, deletes the row (one-time use). Caller continues from there.
func (s *AuthService) verifyEmailCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose, submittedCode string) error {
	row, err := s.store.GetEmailVerificationCode(ctx, email, purpose)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get email verification code"))
	}
	if row == nil {
		return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("invalid or expired code"))
	}
	if time.Now().After(row.ExpiresAt) {
		_ = s.store.DeleteEmailVerificationCode(ctx, email, purpose)
		return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("invalid or expired code"))
	}
	if row.Attempts >= emailCodeMaxAttempts {
		_ = s.store.DeleteEmailVerificationCode(ctx, email, purpose)
		return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("too many attempts"))
	}
	if subtle.ConstantTimeCompare([]byte(s.hashEmailCode(submittedCode)), []byte(row.CodeHash)) != 1 {
		_ = s.store.IncrementEmailVerificationCodeAttempts(ctx, email, purpose)
		return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("invalid or expired code"))
	}
	_ = s.store.DeleteEmailVerificationCode(ctx, email, purpose)
	return nil
}
```

Ensure imports include `crypto/subtle`.

- [ ] **Step 2: Add the email-code branch to authenticateLogin**

Find `authenticateLogin` (around line 864). Modify to insert the email-code branch:

```go
func (s *AuthService) authenticateLogin(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	mfaSecondLogin := request.GetMfaTempToken() != ""

	if mfaSecondLogin {
		return s.completeMFALogin(ctx, request)
	}

	if request.GetIdpName() != "" {
		return s.getOrCreateUserWithIDP(ctx, request)
	}

	if request.EmailCode != nil && *request.EmailCode != "" {
		return s.authenticateEmailCodeLogin(ctx, request)
	}

	return s.getAndVerifyUser(ctx, request)
}
```

- [ ] **Step 3: Implement authenticateEmailCodeLogin**

```go
// authenticateEmailCodeLogin handles the email + 6-digit code flow.
// Existing users: verify code → return user. Unknown emails: verify code → provision new user + workspace.
func (s *AuthService) authenticateEmailCodeLogin(ctx context.Context, request *v1pb.LoginRequest) (*store.UserMessage, error) {
	if request.Password != "" || request.GetIdpName() != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email_code is mutually exclusive with password and idp_name"))
	}
	email := strings.ToLower(strings.TrimSpace(request.Email))
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}

	if err := s.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN, *request.EmailCode); err != nil {
		return nil, err
	}

	// Resolve the workspace for this email. Existing users have it via IAM.
	// Unknown emails may have a pre-invited workspace. Check workspace-level settings
	// (allow_email_code_signin, disallow_signup) BEFORE any state mutation to avoid
	// orphaned accounts if the workspace disallows email-code signin.
	preInvitedWorkspaces, err := s.store.ListWorkspacesByEmail(ctx, &store.FindWorkspaceMessage{Email: email, IncludeAllUser: true})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list workspaces for email"))
	}
	if len(preInvitedWorkspaces) > 0 {
		profile, err := s.store.GetWorkspaceProfileSetting(ctx, preInvitedWorkspaces[0].ResourceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to load workspace profile"))
		}
		if profile != nil && !profile.AllowEmailCodeSignin {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("email code login is not enabled for this workspace"))
		}
	}

	// Existing user → return.
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user"))
	}
	if user != nil {
		return user, nil
	}

	// Unknown email → signup path.
	// Respect disallow_signup only if the email was pre-invited to an existing workspace
	// that has disallow_signup=true. If no workspace is resolvable (SaaS first-time user),
	// signup proceeds normally — provisionWorkspaceForNewUser creates a new workspace.
	// Fail closed on store errors: a transient failure must not let us bypass the signup policy.
	if len(preInvitedWorkspaces) > 0 {
		profile, err := s.store.GetWorkspaceProfileSetting(ctx, preInvitedWorkspaces[0].ResourceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to load workspace profile"))
		}
		if profile != nil && profile.DisallowSignup {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("account not found"))
		}
	}

	// Create principal with random bcrypt password.
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate random password"))
	}
	passwordHash, err := bcrypt.GenerateFromPassword(randomBytes, bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to hash password"))
	}

	// Derive display name from email local-part.
	name := email
	if i := strings.Index(email, "@"); i > 0 {
		name = email[:i]
	}

	newUser, err := s.store.CreateUser(ctx, &store.UserMessage{
		Email:        email,
		Name:         name,
		Type:         storepb.PrincipalType_END_USER,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create user"))
	}

	// Provision workspace for the new user (joins pre-invited workspace or creates a new one).
	// Brand-new workspaces inherit allow_email_code_signin from getAdditionalWorkspaceSettings
	// (enabled by default in SaaS when EMAIL_CONFIG is set), so no separate check is needed.
	if _, err := s.provisionWorkspaceForNewUser(ctx, email); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to provision workspace"))
	}

	return newUser, nil
}
```

Note: double-check the exact `store.CreateUser` signature and `UserMessage` shape while implementing — match the Signup RPC's usage exactly.

- [ ] **Step 4: Build + lint**

```bash
go build ./backend/api/v1/... && golangci-lint run --allow-parallel-runners ./backend/api/v1/...
```

Expected: clean build, 0 lint issues.

---

### Task 11: Rewrite RequestPasswordReset + ResetPassword (code-based)

**Files:**
- Modify: `backend/api/v1/auth_service.go`

- [ ] **Step 1: Replace RequestPasswordReset body**

Replace the existing `RequestPasswordReset` (lines ~1396-1406) with:

```go
// RequestPasswordReset sends a password reset email. Always returns success to avoid leaking email existence.
func (s *AuthService) RequestPasswordReset(ctx context.Context, req *connect.Request[v1pb.RequestPasswordResetRequest]) (*connect.Response[emptypb.Empty], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}

	// Reject if the account doesn't exist — but return success silently (no email enumeration).
	// The cooldown + verification check is handled inside sendEmailVerificationCode.
	go s.sendEmailVerificationCode(
		context.WithoutCancel(ctx),
		email,
		storepb.EmailVerificationCodePurpose_PASSWORD_RESET,
		"[Bytebase] Reset your password",
		"Hi,\n\nYour password reset code is: %s\n\nThis code expires in 10 minutes. If you didn't request this, you can safely ignore this email.\n\n— Bytebase",
	)

	return connect.NewResponse(&emptypb.Empty{}), nil
}
```

- [ ] **Step 2: Replace ResetPassword body**

Replace the existing `ResetPassword` (lines ~1409-1449) with:

```go
// ResetPassword verifies the 6-digit code and updates the user's password.
// Also revokes all refresh tokens to force re-login.
func (s *AuthService) ResetPassword(ctx context.Context, req *connect.Request[v1pb.ResetPasswordRequest]) (*connect.Response[emptypb.Empty], error) {
	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("email is required"))
	}
	if req.Msg.Code == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("code is required"))
	}
	if req.Msg.NewPassword == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("new_password is required"))
	}

	if err := s.verifyEmailCode(ctx, email, storepb.EmailVerificationCodePurpose_PASSWORD_RESET, req.Msg.Code); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find user"))
	}
	if user == nil {
		// Should be rare — row existed when code was sent. Generic error.
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user not found"))
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Msg.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to hash password"))
	}
	passwordHashStr := string(passwordHash)
	if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
		Email:        &user.Email,
		PasswordHash: &passwordHashStr,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update password"))
	}

	if err := s.store.DeleteWebRefreshTokensByUser(ctx, user.Email); err != nil {
		slog.Warn("failed to revoke refresh tokens after password reset", log.BBError(err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}
```

- [ ] **Step 3: Delete the now-unused `sendPasswordResetEmail` function**

The old `sendPasswordResetEmail` (lines ~1451-1500) is no longer used. Delete it in its entirety.

- [ ] **Step 4: Build**

```bash
go build ./backend/... 2>&1 | tail
```

Expected: clean build. If `fmt` or `utils` imports become unused after deletion, remove them.

---

### Task 12: Restriction — expose allow_email_code_signin to actuator

**Files:**
- Modify: `backend/api/v1/auth_service.go:1346-1393` (`getAccountRestriction`)

- [ ] **Step 1: Add the field population**

Inside `getAccountRestriction`, find where `DisallowPasswordSignin` is set (twice — default struct init and the workspace-specific block). Mirror `AllowEmailCodeSignin`:

In the default restriction block:
```go
restriction := &v1pb.Restriction{
	DisallowSignup:         false,
	DisallowPasswordSignin: false,
	PasswordRestriction:    defaultPasswordRestriction,
	AllowEmailCodeSignin:   false,
}
```

In the workspace-specific block (where `setting` is loaded):
```go
restriction = &v1pb.Restriction{
	PasswordRestriction:    convertToV1PasswordRestriction(setting.GetPasswordRestriction()),
	DisallowSignup:         setting.DisallowSignup,
	DisallowPasswordSignin: setting.DisallowPasswordSignin,
	AllowEmailCodeSignin:   setting.AllowEmailCodeSignin,
}
```

No license-gate for `AllowEmailCodeSignin` — the feature requires EMAIL setting, which is already the effective gate.

- [ ] **Step 2: Build**

```bash
go build ./backend/api/v1/...
```

Expected: clean.

---

### Task 13: Setting service — validate allow_email_code_signin + SaaS read-only

**Files:**
- Modify: `backend/api/v1/setting_service.go` (around lines 289-308, the existing `disallow_password_signin` validation)

- [ ] **Step 1: Locate the WORKSPACE_PROFILE update_mask switch**

Open `setting_service.go`. Find the `UpdateSetting` function's WORKSPACE_PROFILE case. Inside it, there's a switch on `update_mask` paths, including `value.workspace_profile.disallow_password_signin` (around line 289).

- [ ] **Step 2: Add a new case for allow_email_code_signin**

Next to the `disallow_password_signin` case, add:

```go
case "value.workspace_profile.allow_email_code_signin":
	if s.profile.SaaS {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("allow_email_code_signin cannot be changed in SaaS mode"))
	}
	if payload.AllowEmailCodeSignin {
		emailSetting, err := s.store.GetSetting(ctx, workspaceID, storepb.SettingName_EMAIL)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to load email setting"))
		}
		if emailSetting == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("cannot enable email code signin without an EMAIL setting"))
		}
	}
	oldSetting.AllowEmailCodeSignin = payload.AllowEmailCodeSignin
```

The field naming (`oldSetting`, `payload`, etc.) depends on the surrounding code — mirror the pattern used by the `disallow_password_signin` case exactly.

- [ ] **Step 3: Build + lint**

```bash
go build ./backend/api/v1/... && golangci-lint run --allow-parallel-runners ./backend/api/v1/...
```

Expected: clean.

---

### Task 14: Inject allow_email_code_signin on SaaS workspace creation

**Files:**
- Modify: `backend/api/v1/auth_service.go` — `getAdditionalWorkspaceSettings` (or wherever the SaaS Signup workspace defaults are set)

- [ ] **Step 1: Find where WorkspaceProfileSetting defaults are assembled in SaaS**

The Signup RPC (or `getAdditionalWorkspaceSettings`) constructs the initial WORKSPACE_PROFILE setting for a new SaaS workspace. It already sets `DisallowPasswordSignin = true`. Find that block.

- [ ] **Step 2: Add allow_email_code_signin injection**

Inside the same block (SaaS + `EMAIL_CONFIG` env var present), set:

```go
if saas && os.Getenv("EMAIL_CONFIG") != "" {
	workspaceProfile.AllowEmailCodeSignin = true
}
```

If the construction happens by way of an `AdditionalSetting` (like the EMAIL setting itself), add a similar entry for WORKSPACE_PROFILE — or modify the existing WORKSPACE_PROFILE injection to include this field.

- [ ] **Step 3: Build**

```bash
go build ./backend/api/v1/...
```

Expected: clean.

---

### Task 15: Background cleanup sweeper

**Files:**
- Modify: `backend/runner/cleaner/data_cleaner.go` (around line 78-83 for main loop, and wherever `cleanupWebRefreshTokens` is defined)

- [ ] **Step 1: Add cleanup method**

Near `cleanupWebRefreshTokens`, add:

```go
func (c *DataCleaner) cleanupEmailVerificationCodes(ctx context.Context) {
	n, err := c.store.DeleteExpiredEmailVerificationCodes(ctx)
	if err != nil {
		slog.Error("Failed to clean up expired email verification codes", log.BBError(err))
		return
	}
	if n > 0 {
		slog.Info("Cleaned up expired email verification codes", slog.Int64("count", n))
	}
}
```

- [ ] **Step 2: Wire into the sweeper loop**

Find the loop that calls `c.cleanupWebRefreshTokens(ctx)`. Immediately after that call, add:

```go
c.cleanupEmailVerificationCodes(ctx)
```

- [ ] **Step 3: Build**

```bash
go build ./backend/runner/...
```

Expected: clean.

---

### Task 16: Integration tests for auth flow

**Files:**
- Create or modify: `backend/tests/auth_test.go` (project-standard integration test location)

- [ ] **Step 1: Add test cases**

Find the existing auth integration test file (check `backend/tests/` for test files that exercise the `Login` RPC). Add these cases, mirroring the style of existing tests:

```go
func TestEmailCodeLogin_HappyPath(t *testing.T) {
	// Setup: create workspace with EMAIL setting + allow_email_code_signin=true.
	// 1. Call SendEmailLoginCode(email).
	// 2. Extract the generated code from the test mail sink (intercept mailer.Send).
	// 3. Call Login(email, email_code=code).
	// 4. Assert token returned + user exists (or was created).
}

func TestEmailCodeLogin_ResendWithinCooldown(t *testing.T) {
	// 1. SendEmailLoginCode.
	// 2. Immediately SendEmailLoginCode again.
	// 3. Assert second call does not reset last_sent_at (query DB directly).
}

func TestEmailCodeLogin_WrongCodeLocksOut(t *testing.T) {
	// 1. SendEmailLoginCode.
	// 2. Call Login(email_code=wrong) 5 times → each returns Unauthenticated "invalid or expired code".
	// 3. 6th Login(email_code=wrong) returns "too many attempts".
	// 4. DB row deleted.
}

func TestEmailCodeLogin_DisallowSignup(t *testing.T) {
	// Setup: workspace with disallow_signup=true.
	// Call Login(new-email, email_code=valid).
	// Assert Unauthenticated "account not found".
}

func TestEmailCodeLogin_WorkspaceDisallows(t *testing.T) {
	// Setup: existing user in workspace with allow_email_code_signin=false.
	// 1. SendEmailLoginCode → returns success (cooldown/send always succeeds).
	// 2. Login(valid code) → FailedPrecondition "email code login is not enabled".
}

func TestEmailCodeLogin_2FAInteraction(t *testing.T) {
	// Setup: existing user with TOTP enabled.
	// Login(email_code=valid) → response has mfa_temp_token, requires MFA completion.
}

func TestPasswordReset_CodeFlow(t *testing.T) {
	// 1. RequestPasswordReset(email).
	// 2. Extract code from mail sink.
	// 3. ResetPassword(email, code, new_password) → success.
	// 4. Login(email, password=new_password) → success.
	// 5. Old refresh tokens invalidated.
}
```

- [ ] **Step 2: Provide the mail-sink test helper**

If a mail-capture helper doesn't exist, either:
- Use a `mailer.Sender` interface stub injected into the test harness.
- Or read the code directly from the DB (`GetEmailVerificationCode`) and hash-compare — acceptable shortcut since the tests verify end-to-end behavior.

Prefer the DB approach for simplicity (no sender mock needed).

- [ ] **Step 3: Run**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/tests -run TestEmailCodeLogin
```

Expected: all PASS.

---

### Task 17: Frontend store — sendEmailLoginCode + login with emailCode

**Files:**
- Modify: `frontend/src/store/modules/v1/auth.ts`

- [ ] **Step 1: Add sendEmailLoginCode action**

Inside the auth store's `defineStore` body, add:

```ts
const sendEmailLoginCode = async (email: string) => {
  await authServiceClientConnect.sendEmailLoginCode(
    create(SendEmailLoginCodeRequestSchema, { email })
  );
};
```

Import `SendEmailLoginCodeRequestSchema` from `@/types/proto-es/v1/auth_service_pb`.

- [ ] **Step 2: Expose sendEmailLoginCode in the return statement**

Find the `return { ... }` at the end of `defineStore`. Add `sendEmailLoginCode` to the exported object.

- [ ] **Step 3: Verify `login()` threads emailCode through**

The existing `login()` calls `authServiceClientConnect.login(request)`. The generated `LoginRequest` now has `emailCode` after proto regen. No code changes needed — callers pass `emailCode` via the `request` object.

- [ ] **Step 4: Type-check**

```bash
pnpm --dir frontend type-check
```

Expected: clean.

---

### Task 18: Frontend — Signin.vue email-code tab

**Files:**
- Create: `frontend/src/components/EmailCodeSigninForm.vue`
- Modify: `frontend/src/views/auth/Signin.vue`

- [ ] **Step 1: Create EmailCodeSigninForm component**

File: `frontend/src/components/EmailCodeSigninForm.vue`

```vue
<template>
  <form class="flex flex-col gap-y-6" @submit.prevent="handleSubmit">
    <div v-if="step === 'email'">
      <label for="email-code-email" class="block text-sm font-medium leading-5 text-control">
        {{ $t("common.email") }}
      </label>
      <div class="mt-1 rounded-md shadow-xs">
        <BBTextField
          v-model:value="state.email"
          required
          type="email"
          :input-props="{ id: 'email-code-email', autocomplete: 'email' }"
        />
      </div>
      <NButton
        class="mt-4 w-full"
        type="primary"
        size="large"
        :disabled="!state.email || state.sending"
        :loading="state.sending"
        @click="sendCode"
      >
        {{ $t("auth.sign-in.send-code") }}
      </NButton>
    </div>

    <div v-else class="flex flex-col gap-y-4">
      <div class="text-sm text-control">
        <span>{{ $t("auth.sign-in.code-sent-hint", { email: state.email }) }}</span>
        <button type="button" class="ml-2 accent-link" @click="step = 'email'">
          {{ $t("common.change") }}
        </button>
      </div>
      <NInputOtp
        v-model:value="state.code"
        :length="6"
        @finish="handleSubmit"
      />
      <div class="flex items-center justify-between">
        <NButton
          text
          :disabled="resendCountdown > 0"
          @click="sendCode"
        >
          {{ resendCountdown > 0
            ? $t("auth.sign-in.resend-in", { seconds: resendCountdown })
            : $t("auth.sign-in.resend-code") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="state.code.length !== 6 || props.loading"
          :loading="props.loading"
          @click="handleSubmit"
        >
          {{ $t("common.verify") }}
        </NButton>
      </div>
    </div>
  </form>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton, NInputOtp } from "naive-ui";
import { onUnmounted, reactive, ref } from "vue";
import { BBTextField } from "@/bbkit";
import { pushNotification, useAuthStore } from "@/store";
import { LoginRequestSchema } from "@/types/proto-es/v1/auth_service_pb";

const props = defineProps<{ loading: boolean }>();
const emit = defineEmits<{
  (e: "signin", request: ReturnType<typeof create<typeof LoginRequestSchema>>): void;
}>();

const authStore = useAuthStore();
const step = ref<"email" | "code">("email");
const resendCountdown = ref(0);
let countdownTimer: ReturnType<typeof setInterval> | null = null;

const state = reactive({
  email: "",
  code: "",
  sending: false,
});

const startCountdown = () => {
  resendCountdown.value = 60;
  if (countdownTimer) clearInterval(countdownTimer);
  countdownTimer = setInterval(() => {
    resendCountdown.value -= 1;
    if (resendCountdown.value <= 0 && countdownTimer) {
      clearInterval(countdownTimer);
      countdownTimer = null;
    }
  }, 1000);
};

onUnmounted(() => {
  if (countdownTimer) clearInterval(countdownTimer);
});

const sendCode = async () => {
  if (!state.email || state.sending || resendCountdown.value > 0) return;
  state.sending = true;
  try {
    await authStore.sendEmailLoginCode(state.email);
    step.value = "code";
    startCountdown();
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Failed to send code",
    });
  } finally {
    state.sending = false;
  }
};

const handleSubmit = () => {
  if (state.code.length !== 6) return;
  emit(
    "signin",
    create(LoginRequestSchema, {
      email: state.email,
      emailCode: state.code,
      web: true,
    })
  );
};
</script>
```

- [ ] **Step 2: Add the email-code tab to Signin.vue**

Open `Signin.vue`. Find the `<NTabs>` block containing the "Standard" `NTabPane` and any IDP `NTabPane`s. After the standard tab's closing tag, add:

```vue
<NTabPane
  v-if="serverInfo?.restriction?.allowEmailCodeSignin"
  name="email-code"
  :tab="$t('auth.sign-in.email-code-tab')"
>
  <EmailCodeSigninForm
    :loading="state.isLoading"
    @signin="trySignin"
  />
</NTabPane>
```

Add the import:
```ts
import EmailCodeSigninForm from "@/components/EmailCodeSigninForm.vue";
```

- [ ] **Step 3: Type-check**

```bash
pnpm --dir frontend type-check
```

Expected: clean.

---

### Task 19: Frontend — PasswordForgot.vue (add cooldown)

**Files:**
- Modify: `frontend/src/views/auth/PasswordForgot.vue`

- [ ] **Step 1: Add 60-second client-side cooldown after successful send**

In the existing `onSubmit` handler, after the successful `requestPasswordReset` call:

```ts
state.sending = true;
state.sent = true;
startResendCountdown();
```

Add the countdown state + timer (pattern from Task 18's EmailCodeSigninForm).

- [ ] **Step 2: Update "Send reset link" button copy**

Rename button label from "Send reset link" to "Send code" (i18n key update in Task 22). Body copy hint: replace "We've sent a password reset link" with "We've sent a 6-digit code to {email}".

- [ ] **Step 3: Navigate to password-reset with email prefill**

After successful send, instead of just showing the success message in-place, offer a button or auto-navigate to `/auth/password-reset?email={email}`.

- [ ] **Step 4: Type-check**

```bash
pnpm --dir frontend type-check
```

Expected: clean.

---

### Task 20: Frontend — PasswordReset.vue rewrite for code-based flow

**Files:**
- Modify: `frontend/src/views/auth/PasswordReset.vue`

- [ ] **Step 1: Determine "mode" from route**

Keep existing "forced reset after login" mode. Detect if route has `email` query param — that signals the new code-based mode.

- [ ] **Step 2: Add code-based UI branch**

Replace the template with a conditional structure:

```vue
<template>
  <div class="mx-auto w-full max-w-sm">
    <img class="h-12 w-auto" src="@/assets/logo-full.svg" alt="Bytebase" />
    <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
      {{ $t("auth.password-reset.title") }}
    </h2>
    <p class="textinfo mt-2">{{ $t("auth.password-reset.content") }}</p>

    <div class="mt-8 flex flex-col gap-y-6">
      <!-- Code-based mode: email + code fields -->
      <template v-if="codeMode">
        <div>
          <label class="block text-sm font-medium leading-5 text-control">
            {{ $t("common.email") }}
          </label>
          <BBTextField v-model:value="state.email" type="email" class="mt-1" />
        </div>
        <div>
          <label class="block text-sm font-medium leading-5 text-control">
            {{ $t("auth.password-reset.code-label") }}
          </label>
          <NInputOtp v-model:value="state.code" :length="6" class="mt-1" />
        </div>
      </template>

      <!-- Both modes: new password -->
      <UserPassword
        ref="userPasswordRef"
        v-model:password="state.password"
        v-model:password-confirm="state.passwordConfirm"
        :show-learn-more="false"
        :password-restriction="passwordRestrictionSetting"
      />

      <NButton
        type="primary"
        size="large"
        style="width: 100%"
        :disabled="!allowConfirm"
        @click="onConfirm"
      >
        {{ $t("common.confirm") }}
      </NButton>
    </div>
  </div>
</template>
```

- [ ] **Step 3: Add onConfirm logic for code-based mode**

Inside the component:

```ts
const codeMode = computed(() => !!route.query.email);

onMounted(() => {
  if (codeMode.value) {
    state.email = route.query.email as string;
    return; // Don't require login for code-mode.
  }
  if (!authStore.requireResetPassword) {
    router.replace(redirectQuery.value);
  }
});

const allowConfirm = computed(() => {
  if (!state.password) return false;
  if (codeMode.value && (!state.email || state.code.length !== 6)) return false;
  return !userPasswordRef.value?.passwordHint && !userPasswordRef.value?.passwordMismatch;
});

const onConfirm = async () => {
  if (codeMode.value) {
    try {
      await authServiceClientConnect.resetPassword(
        create(ResetPasswordRequestSchema, {
          email: state.email,
          code: state.code,
          newPassword: state.password,
        })
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      router.push({ name: AUTH_SIGNIN_MODULE, query: { email: state.email } });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Invalid or expired code",
      });
    }
    return;
  }

  // Legacy forced-reset mode — unchanged.
  const patch: User = { ...me.value, password: state.password };
  await userStore.updateUser(
    createProto(UpdateUserRequestSchema, {
      user: patch,
      updateMask: createProto(FieldMaskSchema, { paths: ["password"] }),
    })
  );
  pushNotification({ module: "bytebase", style: "SUCCESS", title: t("common.updated") });
  authStore.setRequireResetPassword(false);
  router.replace(redirectQuery.value);
};
```

Add imports for `ResetPasswordRequestSchema`, `authServiceClientConnect`, `NInputOtp`, `BBTextField`, `AUTH_SIGNIN_MODULE`.

- [ ] **Step 4: Remove the old `?token=` path**

The pre-existing `route.query.token` branch (if any) can be deleted — we no longer issue tokens.

- [ ] **Step 5: Type-check**

```bash
pnpm --dir frontend type-check
```

Expected: clean.

---

### Task 21: Frontend — router query preservation

**Files:**
- Modify: `frontend/src/router/index.ts`

- [ ] **Step 1: Ensure `email` is preserved in SIGNIN_QUERY_PARAMS**

Grep the file for `SIGNIN_QUERY_PARAMS`. It should already include `email` (we added it earlier for invite flow). If not, add it to the array. No other router changes needed.

- [ ] **Step 2: Type-check**

```bash
pnpm --dir frontend type-check
```

Expected: clean.

---

### Task 22: i18n keys (5 locales)

**Files:**
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/zh-CN.json`
- Modify: `frontend/src/locales/ja-JP.json`
- Modify: `frontend/src/locales/es-ES.json`
- Modify: `frontend/src/locales/vi-VN.json`

- [ ] **Step 1: en-US**

Under `auth.sign-in`, add:
```json
"email-code-tab": "Email code",
"send-code": "Send code",
"resend-code": "Resend code",
"resend-in": "Resend in {seconds}s",
"code-sent-hint": "We've sent a 6-digit code to {email}"
```

Under `auth.password-reset`, add:
```json
"code-label": "Verification code"
```

Under `auth.password-forget`, replace existing keys if needed — ensure `send-code`, `email-sent`, `email-sent-hint` are present (may already exist from prior work; dedup if so).

- [ ] **Step 2: Translate to zh-CN, ja-JP, es-ES, vi-VN**

Mirror the keys. Translations:

| en-US | zh-CN | ja-JP | es-ES | vi-VN |
|-------|-------|-------|-------|-------|
| Email code | 邮箱验证码 | メール認証コード | Código por email | Mã qua email |
| Send code | 发送验证码 | コードを送信 | Enviar código | Gửi mã |
| Resend code | 重新发送 | コードを再送信 | Reenviar código | Gửi lại mã |
| Resend in {seconds}s | {seconds} 秒后可重发 | {seconds}秒後に再送信可能 | Reenviar en {seconds}s | Gửi lại sau {seconds}s |
| We've sent a 6-digit code to {email} | 已向 {email} 发送 6 位验证码 | {email} に6桁のコードを送信しました | Hemos enviado un código de 6 dígitos a {email} | Đã gửi mã 6 chữ số đến {email} |
| Verification code | 验证码 | 認証コード | Código de verificación | Mã xác minh |

- [ ] **Step 3: Run i18n sorter + type-check**

```bash
pnpm --dir frontend fix && pnpm --dir frontend type-check
```

Expected: no changes from sorter (or it auto-sorts), type-check clean.

---

### Task 23: Final validation pass

**Files:** None new — validation.

- [ ] **Step 1: Full backend build**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: success.

- [ ] **Step 2: Backend lint (full)**

```bash
golangci-lint run --allow-parallel-runners
```

Expected: 0 issues.

- [ ] **Step 3: Backend tests**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/store -run TestEmailVerificationCode
go test -v -count=1 github.com/bytebase/bytebase/backend/tests -run TestEmailCodeLogin
go test -v -count=1 github.com/bytebase/bytebase/backend/tests -run TestPasswordReset
```

Expected: all PASS.

- [ ] **Step 4: Frontend fix + type-check**

```bash
pnpm --dir frontend fix && pnpm --dir frontend type-check
```

Expected: no errors.

- [ ] **Step 5: Frontend tests**

```bash
pnpm --dir frontend test
```

Expected: all PASS.
