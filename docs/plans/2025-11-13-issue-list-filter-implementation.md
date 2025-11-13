# Issue List Filter Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign issue list filter UI to GitHub-style with always-visible search bar and quick action buttons.

**Architecture:** Create new PresetButtons and FilterToggles components, modify IssueSearch to remove toggle logic, keep AdvancedSearch mostly unchanged. Use existing SearchParams as single source of truth.

**Tech Stack:** Vue 3, TypeScript, Naive UI, existing AdvancedSearch component

**Design Document:** `/Users/p0ny/bytebase/docs/plans/2025-11-13-issue-list-filter-redesign.md`

---

## Task 1: Create PresetButtons Component

**Files:**
- Create: `frontend/src/components/IssueV1/components/IssueSearch/PresetButtons.vue`
- Reference: `frontend/src/components/IssueV1/components/IssueSearch/IssueSearch.vue` (understand SearchParams)
- Reference: `frontend/src/utils/issue/index.ts` (for upsertScope)

**Step 1: Create PresetButtons component file**

Create `frontend/src/components/IssueV1/components/IssueSearch/PresetButtons.vue`:

```vue
<template>
  <div class="flex items-center gap-x-2">
    <NButtonGroup>
      <NButton
        v-for="preset in presets"
        :key="preset.value"
        :type="isActive(preset.value) ? 'primary' : 'default'"
        size="medium"
        @click="selectPreset(preset.value)"
      >
        {{ preset.label }}
      </NButton>
    </NButtonGroup>
  </div>
</template>

<script lang="ts" setup>
import { NButton, NButtonGroup } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useCurrentUserV1 } from "@/store";
import type { SearchParams } from "@/utils";
import { getValueFromSearchParams, getSemanticIssueStatusFromSearchParams, upsertScope } from "@/utils";

type PresetValue = "WAITING_APPROVAL" | "CREATED" | "ALL";

interface Preset {
  value: PresetValue;
  label: string;
}

const props = defineProps<{
  params: SearchParams;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();
const me = useCurrentUserV1();

const presets = computed((): Preset[] => [
  {
    value: "WAITING_APPROVAL",
    label: t("issue.waiting-approval"),
  },
  {
    value: "CREATED",
    label: t("common.created"),
  },
  {
    value: "ALL",
    label: t("common.all"),
  },
]);

const isActive = (preset: PresetValue): boolean => {
  const myEmail = me.value.email;

  if (preset === "WAITING_APPROVAL") {
    return (
      getSemanticIssueStatusFromSearchParams(props.params) === "OPEN" &&
      getValueFromSearchParams(props.params, "approval") === "pending" &&
      getValueFromSearchParams(props.params, "approver") === myEmail &&
      props.params.scopes.filter((s) => !s.readonly).length === 3
    );
  }

  if (preset === "CREATED") {
    return (
      getValueFromSearchParams(props.params, "creator") === myEmail &&
      props.params.scopes.filter((s) => !s.readonly).length === 1
    );
  }

  if (preset === "ALL") {
    return props.params.scopes.filter((s) => !s.readonly).length === 0;
  }

  return false;
};

const selectPreset = (preset: PresetValue) => {
  const myEmail = me.value.email;
  const readonlyScopes = props.params.scopes.filter((s) => s.readonly);

  let newParams: SearchParams = {
    query: "",
    scopes: [...readonlyScopes],
  };

  if (preset === "WAITING_APPROVAL") {
    newParams = upsertScope({
      params: newParams,
      scopes: [
        { id: "status", value: "OPEN" },
        { id: "approval", value: "pending" },
        { id: "approver", value: myEmail },
      ],
    });
  } else if (preset === "CREATED") {
    newParams = upsertScope({
      params: newParams,
      scopes: { id: "creator", value: myEmail },
    });
  }
  // "ALL" preset keeps only readonly scopes (already done above)

  emit("update:params", newParams);
};
</script>
```

**Step 2: Verify component compiles**

Run: `pnpm --dir frontend type-check`
Expected: No errors related to PresetButtons.vue

**Step 3: Commit**

```bash
git add frontend/src/components/IssueV1/components/IssueSearch/PresetButtons.vue
git commit -m "feat: add PresetButtons component for issue filter"
```

---

## Task 2: Create FilterDropdown Component

**Files:**
- Create: `frontend/src/components/IssueV1/components/IssueSearch/FilterDropdown.vue`
- Reference: `frontend/src/components/AdvancedSearch/types.ts` (for ScopeOption)

**Step 1: Create FilterDropdown component**

Create `frontend/src/components/IssueV1/components/IssueSearch/FilterDropdown.vue`:

```vue
<template>
  <NDropdown
    :options="dropdownOptions"
    :show="showDropdown"
    placement="bottom-start"
    @select="handleSelect"
    @clickoutside="showDropdown = false"
  >
    <NButton
      :type="isActive ? 'primary' : 'default'"
      size="medium"
      @click="showDropdown = !showDropdown"
    >
      {{ buttonLabel }}
      <template #icon>
        <ChevronDownIcon class="w-4 h-4" />
      </template>
    </NButton>
  </NDropdown>
</template>

<script lang="ts" setup>
import { ChevronDownIcon } from "lucide-vue-next";
import { NButton, NDropdown } from "naive-ui";
import { computed, ref } from "vue";
import type { SearchParams } from "@/utils";
import { getValueFromSearchParams } from "@/utils";

const props = defineProps<{
  scopeId: string;
  label: string;
  options: Array<{ value: string; label: string }>;
  params: SearchParams;
  multiple?: boolean;
}>();

const emit = defineEmits<{
  (event: "select", value: string): void;
}>();

const showDropdown = ref(false);

const currentValue = computed(() => {
  return getValueFromSearchParams(props.params, props.scopeId);
});

const isActive = computed(() => {
  return !!currentValue.value;
});

const buttonLabel = computed(() => {
  if (currentValue.value) {
    const option = props.options.find((o) => o.value === currentValue.value);
    return `${props.label}: ${option?.label || currentValue.value}`;
  }
  return `${props.label} ▾`;
});

const dropdownOptions = computed(() => {
  return props.options.map((option) => ({
    key: option.value,
    label: option.label,
    type: props.multiple ? "checkbox" : "radio",
    checked: option.value === currentValue.value,
  }));
});

const handleSelect = (key: string) => {
  emit("select", key);
  if (!props.multiple) {
    showDropdown.value = false;
  }
};
</script>
```

**Step 2: Verify component compiles**

Run: `pnpm --dir frontend type-check`
Expected: No errors related to FilterDropdown.vue

**Step 3: Commit**

```bash
git add frontend/src/components/IssueV1/components/IssueSearch/FilterDropdown.vue
git commit -m "feat: add FilterDropdown component for toggleable filters"
```

---

## Task 3: Create StatusDropdown Component

**Files:**
- Create: `frontend/src/components/IssueV1/components/IssueSearch/StatusDropdown.vue`
- Reference: `frontend/src/components/IssueV1/components/IssueSearch/Status.vue` (existing implementation)

**Step 1: Create StatusDropdown component with mutex checkboxes**

Create `frontend/src/components/IssueV1/components/IssueSearch/StatusDropdown.vue`:

```vue
<template>
  <NDropdown
    :options="dropdownOptions"
    :show="showDropdown"
    placement="bottom-start"
    @select="handleSelect"
    @clickoutside="showDropdown = false"
  >
    <NButton
      :type="isActive ? 'primary' : 'default'"
      size="medium"
      @click="showDropdown = !showDropdown"
    >
      {{ buttonLabel }}
      <template #icon>
        <ChevronDownIcon class="w-4 h-4" />
      </template>
    </NButton>
  </NDropdown>
</template>

<script lang="ts" setup>
import { ChevronDownIcon } from "lucide-vue-next";
import { NButton, NDropdown } from "naive-ui";
import { computed, ref } from "react";
import { useI18n } from "vue-i18n";
import type { SearchParams, SemanticIssueStatus } from "@/utils";
import { getSemanticIssueStatusFromSearchParams, upsertScope } from "@/utils";

const props = defineProps<{
  params: SearchParams;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();
const showDropdown = ref(false);

const currentStatus = computed(() => {
  return getSemanticIssueStatusFromSearchParams(props.params);
});

const isActive = computed(() => {
  return currentStatus.value === "OPEN" || currentStatus.value === "CLOSED";
});

const buttonLabel = computed(() => {
  if (currentStatus.value === "OPEN") {
    return `${t("common.status")}: ${t("issue.table.open")}`;
  }
  if (currentStatus.value === "CLOSED") {
    return `${t("common.status")}: ${t("issue.table.closed")}`;
  }
  return `${t("common.status")} ▾`;
});

const dropdownOptions = computed(() => {
  return [
    {
      key: "OPEN",
      label: t("issue.table.open"),
      type: "checkbox",
      checked: currentStatus.value === "OPEN",
    },
    {
      key: "CLOSED",
      label: t("issue.table.closed"),
      type: "checkbox",
      checked: currentStatus.value === "CLOSED",
    },
  ];
});

const handleSelect = (key: string) => {
  const newStatus = key as SemanticIssueStatus;

  // Mutex behavior: if clicking the currently selected status, deselect it
  if (currentStatus.value === newStatus) {
    // Remove status scope to show "all"
    const updated = {
      ...props.params,
      scopes: props.params.scopes.filter((s) => s.id !== "status"),
    };
    emit("update:params", updated);
  } else {
    // Select the new status
    const updated = upsertScope({
      params: props.params,
      scopes: { id: "status", value: newStatus },
    });
    emit("update:params", updated);
  }

  showDropdown.value = false;
};
</script>
```

**Step 2: Fix import error (vue not react)**

Edit the import line:
```typescript
import { computed, ref } from "vue";
```

**Step 3: Verify component compiles**

Run: `pnpm --dir frontend type-check`
Expected: No errors related to StatusDropdown.vue

**Step 4: Commit**

```bash
git add frontend/src/components/IssueV1/components/IssueSearch/StatusDropdown.vue
git commit -m "feat: add StatusDropdown with mutex checkbox behavior"
```

---

## Task 4: Create FilterToggles Component

**Files:**
- Create: `frontend/src/components/IssueV1/components/IssueSearch/FilterToggles.vue`
- Reference: `frontend/src/components/IssueV1/components/IssueSearch/useIssueSearchScopeOptions.ts`

**Step 1: Create FilterToggles component**

Create `frontend/src/components/IssueV1/components/IssueSearch/FilterToggles.vue`:

```vue
<template>
  <div class="flex items-center gap-x-2">
    <StatusDropdown :params="params" @update:params="$emit('update:params', $event)" />
    <span class="text-control-border">|</span>
    <FilterDropdown
      v-for="filter in filters"
      :key="filter.scopeId"
      :scope-id="filter.scopeId"
      :label="filter.label"
      :options="filter.options"
      :params="params"
      :multiple="filter.multiple"
      @select="handleFilterSelect(filter.scopeId, $event)"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useCurrentUserV1 } from "@/store";
import type { SearchParams } from "@/utils";
import { upsertScope } from "@/utils";
import FilterDropdown from "./FilterDropdown.vue";
import StatusDropdown from "./StatusDropdown.vue";

const props = defineProps<{
  params: SearchParams;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();
const me = useCurrentUserV1();

const filters = computed(() => [
  {
    scopeId: "creator",
    label: t("common.creator"),
    options: [
      { value: me.value.email, label: t("common.me") },
    ],
    multiple: false,
  },
  {
    scopeId: "assignee",
    label: t("common.assignee"),
    options: [
      { value: me.value.email, label: t("common.me") },
    ],
    multiple: false,
  },
]);

const handleFilterSelect = (scopeId: string, value: string) => {
  const currentValue = props.params.scopes.find((s) => s.id === scopeId)?.value;

  // Toggle behavior: if clicking current value, remove it
  if (currentValue === value) {
    const updated = {
      ...props.params,
      scopes: props.params.scopes.filter((s) => s.id !== scopeId),
    };
    emit("update:params", updated);
  } else {
    const updated = upsertScope({
      params: props.params,
      scopes: { id: scopeId, value },
    });
    emit("update:params", updated);
  }
};
</script>
```

**Step 2: Verify component compiles**

Run: `pnpm --dir frontend type-check`
Expected: No errors related to FilterToggles.vue

**Step 3: Commit**

```bash
git add frontend/src/components/IssueV1/components/IssueSearch/FilterToggles.vue
git commit -m "feat: add FilterToggles component with Status and basic filters"
```

---

## Task 5: Modify IssueSearch Component

**Files:**
- Modify: `frontend/src/components/IssueV1/components/IssueSearch/IssueSearch.vue`
- Reference: Design document section "Component Structure"

**Step 1: Read current IssueSearch.vue**

Run: `cat frontend/src/components/IssueV1/components/IssueSearch/IssueSearch.vue`
Note the current structure with conditional rendering based on `state.advanced`

**Step 2: Replace IssueSearch.vue with new always-visible design**

Edit `frontend/src/components/IssueV1/components/IssueSearch/IssueSearch.vue`:

```vue
<template>
  <div class="flex flex-col gap-y-2">
    <!-- Advanced Search Bar - Always Visible -->
    <div class="flex flex-row items-center gap-x-2">
      <AdvancedSearch
        class="flex-1"
        :params="params"
        :scope-options="scopeOptions"
        @update:params="$emit('update:params', $event)"
      />
      <TimeRange
        v-if="components.includes('time-range')"
        v-model:show="showTimeRange"
        :params="params"
        v-bind="componentProps?.['time-range']"
        @update:params="$emit('update:params', $event)"
      />
      <slot name="searchbox-suffix" />
    </div>

    <slot name="default" />

    <!-- Preset Buttons Row -->
    <div v-if="!componentProps?.presets?.hidden" class="flex flex-col gap-y-2">
      <PresetButtons
        v-if="components.includes('presets')"
        :params="params"
        @update:params="$emit('update:params', $event)"
      />

      <!-- Filter Toggles Row -->
      <FilterToggles
        v-if="components.includes('filters')"
        :params="params"
        @update:params="$emit('update:params', $event)"
      />
    </div>

    <slot name="primary" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import AdvancedSearch from "@/components/AdvancedSearch";
import TimeRange from "@/components/AdvancedSearch/TimeRange.vue";
import type { SearchParams, SearchScopeId } from "@/utils";
import { UIIssueFilterScopeIdList } from "@/utils";
import FilterToggles from "./FilterToggles.vue";
import PresetButtons from "./PresetButtons.vue";
import { useIssueSearchScopeOptions } from "./useIssueSearchScopeOptions";

export type SearchComponent = "searchbox" | "presets" | "filters" | "time-range";

const props = withDefaults(
  defineProps<{
    params: SearchParams;
    overrideScopeIdList?: SearchScopeId[];
    autofocus?: boolean;
    components?: SearchComponent[];
    componentProps?: Partial<Record<SearchComponent, any>>;
  }>(),
  {
    overrideScopeIdList: () => [],
    components: () => ["searchbox", "time-range", "presets", "filters"],
    componentProps: undefined,
  }
);

defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const showTimeRange = ref(false);

const allowedScopes = computed((): SearchScopeId[] => {
  if (props.overrideScopeIdList && props.overrideScopeIdList.length > 0) {
    return props.overrideScopeIdList;
  }
  return [
    ...UIIssueFilterScopeIdList,
    "creator",
    "instance",
    "database",
    "status",
    "taskType",
    "issue-label",
    "project",
    "environment",
  ];
});

const scopeOptions = useIssueSearchScopeOptions(
  computed(() => props.params),
  allowedScopes
);
</script>
```

**Step 3: Remove Status.vue references**

The old `Status.vue` component is no longer imported. It's replaced by `StatusDropdown` inside `FilterToggles`.

**Step 4: Verify component compiles**

Run: `pnpm --dir frontend type-check`
Expected: No errors related to IssueSearch.vue

**Step 5: Commit**

```bash
git add frontend/src/components/IssueV1/components/IssueSearch/IssueSearch.vue
git commit -m "refactor: remove toggle mode, always show search with presets and filters"
```

---

## Task 6: Update MyIssues.vue

**Files:**
- Modify: `frontend/src/views/MyIssues.vue`

**Step 1: Remove toggle logic from MyIssues.vue**

Edit `frontend/src/views/MyIssues.vue`:

Remove these parts:
- `state.advanced` from LocalState interface
- `toggleAdvancedSearch()` function
- Conditional component rendering based on `state.advanced`
- Search icon toggle button in template
- ChevronDownIcon hide button in searchbox-suffix slot

Replace the IssueSearch section with:

```vue
<IssueSearch
  v-model:params="state.params"
  :components="['searchbox', 'time-range', 'presets', 'filters']"
  class="px-4 pb-2"
/>
```

Remove imports:
```typescript
import { ChevronDownIcon, SearchIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
```

Remove from LocalState:
```typescript
interface LocalState {
  params: SearchParams;
  // Remove: advanced: boolean;
}
```

Update initialization:
```typescript
const state = reactive<LocalState>({
  params: mergeSearchParamsByTab(defaultSearchParams(), storedTab.value),
  // Remove: advanced: false,
});
```

**Step 2: Remove tab-related code**

Remove these:
- `TabFilter` component import and usage
- `tabItemList` computed
- `tab` computed
- `selectTab` function
- `storedTab` localStorage
- All tab-related watchers

Since presets are now handled by PresetButtons component.

**Step 3: Simplify the component**

The new simplified structure:

```vue
<template>
  <div :key="viewId" class="flex flex-col">
    <IssueSearch
      v-model:params="state.params"
      :components="['searchbox', 'time-range', 'presets', 'filters']"
      class="px-4 pb-2"
    />

    <div class="relative min-h-80">
      <PagedTable
        ref="issuePagedTable"
        session-key="bb.issue-table.my-issues"
        :fetch-list="fetchIssueList"
      >
        <template #table="{ list, loading }">
          <IssueTableV1
            class="border-x-0"
            :loading="loading"
            :issue-list="applyUIIssueFilter(list, mergedUIIssueFilter)"
            :highlight-text="state.params.query"
            :show-project="true"
          />
        </template>
      </PagedTable>
    </div>
  </div>
</template>
```

**Step 4: Verify component compiles**

Run: `pnpm --dir frontend type-check`
Expected: No errors related to MyIssues.vue

**Step 5: Commit**

```bash
git add frontend/src/views/MyIssues.vue
git commit -m "refactor: simplify MyIssues to use new preset/filter components"
```

---

## Task 7: Update ProjectIssuesPanel.vue

**Files:**
- Modify: `frontend/src/components/Project/ProjectIssuesPanel.vue`

**Step 1: Apply same changes as MyIssues.vue**

Edit `frontend/src/components/Project/ProjectIssuesPanel.vue`:

Replace IssueSearch usage:
```vue
<IssueSearch
  v-model:params="state.params"
  :components="['searchbox', 'time-range', 'presets', 'filters']"
/>
```

Remove:
- `state.advanced` from LocalState
- `toggleAdvancedSearch()` function
- Tab-related code (TabFilter, tab computed, selectTab, etc.)
- Toggle button UI elements
- ChevronDownIcon, SearchIcon, NButton, NTooltip imports

**Step 2: Verify component compiles**

Run: `pnpm --dir frontend type-check`
Expected: No errors related to ProjectIssuesPanel.vue

**Step 3: Commit**

```bash
git add frontend/src/components/Project/ProjectIssuesPanel.vue
git commit -m "refactor: simplify ProjectIssuesPanel to use new preset/filter components"
```

---

## Task 8: Update Component Exports

**Files:**
- Modify: `frontend/src/components/IssueV1/components/IssueSearch/index.ts`

**Step 1: Add new component exports**

Edit `frontend/src/components/IssueV1/components/IssueSearch/index.ts`:

```typescript
export { default as IssueSearch } from "./IssueSearch.vue";
export { default as PresetButtons } from "./PresetButtons.vue";
export { default as FilterToggles } from "./FilterToggles.vue";
export { default as FilterDropdown } from "./FilterDropdown.vue";
export { default as StatusDropdown } from "./StatusDropdown.vue";
```

**Step 2: Check if index.ts exists, if not create it**

Run: `test -f frontend/src/components/IssueV1/components/IssueSearch/index.ts && echo "exists" || echo "not found"`

If "not found", create the file with the exports above.

**Step 3: Verify exports compile**

Run: `pnpm --dir frontend type-check`
Expected: No errors

**Step 4: Commit**

```bash
git add frontend/src/components/IssueV1/components/IssueSearch/index.ts
git commit -m "feat: export new preset and filter components"
```

---

## Task 9: Run Linter and Fix Issues

**Files:**
- All modified/created Vue files

**Step 1: Run frontend linter**

Run: `pnpm --dir frontend lint --fix`
Expected: Auto-fixes applied, or errors listed

**Step 2: Fix any remaining lint errors manually**

Check output from step 1. Common issues:
- Unused imports (remove them)
- Missing types (add them)
- Formatting issues (run prettier)

**Step 3: Run prettier**

Run: `pnpm --dir frontend prettier --write "src/components/IssueV1/components/IssueSearch/**/*.vue" "src/views/MyIssues.vue" "src/components/Project/ProjectIssuesPanel.vue"`

**Step 4: Verify no lint errors**

Run: `pnpm --dir frontend lint`
Expected: No errors

**Step 5: Commit fixes if any**

```bash
git add -u
git commit -m "style: fix linting and formatting issues"
```

---

## Task 10: Manual Testing

**Files:**
- N/A (manual testing)

**Step 1: Start dev server**

Run: `pnpm --dir frontend dev`
Expected: Dev server starts on http://localhost:3000 or similar

**Step 2: Test MyIssues page**

Navigate to My Issues page in browser.

Verify:
- AdvancedSearch bar is always visible
- Preset buttons (Waiting Approval, Created, All) are shown
- Filter dropdowns (Status, Creator, Assignee) are shown
- Clicking presets changes the filter tags in search bar
- Clicking filter dropdowns adds/removes filters
- Status dropdown has checkbox behavior (mutex, can deselect)

**Step 3: Test ProjectIssuesPanel**

Navigate to a project's issues tab.

Verify same functionality as MyIssues.

**Step 4: Test filter interactions**

Test these scenarios:
1. Click "Waiting Approval" preset → verify tags: `[status:OPEN] [approval:pending] [approver:me]`
2. Click "Created" preset → verify tags: `[creator:me]`
3. Click "All" preset → verify tags cleared
4. While on "Created", add Status filter → verify preset becomes inactive
5. Use Status dropdown → check Open → verify `[status:OPEN]` tag
6. Use Status dropdown → uncheck Open → verify status tag removed

**Step 5: Document any issues found**

If bugs found, create a list for follow-up tasks.

---

## Task 11: Create Follow-up Tasks (If Needed)

**Files:**
- N/A

**Step 1: Review manual testing results**

Check if any issues were found in Task 10.

**Step 2: Create GitHub issues or fix immediately**

For each issue:
- If minor CSS/layout issue → fix immediately and commit
- If behavior bug → fix immediately and commit
- If major redesign needed → document as GitHub issue

**Step 3: Final commit**

If any fixes made:
```bash
git add -u
git commit -m "fix: address issues found in manual testing"
```

---

## Verification Checklist

After completing all tasks:

- [ ] PresetButtons component created and functional
- [ ] FilterToggles component created and functional
- [ ] StatusDropdown has mutex checkbox behavior
- [ ] IssueSearch always shows search bar (no toggle)
- [ ] MyIssues.vue simplified (no tab logic)
- [ ] ProjectIssuesPanel.vue simplified (no tab logic)
- [ ] All components properly exported
- [ ] No TypeScript errors (`pnpm type-check` passes)
- [ ] No lint errors (`pnpm lint` passes)
- [ ] Manual testing shows correct behavior
- [ ] URL query parameters still work for bookmarking
- [ ] Responsive design works (test mobile viewport)

## Notes

- The Status.vue component can be kept for backward compatibility or removed if not used elsewhere
- Consider adding more filter dropdowns in future iterations (Project, Database, etc.)
- The FilterDropdown component is designed to be reusable - can be extended for search functionality like Creator/Assignee user search
- If NDropdown from Naive UI doesn't support checkbox/radio options natively, may need to create custom dropdown menu with NCheckbox/NRadio components inside NPopover
