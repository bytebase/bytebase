ALTER TABLE sheet ADD COLUMN IF NOT EXISTS sha256 BYTEA;

UPDATE sheet
SET sha256 = sha256(convert_to(sheet.statement, 'utf8'));

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
