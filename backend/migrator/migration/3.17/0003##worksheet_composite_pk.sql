-- Phase A: Drop old FKs and PKs
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS sheet_organizer_sheet_id_fkey;
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_worksheet_id_fkey;
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_worksheet_fkey;
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_pkey;
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS sheet_organizer_pkey;
DROP INDEX IF EXISTS idx_worksheet_unique_resource_id;

-- Phase B: Change worksheet PK to resource_id
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'worksheet_pkey' AND conrelid = 'worksheet'::regclass
          AND array_length(conkey, 1) = 1
          AND conkey[1] = (SELECT attnum FROM pg_attribute WHERE attrelid = 'worksheet'::regclass AND attname = 'resource_id')
    ) THEN
        ALTER TABLE worksheet DROP CONSTRAINT IF EXISTS worksheet_pkey;
        ALTER TABLE worksheet ADD PRIMARY KEY (resource_id);
    END IF;
END $$;

-- Phase C: Convert worksheet_organizer to use worksheet.resource_id
ALTER TABLE worksheet_organizer ADD COLUMN IF NOT EXISTS worksheet text;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'worksheet_organizer' AND column_name = 'worksheet_id') THEN
        UPDATE worksheet_organizer wo SET worksheet = w.resource_id FROM worksheet w WHERE wo.worksheet_id = w.id AND wo.worksheet IS NULL;
        DELETE FROM worksheet_organizer WHERE worksheet IS NULL;
        ALTER TABLE worksheet_organizer DROP COLUMN worksheet_id;
    END IF;
END $$;
ALTER TABLE worksheet_organizer DROP COLUMN IF EXISTS project;
ALTER TABLE worksheet_organizer ALTER COLUMN worksheet SET NOT NULL;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'worksheet_organizer_pkey' AND conrelid = 'worksheet_organizer'::regclass) THEN
        ALTER TABLE worksheet_organizer ADD PRIMARY KEY (worksheet, principal);
    END IF;
END $$;

-- Phase D: Indexes and FK
CREATE INDEX IF NOT EXISTS idx_worksheet_project ON worksheet(project);
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'worksheet_organizer_worksheet_fkey' AND conrelid = 'worksheet_organizer'::regclass) THEN
        ALTER TABLE worksheet_organizer ADD CONSTRAINT worksheet_organizer_worksheet_fkey
            FOREIGN KEY (worksheet) REFERENCES worksheet(resource_id) ON DELETE CASCADE;
    END IF;
END $$;
