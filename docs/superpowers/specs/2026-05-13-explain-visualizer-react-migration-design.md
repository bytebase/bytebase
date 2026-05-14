# Explain Visualizer — Vue → React Migration

**Date:** 2026-05-13
**Scope:** `frontend/explain-visualizer.html` standalone entry and everything it transitively renders.
**Status:** Implementation in progress (uncommitted; will land in the same PR as other Stage 22 work).

## Context

The Explain Visualizer is a standalone Vite entry (`frontend/explain-visualizer.html` → `frontend/src/explain-visualizer-main.ts`) that renders SQL EXPLAIN output passed via a `?token=...` URL parameter. The token is a `sessionStorage` handle to `{ statement, explain, engine }` stashed by `createExplainToken` in `frontend/src/utils/pev2.ts` and is opened in a separate browser tab from the SQL Editor's result view.

The page dispatches on engine:

| Engine | Renderer | Source |
| --- | --- | --- |
| Postgres | `pev2` npm package (Vue component) | external lib |
| MSSQL | `window.QP.showPlan` from `/libs/qp.min.js` | external lib, dynamic load |
| Spanner | Custom recursive tree (`SpannerQueryPlan` + `SpannerPlanNode`) | repo-local Vue |
| anything else | "Unsupported engine" fallback | inline |

This is the last self-contained Vue surface in the project; the SQL Editor result view that *opens* this window is already React, so closing the gap removes a Vue island in an otherwise React-leaning area.

## Surface inventory

Vue files in scope:

- `frontend/src/ExplainVisualizerApp.vue` (156 lines) — engine dispatch + MSSQL imperative mount via `onMounted`
- `frontend/src/components/SpannerQueryPlan/SpannerQueryPlan.vue` — wrapper, parses JSON `planSource`, finds root node by `index === 0`
- `frontend/src/components/SpannerQueryPlan/SpannerPlanNode.vue` — recursive node with expand/collapse, kind badge, metadata
- `frontend/src/components/SpannerQueryPlan/types.ts` — pure TS, framework-agnostic
- `frontend/src/components/SpannerQueryPlan/index.ts` — barrel
- `frontend/src/explain-visualizer-main.ts` — Vue `createApp(...).mount("body")`

Cross-boundary helpers that stay as-is (no Vue deps already):

- `frontend/src/utils/pev2.ts` — `createExplainToken` / `readExplainFromToken` (`sessionStorage` only). Already consumed by both Vue (current) and React (`SingleResultView.tsx`).
- `frontend/src/types/proto-es/v1/common_pb` — `Engine` enum.

Live callers of the in-scope Vue files: **none outside the explain-visualizer entry chain.** `rg` over `frontend/src` confirmed `pev2` (npm), `SpannerQueryPlan`, and `html-query-plan` are referenced only from `ExplainVisualizerApp.vue` and `SpannerQueryPlan/`. Safe to delete all of them at the end of this migration per the playbook deletion rule.

## Strategy

### Postgres — Vue island for `pev2`

`pev2@1.21.0` is a Vue 3 `defineComponent`. There is no maintained React port and reimplementing the Postgres EXPLAIN renderer is well out of scope. Mount a tiny Vue subapp inside a React component:

```tsx
function PostgresPlanView({ planSource, planQuery }: Props) {
  const hostRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!hostRef.current) return;
    const app = createApp(pev2, { planSource, planQuery });
    app.mount(hostRef.current);
    return () => app.unmount();
  }, [planSource, planQuery]);
  return <div ref={hostRef} className="size-full" />;
}
```

Trade-off: Vue runtime stays in this entry's bundle. Since pev2 transitively requires Vue, that cost exists even today — this approach adds no new runtime weight, just a wrapper shim.

### MSSQL — pure React + dynamic load

`html-query-plan` exposes a global `window.QP.showPlan(container, planXml)`. The Vue version dynamically injects `/libs/qp.min.js` once and then calls `showPlan` in `onMounted`. Translate that 1:1 into a `useEffect` with a `ref`. Move the dynamic-load helper out of the component to keep the effect clean and let it be reused if a second caller ever needs it.

### Spanner — pure React port

Direct translation:

- `SpannerQueryPlan.vue` → `SpannerQueryPlan.tsx`: `useMemo` for `planNodes` and `rootNode`.
- `SpannerPlanNode.vue` → `SpannerPlanNode.tsx`: `useState(true)` for `expanded`, `useMemo` for the derived computeds. Recurses on itself like the Vue version.
- Icons: swap `lucide-vue-next` → `lucide-react`.

The original scoped CSS is preserved verbatim by importing a sibling `.css` file (these are not theme-aware tokens — they're standalone visual styling for a dedicated page). Keeping the original CSS avoids visual regressions and keeps the diff focused on framework migration.

### Entry point

Replace `createApp(...).mount("body")` with React's `createRoot(...).render(...)`. React requires a real container, so update the HTML to add a `<div id="app">` and mount there. Move the entry to `frontend/src/react/explain-visualizer/main.tsx` so the file lives where the `react-tsx-transform` Vite plugin already picks it up — no Vite config changes required.

The HTML's `<script src="...">` updates to the new path. The Vite entry key (`explain-visualizer`) stays the same.

### i18n

The Vue version uses zero locale keys for its visible strings (`"Query Plan"`, `"Unsupported Database Engine"`, etc. are hardcoded English in the template). The visualizer is a single-purpose dev-tooling page that opens English-only `pev2` and `html-query-plan` UIs underneath it; adding i18n now would be scope creep and inconsistent with the embedded third-party UI. **Leave strings hardcoded** to preserve current behavior; can be promoted to locale keys in a follow-up if a translation request lands.

## File mapping

| Vue (current) | React (target) |
| --- | --- |
| `frontend/src/explain-visualizer-main.ts` | `frontend/src/react/explain-visualizer/main.tsx` |
| `frontend/src/ExplainVisualizerApp.vue` | `frontend/src/react/explain-visualizer/ExplainVisualizerApp.tsx` |
| (PG branch inline) | `frontend/src/react/explain-visualizer/PostgresPlanView.tsx` |
| (MSSQL branch inline) | `frontend/src/react/explain-visualizer/MSSQLPlanView.tsx` + `loadQueryPlanScript.ts` helper |
| `frontend/src/components/SpannerQueryPlan/SpannerQueryPlan.vue` | `frontend/src/react/explain-visualizer/SpannerQueryPlan.tsx` |
| `frontend/src/components/SpannerQueryPlan/SpannerPlanNode.vue` | `frontend/src/react/explain-visualizer/SpannerPlanNode.tsx` |
| `frontend/src/components/SpannerQueryPlan/types.ts` | `frontend/src/react/explain-visualizer/spanner-types.ts` |
| `frontend/src/components/SpannerQueryPlan/index.ts` | (deleted; no callers left) |
| `frontend/explain-visualizer.html` (mount body, ts entry) | same file, edited to add `<div id="app">` + point at the new `.tsx` entry |

## Verification

Minimum gate sequence per the migration playbook:

1. `pnpm --dir frontend fix`
2. `pnpm --dir frontend check`
3. `pnpm --dir frontend type-check` (vue-tsc for Vue, separate React tsconfig for the new React tree)
4. `pnpm --dir frontend test`

Add a focused unit test for `SpannerPlanNode` (recursion, expand/collapse, metadata filtering) since the recursive tree is the largest piece of net-new React code in this migration.

Manual smoke (can only be done from the SQL Editor UI in a running dev environment): run an `EXPLAIN` on each of Postgres / MSSQL / Spanner and confirm the new tab renders correctly. Listed as a follow-up the user will do; no Playwright in scope.

## Risk and rollback

- `pev2` Vue-island approach is reversible by reverting the migration commit — no Vite/Vue/React config changes are required to keep the surface working.
- The Spanner CSS port preserves classnames so visual regressions, if any, are pinpointable.
- No deletions until the React surface is verified locally; the playbook's "verify no live callers before deleting" check is already complete for the Spanner dir (no callers outside this entry).
