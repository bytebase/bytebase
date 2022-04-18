ALTER TABLE sheet ADD source TEXT NOT NULL CHECK (source IN ('BYTEBASE', 'GITLAB_SELF_HOST', 'GITHUB_COM')) DEFAULT 'BYTEBASE';
ALTER TABLE sheet ADD type TEXT NOT NULL CHECK (type IN ('SQL')) DEFAULT 'SQL';
ALTER TABLE sheet ADD payload JSONB NOT NULL DEFAULT '{}';

CREATE INDEX idx_sheet_project_id ON sheet(project_id);

CREATE INDEX idx_sheet_name ON sheet(name);

ALTER TABLE repository ADD sheet_path_template TEXT NOT NULL DEFAULT '';

CREATE OR REPLACE TRIGGER update_sheet_updated_ts
BEFORE
UPDATE OF project_id, database_id, name, statement, visibility, source, type, payload
    ON sheet FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
