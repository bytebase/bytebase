DELETE FROM anomaly;
DROP INDEX idx_anomaly_database_id_row_status_type;
DROP INDEX idx_anomaly_instance_id_row_status_type;
ALTER TABLE anomaly ADD COLUMN project TEXT NOT NULL;
CREATE UNIQUE INDEX idx_anomaly_unique_project_database_id_type ON anomaly(project, database_id, type);
