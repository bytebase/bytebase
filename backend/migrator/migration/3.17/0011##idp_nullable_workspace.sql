-- Allow global IDPs (SaaS login) by making workspace nullable.
-- NULL workspace = global IDP, non-NULL = workspace-scoped IDP.
ALTER TABLE idp ALTER COLUMN workspace DROP NOT NULL;
