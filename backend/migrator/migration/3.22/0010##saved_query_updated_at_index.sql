-- The governance ListSavedQueries takes an AIP-132 order_by, and its primary
-- consumer is the cross-project per-user recency pull: creator filter +
-- "update_time desc". This index serves that filter and the (updated_at
-- DESC, resource_id DESC) order in one scan: no sort, and LIMIT stops early.
CREATE INDEX idx_saved_query_creator_updated_at_resource_id ON saved_query(creator, updated_at DESC, resource_id DESC);

-- The new index covers every creator-led scan (creator, project) served —
-- including the purge subqueries, which read columns outside it anyway — so
-- drop it rather than write three creator-led btrees on every update.
DROP INDEX IF EXISTS idx_saved_query_creator_project;
