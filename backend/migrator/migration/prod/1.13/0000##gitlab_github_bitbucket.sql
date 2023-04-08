ALTER TABLE vcs DROP CONSTRAINT vcs_type_check;
ALTER TABLE sheet DROP CONSTRAINT sheet_source_check;

UPDATE vcs SET type = 'GITLAB' WHERE type = 'GITLAB_SELF_HOST';
UPDATE vcs SET type = 'GITHUB' WHERE type = 'GITHUB_COM';
UPDATE sheet SET source = 'GITLAB' WHERE source = 'GITLAB_SELF_HOST';
UPDATE sheet SET source = 'GITHUB' WHERE source = 'GITHUB_COM';

ALTER TABLE vcs ADD CONSTRAINT vcs_type_check CHECK (type IN ('GITLAB', 'GITHUB', 'BITBUCKET'));
ALTER TABLE sheet ADD CONSTRAINT sheet_source_check CHECK (source IN ('BYTEBASE', 'GITLAB', 'GITHUB', 'BITBUCKET'));