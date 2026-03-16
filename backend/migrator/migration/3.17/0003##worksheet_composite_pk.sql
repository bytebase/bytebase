-- Phase A: Drop FKs referencing worksheet(id)
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_worksheet_id_fkey;

-- Phase B: Change worksheet PK to resource_id
ALTER TABLE worksheet DROP CONSTRAINT IF EXISTS worksheet_pkey;
ALTER TABLE worksheet ADD PRIMARY KEY (resource_id);

-- Phase C: Convert worksheet_organizer.worksheet_id from integer (worksheet.id) to text (worksheet.resource_id)
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_pkey;

ALTER TABLE worksheet_organizer ADD COLUMN worksheet text;
UPDATE worksheet_organizer wo
    SET worksheet = w.resource_id
    FROM worksheet w
    WHERE wo.worksheet_id = w.id;
DELETE FROM worksheet_organizer WHERE worksheet IS NULL;
ALTER TABLE worksheet_organizer DROP COLUMN worksheet_id;
ALTER TABLE worksheet_organizer ALTER COLUMN worksheet SET NOT NULL;

ALTER TABLE worksheet_organizer ADD PRIMARY KEY (worksheet, principal);

-- Phase D: Add project index for list queries
CREATE INDEX IF NOT EXISTS idx_worksheet_project ON worksheet(project);

-- Phase E: Re-add FK referencing worksheet(resource_id)
ALTER TABLE worksheet_organizer ADD CONSTRAINT worksheet_organizer_worksheet_fkey
    FOREIGN KEY (worksheet) REFERENCES worksheet(resource_id) ON DELETE CASCADE;
