-- Remove deprecated database secret permissions from existing roles
UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',  
    (permissions->'permissions')::jsonb
    - 'bb.databaseSecrets.delete'
    - 'bb.databaseSecrets.list'
    - 'bb.databaseSecrets.update'
);