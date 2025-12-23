-- Remove deployment field from plan config
-- The deployment snapshot is no longer needed as database groups are now evaluated live
-- and environment order is retrieved from the store when needed

UPDATE plan
SET config = config - 'deployment'
WHERE config ? 'deployment';
