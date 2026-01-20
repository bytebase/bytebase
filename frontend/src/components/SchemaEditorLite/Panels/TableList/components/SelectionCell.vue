<template>
  <NCheckbox
    :checked="state.checked"
    :indeterminate="state.indeterminate"
    @update:checked="update"
  />
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { useSchemaEditorContext } from "@/components/SchemaEditorLite/context";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";

const props = defineProps<{
  db: Database;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
  };
}>();
const { getTableSelectionState, updateTableSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getTableSelectionState(props.db, props.metadata);
});
const update = (on: boolean) => {
  updateTableSelection(props.db, props.metadata, on);
};
</script>
