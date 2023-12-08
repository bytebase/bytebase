<template>
  <div class="flex justify-start items-center">
    <LabelsColumn :labels="columnConfig?.labels ?? {}" :show-count="1" />
    <MiniActionButton v-if="!readonly && !disabled" @click="$emit('edit')">
      <PencilIcon class="w-3 h-3" />
    </MiniActionButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useSchemaEditorContext } from "@/components/SchemaEditorLite/context";
import { MiniActionButton } from "@/components/v2";
import LabelsColumn from "@/components/v2/Model/DatabaseV1Table/LabelsColumn.vue";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  column: ColumnMetadata;
  readonly?: boolean;
  disabled?: boolean;
}>();
defineEmits<{
  (event: "edit"): void;
}>();

const { getColumnConfig } = useSchemaEditorContext();

const columnConfig = computed(() => {
  return getColumnConfig(props.db, {
    database: props.database,
    schema: props.schema,
    table: props.table,
    column: props.column,
  });
});
</script>
