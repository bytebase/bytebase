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
      :row-key="getTableKey"
      :columns="columns"
      :data="layoutReady ? filteredTables : []"
      :row-class-name="classesForRow"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      class="schema-editor-table-list"
    />
  </div>

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
import { cloneDeep } from "lodash-es";
import { DataTableColumn, NDataTable } from "naive-ui";
import { computed, h, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import FeatureModal from "@/components/FeatureGuard/FeatureModal.vue";
import SelectClassificationDrawer from "@/components/SchemaTemplate/SelectClassificationDrawer.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { hasFeature, useSettingV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import {
  SchemaTemplateSetting_TableTemplate,
  DataClassificationSetting_DataClassificationConfig as DataClassificationConfig,
} from "@/types/proto/v1/setting_service";
import TableTemplates from "@/views/SchemaTemplate/TableTemplates.vue";
import { useSchemaEditorContext } from "../../context";
import ClassificationCell from "../TableColumnEditor/components/ClassificationCell.vue";
import { markUUID } from "../common";
import { NameCell, OperationCell } from "./components";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  tables: TableMetadata[];
  searchPattern?: string;
}>();

interface LocalState {
  showFeatureModal: boolean;
  showSchemaTemplateDrawer: boolean;
  showClassificationDrawer: boolean;
  activeTable?: TableMetadata;
}

const { t } = useI18n();
const context = useSchemaEditorContext();
const {
  project,
  readonly,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  upsertTableConfig,
} = context;
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
const settingStore = useSettingV1Store();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showSchemaTemplateDrawer: false,
  showClassificationDrawer: false,
});
const filteredTables = computed(() => {
  const keyword = props.searchPattern?.trim();
  if (!keyword) return props.tables;
  return props.tables.filter((table) => table.name.includes(keyword));
});

const engine = computed(() => {
  return props.db.instanceEntity.engine;
});

const classificationConfig = computed(() => {
  if (!project.value.dataClassificationConfigId) {
    return;
  }
  return settingStore.getProjectClassification(
    project.value.dataClassificationConfigId
  );
});

const showClassificationDrawer = (table: TableMetadata) => {
  state.activeTable = table;
  state.showClassificationDrawer = true;
};

const onClassificationSelect = (classificationId: string) => {
  const table = state.activeTable;
  if (!table) return;
  table.classification = classificationId;
  state.activeTable = undefined;
  markEditStatus(props.db, metadataForTable(table), "updated");
};

const metadataForTable = (table: TableMetadata) => {
  return {
    database: props.database,
    schema: props.schema,
    table,
  };
};
const statusForTable = (table: TableMetadata) => {
  return getTableStatus(props.db, metadataForTable(table));
};

const classesForRow = (table: TableMetadata, index: number) => {
  return statusForTable(table);
};

const isDroppedSchema = computed(() => {
  return (
    getSchemaStatus(props.db, {
      database: props.database,
      schema: props.schema,
    }) === "dropped"
  );
});

const columns = computed(() => {
  const columns: (DataTableColumn<TableMetadata> & { hide?: boolean })[] = [
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
          disabled: isDroppedSchema.value || isDroppedTable(table),
          classificationConfig:
            classificationConfig.value ??
            DataClassificationConfig.fromPartial({}),
          onEdit: () => showClassificationDrawer(table),
          onRemove: () => {
            table.classification = "";
            markEditStatus(props.db, metadataForTable(table), "updated");
          },
        });
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
          disabled: isDroppedSchema.value,
          onDrop: () => handleDropTable(table),
          onRestore: () => handleRestoreTable(table),
        });
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});

const handleTableItemClick = (table: TableMetadata) => {
  addTab({
    type: "table",
    database: props.db,
    metadata: metadataForTable(table),
  });
};

const handleDropTable = (table: TableMetadata) => {
  // We don't physically remove it, mark it as 'dropped' instead
  // If it a 'created' table, it will remains till the page is refreshed.
  markEditStatus(props.db, metadataForTable(table), "dropped");
};

const handleRestoreTable = (table: TableMetadata) => {
  removeEditStatus(props.db, metadataForTable(table), /* recursive */ false);
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
  /* eslint-disable-next-line vue/no-mutating-props */
  props.schema.tables.push(table);
  const metadata = metadataForTable(table);
  upsertTableConfig(props.db, metadata, (config) =>
    Object.assign(config, template.config)
  );

  markEditStatus(props.db, metadata, "created");
  addTab({
    type: "table",
    database: props.db,
    metadata,
  });
};

const isDroppedTable = (table: TableMetadata) => {
  return statusForTable(table) === "dropped";
};

const getTableKey = (table: TableMetadata) => {
  return markUUID(table);
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
