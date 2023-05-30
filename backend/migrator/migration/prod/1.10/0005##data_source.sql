ALTER TABLE data_source RENAME COLUMN host_override TO host;
ALTER TABLE data_source RENAME COLUMN port_override TO port;
ALTER TABLE data_source ADD COLUMN database TEXT NOT NULL DEFAULT '';

UPDATE data_source
SET
	host=(SELECT host FROM instance WHERE id = instance_id),
	port=(SELECT port FROM instance WHERE id = instance_id),
	database=(SELECT database FROM instance WHERE id = instance_id)
WHERE type = 'ADMIN';
