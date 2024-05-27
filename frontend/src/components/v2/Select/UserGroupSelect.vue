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
import { NCheckbox, NTag, type SelectOption } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useUserGroupStore } from "@/store";
import type { UserGroup } from "@/types/proto/v1/user_group";

export interface GroupSelectOption extends SelectOption {
  value: string;
  group: UserGroup;
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

const { t } = useI18n();
const groupStore = useUserGroupStore();

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
      <div>
        <div class="flex items-center">
          <span class="font-medium">{group.title}</span>
          <span class="ml-1 font-normal text-control-light">
            (
            {t("settings.members.groups.n-members", {
              n: group.members.length,
            })}
            )
          </span>
        </div>
        <span class="textinfolabel text-sm">{group.description}</span>
      </div>
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
      <div class="flex items-center">
        <span class="font-medium">{group.title}</span>
        <span class="ml-1 font-normal text-control-light">
          ({t("settings.members.groups.n-members", { n: group.members.length })}
          )
        </span>
      </div>
    </NTag>
  );
};
</script>
