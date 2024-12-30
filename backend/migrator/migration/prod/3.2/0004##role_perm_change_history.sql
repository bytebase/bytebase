UPDATE role
SET permissions = replace(permissions::TEXT, 'bb.changeHistories', 'bb.changelogs')::JSONB;