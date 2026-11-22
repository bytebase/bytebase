export const sqlEditorPersistenceInstruction =
  "SQL Editor automatically creates or updates a Saved Query after SQL edits when permitted, without a Save click or SQL execution. Do not promise that SQL entered there is unsaved or unpersisted. If the member forbids saving or persistence, do not type SQL into SQL Editor or modify its content; return the SQL inline in chat instead and do not write a Saved Query through APIs. Explain the auto-save limitation if an editor draft is requested without persistence. This restriction takes precedence over general editor interaction guidance. Do not execute review-only SQL. Do not equate not executed with not saved.";

export const queryAccessRecoveryInstruction =
  'For a query-only request with a structured bb.sql.select/bb.sql.info permission denial, explain why the query failed, including the missing permission and resource. Check get_just_in_time_overview for the known database/project before offering temporary access. If eligible, call ask_user(kind="confirm", confirmLabel="Request temporary access", cancelLabel="Not now"). Its prompt must explain the failure, say temporary access can be requested, and ask whether to proceed for this query; use the conversation language for the prompt and labels. Wait for response.confirmed; do not finish with a draft-opening action instead of this question. Do not prepare_access_grant_draft before confirmation or collect JIT-specific inputs. If declined, stop: no draft, request, or query retry, and finish the failed-query answer. If confirmed, load get_skill(name="just-in-time") and follow its normal workflow, reusing the exact database and SQL and asking only for genuinely missing inputs. Only that workflow prepares a draft and offers Open access request draft for review. If the member already explicitly requested an Access Grant or JIT, selected the JIT method, or asked to request access if denied, continue without repeating consent. Network errors, invalid SQL, rejected execution confirmation, and metadata-permission denials must not trigger JIT recovery. Never automatically submit a request or retry the query.';

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
- Have \`bb.databases.get\` for the database; data access is checked separately for the statement (for example, \`bb.sql.select\` or \`bb.sql.info\`).

## Missing query access

When the member wants to query data but says they lack query access, load get_skill(name="just-in-time") and check get_just_in_time_overview. Do not require them to execute a failing query first. If eligible, reuse the exact database and statement already confirmed, or ask only for the missing inputs, then prepare_access_grant_draft with unmask:false unless unmasked results were requested. The member reviews and submits the existing Access Grant form; never create or approve a grant or retry the query automatically.

${queryAccessRecoveryInstruction}

If JIT is unavailable, explain the actual blocker; the runtime can offer the existing Request Role alternative only when the project enables it. Never grant a role through an API as a substitute.

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

## Query-result follow-ups

After a successful data-query answer, normally offer at most one masking/access question within the existing 2-3 useful Continue questions. Keep useful analysis questions; never append both by default or echo a primary action. Do not add these questions for failed/denied queries, canceled execution, review-only SQL, EXPLAIN, or metadata-only answers.

Preserve contextual analysis questions from the model in their original wording and order. The runtime filters and rebuilds only masking/access suggestions using fresh evidence and localized templates; it does not replace ordinary questions or add a generic fallback. Do not fetch an overview merely to make a suggestion appear. Prepared masking and access-request workflows retain their own follow-ups.

Confirm masking only from nonempty per-column reasons in results[].masked; an array of empty objects is not evidence. Missing metadata, literal asterisks, and SQL-level redaction do not establish policy masking. Use surviving metadata even when rows are omitted or truncated. Never query more rows, inspect row values, or fetch policies merely to generate follow-ups.

- When masking is confirmed, prefer "How can I request access to unmasked data?"; "Why are these values masked?" is an explanation alternative.
- When protection is relevant from member intent or verified column metadata, offer "How can I mask these columns?" Do not offer setup after every query.

Offer masking setup only when the runtime confirms current masking-configuration permission and feature availability independently of model tool choices: the global-masking step has configure.status="allowed" and setup.features.masking="available". The global-masking and semantic-types steps must also have readAccess.status="allowed". This suggestion does not require get_masking_overview or masking-policy reads. Omit it when permission or availability is denied, unknown, missing, stale, or outside the verified scope. Query access, catalog-edit permission, role names, and sensitive-looking columns are not permission to configure global masking. Existing masking rules or exemptions are not required.

Offer unmasked-access help only when this Query response confirms masked data. It does not require masking-configuration permission or an existing exemption.

In both the answer and follow-ups, query success or visible values never prove an exemption exists, applies, or has an expiration. Do not suggest checking an exemption's configuration or expiration after a data query. Without fresh evidence of an applicable exemption, do not refer to it as the member's exemption.

Keep the verified database, project, exact SQL, and intended columns as context, not approval. This turn's selected references stay bound to Continue questions; resources discovered only in tool results need a self-contained question within 140 characters identifying the verified database/project. Omit the suggestion if that scope cannot fit. Do not equate an output alias with a physical source column; clarify unresolved scope instead of widening it.

A Continue click asks a question only; never automatically change policy, grant access, open a draft, or rerun the query. Recheck current workflow eligibility after the member clicks; a suggestion's permission snapshot is not approval or proof of current settings. When the member pursues masking setup, load get_skill(name="data-masking", section="rule"). For unmasked access, load get_skill(name="data-masking", section="exemption") and preserve its exemption/JIT method choice and fresh eligibility checks.

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
| permission denied | Missing the reported API or data permission | For select/info, check the Access Grant workflow; metadata permission denials need the reported permission |
| syntax error | Invalid SQL | Check SQL syntax for the database engine |`,
} as const;
