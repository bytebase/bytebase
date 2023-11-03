<template>
  <div
    class="grid min-w-[14rem] max-w-[18rem] gap-x-2 gap-y-1 break-all"
    style="grid-template-columns: auto 1fr"
  >
    <InfoItem :title="$t('common.name')">
      {{ name }}
    </InfoItem>
    <InfoItem v-if="engine" :title="$t('database.engine')">
      {{ engine }}
    </InfoItem>
    <InfoItem :title="$t('database.row-count-estimate')">
      {{ table.rowCount }}
    </InfoItem>
    <InfoItem :title="$t('database.data-size')">
      {{ bytesToString(table.dataSize.toNumber()) }}
    </InfoItem>
    <InfoItem v-if="indexSize" :title="$t('database.index-size')">
      {{ indexSize }}
    </InfoItem>
    <InfoItem v-if="collation" :title="$t('db.collation')">
      {{ collation }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { bytesToString } from "@/utils";
import InfoItem from "./InfoItem.vue";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}>();

const instanceEngine = computed(() => props.db.instanceEntity.engine);

const hasSchemaProperty = computed(() => {
  return [Engine.POSTGRES, Engine.RISINGWAVE].includes(instanceEngine.value);
});

const name = computed(() => {
  const { schema, table } = props;
  if (hasSchemaProperty.value) {
    return `${schema.name}.${table.name}`;
  }
  return table.name;
});

const engine = computed(() => {
  if ([Engine.POSTGRES, Engine.SNOWFLAKE].includes(instanceEngine.value)) {
    return "";
  }
  return props.table.engine;
});

const indexSize = computed(() => {
  if ([Engine.CLICKHOUSE, Engine.SNOWFLAKE].includes(instanceEngine.value)) {
    return "";
  }
  return bytesToString(props.table.indexSize.toNumber());
});

const collation = computed(() => {
  if (
    [Engine.CLICKHOUSE, Engine.SNOWFLAKE, Engine.POSTGRES].includes(
      instanceEngine.value
    )
  ) {
    return "";
  }
  return props.table.collation;
});
</script>
