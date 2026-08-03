ALTER TABLE instance ADD COLUMN project text REFERENCES project(resource_id);

CREATE INDEX idx_instance_project ON instance(project) WHERE project IS NOT NULL;
