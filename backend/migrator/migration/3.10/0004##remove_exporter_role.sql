-- Remove projectExporter role and merge row_limit condition into sqlEditorUser role
WITH exporter_data AS (
    -- Collect all projectExporter members and their row_limit conditions
    SELECT
        p.id,
        m.member,
        substring(b.binding->'condition'->>'expression' from 'request\.row_limit\s*<=\s*\d+') as row_limit
    FROM policy p,
        jsonb_array_elements(p.payload->'bindings') b(binding),
        jsonb_array_elements_text(b.binding->'members') m(member)
    WHERE p.resource_type = 'PROJECT'
        AND p.type = 'IAM'
        AND b.binding->>'role' = 'roles/projectExporter'
        AND b.binding->'condition'->>'expression' ~ 'request\.row_limit\s*<=\s*\d+'
)
UPDATE policy p
SET payload = jsonb_set(
    p.payload,
    '{bindings}',
    (
        SELECT jsonb_agg(
            CASE
                -- Update sqlEditorUser bindings with row_limit if member exists in exporter_data
                WHEN b.binding->>'role' = 'roles/sqlEditorUser' AND EXISTS (
                    SELECT 1 FROM exporter_data ed
                    WHERE ed.id = p.id
                        AND ed.member IN (SELECT jsonb_array_elements_text(b.binding->'members'))
                ) THEN
                    jsonb_set(
                        b.binding,
                        '{condition,expression}',
                        to_jsonb(
                            COALESCE(b.binding->'condition'->>'expression', '') ||
                            CASE
                                WHEN b.binding->'condition'->>'expression' IS NOT NULL THEN ' && '
                                ELSE ''
                            END ||
                            (SELECT ed.row_limit FROM exporter_data ed
                             WHERE ed.id = p.id
                                AND ed.member IN (SELECT jsonb_array_elements_text(b.binding->'members'))
                             LIMIT 1)
                        )
                    )
                -- Keep all other non-projectExporter bindings unchanged
                ELSE b.binding
            END
        )
        FROM jsonb_array_elements(p.payload->'bindings') b(binding)
        WHERE b.binding->>'role' != 'roles/projectExporter'
    )
)
WHERE p.resource_type = 'PROJECT'
    AND p.type = 'IAM'
    AND EXISTS (
        SELECT 1
        FROM jsonb_array_elements(p.payload->'bindings') b(binding)
        WHERE b.binding->>'role' = 'roles/projectExporter'
    );