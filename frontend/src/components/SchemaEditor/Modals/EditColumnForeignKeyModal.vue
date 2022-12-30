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
        @select-item="(table: Table) => state.referencedTableId = table.id"
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
        @select-item="(column: Column) => state.referencedColumnId = column.id"
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
import { computed, onMounted, PropType, reactive, watch } from "vue";
import { DatabaseId, UNKNOWN_ID } from "@/types";
import { useSchemaEditorStore } from "@/store";
import {
  Column,
  ForeignKey,
  Schema,
  Table,
} from "@/types/schemaEditor/atomType";

interface LocalState {
  referencedSchema?: string;
  referencedTableId?: string;
  referencedColumnId?: string;
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
  tableId: {
    type: String as PropType<string>,
    default: "",
  },
  columnId: {
    type: String as PropType<string>,
    default: "",
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

const table = computed(() => {
  return editorStore.getTable(
    props.databaseId,
    props.schemaName,
    props.tableId
  );
});

const propsColumn = computed(() => {
  return table.value?.columnList.find((column) => column.id === props.columnId);
});

const foreignKeyList = computed(() => {
  return schema.value?.foreignKeyList.filter(
    (pk) => pk.tableId === props.tableId
  );
});

const tableList = computed(() => {
  return (
    editorStore.getSchema(props.databaseId, state.referencedSchema || "")
      ?.tableList || []
  ).filter((table: Table) => table.id !== props.tableId);
});

const selectedTable = computed(() => {
  return tableList.value.find((table) => table.id === state.referencedTableId);
});

const columnList = computed(() => {
  return (
    tableList.value
      .find((table) => table.id === state.referencedTableId)
      ?.columnList.filter(
        (column) =>
          column.type.toUpperCase() === propsColumn.value?.type.toUpperCase()
      ) || []
  );
});

const selectedColumn = computed(() => {
  return columnList.value.find(
    (column) => column.id === state.referencedColumnId
  );
});

const foreignKey = computed(() => {
  for (const fk of foreignKeyList.value) {
    const foundIndex = fk.columnIdList.findIndex(
      (columnId) => columnId === props.columnId
    );
    if (foundIndex > -1) {
      return fk;
    }
  }
  return undefined;
});

onMounted(() => {
  if (foreignKey.value) {
    const foundIndex = foreignKey.value.columnIdList.findIndex(
      (columnId) => columnId === props.columnId
    );
    if (foundIndex > -1) {
      state.referencedSchema = foreignKey.value.referencedSchema;
      state.referencedTableId = foreignKey.value.referencedTableId;
      state.referencedColumnId =
        foreignKey.value.referencedColumnIdList[foundIndex];
    }
  }
});

watch(
  () => state.referencedTableId,
  () => {
    const found = columnList.value.find(
      (column) => column.id === state.referencedColumnId
    );
    if (!found) {
      state.referencedColumnId = undefined;
    }
  }
);

const handleRemoveFKButtonClick = async () => {
  if (isUndefined(foreignKey.value)) {
    return;
  }

  const index = foreignKey.value.columnIdList.findIndex(
    (columnId) => columnId === propsColumn.value?.id
  );
  if (index > -1) {
    foreignKey.value.referencedColumnIdList.splice(index, 1);
    foreignKey.value.columnIdList.splice(index, 1);
    dismissModal();
  }
};

const handleConfirmButtonClick = async () => {
  if (
    isUndefined(state.referencedSchema) ||
    isUndefined(state.referencedTableId) ||
    isUndefined(state.referencedColumnId)
  ) {
    return;
  }

  const column = propsColumn.value as Column;
  if (isUndefined(foreignKey.value)) {
    const fk: ForeignKey = {
      // TODO(steven): it's a constraint name and should be unique.
      name: "",
      tableId: props.tableId,
      columnIdList: [column.id],
      referencedSchema: state.referencedSchema,
      referencedTableId: state.referencedTableId,
      referencedColumnIdList: [state.referencedColumnId],
    };
    schema.value.foreignKeyList.push(fk);
  } else {
    const index = foreignKey.value.columnIdList.findIndex(
      (columnId) => columnId === column.id
    );
    if (index >= 0) {
      foreignKey.value.referencedColumnIdList[index] = state.referencedColumnId;
      foreignKey.value.columnIdList[index] = column.id;
    } else {
      foreignKey.value.referencedColumnIdList.push(state.referencedColumnId);
      foreignKey.value.columnIdList.push(column.id);
    }
  }
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
