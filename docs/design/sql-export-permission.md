# Restore `bb.sql.export` and remove `disable_export`

- Status: proposal for review
- Date: 2026-10-28
- Tracking issue: TBD
- Decision record: [ADR 0006](../adr/0006-control-export-with-a-permission.md)
- Line references point at `afffc8914a`.

## Decision

- **Who** may export is controlled by `bb.sql.export` alone. `SQLService.Export` requires it on
  the data it reads, checked with the same database, schema, table and expiration conditions as
  `bb.sql.select`. Export does not also require `bb.sql.select`.
- A user holds `bb.sql.export` through a role binding (standing) or through an active access
  grant with `export: true` for the exact query (temporary, approved).
- `disable_export` is removed. The workspace setting disappears from the UI, the server stops
  storing the value, and the migration strips it from stored policies. The v1 field stays,
  deprecated, until a later release deletes it: reads return `false`, and writes that set it to
  `true` are rejected with a message that points to export roles.
- A new built-in role, **SQL Editor Exporter** (`roles/sqlEditorExporter`), holds SQL Editor Read
  User's permissions plus `bb.sql.export`. Workspace Admin, Workspace DBA and Project Owner also
  hold `bb.sql.export`. SQL Editor User and SQL Editor Read User do not.
- The server enforces the permission for every caller: console, API and MCP.
- A one-time migration keeps export for users who export today through SQL Editor User or SQL
  Editor Read User. Workspaces that allow export without approval give those users a SQL Editor
  Exporter binding. Workspaces that require approval get no new standing export, so approved
  access grants stay the only export path for ordinary users. The migration does not change
  custom roles: holders of custom roles with `bb.sql.select` lose export until an administrator
  adds `bb.sql.export` to the role.

## Background

Bytebase removed `bb.sql.export` and `roles/projectExporter` in #17470. Today:

- `Export` is gated at the interceptor by `bb.databases.get` only
  (`proto/v1/v1/sql_service.proto:89`). Past that, it runs the same per-statement check as
  `Query` (`backend/api/v1/sql_service.go:1629`), so anyone who can run a `SELECT` can export
  its result.
- `disable_export` is a workspace-level flag, labeled "Allow data export without approval" in
  workspace settings. Turning that toggle off sets the flag. With the flag set, the SQL Editor
  hides the Export button unless the user holds an active export grant for the query, and offers
  "Request export" instead (`frontend/src/modules/sql-editor/components/ResultView/ResultView.tsx:154`,
  `ResultPanel/BatchQuerySelect.tsx:167`). There is no administrator exemption. The flag therefore
  means "export requires approval", and it is only a UI rule: the backend never reads it, so the
  API and MCP export with it set. The 2026-09 API audit lists this as X1
  (`docs/design/v1-api-audit-2026-09.md:116`).
- An access grant with `export: true` skips the role check on its target databases
  (`sql_service.go:1176`, `:1564`). The `export` field exists because there was no export
  permission for the grant to stand in for.

An administrator therefore cannot say "this person may query but not export". The one export
control that exists applies to every user at once and is not enforced. Once `bb.sql.export`
exists, "export requires approval" is expressed by assigning roles without it, and the flag has
nothing left to decide.

## Goals

- Control who may export per person and per database, schema and table, like query access.
- Enforce it on the server for every caller.
- Keep one control for export: role bindings and access grants, with no workspace flag that
  overlaps them.
- On upgrade, keep export for users who export through SQL Editor User or SQL Editor Read User:
  standing export where the workspace allows export without approval, and approval-only export
  where it requires approval. Holders of custom roles with `bb.sql.select` lose export until an
  administrator adds `bb.sql.export` to the role.

## Non-goals

- A workspace switch that turns off all export. If customers ask for one, it gets a new field;
  `disable_export` is not reused.
- Non-local export destinations and their settings, such as `disable_local_export`.
- Per-export row limits as a role condition (`request.row_limit` in the 2025 model). Row and
  size caps stay in `QueryDataPolicy`.
- Changing how `Query` is authorized.
- Downgrading after the migration. It is not supported.

## Product behavior

### Who can export

| Role | `bb.sql.export` | Reason |
|---|---|---|
| Workspace Admin, Workspace DBA, Project Owner | Yes | All three hold `bb.projects.setIamPolicy` and could grant it to themselves. Withholding it would be cosmetic. |
| SQL Editor User, SQL Editor Read User | No | Export becomes an explicit grant. |
| SQL Editor Exporter (new) | Yes | "Query databases and export results, without DDL or DML access." |
| Custom roles | As configured | The migration does not change custom roles. |

SQL Editor Exporter can be stacked on SQL Editor User to add export to someone who can write.

A custom role with `bb.sql.export` but no `bb.sql.select` is valid. Its holder can export through
the API or MCP but not from the SQL Editor, where Export works on a query result.

To require approval for export, an administrator assigns roles without `bb.sql.export`. Users
then request export through an access grant in projects with just-in-time access enabled
(Enterprise plan).

### Removing the workspace setting

- The workspace SQL Editor setting "Allow data export without approval" is removed.
- `GetPolicy` returns `disable_export = false` for every query data policy.
- `UpdatePolicy` accepts `disable_export = false` and ignores it. It rejects `true` with
  `INVALID_ARGUMENT`: "disable_export is removed. Export requires the bb.sql.export permission;
  to require approval for export, assign roles without it." A stale Terraform config or script
  therefore fails at apply with instructions, instead of showing a diff that never converges.

### SQL Editor

| Holds `bb.sql.export` (role or matching grant) | Project allows just-in-time access | Result panel shows |
|---|---|---|
| Yes | Either | Export |
| No | Yes | Request export |
| No | No | Neither |

The SQL Editor checks `bb.sql.export` per database, including database-scoped binding conditions.
Schema and table conditions are checked only by the server, and a denied export shows its error.

### Access grants

An approved grant with `export: true` gives `bb.sql.export` for one read-only query on its target
databases until it expires. Requests follow existing rules:

- Creating a request needs `bb.accessGrants.create` (held by Workspace Member, so every user) and
  a project with just-in-time access enabled.
- Approval rules can route export requests with `request.data_export`.
- Activating a grant directly needs `bb.accessGrants.activate`. Only Admin, DBA and Project Owner
  hold it, and all three already hold `bb.sql.export`.

### Assigning roles

- Member forms offer database scoping for roles with `bb.sql.export`, as they do for
  `bb.sql.select`.
- Workspace setup gives invitees SQL Editor Exporter instead of SQL Editor Read User, alongside
  Project Developer, so invitees in new workspaces export as they do today
  (`frontend/src/routes/auth/workspace-setup/finish.ts:90`).
- "Request query" keeps requesting SQL Editor Read User.

## Technical design

### Permission and roles

- Add `SQLExport = "bb.sql.export"` to `backend/common/permission/permission.go` and
  `permission.yaml`. Custom-role validation reads the YAML, so `CreateRole` and `UpdateRole`
  accept it with no further change.
- In `backend/store/predefined_roles.go`, add `SQLExport` to Workspace Admin, Workspace DBA and
  Project Owner, and add `SQLEditorExporterRole = "sqlEditorExporter"`.
- The frontend permission union is generated from the YAML
  (`frontend/src/types/iam/permission-generated.ts`).

### Export check

The per-statement access check runs with an export flag, captured in the `accessCheckWithGrant`
closure so `accessCheckFunc` keeps its signature.

| Statement type | `Query` (unchanged) | `Export` |
|---|---|---|
| Select, including column-level checks | `bb.sql.select` | `bb.sql.export` |
| Info schema (`SHOW` and similar) | `bb.sql.info` | `bb.sql.export` |
| Unknown on engines without the new ACL (select fallback) | `bb.sql.select` | `bb.sql.export` |
| No spans (engines without a span extractor, `sql_service.go:1613`) | `bb.sql.select` | `bb.sql.export` |
| Explain | `bb.sql.explain` | `bb.sql.explain` |
| DML, DDL | `bb.sql.dml`, `bb.sql.ddl` | Unchanged |

Explain keeps `bb.sql.explain` because its output is a plan, not table data, and grants still do
not cover it. `EXPLAIN ANALYZE` checks its statement, so its reads require `bb.sql.export`.

On engines that classify statements, DML and DDL keep their own permissions, so `Export` does
not become a way to change data. This matters on MySQL, where `Export` skips read-only
validation (`sql_service.go:1180`). Engines with neither statement classification nor a
validator are an existing exception: on Databricks, a write produces no spans and passes the
database-level select check through both `Query` and `Export` today
(`docs/design/mcp-capability-ladder.md:160`). Under this design `Export` checks `bb.sql.export`
there, so an export-only custom role can run such a write through `Export`, as a select role can
today. For an MCP session, `Export` refuses every statement Bytebase cannot show is a read,
Databricks included (`mcp_sql_write.go`). The console keeps the exception.

The CEL input is unchanged: database, schema and table per source column (`sql_service.go:1756`),
and the database-level fallback. `hasDatabaseAccessRights` strips the environment condition for
non-DDL/DML permissions today, and `bb.sql.export` is treated the same way.

An export grant satisfies `bb.sql.export` on its target databases. It is looked up with
`requireExport=true`, as today. The code should express this as "the grant satisfies
`bb.sql.export`" rather than "the grant skips the role check", so a later export rule applies to
grant holders too.

### API and docs

- `proto/v1/v1/org_policy_service.proto`: mark `disable_export` `[deprecated = true]` with the comment
  "Deprecated: export is controlled by the bb.sql.export permission. Always false; setting it to
  true is rejected." `convertToQueryDataPolicyPayload` rejects `true` and drops the value. Delete
  the field in a later release, after the provider stops sending it.
- `proto/store/store/policy.proto`: delete `disable_export` and reserve field 2 and its name. Remove
  `DisableExport` from `store.EffectiveQueryDataPolicy` and the v1 converter.
- `proto/v1/v1/sql_service.proto`: document that `Export` requires `bb.sql.export`.
- `proto/v1/v1/access_grant_service.proto`: the `export` field "grants `bb.sql.export` for this
  query until the grant expires."
- `proto/v1/v1/iam_policy.proto`: list export in the roles that `resource.*` conditions apply to.
- Remove the stale `roles/EXPORTER` examples in `issue_service.proto` and `store/issue.proto`,
  and the "EXPORTER role" comment at `sql_service.go:1227`.
- Mark X1 resolved and update line 680 of `docs/design/v1-api-audit-2026-09.md`.
- `Export` stays a WRITE-class MCP method. The export row in
  `docs/design/mcp-capability-ladder.md` should say the user's own `bb.sql.export` applies.

### Frontend

- Export gates in `ResultView.tsx` and `BatchQuerySelect.tsx` follow the SQL Editor table above.
  Standing export is decided per database with `isDatabaseV1Queryable(database, ["bb.sql.export"])`,
  and `useExportGrantBypass` runs for the databases that lack it, instead of when the flag is set.
- Remove the toggle from `routes/workspace/general/SQLEditorSection.tsx`, its "data-export"
  strings in all five locales (`frontend/src/locales/en-US.json`), and its tests.
- Remove `disableExport` from `useSQLEditorQueryDataPolicy`
  (`modules/sql-editor/hooks/useSQLEditorState.ts`).
- Add the new role to `types/iam/role.ts`, `lib/role.ts` and `utils/role.ts`, with strings in all
  five locales.
- Add `roles/sqlEditorExporter` to the role table in the agent's grant-permission skill
  (`modules/agent/logic/skills/grantPermission.ts`), so the agent can grant export.
- Add `bb.sql.export` to `roleHasDatabaseLimitation` in `lib/project-member/utils.ts`.
- Replace SQL Editor Read User with SQL Editor Exporter in `INVITEE_PROJECT_ROLES`
  (`routes/auth/workspace-setup/finish.ts`).

## Migration

One SQL file, `backend/migrator/migration/3.25/0000##sql_export_permission.sql`; 3.24.0 shipped
with `3.24/0000` to `0002`. Update `TestLatestVersion`. `LATEST.sql` does not change because
role permissions, IAM bindings and policies are JSONB. The migrator runs the file in one
transaction (`backend/migrator/migrator.go:227`).

A workspace is **approval-only** when its workspace-level `QUERY_DATA` policy stores
`disableExport = true`, whether or not it is enforced: `GetPolicy` ignores `enforce`
(`backend/store/policy.go:751`), so the SQL Editor honored the value either way. Every other
workspace is **export-enabled**.

Steps:

1. Fail if a custom role already uses the id `sqlEditorExporter`. Role ids are unique across
   workspaces (`LATEST.sql:98`), and built-in roles shadow custom roles with the same id
   (`backend/store/role.go:152`), so continuing would silently change that role's meaning. The
   new server cannot start until the conflict is gone, and a role id cannot be renamed, so the
   error message gives the fix: on the previous version, create a replacement custom role, move
   its bindings, and delete the conflicting role. Check Cloud data for the id before release.
2. In export-enabled workspaces, for every workspace or project IAM binding of SQL Editor User
   or SQL Editor Read User, add a SQL Editor Exporter binding with the same members and
   condition.
3. Remove the `disableExport` key from every `QUERY_DATA` policy, at workspace and project
   level, enforced or not.

Approval-only workspaces get no new bindings. Ordinary users there keep exporting only through
approved grants. Custom roles are not changed in any workspace. Built-in role definitions live in
Go and need no migration.

```sql
DO $$
DECLARE
  conflicting_workspace text;
BEGIN
  SELECT workspace INTO conflicting_workspace FROM role WHERE resource_id = 'sqlEditorExporter';
  IF FOUND THEN
    RAISE EXCEPTION 'custom role "sqlEditorExporter" in workspace "%" conflicts with a new built-in role. On the previous Bytebase version, create a replacement custom role, move its bindings, and delete "sqlEditorExporter".', conflicting_workspace;
  END IF;
END $$;

CREATE TEMP TABLE approval_only_workspace ON COMMIT DROP AS
SELECT workspace FROM policy
WHERE resource_type = 'WORKSPACE' AND type = 'QUERY_DATA'
  AND COALESCE((payload ->> 'disableExport')::boolean, false);

WITH added AS (
  SELECT p.workspace, p.resource_type, p.resource,
         jsonb_agg(jsonb_set(b, '{role}', '"roles/sqlEditorExporter"') ORDER BY ord) AS bindings
  FROM policy p,
       jsonb_array_elements(p.payload -> 'bindings') WITH ORDINALITY AS t(b, ord)
  WHERE p.type = 'IAM'
    AND p.resource_type IN ('WORKSPACE', 'PROJECT')
    AND p.workspace NOT IN (SELECT workspace FROM approval_only_workspace)
    AND b ->> 'role' IN ('roles/sqlEditorUser', 'roles/sqlEditorReadUser')
  GROUP BY p.workspace, p.resource_type, p.resource
)
UPDATE policy p
SET payload = jsonb_set(p.payload, '{bindings}', (p.payload -> 'bindings') || added.bindings)
FROM added
WHERE p.type = 'IAM' AND p.workspace = added.workspace
  AND p.resource_type = added.resource_type AND p.resource = added.resource;

UPDATE policy
SET payload = payload - 'disableExport'
WHERE type = 'QUERY_DATA' AND payload ? 'disableExport';
```

Copying the binding and changing only the role carries members (users, groups, `allUsers`) and
the condition over exactly, including database, schema, table and expiration scope. A member who
holds both SQL Editor roles receives two exporter bindings, which is harmless. Access grants and
approval rules are not touched.

### Resulting behavior

| Who | Export-enabled workspace | Approval-only workspace |
|---|---|---|
| SQL Editor User, SQL Editor Read User | Export, through the added binding | Request export, then export after approval (unchanged)* |
| Custom roles with `bb.sql.select` | No export until an admin adds `bb.sql.export` to the role | Request export (unchanged)* |
| Admin, DBA, Project Owner | Export | **Export without a grant** |
| Export grant holders | Export | Export (unchanged) |
| API and MCP callers | Same as the SQL Editor | **Same as the SQL Editor** |

\* In projects with just-in-time access enabled (Enterprise plan). Elsewhere, Request export is not
offered, as today.

### Accepted behavior changes

- Workspace Admin, Workspace DBA and Project Owner hold `bb.sql.export` in every workspace. In
  export-enabled workspaces they could already export, so nothing changes. In approval-only
  workspaces they needed an approved grant, which they could approve themselves, and now export
  directly. The release note must state this.
- In approval-only workspaces, API and MCP callers need `bb.sql.export` or a matching export
  grant, where before anyone with select could export. This closes X1.
- In export-enabled workspaces, holders of custom roles with `bb.sql.select` lose export until an
  administrator adds `bb.sql.export` to the role. The release note must state this.
- The workspace export setting is gone. To stop a user exporting, remove export from their
  roles; to stop export through grants in a project, turn off just-in-time access there. The
  release note must state this.
- Writes that set `disable_export = true` are rejected.
- Custom roles with `bb.sql.info` or `bb.sql.explain` but not `bb.sql.select` could export `SHOW`
  output before and cannot after. Giving them `bb.sql.export` would grant export of all data,
  which is broader than what they had.
- New SQL Editor User or Read User assignments, and role requests approved after the upgrade, do
  not include export. Assign SQL Editor Exporter as well.

## Terraform and compatibility

The Terraform provider manages the whole IAM policy of a parent through `bytebase_iam_policy`,
custom roles and their permission sets through `bytebase_role`, and `disable_export` through
`bytebase_policy`. The migration changes IAM policies and query data policies, so
`bytebase_iam_policy` and `bytebase_policy` can drift. It does not change custom roles.

- **IAM drift (export-enabled workspaces).** `terraform plan` wants to remove the added exporter
  bindings. Applying it takes export away from SQL Editor users. This failure removes access and
  never adds it, which is why the migration adds bindings instead of rewriting roles.
- **`disable_export` (approval-only workspaces).** A config that sets `disable_export = true` shows a
  `false → true` change, and applying it fails with the rejection message. A config that sets
  `false` keeps applying until the provider drops the attribute.
- **Order of operations.** A config cannot reference `roles/sqlEditorExporter` before the server
  knows the role.

Provider release, shipped with the server release:

- Mark `disable_export` deprecated in the provider schema, stop sending it, and document its
  removal.
- Document the paired binding pattern below and remove the obsolete export row-limit condition.
- Confirm that role names are not validated against a fixed list.

Upgrade runbook for Terraform users:

1. Pause automatic `terraform apply`.
2. Upgrade Bytebase.
3. Run `terraform plan`.
4. In export-enabled workspaces, add every exporter binding the plan wants to remove: for each
   SQL Editor binding that should keep export, a binding with the same members and condition and
   `role = "roles/sqlEditorExporter"`.
5. Remove `disable_export` from every `bytebase_policy` config.
6. Resume automatic apply when the plan is clean.

```hcl
binding {
  role    = "roles/sqlEditorUser"
  members = ["group:analysts@example.com"]
  condition { database = "instances/prod/databases/app" }
}
binding {
  role    = "roles/sqlEditorExporter"
  members = ["group:analysts@example.com"]
  condition { database = "instances/prod/databases/app" }
}
```

## Testing

- **Migration** (`backend/migrator/migration_test.go`): one export-enabled and one approval-only
  workspace, each with a custom role with and without `bb.sql.select`, a conditioned SQL Editor
  User binding, a group SQL Editor Read User binding, and workspace and project IAM policies.
  Include a third workspace with a non-enforced disabled policy, and a project-level query data
  policy that stores `disableExport`. Assert exact payloads: the approval-only workspace has no
  new bindings; the export-enabled workspace has the added bindings; the third workspace is
  treated as approval-only; every custom role, with or without `bb.sql.select`, is unchanged in
  every workspace; no query data policy keeps a `disableExport` key, their other fields and
  `enforce` are unchanged. Assert that the id collision raises.
- **Export authorization** (`backend/api/v1`):
  - A select-only user cannot export.
  - A user with only `bb.sql.export`, scoped to one table, exports that table, is refused on
    another table, and cannot `Query`.
  - An export grant allows export without the permission.
  - A role with `bb.sql.export` but no `bb.sql.dml` cannot run DML through `Export` on MySQL.
  - `Query` is unaffected.
- **Policy API** (`backend/api/v1`): `UpdatePolicy` rejects `disable_export = true` and accepts
  `false`; `GetPolicy` returns `false`.
- **Roles** (`backend/store`): the new built-in role exists and its permissions are valid.
- **Frontend**: `ResultView`, `BatchQuerySelect`, `useExportGrantBypass`, `RequestExportButton`,
  `SQLEditorSection`, the project-member utils and workspace setup. `ResultView.test.tsx` mocks
  every permission check as false today, so its cases need updating.
- **Provider**: a migrated binding round-trips through `bytebase_iam_policy`, and its CEL condition
  parses back into the same `database`, `schema` and `tables` fields.

## Alternatives considered

### Keep `disable_export` as a switch that turns off all export

The field would keep its name and change meaning from "export requires approval" to "no export
for anyone", including administrators and grant holders, with per-destination settings added
beside it later. This keeps a one-click stop for all export. It also gives export two controls
whose overlap administrators must reason about, and it changes what an existing field means: a
stale config or script that writes `true`, meaning "approval required" before the upgrade, would
silently stop all export after it. A kill switch can come later as a new field if customers ask.

### Enforce `disable_export` on the server with its current meaning

With the flag set, only grant holders could export, for every caller. This closes X1 without a
migration. It keeps a workspace-wide rule that decides who may export beside the role bindings
that also decide it, so "can this person export" would depend on both, and it still could not
express "this person may query but not export" in a workspace that allows export.

### Ignore writes of `disable_export = true`

Accepting and dropping the value keeps old configs applying. `terraform plan` would then show a
`false → true` change on every run, and the administrator is never told the setting is gone.

### Give `bb.sql.export` in every workspace

The migration could add exporter bindings in approval-only workspaces too. That would hand
standing export to every SQL Editor user in workspaces that chose to require approval.

### Add `bb.sql.export` to custom roles with select

This keeps export for holders of custom roles in export-enabled workspaces. It also rewrites
roles that administrators own, often through Terraform `bytebase_role`, whose next plan would
want to undo the change.

### Add an Exporter binding beside custom-role bindings

This keeps export without touching the roles. It also grants the rest of SQL Editor Exporter's
permissions (database sync, IAM policy read, explain, info, saved queries, database groups and
catalogs), which a narrow custom role may not include.

### Require `bb.sql.select` and `bb.sql.export`

Export would only narrow select, so "who can read this table" stays one question. We chose export
alone because it gives administrators one permission to reason about for export, and it matches
the 2025 model. The cost is that `bb.sql.export` is itself a read grant.

### Remove export from access grants

Users who need export would request SQL Editor Exporter through role requests, so bindings alone
answer who can export. An approved role covers every query in its scope rather than one, which
is broader than today's per-query grant. It also removes the "Request export" flow, which is the
only export path in approval-only workspaces, and empties `request.data_export` approval rules.

### Swap roles, or create per-workspace role copies

Built-in roles carry meaning beyond their permissions. Approval flows route to
`roles/projectOwner` by name (`backend/store/workspace.go:128`), Workspace Admin lookup protects the
last admin (`backend/api/v1/user_service.go:495`), and "Request query" asks for a built-in role by
name. Copies would drift from the built-in definitions and require the custom-roles license. A
Terraform apply would also revert a swap.

### Bring back `roles/projectExporter`

Customers may have created a custom role with that id since 2025, and a built-in role would
shadow it. A new id also avoids confusion with the 2025 role, whose permission set differs from
SQL Editor Exporter.

## Open items

1. **Cloud id check.** Confirm before release that no Cloud workspace has a custom role with the
   id `sqlEditorExporter`.
2. **Provider role names.** Confirm that the provider does not validate role names against a
   fixed list.
3. **Opting out of export requests.** If a customer wants grants for query but not export, add a
   project setting such as "allow export in access grant requests". Not planned until asked.
