-- Record the audiences that deployed pipelines already request, so that turning
-- on audience validation does not break them.
--
-- Bytebase has shipped two audience conventions and stored neither. The
-- in-product GitOps generator requests the literal "bytebase"; the docs tell
-- operators to request the provider default. Which one a given pipeline sends
-- is not recorded anywhere, so no single value is derivable from the row.
-- allowed_audiences is repeated and matched any-of, so both are recorded.
--
-- "bytebase" is recorded for every repairable row, whatever the provider or the
-- issuer, because it is what our own generator asks for and does not depend on
-- either. The provider default is appended only where it can be derived: for
-- GitHub that needs the public issuer, since GitHub Enterprise Server has its
-- own; for GitLab it is the instance URL, which is the issuer.
--
-- Rows with no issuer never authenticated -- FetchJWKS rejects an empty URL --
-- so there is nothing to preserve and they are left alone. Rows that already
-- hold an audience are left alone too. An empty list is stored as a missing
-- key: every writer serializes with protojson, which omits an empty repeated
-- field, so that is the only shape the guard needs.
--
-- A pipeline requesting neither value stops authenticating: the audience it
-- requests is recorded nowhere, so it cannot be preserved.
--
-- The GitHub arm is keyed on the row's own evidence, the issuer and the shape
-- of the subject, not on provider_type: that enum is optional, nothing on the
-- token path reads it, and identities created through the API often leave it
-- unset. The GitLab arm accepts the enum as well as the subject shape, because
-- a GitLab row's issuer is its instance URL whatever its subject looks like.
--
-- Soft-deleted rows are repaired too, so that undeleting one restores a working
-- identity wherever an audience is derivable.
--
-- Repositories created after 2026-07-15 carry an immutable id on the owner
-- segment ("repo:octocat@123456/...", GitHub changelog 2026-04-23); the
-- audience names the owner alone.
UPDATE workload_identity
SET config = jsonb_set(
        config,
        '{allowedAudiences}',
        to_jsonb(
            ARRAY['bytebase'] || CASE
                WHEN config->>'issuerUrl' = 'https://token.actions.githubusercontent.com'
                 AND split_part(substring(config->>'subjectPattern' FROM '^repo:([^/]+)/'), '@', 1) NOT IN ('', '*')
                THEN ARRAY['https://github.com/' || split_part(substring(config->>'subjectPattern' FROM '^repo:([^/]+)/'), '@', 1)]

                WHEN config->>'subjectPattern' LIKE 'project\_path:%'
                  OR config->>'providerType' = 'GITLAB'
                THEN ARRAY[config->>'issuerUrl']

                ELSE ARRAY[]::text[]
            END
        )
    )
WHERE config->'allowedAudiences' IS NULL
  AND COALESCE(config->>'issuerUrl', '') <> '';

-- Type the rows that never declared a provider.
--
-- provider_type was optional until this release, so identities written through
-- the API carry no value. The subject's own prefix says which vocabulary the
-- row uses, and it is the same evidence the audience arms above read, so the
-- label can be settled once here instead of being inferred on every read.
--
-- Keyed on the subject prefix alone: the issuer is not evidence about the
-- subject vocabulary, and a self-hosted GitLab or GitHub Enterprise row must
-- be typed too. No `deleted` filter, matching the arms above, so undeleting a
-- row cannot bring back an untyped one.
--
-- The values are the enum names as JSON strings, which is how protojson writes
-- the field and how the GitLab arm above reads it.
UPDATE workload_identity
SET config = jsonb_set(config, '{providerType}', '"GITHUB"')
WHERE config->'providerType' IS NULL
  AND config->>'subjectPattern' LIKE 'repo:%';

UPDATE workload_identity
SET config = jsonb_set(config, '{providerType}', '"GITLAB"')
WHERE config->'providerType' IS NULL
  AND config->>'subjectPattern' LIKE 'project\_path:%';
