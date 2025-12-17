---
name: query
description: Use when running SQL queries, executing SELECT/INSERT/UPDATE/DELETE statements, or fetching data from databases
---

# Execute SQL

## Overview

Run SQL queries against databases managed by Bytebase.

## Prerequisites

- Know the instance and database name
- Have `bb.sql.query` permission

## Workflow

1. **Get the schema**:
   ```
   search_api(operationId="SQLService/Query")
   ```

2. **List databases** (if needed):
   ```
   call_api(operationId="DatabaseService/ListDatabases", body={
     "parent": "workspaces/-",
     "filter": "name.matches(\"db_name\")"
   })
   ```

   Filter examples:
   - `name.matches("employee")` - database name contains "employee"
   - `project == "projects/{project-id}"` - databases in a project
   - `instance == "instances/{instance-id}"` - databases in an instance
   - `engine == "MYSQL"` - MySQL databases only
   - `environment == "environments/prod" && name.matches("user")` - combine filters

   **Extract `dataSourceId`** from `instanceResource.dataSources` in the response. Prefer `type: "READ_ONLY"` over `type: "ADMIN"` when available.

3. **Execute SQL**:
   ```
   call_api(operationId="SQLService/Query", body={
     "name": "instances/{instance-id}/databases/{database-name}",
     "dataSourceId": "{data-source-id}",
     "statement": "SELECT * FROM users LIMIT 10"
   })
   ```

   `dataSourceId` is **required**.

## Notes

- Query results may contain masked values due to data masking policies. Do not remove or modify masked values.
  - Full mask: `******`
  - Partial mask: `**rn**` (only "rn" visible)
- **Displaying partial masks:** Use backticks or code blocks when presenting results. Without escaping, markdown interprets `**text**` as bold formatting.

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| data source id is required | Missing dataSourceId field | Get dataSourceId from instanceResource.dataSources in database listing |
| database not found | Wrong instance/database name | List databases first |
| permission denied | Missing bb.sql.query | Check user permissions |
| syntax error | Invalid SQL | Check SQL syntax for the database engine |
