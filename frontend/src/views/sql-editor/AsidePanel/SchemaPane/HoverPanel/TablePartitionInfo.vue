<template>
  <div class="min-w-[14rem] max-w-[18rem] gap-y-1">
    <InfoItem :title="$t('common.name')">
      {{ partitionMetadata.name }}
    </InfoItem>
    <InfoItem :title="$t('schema-editor.table-partition.type')">
      {{ tablePartitionMetadata_TypeToJSON(partitionMetadata.type) }}
    </InfoItem>
    <InfoItem :title="$t('schema-editor.table-partition.expression')">
      <code>{{ partitionMetadata.expression }}</code>
    </InfoItem>
    <InfoItem :title="$t('schema-editor.table-partition.value')">
      {{ partitionMetadata.value }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useDBSchemaV1Store } from "@/store";
import {
  tablePartitionMetadata_TypeToJSON,
  TablePartitionMetadata,
} from "@/types/proto/v1/database_service";
import InfoItem from "./InfoItem.vue";

const props = defineProps<{
  database: string;
  schema?: string;
  table: string;
  partition: string;
}>();

const dbSchema = useDBSchemaV1Store();

const partitionMetadata = computed(
  () =>
    dbSchema
      .getTableMetadata({
        database: props.database,
        schema: props.schema,
        table: props.table,
      })
      .partitions.find((p) => p.name === props.partition) ??
    TablePartitionMetadata.create()
);
</script>
