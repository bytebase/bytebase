<template>
  <div class="min-w-56 max-w-[18rem] gap-y-1">
    <InfoItem :title="$t('common.name')">
      {{ tableMetadata.name }}
    </InfoItem>
    <InfoItem :title="$t('database.engine')">
      <RichEngineName :engine="instanceEngine" />
    </InfoItem>
    <InfoItem :title="$t('database.row-count-estimate')">
      {{ tableMetadata.rowCount }}
    </InfoItem>
    <InfoItem :title="$t('database.data-size')">
      {{ bytesToString(Number(tableMetadata.dataSize)) }}
    </InfoItem>
    <InfoItem v-if="indexSize" :title="$t('database.index-size')">
      {{ indexSize }}
    </InfoItem>
    <InfoItem v-if="collation" :title="$t('db.collation')">
      {{ collation }}
    </InfoItem>
    <InfoItem v-if="comment" :title="$t('database.comment')">
      {{ comment }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { RichEngineName } from "@/components/v2";
import { useDatabaseV1Store, useDBSchemaV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { bytesToString } from "@/utils";
import InfoItem from "./InfoItem.vue";

const props = defineProps<{
  database: string;
  schema?: string;
  table: string;
}>();

const dbSchema = useDBSchemaV1Store();
const databaseStore = useDatabaseV1Store();

const instanceEngine = computed(
  () => databaseStore.getDatabaseByName(props.database).instanceResource.engine
);

const tableMetadata = computed(() =>
  dbSchema.getTableMetadata({
    database: props.database,
    schema: props.schema,
    table: props.table,
  })
);

const indexSize = computed(() => {
  if ([Engine.CLICKHOUSE, Engine.SNOWFLAKE].includes(instanceEngine.value)) {
    return "";
  }
  return bytesToString(Number(tableMetadata.value.indexSize));
});

const collation = computed(() => {
  if (
    [Engine.CLICKHOUSE, Engine.SNOWFLAKE, Engine.POSTGRES].includes(
      instanceEngine.value
    )
  ) {
    return "";
  }
  return tableMetadata.value.collation;
});

const comment = computed(() => {
  return tableMetadata.value.comment;
});
</script>
