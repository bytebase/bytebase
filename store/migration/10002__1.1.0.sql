ALTER TABLE project DROP CONSTRAINT project_role_provider_check;
ALTER TABLE project ADD CONSTRAINT project_role_provider_check CHECK (role_provider IN ('BYTEBASE', 'GITLAB_SELF_HOST', 'GITHUB_COM'));

ALTER TABLE project_member DROP CONSTRAINT project_member_role_provider_check;
ALTER TABLE project_member ADD CONSTRAINT project_member_role_provider_check CHECK (role_provider IN ('BYTEBASE', 'GITLAB_SELF_HOST', 'GITHUB_COM'));

ALTER TABLE vcs DROP CONSTRAINT vcs_type_check;
ALTER TABLE vcs ADD CONSTRAINT vcs_type_check CHECK (type IN ('GITLAB_SELF_HOST', 'GITHUB_COM'));

ALTER TABLE sheet ADD source TEXT NOT NULL CHECK (source IN ('BYTEBASE', 'GITLAB_SELF_HOST', 'GITHUB_COM')) DEFAULT 'BYTEBASE';
ALTER TABLE sheet ADD type TEXT NOT NULL CHECK (type IN ('SQL')) DEFAULT 'SQL';
ALTER TABLE sheet ADD payload JSONB NOT NULL DEFAULT '{}';

ALTER TABLE repository ADD sheet_path_template TEXT NOT NULL DEFAULT '';
