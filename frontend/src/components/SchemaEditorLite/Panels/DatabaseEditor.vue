<template>
  <div class="flex flex-col w-full h-full overflow-y-hidden">
    <div class="py-2 w-full flex flex-row justify-between items-center">
      <div>
        <div
          v-if="state.selectedSubTab === 'table-list'"
          class="w-full flex justify-between items-center space-x-2"
        >
          <div class="flex flex-row justify-start items-center space-x-3">
            <div
              v-if="shouldShowSchemaSelector"
              class="ml-2 flex flex-row justify-start items-center text-sm"
            >
              <span class="mr-1">Schema:</span>
              <NSelect
                v-if="currentTab"
                v-model:value="currentTab.selectedSchema"
                class="min-w-[8rem]"
                :options="schemaSelectorOptionList"
              />
            </div>
            <button
              v-if="!readonly"
              class="flex flex-row justify-center items-center border px-3 py-1 leading-6 rounded text-sm hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60 whitespace-nowrap"
              :disabled="!allowCreateTable"
              @click="handleCreateNewTable"
            >
              <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
              {{ $t("schema-editor.actions.create-table") }}
            </button>
            <button
              v-if="!readonly"
              class="flex flex-row justify-center items-center border px-3 py-1 leading-6 rounded text-sm hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60 whitespace-nowrap"
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
          class="flex flex-row justify-end items-center bg-gray-100 p-1 rounded whitespace-nowrap"
        >
          <button
            class="px-2 leading-7 text-sm text-gray-500 cursor-pointer select-none rounded flex justify-center items-center"
            :class="
              state.selectedSubTab === 'table-list' &&
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
              state.selectedSubTab === 'schema-diagram' &&
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
      <template v-if="state.selectedSubTab === 'table-list'">
        <TableList
          v-if="selectedSchema"
          :database="database"
          :schema="selectedSchema"
          :table-list="shownTableList"
        />
      </template>
      <template v-else-if="state.selectedSubTab === 'schema-diagram'">
        <!-- TODO: bring status coloring back -->
        <SchemaDiagram
          :database="database"
          :database-metadata="metadata"
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
import { computed, nextTick, reactive, watch } from "vue";
import { SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { Drawer, DrawerContent } from "@/components/v2";
import { hasFeature } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaTemplateSetting_TableTemplate } from "@/types/proto/v1/setting_service";
import TableTemplates from "@/views/SchemaTemplate/TableTemplates.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import { useSchemaEditorContext } from "../context";
import { DatabaseTabContext } from "../types";
import TableList from "./TableList";

const props = withDefaults(
  defineProps<{
    searchPattern: string;
  }>(),
  {
    searchPattern: "",
  }
);

type SubTabType = "table-list" | "schema-diagram";

interface LocalState {
  selectedSubTab: SubTabType;
  showFeatureModal: boolean;
  showSchemaTemplateDrawer: boolean;
  tableNameModalContext?: {
    parentName: string;
    schemaId: string;
    tableName: string | undefined;
  };
  activeTableId?: string;
}

const context = useSchemaEditorContext();
const { readonly, addTab, getSchemaStatus } = context;
const currentTab = computed(() => {
  return context.currentTab.value as DatabaseTabContext;
});
const state = reactive<LocalState>({
  selectedSubTab: "table-list",
  showFeatureModal: false,
  showSchemaTemplateDrawer: false,
});
const database = computed(() => currentTab.value.database);
const engine = computed(() => {
  return database.value.instanceEntity.engine;
});
const metadata = computed(() => {
  return currentTab.value.metadata.database;
});
const schemaList = computed(() => {
  return metadata.value?.schemas ?? [];
});
const selectedSchema = computed(() => {
  const selectedSchema = currentTab.value.selectedSchema;
  return schemaList.value.find((schema) => schema.name === selectedSchema);
});
const tableList = computed(() => {
  return selectedSchema.value?.tables ?? [];
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
  const schema = selectedSchema.value;
  if (!schema) return false;
  if (engine.value === Engine.POSTGRES) {
    const status = getSchemaStatus(database.value, {
      database: metadata.value,
      schema,
    });

    return (
      schemaList.value.length > 0 &&
      selectedSchema.value &&
      status !== "dropped"
    );
  }
  return true;
});

const schemaSelectorOptionList = computed(() => {
  const optionList = [];
  for (const schema of schemaList.value) {
    optionList.push({
      label: schema.name,
      value: schema.name,
    });
  }
  return optionList;
});

watch(
  schemaSelectorOptionList,
  (options) => {
    if (currentTab.value) {
      if (
        !options.find((opt) => opt.value === currentTab.value.selectedSchema)
      ) {
        currentTab.value.selectedSchema = head(options)?.value;
      }
    }
  },
  {
    immediate: true,
  }
);

const handleChangeTab = (tab: SubTabType) => {
  state.selectedSubTab = tab;
};

const handleCreateNewTable = () => {
  if (selectedSchema.value) {
    // TODO
    // state.tableNameModalContext = {
    //   parentName: currentTab.value.parentName,
    //   schemaId: selectedSchema.id,
    //   tableName: undefined,
    // };
  }
};

const tryEditTable = async (schema: SchemaMetadata, table: TableMetadata) => {
  currentTab.value.selectedSchema = schema.name;
  await nextTick();
  addTab({
    type: "table",
    database: database.value,
    metadata: {
      database: metadata.value,
      schema,
      table,
    },
  });
};

const tryEditColumn = async (
  schema: SchemaMetadata,
  table: TableMetadata,
  column: ColumnMetadata,
  target: "name" | "type"
) => {
  if (schema && table && column) {
    await tryEditTable(schema, table);

    // TODO: scroll column into view and focus the input box
    // await nextTick();
    // const container = document.querySelector("#table-editor-container");
    // const input = container?.querySelector(
    //   `.column-${column.id} .column-${target}-input`
    // ) as HTMLInputElement | undefined;
    // if (input) {
    //   input.focus();
    //   scrollIntoView(input);
    // }
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

  // const tableEdit = convertTableMetadataToTable(
  //   template.table,
  //   "created",
  //   template.config
  // );

  // const selectedSchema = schemaList.value.find(
  //   (schema) => schema.id === state.selectedSchema
  // );
  // if (selectedSchema) {
  //   selectedSchema.tableList.push(tableEdit);
  //   editorStore.addTab({
  //     id: generateUniqueTabId(),
  //     type: SchemaEditorTabType.TabForTable,
  //     parentName: currentTab.value.parentName,
  //     schemaId: state.selectedSchema,
  //     tableId: tableEdit.id,
  //     name: template.table.name,
  //   });
  // }
};
</script>
