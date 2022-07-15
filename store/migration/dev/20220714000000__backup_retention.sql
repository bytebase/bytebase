ALTER TABLE backup_setting ADD retention_period_ts INTEGER CHECK (retention_period_ts >= 0);
