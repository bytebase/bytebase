-- sheet_blob is content-addressed shared storage with no scope of its own, and
-- nothing has ever deleted from it: project purge does not touch it and no
-- foreign key points at it, so every surviving reference lives inside JSONB
-- payloads (plan.config, task.payload, release.payload, plan_check_run.result,
-- revision.payload), invisible to referential integrity. sheet_blob_ref makes
-- "which projects may read this hash" an explicit stored fact: it restores the
-- project scoping lost when the sheet table was dropped (#18552) and makes a
-- correct GC possible later. See docs/design/sheet-blob-scoping.md.
CREATE TABLE sheet_blob_ref (
    project text NOT NULL REFERENCES project(resource_id),
    sha256 bytea NOT NULL REFERENCES sheet_blob(sha256),
    PRIMARY KEY (project, sha256)
);

-- For the zero-ref verification audit and a future GC, which start from a
-- hash with no project in hand. Request-path queries always lead with
-- project and use the primary key.
CREATE INDEX idx_sheet_blob_ref_sha256 ON sheet_blob_ref(sha256);

-- Backfill one ref per (project, hash) pair derivable from the four
-- project-scoped reference sources. protojson camelCases keys:
-- sheet_sha256 -> sheetSha256. The EXISTS guard skips references naming a
-- hash with no blob so the foreign key stays satisfiable.
--
-- Known gap: a blob referenced by nothing (a sheet created but never attached
-- to a plan or release) has no derivable project, gets no ref, and becomes
-- unreadable. The alternative - leaving orphans globally readable -
-- reproduces the cross-project read this table closes.
-- The MATERIALIZED fence guarantees the hex-shape filter runs before any
-- decode(), so a malformed stored value is skipped instead of aborting the
-- migration. The jsonb_typeof guards do the same for the array expansions:
-- jsonb_array_elements errors on a non-array, and COALESCE alone would pass
-- a stored JSON null through, so anything that is not an array expands to
-- nothing instead.
WITH src AS MATERIALIZED (
    SELECT project, sha
    FROM (
        SELECT pl.project AS project, spec->'changeDatabaseConfig'->>'sheetSha256' AS sha
        FROM plan pl
        CROSS JOIN LATERAL jsonb_array_elements(
            CASE WHEN jsonb_typeof(pl.config->'specs') = 'array' THEN pl.config->'specs' ELSE '[]'::jsonb END) AS spec
        UNION ALL
        SELECT t.project, t.payload->>'sheetSha256'
        FROM task t
        UNION ALL
        SELECT r.project, f->>'sheetSha256'
        FROM release r
        CROSS JOIN LATERAL jsonb_array_elements(
            CASE WHEN jsonb_typeof(r.payload->'files') = 'array' THEN r.payload->'files' ELSE '[]'::jsonb END) AS f
        UNION ALL
        SELECT pcr.project, res->>'sheetSha256'
        FROM plan_check_run pcr
        CROSS JOIN LATERAL jsonb_array_elements(
            CASE WHEN jsonb_typeof(pcr.result->'results') = 'array' THEN pcr.result->'results' ELSE '[]'::jsonb END) AS res
    ) raw
    WHERE raw.sha ~ '^[0-9a-fA-F]{64}$'
)
INSERT INTO sheet_blob_ref (project, sha256)
SELECT DISTINCT src.project, decode(src.sha, 'hex')
FROM src
WHERE EXISTS (SELECT 1 FROM sheet_blob b WHERE b.sha256 = decode(src.sha, 'hex'))
ON CONFLICT DO NOTHING;

-- revision predates the payload.project stamp that new writers set at
-- creation, and it must NOT be backfilled from db.project: that is the
-- database's *current* project, so a database transferred before this
-- migration would grant its destination access to SQL the source authored -
-- an over-grant no audit could flag, because the ref row looks legitimate.
--
-- Derive the authoring project from the revision's own provenance instead
-- (payload.release, else payload.taskRun), and only when the named row
-- corroborates it: single-row identity, created no later than the revision,
-- and actually referencing this revision's hash. (project, release_id) is not
-- a declared unique key, so the release test requires exactly one matching
-- row; the task-run test matches the (project, id) primary key plus the task
-- ID in the name, and accepts the hash directly on the task or through the
-- task's release's files under the same exactly-one/age/hash test. The
-- temporal test defeats project-ID reuse after a purge; the hash test defeats
-- laundered provenance.
--
-- Corroborated rows get payload.project stamped - the same stored fact new
-- revisions carry - and a ref for that project. Revisions with absent or
-- uncorroborated provenance stay unstamped and get no ref: their statements
-- are unreadable and carry no sheet name until an operator grants access
-- deliberately, guided by the audits in
-- docs/design/sheet-blob-scoping.md#rollout.
--
-- The intermediate sets are session temp tables rather than CTEs, indexed and
-- ANALYZEd before use: CTE materializations carry no statistics and no
-- indexes, which at deployment scale sent the planner into nested loops over
-- hundreds of thousands of rows (measured: minutes-to-hours as CTEs, seconds
-- as temp tables). ON COMMIT DROP keeps them inside the migration
-- transaction, so atomicity and retry-on-failure are unchanged.
-- Hex hashes are compared as text below, and legacy payloads may carry
-- uppercase hex (old callers could name sheets in either case), so every
-- extracted hash is lowered to the canonical encode() form first.
CREATE TEMP TABLE _sheet_scope_rev ON COMMIT DROP AS
SELECT r.resource_id,
       r.created_at,
       lower(r.payload->>'sheetSha256') AS sha,
       (regexp_match(r.payload->>'release', 'projects/([^/]+)/'))[1] AS rel_project,
       (regexp_match(r.payload->>'release', 'releases/([^/]+)'))[1] AS rel_id,
       -- The digit bound keeps the bigint casts total: a malformed name
       -- with a >18-digit run fails the match and leaves the row
       -- uncorroborated instead of aborting the migration on overflow.
       (regexp_match(r.payload->>'taskRun', 'projects/([^/]+)/'))[1] AS tr_project,
       ((regexp_match(r.payload->>'taskRun', 'taskRuns/(\d{1,18})$'))[1])::bigint AS tr_id,
       ((regexp_match(r.payload->>'taskRun', 'tasks/(\d{1,18})/'))[1])::bigint AS task_id
FROM revision r
WHERE r.payload->>'sheetSha256' ~ '^[0-9a-fA-F]{64}$';

CREATE INDEX ON _sheet_scope_rev (rel_project, rel_id, sha);
CREATE INDEX ON _sheet_scope_rev (tr_project, tr_id, task_id);
ANALYZE _sheet_scope_rev;

-- Files of releases whose (project, release_id) matches exactly one row.
-- (project, release_id) is not a declared unique key, so provenance that
-- does not identify a single release has not identified anything. Both
-- corroboration branches test against this one statement of the rule:
-- single-row identity, release no newer than the revision, file carrying
-- the revision's hash.
CREATE TEMP TABLE _sheet_scope_srf ON COMMIT DROP AS
SELECT rel.project, rel.release_id, rel.created_at, lower(f->>'sheetSha256') AS sha
FROM release rel
JOIN (
    SELECT project, release_id
    FROM release
    GROUP BY project, release_id
    HAVING count(*) = 1
) single USING (project, release_id)
CROSS JOIN LATERAL jsonb_array_elements(
    CASE WHEN jsonb_typeof(rel.payload->'files') = 'array' THEN rel.payload->'files' ELSE '[]'::jsonb END) AS f;

CREATE INDEX ON _sheet_scope_srf (project, release_id, sha);
ANALYZE _sheet_scope_srf;

-- Release branch first: it is the preferred provenance.
CREATE TEMP TABLE _sheet_scope_corroborated ON COMMIT DROP AS
SELECT DISTINCT rev.resource_id, rev.rel_project AS project, rev.sha
FROM _sheet_scope_rev rev
JOIN _sheet_scope_srf srf
    ON srf.project = rev.rel_project
    AND srf.release_id = rev.rel_id
    AND srf.created_at <= rev.created_at
    AND srf.sha = rev.sha;

CREATE INDEX ON _sheet_scope_corroborated (resource_id);
ANALYZE _sheet_scope_corroborated;

-- Task-run branch for the rest. The statement's snapshot sees only the
-- release-branch rows inserted above, so NOT EXISTS implements the
-- release-over-task-run preference. task_run and task are probed by primary
-- key; the LEFT JOIN can multiply rows (several files may carry the same
-- hash), and DISTINCT collapses them - the projected columns are fixed per
-- revision, so at most one corroborated row per revision survives.
INSERT INTO _sheet_scope_corroborated
SELECT DISTINCT rev.resource_id, rev.tr_project AS project, rev.sha
FROM _sheet_scope_rev rev
JOIN task_run tr
    ON tr.project = rev.tr_project
    AND tr.id = rev.tr_id
    AND tr.task_id = rev.task_id
    AND tr.created_at <= rev.created_at
JOIN task t ON t.project = tr.project AND t.id = tr.task_id
LEFT JOIN _sheet_scope_srf srf
    ON srf.project = (regexp_match(t.payload->>'release', 'projects/([^/]+)/'))[1]
    AND srf.release_id = (regexp_match(t.payload->>'release', 'releases/([^/]+)'))[1]
    AND srf.created_at <= rev.created_at
    AND srf.sha = rev.sha
WHERE rev.tr_project IS NOT NULL AND rev.tr_id IS NOT NULL AND rev.task_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM _sheet_scope_corroborated c WHERE c.resource_id = rev.resource_id)
  AND (lower(t.payload->>'sheetSha256') = rev.sha OR srf.project IS NOT NULL);

ANALYZE _sheet_scope_corroborated;

UPDATE revision r
SET payload = r.payload || jsonb_build_object('project', c.project)
FROM _sheet_scope_corroborated c
WHERE r.resource_id = c.resource_id;

-- Grant each stamped revision's authoring project read access to its hash.
-- The MATERIALIZED fence keeps decode() behind the hex-shape filter. This
-- reads payload.project back rather than reusing the corroborated set, so it
-- also sees any pre-existing project key; the project EXISTS guard keeps a
-- value naming no live project row from aborting the migration on the
-- foreign key (corroborated stamps always pass it - they came from FK-backed
-- release/task_run rows).
WITH stamped AS MATERIALIZED (
    SELECT r.payload->>'project' AS project, lower(r.payload->>'sheetSha256') AS sha
    FROM revision r
    WHERE r.payload->>'project' IS NOT NULL
      AND r.payload->>'sheetSha256' ~ '^[0-9a-fA-F]{64}$'
)
INSERT INTO sheet_blob_ref (project, sha256)
SELECT DISTINCT stamped.project, decode(stamped.sha, 'hex')
FROM stamped
WHERE EXISTS (SELECT 1 FROM sheet_blob b WHERE b.sha256 = decode(stamped.sha, 'hex'))
  AND EXISTS (SELECT 1 FROM project p WHERE p.resource_id = stamped.project)
ON CONFLICT DO NOTHING;

-- Post-upgrade verification and review queries for operators - the zero-ref
-- count, the multi-project hash review, the source-level missing-ref audit,
-- and the ambiguous-provenance revision list - are maintained in
-- docs/design/sheet-blob-scoping.md#rollout.
