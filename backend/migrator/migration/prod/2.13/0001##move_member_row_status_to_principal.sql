ALTER TABLE principal DISABLE TRIGGER update_principal_updated_ts;
ALTER TABLE member DISABLE TRIGGER update_member_updated_ts;

UPDATE principal
SET row_status = member.row_status
FROM member
WHERE principal.id = member.principal_id;

UPDATE member
SET row_status = 'NORMAL';

ALTER TABLE principal ENABLE TRIGGER update_principal_updated_ts;
ALTER TABLE member ENABLE TRIGGER update_member_updated_ts;
