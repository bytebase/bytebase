DELETE FROM sheet
WHERE sheet.payload->>'type'='SCHEMA_DESIGN';