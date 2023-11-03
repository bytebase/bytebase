<template>
  <div
    class="grid min-w-[14rem] max-w-[18rem] gap-x-2 gap-y-1 break-all"
    style="grid-template-columns: auto 1fr"
  >
    <InfoItem :title="$t('common.name')">
      {{ column.name }}
    </InfoItem>
    <InfoItem :title="$t('common.type')">
      {{ column.type }}
    </InfoItem>
    <InfoItem :title="$t('common.Default')">
      {{ getColumnDefaultValuePlaceholder(column) }}
    </InfoItem>
    <InfoItem :title="$t('database.nullable')">
      {{ column.nullable }}
    </InfoItem>
    <InfoItem v-if="characterSet" :title="$t('db.character-set')">
      {{ characterSet }}
    </InfoItem>
    <InfoItem v-if="collation" :title="$t('db.collation')">
      {{ collation }}
    </InfoItem>
    <InfoItem v-if="column.userComment" :title="$t('database.comment')">
      {{ column.userComment }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorV1/utils/columnDefaultValue";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import InfoItem from "./InfoItem.vue";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  column: ColumnMetadata;
}>();

const instanceEngine = computed(() => {
  return props.db.instanceEntity.engine;
});

const characterSet = computed(() => {
  if (
    [Engine.POSTGRES, Engine.CLICKHOUSE, Engine.SNOWFLAKE].includes(
      instanceEngine.value
    )
  ) {
    return props.column.characterSet;
  }
  return "";
});

const collation = computed(() => {
  if ([Engine.CLICKHOUSE, Engine.SNOWFLAKE].includes(instanceEngine.value)) {
    return props.column.collation;
  }
  return "";
});
</script>
