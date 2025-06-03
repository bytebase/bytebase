UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',  
    (permissions->'permissions')::jsonb
    - 'bb.branches.admin'
    - 'bb.branches.create'
    - 'bb.branches.delete'
    - 'bb.branches.get'
    - 'bb.branches.list'
    - 'bb.branches.update'
);
