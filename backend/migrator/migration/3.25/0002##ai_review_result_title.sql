-- An AI review result kept its title as the first line of its comment.
UPDATE issue_comment
SET payload = jsonb_set(
  payload || jsonb_build_object('comment', regexp_replace(payload->>'comment', '^[^\n]*\n*', '')),
  '{reviewMetadata,title}',
  to_jsonb(btrim(split_part(payload->>'comment', E'\n', 1))))
WHERE payload->'reviewMetadata'->>'runType' = 'AI'
  AND payload ? 'comment'
  AND NOT payload->'reviewMetadata' ? 'title';
