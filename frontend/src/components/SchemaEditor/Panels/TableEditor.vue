<template>
  <div class="flex flex-col w-full h-full overflow-y-auto">
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
        >{{ $t("schema-editor.column-list") }}</span
      >
      <span
        class="-mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none"
        :class="
          state.selectedTab === 'raw-sql' &&
          'bg-white border-gray-300 text-gray-800'
        "
        @click="handleChangeTab('raw-sql')"
        >{{ $t("schema-editor.raw-sql") }}</span
      >
    </div>

    <template v-if="state.selectedTab === 'column-list'">
      <div class="w-full py-2 flex flex-row justify-between items-center">
        <div>
          <button
            class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:opacity-80"
            :disabled="isDroppedTable"
            @click="handleAddColumn"
          >
            <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
            {{ $t("schema-editor.actions.add-column") }}
          </button>
        </div>
        <div class="w-auto flex flex-row justify-end items-center space-x-3">
          <button
            v-if="state.tableCache.status !== 'created'"
            class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="!allowResetTable"
            @click="handleDiscardChanges"
          >
            <heroicons-solid:arrow-uturn-left
              class="w-4 h-auto mr-1 text-gray-400"
            />
            {{ $t("schema-editor.actions.reset") }}
          </button>
        </div>
      </div>

      <!-- column table -->
      <div
        class="w-full h-auto grid auto-rows-auto border-y relative overflow-y-auto"
      >
        <!-- column table header -->
        <div
          class="sticky top-0 z-10 grid grid-cols-[repeat(4,_minmax(0,_1fr))_112px_32px] w-full text-sm leading-6 select-none bg-gray-50 text-gray-400"
          :class="state.tableCache.columnList.length > 0 && 'border-b'"
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
            v-for="(column, index) in state.tableCache.columnList"
            :key="`${index}-${column.oldName}`"
            class="grid grid-cols-[repeat(4,_minmax(0,_1fr))_112px_32px] gr text-sm even:bg-gray-50"
            :class="
              isDroppedColumn(column) &&
              'text-red-700 cursor-not-allowed !bg-red-50 opacity-70'
            "
          >
            <div class="table-body-item-container">
              <input
                v-model="column.newName"
                :disabled="disableAlterColumn(column)"
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
                :disabled="disableAlterColumn(column)"
                placeholder="column type"
                class="column-field-input !pr-8"
                type="text"
              />
              <NDropdown
                trigger="click"
                :disabled="disableAlterColumn(column)"
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
              class="table-body-item-container flex flex-row justify-between items-center"
            >
              <input
                v-model="column.default"
                :disabled="disableAlterColumn(column)"
                :placeholder="column.hasDefault === false ? 'NULL' : ''"
                class="column-field-input !pr-8"
                type="text"
              />
              <NDropdown
                trigger="click"
                :disabled="disableAlterColumn(column)"
                :options="dataDefaultOptions"
                @select="(defaultString:string)=>handleColumnDefaultFieldChange(column, defaultString)"
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
                v-model="column.comment"
                :disabled="disableAlterColumn(column)"
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
                :value="!column.nullable"
                :disabled="disableAlterColumn(column)"
                @toggle="(value) => (column.nullable = !value)"
              />
            </div>
            <div class="w-full flex justify-start items-center">
              <n-tooltip v-if="!isDroppedColumn(column)" trigger="hover">
                <template #trigger>
                  <button
                    :disabled="isDroppedTable"
                    class="text-gray-500 cursor-pointer hover:opacity-80"
                    @click="handleDropColumn(column)"
                  >
                    <heroicons:trash class="w-4 h-auto" />
                  </button>
                </template>
                <span>{{ $t("schema-editor.actions.drop-column") }}</span>
              </n-tooltip>
              <n-tooltip v-else trigger="hover">
                <template #trigger>
                  <button
                    class="text-gray-500 cursor-pointer hover:opacity-80"
                    :disabled="isDroppedTable"
                    @click="handleRestoreColumn(column)"
                  >
                    <heroicons:arrow-uturn-left class="w-4 h-auto" />
                  </button>
                </template>
                <span>{{ $t("schema-editor.actions.restore") }}</span>
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
          {{ $t("schema-editor.nothing-changed") }}
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useDebounceFn } from "@vueuse/core";
import { useNotificationStore, useSchemaEditorStore } from "@/store/modules";
import { TableTabContext } from "@/types";
import { ColumnMetadata } from "@/types/proto/database";
import { DatabaseEdit } from "@/types/schemaEditor";
import { Column, Table } from "@/types/schemaEditor/atomType";
import { getDataTypeSuggestionList } from "@/utils";
import { diffTableList } from "@/utils/schemaEditor/diffTable";
import { transformColumnDataToColumn } from "@/utils/schemaEditor/transform";
import { BBCheckbox, BBSpin } from "@/bbkit";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";

type TabType = "column-list" | "raw-sql";

interface LocalState {
  selectedTab: TabType;
  isFetchingDDL: boolean;
  statement: string;
  tableCache: Table;
}

const { t } = useI18n();
const editorStore = useSchemaEditorStore();
const notificationStore = useNotificationStore();
const currentTab = editorStore.currentTab as TableTabContext;
const databaseState = editorStore.databaseStateById.get(currentTab.databaseId)!;
const state = reactive<LocalState>({
  selectedTab: "column-list",
  isFetchingDDL: false,
  statement: "",
  tableCache: cloneDeep(editorStore.getTableWithTableTab(currentTab) as Table),
});

const table = computed(
  () => editorStore.getTableWithTableTab(currentTab) as Table
);

const isDroppedTable = computed(() => {
  return state.tableCache.status === "dropped";
});

const allowResetTable = computed(() => {
  if (state.tableCache.status === "created") {
    return false;
  }

  const originTable = databaseState.originTableList.find(
    (item) => item.oldName === state.tableCache.oldName
  );
  return !isEqual(originTable, state.tableCache) || isDroppedTable.value;
});

const columnHeaderList = computed(() => {
  return [
    {
      key: "name",
      label: t("schema-editor.column.name"),
    },
    {
      key: "type",
      label: t("schema-editor.column.type"),
    },
    {
      key: "default",
      label: t("schema-editor.column.default"),
    },
    {
      key: "comment",
      label: t("schema-editor.column.comment"),
    },
    {
      key: "nullable",
      label: t("schema-editor.column.not-null"),
    },
  ];
});

const dataTypeOptions = computed(() => {
  const database = databaseState.database;
  return getDataTypeSuggestionList(database.instance.engine).map((dataType) => {
    return {
      label: dataType,
      key: dataType,
    };
  });
});

const dataDefaultOptions = [
  // TODO(steven): support set default field with EMPTY.
  // {
  //   label: "EMPTY",
  //   key: "EMPTY",
  // },
  {
    label: "NULL",
    key: "NULL",
  },
];

watch(
  () => state.tableCache,
  () => {
    handleSaveChanges();
  },
  {
    deep: true,
  }
);

watch([table.value], () => {
  state.tableCache.newName = table.value.newName;
  state.tableCache.status = table.value.status;
});

watch(
  () => state.selectedTab,
  async () => {
    if (state.selectedTab === "raw-sql") {
      const originTable = databaseState.originTableList.find(
        (item) => item.oldName === state.tableCache.oldName
      );
      const diffTableListResult = diffTableList(
        originTable ? [originTable] : [],
        [state.tableCache]
      );
      const databaseEdit: DatabaseEdit = {
        databaseId: currentTab.databaseId,
        ...diffTableListResult,
      };
      state.isFetchingDDL = true;
      const databaseEditResult = await editorStore.postDatabaseEdit(
        databaseEdit
      );
      if (databaseEditResult.validateResultList.length > 0) {
        notificationStore.pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Invalid request",
          description: databaseEditResult.validateResultList
            .map((result) => result.message)
            .join("\n"),
        });
        state.statement = "";
        return;
      }
      state.statement = databaseEditResult.statement;
      state.isFetchingDDL = false;
    }
  }
);

const isDroppedColumn = (column: Column): boolean => {
  return column.status === "dropped";
};

const disableAlterColumn = (column: Column): boolean => {
  return isDroppedTable.value || isDroppedColumn(column);
};

const handleChangeTab = (tab: TabType) => {
  state.selectedTab = tab;
};

const handleSaveChanges = useDebounceFn(async () => {
  const table = editorStore.getTableWithTableTab(currentTab) as Table;
  table.columnList = cloneDeep(state.tableCache.columnList);
}, 500);

const handleAddColumn = () => {
  const column = transformColumnDataToColumn(ColumnMetadata.fromPartial({}));
  column.status = "created";
  state.tableCache.columnList.push(column);
};

const handleColumnDefaultFieldChange = (
  column: Column,
  defaultString: string
) => {
  if (defaultString === "NULL") {
    column.hasDefault = false;
    column.default = "";
  }
};

const handleDropColumn = (column: Column) => {
  if (column.status === "created") {
    state.tableCache.columnList = state.tableCache.columnList.filter(
      (item) => item !== column
    );
  } else {
    column.status = "dropped";
  }
};

const handleRestoreColumn = (column: Column) => {
  if (column.status === "created") {
    return;
  }

  column.status = "normal";
};

const handleDiscardChanges = () => {
  if (state.tableCache.status === "created") {
    return;
  }

  state.tableCache.newName = state.tableCache.oldName;
  state.tableCache.columnList = cloneDeep(state.tableCache.originColumnList);
  state.tableCache.status = "normal";

  const table = editorStore.getTableWithTableTab(currentTab) as Table;
  table.newName = table.oldName;
  table.columnList = cloneDeep(table.columnList);
  table.status = "normal";
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
  @apply w-full pr-1 box-border border-transparent truncate select-none rounded bg-transparent text-sm placeholder:italic placeholder:text-gray-400 focus:bg-white focus:text-black;
}
</style>
