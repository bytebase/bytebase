ALTER TABLE activity DROP CONSTRAINT IF EXISTS activity_container_id_check;

ALTER TABLE activity ADD CONSTRAINT activity_container_id_check CHECK (container_id >= 0);

UPDATE activity SET container_id = 0 WHERE "type" LIKE 'bb.member.%';
