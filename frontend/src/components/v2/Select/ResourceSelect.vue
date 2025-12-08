<template>
  <NSelect
    v-bind="$attrs"
    :filterable="true"
    :clearable="true"
    :virtual-scroll="true"
    :multiple="multiple"
    :value="selected"
    :disabled="disabled"
    :options="options"
    :fallback-option="fallbackOption"
    :filter="filterResource"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :placeholder="placeholder"
    :size="size"
    :consistent-menu-width="consistentMenuWidth"
    class="bb-user-select"
    @search="handleSearch"
    @click="() => handleSearch('')"
    @update:value="handleValueUpdated"
  >
    <template v-if="showHint" #header>
      <div class="flex items-center gap-x-2 px-3 py-2">
        <span class="flex-1 text-sm text-gray-500 truncate">{{ hint }}</span>
        <button
          type="button"
          class="flex-shrink-0 text-gray-400 hover:text-gray-600"
          @click="dismissHint"
        >
          <svg
            class="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>
    </template>
    <template v-if="$slots.empty" #empty>
      <slot name="empty" />
    </template>
  </NSelect>
</template>

<script lang="tsx" setup generic="T extends { name: string }">
import type { SelectOption, SelectProps } from "naive-ui";
import { NCheckbox, NSelect, NTag } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed, onMounted, ref, type VNodeChild } from "vue";
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
    fallbackOption?: SelectProps["fallbackOption"];
    showResourceName?: boolean;
    resourceNameClass?: string;
    customLabel?: (resource: T) => VNodeChild;
    filter?: (pattern: string, resource: T) => boolean;
    hint?: string;
    // Unique key to persist hint dismissal in localStorage
    hintKey?: string;
  }>(),
  {
    customLabel: undefined,
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    filter: (pattern: string, resource: T) => false,
    size: "medium",
    fallbackOption: false,
    disabled: false,
    multiple: false,
    showResourceName: true,
    resourceNameClass: "",
    consistentMenuWidth: true,
    hint: undefined,
    hintKey: undefined,
  }
);

const hintDismissed = ref(false);

const getHintStorageKey = () =>
  props.hintKey ? `bb.hint.dismissed.${props.hintKey}` : null;

const showHint = computed(() => props.hint && !hintDismissed.value);

const dismissHint = (e: Event) => {
  e.stopPropagation();
  hintDismissed.value = true;
  const key = getHintStorageKey();
  if (key) {
    localStorage.setItem(key, "true");
  }
};

onMounted(() => {
  const key = getHintStorageKey();
  if (key && localStorage.getItem(key) === "true") {
    hintDismissed.value = true;
  }
});

const emit = defineEmits<{
  (event: "update:value", value: string | undefined): void;
  (event: "update:values", value: string[]): void;
  (event: "search", value: string): void;
}>();

const selected = computed(() => {
  if (props.multiple) {
    return props.values || [];
  } else {
    return props.value;
  }
});

const handleSearch = (search: string) => {
  emit("search", search.trim().toLowerCase());
};

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
    (label as string).toLowerCase().includes(search) ||
    props.filter(search, resource)
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
