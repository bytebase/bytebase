<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :value="group"
    :values="groups"
    :disabled="disabled"
    :multiple="multiple"
    :custom-label="renderLabel"
    :additional-data="additionalData"
    :search="handleSearch"
    :get-option="getOption"
    @update:value="(val) => $emit('update:group', val)"
    @update:values="(val) => $emit('update:groups', val)"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { useGroupStore } from "@/store";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import RemoteResourceSelector from "./RemoteResourceSelector.vue";

const props = withDefaults(
  defineProps<{
    group?: string | undefined;
    groups?: string[] | undefined;
    disabled?: boolean;
    multiple?: boolean;
    projectName?: string;
  }>(),
  {
    group: undefined,
    groups: undefined,
    multiple: false,
    projectName: undefined,
  }
);

defineEmits<{
  (event: "update:group", val: string | undefined): void;
  (event: "update:groups", val: string[]): void;
}>();

const groupStore = useGroupStore();

const additionalData = computedAsync(async () => {
  const data = [];

  let groupNames: string[] = [];
  if (props.group) {
    groupNames = [props.group];
  } else if (props.groups) {
    groupNames = props.groups;
  }

  const groups = await groupStore.batchFetchGroups(groupNames);
  for (const group of groups) {
    data.push(group);
  }

  return data;
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
    data: groups,
  };
};

const getOption = (group: Group) => ({
  value: group.name,
  label: group.title,
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
