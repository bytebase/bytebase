UPDATE task
SET payload = payload - 'schema_version'
WHERE 
    payload ? 'schema_version'
    AND (payload->>'schema_version' ~ '[^0-9]') -- contains non-digit
;