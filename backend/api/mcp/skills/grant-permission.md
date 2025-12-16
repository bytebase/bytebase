---
name: grant-permission
description: Use when granting roles, managing access control, adding users to projects, or setting up RBAC permissions
---

# Grant Permission

## Overview

Grant roles to users or groups using Bytebase's RBAC system. Permissions can be set at workspace level (global) or project level (scoped).

## Prerequisites

- Have `bb.workspaces.setIamPolicy` for workspace-level grants
- Have `bb.projects.setIamPolicy` for project-level grants

## Permission Levels

| Level | Scope | Use Case |
|-------|-------|----------|
| **Workspace** | All projects | Admins, DBAs, global viewers |
| **Project** | Single project | Developers, project owners |

## Workflow

### Step 1: List Available Roles

```
search_api(operationId="RoleService/ListRoles")
```
```
call_api(operationId="RoleService/ListRoles", body={})
```

**Built-in roles:**

| Role | Description |
|------|-------------|
| `roles/workspaceAdmin` | Full workspace access |
| `roles/workspaceDBA` | Database administration |
| `roles/workspaceMember` | Basic workspace access |
| `roles/projectOwner` | Full project access |
| `roles/projectDeveloper` | Development access (create issues, plans) |
| `roles/projectReleaser` | Execute rollouts |
| `roles/sqlEditorUser` | SQL Editor access only |
| `roles/projectViewer` | Read-only access |
| `roles/gitopsServiceAgent` | GitOps automation |

### Step 2: Get Current IAM Policy

Always fetch current policy first to avoid overwriting existing bindings.

**Workspace level:**
```
search_api(operationId="WorkspaceService/GetIamPolicy")
```
```
call_api(operationId="WorkspaceService/GetIamPolicy", body={
  "resource": "workspaces/-"
})
```

**Project level:**
```
search_api(operationId="ProjectService/GetIamPolicy")
```
```
call_api(operationId="ProjectService/GetIamPolicy", body={
  "resource": "projects/{project-id}"
})
```

Save the returned `etag` for the update.

### Step 3: Set IAM Policy

Add new bindings while preserving existing ones.

**Workspace level:**
```
search_api(operationId="WorkspaceService/SetIamPolicy")
```
```
call_api(operationId="WorkspaceService/SetIamPolicy", body={
  "resource": "workspaces/-",
  "etag": "{etag-from-get}",
  "policy": {
    "bindings": [
      {
        "role": "roles/workspaceDBA",
        "members": ["user:dba@example.com"]
      },
      {
        "role": "roles/workspaceMember",
        "members": [
          "user:dev1@example.com",
          "user:dev2@example.com",
          "group:developers@example.com"
        ]
      }
    ]
  }
})
```

**Project level:**
```
call_api(operationId="ProjectService/SetIamPolicy", body={
  "resource": "projects/{project-id}",
  "etag": "{etag-from-get}",
  "policy": {
    "bindings": [
      {
        "role": "roles/projectOwner",
        "members": ["user:lead@example.com"]
      },
      {
        "role": "roles/projectDeveloper",
        "members": ["group:backend-team@example.com"]
      }
    ]
  }
})
```

## Member Format

| Type | Format | Example |
|------|--------|---------|
| User | `user:{email}` | `user:alice@example.com` |
| Group | `group:{email}` | `group:devs@example.com` |
| All users | `allUsers` | `allUsers` |

## Time-Limited Access (CEL Expressions)

Grant temporary access using CEL conditions.

```
call_api(operationId="ProjectService/SetIamPolicy", body={
  "resource": "projects/{project-id}",
  "etag": "{etag}",
  "policy": {
    "bindings": [{
      "role": "roles/projectDeveloper",
      "members": ["user:contractor@example.com"],
      "condition": {
        "expression": "request.time < timestamp('2024-12-31T23:59:59Z')",
        "title": "Temporary access",
        "description": "Access expires end of 2024"
      }
    }]
  }
})
```

**CEL variables for sqlEditorUser:**
- `resource.database`: Database name (`instances/{id}/databases/{name}`)
- `resource.schema_name`: Schema name
- `resource.table_name`: Table name
- `request.time`: For expiration

**CEL examples:**
- Expire at date: `request.time < timestamp('2024-12-31T23:59:59Z')`
- Limit to database: `resource.database == "instances/prod/databases/main"`

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| etag mismatch | Policy changed | Re-fetch policy and retry |
| role not found | Invalid role name | List roles first, use `roles/{name}` format |
| invalid member | Wrong format | Use `user:email` or `group:email` format |
| permission denied | Missing setIamPolicy | Check workspace/project admin access |
| invalid CEL | Bad expression syntax | Validate CEL expression format |
