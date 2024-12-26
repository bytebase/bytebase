DO $$
DECLARE
    h_row RECORD;
    sync_history_pre_id BIGINT;
    sync_history_cur_id BIGINT;
BEGIN
FOR h_row IN (
    SELECT
        project.resource_id AS project_resource_id,
        instance_change_history.creator_id,
        to_timestamp(instance_change_history.created_ts) AS created_ts,
        instance_change_history.database_id,
        instance_change_history.issue_id,
        instance_change_history.sheet_id,
        instance_change_history.status,
        instance_change_history.schema_prev,
        instance_change_history.schema,
        instance_change_history.type,
        instance_change_history.payload
    FROM instance_change_history
    LEFT JOIN project ON project.id = instance_change_history.project_id
    WHERE instance_change_history.instance_id IS NOT NULL
    AND instance_change_history.database_id IS NOT NULL
)
LOOP
    IF COALESCE(h_row.schema_prev, '') != '' THEN
        INSERT INTO sync_history (
            creator_id,
            created_ts,
            database_id,
            raw_dump
        ) SELECT
            1,
            h_row.created_ts,
            h_row.database_id,
            h_row.schema_prev
        RETURNING id
        INTO sync_history_pre_id;
    END IF;

    IF COALESCE(h_row.schema, '') != '' THEN
        INSERT INTO sync_history (
            creator_id,
            created_ts,
            database_id,
            raw_dump
        ) SELECT
            1,
            h_row.created_ts,
            h_row.database_id,
            h_row.schema
        RETURNING id
        INTO sync_history_cur_id;
    END IF;

    INSERT INTO changelog (
        creator_id,
        created_ts,
        database_id,
        status,
        prev_sync_history_id,
        sync_history_id,
        payload
    ) SELECT
        h_row.creator_id,
        h_row.created_ts,
        h_row.database_id,
        h_row.status,
        sync_history_pre_id,
        sync_history_cur_id,
        jsonb_build_object(
            'issue', COALESCE('projects/'||h_row.project_resource_id||'/issues/'||h_row.issue_id,''),
            'changedResources', h_row.payload->'changedResources',
            'sheet', COALESCE('projects/'||h_row.project_resource_id||'/sheets/'||h_row.sheet_id, ''),
            'type', h_row.type::TEXT
        );
END LOOP;
END $$;
