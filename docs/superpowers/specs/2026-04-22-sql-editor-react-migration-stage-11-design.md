# SQL Editor React Migration — Stage 11 Design

**Date:** 2026-04-22
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Unblock future React migration of the entire WorksheetPane + tree-based surfaces by delivering two pieces of reusable infrastructure and one real consumer:

1. **Hoist the `useSheetContextByView` Vue `provide/inject` context into a Pinia store** (`useSQLEditorSheetStore`) so React components can read the same state the Vue components read.
2. **Build a shadcn-style `Tree` primitive** in `components/ui/tree.tsx` wrapping `react-arborist` (already a project dep, proven in `AsideTree.tsx`).
3. **Port `FolderForm.vue` → React** using the new store + primitive. Swap the single caller (`SaveSheetModal.vue`) to `ReactPageMount`.

**Non-goals (Stage 11):**
- Migrating `SaveSheetModal.vue` itself (deferred — its other internals are unchanged; next attempt at SaveSheetModal will reuse the React `FolderForm` built here).
- Migrating the other 6 `useSheetContextByView` consumers (`SheetTree.vue`, `WorksheetPane.vue`, `TabItem/Prefix.vue`, `TreeNodeSuffix.vue`) — they continue to work via the new Pinia store and migrate in future stages.
- Migrating the other 7 `NTree` consumers — the `Tree` primitive is built with only FolderForm's needs in scope. Additional features (drag-drop, virtualization beyond react-arborist defaults, multi-select) are deferred to future stages when specific consumers need them.
- Advanced `react-arborist` features not used by FolderForm (drag-drop reordering, multi-select, row nesting customization).

## 2. Architecture overview

```
Phase 1 — Sheet context → Pinia
  frontend/src/store/modules/sqlEditor/worksheetSheet.ts      (new)
  frontend/src/views/sql-editor/Sheet/context.ts              (refactor)
  frontend/src/views/sql-editor/Sheet/folder.ts               (refactor)
  frontend/src/views/sql-editor/Sheet/ProvideSheetContext.vue (delete or shrink)
  7 Vue files update imports                                  (modify)

Phase 2 — Tree primitive
  frontend/src/react/components/ui/tree.tsx                   (new)

Phase 3 — React FolderForm
  frontend/src/react/components/sql-editor/FolderForm.tsx     (new)
  frontend/src/react/components/sql-editor/TreeNodePrefix.tsx (new)

Phase 4 — Caller swap
  frontend/src/views/sql-editor/EditorCommon/SaveSheetModal.vue (modify)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/FolderForm.vue (delete)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/TreeNodePrefix.vue (delete)
```

Each phase is commitable independently.

## 3. Phase 1 — Sheet context → Pinia

### 3.1 Current shape (for reference)

`useSheetContextByView(view)` returns per-view state via Vue `inject`:

```
SheetContext = {
  viewContexts: {
    my: ViewContext,
    shared: ViewContext,
    draft: ViewContext,
  }
}

ViewContext = {
  folderContext: FolderContext,   // rootPath, folders, ensureFolderPath, mergeFolders, localCache
  folderTree,                     // ComputedRef<WorksheetFolderNode>
  isInitialized,
  isLoading,
  events,                         // Emittery
  expandedKeys,
  selectedKeys,
  editingNode,
  filter, filterChanged,
  isWorksheetCreator,
  getFoldersForWorksheet,
  batchUpdateWorksheetFolders,
}
```

Dependencies on upstream stores: `useSQLEditorStore` (for `project`), `useSQLEditorTabStore`, `useWorkSheetStore`, `useCurrentUserV1`.

### 3.2 New Pinia store shape

**File:** `frontend/src/store/modules/sqlEditor/worksheetSheet.ts`

```ts
export const useSQLEditorSheetStore = defineStore("sqlEditorSheet", () => {
  // Internal per-view state lazily built on first access.
  const contexts = reactive(new Map<SheetViewMode, ViewContext>());

  const ensureContext = (view: SheetViewMode): ViewContext => {
    if (!contexts.has(view)) {
      contexts.set(view, buildViewContext(view));
    }
    return contexts.get(view)!;
  };

  const getContextByView = (view: SheetViewMode) => ensureContext(view);

  return { getContextByView };
});
```

Each `ViewContext` is the same shape as before. `buildViewContext(view)` performs the composable setup that currently happens inside `useSheetTreeByView` + `useFolderByView`. Calling `getContextByView("my")` twice returns the same instance.

**Why a Pinia store wrapping a Map instead of 3 stores:** view-mode is dynamic; the callers request by string. A single store with `getContextByView(view)` is simpler than 3 parallel stores and matches the current `viewContexts[view]` access pattern.

**Event bus:** each view's `events` Emittery moves inside `buildViewContext`; consumers continue to subscribe/emit as before. React consumers can subscribe via `useEmitteryEventListener` from Stage 8 infra.

### 3.3 Migration steps

1. Create `worksheetSheet.ts` — copy the body of `useSheetTreeByView` + `useFolderByView` into `buildViewContext`, returning the same fields.
2. Rewrite `frontend/src/views/sql-editor/Sheet/context.ts`:
   - `useSheetContextByView(view)` delegates to `useSQLEditorSheetStore().getContextByView(view)`. Keep the function for source-compat; mark `useSheetContext()` and the `KEY` InjectionKey as deprecated (or delete if no remaining users).
3. `Sheet/folder.ts` → fold logic into `buildViewContext` or keep as a helper that `buildViewContext` calls (whichever keeps the diff smaller).
4. `Sheet/ProvideSheetContext.vue` — if it only `provide()`s the context, delete it and remove its mount point. If it does other setup (watchers, route hooks), keep the shell but drop the provide.
5. Update 7 Vue consumers — in most cases the call site `useSheetContextByView(view)` stays identical; only the implementation moved. Run a grep to confirm.

### 3.4 Backward-compat verification

After Phase 1 ships, every Vue caller must render identically. Verification:
- `pnpm --dir frontend fix && type-check && test` clean
- Manual: open SQL editor, switch tabs, save a worksheet into a folder, verify the tree renders + updates + filters + drags correctly. Same behavior expected.

## 4. Phase 2 — Tree primitive

### 4.1 File

`frontend/src/react/components/ui/tree.tsx`

### 4.2 Props (render-prop API — matches `AsideTree.tsx` pattern)

```tsx
interface TreeNode<T> {
  readonly id: string;                      // stable key
  readonly children?: readonly TreeNode<T>[];
  readonly data: T;                          // caller's payload
}

interface TreeProps<T> {
  readonly data: readonly TreeNode<T>[];
  readonly renderNode: (args: {
    node: NodeApi<TreeNode<T>>;              // react-arborist node API
    style: CSSProperties;
  }) => ReactNode;

  readonly selectedIds?: readonly string[];
  readonly expandedIds?: readonly string[];
  readonly onSelect?: (ids: readonly string[]) => void;
  readonly onToggle?: (id: string) => void;

  readonly searchTerm?: string;              // highlighting + filtering via searchMatch
  readonly searchMatch?: (node: TreeNode<T>, term: string) => boolean;

  readonly height?: number;                  // defaults to 300px
  readonly rowHeight?: number;               // defaults to 28px
  readonly indent?: number;                  // defaults to 16px

  readonly className?: string;
}
```

### 4.3 Implementation notes

- Wraps `Tree` from `react-arborist` inside a styled container. Default height 300, row 28, indent 16.
- Selection is controlled: `selectedIds` drives `<Tree selection={ids[0]}>` for single-select. `onSelect` fires on row click.
- Expansion is controlled via `openByDefault={false}` + programmatic `tree.open(id)` / `tree.close(id)` through a ref. The component manages the bridge so callers just pass `expandedIds`.
- Search: `searchTerm` passed to react-arborist; falls back to node's `id` + `data.label` if `searchMatch` not provided. FolderForm will provide its own matcher.
- Styling: `LAYER_SURFACE_CLASS` is NOT used (tree is a normal in-flow element, not an overlay). Tailwind-only, uses `text-control`, `bg-accent/10` for selected row, `hover:bg-accent/5` for hover row. Focus ring via `focus-visible:ring-accent`.
- No drag-drop, no multi-select (YAGNI for Stage 11).

### 4.4 Test coverage

`tree.test.tsx`:
- Renders nodes with the render function
- Clicking a row fires `onSelect` with the row id
- `selectedIds` prop reflects selection visually
- `expandedIds` controls open/close
- `searchTerm` filters visible nodes

## 5. Phase 3 — React FolderForm + TreeNodePrefix

### 5.1 Files

- `frontend/src/react/components/sql-editor/FolderForm.tsx` (~150 lines)
- `frontend/src/react/components/sql-editor/TreeNodePrefix.tsx` (~40 lines)

### 5.2 FolderForm.tsx

**Props:**
```tsx
type Props = {
  readonly folder: string;
  readonly onFolderChange: (folder: string) => void;
};
```

**Shape:**
- `Popover` (the shadcn primitive built in Stage 6) with manual trigger mode
- Trigger = an `<Input>` showing `formattedFolderPath` (slash-joined, rootPath stripped)
- Content = `<Tree>` fed by the Pinia store's `getContextByView("my")`
- `onFocus` on input → open popover
- Clicking outside → close popover (use `useOnClickOutside` — either import an existing hook or write a 15-line `useOnClickOutside` hook inline)
- Typing in input → updates folderPath directly (adds trailing slash handling matching Vue logic lines 107–120)
- Selecting a tree node → sets folderPath + closes popover

**State:**
- `folderPath` (local, synced to/from `props.folder`)
- `expandedIds` — `Set<string>`
- `showPopover` — boolean
- `searchTerm` — `folderPath` value (for tree filter)

**Store access:** `useSQLEditorSheetStore().getContextByView("my")` at top level, then `useVueState` for `folderTree`, `folderContext.rootPath`.

**Vue `defineExpose` contract removal:** Vue FolderForm exposes `folders: computed(...)`. The Vue caller SaveSheetModal reads `folderFormRef.value?.folders` via template ref. This won't work across the React boundary. Solution:
- Vue `SaveSheetModal.vue` already has access to `useSheetContextByView("my").getFoldersForWorksheet`. After Phase 1 (Pinia), it can compute `folders` itself from the current `folder` state. Drop the `folderFormRef.value.folders` read; derive from the Pinia store using the current `folder` value.
- This is a Phase 4 adjustment, not a Phase 3 one.

### 5.3 TreeNodePrefix.tsx

Direct port of `TreeNodePrefix.vue` — icon selection logic based on node state. No store access. Pure props:

```tsx
type Props = {
  readonly node: WorksheetFolderNode;
  readonly expandedIds: ReadonlySet<string>;
  readonly rootPath: string;
  readonly view: SheetViewMode;
};
```

Uses `lucide-react` (already in use) for `FileCode`, `FolderCode`, `FolderMinus`, `FolderOpen`, `FolderPen`, `FolderSync`.

### 5.4 Register page mount

`frontend/src/react/mount.ts` glob already covers `./components/sql-editor/*.tsx`. Verify no change needed.

### 5.5 Tests

- `FolderForm.test.tsx` — 4 tests: renders input with formatted path, clicking input opens popover, selecting a tree node updates folder + closes popover, typing in input updates folderPath with slash normalization
- `TreeNodePrefix.test.tsx` — 3 tests: renders file icon for worksheet node, folder-open for expanded folder, root-variant icons for each view mode

## 6. Phase 4 — SaveSheetModal caller swap

### 6.1 SaveSheetModal.vue change

Find the `<FolderForm>` tag and replace:

```vue
<FolderForm :folder="folder" @update:folder="onFolderChange" />
```

With:

```vue
<ReactPageMount
  page="FolderForm"
  :folder="folder"
  :onFolderChange="onFolderChange"
/>
```

Remember: explicit camelCase `:onFolderChange`, not `:on-folder-change`.

Drop the Vue `FolderForm` import. Add `ReactPageMount` import.

**Folders computation:** Vue currently does `folderFormRef.value?.folders`. Replace with an inline computed that calls `sheetStore.getContextByView("my").getFoldersForWorksheet(folder)` directly. This reads from the Pinia store Phase 1 produced.

### 6.2 Vue file deletions

After the swap is wired and verified:
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/FolderForm.vue` — delete
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/TreeNodePrefix.vue` — delete

Verify via `rg "FolderForm\.vue|TreeNodePrefix\.vue"` that no remaining imports exist.

## 7. Testing & verification

### 7.1 Per-phase checks

After each phase: `pnpm --dir frontend fix && check && type-check && test --run` must be green.

### 7.2 Manual UX (after Phase 4)

1. Open SQL editor → unsaved worksheet with content → cmd+S
2. Save modal opens → click folder input → popover with tree appears
3. Tree shows existing folders under "my" root with correct prefix icons
4. Click a folder → path updates in input, popover closes
5. Type "foo/bar" in input → path updates, tree filters accordingly
6. Submit save → worksheet saved into the picked folder path
7. Open save modal again on a different tab → fresh state, no cross-tab leak

### 7.3 Regression checks

After Phase 1 (before Phase 2-4):
- WorksheetPane (Vue) renders identically — folders expand/collapse, filter works, drag-sort works
- TabItem/Prefix shows correct folder badge
- Rename/move/delete folders still work

## 8. Checklist

### Phase 1
- [ ] `useSQLEditorSheetStore` Pinia store created in `store/modules/sqlEditor/worksheetSheet.ts`
- [ ] `Sheet/context.ts` refactored to delegate to Pinia store
- [ ] `Sheet/folder.ts` logic moved/referenced from the store
- [ ] `Sheet/ProvideSheetContext.vue` — deleted if only provides context; else shrunk
- [ ] All 7 Vue consumers import paths work without modification (grep confirms)
- [ ] `fix && type-check && test --run` green
- [ ] Manual verification of WorksheetPane + SaveSheetModal behavior unchanged

### Phase 2
- [ ] `components/ui/tree.tsx` created with render-prop API
- [ ] `tree.test.tsx` with 5 scenarios
- [ ] `fix && type-check && test --run` green

### Phase 3
- [ ] `FolderForm.tsx` + `TreeNodePrefix.tsx` created
- [ ] Tests for both
- [ ] Uses Pinia store from Phase 1, Tree primitive from Phase 2
- [ ] `fix && type-check && test --run` green

### Phase 4
- [ ] `SaveSheetModal.vue` swapped to `ReactPageMount`
- [ ] `folders` derivation moved from template ref to inline store read
- [ ] `FolderForm.vue` + `TreeNodePrefix.vue` deleted
- [ ] `rg FolderForm.vue|TreeNodePrefix.vue` returns no results
- [ ] Manual UX §7.2 verified

## 7. Phase 5 — Retire `SaveSheetModal.vue`

Added after Phase 4 to close the loop opened in Stage 8 (SaveSheetModal React attempt was reverted because FolderForm wasn't ready). All blockers are now resolved by Phases 1-4.

### 7.1 Prereq audit (already satisfied)

- `save-sheet` event flows through `sqlEditorEvents` singleton (Stage 8) — React subscribes via `useSQLEditorEvent`
- `maybeUpdateWorksheet`, `createWorksheet`, `abortAutoSave` live in `useSQLEditorWorksheetStore` Pinia (Stage 8)
- `useSheetContextByView` is Pinia (Phase 1 of this stage)
- React `FolderForm` exists and accepts `{ folder, onFolderChange }` (Phase 3 of this stage)
- shadcn `Dialog`, `Input`, `Button` primitives exist

No new infrastructure needed. Direct shell port.

### 7.2 Files

**New:**
- `frontend/src/react/components/sql-editor/SaveSheetModal.tsx` (~150 lines)
- `frontend/src/react/components/sql-editor/SaveSheetModal.test.tsx` (~200 lines, 5 tests)

**Modify:**
- `frontend/src/views/sql-editor/EditorPanel/StandardPanel/StandardPanel.vue` — swap `<SaveSheetModal />` → `<ReactPageMount page="SaveSheetModal" />`. No props.
- `frontend/src/views/sql-editor/EditorCommon/index.ts` — drop the `SaveSheetModal` barrel export.

**Delete:**
- `frontend/src/views/sql-editor/EditorCommon/SaveSheetModal.vue`

### 7.3 Component shape

```tsx
export function SaveSheetModal() {
  const { t } = useTranslation();
  const worksheetStore = useWorkSheetStore();
  const editorWorksheetStore = useSQLEditorWorksheetStore();
  const sheetContext = useSheetContextByView("my");

  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [folder, setFolder] = useState("");
  const [rawTab, setRawTab] = useState<SQLEditorTab | undefined>(undefined);

  useSQLEditorEvent("save-sheet", ({ tab, editTitle }) => { ... });

  const doSaveSheet = async (tab?: SQLEditorTab) => { ... };

  return (
    <Dialog open={open} onOpenChange={(next) => !next && setOpen(false)}>
      <DialogContent className="max-w-lg">
        <DialogTitle>{t("sql-editor.save-sheet")}</DialogTitle>
        {/* title input + FolderForm (React, direct import) + Close/Save buttons */}
      </DialogContent>
    </Dialog>
  );
}
```

### 7.4 Full-parity behaviors (audit checklist)

1. Event listener sets `title`, `folder` (empty initially), `rawTab` synchronously on every `save-sheet`
2. `needShowModal(tab) = !tab.worksheet`
3. If `needShowModal` OR `editTitle`:
   - `worksheetStore.getWorksheetByName(tab.worksheet)` → if found, `folder = sheetContext.getPwdForWorksheet(worksheet)`
   - Show modal
4. Else: call `doSaveSheet(tab)` immediately, no modal
5. `doSaveSheet`:
   - Guard on missing `rawTab` and empty `title`
   - `abortAutoSave()` first
   - `folders = sheetContext.getFoldersForWorksheet(folder)`
   - `extractWorksheetID(worksheet ?? "") !== String(UNKNOWN_ID)` → `maybeUpdateWorksheet`, else `createWorksheet`
   - Close modal
6. Save button disabled when `title === ""`

### 7.5 Dialog vs Sheet

**Dialog** — focused 2-field save action, matches Vue `BBModal`'s centered-card shape. Not a multi-section resource edit, so Sheet would be overkill per `frontend/AGENTS.md`.

Width: `max-w-lg` (32rem) — mirrors Vue's `w-lg max-w-[calc(100vw-8rem)]`.

### 7.6 Tests

1. Modal opens on `save-sheet` for an unsaved tab, title prefilled
2. Modal does NOT open for a saved tab without `editTitle`; `maybeUpdateWorksheet` called instead
3. Modal opens for a saved tab when `editTitle: true`
4. Save button disabled when title empty
5. Clicking Save on an unsaved tab calls `createWorksheet` with the right args

Mock `FolderForm` so the test doesn't re-test the tree (already covered in Phase 3 tests).

### 7.7 Phase 5 checklist

- [ ] `SaveSheetModal.tsx` created
- [ ] `SaveSheetModal.test.tsx` created with 5 tests
- [ ] `StandardPanel.vue` swapped to `ReactPageMount`
- [ ] `EditorCommon/index.ts` dropped the `SaveSheetModal` export
- [ ] `SaveSheetModal.vue` deleted
- [ ] `rg "SaveSheetModal\.vue"` returns zero results
- [ ] `pnpm fix && type-check && test --run && check` all green

## 8. Manual UX (after all phases)

**Phase 1-4 flows:**

1. Open SQL editor → unsaved tab → cmd+S → save modal opens (now React)
2. Click folder input → tree popover, width matches input
3. Pick folder → path updates, popover closes
4. Type `foo/bar` → path normalizes, tree filters
5. Worksheet pane → select multiple → "Move to folder" modal (React FolderForm embedded)

**Phase 5 flows (new):**

6. Modal save flow end-to-end: title + folder → Save → worksheet created or updated in the correct folder
7. Right-click saved tab → Rename → modal opens with title + current folder prefilled
8. Saved tab without editTitle → cmd+S → no modal, save proceeds silently

## 9. Out of scope (deferred)

- Migrating `SheetTree.vue`, `WorksheetPane.vue` (shell), `TabItem/Prefix.vue`, `TreeNodeSuffix.vue` — they benefit from the Pinia hoist but remain Vue
- Other NTree consumers — the primitive is ready for them when their stages come
- Drag-drop / multi-select on the Tree primitive — add when a consumer needs it
- AI plugin context hoist (separate future stage)
- EditorAction shell migration (deferred)
- Dead-code cleanup of `Sheet/folder.ts` (Phase 1 left it unimported; separate cleanup)
