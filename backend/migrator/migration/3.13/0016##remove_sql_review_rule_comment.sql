UPDATE
    review_config
SET
    payload = jsonb_set(
        payload,
        '{sqlReviewRules}',
        (
            SELECT
                jsonb_agg(elem - 'comment')
            FROM
                jsonb_array_elements(payload -> 'sqlReviewRules') AS elem
        )
    )
WHERE
    payload ->> 'sqlReviewRules' IS NOT NULL;
