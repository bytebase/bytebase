<template>
  <div class="w-full h-full">
    <BBGrid
      v-if="false"
      class="border"
      :column-list="tableHeaderList"
      :data-source="tableList"
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
        <div class="bb-grid-cell" :class="getTableClassList(table, row)">
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
          class="bb-grid-cell flex items-center gap-x-2 text-sm"
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
        <div class="bb-grid-cell" :class="getTableClassList(table, row)">
          {{ table.rowCount }}
        </div>
        <div class="bb-grid-cell" :class="getTableClassList(table, row)">
          {{ bytesToString(table.dataSize.toNumber()) }}
        </div>
        <div class="bb-grid-cell" :class="getTableClassList(table, row)">
          {{ table.engine }}
        </div>
        <div class="bb-grid-cell" :class="getTableClassList(table, row)">
          {{ table.collation }}
        </div>
        <div class="bb-grid-cell" :class="getTableClassList(table, row)">
          {{ table.userComment }}
        </div>
        <div
          v-if="!readonly"
          class="bb-grid-cell !px-0.5 flex justify-start items-center"
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

  <SelectClassificationDrawer
    v-if="classificationConfig"
    :show="state.showClassificationDrawer"
    :classification-config="classificationConfig"
    @dismiss="state.showClassificationDrawer = false"
    @apply="onClassificationSelect"
  />

  <FeatureModal
    feature="bb.feature.schema-template"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NEllipsis, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  hasFeature,
  generateUniqueTabId,
  useSettingV1Store,
  useSchemaEditorV1Store,
} from "@/store";
import { SchemaTemplateSetting_TableTemplate } from "@/types/proto/v1/setting_service";
import {
  Table,
  DatabaseTabContext,
  SchemaEditorTabType,
  convertTableMetadataToTable,
} from "@/types/v1/schemaEditor";
import { bytesToString } from "@/utils";
import TableTemplates from "@/views/SchemaTemplate/TableTemplates.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import { isTableChanged } from "../utils";

const props = defineProps<{
  schemaId: string;
  tableList: Table[];
}>();

interface LocalState {
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
  showFeatureModal: false,
  showSchemaTemplateDrawer: false,
  showClassificationDrawer: false,
});

const engine = computed(() => {
  return editorStore.getCurrentEngine(currentTab.value.parentName);
});
const readonly = computed(() => editorStore.readonly);

const schemaList = computed(() => {
  return editorStore.currentSchemaList;
});
const tableList = computed(() => {
  return props.tableList;
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
  return table.status === "dropped";
};

const getTableClassList = (table: Table, index: number): string[] => {
  const classList = [];
  if (table.status === "dropped") {
    classList.push("text-red-700 !bg-red-50 opacity-70");
  } else if (table.status === "created") {
    classList.push("text-green-700 !bg-green-50");
  } else if (
    isTableChanged(currentTab.value.parentName, props.schemaId, table.id)
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
  const table = tableList.value.find(
    (table) => table.id === state.activeTableId
  );
  if (!table) {
    return;
  }
  table.classification = classificationId;
  state.activeTableId = undefined;
};

const supportClassification = computed(() => {
  return classificationConfig.value;
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
      width: "minmax(auto, 1.5fr)",
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
      width: "minmax(auto, 0.7fr)",
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

const isDroppedTable = (table: Table) => {
  return table.status === "dropped";
};

const handleTableItemClick = (table: Table) => {
  editorStore.addTab({
    id: generateUniqueTabId(),
    type: SchemaEditorTabType.TabForTable,
    parentName: currentTab.value.parentName,
    schemaId: props.schemaId,
    tableId: table.id,
    name: table.name,
  });
};

const handleDropTable = (table: Table) => {
  editorStore.dropTable(currentTab.value.parentName, props.schemaId, table.id);
};

const handleRestoreTable = (table: Table) => {
  editorStore.restoreTable(
    currentTab.value.parentName,
    props.schemaId,
    table.id
  );
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
    (schema) => schema.id === props.schemaId
  );
  if (selectedSchema) {
    selectedSchema.tableList.push(tableEdit);
    editorStore.addTab({
      id: generateUniqueTabId(),
      type: SchemaEditorTabType.TabForTable,
      parentName: currentTab.value.parentName,
      schemaId: props.schemaId,
      tableId: tableEdit.id,
      name: template.table.name,
    });
  }
};
</script>
