# Remove Archived Settings Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the dedicated archived settings page and integrate archived items into the Projects and Instances list pages with state filtering.

**Architecture:** Add a state filter dropdown (Active/Archived/All) to both list pages, update hard-delete redirects to navigate to list pages instead of archive page, remove archive route and sidebar menu item.

**Tech Stack:** Vue 3, Naive UI (NSelect), Vue Router, TypeScript

---

### Task 1: Add State Filter to Projects Page

**Files:**
- Modify: `frontend/src/views/ProjectDashboard.vue:1-161`

**Step 1: Add state selection to local state**

Add state property to the `LocalState` interface and reactive state:

```typescript
// In LocalState interface (around line 80)
interface LocalState {
  params: SearchParams;
  showCreateDrawer: boolean;
  selectedProjects: Set<string>;
  selectedState: State | "ALL"; // Add this
}

// In reactive state initialization (around line 90)
const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
  selectedProjects: new Set(),
  selectedState: State.ACTIVE, // Add this - default to ACTIVE
});
```

**Step 2: Import State and NSelect**

Update imports:

```typescript
// Add to existing imports around line 61-62
import { State } from "@/types/proto-es/v1/common_pb";
import { NButton, NSelect } from "naive-ui"; // Add NSelect to existing NButton import
```

**Step 3: Add state filter to computed filter**

Update the `filter` computed property to include state:

```typescript
// Around line 112-116
const filter = computed(() => ({
  query: state.params.query,
  excludeDefault: true,
  labels: selectedLabels.value,
  state: state.selectedState === "ALL" ? undefined : state.selectedState, // Add this
}));
```

**Step 4: Create state options computed**

Add state filter options with permission check:

```typescript
// Add after scopeOptions computed (around line 105)
const stateFilterOptions = computed(() => {
  const options = [
    { label: t("common.active"), value: State.ACTIVE },
    { label: t("common.all"), value: "ALL" as const },
  ];

  if (hasWorkspacePermissionV2("bb.projects.undelete")) {
    // Insert archived option before "All"
    options.splice(1, 0, {
      label: t("common.archived"),
      value: State.DELETED,
    });
  }

  return options;
});
```

**Step 5: Add state filter UI to template**

Update the template to add the state filter dropdown:

```vue
<!-- Around line 3-9, modify the first div -->
<div class="flex items-center justify-between px-4 gap-x-2">
  <div class="flex items-center gap-x-2">
    <NSelect
      v-model:value="state.selectedState"
      :options="stateFilterOptions"
      :consistent-menu-width="false"
      class="!w-32"
    />
    <AdvancedSearch
      v-model:params="state.params"
      :scope-options="scopeOptions"
      :autofocus="false"
      :placeholder="$t('project.filter-projects')"
      class="flex-1"
    />
  </div>
  <PermissionGuardWrapper
    v-slot="slotProps"
    :permissions="['bb.projects.create']"
  >
    <NButton
      type="primary"
      :disabled="slotProps.disabled"
      @click="state.showCreateDrawer = true"
    >
      <template #icon>
        <PlusIcon class="h-4 w-4" />
      </template>
      {{ $t("quick-action.new-project") }}
    </NButton>
  </PermissionGuardWrapper>
</div>
```

**Step 6: Import useI18n for translations**

Add to imports:

```typescript
// Around line 62
import { useI18n } from "vue-i18n";

// In script setup, after router (around line 99)
const { t } = useI18n();
```

**Step 7: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 8: Commit**

```bash
git add frontend/src/views/ProjectDashboard.vue
git commit -m "feat: add state filter to Projects page

Add dropdown to filter projects by Active/Archived/All states.
Defaults to showing only Active projects.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Add State Filter to Instances Page

**Files:**
- Modify: `frontend/src/views/InstanceDashboard.vue:1-253`

**Step 1: Add state selection to local state**

```typescript
// In LocalState interface (around line 113)
interface LocalState {
  params: SearchParams;
  showCreateDrawer: boolean;
  showFeatureModal: boolean;
  selectedInstance: Set<string>;
  selectedState: State | "ALL"; // Add this
}

// In reactive state initialization (around line 132)
const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
  showFeatureModal: false,
  selectedInstance: new Set(),
  selectedState: State.ACTIVE, // Add this - default to ACTIVE
});
```

**Step 2: Import State and update NButton import**

Update imports:

```typescript
// Add to existing imports around line 76-77
import { State } from "@/types/proto-es/v1/common_pb";
import { NButton, NSelect } from "naive-ui"; // Add NSelect to existing import (line 77)
```

**Step 3: Add state filter to computed filter**

Update the `filter` computed property:

```typescript
// Around line 176-183
const filter = computed(() => ({
  environment: selectedEnvironment.value,
  host: selectedHost.value,
  port: selectedPort.value,
  query: state.params.query,
  engines: selectedEngines.value,
  labels: selectedLabels.value,
  state: state.selectedState === "ALL" ? undefined : state.selectedState, // Add this
}));
```

**Step 4: Create state options computed**

Add state filter options:

```typescript
// Add after scopeOptions computed (around line 211)
const stateFilterOptions = computed(() => {
  const options = [
    { label: t("common.active"), value: State.ACTIVE },
    { label: t("common.all"), value: "ALL" as const },
  ];

  if (hasWorkspacePermissionV2("bb.instances.undelete")) {
    // Insert archived option before "All"
    options.splice(1, 0, {
      label: t("common.archived"),
      value: State.DELETED,
    });
  }

  return options;
});
```

**Step 5: Import hasWorkspacePermissionV2**

```typescript
// Add to imports around line 110-111
import { hasWorkspacePermissionV2 } from "@/utils";
```

**Step 6: Add state filter UI to template**

Update the template:

```vue
<!-- Around line 9-16, modify the div -->
<div class="px-4 flex items-center gap-x-2">
  <NSelect
    v-model:value="state.selectedState"
    :options="stateFilterOptions"
    :consistent-menu-width="false"
    class="!w-32"
  />
  <AdvancedSearch
    v-model:params="state.params"
    :autofocus="false"
    :placeholder="$t('instance.filter-instance-name')"
    :scope-options="scopeOptions"
    class="flex-1"
  />
  <PermissionGuardWrapper
    v-slot="slotProps"
    :permissions="['bb.instances.create']"
  >
    <NButton
      type="primary"
      :disabled="slotProps.disabled"
      @click="showCreateInstanceDrawer"
    >
      <template #icon>
        <PlusIcon class="h-4 w-4" />
      </template>
      {{ $t("quick-action.add-instance") }}
    </NButton>
  </PermissionGuardWrapper>
</div>
```

**Step 7: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 8: Commit**

```bash
git add frontend/src/views/InstanceDashboard.vue
git commit -m "feat: add state filter to Instances page

Add dropdown to filter instances by Active/Archived/All states.
Defaults to showing only Active instances.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Update Project Hard-Delete Redirect

**Files:**
- Modify: `frontend/src/components/Project/ProjectArchiveRestoreButton.vue:76-82`

**Step 1: Update hardDeleteProject redirect**

Replace the hardDeleteProject function to redirect to projects list with archived filter:

```typescript
// Replace lines 76-82
const hardDeleteProject = async (resource: string) => {
  await projectV1Store.deleteProject(resource);
  router.replace({
    name: PROJECT_V1_ROUTE_DASHBOARD,
    query: { state: "archived" },
  });
};
```

**Step 2: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add frontend/src/components/Project/ProjectArchiveRestoreButton.vue
git commit -m "refactor: redirect to projects list after hard delete

After hard-deleting an archived project, redirect to the Projects
list page instead of the removed archive settings page.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Update Instance Hard-Delete Redirect

**Files:**
- Modify: `frontend/src/components/Instance/InstanceArchiveRestoreButton.vue:112-118`

**Step 1: Update hardDeleteInstance redirect**

Replace the hardDeleteInstance function:

```typescript
// Replace lines 112-118
const hardDeleteInstance = async (resource: string) => {
  await instanceStore.deleteInstance(resource);
  router.replace({
    name: INSTANCE_ROUTE_DASHBOARD,
    query: { state: "archived" },
  });
};
```

**Step 2: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add frontend/src/components/Instance/InstanceArchiveRestoreButton.vue
git commit -m "refactor: redirect to instances list after hard delete

After hard-deleting an archived instance, redirect to the Instances
list page instead of the removed archive settings page.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Remove Archive Route

**Files:**
- Modify: `frontend/src/router/dashboard/workspaceSetting.ts:10,60-72`

**Step 1: Remove SETTING_ROUTE_WORKSPACE_ARCHIVE export**

Delete line 10:

```typescript
// DELETE this line:
export const SETTING_ROUTE_WORKSPACE_ARCHIVE = `${SETTING_ROUTE_WORKSPACE}.archive`;
```

**Step 2: Remove archive route from children array**

Delete lines 60-72 (the entire archive route object):

```typescript
// DELETE this entire route object (lines 60-72):
      {
        path: "archive",
        name: SETTING_ROUTE_WORKSPACE_ARCHIVE,
        meta: {
          title: () => t("common.archived"),
          requiredPermissionList: () => [
            "bb.projects.list",
            "bb.instances.list",
          ],
        },
        component: () => import("@/views/Archive.vue"),
        props: true,
      },
```

**Step 3: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors (might show errors about unused import - that's okay, we'll remove the sidebar next)

**Step 4: Commit**

```bash
git add frontend/src/router/dashboard/workspaceSetting.ts
git commit -m "refactor: remove archived settings route

Remove the /setting/archive route as archived items are now
accessible via state filters on Projects and Instances pages.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Remove Archived Sidebar Item

**Files:**
- Modify: `frontend/src/utils/useDashboardSidebar.ts:39,228-232`

**Step 1: Remove import**

Delete line 39:

```typescript
// DELETE this from the import statement:
  SETTING_ROUTE_WORKSPACE_ARCHIVE,
```

**Step 2: Remove sidebar menu item**

Delete lines 228-232:

```typescript
// DELETE this entire menu item (lines 228-232):
          {
            title: t("common.archived"),
            name: SETTING_ROUTE_WORKSPACE_ARCHIVE,
            type: "route",
          },
```

**Step 3: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 4: Commit**

```bash
git add frontend/src/utils/useDashboardSidebar.ts
git commit -m "refactor: remove archived menu item from settings

Remove the Archived menu item from the Settings sidebar as
archived items are now filtered on their respective list pages.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 7: Delete Archive Page

**Files:**
- Delete: `frontend/src/views/Archive.vue`

**Step 1: Delete the file**

```bash
rm frontend/src/views/Archive.vue
```

**Step 2: Verify no other references exist**

Run: `grep -r "Archive.vue" frontend/src --exclude-dir=node_modules`
Expected: No results (file should not be referenced anywhere)

**Step 3: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 4: Commit**

```bash
git add frontend/src/views/Archive.vue
git commit -m "refactor: delete Archive.vue page

Remove the dedicated archive page component as its functionality
is now integrated into the Projects and Instances list pages.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 8: Handle Query Parameter State Sync

**Files:**
- Modify: `frontend/src/views/ProjectDashboard.vue`
- Modify: `frontend/src/views/InstanceDashboard.vue`

**Step 1: Add router query sync for ProjectDashboard**

Add watchers and initialization to sync state with URL query:

```typescript
// Add after handleCreated function (around line 159)
// Initialize state from query parameter
onMounted(() => {
  const queryState = router.currentRoute.value.query.state as string;
  if (queryState === "archived") {
    state.selectedState = State.DELETED;
  } else if (queryState === "all") {
    state.selectedState = "ALL";
  }

  const uiStateStore = useUIStateStore();
  if (!uiStateStore.getIntroStateByKey("project.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "project.visit",
      newState: true,
    });
  }
});

// Sync state changes to URL query
watch(
  () => state.selectedState,
  (newState) => {
    const query: Record<string, string> = {};
    if (newState === State.DELETED) {
      query.state = "archived";
    } else if (newState === "ALL") {
      query.state = "all";
    }
    // Update URL without creating a new history entry
    router.replace({ query });
  }
);
```

**Step 2: Import watch**

Update imports:

```typescript
// Update line 62
import { computed, onMounted, reactive, ref, watch } from "vue";
```

**Step 3: Add router query sync for InstanceDashboard**

Add similar logic to InstanceDashboard.vue:

```typescript
// Update onMounted function (around line 213-220) to include state initialization
onMounted(() => {
  const queryState = router.currentRoute.value.query.state as string;
  if (queryState === "archived") {
    state.selectedState = State.DELETED;
  } else if (queryState === "all") {
    state.selectedState = "ALL";
  }

  if (!uiStateStore.getIntroStateByKey("instance.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "instance.visit",
      newState: true,
    });
  }
});

// Add after the instanceCountAttention computed (around line 245)
watch(
  () => state.selectedState,
  (newState) => {
    const query: Record<string, string> = {};
    if (newState === State.DELETED) {
      query.state = "archived";
    } else if (newState === "ALL") {
      query.state = "all";
    }
    router.replace({ query });
  }
);
```

**Step 4: Import watch in InstanceDashboard**

Update imports:

```typescript
// Update line 78
import { computed, onMounted, reactive, ref, watch } from "vue";
```

**Step 5: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 6: Commit**

```bash
git add frontend/src/views/ProjectDashboard.vue frontend/src/views/InstanceDashboard.vue
git commit -m "feat: sync state filter with URL query parameter

Sync the state filter selection with URL query parameter for
shareable URLs and proper navigation from hard-delete redirects.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 9: Run Frontend Checks

**Files:**
- N/A (verification only)

**Step 1: Run fix**

Run: `pnpm --dir frontend fix`
Expected: Auto-fixes any ESLint/Biome issues

**Step 2: Run check**

Run: `pnpm --dir frontend check`
Expected: No errors

**Step 3: Run type-check**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 4: Commit any auto-fixes**

```bash
git add -A
git commit -m "chore: apply frontend auto-fixes

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Testing Checklist

After implementation, manually verify:

- [ ] Projects page shows state filter dropdown with Active/Archived/All options
- [ ] Instances page shows state filter dropdown with Active/Archived/All options
- [ ] State filter defaults to "Active" on both pages
- [ ] Archived option only appears if user has undelete permission
- [ ] Selecting "Archived" shows deleted projects/instances
- [ ] Selecting "All" shows both active and deleted items
- [ ] Hard-deleting an archived project redirects to Projects page with archived filter active
- [ ] Hard-deleting an archived instance redirects to Instances page with archived filter active
- [ ] URL query parameter reflects selected state (state=archived, state=all)
- [ ] Navigating with URL query parameter correctly sets filter state
- [ ] Archive settings page is removed from sidebar
- [ ] No broken links or routes related to archive page

## Rollback Plan

If issues arise, revert commits in reverse order:

```bash
git revert HEAD~9..HEAD  # Reverts all 9+ commits
```
