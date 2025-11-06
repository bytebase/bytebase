# Schema Editor as Optional Helper - Design Document

**Date:** 2025-11-06
**Status:** Approved
**Author:** Design review with user

## Overview

This design changes the Schema Editor from a mandatory modal/step to an optional helper tool accessible from the SQL editor. This makes the "Edit Schema" experience consistent with "Change Data" - users go directly to a draft plan with SQL editor, and can optionally invoke the Schema Editor to assist with SQL generation.

## Problem Statement

Currently, clicking "Edit Schema" opens a modal popup with the Schema Editor, which feels heavy and interrupts the workflow. Users must use the Schema Editor even when they just want to write SQL directly. This differs from "Change Data" which goes straight to the SQL editor.

The user wants:
- No modal popup when clicking "Edit Schema"
- Direct navigation to SQL editor (like "Change Data")
- Schema Editor as an optional helper tool, not mandatory

## Goals

1. Remove mandatory Schema Editor modal from all entry points
2. Make "Edit Schema" behave like "Change Data" - direct to SQL editor
3. Add optional Schema Editor button in SQL editor as a helper tool
4. Ensure consistent behavior across all flows (single DB, batch, AddSpecDrawer)

## Non-Goals

- Removing Schema Editor functionality entirely
- Supporting Schema Editor for databases that don't currently support it
- Multi-database schema editing in the drawer

## Design

### Current Flows

**Edit Schema (single database):**
```
Click "Edit Schema" → Schema Editor Modal → Edit → Preview Issue → Navigate to issue page
```

**Edit Schema (batch operations):**
```
Select DBs → Click "Edit Schema" → Schema Editor Modal → Edit → Preview Issue → Navigate to issue page
```

**AddSpecDrawer:**
```
Click "Add Change" → Step 1: Select Type (DDL/DML) →
Step 2: Select DBs → Step 3: Schema Editor (DDL only) → Create Spec
```

### New Unified Flow

**All entry points:**
```
Click "Edit Schema" (or AddSpecDrawer with DDL) → Draft Plan with SQL Editor →
  (Optional) Click "Schema Editor" button → Drawer opens →
  Edit schema visually → Generate SQL → Click "Insert SQL" → SQL appears in editor
```

### Architecture Changes

#### 1. Remove Schema Editor Modal from "Edit Schema" Buttons

**Files:**
- `frontend/src/views/DatabaseDetail/DatabaseDetail.vue`
- `frontend/src/components/v2/Model/DatabaseV1Table/DatabaseOperations.vue`

**Changes:**
- Remove conditional logic that opens `SchemaEditorModal` for supported engines
- Navigate directly to plan creation route for all databases
- Use same navigation pattern as "Change Data" but with `bb.issue.database.schema.update` template

**Before (DatabaseDetail.vue:354-361):**
```typescript
if (type === "bb.issue.database.schema.update") {
  if (
    database.value.state === State.ACTIVE &&
    engineSupportsSchemaEditor(database.value.instanceResource.engine)
  ) {
    state.showSchemaEditorModal = true;
    return;
  }
}
```

**After:**
```typescript
// Remove the conditional - always navigate directly to plan creation
```

**Navigation target:**
```typescript
Route: PROJECT_V1_ROUTE_ISSUE_DETAIL
Params: {
  projectId,
  issueSlug: "create",
  planId: "create",
  specId: "placeholder"
}
Query: {
  template: "bb.issue.database.schema.update",
  databaseList: "comma,separated,db,names",
  name: "[DbName] Edit schema @MM-DD HH:mm"
}
```

#### 2. Remove Schema Editor Step from AddSpecDrawer

**File:** `frontend/src/components/Plan/components/AddSpecDrawer.vue`

**Changes:**
- Remove `Step.SCHEMA_EDITOR` from steps enum
- Remove Step 3 UI rendering (lines 18-21, 97-116)
- Remove schema editor footer buttons (lines 124-130)
- Remove related computed properties: `shouldShowSchemaEditor`, `schemaEditTargets`, `isPreparingMetadata`
- Remove `handlePreviewDDL` method
- Simplify to 2 steps only: Select Change Type → Select Targets → Done

**Before:**
```
NSteps:
  - Step 1: Change Type
  - Step 2: Select Targets
  - Step 3: Schema Editor (if DDL + single DB + supported engine)
```

**After:**
```
NSteps:
  - Step 1: Change Type
  - Step 2: Select Targets
(Both DDL and DML follow same flow)
```

#### 3. Create Schema Editor Drawer Component

**New File:** `frontend/src/components/Plan/components/StatementSection/SchemaEditorDrawer.vue`

**Component Structure:**
```vue
<template>
  <Drawer
    v-model:show="show"
    placement="right"
    :resizable="true"
    :default-width="'50%'"
  >
    <DrawerContent :title="$t('schema-editor.self')">
      <SchemaEditorLite
        ref="schemaEditorRef"
        :project="project"
        :targets="editTargets"
      />

      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="handleCancel">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!hasPendingChanges"
            @click="handleInsertSQL"
          >
            {{ $t("schema-editor.insert-sql") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>
```

**Props:**
- `show` (v-model): Boolean to control drawer visibility
- `database`: Target database for schema editing

**Emits:**
- `insert`: Emits generated SQL string when user clicks "Insert SQL"

**Features:**
- Resizable width (default 50%, min 600px, max 90%)
- Reuses existing `SchemaEditorLite` component
- Shows real-time diff/preview of schema changes
- Generates DDL on "Insert SQL" click
- Closes drawer after inserting SQL

**Visibility Conditions:**
- Migration type is DDL (schema change)
- Database engine supports schema editor (MySQL, TiDB, PostgreSQL)
- Single database only (not multi-DB batch operations)

#### 4. Integrate Drawer into SQL Editor

**File:** `frontend/src/components/Plan/components/StatementSection/EditorView/EditorView.vue`

**Add "Schema Editor" Button:**

Place in action buttons section (after SQL Upload button):

```vue
<NButton
  v-if="shouldShowSchemaEditorButton"
  size="small"
  @click="handleOpenSchemaEditor"
>
  <template #icon>
    <TableIcon />
  </template>
  {{ $t("schema-editor.self") }}
</NButton>
```

**Button positioning:**
- When creating: After "SQL Upload"
- When editing: After "SQL Upload", before "Save"/"Cancel"
- When not editing: After "Edit" button

**Visibility Logic:**

```typescript
const shouldShowSchemaEditorButton = computed(() => {
  // Only for DDL (schema) changes
  const spec = selectedSpec.value;
  if (!spec?.config.value || spec.config.value.migrationType !== MigrationType.DDL) {
    return false;
  }

  // Only if database engine supports schema editor
  const db = database.value;
  if (!engineSupportsSchemaEditor(db.instanceResource.engine)) {
    return false;
  }

  // Only for single database (not batch)
  const targets = spec.config.value.targets || [];
  if (targets.length !== 1) {
    return false;
  }

  return true;
});
```

**State & Handlers:**

```typescript
const state = reactive({
  // ... existing state
  showSchemaEditorDrawer: false,
});

const handleOpenSchemaEditor = () => {
  state.showSchemaEditorDrawer = true;
};

const handleInsertSQL = (sql: string) => {
  // Append generated SQL to existing content
  const currentSQL = state.statement;
  const newSQL = currentSQL ? `${currentSQL}\n\n${sql}` : sql;
  handleUpdateStatement(newSQL);
  state.showSchemaEditorDrawer = false;
};
```

**Add Drawer Component:**

```vue
<SchemaEditorDrawer
  v-if="shouldShowSchemaEditorButton"
  v-model:show="state.showSchemaEditorDrawer"
  :database="database"
  @insert="handleInsertSQL"
/>
```

## Implementation Plan

### Phase 1: Remove Schema Editor from Mandatory Flows

**Step 1:** Modify `DatabaseDetail.vue`
- File: `frontend/src/views/DatabaseDetail/DatabaseDetail.vue`
- Remove lines 354-361 (conditional modal opening)
- Keep navigation logic for both schema and data changes

**Step 2:** Modify `DatabaseOperations.vue`
- File: `frontend/src/components/v2/Model/DatabaseV1Table/DatabaseOperations.vue`
- Remove lines 340-347 (conditional modal opening)
- Navigate directly to plan creation

**Step 3:** Modify `AddSpecDrawer.vue`
- File: `frontend/src/components/Plan/components/AddSpecDrawer.vue`
- Remove Step 3 (Schema Editor) UI and logic
- Remove `shouldShowSchemaEditor`, `schemaEditTargets`, `isPreparingMetadata`
- Remove `handlePreviewDDL` method
- Update step navigation to be 2 steps only

### Phase 2: Create Schema Editor Drawer

**Step 4:** Create `SchemaEditorDrawer.vue`
- File: `frontend/src/components/Plan/components/StatementSection/SchemaEditorDrawer.vue`
- Implement resizable drawer with `SchemaEditorLite`
- Add footer with "Insert SQL" and "Cancel" buttons
- Emit generated SQL on insert

### Phase 3: Integrate Drawer into SQL Editor

**Step 5:** Modify `EditorView.vue`
- File: `frontend/src/components/Plan/components/StatementSection/EditorView/EditorView.vue`
- Add "Schema Editor" button with visibility logic
- Add drawer state management
- Import and render `SchemaEditorDrawer`
- Handle insert event to append SQL

### Phase 4: Cleanup

**Step 6:** Mark `SchemaEditorModal.vue` as deprecated
- File: `frontend/src/components/AlterSchemaPrepForm/SchemaEditorModal.vue`
- Add deprecation comment
- Can be removed later if unused

## Data Flow

### Opening Schema Editor Drawer

```
User clicks "Schema Editor" button in SQL editor
  ↓
EditorView sets state.showSchemaEditorDrawer = true
  ↓
SchemaEditorDrawer opens with current database
  ↓
Fetches database metadata
  ↓
Initializes SchemaEditorLite with baseline metadata
```

### Inserting Generated SQL

```
User makes changes in Schema Editor
  ↓
User clicks "Insert SQL"
  ↓
SchemaEditorDrawer calls schemaEditorRef.applyMetadataEdit()
  ↓
Generates DDL diff between baseline and modified metadata
  ↓
Emits 'insert' event with SQL string
  ↓
EditorView receives SQL and appends to current statement
  ↓
Drawer closes
```

## Edge Cases & Error Handling

1. **No changes in Schema Editor:**
   - "Insert SQL" button is disabled if no pending changes
   - User must make changes to enable insertion

2. **Invalid schema changes:**
   - SchemaEditorLite handles validation internally
   - Shows error messages before allowing SQL generation

3. **Multi-database specs:**
   - Schema Editor button is hidden for multi-DB specs
   - Users must write SQL manually for batch operations

4. **Unsupported database engines:**
   - Schema Editor button is hidden
   - Only available for MySQL, TiDB, PostgreSQL

5. **Drawer state on navigation:**
   - Drawer closes automatically when user navigates away
   - Unsaved changes in drawer are lost (expected behavior)

## Testing Considerations

1. **Single database schema change:**
   - Click "Edit Schema" → Verify direct navigation to SQL editor
   - Click "Schema Editor" button → Verify drawer opens
   - Make changes → Click "Insert SQL" → Verify SQL appears in editor

2. **Batch operations:**
   - Select multiple DBs → Click "Edit Schema" → Verify direct navigation
   - Verify "Schema Editor" button is hidden (multi-DB)

3. **AddSpecDrawer flow:**
   - Select DDL type → Select single DB → Verify no Step 3
   - Verify navigation to SQL editor directly

4. **Unsupported engines:**
   - For Oracle, SQL Server, etc. → Verify "Schema Editor" button is hidden
   - Verify direct navigation still works

5. **DML (data change):**
   - Click "Change Data" → Verify "Schema Editor" button is hidden
   - Only DDL specs show the button

## Migration Notes

- Existing functionality is preserved - users can still access Schema Editor
- Old modal-based flow is replaced with drawer-based flow
- No data migration required
- No API changes required
- Frontend-only changes

## Future Enhancements

1. Multi-database schema editing in drawer (if needed)
2. Remember drawer width preference per user
3. Keyboard shortcuts to open/close drawer
4. Support for more database engines in Schema Editor

## Success Metrics

- Reduced friction when creating schema changes
- Consistent UX between "Edit Schema" and "Change Data"
- Schema Editor becomes a helper tool rather than blocking the flow
- Users can choose to write SQL directly or use visual editor
