UPDATE instance
SET options = (options || jsonb_build_object('schemaTenantMode', (options->>'schema_tenant_mode')::boolean)) - 'schema_tenant_mode'
WHERE options ? 'schema_tenant_mode';
