<template>
  <span class="text-control-placeholder">
    ({{ summary.selected }} / {{ summary.total }})
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";

const props = defineProps<{
  db: Database;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
  };
}>();

const { getTableSelectionState } = useSchemaEditorContext();

const summary = computed(() => {
  const { tables } = props.metadata.schema;
  const selectedTables = tables.filter((table) => {
    const state = getTableSelectionState(props.db, {
      ...props.metadata,
      table,
    });
    return state.checked || state.indeterminate;
  });
  return {
    selected: selectedTables.length,
    total: tables.length,
  };
});
</script>
