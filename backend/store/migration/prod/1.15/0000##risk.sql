-- risk stores the definition of a risk.
CREATE TABLE risk (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    source TEXT NOT NULL CHECK (source LIKE 'bb.risk.%'),
    -- how risky is the risk, the higher the riskier
    level BIGINT NOT NULL,
    name TEXT NOT NULL,
    active BOOLEAN NOT NULL,
    expression JSONB NOT NULL
);

ALTER SEQUENCE risk_id_seq RESTART WITH 101;

CREATE TRIGGER update_risk_updated_ts
BEFORE
UPDATE
    ON risk FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
