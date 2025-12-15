---
name: create-instance
description: Add a database instance to Bytebase
---

# Create Instance

## Overview

Add a database instance (PostgreSQL, MySQL, etc.) to Bytebase for management.

## Prerequisites

- Have `bb.instances.create` permission
- Know the database engine type and connection details

## Workflow

1. **Get the schema**:
   ```
   search_api(operationId="InstanceService/CreateInstance")
   ```

2. **List environments** (to get environment resource name):
   ```
   call_api(operationId="EnvironmentService/ListEnvironments", body={})
   ```

3. **Create the instance**:
   ```
   call_api(operationId="InstanceService/CreateInstance", body={
     "parent": "projects/{project-id}",
     "instance": {
       "title": "Production PostgreSQL",
       "engine": "POSTGRES",
       "environment": "environments/prod",
       "activation": true,
       "dataSources": [{
         "type": "ADMIN",
         "host": "localhost",
         "port": "5432",
         "username": "admin",
         "password": "secret"
       }]
     },
     "instanceId": "prod-pg"
   })
   ```

## Engine Types

POSTGRES, MYSQL, TIDB, CLICKHOUSE, SNOWFLAKE, SQLITE, MONGODB, REDIS, ORACLE, MSSQL, MARIADB, OCEANBASE

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| environment not found | Invalid environment name | List environments first |
| instance already exists | Duplicate instanceId | Choose different instanceId |
| connection failed | Wrong host/port/credentials | Verify connection details |
