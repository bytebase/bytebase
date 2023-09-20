--
-- PostgreSQL database dump
--

-- Dumped from database version 14.3
-- Dumped by pg_dump version 14.3

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

INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config) VALUES (1, 'NORMAL', 1, 1695178575, 1, 1695178575, 'SYSTEM_BOT', 'Bytebase', 'support@bytebase.com', '', '', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config) VALUES (101, 'NORMAL', 1, 1694684977, 101, 1695112774, 'END_USER', 'Demo Owner', 'demo@example.com', '$2a$10$JbwDbh1u86G9UUCMKXehV.uKPQhZYEJIUiLpVXRkVM4pNAUnU1THG', '', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config) VALUES (102, 'NORMAL', 1, 1695112807, 101, 1695112895, 'END_USER', 'Jerry DBA', 'jerry@example.com', '$2a$10$GH4GKACLebRGpY3B/oAgNuuIg/FA/j0a5x7h9.AQxex1TfD8cb6ZG', '', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config) VALUES (103, 'NORMAL', 1, 1695112807, 101, 1695112903, 'END_USER', 'Tom Dev', 'tom@example.com', '$2a$10$5d6.P.g/jb8AmSdsdkqZE.fopcsRDPLdlSRSg.Homdbbl7GpEZPVq', '', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config) VALUES (104, 'NORMAL', 1, 1695112807, 101, 1695112915, 'END_USER', 'Jane Dev', 'jane@example.com', '$2a$10$g451CEsfAi8iTgAP/8hPWOklx/j9fCbl..XuEZDIg4QTUW1mZlcRe', '', '{}') ON CONFLICT DO NOTHING;


ALTER TABLE public.principal ENABLE TRIGGER ALL;

--
-- Data for Name: activity; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.activity DISABLE TRIGGER ALL;

INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (101, 'NORMAL', 101, 1694684977, 101, 1694684977, 101, 'bb.member.create', 'INFO', '', '{"role": "OWNER", "principalId": 101, "memberStatus": "ACTIVE", "principalName": "Demo", "principalEmail": "demo@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (102, 'NORMAL', 102, 1695112807, 102, 1695112807, 102, 'bb.member.create', 'INFO', '', '{"role": "DBA", "principalId": 102, "memberStatus": "ACTIVE", "principalName": "jerry", "principalEmail": "jerry@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (103, 'NORMAL', 103, 1695112807, 103, 1695112807, 103, 'bb.member.create', 'INFO', '', '{"role": "DEVELOPER", "principalId": 103, "memberStatus": "ACTIVE", "principalName": "tom", "principalEmail": "tom@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (104, 'NORMAL', 104, 1695112807, 104, 1695112807, 104, 'bb.member.create', 'INFO', '', '{"role": "DEVELOPER", "principalId": 104, "memberStatus": "ACTIVE", "principalName": "jane", "principalEmail": "jane@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (105, 'NORMAL', 101, 1695112950, 101, 1695112950, 101, 'bb.project.member.create', 'INFO', 'Granted Tom Dev to tom@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (106, 'NORMAL', 101, 1695112950, 101, 1695112950, 101, 'bb.project.member.create', 'INFO', 'Granted Jerry DBA to jerry@example.com (OWNER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (107, 'NORMAL', 101, 1695113006, 101, 1695113006, 101, 'bb.project.member.create', 'INFO', 'Granted Demo Owner to demo@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (108, 'NORMAL', 101, 1695113011, 101, 1695113011, 101, 'bb.project.member.delete', 'INFO', 'Revoked OWNER from Demo Owner (demo@example.com).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (109, 'NORMAL', 101, 1695178711, 101, 1695178711, 101, 'bb.issue.create', 'INFO', '', '{"issueName": "Create database ''sakila_prod''"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (110, 'NORMAL', 1, 1695178711, 1, 1695178711, 101, 'bb.issue.approval.notify', 'INFO', '', '{"approvalStep": {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (111, 'NORMAL', 102, 1695178964, 102, 1695178964, 101, 'bb.issue.comment.create', 'INFO', '', '{"issueName": "Create database ''sakila_prod''", "approvalEvent": {"status": "APPROVED"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (112, 'NORMAL', 101, 1695178988, 101, 1695178988, 101, 'bb.pipeline.taskrun.status.update', 'INFO', '', '{"taskId": 101, "taskName": "Create database sakila_prod", "issueName": "Create database ''sakila_prod''", "newStatus": "PENDING"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (113, 'NORMAL', 1, 1695178988, 1, 1695178988, 101, 'bb.pipeline.taskrun.status.update', 'INFO', '', '{"taskId": 101, "taskName": "Create database sakila_prod", "issueName": "Create database ''sakila_prod''", "newStatus": "DONE"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (114, 'NORMAL', 1, 1695178988, 1, 1695178988, 101, 'bb.pipeline.stage.status.update', 'INFO', '', '{"stageId": 101, "issueName": "Create database ''sakila_prod''", "stageName": "Prod Stage", "stageStatusUpdateType": "END"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (115, 'NORMAL', 1, 1695178988, 1, 1695178988, 101, 'bb.issue.status.update', 'INFO', '', '{"issueName": "Create database ''sakila_prod''", "newStatus": "DONE", "oldStatus": "OPEN"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (116, 'NORMAL', 101, 1695179030, 101, 1695179030, 102, 'bb.issue.create', 'INFO', '', '{"issueName": "Create database ''sakila_test''"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (117, 'NORMAL', 1, 1695179030, 1, 1695179030, 102, 'bb.issue.approval.notify', 'INFO', '', '{"approvalStep": {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (118, 'NORMAL', 102, 1695179041, 102, 1695179041, 102, 'bb.issue.comment.create', 'INFO', '', '{"issueName": "Create database ''sakila_test''", "approvalEvent": {"status": "APPROVED"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (119, 'NORMAL', 101, 1695179054, 101, 1695179054, 102, 'bb.pipeline.taskrun.status.update', 'INFO', '', '{"taskId": 102, "taskName": "Create database sakila_test", "issueName": "Create database ''sakila_test''", "newStatus": "PENDING"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (120, 'NORMAL', 1, 1695179054, 1, 1695179054, 102, 'bb.pipeline.taskrun.status.update', 'INFO', '', '{"taskId": 102, "taskName": "Create database sakila_test", "issueName": "Create database ''sakila_test''", "newStatus": "DONE"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (121, 'NORMAL', 1, 1695179054, 1, 1695179054, 102, 'bb.pipeline.stage.status.update', 'INFO', '', '{"stageId": 102, "issueName": "Create database ''sakila_test''", "stageName": "Prod Stage", "stageStatusUpdateType": "END"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (122, 'NORMAL', 1, 1695179054, 1, 1695179054, 102, 'bb.issue.status.update', 'INFO', '', '{"issueName": "Create database ''sakila_test''", "newStatus": "DONE", "oldStatus": "OPEN"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (123, 'NORMAL', 101, 1695179078, 101, 1695179078, 103, 'bb.issue.create', 'INFO', '', '{"issueName": "Create database ''sakila_staging''"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (124, 'NORMAL', 1, 1695179079, 1, 1695179079, 103, 'bb.issue.approval.notify', 'INFO', '', '{"approvalStep": {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (125, 'NORMAL', 102, 1695179089, 102, 1695179089, 103, 'bb.issue.comment.create', 'INFO', '', '{"issueName": "Create database ''sakila_staging''", "approvalEvent": {"status": "APPROVED"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (126, 'NORMAL', 101, 1695179141, 101, 1695179141, 103, 'bb.pipeline.taskrun.status.update', 'INFO', '', '{"taskId": 103, "taskName": "Create database sakila_staging", "issueName": "Create database ''sakila_staging''", "newStatus": "PENDING"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (127, 'NORMAL', 1, 1695179141, 1, 1695179141, 103, 'bb.pipeline.taskrun.status.update', 'INFO', '', '{"taskId": 103, "taskName": "Create database sakila_staging", "issueName": "Create database ''sakila_staging''", "newStatus": "DONE"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (128, 'NORMAL', 1, 1695179141, 1, 1695179141, 103, 'bb.pipeline.stage.status.update', 'INFO', '', '{"stageId": 103, "issueName": "Create database ''sakila_staging''", "stageName": "Prod Stage", "stageStatusUpdateType": "END"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload) VALUES (129, 'NORMAL', 1, 1695179141, 1, 1695179141, 103, 'bb.issue.status.update', 'INFO', '', '{"issueName": "Create database ''sakila_staging''", "newStatus": "DONE", "oldStatus": "OPEN"}') ON CONFLICT DO NOTHING;


ALTER TABLE public.activity ENABLE TRIGGER ALL;

--
-- Data for Name: environment; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.environment DISABLE TRIGGER ALL;

INSERT INTO public.environment (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order", resource_id) VALUES (101, 'NORMAL', 1, 1694683927, 101, 1695110907, 'Test', 0, 'test') ON CONFLICT DO NOTHING;
INSERT INTO public.environment (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order", resource_id) VALUES (103, 'NORMAL', 101, 1695110903, 101, 1695110907, 'Staging', 1, 'staging') ON CONFLICT DO NOTHING;
INSERT INTO public.environment (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order", resource_id) VALUES (102, 'NORMAL', 1, 1694683927, 101, 1695110907, 'Prod', 2, 'prod') ON CONFLICT DO NOTHING;


ALTER TABLE public.environment ENABLE TRIGGER ALL;

--
-- Data for Name: instance; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.instance DISABLE TRIGGER ALL;

INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (105, 'NORMAL', 101, 1695025993, 101, 1695025993, 102, 'clickhouse', 'CLICKHOUSE', '', '', 'clickhouse', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (106, 'NORMAL', 101, 1695026057, 101, 1695026057, 102, 'mongodb', 'MONGODB', '', '', 'mongodb', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (107, 'NORMAL', 101, 1695026105, 101, 1695026105, 102, 'redis', 'REDIS', '', '', 'redis', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (108, 'NORMAL', 101, 1695026151, 101, 1695026151, 102, 'spanner', 'SPANNER', '', '', 'spanner', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (109, 'NORMAL', 101, 1695026169, 101, 1695026169, 102, 'oracle', 'ORACLE', '', '', 'oracle', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (103, 'NORMAL', 101, 1695025963, 101, 1695025963, 102, 'tidb', 'TIDB', '', '', 'tidb', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (104, 'NORMAL', 101, 1695025982, 101, 1695025982, 102, 'snowflake', 'SNOWFLAKE', '', '', 'snowflake', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (110, 'NORMAL', 101, 1695026328, 101, 1695026328, 102, 'oceanbase', 'OCEANBASE', '', '', 'oceanbase', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (111, 'NORMAL', 101, 1695026339, 101, 1695026339, 102, 'mariadb', 'MARIADB', '', '', 'mariadb', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (112, 'NORMAL', 101, 1695026350, 101, 1695026350, 102, 'mssql', 'MSSQL', '', '', 'mssql', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (113, 'NORMAL', 101, 1695026361, 101, 1695026361, 102, 'redshift', 'REDSHIFT', '', '', 'redshift', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (114, 'NORMAL', 101, 1695026377, 101, 1695026377, 102, 'risingwave', 'RISINGWAVE', '', '', 'risingwave', true, '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (102, 'NORMAL', 101, 1695025945, 1, 1695177903, 102, 'postgres', 'POSTGRES', '14.3', '', 'pg-prod', true, '{}', '{"lastSyncTime": "2023-09-20T02:45:02.695639Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (101, 'NORMAL', 101, 1695025927, 1, 1695177909, 102, 'mysql', 'MYSQL', '8.0.33', '', 'mysql-prod', true, '{}', '{"lastSyncTime": "2023-09-20T02:45:08.853376Z"}') ON CONFLICT DO NOTHING;


ALTER TABLE public.instance ENABLE TRIGGER ALL;

--
-- Data for Name: project; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.project DISABLE TRIGGER ALL;

INSERT INTO public.project (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, workflow_type, visibility, tenant_mode, db_name_template, schema_change_type, resource_id, data_classification_config_id, schema_version_type) VALUES (1, 'NORMAL', 1, 1695178575, 1, 1695178575, 'Default', 'DEFAULT', 'UI', 'PUBLIC', 'DISABLED', '', 'DDL', 'default', '', 'TIMESTAMP') ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, workflow_type, visibility, tenant_mode, db_name_template, schema_change_type, resource_id, data_classification_config_id, schema_version_type) VALUES (101, 'NORMAL', 101, 1694685057, 101, 1695178641, 'sakila', 'TEST', 'UI', 'PUBLIC', 'DISABLED', '', 'DDL', 'test', '', 'TIMESTAMP') ON CONFLICT DO NOTHING;


ALTER TABLE public.project ENABLE TRIGGER ALL;

--
-- Data for Name: db; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.db DISABLE TRIGGER ALL;

INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment_id, source_backup_id, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (101, 'NORMAL', 1, 1695178988, 1, 1695178988, 101, 101, 102, NULL, 'OK', 1695178988, '', 'sakila_prod', '{}', false, '', '{"labels": {"bb.environment": "prod"}, "lastSyncTime": "2023-09-20T03:03:08Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment_id, source_backup_id, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (102, 'NORMAL', 1, 1695179054, 1, 1695179054, 101, 101, 101, NULL, 'OK', 1695179054, '', 'sakila_test', '{}', false, '', '{"labels": {"bb.environment": "test"}, "lastSyncTime": "2023-09-20T03:04:14Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment_id, source_backup_id, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (103, 'NORMAL', 1, 1695179141, 1, 1695179141, 101, 101, 103, NULL, 'OK', 1695179141, '', 'sakila_staging', '{}', false, '', '{"labels": {"bb.environment": "staging"}, "lastSyncTime": "2023-09-20T03:05:41Z"}') ON CONFLICT DO NOTHING;


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



ALTER TABLE public.bookmark ENABLE TRIGGER ALL;

--
-- Data for Name: data_source; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.data_source DISABLE TRIGGER ALL;

INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (101, 'NORMAL', 101, 1695025927, 101, 1695025927, 101, 'adea9f0f-c5bc-489a-adab-ccb1dc46afad', 'ADMIN', 'root', '', '', '', '', '127.0.0.1', '3306', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (102, 'NORMAL', 101, 1695025945, 101, 1695025945, 102, '20662196-d819-48b3-87f2-d6cf3e4ef4f0', 'ADMIN', 'postgres', '', '', '', '', '127.0.0.1', '5432', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (103, 'NORMAL', 101, 1695025963, 101, 1695025963, 103, '071e570c-a84c-4165-ab15-8089c58fee3b', 'ADMIN', '', '', '', '', '', '127.0.0.1', '4000', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (104, 'NORMAL', 101, 1695025982, 101, 1695025982, 104, 'f5a8f7c4-da41-43f4-9441-3f8df0d8b841', 'ADMIN', '', '', '', '', '', 'demo@example.com', '443', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (105, 'NORMAL', 101, 1695025993, 101, 1695025993, 105, '88f8d0fd-7294-46e2-819d-9ceba22ceb79', 'ADMIN', '', '', '', '', '', '127.0.0.1', '9000', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (106, 'NORMAL', 101, 1695026057, 101, 1695026057, 106, 'ea1f1ab9-8462-4c46-8767-e8397c37c043', 'ADMIN', '', '', '', '', '', '127.0.0.1', '27017', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (107, 'NORMAL', 101, 1695026105, 101, 1695026105, 107, 'ea84c685-65eb-477b-b11b-40693fd0ba23', 'ADMIN', '', '', '', '', '', '127.0.0.1', '6379', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (108, 'NORMAL', 101, 1695026151, 101, 1695026151, 108, '3d79930b-b468-47ee-8f1c-a537e1670ba5', 'ADMIN', '', 'CxQHKDB9BD15dTw=', '', '', '', 'projects/example/instances/example', '3306', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (109, 'NORMAL', 101, 1695026169, 101, 1695026169, 109, '8cc4fa61-962a-43cb-a743-aa8f3804a692', 'ADMIN', '', '', '', '', '', '127.0.0.1', '1521', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (110, 'NORMAL', 101, 1695026328, 101, 1695026328, 110, '953f609c-1c40-4732-8a77-6e439069cdbd', 'ADMIN', '', '', '', '', '', '127.0.0.1', '2883', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (111, 'NORMAL', 101, 1695026339, 101, 1695026339, 111, 'c09e455b-be78-48bc-aae5-8739ac44b035', 'ADMIN', '', '', '', '', '', '127.0.0.1', '3306', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (112, 'NORMAL', 101, 1695026350, 101, 1695026350, 112, 'c69ae443-0684-4683-b0a9-cade8f446ba9', 'ADMIN', '', '', '', '', '', '127.0.0.1', '1433', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (113, 'NORMAL', 101, 1695026361, 101, 1695026361, 113, '9b9bb649-7a63-4d3e-a16b-8725dc32a15e', 'ADMIN', '', '', '', '', '', '127.0.0.1', '5439', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (114, 'NORMAL', 101, 1695026377, 101, 1695026377, 114, 'b449371a-3305-4418-8297-86682012f719', 'ADMIN', '', '', '', '', '', '127.0.0.1', '3306', '{}', '') ON CONFLICT DO NOTHING;


ALTER TABLE public.data_source ENABLE TRIGGER ALL;

--
-- Data for Name: db_group; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.db_group DISABLE TRIGGER ALL;



ALTER TABLE public.db_group ENABLE TRIGGER ALL;

--
-- Data for Name: db_label; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.db_label DISABLE TRIGGER ALL;



ALTER TABLE public.db_label ENABLE TRIGGER ALL;

--
-- Data for Name: db_schema; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.db_schema DISABLE TRIGGER ALL;

INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump) VALUES (101, 'NORMAL', 1, 1695178988, 1, 1695178988, 101, '{"name": "sakila_prod", "schemas": [{}], "collation": "utf8mb4_general_ci", "characterSet": "utf8mb4"}', 'SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump) VALUES (102, 'NORMAL', 1, 1695179054, 1, 1695179054, 102, '{"name": "sakila_test", "schemas": [{}], "collation": "utf8mb4_general_ci", "characterSet": "utf8mb4"}', 'SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump) VALUES (103, 'NORMAL', 1, 1695179141, 1, 1695179141, 103, '{"name": "sakila_staging", "schemas": [{}], "collation": "utf8mb4_general_ci", "characterSet": "utf8mb4"}', 'SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
') ON CONFLICT DO NOTHING;


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

INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (101, 'NORMAL', 101, 1695178711, 101, 1695178711, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (102, 'NORMAL', 101, 1695179030, 101, 1695179030, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (103, 'NORMAL', 101, 1695179078, 101, 1695179078, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;


ALTER TABLE public.pipeline ENABLE TRIGGER ALL;

--
-- Data for Name: plan; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.plan DISABLE TRIGGER ALL;

INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (101, 'NORMAL', 101, 1695178711, 101, 1695178711, 101, 101, '', '', '{"steps": [{"specs": [{"id": "494f897d-66e9-4562-8b97-5d213fdf6ef4", "createDatabaseConfig": {"labels": {"bb.environment": "prod"}, "target": "instances/mysql-prod", "database": "sakila_prod", "collation": "utf8mb4_general_ci", "environment": "environments/prod", "characterSet": "utf8mb4"}}]}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (102, 'NORMAL', 101, 1695179030, 101, 1695179030, 101, 102, '', '', '{"steps": [{"specs": [{"id": "11025977-45ca-4043-88fb-8002af053350", "createDatabaseConfig": {"labels": {"bb.environment": "test"}, "target": "instances/mysql-prod", "database": "sakila_test", "collation": "utf8mb4_general_ci", "environment": "environments/test", "characterSet": "utf8mb4"}}]}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (103, 'NORMAL', 101, 1695179078, 101, 1695179078, 101, 103, '', '', '{"steps": [{"specs": [{"id": "51e70d6c-d734-4ac5-bd74-16f01e8d6698", "createDatabaseConfig": {"labels": {"bb.environment": "staging"}, "target": "instances/mysql-prod", "database": "sakila_staging", "collation": "utf8mb4_general_ci", "environment": "environments/staging", "characterSet": "utf8mb4"}}]}]}') ON CONFLICT DO NOTHING;


ALTER TABLE public.plan ENABLE TRIGGER ALL;

--
-- Data for Name: issue; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.issue DISABLE TRIGGER ALL;

INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (101, 'NORMAL', 101, 1695178711, 1, 1695178988, 101, 101, 101, 'Create database ''sakila_prod''', 'DONE', 'bb.issue.database.general', '', 101, false, '{"approval": {"approvers": [{"status": "APPROVED", "principalId": 102}], "approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}]}, "title": "DBA", "creatorId": 1, "description": "系统定义的流程。只需要 DBA 审批"}], "approvalFindingDone": true}}', '''create'':1 ''database'':2 ''prod'':4 ''sakila'':3') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (102, 'NORMAL', 101, 1695179030, 1, 1695179054, 101, 102, 102, 'Create database ''sakila_test''', 'DONE', 'bb.issue.database.general', '', 101, false, '{"approval": {"approvers": [{"status": "APPROVED", "principalId": 102}], "approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}]}, "title": "DBA", "creatorId": 1, "description": "系统定义的流程。只需要 DBA 审批"}], "approvalFindingDone": true}}', '''create'':1 ''database'':2 ''sakila'':3 ''test'':4') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (103, 'NORMAL', 101, 1695179078, 1, 1695179141, 101, 103, 103, 'Create database ''sakila_staging''', 'DONE', 'bb.issue.database.general', '', 101, false, '{"approval": {"approvers": [{"status": "APPROVED", "principalId": 102}], "approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}]}, "title": "DBA", "creatorId": 1, "description": "系统定义的流程。只需要 DBA 审批"}], "approvalFindingDone": true}}', '''create'':1 ''database'':2 ''sakila'':3 ''staging'':4') ON CONFLICT DO NOTHING;


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

INSERT INTO public.inbox (id, receiver_id, activity_id, status) VALUES (101, 101, 109, 'UNREAD') ON CONFLICT DO NOTHING;
INSERT INTO public.inbox (id, receiver_id, activity_id, status) VALUES (102, 101, 115, 'UNREAD') ON CONFLICT DO NOTHING;
INSERT INTO public.inbox (id, receiver_id, activity_id, status) VALUES (103, 101, 116, 'UNREAD') ON CONFLICT DO NOTHING;
INSERT INTO public.inbox (id, receiver_id, activity_id, status) VALUES (104, 101, 122, 'UNREAD') ON CONFLICT DO NOTHING;
INSERT INTO public.inbox (id, receiver_id, activity_id, status) VALUES (105, 101, 123, 'UNREAD') ON CONFLICT DO NOTHING;
INSERT INTO public.inbox (id, receiver_id, activity_id, status) VALUES (106, 101, 129, 'UNREAD') ON CONFLICT DO NOTHING;


ALTER TABLE public.inbox ENABLE TRIGGER ALL;

--
-- Data for Name: instance_change_history; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.instance_change_history DISABLE TRIGGER ALL;

INSERT INTO public.instance_change_history (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, issue_id, release_version, sequence, source, type, status, version, description, statement, sheet_id, schema, schema_prev, execution_duration_ns, payload) VALUES (101, 'NORMAL', 1, 1695178575, 1, 1695178575, NULL, NULL, NULL, 'development', 1, 'LIBRARY', 'MIGRATE', 'DONE', '0002.0008.0004-20230920105614', 'Initial migration version 2.8.4 server version development with file migration/prod/LATEST.sql.', '-- Type
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
    data_classification_config_id TEXT NOT NULL DEFAULT ''''
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
    raw_dump TEXT NOT NULL DEFAULT ''''
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

-- Label
-- label_key stores available label keys at workspace level.
CREATE TABLE label_key (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    key TEXT NOT NULL
);

-- key''s are unique within the label_key table.
CREATE UNIQUE INDEX idx_label_key_unique_key ON label_key(key);

ALTER SEQUENCE label_key_id_seq RESTART WITH 101;

CREATE TRIGGER update_label_key_updated_ts
BEFORE
UPDATE
    ON label_key FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- label_value stores available label key values at workspace level.
CREATE TABLE label_value (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    key TEXT NOT NULL REFERENCES label_key(key),
    value TEXT NOT NULL
);

-- key/value''s are unique within the label_value table.
CREATE UNIQUE INDEX idx_label_value_unique_key_value ON label_value(key, value);

ALTER SEQUENCE label_value_id_seq RESTART WITH 101;

CREATE TRIGGER update_label_value_updated_ts
BEFORE
UPDATE
    ON label_value FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- db_label stores labels associated with databases.
CREATE TABLE db_label (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    key TEXT NOT NULL,
    value TEXT NOT NULL
);

-- database_id/key''s are unique within the db_label table.
CREATE UNIQUE INDEX idx_db_label_unique_database_id_key ON db_label(database_id, key);

ALTER SEQUENCE db_label_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_label_updated_ts
BEFORE
UPDATE
    ON db_label FOR EACH ROW
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

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        resource_type,
        resource_id,
        inherit_from_parent,
        type,
        payload
    )
VALUES
    (
        101,
        1,
        1,
        ''ENVIRONMENT'',
        101,
        TRUE,
        ''bb.policy.pipeline-approval'',
        ''{"value":"MANUAL_APPROVAL_NEVER"}''
    );

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        resource_type,
        resource_id,
        inherit_from_parent,
        type,
        payload
    )
VALUES
    (
        102,
        1,
        1,
        ''ENVIRONMENT'',
        102,
        TRUE,
        ''bb.policy.pipeline-approval'',
        ''{"value":"MANUAL_APPROVAL_NEVER"}''
    );

ALTER SEQUENCE policy_id_seq RESTART WITH 103;
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
    data_classification_config_id TEXT NOT NULL DEFAULT ''''
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
    raw_dump TEXT NOT NULL DEFAULT ''''
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

-- Label
-- label_key stores available label keys at workspace level.
CREATE TABLE label_key (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    key TEXT NOT NULL
);

-- key''s are unique within the label_key table.
CREATE UNIQUE INDEX idx_label_key_unique_key ON label_key(key);

ALTER SEQUENCE label_key_id_seq RESTART WITH 101;

CREATE TRIGGER update_label_key_updated_ts
BEFORE
UPDATE
    ON label_key FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- label_value stores available label key values at workspace level.
CREATE TABLE label_value (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    key TEXT NOT NULL REFERENCES label_key(key),
    value TEXT NOT NULL
);

-- key/value''s are unique within the label_value table.
CREATE UNIQUE INDEX idx_label_value_unique_key_value ON label_value(key, value);

ALTER SEQUENCE label_value_id_seq RESTART WITH 101;

CREATE TRIGGER update_label_value_updated_ts
BEFORE
UPDATE
    ON label_value FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- db_label stores labels associated with databases.
CREATE TABLE db_label (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT ''NORMAL'',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    key TEXT NOT NULL,
    value TEXT NOT NULL
);

-- database_id/key''s are unique within the db_label table.
CREATE UNIQUE INDEX idx_db_label_unique_database_id_key ON db_label(database_id, key);

ALTER SEQUENCE db_label_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_label_updated_ts
BEFORE
UPDATE
    ON db_label FOR EACH ROW
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

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        resource_type,
        resource_id,
        inherit_from_parent,
        type,
        payload
    )
VALUES
    (
        101,
        1,
        1,
        ''ENVIRONMENT'',
        101,
        TRUE,
        ''bb.policy.pipeline-approval'',
        ''{"value":"MANUAL_APPROVAL_NEVER"}''
    );

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        resource_type,
        resource_id,
        inherit_from_parent,
        type,
        payload
    )
VALUES
    (
        102,
        1,
        1,
        ''ENVIRONMENT'',
        102,
        TRUE,
        ''bb.policy.pipeline-approval'',
        ''{"value":"MANUAL_APPROVAL_NEVER"}''
    );

ALTER SEQUENCE policy_id_seq RESTART WITH 103;
', '', 0, '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, issue_id, release_version, sequence, source, type, status, version, description, statement, sheet_id, schema, schema_prev, execution_duration_ns, payload) VALUES (102, 'NORMAL', 1, 1695178575, 1, 1695178575, NULL, NULL, NULL, 'development', 2, 'LIBRARY', 'MIGRATE', 'DONE', '0002.0008.0004-dev20220408000000', 'Migrate version 20220408000000 server version development with files migration/dev/20220408000000##schema_version_type.sql.', 'ALTER TABLE project ADD schema_version_type TEXT NOT NULL CHECK (schema_version_type IN (''TIMESTAMP'', ''SEMANTIC'')) DEFAULT ''TIMESTAMP'';
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

CREATE TABLE public.db_label (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    key text NOT NULL,
    value text NOT NULL
);

CREATE SEQUENCE public.db_label_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_label_id_seq OWNED BY public.db_label.id;

CREATE TABLE public.db_schema (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL,
    raw_dump text DEFAULT ''''::text NOT NULL
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

CREATE TABLE public.label_key (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    key text NOT NULL
);

CREATE SEQUENCE public.label_key_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.label_key_id_seq OWNED BY public.label_key.id;

CREATE TABLE public.label_value (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    key text NOT NULL,
    value text NOT NULL
);

CREATE SEQUENCE public.label_value_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.label_value_id_seq OWNED BY public.label_value.id;

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

ALTER TABLE ONLY public.data_source ALTER COLUMN id SET DEFAULT nextval(''public.data_source_id_seq''::regclass);

ALTER TABLE ONLY public.db ALTER COLUMN id SET DEFAULT nextval(''public.db_id_seq''::regclass);

ALTER TABLE ONLY public.db_group ALTER COLUMN id SET DEFAULT nextval(''public.db_group_id_seq''::regclass);

ALTER TABLE ONLY public.db_label ALTER COLUMN id SET DEFAULT nextval(''public.db_label_id_seq''::regclass);

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

ALTER TABLE ONLY public.label_key ALTER COLUMN id SET DEFAULT nextval(''public.label_key_id_seq''::regclass);

ALTER TABLE ONLY public.label_value ALTER COLUMN id SET DEFAULT nextval(''public.label_value_id_seq''::regclass);

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

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_pkey PRIMARY KEY (id);

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

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_pkey PRIMARY KEY (id);

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

CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON public.data_source USING btree (instance_id, name);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON public.db_group USING btree (project_id, placeholder);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON public.db_group USING btree (project_id, resource_id);

CREATE INDEX idx_db_instance_id ON public.db USING btree (instance_id);

CREATE UNIQUE INDEX idx_db_label_unique_database_id_key ON public.db_label USING btree (database_id, key);

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

CREATE UNIQUE INDEX idx_label_key_unique_key ON public.label_key USING btree (key);

CREATE UNIQUE INDEX idx_label_value_unique_key_value ON public.label_value USING btree (key, value);

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

CREATE TRIGGER update_data_source_updated_ts BEFORE UPDATE ON public.data_source FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_group_updated_ts BEFORE UPDATE ON public.db_group FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_label_updated_ts BEFORE UPDATE ON public.db_label FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

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

CREATE TRIGGER update_label_key_updated_ts BEFORE UPDATE ON public.label_key FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_label_value_updated_ts BEFORE UPDATE ON public.label_value FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

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

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

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

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_key_fkey FOREIGN KEY (key) REFERENCES public.label_key(key);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

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

CREATE TABLE public.db_label (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    key text NOT NULL,
    value text NOT NULL
);

CREATE SEQUENCE public.db_label_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_label_id_seq OWNED BY public.db_label.id;

CREATE TABLE public.db_schema (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL,
    raw_dump text DEFAULT ''''::text NOT NULL
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

CREATE TABLE public.label_key (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    key text NOT NULL
);

CREATE SEQUENCE public.label_key_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.label_key_id_seq OWNED BY public.label_key.id;

CREATE TABLE public.label_value (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    key text NOT NULL,
    value text NOT NULL
);

CREATE SEQUENCE public.label_value_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.label_value_id_seq OWNED BY public.label_value.id;

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

ALTER TABLE ONLY public.data_source ALTER COLUMN id SET DEFAULT nextval(''public.data_source_id_seq''::regclass);

ALTER TABLE ONLY public.db ALTER COLUMN id SET DEFAULT nextval(''public.db_id_seq''::regclass);

ALTER TABLE ONLY public.db_group ALTER COLUMN id SET DEFAULT nextval(''public.db_group_id_seq''::regclass);

ALTER TABLE ONLY public.db_label ALTER COLUMN id SET DEFAULT nextval(''public.db_label_id_seq''::regclass);

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

ALTER TABLE ONLY public.label_key ALTER COLUMN id SET DEFAULT nextval(''public.label_key_id_seq''::regclass);

ALTER TABLE ONLY public.label_value ALTER COLUMN id SET DEFAULT nextval(''public.label_value_id_seq''::regclass);

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

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_pkey PRIMARY KEY (id);

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

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_pkey PRIMARY KEY (id);

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

CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON public.data_source USING btree (instance_id, name);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON public.db_group USING btree (project_id, placeholder);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON public.db_group USING btree (project_id, resource_id);

CREATE INDEX idx_db_instance_id ON public.db USING btree (instance_id);

CREATE UNIQUE INDEX idx_db_label_unique_database_id_key ON public.db_label USING btree (database_id, key);

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

CREATE UNIQUE INDEX idx_label_key_unique_key ON public.label_key USING btree (key);

CREATE UNIQUE INDEX idx_label_value_unique_key_value ON public.label_value USING btree (key, value);

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

CREATE TRIGGER update_data_source_updated_ts BEFORE UPDATE ON public.data_source FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_group_updated_ts BEFORE UPDATE ON public.db_group FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_label_updated_ts BEFORE UPDATE ON public.db_label FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

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

CREATE TRIGGER update_label_key_updated_ts BEFORE UPDATE ON public.label_key FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_label_value_updated_ts BEFORE UPDATE ON public.label_value FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

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

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

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

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_key_fkey FOREIGN KEY (key) REFERENCES public.label_key(key);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

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

', 88775000, '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, issue_id, release_version, sequence, source, type, status, version, description, statement, sheet_id, schema, schema_prev, execution_duration_ns, payload) VALUES (103, 'NORMAL', 1, 1695110681, 1, 1695110681, NULL, NULL, NULL, 'development', 3, 'LIBRARY', 'MIGRATE', 'DONE', '0002.0008.0004-20230919160440', 'Migrate version 2.8.4 server version development with files migration/prod/2.8/0004##issue_type.sql.', 'ALTER TABLE issue DISABLE TRIGGER update_issue_updated_ts;

UPDATE issue
SET type = ''bb.issue.database.general''
WHERE type IN (''bb.issue.database.create'', ''bb.issue.database.schema.update'', ''bb.issue.database.schema.update.ghost'', ''bb.issue.database.data.update'', ''bb.issue.database.restore.pitr'');

ALTER TABLE issue ENABLE TRIGGER update_issue_updated_ts;
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

CREATE TABLE public.db_label (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    key text NOT NULL,
    value text NOT NULL
);

CREATE SEQUENCE public.db_label_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_label_id_seq OWNED BY public.db_label.id;

CREATE TABLE public.db_schema (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL,
    raw_dump text DEFAULT ''''::text NOT NULL
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

CREATE TABLE public.label_key (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    key text NOT NULL
);

CREATE SEQUENCE public.label_key_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.label_key_id_seq OWNED BY public.label_key.id;

CREATE TABLE public.label_value (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    key text NOT NULL,
    value text NOT NULL
);

CREATE SEQUENCE public.label_value_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.label_value_id_seq OWNED BY public.label_value.id;

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

ALTER TABLE ONLY public.data_source ALTER COLUMN id SET DEFAULT nextval(''public.data_source_id_seq''::regclass);

ALTER TABLE ONLY public.db ALTER COLUMN id SET DEFAULT nextval(''public.db_id_seq''::regclass);

ALTER TABLE ONLY public.db_group ALTER COLUMN id SET DEFAULT nextval(''public.db_group_id_seq''::regclass);

ALTER TABLE ONLY public.db_label ALTER COLUMN id SET DEFAULT nextval(''public.db_label_id_seq''::regclass);

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

ALTER TABLE ONLY public.label_key ALTER COLUMN id SET DEFAULT nextval(''public.label_key_id_seq''::regclass);

ALTER TABLE ONLY public.label_value ALTER COLUMN id SET DEFAULT nextval(''public.label_value_id_seq''::regclass);

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

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_pkey PRIMARY KEY (id);

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

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_pkey PRIMARY KEY (id);

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

CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON public.data_source USING btree (instance_id, name);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON public.db_group USING btree (project_id, placeholder);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON public.db_group USING btree (project_id, resource_id);

CREATE INDEX idx_db_instance_id ON public.db USING btree (instance_id);

CREATE UNIQUE INDEX idx_db_label_unique_database_id_key ON public.db_label USING btree (database_id, key);

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

CREATE UNIQUE INDEX idx_label_key_unique_key ON public.label_key USING btree (key);

CREATE UNIQUE INDEX idx_label_value_unique_key_value ON public.label_value USING btree (key, value);

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

CREATE TRIGGER update_data_source_updated_ts BEFORE UPDATE ON public.data_source FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_group_updated_ts BEFORE UPDATE ON public.db_group FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_label_updated_ts BEFORE UPDATE ON public.db_label FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

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

CREATE TRIGGER update_label_key_updated_ts BEFORE UPDATE ON public.label_key FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_label_value_updated_ts BEFORE UPDATE ON public.label_value FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

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

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

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

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_key_fkey FOREIGN KEY (key) REFERENCES public.label_key(key);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

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

CREATE TABLE public.db_label (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    key text NOT NULL,
    value text NOT NULL
);

CREATE SEQUENCE public.db_label_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.db_label_id_seq OWNED BY public.db_label.id;

CREATE TABLE public.db_schema (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
    metadata jsonb DEFAULT ''{}''::jsonb NOT NULL,
    raw_dump text DEFAULT ''''::text NOT NULL
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

CREATE TABLE public.label_key (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    key text NOT NULL
);

CREATE SEQUENCE public.label_key_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.label_key_id_seq OWNED BY public.label_key.id;

CREATE TABLE public.label_value (
    id integer NOT NULL,
    row_status public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    key text NOT NULL,
    value text NOT NULL
);

CREATE SEQUENCE public.label_value_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.label_value_id_seq OWNED BY public.label_value.id;

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

ALTER TABLE ONLY public.data_source ALTER COLUMN id SET DEFAULT nextval(''public.data_source_id_seq''::regclass);

ALTER TABLE ONLY public.db ALTER COLUMN id SET DEFAULT nextval(''public.db_id_seq''::regclass);

ALTER TABLE ONLY public.db_group ALTER COLUMN id SET DEFAULT nextval(''public.db_group_id_seq''::regclass);

ALTER TABLE ONLY public.db_label ALTER COLUMN id SET DEFAULT nextval(''public.db_label_id_seq''::regclass);

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

ALTER TABLE ONLY public.label_key ALTER COLUMN id SET DEFAULT nextval(''public.label_key_id_seq''::regclass);

ALTER TABLE ONLY public.label_value ALTER COLUMN id SET DEFAULT nextval(''public.label_value_id_seq''::regclass);

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

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_pkey PRIMARY KEY (id);

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

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_pkey PRIMARY KEY (id);

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

CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON public.data_source USING btree (instance_id, name);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON public.db_group USING btree (project_id, placeholder);

CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON public.db_group USING btree (project_id, resource_id);

CREATE INDEX idx_db_instance_id ON public.db USING btree (instance_id);

CREATE UNIQUE INDEX idx_db_label_unique_database_id_key ON public.db_label USING btree (database_id, key);

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

CREATE UNIQUE INDEX idx_label_key_unique_key ON public.label_key USING btree (key);

CREATE UNIQUE INDEX idx_label_value_unique_key_value ON public.label_value USING btree (key, value);

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

CREATE TRIGGER update_data_source_updated_ts BEFORE UPDATE ON public.data_source FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_group_updated_ts BEFORE UPDATE ON public.db_group FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_db_label_updated_ts BEFORE UPDATE ON public.db_label FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

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

CREATE TRIGGER update_label_key_updated_ts BEFORE UPDATE ON public.label_key FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

CREATE TRIGGER update_label_value_updated_ts BEFORE UPDATE ON public.label_value FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();

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

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);

ALTER TABLE ONLY public.db_label
    ADD CONSTRAINT db_label_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

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

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_key
    ADD CONSTRAINT label_key_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_key_fkey FOREIGN KEY (key) REFERENCES public.label_key(key);

ALTER TABLE ONLY public.label_value
    ADD CONSTRAINT label_value_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);

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

', 62228000, '{}') ON CONFLICT DO NOTHING;


ALTER TABLE public.instance_change_history ENABLE TRIGGER ALL;

--
-- Data for Name: instance_user; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.instance_user DISABLE TRIGGER ALL;

INSERT INTO public.instance_user (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, "grant") VALUES (101, 'NORMAL', 1, 1695025927, 1, 1695025927, 101, '''root''@''localhost''', 'GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, RELOAD, SHUTDOWN, PROCESS, FILE, REFERENCES, INDEX, ALTER, SHOW DATABASES, SUPER, CREATE TEMPORARY TABLES, LOCK TABLES, EXECUTE, REPLICATION SLAVE, REPLICATION CLIENT, CREATE VIEW, SHOW VIEW, CREATE ROUTINE, ALTER ROUTINE, CREATE USER, EVENT, TRIGGER, CREATE TABLESPACE, CREATE ROLE, DROP ROLE ON *.* TO `root`@`localhost` WITH GRANT OPTION
GRANT APPLICATION_PASSWORD_ADMIN,AUDIT_ADMIN,AUTHENTICATION_POLICY_ADMIN,BACKUP_ADMIN,BINLOG_ADMIN,BINLOG_ENCRYPTION_ADMIN,CLONE_ADMIN,CONNECTION_ADMIN,ENCRYPTION_KEY_ADMIN,FLUSH_OPTIMIZER_COSTS,FLUSH_STATUS,FLUSH_TABLES,FLUSH_USER_RESOURCES,GROUP_REPLICATION_ADMIN,GROUP_REPLICATION_STREAM,INNODB_REDO_LOG_ARCHIVE,INNODB_REDO_LOG_ENABLE,PASSWORDLESS_USER_ADMIN,PERSIST_RO_VARIABLES_ADMIN,REPLICATION_APPLIER,REPLICATION_SLAVE_ADMIN,RESOURCE_GROUP_ADMIN,RESOURCE_GROUP_USER,ROLE_ADMIN,SERVICE_CONNECTION_ADMIN,SESSION_VARIABLES_ADMIN,SET_USER_ID,SHOW_ROUTINE,SYSTEM_USER,SYSTEM_VARIABLES_ADMIN,TABLE_ENCRYPTION_ADMIN,XA_RECOVER_ADMIN ON *.* TO `root`@`localhost` WITH GRANT OPTION
GRANT PROXY ON ``@`` TO `root`@`localhost` WITH GRANT OPTION') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_user (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, "grant") VALUES (102, 'NORMAL', 1, 1695025945, 1, 1695025945, 102, 'postgres', 'Superuser, Create role, Create DB, Replication, Bypass RLS+') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_user (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, "grant") VALUES (103, 'NORMAL', 1, 1695025945, 1, 1695025945, 102, 'david', '') ON CONFLICT DO NOTHING;


ALTER TABLE public.instance_user ENABLE TRIGGER ALL;

--
-- Data for Name: issue_subscriber; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.issue_subscriber DISABLE TRIGGER ALL;



ALTER TABLE public.issue_subscriber ENABLE TRIGGER ALL;

--
-- Data for Name: label_key; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.label_key DISABLE TRIGGER ALL;



ALTER TABLE public.label_key ENABLE TRIGGER ALL;

--
-- Data for Name: label_value; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.label_value DISABLE TRIGGER ALL;



ALTER TABLE public.label_value ENABLE TRIGGER ALL;

--
-- Data for Name: member; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.member DISABLE TRIGGER ALL;

INSERT INTO public.member (id, row_status, creator_id, created_ts, updater_id, updated_ts, status, role, principal_id) VALUES (101, 'NORMAL', 1, 1694684977, 101, 1695112774, 'ACTIVE', 'OWNER', 101) ON CONFLICT DO NOTHING;
INSERT INTO public.member (id, row_status, creator_id, created_ts, updater_id, updated_ts, status, role, principal_id) VALUES (102, 'NORMAL', 1, 1695112807, 101, 1695112895, 'ACTIVE', 'DBA', 102) ON CONFLICT DO NOTHING;
INSERT INTO public.member (id, row_status, creator_id, created_ts, updater_id, updated_ts, status, role, principal_id) VALUES (103, 'NORMAL', 1, 1695112807, 101, 1695112903, 'ACTIVE', 'DEVELOPER', 103) ON CONFLICT DO NOTHING;
INSERT INTO public.member (id, row_status, creator_id, created_ts, updater_id, updated_ts, status, role, principal_id) VALUES (104, 'NORMAL', 1, 1695112807, 101, 1695112915, 'ACTIVE', 'DEVELOPER', 104) ON CONFLICT DO NOTHING;


ALTER TABLE public.member ENABLE TRIGGER ALL;

--
-- Data for Name: plan_check_run; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.plan_check_run DISABLE TRIGGER ALL;



ALTER TABLE public.plan_check_run ENABLE TRIGGER ALL;

--
-- Data for Name: policy; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.policy DISABLE TRIGGER ALL;

INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (101, 'NORMAL', 1, 1694683927, 1, 1694683927, 'bb.policy.pipeline-approval', '{"value": "MANUAL_APPROVAL_NEVER"}', 'ENVIRONMENT', 101, true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (103, 'NORMAL', 101, 1695110903, 101, 1695110903, 'bb.policy.environment-tier', '{"environmentTier": "UNPROTECTED"}', 'ENVIRONMENT', 103, true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (104, 'NORMAL', 101, 1695110903, 101, 1695110903, 'bb.policy.backup-plan', '{"schedule": "UNSET", "retentionPeriodTs": 0}', 'ENVIRONMENT', 103, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (105, 'NORMAL', 101, 1695110903, 101, 1695110903, 'bb.policy.pipeline-approval', '{"value": "MANUAL_APPROVAL_ALWAYS", "assigneeGroupList": [{"value": "PROJECT_OWNER", "issueType": "bb.issue.database.schema.update"}, {"value": "PROJECT_OWNER", "issueType": "bb.issue.database.data.update"}, {"value": "PROJECT_OWNER", "issueType": "bb.issue.database.schema.update.ghost"}, {"value": "PROJECT_OWNER", "issueType": "bb.issue.database.general"}]}', 'ENVIRONMENT', 103, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (102, 'NORMAL', 1, 1694683927, 101, 1695177922, 'bb.policy.pipeline-approval', '{"value": "MANUAL_APPROVAL_ALWAYS", "assigneeGroupList": [{"value": "PROJECT_OWNER", "issueType": "bb.issue.database.schema.update"}, {"value": "PROJECT_OWNER", "issueType": "bb.issue.database.data.update"}, {"value": "PROJECT_OWNER", "issueType": "bb.issue.database.schema.update.ghost"}, {"value": "PROJECT_OWNER", "issueType": "bb.issue.database.general"}]}', 'ENVIRONMENT', 102, true) ON CONFLICT DO NOTHING;


ALTER TABLE public.policy ENABLE TRIGGER ALL;

--
-- Data for Name: project_member; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.project_member DISABLE TRIGGER ALL;

INSERT INTO public.project_member (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id, condition) VALUES (102, 'NORMAL', 101, 1695112950, 101, 1695112950, 101, 'DEVELOPER', 103, '{"title": "Developer"}') ON CONFLICT DO NOTHING;
INSERT INTO public.project_member (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id, condition) VALUES (103, 'NORMAL', 101, 1695112950, 101, 1695112950, 101, 'OWNER', 102, '{"title": "Owner"}') ON CONFLICT DO NOTHING;
INSERT INTO public.project_member (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id, condition) VALUES (104, 'NORMAL', 101, 1695113006, 101, 1695113006, 101, 'DEVELOPER', 101, '{"title": "Developer"}') ON CONFLICT DO NOTHING;


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

INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (104, 'NORMAL', 1, 1694683928, 1, 1695111120, 'bb.enterprise.license', 'eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA', 'Enterprise license') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (102, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.auth.secret', '0DYdOCjqcJSJ9KAfUlCOKsbS743PjSDi', 'Random string used to sign the JWT auth token.') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (101, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.branding.logo', '', 'The branding slogo image in base64 string format.') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (103, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.workspace.id', 'be28420e-5db3-4dc6-80ba-b822be20d36a', 'The workspace identifier') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (105, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.app.im', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (106, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.workspace.watermark', '0', 'Display watermark') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (107, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.plugin.openai.key', '', 'API key to request OpenAI (ChatGPT)') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (108, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.plugin.openai.endpoint', '', 'API Endpoint for OpenAI') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (109, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.workspace.approval.external', '{}', 'The external approval setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (110, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.workspace.schema-template', '{}', 'The schema template setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (111, 'NORMAL', 1, 1694683928, 1, 1694683928, 'bb.workspace.data-classification', '{}', 'The data classification setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (112, 'NORMAL', 1, 1694683928, 101, 1695177966, 'bb.workspace.approval', '{"rules":[{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]}]},"title":"项目所有者 -> DBA","description":"系统定义的流程。先由项目所有者审批，再由 DBA 审批。","creatorId":1},"condition":{"expression":"source == 1 && level == 0"}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]}]},"title":"项目所有者","description":"系统定义的流程。只需要项目所有者审批。","creatorId":1},"condition":{"expression":"source == 2 && level == 0 || source == 4 && level == 0 || source == 5 && level == 0"}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]}]},"title":"DBA","description":"系统定义的流程。只需要 DBA 审批","creatorId":1},"condition":{"expression":"source == 3 && level == 0"}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_OWNER"}]}]},"title":"工作空间所有者","description":"系统定义的流程。只需要管理员审批","creatorId":1},"condition":{}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_OWNER"}]}]},"title":"项目所有者 -> DBA -> 工作空间所有者","description":"系统定义的流程。先由项目所有者审批，再由 DBA 审批，最后由管理员审批。","creatorId":1},"condition":{}}]}', 'The workspace approval setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (113, 'NORMAL', 1, 1694683928, 1, 1695178575, 'bb.workspace.profile', '{}', '') ON CONFLICT DO NOTHING;


ALTER TABLE public.setting ENABLE TRIGGER ALL;

--
-- Data for Name: sheet; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.sheet DISABLE TRIGGER ALL;

INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload) VALUES (102, 'NORMAL', 1, 1695178711, 1, 1695178711, 101, NULL, 'Sheet for creating database sakila_prod', 'CREATE DATABASE `sakila_prod` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;', 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload) VALUES (101, 'NORMAL', 1, 1695178711, 1, 1695178988, 101, NULL, 'Sheet for creating database sakila_prod', 'CREATE DATABASE `sakila_prod` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;', 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload) VALUES (104, 'NORMAL', 1, 1695179030, 1, 1695179030, 101, NULL, 'Sheet for creating database sakila_test', 'CREATE DATABASE `sakila_test` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;', 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload) VALUES (103, 'NORMAL', 1, 1695179030, 1, 1695179054, 101, NULL, 'Sheet for creating database sakila_test', 'CREATE DATABASE `sakila_test` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;', 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload) VALUES (106, 'NORMAL', 1, 1695179079, 1, 1695179079, 101, NULL, 'Sheet for creating database sakila_staging', 'CREATE DATABASE `sakila_staging` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;', 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload) VALUES (105, 'NORMAL', 1, 1695179078, 1, 1695179141, 101, NULL, 'Sheet for creating database sakila_staging', 'CREATE DATABASE `sakila_staging` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;', 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}') ON CONFLICT DO NOTHING;


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

INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, name) VALUES (101, 'NORMAL', 101, 1695178711, 101, 1695178711, 101, 102, 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, name) VALUES (102, 'NORMAL', 101, 1695179030, 101, 1695179030, 102, 102, 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, name) VALUES (103, 'NORMAL', 101, 1695179078, 101, 1695179078, 103, 102, 'Prod Stage') ON CONFLICT DO NOTHING;


ALTER TABLE public.stage ENABLE TRIGGER ALL;

--
-- Data for Name: task; Type: TABLE DATA; Schema: public; Owner: bbdev
--

ALTER TABLE public.task DISABLE TRIGGER ALL;

INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (101, 'NORMAL', 101, 1695178711, 1, 1695178988, 101, 101, 101, 101, 'Create database sakila_prod', 'PENDING_APPROVAL', 'bb.task.database.create', '{"labels": "[{\"key\":\"bb.environment\",\"value\":\"prod\"}]", "specId": "494f897d-66e9-4562-8b97-5d213fdf6ef4", "sheetId": 101, "character": "utf8mb4", "collation": "utf8mb4_general_ci", "projectId": 101, "databaseName": "sakila_prod", "environmentId": "prod"}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (102, 'NORMAL', 101, 1695179030, 1, 1695179054, 102, 102, 101, 102, 'Create database sakila_test', 'PENDING_APPROVAL', 'bb.task.database.create', '{"labels": "[{\"key\":\"bb.environment\",\"value\":\"test\"}]", "specId": "11025977-45ca-4043-88fb-8002af053350", "sheetId": 103, "character": "utf8mb4", "collation": "utf8mb4_general_ci", "projectId": 101, "databaseName": "sakila_test", "environmentId": "test"}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (103, 'NORMAL', 101, 1695179078, 1, 1695179141, 103, 103, 101, 103, 'Create database sakila_staging', 'PENDING_APPROVAL', 'bb.task.database.create', '{"labels": "[{\"key\":\"bb.environment\",\"value\":\"staging\"}]", "specId": "51e70d6c-d734-4ac5-bd74-16f01e8d6698", "sheetId": 105, "character": "utf8mb4", "collation": "utf8mb4_general_ci", "projectId": 101, "databaseName": "sakila_staging", "environmentId": "staging"}', 0) ON CONFLICT DO NOTHING;


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

INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, attempt, name, status, code, result) VALUES (101, 101, 1695178988, 1, 1695178988, 101, 0, 'Create database sakila_prod 1695178988', 'DONE', 0, '{"detail": "Created database \"sakila_prod\""}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, attempt, name, status, code, result) VALUES (102, 101, 1695179054, 1, 1695179054, 102, 0, 'Create database sakila_test 1695179054', 'DONE', 0, '{"detail": "Created database \"sakila_test\""}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, attempt, name, status, code, result) VALUES (103, 101, 1695179141, 1, 1695179141, 103, 0, 'Create database sakila_staging 1695179141', 'DONE', 0, '{"detail": "Created database \"sakila_staging\""}') ON CONFLICT DO NOTHING;


ALTER TABLE public.task_run ENABLE TRIGGER ALL;

--
-- Name: activity_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.activity_id_seq', 129, true);


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

SELECT pg_catalog.setval('public.bookmark_id_seq', 101, false);


--
-- Name: data_source_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.data_source_id_seq', 114, true);


--
-- Name: db_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.db_group_id_seq', 101, false);


--
-- Name: db_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.db_id_seq', 103, true);


--
-- Name: db_label_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.db_label_id_seq', 101, false);


--
-- Name: db_schema_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.db_schema_id_seq', 103, true);


--
-- Name: deployment_config_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.deployment_config_id_seq', 101, false);


--
-- Name: environment_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.environment_id_seq', 103, true);


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

SELECT pg_catalog.setval('public.inbox_id_seq', 106, true);


--
-- Name: instance_change_history_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.instance_change_history_id_seq', 102, true);


--
-- Name: instance_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.instance_id_seq', 114, true);


--
-- Name: instance_user_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.instance_user_id_seq', 103, true);


--
-- Name: issue_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.issue_id_seq', 103, true);


--
-- Name: label_key_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.label_key_id_seq', 101, false);


--
-- Name: label_value_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.label_value_id_seq', 101, false);


--
-- Name: member_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.member_id_seq', 104, true);


--
-- Name: pipeline_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.pipeline_id_seq', 103, true);


--
-- Name: plan_check_run_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.plan_check_run_id_seq', 101, false);


--
-- Name: plan_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.plan_id_seq', 103, true);


--
-- Name: policy_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.policy_id_seq', 105, true);


--
-- Name: principal_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.principal_id_seq', 104, true);


--
-- Name: project_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.project_id_seq', 101, true);


--
-- Name: project_member_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.project_member_id_seq', 104, true);


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

SELECT pg_catalog.setval('public.setting_id_seq', 115, true);


--
-- Name: sheet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.sheet_id_seq', 106, true);


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

SELECT pg_catalog.setval('public.stage_id_seq', 103, true);


--
-- Name: task_dag_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.task_dag_id_seq', 101, false);


--
-- Name: task_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.task_id_seq', 103, true);


--
-- Name: task_run_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.task_run_id_seq', 103, true);


--
-- Name: vcs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: bbdev
--

SELECT pg_catalog.setval('public.vcs_id_seq', 101, false);


--
-- PostgreSQL database dump complete
--

