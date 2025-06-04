-- Migrate from preUpdateBackupDetail structure to enablePriorBackup boolean field
-- This migration updates JSON payloads in plan.config, plan_check_run.config, and task.payload

-- Update plan.config field
-- Replace preUpdateBackupDetail objects with enablePriorBackup boolean in changeDatabaseConfig specs
UPDATE plan
SET config = (
    SELECT jsonb_set(
        config,
        '{specs}',
        COALESCE(
            (
                SELECT jsonb_agg(
                    CASE 
                        WHEN spec->'changeDatabaseConfig'->'preUpdateBackupDetail' IS NOT NULL 
                             AND spec->'changeDatabaseConfig'->'preUpdateBackupDetail' != 'null'::jsonb
                             AND spec->'changeDatabaseConfig'->'preUpdateBackupDetail'->>'database' IS NOT NULL
                             AND spec->'changeDatabaseConfig'->'preUpdateBackupDetail'->>'database' != ''
                        THEN 
                            -- Remove preUpdateBackupDetail and add enablePriorBackup: true
                            jsonb_set(
                                spec #- '{changeDatabaseConfig,preUpdateBackupDetail}',
                                '{changeDatabaseConfig,enablePriorBackup}',
                                'true'::jsonb
                            )
                        WHEN spec->'changeDatabaseConfig'->'preUpdateBackupDetail' IS NOT NULL
                        THEN
                            -- Remove preUpdateBackupDetail and add enablePriorBackup: false
                            jsonb_set(
                                spec #- '{changeDatabaseConfig,preUpdateBackupDetail}',
                                '{changeDatabaseConfig,enablePriorBackup}',
                                'false'::jsonb
                            )
                        ELSE 
                            -- No preUpdateBackupDetail, keep spec as-is
                            spec
                    END
                )
                FROM jsonb_array_elements(config->'specs') AS spec
            ),
            '[]'::jsonb
        )
    )
)
WHERE config->'specs' IS NOT NULL
  AND EXISTS (
      SELECT 1 
      FROM jsonb_array_elements(config->'specs') AS spec
      WHERE spec->'changeDatabaseConfig'->'preUpdateBackupDetail' IS NOT NULL
  );

-- Update plan_check_run.config field
-- Replace preUpdateBackupDetail with enablePriorBackup in plan check run configs
UPDATE plan_check_run
SET config = CASE 
    WHEN config->'preUpdateBackupDetail' IS NOT NULL 
         AND config->'preUpdateBackupDetail' != 'null'::jsonb
         AND config->'preUpdateBackupDetail'->>'database' IS NOT NULL
         AND config->'preUpdateBackupDetail'->>'database' != ''
    THEN 
        -- Remove preUpdateBackupDetail and add enablePriorBackup: true
        jsonb_set(
            config #- '{preUpdateBackupDetail}',
            '{enablePriorBackup}',
            'true'::jsonb
        )
    WHEN config->'preUpdateBackupDetail' IS NOT NULL
    THEN
        -- Remove preUpdateBackupDetail and add enablePriorBackup: false
        jsonb_set(
            config #- '{preUpdateBackupDetail}',
            '{enablePriorBackup}',
            'false'::jsonb
        )
    ELSE 
        -- No preUpdateBackupDetail, keep config as-is
        config
END
WHERE config->'preUpdateBackupDetail' IS NOT NULL;

-- Update task.payload field
-- Replace preUpdateBackupDetail with enablePriorBackup in task payloads
UPDATE task
SET payload = CASE 
    WHEN payload->'preUpdateBackupDetail' IS NOT NULL 
         AND payload->'preUpdateBackupDetail' != 'null'::jsonb
         AND payload->'preUpdateBackupDetail'->>'database' IS NOT NULL
         AND payload->'preUpdateBackupDetail'->>'database' != ''
    THEN 
        -- Remove preUpdateBackupDetail and add enablePriorBackup: true
        jsonb_set(
            payload #- '{preUpdateBackupDetail}',
            '{enablePriorBackup}',
            'true'::jsonb
        )
    WHEN payload->'preUpdateBackupDetail' IS NOT NULL
    THEN
        -- Remove preUpdateBackupDetail and add enablePriorBackup: false
        jsonb_set(
            payload #- '{preUpdateBackupDetail}',
            '{enablePriorBackup}',
            'false'::jsonb
        )
    ELSE 
        -- No preUpdateBackupDetail, keep payload as-is
        payload
END
WHERE payload->'preUpdateBackupDetail' IS NOT NULL;
