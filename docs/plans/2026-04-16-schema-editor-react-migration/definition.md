## Background & Context

The frontend is undergoing an incremental migration from Vue 3 to React. Existing React pages (ProjectDatabaseDetailPage, ProjectSyncSchemaPage, IssueDetailPage) already bridge into Vue stores via `useVueState()` and reuse pure-TS utilities from the Vue layer. The migration playbook (`docs/plans/2026-04-08-react-migration-playbook.md`) governs the process: migrate bounded surfaces, reuse existing stores/utilities, delete Vue files only when callers are gone.

The schema editor is one of the most complex interactive surfaces in the product. It provides visual DDL editing (tables, columns, indexes, partitions, views, procedures, functions), an ER diagram viewer, and a SQL editor sidebar schema browser. The schema editor is accessed through the Plan/Issue flow when editing database change specifications, and the ER diagram and schema pane are used in database detail and SQL editor views.

## Issue Statement

The SchemaEditorLite component suite (88 files, ~10,300 lines), the SchemaDiagram component suite (44 files, ~2,300 lines), and the SchemaPane sidebar (36+ tree node types, ~1,700 lines) are implemented entirely in Vue 3 with naive-ui dependencies, Vue provide/inject context, and Emittery event bus patterns that have no React equivalents. React pages that embed schema editing or visualization currently either use Vue bridge wrappers or import only leaf utilities (e.g., `getColumnDefaultValuePlaceholder`). No React implementation of the schema editor, diagram, or schema pane exists.

## Current State

### SchemaEditorLite (`frontend/src/components/SchemaEditorLite/`)

- **Entry point**: `SchemaEditorLite.vue` (144 lines) — two-pane splitter (NSplit from naive-ui) with aside tree and editor panels
- **88 files, ~10,300 lines total**
- **Panels** (~1,300 lines): `DatabaseEditor.vue` (284), `TableEditor.vue` (480), `TableList.vue` (327), `ViewEditor.vue` (103), `FunctionEditor.vue` (70), `ProcedureEditor.vue` (70), `PreviewPane.vue` (144), `CommonCodeEditor.vue` (67)
- **TableColumnEditor/** (~370 lines): `DataTypeCell.vue` (99), `DefaultValueCell.vue` (64), `ForeignKeyCell.vue` (100), `OperationCell.vue`, `ReorderCell.vue`, `SelectionCell.vue`
- **IndexesEditor/** (133 lines), **PartitionsEditor/** (248 lines)
- **Modals** (~1,069 lines): `ActionConfirmModal.vue` (59), `TableNameModal.vue` (177), `SchemaNameModal.vue` (105), `EditColumnForeignKeyModal.vue` (287), `ViewNameModal.vue` (142), `FunctionNameModal.vue` (150), `ProcedureNameModal.vue` (149)
- **Aside/Tree** navigation: `Tree.vue` uses NTree from naive-ui with virtual scroll, context menus, and 10+ node checkbox variants (~390 lines)
- **Context layer** (`context/`, ~1,267 lines): `provideSchemaEditorContext()` with Emittery event bus, `useTabs()` (131), `useEditStatus()` (237), `useSelection()` (628), `useScrollStatus()` (113)
- **Algorithm layer** (`algorithm/`, ~979 lines): `diff-merge.ts` (790) handles nested metadata comparison/reconciliation, `rebuild.ts` (39), `apply.ts` (77)
- **Types**: `types.ts` (107 lines) defines `EditTarget`, `TabContext` (database/table/view/procedure/function variants), `EditStatus`, `RolloutObject`
- **Spec**: `spec.ts` exports engine-specific feature support predicates (`engineSupportsEditIndexes`, `engineSupportsEditTablePartitions`, etc.)
- **Public API** (`index.ts`): exports types, common, spec, utils, context, edit, `ActionConfirmModal`, `TableColumnEditor`, and `SchemaEditorLite` default

### SchemaDiagram (`frontend/src/components/SchemaDiagram/`)

- **Entry point**: `SchemaDiagram.vue` (265 lines) — ER diagram with navigator, canvas, table nodes, FK lines
- **44 files, ~2,300 lines total**
- **Canvas**: SVG viewport with pan/zoom (`useDragCanvas`, `useFitView`, `useSetCenter`)
- **ER layout**: `TableNode.vue` (228), `ForeignKeyLine.vue`, ELK auto-layout engine (`autoLayout/engines/elk.ts`, 70 lines)
- **Navigator**: schema selector + table tree
- **Geometry utilities** (~750 lines): 2D math for points, rectangles, line intersections

### SchemaPane (`frontend/src/views/sql-editor/AsidePanel/SchemaPane/`)

- **~1,735 lines total**
- **36+ tree node types** in `TreeNode/`: `CommonNode.vue`, database/schema/table/view/procedure/function/index/trigger/sequence/partition nodes
- **HoverPanel/**: `TableInfo.vue`, `ColumnInfo.vue`, `ViewInfo.vue`, etc.
- **Utilities**: `tree.ts` (tree building from metadata), `click.ts` (routing), `hover-state.ts`, `actions.tsx` (context menu)

### Integration points (Vue callers)

- `Plan/components/StatementSection/SchemaEditorDrawer.vue` — wraps SchemaEditorLite in a drawer for plan SQL editing
- `Plan/components/StatementSection/EditorView/EditorView.vue` — triggers SchemaEditorDrawer
- `sql-editor/EditorPanel/Panels/TablesPanel/ColumnsTable.vue` — imports `DefaultValueCell`, `ForeignKeyCell`
- `sql-editor/EditorPanel/Panels/ViewsPanel/ColumnsTable.vue` — imports `DefaultValueCell`
- `sql-editor/EditorPanel/Panels/ExternalTablesPanel/ExternalTableColumnsTable.vue` — imports `DefaultValueCell`
- `sql-editor/EditorPanel/DiagramPanel` — uses SchemaDiagram
- `DatabaseDetail/SchemaDiagramButton.vue` — modal viewer for SchemaDiagram
- `ColumnDataTable/index.vue` — imports `getColumnDefaultValuePlaceholder`
- `sql-editor/AsidePanel/SchemaPane/HoverPanel/ColumnInfo.vue` — imports `getColumnDefaultValuePlaceholder`

### React code that already touches schema utilities

- `react/pages/project/database-detail/overview/DatabaseObjectExplorer.tsx` — imports `getColumnDefaultValuePlaceholder` from `SchemaEditorLite/utils/columnDefaultValue`

### Existing React UI primitives available

`frontend/src/react/components/ui/`: `dialog.tsx`, `sheet.tsx`, `alert-dialog.tsx`, `tabs.tsx`, `table.tsx`, `button.tsx`, `combobox.tsx`, `search-input.tsx`, `input.tsx`, `dropdown-menu.tsx`, `separator.tsx`, `tooltip.tsx`, `badge.tsx`, `switch.tsx`, `radio-group.tsx`, `textarea.tsx`

### Key external dependencies in Vue implementation

- **naive-ui**: NTree (virtual scroll), NSplit (resizable panes), NButton, NSelect, NDropdown, NModal
- **Emittery**: event bus for context communication (6 event types)
- **lodash-es**: `cloneDeep`, `isEqual`, `debounce`
- **ELK**: graph layout for ER diagrams (`elkjs`)
- **Monaco Editor**: SQL code editing (via wrapper)
- **Proto-ES types**: `DatabaseMetadata`, `SchemaMetadata`, `TableMetadata`, `ColumnMetadata`, `ViewMetadata`, `ProcedureMetadata`, `FunctionMetadata`

## Non-Goals

- Rewriting the `algorithm/` layer (diff-merge, rebuild, apply) — these are pure TypeScript with no Vue dependencies and can be imported directly by React components
- Rewriting `spec.ts` or `utils/` — these are pure TS functions, engine-specific predicates, and column default value helpers that work as-is
- Rewriting `types.ts` — type definitions are framework-agnostic
- Migrating the Plan/Issue pages themselves — the schema editor will be embedded as a React component but the parent Plan/Issue Vue pages will bridge to it
- Migrating SQL editor views (TablesPanel, ViewsPanel, ExternalTablesPanel) — these are separate Vue surfaces with their own callers
- Replacing ELK layout engine or Monaco editor integration — these are framework-agnostic libraries
- Introducing new state management libraries (zustand, TanStack Query) unless a concrete problem arises
- Changing schema editor behavior or adding new features during the migration
- Migrating all Vue callers of the old component — Vue callers that are not themselves being migrated to React will continue using the Vue implementation until they are migrated

## Open Questions

1. Should the React SchemaEditorLite replace the Vue one globally, or should both coexist until all callers are migrated? (default: coexist — the React version will be used by React parents; Vue parents keep the Vue version until they are migrated)
2. What replaces naive-ui's NTree with virtual scroll for the aside tree? (default: build a lightweight virtualized tree using `react-window` or a headless tree library; evaluate `@tanstack/react-virtual` as it is already a common pattern)
3. What replaces naive-ui's NSplit for the resizable two-pane layout? (default: use a lightweight splitter like `react-resizable-panels` or build a minimal one with CSS resize)
4. Should Emittery event bus be replaced with React context + callbacks, or kept as-is since it is framework-agnostic? (default: replace with React context + callbacks for idiomatic React; Emittery is framework-agnostic but the event-driven pattern is non-idiomatic in React)
5. Should the SchemaDiagram and SchemaPane be migrated in the same effort or separately? (default: separately — they are distinct surfaces with different callers; the SchemaEditorLite is the primary target)

## Scope

**L** — This is a new subsystem migration spanning 88+ files and ~10,300 lines of interconnected Vue components, context providers, and event-driven state management. The schema editor has deep coupling between its tree navigation, tab management, edit tracking, selection state, and panel rendering. Multiple viable approaches exist for replacing Vue-specific patterns (provide/inject, Emittery, NTree virtual scroll, NSplit panes) with React equivalents. The algorithm/types/utils layers (~1,200 lines) can be reused directly, but the remaining ~9,100 lines of UI and state management require React reimplementation across ~70 component files.
