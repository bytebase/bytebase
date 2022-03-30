ALTER TABLE project DROP CONSTRAINT project_role_provider_check;
ALTER TABLE project ADD CONSTRAINT project_role_provider_check CHECK (role_provider IN ('BYTEBASE', 'GITLAB_SELF_HOST', 'GITHUB_COM'));

ALTER TABLE project_member DROP CONSTRAINT project_member_role_provider_check;
ALTER TABLE project_member ADD CONSTRAINT project_member_role_provider_check CHECK (role_provider IN ('BYTEBASE', 'GITLAB_SELF_HOST', 'GITHUB_COM'));

ALTER TABLE vcs DROP CONSTRAINT vcs_type_check;
ALTER TABLE vcs ADD CONSTRAINT vcs_type_check CHECK (type IN ('GITLAB_SELF_HOST', 'GITHUB_COM'));

ALTER TABLE sheet ADD source TEXT NOT NULL CHECK (source IN ('BYTEBASE', 'GITLAB_SELF_HOST', 'GITHUB_COM')) DEFAULT 'BYTEBASE';
ALTER TABLE sheet ADD type TEXT NOT NULL CHECK (type IN ('SQL')) DEFAULT 'SQL';
ALTER TABLE sheet ADD payload JSONB NOT NULL DEFAULT '{}';

CREATE INDEX idx_sheet_project_id ON sheet(project_id);

CREATE INDEX idx_sheet_name ON sheet(name);

ALTER TABLE repository ADD sheet_path_template TEXT NOT NULL DEFAULT '';

-- task_dag describes task dependency relationship
-- from_task_id is blocked by to_task_id
CREATE TABLE task_dag (
    id SERIAL PRIMARY KEY,
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    from_task_id INTEGER NOT NULL REFERENCES task (id),
    to_task_id INTEGER NOT NULL REFERENCES task (id),
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_task_dag_from_task_id ON task_dag(from_task_id);

CREATE INDEX idx_task_dag_to_task_id ON task_dag(to_task_id);

ALTER SEQUENCE task_dag_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_dag_updated_ts
BEFORE
UPDATE
    ON task_dag FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
