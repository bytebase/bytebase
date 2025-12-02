<template>
  <div class="min-w-56 max-w-[18rem] gap-y-1">
    <InfoItem :title="$t('common.name')">
      {{ columnMetadata.name }}
    </InfoItem>
    <InfoItem :title="$t('common.type')">
      {{ columnMetadata.type }}
    </InfoItem>
    <InfoItem :title="$t('common.Default')">
      {{ getColumnDefaultValuePlaceholder(columnMetadata) }}
    </InfoItem>
    <InfoItem :title="$t('database.nullable')">
      <div class="inline-flex items-center justify-end">
        <CheckIcon v-if="columnMetadata.nullable" class="w-4 h-4" />
        <XIcon v-else class="w-4 h-4" />
      </div>
    </InfoItem>
    <InfoItem v-if="characterSet" :title="$t('db.character-set')">
      {{ characterSet }}
    </InfoItem>
    <InfoItem v-if="collation" :title="$t('db.collation')">
      {{ collation }}
    </InfoItem>
    <InfoItem v-if="columnMetadata.comment" :title="$t('database.comment')">
      {{ columnMetadata.comment }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { CheckIcon, XIcon } from "lucide-vue-next";
import { computed } from "vue";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorLite";
import { useDatabaseV1Store, useDBSchemaV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { ColumnMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import InfoItem from "./InfoItem.vue";

const props = defineProps<{
  database: string;
  schema?: string;
  table: string;
  column: string;
}>();

const dbSchema = useDBSchemaV1Store();
const databaseStore = useDatabaseV1Store();

const columnMetadata = computed(
  () =>
    dbSchema
      .getTableMetadata({
        database: props.database,
        schema: props.schema,
        table: props.table,
      })
      .columns.find((col) => col.name === props.column) ??
    create(ColumnMetadataSchema, {})
);

const instanceEngine = computed(
  () => databaseStore.getDatabaseByName(props.database).instanceResource.engine
);

const characterSet = computed(() => {
  if (
    [Engine.POSTGRES, Engine.CLICKHOUSE, Engine.SNOWFLAKE].includes(
      instanceEngine.value
    )
  ) {
    return columnMetadata.value.characterSet;
  }
  return "";
});

const collation = computed(() => {
  if ([Engine.CLICKHOUSE, Engine.SNOWFLAKE].includes(instanceEngine.value)) {
    return columnMetadata.value.collation;
  }
  return "";
});
</script>
