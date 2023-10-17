ALTER TABLE db DISABLE TRIGGER update_db_updated_ts;

UPDATE db
SET schema_version = '0000.0000.0000-' || schema_version
WHERE schema_version != '' AND schema_version NOT LIKE '0000.0000.0000-%';

ALTER TABLE db ENABLE TRIGGER update_db_updated_ts;
