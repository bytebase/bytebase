<template>
  <div class="min-w-[14rem] max-w-[18rem] gap-y-1">
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
      <div class="inline-flex items-center justify-end">
        <CheckIcon v-if="column.nullable" class="w-4 h-4" />
        <XIcon v-else class="w-4 h-4" />
      </div>
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
import { CheckIcon, XIcon } from "lucide-vue-next";
import { computed } from "vue";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorLite";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import InfoItem from "./InfoItem.vue";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  column: ColumnMetadata;
}>();

const instanceEngine = computed(() => {
  return props.db.instanceResource.engine;
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
