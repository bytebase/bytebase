UPDATE project
SET setting = setting || '{"postgresDatabaseTenantMode": true}'
FROM db_group
WHERE EXISTS (
	SELECT 1 FROM db_group
	WHERE db_group.project_id = project.id AND db_group.payload->>'multitenancy' = 'true'
)