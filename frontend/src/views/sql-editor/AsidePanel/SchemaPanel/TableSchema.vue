<template>
  <div class="overflow-hidden flex flex-col">
    <div class="flex items-center justify-between p-2 pl-4 border-b">
      <div
        class="flex items-center flex-1 truncate cursor-pointer"
        @click="emit('close')"
      >
        <heroicons-outline:table class="h-4 w-4 mr-1 flex-shrink-0" />
        <span v-if="schema.name" class="font-semibold">{{ schema.name }}.</span>
        <span class="font-semibold">{{ table.name }}</span>
      </div>

      <div class="flex justify-end gap-x-0.5">
        <ExternalLinkButton
          :link="tableDetailLink"
          :tooltip="$t('common.detail')"
        />
        <AlterSchemaButton
          :database="database"
          :schema="schema"
          :table="table"
        />
      </div>
    </div>

    <div
      class="grid py-1 pl-4 pr-4 overflow-y-auto gap-x-1 gap-y-2"
      style="grid-template-columns: minmax(4rem, 2fr) minmax(4rem, 1fr)"
    >
      <template v-for="(column, index) in table.columns" :key="index">
        <div class="text-sm text-gray-600whitespace-pre-wrap break-words">
          {{ column.name }}
        </div>
        <div
          class="text-right text-sm text-gray-400 overflow-x-hidden whitespace-nowrap"
        >
          {{ column.type }}
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import type { Database } from "@/types";
import AlterSchemaButton from "./AlterSchemaButton.vue";
import { computed } from "vue";
import { databaseSlug } from "@/utils";
import ExternalLinkButton from "./ExternalLinkButton.vue";

const props = defineProps<{
  database: Database;
  databaseMetadata: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}>();

const emit = defineEmits<{
  (e: "close"): void;
}>();

const tableDetailLink = computed((): string => {
  const { database, schema, table } = props;
  let url = `/db/${databaseSlug(database)}/table/${encodeURIComponent(
    table.name
  )}`;
  if (schema.name) {
    url += `?schema=${encodeURIComponent(schema.name)}`;
  }

  return url;
});
</script>
