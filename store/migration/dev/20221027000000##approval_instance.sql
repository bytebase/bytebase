-- external_approval_instance stores approval instances of third party applications.
CREATE TABLE external_approval_instance ( 
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    issue_id INTEGER NOT NULL REFERENCES issue (id),
    requester_id INTEGER NOT NULL REFERENCES principal (id),
    approver_id INTEGER NOT NULL REFERENCES principal (id),
    type TEXT NOT NULL CHECK (type LIKE 'bb.plugin.app.%'),
    payload JSONB NOT NULL
);

CREATE INDEX idx_external_approval_instance_issue_id_row_status ON external_approval_instance(issue_id, row_status); 

CREATE INDEX idx_external_approval_instance_row_status ON external_approval_instance(row_status);

ALTER SEQUENCE external_approval_instance_id_seq RESTART WITH 101;

CREATE TRIGGER update_external_approval_instance_updated_ts
BEFORE
UPDATE
    ON external_approval_instance FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
