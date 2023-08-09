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
        :item-list="schemaList"
        :placeholder="$t('schema-editor.schema.select')"
        :show-prefix-item="true"
        @select-item="(schema: Schema) => state.referencedSchemaId = schema.id"
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
import { v1 as uuidv1 } from "uuid";
import { computed, onMounted, reactive, watch } from "vue";
import { BBModal, BBSelect } from "@/bbkit";
import { useSchemaEditorStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  Column,
  ForeignKey,
  Schema,
  Table,
} from "@/types/schemaEditor/atomType";

interface LocalState {
  referencedSchemaId?: string;
  referencedTableId?: string;
  referencedColumnId?: string;
}

const props = defineProps({
  databaseId: {
    type: String,
    default: String(UNKNOWN_ID),
  },
  schemaId: {
    type: String,
    default: "",
  },
  tableId: {
    type: String,
    default: "",
  },
  columnId: {
    type: String,
    default: "",
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const editorStore = useSchemaEditorStore();
const databaseSchema = computed(() => {
  return editorStore.databaseSchemaById.get(props.databaseId)!;
});
const database = computed(() => databaseSchema.value.database);
const databaseEngine = computed(() => database.value.instanceEntity.engine);
const schemaList = computed(() => databaseSchema.value.schemaList);
const state = reactive<LocalState>({
  referencedSchemaId: props.schemaId,
});

const schema = computed(() => {
  return editorStore.getSchema(props.databaseId, props.schemaId) as Schema;
});

const table = computed(() => {
  return editorStore.getTable(props.databaseId, props.schemaId, props.tableId);
});

const propsColumn = computed(() => {
  return table.value?.columnList.find((column) => column.id === props.columnId);
});

const foreignKeyList = computed(() => {
  return schema.value?.foreignKeyList.filter(
    (pk) => pk.tableId === props.tableId
  );
});

const shouldShowSchemaSelector = computed(() => {
  return databaseEngine.value === Engine.POSTGRES;
});

const selectedSchema = computed(() => {
  return schemaList.value.find(
    (schema) => schema.id === state.referencedSchemaId
  );
});

const tableList = computed(() => {
  return selectedSchema.value?.tableList || [];
});

const selectedTable = computed(() => {
  return tableList.value.find(
    (table: Table) => table.id === state.referencedTableId
  );
});

const columnList = computed(() => {
  if (!selectedTable.value) {
    return [];
  }

  return selectedTable.value.columnList.filter(
    (column) =>
      column.id !== props.columnId &&
      column.type.toUpperCase() === propsColumn.value?.type.toUpperCase() &&
      selectedTable.value?.primaryKey.columnIdList.includes(column.id)
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
      state.referencedSchemaId = foreignKey.value.referencedSchemaId;
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
    isUndefined(state.referencedSchemaId) ||
    isUndefined(state.referencedTableId) ||
    isUndefined(state.referencedColumnId)
  ) {
    return;
  }

  const column = propsColumn.value as Column;
  if (isUndefined(foreignKey.value)) {
    const fk: ForeignKey = {
      name: uuidv1(),
      tableId: props.tableId,
      columnIdList: [column.id],
      referencedSchemaId: state.referencedSchemaId,
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
