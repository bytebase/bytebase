-- Remove bb.risks.* permissions from existing roles
UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',
    (permissions->'permissions')::jsonb
    - 'bb.risks.create'
    - 'bb.risks.delete'
    - 'bb.risks.list'
    - 'bb.risks.update'
);
