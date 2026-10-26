export const skill = {
  name: "database-change",
  description:
    "Use when making schema changes (DDL), data migrations, ALTER TABLE, CREATE TABLE, or deploying SQL changes through review workflow",
  content: `# Database Change

## Overview

Create database changes (DDL/DML) through Bytebase's review workflow. Supports single database or batch changes across multiple databases.

## Prerequisites

- Have \`bb.plans.create\` and \`bb.issues.create\` permissions to propose a change; manual deployment also requires \`bb.rollouts.create\`
- Know the target database(s)

## Workflow

### Step 1: Create sheet(s) with SQL

Pass plain SQL as Sheet.content; \`call_api\` encodes it as base64 for the API. Engine field is **required**.

\`\`\`
search_api(operationId="SheetService/CreateSheet")
\`\`\`
\`\`\`
call_api(operationId="SheetService/CreateSheet", body={
  "parent": "projects/{project-id}",
  "sheet": {
    "title": "Add users table",
    "engine": "POSTGRES",
    "content": "CREATE TABLE users (id INT PRIMARY KEY);"
  }
})
\`\`\`

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
        "targets": ["projects/{project-id}/instances/{instance-id}/databases/{database-name}"],
        "sheet": "projects/{project-id}/sheets/{sheet-id}",
        "type": "MIGRATE"
      }
    }]
  }
})
\`\`\`

**Batch change (multiple databases):**
\`\`\`
call_api(operationId="PlanService/CreatePlan", body={
  "parent": "projects/{project-id}",
  "plan": {
    "title": "Add users table to all databases",
    "specs": [
      {
        "id": "spec-dev",
        "changeDatabaseConfig": {
          "targets": ["instances/dev-pg/databases/mydb"],
          "sheet": "projects/{project-id}/sheets/{sheet-id}",
          "type": "MIGRATE"
        }
      },
      {
        "id": "spec-prod",
        "changeDatabaseConfig": {
          "targets": ["instances/prod-pg/databases/mydb"],
          "sheet": "projects/{project-id}/sheets/{sheet-id}",
          "type": "MIGRATE"
        }
      }
    ]
  }
})
\`\`\`

**Key concepts:**
- \`specs\` is a flat array directly on the plan (no "steps" wrapper)
- Each \`spec\` has a unique \`id\` (any string, used for identification)
- \`targets\` is an **array** (even for single database)
- Preserve each exact canonical database name returned by ListDatabases. Do not remove a \`projects/{project-id}\` instance parent.
- Specs are executed based on rollout policy (can be sequential or parallel)

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

### Step 4: Review and deploy

Verify the created issue and plan, including checks and target eligibility. Report any blockers instead of claiming that a proposal can be deployed. Do not create a rollout immediately after creating an issue: approval may be required, and an approved issue may create its rollout automatically.

Only if the user explicitly requests deployment, the approval policy permits it, and no rollout already exists, create one using the plan as parent. Omitting \`target\` creates stages for every eligible target.

\`\`\`
search_api(operationId="RolloutService/CreateRollout")
\`\`\`
\`\`\`
call_api(operationId="RolloutService/CreateRollout", body={
  "parent": "projects/{project-id}/plans/{plan-id}"
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
| database not found | Wrong database reference | Use the exact \`instances/{id}/databases/{name}\` or \`projects/{project}/instances/{id}/databases/{name}\` returned by ListDatabases |
| sheet not found | Sheet doesn't exist | Create sheet first (Step 1) |
| missing engine | Sheet without engine field | Add \`engine\` field to sheet |
| plan not found | Plan doesn't exist | Create plan before issue |
| targets must be array | Using string instead of array | Wrap target in array: \`["..."]\` |`,
} as const;
