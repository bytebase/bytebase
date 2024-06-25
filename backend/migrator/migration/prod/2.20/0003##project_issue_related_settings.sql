UPDATE project
SET setting = jsonb_set(
    jsonb_set(
        setting, 
        '{allow_modify_statement}', 
        'true', 
        true
    ),
    '{auto_resolve_issue}', 
    'true', 
    true
) WHERE id != 1;
