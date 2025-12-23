-- Remove maximum_connections field from instance metadata
-- This field is being removed from the proto definition and is no longer used

UPDATE instance
SET metadata = metadata - 'maximumConnections'
WHERE metadata ? 'maximumConnections';
