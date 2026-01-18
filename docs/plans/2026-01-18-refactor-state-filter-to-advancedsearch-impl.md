# Refactor State Filter to AdvancedSearch Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor state filter from separate NSelect dropdown to integrated AdvancedSearch scope option.

**Architecture:** Remove the NSelect state dropdown added in Tasks 1-8, integrate state filtering into AdvancedSearch as a scope option (like environment, engine, label), and update hard-delete redirects to use AdvancedSearch's query format.

**Tech Stack:** Vue 3, TypeScript, AdvancedSearch component, Vue Router

---

### Task 1: Add ALL Option to State Scope

**Files:**
- Modify: `frontend/src/components/AdvancedSearch/useCommonSearchScopeOptions.ts:182-199`

**Step 1: Update state scope definition**

Replace the existing state scope (lines 182-199) with the new version:

```typescript
state: () => ({
  id: "state",
  title: t("common.state"),
  description: t("plan.state.description"),
  options: [
    {
      value: "ACTIVE",
      keywords: ["active", "ACTIVE"],
      render: () => t("common.active"),
    },
    {
      value: "DELETED",
      keywords: ["archived", "ARCHIVED", "deleted", "DELETED"],
      render: () => t("common.archived"),
    },
    {
      value: "ALL",
      keywords: ["all", "ALL"],
      render: () => t("common.all"),
    },
  ],
  allowMultiple: false,
}),
```

**Step 2: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add frontend/src/components/AdvancedSearch/useCommonSearchScopeOptions.ts
git commit -m "feat: add ALL option to state scope in AdvancedSearch

Add 'all' option to state scope and 'archived' keywords for
better user experience when filtering by state.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Refactor ProjectDashboard to Use State Scope

**Files:**
- Modify: `frontend/src/views/ProjectDashboard.vue`

**Step 1: Remove NSelect import**

Update the NButton import to remove NSelect:

```typescript
// Change line 70 from:
import { NButton, NSelect } from "naive-ui";

// To:
import { NButton } from "naive-ui";
```

**Step 2: Remove selectedState from LocalState**

Update the LocalState interface (around line 91-96):

```typescript
// Change from:
interface LocalState {
  params: SearchParams;
  showCreateDrawer: boolean;
  selectedProjects: Set<string>;
  selectedState: State | "ALL";
}

// To:
interface LocalState {
  params: SearchParams;
  showCreateDrawer: boolean;
  selectedProjects: Set<string>;
}
```

**Step 3: Remove selectedState from reactive state**

Update the reactive state initialization (around line 102-110):

```typescript
// Change from:
const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
  selectedProjects: new Set(),
  selectedState: State.ACTIVE,
});

// To:
const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
  selectedProjects: new Set(),
});
```

**Step 4: Add state to scopeOptions**

Update scopeOptions (around line 119):

```typescript
// Change from:
const scopeOptions = useCommonSearchScopeOptions(["label"]);

// To:
const scopeOptions = useCommonSearchScopeOptions(["label", "state"]);
```

**Step 5: Add selectedState computed property**

Add after selectedLabels computed (around line 111-112):

```typescript
// Add after selectedLabels computed
const selectedState = computed(() => {
  const stateValue = getValueFromSearchParams(state.params, "state");
  if (stateValue === "DELETED") return State.DELETED;
  if (stateValue === "ALL") return undefined; // undefined = show all
  return State.ACTIVE; // default
});
```

**Step 6: Update filter computed**

Update the filter computed property (around line 147-152):

```typescript
// Keep this line:
const filter = computed(() => ({
  query: state.params.query,
  excludeDefault: true,
  labels: selectedLabels.value,
  state: selectedState.value, // Change from state.selectedState to selectedState.value
}));
```

**Step 7: Remove stateFilterOptions computed**

Delete the stateFilterOptions computed property (around lines 126-141).

**Step 8: Remove custom state watcher**

Delete the watch block that syncs selectedState to URL (added in Task 8, around lines 169-180).

**Step 9: Simplify onMounted**

Replace the onMounted function to remove state initialization:

```typescript
// Replace the entire onMounted function with:
onMounted(() => {
  const uiStateStore = useUIStateStore();
  if (!uiStateStore.getIntroStateByKey("project.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "project.visit",
      newState: true,
    });
  }
});
```

**Step 10: Update template**

Remove the NSelect dropdown from template (around lines 3-17):

```vue
<!-- Change from: -->
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
  ...
</div>

<!-- To: -->
<div class="flex items-center justify-between px-4 gap-x-2">
  <AdvancedSearch
    v-model:params="state.params"
    :scope-options="scopeOptions"
    :autofocus="false"
    :placeholder="$t('project.filter-projects')"
  />
  ...
</div>
```

**Step 11: Add getValueFromSearchParams import**

Update imports to add getValueFromSearchParams:

```typescript
// Change line 85-89 from:
import {
  getValuesFromSearchParams,
  hasWorkspacePermissionV2,
  type SearchParams,
} from "@/utils";

// To:
import {
  getValueFromSearchParams,
  getValuesFromSearchParams,
  hasWorkspacePermissionV2,
  type SearchParams,
} from "@/utils";
```

**Step 12: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 13: Commit**

```bash
git add frontend/src/views/ProjectDashboard.vue
git commit -m "refactor: integrate state filter into AdvancedSearch for Projects

Remove separate NSelect dropdown and integrate state filtering
into AdvancedSearch as a scope option (state:active/archived/all).

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Refactor InstanceDashboard to Use State Scope

**Files:**
- Modify: `frontend/src/views/InstanceDashboard.vue`

**Step 1: Remove NSelect from import**

Update the NButton import (around line 84):

```typescript
// Change from:
import { NButton, NSelect } from "naive-ui";

// To:
import { NButton } from "naive-ui";
```

**Step 2: Remove selectedState from LocalState**

Update LocalState interface (around line 121-127):

```typescript
// Change from:
interface LocalState {
  params: SearchParams;
  showCreateDrawer: boolean;
  showFeatureModal: boolean;
  selectedInstance: Set<string>;
  selectedState: State | "ALL";
}

// To:
interface LocalState {
  params: SearchParams;
  showCreateDrawer: boolean;
  showFeatureModal: boolean;
  selectedInstance: Set<string>;
}
```

**Step 3: Remove selectedState from reactive state**

Update reactive state initialization (around line 141-149):

```typescript
// Change from:
const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
  showFeatureModal: false,
  selectedInstance: new Set(),
  selectedState: State.ACTIVE,
});

// To:
const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
  showFeatureModal: false,
  selectedInstance: new Set(),
});
```

**Step 4: Add state to scopeOptions**

Update scopeOptions computed (around line 202-218):

```typescript
// In the scopeOptions computed, change the return array to include "state":
return [
  ...useCommonSearchScopeOptions(["environment", "engine", "label", "state"]).value,
  {
    id: "host",
    title: t("instance.advanced-search.scope.host.title"),
    description: t("instance.advanced-search.scope.host.description"),
    options: [],
  },
  {
    id: "port",
    title: t("instance.advanced-search.scope.port.title"),
    description: t("instance.advanced-search.scope.port.description"),
    options: [],
  },
];
```

**Step 5: Add selectedState computed property**

Add after selectedLabels computed (around line 181):

```typescript
// Add after selectedLabels computed
const selectedState = computed(() => {
  const stateValue = getValueFromSearchParams(state.params, "state");
  if (stateValue === "DELETED") return State.DELETED;
  if (stateValue === "ALL") return undefined; // undefined = show all
  return State.ACTIVE; // default
});
```

**Step 6: Update filter computed**

Update filter computed (around line 183-192):

```typescript
// Keep this structure, just change state.selectedState to selectedState.value:
const filter = computed(() => ({
  environment: selectedEnvironment.value,
  host: selectedHost.value,
  port: selectedPort.value,
  query: state.params.query,
  engines: selectedEngines.value,
  labels: selectedLabels.value,
  state: selectedState.value, // Change from state.selectedState
}));
```

**Step 7: Remove stateFilterOptions computed**

Delete the stateFilterOptions computed property (around lines 224-239).

**Step 8: Remove custom state watcher**

Delete the watch block for state sync (added in Task 8, after instanceCountAttention computed).

**Step 9: Simplify onMounted**

Replace onMounted to remove state initialization:

```typescript
// Replace entire onMounted with:
onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("instance.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "instance.visit",
      newState: true,
    });
  }
});
```

**Step 10: Update template**

Remove NSelect dropdown (around lines 9-22):

```vue
<!-- Change from: -->
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
  ...
</div>

<!-- To: -->
<div class="px-4 flex items-center gap-x-2">
  <AdvancedSearch
    v-model:params="state.params"
    :autofocus="false"
    :placeholder="$t('instance.filter-instance-name')"
    :scope-options="scopeOptions"
  />
  ...
</div>
```

**Step 11: Add getValueFromSearchParams import**

Update imports (around line 115-118):

```typescript
// Change from:
import {
  getValueFromSearchParams,
  getValuesFromSearchParams,
  type SearchParams,
} from "@/utils";

// To (add getValueFromSearchParams if not already there):
import {
  getValueFromSearchParams,
  getValuesFromSearchParams,
  type SearchParams,
} from "@/utils";
```

**Step 12: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 13: Commit**

```bash
git add frontend/src/views/InstanceDashboard.vue
git commit -m "refactor: integrate state filter into AdvancedSearch for Instances

Remove separate NSelect dropdown and integrate state filtering
into AdvancedSearch as a scope option (state:active/archived/all).

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Update Project Hard-Delete Redirect Format

**Files:**
- Modify: `frontend/src/components/Project/ProjectArchiveRestoreButton.vue:75-81`

**Step 1: Update redirect query format**

Replace the hardDeleteProject function:

```typescript
// Replace lines 75-81 with:
const hardDeleteProject = async (resource: string) => {
  await projectV1Store.deleteProject(resource);
  router.replace({
    name: PROJECT_V1_ROUTE_DASHBOARD,
    query: { q: "state:archived" },
  });
};
```

**Step 2: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add frontend/src/components/Project/ProjectArchiveRestoreButton.vue
git commit -m "refactor: update project hard-delete redirect to use AdvancedSearch format

Change redirect query from 'state=archived' to 'q=state:archived'
to match AdvancedSearch's query parameter format.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Update Instance Hard-Delete Redirect Format

**Files:**
- Modify: `frontend/src/components/Instance/InstanceArchiveRestoreButton.vue:111-117`

**Step 1: Update redirect query format**

Replace the hardDeleteInstance function:

```typescript
// Replace lines 111-117 with:
const hardDeleteInstance = async (resource: string) => {
  await instanceStore.deleteInstance(resource);
  router.replace({
    name: INSTANCE_ROUTE_DASHBOARD,
    query: { q: "state:archived" },
  });
};
```

**Step 2: Verify changes**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add frontend/src/components/Instance/InstanceArchiveRestoreButton.vue
git commit -m "refactor: update instance hard-delete redirect to use AdvancedSearch format

Change redirect query from 'state=archived' to 'q=state:archived'
to match AdvancedSearch's query parameter format.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Run Frontend Checks

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

- [ ] Projects page no longer has separate state dropdown
- [ ] Instances page no longer has separate state dropdown
- [ ] AdvancedSearch shows "state" as a scope option when clicking the filter icon
- [ ] Typing `state:active` filters to active projects/instances
- [ ] Typing `state:archived` filters to archived projects/instances
- [ ] Typing `state:all` shows all projects/instances
- [ ] Default behavior (no state scope) shows only active items
- [ ] Hard-deleting an archived project redirects to Projects with `?q=state:archived`
- [ ] Hard-deleting an archived instance redirects to Instances with `?q=state:archived`
- [ ] URL query parameter `?q=state:archived` properly filters on page load
- [ ] State filter can be combined with other filters: `state:archived label:migration`
- [ ] No TypeScript errors
- [ ] No console errors in browser

## Rollback Plan

If issues arise, revert commits in reverse order:

```bash
git revert HEAD~6..HEAD  # Reverts all 6+ commits from this refactor
```
