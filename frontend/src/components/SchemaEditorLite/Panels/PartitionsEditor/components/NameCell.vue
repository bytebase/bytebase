<template>
  <InlineInput
    :value="partition.name"
    :disabled="readonly"
    :placeholder="$t('common.name')"
    style="
      --n-padding-left: 6px;
      --n-padding-right: 4px;
      --n-text-color-disabled: rgb(var(--color-main));
      width: inherit;
    "
    @update:value="$emit('update:name', $event)"
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
  (event: "update:name", name: string): void;
}>();
</script>
