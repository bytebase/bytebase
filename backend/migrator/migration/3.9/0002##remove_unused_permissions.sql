-- Remove unused permissions from existing roles
UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',  
    (permissions->'permissions')::jsonb
    - 'bb.databases.adviseIndex'
    - 'bb.identityProviders.undelete'
    - 'bb.slowQueries.list'
);