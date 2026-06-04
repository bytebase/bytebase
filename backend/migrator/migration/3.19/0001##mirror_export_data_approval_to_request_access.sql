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
-- Idempotency: REQUEST_ACCESS is a long-established source used for role
-- grants / unmask requests. Those existing rules must coexist. We only
-- skip when at least one REQUEST_ACCESS rule already references
-- `request.data_export == true` — meaning an admin (or a prior run) has
-- already wired up the export gate.

DO $$
DECLARE
  setting_value jsonb;
  rules jsonb;
  rule jsonb;
  result_rules jsonb;
  fallback_template jsonb;
  existing_expression text;
  combined_expression text;
BEGIN
  SELECT value INTO setting_value
  FROM setting WHERE name = 'WORKSPACE_APPROVAL';

  IF setting_value IS NULL THEN RETURN; END IF;
  rules := COALESCE(setting_value->'rules', '[]'::jsonb);

  IF EXISTS (
    SELECT 1 FROM jsonb_array_elements(rules) r
    WHERE r->>'source' = 'REQUEST_ACCESS'
      AND COALESCE(r->'condition'->>'expression', '') LIKE '%request.data_export == true%'
  ) THEN RETURN; END IF;

  result_rules := rules;

  -- Mirror each EXPORT_DATA rule with a non-trivial condition into a
  -- REQUEST_ACCESS sibling. Trivial-true ("" or "true") EXPORT_DATA rules
  -- are NOT mirrored — they'd produce a REQUEST_ACCESS rule with just
  -- `request.data_export == true`, which is exactly what the explicit
  -- fallback rule (appended after the loop) already covers. Mirroring
  -- the trivial rule would create a duplicate, and worse, if the
  -- EXPORT_DATA fallback appears before any specific EXPORT_DATA rules
  -- in the array, the mirrored duplicate would land before the specific
  -- mirrored rules and shadow them. The trivial rule's template still
  -- contributes — we capture it as the preferred fallback template.
  --
  -- WITH ORDINALITY + ORDER BY pins the iteration order so the mirrored
  -- rules end up in the same relative order as their originating
  -- EXPORT_DATA rules.
  FOR rule IN
    SELECT value
    FROM jsonb_array_elements(rules) WITH ORDINALITY AS arr(value, idx)
    WHERE value->>'source' = 'EXPORT_DATA'
    ORDER BY arr.idx
  LOOP
    existing_expression := COALESCE(rule->'condition'->>'expression', '');

    IF btrim(existing_expression) IN ('', 'true') THEN
      -- Trivial EXPORT_DATA fallback — don't mirror, but remember its
      -- template (this is the canonical "data export fallback" template
      -- the workspace admin already chose).
      fallback_template := rule->'template';
      CONTINUE;
    END IF;

    combined_expression := '(' || existing_expression || ') && request.data_export == true';
    result_rules := result_rules || jsonb_build_array(jsonb_build_object(
      'template', rule->'template',
      'source', 'REQUEST_ACCESS',
      'condition', jsonb_build_object('expression', combined_expression)
    ));

    -- Use the first specific rule's template as the fallback only if
    -- we haven't seen the trivial-fallback's template yet.
    IF fallback_template IS NULL THEN
      fallback_template := rule->'template';
    END IF;
  END LOOP;

  -- Append the catch-all fallback. Skipped when there are no EXPORT_DATA
  -- rules to derive a template from.
  IF fallback_template IS NOT NULL THEN
    result_rules := result_rules || jsonb_build_array(jsonb_build_object(
      'template', fallback_template,
      'source', 'REQUEST_ACCESS',
      'condition', jsonb_build_object('expression', 'request.data_export == true')
    ));
  END IF;

  UPDATE setting
  SET value = jsonb_build_object('rules', result_rules)
  WHERE name = 'WORKSPACE_APPROVAL';
END $$;
