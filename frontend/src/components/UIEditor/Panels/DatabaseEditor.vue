<template>
  <div class="grid auto-rows-auto w-full h-full overflow-y-auto">
    <div
      class="pt-3 pl-1 w-full flex justify-start items-center border-b border-b-gray-300"
    >
      <span
        class="-mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none"
        :class="
          state.selectedTab === 'table-list' &&
          'bg-white border-gray-300 text-gray-800'
        "
        @click="handleChangeTab('table-list')"
        >{{ $t("ui-editor.table-list") }}</span
      >
      <span
        class="hidden -mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none"
        :class="
          state.selectedTab === 'er-diagram' &&
          'bg-white border-gray-300 text-gray-800'
        "
        @click="handleChangeTab('er-diagram')"
        >ER Diagram</span
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

    <!-- List view -->
    <template v-if="state.selectedTab === 'table-list'">
      <div class="py-2 w-full flex justify-between items-center space-x-2">
        <button
          class="flex flex-row justify-center items-center border px-3 py-1 leading-6 rounded text-sm hover:bg-gray-100"
          @click="handleCreateNewTable"
        >
          <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
          {{ $t("ui-editor.actions.create-table") }}
        </button>
      </div>
      <!-- table list -->
      <div
        class="w-full h-auto grid auto-rows-auto border-y relative overflow-y-auto"
      >
        <!-- table header -->
        <div
          class="sticky top-0 z-10 grid grid-cols-[repeat(6,_minmax(0,_1fr))_32px] w-full border-b text-sm leading-6 select-none bg-gray-50 text-gray-400"
        >
          <span
            v-for="header in tableHeaderList"
            :key="header.key"
            class="table-header-item-container"
            >{{ header.label }}</span
          >
          <span></span>
        </div>
        <!-- table body -->
        <div class="w-full">
          <div
            v-for="(table, index) in tableList"
            :key="`${index}-${table.newName}`"
            class="grid grid-cols-[repeat(6,_minmax(0,_1fr))_32px] text-sm even:bg-gray-50"
            :class="
              isDroppedTable(table) && 'text-red-700 !bg-red-50 opacity-70'
            "
          >
            <div class="table-body-item-container">
              <NEllipsis
                class="w-full cursor-pointer leading-6 my-2 hover:text-accent"
              >
                <span @click="handleTableItemClick(table)">
                  {{ table.newName }}
                </span>
              </NEllipsis>
            </div>
            <div class="table-body-item-container">
              {{ table.rowCount }}
            </div>
            <div class="table-body-item-container">
              {{ bytesToString(table.dataSize) }}
            </div>
            <div class="table-body-item-container">
              {{ table.engine }}
            </div>
            <div class="table-body-item-container">
              {{ table.collation }}
            </div>
            <div class="table-body-item-container">
              {{ table.comment }}
            </div>
            <div class="w-full flex justify-start items-center">
              <n-tooltip v-if="!isDroppedTable(table)" trigger="hover">
                <template #trigger>
                  <heroicons:trash
                    class="w-[14px] h-auto text-gray-500 cursor-pointer hover:opacity-80"
                    @click="handleDropTable(table)"
                  />
                </template>
                <span>{{ $t("ui-editor.actions.drop-table") }}</span>
              </n-tooltip>
              <n-tooltip v-else trigger="hover">
                <template #trigger>
                  <heroicons:arrow-uturn-left
                    class="w-[14px] h-auto text-gray-500 cursor-pointer hover:opacity-80"
                    @click="handleRestoreTable(table)"
                  />
                </template>
                <span>{{ $t("ui-editor.actions.restore") }}</span>
              </n-tooltip>
            </div>
          </div>
        </div>
      </div>
    </template>
    <template v-else-if="state.selectedTab === 'er-diagram'">
      <!-- TODO: ER diagram placeholder -->
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

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :database-id="state.tableNameModalContext.databaseId"
    :table-name="state.tableNameModalContext.tableName"
    @close="state.tableNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  generateUniqueTabId,
  useDatabaseStore,
  useNotificationStore,
  useUIEditorStore,
} from "@/store";
import { DatabaseId, DatabaseTabContext, UIEditorTabType } from "@/types";
import { Table } from "@/types/UIEditor";
import { bytesToString } from "@/utils";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";
import TableNameModal from "../Modals/TableNameModal.vue";
import { diffTableList } from "@/utils/UIEditor/diffTable";

type TabType = "table-list" | "er-diagram" | "raw-sql";

interface LocalState {
  selectedTab: TabType;
  isFetchingDDL: boolean;
  statement: string;
  tableNameModalContext?: {
    databaseId: DatabaseId;
    tableName: string | undefined;
  };
}

const { t } = useI18n();
const editorStore = useUIEditorStore();
const databaseStore = useDatabaseStore();
const notificationStore = useNotificationStore();
const state = reactive<LocalState>({
  selectedTab: "table-list",
  isFetchingDDL: false,
  statement: "",
});
const currentTab = editorStore.currentTab as DatabaseTabContext;
const database = databaseStore.getDatabaseById(currentTab.databaseId);

const tableList = computed(() => {
  return editorStore.tableList;
});
const tableHeaderList = computed(() => {
  return [
    {
      key: "name",
      label: t("ui-editor.database.name"),
    },
    {
      key: "raw-count",
      label: t("ui-editor.database.row-count"),
    },
    {
      key: "data-size",
      label: t("ui-editor.database.data-size"),
    },
    {
      key: "engine",
      label: t("ui-editor.database.engine"),
    },
    {
      key: "collation",
      label: t("ui-editor.database.collation"),
    },
    {
      key: "comment",
      label: t("ui-editor.database.comment"),
    },
  ];
});

watch(
  () => state.selectedTab,
  async () => {
    if (state.selectedTab === "raw-sql") {
      state.isFetchingDDL = true;
      const originTableList = editorStore.originTableList.filter(
        (table) => table.databaseId === database.id
      );
      const updatedTableList =
        await editorStore.getOrFetchTableListByDatabaseId(database.id);
      const diffTableListResult = diffTableList(
        originTableList,
        updatedTableList
      );
      if (
        diffTableListResult.createTableList.length > 0 ||
        diffTableListResult.alterTableList.length > 0 ||
        diffTableListResult.renameTableList.length > 0 ||
        diffTableListResult.dropTableList.length > 0
      ) {
        const databaseEdit = {
          databaseId: database.id,
          ...diffTableListResult,
        };
        const databaseEditResult = await editorStore.postDatabaseEdit(
          databaseEdit
        );
        if (databaseEditResult.validateResultList.length > 0) {
          notificationStore.pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: "Invalid request",
            description: JSON.stringify(databaseEditResult.validateResultList),
          });
          state.statement = "";
          return;
        }
        state.statement = databaseEditResult.statement;
      }
      state.isFetchingDDL = false;
    }
  }
);

const isDroppedTable = (table: Table) => {
  return table.status === "dropped";
};

const handleChangeTab = (tab: TabType) => {
  state.selectedTab = tab;
};

const handleCreateNewTable = () => {
  state.tableNameModalContext = {
    databaseId: database.id,
    tableName: undefined,
  };
};

const handleTableItemClick = (table: Table) => {
  editorStore.addTab({
    id: generateUniqueTabId(),
    type: UIEditorTabType.TabForTable,
    databaseId: database.id,
    tableName: table.newName,
  });
};

const handleDropTable = (table: Table) => {
  editorStore.dropTable(table);
};

const handleRestoreTable = (table: Table) => {
  editorStore.restoreTable(table);
};
</script>

<style scoped>
.table-header-item-container {
  @apply py-2 px-3;
}
.table-body-item-container {
  @apply w-full h-10 box-border p-px pl-3 pr-4 relative truncate leading-10;
}
</style>
