-- idp stores generic identity provider.
CREATE TABLE idp (
  id SERIAL PRIMARY KEY,
  row_status row_status NOT NULL DEFAULT 'NORMAL',
  creator_id INTEGER NOT NULL REFERENCES principal (id),
  created_ts BIGINT NOT NULL DEFAULT extract(
    epoch
    from
      now()
  ),
  updater_id INTEGER NOT NULL REFERENCES principal (id),
  updated_ts BIGINT NOT NULL DEFAULT extract(
    epoch
    from
      now()
  ),
  resource_id TEXT NOT NULL,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  type TEXT NOT NULL CONSTRAINT idp_type_check CHECK (type IN ('OAUTH2', 'OIDC')),
  -- config stores the corresponding configuration of the IdP, which may vary depending on the type of the IdP.
  config JSONB NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON idp(resource_id);

ALTER SEQUENCE idp_id_seq RESTART WITH 101;

CREATE TRIGGER update_idp_updated_ts BEFORE
UPDATE
  ON idp FOR EACH ROW EXECUTE FUNCTION trigger_update_updated_ts();

DROP INDEX IF EXISTS idx_principal_unique_email;

ALTER TABLE
  principal
ADD
  COLUMN idp_id INTEGER REFERENCES idp (id);

ALTER TABLE
  principal
ADD
  COLUMN idp_user_info JSONB NOT NULL DEFAULT '{}';