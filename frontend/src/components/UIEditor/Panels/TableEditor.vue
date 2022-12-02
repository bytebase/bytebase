<template>
  <div class="grid auto-rows-auto w-full h-full overflow-y-auto">
    <div
      class="w-full h-auto flex flex-row justify-between items-center py-2 border-b"
    >
      <div class="flex flex-row items-center space-x-2">
        <div class="flex flex-row justify-start items-center">
          <span class="mr-1 text-sm ml-3 whitespace-nowrap text-gray-500"
            >{{ $t("ui-editor.table.name") }}:
          </span>
          <input
            v-model="tableCache.name"
            placeholder=""
            class="w-full leading-6 px-2 py-1 rounded border border-gray-200 text-sm"
            type="text"
          />
        </div>
      </div>
      <div class="flex flex-row items-center space-x-2">
        <NPopover
          trigger="click"
          placement="bottom-center"
          @update:show="handlePreviewDDLStatement"
        >
          <template #trigger>
            <button
              class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:bg-gray-100"
            >
              {{ $t("ui-editor.actions.sql-preview") }}
            </button>
          </template>
          <div class="w-112 min-h-[16em] max-h-[32em] overflow-y-auto">
            <div
              v-if="state.isFetchingDDL"
              class="w-full h-full flex justify-center items-center"
            >
              <BBSpin />
            </div>
            <template v-else>
              <HighlightCodeBlock
                v-if="state.statement !== ''"
                class="text-sm whitespace-pre-wrap break-all"
                language="sql"
                :code="state.statement"
              ></HighlightCodeBlock>
              <span v-else class="py-2 italic">{{
                $t("ui-editor.nothing-changed")
              }}</span>
            </template>
          </div>
        </NPopover>
        <button
          class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="!allowSave"
          @click="handleDiscardChanges"
        >
          <heroicons-solid:arrow-uturn-left
            class="w-4 h-auto mr-1 text-gray-400"
          />
          {{ $t("ui-editor.actions.discard-changes") }}
        </button>
        <button
          class="flex flex-row bg-accent text-white justify-center items-center px-3 py-1 leading-6 text-sm rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="!allowSave"
          @click="handleSaveChanges"
        >
          <heroicons-outline:save class="w-4 h-auto mr-1" />
          {{ $t("ui-editor.actions.save") }}
        </button>
      </div>
    </div>
    <!-- column table -->
    <div class="w-full py-2 flex flex-row justify-between items-center">
      <span class="ml-3 text-gray-500 font-normal text-sm">{{
        t("ui-editor.columns")
      }}</span>
      <div>
        <button
          class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:bg-gray-100"
          @click="handleAddColumn"
        >
          <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
          {{ $t("ui-editor.actions.add-column") }}
        </button>
      </div>
    </div>
    <div
      class="w-full h-auto grid auto-rows-auto border-y relative overflow-y-auto"
    >
      <!-- column table header -->
      <div
        class="sticky top-0 z-10 grid grid-cols-[repeat(5,_minmax(0,_1fr))_32px] w-full text-sm leading-6 select-none bg-gray-50 text-gray-400"
        :class="tableCache.columnList.length > 0 && 'border-b'"
      >
        <span
          v-for="header in columnHeaderList"
          :key="header.key"
          class="table-header-item-container"
          >{{ header.label }}</span
        >
        <span></span>
      </div>
      <!-- column table body -->
      <div class="w-full">
        <div
          v-for="(column, index) in tableCache.columnList"
          :key="`${index}-${column.id}`"
          class="grid grid-cols-[repeat(5,_minmax(0,_1fr))_32px] gr text-sm even:bg-gray-50"
        >
          <div class="table-body-item-container">
            <input
              v-model="column.name"
              placeholder="column name"
              class="column-field-input"
              type="text"
            />
          </div>
          <div
            class="table-body-item-container flex flex-row justify-between items-center"
          >
            <input
              v-model="column.type"
              placeholder="column type"
              class="column-field-input !pr-8"
              type="text"
            />
            <NDropdown
              trigger="click"
              :options="dataTypeOptions"
              @select="(dataType: string) => (column.type = dataType)"
            >
              <button class="absolute right-5">
                <heroicons-solid:chevron-up-down
                  class="w-4 h-auto text-gray-400"
                />
              </button>
            </NDropdown>
          </div>
          <div
            class="table-body-item-container flex justify-start items-center"
          >
            <BBCheckbox
              class="ml-3"
              :value="column.nullable"
              @toggle="(value) => (column.nullable = value)"
            />
          </div>
          <div class="table-body-item-container">
            <input
              v-model="column.default"
              placeholder="column default value"
              class="column-field-input"
              type="text"
            />
          </div>
          <div class="table-body-item-container">
            <input
              v-model="column.comment"
              placeholder="comment"
              class="column-field-input"
              type="text"
            />
          </div>
          <div class="w-full flex justify-start items-center">
            <n-tooltip trigger="hover">
              <template #trigger>
                <heroicons:trash
                  class="w-[14px] h-auto text-gray-500 cursor-pointer hover:opacity-80"
                  @click="handleRemoveColumn(column)"
                />
              </template>
              <span>Drop Column</span>
            </n-tooltip>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  useNotificationStore,
  useTableStore,
  useUIEditorStore,
} from "@/store/modules";
import { TableTabContext, Column, UNKNOWN_ID, DatabaseEdit } from "@/types";
import { BBCheckbox, BBSpin } from "@/bbkit";
import { getDataTypeSuggestionList } from "@/utils";
import { diffTableList } from "@/utils/UIEditor/diffTable";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";

const columnNameFieldRegexp = /\S+/;
const columnTypeFieldRegexp = /\S+/;

interface LocalState {
  isFetchingDDL: boolean;
  statement: string;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  isFetchingDDL: false,
  statement: "",
});
const editorStore = useUIEditorStore();
const tableStore = useTableStore();
const notificationStore = useNotificationStore();
const currentTab = editorStore.currentTab as TableTabContext;
const table = currentTab.table;
const tableCache = currentTab.tableCache;

const allowSave = computed(() => {
  return !isEqual(tableCache, table);
});

const columnHeaderList = computed(() => {
  return [
    {
      key: "name",
      label: t("ui-editor.column.name"),
    },
    {
      key: "type",
      label: t("ui-editor.column.type"),
    },
    {
      key: "nullable",
      label: t("ui-editor.column.nullable"),
    },
    {
      key: "default",
      label: t("ui-editor.column.default"),
    },
    {
      key: "comment",
      label: t("ui-editor.column.comment"),
    },
  ];
});

const dataTypeOptions = computed(() => {
  return getDataTypeSuggestionList(tableCache.database.instance.engine).map(
    (dataType) => {
      return {
        label: dataType,
        key: dataType,
      };
    }
  );
});

const handleSaveChanges = async () => {
  if (tableCache.name === "") {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Table name cannot be empty",
    });
    return;
  }

  const tableNameList = (
    await editorStore.getOrFetchTableListByDatabaseId(tableCache.database.id)
  )
    .filter((item) => item !== table)
    .map((table) => table.name);
  if (tableNameList.includes(tableCache.name)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Invalid table name: duplicate with others",
    });
    return;
  }

  for (const column of tableCache.columnList) {
    if (!columnNameFieldRegexp.test(column.name)) {
      notificationStore.pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Invalid column name",
      });
      return;
    }
    if (!columnTypeFieldRegexp.test(column.type)) {
      notificationStore.pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Invalid column type",
      });
      return;
    }
  }

  editorStore.saveTab(currentTab);
};

const handleAddColumn = () => {
  tableCache.columnList.push({
    id: UNKNOWN_ID,
    name: "",
    type: "",
    nullable: false,
    comment: "",
  } as Column);
};

const handleRemoveColumn = (column: Column) => {
  tableCache.columnList = tableCache.columnList.filter(
    (item) => item !== column
  );
};

const handleDiscardChanges = () => {
  tableCache.name = table.name;
  tableCache.columnList = cloneDeep(table.columnList);
};

const handlePreviewDDLStatement = async (show: boolean) => {
  if (!show) {
    return;
  }

  const databaseEdit: DatabaseEdit = {
    databaseId: table.database.id,
    createTableList: [],
    alterTableList: [],
    renameTableList: [],
    dropTableList: [],
  };
  if (table.id === UNKNOWN_ID) {
    const diffTableListResult = diffTableList([], [table]);
    databaseEdit.createTableList = diffTableListResult.createTableList;
  } else {
    const originTable = tableStore.getTableByDatabaseIdAndTableId(
      table.database.id,
      table.id
    );
    const isDropped = editorStore.droppedTableList.includes(table);
    if (isDropped) {
      const diffTableListResult = diffTableList([originTable], []);
      databaseEdit.dropTableList = diffTableListResult.dropTableList;
    } else {
      const diffTableListResult = diffTableList([originTable], [table]);
      databaseEdit.alterTableList = diffTableListResult.alterTableList;
      databaseEdit.renameTableList = diffTableListResult.renameTableList;
    }
  }
  state.isFetchingDDL = true;
  try {
    const statement = await editorStore.postDatabaseEdit(databaseEdit);
    state.statement = statement;
  } catch (error) {
    state.statement = "";
  }
  state.isFetchingDDL = false;
};
</script>

<style scoped>
.table-header-item-container {
  @apply py-2 px-3;
}
.table-body-item-container {
  @apply w-full h-10 box-border p-px pr-2 relative;
}
.column-field-input {
  @apply w-full pr-1 box-border border-transparent text-ellipsis rounded bg-transparent text-sm placeholder:italic placeholder:text-gray-400 focus:bg-white focus:text-black;
}
</style>
