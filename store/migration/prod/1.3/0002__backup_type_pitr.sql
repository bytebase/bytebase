ALTER TABLE backup DROP CONSTRAINT backup_type_check;
ALTER TABLE backup ADD CONSTRAINT backup_type_check CHECK (type IN ('MANUAL', 'AUTOMATIC', 'PITR'));
