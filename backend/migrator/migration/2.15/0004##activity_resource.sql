UPDATE activity SET resource_container = '' WHERE resource_container IS NULL;
ALTER TABLE activity ALTER resource_container SET DEFAULT '';
ALTER TABLE activity ALTER resource_container SET NOT NULL;
