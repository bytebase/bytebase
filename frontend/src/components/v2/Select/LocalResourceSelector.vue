<template>
  <NSelect
    v-bind="$attrs"
    :filterable="true"
    :clearable="true"
    :remote="false"
    :virtual-scroll="true"
    :multiple="multiple"
    :value="selected"
    :disabled="disabled"
    :options="options"
    :fallback-option="false"
    :filter="filterResource"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :placeholder="placeholder"
    :size="size"
    :consistent-menu-width="consistentMenuWidth"
    class="bb-user-select"
    @update:value="handleValueUpdated"
  >
    <template v-if="$slots.empty" #empty>
      <slot name="empty" />
    </template>
  </NSelect>
</template>

<script lang="tsx" setup generic="T extends { name: string }">
import type { SelectOption } from "naive-ui";
import { NCheckbox, NSelect, NTag } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed, type VNodeChild } from "vue";
import EllipsisText from "@/components/EllipsisText.vue";

type ResourceSelectOption = SelectOption & {
  resource: T;
  value: string;
  label: string;
};

const props = withDefaults(
  defineProps<{
    placeholder?: string;
    multiple?: boolean;
    disabled?: boolean;
    value?: string | undefined | null;
    values?: string[] | undefined | null;
    consistentMenuWidth?: boolean;
    size?: "tiny" | "small" | "medium" | "large";
    options: ResourceSelectOption[];
    showResourceName?: boolean;
    resourceNameClass?: string;
    customLabel?: (resource: T) => VNodeChild;
  }>(),
  {
    customLabel: undefined,
    size: "medium",
    disabled: false,
    multiple: false,
    showResourceName: true,
    resourceNameClass: "",
    consistentMenuWidth: true,
  }
);

const emit = defineEmits<{
  (event: "update:value", value: string | undefined): void;
  (event: "update:values", value: string[]): void;
}>();

const selected = computed(() => {
  if (props.multiple) {
    return props.values || [];
  } else {
    return props.value;
  }
});

const renderLabel = (option: SelectOption, selected: boolean) => {
  const { resource, label } = option as ResourceSelectOption;
  const node = (
    <div class="py-1">
      {props.customLabel ? props.customLabel(resource) : label}
      {props.showResourceName && (
        <div>
          <EllipsisText
            class={`opacity-60 textinfolabel ${props.resourceNameClass}`}
          >
            {resource.name}
          </EllipsisText>
        </div>
      )}
    </div>
  );
  if (props.multiple) {
    return (
      <div class="flex items-center gap-x-2 py-2">
        <NCheckbox checked={selected} size="small" />
        {node}
      </div>
    );
  }

  return node;
};

const renderTag = ({
  option,
  handleClose,
}: {
  option: SelectBaseOption;
  handleClose: () => void;
}) => {
  const { resource, label } = option as ResourceSelectOption;
  const node = props.customLabel ? props.customLabel(resource) : label;
  if (props.multiple) {
    return (
      <NTag size={props.size} closable={!props.disabled} onClose={handleClose}>
        {node}
      </NTag>
    );
  }
  return node;
};

const filterResource = (pattern: string, option: SelectOption) => {
  const { resource, label } = option as ResourceSelectOption;
  const search = pattern.trim().toLowerCase();
  return (
    resource.name.toLowerCase().includes(search) ||
    (label as string).toLowerCase().includes(search)
  );
};

const handleValueUpdated = (value: string | string[] | undefined | null) => {
  if (props.multiple) {
    if (!value) {
      // normalize value
      value = [];
    }
    emit("update:values", value as string[]);
  } else {
    if (value === null) {
      // normalize value
      value = "";
    }
    emit("update:value", value as string);
  }
};
</script>
