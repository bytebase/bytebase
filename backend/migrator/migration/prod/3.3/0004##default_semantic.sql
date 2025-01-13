INSERT INTO setting (creator_id, updater_id, name, value)
VALUES (1, 1, 'bb.workspace.semantic-types', '{}')
ON CONFLICT (name) DO NOTHING;

UPDATE setting
SET value = jsonb_build_object('types', '[{"id":"bb.default","title":"Default","description":"Default type with full masking"},{"id":"bb.default-partial","title":"Default Partial","description":"Default partial type with partial masking"}]'::jsonb || coalesce(value::jsonb->'types', '[]'::jsonb))
WHERE name = 'bb.workspace.semantic-types';