-- Refactor plan config to use migrate_type.
DO $$
DECLARE
    plan_record RECORD;
    spec JSONB;
    new_specs JSONB[] := '{}';
    new_spec JSONB;
    change_config JSONB;
    old_type TEXT;
    i INT;
BEGIN
    FOR plan_record IN SELECT id, config FROM plan LOOP
        new_specs := '{}';
        i := 0;
        FOR spec IN SELECT * FROM jsonb_array_elements(plan_record.config->'specs') LOOP
            i := i + 1;
            new_spec := spec;
            IF new_spec ? 'changeDatabaseConfig' THEN
                change_config := new_spec->'changeDatabaseConfig';
                old_type := change_config->>'type';

                IF old_type = 'MIGRATE' THEN
                    change_config := jsonb_set(change_config, '{migrateType}', '"DDL"');
                ELSIF old_type = 'MIGRATE_GHOST' THEN
                    change_config := jsonb_set(change_config, '{type}', '"MIGRATE"');
                    change_config := jsonb_set(change_config, '{migrateType}', '"GHOST"');
                ELSIF old_type = 'DATA' THEN
                    change_config := jsonb_set(change_config, '{type}', '"MIGRATE"');
                    change_config := jsonb_set(change_config, '{migrateType}', '"DML"');
                ELSIF old_type = 'MIGRATE_SDL' THEN
                    change_config := jsonb_set(change_config, '{type}', '"SDL"');
                END IF;
                new_spec := jsonb_set(new_spec, '{changeDatabaseConfig}', change_config);
            END IF;
            new_specs := array_append(new_specs, new_spec);
        END LOOP;
        UPDATE plan SET config = jsonb_set(plan_record.config, '{specs}', to_jsonb(new_specs)) WHERE id = plan_record.id;
    END LOOP;
END $$;
