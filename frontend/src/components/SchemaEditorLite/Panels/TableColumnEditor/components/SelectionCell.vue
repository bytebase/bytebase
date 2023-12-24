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
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

const props = defineProps<{
  db: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
    column: ColumnMetadata;
  };
}>();
const { getColumnSelectionState, updateColumnSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getColumnSelectionState(props.db, props.metadata);
});
const update = (on: boolean) => {
  updateColumnSelection(props.db, props.metadata, on);
};
</script>
