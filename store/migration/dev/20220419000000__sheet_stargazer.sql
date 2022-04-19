-- sheet_stargazer table stores the sheet stargazers.
CREATE TABLE sheet_stargazer (
    id SERIAL PRIMARY KEY,
    sheet_id INTEGER NOT NULL REFERENCES sheet (id),
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    status TEXT NOT NULL CHECK (status IN ('STARRED', 'UNSTARRED')) DEFAULT 'UNSTARRED'
);

CREATE UNIQUE INDEX idx_sheet_stargazer_unique_sheet_id_principal_id ON sheet_stargazer(sheet_id, principal_id);
