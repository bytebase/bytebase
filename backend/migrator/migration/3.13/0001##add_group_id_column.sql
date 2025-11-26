-- Add id column with UUID default for new rows
ALTER TABLE user_group ADD COLUMN IF NOT EXISTS id TEXT DEFAULT gen_random_uuid()::text;

-- Backfill existing rows:
-- 1. If email doesn't contain '@', it's likely an Azure object ID (legacy mapping: objectId -> externalId -> email)
--    Move the value to id column and set email to NULL
-- 2. Otherwise, generate a new UUID for id
UPDATE user_group SET
    id = CASE
        WHEN email IS NOT NULL AND email NOT LIKE '%@%' THEN email
        ELSE gen_random_uuid()::text
    END,
    email = CASE
        WHEN email IS NOT NULL AND email NOT LIKE '%@%' THEN NULL
        ELSE email
    END
WHERE id IS NULL OR id = '';

-- Set NOT NULL constraint after backfill
ALTER TABLE user_group ALTER COLUMN id SET NOT NULL;

-- Drop the primary key constraint on email if it exists
ALTER TABLE user_group DROP CONSTRAINT IF EXISTS user_group_pkey;

-- Add primary key on id if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'user_group_pkey' AND conrelid = 'user_group'::regclass
    ) THEN
        ALTER TABLE user_group ADD PRIMARY KEY (id);
    END IF;
END $$;

-- Make email nullable
ALTER TABLE user_group ALTER COLUMN email DROP NOT NULL;

-- Create unique index on email (only for non-null values)
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_group_unique_email ON user_group(email) WHERE email IS NOT NULL;
