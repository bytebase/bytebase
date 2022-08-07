ALTER TABLE project DROP CONSTRAINT project_workflow_type_check;
ALTER TABLE project ADD CONSTRAINT project_workflow_type_check CHECK (workflow_type IN ('UI', 'VCS', 'DaC'));
