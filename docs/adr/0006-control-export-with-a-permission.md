# Control export with a permission and remove disable_export

Design: [sql-export-permission.md](../design/sql-export-permission.md)

The Export Permission (`bb.sql.export`) is the only control over who may export, held through a role binding or an Access Grant. The workspace flag `disable_export`, which meant "export requires approval", is removed: requiring approval is now expressed by assigning roles without the permission, so the flag would be a second control for the same decision. If a switch that stops all export is ever needed, it gets a new field rather than a new meaning for `disable_export`.

The upgrade keeps export for SQL Editor users. Workspaces whose workspace-level policy sets `disable_export = true` get no new standing export, so approval stays the only export path for ordinary users. Every other workspace gets a SQL Editor Exporter binding beside each SQL Editor binding, so SQL Editor users keep export. Custom roles are not changed.

## Considered options

- **Keep `disable_export` and redefine it as "no export at all".** Rejected because export would have two overlapping controls, and a stale write of `true`, which meant "approval required", would silently stop all export.
- **Turn export off where the flag was `true`.** Rejected because those administrators chose approval-gated export.
- **Ignore writes of `disable_export = true`.** Rejected because a stale config would never learn the setting is gone. The server rejects them instead.

## Consequences

- Workspace Admin, Workspace DBA and Project Owner hold the Export Permission in every workspace. Where `disable_export` was `true`, they used to need an approved grant, which they could approve themselves, and now export directly. The release note must state this.
- No single switch stops all export in a workspace.
- Where `disable_export` was not set, holders of custom roles with `bb.sql.select` lose export until an administrator adds the Export Permission to the role. The release note must state this.
