-- Add WORKLOAD_IDENTITY to principal type constraint
ALTER TABLE principal
DROP CONSTRAINT principal_type_check,
ADD CONSTRAINT principal_type_check
CHECK (type IN ('END_USER', 'SYSTEM_BOT', 'SERVICE_ACCOUNT', 'WORKLOAD_IDENTITY'));
