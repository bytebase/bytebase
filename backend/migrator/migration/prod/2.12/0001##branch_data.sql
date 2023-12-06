INSERT INTO branch (creator_id, created_ts, updater_id, updated_ts, project_id, name, engine, base, head, config)
SELECT
	sheet.creator_id,
	sheet.created_ts,
	sheet.updater_id,
	sheet.updated_ts,
	sheet.project_id,	
	sheet.name,
	sheet.payload->'schemaDesign'->>'engine',
	(SELECT jsonb_build_object('schema', replace(encode(ps.statement::bytea, 'base64'), E'\n', '')) FROM sheet AS ps WHERE ps.id = (sheet.payload->'schemaDesign'->>'baselineSheetId')::int LIMIT 1),
	jsonb_build_object('schema', replace(encode(sheet.statement::bytea, 'base64'), E'\n', '')),
	jsonb_build_object('sourceDatabase', (SELECT 'instances/' || instance.resource_id || '/databases/' || db.name FROM db JOIN instance ON instance.id = db.instance_id WHERE db.id = sheet.database_id LIMIT 1))
FROM sheet
WHERE sheet.payload->>'type'='SCHEMA_DESIGN';