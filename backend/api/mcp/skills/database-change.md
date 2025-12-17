---
name: database-change
description: Use when making schema changes (DDL), data migrations, ALTER TABLE, CREATE TABLE, or deploying SQL changes through review workflow
---

# Database Change

## Overview

Create database changes (DDL/DML) through Bytebase's review workflow. Supports single database or batch changes across multiple databases.

## Prerequisites

- Have `bb.plans.create`, `bb.issues.create`, `bb.rollouts.create` permissions
- Know the target database(s)

## Workflow

### Step 1: Create sheet(s) with SQL

SQL content must be **base64 encoded**. Engine field is **required**.

```
search_api(operationId="SheetService/CreateSheet")
```
```
call_api(operationId="SheetService/CreateSheet", body={
  "parent": "projects/{project-id}",
  "sheet": {
    "title": "Add users table",
    "engine": "POSTGRES",
    "content": "Q1JFQVRFIFRBQkxFIHVzZXJzIChpZCBJTlQgUFJJTUFSWSBLRVkpOw=="
  }
})
```

Note: `Q1JFQVRFIFRBQkxFIHVzZXJzIChpZCBJTlQgUFJJTUFSWSBLRVkpOw==` decodes to `CREATE TABLE users (id INT PRIMARY KEY);`

Use `search_api(schema="Engine")` to discover valid engine values.

### Step 2: Create a plan

Plan contains `specs` directly (not wrapped in "steps").

**Single database:**
```
search_api(operationId="PlanService/CreatePlan")
```
```
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
```

**Batch change (multiple databases):**
```
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
```

**Key concepts:**
- `specs` is a flat array directly on the plan (no "steps" wrapper)
- Each `spec` has a unique `id` (any string, used for identification)
- `targets` is an **array** (even for single database)
- Specs are executed based on rollout policy (can be sequential or parallel)

**Using database groups:**
```
"targets": ["projects/{project-id}/databaseGroups/{group-name}"]
```

### Step 3: Create an issue

```
search_api(operationId="IssueService/CreateIssue")
```
```
call_api(operationId="IssueService/CreateIssue", body={
  "parent": "projects/{project-id}",
  "issue": {
    "title": "Add users table",
    "type": "DATABASE_CHANGE",
    "plan": "projects/{project-id}/plans/{plan-id}"
  }
})
```

### Step 4: Create a rollout

```
search_api(operationId="RolloutService/CreateRollout")
```
```
call_api(operationId="RolloutService/CreateRollout", body={
  "parent": "projects/{project-id}",
  "rollout": {
    "plan": "projects/{project-id}/plans/{plan-id}"
  }
})
```

## Change Types

| Type | Use Case |
|------|----------|
| `MIGRATE` | Imperative schema/data changes (DDL and DML) |
| `SDL` | State-based declarative schema migration |

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| database not found | Wrong database reference | Verify `instances/{id}/databases/{name}` format |
| sheet not found | Sheet doesn't exist | Create sheet first (Step 1) |
| missing engine | Sheet without engine field | Add `engine` field to sheet |
| plan not found | Plan doesn't exist | Create plan before issue |
| invalid base64 | SQL not encoded | Base64 encode the SQL content |
| targets must be array | Using string instead of array | Wrap target in array: `["..."]` |
