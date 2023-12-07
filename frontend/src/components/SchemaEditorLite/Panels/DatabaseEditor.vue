<template>
  <div class="flex flex-col w-full h-full overflow-y-hidden">
    <div class="py-2 w-full flex flex-row justify-between items-center">
      <div>
        <div
          v-if="state.selectedSubTab === 'table-list'"
          class="w-full flex justify-between items-center space-x-2"
        >
          <div class="flex flex-row justify-start items-center gap-x-3">
            <div
              v-if="shouldShowSchemaSelector"
              class="pl-1 flex flex-row justify-start items-center text-sm gap-x-2"
            >
              <span>Schema:</span>
              <NSelect
                v-if="currentTab"
                v-model:value="currentTab.selectedSchema"
                class="min-w-[8rem]"
                :options="schemaSelectorOptionList"
              />
            </div>
            <NButton
              v-if="!readonly"
              :disabled="!allowCreateTable"
              @click="handleCreateNewTable"
            >
              <template #icon>
                <PlusIcon class="w-4 h-4" />
              </template>
              {{ $t("schema-editor.actions.create-table") }}
            </NButton>
            <NButton
              v-if="!readonly"
              :disabled="!allowCreateTable"
              @click="state.showSchemaTemplateDrawer = true"
            >
              <template #icon>
                <FeatureBadge feature="bb.feature.schema-template" />
                <PlusIcon class="w-4 h-4" />
              </template>
              {{ $t("schema-editor.actions.add-from-template") }}
            </NButton>
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
        <tables
          v-if="selectedSchema"
          :db="db"
          :database="database"
          :schema="selectedSchema"
          :tables="selectedSchema.tables"
        />
      </template>
      <template v-else-if="state.selectedSubTab === 'schema-diagram'">
        <!-- TODO: bring status coloring back -->
        <SchemaDiagram
          :database="db"
          :database-metadata="database"
          :editable="true"
          @edit-table="tryEditTable"
          @edit-column="tryEditColumn"
        />
      </template>
    </div>
  </div>

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :database="db"
    :metadata="database"
    :schema="state.tableNameModalContext.schema"
    mode="create"
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
import { cloneDeep, head } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
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
import tables from "./TableList";

withDefaults(
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
    schema: SchemaMetadata;
  };
  activeTableId?: string;
}

const context = useSchemaEditorContext();
const { readonly, addTab, getSchemaStatus, markEditStatus, upsertTableConfig } =
  context;
const currentTab = computed(() => {
  return context.currentTab.value as DatabaseTabContext;
});
const state = reactive<LocalState>({
  selectedSubTab: "table-list",
  showFeatureModal: false,
  showSchemaTemplateDrawer: false,
});
const db = computed(() => currentTab.value.database);
const engine = computed(() => {
  return db.value.instanceEntity.engine;
});
const database = computed(() => {
  return currentTab.value.metadata.database;
});
const schemaList = computed(() => {
  return database.value?.schemas ?? [];
});
const selectedSchema = computed(() => {
  const selectedSchema = currentTab.value.selectedSchema;
  return schemaList.value.find((schema) => schema.name === selectedSchema);
});
const shouldShowSchemaSelector = computed(() => {
  return engine.value === Engine.POSTGRES;
});

const allowCreateTable = computed(() => {
  const schema = selectedSchema.value;
  if (!schema) return false;
  if (engine.value === Engine.POSTGRES) {
    const status = getSchemaStatus(db.value, {
      database: database.value,
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
    state.tableNameModalContext = {
      schema: selectedSchema.value,
    };
  }
};

const tryEditTable = async (schema: SchemaMetadata, table: TableMetadata) => {
  currentTab.value.selectedSchema = schema.name;
  await nextTick();
  addTab({
    type: "table",
    database: db.value,
    metadata: {
      database: database.value,
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
  if (!template.table) {
    return;
  }
  if (template.engine !== engine.value) {
    return;
  }

  const table = cloneDeep(template.table);
  const schema = selectedSchema.value;
  if (!schema) {
    return;
  }
  schema.tables.push(table);
  const metadataForTable = () => {
    return {
      database: database.value,
      schema,
      table,
    };
  };
  if (template.config) {
    upsertTableConfig(db.value, metadataForTable(), (config) => {
      Object.assign(config, template.config);
    });
  }
  markEditStatus(db.value, metadataForTable(), "created");
  table.columns.forEach((column) => {
    markEditStatus(db.value, { ...metadataForTable(), column }, "created");
  });
};
</script>
