-- Phase A: Drop FKs referencing worksheet(id)
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_worksheet_id_fkey;

-- Phase B: Change worksheet PK to composite (project, id); drop resource_id
DROP INDEX IF EXISTS idx_worksheet_unique_resource_id;
ALTER TABLE worksheet DROP COLUMN IF EXISTS resource_id;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'worksheet_pkey'
          AND conrelid = 'worksheet'::regclass
          AND array_length(conkey, 1) = 2
    ) THEN
        ALTER TABLE worksheet DROP CONSTRAINT IF EXISTS worksheet_pkey;
        ALTER TABLE worksheet ADD PRIMARY KEY (project, id);
    END IF;
END $$;

-- Phase C: Add project column to worksheet_organizer, backfill from worksheet
ALTER TABLE worksheet_organizer ADD COLUMN IF NOT EXISTS project text;
UPDATE worksheet_organizer SET project = worksheet.project FROM worksheet WHERE worksheet_organizer.project IS NULL AND worksheet_organizer.worksheet_id = worksheet.id;
DELETE FROM worksheet_organizer WHERE project IS NULL;
ALTER TABLE worksheet_organizer ALTER COLUMN project SET NOT NULL;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'worksheet_organizer_pkey'
          AND conrelid = 'worksheet_organizer'::regclass
          AND array_length(conkey, 1) = 3
    ) THEN
        ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_pkey;
        ALTER TABLE worksheet_organizer ADD PRIMARY KEY (project, worksheet_id, principal);
    END IF;
END $$;

-- Phase D: Re-add composite FK
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'worksheet_organizer_worksheet_id_fkey' AND conrelid = 'worksheet_organizer'::regclass) THEN
        ALTER TABLE worksheet_organizer ADD CONSTRAINT worksheet_organizer_worksheet_id_fkey
            FOREIGN KEY (project, worksheet_id) REFERENCES worksheet(project, id) ON DELETE CASCADE;
    END IF;
END $$;
