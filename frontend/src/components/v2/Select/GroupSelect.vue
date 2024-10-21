<template>
  <NSelect
    :filterable="true"
    :value="validValue"
    :options="options"
    :disabled="disabled"
    :clearable="clearable"
    :multiple="multiple"
    :filter="filterByTitle"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :placeholder="$t('settings.members.select-group', multiple ? 2 : 1)"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="tsx" setup>
import { NCheckbox, NSelect, NTag, type SelectOption } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed } from "vue";
import {
  getMemberBindingsByRole,
  getMemberBindings,
} from "@/components/Member/utils";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { useGroupStore, useProjectV1Store, useWorkspaceV1Store } from "@/store";
import { PRESET_WORKSPACE_ROLES } from "@/types";
import type { Group } from "@/types/proto/v1/group";

export interface GroupSelectOption extends SelectOption {
  value: string;
  group: Group;
}

const props = withDefaults(
  defineProps<{
    value?: string[] | string | undefined;
    disabled?: boolean;
    clearable?: boolean;
    multiple?: boolean;
    projectName?: string;
    selectFirstAsDefault?: boolean;
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    clearable: false,
    value: undefined,
    multiple: false,
    projectName: undefined,
    selectFirstAsDefault: true,
    size: "medium",
  }
);

defineEmits<{
  (event: "update:value", val: string | string[]): void;
}>();

const groupStore = useGroupStore();
const projectV1Store = useProjectV1Store();
const workspaceStore = useWorkspaceV1Store();

const getGroupListFromProject = (projectName: string) => {
  const project = projectV1Store.getProjectByName(projectName);
  const memberMap = getMemberBindingsByRole({
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

  return getMemberBindings(memberMap)
    .map((binding) => binding.group)
    .filter((g) => g) as Group[];
};

const groupList = computed(() =>
  props.projectName
    ? getGroupListFromProject(props.projectName)
    : groupStore.groupList
);

const options = computed(() => {
  return groupList.value.map<GroupSelectOption>((group) => ({
    value: group.name,
    label: group.title,
    group,
  }));
});

const validValue = computed(() => {
  if (!props.value) {
    return props.value;
  }
  if (props.multiple) {
    return (props.value as string[]).filter((v) => {
      return options.value.findIndex((o) => o.value === v) >= 0;
    });
  }

  if (options.value.findIndex((o) => o.value === props.value) >= 0) {
    return props.value;
  }
  return undefined;
});

const filterByTitle = (pattern: string, option: SelectOption) => {
  const { group } = option as GroupSelectOption;
  pattern = pattern.toLowerCase();
  return (
    group.title.toLowerCase().includes(pattern) ||
    group.name.includes(pattern.toLowerCase())
  );
};

const renderLabel = (option: SelectOption, selected: boolean) => {
  const { group } = option as GroupSelectOption;

  return (
    <div class="flex items-start space-x-2 py-2">
      <NCheckbox checked={selected} size="small" class="mt-1" />
      <GroupNameCell group={group} showIcon={false} link={false} />
    </div>
  );
};

const renderTag = ({
  option,
  handleClose,
}: {
  option: SelectBaseOption;
  handleClose: () => void;
}) => {
  const { group } = option as GroupSelectOption;
  return (
    <NTag size={props.size} closable={!props.disabled} onClose={handleClose}>
      <GroupNameCell
        group={group}
        showIcon={false}
        link={false}
        showEmail={false}
      />
    </NTag>
  );
};
</script>
