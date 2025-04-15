<template>
  <div class="min-w-[14rem] max-w-[18rem] gap-y-1">
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
    <InfoItem v-if="columnMetadata.userComment" :title="$t('database.comment')">
      {{ columnMetadata.userComment }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import { CheckIcon, XIcon } from "lucide-vue-next";
import { computed } from "vue";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorLite";
import { useDBSchemaV1Store, useDatabaseV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { ColumnMetadata } from "@/types/proto/v1/database_service";
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
    ColumnMetadata.create()
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
