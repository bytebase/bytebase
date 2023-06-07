WITH updates AS (
    SELECT id, jsonb_set(payload, '{pushEvent, commits}', jsonb_agg(
        CASE WHEN c.o ? 'ID' THEN
        jsonb_build_object('id', c.o->>'ID',
                          'title', c.o->>'Title',
                          'message', c.o->>'Message',
                          'createdTs', c.o->>'CreatedTs',
                          'url', c.o->>'URL',
                          'authorName', c.o->>'AuthorName',
                          'authorEmail', c.o->>'AuthorEmail',
                          'addedList', c.o->'AddedList',
                          'modifiedList', c.o->'ModifiedList')
        ELSE c.o
        END
        )) AS new_payload FROM instance_change_history
    LEFT JOIN jsonb_array_elements((payload->'pushEvent'->>'commits')::jsonb) WITH ORDINALITY c(o, position) ON TRUE
	WHERE payload->'pushEvent'->>'commits' IS NOT NULL AND jsonb_array_length(payload->'pushEvent'->'commits') > 0 AND payload->'pushEvent'->'commits'->0 ? 'ID'
	GROUP BY id
)
UPDATE instance_change_history
SET payload = updates.new_payload
FROM updates
WHERE instance_change_history.id = updates.id;

WITH updates AS (
    SELECT id, jsonb_set(payload, '{pushEvent, commits}', jsonb_agg(
        CASE WHEN c.o ? 'ID' THEN
        jsonb_build_object('id', c.o->>'ID',
                          'title', c.o->>'Title',
                          'message', c.o->>'Message',
                          'createdTs', c.o->>'CreatedTs',
                          'url', c.o->>'URL',
                          'authorName', c.o->>'AuthorName',
                          'authorEmail', c.o->>'AuthorEmail',
                          'addedList', c.o->'AddedList',
                          'modifiedList', c.o->'ModifiedList')
        ELSE c.o
        END
        )) AS new_payload FROM task
    LEFT JOIN jsonb_array_elements((payload->'pushEvent'->>'commits')::jsonb) WITH ORDINALITY c(o, position) ON TRUE
	WHERE payload->'pushEvent'->>'commits' IS NOT NULL AND jsonb_array_length(payload->'pushEvent'->'commits') > 0 AND payload->'pushEvent'->'commits'->0 ? 'ID'
	GROUP BY id
)
UPDATE task
SET payload = updates.new_payload
FROM updates
WHERE task.id = updates.id;

WITH updates AS (
    SELECT id, jsonb_set(payload, '{pushEvent, commits}', jsonb_agg(
        CASE WHEN c.o ? 'ID' THEN
        jsonb_build_object('id', c.o->>'ID',
                          'title', c.o->>'Title',
                          'message', c.o->>'Message',
                          'createdTs', c.o->>'CreatedTs',
                          'url', c.o->>'URL',
                          'authorName', c.o->>'AuthorName',
                          'authorEmail', c.o->>'AuthorEmail',
                          'addedList', c.o->'AddedList',
                          'modifiedList', c.o->'ModifiedList')
        ELSE c.o
        END
        )) AS new_payload FROM activity
    LEFT JOIN jsonb_array_elements((payload->'pushEvent'->>'commits')::jsonb) WITH ORDINALITY c(o, position) ON TRUE
	WHERE payload->'pushEvent'->>'commits' IS NOT NULL AND jsonb_array_length(payload->'pushEvent'->'commits') > 0 AND payload->'pushEvent'->'commits'->0 ? 'ID'
	GROUP BY id
)
UPDATE activity
SET payload = updates.new_payload
FROM updates
WHERE activity.id = updates.id;