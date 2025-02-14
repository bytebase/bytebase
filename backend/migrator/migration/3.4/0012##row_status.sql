ALTER TABLE plan DROP COLUMN IF EXISTS row_status;
ALTER TABLE issue DROP COLUMN IF EXISTS row_status;

ALTER TABLE idp ADD COLUMN IF NOT EXISTS deleted boolean NOT NULL DEFAULT FALSE;
UPDATE idp SET deleted = TRUE WHERE row_status = 'ARCHIVED';
ALTER TABLE idp DROP COLUMN IF EXISTS row_status;

ALTER TABLE principal ADD COLUMN IF NOT EXISTS deleted boolean NOT NULL DEFAULT FALSE;
UPDATE principal SET deleted = TRUE WHERE row_status = 'ARCHIVED';
ALTER TABLE principal DROP COLUMN IF EXISTS row_status;

ALTER TABLE environment ADD COLUMN IF NOT EXISTS deleted boolean NOT NULL DEFAULT FALSE;
UPDATE environment SET deleted = TRUE WHERE row_status = 'ARCHIVED';
ALTER TABLE environment DROP COLUMN IF EXISTS row_status;

ALTER TABLE project ADD COLUMN IF NOT EXISTS deleted boolean NOT NULL DEFAULT FALSE;
UPDATE project SET deleted = TRUE WHERE row_status = 'ARCHIVED';
ALTER TABLE project DROP COLUMN IF EXISTS row_status;

ALTER TABLE instance ADD COLUMN IF NOT EXISTS deleted boolean NOT NULL DEFAULT FALSE;
UPDATE instance SET deleted = TRUE WHERE row_status = 'ARCHIVED';
ALTER TABLE instance DROP COLUMN IF EXISTS row_status;

ALTER TABLE release ADD COLUMN IF NOT EXISTS deleted boolean NOT NULL DEFAULT FALSE;
UPDATE release SET deleted = TRUE WHERE row_status = 'ARCHIVED';
ALTER TABLE release DROP COLUMN IF EXISTS row_status;

ALTER TABLE policy ADD COLUMN IF NOT EXISTS enforce boolean NOT NULL DEFAULT TRUE;
UPDATE policy SET enforce = FALSE WHERE row_status = 'ARCHIVED';
ALTER TABLE policy DROP COLUMN IF EXISTS row_status;

ALTER TABLE review_config ADD COLUMN IF NOT EXISTS enabled boolean NOT NULL DEFAULT TRUE;
UPDATE review_config SET enabled = FALSE WHERE row_status = 'ARCHIVED';
ALTER TABLE review_config DROP COLUMN IF EXISTS row_status;

DROP TYPE row_status;
