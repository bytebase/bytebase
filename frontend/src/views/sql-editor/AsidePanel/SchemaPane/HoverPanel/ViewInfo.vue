<template>
  <CodeViewer
    :db="db"
    :database="database"
    :schema="schema"
    :name="view.name"
    :code="view.definition"
  />
</template>

<script setup lang="ts">
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";
import CodeViewer from "./CodeViewer.vue";

defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
}>();
</script>
