-- Remove bb.rollouts.preview permission from existing roles
UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',
    (permissions->'permissions')::jsonb
    - 'bb.rollouts.preview'
);
