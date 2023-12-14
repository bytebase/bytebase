ALTER TABLE branch ADD COLUMN base_schema TEXT NOT NULL DEFAULT '';
ALTER TABLE branch ADD COLUMN head_schema TEXT NOT NULL DEFAULT '';
ALTER TABLE branch ADD COLUMN reconcile_state TEXT NOT NULL DEFAULT '';

UPDATE branch SET base_schema = convert_from(decode(base->>'schema', 'base64'), 'utf8');
UPDATE branch SET head_schema = convert_from(decode(head->>'schema', 'base64'), 'utf8');
UPDATE branch SET base = base - 'schema';
UPDATE branch SET head = head - 'schema';
