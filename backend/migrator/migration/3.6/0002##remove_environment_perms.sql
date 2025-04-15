UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',  
    (permissions->'permissions')::jsonb
    - 'bb.environments.get'
    - 'bb.environments.list'
    - 'bb.environments.create'
    - 'bb.environments.delete'
    - 'bb.environments.undelete'
    - 'bb.environments.update'
);
