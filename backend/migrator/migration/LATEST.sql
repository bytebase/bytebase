-- idp stores generic identity provider.
CREATE TABLE idp (
  id serial PRIMARY KEY,
  resource_id text NOT NULL,
  name text NOT NULL,
  domain text NOT NULL,
  type text NOT NULL CONSTRAINT idp_type_check CHECK (type IN ('OAUTH2', 'OIDC', 'LDAP')),
  -- config stores the corresponding configuration of the IdP, which may vary depending on the type of the IdP.
  -- Stored as IdentityProviderConfig (proto/store/store/idp.proto)
  config jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON idp(resource_id);

ALTER SEQUENCE idp_id_seq RESTART WITH 101;

-- principal
CREATE TABLE principal (
    id serial PRIMARY KEY,
    deleted boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    type text NOT NULL CHECK (type IN ('END_USER', 'SYSTEM_BOT', 'SERVICE_ACCOUNT', 'WORKLOAD_IDENTITY')),
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text NOT NULL DEFAULT '',
    -- Stored as MFAConfig (proto/store/store/user.proto)
    mfa_config jsonb NOT NULL DEFAULT '{}',
    -- Stored as UserProfile (proto/store/store/user.proto)
    profile jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_principal_unique_email ON principal(email);

-- Setting
CREATE TABLE setting (
    id serial PRIMARY KEY,
    -- name: SYSTEM, WORKSPACE_PROFILE, WORKSPACE_APPROVAL,
    -- APP_IM, AI, DATA_CLASSIFICATION, SEMANTIC_TYPES, ENVIRONMENT
    -- Enum: SettingName (proto/store/store/setting.proto)
    name text NOT NULL,
    -- Stored as JSON marshalled by protojson.Marshal (camelCase keys)
    value jsonb NOT NULL
);

CREATE UNIQUE INDEX idx_setting_unique_name ON setting(name);

ALTER SEQUENCE setting_id_seq RESTART WITH 101;

-- Role
CREATE TABLE role (
    id bigserial PRIMARY KEY,
    resource_id text NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    -- Stored as RolePermissions (proto/store/store/role.proto)
    permissions jsonb NOT NULL DEFAULT '{}',
    -- saved for future use
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_role_unique_resource_id on role (resource_id);

ALTER SEQUENCE role_id_seq RESTART WITH 101;

-- Policy
-- policy stores the policies for each resources.
CREATE TABLE policy (
    id serial PRIMARY KEY,
    enforce boolean NOT NULL DEFAULT TRUE,
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- resource_type: WORKSPACE, ENVIRONMENT, PROJECT
    -- Enum: Policy.Resource (proto/store/store/policy.proto)
    resource_type text NOT NULL,
    -- resource: resource name in format like "environments/{environment}", "projects/{project}", etc.
    resource TEXT NOT NULL,
    -- type: ROLLOUT, MASKING_EXCEPTION, QUERY_DATA, MASKING_RULE, IAM, TAG, DATA_SOURCE_QUERY
    -- Enum: Policy.Type (proto/store/store/policy.proto)
    type text NOT NULL,
    -- Stored as different types based on policy type (proto/store/store/policy.proto):
    -- ROLLOUT: RolloutPolicy
    -- MASKING_EXCEPTION: MaskingExceptionPolicy
    -- QUERY_DATA: QueryDataPolicy
    -- MASKING_RULE: MaskingRulePolicy
    -- IAM: IamPolicy
    -- TAG: TagPolicy
    -- DATA_SOURCE_QUERY: DataSourceQueryPolicy
    payload jsonb NOT NULL DEFAULT '{}',
    inherit_from_parent boolean NOT NULL DEFAULT TRUE
);

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_type ON policy(resource_type, resource, type);

ALTER SEQUENCE policy_id_seq RESTART WITH 101;

-- Project
CREATE TABLE project (
    id serial PRIMARY KEY,
    deleted boolean NOT NULL DEFAULT FALSE,
    name text NOT NULL,
    resource_id text NOT NULL,
    data_classification_config_id text NOT NULL DEFAULT '',
    -- Stored as Project (proto/store/store/project.proto)
    setting jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_project_unique_resource_id ON project(resource_id);

-- Project Hook
CREATE TABLE project_webhook (
    id serial PRIMARY KEY,
    project text NOT NULL REFERENCES project(resource_id),
    -- Stored as ProjectWebhook (proto/store/store/project_webhook.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_project_webhook_project ON project_webhook(project);

ALTER SEQUENCE project_webhook_id_seq RESTART WITH 101;

-- Instance
CREATE TABLE instance (
    id serial PRIMARY KEY,
    deleted boolean NOT NULL DEFAULT FALSE,
    environment text,
    resource_id text NOT NULL,
    -- Stored as Instance (proto/store/store/instance.proto)
    metadata jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON instance(resource_id);

CREATE INDEX idx_instance_metadata_engine ON instance((metadata->>'engine'));

ALTER SEQUENCE instance_id_seq RESTART WITH 101;

-- db stores the databases for a particular instance
-- data is synced periodically from the instance
CREATE TABLE db (
    id serial PRIMARY KEY,
    deleted boolean NOT NULL DEFAULT FALSE,
    project text NOT NULL REFERENCES project(resource_id),
    instance text NOT NULL REFERENCES instance(resource_id),
    name text NOT NULL,
    environment text,
    -- Stored as DatabaseMetadata (proto/store/store/database.proto)
    metadata jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_db_project ON db(project);

CREATE UNIQUE INDEX idx_db_unique_instance_name ON db(instance, name);

ALTER SEQUENCE db_id_seq RESTART WITH 101;

-- db_schema stores the database schema metadata for a particular database.
CREATE TABLE db_schema (
    id serial PRIMARY KEY,
    instance text NOT NULL,
    db_name text NOT NULL,
    -- Stored as DatabaseSchemaMetadata (proto/store/store/database.proto)
    metadata json NOT NULL DEFAULT '{}',
    raw_dump text NOT NULL DEFAULT '',
    -- Stored as DatabaseConfig (proto/store/store/database.proto)
    config jsonb NOT NULL DEFAULT '{}',
    CONSTRAINT db_schema_instance_db_name_fkey FOREIGN KEY(instance, db_name) REFERENCES db(instance, name)
);

CREATE UNIQUE INDEX idx_db_schema_unique_instance_db_name ON db_schema(instance, db_name);

ALTER SEQUENCE db_schema_id_seq RESTART WITH 101;

CREATE TABLE sheet_blob (
	sha256 bytea NOT NULL PRIMARY KEY,
	content text NOT NULL
);

-- sheet table stores general statements.
CREATE TABLE sheet (
    id serial PRIMARY KEY,
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    name text NOT NULL,
    sha256 bytea NOT NULL,
    -- Stored as SheetPayload (proto/store/store/sheet.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_sheet_project ON sheet(project);

ALTER SEQUENCE sheet_id_seq RESTART WITH 101;

-- plan table stores the plan for a project
CREATE TABLE plan (
    id bigserial PRIMARY KEY,
    deleted boolean NOT NULL DEFAULT FALSE,
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    name text NOT NULL,
    description text NOT NULL,
    -- Stored as PlanConfig (proto/store/store/plan.proto)
    config jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_plan_project ON plan(project);
CREATE INDEX idx_plan_config_has_rollout ON plan ((config->>'hasRollout'));

ALTER SEQUENCE plan_id_seq RESTART WITH 101;

CREATE TABLE plan_check_run (
    id serial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    plan_id bigint NOT NULL REFERENCES plan(id),
    status text NOT NULL CHECK (status IN ('AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    -- Stored as PlanCheckRunResult (proto/store/store/plan_check_run.proto)
    result jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_plan_check_run_unique_plan_id ON plan_check_run(plan_id);

CREATE INDEX idx_plan_check_run_active_status ON plan_check_run(status, id) WHERE status IN ('AVAILABLE', 'RUNNING');

ALTER SEQUENCE plan_check_run_id_seq RESTART WITH 101;

-- Tracks webhook delivery for pipeline events (PIPELINE_FAILED or PIPELINE_COMPLETED).
-- One row per plan at any time - mutually exclusive events.
-- Row is deleted when user clicks BatchRunTasks to reset notification state.
CREATE TABLE plan_webhook_delivery (
    plan_id BIGINT PRIMARY KEY REFERENCES plan(id),
    -- Event type: 'PIPELINE_FAILED' or 'PIPELINE_COMPLETED'
    event_type TEXT NOT NULL,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- task table stores the task for a plan
CREATE TABLE task (
    id serial PRIMARY KEY,
    plan_id bigint NOT NULL REFERENCES plan(id),
    instance text NOT NULL REFERENCES instance(resource_id),
    environment text,
    db_name text,
    type text NOT NULL,
    -- Stored as Task (proto/store/store/task.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_task_plan_id_environment ON task(plan_id, environment);

ALTER SEQUENCE task_id_seq RESTART WITH 101;

-- task run table stores the task run
CREATE TABLE task_run (
    id serial PRIMARY KEY,
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    task_id integer NOT NULL REFERENCES task(id),
    attempt integer NOT NULL,
    status text NOT NULL CHECK (status IN ('PENDING', 'AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    started_at timestamptz NULL,
    run_at timestamptz,
    -- result saves the task run result in json format
    -- Stored as TaskRunResult (proto/store/store/task_run.proto)
    result jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_task_run_task_id ON task_run(task_id);

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON task_run (task_id, attempt);

-- Partial index for active task runs. Most task runs are in terminal states (DONE, FAILED, CANCELED)
-- that never change. Queries frequently filter for active statuses (PENDING, RUNNING), so a partial
-- index is more efficient than a full index on status - smaller size, faster maintenance, better cache efficiency.
CREATE INDEX idx_task_run_active_status_id ON task_run(status, id) WHERE status IN ('PENDING', 'AVAILABLE', 'RUNNING');

ALTER SEQUENCE task_run_id_seq RESTART WITH 101;

CREATE TABLE task_run_log (
    id bigserial PRIMARY KEY,
    task_run_id integer NOT NULL REFERENCES task_run(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Stored as TaskRunLog (proto/store/store/task_run_log.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_task_run_log_task_run_id ON task_run_log(task_run_id);

ALTER SEQUENCE task_run_log_id_seq RESTART WITH 101;

-- Pipeline related END
-----------------------
-- issue
CREATE TABLE issue (
    id serial PRIMARY KEY,
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    plan_id bigint REFERENCES plan(id),
    name text NOT NULL,
    status text NOT NULL CHECK (status IN ('OPEN', 'DONE', 'CANCELED')),
    -- type: DATABASE_CHANGE, GRANT_REQUEST, DATABASE_EXPORT
    -- Enum: Issue.Type (proto/store/store/issue.proto)
    type text NOT NULL,
    description text NOT NULL DEFAULT '',
    -- Stored as Issue (proto/store/store/issue.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    ts_vector tsvector
);

CREATE INDEX idx_issue_project ON issue(project);

CREATE UNIQUE INDEX idx_issue_unique_plan_id ON issue(plan_id);

CREATE INDEX idx_issue_creator ON issue(creator);

CREATE INDEX idx_issue_ts_vector ON issue USING GIN(ts_vector);

ALTER SEQUENCE issue_id_seq RESTART WITH 101;

-- instance change history records the changes an instance and its databases.
CREATE TABLE instance_change_history (
    id bigserial PRIMARY KEY,
    version text NOT NULL
);

CREATE UNIQUE INDEX idx_instance_change_history_unique_version ON instance_change_history (version);

ALTER SEQUENCE instance_change_history_id_seq RESTART WITH 101;

CREATE TABLE audit_log (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Stored as AuditLog (proto/store/store/audit_log.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

CREATE INDEX idx_audit_log_payload_parent ON audit_log((payload->>'parent'));

CREATE INDEX idx_audit_log_payload_method ON audit_log((payload->>'method'));

CREATE INDEX idx_audit_log_payload_resource ON audit_log((payload->>'resource'));

CREATE INDEX idx_audit_log_payload_user ON audit_log((payload->>'user'));

ALTER SEQUENCE audit_log_id_seq RESTART WITH 101;

CREATE TABLE issue_comment (
    id bigserial PRIMARY KEY,
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    issue_id integer NOT NULL REFERENCES issue(id),
    -- Stored as IssueCommentPayload (proto/store/store/issue_comment.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_issue_comment_issue_id ON issue_comment(issue_id);

ALTER SEQUENCE issue_comment_id_seq RESTART WITH 101;

CREATE TABLE query_history (
    id bigserial PRIMARY KEY,
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    project_id text NOT NULL, -- the project resource id
    database text NOT NULL, -- the database resource name, for example, instances/{instance}/databases/{database}
    statement text NOT NULL,
    -- type: QUERY, EXPORT
    type text NOT NULL,
    -- saved for details, like error, duration, etc.
    -- Stored as QueryHistoryPayload (proto/store/store/query_history.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_query_history_creator_created_at_project_id ON query_history(creator, created_at, project_id DESC);

ALTER SEQUENCE query_history_id_seq RESTART WITH 101;

-- worksheet table stores worksheets in SQL Editor.
CREATE TABLE worksheet (
    id serial PRIMARY KEY,
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    instance text,
    db_name text,
    name text NOT NULL,
    statement text NOT NULL,
    -- visibility: PROJECT_READ, PROJECT_WRITE, PRIVATE
    -- Enum: Worksheet.Visibility (proto/v1/v1/worksheet_service.proto)
    visibility text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_worksheet_creator_project ON worksheet(creator, project);

ALTER SEQUENCE worksheet_id_seq RESTART WITH 101;

-- worksheet_organizer table stores the sheet status for a principal.
CREATE TABLE worksheet_organizer (
    id serial PRIMARY KEY,
    worksheet_id integer NOT NULL REFERENCES worksheet(id) ON DELETE CASCADE,
    principal text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal ON worksheet_organizer(worksheet_id, principal);

CREATE INDEX idx_worksheet_organizer_principal ON worksheet_organizer(principal);

CREATE INDEX idx_worksheet_organizer_payload ON worksheet_organizer USING GIN(payload);

CREATE TABLE db_group (
    id bigserial PRIMARY KEY,
    project text NOT NULL REFERENCES project(resource_id),
    resource_id text NOT NULL,
    name text NOT NULL DEFAULT '',
    -- Stored as google.type.Expr (from Google Common Expression Language)
    expression jsonb NOT NULL DEFAULT '{}',
    -- Stored as DatabaseGroupPayload (proto/store/store/db_group.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_db_group_unique_project_resource_id ON db_group(project, resource_id);

ALTER SEQUENCE db_group_id_seq RESTART WITH 101;

CREATE TABLE export_archive (
  id serial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  bytes bytea,
  -- Stored as ExportArchivePayload (proto/store/store/export_archive.proto)
  payload jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE user_group (
  id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
  email text,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  -- Stored as GroupPayload (proto/store/store/group.proto)
  payload jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_user_group_unique_email ON user_group(email) WHERE email IS NOT NULL;

-- review config table.
CREATE TABLE review_config (
    id text NOT NULL PRIMARY KEY,
    enabled boolean NOT NULL DEFAULT TRUE,
    name text NOT NULL,
    -- Stored as ReviewConfigPayload (proto/store/store/review_config.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE revision (
    id bigserial PRIMARY KEY,
    instance text NOT NULL,
    db_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleter text REFERENCES principal(email) ON UPDATE CASCADE,
    deleted_at timestamptz,
    version text NOT NULL,
    -- Stored as RevisionPayload (proto/store/store/revision.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    CONSTRAINT revision_instance_db_name_fkey FOREIGN KEY(instance, db_name) REFERENCES db(instance, name)
);

ALTER SEQUENCE revision_id_seq RESTART WITH 101;

CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_type_version_deleted_at_null ON revision(instance, db_name, (payload->>'type'), version) WHERE deleted_at IS NULL;

CREATE INDEX idx_revision_instance_db_name_type_version ON revision(instance, db_name, (payload->>'type'), version);

CREATE TABLE sync_history (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    instance text NOT NULL,
    db_name text NOT NULL,
    -- Stored as DatabaseSchemaMetadata (proto/store/store/database.proto)
    metadata json NOT NULL DEFAULT '{}',
    raw_dump text NOT NULL DEFAULT '',
    CONSTRAINT sync_history_instance_db_name_fkey FOREIGN KEY(instance, db_name) REFERENCES db(instance, name)
);

ALTER SEQUENCE sync_history_id_seq RESTART WITH 101;

CREATE INDEX idx_sync_history_instance_db_name_created_at ON sync_history (instance, db_name, created_at);

CREATE TABLE changelog (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    instance text NOT NULL,
    db_name text NOT NULL,
    status text NOT NULL CONSTRAINT changelog_status_check CHECK (status IN ('PENDING', 'DONE', 'FAILED')),
    sync_history_id bigint REFERENCES sync_history(id),
    -- Stored as ChangelogPayload (proto/store/store/changelog.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    CONSTRAINT changelog_instance_db_name_fkey FOREIGN KEY(instance, db_name) REFERENCES db(instance, name)
);

ALTER SEQUENCE changelog_id_seq RESTART WITH 101;

CREATE INDEX idx_changelog_instance_db_name ON changelog (instance, db_name);

CREATE TABLE release (
    id bigserial PRIMARY KEY,
    deleted boolean NOT NULL DEFAULT FALSE,
    project text NOT NULL REFERENCES project(resource_id),
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    digest text NOT NULL DEFAULT '',
    -- Stored as ReleasePayload (proto/store/store/release.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

ALTER SEQUENCE release_id_seq RESTART WITH 101;

CREATE INDEX idx_release_project ON release(project);

-- OAuth2 tables
CREATE TABLE oauth2_client (
    client_id text PRIMARY KEY,
    client_secret_hash text NOT NULL,
    config jsonb NOT NULL,
    last_active_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE oauth2_authorization_code (
    code text PRIMARY KEY,
    client_id text NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_email text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    config jsonb NOT NULL,
    expires_at timestamptz NOT NULL
);

CREATE TABLE oauth2_refresh_token (
    token_hash text PRIMARY KEY,
    client_id text NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_email text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    expires_at timestamptz NOT NULL
);

CREATE INDEX idx_oauth2_authorization_code_expires_at ON oauth2_authorization_code(expires_at);
CREATE INDEX idx_oauth2_refresh_token_expires_at ON oauth2_refresh_token(expires_at);
CREATE INDEX idx_oauth2_client_last_active_at ON oauth2_client(last_active_at);

-- Web refresh tokens for session management
CREATE TABLE web_refresh_token (
    token_hash  TEXT PRIMARY KEY,
    user_email  TEXT NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_web_refresh_token_user_email ON web_refresh_token(user_email);
CREATE INDEX idx_web_refresh_token_expires_at ON web_refresh_token(expires_at);

-- Default bytebase system account id is 1.
INSERT INTO principal (id, type, name, email, password_hash) VALUES (1, 'SYSTEM_BOT', 'Bytebase', 'support@bytebase.com', '');

ALTER SEQUENCE principal_id_seq RESTART WITH 101;

-- Default project.
INSERT INTO project (id, name, resource_id) VALUES (1, 'Default', 'default');

ALTER SEQUENCE project_id_seq RESTART WITH 101;

-- Initialize settings with static values
INSERT INTO setting (name, value) VALUES ('APP_IM', '{}'::jsonb);
INSERT INTO setting (name, value) VALUES ('DATA_CLASSIFICATION', '{}'::jsonb);
INSERT INTO setting (name, value) VALUES ('WORKSPACE_APPROVAL', '{"rules":[{"template":{"flow":{"roles":["roles/projectOwner"]},"title":"Fallback Rule","description":"Requires project owner approval when no other rules match."},"condition":{"expression":"true"}}]}'::jsonb);
INSERT INTO setting (name, value) VALUES (
  'WORKSPACE_PROFILE',
  ('{"enableMetricCollection":true,"directorySyncToken":"' || gen_random_uuid()::text || '","passwordRestriction":{"minLength":8}}')::jsonb
);
INSERT INTO setting (name, value) VALUES ('ENVIRONMENT', '{"environments":[{"title":"Test","id":"test"},{"title":"Prod","id":"prod"}]}'::jsonb);

-- Initialize settings with dynamically generated values
-- Generate random alphanumeric string (0-9, a-z, A-Z) compatible with Go's common.RandomString
-- Initialize SYSTEM setting with auth_secret and workspace_id
INSERT INTO setting (name, value)
VALUES (
  'SYSTEM',
  json_build_object(
    'authSecret', (SELECT string_agg(substr('0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ', floor(random() * 62 + 1)::int, 1), '')
     FROM generate_series(1, 32)),
    'workspaceId', gen_random_uuid()::text
  )
);

-- Initialize workspace IAM policy
-- Grant workspace member role to allUsers
INSERT INTO policy (resource_type, resource, type, payload, inherit_from_parent, enforce)
VALUES ('WORKSPACE', '', 'IAM', '{"bindings":[{"role":"roles/workspaceMember","members":["allUsers"]}]}', FALSE, TRUE);
