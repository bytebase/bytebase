CREATE TABLE approval_instance ( 
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    issue_id INTEGER NOT NULL REFERENCES issue (id),
    requester_id INTEGER NOT NULL REFERENCES principal (id),
    approver_id INTEGER NOT NULL REFERENCES principal (id),
    type TEXT NOT NULL CHECK (type LIKE 'bb.application.approval-instance.%'),
    payload JSONB NOT NULL
);

CREATE INDEX idx_approval_instance_issue_id ON approval_instance(issue_id); 

ALTER SEQUENCE approval_instance_seq RESTART WITH 101;

CREATE TRIGGER update_sheet_updated_ts
BEFORE
UPDATE
    ON sheet FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
