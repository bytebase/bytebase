ALTER TABLE db ADD COLUMN environment_id INTEGER NULL REFERENCES environment (id);
ALTER TABLE instance ALTER COLUMN environment_id DROP NOT NULL;
