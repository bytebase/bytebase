ALTER TABLE backup_setting ADD retention_period_ts INTEGER NOT NULL DEFAULT 0 CHECK (retention_period_ts >= 0);
