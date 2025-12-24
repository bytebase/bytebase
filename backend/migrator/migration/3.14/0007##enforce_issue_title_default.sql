-- Enable enforce_issue_title for all existing projects.
UPDATE project
SET setting = jsonb_set(
    setting,
    '{enforceIssueTitle}',
    'true'::jsonb
);
