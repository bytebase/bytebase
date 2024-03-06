INSERT INTO worksheet
    (id, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, payload)
SELECT
    sheet.id,
    sheet.creator_id,
    sheet.created_ts,
    sheet.updater_id,
    sheet.updated_ts,
    sheet.project_id,
    sheet.database_id,
    sheet.name,
    sheet.statement,
    sheet.visibility,
    sheet.payload
FROM sheet
WHERE sheet.source = 'BYTEBASE';