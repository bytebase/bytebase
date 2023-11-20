<template>
  <DropdownInput
    :value="value"
    :options="options"
    :placeholder="placeholder"
    :consistent-menu-width="false"
    :style="style"
    suffix-style="right: 3px"
    class="bb-schema-editor--column-default-value-select"
    @focus="focused = true"
    @blur="focused = false"
    @update:value="handleUpdateValue"
  />
</template>

<script lang="ts" setup>
import { SelectOption } from "naive-ui";
import { CSSProperties, computed, ref } from "vue";
import {
  DefaultValueOption,
  getColumnDefaultDisplayString,
  getColumnDefaultValueOptions,
  getColumnDefaultValuePlaceholder,
  getDefaultValueByKey,
} from "@/components/SchemaEditorV1/utils/columnDefaultValue";
import { DropdownInput } from "@/components/v2";
import { Engine } from "@/types/proto/v1/common";
import { Column } from "@/types/v1/schemaEditor";

const props = defineProps<{
  column: Column;
  engine: Engine;
}>();
const emit = defineEmits<{
  (event: "input", value: string): void;
  (event: "select", option: DefaultValueOption): void;
}>();
const focused = ref(false);

const value = computed(() => {
  return getColumnDefaultDisplayString(props.column) ?? null;
});

const placeholder = computed(() => {
  return getColumnDefaultValuePlaceholder(props.column);
});

const options = computed(() => {
  return getColumnDefaultValueOptions(
    props.engine,
    props.column.type
  ).map<SelectOption>((opt) => ({
    value: opt.key,
    label: opt.label as string,
    defaultValue: opt.value,
  }));
});

const handleUpdateValue = (key: string) => {
  const value = getDefaultValueByKey(key);
  if (value) {
    emit("select", { key, value });
    return;
  }

  emit("input", key);
};

const style = computed(() => {
  const style: CSSProperties = {
    "--n-padding-left": "6px",
    "--n-padding-right": "16px",
    "--n-color": "transparent",
    "--n-color-disabled": "transparent",
    "--n-text-color-disabled": "rgb(var(--color-main))",
    cursor: "default",
  };
  const border = focused.value
    ? "1px solid rgb(var(--color-control-border))"
    : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});
</script>

<style lang="postcss" scoped>
.bb-schema-editor--column-default-value-select :deep(.n-base-selection) {
  --n-padding-single: 0 16px 0 6px !important;
  --n-color: transparent !important;
  --n-color-disabled: transparent !important;
  --n-border: none !important;
  --n-text-color-disabled: rgb(var(--color-main)) !important;
}
.bb-schema-editor--column-default-value-select
  :deep(.n-base-selection .n-base-suffix) {
  right: 4px;
}
</style>
