DROP INDEX idx_project_member_unique_project_id_principal_id_role;
CREATE INDEX idx_project_member_project_id ON project_member(project_id);

ALTER TABLE project_member RENAME COLUMN payload TO condition;