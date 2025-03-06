UPDATE instance SET metadata = metadata || jsonb_build_object('title', name);
UPDATE instance SET metadata = metadata || jsonb_build_object('engine', engine);
UPDATE instance SET metadata = metadata || jsonb_build_object('version', engine_version);
UPDATE instance SET metadata = metadata || jsonb_build_object('externalLink', external_link);
UPDATE instance SET metadata = metadata || jsonb_build_object('activation', activation);

ALTER TABLE instance DROP COLUMN name;
ALTER TABLE instance DROP COLUMN engine;
ALTER TABLE instance DROP COLUMN engine_version;
ALTER TABLE instance DROP COLUMN external_link;
ALTER TABLE instance DROP COLUMN activation;