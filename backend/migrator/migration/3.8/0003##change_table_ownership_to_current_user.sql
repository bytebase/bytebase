-- Change ownership of all Bytebase tables and sequences to the current connection user
DO $$
DECLARE
    current_user_name text;
    obj_rec RECORD;
BEGIN
    -- Get current user
    current_user_name := current_user;
    
    -- Change ownership for all tables in current schema
    FOR obj_rec IN 
        SELECT tablename 
        FROM pg_tables 
        WHERE schemaname = current_schema
    LOOP
        EXECUTE format('ALTER TABLE %I OWNER TO %I', obj_rec.tablename, current_user_name);
    END LOOP;
    
    -- Change ownership for all sequences in current schema
    FOR obj_rec IN 
        SELECT sequence_name 
        FROM information_schema.sequences 
        WHERE sequence_schema = current_schema
    LOOP
        EXECUTE format('ALTER SEQUENCE %I OWNER TO %I', obj_rec.sequence_name, current_user_name);
    END LOOP;
END $$;