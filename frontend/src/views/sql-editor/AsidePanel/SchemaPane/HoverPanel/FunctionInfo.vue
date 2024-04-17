<template>
  <CodeViewer
    :db="db"
    :database="database"
    :schema="schema"
    :name="$props.function.name"
    :code="$props.function.definition"
  />
</template>

<script setup lang="ts">
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  FunctionMetadata,
} from "@/types/proto/v1/database_service";
import CodeViewer from "./CodeViewer.vue";

defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  function: FunctionMetadata;
}>();
</script>
