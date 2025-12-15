-- Remove unused changedResources field from changelog payload
-- This field was write-only and never consumed by any code
UPDATE changelog
SET payload = payload - 'changedResources'
WHERE payload ? 'changedResources';
