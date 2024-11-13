ALTER TABLE sheet ADD COLUMN IF NOT EXISTS sha256 BYTEA;

UPDATE sheet
SET sha256 = sha256(convert_to(sheet.statement, 'utf8'));

ALTER TABLE sheet ALTER COLUMN sha256 SET NOT NULL;

CREATE TABLE IF NOT EXISTS sheet_blob (
	sha256 BYTEA NOT NULL PRIMARY KEY,
	content TEXT NOT NULL
);

INSERT INTO sheet_blob (
	sha256,
	content
) SELECT
	sheet.sha256,
	sheet.statement
FROM sheet
ON CONFLICT DO NOTHING;

ALTER TABLE sheet DROP COLUMN IF EXISTS statement;
ALTER TABLE sheet DROP COLUMN IF EXISTS database_id;
