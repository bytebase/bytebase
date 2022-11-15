ALTER TABLE task ADD rollback_from INTEGER NULL REFERENCES task(id);
