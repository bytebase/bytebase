<template>
  <div class="min-w-[14rem] max-w-[18rem] gap-y-1">
    <InfoItem :title="$t('common.name')">
      {{ table.name }}
    </InfoItem>
    <InfoItem :title="$t('database.engine')">
      <RichEngineName :engine="instanceEngine" />
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
    <InfoItem v-if="comment" :title="$t('database.comment')">
      {{ comment }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { RichEngineName } from "@/components/v2";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { TableMetadata } from "@/types/proto/v1/database_service";
import { bytesToString } from "@/utils";
import InfoItem from "./InfoItem.vue";

const props = defineProps<{
  db: ComposedDatabase;
  table: TableMetadata;
}>();

const instanceEngine = computed(() => props.db.instanceResource.engine);

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

const comment = computed(() => {
  return props.table.userComment;
});
</script>
