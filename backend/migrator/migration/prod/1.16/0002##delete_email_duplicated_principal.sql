BEGIN;
DO $$
DECLARE
  row_data RECORD;
  first_user RECORD;
BEGIN
FOR row_data IN (
  SELECT
    email,
    id
  FROM (
    SELECT
      email,
      id,
      ROW_NUMBER() OVER (PARTITION BY email ORDER BY id) AS rn
    FROM
      principal) AS temp
  WHERE
    temp.rn > 1
)
LOOP
  SELECT * INTO first_user FROM principal WHERE email = row_data.email ORDER BY id LIMIT 1;
  UPDATE setting SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE setting SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE environment SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE environment SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE policy SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE policy SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE project SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE project SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE project_webhook SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE project_webhook SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE instance SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE instance SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE instance_user SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE instance_user SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE db SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE db SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE db_schema SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE db_schema SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE data_source SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE data_source SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE backup SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE backup SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE backup_setting SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE backup_setting SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE pipeline SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE pipeline SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE stage SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE stage SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE task SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE task SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE task_run SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE task_run SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE issue SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE issue SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE issue SET assignee_id = first_user.id WHERE assignee_id = row_data.id;
  UPDATE instance_change_history SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE instance_change_history SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE activity SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE activity SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE inbox SET receiver_id = first_user.id WHERE receiver_id = row_data.id;
  UPDATE bookmark SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE bookmark SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE repository SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE repository SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE anomaly SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE anomaly SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE label_key SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE label_key SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE label_value SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE label_value SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE db_label SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE db_label SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE deployment_config SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE deployment_config SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE sheet SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE sheet SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE sheet_organizer SET principal_id = first_user.id WHERE principal_id = row_data.id;
  UPDATE external_approval SET requester_id = first_user.id WHERE requester_id = row_data.id;
  UPDATE external_approval SET approver_id = first_user.id WHERE approver_id = row_data.id;
  UPDATE risk SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE risk SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE slow_query SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE slow_query SET updater_id = first_user.id WHERE updater_id = row_data.id;

  DELETE FROM member WHERE principal_id = row_data.id;
  DELETE FROM project_member WHERE principal_id = row_data.id;
  DELETE FROM principal WHERE id = row_data.id;
END LOOP;
END $$;
COMMIT;
