UPDATE policy
SET payload = replace(
    payload::text,
    'roles/projectQuerier',
    'roles/sqlEditorUser'
)::jsonb
WHERE type = 'bb.policy.iam';

UPDATE issue
SET payload = replace(
    payload::text,
    'roles/projectQuerier',
    'roles/sqlEditorUser'
)::jsonb
WHERE type = 'bb.issue.grant.request';

UPDATE role
SET permissions = replace(
    replace(
        replace(
            replace(
                replace(
                    replace(
                        replace(
                            permissions::text,
                            'bb.databases.queryDDL',
                            'bb.sql.ddl'
                        ),
                        'bb.databases.queryDML',
                        'bb.sql.dml'
                    ),
                    'bb.databases.queryInfo',
                    'bb.sql.info'
                ),
                'bb.databases.queryExplain',
                'bb.sql.explain'
            ),
            'bb.databases.query',
            'bb.sql.select'
        ),
        'bb.instances.admin',
        'bb.sql.admin'
    ),
    'bb.databases.export',
    'bb.sql.export'
)::jsonb;