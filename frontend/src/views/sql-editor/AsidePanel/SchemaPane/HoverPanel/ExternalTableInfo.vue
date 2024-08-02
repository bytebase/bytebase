<template>
  <div class="min-w-[14rem] max-w-[18rem] gap-y-1">
    <InfoItem :title="$t('common.name')">
      {{ name }}
    </InfoItem>
    <InfoItem :title="$t('database.external-server-name')">
      {{ externalTable.externalServerName }}
    </InfoItem>
    <InfoItem :title="$t('database.external-database-name')">
      {{ externalTable.externalDatabaseName }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  ExternalTableMetadata,
} from "@/types/proto/v1/database_service";
import { hasSchemaProperty } from "@/utils";
import InfoItem from "./InfoItem.vue";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  externalTable: ExternalTableMetadata;
}>();

const instanceEngine = computed(() => props.db.instanceResource.engine);

const name = computed(() => {
  const { schema, externalTable } = props;
  if (hasSchemaProperty(instanceEngine.value)) {
    return `${schema.name}.${externalTable.name}`;
  }
  return externalTable.name;
});
</script>
