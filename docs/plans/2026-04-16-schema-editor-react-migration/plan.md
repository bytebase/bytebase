## Task List

T1: Install dependencies [S] — T2: Create React context types and provider [L] — T3: Implement useTabs hook [M] — T4: Implement useEditStatus hook [M] — T5: Implement useSelection hook [L] — T6: Implement useScrollStatus hook [M] — T7: Create SchemaEditorLite shell with resizable panels [M] — T8: Wire algorithm layer to React context [M] — T9: Create tree node types and builder [L] — T10: Create AsideTree with react-arborist [L] — T11: Implement context menu [M] — T12: Implement node prefix icons and node checkbox [M] — T13: Create TabsContainer and EditorPanel routing [M] — T14: Create DatabaseEditor panel [L] — T15: Create TableEditor panel shell [M] — T16: Create TableColumnEditor with cell components [L] — T17: Create TableList component [M] — T18: Create IndexesEditor [M] — T19: Create PartitionsEditor [M] — T20: Create ViewEditor, ProcedureEditor, FunctionEditor [M] — T21: Create PreviewPane [M] — T22: Create TableNameDialog and SchemaNameDialog [M] — T23: Create ViewNameDialog, FunctionNameDialog, ProcedureNameDialog [M] — T24: Create EditColumnForeignKeySheet [L] — T25: Create ActionConfirmDialog [S] — T26: Wire imperative handle and public API [M] — T27: Add i18n keys for React schema editor [M] — T28: Final validation [M]

---

### T1: Install dependencies [S]

**Objective**: Add react-resizable-panels and react-arborist to the frontend package. Traces to Design Goal 4 (follow established React patterns — headless libraries matching shadcn style).
**Files**: `frontend/package.json`
**Implementation**: Run `pnpm --dir frontend add react-resizable-panels react-arborist`
**Validation**: `cat frontend/package.json | grep -E "react-resizable-panels|react-arborist"` — both packages listed

---

### T2: Create React context types and provider [L]

**Objective**: Create the React SchemaEditorContext that replaces Vue's provide/inject + Emittery. This is the central state container consumed by all child components. Traces to Design Goal 4.
**Size**: L (core architecture, many type definitions, context composition)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/context.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/types.ts`
**Implementation**:
1. In `types.ts`: Define `SchemaEditorContextValue` interface composing all sub-hook return types. Define `SchemaEditorProps` (project, readonly, selectedRolloutObjects, targets, loading, hidePreview, onSelectedRolloutObjectsChange, onIsEditingChange). Re-export `EditTarget`, `TabContext`, `RolloutObject`, `EditStatus` from `@/components/SchemaEditorLite/types`.
2. In `context.tsx`: Create React context via `createContext<SchemaEditorContextValue>`. Create `SchemaEditorProvider` component that composes `useTabs()`, `useEditStatus()`, `useSelection()`, `useScrollStatus()` hooks and provides the combined value. Create `useSchemaEditorContext()` hook that calls `useContext()`. Define callback interfaces that replace Emittery events: `rebuildTree(openFirstChild)`, `rebuildEditStatus(resets)`, `clearTabs()`, `refreshPreview()`, `mergeMetadata(metadatas)`.
**Boundaries**: Do NOT implement the sub-hooks themselves (T3-T6). Use stub/placeholder types for now.
**Dependencies**: None
**Expected Outcome**: Context provider and consumer hook exist, type-check passes
**Validation**: `pnpm --dir frontend type-check` — no errors in new files

---

### T3: Implement useTabs hook [M]

**Objective**: Port Vue `useTabs` (context/tabs.ts, 131 lines) to React. Manages tab CRUD and current tab tracking.
**Size**: M (single file, moderate state logic)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/useTabs.ts`
**Implementation**:
1. State: `tabMap` via `useRef<Map<string, TabContext>>`, `currentTabId` via `useState<string>`, `tabList` via `useMemo` derived from tabMap, `currentTab` via `useMemo` looking up tabMap by currentTabId.
2. Functions: `addTab(coreTab, setAsCurrentTab?)` — reuses existing tab if found (via `findTab`), otherwise creates new with UUID. `setCurrentTab(id)` — validates id exists. `closeTab(id)` — removes tab, selects next/prev. `findTab(target)` — matches by type + database + schema/table/view/procedure/function names. `clearTabs()` — clears map and currentTabId.
3. Use `useCallback` for all functions. Use `requestAnimationFrame` for deferred currentTabId assignment (matching Vue behavior).
**Boundaries**: Does NOT render any UI. Pure state hook.
**Dependencies**: T2 (types)
**Expected Outcome**: Hook returns `TabsContext` matching the interface defined in T2
**Validation**: `pnpm --dir frontend type-check` — no errors

---

### T4: Implement useEditStatus hook [M]

**Objective**: Port Vue `useEditStatus` (context/edit.ts, 237 lines) to React. Tracks which database objects are created/updated/dropped.
**Size**: M (single file, map-based state with derived queries)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/useEditStatus.ts`
**Implementation**:
1. State: `dirtyPaths` via `useRef<Map<string, EditStatus>>` with a `version` counter via `useState` to trigger re-renders on mutation.
2. Functions: `markEditStatus(database, metadata, status)`, `markEditStatusByKey(key, status)`, `getEditStatusByKey(key)`, `removeEditStatus(database, metadata, recursive)`, `clearEditStatus()`.
3. Derived queries: `getSchemaStatus`, `getTableStatus`, `getColumnStatus`, `getPartitionStatus`, `getProcedureStatus`, `getFunctionStatus`, `getViewStatus` — check if resource or any children are dirty.
4. `replaceTableName(database, metadata, newName)` — updates all paths matching old table key.
5. `isDirty` via `useMemo` — true if dirtyPaths is non-empty.
6. Import `keyForResource`, `keyForResourceName` from `@/components/SchemaEditorLite/context/common` (pure TS, no Vue deps).
**Boundaries**: Pure state hook, no UI.
**Dependencies**: T2
**Expected Outcome**: Hook returns `EditStatusContext` matching interface
**Validation**: `pnpm --dir frontend type-check`

---

### T5: Implement useSelection hook [L]

**Objective**: Port Vue `useSelection` (context/selection.ts, 628 lines) to React. Manages rollout object selection state for tables, columns, views, procedures, functions.
**Size**: L (628 lines of selection logic with checked/indeterminate states for 5 resource types)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/useSelection.ts`
**Implementation**:
1. Accept `selectedRolloutObjects` and `onSelectedRolloutObjectsChange` as parameters (lifted state pattern replacing Emittery event).
2. `selectionEnabled` = `selectedRolloutObjects !== undefined`.
3. `selectedRolloutObjectMap` via `useMemo` — Map from key to RolloutObject.
4. For each resource type (table, column, view, procedure, function): `get{Type}SelectionState(db, metadata)` → `{ checked, indeterminate }`, `update{Type}Selection(db, metadata, on)`, `getAll{Type}sSelectionState(db, metadata, items)`, `updateAll{Type}sSelection(db, metadata, items, on)`.
5. Table selection includes all columns. Column selection ensures parent table is selected. Indeterminate = some-but-not-all children selected.
6. All update functions call `onSelectedRolloutObjectsChange(updatedArray)`.
7. Import `keyForResource` from `@/components/SchemaEditorLite/context/common`.
**Boundaries**: Pure state hook. Does NOT render checkboxes.
**Dependencies**: T2
**Expected Outcome**: Hook returns `SelectionContext` matching interface
**Validation**: `pnpm --dir frontend type-check`

---

### T6: Implement useScrollStatus hook [M]

**Objective**: Port Vue `useScrollStatus` (context/scroll.ts, 113 lines) to React. Queues pending scroll-to-table and scroll-to-column requests for consumption by panels.
**Size**: M (2 files — hook + consume pattern)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/useScrollStatus.ts`
**Implementation**:
1. State: `pendingScrollToTable` via `useState`, `pendingScrollToColumn` via `useState`.
2. Functions: `queuePendingScrollToTable(params)` — wraps in `requestAnimationFrame`. `queuePendingScrollToColumn(params)`.
3. Consumer hooks: `useConsumePendingScrollToTable(condition, fn)` — `useEffect` that watches pendingScrollToTable, checks if condition matches (db + schema), calls fn and clears pending. Same for `useConsumePendingScrollToColumn`.
4. Rich metadata types: `RichSchemaMetadata`, `RichTableMetadata`, `RichColumnMetadata`, `RichMetadataWithDB<T>` — port from Vue types.
**Boundaries**: Pure state hook.
**Dependencies**: T2
**Expected Outcome**: Hook returns `ScrollStatusContext`
**Validation**: `pnpm --dir frontend type-check`

---

### T7: Create SchemaEditorLite shell with resizable panels [M]

**Objective**: Create the top-level React SchemaEditorLite component with resizable two-pane layout. Traces to Design Goal 5 (incremental delivery — Phase 1 exit criterion).
**Size**: M (root component, react-resizable-panels integration)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/SchemaEditorLite.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/index.ts`
**Implementation**:
1. In `SchemaEditorLite.tsx`: Accept `SchemaEditorProps`. Render `SchemaEditorProvider` wrapping `PanelGroup` (horizontal) with two `Panel`s separated by `PanelResizeHandle`. Left panel: defaultSize=25, minSize=15, maxSize=40 → placeholder for AsideTree. Right panel: defaultSize=75 → placeholder for EditorPanel. Show loading spinner when `loading` or targets empty. Use `forwardRef` + `useImperativeHandle` to expose `applyMetadataEdit`, `refreshPreview`, `isDirty` (stubs initially).
2. In `index.ts`: Re-export component and types.
**Boundaries**: Aside and Editor panels are placeholders. Imperative handle methods are stubs.
**Dependencies**: T1 (react-resizable-panels), T2 (context)
**Expected Outcome**: Component renders two resizable panes with placeholder content
**Validation**: `pnpm --dir frontend type-check`

---

### T8: Wire algorithm layer to React context [M]

**Objective**: Create React-compatible wrappers for `useRebuildMetadataEdit` and `useApplyMetadataEdit` that work with the React context interface instead of the Vue SchemaEditorContext.
**Size**: M (2 thin wrapper functions + interface adapter)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/useAlgorithm.ts`
**Implementation**:
1. `useRebuildMetadataEdit(context)`: Takes React context. Calls `clearEditStatus()`, constructs `DiffMerge` with a thin adapter `{ markEditStatusByKey: context.markEditStatusByKey }`, calls `dm.merge()`. Then via `setTimeout(0)` (replacing Vue `nextTick`): calls `context.clearTabs()` and/or `context.rebuildTree()` based on resets parameter.
2. `useApplyMetadataEdit(context)`: Takes React context. Returns `applyMetadataEdit(database, metadata)` that clones metadata and filters out dropped schemas/tables/columns/views/procedures/functions using context status getters.
3. The `DiffMerge` class from `@/components/SchemaEditorLite/algorithm/diff-merge` accepts `context` parameter but only uses `markEditStatusByKey`. Create a `DiffMergeContext` interface type with just `{ markEditStatusByKey: (key: string, status: EditStatus) => void }` and cast the React context to satisfy it.
**Boundaries**: Does NOT modify the Vue algorithm files. Creates new React-side wrappers.
**Dependencies**: T2, T4 (useEditStatus)
**Expected Outcome**: `useAlgorithm(context)` returns `{ rebuildMetadataEdit, applyMetadataEdit }`
**Validation**: `pnpm --dir frontend type-check`

---

### T9: Create tree node types and builder [L]

**Objective**: Port the tree node type system and `useBuildTree()` from Vue `Aside/common.ts` (~420 lines) to React. This builds the hierarchical tree data from database metadata.
**Size**: L (complex type hierarchy, recursive tree building, key generation)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Aside/tree-builder.ts`
**Implementation**:
1. Define TreeNode union type: `TreeNodeForInstance`, `TreeNodeForDatabase`, `TreeNodeForSchema`, `TreeNodeForTable`, `TreeNodeForColumn`, `TreeNodeForView`, `TreeNodeForProcedure`, `TreeNodeForFunction`, `TreeNodeForGroup<T>`, `TreeNodeForPlaceholder<T>`. Each extends a base `{ key: string; label: string; isLeaf: boolean; children?: TreeNode[] }`. Adapt from naive-ui's `TreeOption` to a framework-agnostic interface compatible with react-arborist's node data.
2. `buildTree(targets: EditTarget[])`: Maps `EditTarget[]` → `TreeNode[]` hierarchy. Groups by instance. For each database, iterates schemas. For single-schema engines (MySQL/TiDB), skips schema nodes. Creates group nodes (table, view, procedure, function) under each schema. Creates leaf nodes under each group. Handles placeholder nodes for empty groups.
3. Uses `keyForResource`/`keyForResourceName` from `@/components/SchemaEditorLite/context/common`.
4. Returns `{ tree: TreeNode[]; nodeMap: Map<string, TreeNode> }`.
**Boundaries**: Pure data transformation, no UI, no React hooks.
**Dependencies**: T2 (types)
**Expected Outcome**: `buildTree()` generates correct hierarchy from EditTarget metadata
**Validation**: `pnpm --dir frontend type-check`

---

### T10: Create AsideTree with react-arborist [L]

**Objective**: Create the aside tree navigation panel using react-arborist for virtualized rendering. Port from Vue `Aside/Tree.vue` (~680 lines).
**Size**: L (react-arborist integration, custom renderers, search, tree state management)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Aside/index.ts`
**Implementation**:
1. State: `searchPattern` via `useState`, debounced via `useMemo`+`debounce`. Tree data via `useMemo` calling `buildTree(targets)`. `treeRef` via `useRef` for react-arborist's Tree ref.
2. Render: `SearchInput` at top (from ui/). `Tree` from react-arborist with `data={treeData}`, `rowHeight={28}`, custom `renderRow`. Node renderer switches on node type for label styling (green=created, yellow=updated, red+strikethrough=dropped).
3. Click handling: Schema/group → expand + open database tab. Table → open table tab. View/Procedure/Function → open respective tab. Column → open parent table + queue scroll.
4. Listen for `rebuildTree` callback from context to trigger tree data rebuild.
5. Search: Debounced 500ms, fetches filtered metadata from server via `dbSchemaStore.getOrFetchDatabaseMetadata()`, merges via `mergeMetadata()`, triggers rebuild.
**Boundaries**: Context menu is T11, node checkboxes are T12.
**Dependencies**: T1 (react-arborist), T2, T3 (tabs), T9 (tree builder)
**Expected Outcome**: Tree renders database metadata hierarchy with click-to-navigate
**Validation**: `pnpm --dir frontend type-check`

---

### T11: Implement context menu [M]

**Objective**: Port context menu logic from Vue `Aside/context-menu.ts` (~200 lines) to React.
**Size**: M (context menu options + handlers, DropdownMenu integration)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Aside/useContextMenu.ts`
- Modify: `frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx`
**Implementation**:
1. In `useContextMenu.ts`: Hook returning `{ menuState, menuOptions, showMenu(e, node), handleSelect(key) }`. `menuState` = `{ show, x, y, node }`. `menuOptions` computed from node type: database → [Create Schema]; schema → [Drop/Restore]; table group → [Create Table]; table → [Rename, Drop/Restore]; view/procedure/function → [Drop/Restore]. Returns action callbacks that the tree component wires to modal state.
2. In `AsideTree.tsx`: Add `DropdownMenu` from ui/ rendered at `menuState.x, menuState.y` when `menuState.show`. Wire `onContextMenu` on tree nodes to `showMenu()`. Wire menu items to `handleSelect()` which dispatches to callbacks (openTableNameModal, openSchemaNameModal, markDropped, markRestored).
**Boundaries**: Modals themselves are T22-T25.
**Dependencies**: T10
**Expected Outcome**: Right-click on tree nodes shows context menu with correct options
**Validation**: `pnpm --dir frontend type-check`

---

### T12: Implement node prefix icons and node checkbox [M]

**Objective**: Port NodePrefix.vue and the 10+ NodeCheckbox variants to React as unified components.
**Size**: M (icon mapping + single checkbox component)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Aside/NodePrefix.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Aside/NodeCheckbox.tsx`
**Implementation**:
1. `NodePrefix`: Renders icon based on node type (DatabaseIcon, TableIcon, ViewIcon, etc.). For column nodes: PrimaryKeyIcon if in PK, IndexIcon if indexed. Conditionally renders `NodeCheckbox` if `selectionEnabled` from context.
2. `NodeCheckbox`: Single component replacing 10+ Vue variants. Accepts `node: TreeNode`. Switches on node type to call appropriate selection context methods: for table → `getTableSelectionState`/`updateTableSelection`; for column → `getColumnSelectionState`/`updateColumnSelection`; for view/procedure/function → similar. For group nodes → `getAll{Type}sSelectionState`/`updateAll{Type}sSelection`. For schema → aggregate of all child groups.
**Boundaries**: Does NOT handle selection state itself (delegates to context from T5).
**Dependencies**: T5 (selection), T10 (tree)
**Expected Outcome**: Tree nodes show correct icons and functional checkboxes
**Validation**: `pnpm --dir frontend type-check`

---

### T13: Create TabsContainer and EditorPanel routing [M]

**Objective**: Port Editor.vue (68 lines) and TabsContainer.vue (179 lines) to React. Provides tab bar and panel routing.
**Size**: M (2 components, tab rendering + type-based routing)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/EditorPanel.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/TabsContainer.tsx`
**Implementation**:
1. `TabsContainer`: Horizontal scrolling tab bar. Each tab shows icon + name (truncated via `EllipsisText`) + close button. Tab colors based on edit status: dropped=red+line-through, created=green, updated=yellow. Click → `setCurrentTab()`. Close button → `closeTab()`. Auto-scroll active tab into view via `useEffect` + `scrollIntoView`.
2. `EditorPanel`: Renders `TabsContainer` at top. Below, switches on `currentTab.type`: "database" → `DatabaseEditor`, "table" → `TableEditor`, "view" → `ViewEditor`, "procedure" → `ProcedureEditor`, "function" → `FunctionEditor`. Uses `key={currentTab.id}` to force remount on tab switch. Shows empty state when no tab selected.
**Boundaries**: Panel components themselves are T14-T21 (use placeholders initially).
**Dependencies**: T2, T3 (tabs), T4 (edit status)
**Expected Outcome**: Tab bar renders with click/close, panel area routes to correct component
**Validation**: `pnpm --dir frontend type-check`

---

### T14: Create DatabaseEditor panel [L]

**Objective**: Port DatabaseEditor.vue (284 lines) to React. Provides schema selector, table list, and create-table trigger.
**Size**: L (schema selector, sub-tab switching, table list integration)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/DatabaseEditor.tsx`
**Implementation**:
1. Props: `db`, `database`, `selectedSchemaName`, `onSelectedSchemaNameChange`, `searchPattern`.
2. State: `selectedSubTab` ("table-list" | "schema-diagram"), `tableNameModalContext`.
3. Schema selector: `Combobox` (from ui/) visible only for PostgreSQL (`hasSchemaProperty(engine)`). Auto-select first schema if current is invalid via `useEffect`.
4. Toolbar: "Create Table" button (disabled if readonly or schema dropped). Selection summary badge if `selectionEnabled`.
5. Content: Renders `TableList` component when sub-tab is "table-list". Schema diagram disabled for now (non-goal).
6. Handlers: `handleCreateNewTable()` → sets `tableNameModalContext`. `tryEditTable(table)` → `addTab()` with table tab.
**Boundaries**: Schema diagram tab shows "coming soon" or is hidden. TableNameDialog rendered conditionally from modal context.
**Dependencies**: T2, T3, T4, T17 (TableList)
**Expected Outcome**: Database panel shows schema selector (PG), table list, create button
**Validation**: `pnpm --dir frontend type-check`

---

### T15: Create TableEditor panel shell [M]

**Objective**: Port TableEditor.vue (480 lines) to React. Provides the multi-mode editor (Columns/Indexes/Partitions) with toolbar and handlers.
**Size**: M (mode switching, toolbar buttons, handler orchestration)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableEditor.tsx`
**Implementation**:
1. Props: `db`, `database`, `schema`, `table`, `searchPattern`.
2. State: `mode` ("COLUMNS" | "INDEXES" | "PARTITIONS"), `showEditColumnForeignKeyModal`, `editForeignKeyContext`.
3. Computed: `engine`, `disableChangeTable`, `allowChangePrimaryKeys`, `allowReorderColumns`, `showIndexes` (engine supports), `showPartitions` (engine supports).
4. Toolbar: Mode buttons (Columns/Indexes/Partitions), "Add Column"/"Add Index"/"Add Partition" buttons, selection summary badge.
5. Content: Switch on mode → `TableColumnEditor`, `IndexesEditor`, `PartitionsEditor`.
6. Handlers: `handleAddColumn()` — create empty ColumnMetadata, push to table.columns, mark created, rebuild tree, queue scroll. `handleAddIndex/Partition()` — similar. `handleEditColumnForeignKey(column, fk)` — open FK modal. `markTableStatus()` — status management.
7. Footer: `PreviewPane` rendered below editor.
**Boundaries**: Cell components are T16. Index/Partition editors are T18-T19.
**Dependencies**: T2, T3, T4, T6
**Expected Outcome**: Table editor shell renders with mode switching and add buttons
**Validation**: `pnpm --dir frontend type-check`

---

### T16: Create TableColumnEditor with cell components [L]

**Objective**: Port TableColumnEditor.vue (~370 lines) and its 6 cell components to React. This is the most complex editor sub-component.
**Size**: L (editable table with 10+ columns, custom cell renderers, virtual scroll)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor/TableColumnEditor.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor/DataTypeCell.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor/DefaultValueCell.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor/ForeignKeyCell.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor/OperationCell.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor/ReorderCell.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor/SelectionCell.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor/index.ts`
**Implementation**:
1. `TableColumnEditor`: Renders a `Table` (from ui/) with columns: Selection (checkbox), Reorder (up/down), Name (inline input), Type (DataTypeCell), Default (DefaultValueCell), On Update (MySQL only), Comment, Not Null (checkbox), Primary Key (checkbox), Foreign Key (ForeignKeyCell), Operations (OperationCell). Columns conditionally visible based on engine/readonly/selection state.
2. `DataTypeCell`: `Combobox` with type suggestions from `getDataTypeSuggestionList(engine)`.
3. `DefaultValueCell`: Dropdown for null/expression/value + input. Uses `getColumnDefaultValuePlaceholder()`.
4. `ForeignKeyCell`: Display FK reference as "schema.table(column)" link + edit button. Emits `onForeignKeyEdit`/`onForeignKeyClick`.
5. `OperationCell`: Drop button (or Restore if dropped). Prevents dropping last column.
6. `ReorderCell`: Up/down arrow buttons using `arraySwap()`.
7. `SelectionCell`: Checkbox using column selection from context.
8. PK detection: Find index with `primary=true`, check if column in `pk.expressions`.
**Boundaries**: Does NOT implement virtual scroll in first pass (standard table). Can add virtualization later.
**Dependencies**: T2, T4, T5, T6, T15
**Expected Outcome**: Column editor renders editable table with all cell types
**Validation**: `pnpm --dir frontend type-check`

---

### T17: Create TableList component [M]

**Objective**: Port TableList.vue (327 lines) with NameCell, OperationCell, SelectionCell to React.
**Size**: M (table with 3 custom cells)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableList/TableList.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/TableList/index.ts`
**Implementation**:
1. `TableList`: Renders `Table` from ui/ with columns: Selection (checkbox for table), Name (clickable, edit-status-colored), Engine (MySQL), Comment, Collation, Operations (drop/restore).
2. Props: `db`, `database`, `schema`, `tables`, `searchPattern`, `onEditTable(table)`, `onEditColumn(column)`.
3. Filter tables by searchPattern. Row styling: green=created, yellow=updated, red=dropped.
4. Selection: All-tables checkbox header, per-table checkboxes. Uses `getAllTablesSelectionState`/`updateAllTablesSelection` from selection context.
5. NameCell: Clickable → calls `onEditTable(table)`. Shows status color.
6. OperationCell: Drop/restore with status guards.
**Boundaries**: Reuses selection context from T5.
**Dependencies**: T2, T4, T5, T14
**Expected Outcome**: Table list renders with selection, click-to-edit, drop/restore
**Validation**: `pnpm --dir frontend type-check`

---

### T18: Create IndexesEditor [M]

**Objective**: Port IndexesEditor.vue (133 lines) with ColumnsCell and OperationCell to React.
**Size**: M (table with custom cells for index definition)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/IndexesEditor/IndexesEditor.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/IndexesEditor/index.ts`
**Implementation**:
1. `IndexesEditor`: Table with columns: Name (input), Columns (multi-select from table columns), Unique (checkbox), Primary (checkbox, read-only), Comment, Operations.
2. Props: `db`, `database`, `schema`, `table`, `readonly`.
3. ColumnsCell: `Combobox` (multi) selecting from `table.columns[].name`.
4. OperationCell: Drop/restore based on edit status.
5. Index mutations: Direct modification of `table.indexes[]` array + `markEditStatus`.
**Boundaries**: Pure editor, no modal interactions.
**Dependencies**: T2, T4, T15
**Expected Outcome**: Index editor renders with editable fields
**Validation**: `pnpm --dir frontend type-check`

---

### T19: Create PartitionsEditor [M]

**Objective**: Port PartitionsEditor.vue (248 lines) with cell components to React.
**Size**: M (table with 5 custom cells)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/PartitionsEditor/PartitionsEditor.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/PartitionsEditor/index.ts`
**Implementation**:
1. Table columns: Name (input), Type (select: RANGE/LIST/HASH), Expression (input), Value (input), Operations.
2. Props: `db`, `database`, `schema`, `table`, `readonly`.
3. Type defaults: New partitions copy type from first existing partition.
4. OperationCell: Drop/restore.
5. Direct mutation of `table.partitions[]` + `markEditStatus`.
**Boundaries**: Supports MySQL/TiDB partition types only (per spec.ts).
**Dependencies**: T2, T4, T15
**Expected Outcome**: Partition editor renders with editable fields
**Validation**: `pnpm --dir frontend type-check`

---

### T20: Create ViewEditor, ProcedureEditor, FunctionEditor [M]

**Objective**: Port ViewEditor.vue (103 lines), ProcedureEditor.vue (70 lines), FunctionEditor.vue (70 lines) to React. All are similar: Monaco editor for SQL definition.
**Size**: M (3 similar components wrapping Monaco)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/ViewEditor.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/ProcedureEditor.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/FunctionEditor.tsx`
**Implementation**:
1. Each component: Props from tab context (db, database, schema, view/procedure/function). Renders `MonacoEditor` (from `@/react/components/monaco/`) with SQL content. On change → update metadata definition + `markEditStatus("updated")`. Readonly mode from context.
2. ViewEditor: Edits `view.definition`.
3. ProcedureEditor: Edits `procedure.definition`.
4. FunctionEditor: Edits `function.definition`.
**Boundaries**: No additional UI beyond Monaco editor.
**Dependencies**: T2, T4, T13
**Expected Outcome**: SQL editors render and track changes
**Validation**: `pnpm --dir frontend type-check`

---

### T21: Create PreviewPane [M]

**Objective**: Port PreviewPane.vue (144 lines) to React. Shows generated DDL diff in a read-only Monaco editor.
**Size**: M (Monaco wrapper + DDL generation call)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Panels/PreviewPane.tsx`
**Implementation**:
1. Props: `db`, `database`, `schema`, `table`.
2. Computes "mocked" metadata: clone table, filter out dropped columns.
3. Calls `generateDiffDDL()` from `@/components/SchemaEditorLite/common` with baseline vs mocked metadata.
4. Renders `ReadonlyMonaco` (from `@/react/components/monaco/`) with SQL dialect matching engine.
5. Refresh: listens for `refreshPreview` callback from context via `useEffect`.
6. Show/hide toggle via `hidePreview` from context.
**Boundaries**: Does NOT modify metadata.
**Dependencies**: T2, T13
**Expected Outcome**: Preview pane shows generated DDL
**Validation**: `pnpm --dir frontend type-check`

---

### T22: Create TableNameDialog and SchemaNameDialog [M]

**Objective**: Port TableNameModal.vue (177 lines) and SchemaNameModal.vue (105 lines) to React using Dialog from ui/.
**Size**: M (2 dialog forms with validation)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Modals/TableNameDialog.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Modals/SchemaNameDialog.tsx`
**Implementation**:
1. `TableNameDialog`: Props: `open`, `onClose`, `database`, `metadata`, `schema`, `table?` (undefined=create mode). Input for table name. Validation: regex `/^\S[\S ]*\S?$/`, no duplicates. Create flow: create `TableMetadata`, push to `schema.tables`, mark "created", create default "id" column, add tab, queue scroll, rebuild tree. Edit flow: update `table.name`, rebuild edit status.
2. `SchemaNameDialog`: Similar but for schema creation. Input for schema name. Creates `SchemaMetadata`, pushes to `database.schemas`, marks "created", rebuilds tree.
3. Both use `Dialog`, `DialogTitle`, `DialogDescription` from ui/. Input from ui/.
**Boundaries**: Does NOT handle rename through context menu (that's wired in T11).
**Dependencies**: T2, T3, T4
**Expected Outcome**: Dialogs create/rename tables and schemas correctly
**Validation**: `pnpm --dir frontend type-check`

---

### T23: Create ViewNameDialog, FunctionNameDialog, ProcedureNameDialog [M]

**Objective**: Port ViewNameModal.vue (142), FunctionNameModal.vue (150), ProcedureNameModal.vue (149) to React.
**Size**: M (3 similar dialogs)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Modals/ViewNameDialog.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Modals/FunctionNameDialog.tsx`
- Create: `frontend/src/react/components/SchemaEditorLite/Modals/ProcedureNameDialog.tsx`
**Implementation**:
1. All follow same pattern as TableNameDialog: Input for name, validation (no duplicates), create flow (create metadata, push to schema, mark "created", add tab, rebuild tree).
2. `ViewNameDialog`: Creates `ViewMetadata` with empty `definition`.
3. `FunctionNameDialog`: Creates `FunctionMetadata`.
4. `ProcedureNameDialog`: Creates `ProcedureMetadata`.
**Boundaries**: Simpler than TableNameDialog (no default column creation).
**Dependencies**: T2, T3, T4
**Expected Outcome**: Dialogs create views/functions/procedures correctly
**Validation**: `pnpm --dir frontend type-check`

---

### T24: Create EditColumnForeignKeySheet [L]

**Objective**: Port EditColumnForeignKeyModal.vue (287 lines) to React using Sheet (wide) from ui/. The most complex modal with cascading dropdowns.
**Size**: L (multi-field form, cascading selects, FK CRUD)
**Files**:
- Create: `frontend/src/react/components/SchemaEditorLite/Modals/EditColumnForeignKeySheet.tsx`
**Implementation**:
1. Props: `open`, `onClose`, `database`, `metadata`, `schema`, `table`, `column`, `foreignKey?`.
2. State: `referencedSchemaName`, `referencedTableName`, `referencedColumnName`, `foreignKeyName`. Initialize from `foreignKey` prop on mount.
3. Cascading: Schema change → reset table+column. Table change → reset column. Derive options from metadata schemas/tables/columns via `useMemo`.
4. Schema selector: `Combobox` (hidden if single-schema engine).
5. Table selector: `Combobox` with tables from selected schema.
6. Column selector: `Combobox` with columns from selected table.
7. FK name input: `Input` from ui/.
8. Confirm: Create new `ForeignKeyMetadata` or update existing. Uses `upsertColumnFromForeignKey()` from `@/components/SchemaEditorLite/edit`.
9. Delete: Remove column from FK, remove FK if empty. Uses `removeColumnFromForeignKey()`.
10. Mark column as "updated" after changes.
**Boundaries**: Uses Sheet (wide) per design.
**Dependencies**: T2, T4, T15
**Expected Outcome**: FK editing sheet with cascading dropdowns
**Validation**: `pnpm --dir frontend type-check`

---

### T25: Create ActionConfirmDialog [S]

**Objective**: Port ActionConfirmModal.vue (59 lines) to React using AlertDialog from ui/.
**Files**: `frontend/src/react/components/SchemaEditorLite/Modals/ActionConfirmDialog.tsx`
**Implementation**: `AlertDialog` with title, description, cancel, and confirm buttons. Props: `open`, `onClose`, `onConfirm`, `title`, `description`.
**Validation**: `pnpm --dir frontend type-check`

---

### T26: Wire imperative handle and public API [M]

**Objective**: Wire the `forwardRef`/`useImperativeHandle` on SchemaEditorLite to expose `applyMetadataEdit`, `refreshPreview`, `isDirty` to parent components.
**Size**: M (integration wiring)
**Files**:
- Modify: `frontend/src/react/components/SchemaEditorLite/SchemaEditorLite.tsx`
- Modify: `frontend/src/react/components/SchemaEditorLite/context.tsx`
**Implementation**:
1. In `context.tsx`: Add `rebuildMetadataEdit` and `applyMetadataEdit` from `useAlgorithm` to the context value. Add `mergeMetadata()` implementation (port from Vue SchemaEditorLite.vue `mergeTableMetadataToTarge`).
2. In `SchemaEditorLite.tsx`: Replace stub imperative handle with real implementations. `applyMetadataEdit` → calls context's applyMetadataEdit for each target. `refreshPreview` → calls context's refreshPreview callback. `isDirty` → reads context's isDirty value. Wire `rebuild-edit-status` equivalent: `useEffect` that calls `rebuildMetadataEdit` for each target when triggered.
**Boundaries**: Does NOT change the public prop interface.
**Dependencies**: T8 (algorithm), all panel tasks
**Expected Outcome**: Parent components can call imperative methods on SchemaEditorLite ref
**Validation**: `pnpm --dir frontend type-check`

---

### T27: Add i18n keys for React schema editor [M]

**Objective**: Add all user-facing strings to the React locale files. Traces to migration playbook requirement.
**Size**: M (locale key additions across 5 language files)
**Files**:
- Modify: `frontend/src/react/locales/en-US.json`
- Modify: `frontend/src/react/locales/zh-CN.json` (use English placeholders for untranslated)
- Modify: `frontend/src/react/locales/ja-JP.json` (use English placeholders)
- Modify: `frontend/src/react/locales/es-ES.json` (use English placeholders)
- Modify: `frontend/src/react/locales/vi-VN.json` (use English placeholders)
**Implementation**:
1. Add keys under `"schema-editor"` namespace for all user-facing strings: toolbar buttons (add-column, add-index, add-partition, columns, indexes, partitions), dialog titles (create-table, rename-table, create-schema, create-view, etc.), validation messages (duplicate-name, invalid-name), status labels (created, updated, dropped), context menu items (create, rename, drop, restore), preview pane labels.
2. Extract string literals from Vue component templates as reference.
**Boundaries**: Only en-US gets real translations. Other locales get English placeholders.
**Dependencies**: All component tasks
**Expected Outcome**: All `t()` calls in React components resolve to strings
**Validation**: `pnpm --dir frontend check` — no missing i18n keys

---

### T28: Final validation [M]

**Objective**: Run full frontend validation suite to confirm no regressions.
**Size**: M (multiple validation commands)
**Files**: No file changes
**Implementation**:
1. `pnpm --dir frontend fix` — auto-fix ESLint + Biome
2. `pnpm --dir frontend check` — validate
3. `pnpm --dir frontend type-check` — TypeScript
4. `pnpm --dir frontend test` — unit tests
5. Verify Vue SchemaEditorLite is untouched: `git diff frontend/src/components/SchemaEditorLite/` — no changes
**Boundaries**: Does NOT include manual browser testing.
**Dependencies**: All tasks
**Expected Outcome**: All 4 validation commands pass, Vue files unmodified
**Validation**: All commands exit 0

---

## Out-of-Scope Tasks

- **SchemaDiagram migration** — separate surface with different callers (ER diagram viewer)
- **SchemaPane migration** — separate surface in SQL editor sidebar
- **Vue caller migration** — Plan/Issue pages, SQL editor panels continue using Vue SchemaEditorLite
- **Virtual scroll optimization for TableColumnEditor** — standard table in first pass, virtualization added later if needed
- **Schema diagram sub-tab in DatabaseEditor** — shows placeholder/hidden in React version
- **Browser/E2E testing** — requires running dev server with database backends
- **Vue file deletion** — Vue SchemaEditorLite files remain for existing callers
