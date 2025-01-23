UPDATE policy
SET payload = replace(payload::TEXT, '"maskingLevel": "FULL"', '"semanticType": "bb.default"')::JSONB
WHERE type = 'bb.policy.masking-rule';
UPDATE policy
SET payload = replace(payload::TEXT, '"maskingLevel": "PARTIAL"', '"semanticType": "bb.default-partial"')::JSONB
WHERE type = 'bb.policy.masking-rule';