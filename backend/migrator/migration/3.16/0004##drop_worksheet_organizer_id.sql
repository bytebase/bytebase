-- Drop unused id column and legacy PK if exists from worksheet_organizer.
-- All queries use (worksheet_id, principal) which already has a unique index.
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS sheet_organizer_pkey;
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_pkey;
ALTER TABLE worksheet_organizer DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_worksheet_organizer_unique_sheet_id_principal;
ALTER TABLE worksheet_organizer ADD PRIMARY KEY (worksheet_id, principal);
