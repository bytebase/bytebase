--
-- PostgreSQL database dump
--

-- Dumped from database version 16.0
-- Dumped by pg_dump version 16.0

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: principal; Type: TABLE DATA; Schema: public; Owner: bbdev
--

SET SESSION AUTHORIZATION DEFAULT;

ALTER TABLE public.principal DISABLE TRIGGER ALL;

INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config) VALUES (1, 'NORMAL', 1, 1698730968, 1, 1698730968, 'SYSTEM_BOT', 'Bytebase', 'support@bytebase.com', '', '', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config) VALUES (101, 'NORMAL', 1, 1698731517, 1, 1698731517, 'END_USER', 'Demo', 'demo@example.com', '$2a$10$lxxI34zueCq8Dqyh1EDaJe4p6rARa/kYR0UzX66XZcZw5nN7lZpXa', '', '{}') ON CONFLICT DO NOTHING;


ALTER TABLE public.principal ENABLE TRIGGER ALL;

--
-- Data for Name: activity; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.activity DISABLE TRIGGER ALL;

INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 'bb.issue.create', 'INFO', '', '{"issueName": "👉👉👉 [START HERE] Add email column to Employee table"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (102, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 'bb.member.create', 'INFO', '', '{"role": "OWNER", "principalId": 101, "memberStatus": "ACTIVE", "principalName": "Demo", "principalEmail": "demo@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (103, 'NORMAL', 101, 1698731875, 101, 1698731875, 102, 'bb.sql-editor.query', 'INFO', 'Executed `"SELECT * FROM salary;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM salary;", "adviceList": null, "databaseId": 102, "durationNs": 9676000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (104, 'NORMAL', 101, 1698731901, 101, 1698731901, 102, 'bb.sql-editor.query', 'INFO', 'Executed `"SELECT * FROM salary;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM salary;", "adviceList": null, "databaseId": 102, "durationNs": 6699000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (105, 'NORMAL', 101, 1698731928, 101, 1698731928, 102, 'bb.sql-editor.query', 'INFO', 'Executed `"SELECT * FROM salary;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM salary;", "adviceList": null, "databaseId": 102, "durationNs": 6502000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;


ALTER TABLE public.activity ENABLE TRIGGER ALL;

--
-- Data for Name: environment; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.environment DISABLE TRIGGER ALL;

INSERT INTO public.environment (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order", resource_id) VALUES (101, 'NORMAL', 1, 1698730968, 1, 1698730968, 'Test', 0, 'test') ON CONFLICT DO NOTHING;
INSERT INTO public.environment (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order", resource_id) VALUES (102, 'NORMAL', 1, 1698730968, 101, 1698732589, 'Prod', 1, 'prod') ON CONFLICT DO NOTHING;


ALTER TABLE public.environment ENABLE TRIGGER ALL;

--
-- Data for Name: instance; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.instance DISABLE TRIGGER ALL;

INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (102, 'NORMAL', 101, 1698731517, 1, 1698732039, 102, 'Prod Sample Instance', 'POSTGRES', '16.0', '', 'prod-sample-instance', true, '{}', '{"lastSyncTime": "2023-10-31T06:00:38.764018Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (101, 'NORMAL', 101, 1698731517, 1, 1698732044, 101, 'Test Sample Instance', 'POSTGRES', '16.0', '', 'test-sample-instance', true, '{}', '{"lastSyncTime": "2023-10-31T06:00:43.939711Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (103, 'NORMAL', 101, 1698732181, 101, 1698732181, 101, 'Pseudo MySQL Instance', 'MYSQL', '', '', 'pseudo-mysql-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (104, 'NORMAL', 101, 1698732229, 101, 1698732229, 101, 'Pseudo Oracle Instance', 'ORACLE', '', '', 'pseudo-oracle-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (105, 'NORMAL', 101, 1698732264, 101, 1698732264, 101, 'Pseudo SQL Server Instance', 'MSSQL', '', '', 'pseudo-sql-server-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (106, 'NORMAL', 101, 1698732296, 101, 1698732296, 101, 'Pseudo Snowflake Instance', 'SNOWFLAKE', '', '', 'pseudo-snowflake-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (107, 'NORMAL', 101, 1698732313, 101, 1698732313, 101, 'Pseudo ClickHouse Instance', 'CLICKHOUSE', '', '', 'pseudo-clickhouse-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (108, 'NORMAL', 101, 1698732329, 101, 1698732329, 101, 'Pseudo MongoDB Instance', 'MONGODB', '', '', 'pseudo-mongodb-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (109, 'NORMAL', 101, 1698732378, 101, 1698732378, 101, 'Pseudo Redis Instance', 'REDIS', '', '', 'pseudo-redis-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (110, 'NORMAL', 101, 1698732400, 101, 1698732400, 101, 'Pseudo TiDB Instance', 'TIDB', '', '', 'pseudo-tidb-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (111, 'NORMAL', 101, 1698732428, 101, 1698732428, 101, 'Pseudo OceanBase Instance', 'OCEANBASE', '', '', 'pseudo-oceanbase-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (112, 'NORMAL', 101, 1698732462, 101, 1698732462, 101, 'Pseudo Spanner Instance', 'SPANNER', '', '', 'pseudo-spanner-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (113, 'NORMAL', 101, 1698732482, 101, 1698732482, 101, 'Pseudo Redshift Instance', 'REDSHIFT', '', '', 'pseudo-redshift-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (114, 'NORMAL', 101, 1698732511, 101, 1698732511, 101, 'Pseudo MariaDB Instance', 'MARIADB', '', '', 'pseudo-mariadb-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (115, 'NORMAL', 101, 1698732531, 101, 1698732531, 101, 'Pseudo RisingWave Instance', 'RISINGWAVE', '', '', 'pseudo-risingwave-instance', true, '{}', '{}') ON CONFLICT DO NOTHING;


ALTER TABLE public.instance ENABLE TRIGGER ALL;

--
-- Data for Name: project; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.project DISABLE TRIGGER ALL;

INSERT INTO public.project (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, workflow_type, visibility, tenant_mode, db_name_template, schema_change_type, resource_id, data_classification_config_id, setting, schema_version_type) VALUES (1, 'NORMAL', 1, 1698730968, 1, 1698730968, 'Default', 'DEFAULT', 'UI', 'PUBLIC', 'DISABLED', '', 'DDL', 'default', '', '{}', 'TIMESTAMP') ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, workflow_type, visibility, tenant_mode, db_name_template, schema_change_type, resource_id, data_classification_config_id, setting, schema_version_type) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 'Sample Project', 'SAM', 'UI', 'PUBLIC', 'DISABLED', '', 'DDL', 'project-sample', '', '{}', 'TIMESTAMP') ON CONFLICT DO NOTHING;


ALTER TABLE public.project ENABLE TRIGGER ALL;

--
-- Data for Name: db; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.db DISABLE TRIGGER ALL;

INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment_id, source_backup_id, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (102, 'NORMAL', 1, 1698731517, 1, 1698732039, 102, 101, NULL, NULL, 'OK', 1698732038, '', 'hr_prod', '{}', false, '', '{"lastSyncTime": "2023-10-31T06:00:38Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment_id, source_backup_id, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (101, 'NORMAL', 1, 1698731517, 1, 1698732044, 101, 101, NULL, NULL, 'OK', 1698732043, '', 'hr_test', '{}', false, '', '{"lastSyncTime": "2023-10-31T06:00:43Z"}') ON CONFLICT DO NOTHING;


ALTER TABLE public.db ENABLE TRIGGER ALL;

--
-- Data for Name: anomaly; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.anomaly DISABLE TRIGGER ALL;



ALTER TABLE public.anomaly ENABLE TRIGGER ALL;

--
-- Data for Name: backup; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.backup DISABLE TRIGGER ALL;



ALTER TABLE public.backup ENABLE TRIGGER ALL;

--
-- Data for Name: backup_setting; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.backup_setting DISABLE TRIGGER ALL;



ALTER TABLE public.backup_setting ENABLE TRIGGER ALL;

--
-- Data for Name: bookmark; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.bookmark DISABLE TRIGGER ALL;

INSERT INTO public.bookmark (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, link) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 'Sample Issue', '/issue/start-here-add-email-column-to-employee-table-101') ON CONFLICT DO NOTHING;


ALTER TABLE public.bookmark ENABLE TRIGGER ALL;

--
-- Data for Name: changelist; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.changelist DISABLE TRIGGER ALL;



ALTER TABLE public.changelist ENABLE TRIGGER ALL;

--
-- Data for Name: data_source; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.data_source DISABLE TRIGGER ALL;

INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 'admin', 'ADMIN', 'bbsample', '', '', '', '', '/tmp', '446', '{}', 'hr_test') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (102, 'NORMAL', 101, 1698731517, 101, 1698731517, 102, 'admin', 'ADMIN', 'bbsample', '', '', '', '', '/tmp', '447', '{}', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (103, 'NORMAL', 101, 1698732181, 101, 1698732181, 103, 'c1586918-0626-43b4-a537-4753f9f0ad54', 'ADMIN', '', '', '', '', '', '127.0.0.1', '3306', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (104, 'NORMAL', 101, 1698732229, 101, 1698732229, 104, 'f119a7a1-56c0-489c-8251-c45dd10a7112', 'ADMIN', '', '', '', '', '', '127.0.0.1', '1521', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (105, 'NORMAL', 101, 1698732264, 101, 1698732264, 105, '60ad6436-d9cb-4aa8-a49c-1e1ba63cda5d', 'ADMIN', '', '', '', '', '', '127.0.0.1', '1433', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (106, 'NORMAL', 101, 1698732296, 101, 1698732296, 106, 'f75e9908-687e-42a0-94d6-723d4aedad15', 'ADMIN', '', '', '', '', '', 'bytebase', '443', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (107, 'NORMAL', 101, 1698732313, 101, 1698732313, 107, '7436bb7b-b483-4428-93ae-1f8b8488c54c', 'ADMIN', '', '', '', '', '', '127.0.0.1', '9000', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (108, 'NORMAL', 101, 1698732329, 101, 1698732329, 108, '60378ece-6fb6-441d-8a8c-3f7f215be368', 'ADMIN', '', '', '', '', '', '127.0.0.1', '27017', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (109, 'NORMAL', 101, 1698732378, 101, 1698732378, 109, '070848c8-f7a2-4874-aa33-35821afa71b3', 'ADMIN', '', '', '', '', '', '127.0.0.1', '6379', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (110, 'NORMAL', 101, 1698732400, 101, 1698732400, 110, 'f9748880-8aac-4df5-aec1-a9f1641ef1ee', 'ADMIN', '', '', '', '', '', '127.0.0.1', '4000', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (111, 'NORMAL', 101, 1698732428, 101, 1698732428, 111, '6c6bb571-dfaa-4eef-8439-f1078d3fd249', 'ADMIN', '', '', '', '', '', '127.0.0.1', '2883', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (112, 'NORMAL', 101, 1698732462, 101, 1698732462, 112, '194db4c6-e6ba-493b-b489-cb9aa0464c39', 'ADMIN', '', 'VF9QA2NB', '', '', '', 'projects/bytebase/instances/bytebase', '3306', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (113, 'NORMAL', 101, 1698732482, 101, 1698732482, 113, '553db6c4-cc2d-41f6-91c5-d100fd9dad4b', 'ADMIN', '', '', '', '', '', '127.0.0.1', '5439', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (114, 'NORMAL', 101, 1698732511, 101, 1698732511, 114, 'f0cd37fc-6a21-4520-b8ce-4c2342571d0d', 'ADMIN', '', '', '', '', '', '127.0.0.1', '3306', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (115, 'NORMAL', 101, 1698732531, 101, 1698732531, 115, 'ac71e9dd-73b7-4a6c-8bf8-1162769df26d', 'ADMIN', '', '', '', '', '', '127.0.0.1', '3306', '{}', '') ON CONFLICT DO NOTHING;


ALTER TABLE public.data_source ENABLE TRIGGER ALL;

--
-- Data for Name: db_group; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.db_group DISABLE TRIGGER ALL;



ALTER TABLE public.db_group ENABLE TRIGGER ALL;

--
-- Data for Name: db_schema; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.db_schema DISABLE TRIGGER ALL;

INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (101, 'NORMAL', 1, 1698731517, 1, 1698731517, 101, '{"name": "hr_test", "schemas": [{"name": "public", "views": [{"name": "dept_emp_latest_date", "definition": " SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM dept_emp\n  GROUP BY emp_no;", "dependentColumns": [{"table": "dept_emp", "column": "emp_no", "schema": "public"}, {"table": "dept_emp", "column": "from_date", "schema": "public"}, {"table": "dept_emp", "column": "to_date", "schema": "public"}]}, {"name": "current_dept_emp", "definition": " SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (dept_emp d\n     JOIN dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));", "dependentColumns": [{"table": "dept_emp", "column": "dept_no", "schema": "public"}, {"table": "dept_emp", "column": "emp_no", "schema": "public"}, {"table": "dept_emp", "column": "from_date", "schema": "public"}, {"table": "dept_emp", "column": "to_date", "schema": "public"}, {"table": "dept_emp_latest_date", "column": "emp_no", "schema": "public"}, {"table": "dept_emp_latest_date", "column": "from_date", "schema": "public"}, {"table": "dept_emp_latest_date", "column": "to_date", "schema": "public"}]}, {"name": "pg_stat_statements_info", "definition": " SELECT dealloc,\n    stats_reset\n   FROM pg_stat_statements_info() pg_stat_statements_info(dealloc, stats_reset);"}, {"name": "pg_stat_statements", "definition": " SELECT userid,\n    dbid,\n    toplevel,\n    queryid,\n    query,\n    plans,\n    total_plan_time,\n    min_plan_time,\n    max_plan_time,\n    mean_plan_time,\n    stddev_plan_time,\n    calls,\n    total_exec_time,\n    min_exec_time,\n    max_exec_time,\n    mean_exec_time,\n    stddev_exec_time,\n    rows,\n    shared_blks_hit,\n    shared_blks_read,\n    shared_blks_dirtied,\n    shared_blks_written,\n    local_blks_hit,\n    local_blks_read,\n    local_blks_dirtied,\n    local_blks_written,\n    temp_blks_read,\n    temp_blks_written,\n    blk_read_time,\n    blk_write_time,\n    temp_blk_read_time,\n    temp_blk_write_time,\n    wal_records,\n    wal_fpi,\n    wal_bytes,\n    jit_functions,\n    jit_generation_time,\n    jit_inlining_count,\n    jit_inlining_time,\n    jit_optimization_count,\n    jit_optimization_time,\n    jit_emission_count,\n    jit_emission_time\n   FROM pg_stat_statements(true) pg_stat_statements(userid, dbid, toplevel, queryid, query, plans, total_plan_time, min_plan_time, max_plan_time, mean_plan_time, stddev_plan_time, calls, total_exec_time, min_exec_time, max_exec_time, mean_exec_time, stddev_exec_time, rows, shared_blks_hit, shared_blks_read, shared_blks_dirtied, shared_blks_written, local_blks_hit, local_blks_read, local_blks_dirtied, local_blks_written, temp_blks_read, temp_blks_written, blk_read_time, blk_write_time, temp_blk_read_time, temp_blk_write_time, wal_records, wal_fpi, wal_bytes, jit_functions, jit_generation_time, jit_inlining_count, jit_inlining_time, jit_optimization_count, jit_optimization_time, jit_emission_count, jit_emission_time);"}], "tables": [{"name": "department", "columns": [{"name": "dept_no", "type": "text", "position": 1}, {"name": "dept_name", "type": "text", "position": 2}], "indexes": [{"name": "department_dept_name_key", "type": "btree", "unique": true, "expressions": ["dept_name"]}, {"name": "department_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["dept_no"]}], "dataSize": "16384", "indexSize": "32768"}, {"name": "dept_emp", "columns": [{"name": "emp_no", "type": "integer", "position": 1}, {"name": "dept_no", "type": "text", "position": 2}, {"name": "from_date", "type": "date", "position": 3}, {"name": "to_date", "type": "date", "position": 4}], "indexes": [{"name": "dept_emp_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no", "dept_no"]}], "dataSize": "106496", "rowCount": "1103", "indexSize": "57344", "foreignKeys": [{"name": "dept_emp_dept_no_fkey", "columns": ["dept_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "department", "referencedSchema": "public", "referencedColumns": ["dept_no"]}, {"name": "dept_emp_emp_no_fkey", "columns": ["emp_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "employee", "referencedSchema": "public", "referencedColumns": ["emp_no"]}]}, {"name": "dept_manager", "columns": [{"name": "emp_no", "type": "integer", "position": 1}, {"name": "dept_no", "type": "text", "position": 2}, {"name": "from_date", "type": "date", "position": 3}, {"name": "to_date", "type": "date", "position": 4}], "indexes": [{"name": "dept_manager_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no", "dept_no"]}], "dataSize": "16384", "indexSize": "16384", "foreignKeys": [{"name": "dept_manager_dept_no_fkey", "columns": ["dept_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "department", "referencedSchema": "public", "referencedColumns": ["dept_no"]}, {"name": "dept_manager_emp_no_fkey", "columns": ["emp_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "employee", "referencedSchema": "public", "referencedColumns": ["emp_no"]}]}, {"name": "employee", "columns": [{"name": "emp_no", "type": "integer", "position": 1, "defaultExpression": "nextval(''employee_emp_no_seq''::regclass)"}, {"name": "birth_date", "type": "date", "position": 2}, {"name": "first_name", "type": "text", "position": 3}, {"name": "last_name", "type": "text", "position": 4}, {"name": "gender", "type": "text", "position": 5}, {"name": "hire_date", "type": "date", "position": 6}], "indexes": [{"name": "employee_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no"]}], "dataSize": "98304", "rowCount": "1000", "indexSize": "40960"}, {"name": "salary", "columns": [{"name": "emp_no", "type": "integer", "position": 1}, {"name": "amount", "type": "integer", "position": 2}, {"name": "from_date", "type": "date", "position": 3}, {"name": "to_date", "type": "date", "position": 4}], "indexes": [{"name": "salary_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no", "from_date"]}], "dataSize": "458752", "rowCount": "9488", "indexSize": "229376", "foreignKeys": [{"name": "salary_emp_no_fkey", "columns": ["emp_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "employee", "referencedSchema": "public", "referencedColumns": ["emp_no"]}]}, {"name": "title", "columns": [{"name": "emp_no", "type": "integer", "position": 1}, {"name": "title", "type": "text", "position": 2}, {"name": "from_date", "type": "date", "position": 3}, {"name": "to_date", "type": "date", "nullable": true, "position": 4}], "indexes": [{"name": "title_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no", "title", "from_date"]}], "dataSize": "131072", "rowCount": "1470", "indexSize": "73728", "foreignKeys": [{"name": "title_emp_no_fkey", "columns": ["emp_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "employee", "referencedSchema": "public", "referencedColumns": ["emp_no"]}]}], "functions": [{"name": "pg_stat_statements", "definition": "CREATE OR REPLACE FUNCTION public.pg_stat_statements(showtext boolean, OUT userid oid, OUT dbid oid, OUT toplevel boolean, OUT queryid bigint, OUT query text, OUT plans bigint, OUT total_plan_time double precision, OUT min_plan_time double precision, OUT max_plan_time double precision, OUT mean_plan_time double precision, OUT stddev_plan_time double precision, OUT calls bigint, OUT total_exec_time double precision, OUT min_exec_time double precision, OUT max_exec_time double precision, OUT mean_exec_time double precision, OUT stddev_exec_time double precision, OUT rows bigint, OUT shared_blks_hit bigint, OUT shared_blks_read bigint, OUT shared_blks_dirtied bigint, OUT shared_blks_written bigint, OUT local_blks_hit bigint, OUT local_blks_read bigint, OUT local_blks_dirtied bigint, OUT local_blks_written bigint, OUT temp_blks_read bigint, OUT temp_blks_written bigint, OUT blk_read_time double precision, OUT blk_write_time double precision, OUT temp_blk_read_time double precision, OUT temp_blk_write_time double precision, OUT wal_records bigint, OUT wal_fpi bigint, OUT wal_bytes numeric, OUT jit_functions bigint, OUT jit_generation_time double precision, OUT jit_inlining_count bigint, OUT jit_inlining_time double precision, OUT jit_optimization_count bigint, OUT jit_optimization_time double precision, OUT jit_emission_count bigint, OUT jit_emission_time double precision)\n RETURNS SETOF record\n LANGUAGE c\n PARALLEL SAFE STRICT\nAS ''$libdir/pg_stat_statements'', $function$pg_stat_statements_1_10$function$\n"}, {"name": "pg_stat_statements_info", "definition": "CREATE OR REPLACE FUNCTION public.pg_stat_statements_info(OUT dealloc bigint, OUT stats_reset timestamp with time zone)\n RETURNS record\n LANGUAGE c\n PARALLEL SAFE STRICT\nAS ''$libdir/pg_stat_statements'', $function$pg_stat_statements_info$function$\n"}, {"name": "pg_stat_statements_reset", "definition": "CREATE OR REPLACE FUNCTION public.pg_stat_statements_reset(userid oid DEFAULT 0, dbid oid DEFAULT 0, queryid bigint DEFAULT 0)\n RETURNS void\n LANGUAGE c\n PARALLEL SAFE STRICT\nAS ''$libdir/pg_stat_statements'', $function$pg_stat_statements_reset_1_7$function$\n"}]}], "collation": "en_US.UTF-8", "extensions": [{"name": "pg_stat_statements", "schema": "public", "version": "1.10", "description": "track planning and execution statistics of all SQL statements executed"}], "characterSet": "UTF8"}', '
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = ''UTF8'';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config(''search_path'', '''', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

CREATE EXTENSION IF NOT EXISTS pg_stat_statements WITH SCHEMA public;

COMMENT ON EXTENSION pg_stat_statements IS ''track planning and execution statistics of all SQL statements executed'';

SET default_tablespace = '''';

SET default_table_access_method = heap;

CREATE TABLE public.dept_emp (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);

CREATE VIEW public.dept_emp_latest_date AS
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW public.current_dept_emp AS
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TABLE public.department (
    dept_no text NOT NULL,
    dept_name text NOT NULL
);

CREATE TABLE public.dept_manager (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);

CREATE TABLE public.employee (
    emp_no integer NOT NULL,
    birth_date date NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    gender text NOT NULL,
    hire_date date NOT NULL,
    CONSTRAINT employee_gender_check CHECK ((gender = ANY (ARRAY[''M''::text, ''F''::text])))
);

CREATE SEQUENCE public.employee_emp_no_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.employee_emp_no_seq OWNED BY public.employee.emp_no;

CREATE TABLE public.salary (
    emp_no integer NOT NULL,
    amount integer NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);

CREATE TABLE public.title (
    emp_no integer NOT NULL,
    title text NOT NULL,
    from_date date NOT NULL,
    to_date date
);

ALTER TABLE ONLY public.employee ALTER COLUMN emp_no SET DEFAULT nextval(''public.employee_emp_no_seq''::regclass);

ALTER TABLE ONLY public.department
    ADD CONSTRAINT department_dept_name_key UNIQUE (dept_name);

ALTER TABLE ONLY public.department
    ADD CONSTRAINT department_pkey PRIMARY KEY (dept_no);

ALTER TABLE ONLY public.dept_emp
    ADD CONSTRAINT dept_emp_pkey PRIMARY KEY (emp_no, dept_no);

ALTER TABLE ONLY public.dept_manager
    ADD CONSTRAINT dept_manager_pkey PRIMARY KEY (emp_no, dept_no);

ALTER TABLE ONLY public.employee
    ADD CONSTRAINT employee_pkey PRIMARY KEY (emp_no);

ALTER TABLE ONLY public.salary
    ADD CONSTRAINT salary_pkey PRIMARY KEY (emp_no, from_date);

ALTER TABLE ONLY public.title
    ADD CONSTRAINT title_pkey PRIMARY KEY (emp_no, title, from_date);

ALTER TABLE ONLY public.dept_emp
    ADD CONSTRAINT dept_emp_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES public.department(dept_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.dept_emp
    ADD CONSTRAINT dept_emp_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.dept_manager
    ADD CONSTRAINT dept_manager_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES public.department(dept_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.dept_manager
    ADD CONSTRAINT dept_manager_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.salary
    ADD CONSTRAINT salary_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.title
    ADD CONSTRAINT title_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (102, 'NORMAL', 1, 1698731517, 1, 1698731517, 102, '{"name": "hr_prod", "schemas": [{"name": "public", "views": [{"name": "dept_emp_latest_date", "definition": " SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM dept_emp\n  GROUP BY emp_no;", "dependentColumns": [{"table": "dept_emp", "column": "emp_no", "schema": "public"}, {"table": "dept_emp", "column": "from_date", "schema": "public"}, {"table": "dept_emp", "column": "to_date", "schema": "public"}]}, {"name": "current_dept_emp", "definition": " SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (dept_emp d\n     JOIN dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));", "dependentColumns": [{"table": "dept_emp", "column": "dept_no", "schema": "public"}, {"table": "dept_emp", "column": "emp_no", "schema": "public"}, {"table": "dept_emp", "column": "from_date", "schema": "public"}, {"table": "dept_emp", "column": "to_date", "schema": "public"}, {"table": "dept_emp_latest_date", "column": "emp_no", "schema": "public"}, {"table": "dept_emp_latest_date", "column": "from_date", "schema": "public"}, {"table": "dept_emp_latest_date", "column": "to_date", "schema": "public"}]}, {"name": "pg_stat_statements_info", "definition": " SELECT dealloc,\n    stats_reset\n   FROM pg_stat_statements_info() pg_stat_statements_info(dealloc, stats_reset);"}, {"name": "pg_stat_statements", "definition": " SELECT userid,\n    dbid,\n    toplevel,\n    queryid,\n    query,\n    plans,\n    total_plan_time,\n    min_plan_time,\n    max_plan_time,\n    mean_plan_time,\n    stddev_plan_time,\n    calls,\n    total_exec_time,\n    min_exec_time,\n    max_exec_time,\n    mean_exec_time,\n    stddev_exec_time,\n    rows,\n    shared_blks_hit,\n    shared_blks_read,\n    shared_blks_dirtied,\n    shared_blks_written,\n    local_blks_hit,\n    local_blks_read,\n    local_blks_dirtied,\n    local_blks_written,\n    temp_blks_read,\n    temp_blks_written,\n    blk_read_time,\n    blk_write_time,\n    temp_blk_read_time,\n    temp_blk_write_time,\n    wal_records,\n    wal_fpi,\n    wal_bytes,\n    jit_functions,\n    jit_generation_time,\n    jit_inlining_count,\n    jit_inlining_time,\n    jit_optimization_count,\n    jit_optimization_time,\n    jit_emission_count,\n    jit_emission_time\n   FROM pg_stat_statements(true) pg_stat_statements(userid, dbid, toplevel, queryid, query, plans, total_plan_time, min_plan_time, max_plan_time, mean_plan_time, stddev_plan_time, calls, total_exec_time, min_exec_time, max_exec_time, mean_exec_time, stddev_exec_time, rows, shared_blks_hit, shared_blks_read, shared_blks_dirtied, shared_blks_written, local_blks_hit, local_blks_read, local_blks_dirtied, local_blks_written, temp_blks_read, temp_blks_written, blk_read_time, blk_write_time, temp_blk_read_time, temp_blk_write_time, wal_records, wal_fpi, wal_bytes, jit_functions, jit_generation_time, jit_inlining_count, jit_inlining_time, jit_optimization_count, jit_optimization_time, jit_emission_count, jit_emission_time);"}], "tables": [{"name": "department", "columns": [{"name": "dept_no", "type": "text", "position": 1}, {"name": "dept_name", "type": "text", "position": 2}], "indexes": [{"name": "department_dept_name_key", "type": "btree", "unique": true, "expressions": ["dept_name"]}, {"name": "department_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["dept_no"]}], "dataSize": "16384", "indexSize": "32768"}, {"name": "dept_emp", "columns": [{"name": "emp_no", "type": "integer", "position": 1}, {"name": "dept_no", "type": "text", "position": 2}, {"name": "from_date", "type": "date", "position": 3}, {"name": "to_date", "type": "date", "position": 4}], "indexes": [{"name": "dept_emp_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no", "dept_no"]}], "dataSize": "106496", "rowCount": "1103", "indexSize": "57344", "foreignKeys": [{"name": "dept_emp_dept_no_fkey", "columns": ["dept_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "department", "referencedSchema": "public", "referencedColumns": ["dept_no"]}, {"name": "dept_emp_emp_no_fkey", "columns": ["emp_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "employee", "referencedSchema": "public", "referencedColumns": ["emp_no"]}]}, {"name": "dept_manager", "columns": [{"name": "emp_no", "type": "integer", "position": 1}, {"name": "dept_no", "type": "text", "position": 2}, {"name": "from_date", "type": "date", "position": 3}, {"name": "to_date", "type": "date", "position": 4}], "indexes": [{"name": "dept_manager_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no", "dept_no"]}], "dataSize": "16384", "indexSize": "16384", "foreignKeys": [{"name": "dept_manager_dept_no_fkey", "columns": ["dept_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "department", "referencedSchema": "public", "referencedColumns": ["dept_no"]}, {"name": "dept_manager_emp_no_fkey", "columns": ["emp_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "employee", "referencedSchema": "public", "referencedColumns": ["emp_no"]}]}, {"name": "employee", "columns": [{"name": "emp_no", "type": "integer", "position": 1, "defaultExpression": "nextval(''employee_emp_no_seq''::regclass)"}, {"name": "birth_date", "type": "date", "position": 2}, {"name": "first_name", "type": "text", "position": 3}, {"name": "last_name", "type": "text", "position": 4}, {"name": "gender", "type": "text", "position": 5}, {"name": "hire_date", "type": "date", "position": 6}], "indexes": [{"name": "employee_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no"]}], "dataSize": "98304", "rowCount": "1000", "indexSize": "40960"}, {"name": "salary", "columns": [{"name": "emp_no", "type": "integer", "position": 1}, {"name": "amount", "type": "integer", "position": 2}, {"name": "from_date", "type": "date", "position": 3}, {"name": "to_date", "type": "date", "position": 4}], "indexes": [{"name": "salary_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no", "from_date"]}], "dataSize": "458752", "rowCount": "9488", "indexSize": "229376", "foreignKeys": [{"name": "salary_emp_no_fkey", "columns": ["emp_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "employee", "referencedSchema": "public", "referencedColumns": ["emp_no"]}]}, {"name": "title", "columns": [{"name": "emp_no", "type": "integer", "position": 1}, {"name": "title", "type": "text", "position": 2}, {"name": "from_date", "type": "date", "position": 3}, {"name": "to_date", "type": "date", "nullable": true, "position": 4}], "indexes": [{"name": "title_pkey", "type": "btree", "unique": true, "primary": true, "expressions": ["emp_no", "title", "from_date"]}], "dataSize": "131072", "rowCount": "1470", "indexSize": "73728", "foreignKeys": [{"name": "title_emp_no_fkey", "columns": ["emp_no"], "onDelete": "CASCADE", "onUpdate": "NO ACTION", "matchType": "SIMPLE", "referencedTable": "employee", "referencedSchema": "public", "referencedColumns": ["emp_no"]}]}], "functions": [{"name": "pg_stat_statements", "definition": "CREATE OR REPLACE FUNCTION public.pg_stat_statements(showtext boolean, OUT userid oid, OUT dbid oid, OUT toplevel boolean, OUT queryid bigint, OUT query text, OUT plans bigint, OUT total_plan_time double precision, OUT min_plan_time double precision, OUT max_plan_time double precision, OUT mean_plan_time double precision, OUT stddev_plan_time double precision, OUT calls bigint, OUT total_exec_time double precision, OUT min_exec_time double precision, OUT max_exec_time double precision, OUT mean_exec_time double precision, OUT stddev_exec_time double precision, OUT rows bigint, OUT shared_blks_hit bigint, OUT shared_blks_read bigint, OUT shared_blks_dirtied bigint, OUT shared_blks_written bigint, OUT local_blks_hit bigint, OUT local_blks_read bigint, OUT local_blks_dirtied bigint, OUT local_blks_written bigint, OUT temp_blks_read bigint, OUT temp_blks_written bigint, OUT blk_read_time double precision, OUT blk_write_time double precision, OUT temp_blk_read_time double precision, OUT temp_blk_write_time double precision, OUT wal_records bigint, OUT wal_fpi bigint, OUT wal_bytes numeric, OUT jit_functions bigint, OUT jit_generation_time double precision, OUT jit_inlining_count bigint, OUT jit_inlining_time double precision, OUT jit_optimization_count bigint, OUT jit_optimization_time double precision, OUT jit_emission_count bigint, OUT jit_emission_time double precision)\n RETURNS SETOF record\n LANGUAGE c\n PARALLEL SAFE STRICT\nAS ''$libdir/pg_stat_statements'', $function$pg_stat_statements_1_10$function$\n"}, {"name": "pg_stat_statements_info", "definition": "CREATE OR REPLACE FUNCTION public.pg_stat_statements_info(OUT dealloc bigint, OUT stats_reset timestamp with time zone)\n RETURNS record\n LANGUAGE c\n PARALLEL SAFE STRICT\nAS ''$libdir/pg_stat_statements'', $function$pg_stat_statements_info$function$\n"}, {"name": "pg_stat_statements_reset", "definition": "CREATE OR REPLACE FUNCTION public.pg_stat_statements_reset(userid oid DEFAULT 0, dbid oid DEFAULT 0, queryid bigint DEFAULT 0)\n RETURNS void\n LANGUAGE c\n PARALLEL SAFE STRICT\nAS ''$libdir/pg_stat_statements'', $function$pg_stat_statements_reset_1_7$function$\n"}]}], "collation": "en_US.UTF-8", "extensions": [{"name": "pg_stat_statements", "schema": "public", "version": "1.10", "description": "track planning and execution statistics of all SQL statements executed"}], "characterSet": "UTF8"}', '
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = ''UTF8'';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config(''search_path'', '''', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

CREATE EXTENSION IF NOT EXISTS pg_stat_statements WITH SCHEMA public;

COMMENT ON EXTENSION pg_stat_statements IS ''track planning and execution statistics of all SQL statements executed'';

SET default_tablespace = '''';

SET default_table_access_method = heap;

CREATE TABLE public.dept_emp (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);

CREATE VIEW public.dept_emp_latest_date AS
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW public.current_dept_emp AS
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TABLE public.department (
    dept_no text NOT NULL,
    dept_name text NOT NULL
);

CREATE TABLE public.dept_manager (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);

CREATE TABLE public.employee (
    emp_no integer NOT NULL,
    birth_date date NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    gender text NOT NULL,
    hire_date date NOT NULL,
    CONSTRAINT employee_gender_check CHECK ((gender = ANY (ARRAY[''M''::text, ''F''::text])))
);

CREATE SEQUENCE public.employee_emp_no_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.employee_emp_no_seq OWNED BY public.employee.emp_no;

CREATE TABLE public.salary (
    emp_no integer NOT NULL,
    amount integer NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);

CREATE TABLE public.title (
    emp_no integer NOT NULL,
    title text NOT NULL,
    from_date date NOT NULL,
    to_date date
);

ALTER TABLE ONLY public.employee ALTER COLUMN emp_no SET DEFAULT nextval(''public.employee_emp_no_seq''::regclass);

ALTER TABLE ONLY public.department
    ADD CONSTRAINT department_dept_name_key UNIQUE (dept_name);

ALTER TABLE ONLY public.department
    ADD CONSTRAINT department_pkey PRIMARY KEY (dept_no);

ALTER TABLE ONLY public.dept_emp
    ADD CONSTRAINT dept_emp_pkey PRIMARY KEY (emp_no, dept_no);

ALTER TABLE ONLY public.dept_manager
    ADD CONSTRAINT dept_manager_pkey PRIMARY KEY (emp_no, dept_no);

ALTER TABLE ONLY public.employee
    ADD CONSTRAINT employee_pkey PRIMARY KEY (emp_no);

ALTER TABLE ONLY public.salary
    ADD CONSTRAINT salary_pkey PRIMARY KEY (emp_no, from_date);

ALTER TABLE ONLY public.title
    ADD CONSTRAINT title_pkey PRIMARY KEY (emp_no, title, from_date);

ALTER TABLE ONLY public.dept_emp
    ADD CONSTRAINT dept_emp_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES public.department(dept_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.dept_emp
    ADD CONSTRAINT dept_emp_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.dept_manager
    ADD CONSTRAINT dept_manager_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES public.department(dept_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.dept_manager
    ADD CONSTRAINT dept_manager_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.salary
    ADD CONSTRAINT salary_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

ALTER TABLE ONLY public.title
    ADD CONSTRAINT title_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

', '{}') ON CONFLICT DO NOTHING;


ALTER TABLE public.db_schema ENABLE TRIGGER ALL;

--
-- Data for Name: deployment_config; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.deployment_config DISABLE TRIGGER ALL;



ALTER TABLE public.deployment_config ENABLE TRIGGER ALL;

--
-- Data for Name: pipeline; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.pipeline DISABLE TRIGGER ALL;

INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;


ALTER TABLE public.pipeline ENABLE TRIGGER ALL;

--
-- Data for Name: plan; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.plan DISABLE TRIGGER ALL;

INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 101, 'Onboarding sample plan for adding email column to Employee table', '', '{"steps": [{"specs": [{"changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/102", "target": "instances/test-sample-instance/databases/hr_test"}}]}, {"specs": [{"changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/103", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}]}') ON CONFLICT DO NOTHING;


ALTER TABLE public.plan ENABLE TRIGGER ALL;

--
-- Data for Name: issue; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.issue DISABLE TRIGGER ALL;

INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (101, 'NORMAL', 101, 1698731517, 1, 1698731518, 101, 101, 101, '👉👉👉 [START HERE] Add email column to Employee table', 'OPEN', 'bb.issue.database.general', 'A sample issue to showcase how to review database schema change.

				Click "Approve" button to apply the schema update.', 101, false, '{"approval": {"approvalFindingDone": true}}', '''a'':9 ''add'':3 ''apply'':24 ''approve'':21 ''button'':22 ''change'':19 ''click'':20 ''column'':5 ''database'':17 ''email'':4 ''employee'':7 ''here'':2 ''how'':14 ''issue'':11 ''review'':16 ''sample'':10 ''schema'':18,26 ''showcase'':13 ''start'':1 ''table'':8 ''the'':25 ''to'':6,12,15,23 ''update'':27') ON CONFLICT DO NOTHING;


ALTER TABLE public.issue ENABLE TRIGGER ALL;

--
-- Data for Name: external_approval; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.external_approval DISABLE TRIGGER ALL;



ALTER TABLE public.external_approval ENABLE TRIGGER ALL;

--
-- Data for Name: idp; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.idp DISABLE TRIGGER ALL;



ALTER TABLE public.idp ENABLE TRIGGER ALL;

--
-- Data for Name: inbox; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.inbox DISABLE TRIGGER ALL;

INSERT INTO public.inbox (id, receiver_id, activity_id, status) VALUES (101, 101, 101, 'UNREAD') ON CONFLICT DO NOTHING;


ALTER TABLE public.inbox ENABLE TRIGGER ALL;

--
-- Data for Name: instance_change_history; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.instance_change_history DISABLE TRIGGER ALL;

INSERT INTO public.instance_change_history (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, issue_id, release_version, sequence, source, type, status, version, description, statement, sheet_id, schema, schema_prev, execution_duration_ns, payload) VALUES (101, 'NORMAL', 1, 1698730968, 1, 1698730968, NULL, NULL, NULL, 'development', 1, 'LIBRARY', 'MIGRATE', 'DONE', '0002.0010.0003-20231031134248', 'Initial migration version 2.10.3 server version development with file migration/prod/LATEST.sql.', '-- Type
CREATE TYPE row_status AS ENUM (''NORMAL'', ''ARCHIVED'');

-- updated_ts trigger.
CREATE OR REPLACE FUNCTION trigger_update_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = extract(epoch from now());
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- idp stores generic identity provider.
CREATE TABLE idp (
  id SERIAL PRIMARY KEY,
  row_status row_status NOT NULL DEFAULT ''NORMAL'',
  created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  resource_id TEXT NOT NULL,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  type TEXT NOT NULL CONSTRAINT idp_type_check CHECK (type IN (''OAUTH2'', ''OIDC'', ''LDAP'')),
  -- config stores the corresponding configuration of the IdP, which may vary depending on the type of the IdP.
  config JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON idp(resource_id);

ALTER SEQUENCE idp_id_seq RESTART WITH 101;

CREATE TRIGGER update_idp_updated_ts
BEFORE
UPDATE
    ON idp FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- principal
CREATE TABLE principal (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    type TEXT NOT NULL CHECK (type IN (''END_USER'', ''SYSTEM_BOT'', ''SERVICE_ACCOUNT'')),
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    phone TEXT NOT NULL DEFAULT '''',
    mfa_config JSONB NOT NULL DEFAULT ''{}''
);

CREATE TRIGGER update_principal_updated_ts
BEFORE
UPDATE
    ON principal FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Default bytebase system account id is 1
INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        type,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        1,
        ''SYSTEM_BOT'',
        ''Bytebase'',
        ''support@bytebase.com'',
        ''''
    );

ALTER SEQUENCE principal_id_seq RESTART WITH 101;

-- Setting
CREATE TABLE setting (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''''
);

CREATE UNIQUE INDEX idx_setting_unique_name ON setting(name);

ALTER SEQUENCE setting_id_seq RESTART WITH 101;

CREATE TRIGGER update_setting_updated_ts
BEFORE
UPDATE
    ON setting FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Role
CREATE TABLE role (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    resource_id TEXT NOT NULL, -- user-defined id, such as projectDBA
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    permissions JSONB NOT NULL DEFAULT ''{}'', -- saved for future use
    payload JSONB NOT NULL DEFAULT ''{}'' -- saved for future use
);

CREATE UNIQUE INDEX idx_role_unique_resource_id on role (resource_id);

ALTER SEQUENCE role_id_seq RESTART WITH 101;

CREATE TRIGGER update_role_updated_ts
BEFORE
UPDATE
    ON role FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Member
-- We separate the concept from Principal because if we support multiple workspace in the future, each workspace can have different member for the same principal
CREATE TABLE member (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    status TEXT NOT NULL CHECK (status IN (''INVITED'', ''ACTIVE'')),
    role TEXT NOT NULL CHECK (role IN (''OWNER'', ''DBA'', ''DEVELOPER'')),
    principal_id INTEGER NOT NULL REFERENCES principal (id)
);

CREATE UNIQUE INDEX idx_member_unique_principal_id ON member(principal_id);

ALTER SEQUENCE member_id_seq RESTART WITH 101;

CREATE TRIGGER update_member_updated_ts
BEFORE
UPDATE
    ON member FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Environment
CREATE TABLE environment (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    "order" INTEGER NOT NULL CHECK ("order" >= 0),
    resource_id TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_environment_unique_name ON environment(name);

CREATE UNIQUE INDEX idx_environment_unique_resource_id ON environment(resource_id);

ALTER SEQUENCE environment_id_seq RESTART WITH 101;

CREATE TRIGGER update_environment_updated_ts
BEFORE
UPDATE
    ON environment FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Policy
-- policy stores the policies for each environment.
-- Policies are associated with environments. Since we may have policies not associated with environment later, we name the table policy.
CREATE TYPE resource_type AS ENUM (''WORKSPACE'', ''ENVIRONMENT'', ''PROJECT'', ''INSTANCE'', ''DATABASE'');

CREATE TABLE policy (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    type TEXT NOT NULL CHECK (type LIKE ''bb.policy.%''),
    payload JSONB NOT NULL DEFAULT ''{}'',
    resource_type resource_type NOT NULL,
    resource_id INTEGER NOT NULL,
    inherit_from_parent BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_id_type ON policy(resource_type, resource_id, type);

ALTER SEQUENCE policy_id_seq RESTART WITH 101;

CREATE TRIGGER update_policy_updated_ts
BEFORE
UPDATE
    ON policy FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Project
CREATE TABLE project (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    key TEXT NOT NULL,
    workflow_type TEXT NOT NULL CHECK (workflow_type IN (''UI'', ''VCS'')),
    visibility TEXT NOT NULL CHECK (visibility IN (''PUBLIC'', ''PRIVATE'')),
    tenant_mode TEXT NOT NULL CHECK (tenant_mode IN (''DISABLED'', ''TENANT'')) DEFAULT ''DISABLED'',
    -- db_name_template is only used when a project is in tenant mode.
    -- Empty value means {{DB_NAME}}.
    db_name_template TEXT NOT NULL,
    schema_change_type TEXT NOT NULL CHECK (schema_change_type IN (''DDL'', ''SDL'')) DEFAULT ''DDL'',
    resource_id TEXT NOT NULL,
    data_classification_config_id TEXT NOT NULL DEFAULT '''',
    setting  JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_project_unique_key ON project(key);

CREATE UNIQUE INDEX idx_project_unique_resource_id ON project(resource_id);

INSERT INTO
    project (
        id,
        creator_id,
        updater_id,
        name,
        key,
        workflow_type,
        visibility,
        tenant_mode,
        db_name_template,
        resource_id
    )
VALUES
    (
        1,
        1,
        1,
        ''Default'',
        ''DEFAULT'',
        ''UI'',
        ''PUBLIC'',
        ''DISABLED'',
        '''',
        ''default''
    );

ALTER SEQUENCE project_id_seq RESTART WITH 101;

CREATE TRIGGER update_project_updated_ts
BEFORE
UPDATE
    ON project FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Project member
CREATE TABLE project_member (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    role TEXT NOT NULL,
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    condition JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_project_member_project_id ON project_member(project_id);

ALTER SEQUENCE project_member_id_seq RESTART WITH 101;

CREATE TRIGGER update_project_member_updated_ts
BEFORE
UPDATE
    ON project_member FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Project Hook
CREATE TABLE project_webhook (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    type TEXT NOT NULL CHECK (type LIKE ''bb.plugin.webhook.%''),
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    activity_list TEXT ARRAY NOT NULL
);

CREATE INDEX idx_project_webhook_project_id ON project_webhook(project_id);

CREATE UNIQUE INDEX idx_project_webhook_unique_project_id_url ON project_webhook(project_id, url);

ALTER SEQUENCE project_webhook_id_seq RESTART WITH 101;

CREATE TRIGGER update_project_webhook_updated_ts
BEFORE
UPDATE
    ON project_webhook FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Instance
CREATE TABLE instance (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    environment_id INTEGER REFERENCES environment (id),
    name TEXT NOT NULL,
    engine TEXT NOT NULL,
    engine_version TEXT NOT NULL DEFAULT '''',
    external_link TEXT NOT NULL DEFAULT '''',
    resource_id TEXT NOT NULL,
    -- activation should set to be TRUE if users assign license to this instance.
    activation BOOLEAN NOT NULL DEFAULT false,
    options JSONB NOT NULL DEFAULT ''{}'',
    metadata JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON instance(resource_id);

ALTER SEQUENCE instance_id_seq RESTART WITH 101;

CREATE TRIGGER update_instance_updated_ts
BEFORE
UPDATE
    ON instance FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Instance user stores the users for a particular instance
CREATE TABLE instance_user (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    name TEXT NOT NULL,
    "grant" TEXT NOT NULL
);

ALTER SEQUENCE instance_user_id_seq RESTART WITH 101;

CREATE UNIQUE INDEX idx_instance_user_unique_instance_id_name ON instance_user(instance_id, name);

CREATE TRIGGER update_instance_user_updated_ts
BEFORE
UPDATE
    ON instance_user FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- db stores the databases for a particular instance
-- data is synced periodically from the instance
CREATE TABLE db (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    environment_id INTEGER REFERENCES environment (id),
    -- If db is restored from a backup, then we will record that backup id. We can thus trace up to the original db.
    source_backup_id INTEGER,
    sync_status TEXT NOT NULL CHECK (sync_status IN (''OK'', ''NOT_FOUND'')),
    last_successful_sync_ts BIGINT NOT NULL,
    schema_version TEXT NOT NULL,
    name TEXT NOT NULL,
    secrets JSONB NOT NULL DEFAULT ''{}'',
    datashare BOOLEAN NOT NULL DEFAULT FALSE,
    -- service_name is the Oracle specific field.
    service_name TEXT NOT NULL DEFAULT '''',
    metadata JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_db_instance_id ON db(instance_id);

CREATE UNIQUE INDEX idx_db_unique_instance_id_name ON db(instance_id, name);

CREATE INDEX idx_db_project_id ON db(project_id);

ALTER SEQUENCE db_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_updated_ts
BEFORE
UPDATE
    ON db FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- db_schema stores the database schema metadata for a particular database.
CREATE TABLE db_schema (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id) ON DELETE CASCADE,
    metadata JSONB NOT NULL DEFAULT ''{}'',
    raw_dump TEXT NOT NULL DEFAULT '''',
    config JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_db_schema_unique_database_id ON db_schema(database_id);

ALTER SEQUENCE db_schema_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_schema_updated_ts
BEFORE
UPDATE
    ON db_schema FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- data_source table stores the data source for a particular database
CREATE TABLE data_source (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (''ADMIN'', ''RW'', ''RO'')),
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    ssl_key TEXT NOT NULL DEFAULT '''',
    ssl_cert TEXT NOT NULL DEFAULT '''',
    ssl_ca TEXT NOT NULL DEFAULT '''',
    host TEXT NOT NULL DEFAULT '''',
    port TEXT NOT NULL DEFAULT '''',
    options JSONB NOT NULL DEFAULT ''{}'',
    database TEXT NOT NULL DEFAULT ''''
);

CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON data_source(instance_id, name);

ALTER SEQUENCE data_source_id_seq RESTART WITH 101;

CREATE TRIGGER update_data_source_updated_ts
BEFORE
UPDATE
    ON data_source FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- backup stores the backups for a particular database.
CREATE TABLE backup (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN (''PENDING_CREATE'', ''DONE'', ''FAILED'')),
    type TEXT NOT NULL CHECK (type IN (''MANUAL'', ''AUTOMATIC'', ''PITR'')),
    storage_backend TEXT NOT NULL CHECK (storage_backend IN (''LOCAL'', ''S3'', ''GCS'', ''OSS'')),
    migration_history_version TEXT NOT NULL,
    path TEXT NOT NULL,
    comment TEXT NOT NULL DEFAULT '''',
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_backup_database_id ON backup(database_id);

CREATE UNIQUE INDEX idx_backup_unique_database_id_name ON backup(database_id, name);

ALTER SEQUENCE backup_id_seq RESTART WITH 101;

CREATE TRIGGER update_backup_updated_ts
BEFORE
UPDATE
    ON backup FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- backup_setting stores the backup settings for a particular database.
-- This is a strict version of cron expression using UTC timezone uniformly.
CREATE TABLE backup_setting (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    -- enable automatic backup schedule.
    enabled BOOLEAN NOT NULL,
    hour INTEGER NOT NULL CHECK (hour >= 0 AND hour <= 23),
    -- day_of_week can be -1 which is wildcard (daily automatic backup).
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= -1 AND day_of_week <= 6),
    -- retention_period_ts == 0 means unset retention period and we do not delete any data.
    retention_period_ts INTEGER NOT NULL DEFAULT 0 CHECK (retention_period_ts >= 0),
    -- hook_url is the callback url to be requested after a successful backup.
    hook_url TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_backup_setting_unique_database_id ON backup_setting(database_id);

ALTER SEQUENCE backup_setting_id_seq RESTART WITH 101;

CREATE TRIGGER update_backup_setting_updated_ts
BEFORE
UPDATE
    ON backup_setting FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-----------------------
-- Pipeline related BEGIN
-- pipeline table
CREATE TABLE pipeline (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL
);

ALTER SEQUENCE pipeline_id_seq RESTART WITH 101;

CREATE TRIGGER update_pipeline_updated_ts
BEFORE
UPDATE
    ON pipeline FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- stage table stores the stage for the pipeline
CREATE TABLE stage (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    name TEXT NOT NULL
);

CREATE INDEX idx_stage_pipeline_id ON stage(pipeline_id);

ALTER SEQUENCE stage_id_seq RESTART WITH 101;

CREATE TRIGGER update_stage_updated_ts
BEFORE
UPDATE
    ON stage FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- task table stores the task for the stage
CREATE TABLE task (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    stage_id INTEGER NOT NULL REFERENCES stage (id),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- Could be empty for creating database task when the task isn''t yet completed successfully.
    database_id INTEGER REFERENCES db (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN (''PENDING'', ''PENDING_APPROVAL'', ''RUNNING'', ''DONE'', ''FAILED'', ''CANCELED'')),
    type TEXT NOT NULL CHECK (type LIKE ''bb.task.%''),
    payload JSONB NOT NULL DEFAULT ''{}'',
    earliest_allowed_ts BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_task_pipeline_id_stage_id ON task(pipeline_id, stage_id);

CREATE INDEX idx_task_status ON task(status);

CREATE INDEX idx_task_earliest_allowed_ts ON task(earliest_allowed_ts);

ALTER SEQUENCE task_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_updated_ts
BEFORE
UPDATE
    ON task FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- task_dag describes task dependency relationship
-- from_task_id blocks to_task_id
CREATE TABLE task_dag (
    id SERIAL PRIMARY KEY,
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    from_task_id INTEGER NOT NULL REFERENCES task (id),
    to_task_id INTEGER NOT NULL REFERENCES task (id),
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_task_dag_from_task_id ON task_dag(from_task_id);

CREATE INDEX idx_task_dag_to_task_id ON task_dag(to_task_id);

ALTER SEQUENCE task_dag_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_dag_updated_ts
BEFORE
UPDATE
    ON task_dag FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- task run table stores the task run
CREATE TABLE task_run (
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    task_id INTEGER NOT NULL REFERENCES task (id),
    attempt INTEGER NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN (''PENDING'', ''RUNNING'', ''DONE'', ''FAILED'', ''CANCELED'')),
    started_ts BIGINT NOT NULL DEFAULT 0,
    code INTEGER NOT NULL DEFAULT 0,
    -- result saves the task run result in json format
    result  JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_task_run_task_id ON task_run(task_id);

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON task_run (task_id, attempt);

ALTER SEQUENCE task_run_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_run_updated_ts
BEFORE
UPDATE
    ON task_run FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Pipeline related END
-----------------------
-- Plan related BEGIN
CREATE TABLE plan (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    pipeline_id INTEGER REFERENCES pipeline (id),
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_plan_project_id ON plan(project_id);

CREATE INDEX idx_plan_pipeline_id ON plan(pipeline_id);

ALTER SEQUENCE plan_id_seq RESTART WITH 101;

CREATE TRIGGER update_plan_updated_ts
BEFORE
UPDATE
    ON plan FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

CREATE TABLE plan_check_run (
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    plan_id BIGINT NOT NULL REFERENCES plan (id),
    status TEXT NOT NULL CHECK (status IN (''RUNNING'', ''DONE'', ''FAILED'', ''CANCELED'')),
    type TEXT NOT NULL CHECK (type LIKE ''bb.plan-check.%''),
    config JSONB NOT NULL DEFAULT ''{}'',
    result JSONB NOT NULL DEFAULT ''{}'',
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_plan_check_run_plan_id ON plan_check_run (plan_id);

ALTER SEQUENCE plan_check_run_id_seq RESTART WITH 101;

CREATE TRIGGER update_plan_check_run_updated_ts
BEFORE
UPDATE
    ON plan_check_run FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Plan related END
-----------------------
-- issue
CREATE TABLE issue (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    plan_id BIGINT REFERENCES plan (id),
    pipeline_id INTEGER REFERENCES pipeline (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN (''OPEN'', ''DONE'', ''CANCELED'')),
    type TEXT NOT NULL CHECK (type LIKE ''bb.issue.%''),
    description TEXT NOT NULL DEFAULT '''',
    -- While changing assignee_id, one should only change it to a non-robot DBA/owner.
    assignee_id INTEGER NOT NULL REFERENCES principal (id),
    assignee_need_attention BOOLEAN NOT NULL DEFAULT FALSE, 
    payload JSONB NOT NULL DEFAULT ''{}'',
    ts_vector TSVECTOR
);

CREATE INDEX idx_issue_project_id ON issue(project_id);

CREATE INDEX idx_issue_plan_id ON issue(plan_id);

CREATE INDEX idx_issue_pipeline_id ON issue(pipeline_id);

CREATE INDEX idx_issue_creator_id ON issue(creator_id);

CREATE INDEX idx_issue_assignee_id ON issue(assignee_id);

CREATE INDEX idx_issue_created_ts ON issue(created_ts);

CREATE INDEX idx_issue_ts_vector ON issue USING gin(ts_vector);

ALTER SEQUENCE issue_id_seq RESTART WITH 101;

CREATE TRIGGER update_issue_updated_ts
BEFORE
UPDATE
    ON issue FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- stores the issue subscribers. Unlike other tables, it doesn''t have row_status/creator_id/created_ts/updater_id/updated_ts.
-- We use a separate table mainly because we can''t leverage indexed query if the subscriber id is stored
-- as a comma separated id list in the issue table.
CREATE TABLE issue_subscriber (
    issue_id INTEGER NOT NULL REFERENCES issue (id),
    subscriber_id INTEGER NOT NULL REFERENCES principal (id),
    PRIMARY KEY (issue_id, subscriber_id)
);

CREATE INDEX idx_issue_subscriber_subscriber_id ON issue_subscriber(subscriber_id);

-- instance change history records the changes an instance and its databases.
CREATE TABLE instance_change_history (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    -- NULL means the migrations for Bytebase''s own metadata database.
    instance_id INTEGER REFERENCES instance (id),
    -- NULL means an instance-level change.
    database_id INTEGER REFERENCES db (id),
    -- issue_id is nullable because this field is backfilled and may not be present.
    issue_id INTEGER REFERENCES issue (id),
    -- Record the client version creating this migration history. For Bytebase, we use its binary release version. Different Bytebase release might
    -- record different history info and this field helps to handle such situation properly. Moreover, it helps debugging.
    release_version TEXT NOT NULL,
    -- Used to detect out of order migration together with ''namespace'' and ''version'' column.
    sequence BIGINT NOT NULL CONSTRAINT instance_change_history_sequence_check CHECK (sequence >= 0),
    -- We call it source because maybe we could load history from other migration tool.
    -- Currently allowed values are UI, VCS, LIBRARY.
    source TEXT NOT NULL CONSTRAINT instance_change_history_source_check CHECK (source IN (''UI'', ''VCS'', ''LIBRARY'')),
    -- Currently allowed values are BASELINE, MIGRATE, MIGRATE_SDL, BRANCH, DATA.
    type TEXT NOT NULL CONSTRAINT instance_change_history_type_check CHECK (type IN (''BASELINE'', ''MIGRATE'', ''MIGRATE_SDL'', ''BRANCH'', ''DATA'')),
    -- Currently allowed values are PENDING, DONE, FAILED.
    -- PostgreSQL can''t do cross database transaction, so we can''t record DDL and migration_history into a single transaction.
    -- Thus, we create a "PENDING" record before applying the DDL and update that record to "DONE" after applying the DDL.
    status TEXT NOT NULL CONSTRAINT instance_change_history_status_check CHECK (status IN (''PENDING'', ''DONE'', ''FAILED'')),
    -- Record the migration version.
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    -- Record the change statement in preview format.
    statement TEXT NOT NULL,
    -- Record the sheet for the change statement. Optional.
    sheet_id BIGINT NULL,
    -- Record the schema after migration
    schema TEXT NOT NULL,
    -- Record the schema before migration. Though we could also fetch it from the previous migration history, it would complicate fetching logic.
    -- Besides, by storing the schema_prev, we can perform consistency check to see if the migration history has any gaps.
    schema_prev TEXT NOT NULL,
    execution_duration_ns BIGINT NOT NULL,
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_sequence ON instance_change_history (instance_id, database_id, sequence);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_version ON instance_change_history (instance_id, database_id, version);

ALTER SEQUENCE instance_change_history_id_seq RESTART WITH 101;

CREATE TRIGGER update_instance_change_history_updated_ts
BEFORE
UPDATE
    ON instance_change_history FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- activity table stores the activity for the container such as issue
CREATE TABLE activity (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    container_id INTEGER NOT NULL CHECK (container_id > 0),
    type TEXT NOT NULL CHECK (type LIKE ''bb.%''),
    level TEXT NOT NULL CHECK (level IN (''INFO'', ''WARN'', ''ERROR'')),
    comment TEXT NOT NULL DEFAULT '''',
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_activity_container_id ON activity(container_id);

CREATE INDEX idx_activity_created_ts ON activity(created_ts);

ALTER SEQUENCE activity_id_seq RESTART WITH 101;

CREATE TRIGGER update_activity_updated_ts
BEFORE
UPDATE
    ON activity FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- inbox table stores the inbox entry for the corresponding activity.
-- Unlike other tables, it doesn''t have row_status/creator_id/created_ts/updater_id/updated_ts.
-- We design in this way because:
-- 1. The table may potentially contain a lot of rows (an issue activity will generate one inbox record per issue subscriber)
-- 2. Does not provide much value besides what''s contained in the related activity record.
CREATE TABLE inbox (
    id SERIAL PRIMARY KEY,
    receiver_id INTEGER NOT NULL REFERENCES principal (id),
    activity_id INTEGER NOT NULL REFERENCES activity (id),
    status TEXT NOT NULL CHECK (status IN (''UNREAD'', ''READ''))
);

CREATE INDEX idx_inbox_receiver_id_activity_id ON inbox(receiver_id, activity_id);

CREATE INDEX idx_inbox_receiver_id_status ON inbox(receiver_id, status);

ALTER SEQUENCE inbox_id_seq RESTART WITH 101;

-- bookmark table stores the bookmark for the user
CREATE TABLE bookmark (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    link TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_bookmark_unique_creator_id_link ON bookmark(creator_id, link);

ALTER SEQUENCE bookmark_id_seq RESTART WITH 101;

CREATE TRIGGER update_bookmark_updated_ts
BEFORE
UPDATE
    ON bookmark FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- vcs table stores the version control provider config
CREATE TABLE vcs (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (''GITLAB'', ''GITHUB'', ''BITBUCKET'', ''AZURE_DEVOPS'')),
    instance_url TEXT NOT NULL CHECK ((instance_url LIKE ''http://%'' OR instance_url LIKE ''https://%'') AND instance_url = rtrim(instance_url, ''/'')),
    api_url TEXT NOT NULL CHECK ((api_url LIKE ''http://%'' OR api_url LIKE ''https://%'') AND api_url = rtrim(api_url, ''/'')),
    application_id TEXT NOT NULL,
    secret TEXT NOT NULL
);

ALTER SEQUENCE vcs_id_seq RESTART WITH 101;

CREATE TRIGGER update_vcs_updated_ts
BEFORE
UPDATE
    ON vcs FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- repository table stores the repository setting for a project
-- A vcs is associated with many repositories.
-- A project can only link one repository (at least for now).
CREATE TABLE repository (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    vcs_id INTEGER NOT NULL REFERENCES vcs (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    -- Name from the corresponding VCS provider.
    -- For GitLab, this is the project name. e.g. project 1
    name TEXT NOT NULL,
    -- Full path from the corresponding VCS provider.
    -- For GitLab, this is the project full path. e.g. group1/project-1
    full_path TEXT NOT NULL,
    -- Web url from the corresponding VCS provider.
    -- For GitLab, this is the project web url. e.g. https://gitlab.example.com/group1/project-1
    web_url TEXT NOT NULL,
    -- Branch we are interested.
    -- For GitLab, this corresponds to webhook''s push_events_branch_filter. Wildcard is supported
    branch_filter TEXT NOT NULL DEFAULT '''',
    -- Base working directory we are interested.
    base_directory TEXT NOT NULL DEFAULT '''',
    -- The file path template for matching the committed migration script.
    file_path_template TEXT NOT NULL DEFAULT '''',
    -- If enable the SQL review CI in VCS repository.
    enable_sql_review_ci BOOLEAN NOT NULL DEFAULT false,
    -- The file path template for storing the latest schema auto-generated by Bytebase after migration.
    -- If empty, then Bytebase won''t auto generate it.
    schema_path_template TEXT NOT NULL DEFAULT '''',
    -- The file path template to match the script file for sheet.
    sheet_path_template TEXT NOT NULL DEFAULT '''',
    -- Repository id from the corresponding VCS provider.
    -- For GitLab, this is the project id. e.g. 123
    external_id TEXT NOT NULL,
    -- Push webhook id from the corresponding VCS provider.
    -- For GitLab, this is the project webhook id. e.g. 123
    external_webhook_id TEXT NOT NULL,
    -- Identify the host of the webhook url where the webhook event sends. We store this to identify stale webhook url whose url doesn''t match the current bytebase --external-url.
    webhook_url_host TEXT NOT NULL,
    -- Identify the target repository receiving the webhook event. This is a random string.
    webhook_endpoint_id TEXT NOT NULL,
    -- For GitLab, webhook request contains this in the ''X-Gitlab-Token" header and we compare it with the one stored in db to validate it sends to the expected endpoint.
    webhook_secret_token TEXT NOT NULL,
    -- access_token, expires_ts, refresh_token belongs to the user linking the project to the VCS repository.
    access_token TEXT NOT NULL,
    expires_ts BIGINT NOT NULL,
    refresh_token TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_repository_unique_project_id ON repository(project_id);

ALTER SEQUENCE repository_id_seq RESTART WITH 101;

CREATE TRIGGER update_repository_updated_ts
BEFORE
UPDATE
    ON repository FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Anomaly
-- anomaly stores various anomalies found by the scanner.
-- For now, anomaly can be associated with a particular instance or database.
CREATE TABLE anomaly (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- NULL if it''s an instance anomaly
    database_id INTEGER NULL REFERENCES db (id),
    type TEXT NOT NULL CHECK (type LIKE ''bb.anomaly.%''),
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_anomaly_instance_id_row_status_type ON anomaly(instance_id, row_status, type);
CREATE INDEX idx_anomaly_database_id_row_status_type ON anomaly(database_id, row_status, type);

ALTER SEQUENCE anomaly_id_seq RESTART WITH 101;

CREATE TRIGGER update_anomaly_updated_ts
BEFORE
UPDATE
    ON anomaly FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Deployment Configuration.
-- deployment_config stores deployment configurations at project level.
CREATE TABLE deployment_config (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_deployment_config_unique_project_id ON deployment_config(project_id);

ALTER SEQUENCE deployment_config_id_seq RESTART WITH 101;

CREATE TRIGGER update_deployment_config_updated_ts
BEFORE
UPDATE
    ON deployment_config FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- sheet table stores general statements.
CREATE TABLE sheet (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    database_id INTEGER NULL REFERENCES db (id),
    name TEXT NOT NULL,
    statement TEXT NOT NULL,
    visibility TEXT NOT NULL CHECK (visibility IN (''PRIVATE'', ''PROJECT'', ''PUBLIC'')) DEFAULT ''PRIVATE'',
    source TEXT NOT NULL CONSTRAINT sheet_source_check CHECK (source IN (''BYTEBASE'', ''GITLAB'', ''GITHUB'', ''BITBUCKET'', ''AZURE_DEVOPS'', ''BYTEBASE_ARTIFACT'')) DEFAULT ''BYTEBASE'',
    type TEXT NOT NULL CHECK (type IN (''SQL'')) DEFAULT ''SQL'',
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_sheet_creator_id ON sheet(creator_id);

CREATE INDEX idx_sheet_project_id ON sheet(project_id);

CREATE INDEX idx_sheet_name ON sheet(name);

CREATE INDEX idx_sheet_project_id_row_status ON sheet(project_id, row_status);

CREATE INDEX idx_sheet_database_id_row_status ON sheet(database_id, row_status);

ALTER SEQUENCE sheet_id_seq RESTART WITH 101;

CREATE TRIGGER update_sheet_updated_ts
BEFORE
UPDATE
    ON sheet FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- sheet_organizer table stores the sheet status for a principal.
CREATE TABLE sheet_organizer (
    id SERIAL PRIMARY KEY,
    sheet_id INTEGER NOT NULL REFERENCES sheet (id) ON DELETE CASCADE,
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    starred BOOLEAN NOT NULL DEFAULT false,
    pinned BOOLEAN NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX idx_sheet_organizer_unique_sheet_id_principal_id ON sheet_organizer(sheet_id, principal_id);

CREATE INDEX idx_sheet_organizer_principal_id ON sheet_organizer(principal_id);

-- external_approval stores approval instances of third party applications.
CREATE TABLE external_approval ( 
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    issue_id INTEGER NOT NULL REFERENCES issue (id),
    requester_id INTEGER NOT NULL REFERENCES principal (id),
    approver_id INTEGER NOT NULL REFERENCES principal (id),
    type TEXT NOT NULL CHECK (type LIKE ''bb.plugin.app.%''),
    payload JSONB NOT NULL
);

CREATE INDEX idx_external_approval_row_status_issue_id ON external_approval(row_status, issue_id);

ALTER SEQUENCE external_approval_id_seq RESTART WITH 101;

CREATE TRIGGER update_external_approval_updated_ts
BEFORE
UPDATE
    ON external_approval FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();


-- risk stores the definition of a risk.
CREATE TABLE risk (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    source TEXT NOT NULL CHECK (source LIKE ''bb.risk.%''),
    -- how risky is the risk, the higher the riskier
    level BIGINT NOT NULL,
    name TEXT NOT NULL,
    active BOOLEAN NOT NULL,
    expression JSONB NOT NULL
);

ALTER SEQUENCE risk_id_seq RESTART WITH 101;

CREATE TRIGGER update_risk_updated_ts
BEFORE
UPDATE
    ON risk FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- slow_query stores slow query statistics for each database.
CREATE TABLE slow_query (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    -- updated_ts is used to identify the latest timestamp for syncing slow query logs.
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    -- In MySQL, users can query without specifying a database. In this case, instance_id is used to identify the instance.
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- In MySQL, users can query without specifying a database. In this case, database_id is NULL.
    database_id INTEGER NULL REFERENCES db (id),
    -- It''s hard to store all slow query logs, so the slow query is aggregated by day and database.
    log_date_ts INTEGER NOT NULL,
    -- It''s hard to store all slow query logs, we sample the slow query log and store the part of them as details.
    slow_query_statistics JSONB NOT NULL DEFAULT ''{}''
);

-- The slow query log is aggregated by day and database and we usually query the slow query log by day and database.
CREATE UNIQUE INDEX uk_slow_query_database_id_log_date_ts ON slow_query (database_id, log_date_ts);

CREATE INDEX idx_slow_query_instance_id_log_date_ts ON slow_query (instance_id, log_date_ts);

ALTER SEQUENCE slow_query_id_seq RESTART WITH 101;

CREATE TRIGGER update_slow_query_updated_ts
BEFORE
UPDATE
    ON slow_query FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

CREATE TABLE db_group (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    resource_id TEXT NOT NULL,
    placeholder TEXT NOT NULL DEFAULT '''',
    expression JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON db_group(project_id, resource_id);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON db_group(project_id, placeholder);

ALTER SEQUENCE db_group_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_group_updated_ts
BEFORE
UPDATE
    ON db_group FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

CREATE TABLE schema_group (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    db_group_id BIGINT NOT NULL REFERENCES db_group (id),
    resource_id TEXT NOT NULL,
    placeholder TEXT NOT NULL DEFAULT '''',
    expression JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_schema_group_unique_db_group_id_resource_id ON schema_group(db_group_id, resource_id);

CREATE UNIQUE INDEX idx_schema_group_unique_db_group_id_placeholder ON schema_group(db_group_id, placeholder);

ALTER SEQUENCE schema_group_id_seq RESTART WITH 101;

CREATE TRIGGER update_schema_group_updated_ts
BEFORE
UPDATE
    ON schema_group FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- changelist table stores project changelists.
CREATE TABLE changelist (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_changelist_project_id_name ON changelist(project_id, name);

ALTER SEQUENCE changelist_id_seq RESTART WITH 101;

CREATE TRIGGER update_changelist_updated_ts
BEFORE
UPDATE
    ON changelist FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
-- Create "test" and "prod" environments
INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order",
        resource_id
    )
VALUES
    (
        101,
        1,
        1,
        ''Test'',
        0,
        ''test''
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order",
        resource_id
    )
VALUES
    (
        102,
        1,
        1,
        ''Prod'',
        1,
        ''prod''
    );

ALTER SEQUENCE environment_id_seq RESTART WITH 103;
', NULL, '-- Type
CREATE TYPE row_status AS ENUM (''NORMAL'', ''ARCHIVED'');

-- updated_ts trigger.
CREATE OR REPLACE FUNCTION trigger_update_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = extract(epoch from now());
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- idp stores generic identity provider.
CREATE TABLE idp (
  id SERIAL PRIMARY KEY,
  row_status row_status NOT NULL DEFAULT ''NORMAL'',
  created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  resource_id TEXT NOT NULL,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  type TEXT NOT NULL CONSTRAINT idp_type_check CHECK (type IN (''OAUTH2'', ''OIDC'', ''LDAP'')),
  -- config stores the corresponding configuration of the IdP, which may vary depending on the type of the IdP.
  config JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON idp(resource_id);

ALTER SEQUENCE idp_id_seq RESTART WITH 101;

CREATE TRIGGER update_idp_updated_ts
BEFORE
UPDATE
    ON idp FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- principal
CREATE TABLE principal (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    type TEXT NOT NULL CHECK (type IN (''END_USER'', ''SYSTEM_BOT'', ''SERVICE_ACCOUNT'')),
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    phone TEXT NOT NULL DEFAULT '''',
    mfa_config JSONB NOT NULL DEFAULT ''{}''
);

CREATE TRIGGER update_principal_updated_ts
BEFORE
UPDATE
    ON principal FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Default bytebase system account id is 1
INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        type,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        1,
        ''SYSTEM_BOT'',
        ''Bytebase'',
        ''support@bytebase.com'',
        ''''
    );

ALTER SEQUENCE principal_id_seq RESTART WITH 101;

-- Setting
CREATE TABLE setting (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''''
);

CREATE UNIQUE INDEX idx_setting_unique_name ON setting(name);

ALTER SEQUENCE setting_id_seq RESTART WITH 101;

CREATE TRIGGER update_setting_updated_ts
BEFORE
UPDATE
    ON setting FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Role
CREATE TABLE role (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    resource_id TEXT NOT NULL, -- user-defined id, such as projectDBA
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    permissions JSONB NOT NULL DEFAULT ''{}'', -- saved for future use
    payload JSONB NOT NULL DEFAULT ''{}'' -- saved for future use
);

CREATE UNIQUE INDEX idx_role_unique_resource_id on role (resource_id);

ALTER SEQUENCE role_id_seq RESTART WITH 101;

CREATE TRIGGER update_role_updated_ts
BEFORE
UPDATE
    ON role FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Member
-- We separate the concept from Principal because if we support multiple workspace in the future, each workspace can have different member for the same principal
CREATE TABLE member (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    status TEXT NOT NULL CHECK (status IN (''INVITED'', ''ACTIVE'')),
    role TEXT NOT NULL CHECK (role IN (''OWNER'', ''DBA'', ''DEVELOPER'')),
    principal_id INTEGER NOT NULL REFERENCES principal (id)
);

CREATE UNIQUE INDEX idx_member_unique_principal_id ON member(principal_id);

ALTER SEQUENCE member_id_seq RESTART WITH 101;

CREATE TRIGGER update_member_updated_ts
BEFORE
UPDATE
    ON member FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Environment
CREATE TABLE environment (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    "order" INTEGER NOT NULL CHECK ("order" >= 0),
    resource_id TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_environment_unique_name ON environment(name);

CREATE UNIQUE INDEX idx_environment_unique_resource_id ON environment(resource_id);

ALTER SEQUENCE environment_id_seq RESTART WITH 101;

CREATE TRIGGER update_environment_updated_ts
BEFORE
UPDATE
    ON environment FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Policy
-- policy stores the policies for each environment.
-- Policies are associated with environments. Since we may have policies not associated with environment later, we name the table policy.
CREATE TYPE resource_type AS ENUM (''WORKSPACE'', ''ENVIRONMENT'', ''PROJECT'', ''INSTANCE'', ''DATABASE'');

CREATE TABLE policy (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    type TEXT NOT NULL CHECK (type LIKE ''bb.policy.%''),
    payload JSONB NOT NULL DEFAULT ''{}'',
    resource_type resource_type NOT NULL,
    resource_id INTEGER NOT NULL,
    inherit_from_parent BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_id_type ON policy(resource_type, resource_id, type);

ALTER SEQUENCE policy_id_seq RESTART WITH 101;

CREATE TRIGGER update_policy_updated_ts
BEFORE
UPDATE
    ON policy FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Project
CREATE TABLE project (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    key TEXT NOT NULL,
    workflow_type TEXT NOT NULL CHECK (workflow_type IN (''UI'', ''VCS'')),
    visibility TEXT NOT NULL CHECK (visibility IN (''PUBLIC'', ''PRIVATE'')),
    tenant_mode TEXT NOT NULL CHECK (tenant_mode IN (''DISABLED'', ''TENANT'')) DEFAULT ''DISABLED'',
    -- db_name_template is only used when a project is in tenant mode.
    -- Empty value means {{DB_NAME}}.
    db_name_template TEXT NOT NULL,
    schema_change_type TEXT NOT NULL CHECK (schema_change_type IN (''DDL'', ''SDL'')) DEFAULT ''DDL'',
    resource_id TEXT NOT NULL,
    data_classification_config_id TEXT NOT NULL DEFAULT '''',
    setting  JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_project_unique_key ON project(key);

CREATE UNIQUE INDEX idx_project_unique_resource_id ON project(resource_id);

INSERT INTO
    project (
        id,
        creator_id,
        updater_id,
        name,
        key,
        workflow_type,
        visibility,
        tenant_mode,
        db_name_template,
        resource_id
    )
VALUES
    (
        1,
        1,
        1,
        ''Default'',
        ''DEFAULT'',
        ''UI'',
        ''PUBLIC'',
        ''DISABLED'',
        '''',
        ''default''
    );

ALTER SEQUENCE project_id_seq RESTART WITH 101;

CREATE TRIGGER update_project_updated_ts
BEFORE
UPDATE
    ON project FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Project member
CREATE TABLE project_member (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    role TEXT NOT NULL,
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    condition JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_project_member_project_id ON project_member(project_id);

ALTER SEQUENCE project_member_id_seq RESTART WITH 101;

CREATE TRIGGER update_project_member_updated_ts
BEFORE
UPDATE
    ON project_member FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Project Hook
CREATE TABLE project_webhook (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    type TEXT NOT NULL CHECK (type LIKE ''bb.plugin.webhook.%''),
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    activity_list TEXT ARRAY NOT NULL
);

CREATE INDEX idx_project_webhook_project_id ON project_webhook(project_id);

CREATE UNIQUE INDEX idx_project_webhook_unique_project_id_url ON project_webhook(project_id, url);

ALTER SEQUENCE project_webhook_id_seq RESTART WITH 101;

CREATE TRIGGER update_project_webhook_updated_ts
BEFORE
UPDATE
    ON project_webhook FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Instance
CREATE TABLE instance (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    environment_id INTEGER REFERENCES environment (id),
    name TEXT NOT NULL,
    engine TEXT NOT NULL,
    engine_version TEXT NOT NULL DEFAULT '''',
    external_link TEXT NOT NULL DEFAULT '''',
    resource_id TEXT NOT NULL,
    -- activation should set to be TRUE if users assign license to this instance.
    activation BOOLEAN NOT NULL DEFAULT false,
    options JSONB NOT NULL DEFAULT ''{}'',
    metadata JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON instance(resource_id);

ALTER SEQUENCE instance_id_seq RESTART WITH 101;

CREATE TRIGGER update_instance_updated_ts
BEFORE
UPDATE
    ON instance FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Instance user stores the users for a particular instance
CREATE TABLE instance_user (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    name TEXT NOT NULL,
    "grant" TEXT NOT NULL
);

ALTER SEQUENCE instance_user_id_seq RESTART WITH 101;

CREATE UNIQUE INDEX idx_instance_user_unique_instance_id_name ON instance_user(instance_id, name);

CREATE TRIGGER update_instance_user_updated_ts
BEFORE
UPDATE
    ON instance_user FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- db stores the databases for a particular instance
-- data is synced periodically from the instance
CREATE TABLE db (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    environment_id INTEGER REFERENCES environment (id),
    -- If db is restored from a backup, then we will record that backup id. We can thus trace up to the original db.
    source_backup_id INTEGER,
    sync_status TEXT NOT NULL CHECK (sync_status IN (''OK'', ''NOT_FOUND'')),
    last_successful_sync_ts BIGINT NOT NULL,
    schema_version TEXT NOT NULL,
    name TEXT NOT NULL,
    secrets JSONB NOT NULL DEFAULT ''{}'',
    datashare BOOLEAN NOT NULL DEFAULT FALSE,
    -- service_name is the Oracle specific field.
    service_name TEXT NOT NULL DEFAULT '''',
    metadata JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_db_instance_id ON db(instance_id);

CREATE UNIQUE INDEX idx_db_unique_instance_id_name ON db(instance_id, name);

CREATE INDEX idx_db_project_id ON db(project_id);

ALTER SEQUENCE db_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_updated_ts
BEFORE
UPDATE
    ON db FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- db_schema stores the database schema metadata for a particular database.
CREATE TABLE db_schema (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id) ON DELETE CASCADE,
    metadata JSONB NOT NULL DEFAULT ''{}'',
    raw_dump TEXT NOT NULL DEFAULT '''',
    config JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_db_schema_unique_database_id ON db_schema(database_id);

ALTER SEQUENCE db_schema_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_schema_updated_ts
BEFORE
UPDATE
    ON db_schema FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- data_source table stores the data source for a particular database
CREATE TABLE data_source (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (''ADMIN'', ''RW'', ''RO'')),
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    ssl_key TEXT NOT NULL DEFAULT '''',
    ssl_cert TEXT NOT NULL DEFAULT '''',
    ssl_ca TEXT NOT NULL DEFAULT '''',
    host TEXT NOT NULL DEFAULT '''',
    port TEXT NOT NULL DEFAULT '''',
    options JSONB NOT NULL DEFAULT ''{}'',
    database TEXT NOT NULL DEFAULT ''''
);

CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON data_source(instance_id, name);

ALTER SEQUENCE data_source_id_seq RESTART WITH 101;

CREATE TRIGGER update_data_source_updated_ts
BEFORE
UPDATE
    ON data_source FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- backup stores the backups for a particular database.
CREATE TABLE backup (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN (''PENDING_CREATE'', ''DONE'', ''FAILED'')),
    type TEXT NOT NULL CHECK (type IN (''MANUAL'', ''AUTOMATIC'', ''PITR'')),
    storage_backend TEXT NOT NULL CHECK (storage_backend IN (''LOCAL'', ''S3'', ''GCS'', ''OSS'')),
    migration_history_version TEXT NOT NULL,
    path TEXT NOT NULL,
    comment TEXT NOT NULL DEFAULT '''',
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_backup_database_id ON backup(database_id);

CREATE UNIQUE INDEX idx_backup_unique_database_id_name ON backup(database_id, name);

ALTER SEQUENCE backup_id_seq RESTART WITH 101;

CREATE TRIGGER update_backup_updated_ts
BEFORE
UPDATE
    ON backup FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- backup_setting stores the backup settings for a particular database.
-- This is a strict version of cron expression using UTC timezone uniformly.
CREATE TABLE backup_setting (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    -- enable automatic backup schedule.
    enabled BOOLEAN NOT NULL,
    hour INTEGER NOT NULL CHECK (hour >= 0 AND hour <= 23),
    -- day_of_week can be -1 which is wildcard (daily automatic backup).
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= -1 AND day_of_week <= 6),
    -- retention_period_ts == 0 means unset retention period and we do not delete any data.
    retention_period_ts INTEGER NOT NULL DEFAULT 0 CHECK (retention_period_ts >= 0),
    -- hook_url is the callback url to be requested after a successful backup.
    hook_url TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_backup_setting_unique_database_id ON backup_setting(database_id);

ALTER SEQUENCE backup_setting_id_seq RESTART WITH 101;

CREATE TRIGGER update_backup_setting_updated_ts
BEFORE
UPDATE
    ON backup_setting FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-----------------------
-- Pipeline related BEGIN
-- pipeline table
CREATE TABLE pipeline (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL
);

ALTER SEQUENCE pipeline_id_seq RESTART WITH 101;

CREATE TRIGGER update_pipeline_updated_ts
BEFORE
UPDATE
    ON pipeline FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- stage table stores the stage for the pipeline
CREATE TABLE stage (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    name TEXT NOT NULL
);

CREATE INDEX idx_stage_pipeline_id ON stage(pipeline_id);

ALTER SEQUENCE stage_id_seq RESTART WITH 101;

CREATE TRIGGER update_stage_updated_ts
BEFORE
UPDATE
    ON stage FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- task table stores the task for the stage
CREATE TABLE task (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    stage_id INTEGER NOT NULL REFERENCES stage (id),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- Could be empty for creating database task when the task isn''t yet completed successfully.
    database_id INTEGER REFERENCES db (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN (''PENDING'', ''PENDING_APPROVAL'', ''RUNNING'', ''DONE'', ''FAILED'', ''CANCELED'')),
    type TEXT NOT NULL CHECK (type LIKE ''bb.task.%''),
    payload JSONB NOT NULL DEFAULT ''{}'',
    earliest_allowed_ts BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_task_pipeline_id_stage_id ON task(pipeline_id, stage_id);

CREATE INDEX idx_task_status ON task(status);

CREATE INDEX idx_task_earliest_allowed_ts ON task(earliest_allowed_ts);

ALTER SEQUENCE task_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_updated_ts
BEFORE
UPDATE
    ON task FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- task_dag describes task dependency relationship
-- from_task_id blocks to_task_id
CREATE TABLE task_dag (
    id SERIAL PRIMARY KEY,
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    from_task_id INTEGER NOT NULL REFERENCES task (id),
    to_task_id INTEGER NOT NULL REFERENCES task (id),
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_task_dag_from_task_id ON task_dag(from_task_id);

CREATE INDEX idx_task_dag_to_task_id ON task_dag(to_task_id);

ALTER SEQUENCE task_dag_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_dag_updated_ts
BEFORE
UPDATE
    ON task_dag FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- task run table stores the task run
CREATE TABLE task_run (
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    task_id INTEGER NOT NULL REFERENCES task (id),
    attempt INTEGER NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN (''PENDING'', ''RUNNING'', ''DONE'', ''FAILED'', ''CANCELED'')),
    started_ts BIGINT NOT NULL DEFAULT 0,
    code INTEGER NOT NULL DEFAULT 0,
    -- result saves the task run result in json format
    result  JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_task_run_task_id ON task_run(task_id);

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON task_run (task_id, attempt);

ALTER SEQUENCE task_run_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_run_updated_ts
BEFORE
UPDATE
    ON task_run FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Pipeline related END
-----------------------
-- Plan related BEGIN
CREATE TABLE plan (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    pipeline_id INTEGER REFERENCES pipeline (id),
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_plan_project_id ON plan(project_id);

CREATE INDEX idx_plan_pipeline_id ON plan(pipeline_id);

ALTER SEQUENCE plan_id_seq RESTART WITH 101;

CREATE TRIGGER update_plan_updated_ts
BEFORE
UPDATE
    ON plan FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

CREATE TABLE plan_check_run (
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    plan_id BIGINT NOT NULL REFERENCES plan (id),
    status TEXT NOT NULL CHECK (status IN (''RUNNING'', ''DONE'', ''FAILED'', ''CANCELED'')),
    type TEXT NOT NULL CHECK (type LIKE ''bb.plan-check.%''),
    config JSONB NOT NULL DEFAULT ''{}'',
    result JSONB NOT NULL DEFAULT ''{}'',
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_plan_check_run_plan_id ON plan_check_run (plan_id);

ALTER SEQUENCE plan_check_run_id_seq RESTART WITH 101;

CREATE TRIGGER update_plan_check_run_updated_ts
BEFORE
UPDATE
    ON plan_check_run FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Plan related END
-----------------------
-- issue
CREATE TABLE issue (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    plan_id BIGINT REFERENCES plan (id),
    pipeline_id INTEGER REFERENCES pipeline (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN (''OPEN'', ''DONE'', ''CANCELED'')),
    type TEXT NOT NULL CHECK (type LIKE ''bb.issue.%''),
    description TEXT NOT NULL DEFAULT '''',
    -- While changing assignee_id, one should only change it to a non-robot DBA/owner.
    assignee_id INTEGER NOT NULL REFERENCES principal (id),
    assignee_need_attention BOOLEAN NOT NULL DEFAULT FALSE, 
    payload JSONB NOT NULL DEFAULT ''{}'',
    ts_vector TSVECTOR
);

CREATE INDEX idx_issue_project_id ON issue(project_id);

CREATE INDEX idx_issue_plan_id ON issue(plan_id);

CREATE INDEX idx_issue_pipeline_id ON issue(pipeline_id);

CREATE INDEX idx_issue_creator_id ON issue(creator_id);

CREATE INDEX idx_issue_assignee_id ON issue(assignee_id);

CREATE INDEX idx_issue_created_ts ON issue(created_ts);

CREATE INDEX idx_issue_ts_vector ON issue USING gin(ts_vector);

ALTER SEQUENCE issue_id_seq RESTART WITH 101;

CREATE TRIGGER update_issue_updated_ts
BEFORE
UPDATE
    ON issue FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- stores the issue subscribers. Unlike other tables, it doesn''t have row_status/creator_id/created_ts/updater_id/updated_ts.
-- We use a separate table mainly because we can''t leverage indexed query if the subscriber id is stored
-- as a comma separated id list in the issue table.
CREATE TABLE issue_subscriber (
    issue_id INTEGER NOT NULL REFERENCES issue (id),
    subscriber_id INTEGER NOT NULL REFERENCES principal (id),
    PRIMARY KEY (issue_id, subscriber_id)
);

CREATE INDEX idx_issue_subscriber_subscriber_id ON issue_subscriber(subscriber_id);

-- instance change history records the changes an instance and its databases.
CREATE TABLE instance_change_history (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    -- NULL means the migrations for Bytebase''s own metadata database.
    instance_id INTEGER REFERENCES instance (id),
    -- NULL means an instance-level change.
    database_id INTEGER REFERENCES db (id),
    -- issue_id is nullable because this field is backfilled and may not be present.
    issue_id INTEGER REFERENCES issue (id),
    -- Record the client version creating this migration history. For Bytebase, we use its binary release version. Different Bytebase release might
    -- record different history info and this field helps to handle such situation properly. Moreover, it helps debugging.
    release_version TEXT NOT NULL,
    -- Used to detect out of order migration together with ''namespace'' and ''version'' column.
    sequence BIGINT NOT NULL CONSTRAINT instance_change_history_sequence_check CHECK (sequence >= 0),
    -- We call it source because maybe we could load history from other migration tool.
    -- Currently allowed values are UI, VCS, LIBRARY.
    source TEXT NOT NULL CONSTRAINT instance_change_history_source_check CHECK (source IN (''UI'', ''VCS'', ''LIBRARY'')),
    -- Currently allowed values are BASELINE, MIGRATE, MIGRATE_SDL, BRANCH, DATA.
    type TEXT NOT NULL CONSTRAINT instance_change_history_type_check CHECK (type IN (''BASELINE'', ''MIGRATE'', ''MIGRATE_SDL'', ''BRANCH'', ''DATA'')),
    -- Currently allowed values are PENDING, DONE, FAILED.
    -- PostgreSQL can''t do cross database transaction, so we can''t record DDL and migration_history into a single transaction.
    -- Thus, we create a "PENDING" record before applying the DDL and update that record to "DONE" after applying the DDL.
    status TEXT NOT NULL CONSTRAINT instance_change_history_status_check CHECK (status IN (''PENDING'', ''DONE'', ''FAILED'')),
    -- Record the migration version.
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    -- Record the change statement in preview format.
    statement TEXT NOT NULL,
    -- Record the sheet for the change statement. Optional.
    sheet_id BIGINT NULL,
    -- Record the schema after migration
    schema TEXT NOT NULL,
    -- Record the schema before migration. Though we could also fetch it from the previous migration history, it would complicate fetching logic.
    -- Besides, by storing the schema_prev, we can perform consistency check to see if the migration history has any gaps.
    schema_prev TEXT NOT NULL,
    execution_duration_ns BIGINT NOT NULL,
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_sequence ON instance_change_history (instance_id, database_id, sequence);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_version ON instance_change_history (instance_id, database_id, version);

ALTER SEQUENCE instance_change_history_id_seq RESTART WITH 101;

CREATE TRIGGER update_instance_change_history_updated_ts
BEFORE
UPDATE
    ON instance_change_history FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- activity table stores the activity for the container such as issue
CREATE TABLE activity (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    container_id INTEGER NOT NULL CHECK (container_id > 0),
    type TEXT NOT NULL CHECK (type LIKE ''bb.%''),
    level TEXT NOT NULL CHECK (level IN (''INFO'', ''WARN'', ''ERROR'')),
    comment TEXT NOT NULL DEFAULT '''',
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_activity_container_id ON activity(container_id);

CREATE INDEX idx_activity_created_ts ON activity(created_ts);

ALTER SEQUENCE activity_id_seq RESTART WITH 101;

CREATE TRIGGER update_activity_updated_ts
BEFORE
UPDATE
    ON activity FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- inbox table stores the inbox entry for the corresponding activity.
-- Unlike other tables, it doesn''t have row_status/creator_id/created_ts/updater_id/updated_ts.
-- We design in this way because:
-- 1. The table may potentially contain a lot of rows (an issue activity will generate one inbox record per issue subscriber)
-- 2. Does not provide much value besides what''s contained in the related activity record.
CREATE TABLE inbox (
    id SERIAL PRIMARY KEY,
    receiver_id INTEGER NOT NULL REFERENCES principal (id),
    activity_id INTEGER NOT NULL REFERENCES activity (id),
    status TEXT NOT NULL CHECK (status IN (''UNREAD'', ''READ''))
);

CREATE INDEX idx_inbox_receiver_id_activity_id ON inbox(receiver_id, activity_id);

CREATE INDEX idx_inbox_receiver_id_status ON inbox(receiver_id, status);

ALTER SEQUENCE inbox_id_seq RESTART WITH 101;

-- bookmark table stores the bookmark for the user
CREATE TABLE bookmark (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    link TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_bookmark_unique_creator_id_link ON bookmark(creator_id, link);

ALTER SEQUENCE bookmark_id_seq RESTART WITH 101;

CREATE TRIGGER update_bookmark_updated_ts
BEFORE
UPDATE
    ON bookmark FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- vcs table stores the version control provider config
CREATE TABLE vcs (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (''GITLAB'', ''GITHUB'', ''BITBUCKET'', ''AZURE_DEVOPS'')),
    instance_url TEXT NOT NULL CHECK ((instance_url LIKE ''http://%'' OR instance_url LIKE ''https://%'') AND instance_url = rtrim(instance_url, ''/'')),
    api_url TEXT NOT NULL CHECK ((api_url LIKE ''http://%'' OR api_url LIKE ''https://%'') AND api_url = rtrim(api_url, ''/'')),
    application_id TEXT NOT NULL,
    secret TEXT NOT NULL
);

ALTER SEQUENCE vcs_id_seq RESTART WITH 101;

CREATE TRIGGER update_vcs_updated_ts
BEFORE
UPDATE
    ON vcs FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- repository table stores the repository setting for a project
-- A vcs is associated with many repositories.
-- A project can only link one repository (at least for now).
CREATE TABLE repository (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    vcs_id INTEGER NOT NULL REFERENCES vcs (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    -- Name from the corresponding VCS provider.
    -- For GitLab, this is the project name. e.g. project 1
    name TEXT NOT NULL,
    -- Full path from the corresponding VCS provider.
    -- For GitLab, this is the project full path. e.g. group1/project-1
    full_path TEXT NOT NULL,
    -- Web url from the corresponding VCS provider.
    -- For GitLab, this is the project web url. e.g. https://gitlab.example.com/group1/project-1
    web_url TEXT NOT NULL,
    -- Branch we are interested.
    -- For GitLab, this corresponds to webhook''s push_events_branch_filter. Wildcard is supported
    branch_filter TEXT NOT NULL DEFAULT '''',
    -- Base working directory we are interested.
    base_directory TEXT NOT NULL DEFAULT '''',
    -- The file path template for matching the committed migration script.
    file_path_template TEXT NOT NULL DEFAULT '''',
    -- If enable the SQL review CI in VCS repository.
    enable_sql_review_ci BOOLEAN NOT NULL DEFAULT false,
    -- The file path template for storing the latest schema auto-generated by Bytebase after migration.
    -- If empty, then Bytebase won''t auto generate it.
    schema_path_template TEXT NOT NULL DEFAULT '''',
    -- The file path template to match the script file for sheet.
    sheet_path_template TEXT NOT NULL DEFAULT '''',
    -- Repository id from the corresponding VCS provider.
    -- For GitLab, this is the project id. e.g. 123
    external_id TEXT NOT NULL,
    -- Push webhook id from the corresponding VCS provider.
    -- For GitLab, this is the project webhook id. e.g. 123
    external_webhook_id TEXT NOT NULL,
    -- Identify the host of the webhook url where the webhook event sends. We store this to identify stale webhook url whose url doesn''t match the current bytebase --external-url.
    webhook_url_host TEXT NOT NULL,
    -- Identify the target repository receiving the webhook event. This is a random string.
    webhook_endpoint_id TEXT NOT NULL,
    -- For GitLab, webhook request contains this in the ''X-Gitlab-Token" header and we compare it with the one stored in db to validate it sends to the expected endpoint.
    webhook_secret_token TEXT NOT NULL,
    -- access_token, expires_ts, refresh_token belongs to the user linking the project to the VCS repository.
    access_token TEXT NOT NULL,
    expires_ts BIGINT NOT NULL,
    refresh_token TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_repository_unique_project_id ON repository(project_id);

ALTER SEQUENCE repository_id_seq RESTART WITH 101;

CREATE TRIGGER update_repository_updated_ts
BEFORE
UPDATE
    ON repository FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Anomaly
-- anomaly stores various anomalies found by the scanner.
-- For now, anomaly can be associated with a particular instance or database.
CREATE TABLE anomaly (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- NULL if it''s an instance anomaly
    database_id INTEGER NULL REFERENCES db (id),
    type TEXT NOT NULL CHECK (type LIKE ''bb.anomaly.%''),
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_anomaly_instance_id_row_status_type ON anomaly(instance_id, row_status, type);
CREATE INDEX idx_anomaly_database_id_row_status_type ON anomaly(database_id, row_status, type);

ALTER SEQUENCE anomaly_id_seq RESTART WITH 101;

CREATE TRIGGER update_anomaly_updated_ts
BEFORE
UPDATE
    ON anomaly FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Deployment Configuration.
-- deployment_config stores deployment configurations at project level.
CREATE TABLE deployment_config (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_deployment_config_unique_project_id ON deployment_config(project_id);

ALTER SEQUENCE deployment_config_id_seq RESTART WITH 101;

CREATE TRIGGER update_deployment_config_updated_ts
BEFORE
UPDATE
    ON deployment_config FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- sheet table stores general statements.
CREATE TABLE sheet (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    database_id INTEGER NULL REFERENCES db (id),
    name TEXT NOT NULL,
    statement TEXT NOT NULL,
    visibility TEXT NOT NULL CHECK (visibility IN (''PRIVATE'', ''PROJECT'', ''PUBLIC'')) DEFAULT ''PRIVATE'',
    source TEXT NOT NULL CONSTRAINT sheet_source_check CHECK (source IN (''BYTEBASE'', ''GITLAB'', ''GITHUB'', ''BITBUCKET'', ''AZURE_DEVOPS'', ''BYTEBASE_ARTIFACT'')) DEFAULT ''BYTEBASE'',
    type TEXT NOT NULL CHECK (type IN (''SQL'')) DEFAULT ''SQL'',
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE INDEX idx_sheet_creator_id ON sheet(creator_id);

CREATE INDEX idx_sheet_project_id ON sheet(project_id);

CREATE INDEX idx_sheet_name ON sheet(name);

CREATE INDEX idx_sheet_project_id_row_status ON sheet(project_id, row_status);

CREATE INDEX idx_sheet_database_id_row_status ON sheet(database_id, row_status);

ALTER SEQUENCE sheet_id_seq RESTART WITH 101;

CREATE TRIGGER update_sheet_updated_ts
BEFORE
UPDATE
    ON sheet FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- sheet_organizer table stores the sheet status for a principal.
CREATE TABLE sheet_organizer (
    id SERIAL PRIMARY KEY,
    sheet_id INTEGER NOT NULL REFERENCES sheet (id) ON DELETE CASCADE,
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    starred BOOLEAN NOT NULL DEFAULT false,
    pinned BOOLEAN NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX idx_sheet_organizer_unique_sheet_id_principal_id ON sheet_organizer(sheet_id, principal_id);

CREATE INDEX idx_sheet_organizer_principal_id ON sheet_organizer(principal_id);

-- external_approval stores approval instances of third party applications.
CREATE TABLE external_approval ( 
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    issue_id INTEGER NOT NULL REFERENCES issue (id),
    requester_id INTEGER NOT NULL REFERENCES principal (id),
    approver_id INTEGER NOT NULL REFERENCES principal (id),
    type TEXT NOT NULL CHECK (type LIKE ''bb.plugin.app.%''),
    payload JSONB NOT NULL
);

CREATE INDEX idx_external_approval_row_status_issue_id ON external_approval(row_status, issue_id);

ALTER SEQUENCE external_approval_id_seq RESTART WITH 101;

CREATE TRIGGER update_external_approval_updated_ts
BEFORE
UPDATE
    ON external_approval FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();


-- risk stores the definition of a risk.
CREATE TABLE risk (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    source TEXT NOT NULL CHECK (source LIKE ''bb.risk.%''),
    -- how risky is the risk, the higher the riskier
    level BIGINT NOT NULL,
    name TEXT NOT NULL,
    active BOOLEAN NOT NULL,
    expression JSONB NOT NULL
);

ALTER SEQUENCE risk_id_seq RESTART WITH 101;

CREATE TRIGGER update_risk_updated_ts
BEFORE
UPDATE
    ON risk FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- slow_query stores slow query statistics for each database.
CREATE TABLE slow_query (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    -- updated_ts is used to identify the latest timestamp for syncing slow query logs.
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    -- In MySQL, users can query without specifying a database. In this case, instance_id is used to identify the instance.
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- In MySQL, users can query without specifying a database. In this case, database_id is NULL.
    database_id INTEGER NULL REFERENCES db (id),
    -- It''s hard to store all slow query logs, so the slow query is aggregated by day and database.
    log_date_ts INTEGER NOT NULL,
    -- It''s hard to store all slow query logs, we sample the slow query log and store the part of them as details.
    slow_query_statistics JSONB NOT NULL DEFAULT ''{}''
);

-- The slow query log is aggregated by day and database and we usually query the slow query log by day and database.
CREATE UNIQUE INDEX uk_slow_query_database_id_log_date_ts ON slow_query (database_id, log_date_ts);

CREATE INDEX idx_slow_query_instance_id_log_date_ts ON slow_query (instance_id, log_date_ts);

ALTER SEQUENCE slow_query_id_seq RESTART WITH 101;

CREATE TRIGGER update_slow_query_updated_ts
BEFORE
UPDATE
    ON slow_query FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

CREATE TABLE db_group (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    resource_id TEXT NOT NULL,
    placeholder TEXT NOT NULL DEFAULT '''',
    expression JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON db_group(project_id, resource_id);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON db_group(project_id, placeholder);

ALTER SEQUENCE db_group_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_group_updated_ts
BEFORE
UPDATE
    ON db_group FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

CREATE TABLE schema_group (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    db_group_id BIGINT NOT NULL REFERENCES db_group (id),
    resource_id TEXT NOT NULL,
    placeholder TEXT NOT NULL DEFAULT '''',
    expression JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_schema_group_unique_db_group_id_resource_id ON schema_group(db_group_id, resource_id);

CREATE UNIQUE INDEX idx_schema_group_unique_db_group_id_placeholder ON schema_group(db_group_id, placeholder);

ALTER SEQUENCE schema_group_id_seq RESTART WITH 101;

CREATE TRIGGER update_schema_group_updated_ts
BEFORE
UPDATE
    ON schema_group FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- changelist table stores project changelists.
CREATE TABLE changelist (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT ''{}''
);

CREATE UNIQUE INDEX idx_changelist_project_id_name ON changelist(project_id, name);

ALTER SEQUENCE changelist_id_seq RESTART WITH 101;

CREATE TRIGGER update_changelist_updated_ts
BEFORE
UPDATE
    ON changelist FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
-- Create "test" and "prod" environments
INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order",
        resource_id
    )
VALUES
    (
        101,
        1,
        1,
        ''Test'',
        0,
        ''test''
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order",
        resource_id
    )
VALUES
    (
        102,
        1,
        1,
        ''Prod'',
        1,
        ''prod''
    );

ALTER SEQUENCE environment_id_seq RESTART WITH 103;
', '', 0, '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, issue_id, release_version, sequence, source, type, status, version, description, statement, sheet_id, schema, schema_prev, execution_duration_ns, payload) VALUES (102, 'NORMAL', 1, 1698730969, 1, 1698730969, NULL, NULL, NULL, 'development', 2, 'LIBRARY', 'MIGRATE', 'DONE', '0002.0010.0003-dev20220408000000', 'Migrate version 20220408000000 server version development with files migration/dev/20220408000000##schema_version_type.sql.', 'ALTER TABLE project ADD schema_version_type TEXT NOT NULL CHECK (schema_version_type IN (''TIMESTAMP'', ''SEMANTIC'')) DEFAULT ''TIMESTAMP'';
', NULL, '
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = ''UTF8'';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config(''search_path'', '''', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

CREATE TYPE public.resource_type AS ENUM (
    ''WORKSPACE'',
    ''ENVIRONMENT'',
    ''PROJECT'',
    ''INSTANCE'',
    ''DATABASE''
);

CREATE TYPE public.row_status AS ENUM (
    ''NORMAL'',
    ''ARCHIVED''
);

CREATE FUNCTION public.trigger_update_updated_ts() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  NEW.updated_ts = extract(epoch from now());
  RETURN NEW;
END;
$$;

SET default_tablespace = '''';

SET default_table_access_method = heap;

CREATE TABLE public.activity (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    container_id integer NOT NULL,
    type text NOT NULL,
    level text NOT NULL,
    comment text DEFAULT ''''::text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT activity_container_id_check CHECK ((container_id > 0)),
    CONSTRAINT activity_level_check CHECK ((level = ANY (ARRAY[''INFO''::text, ''WARN''::text, ''ERROR''::text]))),
    CONSTRAINT activity_type_check CHECK ((type ~~ ''bb.%''::text))
);

CREATE SEQUENCE public.activity_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.activity_id_seq OWNED BY public.activity.id;

CREATE TABLE public.anomaly (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    database_id integer,
    type text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT anomaly_type_check CHECK ((type ~~ ''bb.anomaly.%''::text))
);

CREATE SEQUENCE public.anomaly_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.anomaly_id_seq OWNED BY public.anomaly.id;

CREATE TABLE public.backup (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    storage_backend text NOT NULL,
    migration_history_version text NOT NULL,
    path text NOT NULL,
    comment text DEFAULT ''''::text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT backup_status_check CHECK ((status = ANY (ARRAY[''PENDING_CREATE''::text, ''DONE''::text, ''FAILED''::text]))),
    CONSTRAINT backup_storage_backend_check CHECK ((storage_backend = ANY (ARRAY[''LOCAL''::text, ''S3''::text, ''GCS''::text, ''OSS''::text]))),
    CONSTRAINT backup_type_check CHECK ((type = ANY (ARRAY[''MANUAL''::text, ''AUTOMATIC''::text, ''PITR''::text])))
);

CREATE SEQUENCE public.backup_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.backup_id_seq OWNED BY public.backup.id;

CREATE TABLE public.backup_setting (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    enabled boolean NOT NULL,
    hour integer NOT NULL,
    day_of_week integer NOT NULL,
    retention_period_ts integer DEFAULT 0 NOT NULL,
    hook_url text NOT NULL,
    CONSTRAINT backup_setting_day_of_week_check CHECK (((day_of_week >= ''-1''::integer) AND (day_of_week <= 6))),
    CONSTRAINT backup_setting_hour_check CHECK (((hour >= 0) AND (hour <= 23))),
    CONSTRAINT backup_setting_retention_period_ts_check CHECK ((retention_period_ts >= 0))
);

CREATE SEQUENCE public.backup_setting_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.backup_setting_id_seq OWNED BY public.backup_setting.id;

CREATE TABLE public.bookmark (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    link text NOT NULL
);

CREATE SEQUENCE public.bookmark_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.bookmark_id_seq OWNED BY public.bookmark.id;

CREATE TABLE public.changelist (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.changelist_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.changelist_id_seq OWNED BY public.changelist.id;

CREATE TABLE public.data_source (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    username text NOT NULL,
    password text NOT NULL,
    ssl_key text DEFAULT ''''::text NOT NULL,
    ssl_cert text DEFAULT ''''::text NOT NULL,
    ssl_ca text DEFAULT ''''::text NOT NULL,
    host text DEFAULT ''''::text NOT NULL,
    port text DEFAULT ''''::text NOT NULL,
    options jsonb DEFAULT ''{}''::jsonb NOT NULL,
    database text DEFAULT ''''::text NOT NULL,
    CONSTRAINT data_source_type_check CHECK ((type = ANY (ARRAY[''ADMIN''::text, ''RW''::text, ''RO''::text])))
);

CREATE SEQUENCE public.data_source_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.data_source_id_seq OWNED BY public.data_source.id;

CREATE TABLE public.db (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    project_id integer NOT NULL,
    environment_id integer,
    source_backup_id integer,
    sync_status text NOT NULL,
    last_successful_sync_ts bigint NOT NULL,
    schema_version text NOT NULL,
    name text NOT NULL,
    secrets jsonb DEFAULT ''{}''::jsonb NOT NULL,
    datashare boolean DEFAULT false NOT NULL,
    service_name text DEFAULT ''''::text NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT db_sync_status_check CHECK ((sync_status = ANY (ARRAY[''OK''::text, ''NOT_FOUND''::text])))
);

CREATE TABLE public.db_group (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT ''''::text NOT NULL,
    expression jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.db_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_group_id_seq OWNED BY public.db_group.id;

CREATE SEQUENCE public.db_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_id_seq OWNED BY public.db.id;

CREATE TABLE public.db_schema (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL,
    raw_dump text DEFAULT ''''::text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.db_schema_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_schema_id_seq OWNED BY public.db_schema.id;

CREATE TABLE public.deployment_config (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.deployment_config_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.deployment_config_id_seq OWNED BY public.deployment_config.id;

CREATE TABLE public.environment (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    "order" integer NOT NULL,
    resource_id text NOT NULL,
    CONSTRAINT environment_order_check CHECK (("order" >= 0))
);

CREATE SEQUENCE public.environment_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.environment_id_seq OWNED BY public.environment.id;

CREATE TABLE public.external_approval (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    issue_id integer NOT NULL,
    requester_id integer NOT NULL,
    approver_id integer NOT NULL,
    type text NOT NULL,
    payload jsonb NOT NULL,
    CONSTRAINT external_approval_type_check CHECK ((type ~~ ''bb.plugin.app.%''::text))
);

CREATE SEQUENCE public.external_approval_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.external_approval_id_seq OWNED BY public.external_approval.id;

CREATE TABLE public.idp (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    resource_id text NOT NULL,
    name text NOT NULL,
    domain text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT idp_type_check CHECK ((type = ANY (ARRAY[''OAUTH2''::text, ''OIDC''::text, ''LDAP''::text])))
);

CREATE SEQUENCE public.idp_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.idp_id_seq OWNED BY public.idp.id;

CREATE TABLE public.inbox (
    id integer NOT NULL,
    receiver_id integer NOT NULL,
    activity_id integer NOT NULL,
    status text NOT NULL,
    CONSTRAINT inbox_status_check CHECK ((status = ANY (ARRAY[''UNREAD''::text, ''READ''::text])))
);

CREATE SEQUENCE public.inbox_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.inbox_id_seq OWNED BY public.inbox.id;

CREATE TABLE public.instance (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    environment_id integer,
    name text NOT NULL,
    engine text NOT NULL,
    engine_version text DEFAULT ''''::text NOT NULL,
    external_link text DEFAULT ''''::text NOT NULL,
    resource_id text NOT NULL,
    activation boolean DEFAULT false NOT NULL,
    options jsonb DEFAULT ''{}''::jsonb NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE TABLE public.instance_change_history (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer,
    database_id integer,
    issue_id integer,
    release_version text NOT NULL,
    sequence bigint NOT NULL,
    source text NOT NULL,
    type text NOT NULL,
    status text NOT NULL,
    version text NOT NULL,
    description text NOT NULL,
    statement text NOT NULL,
    sheet_id bigint,
    schema text NOT NULL,
    schema_prev text NOT NULL,
    execution_duration_ns bigint NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT instance_change_history_sequence_check CHECK ((sequence >= 0)),
    CONSTRAINT instance_change_history_source_check CHECK ((source = ANY (ARRAY[''UI''::text, ''VCS''::text, ''LIBRARY''::text]))),
    CONSTRAINT instance_change_history_status_check CHECK ((status = ANY (ARRAY[''PENDING''::text, ''DONE''::text, ''FAILED''::text]))),
    CONSTRAINT instance_change_history_type_check CHECK ((type = ANY (ARRAY[''BASELINE''::text, ''MIGRATE''::text, ''MIGRATE_SDL''::text, ''BRANCH''::text, ''DATA''::text])))
);

CREATE SEQUENCE public.instance_change_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.instance_change_history_id_seq OWNED BY public.instance_change_history.id;

CREATE SEQUENCE public.instance_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.instance_id_seq OWNED BY public.instance.id;

CREATE TABLE public.instance_user (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    name text NOT NULL,
    "grant" text NOT NULL
);

CREATE SEQUENCE public.instance_user_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.instance_user_id_seq OWNED BY public.instance_user.id;

CREATE TABLE public.issue (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    plan_id bigint,
    pipeline_id integer,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    description text DEFAULT ''''::text NOT NULL,
    assignee_id integer NOT NULL,
    assignee_need_attention boolean DEFAULT false NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    ts_vector tsvector,
    CONSTRAINT issue_status_check CHECK ((status = ANY (ARRAY[''OPEN''::text, ''DONE''::text, ''CANCELED''::text]))),
    CONSTRAINT issue_type_check CHECK ((type ~~ ''bb.issue.%''::text))
);

CREATE SEQUENCE public.issue_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.issue_id_seq OWNED BY public.issue.id;

CREATE TABLE public.issue_subscriber (
    issue_id integer NOT NULL,
    subscriber_id integer NOT NULL
);

CREATE TABLE public.member (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    status text NOT NULL,
    role text NOT NULL,
    principal_id integer NOT NULL,
    CONSTRAINT member_role_check CHECK ((role = ANY (ARRAY[''OWNER''::text, ''DBA''::text, ''DEVELOPER''::text]))),
    CONSTRAINT member_status_check CHECK ((status = ANY (ARRAY[''INVITED''::text, ''ACTIVE''::text])))
);

CREATE SEQUENCE public.member_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.member_id_seq OWNED BY public.member.id;

CREATE TABLE public.pipeline (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL
);

CREATE SEQUENCE public.pipeline_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.pipeline_id_seq OWNED BY public.pipeline.id;

CREATE TABLE public.plan (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    pipeline_id integer,
    name text NOT NULL,
    description text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE TABLE public.plan_check_run (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    plan_id bigint NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL,
    result jsonb DEFAULT ''{}''::jsonb NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT plan_check_run_status_check CHECK ((status = ANY (ARRAY[''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text]))),
    CONSTRAINT plan_check_run_type_check CHECK ((type ~~ ''bb.plan-check.%''::text))
);

CREATE SEQUENCE public.plan_check_run_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.plan_check_run_id_seq OWNED BY public.plan_check_run.id;

CREATE SEQUENCE public.plan_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.plan_id_seq OWNED BY public.plan.id;

CREATE TABLE public.policy (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    resource_type public.resource_type NOT NULL,
    resource_id integer NOT NULL,
    inherit_from_parent boolean DEFAULT true NOT NULL,
    CONSTRAINT policy_type_check CHECK ((type ~~ ''bb.policy.%''::text))
);

CREATE SEQUENCE public.policy_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.policy_id_seq OWNED BY public.policy.id;

CREATE TABLE public.principal (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text DEFAULT ''''::text NOT NULL,
    mfa_config jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT principal_type_check CHECK ((type = ANY (ARRAY[''END_USER''::text, ''SYSTEM_BOT''::text, ''SERVICE_ACCOUNT''::text])))
);

CREATE SEQUENCE public.principal_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.principal_id_seq OWNED BY public.principal.id;

CREATE TABLE public.project (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    key text NOT NULL,
    workflow_type text NOT NULL,
    visibility text NOT NULL,
    tenant_mode text DEFAULT ''DISABLED''::text NOT NULL,
    db_name_template text NOT NULL,
    schema_change_type text DEFAULT ''DDL''::text NOT NULL,
    resource_id text NOT NULL,
    data_classification_config_id text DEFAULT ''''::text NOT NULL,
    setting jsonb DEFAULT ''{}''::jsonb NOT NULL,
    schema_version_type text DEFAULT ''TIMESTAMP''::text NOT NULL,
    CONSTRAINT project_schema_change_type_check CHECK ((schema_change_type = ANY (ARRAY[''DDL''::text, ''SDL''::text]))),
    CONSTRAINT project_schema_version_type_check CHECK ((schema_version_type = ANY (ARRAY[''TIMESTAMP''::text, ''SEMANTIC''::text]))),
    CONSTRAINT project_tenant_mode_check CHECK ((tenant_mode = ANY (ARRAY[''DISABLED''::text, ''TENANT''::text]))),
    CONSTRAINT project_visibility_check CHECK ((visibility = ANY (ARRAY[''PUBLIC''::text, ''PRIVATE''::text]))),
    CONSTRAINT project_workflow_type_check CHECK ((workflow_type = ANY (ARRAY[''UI''::text, ''VCS''::text])))
);

CREATE SEQUENCE public.project_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.project_id_seq OWNED BY public.project.id;

CREATE TABLE public.project_member (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    role text NOT NULL,
    principal_id integer NOT NULL,
    condition jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.project_member_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.project_member_id_seq OWNED BY public.project_member.id;

CREATE TABLE public.project_webhook (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    activity_list text[] NOT NULL,
    CONSTRAINT project_webhook_type_check CHECK ((type ~~ ''bb.plugin.webhook.%''::text))
);

CREATE SEQUENCE public.project_webhook_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.project_webhook_id_seq OWNED BY public.project_webhook.id;

CREATE TABLE public.repository (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    vcs_id integer NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL,
    full_path text NOT NULL,
    web_url text NOT NULL,
    branch_filter text DEFAULT ''''::text NOT NULL,
    base_directory text DEFAULT ''''::text NOT NULL,
    file_path_template text DEFAULT ''''::text NOT NULL,
    enable_sql_review_ci boolean DEFAULT false NOT NULL,
    schema_path_template text DEFAULT ''''::text NOT NULL,
    sheet_path_template text DEFAULT ''''::text NOT NULL,
    external_id text NOT NULL,
    external_webhook_id text NOT NULL,
    webhook_url_host text NOT NULL,
    webhook_endpoint_id text NOT NULL,
    webhook_secret_token text NOT NULL,
    access_token text NOT NULL,
    expires_ts bigint NOT NULL,
    refresh_token text NOT NULL
);

CREATE SEQUENCE public.repository_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.repository_id_seq OWNED BY public.repository.id;

CREATE TABLE public.risk (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    source text NOT NULL,
    level bigint NOT NULL,
    name text NOT NULL,
    active boolean NOT NULL,
    expression jsonb NOT NULL,
    CONSTRAINT risk_source_check CHECK ((source ~~ ''bb.risk.%''::text))
);

CREATE SEQUENCE public.risk_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.risk_id_seq OWNED BY public.risk.id;

CREATE TABLE public.role (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    resource_id text NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    permissions jsonb DEFAULT ''{}''::jsonb NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.role_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.role_id_seq OWNED BY public.role.id;

CREATE TABLE public.schema_group (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    db_group_id bigint NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT ''''::text NOT NULL,
    expression jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.schema_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.schema_group_id_seq OWNED BY public.schema_group.id;

CREATE TABLE public.setting (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    value text NOT NULL,
    description text DEFAULT ''''::text NOT NULL
);

CREATE SEQUENCE public.setting_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.setting_id_seq OWNED BY public.setting.id;

CREATE TABLE public.sheet (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    database_id integer,
    name text NOT NULL,
    statement text NOT NULL,
    visibility text DEFAULT ''PRIVATE''::text NOT NULL,
    source text DEFAULT ''BYTEBASE''::text NOT NULL,
    type text DEFAULT ''SQL''::text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT sheet_source_check CHECK ((source = ANY (ARRAY[''BYTEBASE''::text, ''GITLAB''::text, ''GITHUB''::text, ''BITBUCKET''::text, ''AZURE_DEVOPS''::text, ''BYTEBASE_ARTIFACT''::text]))),
    CONSTRAINT sheet_type_check CHECK ((type = ''SQL''::text)),
    CONSTRAINT sheet_visibility_check CHECK ((visibility = ANY (ARRAY[''PRIVATE''::text, ''PROJECT''::text, ''PUBLIC''::text])))
);

CREATE SEQUENCE public.sheet_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sheet_id_seq OWNED BY public.sheet.id;

CREATE TABLE public.sheet_organizer (
    id integer NOT NULL,
    sheet_id integer NOT NULL,
    principal_id integer NOT NULL,
    starred boolean DEFAULT false NOT NULL,
    pinned boolean DEFAULT false NOT NULL
);

CREATE SEQUENCE public.sheet_organizer_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sheet_organizer_id_seq OWNED BY public.sheet_organizer.id;

CREATE TABLE public.slow_query (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    database_id integer,
    log_date_ts integer NOT NULL,
    slow_query_statistics jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.slow_query_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.slow_query_id_seq OWNED BY public.slow_query.id;

CREATE TABLE public.stage (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    pipeline_id integer NOT NULL,
    environment_id integer NOT NULL,
    name text NOT NULL
);

CREATE SEQUENCE public.stage_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.stage_id_seq OWNED BY public.stage.id;

CREATE TABLE public.task (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    pipeline_id integer NOT NULL,
    stage_id integer NOT NULL,
    instance_id integer NOT NULL,
    database_id integer,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    earliest_allowed_ts bigint DEFAULT 0 NOT NULL,
    CONSTRAINT task_status_check CHECK ((status = ANY (ARRAY[''PENDING''::text, ''PENDING_APPROVAL''::text, ''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text]))),
    CONSTRAINT task_type_check CHECK ((type ~~ ''bb.task.%''::text))
);

CREATE TABLE public.task_dag (
    id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    from_task_id integer NOT NULL,
    to_task_id integer NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.task_dag_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.task_dag_id_seq OWNED BY public.task_dag.id;

CREATE SEQUENCE public.task_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.task_id_seq OWNED BY public.task.id;

CREATE TABLE public.task_run (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    task_id integer NOT NULL,
    attempt integer NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    started_ts bigint DEFAULT 0 NOT NULL,
    code integer DEFAULT 0 NOT NULL,
    result jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT task_run_status_check CHECK ((status = ANY (ARRAY[''PENDING''::text, ''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text])))
);

CREATE SEQUENCE public.task_run_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.task_run_id_seq OWNED BY public.task_run.id;

CREATE TABLE public.vcs (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    instance_url text NOT NULL,
    api_url text NOT NULL,
    application_id text NOT NULL,
    secret text NOT NULL,
    CONSTRAINT vcs_api_url_check CHECK ((((api_url ~~ ''http://%''::text) OR (api_url ~~ ''https://%''::text)) AND (api_url = rtrim(api_url, ''/''::text)))),
    CONSTRAINT vcs_instance_url_check CHECK ((((instance_url ~~ ''http://%''::text) OR (instance_url ~~ ''https://%''::text)) AND (instance_url = rtrim(instance_url, ''/''::text)))),
    CONSTRAINT vcs_type_check CHECK ((type = ANY (ARRAY[''GITLAB''::text, ''GITHUB''::text, ''BITBUCKET''::text, ''AZURE_DEVOPS''::text])))
);

CREATE SEQUENCE public.vcs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.vcs_id_seq OWNED BY public.vcs.id;

ALTER TABLE ONLY public.activity ALTER COLUMN id SET DEFAULT nextval(''public.activity_id_seq''::regclass);

ALTER TABLE ONLY public.anomaly ALTER COLUMN id SET DEFAULT nextval(''public.anomaly_id_seq''::regclass);

ALTER TABLE ONLY public.backup ALTER COLUMN id SET DEFAULT nextval(''public.backup_id_seq''::regclass);

ALTER TABLE ONLY public.backup_setting ALTER COLUMN id SET DEFAULT nextval(''public.backup_setting_id_seq''::regclass);

ALTER TABLE ONLY public.bookmark ALTER COLUMN id SET DEFAULT nextval(''public.bookmark_id_seq''::regclass);

ALTER TABLE ONLY public.changelist ALTER COLUMN id SET DEFAULT nextval(''public.changelist_id_seq''::regclass);

ALTER TABLE ONLY public.data_source ALTER COLUMN id SET DEFAULT nextval(''public.data_source_id_seq''::regclass);

ALTER TABLE ONLY public.db ALTER COLUMN id SET DEFAULT nextval(''public.db_id_seq''::regclass);

ALTER TABLE ONLY public.db_group ALTER COLUMN id SET DEFAULT nextval(''public.db_group_id_seq''::regclass);

ALTER TABLE ONLY public.db_schema ALTER COLUMN id SET DEFAULT nextval(''public.db_schema_id_seq''::regclass);

ALTER TABLE ONLY public.deployment_config ALTER COLUMN id SET DEFAULT nextval(''public.deployment_config_id_seq''::regclass);

ALTER TABLE ONLY public.environment ALTER COLUMN id SET DEFAULT nextval(''public.environment_id_seq''::regclass);

ALTER TABLE ONLY public.external_approval ALTER COLUMN id SET DEFAULT nextval(''public.external_approval_id_seq''::regclass);

ALTER TABLE ONLY public.idp ALTER COLUMN id SET DEFAULT nextval(''public.idp_id_seq''::regclass);

ALTER TABLE ONLY public.inbox ALTER COLUMN id SET DEFAULT nextval(''public.inbox_id_seq''::regclass);

ALTER TABLE ONLY public.instance ALTER COLUMN id SET DEFAULT nextval(''public.instance_id_seq''::regclass);

ALTER TABLE ONLY public.instance_change_history ALTER COLUMN id SET DEFAULT nextval(''public.instance_change_history_id_seq''::regclass);

ALTER TABLE ONLY public.instance_user ALTER COLUMN id SET DEFAULT nextval(''public.instance_user_id_seq''::regclass);

ALTER TABLE ONLY public.issue ALTER COLUMN id SET DEFAULT nextval(''public.issue_id_seq''::regclass);

ALTER TABLE ONLY public.member ALTER COLUMN id SET DEFAULT nextval(''public.member_id_seq''::regclass);

ALTER TABLE ONLY public.pipeline ALTER COLUMN id SET DEFAULT nextval(''public.pipeline_id_seq''::regclass);

ALTER TABLE ONLY public.plan ALTER COLUMN id SET DEFAULT nextval(''public.plan_id_seq''::regclass);

ALTER TABLE ONLY public.plan_check_run ALTER COLUMN id SET DEFAULT nextval(''public.plan_check_run_id_seq''::regclass);

ALTER TABLE ONLY public.policy ALTER COLUMN id SET DEFAULT nextval(''public.policy_id_seq''::regclass);

ALTER TABLE ONLY public.principal ALTER COLUMN id SET DEFAULT nextval(''public.principal_id_seq''::regclass);

ALTER TABLE ONLY public.project ALTER COLUMN id SET DEFAULT nextval(''public.project_id_seq''::regclass);

ALTER TABLE ONLY public.project_member ALTER COLUMN id SET DEFAULT nextval(''public.project_member_id_seq''::regclass);

ALTER TABLE ONLY public.project_webhook ALTER COLUMN id SET DEFAULT nextval(''public.project_webhook_id_seq''::regclass);

ALTER TABLE ONLY public.repository ALTER COLUMN id SET DEFAULT nextval(''public.repository_id_seq''::regclass);

ALTER TABLE ONLY public.risk ALTER COLUMN id SET DEFAULT nextval(''public.risk_id_seq''::regclass);

ALTER TABLE ONLY public.role ALTER COLUMN id SET DEFAULT nextval(''public.role_id_seq''::regclass);

ALTER TABLE ONLY public.schema_group ALTER COLUMN id SET DEFAULT nextval(''public.schema_group_id_seq''::regclass);

ALTER TABLE ONLY public.setting ALTER COLUMN id SET DEFAULT nextval(''public.setting_id_seq''::regclass);

ALTER TABLE ONLY public.sheet ALTER COLUMN id SET DEFAULT nextval(''public.sheet_id_seq''::regclass);

ALTER TABLE ONLY public.sheet_organizer ALTER COLUMN id SET DEFAULT nextval(''public.sheet_organizer_id_seq''::regclass);

ALTER TABLE ONLY public.slow_query ALTER COLUMN id SET DEFAULT nextval(''public.slow_query_id_seq''::regclass);

ALTER TABLE ONLY public.stage ALTER COLUMN id SET DEFAULT nextval(''public.stage_id_seq''::regclass);

ALTER TABLE ONLY public.task ALTER COLUMN id SET DEFAULT nextval(''public.task_id_seq''::regclass);

ALTER TABLE ONLY public.task_dag ALTER COLUMN id SET DEFAULT nextval(''public.task_dag_id_seq''::regclass);

ALTER TABLE ONLY public.task_run ALTER COLUMN id SET DEFAULT nextval(''public.task_run_id_seq''::regclass);

ALTER TABLE ONLY public.vcs ALTER COLUMN id SET DEFAULT nextval(''public.vcs_id_seq''::regclass);

ALTER TABLE ONLY public.activity
    ADD CONSTRAINT activity_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.backup
    ADD CONSTRAINT backup_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.backup_setting
    ADD CONSTRAINT backup_setting_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.bookmark
    ADD CONSTRAINT bookmark_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.idp
    ADD CONSTRAINT idp_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.inbox
    ADD CONSTRAINT inbox_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_pkey PRIMARY KEY (issue_id, subscriber_id);

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.schema_group
    ADD CONSTRAINT schema_group_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.sheet_organizer
    ADD CONSTRAINT sheet_organizer_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task_dag
    ADD CONSTRAINT task_dag_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.vcs
    ADD CONSTRAINT vcs_pkey PRIMARY KEY (id);

CREATE INDEX idx_activity_container_id ON public.activity USING btree (container_id);

CREATE INDEX idx_activity_created_ts ON public.activity USING btree (created_ts);

CREATE INDEX idx_anomaly_database_id_row_status_type ON public.anomaly USING btree (database_id, row_status, type);

CREATE INDEX idx_anomaly_instance_id_row_status_type ON public.anomaly USING btree (instance_id, row_status, type);

CREATE INDEX idx_backup_database_id ON public.backup USING btree (database_id);

CREATE UNIQUE INDEX idx_backup_setting_unique_database_id ON public.backup_setting USING btree (database_id);

CREATE UNIQUE INDEX idx_backup_unique_database_id_name ON public.backup USING btree (database_id, name);

CREATE UNIQUE INDEX idx_bookmark_unique_creator_id_link ON public.bookmark USING btree (creator_id, link);

CREATE UNIQUE INDEX idx_changelist_project_id_name ON public.changelist USING btree (project_id, name);

CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON public.data_source USING btree (instance_id, name);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON public.db_group USING btree (project_id, placeholder);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON public.db_group USING btree (project_id, resource_id);

CREATE INDEX idx_db_instance_id ON public.db USING btree (instance_id);

CREATE INDEX idx_db_project_id ON public.db USING btree (project_id);

CREATE UNIQUE INDEX idx_db_schema_unique_database_id ON public.db_schema USING btree (database_id);

CREATE UNIQUE INDEX idx_db_unique_instance_id_name ON public.db USING btree (instance_id, name);

CREATE UNIQUE INDEX idx_deployment_config_unique_project_id ON public.deployment_config USING btree (project_id);

CREATE UNIQUE INDEX idx_environment_unique_name ON public.environment USING btree (name);

CREATE UNIQUE INDEX idx_environment_unique_resource_id ON public.environment USING btree (resource_id);

CREATE INDEX idx_external_approval_row_status_issue_id ON public.external_approval USING btree (row_status, issue_id);

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON public.idp USING btree (resource_id);

CREATE INDEX idx_inbox_receiver_id_activity_id ON public.inbox USING btree (receiver_id, activity_id);

CREATE INDEX idx_inbox_receiver_id_status ON public.inbox USING btree (receiver_id, status);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_sequ ON public.instance_change_history USING btree (instance_id, database_id, sequence);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_vers ON public.instance_change_history USING btree (instance_id, database_id, version);

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON public.instance USING btree (resource_id);

CREATE UNIQUE INDEX idx_instance_user_unique_instance_id_name ON public.instance_user USING btree (instance_id, name);

CREATE INDEX idx_issue_assignee_id ON public.issue USING btree (assignee_id);

CREATE INDEX idx_issue_created_ts ON public.issue USING btree (created_ts);

CREATE INDEX idx_issue_creator_id ON public.issue USING btree (creator_id);

CREATE INDEX idx_issue_pipeline_id ON public.issue USING btree (pipeline_id);

CREATE INDEX idx_issue_plan_id ON public.issue USING btree (plan_id);

CREATE INDEX idx_issue_project_id ON public.issue USING btree (project_id);

CREATE INDEX idx_issue_subscriber_subscriber_id ON public.issue_subscriber USING btree (subscriber_id);

CREATE INDEX idx_issue_ts_vector ON public.issue USING gin (ts_vector);

CREATE UNIQUE INDEX idx_member_unique_principal_id ON public.member USING btree (principal_id);

CREATE INDEX idx_plan_check_run_plan_id ON public.plan_check_run USING btree (plan_id);

CREATE INDEX idx_plan_pipeline_id ON public.plan USING btree (pipeline_id);

CREATE INDEX idx_plan_project_id ON public.plan USING btree (project_id);

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_id_type ON public.policy USING btree (resource_type, resource_id, type);

CREATE INDEX idx_project_member_project_id ON public.project_member USING btree (project_id);

CREATE UNIQUE INDEX idx_project_unique_key ON public.project USING btree (key);

CREATE UNIQUE INDEX idx_project_unique_resource_id ON public.project USING btree (resource_id);

CREATE INDEX idx_project_webhook_project_id ON public.project_webhook USING btree (project_id);

CREATE UNIQUE INDEX idx_project_webhook_unique_project_id_url ON public.project_webhook USING btree (project_id, url);

CREATE UNIQUE INDEX idx_repository_unique_project_id ON public.repository USING btree (project_id);

CREATE UNIQUE INDEX idx_role_unique_resource_id ON public.role USING btree (resource_id);

CREATE UNIQUE INDEX idx_schema_group_unique_db_group_id_placeholder ON public.schema_group USING btree (db_group_id, placeholder);

CREATE UNIQUE INDEX idx_schema_group_unique_db_group_id_resource_id ON public.schema_group USING btree (db_group_id, resource_id);

CREATE UNIQUE INDEX idx_setting_unique_name ON public.setting USING btree (name);

CREATE INDEX idx_sheet_creator_id ON public.sheet USING btree (creator_id);

CREATE INDEX idx_sheet_database_id_row_status ON public.sheet USING btree (database_id, row_status);

CREATE INDEX idx_sheet_name ON public.sheet USING btree (name);

CREATE INDEX idx_sheet_organizer_principal_id ON public.sheet_organizer USING btree (principal_id);

CREATE UNIQUE INDEX idx_sheet_organizer_unique_sheet_id_principal_id ON public.sheet_organizer USING btree (sheet_id, principal_id);

CREATE INDEX idx_sheet_project_id ON public.sheet USING btree (project_id);

CREATE INDEX idx_sheet_project_id_row_status ON public.sheet USING btree (project_id, row_status);

CREATE INDEX idx_slow_query_instance_id_log_date_ts ON public.slow_query USING btree (instance_id, log_date_ts);

CREATE INDEX idx_stage_pipeline_id ON public.stage USING btree (pipeline_id);

CREATE INDEX idx_task_dag_from_task_id ON public.task_dag USING btree (from_task_id);

CREATE INDEX idx_task_dag_to_task_id ON public.task_dag USING btree (to_task_id);

CREATE INDEX idx_task_earliest_allowed_ts ON public.task USING btree (earliest_allowed_ts);

CREATE INDEX idx_task_pipeline_id_stage_id ON public.task USING btree (pipeline_id, stage_id);

CREATE INDEX idx_task_run_task_id ON public.task_run USING btree (task_id);

CREATE INDEX idx_task_status ON public.task USING btree (status);

CREATE UNIQUE INDEX uk_slow_query_database_id_log_date_ts ON public.slow_query USING btree (database_id, log_date_ts);

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON public.task_run USING btree (task_id, attempt);

CREATE TRIGGER update_activity_updated_ts BEFORE UPDATE ON public.activity FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_anomaly_updated_ts BEFORE UPDATE ON public.anomaly FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_backup_setting_updated_ts BEFORE UPDATE ON public.backup_setting FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_backup_updated_ts BEFORE UPDATE ON public.backup FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_bookmark_updated_ts BEFORE UPDATE ON public.bookmark FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_changelist_updated_ts BEFORE UPDATE ON public.changelist FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_data_source_updated_ts BEFORE UPDATE ON public.data_source FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_group_updated_ts BEFORE UPDATE ON public.db_group FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_schema_updated_ts BEFORE UPDATE ON public.db_schema FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_updated_ts BEFORE UPDATE ON public.db FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_deployment_config_updated_ts BEFORE UPDATE ON public.deployment_config FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_environment_updated_ts BEFORE UPDATE ON public.environment FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_external_approval_updated_ts BEFORE UPDATE ON public.external_approval FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_idp_updated_ts BEFORE UPDATE ON public.idp FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_instance_change_history_updated_ts BEFORE UPDATE ON public.instance_change_history FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_instance_updated_ts BEFORE UPDATE ON public.instance FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_instance_user_updated_ts BEFORE UPDATE ON public.instance_user FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_issue_updated_ts BEFORE UPDATE ON public.issue FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_member_updated_ts BEFORE UPDATE ON public.member FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_pipeline_updated_ts BEFORE UPDATE ON public.pipeline FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_plan_check_run_updated_ts BEFORE UPDATE ON public.plan_check_run FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_plan_updated_ts BEFORE UPDATE ON public.plan FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_policy_updated_ts BEFORE UPDATE ON public.policy FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_principal_updated_ts BEFORE UPDATE ON public.principal FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_project_member_updated_ts BEFORE UPDATE ON public.project_member FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_project_updated_ts BEFORE UPDATE ON public.project FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_project_webhook_updated_ts BEFORE UPDATE ON public.project_webhook FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_repository_updated_ts BEFORE UPDATE ON public.repository FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_risk_updated_ts BEFORE UPDATE ON public.risk FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_role_updated_ts BEFORE UPDATE ON public.role FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_schema_group_updated_ts BEFORE UPDATE ON public.schema_group FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_setting_updated_ts BEFORE UPDATE ON public.setting FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_sheet_updated_ts BEFORE UPDATE ON public.sheet FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_slow_query_updated_ts BEFORE UPDATE ON public.slow_query FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_stage_updated_ts BEFORE UPDATE ON public.stage FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_task_dag_updated_ts BEFORE UPDATE ON public.task_dag FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_task_run_updated_ts BEFORE UPDATE ON public.task_run FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_task_updated_ts BEFORE UPDATE ON public.task FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_vcs_updated_ts BEFORE UPDATE ON public.vcs FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

ALTER TABLE ONLY public.activity
    ADD CONSTRAINT activity_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.activity
    ADD CONSTRAINT activity_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.backup
    ADD CONSTRAINT backup_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.backup
    ADD CONSTRAINT backup_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.backup_setting
    ADD CONSTRAINT backup_setting_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.backup_setting
    ADD CONSTRAINT backup_setting_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.backup_setting
    ADD CONSTRAINT backup_setting_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.backup
    ADD CONSTRAINT backup_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.bookmark
    ADD CONSTRAINT bookmark_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.bookmark
    ADD CONSTRAINT bookmark_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_environment_id_fkey FOREIGN KEY (environment_id) REFERENCES public.environment(id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_approver_id_fkey FOREIGN KEY (approver_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_requester_id_fkey FOREIGN KEY (requester_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.inbox
    ADD CONSTRAINT inbox_activity_id_fkey FOREIGN KEY (activity_id) REFERENCES public.activity(id);

ALTER TABLE ONLY public.inbox
    ADD CONSTRAINT inbox_receiver_id_fkey FOREIGN KEY (receiver_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_environment_id_fkey FOREIGN KEY (environment_id) REFERENCES public.environment(id);

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_assignee_id_fkey FOREIGN KEY (assignee_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plan(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plan(id);

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_vcs_id_fkey FOREIGN KEY (vcs_id) REFERENCES public.vcs(id);

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.schema_group
    ADD CONSTRAINT schema_group_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.schema_group
    ADD CONSTRAINT schema_group_db_group_id_fkey FOREIGN KEY (db_group_id) REFERENCES public.db_group(id);

ALTER TABLE ONLY public.schema_group
    ADD CONSTRAINT schema_group_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.sheet_organizer
    ADD CONSTRAINT sheet_organizer_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.sheet_organizer
    ADD CONSTRAINT sheet_organizer_sheet_id_fkey FOREIGN KEY (sheet_id) REFERENCES public.sheet(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_environment_id_fkey FOREIGN KEY (environment_id) REFERENCES public.environment(id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.task_dag
    ADD CONSTRAINT task_dag_from_task_id_fkey FOREIGN KEY (from_task_id) REFERENCES public.task(id);

ALTER TABLE ONLY public.task_dag
    ADD CONSTRAINT task_dag_to_task_id_fkey FOREIGN KEY (to_task_id) REFERENCES public.task(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.task(id);

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.stage(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.vcs
    ADD CONSTRAINT vcs_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.vcs
    ADD CONSTRAINT vcs_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

', '
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = ''UTF8'';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config(''search_path'', '''', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

CREATE TYPE public.resource_type AS ENUM (
    ''WORKSPACE'',
    ''ENVIRONMENT'',
    ''PROJECT'',
    ''INSTANCE'',
    ''DATABASE''
);

CREATE TYPE public.row_status AS ENUM (
    ''NORMAL'',
    ''ARCHIVED''
);

CREATE FUNCTION public.trigger_update_updated_ts() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  NEW.updated_ts = extract(epoch from now());
  RETURN NEW;
END;
$$;

SET default_tablespace = '''';

SET default_table_access_method = heap;

CREATE TABLE public.activity (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    container_id integer NOT NULL,
    type text NOT NULL,
    level text NOT NULL,
    comment text DEFAULT ''''::text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT activity_container_id_check CHECK ((container_id > 0)),
    CONSTRAINT activity_level_check CHECK ((level = ANY (ARRAY[''INFO''::text, ''WARN''::text, ''ERROR''::text]))),
    CONSTRAINT activity_type_check CHECK ((type ~~ ''bb.%''::text))
);

CREATE SEQUENCE public.activity_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.activity_id_seq OWNED BY public.activity.id;

CREATE TABLE public.anomaly (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    database_id integer,
    type text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT anomaly_type_check CHECK ((type ~~ ''bb.anomaly.%''::text))
);

CREATE SEQUENCE public.anomaly_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.anomaly_id_seq OWNED BY public.anomaly.id;

CREATE TABLE public.backup (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    storage_backend text NOT NULL,
    migration_history_version text NOT NULL,
    path text NOT NULL,
    comment text DEFAULT ''''::text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT backup_status_check CHECK ((status = ANY (ARRAY[''PENDING_CREATE''::text, ''DONE''::text, ''FAILED''::text]))),
    CONSTRAINT backup_storage_backend_check CHECK ((storage_backend = ANY (ARRAY[''LOCAL''::text, ''S3''::text, ''GCS''::text, ''OSS''::text]))),
    CONSTRAINT backup_type_check CHECK ((type = ANY (ARRAY[''MANUAL''::text, ''AUTOMATIC''::text, ''PITR''::text])))
);

CREATE SEQUENCE public.backup_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.backup_id_seq OWNED BY public.backup.id;

CREATE TABLE public.backup_setting (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    enabled boolean NOT NULL,
    hour integer NOT NULL,
    day_of_week integer NOT NULL,
    retention_period_ts integer DEFAULT 0 NOT NULL,
    hook_url text NOT NULL,
    CONSTRAINT backup_setting_day_of_week_check CHECK (((day_of_week >= ''-1''::integer) AND (day_of_week <= 6))),
    CONSTRAINT backup_setting_hour_check CHECK (((hour >= 0) AND (hour <= 23))),
    CONSTRAINT backup_setting_retention_period_ts_check CHECK ((retention_period_ts >= 0))
);

CREATE SEQUENCE public.backup_setting_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.backup_setting_id_seq OWNED BY public.backup_setting.id;

CREATE TABLE public.bookmark (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    link text NOT NULL
);

CREATE SEQUENCE public.bookmark_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.bookmark_id_seq OWNED BY public.bookmark.id;

CREATE TABLE public.changelist (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.changelist_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.changelist_id_seq OWNED BY public.changelist.id;

CREATE TABLE public.data_source (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    username text NOT NULL,
    password text NOT NULL,
    ssl_key text DEFAULT ''''::text NOT NULL,
    ssl_cert text DEFAULT ''''::text NOT NULL,
    ssl_ca text DEFAULT ''''::text NOT NULL,
    host text DEFAULT ''''::text NOT NULL,
    port text DEFAULT ''''::text NOT NULL,
    options jsonb DEFAULT ''{}''::jsonb NOT NULL,
    database text DEFAULT ''''::text NOT NULL,
    CONSTRAINT data_source_type_check CHECK ((type = ANY (ARRAY[''ADMIN''::text, ''RW''::text, ''RO''::text])))
);

CREATE SEQUENCE public.data_source_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.data_source_id_seq OWNED BY public.data_source.id;

CREATE TABLE public.db (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    project_id integer NOT NULL,
    environment_id integer,
    source_backup_id integer,
    sync_status text NOT NULL,
    last_successful_sync_ts bigint NOT NULL,
    schema_version text NOT NULL,
    name text NOT NULL,
    secrets jsonb DEFAULT ''{}''::jsonb NOT NULL,
    datashare boolean DEFAULT false NOT NULL,
    service_name text DEFAULT ''''::text NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT db_sync_status_check CHECK ((sync_status = ANY (ARRAY[''OK''::text, ''NOT_FOUND''::text])))
);

CREATE TABLE public.db_group (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT ''''::text NOT NULL,
    expression jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.db_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_group_id_seq OWNED BY public.db_group.id;

CREATE SEQUENCE public.db_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_id_seq OWNED BY public.db.id;

CREATE TABLE public.db_schema (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL,
    raw_dump text DEFAULT ''''::text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.db_schema_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_schema_id_seq OWNED BY public.db_schema.id;

CREATE TABLE public.deployment_config (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.deployment_config_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.deployment_config_id_seq OWNED BY public.deployment_config.id;

CREATE TABLE public.environment (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    "order" integer NOT NULL,
    resource_id text NOT NULL,
    CONSTRAINT environment_order_check CHECK (("order" >= 0))
);

CREATE SEQUENCE public.environment_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.environment_id_seq OWNED BY public.environment.id;

CREATE TABLE public.external_approval (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    issue_id integer NOT NULL,
    requester_id integer NOT NULL,
    approver_id integer NOT NULL,
    type text NOT NULL,
    payload jsonb NOT NULL,
    CONSTRAINT external_approval_type_check CHECK ((type ~~ ''bb.plugin.app.%''::text))
);

CREATE SEQUENCE public.external_approval_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.external_approval_id_seq OWNED BY public.external_approval.id;

CREATE TABLE public.idp (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    resource_id text NOT NULL,
    name text NOT NULL,
    domain text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT idp_type_check CHECK ((type = ANY (ARRAY[''OAUTH2''::text, ''OIDC''::text, ''LDAP''::text])))
);

CREATE SEQUENCE public.idp_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.idp_id_seq OWNED BY public.idp.id;

CREATE TABLE public.inbox (
    id integer NOT NULL,
    receiver_id integer NOT NULL,
    activity_id integer NOT NULL,
    status text NOT NULL,
    CONSTRAINT inbox_status_check CHECK ((status = ANY (ARRAY[''UNREAD''::text, ''READ''::text])))
);

CREATE SEQUENCE public.inbox_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.inbox_id_seq OWNED BY public.inbox.id;

CREATE TABLE public.instance (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    environment_id integer,
    name text NOT NULL,
    engine text NOT NULL,
    engine_version text DEFAULT ''''::text NOT NULL,
    external_link text DEFAULT ''''::text NOT NULL,
    resource_id text NOT NULL,
    activation boolean DEFAULT false NOT NULL,
    options jsonb DEFAULT ''{}''::jsonb NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE TABLE public.instance_change_history (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer,
    database_id integer,
    issue_id integer,
    release_version text NOT NULL,
    sequence bigint NOT NULL,
    source text NOT NULL,
    type text NOT NULL,
    status text NOT NULL,
    version text NOT NULL,
    description text NOT NULL,
    statement text NOT NULL,
    sheet_id bigint,
    schema text NOT NULL,
    schema_prev text NOT NULL,
    execution_duration_ns bigint NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT instance_change_history_sequence_check CHECK ((sequence >= 0)),
    CONSTRAINT instance_change_history_source_check CHECK ((source = ANY (ARRAY[''UI''::text, ''VCS''::text, ''LIBRARY''::text]))),
    CONSTRAINT instance_change_history_status_check CHECK ((status = ANY (ARRAY[''PENDING''::text, ''DONE''::text, ''FAILED''::text]))),
    CONSTRAINT instance_change_history_type_check CHECK ((type = ANY (ARRAY[''BASELINE''::text, ''MIGRATE''::text, ''MIGRATE_SDL''::text, ''BRANCH''::text, ''DATA''::text])))
);

CREATE SEQUENCE public.instance_change_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.instance_change_history_id_seq OWNED BY public.instance_change_history.id;

CREATE SEQUENCE public.instance_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.instance_id_seq OWNED BY public.instance.id;

CREATE TABLE public.instance_user (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    name text NOT NULL,
    "grant" text NOT NULL
);

CREATE SEQUENCE public.instance_user_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.instance_user_id_seq OWNED BY public.instance_user.id;

CREATE TABLE public.issue (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    plan_id bigint,
    pipeline_id integer,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    description text DEFAULT ''''::text NOT NULL,
    assignee_id integer NOT NULL,
    assignee_need_attention boolean DEFAULT false NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    ts_vector tsvector,
    CONSTRAINT issue_status_check CHECK ((status = ANY (ARRAY[''OPEN''::text, ''DONE''::text, ''CANCELED''::text]))),
    CONSTRAINT issue_type_check CHECK ((type ~~ ''bb.issue.%''::text))
);

CREATE SEQUENCE public.issue_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.issue_id_seq OWNED BY public.issue.id;

CREATE TABLE public.issue_subscriber (
    issue_id integer NOT NULL,
    subscriber_id integer NOT NULL
);

CREATE TABLE public.member (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    status text NOT NULL,
    role text NOT NULL,
    principal_id integer NOT NULL,
    CONSTRAINT member_role_check CHECK ((role = ANY (ARRAY[''OWNER''::text, ''DBA''::text, ''DEVELOPER''::text]))),
    CONSTRAINT member_status_check CHECK ((status = ANY (ARRAY[''INVITED''::text, ''ACTIVE''::text])))
);

CREATE SEQUENCE public.member_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.member_id_seq OWNED BY public.member.id;

CREATE TABLE public.pipeline (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL
);

CREATE SEQUENCE public.pipeline_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.pipeline_id_seq OWNED BY public.pipeline.id;

CREATE TABLE public.plan (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    pipeline_id integer,
    name text NOT NULL,
    description text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE TABLE public.plan_check_run (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    plan_id bigint NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT ''{}''::jsonb NOT NULL,
    result jsonb DEFAULT ''{}''::jsonb NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT plan_check_run_status_check CHECK ((status = ANY (ARRAY[''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text]))),
    CONSTRAINT plan_check_run_type_check CHECK ((type ~~ ''bb.plan-check.%''::text))
);

CREATE SEQUENCE public.plan_check_run_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.plan_check_run_id_seq OWNED BY public.plan_check_run.id;

CREATE SEQUENCE public.plan_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.plan_id_seq OWNED BY public.plan.id;

CREATE TABLE public.policy (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    resource_type public.resource_type NOT NULL,
    resource_id integer NOT NULL,
    inherit_from_parent boolean DEFAULT true NOT NULL,
    CONSTRAINT policy_type_check CHECK ((type ~~ ''bb.policy.%''::text))
);

CREATE SEQUENCE public.policy_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.policy_id_seq OWNED BY public.policy.id;

CREATE TABLE public.principal (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text DEFAULT ''''::text NOT NULL,
    mfa_config jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT principal_type_check CHECK ((type = ANY (ARRAY[''END_USER''::text, ''SYSTEM_BOT''::text, ''SERVICE_ACCOUNT''::text])))
);

CREATE SEQUENCE public.principal_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.principal_id_seq OWNED BY public.principal.id;

CREATE TABLE public.project (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    key text NOT NULL,
    workflow_type text NOT NULL,
    visibility text NOT NULL,
    tenant_mode text DEFAULT ''DISABLED''::text NOT NULL,
    db_name_template text NOT NULL,
    schema_change_type text DEFAULT ''DDL''::text NOT NULL,
    resource_id text NOT NULL,
    data_classification_config_id text DEFAULT ''''::text NOT NULL,
    setting jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT project_schema_change_type_check CHECK ((schema_change_type = ANY (ARRAY[''DDL''::text, ''SDL''::text]))),
    CONSTRAINT project_tenant_mode_check CHECK ((tenant_mode = ANY (ARRAY[''DISABLED''::text, ''TENANT''::text]))),
    CONSTRAINT project_visibility_check CHECK ((visibility = ANY (ARRAY[''PUBLIC''::text, ''PRIVATE''::text]))),
    CONSTRAINT project_workflow_type_check CHECK ((workflow_type = ANY (ARRAY[''UI''::text, ''VCS''::text])))
);

CREATE SEQUENCE public.project_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.project_id_seq OWNED BY public.project.id;

CREATE TABLE public.project_member (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    role text NOT NULL,
    principal_id integer NOT NULL,
    condition jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.project_member_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.project_member_id_seq OWNED BY public.project_member.id;

CREATE TABLE public.project_webhook (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    activity_list text[] NOT NULL,
    CONSTRAINT project_webhook_type_check CHECK ((type ~~ ''bb.plugin.webhook.%''::text))
);

CREATE SEQUENCE public.project_webhook_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.project_webhook_id_seq OWNED BY public.project_webhook.id;

CREATE TABLE public.repository (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    vcs_id integer NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL,
    full_path text NOT NULL,
    web_url text NOT NULL,
    branch_filter text DEFAULT ''''::text NOT NULL,
    base_directory text DEFAULT ''''::text NOT NULL,
    file_path_template text DEFAULT ''''::text NOT NULL,
    enable_sql_review_ci boolean DEFAULT false NOT NULL,
    schema_path_template text DEFAULT ''''::text NOT NULL,
    sheet_path_template text DEFAULT ''''::text NOT NULL,
    external_id text NOT NULL,
    external_webhook_id text NOT NULL,
    webhook_url_host text NOT NULL,
    webhook_endpoint_id text NOT NULL,
    webhook_secret_token text NOT NULL,
    access_token text NOT NULL,
    expires_ts bigint NOT NULL,
    refresh_token text NOT NULL
);

CREATE SEQUENCE public.repository_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.repository_id_seq OWNED BY public.repository.id;

CREATE TABLE public.risk (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    source text NOT NULL,
    level bigint NOT NULL,
    name text NOT NULL,
    active boolean NOT NULL,
    expression jsonb NOT NULL,
    CONSTRAINT risk_source_check CHECK ((source ~~ ''bb.risk.%''::text))
);

CREATE SEQUENCE public.risk_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.risk_id_seq OWNED BY public.risk.id;

CREATE TABLE public.role (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    resource_id text NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    permissions jsonb DEFAULT ''{}''::jsonb NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.role_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.role_id_seq OWNED BY public.role.id;

CREATE TABLE public.schema_group (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    db_group_id bigint NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT ''''::text NOT NULL,
    expression jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.schema_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.schema_group_id_seq OWNED BY public.schema_group.id;

CREATE TABLE public.setting (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    value text NOT NULL,
    description text DEFAULT ''''::text NOT NULL
);

CREATE SEQUENCE public.setting_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.setting_id_seq OWNED BY public.setting.id;

CREATE TABLE public.sheet (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    database_id integer,
    name text NOT NULL,
    statement text NOT NULL,
    visibility text DEFAULT ''PRIVATE''::text NOT NULL,
    source text DEFAULT ''BYTEBASE''::text NOT NULL,
    type text DEFAULT ''SQL''::text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT sheet_source_check CHECK ((source = ANY (ARRAY[''BYTEBASE''::text, ''GITLAB''::text, ''GITHUB''::text, ''BITBUCKET''::text, ''AZURE_DEVOPS''::text, ''BYTEBASE_ARTIFACT''::text]))),
    CONSTRAINT sheet_type_check CHECK ((type = ''SQL''::text)),
    CONSTRAINT sheet_visibility_check CHECK ((visibility = ANY (ARRAY[''PRIVATE''::text, ''PROJECT''::text, ''PUBLIC''::text])))
);

CREATE SEQUENCE public.sheet_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sheet_id_seq OWNED BY public.sheet.id;

CREATE TABLE public.sheet_organizer (
    id integer NOT NULL,
    sheet_id integer NOT NULL,
    principal_id integer NOT NULL,
    starred boolean DEFAULT false NOT NULL,
    pinned boolean DEFAULT false NOT NULL
);

CREATE SEQUENCE public.sheet_organizer_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sheet_organizer_id_seq OWNED BY public.sheet_organizer.id;

CREATE TABLE public.slow_query (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    database_id integer,
    log_date_ts integer NOT NULL,
    slow_query_statistics jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.slow_query_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.slow_query_id_seq OWNED BY public.slow_query.id;

CREATE TABLE public.stage (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    pipeline_id integer NOT NULL,
    environment_id integer NOT NULL,
    name text NOT NULL
);

CREATE SEQUENCE public.stage_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.stage_id_seq OWNED BY public.stage.id;

CREATE TABLE public.task (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    pipeline_id integer NOT NULL,
    stage_id integer NOT NULL,
    instance_id integer NOT NULL,
    database_id integer,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL,
    earliest_allowed_ts bigint DEFAULT 0 NOT NULL,
    CONSTRAINT task_status_check CHECK ((status = ANY (ARRAY[''PENDING''::text, ''PENDING_APPROVAL''::text, ''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text]))),
    CONSTRAINT task_type_check CHECK ((type ~~ ''bb.task.%''::text))
);

CREATE TABLE public.task_dag (
    id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    from_task_id integer NOT NULL,
    to_task_id integer NOT NULL,
    payload jsonb DEFAULT ''{}''::jsonb NOT NULL
);

CREATE SEQUENCE public.task_dag_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.task_dag_id_seq OWNED BY public.task_dag.id;

CREATE SEQUENCE public.task_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.task_id_seq OWNED BY public.task.id;

CREATE TABLE public.task_run (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    task_id integer NOT NULL,
    attempt integer NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    started_ts bigint DEFAULT 0 NOT NULL,
    code integer DEFAULT 0 NOT NULL,
    result jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT task_run_status_check CHECK ((status = ANY (ARRAY[''PENDING''::text, ''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text])))
);

CREATE SEQUENCE public.task_run_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.task_run_id_seq OWNED BY public.task_run.id;

CREATE TABLE public.vcs (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    instance_url text NOT NULL,
    api_url text NOT NULL,
    application_id text NOT NULL,
    secret text NOT NULL,
    CONSTRAINT vcs_api_url_check CHECK ((((api_url ~~ ''http://%''::text) OR (api_url ~~ ''https://%''::text)) AND (api_url = rtrim(api_url, ''/''::text)))),
    CONSTRAINT vcs_instance_url_check CHECK ((((instance_url ~~ ''http://%''::text) OR (instance_url ~~ ''https://%''::text)) AND (instance_url = rtrim(instance_url, ''/''::text)))),
    CONSTRAINT vcs_type_check CHECK ((type = ANY (ARRAY[''GITLAB''::text, ''GITHUB''::text, ''BITBUCKET''::text, ''AZURE_DEVOPS''::text])))
);

CREATE SEQUENCE public.vcs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.vcs_id_seq OWNED BY public.vcs.id;

ALTER TABLE ONLY public.activity ALTER COLUMN id SET DEFAULT nextval(''public.activity_id_seq''::regclass);

ALTER TABLE ONLY public.anomaly ALTER COLUMN id SET DEFAULT nextval(''public.anomaly_id_seq''::regclass);

ALTER TABLE ONLY public.backup ALTER COLUMN id SET DEFAULT nextval(''public.backup_id_seq''::regclass);

ALTER TABLE ONLY public.backup_setting ALTER COLUMN id SET DEFAULT nextval(''public.backup_setting_id_seq''::regclass);

ALTER TABLE ONLY public.bookmark ALTER COLUMN id SET DEFAULT nextval(''public.bookmark_id_seq''::regclass);

ALTER TABLE ONLY public.changelist ALTER COLUMN id SET DEFAULT nextval(''public.changelist_id_seq''::regclass);

ALTER TABLE ONLY public.data_source ALTER COLUMN id SET DEFAULT nextval(''public.data_source_id_seq''::regclass);

ALTER TABLE ONLY public.db ALTER COLUMN id SET DEFAULT nextval(''public.db_id_seq''::regclass);

ALTER TABLE ONLY public.db_group ALTER COLUMN id SET DEFAULT nextval(''public.db_group_id_seq''::regclass);

ALTER TABLE ONLY public.db_schema ALTER COLUMN id SET DEFAULT nextval(''public.db_schema_id_seq''::regclass);

ALTER TABLE ONLY public.deployment_config ALTER COLUMN id SET DEFAULT nextval(''public.deployment_config_id_seq''::regclass);

ALTER TABLE ONLY public.environment ALTER COLUMN id SET DEFAULT nextval(''public.environment_id_seq''::regclass);

ALTER TABLE ONLY public.external_approval ALTER COLUMN id SET DEFAULT nextval(''public.external_approval_id_seq''::regclass);

ALTER TABLE ONLY public.idp ALTER COLUMN id SET DEFAULT nextval(''public.idp_id_seq''::regclass);

ALTER TABLE ONLY public.inbox ALTER COLUMN id SET DEFAULT nextval(''public.inbox_id_seq''::regclass);

ALTER TABLE ONLY public.instance ALTER COLUMN id SET DEFAULT nextval(''public.instance_id_seq''::regclass);

ALTER TABLE ONLY public.instance_change_history ALTER COLUMN id SET DEFAULT nextval(''public.instance_change_history_id_seq''::regclass);

ALTER TABLE ONLY public.instance_user ALTER COLUMN id SET DEFAULT nextval(''public.instance_user_id_seq''::regclass);

ALTER TABLE ONLY public.issue ALTER COLUMN id SET DEFAULT nextval(''public.issue_id_seq''::regclass);

ALTER TABLE ONLY public.member ALTER COLUMN id SET DEFAULT nextval(''public.member_id_seq''::regclass);

ALTER TABLE ONLY public.pipeline ALTER COLUMN id SET DEFAULT nextval(''public.pipeline_id_seq''::regclass);

ALTER TABLE ONLY public.plan ALTER COLUMN id SET DEFAULT nextval(''public.plan_id_seq''::regclass);

ALTER TABLE ONLY public.plan_check_run ALTER COLUMN id SET DEFAULT nextval(''public.plan_check_run_id_seq''::regclass);

ALTER TABLE ONLY public.policy ALTER COLUMN id SET DEFAULT nextval(''public.policy_id_seq''::regclass);

ALTER TABLE ONLY public.principal ALTER COLUMN id SET DEFAULT nextval(''public.principal_id_seq''::regclass);

ALTER TABLE ONLY public.project ALTER COLUMN id SET DEFAULT nextval(''public.project_id_seq''::regclass);

ALTER TABLE ONLY public.project_member ALTER COLUMN id SET DEFAULT nextval(''public.project_member_id_seq''::regclass);

ALTER TABLE ONLY public.project_webhook ALTER COLUMN id SET DEFAULT nextval(''public.project_webhook_id_seq''::regclass);

ALTER TABLE ONLY public.repository ALTER COLUMN id SET DEFAULT nextval(''public.repository_id_seq''::regclass);

ALTER TABLE ONLY public.risk ALTER COLUMN id SET DEFAULT nextval(''public.risk_id_seq''::regclass);

ALTER TABLE ONLY public.role ALTER COLUMN id SET DEFAULT nextval(''public.role_id_seq''::regclass);

ALTER TABLE ONLY public.schema_group ALTER COLUMN id SET DEFAULT nextval(''public.schema_group_id_seq''::regclass);

ALTER TABLE ONLY public.setting ALTER COLUMN id SET DEFAULT nextval(''public.setting_id_seq''::regclass);

ALTER TABLE ONLY public.sheet ALTER COLUMN id SET DEFAULT nextval(''public.sheet_id_seq''::regclass);

ALTER TABLE ONLY public.sheet_organizer ALTER COLUMN id SET DEFAULT nextval(''public.sheet_organizer_id_seq''::regclass);

ALTER TABLE ONLY public.slow_query ALTER COLUMN id SET DEFAULT nextval(''public.slow_query_id_seq''::regclass);

ALTER TABLE ONLY public.stage ALTER COLUMN id SET DEFAULT nextval(''public.stage_id_seq''::regclass);

ALTER TABLE ONLY public.task ALTER COLUMN id SET DEFAULT nextval(''public.task_id_seq''::regclass);

ALTER TABLE ONLY public.task_dag ALTER COLUMN id SET DEFAULT nextval(''public.task_dag_id_seq''::regclass);

ALTER TABLE ONLY public.task_run ALTER COLUMN id SET DEFAULT nextval(''public.task_run_id_seq''::regclass);

ALTER TABLE ONLY public.vcs ALTER COLUMN id SET DEFAULT nextval(''public.vcs_id_seq''::regclass);

ALTER TABLE ONLY public.activity
    ADD CONSTRAINT activity_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.backup
    ADD CONSTRAINT backup_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.backup_setting
    ADD CONSTRAINT backup_setting_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.bookmark
    ADD CONSTRAINT bookmark_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.idp
    ADD CONSTRAINT idp_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.inbox
    ADD CONSTRAINT inbox_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_pkey PRIMARY KEY (issue_id, subscriber_id);

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.schema_group
    ADD CONSTRAINT schema_group_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.sheet_organizer
    ADD CONSTRAINT sheet_organizer_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task_dag
    ADD CONSTRAINT task_dag_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.vcs
    ADD CONSTRAINT vcs_pkey PRIMARY KEY (id);

CREATE INDEX idx_activity_container_id ON public.activity USING btree (container_id);

CREATE INDEX idx_activity_created_ts ON public.activity USING btree (created_ts);

CREATE INDEX idx_anomaly_database_id_row_status_type ON public.anomaly USING btree (database_id, row_status, type);

CREATE INDEX idx_anomaly_instance_id_row_status_type ON public.anomaly USING btree (instance_id, row_status, type);

CREATE INDEX idx_backup_database_id ON public.backup USING btree (database_id);

CREATE UNIQUE INDEX idx_backup_setting_unique_database_id ON public.backup_setting USING btree (database_id);

CREATE UNIQUE INDEX idx_backup_unique_database_id_name ON public.backup USING btree (database_id, name);

CREATE UNIQUE INDEX idx_bookmark_unique_creator_id_link ON public.bookmark USING btree (creator_id, link);

CREATE UNIQUE INDEX idx_changelist_project_id_name ON public.changelist USING btree (project_id, name);

CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON public.data_source USING btree (instance_id, name);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON public.db_group USING btree (project_id, placeholder);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON public.db_group USING btree (project_id, resource_id);

CREATE INDEX idx_db_instance_id ON public.db USING btree (instance_id);

CREATE INDEX idx_db_project_id ON public.db USING btree (project_id);

CREATE UNIQUE INDEX idx_db_schema_unique_database_id ON public.db_schema USING btree (database_id);

CREATE UNIQUE INDEX idx_db_unique_instance_id_name ON public.db USING btree (instance_id, name);

CREATE UNIQUE INDEX idx_deployment_config_unique_project_id ON public.deployment_config USING btree (project_id);

CREATE UNIQUE INDEX idx_environment_unique_name ON public.environment USING btree (name);

CREATE UNIQUE INDEX idx_environment_unique_resource_id ON public.environment USING btree (resource_id);

CREATE INDEX idx_external_approval_row_status_issue_id ON public.external_approval USING btree (row_status, issue_id);

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON public.idp USING btree (resource_id);

CREATE INDEX idx_inbox_receiver_id_activity_id ON public.inbox USING btree (receiver_id, activity_id);

CREATE INDEX idx_inbox_receiver_id_status ON public.inbox USING btree (receiver_id, status);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_sequ ON public.instance_change_history USING btree (instance_id, database_id, sequence);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_vers ON public.instance_change_history USING btree (instance_id, database_id, version);

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON public.instance USING btree (resource_id);

CREATE UNIQUE INDEX idx_instance_user_unique_instance_id_name ON public.instance_user USING btree (instance_id, name);

CREATE INDEX idx_issue_assignee_id ON public.issue USING btree (assignee_id);

CREATE INDEX idx_issue_created_ts ON public.issue USING btree (created_ts);

CREATE INDEX idx_issue_creator_id ON public.issue USING btree (creator_id);

CREATE INDEX idx_issue_pipeline_id ON public.issue USING btree (pipeline_id);

CREATE INDEX idx_issue_plan_id ON public.issue USING btree (plan_id);

CREATE INDEX idx_issue_project_id ON public.issue USING btree (project_id);

CREATE INDEX idx_issue_subscriber_subscriber_id ON public.issue_subscriber USING btree (subscriber_id);

CREATE INDEX idx_issue_ts_vector ON public.issue USING gin (ts_vector);

CREATE UNIQUE INDEX idx_member_unique_principal_id ON public.member USING btree (principal_id);

CREATE INDEX idx_plan_check_run_plan_id ON public.plan_check_run USING btree (plan_id);

CREATE INDEX idx_plan_pipeline_id ON public.plan USING btree (pipeline_id);

CREATE INDEX idx_plan_project_id ON public.plan USING btree (project_id);

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_id_type ON public.policy USING btree (resource_type, resource_id, type);

CREATE INDEX idx_project_member_project_id ON public.project_member USING btree (project_id);

CREATE UNIQUE INDEX idx_project_unique_key ON public.project USING btree (key);

CREATE UNIQUE INDEX idx_project_unique_resource_id ON public.project USING btree (resource_id);

CREATE INDEX idx_project_webhook_project_id ON public.project_webhook USING btree (project_id);

CREATE UNIQUE INDEX idx_project_webhook_unique_project_id_url ON public.project_webhook USING btree (project_id, url);

CREATE UNIQUE INDEX idx_repository_unique_project_id ON public.repository USING btree (project_id);

CREATE UNIQUE INDEX idx_role_unique_resource_id ON public.role USING btree (resource_id);

CREATE UNIQUE INDEX idx_schema_group_unique_db_group_id_placeholder ON public.schema_group USING btree (db_group_id, placeholder);

CREATE UNIQUE INDEX idx_schema_group_unique_db_group_id_resource_id ON public.schema_group USING btree (db_group_id, resource_id);

CREATE UNIQUE INDEX idx_setting_unique_name ON public.setting USING btree (name);

CREATE INDEX idx_sheet_creator_id ON public.sheet USING btree (creator_id);

CREATE INDEX idx_sheet_database_id_row_status ON public.sheet USING btree (database_id, row_status);

CREATE INDEX idx_sheet_name ON public.sheet USING btree (name);

CREATE INDEX idx_sheet_organizer_principal_id ON public.sheet_organizer USING btree (principal_id);

CREATE UNIQUE INDEX idx_sheet_organizer_unique_sheet_id_principal_id ON public.sheet_organizer USING btree (sheet_id, principal_id);

CREATE INDEX idx_sheet_project_id ON public.sheet USING btree (project_id);

CREATE INDEX idx_sheet_project_id_row_status ON public.sheet USING btree (project_id, row_status);

CREATE INDEX idx_slow_query_instance_id_log_date_ts ON public.slow_query USING btree (instance_id, log_date_ts);

CREATE INDEX idx_stage_pipeline_id ON public.stage USING btree (pipeline_id);

CREATE INDEX idx_task_dag_from_task_id ON public.task_dag USING btree (from_task_id);

CREATE INDEX idx_task_dag_to_task_id ON public.task_dag USING btree (to_task_id);

CREATE INDEX idx_task_earliest_allowed_ts ON public.task USING btree (earliest_allowed_ts);

CREATE INDEX idx_task_pipeline_id_stage_id ON public.task USING btree (pipeline_id, stage_id);

CREATE INDEX idx_task_run_task_id ON public.task_run USING btree (task_id);

CREATE INDEX idx_task_status ON public.task USING btree (status);

CREATE UNIQUE INDEX uk_slow_query_database_id_log_date_ts ON public.slow_query USING btree (database_id, log_date_ts);

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON public.task_run USING btree (task_id, attempt);

CREATE TRIGGER update_activity_updated_ts BEFORE UPDATE ON public.activity FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_anomaly_updated_ts BEFORE UPDATE ON public.anomaly FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_backup_setting_updated_ts BEFORE UPDATE ON public.backup_setting FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_backup_updated_ts BEFORE UPDATE ON public.backup FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_bookmark_updated_ts BEFORE UPDATE ON public.bookmark FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_changelist_updated_ts BEFORE UPDATE ON public.changelist FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_data_source_updated_ts BEFORE UPDATE ON public.data_source FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_group_updated_ts BEFORE UPDATE ON public.db_group FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_schema_updated_ts BEFORE UPDATE ON public.db_schema FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_updated_ts BEFORE UPDATE ON public.db FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_deployment_config_updated_ts BEFORE UPDATE ON public.deployment_config FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_environment_updated_ts BEFORE UPDATE ON public.environment FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_external_approval_updated_ts BEFORE UPDATE ON public.external_approval FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_idp_updated_ts BEFORE UPDATE ON public.idp FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_instance_change_history_updated_ts BEFORE UPDATE ON public.instance_change_history FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_instance_updated_ts BEFORE UPDATE ON public.instance FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_instance_user_updated_ts BEFORE UPDATE ON public.instance_user FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_issue_updated_ts BEFORE UPDATE ON public.issue FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_member_updated_ts BEFORE UPDATE ON public.member FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_pipeline_updated_ts BEFORE UPDATE ON public.pipeline FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_plan_check_run_updated_ts BEFORE UPDATE ON public.plan_check_run FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_plan_updated_ts BEFORE UPDATE ON public.plan FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_policy_updated_ts BEFORE UPDATE ON public.policy FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_principal_updated_ts BEFORE UPDATE ON public.principal FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_project_member_updated_ts BEFORE UPDATE ON public.project_member FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_project_updated_ts BEFORE UPDATE ON public.project FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_project_webhook_updated_ts BEFORE UPDATE ON public.project_webhook FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_repository_updated_ts BEFORE UPDATE ON public.repository FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_risk_updated_ts BEFORE UPDATE ON public.risk FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_role_updated_ts BEFORE UPDATE ON public.role FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_schema_group_updated_ts BEFORE UPDATE ON public.schema_group FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_setting_updated_ts BEFORE UPDATE ON public.setting FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_sheet_updated_ts BEFORE UPDATE ON public.sheet FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_slow_query_updated_ts BEFORE UPDATE ON public.slow_query FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_stage_updated_ts BEFORE UPDATE ON public.stage FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_task_dag_updated_ts BEFORE UPDATE ON public.task_dag FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_task_run_updated_ts BEFORE UPDATE ON public.task_run FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_task_updated_ts BEFORE UPDATE ON public.task FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_vcs_updated_ts BEFORE UPDATE ON public.vcs FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

ALTER TABLE ONLY public.activity
    ADD CONSTRAINT activity_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.activity
    ADD CONSTRAINT activity_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.backup
    ADD CONSTRAINT backup_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.backup
    ADD CONSTRAINT backup_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.backup_setting
    ADD CONSTRAINT backup_setting_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.backup_setting
    ADD CONSTRAINT backup_setting_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.backup_setting
    ADD CONSTRAINT backup_setting_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.backup
    ADD CONSTRAINT backup_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.bookmark
    ADD CONSTRAINT bookmark_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.bookmark
    ADD CONSTRAINT bookmark_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_environment_id_fkey FOREIGN KEY (environment_id) REFERENCES public.environment(id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_approver_id_fkey FOREIGN KEY (approver_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_requester_id_fkey FOREIGN KEY (requester_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.inbox
    ADD CONSTRAINT inbox_activity_id_fkey FOREIGN KEY (activity_id) REFERENCES public.activity(id);

ALTER TABLE ONLY public.inbox
    ADD CONSTRAINT inbox_receiver_id_fkey FOREIGN KEY (receiver_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_environment_id_fkey FOREIGN KEY (environment_id) REFERENCES public.environment(id);

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_assignee_id_fkey FOREIGN KEY (assignee_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plan(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plan(id);

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.project_member
    ADD CONSTRAINT project_member_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_vcs_id_fkey FOREIGN KEY (vcs_id) REFERENCES public.vcs(id);

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.schema_group
    ADD CONSTRAINT schema_group_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.schema_group
    ADD CONSTRAINT schema_group_db_group_id_fkey FOREIGN KEY (db_group_id) REFERENCES public.db_group(id);

ALTER TABLE ONLY public.schema_group
    ADD CONSTRAINT schema_group_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.sheet_organizer
    ADD CONSTRAINT sheet_organizer_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.sheet_organizer
    ADD CONSTRAINT sheet_organizer_sheet_id_fkey FOREIGN KEY (sheet_id) REFERENCES public.sheet(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_environment_id_fkey FOREIGN KEY (environment_id) REFERENCES public.environment(id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.task_dag
    ADD CONSTRAINT task_dag_from_task_id_fkey FOREIGN KEY (from_task_id) REFERENCES public.task(id);

ALTER TABLE ONLY public.task_dag
    ADD CONSTRAINT task_dag_to_task_id_fkey FOREIGN KEY (to_task_id) REFERENCES public.task(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.task(id);

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.stage(id);

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.vcs
    ADD CONSTRAINT vcs_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.vcs
    ADD CONSTRAINT vcs_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

', 161523000, '{}') ON CONFLICT DO NOTHING;


ALTER TABLE public.instance_change_history ENABLE TRIGGER ALL;

--
-- Data for Name: instance_user; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.instance_user DISABLE TRIGGER ALL;

INSERT INTO public.instance_user (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, "grant") VALUES (101, 'NORMAL', 1, 1698731517, 1, 1698731517, 101, 'bbsample', 'Superuser, Create role, Create DB, Replication, Bypass RLS+') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_user (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, "grant") VALUES (102, 'NORMAL', 1, 1698731517, 1, 1698731517, 102, 'bbsample', 'Superuser, Create role, Create DB, Replication, Bypass RLS+') ON CONFLICT DO NOTHING;


ALTER TABLE public.instance_user ENABLE TRIGGER ALL;

--
-- Data for Name: issue_subscriber; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.issue_subscriber DISABLE TRIGGER ALL;



ALTER TABLE public.issue_subscriber ENABLE TRIGGER ALL;

--
-- Data for Name: member; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.member DISABLE TRIGGER ALL;

INSERT INTO public.member (id, row_status, creator_id, created_ts, updater_id, updated_ts, status, role, principal_id) VALUES (101, 'NORMAL', 1, 1698731517, 1, 1698731517, 'ACTIVE', 'OWNER', 101) ON CONFLICT DO NOTHING;


ALTER TABLE public.member ENABLE TRIGGER ALL;

--
-- Data for Name: plan_check_run; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.plan_check_run DISABLE TRIGGER ALL;

INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (108, 1, 1698731517, 1, 1698731517, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 103, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": "1000", "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (101, 1, 1698731517, 1, 1698731517, 101, 'DONE', 'bb.plan-check.database.connect', '{"instanceUid": 101, "databaseName": "hr_test"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (105, 1, 1698731517, 1, 1698731517, 101, 'DONE', 'bb.plan-check.database.connect', '{"instanceUid": 102, "databaseName": "hr_prod"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (107, 1, 1698731517, 1, 1698731517, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 103, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"email\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": "402", "line": "1"}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (103, 1, 1698731517, 1, 1698731517, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 102, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (102, 1, 1698731517, 1, 1698731517, 101, 'DONE', 'bb.plan-check.database.statement.type', '{"sheetUid": 102, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (106, 1, 1698731517, 1, 1698731517, 101, 'DONE', 'bb.plan-check.database.statement.type', '{"sheetUid": 103, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (104, 1, 1698731517, 1, 1698731517, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 102, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": "1000", "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_test", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;


ALTER TABLE public.plan_check_run ENABLE TRIGGER ALL;

--
-- Data for Name: policy; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.policy DISABLE TRIGGER ALL;

INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 'bb.policy.sql-review', '{"name": "SQL Review Sample Policy", "ruleList": [{"type": "database.drop-empty-database", "level": "ERROR", "engine": "MYSQL", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "TIDB", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "MARIADB", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MYSQL", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "TIDB", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MARIADB", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "POSTGRES", "payload": "{}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MYSQL", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "TIDB", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MARIADB", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "POSTGRES", "payload": "{\"format\":\"_del$\"}"}]}', 'ENVIRONMENT', 102, true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (102, 'NORMAL', 101, 1698731517, 101, 1698731517, 'bb.policy.masking', '{"mask_data": [{"table": "salary", "column": "amount", "schema": "public", "masking_level": 3}]}', 'DATABASE', 102, true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (103, 'NORMAL', 101, 1698732589, 101, 1698732589, 'bb.policy.rollout', '{"projectRoles": ["roles/OWNER"], "workspaceRoles": ["roles/DBA", "roles/OWNER"]}', 'ENVIRONMENT', 102, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (104, 'NORMAL', 101, 1698732589, 101, 1698732589, 'bb.policy.environment-tier', '{"environmentTier": "PROTECTED"}', 'ENVIRONMENT', 102, true) ON CONFLICT DO NOTHING;


ALTER TABLE public.policy ENABLE TRIGGER ALL;

--
-- Data for Name: project_member; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.project_member DISABLE TRIGGER ALL;

INSERT INTO public.project_member (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id, condition) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 'OWNER', 101, '{}') ON CONFLICT DO NOTHING;


ALTER TABLE public.project_member ENABLE TRIGGER ALL;

--
-- Data for Name: project_webhook; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.project_webhook DISABLE TRIGGER ALL;



ALTER TABLE public.project_webhook ENABLE TRIGGER ALL;

--
-- Data for Name: vcs; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.vcs DISABLE TRIGGER ALL;



ALTER TABLE public.vcs ENABLE TRIGGER ALL;

--
-- Data for Name: repository; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.repository DISABLE TRIGGER ALL;



ALTER TABLE public.repository ENABLE TRIGGER ALL;

--
-- Data for Name: risk; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.risk DISABLE TRIGGER ALL;

INSERT INTO public.risk (id, row_status, creator_id, created_ts, updater_id, updated_ts, source, level, name, active, expression) VALUES (101, 'NORMAL', 101, 1698734757, 101, 1698734757, 'bb.risk.database.schema.update', 300, 'The risk for the production environment is considered to be high.', true, '{"expression": "environment_id == \"prod\""}') ON CONFLICT DO NOTHING;


ALTER TABLE public.risk ENABLE TRIGGER ALL;

--
-- Data for Name: role; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.role DISABLE TRIGGER ALL;



ALTER TABLE public.role ENABLE TRIGGER ALL;

--
-- Data for Name: schema_group; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.schema_group DISABLE TRIGGER ALL;



ALTER TABLE public.schema_group ENABLE TRIGGER ALL;

--
-- Data for Name: setting; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.setting DISABLE TRIGGER ALL;

INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (101, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.branding.logo', '', 'The branding slogo image in base64 string format.') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (102, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.auth.secret', 'emc7Vwxf2eplDkYlT4JAP1KlZtGQMVwV', 'Random string used to sign the JWT auth token.') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (103, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.workspace.id', '8c846bef-6894-4b11-bc58-e564522d6350', 'The workspace identifier') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (105, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.app.im', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (106, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.workspace.watermark', '0', 'Display watermark') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (107, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.plugin.openai.key', '', 'API key to request OpenAI (ChatGPT)') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (108, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.plugin.openai.endpoint', '', 'API Endpoint for OpenAI') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (109, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.workspace.approval.external', '{}', 'The external approval setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (110, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.workspace.schema-template', '{}', 'The schema template setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (111, 'NORMAL', 1, 1698730969, 1, 1698730969, 'bb.workspace.data-classification', '{}', 'The data classification setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (113, 'NORMAL', 1, 1698730969, 101, 1698731971, 'bb.workspace.profile', '{"externalUrl":"https://demo.bytebase.com"}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (104, 'NORMAL', 1, 1698730969, 101, 1698732023, 'bb.enterprise.license', 'eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA', 'Enterprise license') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (113, 'NORMAL', 1, 1698730969, 1, 1698734296, 'bb.workspace.profile', '{"externalUrl":"https://demo.bytebase.com"}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (112, 'NORMAL', 1, 1698730969, 101, 1698734763, 'bb.workspace.approval', '{"rules":[{"template":{"flow":{"steps":[{"type":"ANY", "nodes":[{"type":"ANY_IN_GROUP", "groupValue":"PROJECT_OWNER"}]}, {"type":"ANY", "nodes":[{"type":"ANY_IN_GROUP", "groupValue":"WORKSPACE_DBA"}]}]}, "title":"Project Owner -> DBA", "description":"The system defines the approval process, first the project Owner approves, then the DBA approves.", "creatorId":1}, "condition":{"expression":"source == 1 && level == 300"}}, {"template":{"flow":{"steps":[{"type":"ANY", "nodes":[{"type":"ANY_IN_GROUP", "groupValue":"PROJECT_OWNER"}]}]}, "title":"Project Owner", "description":"The system defines the approval process and only needs the project Owner o approve it.", "creatorId":1}, "condition":{}}, {"template":{"flow":{"steps":[{"type":"ANY", "nodes":[{"type":"ANY_IN_GROUP", "groupValue":"WORKSPACE_DBA"}]}]}, "title":"DBA", "description":"The system defines the approval process and only needs DBA approval.", "creatorId":1}, "condition":{}}, {"template":{"flow":{"steps":[{"type":"ANY", "nodes":[{"type":"ANY_IN_GROUP", "groupValue":"WORKSPACE_OWNER"}]}]}, "title":"Workspace Owner", "description":"The system defines the approval process and only needs Administrator approval.", "creatorId":1}, "condition":{}}, {"template":{"flow":{"steps":[{"type":"ANY", "nodes":[{"type":"ANY_IN_GROUP", "groupValue":"PROJECT_OWNER"}]}, {"type":"ANY", "nodes":[{"type":"ANY_IN_GROUP", "groupValue":"WORKSPACE_DBA"}]}, {"type":"ANY", "nodes":[{"type":"ANY_IN_GROUP", "groupValue":"WORKSPACE_OWNER"}]}]}, "title":"Project Owner -> DBA -> Workspace Owner", "description":"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.", "creatorId":1}, "condition":{}}]}', 'The workspace approval setting') ON CONFLICT DO NOTHING;


ALTER TABLE public.setting ENABLE TRIGGER ALL;

--
-- Data for Name: sheet; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.sheet DISABLE TRIGGER ALL;

INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 102, 'Sample Sheet', 'SELECT * FROM salary;', 'PROJECT', 'BYTEBASE', 'SQL', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload) VALUES (102, 'NORMAL', 1, 1698731517, 1, 1698731517, 101, 101, 'Alter table to test sample instance for sample issue', 'ALTER TABLE employee ADD COLUMN IF NOT EXISTS email TEXT DEFAULT '''';', 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload) VALUES (103, 'NORMAL', 1, 1698731517, 1, 1698731517, 101, 102, 'Alter table to prod sample instance for sample issue', 'ALTER TABLE employee ADD COLUMN IF NOT EXISTS email TEXT DEFAULT '''';', 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}') ON CONFLICT DO NOTHING;


ALTER TABLE public.sheet ENABLE TRIGGER ALL;

--
-- Data for Name: sheet_organizer; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.sheet_organizer DISABLE TRIGGER ALL;



ALTER TABLE public.sheet_organizer ENABLE TRIGGER ALL;

--
-- Data for Name: slow_query; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.slow_query DISABLE TRIGGER ALL;



ALTER TABLE public.slow_query ENABLE TRIGGER ALL;

--
-- Data for Name: stage; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.stage DISABLE TRIGGER ALL;

INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, name) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 101, 'Test Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, name) VALUES (102, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 102, 'Prod Stage') ON CONFLICT DO NOTHING;


ALTER TABLE public.stage ENABLE TRIGGER ALL;

--
-- Data for Name: task; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.task DISABLE TRIGGER ALL;

INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (101, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 101, 101, 101, 'DDL(schema) for database "hr_test"', 'PENDING_APPROVAL', 'bb.task.database.schema.update', '{"sheetId": 102, "schemaVersion": "20231031135157"}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (102, 'NORMAL', 101, 1698731517, 101, 1698731517, 101, 102, 102, 102, 'DDL(schema) for database "hr_prod"', 'PENDING_APPROVAL', 'bb.task.database.schema.update', '{"sheetId": 103, "schemaVersion": "20231031135157"}', 0) ON CONFLICT DO NOTHING;


ALTER TABLE public.task ENABLE TRIGGER ALL;

--
-- Data for Name: task_dag; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.task_dag DISABLE TRIGGER ALL;



ALTER TABLE public.task_dag ENABLE TRIGGER ALL;

--
-- Data for Name: task_run; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.task_run DISABLE TRIGGER ALL;



ALTER TABLE public.task_run ENABLE TRIGGER ALL;

--
-- Name: activity_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.activity_id_seq', 105, true);


--
-- Name: anomaly_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.anomaly_id_seq', 101, false);


--
-- Name: backup_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.backup_id_seq', 101, false);


--
-- Name: backup_setting_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.backup_setting_id_seq', 101, false);


--
-- Name: bookmark_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.bookmark_id_seq', 101, true);


--
-- Name: changelist_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.changelist_id_seq', 101, false);


--
-- Name: data_source_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.data_source_id_seq', 115, true);


--
-- Name: db_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.db_group_id_seq', 101, false);


--
-- Name: db_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.db_id_seq', 102, true);


--
-- Name: db_schema_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.db_schema_id_seq', 102, true);


--
-- Name: deployment_config_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.deployment_config_id_seq', 101, false);


--
-- Name: environment_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.environment_id_seq', 103, false);


--
-- Name: external_approval_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.external_approval_id_seq', 101, false);


--
-- Name: idp_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.idp_id_seq', 101, false);


--
-- Name: inbox_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.inbox_id_seq', 101, true);


--
-- Name: instance_change_history_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.instance_change_history_id_seq', 102, true);


--
-- Name: instance_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.instance_id_seq', 115, true);


--
-- Name: instance_user_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.instance_user_id_seq', 102, true);


--
-- Name: issue_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.issue_id_seq', 101, true);


--
-- Name: member_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.member_id_seq', 101, true);


--
-- Name: pipeline_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.pipeline_id_seq', 101, true);


--
-- Name: plan_check_run_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.plan_check_run_id_seq', 108, true);


--
-- Name: plan_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.plan_id_seq', 101, true);


--
-- Name: policy_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.policy_id_seq', 104, true);


--
-- Name: principal_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.principal_id_seq', 101, true);


--
-- Name: project_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.project_id_seq', 101, true);


--
-- Name: project_member_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.project_member_id_seq', 101, true);


--
-- Name: project_webhook_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.project_webhook_id_seq', 101, false);


--
-- Name: repository_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.repository_id_seq', 101, false);


--
-- Name: risk_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.risk_id_seq', 101, false);


--
-- Name: role_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.role_id_seq', 101, false);


--
-- Name: schema_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.schema_group_id_seq', 101, false);


--
-- Name: setting_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.setting_id_seq', 117, true);


--
-- Name: sheet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.sheet_id_seq', 103, true);


--
-- Name: sheet_organizer_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.sheet_organizer_id_seq', 1, false);


--
-- Name: slow_query_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.slow_query_id_seq', 101, false);


--
-- Name: stage_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.stage_id_seq', 102, true);


--
-- Name: task_dag_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.task_dag_id_seq', 101, false);


--
-- Name: task_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.task_id_seq', 102, true);


--
-- Name: task_run_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.task_run_id_seq', 101, false);


--
-- Name: vcs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.vcs_id_seq', 101, false);


--
-- PostgreSQL database dump complete
--

