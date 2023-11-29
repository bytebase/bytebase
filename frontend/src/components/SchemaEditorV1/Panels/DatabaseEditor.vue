<template>
  <div class="flex flex-col w-full h-full overflow-y-hidden">
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

    <div class="flex-1 overflow-y-hidden">
      <!-- List view -->
      <template v-if="state.selectedSubtab === 'table-list'">
        <TableList
          :schema-id="selectedSchema?.id || ''"
          :table-list="shownTableList"
        />
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
          :engine="engine"
          :readonly="true"
          @apply="handleApplyTemplate"
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
import { head } from "lodash-es";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, nextTick, reactive, watch } from "vue";
import { SchemaDiagram, SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  hasFeature,
  generateUniqueTabId,
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
import TableTemplates from "@/views/SchemaTemplate/TableTemplates.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import { useMetadataForDiagram } from "../utils/useMetadataForDiagram";
import TableList from "./TableList";

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
  showFeatureModal: boolean;
  showSchemaTemplateDrawer: boolean;
  tableNameModalContext?: {
    parentName: string;
    schemaId: string;
    tableName: string | undefined;
  };
  activeTableId?: string;
}

const editorStore = useSchemaEditorV1Store();
const currentTab = computed(() => editorStore.currentTab as DatabaseTabContext);
const state = reactive<LocalState>({
  selectedSubtab: "table-list",
  selectedSchemaId: currentTab.value?.selectedSchemaId ?? "",
  showFeatureModal: false,
  showSchemaTemplateDrawer: false,
});
const databaseV1 = computed(
  () => editorStore.currentDatabase ?? emptyDatabase()
);
const readonly = computed(() => editorStore.readonly);
const engine = computed(() => {
  return editorStore.getCurrentEngine(currentTab.value.parentName);
});
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
const shouldShowSchemaSelector = computed(() => {
  return engine.value === Engine.POSTGRES;
});

const allowCreateTable = computed(() => {
  if (engine.value === Engine.POSTGRES) {
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

watch(
  [() => currentTab.value],
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
  if (!template.table || template.engine !== engine.value) {
    return;
  }

  const tableEdit = convertTableMetadataToTable(
    template.table,
    "created",
    template.config
  );

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
