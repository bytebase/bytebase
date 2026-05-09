# Stage 21 — SQL Editor host-shell flip (Vue → React)

**Date:** 2026-05-08
**Status:** Design — not yet started
**Predecessor:** Stage 20 ("ResultView spike", merged in PR #20227)

This is the final scheduled stage of the bottom-up SQL Editor React
migration. After it, every host shell that currently uses
`<ReactPageMount>` to mount a single React child becomes a React
component directly. The AI plugin (`@/plugins/ai/components/*`) stays
Vue-mounted via a new reverse-direction bridge (`VueMount`); porting it
is deferred to a separate Stage 22 effort if/when desired.

## 1. Why this is mostly mechanical

Every shell already does the same thing: render a `<ReactPageMount
page="…">` whose page is the entire interactive surface. The Vue
wrappers exist only because Vue used to host the inner content; now
they're routing pass-throughs.

The only non-trivial work is:
1. Building the reverse-direction `<VueMount>` bridge so the AI plugin
   can sit inside a React shell without porting it.
2. Wiring the React root into the existing Vue Router entry point
   (`SQLEditorHomePage` is the route target).

Everything else is a per-file flip that should compile after one pass
through the file plus updating callers.

## 2. Scope (~759 LOC of Vue, plus ~50 LOC of new bridge code)

| File | LOC | What it does today | Post-Stage-21 |
|---|---|---|---|
| `views/sql-editor/SQLEditorHomePage.vue` | 195 | Top-level: workspace switcher + sidebar + `<RouterView>` for SQL editor sub-routes | React `SQLEditorHomePage.tsx` |
| `views/sql-editor/SQLEditorPage.vue` | 34 | Trivial wrapper around `<EditorPanel>` | React `SQLEditorPage.tsx` |
| `views/sql-editor/EditorPanel/EditorPanel.vue` | 35 | Trivial wrapper, picks `StandardPanel` vs `TerminalPanel` based on tab mode | React `EditorPanel.tsx` |
| `views/sql-editor/EditorPanel/Panels.vue` | 205 | Splitter / layout. Hosts `<AIChatToSQL>` (AI plugin) via Vue child | React `Panels.tsx` + `<VueMount>` for AI plugin |
| `views/sql-editor/EditorPanel/StandardPanel.vue` | 120 | Wraps React `<EditorMain>` + React `<ResultPanel>` (today via `<ReactPageMount>`) plus AI hoist | React `StandardPanel.tsx` + `<VueMount>` for AI |
| `views/sql-editor/EditorPanel/TerminalPanel.vue` | 184 | Wraps `<CompactSQLEditor>`, `<ConnectionHolder>`, `<ResultViewPage>` (all React); plus history/cancel UI | React `TerminalPanel.tsx` |
| `views/sql-editor/EditorPanel/ResultPanel/ResultPanel.vue` | 298 | Wraps `BatchQuerySelect` (React) + per-database `<DatabaseQueryContext>` (Vue, mounts ResultView) inside an `NTabs` | React `ResultPanel.tsx` directly mounting `ResultView` |
| `views/sql-editor/EditorPanel/ResultPanel/DatabaseQueryContext.vue` | 125 | Today wraps `<ReactPageMount page="ResultViewPage">` + EXECUTING/CANCELLED states | React `DatabaseQueryContext.tsx` (or inlined into `ResultPanel.tsx`) |

Plus a tail item:

- `frontend/src/components/SchemaDiagram/SchemaDiagramIcon.vue` (17
  LOC, inline SVG) — last surviving Vue file under SchemaDiagram. Used
  by `DatabaseEditor.vue` (already deprecated) and SQL Editor sub-tab
  buttons. Inline the SVG into the React caller(s) and delete.

## 3. The reverse-direction `<VueMount>` bridge

`<ReactPageMount.vue>` lets Vue host a React component. Stage 21 needs
the inverse — React shells host the AI plugin (`<AIChatToSQL>`) which
stays Vue. Build a small `VueMount.tsx` mirroring the existing pattern.

### Shape

```tsx
// frontend/src/react/components/VueMount.tsx
import { useEffect, useRef } from "react";
import { type Component, createApp, type App } from "vue";

interface VueMountProps {
  /** The Vue component to mount. */
  component: Component;
  /** Props forwarded to the Vue component. Re-mounted-in-place when the
   *  reference changes — Vue's `app.unmount()` then `createApp(...).mount()`. */
  props?: Record<string, unknown>;
  className?: string;
}

export function VueMount({ component, props, className }: VueMountProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const appRef = useRef<App | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const app = createApp(component, props);
    // Inherit i18n + Pinia from the host app — pass via `provide` or
    // re-install the same plugins. Simplest: reuse the global Vue app's
    // pinia + i18n instances by importing them.
    // (Implementation note: the existing Vue app exposes its `pinia` and
    // `i18n` via module-level imports. Apply them here so stores match.)
    app.mount(containerRef.current);
    appRef.current = app;
    return () => {
      app.unmount();
      appRef.current = null;
    };
  }, [component, props]);

  return <div ref={containerRef} className={className} />;
}
```

### What needs to be solved before this works

1. **Pinia store sharing.** The AI plugin reads `useSQLEditorTabStore`,
   `useCurrentUserV1`, etc. The Vue app installs Pinia at startup; a
   freshly-created `createApp` doesn't share that store. Pattern: import
   the singleton `pinia` instance from `@/store` and `app.use(pinia)`
   inside `VueMount` so the new app shares the same store registry.
2. **i18n sharing.** Same shape — import the existing `i18n` instance
   and `app.use(i18n)`. The React tree already calls
   `i18n.changeLanguage()` via `react-i18next`; the Vue mount picks up
   the same language because it's the same instance.
3. **Vue Router?** Probably not needed — the AI plugin doesn't render
   route-driven content, it's a chat panel. If it does use
   `useRoute()`, install the singleton router too.
4. **Re-render strategy.** When `props` reference changes (parent React
   re-renders), do we unmount/remount the Vue app or update its props?
   Vue 3 supports `app._instance.props.foo = bar` but it's brittle.
   Cleaner: depend the `useEffect` on a `JSON.stringify(props)` or
   memoize the props object, and unmount+remount on change. The AI
   plugin re-renders rarely so the cost is fine.
5. **Layer/portal stacking.** The AI plugin's overlays (modals, tooltips)
   should respect the React layer policy at
   `frontend/src/react/components/ui/layer.ts`. Either pass them through
   (Vue overlays mount into the same DOM) or accept that the AI plugin
   keeps its own stacking order until Stage 22.

The bridge is ~50 LOC plus a careful look at points 1-4. **This is the
risk in Stage 21** — everything else is a flip.

## 4. Per-file flip plan

Order: bottom-up (leaves first), so each PR's mounted child is already
React when its parent flips.

### 4.1. `DatabaseQueryContext.vue` → `DatabaseQueryContext.tsx`

Today: 125 LOC, mostly the EXECUTING/CANCELLED states + a
`ReactPageMount` for `ResultViewPage`.

After: a React component that does the same thing — show a spinner
during EXECUTING, a "click to retry" panel during CANCELLED, otherwise
render `<ResultView>` directly. The `ReactPageMount` indirection
disappears entirely; the new file imports `ResultView` from the
sibling React tree.

Notes:
- The `useExecuteSQL` composable is already called from React in
  `EditorMain.tsx`; the same pattern works here.
- The polling timer (`useCurrentTimestamp`) is already a Vue composable
  with no Pinia dependency — port to a React `useState` + `setInterval`
  hook. Or just inline the elapsed-time calculation into the React
  component since this is the only consumer.

### 4.2. `ResultPanel.vue` → `ResultPanel.tsx`

Today: NTabs over per-context `<DatabaseQueryContext>`, plus the
`BatchQuerySelect` React mount, plus a custom right-click context menu
(`NDropdown`) for closing tabs.

After: shadcn `<Tabs>` from `@/react/components/ui/tabs.tsx` (already
used by Stage 20's `ResultView`), `BatchQuerySelect` and
`DatabaseQueryContext` (now React) imported directly, custom context
menu wired through the existing `ContextMenu` React primitive (built
in Stage 18).

### 4.3. `TerminalPanel.vue` → `TerminalPanel.tsx`

Today: a list-of-queries view with each query rendered as
`<CompactSQLEditor>` + `<ResultViewPage>`-via-`ReactPageMount`. Plus
history navigation and cancel UI.

After: same shape, `<ResultView>` imported directly. The `useHistory`
composable migrates to a React hook (no Pinia dependency).

### 4.4. `StandardPanel.vue` → `StandardPanel.tsx`

Today: `<EditorMain>` (React, via `ReactPageMount`) + `<ResultPanel>`
(Vue, but trivial after step 4.2) + `<AIChatToSQL>` (Vue, AI plugin).

After: React component, both children direct React imports, AI plugin
via `<VueMount component={AIChatToSQL} props={...} />`.

### 4.5. `Panels.vue` → `Panels.tsx`

Today: splitter layout between `<EditorPanel>`/`<StandardPanel>`,
`<DiagramPanel>` (React via `ReactPageMount`), `<AIChatToSQL>` hoist.

After: React, splitter via `react-resizable-panels` (already a peer in
this codebase per `package.json`), AI hoist via `<VueMount>`.

### 4.6. `EditorPanel.vue` → `EditorPanel.tsx`

Trivial — picks `StandardPanel` vs `TerminalPanel` based on tab mode.
No state, no children of its own.

### 4.7. `SQLEditorPage.vue` → `SQLEditorPage.tsx`

Trivial wrapper around `<EditorPanel>`.

### 4.8. `SQLEditorHomePage.vue` → `SQLEditorHomePage.tsx` + Vue Router shim

This is the route target. The route entry in `frontend/src/router/`
points at `SQLEditorHomePage.vue`. Two options:

- **A. Keep the route as a Vue file** (`SQLEditorHomePage.vue` becomes
  a 5-line shim that mounts the React root via `<ReactPageMount
  page="SQLEditorHomePage">`). Cheapest. Recommended.
- **B. Migrate the route into a React Router setup**. The codebase
  already has a partial React Router shell at
  `@/react/router` (per `IAMRemindDialog.tsx`'s `useNavigate` usage),
  but the SQL Editor route is still in Vue Router. Out of scope.

Pick A. The route file ends up looking like every other `ReactPageMount`
shim Stages 18-20 already established.

### 4.9. `SchemaDiagramIcon.vue` (17 LOC)

Inline the SVG into its callers (`viewState.tsx` Vue JSX; the host shell
sub-tab button which is now React after this stage). Delete the file.

## 5. Pre-stage checklist

Before any of the per-file flips, land:

1. **`VueMount.tsx`** with the four pitfalls (Pinia, i18n, props
   re-render, layer stacking) explicitly handled. A test seed that
   mounts a trivial `<button>{{ count }}</button>` Vue component inside
   React, asserts the rendered DOM, and verifies a Pinia-store-reading
   Vue component sees the same store as the rest of the app.
2. **Inventory of `<AIChatToSQL>`'s contract** — what props does it
   take? What stores does it read? What events does it emit upward?
   The bridge has to support that contract.
3. **Decide on the `props` re-render strategy** (point 4 above).
   Document it in `VueMount.tsx`'s JSDoc.

## 6. Acceptance criteria

- All eight `.vue` files listed in §2 deleted.
- `frontend/src/views/sql-editor/` directory contains zero `.vue` files
  (or only the `SQLEditorHomePage.vue` route-shim per option A in §4.8).
- AI plugin functional inside the new React shells (regression-test:
  open AI chat panel, ask a question, see the response stream).
- `VueMount.tsx` covered by at least one unit test.
- Type-check + lint green.
- Manual smoke test of every SQL Editor surface: standard panel,
  terminal panel, batch query, result view (single + multi), schema
  pane, schema diagram, AI chat. (The Stage 18 walkthrough routine
  template is reusable.)

## 7. Risks

1. **`VueMount` bridge complexity.** Sharing Pinia + i18n + router
   instances between two Vue apps in the same page is unusual. If
   stores end up duplicated, we get the kind of "two sources of truth"
   bugs that are hard to trace.
2. **Vue Router pass-through.** `SQLEditorHomePage.vue` is the route
   target; if the React component doesn't render `<RouterView>`
   correctly, sub-routes (history/audit views inside SQL Editor)
   break.
3. **Re-render storms in the AI plugin's `VueMount`.** If parent React
   re-renders frequently and we unmount/remount the Vue tree on every
   re-render, the AI chat history gets nuked. Memoize the `props`
   object aggressively.
4. **Naive UI layer leakage.** The AI plugin uses Naive UI overlays.
   Once it lives inside a React tree it might portal into a non-layer
   `document.body` root and stack incorrectly. Worth a manual check
   after the `StandardPanel` flip.

## 8. After Stage 21

The SQL Editor's only Vue surface is the AI plugin under the
`<VueMount>` bridge. **Stage 22 (optional)** — port the AI plugin if
the maintenance cost of the bridge outweighs the porting cost. The
roadmap memory tracks this as conditional, not mandatory.
