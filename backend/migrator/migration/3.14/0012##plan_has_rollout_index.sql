CREATE INDEX IF NOT EXISTS idx_plan_config_has_rollout ON plan ((config->>'hasRollout'));
