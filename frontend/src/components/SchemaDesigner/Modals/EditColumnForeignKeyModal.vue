<template>
  <BBModal
    :title="$t('schema-editor.edit-foreign-key')"
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div v-if="shouldShowSchemaSelector" class="w-72">
      <p class="mb-2">{{ $t("schema-editor.select-reference-schema") }}</p>
      <BBSelect
        :selected-item="selectedSchema"
        :item-list="schemas"
        :placeholder="$t('schema-editor.schema.select')"
        :show-prefix-item="true"
        @select-item="(schema: SchemaMetadata) => state.referencedSchema = schema.name"
      >
        <template #menuItem="{ item }">
          {{ item.name }}
        </template>
      </BBSelect>
    </div>
    <div class="w-72">
      <p class="mt-4 mb-2">{{ $t("schema-editor.select-reference-table") }}</p>
      <BBSelect
        :selected-item="selectedTable"
        :item-list="tableList"
        :placeholder="$t('schema-editor.table.select')"
        :show-prefix-item="true"
        @select-item="(table: TableMetadata) => state.referencedTable = table.name"
      >
        <template #menuItem="{ item }">
          {{ item.name }}
        </template>
      </BBSelect>
    </div>
    <div class="w-72">
      <p class="mt-4 mb-2">{{ $t("schema-editor.select-reference-column") }}</p>
      <BBSelect
        :selected-item="selectedColumn"
        :item-list="columnList"
        :placeholder="$t('schema-editor.column.select')"
        :show-prefix-item="true"
        @select-item="(column: ColumnMetadata) => state.referencedColumn = column.name"
      >
        <template #menuItem="{ item }">
          {{ item.name }}
        </template>
      </BBSelect>
    </div>
    <div
      class="w-full flex items-center justify-between mt-6 space-x-2 pr-1 pb-1"
    >
      <div class="flex items-center justify-start">
        <button
          v-if="foreignKey !== undefined"
          type="button"
          class="btn-danger"
          @click="handleRemoveFKButtonClick"
        >
          Remove
        </button>
      </div>
      <div class="flex items-center justify-end space-x-2">
        <button type="button" class="btn-normal" @click="dismissModal">
          {{ $t("common.cancel") }}
        </button>
        <button class="btn-primary" @click="handleConfirmButtonClick">
          {{ $t("common.save") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { isUndefined } from "lodash-es";
import { computed, onMounted, reactive, watch } from "vue";
import { BBModal, BBSelect } from "@/bbkit";
import { Engine } from "@/types/proto/v1/common";
import { useSchemaDesignerContext } from "../common";
import {
  ColumnMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

interface LocalState {
  referencedSchema?: string;
  referencedTable?: string;
  referencedColumn?: string;
}

const props = defineProps({
  schema: {
    type: String,
    default: "",
  },
  table: {
    type: String,
    default: "",
  },
  column: {
    type: String,
    default: "",
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { engine, metadata } = useSchemaDesignerContext();
const schemas = computed(() => metadata.value.schemas);
const state = reactive<LocalState>({
  referencedSchema: props.schema,
});

const schema = computed(() => {
  return schemas.value.find((item) => item.name === props.schema);
});

const table = computed(() => {
  return schema.value?.tables.find((item) => item.name === props.table);
});

const propsColumn = computed(() => {
  return table.value?.columns.find((item) => item.name === props.column);
});

const foreignKeyList = computed(() => {
  return table.value?.foreignKeys || [];
});

const shouldShowSchemaSelector = computed(() => {
  return engine.value === Engine.POSTGRES;
});

const selectedSchema = computed(() => {
  return schemas.value.find((schema) => schema.name === state.referencedSchema);
});

const tableList = computed(() => {
  return selectedSchema.value?.tables || [];
});

const selectedTable = computed(() => {
  return tableList.value.find(
    (table: TableMetadata) => table.name === state.referencedTable
  );
});

const columnList = computed(() => {
  if (!selectedTable.value) {
    return [];
  }

  return selectedTable.value.columns.filter(
    (column) =>
      column.name !== props.column &&
      column.type.toUpperCase() === propsColumn.value?.type.toUpperCase()
  );
});

const selectedColumn = computed(() => {
  return columnList.value.find(
    (column) => column.name === state.referencedColumn
  );
});

const foreignKey = computed(() => {
  for (const fk of foreignKeyList.value) {
    const foundIndex = fk.columns.findIndex(
      (column) => column === props.column
    );
    if (foundIndex > -1) {
      return fk;
    }
  }
  return undefined;
});

onMounted(() => {
  if (foreignKey.value) {
    const foundIndex = foreignKey.value.columns.findIndex(
      (column) => column === props.column
    );
    if (foundIndex > -1) {
      state.referencedSchema = foreignKey.value.referencedSchema;
      state.referencedTable = foreignKey.value.referencedTable;
      state.referencedColumn = foreignKey.value.referencedColumns[foundIndex];
    }
  }
});

watch(
  () => state.referencedTable,
  () => {
    const found = columnList.value.find(
      (column) => column.name === state.referencedColumn
    );
    if (!found) {
      state.referencedColumn = undefined;
    }
  }
);

const handleRemoveFKButtonClick = async () => {
  if (isUndefined(foreignKey.value)) {
    return;
  }

  const index = foreignKey.value.columns.findIndex(
    (column) => column === propsColumn.value?.name
  );
  if (index > -1) {
    foreignKey.value.referencedColumns.splice(index, 1);
    foreignKey.value.columns.splice(index, 1);
    dismissModal();
  }
};

const handleConfirmButtonClick = async () => {
  if (
    isUndefined(state.referencedSchema) ||
    isUndefined(state.referencedTable) ||
    isUndefined(state.referencedColumn)
  ) {
    return;
  }

  const column = propsColumn.value as ColumnMetadata;
  if (isUndefined(foreignKey.value)) {
    table.value?.foreignKeys.push(
      ForeignKeyMetadata.fromPartial({
        columns: [column.name],
        referencedSchema: state.referencedSchema,
        referencedTable: state.referencedTable,
        referencedColumns: [state.referencedColumn],
      })
    );
  } else {
    const index = foreignKey.value.columns.findIndex(
      (item) => item === column.name
    );
    if (index >= 0) {
      foreignKey.value.referencedColumns[index] = state.referencedColumn;
      foreignKey.value.columns[index] = column.name;
    } else {
      foreignKey.value.referencedColumns.push(state.referencedColumn);
      foreignKey.value.columns.push(column.name);
    }
  }
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
