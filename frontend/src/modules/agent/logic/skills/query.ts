export const sqlEditorPersistenceInstruction =
  "SQL Editor automatically creates or updates a Saved Query after SQL edits when permitted, without a Save click or SQL execution. Do not promise that SQL entered there is unsaved or unpersisted. If the member forbids saving or persistence, do not type SQL into SQL Editor or modify its content; return the SQL inline in chat instead and do not write a Saved Query through APIs. Explain the auto-save limitation if an editor draft is requested without persistence. This restriction takes precedence over general editor interaction guidance. Do not execute review-only SQL. Do not equate not executed with not saved.";

export const skill = {
  name: "query",
  description:
    "Use when reviewing or running SQL queries, executing SELECT/INSERT/UPDATE/DELETE statements, or fetching data from databases",
  content: `# Execute SQL

## Overview

Run SQL queries against databases managed by Bytebase.

## Review-only requests

${sqlEditorPersistenceInstruction}

## Prerequisites

- Know the instance and database name
- Have \`bb.sql.query\` permission

## Execution workflow (only when requested)

1. **Get the schema**:
   \`\`\`
   search_api(operationId="SQLService/Query")
   \`\`\`

2. **Find databases** (if needed):
   \`\`\`
   call_api(operationId="DatabaseService/SearchDatabases", body={
     "filter": "name.contains(\\"db_name\\")"
   })
   \`\`\`

   SearchDatabases searches across projects where the member can get databases. In a project context, add \`project == "projects/{project-id}"\` to the filter to limit results to that project.

   Filter examples:
   - \`name.contains("employee")\` - database name contains "employee"
   - \`project == "projects/{project-id}"\` - databases in a project
   - \`instance == "instances/{instance-id}"\` - databases in a workspace instance
   - \`instance == "projects/{project-id}/instances/{instance-id}"\` - databases in a project instance
   - \`engine == "MYSQL"\` - MySQL databases only
   - \`environment == "environments/prod" && name.contains("user")\` - combine filters

   **Optionally inspect \`instanceResource.dataSources\`** if you need to target a specific data source. Prefer \`type: "READ_ONLY"\` over \`type: "ADMIN"\` when choosing explicitly.

3. **Execute SQL**:
   \`\`\`
   call_api(operationId="SQLService/Query", body={
     "name": "projects/{project-id}/instances/{instance-id}/databases/{database-name}",
     "statement": "SELECT * FROM users LIMIT 10"
   })
   \`\`\`

   Include \`dataSourceId\` only when you need to override the server-selected data source.

## Notes

- Query results may contain masked values due to data masking policies. Do not remove or modify masked values.
  - Full mask: \`******\`
  - Partial mask: \`**rn**\` (only "rn" visible)
- **Displaying partial masks:** Use backticks or code blocks when presenting results. Without escaping, markdown interprets \`**text**\` as bold formatting.
- Preserve the exact canonical database name returned by SearchDatabases. Workspace databases use \`instances/{instance}/databases/{database}\`; project-instance databases use \`projects/{project}/instances/{instance}/databases/{database}\`.

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| database not found | Wrong instance/database name | List databases first |
| permission denied | Missing bb.sql.query | Check user permissions |
| syntax error | Invalid SQL | Check SQL syntax for the database engine |`,
} as const;
