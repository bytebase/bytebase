# SQL Editor React Migration — Stage 1 Design

**Date:** 2026-04-20
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal

Begin the Vue→React migration of the SQL Editor by establishing the React-island-in-Vue pattern at a non-route boundary, with one tiny leaf as the first proof. Every step preserves UX exactly.

## 2. Non-goals (Stage 1)

- Migrating Monaco, ResultView, AsidePanel, ConnectionPanel, TabList, or any panel under `EditorPanel/Panels/`.
- Refactoring `useSQLEditorContext` into a Pinia store. Deferred to the first stage where it is justified — see §9.
- Flipping the SQL Editor route or layout to React.
- Changing any Vue file that does not sit on the Welcome call path.

## 3. Architecture

The SQL Editor under `frontend/src/views/sql-editor/` is **147 Vue + 68 TS files**, organized as a single-page app rather than a multi-route subsystem. All five `/sql-editor/...` routes mount the same `SQLEditorPage.vue`, which is wrapped by `SQLEditorLayout.vue` and depends on deep `provide/inject` context (`useSQLEditorContext`, `Sheet` context, `TabList` context), Pinia stores, an Emittery event bus, and Monaco. There is no sub-route boundary to flip.

We adopt **bottom-up React islands inside Vue**:

- Keep the Vue orchestrator (`SQLEditorLayout`, `SQLEditorPage`, `SQLEditorHomePage`), all `provide/inject` contexts, all Pinia stores, and the Emittery bus.
- Embed React leaves inline inside the Vue tree using the existing `frontend/src/react/ReactPageMount.vue` (already supports component mounting, not just routes — `mount.ts` already maps `./components/auth/SessionExpiredSurface.tsx` alongside the `pages/*` globs).
- Each new React leaf is registered through a new `./components/sql-editor/*.tsx` glob in `mount.ts`.
- React leaves access Pinia stores and the router directly per the `2026-04-08-react-migration-playbook.md`. For Vue-`provide/inject`-only state (e.g., `showConnectionPanel`), the Vue parent passes a callback prop (Hybrid C — see §5).

This direction matches the existing migration playbook and lets every step be independently verifiable side-by-side. The cost is a one-time investment in registering a non-`pages/` glob in `mount.ts`; thereafter every leaf migration is mechanical.

## 4. Stage 1 work — migrate `EditorPanel/Welcome/Welcome.vue`

The first leaf is `frontend/src/views/sql-editor/EditorPanel/Welcome/Welcome.vue` (77 lines). It renders when no database connection exists: a `BytebaseLogo` plus two large square buttons ("Add a new instance" and "Connect to a database"), each gated by a permission. It is the smallest real component in the SQL Editor that exercises every cross-cutting concern we need to solve once: Pinia from React, router from React, permission checks, i18n, an extracted shared component, and exactly one Vue-context-coupled action.

### 4.1 New React files

| File | Purpose | Notes |
|---|---|---|
| `frontend/src/react/components/sql-editor/Welcome.tsx` | Replaces `Welcome.vue`. | Reads project + permissions from React-side hooks. Receives `onChangeConnection` callback as a prop (see §5). |
| `frontend/src/react/components/sql-editor/WelcomeButton.tsx` | Replaces local `Button.vue` (28 lines wrapping `NButton`). | Mirrors Vue's structure per the extract-common-component policy. Built on shadcn `Button` with a `cva` variant matching the NButton dimensions: height 7rem, min-width 7rem, vertical icon-over-label layout. Lives next to Welcome since it is not yet reused elsewhere. |
| `frontend/src/react/components/BytebaseLogo.tsx` | Replaces `frontend/src/components/BytebaseLogo.vue`. Used here without the `redirect` prop, but built fully so future React pages can reuse. | Reads `useWorkspaceV1Store` directly from React (via `useVueState`). Renders the workspace logo or the bundled SVG fallback (`@/assets/logo-full.svg`). The optional `redirect` branch uses React Router only when needed; Welcome does not pass it. |

### 4.2 Vue caller modified

The single live caller of `<Welcome />` is `frontend/src/views/sql-editor/EditorPanel/StandardPanel/StandardPanel.vue`:

- line 55: `<Welcome v-else />`
- line 98: `import Welcome from "../Welcome";`

(`EditorPanel/Welcome/index.ts` is just a re-export wrapper.)

Swap to:

```vue
<ReactPageMount v-else page="Welcome" :on-change-connection="changeConnection" />
```

`StandardPanel.vue` already destructures `useSQLEditorContext()` (currently for `showAIPanel`, `editorPanelSize`, `handleEditorPanelResize` at line 105-106). Add `showConnectionPanel` and `asidePanelTab` to that destructure and construct:

```ts
const changeConnection = () => {
  asidePanelTab.value = "SCHEMA";
  showConnectionPanel.value = true;
};
```

Replace the Vue `import Welcome from "../Welcome"` with `import ReactPageMount from "@/react/ReactPageMount.vue"` (or whichever import path is used elsewhere — there are existing precedents).

The `gotoInstanceCreatePage` action lives inside React (it is just a `router.push`, no Vue context dependency).

### 4.3 Mount infrastructure

`frontend/src/react/mount.ts`:

- Add `const sqlEditorComponentLoaders = import.meta.glob("./components/sql-editor/*.tsx");`
- Spread it into `pageLoaders`.
- Add `"./components/sql-editor"` to `pageDirs`.

Variable name `pageDirs` stays — renaming to `componentDirs` is a separate cleanup with no functional value.

### 4.4 Vue file deletion

After the `StandardPanel.vue` swap, the entire `frontend/src/views/sql-editor/EditorPanel/Welcome/` directory (`Welcome.vue` + `Button.vue` + `index.ts`) becomes orphaned because `StandardPanel.vue` will import `ReactPageMount` directly instead of `from "../Welcome"`. Per playbook §Deletion: run `rg "from.*EditorPanel/Welcome"` and `rg "import.*Welcome\.vue"` to confirm zero remaining callers, then delete the entire `Welcome/` subdirectory.

## 5. Data flow & context bridge for Welcome

| What Welcome needs | Source | How React gets it |
|---|---|---|
| Workspace logo / fallback SVG | `useWorkspaceV1Store` (Pinia) | Direct Pinia import + `useVueState` (read-only subscription) inside `BytebaseLogo`. |
| Project (for `bb.sql.select` permission scope) | `useSQLEditorStore` (Pinia, `project` ref) → `useProjectV1Store.getProjectByName` | Direct Pinia import + `useVueState`. |
| `bb.instances.create` workspace permission | `hasWorkspacePermissionV2` | `usePermissionCheck(["bb.instances.create"])` (React-native hook from playbook). |
| `bb.sql.select` project permission | `hasProjectPermissionV2(project, ...)` | `usePermissionCheck(["bb.sql.select"], { project })` — same hook with project scope. |
| Navigate to instance create | `router.push({ name: INSTANCE_ROUTE_DASHBOARD, hash: '#add' })` | Direct router import inside React (playbook-approved). |
| Open connection panel + switch to SCHEMA tab | `useSQLEditorContext()` — Vue `provide/inject` only | Vue parent computes `changeConnection = () => { asidePanelTab.value = 'SCHEMA'; showConnectionPanel.value = true }` and passes it as the `onChangeConnection` prop. This is the only prop-callback Welcome receives. |

**Why this surface:** Welcome touches Vue-context state in exactly one place (`changeConnection`). Everything else is Pinia-or-router, both of which React can call directly per the playbook. So we get a 1-prop interface — minimal coupling, no premature Pinia bridge for `useSQLEditorContext`.

## 6. Verification & UX parity

### 6.1 Per-leaf verification (run before opening the Stage 1 PR)

- `pnpm --dir frontend fix` — auto-format + lint
- `pnpm --dir frontend check` — CI-equivalent check
- `pnpm --dir frontend type-check` — covers both Vue (`vue-tsc`) and React (`tsconfig.react.json`) tsconfigs
- `pnpm --dir frontend test` — unit tests
- New focused unit tests:
  - `WelcomeButton.test.tsx` — variant rendering, click handler.
  - `Welcome.test.tsx` — renders both buttons when both permissions present, hides Add-Instance when missing `bb.instances.create`, hides Connect when missing `bb.sql.select`, calls `onChangeConnection` on click.
  - Pattern matches existing `FeatureAttention.test.tsx` and `dialog.test.tsx`.

### 6.2 UX parity verification (manual, in a running dev server)

- Open SQL Editor with no connection → Welcome renders with: BytebaseLogo at top, "Add a new instance" + "Connect to a database" buttons below in a wrapped flex.
- Side-by-side screenshot comparison vs. the Vue version for these states:
  - (a) workspace with custom logo
  - (b) workspace using fallback Bytebase SVG
  - (c) viewer-only role (only Connect button visible)
  - (d) admin role (both buttons visible)
  - (e) viewer with no `bb.sql.select` (no buttons at all)
- Click "Add a new instance" → routes to `/instance#add`.
- Click "Connect to a database" → opens the connection panel and switches the aside panel to SCHEMA tab (verifies the `onChangeConnection` callback wires correctly).
- Locale switch (English ↔ Chinese) → labels update without remount artifacts (the `ReactPageMount` locale-watch handler covers this; just confirm visually).

### 6.3 i18n

Welcome's strings (`sql-editor.add-a-new-instance`, `sql-editor.connect-to-a-database`) already exist in `frontend/src/locales/`. Verify they are also resolvable by the React i18n loader (`frontend/src/react/locales/`). If not, add them per playbook §i18n.

## 7. Out of scope for Stage 1 (explicit deferrals)

- **Pinia bridge for `useSQLEditorContext`.** Welcome only needs one prop-callback. We design the Pinia bridge in Stage 2 against a real consumer (likely `InfoPanel`) so the shape is not speculative.
- **Emittery events bridge** (e.g., `alter-schema`, `insert-at-caret`, `set-editor-selection`). Welcome does not emit or listen to any. First needed when migrating components inside `EditorPanel` proper.
- **Monaco React wrapper.** Hardest piece per the playbook (CI-only failure path under Node 24 from direct `import("monaco-editor")` inside React effects). Build a stable integration seam when migrating the editor — not before.
- **`ReactPageMount.vue` rename to `ReactComponentMount.vue`.** Convention-only cleanup; defer until a rename has zero functional value to land alongside other work.

## 8. Future stage sketch (informational only — each stage gets its own brainstorm + spec + plan)

| Stage | Scope | Why this order |
|---|---|---|
| **1 (this spec)** | `Welcome` leaf | Smallest leaf that exercises Pinia + router + permission + i18n + extracted child component. |
| **2** | One read-only panel under `EditorPanel/Panels/` (likely `InfoPanel`, ~261 lines) | First consumer that justifies the `useSQLEditorContext` → Pinia bridge. |
| **3** | Remaining read-only data panels (`Tables`, `Views`, `Functions`, `Procedures`, `Triggers`, `Sequences`, `Packages`, `ExternalTables`, `Diagram`) | Same shape as Stage 2 — repeat the pattern. |
| **4** | `EditorCommon` chrome (chooser dropdowns, share popover, save modal, action bar) | Replace surrounding chrome before touching Monaco. |
| **5** | `AsidePanel` sub-panes (`HistoryPane`, `AccessPane`, `WorksheetPane`, `SchemaPane`) | Largest sub-trees but each is independently shippable. |
| **6** | `ConnectionPanel` and `TabList` | Self-contained subsystems. |
| **7** | `EditorCommon/ResultView` (incl. `VirtualDataTable`, `SingleResultViewV1`, `DetailPanel`) | Heaviest interior — depends on table virtualization story being settled. |
| **8** | Monaco wrapper (`SQLEditor.vue`, `CompactSQLEditor.vue`) | Behind a stable integration seam per playbook. |
| **9** | `SQLEditorPage` + `SQLEditorHomePage` orchestrator → `react/pages/sql-editor/SQLEditorPage.tsx`; flip the route in `router/sqlEditor.ts`; lift `useSQLEditorContext` providers to React. | Shell flip, only after all interior leaves are React. |
| **10** | Delete `frontend/src/views/sql-editor/` and `SQLEditorLayout.vue`; cleanup. | After `rg` confirms zero callers. |

## 9. Practical checklist for Stage 1

- [ ] `frontend/src/react/components/BytebaseLogo.tsx` + test created.
- [ ] `frontend/src/react/components/sql-editor/WelcomeButton.tsx` + test created.
- [ ] `frontend/src/react/components/sql-editor/Welcome.tsx` + test created (uses `usePermissionCheck`, accepts `onChangeConnection` prop).
- [ ] `frontend/src/react/mount.ts` updated with `./components/sql-editor/*.tsx` glob and `"./components/sql-editor"` in `pageDirs`.
- [ ] `EditorPanel/StandardPanel/StandardPanel.vue` swapped to `<ReactPageMount v-else page="Welcome" :on-change-connection="changeConnection" />` (line 55) and import updated (line 98). `showConnectionPanel` + `asidePanelTab` added to the existing `useSQLEditorContext()` destructure.
- [ ] Entire Vue `EditorPanel/Welcome/` directory (`Welcome.vue`, `Button.vue`, `index.ts`) deleted after `rg "from.*EditorPanel/Welcome"` confirms no other callers.
- [ ] React i18n keys verified to resolve.
- [ ] `pnpm --dir frontend fix && check && type-check && test` all pass.
- [ ] Manual UX-parity screenshots captured for all 5 permission/logo states.
