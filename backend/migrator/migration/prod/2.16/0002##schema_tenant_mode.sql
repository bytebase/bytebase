UPDATE instance
SET options = options - 'schemaTenantMode'
WHERE options ? 'schemaTenantMode';
