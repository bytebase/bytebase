UPDATE instance SET metadata = metadata || options;
ALTER TABLE instance DROP COLUMN options;