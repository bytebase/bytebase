-- Migrate APP_IM setting from object format to array format
-- Old format: { "slack": { "enabled": true, "token": "..." }, "feishu": { ... }, ... }
-- New format: { "settings": [ { "type": "SLACK", "slack": {"token": "..."} }, { "type": "FEISHU", "feishu": {"appId": "...", "appSecret": "..."} }, ... ] }
-- Note: Each setting must have nested payload under its type key (e.g., "slack", "wecom") for protojson oneof deserialization.

DO $$
DECLARE
    old_val jsonb;
    settings_array jsonb := '[]'::jsonb;
BEGIN
    -- Get the current value
    SELECT value INTO old_val FROM setting WHERE name = 'APP_IM';

    -- Return early if no APP_IM setting exists
    IF old_val IS NULL THEN
        RETURN;
    END IF;

    -- Process Slack
    IF old_val->'slack'->>'enabled' = 'true' THEN
        settings_array := settings_array || jsonb_build_array(
            jsonb_build_object(
                'type', 'SLACK',
                'slack', jsonb_build_object(
                    'token', old_val->'slack'->>'token'
                )
            )
        );
    END IF;

    -- Process Feishu
    IF old_val->'feishu'->>'enabled' = 'true' THEN
        settings_array := settings_array || jsonb_build_array(
            jsonb_build_object(
                'type', 'FEISHU',
                'feishu', jsonb_build_object(
                    'appId', old_val->'feishu'->>'appId',
                    'appSecret', old_val->'feishu'->>'appSecret'
                )
            )
        );
    END IF;

    -- Process Wecom
    IF old_val->'wecom'->>'enabled' = 'true' THEN
        settings_array := settings_array || jsonb_build_array(
            jsonb_build_object(
                'type', 'WECOM',
                'wecom', jsonb_build_object(
                    'corpId', old_val->'wecom'->>'corpId',
                    'agentId', old_val->'wecom'->>'agentId',
                    'secret', old_val->'wecom'->>'secret'
                )
            )
        );
    END IF;

    -- Process Lark
    IF old_val->'lark'->>'enabled' = 'true' THEN
        settings_array := settings_array || jsonb_build_array(
            jsonb_build_object(
                'type', 'LARK',
                'lark', jsonb_build_object(
                    'appId', old_val->'lark'->>'appId',
                    'appSecret', old_val->'lark'->>'appSecret'
                )
            )
        );
    END IF;

    -- Process DingTalk
    IF old_val->'dingtalk'->>'enabled' = 'true' THEN
        settings_array := settings_array || jsonb_build_array(
            jsonb_build_object(
                'type', 'DINGTALK',
                'dingtalk', jsonb_build_object(
                    'clientId', old_val->'dingtalk'->>'clientId',
                    'clientSecret', old_val->'dingtalk'->>'clientSecret',
                    'robotCode', old_val->'dingtalk'->>'robotCode'
                )
            )
        );
    END IF;

    -- Update the setting with the new object format containing settings array
    UPDATE setting SET value = jsonb_build_object('settings', settings_array) WHERE name = 'APP_IM';
END $$;