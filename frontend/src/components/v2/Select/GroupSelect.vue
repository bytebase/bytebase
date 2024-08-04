<template>
  <NSelect
    :filterable="true"
    :value="value"
    :options="options"
    :disabled="disabled"
    :clearable="clearable"
    :multiple="multiple"
    :filter="filterByTitle"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :placeholder="'Select group'"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="tsx" setup>
import { NCheckbox, NSelect, NTag, type SelectOption } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed } from "vue";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { useGroupStore } from "@/store";
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
    selectFirstAsDefault?: boolean;
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    clearable: false,
    value: undefined,
    multiple: false,
    selectFirstAsDefault: true,
    size: "medium",
  }
);

defineEmits<{
  (event: "update:value", val: string | string[]): void;
}>();

const groupStore = useGroupStore();

const options = computed(() => {
  return groupStore.groupList.map<GroupSelectOption>((group) => ({
    value: group.name,
    label: group.title,
    group,
  }));
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
