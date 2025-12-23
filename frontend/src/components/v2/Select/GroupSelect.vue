<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :value="value"
    :disabled="disabled"
    :multiple="multiple"
    :size="size"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :additional-options="additionalOptions"
    :search="handleSearch"
    @update:value="(val) => $emit('update:value', val)"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { useGroupStore } from "@/store";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import RemoteResourceSelector from "./RemoteResourceSelector/index.vue";
import type {
  ResourceSelectOption,
  SelectSize,
} from "./RemoteResourceSelector/types";
import {
  getRenderLabelFunc,
  getRenderTagFunc,
} from "./RemoteResourceSelector/utils";

const props = defineProps<{
  value?: string[] | string | undefined;
  disabled?: boolean;
  multiple?: boolean;
  projectName?: string;
  size?: SelectSize;
}>();

defineEmits<{
  (event: "update:value", val: string[] | string | undefined): void;
}>();

const groupStore = useGroupStore();

const getOption = (group: Group): ResourceSelectOption<Group> => ({
  resource: group,
  value: group.name,
  label: group.title,
});

const additionalOptions = computedAsync(async () => {
  const options: ResourceSelectOption<Group>[] = [];

  let groupNames: string[] = [];
  if (Array.isArray(props.value)) {
    groupNames = props.value;
  } else if (props.value) {
    groupNames = [props.value];
  }

  const groups = await groupStore.batchGetOrFetchGroups(groupNames);
  for (const group of groups) {
    if (group) {
      options.push(getOption(group));
    }
  }

  return options;
}, []);

const handleSearch = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  const { groups, nextPageToken } = await groupStore.fetchGroupList({
    filter: {
      query: params.search,
      project: props.projectName,
    },
    pageToken: params.pageToken,
    pageSize: params.pageSize,
  });

  return {
    nextPageToken,
    options: groups.map(getOption),
  };
};

const customLabel = (group: Group, keyword: string) => {
  return (
    <GroupNameCell
      showEmail={false}
      group={group}
      showIcon={false}
      link={false}
      keyword={keyword}
    />
  );
};

const renderLabel = computed(() => {
  return getRenderLabelFunc({
    multiple: props.multiple,
    customLabel,
    showResourceName: true,
  });
});

const renderTag = computed(() => {
  return getRenderTagFunc({
    multiple: props.multiple,
    disabled: props.disabled,
    size: props.size,
    customLabel,
  });
});
</script>
