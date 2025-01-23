ALTER TABLE repository ADD COLUMN enable_cd BOOLEAN NOT NULL DEFAULT false;
UPDATE repository SET enable_cd = true;