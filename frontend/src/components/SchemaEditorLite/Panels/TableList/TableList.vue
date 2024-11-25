<template>
  <div
    ref="containerElRef"
    class="w-full h-full overflow-x-auto"
    :data-height="containerHeight"
    :data-table-header-height="tableHeaderHeight"
    :data-table-body-height="tableBodyHeight"
  >
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
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
      :class="[disableDiffColoring && 'disable-diff-coloring']"
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

  <FeatureModal
    feature="bb.feature.schema-template"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { cloneDeep, pick } from "lodash-es";
import type { DataTableColumn, DataTableInst } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed, h, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import ClassificationCell from "@/components/ColumnDataTable/ClassificationCell.vue";
import FeatureModal from "@/components/FeatureGuard/FeatureModal.vue";
import { Drawer, DrawerContent, InlineInput } from "@/components/v2";
import { hasFeature } from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { TableConfig } from "@/types/proto/v1/database_service";
import type { SchemaTemplateSetting_TableTemplate } from "@/types/proto/v1/setting_service";
import { DataClassificationSetting_DataClassificationConfig as DataClassificationConfig } from "@/types/proto/v1/setting_service";
import TableTemplates from "@/views/SchemaTemplate/TableTemplates.vue";
import { useSchemaEditorContext } from "../../context";
import { markUUID } from "../common";
import { SelectionCell, NameCell, OperationCell } from "./components";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  tables: TableMetadata[];
  searchPattern?: string;
  customClick?: boolean;
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    }
  ): void;
}>();

interface LocalState {
  showFeatureModal: boolean;
  showSchemaTemplateDrawer: boolean;
  activeTable?: TableMetadata;
}

const { t } = useI18n();
const {
  readonly,
  selectionEnabled,
  disableDiffColoring,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableConfig,
  getTableStatus,
  upsertTableConfig,
  useConsumePendingScrollToTable,
  getAllTablesSelectionState,
  updateAllTablesSelection,
  showClassificationColumn,
  classificationConfig,
} = useSchemaEditorContext();
const dataTableRef = ref<DataTableInst>();
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
const state = reactive<LocalState>({
  showFeatureModal: false,
  showSchemaTemplateDrawer: false,
});
const filteredTables = computed(() => {
  const tables = disableDiffColoring.value
    ? props.tables.filter((table) => {
        const status = statusForTable(table);
        return status !== "dropped";
      })
    : props.tables;
  const keyword = props.searchPattern?.trim();
  if (!keyword) {
    return tables;
  }
  return tables.filter((table) => table.name.includes(keyword));
});

const engine = computed(() => {
  return props.db.instanceResource.engine;
});

const configForTable = (table: TableMetadata) => {
  return (
    getTableConfig(props.db, { schema: props.schema, table }) ??
    TableConfig.fromPartial({
      name: table.name,
    })
  );
};

const showClassification = computed(() => {
  return showClassificationColumn(
    engine.value,
    classificationConfig.value?.classificationFromConfig ?? false
  );
});

const onClassificationSelect = (classificationId: string) => {
  const table = state.activeTable;
  if (!table) return;

  upsertTableConfig(
    props.db,
    metadataForTable(table),
    (config) => (config.classificationId = classificationId)
  );

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

const classesForRow = (table: TableMetadata) => {
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
      key: "__selected__",
      width: 32,
      hide: !selectionEnabled.value,
      title: () => {
        const state = getAllTablesSelectionState(
          props.db,
          pick(props, "database", "schema"),
          filteredTables.value
        );
        return h(NCheckbox, {
          checked: state.checked,
          indeterminate: state.indeterminate,
          onUpdateChecked: (on: boolean) => {
            updateAllTablesSelection(
              props.db,
              pick(props, "database", "schema"),
              filteredTables.value,
              on
            );
          },
        });
      },
      render: (table) => {
        return h(SelectionCell, {
          db: props.db,
          metadata: {
            ...pick(props, "database", "schema"),
            table,
          },
        });
      },
    },
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      minWidth: 140,
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
      minWidth: 140,
      maxWidth: 320,
      hide: !showClassification.value,
      render: (table) => {
        const config = configForTable(table);
        return h(ClassificationCell, {
          classification: config.classificationId,
          readonly: readonly.value,
          disabled: isDroppedSchema.value || isDroppedTable(table),
          classificationConfig:
            classificationConfig.value ??
            DataClassificationConfig.fromPartial({}),
          onApply: (id: string) => {
            state.activeTable = table;
            onClassificationSelect(id);
          },
        });
      },
    },
    {
      key: "engine",
      title: t("schema-editor.database.engine"),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
      render: (table) => {
        return table.engine;
      },
    },
    {
      key: "collation",
      title: t("schema-editor.database.collation"),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
      ellipsis: true,
      ellipsisComponent: "performant-ellipsis",
    },
    {
      key: "comment",
      title: t("schema-editor.database.comment"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "input-cell",
      render: (table) => {
        return h(InlineInput, {
          value: table.userComment,
          disabled:
            readonly.value || isDroppedSchema.value || isDroppedTable(table),
          placeholder: "comment",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value) => {
            table.userComment = value;
            markEditStatus(props.db, metadataForTable(table), "updated");
          },
        });
      },
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
  if (props.customClick) {
    emit("click", metadataForTable(table));
    return;
  }
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

const vlRef = computed(() => {
  return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef
    ?.virtualListRef;
});
useConsumePendingScrollToTable(
  computed(() => ({
    db: props.db,
    metadata: {
      database: props.database,
      schema: props.schema,
    },
  })),
  vlRef,
  (params, vl) => {
    const key = getTableKey(params.metadata.table);
    if (!key) return;
    requestAnimationFrame(() => {
      try {
        console.debug("scroll-to-table", vl, params, key);
        vl.scrollTo({ key });
      } catch {
        // Do nothing
      }
    });
  }
);
</script>

<style lang="postcss" scoped>
.schema-editor-table-list
  :deep(.n-data-table-th .n-data-table-resize-button::after) {
  @apply bg-control-bg h-2/3;
}
.schema-editor-table-list :deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
.schema-editor-table-list:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.created .n-data-table-td) {
  @apply text-green-700 !bg-green-50;
}
.schema-editor-table-list:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.dropped .n-data-table-td) {
  @apply text-red-700 !bg-red-50 opacity-70;
}

.schema-editor-table-list:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.updated .n-data-table-td) {
  @apply text-yellow-700 !bg-yellow-50;
}
</style>
