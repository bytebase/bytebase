<template>
  <div class="grid auto-rows-auto w-full h-full overflow-y-auto">
    <div
      class="pt-3 pl-1 w-full flex justify-start items-center border-b border-b-gray-300"
    >
      <span
        class="-mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none"
        :class="
          state.selectedTab === 'column-list' &&
          'bg-white border-gray-300 text-gray-800'
        "
        @click="handleChangeTab('column-list')"
        >{{ $t("ui-editor.column-list") }}</span
      >
      <span
        class="-mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none"
        :class="
          state.selectedTab === 'raw-sql' &&
          'bg-white border-gray-300 text-gray-800'
        "
        @click="handleChangeTab('raw-sql')"
        >{{ $t("ui-editor.raw-sql") }}</span
      >
    </div>

    <template v-if="state.selectedTab === 'column-list'">
      <!-- column table -->
      <div class="w-full py-2 flex flex-row justify-between items-center">
        <div>
          <button
            class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:opacity-80"
            @click="handleAddColumn"
          >
            <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
            {{ $t("ui-editor.actions.add-column") }}
          </button>
        </div>
        <div>
          <button
            v-if="table.id !== UNKNOWN_ID"
            class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="allowResetTable"
            @click="handleDiscardChanges"
          >
            <heroicons-solid:arrow-uturn-left
              class="w-4 h-auto mr-1 text-gray-400"
            />
            {{ $t("ui-editor.actions.reset") }}
          </button>
        </div>
      </div>
      <div
        class="w-full h-auto grid auto-rows-auto border-y relative overflow-y-auto"
      >
        <!-- column table header -->
        <div
          class="sticky top-0 z-10 grid grid-cols-[repeat(4,_minmax(0,_1fr))_112px_32px] w-full text-sm leading-6 select-none bg-gray-50 text-gray-400"
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
            class="grid grid-cols-[repeat(4,_minmax(0,_1fr))_112px_32px] gr text-sm even:bg-gray-50"
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
            <div
              class="table-body-item-container flex justify-start items-center"
            >
              <BBCheckbox
                class="ml-3"
                :value="column.nullable"
                @toggle="(value) => (column.nullable = value)"
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
    </template>
    <div
      v-else-if="state.selectedTab === 'raw-sql'"
      class="w-full h-full overflow-y-auto"
    >
      <div
        v-if="state.isFetchingDDL"
        class="w-full h-full min-h-[64px] flex justify-center items-center"
      >
        <BBSpin />
      </div>
      <template v-else>
        <HighlightCodeBlock
          v-if="state.statement !== ''"
          class="text-sm px-3 py-2 whitespace-pre-wrap break-all"
          language="sql"
          :code="state.statement"
        ></HighlightCodeBlock>
        <div v-else class="flex px-3 py-2 italic text-sm text-gray-600">
          {{ $t("ui-editor.nothing-changed") }}
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useDebounceFn } from "@vueuse/core";
import { useTableStore, useUIEditorStore } from "@/store/modules";
import { TableTabContext, Column, UNKNOWN_ID, DatabaseEdit } from "@/types";
import { BBCheckbox, BBSpin } from "@/bbkit";
import { getDataTypeSuggestionList } from "@/utils";
import { diffTableList } from "@/utils/UIEditor/diffTable";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";

type TabType = "column-list" | "raw-sql";

interface LocalState {
  selectedTab: TabType;
  isFetchingDDL: boolean;
  statement: string;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedTab: "column-list",
  isFetchingDDL: false,
  statement: "",
});
const editorStore = useUIEditorStore();
const tableStore = useTableStore();
const currentTab = editorStore.currentTab as TableTabContext;
const table = currentTab.table;
const tableCache = currentTab.tableCache;

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
      key: "default",
      label: t("ui-editor.column.default"),
    },
    {
      key: "comment",
      label: t("ui-editor.column.comment"),
    },
    {
      key: "nullable",
      label: t("ui-editor.column.is-nullable"),
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

const allowResetTable = computed(() => {
  const originTable = tableStore.getTableByDatabaseIdAndTableId(
    currentTab.databaseId,
    currentTab.tableId
  );

  return isEqual(table, originTable);
});

watch([tableCache], () => {
  handleSaveChanges();
});

watch(
  () => state.selectedTab,
  async () => {
    if (state.selectedTab === "raw-sql") {
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
    }
  }
);

const handleChangeTab = (tab: TabType) => {
  state.selectedTab = tab;
};

const handleSaveChanges = useDebounceFn(async () => {
  editorStore.saveTab(currentTab);
}, 500);

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
  editorStore.discardTabChanges(currentTab);
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
