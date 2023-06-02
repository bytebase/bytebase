UPDATE task SET payload = jsonb_set(payload, '{pushEvent,vcsType}', '"GITLAB"') WHERE payload ? 'pushEvent' AND payload->'pushEvent'->>'vcsType' = 'GITLAB_SELF_HOST';

UPDATE task SET payload = jsonb_set(payload, '{pushEvent,vcsType}', '"GITHUB"') WHERE payload ? 'pushEvent' AND payload->'pushEvent'->>'vcsType' = 'GITHUB_COM';

UPDATE instance_change_history SET payload = jsonb_set(payload, '{pushEvent,vcsType}', '"GITLAB"') WHERE payload ? 'pushEvent' AND payload->'pushEvent'->>'vcsType' = 'GITLAB_SELF_HOST';

UPDATE instance_change_history SET payload = jsonb_set(payload, '{pushEvent,vcsType}', '"GITHUB"') WHERE payload ? 'pushEvent' AND payload->'pushEvent'->>'vcsType' = 'GITHUB_COM';
