-- schema migration seeding data
INSERT INTO "public"."principal" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "type", "name", "email", "password_hash") VALUES
(1, 'NORMAL', 1, 1657272778, 1, 1657272778, 'SYSTEM_BOT', 'Bytebase', 'support@bytebase.com', ''),
(101, 'NORMAL', 1, 1657272815, 1, 1657272815, 'END_USER', 'Demo', 'demo@example.com', '$2a$10$/65QFlHOmDzXshEMt/qYuunbJrXtRLcaYDcRODbyOPa/9/N0N8Zc2');

INSERT INTO "public"."member" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "status", "role", "principal_id") VALUES
(101, 'NORMAL', 1, 1657272815, 1, 1657272815, 'ACTIVE', 'OWNER', 101);

INSERT INTO "public"."project" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "name", "key", "workflow_type", "visibility", "tenant_mode", "db_name_template", "role_provider") VALUES
(1, 'NORMAL', 1, 1657272778, 1, 1657272778, 'Default', 'DEFAULT', 'UI', 'PUBLIC', 'DISABLED', '', 'BYTEBASE'),
(101, 'NORMAL', 101, 1657272873, 101, 1657272873, 'DEMO', 'Z77', 'UI', 'PUBLIC', 'DISABLED', '', 'BYTEBASE');

INSERT INTO "public"."project_member" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "project_id", "role", "principal_id", "role_provider", "payload") VALUES
(101, 'NORMAL', 101, 1657272873, 101, 1657272873, 101, 'OWNER', 101, 'BYTEBASE', '{}');

INSERT INTO "public"."environment" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "name", "order") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 'Test', 0),
(102, 'NORMAL', 1, 1657272778, 1, 1657272778, 'Prod', 1);

INSERT INTO "public"."policy" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "environment_id", "type", "payload") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 'bb.policy.pipeline-approval', '{"value": "MANUAL_APPROVAL_NEVER"}'),
(102, 'NORMAL', 1, 1657272778, 1, 1657272778, 102, 'bb.policy.pipeline-approval', '{"value": "MANUAL_APPROVAL_ALWAYS"}'),
(103, 'NORMAL', 1, 1657272778, 1, 1657272778, 102, 'bb.policy.backup-plan', '{"schedule": "WEEKLY"}');

INSERT INTO "public"."instance" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "environment_id", "name", "engine", "engine_version", "host", "port", "external_link") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 'Sample Test instance', 'POSTGRES', '14.3', 'host.docker.internal', '5432', ''),
(102, 'NORMAL', 101, 1657272848, 1, 1657272850, 101, 'DEMO', 'MYSQL', '8.0.28', 'demo.cfxzcrq5mf2d.us-west-1.rds.amazonaws.com', '', '');

INSERT INTO "public"."instance_user" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "instance_id", "name", "grant") VALUES
(101, 'NORMAL', 1, 1657272852, 1, 1657274591, 102, '''admin''@''%''', 'GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, RELOAD, PROCESS, REFERENCES, INDEX, ALTER, SHOW DATABASES, CREATE TEMPORARY TABLES, LOCK TABLES, EXECUTE, REPLICATION SLAVE, REPLICATION CLIENT, CREATE VIEW, SHOW VIEW, CREATE ROUTINE, ALTER ROUTINE, CREATE USER, EVENT, TRIGGER ON *.* TO `admin`@`%` WITH GRANT OPTION'),
(102, 'NORMAL', 1, 1657272852, 1, 1657274591, 102, '''rdsadmin''@''localhost''', 'GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, RELOAD, SHUTDOWN, PROCESS, FILE, REFERENCES, INDEX, ALTER, SHOW DATABASES, SUPER, CREATE TEMPORARY TABLES, LOCK TABLES, EXECUTE, REPLICATION SLAVE, REPLICATION CLIENT, CREATE VIEW, SHOW VIEW, CREATE ROUTINE, ALTER ROUTINE, CREATE USER, EVENT, TRIGGER, CREATE TABLESPACE, CREATE ROLE, DROP ROLE ON *.* TO `rdsadmin`@`localhost` WITH GRANT OPTION
GRANT SERVICE_CONNECTION_ADMIN,SET_USER_ID,SYSTEM_USER ON *.* TO `rdsadmin`@`localhost` WITH GRANT OPTION');

INSERT INTO "public"."db" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "instance_id", "project_id", "source_backup_id", "sync_status", "last_successful_sync_ts", "schema_version", "name", "character_set", "collation") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 1, NULL, 'OK', 0, '', '*', 'utf8mb4', 'utf8mb4_general_ci'),
(102, 'NORMAL', 101, 1657272848, 101, 1657272848, 102, 1, NULL, 'OK', 1657272848, '', '*', 'utf8mb4', 'utf8mb4_general_ci'),
(103, 'NORMAL', 1, 1657272853, 1, 1657274591, 102, 101, NULL, 'OK', 1657274591, '20220708094434', 'employee', 'utf8mb4', 'utf8mb4_general_ci');

INSERT INTO "public"."tbl" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "database_id", "name", "type", "engine", "collation", "row_count", "data_size", "index_size", "data_free", "create_options", "comment") VALUES
(126, 'NORMAL', 1, 1657190787, 1, 0, 103, 'department', 'BASE TABLE', 'InnoDB', 'utf8mb4_0900_ai_ci', 0, 16384, 16384, 0, '', ''),
(127, 'NORMAL', 1, 1657190788, 1, 0, 103, 'salary', 'BASE TABLE', 'InnoDB', 'utf8mb4_0900_ai_ci', 0, 16384, 0, 0, '', ''),
(128, 'NORMAL', 1, 1657190789, 1, 0, 103, 'title', 'BASE TABLE', 'InnoDB', 'utf8mb4_0900_ai_ci', 0, 16384, 0, 0, '', ''),
(129, 'NORMAL', 1, 1657190933, 1, 0, 103, 'dept_manager', 'BASE TABLE', 'InnoDB', 'utf8mb4_0900_ai_ci', 0, 16384, 16384, 0, '', ''),
(130, 'NORMAL', 1, 1657190939, 1, 0, 103, 'dept_emp', 'BASE TABLE', 'InnoDB', 'utf8mb4_0900_ai_ci', 0, 16384, 16384, 0, '', ''),
(131, 'NORMAL', 1, 1657272992, 1, 0, 103, 'drift_demo', 'BASE TABLE', 'InnoDB', 'utf8mb4_general_ci', 0, 16384, 0, 0, '', ''),
(132, 'NORMAL', 1, 1657273483, 1, 0, 103, 'employee', 'BASE TABLE', 'InnoDB', 'utf8mb4_0900_ai_ci', 0, 16384, 0, 0, '', '');

INSERT INTO "public"."col" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "database_id", "table_id", "name", "position", "default", "nullable", "type", "character_set", "collation", "comment") VALUES
(216, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 126, 'dept_name', 2, NULL, 'f', 'varchar(40)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(217, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 126, 'dept_no', 1, NULL, 'f', 'char(4)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(218, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 127, 'amount', 2, NULL, 'f', 'int', '', '', ''),
(219, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 127, 'emp_no', 1, NULL, 'f', 'int', '', '', ''),
(220, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 127, 'from_date', 3, NULL, 'f', 'date', '', '', ''),
(221, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 127, 'to_date', 4, NULL, 'f', 'date', '', '', ''),
(222, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 128, 'emp_no', 1, NULL, 'f', 'int', '', '', ''),
(223, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 128, 'from_date', 3, NULL, 'f', 'date', '', '', ''),
(224, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 128, 'title', 2, NULL, 'f', 'varchar(50)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(225, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 128, 'to_date', 4, NULL, 'f', 'date', '', '', ''),
(226, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 129, 'dept_no', 2, NULL, 'f', 'char(4)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(227, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 129, 'emp_no', 1, NULL, 'f', 'int', '', '', ''),
(228, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 129, 'from_date', 3, NULL, 'f', 'date', '', '', ''),
(229, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 129, 'to_date', 4, NULL, 'f', 'date', '', '', ''),
(230, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 130, 'dept_no', 2, NULL, 'f', 'char(4)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(231, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 130, 'emp_no', 1, NULL, 'f', 'int', '', '', ''),
(232, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 130, 'from_date', 3, NULL, 'f', 'date', '', '', ''),
(233, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 130, 'to_date', 4, NULL, 'f', 'date', '', '', ''),
(234, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 131, 'drift_date', 2, NULL, 'f', 'date', '', '', ''),
(235, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 131, 'drift_no', 1, NULL, 'f', 'char(4)', 'utf8mb4', 'utf8mb4_general_ci', ''),
(236, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'birth_date', 2, NULL, 'f', 'date', '', '', ''),
(237, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'Credentials_No', 11, NULL, 'f', 'varchar(10)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(238, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'Credentials_type', 10, NULL, 'f', 'char(4)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(239, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'EduBackgrounds', 7, NULL, 'f', 'varchar(50)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(240, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'Email', 8, NULL, 'f', 'varchar(40)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(241, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'Emergency_name', 13, NULL, 'f', 'varchar(16)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(242, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'Emergency_phoneNo', 14, NULL, 'f', 'varchar(15)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(243, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'emp_no', 1, NULL, 'f', 'int', '', '', ''),
(244, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'first_name', 3, NULL, 'f', 'varchar(14)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(245, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'gender', 5, NULL, 'f', 'enum(''M'',''F'')', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(246, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'hire_date', 6, NULL, 'f', 'date', '', '', ''),
(247, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'last_name', 4, NULL, 'f', 'varchar(16)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(248, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'Manager', 12, NULL, 'f', 'varchar(14)', 'utf8mb4', 'utf8mb4_0900_ai_ci', ''),
(249, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'PhoneNo', 9, NULL, 'f', 'varchar(15)', 'utf8mb4', 'utf8mb4_0900_ai_ci', '');

INSERT INTO "public"."idx" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "database_id", "table_id", "name", "expression", "position", "type", "unique", "visible", "comment") VALUES
(158, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 126, 'PRIMARY', 'dept_no', 1, 'BTREE', 't', 't', ''),
(159, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 126, 'dept_name', 'dept_name', 1, 'BTREE', 't', 't', ''),
(160, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 127, 'PRIMARY', 'emp_no', 1, 'BTREE', 't', 't', ''),
(161, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 127, 'PRIMARY', 'from_date', 2, 'BTREE', 't', 't', ''),
(162, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 128, 'PRIMARY', 'emp_no', 1, 'BTREE', 't', 't', ''),
(163, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 128, 'PRIMARY', 'title', 2, 'BTREE', 't', 't', ''),
(164, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 128, 'PRIMARY', 'from_date', 3, 'BTREE', 't', 't', ''),
(165, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 129, 'PRIMARY', 'emp_no', 1, 'BTREE', 't', 't', ''),
(166, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 129, 'PRIMARY', 'dept_no', 2, 'BTREE', 't', 't', ''),
(167, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 129, 'dept_no', 'dept_no', 1, 'BTREE', 'f', 't', ''),
(168, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 130, 'PRIMARY', 'emp_no', 1, 'BTREE', 't', 't', ''),
(169, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 130, 'PRIMARY', 'dept_no', 2, 'BTREE', 't', 't', ''),
(170, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 130, 'dept_no', 'dept_no', 1, 'BTREE', 'f', 't', ''),
(171, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 131, 'PRIMARY', 'drift_no', 1, 'BTREE', 't', 't', ''),
(172, 'NORMAL', 1, 1657274592, 1, 1657274592, 103, 132, 'PRIMARY', 'emp_no', 1, 'BTREE', 't', 't', '');

INSERT INTO "public"."data_source" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "instance_id", "database_id", "name", "type", "username", "password", "ssl_key", "ssl_cert", "ssl_ca") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 101, 'Admin data source', 'ADMIN', 'root', '', '', '', ''),
(102, 'NORMAL', 101, 1657272848, 101, 1657272848, 102, 102, 'Admin data source', 'ADMIN', 'admin', 'Bytebase12345', '', '', '');

INSERT INTO "public"."pipeline" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "name", "status") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 'Pipeline - Hello world', 'OPEN'),
(102, 'NORMAL', 101, 1657272890, 101, 1657272890, 'Establish database baseline pipeline', 'OPEN'),
(103, 'NORMAL', 101, 1657272927, 101, 1657272945, 'Update database schema pipeline', 'DONE'),
(104, 'NORMAL', 101, 1657273475, 101, 1657273475, 'Update database schema pipeline', 'OPEN');

INSERT INTO "public"."stage" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "pipeline_id", "environment_id", "name") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 101, 'Test'),
(102, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 102, 'Prod'),
(103, 'NORMAL', 101, 1657272890, 101, 1657272890, 102, 101, 'Test employee'),
(104, 'NORMAL', 101, 1657272927, 101, 1657272927, 103, 101, 'Test employee'),
(105, 'NORMAL', 101, 1657273475, 101, 1657273475, 104, 101, 'Test employee');

INSERT INTO "public"."task" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "pipeline_id", "stage_id", "instance_id", "database_id", "name", "status", "type", "payload", "earliest_allowed_ts") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 101, 101, NULL, 'Welcome', 'RUNNING', 'bb.task.general', '{"statement": "SELECT ''Welcome Builders'';"}', 0),
(102, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 101, 101, NULL, 'Let''s go', 'PENDING_APPROVAL', 'bb.task.general', '{"statement": "SELECT ''Let''s go'';"}', 0),
(103, 'NORMAL', 101, 1657272890, 1, 1657272902, 102, 103, 102, 103, 'Establish "employee" baseline', 'DONE', 'bb.task.database.schema.update', '{"statement": "/* Establish baseline using current schema */", "migrationType": "BASELINE", "schemaVersion": "20220708093449"}', 0),
(104, 'NORMAL', 101, 1657272927, 1, 1657272938, 103, 104, 102, 103, 'Update "employee" schema', 'DONE', 'bb.task.database.schema.update', '{"statement": "Alter table\n  employee\nadd\n  column (\n    EduBackgrounds varchar(50) not null,\n    Email varchar(40) not null,\n    PhoneNo varchar(15) not null,\n    Credentials_type char(4) not null,\n    Credentials_No varchar(10) not null,\n    Manager varchar(14),\n    Emergency_name varchar(16) not null,\n    Emergency_phoneNo varchar(15) not null,\n    Address varchar(60) not null\n  )", "migrationType": "MIGRATE", "schemaVersion": "20220708093526"}', 0),
(105, 'NORMAL', 101, 1657273475, 1, 1657273487, 104, 105, 102, 103, 'Update "employee" schema', 'DONE', 'bb.task.database.schema.update', '{"statement": "alter table\n  employee drop column Address;", "migrationType": "MIGRATE", "schemaVersion": "20220708094434"}', 0);

INSERT INTO "public"."task_run" ("id", "creator_id", "created_ts", "updater_id", "updated_ts", "task_id", "name", "status", "type", "code", "comment", "result", "payload") VALUES
(101, 1, 1657272778, 1, 1657272778, 101, 'Welcome', 'FAILED', 'bb.task.general', 0, '', '{"detail": "Something is not right..."}', '{}'),
(102, 1, 1657272778, 1, 1657272778, 101, 'Welcome', 'RUNNING', 'bb.task.general', 0, 'Let''s give another try', '{}', '{}'),
(103, 1, 1657272893, 1, 1657272902, 103, 'Establish "employee" baseline 1657272892', 'DONE', 'bb.task.database.schema.update', 0, '', '{"detail": "Established baseline version 20220708093449 for database \"employee\".", "version": "20220708093449", "migrationId": 1}', '{"statement": "/* Establish baseline using current schema */", "migrationType": "BASELINE", "schemaVersion": "20220708093449"}'),
(104, 1, 1657272929, 1, 1657272938, 104, 'Update "employee" schema 1657272928', 'DONE', 'bb.task.database.schema.update', 0, '', '{"detail": "Applied migration version 20220708093526 to database \"employee\".", "version": "20220708093526", "migrationId": 2}', '{"statement": "Alter table\n  employee\nadd\n  column (\n    EduBackgrounds varchar(50) not null,\n    Email varchar(40) not null,\n    PhoneNo varchar(15) not null,\n    Credentials_type char(4) not null,\n    Credentials_No varchar(10) not null,\n    Manager varchar(14),\n    Emergency_name varchar(16) not null,\n    Emergency_phoneNo varchar(15) not null,\n    Address varchar(60) not null\n  )", "migrationType": "MIGRATE", "schemaVersion": "20220708093526"}'),
(105, 1, 1657273477, 1, 1657273487, 105, 'Update "employee" schema 1657273476', 'DONE', 'bb.task.database.schema.update', 0, '', '{"detail": "Applied migration version 20220708094434 to database \"employee\".", "version": "20220708094434", "migrationId": 3}', '{"statement": "alter table\n  employee drop column Address;", "migrationType": "MIGRATE", "schemaVersion": "20220708094434"}');

INSERT INTO "public"."task_check_run" ("id", "creator_id", "created_ts", "updater_id", "updated_ts", "task_id", "status", "type", "code", "comment", "result", "payload") VALUES
(101, 1, 1657272890, 1, 1657272892, 103, 'DONE', 'bb.task-check.database.connect', 0, '', '{"resultList": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"employee\""}]}', '{}'),
(102, 1, 1657272890, 1, 1657272892, 103, 'DONE', 'bb.task-check.instance.migration-schema', 0, '', '{"resultList": [{"title": "OK", "status": "SUCCESS", "content": "Instance \"DEMO\" has setup migration schema"}]}', '{}'),
(103, 1, 1657272890, 1, 1657272891, 103, 'DONE', 'bb.task-check.database.statement.syntax', 0, '', '{"resultList": [{"title": "Syntax OK", "status": "SUCCESS", "content": "OK"}]}', '{"dbType": "MYSQL", "charset": "utf8mb4", "collation": "utf8mb4_general_ci", "statement": "/* Establish baseline using current schema */"}'),
(104, 1, 1657272890, 1, 1657272891, 103, 'DONE', 'bb.task-check.database.statement.advise', 0, '', '{"resultList": [{"code": 401, "title": "Empty SQL review policy or disabled", "status": "WARN"}]}', '{"dbType": "MYSQL", "charset": "utf8mb4", "collation": "utf8mb4_general_ci", "statement": "/* Establish baseline using current schema */"}'),
(105, 1, 1657272927, 1, 1657272929, 104, 'DONE', 'bb.task-check.database.connect', 0, '', '{"resultList": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"employee\""}]}', '{}'),
(106, 1, 1657272927, 1, 1657272929, 104, 'DONE', 'bb.task-check.instance.migration-schema', 0, '', '{"resultList": [{"title": "OK", "status": "SUCCESS", "content": "Instance \"DEMO\" has setup migration schema"}]}', '{}'),
(107, 1, 1657272927, 1, 1657272928, 104, 'DONE', 'bb.task-check.database.statement.syntax', 0, '', '{"resultList": [{"title": "Syntax OK", "status": "SUCCESS", "content": "OK"}]}', '{"dbType": "MYSQL", "charset": "utf8mb4", "collation": "utf8mb4_general_ci", "statement": "Alter table\n  employee\nadd\n  column (\n    EduBackgrounds varchar(50) not null,\n    Email varchar(40) not null,\n    PhoneNo varchar(15) not null,\n    Credentials_type char(4) not null,\n    Credentials_No varchar(10) not null,\n    Manager varchar(14),\n    Emergency_name varchar(16) not null,\n    Emergency_phoneNo varchar(15) not null,\n    Address varchar(60) not null\n  )"}'),
(108, 1, 1657272927, 1, 1657272928, 104, 'DONE', 'bb.task-check.database.statement.advise', 0, '', '{"resultList": [{"code": 401, "title": "Empty SQL review policy or disabled", "status": "WARN"}]}', '{"dbType": "MYSQL", "charset": "utf8mb4", "collation": "utf8mb4_general_ci", "statement": "Alter table\n  employee\nadd\n  column (\n    EduBackgrounds varchar(50) not null,\n    Email varchar(40) not null,\n    PhoneNo varchar(15) not null,\n    Credentials_type char(4) not null,\n    Credentials_No varchar(10) not null,\n    Manager varchar(14),\n    Emergency_name varchar(16) not null,\n    Emergency_phoneNo varchar(15) not null,\n    Address varchar(60) not null\n  )"}'),
(109, 1, 1657273475, 1, 1657273477, 105, 'DONE', 'bb.task-check.database.connect', 0, '', '{"resultList": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"employee\""}]}', '{}'),
(110, 1, 1657273475, 1, 1657273477, 105, 'DONE', 'bb.task-check.instance.migration-schema', 0, '', '{"resultList": [{"title": "OK", "status": "SUCCESS", "content": "Instance \"DEMO\" has setup migration schema"}]}', '{}'),
(111, 1, 1657273475, 1, 1657273476, 105, 'DONE', 'bb.task-check.database.statement.syntax', 0, '', '{"resultList": [{"title": "Syntax OK", "status": "SUCCESS", "content": "OK"}]}', '{"dbType": "MYSQL", "charset": "utf8mb4", "collation": "utf8mb4_general_ci", "statement": "alter table\n  employee drop column Address;"}'),
(112, 1, 1657273475, 1, 1657273476, 105, 'DONE', 'bb.task-check.database.statement.advise', 0, '', '{"resultList": [{"code": 401, "title": "Empty SQL review policy or disabled", "status": "WARN"}]}', '{"dbType": "MYSQL", "charset": "utf8mb4", "collation": "utf8mb4_general_ci", "statement": "alter table\n  employee drop column Address;"}');

INSERT INTO "public"."issue" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "project_id", "pipeline_id", "name", "status", "type", "description", "assignee_id", "payload") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 1, 101, 'Hello world!', 'OPEN', 'bb.issue.general', 'Welcome to Bytebase, this is the issue interface where developers and DBAs collaborate on database schema management issues such as: 
 - Creating a new database
 - Creating a table
 - Creating an index
 - Adding/Altering a column
 - Troubleshooting performance issue
Let''s try some simple tasks:
1. Bookmark this issue by clicking the star icon on the top of this page
2. Leave a comment below to greet future customers.', 1, '{}'),
(102, 'NORMAL', 101, 1657272890, 101, 1657272890, 101, 102, 'Establish "employee" baseline', 'OPEN', 'bb.issue.database.schema.update', '', 101, '{}'),
(103, 'NORMAL', 101, 1657272927, 101, 1657272945, 101, 103, '[employee] Alter schema', 'DONE', 'bb.issue.database.schema.update', '', 101, '{}'),
(104, 'NORMAL', 101, 1657273475, 101, 1657273475, 101, 104, '[employee] Alter schema', 'OPEN', 'bb.issue.database.schema.update', '', 101, '{}');

INSERT INTO "public"."activity" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "container_id", "type", "level", "comment", "payload") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 'bb.issue.create', 'INFO', '', '{}'),
(102, 'NORMAL', 1, 1657272778, 1, 1657272778, 101, 'bb.issue.comment.create', 'INFO', 'Go fish!', '{}'),
(103, 'NORMAL', 101, 1657272815, 101, 1657272815, 101, 'bb.member.create', 'INFO', '', '{"role": "OWNER", "principalId": 101, "memberStatus": "ACTIVE", "principalName": "Demo", "principalEmail": "demo@example.com"}'),
(104, 'NORMAL', 101, 1657272880, 101, 1657272880, 1, 'bb.project.database.transfer', 'INFO', 'Transferred out database "employee" to project "DEMO".', '{"databaseId": 103, "databaseName": "employee"}'),
(105, 'NORMAL', 101, 1657272880, 101, 1657272880, 101, 'bb.project.database.transfer', 'INFO', 'Transferred in database "employee" from project "Default".', '{"databaseId": 103, "databaseName": "employee"}'),
(106, 'NORMAL', 101, 1657272890, 101, 1657272890, 102, 'bb.issue.create', 'INFO', '', '{"issueName": "Establish \"employee\" baseline"}'),
(107, 'NORMAL', 1, 1657272893, 1, 1657272893, 102, 'bb.pipeline.task.status.update', 'INFO', '', '{"taskId": 103, "taskName": "Establish \"employee\" baseline", "issueName": "Establish \"employee\" baseline", "newStatus": "RUNNING", "oldStatus": "PENDING"}'),
(108, 'NORMAL', 1, 1657272902, 1, 1657272902, 102, 'bb.pipeline.task.status.update', 'INFO', '', '{"taskId": 103, "taskName": "Establish \"employee\" baseline", "issueName": "Establish \"employee\" baseline", "newStatus": "DONE", "oldStatus": "RUNNING"}'),
(109, 'NORMAL', 101, 1657272927, 101, 1657272927, 103, 'bb.issue.create', 'INFO', '', '{"issueName": "[employee] Alter schema"}'),
(110, 'NORMAL', 1, 1657272929, 1, 1657272929, 103, 'bb.pipeline.task.status.update', 'INFO', '', '{"taskId": 104, "taskName": "Update \"employee\" schema", "issueName": "[employee] Alter schema", "newStatus": "RUNNING", "oldStatus": "PENDING"}'),
(111, 'NORMAL', 1, 1657272938, 1, 1657272938, 103, 'bb.pipeline.task.status.update', 'INFO', '', '{"taskId": 104, "taskName": "Update \"employee\" schema", "issueName": "[employee] Alter schema", "newStatus": "DONE", "oldStatus": "RUNNING"}'),
(112, 'NORMAL', 101, 1657272945, 101, 1657272945, 103, 'bb.issue.status.update', 'INFO', '', '{"issueName": "[employee] Alter schema", "newStatus": "DONE", "oldStatus": "OPEN"}'),
(113, 'NORMAL', 101, 1657273475, 101, 1657273475, 104, 'bb.issue.create', 'INFO', '', '{"issueName": "[employee] Alter schema"}'),
(114, 'NORMAL', 1, 1657273477, 1, 1657273477, 104, 'bb.pipeline.task.status.update', 'INFO', '', '{"taskId": 105, "taskName": "Update \"employee\" schema", "issueName": "[employee] Alter schema", "newStatus": "RUNNING", "oldStatus": "PENDING"}'),
(115, 'NORMAL', 1, 1657273487, 1, 1657273487, 104, 'bb.pipeline.task.status.update', 'INFO', '', '{"taskId": 105, "taskName": "Update \"employee\" schema", "issueName": "[employee] Alter schema", "newStatus": "DONE", "oldStatus": "RUNNING"}');

INSERT INTO "public"."inbox" ("id", "receiver_id", "activity_id", "status") VALUES
(101, 101, 106, 'UNREAD'),
(102, 101, 109, 'UNREAD'),
(103, 101, 112, 'UNREAD'),
(104, 101, 113, 'UNREAD');

INSERT INTO "public"."anomaly" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "instance_id", "database_id", "type", "payload") VALUES
(101, 'NORMAL', 1, 1657273385, 1, 1657275185, 101, NULL, 'bb.anomaly.instance.connection', '{"detail": "failed to connect database at host.docker.internal:5432 with user \"root\": cannot connecting instance, make sure the connection info is correct"}'),
(102, 'NORMAL', 1, 1657273399, 1, 1657273399, 102, 103, 'bb.anomaly.database.schema.drift', '{"actual": "--\n-- Table structure for `department`\n--\nCREATE TABLE `department` (\n  `dept_no` char(4) NOT NULL,\n  `dept_name` varchar(40) NOT NULL,\n  PRIMARY KEY (`dept_no`),\n  UNIQUE KEY `dept_name` (`dept_name`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `dept_emp`\n--\nCREATE TABLE `dept_emp` (\n  `emp_no` int NOT NULL,\n  `dept_no` char(4) NOT NULL,\n  `from_date` date NOT NULL,\n  `to_date` date NOT NULL,\n  PRIMARY KEY (`emp_no`,`dept_no`),\n  KEY `dept_no` (`dept_no`),\n  CONSTRAINT `dept_emp_ibfk_1` FOREIGN KEY (`emp_no`) REFERENCES `employee` (`emp_no`) ON DELETE CASCADE,\n  CONSTRAINT `dept_emp_ibfk_2` FOREIGN KEY (`dept_no`) REFERENCES `department` (`dept_no`) ON DELETE CASCADE\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `dept_manager`\n--\nCREATE TABLE `dept_manager` (\n  `emp_no` int NOT NULL,\n  `dept_no` char(4) NOT NULL,\n  `from_date` date NOT NULL,\n  `to_date` date NOT NULL,\n  PRIMARY KEY (`emp_no`,`dept_no`),\n  KEY `dept_no` (`dept_no`),\n  CONSTRAINT `dept_manager_ibfk_1` FOREIGN KEY (`emp_no`) REFERENCES `employee` (`emp_no`) ON DELETE CASCADE,\n  CONSTRAINT `dept_manager_ibfk_2` FOREIGN KEY (`dept_no`) REFERENCES `department` (`dept_no`) ON DELETE CASCADE\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `drift_demo`\n--\nCREATE TABLE `drift_demo` (\n  `drift_no` char(4) COLLATE utf8mb4_general_ci NOT NULL,\n  `drift_date` date DEFAULT NULL,\n  PRIMARY KEY (`drift_no`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\n--\n-- Table structure for `employee`\n--\nCREATE TABLE `employee` (\n  `emp_no` int NOT NULL,\n  `birth_date` date NOT NULL,\n  `first_name` varchar(14) NOT NULL,\n  `last_name` varchar(16) NOT NULL,\n  `gender` enum(''M'',''F'') NOT NULL,\n  `hire_date` date NOT NULL,\n  `EduBackgrounds` varchar(50) NOT NULL,\n  `Email` varchar(40) NOT NULL,\n  `PhoneNo` varchar(15) NOT NULL,\n  `Credentials_type` char(4) NOT NULL,\n  `Credentials_No` varchar(10) NOT NULL,\n  `Manager` varchar(14) DEFAULT NULL,\n  `Emergency_name` varchar(16) NOT NULL,\n  `Emergency_phoneNo` varchar(15) NOT NULL,\n  `Address` varchar(60) NOT NULL,\n  PRIMARY KEY (`emp_no`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `salary`\n--\nCREATE TABLE `salary` (\n  `emp_no` int NOT NULL,\n  `amount` int NOT NULL,\n  `from_date` date NOT NULL,\n  `to_date` date NOT NULL,\n  PRIMARY KEY (`emp_no`,`from_date`),\n  CONSTRAINT `salary_ibfk_1` FOREIGN KEY (`emp_no`) REFERENCES `employee` (`emp_no`) ON DELETE CASCADE\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `title`\n--\nCREATE TABLE `title` (\n  `emp_no` int NOT NULL,\n  `title` varchar(50) NOT NULL,\n  `from_date` date NOT NULL,\n  `to_date` date DEFAULT NULL,\n  PRIMARY KEY (`emp_no`,`title`,`from_date`),\n  CONSTRAINT `title_ibfk_1` FOREIGN KEY (`emp_no`) REFERENCES `employee` (`emp_no`) ON DELETE CASCADE\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n", "expect": "--\n-- Table structure for `department`\n--\nCREATE TABLE `department` (\n  `dept_no` char(4) NOT NULL,\n  `dept_name` varchar(40) NOT NULL,\n  PRIMARY KEY (`dept_no`),\n  UNIQUE KEY `dept_name` (`dept_name`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `dept_emp`\n--\nCREATE TABLE `dept_emp` (\n  `emp_no` int NOT NULL,\n  `dept_no` char(4) NOT NULL,\n  `from_date` date NOT NULL,\n  `to_date` date NOT NULL,\n  PRIMARY KEY (`emp_no`,`dept_no`),\n  KEY `dept_no` (`dept_no`),\n  CONSTRAINT `dept_emp_ibfk_1` FOREIGN KEY (`emp_no`) REFERENCES `employee` (`emp_no`) ON DELETE CASCADE,\n  CONSTRAINT `dept_emp_ibfk_2` FOREIGN KEY (`dept_no`) REFERENCES `department` (`dept_no`) ON DELETE CASCADE\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `dept_manager`\n--\nCREATE TABLE `dept_manager` (\n  `emp_no` int NOT NULL,\n  `dept_no` char(4) NOT NULL,\n  `from_date` date NOT NULL,\n  `to_date` date NOT NULL,\n  PRIMARY KEY (`emp_no`,`dept_no`),\n  KEY `dept_no` (`dept_no`),\n  CONSTRAINT `dept_manager_ibfk_1` FOREIGN KEY (`emp_no`) REFERENCES `employee` (`emp_no`) ON DELETE CASCADE,\n  CONSTRAINT `dept_manager_ibfk_2` FOREIGN KEY (`dept_no`) REFERENCES `department` (`dept_no`) ON DELETE CASCADE\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `employee`\n--\nCREATE TABLE `employee` (\n  `emp_no` int NOT NULL,\n  `birth_date` date NOT NULL,\n  `first_name` varchar(14) NOT NULL,\n  `last_name` varchar(16) NOT NULL,\n  `gender` enum(''M'',''F'') NOT NULL,\n  `hire_date` date NOT NULL,\n  `EduBackgrounds` varchar(50) NOT NULL,\n  `Email` varchar(40) NOT NULL,\n  `PhoneNo` varchar(15) NOT NULL,\n  `Credentials_type` char(4) NOT NULL,\n  `Credentials_No` varchar(10) NOT NULL,\n  `Manager` varchar(14) DEFAULT NULL,\n  `Emergency_name` varchar(16) NOT NULL,\n  `Emergency_phoneNo` varchar(15) NOT NULL,\n  `Address` varchar(60) NOT NULL,\n  PRIMARY KEY (`emp_no`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `salary`\n--\nCREATE TABLE `salary` (\n  `emp_no` int NOT NULL,\n  `amount` int NOT NULL,\n  `from_date` date NOT NULL,\n  `to_date` date NOT NULL,\n  PRIMARY KEY (`emp_no`,`from_date`),\n  CONSTRAINT `salary_ibfk_1` FOREIGN KEY (`emp_no`) REFERENCES `employee` (`emp_no`) ON DELETE CASCADE\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n--\n-- Table structure for `title`\n--\nCREATE TABLE `title` (\n  `emp_no` int NOT NULL,\n  `title` varchar(50) NOT NULL,\n  `from_date` date NOT NULL,\n  `to_date` date DEFAULT NULL,\n  PRIMARY KEY (`emp_no`,`title`,`from_date`),\n  CONSTRAINT `title_ibfk_1` FOREIGN KEY (`emp_no`) REFERENCES `employee` (`emp_no`) ON DELETE CASCADE\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n\n", "version": "20220708093526"}');

INSERT INTO "public"."label_key" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "key") VALUES
(101, 'NORMAL', 1, 1657272778, 1, 1657272778, 'bb.location'),
(102, 'NORMAL', 1, 1657272778, 1, 1657272778, 'bb.tenant');

INSERT INTO "public"."setting" ("id", "row_status", "creator_id", "created_ts", "updater_id", "updated_ts", "name", "value", "description") VALUES
(101, 'NORMAL', 1, 1657272785, 1, 1657272785, 'bb.branding.logo', '', 'The branding logo image in base64 string format.'),
(102, 'NORMAL', 1, 1657272785, 1, 1657272785, 'bb.auth.secret', 'qOfnyO0qpMdqThSRdsDvLgQcL8me45EC', 'Random string used to sign the JWT auth token.'),
(103, 'NORMAL', 1, 1657272785, 1, 1657272785, 'bb.workspace.id', 'cd0d7ce7-e7a6-4bbd-b922-d6716829f032', 'The workspace identifier'),
(104, 'NORMAL', 1, 1657272785, 101, 1657272865, 'bb.enterprise.license', 'eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IlRFQU0iLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo0ODExMTA3NTg0LCJpYXQiOjE2NTUzNjQ5NTQsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiNjc2MjA4NTMuNDI0Mjk4MTQifQ.tQHbKBkcG6DEvTnSbd4HK9ysyk4nCUiiCUhRXXOxBTa0aesJW8if35FKS-x6Aw7bOyQ8EEw8o_18qYidWmiWzSss_jKrgwnDLBsHMFdpdTYdnh9nJfiS-52UBbGC1x8fy7k_owmAnnQ7mta4Vc8u8rIa022iCIzb6xiAszvZ0NJD4lGl3MlX5T7R7kFV3nuDy4BZW8m2N_X6Hj4bLFdBSJbrVx_z831tPaymkE2o2NHCNYghDgQgAhYGp9ligOT2w9lg3hB9wCVWItFiGbx2kqVmkEwrvH6OfSgfAORsNbM6oXjtnJd0fd5PzTYy3woLHyVGvyX2pVNKTmOEIt7Abw', 'Enterprise license');

