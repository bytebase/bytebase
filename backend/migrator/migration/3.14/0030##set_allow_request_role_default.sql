-- Set allow_request_role to true for all existing projects
UPDATE project
SET setting = jsonb_set(
    setting,
    '{allowRequestRole}',
    'true'::jsonb
);
