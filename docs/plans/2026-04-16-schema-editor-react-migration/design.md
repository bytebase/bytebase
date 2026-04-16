## References

1. [Vue to React Migration: A Step-by-Step Approach](https://www.index.dev/blog/vue-to-react-migration-strategy) — Covers incremental migration strategies, coexistence patterns, and state management migration paths for large component suites.
2. [Strategy and Tips for Migrating to React](https://brainhub.eu/library/migrating-to-react) — Practical patterns for complex UI migration including encapsulation strategy: wrap small interactive sections as React components, then progressively expand coverage.
3. [DrawDB Architecture (DeepWiki)](https://deepwiki.com/drawdb-io/drawdb/1-overview) — Open-source React database schema editor using React Context + custom hooks for distributed state management, @dnd-kit for drag-and-drop, and modular component organization.
4. [react-resizable-panels (GitHub)](https://github.com/bvaughn/react-resizable-panels) — Headless, unstyled resizable panel library for React. Exports PanelGroup, Panel, PanelResizeHandle. Supports horizontal/vertical, WAI-ARIA accessibility, imperative API, and layout persistence.
5. [React Arborist (GitHub)](https://github.com/brimdata/react-arborist) — Headless virtualized tree component for React. Built on react-window. Supports drag-and-drop, keyboard navigation, ARIA, custom node renderers, and 10,000+ node performance.
6. [Shadcn Resizable](https://www.shadcn.io/ui/resizable) — Shadcn wrapper around react-resizable-panels with Tailwind CSS styling and TypeScript support, providing a battle-tested drag mechanics pattern.
7. [DrawDB Data Management (DeepWiki)](https://deepwiki.com/drawdb-io/drawdb/4-data-management) — DrawDB's schema editing state management: React Context + custom hooks for coordinated state updates, undo/redo history stack, and modular context providers per functional area.

## Industry Baseline

**State management for schema editors**: DrawDB (37k+ GitHub stars) uses React Context with custom hooks to distribute state across specialized providers (tables, relationships, areas, notes). Each functional area has its own context, and components access state through dedicated hooks. This avoids a single monolithic store while maintaining coordinated updates. (DrawDB Architecture, DrawDB Data Management)

**Resizable split panes**: react-resizable-panels is the dominant solution (2025-2026), used by shadcn/ui's Resizable component. It is headless, accessible (WAI-ARIA), supports layout persistence, and provides imperative APIs for programmatic control. It replaces naive-ui's NSplit with a React-idiomatic API. (react-resizable-panels, Shadcn Resizable)

**Virtualized tree navigation**: react-arborist provides a headless virtualized tree built on react-window. It handles 10,000+ nodes, supports keyboard navigation, drag-and-drop, inline rename, and custom node renderers. This maps directly to the NTree + virtual scroll pattern used in the Vue aside tree. (React Arborist)

**Migration strategy**: Industry consensus for large Vue-to-React migrations (2025-2026) favors incremental, component-by-component migration with coexistence. Both sources recommend starting with leaf components, maintaining parallel systems during transition, and using intermediate layers for bridging. Complex editor interfaces should be wrapped as encapsulated React components, then progressively expanded. (Vue to React Migration, Strategy and Tips for Migrating to React)

**Trade-offs and limitations**: Running two frameworks simultaneously adds build complexity and can cause DOM reconciliation conflicts if both try to manage the same DOM subtree. The incremental approach means the Vue version must be maintained until all callers are migrated, creating a temporary duplication cost. DrawDB's distributed context pattern works well for self-contained editors but requires careful coordination when embedding in a larger application with external state (e.g., Pinia stores).

## Research Summary

Key patterns across sources:

1. **React Context replaces Vue provide/inject and Emittery**: All React schema editors studied use React Context for distributed state. DrawDB distributes state across multiple specialized context providers rather than a single store. This maps cleanly to the Vue SchemaEditorLite pattern where `provideSchemaEditorContext()` composes `useTabs`, `useEditStatus`, `useSelection`, and `useScrollStatus` — each becomes a React hook exposed through a single context provider.

2. **Headless component libraries replace naive-ui**: react-resizable-panels replaces NSplit, react-arborist replaces NTree. Both are headless/unstyled, fitting the project's shadcn-style pattern where Base UI primitives are wrapped with CVA variants and Tailwind CSS. This avoids introducing a new opinionated component library.

3. **Pure TS layers are reusable without modification**: The algorithm layer (diff-merge, rebuild, apply), types, specs, and utilities contain no Vue imports. They operate on protobuf metadata types and can be imported directly by React components, eliminating ~1,200 lines from the migration scope.

4. **Incremental migration with coexistence is the correct approach**: The existing migration playbook already establishes this pattern. The React SchemaEditorLite will be a standalone component that Vue parents can embed (via a bridge wrapper if needed), while React parents use it directly. The Vue version continues to serve Vue callers until those callers are themselves migrated.

5. **Event bus maps to callbacks and context dispatch**: The 6 Emittery events in the Vue implementation (`update:selected-rollout-objects`, `rebuild-tree`, `rebuild-edit-status`, `clear-tabs`, `refresh-preview`, `merge-metadata`) map to callback functions passed through React context or directly as props. React's unidirectional data flow replaces the bidirectional event pattern.

## Design Goals

1. **Feature parity with Vue SchemaEditorLite** — The React implementation must support all current editing capabilities: database/schema/table/view/procedure/function editing, column CRUD with type/default/FK cells, index editing, partition editing, tree navigation with context menus, tab management, edit status tracking, selection/rollout object tracking, and DDL preview. Verifiable: every panel type and modal in the Vue implementation has a React counterpart.

2. **Zero regression in Vue callers** — Vue parents (SchemaEditorDrawer, EditorView) continue to work unchanged during and after migration. Verifiable: the Vue SchemaEditorLite component and its imports are not modified or deleted.

3. **Reuse pure TS layers without forking** — The algorithm/, types.ts, spec.ts, and utils/ modules are imported directly by React components from their existing locations. No duplication or parallel implementations. Verifiable: React components import from `@/components/SchemaEditorLite/algorithm`, `@/components/SchemaEditorLite/types`, etc.

4. **Follow established React patterns** — Use React Context (not zustand/redux), useVueState for store access, shadcn-style UI components, react-i18next, and the project's existing Sheet/Dialog/Tabs/Table primitives. Verifiable: no new state management libraries added; all UI primitives from `@/react/components/ui/`.

5. **Incremental delivery** — The migration is structured so that each phase produces a usable, testable component. A partially-complete migration (e.g., table editor without partition editor) is useful on its own. Verifiable: each phase has defined entry and exit criteria.

## Non-Goals

- Rewriting the `algorithm/` layer (diff-merge, rebuild, apply) — pure TS, no Vue dependencies
- Rewriting `spec.ts` or `utils/` — pure TS functions, engine-specific predicates
- Rewriting `types.ts` — framework-agnostic type definitions
- Migrating the Plan/Issue pages themselves
- Migrating SQL editor views (TablesPanel, ViewsPanel, ExternalTablesPanel)
- Replacing ELK layout engine or Monaco editor integration
- Introducing new state management libraries (zustand, TanStack Query)
- Changing schema editor behavior or adding new features during migration
- Migrating all Vue callers — Vue callers keep the Vue version until they are migrated
- Migrating SchemaDiagram or SchemaPane in this effort — these are distinct surfaces with different callers and should be separate migration efforts

## Proposed Design

### Architecture Overview

The React SchemaEditorLite mirrors the Vue component's architecture: a context provider at the root, a two-pane layout with an aside tree on the left and tabbed editor panels on the right, and modal dialogs triggered by user actions. The key difference is replacing Vue-specific patterns (provide/inject, Emittery, naive-ui) with React equivalents (Context, callbacks, headless libraries).

```
SchemaEditorLite (React Context provider + react-resizable-panels)
├── AsideTree (react-arborist with custom node renderers)
│   ├── Node renderers (database, schema, table, view, procedure, function)
│   ├── Node checkboxes (selection state)
│   └── Context menu (dropdown-menu from ui/)
├── EditorPanel (tab-based routing)
│   ├── DatabaseEditor
│   ├── TableEditor
│   │   ├── TableColumnEditor (table from ui/ with editable cells)
│   │   │   ├── DataTypeCell (combobox)
│   │   │   ├── DefaultValueCell (input)
│   │   │   ├── ForeignKeyCell (button + modal trigger)
│   │   │   ├── OperationCell
│   │   │   ├── ReorderCell (drag handle)
│   │   │   └── SelectionCell (checkbox)
│   │   ├── IndexesEditor
│   │   └── PartitionsEditor
│   ├── ViewEditor (Monaco wrapper)
│   ├── ProcedureEditor (Monaco wrapper)
│   ├── FunctionEditor (Monaco wrapper)
│   └── PreviewPane (Monaco wrapper, DDL diff output)
└── Modals
    ├── TableNameModal (dialog)
    ├── SchemaNameModal (dialog)
    ├── EditColumnForeignKeyModal (sheet, wide)
    ├── ViewNameModal (dialog)
    ├── FunctionNameModal (dialog)
    ├── ProcedureNameModal (dialog)
    └── ActionConfirmModal (alert-dialog)
```

### State Management: React Context replacing provide/inject + Emittery

The Vue `provideSchemaEditorContext` composes four sub-hooks (`useTabs`, `useEditStatus`, `useSelection`, `useScrollStatus`) and an Emittery event bus. The React equivalent uses a single context provider that exposes the same composed state through hooks.

**Why React Context over zustand/TanStack**: The schema editor state is scoped to the editor instance, not global. React Context naturally scopes state to a component subtree, which matches the Vue provide/inject pattern exactly. Adding zustand would introduce global state for component-local concerns. (Traces to Design Goal 4: follow established React patterns)

**Emittery replacement**: The 6 event types become callback functions on the context:

| Emittery event | React equivalent |
|---|---|
| `update:selected-rollout-objects` | `onSelectedRolloutObjectsChange` prop (lifted to parent) |
| `rebuild-tree` | `rebuildTree()` function on context, components call directly |
| `rebuild-edit-status` | `rebuildEditStatus(resets)` function on context |
| `clear-tabs` | `clearTabs()` function on context (part of tabs hook) |
| `refresh-preview` | `refreshPreview()` function on context |
| `merge-metadata` | `mergeMetadata(metadatas)` function on context |

The context provider pattern:

```
SchemaEditorProvider (creates context with composed hooks)
  └── provides: tabs, editStatus, selection, scrollStatus, callbacks
      └── children consume via useSchemaEditorContext()
```

Each sub-hook (`useTabs`, `useEditStatus`, `useSelection`, `useScrollStatus`) is reimplemented as a React hook using `useState`/`useReducer`/`useRef`/`useMemo`/`useCallback`. The Vue `ref()` → React `useState()`, Vue `computed()` → React `useMemo()`, Vue `watch()` → React `useEffect()` mappings are direct.

### Resizable Panes: react-resizable-panels replacing NSplit

The Vue component uses `NSplit` with `min=0.15`, `max=0.4`, `default-size=0.25`. The React equivalent uses react-resizable-panels:

```
PanelGroup (direction="horizontal")
  Panel (defaultSize=25, minSize=15, maxSize=40) → AsideTree
  PanelResizeHandle
  Panel (defaultSize=75) → EditorPanel
```

**Why react-resizable-panels over alternatives**: It is headless/unstyled (matches shadcn pattern), widely adopted, WAI-ARIA accessible, and the shadcn ecosystem already wraps it. No opinionated styles to override. (Traces to Design Goal 4)

### Aside Tree: react-arborist replacing NTree

The Vue aside uses naive-ui's NTree with virtual scroll for database/schema/table hierarchy navigation. react-arborist provides:

- Virtualized rendering for large schemas (10,000+ nodes)
- Custom node renderers (database, schema, table group, table, view, procedure, function nodes)
- Keyboard navigation and ARIA
- Selection state management

**Context menu**: Implemented using `dropdown-menu` from `@/react/components/ui/` triggered on right-click, replacing naive-ui's NDropdown. The menu options (create table, rename, drop, etc.) are determined by node type, matching the Vue `context-menu.ts` logic.

**Node checkboxes**: The 10+ node checkbox variants in Vue are simplified to a single `NodeCheckbox` component that receives the node's metadata type and delegates to the selection context. The checkbox variants in Vue exist because Vue template composition requires separate component files; React's JSX conditional rendering handles this in a single component.

**Why react-arborist over building from scratch**: The Vue implementation wraps NTree which provides virtualization, keyboard nav, and node rendering. react-arborist provides the same feature set as a headless component with custom renderers, avoiding reimplementing virtual scroll. (Traces to Design Goal 1: feature parity)

### Tab Management

The Vue `useTabs` hook manages a `Map<string, TabContext>` with current tab tracking. The React equivalent is a `useReducer` with actions for `ADD_TAB`, `CLOSE_TAB`, `SET_CURRENT_TAB`, `CLEAR_TABS`. The tab list renders using the `Tabs` component from `@/react/components/ui/tabs`.

Tab-to-panel routing: The `EditorPanel` reads `currentTab.type` from context and renders the corresponding panel component (DatabaseEditor, TableEditor, ViewEditor, ProcedureEditor, FunctionEditor).

### Editor Panels

Each panel is a direct port of its Vue counterpart:

- **TableEditor**: Multi-mode tabs (Columns, Indexes, Partitions) using `Tabs` from ui/. The column editor is the most complex sub-component, rendering an editable table with custom cell renderers for each column property.
- **TableColumnEditor**: Uses `Table` from `@/react/components/ui/table` with inline-editable cells. Each cell type (DataTypeCell, DefaultValueCell, ForeignKeyCell, OperationCell, ReorderCell, SelectionCell) becomes a React component. The `DataTypeCell` uses `Combobox` for type suggestions. Drag-to-reorder uses HTML5 drag-and-drop or a lightweight drag library.
- **IndexesEditor / PartitionsEditor**: Simpler table-based editors, direct ports.
- **ViewEditor / ProcedureEditor / FunctionEditor**: Wrap the Monaco editor component from `@/react/components/monaco/`.
- **PreviewPane**: Read-only Monaco editor showing generated DDL via `generateDiffDDL()` from the reused algorithm layer.

### Modals

Modals map to existing React UI primitives:

| Vue Modal | React Component | UI Primitive |
|---|---|---|
| ActionConfirmModal | ActionConfirmDialog | `AlertDialog` |
| TableNameModal | TableNameDialog | `Dialog` |
| SchemaNameModal | SchemaNameDialog | `Dialog` |
| EditColumnForeignKeyModal | EditColumnForeignKeySheet | `Sheet` (wide) |
| ViewNameModal | ViewNameDialog | `Dialog` |
| FunctionNameModal | FunctionNameDialog | `Dialog` |
| ProcedureNameModal | ProcedureNameDialog | `Dialog` |

The EditColumnForeignKeyModal is the only one that warrants a Sheet (wide) because it contains a multi-field form with table/column selectors. The others are single-field dialogs (name input + confirm/cancel).

### Pure TS Layer Reuse

The following modules are imported directly by React components without modification:

- `algorithm/diff-merge.ts` — metadata comparison and reconciliation
- `algorithm/rebuild.ts` — rebuild metadata from edit operations
- `algorithm/apply.ts` — apply changes to in-memory metadata
- `types.ts` — EditTarget, TabContext, RolloutObject, EditStatus
- `spec.ts` — engine feature support predicates
- `utils/columnDefaultValue.ts` — default value placeholder logic
- `utils/metadata.ts` — metadata transformation helpers
- `common.ts` — `generateDiffDDL()` RPC call

This preserves ~1,200 lines of tested logic and ensures DDL generation behavior is identical between Vue and React implementations.

### Public API and Bridge

The React SchemaEditorLite exposes the same interface as the Vue version:

- Props: `project`, `readonly`, `selectedRolloutObjects`, `targets`, `loading`, `hidePreview`
- Callbacks: `onSelectedRolloutObjectsChange`, `onIsEditingChange`
- Imperative handle (via `useImperativeHandle`/`forwardRef`): `applyMetadataEdit()`, `refreshPreview()`, `isDirty`

For Vue parents that need to embed the React version (future migration step), a thin Vue wrapper component can mount the React component via `createRoot`. However, this is not required in the initial migration — the React version will be consumed by React parents only, and Vue parents continue using the Vue version. (Traces to Design Goal 2: zero regression, and Non-Goal: not migrating all Vue callers)

### Migration Phases

**Phase 1: Foundation** (~2,000 lines)
- React context provider (replacing provideSchemaEditorContext + Emittery)
- useTabs, useEditStatus, useSelection, useScrollStatus hooks
- SchemaEditorLite shell with react-resizable-panels
- Entry: types.ts, spec.ts, utils/, algorithm/ confirmed importable from React
- Exit: empty editor renders with resizable panes, context available to children

**Phase 2: Aside Tree** (~1,500 lines)
- react-arborist integration with custom node renderers
- Context menu with create/rename/drop actions
- Node checkbox selection (single component with type-based rendering)
- Search/filter functionality
- Exit: tree navigates database metadata, context menu triggers stubs

**Phase 3: Core Editors** (~2,500 lines)
- Tab container with type-based routing
- TableEditor with column editor (DataTypeCell, DefaultValueCell, ForeignKeyCell, OperationCell, ReorderCell, SelectionCell)
- DatabaseEditor (schema selector, table creation trigger)
- Exit: table columns editable, new tables creatable

**Phase 4: Extended Editors** (~1,500 lines)
- IndexesEditor, PartitionsEditor
- ViewEditor, ProcedureEditor, FunctionEditor (Monaco wrappers)
- PreviewPane (DDL diff display)
- Exit: all editor panel types functional

**Phase 5: Modals and Polish** (~1,500 lines)
- All 7 modal dialogs
- Edit status indicators (created/updated/dropped visual markers)
- Scroll position preservation
- Integration testing with `applyMetadataEdit` + `generateDiffDDL`
- Exit: full feature parity verified against Vue implementation
