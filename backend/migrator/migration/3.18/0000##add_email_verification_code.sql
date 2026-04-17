CREATE TABLE email_verification_code (
    email         text NOT NULL,
    -- Stored as EmailVerificationCodePurpose enum name (proto/store/store/email_verification_code.proto)
    purpose       text NOT NULL,
    code_hash     text NOT NULL,
    attempts      int  NOT NULL DEFAULT 0,
    expires_at    timestamptz NOT NULL,
    last_sent_at  timestamptz NOT NULL,
    -- Workspace context captured at send time. Used at verify time for gate checks
    -- (disallow_signup, allow_email_code_signin) and for provisionWorkspaceForNewUser.
    -- NULL for SaaS brand-new signup (no workspace exists yet — provision creates one).
    workspace     text,
    PRIMARY KEY (email, purpose)
);

CREATE INDEX idx_email_verification_code_expires_at ON email_verification_code (expires_at);
