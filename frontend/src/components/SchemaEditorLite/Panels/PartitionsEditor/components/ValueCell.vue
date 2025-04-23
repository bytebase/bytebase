<template>
  <InlineInput
    :value="partition.value"
    :disabled="readonly"
    :placeholder="$t('schema-editor.table-partition.value')"
    style="
      --n-padding-left: 6px;
      --n-padding-right: 4px;
      --n-text-color-disabled: rgb(var(--color-main));
    "
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { InlineInput } from "@/components/v2";
import { type TablePartitionMetadata } from "@/types/proto/api/v1alpha/database_service";

defineProps<{
  readonly?: boolean;
  partition: TablePartitionMetadata;
}>();
defineEmits<{
  (event: "update:value", value: string): void;
}>();
</script>
