ALTER TABLE principal DROP CONSTRAINT IF EXISTS idx_principal_unique_email;
ALTER TABLE principal ADD COLUMN idp_id INTEGER REFERENCES identity_provider (id);
ALTER TABLE principal ADD COLUMN idp_user_info JSONB;
ALTER TABLE principal ADD CONSTRAINT principal_idp_id_idp_user_info_check CHECK ((idp_id IS NULL AND idp_user_info IS NULL) OR (idp_id IS NOT NULL AND idp_user_info IS NOT NULL));
