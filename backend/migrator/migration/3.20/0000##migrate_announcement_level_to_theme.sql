-- Convert the deprecated announcement `level` enum into the new `theme` colors
-- (background/text "r g b"), then drop the obsolete `level` key. Levels map to
-- the previous banner colors: INFO=--color-info, WARNING=--color-warning,
-- CRITICAL=--color-error; text was always white.
--
-- Safety:
--   * Only touches a WORKSPACE_PROFILE row whose value has a real announcement
--     OBJECT carrying a `level` (the `? 'level'` test is false for a missing /
--     null / scalar announcement), so it never fabricates an announcement.
--   * Idempotent: the run removes `level` and the WHERE also excludes rows that
--     already have a `theme`, so a re-run matches nothing (and never overwrites
--     an existing theme).
UPDATE setting
SET value = jsonb_set(
    value #- '{announcement,level}',
    '{announcement,theme}',
    CASE
        WHEN value->'announcement'->>'level' IN ('WARNING', '2')
            THEN jsonb_build_object('background', '245 158 11', 'text', '255 255 255')
        WHEN value->'announcement'->>'level' IN ('CRITICAL', '3')
            THEN jsonb_build_object('background', '220 38 38', 'text', '255 255 255')
        ELSE jsonb_build_object('background', '37 99 235', 'text', '255 255 255')
    END,
    true
)
WHERE name = 'WORKSPACE_PROFILE'
  AND value ? 'announcement'
  AND value->'announcement' ? 'level'
  AND NOT (value->'announcement' ? 'theme');
