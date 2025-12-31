-- Migrate user references from users/{id} format to users/{email} format
--
-- Key optimizations:
-- 1. Pre-build id->email lookup table (eliminates correlated subqueries)
-- 2. O(1) hash lookups instead of O(n×m) correlated subqueries
--
-- NOTE: The audit_log table migration is handled in Go code (migrator.go)
-- to support batching and avoid timeout issues on large datasets.

-- ============================================================================
-- Build temporary lookup table for O(1) id->email conversion
-- ============================================================================
CREATE TEMP TABLE user_id_to_email AS
SELECT
    id,
    'users/' || email AS email_format,
    email
FROM principal
WHERE email IS NOT NULL;

-- Index for fast lookups
CREATE INDEX idx_temp_user_id ON user_id_to_email(id);

-- ============================================================================
-- 1. Update policy table - IAM bindings members
-- ============================================================================
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        policy.payload,
        '{bindings}',
        COALESCE(
            (
                SELECT jsonb_agg(
                    jsonb_set(
                        binding,
                        '{members}',
                        COALESCE(
                            (
                                SELECT jsonb_agg(
                                    CASE
                                        WHEN member LIKE 'users/%' AND member NOT LIKE 'users/%@%' THEN
                                            COALESCE(
                                                (
                                                    SELECT u.email_format
                                                    FROM user_id_to_email u
                                                    WHERE u.id = CAST(SUBSTRING(member FROM 7) AS INTEGER)
                                                ),
                                                member  -- Keep original if not found
                                            )
                                        ELSE member
                                    END
                                )
                                FROM jsonb_array_elements_text(binding->'members') AS member
                            ),
                            '[]'::jsonb
                        )
                    )
                )
                FROM jsonb_array_elements(policy.payload->'bindings') AS binding
            ),
            '[]'::jsonb
        )
    )
)
WHERE type = 'IAM'
  AND payload ? 'bindings'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(payload->'bindings') AS binding,
           jsonb_array_elements_text(binding->'members') AS member
      WHERE member LIKE 'users/%' AND member NOT LIKE 'users/%@%'
  );

-- ============================================================================
-- 2. Update policy table - Masking exception members
-- ============================================================================
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        policy.payload,
        '{maskingExceptions}',
        COALESCE(
            (
                SELECT jsonb_agg(
                    CASE
                        WHEN exception->>'member' LIKE 'users/%' AND exception->>'member' NOT LIKE 'users/%@%' THEN
                            jsonb_set(
                                exception,
                                '{member}',
                                to_jsonb(COALESCE(
                                    (
                                        SELECT u.email_format
                                        FROM user_id_to_email u
                                        WHERE u.id = CAST(SUBSTRING(exception->>'member' FROM 7) AS INTEGER)
                                    ),
                                    exception->>'member'
                                ))
                            )
                        ELSE exception
                    END
                )
                FROM jsonb_array_elements(policy.payload->'maskingExceptions') AS exception
            ),
            '[]'::jsonb
        )
    )
)
WHERE type = 'MASKING_EXCEPTION'
  AND payload ? 'maskingExceptions'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(payload->'maskingExceptions') AS exception
      WHERE exception->>'member' LIKE 'users/%' AND exception->>'member' NOT LIKE 'users/%@%'
  );

-- ============================================================================
-- 3. Update user_group table - Group members
-- ============================================================================
UPDATE user_group
SET payload = (
    SELECT jsonb_set(
        user_group.payload,
        '{members}',
        COALESCE(
            (
                SELECT jsonb_agg(
                    CASE
                        WHEN member->>'member' LIKE 'users/%' AND member->>'member' NOT LIKE 'users/%@%' THEN
                            jsonb_set(
                                member,
                                '{member}',
                                to_jsonb(COALESCE(
                                    (
                                        SELECT u.email_format
                                        FROM user_id_to_email u
                                        WHERE u.id = CAST(SUBSTRING(member->>'member' FROM 7) AS INTEGER)
                                    ),
                                    member->>'member'
                                ))
                            )
                        ELSE member
                    END
                )
                FROM jsonb_array_elements(user_group.payload->'members') AS member
            ),
            '[]'::jsonb
        )
    )
)
WHERE payload ? 'members'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(payload->'members') AS member
      WHERE member->>'member' LIKE 'users/%' AND member->>'member' NOT LIKE 'users/%@%'
  );

-- ============================================================================
-- 4. Update issue table - Grant request user
-- ============================================================================
UPDATE issue
SET payload = (
    SELECT jsonb_set(
        issue.payload,
        '{grantRequest,user}',
        to_jsonb(COALESCE(
            (
                SELECT u.email_format
                FROM user_id_to_email u
                WHERE u.id = CAST(SUBSTRING(issue.payload->'grantRequest'->>'user' FROM 7) AS INTEGER)
            ),
            issue.payload->'grantRequest'->>'user'
        ))
    )
)
WHERE payload->'grantRequest'->>'user' LIKE 'users/%'
  AND payload->'grantRequest'->>'user' NOT LIKE 'users/%@%';

-- ============================================================================
-- 5. Update issue table - Approval approvers (rename principalId to principal and convert value)
-- ============================================================================
UPDATE issue
SET payload = (
    SELECT jsonb_set(
        issue.payload,
        '{approval,approvers}',
        COALESCE(
            (
                SELECT jsonb_agg(
                    CASE
                        -- If principalId exists and is numeric, convert to principal with email format
                        WHEN approver ? 'principalId' AND approver->>'principalId' ~ '^\d+$' THEN
                            jsonb_build_object(
                                'status', approver->'status',
                                'principal', to_jsonb(COALESCE(
                                    (
                                        SELECT u.email_format
                                        FROM user_id_to_email u
                                        WHERE u.id = CAST(approver->>'principalId' AS INTEGER)
                                    ),
                                    'users/' || (approver->>'principalId')
                                ))
                            )
                        -- If principalId exists but is already email format, rename to principal
                        WHEN approver ? 'principalId' THEN
                            jsonb_build_object(
                                'status', approver->'status',
                                'principal', approver->'principalId'
                            )
                        -- Otherwise keep as-is (already has principal field or no field)
                        ELSE approver
                    END
                )
                FROM jsonb_array_elements(issue.payload->'approval'->'approvers') AS approver
            ),
            '[]'::jsonb
        )
    )
)
WHERE payload->'approval' ? 'approvers'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(payload->'approval'->'approvers') AS approver
      WHERE approver ? 'principalId'
  );

-- Cleanup temp table
DROP TABLE user_id_to_email;
