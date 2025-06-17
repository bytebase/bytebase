-- Remove outboundIpList from workspace profile setting
UPDATE setting 
SET value = (value::jsonb - 'outboundIpList')::text
WHERE name = 'WORKSPACE_PROFILE' 
  AND value::jsonb ? 'outboundIpList';