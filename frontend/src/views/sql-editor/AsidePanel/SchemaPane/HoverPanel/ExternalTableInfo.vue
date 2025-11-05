<template>
  <div class="min-w-56 max-w-[18rem] gap-y-1">
    <InfoItem :title="$t('common.name')">
      {{ externalTableMetadata.name }}
    </InfoItem>
    <InfoItem :title="$t('database.external-server-name')">
      {{ externalTableMetadata.externalServerName }}
    </InfoItem>
    <InfoItem :title="$t('database.external-database-name')">
      {{ externalTableMetadata.externalDatabaseName }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useDBSchemaV1Store } from "@/store";
import InfoItem from "./InfoItem.vue";

const props = defineProps<{
  database: string;
  schema?: string;
  externalTable: string;
}>();

const dbSchema = useDBSchemaV1Store();

const externalTableMetadata = computed(() =>
  dbSchema.getExternalTableMetadata({
    database: props.database,
    schema: props.schema,
    externalTable: props.externalTable,
  })
);
</script>
