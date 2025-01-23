-- sheet_organizer table stores the sheet status for a principal.
CREATE TABLE sheet_organizer (
    id SERIAL PRIMARY KEY,
    sheet_id INTEGER NOT NULL REFERENCES sheet (id) ON DELETE CASCADE,
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    starred BOOLEAN NOT NULL DEFAULT false,
    pinned BOOLEAN NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX idx_sheet_organizer_unique_sheet_id_principal_id ON sheet_organizer(sheet_id, principal_id);

CREATE INDEX idx_sheet_organizer_principal_id ON sheet_organizer(principal_id);
