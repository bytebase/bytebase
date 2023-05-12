INSERT INTO project_member (row_status, creator_id, updater_id, project_id, role, principal_id, condition)
SELECT row_status, creator_id, updater_id, project_id, 'EXPORTER', principal_id, condition
FROM project_member
WHERE role = 'DEVELOPER';

INSERT INTO project_member (row_status, creator_id, updater_id, project_id, role, principal_id, condition)
SELECT row_status, creator_id, updater_id, project_id, 'QUERIER', principal_id, condition
FROM project_member
WHERE role = 'DEVELOPER';
