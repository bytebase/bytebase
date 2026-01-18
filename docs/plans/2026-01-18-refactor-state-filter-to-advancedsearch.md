# Refactor State Filter to AdvancedSearch

## Overview

Refactor the state filter from a separate dropdown to be integrated into the AdvancedSearch component as a scope option (like `state:active`, `state:archived`, `state:all`).

## Background

The current implementation (Tasks 1-8) added a separate NSelect dropdown for state filtering next to the AdvancedSearch component. This is inconsistent with how other filters work (environment, engine, label, etc.) which are all integrated into AdvancedSearch as scope options.

## Goals

- Integrate state filtering into AdvancedSearch as a scope option
- Remove the separate NSelect dropdown
- Maintain all existing functionality (filter by active/archived/all)
- Simplify the code by leveraging AdvancedSearch's existing URL sync

## Design

### 1. Update State Scope Definition

Modify `useCommonSearchScopeOptions` to add an "ALL" option to the existing state scope:

**File:** `frontend/src/components/AdvancedSearch/useCommonSearchScopeOptions.ts`

**Current state scope** (lines 182-199):
- ACTIVE
- DELETED

**New state scope**:
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

**Key change**: Add "archived" as a keyword for DELETED so users can type `state:archived` (more intuitive than `state:deleted`).

### 2. Extract State from Search Params

Instead of maintaining `selectedState` in local state, extract it from search params.

**ProjectDashboard.vue changes:**

1. Remove `selectedState` from LocalState interface
2. Remove `NSelect` dropdown from template
3. Remove `NSelect` from imports
4. Add "state" to scope options:
   ```typescript
   const scopeOptions = useCommonSearchScopeOptions(["label", "state"]);
   ```

5. Extract selected state from params:
   ```typescript
   const selectedState = computed(() => {
     const stateValue = getValueFromSearchParams(state.params, "state");
     if (stateValue === "DELETED") return State.DELETED;
     if (stateValue === "ALL") return undefined; // undefined = show all
     return State.ACTIVE; // default
   });
   ```

6. Update filter computed:
   ```typescript
   const filter = computed(() => ({
     query: state.params.query,
     excludeDefault: true,
     labels: selectedLabels.value,
     state: selectedState.value,
   }));
   ```

**InstanceDashboard.vue changes:**

Same pattern as ProjectDashboard:
1. Remove `selectedState` from LocalState
2. Remove `NSelect` dropdown from template
3. Add "state" to scope options:
   ```typescript
   const scopeOptions = useCommonSearchScopeOptions(["environment", "engine", "label", "state"]);
   ```
4. Extract and use selectedState the same way

### 3. URL Query Parameter Sync

Remove the custom state URL sync logic added in Task 8 and leverage AdvancedSearch's existing sync.

**Changes:**

1. Remove `watch(() => state.selectedState, ...)` from both dashboard files
2. Remove state initialization from `onMounted` that reads `router.currentRoute.value.query.state`
3. Keep only the UIStateStore intro logic in `onMounted`

**Hard-delete redirect changes:**

Update the redirect format to use AdvancedSearch's query format:

**ProjectArchiveRestoreButton.vue:**
```typescript
const hardDeleteProject = async (resource: string) => {
  await projectV1Store.deleteProject(resource);
  router.replace({
    name: PROJECT_V1_ROUTE_DASHBOARD,
    query: { q: "state:archived" },
  });
};
```

**InstanceArchiveRestoreButton.vue:**
```typescript
const hardDeleteInstance = async (resource: string) => {
  await instanceStore.deleteInstance(resource);
  router.replace({
    name: INSTANCE_ROUTE_DASHBOARD,
    query: { q: "state:archived" },
  });
};
```

The `q` parameter is what AdvancedSearch uses for its search query. When users land on the page with `?q=state:archived`, AdvancedSearch automatically parses it and sets the state filter.

## Implementation Files

### Files to Modify:

1. `frontend/src/components/AdvancedSearch/useCommonSearchScopeOptions.ts`
   - Add "ALL" option to state scope
   - Add "archived" keywords to DELETED option

2. `frontend/src/views/ProjectDashboard.vue`
   - Remove NSelect dropdown from template
   - Remove selectedState from LocalState
   - Add "state" to scopeOptions
   - Add selectedState computed property
   - Remove custom URL sync watcher
   - Simplify onMounted

3. `frontend/src/views/InstanceDashboard.vue`
   - Same changes as ProjectDashboard

4. `frontend/src/components/Project/ProjectArchiveRestoreButton.vue`
   - Update redirect to use `query: { q: "state:archived" }`

5. `frontend/src/components/Instance/InstanceArchiveRestoreButton.vue`
   - Update redirect to use `query: { q: "state:archived" }`

## Benefits

1. **Consistency**: State filter works like all other filters (environment, engine, label)
2. **Simpler code**: Remove custom URL sync logic, leverage AdvancedSearch's existing sync
3. **Better UX**: Users can type `state:archived` in the filter box, just like `label:critical` or `environment:prod`
4. **Fewer components**: Remove NSelect dropdown, reduce template complexity
5. **Composable**: State filter can be combined with other filters: `state:archived label:migration`

## Default Behavior

When no state filter is specified:
- Default to showing only ACTIVE items (current behavior)
- Users explicitly add `state:archived` or `state:all` to see other states

This is implemented by defaulting `selectedState` to `State.ACTIVE` when no state scope is present in params.

## URL Examples

- `/projects` → Shows active projects (default)
- `/projects?q=state:archived` → Shows archived projects
- `/projects?q=state:all` → Shows all projects
- `/projects?q=state:archived label:migration` → Archived projects with migration label
- `/instances?q=state:active environment:prod` → Active instances in prod environment
