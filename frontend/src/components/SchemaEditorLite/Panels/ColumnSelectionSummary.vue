<template>
  <span class="text-control-placeholder">
    ({{ summary.selected }} / {{ summary.total }})
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useSchemaEditorContext } from "@/components/SchemaEditorLite/context";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";

const props = defineProps<{
  db: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
  };
}>();

const { getColumnSelectionState } = useSchemaEditorContext();

const summary = computed(() => {
  const { columns } = props.metadata.table;
  const selectedColumns = columns.filter((column) => {
    const state = getColumnSelectionState(props.db, {
      ...props.metadata,
      column,
    });
    return state.checked || state.indeterminate;
  });
  return {
    selected: selectedColumns.length,
    total: columns.length,
  };
});
</script>
