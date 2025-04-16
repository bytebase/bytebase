UPDATE setting
SET value = REPLACE(
    REPLACE(
        REPLACE(
            REPLACE(
                REPLACE(value::text, 
                    '"groupValue"', '"role"'
                )::text,
                '"WORKSPACE_OWNER"', '"roles/workspaceAdmin"'
            )::text,
            '"WORKSPACE_DBA"', '"roles/workspaceDBA"'
        )::text,
        '"PROJECT_OWNER"', '"roles/projectOwner"'
    )::text,
    '"PROJECT_MEMBER"', '"roles/projectDeveloper"'
)::jsonb
WHERE name = 'bb.workspace.approval';

UPDATE issue
SET payload = REPLACE(
    REPLACE(
        REPLACE(
            REPLACE(
                REPLACE(payload::text, 
                    '"groupValue"', '"role"'
                )::text,
                '"WORKSPACE_OWNER"', '"roles/workspaceAdmin"'
            )::text,
            '"WORKSPACE_DBA"', '"roles/workspaceDBA"'
        )::text,
        '"PROJECT_OWNER"', '"roles/projectOwner"'
    )::text,
    '"PROJECT_MEMBER"', '"roles/projectDeveloper"'
)::jsonb
WHERE payload#>'{approval,approvalTemplates}' IS NOT NULL;
