UPDATE sheet SET payload = payload - 'vcsPayload' WHERE payload ? 'vcsPayload';
UPDATE instance_change_history SET payload = payload - 'pushEvent' WHERE payload ? 'pushEvent';
DELETE FROM activity where type = 'bb.project.repository.push';