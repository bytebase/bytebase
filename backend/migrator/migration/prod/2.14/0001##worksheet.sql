-- Create worksheet table.
CREATE TABLE worksheet (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    database_id INTEGER NULL REFERENCES db (id),
    name TEXT NOT NULL,
    statement TEXT NOT NULL,
    visibility TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_worksheet_creator_id_project_id ON worksheet(creator_id, project_id);

CREATE TRIGGER update_worksheet_updated_ts
BEFORE
UPDATE
    ON worksheet FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Migrate sheet with BYTEBASE source to the worksheet table.
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

-- Change fk for sheet_organizer.
DELETE FROM sheet_organizer USING sheet WHERE sheet_organizer.sheet_id = sheet.id AND sheet.source != 'BYTEBASE';
ALTER TABLE sheet_organizer DROP CONSTRAINT sheet_organizer_sheet_id_fkey;
ALTER TABLE sheet_organizer ADD CONSTRAINT sheet_organizer_sheet_id_fkey FOREIGN KEY (sheet_id) REFERENCES worksheet (id);

-- Remove worksheet from sheet.
DELETE FROM sheet WHERE sheet.source = 'BYTEBASE';

-- Drop legacy fields for sheet.
ALTER TABLE sheet DROP CONSTRAINT sheet_visibility_check;
ALTER TABLE sheet DROP COLUMN visibility;

ALTER TABLE sheet DROP CONSTRAINT sheet_type_check;
ALTER TABLE sheet DROP COLUMN type;

ALTER TABLE sheet DROP CONSTRAINT sheet_source_check;
ALTER TABLE sheet DROP COLUMN source;

-- Reset workspace id sequence.
SELECT setval('worksheet_id_seq', GREATEST((SELECT MAX(id) FROM worksheet), 101));