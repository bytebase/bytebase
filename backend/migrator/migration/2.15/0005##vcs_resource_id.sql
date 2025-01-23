DROP INDEX idx_repository_unique_project_id;

ALTER TABLE vcs ADD COLUMN resource_id TEXT;
ALTER TABLE repository ADD COLUMN resource_id TEXT;

UPDATE vcs AS one SET resource_id = LOWER(type) WHERE id IN (SELECT id FROM vcs where one.type = vcs.type ORDER BY id LIMIT 1);
UPDATE vcs SET resource_id=CONCAT('vcs-', LEFT(uuid_in(md5(random()::text || random()::text)::cstring)::TEXT, 8)) WHERE resource_id IS NULL OR resource_id = '';
UPDATE repository SET resource_id='default';

ALTER TABLE vcs ALTER COLUMN resource_id SET NOT NULL;
ALTER TABLE repository ALTER COLUMN resource_id SET NOT NULL;

CREATE UNIQUE INDEX idx_vcs_unique_resource_id ON vcs(resource_id);
CREATE UNIQUE INDEX idx_repository_unique_project_id_resource_id ON repository(project_id, resource_id);
