<template>
  <div class="overflow-hidden flex flex-col">
    <div class="flex items-center justify-between pl-4 pr-2 py-1 border-b">
      <div
        class="flex items-center flex-1 truncate cursor-pointer"
        @click="emit('close')"
      >
        <heroicons-outline:table class="h-4 w-4 mr-1 flex-shrink-0" />
        <span v-if="schema.name" class="text-sm">{{ schema.name }}.</span>
        <span class="text-sm">{{ externalTable.name }}</span>
      </div>
    </div>

    <ColumnList
      :db="db"
      :database="database"
      :schema="schema"
      :columns="externalTable.columns"
      class="w-full flex-1 py-1"
    />
  </div>
</template>

<script lang="ts" setup>
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  ExternalTableMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import ColumnList from "./ColumnList.vue";

defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  externalTable: ExternalTableMetadata;
}>();

const emit = defineEmits<{
  (e: "close"): void;
}>();
</script>
