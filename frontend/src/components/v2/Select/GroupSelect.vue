<template>
  <ResourceSelect
    :value="validSelectedGroup"
    :values="validSelectedGroups"
    :disabled="disabled"
    :multiple="multiple"
    :options="options"
    :custom-label="renderLabel"
    :placeholder="$t('settings.members.select-group', multiple ? 2 : 1)"
    @update:value="(val) => $emit('update:group', val)"
    @update:values="(val) => $emit('update:groups', val)"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { getMemberBindings } from "@/components/Member/utils";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { useGroupList, useProjectV1Store, useWorkspaceV1Store } from "@/store";
import { PRESET_WORKSPACE_ROLES } from "@/types";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
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

const groupList = useGroupList();
const projectV1Store = useProjectV1Store();
const workspaceStore = useWorkspaceV1Store();

const filteredGroupList = computedAsync(async () => {
  if (!props.projectName) {
    return groupList.value;
  }
  const project = await projectV1Store.getOrFetchProjectByName(
    props.projectName
  );
  const bindings = await getMemberBindings({
    policies: [
      {
        level: "WORKSPACE",
        policy: workspaceStore.workspaceIamPolicy,
      },
      {
        level: "PROJECT",
        policy: project.iamPolicy,
      },
    ],
    searchText: "",
    ignoreRoles: new Set(PRESET_WORKSPACE_ROLES),
  });

  return bindings.map((binding) => binding.group).filter((g) => g) as Group[];
}, []);

const options = computed(() => {
  return filteredGroupList.value.map((group) => ({
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
