<template>
  <ResourceSelect
    v-bind="$attrs"
    :remote="true"
    :value="validSelectedGroup"
    :values="validSelectedGroups"
    :disabled="disabled"
    :multiple="multiple"
    :options="options"
    :custom-label="renderLabel"
    :placeholder="$t('settings.members.select-group', multiple ? 2 : 1)"
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
  }>(),
  {
    group: undefined,
    groups: undefined,
    multiple: false,
    projectName: undefined,
    selectFirstAsDefault: true,
    size: "medium",
  }
);

defineEmits<{
  (event: "update:group", val: string | undefined): void;
  (event: "update:groups", val: string[]): void;
}>();

interface LocalState {
  loading: boolean;
  rawList: Group[];
}

const state = reactive<LocalState>({
  loading: false,
  rawList: [],
});

const groupStore = useGroupStore();

const searchGroups = async (search: string) => {
  const { groups } = await groupStore.fetchGroupList({
    filter: {
      query: search,
      project: props.projectName,
    },
    pageSize: getDefaultPagination(),
  });
  return groups;
};

const handleSearch = useDebounceFn(async (search: string) => {
  state.loading = true;
  try {
    const groups = await searchGroups(search);
    state.rawList = groups;
    if (!search) {
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
}, DEBOUNCE_SEARCH_DELAY);

const initSelectedGroups = async (groupNames: string[]) => {
  const groups = await groupStore.batchFetchGroups(groupNames);
  for (const group of groups) {
    if (!state.rawList.find((g) => g.name === group.name)) {
      state.rawList.unshift(group);
    }
  }
};

onMounted(async () => {
  await handleSearch("");
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
