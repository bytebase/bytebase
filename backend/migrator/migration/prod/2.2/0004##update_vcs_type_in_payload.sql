UPDATE task SET task.payload = jsonb_set(task.payload, '{pushEvent,vcsType}', '"GITLAB"') WHERE task.payload ? 'pushEvent' AND task.payload->'pushEvent'->>'vcsType' = 'GITLAB_SELF_HOST';

UPDATE task SET task.payload = jsonb_set(task.payload, '{pushEvent,vcsType}', '"GITHUB"') WHERE task.payload ? 'pushEvent' AND task.payload->'pushEvent'->>'vcsType' = 'GITHUB_COM';

UPDATE instance_change_history SET instance_change_history.payload = jsonb_set(instance_change_history.payload, '{pushEvent,vcsType}', '"GITLAB"') WHERE task.payload ? 'pushEvent' AND task.payload->'pushEvent'->>'vcsType' = 'GITLAB_SELF_HOST';

UPDATE instance_change_history SET instance_change_history.payload = jsonb_set(instance_change_history.payload, '{pushEvent,vcsType}', '"GITHUB"') WHERE task.payload ? 'pushEvent' AND task.payload->'pushEvent'->>'vcsType' = 'GITHUB_COM';
