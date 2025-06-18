-- Remove PLUGIN_AGENT setting data as this feature has been removed
DELETE FROM setting WHERE name = 'PLUGIN_AGENT';