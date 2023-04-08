ALTER TABLE project DROP COLUMN role_provider;

DELETE FROM project_member WHERE role_provider != 'BYTEBASE';
DROP INDEX idx_project_member_unique_project_id_role_provider_principal_id;
ALTER TABLE project_member DROP COLUMN role_provider;
CREATE UNIQUE INDEX idx_project_member_unique_project_id_principal_id ON project_member(project_id, principal_id);
