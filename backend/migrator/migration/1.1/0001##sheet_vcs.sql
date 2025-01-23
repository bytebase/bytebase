ALTER TABLE sheet ADD source TEXT NOT NULL CHECK (source IN ('BYTEBASE', 'GITLAB_SELF_HOST', 'GITHUB_COM')) DEFAULT 'BYTEBASE';
ALTER TABLE sheet ADD type TEXT NOT NULL CHECK (type IN ('SQL')) DEFAULT 'SQL';
ALTER TABLE sheet ADD payload JSONB NOT NULL DEFAULT '{}';

CREATE INDEX idx_sheet_project_id ON sheet(project_id);

CREATE INDEX idx_sheet_name ON sheet(name);

ALTER TABLE repository ADD sheet_path_template TEXT NOT NULL DEFAULT '';
