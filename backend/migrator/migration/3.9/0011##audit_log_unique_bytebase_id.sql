-- Add UNIQUE constraint on audit_log bytebase_id and sequence_number
-- This prevents duplicate sequences within a Bytebase deployment
-- NULL values are allowed for legacy logs created before this feature

-- Safety: Remove any duplicate (bytebase_id, sequence_number) pairs before creating UNIQUE constraint
-- Keep the most recent entry (highest id) if duplicates exist
-- This handles edge cases from testing/rollout where duplicates might exist
DELETE FROM audit_log
WHERE id IN (
    SELECT id FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY payload->>'bytebaseId', payload->>'sequenceNumber'
                   ORDER BY id DESC
               ) AS rn
        FROM audit_log
        WHERE payload->>'bytebaseId' IS NOT NULL
          AND payload->>'sequenceNumber' IS NOT NULL
    ) sub
    WHERE rn > 1
);

-- Now safe to create UNIQUE constraint
CREATE UNIQUE INDEX idx_audit_log_unique_bytebase_id_seq ON audit_log (
    (payload->>'bytebaseId'),
    (payload->>'sequenceNumber')
) WHERE payload->>'bytebaseId' IS NOT NULL;

-- Also add index on bytebase_id alone for faster lookups
CREATE INDEX idx_audit_log_bytebase_id ON audit_log((payload->>'bytebaseId'))
WHERE payload->>'bytebaseId' IS NOT NULL;
