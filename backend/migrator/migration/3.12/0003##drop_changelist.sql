-- Remove changelist permissions from existing roles
UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',
    (permissions->'permissions')::jsonb
    - 'bb.changelists.create'
    - 'bb.changelists.delete'
    - 'bb.changelists.get'
    - 'bb.changelists.list'
    - 'bb.changelists.update'
);

-- Drop the changelist table and related objects.
DROP INDEX IF EXISTS idx_changelist_project_name;

DROP TABLE IF EXISTS changelist;

DROP SEQUENCE IF EXISTS changelist_id_seq;
