ALTER TABLE principal DISABLE TRIGGER update_principal_updated_ts;
ALTER TABLE setting DISABLE TRIGGER update_setting_updated_ts;
ALTER TABLE member DISABLE TRIGGER update_member_updated_ts;
ALTER TABLE environment DISABLE TRIGGER update_environment_updated_ts;
ALTER TABLE policy DISABLE TRIGGER update_policy_updated_ts;
ALTER TABLE project DISABLE TRIGGER update_project_updated_ts;
ALTER TABLE project_member DISABLE TRIGGER update_project_member_updated_ts;
ALTER TABLE project_webhook DISABLE TRIGGER update_project_webhook_updated_ts;
ALTER TABLE instance DISABLE TRIGGER update_instance_updated_ts;
ALTER TABLE instance_user DISABLE TRIGGER update_instance_user_updated_ts;
ALTER TABLE db DISABLE TRIGGER update_db_updated_ts;
ALTER TABLE db_schema DISABLE TRIGGER update_db_schema_updated_ts;
ALTER TABLE data_source DISABLE TRIGGER update_data_source_updated_ts;
ALTER TABLE backup DISABLE TRIGGER update_backup_updated_ts;
ALTER TABLE backup_setting DISABLE TRIGGER update_backup_setting_updated_ts;
ALTER TABLE pipeline DISABLE TRIGGER update_pipeline_updated_ts;
ALTER TABLE stage DISABLE TRIGGER update_stage_updated_ts;
ALTER TABLE task DISABLE TRIGGER update_task_updated_ts;
ALTER TABLE task_run DISABLE TRIGGER update_task_run_updated_ts;
ALTER TABLE task_check_run DISABLE TRIGGER update_task_check_run_updated_ts;
ALTER TABLE issue DISABLE TRIGGER update_issue_updated_ts;
ALTER TABLE instance_change_history DISABLE TRIGGER update_instance_change_history_updated_ts;
ALTER TABLE activity DISABLE TRIGGER update_activity_updated_ts;
ALTER TABLE bookmark DISABLE TRIGGER update_bookmark_updated_ts;
ALTER TABLE vcs DISABLE TRIGGER update_vcs_updated_ts;
ALTER TABLE repository DISABLE TRIGGER update_repository_updated_ts;
ALTER TABLE anomaly DISABLE TRIGGER update_anomaly_updated_ts;
ALTER TABLE label_key DISABLE TRIGGER update_label_key_updated_ts;
ALTER TABLE label_value DISABLE TRIGGER update_label_value_updated_ts;
ALTER TABLE db_label DISABLE TRIGGER update_db_label_updated_ts;
ALTER TABLE deployment_config DISABLE TRIGGER update_deployment_config_updated_ts;
ALTER TABLE sheet DISABLE TRIGGER update_sheet_updated_ts;
ALTER TABLE external_approval DISABLE TRIGGER update_external_approval_updated_ts;
ALTER TABLE risk DISABLE TRIGGER update_risk_updated_ts;
ALTER TABLE slow_query DISABLE TRIGGER update_slow_query_updated_ts;

DO $$
DECLARE
  row_data RECORD;
  first_user RECORD;
  row_exists NUMERIC;
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
  UPDATE principal SET creator_id = 1 WHERE creator_id = row_data.id;
  UPDATE principal SET updater_id = 1 WHERE updater_id = row_data.id;
  UPDATE setting SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE setting SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE member SET creator_id = 1 WHERE creator_id = row_data.id;
  UPDATE member SET updater_id = 1 WHERE updater_id = row_data.id;
  UPDATE environment SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE environment SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE policy SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE policy SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE project SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE project SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE project_member SET creator_id = 1 WHERE creator_id = row_data.id;
  UPDATE project_member SET updater_id = 1 WHERE updater_id = row_data.id;
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
  UPDATE task_check_run SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE task_check_run SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE issue SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE issue SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE issue SET assignee_id = first_user.id WHERE assignee_id = row_data.id;
  UPDATE issue_subscriber SET subscriber_id = first_user.id WHERE subscriber_id = row_data.id;
  UPDATE instance_change_history SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE instance_change_history SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE activity SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE activity SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE inbox SET receiver_id = first_user.id WHERE receiver_id = row_data.id;
  UPDATE bookmark SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE bookmark SET updater_id = first_user.id WHERE updater_id = row_data.id;
  UPDATE vcs SET creator_id = first_user.id WHERE creator_id = row_data.id;
  UPDATE vcs SET updater_id = first_user.id WHERE updater_id = row_data.id;
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

  SELECT 1 INTO row_exists FROM project_member WHERE principal_id = first_user.id;
  IF (row_exists > 0) THEN
    DELETE FROM project_member WHERE principal_id = row_data.id;
  ELSE
    UPDATE project_member SET principal_id = first_user.id, creator_id = 1, updater_id = 1 WHERE principal_id = row_data.id;
  END IF;

  DELETE FROM member WHERE principal_id = row_data.id;
  DELETE FROM principal WHERE id = row_data.id;
END LOOP;
END $$;

ALTER TABLE principal ENABLE TRIGGER update_principal_updated_ts;
ALTER TABLE setting ENABLE TRIGGER update_setting_updated_ts;
ALTER TABLE member ENABLE TRIGGER update_member_updated_ts;
ALTER TABLE environment ENABLE TRIGGER update_environment_updated_ts;
ALTER TABLE policy ENABLE TRIGGER update_policy_updated_ts;
ALTER TABLE project ENABLE TRIGGER update_project_updated_ts;
ALTER TABLE project_member ENABLE TRIGGER update_project_member_updated_ts;
ALTER TABLE project_webhook ENABLE TRIGGER update_project_webhook_updated_ts;
ALTER TABLE instance ENABLE TRIGGER update_instance_updated_ts;
ALTER TABLE instance_user ENABLE TRIGGER update_instance_user_updated_ts;
ALTER TABLE db ENABLE TRIGGER update_db_updated_ts;
ALTER TABLE db_schema ENABLE TRIGGER update_db_schema_updated_ts;
ALTER TABLE data_source ENABLE TRIGGER update_data_source_updated_ts;
ALTER TABLE backup ENABLE TRIGGER update_backup_updated_ts;
ALTER TABLE backup_setting ENABLE TRIGGER update_backup_setting_updated_ts;
ALTER TABLE pipeline ENABLE TRIGGER update_pipeline_updated_ts;
ALTER TABLE stage ENABLE TRIGGER update_stage_updated_ts;
ALTER TABLE task ENABLE TRIGGER update_task_updated_ts;
ALTER TABLE task_run ENABLE TRIGGER update_task_run_updated_ts;
ALTER TABLE task_check_run ENABLE TRIGGER update_task_check_run_updated_ts;
ALTER TABLE issue ENABLE TRIGGER update_issue_updated_ts;
ALTER TABLE instance_change_history ENABLE TRIGGER update_instance_change_history_updated_ts;
ALTER TABLE activity ENABLE TRIGGER update_activity_updated_ts;
ALTER TABLE bookmark ENABLE TRIGGER update_bookmark_updated_ts;
ALTER TABLE vcs ENABLE TRIGGER update_vcs_updated_ts;
ALTER TABLE repository ENABLE TRIGGER update_repository_updated_ts;
ALTER TABLE anomaly ENABLE TRIGGER update_anomaly_updated_ts;
ALTER TABLE label_key ENABLE TRIGGER update_label_key_updated_ts;
ALTER TABLE label_value ENABLE TRIGGER update_label_value_updated_ts;
ALTER TABLE db_label ENABLE TRIGGER update_db_label_updated_ts;
ALTER TABLE deployment_config ENABLE TRIGGER update_deployment_config_updated_ts;
ALTER TABLE sheet ENABLE TRIGGER update_sheet_updated_ts;
ALTER TABLE external_approval ENABLE TRIGGER update_external_approval_updated_ts;
ALTER TABLE risk ENABLE TRIGGER update_risk_updated_ts;
ALTER TABLE slow_query ENABLE TRIGGER update_slow_query_updated_ts;
