<template>
  <InlineInput
    :value="partition.expression"
    :disabled="readonly"
    :placeholder="$t('schema-editor.table-partition.expression')"
    style="
      --n-padding-left: 6px;
      --n-padding-right: 4px;
      --n-text-color-disabled: rgb(var(--color-main));
    "
    @update:value="$emit('update:expression', $event)"
  />
</template>

<script lang="ts" setup>
import { InlineInput } from "@/components/v2";
import { type TablePartitionMetadata } from "@/types/proto-es/v1/database_service_pb";

defineProps<{
  readonly?: boolean;
  partition: TablePartitionMetadata;
}>();
defineEmits<{
  (event: "update:expression", expression: string): void;
}>();
</script>
