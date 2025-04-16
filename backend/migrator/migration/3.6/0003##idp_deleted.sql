DELETE FROM idp WHERE deleted = true;
ALTER TABLE idp DROP COLUMN deleted;
