<template>
  <div
    v-for="fk in foreignKeys"
    :key="fk.name"
    class="flex flex-row flex-wrap items-center break-all"
  >
    <span @click="$emit('click', fk)">
      {{ referencedNameForFk(fk) }}
    </span>

    <MiniActionButton
      v-if="!readonly"
      :disabled="disabled"
      @click="$emit('edit', fk)"
    >
      <PenSquareIcon class="w-4 h-4" />
    </MiniActionButton>
  </div>

  <div
    v-if="foreignKeys.length === 0"
    class="flex flex-row flex-wrap items-center break-all"
  >
    <span class="italic text-control-placeholder">EMPTY</span>
    <MiniActionButton
      v-if="!readonly"
      :disabled="disabled"
      @click="$emit('edit', undefined)"
    >
      <PenSquareIcon class="w-4 h-4" />
    </MiniActionButton>
  </div>
</template>

<script lang="ts" setup>
import { PenSquareIcon } from "lucide-vue-next";
import { computed } from "vue";
import { engineHasSchema } from "@/components/SchemaEditorLite/engine-specs";
import { MiniActionButton } from "@/components/v2";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
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
  disabled: boolean;
}>();
defineEmits<{
  (event: "click", fk: ForeignKeyMetadata): void;
  (event: "edit", fk: ForeignKeyMetadata | undefined): void;
}>();

const foreignKeys = computed(() => {
  return props.table.foreignKeys.filter((fk) =>
    fk.columns.includes(props.column.name)
  );
});

const referencedNameForFk = (fk: ForeignKeyMetadata) => {
  const position = fk.columns.indexOf(props.column.name);
  if (position < 0) {
    return "";
  }

  const referencedSchema = props.database.schemas.find(
    (schema) => schema.name === fk.referencedSchema
  );
  if (!referencedSchema) {
    return "";
  }

  const referencedTable = referencedSchema.tables.find(
    (table) => table.name === fk.referencedTable
  );
  if (!referencedTable) {
    return "";
  }

  const referencedColumn = referencedTable.columns.find(
    (column) => column.name === fk.referencedColumns[position]
  );

  if (engineHasSchema(props.db.instanceEntity.engine)) {
    return `${referencedSchema.name}.${referencedTable.name}(${referencedColumn?.name})`;
  } else {
    return `${referencedTable.name}(${referencedColumn?.name})`;
  }
};
</script>
