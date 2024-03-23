ALTER TABLE activity ADD COLUMN resource_container TEXT;
CREATE INDEX idx_activity_resource_container ON activity(resource_container);

DELETE FROM activity WHERE type = 'bb.database.recovery.pitr.done';