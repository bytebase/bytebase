UPDATE project
SET setting = jsonb_set(
    jsonb_set(
        setting, 
        '{allowModifyStatement}', 
        'true', 
        true
    ),
    '{autoResolveIssue}', 
    'true', 
    true
) WHERE id != 1;
