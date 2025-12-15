---
name: create-project
description: Set up a new Bytebase project
---

# Create Project

## Overview

Create a new project to organize databases and team members.

## Prerequisites

- Have `bb.projects.create` permission

## Workflow

1. **Get the schema**:
   ```
   search_api(operationId="ProjectService/CreateProject")
   ```

2. **Create the project**:
   ```
   call_api(operationId="ProjectService/CreateProject", body={
     "project": {
       "title": "My Project",
       "key": "MYPROJ"
     },
     "projectId": "my-project"
   })
   ```

3. **Add team members** (optional):
   ```
   search_api(operationId="ProjectService/SetProjectIamPolicy")
   ```

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| project already exists | Duplicate projectId | Choose different projectId |
| invalid key | Key format wrong | Use uppercase letters, 2-10 chars |
