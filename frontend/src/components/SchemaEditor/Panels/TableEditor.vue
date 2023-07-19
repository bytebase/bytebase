<template>
  <div class="flex flex-col w-full h-full overflow-y-auto">
    <div class="py-2 w-full flex flex-row justify-between items-center">
      <div>
        <div
          v-if="state.selectedSubtab === 'column-list'"
          class="w-full flex justify-between items-center space-x-2"
        >
          <button
            class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="disableChangeTable"
            @click="handleAddColumn"
          >
            <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
            {{ $t("schema-editor.actions.add-column") }}
          </button>
          <button
            class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="disableChangeTable"
            @click="state.showSchemaTemplateDrawer = true"
          >
            <FeatureBadge feature="bb.feature.schema-template" />
            <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
            {{ $t("schema-editor.actions.add-from-template") }}
          </button>
          <button
            v-if="table.status !== 'created'"
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
      <div class="flex justify-end items-center">
        <NInput
          v-if="state.selectedSubtab === 'column-list'"
          v-model:value="searchPattern"
          class="!w-48 mr-3"
          :placeholder="$t('schema-editor.search-column')"
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
              state.selectedSubtab === 'column-list' &&
              'bg-gray-200 text-gray-800'
            "
            @click="handleChangeTab('column-list')"
          >
            <heroicons-outline:queue-list class="inline w-4 h-auto mr-1" />
            {{ $t("schema-editor.columns") }}
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
        </div>
      </div>
    </div>

    <template v-if="state.selectedSubtab === 'column-list'">
      <!-- column table -->
      <div
        id="table-editor-container"
        ref="tableEditorContainerRef"
        class="w-full h-auto grid auto-rows-auto border-y relative overflow-y-auto"
      >
        <!-- column table header -->
        <div
          class="sticky top-0 z-10 grid grid-cols-[repeat(4,_minmax(0,_1fr))_repeat(2,_96px)_minmax(0,_1fr)_32px] w-full text-sm leading-6 select-none bg-gray-50 text-gray-400"
          :class="shownColumnList.length > 0 && 'border-b'"
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
            v-for="(column, index) in shownColumnList"
            :key="`${index}-${column.id}`"
            class="grid grid-cols-[repeat(4,_minmax(0,_1fr))_repeat(2,_96px)_minmax(0,_1fr)_32px] gr text-sm even:bg-gray-50"
            :class="[
              `column-${column.id}`,
              getColumnItemComputedClassList(column),
            ]"
          >
            <div class="table-body-item-container">
              <input
                v-model="column.name"
                :disabled="disableAlterColumn(column)"
                placeholder="column name"
                class="column-field-input column-name-input"
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
                class="column-field-input column-type-input !pr-8"
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
                :placeholder="column.default === undefined ? 'NULL' : 'EMPTY'"
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
                :disabled="
                  disableAlterColumn(column) || isColumnPrimaryKey(column)
                "
                @toggle="(value) => (column.nullable = !value)"
              />
            </div>
            <div
              class="table-body-item-container flex justify-start items-center"
            >
              <BBCheckbox
                class="ml-3"
                :value="isColumnPrimaryKey(column)"
                :disabled="disableAlterColumn(column)"
                @toggle="(value) => setColumnPrimaryKey(column, value)"
              />
            </div>
            <div
              class="table-body-item-container foreign-key-field flex justify-start items-center"
            >
              <span
                v-if="checkColumnHasForeignKey(column)"
                class="column-field-text cursor-pointer !w-auto hover:opacity-80"
                @click="gotoForeignKeyReferencedTable(column)"
              >
                {{ getReferencedForeignKeyName(column) }}
              </span>
              <span
                v-else
                class="column-field-text italic text-gray-400 !w-auto"
                >EMPTY</span
              >
              <button
                class="foreign-key-edit-button hidden cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
                :disabled="disableAlterColumn(column)"
                @click="handleEditColumnForeignKey(column)"
              >
                <heroicons:pencil-square class="w-4 h-auto text-gray-400" />
              </button>
            </div>
            <div class="w-full flex justify-start items-center">
              <n-tooltip v-if="!isDroppedColumn(column)" trigger="hover">
                <template #trigger>
                  <button
                    :disabled="disableChangeTable"
                    class="text-gray-500 cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
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
                    class="text-gray-500 cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
                    :disabled="disableChangeTable"
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
  </div>

  <EditColumnForeignKeyModal
    v-if="state.showEditColumnForeignKeyModal && editForeignKeyColumn"
    :database-id="currentTab.databaseId"
    :schema-id="schema.id"
    :table-id="table.id"
    :column-id="editForeignKeyColumn.id"
    @close="state.showEditColumnForeignKeyModal = false"
  />
  <Drawer
    :show="state.showSchemaTemplateDrawer"
    @close="state.showSchemaTemplateDrawer = false"
  >
    <DrawerContent :title="$t('schema-template.field-template')">
      <div class="w-[46rem]">
        <SettingWorkspaceSchemaTemplate
          :engine="databaseEngine"
          @apply="handleApplyColumnTemplate"
        />
      </div>
    </DrawerContent>
  </Drawer>
  <FeatureModal
    feature="bb.feature.schema-template"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, isUndefined, flatten } from "lodash-es";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { NDropdown } from "naive-ui";
import { v1 as uuidv1 } from "uuid";
import {
  hasFeature,
  generateUniqueTabId,
  useNotificationStore,
  useSchemaEditorStore,
} from "@/store/modules";
import { TableTabContext } from "@/types";
import { ColumnMetadata } from "@/types/proto/store/database";
import { DatabaseEdit, SchemaEditorTabType } from "@/types/schemaEditor";
import {
  Column,
  Table,
  Schema,
  convertColumnMetadataToColumn,
  ForeignKey,
} from "@/types/schemaEditor/atomType";
import { getDataTypeSuggestionList } from "@/utils";
import { BBCheckbox, BBSpin } from "@/bbkit";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";
import { isColumnChanged } from "../utils/column";
import { isTableChanged } from "../utils/table";
import { diffSchema } from "@/utils/schemaEditor/diffSchema";
import EditColumnForeignKeyModal from "../Modals/EditColumnForeignKeyModal.vue";
import { Engine } from "@/types/proto/v1/common";
import { Drawer, DrawerContent } from "@/components/v2";
import SettingWorkspaceSchemaTemplate from "../../../views/SettingWorkspaceSchemaTemplate.vue";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";

type SubtabType = "column-list" | "raw-sql";

interface LocalState {
  selectedSubtab: SubtabType;
  isFetchingDDL: boolean;
  statement: string;
  showEditColumnForeignKeyModal: boolean;
  showSchemaTemplateDrawer: boolean;
  showFeatureModal: boolean;
}

const { t } = useI18n();
const editorStore = useSchemaEditorStore();
const notificationStore = useNotificationStore();
const currentTab = computed(() => editorStore.currentTab as TableTabContext);
const databaseSchema = editorStore.databaseSchemaById.get(
  currentTab.value.databaseId
)!;
const state = reactive<LocalState>({
  selectedSubtab:
    (currentTab.value.selectedSubtab as SubtabType) || "column-list",
  isFetchingDDL: false,
  statement: "",
  showEditColumnForeignKeyModal: false,
  showSchemaTemplateDrawer: false,
  showFeatureModal: false,
});

const table = ref(editorStore.getTableWithTableTab(currentTab.value) as Table);
const schema = computed(() => {
  return databaseSchema.schemaList.find(
    (schema) => schema.id === currentTab.value.schemaId
  ) as Schema;
});
const foreignKeyList = computed(() => {
  return schema.value.foreignKeyList.filter(
    (pk) => pk.tableId === currentTab.value.tableId
  ) as ForeignKey[];
});

const searchPattern = ref("");
const tableEditorContainerRef = ref<HTMLDivElement>();
const editForeignKeyColumn = ref<Column>();

const shownColumnList = computed(() => {
  return table.value.columnList.filter((column) =>
    column.name.includes(searchPattern.value.trim())
  );
});
const isDroppedSchema = computed(() => {
  return schema.value.status === "dropped";
});

const isDroppedTable = computed(() => {
  return table.value.status === "dropped";
});

const allowResetTable = computed(() => {
  if (table.value.status === "created") {
    return false;
  }

  return (
    isTableChanged(
      currentTab.value.databaseId,
      currentTab.value.schemaId,
      currentTab.value.tableId
    ) || isDroppedTable.value
  );
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
    {
      key: "primary",
      label: t("schema-editor.column.primary"),
    },
    {
      key: "foreign_key",
      label: t("schema-editor.column.foreign-key"),
    },
  ];
});

const databaseEngine = computed(() => {
  return databaseSchema.database.instanceEntity.engine;
});

const dataTypeOptions = computed(() => {
  return getDataTypeSuggestionList(databaseEngine.value).map((dataType) => {
    return {
      label: dataType,
      key: dataType,
    };
  });
});

const getColumnItemComputedClassList = (column: Column) => {
  if (column.status === "dropped") {
    return ["text-red-700", "cursor-not-allowed", "!bg-red-50", "opacity-70"];
  } else if (column.status === "created") {
    return ["text-green-700", "!bg-green-50"];
  } else if (
    isColumnChanged(
      currentTab.value.databaseId,
      currentTab.value.schemaId,
      currentTab.value.tableId,
      column.id
    )
  ) {
    return ["text-yellow-700", "!bg-yellow-50"];
  }
  return [];
};

const dataDefaultOptions = [
  {
    label: "NULL",
    key: "NULL",
  },
  {
    label: "EMPTY",
    key: "EMPTY",
  },
];

watch(
  () => state.selectedSubtab,
  async () => {
    currentTab.value.selectedSubtab = state.selectedSubtab;
    if (state.selectedSubtab === "raw-sql") {
      const originSchema = editorStore.getOriginSchema(
        currentTab.value.databaseId,
        currentTab.value.schemaId
      ) as Schema;
      const diffSchemaResult = diffSchema(
        currentTab.value.databaseId,
        originSchema,
        schema.value
      );
      const databaseEdit: DatabaseEdit = {
        databaseId: Number(currentTab.value.databaseId),
        createSchemaList: [],
        renameSchemaList: [],
        dropSchemaList: [],
        createTableList: diffSchemaResult.createTableList.filter(
          (context) => context.name === table.value.name
        ),
        alterTableList: diffSchemaResult.alterTableList.filter(
          (context) => context.name === table.value.name
        ),
        renameTableList: diffSchemaResult.renameTableList.filter(
          (context) => context.newName === table.value.name
        ),
        dropTableList: diffSchemaResult.dropTableList.filter(
          (context) => context.name === table.value.name
        ),
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
  },
  {
    immediate: true,
  }
);

const isColumnPrimaryKey = (column: Column): boolean => {
  return table.value.primaryKey.columnIdList.includes(column.id);
};

const checkColumnHasForeignKey = (column: Column): boolean => {
  const columnIdList = flatten(
    foreignKeyList.value.map((fk) => fk.columnIdList)
  );
  return columnIdList.includes(column.id);
};

const getReferencedForeignKeyName = (column: Column) => {
  if (!checkColumnHasForeignKey(column)) {
    return;
  }
  const fk = foreignKeyList.value.find(
    (fk) =>
      fk.columnIdList.find((columnId) => columnId === column.id) !== undefined
  );
  const index = fk?.columnIdList.findIndex(
    (columnId) => columnId === column.id
  );

  if (isUndefined(fk) || isUndefined(index) || index < 0) {
    return;
  }
  const referencedSchema = editorStore.getSchema(
    currentTab.value.databaseId,
    fk.referencedSchemaId
  );
  const referencedTable = editorStore.getTable(
    currentTab.value.databaseId,
    fk.referencedSchemaId,
    fk.referencedTableId
  );
  if (!referencedTable) {
    return;
  }
  const referColumn = referencedTable.columnList.find(
    (column) => column.id === fk.referencedColumnIdList[index]
  );
  if (databaseEngine.value === Engine.MYSQL) {
    return `${referencedTable.name}(${referColumn?.name})`;
  } else {
    return `${referencedSchema?.name}.${referencedTable.name}(${referColumn?.name})`;
  }
};

const isDroppedColumn = (column: Column): boolean => {
  return column.status === "dropped";
};

const disableChangeTable = computed(() => {
  return isDroppedSchema.value || isDroppedTable.value;
});

const disableAlterColumn = (column: Column): boolean => {
  return (
    isDroppedSchema.value || isDroppedTable.value || isDroppedColumn(column)
  );
};

const setColumnPrimaryKey = (column: Column, isPrimaryKey: boolean) => {
  if (isPrimaryKey) {
    column.nullable = false;
    table.value.primaryKey.columnIdList.push(column.id);
  } else {
    table.value.primaryKey.columnIdList =
      table.value.primaryKey.columnIdList.filter(
        (columnId) => columnId !== column.id
      );
  }
};

const handleChangeTab = (tab: SubtabType) => {
  state.selectedSubtab = tab;
};

const handleApplyColumnTemplate = (
  template: SchemaTemplateSetting_FieldTemplate
) => {
  if (!hasFeature("bb.feature.schema-template")) {
    state.showFeatureModal = true;
    return;
  }
  if (template.engine !== databaseEngine.value || !template.column) {
    return;
  }
  const column: Column = {
    id: uuidv1(),
    name: template.column.name,
    type: template.column.type,
    nullable: template.column.nullable,
    comment: template.column.comment,
    default: template.column.default,
    status: "created",
  };
  table.value.columnList.push(column);
  state.showSchemaTemplateDrawer = false;
};

const handleAddColumn = () => {
  const column = convertColumnMetadataToColumn(ColumnMetadata.fromPartial({}));
  column.status = "created";
  table.value.columnList.push(column);
  nextTick(() => {
    (
      tableEditorContainerRef.value?.querySelector(
        `.column-${column.id} .column-name-input`
      ) as HTMLInputElement
    )?.focus();
  });
};

const handleColumnDefaultFieldChange = (
  column: Column,
  defaultString: string
) => {
  if (defaultString === "NULL") {
    column.default = undefined;
  } else if (defaultString === "EMPTY") {
    column.default = "";
  }
};

const gotoForeignKeyReferencedTable = (column: Column) => {
  if (!checkColumnHasForeignKey(column)) {
    return;
  }
  const fk = foreignKeyList.value.find(
    (fk) =>
      fk.columnIdList.find((columnId) => columnId === column.id) !== undefined
  );
  const index = fk?.columnIdList.findIndex(
    (columnId) => columnId === column.id
  );
  if (isUndefined(fk) || isUndefined(index) || index < 0) {
    return;
  }

  const referencedSchema = editorStore.getSchema(
    currentTab.value.databaseId,
    fk.referencedSchemaId
  );
  const referencedTable = editorStore.getTable(
    currentTab.value.databaseId,
    fk.referencedSchemaId,
    fk.referencedTableId
  );
  const referColumn = referencedTable?.columnList.find(
    (column) => column.id === fk.referencedColumnIdList[index]
  );
  if (!referencedSchema || !referencedTable || !referColumn) {
    return;
  }

  editorStore.addTab({
    id: generateUniqueTabId(),
    type: SchemaEditorTabType.TabForTable,
    databaseId: currentTab.value.databaseId,
    schemaId: referencedSchema.id,
    tableId: referencedTable.id,
  });

  nextTick(() => {
    const container = document.querySelector("#table-editor-container");
    const input = container?.querySelector(
      `.column-${referColumn.id} .column-name-input`
    ) as HTMLInputElement | undefined;
    if (input) {
      input.focus();
      scrollIntoView(input);
    }
  });
};

const handleEditColumnForeignKey = (column: Column) => {
  editForeignKeyColumn.value = column;
  state.showEditColumnForeignKeyModal = true;
};

const handleDropColumn = (column: Column) => {
  if (column.status === "created") {
    table.value.columnList = table.value.columnList.filter(
      (item) => item !== column
    );
    table.value.primaryKey.columnIdList =
      table.value.primaryKey.columnIdList.filter(
        (columnId) => columnId !== column.id
      );

    const foreignKeyList = schema.value.foreignKeyList.filter(
      (fk) => fk.tableId === currentTab.value.tableId
    );
    for (const foreignKey of foreignKeyList) {
      const columnRefIndex = foreignKey.columnIdList.findIndex(
        (columnId) => columnId === column.id
      );
      if (columnRefIndex > -1) {
        foreignKey.columnIdList.splice(columnRefIndex, 1);
        foreignKey.referencedColumnIdList.splice(columnRefIndex, 1);
      }
    }
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
  if (table.value.status === "created") {
    return;
  }

  const originSchema = editorStore.getOriginSchema(
    currentTab.value.databaseId,
    currentTab.value.schemaId
  );
  const originTable = editorStore.getOriginTable(
    currentTab.value.databaseId,
    currentTab.value.schemaId,
    table.value.id
  );

  if (!originTable) {
    return;
  }

  table.value.name = originTable.name;
  table.value.columnList = cloneDeep(originTable.columnList);
  table.value.primaryKey = cloneDeep(originTable.primaryKey);
  table.value.status = "normal";

  schema.value.foreignKeyList = schema.value.foreignKeyList.filter(
    (fk) => fk.tableId !== table.value.id
  );
  const originForeignKeyList =
    originSchema?.foreignKeyList.filter(
      (fk) => fk.tableId === table.value.id
    ) || [];
  for (const originForeignKey of originForeignKeyList) {
    schema.value.foreignKeyList.push(cloneDeep(originForeignKey));
  }
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
.column-field-text {
  @apply w-full pl-3 pr-1 box-border border-transparent truncate select-none rounded bg-transparent text-sm placeholder:italic placeholder:text-gray-400 focus:bg-white focus:text-black;
}
.foreign-key-field:hover .foreign-key-edit-button {
  @apply block;
}
</style>
