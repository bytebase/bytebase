UPDATE risk 
SET expression = expression || jsonb_build_object(
    'expression',
    'role == "roles/sqlEditorUser" && (' || (expression->>'expression') || ')'
), source = 'bb.risk.request.role'
WHERE source = 'bb.risk.request.query';

UPDATE risk 
SET expression = expression || jsonb_build_object(
    'expression',
    'role == "roles/projectExporter" && (' || (expression->>'expression') || ')'
), source = 'bb.risk.request.role'
WHERE source = 'bb.risk.request.export';