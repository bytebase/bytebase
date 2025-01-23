UPDATE db
SET metadata = json_build_object('labels', json_build_object('bb.tenant', db_label.value))
FROM db_label
WHERE db.id = db_label.database_id AND db_label.key = 'bb.tenant';