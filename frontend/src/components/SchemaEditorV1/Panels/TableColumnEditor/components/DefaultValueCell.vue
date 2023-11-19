<template>
  <DropdownInput
    :value="value"
    :options="options"
    :placeholder="placeholder"
    :consistent-menu-width="false"
    style="--n-padding-left: 6px; --n-padding-right: 22px; --n-font-size: 14px"
    class="bb-schema-editor--column-default-value-select"
    @update:value="handleUpdateValue"
  />
</template>

<script lang="ts" setup>
import { SelectOption } from "naive-ui";
import { computed } from "vue";
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
</script>

<style lang="postcss" scoped>
.bb-schema-editor--column-default-value-select :deep(.n-base-selection) {
  --n-padding-single: 0 22px 0 6px !important;
}
</style>
