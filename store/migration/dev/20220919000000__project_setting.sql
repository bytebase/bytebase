-- Project setting
CREATE TABLE project_setting (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    type TEXT NOT NULL CHECK (type LIKE 'bb.project.setting.%'),
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_project_setting_project_id ON project_setting(project_id);

CREATE UNIQUE INDEX idx_project_setting_unique_project_id_type ON project_setting(project_id, type);

ALTER SEQUENCE project_setting RESTART WITH 101;

CREATE TRIGGER update_project_setting_updated_ts
BEFORE
UPDATE
    ON project_setting FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

