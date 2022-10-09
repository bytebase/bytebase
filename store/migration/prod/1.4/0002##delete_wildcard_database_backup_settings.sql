DELETE FROM backup_setting
WHERE id IN
(
  SELECT
    backup_setting.id
  FROM backup_setting
  INNER JOIN db ON backup_setting.database_id = db.id
  WHERE db.name = '*'
);