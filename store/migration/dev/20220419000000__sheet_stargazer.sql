-- sheet_stargazer table stores the sheet stargazers.
CREATE TABLE sheet_stargazer (
    sheet_id INTEGER NOT NULL REFERENCES sheet (id),
    stargazer_id INTEGER NOT NULL REFERENCES principal (id),
    PRIMARY KEY (sheet_id, stargazer_id)
);

CREATE INDEX idx_sheet_stargazer_stargazer_id ON sheet_stargazer(stargazer_id);
