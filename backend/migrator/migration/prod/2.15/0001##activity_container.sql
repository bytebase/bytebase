ALTER TABLE activity ADD COLUMN resource_container TEXT;
DELETE FROM activity WHERE type = 'bb.database.recovery.pitr.done';