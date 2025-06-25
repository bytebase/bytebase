UPDATE db_schema
SET
    metadata = update_data.transformed_metadata
FROM (
    -- The entire "preserve structure" SELECT query is now the data source
    SELECT
        c.id,
        (c.metadata::jsonb - 'schemas') || jsonb_build_object('schemas', sub.new_schemas_array) AS transformed_metadata
    FROM
        db_schema c
    JOIN (
        -- This subquery rebuilds the entire 'schemas' array, preserving all elements
        SELECT
            c.id,
            jsonb_agg(
                s.schema_element - 'tables' || jsonb_build_object('tables', t.new_tables_array)
                ORDER BY s.schema_idx
            ) AS new_schemas_array
        FROM
            db_schema c,
            LATERAL jsonb_array_elements(c.metadata::jsonb -> 'schemas') WITH ORDINALITY AS s(schema_element, schema_idx)
            LEFT JOIN LATERAL (
                SELECT
                    jsonb_agg(
                        -- CORE LOGIC: Decide whether to transform a table or keep it as is.
                        CASE
                            WHEN jsonb_typeof(t.table_element -> 'columns') = 'array' AND jsonb_array_length(t.table_element -> 'columns') > 0 THEN
                                t.table_element - 'columns' || jsonb_build_object('columns', col.new_columns_array)
                            ELSE
                                t.table_element
                        END
                        ORDER BY t.table_idx
                    ) AS new_tables_array
                FROM
                    jsonb_array_elements(s.schema_element -> 'tables') WITH ORDINALITY AS t(table_element, table_idx)
                    LEFT JOIN LATERAL (
                        SELECT
                            jsonb_agg(
                                -- This is the transformation applied to each column
                                CASE
                                    WHEN c.col_meta ->> 'defaultNull' = 'true' THEN
                                        (c.col_meta - 'defaultNull') || '{"default": "NULL"}'::jsonb
                                    WHEN c.col_meta ? 'defaultExpression' THEN
                                        (c.col_meta - 'defaultExpression') || jsonb_build_object('default', c.col_meta -> 'defaultExpression')
                                    ELSE
                                        c.col_meta
                                END
                            ) AS new_columns_array
                        FROM
                            jsonb_array_elements(t.table_element -> 'columns') AS c(col_meta)
                    ) AS col ON TRUE
            ) AS t ON TRUE
        GROUP BY
            c.id
    ) AS sub ON c.id = sub.id
) AS update_data
WHERE
    db_schema.id = update_data.id;
