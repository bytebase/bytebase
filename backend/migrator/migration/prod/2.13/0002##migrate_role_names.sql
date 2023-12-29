UPDATE member
SET role = CASE role
    WHEN 'OWNER' THEN 'workspaceAdmin'
    WHEN 'DBA' THEN 'workspaceDBA'
    WHEN 'DEVELOPER' THEN 'workspaceDeveloper'
    ELSE role
END;


UPDATE project_member
SET role = CASE role
    WHEN 'OWNER' THEN 'projectOwner'
    WHEN 'DEVELOPER' THEN 'projectDeveloper'
    WHEN 'RELEASER' THEN 'projectReleaser'
    WHEN 'VIEWER' THEN 'projectViewer'
    WHEN 'QUERIER' THEN 'projectQuerier'
    WHEN 'EXPORTER' THEN 'projectExporter'
    ELSE role
END;


UPDATE policy
SET payload = replace(
    (
    replace(
        payload::text,
        'roles/EXPORTER',
        'roles/projectExporter'
    ),
    'roles/QUERIER',
    'roles/projectQuerier'
)::jsonb
WHERE type = 'bb.policy.workspace-iam';

UPDATE policy
SET payload = payload || jsonb_build_object(
    'projectRoles',
    replace (
        replace(
            (payload->>'projectRoles')::text,
            'roles/RELEASER',
            'roles/projectReleaser'
        ),
        'roles/OWNER',
        'roles/projectOwner'
    )
) || jsonb_build_object(
    'workspaceRoles',
    replace(
        replace(
            (payload->>'workspaceRoles')::text,
            'roles/OWNER',
            'roles/workspaceAdmin'
        ),
        'roles/DBA',
        'roles/workspaceDBA'
    )
)
WHERE type = 'bb.policy.rollout';

UPDATE issue
SET payload = replace(
    replace( 
        payload::text,
        'roles/EXPORTER',
        'roles/projectExporter'
    ),
    'roles/QUERIER',
    'roles/projectQuerier'
)::jsonb
WHERE type = 'bb.issue.grant.request';

UPDATE project
SET setting = 
replace(
    replace(
        replace(
            replace(
                replace (
                    replace(
                        setting::text,
                        'roles/OWNER',
                        'roles/projectOwner'
                    ),
                    'roles/EXPORTER',
                    'roles/projectExporter'
                ),
                'roles/DEVELOPER',
                'roles/projectDeveloper'
            ),
            'roles/VIEWER',
            'roles/projectViewer'
        ),
        'roles/RELEASER',
        'roles/projectReleaser'
    ),
    'roles/QUERIER',
    'roles/projectQuerier'
)::jsonb;