-- Consolidate DDL and DML approval sources into CHANGE_DATABASE.
-- Preserve backward compatibility by encoding DDL/DML distinction into CEL conditions.

UPDATE setting
SET value = (
  SELECT jsonb_set(
    value::jsonb,
    '{rules}',
    (
      -- DDL rules: rename to CHANGE_DATABASE, append (DDL) to title, and add DDL condition filter
      SELECT COALESCE(jsonb_agg(
        jsonb_set(
          jsonb_set(
            jsonb_set(rule, '{source}', '"CHANGE_DATABASE"'),
            '{template,title}',
            to_jsonb((rule->'template'->>'title') || ' (DDL)')
          ),
          '{condition,expression}',
          to_jsonb(
            CASE
              WHEN COALESCE(rule->'condition'->>'expression', '') IN ('', 'true')
              THEN '!(statement.sql_type in ["DELETE", "INSERT", "UPDATE"])'
              ELSE '(' || (rule->'condition'->>'expression') || ') && !(statement.sql_type in ["DELETE", "INSERT", "UPDATE"])'
            END
          )
        )
      ), '[]'::jsonb)
      FROM jsonb_array_elements(value::jsonb->'rules') AS rule
      WHERE rule->>'source' = 'DDL'
    ) || (
      -- DML rules: rename to CHANGE_DATABASE, append (DML) to title, and add DML condition filter (appended after DDL)
      SELECT COALESCE(jsonb_agg(
        jsonb_set(
          jsonb_set(
            jsonb_set(rule, '{source}', '"CHANGE_DATABASE"'),
            '{template,title}',
            to_jsonb((rule->'template'->>'title') || ' (DML)')
          ),
          '{condition,expression}',
          to_jsonb(
            CASE
              WHEN COALESCE(rule->'condition'->>'expression', '') IN ('', 'true')
              THEN 'statement.sql_type in ["DELETE", "INSERT", "UPDATE"]'
              ELSE '(' || (rule->'condition'->>'expression') || ') && statement.sql_type in ["DELETE", "INSERT", "UPDATE"]'
            END
          )
        )
      ), '[]'::jsonb)
      FROM jsonb_array_elements(value::jsonb->'rules') AS rule
      WHERE rule->>'source' = 'DML'
    ) || (
      -- Other rules: keep unchanged
      SELECT COALESCE(jsonb_agg(rule), '[]'::jsonb)
      FROM jsonb_array_elements(value::jsonb->'rules') AS rule
      WHERE rule->>'source' NOT IN ('DDL', 'DML')
    )
  )::text
)
WHERE name = 'WORKSPACE_APPROVAL';
