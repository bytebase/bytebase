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
-- New rules are appended at the END of the array — we deliberately do
-- NOT reorder existing REQUEST_ACCESS rules (for role grants / unmask /
-- etc.) ahead of our additions. If a workspace already has a broad
-- REQUEST_ACCESS rule that intentionally gates every REQUEST_ACCESS
-- request, reordering would override the admin's chosen evaluation
-- order. The runner returns the first source-matching rule whose
-- condition holds; admins can manually reorder in the UI after this
-- backfill if they want export-specific gates to win over their
-- existing broad rules.
--
-- Per-workspace: `setting` is keyed by `(workspace, name)` so each
-- workspace gets its own computed rules instead of one workspace's rules
-- being broadcast to all (PR #20501 bot review #3353501482).
--
-- This migration relies on REQUEST_ACCESS rule evaluation populating the
-- same `resource.db_engine` / `resource.database_name` /
-- `resource.schema_name` / `resource.table_name` CEL attributes that
-- EXPORT_DATA rules use (see `buildCELVariablesForAccessGrant` in
-- `backend/runner/approval/runner.go`) — so an EXPORT_DATA rule scoped
-- to a specific database / engine / table mirrors verbatim and continues
-- to gate the same exports under the JIT path (PR #20501 bot review
-- #3353501487).
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
BEGIN
  FOR ws_row IN
    SELECT workspace, value FROM setting WHERE name = 'WORKSPACE_APPROVAL'
  LOOP
    setting_value := ws_row.value;
    rules := COALESCE(setting_value->'rules', '[]'::jsonb);

    IF EXISTS (
      SELECT 1 FROM jsonb_array_elements(rules) r
      WHERE r->>'source' = 'REQUEST_ACCESS'
        AND COALESCE(r->'condition'->>'expression', '') LIKE '%request.data_export == true%'
    ) THEN CONTINUE; END IF;

    result_rules := rules;
    fallback_template := NULL;

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

    UPDATE setting
    SET value = jsonb_build_object('rules', result_rules)
    WHERE workspace = ws_row.workspace AND name = 'WORKSPACE_APPROVAL';
  END LOOP;
END $$;
