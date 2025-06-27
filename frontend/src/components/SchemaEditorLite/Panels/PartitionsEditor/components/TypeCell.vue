<template>
  <NSelect
    :value="partition.type || null"
    :options="typeOptions"
    :style="style"
    :disabled="readonly"
    :show-arrow="!readonly"
    :placeholder="$t('schema-editor.table-partition.type')"
    suffix-style="right: 3px"
    class="bb-schema-editor--table-partition-type-select"
    @focus="focused = true"
    @blur="focused = false"
    @update:value="$emit('update:type', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect, type SelectOption } from "naive-ui";
import type { CSSProperties } from "vue";
import { computed, ref } from "vue";
import type { TablePartitionMetadata } from "@/types/proto-es/v1/database_service_pb";
import {
  TablePartitionMetadata_Type,
} from "@/types/proto-es/v1/database_service_pb";

const props = defineProps<{
  readonly?: boolean;
  partition: TablePartitionMetadata;
  parent?: TablePartitionMetadata;
}>();
defineEmits<{
  (event: "update:type", type: TablePartitionMetadata_Type): void;
}>();

const focused = ref(false);

const typeOptions = computed(() => {
  const values: TablePartitionMetadata_Type[] = props.parent
    ? [
        TablePartitionMetadata_Type.HASH,
        TablePartitionMetadata_Type.LINEAR_HASH,
        TablePartitionMetadata_Type.KEY,
        TablePartitionMetadata_Type.LINEAR_KEY,
      ]
    : [
        TablePartitionMetadata_Type.RANGE,
        TablePartitionMetadata_Type.RANGE_COLUMNS,
        TablePartitionMetadata_Type.LIST,
        TablePartitionMetadata_Type.LIST_COLUMNS,
        TablePartitionMetadata_Type.HASH,
        TablePartitionMetadata_Type.LINEAR_HASH,
        TablePartitionMetadata_Type.KEY,
        TablePartitionMetadata_Type.LINEAR_KEY,
      ];
  return values.map<SelectOption>((type) => {
    // Convert enum number to string label
    const label = TablePartitionMetadata_Type[type] || `UNKNOWN_${type}`;
    return {
      value: type,
      label,
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
.bb-schema-editor--table-partition-type-select :deep(.n-base-selection) {
  --n-padding-single: 0 16px 0 6px !important;
  --n-color: transparent !important;
  --n-color-disabled: transparent !important;
  --n-border: none !important;
  --n-text-color-disabled: rgb(var(--color-main)) !important;
}
.bb-schema-editor--table-partition-type-select
  :deep(.n-base-selection .n-base-suffix) {
  right: 4px;
}
</style>
