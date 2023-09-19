ALTER TABLE issue ADD COLUMN ts_vector TSVECTOR;

CREATE INDEX idx_issue_ts_vector ON issue USING GIN(ts_vector);
