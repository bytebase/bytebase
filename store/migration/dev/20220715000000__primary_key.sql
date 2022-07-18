ALTER TABLE idx ADD COLUMN "primary" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE idx ADD CONSTRAINT check_primary_must_unique CHECK(NOT "primary" OR ("primary" AND "unique"));
ALTER TABLE idx ALTER COLUMN "primary" DROP DEFAULT;
