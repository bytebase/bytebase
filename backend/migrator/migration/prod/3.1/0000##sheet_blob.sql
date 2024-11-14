ALTER TABLE sheet ADD COLUMN IF NOT EXISTS sha256 BYTEA;

UPDATE sheet
SET sha256 = sha256(convert_to(sheet.statement, 'utf8'))
WHERE statement IS NOT NULL;

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
WHERE sheet.statement IS NOT NULL
ON CONFLICT DO NOTHING;

ALTER TABLE sheet DROP COLUMN IF EXISTS database_id;
ALTER TABLE sheet ALTER COLUMN statement DROP NOT NULL;
