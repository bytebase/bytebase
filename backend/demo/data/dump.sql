--
-- PostgreSQL database dump
--


-- Dumped from database version 16.10
-- Dumped by pg_dump version 16.10

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
-- Name: audit_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_log (
    id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
-- Name: changelog; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changelog (
    id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    status text NOT NULL,
    prev_sync_history_id bigint,
    sync_history_id bigint,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
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
-- Name: db; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.db (
    id integer NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    project text NOT NULL,
    instance text NOT NULL,
    name text NOT NULL,
    environment text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: db_group; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.db_group (
    id bigint NOT NULL,
    project text NOT NULL,
    resource_id text NOT NULL,
    name text DEFAULT ''::text NOT NULL,
    expression jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}'::json NOT NULL,
    raw_dump text DEFAULT ''::text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL
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
-- Name: export_archive; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.export_archive (
    id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    bytes bytea,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
    deleted boolean DEFAULT false NOT NULL,
    environment text,
    resource_id text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL
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
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    plan_id bigint,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    ts_vector tsvector,
    CONSTRAINT issue_status_check CHECK ((status = ANY (ARRAY['OPEN'::text, 'DONE'::text, 'CANCELED'::text])))
);


--
-- Name: issue_comment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_comment (
    id bigint NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    issue_id integer NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
-- Name: pipeline; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pipeline (
    id integer NOT NULL,
    creator_id integer NOT NULL,
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
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    pipeline_id integer,
    name text NOT NULL,
    description text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);


--
-- Name: plan_check_run; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_check_run (
    id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    plan_id bigint NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
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
    enforce boolean DEFAULT true NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    resource_type text NOT NULL,
    resource text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    inherit_from_parent boolean DEFAULT true NOT NULL
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
    deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text DEFAULT ''::text NOT NULL,
    mfa_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    profile jsonb DEFAULT '{}'::jsonb NOT NULL,
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
    deleted boolean DEFAULT false NOT NULL,
    name text NOT NULL,
    resource_id text NOT NULL,
    data_classification_config_id text DEFAULT ''::text NOT NULL,
    setting jsonb DEFAULT '{}'::jsonb NOT NULL
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
    project text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project_id text NOT NULL,
    database text NOT NULL,
    statement text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
    deleted boolean DEFAULT false NOT NULL,
    project text NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    digest text DEFAULT ''::text NOT NULL
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
    enabled boolean DEFAULT true NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: revision; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.revision (
    id bigint NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleter_id integer,
    deleted_at timestamp with time zone,
    version text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
    name text NOT NULL,
    active boolean NOT NULL,
    expression jsonb NOT NULL,
    level text NOT NULL,
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
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    sha256 bytea NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
-- Name: sync_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sync_history (
    id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}'::json NOT NULL,
    raw_dump text DEFAULT ''::text NOT NULL
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
    instance text NOT NULL,
    db_name text,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    environment text
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
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_id integer NOT NULL,
    sheet_id integer,
    attempt integer NOT NULL,
    status text NOT NULL,
    started_at timestamp with time zone,
    code integer DEFAULT 0 NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    run_at timestamp with time zone,
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
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    instance text,
    db_name text,
    name text NOT NULL,
    statement text NOT NULL,
    visibility text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
-- Name: audit_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_log ALTER COLUMN id SET DEFAULT nextval('public.audit_log_id_seq'::regclass);


--
-- Name: changelog id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog ALTER COLUMN id SET DEFAULT nextval('public.changelog_id_seq'::regclass);


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
-- Data for Name: audit_log; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.audit_log (id, created_at, payload) VALUES (101, '2025-05-26 06:48:06.307193+00', '{"method": "/bytebase.v1.UserService/CreateUser", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"user\":{\"email\":\"demo@example.com\", \"title\":\"Demo\", \"userType\":\"USER\"}}", "response": "{\"name\":\"users/101\", \"email\":\"demo@example.com\", \"title\":\"Demo\", \"userType\":\"USER\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:56517", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (102, '2025-05-26 06:48:06.450846+00', '{"user": "users/101", "method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"email\":\"demo@example.com\", \"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\", \"email\":\"demo@example.com\", \"title\":\"Demo\", \"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:56753", "callerSuppliedUserAgent": "grpc-go/1.71.0"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (103, '2025-05-26 06:48:45.01432+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.profile\", \"value\":{\"workspaceProfileSettingValue\":{\"databaseChangeMode\":\"PIPELINE\"}}}, \"allowMissing\":true, \"updateMask\":\"value.workspaceProfileSettingValue.databaseChangeMode\"}", "resource": "settings/bb.workspace.profile", "response": "{\"name\":\"settings/bb.workspace.profile\", \"value\":{\"workspaceProfileSettingValue\":{\"databaseChangeMode\":\"PIPELINE\"}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.profile", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceProfileSettingValue": {}}}, "requestMetadata": {"callerIp": "[::1]:56516", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (104, '2025-05-26 06:51:21.254849+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.profile\", \"value\":{\"workspaceProfileSettingValue\":{\"externalUrl\":\"https://demo.bytebase.com\", \"databaseChangeMode\":\"PIPELINE\"}}}, \"allowMissing\":true, \"updateMask\":\"value.workspaceProfileSettingValue.externalUrl\"}", "resource": "settings/bb.workspace.profile", "response": "{\"name\":\"settings/bb.workspace.profile\", \"value\":{\"workspaceProfileSettingValue\":{\"externalUrl\":\"https://demo.bytebase.com\", \"databaseChangeMode\":\"PIPELINE\"}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.profile", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceProfileSettingValue": {"databaseChangeMode": "PIPELINE"}}}, "requestMetadata": {"callerIp": "[::1]:56515", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (105, '2025-05-26 07:15:33.833558+00', '{"user": "users/101", "method": "/bytebase.v1.UserService/CreateUser", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"user\":{\"name\":\"users/0\", \"email\":\"dev1@example.com\", \"title\":\"Dev1\", \"userType\":\"USER\"}}", "resource": "users/0", "response": "{\"name\":\"users/102\", \"email\":\"dev1@example.com\", \"title\":\"Dev1\", \"userType\":\"USER\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:58583", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (106, '2025-05-26 07:15:33.845475+00', '{"user": "users/101", "method": "/bytebase.v1.WorkspaceService/SetIamPolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"resource\":\"workspaces/-\", \"policy\":{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\"], \"condition\":{}}], \"etag\":\"1748242086302\"}, \"etag\":\"1748242086302\"}", "resource": "workspaces/-", "response": "{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\"], \"condition\":{}}], \"etag\":\"1748243733843\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:58583", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (107, '2025-05-26 07:16:05.086374+00', '{"user": "users/101", "method": "/bytebase.v1.UserService/CreateUser", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"user\":{\"name\":\"users/0\", \"email\":\"dba1@example.com\", \"title\":\"dba1\", \"userType\":\"USER\"}}", "resource": "users/0", "response": "{\"name\":\"users/103\", \"email\":\"dba1@example.com\", \"title\":\"dba1\", \"userType\":\"USER\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:58583", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (108, '2025-05-26 07:16:05.093675+00', '{"user": "users/101", "method": "/bytebase.v1.WorkspaceService/SetIamPolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"resource\":\"workspaces/-\", \"policy\":{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\"]}], \"etag\":\"1748243733843\"}, \"etag\":\"1748243733843\"}", "resource": "workspaces/-", "response": "{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\"], \"condition\":{}}], \"etag\":\"1748243765092\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:58583", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (109, '2025-05-26 07:16:33.694524+00', '{"user": "users/101", "method": "/bytebase.v1.UserService/CreateUser", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"user\":{\"name\":\"users/0\", \"email\":\"api@service.bytebase.com\", \"title\":\"API user\", \"userType\":\"SERVICE_ACCOUNT\"}}", "resource": "users/0", "response": "{\"name\":\"users/104\", \"email\":\"api@service.bytebase.com\", \"title\":\"API user\", \"userType\":\"SERVICE_ACCOUNT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:58813", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (167, '2025-05-26 08:23:40.136557+00', '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/UpdatePolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"policy\":{\"name\":\"environments/prod/policies/disable_copy_data\", \"type\":\"DISABLE_COPY_DATA\", \"disableCopyDataPolicy\":{\"active\":true}}, \"updateMask\":\"payload\", \"allowMissing\":true}", "response": "{\"name\":\"environments/prod/policies/disable_copy_data\", \"type\":\"DISABLE_COPY_DATA\", \"disableCopyDataPolicy\":{\"active\":true}, \"enforce\":true, \"resourceType\":\"ENVIRONMENT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63481", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (110, '2025-05-26 07:16:33.700462+00', '{"user": "users/101", "method": "/bytebase.v1.WorkspaceService/SetIamPolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"resource\":\"workspaces/-\", \"policy\":{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\"], \"condition\":{}}], \"etag\":\"1748243765092\"}, \"etag\":\"1748243765092\"}", "resource": "workspaces/-", "response": "{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\"], \"condition\":{}}], \"etag\":\"1748243793699\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:58813", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (111, '2025-05-26 07:24:34.037938+00', '{"user": "users/101", "method": "/bytebase.v1.RoleService/UpdateRole", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"role\":{\"name\":\"roles/qa-custom-role\", \"title\":\"QA\", \"description\":\"Custom-defined QA role\", \"permissions\":[\"bb.databases.get\", \"bb.databases.getSchema\", \"bb.databases.list\", \"bb.issueComments.create\", \"bb.issues.get\", \"bb.issues.list\", \"bb.planCheckRuns.list\", \"bb.planCheckRuns.run\", \"bb.plans.get\", \"bb.plans.list\", \"bb.projects.get\", \"bb.projects.getIamPolicy\", \"bb.rollouts.get\", \"bb.taskRuns.list\"]}, \"updateMask\":\"title,description,permissions\", \"allowMissing\":true}", "response": "{\"name\":\"roles/qa-custom-role\", \"title\":\"QA\", \"description\":\"Custom-defined QA role\", \"permissions\":[\"bb.databases.get\", \"bb.databases.getSchema\", \"bb.databases.list\", \"bb.issueComments.create\", \"bb.issues.get\", \"bb.issues.list\", \"bb.planCheckRuns.list\", \"bb.planCheckRuns.run\", \"bb.plans.get\", \"bb.plans.list\", \"bb.projects.get\", \"bb.projects.getIamPolicy\", \"bb.rollouts.get\", \"bb.taskRuns.list\"], \"type\":\"CUSTOM\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:59178", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (112, '2025-05-26 07:27:50.339726+00', '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/UpdatePolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"policy\":{\"name\":\"environments/prod/policies/tag\", \"type\":\"TAG\", \"tagPolicy\":{\"tags\":{\"bb.tag.review_config\":\"reviewConfigs/sql-review-sample-policy\"}}}, \"updateMask\":\"payload\", \"allowMissing\":true}", "response": "{\"name\":\"environments/prod/policies/tag\", \"type\":\"TAG\", \"tagPolicy\":{\"tags\":{\"bb.tag.review_config\":\"reviewConfigs/sql-review-sample-policy\"}}, \"enforce\":true, \"resourceType\":\"ENVIRONMENT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:59178", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (113, '2025-05-26 07:27:50.339901+00', '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/UpdatePolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"policy\":{\"name\":\"environments/test/policies/tag\", \"type\":\"TAG\", \"tagPolicy\":{\"tags\":{\"bb.tag.review_config\":\"reviewConfigs/sql-review-sample-policy\"}}}, \"updateMask\":\"payload\", \"allowMissing\":true}", "response": "{\"name\":\"environments/test/policies/tag\", \"type\":\"TAG\", \"tagPolicy\":{\"tags\":{\"bb.tag.review_config\":\"reviewConfigs/sql-review-sample-policy\"}}, \"enforce\":true, \"resourceType\":\"ENVIRONMENT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:59172", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (114, '2025-05-26 07:38:07.205352+00', '{"user": "users/101", "method": "/bytebase.v1.RiskService/CreateRisk", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"risk\":{\"source\":\"DDL\", \"title\":\" ALTER column in production environment is high risk\", \"level\":300, \"active\":true, \"condition\":{\"expression\":\"environment_id == \\\"prod\\\" && sql_type == \\\"ALTER_TABLE\\\"\"}}}", "response": "{\"name\":\"risks/101\", \"source\":\"DDL\", \"title\":\" ALTER column in production environment is high risk\", \"level\":300, \"active\":true, \"condition\":{\"expression\":\"environment_id == \\\"prod\\\" && sql_type == \\\"ALTER_TABLE\\\"\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:59172", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (115, '2025-05-26 07:38:48.901692+00', '{"user": "users/101", "method": "/bytebase.v1.RiskService/CreateRisk", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"risk\":{\"source\":\"DDL\", \"title\":\"CREATE TABLE in production environment is moderate risk\", \"level\":300, \"active\":true, \"condition\":{\"expression\":\"environment_id == \\\"prod\\\" && sql_type == \\\"CREATE_TABLE\\\"\"}}}", "response": "{\"name\":\"risks/102\", \"source\":\"DDL\", \"title\":\"CREATE TABLE in production environment is moderate risk\", \"level\":300, \"active\":true, \"condition\":{\"expression\":\"environment_id == \\\"prod\\\" && sql_type == \\\"CREATE_TABLE\\\"\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:59172", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (116, '2025-05-26 07:39:45.588851+00', '{"user": "users/101", "method": "/bytebase.v1.RiskService/CreateRisk", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"risk\":{\"source\":\"DML\", \"title\":\"Updated or deleted rows exceeds 100 in prod is high risk\", \"level\":300, \"active\":true, \"condition\":{\"expression\":\"environment_id == \\\"prod\\\" && affected_rows > 100 && sql_type in [\\\"UPDATE\\\", \\\"DELETE\\\"]\"}}}", "response": "{\"name\":\"risks/103\", \"source\":\"DML\", \"title\":\"Updated or deleted rows exceeds 100 in prod is high risk\", \"level\":300, \"active\":true, \"condition\":{\"expression\":\"environment_id == \\\"prod\\\" && affected_rows > 100 && sql_type in [\\\"UPDATE\\\", \\\"DELETE\\\"]\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:59172", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (117, '2025-05-26 07:40:00.03158+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {}}}, "requestMetadata": {"callerIp": "[::1]:60530", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (118, '2025-05-26 07:40:07.663554+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {"rules": [{"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == 300"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs DBA approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs Administrator approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA -> Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves."}, "condition": {}}]}}}, "requestMetadata": {"callerIp": "[::1]:60530", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (119, '2025-05-26 07:40:43.115949+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {"rules": [{"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == 300"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {"expression": "source == \"DML\" && level == 0"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs DBA approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs Administrator approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA -> Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves."}, "condition": {}}]}}}, "requestMetadata": {"callerIp": "[::1]:60530", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (122, '2025-05-26 07:40:53.534087+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {"rules": [{"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == 300 || source == \"DDL\" && level == 300"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {"expression": "source == \"DML\" && level == 0 || source == \"DDL\" && level == 200 || source == \"DDL\" &&\nlevel == 0"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs DBA approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs Administrator approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA -> Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves."}, "condition": {}}]}}}, "requestMetadata": {"callerIp": "[::1]:60530", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (123, '2025-05-26 07:41:12.810007+00', '{"user": "users/101", "method": "/bytebase.v1.WorkspaceService/SetIamPolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"resource\":\"workspaces/-\", \"policy\":{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\", \"user:demo@example.com\"], \"condition\":{}}], \"etag\":\"1748243793699\"}, \"etag\":\"1748243793699\"}", "resource": "workspaces/-", "response": "{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\", \"user:demo@example.com\"], \"condition\":{}}], \"etag\":\"1748245272807\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:60530", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (128, '2025-05-26 07:48:23.462873+00', '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/UpdatePolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"policy\":{\"name\":\"policies/masking_rule\", \"type\":\"MASKING_RULE\", \"maskingRulePolicy\":{\"rules\":[{\"id\":\"1f3018bb-f4ea-4daa-b4b3-6e32bce0d22e\", \"condition\":{\"expression\":\"classification_level in [\\\"2\\\", \\\"3\\\"]\"}, \"semanticType\":\"bb.default-partial\"}, {\"id\":\"e0743172-c9c7-43f8-9923-9a3c06012cee\", \"condition\":{\"expression\":\"classification_level in [\\\"4\\\"]\"}, \"semanticType\":\"bb.default\"}]}, \"resourceType\":\"WORKSPACE\"}, \"updateMask\":\"payload\", \"allowMissing\":true}", "response": "{\"name\":\"policies/masking_rule\", \"type\":\"MASKING_RULE\", \"maskingRulePolicy\":{\"rules\":[{\"id\":\"1f3018bb-f4ea-4daa-b4b3-6e32bce0d22e\", \"condition\":{\"expression\":\"classification_level in [\\\"2\\\", \\\"3\\\"]\"}, \"semanticType\":\"bb.default-partial\"}, {\"id\":\"e0743172-c9c7-43f8-9923-9a3c06012cee\", \"condition\":{\"expression\":\"classification_level in [\\\"4\\\"]\"}, \"semanticType\":\"bb.default\"}]}, \"enforce\":true, \"resourceType\":\"WORKSPACE\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61234", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (129, '2025-05-26 07:49:51.250743+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.watermark\", \"value\":{\"stringValue\":\"1\"}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.watermark", "response": "{\"name\":\"settings/bb.workspace.watermark\", \"value\":{\"stringValue\":\"1\"}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.watermark", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"stringValue": "0"}}, "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (120, '2025-05-26 07:40:47.856611+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {"rules": [{"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == 300 || source == \"DDL\" && level == 300"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {"expression": "source == \"DML\" && level == 0"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs DBA approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs Administrator approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA -> Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves."}, "condition": {}}]}}}, "requestMetadata": {"callerIp": "[::1]:60530", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (121, '2025-05-26 07:40:51.55467+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {"rules": [{"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == 300 || source == \"DDL\" && level == 300"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {"expression": "source == \"DML\" && level == 0 || source == \"DDL\" && level == 200"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs DBA approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs Administrator approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA -> Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves."}, "condition": {}}]}}}, "requestMetadata": {"callerIp": "[::1]:60530", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (124, '2025-05-26 07:44:16.159285+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.data-classification\", \"value\":{\"dataClassificationSettingValue\":{\"configs\":[{\"id\":\"e5680e79-e84b-486e-8cb2-76c984c3fac9\", \"title\":\"Classification Example\", \"levels\":[{\"id\":\"1\", \"title\":\"Level 1\"}, {\"id\":\"2\", \"title\":\"Level 2\"}, {\"id\":\"3\", \"title\":\"Level 3\"}, {\"id\":\"4\", \"title\":\"Level 4\"}], \"classification\":{\"1\":{\"id\":\"1\", \"title\":\"Basic\"}, \"1-1\":{\"id\":\"1-1\", \"title\":\"Basic\", \"levelId\":\"1\"}, \"1-2\":{\"id\":\"1-2\", \"title\":\"Contact\", \"levelId\":\"2\"}, \"1-3\":{\"id\":\"1-3\", \"title\":\"Health\", \"levelId\":\"4\"}, \"2\":{\"id\":\"2\", \"title\":\"Relationship\"}, \"2-1\":{\"id\":\"2-1\", \"title\":\"Social\", \"levelId\":\"1\"}, \"2-2\":{\"id\":\"2-2\", \"title\":\"Business\", \"levelId\":\"3\"}}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.data-classification", "response": "{\"name\":\"settings/bb.workspace.data-classification\", \"value\":{\"dataClassificationSettingValue\":{\"configs\":[{\"id\":\"e5680e79-e84b-486e-8cb2-76c984c3fac9\", \"title\":\"Classification Example\", \"levels\":[{\"id\":\"1\", \"title\":\"Level 1\"}, {\"id\":\"2\", \"title\":\"Level 2\"}, {\"id\":\"3\", \"title\":\"Level 3\"}, {\"id\":\"4\", \"title\":\"Level 4\"}], \"classification\":{\"1\":{\"id\":\"1\", \"title\":\"Basic\"}, \"1-1\":{\"id\":\"1-1\", \"title\":\"Basic\", \"levelId\":\"1\"}, \"1-2\":{\"id\":\"1-2\", \"title\":\"Contact\", \"levelId\":\"2\"}, \"1-3\":{\"id\":\"1-3\", \"title\":\"Health\", \"levelId\":\"4\"}, \"2\":{\"id\":\"2\", \"title\":\"Relationship\"}, \"2-1\":{\"id\":\"2-1\", \"title\":\"Social\", \"levelId\":\"1\"}, \"2-2\":{\"id\":\"2-2\", \"title\":\"Business\", \"levelId\":\"3\"}}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.data-classification", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"dataClassificationSettingValue": {}}}, "requestMetadata": {"callerIp": "[::1]:60530", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (125, '2025-05-26 07:47:11.561832+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.semantic-types\", \"value\":{\"semanticTypeSettingValue\":{\"types\":[{\"id\":\"bb.default\", \"title\":\"Default\", \"description\":\"Default type with full masking\"}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.semantic-types", "response": "{\"name\":\"settings/bb.workspace.semantic-types\", \"value\":{\"semanticTypeSettingValue\":{\"types\":[{\"id\":\"bb.default\", \"title\":\"Default\", \"description\":\"Default type with full masking\"}]}}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61234", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (126, '2025-05-26 07:47:13.546539+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.semantic-types\", \"value\":{\"semanticTypeSettingValue\":{\"types\":[{\"id\":\"bb.default\", \"title\":\"Default\", \"description\":\"Default type with full masking\"}, {\"id\":\"bb.default-partial\", \"title\":\"Default Partial\", \"description\":\"Default partial type with partial masking\"}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.semantic-types", "response": "{\"name\":\"settings/bb.workspace.semantic-types\", \"value\":{\"semanticTypeSettingValue\":{\"types\":[{\"id\":\"bb.default\", \"title\":\"Default\", \"description\":\"Default type with full masking\"}, {\"id\":\"bb.default-partial\", \"title\":\"Default Partial\", \"description\":\"Default partial type with partial masking\"}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.semantic-types", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"semanticTypeSettingValue": {"types": [{"id": "bb.default", "title": "Default", "description": "Default type with full masking"}]}}}, "requestMetadata": {"callerIp": "[::1]:61234", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (127, '2025-05-26 07:48:20.879598+00', '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/UpdatePolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"policy\":{\"name\":\"policies/masking_rule\", \"type\":\"MASKING_RULE\", \"maskingRulePolicy\":{\"rules\":[{\"id\":\"e0743172-c9c7-43f8-9923-9a3c06012cee\", \"condition\":{\"expression\":\"classification_level in [\\\"4\\\"]\"}, \"semanticType\":\"bb.default\"}]}, \"resourceType\":\"WORKSPACE\"}, \"updateMask\":\"payload\", \"allowMissing\":true}", "response": "{\"name\":\"policies/masking_rule\", \"type\":\"MASKING_RULE\", \"maskingRulePolicy\":{\"rules\":[{\"id\":\"e0743172-c9c7-43f8-9923-9a3c06012cee\", \"condition\":{\"expression\":\"classification_level in [\\\"4\\\"]\"}, \"semanticType\":\"bb.default\"}]}, \"enforce\":true, \"resourceType\":\"WORKSPACE\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61234", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (130, '2025-05-26 07:50:36.063194+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.profile\", \"value\":{\"workspaceProfileSettingValue\":{\"externalUrl\":\"https://demo.bytebase.com\", \"domains\":[\"example.com\"], \"databaseChangeMode\":\"PIPELINE\"}}}, \"allowMissing\":true, \"updateMask\":\"value.workspaceProfileSettingValue.domains\"}", "resource": "settings/bb.workspace.profile", "response": "{\"name\":\"settings/bb.workspace.profile\", \"value\":{\"workspaceProfileSettingValue\":{\"externalUrl\":\"https://demo.bytebase.com\", \"domains\":[\"example.com\"], \"databaseChangeMode\":\"PIPELINE\"}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.profile", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceProfileSettingValue": {"externalUrl": "https://demo.bytebase.com", "databaseChangeMode": "PIPELINE"}}}, "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (138, '2025-05-26 07:57:47.352245+00', '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb", "request": "{\"name\":\"instances/bytebase-meta/databases/bb\", \"statement\":\"-- Fully completed issues by project\\nSELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\", \"limit\":1000, \"dataSourceId\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"queryOption\":{\"redisRunCommandsOn\":\"SINGLE_NODE\"}}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\":[{\"error\":\"ERROR: column issue.project_id does not exist (SQLSTATE 42703)\", \"latency\":\"0.000508209s\", \"statement\":\"-- Fully completed issues by project\\nSELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\", \"postgresError\":{\"severity\":\"ERROR\", \"code\":\"42703\", \"message\":\"column issue.project_id does not exist\", \"hint\":\"Perhaps you meant to reference the column \\\"issue.project\\\".\", \"position\":115, \"file\":\"parse_relation.c\", \"line\":3729, \"routine\":\"errorMissingColumn\"}, \"allowExport\":true}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (141, '2025-05-26 08:03:43.346664+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/CreateInstance", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"instance\":{\"name\":\"instances/test-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\"}], \"environment\":\"environments/test\", \"activation\":true}, \"instanceId\":\"test-sample-instance\", \"validateOnly\":true}", "resource": "instances/test-sample-instance", "response": "{\"name\":\"instances/test-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/test\", \"activation\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (142, '2025-05-26 08:03:43.376894+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/CreateInstance", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"instance\":{\"name\":\"instances/test-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\"}], \"environment\":\"environments/test\", \"activation\":true}, \"instanceId\":\"test-sample-instance\"}", "resource": "instances/test-sample-instance", "response": "{\"name\":\"instances/test-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/test\", \"activation\":true, \"roles\":[{\"name\":\"instances/test-sample-instance/roles/bbsample\", \"roleName\":\"bbsample\", \"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (143, '2025-05-26 08:04:29.881979+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/CreateInstance", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"instance\":{\"name\":\"instances/prod-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\"}], \"environment\":\"environments/prod\", \"activation\":true}, \"instanceId\":\"prod-sample-instance\", \"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/prod\", \"activation\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (170, '2025-05-26 08:26:15.268137+00', '{"user": "users/101", "method": "/bytebase.v1.IssueService/CreateIssue", "parent": "projects/hr", "request": "{\"parent\":\"projects/hr\", \"issue\":{\"name\":\"projects/hr/issues/-101\", \"title\":\"👉👉👉 [START HERE] Add email column to Employee table\", \"type\":\"DATABASE_CHANGE\", \"status\":\"OPEN\", \"creator\":\"users/demo@example.com\", \"plan\":\"projects/hr/plans/101\", \"labels\":[\"3.6.2\", \"feature\"]}}", "response": "{\"name\":\"projects/hr/issues/101\", \"title\":\"👉👉👉 [START HERE] Add email column to Employee table\", \"type\":\"DATABASE_CHANGE\", \"status\":\"OPEN\", \"creator\":\"users/demo@example.com\", \"createTime\":\"2025-05-26T08:26:15.266310Z\", \"updateTime\":\"2025-05-26T08:26:15.266310Z\", \"plan\":\"projects/hr/plans/101\", \"labels\":[\"3.6.2\", \"feature\"]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63974", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (131, '2025-05-26 07:54:33.73121+00', '{"user": "users/101", "method": "/bytebase.v1.ProjectService/SetIamPolicy", "parent": "projects/metadb", "request": "{\"resource\":\"projects/metadb\", \"policy\":{\"bindings\":[{\"role\":\"roles/projectOwner\", \"members\":[\"user:demo@example.com\"], \"condition\":{}}, {\"role\":\"roles/sqlEditorUser\", \"members\":[\"user:dev1@example.com\", \"user:dba1@example.com\"], \"condition\":{\"title\":\"SQL Editor User All databases\"}}], \"etag\":\"1748246046446\"}, \"etag\":\"1748246046446\"}", "resource": "projects/metadb", "response": "{\"bindings\":[{\"role\":\"roles/projectOwner\", \"members\":[\"user:demo@example.com\"], \"condition\":{}}, {\"role\":\"roles/sqlEditorUser\", \"members\":[\"user:dev1@example.com\", \"user:dba1@example.com\"], \"condition\":{\"title\":\"SQL Editor User All databases\"}}], \"etag\":\"1748246073728\"}", "severity": "INFO", "serviceData": {"@type": "type.googleapis.com/bytebase.v1.AuditData", "policyDelta": {"bindingDeltas": [{"role": "roles/sqlEditorUser", "action": "ADD", "member": "users/102", "condition": {"title": "SQL Editor User All databases"}}, {"role": "roles/sqlEditorUser", "action": "ADD", "member": "users/103", "condition": {"title": "SQL Editor User All databases"}}]}}, "requestMetadata": {"callerIp": "[::1]:61234", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (132, '2025-05-26 07:56:06.908019+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/CreateInstance", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "status": {"code": 3, "message": "invalid datasource ADMIN, error failed to connect to `user=bb database=postgres`: /tmp/.s.PGSQL.5432 (/tmp): dial error: dial unix /tmp/.s.PGSQL.5432: connect: no such file or directory"}, "request": "{\"instance\":{\"name\":\"instances/bytebase-meta\", \"state\":\"ACTIVE\", \"title\":\"bytebase-meta\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"type\":\"ADMIN\", \"username\":\"bb\", \"host\":\"/tmp\", \"port\":\"5432\", \"authenticationType\":\"PASSWORD\"}], \"environment\":\"environments/prod\", \"activation\":true}, \"instanceId\":\"bytebase-meta\", \"validateOnly\":true}", "resource": "instances/bytebase-meta", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61847", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (133, '2025-05-26 07:56:20.00543+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/CreateInstance", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"instance\":{\"name\":\"instances/bytebase-meta\", \"state\":\"ACTIVE\", \"title\":\"bytebase-meta\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"type\":\"ADMIN\", \"username\":\"bb\", \"host\":\"/tmp\", \"port\":\"8082\", \"authenticationType\":\"PASSWORD\"}], \"environment\":\"environments/prod\", \"activation\":true}, \"instanceId\":\"bytebase-meta\", \"validateOnly\":true}", "resource": "instances/bytebase-meta", "response": "{\"name\":\"instances/bytebase-meta\", \"state\":\"ACTIVE\", \"title\":\"bytebase-meta\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"type\":\"ADMIN\", \"username\":\"bb\", \"host\":\"/tmp\", \"port\":\"8082\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/prod\", \"activation\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61847", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (134, '2025-05-26 07:56:43.29775+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/CreateInstance", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"instance\":{\"name\":\"instances/bytebase-meta\", \"state\":\"ACTIVE\", \"title\":\"bytebase-meta\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"type\":\"ADMIN\", \"username\":\"bb\", \"host\":\"/tmp\", \"port\":\"8082\", \"authenticationType\":\"PASSWORD\"}], \"environment\":\"environments/prod\", \"activation\":true}, \"instanceId\":\"bytebase-meta\", \"validateOnly\":true}", "resource": "instances/bytebase-meta", "response": "{\"name\":\"instances/bytebase-meta\", \"state\":\"ACTIVE\", \"title\":\"bytebase-meta\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"type\":\"ADMIN\", \"username\":\"bb\", \"host\":\"/tmp\", \"port\":\"8082\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/prod\", \"activation\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61847", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (135, '2025-05-26 07:56:43.321404+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/CreateInstance", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"instance\":{\"name\":\"instances/bytebase-meta\", \"state\":\"ACTIVE\", \"title\":\"bytebase-meta\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"type\":\"ADMIN\", \"username\":\"bb\", \"host\":\"/tmp\", \"port\":\"8082\", \"authenticationType\":\"PASSWORD\"}], \"environment\":\"environments/prod\", \"activation\":true}, \"instanceId\":\"bytebase-meta\"}", "resource": "instances/bytebase-meta", "response": "{\"name\":\"instances/bytebase-meta\", \"state\":\"ACTIVE\", \"title\":\"bytebase-meta\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"type\":\"ADMIN\", \"username\":\"bb\", \"host\":\"/tmp\", \"port\":\"8082\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/prod\", \"activation\":true, \"roles\":[{\"name\":\"instances/bytebase-meta/roles/bb\", \"roleName\":\"bb\", \"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61847", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (136, '2025-05-26 07:56:59.903341+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/BatchUpdateDatabases", "parent": "projects/default", "request": "{\"parent\":\"-\", \"requests\":[{\"database\":{\"name\":\"instances/bytebase-meta/databases/bb\", \"project\":\"projects/metadb\"}, \"updateMask\":\"project\"}]}", "resource": "-", "response": "{\"databases\":[{\"name\":\"instances/bytebase-meta/databases/bb\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T07:56:53.177942Z\", \"project\":\"projects/metadb\", \"effectiveEnvironment\":\"environments/prod\", \"instanceResource\":{\"title\":\"bytebase-meta\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"type\":\"ADMIN\", \"username\":\"bb\", \"host\":\"/tmp\", \"port\":\"8082\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/bytebase-meta\", \"environment\":\"environments/prod\"}}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61847", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (137, '2025-05-26 07:56:59.903652+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/BatchUpdateDatabases", "parent": "projects/metadb", "request": "{\"parent\":\"-\", \"requests\":[{\"database\":{\"name\":\"instances/bytebase-meta/databases/bb\", \"project\":\"projects/metadb\"}, \"updateMask\":\"project\"}]}", "resource": "-", "response": "{\"databases\":[{\"name\":\"instances/bytebase-meta/databases/bb\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T07:56:53.177942Z\", \"project\":\"projects/metadb\", \"effectiveEnvironment\":\"environments/prod\", \"instanceResource\":{\"title\":\"bytebase-meta\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"type\":\"ADMIN\", \"username\":\"bb\", \"host\":\"/tmp\", \"port\":\"8082\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/bytebase-meta\", \"environment\":\"environments/prod\"}}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61847", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (139, '2025-05-26 07:58:13.180421+00', '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb", "request": "{\"name\":\"instances/bytebase-meta/databases/bb\", \"statement\":\"-- Issues created by user\\nSELECT\\n  issue.creator_id,\\n  principal.email,\\n  COUNT(issue.creator_id) AS amount\\nFROM\\n  issue\\n  INNER JOIN principal ON issue.creator_id = principal.id\\nGROUP BY\\n  issue.creator_id,\\n  principal.email\\nORDER BY\\n  COUNT(issue.creator_id) DESC;\", \"limit\":1000, \"dataSourceId\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"queryOption\":{\"redisRunCommandsOn\":\"SINGLE_NODE\"}}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\":[{\"columnNames\":[\"creator_id\", \"email\", \"amount\"], \"columnTypeNames\":[\"INT4\", \"TEXT\", \"INT8\"], \"masked\":[false, false, false], \"sensitive\":[false, false, false], \"latency\":\"0.001358958s\", \"statement\":\"-- Issues created by user\\nSELECT\\n  issue.creator_id,\\n  principal.email,\\n  COUNT(issue.creator_id) AS amount\\nFROM\\n  issue\\n  INNER JOIN principal ON issue.creator_id = principal.id\\nGROUP BY\\n  issue.creator_id,\\n  principal.email\\nORDER BY\\n  COUNT(issue.creator_id) DESC;\", \"allowExport\":true}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (140, '2025-05-26 08:03:36.061032+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/CreateInstance", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"instance\":{\"name\":\"instances/test-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\"}], \"environment\":\"environments/test\", \"activation\":true}, \"instanceId\":\"test-sample-instance\", \"validateOnly\":true}", "resource": "instances/test-sample-instance", "response": "{\"name\":\"instances/test-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/test\", \"activation\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (145, '2025-05-26 08:04:47.263883+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "status": {"code": 3, "message": "invalid datasource READ_ONLY, error failed to connect to `user=bytebase_readonly database=postgres`: /tmp/.s.PGSQL.8084 (/tmp): server error: FATAL: role \"bytebase_readonly\" does not exist (SQLSTATE 28000)"}, "request": "{\"name\":\"instances/prod-sample-instance\", \"dataSource\":{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"type\":\"READ_ONLY\", \"username\":\"bytebase_readonly\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\"}, \"validateOnly\":true}", "resource": "instances/prod-sample-instance", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (146, '2025-05-26 08:04:55.369194+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"name\":\"instances/prod-sample-instance\", \"dataSource\":{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"type\":\"READ_ONLY\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\"}, \"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/prod\", \"activation\":true, \"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\", \"roleName\":\"bbsample\", \"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (149, '2025-05-26 08:05:18.005999+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/BatchUpdateDatabases", "parent": "projects/hr", "request": "{\"parent\":\"-\", \"requests\":[{\"database\":{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"project\":\"projects/hr\"}, \"updateMask\":\"project\"}]}", "resource": "-", "response": "{\"databases\":[{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T08:04:33.186522Z\", \"project\":\"projects/hr\", \"effectiveEnvironment\":\"environments/prod\", \"instanceResource\":{\"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}, {\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"type\":\"READ_ONLY\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/prod-sample-instance\", \"environment\":\"environments/prod\"}, \"backupAvailable\":true}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (144, '2025-05-26 08:04:29.903741+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/CreateInstance", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"instance\":{\"name\":\"instances/prod-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\"}], \"environment\":\"environments/prod\", \"activation\":true}, \"instanceId\":\"prod-sample-instance\"}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/prod\", \"activation\":true, \"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\", \"roleName\":\"bbsample\", \"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (147, '2025-05-26 08:05:03.50721+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"name\":\"instances/prod-sample-instance\", \"dataSource\":{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"type\":\"READ_ONLY\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\"}, \"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/prod\", \"activation\":true, \"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\", \"roleName\":\"bbsample\", \"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (148, '2025-05-26 08:05:03.519785+00', '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"name\":\"instances/prod-sample-instance\", \"dataSource\":{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"type\":\"READ_ONLY\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\"}}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\", \"state\":\"ACTIVE\", \"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}, {\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"type\":\"READ_ONLY\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"environment\":\"environments/prod\", \"activation\":true, \"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\", \"roleName\":\"bbsample\", \"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (150, '2025-05-26 08:05:18.00631+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/BatchUpdateDatabases", "parent": "projects/default", "request": "{\"parent\":\"-\", \"requests\":[{\"database\":{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"project\":\"projects/hr\"}, \"updateMask\":\"project\"}]}", "resource": "-", "response": "{\"databases\":[{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T08:04:33.186522Z\", \"project\":\"projects/hr\", \"effectiveEnvironment\":\"environments/prod\", \"instanceResource\":{\"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}, {\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"type\":\"READ_ONLY\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/prod-sample-instance\", \"environment\":\"environments/prod\"}, \"backupAvailable\":true}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (151, '2025-05-26 08:05:25.065384+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/BatchUpdateDatabases", "parent": "projects/hr", "request": "{\"parent\":\"-\", \"requests\":[{\"database\":{\"name\":\"instances/test-sample-instance/databases/hr_test\", \"project\":\"projects/hr\"}, \"updateMask\":\"project\"}]}", "resource": "-", "response": "{\"databases\":[{\"name\":\"instances/test-sample-instance/databases/hr_test\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T08:03:53.183103Z\", \"project\":\"projects/hr\", \"effectiveEnvironment\":\"environments/test\", \"instanceResource\":{\"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/test-sample-instance\", \"environment\":\"environments/test\"}, \"backupAvailable\":true}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (152, '2025-05-26 08:05:25.065664+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/BatchUpdateDatabases", "parent": "projects/default", "request": "{\"parent\":\"-\", \"requests\":[{\"database\":{\"name\":\"instances/test-sample-instance/databases/hr_test\", \"project\":\"projects/hr\"}, \"updateMask\":\"project\"}]}", "resource": "-", "response": "{\"databases\":[{\"name\":\"instances/test-sample-instance/databases/hr_test\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T08:03:53.183103Z\", \"project\":\"projects/hr\", \"effectiveEnvironment\":\"environments/test\", \"instanceResource\":{\"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/test-sample-instance\", \"environment\":\"environments/test\"}, \"backupAvailable\":true}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:61457", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (153, '2025-05-26 08:07:29.78582+00', '{"user": "users/101", "method": "/bytebase.v1.ProjectService/SetIamPolicy", "parent": "projects/hr", "request": "{\"resource\":\"projects/hr\", \"policy\":{\"bindings\":[{\"role\":\"roles/projectOwner\", \"members\":[\"user:demo@example.com\"], \"condition\":{}}, {\"role\":\"roles/projectDeveloper\", \"members\":[\"user:dev1@example.com\"], \"condition\":{\"title\":\"Project Developer All databases\"}}], \"etag\":\"1748242124999\"}, \"etag\":\"1748242124999\"}", "resource": "projects/hr", "response": "{\"bindings\":[{\"role\":\"roles/projectOwner\", \"members\":[\"user:demo@example.com\"], \"condition\":{}}, {\"role\":\"roles/projectDeveloper\", \"members\":[\"user:dev1@example.com\"], \"condition\":{\"title\":\"Project Developer All databases\"}}], \"etag\":\"1748246849784\"}", "severity": "INFO", "serviceData": {"@type": "type.googleapis.com/bytebase.v1.AuditData", "policyDelta": {"bindingDeltas": [{"role": "roles/projectDeveloper", "action": "ADD", "member": "users/102", "condition": {"title": "Project Developer All databases"}}]}}, "requestMetadata": {"callerIp": "[::1]:62678", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (154, '2025-05-26 08:08:54.155379+00', '{"user": "users/101", "method": "/bytebase.v1.UserService/CreateUser", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"user\":{\"name\":\"users/0\", \"email\":\"qa1@example.com\", \"title\":\"QA1\", \"userType\":\"USER\"}}", "resource": "users/0", "response": "{\"name\":\"users/105\", \"email\":\"qa1@example.com\", \"title\":\"QA1\", \"userType\":\"USER\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:62675", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (155, '2025-05-26 08:08:54.164132+00', '{"user": "users/101", "method": "/bytebase.v1.WorkspaceService/SetIamPolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"resource\":\"workspaces/-\", \"policy\":{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\", \"user:qa1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\", \"user:demo@example.com\"], \"condition\":{}}], \"etag\":\"1748245272807\"}, \"etag\":\"1748245272807\"}", "resource": "workspaces/-", "response": "{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\", \"user:qa1@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\", \"user:demo@example.com\"], \"condition\":{}}], \"etag\":\"1748246934163\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:62675", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (156, '2025-05-26 08:09:14.424273+00', '{"user": "users/101", "method": "/bytebase.v1.UserService/CreateUser", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"user\":{\"name\":\"users/0\", \"email\":\"qa2@example.com\", \"title\":\"QA2\", \"userType\":\"USER\"}}", "resource": "users/0", "response": "{\"name\":\"users/106\", \"email\":\"qa2@example.com\", \"title\":\"QA2\", \"userType\":\"USER\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:62675", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (157, '2025-05-26 08:09:14.43372+00', '{"user": "users/101", "method": "/bytebase.v1.WorkspaceService/SetIamPolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"resource\":\"workspaces/-\", \"policy\":{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\", \"user:qa1@example.com\", \"user:qa2@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\", \"user:demo@example.com\"], \"condition\":{}}], \"etag\":\"1748246934163\"}, \"etag\":\"1748246934163\"}", "resource": "workspaces/-", "response": "{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\", \"user:qa1@example.com\", \"user:qa2@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\", \"user:demo@example.com\"], \"condition\":{}}], \"etag\":\"1748246954432\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:62675", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (158, '2025-05-26 08:09:47.440116+00', '{"user": "users/101", "method": "/bytebase.v1.GroupService/CreateGroup", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"group\":{\"name\":\"groups/qa-group@example.com\", \"title\":\"QA Group\", \"members\":[{\"member\":\"users/qa1@example.com\", \"role\":\"OWNER\"}, {\"member\":\"users/qa2@example.com\", \"role\":\"MEMBER\"}]}, \"groupEmail\":\"qa-group@example.com\"}", "response": "{\"name\":\"groups/qa-group@example.com\", \"title\":\"QA Group\", \"members\":[{\"member\":\"users/qa1@example.com\", \"role\":\"OWNER\"}, {\"member\":\"users/qa2@example.com\", \"role\":\"MEMBER\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:62675", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (159, '2025-05-26 08:10:00.959856+00', '{"user": "users/101", "method": "/bytebase.v1.WorkspaceService/SetIamPolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"resource\":\"workspaces/-\", \"policy\":{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\", \"user:qa1@example.com\", \"user:qa2@example.com\", \"group:qa-group@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\", \"user:demo@example.com\"], \"condition\":{}}, {\"role\":\"roles/qa-custom-role\", \"members\":[\"group:qa-group@example.com\"]}], \"etag\":\"1748246954432\"}, \"etag\":\"1748246954432\"}", "resource": "workspaces/-", "response": "{\"bindings\":[{\"role\":\"roles/workspaceMember\", \"members\":[\"allUsers\", \"user:dev1@example.com\", \"user:qa1@example.com\", \"user:qa2@example.com\", \"group:qa-group@example.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceAdmin\", \"members\":[\"user:demo@example.com\", \"user:api@service.bytebase.com\"], \"condition\":{}}, {\"role\":\"roles/workspaceDBA\", \"members\":[\"user:dba1@example.com\", \"user:demo@example.com\"], \"condition\":{}}, {\"role\":\"roles/qa-custom-role\", \"members\":[\"group:qa-group@example.com\"], \"condition\":{}}], \"etag\":\"1748247000958\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:62675", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (160, '2025-05-26 08:14:34.184265+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.data-classification\", \"value\":{\"dataClassificationSettingValue\":{\"configs\":[{\"id\":\"e5680e79-e84b-486e-8cb2-76c984c3fac9\", \"title\":\"Classification Example\", \"levels\":[{\"id\":\"1\", \"title\":\"Level 1\"}, {\"id\":\"2\", \"title\":\"Level 2\"}, {\"id\":\"3\", \"title\":\"Level 3\"}, {\"id\":\"4\", \"title\":\"Level 4\"}], \"classification\":{\"1\":{\"id\":\"1\", \"title\":\"Basic\"}, \"1-1\":{\"id\":\"1-1\", \"title\":\"Basic\", \"levelId\":\"1\"}, \"1-2\":{\"id\":\"1-2\", \"title\":\"Contact\", \"levelId\":\"2\"}, \"1-3\":{\"id\":\"1-3\", \"title\":\"Health\", \"levelId\":\"4\"}, \"2\":{\"id\":\"2\", \"title\":\"Relationship\"}, \"2-1\":{\"id\":\"2-1\", \"title\":\"Social\", \"levelId\":\"1\"}, \"2-2\":{\"id\":\"2-2\", \"title\":\"Business\", \"levelId\":\"3\"}}, \"classificationFromConfig\":true}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.data-classification", "response": "{\"name\":\"settings/bb.workspace.data-classification\", \"value\":{\"dataClassificationSettingValue\":{\"configs\":[{\"id\":\"e5680e79-e84b-486e-8cb2-76c984c3fac9\", \"title\":\"Classification Example\", \"levels\":[{\"id\":\"1\", \"title\":\"Level 1\"}, {\"id\":\"2\", \"title\":\"Level 2\"}, {\"id\":\"3\", \"title\":\"Level 3\"}, {\"id\":\"4\", \"title\":\"Level 4\"}], \"classification\":{\"1\":{\"id\":\"1\", \"title\":\"Basic\"}, \"1-1\":{\"id\":\"1-1\", \"title\":\"Basic\", \"levelId\":\"1\"}, \"1-2\":{\"id\":\"1-2\", \"title\":\"Contact\", \"levelId\":\"2\"}, \"1-3\":{\"id\":\"1-3\", \"title\":\"Health\", \"levelId\":\"4\"}, \"2\":{\"id\":\"2\", \"title\":\"Relationship\"}, \"2-1\":{\"id\":\"2-1\", \"title\":\"Social\", \"levelId\":\"1\"}, \"2-2\":{\"id\":\"2-2\", \"title\":\"Business\", \"levelId\":\"3\"}}, \"classificationFromConfig\":true}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.data-classification", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"dataClassificationSettingValue": {"configs": [{"id": "e5680e79-e84b-486e-8cb2-76c984c3fac9", "title": "Classification Example", "levels": [{"id": "1", "title": "Level 1"}, {"id": "2", "title": "Level 2"}, {"id": "3", "title": "Level 3"}, {"id": "4", "title": "Level 4"}], "classification": {"1": {"id": "1", "title": "Basic"}, "2": {"id": "2", "title": "Relationship"}, "1-1": {"id": "1-1", "title": "Basic", "levelId": "1"}, "1-2": {"id": "1-2", "title": "Contact", "levelId": "2"}, "1-3": {"id": "1-3", "title": "Health", "levelId": "4"}, "2-1": {"id": "2-1", "title": "Social", "levelId": "1"}, "2-2": {"id": "2-2", "title": "Business", "levelId": "3"}}}]}}}, "requestMetadata": {"callerIp": "[::1]:63103", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (161, '2025-05-26 08:14:52.612657+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseCatalogService/UpdateDatabaseCatalog", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"catalog\":{\"name\":\"instances/prod-sample-instance/databases/hr_prod/catalog\", \"schemas\":[{\"name\":\"public\", \"tables\":[{\"name\":\"employee\", \"columns\":{\"columns\":[{\"name\":\"first_name\", \"classification\":\"1-2\"}]}}]}]}}", "resource": "instances/prod-sample-instance/databases/hr_prod/catalog", "response": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod/catalog\", \"schemas\":[{\"name\":\"public\", \"tables\":[{\"name\":\"employee\", \"columns\":{\"columns\":[{\"name\":\"first_name\", \"classification\":\"1-2\"}]}}]}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63103", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (162, '2025-05-26 08:14:55.969774+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseCatalogService/UpdateDatabaseCatalog", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"catalog\":{\"name\":\"instances/prod-sample-instance/databases/hr_prod/catalog\", \"schemas\":[{\"name\":\"public\", \"tables\":[{\"name\":\"employee\", \"columns\":{\"columns\":[{\"name\":\"first_name\", \"classification\":\"1-2\"}, {\"name\":\"last_name\", \"classification\":\"1-2\"}]}}]}]}}", "resource": "instances/prod-sample-instance/databases/hr_prod/catalog", "response": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod/catalog\", \"schemas\":[{\"name\":\"public\", \"tables\":[{\"name\":\"employee\", \"columns\":{\"columns\":[{\"name\":\"first_name\", \"classification\":\"1-2\"}, {\"name\":\"last_name\", \"classification\":\"1-2\"}]}}]}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63103", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (165, '2025-05-26 08:15:49.543617+00', '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/hr", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"statement\":\"SELECT * FROM employee\", \"limit\":1000, \"dataSourceId\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"queryOption\":{\"redisRunCommandsOn\":\"SINGLE_NODE\"}}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\":[{\"columnNames\":[\"emp_no\", \"birth_date\", \"first_name\", \"last_name\", \"gender\", \"hire_date\"], \"columnTypeNames\":[\"INT4\", \"DATE\", \"TEXT\", \"TEXT\", \"TEXT\", \"DATE\"], \"rowsCount\":\"1000\", \"masked\":[false, false, true, true, false, false], \"sensitive\":[false, false, true, true, false, false], \"latency\":\"0.005566459s\", \"statement\":\"WITH result AS (\\nSELECT * FROM employee\\n) SELECT * FROM result LIMIT 1000;\", \"allowExport\":true}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63110", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (163, '2025-05-26 08:15:29.44377+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseCatalogService/UpdateDatabaseCatalog", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"catalog\":{\"name\":\"instances/prod-sample-instance/databases/hr_prod/catalog\", \"schemas\":[{\"name\":\"public\", \"tables\":[{\"name\":\"employee\", \"columns\":{\"columns\":[{\"name\":\"first_name\", \"classification\":\"1-2\"}, {\"name\":\"last_name\", \"classification\":\"1-2\"}]}}, {\"name\":\"salary\", \"columns\":{\"columns\":[{\"name\":\"amount\", \"semanticType\":\"bb.default\"}]}}]}]}}", "resource": "instances/prod-sample-instance/databases/hr_prod/catalog", "response": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod/catalog\", \"schemas\":[{\"name\":\"public\", \"tables\":[{\"name\":\"employee\", \"columns\":{\"columns\":[{\"name\":\"first_name\", \"classification\":\"1-2\"}, {\"name\":\"last_name\", \"classification\":\"1-2\"}]}}, {\"name\":\"salary\", \"columns\":{\"columns\":[{\"name\":\"amount\", \"semanticType\":\"bb.default\"}]}}]}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63103", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (164, '2025-05-26 08:15:44.970189+00', '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/hr", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"statement\":\"SELECT * FROM salary;\", \"limit\":1000, \"dataSourceId\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"queryOption\":{\"redisRunCommandsOn\":\"SINGLE_NODE\"}}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\":[{\"columnNames\":[\"emp_no\", \"amount\", \"from_date\", \"to_date\"], \"columnTypeNames\":[\"INT4\", \"INT4\", \"DATE\", \"DATE\"], \"rowsCount\":\"1000\", \"masked\":[false, true, false, false], \"sensitive\":[false, true, false, false], \"latency\":\"0.003390584s\", \"statement\":\"WITH result AS (\\nSELECT * FROM salary\\n) SELECT * FROM result LIMIT 1000;\", \"allowExport\":true}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63103", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (166, '2025-05-26 08:23:40.127402+00', '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/UpdatePolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"policy\":{\"name\":\"environments/prod/policies/data_source_query\", \"type\":\"DATA_SOURCE_QUERY\", \"dataSourceQueryPolicy\":{\"disallowDdl\":true, \"disallowDml\":true}}, \"updateMask\":\"payload\", \"allowMissing\":true}", "response": "{\"name\":\"environments/prod/policies/data_source_query\", \"type\":\"DATA_SOURCE_QUERY\", \"dataSourceQueryPolicy\":{\"disallowDdl\":true, \"disallowDml\":true}, \"enforce\":true, \"resourceType\":\"ENVIRONMENT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63481", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (168, '2025-05-26 08:23:44.737109+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.environment\", \"value\":{\"environmentSetting\":{\"environments\":[{\"name\":\"environments/test\", \"id\":\"test\", \"title\":\"Test\"}, {\"name\":\"environments/prod\", \"id\":\"prod\", \"title\":\"Prod\", \"tags\":{\"protected\":\"protected\"}}]}}}, \"updateMask\":\"environmentSetting\"}", "resource": "settings/bb.workspace.environment", "response": "{\"name\":\"settings/bb.workspace.environment\", \"value\":{\"environmentSetting\":{\"environments\":[{\"name\":\"environments/test\", \"id\":\"test\", \"title\":\"Test\"}, {\"name\":\"environments/prod\", \"id\":\"prod\", \"title\":\"Prod\", \"tags\":{\"protected\":\"protected\"}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.environment", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"environmentSetting": {"environments": [{"id": "test", "name": "environments/test", "title": "Test"}, {"id": "prod", "name": "environments/prod", "title": "Prod"}]}}}, "requestMetadata": {"callerIp": "[::1]:63481", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (169, '2025-05-26 08:26:15.25803+00', '{"user": "users/101", "method": "/bytebase.v1.PlanService/CreatePlan", "parent": "projects/hr", "request": "{\"parent\":\"projects/hr\", \"plan\":{\"name\":\"projects/hr/plans/-102\", \"steps\":[{\"specs\":[{\"earliestAllowedTime\":\"2030-05-25T18:00:00Z\", \"id\":\"036e8f34-05e6-4916-ba58-45c6a452fc60\", \"changeDatabaseConfig\":{\"target\":\"instances/test-sample-instance/databases/hr_test\", \"sheet\":\"projects/hr/sheets/101\", \"type\":\"MIGRATE\"}}]}, {\"specs\":[{\"id\":\"ea634b72-91e0-48b5-9ab6-ecc8d9355ff8\", \"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod\", \"sheet\":\"projects/hr/sheets/101\", \"type\":\"MIGRATE\"}}]}]}}", "response": "{\"name\":\"projects/hr/plans/101\", \"steps\":[{\"specs\":[{\"earliestAllowedTime\":\"2030-05-25T18:00:00Z\", \"id\":\"036e8f34-05e6-4916-ba58-45c6a452fc60\", \"specReleaseSource\":{}, \"changeDatabaseConfig\":{\"target\":\"instances/test-sample-instance/databases/hr_test\", \"sheet\":\"projects/hr/sheets/101\", \"type\":\"MIGRATE\", \"preUpdateBackupDetail\":{}}}]}, {\"specs\":[{\"id\":\"ea634b72-91e0-48b5-9ab6-ecc8d9355ff8\", \"specReleaseSource\":{}, \"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod\", \"sheet\":\"projects/hr/sheets/101\", \"type\":\"MIGRATE\", \"preUpdateBackupDetail\":{}}}]}], \"creator\":\"users/demo@example.com\", \"createTime\":\"2025-05-26T08:26:15.250094Z\", \"updateTime\":\"2025-05-26T08:26:15.250094Z\", \"releaseSource\":{}, \"deployment\":{\"environments\":[\"test\", \"prod\"]}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63974", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (171, '2025-05-26 08:26:15.276663+00', '{"user": "users/101", "method": "/bytebase.v1.RolloutService/CreateRollout", "parent": "projects/hr", "request": "{\"parent\":\"projects/hr\", \"rollout\":{\"plan\":\"projects/hr/plans/101\"}}", "response": "{\"name\":\"projects/hr/rollouts/101\", \"plan\":\"projects/hr/plans/101\", \"stages\":[{\"name\":\"projects/hr/rollouts/101/stages/101\", \"environment\":\"environments/test\", \"tasks\":[{\"name\":\"projects/hr/rollouts/101/stages/101/tasks/101\", \"specId\":\"036e8f34-05e6-4916-ba58-45c6a452fc60\", \"status\":\"NOT_STARTED\", \"type\":\"DATABASE_SCHEMA_UPDATE\", \"target\":\"instances/test-sample-instance/databases/hr_test\", \"databaseSchemaUpdate\":{\"sheet\":\"projects/hr/sheets/101\"}}]}, {\"name\":\"projects/hr/rollouts/101/stages/102\", \"environment\":\"environments/prod\", \"tasks\":[{\"name\":\"projects/hr/rollouts/101/stages/102/tasks/102\", \"specId\":\"ea634b72-91e0-48b5-9ab6-ecc8d9355ff8\", \"status\":\"NOT_STARTED\", \"type\":\"DATABASE_SCHEMA_UPDATE\", \"target\":\"instances/prod-sample-instance/databases/hr_prod\", \"databaseSchemaUpdate\":{\"sheet\":\"projects/hr/sheets/101\"}}]}], \"creator\":\"users/demo@example.com\", \"createTime\":\"2025-05-26T08:26:15.272294Z\", \"issue\":\"projects/hr/issues/101\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63974", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (172, '2025-05-26 08:27:15.379272+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/UpdateDatabase", "parent": "projects/hr", "request": "{\"database\":{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T08:04:33.186522Z\", \"project\":\"projects/hr\", \"environment\":\"environments/prod\", \"effectiveEnvironment\":\"environments/prod\", \"instanceResource\":{\"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}, {\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"type\":\"READ_ONLY\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/prod-sample-instance\", \"environment\":\"environments/prod\"}, \"backupAvailable\":true}, \"updateMask\":\"drifted\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T08:04:33.186522Z\", \"project\":\"projects/hr\", \"effectiveEnvironment\":\"environments/prod\", \"instanceResource\":{\"title\":\"Prod Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}, {\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\", \"type\":\"READ_ONLY\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8084\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/prod-sample-instance\", \"environment\":\"environments/prod\"}, \"backupAvailable\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63974", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (173, '2025-05-26 08:27:35.2587+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/UpdateDatabase", "parent": "projects/hr", "request": "{\"database\":{\"name\":\"instances/test-sample-instance/databases/hr_test\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T08:03:53.183103Z\", \"project\":\"projects/hr\", \"environment\":\"environments/test\", \"effectiveEnvironment\":\"environments/test\", \"instanceResource\":{\"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/test-sample-instance\", \"environment\":\"environments/test\"}, \"backupAvailable\":true}, \"updateMask\":\"drifted\"}", "resource": "instances/test-sample-instance/databases/hr_test", "response": "{\"name\":\"instances/test-sample-instance/databases/hr_test\", \"state\":\"ACTIVE\", \"successfulSyncTime\":\"2025-05-26T08:03:53.183103Z\", \"project\":\"projects/hr\", \"effectiveEnvironment\":\"environments/test\", \"instanceResource\":{\"title\":\"Test Sample Instance\", \"engine\":\"POSTGRES\", \"engineVersion\":\"16.0.0\", \"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\", \"type\":\"ADMIN\", \"username\":\"bbsample\", \"host\":\"/tmp\", \"port\":\"8083\", \"authenticationType\":\"PASSWORD\", \"redisType\":\"STANDALONE\"}], \"activation\":true, \"name\":\"instances/test-sample-instance\", \"environment\":\"environments/test\"}, \"backupAvailable\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:63859", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (174, '2025-05-26 08:31:19.335918+00', '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb", "request": "{\"name\":\"instances/bytebase-meta/databases/bb\", \"statement\":\"-- Issues created by user\\nSELECT\\n  issue.creator_id,\\n  principal.email,\\n  COUNT(issue.creator_id) AS amount\\nFROM\\n  issue\\n  INNER JOIN principal ON issue.creator_id = principal.id\\nGROUP BY\\n  issue.creator_id,\\n  principal.email\\nORDER BY\\n  COUNT(issue.creator_id) DESC;\", \"limit\":1000, \"dataSourceId\":\"35a64b4a-543f-4eac-ad32-b191a958c66d\", \"queryOption\":{\"redisRunCommandsOn\":\"SINGLE_NODE\"}}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\":[{\"columnNames\":[\"creator_id\", \"email\", \"amount\"], \"columnTypeNames\":[\"INT4\", \"TEXT\", \"INT8\"], \"rowsCount\":\"1\", \"masked\":[false, false, false], \"sensitive\":[false, false, false], \"latency\":\"0.001571042s\", \"statement\":\"-- Issues created by user\\nSELECT\\n  issue.creator_id,\\n  principal.email,\\n  COUNT(issue.creator_id) AS amount\\nFROM\\n  issue\\n  INNER JOIN principal ON issue.creator_id = principal.id\\nGROUP BY\\n  issue.creator_id,\\n  principal.email\\nORDER BY\\n  COUNT(issue.creator_id) DESC;\", \"allowExport\":true}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:64314", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (175, '2025-05-26 08:47:09.084937+00', '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/UpdatePolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"policy\":{\"name\":\"environments/test/policies/tag\", \"type\":\"TAG\", \"tagPolicy\":{\"tags\":{\"bb.tag.review_config\":\"reviewConfigs/sql-review-sample-policy\"}}}, \"updateMask\":\"payload\", \"allowMissing\":true}", "response": "{\"name\":\"environments/test/policies/tag\", \"type\":\"TAG\", \"tagPolicy\":{\"tags\":{\"bb.tag.review_config\":\"reviewConfigs/sql-review-sample-policy\"}}, \"enforce\":true, \"resourceType\":\"ENVIRONMENT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49782", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (176, '2025-05-26 08:47:09.084988+00', '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/UpdatePolicy", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"policy\":{\"name\":\"environments/prod/policies/tag\", \"type\":\"TAG\", \"tagPolicy\":{\"tags\":{\"bb.tag.review_config\":\"reviewConfigs/sql-review-sample-policy\"}}}, \"updateMask\":\"payload\", \"allowMissing\":true}", "response": "{\"name\":\"environments/prod/policies/tag\", \"type\":\"TAG\", \"tagPolicy\":{\"tags\":{\"bb.tag.review_config\":\"reviewConfigs/sql-review-sample-policy\"}}, \"enforce\":true, \"resourceType\":\"ENVIRONMENT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49799", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (177, '2025-05-26 08:47:41.77037+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200 || source == \\\"DATA_EXPORT\\\" &&\\nlevel == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200 || source == \\\"DATA_EXPORT\\\" &&\\nlevel == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {"rules": [{"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == 300 || source == \"DDL\" && level == 300"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {"expression": "source == \"DML\" && level == 0 || source == \"DDL\" && level == 200 || source == \"DDL\" &&\nlevel == 0 || source == \"DML\" && level == 200"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs DBA approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs Administrator approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA -> Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves."}, "condition": {}}]}}}, "requestMetadata": {"callerIp": "[::1]:49799", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (178, '2025-05-26 08:47:45.801694+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200 || source == \\\"DATA_EXPORT\\\" &&\\nlevel == 0 || source == \\\"REQUEST_QUERY\\\" && level == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200 || source == \\\"DATA_EXPORT\\\" &&\\nlevel == 0 || source == \\\"REQUEST_QUERY\\\" && level == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {"rules": [{"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == 300 || source == \"DDL\" && level == 300"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {"expression": "source == \"DML\" && level == 0 || source == \"DDL\" && level == 200 || source == \"DDL\" &&\nlevel == 0 || source == \"DML\" && level == 200 || source == \"DATA_EXPORT\" &&\nlevel == 0"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs DBA approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs Administrator approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA -> Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves."}, "condition": {}}]}}}, "requestMetadata": {"callerIp": "[::1]:49799", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (179, '2025-05-26 08:47:49.174483+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200 || source == \\\"DATA_EXPORT\\\" &&\\nlevel == 0 || source == \\\"REQUEST_QUERY\\\" && level == 0 || source == \\\"REQUEST_EXPORT\\\" &&\\nlevel == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200 || source == \\\"DATA_EXPORT\\\" &&\\nlevel == 0 || source == \\\"REQUEST_QUERY\\\" && level == 0 || source == \\\"REQUEST_EXPORT\\\" &&\\nlevel == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {"rules": [{"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == 300 || source == \"DDL\" && level == 300"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {"expression": "source == \"DML\" && level == 0 || source == \"DDL\" && level == 200 || source == \"DDL\" &&\nlevel == 0 || source == \"DML\" && level == 200 || source == \"DATA_EXPORT\" &&\nlevel == 0 || source == \"REQUEST_QUERY\" && level == 0"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs DBA approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs Administrator approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA -> Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves."}, "condition": {}}]}}}, "requestMetadata": {"callerIp": "[::1]:49799", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (180, '2025-05-26 08:48:05.008683+00', '{"user": "users/101", "method": "/bytebase.v1.SettingService/UpdateSetting", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"setting\":{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200 || source == \\\"DATA_EXPORT\\\" &&\\nlevel == 0 || source == \\\"REQUEST_QUERY\\\" && level == 0 || source == \\\"REQUEST_EXPORT\\\" &&\\nlevel == 0 || source == \\\"CREATE_DATABASE\\\" && level == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}, \"allowMissing\":true}", "resource": "settings/bb.workspace.approval", "response": "{\"name\":\"settings/bb.workspace.approval\", \"value\":{\"workspaceApprovalSettingValue\":{\"rules\":[{\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Project Owner -> Workspace DBA\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 300 || source == \\\"DDL\\\" && level == 300\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}]}, \"title\":\"Project Owner\", \"description\":\"The system defines the approval process and only needs the project Owner to approve it.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{\"expression\":\"source == \\\"DML\\\" && level == 0 || source == \\\"DDL\\\" && level == 200 || source == \\\"DDL\\\" &&\\nlevel == 0 || source == \\\"DML\\\" && level == 200 || source == \\\"DATA_EXPORT\\\" &&\\nlevel == 0 || source == \\\"REQUEST_QUERY\\\" && level == 0 || source == \\\"REQUEST_EXPORT\\\" &&\\nlevel == 0 || source == \\\"CREATE_DATABASE\\\" && level == 0\"}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}]}, \"title\":\"Workspace DBA\", \"description\":\"The system defines the approval process and only needs DBA approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Workspace Admin\", \"description\":\"The system defines the approval process and only needs Administrator approval.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}, {\"template\":{\"flow\":{\"steps\":[{\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/projectOwner\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceDBA\"}]}, {\"type\":\"ANY\", \"nodes\":[{\"type\":\"ANY_IN_GROUP\", \"role\":\"roles/workspaceAdmin\"}]}]}, \"title\":\"Project Owner -> Workspace DBA -> Workspace Admin\", \"description\":\"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.\", \"creator\":\"users/support@bytebase.com\"}, \"condition\":{}}]}}}", "severity": "INFO", "serviceData": {"name": "settings/bb.workspace.approval", "@type": "type.googleapis.com/bytebase.v1.Setting", "value": {"workspaceApprovalSettingValue": {"rules": [{"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == 300 || source == \"DDL\" && level == 300"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {"expression": "source == \"DML\" && level == 0 || source == \"DDL\" && level == 200 || source == \"DDL\" &&\nlevel == 0 || source == \"DML\" && level == 200 || source == \"DATA_EXPORT\" &&\nlevel == 0 || source == \"REQUEST_QUERY\" && level == 0 || source == \"REQUEST_EXPORT\" &&\nlevel == 0"}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace DBA", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs DBA approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process and only needs Administrator approval."}, "condition": {}}, {"template": {"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/projectOwner", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceDBA", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"role": "roles/workspaceAdmin", "type": "ANY_IN_GROUP"}]}]}, "title": "Project Owner -> Workspace DBA -> Workspace Admin", "creator": "users/support@bytebase.com", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves."}, "condition": {}}]}}}, "requestMetadata": {"callerIp": "[::1]:49799", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (181, '2025-06-05 07:04:57.22349+00', '{"user": "users/101", "method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"email\":\"demo@example.com\", \"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\", \"email\":\"demo@example.com\", \"title\":\"Demo\", \"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:58691", "callerSuppliedUserAgent": "grpc-go/1.72.2"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (182, '2025-06-09 02:20:35.270694+00', '{"user": "users/101", "method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\",\"email\":\"demo@example.com\",\"title\":\"Demo\",\"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:56360", "callerSuppliedUserAgent": "grpc-go/1.72.2"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (183, '2025-06-09 02:20:50.345446+00', '{"user": "users/101", "method": "/bytebase.v1.RiskService/UpdateRisk", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "request": "{\"risk\":{\"name\":\"risks/102\",\"source\":\"DDL\",\"title\":\"CREATE TABLE in production environment is moderate risk\",\"level\":200,\"active\":true,\"condition\":{\"expression\":\"environment_id == \\\"prod\\\" && sql_type == \\\"CREATE_TABLE\\\"\"}},\"updateMask\":\"title,level,active,condition,source\"}", "resource": "risks/102", "response": "{\"name\":\"risks/102\",\"source\":\"DDL\",\"title\":\"CREATE TABLE in production environment is moderate risk\",\"level\":200,\"active\":true,\"condition\":{\"expression\":\"environment_id == \\\"prod\\\" && sql_type == \\\"CREATE_TABLE\\\"\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:56353", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (184, '2025-06-09 02:21:04.278663+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/UpdateDatabase", "parent": "projects/hr", "request": "{\"database\":{\"name\":\"instances/prod-sample-instance/databases/hr_prod\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-06-05T07:08:02.301195Z\",\"project\":\"projects/hr\",\"environment\":\"environments/prod\",\"effectiveEnvironment\":\"environments/prod\",\"instanceResource\":{\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/prod-sample-instance\",\"environment\":\"environments/prod\"},\"backupAvailable\":true},\"updateMask\":\"drifted\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-06-05T07:08:02.301195Z\",\"project\":\"projects/hr\",\"effectiveEnvironment\":\"environments/prod\",\"instanceResource\":{\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/prod-sample-instance\",\"environment\":\"environments/prod\"},\"backupAvailable\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:56354", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (185, '2025-06-09 02:21:07.597633+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/UpdateDatabase", "parent": "projects/hr", "request": "{\"database\":{\"name\":\"instances/test-sample-instance/databases/hr_test\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-06-05T07:08:02.306591Z\",\"project\":\"projects/hr\",\"environment\":\"environments/test\",\"effectiveEnvironment\":\"environments/test\",\"instanceResource\":{\"title\":\"Test Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8083\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/test-sample-instance\",\"environment\":\"environments/test\"},\"backupAvailable\":true},\"updateMask\":\"drifted\"}", "resource": "instances/test-sample-instance/databases/hr_test", "response": "{\"name\":\"instances/test-sample-instance/databases/hr_test\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-06-05T07:08:02.306591Z\",\"project\":\"projects/hr\",\"effectiveEnvironment\":\"environments/test\",\"instanceResource\":{\"title\":\"Test Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8083\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/test-sample-instance\",\"environment\":\"environments/test\"},\"backupAvailable\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:56353", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (186, '2025-09-09 03:25:51.338104+00', '{"user": "users/101", "method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "latency": "0.092258458s", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\",\"email\":\"demo@example.com\",\"title\":\"Demo\",\"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (187, '2025-09-09 03:27:04.928131+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/UpdateDatabase", "parent": "projects/hr", "latency": "0.045281250s", "request": "{\"database\":{\"name\":\"instances/prod-sample-instance/databases/hr_prod\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-06-05T07:08:02.301195Z\",\"project\":\"projects/hr\",\"environment\":\"environments/prod\",\"effectiveEnvironment\":\"environments/prod\",\"instanceResource\":{\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/prod-sample-instance\",\"environment\":\"environments/prod\"},\"backupAvailable\":true},\"updateMask\":\"drifted\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-09-09T03:27:04.921330Z\",\"project\":\"projects/hr\",\"effectiveEnvironment\":\"environments/prod\",\"instanceResource\":{\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/prod-sample-instance\",\"environment\":\"environments/prod\"},\"backupAvailable\":true}", "severity": "INFO", "requestMetadata": {"callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (188, '2025-09-09 03:27:11.850022+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/UpdateDatabase", "parent": "projects/hr", "latency": "0.049809208s", "request": "{\"database\":{\"name\":\"instances/test-sample-instance/databases/hr_test\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-06-05T07:08:02.306591Z\",\"project\":\"projects/hr\",\"environment\":\"environments/test\",\"effectiveEnvironment\":\"environments/test\",\"instanceResource\":{\"title\":\"Test Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8083\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/test-sample-instance\",\"environment\":\"environments/test\"},\"backupAvailable\":true},\"updateMask\":\"drifted\"}", "resource": "instances/test-sample-instance/databases/hr_test", "response": "{\"name\":\"instances/test-sample-instance/databases/hr_test\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-09-09T03:27:11.843520Z\",\"project\":\"projects/hr\",\"effectiveEnvironment\":\"environments/test\",\"instanceResource\":{\"title\":\"Test Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8083\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/test-sample-instance\",\"environment\":\"environments/test\"},\"backupAvailable\":true}", "severity": "INFO", "requestMetadata": {"callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (189, '2025-10-10 07:13:53.406615+00', '{"user": "users/101", "method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "latency": "0.078084416s", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\",\"email\":\"demo@example.com\",\"title\":\"Demo\",\"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (190, '2025-10-10 07:15:40.836602+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/UpdateDatabase", "parent": "projects/hr", "latency": "0.029922417s", "request": "{\"database\":{\"name\":\"instances/prod-sample-instance/databases/hr_prod\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-10-10T07:14:57.127856Z\",\"project\":\"projects/hr\",\"environment\":\"environments/prod\",\"effectiveEnvironment\":\"environments/prod\",\"instanceResource\":{\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/prod-sample-instance\",\"environment\":\"environments/prod\"},\"backupAvailable\":true},\"updateMask\":\"drifted\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-10-10T07:15:40.832105Z\",\"project\":\"projects/hr\",\"effectiveEnvironment\":\"environments/prod\",\"instanceResource\":{\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"9af4f227-a55e-4e82-b7f5-c7193b5f405c\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"e700ae12-173e-4f0d-8590-0414cf6a9405\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/prod-sample-instance\",\"environment\":\"environments/prod\"},\"backupAvailable\":true}", "severity": "INFO", "requestMetadata": {"callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (191, '2025-10-10 07:15:45.142666+00', '{"user": "users/101", "method": "/bytebase.v1.DatabaseService/UpdateDatabase", "parent": "projects/hr", "latency": "0.037116417s", "request": "{\"database\":{\"name\":\"instances/test-sample-instance/databases/hr_test\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-10-10T07:14:57.127853Z\",\"project\":\"projects/hr\",\"environment\":\"environments/test\",\"effectiveEnvironment\":\"environments/test\",\"instanceResource\":{\"title\":\"Test Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8083\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/test-sample-instance\",\"environment\":\"environments/test\"},\"backupAvailable\":true},\"updateMask\":\"drifted\"}", "resource": "instances/test-sample-instance/databases/hr_test", "response": "{\"name\":\"instances/test-sample-instance/databases/hr_test\",\"state\":\"ACTIVE\",\"successfulSyncTime\":\"2025-10-10T07:15:45.139052Z\",\"project\":\"projects/hr\",\"effectiveEnvironment\":\"environments/test\",\"instanceResource\":{\"title\":\"Test Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.0\",\"dataSources\":[{\"id\":\"a7f206f9-37c4-41ca-8b59-fcf4c6148105\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8083\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"activation\":true,\"name\":\"instances/test-sample-instance\",\"environment\":\"environments/test\"},\"backupAvailable\":true}", "severity": "INFO", "requestMetadata": {"callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_at, payload) VALUES (192, '2025-10-13 22:55:22.871747+00', '{"user": "users/101", "method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/a6b014b9-d0d4-4974-9be6-53ec61ea5f48", "latency": "0.092679833s", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\",\"email\":\"demo@example.com\",\"title\":\"Demo\",\"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;


--
-- Data for Name: changelog; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.changelog (id, created_at, instance, db_name, status, prev_sync_history_id, sync_history_id, payload) VALUES (101, '2025-05-26 08:27:15.376781+00', 'prod-sample-instance', 'hr_prod', 'DONE', 101, 101, '{"type": "BASELINE", "gitCommit": "unknown"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, instance, db_name, status, prev_sync_history_id, sync_history_id, payload) VALUES (102, '2025-05-26 08:27:35.257981+00', 'test-sample-instance', 'hr_test', 'DONE', 102, 102, '{"type": "BASELINE", "gitCommit": "unknown"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, instance, db_name, status, prev_sync_history_id, sync_history_id, payload) VALUES (103, '2025-06-09 02:21:04.276704+00', 'prod-sample-instance', 'hr_prod', 'DONE', 103, 103, '{"type": "BASELINE", "gitCommit": "unknown"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, instance, db_name, status, prev_sync_history_id, sync_history_id, payload) VALUES (104, '2025-06-09 02:21:07.596692+00', 'test-sample-instance', 'hr_test', 'DONE', 104, 104, '{"type": "BASELINE", "gitCommit": "unknown"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, instance, db_name, status, prev_sync_history_id, sync_history_id, payload) VALUES (105, '2025-09-09 03:27:04.925923+00', 'prod-sample-instance', 'hr_prod', 'DONE', 105, 105, '{"type": "BASELINE", "gitCommit": "unknown"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, instance, db_name, status, prev_sync_history_id, sync_history_id, payload) VALUES (106, '2025-09-09 03:27:11.848726+00', 'test-sample-instance', 'hr_test', 'DONE', 106, 106, '{"type": "BASELINE", "gitCommit": "unknown"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, instance, db_name, status, prev_sync_history_id, sync_history_id, payload) VALUES (107, '2025-10-10 07:15:40.835471+00', 'prod-sample-instance', 'hr_prod', 'DONE', 107, 107, '{"type": "BASELINE", "gitCommit": "unknown"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, created_at, instance, db_name, status, prev_sync_history_id, sync_history_id, payload) VALUES (108, '2025-10-10 07:15:45.142133+00', 'test-sample-instance', 'hr_test', 'DONE', 108, 108, '{"type": "BASELINE", "gitCommit": "unknown"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: db; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.db (id, deleted, project, instance, name, environment, metadata) VALUES (105, false, 'default', 'prod-sample-instance', 'postgres', NULL, '{"lastSyncTime": "2025-06-05T07:08:02.311361Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, deleted, project, instance, name, environment, metadata) VALUES (102, false, 'metadb', 'bytebase-meta', 'bb', NULL, '{"lastSyncTime": "2025-06-05T07:08:02.312662Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, deleted, project, instance, name, environment, metadata) VALUES (103, false, 'default', 'test-sample-instance', 'postgres', NULL, '{"lastSyncTime": "2025-06-05T07:08:02.313191Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, deleted, project, instance, name, environment, metadata) VALUES (101, false, 'default', 'bytebase-meta', 'postgres', NULL, '{"lastSyncTime": "2025-06-05T07:08:02.316233Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, deleted, project, instance, name, environment, metadata) VALUES (106, false, 'hr', 'prod-sample-instance', 'hr_prod', NULL, '{"lastSyncTime": "2025-10-10T07:15:40.832105Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, deleted, project, instance, name, environment, metadata) VALUES (104, false, 'hr', 'test-sample-instance', 'hr_test', NULL, '{"lastSyncTime": "2025-10-10T07:15:45.139052Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;


--
-- Data for Name: db_group; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: db_schema; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.db_schema (id, instance, db_name, metadata, raw_dump, config) VALUES (102, 'bytebase-meta', 'bb', '{"name":"bb","schemas":[{"name":"public","tables":[{"name":"audit_log","columns":[{"name":"id","position":1,"default":"nextval(''public.audit_log_id_seq''::regclass)","type":"bigint"},{"name":"created_at","position":2,"default":"now()","type":"timestamp with time zone"},{"name":"payload","position":3,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"audit_log_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_log_pkey ON public.audit_log USING btree (id);","isConstraint":true},{"name":"idx_audit_log_created_at","expressions":["created_at"],"type":"btree","definition":"CREATE INDEX idx_audit_log_created_at ON public.audit_log USING btree (created_at);"},{"name":"idx_audit_log_payload_method","expressions":["payload ->> ''method''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_method ON public.audit_log USING btree (((payload ->> ''method''::text)));"},{"name":"idx_audit_log_payload_parent","expressions":["payload ->> ''parent''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_parent ON public.audit_log USING btree (((payload ->> ''parent''::text)));"},{"name":"idx_audit_log_payload_resource","expressions":["payload ->> ''resource''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_resource ON public.audit_log USING btree (((payload ->> ''resource''::text)));"},{"name":"idx_audit_log_payload_user","expressions":["payload ->> ''user''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_user ON public.audit_log USING btree (((payload ->> ''user''::text)));"}],"rowCount":"81","dataSize":"139264","indexSize":"98304","owner":"bb"},{"name":"changelist","columns":[{"name":"id","position":1,"default":"nextval(''public.changelist_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"updated_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"project","position":4,"type":"text"},{"name":"name","position":5,"type":"text"},{"name":"payload","position":6,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"changelist_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX changelist_pkey ON public.changelist USING btree (id);","isConstraint":true},{"name":"idx_changelist_project_name","expressions":["project","name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_changelist_project_name ON public.changelist USING btree (project, name);"}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"changelist_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelist_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"changelog","columns":[{"name":"id","position":1,"default":"nextval(''public.changelog_id_seq''::regclass)","type":"bigint"},{"name":"created_at","position":2,"default":"now()","type":"timestamp with time zone"},{"name":"instance","position":3,"type":"text"},{"name":"db_name","position":4,"type":"text"},{"name":"status","position":5,"type":"text"},{"name":"prev_sync_history_id","position":6,"nullable":true,"type":"bigint"},{"name":"sync_history_id","position":7,"nullable":true,"type":"bigint"},{"name":"payload","position":8,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"changelog_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX changelog_pkey ON public.changelog USING btree (id);","isConstraint":true},{"name":"idx_changelog_instance_db_name","expressions":["instance","db_name"],"type":"btree","definition":"CREATE INDEX idx_changelog_instance_db_name ON public.changelog USING btree (instance, db_name);"}],"rowCount":"2","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"changelog_instance_db_name_fkey","columns":["instance","db_name"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["instance","name"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelog_prev_sync_history_id_fkey","columns":["prev_sync_history_id"],"referencedSchema":"public","referencedTable":"sync_history","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelog_sync_history_id_fkey","columns":["sync_history_id"],"referencedSchema":"public","referencedTable":"sync_history","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"checkConstraints":[{"name":"changelog_status_check","expression":"(status = ANY (ARRAY[''PENDING''::text, ''DONE''::text, ''FAILED''::text]))"}],"owner":"bb"},{"name":"data_source","columns":[{"name":"id","position":1,"default":"nextval(''public.data_source_id_seq''::regclass)","type":"integer"},{"name":"instance","position":2,"type":"text"},{"name":"options","position":3,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"data_source_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX data_source_pkey ON public.data_source USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"8192","foreignKeys":[{"name":"data_source_instance_fkey","columns":["instance"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"db","columns":[{"name":"id","position":1,"default":"nextval(''public.db_id_seq''::regclass)","type":"integer"},{"name":"deleted","position":2,"default":"false","type":"boolean"},{"name":"project","position":3,"type":"text"},{"name":"instance","position":4,"type":"text"},{"name":"name","position":5,"type":"text"},{"name":"environment","position":6,"nullable":true,"type":"text"},{"name":"metadata","position":7,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"db_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX db_pkey ON public.db USING btree (id);","isConstraint":true},{"name":"idx_db_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_db_project ON public.db USING btree (project);"},{"name":"idx_db_unique_instance_name","expressions":["instance","name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_unique_instance_name ON public.db USING btree (instance, name);"}],"rowCount":"6","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"db_instance_fkey","columns":["instance"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"db_group","columns":[{"name":"id","position":1,"default":"nextval(''public.db_group_id_seq''::regclass)","type":"bigint"},{"name":"project","position":2,"type":"text"},{"name":"resource_id","position":3,"type":"text"},{"name":"placeholder","position":4,"default":"''''::text","type":"text"},{"name":"expression","position":5,"default":"''{}''::jsonb","type":"jsonb"},{"name":"payload","position":6,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"db_group_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX db_group_pkey ON public.db_group USING btree (id);","isConstraint":true},{"name":"idx_db_group_unique_project_placeholder","expressions":["project","placeholder"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_group_unique_project_placeholder ON public.db_group USING btree (project, placeholder);"},{"name":"idx_db_group_unique_project_resource_id","expressions":["project","resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_group_unique_project_resource_id ON public.db_group USING btree (project, resource_id);"}],"dataSize":"8192","indexSize":"24576","foreignKeys":[{"name":"db_group_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"db_schema","columns":[{"name":"id","position":1,"default":"nextval(''public.db_schema_id_seq''::regclass)","type":"integer"},{"name":"instance","position":2,"type":"text"},{"name":"db_name","position":3,"type":"text"},{"name":"metadata","position":4,"default":"''{}''::json","type":"json"},{"name":"raw_dump","position":5,"default":"''''::text","type":"text"},{"name":"config","position":6,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"db_schema_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX db_schema_pkey ON public.db_schema USING btree (id);","isConstraint":true},{"name":"idx_db_schema_unique_instance_db_name","expressions":["instance","db_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_schema_unique_instance_db_name ON public.db_schema USING btree (instance, db_name);"}],"rowCount":"6","dataSize":"73728","indexSize":"32768","foreignKeys":[{"name":"db_schema_instance_db_name_fkey","columns":["instance","db_name"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["instance","name"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"export_archive","columns":[{"name":"id","position":1,"default":"nextval(''public.export_archive_id_seq''::regclass)","type":"integer"},{"name":"created_at","position":2,"default":"now()","type":"timestamp with time zone"},{"name":"bytes","position":3,"nullable":true,"type":"bytea"},{"name":"payload","position":4,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"export_archive_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX export_archive_pkey ON public.export_archive USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"8192","owner":"bb"},{"name":"idp","columns":[{"name":"id","position":1,"default":"nextval(''public.idp_id_seq''::regclass)","type":"integer"},{"name":"resource_id","position":2,"type":"text"},{"name":"name","position":3,"type":"text"},{"name":"domain","position":4,"type":"text"},{"name":"type","position":5,"type":"text"},{"name":"config","position":6,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idp_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX idp_pkey ON public.idp USING btree (id);","isConstraint":true},{"name":"idx_idp_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_idp_unique_resource_id ON public.idp USING btree (resource_id);"}],"dataSize":"8192","indexSize":"16384","checkConstraints":[{"name":"idp_type_check","expression":"(type = ANY (ARRAY[''OAUTH2''::text, ''OIDC''::text, ''LDAP''::text]))"}],"owner":"bb"},{"name":"instance","columns":[{"name":"id","position":1,"default":"nextval(''public.instance_id_seq''::regclass)","type":"integer"},{"name":"deleted","position":2,"default":"false","type":"boolean"},{"name":"environment","position":3,"nullable":true,"type":"text"},{"name":"resource_id","position":4,"type":"text"},{"name":"metadata","position":5,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_instance_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_instance_unique_resource_id ON public.instance USING btree (resource_id);"},{"name":"instance_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX instance_pkey ON public.instance USING btree (id);","isConstraint":true}],"rowCount":"3","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"instance_change_history","columns":[{"name":"id","position":1,"default":"nextval(''public.instance_change_history_id_seq''::regclass)","type":"bigint"},{"name":"version","position":2,"type":"text"}],"indexes":[{"name":"idx_instance_change_history_unique_version","expressions":["version"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_instance_change_history_unique_version ON public.instance_change_history USING btree (version);"},{"name":"instance_change_history_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX instance_change_history_pkey ON public.instance_change_history USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"issue","columns":[{"name":"id","position":1,"default":"nextval(''public.issue_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":4,"default":"now()","type":"timestamp with time zone"},{"name":"project","position":5,"type":"text"},{"name":"plan_id","position":6,"nullable":true,"type":"bigint"},{"name":"pipeline_id","position":7,"nullable":true,"type":"integer"},{"name":"name","position":8,"type":"text"},{"name":"status","position":9,"type":"text"},{"name":"type","position":10,"type":"text"},{"name":"description","position":11,"default":"''''::text","type":"text"},{"name":"payload","position":12,"default":"''{}''::jsonb","type":"jsonb"},{"name":"ts_vector","position":13,"nullable":true,"type":"tsvector"}],"indexes":[{"name":"idx_issue_creator_id","expressions":["creator_id"],"type":"btree","definition":"CREATE INDEX idx_issue_creator_id ON public.issue USING btree (creator_id);"},{"name":"idx_issue_pipeline_id","expressions":["pipeline_id"],"type":"btree","definition":"CREATE INDEX idx_issue_pipeline_id ON public.issue USING btree (pipeline_id);"},{"name":"idx_issue_plan_id","expressions":["plan_id"],"type":"btree","definition":"CREATE INDEX idx_issue_plan_id ON public.issue USING btree (plan_id);"},{"name":"idx_issue_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_issue_project ON public.issue USING btree (project);"},{"name":"idx_issue_ts_vector","expressions":["ts_vector"],"type":"gin","definition":"CREATE INDEX idx_issue_ts_vector ON public.issue USING gin (ts_vector);"},{"name":"issue_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX issue_pkey ON public.issue USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"98304","foreignKeys":[{"name":"issue_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_plan_id_fkey","columns":["plan_id"],"referencedSchema":"public","referencedTable":"plan","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"checkConstraints":[{"name":"issue_status_check","expression":"(status = ANY (ARRAY[''OPEN''::text, ''DONE''::text, ''CANCELED''::text]))"}],"owner":"bb"},{"name":"issue_comment","columns":[{"name":"id","position":1,"default":"nextval(''public.issue_comment_id_seq''::regclass)","type":"bigint"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":4,"default":"now()","type":"timestamp with time zone"},{"name":"issue_id","position":5,"type":"integer"},{"name":"payload","position":6,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_issue_comment_issue_id","expressions":["issue_id"],"type":"btree","definition":"CREATE INDEX idx_issue_comment_issue_id ON public.issue_comment USING btree (issue_id);"},{"name":"issue_comment_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX issue_comment_pkey ON public.issue_comment USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"issue_comment_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_comment_issue_id_fkey","columns":["issue_id"],"referencedSchema":"public","referencedTable":"issue","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"issue_subscriber","columns":[{"name":"issue_id","position":1,"type":"integer"},{"name":"subscriber_id","position":2,"type":"integer"}],"indexes":[{"name":"idx_issue_subscriber_subscriber_id","expressions":["subscriber_id"],"type":"btree","definition":"CREATE INDEX idx_issue_subscriber_subscriber_id ON public.issue_subscriber USING btree (subscriber_id);"},{"name":"issue_subscriber_pkey","expressions":["issue_id","subscriber_id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX issue_subscriber_pkey ON public.issue_subscriber USING btree (issue_id, subscriber_id);","isConstraint":true}],"indexSize":"16384","foreignKeys":[{"name":"issue_subscriber_issue_id_fkey","columns":["issue_id"],"referencedSchema":"public","referencedTable":"issue","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_subscriber_subscriber_id_fkey","columns":["subscriber_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"pipeline","columns":[{"name":"id","position":1,"default":"nextval(''public.pipeline_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"project","position":4,"type":"text"},{"name":"name","position":5,"type":"text"}],"indexes":[{"name":"pipeline_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX pipeline_pkey ON public.pipeline USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"pipeline_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"pipeline_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"plan","columns":[{"name":"id","position":1,"default":"nextval(''public.plan_id_seq''::regclass)","type":"bigint"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":4,"default":"now()","type":"timestamp with time zone"},{"name":"project","position":5,"type":"text"},{"name":"pipeline_id","position":6,"nullable":true,"type":"integer"},{"name":"name","position":7,"type":"text"},{"name":"description","position":8,"type":"text"},{"name":"config","position":9,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_plan_pipeline_id","expressions":["pipeline_id"],"type":"btree","definition":"CREATE INDEX idx_plan_pipeline_id ON public.plan USING btree (pipeline_id);"},{"name":"idx_plan_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_plan_project ON public.plan USING btree (project);"},{"name":"plan_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX plan_pkey ON public.plan USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"plan_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"plan_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"plan_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"plan_check_run","columns":[{"name":"id","position":1,"default":"nextval(''public.plan_check_run_id_seq''::regclass)","type":"integer"},{"name":"created_at","position":2,"default":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"plan_id","position":4,"type":"bigint"},{"name":"status","position":5,"type":"text"},{"name":"type","position":6,"type":"text"},{"name":"config","position":7,"default":"''{}''::jsonb","type":"jsonb"},{"name":"result","position":8,"default":"''{}''::jsonb","type":"jsonb"},{"name":"payload","position":9,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_plan_check_run_plan_id","expressions":["plan_id"],"type":"btree","definition":"CREATE INDEX idx_plan_check_run_plan_id ON public.plan_check_run USING btree (plan_id);"},{"name":"plan_check_run_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX plan_check_run_pkey ON public.plan_check_run USING btree (id);","isConstraint":true}],"rowCount":"6","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"plan_check_run_plan_id_fkey","columns":["plan_id"],"referencedSchema":"public","referencedTable":"plan","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"checkConstraints":[{"name":"plan_check_run_status_check","expression":"(status = ANY (ARRAY[''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text]))"},{"name":"plan_check_run_type_check","expression":"(type ~~ ''bb.plan-check.%''::text)"}],"owner":"bb"},{"name":"policy","columns":[{"name":"id","position":1,"default":"nextval(''public.policy_id_seq''::regclass)","type":"integer"},{"name":"enforce","position":2,"default":"true","type":"boolean"},{"name":"updated_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"resource_type","position":4,"type":"text"},{"name":"resource","position":5,"type":"text"},{"name":"type","position":6,"type":"text"},{"name":"payload","position":7,"default":"''{}''::jsonb","type":"jsonb"},{"name":"inherit_from_parent","position":8,"default":"true","type":"boolean"}],"indexes":[{"name":"idx_policy_unique_resource_type_resource_type","expressions":["resource_type","resource","type"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_type ON public.policy USING btree (resource_type, resource, type);"},{"name":"policy_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX policy_pkey ON public.policy USING btree (id);","isConstraint":true}],"rowCount":"8","dataSize":"16384","indexSize":"32768","checkConstraints":[{"name":"policy_resource_type_check","expression":"(resource_type = ANY (ARRAY[''WORKSPACE''::text, ''ENVIRONMENT''::text, ''PROJECT''::text, ''INSTANCE''::text]))"}],"owner":"bb"},{"name":"principal","columns":[{"name":"id","position":1,"default":"nextval(''public.principal_id_seq''::regclass)","type":"integer"},{"name":"deleted","position":2,"default":"false","type":"boolean"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"type","position":4,"type":"text"},{"name":"name","position":5,"type":"text"},{"name":"email","position":6,"type":"text"},{"name":"password_hash","position":7,"type":"text"},{"name":"phone","position":8,"default":"''''::text","type":"text"},{"name":"mfa_config","position":9,"default":"''{}''::jsonb","type":"jsonb"},{"name":"profile","position":10,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"principal_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX principal_pkey ON public.principal USING btree (id);","isConstraint":true}],"rowCount":"7","dataSize":"16384","indexSize":"16384","checkConstraints":[{"name":"principal_type_check","expression":"(type = ANY (ARRAY[''END_USER''::text, ''SYSTEM_BOT''::text, ''SERVICE_ACCOUNT''::text]))"}],"owner":"bb"},{"name":"project","columns":[{"name":"id","position":1,"default":"nextval(''public.project_id_seq''::regclass)","type":"integer"},{"name":"deleted","position":2,"default":"false","type":"boolean"},{"name":"name","position":3,"type":"text"},{"name":"resource_id","position":4,"type":"text"},{"name":"data_classification_config_id","position":5,"default":"''''::text","type":"text"},{"name":"setting","position":6,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_project_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_project_unique_resource_id ON public.project USING btree (resource_id);"},{"name":"project_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX project_pkey ON public.project USING btree (id);","isConstraint":true}],"rowCount":"3","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"project_webhook","columns":[{"name":"id","position":1,"default":"nextval(''public.project_webhook_id_seq''::regclass)","type":"integer"},{"name":"project","position":2,"type":"text"},{"name":"type","position":3,"type":"text"},{"name":"name","position":4,"type":"text"},{"name":"url","position":5,"type":"text"},{"name":"event_list","position":6,"type":"_text"},{"name":"payload","position":7,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_project_webhook_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_project_webhook_project ON public.project_webhook USING btree (project);"},{"name":"project_webhook_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX project_webhook_pkey ON public.project_webhook USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"project_webhook_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"checkConstraints":[{"name":"project_webhook_type_check","expression":"(type ~~ ''bb.plugin.webhook.%''::text)"}],"owner":"bb"},{"name":"query_history","columns":[{"name":"id","position":1,"default":"nextval(''public.query_history_id_seq''::regclass)","type":"bigint"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"project_id","position":4,"type":"text"},{"name":"database","position":5,"type":"text"},{"name":"statement","position":6,"type":"text"},{"name":"type","position":7,"type":"text"},{"name":"payload","position":8,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_query_history_creator_id_created_at_project_id","expressions":["creator_id","created_at","project_id"],"type":"btree","definition":"CREATE INDEX idx_query_history_creator_id_created_at_project_id ON public.query_history USING btree (creator_id, created_at, project_id DESC);"},{"name":"query_history_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX query_history_pkey ON public.query_history USING btree (id);","isConstraint":true}],"rowCount":"5","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"query_history_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"release","columns":[{"name":"id","position":1,"default":"nextval(''public.release_id_seq''::regclass)","type":"bigint"},{"name":"deleted","position":2,"default":"false","type":"boolean"},{"name":"project","position":3,"type":"text"},{"name":"creator_id","position":4,"type":"integer"},{"name":"created_at","position":5,"default":"now()","type":"timestamp with time zone"},{"name":"payload","position":6,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_release_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_release_project ON public.release USING btree (project);"},{"name":"release_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX release_pkey ON public.release USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"release_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"release_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"review_config","columns":[{"name":"id","position":1,"type":"text"},{"name":"enabled","position":2,"default":"true","type":"boolean"},{"name":"name","position":3,"type":"text"},{"name":"payload","position":4,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"review_config_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX review_config_pkey ON public.review_config USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"16384","owner":"bb"},{"name":"revision","columns":[{"name":"id","position":1,"default":"nextval(''public.revision_id_seq''::regclass)","type":"bigint"},{"name":"instance","position":2,"type":"text"},{"name":"db_name","position":3,"type":"text"},{"name":"created_at","position":4,"default":"now()","type":"timestamp with time zone"},{"name":"deleter_id","position":5,"nullable":true,"type":"integer"},{"name":"deleted_at","position":6,"nullable":true,"type":"timestamp with time zone"},{"name":"version","position":7,"type":"text"},{"name":"payload","position":8,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_revision_instance_db_name_version","expressions":["instance","db_name","version"],"type":"btree","definition":"CREATE INDEX idx_revision_instance_db_name_version ON public.revision USING btree (instance, db_name, version);"},{"name":"idx_revision_unique_instance_db_name_version_deleted_at_null","expressions":["instance","db_name","version"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_version_deleted_at_null ON public.revision USING btree (instance, db_name, version) WHERE (deleted_at IS NULL);"},{"name":"revision_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX revision_pkey ON public.revision USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"24576","foreignKeys":[{"name":"revision_deleter_id_fkey","columns":["deleter_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"revision_instance_db_name_fkey","columns":["instance","db_name"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["instance","name"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"risk","columns":[{"name":"id","position":1,"default":"nextval(''public.risk_id_seq''::regclass)","type":"bigint"},{"name":"source","position":2,"type":"text"},{"name":"level","position":3,"type":"bigint"},{"name":"name","position":4,"type":"text"},{"name":"active","position":5,"type":"boolean"},{"name":"expression","position":6,"type":"jsonb"}],"indexes":[{"name":"risk_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX risk_pkey ON public.risk USING btree (id);","isConstraint":true}],"rowCount":"3","dataSize":"16384","indexSize":"16384","checkConstraints":[{"name":"risk_source_check","expression":"(source ~~ ''bb.risk.%''::text)"}],"owner":"bb"},{"name":"role","columns":[{"name":"id","position":1,"default":"nextval(''public.role_id_seq''::regclass)","type":"bigint"},{"name":"resource_id","position":2,"type":"text"},{"name":"name","position":3,"type":"text"},{"name":"description","position":4,"type":"text"},{"name":"permissions","position":5,"default":"''{}''::jsonb","type":"jsonb"},{"name":"payload","position":6,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_role_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_role_unique_resource_id ON public.role USING btree (resource_id);"},{"name":"role_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX role_pkey ON public.role USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"32768","owner":"bb"},{"name":"setting","columns":[{"name":"id","position":1,"default":"nextval(''public.setting_id_seq''::regclass)","type":"integer"},{"name":"name","position":2,"type":"text"},{"name":"value","position":3,"type":"text"}],"indexes":[{"name":"idx_setting_unique_name","expressions":["name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_setting_unique_name ON public.setting USING btree (name);"},{"name":"setting_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX setting_pkey ON public.setting USING btree (id);","isConstraint":true}],"rowCount":"14","dataSize":"49152","indexSize":"32768","owner":"bb"},{"name":"sheet","columns":[{"name":"id","position":1,"default":"nextval(''public.sheet_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"project","position":4,"type":"text"},{"name":"name","position":5,"type":"text"},{"name":"sha256","position":6,"type":"bytea"},{"name":"payload","position":7,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_sheet_project","expressions":["project"],"type":"btree","definition":"CREATE INDEX idx_sheet_project ON public.sheet USING btree (project);"},{"name":"sheet_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sheet_pkey ON public.sheet USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"sheet_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"sheet_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"sheet_blob","columns":[{"name":"sha256","position":1,"type":"bytea"},{"name":"content","position":2,"type":"text"}],"indexes":[{"name":"sheet_blob_pkey","expressions":["sha256"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sheet_blob_pkey ON public.sheet_blob USING btree (sha256);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"16384","owner":"bb"},{"name":"sync_history","columns":[{"name":"id","position":1,"default":"nextval(''public.sync_history_id_seq''::regclass)","type":"bigint"},{"name":"created_at","position":2,"default":"now()","type":"timestamp with time zone"},{"name":"instance","position":3,"type":"text"},{"name":"db_name","position":4,"type":"text"},{"name":"metadata","position":5,"default":"''{}''::json","type":"json"},{"name":"raw_dump","position":6,"default":"''''::text","type":"text"}],"indexes":[{"name":"idx_sync_history_instance_db_name_created_at","expressions":["instance","db_name","created_at"],"type":"btree","definition":"CREATE INDEX idx_sync_history_instance_db_name_created_at ON public.sync_history USING btree (instance, db_name, created_at);"},{"name":"sync_history_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sync_history_pkey ON public.sync_history USING btree (id);","isConstraint":true}],"rowCount":"2","dataSize":"32768","indexSize":"32768","foreignKeys":[{"name":"sync_history_instance_db_name_fkey","columns":["instance","db_name"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["instance","name"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"task","columns":[{"name":"id","position":1,"default":"nextval(''public.task_id_seq''::regclass)","type":"integer"},{"name":"pipeline_id","position":2,"type":"integer"},{"name":"instance","position":4,"type":"text"},{"name":"db_name","position":5,"nullable":true,"type":"text"},{"name":"type","position":6,"type":"text"},{"name":"payload","position":7,"default":"''{}''::jsonb","type":"jsonb"},{"name":"environment","position":9,"nullable":true,"type":"text"}],"indexes":[{"name":"idx_task_pipeline_id_environment","expressions":["pipeline_id","environment"],"type":"btree","definition":"CREATE INDEX idx_task_pipeline_id_environment ON public.task USING btree (pipeline_id, environment);"},{"name":"task_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_pkey ON public.task USING btree (id);","isConstraint":true}],"rowCount":"2","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"task_instance_fkey","columns":["instance"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"task_run","columns":[{"name":"id","position":1,"default":"nextval(''public.task_run_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":4,"default":"now()","type":"timestamp with time zone"},{"name":"task_id","position":5,"type":"integer"},{"name":"sheet_id","position":6,"nullable":true,"type":"integer"},{"name":"attempt","position":7,"type":"integer"},{"name":"status","position":8,"type":"text"},{"name":"started_at","position":9,"nullable":true,"type":"timestamp with time zone"},{"name":"code","position":10,"default":"0","type":"integer"},{"name":"result","position":11,"default":"''{}''::jsonb","type":"jsonb"},{"name":"run_at","position":12,"nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"idx_task_run_task_id","expressions":["task_id"],"type":"btree","definition":"CREATE INDEX idx_task_run_task_id ON public.task_run USING btree (task_id);"},{"name":"task_run_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_run_pkey ON public.task_run USING btree (id);","isConstraint":true},{"name":"uk_task_run_task_id_attempt","expressions":["task_id","attempt"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON public.task_run USING btree (task_id, attempt);"}],"dataSize":"8192","indexSize":"24576","foreignKeys":[{"name":"task_run_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_run_sheet_id_fkey","columns":["sheet_id"],"referencedSchema":"public","referencedTable":"sheet","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_run_task_id_fkey","columns":["task_id"],"referencedSchema":"public","referencedTable":"task","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"checkConstraints":[{"name":"task_run_status_check","expression":"(status = ANY (ARRAY[''PENDING''::text, ''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text]))"}],"owner":"bb"},{"name":"task_run_log","columns":[{"name":"id","position":1,"default":"nextval(''public.task_run_log_id_seq''::regclass)","type":"bigint"},{"name":"task_run_id","position":2,"type":"integer"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"payload","position":4,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_task_run_log_task_run_id","expressions":["task_run_id"],"type":"btree","definition":"CREATE INDEX idx_task_run_log_task_run_id ON public.task_run_log USING btree (task_run_id);"},{"name":"task_run_log_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_run_log_pkey ON public.task_run_log USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"task_run_log_task_run_id_fkey","columns":["task_run_id"],"referencedSchema":"public","referencedTable":"task_run","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"user_group","columns":[{"name":"email","position":1,"type":"text"},{"name":"name","position":2,"type":"text"},{"name":"description","position":3,"default":"''''::text","type":"text"},{"name":"payload","position":4,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"user_group_pkey","expressions":["email"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX user_group_pkey ON public.user_group USING btree (email);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"16384","owner":"bb"},{"name":"worksheet","columns":[{"name":"id","position":1,"default":"nextval(''public.worksheet_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_at","position":3,"default":"now()","type":"timestamp with time zone"},{"name":"updated_at","position":4,"default":"now()","type":"timestamp with time zone"},{"name":"project","position":5,"type":"text"},{"name":"instance","position":6,"nullable":true,"type":"text"},{"name":"db_name","position":7,"nullable":true,"type":"text"},{"name":"name","position":8,"type":"text"},{"name":"statement","position":9,"type":"text"},{"name":"visibility","position":10,"type":"text"},{"name":"payload","position":11,"default":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_worksheet_creator_id_project","expressions":["creator_id","project"],"type":"btree","definition":"CREATE INDEX idx_worksheet_creator_id_project ON public.worksheet USING btree (creator_id, project);"},{"name":"worksheet_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX worksheet_pkey ON public.worksheet USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"worksheet_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"worksheet_project_fkey","columns":["project"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"worksheet_organizer","columns":[{"name":"id","position":1,"default":"nextval(''public.worksheet_organizer_id_seq''::regclass)","type":"integer"},{"name":"worksheet_id","position":2,"type":"integer"},{"name":"principal_id","position":3,"type":"integer"},{"name":"starred","position":4,"default":"false","type":"boolean"}],"indexes":[{"name":"idx_worksheet_organizer_principal_id","expressions":["principal_id"],"type":"btree","definition":"CREATE INDEX idx_worksheet_organizer_principal_id ON public.worksheet_organizer USING btree (principal_id);"},{"name":"idx_worksheet_organizer_unique_sheet_id_principal_id","expressions":["worksheet_id","principal_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal_id ON public.worksheet_organizer USING btree (worksheet_id, principal_id);"},{"name":"worksheet_organizer_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX worksheet_organizer_pkey ON public.worksheet_organizer USING btree (id);","isConstraint":true}],"indexSize":"24576","foreignKeys":[{"name":"worksheet_organizer_principal_id_fkey","columns":["principal_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"worksheet_organizer_worksheet_id_fkey","columns":["worksheet_id"],"referencedSchema":"public","referencedTable":"worksheet","referencedColumns":["id"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"}],"sequences":[{"name":"audit_log_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"181","ownerTable":"audit_log","ownerColumn":"id"},{"name":"changelist_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"changelist","ownerColumn":"id"},{"name":"changelog_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"102","ownerTable":"changelog","ownerColumn":"id"},{"name":"data_source_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"data_source","ownerColumn":"id"},{"name":"db_group_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","ownerTable":"db_group","ownerColumn":"id"},{"name":"db_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"106","ownerTable":"db","ownerColumn":"id"},{"name":"db_schema_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"108","ownerTable":"db_schema","ownerColumn":"id"},{"name":"export_archive_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"export_archive","ownerColumn":"id"},{"name":"idp_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"idp","ownerColumn":"id"},{"name":"instance_change_history_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"123","ownerTable":"instance_change_history","ownerColumn":"id"},{"name":"instance_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"103","ownerTable":"instance","ownerColumn":"id"},{"name":"issue_comment_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","ownerTable":"issue_comment","ownerColumn":"id"},{"name":"issue_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"issue","ownerColumn":"id"},{"name":"pipeline_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"pipeline","ownerColumn":"id"},{"name":"plan_check_run_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"106","ownerTable":"plan_check_run","ownerColumn":"id"},{"name":"plan_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"plan","ownerColumn":"id"},{"name":"policy_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"111","ownerTable":"policy","ownerColumn":"id"},{"name":"principal_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"106","ownerTable":"principal","ownerColumn":"id"},{"name":"project_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"102","ownerTable":"project","ownerColumn":"id"},{"name":"project_webhook_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"project_webhook","ownerColumn":"id"},{"name":"query_history_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"105","ownerTable":"query_history","ownerColumn":"id"},{"name":"release_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","ownerTable":"release","ownerColumn":"id"},{"name":"revision_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","ownerTable":"revision","ownerColumn":"id"},{"name":"risk_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"103","ownerTable":"risk","ownerColumn":"id"},{"name":"role_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"role","ownerColumn":"id"},{"name":"setting_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"139","ownerTable":"setting","ownerColumn":"id"},{"name":"sheet_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"sheet","ownerColumn":"id"},{"name":"sync_history_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"102","ownerTable":"sync_history","ownerColumn":"id"},{"name":"task_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"102","ownerTable":"task","ownerColumn":"id"},{"name":"task_run_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"task_run","ownerColumn":"id"},{"name":"task_run_log_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","ownerTable":"task_run_log","ownerColumn":"id"},{"name":"worksheet_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"worksheet","ownerColumn":"id"},{"name":"worksheet_organizer_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"worksheet_organizer","ownerColumn":"id"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bb","searchPath":"\"$user\", public"}', '
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

CREATE SEQUENCE "public"."audit_log_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."audit_log" (
    "id" bigint DEFAULT nextval(''public.audit_log_id_seq''::regclass) NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL,
    "name" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "instance" text NOT NULL,
    "db_name" text NOT NULL,
    "status" text NOT NULL,
    "prev_sync_history_id" bigint,
    "sync_history_id" bigint,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT "changelog_status_check" CHECK (status = ANY (ARRAY[''PENDING''::text, ''DONE''::text, ''FAILED''::text]))
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
    "instance" text NOT NULL,
    "options" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "deleted" boolean DEFAULT false NOT NULL,
    "project" text NOT NULL,
    "instance" text NOT NULL,
    "name" text NOT NULL,
    "environment" text,
    "metadata" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "project" text NOT NULL,
    "resource_id" text NOT NULL,
    "placeholder" text DEFAULT ''''::text NOT NULL,
    "expression" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "instance" text NOT NULL,
    "db_name" text NOT NULL,
    "metadata" json DEFAULT ''{}''::json NOT NULL,
    "raw_dump" text DEFAULT ''''::text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."db_schema_id_seq" OWNED BY "public"."db_schema"."id";

ALTER TABLE ONLY "public"."db_schema" ADD CONSTRAINT "db_schema_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_db_schema_unique_instance_db_name" ON ONLY "public"."db_schema" ("instance", "db_name");

CREATE SEQUENCE "public"."export_archive_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."export_archive" (
    "id" integer DEFAULT nextval(''public.export_archive_id_seq''::regclass) NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "bytes" bytea,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    CONSTRAINT "idp_type_check" CHECK (type = ANY (ARRAY[''OAUTH2''::text, ''OIDC''::text, ''LDAP''::text]))
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
    "deleted" boolean DEFAULT false NOT NULL,
    "environment" text,
    "resource_id" text NOT NULL,
    "metadata" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL,
    "plan_id" bigint,
    "pipeline_id" integer,
    "name" text NOT NULL,
    "status" text NOT NULL,
    "type" text NOT NULL,
    "description" text DEFAULT ''''::text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "ts_vector" tsvector,
    CONSTRAINT "issue_status_check" CHECK (status = ANY (ARRAY[''OPEN''::text, ''DONE''::text, ''CANCELED''::text]))
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
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "issue_id" integer NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL,
    "name" text NOT NULL
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
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL,
    "pipeline_id" integer,
    "name" text NOT NULL,
    "description" text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "plan_id" bigint NOT NULL,
    "status" text NOT NULL,
    "type" text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "result" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT "plan_check_run_status_check" CHECK (status = ANY (ARRAY[''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text])),
    CONSTRAINT "plan_check_run_type_check" CHECK (type ~~ ''bb.plan-check.%''::text)
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
    "enforce" boolean DEFAULT true NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "resource_type" text NOT NULL,
    "resource" text NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "inherit_from_parent" boolean DEFAULT true NOT NULL,
    CONSTRAINT "policy_resource_type_check" CHECK (resource_type = ANY (ARRAY[''WORKSPACE''::text, ''ENVIRONMENT''::text, ''PROJECT''::text, ''INSTANCE''::text]))
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
    "deleted" boolean DEFAULT false NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "type" text NOT NULL,
    "name" text NOT NULL,
    "email" text NOT NULL,
    "password_hash" text NOT NULL,
    "phone" text DEFAULT ''''::text NOT NULL,
    "mfa_config" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "profile" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT "principal_type_check" CHECK (type = ANY (ARRAY[''END_USER''::text, ''SYSTEM_BOT''::text, ''SERVICE_ACCOUNT''::text]))
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
    "deleted" boolean DEFAULT false NOT NULL,
    "name" text NOT NULL,
    "resource_id" text NOT NULL,
    "data_classification_config_id" text DEFAULT ''''::text NOT NULL,
    "setting" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "project" text NOT NULL,
    "type" text NOT NULL,
    "name" text NOT NULL,
    "url" text NOT NULL,
    "event_list" _text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    CONSTRAINT "project_webhook_type_check" CHECK (type ~~ ''bb.plugin.webhook.%''::text)
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
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project_id" text NOT NULL,
    "database" text NOT NULL,
    "statement" text NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "deleted" boolean DEFAULT false NOT NULL,
    "project" text NOT NULL,
    "creator_id" integer NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."release_id_seq" OWNED BY "public"."release"."id";

ALTER TABLE ONLY "public"."release" ADD CONSTRAINT "release_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_release_project" ON ONLY "public"."release" ("project");

CREATE TABLE "public"."review_config" (
    "id" text NOT NULL,
    "enabled" boolean DEFAULT true NOT NULL,
    "name" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "instance" text NOT NULL,
    "db_name" text NOT NULL,
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "deleter_id" integer,
    "deleted_at" timestamp with time zone,
    "version" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "expression" jsonb NOT NULL,
    CONSTRAINT "risk_source_check" CHECK (source ~~ ''bb.risk.%''::text)
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
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL,
    "name" text NOT NULL,
    "sha256" bytea NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."sheet_id_seq" OWNED BY "public"."sheet"."id";

ALTER TABLE ONLY "public"."sheet" ADD CONSTRAINT "sheet_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_sheet_project" ON ONLY "public"."sheet" ("project");

CREATE TABLE "public"."sheet_blob" (
    "sha256" bytea NOT NULL,
    "content" text NOT NULL
);

ALTER TABLE ONLY "public"."sheet_blob" ADD CONSTRAINT "sheet_blob_pkey" PRIMARY KEY ("sha256");

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
    "instance" text NOT NULL,
    "db_name" text NOT NULL,
    "metadata" json DEFAULT ''{}''::json NOT NULL,
    "raw_dump" text DEFAULT ''''::text NOT NULL
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
    "instance" text NOT NULL,
    "db_name" text,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "environment" text
);

ALTER SEQUENCE "public"."task_id_seq" OWNED BY "public"."task"."id";

ALTER TABLE ONLY "public"."task" ADD CONSTRAINT "task_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_task_pipeline_id_environment" ON ONLY "public"."task" ("pipeline_id", "environment");

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
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "task_id" integer NOT NULL,
    "sheet_id" integer,
    "attempt" integer NOT NULL,
    "status" text NOT NULL,
    "started_at" timestamp with time zone,
    "code" integer DEFAULT 0 NOT NULL,
    "result" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "run_at" timestamp with time zone,
    CONSTRAINT "task_run_status_check" CHECK (status = ANY (ARRAY[''PENDING''::text, ''RUNNING''::text, ''DONE''::text, ''FAILED''::text, ''CANCELED''::text]))
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
    "created_at" timestamp with time zone DEFAULT now() NOT NULL,
    "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
    "project" text NOT NULL,
    "instance" text,
    "db_name" text,
    "name" text NOT NULL,
    "statement" text NOT NULL,
    "visibility" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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

ALTER TABLE "public"."sync_history"
    ADD CONSTRAINT "sync_history_instance_db_name_fkey" FOREIGN KEY ("instance", "db_name")
    REFERENCES "public"."db" ("instance", "name");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_instance_fkey" FOREIGN KEY ("instance")
    REFERENCES "public"."instance" ("resource_id");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_pipeline_id_fkey" FOREIGN KEY ("pipeline_id")
    REFERENCES "public"."pipeline" ("id");

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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, instance, db_name, metadata, raw_dump, config) VALUES (101, 'bytebase-meta', 'postgres', '{"name":"postgres", "schemas":[{"name":"public", "owner":"pg_database_owner"}], "characterSet":"UTF8", "collation":"en_US.UTF-8", "owner":"bb", "searchPath":"\"$user\", public"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, instance, db_name, metadata, raw_dump, config) VALUES (105, 'prod-sample-instance', 'postgres', '{"name":"postgres", "schemas":[{"name":"public", "owner":"pg_database_owner"}], "characterSet":"UTF8", "collation":"en_US.UTF-8", "owner":"bbsample", "searchPath":"\"$user\", public"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, instance, db_name, metadata, raw_dump, config) VALUES (103, 'test-sample-instance', 'postgres', '{"name":"postgres", "schemas":[{"name":"public", "owner":"pg_database_owner"}], "characterSet":"UTF8", "collation":"en_US.UTF-8", "owner":"bbsample", "searchPath":"\"$user\", public"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, instance, db_name, metadata, raw_dump, config) VALUES (106, 'prod-sample-instance', 'hr_prod', '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"default":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"default":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp(6) with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_audit_changed_at","expressions":["changed_at"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);","opclassNames":["timestamptz_ops"],"opclassDefaults":[true]},{"name":"idx_audit_operation","expressions":["operation"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);","opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"idx_audit_username","expressions":["user_name"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);","opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"descending":[false],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"department_pkey","expressions":["dept_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"default":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_employee_hire_date","expressions":["hire_date"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);","opclassNames":["date_ops"],"opclassDefaults":[true]}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","checkConstraints":[{"name":"employee_gender_check","expression":"(gender = ANY (ARRAY[''M''::text, ''F''::text]))"}],"owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);","opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"salary_pkey","expressions":["emp_no","from_date"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true,"opclassNames":["int4_ops","date_ops"],"opclassDefaults":[true,true]}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"descending":[false,false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true,"opclassNames":["int4_ops","text_ops","date_ops"],"opclassDefaults":[true,true,true]}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner","comment":"standard public schema"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample","searchPath":"\"$user\", public"}', '
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

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

SET default_tablespace = '''';

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY (id);

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" (changed_at);

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" (operation);

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" (user_name);

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY (dept_no);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE (dept_name);

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL,
    CONSTRAINT "employee_gender_check" CHECK (gender = ANY (ARRAY[''M''::text, ''F''::text]))
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY (emp_no);

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" (hire_date);

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

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY (emp_no, from_date);

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" (amount);

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY (emp_no, title, from_date);

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
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

', '{"schemas": [{"name": "public", "tables": [{"name": "salary", "columns": [{"name": "to_date"}, {"name": "amount", "semanticType": "bb.default"}, {"name": "emp_no"}, {"name": "from_date"}]}, {"name": "employee", "columns": [{"name": "last_name", "classification": "1-2"}, {"name": "emp_no"}, {"name": "birth_date"}, {"name": "gender"}, {"name": "hire_date"}, {"name": "first_name", "classification": "1-2"}]}]}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, instance, db_name, metadata, raw_dump, config) VALUES (104, 'test-sample-instance', 'hr_test', '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"default":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"default":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp(6) with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_audit_changed_at","expressions":["changed_at"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);","opclassNames":["timestamptz_ops"],"opclassDefaults":[true]},{"name":"idx_audit_operation","expressions":["operation"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);","opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"idx_audit_username","expressions":["user_name"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);","opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"descending":[false],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"department_pkey","expressions":["dept_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"default":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_employee_hire_date","expressions":["hire_date"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);","opclassNames":["date_ops"],"opclassDefaults":[true]}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","checkConstraints":[{"name":"employee_gender_check","expression":"(gender = ANY (ARRAY[''M''::text, ''F''::text]))"}],"owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);","opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"salary_pkey","expressions":["emp_no","from_date"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true,"opclassNames":["int4_ops","date_ops"],"opclassDefaults":[true,true]}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"descending":[false,false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true,"opclassNames":["int4_ops","text_ops","date_ops"],"opclassDefaults":[true,true,true]}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner","comment":"standard public schema"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample","searchPath":"\"$user\", public"}', '
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

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

SET default_tablespace = '''';

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY (id);

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" (changed_at);

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" (operation);

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" (user_name);

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY (dept_no);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE (dept_name);

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL,
    CONSTRAINT "employee_gender_check" CHECK (gender = ANY (ARRAY[''M''::text, ''F''::text]))
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY (emp_no);

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" (hire_date);

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

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY (emp_no, from_date);

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" (amount);

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY (emp_no, title, from_date);

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
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

', '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: export_archive; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: idp; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: instance; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.instance (id, deleted, environment, resource_id, metadata) VALUES (101, false, 'prod', 'bytebase-meta', '{"roles": [{"name": "bb", "attribute": "Superuser Create role Create DB Replication Bypass RLS+"}], "title": "bytebase-meta", "engine": "POSTGRES", "version": "16.0.0", "activation": true, "dataSources": [{"id": "35a64b4a-543f-4eac-ad32-b191a958c66d", "host": "/tmp", "port": "8082", "type": "ADMIN", "username": "bb", "authenticationType": "PASSWORD"}], "lastSyncTime": "2025-06-05T07:07:57.638515Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, deleted, environment, resource_id, metadata) VALUES (103, false, 'prod', 'prod-sample-instance', '{"roles": [{"name": "bbsample", "attribute": "Superuser Create role Create DB Replication Bypass RLS+"}], "title": "Prod Sample Instance", "engine": "POSTGRES", "version": "16.0.0", "activation": true, "dataSources": [{"id": "9af4f227-a55e-4e82-b7f5-c7193b5f405c", "host": "/tmp", "port": "8084", "type": "ADMIN", "username": "bbsample", "authenticationType": "PASSWORD"}, {"id": "e700ae12-173e-4f0d-8590-0414cf6a9405", "host": "/tmp", "port": "8084", "type": "READ_ONLY", "username": "bbsample", "authenticationType": "PASSWORD"}], "lastSyncTime": "2025-06-05T07:07:57.649471Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, deleted, environment, resource_id, metadata) VALUES (102, false, 'test', 'test-sample-instance', '{"roles": [{"name": "bbsample", "attribute": "Superuser Create role Create DB Replication Bypass RLS+"}], "title": "Test Sample Instance", "engine": "POSTGRES", "version": "16.0.0", "activation": true, "dataSources": [{"id": "a7f206f9-37c4-41ca-8b59-fcf4c6148105", "host": "/tmp", "port": "8083", "type": "ADMIN", "username": "bbsample", "authenticationType": "PASSWORD"}], "lastSyncTime": "2025-06-05T07:07:57.659121Z"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: instance_change_history; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.instance_change_history (id, version) VALUES (101, '3.6.5') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (102, '3.7.0') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (103, '3.7.1') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (104, '3.7.2') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (105, '3.7.3') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (106, '3.7.4') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (107, '3.7.5') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (108, '3.7.6') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (109, '3.7.7') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (110, '3.7.8') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (111, '3.7.9') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (112, '3.7.10') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (113, '3.7.11') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (114, '3.7.12') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (115, '3.7.13') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (116, '3.7.14') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (117, '3.7.15') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (118, '3.7.16') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (119, '3.7.17') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (120, '3.7.18') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (121, '3.7.19') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (122, '3.7.20') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (123, '3.7.21') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (124, '3.7.22') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (125, '3.7.23') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (126, '3.7.24') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (127, '3.7.25') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (128, '3.7.26') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (129, '3.7.27') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (130, '3.8.0') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (131, '3.8.1') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (132, '3.8.2') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (133, '3.8.3') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (134, '3.8.4') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (135, '3.9.0') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (136, '3.9.1') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (137, '3.9.2') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (138, '3.9.3') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (139, '3.9.4') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (140, '3.9.5') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (141, '3.9.6') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (142, '3.9.7') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (143, '3.9.8') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (144, '3.9.9') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (145, '3.9.10') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (146, '3.10.0') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (147, '3.10.1') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (148, '3.10.2') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (149, '3.10.3') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (150, '3.10.4') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (151, '3.10.5') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (152, '3.10.6') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (153, '3.10.7') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (154, '3.10.8') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (155, '3.11.0') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (156, '3.11.1') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (157, '3.11.2') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (158, '3.11.3') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (159, '3.11.4') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (160, '3.11.5') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (161, '3.11.6') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (162, '3.11.7') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (163, '3.11.8') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (164, '3.11.9') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (165, '3.11.10') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (166, '3.11.11') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (167, '3.11.12') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (168, '3.11.13') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (169, '3.11.14') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (170, '3.11.15') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (171, '3.11.16') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (172, '3.11.17') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (173, '3.11.18') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (174, '3.11.19') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (175, '3.11.20') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (176, '3.11.21') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (177, '3.12.0') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (178, '3.12.1') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (179, '3.12.2') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, version) VALUES (180, '3.12.3') ON CONFLICT DO NOTHING;


--
-- Data for Name: issue; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.issue (id, creator_id, created_at, updated_at, project, plan_id, name, status, type, description, payload, ts_vector) VALUES (101, 101, '2025-05-26 08:26:15.26631+00', '2025-05-26 08:26:16.121796+00', 'hr', 101, '', 'OPEN', 'DATABASE_CHANGE', '', '{"labels": ["3.6.2", "feature"], "approval": {"riskLevel": "HIGH", "approvalTemplate": {"id": "bb.project-owner-workspace-dba", "flow": {"roles": ["roles/projectOwner", "roles/workspaceDBA"]}, "title": "Project Owner -> Workspace DBA", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "approvalFindingDone": true}}', '''add'':3 ''column'':5 ''email'':4 ''employee'':7 ''here'':2 ''start'':1 ''table'':8 ''to'':6') ON CONFLICT DO NOTHING;


--
-- Data for Name: issue_comment; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: pipeline; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.pipeline (id, creator_id, created_at, project) VALUES (101, 101, '2025-05-26 08:26:15.272294+00', 'hr') ON CONFLICT DO NOTHING;


--
-- Data for Name: plan; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.plan (id, creator_id, created_at, updated_at, project, pipeline_id, name, description, config, deleted) VALUES (101, 101, '2025-05-26 08:26:15.250094+00', '2025-05-26 08:26:15.250094+00', 'hr', 101, '👉👉👉 [START HERE] Add email column to Employee table', '', '{"specs": [{"id": "036e8f34-05e6-4916-ba58-45c6a452fc60", "changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/hr/sheets/101", "targets": ["instances/prod-sample-instance/databases/hr_prod", "instances/test-sample-instance/databases/hr_test"], "migrateType": "DDL"}}], "deployment": {"environments": ["test", "prod"]}}', false) ON CONFLICT DO NOTHING;


--
-- Data for Name: plan_check_run; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.plan_check_run (id, created_at, updated_at, plan_id, status, type, config, result, payload) VALUES (103, '2025-05-26 08:26:15.253027+00', '2025-05-26 08:26:15.255968+00', 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 101, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_test", "schemas": [{"name": "public", "tables": [{"name": "employee", "ranges": [{"end": 68}], "tableRows": "1000"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, created_at, updated_at, plan_id, status, type, config, result, payload) VALUES (106, '2025-05-26 08:26:15.253027+00', '2025-05-26 08:26:15.255984+00', 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 101, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee", "ranges": [{"end": 68}], "tableRows": "1000"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, created_at, updated_at, plan_id, status, type, config, result, payload) VALUES (101, '2025-05-26 08:26:15.253027+00', '2025-05-26 08:26:15.25678+00', 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 101, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, created_at, updated_at, plan_id, status, type, config, result, payload) VALUES (102, '2025-05-26 08:26:15.253027+00', '2025-05-26 08:26:15.258632+00', 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 101, "instanceId": "test-sample-instance", "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"code": 402, "title": "column.no-null", "status": "WARNING", "content": "Column \"email\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"startPosition": {}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, created_at, updated_at, plan_id, status, type, config, result, payload) VALUES (104, '2025-05-26 08:26:15.253027+00', '2025-05-26 08:26:15.256299+00', 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 101, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, created_at, updated_at, plan_id, status, type, config, result, payload) VALUES (105, '2025-05-26 08:26:15.253027+00', '2025-05-26 08:26:15.260276+00', 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 101, "instanceId": "prod-sample-instance", "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"code": 402, "title": "column.no-null", "status": "WARNING", "content": "Column \"email\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"startPosition": {}}}]}', '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: policy; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (107, true, '2025-05-26 07:54:33.728664+00', 'PROJECT', 'projects/metadb', 'IAM', '{"bindings": [{"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/sqlEditorUser", "members": ["users/102", "users/103"], "condition": {"title": "SQL Editor User All databases"}}]}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (103, true, '2025-05-26 08:07:29.784956+00', 'PROJECT', 'projects/hr', 'IAM', '{"bindings": [{"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/projectDeveloper", "members": ["users/102"], "condition": {"title": "Project Developer All databases"}}]}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (101, true, '2025-05-26 08:10:00.958154+00', 'WORKSPACE', '', 'IAM', '{"bindings": [{"role": "roles/workspaceMember", "members": ["allUsers", "users/102", "users/105", "users/106", "groups/qa-group@example.com"], "condition": {}}, {"role": "roles/workspaceAdmin", "members": ["users/101", "users/104"], "condition": {}}, {"role": "roles/workspaceDBA", "members": ["users/103", "users/101"], "condition": {}}, {"role": "roles/qa-custom-role", "members": ["groups/qa-group@example.com"], "condition": {}}]}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (110, true, '2025-05-26 08:23:40.12567+00', 'ENVIRONMENT', 'environments/prod', 'DATA_SOURCE_QUERY', '{"disallowDdl": true, "disallowDml": true}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (104, true, '2025-05-26 08:47:09.079713+00', 'ENVIRONMENT', 'environments/test', 'TAG', '{"tags": {"bb.tag.review_config": "reviewConfigs/sql-review-sample-policy"}}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (105, true, '2025-05-26 08:47:09.079724+00', 'ENVIRONMENT', 'environments/prod', 'TAG', '{"tags": {"bb.tag.review_config": "reviewConfigs/sql-review-sample-policy"}}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (112, true, '2025-09-09 03:16:27.935445+00', 'WORKSPACE', '', 'QUERY_DATA', '{"disableExport": false, "maximumResultRows": -1, "maximumResultSize": 104857600}', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (111, true, '2025-10-10 07:01:01.383813+00', 'ENVIRONMENT', 'environments/prod', 'QUERY_DATA', '{"disableCopyData": true}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (106, true, '2025-05-26 07:48:23.461198+00', 'WORKSPACE', '', 'MASKING_RULE', '{"rules": [{"id": "1f3018bb-f4ea-4daa-b4b3-6e32bce0d22e", "condition": {"expression": "resource.classification_level in [\"2\", \"3\"]"}, "semanticType": "bb.default-partial"}, {"id": "e0743172-c9c7-43f8-9923-9a3c06012cee", "condition": {"expression": "resource.classification_level in [\"4\"]"}, "semanticType": "bb.default"}]}', false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (113, true, '2025-10-13 22:54:21.894178+00', 'ENVIRONMENT', 'environments/test', 'ROLLOUT', '{"checkers": {"requiredStatusChecks": {"planCheckEnforcement": "ERROR_ONLY"}, "requiredIssueApproval": true}}', true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, enforce, updated_at, resource_type, resource, type, payload, inherit_from_parent) VALUES (114, true, '2025-10-13 22:54:21.894178+00', 'ENVIRONMENT', 'environments/prod', 'ROLLOUT', '{"checkers": {"requiredStatusChecks": {"planCheckEnforcement": "ERROR_ONLY"}, "requiredIssueApproval": true}}', true) ON CONFLICT DO NOTHING;


--
-- Data for Name: principal; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.principal (id, deleted, created_at, type, name, email, password_hash, phone, mfa_config, profile) VALUES (1, false, '2025-05-26 06:23:21.385682+00', 'SYSTEM_BOT', 'Bytebase', 'support@bytebase.com', '', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, deleted, created_at, type, name, email, password_hash, phone, mfa_config, profile) VALUES (102, false, '2025-05-26 07:15:33.831094+00', 'END_USER', 'Dev1', 'dev1@example.com', '$2a$10$KPcwy0hDaEWKqNBvDhr1eORTibMOiMlkVIk5NAuvkxkqx.HVdESsO', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, deleted, created_at, type, name, email, password_hash, phone, mfa_config, profile) VALUES (103, false, '2025-05-26 07:16:05.084865+00', 'END_USER', 'dba1', 'dba1@example.com', '$2a$10$6N6Cf2mFthj.GHvIHYwySei5xdNdnJDgsvt.5ez7TRsiFrtQXVM82', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, deleted, created_at, type, name, email, password_hash, phone, mfa_config, profile) VALUES (104, false, '2025-05-26 07:16:33.693169+00', 'SERVICE_ACCOUNT', 'API user', 'api@service.bytebase.com', '$2a$10$dm2.6B6YYbSDoKRDAmph2O4amsa4RDSiHjWpO2JfosO8ceP5vErj2', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, deleted, created_at, type, name, email, password_hash, phone, mfa_config, profile) VALUES (105, false, '2025-05-26 08:08:54.153568+00', 'END_USER', 'QA1', 'qa1@example.com', '$2a$10$ItgVGF7yA68QAlDPPykM.eDTXVYTVXdpqilNcVNGutI0XUExq2nZG', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, deleted, created_at, type, name, email, password_hash, phone, mfa_config, profile) VALUES (106, false, '2025-05-26 08:09:14.423135+00', 'END_USER', 'QA2', 'qa2@example.com', '$2a$10$rUSSIn7pKKuRfBjrz1xdJud3kIY79zymiIuu4k8ufza7EO5Phqi76', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, deleted, created_at, type, name, email, password_hash, phone, mfa_config, profile) VALUES (101, false, '2025-05-26 06:48:06.299115+00', 'END_USER', 'Demo', 'demo@example.com', '$2a$10$rounVehKcCdUp3ykPl.K6.ebxXWOLxtcFEmDBHNQFcHK/pGTDUREy', '', '{}', '{"lastLoginTime": "2025-10-13T22:55:22.866265Z"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: project; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.project (id, deleted, name, resource_id, data_classification_config_id, setting) VALUES (102, false, 'Bytebase MetaDB', 'metadb', 'e5680e79-e84b-486e-8cb2-76c984c3fac9', '{"autoResolveIssue": true, "allowModifyStatement": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, deleted, name, resource_id, data_classification_config_id, setting) VALUES (1, false, 'Default', 'default', 'e5680e79-e84b-486e-8cb2-76c984c3fac9', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, deleted, name, resource_id, data_classification_config_id, setting) VALUES (101, false, 'HR Project', 'hr', 'e5680e79-e84b-486e-8cb2-76c984c3fac9', '{"issueLabels": [{"color": "#4f46e5", "value": "3.6.2"}, {"color": "#E54646", "value": "bug"}, {"color": "#63E546", "value": "feature"}], "autoResolveIssue": true, "allowSelfApproval": true, "allowModifyStatement": true}') ON CONFLICT DO NOTHING;


--
-- Data for Name: project_webhook; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: query_history; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.query_history (id, creator_id, created_at, project_id, database, statement, type, payload) VALUES (101, 101, '2025-05-26 07:57:47.34796+00', 'metadb', 'instances/bytebase-meta/databases/bb', '-- Fully completed issues by project
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
  project.resource_id;', 'QUERY', '{"duration": "0.000827625s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, created_at, project_id, database, statement, type, payload) VALUES (102, 101, '2025-05-26 07:58:13.178283+00', 'metadb', 'instances/bytebase-meta/databases/bb', '-- Issues created by user
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
  COUNT(issue.creator_id) DESC;', 'QUERY', '{"duration": "0.001645875s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, created_at, project_id, database, statement, type, payload) VALUES (103, 101, '2025-05-26 08:15:44.969221+00', 'hr', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM salary;', 'QUERY', '{"duration": "0.003704s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, created_at, project_id, database, statement, type, payload) VALUES (104, 101, '2025-05-26 08:15:49.542826+00', 'hr', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee', 'QUERY', '{"duration": "0.005847792s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, creator_id, created_at, project_id, database, statement, type, payload) VALUES (105, 101, '2025-05-26 08:31:19.334518+00', 'metadb', 'instances/bytebase-meta/databases/bb', '-- Issues created by user
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
  COUNT(issue.creator_id) DESC;', 'QUERY', '{"duration": "0.001888209s"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: release; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: review_config; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.review_config (id, enabled, name, payload) VALUES ('sql-review-sample-policy', true, 'SQL Review Sample Policy', '{"sqlReviewRules": [{"type": "database.drop-empty-database", "level": "ERROR", "engine": "MYSQL", "payload": "{}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MYSQL", "payload": "{\"format\":\"_del$\"}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "MYSQL", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "TIDB", "payload": "{}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "TIDB", "payload": "{\"format\":\"_del$\"}"}, {"type": "column.no-null", "level": "WARNING", "engine": "TIDB", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "OCEANBASE", "payload": "{}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "OCEANBASE", "payload": "{\"format\":\"_del$\"}"}, {"type": "column.no-null", "level": "WARNING", "engine": "OCEANBASE", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "MARIADB", "payload": "{}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MARIADB", "payload": "{\"format\":\"_del$\"}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MARIADB", "payload": "{}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "POSTGRES", "payload": "{\"format\":\"_del$\"}"}, {"type": "column.no-null", "level": "WARNING", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.maximum-limit-value", "level": "WARNING", "engine": "POSTGRES", "comment": "Limiting the number of rows through LIMIT ensures the database processes manageable chunks, improving query execution speed.  A capped LIMIT value prevents excessive memory usage, safeguarding overall system stability and preventing performance degradation.", "payload": "{\"number\":100}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "POSTGRES", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "SNOWFLAKE", "payload": "{\"format\":\"_del$\"}"}, {"type": "column.no-null", "level": "WARNING", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MSSQL", "payload": "{\"format\":\"_del$\"}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MSSQL", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "ORACLE", "payload": "{}"}]}') ON CONFLICT DO NOTHING;


--
-- Data for Name: revision; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: risk; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.risk (id, source, name, active, expression, level) VALUES (101, 'bb.risk.database.schema.update', ' ALTER column in production environment is high risk', true, '{"expression": "resource.environment_id == \"prod\" && statement.sql_type == \"ALTER_TABLE\""}', 'HIGH') ON CONFLICT DO NOTHING;
INSERT INTO public.risk (id, source, name, active, expression, level) VALUES (103, 'bb.risk.database.data.update', 'Updated or deleted rows exceeds 100 in prod is high risk', true, '{"expression": "resource.environment_id == \"prod\" && statement.affected_rows > 100 && statement.sql_type in [\"UPDATE\", \"DELETE\"]"}', 'HIGH') ON CONFLICT DO NOTHING;
INSERT INTO public.risk (id, source, name, active, expression, level) VALUES (102, 'bb.risk.database.schema.update', 'CREATE TABLE in production environment is moderate risk', true, '{"expression": "resource.environment_id == \"prod\" && statement.sql_type == \"CREATE_TABLE\""}', 'MODERATE') ON CONFLICT DO NOTHING;


--
-- Data for Name: role; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.role (id, resource_id, name, description, permissions, payload) VALUES (101, 'qa-custom-role', 'QA', 'Custom-defined QA role', '{"permissions": ["bb.rollouts.get", "bb.taskRuns.list", "bb.databases.get", "bb.databases.getSchema", "bb.databases.list", "bb.plans.get", "bb.plans.list", "bb.projects.getIamPolicy", "bb.issueComments.create", "bb.issues.get", "bb.issues.list", "bb.planCheckRuns.list", "bb.planCheckRuns.run", "bb.projects.get"]}', '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: setting; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.setting (id, name, value) VALUES (101, 'BRANDING_LOGO', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (102, 'AUTH_SECRET', 'l71IPJkuT7aTj7McDY3MSJ9BVqBAt2NQ') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (103, 'WORKSPACE_ID', 'a6b014b9-d0d4-4974-9be6-53ec61ea5f48') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (104, 'SCIM', '{"token":"3nnNo4tmEFH9FFyTACCfzGhyZUb4QsUC"}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (105, 'PASSWORD_RESTRICTION', '{"minLength":8}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (109, 'SCHEMA_TEMPLATE', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (106, 'ENTERPRISE_LICENSE', 'eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (113, 'ENVIRONMENT', '{"environments":[{"id":"test", "title":"Test"}, {"id":"prod", "title":"Prod", "tags":{"protected":"protected"}}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (128, 'SEMANTIC_TYPES', '{"types":[{"id":"bb.default", "title":"Default", "description":"Default type with full masking"}, {"id":"bb.default-partial", "title":"Default Partial", "description":"Default partial type with partial masking"}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (108, 'WATERMARK', '1') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (110, 'DATA_CLASSIFICATION', '{"configs":[{"id":"e5680e79-e84b-486e-8cb2-76c984c3fac9", "title":"Classification Example", "levels":[{"id":"1", "title":"Level 1"}, {"id":"2", "title":"Level 2"}, {"id":"3", "title":"Level 3"}, {"id":"4", "title":"Level 4"}], "classification":{"1":{"id":"1", "title":"Basic"}, "1-1":{"id":"1-1", "title":"Basic", "levelId":"1"}, "1-2":{"id":"1-2", "title":"Contact", "levelId":"2"}, "1-3":{"id":"1-3", "title":"Health", "levelId":"4"}, "2":{"id":"2", "title":"Relationship"}, "2-1":{"id":"2-1", "title":"Social", "levelId":"1"}, "2-2":{"id":"2-2", "title":"Business", "levelId":"3"}}, "classificationFromConfig":true}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (111, 'WORKSPACE_APPROVAL', '{"rules": [{"template": {"id": "bb.project-owner", "flow": {"roles": ["roles/projectOwner"]}, "title": "Project Owner", "description": "The system defines the approval process and only needs the project Owner to approve it."}, "condition": {"expression": "source == \"DML\" && level == \"RISK_LEVEL_UNSPECIFIED\" || source == \"DDL\" && level == \"MODERATE\" || source == \"DDL\" && level == \"RISK_LEVEL_UNSPECIFIED\" || source == \"DML\" && level == \"MODERATE\" || source == \"DATA_EXPORT\" && level == \"RISK_LEVEL_UNSPECIFIED\" || source == \"REQUEST_ROLE\" && level == \"RISK_LEVEL_UNSPECIFIED\" || source == \"CREATE_DATABASE\" && level == \"RISK_LEVEL_UNSPECIFIED\""}}, {"template": {"id": "bb.project-owner-workspace-dba", "flow": {"roles": ["roles/projectOwner", "roles/workspaceDBA"]}, "title": "Project Owner -> Workspace DBA", "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}, "condition": {"expression": "source == \"DML\" && level == \"HIGH\" || source == \"DDL\" && level == \"HIGH\""}}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (107, 'APP_IM', '{"settings": []}') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, name, value) VALUES (112, 'WORKSPACE_PROFILE', '{"externalUrl":"https://demo.bytebase.com", "domains":["example.com"], "databaseChangeMode":"PIPELINE", "enableMetricCollection":true}') ON CONFLICT DO NOTHING;


--
-- Data for Name: sheet; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.sheet (id, creator_id, created_at, project, name, sha256, payload) VALUES (101, 101, '2025-05-26 08:26:15.238969+00', 'hr', '👉👉👉 [START HERE] Add email column to Employee table', '\xdc3cbdad177e12396a1be4e31c959f7b0fdf03193c21bcfa113da7fa23109222', '{"engine": "POSTGRES", "commands": [{"end": 68}]}') ON CONFLICT DO NOTHING;


--
-- Data for Name: sheet_blob; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.sheet_blob (sha256, content) VALUES ('\xdc3cbdad177e12396a1be4e31c959f7b0fdf03193c21bcfa113da7fa23109222', 'ALTER TABLE employee ADD COLUMN IF NOT EXISTS email TEXT DEFAULT '''';') ON CONFLICT DO NOTHING;


--
-- Data for Name: sync_history; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.sync_history (id, created_at, instance, db_name, metadata, raw_dump) VALUES (101, '2025-05-26 08:27:15.374695+00', 'prod-sample-instance', 'hr_prod', '{"name":"hr_prod", "schemas":[{"name":"bbdataarchive", "owner":"bbsample"}, {"name":"public", "tables":[{"name":"audit", "columns":[{"name":"id", "position":1, "defaultExpression":"nextval(''public.audit_id_seq''::regclass)", "type":"integer"}, {"name":"operation", "position":2, "type":"text"}, {"name":"query", "position":3, "nullable":true, "type":"text"}, {"name":"user_name", "position":4, "type":"text"}, {"name":"changed_at", "position":5, "defaultExpression":"CURRENT_TIMESTAMP", "nullable":true, "type":"timestamp with time zone"}], "indexes":[{"name":"audit_pkey", "expressions":["id"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);", "isConstraint":true}, {"name":"idx_audit_changed_at", "expressions":["changed_at"], "type":"btree", "definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"}, {"name":"idx_audit_operation", "expressions":["operation"], "type":"btree", "definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"}, {"name":"idx_audit_username", "expressions":["user_name"], "type":"btree", "definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}], "dataSize":"8192", "indexSize":"32768", "owner":"bbsample"}, {"name":"department", "columns":[{"name":"dept_no", "position":1, "type":"text"}, {"name":"dept_name", "position":2, "type":"text"}], "indexes":[{"name":"department_dept_name_key", "expressions":["dept_name"], "type":"btree", "unique":true, "definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);", "isConstraint":true}, {"name":"department_pkey", "expressions":["dept_no"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);", "isConstraint":true}], "dataSize":"16384", "indexSize":"32768", "owner":"bbsample"}, {"name":"dept_emp", "columns":[{"name":"emp_no", "position":1, "type":"integer"}, {"name":"dept_no", "position":2, "type":"text"}, {"name":"from_date", "position":3, "type":"date"}, {"name":"to_date", "position":4, "type":"date"}], "indexes":[{"name":"dept_emp_pkey", "expressions":["emp_no", "dept_no"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);", "isConstraint":true}], "rowCount":"1103", "dataSize":"106496", "indexSize":"57344", "foreignKeys":[{"name":"dept_emp_dept_no_fkey", "columns":["dept_no"], "referencedSchema":"public", "referencedTable":"department", "referencedColumns":["dept_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}, {"name":"dept_emp_emp_no_fkey", "columns":["emp_no"], "referencedSchema":"public", "referencedTable":"employee", "referencedColumns":["emp_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}], "owner":"bbsample"}, {"name":"dept_manager", "columns":[{"name":"emp_no", "position":1, "type":"integer"}, {"name":"dept_no", "position":2, "type":"text"}, {"name":"from_date", "position":3, "type":"date"}, {"name":"to_date", "position":4, "type":"date"}], "indexes":[{"name":"dept_manager_pkey", "expressions":["emp_no", "dept_no"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);", "isConstraint":true}], "dataSize":"16384", "indexSize":"16384", "foreignKeys":[{"name":"dept_manager_dept_no_fkey", "columns":["dept_no"], "referencedSchema":"public", "referencedTable":"department", "referencedColumns":["dept_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}, {"name":"dept_manager_emp_no_fkey", "columns":["emp_no"], "referencedSchema":"public", "referencedTable":"employee", "referencedColumns":["emp_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}], "owner":"bbsample"}, {"name":"employee", "columns":[{"name":"emp_no", "position":1, "defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)", "type":"integer"}, {"name":"birth_date", "position":2, "type":"date"}, {"name":"first_name", "position":3, "type":"text"}, {"name":"last_name", "position":4, "type":"text"}, {"name":"gender", "position":5, "type":"text"}, {"name":"hire_date", "position":6, "type":"date"}], "indexes":[{"name":"employee_pkey", "expressions":["emp_no"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);", "isConstraint":true}, {"name":"idx_employee_hire_date", "expressions":["hire_date"], "type":"btree", "definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}], "rowCount":"1000", "dataSize":"98304", "indexSize":"98304", "owner":"bbsample"}, {"name":"salary", "columns":[{"name":"emp_no", "position":1, "type":"integer"}, {"name":"amount", "position":2, "type":"integer"}, {"name":"from_date", "position":3, "type":"date"}, {"name":"to_date", "position":4, "type":"date"}], "indexes":[{"name":"idx_salary_amount", "expressions":["amount"], "type":"btree", "definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"}, {"name":"salary_pkey", "expressions":["emp_no", "from_date"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);", "isConstraint":true}], "rowCount":"9488", "dataSize":"458752", "indexSize":"548864", "foreignKeys":[{"name":"salary_emp_no_fkey", "columns":["emp_no"], "referencedSchema":"public", "referencedTable":"employee", "referencedColumns":["emp_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}], "owner":"bbsample", "triggers":[{"name":"salary_log_trigger", "body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]}, {"name":"title", "columns":[{"name":"emp_no", "position":1, "type":"integer"}, {"name":"title", "position":2, "type":"text"}, {"name":"from_date", "position":3, "type":"date"}, {"name":"to_date", "position":4, "nullable":true, "type":"date"}], "indexes":[{"name":"title_pkey", "expressions":["emp_no", "title", "from_date"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);", "isConstraint":true}], "rowCount":"1470", "dataSize":"131072", "indexSize":"73728", "foreignKeys":[{"name":"title_emp_no_fkey", "columns":["emp_no"], "referencedSchema":"public", "referencedTable":"employee", "referencedColumns":["emp_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}], "owner":"bbsample"}], "views":[{"name":"current_dept_emp", "definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));", "dependencyColumns":[{"schema":"public", "table":"dept_emp", "column":"dept_no"}, {"schema":"public", "table":"dept_emp", "column":"emp_no"}, {"schema":"public", "table":"dept_emp", "column":"from_date"}, {"schema":"public", "table":"dept_emp", "column":"to_date"}, {"schema":"public", "table":"dept_emp_latest_date", "column":"emp_no"}, {"schema":"public", "table":"dept_emp_latest_date", "column":"from_date"}, {"schema":"public", "table":"dept_emp_latest_date", "column":"to_date"}], "columns":[{"name":"emp_no", "position":1, "nullable":true, "type":"integer"}, {"name":"dept_no", "position":2, "nullable":true, "type":"text"}, {"name":"from_date", "position":3, "nullable":true, "type":"date"}, {"name":"to_date", "position":4, "nullable":true, "type":"date"}]}, {"name":"dept_emp_latest_date", "definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;", "dependencyColumns":[{"schema":"public", "table":"dept_emp", "column":"emp_no"}, {"schema":"public", "table":"dept_emp", "column":"from_date"}, {"schema":"public", "table":"dept_emp", "column":"to_date"}], "columns":[{"name":"emp_no", "position":1, "nullable":true, "type":"integer"}, {"name":"from_date", "position":2, "nullable":true, "type":"date"}, {"name":"to_date", "position":3, "nullable":true, "type":"date"}]}], "functions":[{"name":"log_dml_operations", "definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n", "signature":"log_dml_operations()"}], "sequences":[{"name":"audit_id_seq", "dataType":"integer", "start":"1", "minValue":"1", "maxValue":"2147483647", "increment":"1", "cacheSize":"1", "ownerTable":"audit", "ownerColumn":"id"}, {"name":"employee_emp_no_seq", "dataType":"integer", "start":"1", "minValue":"1", "maxValue":"2147483647", "increment":"1", "cacheSize":"1", "ownerTable":"employee", "ownerColumn":"emp_no"}], "owner":"pg_database_owner"}], "characterSet":"UTF8", "collation":"en_US.UTF-8", "owner":"bbsample"}', '
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, instance, db_name, metadata, raw_dump) VALUES (102, '2025-05-26 08:27:35.257294+00', 'test-sample-instance', 'hr_test', '{"name":"hr_test", "schemas":[{"name":"bbdataarchive", "owner":"bbsample"}, {"name":"public", "tables":[{"name":"audit", "columns":[{"name":"id", "position":1, "defaultExpression":"nextval(''public.audit_id_seq''::regclass)", "type":"integer"}, {"name":"operation", "position":2, "type":"text"}, {"name":"query", "position":3, "nullable":true, "type":"text"}, {"name":"user_name", "position":4, "type":"text"}, {"name":"changed_at", "position":5, "defaultExpression":"CURRENT_TIMESTAMP", "nullable":true, "type":"timestamp with time zone"}], "indexes":[{"name":"audit_pkey", "expressions":["id"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);", "isConstraint":true}, {"name":"idx_audit_changed_at", "expressions":["changed_at"], "type":"btree", "definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"}, {"name":"idx_audit_operation", "expressions":["operation"], "type":"btree", "definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"}, {"name":"idx_audit_username", "expressions":["user_name"], "type":"btree", "definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}], "dataSize":"8192", "indexSize":"32768", "owner":"bbsample"}, {"name":"department", "columns":[{"name":"dept_no", "position":1, "type":"text"}, {"name":"dept_name", "position":2, "type":"text"}], "indexes":[{"name":"department_dept_name_key", "expressions":["dept_name"], "type":"btree", "unique":true, "definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);", "isConstraint":true}, {"name":"department_pkey", "expressions":["dept_no"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);", "isConstraint":true}], "dataSize":"16384", "indexSize":"32768", "owner":"bbsample"}, {"name":"dept_emp", "columns":[{"name":"emp_no", "position":1, "type":"integer"}, {"name":"dept_no", "position":2, "type":"text"}, {"name":"from_date", "position":3, "type":"date"}, {"name":"to_date", "position":4, "type":"date"}], "indexes":[{"name":"dept_emp_pkey", "expressions":["emp_no", "dept_no"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);", "isConstraint":true}], "rowCount":"1103", "dataSize":"106496", "indexSize":"57344", "foreignKeys":[{"name":"dept_emp_dept_no_fkey", "columns":["dept_no"], "referencedSchema":"public", "referencedTable":"department", "referencedColumns":["dept_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}, {"name":"dept_emp_emp_no_fkey", "columns":["emp_no"], "referencedSchema":"public", "referencedTable":"employee", "referencedColumns":["emp_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}], "owner":"bbsample"}, {"name":"dept_manager", "columns":[{"name":"emp_no", "position":1, "type":"integer"}, {"name":"dept_no", "position":2, "type":"text"}, {"name":"from_date", "position":3, "type":"date"}, {"name":"to_date", "position":4, "type":"date"}], "indexes":[{"name":"dept_manager_pkey", "expressions":["emp_no", "dept_no"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);", "isConstraint":true}], "dataSize":"16384", "indexSize":"16384", "foreignKeys":[{"name":"dept_manager_dept_no_fkey", "columns":["dept_no"], "referencedSchema":"public", "referencedTable":"department", "referencedColumns":["dept_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}, {"name":"dept_manager_emp_no_fkey", "columns":["emp_no"], "referencedSchema":"public", "referencedTable":"employee", "referencedColumns":["emp_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}], "owner":"bbsample"}, {"name":"employee", "columns":[{"name":"emp_no", "position":1, "defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)", "type":"integer"}, {"name":"birth_date", "position":2, "type":"date"}, {"name":"first_name", "position":3, "type":"text"}, {"name":"last_name", "position":4, "type":"text"}, {"name":"gender", "position":5, "type":"text"}, {"name":"hire_date", "position":6, "type":"date"}], "indexes":[{"name":"employee_pkey", "expressions":["emp_no"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);", "isConstraint":true}, {"name":"idx_employee_hire_date", "expressions":["hire_date"], "type":"btree", "definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}], "rowCount":"1000", "dataSize":"98304", "indexSize":"98304", "owner":"bbsample"}, {"name":"salary", "columns":[{"name":"emp_no", "position":1, "type":"integer"}, {"name":"amount", "position":2, "type":"integer"}, {"name":"from_date", "position":3, "type":"date"}, {"name":"to_date", "position":4, "type":"date"}], "indexes":[{"name":"idx_salary_amount", "expressions":["amount"], "type":"btree", "definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"}, {"name":"salary_pkey", "expressions":["emp_no", "from_date"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);", "isConstraint":true}], "rowCount":"9488", "dataSize":"458752", "indexSize":"548864", "foreignKeys":[{"name":"salary_emp_no_fkey", "columns":["emp_no"], "referencedSchema":"public", "referencedTable":"employee", "referencedColumns":["emp_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}], "owner":"bbsample", "triggers":[{"name":"salary_log_trigger", "body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]}, {"name":"title", "columns":[{"name":"emp_no", "position":1, "type":"integer"}, {"name":"title", "position":2, "type":"text"}, {"name":"from_date", "position":3, "type":"date"}, {"name":"to_date", "position":4, "nullable":true, "type":"date"}], "indexes":[{"name":"title_pkey", "expressions":["emp_no", "title", "from_date"], "type":"btree", "unique":true, "primary":true, "definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);", "isConstraint":true}], "rowCount":"1470", "dataSize":"131072", "indexSize":"73728", "foreignKeys":[{"name":"title_emp_no_fkey", "columns":["emp_no"], "referencedSchema":"public", "referencedTable":"employee", "referencedColumns":["emp_no"], "onDelete":"CASCADE", "onUpdate":"NO ACTION", "matchType":"SIMPLE"}], "owner":"bbsample"}], "views":[{"name":"current_dept_emp", "definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));", "dependencyColumns":[{"schema":"public", "table":"dept_emp", "column":"dept_no"}, {"schema":"public", "table":"dept_emp", "column":"emp_no"}, {"schema":"public", "table":"dept_emp", "column":"from_date"}, {"schema":"public", "table":"dept_emp", "column":"to_date"}, {"schema":"public", "table":"dept_emp_latest_date", "column":"emp_no"}, {"schema":"public", "table":"dept_emp_latest_date", "column":"from_date"}, {"schema":"public", "table":"dept_emp_latest_date", "column":"to_date"}], "columns":[{"name":"emp_no", "position":1, "nullable":true, "type":"integer"}, {"name":"dept_no", "position":2, "nullable":true, "type":"text"}, {"name":"from_date", "position":3, "nullable":true, "type":"date"}, {"name":"to_date", "position":4, "nullable":true, "type":"date"}]}, {"name":"dept_emp_latest_date", "definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;", "dependencyColumns":[{"schema":"public", "table":"dept_emp", "column":"emp_no"}, {"schema":"public", "table":"dept_emp", "column":"from_date"}, {"schema":"public", "table":"dept_emp", "column":"to_date"}], "columns":[{"name":"emp_no", "position":1, "nullable":true, "type":"integer"}, {"name":"from_date", "position":2, "nullable":true, "type":"date"}, {"name":"to_date", "position":3, "nullable":true, "type":"date"}]}], "functions":[{"name":"log_dml_operations", "definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n", "signature":"log_dml_operations()"}], "sequences":[{"name":"audit_id_seq", "dataType":"integer", "start":"1", "minValue":"1", "maxValue":"2147483647", "increment":"1", "cacheSize":"1", "ownerTable":"audit", "ownerColumn":"id"}, {"name":"employee_emp_no_seq", "dataType":"integer", "start":"1", "minValue":"1", "maxValue":"2147483647", "increment":"1", "cacheSize":"1", "ownerTable":"employee", "ownerColumn":"emp_no"}], "owner":"pg_database_owner"}], "characterSet":"UTF8", "collation":"en_US.UTF-8", "owner":"bbsample"}', '
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, instance, db_name, metadata, raw_dump) VALUES (103, '2025-06-09 02:21:04.275364+00', 'prod-sample-instance', 'hr_prod', '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","checkConstraints":[{"name":"employee_gender_check","expression":"(gender = ANY (ARRAY[''M''::text, ''F''::text]))"}],"owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample","searchPath":"\"$user\", public"}', '
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
    "hire_date" date NOT NULL,
    CONSTRAINT "employee_gender_check" CHECK (gender = ANY (ARRAY[''M''::text, ''F''::text]))
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, instance, db_name, metadata, raw_dump) VALUES (104, '2025-06-09 02:21:07.595494+00', 'test-sample-instance', 'hr_test', '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","checkConstraints":[{"name":"employee_gender_check","expression":"(gender = ANY (ARRAY[''M''::text, ''F''::text]))"}],"owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample","searchPath":"\"$user\", public"}', '
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
    "hire_date" date NOT NULL,
    CONSTRAINT "employee_gender_check" CHECK (gender = ANY (ARRAY[''M''::text, ''F''::text]))
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, instance, db_name, metadata, raw_dump) VALUES (105, '2025-09-09 03:27:04.924+00', 'prod-sample-instance', 'hr_prod', '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"default":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"default":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp(6) with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_audit_changed_at","expressions":["changed_at"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);","opclassNames":["timestamptz_ops"],"opclassDefaults":[true]},{"name":"idx_audit_operation","expressions":["operation"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);","opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"idx_audit_username","expressions":["user_name"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);","opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"descending":[false],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"department_pkey","expressions":["dept_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"default":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_employee_hire_date","expressions":["hire_date"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);","opclassNames":["date_ops"],"opclassDefaults":[true]}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","checkConstraints":[{"name":"employee_gender_check","expression":"(gender = ANY (ARRAY[''M''::text, ''F''::text]))"}],"owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);","opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"salary_pkey","expressions":["emp_no","from_date"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true,"opclassNames":["int4_ops","date_ops"],"opclassDefaults":[true,true]}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"descending":[false,false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true,"opclassNames":["int4_ops","text_ops","date_ops"],"opclassDefaults":[true,true,true]}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner","comment":"standard public schema"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample","searchPath":"\"$user\", public"}', '
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
    "changed_at" timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" (changed_at);

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" (operation);

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" (user_name);

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
    CONSTRAINT "employee_gender_check" CHECK (gender = ANY (ARRAY[''M''::text, ''F''::text]))
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" (hire_date);

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

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" (amount);

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
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, instance, db_name, metadata, raw_dump) VALUES (106, '2025-09-09 03:27:11.847779+00', 'test-sample-instance', 'hr_test', '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"default":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"default":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp(6) with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_audit_changed_at","expressions":["changed_at"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);","opclassNames":["timestamptz_ops"],"opclassDefaults":[true]},{"name":"idx_audit_operation","expressions":["operation"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);","opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"idx_audit_username","expressions":["user_name"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);","opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"descending":[false],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"department_pkey","expressions":["dept_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"default":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_employee_hire_date","expressions":["hire_date"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);","opclassNames":["date_ops"],"opclassDefaults":[true]}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","checkConstraints":[{"name":"employee_gender_check","expression":"(gender = ANY (ARRAY[''M''::text, ''F''::text]))"}],"owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);","opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"salary_pkey","expressions":["emp_no","from_date"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true,"opclassNames":["int4_ops","date_ops"],"opclassDefaults":[true,true]}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"descending":[false,false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true,"opclassNames":["int4_ops","text_ops","date_ops"],"opclassDefaults":[true,true,true]}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner","comment":"standard public schema"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample","searchPath":"\"$user\", public"}', '
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
    "changed_at" timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" (changed_at);

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" (operation);

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" (user_name);

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
    CONSTRAINT "employee_gender_check" CHECK (gender = ANY (ARRAY[''M''::text, ''F''::text]))
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY ("emp_no");

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" (hire_date);

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

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" (amount);

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
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, instance, db_name, metadata, raw_dump) VALUES (107, '2025-10-10 07:15:40.833999+00', 'prod-sample-instance', 'hr_prod', '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"default":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"default":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp(6) with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_audit_changed_at","expressions":["changed_at"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);","opclassNames":["timestamptz_ops"],"opclassDefaults":[true]},{"name":"idx_audit_operation","expressions":["operation"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);","opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"idx_audit_username","expressions":["user_name"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);","opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"descending":[false],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"department_pkey","expressions":["dept_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"default":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_employee_hire_date","expressions":["hire_date"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);","opclassNames":["date_ops"],"opclassDefaults":[true]}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","checkConstraints":[{"name":"employee_gender_check","expression":"(gender = ANY (ARRAY[''M''::text, ''F''::text]))"}],"owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);","opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"salary_pkey","expressions":["emp_no","from_date"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true,"opclassNames":["int4_ops","date_ops"],"opclassDefaults":[true,true]}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"descending":[false,false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true,"opclassNames":["int4_ops","text_ops","date_ops"],"opclassDefaults":[true,true,true]}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner","comment":"standard public schema"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample","searchPath":"\"$user\", public"}', '
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

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

SET default_tablespace = '''';

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY (id);

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" (changed_at);

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" (operation);

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" (user_name);

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY (dept_no);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE (dept_name);

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL,
    CONSTRAINT "employee_gender_check" CHECK (gender = ANY (ARRAY[''M''::text, ''F''::text]))
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY (emp_no);

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" (hire_date);

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

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY (emp_no, from_date);

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" (amount);

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY (emp_no, title, from_date);

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
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, created_at, instance, db_name, metadata, raw_dump) VALUES (108, '2025-10-10 07:15:45.141539+00', 'test-sample-instance', 'hr_test', '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"default":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"default":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp(6) with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_audit_changed_at","expressions":["changed_at"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);","opclassNames":["timestamptz_ops"],"opclassDefaults":[true]},{"name":"idx_audit_operation","expressions":["operation"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);","opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"idx_audit_username","expressions":["user_name"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);","opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"descending":[false],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]},{"name":"department_pkey","expressions":["dept_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true,"opclassNames":["text_ops"],"opclassDefaults":[true]}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true,"opclassNames":["int4_ops","text_ops"],"opclassDefaults":[true,true]}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"default":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"descending":[false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true,"opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"idx_employee_hire_date","expressions":["hire_date"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);","opclassNames":["date_ops"],"opclassDefaults":[true]}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","checkConstraints":[{"name":"employee_gender_check","expression":"(gender = ANY (ARRAY[''M''::text, ''F''::text]))"}],"owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"descending":[false],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);","opclassNames":["int4_ops"],"opclassDefaults":[true]},{"name":"salary_pkey","expressions":["emp_no","from_date"],"descending":[false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true,"opclassNames":["int4_ops","date_ops"],"opclassDefaults":[true,true]}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"descending":[false,false,false],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true,"opclassNames":["int4_ops","text_ops","date_ops"],"opclassDefaults":[true,true,true]}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner","comment":"standard public schema"}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bbsample","searchPath":"\"$user\", public"}', '
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

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

SET default_tablespace = '''';

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval(''public.audit_id_seq''::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY (id);

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" (changed_at);

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" (operation);

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" (user_name);

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY (dept_no);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE (dept_name);

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval(''public.employee_emp_no_seq''::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL,
    CONSTRAINT "employee_gender_check" CHECK (gender = ANY (ARRAY[''M''::text, ''F''::text]))
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY (emp_no);

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" (hire_date);

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

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY (emp_no, from_date);

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" (amount);

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY (emp_no, title, from_date);

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
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

') ON CONFLICT DO NOTHING;


--
-- Data for Name: task; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.task (id, pipeline_id, instance, db_name, type, payload, environment) VALUES (101, 101, 'test-sample-instance', 'hr_test', 'DATABASE_MIGRATE', '{"specId": "036e8f34-05e6-4916-ba58-45c6a452fc60", "sheetId": 101, "migrateType": "DDL", "taskReleaseSource": {}}', 'test') ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, pipeline_id, instance, db_name, type, payload, environment) VALUES (102, 101, 'prod-sample-instance', 'hr_prod', 'DATABASE_MIGRATE', '{"specId": "ea634b72-91e0-48b5-9ab6-ecc8d9355ff8", "sheetId": 101, "migrateType": "DDL", "taskReleaseSource": {}}', 'prod') ON CONFLICT DO NOTHING;


--
-- Data for Name: task_run; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: task_run_log; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: user_group; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.user_group (email, name, description, payload) VALUES ('qa-group@example.com', 'QA Group', '', '{"members": [{"role": "OWNER", "member": "users/105"}, {"role": "MEMBER", "member": "users/106"}]}') ON CONFLICT DO NOTHING;


--
-- Data for Name: worksheet; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.worksheet (id, creator_id, created_at, updated_at, project, instance, db_name, name, statement, visibility, payload) VALUES (101, 101, '2025-05-26 07:59:09.119115+00', '2025-05-26 07:59:12.540192+00', 'metadb', 'bytebase-meta', 'bb', 'Issues created by user', '-- Issues created by user
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
  COUNT(issue.creator_id) DESC;', 'PROJECT_READ', '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: worksheet_organizer; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Name: audit_log_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.audit_log_id_seq', 192, true);


--
-- Name: changelog_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.changelog_id_seq', 108, true);


--
-- Name: db_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.db_group_id_seq', 101, false);


--
-- Name: db_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.db_id_seq', 106, true);


--
-- Name: db_schema_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.db_schema_id_seq', 122, true);


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

SELECT pg_catalog.setval('public.instance_change_history_id_seq', 180, true);


--
-- Name: instance_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.instance_id_seq', 103, true);


--
-- Name: issue_comment_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.issue_comment_id_seq', 101, false);


--
-- Name: issue_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.issue_id_seq', 101, true);


--
-- Name: pipeline_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.pipeline_id_seq', 101, true);


--
-- Name: plan_check_run_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.plan_check_run_id_seq', 106, true);


--
-- Name: plan_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.plan_id_seq', 101, true);


--
-- Name: policy_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.policy_id_seq', 114, true);


--
-- Name: principal_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.principal_id_seq', 106, true);


--
-- Name: project_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.project_id_seq', 102, true);


--
-- Name: project_webhook_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.project_webhook_id_seq', 101, false);


--
-- Name: query_history_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.query_history_id_seq', 105, true);


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

SELECT pg_catalog.setval('public.risk_id_seq', 103, true);


--
-- Name: role_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.role_id_seq', 101, true);


--
-- Name: setting_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.setting_id_seq', 145, true);


--
-- Name: sheet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.sheet_id_seq', 101, true);


--
-- Name: sync_history_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.sync_history_id_seq', 108, true);


--
-- Name: task_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.task_id_seq', 102, true);


--
-- Name: task_run_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.task_run_id_seq', 101, false);


--
-- Name: task_run_log_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.task_run_log_id_seq', 101, false);


--
-- Name: worksheet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.worksheet_id_seq', 101, true);


--
-- Name: worksheet_organizer_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.worksheet_organizer_id_seq', 1, false);


--
-- Name: audit_log audit_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);


--
-- Name: changelog changelog_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_pkey PRIMARY KEY (id);


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
-- Name: idx_changelog_instance_db_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_changelog_instance_db_name ON public.changelog USING btree (instance, db_name);


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
-- Name: idx_issue_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_project ON public.issue USING btree (project);


--
-- Name: idx_issue_ts_vector; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_ts_vector ON public.issue USING gin (ts_vector);


--
-- Name: idx_issue_unique_plan_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_issue_unique_plan_id ON public.issue USING btree (plan_id);


--
-- Name: idx_plan_check_run_active_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_check_run_active_status ON public.plan_check_run USING btree (status, id) WHERE (status = 'RUNNING'::text);


--
-- Name: idx_plan_check_run_plan_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_check_run_plan_id ON public.plan_check_run USING btree (plan_id);


--
-- Name: idx_plan_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_project ON public.plan USING btree (project);


--
-- Name: idx_plan_unique_pipeline_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_plan_unique_pipeline_id ON public.plan USING btree (pipeline_id);


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
-- Name: idx_release_unique_project_digest; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_release_unique_project_digest ON public.release USING btree (project, digest) WHERE (digest <> ''::text);


--
-- Name: idx_revision_instance_db_name_type_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_revision_instance_db_name_type_version ON public.revision USING btree (instance, db_name, ((payload ->> 'type'::text)), version);


--
-- Name: idx_revision_unique_instance_db_name_type_version_deleted_at_nu; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_type_version_deleted_at_nu ON public.revision USING btree (instance, db_name, ((payload ->> 'type'::text)), version) WHERE (deleted_at IS NULL);


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
-- Name: idx_sync_history_instance_db_name_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sync_history_instance_db_name_created_at ON public.sync_history USING btree (instance, db_name, created_at);


--
-- Name: idx_task_pipeline_id_environment; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_pipeline_id_environment ON public.task USING btree (pipeline_id, environment);


--
-- Name: idx_task_run_active_status_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_run_active_status_id ON public.task_run USING btree (status, id) WHERE (status = ANY (ARRAY['PENDING'::text, 'RUNNING'::text]));


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


