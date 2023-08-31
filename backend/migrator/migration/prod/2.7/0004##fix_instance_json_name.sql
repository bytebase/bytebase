ALTER TABLE instance DISABLE TRIGGER update_instance_updated_ts;

UPDATE instance
SET metadata = (metadata || jsonb_build_object('mysqlLowerCaseTableNames', (metadata->>'mysql_lower_case_table_names')::int)) - 'mysql_lower_case_table_names'
WHERE metadata ? 'mysql_lower_case_table_names';

UPDATE instance
SET options = (options || jsonb_build_object('schemaTenantMode', (options->>'schema_tenant_mode')::boolean)) - 'schema_tenant_mode'
WHERE options ? 'schema_tenant_mode';

ALTER TABLE instance ENABLE TRIGGER update_instance_updated_ts;
