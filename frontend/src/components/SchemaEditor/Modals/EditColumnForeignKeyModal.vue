<template>
  <BBModal
    :title="$t('schema-editor.edit-foreign-key')"
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <p class="mb-2">{{ $t("schema-editor.select-reference-table") }}</p>
      <BBSelect
        :selected-item="selectedTable"
        :item-list="tableList"
        :placeholder="$t('schema-editor.table.select')"
        :show-prefix-item="true"
        @select-item="(table: Table) => state.referencedTable = table.newName"
      >
        <template #menuItem="{ item: table }">
          {{ table.newName }}
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
        @select-item="(column: Column) => state.referencedColumn = column.newName"
      >
        <template #menuItem="{ item }">
          {{ item.newName }}
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
import { computed, onMounted, PropType, reactive, ref, watch } from "vue";
import { DatabaseId, UNKNOWN_ID } from "@/types";
import { useSchemaEditorStore } from "@/store";
import { Column, Schema, Table } from "@/types/schemaEditor/atomType";

interface LocalState {
  referencedSchema?: string;
  referencedTable?: string;
  referencedColumn?: string;
}

const props = defineProps({
  databaseId: {
    type: Number as PropType<DatabaseId>,
    default: UNKNOWN_ID,
  },
  schemaName: {
    type: String as PropType<string>,
    default: "",
  },
  tableName: {
    type: String as PropType<string>,
    default: "",
  },
  column: {
    required: true,
    type: Object as PropType<Column>,
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const editorStore = useSchemaEditorStore();
const state = reactive<LocalState>({
  referencedSchema: props.schemaName,
});

const schema = computed(() => {
  return editorStore.getSchema(props.databaseId, props.schemaName) as Schema;
});

const foreignKeyList = computed(() => {
  return schema.value?.foreignKeyList.filter(
    (pk) => pk.table === props.tableName
  );
});

const tableList = computed(() => {
  return (
    editorStore.getSchema(props.databaseId, state.referencedSchema || "")
      ?.tableList || []
  ).filter((table: Table) => table.newName !== props.tableName);
});

const selectedTable = computed(() => {
  return tableList.value.find(
    (table) => table.newName === state.referencedTable
  );
});

const columnList = computed(() => {
  return (
    tableList.value
      .find((item) => item.newName === state.referencedTable)
      ?.columnList.filter(
        (column) =>
          column.type.toUpperCase() === props.column.type.toUpperCase()
      ) || []
  );
});

const selectedColumn = computed(() => {
  return columnList.value.find(
    (column) => column.newName === state.referencedColumn
  );
});

const foreignKey = computed(() => {
  for (const fk of foreignKeyList.value) {
    const foundIndex = fk.columnList.findIndex(
      (columnRef) => columnRef.value === props.column
    );
    if (foundIndex > -1) {
      return fk;
    }
  }
  return undefined;
});

onMounted(() => {
  if (foreignKey.value) {
    const foundIndex = foreignKey.value.columnList.findIndex(
      (columnRef) => columnRef.value === props.column
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
      (column) => column.newName === state.referencedColumn
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

  const column = schema.value.tableList
    .find((table) => table.newName === props.tableName)
    ?.columnList.find((column) => column === props.column) as Column;
  const index = foreignKey.value.columnList.findIndex(
    (columnRef) => columnRef.value === column
  );
  if (index > -1) {
    foreignKey.value.referencedColumns.splice(index, 1);
    foreignKey.value.columnList.splice(index, 1);
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

  const column = schema.value.tableList
    .find((table) => table.newName === props.tableName)
    ?.columnList.find((column) => column === props.column) as Column;

  if (isUndefined(foreignKey.value)) {
    const fk = {
      name: "",
      table: props.tableName,
      columnList: [ref(column)],
      referencedSchema: state.referencedSchema,
      referencedTable: state.referencedTable,
      referencedColumns: [state.referencedColumn],
    };
    schema.value.foreignKeyList.push(fk);
  } else {
    foreignKey.value.referencedColumns.push(state.referencedColumn);
    foreignKey.value.columnList.push(ref(column));
  }
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
