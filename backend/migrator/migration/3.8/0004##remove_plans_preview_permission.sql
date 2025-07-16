UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',  
    (permissions->'permissions')::jsonb
    - 'bb.plans.preview'
);
