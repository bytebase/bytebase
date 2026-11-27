# SQL Editor module

Follow `../../../AGENTS.md`. This directory owns the SQL Editor application subsystem used by the `/sql-editor` routes.

## Ownership map

```text
src/modules/sql-editor/
├── components/  # Route shell, panes, editor, results, schema browser, and UI
├── hooks/       # SQL Editor hooks backed by the app and editor stores
├── store/       # Zustand editor, tab, saved query, tree, history, and terminal state
├── model/       # Framework-neutral saved query tree and event models
└── legacy/      # One-time persisted-data migration only
```

- Route definitions live in `src/app/router/routes/sqlEditor.tsx`; this module owns the route implementation.
- Keep SQL Editor state under `store/`. Do not add state under the global `stores/` tree unless it is genuinely used outside SQL Editor.
- Keep SQL Editor UI under `components/`; use `@/components/ui` for shared primitives.
- `legacy/` may read old persisted formats but must not become a home for new runtime behavior.
- The current React and Zustand imports are authoritative. Some parity comments retain historical implementation context; they do not describe active framework boundaries.
- `SQLEditorAgentHost` owns the dock slot and native editor adapter. The root-mounted page Agent remains the sole chat and execution owner; do not mount another Agent window or input here.
- SQL-specific entry points pass bounded explicit attachments through the shared Agent context path. Native edits require a member click and a matching mounted editor target; they are not registered DOM actions.
- The connection picker and SQL header profile menu temporarily minimize the Agent through `useSqlEditorAgentOverlay`. Their cleanup restores only unchanged visibility intent in the same identity; never raise ordinary overlays above the Agent family or stop a run to show them.
- Run focused tests under this module, then `pnpm --dir frontend fix` and `pnpm --dir frontend test`.
