-- Remove deprecated VCS permissions from existing roles
UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}',  
    (permissions->'permissions')::jsonb
    - 'bb.vcsConnectors.create'
    - 'bb.vcsConnectors.delete'
    - 'bb.vcsConnectors.get'
    - 'bb.vcsConnectors.list'
    - 'bb.vcsConnectors.update'
    - 'bb.vcsProviders.create'
    - 'bb.vcsProviders.delete'
    - 'bb.vcsProviders.get'
    - 'bb.vcsProviders.list'
    - 'bb.vcsProviders.listProjects'
    - 'bb.vcsProviders.searchProjects'
    - 'bb.vcsProviders.update'
);