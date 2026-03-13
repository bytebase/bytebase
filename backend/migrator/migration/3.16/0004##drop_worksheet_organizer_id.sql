-- Drop unused id column from worksheet_organizer.
-- All queries use (worksheet_id, principal) which already has a unique index.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'worksheet_organizer' AND column_name = 'id'
    ) THEN
        ALTER TABLE worksheet_organizer DROP CONSTRAINT worksheet_organizer_pkey;
        ALTER TABLE worksheet_organizer DROP COLUMN id;
        ALTER TABLE worksheet_organizer ADD CONSTRAINT worksheet_organizer_pkey PRIMARY KEY USING INDEX idx_worksheet_organizer_unique_sheet_id_principal;
    END IF;
END $$;
