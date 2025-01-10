ALTER TABLE anomaly ADD COLUMN project TEXT;
UPDATE anomaly
SET project = project.resource_id
FROM db
    JOIN project ON db.project_id = project.id
WHERE
    db.id = anomaly.database_id;