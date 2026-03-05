# Bytebase SaaS Multi-Tenancy Migration Plan

## Goal

Refactor Bytebase from per-user isolation (one container per workspace) to a true SaaS model (single container, shared endpoint at `console.bytebase.com`).

## Chosen Approach: Row-Level Isolation (Approach 2)

Add a `workspace_id` column to root-level tables. Project-child tables reuse their existing `project` FK for workspace scoping — the `project` table has `workspace_id`, so any query that filters by `project` is implicitly workspace-scoped.

---

## Key Challenges

### Challenge 1: Workspace Isolation

Separate different workspaces so users only see their own data.

### Challenge 2: Database Connection Routing

Single connection pool; workspace resolved from JWT, applied as query filter (`WHERE workspace_id = $1` on root tables, `WHERE project = $1` on project children).

### Challenge 3: Background Runners

Runners poll for pending tasks across ALL workspaces with a single query. Each task carries `workspace_id` for context when processing.

### Challenge 4: API Resource Naming

Current API uses flat resource names with no workspace prefix:
- `projects/{project}`, `instances/{instance}`, `environments/{environment}`
- `projects/{project}/issues/{issue}`, `instances/{instance}/databases/{database}`

Two options:

**Option A: Keep flat names, resolve workspace from JWT (recommended)**

The middleware resolves workspace from the JWT and injects it as a query parameter. Resource IDs like `projects/default` are unique per-workspace because `project` has `UNIQUE(workspace_id, resource_id)`. Project-child queries use the project FK which is already workspace-scoped.

- Zero API breaking changes
- Zero proto changes
- Resource ID uniqueness scoped by `workspace_id` in unique indexes on root tables

**Option B: Explicit workspace prefix (`workspaces/{workspace}/projects/{project}`)**

AIP-compliant but requires changing every endpoint, proto definition, resource name parser, frontend API call, and external integration (Terraform provider, API clients, webhooks).

**Recommendation**: Option A. Workspace scoping is enforced at the DB level (root tables via `workspace_id`, child tables via `project` FK), making URL-level scoping redundant.

### Challenge 5: Multi-Workspace User Identity

Current model: single workspace per installation. `principal` table holds both identity (email, password, MFA) and workspace membership.

SaaS model requires:
- A user can belong to multiple workspaces
- Auth (login, password, MFA, OAuth) is workspace-independent
- After login, user picks a workspace (or auto-redirects if only one)
- JWT includes both `account_id` and `workspace_id`

This requires splitting `principal` into a global identity layer (see Phase 0).

---

## New Global Tables

These live in a shared/control-plane database, not scoped to any workspace.

### `workspace` table

```sql
CREATE TABLE workspace (
    id           text PRIMARY KEY,          -- 'ws-xxxxxx'
    resource_id  text UNIQUE NOT NULL,      -- user-chosen slug: 'acme-corp'
    name         text NOT NULL,             -- display name
    plan         text NOT NULL DEFAULT 'FREE',
    created_at   timestamptz NOT NULL DEFAULT now(),
    deleted      boolean NOT NULL DEFAULT FALSE
);
```

### `account` table (split from `principal`)

```sql
CREATE TABLE account (
    id              serial PRIMARY KEY,
    email           text UNIQUE NOT NULL,
    password_hash   text NOT NULL DEFAULT '',
    mfa_config      jsonb NOT NULL DEFAULT '{}',
    profile         jsonb NOT NULL DEFAULT '{}',
    created_at      timestamptz NOT NULL DEFAULT now()
);
```

### `workspace_member` table

```sql
CREATE TABLE workspace_member (
    account_id      int NOT NULL REFERENCES account(id),
    workspace_id    text NOT NULL REFERENCES workspace(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, workspace_id)
);
```

Auth token tables (`web_refresh_token`, `oauth2_client`, `oauth2_authorization_code`, `oauth2_refresh_token`) move to the account layer since login is cross-workspace.

---

## Table Classification

### Tier 1: Root Tables That Need `workspace_id` (11 tables)

These are top-level entities with no parent FK that provides workspace scoping. They get a `workspace_id` column directly.

| Table | Notes |
|-------|-------|
| `project` | Top-level resource, anchor for project-child scoping |
| `instance` | Top-level resource |
| `principal` | Users within workspace (becomes workspace membership record) |
| `setting` | Workspace configuration |
| `policy` | Workspace/env/project policies |
| `role` | Custom roles |
| `idp` | Identity providers |
| `review_config` | Review configurations |
| `user_group` | Groups |
| `export_archive` | Data exports |
| `audit_log` | Audit trail |

### Tier 2: Project Children -- Scoped via `project` FK (9 tables)

These already have a `project` FK referencing `project(resource_id)`. Since `project` has `workspace_id`, queries on these tables filter by `WHERE project = $1` where the project is already validated as belonging to the current workspace. **No `workspace_id` column needed.**

| Table | FK |
|-------|-----|
| `plan` | `project text NOT NULL REFERENCES project(resource_id)` |
| `issue` | `project text NOT NULL REFERENCES project(resource_id)` |
| `db` | `project text NOT NULL REFERENCES project(resource_id)` |
| `project_webhook` | `project text NOT NULL REFERENCES project(resource_id)` |
| `worksheet` | `project text NOT NULL REFERENCES project(resource_id)` |
| `db_group` | `project text NOT NULL REFERENCES project(resource_id)` |
| `release` | `project text NOT NULL REFERENCES project(resource_id)` |
| `access_grant` | `project text NOT NULL REFERENCES project(resource_id)` |
| `query_history` | `project_id text NOT NULL` (resource id, no FK constraint) |

**How workspace scoping works for these tables:**

```go
// 1. Middleware resolves workspace from JWT
workspaceID := extractWorkspaceFromJWT(ctx)

// 2. Validate project belongs to workspace
project, err := store.GetProject(ctx, workspaceID, projectResourceID)
// GetProject: SELECT ... FROM project WHERE workspace_id = $1 AND resource_id = $2

// 3. Query child table by project (already workspace-scoped)
plans, err := store.ListPlans(ctx, &FindPlanMessage{ProjectID: &project.ResourceID})
// ListPlans: SELECT ... FROM plan WHERE project = $1
```

### Tier 3: Grandchild Tables -- Scoped via Parent Chain (11 tables)

These are children of Tier 2 tables. They have no `project` FK and no `workspace_id`. Workspace scoping is inherited through the FK chain — they are always accessed via their parent, which is already workspace-scoped.

**Plan/pipeline grandchildren (5):**

| Table | FK chain to workspace |
|-------|----------------------|
| `plan_check_run` | -> `plan(id)` -> `plan.project` -> `project.workspace_id` |
| `plan_webhook_delivery` | -> `plan(id)` -> `plan.project` -> `project.workspace_id` |
| `task` | -> `plan(id)` -> `plan.project` -> `project.workspace_id` |
| `task_run` | -> `task(id)` -> `plan(id)` -> ... |
| `task_run_log` | -> `task_run(id)` -> `task(id)` -> ... |

**Issue grandchildren (1):**

| Table | FK chain to workspace |
|-------|----------------------|
| `issue_comment` | -> `issue(id)` -> `issue.project` -> `project.workspace_id` |

**DB grandchildren (3):**

| Table | FK chain to workspace |
|-------|----------------------|
| `db_schema` | -> `db` (via instance+db_name) -> `db.project` -> `project.workspace_id` |
| `revision` | -> `db` (via instance+db_name) -> `db.project` -> `project.workspace_id` |
| `sync_history` | -> `db` (via instance+db_name) -> `db.project` -> `project.workspace_id` |
| `changelog` | -> `db` (via instance+db_name) -> `db.project` -> `project.workspace_id` |

**Worksheet grandchildren (1):**

| Table | FK chain to workspace |
|-------|----------------------|
| `worksheet_organizer` | -> `worksheet(id)` -> `worksheet.project` -> `project.workspace_id` |

**Important**: These tables support querying by their own PK alone (e.g., `GetPlan(ctx, &FindPlanMessage{UID: &planID})`). In the current single-workspace model this is safe, but in SaaS mode, callers must ensure the parent is validated first. The typical API flow already does this:

```
API: GET /v1/projects/{project}/plans/{plan}
  1. Parse project from URL -> validate project belongs to workspace
  2. Parse plan UID from URL -> query plan by UID
  3. Verify plan.project == requested project (already workspace-scoped)
```

For **background runners** that query across all workspaces (e.g., `ListPendingTaskRuns` with no filter), this is intentional — runners process tasks for all workspaces and use the task's parent chain to resolve workspace context.

### Tables That Do NOT Need `workspace_id` (5 tables)

| Table | Reason |
|-------|--------|
| `sheet_blob` | Content-addressed by SHA256, shared/deduped across workspaces |
| `replica_heartbeat` | Infrastructure, not workspace-scoped |
| `instance_change_history` | Bytebase internal migration version tracker |
| `workspace` | It IS the workspace |
| `account` / `workspace_member` | Global identity layer |

### Auth Token Tables -- Move to Account Layer (no `workspace_id`)

| Table | Reason |
|-------|--------|
| `web_refresh_token` | Tied to account, not workspace |
| `oauth2_client` | Global OAuth2 app registration |
| `oauth2_authorization_code` | Account login flow |
| `oauth2_refresh_token` | Account login flow |

### Summary

| Tier | Tables | `workspace_id` column | Workspace scoping mechanism |
|------|--------|-----------------------|----------------------------|
| **Root** | 11 | Yes | Direct `WHERE workspace_id = $1` |
| **Project children** | 9 | No | `WHERE project = $1` (project already validated) |
| **Grandchildren** | 11 | No | Parent FK chain (parent validated by caller) |
| **Global** | 5 | No | Not workspace-scoped |
| **Auth (moved)** | 4 | No | Account layer |
| **Total** | **40** | **11 need column** | |

---

## Migration Phases

### Phase 0: Global Identity Layer

Create `workspace`, `account`, and `workspace_member` tables in the shared DB.

For existing installations, migrate auth data from `principal`:

```sql
-- Create the default workspace
INSERT INTO workspace (id, resource_id, name)
VALUES ('ws-default', 'default', 'Default Workspace');

-- Migrate END_USER principals to accounts
INSERT INTO account (email, password_hash, mfa_config, profile)
SELECT email, password_hash, mfa_config, profile
FROM principal WHERE type = 'END_USER';

-- Create workspace memberships
INSERT INTO workspace_member (account_id, workspace_id)
SELECT a.id, 'ws-default' FROM account a;
```

### Phase 1: Add `workspace_id` to Root Tables

Add `workspace_id` to the 11 root-level tables only:

```sql
-- For each root table (project, instance, principal, setting, policy,
-- role, idp, review_config, user_group, export_archive, audit_log)
ALTER TABLE project ADD COLUMN workspace_id text REFERENCES workspace(id);
UPDATE project SET workspace_id = 'ws-default';
ALTER TABLE project ALTER COLUMN workspace_id SET NOT NULL;

-- Repeat for remaining 10 root tables
```

Project-child tables (plan, issue, db, project_webhook, worksheet, db_group, release, access_grant, query_history) and grandchild tables do NOT get a `workspace_id` column — they are scoped through their existing `project` FK chain.

### Phase 2: Update Unique Indexes

Only root tables need index changes to include `workspace_id`:

```sql
-- BEFORE -> AFTER (root tables only)

-- project
DROP INDEX idx_project_unique_resource_id;
CREATE UNIQUE INDEX idx_project_unique_resource_id ON project(workspace_id, resource_id);

-- instance
DROP INDEX idx_instance_unique_resource_id;
CREATE UNIQUE INDEX idx_instance_unique_resource_id ON instance(workspace_id, resource_id);

-- principal (email now unique per-workspace, global uniqueness in account.email)
DROP INDEX idx_principal_unique_email;
CREATE UNIQUE INDEX idx_principal_unique_email ON principal(workspace_id, email);

-- setting
DROP INDEX idx_setting_unique_name;
CREATE UNIQUE INDEX idx_setting_unique_name ON setting(workspace_id, name);

-- role
DROP INDEX idx_role_unique_resource_id;
CREATE UNIQUE INDEX idx_role_unique_resource_id ON role(workspace_id, resource_id);

-- idp
DROP INDEX idx_idp_unique_resource_id;
CREATE UNIQUE INDEX idx_idp_unique_resource_id ON idp(workspace_id, resource_id);

-- user_group
DROP INDEX idx_user_group_unique_email;
CREATE UNIQUE INDEX idx_user_group_unique_email ON user_group(workspace_id, email) WHERE email IS NOT NULL;
```

Child table unique indexes remain unchanged — their uniqueness is naturally scoped because their parent references (e.g., `plan.project`, `db.instance`) are already workspace-unique:

```sql
-- These DO NOT change (already scoped through parent FKs):
-- idx_db_unique_instance_name ON db(instance, name)         -- instance is workspace-unique
-- idx_plan_check_run_unique_plan_id ON plan_check_run(plan_id)  -- plan_id is globally unique (serial)
-- idx_issue_unique_plan_id ON issue(plan_id)                    -- plan_id is globally unique (serial)
-- uk_task_run_task_id_attempt ON task_run(task_id, attempt)     -- task_id is globally unique (serial)
-- idx_db_group_unique_project_resource_id ON db_group(project, resource_id) -- project is workspace-unique
-- idx_release_project_train_iteration ON release(project, train, iteration) -- project is workspace-unique
```

Add workspace_id to non-unique indexes for query performance:

```sql
CREATE INDEX idx_project_workspace ON project(workspace_id);
CREATE INDEX idx_instance_workspace ON instance(workspace_id);
CREATE INDEX idx_setting_workspace ON setting(workspace_id);
CREATE INDEX idx_policy_workspace ON policy(workspace_id);
CREATE INDEX idx_audit_log_workspace ON audit_log(workspace_id);
```

### Phase 3: Application Layer Changes

**Middleware: extract workspace from JWT, inject into context**

```go
func WorkspaceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        workspaceID := extractWorkspaceFromJWT(r)
        ctx := context.WithValue(r.Context(), workspaceIDKey{}, workspaceID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Root table queries: add `WHERE workspace_id = $1`**

```go
// Store: root tables add workspace_id filter
func (s *Store) GetProject(ctx context.Context, workspaceID, resourceID string) (*ProjectMessage, error) {
    query := "SELECT ... FROM project WHERE workspace_id = $1 AND resource_id = $2"
    // ...
}

func (s *Store) ListInstances(ctx context.Context, workspaceID string, find *FindInstanceMessage) ([]*InstanceMessage, error) {
    // WHERE workspace_id = $1 AND ...
}

// Service: extract workspace from context, pass to store
func (s *ProjectService) GetProject(ctx context.Context, req *v1pb.GetProjectRequest) (*v1pb.Project, error) {
    workspaceID := WorkspaceIDFromContext(ctx)
    project, err := s.store.GetProject(ctx, workspaceID, resourceID)
    // ...
}
```

**Project-child table queries: filter by `project` (no workspace_id needed)**

```go
// Store: project children remain unchanged -- filter by project FK
func (s *Store) ListPlans(ctx context.Context, find *FindPlanMessage) ([]*PlanMessage, error) {
    // WHERE project = $1  (project already validated as belonging to workspace)
}

// Service: validate project ownership first, then query children
func (s *PlanService) ListPlans(ctx context.Context, req *v1pb.ListPlansRequest) (*v1pb.ListPlansResponse, error) {
    workspaceID := WorkspaceIDFromContext(ctx)
    projectID := getProjectID(req.Parent)

    // Step 1: validate project belongs to this workspace
    project, err := s.store.GetProject(ctx, workspaceID, projectID)
    if err != nil { return nil, err }

    // Step 2: query plans by project (already workspace-scoped)
    plans, err := s.store.ListPlans(ctx, &FindPlanMessage{ProjectID: &project.ResourceID})
    // ...
}
```

**Grandchild table queries: accessed through validated parent**

```go
// Task runs are accessed through plan -> project -> workspace chain
func (s *RolloutService) GetTaskRun(ctx context.Context, req *v1pb.GetTaskRunRequest) (*v1pb.TaskRun, error) {
    workspaceID := WorkspaceIDFromContext(ctx)
    projectID, planUID, _, taskRunUID := parseTaskRunName(req.Name)

    // Validate project belongs to workspace
    project, err := s.store.GetProject(ctx, workspaceID, projectID)
    if err != nil { return nil, err }

    // Validate plan belongs to project
    plan, err := s.store.GetPlan(ctx, &FindPlanMessage{UID: &planUID, ProjectID: &project.ResourceID})
    if err != nil { return nil, err }

    // Query task_run (parent chain already validated)
    taskRun, err := s.store.GetTaskRun(ctx, &FindTaskRunMessage{UID: &taskRunUID})
    // ...
}
```

### Phase 4: Background Runners

Runners query across all workspaces without a workspace filter. They resolve workspace context from the task's parent chain when processing.

```go
func (r *Scheduler) Run(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    for {
        select {
        case <-ticker.C:
            // No workspace filter -- returns pending task runs across all workspaces
            tasks, _ := r.store.ListPendingTaskRuns(ctx)
            for _, task := range tasks {
                go r.processTask(ctx, task)
            }
        }
    }
}

func (r *Scheduler) processTask(ctx context.Context, taskRun *TaskRun) {
    // Resolve workspace through parent chain:
    // task_run -> task -> plan -> project -> workspace_id
    task, _ := r.store.GetTask(ctx, taskRun.TaskID)
    plan, _ := r.store.GetPlan(ctx, &FindPlanMessage{UID: &task.PlanID})
    project, _ := r.store.GetProjectByResourceID(ctx, plan.ProjectID)
    workspaceID := project.WorkspaceID

    // Use workspaceID for workspace-specific operations
    // (e.g., loading instance credentials, checking workspace settings)
}
```

**Note**: For frequently polled runner queries, consider denormalizing `project` onto `task` or `plan_check_run` to avoid the join chain on every poll cycle. This is an optimization, not a requirement.

### Phase 5: Auth Flow Changes

**Current flow:**
1. User hits `{id}.bytebase.com` -> single workspace container
2. Login with email/password or SSO
3. JWT scoped to workspace implicitly

**SaaS flow:**
1. User hits `console.bytebase.com`
2. Login (password/SSO) -> authenticates against global `account` table
3. If user has 1 workspace -> auto-redirect
4. If user has N workspaces -> workspace picker
5. If user has 0 workspaces -> create workspace or accept invitation
6. After workspace selected -> JWT includes both `account_id` and `workspace_id`
7. All subsequent requests scoped to that workspace

**Service accounts and workload identities** remain workspace-scoped (they live in the per-workspace `principal` table as they are today).

---

## Why Approach 4 (RLS) Was Rejected

RLS policies are per-table: each table must have its own `workspace_id` column for the policy `USING (workspace_id = current_setting('app.workspace_id'))` to evaluate. This means:

- All 30+ workspace-scoped tables would need a `workspace_id` column
- Cannot leverage existing `project` FK for workspace scoping on child tables
- Conflicts with the goal of minimizing schema changes

Approach 2 (Row-Level) with project-FK scoping requires `workspace_id` on only 11 root tables, while child tables use their existing FK relationships.

---

## Estimated Code Changes

### Schema Migration
```
- New tables: workspace, account, workspace_member
- ALTER TABLE ADD COLUMN workspace_id: 11 root tables
- UPDATE backfill: 11 root tables
- DROP + CREATE UNIQUE INDEX: ~7 indexes (root tables only)
- CREATE INDEX (workspace filtering): ~5 indexes
- Migrate principal -> account data
```

### Application Changes
```
- Store layer: ~20-30 root table queries modified (add WHERE workspace_id = $1)
- Store layer: INSERT queries for root tables modified (set workspace_id)
- Services: ~20-30 methods for root tables updated (pass workspaceID)
- Services: project-child endpoints add project ownership validation
- Middleware: ~50 lines (extract workspace from JWT, inject into context)
- Context helpers: ~20 lines
- Runner modifications: ~100 lines (resolve workspace from parent chain)
- Auth flow: ~300 lines (account-based login, workspace picker)
- Project-child queries: 0 changes (filter by project FK as before)
- Grandchild queries: 0 changes (accessed through validated parent)
```
