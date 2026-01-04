-- Consolidate DATABASE_SDL into DATABASE_MIGRATE
-- IMPORTANT: This migration runs as part of the task type consolidation.
-- The execution strategy will be determined by release.payload.type (VERSIONED vs DECLARATIVE)
-- for release-based tasks, or defaulted to imperative execution for sheet-based tasks.

-- Convert all DATABASE_SDL tasks to DATABASE_MIGRATE
UPDATE task
SET type = 'DATABASE_MIGRATE'
WHERE type = 'DATABASE_SDL';
