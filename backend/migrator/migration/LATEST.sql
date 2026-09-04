-----------------------
-- Global identity: workspace and principal
-- We will use the IAM policy to list the principal's workspaces.
-----------------------

-- Global server configuration (single row, not workspace-scoped).
CREATE TABLE server_config (
    -- Stored as ServerConfigPayload (proto/store/store/server_config.proto)
    payload     jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE workspace (
    resource_id text PRIMARY KEY,
    -- Stored as WorkspacePayload (proto/store/store/workspace.proto)
    payload     jsonb NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now(),
    deleted     boolean NOT NULL DEFAULT FALSE
);

-- Tracks one sample setup lifecycle per workspace. The payload is owned by the
-- selected sample manager implementation.
CREATE TABLE sample_instance_setup (
    workspace text PRIMARY KEY REFERENCES workspace(resource_id),
    replica_id text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    activated_at timestamptz,
    expires_at timestamptz,
    deleted_at timestamptz,
    CHECK (expires_at IS NULL OR activated_at IS NOT NULL),
    CHECK (deleted_at IS NULL OR activated_at IS NOT NULL)
);

CREATE TABLE subscription (
    workspace   text        NOT NULL REFERENCES workspace(resource_id) PRIMARY KEY,
    -- Stored as SubscriptionPayload (proto/store/store/subscription.proto)
    payload     jsonb       NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE vcs_provider_user (
    workspace text NOT NULL REFERENCES workspace(resource_id),
    vcs_type text NOT NULL,
    user_id text NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    payload jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (workspace, vcs_type, user_id)
);

CREATE INDEX idx_vcs_provider_user_workspace_last_seen_at
    ON vcs_provider_user(workspace, last_seen_at DESC);

CREATE INDEX idx_vcs_provider_user_last_seen_at
    ON vcs_provider_user(last_seen_at);

CREATE TABLE principal (
    id serial PRIMARY KEY,
    deleted boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    name text NOT NULL,
    -- golbal unique
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text NOT NULL DEFAULT '',
    -- Stored as MFAConfig (proto/store/store/user.proto)
    mfa_config jsonb NOT NULL DEFAULT '{}',
    -- Stored as UserProfile (proto/store/store/user.proto)
    profile jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_principal_unique_email ON principal(email);

ALTER SEQUENCE principal_id_seq RESTART WITH 101;

-----------------------
-- Workspace-scoped tables
-----------------------

-- Setting
CREATE TABLE setting (
    -- name: SYSTEM, WORKSPACE_PROFILE, WORKSPACE_APPROVAL,
    -- APP_IM, AI, DATA_CLASSIFICATION, SEMANTIC_TYPES, ENVIRONMENT, EMAIL, MCP
    -- Enum: SettingName (proto/store/store/setting.proto)
    name text NOT NULL,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    -- Stored as JSON marshalled by protojson.Marshal (camelCase keys)
    value jsonb NOT NULL,
    PRIMARY KEY (workspace, name)
);

CREATE INDEX idx_setting_workspace ON setting(workspace);

-- Role
CREATE TABLE role (
    -- golbal unique
    resource_id text NOT NULL PRIMARY KEY,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    name text NOT NULL,
    description text NOT NULL,
    -- Stored as RolePermissions (proto/store/store/role.proto)
    permissions jsonb NOT NULL DEFAULT '{}',
    -- saved for future use
    payload jsonb NOT NULL DEFAULT '{}'
);

-- Policy
-- policy stores the policies for each resources.
CREATE TABLE policy (
    enforce boolean NOT NULL DEFAULT TRUE,
    updated_at timestamptz NOT NULL DEFAULT now(),
    workspace text NOT NULL REFERENCES workspace(resource_id),
    -- resource_type: WORKSPACE, ENVIRONMENT, PROJECT
    -- Enum: Policy.Resource (proto/store/store/policy.proto)
    resource_type text NOT NULL,
    -- resource: resource name in format like "environments/{environment}", "projects/{project}", etc.
    resource TEXT NOT NULL,
    -- type: ROLLOUT, MASKING_EXCEPTION, QUERY_DATA, MASKING_RULE, IAM, TAG
    -- Enum: Policy.Type (proto/store/store/policy.proto)
    type text NOT NULL,
    -- Stored as different types based on policy type (proto/store/store/policy.proto):
    -- ROLLOUT: RolloutPolicy
    -- MASKING_EXCEPTION: MaskingExceptionPolicy
    -- QUERY_DATA: QueryDataPolicy (includes query limits, export/copy restrictions, DDL/DML restrictions, admin data source restrictions)
    -- MASKING_RULE: MaskingRulePolicy
    -- IAM: IamPolicy
    -- TAG: TagPolicy
    payload jsonb NOT NULL DEFAULT '{}',
    inherit_from_parent boolean NOT NULL DEFAULT TRUE,
    PRIMARY KEY (workspace, resource_type, resource, type)
);

CREATE INDEX idx_policy_workspace ON policy(workspace);

-- idp stores generic identity provider.
CREATE TABLE idp (
    -- golbal unique
    resource_id text NOT NULL PRIMARY KEY,
    -- NULL for global IDPs (SaaS login), non-NULL for workspace-scoped IDPs.
    workspace text REFERENCES workspace(resource_id),
    name text NOT NULL,
    domain text NOT NULL,
    type text NOT NULL CONSTRAINT idp_type_check CHECK (type IN ('OAUTH2', 'OIDC', 'LDAP')),
    -- config stores the corresponding configuration of the IdP, which may vary depending on the type of the IdP.
    -- Stored as IdentityProviderConfig (proto/store/store/idp.proto)
    config jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE user_group (
    -- golbal unique
    id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    email text,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    -- Stored as GroupPayload (proto/store/store/group.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_user_group_unique_email ON user_group(workspace, email) WHERE email IS NOT NULL;

-- review config table.
CREATE TABLE review_config (
    -- golbal unique
    id text NOT NULL PRIMARY KEY,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    enabled boolean NOT NULL DEFAULT TRUE,
    name text NOT NULL,
    -- Stored as ReviewConfigPayload (proto/store/store/review_config.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE audit_log (
    -- golbal unique
    resource_id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Stored as AuditLog (proto/store/store/audit_log.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

-- Composite index for the most common query: filter by workspace, order/range by time.
CREATE INDEX idx_audit_log_workspace_created_at ON audit_log(workspace, created_at DESC);
-- JSONB indexes for filtering by specific fields within a workspace.
CREATE INDEX idx_audit_log_payload_parent ON audit_log((payload->>'parent'));
CREATE INDEX idx_audit_log_payload_method ON audit_log((payload->>'method'));
CREATE INDEX idx_audit_log_payload_resource ON audit_log((payload->>'resource'));
CREATE INDEX idx_audit_log_payload_user ON audit_log((payload->>'user'));

-----------------------
-- Project and project-scoped tables
-----------------------

CREATE TABLE project (
    -- golbal unique
    resource_id text NOT NULL PRIMARY KEY,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    deleted boolean NOT NULL DEFAULT FALSE,
    name text NOT NULL,
    -- Stored as Project (proto/store/store/project.proto)
    setting jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_project_workspace ON project(workspace);

-- service_account
-- Service Account needs both workspace and project
CREATE TABLE service_account (
    deleted boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    name text NOT NULL,
    -- golbal unique
    email text NOT NULL PRIMARY KEY,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    service_key_hash text NOT NULL,
    project text REFERENCES project(resource_id)
);

CREATE INDEX idx_service_account_project ON service_account(project) WHERE project IS NOT NULL;
CREATE UNIQUE INDEX idx_service_account_unique_workspace_email ON service_account(workspace, email);
CREATE INDEX idx_service_account_workspace ON service_account(workspace);

-- workload_identity
CREATE TABLE workload_identity (
    deleted boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    name text NOT NULL,
    -- golbal unique
    email text NOT NULL PRIMARY KEY,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    project text REFERENCES project(resource_id),
    -- Stored as WorkloadIdentityConfig (proto/store/store/user.proto)
    config jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_workload_identity_project ON workload_identity(project) WHERE project IS NOT NULL;
CREATE UNIQUE INDEX idx_workload_identity_unique_workspace_email ON workload_identity(workspace, email);
CREATE INDEX idx_workload_identity_workspace ON workload_identity(workspace);

-- Project Hook
CREATE TABLE project_webhook (
    -- golbal unique
    resource_id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project text NOT NULL REFERENCES project(resource_id),
    -- Stored as ProjectWebhook (proto/store/store/project_webhook.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_project_webhook_project ON project_webhook(project);

-- sheet_blob is content-addressed shared storage; nothing deletes from it and
-- every reference lives inside JSONB payloads (plan.config, task.payload,
-- release.payload, plan_check_run.result, revision.payload), invisible to
-- referential integrity. A GC written as "delete blobs no FK points at" would
-- empty the table; a correct GC must consult sheet_blob_ref and the JSONB
-- references.
CREATE TABLE sheet_blob (
    sha256 bytea NOT NULL PRIMARY KEY,
    content text NOT NULL
);

-- Records which projects may read a hash. Two projects that independently
-- author identical SQL share one blob and hold one ref row each. Written by
-- sheet creation, deleted by project purge, and enforced by the store's
-- project-scoped sheet accessors. A database transfer carries no refs:
-- change history follows
-- the database, statement content stays with the authoring project
-- (docs/design/sheet-history-on-database-transfer.md).
CREATE TABLE sheet_blob_ref (
    project text NOT NULL REFERENCES project(resource_id),
    sha256 bytea NOT NULL REFERENCES sheet_blob(sha256),
    PRIMARY KEY (project, sha256)
);

-- For the zero-ref audit and a future GC, which start from a hash with no
-- project in hand. Request-path queries lead with project and use the PK.
CREATE INDEX idx_sheet_blob_ref_sha256 ON sheet_blob_ref(sha256);

-- plan table stores the plan for a project
CREATE TABLE plan (
    -- unique and auto-increase per project
    id bigint NOT NULL,
    deleted boolean NOT NULL DEFAULT FALSE,
    creator text NOT NULL,
    -- The last actor to create or update the plan specs. Nullable for legacy plans.
    last_plan_editor text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    name text NOT NULL,
    description text NOT NULL,
    -- Stored as PlanConfig (proto/store/store/plan.proto)
    config jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (project, id)
);

CREATE INDEX idx_plan_project ON plan(project);
CREATE INDEX idx_plan_creator ON plan(creator);
CREATE INDEX idx_plan_config_has_rollout ON plan ((config->>'hasRollout'));

CREATE TABLE plan_check_run (
    -- unique and auto-increase per project
    id bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    plan_id bigint NOT NULL,
    status text NOT NULL CHECK (status IN ('AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    -- Stored as PlanCheckRunResult (proto/store/store/plan_check_run.proto)
    result jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (project, id),
    FOREIGN KEY (project, plan_id) REFERENCES plan(project, id)
);

CREATE UNIQUE INDEX idx_plan_check_run_unique_plan_id ON plan_check_run(project, plan_id);
CREATE INDEX idx_plan_check_run_active_status ON plan_check_run(status, id) WHERE status IN ('AVAILABLE', 'RUNNING');

-- Tracks webhook delivery for pipeline events (PIPELINE_FAILED or PIPELINE_COMPLETED).
-- One row per plan at any time - mutually exclusive events.
-- Row is deleted when user clicks BatchRunTasks to reset notification state.
CREATE TABLE plan_webhook_delivery (
    project TEXT NOT NULL REFERENCES project(resource_id),
    plan_id BIGINT NOT NULL,
    -- Event type: 'PIPELINE_FAILED' or 'PIPELINE_COMPLETED'
    event_type TEXT NOT NULL,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project, plan_id),
    FOREIGN KEY (project, plan_id) REFERENCES plan(project, id)
);

-- issue
CREATE TABLE issue (
    -- unique and auto-increase per project
    id bigint NOT NULL,
    creator text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    plan_id bigint,
    name text NOT NULL,
    status text NOT NULL CHECK (status IN ('OPEN', 'DONE', 'CANCELED')),
    -- type: DATABASE_CHANGE, ROLE_GRANT, ACCESS_GRANT
    -- Enum: Issue.Type (proto/store/store/issue.proto)
    type text NOT NULL,
    description text NOT NULL DEFAULT '',
    -- Stored as Issue (proto/store/store/issue.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    ts_vector tsvector,
    PRIMARY KEY (project, id),
    FOREIGN KEY (project, plan_id) REFERENCES plan(project, id)
);

CREATE INDEX idx_issue_project ON issue(project);
CREATE UNIQUE INDEX idx_issue_unique_plan_id ON issue(project, plan_id);
CREATE INDEX idx_issue_creator ON issue(creator);
CREATE INDEX idx_issue_ts_vector ON issue USING GIN(ts_vector);

CREATE TABLE issue_comment (
    -- global unique
    resource_id text NOT NULL DEFAULT gen_random_uuid()::text,
    creator text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    issue_id integer NOT NULL,
    -- Stored as IssueCommentPayload (proto/store/store/issue_comment.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    -- The root comment of this reply's thread; NULL on root comments and
    -- events. A reply references the root directly, never another reply.
    parent_id text REFERENCES issue_comment(resource_id),
    -- OPEN/RESOLVED on thread roots, the comments with a statement anchor;
    -- NULL on plain comments, replies, and events.
    thread_state text CHECK (thread_state IN ('OPEN', 'RESOLVED')),
    PRIMARY KEY (resource_id),
    FOREIGN KEY (project, issue_id) REFERENCES issue(project, id)
);

CREATE INDEX idx_issue_comment_issue_id ON issue_comment(project, issue_id);
CREATE UNIQUE INDEX idx_issue_comment_unique_resource_id ON issue_comment(resource_id);
CREATE INDEX idx_issue_comment_parent_id ON issue_comment(parent_id)
    WHERE parent_id IS NOT NULL;
CREATE INDEX idx_issue_comment_open_thread ON issue_comment(project, issue_id)
    WHERE thread_state = 'OPEN';

-- SQL Review V2 run status slot: one row per (issue, reviewer type), reset in
-- place on re-run (created_at = now() on reset — the row is the current run,
-- not the slot's history). Results live in issue comments; the run carries
-- none. No standalone id on purpose: nothing may durably reference a run.
CREATE TABLE review_run (
    project text NOT NULL REFERENCES project(resource_id),
    issue_id bigint NOT NULL,
    -- Reviewer type: 'RULE' (standard rules) or 'GUIDELINE' (natural-language
    -- guidelines, performed by AI). No CHECK on purpose: the reviewer-id space
    -- is open.
    type text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- Attempt number of the current run, 0-based like task_run.attempt.
    -- Bumped on every slot reset (issue created / SQL updated / manual
    -- re-run); it counts triggers, not completed executions. The completion
    -- transaction is fenced on it, so a superseded execution posts zero
    -- comments.
    attempt integer NOT NULL DEFAULT 0,
    status text NOT NULL CHECK (status IN ('AVAILABLE', 'RUNNING', 'DONE', 'FAILED')),
    replica_id text,
    -- Stored as ReviewRunPayload (proto/store/store/review_run.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (project, issue_id, type),
    FOREIGN KEY (project, issue_id) REFERENCES issue(project, id)
);

-- Most rows are terminal; schedulers scan only active rows
-- (cf. idx_task_run_active_status_id).
CREATE INDEX idx_review_run_active_status ON review_run(status)
    WHERE status IN ('AVAILABLE', 'RUNNING');

-- For the heartbeat reaper's dead-replica lookup
-- (cf. idx_task_run_running_replica).
CREATE INDEX idx_review_run_running_replica ON review_run(replica_id)
    WHERE status = 'RUNNING' AND replica_id IS NOT NULL;

-- saved_query table stores SQL Editor saved queries.
CREATE TABLE saved_query (
    -- global unique
    resource_id text NOT NULL DEFAULT gen_random_uuid()::text,
    creator text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    name text NOT NULL,
    statement text NOT NULL,
    -- The folder path this saved query lives in ("a/b/c", '' = unfiled),
    -- written under the ordinary update permission like any other field. A
    -- folder is a path on rows, so empty folders cannot exist.
    folder text NOT NULL DEFAULT '',
    -- Stored as SavedQueryPayload (proto/store/store/saved_query.proto); the
    -- connected database is a soft reference kept as its canonical name.
    payload jsonb NOT NULL DEFAULT '{}',
    -- Per-object grants: a protojson array of SavedQueryBinding at the jsonb
    -- root, e.g. [{"level":"EDITOR","members":["groups/eng@corp.com"]}]. The
    -- array must sit at the root so the access queries' `@>` probes use the
    -- GIN index below. Empty means private to the creator.
    bindings jsonb NOT NULL DEFAULT '[]',
    PRIMARY KEY (resource_id)
);

-- Serves the folder tree and the rows inside a folder: DISTINCT folder over a
-- project (admin) or a project + creator (the SQL Editor's own tree) both run
-- as index-only scans, and the project prefix still answers a project lookup
-- on its own.
CREATE INDEX idx_saved_query_project_creator_folder ON saved_query(project, creator, folder);

-- The governance ListSavedQueries recency pull (creator filter +
-- order_by "update_time desc") reads this index in order with no sort; it
-- also covers every other creator-led scan via its prefix.
CREATE INDEX idx_saved_query_creator_updated_at_resource_id ON saved_query(creator, updated_at DESC, resource_id DESC);

-- "Shared with me" probes bindings once per principal in the caller's set and
-- BitmapOrs the results. jsonb_path_ops is smaller and faster than the default
-- opclass for the containment these queries do.
CREATE INDEX idx_saved_query_bindings ON saved_query USING gin (bindings jsonb_path_ops);

-- saved_query_star stores per-user stars: row existence is the star. The FK
-- cascade is a race backstop only — code paths delete star rows explicitly
-- before their parent.
CREATE TABLE saved_query_star (
    saved_query text NOT NULL REFERENCES saved_query(resource_id) ON DELETE CASCADE,
    principal text NOT NULL,
    PRIMARY KEY (saved_query, principal)
);

CREATE INDEX idx_saved_query_star_principal ON saved_query_star(principal);

CREATE TABLE db_group (
    project text NOT NULL REFERENCES project(resource_id),
    -- project-level unique
    resource_id text NOT NULL,
    name text NOT NULL DEFAULT '',
    -- Stored as google.type.Expr (from Google Common Expression Language)
    expression jsonb NOT NULL DEFAULT '{}',
    -- Stored as DatabaseGroupPayload (proto/store/store/db_group.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (project, resource_id)
);

CREATE TABLE release (
    project text NOT NULL REFERENCES project(resource_id),
    train text NOT NULL DEFAULT '',
    iteration integer NOT NULL DEFAULT 0,
    deleted boolean NOT NULL DEFAULT FALSE,
    release_id text NOT NULL DEFAULT '',
    creator text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    category text NOT NULL DEFAULT '',
    -- Stored as ReleasePayload (proto/store/store/release.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (project, train, iteration)
);

CREATE INDEX idx_release_project ON release(project);
CREATE INDEX idx_release_project_release_id ON release(project, release_id);
CREATE INDEX idx_release_category ON release(project, category);

CREATE TABLE access_grant (
    -- global unique
    id text PRIMARY KEY,
    project text NOT NULL REFERENCES project(resource_id),
    creator text NOT NULL,
    status text NOT NULL DEFAULT 'PENDING',
    expire_time timestamptz,
    -- Stored as AccessGrantPayload (proto/store/store/access_grant.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_access_grant_project_creator_expire_time ON access_grant(project, creator, expire_time);

CREATE TABLE query_history (
    -- global unique
    resource_id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    creator text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    database text NOT NULL, -- the database resource name, for example, instances/{instance}/databases/{database}
    statement text NOT NULL,
    -- type: QUERY, EXPORT
    type text NOT NULL,
    -- saved for details, like error, duration, etc.
    -- Stored as QueryHistoryPayload (proto/store/store/query_history.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_query_history_creator_created_at_project ON query_history(creator, created_at, project DESC);

CREATE INDEX idx_query_history_project_created_at ON query_history(project, created_at DESC);

CREATE INDEX idx_query_history_created_at ON query_history(created_at DESC);

-----------------------
-- Instance and instance-scoped tables
-----------------------

CREATE TABLE instance (
    -- global unique
    resource_id text NOT NULL PRIMARY KEY,
    workspace text NOT NULL REFERENCES workspace(resource_id),
    -- NULL for workspace instances; set for project instances.
    project text REFERENCES project(resource_id),
    deleted boolean NOT NULL DEFAULT FALSE,
    environment text,
    -- Stored as Instance (proto/store/store/instance.proto)
    metadata jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_instance_workspace ON instance(workspace);
CREATE INDEX idx_instance_project ON instance(project) WHERE project IS NOT NULL;
CREATE INDEX idx_instance_metadata_engine ON instance((metadata->>'engine'));

-- db stores the databases for a particular instance
-- data is synced periodically from the instance
CREATE TABLE db (
    instance text NOT NULL REFERENCES instance(resource_id),
    name text NOT NULL,
    deleted boolean NOT NULL DEFAULT FALSE,
    project text NOT NULL REFERENCES project(resource_id),
    environment text,
    -- Stored as DatabaseMetadata (proto/store/store/database.proto)
    metadata jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (instance, name)
);

CREATE INDEX idx_db_project ON db(project);

-- db_schema stores the database schema metadata for a particular database.
CREATE TABLE db_schema (
    instance text NOT NULL,
    db_name text NOT NULL,
    -- Stored as DatabaseSchemaMetadata (proto/store/store/database.proto)
    metadata json NOT NULL DEFAULT '{}',
    raw_dump text NOT NULL DEFAULT '',
    -- Stored as DatabaseConfig (proto/store/store/database.proto)
    config jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (instance, db_name),
    CONSTRAINT db_schema_instance_db_name_fkey FOREIGN KEY(instance, db_name) REFERENCES db(instance, name)
);

CREATE TABLE revision (
    -- global unique
    resource_id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    instance text NOT NULL,
    db_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleter text,
    deleted_at timestamptz,
    version text NOT NULL,
    -- Stored as RevisionPayload (proto/store/store/revision.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    CONSTRAINT revision_instance_db_name_fkey FOREIGN KEY(instance, db_name) REFERENCES db(instance, name)
);

CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_type_version_deleted_at_null ON revision(instance, db_name, (payload->>'type'), version) WHERE deleted_at IS NULL;
CREATE INDEX idx_revision_instance_db_name_type_version ON revision(instance, db_name, (payload->>'type'), version);

CREATE TABLE sync_history (
    -- global unique
    resource_id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    created_at timestamptz NOT NULL DEFAULT now(),
    instance text NOT NULL,
    db_name text NOT NULL,
    -- Stored as DatabaseSchemaMetadata (proto/store/store/database.proto)
    metadata json NOT NULL DEFAULT '{}',
    raw_dump text NOT NULL DEFAULT '',
    CONSTRAINT sync_history_instance_db_name_fkey FOREIGN KEY(instance, db_name) REFERENCES db(instance, name)
);

CREATE INDEX idx_sync_history_instance_db_name_created_at ON sync_history (instance, db_name, created_at);

CREATE TABLE changelog (
    -- global unique
    resource_id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    created_at timestamptz NOT NULL DEFAULT now(),
    instance text NOT NULL,
    db_name text NOT NULL,
    status text NOT NULL CONSTRAINT changelog_status_check CHECK (status IN ('PENDING', 'DONE', 'FAILED')),
    sync_history text REFERENCES sync_history(resource_id),
    -- Stored as ChangelogPayload (proto/store/store/changelog.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    CONSTRAINT changelog_instance_db_name_fkey FOREIGN KEY(instance, db_name) REFERENCES db(instance, name)
);

CREATE INDEX idx_changelog_instance_db_name ON changelog (instance, db_name);

-- instance change history records the changes an instance and its databases.
CREATE TABLE instance_change_history (
    id bigserial PRIMARY KEY,
    version text NOT NULL
);

CREATE UNIQUE INDEX idx_instance_change_history_unique_version ON instance_change_history (version);

ALTER SEQUENCE instance_change_history_id_seq RESTART WITH 101;

-----------------------
-- Pipeline (cross project + instance)
-----------------------

-- task table stores the task for a plan
CREATE TABLE task (
    -- unique and auto-increase per project
    id bigint NOT NULL,
    project text NOT NULL REFERENCES project(resource_id),
    plan_id bigint NOT NULL,
    instance text NOT NULL REFERENCES instance(resource_id),
    environment text,
    db_name text,
    type text NOT NULL,
    -- Stored as Task (proto/store/store/task.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (project, id),
    FOREIGN KEY (project, plan_id) REFERENCES plan(project, id)
);

CREATE INDEX idx_task_plan_id_environment ON task(project, plan_id, environment);

-- task run table stores the task run
CREATE TABLE task_run (
    -- unique and auto-increase per project
    id bigint NOT NULL,
    creator text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    task_id integer NOT NULL,
    attempt integer NOT NULL,
    status text NOT NULL CHECK (status IN ('PENDING', 'AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    started_at timestamptz NULL,
    run_at timestamptz,
    -- result saves the task run result in json format
    -- Stored as TaskRunResult (proto/store/store/task_run.proto)
    result jsonb NOT NULL DEFAULT '{}',
    replica_id TEXT,
    -- Stored as TaskRunPayload (proto/store/store/task_run.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (project, id),
    FOREIGN KEY (project, task_id) REFERENCES task(project, id)
);

CREATE INDEX idx_task_run_task_id ON task_run(task_id);
CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON task_run(project, task_id, attempt);
-- Partial index for active task runs. Most task runs are in terminal states (DONE, FAILED, CANCELED)
-- that never change. Queries frequently filter for active statuses (PENDING, RUNNING), so a partial
-- index is more efficient than a full index on status - smaller size, faster maintenance, better cache efficiency.
CREATE INDEX idx_task_run_active_status_id ON task_run(status, id) WHERE status IN ('PENDING', 'AVAILABLE', 'RUNNING');
CREATE INDEX idx_task_run_running_replica ON task_run(replica_id) WHERE status = 'RUNNING' AND replica_id IS NOT NULL;

-- replica_heartbeat tracks active replicas in HA deployments.
-- Used to detect and clean up stale RUNNING task runs from crashed replicas.
CREATE TABLE replica_heartbeat (
    replica_id TEXT PRIMARY KEY,
    last_heartbeat TIMESTAMPTZ NOT NULL
);

-- Append-only log with no primary key on purpose: entries for one task run can
-- legitimately share a created_at microsecond (BYT-10035).
CREATE TABLE task_run_log (
    project text NOT NULL REFERENCES project(resource_id),
    task_run_id integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Stored as TaskRunLog (proto/store/store/task_run_log.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    FOREIGN KEY (project, task_run_id) REFERENCES task_run(project, id)
);

CREATE INDEX idx_task_run_log_project_task_run_id_created_at ON task_run_log(project, task_run_id, created_at);

-----------------------
-- OAuth2 and auth
-----------------------

CREATE TABLE oauth2_client (
    client_id text PRIMARY KEY,
    -- workspace is nullable: clients registered via unauthenticated DCR are
    -- workspace-agnostic and get bound to a workspace at consent time on the
    -- issued authorization code / refresh token.
    workspace text REFERENCES workspace(resource_id),
    client_secret_hash text NOT NULL,
    config jsonb NOT NULL,
    last_active_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE oauth2_authorization_code (
    code text PRIMARY KEY,
    client_id text NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_email text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    -- Workspace selected at consent time. Carried through into the issued
    -- access token's workspace_id claim.
    workspace text REFERENCES workspace(resource_id),
    config jsonb NOT NULL,
    expires_at timestamptz NOT NULL,
    -- When the row was issued.
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE oauth2_refresh_token (
    token_hash text PRIMARY KEY,
    client_id text NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_email text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    -- Workspace inherited from the authorization code that originally issued
    -- this refresh token; preserved across refresh.
    workspace text REFERENCES workspace(resource_id),
    -- Stored as OAuth2RefreshTokenConfig (proto/store/store/oauth2.proto): the
    -- consented resource and scope, inherited from the authorization code and
    -- carried forward unchanged by every refresh.
    config jsonb NOT NULL DEFAULT '{}',
    expires_at timestamptz NOT NULL,
    -- When the row was issued.
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_oauth2_authorization_code_expires_at ON oauth2_authorization_code(expires_at);
CREATE INDEX idx_oauth2_refresh_token_expires_at ON oauth2_refresh_token(expires_at);
-- Referencing columns of the two foreign keys these tables carry: PostgreSQL
-- indexes only the referenced side, so without these the ON UPDATE CASCADE
-- from principal(email) and the ON DELETE CASCADE from oauth2_client are
-- sequential scans.
CREATE INDEX idx_oauth2_authorization_code_user_email ON oauth2_authorization_code(user_email);
CREATE INDEX idx_oauth2_refresh_token_user_email ON oauth2_refresh_token(user_email);
CREATE INDEX idx_oauth2_authorization_code_client_id ON oauth2_authorization_code(client_id);
CREATE INDEX idx_oauth2_refresh_token_client_id ON oauth2_refresh_token(client_id);
CREATE INDEX idx_oauth2_client_last_active_at ON oauth2_client(last_active_at);
CREATE INDEX idx_oauth2_client_workspace ON oauth2_client(workspace);

-- Web refresh tokens for session management
CREATE TABLE web_refresh_token (
    token_hash  TEXT PRIMARY KEY,
    user_email  TEXT NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    -- When the row was issued.
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_web_refresh_token_user_email ON web_refresh_token(user_email);
CREATE INDEX idx_web_refresh_token_expires_at ON web_refresh_token(expires_at);

CREATE TABLE email_verification_code (
    email         text NOT NULL,
    -- Stored as EmailVerificationCodePurpose enum name (proto/store/store/auth.proto)
    purpose       text NOT NULL,
    code_hash     text NOT NULL,
    expires_at    timestamptz NOT NULL,
    last_sent_at  timestamptz NOT NULL,
    PRIMARY KEY (email, purpose)
);

CREATE INDEX idx_email_verification_code_expires_at ON email_verification_code (expires_at);

-- Attempt limits for guessable login credentials (docs/design/login-attempt-lockout.md).
-- One row per (identity, kind): attempts since the last success, and when the latest was.
CREATE TABLE login_attempt (
    -- The identity under attack, server-resolved and globally unique: the normalized
    -- email, or the identity-provider ID joined with the submitted username for LDAP.
    -- Not a FK, so unknown identities count too (no existence oracle).
    identity        text NOT NULL,
    -- Stored as LoginAttemptKind enum name (proto/store/store/auth.proto):
    -- PASSWORD | EMAIL_CODE | MFA.
    kind            text NOT NULL,
    attempts        int NOT NULL,
    last_attempt_at timestamptz NOT NULL,
    PRIMARY KEY (identity, kind)
);

CREATE INDEX idx_login_attempt_last_attempt_at ON login_attempt (last_attempt_at);

-----------------------
-- Seed data
-----------------------

-- Global server config (auth secret only).
-- Workspace and its settings/policies/project are created by the Go signup flow (store.CreateWorkspace).
DO $$
DECLARE
  auth_secret text;
BEGIN
  SELECT string_agg(substr('0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ', floor(random() * 62 + 1)::int, 1), '')
    INTO auth_secret
    FROM generate_series(1, 32);

  INSERT INTO server_config (payload) VALUES (
    json_build_object('authSecret', auth_secret)
  );
END $$;
