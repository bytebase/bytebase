<template>
  <NSelect
    :value="index.expressions"
    :options="columnOptions"
    :disabled="readonly"
    :consistent-menu-width="true"
    :style="style"
    :multiple="true"
    :max-tag-count="'responsive'"
    :placeholder="$t('schema-editor.columns')"
    :render-tag="renderTag"
    suffix-style="right: 3px"
    class="bb-schema-editor--index-columns-select"
    @focus="focused = true"
    @blur="focused = false"
    @update:value="$emit('update:expressions', $event as string[])"
  />
</template>

<script lang="ts" setup>
import { NSelect, NTag, SelectOption } from "naive-ui";
import { SelectBaseOption } from "naive-ui/es/select/src/interface";
import { CSSProperties, computed, h, ref } from "vue";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

type ColumnOption = SelectOption & {
  label: string;
  value: string;
  column: ColumnMetadata;
};

const props = defineProps<{
  readonly?: boolean;
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  index: IndexMetadata;
}>();
defineEmits<{
  (event: "update:expressions", expressions: string[]): void;
}>();

const focused = ref(false);

const columnOptions = computed(() => {
  return props.table.columns.map<ColumnOption>((column) => ({
    label: column.name,
    value: column.name,
    column,
  }));
});

const style = computed(() => {
  const style: CSSProperties = {
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

const renderTag = (props: {
  option: SelectBaseOption;
  handleClose: () => void;
}) => {
  return h(
    NTag,
    {
      size: "small",
      closable: true,
      onClose: props.handleClose,
    },
    {
      default: () => props.option.label,
    }
  );
};
</script>

<style lang="postcss" scoped>
.bb-schema-editor--index-columns-select :deep(.n-base-selection) {
  --n-padding-multiple: 4px 16px 2px 4px !important;
  --n-color: transparent !important;
  --n-color-disabled: transparent !important;
  --n-border: none !important;
  --n-text-color-disabled: rgb(var(--color-main)) !important;
}
.bb-schema-editor--index-columns-select
  :deep(.n-base-selection .n-base-suffix) {
  right: 4px;
}
</style>
