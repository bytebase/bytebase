-- Rename the permission family in custom roles: bb.worksheets.get maps to the
-- discovery permission bb.savedQueries.search (reads now come from per-object
-- grants, so there is no get), list and manage rename in place.
UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}', (
    SELECT COALESCE(jsonb_agg(DISTINCT CASE p
        WHEN 'bb.worksheets.get' THEN 'bb.savedQueries.search'
        WHEN 'bb.worksheets.list' THEN 'bb.savedQueries.list'
        WHEN 'bb.worksheets.manage' THEN 'bb.savedQueries.manage'
        ELSE p
    END), '[]'::jsonb)
    FROM jsonb_array_elements_text(permissions->'permissions') AS p))
WHERE jsonb_typeof(permissions->'permissions') = 'array';

-- Creating and finding your own saved queries needed no permission before this
-- version (create and caller-scoped search carried no check), so the rename
-- alone would strip those basics from users whose only role is a custom role.
-- Grant every custom role the successors: create plus the caller-scoped
-- discovery gate. Predefined roles carry these in code.
UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',
    permissions->'permissions'
    || CASE WHEN permissions->'permissions' ? 'bb.savedQueries.create'
        THEN '[]'::jsonb ELSE '["bb.savedQueries.create"]'::jsonb END
    || CASE WHEN permissions->'permissions' ? 'bb.savedQueries.search'
        THEN '[]'::jsonb ELSE '["bb.savedQueries.search"]'::jsonb END)
WHERE jsonb_typeof(permissions->'permissions') = 'array';

-- A saved query lives in exactly one folder, set by its creator: the folder
-- becomes a path column on the object. The creator's own organizer placement
-- migrates; non-creator organizer rows are dropped (per-user re-foldering of
-- other people's queries is retired).
ALTER TABLE saved_query ADD COLUMN folder text NOT NULL DEFAULT '';

UPDATE saved_query
SET folder = COALESCE((
    SELECT string_agg(f.elem, '/' ORDER BY f.ord)
    FROM jsonb_array_elements_text(o.payload->'folders') WITH ORDINALITY AS f(elem, ord)
), '')
FROM saved_query_organizer o
WHERE o.saved_query = saved_query.resource_id
    AND o.principal = saved_query.creator;

-- A star is a per-user marker on a readable saved query: row existence is the
-- star. The FK cascade is a race backstop only — code paths delete star rows
-- explicitly before their parent.
CREATE TABLE saved_query_star (
    saved_query text NOT NULL REFERENCES saved_query(resource_id) ON DELETE CASCADE,
    principal text NOT NULL,
    PRIMARY KEY (saved_query, principal)
);

CREATE INDEX idx_saved_query_star_principal ON saved_query_star(principal);

-- Stars are per-principal, so every starred organizer row migrates -- unlike
-- the folder backfill above, which is creator-only because a row has exactly
-- one folder. The join drops organizer rows whose saved query is already gone.
INSERT INTO saved_query_star (saved_query, principal)
SELECT o.saved_query, o.principal
FROM saved_query_organizer o
    JOIN saved_query sq ON sq.resource_id = o.saved_query
WHERE COALESCE((o.payload->>'starred')::boolean, FALSE);

DROP TABLE saved_query_organizer;
