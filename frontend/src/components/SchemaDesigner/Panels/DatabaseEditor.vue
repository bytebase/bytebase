<template>
  <div class="flex flex-col w-full h-full overflow-y-auto">
    <div class="py-2 w-full flex flex-row justify-between items-center">
      <div>
        <div
          v-if="state.selectedSubtab === 'table-list'"
          class="w-full flex justify-between items-center space-x-2"
        >
          <div class="flex flex-row justify-start items-center">
            <div
              v-if="shouldShowSchemaSelector"
              class="ml-2 flex flex-row justify-start items-center mr-3 text-sm"
            >
              <span class="mr-1">Schema:</span>
              <n-select
                v-model:value="state.selectedSchemaId"
                class="min-w-[8rem]"
                :options="schemaSelectorOptionList"
              />
            </div>
            <button
              class="flex flex-row justify-center items-center border px-3 py-1 leading-6 rounded text-sm hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="!allowCreateTable"
              @click="handleCreateNewTable"
            >
              <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
              {{ $t("schema-editor.actions.create-table") }}
            </button>
          </div>
        </div>
      </div>
      <div class="flex justify-end items-center">
        <NInput
          v-if="state.selectedSubtab === 'table-list'"
          v-model:value="searchPattern"
          class="!w-48 mr-3"
          :placeholder="$t('schema-editor.search-table')"
        >
          <template #prefix>
            <heroicons-outline:search class="w-4 h-auto text-gray-300" />
          </template>
        </NInput>
        <div
          class="flex flex-row justify-end items-center bg-gray-100 p-1 rounded"
        >
          <button
            class="px-2 leading-7 text-sm text-gray-500 cursor-pointer select-none rounded flex justify-center items-center"
            :class="
              state.selectedSubtab === 'table-list' &&
              'bg-gray-200 text-gray-800'
            "
            @click="handleChangeTab('table-list')"
          >
            <heroicons-outline:queue-list class="inline w-4 h-auto mr-1" />
            {{ $t("schema-editor.tables") }}
          </button>
          <button
            class="px-2 leading-7 text-sm text-gray-500 cursor-pointer select-none rounded flex justify-center items-center"
            :class="
              state.selectedSubtab === 'raw-sql' && 'bg-gray-200 text-gray-800'
            "
            @click="handleChangeTab('raw-sql')"
          >
            <heroicons-outline:clipboard class="inline w-4 h-auto mr-1" />
            {{ $t("schema-editor.raw-sql") }}
          </button>
          <button
            class="px-2 leading-7 text-sm text-gray-500 cursor-pointer select-none rounded flex justify-center items-center"
            :class="
              state.selectedSubtab === 'schema-diagram' &&
              'bg-gray-200 text-gray-800'
            "
            @click="handleChangeTab('schema-diagram')"
          >
            <SchemaDiagramIcon class="mr-1" />
            {{ $t("schema-diagram.self") }}
          </button>
        </div>
      </div>
    </div>

    <!-- List view -->
    <template v-if="state.selectedSubtab === 'table-list'">
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
            v-for="(table, index) in shownTableList"
            :key="`${index}-${table.name}`"
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
                  {{ table.name }}
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
              <NTooltip v-if="!isDroppedTable(table)" trigger="hover" to="body">
                <template #trigger>
                  <heroicons:trash
                    class="w-[14px] h-auto text-gray-500 cursor-pointer hover:opacity-80"
                    @click="handleDropTable(table)"
                  />
                </template>
                <span>{{ $t("schema-editor.actions.drop-table") }}</span>
              </NTooltip>
              <NTooltip v-else trigger="hover" to="body">
                <template #trigger>
                  <heroicons:arrow-uturn-left
                    class="w-[14px] h-auto text-gray-500 cursor-pointer hover:opacity-80"
                    @click="handleRestoreTable(table)"
                  />
                </template>
                <span>{{ $t("schema-editor.actions.restore") }}</span>
              </NTooltip>
            </div>
          </div>
        </div>
      </div>
    </template>
    <div
      v-else-if="state.selectedSubtab === 'raw-sql'"
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
    <template v-else-if="state.selectedSubtab === 'schema-diagram'">
      <SchemaDiagram
        :key="currentTab.databaseId"
        :database="databaseV1"
        :database-metadata="databaseMetadata"
        :schema-status="schemaStatus"
        :table-status="tableStatus"
        :column-status="columnStatus"
        :editable="true"
        @edit-table="tryEditTable"
        @edit-column="tryEditColumn"
      />
    </template>
  </div>

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :database-id="state.tableNameModalContext.databaseId"
    :schema-id="state.tableNameModalContext.schemaId"
    :table-name="state.tableNameModalContext.tableName"
    @close="state.tableNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NEllipsis, NTooltip } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  generateUniqueTabId,
  useDatabaseV1Store,
  useNotificationStore,
  useSchemaEditorStore,
} from "@/store";
import {
  DatabaseTabContext,
  DatabaseSchema,
  SchemaEditorTabType,
  DatabaseEdit,
} from "@/types";
import { Table } from "@/types/schemaEditor/atomType";
import { bytesToString } from "@/utils";
import {
  checkHasSchemaChanges,
  diffSchema,
  mergeDiffResults,
} from "@/utils/schemaEditor/diffSchema";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";
import TableNameModal from "../Modals/TableNameModal.vue";
import { SchemaDiagram, SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { useMetadataForDiagram } from "../utils/useMetadataForDiagram";
import {
  ColumnMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import { Engine } from "@/types/proto/v1/common";

type SubtabType = "table-list" | "schema-diagram" | "raw-sql";

interface LocalState {
  selectedSubtab: SubtabType;
  selectedSchemaId: string;
  isFetchingDDL: boolean;
  statement: string;
  tableNameModalContext?: {
    databaseId: string;
    schemaId: string;
    tableName: string | undefined;
  };
}

const { t } = useI18n();
const editorStore = useSchemaEditorStore();
const notificationStore = useNotificationStore();
const searchPattern = ref("");
const currentTab = computed(() => editorStore.currentTab as DatabaseTabContext);
const state = reactive<LocalState>({
  selectedSubtab:
    (currentTab.value.selectedSubtab as SubtabType) || "table-list",
  selectedSchemaId: "",
  isFetchingDDL: false,
  statement: "",
});
const databaseSchema = computed(() => {
  return editorStore.databaseSchemaById.get(
    currentTab.value.databaseId
  ) as DatabaseSchema;
});
const database = computed(() => databaseSchema.value.database);
const databaseV1 = computed(() => {
  return useDatabaseV1Store().getDatabaseByUID(database.value.uid);
});
const databaseEngine = computed(() => database.value.instanceEntity.engine);
const schemaList = computed(() => {
  return databaseSchema.value.schemaList;
});
const selectedSchema = computed(() => {
  return schemaList.value.find(
    (schema) => schema.id === state.selectedSchemaId
  );
});
const tableList = computed(() => {
  return selectedSchema.value?.tableList ?? [];
});
const shownTableList = computed(() => {
  return tableList.value.filter((table) =>
    table.name.includes(searchPattern.value.trim())
  );
});

const shouldShowSchemaSelector = computed(() => {
  return databaseEngine.value === Engine.POSTGRES;
});

const allowCreateTable = computed(() => {
  if (databaseEngine.value === Engine.POSTGRES) {
    return (
      schemaList.value.length > 0 &&
      selectedSchema.value &&
      selectedSchema.value.status !== "dropped"
    );
  }
  return true;
});

const schemaSelectorOptionList = computed(() => {
  const optionList = [];
  for (const schema of schemaList.value) {
    optionList.push({
      label: schema.name,
      value: schema.id,
    });
  }
  return optionList;
});

const tableHeaderList = computed(() => {
  return [
    {
      key: "name",
      label: t("schema-editor.database.name"),
    },
    {
      key: "raw-count",
      label: t("schema-editor.database.row-count"),
    },
    {
      key: "data-size",
      label: t("schema-editor.database.data-size"),
    },
    {
      key: "engine",
      label: t("schema-editor.database.engine"),
    },
    {
      key: "collation",
      label: t("schema-editor.database.collation"),
    },
    {
      key: "comment",
      label: t("schema-editor.database.comment"),
    },
  ];
});

watch(
  [() => currentTab.value, () => schemaList],
  () => {
    const schemaIdList = schemaList.value.map((schema) => schema.id);
    if (
      currentTab.value &&
      currentTab.value.selectedSchemaId &&
      schemaIdList.includes(currentTab.value.selectedSchemaId)
    ) {
      state.selectedSchemaId = currentTab.value.selectedSchemaId;
    } else {
      state.selectedSchemaId = head(schemaIdList) || "";
    }
  },
  {
    immediate: true,
    deep: true,
  }
);

watch(
  () => state.selectedSubtab,
  async () => {
    currentTab.value.selectedSubtab = state.selectedSubtab;
    if (state.selectedSubtab === "raw-sql") {
      state.isFetchingDDL = true;
      const databaseEditList: DatabaseEdit[] = [];
      for (const schema of databaseSchema.value.schemaList) {
        const originSchema = databaseSchema.value.originSchemaList.find(
          (originSchema) => originSchema.id === schema.id
        );
        const diffSchemaResult = diffSchema(
          database.value.uid,
          originSchema,
          schema
        );
        if (checkHasSchemaChanges(diffSchemaResult)) {
          const index = databaseEditList.findIndex(
            (edit) => String(edit.databaseId) === database.value.uid
          );
          if (index !== -1) {
            databaseEditList[index] = {
              databaseId: Number(database.value.uid),
              ...mergeDiffResults([diffSchemaResult, databaseEditList[index]]),
            };
          } else {
            databaseEditList.push({
              databaseId: Number(database.value.uid),
              ...diffSchemaResult,
            });
          }
        }
      }

      if (databaseEditList.length > 0) {
        const statementList: string[] = [];
        for (const databaseEdit of databaseEditList) {
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
          statementList.push(databaseEditResult.statement);
        }
        state.statement = statementList.join("\n");
      } else {
        state.statement = "";
      }
      state.isFetchingDDL = false;
    }
  },
  {
    immediate: true,
  }
);

const isDroppedTable = (table: Table) => {
  return table.status === "dropped";
};

const handleChangeTab = (tab: SubtabType) => {
  state.selectedSubtab = tab;
};

const handleCreateNewTable = () => {
  const selectedSchema = schemaList.value.find(
    (schema) => schema.id === state.selectedSchemaId
  );
  if (selectedSchema) {
    state.tableNameModalContext = {
      databaseId: database.value.uid,
      schemaId: selectedSchema.id,
      tableName: undefined,
    };
  }
};

const handleTableItemClick = (table: Table) => {
  editorStore.addTab({
    id: generateUniqueTabId(),
    type: SchemaEditorTabType.TabForTable,
    databaseId: database.value.uid,
    schemaId: state.selectedSchemaId,
    tableId: table.id,
  });
};

const handleDropTable = (table: Table) => {
  editorStore.dropTable(database.value.uid, state.selectedSchemaId, table.id);
};

const handleRestoreTable = (table: Table) => {
  editorStore.restoreTable(
    database.value.uid,
    state.selectedSchemaId,
    table.id
  );
};

const {
  databaseMetadata,
  schemaStatus,
  tableStatus,
  columnStatus,
  editableSchema,
  editableTable,
  editableColumn,
} = useMetadataForDiagram(databaseSchema);

const tryEditTable = async (
  schemaMeta: SchemaMetadata,
  tableMeta: TableMetadata
) => {
  const schema = editableSchema(schemaMeta);
  const table = editableTable(tableMeta);
  if (schema && table) {
    state.selectedSchemaId = schema.id;
    await nextTick();
    handleTableItemClick(table);
  }
};

const tryEditColumn = async (
  schemaMeta: SchemaMetadata,
  tableMeta: TableMetadata,
  columnMeta: ColumnMetadata,
  target: "name" | "type"
) => {
  const schema = editableSchema(schemaMeta);
  const table = editableTable(tableMeta);
  const column = editableColumn(columnMeta);
  if (schema && table && column) {
    await tryEditTable(schemaMeta, tableMeta);
    await nextTick();
    const container = document.querySelector("#table-editor-container");
    const input = container?.querySelector(
      `.column-${column.id} .column-${target}-input`
    ) as HTMLInputElement | undefined;
    if (input) {
      input.focus();
      scrollIntoView(input);
    }
  }
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
