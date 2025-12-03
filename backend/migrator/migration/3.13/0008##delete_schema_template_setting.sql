-- Delete the deprecated SCHEMA_TEMPLATE setting
DELETE FROM setting WHERE name = 'SCHEMA_TEMPLATE';
