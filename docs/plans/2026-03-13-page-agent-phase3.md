# Page Agent Phase 3: Skills, Context & Polish — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add on-demand skill loading, rich route-aware page context, keyboard shortcut to toggle the agent window, and error handling/retry/loading improvements to the agent loop.

**Architecture:** Skills are ported from backend MCP (`backend/api/mcp/skills/*.md`) as static TypeScript modules in `plugins/agent/logic/skills/`. A new `get_skill` tool lets the LLM load workflow guides on demand. The `get_page_state` semantic mode is enhanced to read Pinia stores based on the current route, returning structured data (project info, database list, issue status, members). The agent window gets a global keyboard shortcut. The agent loop gets retry logic for transient failures and better error/loading state management.

**Tech Stack:** TypeScript, Vue 3, Pinia, Vue Router, Vite

---

## File Overview

```
frontend/src/plugins/agent/
├── logic/
│   ├── skills/                      # NEW — on-demand skill content
│   │   ├── index.ts                 # Skill registry + get_skill implementation
│   │   ├── query.ts                 # Ported from backend/api/mcp/skills/query.md
│   │   ├── databaseChange.ts        # Ported from backend/api/mcp/skills/database-change.md
│   │   └── grantPermission.ts       # Ported from backend/api/mcp/skills/grant-permission.md
│   ├── tools/
│   │   └── index.ts                 # MODIFY — register get_skill tool
│   ├── context.ts                   # NEW — route-aware page context extraction
│   ├── agentLoop.ts                 # MODIFY — add retry logic, better error handling
│   └── prompt.ts                    # MODIFY — add skill tool guidance
├── store/
│   └── agent.ts                     # MODIFY — add error state
├── components/
│   ├── AgentInput.vue               # MODIFY — show error state
│   └── AgentChat.vue                # MODIFY — i18n for "Thinking..."
└── index.ts                         # MODIFY — register keyboard shortcut
```

---

## Task 1: Skill Content Modules

Port MCP skill markdown files to TypeScript string constants. Each skill is a module exporting `name`, `description`, and `content`.

**Files:**
- Create: `frontend/src/plugins/agent/logic/skills/query.ts`
- Create: `frontend/src/plugins/agent/logic/skills/databaseChange.ts`
- Create: `frontend/src/plugins/agent/logic/skills/grantPermission.ts`

**Step 1: Create `query.ts`**

```typescript
// frontend/src/plugins/agent/logic/skills/query.ts

export const skill = {
  name: "query",
  description:
    "Use when running SQL queries, executing SELECT/INSERT/UPDATE/DELETE statements, or fetching data from databases",
  content: `# Execute SQL

## Overview

Run SQL queries against databases managed by Bytebase.

## Prerequisites

- Know the instance and database name
- Have \`bb.sql.query\` permission

## Workflow

1. **Get the schema**:
   \`\`\`
   search_api(operationId="SQLService/Query")
   \`\`\`

2. **List databases** (if needed):
   \`\`\`
   call_api(operationId="DatabaseService/ListDatabases", body={
     "parent": "workspaces/{id}",
     "filter": "name.matches(\\"db_name\\")"
   })
   \`\`\`

   Filter examples:
   - \`name.matches("employee")\` - database name contains "employee"
   - \`project == "projects/{project-id}"\` - databases in a project
   - \`instance == "instances/{instance-id}"\` - databases in an instance
   - \`engine == "MYSQL"\` - MySQL databases only
   - \`environment == "environments/prod" && name.matches("user")\` - combine filters

   **Extract \`dataSourceId\`** from \`instanceResource.dataSources\` in the response. Prefer \`type: "READ_ONLY"\` over \`type: "ADMIN"\` when available.

3. **Execute SQL**:
   \`\`\`
   call_api(operationId="SQLService/Query", body={
     "name": "instances/{instance-id}/databases/{database-name}",
     "dataSourceId": "{data-source-id}",
     "statement": "SELECT * FROM users LIMIT 10"
   })
   \`\`\`

   \`dataSourceId\` is **required**.

## Notes

- Query results may contain masked values due to data masking policies. Do not remove or modify masked values.
  - Full mask: \`******\`
  - Partial mask: \`**rn**\` (only "rn" visible)
- **Displaying partial masks:** Use backticks or code blocks when presenting results. Without escaping, markdown interprets \`**text**\` as bold formatting.

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| data source id is required | Missing dataSourceId field | Get dataSourceId from instanceResource.dataSources in database listing |
| database not found | Wrong instance/database name | List databases first |
| permission denied | Missing bb.sql.query | Check user permissions |
| syntax error | Invalid SQL | Check SQL syntax for the database engine |`,
} as const;
```

**Step 2: Create `databaseChange.ts`**

```typescript
// frontend/src/plugins/agent/logic/skills/databaseChange.ts

export const skill = {
  name: "database-change",
  description:
    "Use when making schema changes (DDL), data migrations, ALTER TABLE, CREATE TABLE, or deploying SQL changes through review workflow",
  content: `# Database Change

## Overview

Create database changes (DDL/DML) through Bytebase's review workflow. Supports single database or batch changes across multiple databases.

## Prerequisites

- Have \`bb.plans.create\`, \`bb.issues.create\`, \`bb.rollouts.create\` permissions
- Know the target database(s)

## Workflow

### Step 1: Create sheet(s) with SQL

SQL content must be **base64 encoded**. Engine field is **required**.

\`\`\`
search_api(operationId="SheetService/CreateSheet")
\`\`\`
\`\`\`
call_api(operationId="SheetService/CreateSheet", body={
  "parent": "projects/{project-id}",
  "sheet": {
    "title": "Add users table",
    "engine": "POSTGRES",
    "content": "Q1JFQVRFIFRBQkxFIHVzZXJzIChpZCBJTlQgUFJJTUFSWSBLRVkpOw=="
  }
})
\`\`\`

Note: \`Q1JFQVRFIFRBQkxFIHVzZXJzIChpZCBJTlQgUFJJTUFSWSBLRVkpOw==\` decodes to \`CREATE TABLE users (id INT PRIMARY KEY);\`

Use \`search_api(schema="Engine")\` to discover valid engine values.

### Step 2: Create a plan

Plan contains \`specs\` directly (not wrapped in "steps").

**Single database:**
\`\`\`
search_api(operationId="PlanService/CreatePlan")
\`\`\`
\`\`\`
call_api(operationId="PlanService/CreatePlan", body={
  "parent": "projects/{project-id}",
  "plan": {
    "title": "Add users table",
    "specs": [{
      "id": "spec-1",
      "changeDatabaseConfig": {
        "targets": ["instances/{instance-id}/databases/{database-name}"],
        "sheet": "projects/{project-id}/sheets/{sheet-id}",
        "type": "MIGRATE"
      }
    }]
  }
})
\`\`\`

**Key concepts:**
- \`specs\` is a flat array directly on the plan (no "steps" wrapper)
- Each \`spec\` has a unique \`id\` (any string, used for identification)
- \`targets\` is an **array** (even for single database)
- Use \`search_api(schema="Engine")\` to discover valid engine values

**Using database groups:**
\`\`\`
"targets": ["projects/{project-id}/databaseGroups/{group-name}"]
\`\`\`

### Step 3: Create an issue

\`\`\`
search_api(operationId="IssueService/CreateIssue")
\`\`\`
\`\`\`
call_api(operationId="IssueService/CreateIssue", body={
  "parent": "projects/{project-id}",
  "issue": {
    "title": "Add users table",
    "type": "DATABASE_CHANGE",
    "plan": "projects/{project-id}/plans/{plan-id}"
  }
})
\`\`\`

### Step 4: Create a rollout

\`\`\`
search_api(operationId="RolloutService/CreateRollout")
\`\`\`
\`\`\`
call_api(operationId="RolloutService/CreateRollout", body={
  "parent": "projects/{project-id}",
  "rollout": {
    "plan": "projects/{project-id}/plans/{plan-id}"
  }
})
\`\`\`

## Change Types

| Type | Use Case |
|------|----------|
| \`MIGRATE\` | Imperative schema/data changes (DDL and DML) |
| \`SDL\` | State-based declarative schema migration |

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| database not found | Wrong database reference | Verify \`instances/{id}/databases/{name}\` format |
| sheet not found | Sheet doesn't exist | Create sheet first (Step 1) |
| missing engine | Sheet without engine field | Add \`engine\` field to sheet |
| plan not found | Plan doesn't exist | Create plan before issue |
| invalid base64 | SQL not encoded | Base64 encode the SQL content |
| targets must be array | Using string instead of array | Wrap target in array: \`["..."]\` |`,
} as const;
```

**Step 3: Create `grantPermission.ts`**

```typescript
// frontend/src/plugins/agent/logic/skills/grantPermission.ts

export const skill = {
  name: "grant-permission",
  description:
    "Use when granting roles, managing access control, adding users to projects, or setting up RBAC permissions",
  content: `# Grant Permission

## Overview

Grant roles to users or groups using Bytebase's RBAC system. Permissions can be set at workspace level (global) or project level (scoped).

## Prerequisites

- Have \`bb.workspaces.setIamPolicy\` for workspace-level grants
- Have \`bb.projects.setIamPolicy\` for project-level grants

## Permission Levels

| Level | Scope | Use Case |
|-------|-------|----------|
| **Workspace** | All projects | Admins, DBAs, global viewers |
| **Project** | Single project | Developers, project owners |

## Workflow

### Step 1: List Available Roles

\`\`\`
search_api(operationId="RoleService/ListRoles")
\`\`\`
\`\`\`
call_api(operationId="RoleService/ListRoles", body={})
\`\`\`

**Built-in roles:**

| Role | Description |
|------|-------------|
| \`roles/workspaceAdmin\` | Full workspace access |
| \`roles/workspaceDBA\` | Database administration |
| \`roles/workspaceMember\` | Basic workspace access |
| \`roles/projectOwner\` | Full project access |
| \`roles/projectDeveloper\` | Development access (create issues, plans) |
| \`roles/projectReleaser\` | Execute rollouts |
| \`roles/sqlEditorUser\` | SQL Editor access only |
| \`roles/projectViewer\` | Read-only access |

### Step 2: Get Current IAM Policy

Always fetch current policy first to avoid overwriting existing bindings.

**Workspace level:**
\`\`\`
call_api(operationId="WorkspaceService/GetIamPolicy", body={
  "resource": "workspaces/{id}"
})
\`\`\`

**Project level:**
\`\`\`
call_api(operationId="ProjectService/GetIamPolicy", body={
  "resource": "projects/{project-id}"
})
\`\`\`

Save the returned \`etag\` for the update.

### Step 3: Set IAM Policy

Add new bindings while preserving existing ones.

**Workspace level:**
\`\`\`
call_api(operationId="WorkspaceService/SetIamPolicy", body={
  "resource": "workspaces/{id}",
  "etag": "{etag-from-get}",
  "policy": {
    "bindings": [
      {
        "role": "roles/workspaceDBA",
        "members": ["user:dba@example.com"]
      }
    ]
  }
})
\`\`\`

**Project level:**
\`\`\`
call_api(operationId="ProjectService/SetIamPolicy", body={
  "resource": "projects/{project-id}",
  "etag": "{etag-from-get}",
  "policy": {
    "bindings": [
      {
        "role": "roles/projectOwner",
        "members": ["user:lead@example.com"]
      }
    ]
  }
})
\`\`\`

## Member Format

| Type | Format | Example |
|------|--------|---------|
| User | \`user:{email}\` | \`user:alice@example.com\` |
| Group | \`group:{email}\` | \`group:devs@example.com\` |
| All users | \`allUsers\` | \`allUsers\` |

## Time-Limited Access (CEL Expressions)

Grant temporary access using CEL conditions:
\`\`\`
"condition": {
  "expression": "request.time < timestamp('2024-12-31T23:59:59Z')",
  "title": "Temporary access",
  "description": "Access expires end of 2024"
}
\`\`\`

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| etag mismatch | Policy changed | Re-fetch policy and retry |
| role not found | Invalid role name | List roles first, use \`roles/{name}\` format |
| invalid member | Wrong format | Use \`user:email\` or \`group:email\` format |
| permission denied | Missing setIamPolicy | Check workspace/project admin access |`,
} as const;
```

**Step 4: Commit**

```
feat(agent): port MCP skill content to frontend modules
```

---

## Task 2: Skill Registry and `get_skill` Tool

Create the skill registry that lists and retrieves skills, and register the `get_skill` tool.

**Files:**
- Create: `frontend/src/plugins/agent/logic/skills/index.ts`
- Modify: `frontend/src/plugins/agent/logic/tools/index.ts`

**Step 1: Create `skills/index.ts`**

```typescript
// frontend/src/plugins/agent/logic/skills/index.ts

import { skill as databaseChangeSkill } from "./databaseChange";
import { skill as grantPermissionSkill } from "./grantPermission";
import { skill as querySkill } from "./query";

interface Skill {
  name: string;
  description: string;
  content: string;
}

const skills: Skill[] = [querySkill, databaseChangeSkill, grantPermissionSkill];

const skillMap = new Map(skills.map((s) => [s.name, s]));

export interface GetSkillArgs {
  name?: string;
}

export function getSkill(args: GetSkillArgs): string {
  if (!args.name) {
    // List all skills
    const list = skills
      .map((s) => `- **${s.name}**: ${s.description}`)
      .join("\n");
    return `Available skills:\n${list}\n\nCall get_skill(name="skill-name") to load a specific skill.`;
  }

  const skill = skillMap.get(args.name);
  if (!skill) {
    const names = skills.map((s) => s.name).join(", ");
    return `Skill "${args.name}" not found. Available skills: ${names}`;
  }

  return skill.content;
}
```

**Step 2: Register `get_skill` in `tools/index.ts`**

Add import at top:
```typescript
import { getSkill, type GetSkillArgs } from "../skills";
```

Add to the `getToolDefinitions()` return array (after `dom_action`):
```typescript
{
  name: "get_skill",
  description: `Get step-by-step workflow guides for common Bytebase tasks. Load a skill before performing multi-step operations.

| Mode | Parameters | Result |
|------|------------|--------|
| List | (none) | All available skills |
| Load | name="query" | Step-by-step workflow guide |

**Available skills:** query, database-change, grant-permission

**Workflow:** get_skill() → get_skill(name="...") → follow the guide using search_api + call_api`,
  parametersSchema: {
    type: "object",
    properties: {
      name: {
        type: "string",
        description:
          'Skill name to load. Omit to list all skills. Examples: "query", "database-change", "grant-permission"',
      },
    },
  },
},
```

Add to the `switch` in `createToolExecutor`:
```typescript
case "get_skill":
  return getSkill(args as GetSkillArgs);
```

**Step 3: Commit**

```
feat(agent): add get_skill tool with on-demand workflow guides
```

---

## Task 3: Route-Aware Page Context

Enhance `get_page_state` semantic mode to read Pinia stores based on the current route, returning structured data about the current project, databases, issue, etc.

**Files:**
- Create: `frontend/src/plugins/agent/logic/context.ts`
- Modify: `frontend/src/plugins/agent/logic/tools/pageState.ts`

**Step 1: Create `context.ts`**

This module reads Pinia stores based on the current route and returns structured context. It uses `getActivePinia()` to check if stores are initialized — stores may not be active on all routes.

```typescript
// frontend/src/plugins/agent/logic/context.ts

import type { RouteLocationNormalizedLoaded } from "vue-router";
import { useCurrentUserV1 } from "@/store";

interface PageContext {
  user?: { name: string; email: string; role: string };
  project?: { name: string; title: string; state: string };
  database?: { name: string; engine: string; environment: string };
  issue?: { name: string; title: string; status: string; type: string };
  [key: string]: unknown;
}

export async function extractRouteContext(
  route: RouteLocationNormalizedLoaded
): Promise<PageContext> {
  const ctx: PageContext = {};

  // Current user — always available
  try {
    const user = useCurrentUserV1();
    if (user.value?.name) {
      ctx.user = {
        name: user.value.name,
        email: user.value.email,
        role: user.value.userRole?.toString() ?? "",
      };
    }
  } catch {
    // Store not initialized
  }

  const { projectId, databaseName, instanceId, issueId } = route.params as Record<string, string>;

  // Project context
  if (projectId) {
    try {
      const { useProjectV1Store } = await import("@/store");
      const store = useProjectV1Store();
      const project = store.getProjectByName(`projects/${projectId}`);
      if (project?.name) {
        ctx.project = {
          name: project.name,
          title: project.title,
          state: project.state?.toString() ?? "",
        };
      }
    } catch {
      // Store not available
    }
  }

  // Database context
  if (instanceId && databaseName) {
    try {
      const { useDatabaseV1Store } = await import("@/store");
      const store = useDatabaseV1Store();
      const db = store.getDatabaseByName(
        `instances/${instanceId}/databases/${databaseName}`
      );
      if (db?.name) {
        ctx.database = {
          name: db.name,
          engine: db.instanceResource?.engine?.toString() ?? "",
          environment: db.effectiveEnvironment ?? "",
        };
      }
    } catch {
      // Store not available
    }
  }

  // Issue context
  if (projectId && issueId) {
    try {
      const { useIssueV1Store } = await import("@/store");
      const store = useIssueV1Store();
      const issue = await store.fetchIssueByName(
        `projects/${projectId}/issues/${issueId}`
      );
      if (issue?.name) {
        ctx.issue = {
          name: issue.name,
          title: issue.title,
          status: issue.status?.toString() ?? "",
          type: issue.type?.toString() ?? "",
        };
      }
    } catch {
      // Store not available or fetch failed
    }
  }

  return ctx;
}
```

**Step 2: Update `pageState.ts` to use `extractRouteContext`**

Replace the file:

```typescript
// frontend/src/plugins/agent/logic/tools/pageState.ts

import type { Router } from "vue-router";
import { lazyExtractDomTree } from "../../dom";
import { extractRouteContext } from "../context";

export interface PageStateArgs {
  mode?: "semantic" | "dom";
}

export function createPageStateTool(router: Router) {
  return async (args?: PageStateArgs): Promise<string> => {
    const route = router.currentRoute.value;
    const base: Record<string, unknown> = {
      path: route.fullPath,
      name: String(route.name ?? ""),
      params: route.params,
      query: route.query,
      title: document.title,
    };

    // Enrich with Pinia store data for semantic mode
    const ctx = await extractRouteContext(route);
    if (Object.keys(ctx).length > 0) {
      base.context = ctx;
    }

    if (args?.mode === "dom") {
      const { tree, count } = await lazyExtractDomTree();
      return JSON.stringify({
        ...base,
        interactiveElements: count,
        domTree: tree,
      });
    }

    return JSON.stringify(base);
  };
}
```

**Step 3: Commit**

```
feat(agent): add route-aware page context from Pinia stores
```

---

## Task 4: Update System Prompt with Skill Guidance

Update the system prompt to mention the `get_skill` tool and improve context injection.

**Files:**
- Modify: `frontend/src/plugins/agent/logic/prompt.ts`

**Step 1: Update `prompt.ts`**

```typescript
// frontend/src/plugins/agent/logic/prompt.ts

export function buildSystemPrompt(pageContext: {
  path: string;
  title: string;
  role?: string;
}): string {
  return `You are Bytebase Assistant, an AI agent embedded in the Bytebase console.
You help DBAs and developers manage databases, write SQL, review changes,
and navigate the platform.

Rules:
- Use search_api + call_api for actions. Prefer API over DOM interaction.
- Use navigate for "show me" / "go to" requests.
- Use get_skill to load step-by-step workflow guides before multi-step tasks (SQL queries, schema changes, permission grants).
- Use dom_action only when no API covers the task. Always call get_page_state(mode="dom") first.
- Workflow for DOM interaction: get_page_state(mode="dom") → read element indices → dom_action(type, index, value).
- Always confirm destructive actions (drop database, delete project) before executing.
- Use get_page_state to understand the current page context before answering questions.

Core concepts:
- Workspace: top-level container. One workspace per deployment.
- Project: groups databases and members. All changes happen within a project.
- Database: belongs to a project, hosted on an instance.
- Instance: a database server (MySQL, PostgreSQL, etc.) in an environment.
- Environment: dev, staging, prod. Controls approval policies.
- Change ticket (Issue): the review workflow for schema/data changes.
  Flow: create → review → approve → roll out.
- SQL Editor: interactive query tool with access control.

Current page: ${pageContext.path}
Page title: ${pageContext.title}${pageContext.role ? `\nYour role: ${pageContext.role}` : ""}`;
}
```

**Step 2: Commit**

```
feat(agent): add skill tool guidance to system prompt
```

---

## Task 5: Keyboard Shortcut to Toggle Agent

Add a global keyboard shortcut (Ctrl+Shift+A / Cmd+Shift+A) to toggle the agent window.

**Files:**
- Modify: `frontend/src/plugins/agent/index.ts`

**Step 1: Read current `index.ts`**

Read `frontend/src/plugins/agent/index.ts` to understand current exports.

**Step 2: Add keyboard shortcut registration**

Add a composable that registers the shortcut. This should be called from the app root (where `AgentWindow` is mounted).

```typescript
// Add to frontend/src/plugins/agent/index.ts

import { onMounted, onUnmounted } from "vue";
import { useAgentStore } from "./store/agent";

export { default as AgentWindow } from "./components/AgentWindow.vue";
export { useAgentStore } from "./store/agent";

export function useAgentShortcut() {
  const store = useAgentStore();

  function handleKeydown(e: KeyboardEvent) {
    // Ctrl+Shift+A (Windows/Linux) or Cmd+Shift+A (Mac)
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "A") {
      e.preventDefault();
      store.toggle();
    }
  }

  onMounted(() => {
    window.addEventListener("keydown", handleKeydown);
  });

  onUnmounted(() => {
    window.removeEventListener("keydown", handleKeydown);
  });
}
```

**Step 3: Wire shortcut in `BodyLayout.vue`**

In `frontend/src/layouts/BodyLayout.vue`, import and call `useAgentShortcut()` in the `<script setup>` block alongside where `<AgentWindow />` is rendered:

```typescript
import { useAgentShortcut } from "@/plugins/agent";
useAgentShortcut();
```

**Step 4: Commit**

```
feat(agent): add Ctrl+Shift+A keyboard shortcut to toggle agent
```

---

## Task 6: Agent Loop Error Handling and Retries

Add retry logic for transient failures (network errors, 5xx) in the agent loop, and expose error state to the UI.

**Files:**
- Modify: `frontend/src/plugins/agent/logic/agentLoop.ts`
- Modify: `frontend/src/plugins/agent/store/agent.ts`

**Step 1: Add retry logic to `agentLoop.ts`**

Add a `callWithRetry` helper that retries the backend `AIService.Chat` call on transient failures. Add it before the `runAgentLoop` function:

```typescript
const MAX_RETRIES = 2;
const RETRY_DELAY_MS = 1000;

async function callWithRetry<T>(
  fn: () => Promise<T>,
  signal?: AbortSignal
): Promise<T> {
  let lastError: Error | undefined;
  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    if (signal?.aborted) {
      throw new DOMException("Agent loop aborted", "AbortError");
    }
    try {
      return await fn();
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err));
      // Don't retry client errors (4xx) or abort
      if (lastError.name === "AbortError") throw lastError;
      const msg = lastError.message.toLowerCase();
      if (msg.includes("400") || msg.includes("401") || msg.includes("403") || msg.includes("404")) {
        throw lastError;
      }
      if (attempt < MAX_RETRIES) {
        await new Promise((r) => setTimeout(r, RETRY_DELAY_MS * (attempt + 1)));
      }
    }
  }
  throw lastError!;
}
```

Then wrap the `aiServiceClientConnect.chat()` call in `runAgentLoop`:

```typescript
// Replace:
const response = await aiServiceClientConnect.chat(
  { messages: protoMessages, toolDefinitions: protoTools },
  { signal }
);

// With:
const response = await callWithRetry(
  () =>
    aiServiceClientConnect.chat(
      { messages: protoMessages, toolDefinitions: protoTools },
      { signal }
    ),
  signal
);
```

**Step 2: Add error state to store**

In `frontend/src/plugins/agent/store/agent.ts`, add an `error` ref:

```typescript
const error = ref<string | null>(null);
```

Add `clearError` function:
```typescript
function clearError() {
  error.value = null;
}
```

Export `error` and `clearError` in the return object.

**Step 3: Update `AgentInput.vue` to clear error before send and set error on failure**

In the `send()` function in `AgentInput.vue`, add `agentStore.clearError()` at the start and set `agentStore.error` in the catch block:

```typescript
// At the start of send():
agentStore.clearError();

// In the catch block, before the existing code:
if ((err as Error).name !== "AbortError") {
  agentStore.error = (err as Error).message;
  agentStore.addMessage({
    role: "assistant",
    content: `Error: ${(err as Error).message}`,
  });
}
```

**Step 4: Commit**

```
feat(agent): add retry logic and error state to agent loop
```

---

## Task 7: i18n Keys for New Features

Add locale strings for the new Phase 3 features.

**Files:**
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/zh-CN.json`

**Step 1: Add keys under `"agent"` in `en-US.json`**

Add these keys to the existing `"agent"` block (after `"result"`):

```json
"loading-skill": "Loading skill...",
"shortcut-hint": "Ctrl+Shift+A",
"retry-failed": "Request failed after retries",
"error-prefix": "Error"
```

**Step 2: Add equivalent keys in `zh-CN.json`**

```json
"loading-skill": "加载技能...",
"shortcut-hint": "Ctrl+Shift+A",
"retry-failed": "重试后请求失败",
"error-prefix": "错误"
```

**Step 3: Update `AgentChat.vue` to use i18n for "Thinking..."**

In `AgentChat.vue`, the loading indicator currently hardcodes "Thinking...". Replace:

```html
<!-- Before -->
<span class="animate-pulse">&#9679;</span> Thinking...

<!-- After -->
<span class="animate-pulse">&#9679;</span> {{ $t("common.loading") }}
```

Note: `common.loading` already exists in the locale files. If the team prefers a distinct key, add `"thinking": "Thinking..."` to the agent block instead.

**Step 4: Commit**

```
feat(agent): add i18n keys for Phase 3 features
```

---

## Task 8: Update `get_page_state` Tool Description

Update the tool description to reflect the richer semantic context.

**Files:**
- Modify: `frontend/src/plugins/agent/logic/tools/index.ts`

**Step 1: Update description**

Replace the existing `get_page_state` description:

```typescript
{
  name: "get_page_state",
  description: `Get information about the current page.

| Mode | Result |
|------|--------|
| semantic (default) | Route path, params, title + context from Pinia stores (project, database, issue, user info when available) |
| dom | Above + indexed DOM tree of interactive elements |

Use mode="dom" before dom_action to get element indices. Use semantic mode (default) to understand the current page context.`,
  parametersSchema: {
    type: "object",
    properties: {
      mode: {
        type: "string",
        enum: ["semantic", "dom"],
        description:
          'Default "semantic" returns route info + store context. "dom" adds an indexed tree of interactive elements for use with dom_action.',
      },
    },
  },
},
```

**Step 2: Commit**

```
feat(agent): update get_page_state description for rich context
```

---

## Task 9: Lint, Type-Check, and Build

Validate everything compiles and passes checks.

**Step 1: Run frontend fix**

```bash
pnpm --dir frontend fix
```

**Step 2: Run type-check**

```bash
pnpm --dir frontend type-check
```

Fix any type errors. Common issues:
- Import paths for store composables
- Type narrowing on route params (they may be `string | string[]`)
- Proto enum values that need `.toString()`

**Step 3: Run check**

```bash
pnpm --dir frontend check
```

**Step 4: Commit any fixes**

```
fix(agent): lint and type-check fixes for Phase 3
```

---

## Task 10: Manual Integration Test

Verify Phase 3 features work end-to-end.

**Step 1: Start dev server**

```bash
pnpm --dir frontend dev
```

**Step 2: Test keyboard shortcut**

1. Press Ctrl+Shift+A (or Cmd+Shift+A on Mac) — agent window should appear
2. Press again — window should close
3. Verify no conflict with browser shortcuts

**Step 3: Test skill loading**

1. Open the agent window
2. Ask: "How do I run a SQL query?"
3. The agent should call `get_skill(name="query")` and use the workflow guide
4. Ask: "What skills are available?"
5. The agent should call `get_skill()` and list all three skills

**Step 4: Test rich page context**

1. Navigate to a project page (e.g., `/projects/my-project`)
2. Ask: "What page am I on?"
3. The agent should call `get_page_state()` and return project info (title, state) along with route info
4. Navigate to a database detail page
5. Ask the same — should include database engine and environment

**Step 5: Test error handling**

1. Disconnect network (DevTools → Network → Offline)
2. Send a message to the agent
3. Verify the error message appears after retries
4. Reconnect and verify the agent works again

---

## Summary

| Task | Files | What |
|------|-------|------|
| 1 | `logic/skills/query.ts`, `databaseChange.ts`, `grantPermission.ts` | Port MCP skill content to frontend |
| 2 | `logic/skills/index.ts`, `logic/tools/index.ts` | Skill registry + `get_skill` tool |
| 3 | `logic/context.ts`, `logic/tools/pageState.ts` | Route-aware Pinia store context |
| 4 | `logic/prompt.ts` | System prompt with skill guidance |
| 5 | `index.ts`, `BodyLayout.vue` | Keyboard shortcut (Ctrl+Shift+A) |
| 6 | `logic/agentLoop.ts`, `store/agent.ts`, `AgentInput.vue` | Retry logic + error state |
| 7 | `locales/en-US.json`, `locales/zh-CN.json`, `AgentChat.vue` | i18n keys |
| 8 | `logic/tools/index.ts` | Updated `get_page_state` description |
| 9 | — | Lint + type-check + build |
| 10 | — | Manual integration test |
