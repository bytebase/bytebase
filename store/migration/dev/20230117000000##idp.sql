-- identity_provider stores generic identity provider.
CREATE TABLE identity_provider (
  id SERIAL PRIMARY KEY,
  row_status row_status NOT NULL DEFAULT 'NORMAL',
  creator_id INTEGER NOT NULL REFERENCES principal (id),
  created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  updater_id INTEGER NOT NULL REFERENCES principal (id),
  updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  resource_id TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL CONSTRAINT identity_provider_type_check CHECK (type IN ('OAUTH2', 'OIDC')),
  -- config stores the corresponding configuration of the IdP, which may vary depending on the type of the IdP.
  config JSONB NOT NULL
);

CREATE UNIQUE INDEX idx_identity_provider_unique_resource_id ON identity_provider(resource_id);

CREATE TRIGGER update_identity_provider_updated_ts
BEFORE
UPDATE
    ON identity_provider FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
