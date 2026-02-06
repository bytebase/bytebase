-- Migrate policy members from legacy "users/" prefix to proper prefixes:
-- - Service accounts: "users/xxx@..." -> "serviceAccounts/xxx@..."
-- - Workload identities: "users/xxx@..." -> "workloadIdentities/xxx@..."
-- Uses principal table to determine the actual type.
-- Only processes members with "users/" prefix; already-migrated data is unchanged.

-- Update IAM policies in the policy table
-- IAM policy structure: { bindings: [{ members: [...], role: "..." }] }
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{bindings}',
        (
            SELECT jsonb_agg(
                jsonb_set(
                    binding,
                    '{members}',
                    (
                        SELECT jsonb_agg(
                            CASE
                                WHEN member_text NOT LIKE 'users/%'
                                THEN member
                                WHEN p.type = 'SERVICE_ACCOUNT'
                                THEN to_jsonb('serviceAccounts/' || email_extracted.email)
                                WHEN p.type = 'WORKLOAD_IDENTITY'
                                THEN to_jsonb('workloadIdentities/' || email_extracted.email)
                                ELSE member
                            END
                        )
                        FROM jsonb_array_elements(binding->'members') AS member,
                             LATERAL (SELECT member #>> '{}' AS member_text) AS extracted,
                             LATERAL (SELECT substring(member_text FROM 7) AS email) AS email_extracted
                        LEFT JOIN principal p ON p.email = email_extracted.email
                    )
                )
            )
            FROM jsonb_array_elements(payload->'bindings') AS binding
        )
    )
)
WHERE type = 'IAM'
  AND payload->'bindings' IS NOT NULL;

-- Update Masking Exemption policies in the policy table
-- Masking Exemption policy structure: { exemptions: [{ members: [...], condition: {...} }] }
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{exemptions}',
        (
            SELECT jsonb_agg(
                jsonb_set(
                    exemption,
                    '{members}',
                    (
                        SELECT jsonb_agg(
                            CASE
                                WHEN member_text NOT LIKE 'users/%'
                                THEN member
                                WHEN p.type = 'SERVICE_ACCOUNT'
                                THEN to_jsonb('serviceAccounts/' || email_extracted.email)
                                WHEN p.type = 'WORKLOAD_IDENTITY'
                                THEN to_jsonb('workloadIdentities/' || email_extracted.email)
                                ELSE member
                            END
                        )
                        FROM jsonb_array_elements(exemption->'members') AS member,
                             LATERAL (SELECT member #>> '{}' AS member_text) AS extracted,
                             LATERAL (SELECT substring(member_text FROM 7) AS email) AS email_extracted
                        LEFT JOIN principal p ON p.email = email_extracted.email
                    )
                )
            )
            FROM jsonb_array_elements(payload->'exemptions') AS exemption
        )
    )
)
WHERE type = 'MASKING_EXEMPTION'
  AND payload->'exemptions' IS NOT NULL;
