-- Migrate IAM binding members from legacy "users/" prefix to proper prefixes:
-- - Service accounts: "users/xxx@service.bytebase.com" -> "serviceAccounts/xxx@service.bytebase.com"
-- - Workload identities: "users/xxx@workload.bytebase.com" -> "workloadIdentities/xxx@workload.bytebase.com"

-- Update IAM policies in the policy table
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
                                -- Service account: workspace-level (ends with @service.bytebase.com)
                                WHEN member_text LIKE 'users/%@service.bytebase.com'
                                THEN to_jsonb(regexp_replace(member_text, '^users/', 'serviceAccounts/'))
                                -- Service account: project-level (contains .service.bytebase.com)
                                WHEN member_text LIKE 'users/%@%.service.bytebase.com'
                                THEN to_jsonb(regexp_replace(member_text, '^users/', 'serviceAccounts/'))
                                -- Workload identity: workspace-level (ends with @workload.bytebase.com)
                                WHEN member_text LIKE 'users/%@workload.bytebase.com'
                                THEN to_jsonb(regexp_replace(member_text, '^users/', 'workloadIdentities/'))
                                -- Workload identity: project-level (contains .workload.bytebase.com)
                                WHEN member_text LIKE 'users/%@%.workload.bytebase.com'
                                THEN to_jsonb(regexp_replace(member_text, '^users/', 'workloadIdentities/'))
                                ELSE member
                            END
                        )
                        FROM jsonb_array_elements(binding->'members') AS member,
                             LATERAL (SELECT member #>> '{}' AS member_text) AS extracted
                    )
                )
            )
            FROM jsonb_array_elements(payload->'bindings') AS binding
        )
    )
)
WHERE type = 'IAM'
  AND payload->'bindings' IS NOT NULL;
