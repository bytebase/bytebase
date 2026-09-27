-- Convert the deprecated announcement `level` enum into the new `theme` colors
-- (background/text google.type.Color JSON), then drop the obsolete `level` key. Levels map to
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
            THEN jsonb_build_object(
                'background', jsonb_build_object('red', 0.960784, 'green', 0.619608, 'blue', 0.043137),
                'text', jsonb_build_object('red', 1, 'green', 1, 'blue', 1)
            )
        WHEN value->'announcement'->>'level' IN ('CRITICAL', '3')
            THEN jsonb_build_object(
                'background', jsonb_build_object('red', 0.862745, 'green', 0.14902, 'blue', 0.14902),
                'text', jsonb_build_object('red', 1, 'green', 1, 'blue', 1)
            )
        ELSE jsonb_build_object(
            'background', jsonb_build_object('red', 0.145098, 'green', 0.388235, 'blue', 0.921569),
            'text', jsonb_build_object('red', 1, 'green', 1, 'blue', 1)
        )
    END,
    true
)
WHERE name = 'WORKSPACE_PROFILE'
  AND value ? 'announcement'
  AND value->'announcement' ? 'level'
  AND NOT (value->'announcement' ? 'theme');
