UPDATE policy
SET payload = replace(payload::TEXT, '"maskingLevel": "FULL"', '"semanticType": "default"')::JSONB
WHERE type = 'bb.policy.masking-rule';
UPDATE policy
SET payload = replace(payload::TEXT, '"maskingLevel": "PARTIAL"', '"semanticType": "default-partial"')::JSONB
WHERE type = 'bb.policy.masking-rule';