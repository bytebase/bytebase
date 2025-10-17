<template>
  <NInput
    ref="inputRef"
    :value="column.default"
    :placeholder="$t('schema-editor.default.placeholder')"
    :disabled="disabled"
    :style="simpleInputStyle"
    @focus="focused = true"
    @blur="focused = false"
    @update:value="handleSimpleInput"
  />
</template>

<script lang="ts" setup>
import { type InputInst, NInput } from "naive-ui";
import type { CSSProperties } from "vue";
import { computed, ref } from "vue";
import type { DefaultValue } from "@/components/SchemaEditorLite/utils";
import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";

const props = withDefaults(
  defineProps<{
    column: ColumnMetadata;
    disabled?: boolean;
    border?: string;
  }>(),
  {
    border: "none",
  }
);

const emit = defineEmits<{
  (event: "update", option: DefaultValue): void;
}>();

const focused = ref(false);
const inputRef = ref<InputInst>();

const simpleInputStyle = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    "--n-color": "transparent",
    "--n-color-disabled": "transparent",
    "--n-padding-left": "6px",
    "--n-padding-right": "4px",
    "--n-text-color-disabled": "rgb(var(--color-main))",
  };
  const border = focused.value
    ? "1px solid rgb(var(--color-control-border))"
    : props.border;
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

const handleSimpleInput = (value: string) => {
  const defaultValue = {
    hasDefault: !!value.trim(),
    default: value.trim(), // Both PostgreSQL and MySQL use default for now
  };
  emit("update", defaultValue);
};
</script>
