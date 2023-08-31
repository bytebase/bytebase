-- Strip the NULL values.
UPDATE policy SET payload = jsonb_strip_nulls(payload) WHERE type = 'bb.policy.sensitive-data';

-- Delete the maskType field for each elements in array.
UPDATE 
  policy 
SET 
  payload = jsonb_set(
    payload, 
    '{sensitiveDataList}', 
    (
      SELECT 
        jsonb_agg(item - 'maskType') 
      FROM 
        jsonb_array_elements(payload -> 'sensitiveDataList') AS arr(item)
    )
  ) 
WHERE 
  TYPE = 'bb.policy.sensitive-data' 
  AND payload != '{}' :: jsonb;


-- Add "maskingLevel":"FULL" in each elements in array.
UPDATE 
  policy 
SET 
  payload = jsonb_set(
    payload, 
    '{sensitiveDataList}', 
    (
      SELECT 
        jsonb_agg(
          item || '{"maskingLevel": "FULL"}'
        ) 
      FROM 
        jsonb_array_elements(payload -> 'sensitiveDataList') AS arr(item)
    )
  ) 
WHERE 
  TYPE = 'bb.policy.sensitive-data' 
  AND payload != '{}' :: jsonb;


-- Rename `sensitiveDataList` to `maskData`.
UPDATE 
  policy 
SET 
  payload = jsonb_build_object(
      'maskData', payload -> 'sensitiveDataList'
    )
WHERE 
  type = 'bb.policy.sensitive-data' 
  AND payload != '{}' :: jsonb;

UPDATE policy SET type = 'bb.policy.masking' WHERE type = 'bb.policy.sensitive-data';