<template>
  <div class="min-w-56 max-w-[18rem] gap-y-1">
    <InfoItem :title="$t('common.name')">
      {{ partitionMetadata.name }}
    </InfoItem>
    <InfoItem :title="$t('schema-editor.table-partition.type')">
      {{ TablePartitionMetadata_Type[partitionMetadata.type] }}
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
import { create } from "@bufbuild/protobuf";
import { computed } from "vue";
import { useDBSchemaV1Store } from "@/store";
import {
  TablePartitionMetadata_Type,
  TablePartitionMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
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
    create(TablePartitionMetadataSchema, {})
);
</script>
