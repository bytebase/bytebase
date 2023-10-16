<template>
  <div class="flex flex-col w-full h-full overflow-y-auto">
    <div class="py-2 w-full flex flex-row justify-between items-center">
      <div>
        <div
          v-if="state.selectedSubtab === 'table-list'"
          class="w-full flex justify-between items-center space-x-2"
        >
          <div class="flex flex-row justify-start items-center space-x-3">
            <div
              v-if="shouldShowSchemaSelector"
              class="ml-2 flex flex-row justify-start items-center text-sm"
            >
              <span class="mr-1">Schema:</span>
              <n-select
                v-model:value="state.selectedSchemaId"
                class="min-w-[8rem]"
                :options="schemaSelectorOptionList"
              />
            </div>
            <button
              v-if="!readonly"
              class="flex flex-row justify-center items-center border px-3 py-1 leading-6 rounded text-sm hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="!allowCreateTable"
              @click="handleCreateNewTable"
            >
              <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
              {{ $t("schema-editor.actions.create-table") }}
            </button>
            <button
              v-if="!readonly"
              class="flex flex-row justify-center items-center border px-3 py-1 leading-6 rounded text-sm hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="!allowCreateTable"
              @click="state.showSchemaTemplateDrawer = true"
            >
              <FeatureBadge feature="bb.feature.schema-template" />
              <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
              {{ $t("schema-editor.actions.add-from-template") }}
            </button>
          </div>
        </div>
      </div>
      <div class="flex justify-end items-center">
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
      <BBGrid
        class="border"
        :column-list="tableHeaderList"
        :data-source="shownTableList"
        :custom-header="true"
        :row-clickable="false"
      >
        <template #header>
          <div role="table-row" class="bb-grid-row bb-grid-header-row group">
            <div
              v-for="(header, index) in tableHeaderList"
              :key="index"
              role="table-cell"
              class="bb-grid-header-cell"
            >
              {{ header.title }}
            </div>
          </div>
        </template>
        <template #item="{ item: table, row }: { item: Table, row: number }">
          <div
            class="bb-grid-cell table-item-cell"
            :class="getTableClassList(table, row)"
          >
            <NEllipsis
              class="w-full cursor-pointer leading-6 my-2 hover:text-accent"
            >
              <span @click="handleTableItemClick(table)">
                {{ table.name }}
              </span>
            </NEllipsis>
          </div>
          <div
            v-if="supportClassification"
            class="bb-grid-cell table-item-cell flex items-center gap-x-2 text-sm"
            :class="getTableClassList(table, row)"
          >
            <ClassificationLevelBadge
              :classification="table.classification"
              :classification-config="classificationConfig"
            />
            <div v-if="!readonly && !disableChangeTable(table)" class="flex">
              <button
                v-if="table.classification"
                class="w-4 h-4 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer"
                @click.prevent="table.classification = ''"
              >
                <heroicons-outline:x class="w-3 h-3" />
              </button>
              <button
                class="w-4 h-4 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer"
                @click.prevent="showClassificationDrawer(table)"
              >
                <heroicons-outline:pencil class="w-3 h-3" />
              </button>
            </div>
          </div>
          <div
            class="bb-grid-cell table-item-cell"
            :class="getTableClassList(table, row)"
          >
            {{ table.rowCount }}
          </div>
          <div
            class="bb-grid-cell table-item-cell"
            :class="getTableClassList(table, row)"
          >
            {{ bytesToString(table.dataSize) }}
          </div>
          <div
            class="bb-grid-cell table-item-cell"
            :class="getTableClassList(table, row)"
          >
            {{ table.engine }}
          </div>
          <div
            class="bb-grid-cell table-item-cell"
            :class="getTableClassList(table, row)"
          >
            {{ table.collation }}
          </div>
          <div
            class="bb-grid-cell table-item-cell"
            :class="getTableClassList(table, row)"
          >
            {{ table.userComment }}
          </div>
          <div
            v-if="!readonly"
            class="bb-grid-cell table-item-cell !px-0.5 flex justify-start items-center"
            :class="getTableClassList(table, row)"
          >
            <NTooltip v-if="!isDroppedTable(table)" trigger="hover" to="body">
              <template #trigger>
                <heroicons:trash
                  class="w-4 h-auto text-gray-500 cursor-pointer hover:opacity-80"
                  @click="handleDropTable(table)"
                />
              </template>
              <span>{{ $t("schema-editor.actions.drop-table") }}</span>
            </NTooltip>
            <NTooltip v-else trigger="hover" to="body">
              <template #trigger>
                <heroicons:arrow-uturn-left
                  class="w-4 h-auto text-gray-500 cursor-pointer hover:opacity-80"
                  @click="handleRestoreTable(table)"
                />
              </template>
              <span>{{ $t("schema-editor.actions.restore") }}</span>
            </NTooltip>
          </div>
        </template>
      </BBGrid>
    </template>
    <template v-else-if="state.selectedSubtab === 'schema-diagram'">
      <SchemaDiagram
        :key="currentTab.parentName"
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
    :parent-name="state.tableNameModalContext.parentName"
    :schema-id="state.tableNameModalContext.schemaId"
    :table-name="state.tableNameModalContext.tableName"
    @close="state.tableNameModalContext = undefined"
  />

  <Drawer
    :show="state.showSchemaTemplateDrawer"
    @close="state.showSchemaTemplateDrawer = false"
  >
    <DrawerContent :title="$t('schema-template.table-template.self')">
      <div class="w-[calc(100vw-36rem)] min-w-[64rem] max-w-[calc(100vw-8rem)]">
        <TableTemplates
          :engine="databaseEngine"
          :readonly="true"
          @apply="handleApplyTemplate"
        />
      </div>
    </DrawerContent>
  </Drawer>

  <SelectClassificationDrawer
    v-if="classificationConfig"
    :show="state.showClassificationDrawer"
    :classification-config="classificationConfig"
    @dismiss="state.showClassificationDrawer = false"
    @select="onClassificationSelect"
  />

  <FeatureModal
    feature="bb.feature.schema-template"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NEllipsis, NTooltip } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, nextTick, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { SchemaDiagram, SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  hasFeature,
  generateUniqueTabId,
  useSettingV1Store,
  useSchemaEditorV1Store,
} from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaTemplateSetting_TableTemplate } from "@/types/proto/v1/setting_service";
import { emptyDatabase } from "@/types/v1/database";
import {
  Table,
  DatabaseTabContext,
  DatabaseSchema,
  SchemaEditorTabType,
  convertTableMetadataToTable,
} from "@/types/v1/schemaEditor";
import { bytesToString, isDev } from "@/utils";
import TableTemplates from "@/views/SchemaTemplate/TableTemplates.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import { isTableChanged } from "../utils";
import { useMetadataForDiagram } from "../utils/useMetadataForDiagram";

const props = withDefaults(
  defineProps<{
    searchPattern: string;
  }>(),
  {
    searchPattern: "",
  }
);

type SubtabType = "table-list" | "schema-diagram";

interface LocalState {
  selectedSubtab: SubtabType;
  selectedSchemaId: string;
  isFetchingDDL: boolean;
  statement: string;
  showFeatureModal: boolean;
  showSchemaTemplateDrawer: boolean;
  showClassificationDrawer: boolean;
  tableNameModalContext?: {
    parentName: string;
    schemaId: string;
    tableName: string | undefined;
  };
  activeTableId?: string;
}

const { t } = useI18n();
const editorStore = useSchemaEditorV1Store();
const settingStore = useSettingV1Store();
const currentTab = computed(
  () => (editorStore.currentTab || {}) as DatabaseTabContext
);
const state = reactive<LocalState>({
  selectedSubtab: "table-list",
  selectedSchemaId: "",
  isFetchingDDL: false,
  statement: "",
  showFeatureModal: false,
  showSchemaTemplateDrawer: false,
  showClassificationDrawer: false,
});
const databaseV1 = computed(
  () => editorStore.currentDatabase ?? emptyDatabase()
);
const readonly = computed(() => editorStore.readonly);

const databaseEngine = computed(() => databaseV1.value?.instanceEntity.engine);
const schemaList = computed(() => {
  return editorStore.currentSchemaList;
});
const databaseSchema = computed((): DatabaseSchema => {
  return {
    database: databaseV1.value,
    schemaList: schemaList.value,
    originSchemaList: [],
  };
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
    table.name.includes(props.searchPattern.trim())
  );
});
const classificationConfig = computed(() => {
  if (!editorStore.project.dataClassificationConfigId) {
    return;
  }
  return settingStore.getProjectClassification(
    editorStore.project.dataClassificationConfigId
  );
});
const disableChangeTable = (table: Table): boolean => {
  return (
    selectedSchema.value?.status === "dropped" || table.status === "dropped"
  );
};

const getTableClassList = (table: Table, index: number): string[] => {
  const classList = [];
  if (table.status === "dropped") {
    classList.push("text-red-700 !bg-red-50 opacity-70");
  } else if (table.status === "created") {
    classList.push("text-green-700 !bg-green-50");
  } else if (
    isTableChanged(
      currentTab.value.parentName,
      state.selectedSchemaId,
      table.id
    )
  ) {
    classList.push("text-yellow-700 !bg-yellow-50");
  }
  if (index % 2 === 1) {
    classList.push("bg-gray-50");
  }
  return classList;
};

const showClassificationDrawer = (table: Table) => {
  state.activeTableId = table.id;
  state.showClassificationDrawer = true;
};

const onClassificationSelect = (classificationId: string) => {
  state.showClassificationDrawer = false;
  const table = tableList.value.find(
    (table) => table.id === state.activeTableId
  );
  if (!table) {
    return;
  }
  table.classification = classificationId;
  state.activeTableId = undefined;
};

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

const supportClassification = computed(() => {
  return classificationConfig.value && isDev();
});

const tableHeaderList = computed(() => {
  return [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      width: "minmax(auto, 1fr)",
    },
    {
      key: "classification",
      title: t("schema-editor.column.classification"),
      hide: !supportClassification.value,
      width: "minmax(auto, 2fr)",
    },
    {
      key: "raw-count",
      title: t("schema-editor.database.row-count"),
      width: "minmax(auto, 0.7fr)",
    },
    {
      key: "data-size",
      title: t("schema-editor.database.data-size"),
      width: "minmax(auto, 0.7fr)",
    },
    {
      key: "engine",
      title: t("schema-editor.database.engine"),
      width: "minmax(auto, 1fr)",
    },
    {
      key: "collation",
      title: t("schema-editor.database.collation"),
      width: "minmax(auto, 1fr)",
    },
    {
      key: "comment",
      title: t("schema-editor.database.comment"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: "",
      width: "30px",
      hide: readonly.value,
    },
  ].filter((header) => !header.hide);
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
      parentName: currentTab.value.parentName,
      schemaId: selectedSchema.id,
      tableName: undefined,
    };
  }
};

const handleTableItemClick = (table: Table) => {
  editorStore.addTab({
    id: generateUniqueTabId(),
    type: SchemaEditorTabType.TabForTable,
    parentName: currentTab.value.parentName,
    schemaId: state.selectedSchemaId,
    tableId: table.id,
    name: table.name,
  });
};

const handleDropTable = (table: Table) => {
  editorStore.dropTable(
    currentTab.value.parentName,
    state.selectedSchemaId,
    table.id
  );
};

const handleRestoreTable = (table: Table) => {
  editorStore.restoreTable(
    currentTab.value.parentName,
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

const handleApplyTemplate = (template: SchemaTemplateSetting_TableTemplate) => {
  state.showSchemaTemplateDrawer = false;
  if (!hasFeature("bb.feature.schema-template")) {
    state.showFeatureModal = true;
    return;
  }
  if (!template.table || template.engine !== databaseEngine.value) {
    return;
  }

  const tableEdit = convertTableMetadataToTable(template.table, "created");

  const selectedSchema = schemaList.value.find(
    (schema) => schema.id === state.selectedSchemaId
  );
  if (selectedSchema) {
    selectedSchema.tableList.push(tableEdit);
    editorStore.addTab({
      id: generateUniqueTabId(),
      type: SchemaEditorTabType.TabForTable,
      parentName: currentTab.value.parentName,
      schemaId: state.selectedSchemaId,
      tableId: tableEdit.id,
      name: template.table.name,
    });
  }
};
</script>

<style scoped>
.table-item-cell {
  @apply !py-0.5;
}
</style>
