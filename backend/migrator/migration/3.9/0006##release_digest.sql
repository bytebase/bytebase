ALTER TABLE release ADD COLUMN digest TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX idx_release_unique_project_digest ON release(project, digest) WHERE digest != '';
