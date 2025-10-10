-- Drop the unnecessary unique index on (project, placeholder) before renaming the column
-- The unique constraint on (project, resource_id) is sufficient
DROP INDEX idx_db_group_unique_project_placeholder;

-- Rename db_group.placeholder to db_group.name for consistency with other tables
ALTER TABLE db_group RENAME COLUMN placeholder TO name;
