-- Add environment field to task table
ALTER TABLE task ADD COLUMN environment TEXT;

-- Migrate environment data from stage to task
UPDATE task 
SET environment = stage.environment
FROM stage
WHERE task.stage_id = stage.id;

-- Migrate task and task_run references from numeric stage IDs to environment IDs
-- The format changes from:
--   projects/{project}/rollouts/{rollout}/stages/{numeric_stage_id}/tasks/{task}
--   projects/{project}/rollouts/{rollout}/stages/{numeric_stage_id}/tasks/{task}/taskruns/{taskrun}
-- to:
--   projects/{project}/rollouts/{rollout}/stages/{environment_id}/tasks/{task}
--   projects/{project}/rollouts/{rollout}/stages/{environment_id}/tasks/{task}/taskruns/{taskrun}

-- Create a temporary function to update stage references in resource paths
CREATE OR REPLACE FUNCTION update_stage_reference(resource_path text) RETURNS text AS $$
DECLARE
    stage_match text;
    stage_id int;
    environment_id text;
BEGIN
    -- Check if the path contains /stages/{numeric_id}
    IF resource_path !~ '/stages/[0-9]+' THEN
        RETURN resource_path; -- Return unchanged if no numeric stage ID found
    END IF;
    
    -- Extract the numeric stage ID
    stage_match := substring(resource_path from '/stages/([0-9]+)');
    IF stage_match IS NULL THEN
        RETURN resource_path;
    END IF;
    stage_id := stage_match::int;
    
    -- Get the environment ID from the stage table (stage_id is primary key)
    SELECT s.environment INTO environment_id
    FROM stage s
    WHERE s.id = stage_id;
    
    IF environment_id IS NULL THEN
        RETURN resource_path; -- Return unchanged if stage not found
    END IF;
    
    -- Replace /stages/{numeric_id} with /stages/{environment_id}
    RETURN regexp_replace(resource_path, '/stages/' || stage_id, '/stages/' || environment_id);
END;
$$ LANGUAGE plpgsql;

-- Update all changelog entries with taskRun references
UPDATE changelog
SET payload = jsonb_set(
    payload,
    '{taskRun}',
    to_jsonb(update_stage_reference(payload->>'taskRun'))
)
WHERE payload ? 'taskRun' 
  AND payload->>'taskRun' != ''
  AND payload->>'taskRun' IS NOT NULL;

-- Update all issue_comment entries with task references in taskUpdate
UPDATE issue_comment
SET payload = jsonb_set(
    payload,
    '{taskUpdate,tasks}',
    (
        SELECT jsonb_agg(update_stage_reference(task_ref))
        FROM jsonb_array_elements_text(payload->'taskUpdate'->'tasks') AS task_ref
    )
)
WHERE payload->'taskUpdate' IS NOT NULL
  AND jsonb_typeof(payload->'taskUpdate'->'tasks') = 'array'
  AND CASE WHEN jsonb_typeof(payload->'taskUpdate'->'tasks') = 'array'
           THEN jsonb_array_length(payload->'taskUpdate'->'tasks') > 0
           ELSE false END;

-- Update all issue_comment entries with task references in taskPriorBackup
UPDATE issue_comment
SET payload = jsonb_set(
    payload,
    '{taskPriorBackup,task}',
    to_jsonb(update_stage_reference(payload->'taskPriorBackup'->>'task'))
)
WHERE payload->'taskPriorBackup' IS NOT NULL
  AND payload->'taskPriorBackup'->'task' IS NOT NULL
  AND payload->'taskPriorBackup'->>'task' != '';

-- Update all issue_comment entries with stage references in stageEnd
UPDATE issue_comment
SET payload = jsonb_set(
    payload,
    '{stageEnd,stage}',
    to_jsonb(update_stage_reference(payload->'stageEnd'->>'stage'))
)
WHERE payload->'stageEnd' IS NOT NULL
  AND payload->'stageEnd'->'stage' IS NOT NULL
  AND payload->'stageEnd'->>'stage' != '';

-- Drop the temporary function
DROP FUNCTION update_stage_reference(text);

-- Remove stage_id foreign key and column from task table
ALTER TABLE task DROP COLUMN stage_id;

-- Drop indexes related to stage
DROP INDEX IF EXISTS idx_task_pipeline_id_stage_id;
CREATE INDEX idx_task_pipeline_id_environment ON task(pipeline_id, environment);

-- Drop stage table
DROP TABLE stage;