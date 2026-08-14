-- Per-object sharing: a saved query carries its own grants instead of a
-- project-wide visibility flag. Existing rows start with no bindings, which
-- makes them private to their creator -- the owner-private migration the
-- design calls for, matching BigQuery's own classic-to-modern path.
--
-- The column holds a protojson array of SavedQueryBinding at the jsonb root
-- (not a wrapper object), so the access queries' `@>` containment probes hit
-- the GIN index below without an expression index.
ALTER TABLE saved_query ADD COLUMN bindings jsonb NOT NULL DEFAULT '[]';

-- "Shared with me" is one GIN probe per principal in the caller's set
-- (themselves plus each of their groups), BitmapOr'd together. jsonb_path_ops
-- is smaller and faster than the default opclass for pure containment, which
-- is all these queries do.
CREATE INDEX idx_saved_query_bindings ON saved_query USING gin (bindings jsonb_path_ops);
