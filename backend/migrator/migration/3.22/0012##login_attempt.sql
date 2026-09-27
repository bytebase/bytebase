-- One table gives every credential — password, emailed code, second factor — an attempt
-- limit keyed on the identity under attack (docs/design/login-attempt-lockout.md). It
-- replaces the audit-log lockout counters and the per-code attempts column.
-- One row per (identity, kind): attempts since the last success, and when the latest was.
CREATE TABLE login_attempt (
    -- The identity under attack, server-resolved and globally unique: the normalized
    -- email, or the identity-provider ID joined with the submitted username for LDAP.
    -- Not a FK, so unknown identities count too (no existence oracle).
    identity        text NOT NULL,
    -- Stored as LoginAttemptKind enum name (proto/store/store/auth.proto):
    -- PASSWORD | EMAIL_CODE | MFA.
    kind            text NOT NULL,
    attempts        int NOT NULL,
    last_attempt_at timestamptz NOT NULL,
    PRIMARY KEY (identity, kind)
);

CREATE INDEX idx_login_attempt_last_attempt_at ON login_attempt (last_attempt_at);

-- The per-code attempt counter is replaced by the EMAIL_CODE lockout above. The code row
-- now lives until it expires or is consumed, so the resend cooldown always has a row to
-- evaluate: the delete-on-exhaustion cooldown bypass is closed structurally.
ALTER TABLE email_verification_code DROP COLUMN attempts;

-- Verify never reads a caller-supplied workspace; the password policy and audit
-- workspace come from what the server knows about the email at verify time.
ALTER TABLE email_verification_code DROP COLUMN workspace;
