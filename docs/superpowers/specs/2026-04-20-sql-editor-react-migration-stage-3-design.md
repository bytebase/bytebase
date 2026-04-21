# SQL Editor React Migration — Stage 3 Design

**Date:** 2026-04-20
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Migrate the `AsidePanel/GutterBar/` subsystem (`GutterBar.vue` 79 lines + `TabItem.vue` 57 lines + `common.ts`) to React as a single unit. Exercises the **READ** direction of the Stage 2 Pinia bridge: `asidePanelTab` is subscribed via `useVueState` for active-state rendering, and written via direct store mutation on click.

**Non-goals (Stage 3):**
- Migrating `AsidePanel.vue` itself (the parent shell) — stays Vue.
- Migrating any other aside-panel subtree (`SchemaPane`, `WorksheetPane`, `HistoryPane`, `AccessPane`).
- Porting the `useButton` composable as a React hook — inline the styling (Q2a).
- Supporting `small`/`large` sizes — medium-only (no other callers).
- Porting the custom `DatabaseIcon.vue` to React — use `lucide-react`'s stock `Database` icon (§3.2).

## 2. Architecture

The entire `GutterBar/` subsystem becomes a single React mount. `AsidePanel.vue:8`'s `<GutterBar size="medium" />` swaps to `<ReactPageMount page="GutterBar" />` (no props — size dropped, only caller). The React `GutterBar.tsx` renders an inline `TabItem` React component for each of the 4 tabs. No React-Vue boundary below the mount point.

**State flow:**
- `asidePanelTab` — READ via `useVueState(() => uiStore.asidePanelTab)` for active class. WRITE via `uiStore.asidePanelTab = target` on click.
- Project for JIT gate — READ via `useVueState` on a getter that reads both `useSQLEditorStore` and `useProjectV1Store`.
- Route params — READ via `useVueState(() => router.currentRoute.value.params.project)` for the logo link target.

**Module-level store instances:** Pinia store factories lifted to component top level (per Stage 1 learning — can't call `use*Store()` inside `useVueState` callbacks because of the `react-hooks/rules-of-hooks` lint).

## 3. React file structure

### 3.1 New files

| File | Purpose |
|---|---|
| `frontend/src/react/components/sql-editor/GutterBar.tsx` | Container. Renders: logo link (`<a>` wrapping `@/assets/logo-icon.svg`), a thin horizontal separator, then 4 conditionally-rendered `<TabItem>` components. Reads the project from the SQL Editor stores to gate the ACCESS tab. Exports a zero-prop `GutterBar` function. |
| `frontend/src/react/components/sql-editor/TabItem.tsx` | A single icon+tooltip button. Accepts `{ tab: AsidePanelTab; onClick: () => void }` props. Reads `asidePanelTab` via `useVueState` to compute `isActive`. Applies inline active/inactive classNames to shadcn `Button variant="ghost"`. Tooltip wraps the button using the existing `Tooltip` primitive from `@/react/components/ui/tooltip`. |
| `frontend/src/react/components/sql-editor/GutterBar.test.tsx` | Tests: renders logo + 3 tabs (WORKSHEET/SCHEMA/HISTORY) when `allowJustInTimeAccess=false`; renders 4 tabs when true; clicking a tab writes `asidePanelTab` on the store. |
| `frontend/src/react/components/sql-editor/TabItem.test.tsx` | Tests: renders correct icon + label per tab type; applies active className when `asidePanelTab` matches; calls `onClick` on click. |

### 3.2 Icons

Vue uses `FileCodeIcon`, `HistoryIcon`, `ShieldCheckIcon` from `lucide-vue-next` plus a custom `DatabaseIcon` from `@/components/Icon/DatabaseIcon.vue`. React swaps:
- `FileCodeIcon` → `FileCode` from `lucide-react`
- `HistoryIcon` → `History` from `lucide-react`
- `ShieldCheckIcon` → `ShieldCheck` from `lucide-react`
- `DatabaseIcon` (custom) → **`Database` from `lucide-react`** (per §Q3.2a — YAGNI, don't port the custom icon for one consumer).

### 3.3 Logo link

Vue uses `<router-link :to="linkTarget" target="_blank">` with `linkTarget` computed from `route.params.project`. React approach: render a plain `<a href={href} target="_blank" rel="noopener noreferrer">` where `href` is computed via `router.resolve({ name, params }).href` — matches the existing pattern in `react/components/DashboardSidebar.tsx`.

### 3.4 Tooltip

`@/react/components/ui/tooltip` exposes `<Tooltip content={...} side="right">children</Tooltip>`. Vue uses `side="right" delay={300}`. React: pass `side="right" delayDuration={300}`.

### 3.5 Button styling (Q2a inline)

shadcn `Button variant="ghost"` with a `className` override. Active state: `bg-accent/10 text-accent hover:bg-accent/10 hover:text-accent`. Inactive state: default ghost (`hover:bg-control-bg text-control`). Medium size matches the Vue composable's intent: `h-10 w-10` (40×40 px), icon `size-5` (20 px).

## 4. Data flow & bridge usage

| What GutterBar/TabItem needs | Source | How React gets it |
|---|---|---|
| `asidePanelTab` (READ for active state in TabItem) | `useSQLEditorUIStore().asidePanelTab` (Stage 2 store) | `useVueState(() => uiStore.asidePanelTab)` — reactive subscription. |
| `asidePanelTab` (WRITE on tab click in GutterBar) | Same | Direct assignment `uiStore.asidePanelTab = target` (Pinia setup-store auto-unwraps refs on the store instance). |
| Project for JIT check | `useSQLEditorStore().project` (string name) → `useProjectV1Store().getProjectByName(name)` | `useVueState(() => projectStore.getProjectByName(editorStore.project))`. Store instances lifted to component top level. |
| Current route's project param (for logo link) | `router.currentRoute.value.params.project` | `useVueState(() => router.currentRoute.value.params.project as string \| undefined)`. |
| Route name constants | `PROJECT_V1_ROUTE_DETAIL`, `WORKSPACE_ROUTE_LANDING` | Direct imports from `@/router/dashboard/projectV1` and `@/router/dashboard/workspaceRoutes`. |
| i18n labels | 4 keys (see §6.3) | `useTranslation()`. |

**Stage 3 is the first real reactive READ of `useSQLEditorUIStore` from React.** The test suite verifies this end-to-end: mocking the store as a plain object and asserting that changing `asidePanelTab` triggers a re-render of TabItem's active class. This validates that `useVueState` on a Pinia setup-store field subscribes correctly.

## 5. Vue caller swap

**One call site:** `frontend/src/views/sql-editor/AsidePanel/AsidePanel.vue`:

- Line 8: `<GutterBar size="medium" />`
- Line 81: `import GutterBar from "./GutterBar";`

**Swap:**
- Line 8 → `<ReactPageMount page="GutterBar" />`
- Line 81 → `import ReactPageMount from "@/react/ReactPageMount.vue";`

**No flex-wrapper div needed** — the Vue GutterBar's root is already `h-full flex flex-col ... p-1` and the parent in `AsidePanel.vue` is a sized container. The `ReactPageMount`'s default `<div class="h-full">` fills the parent directly. If the height collapses during manual UX verification, fall back to the Stage 1 `<div class="flex-1 flex flex-col min-h-0">` wrapper.

**`size="medium"` prop dropped.** Only caller, only value (YAGNI).

## 6. Verification

### 6.1 Per-leaf verification

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test` — existing 1160 + new: 3 `TabItem` tests + 3 `GutterBar` tests (~6 new).

### 6.2 UX parity (manual, dev server)

1. Open the SQL Editor. The left gutter shows the logo + 3 tabs (WORKSHEET active by default, SCHEMA, HISTORY) for a project without JIT access.
2. Switch to a project with `allowJustInTimeAccess = true` → 4th tab (ACCESS / "JIT" label) appears.
3. Click each tab → aside panel content changes AND the clicked tab gains active-state tint (`bg-accent/10 text-accent`). Inactive tabs keep ghost styling.
4. Hover each tab → tooltip appears on the right with the correct label.
5. Click the logo at the top → opens the project detail page in a new tab (or workspace landing if no project context).
6. Locale switch (English ↔ Chinese) → all labels update without remount.

### 6.3 Bridge reactivity check (critical)

Trigger `asidePanelTab` writes from elsewhere (e.g., Welcome's "Connect to database" button sets `asidePanelTab = "SCHEMA"`). The React GutterBar's SCHEMA tab should immediately gain active styling without any manual re-render. This confirms `useVueState` subscribes to store writes made outside the React subtree.

### 6.4 i18n

4 keys used: `common.schema`, `worksheet.self`, `common.history`, `sql-editor.jit`. Pre-Stage-3 state of React locales:

- `common.schema` — EXISTS
- `worksheet.self` — MISSING
- `common.history` — MISSING
- `sql-editor.jit` — MISSING

All missing keys must be added to all 5 React locales (`en-US`, `zh-CN`, `es-ES`, `ja-JP`, `vi-VN`) with values copied byte-exact from the corresponding Vue locale (`frontend/src/locales/*.json`). `check-react-i18n.mjs` enforces 1:1 parity.

## 7. Out of scope (deferred)

- **Custom `DatabaseIcon.vue` React port.** Using `lucide-react`'s stock `Database` icon (§3.2). Port only when a visual-parity requirement arises.
- **`AsidePanel.vue` migration.** Shell stays Vue; GutterBar is just a child.
- **`useButton` React hook.** Inline styling sufficient; only one consumer (§Q2a).
- **Size variants (small/large).** Medium-only.
- **Vue-in-React mount infrastructure.** Still deferred until a panel migration requires it.

## 8. Future stage sketch

| Stage | Scope |
|---|---|
| 1 ✅ | `Welcome` leaf (React island mount pattern) |
| 2 ✅ | `ConnectionHolder` + `useSQLEditorUIStore` Pinia bridge |
| **3 (this spec)** | `GutterBar` + `TabItem` (first reactive READ of store from React) |
| 4 | Remaining small `EditorCommon/*` leaves (`DatabaseChooser`, `DisconnectedIcon`, `ReadonlyDatasourceHint`, `OpenAIButton`) |
| 5 | First `EditorPanel/Panels/*` panel — forces Vue-in-React vs cascade-migrate decision |
| 6+ | Bulk panels, AsidePanel subtrees, ConnectionPanel, TabList, ResultView |
| 7+ | Monaco wrapper, shell flip, cleanup |

## 9. Practical checklist

- [ ] 3 missing i18n keys (`worksheet.self`, `common.history`, `sql-editor.jit`) added to all 5 React locale files with byte-exact values from Vue locales.
- [ ] `frontend/src/react/components/sql-editor/TabItem.tsx` + test created.
- [ ] `frontend/src/react/components/sql-editor/GutterBar.tsx` + test created.
- [ ] `AsidePanel.vue:8` swapped to `<ReactPageMount page="GutterBar" />`; `./GutterBar` import replaced with `ReactPageMount` import.
- [ ] Entire Vue `AsidePanel/GutterBar/` directory (`GutterBar.vue`, `TabItem.vue`, `common.ts`, `index.ts`) deleted after `rg` confirms no callers remain.
- [ ] `pnpm --dir frontend fix && check && type-check && test` all pass.
- [ ] Manual UX parity verified (5 visual states + logo link + locale switch).
- [ ] Bridge reactivity verified (external `asidePanelTab` write flips React tab).
