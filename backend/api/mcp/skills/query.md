---
name: query
description: Use when running SQL queries, executing SELECT/INSERT/UPDATE/DELETE statements, or fetching data from databases
---

# Execute SQL

## Overview

Run SQL queries against databases managed by Bytebase.

## Prerequisites

- Know the instance and database name
- Have `bb.databases.query` permission

## Workflow

1. **Get the schema**:
   ```
   search_api(operationId="SQLService/Query")
   ```

2. **List databases in project** (if needed):
   ```
   call_api(operationId="DatabaseService/ListDatabases", body={
     "parent": "projects/{project-id}"
   })
   ```

3. **Execute SQL**:
   ```
   call_api(operationId="SQLService/Query", body={
     "name": "instances/{instance-id}/databases/{database-name}",
     "statement": "SELECT * FROM users LIMIT 10"
   })
   ```

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| database not found | Wrong instance/database name | List databases first |
| permission denied | Missing bb.databases.query | Check user permissions |
| syntax error | Invalid SQL | Check SQL syntax for the database engine |
