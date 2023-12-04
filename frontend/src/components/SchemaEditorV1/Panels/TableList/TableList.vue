<template>
  <div
    ref="containerElRef"
    class="w-full h-full"
    :data-height="containerHeight"
    :data-table-header-height="tableHeaderHeight"
    :data-table-body-height="tableBodyHeight"
  >
    <NDataTable
      v-bind="$attrs"
      size="small"
      :row-key="(table: Table) => table.id"
      :columns="columns"
      :data="layoutReady ? tableList : []"
      :row-class-name="classesForRow"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      class="schema-editor-table-list"
    />
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
import { useElementSize } from "@vueuse/core";
import { DataTableColumn, NDataTable } from "naive-ui";
import { computed, h, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import FeatureModal from "@/components/FeatureGuard/FeatureModal.vue";
import SelectClassificationDrawer from "@/components/SchemaTemplate/SelectClassificationDrawer.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  hasFeature,
  generateUniqueTabId,
  useSettingV1Store,
  useSchemaEditorV1Store,
} from "@/store";
import {
  SchemaTemplateSetting_TableTemplate,
  DataClassificationSetting_DataClassificationConfig as DataClassificationConfig,
} from "@/types/proto/v1/setting_service";
import {
  Table,
  DatabaseTabContext,
  SchemaEditorTabType,
  convertTableMetadataToTable,
} from "@/types/v1/schemaEditor";
import { bytesToString } from "@/utils";
import TableTemplates from "@/views/SchemaTemplate/TableTemplates.vue";
import TableNameModal from "../../Modals/TableNameModal.vue";
import { isTableChanged } from "../../utils";
import ClassificationCell from "../TableColumnEditor/components/ClassificationCell.vue";
import { NameCell, OperationCell } from "./components";

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
const containerElRef = ref<HTMLElement>();
const tableHeaderElRef = computed(
  () =>
    containerElRef.value?.querySelector("thead.n-data-table-thead") as
      | HTMLElement
      | undefined
);
const { height: containerHeight } = useElementSize(containerElRef);
const { height: tableHeaderHeight } = useElementSize(tableHeaderElRef);
const tableBodyHeight = computed(() => {
  return containerHeight.value - tableHeaderHeight.value - 2;
});
// Use this to avoid unnecessary initial rendering
const layoutReady = computed(() => tableHeaderHeight.value > 0);
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

const columns = computed(() => {
  const columns: (DataTableColumn<Table> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      width: 140,
      className: "truncate",
      render: (table) => {
        return h(NameCell, {
          table,
          onClick: () => handleTableItemClick(table),
        });
      },
    },
    {
      key: "classification",
      title: t("schema-editor.column.classification"),
      resizable: true,
      width: 140,
      hide: !classificationConfig.value,
      render: (table) => {
        return h(ClassificationCell, {
          classification: table.classification,
          readonly: readonly.value,
          disabled: !disableChangeTable(table),
          classificationConfig:
            classificationConfig.value ??
            DataClassificationConfig.fromPartial({}),
          onEdit: () => showClassificationDrawer(table),
          onRemove: () => (table.classification = ""),
        });
      },
    },
    {
      key: "rowCount",
      title: t("schema-editor.database.row-count"),
      resizable: true,
      width: 120,
      render: (table) => {
        return table.rowCount.toString();
      },
    },
    {
      key: "dataSize",
      title: t("schema-editor.database.data-size"),
      resizable: true,
      width: 120,
      render: (table) => {
        return bytesToString(table.dataSize.toNumber());
      },
    },
    {
      key: "engine",
      title: t("schema-editor.database.engine"),
      resizable: true,
      width: 120,
      render: (table) => {
        return table.engine;
      },
    },
    {
      key: "collation",
      title: t("schema-editor.database.collation"),
      resizable: true,
      width: 120,
      ellipsis: true,
      ellipsisComponent: "performant-ellipsis",
    },
    {
      key: "comment",
      title: t("schema-editor.database.comment"),
      resizable: true,
      width: 140,
      ellipsis: true,
      ellipsisComponent: "performant-ellipsis",
    },
    {
      key: "operations",
      title: "",
      resizable: false,
      width: 30,
      hide: readonly.value,
      className: "!px-0",
      render: (table) => {
        return h(OperationCell, {
          table,
          dropped: isDroppedTable(table),
          onDrop: () => handleDropTable(table),
          onRestore: () => handleRestoreTable(table),
        });
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});

const classesForRow = (table: Table, index: number) => {
  if (table.status === "dropped") {
    return "dropped";
  } else if (table.status === "created") {
    return "created";
  } else if (
    isTableChanged(currentTab.value.parentName, props.schemaId, table.id)
  ) {
    return "updated";
  }
  return "";
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

const disableChangeTable = (table: Table): boolean => {
  return table.status === "dropped";
};
const isDroppedTable = (table: Table) => {
  return table.status === "dropped";
};
</script>

<style lang="postcss" scoped>
.schema-editor-table-list
  :deep(.n-data-table-th .n-data-table-resize-button::after) {
  @apply bg-control-bg h-2/3;
}
.schema-editor-table-list :deep(.n-data-table-tr.created .n-data-table-td) {
  @apply text-green-700 !bg-green-50;
}
.schema-editor-table-list :deep(.n-data-table-tr.dropped .n-data-table-td) {
  @apply text-red-700 !bg-red-50 opacity-70;
}

.schema-editor-table-list :deep(.n-data-table-tr.updated .n-data-table-td) {
  @apply text-yellow-700 !bg-yellow-50;
}
</style>
