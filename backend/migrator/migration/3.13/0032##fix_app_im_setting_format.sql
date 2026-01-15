-- Fix APP_IM setting format: nest payload fields under their respective oneof keys
-- The migration 3.12/0000##migrate_app_im_to_array.sql created flat structures like:
--   { "type": "WECOM", "corpId": "...", "agentId": "...", "secret": "..." }
-- But protojson expects nested structures for oneof fields:
--   { "type": "WECOM", "wecom": { "corpId": "...", "agentId": "...", "secret": "..." } }

DO $$
DECLARE
    old_val jsonb;
    new_settings jsonb := '[]'::jsonb;
    setting jsonb;
    setting_type text;
    new_setting jsonb;
BEGIN
    -- Get the current value
    SELECT value INTO old_val FROM setting WHERE name = 'APP_IM';

    -- Return early if no APP_IM setting exists
    IF old_val IS NULL THEN
        RETURN;
    END IF;

    -- Return early if no settings array
    IF NOT old_val ? 'settings' THEN
        RETURN;
    END IF;

    -- Process each setting in the array
    FOR setting IN SELECT * FROM jsonb_array_elements(old_val->'settings')
    LOOP
        setting_type := setting->>'type';

        -- Check if already in correct format (has nested payload)
        IF setting ? 'slack' OR setting ? 'feishu' OR setting ? 'wecom' OR
           setting ? 'lark' OR setting ? 'dingtalk' OR setting ? 'teams' THEN
            -- Already in correct format, keep as is
            new_settings := new_settings || jsonb_build_array(setting);
        ELSE
            -- Convert flat format to nested format
            CASE setting_type
                WHEN 'SLACK' THEN
                    new_setting := jsonb_build_object(
                        'type', 'SLACK',
                        'slack', jsonb_build_object(
                            'token', setting->>'token'
                        )
                    );
                WHEN 'FEISHU' THEN
                    new_setting := jsonb_build_object(
                        'type', 'FEISHU',
                        'feishu', jsonb_build_object(
                            'appId', setting->>'appId',
                            'appSecret', setting->>'appSecret'
                        )
                    );
                WHEN 'WECOM' THEN
                    new_setting := jsonb_build_object(
                        'type', 'WECOM',
                        'wecom', jsonb_build_object(
                            'corpId', setting->>'corpId',
                            'agentId', setting->>'agentId',
                            'secret', setting->>'secret'
                        )
                    );
                WHEN 'LARK' THEN
                    new_setting := jsonb_build_object(
                        'type', 'LARK',
                        'lark', jsonb_build_object(
                            'appId', setting->>'appId',
                            'appSecret', setting->>'appSecret'
                        )
                    );
                WHEN 'DINGTALK' THEN
                    new_setting := jsonb_build_object(
                        'type', 'DINGTALK',
                        'dingtalk', jsonb_build_object(
                            'clientId', setting->>'clientId',
                            'clientSecret', setting->>'clientSecret',
                            'robotCode', setting->>'robotCode'
                        )
                    );
                WHEN 'TEAMS' THEN
                    new_setting := jsonb_build_object(
                        'type', 'TEAMS',
                        'teams', jsonb_build_object(
                            'tenantId', setting->>'tenantId',
                            'clientId', setting->>'clientId',
                            'clientSecret', setting->>'clientSecret'
                        )
                    );
                ELSE
                    -- Unknown type, keep as is
                    new_setting := setting;
            END CASE;
            new_settings := new_settings || jsonb_build_array(new_setting);
        END IF;
    END LOOP;

    -- Update the setting with the corrected format
    UPDATE setting SET value = jsonb_build_object('settings', new_settings) WHERE name = 'APP_IM';
END $$;