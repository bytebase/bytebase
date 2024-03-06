ALTER TABLE sheet DROP CONSTRAINT sheet_visibility_check;
ALTER TABLE sheet DROP COLUMN visibility;

ALTER TABLE sheet DROP CONSTRAINT sheet_type_check;
ALTER TABLE sheet DROP COLUMN type;

ALTER TABLE sheet DROP CONSTRAINT sheet_source_check;
