--
-- PostgreSQL database dump
--

-- Dumped from database version 16.2
-- Dumped by pg_dump version 16.6 (Homebrew)

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
-- Name: resource_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.resource_type AS ENUM (
    'WORKSPACE',
    'ENVIRONMENT',
    'PROJECT',
    'INSTANCE',
    'DATABASE'
);


--
-- Name: row_status; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.row_status AS ENUM (
    'NORMAL',
    'ARCHIVED'
);


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: activity; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.activity (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    resource_container text DEFAULT ''::text NOT NULL,
    container_id integer NOT NULL,
    type text NOT NULL,
    level text NOT NULL,
    comment text DEFAULT ''::text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT activity_container_id_check CHECK ((container_id > 0)),
    CONSTRAINT activity_level_check CHECK ((level = ANY (ARRAY['INFO'::text, 'WARN'::text, 'ERROR'::text]))),
    CONSTRAINT activity_type_check CHECK ((type ~~ 'bb.%'::text))
);


--
-- Name: activity_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.activity_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: activity_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.activity_id_seq OWNED BY public.activity.id;


--
-- Name: anomaly; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.anomaly (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project text NOT NULL,
    instance_id integer NOT NULL,
    database_id integer,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
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
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
-- Name: branch; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.branch (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL,
    engine text NOT NULL,
    base jsonb DEFAULT '{}'::jsonb NOT NULL,
    head jsonb DEFAULT '{}'::jsonb NOT NULL,
    base_schema text DEFAULT ''::text NOT NULL,
    head_schema text DEFAULT ''::text NOT NULL,
    reconcile_state text DEFAULT ''::text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: branch_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.branch_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: branch_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.branch_id_seq OWNED BY public.branch.id;


--
-- Name: changelist; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changelist (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
    creator_id integer NOT NULL,
    created_ts timestamp with time zone DEFAULT now() NOT NULL,
    database_id integer NOT NULL,
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
-- Name: data_source; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.data_source (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    username text NOT NULL,
    password text NOT NULL,
    ssl_key text DEFAULT ''::text NOT NULL,
    ssl_cert text DEFAULT ''::text NOT NULL,
    ssl_ca text DEFAULT ''::text NOT NULL,
    host text DEFAULT ''::text NOT NULL,
    port text DEFAULT ''::text NOT NULL,
    options jsonb DEFAULT '{}'::jsonb NOT NULL,
    database text DEFAULT ''::text NOT NULL,
    CONSTRAINT data_source_type_check CHECK ((type = ANY (ARRAY['ADMIN'::text, 'RW'::text, 'RO'::text])))
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    project_id integer NOT NULL,
    environment text,
    sync_status text NOT NULL,
    last_successful_sync_ts bigint NOT NULL,
    schema_version text NOT NULL,
    name text NOT NULL,
    secrets jsonb DEFAULT '{}'::jsonb NOT NULL,
    datashare boolean DEFAULT false NOT NULL,
    service_name text DEFAULT ''::text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT db_sync_status_check CHECK ((sync_status = ANY (ARRAY['OK'::text, 'NOT_FOUND'::text])))
);


--
-- Name: db_group; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.db_group (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT ''::text NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    database_id integer NOT NULL,
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
-- Name: deployment_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.deployment_config (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: deployment_config_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.deployment_config_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: deployment_config_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.deployment_config_id_seq OWNED BY public.deployment_config.id;


--
-- Name: environment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.environment (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    "order" integer NOT NULL,
    resource_id text NOT NULL,
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
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
-- Name: external_approval; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.external_approval (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    issue_id integer NOT NULL,
    requester_id integer NOT NULL,
    approver_id integer NOT NULL,
    type text NOT NULL,
    payload jsonb NOT NULL,
    CONSTRAINT external_approval_type_check CHECK ((type ~~ 'bb.plugin.app.%'::text))
);


--
-- Name: external_approval_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.external_approval_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: external_approval_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.external_approval_id_seq OWNED BY public.external_approval.id;


--
-- Name: idp; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.idp (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    environment text,
    name text NOT NULL,
    engine text NOT NULL,
    engine_version text DEFAULT ''::text NOT NULL,
    external_link text DEFAULT ''::text NOT NULL,
    resource_id text NOT NULL,
    activation boolean DEFAULT false NOT NULL,
    options jsonb DEFAULT '{}'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: instance_change_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.instance_change_history (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer,
    database_id integer,
    project_id integer,
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
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT instance_change_history_sequence_check CHECK ((sequence >= 0)),
    CONSTRAINT instance_change_history_source_check CHECK ((source = ANY (ARRAY['UI'::text, 'VCS'::text, 'LIBRARY'::text]))),
    CONSTRAINT instance_change_history_status_check CHECK ((status = ANY (ARRAY['PENDING'::text, 'DONE'::text, 'FAILED'::text]))),
    CONSTRAINT instance_change_history_type_check CHECK ((type = ANY (ARRAY['BASELINE'::text, 'MIGRATE'::text, 'MIGRATE_SDL'::text, 'BRANCH'::text, 'DATA'::text])))
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
-- Name: instance_user; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.instance_user (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    name text NOT NULL,
    "grant" text NOT NULL
);


--
-- Name: instance_user_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.instance_user_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: instance_user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.instance_user_id_seq OWNED BY public.instance_user.id;


--
-- Name: issue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
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
    description text DEFAULT ''::text NOT NULL,
    assignee_id integer,
    assignee_need_attention boolean DEFAULT false NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    ts_vector tsvector,
    CONSTRAINT issue_status_check CHECK ((status = ANY (ARRAY['OPEN'::text, 'DONE'::text, 'CANCELED'::text]))),
    CONSTRAINT issue_type_check CHECK ((type ~~ 'bb.issue.%'::text))
);


--
-- Name: issue_comment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_comment (
    id bigint NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    name text NOT NULL
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    pipeline_id integer,
    name text NOT NULL,
    description text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: plan_check_run; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_check_run (
    id integer NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    resource_type public.resource_type NOT NULL,
    resource_id integer NOT NULL,
    inherit_from_parent boolean DEFAULT true NOT NULL,
    CONSTRAINT policy_type_check CHECK ((type ~~ 'bb.policy.%'::text))
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    key text NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    activity_list text[] NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    project_id integer NOT NULL,
    creator_id integer NOT NULL,
    created_ts timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: revision; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.revision (
    id bigint NOT NULL,
    database_id integer NOT NULL,
    creator_id integer NOT NULL,
    created_ts timestamp with time zone DEFAULT now() NOT NULL,
    deleter_id integer,
    deleted_ts timestamp with time zone,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    value text NOT NULL,
    description text DEFAULT ''::text NOT NULL
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
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
-- Name: slow_query; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.slow_query (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    instance_id integer NOT NULL,
    database_id integer,
    log_date_ts integer NOT NULL,
    slow_query_statistics jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: slow_query_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.slow_query_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: slow_query_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.slow_query_id_seq OWNED BY public.slow_query.id;


--
-- Name: sql_lint_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sql_lint_config (
    id text NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: stage; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stage (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    pipeline_id integer NOT NULL,
    environment_id integer NOT NULL,
    deployment_id text DEFAULT ''::text NOT NULL,
    name text NOT NULL
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
    creator_id integer NOT NULL,
    created_ts timestamp with time zone DEFAULT now() NOT NULL,
    database_id integer NOT NULL,
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
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
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
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    earliest_allowed_ts bigint DEFAULT 0 NOT NULL,
    CONSTRAINT task_status_check CHECK ((status = ANY (ARRAY['PENDING'::text, 'PENDING_APPROVAL'::text, 'RUNNING'::text, 'DONE'::text, 'FAILED'::text, 'CANCELED'::text]))),
    CONSTRAINT task_type_check CHECK ((type ~~ 'bb.task.%'::text))
);


--
-- Name: task_dag; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_dag (
    id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    from_task_id integer NOT NULL,
    to_task_id integer NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: task_dag_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_dag_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_dag_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_dag_id_seq OWNED BY public.task_dag.id;


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
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    task_id integer NOT NULL,
    sheet_id integer,
    attempt integer NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    started_ts bigint DEFAULT 0 NOT NULL,
    code integer DEFAULT 0 NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
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
    created_ts timestamp with time zone DEFAULT now() NOT NULL,
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
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: vcs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vcs (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    resource_id text NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    instance_url text NOT NULL,
    access_token text DEFAULT ''::text NOT NULL,
    CONSTRAINT vcs_instance_url_check CHECK ((((instance_url ~~ 'http://%'::text) OR (instance_url ~~ 'https://%'::text)) AND (instance_url = rtrim(instance_url, '/'::text)))),
    CONSTRAINT vcs_type_check CHECK ((type = ANY (ARRAY['GITLAB'::text, 'GITHUB'::text, 'BITBUCKET'::text, 'AZURE_DEVOPS'::text])))
);


--
-- Name: vcs_connector; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vcs_connector (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    vcs_id integer NOT NULL,
    project_id integer NOT NULL,
    resource_id text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: vcs_connector_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.vcs_connector_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: vcs_connector_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.vcs_connector_id_seq OWNED BY public.vcs_connector.id;


--
-- Name: vcs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.vcs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: vcs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.vcs_id_seq OWNED BY public.vcs.id;


--
-- Name: worksheet; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.worksheet (
    id integer NOT NULL,
    row_status public.row_status DEFAULT 'NORMAL'::public.row_status NOT NULL,
    creator_id integer NOT NULL,
    created_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    updater_id integer NOT NULL,
    updated_ts bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    project_id integer NOT NULL,
    database_id integer,
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
-- Name: activity id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity ALTER COLUMN id SET DEFAULT nextval('public.activity_id_seq'::regclass);


--
-- Name: anomaly id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.anomaly ALTER COLUMN id SET DEFAULT nextval('public.anomaly_id_seq'::regclass);


--
-- Name: audit_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_log ALTER COLUMN id SET DEFAULT nextval('public.audit_log_id_seq'::regclass);


--
-- Name: branch id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.branch ALTER COLUMN id SET DEFAULT nextval('public.branch_id_seq'::regclass);


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
-- Name: deployment_config id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployment_config ALTER COLUMN id SET DEFAULT nextval('public.deployment_config_id_seq'::regclass);


--
-- Name: environment id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment ALTER COLUMN id SET DEFAULT nextval('public.environment_id_seq'::regclass);


--
-- Name: export_archive id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.export_archive ALTER COLUMN id SET DEFAULT nextval('public.export_archive_id_seq'::regclass);


--
-- Name: external_approval id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_approval ALTER COLUMN id SET DEFAULT nextval('public.external_approval_id_seq'::regclass);


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
-- Name: instance_user id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_user ALTER COLUMN id SET DEFAULT nextval('public.instance_user_id_seq'::regclass);


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
-- Name: slow_query id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slow_query ALTER COLUMN id SET DEFAULT nextval('public.slow_query_id_seq'::regclass);


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
-- Name: task_dag id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_dag ALTER COLUMN id SET DEFAULT nextval('public.task_dag_id_seq'::regclass);


--
-- Name: task_run id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run ALTER COLUMN id SET DEFAULT nextval('public.task_run_id_seq'::regclass);


--
-- Name: task_run_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run_log ALTER COLUMN id SET DEFAULT nextval('public.task_run_log_id_seq'::regclass);


--
-- Name: vcs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs ALTER COLUMN id SET DEFAULT nextval('public.vcs_id_seq'::regclass);


--
-- Name: vcs_connector id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs_connector ALTER COLUMN id SET DEFAULT nextval('public.vcs_connector_id_seq'::regclass);


--
-- Name: worksheet id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet ALTER COLUMN id SET DEFAULT nextval('public.worksheet_id_seq'::regclass);


--
-- Name: worksheet_organizer id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet_organizer ALTER COLUMN id SET DEFAULT nextval('public.worksheet_organizer_id_seq'::regclass);


--
-- Data for Name: activity; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (102, 'NORMAL', 101, 1699026391, 101, 1712734700, '', 101, 'bb.member.create', 'INFO', '', '{"role": "OWNER", "principalId": 101, "memberStatus": "ACTIVE", "principalName": "Demo", "principalEmail": "demo@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (119, 'NORMAL', 102, 1699028630, 102, 1712734700, '', 102, 'bb.member.create', 'INFO', '', '{"role": "DBA", "principalId": 102, "memberStatus": "ACTIVE", "principalName": "dba1", "principalEmail": "dba1@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (120, 'NORMAL', 103, 1699028631, 103, 1712734700, '', 103, 'bb.member.create', 'INFO', '', '{"role": "DBA", "principalId": 103, "memberStatus": "ACTIVE", "principalName": "dba2", "principalEmail": "dba2@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (121, 'NORMAL', 104, 1699028631, 104, 1712734700, '', 104, 'bb.member.create', 'INFO', '', '{"role": "DEVELOPER", "principalId": 104, "memberStatus": "ACTIVE", "principalName": "dev1", "principalEmail": "dev1@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (122, 'NORMAL', 105, 1699028631, 105, 1712734700, '', 105, 'bb.member.create', 'INFO', '', '{"role": "DEVELOPER", "principalId": 105, "memberStatus": "ACTIVE", "principalName": "dev2", "principalEmail": "dev2@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (123, 'NORMAL', 106, 1699028631, 106, 1712734700, '', 106, 'bb.member.create', 'INFO', '', '{"role": "DEVELOPER", "principalId": 106, "memberStatus": "ACTIVE", "principalName": "dev3", "principalEmail": "dev3@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (124, 'NORMAL', 107, 1699028631, 107, 1712734700, '', 107, 'bb.member.create', 'INFO', '', '{"role": "DEVELOPER", "principalId": 107, "memberStatus": "ACTIVE", "principalName": "dev4", "principalEmail": "dev4@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (134, 'NORMAL', 101, 1699028728, 101, 1699028728, 'projects/batch-project', 103, 'bb.project.member.create', 'INFO', 'Granted dev1 to dev1@example.com (OWNER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (159, 'NORMAL', 108, 1699109310, 108, 1712734700, '', 108, 'bb.member.create', 'INFO', '', '{"role": "DEVELOPER", "principalId": 108, "memberStatus": "ACTIVE", "principalName": "qa1", "principalEmail": "qa1@example.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (103, 'NORMAL', 101, 1699027049, 101, 1699027049, 'projects/default', 1, 'bb.project.database.transfer', 'INFO', 'Transferred out database "hr_prod_vcs" to project "GitOps Project".', '{"databaseId": 109, "databaseName": "hr_prod_vcs"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (104, 'NORMAL', 101, 1699027049, 101, 1699027049, 'projects/gitops-project', 102, 'bb.project.database.transfer', 'INFO', 'Transferred in database "hr_prod_vcs" from project "Default".', '{"databaseId": 109, "databaseName": "hr_prod_vcs"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (107, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/default', 1, 'bb.project.database.transfer', 'INFO', 'Transferred out database "hr_prod_2" to project "Batch Project".', '{"databaseId": 104, "databaseName": "hr_prod_2"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (108, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/batch-project', 103, 'bb.project.database.transfer', 'INFO', 'Transferred in database "hr_prod_2" from project "Default".', '{"databaseId": 104, "databaseName": "hr_prod_2"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (110, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/default', 1, 'bb.project.database.transfer', 'INFO', 'Transferred out database "hr_prod_5" to project "Batch Project".', '{"databaseId": 107, "databaseName": "hr_prod_5"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (111, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/batch-project', 103, 'bb.project.database.transfer', 'INFO', 'Transferred in database "hr_prod_5" from project "Default".', '{"databaseId": 107, "databaseName": "hr_prod_5"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (109, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/default', 1, 'bb.project.database.transfer', 'INFO', 'Transferred out database "hr_prod_1" to project "Batch Project".', '{"databaseId": 103, "databaseName": "hr_prod_1"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (112, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/batch-project', 103, 'bb.project.database.transfer', 'INFO', 'Transferred in database "hr_prod_1" from project "Default".', '{"databaseId": 103, "databaseName": "hr_prod_1"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (113, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/default', 1, 'bb.project.database.transfer', 'INFO', 'Transferred out database "hr_prod_6" to project "Batch Project".', '{"databaseId": 108, "databaseName": "hr_prod_6"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (114, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/batch-project', 103, 'bb.project.database.transfer', 'INFO', 'Transferred in database "hr_prod_6" from project "Default".', '{"databaseId": 108, "databaseName": "hr_prod_6"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (115, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/default', 1, 'bb.project.database.transfer', 'INFO', 'Transferred out database "hr_prod_3" to project "Batch Project".', '{"databaseId": 105, "databaseName": "hr_prod_3"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (116, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/batch-project', 103, 'bb.project.database.transfer', 'INFO', 'Transferred in database "hr_prod_3" from project "Default".', '{"databaseId": 105, "databaseName": "hr_prod_3"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (117, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/default', 1, 'bb.project.database.transfer', 'INFO', 'Transferred out database "hr_prod_4" to project "Batch Project".', '{"databaseId": 106, "databaseName": "hr_prod_4"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (118, 'NORMAL', 101, 1699027712, 101, 1699027712, 'projects/batch-project', 103, 'bb.project.database.transfer', 'INFO', 'Transferred in database "hr_prod_4" from project "Default".', '{"databaseId": 106, "databaseName": "hr_prod_4"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (125, 'NORMAL', 101, 1699028682, 101, 1699028682, 'projects/project-sample', 101, 'bb.project.member.create', 'INFO', 'Granted dba1 to dba1@example.com (RELEASER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (126, 'NORMAL', 101, 1699028691, 101, 1699028691, 'projects/project-sample', 101, 'bb.project.member.create', 'INFO', 'Granted dev1 to dev1@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (127, 'NORMAL', 101, 1699028691, 101, 1699028691, 'projects/project-sample', 101, 'bb.project.member.create', 'INFO', 'Granted dev2 to dev2@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (128, 'NORMAL', 101, 1699028691, 101, 1699028691, 'projects/project-sample', 101, 'bb.project.member.create', 'INFO', 'Granted dev3 to dev3@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (129, 'NORMAL', 101, 1699028691, 101, 1699028691, 'projects/project-sample', 101, 'bb.project.member.create', 'INFO', 'Granted dev4 to dev4@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (130, 'NORMAL', 101, 1699028728, 101, 1699028728, 'projects/batch-project', 103, 'bb.project.member.create', 'INFO', 'Granted dev3 to dev3@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (131, 'NORMAL', 101, 1699028728, 101, 1699028728, 'projects/batch-project', 103, 'bb.project.member.create', 'INFO', 'Granted dev4 to dev4@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (132, 'NORMAL', 101, 1699028728, 101, 1699028728, 'projects/batch-project', 103, 'bb.project.member.create', 'INFO', 'Granted dba1 to dba1@example.com (OWNER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (133, 'NORMAL', 101, 1699028728, 101, 1699028728, 'projects/batch-project', 103, 'bb.project.member.create', 'INFO', 'Granted dba2 to dba2@example.com (OWNER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (135, 'NORMAL', 101, 1699028728, 101, 1699028728, 'projects/batch-project', 103, 'bb.project.member.create', 'INFO', 'Granted dev2 to dev2@example.com (OWNER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (136, 'NORMAL', 101, 1699028792, 101, 1699028792, 'projects/gitops-project', 102, 'bb.project.member.create', 'INFO', 'Granted dev3 to dev3@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (137, 'NORMAL', 101, 1699028792, 101, 1699028792, 'projects/gitops-project', 102, 'bb.project.member.create', 'INFO', 'Granted dev4 to dev4@example.com (DEVELOPER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (138, 'NORMAL', 101, 1699028792, 101, 1699028792, 'projects/gitops-project', 102, 'bb.project.member.create', 'INFO', 'Granted dev1 to dev1@example.com (OWNER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (139, 'NORMAL', 101, 1699028792, 101, 1699028792, 'projects/gitops-project', 102, 'bb.project.member.create', 'INFO', 'Granted dev2 to dev2@example.com (OWNER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (143, 'NORMAL', 104, 1699029997, 104, 1699029997, 'projects/batch-project', 103, 'bb.project.member.delete', 'INFO', 'Revoked OWNER from dev2 (dev2@example.com).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (144, 'NORMAL', 104, 1699030005, 104, 1699030005, 'projects/batch-project', 103, 'bb.project.member.create', 'INFO', 'Granted dev2 to dev2@example.com (QUERIER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (145, 'NORMAL', 104, 1699030022, 104, 1699030022, 'projects/batch-project', 103, 'bb.project.member.create', 'INFO', 'Granted dev1 to dev1@example.com (QUERIER).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (146, 'NORMAL', 104, 1699030025, 104, 1699030025, 'projects/batch-project', 103, 'bb.project.member.delete', 'INFO', 'Revoked OWNER from dev1 (dev1@example.com).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (160, 'NORMAL', 101, 1699109324, 101, 1699109324, 'projects/project-sample', 101, 'bb.project.member.create', 'INFO', 'Granted qa1 to qa1@example.com (tester).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (163, 'NORMAL', 101, 1699109900, 101, 1699109900, 'projects/project-sample', 101, 'bb.project.member.create', 'INFO', 'Granted Demo to demo@example.com (tester).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (165, 'NORMAL', 101, 1699109983, 101, 1699109983, 'projects/project-sample', 101, 'bb.project.member.delete', 'INFO', 'Revoked tester from Demo (demo@example.com).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (101, 'NORMAL', 101, 1699026391, 101, 1699026391, 'projects/project-sample', 101, 'bb.issue.create', 'INFO', '', '{"issueName": "👉👉👉 [START HERE] Add email column to Employee table"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (105, 'NORMAL', 1, 1699027633, 1, 1699027633, 'projects/gitops-project', 102, 'bb.issue.create', 'INFO', '', '{"issueName": "feat: add city to employee table"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (158, 'NORMAL', 106, 1699032519, 106, 1699032519, 'projects/batch-project', 103, 'bb.issue.create', 'INFO', '', '{"issueName": "Add Investor Relation department"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (161, 'NORMAL', 104, 1699109832, 104, 1699109832, 'projects/project-sample', 104, 'bb.issue.create', 'INFO', '', '{"issueName": "[hr_prod] Alter schema @11-04 22:56 UTC+0800"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (162, 'NORMAL', 1, 1699109832, 1, 1699109832, 'projects/project-sample', 104, 'bb.issue.approval.notify', 'INFO', '', '{"approvalStep": {"type": "ANY", "nodes": [{"role": "roles/tester", "type": "ANY_IN_GROUP"}]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (164, 'NORMAL', 104, 1699109967, 104, 1699109967, 'projects/project-sample', 104, 'bb.issue.status.update', 'INFO', '', '{"issueName": "[hr_prod] Alter schema @11-04 22:56 UTC+0800", "newStatus": "CANCELED", "oldStatus": "OPEN"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (166, 'NORMAL', 104, 1699110335, 104, 1699110335, 'projects/project-sample', 105, 'bb.issue.create', 'INFO', '', '{"issueName": "Add performance table"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (167, 'NORMAL', 1, 1699110336, 1, 1699110336, 'projects/project-sample', 105, 'bb.issue.approval.notify', 'INFO', '', '{"approvalStep": {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (174, 'NORMAL', 1, 1702562147, 1, 1702562147, 'projects/gitops-project', 102, 'bb.issue.status.update', 'INFO', '', '{"issueName": "feat: add city to employee table", "newStatus": "DONE", "oldStatus": "OPEN"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (170, 'NORMAL', 101, 1702562144, 101, 1702562144, 'projects/gitops-project', 102, 'bb.pipeline.taskrun.status.update', 'INFO', '', '{"taskId": 103, "taskName": "DDL(schema) for database \"hr_prod_vcs\"", "issueName": "feat: add city to employee table", "newStatus": "PENDING"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (172, 'NORMAL', 1, 1702562147, 1, 1702562147, 'projects/gitops-project', 102, 'bb.pipeline.taskrun.status.update', 'INFO', '', '{"taskId": 103, "taskName": "DDL(schema) for database \"hr_prod_vcs\"", "issueName": "feat: add city to employee table", "newStatus": "DONE"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (173, 'NORMAL', 1, 1702562147, 1, 1702562147, 'projects/gitops-project', 102, 'bb.pipeline.stage.status.update', 'INFO', '', '{"stageId": 103, "issueName": "feat: add city to employee table", "stageName": "Prod Stage", "stageStatusUpdateType": "END"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (140, 'NORMAL', 101, 1699029734, 101, 1699029734, 'projects/project-sample', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM salary;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM salary;", "adviceList": null, "databaseId": 102, "durationNs": 5067000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (141, 'NORMAL', 101, 1699029868, 101, 1699029868, 'projects/project-sample', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM salary;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM salary;", "adviceList": null, "databaseId": 102, "durationNs": 3585000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (142, 'NORMAL', 104, 1699029898, 104, 1699029898, 'projects/batch-project', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM salary"` in database "hr_prod_1" of instance 102.', '{"error": "", "statement": "SELECT * FROM salary", "adviceList": null, "databaseId": 103, "durationNs": 5666000, "instanceId": 102, "databaseName": "hr_prod_1", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (147, 'NORMAL', 104, 1699030039, 104, 1699030039, 'projects/batch-project', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM department"` in database "hr_prod_1" of instance 102.', '{"error": "", "statement": "SELECT * FROM department", "adviceList": null, "databaseId": 103, "durationNs": 2445000, "instanceId": 102, "databaseName": "hr_prod_1", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (148, 'NORMAL', 104, 1699030045, 104, 1699030045, 'projects/batch-project', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM department"` in database "hr_prod_1" of instance 102.', '{"error": "", "statement": "SELECT * FROM department", "adviceList": null, "databaseId": 103, "durationNs": 1490000, "instanceId": 102, "databaseName": "hr_prod_1", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (149, 'NORMAL', 104, 1699030045, 104, 1699030045, 'projects/batch-project', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM department"` in database "hr_prod_2" of instance 102.', '{"error": "", "statement": "SELECT * FROM department", "adviceList": null, "databaseId": 104, "durationNs": 1715000, "instanceId": 102, "databaseName": "hr_prod_2", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (150, 'NORMAL', 104, 1699030045, 104, 1699030045, 'projects/batch-project', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM department"` in database "hr_prod_3" of instance 102.', '{"error": "", "statement": "SELECT * FROM department", "adviceList": null, "databaseId": 105, "durationNs": 1481000, "instanceId": 102, "databaseName": "hr_prod_3", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (151, 'NORMAL', 104, 1699030045, 104, 1699030045, 'projects/batch-project', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM department"` in database "hr_prod_4" of instance 102.', '{"error": "", "statement": "SELECT * FROM department", "adviceList": null, "databaseId": 106, "durationNs": 1159000, "instanceId": 102, "databaseName": "hr_prod_4", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (152, 'NORMAL', 104, 1699030045, 104, 1699030045, 'projects/batch-project', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM department"` in database "hr_prod_5" of instance 102.', '{"error": "", "statement": "SELECT * FROM department", "adviceList": null, "databaseId": 107, "durationNs": 1010000, "instanceId": 102, "databaseName": "hr_prod_5", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (153, 'NORMAL', 104, 1699030045, 104, 1699030045, 'projects/batch-project', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM department"` in database "hr_prod_6" of instance 102.', '{"error": "", "statement": "SELECT * FROM department", "adviceList": null, "databaseId": 108, "durationNs": 1091000, "instanceId": 102, "databaseName": "hr_prod_6", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (154, 'NORMAL', 101, 1699032082, 101, 1699032082, 'projects/project-sample', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM employee;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM employee;", "adviceList": null, "databaseId": 102, "durationNs": 5898000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (155, 'NORMAL', 101, 1699032153, 101, 1699032153, 'projects/project-sample', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM salary;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM salary;", "adviceList": null, "databaseId": 102, "durationNs": 3934000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (156, 'NORMAL', 101, 1699032179, 101, 1699032179, 'projects/project-sample', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM employee;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM employee;", "adviceList": null, "databaseId": 102, "durationNs": 4910000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (157, 'NORMAL', 101, 1699032394, 101, 1699032394, 'projects/project-sample', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM department;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM department;", "adviceList": null, "databaseId": 102, "durationNs": 2054000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (168, 'NORMAL', 101, 1700552743, 101, 1700552743, 'projects/project-sample', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM employee;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM employee;", "adviceList": null, "databaseId": 102, "durationNs": 10371000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (169, 'NORMAL', 101, 1700552753, 101, 1700552753, 'projects/project-sample', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT * FROM employee;"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT * FROM employee;", "adviceList": null, "databaseId": 102, "durationNs": 7757000, "instanceId": 102, "databaseName": "hr_prod", "instanceName": "Prod Sample Instance"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (175, 'NORMAL', 101, 1712736117, 101, 1712736117, 'projects/project-sample', 101, 'bb.pipeline.task.statement.update', 'INFO', '', '{"taskId": 101, "taskName": "DDL(schema) for database \"hr_test\"", "issueName": "👉👉👉 [START HERE] Add email column to Employee table", "newSheetId": 129, "oldSheetId": 102}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (176, 'NORMAL', 101, 1712736117, 101, 1712736117, 'projects/project-sample', 101, 'bb.pipeline.task.statement.update', 'INFO', '', '{"taskId": 102, "taskName": "DDL(schema) for database \"hr_prod\"", "issueName": "👉👉👉 [START HERE] Add email column to Employee table", "newSheetId": 129, "oldSheetId": 103}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (177, 'NORMAL', 1, 1712736118, 1, 1712736118, 'projects/project-sample', 101, 'bb.issue.approval.notify', 'INFO', '', '{"approvalStep": {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (178, 'NORMAL', 101, 1712736157, 101, 1712736157, 'projects/project-sample', 101, 'bb.pipeline.task.statement.update', 'INFO', '', '{"taskId": 101, "taskName": "DDL(schema) for database \"hr_test\"", "issueName": "👉👉👉 [START HERE] Add email column to Employee table", "newSheetId": 130, "oldSheetId": 129}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (179, 'NORMAL', 101, 1712736157, 101, 1712736157, 'projects/project-sample', 101, 'bb.pipeline.task.statement.update', 'INFO', '', '{"taskId": 102, "taskName": "DDL(schema) for database \"hr_prod\"", "issueName": "👉👉👉 [START HERE] Add email column to Employee table", "newSheetId": 130, "oldSheetId": 129}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (180, 'NORMAL', 1, 1712736163, 1, 1712736163, 'projects/project-sample', 101, 'bb.issue.approval.notify', 'INFO', '', '{"approvalStep": {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (181, 'NORMAL', 109, 1712736378, 109, 1712736378, '', 109, 'bb.member.create', 'INFO', '', '{"role": "workspaceDBA", "principalId": 109, "memberStatus": "ACTIVE", "principalName": "ci", "principalEmail": "ci@service.bytebase.com"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (182, 'NORMAL', 1, 1712737090, 1, 1712737090, 'projects/gitops-project', 106, 'bb.issue.create', 'INFO', '', '{"issueName": "feat: add phone to employee table"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (183, 'NORMAL', 1, 1712737090, 1, 1712737090, 'projects/gitops-project', 102, 'bb.project.repository.push', 'INFO', 'Created issue "feat: add phone to employee table".', '{"issueId": 106, "issueName": "feat: add phone to employee table"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (184, 'NORMAL', 1, 1712737090, 1, 1712737090, 'projects/gitops-project', 106, 'bb.issue.approval.notify', 'INFO', '', '{"approvalStep": {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (185, 'NORMAL', 101, 1712757522, 101, 1712757522, 'projects/default', 1, 'bb.project.database.transfer', 'INFO', 'Transferred out database "bb" to project "MetaDB Project".', '{"databaseId": 111, "databaseName": "bb"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (186, 'NORMAL', 101, 1712757522, 101, 1712757522, 'projects/metadb-project', 104, 'bb.project.database.transfer', 'INFO', 'Transferred in database "bb" from project "Default".', '{"databaseId": 111, "databaseName": "bb"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (187, 'NORMAL', 101, 1712757578, 101, 1712757577, 'projects/metadb-project', 103, 'bb.sql.query', 'INFO', 'Executed `"SELECT\n  project.resource_id,\n  count(*)\nFROM\n  issue\n  LEFT JOIN project ON issue.project_id = project.id\nWHERE\n  NOT EXISTS (\n    SELECT\n      1\n    FROM\n      task,\n      task_run\n    WHERE\n      task.pipeline_id = issue.pipeline_id\n      AND task.id = task_run.task_id\n      AND task_run.status != ''DONE''\n  )\n  AND issue.status = ''DONE''\nGROUP BY\n  project.resource_id;"` in database "bb" of instance 103.', '{"error": "", "statement": "SELECT\n  project.resource_id,\n  count(*)\nFROM\n  issue\n  LEFT JOIN project ON issue.project_id = project.id\nWHERE\n  NOT EXISTS (\n    SELECT\n      1\n    FROM\n      task,\n      task_run\n    WHERE\n      task.pipeline_id = issue.pipeline_id\n      AND task.id = task_run.task_id\n      AND task_run.status != ''DONE''\n  )\n  AND issue.status = ''DONE''\nGROUP BY\n  project.resource_id;", "adviceList": null, "databaseId": 111, "durationNs": 2000000, "instanceId": 103, "databaseName": "bb"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (194, 'NORMAL', 101, 1715935615, 101, 1715935620, 'projects/project-sample', 102, 'bb.sql.query', 'INFO', 'Executed `"SELECT pg_sleep(5)"` in database "hr_prod" of instance 102.', '{"error": "", "statement": "SELECT pg_sleep(5)", "adviceList": null, "databaseId": 102, "durationNs": 5013435000, "instanceId": 102, "databaseName": "hr_prod"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (188, 'NORMAL', 101, 1712757749, 101, 1712757749, 'projects/metadb-project', 103, 'bb.sql.query', 'INFO', 'Executed `"SELECT project.resource_id, count(*)\nFROM issue\nLEFT JOIN project ON issue.project_id = project.id\nWHERE EXISTS (\n        SELECT 1 FROM activity, principal, member\n        WHERE TO_TIMESTAMP(activity.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\n        AND activity.type = ''bb.issue.comment.create''\n        AND activity.container_id = issue.id\n        AND activity.payload->''approvalEvent''->>''status'' = ''APPROVED''\n        AND activity.creator_id = principal.id\n        AND principal.id = member.principal_id\n        AND member.\"role\" = ''DBA''\n) AND TO_TIMESTAMP(issue.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\nGROUP BY project.resource_id;"` in database "bb" of instance 103.', '{"error": "", "statement": "SELECT project.resource_id, count(*)\nFROM issue\nLEFT JOIN project ON issue.project_id = project.id\nWHERE EXISTS (\n        SELECT 1 FROM activity, principal, member\n        WHERE TO_TIMESTAMP(activity.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\n        AND activity.type = ''bb.issue.comment.create''\n        AND activity.container_id = issue.id\n        AND activity.payload->''approvalEvent''->>''status'' = ''APPROVED''\n        AND activity.creator_id = principal.id\n        AND principal.id = member.principal_id\n        AND member.\"role\" = ''DBA''\n) AND TO_TIMESTAMP(issue.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\nGROUP BY project.resource_id;", "adviceList": null, "databaseId": 111, "durationNs": 3567000, "instanceId": 103, "databaseName": "bb"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (189, 'NORMAL', 101, 1712757976, 101, 1712757976, 'projects/metadb-project', 103, 'bb.sql.query', 'INFO', 'Executed `"SELECT\n  issue.id AS issue_id,\n  issue.creator_id as creator_id,\n  COALESCE(\n    array_agg(DISTINCT principal.email) FILTER (\n      WHERE\n        task_run.creator_id IS NOT NULL\n    ),\n    ''{}''\n  ) AS releaser_emails\nFROM\n  issue\n  LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\n  LEFT JOIN task_run ON task_run.task_id = task.id\n  LEFT JOIN principal ON task_run.creator_id = principal.id\nWHERE\n  principal.id = issue.creator_id\n  AND issue.status = ''DONE''\nGROUP BY\n  issue.id\nORDER BY\n  issue.id\n"` in database "bb" of instance 103.', '{"error": "", "statement": "SELECT\n  issue.id AS issue_id,\n  issue.creator_id as creator_id,\n  COALESCE(\n    array_agg(DISTINCT principal.email) FILTER (\n      WHERE\n        task_run.creator_id IS NOT NULL\n    ),\n    ''{}''\n  ) AS releaser_emails\nFROM\n  issue\n  LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\n  LEFT JOIN task_run ON task_run.task_id = task.id\n  LEFT JOIN principal ON task_run.creator_id = principal.id\nWHERE\n  principal.id = issue.creator_id\n  AND issue.status = ''DONE''\nGROUP BY\n  issue.id\nORDER BY\n  issue.id\n", "adviceList": null, "databaseId": 111, "durationNs": 2620000, "instanceId": 103, "databaseName": "bb"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (190, 'NORMAL', 101, 1712758421, 101, 1712758420, 'projects/metadb-project', 103, 'bb.sql.query', 'INFO', 'Executed `"WITH issue_approvers AS (\n  SELECT\n    issue.id AS issue_id,\n    COALESCE(\n      array_agg(DISTINCT principal.email) FILTER (\n        WHERE\n          x.status = ''APPROVED''\n      ),\n      ''{}''\n    ) AS approver_emails\n  FROM\n    issue\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \"principalId\" int) ON TRUE\n    LEFT JOIN principal ON principal.id = x.\"principalId\"\n  GROUP BY\n    issue.id\n  ORDER BY\n    issue.id\n),\nissue_releasers AS (\n  SELECT\n    issue.id AS issue_id,\n    COALESCE(\n      array_agg(DISTINCT principal.email) FILTER (\n        WHERE\n          task_run.creator_id IS NOT NULL\n      ),\n      ''{}''\n    ) AS releaser_emails\n  FROM\n    issue\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\n    LEFT JOIN task_run ON task_run.task_id = task.id\n    LEFT JOIN principal ON task_run.creator_id = principal.id\n  GROUP BY\n    issue.id\n  ORDER BY\n    issue.id\n)\n\nSELECT\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\n  COUNT(issue.id) AS issue_count,\n  ia.approver_emails,\n  ir.releaser_emails\nFROM\n  issue\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\nWHERE\n  issue.status = ''DONE''\n  AND ia.approver_emails @> ir.releaser_emails\n  AND ir.releaser_emails @> ia.approver_emails\n  AND array_length(ir.releaser_emails, 1) > 0\nGROUP BY\n  month,\n  ia.approver_emails,\n  ir.releaser_emails\nORDER BY\n  month;"` in database "bb" of instance 103.', '{"error": "", "statement": "WITH issue_approvers AS (\n  SELECT\n    issue.id AS issue_id,\n    COALESCE(\n      array_agg(DISTINCT principal.email) FILTER (\n        WHERE\n          x.status = ''APPROVED''\n      ),\n      ''{}''\n    ) AS approver_emails\n  FROM\n    issue\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \"principalId\" int) ON TRUE\n    LEFT JOIN principal ON principal.id = x.\"principalId\"\n  GROUP BY\n    issue.id\n  ORDER BY\n    issue.id\n),\nissue_releasers AS (\n  SELECT\n    issue.id AS issue_id,\n    COALESCE(\n      array_agg(DISTINCT principal.email) FILTER (\n        WHERE\n          task_run.creator_id IS NOT NULL\n      ),\n      ''{}''\n    ) AS releaser_emails\n  FROM\n    issue\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\n    LEFT JOIN task_run ON task_run.task_id = task.id\n    LEFT JOIN principal ON task_run.creator_id = principal.id\n  GROUP BY\n    issue.id\n  ORDER BY\n    issue.id\n)\n\nSELECT\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\n  COUNT(issue.id) AS issue_count,\n  ia.approver_emails,\n  ir.releaser_emails\nFROM\n  issue\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\nWHERE\n  issue.status = ''DONE''\n  AND ia.approver_emails @> ir.releaser_emails\n  AND ir.releaser_emails @> ia.approver_emails\n  AND array_length(ir.releaser_emails, 1) > 0\nGROUP BY\n  month,\n  ia.approver_emails,\n  ir.releaser_emails\nORDER BY\n  month;", "adviceList": null, "databaseId": 111, "durationNs": 2993000, "instanceId": 103, "databaseName": "bb"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (191, 'NORMAL', 101, 1712758550, 101, 1712758550, 'projects/metadb-project', 103, 'bb.sql.query', 'INFO', 'Executed `"WITH issue_approvers AS (\n  SELECT\n    issue.id AS issue_id,\n    COALESCE(\n      array_agg(DISTINCT principal.email) FILTER (\n        WHERE\n          x.status = ''APPROVED''\n      ),\n      ''{}''\n    ) AS approver_emails\n  FROM\n    issue\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \"principalId\" int) ON TRUE\n    LEFT JOIN principal ON principal.id = x.\"principalId\"\n  GROUP BY\n    issue.id\n  ORDER BY\n    issue.id\n),\nissue_releasers AS (\n  SELECT\n    issue.id AS issue_id,\n    COALESCE(\n      array_agg(DISTINCT principal.email) FILTER (\n        WHERE\n          task_run.creator_id IS NOT NULL\n      ),\n      ''{}''\n    ) AS releaser_emails\n  FROM\n    issue\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\n    LEFT JOIN task_run ON task_run.task_id = task.id\n    LEFT JOIN principal ON task_run.creator_id = principal.id\n  GROUP BY\n    issue.id\n  ORDER BY\n    issue.id\n)\n\nSELECT\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\n  COUNT(issue.id) AS issue_count,\n  ia.approver_emails,\n  ir.releaser_emails\nFROM\n  issue\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\nWHERE\n  issue.status = ''DONE''\n  AND ia.approver_emails @> ir.releaser_emails\n  AND ir.releaser_emails @> ia.approver_emails\n  AND array_length(ir.releaser_emails, 1) > 0\nGROUP BY\n  month,\n  ia.approver_emails,\n  ir.releaser_emails\nORDER BY\n  month;"` in database "bb" of instance 103.', '{"error": "", "statement": "WITH issue_approvers AS (\n  SELECT\n    issue.id AS issue_id,\n    COALESCE(\n      array_agg(DISTINCT principal.email) FILTER (\n        WHERE\n          x.status = ''APPROVED''\n      ),\n      ''{}''\n    ) AS approver_emails\n  FROM\n    issue\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \"principalId\" int) ON TRUE\n    LEFT JOIN principal ON principal.id = x.\"principalId\"\n  GROUP BY\n    issue.id\n  ORDER BY\n    issue.id\n),\nissue_releasers AS (\n  SELECT\n    issue.id AS issue_id,\n    COALESCE(\n      array_agg(DISTINCT principal.email) FILTER (\n        WHERE\n          task_run.creator_id IS NOT NULL\n      ),\n      ''{}''\n    ) AS releaser_emails\n  FROM\n    issue\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\n    LEFT JOIN task_run ON task_run.task_id = task.id\n    LEFT JOIN principal ON task_run.creator_id = principal.id\n  GROUP BY\n    issue.id\n  ORDER BY\n    issue.id\n)\n\nSELECT\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\n  COUNT(issue.id) AS issue_count,\n  ia.approver_emails,\n  ir.releaser_emails\nFROM\n  issue\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\nWHERE\n  issue.status = ''DONE''\n  AND ia.approver_emails @> ir.releaser_emails\n  AND ir.releaser_emails @> ia.approver_emails\n  AND array_length(ir.releaser_emails, 1) > 0\nGROUP BY\n  month,\n  ia.approver_emails,\n  ir.releaser_emails\nORDER BY\n  month;", "adviceList": null, "databaseId": 111, "durationNs": 3365000, "instanceId": 103, "databaseName": "bb"}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (192, 'NORMAL', 101, 1712758577, 101, 1712758577, 'projects/metadb-project', 104, 'bb.project.member.create', 'INFO', 'Granted dev1 to dev1@example.com (projectQuerier).', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.activity (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_container, container_id, type, level, comment, payload) VALUES (193, 'NORMAL', 104, 1712758765, 104, 1712758764, 'projects/metadb-project', 103, 'bb.sql.query', 'INFO', 'Executed `"-- Fully completed issues by project\nSELECT\n  project.resource_id,\n  count(*)\nFROM\n  issue\n  LEFT JOIN project ON issue.project_id = project.id\nWHERE\n  NOT EXISTS (\n    SELECT\n      1\n    FROM\n      task,\n      task_run\n    WHERE\n      task.pipeline_id = issue.pipeline_id\n      AND task.id = task_run.task_id\n      AND task_run.status != ''DONE''\n  )\n  AND issue.status = ''DONE''\nGROUP BY\n  project.resource_id;"` in database "bb" of instance 103.', '{"error": "", "statement": "-- Fully completed issues by project\nSELECT\n  project.resource_id,\n  count(*)\nFROM\n  issue\n  LEFT JOIN project ON issue.project_id = project.id\nWHERE\n  NOT EXISTS (\n    SELECT\n      1\n    FROM\n      task,\n      task_run\n    WHERE\n      task.pipeline_id = issue.pipeline_id\n      AND task.id = task_run.task_id\n      AND task_run.status != ''DONE''\n  )\n  AND issue.status = ''DONE''\nGROUP BY\n  project.resource_id;", "adviceList": null, "databaseId": 111, "durationNs": 2810000, "instanceId": 103, "databaseName": "bb"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: anomaly; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.anomaly (id, row_status, creator_id, created_ts, updater_id, updated_ts, project, instance_id, database_id, type, payload) VALUES (107, 'NORMAL', 1, 1737001377, 1, 1737001377, 'gitops-project', 102, 109, 'bb.anomaly.database.schema.drift', '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: audit_log; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.audit_log (id, created_ts, payload) VALUES (101, 1699029734, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM salary;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.005067s\", \"statement\": \"SELECT * FROM salary;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (102, 1699029868, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM salary;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.003585s\", \"statement\": \"SELECT * FROM salary;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (103, 1699029898, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_1\", \"statement\": \"SELECT * FROM salary\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_1", "response": "{\"results\": [{\"latency\": \"0.005666s\", \"statement\": \"SELECT * FROM salary\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (104, 1699030039, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_1\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_1", "response": "{\"results\": [{\"latency\": \"0.002445s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (105, 1699030045, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_1\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_1", "response": "{\"results\": [{\"latency\": \"0.00149s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (106, 1699030045, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_2\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_2", "response": "{\"results\": [{\"latency\": \"0.001715s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (107, 1699030045, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_3\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_3", "response": "{\"results\": [{\"latency\": \"0.001481s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (108, 1699030045, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_4\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_4", "response": "{\"results\": [{\"latency\": \"0.001159s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (109, 1699030045, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_5\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_5", "response": "{\"results\": [{\"latency\": \"0.00101s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (110, 1699030045, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/batch-project", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod_6\", \"statement\": \"SELECT * FROM department\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_6", "response": "{\"results\": [{\"latency\": \"0.001091s\", \"statement\": \"SELECT * FROM department\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (111, 1699032082, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM employee;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.005898s\", \"statement\": \"SELECT * FROM employee;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (112, 1699032153, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM salary;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.003934s\", \"statement\": \"SELECT * FROM salary;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (113, 1699032179, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM employee;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.00491s\", \"statement\": \"SELECT * FROM employee;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (114, 1699032394, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM department;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.002054s\", \"statement\": \"SELECT * FROM department;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (115, 1700552743, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM employee;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.010371s\", \"statement\": \"SELECT * FROM employee;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (116, 1700552753, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\": \"instances/prod-sample-instance/databases/hr_prod\", \"statement\": \"SELECT * FROM employee;\"}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\": [{\"latency\": \"0.007757s\", \"statement\": \"SELECT * FROM employee;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (125, 1720669329, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"connectionDatabase\":\"hr_prod\", \"statement\":\"SELECT * FROM employee;\", \"limit\":1000}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\":[{\"columnNames\":[\"emp_no\", \"birth_date\", \"first_name\", \"last_name\", \"gender\", \"hire_date\"], \"columnTypeNames\":[\"INT4\", \"DATE\", \"TEXT\", \"TEXT\", \"TEXT\", \"DATE\"], \"masked\":[false, false, true, true, false, false], \"sensitive\":[false, false, true, true, false, false], \"latency\":\"0.008539708s\", \"statement\":\"WITH result AS (\\nSELECT * FROM employee\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (117, 1712757578, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"SELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.002s\", \"statement\": \"SELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (118, 1712757749, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"SELECT project.resource_id, count(*)\\nFROM issue\\nLEFT JOIN project ON issue.project_id = project.id\\nWHERE EXISTS (\\n        SELECT 1 FROM activity, principal, member\\n        WHERE TO_TIMESTAMP(activity.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\\n        AND activity.type = ''bb.issue.comment.create''\\n        AND activity.container_id = issue.id\\n        AND activity.payload->''approvalEvent''->>''status'' = ''APPROVED''\\n        AND activity.creator_id = principal.id\\n        AND principal.id = member.principal_id\\n        AND member.\\\"role\\\" = ''DBA''\\n) AND TO_TIMESTAMP(issue.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\\nGROUP BY project.resource_id;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.003567s\", \"statement\": \"SELECT project.resource_id, count(*)\\nFROM issue\\nLEFT JOIN project ON issue.project_id = project.id\\nWHERE EXISTS (\\n        SELECT 1 FROM activity, principal, member\\n        WHERE TO_TIMESTAMP(activity.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\\n        AND activity.type = ''bb.issue.comment.create''\\n        AND activity.container_id = issue.id\\n        AND activity.payload->''approvalEvent''->>''status'' = ''APPROVED''\\n        AND activity.creator_id = principal.id\\n        AND principal.id = member.principal_id\\n        AND member.\\\"role\\\" = ''DBA''\\n) AND TO_TIMESTAMP(issue.created_ts)::TIME BETWEEN TIME ''17:30:00+08'' AND ''23:59:59+08''\\nGROUP BY project.resource_id;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (119, 1712757976, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"SELECT\\n  issue.id AS issue_id,\\n  issue.creator_id as creator_id,\\n  COALESCE(\\n    array_agg(DISTINCT principal.email) FILTER (\\n      WHERE\\n        task_run.creator_id IS NOT NULL\\n    ),\\n    ''{}''\\n  ) AS releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n  LEFT JOIN task_run ON task_run.task_id = task.id\\n  LEFT JOIN principal ON task_run.creator_id = principal.id\\nWHERE\\n  principal.id = issue.creator_id\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  issue.id\\nORDER BY\\n  issue.id\\n\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.00262s\", \"statement\": \"SELECT\\n  issue.id AS issue_id,\\n  issue.creator_id as creator_id,\\n  COALESCE(\\n    array_agg(DISTINCT principal.email) FILTER (\\n      WHERE\\n        task_run.creator_id IS NOT NULL\\n    ),\\n    ''{}''\\n  ) AS releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n  LEFT JOIN task_run ON task_run.task_id = task.id\\n  LEFT JOIN principal ON task_run.creator_id = principal.id\\nWHERE\\n  principal.id = issue.creator_id\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  issue.id\\nORDER BY\\n  issue.id\\n\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (120, 1712758421, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"WITH issue_approvers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          x.status = ''APPROVED''\\n      ),\\n      ''{}''\\n    ) AS approver_emails\\n  FROM\\n    issue\\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \\\"principalId\\\" int) ON TRUE\\n    LEFT JOIN principal ON principal.id = x.\\\"principalId\\\"\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n),\\nissue_releasers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          task_run.creator_id IS NOT NULL\\n      ),\\n      ''{}''\\n    ) AS releaser_emails\\n  FROM\\n    issue\\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n    LEFT JOIN task_run ON task_run.task_id = task.id\\n    LEFT JOIN principal ON task_run.creator_id = principal.id\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n)\\n\\nSELECT\\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\\n  COUNT(issue.id) AS issue_count,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\\nWHERE\\n  issue.status = ''DONE''\\n  AND ia.approver_emails @> ir.releaser_emails\\n  AND ir.releaser_emails @> ia.approver_emails\\n  AND array_length(ir.releaser_emails, 1) > 0\\nGROUP BY\\n  month,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nORDER BY\\n  month;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.002993s\", \"statement\": \"WITH issue_approvers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          x.status = ''APPROVED''\\n      ),\\n      ''{}''\\n    ) AS approver_emails\\n  FROM\\n    issue\\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \\\"principalId\\\" int) ON TRUE\\n    LEFT JOIN principal ON principal.id = x.\\\"principalId\\\"\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n),\\nissue_releasers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          task_run.creator_id IS NOT NULL\\n      ),\\n      ''{}''\\n    ) AS releaser_emails\\n  FROM\\n    issue\\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n    LEFT JOIN task_run ON task_run.task_id = task.id\\n    LEFT JOIN principal ON task_run.creator_id = principal.id\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n)\\n\\nSELECT\\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\\n  COUNT(issue.id) AS issue_count,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\\nWHERE\\n  issue.status = ''DONE''\\n  AND ia.approver_emails @> ir.releaser_emails\\n  AND ir.releaser_emails @> ia.approver_emails\\n  AND array_length(ir.releaser_emails, 1) > 0\\nGROUP BY\\n  month,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nORDER BY\\n  month;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (121, 1712758550, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"WITH issue_approvers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          x.status = ''APPROVED''\\n      ),\\n      ''{}''\\n    ) AS approver_emails\\n  FROM\\n    issue\\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \\\"principalId\\\" int) ON TRUE\\n    LEFT JOIN principal ON principal.id = x.\\\"principalId\\\"\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n),\\nissue_releasers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          task_run.creator_id IS NOT NULL\\n      ),\\n      ''{}''\\n    ) AS releaser_emails\\n  FROM\\n    issue\\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n    LEFT JOIN task_run ON task_run.task_id = task.id\\n    LEFT JOIN principal ON task_run.creator_id = principal.id\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n)\\n\\nSELECT\\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\\n  COUNT(issue.id) AS issue_count,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\\nWHERE\\n  issue.status = ''DONE''\\n  AND ia.approver_emails @> ir.releaser_emails\\n  AND ir.releaser_emails @> ia.approver_emails\\n  AND array_length(ir.releaser_emails, 1) > 0\\nGROUP BY\\n  month,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nORDER BY\\n  month;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.003365s\", \"statement\": \"WITH issue_approvers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          x.status = ''APPROVED''\\n      ),\\n      ''{}''\\n    ) AS approver_emails\\n  FROM\\n    issue\\n    LEFT JOIN LATERAL jsonb_to_recordset(issue.payload -> ''approval'' -> ''approvers'') as x(status text, \\\"principalId\\\" int) ON TRUE\\n    LEFT JOIN principal ON principal.id = x.\\\"principalId\\\"\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n),\\nissue_releasers AS (\\n  SELECT\\n    issue.id AS issue_id,\\n    COALESCE(\\n      array_agg(DISTINCT principal.email) FILTER (\\n        WHERE\\n          task_run.creator_id IS NOT NULL\\n      ),\\n      ''{}''\\n    ) AS releaser_emails\\n  FROM\\n    issue\\n    LEFT JOIN task ON issue.pipeline_id = task.pipeline_id\\n    LEFT JOIN task_run ON task_run.task_id = task.id\\n    LEFT JOIN principal ON task_run.creator_id = principal.id\\n  GROUP BY\\n    issue.id\\n  ORDER BY\\n    issue.id\\n)\\n\\nSELECT\\n  date_trunc(''month'', to_timestamp(issue.created_ts)) AS month,\\n  COUNT(issue.id) AS issue_count,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nFROM\\n  issue\\n  LEFT JOIN issue_approvers ia ON ia.issue_id = issue.id\\n  LEFT JOIN issue_releasers ir ON ir.issue_id = issue.id\\nWHERE\\n  issue.status = ''DONE''\\n  AND ia.approver_emails @> ir.releaser_emails\\n  AND ir.releaser_emails @> ia.approver_emails\\n  AND array_length(ir.releaser_emails, 1) > 0\\nGROUP BY\\n  month,\\n  ia.approver_emails,\\n  ir.releaser_emails\\nORDER BY\\n  month;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (122, 1712758765, '{"user": "users/104", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\": \"instances/bytebase-meta/databases/bb\", \"statement\": \"-- Fully completed issues by project\\nSELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\": [{\"latency\": \"0.00281s\", \"statement\": \"-- Fully completed issues by project\\nSELECT\\n  project.resource_id,\\n  count(*)\\nFROM\\n  issue\\n  LEFT JOIN project ON issue.project_id = project.id\\nWHERE\\n  NOT EXISTS (\\n    SELECT\\n      1\\n    FROM\\n      task,\\n      task_run\\n    WHERE\\n      task.pipeline_id = issue.pipeline_id\\n      AND task.id = task_run.task_id\\n      AND task_run.status != ''DONE''\\n  )\\n  AND issue.status = ''DONE''\\nGROUP BY\\n  project.resource_id;\"}]}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (124, 1720666255, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (126, 1720669334, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"connectionDatabase\":\"hr_prod\", \"statement\":\"SELECT * FROM salary;\", \"limit\":1000}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\":[{\"columnNames\":[\"emp_no\", \"amount\", \"from_date\", \"to_date\"], \"columnTypeNames\":[\"INT4\", \"INT4\", \"DATE\", \"DATE\"], \"masked\":[false, true, false, false], \"sensitive\":[false, true, false, false], \"latency\":\"0.008754250s\", \"statement\":\"WITH result AS (\\nSELECT * FROM salary\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (127, 1720669353, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/project-sample", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod\", \"connectionDatabase\":\"hr_prod\", \"statement\":\"SELECT * FROM employee;\", \"limit\":1000}", "resource": "instances/prod-sample-instance/databases/hr_prod", "response": "{\"results\":[{\"columnNames\":[\"emp_no\", \"birth_date\", \"first_name\", \"last_name\", \"gender\", \"hire_date\"], \"columnTypeNames\":[\"INT4\", \"DATE\", \"TEXT\", \"TEXT\", \"TEXT\", \"DATE\"], \"masked\":[false, false, true, true, false, false], \"sensitive\":[false, false, true, true, false, false], \"latency\":\"0.010567041s\", \"statement\":\"WITH result AS (\\nSELECT * FROM employee\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (128, 1721635756, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\", \"web\":true}", "resource": "demo@example.com", "severity": "INFO"}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (129, 1726820034, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\":\"instances/bytebase-meta/databases/bb\",\"statement\":\"SELECT\\n  issue.creator_id,\\n  principal.email,\\n  COUNT(issue.creator_id) AS amount\\nFROM issue\\nINNER JOIN principal\\nON issue.creator_id = principal.id\\nGROUP BY issue.creator_id, principal.email\\nORDER BY COUNT(issue.creator_id) DESC;\",\"limit\":1000,\"dataSourceId\":\"777072ed-539e-4cc2-a41e-6cc2917a7e7c\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\":[{\"columnNames\":[\"creator_id\",\"email\",\"amount\"],\"columnTypeNames\":[\"INT4\",\"TEXT\",\"INT8\"],\"masked\":[false,false,false],\"sensitive\":[false,false,false],\"latency\":\"0.021570708s\",\"statement\":\"WITH result AS (\\nSELECT\\n  issue.creator_id,\\n  principal.email,\\n  COUNT(issue.creator_id) AS amount\\nFROM issue\\nINNER JOIN principal\\nON issue.creator_id = principal.id\\nGROUP BY issue.creator_id, principal.email\\nORDER BY COUNT(issue.creator_id) DESC\\n) SELECT * FROM result LIMIT 1000;\"}],\"allowExport\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:62138", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (130, 1727534004, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (131, 1727534030, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (132, 1727534031, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (133, 1727534072, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (134, 1727534082, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"DISALLOW\"},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (135, 1727534093, '{"user": "users/101", "method": "/bytebase.v1.OrgPolicyService/CreatePolicy", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"policy\":{\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"FALLBACK\"}}}", "response": "{\"name\":\"projects/project-sample/policies/data_source_query\",\"type\":\"DATA_SOURCE_QUERY\",\"dataSourceQueryPolicy\":{\"adminDataSourceRestriction\":\"FALLBACK\"},\"enforce\":true,\"resourceType\":\"PROJECT\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (142, 1736952852, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\",\"email\":\"demo@example.com\",\"title\":\"Demo\",\"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:64249", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (136, 1727534142, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"},\"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (137, 1727534160, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"e543ddfd-f633-4dbe-87f3-6b171a96e20a\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"},\"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (138, 1727534163, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"},\"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (139, 1727534163, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"e543ddfd-f633-4dbe-87f3-6b171a96e20a\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"},\"validateOnly\":true}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (140, 1727534163, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"}}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (141, 1727534163, '{"user": "users/101", "method": "/bytebase.v1.InstanceService/AddDataSource", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance\",\"dataSource\":{\"id\":\"e543ddfd-f633-4dbe-87f3-6b171a96e20a\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\"}}", "resource": "instances/prod-sample-instance", "response": "{\"name\":\"instances/prod-sample-instance\",\"state\":\"ACTIVE\",\"title\":\"Prod Sample Instance\",\"engine\":\"POSTGRES\",\"engineVersion\":\"16.0.2\",\"dataSources\":[{\"id\":\"admin\",\"type\":\"ADMIN\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"351173a6-f320-45c5-8d95-8e17abe08964\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"},{\"id\":\"e543ddfd-f633-4dbe-87f3-6b171a96e20a\",\"type\":\"READ_ONLY\",\"username\":\"bbsample\",\"host\":\"/tmp\",\"port\":\"8084\",\"authenticationType\":\"PASSWORD\",\"redisType\":\"STANDALONE\"}],\"environment\":\"environments/prod\",\"activation\":true,\"options\":{},\"roles\":[{\"name\":\"instances/prod-sample-instance/roles/bbsample\",\"roleName\":\"bbsample\",\"attribute\":\"Superuser Create role Create DB Replication Bypass RLS+\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:51355", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (143, 1736961540, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\",\"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\",\"email\":\"demo@example.com\",\"title\":\"Demo\",\"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:55846", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (144, 1736962796, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\", \"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\", \"email\":\"demo@example.com\", \"title\":\"Demo\", \"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:61958", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (145, 1737001075, '{"user": "users/101", "method": "/bytebase.v1.PlanService/CreatePlan", "parent": "projects/gitops-project", "request": "{\"parent\":\"projects/gitops-project\",\"plan\":{\"name\":\"projects/gitops-project/plans/-102\",\"steps\":[{\"specs\":[{\"id\":\"ff8ecf1c-f037-4544-971c-c3f4c8ff5889\",\"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"sheet\":\"projects/gitops-project/sheets/133\",\"type\":\"BASELINE\"}}]}]}}", "response": "{\"name\":\"projects/gitops-project/plans/108\",\"steps\":[{\"specs\":[{\"id\":\"ff8ecf1c-f037-4544-971c-c3f4c8ff5889\",\"specReleaseSource\":{},\"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"sheet\":\"projects/gitops-project/sheets/133\",\"type\":\"BASELINE\",\"preUpdateBackupDetail\":{}}}]}],\"vcsSource\":{},\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:17:55Z\",\"updateTime\":\"2025-01-16T04:17:55Z\",\"releaseSource\":{},\"deploymentConfigSnapshot\":{\"name\":\"gitops-project/deploymentConfigs/default\",\"schedule\":{\"deployments\":[{\"title\":\"Test Stage\",\"id\":\"0\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"test\"]}]}}},{\"title\":\"Prod Stage\",\"id\":\"1\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"prod\"]}]}}}]}}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49476", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (146, 1737001075, '{"user": "users/101", "method": "/bytebase.v1.IssueService/CreateIssue", "parent": "projects/gitops-project", "request": "{\"parent\":\"projects/gitops-project\",\"issue\":{\"name\":\"projects/gitops-project/issues/-101\",\"title\":\"Establish \\\"hr_prod_vcs\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"plan\":\"projects/gitops-project/plans/108\"}}", "response": "{\"name\":\"projects/gitops-project/issues/108\",\"title\":\"Establish \\\"hr_prod_vcs\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:17:55Z\",\"updateTime\":\"2025-01-16T04:17:55Z\",\"plan\":\"projects/gitops-project/plans/108\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49476", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (147, 1737001075, '{"user": "users/101", "method": "/bytebase.v1.RolloutService/CreateRollout", "parent": "projects/gitops-project", "request": "{\"parent\":\"projects/gitops-project\",\"rollout\":{\"plan\":\"projects/gitops-project/plans/108\"}}", "response": "{\"name\":\"projects/gitops-project/rollouts/108\",\"plan\":\"projects/gitops-project/plans/108\",\"title\":\"Rollout Pipeline\",\"stages\":[{\"name\":\"projects/gitops-project/rollouts/108/stages/111\",\"id\":\"1\",\"title\":\"Prod Stage\",\"tasks\":[{\"name\":\"projects/gitops-project/rollouts/108/stages/111/tasks/114\",\"title\":\"Establish baseline for database \\\"hr_prod_vcs\\\"\",\"specId\":\"ff8ecf1c-f037-4544-971c-c3f4c8ff5889\",\"status\":\"NOT_STARTED\",\"type\":\"DATABASE_SCHEMA_BASELINE\",\"target\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"databaseSchemaBaseline\":{}}]}],\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:17:55Z\",\"updateTime\":\"2025-01-16T04:17:55Z\",\"issue\":\"projects/gitops-project/issues/108\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49476", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (148, 1737001175, '{"user": "users/101", "method": "/bytebase.v1.PlanService/CreatePlan", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"plan\":{\"name\":\"projects/project-sample/plans/-108\",\"steps\":[{\"specs\":[{\"id\":\"231a929d-bb89-4845-8b7c-6e4870116d32\",\"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod\",\"sheet\":\"projects/project-sample/sheets/134\",\"type\":\"BASELINE\"}}]}]}}", "response": "{\"name\":\"projects/project-sample/plans/109\",\"steps\":[{\"specs\":[{\"id\":\"231a929d-bb89-4845-8b7c-6e4870116d32\",\"specReleaseSource\":{},\"changeDatabaseConfig\":{\"target\":\"instances/prod-sample-instance/databases/hr_prod\",\"sheet\":\"projects/project-sample/sheets/134\",\"type\":\"BASELINE\",\"preUpdateBackupDetail\":{}}}]}],\"vcsSource\":{},\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:19:35Z\",\"updateTime\":\"2025-01-16T04:19:35Z\",\"releaseSource\":{},\"deploymentConfigSnapshot\":{\"name\":\"project-sample/deploymentConfigs/default\",\"schedule\":{\"deployments\":[{\"title\":\"Test Stage\",\"id\":\"0\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"test\"]}]}}},{\"title\":\"Prod Stage\",\"id\":\"1\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"prod\"]}]}}}]}}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49477", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (149, 1737001175, '{"user": "users/101", "method": "/bytebase.v1.IssueService/CreateIssue", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"issue\":{\"name\":\"projects/project-sample/issues/-107\",\"title\":\"Establish \\\"hr_prod\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"plan\":\"projects/project-sample/plans/109\"}}", "response": "{\"name\":\"projects/project-sample/issues/109\",\"title\":\"Establish \\\"hr_prod\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:19:35Z\",\"updateTime\":\"2025-01-16T04:19:35Z\",\"plan\":\"projects/project-sample/plans/109\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49477", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (150, 1737001175, '{"user": "users/101", "method": "/bytebase.v1.RolloutService/CreateRollout", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"rollout\":{\"plan\":\"projects/project-sample/plans/109\"}}", "response": "{\"name\":\"projects/project-sample/rollouts/109\",\"plan\":\"projects/project-sample/plans/109\",\"title\":\"Rollout Pipeline\",\"stages\":[{\"name\":\"projects/project-sample/rollouts/109/stages/112\",\"id\":\"1\",\"title\":\"Prod Stage\",\"tasks\":[{\"name\":\"projects/project-sample/rollouts/109/stages/112/tasks/115\",\"title\":\"Establish baseline for database \\\"hr_prod\\\"\",\"specId\":\"231a929d-bb89-4845-8b7c-6e4870116d32\",\"status\":\"NOT_STARTED\",\"type\":\"DATABASE_SCHEMA_BASELINE\",\"target\":\"instances/prod-sample-instance/databases/hr_prod\",\"databaseSchemaBaseline\":{}}]}],\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:19:35Z\",\"updateTime\":\"2025-01-16T04:19:35Z\",\"issue\":\"projects/project-sample/issues/109\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49477", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (151, 1737001312, '{"user": "users/101", "method": "/bytebase.v1.PlanService/CreatePlan", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"plan\":{\"name\":\"projects/project-sample/plans/-102\",\"steps\":[{\"specs\":[{\"id\":\"913aa19f-18e6-42c5-b6e7-2fbb358cffee\",\"changeDatabaseConfig\":{\"target\":\"instances/test-sample-instance/databases/hr_test\",\"sheet\":\"projects/project-sample/sheets/135\",\"type\":\"BASELINE\"}}]}]}}", "response": "{\"name\":\"projects/project-sample/plans/110\",\"steps\":[{\"specs\":[{\"id\":\"913aa19f-18e6-42c5-b6e7-2fbb358cffee\",\"specReleaseSource\":{},\"changeDatabaseConfig\":{\"target\":\"instances/test-sample-instance/databases/hr_test\",\"sheet\":\"projects/project-sample/sheets/135\",\"type\":\"BASELINE\",\"preUpdateBackupDetail\":{}}}]}],\"vcsSource\":{},\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:21:52Z\",\"updateTime\":\"2025-01-16T04:21:52Z\",\"releaseSource\":{},\"deploymentConfigSnapshot\":{\"name\":\"project-sample/deploymentConfigs/default\",\"schedule\":{\"deployments\":[{\"title\":\"Test Stage\",\"id\":\"0\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"test\"]}]}}},{\"title\":\"Prod Stage\",\"id\":\"1\",\"spec\":{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"environment\",\"operator\":\"OPERATOR_TYPE_IN\",\"values\":[\"prod\"]}]}}}]}}}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49478", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (152, 1737001312, '{"user": "users/101", "method": "/bytebase.v1.IssueService/CreateIssue", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"issue\":{\"name\":\"projects/project-sample/issues/-101\",\"title\":\"Establish \\\"hr_test\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"plan\":\"projects/project-sample/plans/110\"}}", "response": "{\"name\":\"projects/project-sample/issues/110\",\"title\":\"Establish \\\"hr_test\\\" baseline\",\"type\":\"DATABASE_CHANGE\",\"status\":\"OPEN\",\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:21:52Z\",\"updateTime\":\"2025-01-16T04:21:52Z\",\"plan\":\"projects/project-sample/plans/110\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49478", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (154, 1737001369, '{"user": "users/101", "method": "/bytebase.v1.SQLService/AdminExecute", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"statement\":\"ALTER TABLE employee ADD COLUMN bugfix TEXT NOT NULL DEFAULT '''';\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_vcs", "response": "{\"results\":[{\"columnNames\":[\"Affected Rows\"],\"columnTypeNames\":[\"INT\"],\"latency\":\"0.007736708s\",\"statement\":\"ALTER TABLE employee ADD COLUMN bugfix TEXT NOT NULL DEFAULT '''';\"}]}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:64858", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (155, 1737002113, '{"user": "users/101", "method": "/bytebase.v1.SQLService/AdminExecute", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "status": {"code": 13, "message": "failed to receive request: rpc error: code = Canceled desc = context canceled"}, "request": "{\"name\":\"instances/prod-sample-instance/databases/hr_prod_vcs\",\"statement\":\"ALTER TABLE employee ADD COLUMN bugfix TEXT NOT NULL DEFAULT '''';\"}", "resource": "instances/prod-sample-instance/databases/hr_prod_vcs", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:64858", "callerSuppliedUserAgent": "grpc-go/1.69.2"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (153, 1737001312, '{"user": "users/101", "method": "/bytebase.v1.RolloutService/CreateRollout", "parent": "projects/project-sample", "request": "{\"parent\":\"projects/project-sample\",\"rollout\":{\"plan\":\"projects/project-sample/plans/110\"}}", "response": "{\"name\":\"projects/project-sample/rollouts/110\",\"plan\":\"projects/project-sample/plans/110\",\"title\":\"Rollout Pipeline\",\"stages\":[{\"name\":\"projects/project-sample/rollouts/110/stages/113\",\"id\":\"0\",\"title\":\"Test Stage\",\"tasks\":[{\"name\":\"projects/project-sample/rollouts/110/stages/113/tasks/116\",\"title\":\"Establish baseline for database \\\"hr_test\\\"\",\"specId\":\"913aa19f-18e6-42c5-b6e7-2fbb358cffee\",\"status\":\"NOT_STARTED\",\"type\":\"DATABASE_SCHEMA_BASELINE\",\"target\":\"instances/test-sample-instance/databases/hr_test\",\"databaseSchemaBaseline\":{}}]}],\"creator\":\"users/demo@example.com\",\"createTime\":\"2025-01-16T04:21:52Z\",\"updateTime\":\"2025-01-16T04:21:52Z\",\"issue\":\"projects/project-sample/issues/110\"}", "severity": "INFO", "requestMetadata": {"callerIp": "[::1]:49478", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (156, 1737613572, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\":\"instances/bytebase-meta/databases/bb\", \"statement\":\"SELECT\\n  *\\nFROM\\n  \\\"public\\\".\\\"release\\\"\\nLIMIT\\n  50;\", \"limit\":1000, \"dataSourceId\":\"777072ed-539e-4cc2-a41e-6cc2917a7e7c\", \"schema\":\"public\", \"queryOption\":{\"redisRunCommandsOn\":\"SINGLE_NODE\"}, \"container\":\"release\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\":[{\"columnNames\":[\"id\", \"row_status\", \"project_id\", \"creator_id\", \"created_ts\", \"payload\"], \"columnTypeNames\":[\"INT8\", \"16398\", \"INT4\", \"INT4\", \"INT8\", \"JSONB\"], \"latency\":\"0.000670256s\", \"statement\":\"WITH result AS (\\nSELECT\\n  *\\nFROM\\n  \\\"public\\\".\\\"release\\\"\\nLIMIT\\n  50\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "192.168.215.1:34778", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (157, 1737613575, '{"user": "users/101", "method": "/bytebase.v1.SQLService/Query", "parent": "projects/metadb-project", "request": "{\"name\":\"instances/bytebase-meta/databases/bb\", \"statement\":\"SELECT\\n  *\\nFROM\\n  \\\"public\\\".\\\"release\\\"\\nLIMIT\\n  50;\", \"limit\":1000, \"dataSourceId\":\"777072ed-539e-4cc2-a41e-6cc2917a7e7c\", \"schema\":\"public\", \"queryOption\":{\"redisRunCommandsOn\":\"SINGLE_NODE\"}, \"container\":\"release\"}", "resource": "instances/bytebase-meta/databases/bb", "response": "{\"results\":[{\"columnNames\":[\"id\", \"row_status\", \"project_id\", \"creator_id\", \"created_ts\", \"payload\"], \"columnTypeNames\":[\"INT8\", \"16398\", \"INT4\", \"INT4\", \"INT8\", \"JSONB\"], \"latency\":\"0.001700141s\", \"statement\":\"WITH result AS (\\nSELECT\\n  *\\nFROM\\n  \\\"public\\\".\\\"release\\\"\\nLIMIT\\n  50\\n) SELECT * FROM result LIMIT 1000;\"}], \"allowExport\":true}", "severity": "INFO", "requestMetadata": {"callerIp": "192.168.215.1:34778", "callerSuppliedUserAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.audit_log (id, created_ts, payload) VALUES (158, 1738990171, '{"method": "/bytebase.v1.AuthService/Login", "parent": "workspaces/6c86d081-379d-4366-be6f-481425e6f397", "request": "{\"email\":\"demo@example.com\", \"web\":true}", "resource": "demo@example.com", "response": "{\"user\":{\"name\":\"users/101\", \"email\":\"demo@example.com\", \"title\":\"Demo\", \"userType\":\"USER\"}}", "severity": "INFO", "requestMetadata": {"callerIp": "127.0.0.1:35806", "callerSuppliedUserAgent": "grpc-go/1.70.0"}}') ON CONFLICT DO NOTHING;


--
-- Data for Name: branch; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: changelist; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: changelog; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (110, 1, '2023-12-14 05:55:44-08', 109, 'DONE', 101, 102, '{"type": "MIGRATE", "issue": "projects/gitops-project/issues/102", "sheet": "projects/gitops-project/sheets/104", "taskRun": "projects/gitops-project/rollouts/102/stages/103/tasks/103/taskRuns/101", "changedResources": {"databases": [{"name": "hr_prod_vcs", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (112, 101, '2025-01-15 20:19:11.016112-08', 109, 'DONE', 103, 104, '{"type": "BASELINE", "issue": "projects/gitops-project/issues/108", "taskRun": "projects/gitops-project/rollouts/108/stages/111/tasks/114/taskRuns/102"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (113, 101, '2025-01-15 20:19:38.010659-08', 102, 'DONE', 105, 106, '{"type": "BASELINE", "issue": "projects/project-sample/issues/109", "taskRun": "projects/project-sample/rollouts/109/stages/112/tasks/115/taskRuns/103"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (114, 101, '2025-01-15 20:21:56.237441-08', 101, 'DONE', 107, 108, '{"type": "BASELINE", "issue": "projects/project-sample/issues/110", "taskRun": "projects/project-sample/rollouts/110/stages/113/tasks/116/taskRuns/104"}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (115, 106, '2025-01-22 22:25:51.581999-08', 106, 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/104/tasks/105/taskRuns/106", "changedResources": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (116, 106, '2025-01-22 22:25:51.583034-08', 103, 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/104/tasks/104/taskRuns/105", "changedResources": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (117, 106, '2025-01-22 22:25:56.577497-08', 104, 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/105/tasks/106/taskRuns/107", "changedResources": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (118, 106, '2025-01-22 22:25:56.578394-08', 107, 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/105/tasks/107/taskRuns/108", "changedResources": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (119, 106, '2025-01-22 22:26:01.571769-08', 108, 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/106/tasks/109/taskRuns/110", "changedResources": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.changelog (id, creator_id, created_ts, database_id, status, prev_sync_history_id, sync_history_id, payload) VALUES (120, 106, '2025-01-22 22:26:01.580441-08', 105, 'DONE', NULL, NULL, '{"type": "DATA", "issue": "projects/batch-project/issues/103", "sheet": "projects/batch-project/sheets/106", "taskRun": "projects/batch-project/rollouts/103/stages/106/tasks/108/taskRuns/109", "changedResources": {}}') ON CONFLICT DO NOTHING;


--
-- Data for Name: data_source; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (101, 'NORMAL', 101, 1699026391, 101, 1699026391, 101, 'admin', 'ADMIN', 'bbsample', '', '', '', '', '/tmp', '8083', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (102, 'NORMAL', 101, 1699026391, 101, 1699026391, 102, 'admin', 'ADMIN', 'bbsample', '', '', '', '', '/tmp', '8084', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (103, 'NORMAL', 101, 1712757472, 101, 1712757472, 103, '777072ed-539e-4cc2-a41e-6cc2917a7e7c', 'ADMIN', 'bb', '', '', '', '', '/tmp', '8082', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (104, 'NORMAL', 101, 1727534163, 101, 1727534163, 102, '351173a6-f320-45c5-8d95-8e17abe08964', 'RO', 'bbsample', 'WyYTVD4=', '', '', '', '/tmp', '8084', '{"authenticationType": "PASSWORD"}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.data_source (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database) VALUES (105, 'NORMAL', 101, 1727534163, 101, 1727534163, 102, 'e543ddfd-f633-4dbe-87f3-6b171a96e20a', 'RO', 'bbsample', 'WyYTVD4=', '', '', '', '/tmp', '8084', '{"authenticationType": "PASSWORD"}', '') ON CONFLICT DO NOTHING;


--
-- Data for Name: db; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (110, 'NORMAL', 1, 1712757472, 1, 1736962840, 103, 1, NULL, 'OK', 1736962840, '', 'postgres', '{}', false, '', '{"lastSyncTime": "2025-01-15T17:40:40Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (107, 'NORMAL', 1, 1699026391, 1, 1737000968, 102, 103, NULL, 'OK', 1737000968, '', 'hr_prod_5', '{}', false, '', '{"labels": {"location": "eu"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (108, 'NORMAL', 1, 1699026391, 1, 1737000968, 102, 103, NULL, 'OK', 1737000968, '', 'hr_prod_6', '{}', false, '', '{"labels": {"location": "na"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (105, 'NORMAL', 1, 1699026391, 1, 1737000968, 102, 103, NULL, 'OK', 1737000968, '', 'hr_prod_3', '{}', false, '', '{"labels": {"location": "na"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (104, 'NORMAL', 1, 1699026391, 1, 1737000968, 102, 103, NULL, 'OK', 1737000968, '', 'hr_prod_2', '{}', false, '', '{"labels": {"location": "eu"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (103, 'NORMAL', 1, 1699026391, 1, 1737000968, 102, 103, NULL, 'OK', 1737000968, '', 'hr_prod_1', '{}', false, '', '{"labels": {"location": "asia"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (106, 'NORMAL', 1, 1699026391, 1, 1737000968, 102, 103, NULL, 'OK', 1737000968, '', 'hr_prod_4', '{}', false, '', '{"labels": {"location": "asia"}, "lastSyncTime": "2025-01-16T04:16:08Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (102, 'NORMAL', 1, 1699026391, 1, 1737001178, 102, 101, NULL, 'OK', 1737001178, '', 'hr_prod', '{}', false, '', '{"lastSyncTime": "2025-01-16T04:19:38Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (101, 'NORMAL', 1, 1699026391, 1, 1737001316, 101, 101, NULL, 'OK', 1737001316, '', 'hr_test', '{}', false, '', '{"lastSyncTime": "2025-01-16T04:21:56Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (109, 'NORMAL', 1, 1699027042, 1, 1737001377, 102, 102, NULL, 'OK', 1737001377, '0000.0000.0000-1000-ddl', 'hr_prod_vcs', '{}', false, '', '{"lastSyncTime": "2025-01-16T04:22:57Z", "backupAvailable": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.db (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, project_id, environment, sync_status, last_successful_sync_ts, schema_version, name, secrets, datashare, service_name, metadata) VALUES (111, 'NORMAL', 1, 1712757472, 1, 1738999511, 103, 104, NULL, 'OK', 1738999511, '', 'bb', '{}', false, '', '{"lastSyncTime": "2025-02-08T07:25:11Z"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: db_group; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.db_group (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, resource_id, placeholder, expression, payload) VALUES (101, 'NORMAL', 101, 1699027959, 101, 1699027959, 103, 'all-hr-group', 'all-hr-group', '{"expression": "resource.environment_name == \"environments/prod\" && resource.database_name.startsWith(\"hr_prod\")"}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_group (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, resource_id, placeholder, expression, payload) VALUES (102, 'NORMAL', 101, 1720666192, 101, 1720666192, 103, 'all-databases', 'all-databases', '{"expression": "true"}', '{"multitenancy": true}') ON CONFLICT DO NOTHING;


--
-- Data for Name: db_schema; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (120, 'NORMAL', 1, 1712757472, 1, 1736962840, 110, '{"name":"postgres", "schemas":[{"name":"public", "owner":"pg_database_owner"}], "characterSet":"UTF8", "collation":"en_US.UTF-8", "owner":"bb"}', '
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
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (104, 'NORMAL', 1, 1699027042, 1, 1737000968, 104, '{"name":"hr_prod_2","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (102, 'NORMAL', 1, 1699026391, 1, 1737001178, 102, '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

', '{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "department", "columns": [{"name": "dept_name", "classification": "1-1"}]}, {"name": "salary", "columns": [{"name": "amount"}]}, {"name": "employee", "columns": [{"name": "last_name", "semanticType": "be433ce5-72e7-4dcf-8b58-e77b52a18e81", "classification": "1-3"}, {"name": "first_name", "semanticType": "be433ce5-72e7-4dcf-8b58-e77b52a18e81", "classification": "1-3"}]}, {"name": "title", "columns": [{"name": "title", "classification": "2-1"}]}]}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (108, 'NORMAL', 1, 1699027042, 1, 1737000968, 108, '{"name":"hr_prod_6","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (103, 'NORMAL', 1, 1699027042, 1, 1737000968, 103, '{"name":"hr_prod_1","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (121, 'NORMAL', 1, 1712757472, 1, 1738999512, 111, '{"name":"bb","schemas":[{"name":"public","tables":[{"name":"activity","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.activity_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"resource_container","position":7,"defaultExpression":"''''::text","type":"text"},{"name":"container_id","position":8,"type":"integer"},{"name":"type","position":9,"type":"text"},{"name":"level","position":10,"type":"text"},{"name":"comment","position":11,"defaultExpression":"''''::text","type":"text"},{"name":"payload","position":12,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"activity_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX activity_pkey ON public.activity USING btree (id);","isConstraint":true},{"name":"idx_activity_container_id","expressions":["container_id"],"type":"btree","definition":"CREATE INDEX idx_activity_container_id ON public.activity USING btree (container_id);"},{"name":"idx_activity_created_ts","expressions":["created_ts"],"type":"btree","definition":"CREATE INDEX idx_activity_created_ts ON public.activity USING btree (created_ts);"},{"name":"idx_activity_resource_container","expressions":["resource_container"],"type":"btree","definition":"CREATE INDEX idx_activity_resource_container ON public.activity USING btree (resource_container);"}],"rowCount":"92","dataSize":"65536","indexSize":"65536","foreignKeys":[{"name":"activity_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"activity_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"anomaly","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.anomaly_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project","position":7,"type":"text"},{"name":"instance_id","position":8,"type":"integer"},{"name":"database_id","position":9,"nullable":true,"type":"integer"},{"name":"type","position":10,"type":"text"},{"name":"payload","position":11,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"anomaly_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX anomaly_pkey ON public.anomaly USING btree (id);","isConstraint":true},{"name":"idx_anomaly_unique_project_database_id_type","expressions":["project","database_id","type"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_anomaly_unique_project_database_id_type ON public.anomaly USING btree (project, database_id, type);"}],"rowCount":"1","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"anomaly_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"anomaly_database_id_fkey","columns":["database_id"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"anomaly_instance_id_fkey","columns":["instance_id"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"anomaly_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"audit_log","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_log_id_seq''::regclass)","type":"bigint"},{"name":"created_ts","position":2,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"payload","position":3,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"audit_log_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_log_pkey ON public.audit_log USING btree (id);","isConstraint":true},{"name":"idx_audit_log_created_ts","expressions":["created_ts"],"type":"btree","definition":"CREATE INDEX idx_audit_log_created_ts ON public.audit_log USING btree (created_ts);"},{"name":"idx_audit_log_payload_method","expressions":["payload ->> ''method''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_method ON public.audit_log USING btree (((payload ->> ''method''::text)));"},{"name":"idx_audit_log_payload_parent","expressions":["payload ->> ''parent''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_parent ON public.audit_log USING btree (((payload ->> ''parent''::text)));"},{"name":"idx_audit_log_payload_resource","expressions":["payload ->> ''resource''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_resource ON public.audit_log USING btree (((payload ->> ''resource''::text)));"},{"name":"idx_audit_log_payload_user","expressions":["payload ->> ''user''::text"],"type":"btree","definition":"CREATE INDEX idx_audit_log_payload_user ON public.audit_log USING btree (((payload ->> ''user''::text)));"}],"rowCount":"57","dataSize":"90112","indexSize":"98304","owner":"bb"},{"name":"branch","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.branch_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"name","position":8,"type":"text"},{"name":"engine","position":9,"type":"text"},{"name":"base","position":10,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"head","position":11,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"base_schema","position":12,"defaultExpression":"''''::text","type":"text"},{"name":"head_schema","position":13,"defaultExpression":"''''::text","type":"text"},{"name":"reconcile_state","position":14,"defaultExpression":"''''::text","type":"text"},{"name":"config","position":15,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"branch_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX branch_pkey ON public.branch USING btree (id);","isConstraint":true},{"name":"idx_branch_reconcile_state","expressions":["reconcile_state"],"type":"btree","definition":"CREATE INDEX idx_branch_reconcile_state ON public.branch USING btree (reconcile_state);"},{"name":"idx_branch_unique_project_id_name","expressions":["project_id","name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_branch_unique_project_id_name ON public.branch USING btree (project_id, name);"}],"dataSize":"8192","indexSize":"24576","foreignKeys":[{"name":"branch_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"branch_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"branch_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"changelist","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.changelist_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"name","position":8,"type":"text"},{"name":"payload","position":9,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"changelist_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX changelist_pkey ON public.changelist USING btree (id);","isConstraint":true},{"name":"idx_changelist_project_id_name","expressions":["project_id","name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_changelist_project_id_name ON public.changelist USING btree (project_id, name);"}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"changelist_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelist_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelist_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"changelog","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.changelog_id_seq''::regclass)","type":"bigint"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_ts","position":3,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"database_id","position":4,"type":"integer"},{"name":"status","position":5,"type":"text"},{"name":"prev_sync_history_id","position":6,"nullable":true,"type":"bigint"},{"name":"sync_history_id","position":7,"nullable":true,"type":"bigint"},{"name":"payload","position":8,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"changelog_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX changelog_pkey ON public.changelog USING btree (id);","isConstraint":true},{"name":"idx_changelog_database_id","expressions":["database_id"],"type":"btree","definition":"CREATE INDEX idx_changelog_database_id ON public.changelog USING btree (database_id);"}],"rowCount":"10","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"changelog_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelog_database_id_fkey","columns":["database_id"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelog_prev_sync_history_id_fkey","columns":["prev_sync_history_id"],"referencedSchema":"public","referencedTable":"sync_history","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"changelog_sync_history_id_fkey","columns":["sync_history_id"],"referencedSchema":"public","referencedTable":"sync_history","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"data_source","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.data_source_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"instance_id","position":7,"type":"integer"},{"name":"name","position":8,"type":"text"},{"name":"type","position":9,"type":"text"},{"name":"username","position":10,"type":"text"},{"name":"password","position":11,"type":"text"},{"name":"ssl_key","position":12,"defaultExpression":"''''::text","type":"text"},{"name":"ssl_cert","position":13,"defaultExpression":"''''::text","type":"text"},{"name":"ssl_ca","position":14,"defaultExpression":"''''::text","type":"text"},{"name":"host","position":15,"defaultExpression":"''''::text","type":"text"},{"name":"port","position":16,"defaultExpression":"''''::text","type":"text"},{"name":"options","position":17,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"database","position":18,"defaultExpression":"''''::text","type":"text"}],"indexes":[{"name":"data_source_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX data_source_pkey ON public.data_source USING btree (id);","isConstraint":true},{"name":"idx_data_source_unique_instance_id_name","expressions":["instance_id","name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON public.data_source USING btree (instance_id, name);"}],"rowCount":"5","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"data_source_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"data_source_instance_id_fkey","columns":["instance_id"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"data_source_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"db","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.db_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"instance_id","position":7,"type":"integer"},{"name":"project_id","position":8,"type":"integer"},{"name":"environment","position":9,"nullable":true,"type":"text"},{"name":"sync_status","position":10,"type":"text"},{"name":"last_successful_sync_ts","position":11,"type":"bigint"},{"name":"schema_version","position":12,"type":"text"},{"name":"name","position":13,"type":"text"},{"name":"secrets","position":14,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"datashare","position":15,"defaultExpression":"false","type":"boolean"},{"name":"service_name","position":16,"defaultExpression":"''''::text","type":"text"},{"name":"metadata","position":17,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"db_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX db_pkey ON public.db USING btree (id);","isConstraint":true},{"name":"idx_db_instance_id","expressions":["instance_id"],"type":"btree","definition":"CREATE INDEX idx_db_instance_id ON public.db USING btree (instance_id);"},{"name":"idx_db_project_id","expressions":["project_id"],"type":"btree","definition":"CREATE INDEX idx_db_project_id ON public.db USING btree (project_id);"},{"name":"idx_db_unique_instance_id_name","expressions":["instance_id","name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_unique_instance_id_name ON public.db USING btree (instance_id, name);"}],"rowCount":"11","dataSize":"16384","indexSize":"65536","foreignKeys":[{"name":"db_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_environment_fkey","columns":["environment"],"referencedSchema":"public","referencedTable":"environment","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_instance_id_fkey","columns":["instance_id"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"db_group","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.db_group_id_seq''::regclass)","type":"bigint"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"resource_id","position":8,"type":"text"},{"name":"placeholder","position":9,"defaultExpression":"''''::text","type":"text"},{"name":"expression","position":10,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"payload","position":11,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"db_group_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX db_group_pkey ON public.db_group USING btree (id);","isConstraint":true},{"name":"idx_db_group_unique_project_id_placeholder","expressions":["project_id","placeholder"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON public.db_group USING btree (project_id, placeholder);"},{"name":"idx_db_group_unique_project_id_resource_id","expressions":["project_id","resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON public.db_group USING btree (project_id, resource_id);"}],"rowCount":"2","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"db_group_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_group_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_group_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"db_schema","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.db_schema_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"database_id","position":7,"type":"integer"},{"name":"metadata","position":8,"defaultExpression":"''{}''::json","type":"json"},{"name":"raw_dump","position":9,"defaultExpression":"''''::text","type":"text"},{"name":"config","position":10,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"db_schema_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX db_schema_pkey ON public.db_schema USING btree (id);","isConstraint":true},{"name":"idx_db_schema_unique_database_id","expressions":["database_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_db_schema_unique_database_id ON public.db_schema USING btree (database_id);"}],"rowCount":"11","dataSize":"139264","indexSize":"32768","foreignKeys":[{"name":"db_schema_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_schema_database_id_fkey","columns":["database_id"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["id"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"db_schema_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"deployment_config","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.deployment_config_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"name","position":8,"type":"text"},{"name":"config","position":9,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"deployment_config_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX deployment_config_pkey ON public.deployment_config USING btree (id);","isConstraint":true},{"name":"idx_deployment_config_unique_project_id","expressions":["project_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_deployment_config_unique_project_id ON public.deployment_config USING btree (project_id);"}],"rowCount":"1","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"deployment_config_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"deployment_config_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"deployment_config_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"environment","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.environment_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"name","position":7,"type":"text"},{"name":"order","position":8,"type":"integer"},{"name":"resource_id","position":9,"type":"text"}],"indexes":[{"name":"environment_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX environment_pkey ON public.environment USING btree (id);","isConstraint":true},{"name":"idx_environment_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_environment_unique_resource_id ON public.environment USING btree (resource_id);"}],"rowCount":"2","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"environment_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"environment_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"export_archive","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.export_archive_id_seq''::regclass)","type":"integer"},{"name":"created_ts","position":2,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"bytes","position":3,"nullable":true,"type":"bytea"},{"name":"payload","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"export_archive_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX export_archive_pkey ON public.export_archive USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"8192","owner":"bb"},{"name":"external_approval","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.external_approval_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"created_ts","position":3,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updated_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"issue_id","position":5,"type":"integer"},{"name":"requester_id","position":6,"type":"integer"},{"name":"approver_id","position":7,"type":"integer"},{"name":"type","position":8,"type":"text"},{"name":"payload","position":9,"type":"jsonb"}],"indexes":[{"name":"external_approval_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX external_approval_pkey ON public.external_approval USING btree (id);","isConstraint":true},{"name":"idx_external_approval_row_status_issue_id","expressions":["row_status","issue_id"],"type":"btree","definition":"CREATE INDEX idx_external_approval_row_status_issue_id ON public.external_approval USING btree (row_status, issue_id);"}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"external_approval_approver_id_fkey","columns":["approver_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"external_approval_issue_id_fkey","columns":["issue_id"],"referencedSchema":"public","referencedTable":"issue","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"external_approval_requester_id_fkey","columns":["requester_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"idp","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.idp_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"created_ts","position":3,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updated_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"resource_id","position":5,"type":"text"},{"name":"name","position":6,"type":"text"},{"name":"domain","position":7,"type":"text"},{"name":"type","position":8,"type":"text"},{"name":"config","position":9,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idp_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX idp_pkey ON public.idp USING btree (id);","isConstraint":true},{"name":"idx_idp_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_idp_unique_resource_id ON public.idp USING btree (resource_id);"}],"dataSize":"8192","indexSize":"16384","owner":"bb"},{"name":"instance","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.instance_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"environment","position":7,"nullable":true,"type":"text"},{"name":"name","position":8,"type":"text"},{"name":"engine","position":9,"type":"text"},{"name":"engine_version","position":10,"defaultExpression":"''''::text","type":"text"},{"name":"external_link","position":11,"defaultExpression":"''''::text","type":"text"},{"name":"resource_id","position":12,"type":"text"},{"name":"activation","position":13,"defaultExpression":"false","type":"boolean"},{"name":"options","position":14,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"metadata","position":15,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_instance_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_instance_unique_resource_id ON public.instance USING btree (resource_id);"},{"name":"instance_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX instance_pkey ON public.instance USING btree (id);","isConstraint":true}],"rowCount":"3","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"instance_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"instance_environment_fkey","columns":["environment"],"referencedSchema":"public","referencedTable":"environment","referencedColumns":["resource_id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"instance_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"instance_change_history","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.instance_change_history_id_seq''::regclass)","type":"bigint"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"instance_id","position":7,"nullable":true,"type":"integer"},{"name":"database_id","position":8,"nullable":true,"type":"integer"},{"name":"project_id","position":9,"nullable":true,"type":"integer"},{"name":"issue_id","position":10,"nullable":true,"type":"integer"},{"name":"release_version","position":11,"type":"text"},{"name":"sequence","position":12,"type":"bigint"},{"name":"source","position":13,"type":"text"},{"name":"type","position":14,"type":"text"},{"name":"status","position":15,"type":"text"},{"name":"version","position":16,"type":"text"},{"name":"description","position":17,"type":"text"},{"name":"statement","position":18,"type":"text"},{"name":"sheet_id","position":19,"nullable":true,"type":"bigint"},{"name":"schema","position":20,"type":"text"},{"name":"schema_prev","position":21,"type":"text"},{"name":"execution_duration_ns","position":22,"type":"bigint"},{"name":"payload","position":23,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_instance_change_history_unique_instance_id_database_id_sequ","expressions":["instance_id","database_id","sequence"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_sequ ON public.instance_change_history USING btree (instance_id, database_id, sequence);"},{"name":"idx_instance_change_history_unique_instance_id_database_id_vers","expressions":["instance_id","database_id","version"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_vers ON public.instance_change_history USING btree (instance_id, database_id, version);"},{"name":"instance_change_history_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX instance_change_history_pkey ON public.instance_change_history USING btree (id);","isConstraint":true}],"rowCount":"2","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"instance_change_history_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"instance_change_history_database_id_fkey","columns":["database_id"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"instance_change_history_instance_id_fkey","columns":["instance_id"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"instance_change_history_issue_id_fkey","columns":["issue_id"],"referencedSchema":"public","referencedTable":"issue","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"instance_change_history_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"instance_change_history_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"instance_user","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.instance_user_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"instance_id","position":7,"type":"integer"},{"name":"name","position":8,"type":"text"},{"name":"grant","position":9,"type":"text"}],"indexes":[{"name":"idx_instance_user_unique_instance_id_name","expressions":["instance_id","name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_instance_user_unique_instance_id_name ON public.instance_user USING btree (instance_id, name);"},{"name":"instance_user_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX instance_user_pkey ON public.instance_user USING btree (id);","isConstraint":true}],"rowCount":"3","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"instance_user_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"instance_user_instance_id_fkey","columns":["instance_id"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"instance_user_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"issue","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.issue_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"plan_id","position":8,"nullable":true,"type":"bigint"},{"name":"pipeline_id","position":9,"nullable":true,"type":"integer"},{"name":"name","position":10,"type":"text"},{"name":"status","position":11,"type":"text"},{"name":"type","position":12,"type":"text"},{"name":"description","position":13,"defaultExpression":"''''::text","type":"text"},{"name":"assignee_id","position":14,"nullable":true,"type":"integer"},{"name":"assignee_need_attention","position":15,"defaultExpression":"false","type":"boolean"},{"name":"payload","position":16,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"ts_vector","position":17,"nullable":true,"type":"tsvector"}],"indexes":[{"name":"idx_issue_assignee_id","expressions":["assignee_id"],"type":"btree","definition":"CREATE INDEX idx_issue_assignee_id ON public.issue USING btree (assignee_id);"},{"name":"idx_issue_created_ts","expressions":["created_ts"],"type":"btree","definition":"CREATE INDEX idx_issue_created_ts ON public.issue USING btree (created_ts);"},{"name":"idx_issue_creator_id","expressions":["creator_id"],"type":"btree","definition":"CREATE INDEX idx_issue_creator_id ON public.issue USING btree (creator_id);"},{"name":"idx_issue_pipeline_id","expressions":["pipeline_id"],"type":"btree","definition":"CREATE INDEX idx_issue_pipeline_id ON public.issue USING btree (pipeline_id);"},{"name":"idx_issue_plan_id","expressions":["plan_id"],"type":"btree","definition":"CREATE INDEX idx_issue_plan_id ON public.issue USING btree (plan_id);"},{"name":"idx_issue_project_id","expressions":["project_id"],"type":"btree","definition":"CREATE INDEX idx_issue_project_id ON public.issue USING btree (project_id);"},{"name":"idx_issue_ts_vector","expressions":["ts_vector"],"type":"gin","definition":"CREATE INDEX idx_issue_ts_vector ON public.issue USING gin (ts_vector);"},{"name":"issue_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX issue_pkey ON public.issue USING btree (id);","isConstraint":true}],"rowCount":"10","dataSize":"16384","indexSize":"131072","foreignKeys":[{"name":"issue_assignee_id_fkey","columns":["assignee_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_plan_id_fkey","columns":["plan_id"],"referencedSchema":"public","referencedTable":"plan","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"issue_comment","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.issue_comment_id_seq''::regclass)","type":"bigint"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"issue_id","position":7,"type":"integer"},{"name":"payload","position":8,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_issue_comment_issue_id","expressions":["issue_id"],"type":"btree","definition":"CREATE INDEX idx_issue_comment_issue_id ON public.issue_comment USING btree (issue_id);"},{"name":"issue_comment_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX issue_comment_pkey ON public.issue_comment USING btree (id);","isConstraint":true}],"rowCount":"37","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"issue_comment_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_comment_issue_id_fkey","columns":["issue_id"],"referencedSchema":"public","referencedTable":"issue","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_comment_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"issue_subscriber","columns":[{"name":"issue_id","position":1,"type":"integer"},{"name":"subscriber_id","position":2,"type":"integer"}],"indexes":[{"name":"idx_issue_subscriber_subscriber_id","expressions":["subscriber_id"],"type":"btree","definition":"CREATE INDEX idx_issue_subscriber_subscriber_id ON public.issue_subscriber USING btree (subscriber_id);"},{"name":"issue_subscriber_pkey","expressions":["issue_id","subscriber_id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX issue_subscriber_pkey ON public.issue_subscriber USING btree (issue_id, subscriber_id);","isConstraint":true}],"indexSize":"16384","foreignKeys":[{"name":"issue_subscriber_issue_id_fkey","columns":["issue_id"],"referencedSchema":"public","referencedTable":"issue","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"issue_subscriber_subscriber_id_fkey","columns":["subscriber_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"pipeline","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.pipeline_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"name","position":8,"type":"text"}],"indexes":[{"name":"pipeline_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX pipeline_pkey ON public.pipeline USING btree (id);","isConstraint":true}],"rowCount":"10","dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"pipeline_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"pipeline_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"pipeline_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"plan","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.plan_id_seq''::regclass)","type":"bigint"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"pipeline_id","position":8,"nullable":true,"type":"integer"},{"name":"name","position":9,"type":"text"},{"name":"description","position":10,"type":"text"},{"name":"config","position":11,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_plan_pipeline_id","expressions":["pipeline_id"],"type":"btree","definition":"CREATE INDEX idx_plan_pipeline_id ON public.plan USING btree (pipeline_id);"},{"name":"idx_plan_project_id","expressions":["project_id"],"type":"btree","definition":"CREATE INDEX idx_plan_project_id ON public.plan USING btree (project_id);"},{"name":"plan_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX plan_pkey ON public.plan USING btree (id);","isConstraint":true}],"rowCount":"10","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"plan_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"plan_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"plan_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"plan_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"plan_check_run","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.plan_check_run_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_ts","position":3,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":4,"type":"integer"},{"name":"updated_ts","position":5,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"plan_id","position":6,"type":"bigint"},{"name":"status","position":7,"type":"text"},{"name":"type","position":8,"type":"text"},{"name":"config","position":9,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"result","position":10,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"payload","position":11,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_plan_check_run_plan_id","expressions":["plan_id"],"type":"btree","definition":"CREATE INDEX idx_plan_check_run_plan_id ON public.plan_check_run USING btree (plan_id);"},{"name":"plan_check_run_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX plan_check_run_pkey ON public.plan_check_run USING btree (id);","isConstraint":true}],"rowCount":"54","dataSize":"57344","indexSize":"32768","foreignKeys":[{"name":"plan_check_run_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"plan_check_run_plan_id_fkey","columns":["plan_id"],"referencedSchema":"public","referencedTable":"plan","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"plan_check_run_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"policy","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.policy_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"type","position":7,"type":"text"},{"name":"payload","position":8,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"resource_type","position":9,"type":"public.resource_type"},{"name":"resource_id","position":10,"type":"integer"},{"name":"inherit_from_parent","position":11,"defaultExpression":"true","type":"boolean"}],"indexes":[{"name":"idx_policy_unique_resource_type_resource_id_type","expressions":["resource_type","resource_id","type"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_id_type ON public.policy USING btree (resource_type, resource_id, type);"},{"name":"policy_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX policy_pkey ON public.policy USING btree (id);","isConstraint":true}],"rowCount":"16","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"policy_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"policy_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"principal","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.principal_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"type","position":7,"type":"text"},{"name":"name","position":8,"type":"text"},{"name":"email","position":9,"type":"text"},{"name":"password_hash","position":10,"type":"text"},{"name":"phone","position":11,"defaultExpression":"''''::text","type":"text"},{"name":"mfa_config","position":12,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"profile","position":13,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"principal_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX principal_pkey ON public.principal USING btree (id);","isConstraint":true}],"rowCount":"11","dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"principal_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"principal_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"project","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.project_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"name","position":7,"type":"text"},{"name":"key","position":8,"type":"text"},{"name":"resource_id","position":9,"type":"text"},{"name":"data_classification_config_id","position":10,"defaultExpression":"''''::text","type":"text"},{"name":"setting","position":11,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_project_unique_key","expressions":["key"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_project_unique_key ON public.project USING btree (key);"},{"name":"idx_project_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_project_unique_resource_id ON public.project USING btree (resource_id);"},{"name":"project_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX project_pkey ON public.project USING btree (id);","isConstraint":true}],"rowCount":"5","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"project_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"project_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"project_webhook","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.project_webhook_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"type","position":8,"type":"text"},{"name":"name","position":9,"type":"text"},{"name":"url","position":10,"type":"text"},{"name":"activity_list","position":11,"type":"_text"},{"name":"payload","position":12,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_project_webhook_project_id","expressions":["project_id"],"type":"btree","definition":"CREATE INDEX idx_project_webhook_project_id ON public.project_webhook USING btree (project_id);"},{"name":"idx_project_webhook_unique_project_id_url","expressions":["project_id","url"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_project_webhook_unique_project_id_url ON public.project_webhook USING btree (project_id, url);"},{"name":"project_webhook_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX project_webhook_pkey ON public.project_webhook USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"24576","foreignKeys":[{"name":"project_webhook_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"project_webhook_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"project_webhook_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"query_history","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.query_history_id_seq''::regclass)","type":"bigint"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":5,"type":"text"},{"name":"database","position":6,"type":"text"},{"name":"statement","position":7,"type":"text"},{"name":"type","position":8,"type":"text"},{"name":"payload","position":9,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_query_history_creator_id_created_ts_project_id","expressions":["creator_id","created_ts","project_id"],"type":"btree","definition":"CREATE INDEX idx_query_history_creator_id_created_ts_project_id ON public.query_history USING btree (creator_id, created_ts, project_id DESC);"},{"name":"query_history_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX query_history_pkey ON public.query_history USING btree (id);","isConstraint":true}],"rowCount":"30","dataSize":"49152","indexSize":"32768","foreignKeys":[{"name":"query_history_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"release","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.release_id_seq''::regclass)","type":"bigint"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"project_id","position":3,"type":"integer"},{"name":"creator_id","position":4,"type":"integer"},{"name":"created_ts","position":5,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"payload","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_release_project_id","expressions":["project_id"],"type":"btree","definition":"CREATE INDEX idx_release_project_id ON public.release USING btree (project_id);"},{"name":"release_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX release_pkey ON public.release USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"16384","foreignKeys":[{"name":"release_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"release_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"review_config","columns":[{"name":"id","position":1,"type":"text"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"name","position":7,"type":"text"},{"name":"payload","position":8,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"review_config_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX review_config_pkey ON public.review_config USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"32768","indexSize":"16384","foreignKeys":[{"name":"review_config_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"review_config_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"revision","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.revision_id_seq''::regclass)","type":"bigint"},{"name":"database_id","position":2,"type":"integer"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"deleter_id","position":5,"nullable":true,"type":"integer"},{"name":"deleted_ts","position":6,"nullable":true,"type":"timestamp with time zone"},{"name":"version","position":7,"type":"text"},{"name":"payload","position":8,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_revision_database_id_version","expressions":["database_id","version"],"type":"btree","definition":"CREATE INDEX idx_revision_database_id_version ON public.revision USING btree (database_id, version);"},{"name":"idx_revision_unique_database_id_version_deleted_ts_null","expressions":["database_id","version"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_revision_unique_database_id_version_deleted_ts_null ON public.revision USING btree (database_id, version) WHERE (deleted_ts IS NULL);"},{"name":"revision_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX revision_pkey ON public.revision USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"24576","foreignKeys":[{"name":"revision_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"revision_database_id_fkey","columns":["database_id"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"revision_deleter_id_fkey","columns":["deleter_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"risk","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.risk_id_seq''::regclass)","type":"bigint"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"source","position":7,"type":"text"},{"name":"level","position":8,"type":"bigint"},{"name":"name","position":9,"type":"text"},{"name":"active","position":10,"type":"boolean"},{"name":"expression","position":11,"type":"jsonb"}],"indexes":[{"name":"risk_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX risk_pkey ON public.risk USING btree (id);","isConstraint":true}],"rowCount":"2","dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"risk_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"risk_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"role","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.role_id_seq''::regclass)","type":"bigint"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"resource_id","position":7,"type":"text"},{"name":"name","position":8,"type":"text"},{"name":"description","position":9,"type":"text"},{"name":"permissions","position":10,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"payload","position":11,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_role_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_role_unique_resource_id ON public.role USING btree (resource_id);"},{"name":"role_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX role_pkey ON public.role USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"role_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"role_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"setting","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.setting_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"name","position":7,"type":"text"},{"name":"value","position":8,"type":"text"},{"name":"description","position":9,"defaultExpression":"''''::text","type":"text"}],"indexes":[{"name":"idx_setting_unique_name","expressions":["name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_setting_unique_name ON public.setting USING btree (name);"},{"name":"setting_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX setting_pkey ON public.setting USING btree (id);","isConstraint":true}],"rowCount":"16","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"setting_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"setting_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"sheet","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.sheet_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"name","position":8,"type":"text"},{"name":"sha256","position":9,"type":"bytea"},{"name":"payload","position":10,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_sheet_creator_id","expressions":["creator_id"],"type":"btree","definition":"CREATE INDEX idx_sheet_creator_id ON public.sheet USING btree (creator_id);"},{"name":"idx_sheet_name","expressions":["name"],"type":"btree","definition":"CREATE INDEX idx_sheet_name ON public.sheet USING btree (name);"},{"name":"idx_sheet_project_id","expressions":["project_id"],"type":"btree","definition":"CREATE INDEX idx_sheet_project_id ON public.sheet USING btree (project_id);"},{"name":"idx_sheet_project_id_row_status","expressions":["project_id","row_status"],"type":"btree","definition":"CREATE INDEX idx_sheet_project_id_row_status ON public.sheet USING btree (project_id, row_status);"},{"name":"sheet_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sheet_pkey ON public.sheet USING btree (id);","isConstraint":true}],"rowCount":"13","dataSize":"16384","indexSize":"81920","foreignKeys":[{"name":"sheet_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"sheet_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"sheet_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"sheet_blob","columns":[{"name":"sha256","position":1,"type":"bytea"},{"name":"content","position":2,"type":"text"}],"indexes":[{"name":"sheet_blob_pkey","expressions":["sha256"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sheet_blob_pkey ON public.sheet_blob USING btree (sha256);","isConstraint":true}],"rowCount":"9","dataSize":"16384","indexSize":"16384","owner":"bb"},{"name":"slow_query","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.slow_query_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"instance_id","position":7,"type":"integer"},{"name":"database_id","position":8,"nullable":true,"type":"integer"},{"name":"log_date_ts","position":9,"type":"integer"},{"name":"slow_query_statistics","position":10,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_slow_query_instance_id_log_date_ts","expressions":["instance_id","log_date_ts"],"type":"btree","definition":"CREATE INDEX idx_slow_query_instance_id_log_date_ts ON public.slow_query USING btree (instance_id, log_date_ts);"},{"name":"slow_query_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX slow_query_pkey ON public.slow_query USING btree (id);","isConstraint":true},{"name":"uk_slow_query_database_id_log_date_ts","expressions":["database_id","log_date_ts"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX uk_slow_query_database_id_log_date_ts ON public.slow_query USING btree (database_id, log_date_ts);"}],"dataSize":"8192","indexSize":"24576","foreignKeys":[{"name":"slow_query_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"slow_query_database_id_fkey","columns":["database_id"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"slow_query_instance_id_fkey","columns":["instance_id"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"slow_query_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"sql_lint_config","columns":[{"name":"id","position":1,"type":"text"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_ts","position":3,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":4,"type":"integer"},{"name":"updated_ts","position":5,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"config","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"sql_lint_config_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sql_lint_config_pkey ON public.sql_lint_config USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"8192","foreignKeys":[{"name":"sql_lint_config_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"sql_lint_config_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"stage","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.stage_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"pipeline_id","position":7,"type":"integer"},{"name":"environment_id","position":8,"type":"integer"},{"name":"deployment_id","position":9,"defaultExpression":"''''::text","type":"text"},{"name":"name","position":10,"type":"text"}],"indexes":[{"name":"idx_stage_pipeline_id","expressions":["pipeline_id"],"type":"btree","definition":"CREATE INDEX idx_stage_pipeline_id ON public.stage USING btree (pipeline_id);"},{"name":"stage_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX stage_pkey ON public.stage USING btree (id);","isConstraint":true}],"rowCount":"13","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"stage_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"stage_environment_id_fkey","columns":["environment_id"],"referencedSchema":"public","referencedTable":"environment","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"stage_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"stage_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"sync_history","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.sync_history_id_seq''::regclass)","type":"bigint"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_ts","position":3,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"database_id","position":4,"type":"integer"},{"name":"metadata","position":5,"defaultExpression":"''{}''::json","type":"json"},{"name":"raw_dump","position":6,"defaultExpression":"''''::text","type":"text"}],"indexes":[{"name":"idx_sync_history_database_id_created_ts","expressions":["database_id","created_ts"],"type":"btree","definition":"CREATE INDEX idx_sync_history_database_id_created_ts ON public.sync_history USING btree (database_id, created_ts);"},{"name":"sync_history_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX sync_history_pkey ON public.sync_history USING btree (id);","isConstraint":true}],"rowCount":"8","dataSize":"98304","indexSize":"32768","foreignKeys":[{"name":"sync_history_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"sync_history_database_id_fkey","columns":["database_id"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"task","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.task_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"pipeline_id","position":7,"type":"integer"},{"name":"stage_id","position":8,"type":"integer"},{"name":"instance_id","position":9,"type":"integer"},{"name":"database_id","position":10,"nullable":true,"type":"integer"},{"name":"name","position":11,"type":"text"},{"name":"status","position":12,"type":"text"},{"name":"type","position":13,"type":"text"},{"name":"payload","position":14,"defaultExpression":"''{}''::jsonb","type":"jsonb"},{"name":"earliest_allowed_ts","position":15,"defaultExpression":"0","type":"bigint"}],"indexes":[{"name":"idx_task_earliest_allowed_ts","expressions":["earliest_allowed_ts"],"type":"btree","definition":"CREATE INDEX idx_task_earliest_allowed_ts ON public.task USING btree (earliest_allowed_ts);"},{"name":"idx_task_pipeline_id_stage_id","expressions":["pipeline_id","stage_id"],"type":"btree","definition":"CREATE INDEX idx_task_pipeline_id_stage_id ON public.task USING btree (pipeline_id, stage_id);"},{"name":"idx_task_status","expressions":["status"],"type":"btree","definition":"CREATE INDEX idx_task_status ON public.task USING btree (status);"},{"name":"task_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_pkey ON public.task USING btree (id);","isConstraint":true}],"rowCount":"16","dataSize":"16384","indexSize":"65536","foreignKeys":[{"name":"task_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_database_id_fkey","columns":["database_id"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_instance_id_fkey","columns":["instance_id"],"referencedSchema":"public","referencedTable":"instance","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_pipeline_id_fkey","columns":["pipeline_id"],"referencedSchema":"public","referencedTable":"pipeline","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_stage_id_fkey","columns":["stage_id"],"referencedSchema":"public","referencedTable":"stage","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"task_dag","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.task_dag_id_seq''::regclass)","type":"integer"},{"name":"created_ts","position":2,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updated_ts","position":3,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"from_task_id","position":4,"type":"integer"},{"name":"to_task_id","position":5,"type":"integer"},{"name":"payload","position":6,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_task_dag_from_task_id","expressions":["from_task_id"],"type":"btree","definition":"CREATE INDEX idx_task_dag_from_task_id ON public.task_dag USING btree (from_task_id);"},{"name":"idx_task_dag_to_task_id","expressions":["to_task_id"],"type":"btree","definition":"CREATE INDEX idx_task_dag_to_task_id ON public.task_dag USING btree (to_task_id);"},{"name":"task_dag_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_dag_pkey ON public.task_dag USING btree (id);","isConstraint":true}],"dataSize":"8192","indexSize":"24576","foreignKeys":[{"name":"task_dag_from_task_id_fkey","columns":["from_task_id"],"referencedSchema":"public","referencedTable":"task","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_dag_to_task_id_fkey","columns":["to_task_id"],"referencedSchema":"public","referencedTable":"task","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"task_run","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.task_run_id_seq''::regclass)","type":"integer"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_ts","position":3,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":4,"type":"integer"},{"name":"updated_ts","position":5,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"task_id","position":6,"type":"integer"},{"name":"sheet_id","position":7,"nullable":true,"type":"integer"},{"name":"attempt","position":8,"type":"integer"},{"name":"name","position":9,"type":"text"},{"name":"status","position":10,"type":"text"},{"name":"started_ts","position":11,"defaultExpression":"0","type":"bigint"},{"name":"code","position":12,"defaultExpression":"0","type":"integer"},{"name":"result","position":13,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_task_run_task_id","expressions":["task_id"],"type":"btree","definition":"CREATE INDEX idx_task_run_task_id ON public.task_run USING btree (task_id);"},{"name":"task_run_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_run_pkey ON public.task_run USING btree (id);","isConstraint":true},{"name":"uk_task_run_task_id_attempt","expressions":["task_id","attempt"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON public.task_run USING btree (task_id, attempt);"}],"rowCount":"10","dataSize":"16384","indexSize":"49152","foreignKeys":[{"name":"task_run_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_run_sheet_id_fkey","columns":["sheet_id"],"referencedSchema":"public","referencedTable":"sheet","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_run_task_id_fkey","columns":["task_id"],"referencedSchema":"public","referencedTable":"task","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"task_run_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"task_run_log","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.task_run_log_id_seq''::regclass)","type":"bigint"},{"name":"task_run_id","position":2,"type":"integer"},{"name":"created_ts","position":3,"defaultExpression":"now()","type":"timestamp with time zone"},{"name":"payload","position":4,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_task_run_log_task_run_id","expressions":["task_run_id"],"type":"btree","definition":"CREATE INDEX idx_task_run_log_task_run_id ON public.task_run_log USING btree (task_run_id);"},{"name":"task_run_log_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX task_run_log_pkey ON public.task_run_log USING btree (id);","isConstraint":true}],"rowCount":"54","dataSize":"49152","indexSize":"32768","foreignKeys":[{"name":"task_run_log_task_run_id_fkey","columns":["task_run_id"],"referencedSchema":"public","referencedTable":"task_run","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"user_group","columns":[{"name":"email","position":1,"type":"text"},{"name":"creator_id","position":2,"type":"integer"},{"name":"created_ts","position":3,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":4,"type":"integer"},{"name":"updated_ts","position":5,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"name","position":6,"type":"text"},{"name":"description","position":7,"defaultExpression":"''''::text","type":"text"},{"name":"payload","position":8,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"user_group_pkey","expressions":["email"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX user_group_pkey ON public.user_group USING btree (email);","isConstraint":true}],"dataSize":"8192","indexSize":"8192","foreignKeys":[{"name":"user_group_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"user_group_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"vcs","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.vcs_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"resource_id","position":7,"type":"text"},{"name":"name","position":8,"type":"text"},{"name":"type","position":9,"type":"text"},{"name":"instance_url","position":10,"type":"text"},{"name":"access_token","position":11,"defaultExpression":"''''::text","type":"text"}],"indexes":[{"name":"idx_vcs_unique_resource_id","expressions":["resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_vcs_unique_resource_id ON public.vcs USING btree (resource_id);"},{"name":"vcs_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX vcs_pkey ON public.vcs USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"vcs_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"vcs_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"vcs_connector","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.vcs_connector_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"vcs_id","position":7,"type":"integer"},{"name":"project_id","position":8,"type":"integer"},{"name":"resource_id","position":9,"type":"text"},{"name":"payload","position":10,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_vcs_connector_unique_project_id_resource_id","expressions":["project_id","resource_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_vcs_connector_unique_project_id_resource_id ON public.vcs_connector USING btree (project_id, resource_id);"},{"name":"vcs_connector_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX vcs_connector_pkey ON public.vcs_connector USING btree (id);","isConstraint":true}],"rowCount":"1","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"vcs_connector_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"vcs_connector_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"vcs_connector_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"vcs_connector_vcs_id_fkey","columns":["vcs_id"],"referencedSchema":"public","referencedTable":"vcs","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"worksheet","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.worksheet_id_seq''::regclass)","type":"integer"},{"name":"row_status","position":2,"defaultExpression":"''NORMAL''::public.row_status","type":"public.row_status"},{"name":"creator_id","position":3,"type":"integer"},{"name":"created_ts","position":4,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"updater_id","position":5,"type":"integer"},{"name":"updated_ts","position":6,"defaultExpression":"EXTRACT(epoch FROM now())","type":"bigint"},{"name":"project_id","position":7,"type":"integer"},{"name":"database_id","position":8,"nullable":true,"type":"integer"},{"name":"name","position":9,"type":"text"},{"name":"statement","position":10,"type":"text"},{"name":"visibility","position":11,"type":"text"},{"name":"payload","position":12,"defaultExpression":"''{}''::jsonb","type":"jsonb"}],"indexes":[{"name":"idx_worksheet_creator_id_project_id","expressions":["creator_id","project_id"],"type":"btree","definition":"CREATE INDEX idx_worksheet_creator_id_project_id ON public.worksheet USING btree (creator_id, project_id);"},{"name":"worksheet_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX worksheet_pkey ON public.worksheet USING btree (id);","isConstraint":true}],"rowCount":"7","dataSize":"16384","indexSize":"32768","foreignKeys":[{"name":"worksheet_creator_id_fkey","columns":["creator_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"worksheet_database_id_fkey","columns":["database_id"],"referencedSchema":"public","referencedTable":"db","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"worksheet_project_id_fkey","columns":["project_id"],"referencedSchema":"public","referencedTable":"project","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"worksheet_updater_id_fkey","columns":["updater_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"},{"name":"worksheet_organizer","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.worksheet_organizer_id_seq''::regclass)","type":"integer"},{"name":"worksheet_id","position":2,"type":"integer"},{"name":"principal_id","position":3,"type":"integer"},{"name":"starred","position":4,"defaultExpression":"false","type":"boolean"}],"indexes":[{"name":"idx_worksheet_organizer_principal_id","expressions":["principal_id"],"type":"btree","definition":"CREATE INDEX idx_worksheet_organizer_principal_id ON public.worksheet_organizer USING btree (principal_id);"},{"name":"idx_worksheet_organizer_unique_sheet_id_principal_id","expressions":["worksheet_id","principal_id"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal_id ON public.worksheet_organizer USING btree (worksheet_id, principal_id);"},{"name":"worksheet_organizer_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX worksheet_organizer_pkey ON public.worksheet_organizer USING btree (id);","isConstraint":true}],"indexSize":"24576","foreignKeys":[{"name":"worksheet_organizer_principal_id_fkey","columns":["principal_id"],"referencedSchema":"public","referencedTable":"principal","referencedColumns":["id"],"onDelete":"NO ACTION","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"worksheet_organizer_worksheet_id_fkey","columns":["worksheet_id"],"referencedSchema":"public","referencedTable":"worksheet","referencedColumns":["id"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bb"}],"sequences":[{"name":"activity_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"194","ownerTable":"activity","ownerColumn":"id"},{"name":"anomaly_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"107","ownerTable":"anomaly","ownerColumn":"id"},{"name":"audit_log_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"158","ownerTable":"audit_log","ownerColumn":"id"},{"name":"branch_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"105","ownerTable":"branch","ownerColumn":"id"},{"name":"changelist_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"changelist","ownerColumn":"id"},{"name":"changelog_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"120","ownerTable":"changelog","ownerColumn":"id"},{"name":"data_source_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"105","ownerTable":"data_source","ownerColumn":"id"},{"name":"db_group_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"102","ownerTable":"db_group","ownerColumn":"id"},{"name":"db_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"111","ownerTable":"db","ownerColumn":"id"},{"name":"db_schema_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"178","ownerTable":"db_schema","ownerColumn":"id"},{"name":"deployment_config_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"deployment_config","ownerColumn":"id"},{"name":"environment_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"environment","ownerColumn":"id"},{"name":"export_archive_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"export_archive","ownerColumn":"id"},{"name":"external_approval_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"external_approval","ownerColumn":"id"},{"name":"idp_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"idp","ownerColumn":"id"},{"name":"instance_change_history_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"197","ownerTable":"instance_change_history","ownerColumn":"id"},{"name":"instance_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"103","ownerTable":"instance","ownerColumn":"id"},{"name":"instance_user_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"103","ownerTable":"instance_user","ownerColumn":"id"},{"name":"issue_comment_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"141","ownerTable":"issue_comment","ownerColumn":"id"},{"name":"issue_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"issue","ownerColumn":"id"},{"name":"pipeline_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"pipeline","ownerColumn":"id"},{"name":"plan_check_run_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"171","ownerTable":"plan_check_run","ownerColumn":"id"},{"name":"plan_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"plan","ownerColumn":"id"},{"name":"policy_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"169","ownerTable":"policy","ownerColumn":"id"},{"name":"principal_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"109","ownerTable":"principal","ownerColumn":"id"},{"name":"project_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"104","ownerTable":"project","ownerColumn":"id"},{"name":"project_webhook_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"project_webhook","ownerColumn":"id"},{"name":"query_history_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"130","ownerTable":"query_history","ownerColumn":"id"},{"name":"release_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","ownerTable":"release","ownerColumn":"id"},{"name":"revision_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","ownerTable":"revision","ownerColumn":"id"},{"name":"risk_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"102","ownerTable":"risk","ownerColumn":"id"},{"name":"role_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"role","ownerColumn":"id"},{"name":"setting_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"196","ownerTable":"setting","ownerColumn":"id"},{"name":"sheet_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"135","ownerTable":"sheet","ownerColumn":"id"},{"name":"slow_query_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"101","ownerTable":"slow_query","ownerColumn":"id"},{"name":"stage_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"113","ownerTable":"stage","ownerColumn":"id"},{"name":"sync_history_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"108","ownerTable":"sync_history","ownerColumn":"id"},{"name":"task_dag_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"task_dag","ownerColumn":"id"},{"name":"task_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"116","ownerTable":"task","ownerColumn":"id"},{"name":"task_run_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"task_run","ownerColumn":"id"},{"name":"task_run_log_id_seq","dataType":"bigint","start":"1","minValue":"1","maxValue":"9223372036854775807","increment":"1","cacheSize":"1","lastValue":"154","ownerTable":"task_run_log","ownerColumn":"id"},{"name":"vcs_connector_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"vcs_connector","ownerColumn":"id"},{"name":"vcs_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"102","ownerTable":"vcs","ownerColumn":"id"},{"name":"worksheet_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","lastValue":"110","ownerTable":"worksheet","ownerColumn":"id"},{"name":"worksheet_organizer_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"worksheet_organizer","ownerColumn":"id"}],"owner":"pg_database_owner","enumTypes":[{"name":"resource_type","values":["WORKSPACE","ENVIRONMENT","PROJECT","INSTANCE","DATABASE"]},{"name":"row_status","values":["NORMAL","ARCHIVED"]}]}],"characterSet":"UTF8","collation":"en_US.UTF-8","owner":"bb"}', '
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

CREATE TYPE "public"."resource_type" AS ENUM (
    ''WORKSPACE'',
    ''ENVIRONMENT'',
    ''PROJECT'',
    ''INSTANCE'',
    ''DATABASE''
);

CREATE TYPE "public"."row_status" AS ENUM (
    ''NORMAL'',
    ''ARCHIVED''
);

SET default_tablespace = '''';

CREATE SEQUENCE "public"."activity_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."activity" (
    "id" integer DEFAULT nextval(''public.activity_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "resource_container" text DEFAULT ''''::text NOT NULL,
    "container_id" integer NOT NULL,
    "type" text NOT NULL,
    "level" text NOT NULL,
    "comment" text DEFAULT ''''::text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."activity_id_seq" OWNED BY "public"."activity"."id";

ALTER TABLE ONLY "public"."activity" ADD CONSTRAINT "activity_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_activity_container_id" ON ONLY "public"."activity" ("container_id");

CREATE INDEX "idx_activity_created_ts" ON ONLY "public"."activity" ("created_ts");

CREATE INDEX "idx_activity_resource_container" ON ONLY "public"."activity" ("resource_container");

CREATE SEQUENCE "public"."anomaly_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."anomaly" (
    "id" integer DEFAULT nextval(''public.anomaly_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project" text NOT NULL,
    "instance_id" integer NOT NULL,
    "database_id" integer,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."anomaly_id_seq" OWNED BY "public"."anomaly"."id";

ALTER TABLE ONLY "public"."anomaly" ADD CONSTRAINT "anomaly_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_anomaly_unique_project_database_id_type" ON ONLY "public"."anomaly" ("project", "database_id", "type");

CREATE SEQUENCE "public"."audit_log_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."audit_log" (
    "id" bigint DEFAULT nextval(''public.audit_log_id_seq''::regclass) NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."audit_log_id_seq" OWNED BY "public"."audit_log"."id";

ALTER TABLE ONLY "public"."audit_log" ADD CONSTRAINT "audit_log_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_audit_log_created_ts" ON ONLY "public"."audit_log" ("created_ts");

CREATE INDEX "idx_audit_log_payload_method" ON ONLY "public"."audit_log" ((payload ->> ''method''::text));

CREATE INDEX "idx_audit_log_payload_parent" ON ONLY "public"."audit_log" ((payload ->> ''parent''::text));

CREATE INDEX "idx_audit_log_payload_resource" ON ONLY "public"."audit_log" ((payload ->> ''resource''::text));

CREATE INDEX "idx_audit_log_payload_user" ON ONLY "public"."audit_log" ((payload ->> ''user''::text));

CREATE SEQUENCE "public"."branch_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."branch" (
    "id" integer DEFAULT nextval(''public.branch_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
    "name" text NOT NULL,
    "engine" text NOT NULL,
    "base" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "head" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "base_schema" text DEFAULT ''''::text NOT NULL,
    "head_schema" text DEFAULT ''''::text NOT NULL,
    "reconcile_state" text DEFAULT ''''::text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."branch_id_seq" OWNED BY "public"."branch"."id";

ALTER TABLE ONLY "public"."branch" ADD CONSTRAINT "branch_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_branch_reconcile_state" ON ONLY "public"."branch" ("reconcile_state");

CREATE UNIQUE INDEX "idx_branch_unique_project_id_name" ON ONLY "public"."branch" ("project_id", "name");

CREATE SEQUENCE "public"."changelist_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."changelist" (
    "id" integer DEFAULT nextval(''public.changelist_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
    "name" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."changelist_id_seq" OWNED BY "public"."changelist"."id";

ALTER TABLE ONLY "public"."changelist" ADD CONSTRAINT "changelist_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_changelist_project_id_name" ON ONLY "public"."changelist" ("project_id", "name");

CREATE SEQUENCE "public"."changelog_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."changelog" (
    "id" bigint DEFAULT nextval(''public.changelog_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" timestamp with time zone DEFAULT now() NOT NULL,
    "database_id" integer NOT NULL,
    "status" text NOT NULL,
    "prev_sync_history_id" bigint,
    "sync_history_id" bigint,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."changelog_id_seq" OWNED BY "public"."changelog"."id";

ALTER TABLE ONLY "public"."changelog" ADD CONSTRAINT "changelog_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_changelog_database_id" ON ONLY "public"."changelog" ("database_id");

CREATE SEQUENCE "public"."data_source_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."data_source" (
    "id" integer DEFAULT nextval(''public.data_source_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "instance_id" integer NOT NULL,
    "name" text NOT NULL,
    "type" text NOT NULL,
    "username" text NOT NULL,
    "password" text NOT NULL,
    "ssl_key" text DEFAULT ''''::text NOT NULL,
    "ssl_cert" text DEFAULT ''''::text NOT NULL,
    "ssl_ca" text DEFAULT ''''::text NOT NULL,
    "host" text DEFAULT ''''::text NOT NULL,
    "port" text DEFAULT ''''::text NOT NULL,
    "options" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "database" text DEFAULT ''''::text NOT NULL
);

ALTER SEQUENCE "public"."data_source_id_seq" OWNED BY "public"."data_source"."id";

ALTER TABLE ONLY "public"."data_source" ADD CONSTRAINT "data_source_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_data_source_unique_instance_id_name" ON ONLY "public"."data_source" ("instance_id", "name");

CREATE SEQUENCE "public"."db_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."db" (
    "id" integer DEFAULT nextval(''public.db_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "instance_id" integer NOT NULL,
    "project_id" integer NOT NULL,
    "environment" text,
    "sync_status" text NOT NULL,
    "last_successful_sync_ts" bigint NOT NULL,
    "schema_version" text NOT NULL,
    "name" text NOT NULL,
    "secrets" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "datashare" boolean DEFAULT false NOT NULL,
    "service_name" text DEFAULT ''''::text NOT NULL,
    "metadata" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."db_id_seq" OWNED BY "public"."db"."id";

ALTER TABLE ONLY "public"."db" ADD CONSTRAINT "db_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_db_instance_id" ON ONLY "public"."db" ("instance_id");

CREATE INDEX "idx_db_project_id" ON ONLY "public"."db" ("project_id");

CREATE UNIQUE INDEX "idx_db_unique_instance_id_name" ON ONLY "public"."db" ("instance_id", "name");

CREATE SEQUENCE "public"."db_group_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."db_group" (
    "id" bigint DEFAULT nextval(''public.db_group_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
    "resource_id" text NOT NULL,
    "placeholder" text DEFAULT ''''::text NOT NULL,
    "expression" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."db_group_id_seq" OWNED BY "public"."db_group"."id";

ALTER TABLE ONLY "public"."db_group" ADD CONSTRAINT "db_group_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_db_group_unique_project_id_placeholder" ON ONLY "public"."db_group" ("project_id", "placeholder");

CREATE UNIQUE INDEX "idx_db_group_unique_project_id_resource_id" ON ONLY "public"."db_group" ("project_id", "resource_id");

CREATE SEQUENCE "public"."db_schema_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."db_schema" (
    "id" integer DEFAULT nextval(''public.db_schema_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "database_id" integer NOT NULL,
    "metadata" json DEFAULT ''{}''::json NOT NULL,
    "raw_dump" text DEFAULT ''''::text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."db_schema_id_seq" OWNED BY "public"."db_schema"."id";

ALTER TABLE ONLY "public"."db_schema" ADD CONSTRAINT "db_schema_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_db_schema_unique_database_id" ON ONLY "public"."db_schema" ("database_id");

CREATE SEQUENCE "public"."deployment_config_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."deployment_config" (
    "id" integer DEFAULT nextval(''public.deployment_config_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
    "name" text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."deployment_config_id_seq" OWNED BY "public"."deployment_config"."id";

ALTER TABLE ONLY "public"."deployment_config" ADD CONSTRAINT "deployment_config_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_deployment_config_unique_project_id" ON ONLY "public"."deployment_config" ("project_id");

CREATE SEQUENCE "public"."environment_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."environment" (
    "id" integer DEFAULT nextval(''public.environment_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "name" text NOT NULL,
    "order" integer NOT NULL,
    "resource_id" text NOT NULL
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
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "bytes" bytea,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."export_archive_id_seq" OWNED BY "public"."export_archive"."id";

ALTER TABLE ONLY "public"."export_archive" ADD CONSTRAINT "export_archive_pkey" PRIMARY KEY ("id");

CREATE SEQUENCE "public"."external_approval_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."external_approval" (
    "id" integer DEFAULT nextval(''public.external_approval_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "issue_id" integer NOT NULL,
    "requester_id" integer NOT NULL,
    "approver_id" integer NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb NOT NULL
);

ALTER SEQUENCE "public"."external_approval_id_seq" OWNED BY "public"."external_approval"."id";

ALTER TABLE ONLY "public"."external_approval" ADD CONSTRAINT "external_approval_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_external_approval_row_status_issue_id" ON ONLY "public"."external_approval" ("row_status", "issue_id");

CREATE SEQUENCE "public"."idp_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."idp" (
    "id" integer DEFAULT nextval(''public.idp_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "resource_id" text NOT NULL,
    "name" text NOT NULL,
    "domain" text NOT NULL,
    "type" text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "environment" text,
    "name" text NOT NULL,
    "engine" text NOT NULL,
    "engine_version" text DEFAULT ''''::text NOT NULL,
    "external_link" text DEFAULT ''''::text NOT NULL,
    "resource_id" text NOT NULL,
    "activation" boolean DEFAULT false NOT NULL,
    "options" jsonb DEFAULT ''{}''::jsonb NOT NULL,
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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "instance_id" integer,
    "database_id" integer,
    "project_id" integer,
    "issue_id" integer,
    "release_version" text NOT NULL,
    "sequence" bigint NOT NULL,
    "source" text NOT NULL,
    "type" text NOT NULL,
    "status" text NOT NULL,
    "version" text NOT NULL,
    "description" text NOT NULL,
    "statement" text NOT NULL,
    "sheet_id" bigint,
    "schema" text NOT NULL,
    "schema_prev" text NOT NULL,
    "execution_duration_ns" bigint NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."instance_change_history_id_seq" OWNED BY "public"."instance_change_history"."id";

ALTER TABLE ONLY "public"."instance_change_history" ADD CONSTRAINT "instance_change_history_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_instance_change_history_unique_instance_id_database_id_sequ" ON ONLY "public"."instance_change_history" ("instance_id", "database_id", "sequence");

CREATE UNIQUE INDEX "idx_instance_change_history_unique_instance_id_database_id_vers" ON ONLY "public"."instance_change_history" ("instance_id", "database_id", "version");

CREATE SEQUENCE "public"."instance_user_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."instance_user" (
    "id" integer DEFAULT nextval(''public.instance_user_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "instance_id" integer NOT NULL,
    "name" text NOT NULL,
    "grant" text NOT NULL
);

ALTER SEQUENCE "public"."instance_user_id_seq" OWNED BY "public"."instance_user"."id";

ALTER TABLE ONLY "public"."instance_user" ADD CONSTRAINT "instance_user_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_instance_user_unique_instance_id_name" ON ONLY "public"."instance_user" ("instance_id", "name");

CREATE SEQUENCE "public"."issue_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."issue" (
    "id" integer DEFAULT nextval(''public.issue_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
    "plan_id" bigint,
    "pipeline_id" integer,
    "name" text NOT NULL,
    "status" text NOT NULL,
    "type" text NOT NULL,
    "description" text DEFAULT ''''::text NOT NULL,
    "assignee_id" integer,
    "assignee_need_attention" boolean DEFAULT false NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "ts_vector" tsvector
);

ALTER SEQUENCE "public"."issue_id_seq" OWNED BY "public"."issue"."id";

ALTER TABLE ONLY "public"."issue" ADD CONSTRAINT "issue_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_issue_assignee_id" ON ONLY "public"."issue" ("assignee_id");

CREATE INDEX "idx_issue_created_ts" ON ONLY "public"."issue" ("created_ts");

CREATE INDEX "idx_issue_creator_id" ON ONLY "public"."issue" ("creator_id");

CREATE INDEX "idx_issue_pipeline_id" ON ONLY "public"."issue" ("pipeline_id");

CREATE INDEX "idx_issue_plan_id" ON ONLY "public"."issue" ("plan_id");

CREATE INDEX "idx_issue_project_id" ON ONLY "public"."issue" ("project_id");

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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
    "pipeline_id" integer,
    "name" text NOT NULL,
    "description" text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."plan_id_seq" OWNED BY "public"."plan"."id";

ALTER TABLE ONLY "public"."plan" ADD CONSTRAINT "plan_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_plan_pipeline_id" ON ONLY "public"."plan" ("pipeline_id");

CREATE INDEX "idx_plan_project_id" ON ONLY "public"."plan" ("project_id");

CREATE SEQUENCE "public"."plan_check_run_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."plan_check_run" (
    "id" integer DEFAULT nextval(''public.plan_check_run_id_seq''::regclass) NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "plan_id" bigint NOT NULL,
    "status" text NOT NULL,
    "type" text NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "result" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "resource_type" public.resource_type NOT NULL,
    "resource_id" integer NOT NULL,
    "inherit_from_parent" boolean DEFAULT true NOT NULL
);

ALTER SEQUENCE "public"."policy_id_seq" OWNED BY "public"."policy"."id";

ALTER TABLE ONLY "public"."policy" ADD CONSTRAINT "policy_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_policy_unique_resource_type_resource_id_type" ON ONLY "public"."policy" ("resource_type", "resource_id", "type");

CREATE SEQUENCE "public"."principal_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."principal" (
    "id" integer DEFAULT nextval(''public.principal_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "type" text NOT NULL,
    "name" text NOT NULL,
    "email" text NOT NULL,
    "password_hash" text NOT NULL,
    "phone" text DEFAULT ''''::text NOT NULL,
    "mfa_config" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "profile" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "name" text NOT NULL,
    "key" text NOT NULL,
    "resource_id" text NOT NULL,
    "data_classification_config_id" text DEFAULT ''''::text NOT NULL,
    "setting" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."project_id_seq" OWNED BY "public"."project"."id";

ALTER TABLE ONLY "public"."project" ADD CONSTRAINT "project_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_project_unique_key" ON ONLY "public"."project" ("key");

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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
    "type" text NOT NULL,
    "name" text NOT NULL,
    "url" text NOT NULL,
    "activity_list" _text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."project_webhook_id_seq" OWNED BY "public"."project_webhook"."id";

ALTER TABLE ONLY "public"."project_webhook" ADD CONSTRAINT "project_webhook_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_project_webhook_project_id" ON ONLY "public"."project_webhook" ("project_id");

CREATE UNIQUE INDEX "idx_project_webhook_unique_project_id_url" ON ONLY "public"."project_webhook" ("project_id", "url");

CREATE SEQUENCE "public"."query_history_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."query_history" (
    "id" bigint DEFAULT nextval(''public.query_history_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" text NOT NULL,
    "database" text NOT NULL,
    "statement" text NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."query_history_id_seq" OWNED BY "public"."query_history"."id";

ALTER TABLE ONLY "public"."query_history" ADD CONSTRAINT "query_history_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_query_history_creator_id_created_ts_project_id" ON ONLY "public"."query_history" ("creator_id", "created_ts", "project_id" DESC);

CREATE SEQUENCE "public"."release_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."release" (
    "id" bigint DEFAULT nextval(''public.release_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "project_id" integer NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" timestamp with time zone DEFAULT now() NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."release_id_seq" OWNED BY "public"."release"."id";

ALTER TABLE ONLY "public"."release" ADD CONSTRAINT "release_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_release_project_id" ON ONLY "public"."release" ("project_id");

CREATE TABLE "public"."review_config" (
    "id" text NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    "database_id" integer NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" timestamp with time zone DEFAULT now() NOT NULL,
    "deleter_id" integer,
    "deleted_ts" timestamp with time zone,
    "version" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."revision_id_seq" OWNED BY "public"."revision"."id";

ALTER TABLE ONLY "public"."revision" ADD CONSTRAINT "revision_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_revision_database_id_version" ON ONLY "public"."revision" ("database_id", "version");

CREATE UNIQUE INDEX "idx_revision_unique_database_id_version_deleted_ts_null" ON ONLY "public"."revision" ("database_id", "version");

CREATE SEQUENCE "public"."risk_id_seq"
    AS bigint
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	NO CYCLE;

CREATE TABLE "public"."risk" (
    "id" bigint DEFAULT nextval(''public.risk_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "name" text NOT NULL,
    "value" text NOT NULL,
    "description" text DEFAULT ''''::text NOT NULL
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
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
    "name" text NOT NULL,
    "sha256" bytea NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."sheet_id_seq" OWNED BY "public"."sheet"."id";

ALTER TABLE ONLY "public"."sheet" ADD CONSTRAINT "sheet_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_sheet_creator_id" ON ONLY "public"."sheet" ("creator_id");

CREATE INDEX "idx_sheet_name" ON ONLY "public"."sheet" ("name");

CREATE INDEX "idx_sheet_project_id" ON ONLY "public"."sheet" ("project_id");

CREATE INDEX "idx_sheet_project_id_row_status" ON ONLY "public"."sheet" ("project_id", "row_status");

CREATE TABLE "public"."sheet_blob" (
    "sha256" bytea NOT NULL,
    "content" text NOT NULL
);

ALTER TABLE ONLY "public"."sheet_blob" ADD CONSTRAINT "sheet_blob_pkey" PRIMARY KEY ("sha256");

CREATE SEQUENCE "public"."slow_query_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."slow_query" (
    "id" integer DEFAULT nextval(''public.slow_query_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "instance_id" integer NOT NULL,
    "database_id" integer,
    "log_date_ts" integer NOT NULL,
    "slow_query_statistics" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."slow_query_id_seq" OWNED BY "public"."slow_query"."id";

ALTER TABLE ONLY "public"."slow_query" ADD CONSTRAINT "slow_query_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_slow_query_instance_id_log_date_ts" ON ONLY "public"."slow_query" ("instance_id", "log_date_ts");

CREATE UNIQUE INDEX "uk_slow_query_database_id_log_date_ts" ON ONLY "public"."slow_query" ("database_id", "log_date_ts");

CREATE TABLE "public"."sql_lint_config" (
    "id" text NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "config" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER TABLE ONLY "public"."sql_lint_config" ADD CONSTRAINT "sql_lint_config_pkey" PRIMARY KEY ("id");

CREATE SEQUENCE "public"."stage_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."stage" (
    "id" integer DEFAULT nextval(''public.stage_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "pipeline_id" integer NOT NULL,
    "environment_id" integer NOT NULL,
    "deployment_id" text DEFAULT ''''::text NOT NULL,
    "name" text NOT NULL
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
    "creator_id" integer NOT NULL,
    "created_ts" timestamp with time zone DEFAULT now() NOT NULL,
    "database_id" integer NOT NULL,
    "metadata" json DEFAULT ''{}''::json NOT NULL,
    "raw_dump" text DEFAULT ''''::text NOT NULL
);

ALTER SEQUENCE "public"."sync_history_id_seq" OWNED BY "public"."sync_history"."id";

ALTER TABLE ONLY "public"."sync_history" ADD CONSTRAINT "sync_history_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_sync_history_database_id_created_ts" ON ONLY "public"."sync_history" ("database_id", "created_ts");

CREATE SEQUENCE "public"."task_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."task" (
    "id" integer DEFAULT nextval(''public.task_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "pipeline_id" integer NOT NULL,
    "stage_id" integer NOT NULL,
    "instance_id" integer NOT NULL,
    "database_id" integer,
    "name" text NOT NULL,
    "status" text NOT NULL,
    "type" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL,
    "earliest_allowed_ts" bigint DEFAULT 0 NOT NULL
);

ALTER SEQUENCE "public"."task_id_seq" OWNED BY "public"."task"."id";

ALTER TABLE ONLY "public"."task" ADD CONSTRAINT "task_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_task_earliest_allowed_ts" ON ONLY "public"."task" ("earliest_allowed_ts");

CREATE INDEX "idx_task_pipeline_id_stage_id" ON ONLY "public"."task" ("pipeline_id", "stage_id");

CREATE INDEX "idx_task_status" ON ONLY "public"."task" ("status");

CREATE SEQUENCE "public"."task_dag_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."task_dag" (
    "id" integer DEFAULT nextval(''public.task_dag_id_seq''::regclass) NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "from_task_id" integer NOT NULL,
    "to_task_id" integer NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."task_dag_id_seq" OWNED BY "public"."task_dag"."id";

ALTER TABLE ONLY "public"."task_dag" ADD CONSTRAINT "task_dag_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_task_dag_from_task_id" ON ONLY "public"."task_dag" ("from_task_id");

CREATE INDEX "idx_task_dag_to_task_id" ON ONLY "public"."task_dag" ("to_task_id");

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
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "task_id" integer NOT NULL,
    "sheet_id" integer,
    "attempt" integer NOT NULL,
    "name" text NOT NULL,
    "status" text NOT NULL,
    "started_ts" bigint DEFAULT 0 NOT NULL,
    "code" integer DEFAULT 0 NOT NULL,
    "result" jsonb DEFAULT ''{}''::jsonb NOT NULL
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
    "created_ts" timestamp with time zone DEFAULT now() NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."task_run_log_id_seq" OWNED BY "public"."task_run_log"."id";

ALTER TABLE ONLY "public"."task_run_log" ADD CONSTRAINT "task_run_log_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_task_run_log_task_run_id" ON ONLY "public"."task_run_log" ("task_run_id");

CREATE TABLE "public"."user_group" (
    "email" text NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "name" text NOT NULL,
    "description" text DEFAULT ''''::text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER TABLE ONLY "public"."user_group" ADD CONSTRAINT "user_group_pkey" PRIMARY KEY ("email");

CREATE SEQUENCE "public"."vcs_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."vcs" (
    "id" integer DEFAULT nextval(''public.vcs_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "resource_id" text NOT NULL,
    "name" text NOT NULL,
    "type" text NOT NULL,
    "instance_url" text NOT NULL,
    "access_token" text DEFAULT ''''::text NOT NULL
);

ALTER SEQUENCE "public"."vcs_id_seq" OWNED BY "public"."vcs"."id";

ALTER TABLE ONLY "public"."vcs" ADD CONSTRAINT "vcs_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_vcs_unique_resource_id" ON ONLY "public"."vcs" ("resource_id");

CREATE SEQUENCE "public"."vcs_connector_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."vcs_connector" (
    "id" integer DEFAULT nextval(''public.vcs_connector_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "vcs_id" integer NOT NULL,
    "project_id" integer NOT NULL,
    "resource_id" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."vcs_connector_id_seq" OWNED BY "public"."vcs_connector"."id";

ALTER TABLE ONLY "public"."vcs_connector" ADD CONSTRAINT "vcs_connector_pkey" PRIMARY KEY ("id");

CREATE UNIQUE INDEX "idx_vcs_connector_unique_project_id_resource_id" ON ONLY "public"."vcs_connector" ("project_id", "resource_id");

CREATE SEQUENCE "public"."worksheet_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."worksheet" (
    "id" integer DEFAULT nextval(''public.worksheet_id_seq''::regclass) NOT NULL,
    "row_status" public.row_status DEFAULT ''NORMAL''::public.row_status NOT NULL,
    "creator_id" integer NOT NULL,
    "created_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "updater_id" integer NOT NULL,
    "updated_ts" bigint DEFAULT EXTRACT(epoch FROM now()) NOT NULL,
    "project_id" integer NOT NULL,
    "database_id" integer,
    "name" text NOT NULL,
    "statement" text NOT NULL,
    "visibility" text NOT NULL,
    "payload" jsonb DEFAULT ''{}''::jsonb NOT NULL
);

ALTER SEQUENCE "public"."worksheet_id_seq" OWNED BY "public"."worksheet"."id";

ALTER TABLE ONLY "public"."worksheet" ADD CONSTRAINT "worksheet_pkey" PRIMARY KEY ("id");

CREATE INDEX "idx_worksheet_creator_id_project_id" ON ONLY "public"."worksheet" ("creator_id", "project_id");

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

ALTER TABLE "public"."activity"
    ADD CONSTRAINT "activity_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."activity"
    ADD CONSTRAINT "activity_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."anomaly"
    ADD CONSTRAINT "anomaly_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."anomaly"
    ADD CONSTRAINT "anomaly_database_id_fkey" FOREIGN KEY ("database_id")
    REFERENCES "public"."db" ("id");

ALTER TABLE "public"."anomaly"
    ADD CONSTRAINT "anomaly_instance_id_fkey" FOREIGN KEY ("instance_id")
    REFERENCES "public"."instance" ("id");

ALTER TABLE "public"."anomaly"
    ADD CONSTRAINT "anomaly_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."branch"
    ADD CONSTRAINT "branch_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."branch"
    ADD CONSTRAINT "branch_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."branch"
    ADD CONSTRAINT "branch_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."changelist"
    ADD CONSTRAINT "changelist_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."changelist"
    ADD CONSTRAINT "changelist_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."changelist"
    ADD CONSTRAINT "changelist_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."changelog"
    ADD CONSTRAINT "changelog_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."changelog"
    ADD CONSTRAINT "changelog_database_id_fkey" FOREIGN KEY ("database_id")
    REFERENCES "public"."db" ("id");

ALTER TABLE "public"."changelog"
    ADD CONSTRAINT "changelog_prev_sync_history_id_fkey" FOREIGN KEY ("prev_sync_history_id")
    REFERENCES "public"."sync_history" ("id");

ALTER TABLE "public"."changelog"
    ADD CONSTRAINT "changelog_sync_history_id_fkey" FOREIGN KEY ("sync_history_id")
    REFERENCES "public"."sync_history" ("id");

ALTER TABLE "public"."data_source"
    ADD CONSTRAINT "data_source_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."data_source"
    ADD CONSTRAINT "data_source_instance_id_fkey" FOREIGN KEY ("instance_id")
    REFERENCES "public"."instance" ("id");

ALTER TABLE "public"."data_source"
    ADD CONSTRAINT "data_source_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."db"
    ADD CONSTRAINT "db_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."db"
    ADD CONSTRAINT "db_environment_fkey" FOREIGN KEY ("environment")
    REFERENCES "public"."environment" ("resource_id");

ALTER TABLE "public"."db"
    ADD CONSTRAINT "db_instance_id_fkey" FOREIGN KEY ("instance_id")
    REFERENCES "public"."instance" ("id");

ALTER TABLE "public"."db"
    ADD CONSTRAINT "db_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."db"
    ADD CONSTRAINT "db_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."db_group"
    ADD CONSTRAINT "db_group_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."db_group"
    ADD CONSTRAINT "db_group_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."db_group"
    ADD CONSTRAINT "db_group_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."db_schema"
    ADD CONSTRAINT "db_schema_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."db_schema"
    ADD CONSTRAINT "db_schema_database_id_fkey" FOREIGN KEY ("database_id")
    REFERENCES "public"."db" ("id");

ALTER TABLE "public"."db_schema"
    ADD CONSTRAINT "db_schema_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."deployment_config"
    ADD CONSTRAINT "deployment_config_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."deployment_config"
    ADD CONSTRAINT "deployment_config_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."deployment_config"
    ADD CONSTRAINT "deployment_config_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."environment"
    ADD CONSTRAINT "environment_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."environment"
    ADD CONSTRAINT "environment_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."external_approval"
    ADD CONSTRAINT "external_approval_approver_id_fkey" FOREIGN KEY ("approver_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."external_approval"
    ADD CONSTRAINT "external_approval_issue_id_fkey" FOREIGN KEY ("issue_id")
    REFERENCES "public"."issue" ("id");

ALTER TABLE "public"."external_approval"
    ADD CONSTRAINT "external_approval_requester_id_fkey" FOREIGN KEY ("requester_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."instance"
    ADD CONSTRAINT "instance_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."instance"
    ADD CONSTRAINT "instance_environment_fkey" FOREIGN KEY ("environment")
    REFERENCES "public"."environment" ("resource_id");

ALTER TABLE "public"."instance"
    ADD CONSTRAINT "instance_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."instance_change_history"
    ADD CONSTRAINT "instance_change_history_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."instance_change_history"
    ADD CONSTRAINT "instance_change_history_database_id_fkey" FOREIGN KEY ("database_id")
    REFERENCES "public"."db" ("id");

ALTER TABLE "public"."instance_change_history"
    ADD CONSTRAINT "instance_change_history_instance_id_fkey" FOREIGN KEY ("instance_id")
    REFERENCES "public"."instance" ("id");

ALTER TABLE "public"."instance_change_history"
    ADD CONSTRAINT "instance_change_history_issue_id_fkey" FOREIGN KEY ("issue_id")
    REFERENCES "public"."issue" ("id");

ALTER TABLE "public"."instance_change_history"
    ADD CONSTRAINT "instance_change_history_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."instance_change_history"
    ADD CONSTRAINT "instance_change_history_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."instance_user"
    ADD CONSTRAINT "instance_user_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."instance_user"
    ADD CONSTRAINT "instance_user_instance_id_fkey" FOREIGN KEY ("instance_id")
    REFERENCES "public"."instance" ("id");

ALTER TABLE "public"."instance_user"
    ADD CONSTRAINT "instance_user_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."issue"
    ADD CONSTRAINT "issue_assignee_id_fkey" FOREIGN KEY ("assignee_id")
    REFERENCES "public"."principal" ("id");

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
    ADD CONSTRAINT "issue_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."issue"
    ADD CONSTRAINT "issue_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."issue_comment"
    ADD CONSTRAINT "issue_comment_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."issue_comment"
    ADD CONSTRAINT "issue_comment_issue_id_fkey" FOREIGN KEY ("issue_id")
    REFERENCES "public"."issue" ("id");

ALTER TABLE "public"."issue_comment"
    ADD CONSTRAINT "issue_comment_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

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
    ADD CONSTRAINT "pipeline_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."pipeline"
    ADD CONSTRAINT "pipeline_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."plan"
    ADD CONSTRAINT "plan_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."plan"
    ADD CONSTRAINT "plan_pipeline_id_fkey" FOREIGN KEY ("pipeline_id")
    REFERENCES "public"."pipeline" ("id");

ALTER TABLE "public"."plan"
    ADD CONSTRAINT "plan_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."plan"
    ADD CONSTRAINT "plan_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."plan_check_run"
    ADD CONSTRAINT "plan_check_run_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."plan_check_run"
    ADD CONSTRAINT "plan_check_run_plan_id_fkey" FOREIGN KEY ("plan_id")
    REFERENCES "public"."plan" ("id");

ALTER TABLE "public"."plan_check_run"
    ADD CONSTRAINT "plan_check_run_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."policy"
    ADD CONSTRAINT "policy_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."policy"
    ADD CONSTRAINT "policy_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."principal"
    ADD CONSTRAINT "principal_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."principal"
    ADD CONSTRAINT "principal_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."project"
    ADD CONSTRAINT "project_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."project"
    ADD CONSTRAINT "project_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."project_webhook"
    ADD CONSTRAINT "project_webhook_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."project_webhook"
    ADD CONSTRAINT "project_webhook_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."project_webhook"
    ADD CONSTRAINT "project_webhook_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."query_history"
    ADD CONSTRAINT "query_history_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."release"
    ADD CONSTRAINT "release_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."release"
    ADD CONSTRAINT "release_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."review_config"
    ADD CONSTRAINT "review_config_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."review_config"
    ADD CONSTRAINT "review_config_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."revision"
    ADD CONSTRAINT "revision_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."revision"
    ADD CONSTRAINT "revision_database_id_fkey" FOREIGN KEY ("database_id")
    REFERENCES "public"."db" ("id");

ALTER TABLE "public"."revision"
    ADD CONSTRAINT "revision_deleter_id_fkey" FOREIGN KEY ("deleter_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."risk"
    ADD CONSTRAINT "risk_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."risk"
    ADD CONSTRAINT "risk_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."role"
    ADD CONSTRAINT "role_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."role"
    ADD CONSTRAINT "role_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."setting"
    ADD CONSTRAINT "setting_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."setting"
    ADD CONSTRAINT "setting_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."sheet"
    ADD CONSTRAINT "sheet_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."sheet"
    ADD CONSTRAINT "sheet_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."sheet"
    ADD CONSTRAINT "sheet_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."slow_query"
    ADD CONSTRAINT "slow_query_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."slow_query"
    ADD CONSTRAINT "slow_query_database_id_fkey" FOREIGN KEY ("database_id")
    REFERENCES "public"."db" ("id");

ALTER TABLE "public"."slow_query"
    ADD CONSTRAINT "slow_query_instance_id_fkey" FOREIGN KEY ("instance_id")
    REFERENCES "public"."instance" ("id");

ALTER TABLE "public"."slow_query"
    ADD CONSTRAINT "slow_query_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."sql_lint_config"
    ADD CONSTRAINT "sql_lint_config_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."sql_lint_config"
    ADD CONSTRAINT "sql_lint_config_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."stage"
    ADD CONSTRAINT "stage_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."stage"
    ADD CONSTRAINT "stage_environment_id_fkey" FOREIGN KEY ("environment_id")
    REFERENCES "public"."environment" ("id");

ALTER TABLE "public"."stage"
    ADD CONSTRAINT "stage_pipeline_id_fkey" FOREIGN KEY ("pipeline_id")
    REFERENCES "public"."pipeline" ("id");

ALTER TABLE "public"."stage"
    ADD CONSTRAINT "stage_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."sync_history"
    ADD CONSTRAINT "sync_history_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."sync_history"
    ADD CONSTRAINT "sync_history_database_id_fkey" FOREIGN KEY ("database_id")
    REFERENCES "public"."db" ("id");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_database_id_fkey" FOREIGN KEY ("database_id")
    REFERENCES "public"."db" ("id");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_instance_id_fkey" FOREIGN KEY ("instance_id")
    REFERENCES "public"."instance" ("id");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_pipeline_id_fkey" FOREIGN KEY ("pipeline_id")
    REFERENCES "public"."pipeline" ("id");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_stage_id_fkey" FOREIGN KEY ("stage_id")
    REFERENCES "public"."stage" ("id");

ALTER TABLE "public"."task"
    ADD CONSTRAINT "task_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."task_dag"
    ADD CONSTRAINT "task_dag_from_task_id_fkey" FOREIGN KEY ("from_task_id")
    REFERENCES "public"."task" ("id");

ALTER TABLE "public"."task_dag"
    ADD CONSTRAINT "task_dag_to_task_id_fkey" FOREIGN KEY ("to_task_id")
    REFERENCES "public"."task" ("id");

ALTER TABLE "public"."task_run"
    ADD CONSTRAINT "task_run_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."task_run"
    ADD CONSTRAINT "task_run_sheet_id_fkey" FOREIGN KEY ("sheet_id")
    REFERENCES "public"."sheet" ("id");

ALTER TABLE "public"."task_run"
    ADD CONSTRAINT "task_run_task_id_fkey" FOREIGN KEY ("task_id")
    REFERENCES "public"."task" ("id");

ALTER TABLE "public"."task_run"
    ADD CONSTRAINT "task_run_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."task_run_log"
    ADD CONSTRAINT "task_run_log_task_run_id_fkey" FOREIGN KEY ("task_run_id")
    REFERENCES "public"."task_run" ("id");

ALTER TABLE "public"."user_group"
    ADD CONSTRAINT "user_group_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."user_group"
    ADD CONSTRAINT "user_group_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."vcs"
    ADD CONSTRAINT "vcs_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."vcs"
    ADD CONSTRAINT "vcs_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."vcs_connector"
    ADD CONSTRAINT "vcs_connector_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."vcs_connector"
    ADD CONSTRAINT "vcs_connector_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."vcs_connector"
    ADD CONSTRAINT "vcs_connector_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."vcs_connector"
    ADD CONSTRAINT "vcs_connector_vcs_id_fkey" FOREIGN KEY ("vcs_id")
    REFERENCES "public"."vcs" ("id");

ALTER TABLE "public"."worksheet"
    ADD CONSTRAINT "worksheet_creator_id_fkey" FOREIGN KEY ("creator_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."worksheet"
    ADD CONSTRAINT "worksheet_database_id_fkey" FOREIGN KEY ("database_id")
    REFERENCES "public"."db" ("id");

ALTER TABLE "public"."worksheet"
    ADD CONSTRAINT "worksheet_project_id_fkey" FOREIGN KEY ("project_id")
    REFERENCES "public"."project" ("id");

ALTER TABLE "public"."worksheet"
    ADD CONSTRAINT "worksheet_updater_id_fkey" FOREIGN KEY ("updater_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."worksheet_organizer"
    ADD CONSTRAINT "worksheet_organizer_principal_id_fkey" FOREIGN KEY ("principal_id")
    REFERENCES "public"."principal" ("id");

ALTER TABLE "public"."worksheet_organizer"
    ADD CONSTRAINT "worksheet_organizer_worksheet_id_fkey" FOREIGN KEY ("worksheet_id")
    REFERENCES "public"."worksheet" ("id");

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (101, 'NORMAL', 1, 1699026391, 1, 1737001316, 101, '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (107, 'NORMAL', 1, 1699027042, 1, 1737000968, 107, '{"name":"hr_prod_5","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (105, 'NORMAL', 1, 1699027042, 1, 1737000968, 105, '{"name":"hr_prod_3","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (106, 'NORMAL', 1, 1699027042, 1, 1737000968, 106, '{"name":"hr_prod_4","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.db_schema (id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, metadata, raw_dump, config) VALUES (109, 'NORMAL', 1, 1699027042, 1, 1737001377, 109, '{"name":"hr_prod_vcs","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"},{"name":"bugfix","position":7,"defaultExpression":"''''::text","type":"text"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

', '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: deployment_config; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.deployment_config (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, config) VALUES (101, 'NORMAL', 101, 1699028338, 101, 1699028338, 103, '', '{"schedule": {"deployments": [{"id": "98cc534c-1c61-4707-8875-6765a30f5e65", "spec": {"selector": {"matchExpressions": [{"key": "location", "values": ["asia"], "operator": "IN"}, {"key": "environment", "values": ["prod"], "operator": "IN"}]}}, "title": "Asia"}, {"id": "536b8383-d8c4-41ad-b735-dcdb73577fb4", "spec": {"selector": {"matchExpressions": [{"key": "location", "values": ["eu"], "operator": "IN"}, {"key": "environment", "values": ["prod"], "operator": "IN"}]}}, "title": "Europe"}, {"id": "8dbd167f-1cbb-485b-9b7c-639d77ac92fd", "spec": {"selector": {"matchExpressions": [{"key": "location", "values": ["na"], "operator": "IN"}, {"key": "environment", "values": ["prod"], "operator": "IN"}]}}, "title": "North America"}]}}') ON CONFLICT DO NOTHING;


--
-- Data for Name: environment; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.environment (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order", resource_id) VALUES (101, 'NORMAL', 1, 1699026378, 1, 1699026378, 'Test', 0, 'test') ON CONFLICT DO NOTHING;
INSERT INTO public.environment (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order", resource_id) VALUES (102, 'NORMAL', 1, 1699026378, 101, 1699028507, 'Prod', 1, 'prod') ON CONFLICT DO NOTHING;


--
-- Data for Name: export_archive; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: external_approval; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: idp; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: instance; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (103, 'NORMAL', 101, 1712757472, 1, 1722758734, 'prod', 'bytebase-meta', 'POSTGRES', '16.0.2', '', 'bytebase-meta', true, '{}', '{"roles": [{"name": "bb", "attribute": "Superuser Create role Create DB Replication Bypass RLS+"}], "lastSyncTime": "2024-08-04T08:05:34.620309Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (101, 'NORMAL', 101, 1699026391, 1, 1722758740, 'test', 'Test Sample Instance', 'POSTGRES', '16.0.2', '', 'test-sample-instance', true, '{}', '{"roles": [{"name": "bbsample", "attribute": "Superuser Create role Create DB Replication Bypass RLS+"}], "lastSyncTime": "2024-08-04T08:05:40.864680Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance (id, row_status, creator_id, created_ts, updater_id, updated_ts, environment, name, engine, engine_version, external_link, resource_id, activation, options, metadata) VALUES (102, 'NORMAL', 101, 1699026391, 1, 1737000964, 'prod', 'Prod Sample Instance', 'POSTGRES', '16.0.0', '', 'prod-sample-instance', true, '{}', '{"roles": [{"name": "bbsample", "attribute": "Superuser Create role Create DB Replication Bypass RLS+"}], "lastSyncTime": "2025-01-16T04:16:04.772800Z"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: instance_change_history; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.instance_change_history (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, project_id, issue_id, release_version, sequence, source, type, status, version, description, statement, sheet_id, schema, schema_prev, execution_duration_ns, payload) VALUES (197, 'NORMAL', 1, 1738990137, 1, 1738990137, NULL, NULL, NULL, NULL, '3.3.1', 96, 'LIBRARY', 'MIGRATE', 'DONE', '0003.0004.0000-20250208044856', 'Migrate version 3.4.0 server version 3.3.1 with files migration/3.4/0000##sheet_statement.sql.', 'ALTER TABLE sheet DROP COLUMN IF EXISTS statement;', NULL, '', '', 1274667, '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_change_history (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, project_id, issue_id, release_version, sequence, source, type, status, version, description, statement, sheet_id, schema, schema_prev, execution_duration_ns, payload) VALUES (196, 'NORMAL', 1, 1737613546, 1, 1737613546, NULL, NULL, NULL, NULL, '3.3.0', 95, 'LIBRARY', 'MIGRATE', 'DONE', '0003.0003.0008-20250123062546', 'Migrate version 3.3.8 server version 3.3.0 with files migration/prod/3.3/0008##semantic_null.sql.', 'DELETE FROM setting where name = ''bb.workspace.semantic-types'' AND value LIKE ''%null%'';', NULL, '', '', 1708599, '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: instance_user; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.instance_user (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, "grant") VALUES (101, 'NORMAL', 1, 1699026391, 1, 1699026391, 101, 'bbsample', 'Superuser, Create role, Create DB, Replication, Bypass RLS+') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_user (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, "grant") VALUES (102, 'NORMAL', 1, 1699026391, 1, 1699026391, 102, 'bbsample', 'Superuser, Create role, Create DB, Replication, Bypass RLS+') ON CONFLICT DO NOTHING;
INSERT INTO public.instance_user (id, row_status, creator_id, created_ts, updater_id, updated_ts, instance_id, name, "grant") VALUES (103, 'NORMAL', 1, 1712757472, 1, 1712757471, 103, 'bb', 'Superuser, Create role, Create DB, Replication, Bypass RLS+') ON CONFLICT DO NOTHING;


--
-- Data for Name: issue; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (104, 'NORMAL', 104, 1699109832, 104, 1699109967, 101, 104, 104, '[hr_prod] Alter schema @11-04 22:56 UTC+0800', 'CANCELED', 'bb.issue.database.general', '', 1, false, '{"approval": {"approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"role": "roles/tester", "type": "ANY_IN_GROUP"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}]}, "title": "Tester -> Project Owner -> DBA", "creatorId": 101, "description": "Tester -> Project Owner -> DBA"}], "approvalFindingDone": true}}', '''04'':6 ''0800'':10 ''11'':5 ''22'':7 ''56'':8 ''alter'':3 ''hr'':1 ''prod'':2 ''schema'':4 ''utc'':9') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (102, 'NORMAL', 1, 1699027633, 1, 1702562147, 102, 102, 102, 'feat: add city to employee table', 'DONE', 'bb.issue.database.general', '', 1, false, '{"approval": {"approvalFindingDone": true}}', '''20231101'':15 ''add'':6,17 ''alter'':4 ''by'':8 ''city'':7,18 ''ddl'':16 ''files'':10 ''hr'':1,12 ''prod'':2,11,13 ''schema'':5 ''sql'':19 ''vcs'':3,9,14') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (106, 'NORMAL', 1, 1712737090, 1, 1712737090, 102, 106, 106, 'feat: add phone to employee table', 'OPEN', 'bb.issue.database.general', '', 1, false, '{"approval": {"riskLevel": "HIGH", "approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_OWNER"}]}]}, "title": "Project Owner -> Workspace Admin", "creatorId": 101, "description": "Project Owner -> Workspace Admin"}], "approvalFindingDone": true}}', '''add'':2 ''employee'':5 ''feat'':1 ''phone'':3 ''table'':6 ''to'':4') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (101, 'NORMAL', 101, 1699026391, 101, 1715877770, 101, 101, 101, '👉👉👉 [START HERE] Add email column to Employee table', 'OPEN', 'bb.issue.database.general', 'A sample issue to showcase how to review database schema change.

				Click "Approve" button to apply the schema update.', 101, false, '{"labels": ["2.17.0", "bug"], "approval": {"riskLevel": "HIGH", "approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_OWNER"}]}]}, "title": "Project Owner -> Workspace Admin", "creatorId": 101, "description": "Project Owner -> Workspace Admin"}], "approvalFindingDone": true}}', '''a'':9 ''add'':3 ''apply'':24 ''approve'':21 ''button'':22 ''change'':19 ''click'':20 ''column'':5 ''database'':17 ''email'':4 ''employee'':7 ''here'':2 ''how'':14 ''issue'':11 ''review'':16 ''sample'':10 ''schema'':18,26 ''showcase'':13 ''start'':1 ''table'':8 ''the'':25 ''to'':6,12,15,23 ''update'':27') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (105, 'NORMAL', 104, 1699110335, 101, 1715877781, 101, 105, 105, 'Add performance table', 'OPEN', 'bb.issue.database.general', '', 1, false, '{"labels": ["feature"], "approval": {"approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}, {"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "WORKSPACE_DBA"}]}]}, "title": "Project Owner -> DBA", "creatorId": 1, "description": "The system defines the approval process, first the project Owner approves, then the DBA approves."}], "approvalFindingDone": true}}', '''add'':1 ''performance'':2 ''table'':3') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (107, 'NORMAL', 101, 1715878686, 1, 1715878687, 101, 107, 107, 'Update employee gender to M', 'OPEN', 'bb.issue.database.general', '', NULL, false, '{"approval": {"approvalTemplates": [{"flow": {"steps": [{"type": "ANY", "nodes": [{"type": "ANY_IN_GROUP", "groupValue": "PROJECT_OWNER"}]}]}, "title": "Project Owner", "creatorId": 1, "description": "The system defines the approval process and only needs the project Owner o approve it."}], "approvalFindingDone": true}}', '''employee'':2 ''gender'':3 ''m'':5 ''to'':4 ''update'':1') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (108, 'NORMAL', 101, 1737001075, 1, 1737001151, 102, 108, 108, 'Establish "hr_prod_vcs" baseline', 'DONE', 'bb.issue.database.general', '', NULL, false, '{"approval": {"approvalFindingDone": true}}', '''baseline'':5 ''establish'':1 ''hr'':2 ''prod'':3 ''vcs'':4') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (110, 'NORMAL', 101, 1737001312, 1, 1737001316, 101, 110, 110, 'Establish "hr_test" baseline', 'DONE', 'bb.issue.database.general', '', NULL, false, '{"approval": {"approvalFindingDone": true}}', '''baseline'':4 ''establish'':1 ''hr'':2 ''test'':3') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (109, 'NORMAL', 101, 1737001175, 1, 1737001178, 101, 109, 109, 'Establish "hr_prod" baseline', 'DONE', 'bb.issue.database.general', '', NULL, false, '{"approval": {"approvalFindingDone": true}}', '''baseline'':4 ''establish'':1 ''hr'':2 ''prod'':3') ON CONFLICT DO NOTHING;
INSERT INTO public.issue (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, plan_id, pipeline_id, name, status, type, description, assignee_id, assignee_need_attention, payload, ts_vector) VALUES (103, 'NORMAL', 106, 1699032519, 1, 1737613561, 103, 103, 103, 'Add Investor Relation department', 'DONE', 'bb.issue.database.general', '', 1, false, '{"approval": {"approvalFindingDone": true}}', '''add'':1 ''department'':4 ''investor'':2 ''relation'':3') ON CONFLICT DO NOTHING;


--
-- Data for Name: issue_comment; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (105, 'NORMAL', 104, 1699109967, 104, 1699109967, 104, '{"issueUpdate": {"toStatus": "CANCELED", "fromStatus": "OPEN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (106, 'NORMAL', 101, 1702562144, 101, 1702562144, 102, '{"taskUpdate": {"tasks": ["projects/gitops-project/rollouts/102/stages/103/tasks/103"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (107, 'NORMAL', 1, 1702562147, 1, 1702562147, 102, '{"taskUpdate": {"tasks": ["projects/gitops-project/rollouts/102/stages/103/tasks/103"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (108, 'NORMAL', 1, 1702562147, 1, 1702562147, 102, '{"stageEnd": {"stage": "projects/gitops-project/rollouts/102/stages/103"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (109, 'NORMAL', 1, 1702562147, 1, 1702562147, 102, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (110, 'NORMAL', 101, 1712736117, 101, 1712736117, 101, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/101/stages/101/tasks/101"], "toSheet": "projects/project-sample/sheets/129", "fromSheet": "projects/project-sample/sheets/102"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (111, 'NORMAL', 101, 1712736117, 101, 1712736117, 101, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/101/stages/102/tasks/102"], "toSheet": "projects/project-sample/sheets/129", "fromSheet": "projects/project-sample/sheets/103"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (112, 'NORMAL', 101, 1712736157, 101, 1712736157, 101, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/101/stages/101/tasks/101"], "toSheet": "projects/project-sample/sheets/130", "fromSheet": "projects/project-sample/sheets/129"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (113, 'NORMAL', 101, 1712736157, 101, 1712736157, 101, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/101/stages/102/tasks/102"], "toSheet": "projects/project-sample/sheets/130", "fromSheet": "projects/project-sample/sheets/129"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (114, 'NORMAL', 101, 1737001151, 101, 1737001151, 108, '{"taskUpdate": {"tasks": ["projects/gitops-project/rollouts/108/stages/111/tasks/114"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (115, 'NORMAL', 1, 1737001151, 1, 1737001151, 108, '{"taskUpdate": {"tasks": ["projects/gitops-project/rollouts/108/stages/111/tasks/114"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (116, 'NORMAL', 1, 1737001151, 1, 1737001151, 108, '{"stageEnd": {"stage": "projects/gitops-project/rollouts/108/stages/111"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (117, 'NORMAL', 1, 1737001151, 1, 1737001151, 108, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (118, 'NORMAL', 101, 1737001178, 101, 1737001178, 109, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/109/stages/112/tasks/115"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (119, 'NORMAL', 1, 1737001178, 1, 1737001178, 109, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/109/stages/112/tasks/115"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (120, 'NORMAL', 1, 1737001178, 1, 1737001178, 109, '{"stageEnd": {"stage": "projects/project-sample/rollouts/109/stages/112"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (121, 'NORMAL', 1, 1737001178, 1, 1737001178, 109, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (122, 'NORMAL', 101, 1737001316, 101, 1737001316, 110, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/110/stages/113/tasks/116"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (123, 'NORMAL', 1, 1737001316, 1, 1737001316, 110, '{"taskUpdate": {"tasks": ["projects/project-sample/rollouts/110/stages/113/tasks/116"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (124, 'NORMAL', 1, 1737001316, 1, 1737001316, 110, '{"stageEnd": {"stage": "projects/project-sample/rollouts/110/stages/113"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (125, 'NORMAL', 1, 1737001316, 1, 1737001316, 110, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (126, 'NORMAL', 1, 1737613552, 1, 1737613552, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/104/tasks/104"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (127, 'NORMAL', 1, 1737613552, 1, 1737613552, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/104/tasks/105"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (128, 'NORMAL', 1, 1737613552, 1, 1737613552, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/104/tasks/105"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (129, 'NORMAL', 1, 1737613552, 1, 1737613552, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/104/tasks/104"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (130, 'NORMAL', 1, 1737613552, 1, 1737613552, 103, '{"stageEnd": {"stage": "projects/batch-project/rollouts/103/stages/104"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (131, 'NORMAL', 1, 1737613557, 1, 1737613557, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/105/tasks/106"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (132, 'NORMAL', 1, 1737613557, 1, 1737613557, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/105/tasks/107"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (133, 'NORMAL', 1, 1737613557, 1, 1737613557, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/105/tasks/106"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (134, 'NORMAL', 1, 1737613557, 1, 1737613557, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/105/tasks/107"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (135, 'NORMAL', 1, 1737613557, 1, 1737613557, 103, '{"stageEnd": {"stage": "projects/batch-project/rollouts/103/stages/105"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (136, 'NORMAL', 1, 1737613562, 1, 1737613562, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/106/tasks/108"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (137, 'NORMAL', 1, 1737613562, 1, 1737613562, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/106/tasks/109"], "toStatus": "PENDING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (138, 'NORMAL', 1, 1737613562, 1, 1737613562, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/106/tasks/109"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (139, 'NORMAL', 1, 1737613562, 1, 1737613562, 103, '{"taskUpdate": {"tasks": ["projects/batch-project/rollouts/103/stages/106/tasks/108"], "toStatus": "DONE"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (140, 'NORMAL', 1, 1737613562, 1, 1737613562, 103, '{"stageEnd": {"stage": "projects/batch-project/rollouts/103/stages/106"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.issue_comment (id, row_status, creator_id, created_ts, updater_id, updated_ts, issue_id, payload) VALUES (141, 'NORMAL', 1, 1737613562, 1, 1737613562, 103, '{"issueUpdate": {"toStatus": "DONE", "fromStatus": "OPEN"}}') ON CONFLICT DO NOTHING;


--
-- Data for Name: issue_subscriber; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: pipeline; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (101, 'NORMAL', 101, 1699026391, 101, 1699026391, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (102, 'NORMAL', 1, 1699027633, 1, 1699027633, 102, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (103, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (104, 'NORMAL', 104, 1699109832, 104, 1699109832, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (105, 'NORMAL', 104, 1699110335, 104, 1699110335, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (106, 'NORMAL', 1, 1712737090, 1, 1712737090, 102, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (107, 'NORMAL', 101, 1715878686, 101, 1715878686, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (108, 'NORMAL', 101, 1737001075, 101, 1737001075, 102, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (109, 'NORMAL', 101, 1737001175, 101, 1737001175, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;
INSERT INTO public.pipeline (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name) VALUES (110, 'NORMAL', 101, 1737001312, 101, 1737001312, 101, 'Rollout Pipeline') ON CONFLICT DO NOTHING;


--
-- Data for Name: plan; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (102, 'NORMAL', 1, 1699027633, 1, 1699027633, 102, 102, 'feat: add city to employee table', '', '{"steps": [{"specs": [{"changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/gitops-project/sheets/104", "target": "instances/prod-sample-instance/databases/hr_prod_vcs", "schemaVersion": "1000-ddl"}}]}], "vcsSource": {"vcsType": "GITHUB", "vcsConnector": "projects/gitops-project/vcsConnectors/hr-sample", "pullRequestUrl": "https://github.com/s-bytebase/hr-sample/pull/18"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (104, 'NORMAL', 104, 1699109832, 104, 1699109832, 101, 104, '', '', '{"steps": [{"specs": [{"id": "96967c30-ee17-468e-8368-6366ccc83c52", "changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/107", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (105, 'NORMAL', 104, 1699110335, 104, 1699110335, 101, 105, '', '', '{"steps": [{"specs": [{"id": "9227f0c7-fa7d-44f3-9282-a32da230e2e4", "changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/108", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (101, 'NORMAL', 101, 1699026391, 101, 1712736157, 101, 101, 'Onboarding sample plan for adding email column to Employee table', '', '{"steps": [{"specs": [{"changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/130", "target": "instances/test-sample-instance/databases/hr_test"}}]}, {"specs": [{"changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/project-sample/sheets/130", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (106, 'NORMAL', 1, 1712737090, 1, 1712737090, 102, 106, 'feat: add phone to employee table', '', '{"steps": [{"specs": [{"id": "e4010ea4-dd1e-441a-9ea2-90f467ed8506", "changeDatabaseConfig": {"type": "MIGRATE", "sheet": "projects/gitops-project/sheets/131", "target": "instances/prod-sample-instance/databases/hr_prod_vcs", "schemaVersion": "1001"}}]}], "vcsSource": {"vcsType": "GITHUB", "vcsConnector": "projects/gitops-project/vcsConnectors/hr-sample", "pullRequestUrl": "https://github.com/s-bytebase/hr-sample/pull/17"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (107, 'NORMAL', 101, 1715878686, 101, 1715878686, 101, 107, '', '', '{"steps": [{"specs": [{"id": "0992ef9b-3d08-4745-ab40-ff74d34208a8", "changeDatabaseConfig": {"type": "DATA", "sheet": "projects/project-sample/sheets/132", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (103, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 103, '', '', '{"steps": [{"specs": [{"id": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "changeDatabaseConfig": {"type": "DATA", "sheet": "projects/batch-project/sheets/106", "target": "projects/batch-project/databaseGroups/all-databases"}}]}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (108, 'NORMAL', 101, 1737001075, 101, 1737001075, 102, 108, '', '', '{"steps": [{"specs": [{"id": "ff8ecf1c-f037-4544-971c-c3f4c8ff5889", "specReleaseSource": {}, "changeDatabaseConfig": {"type": "BASELINE", "sheet": "projects/gitops-project/sheets/133", "target": "instances/prod-sample-instance/databases/hr_prod_vcs"}}]}], "deploymentSnapshot": {"deploymentConfigSnapshot": {"name": "gitops-project/deploymentConfigs/default", "deploymentConfig": {"schedule": {"deployments": [{"id": "0", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["test"], "operator": "IN"}]}}, "title": "Test Stage"}, {"id": "1", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["prod"], "operator": "IN"}]}}, "title": "Prod Stage"}]}}}}}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (109, 'NORMAL', 101, 1737001175, 101, 1737001175, 101, 109, '', '', '{"steps": [{"specs": [{"id": "231a929d-bb89-4845-8b7c-6e4870116d32", "specReleaseSource": {}, "changeDatabaseConfig": {"type": "BASELINE", "sheet": "projects/project-sample/sheets/134", "target": "instances/prod-sample-instance/databases/hr_prod"}}]}], "deploymentSnapshot": {"deploymentConfigSnapshot": {"name": "project-sample/deploymentConfigs/default", "deploymentConfig": {"schedule": {"deployments": [{"id": "0", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["test"], "operator": "IN"}]}}, "title": "Test Stage"}, {"id": "1", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["prod"], "operator": "IN"}]}}, "title": "Prod Stage"}]}}}}}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, pipeline_id, name, description, config) VALUES (110, 'NORMAL', 101, 1737001312, 101, 1737001312, 101, 110, '', '', '{"steps": [{"specs": [{"id": "913aa19f-18e6-42c5-b6e7-2fbb358cffee", "specReleaseSource": {}, "changeDatabaseConfig": {"type": "BASELINE", "sheet": "projects/project-sample/sheets/135", "target": "instances/test-sample-instance/databases/hr_test"}}]}], "deploymentSnapshot": {"deploymentConfigSnapshot": {"name": "project-sample/deploymentConfigs/default", "deploymentConfig": {"schedule": {"deployments": [{"id": "0", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["test"], "operator": "IN"}]}}, "title": "Test Stage"}, {"id": "1", "spec": {"selector": {"matchExpressions": [{"key": "environment", "values": ["prod"], "operator": "IN"}]}}, "title": "Prod Stage"}]}}}}}') ON CONFLICT DO NOTHING;


--
-- Data for Name: plan_check_run; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (108, 1, 1699026391, 1, 1699026391, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 103, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (105, 1, 1699026391, 1, 1699026391, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 103, "instanceUid": 102, "databaseName": "hr_prod"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (104, 1, 1699026391, 1, 1699026391, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 102, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_test", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (101, 1, 1699026391, 1, 1699026391, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 102, "instanceUid": 101, "databaseName": "hr_test"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (103, 1, 1699026391, 1, 1699026391, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 102, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (107, 1, 1699026391, 1, 1699026391, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 103, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"email\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (109, 1, 1699027633, 1, 1699027633, 102, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 104, "instanceUid": 102, "databaseName": "hr_prod_vcs"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_vcs\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (111, 1, 1699027633, 1, 1699027633, 102, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 104, "instanceUid": 102, "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"city\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (112, 1, 1699027633, 1, 1699027633, 102, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 104, "instanceUid": 102, "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod_vcs", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (117, 1, 1699032519, 1, 1699032519, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_4"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_4\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (113, 1, 1699032519, 1, 1699032519, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_1"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_1\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (119, 1, 1699032519, 1, 1699032519, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_4", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (121, 1, 1699032519, 1, 1699032519, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_2"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_2\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (125, 1, 1699032519, 1, 1699032522, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_5"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_5\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (124, 1, 1699032519, 1, 1699032522, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_2", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (115, 1, 1699032519, 1, 1699032519, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_1", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (116, 1, 1699032519, 1, 1699032519, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_1", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (120, 1, 1699032519, 1, 1699032519, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_4", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (132, 1, 1699032519, 1, 1699032522, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_3", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (123, 1, 1699032519, 1, 1699032522, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_2", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (131, 1, 1699032519, 1, 1699032522, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_3", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (129, 1, 1699032519, 1, 1699032522, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_3"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_3\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (127, 1, 1699032519, 1, 1699032522, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_5", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (128, 1, 1699032519, 1, 1699032522, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_5", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (136, 1, 1699032519, 1, 1699032527, 103, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_6", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1, "statementTypes": ["INSERT"], "changedResources": {}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (135, 1, 1699032519, 1, 1699032527, 103, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_6", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (133, 1, 1699032519, 1, 1699032527, 103, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 106, "instanceUid": 102, "databaseName": "hr_prod_6"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_6\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (139, 1, 1699109832, 1, 1699109832, 104, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 107, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (140, 1, 1699109832, 1, 1699109832, 104, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 107, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (137, 1, 1699109832, 1, 1699109832, 104, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 107, "instanceUid": 102, "databaseName": "hr_prod"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (144, 1, 1699110335, 1, 1699110335, 105, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 108, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"statementTypes": ["CREATE_TABLE", "COMMENT"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "performance"}, {"name": "performance"}, {"name": "performance"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (141, 1, 1699110335, 1, 1699110335, 105, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 108, "instanceUid": 102, "databaseName": "hr_prod"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (143, 1, 1699110335, 1, 1699110335, 105, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 108, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (152, 1, 1712736117, 1, 1712736117, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 129, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (145, 1, 1712736117, 1, 1712736117, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 129, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (149, 1, 1712736117, 1, 1712736117, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 129, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (148, 1, 1712736117, 1, 1712736117, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 129, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_test", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (147, 1, 1712736117, 1, 1712736117, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 129, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (151, 1, 1712736117, 1, 1712736117, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 129, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"email\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (160, 1, 1712736157, 1, 1712736162, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 130, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (157, 1, 1712736157, 1, 1712736162, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 130, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (153, 1, 1712736157, 1, 1712736162, 101, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 130, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (156, 1, 1712736157, 1, 1712736162, 101, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 130, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_test", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (155, 1, 1712736157, 1, 1712736162, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 130, "instanceUid": 101, "databaseName": "hr_test", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS"}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (159, 1, 1712736157, 1, 1712736162, 101, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 130, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"email\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (161, 1, 1712737090, 1, 1712737090, 106, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 131, "instanceUid": 102, "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_vcs\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (163, 1, 1712737090, 1, 1712737090, 106, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 131, "instanceUid": 102, "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "column.no-null", "status": "WARNING", "content": "Column \"phone\" in \"public\".\"employee\" cannot have NULL value", "sqlReviewReport": {"code": 402, "line": 1}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (164, 1, 1712737090, 1, 1712737090, 106, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 131, "instanceUid": 102, "databaseName": "hr_prod_vcs", "changeDatabaseType": "DDL"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["ALTER_TABLE"], "changedResources": {"databases": [{"name": "hr_prod_vcs", "schemas": [{"name": "public", "tables": [{"name": "employee", "tableRows": "1000"}]}]}]}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (165, 1, 1715878686, 1, 1715878686, 107, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 132, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (168, 1, 1715878686, 1, 1715878686, 107, 'DONE', 'bb.plan-check.database.statement.summary.report', '{"sheetUid": 132, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DML"}', '{"results": [{"title": "OK", "status": "SUCCESS", "sqlSummaryReport": {"affectedRows": 1000, "statementTypes": ["UPDATE"], "changedResources": {}}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (167, 1, 1715878686, 1, 1715878686, 107, 'DONE', 'bb.plan-check.database.statement.advise', '{"sheetUid": 132, "instanceUid": 102, "databaseName": "hr_prod", "changeDatabaseType": "DML"}', '{"results": [{"title": "statement.affected-row-limit", "status": "WARNING", "content": "The statement \"UPDATE employee\nSET\n  gender = ''M''\" affected 1000 rows (estimated). The count exceeds 100.", "sqlReviewReport": {"code": 209, "line": 3}}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (169, 1, 1737001075, 1, 1737001074, 108, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 133, "instanceUid": 102, "databaseName": "hr_prod_vcs"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod_vcs\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (170, 1, 1737001175, 1, 1737001174, 109, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 134, "instanceUid": 102, "databaseName": "hr_prod"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_prod\""}]}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.plan_check_run (id, creator_id, created_ts, updater_id, updated_ts, plan_id, status, type, config, result, payload) VALUES (171, 1, 1737001312, 1, 1737001312, 110, 'DONE', 'bb.plan-check.database.connect', '{"sheetUid": 135, "instanceUid": 101, "databaseName": "hr_test"}', '{"results": [{"title": "OK", "status": "SUCCESS", "content": "Successfully connected \"hr_test\""}]}', '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: policy; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (105, 'NORMAL', 101, 1699028507, 101, 1699028507, 'bb.policy.environment-tier', '{"environmentTier": "PROTECTED"}', 'ENVIRONMENT', 102, true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (104, 'NORMAL', 101, 1699028495, 101, 1699028544, 'bb.policy.rollout', '{"issueRoles": ["roles/LAST_APPROVER"]}', 'ENVIRONMENT', 102, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (103, 'NORMAL', 101, 1699028468, 101, 1699028581, 'bb.policy.rollout', '{"automatic": true}', 'ENVIRONMENT', 101, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (108, 'NORMAL', 101, 1699029857, 101, 1699029858, 'bb.policy.disable-copy-data', '{"active": false}', 'ENVIRONMENT', 101, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (109, 'NORMAL', 101, 1699029857, 101, 1699029858, 'bb.policy.disable-copy-data', '{"active": false}', 'ENVIRONMENT', 102, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (113, 'NORMAL', 101, 1715935606, 101, 1715935606, 'bb.policy.slow-query', '{"active": true}', 'INSTANCE', 102, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (118, 'NORMAL', 101, 1699026391, 101, 1715878593, 'bb.policy.tag', '{"tags": {"bb.tag.review_config": "reviewConfigs/prod"}}', 'ENVIRONMENT', 102, true) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (134, 'NORMAL', 101, 1727534004, 101, 1727534093, 'bb.policy.data-source-query', '{"adminDataSourceRestriction": "FALLBACK"}', 'PROJECT', 101, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (106, 'NORMAL', 101, 1699029852, 101, 1720668656, 'bb.policy.masking-rule', '{"rules": [{"id": "9dda9145-895e-451a-99d8-16254c4eb287", "condition": {"expression": "environment_id == \"test\""}, "maskingLevel": "NONE"}, {"id": "d188a226-5ed6-45cc-82e3-baa890a87962", "condition": {"expression": "classification_level in [\"1\"]"}, "maskingLevel": "NONE"}, {"id": "76356d81-6231-4128-9be7-2c549fc505f5", "condition": {"expression": "classification_level in [\"2\", \"3\"]"}, "semanticType": "bb.default-partial"}, {"id": "1ddc47c9-6ab6-4760-accd-947bc1a5f155", "condition": {"expression": "classification_level in [\"4\"]"}, "semanticType": "bb.default"}]}', 'WORKSPACE', 0, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (122, 'NORMAL', 101, 1722758657, 101, 1722758657, 'bb.policy.data-source-query', '{"disallowDdl": true, "disallowDml": true, "adminDataSourceRestriction": "FALLBACK"}', 'ENVIRONMENT', 101, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (123, 'NORMAL', 101, 1722758657, 101, 1722758657, 'bb.policy.data-source-query', '{"disallowDdl": true, "disallowDml": true, "adminDataSourceRestriction": "FALLBACK"}', 'ENVIRONMENT', 102, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (114, 'NORMAL', 1, 1720666190, 1, 1720666190, 'bb.policy.iam', '{"bindings": [{"role": "roles/projectDeveloper", "members": ["users/104", "users/107", "users/106", "users/105"], "condition": {"title": "Developer"}}, {"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/projectReleaser", "members": ["users/102"], "condition": {"title": "Releaser"}}, {"role": "roles/tester", "members": ["users/108"], "condition": {"title": "Tester"}}]}', 'PROJECT', 101, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (115, 'NORMAL', 1, 1720666190, 1, 1720666190, 'bb.policy.iam', '{"bindings": [{"role": "roles/projectDeveloper", "members": ["users/107", "users/106"], "condition": {"title": "Developer"}}, {"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/projectOwner", "members": ["users/104", "users/105"], "condition": {"title": "Owner"}}]}', 'PROJECT', 102, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (116, 'NORMAL', 1, 1720666190, 1, 1720666190, 'bb.policy.iam', '{"bindings": [{"role": "roles/projectDeveloper", "members": ["users/106", "users/107"], "condition": {"title": "Developer"}}, {"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/projectOwner", "members": ["users/102", "users/103"], "condition": {"title": "Owner"}}, {"role": "roles/sqlEditorUser", "members": ["users/104", "users/105"], "condition": {"title": "Querier All"}}]}', 'PROJECT', 103, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (117, 'NORMAL', 1, 1720666190, 1, 1720666190, 'bb.policy.iam', '{"bindings": [{"role": "roles/projectOwner", "members": ["users/101"], "condition": {}}, {"role": "roles/sqlEditorUser", "members": ["users/104"], "condition": {"title": "Project Querier All"}}]}', 'PROJECT', 104, false) ON CONFLICT DO NOTHING;
INSERT INTO public.policy (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent) VALUES (119, 'NORMAL', 1, 1721635577, 1, 1738999494, 'bb.policy.iam', '{"bindings": [{"role": "roles/workspaceDBA", "members": ["users/102", "users/103", "users/109"]}, {"role": "roles/workspaceAdmin", "members": ["users/101", "users/1"]}, {"role": "roles/workspaceMember", "members": ["allUsers"]}]}', 'WORKSPACE', 0, false) ON CONFLICT DO NOTHING;


--
-- Data for Name: principal; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (1, 'NORMAL', 1, 1738995349, 1, 1738995349, 'SYSTEM_BOT', 'Bytebase', 'support@bytebase.com', '', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (2, 'NORMAL', 2, 1738995349, 2, 1738995349, 'SYSTEM_BOT', 'All Users', 'allUsers', '', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (102, 'NORMAL', 1, 1699028630, 101, 1699028932, 'END_USER', 'dba1', 'dba1@example.com', '$2a$10$mjuC.ej22zhysY3ylsR00eqFGVPxctD4RMZN7mio7GjhTFg5o6nPG', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (103, 'NORMAL', 1, 1699028631, 101, 1699028941, 'END_USER', 'dba2', 'dba2@example.com', '$2a$10$UIKJY.ziyCuB0fIG.AkuBOlcPoYtzvVZZfm4Uh3OrgbF0VLTneUbC', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (105, 'NORMAL', 1, 1699028631, 101, 1699028964, 'END_USER', 'dev2', 'dev2@example.com', '$2a$10$Fst2F8T3GCRKsLoAh5937.qkFVwsbygmu2FKriu0B1nQave1VKXQC', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (106, 'NORMAL', 1, 1699028631, 101, 1699028972, 'END_USER', 'dev3', 'dev3@example.com', '$2a$10$b6X5Pk/Ffe7YtDTrJcqtKuP.e9OmdH3Kq9i/WaTUO9225Pud6yd/6', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (107, 'NORMAL', 1, 1699028631, 101, 1699028978, 'END_USER', 'dev4', 'dev4@example.com', '$2a$10$ikN0OjIzqoCuOtR21FRtTuTS5LenyJSdonyL.VOphI9LDTgOQ6NcC', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (108, 'NORMAL', 1, 1699109310, 1, 1699109310, 'END_USER', 'qa1', 'qa1@example.com', '$2a$10$tgPwB2JdZlyu2MD/W.IxluFMI8bM9IPgYSQYaQEIBYT0SO23QM5Iu', '', '{}', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (104, 'NORMAL', 1, 1699028631, 1, 1726820189, 'END_USER', 'dev1', 'dev1@example.com', '$2a$10$hX4vTGH7Id6v9BWhHHtW9uHT.M/ANZ25owa5J9m1tSS5qzlSCkjSu', '', '{}', '{"lastLoginTime": "2024-09-20T08:16:29.553490Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (109, 'NORMAL', 1, 1712736378, 101, 1737650968, 'SERVICE_ACCOUNT', 'ci', 'ci@service.bytebase.com', '$2a$10$LLSrQ6pPSnqIqml/PUh3G.WjrLUMod2l8hWbSKa4qsVQ7bw9ZTNc.', '', '{}', '{"lastChangePasswordTime": "2025-01-23T16:49:28.865322699Z"}') ON CONFLICT DO NOTHING;
INSERT INTO public.principal (id, row_status, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash, phone, mfa_config, profile) VALUES (101, 'NORMAL', 1, 1699026391, 1, 1738990171, 'END_USER', 'Demo', 'demo@example.com', '$2a$10$aKjyVRxwbzmNToxYLXgTn.cQZX9x8KI1LLu5U69zzn5wcaoagoBLG', '', '{}', '{"lastLoginTime": "2025-02-08T04:49:31.431086962Z"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: project; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.project (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, resource_id, data_classification_config_id, setting) VALUES (1, 'NORMAL', 1, 1699026378, 101, 1720669088, 'Default', 'DEFAULT', 'default', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"postgresDatabaseTenantMode": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, resource_id, data_classification_config_id, setting) VALUES (102, 'NORMAL', 101, 1699026423, 101, 1720669088, 'GitOps Project', 'GITP', 'gitops-project', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"autoResolveIssue": true, "allowModifyStatement": true, "postgresDatabaseTenantMode": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, resource_id, data_classification_config_id, setting) VALUES (103, 'NORMAL', 101, 1699027705, 101, 1720669088, 'Batch Project', 'BATP', 'batch-project', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"autoResolveIssue": true, "allowModifyStatement": true, "postgresDatabaseTenantMode": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, resource_id, data_classification_config_id, setting) VALUES (104, 'NORMAL', 101, 1712757515, 101, 1720669088, 'MetaDB Project', 'META', 'metadb-project', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"autoResolveIssue": true, "allowModifyStatement": true, "postgresDatabaseTenantMode": true}') ON CONFLICT DO NOTHING;
INSERT INTO public.project (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, resource_id, data_classification_config_id, setting) VALUES (101, 'NORMAL', 101, 1699026391, 101, 1736961552, 'Basic Project', 'BASP', 'project-sample', '2b599739-41da-4c35-a9ff-4a73c6cfe32c', '{"issueLabels": [{"color": "#4f46e5", "value": "2.17.0"}, {"color": "#E55146", "value": "bug"}, {"color": "#E5B546", "value": "feature"}], "autoResolveIssue": true, "allowSelfApproval": true, "allowModifyStatement": true, "postgresDatabaseTenantMode": true}') ON CONFLICT DO NOTHING;


--
-- Data for Name: project_webhook; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: query_history; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (101, 'NORMAL', 101, 1699029734, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM salary;', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (102, 'NORMAL', 101, 1699029868, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM salary;', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (103, 'NORMAL', 104, 1699029898, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_1', 'SELECT * FROM salary', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (104, 'NORMAL', 104, 1699030039, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_1', 'SELECT * FROM department', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (105, 'NORMAL', 104, 1699030045, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_1', 'SELECT * FROM department', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (106, 'NORMAL', 104, 1699030045, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_2', 'SELECT * FROM department', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (107, 'NORMAL', 104, 1699030045, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_3', 'SELECT * FROM department', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (108, 'NORMAL', 104, 1699030045, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_4', 'SELECT * FROM department', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (109, 'NORMAL', 104, 1699030045, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_5', 'SELECT * FROM department', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (110, 'NORMAL', 104, 1699030045, 'batch-project', 'instances/prod-sample-instance/databases/hr_prod_6', 'SELECT * FROM department', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (111, 'NORMAL', 101, 1699032082, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (112, 'NORMAL', 101, 1699032153, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM salary;', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (113, 'NORMAL', 101, 1699032179, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (114, 'NORMAL', 101, 1699032394, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM department;', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (115, 'NORMAL', 101, 1700552743, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (116, 'NORMAL', 101, 1700552753, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (117, 'NORMAL', 101, 1712757578, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
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
  project.resource_id;', 'QUERY', '{"error": "", "duration": "0.002s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (118, 'NORMAL', 101, 1712757749, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT project.resource_id, count(*)
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
GROUP BY project.resource_id;', 'QUERY', '{"error": "", "duration": "0.003567s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (119, 'NORMAL', 101, 1712757976, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
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
', 'QUERY', '{"error": "", "duration": "0.002620s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (120, 'NORMAL', 101, 1712758421, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'WITH issue_approvers AS (
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
  month;', 'QUERY', '{"error": "", "duration": "0.002993s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (121, 'NORMAL', 101, 1712758550, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'WITH issue_approvers AS (
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
  month;', 'QUERY', '{"error": "", "duration": "0.003365s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (122, 'NORMAL', 104, 1712758765, 'metadb-project', 'instances/bytebase-meta/databases/bb', '-- Fully completed issues by project
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
  project.resource_id;', 'QUERY', '{"error": "", "duration": "0.002810s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (123, 'NORMAL', 101, 1715935620, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT pg_sleep(5)', 'QUERY', '{"error": "", "duration": "5.013435s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (124, 'NORMAL', 101, 1720669329, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{"duration": "0.008848s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (125, 'NORMAL', 101, 1720669334, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM salary;', 'QUERY', '{"duration": "0.009147s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (126, 'NORMAL', 101, 1720669353, 'project-sample', 'instances/prod-sample-instance/databases/hr_prod', 'SELECT * FROM employee;', 'QUERY', '{"duration": "0.011041s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (127, 'NORMAL', 101, 1726820034, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
  issue.creator_id,
  principal.email,
  COUNT(issue.creator_id) AS amount
FROM issue
INNER JOIN principal
ON issue.creator_id = principal.id
GROUP BY issue.creator_id, principal.email
ORDER BY COUNT(issue.creator_id) DESC;', 'QUERY', '{"duration": "0.024319125s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (128, 'NORMAL', 101, 1737001369, 'gitops-project', 'instances/prod-sample-instance/databases/hr_prod_vcs', 'ALTER TABLE employee ADD COLUMN bugfix TEXT NOT NULL DEFAULT '''';', 'QUERY', '{"duration": "0.009309583s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (129, 'NORMAL', 101, 1737613572, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
  *
FROM
  "public"."release"
LIMIT
  50;', 'QUERY', '{"duration": "0.000996134s"}') ON CONFLICT DO NOTHING;
INSERT INTO public.query_history (id, row_status, creator_id, created_ts, project_id, database, statement, type, payload) VALUES (130, 'NORMAL', 101, 1737613575, 'metadb-project', 'instances/bytebase-meta/databases/bb', 'SELECT
  *
FROM
  "public"."release"
LIMIT
  50;', 'QUERY', '{"duration": "0.002934110s"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: release; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: review_config; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.review_config (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, payload) VALUES ('prod', 'NORMAL', 101, 1699026391, 101, 1715878593, 'SQL Review Sample Policy', '{"sqlReviewRules": [{"type": "database.drop-empty-database", "level": "ERROR", "engine": "MYSQL", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "TIDB", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "OCEANBASE", "payload": "{}"}, {"type": "database.drop-empty-database", "level": "ERROR", "engine": "MARIADB", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MYSQL", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "TIDB", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "POSTGRES", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "ORACLE", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MSSQL", "payload": "{}"}, {"type": "column.no-null", "level": "WARNING", "engine": "MARIADB", "payload": "{}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MYSQL", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "TIDB", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "POSTGRES", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "OCEANBASE", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "SNOWFLAKE", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MSSQL", "payload": "{\"format\":\"_del$\"}"}, {"type": "table.drop-naming-convention", "level": "ERROR", "engine": "MARIADB", "payload": "{\"format\":\"_del$\"}"}, {"type": "engine.mysql.use-innodb", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "engine.mysql.use-innodb", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "table.require-pk", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "table.no-foreign-key", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "table.disallow-partition", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "table.disallow-trigger", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.comment", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "table.comment", "level": "DISABLED", "engine": "TIDB", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "table.comment", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "table.comment", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "table.no-duplicate-index", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "table.text-fields-total-length", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":1000}"}, {"type": "table.disallow-set-charset", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "statement.select.no-select-all", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "statement.where.require", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.where.no-leading-wildcard-like", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.disallow-commit", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.disallow-on-del-cascade", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-rm-tbl-cascade", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.disallow-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.disallow-order-by", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-order-by", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-order-by", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.disallow-order-by", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.merge-alter-table", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.insert.must-specify-column", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.insert.disallow-order-by-rand", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.insert.row-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":1000}"}, {"type": "statement.insert.row-limit", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":1000}"}, {"type": "statement.insert.row-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":1000}"}, {"type": "statement.insert.row-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":1000}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "MYSQL", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "POSTGRES", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "OCEANBASE", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "statement.affected-row-limit", "level": "WARNING", "engine": "MARIADB", "comment": "Reveal the number of rows to be updated or deleted can help determine whether the statement meets business expectations. Suggestion error level: Warning", "payload": "{\"number\":100}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.dml-dry-run", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-add-column-with-default", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.add-check-not-valid", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-add-not-null", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.select-full-table-scan", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.select-full-table-scan", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.select-full-table-scan", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "statement.create-specify-schema", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.check-set-role-variable", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-using-filesort", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-using-temporary", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.where.no-equal-null", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.query.minimum-plan-level", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"string\":\"INDEX\"}"}, {"type": "statement.where.maximum-logical-operator-count", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":10}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-limit-value", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":1000}"}, {"type": "statement.maximum-join-table-count", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":10}"}, {"type": "statement.maximum-statements-in-transaction", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":10}"}, {"type": "statement.join-strict-column-attrs", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.prior-backup-check", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.prior-backup-check", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.prior-backup-check", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "naming.fully-qualified", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "naming.table", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.table.no-keyword", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "naming.table.no-keyword", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "naming.table.no-keyword", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "naming.table.no-keyword", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "naming.column", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.column", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.column", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.column", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.column", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^[a-z]+(_[a-z]+)*$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.uk", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^$|^uk_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.pk", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^$|^pk_{{table}}_{{column_list}}$\"}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.idx", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^$|^idx_{{table}}_{{column_list}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.index.fk", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$\",\"maxLength\":63}"}, {"type": "naming.column.auto-increment", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"format\":\"^id$\",\"maxLength\":63}"}, {"type": "naming.column.auto-increment", "level": "DISABLED", "engine": "TIDB", "payload": "{\"format\":\"^id$\",\"maxLength\":63}"}, {"type": "naming.column.auto-increment", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"format\":\"^id$\",\"maxLength\":63}"}, {"type": "naming.column.auto-increment", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"format\":\"^id$\",\"maxLength\":63}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "naming.identifier.no-keyword", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "naming.identifier.case", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"upper\":true}"}, {"type": "naming.identifier.case", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"upper\":true}"}, {"type": "naming.identifier.case", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{\"upper\":true}"}, {"type": "column.required", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"list\":[\"ID\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{\"list\":[\"ID\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.required", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"list\":[\"id\",\"created_ts\",\"updated_ts\",\"creator_id\",\"updater_id\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.type-disallow-list", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"list\":[\"JSON\"]}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-change-type", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.disallow-drop-in-index", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-drop-in-index", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-drop-in-index", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-drop-in-index", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.set-default-for-not-null", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.disallow-change", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-change", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-change", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-change", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.disallow-changing-order", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-changing-order", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-changing-order", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-changing-order", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.auto-increment-must-integer", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.auto-increment-must-integer", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.auto-increment-must-integer", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.auto-increment-must-integer", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.disallow-set-charset", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.disallow-set-charset", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.disallow-set-charset", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-set-charset", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.auto-increment-must-unsigned", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.auto-increment-must-unsigned", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.auto-increment-must-unsigned", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.auto-increment-must-unsigned", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.comment", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "column.comment", "level": "DISABLED", "engine": "TIDB", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "column.comment", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "column.comment", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"required\":true,\"maxLength\":64}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":20}"}, {"type": "column.maximum-character-length", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":20}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{\"number\":2560}"}, {"type": "column.maximum-varchar-length", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"number\":2560}"}, {"type": "column.auto-increment-initial-value", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":1}"}, {"type": "column.auto-increment-initial-value", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":1}"}, {"type": "column.auto-increment-initial-value", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":1}"}, {"type": "column.auto-increment-initial-value", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":1}"}, {"type": "column.current-time-count-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.current-time-count-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.current-time-count-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.current-time-count-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "ORACLE", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.require-default", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "SNOWFLAKE", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "MSSQL", "payload": "{}"}, {"type": "schema.backward-compatibility", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "index.no-duplicate-column", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "index.type-no-blob", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "index.type-no-blob", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "index.type-no-blob", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "index.type-no-blob", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "index.pk-type-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "index.pk-type-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "index.pk-type-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "index.pk-type-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "ORACLE", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "OCEANBASE_ORACLE", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":5}"}, {"type": "index.key-number-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "TIDB", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"number\":5}"}, {"type": "index.total-number-limit", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"number\":5}"}, {"type": "index.primary-key-type-allowlist", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"SERIAL\",\"BIGSERIAL\",\"INT\",\"BIGINT\"]}"}, {"type": "index.primary-key-type-allowlist", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"SERIAL\",\"BIGSERIAL\",\"INT\",\"BIGINT\"]}"}, {"type": "index.primary-key-type-allowlist", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"SERIAL\",\"BIGSERIAL\",\"INT\",\"BIGINT\"]}"}, {"type": "index.primary-key-type-allowlist", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"SERIAL\",\"BIGSERIAL\",\"INT\",\"BIGINT\"]}"}, {"type": "index.create-concurrently", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "index.type-allow-list", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"BTREE\",\"HASH\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.charset.allowlist", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"list\":[\"utf8mb4\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "TIDB", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.collation.allowlist", "level": "DISABLED", "engine": "MARIADB", "payload": "{\"list\":[\"utf8mb4_0900_ai_ci\"]}"}, {"type": "system.comment.length", "level": "DISABLED", "engine": "POSTGRES", "payload": "{\"number\":64}"}, {"type": "system.procedure.disallow-create", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "system.event.disallow-create", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "system.view.disallow-create", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "system.function.disallow-create", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "system.function.disallowed-list", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"list\":[\"RAND\",\"UUID\",\"SLEEP\"]}"}, {"type": "table.disallow-ddl", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"list\":[]}"}, {"type": "table.disallow-dml", "level": "DISABLED", "engine": "MSSQL", "payload": "{\"list\":[]}"}, {"type": "table.limit-size", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":10000000}"}, {"type": "statement.add-column-without-position", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "statement.disallow-offline-ddl", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "column.disallow-drop", "level": "DISABLED", "engine": "OCEANBASE", "payload": "{}"}, {"type": "advice.online-migration", "level": "DISABLED", "engine": "MYSQL", "payload": "{\"number\":100000000}"}, {"type": "statement.add-foreign-key-not-valid", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.non-transactional", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-mix-in-ddl", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-mix-in-ddl", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-mix-in-ddl", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}, {"type": "statement.disallow-mix-in-dml", "level": "DISABLED", "engine": "MYSQL", "payload": "{}"}, {"type": "statement.disallow-mix-in-dml", "level": "DISABLED", "engine": "POSTGRES", "payload": "{}"}, {"type": "statement.disallow-mix-in-dml", "level": "DISABLED", "engine": "TIDB", "payload": "{}"}]}') ON CONFLICT DO NOTHING;


--
-- Data for Name: revision; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: risk; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.risk (id, row_status, creator_id, created_ts, updater_id, updated_ts, source, level, name, active, expression) VALUES (101, 'NORMAL', 101, 1699029149, 101, 1699110101, 'bb.risk.database.schema.update', 300, 'ALTER column in production environment is high risk', true, '{"expression": "environment_id == \"prod\" && sql_type == \"ALTER_TABLE\""}') ON CONFLICT DO NOTHING;
INSERT INTO public.risk (id, row_status, creator_id, created_ts, updater_id, updated_ts, source, level, name, active, expression) VALUES (102, 'NORMAL', 101, 1699110145, 101, 1699110145, 'bb.risk.database.schema.update', 200, 'CREATE TABLE in production environment is moderate risk', true, '{"expression": "environment_id == \"prod\" && sql_type == \"CREATE_TABLE\""}') ON CONFLICT DO NOTHING;


--
-- Data for Name: role; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.role (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_id, name, description, permissions, payload) VALUES (101, 'NORMAL', 101, 1699029034, 101, 1712734700, 'tester', 'Tester', 'Custom defined Tester role', '{"permissions": ["bb.changelogs.get", "bb.changelogs.list", "bb.databases.get", "bb.databases.getSchema", "bb.databases.list", "bb.issueComments.create", "bb.issues.get", "bb.issues.list", "bb.planCheckRuns.list", "bb.planCheckRuns.run", "bb.plans.get", "bb.plans.list", "bb.projects.get", "bb.projects.getIamPolicy", "bb.rollouts.get", "bb.taskRuns.list"]}', '{}') ON CONFLICT DO NOTHING;


--
-- Data for Name: setting; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (101, 'NORMAL', 1, 1699026378, 1, 1699026378, 'bb.branding.logo', '', 'The branding slogo image in base64 string format.') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (102, 'NORMAL', 1, 1699026378, 1, 1699026378, 'bb.auth.secret', '9Dw1H9JSeEWfjfRnxR5VZ8wuDCIL9ERq', 'Random string used to sign the JWT auth token.') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (103, 'NORMAL', 1, 1699026378, 1, 1699026378, 'bb.workspace.id', '6c86d081-379d-4366-be6f-481425e6f397', 'The workspace identifier') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (106, 'NORMAL', 1, 1699026378, 1, 1699026378, 'bb.workspace.watermark', '0', 'Display watermark') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (107, 'NORMAL', 1, 1699026378, 1, 1699026378, 'bb.plugin.openai.key', '', 'API key to request OpenAI (ChatGPT)') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (108, 'NORMAL', 1, 1699026378, 1, 1699026378, 'bb.plugin.openai.endpoint', '', 'API Endpoint for OpenAI') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (109, 'NORMAL', 1, 1699026378, 1, 1699026378, 'bb.workspace.approval.external', '{}', 'The external approval setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (104, 'NORMAL', 1, 1699026378, 101, 1699027105, 'bb.enterprise.license', 'eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA', 'Enterprise license') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (112, 'NORMAL', 1, 1699026378, 101, 1715878490, 'bb.workspace.approval', '{"rules":[{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_OWNER"}]}]},"title":"Project Owner -> Workspace Admin","description":"Project Owner -> Workspace Admin","creatorId":101},"condition":{"expression":"source == \"DDL\" && level == 300"}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","role":"roles/tester"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]}]},"title":"Tester -> Project Owner -> DBA","description":"Tester -> Project Owner -> DBA","creatorId":101},"condition":{}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]}]},"title":"Project Owner -> DBA","description":"The system defines the approval process, first the project Owner approves, then the DBA approves.","creatorId":1},"condition":{}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]}]},"title":"Project Owner","description":"The system defines the approval process and only needs the project Owner o approve it.","creatorId":1},"condition":{"expression":"source == \"DDL\" && level == 200 || source == \"REQUEST_QUERY\" && level == 0 || source == \"REQUEST_EXPORT\" &&\nlevel == 0 || source == \"DATA_EXPORT\" && level == 0 || source == \"DML\" && level == 0 ||\nsource == \"CREATE_DATABASE\" && level == 0 || source == \"DDL\" && level == 0"}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]}]},"title":"DBA","description":"The system defines the approval process and only needs DBA approval.","creatorId":1},"condition":{}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_OWNER"}]}]},"title":"Workspace Admin","description":"The system defines the approval process and only needs Administrator approval.","creatorId":1},"condition":{}},{"template":{"flow":{"steps":[{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"PROJECT_OWNER"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_DBA"}]},{"type":"ANY","nodes":[{"type":"ANY_IN_GROUP","groupValue":"WORKSPACE_OWNER"}]}]},"title":"Project Owner -> DBA -> Workspace Admin","description":"The system defines the approval process, first the project Owner approves, then the DBA approves, and finally the Administrator approves.","creatorId":1},"condition":{}}]}', 'The workspace approval setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (105, 'NORMAL', 1, 1699026378, 1, 1699026378, 'bb.app.im', '{}', '') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (111, 'NORMAL', 1, 1699026378, 101, 1720669088, 'bb.workspace.data-classification', '{"configs":[{"id":"2b599739-41da-4c35-a9ff-4a73c6cfe32c", "title":"Classification Example", "levels":[{"id":"1", "title":"Level 1"}, {"id":"2", "title":"Level 2"}, {"id":"3", "title":"Level 3"}, {"id":"4", "title":"Level 4"}], "classification":{"1":{"id":"1", "title":"Basic"}, "1-1":{"id":"1-1", "title":"Basic", "levelId":"1"}, "1-2":{"id":"1-2", "title":"Assert", "levelId":"1"}, "1-3":{"id":"1-3", "title":"Contact", "levelId":"2"}, "1-4":{"id":"1-4", "title":"Health", "levelId":"4"}, "2":{"id":"2", "title":"Relationship"}, "2-1":{"id":"2-1", "title":"Social", "levelId":"1"}, "2-2":{"id":"2-2", "title":"Business", "levelId":"3"}}, "classificationFromConfig":true}]}', 'The data classification setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (175, 'NORMAL', 1, 1726805202, 1, 1726805202, 'bb.workspace.scim', '{"token":"2dc3NMIruyrj0NuaJNOiPCZLLhJGpNr2"}', 'The SCIM sync') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (176, 'NORMAL', 1, 1726805202, 1, 1726805202, 'bb.workspace.password-restriction', '{"minLength":8,"requireLetter":true}', 'The password validation') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (110, 'NORMAL', 1, 1699026378, 101, 1699031931, 'bb.workspace.schema-template', '{"fieldTemplates":[{"id":"b281c610-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "column":{"name":"creator", "type":"TEXT", "classification":"1-1"}, "catalog":{"name":"creator"}}, {"id":"c5ddd410-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "column":{"name":"updater", "type":"TEXT", "classification":"1-1"}, "catalog":{"name":"updater"}}, {"id":"ce566850-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "column":{"name":"created_ts", "type":"DATE"}, "catalog":{"name":"created_ts"}}, {"id":"d8900d80-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "column":{"name":"updated_ts", "type":"DATE"}, "catalog":{"name":"updated_ts"}}], "tableTemplates":[{"id":"f0fca590-7a6c-11ee-bfb8-958ed997c3e9", "engine":"POSTGRES", "category":"common", "table":{"name":"Basic Table", "columns":[{"name":"creator", "type":"TEXT", "classification":"1-1"}, {"name":"created_ts", "type":"DATE"}, {"name":"updater", "type":"TEXT", "classification":"1-1"}, {"name":"updated_ts", "type":"DATE"}]}, "catalog":{"name":"Basic Table", "columnConfigs":[{"name":"creator"}, {"name":"created_ts"}, {"name":"updater"}, {"name":"updated_ts"}]}}]}', 'The schema template setting') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (183, 'NORMAL', 1, 1736952715, 1, 1736952715, 'bb.plugin.openai.model', '', 'Model for OpenAI') ON CONFLICT DO NOTHING;
INSERT INTO public.setting (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, value, description) VALUES (113, 'NORMAL', 1, 1699026378, 1, 1738999493, 'bb.workspace.profile', '{"externalUrl":"https://demo.bytebase.com"}', '') ON CONFLICT DO NOTHING;


--
-- Data for Name: sheet; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (102, 'NORMAL', 1, 1699026391, 1, 1699026391, 101, 'Alter table to test sample instance for sample issue', '\xdc3cbdad177e12396a1be4e31c959f7b0fdf03193c21bcfa113da7fa23109222', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (103, 'NORMAL', 1, 1699026391, 1, 1699026391, 101, 'Alter table to prod sample instance for sample issue', '\xdc3cbdad177e12396a1be4e31c959f7b0fdf03193c21bcfa113da7fa23109222', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (106, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 'Add Investor Relation department', '\xcc61fce0c8a2f03c11a9850cafa453f9206ae0ca61916a8fe73013e4107d487d', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (107, 'NORMAL', 104, 1699109832, 104, 1699109832, 101, '[hr_prod] Alter schema @11-04 22:56 UTC+0800', '\x4252fc11123f200fefb2a248f6af18dfebe2990efe69276f8cf282b038ac742f', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (108, 'NORMAL', 104, 1699110335, 104, 1699110335, 101, 'Add performance table', '\x57a98f9ee5dcd8876d226a80be428980e3a5363fbb208274f8be1a1186b5a77e', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (104, 'NORMAL', 1, 1699027633, 1, 1712734702, 102, 'bytebase/prod/hr_prod_vcs##20231101##ddl##add_city.sql', '\xecae727c3bd9f84b999130b569547715ba3b02e273f2edc0d3424e996b0b9980', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (129, 'NORMAL', 101, 1712736117, 101, 1712736117, 101, '👉👉👉 [START HERE] Add email column to Employee table', '\x1bea40d232b6f1beafe7aac3ffb41b7e2b438c614ba0edfa21d41559e30ce4e6', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (130, 'NORMAL', 101, 1712736157, 101, 1712736157, 101, '👉👉👉 [START HERE] Add email column to Employee table', '\xdc3cbdad177e12396a1be4e31c959f7b0fdf03193c21bcfa113da7fa23109222', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (131, 'NORMAL', 1, 1712737090, 1, 1712737090, 102, 'bytebase/1001_add_phone.sql', '\x11af6768516979ab350006efbdb9cc4b8edbd9fd6be129233862d5f9784069ee', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (132, 'NORMAL', 101, 1715878686, 101, 1715878686, 101, 'Update employee gender to M', '\x012e88c9ae3bed1ad96913baaf979523a61e705ba0113460c6f8dca0182cb54d', '{"engine": "POSTGRES", "commands": [{"end": 34}]}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (133, 'NORMAL', 101, 1737001075, 101, 1737001075, 102, 'Establish "hr_prod_vcs" baseline', '\xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', '{"engine": "POSTGRES"}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (134, 'NORMAL', 101, 1737001175, 101, 1737001175, 101, 'Establish "hr_prod" baseline', '\xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', '{"engine": "POSTGRES"}') ON CONFLICT DO NOTHING;
INSERT INTO public.sheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, name, sha256, payload) VALUES (135, 'NORMAL', 101, 1737001312, 101, 1737001312, 101, 'Establish "hr_test" baseline', '\xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', '{"engine": "POSTGRES"}') ON CONFLICT DO NOTHING;


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
-- Data for Name: slow_query; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: sql_lint_config; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: stage; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (101, 'NORMAL', 101, 1699026391, 101, 1699026391, 101, 101, '', 'Test Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (102, 'NORMAL', 101, 1699026391, 101, 1699026391, 101, 102, '', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (103, 'NORMAL', 1, 1699027633, 1, 1699027633, 102, 102, '', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (104, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 102, '', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (105, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 102, '', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (106, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 102, '', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (107, 'NORMAL', 104, 1699109832, 104, 1699109832, 104, 102, '', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (108, 'NORMAL', 104, 1699110335, 104, 1699110335, 105, 102, '', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (109, 'NORMAL', 1, 1712737090, 1, 1712737090, 106, 102, '', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (110, 'NORMAL', 101, 1715878686, 101, 1715878686, 107, 102, '', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (111, 'NORMAL', 101, 1737001075, 101, 1737001075, 108, 102, '1', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (112, 'NORMAL', 101, 1737001175, 101, 1737001175, 109, 102, '1', 'Prod Stage') ON CONFLICT DO NOTHING;
INSERT INTO public.stage (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, environment_id, deployment_id, name) VALUES (113, 'NORMAL', 101, 1737001312, 101, 1737001312, 110, 101, '0', 'Test Stage') ON CONFLICT DO NOTHING;


--
-- Data for Name: sync_history; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.sync_history (id, creator_id, created_ts, database_id, metadata, raw_dump) VALUES (101, 1, '2023-12-14 05:55:44-08', 109, '{}', '
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, creator_id, created_ts, database_id, metadata, raw_dump) VALUES (102, 1, '2023-12-14 05:55:44-08', 109, '{}', '
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, creator_id, created_ts, database_id, metadata, raw_dump) VALUES (103, 1, '2025-01-15 20:19:11.013893-08', 109, '{"name":"hr_prod_vcs","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, creator_id, created_ts, database_id, metadata, raw_dump) VALUES (104, 1, '2025-01-15 20:19:11.102061-08', 109, '{"name":"hr_prod_vcs","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, creator_id, created_ts, database_id, metadata, raw_dump) VALUES (105, 1, '2025-01-15 20:19:38.009574-08', 102, '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, creator_id, created_ts, database_id, metadata, raw_dump) VALUES (106, 1, '2025-01-15 20:19:38.056824-08', 102, '{"name":"hr_prod","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, creator_id, created_ts, database_id, metadata, raw_dump) VALUES (107, 1, '2025-01-15 20:21:56.236717-08', 101, '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

') ON CONFLICT DO NOTHING;
INSERT INTO public.sync_history (id, creator_id, created_ts, database_id, metadata, raw_dump) VALUES (108, 1, '2025-01-15 20:21:56.282054-08', 101, '{"name":"hr_test","schemas":[{"name":"bbdataarchive","owner":"bbsample"},{"name":"public","tables":[{"name":"audit","columns":[{"name":"id","position":1,"defaultExpression":"nextval(''public.audit_id_seq''::regclass)","type":"integer"},{"name":"operation","position":2,"type":"text"},{"name":"query","position":3,"nullable":true,"type":"text"},{"name":"user_name","position":4,"type":"text"},{"name":"changed_at","position":5,"defaultExpression":"CURRENT_TIMESTAMP","nullable":true,"type":"timestamp with time zone"}],"indexes":[{"name":"audit_pkey","expressions":["id"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX audit_pkey ON public.audit USING btree (id);","isConstraint":true},{"name":"idx_audit_changed_at","expressions":["changed_at"],"type":"btree","definition":"CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);"},{"name":"idx_audit_operation","expressions":["operation"],"type":"btree","definition":"CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);"},{"name":"idx_audit_username","expressions":["user_name"],"type":"btree","definition":"CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);"}],"dataSize":"8192","indexSize":"32768","owner":"bbsample"},{"name":"department","columns":[{"name":"dept_no","position":1,"type":"text"},{"name":"dept_name","position":2,"type":"text"}],"indexes":[{"name":"department_dept_name_key","expressions":["dept_name"],"type":"btree","unique":true,"definition":"CREATE UNIQUE INDEX department_dept_name_key ON public.department USING btree (dept_name);","isConstraint":true},{"name":"department_pkey","expressions":["dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX department_pkey ON public.department USING btree (dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"32768","owner":"bbsample"},{"name":"dept_emp","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_emp_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_emp_pkey ON public.dept_emp USING btree (emp_no, dept_no);","isConstraint":true}],"rowCount":"1103","dataSize":"106496","indexSize":"57344","foreignKeys":[{"name":"dept_emp_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_emp_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"dept_manager","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"dept_no","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"dept_manager_pkey","expressions":["emp_no","dept_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX dept_manager_pkey ON public.dept_manager USING btree (emp_no, dept_no);","isConstraint":true}],"dataSize":"16384","indexSize":"16384","foreignKeys":[{"name":"dept_manager_dept_no_fkey","columns":["dept_no"],"referencedSchema":"public","referencedTable":"department","referencedColumns":["dept_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"},{"name":"dept_manager_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"},{"name":"employee","columns":[{"name":"emp_no","position":1,"defaultExpression":"nextval(''public.employee_emp_no_seq''::regclass)","type":"integer"},{"name":"birth_date","position":2,"type":"date"},{"name":"first_name","position":3,"type":"text"},{"name":"last_name","position":4,"type":"text"},{"name":"gender","position":5,"type":"text"},{"name":"hire_date","position":6,"type":"date"}],"indexes":[{"name":"employee_pkey","expressions":["emp_no"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX employee_pkey ON public.employee USING btree (emp_no);","isConstraint":true},{"name":"idx_employee_hire_date","expressions":["hire_date"],"type":"btree","definition":"CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);"}],"rowCount":"1000","dataSize":"98304","indexSize":"98304","owner":"bbsample"},{"name":"salary","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"amount","position":2,"type":"integer"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"type":"date"}],"indexes":[{"name":"idx_salary_amount","expressions":["amount"],"type":"btree","definition":"CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);"},{"name":"salary_pkey","expressions":["emp_no","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX salary_pkey ON public.salary USING btree (emp_no, from_date);","isConstraint":true}],"rowCount":"9488","dataSize":"458752","indexSize":"548864","foreignKeys":[{"name":"salary_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample","triggers":[{"name":"salary_log_trigger","body":"CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations()"}]},{"name":"title","columns":[{"name":"emp_no","position":1,"type":"integer"},{"name":"title","position":2,"type":"text"},{"name":"from_date","position":3,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}],"indexes":[{"name":"title_pkey","expressions":["emp_no","title","from_date"],"type":"btree","unique":true,"primary":true,"definition":"CREATE UNIQUE INDEX title_pkey ON public.title USING btree (emp_no, title, from_date);","isConstraint":true}],"rowCount":"1470","dataSize":"131072","indexSize":"73728","foreignKeys":[{"name":"title_emp_no_fkey","columns":["emp_no"],"referencedSchema":"public","referencedTable":"employee","referencedColumns":["emp_no"],"onDelete":"CASCADE","onUpdate":"NO ACTION","matchType":"SIMPLE"}],"owner":"bbsample"}],"views":[{"name":"current_dept_emp","definition":" SELECT l.emp_no,\n    d.dept_no,\n    l.from_date,\n    l.to_date\n   FROM (public.dept_emp d\n     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"dept_no"},{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"},{"schema":"public","table":"dept_emp_latest_date","column":"emp_no"},{"schema":"public","table":"dept_emp_latest_date","column":"from_date"},{"schema":"public","table":"dept_emp_latest_date","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"dept_no","position":2,"nullable":true,"type":"text"},{"name":"from_date","position":3,"nullable":true,"type":"date"},{"name":"to_date","position":4,"nullable":true,"type":"date"}]},{"name":"dept_emp_latest_date","definition":" SELECT emp_no,\n    max(from_date) AS from_date,\n    max(to_date) AS to_date\n   FROM public.dept_emp\n  GROUP BY emp_no;","dependencyColumns":[{"schema":"public","table":"dept_emp","column":"emp_no"},{"schema":"public","table":"dept_emp","column":"from_date"},{"schema":"public","table":"dept_emp","column":"to_date"}],"columns":[{"name":"emp_no","position":1,"nullable":true,"type":"integer"},{"name":"from_date","position":2,"nullable":true,"type":"date"},{"name":"to_date","position":3,"nullable":true,"type":"date"}]}],"functions":[{"name":"log_dml_operations","definition":"CREATE OR REPLACE FUNCTION public.log_dml_operations()\n RETURNS trigger\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n    IF (TG_OP = ''INSERT'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''INSERT'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''UPDATE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''UPDATE'', current_query(), current_user);\n        RETURN NEW;\n    ELSIF (TG_OP = ''DELETE'') THEN\n        INSERT INTO audit (operation, query, user_name)\n        VALUES (''DELETE'', current_query(), current_user);\n        RETURN OLD;\n    END IF;\n    RETURN NULL;\nEND;\n$function$\n","signature":"log_dml_operations()"}],"sequences":[{"name":"audit_id_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"audit","ownerColumn":"id"},{"name":"employee_emp_no_seq","dataType":"integer","start":"1","minValue":"1","maxValue":"2147483647","increment":"1","cacheSize":"1","ownerTable":"employee","ownerColumn":"emp_no"}],"owner":"pg_database_owner"}],"characterSet":"UTF8","collation":"en_US.UTF-8","extensions":[{"name":"pg_stat_statements","schema":"public","version":"1.10","description":"track planning and execution statistics of all SQL statements executed"}],"owner":"bbsample"}', '
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

') ON CONFLICT DO NOTHING;


--
-- Data for Name: task; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (103, 'NORMAL', 1, 1699027633, 1, 1699027633, 102, 103, 102, 109, 'DDL(schema) for database "hr_prod_vcs"', 'PENDING_APPROVAL', 'bb.task.database.schema.update', '{"sheetId": 104}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (104, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 104, 102, 103, 'DML(data) for database "hr_prod_1"', 'PENDING_APPROVAL', 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (105, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 104, 102, 106, 'DML(data) for database "hr_prod_4"', 'PENDING_APPROVAL', 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (106, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 105, 102, 104, 'DML(data) for database "hr_prod_2"', 'PENDING_APPROVAL', 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (107, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 105, 102, 107, 'DML(data) for database "hr_prod_5"', 'PENDING_APPROVAL', 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (108, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 106, 102, 105, 'DML(data) for database "hr_prod_3"', 'PENDING_APPROVAL', 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (109, 'NORMAL', 106, 1699032519, 106, 1699032519, 103, 106, 102, 108, 'DML(data) for database "hr_prod_6"', 'PENDING_APPROVAL', 'bb.task.database.data.update', '{"specId": "2b77e8db-cfbf-4148-aac9-39965fbd43e3", "sheetId": 106, "rollbackSqlStatus": "PENDING"}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (110, 'NORMAL', 104, 1699109832, 104, 1699109832, 104, 107, 102, 102, 'DDL(schema) for database "hr_prod"', 'PENDING_APPROVAL', 'bb.task.database.schema.update', '{"specId": "96967c30-ee17-468e-8368-6366ccc83c52", "sheetId": 107}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (111, 'NORMAL', 104, 1699110335, 104, 1699110335, 105, 108, 102, 102, 'DDL(schema) for database "hr_prod"', 'PENDING_APPROVAL', 'bb.task.database.schema.update', '{"specId": "9227f0c7-fa7d-44f3-9282-a32da230e2e4", "sheetId": 108}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (101, 'NORMAL', 101, 1699026391, 101, 1712736157, 101, 101, 101, 101, 'DDL(schema) for database "hr_test"', 'PENDING_APPROVAL', 'bb.task.database.schema.update', '{"sheetId": 130}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (102, 'NORMAL', 101, 1699026391, 101, 1712736157, 101, 102, 102, 102, 'DDL(schema) for database "hr_prod"', 'PENDING_APPROVAL', 'bb.task.database.schema.update', '{"sheetId": 130}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (112, 'NORMAL', 1, 1712737090, 1, 1712737090, 106, 109, 102, 109, 'DDL(schema) for database "hr_prod_vcs"', 'PENDING_APPROVAL', 'bb.task.database.schema.update', '{"specId": "e4010ea4-dd1e-441a-9ea2-90f467ed8506", "sheetId": 131}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (113, 'NORMAL', 101, 1715878686, 101, 1715878686, 107, 110, 102, 102, 'DML(data) for database "hr_prod"', 'PENDING_APPROVAL', 'bb.task.database.data.update', '{"specId": "0992ef9b-3d08-4745-ab40-ff74d34208a8", "sheetId": 132, "rollbackSqlStatus": "PENDING", "preUpdateBackupDetail": {}}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (114, 'NORMAL', 101, 1737001075, 101, 1737001075, 108, 111, 102, 109, 'Establish baseline for database "hr_prod_vcs"', 'PENDING_APPROVAL', 'bb.task.database.schema.baseline', '{"specId": "ff8ecf1c-f037-4544-971c-c3f4c8ff5889", "taskReleaseSource": {}}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (115, 'NORMAL', 101, 1737001175, 101, 1737001175, 109, 112, 102, 102, 'Establish baseline for database "hr_prod"', 'PENDING_APPROVAL', 'bb.task.database.schema.baseline', '{"specId": "231a929d-bb89-4845-8b7c-6e4870116d32", "taskReleaseSource": {}}', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.task (id, row_status, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts) VALUES (116, 'NORMAL', 101, 1737001312, 101, 1737001312, 110, 113, 101, 101, 'Establish baseline for database "hr_test"', 'PENDING_APPROVAL', 'bb.task.database.schema.baseline', '{"specId": "913aa19f-18e6-42c5-b6e7-2fbb358cffee", "taskReleaseSource": {}}', 0) ON CONFLICT DO NOTHING;


--
-- Data for Name: task_dag; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: task_run; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (101, 101, 1702562144, 1, 1702562147, 103, 104, 0, 'DDL(schema) for database "hr_prod_vcs" 1702562144', 'DONE', 1702562144, 0, '{"detail": "Applied migration version 1000-ddl to database \"hr_prod_vcs\".", "version": "0000.0000.0000-1000-ddl", "changelog": "instances/prod-sample-instance/databases/hr_prod_vcs/changelogs/110", "changeHistory": "instances/prod-sample-instance/databases/hr_prod_vcs/changeHistories/110"}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (102, 101, 1737001151, 1, 1737001151, 114, NULL, 0, 'Establish baseline for database "hr_prod_vcs" 1737001150', 'DONE', 1737001151, 0, '{"detail": "Established baseline version  for database \"hr_prod_vcs\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_vcs/changelogs/112"}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (103, 101, 1737001178, 1, 1737001178, 115, NULL, 0, 'Establish baseline for database "hr_prod" 1737001177', 'DONE', 1737001178, 0, '{"detail": "Established baseline version  for database \"hr_prod\".", "changelog": "instances/prod-sample-instance/databases/hr_prod/changelogs/113"}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (104, 101, 1737001316, 1, 1737001316, 116, NULL, 0, 'Establish baseline for database "hr_test" 1737001316', 'DONE', 1737001316, 0, '{"detail": "Established baseline version  for database \"hr_test\".", "changelog": "instances/test-sample-instance/databases/hr_test/changelogs/114"}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (106, 1, 1737613552, 1, 1737613551, 105, 106, 0, 'DML(data) for database "hr_prod_4" 1737613551', 'DONE', 1737613552, 0, '{"detail": "Applied migration version  to database \"hr_prod_4\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_4/changelogs/115"}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (105, 1, 1737613552, 1, 1737613551, 104, 106, 0, 'DML(data) for database "hr_prod_1" 1737613551', 'DONE', 1737613552, 0, '{"detail": "Applied migration version  to database \"hr_prod_1\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_1/changelogs/116"}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (107, 1, 1737613557, 1, 1737613556, 106, 106, 0, 'DML(data) for database "hr_prod_2" 1737613556', 'DONE', 1737613557, 0, '{"detail": "Applied migration version  to database \"hr_prod_2\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_2/changelogs/117"}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (108, 1, 1737613557, 1, 1737613556, 107, 106, 0, 'DML(data) for database "hr_prod_5" 1737613556', 'DONE', 1737613557, 0, '{"detail": "Applied migration version  to database \"hr_prod_5\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_5/changelogs/118"}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (110, 1, 1737613562, 1, 1737613561, 109, 106, 0, 'DML(data) for database "hr_prod_6" 1737613561', 'DONE', 1737613562, 0, '{"detail": "Applied migration version  to database \"hr_prod_6\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_6/changelogs/119"}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run (id, creator_id, created_ts, updater_id, updated_ts, task_id, sheet_id, attempt, name, status, started_ts, code, result) VALUES (109, 1, 1737613562, 1, 1737613561, 108, 106, 0, 'DML(data) for database "hr_prod_3" 1737613561', 'DONE', 1737613562, 0, '{"detail": "Applied migration version  to database \"hr_prod_3\".", "changelog": "instances/prod-sample-instance/databases/hr_prod_3/changelogs/120"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: task_run_log; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (101, 102, '2025-01-15 20:19:10.926834-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (102, 102, '2025-01-15 20:19:10.930371-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (103, 103, '2025-01-15 20:19:37.82721-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (104, 103, '2025-01-15 20:19:37.828077-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (105, 104, '2025-01-15 20:21:56.124197-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (106, 104, '2025-01-15 20:21:56.125869-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "e5644974", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (107, 105, '2025-01-22 22:25:51.541499-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (108, 106, '2025-01-22 22:25:51.545367-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (109, 105, '2025-01-22 22:25:51.547841-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (110, 106, '2025-01-22 22:25:51.550248-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (111, 106, '2025-01-22 22:25:51.554241-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (112, 106, '2025-01-22 22:25:51.55543-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (113, 105, '2025-01-22 22:25:51.556511-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (114, 105, '2025-01-22 22:25:51.558478-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (115, 106, '2025-01-22 22:25:51.586945-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (116, 105, '2025-01-22 22:25:51.586968-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (117, 106, '2025-01-22 22:25:51.587939-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (118, 105, '2025-01-22 22:25:51.589077-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (119, 106, '2025-01-22 22:25:51.590906-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (120, 105, '2025-01-22 22:25:51.591974-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (121, 106, '2025-01-22 22:25:51.592644-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (122, 105, '2025-01-22 22:25:51.595153-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (123, 107, '2025-01-22 22:25:56.545924-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (124, 108, '2025-01-22 22:25:56.549066-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (125, 107, '2025-01-22 22:25:56.551389-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (126, 108, '2025-01-22 22:25:56.553426-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (127, 107, '2025-01-22 22:25:56.55349-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (128, 107, '2025-01-22 22:25:56.555496-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (129, 108, '2025-01-22 22:25:56.557263-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (130, 108, '2025-01-22 22:25:56.558372-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (131, 107, '2025-01-22 22:25:56.580767-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (132, 108, '2025-01-22 22:25:56.582381-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (133, 107, '2025-01-22 22:25:56.582827-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (134, 108, '2025-01-22 22:25:56.584187-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (135, 107, '2025-01-22 22:25:56.58685-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (136, 108, '2025-01-22 22:25:56.586902-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (137, 107, '2025-01-22 22:25:56.589412-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (138, 108, '2025-01-22 22:25:56.590801-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (139, 109, '2025-01-22 22:26:01.54505-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (140, 110, '2025-01-22 22:26:01.547684-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_WAITING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (141, 109, '2025-01-22 22:26:01.549862-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (142, 110, '2025-01-22 22:26:01.551335-08', '{"type": "TASK_RUN_STATUS_UPDATE", "deployId": "ed4b14ad", "taskRunStatusUpdate": {"status": "RUNNING_RUNNING"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (143, 109, '2025-01-22 22:26:01.552123-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (144, 109, '2025-01-22 22:26:01.553567-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (145, 110, '2025-01-22 22:26:01.554971-08', '{"type": "PRIOR_BACKUP_START", "deployId": "ed4b14ad", "priorBackupStart": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (146, 110, '2025-01-22 22:26:01.55597-08', '{"type": "PRIOR_BACKUP_END", "deployId": "ed4b14ad", "priorBackupEnd": {}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (147, 110, '2025-01-22 22:26:01.578364-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (151, 110, '2025-01-22 22:26:01.583553-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (152, 110, '2025-01-22 22:26:01.585495-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (154, 109, '2025-01-22 22:26:01.588284-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "COMMIT"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (148, 110, '2025-01-22 22:26:01.580435-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (149, 109, '2025-01-22 22:26:01.58233-08', '{"type": "TRANSACTION_CONTROL", "deployId": "ed4b14ad", "transactionControl": {"type": "BEGIN"}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (150, 109, '2025-01-22 22:26:01.583143-08', '{"type": "COMMAND_EXECUTE", "deployId": "ed4b14ad", "commandExecute": {"commandIndexes": [0]}}') ON CONFLICT DO NOTHING;
INSERT INTO public.task_run_log (id, task_run_id, created_ts, payload) VALUES (153, 109, '2025-01-22 22:26:01.585567-08', '{"type": "COMMAND_RESPONSE", "deployId": "ed4b14ad", "commandResponse": {"affectedRows": 1, "commandIndexes": [0], "allAffectedRows": [1]}}') ON CONFLICT DO NOTHING;


--
-- Data for Name: user_group; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: vcs; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.vcs (id, row_status, creator_id, created_ts, updater_id, updated_ts, resource_id, name, type, instance_url, access_token) VALUES (102, 'NORMAL', 101, 1712736355, 101, 1712736355, 'githubucom-a6ug', 'GitHub.com', 'GITHUB', 'https://github.com', 'redacted') ON CONFLICT DO NOTHING;


--
-- Data for Name: vcs_connector; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.vcs_connector (id, row_status, creator_id, created_ts, updater_id, updated_ts, vcs_id, project_id, resource_id, payload) VALUES (104, 'NORMAL', 101, 1712736981, 101, 1712736981, 102, 102, 'hr-sample', '{"title": "hr-sample", "branch": "main", "webUrl": "https://github.com/s-bytebase/hr-sample", "fullPath": "s-bytebase/hr-sample", "externalId": "s-bytebase/hr-sample", "baseDirectory": "bytebase", "externalWebhookId": "471715274", "webhookSecretToken": "JiUzpc2tBHX7LVeI"}') ON CONFLICT DO NOTHING;


--
-- Data for Name: worksheet; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.worksheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, payload) VALUES (105, 'NORMAL', 101, 1699032185, 101, 1712734699, 101, 102, 'All employee', 'SELECT * FROM employee;', 'PROJECT_READ', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, payload) VALUES (101, 'NORMAL', 101, 1699026391, 101, 1712734699, 101, 102, 'All salary', 'SELECT * FROM salary;', 'PROJECT_READ', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, payload) VALUES (108, 'NORMAL', 101, 1712758032, 101, 1712758753, 104, 111, 'Issues with the same creator and releaser', 'SELECT
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
  issue.id', 'PROJECT_READ', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, payload) VALUES (106, 'NORMAL', 101, 1712757712, 101, 1712757726, 104, 111, 'Completed issues by project', '-- Fully completed issues by project
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
  project.resource_id;', 'PROJECT_READ', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, payload) VALUES (107, 'NORMAL', 101, 1712757913, 101, 1712758092, 104, 111, 'Issues reviewed during a period', '-- Issues reviewed between 17:30 and 23:59:59
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
  project.resource_id;', 'PROJECT_READ', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, payload) VALUES (109, 'NORMAL', 101, 1712758142, 101, 1712758750, 104, 111, 'Issues with the same approver and releaser by month', 'WITH issue_approvers AS (
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
  month;', 'PROJECT_READ', '{}') ON CONFLICT DO NOTHING;
INSERT INTO public.worksheet (id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, payload) VALUES (110, 'NORMAL', 101, 1726820093, 101, 1726820141, 104, 111, 'Issues created by user', '-- Issues created by user
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
-- Name: activity_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.activity_id_seq', 194, true);


--
-- Name: anomaly_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.anomaly_id_seq', 107, true);


--
-- Name: audit_log_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.audit_log_id_seq', 158, true);


--
-- Name: branch_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.branch_id_seq', 105, true);


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

SELECT pg_catalog.setval('public.db_id_seq', 111, true);


--
-- Name: db_schema_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.db_schema_id_seq', 179, true);


--
-- Name: deployment_config_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.deployment_config_id_seq', 101, true);


--
-- Name: environment_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.environment_id_seq', 103, false);


--
-- Name: export_archive_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.export_archive_id_seq', 1, false);


--
-- Name: external_approval_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.external_approval_id_seq', 101, false);


--
-- Name: idp_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.idp_id_seq', 101, false);


--
-- Name: instance_change_history_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.instance_change_history_id_seq', 197, true);


--
-- Name: instance_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.instance_id_seq', 103, true);


--
-- Name: instance_user_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.instance_user_id_seq', 103, true);


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

SELECT pg_catalog.setval('public.policy_id_seq', 169, true);


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

SELECT pg_catalog.setval('public.setting_id_seq', 196, true);


--
-- Name: sheet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.sheet_id_seq', 135, true);


--
-- Name: slow_query_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.slow_query_id_seq', 101, true);


--
-- Name: stage_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.stage_id_seq', 113, true);


--
-- Name: sync_history_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.sync_history_id_seq', 108, true);


--
-- Name: task_dag_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.task_dag_id_seq', 101, false);


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
-- Name: vcs_connector_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.vcs_connector_id_seq', 101, false);


--
-- Name: vcs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.vcs_id_seq', 102, true);


--
-- Name: worksheet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.worksheet_id_seq', 110, true);


--
-- Name: worksheet_organizer_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.worksheet_organizer_id_seq', 1, false);


--
-- Name: activity activity_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity
    ADD CONSTRAINT activity_pkey PRIMARY KEY (id);


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
-- Name: branch branch_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.branch
    ADD CONSTRAINT branch_pkey PRIMARY KEY (id);


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
-- Name: deployment_config deployment_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_pkey PRIMARY KEY (id);


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
-- Name: external_approval external_approval_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_pkey PRIMARY KEY (id);


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
-- Name: instance_user instance_user_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_pkey PRIMARY KEY (id);


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
-- Name: slow_query slow_query_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_pkey PRIMARY KEY (id);


--
-- Name: sql_lint_config sql_lint_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sql_lint_config
    ADD CONSTRAINT sql_lint_config_pkey PRIMARY KEY (id);


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
-- Name: task_dag task_dag_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_dag
    ADD CONSTRAINT task_dag_pkey PRIMARY KEY (id);


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
-- Name: vcs_connector vcs_connector_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs_connector
    ADD CONSTRAINT vcs_connector_pkey PRIMARY KEY (id);


--
-- Name: vcs vcs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs
    ADD CONSTRAINT vcs_pkey PRIMARY KEY (id);


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
-- Name: idx_activity_container_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_activity_container_id ON public.activity USING btree (container_id);


--
-- Name: idx_activity_created_ts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_activity_created_ts ON public.activity USING btree (created_ts);


--
-- Name: idx_activity_resource_container; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_activity_resource_container ON public.activity USING btree (resource_container);


--
-- Name: idx_anomaly_unique_project_database_id_type; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_anomaly_unique_project_database_id_type ON public.anomaly USING btree (project, database_id, type);


--
-- Name: idx_audit_log_created_ts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_created_ts ON public.audit_log USING btree (created_ts);


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
-- Name: idx_branch_reconcile_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_branch_reconcile_state ON public.branch USING btree (reconcile_state);


--
-- Name: idx_branch_unique_project_id_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_branch_unique_project_id_name ON public.branch USING btree (project_id, name);


--
-- Name: idx_changelist_project_id_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_changelist_project_id_name ON public.changelist USING btree (project_id, name);


--
-- Name: idx_changelog_database_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_changelog_database_id ON public.changelog USING btree (database_id);


--
-- Name: idx_data_source_unique_instance_id_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON public.data_source USING btree (instance_id, name);


--
-- Name: idx_db_group_unique_project_id_placeholder; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_id_placeholder ON public.db_group USING btree (project_id, placeholder);


--
-- Name: idx_db_group_unique_project_id_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_id_resource_id ON public.db_group USING btree (project_id, resource_id);


--
-- Name: idx_db_instance_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_db_instance_id ON public.db USING btree (instance_id);


--
-- Name: idx_db_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_db_project_id ON public.db USING btree (project_id);


--
-- Name: idx_db_schema_unique_database_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_schema_unique_database_id ON public.db_schema USING btree (database_id);


--
-- Name: idx_db_unique_instance_id_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_unique_instance_id_name ON public.db USING btree (instance_id, name);


--
-- Name: idx_deployment_config_unique_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_deployment_config_unique_project_id ON public.deployment_config USING btree (project_id);


--
-- Name: idx_environment_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_environment_unique_resource_id ON public.environment USING btree (resource_id);


--
-- Name: idx_external_approval_row_status_issue_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_approval_row_status_issue_id ON public.external_approval USING btree (row_status, issue_id);


--
-- Name: idx_idp_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON public.idp USING btree (resource_id);


--
-- Name: idx_instance_change_history_unique_instance_id_database_id_sequ; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_sequ ON public.instance_change_history USING btree (instance_id, database_id, sequence);


--
-- Name: idx_instance_change_history_unique_instance_id_database_id_vers; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_vers ON public.instance_change_history USING btree (instance_id, database_id, version);


--
-- Name: idx_instance_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON public.instance USING btree (resource_id);


--
-- Name: idx_instance_user_unique_instance_id_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_instance_user_unique_instance_id_name ON public.instance_user USING btree (instance_id, name);


--
-- Name: idx_issue_assignee_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_assignee_id ON public.issue USING btree (assignee_id);


--
-- Name: idx_issue_comment_issue_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_comment_issue_id ON public.issue_comment USING btree (issue_id);


--
-- Name: idx_issue_created_ts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_created_ts ON public.issue USING btree (created_ts);


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
-- Name: idx_issue_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_project_id ON public.issue USING btree (project_id);


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
-- Name: idx_plan_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_project_id ON public.plan USING btree (project_id);


--
-- Name: idx_policy_unique_resource_type_resource_id_type; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_id_type ON public.policy USING btree (resource_type, resource_id, type);


--
-- Name: idx_project_unique_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_project_unique_key ON public.project USING btree (key);


--
-- Name: idx_project_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_project_unique_resource_id ON public.project USING btree (resource_id);


--
-- Name: idx_project_webhook_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_project_webhook_project_id ON public.project_webhook USING btree (project_id);


--
-- Name: idx_project_webhook_unique_project_id_url; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_project_webhook_unique_project_id_url ON public.project_webhook USING btree (project_id, url);


--
-- Name: idx_query_history_creator_id_created_ts_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_query_history_creator_id_created_ts_project_id ON public.query_history USING btree (creator_id, created_ts, project_id DESC);


--
-- Name: idx_release_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_release_project_id ON public.release USING btree (project_id);


--
-- Name: idx_revision_database_id_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_revision_database_id_version ON public.revision USING btree (database_id, version);


--
-- Name: idx_revision_unique_database_id_version_deleted_ts_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_revision_unique_database_id_version_deleted_ts_null ON public.revision USING btree (database_id, version) WHERE (deleted_ts IS NULL);


--
-- Name: idx_role_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_role_unique_resource_id ON public.role USING btree (resource_id);


--
-- Name: idx_setting_unique_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_setting_unique_name ON public.setting USING btree (name);


--
-- Name: idx_sheet_creator_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sheet_creator_id ON public.sheet USING btree (creator_id);


--
-- Name: idx_sheet_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sheet_name ON public.sheet USING btree (name);


--
-- Name: idx_sheet_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sheet_project_id ON public.sheet USING btree (project_id);


--
-- Name: idx_sheet_project_id_row_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sheet_project_id_row_status ON public.sheet USING btree (project_id, row_status);


--
-- Name: idx_slow_query_instance_id_log_date_ts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_slow_query_instance_id_log_date_ts ON public.slow_query USING btree (instance_id, log_date_ts);


--
-- Name: idx_stage_pipeline_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_stage_pipeline_id ON public.stage USING btree (pipeline_id);


--
-- Name: idx_sync_history_database_id_created_ts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sync_history_database_id_created_ts ON public.sync_history USING btree (database_id, created_ts);


--
-- Name: idx_task_dag_from_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_dag_from_task_id ON public.task_dag USING btree (from_task_id);


--
-- Name: idx_task_dag_to_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_dag_to_task_id ON public.task_dag USING btree (to_task_id);


--
-- Name: idx_task_earliest_allowed_ts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_earliest_allowed_ts ON public.task USING btree (earliest_allowed_ts);


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
-- Name: idx_task_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_status ON public.task USING btree (status);


--
-- Name: idx_vcs_connector_unique_project_id_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_vcs_connector_unique_project_id_resource_id ON public.vcs_connector USING btree (project_id, resource_id);


--
-- Name: idx_vcs_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_vcs_unique_resource_id ON public.vcs USING btree (resource_id);


--
-- Name: idx_worksheet_creator_id_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_worksheet_creator_id_project_id ON public.worksheet USING btree (creator_id, project_id);


--
-- Name: idx_worksheet_organizer_principal_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_worksheet_organizer_principal_id ON public.worksheet_organizer USING btree (principal_id);


--
-- Name: idx_worksheet_organizer_unique_sheet_id_principal_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal_id ON public.worksheet_organizer USING btree (worksheet_id, principal_id);


--
-- Name: uk_slow_query_database_id_log_date_ts; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uk_slow_query_database_id_log_date_ts ON public.slow_query USING btree (database_id, log_date_ts);


--
-- Name: uk_task_run_task_id_attempt; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON public.task_run USING btree (task_id, attempt);


--
-- Name: activity activity_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity
    ADD CONSTRAINT activity_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: activity activity_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity
    ADD CONSTRAINT activity_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: anomaly anomaly_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: anomaly anomaly_database_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);


--
-- Name: anomaly anomaly_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);


--
-- Name: anomaly anomaly_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.anomaly
    ADD CONSTRAINT anomaly_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: branch branch_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.branch
    ADD CONSTRAINT branch_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: branch branch_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.branch
    ADD CONSTRAINT branch_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: branch branch_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.branch
    ADD CONSTRAINT branch_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: changelist changelist_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: changelist changelist_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: changelist changelist_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: changelog changelog_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: changelog changelog_database_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);


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
-- Name: data_source data_source_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: data_source data_source_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);


--
-- Name: data_source data_source_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: db db_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: db db_environment_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_environment_fkey FOREIGN KEY (environment) REFERENCES public.environment(resource_id);


--
-- Name: db_group db_group_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: db_group db_group_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: db_group db_group_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: db db_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);


--
-- Name: db db_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: db_schema db_schema_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: db_schema db_schema_database_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id) ON DELETE CASCADE;


--
-- Name: db_schema db_schema_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: db db_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: deployment_config deployment_config_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: deployment_config deployment_config_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: deployment_config deployment_config_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployment_config
    ADD CONSTRAINT deployment_config_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: environment environment_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: environment environment_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: external_approval external_approval_approver_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_approver_id_fkey FOREIGN KEY (approver_id) REFERENCES public.principal(id);


--
-- Name: external_approval external_approval_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);


--
-- Name: external_approval external_approval_requester_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_approval
    ADD CONSTRAINT external_approval_requester_id_fkey FOREIGN KEY (requester_id) REFERENCES public.principal(id);


--
-- Name: instance_change_history instance_change_history_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: instance_change_history instance_change_history_database_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);


--
-- Name: instance_change_history instance_change_history_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);


--
-- Name: instance_change_history instance_change_history_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);


--
-- Name: instance_change_history instance_change_history_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: instance_change_history instance_change_history_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: instance instance_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: instance instance_environment_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_environment_fkey FOREIGN KEY (environment) REFERENCES public.environment(resource_id);


--
-- Name: instance instance_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: instance_user instance_user_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: instance_user instance_user_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);


--
-- Name: instance_user instance_user_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_user
    ADD CONSTRAINT instance_user_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: issue issue_assignee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_assignee_id_fkey FOREIGN KEY (assignee_id) REFERENCES public.principal(id);


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
-- Name: issue_comment issue_comment_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comment
    ADD CONSTRAINT issue_comment_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


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
-- Name: issue issue_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


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
-- Name: issue issue_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: pipeline pipeline_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: pipeline pipeline_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: pipeline pipeline_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: plan_check_run plan_check_run_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: plan_check_run plan_check_run_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plan(id);


--
-- Name: plan_check_run plan_check_run_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


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
-- Name: plan plan_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: plan plan_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: policy policy_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: policy policy_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: principal principal_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: principal principal_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: project project_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: project project_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: project_webhook project_webhook_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: project_webhook project_webhook_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: project_webhook project_webhook_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


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
-- Name: release release_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release
    ADD CONSTRAINT release_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: review_config review_config_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_config
    ADD CONSTRAINT review_config_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: review_config review_config_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_config
    ADD CONSTRAINT review_config_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: revision revision_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision
    ADD CONSTRAINT revision_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: revision revision_database_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision
    ADD CONSTRAINT revision_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);


--
-- Name: revision revision_deleter_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision
    ADD CONSTRAINT revision_deleter_id_fkey FOREIGN KEY (deleter_id) REFERENCES public.principal(id);


--
-- Name: risk risk_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: risk risk_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: role role_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: role role_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: setting setting_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: setting setting_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: sheet sheet_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: sheet sheet_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: sheet sheet_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: slow_query slow_query_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: slow_query slow_query_database_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);


--
-- Name: slow_query slow_query_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);


--
-- Name: slow_query slow_query_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slow_query
    ADD CONSTRAINT slow_query_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: sql_lint_config sql_lint_config_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sql_lint_config
    ADD CONSTRAINT sql_lint_config_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: sql_lint_config sql_lint_config_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sql_lint_config
    ADD CONSTRAINT sql_lint_config_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: stage stage_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: stage stage_environment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_environment_id_fkey FOREIGN KEY (environment_id) REFERENCES public.environment(id);


--
-- Name: stage stage_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);


--
-- Name: stage stage_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage
    ADD CONSTRAINT stage_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: sync_history sync_history_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sync_history
    ADD CONSTRAINT sync_history_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: sync_history sync_history_database_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sync_history
    ADD CONSTRAINT sync_history_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);


--
-- Name: task task_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: task_dag task_dag_from_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_dag
    ADD CONSTRAINT task_dag_from_task_id_fkey FOREIGN KEY (from_task_id) REFERENCES public.task(id);


--
-- Name: task_dag task_dag_to_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_dag
    ADD CONSTRAINT task_dag_to_task_id_fkey FOREIGN KEY (to_task_id) REFERENCES public.task(id);


--
-- Name: task task_database_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);


--
-- Name: task task_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.instance(id);


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
-- Name: task_run task_run_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: task task_stage_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.stage(id);


--
-- Name: task task_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: user_group user_group_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group
    ADD CONSTRAINT user_group_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: user_group user_group_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group
    ADD CONSTRAINT user_group_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: vcs_connector vcs_connector_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs_connector
    ADD CONSTRAINT vcs_connector_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: vcs_connector vcs_connector_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs_connector
    ADD CONSTRAINT vcs_connector_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: vcs_connector vcs_connector_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs_connector
    ADD CONSTRAINT vcs_connector_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: vcs_connector vcs_connector_vcs_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs_connector
    ADD CONSTRAINT vcs_connector_vcs_id_fkey FOREIGN KEY (vcs_id) REFERENCES public.vcs(id);


--
-- Name: vcs vcs_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs
    ADD CONSTRAINT vcs_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: vcs vcs_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vcs
    ADD CONSTRAINT vcs_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- Name: worksheet worksheet_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: worksheet worksheet_database_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_database_id_fkey FOREIGN KEY (database_id) REFERENCES public.db(id);


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
-- Name: worksheet worksheet_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project(id);


--
-- Name: worksheet worksheet_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_updater_id_fkey FOREIGN KEY (updater_id) REFERENCES public.principal(id);


--
-- PostgreSQL database dump complete
--

