# SQL Editor module

Follow `../../../AGENTS.md`. This directory owns the SQL Editor application subsystem used by the `/sql-editor` routes.

## Ownership map

```text
src/modules/sql-editor/
├── components/  # Route shell, panes, editor, results, schema browser, and UI
├── hooks/       # SQL Editor hooks backed by the app and editor stores
├── store/       # Zustand editor, tab, worksheet, tree, history, and terminal state
├── model/       # Framework-neutral worksheet tree and event models
└── legacy/      # One-time persisted-data migration only
```

- Route definitions live in `src/app/router/routes/sqlEditor.tsx`; this module owns the route implementation.
- Keep SQL Editor state under `store/`. Do not add state under the global `stores/` tree unless it is genuinely used outside SQL Editor.
- Keep SQL Editor UI under `components/`; use `@/components/ui` for shared primitives.
- `legacy/` may read old persisted formats but must not become a home for new runtime behavior.
- The current React and Zustand imports are authoritative. Some parity comments retain historical implementation context; they do not describe active framework boundaries.
- The AI module may consume SQL Editor hooks and state. Avoid the reverse dependency unless the UI is explicitly an AI integration surface.
- Run focused tests under this module, then the standard frontend fix, check, type-check, and full test commands.
