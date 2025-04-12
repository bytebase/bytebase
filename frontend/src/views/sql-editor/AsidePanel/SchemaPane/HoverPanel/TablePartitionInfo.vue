<template>
  <div class="min-w-[14rem] max-w-[18rem] gap-y-1">
    <InfoItem :title="$t('common.name')">
      {{ partition.name }}
    </InfoItem>
    <InfoItem :title="$t('schema-editor.table-partition.type')">
      {{ tablePartitionMetadata_TypeToJSON(partition.type) }}
    </InfoItem>
    <InfoItem :title="$t('schema-editor.table-partition.expression')">
      <code>{{ partition.expression }}</code>
    </InfoItem>
    <InfoItem :title="$t('schema-editor.table-partition.value')">
      {{ partition.value }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import type { ComposedDatabase } from "@/types";
import {
  tablePartitionMetadata_TypeToJSON,
  type TablePartitionMetadata,
} from "@/types/proto/v1/database_service";
import InfoItem from "./InfoItem.vue";

defineProps<{
  db: ComposedDatabase;
  partition: TablePartitionMetadata;
}>();
</script>
