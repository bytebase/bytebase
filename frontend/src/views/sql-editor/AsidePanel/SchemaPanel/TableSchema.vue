<template>
  <div class="h-full overflow-hidden flex flex-col">
    <div class="flex items-center justify-between p-2 border-b">
      <div class="flex items-center truncate">
        <heroicons-outline:table class="h-4 w-4 mr-1" />
        <a
          :href="tableDetailLink"
          target="__BLANK"
          class="font-semibold anchor-link"
        >
          <span v-if="schema.name">{{ schema.name }}.</span>
          <span>{{ table.name }}</span>
        </a>
      </div>

      <div class="flex justify-end">
        <AlterSchemaButton
          :database="database"
          :schema="schema"
          :table="table"
        />

        <NButton quaternary size="tiny" @click="handleClose">
          <heroicons-outline:x class="w-4 h-4" />
        </NButton>
      </div>
    </div>

    <div class="px-2 py-2 border-b text-gray-500 text-xs space-y-1">
      <div class="flex items-center justify-between">
        <span class="mr-1">{{ $t("database.row-count-est") }}</span>
        <span>{{ table.rowCount }}</span>
      </div>
    </div>

    <div
      class="grid px-2 overflow-y-auto gap-x-1 gap-y-2"
      style="grid-template-columns: minmax(4rem, 2fr) minmax(4rem, 1fr)"
    >
      <div class="mt-2 text-sm text-gray-500">{{ $t("database.columns") }}</div>
      <div class="mt-2 text-right text-sm text-gray-500">
        {{ $t("database.data-type") }}
      </div>

      <template v-for="(column, index) in table.columns" :key="index">
        <div class="text-xs text-gray-600whitespace-pre-wrap break-words">
          {{ column.name }}
        </div>
        <div
          class="text-right text-xs text-gray-400 overflow-x-hidden whitespace-nowrap"
        >
          {{ column.type }}
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";

import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import type { Database } from "@/types";
import AlterSchemaButton from "./AlterSchemaButton.vue";
import { computed } from "vue";
import { databaseSlug } from "@/utils";

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

const handleClose = () => {
  emit("close");
};
</script>
