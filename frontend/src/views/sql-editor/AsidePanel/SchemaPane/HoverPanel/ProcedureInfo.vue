<template>
  <CodeViewer
    :db="db"
    :database="database"
    :schema="schema"
    :name="procedure.name"
    :code="procedure.definition"
  />
</template>

<script setup lang="ts">
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  ProcedureMetadata,
} from "@/types/proto/v1/database_service";
import CodeViewer from "./CodeViewer.vue";

defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedure: ProcedureMetadata;
}>();
</script>
