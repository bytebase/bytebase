UPDATE project
SET setting = jsonb_set(
    jsonb_set(
        setting, 
        '{allowModifyStatement}', 
        'true'
    ),
    '{autoResolveIssue}', 
    'true'
) WHERE id != 1;
