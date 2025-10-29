-- Add UNIQUE constraint on audit_log bytebase_id and sequence_number
-- This prevents duplicate sequences within a Bytebase deployment
-- NULL values are allowed for legacy logs created before this feature

CREATE UNIQUE INDEX idx_audit_log_unique_bytebase_id_seq ON audit_log (
    (payload->>'bytebaseId'),
    (payload->>'sequenceNumber')
) WHERE payload->>'bytebaseId' IS NOT NULL;

-- Also add index on bytebase_id alone for faster lookups
CREATE INDEX idx_audit_log_bytebase_id ON audit_log((payload->>'bytebaseId'))
WHERE payload->>'bytebaseId' IS NOT NULL;
