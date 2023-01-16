<template>
  <div class="h-full overflow-hidden flex flex-col">
    <div class="flex items-center justify-between p-2 border-b">
      <div class="flex items-center">
        <heroicons-outline:table class="h-4 w-4 mr-1" />
        <span v-if="schema.name" class="font-semibold">{{ schema.name }}.</span>
        <span class="font-semibold">{{ table.name }}</span>
      </div>

      <div class="flex justify-end space-x-2">
        <AlterSchemaButton
          :database="database"
          :schema="schema"
          :table="table"
        />
        <NTooltip trigger="hover">
          <template #trigger>
            <NButton text @click="handleClose">
              <heroicons-outline:x class="w-4 h-4" />
            </NButton>
          </template>
          {{ $t("sql-editor.close-pane") }}
        </NTooltip>
      </div>
    </div>
    <div class="px-2 py-2 border-b text-gray-500 text-xs space-y-1">
      <div class="flex items-center justify-between">
        <span class="mr-1">{{ $t("database.row-count-est") }}</span>
        <span>{{ table.rowCount }}</span>
      </div>
    </div>
    <div class="flex-1 px-2 overflow-y-auto">
      <div class="flex justify-between items-center text-sm text-gray-500 py-1">
        <div>{{ $t("database.columns") }}</div>
        <div>{{ $t("database.data-type") }}</div>
      </div>

      <div
        v-for="(column, index) in table.columns"
        :key="index"
        class="flex justify-between items-center text-xs text-gray-600 py-1"
      >
        <div>{{ column.name }}</div>
        <div class="text-gray-400">{{ column.type }}</div>
      </div>
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

defineProps<{
  database: Database;
  databaseMetadata: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}>();

const emit = defineEmits<{
  (e: "close"): void;
}>();

const handleClose = () => {
  emit("close");
};
</script>
