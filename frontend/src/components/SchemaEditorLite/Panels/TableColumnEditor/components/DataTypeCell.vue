<template>
  <DropdownInput
    :value="column.type || null"
    :allow-input-value="allowInputValue"
    :options="columnTypeOptions"
    :consistent-menu-width="true"
    :allow-filter="allowFilter"
    :style="style"
    suffix-style="right: 3px"
    placeholder="column type"
    class="bb-schema-editor--column-data-type-select"
    @focus="focused = true"
    @blur="focused = false"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import type { SelectOption } from "naive-ui";
import type { CSSProperties } from "vue";
import { computed, ref, onMounted } from "vue";
import { DropdownInput } from "@/components/v2";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import { getDataTypeSuggestionList } from "@/utils";

const props = defineProps<{
  column: ColumnMetadata;
  readonly?: boolean;
  engine: Engine;
  schemaTemplateColumnTypes: string[];
}>();
defineEmits<{
  (event: "update:value", value: string): void;
}>();

const focused = ref(false);
const originalType = ref("");

onMounted(() => (originalType.value = props.column.type));

const allowFilter = computed(() => {
  return !props.column.type || originalType.value !== props.column.type;
});

const allowInputValue = computed(() => {
  return props.schemaTemplateColumnTypes.length === 0;
});

const columnTypeOptions = computed(() => {
  const { schemaTemplateColumnTypes, engine } = props;
  if (allowInputValue.value) {
    return getDataTypeSuggestionList(engine).map<SelectOption>((dataType) => {
      return {
        value: dataType,
        label: dataType,
      };
    });
  }
  return schemaTemplateColumnTypes.map<SelectOption>((columnType) => {
    return {
      value: columnType,
      label: columnType,
    };
  });
});

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
.bb-schema-editor--column-data-type-select :deep(.n-base-selection) {
  --n-padding-single: 0 16px 0 6px !important;
  --n-color: transparent !important;
  --n-color-disabled: transparent !important;
  --n-border: none !important;
  --n-text-color-disabled: rgb(var(--color-main)) !important;
}
.bb-schema-editor--column-data-type-select
  :deep(.n-base-selection .n-base-suffix) {
  right: 4px;
}
</style>
