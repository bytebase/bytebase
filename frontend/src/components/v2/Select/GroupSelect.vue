<template>
  <ResourceSelect
    v-bind="$attrs"
    :remote="true"
    :loading="state.loading"
    :value="validSelectedGroup"
    :values="validSelectedGroups"
    :disabled="disabled"
    :multiple="multiple"
    :options="options"
    :custom-label="renderLabel"
    :placeholder="$t('settings.members.select-group', multiple ? 2 : 1)"
    :hint="hint"
    :hint-key="hintKey"
    @search="handleSearch"
    @update:value="(val) => $emit('update:group', val)"
    @update:values="(val) => $emit('update:groups', val)"
  />
</template>

<script lang="tsx" setup>
import { useDebounceFn } from "@vueuse/core";
import { computed, onMounted, reactive } from "vue";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { useGroupStore } from "@/store";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { getDefaultPagination } from "@/utils";
import ResourceSelect from "./ResourceSelect.vue";

const props = withDefaults(
  defineProps<{
    group?: string | undefined;
    groups?: string[] | undefined;
    disabled?: boolean;
    multiple?: boolean;
    projectName?: string;
    selectFirstAsDefault?: boolean;
    size?: "tiny" | "small" | "medium" | "large";
    // Filter to groups with this permission on the project.
    // Example: "bb.sql.select"
    requiredPermission?: string;
    // Hint text shown at the top of the dropdown.
    hint?: string;
    // Unique key to persist hint dismissal in localStorage.
    hintKey?: string;
  }>(),
  {
    group: undefined,
    groups: undefined,
    multiple: false,
    projectName: undefined,
    selectFirstAsDefault: true,
    size: "medium",
    requiredPermission: undefined,
    hint: undefined,
    hintKey: undefined,
  }
);

defineEmits<{
  (event: "update:group", val: string | undefined): void;
  (event: "update:groups", val: string[]): void;
}>();

interface LocalState {
  loading: boolean;
  rawList: Group[];
  // Track if initial fetch has been done to avoid redundant API calls
  initialized: boolean;
}

const state = reactive<LocalState>({
  loading: false,
  rawList: [],
  initialized: false,
});

const groupStore = useGroupStore();

const searchGroups = async (search: string) => {
  const { groups } = await groupStore.fetchGroupList({
    filter: {
      query: search,
      project: props.projectName,
      requiredPermission: props.requiredPermission,
    },
    pageSize: getDefaultPagination(),
  });
  return groups;
};

const initSelectedGroups = async (groupNames: string[]) => {
  if (groupNames.length === 0) return;
  const groups = await groupStore.batchFetchGroups(groupNames);
  for (const group of groups) {
    if (!state.rawList.find((g) => g.name === group.name)) {
      state.rawList.unshift(group);
    }
  }
};

const doSearch = async (search: string) => {
  // Skip if no search term and already initialized (lazy loading optimization)
  if (!search && state.initialized) {
    return;
  }

  state.loading = true;
  try {
    const groups = await searchGroups(search);
    state.rawList = groups;
    if (!search) {
      state.initialized = true;
      if (props.group) {
        await initSelectedGroups([props.group]);
      }
      if (props.groups) {
        await initSelectedGroups(props.groups);
      }
    }
  } finally {
    state.loading = false;
  }
};

const debouncedSearch = useDebounceFn(doSearch, DEBOUNCE_SEARCH_DELAY);

// For initial load (empty search), fetch immediately without debounce.
// For search queries, use debounce to avoid excessive API calls.
const handleSearch = (search: string) => {
  if (!search && !state.initialized) {
    // Initial load: fetch immediately
    doSearch(search);
  } else if (search) {
    // Search query: use debounce
    debouncedSearch(search);
  }
  // If !search && initialized, doSearch will return early anyway
};

// Only fetch selected group(s) on mount, not the entire list.
// The full list will be fetched lazily when dropdown is opened.
onMounted(async () => {
  if (props.group) {
    await initSelectedGroups([props.group]);
  } else if (props.groups) {
    await initSelectedGroups(props.groups);
  }
});

const options = computed(() => {
  return state.rawList.map((group) => ({
    value: group.name,
    label: group.title,
    resource: group,
  }));
});

const validSelectedGroup = computed(() => {
  if (props.multiple) {
    return undefined;
  }
  if (options.value.findIndex((o) => o.value === props.group) >= 0) {
    return props.group;
  }
  return undefined;
});

const validSelectedGroups = computed(() => {
  if (!props.multiple) {
    return undefined;
  }

  return props.groups?.filter((v) => {
    return options.value.findIndex((o) => o.value === v) >= 0;
  });
});

const renderLabel = (group: Group) => {
  return (
    <GroupNameCell
      showEmail={false}
      group={group}
      showIcon={false}
      link={false}
    />
  );
};
</script>
