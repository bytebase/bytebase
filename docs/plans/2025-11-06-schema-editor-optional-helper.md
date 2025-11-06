# Schema Editor as Optional Helper - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Convert Schema Editor from mandatory modal to optional helper drawer accessible from SQL editor, making "Edit Schema" behave like "Change Data" with direct navigation to SQL editor.

**Architecture:** Remove Schema Editor modal from all entry points (DatabaseDetail, DatabaseOperations, AddSpecDrawer), create new resizable drawer component with SchemaEditorLite, add optional button in SQL editor that opens drawer and inserts generated SQL.

**Tech Stack:** Vue 3, TypeScript, Naive UI, existing SchemaEditorLite component

---

## Phase 1: Remove Schema Editor from Mandatory Flows

### Task 1: Remove Schema Editor Modal from DatabaseDetail.vue

**Files:**
- Modify: `frontend/src/views/DatabaseDetail/DatabaseDetail.vue:351-380`
- Modify: `frontend/src/views/DatabaseDetail/DatabaseDetail.vue:253-260` (LocalState interface)
- Modify: `frontend/src/views/DatabaseDetail/DatabaseDetail.vue:270-277` (state initialization)

**Step 1: Remove showSchemaEditorModal from LocalState**

In `frontend/src/views/DatabaseDetail/DatabaseDetail.vue` around line 253:

```typescript
interface LocalState {
  showTransferDatabaseModal: boolean;
  showIncorrectProjectModal: boolean;
  // Remove: showSchemaEditorModal: boolean;
  currentProjectName: string;
  selectedIndex: number;
  selectedTab: DatabaseHash;
}
```

**Step 2: Remove showSchemaEditorModal from state initialization**

Around line 270:

```typescript
const state = reactive<LocalState>({
  showTransferDatabaseModal: false,
  showIncorrectProjectModal: false,
  // Remove: showSchemaEditorModal: false,
  currentProjectName: UNKNOWN_PROJECT_NAME,
  selectedIndex: 0,
  selectedTab: "overview",
});
```

**Step 3: Remove conditional modal opening logic from createMigration**

Around line 354-361, remove this block:

```typescript
// DELETE THESE LINES:
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

**Step 4: Remove SchemaEditorModal component import and usage**

Around line 202, remove import:

```typescript
// DELETE THIS LINE:
import SchemaEditorModal from "@/components/AlterSchemaPrepForm/SchemaEditorModal.vue";
```

Search for `<SchemaEditorModal` in the template and remove the component usage (likely near the end of the template).

**Step 5: Verify changes and format**

Run: `pnpm --dir frontend prettier --write src/views/DatabaseDetail/DatabaseDetail.vue`

**Step 6: Commit**

```bash
git add frontend/src/views/DatabaseDetail/DatabaseDetail.vue
git commit -m "refactor: remove Schema Editor modal from DatabaseDetail

Remove mandatory Schema Editor modal from DatabaseDetail view.
Edit Schema now navigates directly to plan creation like Change Data."
```

---

### Task 2: Remove Schema Editor Modal from DatabaseOperations.vue

**Files:**
- Modify: `frontend/src/components/v2/Model/DatabaseV1Table/DatabaseOperations.vue:340-347`
- Modify: `frontend/src/components/v2/Model/DatabaseV1Table/DatabaseOperations.vue` (state and imports)

**Step 1: Find and remove showSchemaEditorModal from state**

Search for `showSchemaEditorModal` in the file and remove it from the reactive state object.

**Step 2: Remove conditional modal opening logic from generateMultiDb**

Around line 340-347, remove this block:

```typescript
// DELETE THESE LINES:
if (
  props.databases.length === 1 &&
  type === "bb.issue.database.schema.update" &&
  allowUsingSchemaEditor(props.databases)
) {
  state.showSchemaEditorModal = true;
  return;
}
```

**Step 3: Remove SchemaEditorModal component import**

Remove the import statement for SchemaEditorModal.

**Step 4: Remove SchemaEditorModal component from template**

Search for `<SchemaEditorModal` in the template and remove the component usage.

**Step 5: Verify changes and format**

Run: `pnpm --dir frontend prettier --write src/components/v2/Model/DatabaseV1Table/DatabaseOperations.vue`

**Step 6: Commit**

```bash
git add frontend/src/components/v2/Model/DatabaseV1Table/DatabaseOperations.vue
git commit -m "refactor: remove Schema Editor modal from DatabaseOperations

Remove mandatory Schema Editor modal from batch operations.
Edit Schema now navigates directly to plan creation."
```

---

### Task 3: Remove Schema Editor Step from AddSpecDrawer.vue

**Files:**
- Modify: `frontend/src/components/Plan/components/AddSpecDrawer.vue`

**Step 1: Remove Step.SCHEMA_EDITOR from Steps enum**

Find the Steps enum definition and remove `SCHEMA_EDITOR` entry. Update the enum values if needed.

**Step 2: Remove Schema Editor step from NSteps in template**

Around line 18-21, remove:

```vue
<!-- DELETE THESE LINES: -->
<NStep
  v-if="shouldShowSchemaEditor"
  :title="$t('schema-editor.self')"
/>
```

**Step 3: Remove Schema Editor step content from template**

Around line 97-116, remove:

```vue
<!-- DELETE THIS ENTIRE BLOCK: -->
<template
  v-else-if="
    currentStep === Step.SCHEMA_EDITOR && shouldShowSchemaEditor
  "
>
  <div
    class="relative h-[600px] w-full md:w-[90vw] lg:w-[calc(100vw-20rem)] lg:min-w-[80vw]"
  >
    <MaskSpinner v-if="isPreparingMetadata" />
    <SchemaEditorLite
      v-if="schemaEditTargets.length > 0"
      ref="schemaEditorRef"
      :project="project"
      :targets="schemaEditTargets"
      :loading="isPreparingMetadata"
      :hide-preview="false"
    />
  </div>
</template>
```

**Step 4: Remove Schema Editor footer buttons**

Around line 124-130, remove:

```vue
<!-- DELETE THIS BLOCK: -->
<NButton
  v-if="
    currentStep === Step.SCHEMA_EDITOR && shouldShowSchemaEditor
  "
  :loading="isGeneratingPreview"
  @click="handlePreviewDDL"
>
  {{ $t("schema-editor.preview-schema-text") }}
</NButton>
```

**Step 5: Remove computed properties and methods**

Remove these computed properties and methods from the script section:
- `shouldShowSchemaEditor`
- `schemaEditTargets`
- `isPreparingMetadata`
- `handlePreviewDDL`
- Any imports for SchemaEditorLite if no longer used

**Step 6: Update step navigation logic**

Update `isLastStep` computed property to reflect 2 steps instead of 3.

Update `handleNextStep` to skip schema editor logic.

**Step 7: Remove Preview DDL Modal**

Around line 170-179, remove the entire Preview DDL Modal section if it exists.

**Step 8: Verify changes and format**

Run: `pnpm --dir frontend prettier --write src/components/Plan/components/AddSpecDrawer.vue`

**Step 9: Commit**

```bash
git add frontend/src/components/Plan/components/AddSpecDrawer.vue
git commit -m "refactor: remove Schema Editor step from AddSpecDrawer

Simplify AddSpecDrawer to 2 steps: Select Type and Select Targets.
Both DDL and DML now follow the same flow to SQL editor."
```

---

## Phase 2: Create Schema Editor Drawer Component

### Task 4: Create SchemaEditorDrawer.vue Component

**Files:**
- Create: `frontend/src/components/Plan/components/StatementSection/SchemaEditorDrawer.vue`

**Step 1: Create the component file with basic structure**

Create `frontend/src/components/Plan/components/StatementSection/SchemaEditorDrawer.vue`:

```vue
<template>
  <Drawer
    :show="show"
    placement="right"
    :resizable="true"
    :default-width="'50%'"
    :min-width="600"
    :max-width="'90vw'"
    @update:show="$emit('update:show', $event)"
  >
    <DrawerContent :title="$t('schema-editor.self')" closable>
      <div class="h-full flex flex-col gap-y-4">
        <MaskSpinner v-if="state.isPreparingMetadata" />

        <SchemaEditorLite
          v-if="state.editTargets.length > 0 && !state.isPreparingMetadata"
          ref="schemaEditorRef"
          :project="project"
          :targets="state.editTargets"
          :loading="state.isPreparingMetadata"
        />
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton quaternary @click="handleCancel">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!hasPendingChanges"
            :loading="state.isGenerating"
            @click="handleInsertSQL"
          >
            {{ $t("schema-editor.insert-sql") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { NButton } from "naive-ui";
import { Drawer, DrawerContent } from "@/components/v2";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import SchemaEditorLite from "@/components/SchemaEditorLite/SchemaEditorLite.vue";
import { generateDiffDDL } from "@/components/SchemaEditorLite/common";
import type { ComposedDatabase } from "@/types";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";

interface EditTarget {
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  baselineMetadata: DatabaseMetadata;
}

interface LocalState {
  isPreparingMetadata: boolean;
  isGenerating: boolean;
  editTargets: EditTarget[];
}

const props = defineProps<{
  show: boolean;
  database: ComposedDatabase;
  project: any;
}>();

const emit = defineEmits<{
  "update:show": [boolean];
  insert: [string];
}>();

const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();

const state = reactive<LocalState>({
  isPreparingMetadata: false,
  isGenerating: false,
  editTargets: [],
});

const hasPendingChanges = computed(() => {
  return schemaEditorRef.value?.isDirty ?? false;
});

const handleCancel = () => {
  emit("update:show", false);
};

const handleInsertSQL = async () => {
  if (!schemaEditorRef.value) return;

  state.isGenerating = true;
  try {
    // Apply metadata edits
    const result = await schemaEditorRef.value.applyMetadataEdit();

    // Generate DDL from the edited metadata
    const sql = await generateDiffDDL(
      state.editTargets[0].database.instanceResource.engine,
      state.editTargets[0].baselineMetadata,
      result.metadata
    );

    // Emit the generated SQL
    emit("insert", sql);
    emit("update:show", false);
  } catch (error) {
    console.error("Failed to generate DDL:", error);
  } finally {
    state.isGenerating = false;
  }
};

// Fetch metadata when drawer opens
watch(
  () => props.show,
  async (show) => {
    if (!show) return;

    state.isPreparingMetadata = true;
    try {
      // TODO: Fetch fresh metadata for the database
      // For now, use a placeholder
      const metadata = props.database.metadata;
      const baselineMetadata = structuredClone(metadata);

      state.editTargets = [
        {
          database: props.database,
          metadata: structuredClone(metadata),
          baselineMetadata,
        },
      ];
    } catch (error) {
      console.error("Failed to fetch metadata:", error);
    } finally {
      state.isPreparingMetadata = false;
    }
  },
  { immediate: true }
);
</script>
```

**Step 2: Format the file**

Run: `pnpm --dir frontend prettier --write src/components/Plan/components/StatementSection/SchemaEditorDrawer.vue`

**Step 3: Commit**

```bash
git add frontend/src/components/Plan/components/StatementSection/SchemaEditorDrawer.vue
git commit -m "feat: create SchemaEditorDrawer component

Add resizable drawer component that wraps SchemaEditorLite.
Provides Insert SQL and Cancel actions."
```

---

## Phase 3: Integrate Drawer into SQL Editor

### Task 5: Add Schema Editor Button to EditorView.vue

**Files:**
- Modify: `frontend/src/components/Plan/components/StatementSection/EditorView/EditorView.vue`

**Step 1: Import SchemaEditorDrawer and required utilities**

At the top of the script section, add imports:

```typescript
import SchemaEditorDrawer from "./SchemaEditorDrawer.vue";
import { MigrationType } from "@/types/proto-es/v1/plan_service_pb";
import { engineSupportsSchemaEditor } from "@/utils";
```

**Step 2: Add showSchemaEditorDrawer to state**

Find the state reactive object and add:

```typescript
const state = reactive({
  // ... existing state
  showSchemaEditorDrawer: false,
});
```

**Step 3: Add shouldShowSchemaEditorButton computed property**

After the state definition, add:

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

**Step 4: Add handler methods**

Add these methods:

```typescript
const handleOpenSchemaEditor = () => {
  state.showSchemaEditorDrawer = true;
};

const handleInsertSQL = (sql: string) => {
  // Append generated SQL to existing content
  const currentSQL = state.statement;
  const newSQL = currentSQL ? `${currentSQL}\n\n${sql}` : sql;
  handleUpdateStatement(newSQL);
};
```

**Step 5: Add Schema Editor button to action bar**

In the template, find the action buttons section (around line 15-83). Add the Schema Editor button after the SQL Upload button:

```vue
<!-- Add this after SQLUploadButton -->
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

Import TableIcon from lucide-vue-next if not already imported:

```typescript
import { TableIcon } from "lucide-vue-next";
```

**Step 6: Add SchemaEditorDrawer component to template**

At the end of the template (before closing `</template>`), add:

```vue
<SchemaEditorDrawer
  v-if="shouldShowSchemaEditorButton"
  v-model:show="state.showSchemaEditorDrawer"
  :database="database"
  :project="project"
  @insert="handleInsertSQL"
/>
```

**Step 7: Format the file**

Run: `pnpm --dir frontend prettier --write src/components/Plan/components/StatementSection/EditorView/EditorView.vue`

**Step 8: Commit**

```bash
git add frontend/src/components/Plan/components/StatementSection/EditorView/EditorView.vue
git commit -m "feat: add optional Schema Editor button to SQL editor

Add Schema Editor button that opens drawer for DDL specs.
Button only visible for single-database DDL changes with
supported engines (MySQL, TiDB, PostgreSQL)."
```

---

## Phase 4: Cleanup and Internationalization

### Task 6: Add i18n Key for "Insert SQL"

**Files:**
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/zh-CN.json`

**Step 1: Add "insert-sql" key to en-US.json**

In `frontend/src/locales/en-US.json`, find the `"schema-editor"` section and add:

```json
"schema-editor": {
  "self": "Schema Editor",
  "insert-sql": "Insert SQL",
  // ... other existing keys
}
```

**Step 2: Add "insert-sql" key to zh-CN.json**

In `frontend/src/locales/zh-CN.json`, find the `"schema-editor"` section and add:

```json
"schema-editor": {
  "self": "模式编辑器",
  "insert-sql": "插入 SQL",
  // ... other existing keys
}
```

**Step 3: Format the files**

Run: `pnpm --dir frontend prettier --write src/locales/en-US.json src/locales/zh-CN.json`

**Step 4: Commit**

```bash
git add frontend/src/locales/en-US.json frontend/src/locales/zh-CN.json
git commit -m "i18n: add 'insert-sql' translation key

Add translation for Insert SQL button in Schema Editor drawer."
```

---

### Task 7: Remove Unused SchemaEditorModal Files and References

**Files:**
- Delete: `frontend/src/components/AlterSchemaPrepForm/SchemaEditorModal.vue`
- Check: Other files for remaining imports

**Step 1: Search for remaining references to SchemaEditorModal**

Run: `cd frontend && grep -r "SchemaEditorModal" src/`

Expected: Should find no results (we already removed imports in earlier tasks)

**Step 2: Delete SchemaEditorModal.vue**

Run: `rm frontend/src/components/AlterSchemaPrepForm/SchemaEditorModal.vue`

**Step 3: Check if AlterSchemaPrepForm directory is now empty**

Run: `ls frontend/src/components/AlterSchemaPrepForm/`

If empty, remove the directory:

Run: `rmdir frontend/src/components/AlterSchemaPrepForm/`

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: remove unused SchemaEditorModal

Remove SchemaEditorModal component and directory as it has been
replaced by SchemaEditorDrawer."
```

---

## Phase 5: Testing and Verification

### Task 8: Run Frontend Linter and Type Checker

**Step 1: Run linter**

Run: `pnpm --dir frontend lint --fix`

Expected: No errors. If there are errors, fix them before proceeding.

**Step 2: Run type checker**

Run: `pnpm --dir frontend type-check`

Expected: No type errors. Fix any type issues.

**Step 3: Format all modified files**

Run: `pnpm --dir frontend prettier --write "src/**/*.{vue,ts,js}"`

**Step 4: Commit any fixes**

If there were lint/type fixes:

```bash
git add frontend/src
git commit -m "fix: resolve lint and type errors"
```

---

### Task 9: Manual Testing Checklist

**Test Scenario 1: Single Database Edit Schema**

1. Navigate to a single database detail page (MySQL, TiDB, or PostgreSQL)
2. Click "Edit Schema" button
3. Verify: Should navigate directly to plan creation with SQL editor (no modal)
4. Verify: "Schema Editor" button should appear in the action bar
5. Click "Schema Editor" button
6. Verify: Drawer opens from the right with SchemaEditorLite
7. Make a schema change (e.g., add a column)
8. Click "Insert SQL" button
9. Verify: Generated DDL appears in the SQL editor
10. Verify: Drawer closes

**Test Scenario 2: Batch Operations Edit Schema**

1. Select multiple databases in database list
2. Click "Edit Schema" from batch operations toolbar
3. Verify: Should navigate directly to plan creation with SQL editor (no modal)
4. Verify: "Schema Editor" button should NOT appear (multi-DB not supported)

**Test Scenario 3: AddSpecDrawer Flow**

1. Go to a project's plan page
2. Click "Add Change" button
3. Select "Schema Migration" (DDL)
4. Select a single database
5. Verify: Should show only 2 steps (no Schema Editor step)
6. Click "Confirm"
7. Verify: Navigate to SQL editor directly
8. Verify: "Schema Editor" button appears

**Test Scenario 4: Change Data (DML) Flow**

1. Navigate to a database detail page
2. Click "Change Data" button
3. Verify: Navigates to SQL editor
4. Verify: "Schema Editor" button does NOT appear (DML, not DDL)

**Test Scenario 5: Unsupported Database Engine**

1. Navigate to a database that doesn't support schema editor (e.g., Oracle, SQL Server)
2. Click "Edit Schema"
3. Verify: Navigates to SQL editor directly
4. Verify: "Schema Editor" button does NOT appear

---

## Rollback Plan

If issues are found during testing:

1. **Revert all commits in this branch:**
   ```bash
   git reset --hard main
   ```

2. **Or revert specific problematic commits:**
   ```bash
   git revert <commit-hash>
   ```

3. **Return to main worktree:**
   ```bash
   cd ../..
   git worktree remove .worktrees/schema-editor-optional
   ```

---

## Success Criteria

- ✅ No Schema Editor modal appears when clicking "Edit Schema"
- ✅ "Edit Schema" navigates directly to SQL editor (like "Change Data")
- ✅ AddSpecDrawer has only 2 steps for both DDL and DML
- ✅ "Schema Editor" button appears in SQL editor for DDL single-DB specs
- ✅ "Schema Editor" button opens drawer from the right
- ✅ Generated SQL inserts correctly into the editor
- ✅ No lint or type errors
- ✅ SchemaEditorModal.vue and directory removed
- ✅ All tests pass (if applicable)

---

## Notes for Engineer

- **DRY**: Reuse existing `SchemaEditorLite` component; don't duplicate schema editing logic
- **YAGNI**: Don't add multi-database support in drawer unless explicitly requested
- **TDD**: Not strictly applicable here as this is primarily UI refactoring, but verify each change works before moving on
- **Commit frequently**: After each task completion
- **Test as you go**: Don't wait until the end to test; verify each phase works before proceeding

---
