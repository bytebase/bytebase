--
-- PostgreSQL database dump
--

-- Dumped from database version 16.2
-- Dumped by pg_dump version 16.8 (Homebrew)

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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: anomaly; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.anomaly (
    id integer NOT NULL,
    project text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    CONSTRAINT anomaly_type_check CHECK ((type ~~ 'bb.anomaly.%'::text))
);


--
-- Name: anomaly_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.anomaly_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: anomaly_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.anomaly_id_seq OWNED BY public.anomaly.id;


--
-- Name: audit_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_log (
    id bigint NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: audit_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.audit_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: audit_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.audit_log_id_seq OWNED BY public.audit_log.id;


--
-- Name: changelist; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changelist (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL
);


--
-- Name: changelist_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.changelist_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: changelist_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.changelist_id_seq OWNED BY public.changelist.id;


--
-- Name: changelog; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changelog (
    id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    status text NOT NULL,
    prev_sync_history_id bigint,
    sync_history_id bigint,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    CONSTRAINT changelog_status_check CHECK ((status = ANY (ARRAY['PENDING'::text, 'DONE'::text, 'FAILED'::text])))
);


--
-- Name: changelog_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.changelog_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: changelog_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.changelog_id_seq OWNED BY public.changelog.id;


--
-- Name: data_source; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.data_source (
    id integer NOT NULL,
    options jsonb DEFAULT '{}'::jsonb NOT NULL,
    instance text NOT NULL
);


--
-- Name: data_source_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.data_source_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: data_source_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.data_source_id_seq OWNED BY public.data_source.id;


--
-- Name: db; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.db (
    id integer NOT NULL,
    environment text,
    name text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    project text NOT NULL,
    instance text NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);


--
-- Name: db_group; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.db_group (
    id bigint NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT ''::text NOT NULL,
    expression jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    project text NOT NULL
);


--
-- Name: db_group_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.db_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: db_group_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.db_group_id_seq OWNED BY public.db_group.id;


--
-- Name: db_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.db_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: db_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.db_id_seq OWNED BY public.db.id;


--
-- Name: db_schema; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.db_schema (
    id integer NOT NULL,
    metadata json DEFAULT '{}'::json NOT NULL,
    raw_dump text DEFAULT ''::text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL
);


--
-- Name: db_schema_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.db_schema_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: db_schema_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.db_schema_id_seq OWNED BY public.db_schema.id;


--
-- Name: environment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.environment (
    id integer NOT NULL,
    name text NOT NULL,
    "order" integer NOT NULL,
    resource_id text NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    CONSTRAINT environment_order_check CHECK (("order" >= 0))
);


--
-- Name: environment_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.environment_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: environment_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.environment_id_seq OWNED BY public.environment.id;


--
-- Name: export_archive; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.export_archive (
    id integer NOT NULL,
    bytes bytea,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: export_archive_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.export_archive_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: export_archive_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.export_archive_id_seq OWNED BY public.export_archive.id;


--
-- Name: idp; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.idp (
    id integer NOT NULL,
    resource_id text NOT NULL,
    name text NOT NULL,
    domain text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    CONSTRAINT idp_type_check CHECK ((type = ANY (ARRAY['OAUTH2'::text, 'OIDC'::text, 'LDAP'::text])))
);


--
-- Name: idp_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.idp_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: idp_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.idp_id_seq OWNED BY public.idp.id;


--
-- Name: instance; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.instance (
    id integer NOT NULL,
    environment text,
    resource_id text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);


--
-- Name: instance_change_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.instance_change_history (
    id bigint NOT NULL,
    version text NOT NULL
);


--
-- Name: instance_change_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.instance_change_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: instance_change_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.instance_change_history_id_seq OWNED BY public.instance_change_history.id;


--
-- Name: instance_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.instance_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: instance_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.instance_id_seq OWNED BY public.instance.id;


--
-- Name: issue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    plan_id bigint,
    pipeline_id integer,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    ts_vector tsvector,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    CONSTRAINT issue_status_check CHECK ((status = ANY (ARRAY['OPEN'::text, 'DONE'::text, 'CANCELED'::text]))),
    CONSTRAINT issue_type_check CHECK ((type ~~ 'bb.issue.%'::text))
);


--
-- Name: issue_comment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_comment (
    id bigint NOT NULL,
    creator_id integer NOT NULL,
    issue_id integer NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: issue_comment_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.issue_comment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: issue_comment_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.issue_comment_id_seq OWNED BY public.issue_comment.id;


--
-- Name: issue_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.issue_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: issue_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.issue_id_seq OWNED BY public.issue.id;


--
-- Name: issue_subscriber; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_subscriber (
    issue_id integer NOT NULL,
    subscriber_id integer NOT NULL
);


--
-- Name: pipeline; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pipeline (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL
);


--
-- Name: pipeline_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.pipeline_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pipeline_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.pipeline_id_seq OWNED BY public.pipeline.id;


--
-- Name: plan; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan (
    id bigint NOT NULL,
    creator_id integer NOT NULL,
    pipeline_id integer,
    name text NOT NULL,
    description text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL
);


--
-- Name: plan_check_run; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_check_run (
    id integer NOT NULL,
    plan_id bigint NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT plan_check_run_status_check CHECK ((status = ANY (ARRAY['RUNNING'::text, 'DONE'::text, 'FAILED'::text, 'CANCELED'::text]))),
    CONSTRAINT plan_check_run_type_check CHECK ((type ~~ 'bb.plan-check.%'::text))
);


--
-- Name: plan_check_run_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.plan_check_run_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: plan_check_run_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.plan_check_run_id_seq OWNED BY public.plan_check_run.id;


--
-- Name: plan_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.plan_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: plan_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.plan_id_seq OWNED BY public.plan.id;


--
-- Name: policy; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.policy (
    id integer NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    resource_type text NOT NULL,
    inherit_from_parent boolean DEFAULT true NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    resource text NOT NULL,
    enforce boolean DEFAULT true NOT NULL,
    CONSTRAINT policy_type_check CHECK ((type ~~ 'bb.policy.%'::text)),
    CONSTRAINT resource_type_type CHECK ((resource_type = ANY (ARRAY['WORKSPACE'::text, 'ENVIRONMENT'::text, 'PROJECT'::text, 'INSTANCE'::text])))
);


--
-- Name: policy_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.policy_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: policy_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.policy_id_seq OWNED BY public.policy.id;


--
-- Name: principal; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.principal (
    id integer NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text DEFAULT ''::text NOT NULL,
    mfa_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    profile jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    CONSTRAINT principal_type_check CHECK ((type = ANY (ARRAY['END_USER'::text, 'SYSTEM_BOT'::text, 'SERVICE_ACCOUNT'::text])))
);


--
-- Name: principal_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.principal_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: principal_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.principal_id_seq OWNED BY public.principal.id;


--
-- Name: project; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project (
    id integer NOT NULL,
    name text NOT NULL,
    resource_id text NOT NULL,
    data_classification_config_id text DEFAULT ''::text NOT NULL,
    setting jsonb DEFAULT '{}'::jsonb NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);


--
-- Name: project_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.project_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: project_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.project_id_seq OWNED BY public.project.id;


--
-- Name: project_webhook; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_webhook (
    id integer NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    activity_list text[] NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    project text NOT NULL,
    CONSTRAINT project_webhook_type_check CHECK ((type ~~ 'bb.plugin.webhook.%'::text))
);


--
-- Name: project_webhook_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.project_webhook_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: project_webhook_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.project_webhook_id_seq OWNED BY public.project_webhook.id;


--
-- Name: query_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.query_history (
    id bigint NOT NULL,
    creator_id integer NOT NULL,
    project_id text NOT NULL,
    database text NOT NULL,
    statement text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: query_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.query_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: query_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.query_history_id_seq OWNED BY public.query_history.id;


--
-- Name: release; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.release (
    id bigint NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    project text NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);


--
-- Name: release_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.release_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: release_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.release_id_seq OWNED BY public.release.id;


--
-- Name: review_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.review_config (
    id text NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    enabled boolean DEFAULT true NOT NULL
);


--
-- Name: revision; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.revision (
    id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleter_id integer,
    deleted_at timestamp with time zone,
    version text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL
);


--
-- Name: revision_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.revision_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: revision_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.revision_id_seq OWNED BY public.revision.id;


--
-- Name: risk; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.risk (
    id bigint NOT NULL,
    source text NOT NULL,
    level bigint NOT NULL,
    name text NOT NULL,
    active boolean NOT NULL,
    expression jsonb NOT NULL,
    CONSTRAINT risk_source_check CHECK ((source ~~ 'bb.risk.%'::text))
);


--
-- Name: risk_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.risk_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: risk_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.risk_id_seq OWNED BY public.risk.id;


--
-- Name: role; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role (
    id bigint NOT NULL,
    resource_id text NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    permissions jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: role_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.role_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: role_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.role_id_seq OWNED BY public.role.id;


--
-- Name: setting; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.setting (
    id integer NOT NULL,
    name text NOT NULL,
    value text NOT NULL
);


--
-- Name: setting_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.setting_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: setting_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.setting_id_seq OWNED BY public.setting.id;


--
-- Name: sheet; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sheet (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    name text NOT NULL,
    sha256 bytea NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL
);


--
-- Name: sheet_blob; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sheet_blob (
    sha256 bytea NOT NULL,
    content text NOT NULL
);


--
-- Name: sheet_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sheet_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sheet_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sheet_id_seq OWNED BY public.sheet.id;


--
-- Name: stage; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stage (
    id integer NOT NULL,
    pipeline_id integer NOT NULL,
    environment text NOT NULL
);


--
-- Name: stage_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.stage_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: stage_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.stage_id_seq OWNED BY public.stage.id;


--
-- Name: sync_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sync_history (
    id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    metadata json DEFAULT '{}'::json NOT NULL,
    raw_dump text DEFAULT ''::text NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL
);


--
-- Name: sync_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sync_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sync_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sync_history_id_seq OWNED BY public.sync_history.id;


--
-- Name: task; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task (
    id integer NOT NULL,
    pipeline_id integer NOT NULL,
    stage_id integer NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    earliest_allowed_at timestamp with time zone,
    instance text NOT NULL,
    db_name text,
    CONSTRAINT task_type_check CHECK ((type ~~ 'bb.task.%'::text))
);


--
-- Name: task_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_id_seq OWNED BY public.task.id;


--
-- Name: task_run; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_run (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    task_id integer NOT NULL,
    sheet_id integer,
    attempt integer NOT NULL,
    status text NOT NULL,
    code integer DEFAULT 0 NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    CONSTRAINT task_run_status_check CHECK ((status = ANY (ARRAY['PENDING'::text, 'RUNNING'::text, 'DONE'::text, 'FAILED'::text, 'CANCELED'::text])))
);


--
-- Name: task_run_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_run_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_run_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_run_id_seq OWNED BY public.task_run.id;


--
-- Name: task_run_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_run_log (
    id bigint NOT NULL,
    task_run_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: task_run_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_run_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_run_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_run_log_id_seq OWNED BY public.task_run_log.id;


--
-- Name: user_group; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_group (
    email text NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: worksheet; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.worksheet (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    name text NOT NULL,
    statement text NOT NULL,
    visibility text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    instance text,
    db_name text
);


--
-- Name: worksheet_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.worksheet_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: worksheet_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.worksheet_id_seq OWNED BY public.worksheet.id;


--
-- Name: worksheet_organizer; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.worksheet_organizer (
    id integer NOT NULL,
    worksheet_id integer NOT NULL,
    principal_id integer NOT NULL,
    starred boolean DEFAULT false NOT NULL
);


--
-- Name: worksheet_organizer_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.worksheet_organizer_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: worksheet_organizer_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.worksheet_organizer_id_seq OWNED BY public.worksheet_organizer.id;


--
-- Name: anomaly id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.anomaly ALTER COLUMN id SET DEFAULT nextval('public.anomaly_id_seq'::regclass);


--
-- Name: audit_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_log ALTER COLUMN id SET DEFAULT nextval('public.audit_log_id_seq'::regclass);


--
-- Name: changelist id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist ALTER COLUMN id SET DEFAULT nextval('public.changelist_id_seq'::regclass);


--
-- Name: changelog id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog ALTER COLUMN id SET DEFAULT nextval('public.changelog_id_seq'::regclass);


--
-- Name: data_source id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_source ALTER COLUMN id SET DEFAULT nextval('public.data_source_id_seq'::regclass);


--
-- Name: db id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db ALTER COLUMN id SET DEFAULT nextval('public.db_id_seq'::regclass);


--
-- Name: db_group id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_group ALTER COLUMN id SET DEFAULT nextval('public.db_group_id_seq'::regclass);


--
-- Name: db_schema id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_schema ALTER COLUMN id SET DEFAULT nextval('public.db_schema_id_seq'::regclass);


--
-- Name: environment id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment ALTER COLUMN id SET DEFAULT nextval('public.environment_id_seq'::regclass);


--
-- Name: export_archive id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.export_archive ALTER COLUMN id SET DEFAULT nextval('public.export_archive_id_seq'::regclass);


--
-- Name: idp id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.idp ALTER COLUMN id SET DEFAULT nextval('public.idp_id_seq'::regclass);


--
-- Name: instance id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance ALTER COLUMN id SET DEFAULT nextval('public.instance_id_seq'::regclass);


--
-- Name: instance_change_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_change_history ALTER COLUMN id SET DEFAULT nextval('public.instance_change_history_id_seq'::regclass);


--
-- Name: issue id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue ALTER COLUMN id SET DEFAULT nextval('public.issue_id_seq'::regclass);


--
-- Name: issue_comment id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comment ALTER COLUMN id SET DEFAULT nextval('public.issue_comment_id_seq'::regclass);


--
-- Name: pipeline id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline ALTER COLUMN id SET DEFAULT nextval('public.pipeline_id_seq'::regclass);


--
-- Name: plan id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan ALTER COLUMN id SET DEFAULT nextval('public.plan_id_seq'::regclass);


--
-- Name: plan_check_run id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_check_run ALTER COLUMN id SET DEFAULT nextval('public.plan_check_run_id_seq'::regclass);


--
-- Name: policy id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.policy ALTER COLUMN id SET DEFAULT nextval('public.policy_id_seq'::regclass);


--
-- Name: principal id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.principal ALTER COLUMN id SET DEFAULT nextval('public.principal_id_seq'::regclass);


--
-- Name: project id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project ALTER COLUMN id SET DEFAULT nextval('public.project_id_seq'::regclass);


--
-- Name: project_webhook id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhook ALTER COLUMN id SET DEFAULT nextval('public.project_webhook_id_seq'::regclass);


--
-- Name: query_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.query_history ALTER COLUMN id SET DEFAULT nextval('public.query_history_id_seq'::regclass);


--
-- Name: release id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release ALTER COLUMN id SET DEFAULT nextval('public.release_id_seq'::regclass);


--
-- Name: revision id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision ALTER COLUMN id SET DEFAULT nextval('public.revision_id_seq'::regclass);


--
-- Name: risk id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.risk ALTER COLUMN id SET DEFAULT nextval('public.risk_id_seq'::regclass);


--
-- Name: role id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role ALTER COLUMN id SET DEFAULT nextval('public.role_id_seq'::regclass);


--
-- Name: setting id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.setting ALTER COLUMN id SET DEFAULT nextval('public.setting_id_seq'::regclass);


--
-- Name: sheet id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet ALTER COLUMN id SET DEFAULT nextval('public.sheet_id_seq'::regclass);


--
-- Name: stage id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage ALTER COLUMN id SET DEFAULT nextval('public.stage_id_seq'::regclass);


--
-- Name: sync_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sync_history ALTER COLUMN id SET DEFAULT nextval('public.sync_history_id_seq'::regclass);


--
-- Name: task id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task ALTER COLUMN id SET DEFAULT nextval('public.task_id_seq'::regclass);


--
-- Name: task_run id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run ALTER COLUMN id SET DEFAULT nextval('public.task_run_id_seq'::regclass);


--
-- Name: task_run_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run_log ALTER COLUMN id SET DEFAULT nextval('public.task_run_log_id_seq'::regclass);


--
-- Name: worksheet id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet ALTER COLUMN id SET DEFAULT nextval('public.worksheet_id_seq'::regclass);


--
-- Name: worksheet_organizer id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet_organizer ALTER COLUMN id SET DEFAULT nextval('public.worksheet_organizer_id_seq'::regclass);


--
-- Data for Name: anomaly; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.anomaly (id, project, type, payload, updated_at, instance, db_name) VALUES (108, 'project-sample', 'bb.anomaly.database.schema.drift', '{}', '2025-03-12 03:24:50.877537-07', 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.anomaly (id, project, type, payload, updated_at, instance, db_name) VALUES (109, 'project-sample', 'bb.anomaly.database.schema.drift', '{}', '2025-03-12 03:25:00.879189-07', 'test-sample-instance', 'hr_test') ON CONFLICT DO NOTHING;


--
-- Data for Name: audit_log; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.audit_log (id, payload, created_at) VALUES (124, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "severity": "INFO"}', '2024-07-10 19:50:55-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (115, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM employee;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.010371s\", \"statement\": \"SELECT * FROM employee;\"}]}", "severity": "INFO"}', '2023-11-20 23:45:43-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (116, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM employee;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.007757s\", \"statement\": \"SELECT * FROM employee;\"}]}", "severity": "INFO"}', '2023-11-20 23:45:53-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (142, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\",\"email\":\"demo@example.com\",\"title\":\"Demo\",\"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:64249", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}', '2025-01-15 06:54:12-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (101, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM salary;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.005067s\", \"statement\": \"SELECT * FROM salary;\"}]}", "severity": "INFO"}', '2023-11-03 09:42:14-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (102, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM salary;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.003585s\", \"statement\": \"SELECT * FROM salary;\"}]}", "severity": "INFO"}', '2023-11-03 09:44:28-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (103, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_1\", \"statement\": \"SELECT * FROM salary\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_1", "response": "{\"results\": [{\"latency\": \"0.005666s\", \"statement\": \"SELECT * FROM salary\"}]}", "severity": "INFO"}', '2023-11-03 09:44:58-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (104, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_1\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_1", "response": "{\"results\": [{\"latency\": \"0.002445s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}', '2023-11-03 09:47:19-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (105, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_1\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_1", "response": "{\"results\": [{\"latency\": \"0.00149s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (106, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_2\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_2", "response": "{\"results\": [{\"latency\": \"0.001715s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (107, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_3\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_3", "response": "{\"results\": [{\"latency\": \"0.001481s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (108, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_4\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_4", "response": "{\"results\": [{\"latency\": \"0.001159s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (109, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_5\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_5", "response": "{\"results\": [{\"latency\": \"0.00101s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (110, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_6\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_6", "response": "{\"results\": [{\"latency\": \"0.001091s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (111, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM employee;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.005898s\", \"statement\": \"SELECT * FROM employee;\"}]}", "severity": "INFO"}', '2023-11-03 10:21:22-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (112, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM salary;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.003934s\", \"statement\": \"SELECT * FROM salary;\"}]}", "severity": "INFO"}', '2023-11-03 10:22:33-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (113, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM employee;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.00491s\", \"statement\": \"SELECT * FROM employee;\"}]}", "severity": "INFO"}', '2023-11-03 10:22:59-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (114, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM department;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.002054s\", \"statement\": \"SELECT * FROM department;\"}]}", "severity": "INFO"}', '2023-11-03 10:26:34-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (125, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"connectionDatabase\":\"hr_prod\", \"statement\":\"SELECT * FROM employee;\", \"limit\":1000}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\":[{\"columnNames\":[\"emp_no\", \"birth_date\", \"first_name\", \"last_name\", \"gender\", \"hire_date\"], \"columnTypeNames\":[\"INT4\", \"DATE\", \"TEXT\", \"TEXT\", \"TEXT\", \"DATE\"], \"masked\":[false, false, true, true, false, false], \"sensitive\":[false, false, true, true, false, false], \"latency\":\"0.008539708s\", \"statement\":\"WITH result AS (\\nSELECT * FROM employee\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO"}', '2024-07-10 20:42:09-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (117, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"SELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.002s\", \"statement\": \"SELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\"}]}", "severity": "INFO"}', '2024-04-10 06:59:38-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (118, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"SELECT project.resource_id, count(*)\\nFROM issue\\nLEFT JOIN project ON issue.project_id = project.id\\nWHERE EXISTS (\\n        SELECT 1 FROM activity, principal, member\\n        WHERE TO_TIMESTAMP(activity.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\\n        AND activity.type = ''bb.issue.comment.create''\\n        AND activity.container_id = issue.id\\n        AND activity.payload->''approvalEvent''->>''status'' = ''APPROVED''\\n        AND activity.creator_id = principal.id\\n        AND principal.id = member.principal_id\\n        AND member.\\\"role\\\" = ''DBA''\\n) AND TO_TIMESTAMP(issue.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\\nGROUP BY project.resource_id;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.003567s\", \"statement\": \"SELECT project.resource_id, count(*)\\nFROM issue\\nLEFT JOIN project ON issue.project_id = project.id\\nWHERE EXISTS (\\n        SELECT 1 FROM activity, principal, member\\n        WHERE TO_TIMESTAMP(activity.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\\n        AND activity.type = ''bb.issue.comment.create''\\n        AND activity.container_id = issue.id\\n        AND activity.payload->''approvalEvent''->>''status'' = ''APPROVED''\\n        AND activity.creator_id = principal.id\\n        AND principal.id = member.principal_id\\n        AND member.\\\"role\\\" = ''DBA''\\n) AND TO_TIMESTAMP(issue.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\\nGROUP BY project.resource_id;\"}]}", "severity": "INFO"}', '2024-04-10 07:02:29-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (119, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"SELECT\\n  issue.id AS issue_id,\\n  issue.creator_id as creator_id,\\n  COALESCE(\\n    array_agg(DISTINCT principal.email) FILTER (\\n      WHERE\\n        task_run.creator_id IS NOT NULL\\n    ),\\n    ''{}''\\n  ) AS releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n  LEFT JOIN task_run ON task_run.task_id = task.id\\n  LEFT JOIN principal ON task_run.creator_id = principal.id\\nWHERE\\n  principal.id = issue.creator_id\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  issue.id\\nORDER BY\\n  issue.id\\n\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.00262s\", \"statement\": \"SELECT\\n  issue.id AS issue_id,\\n  issue.creator_id as creator_id,\\n  COALESCE(\\n    array_agg(DISTINCT principal.email) FILTER (\\n      WHERE\\n        task_run.creator_id IS NOT NULL\\n    ),\\n    ''{}''\\n  ) AS releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n  LEFT JOIN task_run ON task_run.task_id = task.id\\n  LEFT JOIN principal ON task_run.creator_id = principal.id\\nWHERE\\n  principal.id = issue.creator_id\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  issue.id\\nORDER BY\\n  issue.id\\n\"}]}", "severity": "INFO"}', '2024-04-10 07:06:16-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (120, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"WITH issue_approvers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          x.status = ''APPROVED''\\n      ),\\n      ''{}''\\n    ) AS approver_emails\\n  FROM\\n    issue\\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \\\"principalId\\\" int) ON TRUE\\n    LEFT JOIN principal ON principal.id = x.\\\"principalId\\\"\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n),\\nissue_releasers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          task_run.creator_id IS NOT NULL\\n      ),\\n      ''{}''\\n    ) AS releaser_emails\\n  FROM\\n    issue\\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n    LEFT JOIN task_run ON task_run.task_id = task.id\\n    LEFT JOIN principal ON task_run.creator_id = principal.id\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n)\\n\\nSELECT\\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\\n  COUNT(issue.id) AS issue_count,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\\nWHERE\\n  issue.status = ''DONE''\\n  AND ia.approver_emails @> ir.releaser_emails\\n  AND ir.releaser_emails @> ia.approver_emails\\n  AND array_length(ir.releaser_emails, 1) > 0\\nGROUP BY\\n  month,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nORDER BY\\n  month;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.002993s\", \"statement\": \"WITH issue_approvers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          x.status = ''APPROVED''\\n      ),\\n      ''{}''\\n    ) AS approver_emails\\n  FROM\\n    issue\\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \\\"principalId\\\" int) ON TRUE\\n    LEFT JOIN principal ON principal.id = x.\\\"principalId\\\"\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n),\\nissue_releasers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          task_run.creator_id IS NOT NULL\\n      ),\\n      ''{}''\\n    ) AS releaser_emails\\n  FROM\\n    issue\\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n    LEFT JOIN task_run ON task_run.task_id = task.id\\n    LEFT JOIN principal ON task_run.creator_id = principal.id\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n)\\n\\nSELECT\\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\\n  COUNT(issue.id) AS issue_count,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\\nWHERE\\n  issue.status = ''DONE''\\n  AND ia.approver_emails @> ir.releaser_emails\\n  AND ir.releaser_emails @> ia.approver_emails\\n  AND array_length(ir.releaser_emails, 1) > 0\\nGROUP BY\\n  month,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nORDER BY\\n  month;\"}]}", "severity": "INFO"}', '2024-04-10 07:13:41-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (121, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"WITH issue_approvers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          x.status = ''APPROVED''\\n      ),\\n      ''{}''\\n    ) AS approver_emails\\n  FROM\\n    issue\\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \\\"principalId\\\" int) ON TRUE\\n    LEFT JOIN principal ON principal.id = x.\\\"principalId\\\"\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n),\\nissue_releasers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          task_run.creator_id IS NOT NULL\\n      ),\\n      ''{}''\\n    ) AS releaser_emails\\n  FROM\\n    issue\\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n    LEFT JOIN task_run ON task_run.task_id = task.id\\n    LEFT JOIN principal ON task_run.creator_id = principal.id\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n)\\n\\nSELECT\\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\\n  COUNT(issue.id) AS issue_count,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\\nWHERE\\n  issue.status = ''DONE''\\n  AND ia.approver_emails @> ir.releaser_emails\\n  AND ir.releaser_emails @> ia.approver_emails\\n  AND array_length(ir.releaser_emails, 1) > 0\\nGROUP BY\\n  month,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nORDER BY\\n  month;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.003365s\", \"statement\": \"WITH issue_approvers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          x.status = ''APPROVED''\\n      ),\\n      ''{}''\\n    ) AS approver_emails\\n  FROM\\n    issue\\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \\\"principalId\\\" int) ON TRUE\\n    LEFT JOIN principal ON principal.id = x.\\\"principalId\\\"\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n),\\nissue_releasers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          task_run.creator_id IS NOT NULL\\n      ),\\n      ''{}''\\n    ) AS releaser_emails\\n  FROM\\n    issue\\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n    LEFT JOIN task_run ON task_run.task_id = task.id\\n    LEFT JOIN principal ON task_run.creator_id = principal.id\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n)\\n\\nSELECT\\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\\n  COUNT(issue.id) AS issue_count,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\\nWHERE\\n  issue.status = ''DONE''\\n  AND ia.approver_emails @> ir.releaser_emails\\n  AND ir.releaser_emails @> ia.approver_emails\\n  AND array_length(ir.releaser_emails, 1) > 0\\nGROUP BY\\n  month,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nORDER BY\\n  month;\"}]}", "severity": "INFO"}', '2024-04-10 07:15:50-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (135, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"FALLBACK\"}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"FALLBACK\"},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:34:53-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (122, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"-- Fully completed issues by project\\nSELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.00281s\", \"statement\": \"-- Fully completed issues by project\\nSELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\"}]}", "severity": "INFO"}', '2024-04-10 07:19:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (126, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"connectionDatabase\":\"hr_prod\", \"statement\":\"SELECT * FROM salary;\", \"limit\":1000}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\":[{\"columnNames\":[\"emp_no\", \"amount\", \"from_date\", \"to_date\"], \"columnTypeNames\":[\"INT4\", \"INT4\", \"DATE\", \"DATE\"], \"masked\":[false, true, false, false], \"sensitive\":[false, true, false, false], \"latency\":\"0.008754250s\", \"statement\":\"WITH result AS (\\nSELECT * FROM salary\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO"}', '2024-07-10 20:42:14-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (127, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"connectionDatabase\":\"hr_prod\", \"statement\":\"SELECT * FROM employee;\", \"limit\":1000}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\":[{\"columnNames\":[\"emp_no\", \"birth_date\", \"first_name\", \"last_name\", \"gender\", \"hire_date\"], \"columnTypeNames\":[\"INT4\", \"DATE\", \"TEXT\", \"TEXT\", \"TEXT\", \"DATE\"], \"masked\":[false, false, true, true, false, false], \"sensitive\":[false, false, true, true, false, false], \"latency\":\"0.010567041s\", \"statement\":\"WITH result AS (\\nSELECT * FROM employee\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO"}', '2024-07-10 20:42:33-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (128, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\", \"web\":true}", "resource": "demo@example.com", "severity": "INFO"}', '2024-07-22 01:09:16-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (129, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\":\"instances/bytebase-meta/databases/bb\",\"statement\":\"SELECT\\n  issue.creator_id,\\n  principal.email,\\n  COUNT(issue.creator_id) AS amount\\nFROM issue\\nINNER JOIN principal\\nON issue.creator_id = principal.id\\nGROUP BY issue.creator_id, principal.email\\nORDER BY COUNT(issue.creator_id) DESC;\",\"limit\":1000,\"dataSourceId\":\"777072ed-539e-4cc2-a41e-6cc2917a7e7c\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\":[{\"columnNames\":[\"creator_id\",\"email\",\"amount\"],\"columnTypeNames\":[\"INT4\",\"TEXT\",\"INT8\"],\"masked\":[false,false,false],\"sensitive\":[false,false,false],\"latency\":\"0.021570708s\",\"statement\":\"WITH result AS (\\nSELECT\\n  issue.creator_id,\\n  principal.email,\\n  COUNT(issue.creator_id) AS amount\\nFROM issue\\nINNER JOIN principal\\nON issue.creator_id = principal.id\\nGROUP BY issue.creator_id, principal.email\\nORDER BY COUNT(issue.creator_id) DESC\\n) SELECT * FROM result LIMIT 1000;\"}],\"allowExport\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:62138", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"}}', '2024-09-20 01:13:54-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (130, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:33:24-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (131, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:33:50-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (132, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:33:51-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (133, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:34:32-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (134, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:34:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (136, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"},\"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:35:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (137, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"e543ddfd-f633-4dbe-87f3-6b171a96e20a\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"},\"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:36:00-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (138, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"},\"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:36:03-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (139, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"e543ddfd-f633-4dbe-87f3-6b171a96e20a\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"},\"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:36:03-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (140, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"}}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:36:03-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (141, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"e543ddfd-f633-4dbe-87f3-6b171a96e20a\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"}}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"e543ddfd-f633-4dbe-87f3-6b171a96e20a\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}', '2024-09-28 07:36:03-07') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (143, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\",\"email\":\"demo@example.com\",\"title\":\"Demo\",\"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:55846", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}', '2025-01-15 09:19:00-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (144, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\", \"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\", \"email\":\"demo@example.com\", \"title\":\"Demo\", \"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:61958", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}', '2025-01-15 09:39:56-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (145, '{"user": "users/101", "method": "/bytebase.v1.PlanService/CreatePlan", "parent": "projects/gitops-project", "request": "{\"parent\":\"projects/gitops-project\",\"plan\":{\"name\":\"projects/gitops-project/plans/-102\",\"steps\":[{\"specs\":[{\"id\":\"ff8ecf1c-f037-4544-971c-c3f4c8ff5889\",\"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"sheet\":\"projects/gitops-project/sheets/133\",\"type\":\"BASELINE\"}}]}]}}", "response": "{\"name\":\"projects/gitops-project/plans/108\",\"steps\":[{\"specs\":[{\"id\":\"ff8ecf1c-f037-4544-971c-c3f4c8ff5889\",\"specReleaseSource\":{},\"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"sheet\":\"projects/gitops-project/sheets/133\",\"type\":\"BASELINE\",\"preUpdateBackupDetail\":{}}}]}],\"vcsSource\":{},\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:17:55Z\",\"updateTime\":\"2025-01-16T04:17:55Z\",\"releaseSource\":{},\"deploymentConfigSnapshot\":{\"name\":\"gitops-project/deploymentConfigs/default\",\"schedule\":{\"deployments\":[{\"title\":\"Test Stage\",\"id\":\"0\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"test\"]}]}}},{\"title\":\"Prod Stage\",\"id\":\"1\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"prod\"]}]}}}]}}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49476", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-15 20:17:55-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (146, '{"user": "users/101", "method": "/bytebase.v1.IssueService/CreateIssue", "parent": "projects/gitops-project", "request": "{\"parent\":\"projects/gitops-project\",\"issue\":{\"name\":\"projects/gitops-project/issues/-101\",\"title\":\"Establish \\\"hr_prod_vcs\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"plan\":\"projects/gitops-project/plans/108\"}}", "response": "{\"name\":\"projects/gitops-project/issues/108\",\"title\":\"Establish \\\"hr_prod_vcs\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:17:55Z\",\"updateTime\":\"2025-01-16T04:17:55Z\",\"plan\":\"projects/gitops-project/plans/108\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49476", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-15 20:17:55-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (147, '{"user": "users/101", "method": "/bytebase.v1.RolloutService/CreateRollout", "parent": "projects/gitops-project", "request": "{\"parent\":\"projects/gitops-project\",\"rollout\":{\"plan\":\"projects/gitops-project/plans/108\"}}", "response": "{\"name\":\"projects/gitops-project/rollouts/108\",\"plan\":\"projects/gitops-project/plans/108\",\"title\":\"Rollout Pipeline\",\"stages\":[{\"name\":\"projects/gitops-project/rollouts/108/stages/111\",\"id\":\"1\",\"title\":\"Prod Stage\",\"tasks\":[{\"name\":\"projects/gitops-project/rollouts/108/stages/111/tasks/114\",\"title\":\"Establish baseline for database \\\"hr_prod_vcs\\\"\",\"specId\":\"ff8ecf1c-f037-4544-971c-c3f4c8ff5889\",\"status\":\"NOT_STARTED\",\"type\":\"DATABASE_SCHEMA_BASELINE\",\"target\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"databaseSchemaBaseline\":{}}]}],\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:17:55Z\",\"updateTime\":\"2025-01-16T04:17:55Z\",\"issue\":\"projects/gitops-project/issues/108\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49476", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-15 20:17:55-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (148, '{"user": "users/101", "method": "/bytebase.v1.PlanService/CreatePlan", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"plan\":{\"name\":\"projects/project-sample/plans/-108\",\"steps\":[{\"specs\":[{\"id\":\"231a929d-bb89-4845-8b7c-6e4870116d32\",\"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod\",\"sheet\":\"projects/project-sample/sheets/134\",\"type\":\"BASELINE\"}}]}]}}", "response": "{\"name\":\"projects/project-sample/plans/109\",\"steps\":[{\"specs\":[{\"id\":\"231a929d-bb89-4845-8b7c-6e4870116d32\",\"specReleaseSource\":{},\"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod\",\"sheet\":\"projects/project-sample/sheets/134\",\"type\":\"BASELINE\",\"preUpdateBackupDetail\":{}}}]}],\"vcsSource\":{},\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:19:35Z\",\"updateTime\":\"2025-01-16T04:19:35Z\",\"releaseSource\":{},\"deploymentConfigSnapshot\":{\"name\":\"project-sample/deploymentConfigs/default\",\"schedule\":{\"deployments\":[{\"title\":\"Test Stage\",\"id\":\"0\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"test\"]}]}}},{\"title\":\"Prod Stage\",\"id\":\"1\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"prod\"]}]}}}]}}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49477", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-15 20:19:35-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (149, '{"user": "users/101", "method": "/bytebase.v1.IssueService/CreateIssue", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"issue\":{\"name\":\"projects/project-sample/issues/-107\",\"title\":\"Establish \\\"hr_prod\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"plan\":\"projects/project-sample/plans/109\"}}", "response": "{\"name\":\"projects/project-sample/issues/109\",\"title\":\"Establish \\\"hr_prod\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:19:35Z\",\"updateTime\":\"2025-01-16T04:19:35Z\",\"plan\":\"projects/project-sample/plans/109\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49477", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-15 20:19:35-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (150, '{"user": "users/101", "method": "/bytebase.v1.RolloutService/CreateRollout", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"rollout\":{\"plan\":\"projects/project-sample/plans/109\"}}", "response": "{\"name\":\"projects/project-sample/rollouts/109\",\"plan\":\"projects/project-sample/plans/109\",\"title\":\"Rollout Pipeline\",\"stages\":[{\"name\":\"projects/project-sample/rollouts/109/stages/112\",\"id\":\"1\",\"title\":\"Prod Stage\",\"tasks\":[{\"name\":\"projects/project-sample/rollouts/109/stages/112/tasks/115\",\"title\":\"Establish baseline for database \\\"hr_prod\\\"\",\"specId\":\"231a929d-bb89-4845-8b7c-6e4870116d32\",\"status\":\"NOT_STARTED\",\"type\":\"DATABASE_SCHEMA_BASELINE\",\"target\":\"instances/prod-sample-instance/databases/hr_prod\",\"databaseSchemaBaseline\":{}}]}],\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:19:35Z\",\"updateTime\":\"2025-01-16T04:19:35Z\",\"issue\":\"projects/project-sample/issues/109\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49477", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-15 20:19:35-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (151, '{"user": "users/101", "method": "/bytebase.v1.PlanService/CreatePlan", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"plan\":{\"name\":\"projects/project-sample/plans/-102\",\"steps\":[{\"specs\":[{\"id\":\"913aa19f-18e6-42c5-b6e7-2fbb358cffee\",\"changeDatabaseConfig\":{\"target\":\"instances/test-sample-instance/databases/hr_test\",\"sheet\":\"projects/project-sample/sheets/135\",\"type\":\"BASELINE\"}}]}]}}", "response": "{\"name\":\"projects/project-sample/plans/110\",\"steps\":[{\"specs\":[{\"id\":\"913aa19f-18e6-42c5-b6e7-2fbb358cffee\",\"specReleaseSource\":{},\"changeDatabaseConfig\":{\"target\":\"instances/test-sample-instance/databases/hr_test\",\"sheet\":\"projects/project-sample/sheets/135\",\"type\":\"BASELINE\",\"preUpdateBackupDetail\":{}}}]}],\"vcsSource\":{},\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:21:52Z\",\"updateTime\":\"2025-01-16T04:21:52Z\",\"releaseSource\":{},\"deploymentConfigSnapshot\":{\"name\":\"project-sample/deploymentConfigs/default\",\"schedule\":{\"deployments\":[{\"title\":\"Test Stage\",\"id\":\"0\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"test\"]}]}}},{\"title\":\"Prod Stage\",\"id\":\"1\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"prod\"]}]}}}]}}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49478", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-15 20:21:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (152, '{"user": "users/101", "method": "/bytebase.v1.IssueService/CreateIssue", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"issue\":{\"name\":\"projects/project-sample/issues/-101\",\"title\":\"Establish \\\"hr_test\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"plan\":\"projects/project-sample/plans/110\"}}", "response": "{\"name\":\"projects/project-sample/issues/110\",\"title\":\"Establish \\\"hr_test\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:21:52Z\",\"updateTime\":\"2025-01-16T04:21:52Z\",\"plan\":\"projects/project-sample/plans/110\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49478", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-15 20:21:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (154, '{"user": "users/101", "method": "/bytebase.v1.SQLService/AdminExecute", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"statement\":\"ALTER TABLE employee ADD COLUMN bugfix TEXT NOT NULL DEFAULT '''';\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_vcs", "response": "{\"results\":[{\"columnNames\":[\"Affected Rows\"],\"columnTypeNames\":[\"INT\"],\"latency\":\"0.007736708s\",\"statement\":\"ALTER TABLE employee ADD COLUMN bugfix TEXT NOT NULL DEFAULT '''';\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:64858", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}', '2025-01-15 20:22:49-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (155, '{"user": "users/101", "method": "/bytebase.v1.SQLService/AdminExecute", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "status": {"code": 13, "message": "failed to receive request: rpc error: code = Canceled desc = context canceled"}, "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"statement\":\"ALTER TABLE employee ADD COLUMN bugfix TEXT NOT NULL DEFAULT '''';\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_vcs", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:64858", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}', '2025-01-15 20:35:13-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (153, '{"user": "users/101", "method": "/bytebase.v1.RolloutService/CreateRollout", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"rollout\":{\"plan\":\"projects/project-sample/plans/110\"}}", "response": "{\"name\":\"projects/project-sample/rollouts/110\",\"plan\":\"projects/project-sample/plans/110\",\"title\":\"Rollout Pipeline\",\"stages\":[{\"name\":\"projects/project-sample/rollouts/110/stages/113\",\"id\":\"0\",\"title\":\"Test Stage\",\"tasks\":[{\"name\":\"projects/project-sample/rollouts/110/stages/113/tasks/116\",\"title\":\"Establish baseline for database \\\"hr_test\\\"\",\"specId\":\"913aa19f-18e6-42c5-b6e7-2fbb358cffee\",\"status\":\"NOT_STARTED\",\"type\":\"DATABASE_SCHEMA_BASELINE\",\"target\":\"instances/test-sample-instance/databases/hr_test\",\"databaseSchemaBaseline\":{}}]}],\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:21:52Z\",\"updateTime\":\"2025-01-16T04:21:52Z\",\"issue\":\"projects/project-sample/issues/110\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49478", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-15 20:21:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (156, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\":\"instances/bytebase-meta/databases/bb\", \"statement\":\"SELECT\\n  *\\nFROM\\n  \\\"public\\\".\\\"release\\\"\\nLIMIT\\n  50;\", \"limit\":1000, \"dataSourceId\":\"777072ed-539e-4cc2-a41e-6cc2917a7e7c\", \"schema\":\"public\", \"queryOption\":{\"redisRunCommandsOn\":\"SINGLE_NODE\"}, \"container\":\"release\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\":[{\"columnNames\":[\"id\", \"row_status\", \"project_id\", \"creator_id\", \"created_ts\", \"payload\"], \"columnTypeNames\":[\"INT8\", \"16398\", \"INT4\", \"INT4\", \"INT8\", \"JSONB\"], \"latency\":\"0.000670256s\", \"statement\":\"WITH result AS (\\nSELECT\\n  *\\nFROM\\n  \\\"public\\\".\\\"release\\\"\\nLIMIT\\n  50\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "192.168.215.1:34778", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-22 22:26:12-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (157, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\":\"instances/bytebase-meta/databases/bb\", \"statement\":\"SELECT\\n  *\\nFROM\\n  \\\"public\\\".\\\"release\\\"\\nLIMIT\\n  50;\", \"limit\":1000, \"dataSourceId\":\"777072ed-539e-4cc2-a41e-6cc2917a7e7c\", \"schema\":\"public\", \"queryOption\":{\"redisRunCommandsOn\":\"SINGLE_NODE\"}, \"container\":\"release\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\":[{\"columnNames\":[\"id\", \"row_status\", \"project_id\", \"creator_id\", \"created_ts\", \"payload\"], \"columnTypeNames\":[\"INT8\", \"16398\", \"INT4\", \"INT4\", \"INT8\", \"JSONB\"], \"latency\":\"0.001700141s\", \"statement\":\"WITH result AS (\\nSELECT\\n  *\\nFROM\\n  \\\"public\\\".\\\"release\\\"\\nLIMIT\\n  50\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "192.168.215.1:34778", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}', '2025-01-22 22:26:15-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (158, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\", \"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\", \"email\":\"demo@example.com\", \"title\":\"Demo\", \"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:35806", "callerSuppliedUserAgent": "grpc-go/1.70.0"}}', '2025-02-07 20:49:31-08') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, payload, created_at) VALUES (159, '{"user": "users/101", "method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\", \"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\", \"email\":\"demo@example.com\", \"title\":\"Demo\", \"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:51235", "callerSuppliedUserAgent": "grpc-go/1.70.0"}}', '2025-03-12 03:20:19.117797-07') ON CONFLICT DO NOTHING;


--
-- Data for Name: changelist; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: changelog; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (110, '2023-12-14 05:55:44-08', 'DONE', 101, 102, '{"type": "MIGRATE", "issue": "projects/gitops-project/issues/102", "sheet": "projects/gitops-project/sheets/104", "taskRun": "projects/gitops-project/rollouts/102/stages/103/tasks/103/taskRuns/101", "changedResources": {"databases": [{"name": "hr_prod_vcs", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}', 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (112, '2025-01-15 20:19:11.016112-08', 'DONE', 103, 104, '{"type": "BASELINE", "issue": "projects/gitops-project/issues/108", "taskRun": "projects/gitops-project/rollouts/108/stages/111/tasks/114/taskRuns/102"}', 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (113, '2025-01-15 20:19:38.010659-08', 'DONE', 105, 106, '{"type": "BASELINE", "issue": "projects/project-sample/issues/109", "taskRun": "projects/project-sample/rollouts/109/stages/112/tasks/115/taskRuns/103"}', 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (114, '2025-01-15 20:21:56.237441-08', 'DONE', 107, 108, '{"type": "BASELINE", "issue": "projects/project-sample/issues/110", "taskRun": "projects/project-sample/rollouts/110/stages/113/tasks/116/taskRuns/104"}', 'test-sample-instance', 'hr_test') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (115, '2025-01-22 22:25:51.581999-08', 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/104/tasks/105/taskRuns/106", "changedResources": {}}', 'prod-sample-instance', 'hr_prod_4') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (116, '2025-01-22 22:25:51.583034-08', 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/104/tasks/104/taskRuns/105", "changedResources": {}}', 'prod-sample-instance', 'hr_prod_1') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (117, '2025-01-22 22:25:56.577497-08', 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/105/tasks/106/taskRuns/107", "changedResources": {}}', 'prod-sample-instance', 'hr_prod_2') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (118, '2025-01-22 22:25:56.578394-08', 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/105/tasks/107/taskRuns/108", "changedResources": {}}', 'prod-sample-instance', 'hr_prod_5') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (119, '2025-01-22 22:26:01.571769-08', 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/106/tasks/109/taskRuns/110", "changedResources": {}}', 'prod-sample-instance', 'hr_prod_6') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, status, prev_sync_history_id, sync_history_id, payload, instance, db_name) VALUES (120, '2025-01-22 22:26:01.580441-08', 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/106/tasks/108/taskRuns/109", "changedResources": {}}', 'prod-sample-instance', 'hr_prod_3') ON CONFLICT DO NOTHING;


--
-- Data for Name: data_source; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.data_source (id, options, instance) VALUES (101, '{"id": "admin", "host": "/tmp", "port": "8083", "type": "ADMIN", "database": "", "username": "bbsample", "obfuscatedSslCa": "", "obfuscatedSslKey": "", "obfuscatedSslCert": "", "obfuscatedPassword": ""}', 'test-sample-instance') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, options, instance) VALUES (102, '{"id": "admin", "host": "/tmp", "port": "8084", "type": "ADMIN", "database": "", "username": "bbsample", "obfuscatedSslCa": "", "obfuscatedSslKey": "", "obfuscatedSslCert": "", "obfuscatedPassword": ""}', 'prod-sample-instance') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, options, instance) VALUES (103, '{"id": "777072ed-539e-4cc2-a41e-6cc2917a7e7c", "host": "/tmp", "port": "8082", "type": "ADMIN", "database": "", "username": "bb", "obfuscatedSslCa": "", "obfuscatedSslKey": "", "obfuscatedSslCert": "", "obfuscatedPassword": ""}', 'bytebase-meta') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, options, instance) VALUES (104, '{"id": "351173a6-f320-45c5-8d95-8e17abe08964", "host": "/tmp", "port": "8084", "type": "READ_ONLY", "database": "", "username": "bbsample", "obfuscatedSslCa": "", "obfuscatedSslKey": "", "obfuscatedSslCert": "", "authenticationType": "PASSWORD", "obfuscatedPassword": "WyYTVD4="}', 'prod-sample-instance') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, options, instance) VALUES (105, '{"id": "e543ddfd-f633-4dbe-87f3-6b171a96e20a", "host": "/tmp", "port": "8084", "type": "READ_ONLY", "database": "", "username": "bbsample", "obfuscatedSslCa": "", "obfuscatedSslKey": "", "obfuscatedSslCert": "", "authenticationType": "PASSWORD", "obfuscatedPassword": "WyYTVD4="}', 'prod-sample-instance') ON CONFLICT DO NOTHING;


--
-- Data for Name: db; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (103, NULL, 'hr_prod_1', '{"labels": {"location": "asia"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}', 'batch-project', 'prod-sample-instance', true) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (104, NULL, 'hr_prod_2', '{"labels": {"location": "eu"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}', 'batch-project', 'prod-sample-instance', true) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (105, NULL, 'hr_prod_3', '{"labels": {"location": "na"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}', 'batch-project', 'prod-sample-instance', true) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (106, NULL, 'hr_prod_4', '{"labels": {"location": "asia"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}', 'batch-project', 'prod-sample-instance', true) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (107, NULL, 'hr_prod_5', '{"labels": {"location": "eu"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}', 'batch-project', 'prod-sample-instance', true) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (108, NULL, 'hr_prod_6', '{"labels": {"location": "na"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}', 'batch-project', 'prod-sample-instance', true) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (109, NULL, 'hr_prod_vcs', '{"lastSyncTime": "2025-01-16T04:22:57Z", "backupAvailable": true}', 'gitops-project', 'prod-sample-instance', true) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (112, NULL, 'postgres', '{"lastSyncTime": "2025-03-12T10:24:50.866764Z"}', 'default', 'prod-sample-instance', false) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (102, NULL, 'hr_prod', '{"lastSyncTime": "2025-03-12T10:24:50.874613Z", "backupAvailable": true}', 'project-sample', 'prod-sample-instance', false) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (113, NULL, 'postgres', '{"lastSyncTime": "2025-03-12T10:25:00.867661Z"}', 'default', 'test-sample-instance', false) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (101, NULL, 'hr_test', '{"lastSyncTime": "2025-03-12T10:25:00.875983Z", "backupAvailable": true}', 'project-sample', 'test-sample-instance', false) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (110, NULL, 'postgres', '{"lastSyncTime": "2025-03-12T10:25:20.852689Z"}', 'default', 'bytebase-meta', false) ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, environment, name, metadata, project, instance, deleted) VALUES (111, NULL, 'bb', '{"lastSyncTime": "2025-03-12T10:25:20.867851Z"}', 'metadb-project', 'bytebase-meta', false) ON CONFLICT DO NOTHING;


--
-- Data for Name: db_group; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.db_group (id, resource_id, placeholder, expression, payload, project) VALUES (102, 'all-databases', 'all-databases', '{"expression": "true"}', '{"multitenancy": true}', 'batch-project') ON CONFLICT DO NOTHING;
INSERT INTO public.db_group (id, resource_id, placeholder, expression, payload, project) VALUES (101, 'all-hr-group', 'all-hr-group', '{"expression": "resource.environment_name == \"environments/prod\" && resource.database_name.startsWith(\"hr_prod\")"}', '{}', 'batch-project') ON CONFLICT DO NOTHING;


--
-- Data for Name: db_schema; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (104, '{"name":"hr_prod_2","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', '{}', 'prod-sample-instance', 'hr_prod_2') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (108, '{"name":"hr_prod_6","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', '{}', 'prod-sample-instance', 'hr_prod_6') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (103, '{"name":"hr_prod_1","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', '{}', 'prod-sample-instance', 'hr_prod_1') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (121, '{"name":"bb","schemas":[{"name":"public","tables":[{"name":"anomaly","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.anomaly_id_seq''::regclass)","type":"integer"},{"name":"project","position":2,"type":"text"},{"name":"type","position":3,"type":"text"},{"name":"payload","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"updated_at","position":5,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"instance","position":6,"type":"text"},{"name":"db_name","position":7,"type":"text"}],"indexes":[{"name":"anomaly_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX anomaly_pkey ON public.anomaly USING btree (id);","isConstraint":true},{"name":"idx_anomaly_unique_project_instance_dn_name_type","expressions":["project","instance","db_name","type"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_anomaly_unique_project_instance_dn_name_type ON public.anomaly USING btree (project, instance, db_name, type);"}],"dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"anomaly_instance_db_name_fkey","columns":["instance","db_name"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["instance","name"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"audit_log","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_log_id_seq''::regclass)","type":"bigint"},{"name":"payload","position":2,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":3,"defaultExpression":"now()","type":"timestamp with time zone"}],"indexes":[{"name":"audit_log_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_log_pkey ON public.audit_log USING btree (id);","isConstraint":true},{"name":"idx_audit_log_created_at","expressions":["created_at"],"type":"btree","definition":"CREATE INDEX idx_audit_log_created_at ON public.audit_log USING btree (created_at);"},{"name":"idx_audit_log_payload_method","expressions":["payload ->> ''method''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_method ON public.audit_log USING btree (((payload ->> ''method''::text)));"},{"name":"idx_audit_log_payload_parent","expressions":["payload ->> ''parent''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_parent ON public.audit_log USING btree (((payload ->> ''parent''::text)));"},{"name":"idx_audit_log_payload_resource","expressions":["payload ->> ''resource''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_resource ON public.audit_log USING btree (((payload ->> ''resource''::text)));"},{"name":"idx_audit_log_payload_user","expressions":["payload ->> ''user''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_user ON public.audit_log USING btree (((payload ->> ''user''::text)));"}],"rowCount":"57","dataSize":"90112","indexSize":"98304","owner":"bb"},{"name":"changelist","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.changelist_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"name","position":3,"type":"text"},{"name":"payload","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"updated_at","position":5,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"project","position":6,"type":"text"}],"indexes":[{"name":"changelist_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX changelist_pkey ON public.changelist USING btree (id);","isConstraint":true},{"name":"idx_changelist_project_name","expressions":["project","name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_changelist_project_name ON public.changelist USING btree (project, name);"}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"changelist_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelist_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"changelog","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.changelog_id_seq''::regclass)","type":"bigint"},{"name":"created_at","position":2,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"status","position":3,"type":"text"},{"name":"prev_sync_history_id","position":4,"nullable":true,"type":"bigint"},{"name":"sync_history_id","position":5,"nullable":true,"type":"bigint"},{"name":"payload","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"instance","position":7,"type":"text"},{"name":"db_name","position":8,"type":"text"}],"indexes":[{"name":"changelog_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX changelog_pkey ON public.changelog USING btree (id);","isConstraint":true},{"name":"idx_changelog_instance_db_name","expressions":["instance","db_name"],"type":"btree","definition":"CREATE INDEX idx_changelog_instance_db_name ON public.changelog USING btree (instance, db_name);"}],"rowCount":"10","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"changelog_instance_db_name_fkey","columns":["instance","db_name"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["instance","name"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelog_prev_sync_history_id_fkey","columns":["prev_sync_history_id"],"referencedSchema":"public","referencedTable":"sync_history","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelog_sync_history_id_fkey","columns":["sync_history_id"],"referencedSchema":"public","referencedTable":"sync_history","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"data_source","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.data_source_id_seq''::regclass)","type":"integer"},{"name":"options","position":11,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"instance","position":13,"type":"text"}],"indexes":[{"name":"data_source_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX data_source_pkey ON public.data_source USING btree (id);","isConstraint":true}],"rowCount":"5","dataSize":"49152","indexSize":"16384","foreignKeys":[{"name":"data_source_instance_fkey","columns":["instance"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"db","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.db_id_seq''::regclass)","type":"integer"},{"name":"environment","position":2,"nullable":true,"type":"text"},{"name":"name","position":5,"type":"text"},{"name":"metadata","position":9,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"project","position":11,"type":"text"},{"name":"instance","position":12,"type":"text"},{"name":"deleted","position":13,"defaultExpression":"false","type":"boolean"}],"indexes":[{"name":"db_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX db_pkey ON public.db USING btree (id);","isConstraint":true},{"name":"idx_db_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_db_project ON public.db USING btree (project);"},{"name":"idx_db_unique_instance_name","expressions":["instance","name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_unique_instance_name ON public.db USING btree (instance, name);"}],"rowCount":"11","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"db_environment_fkey","columns":["environment"],"referencedSchema":"public","referencedTable":"environment","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_instance_fkey","columns":["instance"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"db_group","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.db_group_id_seq''::regclass)","type":"bigint"},{"name":"resource_id","position":2,"type":"text"},{"name":"placeholder","position":3,"defaultExpression":"''''::text","type":"text"},{"name":"expression","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"payload","position":5,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"project","position":6,"type":"text"}],"indexes":[{"name":"db_group_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX db_group_pkey ON public.db_group USING btree (id);","isConstraint":true},{"name":"idx_db_group_unique_project_placeholder","expressions":["project","placeholder"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_group_unique_project_placeholder ON public.db_group USING btree (project, placeholder);"},{"name":"idx_db_group_unique_project_resource_id","expressions":["project","resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_group_unique_project_resource_id ON public.db_group USING btree (project, resource_id);"}],"rowCount":"2","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"db_group_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"db_schema","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.db_schema_id_seq''::regclass)","type":"integer"},{"name":"metadata","position":2,"defaultExpression":"''{}''::json","type":"json"},{"name":"raw_dump","position":3,"defaultExpression":"''''::text","type":"text"},{"name":"config","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"instance","position":5,"type":"text"},{"name":"db_name","position":6,"type":"text"}],"indexes":[{"name":"db_schema_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX db_schema_pkey ON public.db_schema USING btree (id);","isConstraint":true},{"name":"idx_db_schema_unique_instance_db_name","expressions":["instance","db_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_schema_unique_instance_db_name ON public.db_schema USING btree (instance, db_name);"}],"rowCount":"11","dataSize":"163840","indexSize":"32768","foreignKeys":[{"name":"db_schema_instance_db_name_fkey","columns":["instance","db_name"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["instance","name"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"environment","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.environment_id_seq''::regclass)","type":"integer"},{"name":"name","position":2,"type":"text"},{"name":"order","position":3,"type":"integer"},{"name":"resource_id","position":4,"type":"text"},{"name":"deleted","position":5,"defaultExpression":"false","type":"boolean"}],"indexes":[{"name":"environment_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX environment_pkey ON public.environment USING btree (id);","isConstraint":true},{"name":"idx_environment_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_environment_unique_resource_id ON public.environment USING btree (resource_id);"}],"rowCount":"2","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"export_archive","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.export_archive_id_seq''::regclass)","type":"integer"},{"name":"bytes","position":2,"nullable":true,"type":"bytea"},{"name":"payload","position":3,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":4,"defaultExpression":"now()","type":"timestamp with time zone"}],"indexes":[{"name":"export_archive_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX export_archive_pkey ON public.export_archive USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"8192","owner":"bb"},{"name":"idp","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.idp_id_seq''::regclass)","type":"integer"},{"name":"resource_id","position":2,"type":"text"},{"name":"name","position":3,"type":"text"},{"name":"domain","position":4,"type":"text"},{"name":"type","position":5,"type":"text"},{"name":"config","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"deleted","position":7,"defaultExpression":"false","type":"boolean"}],"indexes":[{"name":"idp_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX idp_pkey ON public.idp USING btree (id);","isConstraint":true},{"name":"idx_idp_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_idp_unique_resource_id ON public.idp USING btree (resource_id);"}],"dataSize":"8192","indexSize":"16384","owner":"bb"},{"name":"instance","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.instance_id_seq''::regclass)","type":"integer"},{"name":"environment","position":2,"nullable":true,"type":"text"},{"name":"resource_id","position":7,"type":"text"},{"name":"metadata","position":10,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"deleted","position":11,"defaultExpression":"false","type":"boolean"}],"indexes":[{"name":"idx_instance_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_instance_unique_resource_id ON public.instance USING btree (resource_id);"},{"name":"instance_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX instance_pkey ON public.instance USING btree (id);","isConstraint":true}],"rowCount":"3","dataSize":"57344","indexSize":"32768","foreignKeys":[{"name":"instance_environment_fkey","columns":["environment"],"referencedSchema":"public","referencedTable":"environment","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"instance_change_history","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.instance_change_history_id_seq''::regclass)","type":"bigint"},{"name":"version","position":3,"type":"text"}],"indexes":[{"name":"idx_instance_change_history_unique_version","expressions":["version"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_instance_change_history_unique_version ON public.instance_change_history USING btree (version);"},{"name":"instance_change_history_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX instance_change_history_pkey ON public.instance_change_history USING btree (id);","isConstraint":true}],"rowCount":"20","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"issue","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.issue_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"plan_id","position":3,"nullable":true,"type":"bigint"},{"name":"pipeline_id","position":4,"nullable":true,"type":"integer"},{"name":"name","position":5,"type":"text"},{"name":"status","position":6,"type":"text"},{"name":"type","position":7,"type":"text"},{"name":"description","position":8,"defaultExpression":"''''::text","type":"text"},{"name":"payload","position":11,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"ts_vector","position":12,"nullable":true,"type":"tsvector"},{"name":"created_at","position":13,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":14,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"project","position":15,"type":"text"}],"indexes":[{"name":"idx_issue_creator_id","expressions":["creator_id"],"type":"btree","definition":"CREATE INDEX idx_issue_creator_id ON public.issue USING btree (creator_id);"},{"name":"idx_issue_pipeline_id","expressions":["pipeline_id"],"type":"btree","definition":"CREATE INDEX idx_issue_pipeline_id ON public.issue USING btree (pipeline_id);"},{"name":"idx_issue_plan_id","expressions":["plan_id"],"type":"btree","definition":"CREATE INDEX idx_issue_plan_id ON public.issue USING btree (plan_id);"},{"name":"idx_issue_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_issue_project ON public.issue USING btree (project);"},{"name":"idx_issue_ts_vector","expressions":["ts_vector"],"type":"gin","definition":"CREATE INDEX idx_issue_ts_vector ON public.issue USING gin (ts_vector);"},{"name":"issue_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX issue_pkey ON public.issue USING btree (id);","isConstraint":true}],"rowCount":"10","dataSize":"16384","indexSize":"98304","foreignKeys":[{"name":"issue_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_plan_id_fkey","columns":["plan_id"],"referencedSchema":"public","referencedTable":"plan","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"issue_comment","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.issue_comment_id_seq''::regclass)","type":"bigint"},{"name":"creator_id","position":2,"type":"integer"},{"name":"issue_id","position":3,"type":"integer"},{"name":"payload","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":5,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":6,"defaultExpression":"now()","type":"timestamp with time zone"}],"indexes":[{"name":"idx_issue_comment_issue_id","expressions":["issue_id"],"type":"btree","definition":"CREATE INDEX idx_issue_comment_issue_id ON public.issue_comment USING btree (issue_id);"},{"name":"issue_comment_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX issue_comment_pkey ON public.issue_comment USING btree (id);","isConstraint":true}],"rowCount":"37","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"issue_comment_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_comment_issue_id_fkey","columns":["issue_id"],"referencedSchema":"public","referencedTable":"issue","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"issue_subscriber","columns":[{"name":"issue_id","position":1,"type":"integer"},{"name":"subscriber_id","position":2,"type":"integer"}],"indexes":[{"name":"idx_issue_subscriber_subscriber_id","expressions":["subscriber_id"],"type":"btree","definition":"CREATE INDEX idx_issue_subscriber_subscriber_id ON public.issue_subscriber USING btree (subscriber_id);"},{"name":"issue_subscriber_pkey","expressions":["issue_id","subscriber_id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX issue_subscriber_pkey ON public.issue_subscriber USING btree (issue_id, subscriber_id);","isConstraint":true}],"indexSize":"16384","foreignKeys":[{"name":"issue_subscriber_issue_id_fkey","columns":["issue_id"],"referencedSchema":"public","referencedTable":"issue","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_subscriber_subscriber_id_fkey","columns":["subscriber_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"pipeline","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.pipeline_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"name","position":3,"type":"text"},{"name":"created_at","position":4,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"project","position":5,"type":"text"}],"indexes":[{"name":"pipeline_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX pipeline_pkey ON public.pipeline USING btree (id);","isConstraint":true}],"rowCount":"10","dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"pipeline_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"pipeline_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"plan","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.plan_id_seq''::regclass)","type":"bigint"},{"name":"creator_id","position":2,"type":"integer"},{"name":"pipeline_id","position":3,"nullable":true,"type":"integer"},{"name":"name","position":4,"type":"text"},{"name":"description","position":5,"type":"text"},{"name":"config","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":7,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":8,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"project","position":9,"type":"text"}],"indexes":[{"name":"idx_plan_pipeline_id","expressions":["pipeline_id"],"type":"btree","definition":"CREATE INDEX idx_plan_pipeline_id ON public.plan USING btree (pipeline_id);"},{"name":"idx_plan_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_plan_project ON public.plan USING btree (project);"},{"name":"plan_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX plan_pkey ON public.plan USING btree (id);","isConstraint":true}],"rowCount":"10","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"plan_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"plan_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"plan_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"plan_check_run","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.plan_check_run_id_seq''::regclass)","type":"integer"},{"name":"plan_id","position":2,"type":"bigint"},{"name":"status","position":3,"type":"text"},{"name":"type","position":4,"type":"text"},{"name":"config","position":5,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"result","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"payload","position":7,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":8,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":9,"defaultExpression":"now()","type":"timestamp with time zone"}],"indexes":[{"name":"idx_plan_check_run_plan_id","expressions":["plan_id"],"type":"btree","definition":"CREATE INDEX idx_plan_check_run_plan_id ON public.plan_check_run USING btree (plan_id);"},{"name":"plan_check_run_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX plan_check_run_pkey ON public.plan_check_run USING btree (id);","isConstraint":true}],"rowCount":"54","dataSize":"57344","indexSize":"32768","foreignKeys":[{"name":"plan_check_run_plan_id_fkey","columns":["plan_id"],"referencedSchema":"public","referencedTable":"plan","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"policy","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.policy_id_seq''::regclass)","type":"integer"},{"name":"type","position":2,"type":"text"},{"name":"payload","position":3,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"resource_type","position":4,"type":"text"},{"name":"inherit_from_parent","position":5,"defaultExpression":"true","type":"boolean"},{"name":"updated_at","position":6,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"resource","position":7,"type":"text"},{"name":"enforce","position":8,"defaultExpression":"true","type":"boolean"}],"indexes":[{"name":"idx_policy_unique_resource_type_resource_type","expressions":["resource_type","resource","type"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_type ON public.policy USING btree (resource_type, resource, type);"},{"name":"policy_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX policy_pkey ON public.policy USING btree (id);","isConstraint":true}],"rowCount":"15","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"principal","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.principal_id_seq''::regclass)","type":"integer"},{"name":"type","position":2,"type":"text"},{"name":"name","position":3,"type":"text"},{"name":"email","position":4,"type":"text"},{"name":"password_hash","position":5,"type":"text"},{"name":"phone","position":6,"defaultExpression":"''''::text","type":"text"},{"name":"mfa_config","position":7,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"profile","position":8,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":9,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"deleted","position":10,"defaultExpression":"false","type":"boolean"}],"indexes":[{"name":"principal_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX principal_pkey ON public.principal USING btree (id);","isConstraint":true}],"rowCount":"11","dataSize":"16384","indexSize":"16384","owner":"bb"},{"name":"project","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.project_id_seq''::regclass)","type":"integer"},{"name":"name","position":2,"type":"text"},{"name":"resource_id","position":3,"type":"text"},{"name":"data_classification_config_id","position":4,"defaultExpression":"''''::text","type":"text"},{"name":"setting","position":5,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"deleted","position":6,"defaultExpression":"false","type":"boolean"}],"indexes":[{"name":"idx_project_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_project_unique_resource_id ON public.project USING btree (resource_id);"},{"name":"project_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX project_pkey ON public.project USING btree (id);","isConstraint":true}],"rowCount":"5","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"project_webhook","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.project_webhook_id_seq''::regclass)","type":"integer"},{"name":"type","position":2,"type":"text"},{"name":"name","position":3,"type":"text"},{"name":"url","position":4,"type":"text"},{"name":"activity_list","position":5,"type":"_text"},{"name":"payload","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"project","position":7,"type":"text"}],"indexes":[{"name":"idx_project_webhook_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_project_webhook_project ON public.project_webhook USING btree (project);"},{"name":"project_webhook_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX project_webhook_pkey ON public.project_webhook USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"project_webhook_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"query_history","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.query_history_id_seq''::regclass)","type":"bigint"},{"name":"creator_id","position":2,"type":"integer"},{"name":"project_id","position":3,"type":"text"},{"name":"database","position":4,"type":"text"},{"name":"statement","position":5,"type":"text"},{"name":"type","position":6,"type":"text"},{"name":"payload","position":7,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":8,"defaultExpression":"now()","type":"timestamp with time zone"}],"indexes":[{"name":"idx_query_history_creator_id_created_at_project_id","expressions":["creator_id","created_at","project_id"],"type":"btree","definition":"CREATE INDEX idx_query_history_creator_id_created_at_project_id ON public.query_history USING btree (creator_id, created_at, project_id DESC);"},{"name":"query_history_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX query_history_pkey ON public.query_history USING btree (id);","isConstraint":true}],"rowCount":"30","dataSize":"49152","indexSize":"32768","foreignKeys":[{"name":"query_history_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"release","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.release_id_seq''::regclass)","type":"bigint"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_at","position":3,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"payload","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"project","position":5,"type":"text"},{"name":"deleted","position":6,"defaultExpression":"false","type":"boolean"}],"indexes":[{"name":"idx_release_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_release_project ON public.release USING btree (project);"},{"name":"release_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX release_pkey ON public.release USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"release_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"release_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"review_config","columns":[{"name":"id","position":1,"type":"text"},{"name":"name","position":2,"type":"text"},{"name":"payload","position":3,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"enabled","position":4,"defaultExpression":"true","type":"boolean"}],"indexes":[{"name":"review_config_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX review_config_pkey ON public.review_config USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"32768","indexSize":"16384","owner":"bb"},{"name":"revision","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.revision_id_seq''::regclass)","type":"bigint"},{"name":"created_at","position":2,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"deleter_id","position":3,"nullable":true,"type":"integer"},{"name":"deleted_at","position":4,"nullable":true,"type":"timestamp with time zone"},{"name":"version","position":5,"type":"text"},{"name":"payload","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"instance","position":7,"type":"text"},{"name":"db_name","position":8,"type":"text"}],"indexes":[{"name":"idx_revision_instance_db_name_version","expressions":["instance","db_name","version"],"type":"btree","definition":"CREATE INDEX idx_revision_instance_db_name_version ON public.revision USING btree (instance, db_name, version);"},{"name":"idx_revision_unique_instance_db_name_version_deleted_at_null","expressions":["instance","db_name","version"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_version_deleted_at_null ON public.revision USING btree (instance, db_name, version) WHERE (deleted_at IS NULL);"},{"name":"revision_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX revision_pkey ON public.revision USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"24576","foreignKeys":[{"name":"revision_deleter_id_fkey","columns":["deleter_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"revision_instance_db_name_fkey","columns":["instance","db_name"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["instance","name"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"risk","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.risk_id_seq''::regclass)","type":"bigint"},{"name":"source","position":2,"type":"text"},{"name":"level","position":3,"type":"bigint"},{"name":"name","position":4,"type":"text"},{"name":"active","position":5,"type":"boolean"},{"name":"expression","position":6,"type":"jsonb"}],"indexes":[{"name":"risk_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX risk_pkey ON public.risk USING btree (id);","isConstraint":true}],"rowCount":"2","dataSize":"16384","indexSize":"16384","owner":"bb"},{"name":"role","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.role_id_seq''::regclass)","type":"bigint"},{"name":"resource_id","position":2,"type":"text"},{"name":"name","position":3,"type":"text"},{"name":"description","position":4,"type":"text"},{"name":"permissions","position":5,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"payload","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_role_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_role_unique_resource_id ON public.role USING btree (resource_id);"},{"name":"role_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX role_pkey ON public.role USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"setting","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.setting_id_seq''::regclass)","type":"integer"},{"name":"name","position":2,"type":"text"},{"name":"value","position":3,"type":"text"}],"indexes":[{"name":"idx_setting_unique_name","expressions":["name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_setting_unique_name ON public.setting USING btree (name);"},{"name":"setting_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX setting_pkey ON public.setting USING btree (id);","isConstraint":true}],"rowCount":"15","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"sheet","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.sheet_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"name","position":3,"type":"text"},{"name":"sha256","position":4,"type":"bytea"},{"name":"payload","position":5,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":6,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"project","position":7,"type":"text"}],"indexes":[{"name":"idx_sheet_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_sheet_project ON public.sheet USING btree (project);"},{"name":"sheet_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sheet_pkey ON public.sheet USING btree (id);","isConstraint":true}],"rowCount":"13","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"sheet_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"sheet_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"sheet_blob","columns":[{"name":"sha256","position":1,"type":"bytea"},{"name":"content","position":2,"type":"text"}],"indexes":[{"name":"sheet_blob_pkey","expressions":["sha256"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sheet_blob_pkey ON public.sheet_blob USING btree (sha256);","isConstraint":true}],"rowCount":"9","dataSize":"16384","indexSize":"16384","owner":"bb"},{"name":"stage","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.stage_id_seq''::regclass)","type":"integer"},{"name":"pipeline_id","position":2,"type":"integer"},{"name":"environment","position":5,"type":"text"}],"indexes":[{"name":"idx_stage_pipeline_id","expressions":["pipeline_id"],"type":"btree","definition":"CREATE INDEX idx_stage_pipeline_id ON public.stage USING btree (pipeline_id);"},{"name":"stage_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX stage_pkey ON public.stage USING btree (id);","isConstraint":true}],"rowCount":"13","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"stage_environment_fkey","columns":["environment"],"referencedSchema":"public","referencedTable":"environment","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"stage_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"sync_history","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.sync_history_id_seq''::regclass)","type":"bigint"},{"name":"created_at","position":2,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"metadata","position":3,"defaultExpression":"''{}''::json","type":"json"},{"name":"raw_dump","position":4,"defaultExpression":"''''::text","type":"text"},{"name":"instance","position":5,"type":"text"},{"name":"db_name","position":6,"type":"text"}],"indexes":[{"name":"idx_sync_history_instance_db_name_created_at","expressions":["instance","db_name","created_at"],"type":"btree","definition":"CREATE INDEX idx_sync_history_instance_db_name_created_at ON public.sync_history USING btree (instance, db_name, created_at);"},{"name":"sync_history_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sync_history_pkey ON public.sync_history USING btree (id);","isConstraint":true}],"rowCount":"8","dataSize":"98304","indexSize":"32768","foreignKeys":[{"name":"sync_history_instance_db_name_fkey","columns":["instance","db_name"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["instance","name"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"task","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.task_id_seq''::regclass)","type":"integer"},{"name":"pipeline_id","position":2,"type":"integer"},{"name":"stage_id","position":3,"type":"integer"},{"name":"type","position":6,"type":"text"},{"name":"payload","position":7,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"earliest_allowed_at","position":8,"nullable":true,"type":"timestamp with time zone"},{"name":"instance","position":9,"type":"text"},{"name":"db_name","position":10,"nullable":true,"type":"text"}],"indexes":[{"name":"idx_task_pipeline_id_stage_id","expressions":["pipeline_id","stage_id"],"type":"btree","definition":"CREATE INDEX idx_task_pipeline_id_stage_id ON public.task USING btree (pipeline_id, stage_id);"},{"name":"task_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_pkey ON public.task USING btree (id);","isConstraint":true}],"rowCount":"16","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"task_instance_fkey","columns":["instance"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_stage_id_fkey","columns":["stage_id"],"referencedSchema":"public","referencedTable":"stage","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"task_run","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.task_run_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"task_id","position":3,"type":"integer"},{"name":"sheet_id","position":4,"nullable":true,"type":"integer"},{"name":"attempt","position":5,"type":"integer"},{"name":"status","position":7,"type":"text"},{"name":"code","position":8,"defaultExpression":"0","type":"integer"},{"name":"result","position":9,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":10,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":11,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"started_at","position":12,"nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"idx_task_run_task_id","expressions":["task_id"],"type":"btree","definition":"CREATE INDEX idx_task_run_task_id ON public.task_run USING btree (task_id);"},{"name":"task_run_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_run_pkey ON public.task_run USING btree (id);","isConstraint":true},{"name":"uk_task_run_task_id_attempt","expressions":["task_id","attempt"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON public.task_run USING btree (task_id, attempt);"}],"rowCount":"10","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"task_run_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_run_sheet_id_fkey","columns":["sheet_id"],"referencedSchema":"public","referencedTable":"sheet","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_run_task_id_fkey","columns":["task_id"],"referencedSchema":"public","referencedTable":"task","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"task_run_log","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.task_run_log_id_seq''::regclass)","type":"bigint"},{"name":"task_run_id","position":2,"type":"integer"},{"name":"created_at","position":3,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"payload","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_task_run_log_task_run_id","expressions":["task_run_id"],"type":"btree","definition":"CREATE INDEX idx_task_run_log_task_run_id ON public.task_run_log USING btree (task_run_id);"},{"name":"task_run_log_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_run_log_pkey ON public.task_run_log USING btree (id);","isConstraint":true}],"rowCount":"54","dataSize":"49152","indexSize":"32768","foreignKeys":[{"name":"task_run_log_task_run_id_fkey","columns":["task_run_id"],"referencedSchema":"public","referencedTable":"task_run","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"user_group","columns":[{"name":"email","position":1,"type":"text"},{"name":"name","position":2,"type":"text"},{"name":"description","position":3,"defaultExpression":"''''::text","type":"text"},{"name":"payload","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"user_group_pkey","expressions":["email"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX user_group_pkey ON public.user_group USING btree (email);","isConstraint":true}],"dataSize":"8192","indexSize":"8192","owner":"bb"},{"name":"worksheet","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.worksheet_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"name","position":3,"type":"text"},{"name":"statement","position":4,"type":"text"},{"name":"visibility","position":5,"type":"text"},{"name":"payload","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"created_at","position":7,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":8,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"project","position":9,"type":"text"},{"name":"instance","position":10,"nullable":true,"type":"text"},{"name":"db_name","position":11,"nullable":true,"type":"text"}],"indexes":[{"name":"idx_worksheet_creator_id_project","expressions":["creator_id","project"],"type":"btree","definition":"CREATE INDEX idx_worksheet_creator_id_project ON public.worksheet USING btree (creator_id, project);"},{"name":"worksheet_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX worksheet_pkey ON public.worksheet USING btree (id);","isConstraint":true}],"rowCount":"7","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"worksheet_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"worksheet_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"worksheet_organizer","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.worksheet_organizer_id_seq''::regclass)","type":"integer"},{"name":"worksheet_id","position":2,"type":"integer"},{"name":"principal_id","position":3,"type":"integer"},{"name":"starred","position":4,"defaultExpression":"false","type":"boolean"}],"indexes":[{"name":"idx_worksheet_organizer_principal_id","expressions":["principal_id"],"type":"btree","definition":"CREATE INDEX idx_worksheet_organizer_principal_id ON public.worksheet_organizer USING btree (principal_id);"},{"name":"idx_worksheet_organizer_unique_sheet_id_principal_id","expressions":["worksheet_id","principal_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal_id ON public.worksheet_organizer USING btree (worksheet_id, principal_id);"},{"name":"worksheet_organizer_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX worksheet_organizer_pkey ON public.worksheet_organizer USING btree (id);","isConstraint":true}],"indexSize":"24576","foreignKeys":[{"name":"worksheet_organizer_principal_id_fkey","columns":["principal_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"worksheet_organizer_worksheet_id_fkey","columns":["worksheet_id"],"referencedSchema":"public","referencedTable":"worksheet","referencedColumns":["id"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"}],"sequences":[{"name":"anomaly_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"109","ownerTable":"anomaly","ownerColumn":"id"},{"name":"audit_log_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"159","ownerTable":"audit_log","ownerColumn":"id"},{"name":"changelist_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"changelist","ownerColumn":"id"},{"name":"changelog_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"120","ownerTable":"changelog","ownerColumn":"id"},{"name":"data_source_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"105","ownerTable":"data_source","ownerColumn":"id"},{"name":"db_group_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"102","ownerTable":"db_group","ownerColumn":"id"},{"name":"db_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"115","ownerTable":"db","ownerColumn":"id"},{"name":"db_schema_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"188","ownerTable":"db_schema","ownerColumn":"id"},{"name":"environment_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"environment","ownerColumn":"id"},{"name":"export_archive_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"export_archive","ownerColumn":"id"},{"name":"idp_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"idp","ownerColumn":"id"},{"name":"instance_change_history_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"231","ownerTable":"instance_change_history","ownerColumn":"id"},{"name":"instance_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"103","ownerTable":"instance","ownerColumn":"id"},{"name":"issue_comment_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"141","ownerTable":"issue_comment","ownerColumn":"id"},{"name":"issue_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"issue","ownerColumn":"id"},{"name":"pipeline_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"pipeline","ownerColumn":"id"},{"name":"plan_check_run_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"171","ownerTable":"plan_check_run","ownerColumn":"id"},{"name":"plan_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"plan","ownerColumn":"id"},{"name":"policy_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"190","ownerTable":"policy","ownerColumn":"id"},{"name":"principal_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"109","ownerTable":"principal","ownerColumn":"id"},{"name":"project_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"104","ownerTable":"project","ownerColumn":"id"},{"name":"project_webhook_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"project_webhook","ownerColumn":"id"},{"name":"query_history_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"130","ownerTable":"query_history","ownerColumn":"id"},{"name":"release_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","ownerTable":"release","ownerColumn":"id"},{"name":"revision_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","ownerTable":"revision","ownerColumn":"id"},{"name":"risk_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"102","ownerTable":"risk","ownerColumn":"id"},{"name":"role_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"role","ownerColumn":"id"},{"name":"setting_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"206","ownerTable":"setting","ownerColumn":"id"},{"name":"sheet_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"135","ownerTable":"sheet","ownerColumn":"id"},{"name":"stage_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"113","ownerTable":"stage","ownerColumn":"id"},{"name":"sync_history_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"108","ownerTable":"sync_history","ownerColumn":"id"},{"name":"task_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"116","ownerTable":"task","ownerColumn":"id"},{"name":"task_run_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"task_run","ownerColumn":"id"},{"name":"task_run_log_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"154","ownerTable":"task_run_log","ownerColumn":"id"},{"name":"worksheet_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"worksheet","ownerColumn":"id"},{"name":"worksheet_organizer_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"worksheet_organizer","ownerColumn":"id"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bb"}', '
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

SET default_tablespace = '''';

CREATE SEQUENCE "public"."anomaly_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."anomaly" (
    "id" integer DEFAULT nextval(''public.anomaly_id_seq''::regclass) NOT NULL,
    "project" text NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "instance" text NOT NULL,
    "db_name" text NOT NULL
);

ALTER SEQUENCE "public"."anomaly_id_seq" OWNED BY "public"."anomaly"."id";

ALTER TABLE ONLY "public"."anomaly" ADD CONSTRAINT "anomaly_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_anomaly_unique_project_instance_dn_name_type" ON ONLY "public"."anomaly" ("project", "instance", "db_name", "type");

CREATE SEQUENCE "public"."audit_log_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."audit_log" (
    "id" bigint DEFAULT nextval(''public.audit_log_id_seq''::regclass) NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL
);

ALTER SEQUENCE "public"."audit_log_id_seq" OWNED BY "public"."audit_log"."id";

ALTER TABLE ONLY "public"."audit_log" ADD CONSTRAINT "audit_log_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_log_created_at" ON ONLY "public"."audit_log" ("created_at");

CREATE INDEX "idx_audit_log_payload_method" ON ONLY "public"."audit_log" ((payload ->> ''method''::text));

CREATE INDEX "idx_audit_log_payload_parent" ON ONLY "public"."audit_log" ((payload ->> ''parent''::text));

CREATE INDEX "idx_audit_log_payload_resource" ON ONLY "public"."audit_log" ((payload ->> ''resource''::text));

CREATE INDEX "idx_audit_log_payload_user" ON ONLY "public"."audit_log" ((payload ->> ''user''::text));

CREATE SEQUENCE "public"."changelist_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."changelist" (
    "id" integer DEFAULT nextval(''public.changelist_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "name" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL
);

ALTER SEQUENCE "public"."changelist_id_seq" OWNED BY "public"."changelist"."id";

ALTER TABLE ONLY "public"."changelist" ADD CONSTRAINT "changelist_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_changelist_project_name" ON ONLY "public"."changelist" ("project", "name");

CREATE SEQUENCE "public"."changelog_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."changelog" (
    "id" bigint DEFAULT nextval(''public.changelog_id_seq''::regclass) NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "status" text NOT NULL,
    "prev_sync_history_id" bigint,
    "sync_history_id" bigint,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "instance" text NOT NULL,
    "db_name" text NOT NULL
);

ALTER SEQUENCE "public"."changelog_id_seq" OWNED BY "public"."changelog"."id";

ALTER TABLE ONLY "public"."changelog" ADD CONSTRAINT "changelog_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_changelog_instance_db_name" ON ONLY "public"."changelog" ("instance", "db_name");

CREATE SEQUENCE "public"."data_source_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."data_source" (
    "id" integer DEFAULT nextval(''public.data_source_id_seq''::regclass) NOT NULL,
    "options" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "instance" text NOT NULL
);

ALTER SEQUENCE "public"."data_source_id_seq" OWNED BY "public"."data_source"."id";

ALTER TABLE ONLY "public"."data_source" ADD CONSTRAINT "data_source_pkey" PRIMARY KEY ("id");

CREATE SEQUENCE "public"."db_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."db" (
    "id" integer DEFAULT nextval(''public.db_id_seq''::regclass) NOT NULL,
    "environment" text,
    "name" text NOT NULL,
    "metadata" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "project" text NOT NULL,
    "instance" text NOT NULL,
    "deleted" boolean DEFAULT false NOT NULL
);

ALTER SEQUENCE "public"."db_id_seq" OWNED BY "public"."db"."id";

ALTER TABLE ONLY "public"."db" ADD CONSTRAINT "db_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_db_project" ON ONLY "public"."db" ("project");

CREATE UNIQUE INDEX "idx_db_unique_instance_name" ON ONLY "public"."db" ("instance", "name");

CREATE SEQUENCE "public"."db_group_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."db_group" (
    "id" bigint DEFAULT nextval(''public.db_group_id_seq''::regclass) NOT NULL,
    "resource_id" text NOT NULL,
    "placeholder" text DEFAULT ''''::text NOT NULL,
    "expression" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "project" text NOT NULL
);

ALTER SEQUENCE "public"."db_group_id_seq" OWNED BY "public"."db_group"."id";

ALTER TABLE ONLY "public"."db_group" ADD CONSTRAINT "db_group_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_db_group_unique_project_placeholder" ON ONLY "public"."db_group" ("project", "placeholder");

CREATE UNIQUE INDEX "idx_db_group_unique_project_resource_id" ON ONLY "public"."db_group" ("project", "resource_id");

CREATE SEQUENCE "public"."db_schema_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."db_schema" (
    "id" integer DEFAULT nextval(''public.db_schema_id_seq''::regclass) NOT NULL,
    "metadata" json DEFAULT ''{}''::json NOT NULL,
    "raw_dump" text DEFAULT ''''::text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "instance" text NOT NULL,
    "db_name" text NOT NULL
);

ALTER SEQUENCE "public"."db_schema_id_seq" OWNED BY "public"."db_schema"."id";

ALTER TABLE ONLY "public"."db_schema" ADD CONSTRAINT "db_schema_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_db_schema_unique_instance_db_name" ON ONLY "public"."db_schema" ("instance", "db_name");

CREATE SEQUENCE "public"."environment_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."environment" (
    "id" integer DEFAULT nextval(''public.environment_id_seq''::regclass) NOT NULL,
    "name" text NOT NULL,
    "order" integer NOT NULL,
    "resource_id" text NOT NULL,
    "deleted" boolean DEFAULT false NOT NULL
);

ALTER SEQUENCE "public"."environment_id_seq" OWNED BY "public"."environment"."id";

ALTER TABLE ONLY "public"."environment" ADD CONSTRAINT "environment_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_environment_unique_resource_id" ON ONLY "public"."environment" ("resource_id");

CREATE SEQUENCE "public"."export_archive_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."export_archive" (
    "id" integer DEFAULT nextval(''public.export_archive_id_seq''::regclass) NOT NULL,
    "bytes" bytea,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL
);

ALTER SEQUENCE "public"."export_archive_id_seq" OWNED BY "public"."export_archive"."id";

ALTER TABLE ONLY "public"."export_archive" ADD CONSTRAINT "export_archive_pkey" PRIMARY KEY ("id");

CREATE SEQUENCE "public"."idp_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."idp" (
    "id" integer DEFAULT nextval(''public.idp_id_seq''::regclass) NOT NULL,
    "resource_id" text NOT NULL,
    "name" text NOT NULL,
    "domain" text NOT NULL,
    "type" text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "deleted" boolean DEFAULT false NOT NULL
);

ALTER SEQUENCE "public"."idp_id_seq" OWNED BY "public"."idp"."id";

ALTER TABLE ONLY "public"."idp" ADD CONSTRAINT "idp_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_idp_unique_resource_id" ON ONLY "public"."idp" ("resource_id");

CREATE SEQUENCE "public"."instance_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."instance" (
    "id" integer DEFAULT nextval(''public.instance_id_seq''::regclass) NOT NULL,
    "environment" text,
    "resource_id" text NOT NULL,
    "metadata" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "deleted" boolean DEFAULT false NOT NULL
);

ALTER SEQUENCE "public"."instance_id_seq" OWNED BY "public"."instance"."id";

ALTER TABLE ONLY "public"."instance" ADD CONSTRAINT "instance_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_instance_unique_resource_id" ON ONLY "public"."instance" ("resource_id");

CREATE SEQUENCE "public"."instance_change_history_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."instance_change_history" (
    "id" bigint DEFAULT nextval(''public.instance_change_history_id_seq''::regclass) NOT NULL,
    "version" text NOT NULL
);

ALTER SEQUENCE "public"."instance_change_history_id_seq" OWNED BY "public"."instance_change_history"."id";

ALTER TABLE ONLY "public"."instance_change_history" ADD CONSTRAINT "instance_change_history_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_instance_change_history_unique_version" ON ONLY "public"."instance_change_history" ("version");

CREATE SEQUENCE "public"."issue_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."issue" (
    "id" integer DEFAULT nextval(''public.issue_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "plan_id" bigint,
    "pipeline_id" integer,
    "name" text NOT NULL,
    "status" text NOT NULL,
    "type" text NOT NULL,
    "description" text DEFAULT ''''::text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "ts_vector" tsvector,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL
);

ALTER SEQUENCE "public"."issue_id_seq" OWNED BY "public"."issue"."id";

ALTER TABLE ONLY "public"."issue" ADD CONSTRAINT "issue_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_issue_creator_id" ON ONLY "public"."issue" ("creator_id");

CREATE INDEX "idx_issue_pipeline_id" ON ONLY "public"."issue" ("pipeline_id");

CREATE INDEX "idx_issue_plan_id" ON ONLY "public"."issue" ("plan_id");

CREATE INDEX "idx_issue_project" ON ONLY "public"."issue" ("project");

CREATE INDEX "idx_issue_ts_vector" ON ONLY "public"."issue" ("ts_vector");

CREATE SEQUENCE "public"."issue_comment_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."issue_comment" (
    "id" bigint DEFAULT nextval(''public.issue_comment_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "issue_id" integer NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL
);

ALTER SEQUENCE "public"."issue_comment_id_seq" OWNED BY "public"."issue_comment"."id";

ALTER TABLE ONLY "public"."issue_comment" ADD CONSTRAINT "issue_comment_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_issue_comment_issue_id" ON ONLY "public"."issue_comment" ("issue_id");

CREATE TABLE "public"."issue_subscriber" (
    "issue_id" integer NOT NULL,
    "subscriber_id" integer NOT NULL
);

ALTER TABLE ONLY "public"."issue_subscriber" ADD CONSTRAINT "issue_subscriber_pkey" PRIMARY KEY ("issue_id", "subscriber_id");

CREATE INDEX "idx_issue_subscriber_subscriber_id" ON ONLY "public"."issue_subscriber" ("subscriber_id");

CREATE SEQUENCE "public"."pipeline_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."pipeline" (
    "id" integer DEFAULT nextval(''public.pipeline_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "name" text NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL
);

ALTER SEQUENCE "public"."pipeline_id_seq" OWNED BY "public"."pipeline"."id";

ALTER TABLE ONLY "public"."pipeline" ADD CONSTRAINT "pipeline_pkey" PRIMARY KEY ("id");

CREATE SEQUENCE "public"."plan_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."plan" (
    "id" bigint DEFAULT nextval(''public.plan_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "pipeline_id" integer,
    "name" text NOT NULL,
    "description" text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL
);

ALTER SEQUENCE "public"."plan_id_seq" OWNED BY "public"."plan"."id";

ALTER TABLE ONLY "public"."plan" ADD CONSTRAINT "plan_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_plan_pipeline_id" ON ONLY "public"."plan" ("pipeline_id");

CREATE INDEX "idx_plan_project" ON ONLY "public"."plan" ("project");

CREATE SEQUENCE "public"."plan_check_run_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."plan_check_run" (
    "id" integer DEFAULT nextval(''public.plan_check_run_id_seq''::regclass) NOT NULL,
    "plan_id" bigint NOT NULL,
    "status" text NOT NULL,
    "type" text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "result" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL
);

ALTER SEQUENCE "public"."plan_check_run_id_seq" OWNED BY "public"."plan_check_run"."id";

ALTER TABLE ONLY "public"."plan_check_run" ADD CONSTRAINT "plan_check_run_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_plan_check_run_plan_id" ON ONLY "public"."plan_check_run" ("plan_id");

CREATE SEQUENCE "public"."policy_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."policy" (
    "id" integer DEFAULT nextval(''public.policy_id_seq''::regclass) NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "resource_type" text NOT NULL,
    "inherit_from_parent" boolean DEFAULT true NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "resource" text NOT NULL,
    "enforce" boolean DEFAULT true NOT NULL
);

ALTER SEQUENCE "public"."policy_id_seq" OWNED BY "public"."policy"."id";

ALTER TABLE ONLY "public"."policy" ADD CONSTRAINT "policy_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_policy_unique_resource_type_resource_type" ON ONLY "public"."policy" ("resource_type", "resource", "type");

CREATE SEQUENCE "public"."principal_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."principal" (
    "id" integer DEFAULT nextval(''public.principal_id_seq''::regclass) NOT NULL,
    "type" text NOT NULL,
    "name" text NOT NULL,
    "email" text NOT NULL,
    "password_hash" text NOT NULL,
    "phone" text DEFAULT ''''::text NOT NULL,
    "mfa_config" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "profile" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "deleted" boolean DEFAULT false NOT NULL
);

ALTER SEQUENCE "public"."principal_id_seq" OWNED BY "public"."principal"."id";

ALTER TABLE ONLY "public"."principal" ADD CONSTRAINT "principal_pkey" PRIMARY KEY ("id");

CREATE SEQUENCE "public"."project_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."project" (
    "id" integer DEFAULT nextval(''public.project_id_seq''::regclass) NOT NULL,
    "name" text NOT NULL,
    "resource_id" text NOT NULL,
    "data_classification_config_id" text DEFAULT ''''::text NOT NULL,
    "setting" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "deleted" boolean DEFAULT false NOT NULL
);

ALTER SEQUENCE "public"."project_id_seq" OWNED BY "public"."project"."id";

ALTER TABLE ONLY "public"."project" ADD CONSTRAINT "project_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_project_unique_resource_id" ON ONLY "public"."project" ("resource_id");

CREATE SEQUENCE "public"."project_webhook_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."project_webhook" (
    "id" integer DEFAULT nextval(''public.project_webhook_id_seq''::regclass) NOT NULL,
    "type" text NOT NULL,
    "name" text NOT NULL,
    "url" text NOT NULL,
    "activity_list" _text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "project" text NOT NULL
);

ALTER SEQUENCE "public"."project_webhook_id_seq" OWNED BY "public"."project_webhook"."id";

ALTER TABLE ONLY "public"."project_webhook" ADD CONSTRAINT "project_webhook_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_project_webhook_project" ON ONLY "public"."project_webhook" ("project");

CREATE SEQUENCE "public"."query_history_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."query_history" (
    "id" bigint DEFAULT nextval(''public.query_history_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "project_id" text NOT NULL,
    "database" text NOT NULL,
    "statement" text NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL
);

ALTER SEQUENCE "public"."query_history_id_seq" OWNED BY "public"."query_history"."id";

ALTER TABLE ONLY "public"."query_history" ADD CONSTRAINT "query_history_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_query_history_creator_id_created_at_project_id" ON ONLY "public"."query_history" ("creator_id", "created_at", "project_id" DESC);

CREATE SEQUENCE "public"."release_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."release" (
    "id" bigint DEFAULT nextval(''public.release_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "project" text NOT NULL,
    "deleted" boolean DEFAULT false NOT NULL
);

ALTER SEQUENCE "public"."release_id_seq" OWNED BY "public"."release"."id";

ALTER TABLE ONLY "public"."release" ADD CONSTRAINT "release_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_release_project" ON ONLY "public"."release" ("project");

CREATE TABLE "public"."review_config" (
    "id" text NOT NULL,
    "name" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "enabled" boolean DEFAULT true NOT NULL
);

ALTER TABLE ONLY "public"."review_config" ADD CONSTRAINT "review_config_pkey" PRIMARY KEY ("id");

CREATE SEQUENCE "public"."revision_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."revision" (
    "id" bigint DEFAULT nextval(''public.revision_id_seq''::regclass) NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "deleter_id" integer,
    "deleted_at" timestamp with time zone,
    "version" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "instance" text NOT NULL,
    "db_name" text NOT NULL
);

ALTER SEQUENCE "public"."revision_id_seq" OWNED BY "public"."revision"."id";

ALTER TABLE ONLY "public"."revision" ADD CONSTRAINT "revision_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_revision_instance_db_name_version" ON ONLY "public"."revision" ("instance", "db_name", "version");

CREATE UNIQUE INDEX "idx_revision_unique_instance_db_name_version_deleted_at_null" ON ONLY "public"."revision" ("instance", "db_name", "version");

CREATE SEQUENCE "public"."risk_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."risk" (
    "id" bigint DEFAULT nextval(''public.risk_id_seq''::regclass) NOT NULL,
    "source" text NOT NULL,
    "level" bigint NOT NULL,
    "name" text NOT NULL,
    "active" boolean NOT NULL,
    "expression" jsonb NOT NULL
);

ALTER SEQUENCE "public"."risk_id_seq" OWNED BY "public"."risk"."id";

ALTER TABLE ONLY "public"."risk" ADD CONSTRAINT "risk_pkey" PRIMARY KEY ("id");

CREATE SEQUENCE "public"."role_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."role" (
    "id" bigint DEFAULT nextval(''public.role_id_seq''::regclass) NOT NULL,
    "resource_id" text NOT NULL,
    "name" text NOT NULL,
    "description" text NOT NULL,
    "permissions" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."role_id_seq" OWNED BY "public"."role"."id";

ALTER TABLE ONLY "public"."role" ADD CONSTRAINT "role_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_role_unique_resource_id" ON ONLY "public"."role" ("resource_id");

CREATE SEQUENCE "public"."setting_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."setting" (
    "id" integer DEFAULT nextval(''public.setting_id_seq''::regclass) NOT NULL,
    "name" text NOT NULL,
    "value" text NOT NULL
);

ALTER SEQUENCE "public"."setting_id_seq" OWNED BY "public"."setting"."id";

ALTER TABLE ONLY "public"."setting" ADD CONSTRAINT "setting_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_setting_unique_name" ON ONLY "public"."setting" ("name");

CREATE SEQUENCE "public"."sheet_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."sheet" (
    "id" integer DEFAULT nextval(''public.sheet_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "name" text NOT NULL,
    "sha256" bytea NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL
);

ALTER SEQUENCE "public"."sheet_id_seq" OWNED BY "public"."sheet"."id";

ALTER TABLE ONLY "public"."sheet" ADD CONSTRAINT "sheet_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_sheet_project" ON ONLY "public"."sheet" ("project");

CREATE TABLE "public"."sheet_blob" (
    "sha256" bytea NOT NULL,
    "content" text NOT NULL
);

ALTER TABLE ONLY "public"."sheet_blob" ADD CONSTRAINT "sheet_blob_pkey" PRIMARY KEY ("sha256");

CREATE SEQUENCE "public"."stage_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."stage" (
    "id" integer DEFAULT nextval(''public.stage_id_seq''::regclass) NOT NULL,
    "pipeline_id" integer NOT NULL,
    "environment" text NOT NULL
);

ALTER SEQUENCE "public"."stage_id_seq" OWNED BY "public"."stage"."id";

ALTER TABLE ONLY "public"."stage" ADD CONSTRAINT "stage_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_stage_pipeline_id" ON ONLY "public"."stage" ("pipeline_id");

CREATE SEQUENCE "public"."sync_history_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."sync_history" (
    "id" bigint DEFAULT nextval(''public.sync_history_id_seq''::regclass) NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "metadata" json DEFAULT ''{}''::json NOT NULL,
    "raw_dump" text DEFAULT ''''::text NOT NULL,
    "instance" text NOT NULL,
    "db_name" text NOT NULL
);

ALTER SEQUENCE "public"."sync_history_id_seq" OWNED BY "public"."sync_history"."id";

ALTER TABLE ONLY "public"."sync_history" ADD CONSTRAINT "sync_history_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_sync_history_instance_db_name_created_at" ON ONLY "public"."sync_history" ("instance", "db_name", "created_at");

CREATE SEQUENCE "public"."task_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."task" (
    "id" integer DEFAULT nextval(''public.task_id_seq''::regclass) NOT NULL,
    "pipeline_id" integer NOT NULL,
    "stage_id" integer NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "earliest_allowed_at" timestamp with time zone,
    "instance" text NOT NULL,
    "db_name" text
);

ALTER SEQUENCE "public"."task_id_seq" OWNED BY "public"."task"."id";

ALTER TABLE ONLY "public"."task" ADD CONSTRAINT "task_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_task_pipeline_id_stage_id" ON ONLY "public"."task" ("pipeline_id", "stage_id");

CREATE SEQUENCE "public"."task_run_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."task_run" (
    "id" integer DEFAULT nextval(''public.task_run_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "task_id" integer NOT NULL,
    "sheet_id" integer,
    "attempt" integer NOT NULL,
    "status" text NOT NULL,
    "code" integer DEFAULT 0 NOT NULL,
    "result" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "started_at" timestamp with time zone
);

ALTER SEQUENCE "public"."task_run_id_seq" OWNED BY "public"."task_run"."id";

ALTER TABLE ONLY "public"."task_run" ADD CONSTRAINT "task_run_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_task_run_task_id" ON ONLY "public"."task_run" ("task_id");

CREATE UNIQUE INDEX "uk_task_run_task_id_attempt" ON ONLY "public"."task_run" ("task_id", "attempt");

CREATE SEQUENCE "public"."task_run_log_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."task_run_log" (
    "id" bigint DEFAULT nextval(''public.task_run_log_id_seq''::regclass) NOT NULL,
    "task_run_id" integer NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."task_run_log_id_seq" OWNED BY "public"."task_run_log"."id";

ALTER TABLE ONLY "public"."task_run_log" ADD CONSTRAINT "task_run_log_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_task_run_log_task_run_id" ON ONLY "public"."task_run_log" ("task_run_id");

CREATE TABLE "public"."user_group" (
    "email" text NOT NULL,
    "name" text NOT NULL,
    "description" text DEFAULT ''''::text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER TABLE ONLY "public"."user_group" ADD CONSTRAINT "user_group_pkey" PRIMARY KEY ("email");

CREATE SEQUENCE "public"."worksheet_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."worksheet" (
    "id" integer DEFAULT nextval(''public.worksheet_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "name" text NOT NULL,
    "statement" text NOT NULL,
    "visibility" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL,
    "instance" text,
    "db_name" text
);

ALTER SEQUENCE "public"."worksheet_id_seq" OWNED BY "public"."worksheet"."id";

ALTER TABLE ONLY "public"."worksheet" ADD CONSTRAINT "worksheet_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_worksheet_creator_id_project" ON ONLY "public"."worksheet" ("creator_id", "project");

CREATE SEQUENCE "public"."worksheet_organizer_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."worksheet_organizer" (
    "id" integer DEFAULT nextval(''public.worksheet_organizer_id_seq''::regclass) NOT NULL,
    "worksheet_id" integer NOT NULL,
    "principal_id" integer NOT NULL,
    "starred" boolean DEFAULT false NOT NULL
);

ALTER SEQUENCE "public"."worksheet_organizer_id_seq" OWNED BY "public"."worksheet_organizer"."id";

ALTER TABLE ONLY "public"."worksheet_organizer" ADD CONSTRAINT "worksheet_organizer_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_worksheet_organizer_principal_id" ON ONLY "public"."worksheet_organizer" ("principal_id");

CREATE UNIQUE INDEX "idx_worksheet_organizer_unique_sheet_id_principal_id" ON ONLY "public"."worksheet_organizer" ("worksheet_id", "principal_id");

ALTER TABLE "public"."anomaly"
    ADD CONSTRAINT "anomaly_instance_db_name_fkey" FOREIGN KEY ("instance", "db_name")
    REFERENCES "public"."db" ("instance", "name");

ALTER TABLE "public"."changelist"
    ADD CONSTRAINT "changelist_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."changelist"
    ADD CONSTRAINT "changelist_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."changelog"
    ADD CONSTRAINT "changelog_instance_db_name_fkey" FOREIGN KEY ("instance", "db_name")
    REFERENCES "public"."db" ("instance", "name");

ALTER TABLE "public"."changelog"
    ADD CONSTRAINT "changelog_prev_sync_history_id_fkey" FOREIGN KEY ("prev_sync_history_id")
    REFERENCES "public"."sync_history" ("id");

ALTER TABLE "public"."changelog"
    ADD CONSTRAINT "changelog_sync_history_id_fkey" FOREIGN KEY ("sync_history_id")
    REFERENCES "public"."sync_history" ("id");

ALTER TABLE "public"."data_source"
    ADD CONSTRAINT "data_source_instance_fkey" FOREIGN KEY ("instance")
    REFERENCES "public"."instance" ("resource_id");

ALTER TABLE "public"."db"
    ADD CONSTRAINT "db_environment_fkey" FOREIGN KEY ("environment")
    REFERENCES "public"."environment" ("resource_id");

ALTER TABLE "public"."db"
    ADD CONSTRAINT "db_instance_fkey" FOREIGN KEY ("instance")
    REFERENCES "public"."instance" ("resource_id");

ALTER TABLE "public"."db"
    ADD CONSTRAINT "db_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."db_group"
    ADD CONSTRAINT "db_group_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."db_schema"
    ADD CONSTRAINT "db_schema_instance_db_name_fkey" FOREIGN KEY ("instance", "db_name")
    REFERENCES "public"."db" ("instance", "name");

ALTER TABLE "public"."instance"
    ADD CONSTRAINT "instance_environment_fkey" FOREIGN KEY ("environment")
    REFERENCES "public"."environment" ("resource_id");

ALTER TABLE "public"."issue"
    ADD CONSTRAINT "issue_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."issue"
    ADD CONSTRAINT "issue_pipeline_id_fkey" FOREIGN KEY ("pipeline_id")
    REFERENCES "public"."pipeline" ("id");

ALTER TABLE "public"."issue"
    ADD CONSTRAINT "issue_plan_id_fkey" FOREIGN KEY ("plan_id")
    REFERENCES "public"."plan" ("id");

ALTER TABLE "public"."issue"
    ADD CONSTRAINT "issue_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."issue_comment"
    ADD CONSTRAINT "issue_comment_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."issue_comment"
    ADD CONSTRAINT "issue_comment_issue_id_fkey" FOREIGN KEY ("issue_id")
    REFERENCES "public"."issue" ("id");

ALTER TABLE "public"."issue_subscriber"
    ADD CONSTRAINT "issue_subscriber_issue_id_fkey" FOREIGN KEY ("issue_id")
    REFERENCES "public"."issue" ("id");

ALTER TABLE "public"."issue_subscriber"
    ADD CONSTRAINT "issue_subscriber_subscriber_id_fkey" FOREIGN KEY ("subscriber_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."pipeline"
    ADD CONSTRAINT "pipeline_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."pipeline"
    ADD CONSTRAINT "pipeline_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."plan"
    ADD CONSTRAINT "plan_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."plan"
    ADD CONSTRAINT "plan_pipeline_id_fkey" FOREIGN KEY ("pipeline_id")
    REFERENCES "public"."pipeline" ("id");

ALTER TABLE "public"."plan"
    ADD CONSTRAINT "plan_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."plan_check_run"
    ADD CONSTRAINT "plan_check_run_plan_id_fkey" FOREIGN KEY ("plan_id")
    REFERENCES "public"."plan" ("id");

ALTER TABLE "public"."project_webhook"
    ADD CONSTRAINT "project_webhook_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."query_history"
    ADD CONSTRAINT "query_history_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."release"
    ADD CONSTRAINT "release_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."release"
    ADD CONSTRAINT "release_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."revision"
    ADD CONSTRAINT "revision_deleter_id_fkey" FOREIGN KEY ("deleter_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."revision"
    ADD CONSTRAINT "revision_instance_db_name_fkey" FOREIGN KEY ("instance", "db_name")
    REFERENCES "public"."db" ("instance", "name");

ALTER TABLE "public"."sheet"
    ADD CONSTRAINT "sheet_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."sheet"
    ADD CONSTRAINT "sheet_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."stage"
    ADD CONSTRAINT "stage_environment_fkey" FOREIGN KEY ("environment")
    REFERENCES "public"."environment" ("resource_id");

ALTER TABLE "public"."stage"
    ADD CONSTRAINT "stage_pipeline_id_fkey" FOREIGN KEY ("pipeline_id")
    REFERENCES "public"."pipeline" ("id");

ALTER TABLE "public"."sync_history"
    ADD CONSTRAINT "sync_history_instance_db_name_fkey" FOREIGN KEY ("instance", "db_name")
    REFERENCES "public"."db" ("instance", "name");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_instance_fkey" FOREIGN KEY ("instance")
    REFERENCES "public"."instance" ("resource_id");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_pipeline_id_fkey" FOREIGN KEY ("pipeline_id")
    REFERENCES "public"."pipeline" ("id");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_stage_id_fkey" FOREIGN KEY ("stage_id")
    REFERENCES "public"."stage" ("id");

ALTER TABLE "public"."task_run"
    ADD CONSTRAINT "task_run_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."task_run"
    ADD CONSTRAINT "task_run_sheet_id_fkey" FOREIGN KEY ("sheet_id")
    REFERENCES "public"."sheet" ("id");

ALTER TABLE "public"."task_run"
    ADD CONSTRAINT "task_run_task_id_fkey" FOREIGN KEY ("task_id")
    REFERENCES "public"."task" ("id");

ALTER TABLE "public"."task_run_log"
    ADD CONSTRAINT "task_run_log_task_run_id_fkey" FOREIGN KEY ("task_run_id")
    REFERENCES "public"."task_run" ("id");

ALTER TABLE "public"."worksheet"
    ADD CONSTRAINT "worksheet_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."worksheet"
    ADD CONSTRAINT "worksheet_project_fkey" FOREIGN KEY ("project")
    REFERENCES "public"."project" ("resource_id");

ALTER TABLE "public"."worksheet_organizer"
    ADD CONSTRAINT "worksheet_organizer_principal_id_fkey" FOREIGN KEY ("principal_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."worksheet_organizer"
    ADD CONSTRAINT "worksheet_organizer_worksheet_id_fkey" FOREIGN KEY ("worksheet_id")
    REFERENCES "public"."worksheet" ("id");

', '{}', 'bytebase-meta', 'bb') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (102, '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"98304","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"dataSize":"450560","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"dataSize":"122880","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample"}', '
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

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', '{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "department", "columns": [{"name": "dept_name", "classification": "1-1"}]}, {"name": "salary", "columns": [{"name": "amount"}]}, {"name": "employee", "columns": [{"name": "first_name", "semanticType": "be433ce5-72e7-4dcf-8b58-e77b52a18e81", "classification": "1-3"}, {"name": "last_name", "semanticType": "be433ce5-72e7-4dcf-8b58-e77b52a18e81", "classification": "1-3"}]}, {"name": "title", "columns": [{"name": "title", "classification": "2-1"}]}]}]}', 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (120, '{"name":"postgres","schemas":[{"name":"public","owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bb"}', '
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

SET default_tablespace = '''';

', '{}', 'bytebase-meta', 'postgres') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (107, '{"name":"hr_prod_5","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', '{}', 'prod-sample-instance', 'hr_prod_5') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (105, '{"name":"hr_prod_3","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', '{}', 'prod-sample-instance', 'hr_prod_3') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (106, '{"name":"hr_prod_4","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', '{}', 'prod-sample-instance', 'hr_prod_4') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (109, '{"name":"hr_prod_vcs","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"},{"name":"bugfix","position":7,"defaultExpression":"''''::text","type":"text"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL,
    "bugfix" text DEFAULT ''''::text NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', '{}', 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (182, '{"name":"postgres","schemas":[{"name":"public","owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample"}', '
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

SET default_tablespace = '''';

', '{}', 'prod-sample-instance', 'postgres') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (183, '{"name":"postgres","schemas":[{"name":"public","owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample"}', '
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

SET default_tablespace = '''';

', '{}', 'test-sample-instance', 'postgres') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, metadata, raw_dump, config, instance, db_name) VALUES (101, '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"98304","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"dataSize":"450560","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"dataSize":"122880","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample"}', '
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

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', '{}', 'test-sample-instance', 'hr_test') ON CONFLICT DO NOTHING;


--
-- Data for Name: environment; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.environment (id, name, "order", resource_id, deleted) VALUES (101, 'Test', 0, 'test', false) ON CONFLICT DO NOTHING;
INSERT INTO public.environment (id, name, "order", resource_id, deleted) VALUES (102, 'Prod', 1, 'prod', false) ON CONFLICT DO NOTHING;


--
-- Data for Name: export_archive; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: idp; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: instance; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.instance (id, environment, resource_id, metadata, deleted) VALUES (101, 'test', 'test-sample-instance', '{"roles": [{"name": "bbsample", "attribute": "Superuser Create role Create DB Replication Bypass RLS+"}], "title": "Test Sample Instance", "engine": "POSTGRES", "version": "16.0.2", "activation": true, "dataSources": [{"id": "admin", "host": "/tmp", "port": "8083", "type": "ADMIN", "username": "bbsample"}], "lastSyncTime": "2025-03-12T10:24:52.683555Z"}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, environment, resource_id, metadata, deleted) VALUES (103, 'prod', 'bytebase-meta', '{"roles": [{"name": "bb", "attribute": "Superuser Create role Create DB Replication Bypass RLS+"}], "title": "bytebase-meta", "engine": "POSTGRES", "version": "16.0.2", "activation": true, "dataSources": [{"id": "777072ed-539e-4cc2-a41e-6cc2917a7e7c", "host": "/tmp", "port": "8082", "type": "ADMIN", "username": "bb"}], "lastSyncTime": "2025-03-12T10:25:11.806883Z"}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, environment, resource_id, metadata, deleted) VALUES (102, 'prod', 'prod-sample-instance', '{"roles": [{"name": "bbsample", "attribute": "Superuser Create role Create DB Replication Bypass RLS+"}], "title": "Prod Sample Instance", "engine": "POSTGRES", "version": "16.0.2", "activation": true, "dataSources": [{"id": "admin", "host": "/tmp", "port": "8084", "type": "ADMIN", "username": "bbsample"}, {"id": "351173a6-f320-45c5-8d95-8e17abe08964", "host": "/tmp", "port": "8084", "type": "READ_ONLY", "username": "bbsample", "authenticationType": "PASSWORD", "obfuscatedPassword": "WyYTVD4="}, {"id": "e543ddfd-f633-4dbe-87f3-6b171a96e20a", "host": "/tmp", "port": "8084", "type": "READ_ONLY", "username": "bbsample", "authenticationType": "PASSWORD", "obfuscatedPassword": "WyYTVD4="}], "lastSyncTime": "2025-03-12T10:24:47.554607Z"}', false) ON CONFLICT DO NOTHING;


--
-- Data for Name: instance_change_history; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.instance_change_history (id, version) VALUES (197, '3.4.0') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (196, '3.3.8') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (198, '3.4.1') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (199, '3.4.2') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (200, '3.4.3') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (201, '3.4.4') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (202, '3.4.5') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (203, '3.4.6') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (204, '3.4.7') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (205, '3.4.8') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (206, '3.4.9') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (207, '3.4.10') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (208, '3.4.11') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (209, '3.4.12') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (210, '3.4.13') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (211, '3.4.14') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (212, '3.4.15') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (213, '3.4.16') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (214, '3.4.17') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (215, '3.4.18') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (216, '3.5.0') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (217, '3.5.1') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (218, '3.5.2') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (219, '3.5.3') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (220, '3.5.4') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (221, '3.5.5') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (222, '3.5.6') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (223, '3.5.7') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (224, '3.5.8') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (225, '3.5.9') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (226, '3.5.10') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (227, '3.5.11') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (228, '3.5.12') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (229, '3.5.13') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (230, '3.5.14') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (231, '3.5.15') ON CONFLICT DO NOTHING;


--
-- Data for Name: issue; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (104, 104, 104, 104, '[hr_prod] Alter schema @11-04 22:56 UTC+0800', 'CANCELED', 'bb.issue.database.general', '', '{"approval": {"approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/tester", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}]}, "title": "Tester -> Project Owner -> DBA", "creatorId": 101, "description": "Tester -> Project Owner -> DBA"}], "approvalFindingDone": true}}', '''04'':6 ''0800'':10 ''11'':5 ''22'':7 ''56'':8 ''alter'':3 ''hr'':1 ''prod'':2 ''schema'':4 ''utc'':9', '2023-11-04 07:57:12-07', '2023-11-04 07:59:27-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (102, 1, 102, 102, 'feat: add city to employee table', 'DONE', 'bb.issue.database.general', '', '{"approval": {"approvalFindingDone": true}}', '''20231101'':15 ''add'':6,17 ''alter'':4 ''by'':8 ''city'':7,18 ''ddl'':16 ''files'':10 ''hr'':1,12 ''prod'':2,11,13 ''schema'':5 ''sql'':19 ''vcs'':3,9,14', '2023-11-03 09:07:13-07', '2023-12-14 05:55:47-08', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (106, 1, 106, 106, 'feat: add phone to employee table', 'OPEN', 'bb.issue.database.general', '', '{"approval": {"riskLevel": "HIGH", "approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_OWNER"}]}]}, "title": "Project Owner -> Workspace Admin", "creatorId": 101, "description": "Project Owner -> Workspace Admin"}], "approvalFindingDone": true}}', '''add'':2 ''employee'':5 ''feat'':1 ''phone'':3 ''table'':6 ''to'':4', '2024-04-10 01:18:10-07', '2024-04-10 01:18:10-07', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (108, 101, 108, 108, 'Establish "hr_prod_vcs" baseline', 'DONE', 'bb.issue.database.general', '', '{"approval": {"approvalFindingDone": true}}', '''baseline'':5 ''establish'':1 ''hr'':2 ''prod'':3 ''vcs'':4', '2025-01-15 20:17:55-08', '2025-01-15 20:19:11-08', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (101, 101, 101, 101, '👉👉👉 [START HERE] Add email column to Employee table', 'OPEN', 'bb.issue.database.general', 'A sample issue to showcase how to review database schema change.

				Click "Approve" button to apply the schema update.', '{"labels": ["2.17.0", "bug"], "approval": {"riskLevel": "HIGH", "approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_OWNER"}]}]}, "title": "Project Owner -> Workspace Admin", "creatorId": 101, "description": "Project Owner -> Workspace Admin"}], "approvalFindingDone": true}}', '''a'':9 ''add'':3 ''apply'':24 ''approve'':21 ''button'':22 ''change'':19 ''click'':20 ''column'':5 ''database'':17 ''email'':4 ''employee'':7 ''here'':2 ''how'':14 ''issue'':11 ''review'':16 ''sample'':10 ''schema'':18,26 ''showcase'':13 ''start'':1 ''table'':8 ''the'':25 ''to'':6,12,15,23 ''update'':27', '2023-11-03 08:46:31-07', '2024-05-16 09:42:50-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (105, 104, 105, 105, 'Add performance table', 'OPEN', 'bb.issue.database.general', '', '{"labels": ["feature"], "approval": {"approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}]}, "title": "Project Owner -> DBA", "creatorId": 1, "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}], "approvalFindingDone": true}}', '''add'':1 ''performance'':2 ''table'':3', '2023-11-04 08:05:35-07', '2024-05-16 09:43:01-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (110, 101, 110, 110, 'Establish "hr_test" baseline', 'DONE', 'bb.issue.database.general', '', '{"approval": {"approvalFindingDone": true}}', '''baseline'':4 ''establish'':1 ''hr'':2 ''test'':3', '2025-01-15 20:21:52-08', '2025-01-15 20:21:56-08', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (107, 101, 107, 107, 'Update employee gender to M', 'OPEN', 'bb.issue.database.general', '', '{"approval": {"approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}]}, "title": "Project Owner", "creatorId": 1, "description": "The system defines the approval process and only needs the project Owner o approve it."}], "approvalFindingDone": true}}', '''employee'':2 ''gender'':3 ''m'':5 ''to'':4 ''update'':1', '2024-05-16 09:58:06-07', '2024-05-16 09:58:07-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (109, 101, 109, 109, 'Establish "hr_prod" baseline', 'DONE', 'bb.issue.database.general', '', '{"approval": {"approvalFindingDone": true}}', '''baseline'':4 ''establish'':1 ''hr'':2 ''prod'':3', '2025-01-15 20:19:35-08', '2025-01-15 20:19:38-08', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, creator_id, plan_id, pipeline_id, name, status, type, description, payload, ts_vector, created_at, updated_at, project) VALUES (103, 106, 103, 103, 'Add Investor Relation department', 'DONE', 'bb.issue.database.general', '', '{"approval": {"approvalFindingDone": true}}', '''add'':1 ''department'':4 ''investor'':2 ''relation'':3', '2023-11-03 10:28:39-07', '2025-01-22 22:26:01-08', 'batch-project') ON CONFLICT DO NOTHING;


--
-- Data for Name: issue_comment; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (105, 104, 104, '{"issueUpdate": {"toStatus": "CANCELED", "fromStatus": "OPEN"}}', '2023-11-04 07:59:27-07', '2023-11-04 07:59:27-07') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (106, 101, 102, '{"taskUpdate": {"tasks": ["projects/gitops-project/rollouts/102/stages/103/tasks/103"], "toStatus": "PENDING"}}', '2023-12-14 05:55:44-08', '2023-12-14 05:55:44-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (107, 1, 102, '{"taskUpdate": {"tasks": ["projects/gitops-project/rollouts/102/stages/103/tasks/103"], "toStatus": "DONE"}}', '2023-12-14 05:55:47-08', '2023-12-14 05:55:47-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (108, 1, 102, '{"stageEnd": {"stage": "projects/gitops-project/rollouts/102/stages/103"}}', '2023-12-14 05:55:47-08', '2023-12-14 05:55:47-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (109, 1, 102, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}', '2023-12-14 05:55:47-08', '2023-12-14 05:55:47-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (110, 101, 101, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/101/stages/101/tasks/101"], "toSheet": "projects/project-sample/sheets/129", "fromSheet": "projects/project-sample/sheets/102"}}', '2024-04-10 01:01:57-07', '2024-04-10 01:01:57-07') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (111, 101, 101, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/101/stages/102/tasks/102"], "toSheet": "projects/project-sample/sheets/129", "fromSheet": "projects/project-sample/sheets/103"}}', '2024-04-10 01:01:57-07', '2024-04-10 01:01:57-07') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (112, 101, 101, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/101/stages/101/tasks/101"], "toSheet": "projects/project-sample/sheets/130", "fromSheet": "projects/project-sample/sheets/129"}}', '2024-04-10 01:02:37-07', '2024-04-10 01:02:37-07') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (113, 101, 101, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/101/stages/102/tasks/102"], "toSheet": "projects/project-sample/sheets/130", "fromSheet": "projects/project-sample/sheets/129"}}', '2024-04-10 01:02:37-07', '2024-04-10 01:02:37-07') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (114, 101, 108, '{"taskUpdate": {"tasks": ["projects/gitops-project/rollouts/108/stages/111/tasks/114"], "toStatus": "PENDING"}}', '2025-01-15 20:19:11-08', '2025-01-15 20:19:11-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (115, 1, 108, '{"taskUpdate": {"tasks": ["projects/gitops-project/rollouts/108/stages/111/tasks/114"], "toStatus": "DONE"}}', '2025-01-15 20:19:11-08', '2025-01-15 20:19:11-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (116, 1, 108, '{"stageEnd": {"stage": "projects/gitops-project/rollouts/108/stages/111"}}', '2025-01-15 20:19:11-08', '2025-01-15 20:19:11-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (117, 1, 108, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}', '2025-01-15 20:19:11-08', '2025-01-15 20:19:11-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (118, 101, 109, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/109/stages/112/tasks/115"], "toStatus": "PENDING"}}', '2025-01-15 20:19:38-08', '2025-01-15 20:19:38-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (119, 1, 109, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/109/stages/112/tasks/115"], "toStatus": "DONE"}}', '2025-01-15 20:19:38-08', '2025-01-15 20:19:38-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (120, 1, 109, '{"stageEnd": {"stage": "projects/project-sample/rollouts/109/stages/112"}}', '2025-01-15 20:19:38-08', '2025-01-15 20:19:38-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (121, 1, 109, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}', '2025-01-15 20:19:38-08', '2025-01-15 20:19:38-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (122, 101, 110, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/110/stages/113/tasks/116"], "toStatus": "PENDING"}}', '2025-01-15 20:21:56-08', '2025-01-15 20:21:56-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (123, 1, 110, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/110/stages/113/tasks/116"], "toStatus": "DONE"}}', '2025-01-15 20:21:56-08', '2025-01-15 20:21:56-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (124, 1, 110, '{"stageEnd": {"stage": "projects/project-sample/rollouts/110/stages/113"}}', '2025-01-15 20:21:56-08', '2025-01-15 20:21:56-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (125, 1, 110, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}', '2025-01-15 20:21:56-08', '2025-01-15 20:21:56-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (126, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/104/tasks/104"], "toStatus": "PENDING"}}', '2025-01-22 22:25:52-08', '2025-01-22 22:25:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (127, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/104/tasks/105"], "toStatus": "PENDING"}}', '2025-01-22 22:25:52-08', '2025-01-22 22:25:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (128, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/104/tasks/105"], "toStatus": "DONE"}}', '2025-01-22 22:25:52-08', '2025-01-22 22:25:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (129, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/104/tasks/104"], "toStatus": "DONE"}}', '2025-01-22 22:25:52-08', '2025-01-22 22:25:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (130, 1, 103, '{"stageEnd": {"stage": "projects/batch-project/rollouts/103/stages/104"}}', '2025-01-22 22:25:52-08', '2025-01-22 22:25:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (131, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/105/tasks/106"], "toStatus": "PENDING"}}', '2025-01-22 22:25:57-08', '2025-01-22 22:25:57-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (132, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/105/tasks/107"], "toStatus": "PENDING"}}', '2025-01-22 22:25:57-08', '2025-01-22 22:25:57-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (133, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/105/tasks/106"], "toStatus": "DONE"}}', '2025-01-22 22:25:57-08', '2025-01-22 22:25:57-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (134, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/105/tasks/107"], "toStatus": "DONE"}}', '2025-01-22 22:25:57-08', '2025-01-22 22:25:57-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (135, 1, 103, '{"stageEnd": {"stage": "projects/batch-project/rollouts/103/stages/105"}}', '2025-01-22 22:25:57-08', '2025-01-22 22:25:57-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (136, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/106/tasks/108"], "toStatus": "PENDING"}}', '2025-01-22 22:26:02-08', '2025-01-22 22:26:02-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (137, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/106/tasks/109"], "toStatus": "PENDING"}}', '2025-01-22 22:26:02-08', '2025-01-22 22:26:02-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (138, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/106/tasks/109"], "toStatus": "DONE"}}', '2025-01-22 22:26:02-08', '2025-01-22 22:26:02-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (139, 1, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/106/tasks/108"], "toStatus": "DONE"}}', '2025-01-22 22:26:02-08', '2025-01-22 22:26:02-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (140, 1, 103, '{"stageEnd": {"stage": "projects/batch-project/rollouts/103/stages/106"}}', '2025-01-22 22:26:02-08', '2025-01-22 22:26:02-08') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, creator_id, issue_id, payload, created_at, updated_at) VALUES (141, 1, 103, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}', '2025-01-22 22:26:02-08', '2025-01-22 22:26:02-08') ON CONFLICT DO NOTHING;


--
-- Data for Name: issue_subscriber; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: pipeline; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (101, 101, 'Rollout Pipeline', '2023-11-03 08:46:31-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (102, 1, 'Rollout Pipeline', '2023-11-03 09:07:13-07', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (103, 106, 'Rollout Pipeline', '2023-11-03 10:28:39-07', 'batch-project') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (104, 104, 'Rollout Pipeline', '2023-11-04 07:57:12-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (105, 104, 'Rollout Pipeline', '2023-11-04 08:05:35-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (106, 1, 'Rollout Pipeline', '2024-04-10 01:18:10-07', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (107, 101, 'Rollout Pipeline', '2024-05-16 09:58:06-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (108, 101, 'Rollout Pipeline', '2025-01-15 20:17:55-08', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (109, 101, 'Rollout Pipeline', '2025-01-15 20:19:35-08', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, creator_id, name, created_at, project) VALUES (110, 101, 'Rollout Pipeline', '2025-01-15 20:21:52-08', 'project-sample') ON CONFLICT DO NOTHING;


--
-- Data for Name: plan; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (102, 1, 102, 'feat: add city to employee table', '', '{"steps": [{"specs": [{"changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/gitops-project/sheets/104", "target": "instances/prod-sample-instance/databases/hr_prod_vcs", "schemaVersion": "1000-ddl"}}]}], "vcsSource": {"vcsType": "GITHUB", "vcsConnector": "projects/gitops-project/vcsConnectors/hr-sample", "pullRequestUrl": "https://github.com/s-bytebase/hr-sample/pull/18"}}', '2023-11-03 09:07:13-07', '2023-11-03 09:07:13-07', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (104, 104, 104, '', '', '{"steps": [{"specs": [{"id": "96967c30-ee17-468e-8368-6366ccc83c52", "changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/107", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}]}', '2023-11-04 07:57:12-07', '2023-11-04 07:57:12-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (105, 104, 105, '', '', '{"steps": [{"specs": [{"id": "9227f0c7-fa7d-44f3-9282-a32da230e2e4", "changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/108", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}]}', '2023-11-04 08:05:35-07', '2023-11-04 08:05:35-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (101, 101, 101, 'Onboarding sample plan for adding email column to Employee table', '', '{"steps": [{"specs": [{"changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/130", "target": "instances/test-sample-instance/databases/hr_test"}}]}, {"specs": [{"changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/130", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}]}', '2023-11-03 08:46:31-07', '2024-04-10 01:02:37-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (106, 1, 106, 'feat: add phone to employee table', '', '{"steps": [{"specs": [{"id": "e4010ea4-dd1e-441a-9ea2-90f467ed8506", "changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/gitops-project/sheets/131", "target": "instances/prod-sample-instance/databases/hr_prod_vcs", "schemaVersion": "1001"}}]}], "vcsSource": {"vcsType": "GITHUB", "vcsConnector": "projects/gitops-project/vcsConnectors/hr-sample", "pullRequestUrl": "https://github.com/s-bytebase/hr-sample/pull/17"}}', '2024-04-10 01:18:10-07', '2024-04-10 01:18:10-07', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (107, 101, 107, '', '', '{"steps": [{"specs": [{"id": "0992ef9b-3d08-4745-ab40-ff74d34208a8", "changeDatabaseConfig": {"type": "DATA", "sheet": "projects/project-sample/sheets/132", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}]}', '2024-05-16 09:58:06-07', '2024-05-16 09:58:06-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (103, 106, 103, '', '', '{"steps": [{"specs": [{"id": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "changeDatabaseConfig": {"type": "DATA", "sheet": "projects/batch-project/sheets/106", "target": "projects/batch-project/databaseGroups/all-databases"}}]}]}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:39-07', 'batch-project') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (108, 101, 108, '', '', '{"steps": [{"specs": [{"id": "ff8ecf1c-f037-4544-971c-c3f4c8ff5889", "specReleaseSource": {}, "changeDatabaseConfig": {"type": "BASELINE", "sheet": "projects/gitops-project/sheets/133", "target": "instances/prod-sample-instance/databases/hr_prod_vcs"}}]}], "deploymentSnapshot": {"deploymentConfigSnapshot": {"name": "gitops-project/deploymentConfigs/default", "deploymentConfig": {"schedule": {"deployments": [{"id": "0", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["test"], "operator": "IN"}]}}, "title": "Test Stage"}, {"id": "1", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["prod"], "operator": "IN"}]}}, "title": "Prod Stage"}]}}}}}', '2025-01-15 20:17:55-08', '2025-01-15 20:17:55-08', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (109, 101, 109, '', '', '{"steps": [{"specs": [{"id": "231a929d-bb89-4845-8b7c-6e4870116d32", "specReleaseSource": {}, "changeDatabaseConfig": {"type": "BASELINE", "sheet": "projects/project-sample/sheets/134", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}], "deploymentSnapshot": {"deploymentConfigSnapshot": {"name": "project-sample/deploymentConfigs/default", "deploymentConfig": {"schedule": {"deployments": [{"id": "0", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["test"], "operator": "IN"}]}}, "title": "Test Stage"}, {"id": "1", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["prod"], "operator": "IN"}]}}, "title": "Prod Stage"}]}}}}}', '2025-01-15 20:19:35-08', '2025-01-15 20:19:35-08', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, creator_id, pipeline_id, name, description, config, created_at, updated_at, project) VALUES (110, 101, 110, '', '', '{"steps": [{"specs": [{"id": "913aa19f-18e6-42c5-b6e7-2fbb358cffee", "specReleaseSource": {}, "changeDatabaseConfig": {"type": "BASELINE", "sheet": "projects/project-sample/sheets/135", "target": "instances/test-sample-instance/databases/hr_test"}}]}], "deploymentSnapshot": {"deploymentConfigSnapshot": {"name": "project-sample/deploymentConfigs/default", "deploymentConfig": {"schedule": {"deployments": [{"id": "0", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["test"], "operator": "IN"}]}}, "title": "Test Stage"}, {"id": "1", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["prod"], "operator": "IN"}]}}, "title": "Prod Stage"}]}}}}}', '2025-01-15 20:21:52-08', '2025-01-15 20:21:52-08', 'project-sample') ON CONFLICT DO NOTHING;


--
-- Data for Name: plan_check_run; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (135, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_6", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:47-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (101, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 102, "instanceId": "test-sample-instance", "databaseName": "hr_test"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}', '2023-11-03 08:46:31-07', '2023-11-03 08:46:31-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (108, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 103, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}', '2023-11-03 08:46:31-07', '2023-11-03 08:46:31-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (105, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 103, "instanceId": "prod-sample-instance", "databaseName": "hr_prod"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}', '2023-11-03 08:46:31-07', '2023-11-03 08:46:31-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (104, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 102, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_test", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}', '2023-11-03 08:46:31-07', '2023-11-03 08:46:31-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (155, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 130, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2024-04-10 01:02:37-07', '2024-04-10 01:02:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (103, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 102, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2023-11-03 08:46:31-07', '2023-11-03 08:46:31-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (107, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 103, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"email\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}', '2023-11-03 08:46:31-07', '2023-11-03 08:46:31-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (109, 102, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 104, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_vcs"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_vcs\""}]}', '{}', '2023-11-03 09:07:13-07', '2023-11-03 09:07:13-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (111, 102, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 104, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"city\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}', '2023-11-03 09:07:13-07', '2023-11-03 09:07:13-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (112, 102, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 104, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod_vcs", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}', '2023-11-03 09:07:13-07', '2023-11-03 09:07:13-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (117, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_4"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_4\""}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:39-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (113, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_1"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_1\""}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:39-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (119, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_4", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:39-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (121, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_2"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_2\""}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:39-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (125, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_5"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_5\""}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (124, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_2", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (115, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_1", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:39-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (116, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_1", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:39-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (120, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_4", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:39-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (132, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_3", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (123, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_2", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (131, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_3", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (129, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_3"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_3\""}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (127, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_5", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (128, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_5", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (136, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_6", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:47-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (133, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_6"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_6\""}]}', '{}', '2023-11-03 10:28:39-07', '2023-11-03 10:28:47-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (139, 104, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 107, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2023-11-04 07:57:12-07', '2023-11-04 07:57:12-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (140, 104, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 107, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}', '2023-11-04 07:57:12-07', '2023-11-04 07:57:12-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (137, 104, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 107, "instanceId": "prod-sample-instance", "databaseName": "hr_prod"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}', '2023-11-04 07:57:12-07', '2023-11-04 07:57:12-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (144, 105, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 108, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"statementTypes": ["CREATE_TABLE", "COMMENT"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "performance"}, {"name": "performance"}, {"name": "performance"}]}]}]}}}]}', '{}', '2023-11-04 08:05:35-07', '2023-11-04 08:05:35-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (141, 105, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 108, "instanceId": "prod-sample-instance", "databaseName": "hr_prod"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}', '2023-11-04 08:05:35-07', '2023-11-04 08:05:35-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (143, 105, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 108, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2023-11-04 08:05:35-07', '2023-11-04 08:05:35-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (152, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 129, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}', '2024-04-10 01:01:57-07', '2024-04-10 01:01:57-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (145, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 129, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}', '2024-04-10 01:01:57-07', '2024-04-10 01:01:57-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (149, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 129, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}', '2024-04-10 01:01:57-07', '2024-04-10 01:01:57-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (148, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 129, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_test", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}', '2024-04-10 01:01:57-07', '2024-04-10 01:01:57-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (147, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 129, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}', '2024-04-10 01:01:57-07', '2024-04-10 01:01:57-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (151, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 129, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"email\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}', '2024-04-10 01:01:57-07', '2024-04-10 01:01:57-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (160, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 130, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}', '2024-04-10 01:02:37-07', '2024-04-10 01:02:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (157, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 130, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}', '2024-04-10 01:02:37-07', '2024-04-10 01:02:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (153, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 130, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}', '2024-04-10 01:02:37-07', '2024-04-10 01:02:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (156, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 130, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_test", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}', '2024-04-10 01:02:37-07', '2024-04-10 01:02:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (159, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 130, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"email\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}', '2024-04-10 01:02:37-07', '2024-04-10 01:02:42-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (161, 106, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 131, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_vcs\""}]}', '{}', '2024-04-10 01:18:10-07', '2024-04-10 01:18:10-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (163, 106, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 131, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"phone\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}', '2024-04-10 01:18:10-07', '2024-04-10 01:18:10-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (164, 106, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 131, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod_vcs", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}', '2024-04-10 01:18:10-07', '2024-04-10 01:18:10-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (165, 107, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 132, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}', '2024-05-16 09:58:06-07', '2024-05-16 09:58:06-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (168, 107, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 132, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["UPDATE"], "changedResources": {}}}]}', '{}', '2024-05-16 09:58:06-07', '2024-05-16 09:58:06-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (167, 107, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 132, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DML"}', '{"results": [{"title": "statement.affected-row-limit", "status": "WARNING", "content": "The statement \"UPDATE employee\nSET\n  gender = ''M''\" affected 1000 rows (estimated). The count exceeds 100.", "sqlReviewReport": {"code": 209, "line": 3}}]}', '{}', '2024-05-16 09:58:06-07', '2024-05-16 09:58:06-07') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (169, 108, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 133, "instanceId": "prod-sample-instance", "databaseName": "hr_prod_vcs"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_vcs\""}]}', '{}', '2025-01-15 20:17:55-08', '2025-01-15 20:17:54-08') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (170, 109, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 134, "instanceId": "prod-sample-instance", "databaseName": "hr_prod"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}', '2025-01-15 20:19:35-08', '2025-01-15 20:19:34-08') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, plan_id, status, type, config, result, payload, created_at, updated_at) VALUES (171, 110, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 135, "instanceId": "test-sample-instance", "databaseName": "hr_test"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}', '2025-01-15 20:21:52-08', '2025-01-15 20:21:52-08') ON CONFLICT DO NOTHING;


--
-- Data for Name: policy; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (106, 'bb.policy.masking-rule', '{"rules": [{"id": "9dda9145-895e-451a-99d8-16254c4eb287", "condition": {"expression": "environment_id == \"test\""}, "maskingLevel": "NONE"}, {"id": "d188a226-5ed6-45cc-82e3-baa890a87962", "condition": {"expression": "classification_level in [\"1\"]"}, "maskingLevel": "NONE"}, {"id": "76356d81-6231-4128-9be7-2c549fc505f5", "condition": {"expression": "classification_level in [\"2\", \"3\"]"}, "semanticType": "bb.default-partial"}, {"id": "1ddc47c9-6ab6-4760-accd-947bc1a5f155", "condition": {"expression": "classification_level in [\"4\"]"}, "semanticType": "bb.default"}]}', 'WORKSPACE', false, '2025-02-24 22:35:21.127989-08', '', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (134, 'bb.policy.data-source-query', '{"adminDataSourceRestriction": "FALLBACK"}', 'PROJECT', false, '2025-02-24 22:35:21.127989-08', 'projects/project-sample', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (114, 'bb.policy.iam', '{"bindings": [{"role": "roles/projectDeveloper", "members": ["users/104", "users/107", "users/106", "users/105"], "condition": {"title": "Developer"}}, {"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/projectReleaser", "members": ["users/102"], "condition": {"title": "Releaser"}}, {"role": "roles/tester", "members": ["users/108"], "condition": {"title": "Tester"}}]}', 'PROJECT', false, '2025-02-24 22:35:21.127989-08', 'projects/project-sample', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (115, 'bb.policy.iam', '{"bindings": [{"role": "roles/projectDeveloper", "members": ["users/107", "users/106"], "condition": {"title": "Developer"}}, {"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/projectOwner", "members": ["users/104", "users/105"], "condition": {"title": "Owner"}}]}', 'PROJECT', false, '2025-02-24 22:35:21.127989-08', 'projects/gitops-project', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (116, 'bb.policy.iam', '{"bindings": [{"role": "roles/projectDeveloper", "members": ["users/106", "users/107"], "condition": {"title": "Developer"}}, {"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/projectOwner", "members": ["users/102", "users/103"], "condition": {"title": "Owner"}}, {"role": "roles/sqlEditorUser", "members": ["users/104", "users/105"], "condition": {"title": "Querier All"}}]}', 'PROJECT', false, '2025-02-24 22:35:21.127989-08', 'projects/batch-project', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (105, 'bb.policy.environment-tier', '{"environmentTier": "PROTECTED"}', 'ENVIRONMENT', true, '2025-02-24 22:35:21.127989-08', 'environments/prod', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (104, 'bb.policy.rollout', '{}', 'ENVIRONMENT', false, '2025-02-24 22:35:21.127989-08', 'environments/prod', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (117, 'bb.policy.iam', '{"bindings": [{"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/sqlEditorUser", "members": ["users/104"], "condition": {"title": "Project Querier All"}}]}', 'PROJECT', false, '2025-02-24 22:35:21.127989-08', 'projects/metadb-project', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (103, 'bb.policy.rollout', '{}', 'ENVIRONMENT', false, '2025-02-24 22:35:21.127989-08', 'environments/test', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (108, 'bb.policy.disable-copy-data', '{"active": false}', 'ENVIRONMENT', false, '2025-02-24 22:35:21.127989-08', 'environments/test', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (109, 'bb.policy.disable-copy-data', '{"active": false}', 'ENVIRONMENT', false, '2025-02-24 22:35:21.127989-08', 'environments/prod', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (118, 'bb.policy.tag', '{"tags": {"bb.tag.review_config": "reviewConfigs/prod"}}', 'ENVIRONMENT', true, '2025-02-24 22:35:21.127989-08', 'environments/prod', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (122, 'bb.policy.data-source-query', '{"disallowDdl": true, "disallowDml": true, "adminDataSourceRestriction": "FALLBACK"}', 'ENVIRONMENT', false, '2025-02-24 22:35:21.127989-08', 'environments/test', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (123, 'bb.policy.data-source-query', '{"disallowDdl": true, "disallowDml": true, "adminDataSourceRestriction": "FALLBACK"}', 'ENVIRONMENT', false, '2025-02-24 22:35:21.127989-08', 'environments/prod', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (174, 'bb.policy.query-data', '{"timeout": "600s"}', 'WORKSPACE', false, '2025-03-12 03:10:55.082145-07', '', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, type, payload, resource_type, inherit_from_parent, updated_at, resource, enforce) VALUES (119, 'bb.policy.iam', '{"bindings": [{"role": "roles/workspaceDBA", "members": ["users/102", "users/103", "users/109"]}, {"role": "roles/workspaceAdmin", "members": ["users/101", "users/1"]}, {"role": "roles/workspaceMember", "members": ["allUsers"]}]}', 'WORKSPACE', false, '2025-03-12 03:25:30.021918-07', '', true) ON CONFLICT DO NOTHING;


--
-- Data for Name: principal; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (1, 'SYSTEM_BOT', 'Bytebase', 'support@bytebase.com', '', '', '{}', '{}', '2025-02-07 22:15:49-08', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (2, 'SYSTEM_BOT', 'All Users', 'allUsers', '', '', '{}', '{}', '2025-02-07 22:15:49-08', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (102, 'END_USER', 'dba1', 'dba1@example.com', '$2a$10$mjuC.ej22zhysY3ylsR00eqFGVPxctD4RMZN7mio7GjhTFg5o6nPG', '', '{}', '{}', '2023-11-03 09:23:50-07', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (103, 'END_USER', 'dba2', 'dba2@example.com', '$2a$10$UIKJY.ziyCuB0fIG.AkuBOlcPoYtzvVZZfm4Uh3OrgbF0VLTneUbC', '', '{}', '{}', '2023-11-03 09:23:51-07', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (105, 'END_USER', 'dev2', 'dev2@example.com', '$2a$10$Fst2F8T3GCRKsLoAh5937.qkFVwsbygmu2FKriu0B1nQave1VKXQC', '', '{}', '{}', '2023-11-03 09:23:51-07', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (106, 'END_USER', 'dev3', 'dev3@example.com', '$2a$10$b6X5Pk/Ffe7YtDTrJcqtKuP.e9OmdH3Kq9i/WaTUO9225Pud6yd/6', '', '{}', '{}', '2023-11-03 09:23:51-07', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (107, 'END_USER', 'dev4', 'dev4@example.com', '$2a$10$ikN0OjIzqoCuOtR21FRtTuTS5LenyJSdonyL.VOphI9LDTgOQ6NcC', '', '{}', '{}', '2023-11-03 09:23:51-07', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (108, 'END_USER', 'qa1', 'qa1@example.com', '$2a$10$tgPwB2JdZlyu2MD/W.IxluFMI8bM9IPgYSQYaQEIBYT0SO23QM5Iu', '', '{}', '{}', '2023-11-04 07:48:30-07', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (104, 'END_USER', 'dev1', 'dev1@example.com', '$2a$10$hX4vTGH7Id6v9BWhHHtW9uHT.M/ANZ25owa5J9m1tSS5qzlSCkjSu', '', '{}', '{"lastLoginTime": "2024-09-20T08:16:29.553490Z"}', '2023-11-03 09:23:51-07', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (109, 'SERVICE_ACCOUNT', 'ci', 'ci@service.bytebase.com', '$2a$10$LLSrQ6pPSnqIqml/PUh3G.WjrLUMod2l8hWbSKa4qsVQ7bw9ZTNc.', '', '{}', '{"lastChangePasswordTime": "2025-01-23T16:49:28.865322699Z"}', '2024-04-10 01:06:18-07', false) ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, type, name, email, password_hash, phone, mfa_config, profile, created_at, deleted) VALUES (101, 'END_USER', 'Demo', 'demo@example.com', '$2a$10$aKjyVRxwbzmNToxYLXgTn.cQZX9x8KI1LLu5U69zzn5wcaoagoBLG', '', '{}', '{"lastLoginTime": "2025-03-12T10:20:19.115719Z"}', '2023-11-03 08:46:31-07', false) ON CONFLICT DO NOTHING;


--
-- Data for Name: project; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.project (id, name, resource_id, data_classification_config_id, setting, deleted) VALUES (1, 'Default', 'default', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"postgresDatabaseTenantMode": true}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, name, resource_id, data_classification_config_id, setting, deleted) VALUES (102, 'GitOps Project', 'gitops-project', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"autoResolveIssue": true, "allowModifyStatement": true, "postgresDatabaseTenantMode": true}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, name, resource_id, data_classification_config_id, setting, deleted) VALUES (103, 'Batch Project', 'batch-project', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"autoResolveIssue": true, "allowModifyStatement": true, "postgresDatabaseTenantMode": true}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, name, resource_id, data_classification_config_id, setting, deleted) VALUES (104, 'MetaDB Project', 'metadb-project', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"autoResolveIssue": true, "allowModifyStatement": true, "postgresDatabaseTenantMode": true}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, name, resource_id, data_classification_config_id, setting, deleted) VALUES (101, 'Basic Project', 'project-sample', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"issueLabels": [{"color": "#4f46e5", "value": "2.17.0"}, {"color": "#E55146", "value": "bug"}, {"color": "#E5B546", "value": "feature"}], "autoResolveIssue": true, "allowSelfApproval": true, "allowModifyStatement": true, "postgresDatabaseTenantMode": true}', false) ON CONFLICT DO NOTHING;


--
-- Data for Name: project_webhook; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: query_history; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (101, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM salary;', 'QUERY', '{}', '2023-11-03 09:42:14-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (102, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM salary;', 'QUERY', '{}', '2023-11-03 09:44:28-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (103, 104, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_1', 'SELECT * FROM salary', 'QUERY', '{}', '2023-11-03 09:44:58-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (104, 104, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_1', 'SELECT * FROM department', 'QUERY', '{}', '2023-11-03 09:47:19-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (105, 104, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_1', 'SELECT * FROM department', 'QUERY', '{}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (106, 104, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_2', 'SELECT * FROM department', 'QUERY', '{}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (107, 104, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_3', 'SELECT * FROM department', 'QUERY', '{}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (108, 104, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_4', 'SELECT * FROM department', 'QUERY', '{}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (109, 104, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_5', 'SELECT * FROM department', 'QUERY', '{}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (110, 104, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_6', 'SELECT * FROM department', 'QUERY', '{}', '2023-11-03 09:47:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (111, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{}', '2023-11-03 10:21:22-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (112, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM salary;', 'QUERY', '{}', '2023-11-03 10:22:33-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (113, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{}', '2023-11-03 10:22:59-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (114, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM department;', 'QUERY', '{}', '2023-11-03 10:26:34-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (115, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{}', '2023-11-20 23:45:43-08') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (116, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{}', '2023-11-20 23:45:53-08') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (117, 101, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
  project.resource_id,
  count(*)
FROM
  issue
  LEFT JOIN project ON issue.project_id = project.id
WHERE
  NOT EXISTS (
    SELECT
      1
    FROM
      task,
      task_run
    WHERE
      task.pipeline_id = issue.pipeline_id
      AND task.id = task_run.task_id
      AND task_run.status != ''DONE''
  )
  AND issue.status = ''DONE''
GROUP BY
  project.resource_id;', 'QUERY', '{"error": "", "duration": "0.002s"}', '2024-04-10 06:59:38-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (118, 101, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT project.resource_id, count(*)
FROM issue
LEFT JOIN project ON issue.project_id = project.id
WHERE EXISTS (
        SELECT 1 FROM activity, principal, member
        WHERE TO_TIMESTAMP(activity.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''
        AND activity.type = ''bb.issue.comment.create''
        AND activity.container_id = issue.id
        AND activity.payload->''approvalEvent''->>''status'' = ''APPROVED''
        AND activity.creator_id = principal.id
        AND principal.id = member.principal_id
        AND member."role" = ''DBA''
) AND TO_TIMESTAMP(issue.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''
GROUP BY project.resource_id;', 'QUERY', '{"error": "", "duration": "0.003567s"}', '2024-04-10 07:02:29-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (119, 101, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
  issue.id AS issue_id,
  issue.creator_id as creator_id,
  COALESCE(
    array_agg(DISTINCT principal.email) FILTER (
      WHERE
        task_run.creator_id IS NOT NULL
    ),
    ''{}''
  ) AS releaser_emails
FROM
  issue
  LEFT JOIN task ON issue.pipeline_id = task.pipeline_id
  LEFT JOIN task_run ON task_run.task_id = task.id
  LEFT JOIN principal ON task_run.creator_id = principal.id
WHERE
  principal.id = issue.creator_id
  AND issue.status = ''DONE''
GROUP BY
  issue.id
ORDER BY
  issue.id
', 'QUERY', '{"error": "", "duration": "0.002620s"}', '2024-04-10 07:06:16-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (122, 104, 'metadb-project', 'instances/bytebase-meta/databases/bb', '-- Fully completed issues by project
SELECT
  project.resource_id,
  count(*)
FROM
  issue
  LEFT JOIN project ON issue.project_id = project.id
WHERE
  NOT EXISTS (
    SELECT
      1
    FROM
      task,
      task_run
    WHERE
      task.pipeline_id = issue.pipeline_id
      AND task.id = task_run.task_id
      AND task_run.status != ''DONE''
  )
  AND issue.status = ''DONE''
GROUP BY
  project.resource_id;', 'QUERY', '{"error": "", "duration": "0.002810s"}', '2024-04-10 07:19:25-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (123, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT pg_sleep(5)', 'QUERY', '{"error": "", "duration": "5.013435s"}', '2024-05-17 01:47:00-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (124, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{"duration": "0.008848s"}', '2024-07-10 20:42:09-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (125, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM salary;', 'QUERY', '{"duration": "0.009147s"}', '2024-07-10 20:42:14-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (120, 101, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'WITH issue_approvers AS (
  SELECT
    issue.id AS issue_id,
    COALESCE(
      array_agg(DISTINCT principal.email) FILTER (
        WHERE
          x.status = ''APPROVED''
      ),
      ''{}''
    ) AS approver_emails
  FROM
    issue
    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, "principalId" int) ON TRUE
    LEFT JOIN principal ON principal.id = x."principalId"
  GROUP BY
    issue.id
  ORDER BY
    issue.id
),
issue_releasers AS (
  SELECT
    issue.id AS issue_id,
    COALESCE(
      array_agg(DISTINCT principal.email) FILTER (
        WHERE
          task_run.creator_id IS NOT NULL
      ),
      ''{}''
    ) AS releaser_emails
  FROM
    issue
    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id
    LEFT JOIN task_run ON task_run.task_id = task.id
    LEFT JOIN principal ON task_run.creator_id = principal.id
  GROUP BY
    issue.id
  ORDER BY
    issue.id
)

SELECT
  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,
  COUNT(issue.id) AS issue_count,
  ia.approver_emails,
  ir.releaser_emails
FROM
  issue
  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id
  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id
WHERE
  issue.status = ''DONE''
  AND ia.approver_emails @> ir.releaser_emails
  AND ir.releaser_emails @> ia.approver_emails
  AND array_length(ir.releaser_emails, 1) > 0
GROUP BY
  month,
  ia.approver_emails,
  ir.releaser_emails
ORDER BY
  month;', 'QUERY', '{"error": "", "duration": "0.002993s"}', '2024-04-10 07:13:41-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (121, 101, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'WITH issue_approvers AS (
  SELECT
    issue.id AS issue_id,
    COALESCE(
      array_agg(DISTINCT principal.email) FILTER (
        WHERE
          x.status = ''APPROVED''
      ),
      ''{}''
    ) AS approver_emails
  FROM
    issue
    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, "principalId" int) ON TRUE
    LEFT JOIN principal ON principal.id = x."principalId"
  GROUP BY
    issue.id
  ORDER BY
    issue.id
),
issue_releasers AS (
  SELECT
    issue.id AS issue_id,
    COALESCE(
      array_agg(DISTINCT principal.email) FILTER (
        WHERE
          task_run.creator_id IS NOT NULL
      ),
      ''{}''
    ) AS releaser_emails
  FROM
    issue
    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id
    LEFT JOIN task_run ON task_run.task_id = task.id
    LEFT JOIN principal ON task_run.creator_id = principal.id
  GROUP BY
    issue.id
  ORDER BY
    issue.id
)

SELECT
  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,
  COUNT(issue.id) AS issue_count,
  ia.approver_emails,
  ir.releaser_emails
FROM
  issue
  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id
  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id
WHERE
  issue.status = ''DONE''
  AND ia.approver_emails @> ir.releaser_emails
  AND ir.releaser_emails @> ia.approver_emails
  AND array_length(ir.releaser_emails, 1) > 0
GROUP BY
  month,
  ia.approver_emails,
  ir.releaser_emails
ORDER BY
  month;', 'QUERY', '{"error": "", "duration": "0.003365s"}', '2024-04-10 07:15:50-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (126, 101, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{"duration": "0.011041s"}', '2024-07-10 20:42:33-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (127, 101, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
  issue.creator_id,
  principal.email,
  COUNT(issue.creator_id) AS amount
FROM issue
INNER JOIN principal
ON issue.creator_id = principal.id
GROUP BY issue.creator_id, principal.email
ORDER BY COUNT(issue.creator_id) DESC;', 'QUERY', '{"duration": "0.024319125s"}', '2024-09-20 01:13:54-07') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (128, 101, 'gitops-project', 'instances/prod-sample-instance/databases/hr_prod_vcs', 'ALTER TABLE employee ADD COLUMN bugfix TEXT NOT NULL DEFAULT '''';', 'QUERY', '{"duration": "0.009309583s"}', '2025-01-15 20:22:49-08') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (129, 101, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
  *
FROM
  "public"."release"
LIMIT
  50;', 'QUERY', '{"duration": "0.000996134s"}', '2025-01-22 22:26:12-08') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, project_id, database, statement, type, payload, created_at) VALUES (130, 101, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
  *
FROM
  "public"."release"
LIMIT
  50;', 'QUERY', '{"duration": "0.002934110s"}', '2025-01-22 22:26:15-08') ON CONFLICT DO NOTHING;


--
-- Data for Name: release; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: review_config; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.review_config (id, name, payload, enabled) VALUES ('prod', 'SQL Review Sample Policy', '{"sqlReviewRules": [{"type": "database.drop-empty-database", "level": "ERROR", "engine": "MYSQL", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "TIDB", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "OCEANBASE", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "MARIADB", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MYSQL", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "TIDB", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "POSTGRES", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "ORACLE", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MSSQL", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MARIADB", "payload": "{}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MYSQL", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "TIDB", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "POSTGRES", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "OCEANBASE", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "SNOWFLAKE", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MSSQL", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MARIADB", "payload": "{\"format\":\"_del$\"}"}, {"type": "engine.mysql.use-innodb", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "engine.mysql.use-innodb", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "table.disallow-trigger", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.comment", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "table.comment", "level": "DISABLED", "engine": "TIDB", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "table.comment", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "table.comment", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "table.no-duplicate-index", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.text-fields-total-length", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":1000}"}, {"type": "table.disallow-set-charset", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.disallow-on-del-cascade", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-rm-tbl-cascade", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.disallow-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.disallow-order-by", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-order-by", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-order-by", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.disallow-order-by", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.insert.row-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":1000}"}, {"type": "statement.insert.row-limit", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":1000}"}, {"type": "statement.insert.row-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":1000}"}, {"type": "statement.insert.row-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":1000}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "MYSQL", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "POSTGRES", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "OCEANBASE", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "MARIADB", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-add-column-with-default", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.add-check-not-valid", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-add-not-null", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.select-full-table-scan", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.select-full-table-scan", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.select-full-table-scan", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.create-specify-schema", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.check-set-role-variable", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-using-filesort", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-using-temporary", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.where.no-equal-null", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.query.minimum-plan-level", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"string\":\"INDEX\"}"}, {"type": "statement.where.maximum-logical-operator-count", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":10}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-join-table-count", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":10}"}, {"type": "statement.maximum-statements-in-transaction", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":10}"}, {"type": "statement.join-strict-column-attrs", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.prior-backup-check", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.prior-backup-check", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.prior-backup-check", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "naming.fully-qualified", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "naming.table", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table.no-keyword", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "naming.table.no-keyword", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "naming.table.no-keyword", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "naming.table.no-keyword", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "naming.column", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.column", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.column", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.column", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.column", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.pk", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^$|^pk_{{table}}_{{column_list}}$\"}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.column.auto-increment", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^id$\",\"maxLength\":63}"}, {"type": "naming.column.auto-increment", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^id$\",\"maxLength\":63}"}, {"type": "naming.column.auto-increment", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^id$\",\"maxLength\":63}"}, {"type": "naming.column.auto-increment", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^id$\",\"maxLength\":63}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "naming.identifier.case", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"upper\":true}"}, {"type": "naming.identifier.case", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"upper\":true}"}, {"type": "naming.identifier.case", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{\"upper\":true}"}, {"type": "column.required", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"list\":[\"ID\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{\"list\":[\"ID\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.disallow-drop-in-index", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-drop-in-index", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-drop-in-index", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-drop-in-index", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.disallow-change", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-change", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-change", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-change", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.disallow-changing-order", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-changing-order", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-changing-order", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-changing-order", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.auto-increment-must-integer", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.auto-increment-must-integer", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.auto-increment-must-integer", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.auto-increment-must-integer", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.disallow-set-charset", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-set-charset", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-set-charset", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-set-charset", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.auto-increment-must-unsigned", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.auto-increment-must-unsigned", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.auto-increment-must-unsigned", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.auto-increment-must-unsigned", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.comment", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "column.comment", "level": "DISABLED", "engine": "TIDB", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "column.comment", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "column.comment", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":20}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"number\":2560}"}, {"type": "column.auto-increment-initial-value", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":1}"}, {"type": "column.auto-increment-initial-value", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":1}"}, {"type": "column.auto-increment-initial-value", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":1}"}, {"type": "column.auto-increment-initial-value", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":1}"}, {"type": "column.current-time-count-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.current-time-count-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.current-time-count-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.current-time-count-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "index.type-no-blob", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "index.type-no-blob", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "index.type-no-blob", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "index.type-no-blob", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "index.pk-type-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "index.pk-type-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "index.pk-type-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "index.pk-type-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":5}"}, {"type": "index.primary-key-type-allowlist", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"SERIAL\",\"BIGSERIAL\",\"INT\",\"BIGINT\"]}"}, {"type": "index.primary-key-type-allowlist", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"SERIAL\",\"BIGSERIAL\",\"INT\",\"BIGINT\"]}"}, {"type": "index.primary-key-type-allowlist", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"SERIAL\",\"BIGSERIAL\",\"INT\",\"BIGINT\"]}"}, {"type": "index.primary-key-type-allowlist", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"SERIAL\",\"BIGSERIAL\",\"INT\",\"BIGINT\"]}"}, {"type": "index.create-concurrently", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "index.type-allow-list", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"BTREE\",\"HASH\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.comment.length", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":64}"}, {"type": "system.procedure.disallow-create", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "system.event.disallow-create", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "system.view.disallow-create", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "system.function.disallow-create", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "system.function.disallowed-list", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"RAND\",\"UUID\",\"SLEEP\"]}"}, {"type": "table.disallow-ddl", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"list\":[]}"}, {"type": "table.disallow-dml", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"list\":[]}"}, {"type": "table.limit-size", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":10000000}"}, {"type": "statement.add-column-without-position", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.disallow-offline-ddl", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-drop", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "advice.online-migration", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":100000000}"}, {"type": "statement.add-foreign-key-not-valid", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.non-transactional", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-mix-in-ddl", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-mix-in-ddl", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-mix-in-ddl", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-mix-in-dml", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-mix-in-dml", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-mix-in-dml", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}]}', true) ON CONFLICT DO NOTHING;


--
-- Data for Name: revision; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: risk; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.risk (id, source, level, name, active, expression) VALUES (101, 'bb.risk.database.schema.update', 300, 'ALTER column in production environment is high risk', true, '{"expression": "environment_id == \"prod\" && sql_type == \"ALTER_TABLE\""}') ON CONFLICT DO NOTHING;
INSERT INTO public.risk (id, source, level, name, active, expression) VALUES (102, 'bb.risk.database.schema.update', 200, 'CREATE TABLE in production environment is moderate risk', true, '{"expression": "environment_id == \"prod\" && sql_type == \"CREATE_TABLE\""}') ON CONFLICT DO NOTHING;


--
-- Data for Name: role; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.role (id, resource_id, name, description, permissions, payload) VALUES (101, 'tester', 'Tester', 'Custom defined Tester role', '{"permissions": ["bb.changelogs.get", "bb.changelogs.list", "bb.databases.get", "bb.databases.getSchema", "bb.databases.list", "bb.issueComments.create", "bb.issues.get", "bb.issues.list", "bb.planCheckRuns.list", "bb.planCheckRuns.run", "bb.plans.get", "bb.plans.list", "bb.projects.get", "bb.projects.getIamPolicy", "bb.rollouts.get", "bb.taskRuns.list"]}', '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: setting; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.setting (id, name, value) VALUES (101, 'bb.branding.logo', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (102, 'bb.auth.secret', '9Dw1H9JSeEWfjfRnxR5VZ8wuDCIL9ERq') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (103, 'bb.workspace.id', '6c86d081-379d-4366-be6f-481425e6f397') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (106, 'bb.workspace.watermark', '0') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (107, 'bb.plugin.openai.key', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (108, 'bb.plugin.openai.endpoint', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (104, 'bb.enterprise.license', 'eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (112, 'bb.workspace.approval', '{"rules":[{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_OWNER"}]}]},"title":"Project Owner -> Workspace Admin","description":"Project Owner -> Workspace Admin","creatorId":101},"condition":{"expression":"source == \"DDL\" && level == 300"}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","role":"roles/tester"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]}]},"title":"Tester -> Project Owner -> DBA","description":"Tester -> Project Owner -> DBA","creatorId":101},"condition":{}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]}]},"title":"Project Owner -> DBA","description":"The system defines the approval process, first the project Owner approves, then the DBA approves.","creatorId":1},"condition":{}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]}]},"title":"Project Owner","description":"The system defines the approval process and only needs the project Owner o approve it.","creatorId":1},"condition":{"expression":"source == \"DDL\" && level == 200 || source == \"REQUEST_QUERY\" && level == 0 || source == \"REQUEST_EXPORT\" &&\nlevel == 0 || source == \"DATA_EXPORT\" && level == 0 || source == \"DML\" && level == 0 ||\nsource == \"CREATE_DATABASE\" && level == 0 || source == \"DDL\" && level == 0"}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]}]},"title":"DBA","description":"The system defines the approval process and only needs DBA approval.","creatorId":1},"condition":{}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_OWNER"}]}]},"title":"Workspace Admin","description":"The system defines the approval process and only needs Administrator approval.","creatorId":1},"condition":{}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_OWNER"}]}]},"title":"Project Owner -> DBA -> Workspace Admin","description":"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.","creatorId":1},"condition":{}}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (105, 'bb.app.im', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (111, 'bb.workspace.data-classification', '{"configs":[{"id":"2b599739-41da-4c35-a9ff-4a73c6cfe32c", "title":"Classification Example", "levels":[{"id":"1", "title":"Level 1"}, {"id":"2", "title":"Level 2"}, {"id":"3", "title":"Level 3"}, {"id":"4", "title":"Level 4"}], "classification":{"1":{"id":"1", "title":"Basic"}, "1-1":{"id":"1-1", "title":"Basic", "levelId":"1"}, "1-2":{"id":"1-2", "title":"Assert", "levelId":"1"}, "1-3":{"id":"1-3", "title":"Contact", "levelId":"2"}, "1-4":{"id":"1-4", "title":"Health", "levelId":"4"}, "2":{"id":"2", "title":"Relationship"}, "2-1":{"id":"2-1", "title":"Social", "levelId":"1"}, "2-2":{"id":"2-2", "title":"Business", "levelId":"3"}}, "classificationFromConfig":true}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (175, 'bb.workspace.scim', '{"token":"2dc3NMIruyrj0NuaJNOiPCZLLhJGpNr2"}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (176, 'bb.workspace.password-restriction', '{"minLength":8,"requireLetter":true}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (110, 'bb.workspace.schema-template', '{"fieldTemplates":[{"id":"b281c610-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "column":{"name":"creator", "type":"TEXT", "classification":"1-1"}, "catalog":{"name":"creator"}}, {"id":"c5ddd410-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "column":{"name":"updater", "type":"TEXT", "classification":"1-1"}, "catalog":{"name":"updater"}}, {"id":"ce566850-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "column":{"name":"created_ts", "type":"DATE"}, "catalog":{"name":"created_ts"}}, {"id":"d8900d80-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "column":{"name":"updated_ts", "type":"DATE"}, "catalog":{"name":"updated_ts"}}], "tableTemplates":[{"id":"f0fca590-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "table":{"name":"Basic Table", "columns":[{"name":"creator", "type":"TEXT", "classification":"1-1"}, {"name":"created_ts", "type":"DATE"}, {"name":"updater", "type":"TEXT", "classification":"1-1"}, {"name":"updated_ts", "type":"DATE"}]}, "catalog":{"name":"Basic Table", "columnConfigs":[{"name":"creator"}, {"name":"created_ts"}, {"name":"updater"}, {"name":"updated_ts"}]}}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (183, 'bb.plugin.openai.model', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (113, 'bb.workspace.profile', '{"externalUrl":"https://demo.bytebase.com"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: sheet; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (102, 1, 'Alter table to test sample instance for sample issue', '\xdc3cbdad177e12396a1be4e31c959f7b0fdf03193c21bcfa113da7fa23109222', '{}', '2023-11-03 08:46:31-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (103, 1, 'Alter table to prod sample instance for sample issue', '\xdc3cbdad177e12396a1be4e31c959f7b0fdf03193c21bcfa113da7fa23109222', '{}', '2023-11-03 08:46:31-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (106, 106, 'Add Investor Relation department', '\xcc61fce0c8a2f03c11a9850cafa453f9206ae0ca61916a8fe73013e4107d487d', '{}', '2023-11-03 10:28:39-07', 'batch-project') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (107, 104, '[hr_prod] Alter schema @11-04 22:56 UTC+0800', '\x4252fc11123f200fefb2a248f6af18dfebe2990efe69276f8cf282b038ac742f', '{}', '2023-11-04 07:57:12-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (108, 104, 'Add performance table', '\x57a98f9ee5dcd8876d226a80be428980e3a5363fbb208274f8be1a1186b5a77e', '{}', '2023-11-04 08:05:35-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (104, 1, 'bytebase/prod/hr_prod_vcs##20231101##ddl##add_city.sql', '\xecae727c3bd9f84b999130b569547715ba3b02e273f2edc0d3424e996b0b9980', '{}', '2023-11-03 09:07:13-07', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (129, 101, '👉👉👉 [START HERE] Add email column to Employee table', '\x1bea40d232b6f1beafe7aac3ffb41b7e2b438c614ba0edfa21d41559e30ce4e6', '{}', '2024-04-10 01:01:57-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (130, 101, '👉👉👉 [START HERE] Add email column to Employee table', '\xdc3cbdad177e12396a1be4e31c959f7b0fdf03193c21bcfa113da7fa23109222', '{}', '2024-04-10 01:02:37-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (131, 1, 'bytebase/1001_add_phone.sql', '\x11af6768516979ab350006efbdb9cc4b8edbd9fd6be129233862d5f9784069ee', '{}', '2024-04-10 01:18:10-07', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (132, 101, 'Update employee gender to M', '\x012e88c9ae3bed1ad96913baaf979523a61e705ba0113460c6f8dca0182cb54d', '{"engine": "POSTGRES", "commands": [{"end": 34}]}', '2024-05-16 09:58:06-07', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (133, 101, 'Establish "hr_prod_vcs" baseline', '\xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', '{"engine": "POSTGRES"}', '2025-01-15 20:17:55-08', 'gitops-project') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (134, 101, 'Establish "hr_prod" baseline', '\xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', '{"engine": "POSTGRES"}', '2025-01-15 20:19:35-08', 'project-sample') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, creator_id, name, sha256, payload, created_at, project) VALUES (135, 101, 'Establish "hr_test" baseline', '\xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', '{"engine": "POSTGRES"}', '2025-01-15 20:21:52-08', 'project-sample') ON CONFLICT DO NOTHING;


--
-- Data for Name: sheet_blob; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.sheet_blob (sha256, content) VALUES ('\xdc3cbdad177e12396a1be4e31c959f7b0fdf03193c21bcfa113da7fa23109222', 'ALTER TABLE employee ADD COLUMN IF NOT EXISTS email TEXT DEFAULT '''';') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet_blob (sha256, content) VALUES ('\xcc61fce0c8a2f03c11a9850cafa453f9206ae0ca61916a8fe73013e4107d487d', 'INSERT INTO department VALUES(''d010'', ''Investor Relation'');') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet_blob (sha256, content) VALUES ('\x4252fc11123f200fefb2a248f6af18dfebe2990efe69276f8cf282b038ac742f', 'ALTER TABLE "public"."employee"
    ADD COLUMN "city" text NOT NULL;

') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet_blob (sha256, content) VALUES ('\x57a98f9ee5dcd8876d226a80be428980e3a5363fbb208274f8be1a1186b5a77e', 'CREATE TABLE "public"."performance" (
    "creator" text NOT NULL,
    "created_ts" date NOT NULL,
    "updater" text NOT NULL,
    "updated_ts" date NOT NULL,
    "quarter" text NOT NULL,
    "rating" integer NOT NULL
);

COMMENT ON COLUMN "public"."performance"."creator" IS ''1-1'';

COMMENT ON COLUMN "public"."performance"."updater" IS ''1-1'';

') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet_blob (sha256, content) VALUES ('\xecae727c3bd9f84b999130b569547715ba3b02e273f2edc0d3424e996b0b9980', 'ALTER TABLE employee ADD COLUMN city TEXT;
') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet_blob (sha256, content) VALUES ('\x1bea40d232b6f1beafe7aac3ffb41b7e2b438c614ba0edfa21d41559e30ce4e6', 'ALTER TABLE employee ADD COLUMN IF NOT EXISTS email TEXT;') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet_blob (sha256, content) VALUES ('\x11af6768516979ab350006efbdb9cc4b8edbd9fd6be129233862d5f9784069ee', 'ALTER TABLE employee ADD phone TEXT;') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet_blob (sha256, content) VALUES ('\x012e88c9ae3bed1ad96913baaf979523a61e705ba0113460c6f8dca0182cb54d', 'UPDATE employee
SET
  gender = ''M''') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet_blob (sha256, content) VALUES ('\xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', '') ON CONFLICT DO NOTHING;


--
-- Data for Name: stage; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.stage (id, pipeline_id, environment) VALUES (101, 101, 'test') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (102, 101, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (103, 102, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (104, 103, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (105, 103, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (106, 103, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (107, 104, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (108, 105, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (109, 106, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (110, 107, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (111, 108, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (112, 109, 'prod') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, pipeline_id, environment) VALUES (113, 110, 'test') ON CONFLICT DO NOTHING;


--
-- Data for Name: sync_history; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.sync_history (id, created_at, metadata, raw_dump, instance, db_name) VALUES (101, '2023-12-14 05:55:44-08', '{}', '
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

', 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, metadata, raw_dump, instance, db_name) VALUES (102, '2023-12-14 05:55:44-08', '{}', '
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
    city text,
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

', 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, metadata, raw_dump, instance, db_name) VALUES (103, '2025-01-15 20:19:11.013893-08', '{"name":"hr_prod_vcs","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, metadata, raw_dump, instance, db_name) VALUES (104, '2025-01-15 20:19:11.102061-08', '{"name":"hr_prod_vcs","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, metadata, raw_dump, instance, db_name) VALUES (105, '2025-01-15 20:19:38.009574-08', '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, metadata, raw_dump, instance, db_name) VALUES (106, '2025-01-15 20:19:38.056824-08', '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, metadata, raw_dump, instance, db_name) VALUES (107, '2025-01-15 20:21:56.236717-08', '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', 'test-sample-instance', 'hr_test') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, metadata, raw_dump, instance, db_name) VALUES (108, '2025-01-15 20:21:56.282054-08', '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

CREATE SCHEMA "bbdataarchive";

CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "public";

SET default_tablespace = '''';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" ("changed_at");

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" ("operation");

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" ("user_name");

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY ("dept_no");

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY ("emp_no", "dept_no");

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" ("hire_date");

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF (TG_OP = ''INSERT'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''INSERT'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''UPDATE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''UPDATE'', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = ''DELETE'') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (''DELETE'', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY ("emp_no", "from_date");

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" ("amount");

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY ("emp_no", "title", "from_date");

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no");

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no");

', 'test-sample-instance', 'hr_test') ON CONFLICT DO NOTHING;


--
-- Data for Name: task; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (103, 102, 103, 'bb.task.database.schema.update', '{"sheetId": 104}', NULL, 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (104, 103, 104, 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', NULL, 'prod-sample-instance', 'hr_prod_1') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (105, 103, 104, 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', NULL, 'prod-sample-instance', 'hr_prod_4') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (106, 103, 105, 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', NULL, 'prod-sample-instance', 'hr_prod_2') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (107, 103, 105, 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', NULL, 'prod-sample-instance', 'hr_prod_5') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (108, 103, 106, 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', NULL, 'prod-sample-instance', 'hr_prod_3') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (109, 103, 106, 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', NULL, 'prod-sample-instance', 'hr_prod_6') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (110, 104, 107, 'bb.task.database.schema.update', '{"specId": "96967c30-ee17-468e-8368-6366ccc83c52", "sheetId": 107}', NULL, 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (111, 105, 108, 'bb.task.database.schema.update', '{"specId": "9227f0c7-fa7d-44f3-9282-a32da230e2e4", "sheetId": 108}', NULL, 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (101, 101, 101, 'bb.task.database.schema.update', '{"sheetId": 130}', NULL, 'test-sample-instance', 'hr_test') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (102, 101, 102, 'bb.task.database.schema.update', '{"sheetId": 130}', NULL, 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (112, 106, 109, 'bb.task.database.schema.update', '{"specId": "e4010ea4-dd1e-441a-9ea2-90f467ed8506", "sheetId": 131}', NULL, 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (113, 107, 110, 'bb.task.database.data.update', '{"specId": "0992ef9b-3d08-4745-ab40-ff74d34208a8", "sheetId": 132, "rollbackSqlStatus": "PENDING", "preUpdateBackupDetail": {}}', NULL, 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (114, 108, 111, 'bb.task.database.schema.baseline', '{"specId": "ff8ecf1c-f037-4544-971c-c3f4c8ff5889", "taskReleaseSource": {}}', NULL, 'prod-sample-instance', 'hr_prod_vcs') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (115, 109, 112, 'bb.task.database.schema.baseline', '{"specId": "231a929d-bb89-4845-8b7c-6e4870116d32", "taskReleaseSource": {}}', NULL, 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, stage_id, type, payload, earliest_allowed_at, instance, db_name) VALUES (116, 110, 113, 'bb.task.database.schema.baseline', '{"specId": "913aa19f-18e6-42c5-b6e7-2fbb358cffee", "taskReleaseSource": {}}', NULL, 'test-sample-instance', 'hr_test') ON CONFLICT DO NOTHING;


--
-- Data for Name: task_run; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (101, 101, 103, 104, 0, 'DONE', 0, '{"detail": "Applied migration version 1000-ddl to database \"hr_prod_vcs\".", "version": "0000.0000.0000-1000-ddl", "changelog": "instances/prod-sample-instance/databases/hr_prod_vcs/changelogs/110", "changeHistory": "instances/prod-sample-instance/databases/hr_prod_vcs/changeHistories/110"}', '2023-12-14 05:55:44-08', '2023-12-14 05:55:47-08', '2023-12-14 05:55:44-08') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (102, 101, 114, NULL, 0, 'DONE', 0, '{"detail": "Established baseline version  for database \"hr_prod_vcs\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_vcs/changelogs/112"}', '2025-01-15 20:19:11-08', '2025-01-15 20:19:11-08', '2025-01-15 20:19:11-08') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (103, 101, 115, NULL, 0, 'DONE', 0, '{"detail": "Established baseline version  for database \"hr_prod\".", "changelog": "instances/prod-sample-instance/databases/hr_prod/changelogs/113"}', '2025-01-15 20:19:38-08', '2025-01-15 20:19:38-08', '2025-01-15 20:19:38-08') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (104, 101, 116, NULL, 0, 'DONE', 0, '{"detail": "Established baseline version  for database \"hr_test\".", "changelog": "instances/test-sample-instance/databases/hr_test/changelogs/114"}', '2025-01-15 20:21:56-08', '2025-01-15 20:21:56-08', '2025-01-15 20:21:56-08') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (106, 1, 105, 106, 0, 'DONE', 0, '{"detail": "Applied migration version  to database \"hr_prod_4\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_4/changelogs/115"}', '2025-01-22 22:25:52-08', '2025-01-22 22:25:51-08', '2025-01-22 22:25:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (105, 1, 104, 106, 0, 'DONE', 0, '{"detail": "Applied migration version  to database \"hr_prod_1\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_1/changelogs/116"}', '2025-01-22 22:25:52-08', '2025-01-22 22:25:51-08', '2025-01-22 22:25:52-08') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (107, 1, 106, 106, 0, 'DONE', 0, '{"detail": "Applied migration version  to database \"hr_prod_2\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_2/changelogs/117"}', '2025-01-22 22:25:57-08', '2025-01-22 22:25:56-08', '2025-01-22 22:25:57-08') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (108, 1, 107, 106, 0, 'DONE', 0, '{"detail": "Applied migration version  to database \"hr_prod_5\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_5/changelogs/118"}', '2025-01-22 22:25:57-08', '2025-01-22 22:25:56-08', '2025-01-22 22:25:57-08') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (110, 1, 109, 106, 0, 'DONE', 0, '{"detail": "Applied migration version  to database \"hr_prod_6\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_6/changelogs/119"}', '2025-01-22 22:26:02-08', '2025-01-22 22:26:01-08', '2025-01-22 22:26:02-08') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, task_id, sheet_id, attempt, status, code, result, created_at, updated_at, started_at) VALUES (109, 1, 108, 106, 0, 'DONE', 0, '{"detail": "Applied migration version  to database \"hr_prod_3\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_3/changelogs/120"}', '2025-01-22 22:26:02-08', '2025-01-22 22:26:01-08', '2025-01-22 22:26:02-08') ON CONFLICT DO NOTHING;


--
-- Data for Name: task_run_log; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (101, 102, '2025-01-15 20:19:10.926834-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (102, 102, '2025-01-15 20:19:10.930371-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (103, 103, '2025-01-15 20:19:37.82721-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (104, 103, '2025-01-15 20:19:37.828077-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (105, 104, '2025-01-15 20:21:56.124197-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (106, 104, '2025-01-15 20:21:56.125869-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (107, 105, '2025-01-22 22:25:51.541499-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (108, 106, '2025-01-22 22:25:51.545367-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (109, 105, '2025-01-22 22:25:51.547841-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (110, 106, '2025-01-22 22:25:51.550248-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (111, 106, '2025-01-22 22:25:51.554241-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (112, 106, '2025-01-22 22:25:51.55543-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (113, 105, '2025-01-22 22:25:51.556511-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (114, 105, '2025-01-22 22:25:51.558478-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (115, 106, '2025-01-22 22:25:51.586945-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (116, 105, '2025-01-22 22:25:51.586968-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (117, 106, '2025-01-22 22:25:51.587939-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (118, 105, '2025-01-22 22:25:51.589077-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (119, 106, '2025-01-22 22:25:51.590906-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (120, 105, '2025-01-22 22:25:51.591974-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (121, 106, '2025-01-22 22:25:51.592644-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (122, 105, '2025-01-22 22:25:51.595153-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (123, 107, '2025-01-22 22:25:56.545924-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (124, 108, '2025-01-22 22:25:56.549066-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (125, 107, '2025-01-22 22:25:56.551389-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (126, 108, '2025-01-22 22:25:56.553426-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (127, 107, '2025-01-22 22:25:56.55349-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (128, 107, '2025-01-22 22:25:56.555496-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (129, 108, '2025-01-22 22:25:56.557263-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (130, 108, '2025-01-22 22:25:56.558372-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (131, 107, '2025-01-22 22:25:56.580767-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (132, 108, '2025-01-22 22:25:56.582381-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (133, 107, '2025-01-22 22:25:56.582827-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (134, 108, '2025-01-22 22:25:56.584187-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (135, 107, '2025-01-22 22:25:56.58685-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (136, 108, '2025-01-22 22:25:56.586902-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (137, 107, '2025-01-22 22:25:56.589412-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (138, 108, '2025-01-22 22:25:56.590801-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (139, 109, '2025-01-22 22:26:01.54505-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (140, 110, '2025-01-22 22:26:01.547684-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (141, 109, '2025-01-22 22:26:01.549862-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (142, 110, '2025-01-22 22:26:01.551335-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (143, 109, '2025-01-22 22:26:01.552123-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (144, 109, '2025-01-22 22:26:01.553567-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (145, 110, '2025-01-22 22:26:01.554971-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (146, 110, '2025-01-22 22:26:01.55597-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (147, 110, '2025-01-22 22:26:01.578364-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (151, 110, '2025-01-22 22:26:01.583553-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (152, 110, '2025-01-22 22:26:01.585495-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (154, 109, '2025-01-22 22:26:01.588284-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (148, 110, '2025-01-22 22:26:01.580435-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (149, 109, '2025-01-22 22:26:01.58233-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (150, 109, '2025-01-22 22:26:01.583143-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_at, payload) VALUES (153, 109, '2025-01-22 22:26:01.585567-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;


--
-- Data for Name: user_group; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: worksheet; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.worksheet (id, creator_id, name, statement, visibility, payload, created_at, updated_at, project, instance, db_name) VALUES (105, 101, 'All employee', 'SELECT * FROM employee;', 'PROJECT_READ', '{}', '2023-11-03 10:23:05-07', '2024-04-10 00:38:19-07', 'project-sample', 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, creator_id, name, statement, visibility, payload, created_at, updated_at, project, instance, db_name) VALUES (101, 101, 'All salary', 'SELECT * FROM salary;', 'PROJECT_READ', '{}', '2023-11-03 08:46:31-07', '2024-04-10 00:38:19-07', 'project-sample', 'prod-sample-instance', 'hr_prod') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, creator_id, name, statement, visibility, payload, created_at, updated_at, project, instance, db_name) VALUES (108, 101, 'Issues with the same creator and releaser', 'SELECT
  issue.id AS issue_id,
  issue.creator_id as creator_id,
  COALESCE(
    array_agg(DISTINCT principal.email) FILTER (
      WHERE
        task_run.creator_id IS NOT NULL
    ),
    ''{}''
  ) AS releaser_emails
FROM
  issue
  LEFT JOIN task ON issue.pipeline_id = task.pipeline_id
  LEFT JOIN task_run ON task_run.task_id = task.id
  LEFT JOIN principal ON task_run.creator_id = principal.id
WHERE
  principal.id = issue.creator_id
  AND issue.status = ''DONE''
GROUP BY
  issue.id
ORDER BY
  issue.id', 'PROJECT_READ', '{}', '2024-04-10 07:07:12-07', '2024-04-10 07:19:13-07', 'metadb-project', 'bytebase-meta', 'bb') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, creator_id, name, statement, visibility, payload, created_at, updated_at, project, instance, db_name) VALUES (106, 101, 'Completed issues by project', '-- Fully completed issues by project
SELECT
  project.resource_id,
  count(*)
FROM
  issue
  LEFT JOIN project ON issue.project_id = project.id
WHERE
  NOT EXISTS (
    SELECT
      1
    FROM
      task,
      task_run
    WHERE
      task.pipeline_id = issue.pipeline_id
      AND task.id = task_run.task_id
      AND task_run.status != ''DONE''
  )
  AND issue.status = ''DONE''
GROUP BY
  project.resource_id;', 'PROJECT_READ', '{}', '2024-04-10 07:01:52-07', '2024-04-10 07:02:06-07', 'metadb-project', 'bytebase-meta', 'bb') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, creator_id, name, statement, visibility, payload, created_at, updated_at, project, instance, db_name) VALUES (107, 101, 'Issues reviewed during a period', '-- Issues reviewed between 17:30 and 23:59:59
SELECT
  project.resource_id,
  count(*)
FROM
  issue
  LEFT JOIN project ON issue.project_id = project.id
WHERE
  EXISTS (
    SELECT
      1
    FROM
      activity,
      principal,
      member
    WHERE
      TO_TIMESTAMP(activity.created_ts) :: TIME BETWEEN TIME ''17:30:00+08''
      AND ''23:59:59+08''
      AND activity.type = ''bb.issue.comment.create''
      AND activity.container_id = issue.id
      AND activity.payload -> ''approvalEvent'' ->> ''status'' = ''APPROVED''
      AND activity.creator_id = principal.id
      AND principal.id = member.principal_id
      AND member."role" = ''DBA''
  )
  AND TO_TIMESTAMP(issue.created_ts) :: TIME BETWEEN TIME ''17:30:00+08''
  AND ''23:59:59+08''
GROUP BY
  project.resource_id;', 'PROJECT_READ', '{}', '2024-04-10 07:05:13-07', '2024-04-10 07:08:12-07', 'metadb-project', 'bytebase-meta', 'bb') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, creator_id, name, statement, visibility, payload, created_at, updated_at, project, instance, db_name) VALUES (110, 101, 'Issues created by user', '-- Issues created by user
SELECT
  issue.creator_id,
  principal.email,
  COUNT(issue.creator_id) AS amount
FROM
  issue
  INNER JOIN principal ON issue.creator_id = principal.id
GROUP BY
  issue.creator_id,
  principal.email
ORDER BY
  COUNT(issue.creator_id) DESC;', 'PROJECT_READ', '{}', '2024-09-20 01:14:53-07', '2024-09-20 01:15:41-07', 'metadb-project', 'bytebase-meta', 'bb') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, creator_id, name, statement, visibility, payload, created_at, updated_at, project, instance, db_name) VALUES (109, 101, 'Issues with the same approver and releaser by month', 'WITH issue_approvers AS (
  SELECT
    issue.id AS issue_id,
    COALESCE(
      array_agg(DISTINCT principal.email) FILTER (
        WHERE
          x.status = ''APPROVED''
      ),
      ''{}''
    ) AS approver_emails
  FROM
    issue
    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, "principalId" int) ON TRUE
    LEFT JOIN principal ON principal.id = x."principalId"
  GROUP BY
    issue.id
  ORDER BY
    issue.id
),
issue_releasers AS (
  SELECT
    issue.id AS issue_id,
    COALESCE(
      array_agg(DISTINCT principal.email) FILTER (
        WHERE
          task_run.creator_id IS NOT NULL
      ),
      ''{}''
    ) AS releaser_emails
  FROM
    issue
    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id
    LEFT JOIN task_run ON task_run.task_id = task.id
    LEFT JOIN principal ON task_run.creator_id = principal.id
  GROUP BY
    issue.id
  ORDER BY
    issue.id
)

SELECT
  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,
  COUNT(issue.id) AS issue_count,
  ia.approver_emails,
  ir.releaser_emails
FROM
  issue
  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id
  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id
WHERE
  issue.status = ''DONE''
  AND ia.approver_emails @> ir.releaser_emails
  AND ir.releaser_emails @> ia.approver_emails
  AND array_length(ir.releaser_emails, 1) > 0
GROUP BY
  month,
  ia.approver_emails,
  ir.releaser_emails
ORDER BY
  month;', 'PROJECT_READ', '{}', '2024-04-10 07:09:02-07', '2024-04-10 07:19:10-07', 'metadb-project', 'bytebase-meta', 'bb') ON CONFLICT DO NOTHING;


--
-- Data for Name: worksheet_organizer; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Name: anomaly_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.anomaly_id_seq', 109, true);


--
-- Name: audit_log_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.audit_log_id_seq', 159, true);


--
-- Name: changelist_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.changelist_id_seq', 101, true);


--
-- Name: changelog_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.changelog_id_seq', 120, true);


--
-- Name: data_source_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.data_source_id_seq', 105, true);


--
-- Name: db_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.db_group_id_seq', 102, true);


--
-- Name: db_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.db_id_seq', 115, true);


--
-- Name: db_schema_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.db_schema_id_seq', 189, true);


--
-- Name: environment_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.environment_id_seq', 103, false);


--
-- Name: export_archive_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.export_archive_id_seq', 1, false);


--
-- Name: idp_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.idp_id_seq', 101, false);


--
-- Name: instance_change_history_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.instance_change_history_id_seq', 231, true);


--
-- Name: instance_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.instance_id_seq', 103, true);


--
-- Name: issue_comment_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.issue_comment_id_seq', 141, true);


--
-- Name: issue_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.issue_id_seq', 110, true);


--
-- Name: pipeline_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.pipeline_id_seq', 110, true);


--
-- Name: plan_check_run_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.plan_check_run_id_seq', 171, true);


--
-- Name: plan_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.plan_id_seq', 110, true);


--
-- Name: policy_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.policy_id_seq', 192, true);


--
-- Name: principal_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.principal_id_seq', 109, true);


--
-- Name: project_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.project_id_seq', 104, true);


--
-- Name: project_webhook_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.project_webhook_id_seq', 101, false);


--
-- Name: query_history_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.query_history_id_seq', 130, true);


--
-- Name: release_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.release_id_seq', 101, false);


--
-- Name: revision_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.revision_id_seq', 101, false);


--
-- Name: risk_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.risk_id_seq', 102, true);


--
-- Name: role_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.role_id_seq', 101, true);


--
-- Name: setting_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.setting_id_seq', 207, true);


--
-- Name: sheet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.sheet_id_seq', 135, true);


--
-- Name: stage_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.stage_id_seq', 113, true);


--
-- Name: sync_history_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.sync_history_id_seq', 108, true);


--
-- Name: task_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.task_id_seq', 116, true);


--
-- Name: task_run_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.task_run_id_seq', 110, true);


--
-- Name: task_run_log_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.task_run_log_id_seq', 154, true);


--
-- Name: worksheet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.worksheet_id_seq', 110, true);


--
-- Name: worksheet_organizer_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.worksheet_organizer_id_seq', 1, false);


--
-- Name: anomaly anomaly_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_pkey PRIMARY KEY (id);


--
-- Name: audit_log audit_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);


--
-- Name: changelist changelist_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_pkey PRIMARY KEY (id);


--
-- Name: changelog changelog_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_pkey PRIMARY KEY (id);


--
-- Name: data_source data_source_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_pkey PRIMARY KEY (id);


--
-- Name: db_group db_group_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_pkey PRIMARY KEY (id);


--
-- Name: db db_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_pkey PRIMARY KEY (id);


--
-- Name: db_schema db_schema_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_pkey PRIMARY KEY (id);


--
-- Name: environment environment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_pkey PRIMARY KEY (id);


--
-- Name: export_archive export_archive_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.export_archive
    ADD CONSTRAINT export_archive_pkey PRIMARY KEY (id);


--
-- Name: idp idp_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.idp
    ADD CONSTRAINT idp_pkey PRIMARY KEY (id);


--
-- Name: instance_change_history instance_change_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_pkey PRIMARY KEY (id);


--
-- Name: instance instance_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_pkey PRIMARY KEY (id);


--
-- Name: issue_comment issue_comment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comment
    ADD CONSTRAINT issue_comment_pkey PRIMARY KEY (id);


--
-- Name: issue issue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_pkey PRIMARY KEY (id);


--
-- Name: issue_subscriber issue_subscriber_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_pkey PRIMARY KEY (issue_id, subscriber_id);


--
-- Name: pipeline pipeline_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_pkey PRIMARY KEY (id);


--
-- Name: plan_check_run plan_check_run_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_pkey PRIMARY KEY (id);


--
-- Name: plan plan_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_pkey PRIMARY KEY (id);


--
-- Name: policy policy_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_pkey PRIMARY KEY (id);


--
-- Name: principal principal_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_pkey PRIMARY KEY (id);


--
-- Name: project project_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_pkey PRIMARY KEY (id);


--
-- Name: project_webhook project_webhook_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_pkey PRIMARY KEY (id);


--
-- Name: query_history query_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.query_history
    ADD CONSTRAINT query_history_pkey PRIMARY KEY (id);


--
-- Name: release release_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release
    ADD CONSTRAINT release_pkey PRIMARY KEY (id);


--
-- Name: review_config review_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_config
    ADD CONSTRAINT review_config_pkey PRIMARY KEY (id);


--
-- Name: revision revision_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision
    ADD CONSTRAINT revision_pkey PRIMARY KEY (id);


--
-- Name: risk risk_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_pkey PRIMARY KEY (id);


--
-- Name: role role_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_pkey PRIMARY KEY (id);


--
-- Name: setting setting_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_pkey PRIMARY KEY (id);


--
-- Name: sheet_blob sheet_blob_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet_blob
    ADD CONSTRAINT sheet_blob_pkey PRIMARY KEY (sha256);


--
-- Name: sheet sheet_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_pkey PRIMARY KEY (id);


--
-- Name: stage stage_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_pkey PRIMARY KEY (id);


--
-- Name: sync_history sync_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sync_history
    ADD CONSTRAINT sync_history_pkey PRIMARY KEY (id);


--
-- Name: task task_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_pkey PRIMARY KEY (id);


--
-- Name: task_run_log task_run_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run_log
    ADD CONSTRAINT task_run_log_pkey PRIMARY KEY (id);


--
-- Name: task_run task_run_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_pkey PRIMARY KEY (id);


--
-- Name: user_group user_group_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group
    ADD CONSTRAINT user_group_pkey PRIMARY KEY (email);


--
-- Name: worksheet_organizer worksheet_organizer_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet_organizer
    ADD CONSTRAINT worksheet_organizer_pkey PRIMARY KEY (id);


--
-- Name: worksheet worksheet_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_pkey PRIMARY KEY (id);


--
-- Name: idx_anomaly_unique_project_instance_dn_name_type; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_anomaly_unique_project_instance_dn_name_type ON public.anomaly USING btree (project, instance, db_name, type);


--
-- Name: idx_audit_log_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_created_at ON public.audit_log USING btree (created_at);


--
-- Name: idx_audit_log_payload_method; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_payload_method ON public.audit_log USING btree (((payload ->> 'method'::text)));


--
-- Name: idx_audit_log_payload_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_payload_parent ON public.audit_log USING btree (((payload ->> 'parent'::text)));


--
-- Name: idx_audit_log_payload_resource; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_payload_resource ON public.audit_log USING btree (((payload ->> 'resource'::text)));


--
-- Name: idx_audit_log_payload_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_payload_user ON public.audit_log USING btree (((payload ->> 'user'::text)));


--
-- Name: idx_changelist_project_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_changelist_project_name ON public.changelist USING btree (project, name);


--
-- Name: idx_changelog_instance_db_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_changelog_instance_db_name ON public.changelog USING btree (instance, db_name);


--
-- Name: idx_db_group_unique_project_placeholder; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_placeholder ON public.db_group USING btree (project, placeholder);


--
-- Name: idx_db_group_unique_project_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_resource_id ON public.db_group USING btree (project, resource_id);


--
-- Name: idx_db_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_db_project ON public.db USING btree (project);


--
-- Name: idx_db_schema_unique_instance_db_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_schema_unique_instance_db_name ON public.db_schema USING btree (instance, db_name);


--
-- Name: idx_db_unique_instance_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_unique_instance_name ON public.db USING btree (instance, name);


--
-- Name: idx_environment_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_environment_unique_resource_id ON public.environment USING btree (resource_id);


--
-- Name: idx_idp_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON public.idp USING btree (resource_id);


--
-- Name: idx_instance_change_history_unique_version; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_instance_change_history_unique_version ON public.instance_change_history USING btree (version);


--
-- Name: idx_instance_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON public.instance USING btree (resource_id);


--
-- Name: idx_issue_comment_issue_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_comment_issue_id ON public.issue_comment USING btree (issue_id);


--
-- Name: idx_issue_creator_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_creator_id ON public.issue USING btree (creator_id);


--
-- Name: idx_issue_pipeline_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_pipeline_id ON public.issue USING btree (pipeline_id);


--
-- Name: idx_issue_plan_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_plan_id ON public.issue USING btree (plan_id);


--
-- Name: idx_issue_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_project ON public.issue USING btree (project);


--
-- Name: idx_issue_subscriber_subscriber_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_subscriber_subscriber_id ON public.issue_subscriber USING btree (subscriber_id);


--
-- Name: idx_issue_ts_vector; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_ts_vector ON public.issue USING gin (ts_vector);


--
-- Name: idx_plan_check_run_plan_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_check_run_plan_id ON public.plan_check_run USING btree (plan_id);


--
-- Name: idx_plan_pipeline_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_pipeline_id ON public.plan USING btree (pipeline_id);


--
-- Name: idx_plan_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_project ON public.plan USING btree (project);


--
-- Name: idx_policy_unique_resource_type_resource_type; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_type ON public.policy USING btree (resource_type, resource, type);


--
-- Name: idx_project_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_project_unique_resource_id ON public.project USING btree (resource_id);


--
-- Name: idx_project_webhook_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_project_webhook_project ON public.project_webhook USING btree (project);


--
-- Name: idx_query_history_creator_id_created_at_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_query_history_creator_id_created_at_project_id ON public.query_history USING btree (creator_id, created_at, project_id DESC);


--
-- Name: idx_release_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_release_project ON public.release USING btree (project);


--
-- Name: idx_revision_instance_db_name_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_revision_instance_db_name_version ON public.revision USING btree (instance, db_name, version);


--
-- Name: idx_revision_unique_instance_db_name_version_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_version_deleted_at_null ON public.revision USING btree (instance, db_name, version) WHERE (deleted_at IS NULL);


--
-- Name: idx_role_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_role_unique_resource_id ON public.role USING btree (resource_id);


--
-- Name: idx_setting_unique_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_setting_unique_name ON public.setting USING btree (name);


--
-- Name: idx_sheet_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sheet_project ON public.sheet USING btree (project);


--
-- Name: idx_stage_pipeline_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_stage_pipeline_id ON public.stage USING btree (pipeline_id);


--
-- Name: idx_sync_history_instance_db_name_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sync_history_instance_db_name_created_at ON public.sync_history USING btree (instance, db_name, created_at);


--
-- Name: idx_task_pipeline_id_stage_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_pipeline_id_stage_id ON public.task USING btree (pipeline_id, stage_id);


--
-- Name: idx_task_run_log_task_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_run_log_task_run_id ON public.task_run_log USING btree (task_run_id);


--
-- Name: idx_task_run_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_run_task_id ON public.task_run USING btree (task_id);


--
-- Name: idx_worksheet_creator_id_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_worksheet_creator_id_project ON public.worksheet USING btree (creator_id, project);


--
-- Name: idx_worksheet_organizer_principal_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_worksheet_organizer_principal_id ON public.worksheet_organizer USING btree (principal_id);


--
-- Name: idx_worksheet_organizer_unique_sheet_id_principal_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal_id ON public.worksheet_organizer USING btree (worksheet_id, principal_id);


--
-- Name: uk_task_run_task_id_attempt; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON public.task_run USING btree (task_id, attempt);


--
-- Name: anomaly anomaly_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES public.db(instance, name);


--
-- Name: changelist changelist_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: changelist changelist_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: changelog changelog_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES public.db(instance, name);


--
-- Name: changelog changelog_prev_sync_history_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_prev_sync_history_id_fkey FOREIGN KEY (prev_sync_history_id) REFERENCES public.sync_history(id);


--
-- Name: changelog changelog_sync_history_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_sync_history_id_fkey FOREIGN KEY (sync_history_id) REFERENCES public.sync_history(id);


--
-- Name: data_source data_source_instance_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_instance_fkey FOREIGN KEY (instance) REFERENCES public.instance(resource_id);


--
-- Name: db db_environment_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_environment_fkey FOREIGN KEY (environment) REFERENCES public.environment(resource_id);


--
-- Name: db_group db_group_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: db db_instance_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_instance_fkey FOREIGN KEY (instance) REFERENCES public.instance(resource_id);


--
-- Name: db db_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: db_schema db_schema_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES public.db(instance, name);


--
-- Name: instance instance_environment_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_environment_fkey FOREIGN KEY (environment) REFERENCES public.environment(resource_id);


--
-- Name: issue_comment issue_comment_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comment
    ADD CONSTRAINT issue_comment_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: issue_comment issue_comment_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comment
    ADD CONSTRAINT issue_comment_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);


--
-- Name: issue issue_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: issue issue_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);


--
-- Name: issue issue_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plan(id);


--
-- Name: issue issue_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: issue_subscriber issue_subscriber_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);


--
-- Name: issue_subscriber issue_subscriber_subscriber_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES public.principal(id);


--
-- Name: pipeline pipeline_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: pipeline pipeline_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: plan_check_run plan_check_run_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plan(id);


--
-- Name: plan plan_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: plan plan_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);


--
-- Name: plan plan_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: project_webhook project_webhook_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: query_history query_history_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.query_history
    ADD CONSTRAINT query_history_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: release release_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release
    ADD CONSTRAINT release_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: release release_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release
    ADD CONSTRAINT release_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: revision revision_deleter_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision
    ADD CONSTRAINT revision_deleter_id_fkey FOREIGN KEY (deleter_id) REFERENCES public.principal(id);


--
-- Name: revision revision_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision
    ADD CONSTRAINT revision_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES public.db(instance, name);


--
-- Name: sheet sheet_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: sheet sheet_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: stage stage_environment_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_environment_fkey FOREIGN KEY (environment) REFERENCES public.environment(resource_id);


--
-- Name: stage stage_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);


--
-- Name: sync_history sync_history_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sync_history
    ADD CONSTRAINT sync_history_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES public.db(instance, name);


--
-- Name: task task_instance_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_instance_fkey FOREIGN KEY (instance) REFERENCES public.instance(resource_id);


--
-- Name: task task_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);


--
-- Name: task_run task_run_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: task_run_log task_run_log_task_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run_log
    ADD CONSTRAINT task_run_log_task_run_id_fkey FOREIGN KEY (task_run_id) REFERENCES public.task_run(id);


--
-- Name: task_run task_run_sheet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_sheet_id_fkey FOREIGN KEY (sheet_id) REFERENCES public.sheet(id);


--
-- Name: task_run task_run_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.task(id);


--
-- Name: task task_stage_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.stage(id);


--
-- Name: worksheet worksheet_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: worksheet_organizer worksheet_organizer_principal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet_organizer
    ADD CONSTRAINT worksheet_organizer_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES public.principal(id);


--
-- Name: worksheet_organizer worksheet_organizer_worksheet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet_organizer
    ADD CONSTRAINT worksheet_organizer_worksheet_id_fkey FOREIGN KEY (worksheet_id) REFERENCES public.worksheet(id) ON DELETE CASCADE;


--
-- Name: worksheet worksheet_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- PostgreSQL database dump complete
--

