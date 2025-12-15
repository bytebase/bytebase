---
name: schema-change
description: Create schema migration issues
---

# Schema Change

## Overview

Create a schema change to modify database structure (DDL) through Bytebase's review workflow.

## Prerequisites

- Have `bb.plans.create`, `bb.issues.create`, `bb.rollouts.create` permissions
- Know the target database

## Workflow

1. **Create a sheet with the SQL** (content is base64 encoded):
   ```
   search_api(operationId="SheetService/CreateSheet")
   ```
   ```
   call_api(operationId="SheetService/CreateSheet", body={
     "parent": "projects/{project-id}",
     "sheet": {
       "title": "Add users table",
       "content": "Q1JFQVRFIFRBQkxFIHVzZXJzIChpZCBJTlQgUFJJTUFSWSBLRVksIG5hbWUgVkFSQ0hBUigyNTUpKTs="
     }
   })
   ```
   Note: `content` is base64 encoded. The example above decodes to:
   `CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));`

2. **Create a plan**:
   ```
   search_api(operationId="PlanService/CreatePlan")
   ```
   ```
   call_api(operationId="PlanService/CreatePlan", body={
     "parent": "projects/{project-id}",
     "plan": {
       "title": "Add users table",
       "steps": [{
         "specs": [{
           "changeDatabaseConfig": {
             "target": "instances/{instance-id}/databases/{database-name}",
             "type": "MIGRATE",
             "sheet": "projects/{project-id}/sheets/{sheet-id}"
           }
         }]
       }]
     }
   })
   ```

3. **Create an issue**:
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

4. **Create a rollout** (to execute after approval):
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

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| database not found | Wrong database reference | Verify database resource name |
| sheet not found | Sheet doesn't exist | Create sheet first |
| plan not found | Plan doesn't exist | Create plan before issue |
| issue not approved | Issue needs approval | Wait for approval before rollout |
