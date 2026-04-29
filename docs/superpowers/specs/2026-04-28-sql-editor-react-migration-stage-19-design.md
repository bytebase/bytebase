# SQL Editor React Migration — Stage 19 Design

**Date:** 2026-04-28
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Migrate the entire SchemaDiagram subsystem from Vue to React: the
`DiagramPanel.vue` wrapper inside the SQL Editor's right-panel chooser plus
all 44 files under `frontend/src/components/SchemaDiagram/`. After this
stage, the only remaining Vue surfaces in `frontend/src/views/sql-editor/`
will be the ResultView family (Stage 20) and the host shells (Stage 21).

**Non-goals:**
- `EditorCommon/ResultView/*` — Stage 20 (perf-spike headline).
- The five host shells (`Panels`, `EditorPanel`, `StandardPanel`,
  `TerminalPanel`, `ResultPanel`, `SQLEditorHomePage`) — Stage 21.
- Visual / behavioral changes to the diagram itself — parity only.
- New SchemaDiagram features (column edit, relationship inference, etc.) —
  Stage 19 is a port, not a rewrite.
- Test coverage for SchemaDiagram beyond what the migration directly
  requires (the Vue subsystem has zero existing tests; we'll add tests
  only for the new React seams that warrant them — see §5).

> **Heads-up — no cytoscape.** Initial planning assumed SchemaDiagram was
> built on cytoscape. It isn't. The canvas is custom SVG/DOM with a CSS
> `transform: matrix(...)` viewport, drag/wheel handlers via `vueuse`,
> auto-layout via ELK (`elkjs`), and FK lines as inline SVG paths. The
> "cytoscape spike" originally scoped into this stage is therefore
> deleted from the plan.

## 2. Inventory

**In scope (~2,288 LOC):**

| File / group | LOC | Notes |
|---|---|---|
| `EditorPanel/Panels/DiagramPanel/DiagramPanel.vue` | 23 | Thin wrapper. Reads the current tab's `database` + `useDBSchemaV1Store().getDatabaseMetadata(...)` and forwards to `<SchemaDiagram>`. |
| `components/SchemaDiagram/SchemaDiagram.vue` | 265 | Root layout: Navigator (left) + Canvas (right). Owns the context provider. |
| `components/SchemaDiagram/ER/TableNode.vue` | 229 | Per-table card: title, columns list, FK glyphs, hover affordances. |
| `components/SchemaDiagram/ER/ForeignKeyLine.vue` | 176 | SVG path between two tables; recomputes anchor sides + path on layout / pan / zoom changes. |
| `components/SchemaDiagram/Canvas/DummyCanvas.vue` | 176 | Off-screen render of the diagram for `html-to-image` screenshot capture. Exposes `capture(filename)`. |
| `components/SchemaDiagram/Navigator/Tree.vue` | 146 | NTree of schemas → tables with keyword highlight. |
| `components/SchemaDiagram/common/geometry.ts` | 139 | Pure rect / point / segment math. No Vue deps. |
| `components/SchemaDiagram/ER/libs/SVGLine.vue` | 121 | Path renderer with arrowhead decorators. |
| `components/SchemaDiagram/Canvas/Canvas.vue` | 93 | Viewport: applies `transform: matrix(zoom, 0, 0, zoom, x, y)`, mounts `<DummyCanvas>` for screenshot, hosts zoom controls. |
| `components/SchemaDiagram/types/context.ts` | 72 | `SchemaDiagramContext` shape: Emittery event bus + reactive refs. |
| `components/SchemaDiagram/Canvas/composables/useDragCanvas.ts` | 70 | Pan + wheel zoom; uses `normalize-wheel`. |
| `components/SchemaDiagram/ER/libs/autoLayout/engines/elk.ts` | 70 | Dynamic-imports `elkjs`, returns `Map<tableId, Rect>`. |
| `components/SchemaDiagram/Navigator/Navigator.vue` | 66 | Collapsible sidebar shell + search input. |
| `components/SchemaDiagram/common/useDraggable.ts` | 57 | Low-level mousedown/mousemove/mouseup pan helper. |
| `components/SchemaDiagram/common/FocusButton.vue` | 55 | Per-table "focus" button (centers + zooms on a table). |
| `components/SchemaDiagram/Navigator/TreeNode/Label.vue` | 44 | Tree-node label with highlight. |
| `components/SchemaDiagram/Canvas/composables/useSetCenter.ts` | 43 | Listens for `set-center` events, animates `(zoom, position)`. |
| `components/SchemaDiagram/Canvas/libs/fitView.ts` | 39 | Pure math: bbox → `(zoom, position)`. |
| `components/SchemaDiagram/Canvas/ZoomButton.vue` | 38 | Zoom in / out / reset / fit / screenshot button group. |
| `components/SchemaDiagram/Navigator/SchemaSelector.vue` | 37 | Multi-select for which schemas to render (Postgres-style multi-schema DBs). |
| `components/SchemaDiagram/common/context.ts` | 28 | `provide` / `inject` wrapper around the context. |
| `components/SchemaDiagram/Canvas/composables/useFitView.ts` | 26 | Listens for `fit-view`, calls `fitView()` math. |
| `components/SchemaDiagram/common/schema.ts` | 24 | Helpers (engine name, schema-key generation, etc.). |
| `components/SchemaDiagram/ER/libs/autoLayout/types.ts` | 21 | Graph node / edge types passed to ELK. |
| `components/SchemaDiagram/ER/libs/isFocusedFKTable.ts` | 21 | "Should this FK line be highlighted?" predicate. |
| `components/SchemaDiagram/types/schema.ts` | 19 | Internal table / column types. |
| `components/SchemaDiagram/SchemaDiagramIcon.vue` | 17 | Wrapper for the icon shown in tab strips. |
| `components/SchemaDiagram/Navigator/TreeNode/Prefix.vue` | 17 | Tree-node prefix slot. |
| `components/SchemaDiagram/types/geometry.ts` | 15 | `Rect`, `Point` etc. |
| `components/SchemaDiagram/common/FocusIcon.vue` | 15 | Inline SVG icon. |
| `components/SchemaDiagram/common/utils.ts` | 14 | Misc. helpers. |
| `components/SchemaDiagram/Navigator/TreeNode/Suffix.vue` | 27 | Tree-node suffix slot. |
| `components/SchemaDiagram/common/const.ts` | 6 | Constants (default zoom, padding, etc.). |
| `components/SchemaDiagram/types/edit.ts` | 1 | Edit-state placeholder. |
| `components/SchemaDiagram/{ER,Canvas,Navigator,types,common}/index.ts` | ~3-8 each | Barrel re-exports. |

**Total**: 2,288 LOC across 44 files. The canvas + ER + navigator triplet
accounts for ~75% of it; the remaining ~25% is composables, types,
geometry math, and barrels.

### Out of scope (later stages)

```
EditorCommon/ResultView/*                     → Stage 20 (perf spike)
ResultPanel/DatabaseQueryContext.vue          → Stage 20 (mounts ResultViewV1)
ResultPanel/ResultPanel.vue                   → Stage 21 (host shell)
StandardPanel/StandardPanel.vue               → Stage 21
TerminalPanel/TerminalPanel.vue               → Stage 21
Panels/Panels.vue                             → Stage 21
EditorPanel/EditorPanel.vue                   → Stage 21
SQLEditorHomePage.vue                         → Stage 21
SQLEditorPage.vue                             → Stage 21
components/DataExportButton.vue               → Stage 20 (last Vue caller deleted with ResultView)
```

## 3. External deps & primitives

| Vue dep | React replacement | Status |
|---|---|---|
| `NButton`, `NButtonGroup` (Canvas / ZoomButton) | `Button` (`@/react/components/ui/button`) | ✓ |
| `NTooltip` | shared `Tooltip` (`@/react/components/ui/tooltip`) | ✓ |
| `NInput` (Navigator search) | shared `PanelSearchBox` from Stage 16 — a clearable shadcn `Input` with leading icon | ✓ |
| `NSelect` (`SchemaSelector`) | shadcn `Combobox` with `multiple` (already used elsewhere) | ✓ |
| `NTree` (Navigator) | `react-arborist` — already used by Stage 17 `SheetTree` for the worksheet sidebar | ✓ |
| `Emittery` (event bus inside `SchemaDiagramContext`) | Module-level `Emittery` singleton (Stage 16/17 pattern) | ✓ |
| Vue `provide` / `inject` | React context (one per SchemaDiagram instance) **or** module-level state — see §4. | Needs decision (low-risk; default to React context) |
| `vueuse` `useEventListener` | Plain `addEventListener` in `useEffect` cleanup | ✓ |
| `normalize-wheel` (wheel-zoom) | Same package, framework-agnostic | ✓ |
| `elkjs` (auto-layout) | Same package, dynamic-imported the same way | ✓ |
| `html-to-image` + `downloadjs` (screenshot) | Same packages, framework-agnostic | ✓ |
| `lucide-vue-next` icons | `lucide-react` (already in use everywhere else) | ✓ |
| Vue `defineExpose({ capture })` on `DummyCanvas` | Replace with React imperative handle, **or** lift the capture logic to a hook called from `Canvas.tsx` directly. See §4 / §8. | Needs decision |

**No cytoscape, no react-cytoscapejs.** The original roadmap's
"react-cytoscapejs vs direct cytoscape" question is dropped.

## 4. Architecture & phases

A single SchemaDiagram instance is per-tab (rendered inside `DiagramPanel`
which lives inside the right panel chooser). Multiple tabs ⇒ multiple
SchemaDiagram instances at most one of which is visible. That makes a
**React context per instance** the right boundary for state (selected
schemas, zoom, pan, hovered table, …) — module-level singletons would
collide across tabs.

The Emittery event bus, however, **stays per-instance** — it's a private
back-channel between Canvas children and composables, not a cross-tab one.
We carry it inside the same React context.

### Phase 1 — Foundations (types, context, geometry math)

1. Port `types/{context,geometry,schema,edit}.ts` and `common/{const,utils,schema,geometry}.ts` 1:1. These are pure TS, no Vue. Drop one tiny `vue` ref in `types/context.ts` — replace with React `MutableRefObject` / plain values.
2. Create `frontend/src/react/components/SchemaDiagram/context.tsx` — React context provider holding:
   - The Emittery event bus
   - `zoom`, `position`, `rectsByTableId`, `focusedTable`, `keyword`, `selectedSchemas` (as React state)
   - Imperative refs (canvas / desktop DOM nodes) for composables that need them
3. Port `common/useDraggable.ts` → `useDraggable.ts` (React hook, captures via `useEffect` + `addEventListener`).

**Deliverables:** types + geometry helpers + `SchemaDiagramProvider` shell + `useDraggable` hook + 1 unit test for `geometry.ts` segment math.

### Phase 2 — Auto-layout + ELK integration

1. Port `ER/libs/autoLayout/engines/elk.ts` and `ER/libs/autoLayout/types.ts` 1:1. They're plain async TS already; only the dynamic import path needs to keep working under the React esbuild transform (verify in dev — Stage 16 already proved Vite handles dynamic chunks identically for `.tsx`).
2. Wrap with `useAutoLayout(metadata, schemas) → Map<tableId, Rect>` hook for cleanliness.

**Deliverables:** `useAutoLayout` hook + 1 unit test stubbing ELK output.

### Phase 3 — Navigator (left sidebar)

1. Port `Navigator.vue` (66) → `Navigator.tsx`: collapsible sidebar shell + search input (reuse `PanelSearchBox`).
2. Port `Tree.vue` (146) + `TreeNode/{Label,Prefix,Suffix}.vue` → `Tree.tsx` using `react-arborist` (same library Stage 17 used). Keyword highlight reuses the shared `HighlightLabelText` (per existing memory rule).
3. Port `SchemaSelector.vue` (37) → `SchemaSelector.tsx` using shadcn `Combobox` with multi-select.

**Deliverables:** 3 React surfaces + 1 test (tree filter + selection wiring).

### Phase 4 — Canvas (viewport + pan + zoom)

1. Port `Canvas.vue` (93) → `Canvas.tsx`. The whole render is `<div ref={canvas} style={{ transform: matrix(...) }}>...</div>` plus zoom/screenshot button group. Reuse Stage 18 `Tooltip` and `Button`.
2. Port `Canvas/libs/fitView.ts` (39) — pure math, 1:1.
3. Port `Canvas/composables/useDragCanvas.ts` (70) → `useDragCanvas.ts` hook. Calls `useDraggable` from Phase 1, plus a wheel listener for zoom.
4. Port `useFitView.ts` (26) and `useSetCenter.ts` (43) → React hooks listening to the context's Emittery channel.
5. Port `ZoomButton.vue` (38) → `ZoomButton.tsx` (one row of `Button`s). Triggers context emittery events.

**Deliverables:** 5 React surfaces / hooks + 1 test (drag + wheel-zoom math, `fitView()` math).

### Phase 5 — ER nodes + FK lines

1. Port `SVGLine.vue` (121) → `SVGLine.tsx`. Pure presentation; React's SVG support is identical to Vue's.
2. Port `TableNode.vue` (229) → `TableNode.tsx`. Includes `FocusButton` (55) and the column FK glyphs (lucide icons). Uses `useDraggable` to support per-table drag-to-reposition.
3. Port `ForeignKeyLine.vue` (176) → `ForeignKeyLine.tsx`. Reads two table rects from context, recomputes anchor sides + SVG path on layout / drag / pan changes.
4. Port `isFocusedFKTable.ts` 1:1.

**Deliverables:** 4 React surfaces + 1 test (FK anchor-side selection + path generation).

### Phase 6 — Screenshot (DummyCanvas → utility)

1. The Vue `DummyCanvas.vue` (176) `defineExpose({ capture })` — `Canvas.vue` calls `dummy.value?.capture(filename)`. Two options:
   - **(a)** Mirror the Vue split: `DummyCanvas.tsx` exposes `capture()` via `useImperativeHandle`, `Canvas.tsx` keeps a ref. Familiar pattern.
   - **(b)** Lift to a hook: `useScreenshot(elementRef) → capture(filename)`. The hook reads the canvas DOM directly, runs `html-to-image` + `downloadjs` against it. No DummyCanvas component at all.
   - **Recommendation: (b)**. The existing DummyCanvas is "render the diagram a second time off-screen so html-to-image has a clean target without Naive UI chrome". A React hook + a portaled `<div className="hidden">` is enough for the same effect, and it removes the only imperative-ref coupling in the subsystem.

**Deliverables:** 1 hook + 1 manual verification (screenshot ≥1 large schema).

### Phase 7 — `SchemaDiagram` root + `DiagramPanel` wrapper + caller swap

1. Port `SchemaDiagram.vue` (265) → `SchemaDiagram.tsx`: composes Navigator + Canvas inside the context provider.
2. Port `DiagramPanel.vue` (23) → `DiagramPanel.tsx`: reads the current tab's `database` (via `useVueState`) + `useDBSchemaV1Store().getDatabaseMetadata(database.name)` (Pinia store, framework-agnostic), forwards to `<SchemaDiagram>`.
3. Update `Panels.vue` to mount `<ReactPageMount page="DiagramPanel" />` instead of `<DiagramPanel>`.
4. Delete the Vue `DiagramPanel.vue` and the entire `frontend/src/components/SchemaDiagram/` tree.

**Deliverables:** 2 React surfaces + 1 mount shim + Vue file deletions.

## 5. Per-phase checklist

### Phase 1 — Foundations
- [ ] Port `types/*` and `common/{const,utils,schema,geometry}.ts` to `frontend/src/react/components/SchemaDiagram/` 1:1
- [ ] `SchemaDiagramContext` React context + provider component
- [ ] `useDraggable` hook
- [ ] Unit test for segment-overlap geometry helpers

### Phase 2 — Auto-layout
- [ ] `ER/libs/autoLayout/{engines/elk,types}.ts` ported, dynamic import preserved
- [ ] `useAutoLayout(metadata, selectedSchemas)` hook
- [ ] Unit test stubbing ELK to assert output Map shape

### Phase 3 — Navigator
- [ ] `Navigator.tsx` (sidebar shell + search via `PanelSearchBox`)
- [ ] `Tree.tsx` using `react-arborist`, with keyword highlight via `HighlightLabelText`
- [ ] `TreeNode/{Label,Prefix,Suffix}.tsx`
- [ ] `SchemaSelector.tsx` (multi-select via shadcn `Combobox`)
- [ ] Unit test: filter + select fires the right context events

### Phase 4 — Canvas
- [ ] `Canvas.tsx` (transform viewport + zoom button group + screenshot button)
- [ ] `useDragCanvas` hook (pan + wheel zoom)
- [ ] `useFitView`, `useSetCenter` hooks
- [ ] `ZoomButton.tsx`
- [ ] Pure-math tests for `fitView` and `useDragCanvas` zoom math

### Phase 5 — ER nodes + FK lines
- [ ] `SVGLine.tsx`
- [ ] `TableNode.tsx` (incl. `FocusButton`, drag-to-reposition)
- [ ] `ForeignKeyLine.tsx` (anchor side selection + SVG path)
- [ ] Unit test for FK anchor-side selection given two rects

### Phase 6 — Screenshot
- [ ] `useScreenshot(canvasRef)` hook running `html-to-image` + `downloadjs`
- [ ] Manual: capture a 50+ table schema, verify the PNG matches the visible diagram (no Naive UI chrome leaked, FK lines preserved)

### Phase 7 — Root + wrapper + caller swap
- [ ] `SchemaDiagram.tsx` root
- [ ] `DiagramPanel.tsx` wrapper (mount shim added to `frontend/src/react/mount.ts` glob if needed)
- [ ] `Panels.vue` swap (`<DiagramPanel>` → `<ReactPageMount page="DiagramPanel" ... />`)
- [ ] Delete Vue `DiagramPanel.vue` and `frontend/src/components/SchemaDiagram/*`
- [ ] Gates green
- [ ] Manual: walk through §6 verification list

## 6. Manual UX verification

- **Open DiagramPanel** on a Postgres database with multiple schemas + 50+
  tables. Layout should converge to the same structure as Vue (ELK is
  deterministic given the same input → identical positions are expected).
- **Pan / zoom**: drag the canvas, scroll-wheel zoom — the gesture
  responsiveness should match Vue's. Zoom in / out / reset / fit-view
  buttons all work.
- **Schema selector**: toggle schemas on/off, layout re-runs, FK lines
  update.
- **Navigator tree**: search filters schemas + tables, click a table →
  centers it via `set-center` (uses Phase 4's `useSetCenter`).
- **Hover a table**: FocusButton appears, click it → centers + zooms on
  that table.
- **Drag a table**: per-table drag re-positions the node, FK lines
  re-route to the new anchor sides.
- **Screenshot**: click the screenshot button → PNG download + clipboard
  copy. Open the PNG and verify all visible tables + FK lines are
  captured without Naive UI / shadcn chrome.
- **Engine variety**: open the diagram on MySQL (single-schema), MongoDB
  (no FKs, no schemas), and a Postgres DB with 100+ tables to make sure
  edge cases hold.

## 7. Out of scope (deferred)

- All of `EditorCommon/ResultView/*` and `ResultPanel/DatabaseQueryContext.vue` — Stage 20
- The host shells — Stage 21
- AI plugin port — separate effort (currently hosted by Vue at `Panels.vue` / `StandardPanel.vue` via the Stage 16/17 hoist)

## 8. Risks & open questions

- **No existing tests.** SchemaDiagram has zero `.test.*` / `.spec.*`
  files in the Vue tree. We add seed tests where the math or wiring is
  non-trivial (geometry helpers, FK anchor selection, ELK plumbing,
  drag/zoom math, navigator filter wiring) but full coverage is out of
  scope for a parity port.

- **ELK dynamic import under React esbuild.** The existing call
  `await import("elkjs/lib/elk.bundled.js")` works under Vue's Vite
  pipeline. The React `.tsx` files go through esbuild (per
  `react-tsx-transform`) but the resulting modules still hit Vite's chunk
  splitting. Verify in dev that the ELK chunk loads exactly once and
  doesn't re-trigger on every layout. (Cheap to confirm with the
  Network tab.)

- **DummyCanvas → `useScreenshot` hook (recommended).** The Vue subsystem
  has a quirky split where `DummyCanvas.vue` does a second off-screen
  render of the diagram so `html-to-image` has a chrome-free target.
  Replacing this with a hook that runs against the live canvas DOM is
  cleaner — but the live canvas includes Tailwind / shadcn affordances
  (zoom buttons, hover focus button) that we'd need to hide during
  capture. Two sub-options:
  - (b1) Add a `data-screenshot-hide` attribute to those affordances and
    a CSS selector that hides them while a `body[data-capturing]` flag is
    set during the html-to-image call.
  - (b2) Keep a thin hidden DOM wrapper that re-renders only the canvas
    children. Closer to the Vue setup, but loses the simplification win.
  - Recommendation: try (b1) first — it's the cleaner React idiom. Fall
    back to (b2) if affordance hiding turns out to be lossy.

- **Per-instance vs module-level state.** The Vue context is per-instance
  via `provide`/`inject`. We mirror that with a React context per
  `<SchemaDiagram>` mount. **Don't** lift to a module-level singleton —
  multiple tabs can have multiple SchemaDiagram instances open
  simultaneously, and a singleton would cross-talk between them.

- **`react-arborist` reuse.** Stage 17 introduced react-arborist for the
  worksheet `SheetTree`. The Navigator's tree is structurally similar
  (parent = schema, children = tables). Reuse the same lib + the
  existing `onSelect` mount-time-`[]` guard memory.

- **Drag interactions stack with pan.** The canvas listens for
  drag-to-pan; tables also listen for drag-to-reposition; the wheel both
  pans with shift held and zooms otherwise. Vue's implementation
  threads this via event delegation in `useDraggable` + per-table
  `mousedown` handlers with `stopPropagation`. Mirror exactly to avoid
  regressions on click-vs-drag thresholds and middle-mouse panning.
