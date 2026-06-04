-- Backfill REQUEST_ACCESS approval rules from existing EXPORT_DATA rules.
--
-- Before this migration access-grant requests with `export = true` had no
-- matching approval rule (the WORKSPACE_APPROVAL setting only covered the
-- direct DATABASE_EXPORT issue path), so JIT-grant exports effectively
-- skipped the workspace's export approval policy — a data-security gap.
--
-- For each EXPORT_DATA rule we copy the template into a sibling
-- REQUEST_ACCESS rule and AND `request.data_export == true` onto the
-- existing CEL expression. The original EXPORT_DATA rule stays in place.
-- We also append a catch-all REQUEST_ACCESS rule keyed on just
-- `request.data_export == true` so any export-capable access-grant
-- request still triggers approval even if no specific rule matches.
--
-- The migration iterates per-workspace — `setting` is keyed by
-- `(workspace, name)` so each workspace gets its own computed rules
-- instead of one workspace's rules being broadcast to all (see PR
-- #20501 bot review #3353501482).
--
-- EXPORT_DATA rules that reference database-specific CEL attributes
-- (`resource.db_engine`, `resource.database_name`, `resource.schema_name`,
-- `resource.table_name`) are NOT mirrored — REQUEST_ACCESS evaluation
-- only sees project/env + request.unmask + request.data_export, so a
-- verbatim copy would never match and silently fall through to the
-- catch-all fallback. We skip the mirror, emit a NOTICE so admins know
-- they need to manually configure REQUEST_ACCESS rules for those
-- specific scopes, and still factor their template into the fallback
-- choice so the security intent isn't entirely lost (see PR #20501
-- bot review #3353501487).
--
-- Idempotency: REQUEST_ACCESS is a long-established source used for role
-- grants / unmask requests. Those existing rules must coexist. We only
-- skip a workspace when at least one of its REQUEST_ACCESS rules already
-- references `request.data_export == true` — meaning an admin (or a
-- prior run) has already wired up the export gate.

DO $$
DECLARE
  ws_row RECORD;
  setting_value jsonb;
  rules jsonb;
  rule jsonb;
  result_rules jsonb;
  fallback_template jsonb;
  existing_expression text;
  combined_expression text;
  skipped_count int;
BEGIN
  FOR ws_row IN
    SELECT workspace, value FROM setting WHERE name = 'WORKSPACE_APPROVAL'
  LOOP
    setting_value := ws_row.value;
    rules := COALESCE(setting_value->'rules', '[]'::jsonb);

    IF EXISTS (
      -- Per-workspace idempotency: skip workspaces that already have a
      -- REQUEST_ACCESS rule covering `request.data_export == true`.
      SELECT 1 FROM jsonb_array_elements(rules) r
      WHERE r->>'source' = 'REQUEST_ACCESS'
        AND COALESCE(r->'condition'->>'expression', '') LIKE '%request.data_export == true%'
    ) THEN CONTINUE; END IF;

    result_rules := rules;
    fallback_template := NULL;
    skipped_count := 0;

    -- Mirror each EXPORT_DATA rule with a non-trivial condition into a
    -- REQUEST_ACCESS sibling. Trivial-true rules and rules with
    -- unsupported attributes only contribute their template (see
    -- preamble).
    FOR rule IN
      SELECT value
      FROM jsonb_array_elements(rules) WITH ORDINALITY AS arr(value, idx)
      WHERE value->>'source' = 'EXPORT_DATA'
      ORDER BY arr.idx
    LOOP
      existing_expression := COALESCE(rule->'condition'->>'expression', '');

      IF btrim(existing_expression) IN ('', 'true') THEN
        -- Trivial EXPORT_DATA fallback — don't mirror (the explicit
        -- fallback at the end already covers this), but adopt its
        -- template as the canonical fallback template.
        fallback_template := rule->'template';
        CONTINUE;
      END IF;

      IF existing_expression LIKE '%resource.database_name%'
         OR existing_expression LIKE '%resource.db_engine%'
         OR existing_expression LIKE '%resource.schema_name%'
         OR existing_expression LIKE '%resource.table_name%' THEN
        -- Unsupported in REQUEST_ACCESS evaluation context — skip the
        -- mirror; the explicit fallback catches this rule's export
        -- requests instead. Still factor the template into the fallback
        -- (so a stricter "sensitive_db" template at least propagates
        -- to the fallback for the first un-mirrorable rule).
        skipped_count := skipped_count + 1;
        IF fallback_template IS NULL THEN
          fallback_template := rule->'template';
        END IF;
        CONTINUE;
      END IF;

      combined_expression := '(' || existing_expression || ') && request.data_export == true';
      result_rules := result_rules || jsonb_build_array(jsonb_build_object(
        'template', rule->'template',
        'source', 'REQUEST_ACCESS',
        'condition', jsonb_build_object('expression', combined_expression)
      ));

      IF fallback_template IS NULL THEN
        fallback_template := rule->'template';
      END IF;
    END LOOP;

    IF fallback_template IS NOT NULL THEN
      result_rules := result_rules || jsonb_build_array(jsonb_build_object(
        'template', fallback_template,
        'source', 'REQUEST_ACCESS',
        'condition', jsonb_build_object('expression', 'request.data_export == true')
      ));
    END IF;

    IF skipped_count > 0 THEN
      RAISE NOTICE
        'Workspace %: skipped % EXPORT_DATA rule(s) with database-specific CEL attributes (db_engine / database_name / schema_name / table_name) — REQUEST_ACCESS evaluation does not see those attributes. The catch-all fallback rule gates exports from those databases; configure stricter REQUEST_ACCESS rules manually if needed.',
        ws_row.workspace, skipped_count;
    END IF;

    UPDATE setting
    SET value = jsonb_build_object('rules', result_rules)
    WHERE workspace = ws_row.workspace AND name = 'WORKSPACE_APPROVAL';
  END LOOP;
END $$;
